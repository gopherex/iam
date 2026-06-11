package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRateLimitReader is a test RateLimitConfigReader. It returns the rules for
// the given (clientID,env) and counts calls; an empty rules result mimics a
// project with no doc, and errFor forces a reader error for a scope.
type fakeRateLimitReader struct {
	calls  int64
	byKey  map[string][]RateLimitRule // key: clientID|env
	errFor map[string]error           // key: clientID|env
}

func (f *fakeRateLimitReader) RateLimitRules(_ context.Context, clientID, env string) ([]RateLimitRule, error) {
	atomic.AddInt64(&f.calls, 1)
	if env == "" {
		env = "live"
	}
	k := clientID + "|" + env
	if f.errFor != nil {
		if err, ok := f.errFor[k]; ok {
			return nil, err
		}
	}
	return f.byKey[k], nil
}

func doReq(t *testing.T, h http.Handler, headers map[string]string) int {
	t.Helper()
	req := httptest.NewRequest("POST", "/v1/auth/sign-in/password", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestRateLimiterAllowsUpToLimit(t *testing.T) {
	rl := newRateLimiter(3, time.Second)

	for i := 0; i < 3; i++ {
		if !rl.allow("key1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if rl.allow("key1") {
		t.Fatal("4th request should be rejected")
	}
}

func TestRateLimiterDifferentKeys(t *testing.T) {
	rl := newRateLimiter(1, time.Second)

	if !rl.allow("key1") {
		t.Fatal("key1 first request should be allowed")
	}
	if !rl.allow("key2") {
		t.Fatal("key2 first request should be allowed (independent counter)")
	}
	if rl.allow("key1") {
		t.Fatal("key1 second request should be rejected")
	}
}

func TestRateLimiterWindowReset(t *testing.T) {
	rl := newRateLimiter(1, 50*time.Millisecond)

	if !rl.allow("key1") {
		t.Fatal("first request should be allowed")
	}
	if rl.allow("key1") {
		t.Fatal("second request should be rejected")
	}
	time.Sleep(60 * time.Millisecond)
	if !rl.allow("key1") {
		t.Fatal("request after window expiry should be allowed")
	}
}

func TestRateLimitMiddlewareRejects(t *testing.T) {
	old := sensitiveLimiter
	sensitiveLimiter = newRateLimiter(2, time.Minute)
	t.Cleanup(func() { sensitiveLimiter = old })

	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called++ })
	handler := RateLimitMiddleware(next)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/v1/auth/sign-in/password", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest("POST", "/v1/auth/sign-in/password", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request: expected 429, got %d", rec.Code)
	}
	if called != 2 {
		t.Fatalf("next handler called %d times, want 2", called)
	}
}

func TestRateLimitMiddlewareSkipsNonAuthPaths(t *testing.T) {
	called := 0
	handler := RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called++ }))

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/v1/projects/proj/admin/users", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
	if called != 20 {
		t.Fatalf("next handler called %d times, want 20", called)
	}
}

func TestRateLimitKeyIgnoresRemotePort(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	if got := rateLimitKey(req); got != "1.2.3.4" {
		t.Fatalf("key = %q", got)
	}

	req.RemoteAddr = "[2001:db8::1]:5678"
	if got := rateLimitKey(req); got != "2001:db8::1" {
		t.Fatalf("ipv6 key = %q", got)
	}
}

func TestRateLimitKeyIgnoresXForwardedForWithoutTrustedProxy(t *testing.T) {
	SetTrustedProxies(nil)
	t.Cleanup(func() { SetTrustedProxies(nil) })
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	// A spoofed XFF must NOT change the bucket when no proxy is trusted.
	req.Header.Set("X-Forwarded-For", "9.9.9.9, 10.0.0.1")
	if got := rateLimitKey(req); got != "10.0.0.1" {
		t.Fatalf("key = %q, want real peer (XFF ignored)", got)
	}
}

func TestRateLimitKeyHonorsXForwardedForBehindTrustedProxy(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8"})
	t.Cleanup(func() { SetTrustedProxies(nil) })

	// Peer is the trusted proxy: take the rightmost non-trusted XFF entry.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	req.Header.Set("X-Forwarded-For", "9.9.9.9, 10.0.0.2")
	if got := rateLimitKey(req); got != "9.9.9.9" {
		t.Fatalf("key = %q, want real client behind proxy", got)
	}

	// A client spoofing extra left hops can't escape its real IP: the rightmost
	// non-trusted entry is still the spoofer's real forwarded IP.
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 8.8.8.8, 10.0.0.2")
	if got := rateLimitKey(req); got != "8.8.8.8" {
		t.Fatalf("key = %q, want rightmost non-trusted hop", got)
	}

	// Peer NOT in the trusted set: ignore XFF, use the real peer.
	req.RemoteAddr = "203.0.113.5:5678"
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	if got := rateLimitKey(req); got != "203.0.113.5" {
		t.Fatalf("key = %q, want real peer (untrusted source)", got)
	}
}

func TestRateLimitMiddlewarePerProjectOverride(t *testing.T) {
	reader := &fakeRateLimitReader{byKey: map[string][]RateLimitRule{
		"P|live": {{Endpoint: "/v1/auth/sign-in/password", Limit: 2, Window: time.Minute, By: "ip"}},
	}}
	called := 0
	h := NewRateLimitMiddleware(reader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called++ }))

	// Project P: override limit 2 -> 3rd request blocked.
	for i := 0; i < 2; i++ {
		if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusOK {
			t.Fatalf("P request %d: got %d", i+1, code)
		}
	}
	if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusTooManyRequests {
		t.Fatalf("P 3rd request: got %d, want 429", code)
	}
}

func TestRateLimitMiddlewareFallsBackToDefaultWhenNoProjectRule(t *testing.T) {
	old := sensitiveLimiter
	sensitiveLimiter = newRateLimiter(2, time.Minute)
	t.Cleanup(func() { sensitiveLimiter = old })

	reader := &fakeRateLimitReader{byKey: map[string][]RateLimitRule{}} // no rules
	h := NewRateLimitMiddleware(reader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for i := 0; i < 2; i++ {
		if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusOK {
			t.Fatalf("request %d: got %d", i+1, code)
		}
	}
	if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusTooManyRequests {
		t.Fatalf("default 3rd request: got %d, want 429", code)
	}
}

func TestRateLimitMiddlewarePerProjectIsolation(t *testing.T) {
	reader := &fakeRateLimitReader{byKey: map[string][]RateLimitRule{
		"P1|live": {{Endpoint: "/v1/auth/sign-in/password", Limit: 1, Window: time.Minute, By: "ip"}},
		"P2|live": {{Endpoint: "/v1/auth/sign-in/password", Limit: 5, Window: time.Minute, By: "ip"}},
	}}
	h := NewRateLimitMiddleware(reader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if code := doReq(t, h, map[string]string{"X-Client-ID": "P1"}); code != http.StatusOK {
		t.Fatalf("P1 first: got %d", code)
	}
	if code := doReq(t, h, map[string]string{"X-Client-ID": "P1"}); code != http.StatusTooManyRequests {
		t.Fatalf("P1 second: got %d, want 429", code)
	}
	// Same IP+endpoint, different project: independent counter.
	if code := doReq(t, h, map[string]string{"X-Client-ID": "P2"}); code != http.StatusOK {
		t.Fatalf("P2 first: got %d, want 200 (isolated)", code)
	}
}

func TestRateLimitMiddlewareEnvironmentIsolation(t *testing.T) {
	reader := &fakeRateLimitReader{byKey: map[string][]RateLimitRule{
		"P|live": {{Endpoint: "/v1/auth/sign-in/password", Limit: 1, Window: time.Minute, By: "ip"}},
		"P|test": {{Endpoint: "/v1/auth/sign-in/password", Limit: 5, Window: time.Minute, By: "ip"}},
	}}
	h := NewRateLimitMiddleware(reader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusOK {
		t.Fatalf("live first: got %d", code)
	}
	if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusTooManyRequests {
		t.Fatalf("live second: got %d, want 429", code)
	}
	if code := doReq(t, h, map[string]string{"X-Client-ID": "P", "X-Environment": "test"}); code != http.StatusOK {
		t.Fatalf("test env first: got %d, want 200 (isolated)", code)
	}
}

func TestRateLimitMiddlewareReaderErrorFailsOpen(t *testing.T) {
	old := sensitiveLimiter
	sensitiveLimiter = newRateLimiter(2, time.Minute)
	t.Cleanup(func() { sensitiveLimiter = old })

	reader := &fakeRateLimitReader{
		byKey:  map[string][]RateLimitRule{},
		errFor: map[string]error{"P|live": errors.New("db down")},
	}
	h := NewRateLimitMiddleware(reader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// Reader errors -> fall open to hardcoded default (limit 2).
	for i := 0; i < 2; i++ {
		if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusOK {
			t.Fatalf("request %d: got %d", i+1, code)
		}
	}
	if code := doReq(t, h, map[string]string{"X-Client-ID": "P"}); code != http.StatusTooManyRequests {
		t.Fatalf("3rd request: got %d, want 429 (default applies)", code)
	}
}

func TestRateLimitRuleCacheTTL(t *testing.T) {
	reader := &fakeRateLimitReader{byKey: map[string][]RateLimitRule{
		"P|live": {{Endpoint: "/v1/auth/sign-in/password", Limit: 1000, Window: time.Minute, By: "ip"}},
	}}
	pl := newProjectLimiters(reader, time.Minute)
	for i := 0; i < 10; i++ {
		pl.limiterFor(context.Background(), "P", "live", "/v1/auth/sign-in/password", "1.2.3.4")
	}
	if got := atomic.LoadInt64(&reader.calls); got != 1 {
		t.Fatalf("reader called %d times within TTL, want 1", got)
	}
}
