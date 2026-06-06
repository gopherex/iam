package api

import "net/http"

// CookieAuthMiddleware lets cookie-mode browser clients authenticate without an
// Authorization header: when a request has no Authorization header but carries
// the session cookie (api.SessionCookieName), the cookie value is promoted to a
// `Bearer` Authorization header so the generated bearerAuth security handler
// validates it transparently.
//
// It MUST run INSIDE CSRFMiddleware (which keys off the cookie + the *absence*
// of an Authorization header): CSRF evaluates the original request first, then
// this middleware adds the header for the auth layer.
func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			if ck, err := r.Cookie(SessionCookieName); err == nil && ck.Value != "" {
				r.Header.Set("Authorization", "Bearer "+ck.Value)
			}
		}
		next.ServeHTTP(w, r)
	})
}
