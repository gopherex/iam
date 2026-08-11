//go:build integration

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// TestE2ETestMode covers the /v1/test/* endpoints: seed a fixture user, list
// messages, advance the clock, reset (wipe) the environment, and confirm the
// live-environment guard refuses.
func TestE2ETestMode(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	email := fmt.Sprintf("seed-%s@example.com", newUUID()[:8])
	testHdr := map[string]string{"Authorization": "Bearer " + token, "X-Environment": "test"}

	// Seed a fixture user into the test environment.
	rs := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/seed",
		map[string]any{"email": email, "name": "Seed"}, testHdr)
	e2eWantStatus(t, rs, http.StatusOK)

	var n int
	if err := testDB.Pool.QueryRow(ctx,
		`SELECT count(*) FROM iam_users WHERE project_id=$1 AND environment='test' AND primary_email=$2`,
		projectID, email).Scan(&n); err != nil {
		t.Fatal(err)
	}

	if n != 1 {
		t.Fatalf("seeded user count = %d, want 1", n)
	}

	// Messages + clock respond.
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/test/messages", nil, testHdr), http.StatusOK)
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/clock", map[string]any{"advance_seconds": 3600}, testHdr), http.StatusOK)

	// Reset wipes the test environment.
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/reset", nil, testHdr), http.StatusOK)

	if err := testDB.Pool.QueryRow(ctx,
		`SELECT count(*) FROM iam_users WHERE project_id=$1 AND environment='test' AND primary_email=$2`,
		projectID, email).Scan(&n); err != nil {
		t.Fatal(err)
	}

	if n != 0 {
		t.Fatalf("after reset: user count = %d, want 0", n)
	}

	// The live-environment guard refuses test mode.
	rLive := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/reset", nil,
		map[string]string{"Authorization": "Bearer " + token, "X-Environment": "live"})
	e2eWantStatus(t, rLive, http.StatusForbidden)
}
