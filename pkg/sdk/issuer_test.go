package sdk

import "testing"

// TestIssuerForIsAbsolute pins the SDK's view of the canonical IAM issuer: an
// absolute URL built from the deployment base URL, matching what the server
// mints and what the discovery document advertises.
func TestIssuerForIsAbsolute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseURL     string
		projectID   string
		environment string
		want        string
	}{
		{
			name:        "canonical",
			baseURL:     "https://auth.example.com",
			projectID:   "proj_123",
			environment: "live",
			want:        "https://auth.example.com/p/proj_123/e/live",
		},
		{
			name:        "trailing slash trimmed",
			baseURL:     "https://auth.example.com/",
			projectID:   "proj_123",
			environment: "live",
			want:        "https://auth.example.com/p/proj_123/e/live",
		},
		{
			// Without a base URL there is no issuer to pin; returning "" leaves the
			// issuer check disabled rather than pinning a value that can never match.
			name:        "missing base url",
			projectID:   "proj_123",
			environment: "live",
		},
		{name: "missing project", baseURL: "https://auth.example.com", environment: "live"},
		{name: "missing environment", baseURL: "https://auth.example.com", projectID: "proj_123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := issuerFor(tt.baseURL, tt.projectID, tt.environment)
			if got != tt.want {
				t.Fatalf("issuerFor(%q, %q, %q) = %q, want %q",
					tt.baseURL, tt.projectID, tt.environment, got, tt.want)
			}
		})
	}
}

func TestParseIssuer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		issuer      string
		wantProject string
		wantEnv     string
	}{
		{
			name:        "canonical",
			issuer:      "https://auth.example.com/p/proj_123/e/live",
			wantProject: "proj_123",
			wantEnv:     "live",
		},
		{
			name:        "base with path prefix",
			issuer:      "https://example.com/iam/p/proj_123/e/test",
			wantProject: "proj_123",
			wantEnv:     "test",
		},
		// The pre-1.5 relative issuer is no longer minted and is not accepted.
		{name: "legacy relative form", issuer: "/p/proj_123/e/live"},
		{name: "not a tenant path", issuer: "https://auth.example.com/anything/else"},
		{name: "too short", issuer: "https://auth.example.com/p/proj_123"},
		{name: "empty", issuer: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotProject, gotEnv := parseIssuer(tt.issuer)
			if gotProject != tt.wantProject || gotEnv != tt.wantEnv {
				t.Fatalf("parseIssuer(%q) = (%q, %q), want (%q, %q)",
					tt.issuer, gotProject, gotEnv, tt.wantProject, tt.wantEnv)
			}
		})
	}
}
