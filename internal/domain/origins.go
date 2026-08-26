package domain

import (
	"net/url"
	"strings"
)

// maxAllowedOrigins caps how many origins a single client may register.
const maxAllowedOrigins = 1000

// schemeHTTP is the plain (non-TLS) URL scheme, checked in several places
// across this package that validate a URL is http(s).
const schemeHTTP = "http"

// NormalizeOrigin validates and canonicalises a browser origin
// (scheme://host[:port]). It returns "" for anything that must NOT be allowed
// as a CORS origin: empty, "*", "null", non-http(s) schemes, missing host, any
// path/query/fragment/userinfo, or plain http on a non-loopback host. The result
// is lowercase scheme + host so comparisons are exact.
//
// Rejecting "*"/"null" prevents credentialed wildcard and sandboxed-iframe
// (Origin: null) bypasses; requiring https off-localhost avoids registering
// insecure origins for credentialed CORS.
func NormalizeOrigin(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || s == "*" || strings.EqualFold(s, "null") {
		return ""
	}

	parsedURL, err := url.Parse(s)
	if err != nil || parsedURL.Host == "" {
		return ""
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != schemeHTTP && scheme != "https" {
		return ""
	}

	if hasOriginOnlyDisallowedParts(parsedURL) {
		return ""
	}

	if scheme == schemeHTTP && !isLoopbackHost(strings.ToLower(parsedURL.Hostname())) {
		return ""
	}

	return scheme + "://" + strings.ToLower(parsedURL.Host)
}

// hasOriginOnlyDisallowedParts reports whether u carries anything beyond
// scheme://host[:port] — a path, query, fragment, or userinfo — none of which
// belong in a CORS origin.
func hasOriginOnlyDisallowedParts(u *url.URL) bool {
	return (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil
}

// isLoopbackHost reports whether host (already lowercased) is one of the
// loopback names plain http is allowed on.
func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// NormalizeOrigins validates, canonicalises, and de-duplicates a list of origins,
// dropping invalid entries and capping the count.
func NormalizeOrigins(in []string) []string {
	seen := make(map[string]struct{}, len(in))

	out := make([]string, 0, len(in))
	for _, o := range in {
		n := NormalizeOrigin(o)
		if n == "" {
			continue
		}

		if _, ok := seen[n]; ok {
			continue
		}

		seen[n] = struct{}{}

		out = append(out, n)
		if len(out) >= maxAllowedOrigins {
			break
		}
	}

	return out
}
