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
