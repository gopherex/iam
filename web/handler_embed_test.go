//go:build embed

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandlerServesSPAShellForClientRoutes verifies the other half of the
// routing the provider screens rely on: an unknown path is not a 404 but the SPA
// shell, so /oauth/interaction/<id> is rendered client-side.
//
// Build-tagged `embed` because it needs the built assets (make build / yarn build).
func TestHandlerServesSPAShellForClientRoutes(t *testing.T) {
	t.Parallel()

	h := Handler()

	for _, path := range []string{
		"/oauth/interaction/2f1c1b1e-0000-4000-8000-000000000000",
		"/oauth/device",
		"/projects",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("%s = %d, want 200 (the SPA shell)", path, rec.Code)
			}

			if body := rec.Body.String(); !strings.Contains(body, "<div id=\"root\"") {
				t.Fatalf("%s did not serve the SPA shell; body starts %.80q", path, body)
			}
		})
	}
}
