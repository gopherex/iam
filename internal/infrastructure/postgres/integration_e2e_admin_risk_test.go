//go:build integration

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// TestE2EAdminRiskRules covers risk-rule CRUD and the risk-events read.
func TestE2EAdminRiskRules(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	base := ts.URL + "/v1/projects/" + projectID + "/admin/risk/rules"

	rc := e2eReq(t, ctx, http.MethodPost, base,
		map[string]any{"name": "block-tor", "condition": "ip.tor == true", "action": "block", "enabled": true},
		e2eBearer(token))
	e2eWantStatus(t, rc, http.StatusCreated)

	var rule struct {
		ID string `json:"id"`
	}
	e2eDecode(t, rc, &rule)

	if rule.ID == "" {
		t.Fatal("no rule id")
	}

	rl := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, rl, http.StatusOK)

	rd := e2eReq(t, ctx, http.MethodDelete, base+"/"+rule.ID, nil, e2eBearer(token))
	e2eWantStatus(t, rd, http.StatusOK)

	// Risk events read (empty until an engine populates it).
	re := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/projects/"+projectID+"/admin/risk/events", nil, e2eBearer(token))
	e2eWantStatus(t, re, http.StatusOK)
}

// TestE2EAdminRateLimitBlockEnforced covers manual blocks: blocking an email
// makes a sign-up flow for that email fail, and removing the block restores it.
func TestE2EAdminRateLimitBlockEnforced(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	email := fmt.Sprintf("blocked-%s@example.com", newUUID()[:8])

	// Block the email.
	rb := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/projects/"+projectID+"/admin/rate-limit/blocks",
		map[string]any{"type": "email", "value": email, "reason": "abuse"}, e2eBearer(token))
	e2eWantStatus(t, rb, http.StatusOK)

	var block struct {
		ID string `json:"id"`
	}
	e2eDecode(t, rb, &block)

	if block.ID == "" {
		t.Fatal("no block id")
	}

	// A sign-up flow for the blocked email is refused.
	_, r := flowCreate(t, ctx, ts, projectID, map[string]any{"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass"})
	e2eWantStatus(t, r, http.StatusForbidden)

	// Remove the block → the flow is allowed again.
	rd := e2eReq(t, ctx, http.MethodDelete, ts.URL+"/v1/projects/"+projectID+"/admin/rate-limit/blocks/"+block.ID, nil, e2eBearer(token))
	e2eWantStatus(t, rd, http.StatusOK)

	_, r2 := flowCreate(t, ctx, ts, projectID, map[string]any{"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass"})
	e2eWantStatus(t, r2, http.StatusOK)
}
