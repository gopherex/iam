package postgres

// Connection endpoint resolution.
//
// A federation connection has two halves. The upstream half — the IdP's entity,
// its SSO URL, its keys — is whatever the customer's administrator gives us. The
// local half is ours: the entity id we call ourselves by, where the IdP posts an
// assertion, where an OIDC provider redirects back to. Nobody should have to
// type the local half in, and letting them get it wrong is how a SAML
// integration fails with an error message about an audience mismatch that names
// nothing useful.
//
// So the local endpoints are derived from the deployment's public URL and the
// connection id, and only overridden when a connection explicitly stores
// something else (a customer whose IdP was already configured against an older
// entity id, say). Derivation runs on every read, so a connection created before
// these fields existed resolves correctly without a migration.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/crewjam/saml/samlsp"

	"github.com/gopherex/iam/internal/domain"
)

// fedDiscoveryTimeout bounds a discovery fetch. It runs inside an admin request,
// so it fails fast rather than holding the caller.
const fedDiscoveryTimeout = 10 * time.Second

// fedDiscoveryMaxBytes caps a discovery document. The URL is operator-supplied
// but the response is not, and an unbounded read is a memory DoS.
const fedDiscoveryMaxBytes = 1 << 20

// fedSAMLBase / fedOIDCBase are the runtime route prefixes the local endpoints
// are built on. They must match the paths served in openapi.yaml.
const (
	fedSAMLBase = "/v1/sso/saml/"
	fedOIDCBase = "/v1/sso/oidc/"
)

// Connection types, as stored.
const (
	fedConnectionTypeSAML = "saml"
	fedConnectionTypeOIDC = "oidc"
)

// fedApplyEndpointDefaults fills in the local half of a connection's config from
// the deployment's public URL. Values already stored win, so an explicit
// override is never clobbered.
func fedApplyEndpointDefaults(publicURL string, c *domain.Connection) {
	if c == nil || c.ID == "" {
		return
	}

	base := strings.TrimRight(publicURL, "/")
	if base == "" {
		return
	}

	switch c.Type {
	case fedConnectionTypeSAML:
		fedApplySAMLEndpointDefaults(base, c)
	case fedConnectionTypeOIDC:
		fedApplyOIDCEndpointDefaults(base, c)
	}
}

// fedApplySAMLEndpointDefaults fills the SP half of a SAML connection.
func fedApplySAMLEndpointDefaults(base string, c *domain.Connection) {
	if c.Config == nil {
		c.Config = &domain.FederationConnectionConfig{}
	}

	if c.Config.Saml == nil {
		c.Config.Saml = &domain.FederationSamlConfig{}
	}

	cfg := c.Config.Saml
	derived := base + fedSAMLBase + c.ID + "/metadata"
	// Connections written before the SP and IdP metadata URLs were separate
	// fields kept the IdP's address in MetadataURL. Recover it rather than
	// migrate: a value that is not the endpoint we serve was the IdP's.
	if cfg.IDPMetadataURL == "" && cfg.MetadataURL != "" && cfg.MetadataURL != derived {
		cfg.IDPMetadataURL = cfg.MetadataURL
		cfg.MetadataURL = ""
	}

	if cfg.MetadataURL == "" {
		cfg.MetadataURL = derived
	}

	if cfg.AcsURL == "" {
		cfg.AcsURL = base + fedSAMLBase + c.ID + "/acs"
	}
	// The SP entity id defaults to its metadata URL, which is what most IdPs
	// expect and what the SAML metadata profile suggests.
	if cfg.EntityID == "" {
		cfg.EntityID = cfg.MetadataURL
	}
}

// fedApplyOIDCEndpointDefaults fills the callback of an OIDC connection.
func fedApplyOIDCEndpointDefaults(base string, c *domain.Connection) {
	if c.Config == nil {
		c.Config = &domain.FederationConnectionConfig{}
	}

	if c.Config.Oidc == nil {
		c.Config.Oidc = &domain.FederationOidcConfig{}
	}

	if c.Config.Oidc.RedirectURL == "" {
		c.Config.Oidc.RedirectURL = base + fedOIDCBase + c.ID + "/callback"
	}
}

// fedOIDCDiscovery is the subset of an OpenID provider's discovery document we
// need to drive the authorization code flow against it.
type fedOIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// fedResolveOIDCDiscovery fills an OIDC connection's endpoints from the
// provider's discovery document.
//
// Asking an administrator to copy three URLs out of a document the provider
// already publishes is three chances to paste the wrong one, and a wrong
// jwks_uri fails as an unverifiable id_token long after the mistake. So the
// document is fetched here, at configuration time, where a failure is reported
// to the person who can fix it.
//
// Endpoints already set are left alone: an explicit override survives.
func fedResolveOIDCDiscovery(ctx context.Context, client *http.Client, cfg *domain.FederationOidcConfig) error {
	if cfg == nil {
		return nil
	}

	if cfg.AuthURL != "" && cfg.TokenURL != "" && cfg.JWKSURL != "" {
		return nil
	}

	url := fedDiscoveryURL(cfg)
	if url == "" {
		return nil
	}

	doc, err := fedFetchOIDCDiscovery(ctx, client, url)
	if err != nil {
		return err
	}

	if cfg.Issuer == "" {
		cfg.Issuer = doc.Issuer
	}

	if cfg.AuthURL == "" {
		cfg.AuthURL = doc.AuthorizationEndpoint
	}

	if cfg.TokenURL == "" {
		cfg.TokenURL = doc.TokenEndpoint
	}

	if cfg.JWKSURL == "" {
		cfg.JWKSURL = doc.JWKSURI
	}

	return nil
}

// fedDiscoveryURL is where the provider's metadata lives: the explicit discovery
// URL when one was configured, otherwise the well-known path under the issuer.
func fedDiscoveryURL(cfg *domain.FederationOidcConfig) string {
	if cfg.DiscoveryURL != "" {
		return cfg.DiscoveryURL
	}

	if cfg.Issuer == "" {
		return ""
	}

	return strings.TrimRight(cfg.Issuer, "/") + "/.well-known/openid-configuration"
}

// fedFetchOIDCDiscovery reads and parses a discovery document.
func fedFetchOIDCDiscovery(ctx context.Context, client *http.Client, url string) (*fedOIDCDiscovery, error) {
	if err := domain.ValidateAbsoluteHTTPURL("oidc_discovery_url", url); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, domain.ErrProviderError.WithMessage("discovery request could not be built")
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, domain.ErrProviderError.WithMessage("discovery document could not be fetched: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, domain.ErrProviderError.WithMessage(
			fmt.Sprintf("discovery document returned %d", resp.StatusCode))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, fedDiscoveryMaxBytes))
	if err != nil {
		return nil, domain.ErrProviderError.WithMessage("discovery document could not be read")
	}

	var doc fedOIDCDiscovery
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, domain.ErrProviderError.WithMessage("discovery document is not JSON")
	}

	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" || doc.JWKSURI == "" {
		return nil, domain.ErrProviderError.WithMessage(
			"discovery document is missing authorization_endpoint, token_endpoint or jwks_uri")
	}

	return &doc, nil
}

// fedResolveSAMLMetadata fetches the IdP's metadata document when a connection
// names one but does not carry it.
//
// The document is the authoritative source for the IdP's entity id, SSO endpoint
// and signing certificate, and it changes rarely — so it is fetched once, here,
// and stored. Fetching it per assertion would put the IdP's availability in the
// path of every sign-in, and not fetching it at all leaves a connection that
// looks configured and cannot verify anything.
func fedResolveSAMLMetadata(ctx context.Context, client *http.Client, cfg *domain.FederationSamlConfig) error {
	if cfg == nil || cfg.IDPMetadataXML != "" || cfg.IDPMetadataURL == "" {
		return nil
	}

	xml, err := fedFetchSAMLMetadata(ctx, client, cfg.IDPMetadataURL)
	if err != nil {
		return err
	}

	cfg.IDPMetadataXML = xml

	return nil
}

// fedFetchSAMLMetadata reads an IdP metadata document.
func fedFetchSAMLMetadata(ctx context.Context, client *http.Client, url string) (string, error) {
	if url == "" {
		return "", nil
	}

	if err := domain.ValidateAbsoluteHTTPURL("saml_metadata_url", url); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", domain.ErrProviderError.WithMessage("metadata request could not be built")
	}

	req.Header.Set("Accept", "application/samlmetadata+xml, application/xml, text/xml")

	resp, err := client.Do(req)
	if err != nil {
		return "", domain.ErrProviderError.WithMessage("IdP metadata could not be fetched: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", domain.ErrProviderError.WithMessage(
			fmt.Sprintf("IdP metadata returned %d", resp.StatusCode))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, fedDiscoveryMaxBytes))
	if err != nil {
		return "", domain.ErrProviderError.WithMessage("IdP metadata could not be read")
	}

	if _, err := samlsp.ParseMetadata(raw); err != nil {
		return "", domain.ErrProviderError.WithMessage("IdP metadata is not a SAML metadata document")
	}

	return string(raw), nil
}
