package config

import (
	"strings"
	"testing"
)

// baseConfig returns a Config whose only interesting field is the one under
// test, so a failure names the field that broke.
func baseConfig(publicURL string) Config {
	c := Config{}
	c.Service.HTTP.PublicURL = publicURL

	return c
}

func TestNormalizePublicURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "absolute https", in: "https://auth.example.com", want: "https://auth.example.com"},
		{name: "absolute http", in: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "trailing slash trimmed", in: "https://auth.example.com/", want: "https://auth.example.com"},
		{name: "repeated trailing slashes trimmed", in: "https://auth.example.com///", want: "https://auth.example.com"},
		{name: "path prefix kept", in: "https://example.com/iam/", want: "https://example.com/iam"},
		{name: "surrounding space trimmed", in: "  https://auth.example.com  ", want: "https://auth.example.com"},
		{name: "empty rejected", in: "", wantErr: true},
		{name: "blank rejected", in: "   ", wantErr: true},
		// The old relative issuer form must not be accepted as a base URL.
		{name: "relative rejected", in: "/p/proj/e/live", wantErr: true},
		{name: "host only rejected", in: "auth.example.com", wantErr: true},
		{name: "non-http scheme rejected", in: "ftp://auth.example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := baseConfig(tt.in)

			err := cfg.Normalize()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) = nil error, want error", tt.in)
				}

				return
			}

			if err != nil {
				t.Fatalf("Normalize(%q): %v", tt.in, err)
			}

			if got := cfg.Service.HTTP.PublicURL; got != tt.want {
				t.Fatalf("Normalize(%q) public_url = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestNormalizeEmptyPublicURLMessage pins the operator-facing wording: an empty
// public_url must name the key and say it is required, mirroring how the service
// refuses to start without service.auth.encryption_key.
func TestNormalizeEmptyPublicURLMessage(t *testing.T) {
	t.Parallel()

	cfg := baseConfig("")

	err := cfg.Normalize()
	if err == nil {
		t.Fatal("Normalize with empty public_url = nil error, want error")
	}

	msg := err.Error()
	for _, want := range []string{"service.http.public_url", "required"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not mention %q", msg, want)
		}
	}
}
