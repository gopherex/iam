package postgres

// Cookie-mode session cookies. Browser (redirect) auth flows hand the minted
// access + refresh tokens back as HttpOnly cookies; the actual Set-Cookie format
// lives in pkg/api (the transport layer owns it) and is shared with the refresh
// handler. api.CookieAuthMiddleware reads the access cookie back and
// api.CSRFMiddleware enforces CSRF while it is present.

import (
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

// sessionCookies renders the access + refresh Set-Cookie pair minted on a cookie
// -mode login (delegates to the shared pkg/api builder).
func sessionCookies(access, refresh string, accessTTL, refreshTTL time.Duration) []string {
	return api.SessionCookies(access, refresh, accessTTL, refreshTTL)
}

// sessionCookiesFor renders the pair for a minted session, taking both lifetimes
// from the session — i.e. from the project's session_policy — so a browser login
// does not silently get a different lifetime than the policy configures.
func sessionCookiesFor(sess *domain.Session) []string {
	return api.SessionCookiesFor(sess)
}
