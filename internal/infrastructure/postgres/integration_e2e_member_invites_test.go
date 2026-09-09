//go:build integration

package postgres

// integration_e2e_member_invites_test.go — user-driven invitations with a
// per-user cap: quota enforcement (pending-unexpired + accepted occupy a
// slot), revoke freeing the slot, the per-user override beating the project
// default, the revoked right, creator isolation, and admin invites being
// uncapped.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestE2EMemberInvites(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	// Enable member invitations project-wide with a default cap of 1.
	patchMemberInvites := func(body map[string]any) {
		t.Helper()
		r := e2eReq(t, ctx, http.MethodPatch,
			ts.URL+"/v1/projects/"+projectID+"/admin/config/auth",
			map[string]any{"member_invites": body},
			e2eBearer(adminToken))
		e2eWantStatus(t, r, http.StatusOK)
	}

	// Provision a user through the real signup flow; returns (userID, token).
	signup := func(email string) (string, string) {
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": email, "password": "Sup3rStr0ng!",
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "verify_email" {
			t.Fatalf("signup step = %q", fs.Step)
		}

		chID := findFlowChallengeID(t, ctx, fs.FlowToken)
		code := captureCode(chID)
		if code == "" {
			t.Fatal("no verification code captured")
		}

		fs2, r2 := flowSubmit(t, ctx, ts, projectID, fs.FlowToken, "verify_email", map[string]any{"code": code})
		e2eWantStatus(t, r2, http.StatusOK)

		if fs2.Session == nil || fs2.Session.AccessToken == "" {
			t.Fatalf("no session after verify: %+v", fs2.Session)
		}

		tok := fs2.Session.AccessToken
		me := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/users/me", nil,
			map[string]string{"Authorization": "Bearer " + tok, "X-Client-Id": projectID})
		e2eWantStatus(t, me, http.StatusOK)

		var meBody struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(me.Body, &meBody); err != nil {
			t.Fatalf("decode /users/me: %v", err)
		}

		return meBody.User.ID, tok
	}

	memberHdr := func(tok string) map[string]string {
		return map[string]string{
			"Authorization": "Bearer " + tok,
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
	}

	createInvite := func(tok, email string) e2eResp {
		return e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/invites",
			map[string]any{"email": email}, memberHdr(tok))
	}

	userID, userToken := signup("mi-" + newUUID()[:8] + "@example.com")
	_, otherToken := signup("mo-" + newUUID()[:8] + "@example.com")

	// 1. Disabled → 403.
	patchMemberInvites(map[string]any{"enabled": false, "default_cap": 0})
	r := createInvite(userToken, "friend-0@example.com")
	e2eWantStatus(t, r, http.StatusForbidden)

	// 2. Enabled, default cap 1 → first create OK, token returned once.
	patchMemberInvites(map[string]any{"enabled": true, "default_cap": 1})
	r = createInvite(userToken, "friend-1@example.com")
	e2eWantStatus(t, r, http.StatusCreated)

	var created struct {
		Invite struct {
			ID          string `json:"id"`
			InviteToken string `json:"invite_token"`
		} `json:"invite"`
		Quota struct {
			Cap  int `json:"cap"`
			Used int `json:"used"`
			Left int `json:"left"`
		} `json:"quota"`
	}
	if err := json.Unmarshal(r.Body, &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.Invite.InviteToken == "" {
		t.Fatal("invite_token must be returned exactly once")
	}
	if created.Quota.Cap != 1 || created.Quota.Used != 1 || created.Quota.Left != 0 {
		t.Fatalf("quota after create = %+v", created.Quota)
	}

	// 3. Cap reached → 409.
	r = createInvite(userToken, "friend-2@example.com")
	e2eWantStatus(t, r, http.StatusConflict)

	// 4. Another user has an independent budget.
	r = createInvite(otherToken, "friend-3@example.com")
	e2eWantStatus(t, r, http.StatusCreated)

	// 5. Revoke frees the slot.
	rv := e2eReq(t, ctx, http.MethodPost,
		ts.URL+"/v1/auth/invites/"+created.Invite.ID+"/revoke", nil, memberHdr(userToken))
	e2eWantStatus(t, rv, http.StatusOK)

	r = createInvite(userToken, "friend-4@example.com")
	e2eWantStatus(t, r, http.StatusCreated)

	// 6. Own list: own invitations only + quota.
	lr := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/auth/invites", nil, memberHdr(userToken))
	e2eWantStatus(t, lr, http.StatusOK)

	var list struct {
		Invites []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"invites"`
		Quota struct {
			Cap  int `json:"cap"`
			Used int `json:"used"`
		} `json:"quota"`
	}
	if err := json.Unmarshal(lr.Body, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}

	var pendingID string
	for _, inv := range list.Invites {
		if inv.Status == "pending" {
			pendingID = inv.ID
		}
	}
	if pendingID == "" {
		t.Fatal("pending invite missing from own list")
	}
	if list.Quota.Cap != 1 || list.Quota.Used != 1 {
		t.Fatalf("list quota = %+v", list.Quota)
	}

	// 7. Cross-user revoke → 404 (no probing of others' rows).
	cr := e2eReq(t, ctx, http.MethodPost,
		ts.URL+"/v1/auth/invites/"+pendingID+"/revoke", nil, memberHdr(otherToken))
	e2eWantStatus(t, cr, http.StatusNotFound)

	// 8. Per-user override 0 revokes the right despite the default.
	ur := e2eReq(t, ctx, http.MethodPatch,
		ts.URL+"/v1/projects/"+projectID+"/admin/users/"+userID,
		map[string]any{"invite_cap": 0}, e2eBearer(adminToken))
	e2eWantStatus(t, ur, http.StatusOK)

	r = createInvite(userToken, "friend-5@example.com")
	e2eWantStatus(t, r, http.StatusForbidden)

	// 9. Override above the default raises the budget.
	ur = e2eReq(t, ctx, http.MethodPatch,
		ts.URL+"/v1/projects/"+projectID+"/admin/users/"+userID,
		map[string]any{"invite_cap": 3}, e2eBearer(adminToken))
	e2eWantStatus(t, ur, http.StatusOK)

	r = createInvite(userToken, "friend-6@example.com")
	e2eWantStatus(t, r, http.StatusCreated)

	// 10. Admin invites stay uncapped and do not eat member slots.
	ar := e2eReq(t, ctx, http.MethodPost,
		ts.URL+"/v1/projects/"+projectID+"/admin/invites",
		map[string]any{"email": "admin-invited@example.com"}, e2eBearer(adminToken))
	e2eWantStatus(t, ar, http.StatusCreated)
}
