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

// FlowCookieName carries the resumable-auth flow_token in cookie mode so the
// token is never exposed to JS (GET /v1/auth/flows/current reads it). Scoped to
// the flows path so it is only presented to flow endpoints.
const FlowCookieName = "iam_flow"

const cookieFlowPath = "/v1/auth/flows"

// cookieFlowTTL matches the server-side flow TTL (engine flowTTL).
const cookieFlowTTL = 30 * time.Minute

// FlowCookieSet renders the Set-Cookie header that stores the flow_token while a
// flow is pending. ttl should match the server-side flow TTL.
func FlowCookieSet(token string, ttl time.Duration) []string {
	return []string{cookieHeader(FlowCookieName, token, cookieFlowPath, ttl)}
}

// FlowCookieClear renders the Set-Cookie header that deletes the flow cookie
// (flow completed or abandoned).
func FlowCookieClear() []string {
	c := &http.Cookie{
		Name:     FlowCookieName,
		Value:    "",
		Path:     cookieFlowPath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	return []string{c.String()}
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
