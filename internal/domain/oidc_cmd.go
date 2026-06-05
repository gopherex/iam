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
	ClientID      string
	ResponseType  string
	RedirectURI   string
	Scope         string
	State         string
	CodeChallenge string
	Nonce         string
	Prompt        string
	RequestURI    string
}

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
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	ClientID     string
	ClientSecret string
	DeviceCode   string
}

// OIDCIntrospectCmd is the /oauth2/introspect request (RFC 7662). It is
// client-authenticated. The adapter returns the introspection response map,
// including the mandatory "active" flag.
type OIDCIntrospectCmd struct {
	Token string
}

// OIDCRevokeCmd is the /oauth2/revoke request (RFC 7009). It is
// client-authenticated. TokenTypeHint is optional.
type OIDCRevokeCmd struct {
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
	Request             string
	ClientAssertionType string
	ClientAssertion     string
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
