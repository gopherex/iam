package domain

// OAuthSocialExchangeCmd exchanges a one-time OAuth authorization code for a
// session. ProjectID scopes the exchange to the client's project; CodeVerifier
// carries the optional PKCE verifier.
type OAuthSocialExchangeCmd struct {
	ProjectID    string
	Code         string
	CodeVerifier string
}

// OAuthSocialStartCmd begins a browser-driven social login: the adapter builds
// the provider authorize URL (carrying state/PKCE/prompt hints) and returns it
// for a 302 redirect. RedirectTo is the product URL the provider callback will
// ultimately bounce the user back to.
type OAuthSocialStartCmd struct {
	ProjectID     string
	Provider      string
	RedirectTo    string
	State         string
	CodeChallenge string
	Prompt        string
	LoginHint     string
}

// OAuthSocialCallbackCmd carries the provider callback query parameters back to
// the adapter, which validates state, exchanges the code, mints a session, and
// returns the product redirect URL (plus an optional Set-Cookie value in cookie
// mode).
type OAuthSocialCallbackCmd struct {
	ProjectID    string
	Provider     string
	Code         string
	State        string
	Error        string
	CodeVerifier string // optional PKCE verifier paired with the StartLogin challenge
	RedirectTo   string // product URL to bounce the browser back to after minting
}

// OAuthSocialCallbackResult is the outcome of a social-login callback: the URL
// to redirect the browser to and, in cookie mode, the Set-Cookie header value.
type OAuthSocialCallbackResult struct {
	RedirectURL string
	SetCookie   string
}

// OAuthSocialLinkStartCmd begins linking a provider identity to an already
// authenticated account. AccountID comes from the principal, never the request.
type OAuthSocialLinkStartCmd struct {
	AccountID  string
	ProjectID  string
	Provider   string
	RedirectTo string
	State      string
}

// OAuthSocialLinkCallbackCmd carries the provider callback for an account-link
// flow; the adapter attaches the identity and returns the product redirect URL.
type OAuthSocialLinkCallbackCmd struct {
	AccountID    string
	ProjectID    string
	Provider     string
	Code         string
	State        string
	Error        string
	CodeVerifier string // optional PKCE verifier paired with the StartLink challenge
	RedirectTo   string // product URL to bounce the browser back to after linking
}
