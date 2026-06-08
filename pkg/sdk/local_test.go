package sdk

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func TestLocalVerifierAuthenticate(t *testing.T) {
	key := newTestSigningKey(t, "kid-1")
	server := newJWKSServer(t, key.publicSet)
	defer server.Close()

	verifier, err := NewLocalVerifier(LocalConfig{
		BaseURL:     server.URL,
		ProjectID:   "proj_123",
		Environment: "live",
		Audience:    "api",
	})
	if err != nil {
		t.Fatalf("NewLocalVerifier() error = %v", err)
	}
	token := key.sign(t, map[string]any{
		"iss":       "/p/proj_123/e/live",
		"sub":       "acct_123",
		"aud":       "api",
		"client_id": "api",
		"scope":     "read write",
		"typ":       "access",
		"env":       "live",
		"amr":       []string{"pwd"},
		"aal":       1,
	})

	principal, err := verifier.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.ProjectID != "proj_123" {
		t.Fatalf("principal project = %q, want proj_123", principal.ProjectID)
	}
	if principal.Environment != "live" {
		t.Fatalf("principal env = %q, want live", principal.Environment)
	}
	if principal.ClientID != "api" {
		t.Fatalf("principal client = %q, want api", principal.ClientID)
	}
	if got := principal.Scopes; len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("principal scopes = %#v, want [read write]", got)
	}
}

func TestLocalVerifierRejectsInvalidAudience(t *testing.T) {
	key := newTestSigningKey(t, "kid-1")
	server := newJWKSServer(t, key.publicSet)
	defer server.Close()

	verifier, err := NewLocalVerifier(LocalConfig{
		BaseURL:     server.URL,
		ProjectID:   "proj_123",
		Environment: "live",
		Audience:    "api",
	})
	if err != nil {
		t.Fatalf("NewLocalVerifier() error = %v", err)
	}
	token := key.sign(t, map[string]any{
		"iss": "/p/proj_123/e/live",
		"sub": "acct_123",
		"aud": "other",
		"typ": "access",
		"env": "live",
	})

	_, err = verifier.Authenticate(context.Background(), token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Authenticate() error = %v, want ErrInvalidToken", err)
	}
	if err == nil || !containsErrorText(err, "invalid_audience") {
		t.Fatalf("Authenticate() error = %v, want invalid_audience detail", err)
	}
}

func TestLocalVerifierHTTPMiddleware(t *testing.T) {
	key := newTestSigningKey(t, "kid-1")
	server := newJWKSServer(t, key.publicSet)
	defer server.Close()

	verifier, err := NewLocalVerifier(LocalConfig{
		BaseURL:     server.URL,
		ProjectID:   "proj_123",
		Environment: "live",
		Audience:    "api",
	})
	if err != nil {
		t.Fatalf("NewLocalVerifier() error = %v", err)
	}
	token := key.sign(t, map[string]any{
		"iss": "/p/proj_123/e/live",
		"sub": "acct_123",
		"aud": "api",
		"typ": "access",
		"env": "live",
	})
	handler := verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFrom(r.Context())
		if !ok {
			t.Fatal("principal missing from context")
		}
		_, _ = w.Write([]byte(principal.Subject))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "acct_123" {
		t.Fatalf("body = %q, want acct_123", rec.Body.String())
	}
}

type testSigningKey struct {
	private   jwk.Key
	publicSet jwk.Set
}

func newTestSigningKey(t *testing.T, kid string) testSigningKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	privateKey, err := jwk.Import(priv)
	if err != nil {
		t.Fatalf("jwk.Import(private) error = %v", err)
	}
	if err := privateKey.Set(jwk.KeyIDKey, kid); err != nil {
		t.Fatalf("private.Set(kid) error = %v", err)
	}
	if err := privateKey.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		t.Fatalf("private.Set(alg) error = %v", err)
	}

	publicKey, err := jwk.Import(&priv.PublicKey)
	if err != nil {
		t.Fatalf("jwk.Import(public) error = %v", err)
	}
	if err := publicKey.Set(jwk.KeyIDKey, kid); err != nil {
		t.Fatalf("public.Set(kid) error = %v", err)
	}
	if err := publicKey.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		t.Fatalf("public.Set(alg) error = %v", err)
	}
	set := jwk.NewSet()
	if err := set.AddKey(publicKey); err != nil {
		t.Fatalf("set.AddKey() error = %v", err)
	}
	return testSigningKey{private: privateKey, publicSet: set}
}

func (k testSigningKey) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	now := time.Now().UTC()
	builder := jwt.NewBuilder().IssuedAt(now).Expiration(now.Add(time.Hour)).NotBefore(now.Add(-time.Second))
	for key, value := range claims {
		builder = builder.Claim(key, value)
	}
	token, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), k.private))
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	return string(signed)
}

func newJWKSServer(t *testing.T, set jwk.Set) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/p/proj_123/e/live/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(set); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
	}))
}

func containsErrorText(err error, want string) bool {
	return err != nil && strings.Contains(err.Error(), want)
}
