package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeOriginSource struct{ origins []string }

func (f fakeOriginSource) AllowedOrigins(context.Context) ([]string, error) { return f.origins, nil }

func TestCORSDynamicOriginAllowed(t *testing.T) {
	// No static origins; the dynamic per-client union allows the origin.
	handler := CORSMiddleware(nil, fakeOriginSource{origins: []string{"https://landing.example.com"}}, time.Minute)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	// Registered dynamic origin -> reflected with credentials.
	req := httptest.NewRequest("GET", "/v1/config/public", nil)
	req.Header.Set("Origin", "https://landing.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://landing.example.com" {
		t.Fatalf("dynamic origin ACAO = %q, want reflected", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("dynamic origin must be credentialed")
	}

	// Unregistered origin -> no ACAO.
	req2 := httptest.NewRequest("GET", "/v1/config/public", nil)
	req2.Header.Set("Origin", "https://evil.com")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if got := rec2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unregistered origin ACAO = %q, want empty", got)
	}
}

func TestCORSWildcardNoCredentials(t *testing.T) {
	handler := CORSMiddleware([]string{"*"}, nil, 0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/v1/auth/register", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight: expected 204, got %d", rec.Code)
	}
	acao := rec.Header().Get("Access-Control-Allow-Origin")
	if acao != "*" {
		t.Fatalf("wildcard CORS: expected ACAO=*, got %q", acao)
	}
	acac := rec.Header().Get("Access-Control-Allow-Credentials")
	if acac == "true" {
		t.Fatal("wildcard CORS must NOT set Allow-Credentials: true")
	}
}

func TestCORSExplicitOriginWithCredentials(t *testing.T) {
	handler := CORSMiddleware([]string{"https://app.example.com"}, nil, 0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/v1/auth/register", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	acao := rec.Header().Get("Access-Control-Allow-Origin")
	if acao != "https://app.example.com" {
		t.Fatalf("explicit origin: expected ACAO=https://app.example.com, got %q", acao)
	}
	acac := rec.Header().Get("Access-Control-Allow-Credentials")
	if acac != "true" {
		t.Fatal("explicit origin should set Allow-Credentials: true")
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	handler := CORSMiddleware([]string{"https://app.example.com"}, nil, 0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/v1/auth/register", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	acao := rec.Header().Get("Access-Control-Allow-Origin")
	if acao != "" {
		t.Fatalf("unknown origin: expected empty ACAO, got %q", acao)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	for _, h := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Strict-Transport-Security",
		"Referrer-Policy",
		"Permissions-Policy",
	} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing security header: %s", h)
		}
	}
	hsts := rec.Header().Get("Strict-Transport-Security")
	if strings.Contains(hsts, "includeSubDomains") {
		t.Errorf("HSTS should not include includeSubDomains: %q", hsts)
	}
}
