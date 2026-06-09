//go:build integration

package postgres

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
	"github.com/gopherex/iam/pkg/api"
)

// e2eServer assembles the same admin handler + middleware pipeline main.go wires,
// over the shared testcontainers Postgres, and returns a live httptest server.
func e2eServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler := api.New(
		api.WithAdmin(api.NewAdminService(api.AdminDeps{
			Users:           NewPgAdminUsers(testDB, nopEmitter{}),
			Apps:            NewPgAdminApps(testDB, nopEmitter{}),
			ServiceAccounts: NewPgAdminServiceAccounts(testDB, nopEmitter{}),
			APIKeys:         NewPgAdminAPIKeys(testDB, nopEmitter{}),
			Connections:     NewPgAdminConnections(testDB, nopEmitter{}),
			Config:          NewPgAdminConfig(testDB, nopEmitter{}),
			Keys:            NewPgAdminKeys(testDB, nopEmitter{}),
			AccessRequests:  NewPgAdminAccessRequests(testDB, nopEmitter{}),
		})),
	)
	auth := NewAuthenticator(testDB, "")
	srv, err := oas.NewServer(handler, api.NewSecurityHandler(auth), oas.WithErrorHandler(api.ErrorHandler))
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	pipeline := api.EnvironmentMiddleware(
		api.CSRFMiddleware(NewPgPlatform(testDB))(
			api.CookieAuthMiddleware(srv)))
	ts := httptest.NewServer(pipeline)
	t.Cleanup(ts.Close)
	return ts
}

// e2eProjectAdmin creates a fresh project and mints a project-admin bearer token.
func e2eProjectAdmin(t *testing.T, ctx context.Context) (projectID, token string) {
	t.Helper()
	op := NewPgOperator(testDB, nopEmitter{})
	proj, err := op.CreateProject(ctx, domain.ProjectCmd{Name: "E2E Admin Config"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	tok, _, err := op.MintAdminToken(ctx, domain.OperatorAdminTokenCmd{
		ProjectID: proj.ID,
		Name:      "e2e",
		Scopes:    []string{"admin:ui"},
		ExpiresAt: nowUTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("mint admin token: %v", err)
	}
	return proj.ID, tok
}

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
		{"consents", base + "/consents", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Environment", "live")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tc.want {
				t.Fatalf("GET %s = %d, want %d\nbody: %s", tc.path, resp.StatusCode, tc.want, body)
			}
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
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET missing app = %d, want 404\nbody: %s", resp.StatusCode, body)
	}
}
