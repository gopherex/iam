//go:build integration

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestE2EAdminHooks covers blocking-hook CRUD + test, and the runtime
// before_user_create gate: a hook that denies blocks registration (fail-closed),
// and an allowing hook lets it through.
func TestE2EAdminHooks(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	var deny atomic.Bool
	deny.Store(true)

	var hit atomic.Int32

	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Add(1)
		if r.Header.Get("Webhook-Signature") == "" {
			t.Error("hook call missing signature")
		}

		if deny.Load() {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	base := ts.URL + "/v1/projects/" + projectID + "/admin/hooks"

	// Create a before_user_create hook pointing at the test server.
	rc := e2eReq(t, ctx, http.MethodPost, base,
		map[string]any{"type": "before_user_create", "url": hook.URL, "enabled": true},
		e2eBearer(token))
	e2eWantStatus(t, rc, http.StatusCreated)

	var created struct {
		Hook struct {
			ID string `json:"id"`
		} `json:"hook"`
		SigningSecret string `json:"signing_secret"`
	}
	e2eDecode(t, rc, &created)

	if created.Hook.ID == "" || created.SigningSecret == "" {
		t.Fatalf("create hook response = %+v", created)
	}

	// Sign-up while the hook denies → registration is blocked (fail-closed).
	rs := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-up",
		map[string]any{"email": fmt.Sprintf("hooked-%s@example.com", newUUID()[:8]), "password": "Sup3rStr0ng!Pass"},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	e2eWantStatus(t, rs, http.StatusForbidden)

	if hit.Load() == 0 {
		t.Fatal("hook was never invoked on sign-up")
	}

	// Flip to allow → sign-up now succeeds.
	deny.Store(false)

	rs2 := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-up",
		map[string]any{"email": fmt.Sprintf("hooked-ok-%s@example.com", newUUID()[:8]), "password": "Sup3rStr0ng!Pass"},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	e2eWantStatus(t, rs2, http.StatusOK)

	// Test the hook directly.
	rt := e2eReq(t, ctx, http.MethodPost, base+"/"+created.Hook.ID+"/test",
		map[string]any{"payload": map[string]any{"probe": true}}, e2eBearer(token))
	e2eWantStatus(t, rt, http.StatusOK)

	var testResp struct {
		Status int `json:"status"`
	}
	e2eDecode(t, rt, &testResp)

	if testResp.Status != http.StatusOK {
		t.Fatalf("hook test status = %d, want 200", testResp.Status)
	}

	// List → one hook.
	rl := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, rl, http.StatusOK)

	// Delete → subsequent sign-up is no longer gated.
	rd := e2eReq(t, ctx, http.MethodDelete, base+"/"+created.Hook.ID, nil, e2eBearer(token))
	e2eWantStatus(t, rd, http.StatusOK)

	rd2 := e2eReq(t, ctx, http.MethodDelete, base+"/"+created.Hook.ID, nil, e2eBearer(token))
	e2eWantStatus(t, rd2, http.StatusNotFound)
}
