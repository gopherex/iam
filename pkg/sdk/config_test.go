package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewAuthenticatorRemote(t *testing.T) {
	server := newVerifyServer(t)
	defer server.Close()

	auth, err := NewAuthenticator(AuthenticatorConfig{
		Mode:       ValidationModeRemote,
		BaseURL:    server.URL,
		Credential: "service-token",
		Audience:   "api",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	principal, err := auth.Authenticate(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.ProjectID != "proj_123" {
		t.Fatalf("project = %q, want proj_123", principal.ProjectID)
	}
}

func TestNewAuthenticatorLocalWarm(t *testing.T) {
	key := newTestSigningKey(t, "kid-1")
	var jwksCalls atomic.Int32
	server := newSDKWiringServer(t, key.publicSet, &jwksCalls, nil)
	defer server.Close()

	auth, err := NewAuthenticator(AuthenticatorConfig{
		Mode:        ValidationModeLocal,
		BaseURL:     server.URL,
		ProjectID:   "proj_123",
		Environment: "live",
		Audience:    "api",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	if err := Warm(context.Background(), auth); err != nil {
		t.Fatalf("Warm() error = %v", err)
	}
	if jwksCalls.Load() != 1 {
		t.Fatalf("jwks calls = %d, want 1", jwksCalls.Load())
	}
}

func TestNewAuthenticatorHybridPrefersLocal(t *testing.T) {
	key := newTestSigningKey(t, "kid-1")
	var verifyCalls atomic.Int32
	server := newSDKWiringServer(t, key.publicSet, nil, &verifyCalls)
	defer server.Close()

	auth, err := NewAuthenticator(AuthenticatorConfig{
		Mode:        ValidationModeHybrid,
		BaseURL:     server.URL,
		Credential:  "service-token",
		ProjectID:   "proj_123",
		Environment: "live",
		Audience:    "api",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	token := key.sign(t, map[string]any{
		"iss": "/p/proj_123/e/live",
		"sub": "acct_123",
		"aud": "api",
		"typ": "access",
		"env": "live",
	})

	principal, err := auth.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Subject != "acct_123" {
		t.Fatalf("subject = %q, want acct_123", principal.Subject)
	}
	if verifyCalls.Load() != 0 {
		t.Fatalf("remote verify calls = %d, want 0", verifyCalls.Load())
	}
}

func TestNewAuthenticatorHybridFallsBackToRemote(t *testing.T) {
	key := newTestSigningKey(t, "kid-1")
	var verifyCalls atomic.Int32
	server := newSDKWiringServer(t, key.publicSet, nil, &verifyCalls)
	defer server.Close()

	auth, err := NewAuthenticator(AuthenticatorConfig{
		Mode:        ValidationModeHybrid,
		BaseURL:     server.URL,
		Credential:  "service-token",
		ProjectID:   "proj_123",
		Environment: "live",
		Audience:    "api",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}

	principal, err := auth.Authenticate(context.Background(), "remote-token")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Subject != "remote_acct" {
		t.Fatalf("subject = %q, want remote_acct", principal.Subject)
	}
	if verifyCalls.Load() != 1 {
		t.Fatalf("remote verify calls = %d, want 1", verifyCalls.Load())
	}
}

func newSDKWiringServer(t *testing.T, set any, jwksCalls *atomic.Int32, verifyCalls *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/p/proj_123/e/live/.well-known/jwks.json":
			if jwksCalls != nil {
				jwksCalls.Add(1)
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(set); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
		case "/v1/tokens/verify":
			if verifyCalls != nil {
				verifyCalls.Add(1)
			}
			if r.Header.Get("Authorization") != "Bearer service-token" {
				http.Error(w, "bad service credential", http.StatusUnauthorized)
				return
			}
			var req struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if req.Token != "remote-token" {
				writeJSON(w, map[string]any{"valid": false, "error": "invalid_token"})
				return
			}
			writeJSON(w, map[string]any{
				"valid": true,
				"claims": map[string]any{
					"sub": "remote_acct",
					"pid": "proj_123",
					"aud": "api",
					"typ": "access",
					"env": "live",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
}
