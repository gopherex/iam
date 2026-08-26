package postgres

import "testing"

// TestCoreAuthSameOrigin guards the open-redirect check used by the email
// verification callback: only an exact scheme+host match is same-origin.
func TestCoreAuthSameOrigin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b string
		want bool
	}{
		{"https://app.example.com/verified", "https://app.example.com", true},
		{"https://app.example.com/x", "https://app.example.com/", true},
		{"https://APP.example.com/x", "https://app.example.com", true}, // host case-insensitive
		{"http://app.example.com", "https://app.example.com", false},   // scheme differs
		{"https://evil.com/phish", "https://app.example.com", false},   // host differs
		{"https://app.example.com.evil.com", "https://app.example.com", false},
		{"/relative/path", "https://app.example.com", false}, // no scheme/host
		{"", "https://app.example.com", false},
		{"https://app.example.com", "", false},
	}

	for _, c := range cases {
		if got := coreAuthSameOrigin(c.a, c.b); got != c.want {
			t.Errorf("coreAuthSameOrigin(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
