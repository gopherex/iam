package domain

import "testing"

func TestNormalizeOrigin(t *testing.T) {
	ok := map[string]string{
		"https://app.example.com":      "https://app.example.com",
		"https://App.Example.com":      "https://app.example.com",
		"https://app.example.com/":     "https://app.example.com",
		"https://app.example.com:8443": "https://app.example.com:8443",
		"http://localhost:1421":        "http://localhost:1421",
		"http://127.0.0.1:3000":        "http://127.0.0.1:3000",
	}
	for in, want := range ok {
		if got := NormalizeOrigin(in); got != want {
			t.Errorf("NormalizeOrigin(%q) = %q, want %q", in, got, want)
		}
	}
	// Rejected (return ""): wildcard, null, paths, query, non-http scheme, plain
	// http off-localhost, empty, junk.
	bad := []string{
		"*", "null", "NULL", "",
		"https://app.example.com/path",
		"https://app.example.com?x=1",
		"https://app.example.com#f",
		"http://evil.example.com", // http off-localhost
		"ftp://app.example.com",
		"app.example.com", // no scheme
		"https://user:pass@app.example.com",
	}
	for _, in := range bad {
		if got := NormalizeOrigin(in); got != "" {
			t.Errorf("NormalizeOrigin(%q) = %q, want \"\" (rejected)", in, got)
		}
	}
}

func TestNormalizeOriginsDedup(t *testing.T) {
	got := NormalizeOrigins([]string{
		"https://a.com", "https://A.com/", "*", "bad", "https://b.com",
	})
	if len(got) != 2 || got[0] != "https://a.com" || got[1] != "https://b.com" {
		t.Fatalf("dedup/normalize wrong: %v", got)
	}
}
