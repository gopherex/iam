package sdk

// Claim guards enforce constraints that IAM itself minted onto a token: OAuth2
// scope grants and authenticator-strength signals (AAL/AMR). They are NOT
// relationship-based authorization (ReBAC) — they never ask "can subject S act
// on resource R"; they only assert that the presented token already carries the
// required claim. Relationship checks belong to the separate AuthZ product and
// its own SDK.
//
// Guards run AFTER an authentication middleware has placed a Principal in the
// request context (Verifier/LocalVerifier/HybridVerifier .Middleware). A guard
// with no Principal in context fails closed with 401; a Principal that lacks the
// required claim fails with 403.

import (
	"errors"
	"net/http"
	"slices"
)

// ErrForbidden is returned when an authenticated principal lacks a required
// claim (scope or assurance level).
var ErrForbidden = errors.New("iam sdk: forbidden")

// HasScope reports whether the principal was granted scope.
func (p *Principal) HasScope(scope string) bool {
	if p == nil || scope == "" {
		return false
	}
	return slices.Contains(p.Scopes, scope)
}

// HasAnyScope reports whether the principal holds at least one of scopes. With
// no scopes it returns true (no constraint).
func (p *Principal) HasAnyScope(scopes ...string) bool {
	if len(scopes) == 0 {
		return true
	}
	return slices.ContainsFunc(scopes, p.HasScope)
}

// HasAllScopes reports whether the principal holds every one of scopes.
func (p *Principal) HasAllScopes(scopes ...string) bool {
	for _, s := range scopes {
		if !p.HasScope(s) {
			return false
		}
	}
	return true
}

// HasAMR reports whether the principal authenticated with the given method
// (e.g. "pwd", "otp", "webauthn").
func (p *Principal) HasAMR(method string) bool {
	if p == nil || method == "" {
		return false
	}
	return slices.Contains(p.AMR, method)
}

// MeetsAAL reports whether the principal's authenticator assurance level is at
// least min (e.g. MeetsAAL(2) requires a step-up/MFA session).
func (p *Principal) MeetsAAL(min int) bool {
	return p != nil && p.AAL >= min
}

// GuardOption customizes a claim guard's failure handlers.
type GuardOption func(*guardConfig)

type guardConfig struct {
	onUnauthenticated HTTPErrorHandler
	onForbidden       HTTPErrorHandler
}

// WithUnauthenticatedHandler overrides the response when no Principal is present
// in the request context (defaults to 401).
func WithUnauthenticatedHandler(h HTTPErrorHandler) GuardOption {
	return func(c *guardConfig) {
		if h != nil {
			c.onUnauthenticated = h
		}
	}
}

// WithForbiddenHandler overrides the response when the Principal lacks a
// required claim (defaults to 403).
func WithForbiddenHandler(h HTTPErrorHandler) GuardOption {
	return func(c *guardConfig) {
		if h != nil {
			c.onForbidden = h
		}
	}
}

func newGuardConfig(opts []GuardOption) guardConfig {
	cfg := guardConfig{
		onUnauthenticated: defaultHTTPErrorHandler,
		onForbidden:       defaultForbiddenHandler,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// guard builds an HTTP middleware that admits a request only when ok(principal)
// holds. It must be chained after an authentication middleware.
func guard(ok func(*Principal) bool, opts ...GuardOption) func(http.Handler) http.Handler {
	cfg := newGuardConfig(opts)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, present := PrincipalFrom(r.Context())
			if !present {
				cfg.onUnauthenticated(w, r, ErrMissingToken)
				return
			}
			if !ok(principal) {
				cfg.onForbidden(w, r, ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireScopes admits a request only when the principal holds every listed
// scope. Chain after an authentication middleware.
func RequireScopes(scopes []string, opts ...GuardOption) func(http.Handler) http.Handler {
	return guard(func(p *Principal) bool { return p.HasAllScopes(scopes...) }, opts...)
}

// RequireAnyScope admits a request when the principal holds at least one of the
// listed scopes. Chain after an authentication middleware.
func RequireAnyScope(scopes []string, opts ...GuardOption) func(http.Handler) http.Handler {
	return guard(func(p *Principal) bool { return p.HasAnyScope(scopes...) }, opts...)
}

// RequireAAL admits a request only when the principal's assurance level is at
// least min (e.g. RequireAAL(2) demands a step-up/MFA session). Chain after an
// authentication middleware.
func RequireAAL(min int, opts ...GuardOption) func(http.Handler) http.Handler {
	return guard(func(p *Principal) bool { return p.MeetsAAL(min) }, opts...)
}

func defaultForbiddenHandler(w http.ResponseWriter, _ *http.Request, _ error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"forbidden"}`))
}
