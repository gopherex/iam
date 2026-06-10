package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/gopherex/iam/internal/domain"
)

// DeviceFingerprintHeader is an optional client-supplied stable device id; when
// present it is bound to the session for self-managed-session UIs.
const DeviceFingerprintHeader = "X-Device-Fingerprint"

// RequestMetaMiddleware captures the originating device/network context (client
// IP, User-Agent, optional fingerprint) into the request context so the
// session-minting path can record it on the session. Place it early in the
// pipeline (it only reads the request).
func RequestMetaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := domain.RequestMeta{
			IP:          clientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
			Fingerprint: r.Header.Get(DeviceFingerprintHeader),
		}
		next.ServeHTTP(w, r.WithContext(domain.WithRequestMeta(r.Context(), meta)))
	})
}

// clientIP resolves the caller IP, honoring the first hop of X-Forwarded-For and
// then X-Real-IP, falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.Trim(r.RemoteAddr, "[]")
}
