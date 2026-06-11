package api

// In-memory sliding-window rate limiter middleware.
//
// WARNING: this implementation stores counters in-process (sync.Map under a
// mutex). It is correct for single-instance deployments but does NOT coordinate
// across multiple instances. For multi-instance (horizontal scaling) deployments,
// replace the three package-level limiters with a Redis-, Postgres-, or
// memcached-backed implementation sharing the same rateLimiter interface.

import (
	"context"
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
	stop    chan struct{}
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	// Defense-in-depth: a non-positive window would panic time.NewTicker and is
	// otherwise meaningless. Write-time validation already enforces window>=1s,
	// but the reader path is an exported boundary — clamp rather than trust it.
	if window <= 0 {
		window = time.Minute
	}
	if limit < 1 {
		limit = 1
	}
	rl := &rateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
		stop:    make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Stop terminates the cleanup goroutine. Safe to call once; idempotent only via
// the caller (projectLimiters) replacing the instance, which never re-Stops.
func (rl *rateLimiter) Stop() { close(rl.stop) }

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
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

// rateLimitKey derives the per-caller counter key from clientIP, which honors
// forwarding headers ONLY behind a configured trusted proxy (requestmeta.go).
// With no trusted proxies set it is the real TCP peer, so a client cannot spoof
// X-Forwarded-For to escape its IP bucket.
func rateLimitKey(r *http.Request) string {
	return clientIP(r)
}

var (
	authLimiter      = newRateLimiter(30, time.Minute)
	sensitiveLimiter = newRateLimiter(10, time.Minute)
	guestLimiter     = newRateLimiter(5, time.Minute)
)

// RateLimitConfigReader yields a project's effective rate-limit rules for the
// request environment. clientID is the X-Client-ID (the project id); env is the
// raw X-Environment header ("" => the persistence default "live"). It returns
// (nil, nil) when the project has no rate_limits doc (the caller then falls back
// to the hardcoded defaults). The reader runs before the env/meta middlewares,
// so identity is passed as explicit strings, not via ctx.
type RateLimitConfigReader interface {
	RateLimitRules(ctx context.Context, clientID, env string) ([]RateLimitRule, error)
}

// RateLimitRule is a runtime-resolved override (a subset of
// domain.RateLimitRuleSpec, already validated on write). Endpoint matches
// r.URL.Path exactly; By is always "ip" today.
type RateLimitRule struct {
	Endpoint string
	Limit    int
	Window   time.Duration
	By       string
}

// RateLimitMiddleware enforces the built-in hardcoded limits only (no per-project
// overrides). Kept for back-compat with existing callers/tests; equivalent to
// NewRateLimitMiddleware(nil).
func RateLimitMiddleware(next http.Handler) http.Handler {
	return NewRateLimitMiddleware(nil)(next)
}

// NewRateLimitMiddleware builds the rate-limit middleware backed by an optional
// per-project config reader. When reader is nil (or returns no rule for the
// classified endpoint) the hardcoded defaults apply, preserving current
// behavior. Per-project rules override only limit/window of the existing
// IP-keyed, path-classified buckets, merged per-endpoint over the defaults.
func NewRateLimitMiddleware(reader RateLimitConfigReader) func(http.Handler) http.Handler {
	var pl *projectLimiters
	if reader != nil {
		pl = newProjectLimiters(reader, 30*time.Second)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter, message, ok := rateLimitForRequest(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			key := rateLimitKey(r)

			// Per-project override (limit/window) for this endpoint, if any.
			// Falls open to the hardcoded limiter on miss or reader error.
			if pl != nil {
				clientID := r.Header.Get("X-Client-ID")
				env := r.Header.Get(EnvironmentHeader)
				if pLimiter, pKey, found := pl.limiterFor(r.Context(), clientID, env, r.URL.Path, key); found {
					limiter, key = pLimiter, pKey
				}
			}

			if !limiter.allow(key) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"rate_limit_exceeded","message":"`+message+`"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// projectLimiters caches per-(project,env) resolved rules (short TTL) and the
// per-(project,env,endpoint) limiter instances they spawn, so the reader (DB) is
// consulted at most once per project/env per TTL window. Mirrors the CORS
// originCache fail-open behavior: on a reader error the last-good rule set is
// kept and refresh is deferred briefly.
type projectLimiters struct {
	reader RateLimitConfigReader
	ttl    time.Duration

	mu       sync.Mutex
	rules    map[string]ruleCacheEntry // key: projectID|env
	limiters map[string]*rateLimiter   // key: projectID|env|endpoint
}

type ruleCacheEntry struct {
	byEndpoint map[string]RateLimitRule // endpoint -> override
	exp        time.Time
}

func newProjectLimiters(reader RateLimitConfigReader, ttl time.Duration) *projectLimiters {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &projectLimiters{
		reader:   reader,
		ttl:      ttl,
		rules:    make(map[string]ruleCacheEntry),
		limiters: make(map[string]*rateLimiter),
	}
}

// limiterFor returns the per-project limiter and namespaced key for path, or
// (nil,"",false) when no project override applies (caller uses the default).
func (p *projectLimiters) limiterFor(ctx context.Context, clientID, env, path, ipKey string) (*rateLimiter, string, bool) {
	if clientID == "" {
		return nil, "", false
	}
	if env == "" {
		env = "live"
	}
	scope := clientID + "|" + env

	entry := p.resolve(ctx, scope, clientID, env)
	rule, ok := entry.byEndpoint[path]
	if !ok {
		return nil, "", false
	}

	limiter := p.getOrCreateLimiter(scope+"|"+path, rule.Limit, rule.Window)
	// Namespace the counter so two projects/environments never collide.
	return limiter, scope + "|" + path + "|" + ipKey, true
}

// resolve returns the cached rule set for scope, refreshing from the reader on
// miss/expiry. On a reader error it keeps the last-good set (fail-open) and
// briefly defers the next refresh.
func (p *projectLimiters) resolve(ctx context.Context, scope, clientID, env string) ruleCacheEntry {
	p.mu.Lock()
	defer p.mu.Unlock()

	if e, ok := p.rules[scope]; ok && time.Now().Before(e.exp) {
		return e
	}

	rules, err := p.reader.RateLimitRules(ctx, clientID, env)
	if err != nil {
		if e, ok := p.rules[scope]; ok {
			e.exp = time.Now().Add(5 * time.Second) // keep stale, back off
			p.rules[scope] = e
			return e
		}
		e := ruleCacheEntry{byEndpoint: map[string]RateLimitRule{}, exp: time.Now().Add(5 * time.Second)}
		p.rules[scope] = e
		return e
	}

	byEndpoint := make(map[string]RateLimitRule, len(rules))
	for _, r := range rules {
		byEndpoint[r.Endpoint] = r
	}
	e := ruleCacheEntry{byEndpoint: byEndpoint, exp: time.Now().Add(p.ttl)}
	p.rules[scope] = e
	return e
}

// getOrCreateLimiter returns the limiter for key, (re)creating it only when its
// limit/window differs from the cached instance — so a stable rule reuses the
// same limiter (and its single cleanup goroutine) rather than leaking one per
// request. Called under p.mu via resolve's lock is NOT held here; guard locally.
func (p *projectLimiters) getOrCreateLimiter(key string, limit int, window time.Duration) *rateLimiter {
	p.mu.Lock()
	defer p.mu.Unlock()
	if rl, ok := p.limiters[key]; ok && rl.limit == limit && rl.window == window {
		return rl
	}
	// Replacing a limiter whose rule changed: stop the old one's cleanup
	// goroutine so it doesn't leak for the process lifetime.
	if old, ok := p.limiters[key]; ok {
		old.Stop()
	}
	rl := newRateLimiter(limit, window)
	p.limiters[key] = rl
	return rl
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
