package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestOAuthUIPathsFallThroughToTheSPA pins the routing the hosted OIDC provider
// screens depend on.
//
// /oauth2/ is the protocol surface (authorize, token, …) and is served by the
// API; the user-facing screens live under /oauth/, one character shorter, and
// must reach the SPA so client-side routing can render them. A prefix list that
// accidentally swallowed /oauth/ would leave the browser staring at a JSON 404
// at the end of every authorization redirect.
func TestOAuthUIPathsFallThroughToTheSPA(t *testing.T) {
	t.Parallel()

	const (
		api = "api"
		spa = "spa"
	)

	mux := http.NewServeMux()
	for _, prefix := range apiPathPrefixes() {
		mux.Handle(prefix, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(api))
		}))
	}

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(spa))
	}))

	tests := []struct {
		path string
		want string
	}{
		{path: "/oauth/interaction/2f1c1b1e-0000-4000-8000-000000000000", want: spa},
		{path: "/oauth/device", want: spa},
		{path: "/oauth/device?user_code=ABCD-EFGH", want: spa},
		{path: "/", want: spa},
		{path: "/projects", want: spa},

		{path: "/oauth2/authorize", want: api},
		{path: "/oauth2/token", want: api},
		{path: "/v1/auth/flows", want: api},
		{path: "/mgmt/v1/projects", want: api},
		{path: "/p/proj/e/live/.well-known/openid-configuration", want: api},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if got := rec.Body.String(); got != tt.want {
				t.Fatalf("%s served by %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
