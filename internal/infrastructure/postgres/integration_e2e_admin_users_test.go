//go:build integration

package postgres

// integration_e2e_admin_users_test.go — HTTP e2e tests for the Admin feature
// group: Users, Apps (app clients + secrets), and Service Accounts.
//
// All tests hit the real server (e2eServer) backed by a real Postgres
// (testDB via testcontainers). Each test is isolated via fresh project IDs
// minted by e2eProjectAdmin.
//
// Naming:
//   TestE2EAdminUsers*          — /v1/projects/{id}/admin/users/*
//   TestE2EAdminApps*           — /v1/projects/{id}/admin/apps/*
//   TestE2EAdminServiceAccounts* — /v1/projects/{id}/admin/service-accounts/*

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// ─── Admin Users ─────────────────────────────────────────────────────────────

// TestE2EAdminUsersListEmpty verifies that a fresh project returns an empty
// user list (200 with data=[]) and that no auth yields 401.
func TestE2EAdminUsersListEmpty(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	base := ts.URL + "/v1/projects/" + projectID + "/admin/users"

	t.Run("happy path returns empty list", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Data []any `json:"data"`
		}
		e2eDecode(t, r, &body)
		if body.Data == nil {
			t.Error("expected non-nil data array")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("cross-project token returns 403", func(t *testing.T) {
		// Admin tokens for a different project are authenticated but not authorized;
		// the server returns 403 Forbidden (not 401 Unauthorized).
		_, otherToken := e2eProjectAdmin(t, ctx)
		r := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(otherToken))
		e2eWantStatus(t, r, http.StatusForbidden)
	})
}

// TestE2EAdminUsersCreate verifies POST /v1/projects/{id}/admin/users —
// create a user as project admin.
func TestE2EAdminUsersCreate(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	base := ts.URL + "/v1/projects/" + projectID + "/admin/users"

	t.Run("happy path creates user", func(t *testing.T) {
		email := fmt.Sprintf("admin-create-%s@example.com", newUUID()[:8])
		body := map[string]any{
			"email":    email,
			"password": "Sup3rStr0ng!Pass",
		}
		r := e2eReq(t, ctx, http.MethodPost, base, body, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)
		var resp struct {
			User struct {
				ID           string `json:"id"`
				PrimaryEmail string `json:"primary_email"`
			} `json:"user"`
		}
		e2eDecode(t, r, &resp)
		if resp.User.ID == "" {
			t.Fatal("expected non-empty user.id")
		}
		// NOTE: oasUser returns the email as primary_email. The admin create
		// endpoint populates PrimaryEmail from domain.Account.PrimaryEmail.
		if resp.User.PrimaryEmail != email {
			t.Errorf("user.primary_email = %q, want %q", resp.User.PrimaryEmail, email)
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, base,
			map[string]any{"email": "x@example.com", "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("invalid email returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, base,
			map[string]any{"email": "not-an-email", "password": "Sup3rStr0ng!Pass"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})
}

// TestE2EAdminUsersGet verifies GET /v1/projects/{id}/admin/users/{user_id}.
func TestE2EAdminUsersGet(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-get-%s@example.com", newUUID()[:8]))

	base := ts.URL + "/v1/projects/" + projectID + "/admin/users/"

	t.Run("happy path returns user", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base+acct.ID, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		e2eDecode(t, r, &resp)
		if resp.User.ID != acct.ID {
			t.Errorf("user.id = %q, want %q", resp.User.ID, acct.ID)
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base+acct.ID, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base+newUUID(), nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("user from different project returns 404", func(t *testing.T) {
		otherProject := e2eProject(t, ctx)
		otherAcct, _ := registerUser(t, ctx, otherProject, fmt.Sprintf("other-%s@example.com", newUUID()[:8]))
		r := e2eReq(t, ctx, http.MethodGet, base+otherAcct.ID, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersList verifies that users registered in a project appear in
// GET /v1/projects/{id}/admin/users list.
func TestE2EAdminUsersList(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-list-%s@example.com", newUUID()[:8]))
	base := ts.URL + "/v1/projects/" + projectID + "/admin/users"

	t.Run("registered user appears in list", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		e2eDecode(t, r, &resp)
		found := false
		for _, u := range resp.Data {
			if u.ID == acct.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("registered user %q not found in admin user list", acct.ID)
		}
	})
}

// TestE2EAdminUsersUpdate verifies PATCH /v1/projects/{id}/admin/users/{user_id}.
func TestE2EAdminUsersUpdate(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-update-%s@example.com", newUUID()[:8]))

	userURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s", ts.URL, projectID, acct.ID)

	t.Run("update name returns updated user", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPatch, userURL,
			map[string]any{"name": "Updated Admin Name"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		e2eDecode(t, r, &resp)
		if resp.User.ID != acct.ID {
			t.Errorf("user.id = %q, want %q", resp.User.ID, acct.ID)
		}
	})

	t.Run("update locale (empty-locale bug fixed) returns 200", func(t *testing.T) {
		// Regression: empty locale in oasUser used to cause ogen 400. Fixed.
		r := e2eReq(t, ctx, http.MethodPatch, userURL,
			map[string]any{"locale": "en"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPatch, userURL,
			map[string]any{"name": "X"},
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodPatch, ghostURL,
			map[string]any{"name": "Ghost"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersDelete verifies DELETE /v1/projects/{id}/admin/users/{user_id}.
func TestE2EAdminUsersDelete(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	t.Run("happy path deletes user", func(t *testing.T) {
		acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-del-%s@example.com", newUUID()[:8]))
		userURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s", ts.URL, projectID, acct.ID)

		r := e2eReq(t, ctx, http.MethodDelete, userURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Ok bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if !body.Ok {
			t.Error("expected ok=true after delete")
		}

		// After deletion the user should no longer be found.
		r2 := e2eReq(t, ctx, http.MethodGet, userURL, nil, e2eBearer(token))
		e2eWantStatus(t, r2, http.StatusNotFound)
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-del-noauth-%s@example.com", newUUID()[:8]))
		userURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s", ts.URL, projectID, acct.ID)
		r := e2eReq(t, ctx, http.MethodDelete, userURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodDelete, ghostURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersBan verifies POST /admin/users/{id}/ban and
// POST /admin/users/{id}/unban.
func TestE2EAdminUsersBan(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-ban-%s@example.com", newUUID()[:8]))

	banURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/ban", ts.URL, projectID, acct.ID)
	unbanURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/unban", ts.URL, projectID, acct.ID)

	t.Run("ban user with reason succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, banURL,
			map[string]any{"reason": "violates TOS"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		e2eDecode(t, r, &resp)
		if resp.User.ID != acct.ID {
			t.Errorf("user.id = %q, want %q", resp.User.ID, acct.ID)
		}
	})

	t.Run("unban user succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, unbanURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		e2eDecode(t, r, &resp)
		if resp.User.ID != acct.ID {
			t.Errorf("user.id = %q, want %q", resp.User.ID, acct.ID)
		}
	})

	t.Run("ban no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, banURL,
			map[string]any{"reason": "test"},
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("ban nonexistent user returns 404", func(t *testing.T) {
		ghostBanURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/ban", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, ghostBanURL,
			map[string]any{"reason": "test"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersVerifyEmail verifies POST /admin/users/{id}/verify-email.
func TestE2EAdminUsersVerifyEmail(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-verifyemail-%s@example.com", newUUID()[:8]))

	verifyURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/verify-email", ts.URL, projectID, acct.ID)

	t.Run("verify email succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		e2eDecode(t, r, &resp)
		if resp.User.ID != acct.ID {
			t.Errorf("user.id = %q, want %q", resp.User.ID, acct.ID)
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/verify-email", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, ghostURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersSetPassword verifies POST /admin/users/{id}/password.
func TestE2EAdminUsersSetPassword(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-setpw-%s@example.com", newUUID()[:8]))

	pwURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/password", ts.URL, projectID, acct.ID)

	t.Run("set password succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, pwURL,
			map[string]any{
				"password":        "N3wStr0ng!Pass",
				"revoke_sessions": false,
			},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Ok bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if !body.Ok {
			t.Error("expected ok=true after set password")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, pwURL,
			map[string]any{"password": "N3wStr0ng!Pass"},
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("missing password returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, pwURL,
			map[string]any{},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/password", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, ghostURL,
			map[string]any{"password": "N3wStr0ng!Pass"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersImpersonate verifies POST /admin/users/{id}/impersonate.
func TestE2EAdminUsersImpersonate(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-imp-%s@example.com", newUUID()[:8]))

	impURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/impersonate", ts.URL, projectID, acct.ID)

	t.Run("impersonate returns url and expires_at", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, impURL,
			map[string]any{
				"reason":           "support request",
				"duration_seconds": 300,
			},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			ImpersonationURL string `json:"impersonation_url"`
			ExpiresAt        string `json:"expires_at"`
		}
		e2eDecode(t, r, &resp)
		if resp.ImpersonationURL == "" {
			t.Error("expected non-empty impersonation_url")
		}
		if resp.ExpiresAt == "" {
			t.Error("expected non-empty expires_at")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, impURL,
			map[string]any{"reason": "support", "duration_seconds": 300},
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("missing required fields returns 422", func(t *testing.T) {
		// reason and duration_seconds are required per schema
		r := e2eReq(t, ctx, http.MethodPost, impURL,
			map[string]any{},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/impersonate", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, ghostURL,
			map[string]any{"reason": "test", "duration_seconds": 300},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersListSessions verifies GET /admin/users/{id}/sessions and
// POST /admin/users/{id}/sessions/revoke.
func TestE2EAdminUsersListSessions(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, sess := registerUser(t, ctx, projectID, fmt.Sprintf("admin-sess-%s@example.com", newUUID()[:8]))

	sessURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/sessions", ts.URL, projectID, acct.ID)
	revokeURL := sessURL + "/revoke"

	t.Run("list sessions returns session containing current", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, sessURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		e2eDecode(t, r, &resp)
		found := false
		for _, s := range resp.Data {
			if s.ID == sess.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("session %q not found in admin sessions list", sess.ID)
		}
	})

	t.Run("list sessions no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, sessURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("list sessions for nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/sessions", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodGet, ghostURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("revoke sessions returns revoked count", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, revokeURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			RevokedCount int `json:"revoked_count"`
		}
		e2eDecode(t, r, &resp)
		if resp.RevokedCount < 1 {
			t.Errorf("revoked_count = %d, want >= 1", resp.RevokedCount)
		}
	})
}

// TestE2EAdminUsersDeleteSession verifies DELETE /admin/users/{id}/sessions/{session_id}.
func TestE2EAdminUsersDeleteSession(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, sess := registerUser(t, ctx, projectID, fmt.Sprintf("admin-delsess-%s@example.com", newUUID()[:8]))

	sessURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/sessions/%s", ts.URL, projectID, acct.ID, sess.ID)

	t.Run("delete specific session succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodDelete, sessURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Ok bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if !body.Ok {
			t.Error("expected ok=true after session delete")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		acct2, sess2 := registerUser(t, ctx, projectID, fmt.Sprintf("admin-delsess-noauth-%s@example.com", newUUID()[:8]))
		url := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/sessions/%s", ts.URL, projectID, acct2.ID, sess2.ID)
		r := e2eReq(t, ctx, http.MethodDelete, url, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EAdminUsersListIdentities verifies GET /admin/users/{id}/identities
// and DELETE /admin/users/{id}/identities/{identity_id}.
func TestE2EAdminUsersListIdentities(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-ident-%s@example.com", newUUID()[:8]))

	identitiesURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/identities", ts.URL, projectID, acct.ID)

	t.Run("list identities returns non-nil data", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, identitiesURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		e2eDecode(t, r, &resp)
		if resp.Data == nil {
			t.Error("expected non-nil data array")
		}
	})

	t.Run("list identities no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, identitiesURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("list identities for nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/identities", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodGet, ghostURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("delete nonexistent identity returns 404", func(t *testing.T) {
		delURL := identitiesURL + "/" + newUUID()
		r := e2eReq(t, ctx, http.MethodDelete, delURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersResetMFA verifies POST /admin/users/{id}/mfa/reset.
func TestE2EAdminUsersResetMFA(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-resetmfa-%s@example.com", newUUID()[:8]))

	mfaURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/mfa/reset", ts.URL, projectID, acct.ID)

	t.Run("reset MFA returns removed count", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, mfaURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			RemovedCount int `json:"removed_count"`
		}
		e2eDecode(t, r, &resp)
		// 0 removed is valid (no MFA factors set up yet).
		if resp.RemovedCount < 0 {
			t.Errorf("removed_count = %d, want >= 0", resp.RemovedCount)
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, mfaURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/mfa/reset", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, ghostURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EAdminUsersAnonymize verifies POST /admin/users/{id}/anonymize.
func TestE2EAdminUsersAnonymize(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	t.Run("anonymize user succeeds", func(t *testing.T) {
		acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-anon-%s@example.com", newUUID()[:8]))
		anonURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/anonymize", ts.URL, projectID, acct.ID)
		r := e2eReq(t, ctx, http.MethodPost, anonURL,
			map[string]any{"reason": "GDPR erasure request"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Ok bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if !body.Ok {
			t.Error("expected ok=true after anonymize")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("admin-anon-noauth-%s@example.com", newUUID()[:8]))
		anonURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/anonymize", ts.URL, projectID, acct.ID)
		r := e2eReq(t, ctx, http.MethodPost, anonURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		ghostURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/anonymize", ts.URL, projectID, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, ghostURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// ─── Admin Apps ───────────────────────────────────────────────────────────────

// TestE2EAdminAppsCRUD verifies the full create→get→list→patch→delete lifecycle
// for /v1/projects/{id}/admin/apps and related endpoints.
func TestE2EAdminAppsCRUD(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	appsURL := fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID)

	// CREATE
	var createdAppID string
	t.Run("create app returns 201 with app", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, appsURL,
			map[string]any{
				"name":          "Test App",
				"type":          "spa",
				"redirect_uris": []string{"https://example.com/callback"},
			},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)
		var resp struct {
			App struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"app"`
		}
		e2eDecode(t, r, &resp)
		if resp.App.ID == "" {
			t.Fatal("expected non-empty app.id")
		}
		if resp.App.Name != "Test App" {
			t.Errorf("app.name = %q, want %q", resp.App.Name, "Test App")
		}
		createdAppID = resp.App.ID
	})

	if createdAppID == "" {
		t.Skip("create failed; skipping dependent subtests")
	}

	appURL := appsURL + "/" + createdAppID

	// GET
	t.Run("get app returns app", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, appURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			App struct {
				ID string `json:"id"`
			} `json:"app"`
		}
		e2eDecode(t, r, &resp)
		if resp.App.ID != createdAppID {
			t.Errorf("app.id = %q, want %q", resp.App.ID, createdAppID)
		}
	})

	// LIST
	t.Run("list apps contains created app", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, appsURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		e2eDecode(t, r, &resp)
		found := false
		for _, a := range resp.Data {
			if a.ID == createdAppID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("created app %q not found in list", createdAppID)
		}
	})

	// PATCH
	t.Run("patch app name succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPatch, appURL,
			map[string]any{"name": "Renamed App"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			App struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"app"`
		}
		e2eDecode(t, r, &resp)
		if resp.App.Name != "Renamed App" {
			t.Errorf("app.name = %q, want %q", resp.App.Name, "Renamed App")
		}
	})

	// SECRETS
	var createdSecretID string
	t.Run("add secret returns secret_id and client_secret", func(t *testing.T) {
		secretsURL := appURL + "/secrets"
		r := e2eReq(t, ctx, http.MethodPost, secretsURL,
			map[string]any{"name": "prod-secret"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)
		var resp struct {
			SecretID     string `json:"secret_id"`
			ClientSecret string `json:"client_secret"`
		}
		e2eDecode(t, r, &resp)
		if resp.SecretID == "" {
			t.Fatal("expected non-empty secret_id")
		}
		if resp.ClientSecret == "" {
			t.Error("expected non-empty client_secret (returned only once)")
		}
		createdSecretID = resp.SecretID
	})

	if createdSecretID != "" {
		t.Run("delete secret succeeds", func(t *testing.T) {
			secretURL := appURL + "/secrets/" + createdSecretID
			r := e2eReq(t, ctx, http.MethodDelete, secretURL, nil, e2eBearer(token))
			e2eWantStatus(t, r, http.StatusOK)
			var body struct {
				Ok bool `json:"ok"`
			}
			e2eDecode(t, r, &body)
			if !body.Ok {
				t.Error("expected ok=true after secret delete")
			}
		})
	}

	// DELETE
	t.Run("delete app succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodDelete, appURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Ok bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if !body.Ok {
			t.Error("expected ok=true after delete")
		}

		// Deleted app must not be found.
		r2 := e2eReq(t, ctx, http.MethodGet, appURL, nil, e2eBearer(token))
		e2eWantStatus(t, r2, http.StatusNotFound)
	})
}

// TestE2EAdminAppsErrors verifies error cases for admin apps endpoints.
func TestE2EAdminAppsErrors(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	appsURL := fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID)

	t.Run("list apps no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, appsURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("create app no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, appsURL,
			map[string]any{"name": "X", "type": "spa"},
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("create app missing name returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, appsURL,
			map[string]any{"type": "spa"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("create app invalid type returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, appsURL,
			map[string]any{"name": "Test", "type": "invalid_type"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("get nonexistent app returns 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, appsURL+"/"+newUUID(), nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("patch nonexistent app returns 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPatch, appsURL+"/"+newUUID(),
			map[string]any{"name": "Ghost"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("delete nonexistent app returns 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodDelete, appsURL+"/"+newUUID(), nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("add secret to nonexistent app returns 404", func(t *testing.T) {
		secretsURL := appsURL + "/" + newUUID() + "/secrets"
		r := e2eReq(t, ctx, http.MethodPost, secretsURL,
			map[string]any{"name": "ghost-secret"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("cross-project token returns 403 on list", func(t *testing.T) {
		// Admin tokens for a different project are authenticated but not authorized;
		// the server returns 403 Forbidden (not 401 Unauthorized).
		_, otherToken := e2eProjectAdmin(t, ctx)
		r := e2eReq(t, ctx, http.MethodGet, appsURL, nil, e2eBearer(otherToken))
		e2eWantStatus(t, r, http.StatusForbidden)
	})
}

// TestE2EAdminAppsSecretsMissingName verifies that adding a secret without a
// name returns 422.
func TestE2EAdminAppsSecretsMissingName(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	appsURL := fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID)

	// Create an app to work with.
	r := e2eReq(t, ctx, http.MethodPost, appsURL,
		map[string]any{"name": "Secret Test App", "type": "machine"},
		e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)
	var created struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
	}
	e2eDecode(t, r, &created)
	if created.App.ID == "" {
		t.Skip("app creation failed; skipping")
	}

	secretsURL := appsURL + "/" + created.App.ID + "/secrets"
	r2 := e2eReq(t, ctx, http.MethodPost, secretsURL,
		map[string]any{}, // missing name
		e2eBearer(token))
	e2eWantStatus(t, r2, http.StatusUnprocessableEntity)
}

// ─── Admin Service Accounts ───────────────────────────────────────────────────

// TestE2EAdminServiceAccountsCRUD verifies the full create→get→list→patch→delete
// lifecycle for /v1/projects/{id}/admin/service-accounts.
func TestE2EAdminServiceAccountsCRUD(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	saURL := fmt.Sprintf("%s/v1/projects/%s/admin/service-accounts", ts.URL, projectID)

	// CREATE
	var createdSAID string
	t.Run("create service account returns 201", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, saURL,
			map[string]any{
				"name":   "CI Deploy Bot",
				"scopes": []string{"deploy:read", "deploy:write"},
			},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)
		var resp struct {
			ServiceAccount struct {
				ID     string   `json:"id"`
				Name   string   `json:"name"`
				Scopes []string `json:"scopes"`
			} `json:"service_account"`
		}
		e2eDecode(t, r, &resp)
		if resp.ServiceAccount.ID == "" {
			t.Fatal("expected non-empty service_account.id")
		}
		if resp.ServiceAccount.Name != "CI Deploy Bot" {
			t.Errorf("service_account.name = %q, want %q", resp.ServiceAccount.Name, "CI Deploy Bot")
		}
		createdSAID = resp.ServiceAccount.ID
	})

	if createdSAID == "" {
		t.Skip("create failed; skipping dependent subtests")
	}

	oneSAURL := saURL + "/" + createdSAID

	// GET
	t.Run("get service account returns account", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, oneSAURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			ServiceAccount struct {
				ID string `json:"id"`
			} `json:"service_account"`
		}
		e2eDecode(t, r, &resp)
		if resp.ServiceAccount.ID != createdSAID {
			t.Errorf("service_account.id = %q, want %q", resp.ServiceAccount.ID, createdSAID)
		}
	})

	// LIST
	t.Run("list service accounts contains created SA", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, saURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		e2eDecode(t, r, &resp)
		found := false
		for _, sa := range resp.Data {
			if sa.ID == createdSAID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("created SA %q not found in list", createdSAID)
		}
	})

	// PATCH
	t.Run("patch service account scopes succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPatch, oneSAURL,
			map[string]any{
				"scopes":   []string{"deploy:read"},
				"disabled": false,
			},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var resp struct {
			ServiceAccount struct {
				ID string `json:"id"`
			} `json:"service_account"`
		}
		e2eDecode(t, r, &resp)
		if resp.ServiceAccount.ID != createdSAID {
			t.Errorf("service_account.id = %q, want %q", resp.ServiceAccount.ID, createdSAID)
		}
	})

	// ADD SECRET
	var createdSecretID string
	t.Run("add secret returns secret_id and client_secret", func(t *testing.T) {
		secretsURL := oneSAURL + "/secrets"
		r := e2eReq(t, ctx, http.MethodPost, secretsURL,
			map[string]any{"name": "deploy-key-v1"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)
		var resp struct {
			SecretID     string `json:"secret_id"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		e2eDecode(t, r, &resp)
		if resp.SecretID == "" {
			t.Fatal("expected non-empty secret_id")
		}
		if resp.ClientSecret == "" {
			t.Error("expected non-empty client_secret (returned only once)")
		}
		if resp.ClientID == "" {
			t.Error("expected non-empty client_id")
		}
		createdSecretID = resp.SecretID
	})

	if createdSecretID != "" {
		t.Run("delete secret succeeds", func(t *testing.T) {
			secretURL := oneSAURL + "/secrets/" + createdSecretID
			r := e2eReq(t, ctx, http.MethodDelete, secretURL, nil, e2eBearer(token))
			e2eWantStatus(t, r, http.StatusOK)
			var body struct {
				Ok bool `json:"ok"`
			}
			e2eDecode(t, r, &body)
			if !body.Ok {
				t.Error("expected ok=true after secret delete")
			}
		})
	}

	// DELETE
	t.Run("delete service account succeeds", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodDelete, oneSAURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Ok bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if !body.Ok {
			t.Error("expected ok=true after delete")
		}

		// Deleted SA must not be found.
		r2 := e2eReq(t, ctx, http.MethodGet, oneSAURL, nil, e2eBearer(token))
		e2eWantStatus(t, r2, http.StatusNotFound)
	})
}

// TestE2EAdminServiceAccountsErrors verifies error cases for service accounts.
func TestE2EAdminServiceAccountsErrors(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	saURL := fmt.Sprintf("%s/v1/projects/%s/admin/service-accounts", ts.URL, projectID)

	t.Run("list no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, saURL, nil,
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("create no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, saURL,
			map[string]any{"name": "X", "scopes": []string{}},
			map[string]string{"X-Environment": "live"})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("create missing name returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, saURL,
			map[string]any{"scopes": []string{"read"}},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("get nonexistent SA returns 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, saURL+"/"+newUUID(), nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("patch nonexistent SA returns 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPatch, saURL+"/"+newUUID(),
			map[string]any{"scopes": []string{"x"}, "disabled": false},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("delete nonexistent SA returns 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodDelete, saURL+"/"+newUUID(), nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("add secret to nonexistent SA returns 404", func(t *testing.T) {
		secretsURL := saURL + "/" + newUUID() + "/secrets"
		r := e2eReq(t, ctx, http.MethodPost, secretsURL,
			map[string]any{"name": "ghost-key"},
			e2eBearer(token))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("cross-project token returns 403 on list", func(t *testing.T) {
		// Admin tokens for a different project are authenticated but not authorized;
		// the server returns 403 Forbidden (not 401 Unauthorized).
		_, otherToken := e2eProjectAdmin(t, ctx)
		r := e2eReq(t, ctx, http.MethodGet, saURL, nil, e2eBearer(otherToken))
		e2eWantStatus(t, r, http.StatusForbidden)
	})
}

// TestE2EAdminServiceAccountsSecretsMissingName verifies that creating a SA
// secret without a name returns 422.
func TestE2EAdminServiceAccountsSecretsMissingName(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	saURL := fmt.Sprintf("%s/v1/projects/%s/admin/service-accounts", ts.URL, projectID)

	r := e2eReq(t, ctx, http.MethodPost, saURL,
		map[string]any{"name": "SA Secret Test", "scopes": []string{"read"}},
		e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)
	var created struct {
		ServiceAccount struct {
			ID string `json:"id"`
		} `json:"service_account"`
	}
	e2eDecode(t, r, &created)
	if created.ServiceAccount.ID == "" {
		t.Skip("SA creation failed; skipping")
	}

	secretsURL := saURL + "/" + created.ServiceAccount.ID + "/secrets"
	r2 := e2eReq(t, ctx, http.MethodPost, secretsURL,
		map[string]any{}, // missing name
		e2eBearer(token))
	e2eWantStatus(t, r2, http.StatusUnprocessableEntity)
}

// TestE2EAdminServiceAccountsDeleteSecretNoAuth verifies that deleting a SA
// secret without auth returns 401.
func TestE2EAdminServiceAccountsDeleteSecretNoAuth(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	saURL := fmt.Sprintf("%s/v1/projects/%s/admin/service-accounts", ts.URL, projectID)

	// Create SA and a secret.
	r := e2eReq(t, ctx, http.MethodPost, saURL,
		map[string]any{"name": "SA Delete Secret NoAuth", "scopes": []string{}},
		e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)
	var created struct {
		ServiceAccount struct {
			ID string `json:"id"`
		} `json:"service_account"`
	}
	e2eDecode(t, r, &created)
	if created.ServiceAccount.ID == "" {
		t.Skip("SA creation failed; skipping")
	}

	secretsURL := saURL + "/" + created.ServiceAccount.ID + "/secrets"
	r2 := e2eReq(t, ctx, http.MethodPost, secretsURL,
		map[string]any{"name": "test-secret"},
		e2eBearer(token))
	e2eWantStatus(t, r2, http.StatusCreated)
	var secretResp struct {
		SecretID string `json:"secret_id"`
	}
	e2eDecode(t, r2, &secretResp)
	if secretResp.SecretID == "" {
		t.Skip("secret creation failed; skipping")
	}

	// Attempt delete without auth.
	secretURL := secretsURL + "/" + secretResp.SecretID
	r3 := e2eReq(t, ctx, http.MethodDelete, secretURL, nil,
		map[string]string{"X-Environment": "live"})
	e2eWantStatus(t, r3, http.StatusUnauthorized)
}

// TestE2EAdminAppsAllowedOrigins verifies app-client allowed_origins round-trip
// + server-side normalization (invalid/wildcard entries dropped, deduped,
// lowercased), which feed the dynamic CORS union.
func TestE2EAdminAppsAllowedOrigins(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	appsURL := fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID)

	r := e2eReq(t, ctx, http.MethodPost, appsURL,
		map[string]any{
			"name": "Origins App",
			"type": "spa",
			"allowed_origins": []string{
				"https://App.Example.com/", // normalized -> https://app.example.com
				"*",                        // dropped
				"http://evil.com",          // dropped (http off-localhost)
				"http://localhost:1421",    // kept
				"https://app.example.com",  // dedup
			},
		},
		e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)
	var resp struct {
		App struct {
			ID             string   `json:"id"`
			AllowedOrigins []string `json:"allowed_origins"`
		} `json:"app"`
	}
	e2eDecode(t, r, &resp)
	got := map[string]bool{}
	for _, o := range resp.App.AllowedOrigins {
		got[o] = true
	}
	if !got["https://app.example.com"] || !got["http://localhost:1421"] {
		t.Fatalf("expected normalized origins kept, got %v", resp.App.AllowedOrigins)
	}
	if got["*"] || got["http://evil.com"] || len(resp.App.AllowedOrigins) != 2 {
		t.Fatalf("invalid origins not dropped/deduped: %v", resp.App.AllowedOrigins)
	}
}
