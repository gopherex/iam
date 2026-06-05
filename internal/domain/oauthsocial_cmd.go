package domain

// OAuthSocialExchangeCmd exchanges a one-time OAuth authorization code for a
// session. ProjectID scopes the exchange to the client's project; CodeVerifier
// carries the optional PKCE verifier.
type OAuthSocialExchangeCmd struct {
	ProjectID    string
	Code         string
	CodeVerifier string
}
