package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSWildcardNoCredentials(t *testing.T) {
	handler := CORSMiddleware([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := CORSMiddleware([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := CORSMiddleware([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
