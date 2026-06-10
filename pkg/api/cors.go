package api

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OriginSource supplies the per-tenant CORS allow-list: the union of every app
// client's allowed_origins. CORS preflight (OPTIONS) carries no X-Client-Id, so
// the decision can only be made against this global union; tenant isolation is
// enforced separately (X-Client-Id + tokens). It is consulted in addition to the
// statically configured origins.
type OriginSource interface {
	AllowedOrigins(ctx context.Context) ([]string, error)
}

// originCache memoizes the dynamic origin union with a short TTL so CORS does not
// hit the database on every request. On a refresh error the last good set is
// kept (fail-open to the previously-allowed origins, never widening).
type originCache struct {
	src OriginSource
	ttl time.Duration
	mu  sync.RWMutex
	set map[string]struct{}
	exp time.Time
}

func (c *originCache) allowed(origin string) bool {
	if c == nil || c.src == nil {
		return false
	}
	c.mu.RLock()
	fresh := time.Now().Before(c.exp)
	if fresh {
		_, ok := c.set[origin]
		c.mu.RUnlock()
		return ok
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.exp) { // another goroutine refreshed
		_, ok := c.set[origin]
		return ok
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	origins, err := c.src.AllowedOrigins(ctx)
	if err != nil {
		// Keep the stale set; extend exp briefly to avoid hammering the DB.
		c.exp = time.Now().Add(5 * time.Second)
		_, ok := c.set[origin]
		return ok
	}
	set := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		set[o] = struct{}{}
	}
	c.set = set
	c.exp = time.Now().Add(c.ttl)
	_, ok := set[origin]
	return ok
}

// CORSMiddleware applies the configured browser cross-origin policy to runtime
// endpoints and handles preflight requests before they reach the generated
// router. An origin is reflected with credentials when it is in the static
// allow-list OR the dynamic per-client union (source). Wildcard ("*") in the
// static list means "allow any origin WITHOUT credentials" (no
// Access-Control-Allow-Credentials), preventing credential theft.
func CORSMiddleware(allowedOrigins []string, source OriginSource, ttl time.Duration) func(http.Handler) http.Handler {
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
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	cache := &originCache{src: source, ttl: ttl}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			if allowAny {
				setCORSHeadersPublic(w, origin)
			} else if _, ok := allowed[origin]; ok || cache.allowed(origin) {
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
