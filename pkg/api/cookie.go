package api

import (
	"net/http"
	"time"
)

// Default cookie lifetimes for the refresh handler (match the adapters' access /
// refresh token TTLs).
const (
	cookieAccessTTL  = time.Hour
	cookieRefreshTTL = 30 * 24 * time.Hour
)

// SessionCookies renders the access + refresh Set-Cookie header pair for a
// cookie-mode session. The access cookie (SessionCookieName) is sent on every
// path; the refresh cookie (RefreshCookieName) is scoped to the refresh endpoint
// so it is only presented there. Both are HttpOnly + Secure + SameSite=Lax.
func SessionCookies(access, refresh string, accessTTL, refreshTTL time.Duration) []string {
	return []string{
		cookieHeader(SessionCookieName, access, "/", accessTTL),
		cookieHeader(RefreshCookieName, refresh, "/v1/auth/token/refresh", refreshTTL),
	}
}

func cookieHeader(name, value, path string, ttl time.Duration) string {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl / time.Second),
	}
	return c.String()
}
