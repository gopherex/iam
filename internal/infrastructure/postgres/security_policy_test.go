package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

func TestSecurityPolicyContinuationValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, url string
		notify    bool
		clients   map[string]string
		field     string
	}{
		{name: "disabled notifications allow no URL"},
		{name: "application HTTPS", url: "https://app.example.com/security"},
		{name: "local development", url: "http://localhost:3000/security"},
		{name: "notifications require default", notify: true, field: "continue_url"},
		{name: "bare hostname", url: "app.example.com", field: "continue_url"},
		{name: "public HTTP", url: "http://app.example.com/security", field: "continue_url"},
		{name: "query", url: "https://app.example.com/security?next=home", field: "continue_url"},
		{name: "fragment", url: "https://app.example.com/#/security", field: "continue_url"},
		{name: "credentials", url: "https://user:password@app.example.com/security", field: "continue_url"},
		{name: "valid client override", clients: map[string]string{"client-1": "https://app.example.com/security"}},
		{name: "invalid override", clients: map[string]string{"client-1": "app.example.com"}, field: `client_urls["client-1"]`},
		{name: "empty client key", clients: map[string]string{"": "https://app.example.com/security"}, field: "client_urls"},
		{name: "blank client key", clients: map[string]string{" ": "https://app.example.com/security"}, field: "client_urls"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			policy := securityDefaultPolicy()
			policy.ContinueURL = tc.url
			policy.Notify = tc.notify
			policy.ClientURLs = tc.clients

			err := validateSecurityPolicy(policy)
			if tc.field == "" {
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("expected policy validation error: %v", err)
			}

			if !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("error does not identify %s: %v", tc.field, err)
			}

			if errors.Is(err, domain.ErrInvalidRedirectURI) {
				t.Fatal("policy error misreported as OAuth redirect mismatch")
			}
		})
	}
}
