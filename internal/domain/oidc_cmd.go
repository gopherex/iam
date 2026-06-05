package domain

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
