package postgres

// Federation adapter — Postgres persistence for the SSO/SCIM federation ports
// (api.FederationConnections, api.FederationRuntime, api.FederationScim).
//
// Tables it owns:
//   - iam_sso_connections : SSO connection aggregate (saml | oidc).
//   - iam_domains         : verified email domains routed to a connection.
//   - iam_scim_tokens     : SCIM provisioning bearer credentials (hash only).
//   - iam_scim_resources  : SCIM Users / Groups (connection-scoped).
//
// Persistence follows the package gold pattern (reference.go): the domain
// aggregate is marshalled into the `data jsonb` envelope, the typed columns are
// lookup-only, every mutation runs inside db.withTx / withTxRet, reads run on
// db.Bobx(). The tenant boundary is project_id on every query; SCIM resources
// are additionally scoped to their connection_id (a row whose connection/project
// does not match the request is treated as not-found).
//
// Crypto policy: SCIM token secrets and SSO exchange/auth codes are opaque
// tokens minted with crypto/rand; only their sha256 hash is persisted. The SAML
// and external-OIDC protocol crypto IS implemented here with the upstream
// libraries — github.com/crewjam/saml (AuthnRequest build/sign, signed-assertion
// verification, SP metadata, SP cert rotation) and golang.org/x/oauth2 + jwx
// (OIDC code exchange + id_token signature verification against the provider
// JWKS). After verification the external subject is provisioned to an IAM user +
// session and a single-use exchange code (iam_auth_codes) is issued for the
// /v1/sso/exchange leg to resolve.

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"math/big"
	"net/url"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"golang.org/x/oauth2"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// ===========================================================================
// shared federation helpers
// ===========================================================================

// fedRandomToken mints an opaque, URL-safe random token (crypto/rand). It is the
// plaintext secret handed back to the caller exactly once; only its hash is
// stored.
func fedRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// fedHashToken is the sha256 hex digest persisted in hash columns; the plaintext
// opaque token is never stored.
func fedHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// fedConnectionFromRow rebuilds a domain.Connection from its envelope. The jsonb
// blob is authoritative; lookup columns backfill anything the blob omits.
func fedConnectionFromRow(cipher Cipher, row *models.IamSsoConnection) (*domain.Connection, error) {
	var c domain.Connection
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &c); err != nil {
			return nil, err
		}
	}
	if err := fedDecryptConnSecrets(cipher, &c); err != nil {
		return nil, err
	}
	c.ID = row.ID
	c.ProjectID = row.ProjectID
	if c.Type == "" {
		c.Type = row.Type
	}
	if c.Status == "" {
		c.Status = row.Status
	}
	if c.Name == "" {
		c.Name = row.Name
	}
	if c.ExternalRef == "" {
		if v, ok := row.ExternalRef.Get(); ok {
			c.ExternalRef = v
		}
	}
	return &c, nil
}

// fedDomainFromRow rebuilds a domain.Domain from its envelope.
func fedDomainFromRow(row *models.IamDomain) (*domain.Domain, error) {
	var d domain.Domain
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &d); err != nil {
			return nil, err
		}
	}
	d.ID = row.ID
	d.ProjectID = row.ProjectID
	if d.Domain == "" {
		d.Domain = row.Domain
	}
	if d.Status == "" {
		d.Status = row.Status
	}
	if d.ConnectionID == "" {
		if v, ok := row.ConnectionID.Get(); ok {
			d.ConnectionID = v
		}
	}
	return &d, nil
}

// fedTokenFromRow rebuilds a domain.ScimToken from its envelope (secret excluded;
// only the hash lives in the hash column).
func fedTokenFromRow(row *models.IamScimToken) (*domain.ScimToken, error) {
	var t domain.ScimToken
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &t); err != nil {
			return nil, err
		}
	}
	t.ID = row.ID
	t.ProjectID = row.ProjectID
	t.ConnectionID = row.ConnectionID
	return &t, nil
}

// fedEncryptConnSecrets encrypts the reversible secrets in a connection's config
// (SAML SP private key, OIDC client secret) in place before persistence.
func fedEncryptConnSecrets(cipher Cipher, c *domain.Connection) error {
	if c.Config == nil {
		return nil
	}
	if c.Config.Saml != nil && c.Config.Saml.SPPrivateKeyPEM != "" {
		v, err := cipher.Encrypt(c.Config.Saml.SPPrivateKeyPEM)
		if err != nil {
			return err
		}
		c.Config.Saml.SPPrivateKeyPEM = v
	}
	if c.Config.Oidc != nil && c.Config.Oidc.ClientSecret != "" {
		v, err := cipher.Encrypt(c.Config.Oidc.ClientSecret)
		if err != nil {
			return err
		}
		c.Config.Oidc.ClientSecret = v
	}
	return nil
}

// fedDecryptConnSecrets reverses fedEncryptConnSecrets after a row is loaded.
func fedDecryptConnSecrets(cipher Cipher, c *domain.Connection) error {
	if c.Config == nil {
		return nil
	}
	if c.Config.Saml != nil && c.Config.Saml.SPPrivateKeyPEM != "" {
		v, err := cipher.Decrypt(c.Config.Saml.SPPrivateKeyPEM)
		if err != nil {
			return err
		}
		c.Config.Saml.SPPrivateKeyPEM = v
	}
	if c.Config.Oidc != nil && c.Config.Oidc.ClientSecret != "" {
		v, err := cipher.Decrypt(c.Config.Oidc.ClientSecret)
		if err != nil {
			return err
		}
		c.Config.Oidc.ClientSecret = v
	}
	return nil
}

// fedConnSetter builds an insert/update setter from a connection aggregate: the
// lookup columns are projected from the struct, the aggregate is stored whole in
// the jsonb envelope.
func fedConnSetter(cipher Cipher, c *domain.Connection) (*models.IamSsoConnectionSetter, error) {
	// Encrypt reversible secrets for persistence without corrupting the caller's
	// in-memory plaintext aggregate: snapshot, encrypt, marshal, restore.
	var samlOrig, oidcOrig string
	hasSaml := c.Config != nil && c.Config.Saml != nil
	hasOidc := c.Config != nil && c.Config.Oidc != nil
	if hasSaml {
		samlOrig = c.Config.Saml.SPPrivateKeyPEM
	}
	if hasOidc {
		oidcOrig = c.Config.Oidc.ClientSecret
	}
	if err := fedEncryptConnSecrets(cipher, c); err != nil {
		return nil, err
	}
	raw, err := marshal(c)
	if hasSaml {
		c.Config.Saml.SPPrivateKeyPEM = samlOrig
	}
	if hasOidc {
		c.Config.Oidc.ClientSecret = oidcOrig
	}
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamSsoConnectionSetter{
		ID:        &c.ID,
		ProjectID: &c.ProjectID,
		Type:      ptr(c.Type),
		Status:    ptr(c.Status),
		Name:      ptr(c.Name),
		Data:      &rm,
	}
	if c.ExternalRef != "" {
		v := null.From(c.ExternalRef)
		setter.ExternalRef = &v
	}
	return setter, nil
}

// ===========================================================================
// SAML / OIDC protocol crypto helpers (github.com/crewjam/saml, x/oauth2, jwx)
// ===========================================================================

// fedParsePrivateKeyPEM decodes a PEM-encoded RSA/PKCS#8 private key used to sign
// SAML AuthnRequests. It accepts PKCS#1 ("RSA PRIVATE KEY") and PKCS#8 ("PRIVATE
// KEY") blocks.
func fedParsePrivateKeyPEM(pemStr string) (crypto.Signer, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, domain.ErrProviderError
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, domain.ErrProviderError
	}
	signer, ok := keyAny.(crypto.Signer)
	if !ok {
		return nil, domain.ErrProviderError
	}
	return signer, nil
}

// fedParseCertificatePEM decodes a PEM-encoded X.509 certificate (the SP signing
// certificate advertised in metadata / used to sign AuthnRequests).
func fedParseCertificatePEM(pemStr string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, domain.ErrProviderError
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, domain.ErrProviderError
	}
	return cert, nil
}

// fedSamlServiceProvider builds a crewjam saml.ServiceProvider from a connection's
// stored config. IdP metadata XML is the authoritative source for the IdP entity
// (SSO endpoint + signing certificate); when only a raw IdP certificate is stored
// it is wrapped into a minimal IDPMetadata so signature verification still works.
// The SP keypair (Key/Certificate) is optional — when present, AuthnRequests are
// signed.
func fedSamlServiceProvider(c *domain.Connection) (*saml.ServiceProvider, error) {
	if c.Config == nil || c.Config.Saml == nil {
		return nil, domain.ErrProviderError
	}
	cfg := c.Config.Saml

	sp := &saml.ServiceProvider{
		EntityID: cfg.EntityID,
	}
	if cfg.AcsURL != "" {
		u, err := url.Parse(cfg.AcsURL)
		if err != nil {
			return nil, domain.ErrProviderError
		}
		sp.AcsURL = *u
	}
	if cfg.MetadataURL != "" {
		u, err := url.Parse(cfg.MetadataURL)
		if err != nil {
			return nil, domain.ErrProviderError
		}
		sp.MetadataURL = *u
	}

	switch {
	case cfg.IDPMetadataXML != "":
		md, err := samlsp.ParseMetadata([]byte(cfg.IDPMetadataXML))
		if err != nil {
			return nil, domain.ErrProviderError
		}
		sp.IDPMetadata = md
	case cfg.IDPCertificatePEM != "":
		// Wrap the raw IdP signing cert into a minimal IDPMetadata so the library's
		// signature verification (getIDPSigningCerts) can find it. The DER bytes are
		// base64-encoded into an X509Certificate descriptor.
		cert, err := fedParseCertificatePEM(cfg.IDPCertificatePEM)
		if err != nil {
			return nil, err
		}
		sp.IDPMetadata = &saml.EntityDescriptor{
			IDPSSODescriptors: []saml.IDPSSODescriptor{{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						KeyDescriptors: []saml.KeyDescriptor{{
							Use: "signing",
							KeyInfo: saml.KeyInfo{
								X509Data: saml.X509Data{
									X509Certificates: []saml.X509Certificate{{
										Data: base64.StdEncoding.EncodeToString(cert.Raw),
									}},
								},
							},
						}},
					},
				},
			}},
		}
	default:
		return nil, domain.ErrProviderError
	}

	// Optional SP signing keypair (enables signed AuthnRequests + signing cert in
	// the SP metadata document).
	if cfg.SPPrivateKeyPEM != "" {
		key, err := fedParsePrivateKeyPEM(cfg.SPPrivateKeyPEM)
		if err != nil {
			return nil, err
		}
		sp.Key = key
	}
	if cfg.SPCertificatePEM != "" {
		cert, err := fedParseCertificatePEM(cfg.SPCertificatePEM)
		if err != nil {
			return nil, err
		}
		sp.Certificate = cert
	}
	return sp, nil
}

// fedSamlSubject extracts the external subject (a stable opaque id) and email
// from a verified SAML assertion: the NameID is the subject; the email is taken
// from common email attributes (falling back to the NameID when it looks like an
// address).
func fedSamlSubject(a *saml.Assertion) (subject, email string) {
	if a.Subject != nil && a.Subject.NameID != nil {
		subject = a.Subject.NameID.Value
	}
	emailNames := map[string]bool{
		"email":        true,
		"mail":         true,
		"emailaddress": true,
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": true,
		"urn:oid:0.9.2342.19200300.100.1.3":                                  true,
	}
	for _, stmt := range a.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if email != "" {
				break
			}
			if emailNames[attr.FriendlyName] || emailNames[attr.Name] {
				for _, v := range attr.Values {
					if v.Value != "" {
						email = v.Value
						break
					}
				}
			}
		}
	}
	if email == "" && subject != "" && fedEmailDomain(subject) != "" {
		email = subject
	}
	if subject == "" {
		subject = email
	}
	return subject, email
}

// fedOauth2Config builds the x/oauth2 Config for an external OIDC connection.
func fedOauth2Config(c *domain.Connection) (*oauth2.Config, *domain.FederationOidcConfig, error) {
	if c.Config == nil || c.Config.Oidc == nil {
		return nil, nil, domain.ErrProviderError
	}
	cfg := c.Config.Oidc
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.AuthURL,
			TokenURL: cfg.TokenURL,
		},
	}, cfg, nil
}

// fedVerifyIDToken verifies an OIDC id_token against the provider's JWKS
// (cfg.JWKSURL) and returns its claims. The signature is checked with the
// provider keys; issuer/audience are checked against the connection config.
func fedVerifyIDToken(ctx context.Context, cfg *domain.FederationOidcConfig, rawIDToken string) (map[string]any, error) {
	if cfg.JWKSURL == "" {
		return nil, domain.ErrProviderError
	}
	set, err := jwk.Fetch(ctx, cfg.JWKSURL)
	if err != nil {
		return nil, domain.ErrProviderError
	}
	tok, err := jwt.Parse([]byte(rawIDToken),
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	claims, err := tokenClaims(tok)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if cfg.Issuer != "" {
		if iss, _ := claims["iss"].(string); iss != cfg.Issuer {
			return nil, domain.ErrInvalidToken
		}
	}
	if cfg.ClientID != "" {
		if !fedAudienceContains(claims["aud"], cfg.ClientID) {
			return nil, domain.ErrInvalidToken
		}
	}
	return claims, nil
}

// fedAudienceContains reports whether the id_token "aud" claim (string or []any)
// contains the expected client id.
func fedAudienceContains(aud any, clientID string) bool {
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == clientID {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == clientID {
				return true
			}
		}
	}
	return false
}

// fedGenerateSPCertificate mints a fresh self-signed RSA-2048 SP signing
// certificate (PEM) and its private key (PKCS#1 PEM), returning both plus the
// SHA-1 fingerprint of the certificate (a stable external reference). Used by
// RotateConnectionCertificate to roll the SP's SAML signing material.
func fedGenerateSPCertificate(commonName string) (certPEM, keyPEM, fingerprint string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", "", err
	}
	now := nowUTC()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.AddDate(2, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return "", "", "", err
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}))
	sum := sha1.Sum(der)
	fingerprint = hex.EncodeToString(sum[:])
	return certPEM, keyPEM, fingerprint, nil
}

// xmlMarshalIndent renders a SAML metadata document to a pretty-printed XML
// document with the standard declaration prepended.
func xmlMarshalIndent(v any) ([]byte, error) {
	body, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), body...)
	return out, nil
}

// ===========================================================================
// FederationConnections
// ===========================================================================

// pgFederationConnections persists the SSO connection aggregate plus the domains
// and SCIM tokens bound to it. Every method is scoped to a project (tenant
// boundary); a row whose project_id does not match is treated as not-found.
type pgFederationConnections struct {
	db      *DB
	emitter Emitter
}

// NewPgFederationConnections builds the Postgres-backed FederationConnections adapter.
func NewPgFederationConnections(db *DB, emitter Emitter) *pgFederationConnections {
	return &pgFederationConnections{db: db, emitter: emitter}
}

var _ api.FederationConnections = (*pgFederationConnections)(nil)

func (a *pgFederationConnections) CreateConnection(ctx context.Context, cmd domain.ConnectionCmd) (*domain.Connection, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Connection, error) {
		conn := &domain.Connection{
			ID:        newUUID(),
			ProjectID: cmd.ProjectID,
			Type:      cmd.Type,
			Name:      cmd.Name,
			Status:    "active",
			Domains:   cmd.Domains,
		}
		setter, err := fedConnSetter(a.db.Cipher, conn)
		if err != nil {
			return nil, err
		}
		if _, err := models.IamSsoConnections.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.connection.created",
			ProjectID:   conn.ProjectID,
			Environment: "",
			AggregateID: conn.ID,
			Payload:     conn,
		}); err != nil {
			return nil, err
		}
		return conn, nil
	})
}

func (a *pgFederationConnections) GetConnection(ctx context.Context, projectID, id string) (*domain.Connection, error) {
	row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), id)
	if err != nil {
		if errors.Is(translatePgErr("connection", err), ErrNotFound) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, domain.ErrConnectionNotFound
	}
	return fedConnectionFromRow(a.db.Cipher, row)
}

func (a *pgFederationConnections) ListConnections(ctx context.Context, projectID string) ([]domain.Connection, error) {
	rows, err := models.IamSsoConnections.Query(
		sm.Where(models.IamSsoConnections.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Connection, 0, len(rows))
	for _, row := range rows {
		c, err := fedConnectionFromRow(a.db.Cipher, row)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

func (a *pgFederationConnections) UpdateConnection(ctx context.Context, cmd domain.FederationConnectionUpdateCmd) (*domain.Connection, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Connection, error) {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), cmd.ID)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return nil, domain.ErrConnectionNotFound
			}
			return nil, err
		}
		if row.ProjectID != cmd.ProjectID {
			return nil, domain.ErrConnectionNotFound
		}
		conn, err := fedConnectionFromRow(a.db.Cipher, row)
		if err != nil {
			return nil, err
		}
		fedApplyConnectionPatch(conn, cmd.Patch)
		setter, err := fedConnSetter(a.db.Cipher, conn)
		if err != nil {
			return nil, err
		}
		setter.ID = nil // never re-set the pk on update
		setter.ProjectID = nil
		setter.UpdatedAt = ptr(nowUTC())
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.connection.updated",
			ProjectID:   conn.ProjectID,
			Environment: "",
			AggregateID: conn.ID,
			Payload:     conn,
		}); err != nil {
			return nil, err
		}
		return conn, nil
	})
}

// fedApplyConnectionPatch applies the supplied merge-patch fields onto the
// connection aggregate (only keys the caller provided are touched).
func fedApplyConnectionPatch(c *domain.Connection, patch map[string]any) {
	if patch == nil {
		return
	}
	if v, ok := patch["name"].(string); ok {
		c.Name = v
	}
	if v, ok := patch["status"].(string); ok {
		c.Status = v
	}
	if v, ok := patch["type"].(string); ok {
		c.Type = v
	}
	if v, ok := patch["external_ref"].(string); ok {
		c.ExternalRef = v
	}
	if raw, ok := patch["domains"].([]any); ok {
		doms := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				doms = append(doms, s)
			}
		}
		c.Domains = doms
	} else if doms, ok := patch["domains"].([]string); ok {
		c.Domains = doms
	}
}

func (a *pgFederationConnections) DeleteConnection(ctx context.Context, projectID, id string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), id)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return domain.ErrConnectionNotFound
			}
			return err
		}
		if row.ProjectID != projectID {
			return domain.ErrConnectionNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.connection.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: id,
			Payload:     map[string]any{"id": id, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgFederationConnections) TestConnection(ctx context.Context, projectID, id string) (string, error) {
	conn, err := a.GetConnection(ctx, projectID, id)
	if err != nil {
		return "", err
	}
	// The SSO test leg drives the provider login flow and validates the round
	// trip by building a real provider login URL from the connection's protocol
	// config: a signed SAML AuthnRequest redirect for SAML, or the OIDC authorize
	// URL for OIDC. A config error surfaces as a provider error so the operator
	// sees that the connection is misconfigured.
	switch conn.Type {
	case "saml":
		sp, err := fedSamlServiceProvider(conn)
		if err != nil {
			return "", err
		}
		redirectURL, err := sp.MakeRedirectAuthenticationRequest("test")
		if err != nil {
			return "", domain.ErrSSOError
		}
		return redirectURL.String(), nil
	case "oidc":
		oauthCfg, _, err := fedOauth2Config(conn)
		if err != nil {
			return "", err
		}
		return oauthCfg.AuthCodeURL("test"), nil
	default:
		return "", domain.ErrProviderError
	}
}

func (a *pgFederationConnections) RotateConnectionCertificate(ctx context.Context, projectID, id string) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), id)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return "", domain.ErrConnectionNotFound
			}
			return "", err
		}
		if row.ProjectID != projectID {
			return "", domain.ErrConnectionNotFound
		}
		conn, err := fedConnectionFromRow(a.db.Cipher, row)
		if err != nil {
			return "", err
		}
		// Rotation generates a fresh SP signing keypair (self-signed X.509) and
		// stores it on the connection's SAML config; the new public certificate
		// (PEM) is returned and also advertised in the SP metadata document. The
		// private key never leaves the envelope.
		certPEM, keyPEM, fp, err := fedGenerateSPCertificate(conn.ID)
		if err != nil {
			return "", err
		}
		if conn.Config == nil {
			conn.Config = &domain.FederationConnectionConfig{}
		}
		if conn.Config.Saml == nil {
			conn.Config.Saml = &domain.FederationSamlConfig{}
		}
		conn.Config.Saml.SPCertificatePEM = certPEM
		conn.Config.Saml.SPPrivateKeyPEM = keyPEM
		conn.ExternalRef = fp // fingerprint as the stable external reference
		setter, err := fedConnSetter(a.db.Cipher, conn)
		if err != nil {
			return "", err
		}
		setter.ID = nil
		setter.ProjectID = nil
		setter.UpdatedAt = ptr(nowUTC())
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return "", err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.connection.certificate_rotated",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: conn.ID,
			Payload:     conn,
		}); err != nil {
			return "", err
		}
		return certPEM, nil
	})
}

func (a *pgFederationConnections) AddDomain(ctx context.Context, projectID, connectionID, name string) (*domain.Domain, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Domain, error) {
		dom := &domain.Domain{
			ID:           newUUID(),
			ProjectID:    projectID,
			Domain:       name,
			Status:       "pending",
			ConnectionID: connectionID,
		}
		raw, err := marshal(dom)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamDomainSetter{
			ID:        &dom.ID,
			ProjectID: &dom.ProjectID,
			Domain:    ptr(dom.Domain),
			Status:    ptr(dom.Status),
			Data:      &rm,
		}
		if connectionID != "" {
			v := null.From(connectionID)
			setter.ConnectionID = &v
		}
		if _, err := models.IamDomains.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrDomainTaken
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.domain.added",
			ProjectID:   dom.ProjectID,
			Environment: "",
			AggregateID: dom.ID,
			Payload:     dom,
		}); err != nil {
			return nil, err
		}
		return dom, nil
	})
}

func (a *pgFederationConnections) VerifyDomain(ctx context.Context, projectID, domainID string) (*domain.Domain, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Domain, error) {
		row, err := models.FindIamDomain(ctx, a.db.Bobx(), domainID)
		if err != nil {
			if errors.Is(translatePgErr("domain", err), ErrNotFound) {
				return nil, domain.ErrDomainNotFound
			}
			return nil, err
		}
		if row.ProjectID != projectID {
			return nil, domain.ErrDomainNotFound
		}
		dom, err := fedDomainFromRow(row)
		if err != nil {
			return nil, err
		}
		// Verification of the DNS challenge is performed out of band; here we
		// flip the persisted state to verified.
		dom.Status = "verified"
		raw, err := marshal(dom)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		verifiedAt := null.From(nowUTC())
		setter := &models.IamDomainSetter{
			Status:     ptr("verified"),
			VerifiedAt: &verifiedAt,
			Data:       &rm,
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.domain.verified",
			ProjectID:   dom.ProjectID,
			Environment: "",
			AggregateID: dom.ID,
			Payload:     dom,
		}); err != nil {
			return nil, err
		}
		return dom, nil
	})
}

func (a *pgFederationConnections) ListDomains(ctx context.Context, projectID string) ([]domain.Domain, error) {
	rows, err := models.IamDomains.Query(
		sm.Where(models.IamDomains.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Domain, 0, len(rows))
	for _, row := range rows {
		d, err := fedDomainFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, nil
}

func (a *pgFederationConnections) DeleteDomain(ctx context.Context, projectID, domainID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamDomain(ctx, a.db.Bobx(), domainID)
		if err != nil {
			if errors.Is(translatePgErr("domain", err), ErrNotFound) {
				return domain.ErrDomainNotFound
			}
			return err
		}
		if row.ProjectID != projectID {
			return domain.ErrDomainNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.domain.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: domainID,
			Payload:     map[string]any{"id": domainID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgFederationConnections) CreateScimToken(ctx context.Context, cmd domain.FederationScimTokenCmd) (*domain.ScimToken, string, error) {
	type result struct {
		tok    *domain.ScimToken
		secret string
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// Verify the connection exists and is in the requested project before
		// minting a credential scoped to it (tenant boundary).
		connRow, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), cmd.ConnectionID)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return result{}, domain.ErrConnectionNotFound
			}
			return result{}, err
		}
		if connRow.ProjectID != cmd.ProjectID {
			return result{}, domain.ErrConnectionNotFound
		}

		secret, err := fedRandomToken()
		if err != nil {
			return result{}, err
		}
		tok := &domain.ScimToken{
			ID:           newUUID(),
			ProjectID:    cmd.ProjectID,
			ConnectionID: cmd.ConnectionID,
			Name:         cmd.Name,
			ExpiresAt:    cmd.ExpiresAt,
		}
		raw, err := marshal(tok)
		if err != nil {
			return result{}, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimTokenSetter{
			ID:           &tok.ID,
			ProjectID:    &tok.ProjectID,
			ConnectionID: &tok.ConnectionID,
			Hash:         ptr(fedHashToken(secret)), // store only the hash, never plaintext
			Data:         &rm,
		}
		if _, err := models.IamScimTokens.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return result{}, domain.ErrConflict
			}
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.scim_token.created",
			ProjectID:   tok.ProjectID,
			Environment: "",
			AggregateID: tok.ID,
			Payload:     tok,
		}); err != nil {
			return result{}, err
		}
		return result{tok: tok, secret: secret}, nil
	})
	if err != nil {
		return nil, "", err
	}
	return res.tok, res.secret, nil
}

func (a *pgFederationConnections) ListScimTokens(ctx context.Context, projectID, connectionID string) ([]domain.ScimToken, error) {
	rows, err := models.IamScimTokens.Query(
		sm.Where(models.IamScimTokens.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamScimTokens.Columns.ConnectionID.EQ(psql.Arg(connectionID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScimToken, 0, len(rows))
	for _, row := range rows {
		t, err := fedTokenFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, nil
}

func (a *pgFederationConnections) DeleteScimToken(ctx context.Context, projectID, connectionID, tokenID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamScimToken(ctx, a.db.Bobx(), tokenID)
		if err != nil {
			if errors.Is(translatePgErr("scim_token", err), ErrNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.ConnectionID != connectionID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.scim_token.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: tokenID,
			Payload:     map[string]any{"id": tokenID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (a *pgFederationConnections) ResolveConnection(ctx context.Context, projectID, email string) (*domain.Connection, error) {
	// Route the email's domain to its verified connection: find the domain row
	// for the (project, host), then load the connection it binds to.
	host := fedEmailDomain(email)
	if host == "" {
		return nil, domain.ErrConnectionNotFound
	}
	domRow, err := models.IamDomains.Query(
		sm.Where(models.IamDomains.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamDomains.Columns.Domain.EQ(psql.Arg(host))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if errors.Is(translatePgErr("domain", err), ErrNotFound) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	connID, ok := domRow.ConnectionID.Get()
	if !ok || connID == "" {
		return nil, domain.ErrConnectionNotFound
	}
	return a.GetConnection(ctx, projectID, connID)
}

// fedEmailDomain extracts the host portion of an email address (lowercased by
// the caller's expectation; stored domains are matched verbatim).
func fedEmailDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return ""
}

// ===========================================================================
// FederationRuntime — SSO outbound/inbound legs + exchange
// ===========================================================================

// pgFederationRuntime drives the OIDC / SAML authentication legs. The protocol
// crypto is implemented with github.com/crewjam/saml (AuthnRequest build/sign,
// signed-assertion verification, SP metadata) and golang.org/x/oauth2 + jwx
// (OIDC code exchange + id_token JWKS verification). After verification the
// external subject is provisioned (find/create iam_users + iam_identities) and a
// session is minted; a single-use exchange code is stored in iam_auth_codes and
// resolved by Exchange (consume -> account + session).
type pgFederationRuntime struct {
	db      *DB
	emitter Emitter
}

const (
	// fedDefaultEnv is the environment whose signing key mints the access token for
	// a federated (SSO) login session.
	fedDefaultEnv = "live"

	// fedAccessTTL / fedRefreshTTL bound the access and refresh JWTs minted for an
	// SSO-provisioned session.
	fedAccessTTL  = time.Hour
	fedRefreshTTL = 30 * 24 * time.Hour

	// fedExchangeCodeTTL bounds the single-use exchange code that maps a verified
	// external subject's minted session to the /v1/sso/exchange leg.
	fedExchangeCodeTTL = 5 * time.Minute
)

// NewPgFederationRuntime builds the Postgres-backed FederationRuntime adapter.
func NewPgFederationRuntime(db *DB, emitter Emitter) *pgFederationRuntime {
	return &pgFederationRuntime{db: db, emitter: emitter}
}

var _ api.FederationRuntime = (*pgFederationRuntime)(nil)

// fedConnByID loads a connection by id without a project filter (the runtime
// legs are reached by connection id from the public surface).
func (a *pgFederationRuntime) fedConnByID(ctx context.Context, connectionID string) (*models.IamSsoConnection, error) {
	row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), connectionID)
	if err != nil {
		if errors.Is(translatePgErr("connection", err), ErrNotFound) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	return row, nil
}

func (a *pgFederationRuntime) OidcStart(ctx context.Context, cmd domain.FederationSsoStartCmd) (*domain.FederationSsoRedirect, error) {
	row, err := a.fedConnByID(ctx, cmd.ConnectionID)
	if err != nil {
		return nil, err
	}
	conn, err := fedConnectionFromRow(a.db.Cipher, row)
	if err != nil {
		return nil, err
	}
	oauthCfg, _, err := fedOauth2Config(conn)
	if err != nil {
		return nil, err
	}
	// Build the OIDC authorization-code authorize URL with state. The caller's
	// RedirectTo is carried back through the state token; we mint an opaque,
	// unguessable state value.
	state := cmd.State
	if state == "" {
		state, err = fedRandomToken()
		if err != nil {
			return nil, err
		}
	}
	opts := []oauth2.AuthCodeOption{}
	if cmd.LoginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", cmd.LoginHint))
	}
	authURL := oauthCfg.AuthCodeURL(state, opts...)
	return &domain.FederationSsoRedirect{URL: authURL}, nil
}

func (a *pgFederationRuntime) OidcCallback(ctx context.Context, cmd domain.FederationSsoCallbackCmd) (*domain.FederationSsoRedirect, error) {
	row, err := a.fedConnByID(ctx, cmd.ConnectionID)
	if err != nil {
		return nil, err
	}
	conn, err := fedConnectionFromRow(a.db.Cipher, row)
	if err != nil {
		return nil, err
	}
	oauthCfg, oidcCfg, err := fedOauth2Config(conn)
	if err != nil {
		return nil, err
	}
	// Exchange the authorization code at the provider token endpoint.
	tok, err := oauthCfg.Exchange(ctx, cmd.Code)
	if err != nil {
		return nil, domain.ErrProviderError
	}
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, domain.ErrSSOError
	}
	// Verify the id_token signature against the provider JWKS and check iss/aud.
	claims, err := fedVerifyIDToken(ctx, oidcCfg, rawIDToken)
	if err != nil {
		return nil, err
	}
	subject, _ := claims["sub"].(string)
	if subject == "" {
		return nil, domain.ErrSSOError
	}
	email, _ := claims["email"].(string)
	// The id_token is verified; the external subject is now trusted. Provision/link
	// the verified subject to an IAM user + session, then persist a single-use
	// exchange code (code -> minted session) for the provisioning leg to resolve.
	// The provider key is the issuer when known, else the connection id.
	provider := oidcCfg.Issuer
	if provider == "" {
		provider = cmd.ConnectionID
	}
	// Provision + emit atomically: the callback event is recorded iff the
	// exchange code commits (nested withTx joins fedProvisionAndStoreCode's tx).
	var exchangeCode string
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		code, err := a.fedProvisionAndStoreCode(ctx, row.ProjectID, cmd.ConnectionID, provider, "oidc", subject, email)
		if err != nil {
			return err
		}
		exchangeCode = code
		return a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.sso.oidc_callback",
			ProjectID:   row.ProjectID,
			AggregateID: cmd.ConnectionID,
			Payload:     map[string]any{"connection_id": cmd.ConnectionID, "subject": subject, "email": email, "provider": provider},
		})
	}); err != nil {
		return nil, err
	}
	return &domain.FederationSsoRedirect{
		URL: "/v1/sso/exchange?code=" + exchangeCode,
	}, nil
}

func (a *pgFederationRuntime) SamlLogin(ctx context.Context, cmd domain.FederationSsoStartCmd) (*domain.FederationSsoRedirect, error) {
	row, err := a.fedConnByID(ctx, cmd.ConnectionID)
	if err != nil {
		return nil, err
	}
	conn, err := fedConnectionFromRow(a.db.Cipher, row)
	if err != nil {
		return nil, err
	}
	sp, err := fedSamlServiceProvider(conn)
	if err != nil {
		return nil, err
	}
	// Build the SAML AuthnRequest and the redirect URL to the IdP SSO endpoint.
	// When the connection carries an SP signing keypair the request is signed by
	// the library; RelayState carries the caller's post-login target.
	relayState := cmd.State
	if relayState == "" {
		relayState = cmd.RedirectTo
	}
	redirectURL, err := sp.MakeRedirectAuthenticationRequest(relayState)
	if err != nil {
		return nil, domain.ErrSSOError
	}
	// Record the outstanding RelayState so the ACS can reject assertions that
	// did not originate from an SP-initiated request (IdP-initiated CSRF).
	if err := a.fedStoreSamlRequest(ctx, row.ProjectID, cmd.ConnectionID, relayState, cmd.RedirectTo); err != nil {
		return nil, err
	}
	return &domain.FederationSsoRedirect{URL: redirectURL.String()}, nil
}

func (a *pgFederationRuntime) SamlAcs(ctx context.Context, cmd domain.FederationSamlAcsCmd) (*domain.FederationSsoRedirect, error) {
	row, err := a.fedConnByID(ctx, cmd.ConnectionID)
	if err != nil {
		return nil, err
	}
	conn, err := fedConnectionFromRow(a.db.Cipher, row)
	if err != nil {
		return nil, err
	}
	sp, err := fedSamlServiceProvider(conn)
	if err != nil {
		return nil, err
	}
	// The cmd carries the raw base64 SAMLResponse posted by the IdP; decode it to
	// the assertion XML and let the library verify the XML-DSig signature, the
	// conditions and the audience.
	decoded, err := base64.StdEncoding.DecodeString(cmd.SAMLResponse)
	if err != nil {
		return nil, domain.ErrSSOError
	}
	// Correlate the RelayState with an outstanding SP-initiated request. A POST
	// with no matching RelayState is an IdP-initiated / forged assertion and is
	// rejected (the library still verifies the XML-DSig signature below).
	correlated, err := a.fedConsumeSamlRequest(ctx, row.ProjectID, cmd.ConnectionID, cmd.RelayState)
	if err != nil {
		return nil, err
	}
	if !correlated {
		return nil, domain.ErrSSOError.WithMessage("unsolicited SAML response")
	}
	assertion, err := sp.ParseXMLResponse(decoded, []string{}, sp.AcsURL)
	if err != nil {
		return nil, domain.ErrSSOError
	}
	// Reject a replayed assertion (single-use within its validity window).
	var notAfter time.Time
	if assertion.Conditions != nil {
		notAfter = assertion.Conditions.NotOnOrAfter
	}
	if err := a.fedAssertNotReplayed(ctx, row.ProjectID, cmd.ConnectionID, assertion.ID, notAfter); err != nil {
		return nil, err
	}
	subject, email := fedSamlSubject(assertion)
	if subject == "" {
		return nil, domain.ErrSSOError
	}
	// The assertion signature is verified; the external subject is now trusted.
	// Provision/link the subject to an IAM user + session, then persist a
	// single-use exchange code (code -> minted session) for the provisioning leg
	// to resolve. The provider key is the connection id (no issuer for SAML).
	// Provision + emit atomically (nested withTx joins fedProvisionAndStoreCode's tx).
	var exchangeCode string
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		code, err := a.fedProvisionAndStoreCode(ctx, row.ProjectID, cmd.ConnectionID, cmd.ConnectionID, "saml", subject, email)
		if err != nil {
			return err
		}
		exchangeCode = code
		return a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.sso.saml_acs",
			ProjectID:   row.ProjectID,
			AggregateID: cmd.ConnectionID,
			Payload:     map[string]any{"connection_id": cmd.ConnectionID, "subject": subject, "email": email},
		})
	}); err != nil {
		return nil, err
	}
	return &domain.FederationSsoRedirect{
		URL: "/v1/sso/exchange?code=" + exchangeCode,
	}, nil
}

func (a *pgFederationRuntime) SamlSlo(ctx context.Context, connectionID string) (*domain.FederationSsoRedirect, error) {
	row, err := a.fedConnByID(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	conn, err := fedConnectionFromRow(a.db.Cipher, row)
	if err != nil {
		return nil, err
	}
	sp, err := fedSamlServiceProvider(conn)
	if err != nil {
		return nil, err
	}
	// Resolve the IdP's Single-Logout endpoint from its metadata. Building the
	// fully-signed LogoutRequest (sp.MakeRedirectLogoutRequest) additionally needs
	// the user's NameID, which this signature (connection id only, no session
	// context) does not carry; the per-subject LogoutRequest is therefore built by
	// the caller leg that holds the session. We return the real IdP SLO location.
	sloLocation := sp.GetSLOBindingLocation(saml.HTTPRedirectBinding)
	if sloLocation == "" {
		// No SLO endpoint advertised by the IdP — nothing to redirect to.
		return nil, domain.ErrSSOError
	}
	// NOTE: the per-subject signed LogoutRequest (sp.MakeRedirectLogoutRequest)
	// needs the user's NameID, which the connection-scoped port signature does
	// not carry; the caller leg holding the session builds it. We return the
	// verified IdP SLO location.
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "federation.sso.saml_slo",
		ProjectID:   row.ProjectID,
		Environment: "",
		AggregateID: connectionID,
		Payload:     map[string]any{"connection_id": connectionID},
	}); err != nil {
		return nil, err
	}
	return &domain.FederationSsoRedirect{URL: sloLocation}, nil
}

func (a *pgFederationRuntime) SamlMetadata(ctx context.Context, connectionID string) ([]byte, error) {
	row, err := a.fedConnByID(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	conn, err := fedConnectionFromRow(a.db.Cipher, row)
	if err != nil {
		return nil, err
	}
	sp, err := fedSamlServiceProvider(conn)
	if err != nil {
		return nil, err
	}
	// Render the SP metadata XML (entityID, ACS URL, signing certificate) from the
	// library. The embedded signing certificate is the connection's SP certificate
	// when stored; when absent the metadata is rendered without a signing
	// KeyDescriptor (the SP simply runs without request signing).
	md := sp.Metadata()
	out, err := xmlMarshalIndent(md)
	if err != nil {
		return nil, domain.ErrSSOError
	}
	return out, nil
}

func (a *pgFederationRuntime) Exchange(ctx context.Context, projectID, code string) (*domain.Account, *domain.Session, error) {
	type result struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// The exchange code is an opaque single-use token persisted (hashed) by the
		// callback leg; resolve it (project-scoped) to the session it authenticated.
		// The session JSON was stored whole in the code's data envelope at callback
		// time (after the external subject was provisioned to an IAM user); here we
		// validate single-use + expiry, consume it, and return account + session.
		hash := fedHashToken(code)
		rows, err := models.IamAuthCodes.Query(
			sm.Where(models.IamAuthCodes.Columns.CodeHash.EQ(psql.Arg(hash))),
			sm.Where(models.IamAuthCodes.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Limit(1),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return result{}, err
		}
		if len(rows) == 0 {
			return result{}, domain.ErrInvalidToken
		}
		row := rows[0]
		if row.Consumed {
			return result{}, domain.ErrInvalidToken
		}
		if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowUTC()) {
			return result{}, domain.ErrInvalidToken
		}
		// Mark consumed (single-use) before handing back the session.
		consumed := true
		if err := row.Update(ctx, a.db.Bobx(), &models.IamAuthCodeSetter{Consumed: &consumed}); err != nil {
			return result{}, err
		}
		var sess domain.Session
		if err := unmarshal(row.Data, &sess); err != nil {
			return result{}, err
		}
		acc, err := a.fedLoadAccount(ctx, projectID, row.UserID.GetOrZero())
		if err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.sso.exchanged",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return result{}, err
		}
		return result{acc: acc, sess: &sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// ===========================================================================
// SSO provisioning + exchange-code store
// ===========================================================================

// fedProvisionAndStoreCode is the post-verification provisioning path shared by
// OidcCallback and SamlAcs: inside one serializable tx it provisions/links the
// verified external subject to an IAM user + session (fedProvisionSubject) and
// persists a single-use exchange code (sha256(code) -> minted session JSON,
// user_id = account id, expires_at = now+5m). The opaque code is returned for the
// /v1/sso/exchange redirect; only its hash is stored.
func (a *pgFederationRuntime) fedProvisionAndStoreCode(ctx context.Context, projectID, connectionID, provider, idType, providerAccountID, email string) (string, error) {
	code, err := fedRandomToken()
	if err != nil {
		return "", err
	}
	err = a.db.withTx(ctx, func(ctx context.Context) error {
		acct, sess, err := a.fedProvisionSubject(ctx, projectID, connectionID, provider, idType, providerAccountID, email)
		if err != nil {
			return err
		}
		raw, err := marshal(sess)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		uid := null.From(acct.ID)
		setter := &models.IamAuthCodeSetter{
			ID:        ptr(newUUID()),
			ProjectID: &projectID,
			CodeHash:  ptr(fedHashToken(code)),
			UserID:    &uid,
			ExpiresAt: ptr(nowUTC().Add(fedExchangeCodeTTL)),
			Data:      &rm,
		}
		if _, err := models.IamAuthCodes.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.sso.exchange_code_issued",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: acct.ID,
			Payload:     map[string]any{"connection_id": connectionID, "user_id": acct.ID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return code, nil
}

// fedProvisionSubject mirrors oauthsocial.resolveLoginAndMint for SSO: it finds an
// iam_identities row by (projectID, provider, providerAccountID); when absent it
// provisions a fresh iam_users account (status active, kind human,
// primary_email=email) and links an iam_identities row (Type "saml" | "oidc");
// otherwise it loads the existing account. It then mints an iam_sessions row +
// signed access-token JWT (fedMintSession). Runs inside the caller's tx.
func (a *pgFederationRuntime) fedProvisionSubject(ctx context.Context, projectID, connectionID, provider, idType, providerAccountID, email string) (*domain.Account, *domain.Session, error) {
	ident, err := a.fedFindIdentity(ctx, projectID, provider, providerAccountID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}
	var acct *domain.Account
	if errors.Is(err, domain.ErrNotFound) {
		acct, err = a.fedCreateAccount(ctx, projectID, email)
		if err != nil {
			return nil, nil, err
		}
		if err := a.fedInsertIdentity(ctx, &domain.Identity{
			ID:                newUUID(),
			Type:              idType,
			Provider:          provider,
			ProviderAccountID: providerAccountID,
			Email:             email,
		}, projectID, acct.ID); err != nil {
			return nil, nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "identity.linked",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: acct.ID,
			Payload:     map[string]any{"user_id": acct.ID, "project_id": projectID, "provider": provider, "provider_account_id": providerAccountID, "id_type": idType},
		}); err != nil {
			return nil, nil, err
		}
	} else {
		acct, err = a.fedLoadAccount(ctx, projectID, ident.UserID)
		if err != nil {
			return nil, nil, err
		}
	}
	sess, err := a.fedMintSession(ctx, acct, idType)
	if err != nil {
		return nil, nil, err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "session.created",
		ProjectID:   acct.ProjectID,
		Environment: "",
		AggregateID: sess.ID,
		Payload:     sess,
	}); err != nil {
		return nil, nil, err
	}
	return acct, sess, nil
}

// fedFindIdentity loads the SSO identity for a (project, provider, providerAccountID)
// triple, mapping no-rows to domain.ErrNotFound. Tenant-scoped by project_id.
func (a *pgFederationRuntime) fedFindIdentity(ctx context.Context, projectID, provider, providerAccountID string) (*models.IamIdentity, error) {
	rows, err := models.IamIdentities.Query(
		sm.Where(models.IamIdentities.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamIdentities.Columns.Provider.EQ(psql.Arg(provider))),
		sm.Where(models.IamIdentities.Columns.ProviderAccountID.EQ(psql.Arg(providerAccountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	return rows[0], nil
}

// fedInsertIdentity writes the SSO provider link row for an account. Lookup
// columns carry the provider correlation; the domain Identity is stored in the
// data envelope.
func (a *pgFederationRuntime) fedInsertIdentity(ctx context.Context, ident *domain.Identity, projectID, userID string) error {
	raw, err := marshal(ident)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamIdentitySetter{
		ID:        &ident.ID,
		ProjectID: &projectID,
		UserID:    &userID,
		Type:      ptr(ident.Type),
		Data:      &rm,
	}
	if ident.Provider != "" {
		v := null.From(ident.Provider)
		setter.Provider = &v
	}
	if ident.ProviderAccountID != "" {
		v := null.From(ident.ProviderAccountID)
		setter.ProviderAccountID = &v
	}
	if ident.Email != "" {
		v := null.From(ident.Email)
		setter.Email = &v
	}
	if _, err := models.IamIdentities.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrIdentityExists
		}
		return err
	}
	return nil
}

// fedCreateAccount provisions a new IAM account for a first-time SSO login.
func (a *pgFederationRuntime) fedCreateAccount(ctx context.Context, projectID, email string) (*domain.Account, error) {
	acct := &domain.Account{
		ID:            newUUID(),
		ProjectID:     projectID,
		Kind:          "human",
		Status:        "active",
		PrimaryEmail:  email,
		EmailVerified: email != "", // IdP-asserted email is treated as verified
		CreatedAt:     nowUTC(),
		UpdatedAt:     nowUTC(),
	}
	raw, err := marshal(acct)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamUserSetter{
		ID:        &acct.ID,
		ProjectID: &acct.ProjectID,
		Kind:      ptr(acct.Kind),
		Status:    ptr(acct.Status),
		Data:      &rm,
	}
	if acct.PrimaryEmail != "" {
		v := null.From(acct.PrimaryEmail)
		setter.PrimaryEmail = &v
	}
	if _, err := models.IamUsers.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrEmailExists
		}
		return nil, err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "user.created",
		ProjectID:   acct.ProjectID,
		Environment: "",
		AggregateID: acct.ID,
		Payload:     acct,
	}); err != nil {
		return nil, err
	}
	return acct, nil
}

// fedLoadAccount reads the account aggregate from iam_users, tenant-scoped.
func (a *pgFederationRuntime) fedLoadAccount(ctx context.Context, projectID, userID string) (*domain.Account, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), userID)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, domain.ErrUserNotFound
	}
	var acct domain.Account
	if err := unmarshal(row.Data, &acct); err != nil {
		return nil, err
	}
	return &acct, nil
}

// fedMintSession creates an iam_sessions row for an account and returns it with a
// signed RS256 JWT access token (minted by the project Signer) plus a refresh
// token signed by the same key. amr carries the SSO method ("saml" | "oidc").
func (a *pgFederationRuntime) fedMintSession(ctx context.Context, acct *domain.Account, amr string) (*domain.Session, error) {
	sessionID := newUUID()
	signEnv, err := resolveSignEnv(ctx, a.db, acct.ProjectID, fedDefaultEnv)
	if err != nil {
		return nil, err
	}
	access, err := a.db.Signer().Sign(ctx, acct.ProjectID, signEnv, map[string]any{
		"iss": acct.ProjectID,
		"sub": acct.ID,
		"sid": sessionID,
		"pid": acct.ProjectID,
		"aal": 1,
		"amr": []string{amr},
		"typ": "access",
		"env": signEnv,
	}, fedAccessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := a.db.Signer().Sign(ctx, acct.ProjectID, signEnv, map[string]any{
		"iss": acct.ProjectID,
		"sub": acct.ID,
		"sid": sessionID,
		"pid": acct.ProjectID,
		"typ": "refresh",
		"env": signEnv,
	}, fedRefreshTTL)
	if err != nil {
		return nil, err
	}
	sess := &domain.Session{
		ID:           sessionID,
		AccountID:    acct.ID,
		ProjectID:    acct.ProjectID,
		AMR:          []string{amr},
		AAL:          1,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(fedAccessTTL / time.Second),
		CreatedAt:    nowUTC(),
	}
	raw, err := marshal(sess)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamSessionSetter{
		ID:        &sess.ID,
		ProjectID: &sess.ProjectID,
		UserID:    &sess.AccountID,
		Aal:       ptr(int32(sess.AAL)),
		Data:      &rm,
	}
	if _, err := models.IamSessions.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		return nil, err
	}
	return sess, nil
}

// ===========================================================================
// FederationScim — connection-scoped SCIM v2 provisioning
// ===========================================================================

// pgFederationScim persists SCIM Users and Groups as free-form attribute maps in
// iam_scim_resources. Every operation is scoped to the connection it is invoked
// for (and, transitively, the connection's project): a resource that does not
// belong to the requested connection is treated as not-found. The SCIM schema
// semantics (id assignment, meta block, list envelope) are owned here.
type pgFederationScim struct {
	db      *DB
	emitter Emitter
}

// NewPgFederationScim builds the Postgres-backed FederationScim adapter.
func NewPgFederationScim(db *DB, emitter Emitter) *pgFederationScim {
	return &pgFederationScim{db: db, emitter: emitter}
}

var _ api.FederationScim = (*pgFederationScim)(nil)

const (
	scimResourceTypeUser  = "User"
	scimResourceTypeGroup = "Group"
)

// fedScimProjectForConnection resolves the project a connection belongs to so
// SCIM rows can be persisted with the right tenant boundary. A missing
// connection is a not-found.
func (a *pgFederationScim) fedScimProjectForConnection(ctx context.Context, connectionID string) (string, error) {
	row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), connectionID)
	if err != nil {
		if errors.Is(translatePgErr("connection", err), ErrNotFound) {
			return "", domain.ErrConnectionNotFound
		}
		return "", err
	}
	return row.ProjectID, nil
}

// fedScimResourceFromRow decodes the stored SCIM attribute map and stamps the
// canonical id / meta so the wire representation is self-consistent.
func fedScimResourceFromRow(row *models.IamScimResource) (map[string]any, error) {
	attrs := map[string]any{}
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &attrs); err != nil {
			return nil, err
		}
	}
	attrs["id"] = row.ID
	if v, ok := row.ExternalID.Get(); ok && v != "" {
		attrs["externalId"] = v
	}
	meta, _ := attrs["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["resourceType"] = row.ResourceType
	meta["created"] = row.CreatedAt.UTC().Format(time.RFC3339)
	meta["lastModified"] = row.UpdatedAt.UTC().Format(time.RFC3339)
	attrs["meta"] = meta
	return attrs, nil
}

// fedScimExternalID pulls the SCIM externalId out of an attribute map, if present.
func fedScimExternalID(attrs map[string]any) string {
	if v, ok := attrs["externalId"].(string); ok {
		return v
	}
	return ""
}

// fedScimListEnvelope wraps resources in the SCIM v2 ListResponse envelope.
func fedScimListEnvelope(resources []map[string]any, startIndex, count int) map[string]any {
	if startIndex <= 0 {
		startIndex = 1
	}
	return map[string]any{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": len(resources),
		"startIndex":   startIndex,
		"itemsPerPage": len(resources),
		"Resources":    resources,
	}
}

// fedScimList is the shared list path for Users and Groups (scoped by connection
// + resource type).
func (a *pgFederationScim) fedScimList(ctx context.Context, q domain.FederationScimListQuery, resourceType string) (map[string]any, error) {
	rows, err := models.IamScimResources.Query(
		sm.Where(models.IamScimResources.Columns.ConnectionID.EQ(psql.Arg(q.ConnectionID))),
		sm.Where(models.IamScimResources.Columns.ResourceType.EQ(psql.Arg(resourceType))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	resources := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m, err := fedScimResourceFromRow(row)
		if err != nil {
			return nil, err
		}
		resources = append(resources, m)
	}
	return fedScimListEnvelope(resources, q.StartIndex, q.Count), nil
}

// fedScimGet loads a single resource scoped to its connection + type.
func (a *pgFederationScim) fedScimGet(ctx context.Context, connectionID, resourceID, resourceType string) (*models.IamScimResource, error) {
	row, err := models.FindIamScimResource(ctx, a.db.Bobx(), resourceID)
	if err != nil {
		if errors.Is(translatePgErr("scim_resource", err), ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if row.ConnectionID != connectionID || row.ResourceType != resourceType { // tenant boundary
		return nil, domain.ErrNotFound
	}
	return row, nil
}

// fedScimCreate inserts a new SCIM resource on a connection (id minted here).
func (a *pgFederationScim) fedScimCreate(ctx context.Context, cmd domain.FederationScimWriteCmd, resourceType string) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		projectID, err := a.fedScimProjectForConnection(ctx, cmd.ConnectionID)
		if err != nil {
			return nil, err
		}
		id := newUUID()
		attrs := map[string]any{}
		for k, v := range cmd.Attributes {
			attrs[k] = v
		}
		attrs["id"] = id

		raw, err := marshal(attrs)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimResourceSetter{
			ID:           &id,
			ProjectID:    &projectID,
			ConnectionID: &cmd.ConnectionID,
			ResourceType: ptr(resourceType),
			Data:         &rm,
		}
		if ext := fedScimExternalID(cmd.Attributes); ext != "" {
			v := null.From(ext)
			setter.ExternalID = &v
		}
		if _, err := models.IamScimResources.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		row, err := models.FindIamScimResource(ctx, a.db.Bobx(), id)
		if err != nil {
			return nil, err
		}
		out, err := fedScimResourceFromRow(row)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.scim.resource_created",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: id,
			Payload:     out,
		}); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// fedScimReplace overwrites a resource's attributes wholesale (PUT semantics).
func (a *pgFederationScim) fedScimReplace(ctx context.Context, cmd domain.FederationScimWriteCmd, resourceType string) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		row, err := a.fedScimGet(ctx, cmd.ConnectionID, cmd.ResourceID, resourceType)
		if err != nil {
			return nil, err
		}
		attrs := map[string]any{}
		for k, v := range cmd.Attributes {
			attrs[k] = v
		}
		attrs["id"] = row.ID

		raw, err := marshal(attrs)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimResourceSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}
		if ext := fedScimExternalID(cmd.Attributes); ext != "" {
			v := null.From(ext)
			setter.ExternalID = &v
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		out, err := fedScimResourceFromRow(row)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.scim.resource_replaced",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload:     out,
		}); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// fedScimPatch applies a shallow attribute merge (the supplied keys overwrite the
// stored ones; this adapter does not interpret SCIM PATCH op/path grammar).
func (a *pgFederationScim) fedScimPatch(ctx context.Context, cmd domain.FederationScimPatchCmd, resourceType string) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		row, err := a.fedScimGet(ctx, cmd.ConnectionID, cmd.ResourceID, resourceType)
		if err != nil {
			return nil, err
		}
		attrs := map[string]any{}
		if len(row.Data) > 0 {
			if err := unmarshal(row.Data, &attrs); err != nil {
				return nil, err
			}
		}
		for k, v := range cmd.Patch {
			attrs[k] = v
		}
		attrs["id"] = row.ID

		raw, err := marshal(attrs)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimResourceSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}
		if ext := fedScimExternalID(attrs); ext != "" {
			v := null.From(ext)
			setter.ExternalID = &v
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		out, err := fedScimResourceFromRow(row)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.scim.resource_patched",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload:     out,
		}); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// fedScimDelete removes a resource scoped to its connection + type.
func (a *pgFederationScim) fedScimDelete(ctx context.Context, connectionID, resourceID, resourceType string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := a.fedScimGet(ctx, connectionID, resourceID, resourceType)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "federation.scim.resource_deleted",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: resourceID,
			Payload:     map[string]any{"id": resourceID, "project_id": row.ProjectID, "connection_id": connectionID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// --- Users ---

func (a *pgFederationScim) ListUsers(ctx context.Context, q domain.FederationScimListQuery) (map[string]any, error) {
	return a.fedScimList(ctx, q, scimResourceTypeUser)
}

func (a *pgFederationScim) GetUser(ctx context.Context, connectionID, scimUserID string) (map[string]any, error) {
	row, err := a.fedScimGet(ctx, connectionID, scimUserID, scimResourceTypeUser)
	if err != nil {
		return nil, err
	}
	return fedScimResourceFromRow(row)
}

func (a *pgFederationScim) CreateUser(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimCreate(ctx, cmd, scimResourceTypeUser)
}

func (a *pgFederationScim) ReplaceUser(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimReplace(ctx, cmd, scimResourceTypeUser)
}

func (a *pgFederationScim) PatchUser(ctx context.Context, cmd domain.FederationScimPatchCmd) (map[string]any, error) {
	return a.fedScimPatch(ctx, cmd, scimResourceTypeUser)
}

func (a *pgFederationScim) DeleteUser(ctx context.Context, connectionID, scimUserID string) error {
	return a.fedScimDelete(ctx, connectionID, scimUserID, scimResourceTypeUser)
}

// --- Groups ---

func (a *pgFederationScim) ListGroups(ctx context.Context, q domain.FederationScimListQuery) (map[string]any, error) {
	return a.fedScimList(ctx, q, scimResourceTypeGroup)
}

func (a *pgFederationScim) GetGroup(ctx context.Context, connectionID, groupID string) (map[string]any, error) {
	row, err := a.fedScimGet(ctx, connectionID, groupID, scimResourceTypeGroup)
	if err != nil {
		return nil, err
	}
	return fedScimResourceFromRow(row)
}

func (a *pgFederationScim) CreateGroup(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimCreate(ctx, cmd, scimResourceTypeGroup)
}

func (a *pgFederationScim) ReplaceGroup(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimReplace(ctx, cmd, scimResourceTypeGroup)
}

func (a *pgFederationScim) PatchGroup(ctx context.Context, cmd domain.FederationScimPatchCmd) (map[string]any, error) {
	return a.fedScimPatch(ctx, cmd, scimResourceTypeGroup)
}

func (a *pgFederationScim) DeleteGroup(ctx context.Context, connectionID, groupID string) error {
	return a.fedScimDelete(ctx, connectionID, groupID, scimResourceTypeGroup)
}
