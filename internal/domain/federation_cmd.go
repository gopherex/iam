package domain

import "time"

// FederationConnectionUpdateCmd is a partial update of an SSO connection. The
// Patch map carries only the fields the caller supplied (decoded from the
// free-form JSON merge-patch body); the adapter applies them atomically.
type FederationConnectionUpdateCmd struct {
	ProjectID string
	ID        string
	Patch     map[string]any
}

// FederationScimTokenCmd mints a new SCIM provisioning bearer token scoped to a
// single SSO connection.
type FederationScimTokenCmd struct {
	ProjectID    string
	ConnectionID string
	Name         string
	ExpiresAt    time.Time
}

// ScimToken is a provisioning credential bound to an SSO connection. The secret
// is returned only at creation time and is never persisted in clear.
type ScimToken struct {
	ID           string
	ProjectID    string
	ConnectionID string
	Name         string
	ExpiresAt    time.Time
}

// FederationSsoStartCmd begins an outbound SSO authentication leg (OIDC
// authorization-code or SAML AuthnRequest). The adapter builds the provider
// redirect URL (with PKCE / RelayState baked in) and returns it; pkg/api just
// orchestrates the 302.
type FederationSsoStartCmd struct {
	ConnectionID string
	RedirectTo   string
	State        string
	LoginHint    string
}

// FederationSsoCallbackCmd completes an inbound OIDC callback. The adapter
// exchanges the code, provisions/links the account and returns the post-login
// redirect URL plus an optional session cookie.
type FederationSsoCallbackCmd struct {
	ConnectionID string
	Code         string
	State        string
}

// FederationSamlAcsCmd completes an inbound SAML assertion consumer leg. The
// adapter validates the assertion, links the account and returns the post-login
// redirect URL plus an optional session cookie.
type FederationSamlAcsCmd struct {
	ConnectionID string
	SAMLResponse string
	RelayState   string
}

// FederationSsoRedirect is a runtime SSO redirect result: the URL the caller's
// browser must be sent to, plus an optional Set-Cookie value.
type FederationSsoRedirect struct {
	URL    string
	Cookie string
}

// FederationScimResource is an opaque SCIM resource (User or Group) carried as a
// free-form attribute map. pkg/api maps it to/from the generated wire maps; the
// adapter owns the SCIM schema semantics.
type FederationScimResource struct {
	Attributes map[string]any
}

// FederationScimWriteCmd creates or replaces a SCIM resource on a connection.
type FederationScimWriteCmd struct {
	ConnectionID string
	ResourceID   string // empty on create
	Attributes   map[string]any
}

// FederationScimPatchCmd applies a SCIM PATCH operation set to a resource.
type FederationScimPatchCmd struct {
	ConnectionID string
	ResourceID   string
	Patch        map[string]any
}

// FederationScimListQuery is a paginated SCIM list request.
type FederationScimListQuery struct {
	ConnectionID string
	Filter       string
	StartIndex   int
	Count        int
}

// FederationConnectionConfig is the protocol-specific provider configuration of
// an SSO connection, persisted whole in the connection's jsonb envelope. Exactly
// one of Saml / Oidc is populated, matching Connection.Type.
type FederationConnectionConfig struct {
	Saml *FederationSamlConfig `json:"saml,omitempty"`
	Oidc *FederationOidcConfig `json:"oidc,omitempty"`
}

// FederationSamlConfig carries the SAML Service Provider's view of an IdP: the
// IdP metadata XML (preferred) or a raw IdP signing certificate, plus the SP
// endpoint URLs used to build the AuthnRequest / ACS / metadata documents.
type FederationSamlConfig struct {
	// IDPMetadataXML is the IdP's SAML metadata document (entityID, SSO URL,
	// signing certificate). When present it is the authoritative source.
	IDPMetadataXML string `json:"idp_metadata_xml,omitempty"`
	// IDPCertificatePEM is a fallback IdP signing certificate (PEM) used when no
	// full metadata document is available.
	IDPCertificatePEM string `json:"idp_certificate_pem,omitempty"`
	// EntityID is the SP entity ID advertised to the IdP. Defaults to MetadataURL.
	EntityID string `json:"entity_id,omitempty"`
	// AcsURL is the SP Assertion Consumer Service URL (where the IdP POSTs the
	// signed Response).
	AcsURL string `json:"acs_url,omitempty"`
	// MetadataURL is the SP metadata endpoint URL.
	MetadataURL string `json:"metadata_url,omitempty"`
	// SPCertificatePEM / SPPrivateKeyPEM is the SP's own signing keypair used to
	// sign AuthnRequests and to advertise the SP signing certificate in metadata.
	// Optional: when absent the SP runs without request signing.
	SPCertificatePEM string `json:"sp_certificate_pem,omitempty"`
	SPPrivateKeyPEM  string `json:"sp_private_key_pem,omitempty"`
}

// FederationOidcConfig carries the external OIDC provider settings used to drive
// the authorization-code leg and to verify the returned id_token.
type FederationOidcConfig struct {
	Issuer       string   `json:"issuer,omitempty"`
	AuthURL      string   `json:"auth_url,omitempty"`
	TokenURL     string   `json:"token_url,omitempty"`
	JWKSURL      string   `json:"jwks_url,omitempty"`
	ClientID     string   `json:"client_id,omitempty"`
	ClientSecret string   `json:"client_secret,omitempty"`
	RedirectURL  string   `json:"redirect_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}
