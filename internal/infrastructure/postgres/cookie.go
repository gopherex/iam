package postgres

// Cookie-mode session cookies. Browser (redirect) auth flows hand the minted
// access token back as an HttpOnly session cookie instead of (or alongside) the
// token body; api.CookieAuthMiddleware reads it back on subsequent requests and
// api.CSRFMiddleware enforces CSRF while it is present. The cookie name is the
// shared api.SessionCookieName so all three pieces agree.

import (
	"net/http"
	"time"

	"github.com/gopherex/iam/pkg/api"
)

// sessionCookieHeader renders the Set-Cookie header value for a session cookie
// carrying token, valid for ttl. HttpOnly + Secure + SameSite=Lax: not readable
// by JS, HTTPS-only, sent on top-level navigations (CSRF middleware still guards
// state-changing requests).
func sessionCookieHeader(token string, ttl time.Duration) string {
	c := &http.Cookie{
		Name:     api.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl / time.Second),
	}
	return c.String()
}
