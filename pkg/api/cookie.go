package api

import (
	"net/http"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

// cookieRefreshFallbackTTL is used only when a minted session does not carry a
// refresh lifetime (an adapter that predates RefreshExpiresIn). The real value
// comes from the project's session_policy, via Session.RefreshExpiresIn — a
// cookie that outlives or undercuts the token it carries is a session that ends
// at a time nobody configured.
const cookieRefreshFallbackTTL = 30 * 24 * time.Hour

// SessionCookiesFor renders the cookie pair for a minted session, taking both
// lifetimes from the session itself (i.e. from the project's session_policy).
func SessionCookiesFor(sess *domain.Session) []string {
	access := time.Duration(sess.ExpiresIn) * time.Second

	refresh := time.Duration(sess.RefreshExpiresIn) * time.Second
	if refresh <= 0 {
		refresh = cookieRefreshFallbackTTL
	}

	return SessionCookies(sess.AccessToken, sess.RefreshToken, access, refresh)
}

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
