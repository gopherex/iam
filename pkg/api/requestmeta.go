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

// trustedProxyNets is the set of reverse-proxy/LB networks whose forwarding
// headers (X-Forwarded-For / X-Real-IP) we trust. It is set once at startup via
// SetTrustedProxies and read-only thereafter. Empty => trust no proxy headers.
var trustedProxyNets []*net.IPNet

// SetTrustedProxies configures the trusted reverse-proxy CIDRs (or bare IPs).
// Call once during startup, before serving. Unparseable entries are ignored.
// When empty, clientIP returns the real TCP peer and never honors forwarding
// headers — this prevents a client from spoofing its IP (e.g. to bypass
// IP-keyed rate limits).
func SetTrustedProxies(cidrs []string) {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.ContainsRune(c, '/') {
			if strings.Contains(c, ":") {
				c += "/128"
			} else {
				c += "/32"
			}
		}
		if _, n, err := net.ParseCIDR(c); err == nil {
			nets = append(nets, n)
		}
	}
	trustedProxyNets = nets
}

func ipInTrustedProxies(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	for _, n := range trustedProxyNets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

// remoteHost returns the host portion of RemoteAddr (the real TCP peer).
func remoteHost(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.Trim(r.RemoteAddr, "[]")
}

// clientIP resolves the caller IP. Forwarding headers are honored ONLY when the
// connecting peer is a trusted proxy; then the rightmost X-Forwarded-For entry
// that is NOT itself a trusted proxy is the real client (left entries are
// client-controlled and spoofable). With no trusted proxies configured, or a
// peer outside them, the real TCP peer is returned.
func clientIP(r *http.Request) string {
	peer := remoteHost(r)
	if len(trustedProxyNets) == 0 || !ipInTrustedProxies(peer) {
		return peer
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip != "" && !ipInTrustedProxies(ip) {
				return ip
			}
		}
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return xr
	}
	return peer
}
