//go:build integration

package postgres

// integration_e2e_consent_gate_test.go — HTTP e2e tests for the consent.required
// gate on the two signup paths:
//
//   - Flow signup: after the identity step the flow halts at accept_consents
//     until every required document is accepted (no session until then), then
//     completes with a session and rotated token. Wrong/missing version → 403.
//   - Non-flow POST /v1/auth/sign-up: a missing required consent → 403 with no
//     user row created; accepting it → 200 with a locale-stamped consent row.
//   - Regression: a project with NO consent config completes exactly as before.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// putConsent writes the project's consent config via the admin PUT endpoint
// (which validates fail-closed and persists to iam_config key=consent).
func putConsent(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token string, documents []map[string]any) {
	t.Helper()
	base := ts.URL + "/v1/projects/" + projectID + "/admin/consents"
	body := map[string]any{"documents": documents}
	r := e2eReq(t, ctx, http.MethodPut, base, body,
		map[string]string{"Authorization": "Bearer " + token, "X-Environment": "live"})
	e2eWantStatus(t, r, http.StatusOK)
}

// requiredConsent is a single required ToS document used by the tests.
func requiredConsent(key, version string) map[string]any {
	return map[string]any{
		"key":      key,
		"version":  version,
		"title":    "Terms of Service",
		"body":     "You must accept the terms.",
		"locale":   "en",
		"required": true,
	}
}

// countConsentRows counts iam_consents rows for a user.
func countConsentRows(t *testing.T, ctx context.Context, userID string) int {
	t.Helper()
	rows, err := models.IamConsents.Query(
		sm.Where(models.IamConsents.Columns.UserID.EQ(psql.Arg(userID))),
	).All(ctx, testDB.Bobx())
	if err != nil {
		t.Fatalf("query consents: %v", err)
	}
	return len(rows)
}

// ─── flow signup gate ──────────────────────────────────────────────────────────

// consentFlowState extends flowState with consents_required for assertions.
type consentFlowState struct {
	flowState
	ConsentsRequired []struct {
		Key     string `json:"key"`
		Version string `json:"version"`
	} `json:"consents_required"`
}

func flowSubmitConsent(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token, action string, payload map[string]any) (consentFlowState, e2eResp) {
	t.Helper()
	body := map[string]any{"action": action, "payload": payload}
	r := e2eReq(t, ctx, http.MethodPost,
		ts.URL+"/v1/auth/flows/"+token+"/submit", body,
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	var fs consentFlowState
	if r.Status == http.StatusOK {
		e2eDecode(t, r, &fs)
	}
	return fs, r
}

func TestE2EConsentGateFlowSignup(t *testing.T) {
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

	// reachAcceptConsents drives a password_first signup flow up to the
	// accept_consents step and returns the current flow state + token.
	reachAcceptConsents := func(t *testing.T, projectID string) consentFlowState {
		t.Helper()
		email := "cg-" + newUUID()[:8] + "@example.com"
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
			"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "CG",
		})
		e2eWantStatus(t, r, http.StatusOK)
		if fs.Step != "verify_email" {
			t.Fatalf("create step = %q, want verify_email", fs.Step)
		}
		chID := findFlowChallengeID(t, ctx, fs.FlowToken)
		code := captureCode(chID)
		if code == "" {
			t.Fatal("no verification code captured")
		}
		fs2, r2 := flowSubmitConsent(t, ctx, ts, projectID, fs.FlowToken, "verify_email", map[string]any{"code": code})
		e2eWantStatus(t, r2, http.StatusOK)
		return fs2
	}

	t.Run("halts at accept_consents with no session", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		putConsent(t, ctx, ts, projectID, token, []map[string]any{requiredConsent("tos", "2026-06-01")})

		fs := reachAcceptConsents(t, projectID)
		if fs.Step != "accept_consents" {
			t.Fatalf("step = %q, want accept_consents", fs.Step)
		}
		if fs.Status != "pending" {
			t.Fatalf("status = %q, want pending", fs.Status)
		}
		if fs.Session != nil {
			t.Fatal("session must not be issued before consents accepted")
		}
		if len(fs.ConsentsRequired) != 1 || fs.ConsentsRequired[0].Key != "tos" || fs.ConsentsRequired[0].Version != "2026-06-01" {
			t.Fatalf("consents_required = %+v, want [{tos 2026-06-01}]", fs.ConsentsRequired)
		}
	})

	t.Run("accept completes flow with session and consent row", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		putConsent(t, ctx, ts, projectID, token, []map[string]any{requiredConsent("tos", "2026-06-01")})

		fs := reachAcceptConsents(t, projectID)
		if fs.Step != "accept_consents" {
			t.Fatalf("pre-accept step = %q, want accept_consents", fs.Step)
		}
		preToken := fs.FlowToken

		fs2, r := flowSubmitConsent(t, ctx, ts, projectID, fs.FlowToken, "accept_consents",
			map[string]any{"accept": []map[string]string{{"key": "tos", "version": "2026-06-01"}}})
		e2eWantStatus(t, r, http.StatusOK)
		if fs2.Status != "completed" || fs2.Step != "completed" {
			t.Fatalf("status/step = %q/%q, want completed/completed", fs2.Status, fs2.Step)
		}
		if fs2.Session == nil || fs2.Session.AccessToken == "" {
			t.Fatal("session not minted on consent completion")
		}
		if fs2.FlowToken == preToken {
			t.Error("token was NOT rotated on consent completion")
		}

		// A consent row must exist for the flow's user with the resolved locale.
		userID := flowUserID(t, ctx, fs2.FlowToken, fs.FlowToken)
		rows, err := models.IamConsents.Query(
			sm.Where(models.IamConsents.Columns.UserID.EQ(psql.Arg(userID))),
		).All(ctx, testDB.Bobx())
		if err != nil {
			t.Fatalf("query consents: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("consent rows = %d, want 1", len(rows))
		}
		if rows[0].DocKey != "tos" || rows[0].Version != "2026-06-01" {
			t.Fatalf("consent row = %s/%s, want tos/2026-06-01", rows[0].DocKey, rows[0].Version)
		}
		if loc, _ := rows[0].Locale.Get(); loc != "en" {
			t.Errorf("consent locale = %q, want en", loc)
		}
	})

	t.Run("missing required consent stays pending with 403", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		putConsent(t, ctx, ts, projectID, token, []map[string]any{requiredConsent("tos", "2026-06-01")})

		fs := reachAcceptConsents(t, projectID)
		// Submit empty acceptance.
		_, r := flowSubmitConsent(t, ctx, ts, projectID, fs.FlowToken, "accept_consents",
			map[string]any{"accept": []map[string]string{}})
		e2eWantStatus(t, r, http.StatusForbidden)

		// Flow remains at accept_consents (token not rotated).
		again, rg := flowGet(t, ctx, ts, projectID, fs.FlowToken)
		e2eWantStatus(t, rg, http.StatusOK)
		if again.Step != "accept_consents" {
			t.Fatalf("step = %q, want accept_consents (still pending)", again.Step)
		}
	})

	t.Run("wrong version rejected with 403", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		putConsent(t, ctx, ts, projectID, token, []map[string]any{requiredConsent("tos", "2026-06-01")})

		fs := reachAcceptConsents(t, projectID)
		_, r := flowSubmitConsent(t, ctx, ts, projectID, fs.FlowToken, "accept_consents",
			map[string]any{"accept": []map[string]string{{"key": "tos", "version": "2026-05-01"}}})
		e2eWantStatus(t, r, http.StatusForbidden)
	})

	t.Run("no consent config completes directly (regression)", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		fs := reachAcceptConsents(t, projectID)
		if fs.Step != "completed" || fs.Status != "completed" {
			t.Fatalf("status/step = %q/%q, want completed (no consent config)", fs.Status, fs.Step)
		}
		if fs.Session == nil || fs.Session.AccessToken == "" {
			t.Fatal("session must be minted directly when no consent configured")
		}
	})

	t.Run("after_verify strategy gates after set_password", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		patchAuth(projectID, token, map[string]any{"mode": "open", "password_strategy": "after_verify"})
		putConsent(t, ctx, ts, projectID, token, []map[string]any{requiredConsent("tos", "2026-06-01")})

		email := "cgav-" + newUUID()[:8] + "@example.com"
		fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{"kind": "signup", "email": email})
		e2eWantStatus(t, r, http.StatusOK)

		chID := findFlowChallengeID(t, ctx, fs.FlowToken)
		code := captureCode(chID)
		fs2, r2 := flowSubmitConsent(t, ctx, ts, projectID, fs.FlowToken, "verify_email", map[string]any{"code": code})
		e2eWantStatus(t, r2, http.StatusOK)
		if fs2.Step != "set_password" {
			t.Fatalf("after verify step = %q, want set_password", fs2.Step)
		}

		fs3, r3 := flowSubmitConsent(t, ctx, ts, projectID, fs2.FlowToken, "set_password",
			map[string]any{"password": "Sup3rStr0ng!Pass"})
		e2eWantStatus(t, r3, http.StatusOK)
		if fs3.Step != "accept_consents" {
			t.Fatalf("after set_password step = %q, want accept_consents", fs3.Step)
		}
		if fs3.Session != nil {
			t.Fatal("session must not be issued before consents accepted (after_verify)")
		}

		fs4, r4 := flowSubmitConsent(t, ctx, ts, projectID, fs3.FlowToken, "accept_consents",
			map[string]any{"accept": []map[string]string{{"key": "tos", "version": "2026-06-01"}}})
		e2eWantStatus(t, r4, http.StatusOK)
		if fs4.Status != "completed" || fs4.Session == nil {
			t.Fatalf("after consent: status=%q sess=%v, want completed/session", fs4.Status, fs4.Session)
		}
	})
}

// flowUserID returns the user_id of the flow row identified by either of the two
// supplied tokens (the row's hash matches the post-rotation token; the pre-token
// is accepted as a fallback for clarity at the call site).
func flowUserID(t *testing.T, ctx context.Context, tokens ...string) string {
	t.Helper()
	rows, err := models.IamFlows.Query().All(ctx, testDB.Bobx())
	if err != nil {
		t.Fatalf("query flows: %v", err)
	}
	hashes := make(map[string]struct{}, len(tokens))
	for _, tok := range tokens {
		hashes[flowHashToken(tok)] = struct{}{}
	}
	for _, row := range rows {
		if _, ok := hashes[row.TokenHash]; ok {
			if uid, ok := row.UserID.Get(); ok {
				return uid
			}
		}
	}
	t.Fatal("flow row / user_id not found for supplied tokens")
	return ""
}

// ─── non-flow signup gate ───────────────────────────────────────────────────────

func TestE2EConsentGateNonFlowSignup(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	signup := func(projectID string, body map[string]any) e2eResp {
		return e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-up", body,
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	}

	t.Run("missing required consent → 403, no user created", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		putConsent(t, ctx, ts, projectID, token, []map[string]any{requiredConsent("tos", "2026-06-01")})

		email := "nfc-" + newUUID()[:8] + "@example.com"
		r := signup(projectID, map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"})
		e2eWantStatus(t, r, http.StatusForbidden)

		// No user row must have been created.
		rows, err := models.IamUsers.Query(
			sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(projectID))),
		).All(ctx, testDB.Bobx())
		if err != nil {
			t.Fatalf("query users: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("user rows = %d, want 0 (rejected before insert)", len(rows))
		}
	})

	t.Run("accepted required consent → 200 with locale-stamped row", func(t *testing.T) {
		projectID, token := e2eProjectAdmin(t, ctx)
		putConsent(t, ctx, ts, projectID, token, []map[string]any{requiredConsent("tos", "2026-06-01")})

		email := "nfc-ok-" + newUUID()[:8] + "@example.com"
		r := signup(projectID, map[string]any{
			"email":    email,
			"password": "Sup3rStr0ng!Pass",
			"consents": []map[string]any{{"key": "tos", "version": "2026-06-01"}},
		})
		e2eWantStatus(t, r, http.StatusOK)

		var body struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		e2eDecode(t, r, &body)
		if body.User.ID == "" {
			t.Fatalf("missing user.id, body=%s", r.Body)
		}
		if n := countConsentRows(t, ctx, body.User.ID); n != 1 {
			t.Fatalf("consent rows = %d, want 1", n)
		}
		rows, _ := models.IamConsents.Query(
			sm.Where(models.IamConsents.Columns.UserID.EQ(psql.Arg(body.User.ID))),
		).All(ctx, testDB.Bobx())
		if loc, _ := rows[0].Locale.Get(); loc != "en" {
			t.Errorf("consent locale = %q, want en", loc)
		}
	})

	t.Run("no consent config → signup succeeds (regression)", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		email := "nfc-none-" + newUUID()[:8] + "@example.com"
		r := signup(projectID, map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"})
		e2eWantStatus(t, r, http.StatusOK)
	})
}
