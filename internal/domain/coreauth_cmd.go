package domain

// Command value-objects for the Core Auth service's verification, password,
// session step-up and token operations. Names are prefixed CoreAuth* to avoid
// collisions with commands owned by other services.

// CoreAuthVerifyStartCmd starts an email or phone verification/change flow.
// One challenge is minted and (for email/phone) a code or link is dispatched.
type CoreAuthVerifyStartCmd struct {
	ProjectID  string
	AccountID  string // empty for unauthenticated start (public verification)
	Contact    string // the email or phone to verify; empty -> use the account's primary
	RedirectTo string
	Locale     string
	Channel    string // sms | whatsapp (phone only)
}

// CoreAuthVerifyConsumeCmd consumes a verification/change challenge. Exactly one
// of (ChallengeID+Code) or Token identifies the challenge.
type CoreAuthVerifyConsumeCmd struct {
	ProjectID   string
	AccountID   string // empty for unauthenticated consume
	ChallengeID string
	Code        string
	Token       string
}

// CoreAuthPasswordForgotCmd begins a password-reset flow for an email.
type CoreAuthPasswordForgotCmd struct {
	ProjectID    string
	Email        string
	RedirectTo   string
	Locale       string
	CaptchaToken string
}

// CoreAuthPasswordResetCmd completes a password-reset flow with a fresh
// password. Exactly one of Token or (ChallengeID+Code) identifies the request.
type CoreAuthPasswordResetCmd struct {
	ProjectID   string
	Token       string
	ChallengeID string
	Code        string
	NewPassword string
}

// CoreAuthPasswordChangeCmd changes the current account's password.
type CoreAuthPasswordChangeCmd struct {
	AccountID           string
	SessionID           string
	CurrentPassword     string
	NewPassword         string
	RevokeOtherSessions bool
}

// CoreAuthPasswordCheckResult is the outcome of a policy/strength check.
type CoreAuthPasswordCheckResult struct {
	Valid      bool
	Score      int
	Violations []string
}

// CoreAuthPasswordVerifyResult is the outcome of re-asserting the current
// account's password (e.g. for a sudo/step-up gate).
type CoreAuthPasswordVerifyResult struct {
	OK  bool
	AAL int
	AMR []string
}

// CoreAuthPasswordResult is the outcome of a password sign-in. When MFARequired
// is true the password (factor 1) verified but the account has an enrolled MFA
// factor, so no session is issued yet: the caller must complete a second factor.
// Session is non-nil only when MFARequired is false. Factors lists the account's
// active MFA factors so the client can choose which to use.
type CoreAuthPasswordResult struct {
	Account     *Account
	Session     *Session
	MFARequired bool
	Factors     []Factor
}

// CoreAuthStepUpCmd requests elevation of the current session to a higher AAL.
type CoreAuthStepUpCmd struct {
	AccountID     string
	SessionID     string
	Purpose       string
	RequiredAAL   int
	MaxAgeSeconds int
	HasMaxAge     bool
}

// CoreAuthStepUpResult is the outcome of a step-up request. When Satisfied is
// true the session already meets the bar; otherwise a Challenge gates it.
type CoreAuthStepUpResult struct {
	Satisfied bool
	Challenge *Challenge
}

// CoreAuthAccessRequest is a request for access to a project that gates
// self-service sign-up behind approval.
type CoreAuthAccessRequest struct {
	ID        string
	ProjectID string
	Email     string
	Reason    string
	Status    string
}

// CoreAuthAccessRequestCmd creates an access request.
type CoreAuthAccessRequestCmd struct {
	ProjectID    string
	Email        string
	Reason       string
	CaptchaToken string
}

// CoreAuthTokenIntrospection is the result of introspecting a token.
type CoreAuthTokenIntrospection struct {
	Active bool
	Claims map[string]any
}

// CoreAuthTokenVerification is the result of verifying a token.
type CoreAuthTokenVerification struct {
	Valid  bool
	Claims map[string]any
	Error  string
}

// CoreAuthRevokeCmd revokes a token or a whole session.
type CoreAuthRevokeCmd struct {
	Token     string
	SessionID string
	Reason    string
}

// CoreAuthEmailVerificationCallbackCmd consumes an opaque email-verification
// link token and resolves where to send the browser next.
type CoreAuthEmailVerificationCallbackCmd struct {
	Token      string
	RedirectTo string
}

// CoreAuthEmailVerificationCallbackResult is the outcome of consuming an email
// verification link: the absolute redirect URL and an optional Set-Cookie value
// (e.g. a freshly issued session cookie) for the browser.
type CoreAuthEmailVerificationCallbackResult struct {
	RedirectURL string
	SetCookie   string
}

// CoreAuthCaptchaVerifyResult is the outcome of verifying a CAPTCHA token with
// the configured provider.
type CoreAuthCaptchaVerifyResult struct {
	Valid bool
	Score float64
}
