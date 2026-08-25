package domain

import "time"

// OIDCConsentCmd records the resource-owner's consent decision for an OIDC
// authorization interaction. AccountID and SessionID identify the authenticated
// caller (taken from the principal, never the body) and let the adapter verify
// the interaction belongs to this session before recording consent. Remember
// persists the grant so the client is not prompted again.
type OIDCConsentCmd struct {
	InteractionID string
	AccountID     string
	SessionID     string
	GrantedScopes []string
	Remember      bool
}

// OIDCRejectCmd cancels an OIDC authorization interaction. It is a public
// operation (no authenticated caller): Error and ErrorDescription carry the
// OAuth2 error payload that is propagated back to the client via the redirect.
type OIDCRejectCmd struct {
	InteractionID    string
	Error            string
	ErrorDescription string
}

// OIDCAuthorizeCmd is the front-channel /oauth2/authorize request. It is a
// public operation: the client is identified by ClientID and the request is
// scoped to that client's project. The adapter validates the client, builds an
// interaction and returns the redirect URL the user-agent must follow (to the
// login/consent UI, or back to the client with code/error).
type OIDCAuthorizeCmd struct {
	ClientID     string
	ResponseType string
	RedirectURI  string
	Scope        string
	State        string
	// CodeChallenge / CodeChallengeMethod carry PKCE (RFC 7636). The challenge is
	// bound to the issued authorization code and the token endpoint refuses the
	// exchange without the matching verifier.
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	Prompt              string
	// MaxAge bounds how old the relied-upon authentication may be, in seconds.
	// Negative means the client did not ask.
	MaxAge int
	// ResponseMode selects how the response is returned: query (default) or
	// fragment.
	ResponseMode string
	RequestURI   string
	// Request is a signed request object (RFC 9101) carrying the whole
	// authorization request as a JWT the client signed.
	Request string
}

// OIDCFormPost is an authorization response delivered as an auto-submitting
// HTML form (OAuth 2.0 Form Post Response Mode) instead of a redirect. The
// response parameters travel in a POST body, so they never appear in the
// browser's address bar, its history, or the Referer header of the page the
// client renders next.
type OIDCFormPost struct {
	// Action is the client's registered redirect_uri, the form's target.
	Action string
	// Fields are the response parameters, posted as form fields.
	Fields map[string]string
}

// OIDCAuthorizeResult is how an authorization request is answered: either the
// user-agent is redirected somewhere (the interaction UI, or the client's
// redirect_uri carrying the response), or the response is form-posted back.
// Exactly one of the two is set.
type OIDCAuthorizeResult struct {
	RedirectTo string
	FormPost   *OIDCFormPost
}

// OIDCClientRegistration is a client's metadata in the shape RFC 7591 defines.
// It is deliberately its own type rather than AppClient: the registration API is
// a public contract with standard field names, and letting the storage type
// drive it would leak our naming into somebody else's client library.
type OIDCClientRegistration struct {
	ClientID                string
	ClientName              string
	RedirectURIs            []string
	ApplicationType         string
	TokenEndpointAuthMethod string
	GrantTypes              []string
	ResponseTypes           []string
	Scope                   []string
	JWKS                    string
	JWKSURI                 string
	PostLogoutRedirectURIs  []string
	BackchannelLogoutURI    string
	// IssuedAt is when the client was registered.
	IssuedAt time.Time
	// ClientSecret and RegistrationAccessToken are returned once, at
	// registration; afterwards only their digests are kept.
	ClientSecret            string
	RegistrationAccessToken string
	RegistrationClientURI   string
}

// OIDCClientRegisterCmd registers a client inside a project environment.
type OIDCClientRegisterCmd struct {
	ProjectID    string
	Environment  string
	Registration OIDCClientRegistration
}

// OIDCClientUpdateCmd replaces a registered client's metadata. RFC 7592 makes
// this a replacement, not a patch: a field left out is cleared.
type OIDCClientUpdateCmd struct {
	ClientID     string
	ProjectID    string
	Registration OIDCClientRegistration
}

// OIDCLogoutResult is the outcome of an RP-initiated logout: where to send the
// browser, and whether its session cookies should be cleared on the way.
type OIDCLogoutResult struct {
	RedirectURL         string
	ClearSessionCookies bool
}

// OIDCInteractionContext is everything the hosted login/consent UI needs to
// render an interaction: which application is asking, what it is asking for, and
// which project's language the page should speak. The interaction id alone is a
// handle — the UI cannot show "Sign in to <app>" without this.
type OIDCInteractionContext struct {
	Interaction Interaction
	// ProjectID / Environment are the tenant the interaction belongs to. The
	// page needs them for the X-Client-Id it sends on the CSRF and interaction
	// calls.
	ProjectID   string
	Environment string
	// ClientName is the application's display name, ClientType its kind
	// (spa/native/web/machine).
	ClientName string
	ClientType string
	// DefaultLocale / SupportedLocales come from the project's auth config, so
	// the hosted pages speak the project's language rather than the browser's.
	DefaultLocale    string
	SupportedLocales []string
	// ExpiresAt is when the interaction stops being resumable.
	ExpiresAt time.Time
	// Stage tells the UI what is still missing: OIDCStageLogin while nobody has
	// claimed the interaction, OIDCStageConsent once a user has signed in.
	Stage string
}

// Stages of an authorization interaction, as reported to the hosted UI.
const (
	OIDCStageLogin   = "login"
	OIDCStageConsent = "consent"
)

// OIDCLogoutCmd is the front-channel RP-initiated logout request
// (/oauth2/logout). It is public. The adapter terminates the session referenced
// by IDTokenHint and returns the post-logout redirect URL.
type OIDCLogoutCmd struct {
	IDTokenHint           string
	PostLogoutRedirectURI string
	State                 string
}

// OIDCTokenCmd is the /oauth2/token request (RFC 6749 + extensions). It is a
// client-authenticated, public-facing endpoint: the client is identified by
// ClientID/ClientSecret in the body (or via HTTP basic auth at the adapter).
// The adapter dispatches on GrantType and returns the raw token response map.
type OIDCTokenCmd struct {
	ProjectID    string // request tenant (authenticated client), NOT the token
	Env          string
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	ClientID     string
	ClientSecret string
	DeviceCode   string
	// ClientAssertionType / ClientAssertion carry private_key_jwt client
	// authentication (RFC 7523 §3): the client signs a short-lived JWT with a key
	// only it holds instead of presenting a shared secret.
	ClientAssertionType string
	ClientAssertion     string
	// AuthenticatedClientID is the client the TRANSPORT already authenticated —
	// client_secret_basic (RFC 6749 §2.3.1), the method most OAuth clients use by
	// default. It is empty when the client sent its credentials in the form body
	// (client_secret_post) instead, in which case ClientSecret carries them.
	AuthenticatedClientID string
	// Scope narrows a client_credentials request. It can only ever ask for less
	// than the caller already holds.
	Scope string
}

// OIDCIntrospectCmd is the /oauth2/introspect request (RFC 7662). It is
// client-authenticated. The adapter returns the introspection response map,
// including the mandatory "active" flag.
type OIDCIntrospectCmd struct {
	ProjectID string // request tenant (authenticated client), NOT the token
	Env       string
	Token     string
}

// OIDCRevokeCmd is the /oauth2/revoke request (RFC 7009). It is
// client-authenticated. TokenTypeHint is optional.
type OIDCRevokeCmd struct {
	ProjectID     string // request tenant (authenticated client), NOT the token
	Env           string
	Token         string
	TokenTypeHint string
}

// OIDCParCmd is a pushed authorization request (RFC 9126). It mirrors the
// authorization request parameters and is client-authenticated. The adapter
// stores the request and returns a request_uri reference plus its lifetime.
type OIDCParCmd struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	ResponseMode        string
	Prompt              string
	LoginHint           string
}

// OIDCParResult is the result of a pushed authorization request: an opaque
// request_uri the client passes to /oauth2/authorize, and its lifetime in
// seconds.
type OIDCParResult struct {
	RequestURI string
	ExpiresIn  int
}

// OIDCDeviceAuthorizationCmd is the /oauth2/device_authorization request
// (RFC 8628). It is client-authenticated.
type OIDCDeviceAuthorizationCmd struct {
	ClientID string
	Scope    string
}

// OIDCDeviceAuthorization is the device authorization response (RFC 8628).
type OIDCDeviceAuthorization struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int
	Interval                int
}

// OIDCBackchannelLogoutCmd carries the back-channel logout token (RFC) the OP
// receives. It is public; the adapter validates the logout_token and terminates
// the referenced sessions.
type OIDCBackchannelLogoutCmd struct {
	LogoutToken string
}

// OIDCDeviceUserCode identifies a pending device authorization by its
// user-facing code, scoped to the project that owns the requesting client.
type OIDCDeviceUserCode struct {
	ProjectID string
	UserCode  string
}

// OIDCDevicePending is the pending device authorization shown to the user at
// the verification UI before they approve or deny it.
type OIDCDevicePending struct {
	ClientID  string
	ClientMap map[string]any
	Scopes    []string
	ExpiresAt time.Time
}

// OIDCDeviceDecisionCmd records a logged-in user's approve/deny decision for a
// pending device authorization. AccountID/SessionID come from the principal.
type OIDCDeviceDecisionCmd struct {
	ProjectID string
	UserCode  string
	AccountID string
	SessionID string
}
