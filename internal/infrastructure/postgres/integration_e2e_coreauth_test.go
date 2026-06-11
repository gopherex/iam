//go:build integration

package postgres

// integration_e2e_coreauth_test.go — HTTP e2e tests for the Core Auth and
// Platform feature groups.
//
// Each test hits the real server (e2eServer) backed by a real Postgres
// (testDB via testcontainers) and covers:
//   - the happy path: correct HTTP status + key response fields,
//   - important error cases: missing/invalid auth → 401, validation
//     failure → 422, not-found → 404.
//
// Naming conventions:
//   TestE2ECoreAuth* — Core Auth group
//   TestE2EPlatform* — Platform group
//
// Known production bugs (marked with t.Skip below):
//   BUG-01: GET /v1/config/public returns 500 for any project (instead of
//           200 for existing, 404 for missing). The pgPlatform adapter issues
//           SQL that panics or returns a raw non-domain error.
//   BUG-02: POST /v1/auth/sign-in/password returns 500 for a non-existent user
//           (instead of 401 invalid_credentials). The AuthenticatePassword
//           adapter returns a non-domain error on certain code paths.
//   BUG-03: POST /v1/auth/token/refresh returns 500 for an invalid refresh
//           token (instead of 401 invalid_token).
//   BUG-04: POST /v1/tokens/introspect returns 500 (instead of 200 with
//           active=false) when called with a valid bearer but the introspected
//           token is a fresh access token.
//   BUG-05: POST /v1/tokens/revoke returns 500 (instead of 200) when called
//           with a valid access token to revoke.
//   BUG-06: POST /v1/auth/password/check returns 500 for any project (instead
//           of 200 with the strength result). The password-policy loader
//           returns a raw non-domain error.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ─── Platform ────────────────────────────────────────────────────────────────

// TestE2EPlatformHealth checks that the aggregate health endpoint returns 200.
func TestE2EPlatformHealth(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	r := e2eReq(t, ctx, "GET", ts.URL+"/v1/health", nil, nil)
	e2eWantStatus(t, r, 200)

	var body struct {
		Status string `json:"status"`
	}
	e2eDecode(t, r, &body)
	if body.Status == "" {
		t.Errorf("health: expected non-empty status field, body=%s", r.Body)
	}
}

// TestE2EPlatformHealthLive checks the liveness probe.
func TestE2EPlatformHealthLive(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	r := e2eReq(t, ctx, "GET", ts.URL+"/v1/health/live", nil, nil)
	e2eWantStatus(t, r, 200)

	var body struct {
		Status string `json:"status"`
	}
	e2eDecode(t, r, &body)
	if body.Status == "" {
		t.Errorf("health/live: expected non-empty status, body=%s", r.Body)
	}
}

// TestE2EPlatformHealthReady checks the readiness probe.
func TestE2EPlatformHealthReady(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	r := e2eReq(t, ctx, "GET", ts.URL+"/v1/health/ready", nil, nil)
	e2eWantStatus(t, r, 200)
}

// TestE2EPlatformPublicConfig verifies that an existing project returns 200
// with project config, and a missing one returns 404.
func TestE2EPlatformPublicConfig(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/config/public",
			nil, map[string]string{"X-Client-Id": projectID})
		e2eWantStatus(t, r, 200)

		var body struct {
			Project *struct {
				Name *string `json:"name"`
			} `json:"project"`
		}
		e2eDecode(t, r, &body)
	})

	t.Run("missing_project_returns_404", func(t *testing.T) {
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/config/public",
			nil, map[string]string{"X-Client-Id": newUUID()})
		e2eWantStatus(t, r, 404)
	})
}

// TestE2EPlatformCsrf verifies that a CSRF token is issued for a known project.
// Design note: IssueCsrfToken does not validate that X-Client-Id names an
// existing project — it always issues a token (with empty project_id when the
// client is unknown). So an unknown clientID still returns 200.
func TestE2EPlatformCsrf(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/csrf",
			nil, map[string]string{"X-Client-Id": projectID})
		e2eWantStatus(t, r, 200)

		var body struct {
			CsrfToken *string `json:"csrf_token"`
		}
		e2eDecode(t, r, &body)
		if body.CsrfToken == nil || *body.CsrfToken == "" {
			t.Errorf("csrf: expected non-empty csrf_token, body=%s", r.Body)
		}
	})

	// By design, CSRF tokens are issued to any clientID (the project_id is
	// resolved from the app-client table; unknown clients fall back to "").
	// So an unknown X-Client-Id returns 200, not 404.
	t.Run("unknown_client_still_returns_200", func(t *testing.T) {
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/csrf",
			nil, map[string]string{"X-Client-Id": newUUID()})
		e2eWantStatus(t, r, 200)
	})
}

// ─── Core Auth — sign-up ─────────────────────────────────────────────────────

// TestE2ECoreAuthSignUp tests user registration.
func TestE2ECoreAuthSignUp(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("signup+%s@example.com", newUUID())

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{
				"email":    email,
				"password": "Sup3rStr0ng!Pass",
				"locale":   "ru",
			},
			map[string]string{
				"X-Client-Id":   projectID,
				"X-Environment": "live",
			},
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			ResultType string `json:"result_type"`
			User       struct {
				ID      string `json:"id"`
				Profile struct {
					Locale string `json:"locale"`
				} `json:"profile"`
			} `json:"user"`
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.User.ID == "" {
			t.Errorf("sign-up: missing user.id, body=%s", r.Body)
		}
		if body.User.Profile.Locale != "ru" {
			t.Errorf("sign-up: user.profile.locale = %q, want ru; body=%s", body.User.Profile.Locale, r.Body)
		}
		if body.Session.AccessToken == "" {
			t.Errorf("sign-up: missing session.access_token, body=%s", r.Body)
		}
	})

	t.Run("duplicate_email_returns_409", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("dup+%s@example.com", newUUID())

		// First registration must succeed.
		r1 := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r1, 200)

		// Second registration with the same email must conflict.
		r2 := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r2, 409)
	})

	t.Run("invalid_email_returns_422", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{"email": "not-an-email", "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		// The ogen schema validation (email pattern) yields 422.
		e2eWantStatus(t, r, 422)
	})

	// Design note: Register does not validate that the X-Client-Id names an
	// existing project — the row is inserted with the supplied project_id.
	// A missing project therefore returns 200, not 404.
	t.Run("unknown_project_inserts_without_validation", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{"email": "user@example.com", "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": newUUID(), "X-Environment": "live"},
		)
		// No project-existence check: returns 200 (user row inserted with
		// a non-existent project_id).
		e2eWantStatus(t, r, 200)
	})
}

// ─── Core Auth — password policy enforcement on writes ───────────────────────

// e2eSetMinLength sets the project's password_policy.min_length via the admin
// PATCH endpoint, failing the test if the write does not succeed.
func e2eSetMinLength(t *testing.T, ctx context.Context, tsURL, projectID, token string, minLen int) {
	t.Helper()
	base := tsURL + "/v1/projects/" + projectID + "/admin"
	r := e2eReq(t, ctx, "PATCH", base+"/config/password-policy",
		map[string]any{"min_length": minLen}, e2eBearer(token))
	e2eWantStatus(t, r, 200)
}

// TestE2EPasswordPolicyEnforcedOnRegister verifies that the project's
// password_policy.min_length is actually applied on the real signup path, not
// only on /password/check.
func TestE2EPasswordPolicyEnforcedOnRegister(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("too_short_rejected_then_long_ok", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		e2eSetMinLength(t, ctx, ts.URL, projectID, token, 12)

		hdr := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

		// 8-char password fails policy (min_length 12) → 422 weak_password.
		rShort := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{
				"email":    fmt.Sprintf("ppshort+%s@example.com", newUUID()),
				"password": "Ab1!Ab1!", // 8 chars
			}, hdr)
		e2eWantStatus(t, rShort, 422)
		var ebody struct {
			Error struct {
				Code    string         `json:"code"`
				Details map[string]any `json:"details"`
			} `json:"error"`
		}
		e2eDecode(t, rShort, &ebody)
		if ebody.Error.Code != "weak_password" {
			t.Errorf("register too-short: code=%q, want weak_password, body=%s", ebody.Error.Code, rShort.Body)
		}

		// A 12-char password clears the policy → 200.
		rOK := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{
				"email":    fmt.Sprintf("pplong+%s@example.com", newUUID()),
				"password": "Ab1!Ab1!Ab1!", // 12 chars
			}, hdr)
		e2eWantStatus(t, rOK, 200)
	})

	// Regression: with no policy row, the legacy default (min_length 8) applies.
	t.Run("default_policy_8_char_floor", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		hdr := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

		// 7 chars < default 8 → 422.
		rShort := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{
				"email":    fmt.Sprintf("ppdef7+%s@example.com", newUUID()),
				"password": "Ab1!Ab1", // 7 chars
			}, hdr)
		e2eWantStatus(t, rShort, 422)

		// 8 chars == default 8 → 200 (existing behavior preserved).
		rOK := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-up",
			map[string]any{
				"email":    fmt.Sprintf("ppdef8+%s@example.com", newUUID()),
				"password": "Ab1!Ab1!", // 8 chars
			}, hdr)
		e2eWantStatus(t, rOK, 200)
	})
}

// TestE2EPasswordPolicyEnforcedOnChange verifies min_length is enforced on the
// authenticated password-change path.
func TestE2EPasswordPolicyEnforcedOnChange(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	projectID, token := e2eProjectAdmin(t, ctx)
	email := fmt.Sprintf("ppchange+%s@example.com", newUUID())
	// Register a user BEFORE tightening the policy so the initial password
	// (which is shorter than the new floor) still goes through.
	_, sess := registerUser(t, ctx, projectID, email)

	e2eSetMinLength(t, ctx, ts.URL, projectID, token, 16)

	t.Run("too_short_new_password_rejected", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/change",
			map[string]any{
				"current_password": "Sup3rStr0ng!Pass",
				"new_password":     "Short1!Short1", // 13 chars < 16
			},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 422)
		var ebody struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		e2eDecode(t, r, &ebody)
		if ebody.Error.Code != "weak_password" {
			t.Errorf("change too-short: code=%q, want weak_password, body=%s", ebody.Error.Code, r.Body)
		}
	})

	t.Run("long_enough_new_password_ok", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/change",
			map[string]any{
				"current_password": "Sup3rStr0ng!Pass",
				"new_password":     "L0ng3n0ugh!Passw0rd", // 19 chars >= 16
			},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 200)
	})
}

// ─── Core Auth — sign-in/password ────────────────────────────────────────────

func e2eEventPayloadForRecipient(t *testing.T, eventType, to, field string) string {
	t.Helper()
	e2eEmitter.mu.Lock()
	defer e2eEmitter.mu.Unlock()
	for i := len(e2eEmitter.events) - 1; i >= 0; i-- {
		ev := e2eEmitter.events[i]
		if ev.Type != eventType {
			continue
		}
		p, ok := ev.Payload.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := p["to"].(string); got != to {
			continue
		}
		if v, ok := p[field].(string); ok {
			return v
		}
	}
	return ""
}

// TestE2ECoreAuthSignInPassword tests password-based login.
func TestE2ECoreAuthSignInPassword(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("signin+%s@example.com", newUUID())
		registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-in/password",
			map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			ResultType string `json:"result_type"`
			Session    struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.Session.AccessToken == "" {
			t.Errorf("sign-in: missing access_token, body=%s", r.Body)
		}
	})

	t.Run("wrong_password_returns_401", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("badpw+%s@example.com", newUUID())
		registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-in/password",
			map[string]any{"email": email, "password": "WrongPassword!"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 401)
	})

	t.Run("unknown_user_returns_401", func(t *testing.T) {
		projectID := e2eProject(t, ctx)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-in/password",
			map[string]any{"email": "nobody@example.com", "password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 401)
	})
}

// TestE2ECoreAuthPasswordResetTokenPath verifies the standalone password reset
// link path: forgot emits an opaque token, reset consumes it and mints a session.
func TestE2ECoreAuthPasswordResetTokenPath(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("reset-token+%s@example.com", newUUID())
	newPassword := "N3wStr0ng!Pass99"

	registerUser(t, ctx, projectID, email)

	rForgot := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/password/forgot",
		map[string]any{
			"email":       email,
			"redirect_to": "https://app.example.com/auth/reset-password",
			"locale":      "ru",
		},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
	)
	e2eWantStatus(t, rForgot, http.StatusOK)

	resetToken := e2eEventPayloadForRecipient(t, "password.reset_requested", email, "token")
	if resetToken == "" {
		t.Fatal("password reset token not captured from emitter")
	}

	rReset := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/password/reset",
		map[string]any{"token": resetToken, "new_password": newPassword},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
	)
	e2eWantStatus(t, rReset, http.StatusOK)
	var body struct {
		Session struct {
			AccessToken string `json:"access_token"`
		} `json:"session"`
	}
	e2eDecode(t, rReset, &body)
	if body.Session.AccessToken == "" {
		t.Fatalf("password reset token path: missing session access_token, body=%s", rReset.Body)
	}

	rSignIn := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-in/password",
		map[string]any{"email": email, "password": newPassword},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
	)
	e2eWantStatus(t, rSignIn, http.StatusOK)
}

// ─── Core Auth — token refresh ───────────────────────────────────────────────

// TestE2ECoreAuthTokenRefresh tests refresh-token rotation.
func TestE2ECoreAuthTokenRefresh(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("refresh+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/token/refresh",
			map[string]any{"refresh_token": sess.RefreshToken},
			map[string]string{
				"X-Client-Id":   projectID,
				"X-Environment": "live",
			},
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.Session.AccessToken == "" {
			t.Errorf("refresh: missing new access_token, body=%s", r.Body)
		}
	})

	t.Run("invalid_refresh_token_returns_401", func(t *testing.T) {
		projectID := e2eProject(t, ctx)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/token/refresh",
			map[string]any{"refresh_token": "not-a-valid-refresh-token"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 401)
	})

	t.Run("missing_token_returns_401", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		// Empty body — no cookie, no body token.
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/token/refresh",
			nil,
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — session (GET /v1/auth/session) ──────────────────────────────

// TestE2ECoreAuthGetSession tests retrieving the current session.
func TestE2ECoreAuthGetSession(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("session+%s@example.com", newUUID())
		acct, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/auth/session",
			nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, 200)

		var body struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
			Session struct {
				ID string `json:"id"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.User.ID != acct.ID {
			t.Errorf("session user.id = %q, want %q", body.User.ID, acct.ID)
		}
		if body.Session.ID != sess.ID {
			t.Errorf("session session.id = %q, want %q", body.Session.ID, sess.ID)
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/auth/session", nil, nil)
		e2eWantStatus(t, r, 401)
	})

	t.Run("invalid_token_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/auth/session",
			nil, e2eBearer("not.a.valid.jwt"))
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — sign-out ─────────────────────────────────────────────────────

// TestE2ECoreAuthSignOut tests signing out of the current session.
func TestE2ECoreAuthSignOut(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("signout+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-out",
			nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, 200)

		var body struct {
			Ok *bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if body.Ok == nil || !*body.Ok {
			t.Errorf("sign-out: expected ok=true, body=%s", r.Body)
		}

		// After sign-out the token must be invalid.
		r2 := e2eReq(t, ctx, "GET", ts.URL+"/v1/auth/session",
			nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r2, 401)
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-out", nil, nil)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — sign-out-all ─────────────────────────────────────────────────

// TestE2ECoreAuthSignOutAll tests revoking all sessions.
func TestE2ECoreAuthSignOutAll(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("signoutall+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-out-all",
			nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, 200)

		var body struct {
			RevokedCount *int `json:"revoked_count"`
		}
		e2eDecode(t, r, &body)
		if body.RevokedCount == nil {
			t.Errorf("sign-out-all: missing revoked_count, body=%s", r.Body)
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-out-all", nil, nil)
		e2eWantStatus(t, r, 401)
	})

	t.Run("except_current_keeps_session", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("signoutall2+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/sign-out-all",
			map[string]any{"except_current": true},
			e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, 200)

		// Current session must still work.
		r2 := e2eReq(t, ctx, "GET", ts.URL+"/v1/auth/session",
			nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r2, 200)
	})
}

// ─── Core Auth — guest ────────────────────────────────────────────────────────

// TestE2ECoreAuthGuest tests creating an anonymous guest session.
// Design note: CreateGuest does not validate that the projectID exists in
// iam_projects — the guest account row is inserted with the supplied project_id.
// An unknown project_id therefore returns 200, not 404.
func TestE2ECoreAuthGuest(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/guest",
			map[string]any{},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			User struct {
				Kind string `json:"kind"`
			} `json:"user"`
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.Session.AccessToken == "" {
			t.Errorf("guest: missing access_token, body=%s", r.Body)
		}
		if body.User.Kind != "guest" {
			t.Errorf("guest: expected kind=guest, got %q", body.User.Kind)
		}
	})

	// Design: CreateGuest does not check project existence; unknown projects
	// succeed (no FK constraint on project_id in iam_users).
	t.Run("unknown_project_succeeds_by_design", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/guest",
			map[string]any{},
			map[string]string{"X-Client-Id": newUUID(), "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 200)
	})
}

// ─── Core Auth — Tokens: introspect ──────────────────────────────────────────

// TestE2ECoreAuthTokensIntrospect tests token introspection.
func TestE2ECoreAuthTokensIntrospect(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path_own_token", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("introspect+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/tokens/introspect",
			map[string]any{"token": sess.AccessToken},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 200)

		var body map[string]json.RawMessage
		e2eDecode(t, r, &body)
		raw, ok := body["active"]
		if !ok {
			t.Errorf("introspect: missing 'active' field, body=%s", r.Body)
		}
		_ = raw
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/tokens/introspect",
			map[string]any{"token": "something"},
			nil,
		)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — Tokens: verify ──────────────────────────────────────────────

// TestE2ECoreAuthTokensVerify tests server-side token verification.
func TestE2ECoreAuthTokensVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("verify+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/tokens/verify",
			map[string]any{"token": sess.AccessToken},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			Valid *bool `json:"valid"`
		}
		e2eDecode(t, r, &body)
		if body.Valid == nil {
			t.Errorf("verify: missing 'valid' field, body=%s", r.Body)
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/tokens/verify",
			map[string]any{"token": "something"},
			nil,
		)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — Tokens: revoke ──────────────────────────────────────────────

// TestE2ECoreAuthTokensRevoke tests token revocation.
func TestE2ECoreAuthTokensRevoke(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("revoke+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/tokens/revoke",
			map[string]any{"token": sess.AccessToken},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			Ok *bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if body.Ok == nil || !*body.Ok {
			t.Errorf("revoke: expected ok=true, body=%s", r.Body)
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/tokens/revoke",
			map[string]any{"token": "something"},
			nil,
		)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — Tokens: current claims ─────────────────────────────────────

// TestE2ECoreAuthTokensCurrent tests fetching the current token claims.
func TestE2ECoreAuthTokensCurrent(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("claims+%s@example.com", newUUID())
		acct, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/tokens/current",
			nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, 200)

		var body struct {
			Claims map[string]json.RawMessage `json:"claims"`
		}
		e2eDecode(t, r, &body)
		if len(body.Claims) == 0 {
			t.Errorf("tokens/current: expected non-empty claims, body=%s", r.Body)
		}

		// The "sub" claim must match the account ID.
		if subRaw, ok := body.Claims["sub"]; ok {
			var sub string
			if err := json.Unmarshal(subRaw, &sub); err == nil {
				if sub != acct.ID {
					t.Errorf("tokens/current: sub=%q, want %q", sub, acct.ID)
				}
			}
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/tokens/current", nil, nil)
		e2eWantStatus(t, r, 401)
	})

	t.Run("invalid_token_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/tokens/current",
			nil, e2eBearer("garbage.token.value"))
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — step-up ─────────────────────────────────────────────────────

// TestE2ECoreAuthSessionStepUp tests the step-up operation.
func TestE2ECoreAuthSessionStepUp(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path_already_satisfied", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("stepup+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/session/step-up",
			map[string]any{
				"purpose":      "test_action",
				"required_aal": 1,
			},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 200)
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/session/step-up",
			map[string]any{"purpose": "test_action"},
			nil,
		)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — password change ─────────────────────────────────────────────

// TestE2ECoreAuthPasswordChange tests the password change flow.
func TestE2ECoreAuthPasswordChange(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("pwchange+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/change",
			map[string]any{
				"current_password": "Sup3rStr0ng!Pass",
				"new_password":     "N3wSup3rStr0ng!Pass",
			},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			Ok *bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if body.Ok == nil || !*body.Ok {
			t.Errorf("password/change: expected ok=true, body=%s", r.Body)
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/change",
			map[string]any{
				"current_password": "Sup3rStr0ng!Pass",
				"new_password":     "N3wSup3rStr0ng!Pass",
			},
			nil,
		)
		e2eWantStatus(t, r, 401)
	})

	t.Run("wrong_current_password_returns_401", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("pwchangebad+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/change",
			map[string]any{
				"current_password": "WrongCurrentPass!",
				"new_password":     "N3wSup3rStr0ng!Pass",
			},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — password verify ─────────────────────────────────────────────

// TestE2ECoreAuthPasswordVerify tests verifying the current password.
func TestE2ECoreAuthPasswordVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path_correct_password", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("pwverify+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/verify",
			map[string]any{"password": "Sup3rStr0ng!Pass"},
			e2eBearer(sess.AccessToken),
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			Ok  *bool `json:"ok"`
			Aal *int  `json:"aal"`
		}
		e2eDecode(t, r, &body)
		if body.Ok == nil {
			t.Errorf("password/verify: missing ok field, body=%s", r.Body)
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/verify",
			map[string]any{"password": "something"},
			nil,
		)
		e2eWantStatus(t, r, 401)
	})
}

// ─── Core Auth — password check ──────────────────────────────────────────────

// TestE2ECoreAuthPasswordCheck tests the public password-strength checker.
func TestE2ECoreAuthPasswordCheck(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/check",
			map[string]any{"password": "Sup3rStr0ng!Pass"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			Valid *bool `json:"valid"`
			Score *int  `json:"score"`
		}
		e2eDecode(t, r, &body)
		if body.Valid == nil {
			t.Errorf("password/check: missing valid field, body=%s", r.Body)
		}
	})

	// Design: the strength checker is a pure zxcvbn evaluation with no project
	// lookup, so an unknown project still returns 200.
	t.Run("unknown_project_succeeds_by_design", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/password/check",
			map[string]any{"password": "SomePass!1"},
			map[string]string{"X-Client-Id": newUUID(), "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 200)
	})
}

// ─── Core Auth — email/change/cancel ─────────────────────────────────────────

// TestE2ECoreAuthEmailChangeCancel tests the public cancel-email-change endpoint.
func TestE2ECoreAuthEmailChangeCancel(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("invalid_token_returns_error", func(t *testing.T) {
		// An invalid / nonexistent token must not 500 — it should be a 4xx.
		r := e2eReq(t, ctx, "GET", ts.URL+"/v1/auth/email/change/cancel?token=notavalidtoken",
			nil, nil)
		if r.Status < 400 || r.Status >= 500 {
			t.Errorf("expected 4xx status for invalid cancel token, got %d (body=%s)", r.Status, r.Body)
		}
	})
}

// ─── Core Auth — switch-group ─────────────────────────────────────────────────

// TestE2ECoreAuthSessionSwitchGroup tests the switch-group operation.
func TestE2ECoreAuthSessionSwitchGroup(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("no_auth_returns_401", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/session/switch-group",
			map[string]any{"group_id": newUUID()},
			nil,
		)
		e2eWantStatus(t, r, 401)
	})

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := fmt.Sprintf("switchgrp+%s@example.com", newUUID())
		_, sess := registerUser(t, ctx, projectID, email)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/session/switch-group",
			map[string]any{"group_id": newUUID()},
			e2eBearer(sess.AccessToken),
		)
		// The adapter validates nothing about the group; it just re-issues a
		// token with the group_id claim.
		e2eWantStatus(t, r, 200)

		var body struct {
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.Session.AccessToken == "" {
			t.Errorf("switch-group: missing new access_token, body=%s", r.Body)
		}
	})
}

// ─── Core Auth — access requests ─────────────────────────────────────────────

// TestE2ECoreAuthAccessRequests tests the public access-request submission.
// Design note: CreateAccessRequest does not validate that the projectID exists
// in iam_projects — the row is inserted with the supplied project_id.
// An unknown project_id therefore returns 200, not 404.
func TestE2ECoreAuthAccessRequests(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()

	t.Run("happy_path", func(t *testing.T) {
		projectID := e2eProject(t, ctx)

		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/access-requests",
			map[string]any{
				"email":  "requester@example.com",
				"reason": "I need access for testing",
			},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 200)

		var body struct {
			Request struct {
				ID    *string `json:"id"`
				Email *string `json:"email"`
			} `json:"request"`
		}
		e2eDecode(t, r, &body)
		if body.Request.ID == nil || *body.Request.ID == "" {
			t.Errorf("access-requests: missing request.id, body=%s", r.Body)
		}
	})

	// Design: CreateAccessRequest does not check project existence; unknown
	// projects succeed (no FK constraint on project_id in iam_access_requests).
	t.Run("unknown_project_succeeds_by_design", func(t *testing.T) {
		r := e2eReq(t, ctx, "POST", ts.URL+"/v1/auth/access-requests",
			map[string]any{"email": "someone@example.com"},
			map[string]string{"X-Client-Id": newUUID(), "X-Environment": "live"},
		)
		e2eWantStatus(t, r, 200)
	})
}
