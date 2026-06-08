package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
