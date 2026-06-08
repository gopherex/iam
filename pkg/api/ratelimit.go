package api

// In-memory sliding-window rate limiter middleware.
//
// WARNING: this implementation stores counters in-process (sync.Map under a
// mutex). It is correct for single-instance deployments but does NOT coordinate
// across multiple instances. For multi-instance (horizontal scaling) deployments,
// replace the three package-level limiters with a Redis-, Postgres-, or
// memcached-backed implementation sharing the same rateLimiter interface.

import (
	"net"
	"net/http"
	"strings"
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
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(r.RemoteAddr, "[]")
}

var (
	authLimiter      = newRateLimiter(30, time.Minute)
	sensitiveLimiter = newRateLimiter(10, time.Minute)
	guestLimiter     = newRateLimiter(5, time.Minute)
)

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter, message, ok := rateLimitForRequest(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		key := rateLimitKey(r)
		if !limiter.allow(key) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate_limit_exceeded","message":"`+message+`"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimitForRequest(r *http.Request) (*rateLimiter, string, bool) {
	if r.Method == http.MethodOptions {
		return nil, "", false
	}
	path := r.URL.Path
	if path == "/v1/auth/guest" {
		return guestLimiter, "Too many guest accounts created.", true
	}
	if isSensitiveRateLimitedPath(path) {
		return sensitiveLimiter, "Too many attempts. Try again later.", true
	}
	if isAuthRateLimitedPath(path) {
		return authLimiter, "Too many requests. Try again later.", true
	}
	return nil, "", false
}

func isSensitiveRateLimitedPath(path string) bool {
	switch path {
	case "/v1/auth/sign-in/password",
		"/v1/auth/password/forgot",
		"/v1/auth/password/reset",
		"/v1/auth/password/verify",
		"/v1/auth/email/verification/start",
		"/v1/auth/email/verification/verify",
		"/v1/auth/phone/verification/start",
		"/v1/auth/phone/verification/verify",
		"/v1/auth/otp/start",
		"/v1/auth/otp/verify",
		"/v1/auth/magic-link/start",
		"/v1/auth/magic-link/verify",
		"/v1/auth/mfa/challenge",
		"/v1/auth/mfa/verify",
		"/v1/auth/webauthn/login/options",
		"/v1/auth/webauthn/login/verify",
		"/v1/auth/webauthn/register/options",
		"/v1/auth/webauthn/register/verify",
		"/v1/challenges/captcha/verify":
		return true
	default:
		return false
	}
}

func isAuthRateLimitedPath(path string) bool {
	switch path {
	case "/v1/auth/sign-up",
		"/v1/auth/token/refresh",
		"/v1/auth/token/exchange",
		"/v1/auth/oauth/exchange",
		"/v1/auth/access-requests":
		return true
	default:
		return false
	}
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
