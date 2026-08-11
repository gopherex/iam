//go:build integration

package postgres

import (
	"context"
	"net/http"
	"testing"

	"github.com/gopherex/xlog"
)

// TestE2EAdminRetentionPolicy verifies the retention policy round-trips through
// GET/PUT and is used by the GC retention sweep.
func TestE2EAdminRetentionPolicy(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	base := ts.URL + "/v1/projects/" + projectID + "/admin/retention-policy"

	// Default (unset) → empty policy.
	r0 := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, r0, http.StatusOK)

	// Put a policy.
	rp := e2eReq(t, ctx, http.MethodPut, base, map[string]any{
		"audit_log_retention_days": 30,
		"event_retention_days":     14,
		"soft_delete_grace_days":   7,
	}, e2eBearer(token))
	e2eWantStatus(t, rp, http.StatusOK)

	// Get it back.
	rg := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, rg, http.StatusOK)

	var got struct {
		AuditLogRetentionDays int `json:"audit_log_retention_days"`
		EventRetentionDays    int `json:"event_retention_days"`
	}
	e2eDecode(t, rg, &got)

	if got.AuditLogRetentionDays != 30 || got.EventRetentionDays != 14 {
		t.Fatalf("retention round-trip = %+v, want 30/14", got)
	}

	// The GC retention sweep prunes audit rows older than the window.
	old := newUUID()
	if _, err := testDB.Pool.Exec(ctx,
		`INSERT INTO iam_audit_logs (id, project_id, type, at, data) VALUES ($1,$2,'old.event', now() - interval '60 days', '{}'::jsonb)`,
		old, projectID); err != nil {
		t.Fatalf("seed old audit: %v", err)
	}

	testDB.gcRetentionSweep(ctx, xlog.NewJSON())

	var n int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_audit_logs WHERE id=$1`, old).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}

	if n != 0 {
		t.Fatalf("60-day-old audit row not pruned by 30-day retention (count=%d)", n)
	}
}
