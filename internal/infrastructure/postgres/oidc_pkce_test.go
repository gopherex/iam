package postgres

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// pkceVerifier is a syntactically valid RFC 7636 verifier (43 characters, the
// minimum, drawn from the unreserved set).
const pkceVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func challengeFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// TestOIDCCodeChallengeForMatchesRFC7636 pins the S256 transformation against
// the worked example in RFC 7636 appendix B.
func TestOIDCCodeChallengeForMatchesRFC7636(t *testing.T) {
	t.Parallel()

	const (
		verifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	)

	if got := oidcCodeChallengeFor(verifier); got != challenge {
		t.Fatalf("oidcCodeChallengeFor(%q) = %q, want %q", verifier, got, challenge)
	}
}

func TestOIDCValidPKCEValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "minimum length", in: strings.Repeat("a", 43), want: true},
		{name: "maximum length", in: strings.Repeat("a", 128), want: true},
		{name: "unreserved set", in: strings.Repeat("a", 39) + "-._~", want: true},
		{name: "too short", in: strings.Repeat("a", 42)},
		{name: "too long", in: strings.Repeat("a", 129)},
		{name: "empty", in: ""},
		{name: "space rejected", in: strings.Repeat("a", 42) + " "},
		{name: "plus rejected", in: strings.Repeat("a", 42) + "+"},
		{name: "slash rejected", in: strings.Repeat("a", 42) + "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := oidcValidPKCEValue(tt.in); got != tt.want {
				t.Fatalf("oidcValidPKCEValue(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestOIDCCheckPKCERequest covers what the authorization endpoint accepts. A
// public client has no secret, so PKCE is the only thing protecting its code and
// is mandatory; a confidential client may omit it.
func TestOIDCCheckPKCERequest(t *testing.T) {
	t.Parallel()

	challenge := challengeFor(pkceVerifier)

	tests := []struct {
		name       string
		clientType string
		challenge  string
		method     string
		wantErr    string
	}{
		{name: "public with S256", clientType: "spa", challenge: challenge, method: "S256"},
		{name: "native with S256", clientType: "native", challenge: challenge, method: "S256"},
		{name: "confidential with S256", clientType: "web", challenge: challenge, method: "S256"},
		{name: "confidential without PKCE", clientType: "web"},
		{name: "machine without PKCE", clientType: "machine"},

		{name: "public without PKCE", clientType: "spa", wantErr: "invalid_request"},
		{name: "native without PKCE", clientType: "native", wantErr: "invalid_request"},
		{
			name: "plain method rejected", clientType: "spa",
			challenge: challenge, method: "plain", wantErr: "invalid_request",
		},
		{
			name: "missing method rejected", clientType: "spa",
			challenge: challenge, wantErr: "invalid_request",
		},
		{
			name: "method without challenge rejected", clientType: "web",
			method: "S256", wantErr: "invalid_request",
		},
		{
			name: "malformed challenge rejected", clientType: "spa",
			challenge: "too-short", method: "S256", wantErr: "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := &domain.AppClient{Type: tt.clientType}

			gotErr, desc := oidcCheckPKCERequest(app, tt.challenge, tt.method)
			if gotErr != tt.wantErr {
				t.Fatalf("oidcCheckPKCERequest = (%q, %q), want error %q", gotErr, desc, tt.wantErr)
			}

			if gotErr != "" && desc == "" {
				t.Fatal("an error code was returned without a description")
			}
		})
	}
}

// TestOIDCVerifyPKCE covers the token endpoint: a code bound to a challenge is
// unusable without the verifier, and a code issued without one is unusable WITH
// a verifier (downgrade protection, RFC 7636 §4.6).
func TestOIDCVerifyPKCE(t *testing.T) {
	t.Parallel()

	challenge := challengeFor(pkceVerifier)

	tests := []struct {
		name      string
		challenge string
		method    string
		verifier  string
		wantErr   bool
	}{
		{name: "matching verifier", challenge: challenge, method: "S256", verifier: pkceVerifier},
		{name: "no PKCE at all", challenge: "", method: "", verifier: ""},

		{name: "missing verifier", challenge: challenge, method: "S256", wantErr: true},
		{
			name: "wrong verifier", challenge: challenge, method: "S256",
			verifier: strings.Repeat("b", 43), wantErr: true,
		},
		{
			name: "malformed verifier", challenge: challenge, method: "S256",
			verifier: "short", wantErr: true,
		},
		{
			// The challenge itself is not the verifier; a client replaying what it
			// sent at authorize time must not get in.
			name: "challenge replayed as verifier", challenge: challenge, method: "S256",
			verifier: challenge, wantErr: true,
		},
		{
			name: "unsupported stored method", challenge: challenge, method: "plain",
			verifier: pkceVerifier, wantErr: true,
		},
		{
			name: "verifier without a challenge", challenge: "", method: "",
			verifier: pkceVerifier, wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := oidcVerifyPKCE(tt.challenge, tt.method, tt.verifier)
			if tt.wantErr {
				if err == nil {
					t.Fatal("oidcVerifyPKCE = nil error, want a rejection")
				}

				if !errors.Is(err, domain.ErrInvalidGrant) {
					t.Fatalf("oidcVerifyPKCE error = %v, want invalid_grant", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("oidcVerifyPKCE: %v", err)
			}
		})
	}
}
