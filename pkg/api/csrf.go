package api

import (
	"context"
	"net/http"

	"github.com/gopherex/iam/internal/domain"
)

// SessionCookieName is the cookie that carries a cookie-mode browser session.
// Cookie-minting flows MUST use this name; the CSRF middleware keys off its
// presence to decide whether a request is cookie-authenticated.
const SessionCookieName = "iam_session"

// RefreshCookieName carries the refresh token in cookie mode so a cookie session
// can be refreshed past the access token's TTL (see PostV1AuthTokenRefresh).
const RefreshCookieName = "iam_refresh"

// csrfVerifier is the subset of PlatformCsrf the middleware needs.
type csrfVerifier interface {
	VerifyCsrfToken(ctx context.Context, clientID, token string) error
}

// CSRFMiddleware enforces CSRF protection on cookie-mode requests using the
// synchronizer-token pattern. A request is challenged only when it is BOTH a
// state-changing method AND cookie-authenticated:
//
//   - safe methods (GET/HEAD/OPTIONS/TRACE) always pass — they must not mutate;
//   - requests carrying an Authorization header pass — bearer/API-key/Basic
//     callers are immune to CSRF (the credential is not ambiently attached);
//   - requests without the session cookie pass — they are not cookie-mode.
//
// A challenged request must present a valid X-CSRF-Token (issued via /v1/csrf)
// together with the X-Client-ID it was bound to; otherwise it is rejected with
// 403 invalid_csrf in the standard ErrorEnvelope.
func CSRFMiddleware(v csrfVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if csrfSafeMethod(r.Method) ||
				r.Header.Get("Authorization") != "" ||
				!hasSessionCookie(r) {
				next.ServeHTTP(w, r)
				return
			}
			clientID := r.Header.Get("X-Client-ID")
			token := r.Header.Get("X-CSRF-Token")
			if clientID == "" || token == "" {
				writeEnvelope(w, domain.ErrInvalidCsrf)
				return
			}
			if err := v.VerifyCsrfToken(r.Context(), clientID, token); err != nil {
				de := domain.ErrInvalidCsrf
				if e, ok := err.(*domain.Error); ok {
					de = e
				}
				writeEnvelope(w, de)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func csrfSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

func hasSessionCookie(r *http.Request) bool {
	_, err := r.Cookie(SessionCookieName)
	return err == nil
}
