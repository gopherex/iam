package api

// In-memory sliding-window rate limiter middleware.
//
// WARNING: this implementation stores counters in-process (sync.Map under a
// mutex). It is correct for single-instance deployments but does NOT coordinate
// across multiple instances. For multi-instance (horizontal scaling) deployments,
// replace the three package-level limiters with a Redis-, Postgres-, or
// memcached-backed implementation sharing the same rateLimiter interface.

import (
	"net/http"
	"sync"
	"time"
)

type rateLimitEntry struct {
	count  int
	expiry time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, v := range rl.entries {
			if now.After(v.expiry) {
				delete(rl.entries, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	e, ok := rl.entries[key]
	if !ok || now.After(e.expiry) {
		rl.entries[key] = &rateLimitEntry{count: 1, expiry: now.Add(rl.window)}
		return true
	}
	e.count++
	return e.count <= rl.limit
}

func rateLimitKey(r *http.Request) string {
	return r.RemoteAddr
}

var (
	authLimiter      = newRateLimiter(10, time.Minute)
	sensitiveLimiter = newRateLimiter(5, time.Minute)
	guestLimiter     = newRateLimiter(5, time.Minute)
)

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rateLimitKey(r)
		if !authLimiter.allow(key) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate_limit_exceeded","message":"Too many requests. Try again later."}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func SensitiveRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rateLimitKey(r)
		if !sensitiveLimiter.allow(key) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate_limit_exceeded","message":"Too many attempts. Try again later."}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GuestRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rateLimitKey(r)
		if !guestLimiter.allow(key) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate_limit_exceeded","message":"Too many guest accounts created."}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
