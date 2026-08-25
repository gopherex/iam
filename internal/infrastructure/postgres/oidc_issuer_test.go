package postgres

import "testing"

const testIssuerBase = "https://auth.example.com"

// TestOIDCIssuerIsAbsolute pins the issuer shape required by OIDC Discovery 1.0
// §3: an absolute URL formed from the deployment's public base URL plus the
// tenant path, so it is a literal prefix of the discovery document's own URL.
func TestOIDCIssuerIsAbsolute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		base      string
		projectID string
		env       string
		want      string
	}{
		{
			name:      "root base",
			base:      testIssuerBase,
			projectID: "52cfe9be-89d7-40b6-8e17-900f4ae141c6",
			env:       "live",
			want:      testIssuerBase + "/p/52cfe9be-89d7-40b6-8e17-900f4ae141c6/e/live",
		},
		{
			name:      "base with path prefix",
			base:      "https://example.com/iam",
			projectID: "proj",
			env:       "test",
			want:      "https://example.com/iam/p/proj/e/test",
		},
		{
			// Fail closed: with no configured base there is no issuer to assert,
			// and an empty string matches nothing on the verify paths.
			name:      "missing base yields empty issuer",
			base:      "",
			projectID: "proj",
			env:       "live",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := oidcIssuer(tt.base, tt.projectID, tt.env); got != tt.want {
				t.Fatalf("oidcIssuer(%q, %q, %q) = %q, want %q", tt.base, tt.projectID, tt.env, got, tt.want)
			}
		})
	}
}

// TestOIDCIssuerPrefixesDiscoveryURL is the property go-oidc (and every other
// conforming client) checks: the discovery document lives at
// <issuer>/.well-known/openid-configuration.
func TestOIDCIssuerPrefixesDiscoveryURL(t *testing.T) {
	t.Parallel()

	issuer := oidcIssuer(testIssuerBase, "proj", "live")
	discovery := testIssuerBase + "/p/proj/e/live/.well-known/openid-configuration"

	if want := issuer + "/.well-known/openid-configuration"; want != discovery {
		t.Fatalf("discovery URL = %q, want issuer-prefixed %q", discovery, want)
	}
}

func TestOIDCParseIssuer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		base        string
		iss         string
		wantProject string
		wantEnv     string
	}{
		{
			name:        "canonical",
			base:        testIssuerBase,
			iss:         testIssuerBase + "/p/proj/e/live",
			wantProject: "proj",
			wantEnv:     "live",
		},
		{
			name:        "base with path prefix",
			base:        "https://example.com/iam",
			iss:         "https://example.com/iam/p/proj/e/test",
			wantProject: "proj",
			wantEnv:     "test",
		},
		{
			// A token minted by somebody else's IdP must not route to one of our
			// tenants just because its path happens to look like ours.
			name: "foreign origin rejected",
			base: testIssuerBase,
			iss:  "https://evil.example.net/p/proj/e/live",
		},
		{
			name: "scheme mismatch rejected",
			base: testIssuerBase,
			iss:  "http://auth.example.com/p/proj/e/live",
		},
		{
			name: "host prefix collision rejected",
			base: testIssuerBase,
			iss:  "https://auth.example.com.evil.net/p/proj/e/live",
		},
		{
			// The pre-1.5 relative form is gone; it is not accepted anywhere.
			name: "legacy relative issuer rejected",
			base: testIssuerBase,
			iss:  "/p/proj/e/live",
		},
		{
			name: "trailing segments rejected",
			base: testIssuerBase,
			iss:  testIssuerBase + "/p/proj/e/live/extra",
		},
		{
			name: "empty env rejected",
			base: testIssuerBase,
			iss:  testIssuerBase + "/p/proj/e/",
		},
		{
			name: "empty base rejects everything",
			base: "",
			iss:  testIssuerBase + "/p/proj/e/live",
		},
		{
			name: "empty issuer",
			base: testIssuerBase,
			iss:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotProject, gotEnv := oidcParseIssuer(tt.base, tt.iss)
			if gotProject != tt.wantProject || gotEnv != tt.wantEnv {
				t.Fatalf("oidcParseIssuer(%q, %q) = (%q, %q), want (%q, %q)",
					tt.base, tt.iss, gotProject, gotEnv, tt.wantProject, tt.wantEnv)
			}
		})
	}
}

// TestOIDCIssuerRoundTrip: whatever oidcIssuer builds, oidcParseIssuer must take
// apart again — the mint and verify paths cannot drift.
func TestOIDCIssuerRoundTrip(t *testing.T) {
	t.Parallel()

	for _, base := range []string{testIssuerBase, "http://localhost:8080", "https://example.com/iam"} {
		projectID, env := oidcParseIssuer(base, oidcIssuer(base, "proj-1", "staging"))
		if projectID != "proj-1" || env != "staging" {
			t.Fatalf("round trip with base %q = (%q, %q), want (proj-1, staging)", base, projectID, env)
		}
	}
}
