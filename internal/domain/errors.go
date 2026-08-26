package domain

import "net/http"

// Error is a domain error carrying the stable machine code, an HTTP status and
// a default human message. The persistence/service layers return these; the API
// layer (pkg/api NewError) renders them into the ErrorEnvelope. Equality is by
// Code, so errors.Is(err, ErrInvalidCredentials) matches any *Error with the
// same code, whether a sentinel or a freshly built one with details.
type Error struct {
	Code    string
	Message string
	Status  int
	Details map[string]any
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Is matches by Code so wrapped/rebuilt errors still compare equal to the
// sentinel of the same code.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

// WithMessage returns a copy with an overridden human message.
func (e *Error) WithMessage(msg string) *Error {
	c := *e
	c.Message = msg

	return &c
}

// WithDetails returns a copy carrying structured context.
func (e *Error) WithDetails(d map[string]any) *Error {
	c := *e
	c.Details = d

	return &c
}

func newErr(status int, code, msg string) *Error {
	return &Error{Code: code, Message: msg, Status: status}
}

// The complete catalog of domain errors. Branch with errors.Is.
var (
	// 401 — authentication.
	ErrInvalidCredentials    = newErr(http.StatusUnauthorized, "invalid_credentials", "Invalid credentials.")
	ErrUnauthorized          = newErr(http.StatusUnauthorized, "unauthorized", "Authentication required.")
	ErrInvalidToken          = newErr(http.StatusUnauthorized, "invalid_token", "The token is invalid.")
	ErrTokenExpired          = newErr(http.StatusUnauthorized, "token_expired", "The token has expired.")
	ErrTokenRevoked          = newErr(http.StatusUnauthorized, "token_revoked", "The token has been revoked.")
	ErrSessionExpired        = newErr(http.StatusUnauthorized, "session_expired", "The session has expired.")
	ErrSessionDeviceMismatch = newErr(
		http.StatusUnauthorized, "device_mismatch", "The session was used from a different device.",
	)
	ErrMFAInvalid       = newErr(http.StatusUnauthorized, "mfa_invalid", "Invalid MFA code.")
	ErrInvalidOTP       = newErr(http.StatusUnauthorized, "invalid_otp", "Invalid one-time code.")
	ErrChallengeInvalid = newErr(http.StatusUnauthorized, "challenge_invalid", "Invalid challenge.")

	// 403 — permitted-but-blocked / gated.
	ErrForbidden           = newErr(http.StatusForbidden, "forbidden", "Not permitted.")
	ErrAccountSuspended    = newErr(http.StatusForbidden, "account_suspended", "Account suspended.")
	ErrAccountBanned       = newErr(http.StatusForbidden, "account_banned", "Account banned.")
	ErrAccountLocked       = newErr(http.StatusForbidden, "account_locked", "Account locked.")
	ErrEmailNotVerified    = newErr(http.StatusForbidden, "email_not_verified", "Email not verified.")
	ErrPhoneNotVerified    = newErr(http.StatusForbidden, "phone_not_verified", "Phone not verified.")
	ErrMFARequired         = newErr(http.StatusForbidden, "mfa_required", "MFA is required.")
	ErrMFAFactorNotAllowed = newErr(
		http.StatusForbidden, "mfa_factor_not_allowed", "This MFA factor is not permitted by policy.",
	)
	ErrStepUpRequired     = newErr(http.StatusForbidden, "step_up_required", "Step-up authentication is required.")
	ErrRegistrationClosed = newErr(http.StatusForbidden, "registration_closed", "Registration is closed.")
	ErrInvitationRequired = newErr(http.StatusForbidden, "invitation_required", "An invitation is required.")
	ErrCaptchaRequired    = newErr(http.StatusForbidden, "captcha_required", "Captcha is required.")
	ErrCaptchaInvalid     = newErr(http.StatusForbidden, "captcha_invalid", "Invalid captcha.")
	ErrConsentRequired    = newErr(http.StatusForbidden, "consent_required", "Consent is required.")
	ErrInvalidCsrf        = newErr(http.StatusForbidden, "invalid_csrf", "Missing or invalid CSRF token.")

	// 404 — not found / not visible.
	ErrNotFound            = newErr(http.StatusNotFound, "not_found", "Not found.")
	ErrUserNotFound        = newErr(http.StatusNotFound, "user_not_found", "User not found.")
	ErrSessionNotFound     = newErr(http.StatusNotFound, "session_not_found", "Session not found.")
	ErrProjectNotFound     = newErr(http.StatusNotFound, "project_not_found", "Project not found.")
	ErrEnvironmentNotFound = newErr(http.StatusNotFound, "environment_not_found", "Environment not found.")
	ErrClientNotFound      = newErr(http.StatusNotFound, "client_not_found", "App client not found.")
	ErrConnectionNotFound  = newErr(http.StatusNotFound, "connection_not_found", "Connection not found.")
	ErrDomainNotFound      = newErr(http.StatusNotFound, "domain_not_found", "Domain not found.")

	// 409 — conflict.
	ErrConflict       = newErr(http.StatusConflict, "conflict", "Conflicting state.")
	ErrEmailExists    = newErr(http.StatusConflict, "email_exists", "Email already in use.")
	ErrPhoneExists    = newErr(http.StatusConflict, "phone_exists", "Phone already in use.")
	ErrIdentityExists = newErr(http.StatusConflict, "identity_exists", "Identity already linked.")
	ErrAlreadyLinked  = newErr(http.StatusConflict, "already_linked", "Already linked.")
	ErrDomainTaken    = newErr(http.StatusConflict, "domain_taken", "Domain already claimed.")

	// 410 — gone / single-use consumed.
	ErrChallengeExpired = newErr(http.StatusGone, "challenge_expired", "The challenge has expired.")
	ErrTokenUsed        = newErr(http.StatusGone, "token_already_used", "The token has already been used.")
	ErrFlowNotFound     = newErr(http.StatusGone, "flow_not_found", "Flow not found or expired.")
	ErrFlowExpired      = newErr(http.StatusGone, "flow_expired", "The flow has expired.")

	// 422 — validation.
	ErrValidation     = newErr(http.StatusUnprocessableEntity, "validation_failed", "Validation failed.")
	ErrWeakPassword   = newErr(http.StatusUnprocessableEntity, "weak_password", "The password is too weak.")
	ErrPasswordReused = newErr(http.StatusUnprocessableEntity, "password_reused", "The password was used before.")

	// 400 — malformed.
	ErrBadRequest       = newErr(http.StatusBadRequest, "bad_request", "Invalid request.")
	ErrUnsupportedGrant = newErr(http.StatusBadRequest, "unsupported_grant", "Unsupported grant type.")
	// RFC 6749 §4.1.2.1: when the client is unknown/disabled or the redirect_uri
	// is not one the client registered, the authorization server MUST NOT
	// redirect the user-agent — the target cannot be trusted — and reports the
	// error directly instead.
	// RFC 6749 §5.2 invalid_grant: the presented authorization grant is invalid,
	// expired, revoked, or does not match the request it was issued for — which
	// is also how a failed PKCE verification is reported.
	ErrInvalidGrant = newErr(http.StatusBadRequest, "invalid_grant", "The authorization grant is invalid.")
	// RFC 9126 §4 invalid_request_uri: the pushed request is unknown, expired or
	// already redeemed. It cannot be reported on a redirect, because without the
	// pushed request there is no validated redirect_uri to report it on.
	ErrInvalidRequestURI  = newErr(http.StatusBadRequest, "invalid_request_uri", "The request_uri is unknown or expired.")
	ErrInvalidClient      = newErr(http.StatusBadRequest, "invalid_client", "Unknown or disabled OAuth client.")
	ErrInvalidRedirectURI = newErr(http.StatusBadRequest, "invalid_redirect_uri",
		"The redirect_uri is not registered for this client.")

	// 429 — throttled.
	ErrRateLimited       = newErr(http.StatusTooManyRequests, "rate_limited", "Too many requests.")
	ErrFlowResendTooSoon = newErr(http.StatusTooManyRequests, "flow_resend_too_soon", "Please wait before resending.")

	// 501 — not implemented.
	ErrNotImplemented = newErr(http.StatusNotImplemented, "not_implemented", "Not implemented.")

	// 502/503/500 — downstream / server.
	ErrProviderError      = newErr(http.StatusBadGateway, "provider_error", "Upstream provider error.")
	ErrSSOError           = newErr(http.StatusBadGateway, "sso_error", "SSO error.")
	ErrSCIMError          = newErr(http.StatusBadGateway, "scim_error", "SCIM error.")
	ErrServiceUnavailable = newErr(http.StatusServiceUnavailable, "service_unavailable", "Service unavailable.")
	ErrInternal           = newErr(http.StatusInternalServerError, "internal_error", "Internal error.")
)
