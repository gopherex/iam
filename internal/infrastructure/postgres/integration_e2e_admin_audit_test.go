//go:build integration

package postgres

import (
	"context"
	"net/http"
	"testing"
)

// TestE2EAdminAuditLog verifies that a privileged mutation records an audit-log
// row (write side, via the auditing emitter) and that the admin audit endpoints
// list, fetch and export it (read side).
func TestE2EAdminAuditLog(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	// A privileged mutation: create an OAuth provider (emits an event, so the
	// auditing emitter records a row with the admin actor).
	rc := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/projects/"+projectID+"/admin/oauth-providers",
		map[string]any{"provider": "google", "client_id": "cid", "client_secret": "sec", "enabled": true},
		e2eBearer(token))
	e2eWantStatus(t, rc, http.StatusCreated)

	// List audit logs → the create is recorded.
	base := ts.URL + "/v1/projects/" + projectID + "/admin/audit-logs"
	r := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)

	var list struct {
		Data []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			ActorID string `json:"actor_id"`
		} `json:"data"`
	}
	e2eDecode(t, r, &list)

	var found string

	for _, e := range list.Data {
		if e.Type == "config.oauth_provider_created" {
			found = e.ID
		}
	}

	if found == "" {
		t.Fatalf("audit log missing config.oauth_provider_created; got %+v", list.Data)
	}

	// Fetch the single entry by id.
	rg := e2eReq(t, ctx, http.MethodGet, base+"/"+found, nil, e2eBearer(token))
	e2eWantStatus(t, rg, http.StatusOK)

	// Export → returns a queued job.
	re := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/projects/"+projectID+"/admin/audit/export",
		map[string]any{"format": "json"}, e2eBearer(token))
	e2eWantStatus(t, re, http.StatusOK)

	var exp struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	e2eDecode(t, re, &exp)

	if exp.JobID == "" || exp.Status == "" {
		t.Fatalf("export response = %+v, want job_id+status", exp)
	}

	// Unauthenticated → 401.
	rNoAuth := e2eReq(t, ctx, http.MethodGet, base, nil, map[string]string{"X-Environment": "live"})
	e2eWantStatus(t, rNoAuth, http.StatusUnauthorized)
}
