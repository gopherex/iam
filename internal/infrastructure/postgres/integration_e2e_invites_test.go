//go:build integration

package postgres

// integration_e2e_invites_test.go — HTTP e2e tests for the invitation system:
// admin create/list/revoke and signup-flow redemption under invite_only.
//
// Coverage:
//   - Admin create invite returns a one-time raw token.
//   - invite_only signup WITHOUT a token → blocked/invite_required.
//   - invite_only signup WITH the token → proceeds to verify_email.
//   - The redeemed invite is marked accepted (list reflects it).
//   - Re-using the same (now accepted) token → blocked/invite_invalid.

import (
	"context"
	"net/http"
	"testing"
)

func TestE2EInvites(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	projectID, token := e2eProjectAdmin(t, ctx)

	patchAuth := func(reg map[string]any) {
		t.Helper()
		base := ts.URL + "/v1/projects/" + projectID + "/admin/config/auth"
		r := e2eReq(t, ctx, http.MethodPatch, base, map[string]any{"registration": reg}, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
	}

	invitesURL := ts.URL + "/v1/projects/" + projectID + "/admin/invites"

	// 1. Set registration mode to invite_only.
	patchAuth(map[string]any{"mode": "invite_only"})

	// 2. Admin creates an invite bound to an email.
	email := "inv-" + newUUID()[:8] + "@example.com"
	var created struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		Status      string `json:"status"`
		InviteToken string `json:"invite_token"`
	}
	{
		r := e2eReq(t, ctx, http.MethodPost, invitesURL, map[string]any{"email": email}, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)
		e2eDecode(t, r, &created)
	}
	if created.InviteToken == "" {
		t.Fatal("create invite did not return invite_token")
	}
	if created.Status != "pending" {
		t.Fatalf("created status = %q, want pending", created.Status)
	}
	if created.ID == "" {
		t.Fatal("create invite did not return id")
	}

	// 3. Signup WITHOUT a token → blocked/invite_required.
	{
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": email, "password": "Sup3rStr0ng!",
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "blocked" || fs.Error == nil || fs.Error.Code != "invite_required" {
			t.Fatalf("no-token: step=%q err=%+v, want blocked/invite_required", fs.Step, fs.Error)
		}
	}

	// 4. Signup WITH the token → proceeds to verify_email.
	{
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": email, "password": "Sup3rStr0ng!",
			"invite_token": created.InviteToken,
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "verify_email" {
			t.Fatalf("with-token: step = %q (err=%+v), want verify_email", fs.Step, fs.Error)
		}
	}

	// 5. The invite is now accepted (list reflects it).
	{
		r := e2eReq(t, ctx, http.MethodGet, invitesURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var list struct {
			Invites []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"invites"`
		}
		e2eDecode(t, r, &list)
		var found bool
		for _, inv := range list.Invites {
			if inv.ID == created.ID {
				found = true
				if inv.Status != "accepted" {
					t.Fatalf("invite status = %q, want accepted", inv.Status)
				}
			}
		}
		if !found {
			t.Fatalf("created invite %s not in list", created.ID)
		}
	}

	// 6. Re-using the now-accepted token → blocked/invite_invalid.
	{
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": "inv2-" + newUUID()[:8] + "@example.com", "password": "Sup3rStr0ng!",
			"invite_token": created.InviteToken,
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "blocked" || fs.Error == nil || fs.Error.Code != "invite_invalid" {
			t.Fatalf("reuse: step=%q err=%+v, want blocked/invite_invalid", fs.Step, fs.Error)
		}
	}

	// 7. Revoke flow: create another invite then revoke it; redeeming → invite_invalid.
	var created2 struct {
		ID          string `json:"id"`
		InviteToken string `json:"invite_token"`
	}
	{
		r := e2eReq(t, ctx, http.MethodPost, invitesURL, map[string]any{}, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)
		e2eDecode(t, r, &created2)
	}
	{
		r := e2eReq(t, ctx, http.MethodPost, invitesURL+"/"+created2.ID+"/revoke", nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
	}
	{
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": "inv3-" + newUUID()[:8] + "@example.com", "password": "Sup3rStr0ng!",
			"invite_token": created2.InviteToken,
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "blocked" || fs.Error == nil || fs.Error.Code != "invite_invalid" {
			t.Fatalf("revoked: step=%q err=%+v, want blocked/invite_invalid", fs.Step, fs.Error)
		}
	}
}
