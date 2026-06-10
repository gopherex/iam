//go:build integration

package postgres

// integration_e2e_flow_registration_test.go — registration-mode enforcement and
// the password strategy (C2) on the signup flow. These exercise the server-side
// reading of the project auth config that was previously ignored.

import (
	"context"
	"net/http"
	"testing"
)

func TestE2EFlowRegistrationEnforcement(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	patchAuth := func(projectID, token string, reg map[string]any) {
		t.Helper()
		base := ts.URL + "/v1/projects/" + projectID + "/admin/config/auth"
		body := map[string]any{"registration": reg}
		r := e2eReq(t, ctx, http.MethodPatch, base, body,
			map[string]string{"Authorization": "Bearer " + token, "X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusOK)
	}

	t.Run("closed mode blocks signup", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		patchAuth(projectID, token, map[string]any{"mode": "closed"})

		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": "c-" + newUUID()[:8] + "@example.com", "password": "Sup3rStr0ng!",
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "blocked" {
			t.Fatalf("step = %q, want blocked", fs.Step)
		}
		if fs.Error == nil || fs.Error.Code != "registration_closed" {
			t.Fatalf("error = %+v, want registration_closed", fs.Error)
		}
	})

	t.Run("invite_only blocks signup without invite", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		patchAuth(projectID, token, map[string]any{"mode": "invite_only"})

		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": "i-" + newUUID()[:8] + "@example.com", "password": "Sup3rStr0ng!",
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "blocked" || fs.Error == nil || fs.Error.Code != "invite_required" {
			t.Fatalf("step=%q err=%+v, want blocked/invite_required", fs.Step, fs.Error)
		}
	})

	t.Run("request_access routes to request_access step", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		patchAuth(projectID, token, map[string]any{"mode": "request_access"})

		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": "r-" + newUUID()[:8] + "@example.com", "password": "Sup3rStr0ng!",
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "request_access" {
			t.Fatalf("step = %q, want request_access", fs.Step)
		}
	})

	t.Run("open mode signs up normally", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		patchAuth(projectID, token, map[string]any{"mode": "open"})

		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": "o-" + newUUID()[:8] + "@example.com", "password": "Sup3rStr0ng!",
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "verify_email" {
			t.Fatalf("step = %q, want verify_email", fs.Step)
		}
	})

	t.Run("after_verify defers password to set_password step", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		patchAuth(projectID, token, map[string]any{"mode": "open", "password_strategy": "after_verify"})

		email := "av-" + newUUID()[:8] + "@example.com"
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{"kind": "signup", "email": email})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "verify_email" {
			t.Fatalf("create step = %q, want verify_email", fs.Step)
		}

		chID := findFlowChallengeID(t, ctx, fs.FlowToken)
		code := captureCode(chID)
		if code == "" {
			t.Fatal("no verification code captured")
		}

		fs2, r2 := flowSubmit(t, ctx, ts, projectID, fs.FlowToken, "verify_email", map[string]any{"code": code})
		e2eWantStatus(t, r2, http.StatusOK)
		if fs2.Step != "set_password" {
			t.Fatalf("after verify step = %q, want set_password", fs2.Step)
		}
		if fs2.Session != nil {
			t.Fatal("session must not be issued before set_password")
		}

		fs3, r3 := flowSubmit(t, ctx, ts, projectID, fs2.FlowToken, "set_password", map[string]any{"password": "Sup3rStr0ng!Pass"})
		e2eWantStatus(t, r3, http.StatusOK)
		if fs3.Status != "completed" || fs3.Session == nil || fs3.Session.AccessToken == "" {
			t.Fatalf("set_password did not complete with a session: status=%q sess=%+v", fs3.Status, fs3.Session)
		}
	})
}
