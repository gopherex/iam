//go:build integration

package postgres

// integration_e2e_environment_test.go — HTTP e2e tests proving Stripe-like
// test/live environment isolation across the auth-critical data path.
//
// Coverage:
//   - The SAME email registers under env=test AND env=live (both succeed; the
//     composite (project_id, environment, primary_email) unique index allows it).
//   - The two registrations are distinct accounts (different user ids).
//   - A token minted in env=test resolves the TEST account (not the live one),
//     and a token minted in env=live resolves the LIVE account.
//   - A duplicate within the SAME environment still conflicts (409).
//   - A flow created in env=test is invisible from env=live (and vice versa).
//   - An unknown environment is rejected (the project must declare it first).

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// e2eCreateEnvironment declares a named environment on a project so the runtime
// will accept it on X-Environment (effectiveEnv validates against iam_environments).
func e2eCreateEnvironment(t *testing.T, ctx context.Context, projectID, name string) {
	t.Helper()
	op := NewPgOperator(testDB, nopEmitter{})
	if _, err := op.CreateEnvironment(ctx, domain.EnvironmentCmd{ProjectID: projectID, Name: name}); err != nil {
		t.Fatalf("create environment %q: %v", name, err)
	}
}

// envSignUp registers email/password under the given environment and returns the
// raw response plus the decoded user id + access token.
func envSignUp(t *testing.T, ctx context.Context, ts string, projectID, env, email, password string) (e2eResp, string, string) {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodPost, ts+"/v1/auth/sign-up",
		map[string]any{"email": email, "password": password},
		map[string]string{"X-Client-Id": projectID, "X-Environment": env},
	)
	var body struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Session struct {
			AccessToken string `json:"access_token"`
		} `json:"session"`
	}
	if r.Status == http.StatusOK {
		e2eDecode(t, r, &body)
	}
	return r, body.User.ID, body.Session.AccessToken
}

// TestE2EEnvironmentIsolation proves test/live data isolation end-to-end.
func TestE2EEnvironmentIsolation(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("same_email_registers_in_test_and_live", func(t *testing.T) {
		projectID := e2eProject(t, ctx) // seeds "live"
		e2eCreateEnvironment(t, ctx, projectID, "test")
		email := fmt.Sprintf("dual+%s@example.com", newUUID())

		// Register the same email in both environments — both must succeed.
		rLive, liveID, liveTok := envSignUp(t, ctx, ts.URL, projectID, "live", email, "Sup3rStr0ng!Live")
		e2eWantStatus(t, rLive, http.StatusOK)
		rTest, testID, testTok := envSignUp(t, ctx, ts.URL, projectID, "test", email, "Sup3rStr0ng!Test")
		e2eWantStatus(t, rTest, http.StatusOK)

		if liveID == "" || testID == "" {
			t.Fatalf("missing user ids: live=%q test=%q", liveID, testID)
		}
		// The two registrations are independent accounts.
		if liveID == testID {
			t.Fatalf("expected distinct accounts per environment, got same id %q", liveID)
		}

		// A token minted in test resolves the TEST account; the live token resolves
		// the LIVE account — proving the session/account lookups are env-correct.
		assertMeID(t, ctx, ts.URL, testTok, testID)
		assertMeID(t, ctx, ts.URL, liveTok, liveID)
	})

	t.Run("duplicate_within_same_env_conflicts", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		e2eCreateEnvironment(t, ctx, projectID, "test")
		email := fmt.Sprintf("dupenv+%s@example.com", newUUID())

		r1, _, _ := envSignUp(t, ctx, ts.URL, projectID, "test", email, "Sup3rStr0ng!Pass")
		e2eWantStatus(t, r1, http.StatusOK)
		r2, _, _ := envSignUp(t, ctx, ts.URL, projectID, "test", email, "Sup3rStr0ng!Pass")
		e2eWantStatus(t, r2, http.StatusConflict)
	})

	t.Run("flow_in_test_invisible_from_live", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		e2eCreateEnvironment(t, ctx, projectID, "test")
		email := fmt.Sprintf("flowenv+%s@example.com", newUUID())

		// Create a signup flow under env=test.
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/flows",
			map[string]any{"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "test"})
		e2eWantStatus(t, r, http.StatusOK)
		var fs flowState
		e2eDecode(t, r, &fs)
		if fs.FlowToken == "" {
			t.Fatalf("flow create: missing flow_token, body=%s", r.Body)
		}

		// The same token resolves under env=test...
		rTest := e2eReq(t, ctx, http.MethodGet,
			fmt.Sprintf("%s/v1/auth/flows/%s", ts.URL, fs.FlowToken), nil,
			map[string]string{"X-Client-Id": projectID, "X-Environment": "test"})
		e2eWantStatus(t, rTest, http.StatusOK)

		// ...but is invisible from env=live (not-found / gone, never leaked).
		rLive := e2eReq(t, ctx, http.MethodGet,
			fmt.Sprintf("%s/v1/auth/flows/%s", ts.URL, fs.FlowToken), nil,
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
		if rLive.Status != http.StatusGone && rLive.Status != http.StatusNotFound {
			t.Fatalf("cross-env flow get: status = %d, want 410/404\nbody: %s", rLive.Status, rLive.Body)
		}
	})

	t.Run("unknown_environment_rejected", func(t *testing.T) {
		projectID := e2eProject(t, ctx) // only "live" is declared
		email := fmt.Sprintf("unknownenv+%s@example.com", newUUID())

		r, _, _ := envSignUp(t, ctx, ts.URL, projectID, "staging", email, "Sup3rStr0ng!Pass")
		// effectiveEnv rejects an undeclared environment with ErrBadRequest (400).
		if r.Status != http.StatusBadRequest {
			t.Fatalf("unknown environment: status = %d, want 400\nbody: %s", r.Status, r.Body)
		}
	})
}

// assertMeID calls GET /v1/users/me with the bearer token and asserts the
// resolved user id, proving the token resolves the expected account.
func assertMeID(t *testing.T, ctx context.Context, ts, token, wantID string) {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodGet, ts+"/v1/users/me", nil, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)
	var body struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	e2eDecode(t, r, &body)
	if body.User.ID != wantID {
		t.Fatalf("users/me id = %q, want %q", body.User.ID, wantID)
	}
}
