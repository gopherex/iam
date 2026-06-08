package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifierAuthenticate(t *testing.T) {
	server := newVerifyServer(t)
	defer server.Close()

	verifier, err := NewVerifier(Config{
		BaseURL:    server.URL,
		Credential: "service-token",
	}, WithAudience("api"))
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	principal, err := verifier.Authenticate(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Subject != "acct_123" {
		t.Fatalf("principal subject = %q, want acct_123", principal.Subject)
	}
	if principal.ProjectID != "proj_123" {
		t.Fatalf("principal project = %q, want proj_123", principal.ProjectID)
	}
	if principal.SessionID != "sess_123" {
		t.Fatalf("principal session = %q, want sess_123", principal.SessionID)
	}
	if principal.ClientID != "api" {
		t.Fatalf("principal client = %q, want api", principal.ClientID)
	}
	if principal.AAL != 2 {
		t.Fatalf("principal aal = %d, want 2", principal.AAL)
	}
	if got := principal.Scopes; len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("principal scopes = %#v, want [read write]", got)
	}
	if got := principal.AMR; len(got) != 2 || got[0] != "pwd" || got[1] != "mfa" {
		t.Fatalf("principal amr = %#v, want [pwd mfa]", got)
	}
}

func TestVerifierAuthenticateRejectsInvalidToken(t *testing.T) {
	server := newVerifyServer(t)
	defer server.Close()

	verifier, err := NewVerifier(Config{
		BaseURL:    server.URL,
		Credential: "service-token",
	})
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	_, err = verifier.Authenticate(context.Background(), "invalid-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Authenticate() error = %v, want ErrInvalidToken", err)
	}
}

func TestHTTPMiddlewareStoresPrincipal(t *testing.T) {
	server := newVerifyServer(t)
	defer server.Close()

	verifier, err := NewVerifier(Config{
		BaseURL:    server.URL,
		Credential: "service-token",
	})
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	handler := verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFrom(r.Context())
		if !ok {
			t.Fatal("principal missing from context")
		}
		_, _ = w.Write([]byte(principal.ProjectID))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "proj_123" {
		t.Fatalf("body = %q, want proj_123", rec.Body.String())
	}
}

func TestHTTPMiddlewareRejectsMissingToken(t *testing.T) {
	server := newVerifyServer(t)
	defer server.Close()

	verifier, err := NewVerifier(Config{
		BaseURL:    server.URL,
		Credential: "service-token",
	})
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	handler := verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func newVerifyServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tokens/verify" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer service-token" {
			http.Error(w, "bad service credential", http.StatusUnauthorized)
			return
		}
		var req struct {
			Token    string `json:"token"`
			Audience string `json:"audience"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Token != "valid-token" {
			writeJSON(w, map[string]any{"valid": false, "error": "invalid_token"})
			return
		}
		if req.Audience != "" && req.Audience != "api" {
			writeJSON(w, map[string]any{"valid": false, "error": "invalid_audience"})
			return
		}
		writeJSON(w, map[string]any{
			"valid": true,
			"claims": map[string]any{
				"sub":   "acct_123",
				"pid":   "proj_123",
				"sid":   "sess_123",
				"aud":   "api",
				"env":   "live",
				"scope": "read write",
				"aal":   2,
				"amr":   []string{"pwd", "mfa"},
			},
		})
	}))
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
