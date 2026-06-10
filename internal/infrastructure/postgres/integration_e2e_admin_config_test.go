//go:build integration

package postgres

import (
	"context"
	"net/http"
	"testing"
)

// TestE2EAdminConfigFreshProject reproduces the admin UI loading config pages on
// a brand-new project (no config rows written). Every implemented config GET must
// return 200 with an (empty) document — never a 500.
func TestE2EAdminConfigFreshProject(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	base := ts.URL + "/v1/projects/" + projectID + "/admin"
	cases := []struct {
		name string
		path string
		want int
	}{
		{"auth", base + "/config/auth", http.StatusOK},
		{"password-policy", base + "/config/password-policy", http.StatusOK},
		{"session-policy", base + "/config/session-policy", http.StatusOK},
		{"mfa-policy", base + "/config/mfa-policy", http.StatusOK},
		{"rate-limits", base + "/config/rate-limits", http.StatusOK},
		{"consents", base + "/consents", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e2eReq(t, ctx, http.MethodGet, tc.path, nil, e2eBearer(token))
			e2eWantStatus(t, r, tc.want)
		})
	}
}

// TestE2EAdminNotFoundIs404 guards the not-found translation: a missing record
// (a no-rows from the store) must surface as 404, never as a 500. This is the
// regression contract for the pgx/sql ErrNoRows sentinel mismatch.
func TestE2EAdminNotFoundIs404(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	path := ts.URL + "/v1/projects/" + projectID + "/admin/apps/" + newUUID()
	r := e2eReq(t, ctx, http.MethodGet, path, nil, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusNotFound)
}
