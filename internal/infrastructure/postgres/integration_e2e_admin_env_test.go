//go:build integration

package postgres

// integration_e2e_admin_env_test.go — HTTP e2e proof that the ADMIN-MANAGEMENT
// surface honours the X-Environment header. The environment switcher in the
// admin UI sends X-Environment on project-admin calls; this test proves the
// admin user reads/writes are scoped by (project_id, environment) so switching
// the env actually filters the data.
//
// Coverage:
//   - The SAME email is created via admin under env=live AND env=test (both
//     succeed; the two are independent accounts).
//   - GET .../admin/users with X-Environment: live returns ONLY the live user.
//   - GET .../admin/users with X-Environment: test returns ONLY the test user.
//   - GET a live user id under env=test is a 404 (cross-env isolation).

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// adminBearer is the header map for an admin/bearer request scoped to a named
// environment (e2eBearer is hard-wired to "live").
func adminBearer(token, env string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token, "X-Environment": env}
}

// adminCreateUser POSTs a new user via the admin surface in the given
// environment and returns the created user id.
func adminCreateUser(t *testing.T, ctx context.Context, base, token, env, email string) string {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodPost, base,
		map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"},
		adminBearer(token, env))
	e2eWantStatus(t, r, http.StatusCreated)
	var resp struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	e2eDecode(t, r, &resp)
	if resp.User.ID == "" {
		t.Fatalf("admin create (env=%s): missing user.id, body=%s", env, r.Body)
	}
	return resp.User.ID
}

// adminListUserIDs GETs the admin user list in the given environment and returns
// the set of user ids.
func adminListUserIDs(t *testing.T, ctx context.Context, base, token, env string) map[string]bool {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodGet, base, nil, adminBearer(token, env))
	e2eWantStatus(t, r, http.StatusOK)
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	e2eDecode(t, r, &body)
	ids := make(map[string]bool, len(body.Data))
	for _, u := range body.Data {
		ids[u.ID] = true
	}
	return ids
}

// TestE2EAdminUsersEnvironmentScoped proves the admin user surface is env-scoped.
func TestE2EAdminUsersEnvironmentScoped(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	e2eCreateEnvironment(t, ctx, projectID, "test")

	base := ts.URL + "/v1/projects/" + projectID + "/admin/users"
	email := fmt.Sprintf("admin-env-%s@example.com", newUUID()[:8])

	// Create the same email under both environments. Both succeed because the
	// uniqueness constraint is per (project_id, environment, primary_email).
	liveID := adminCreateUser(t, ctx, base, token, "live", email)
	testID := adminCreateUser(t, ctx, base, token, "test", email)
	if liveID == testID {
		t.Fatalf("expected distinct accounts per environment, got same id %q", liveID)
	}

	t.Run("list under live returns only the live user", func(t *testing.T) {
		ids := adminListUserIDs(t, ctx, base, token, "live")
		if !ids[liveID] {
			t.Errorf("live list missing live user %q (got %v)", liveID, ids)
		}
		if ids[testID] {
			t.Errorf("live list leaked test user %q (got %v)", testID, ids)
		}
	})

	t.Run("list under test returns only the test user", func(t *testing.T) {
		ids := adminListUserIDs(t, ctx, base, token, "test")
		if !ids[testID] {
			t.Errorf("test list missing test user %q (got %v)", testID, ids)
		}
		if ids[liveID] {
			t.Errorf("test list leaked live user %q (got %v)", liveID, ids)
		}
	})

	t.Run("get live user under test env is 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base+"/"+liveID, nil, adminBearer(token, "test"))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("get test user under live env is 404", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base+"/"+testID, nil, adminBearer(token, "live"))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("no env header defaults to live", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, base, nil,
			map[string]string{"Authorization": "Bearer " + token})
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		e2eDecode(t, r, &body)
		ids := make(map[string]bool, len(body.Data))
		for _, u := range body.Data {
			ids[u.ID] = true
		}
		if !ids[liveID] {
			t.Errorf("default (no header) list missing live user %q (got %v)", liveID, ids)
		}
		if ids[testID] {
			t.Errorf("default (no header) list leaked test user %q (got %v)", testID, ids)
		}
	})
}
