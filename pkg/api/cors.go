package api

import (
	"net/http"
	"strings"
)

// CORSMiddleware applies the configured browser cross-origin policy to runtime
// endpoints and handles preflight requests before they reach the generated
// router. Only explicitly listed origins are reflected; wildcard ("*") is
// treated as "allow any origin WITHOUT credentials" (no Access-Control-Allow-
// Credentials header), preventing credential theft via cross-origin requests.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAny := false
	allowed := map[string]struct{}{}
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAny = true
			continue
		}
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			if allowAny {
				setCORSHeadersPublic(w, origin)
			} else if _, ok := allowed[origin]; ok {
				setCORSHeaders(w, origin)
			} else {
				next.ServeHTTP(w, r)
				return
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setCORSHeaders(w http.ResponseWriter, origin string) {
	h := w.Header()
	h.Add("Vary", "Origin")
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Client-ID,X-CSRF-Token,X-Environment,Idempotency-Key")
	h.Set("Access-Control-Expose-Headers", "Set-Cookie,RateLimit-Limit,RateLimit-Remaining,RateLimit-Reset")
	h.Set("Access-Control-Max-Age", "600")
}

func setCORSHeadersPublic(w http.ResponseWriter, origin string) {
	h := w.Header()
	h.Add("Vary", "Origin")
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Client-ID,X-CSRF-Token,X-Environment,Idempotency-Key")
	h.Set("Access-Control-Max-Age", "600")
}
