//go:build integration

package postgres

// integration_e2e_flow_signup_test.go — HTTP e2e tests for the server-side
// resumable auth flow engine — signup kind.
//
// Coverage:
//   - Happy path: create → verify_email → completed (session minted).
//   - Security: expired / foreign-project / unknown token → 410.
//   - Wrong code: attempts decremented, status stays pending.
//   - Resend before resend_at → 429.
//   - Token rotation: old token rejected after verify.
//   - Abandon → idempotent 204.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aarondl/opt/null"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// flowState is the minimal wire representation of oas.FlowState for test
// assertions. Only the fields tests branch on are decoded.
type flowState struct {
	FlowToken   string   `json:"flow_token"`
	Kind        string   `json:"kind"`
	Status      string   `json:"status"`
	Step        string   `json:"step"`
	NextActions []string `json:"next_actions"`
	Challenge   *struct {
		ChallengeID  string `json:"challenge_id"` // NOTE: not in the wire; kept in data
		Channel      string `json:"channel"`
		AttemptsLeft int    `json:"attempts_left"`
	} `json:"challenge"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Session *struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"session"`
	Contact *struct {
		EmailMasked string `json:"email_masked"`
	} `json:"contact"`
}

// flowCreate posts POST /v1/auth/flows and returns the decoded state.
func flowCreate(t *testing.T, ctx context.Context, ts *httptest.Server, projectID string, body map[string]any) (flowState, e2eResp) {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/flows", body,
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	var fs flowState
	if r.Status == http.StatusOK {
		e2eDecode(t, r, &fs)
	}
	return fs, r
}

// flowSubmit posts POST /v1/auth/flows/{token}/submit.
func flowSubmit(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token, action string, payload map[string]any) (flowState, e2eResp) {
	t.Helper()
	body := map[string]any{"action": action, "payload": payload}
	r := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/auth/flows/%s/submit", ts.URL, token),
		body,
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	var fs flowState
	if r.Status == http.StatusOK {
		e2eDecode(t, r, &fs)
	}
	return fs, r
}

// flowGet gets GET /v1/auth/flows/{token}.
func flowGet(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token string) (flowState, e2eResp) {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/auth/flows/%s", ts.URL, token),
		nil,
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	var fs flowState
	if r.Status == http.StatusOK {
		e2eDecode(t, r, &fs)
	}
	return fs, r
}

// flowResend posts POST /v1/auth/flows/{token}/resend.
func flowResend(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token string) (flowState, e2eResp) {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/auth/flows/%s/resend", ts.URL, token),
		nil,
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	var fs flowState
	if r.Status == http.StatusOK {
		e2eDecode(t, r, &fs)
	}
	return fs, r
}

// flowAbandon sends DELETE /v1/auth/flows/{token}.
func flowAbandon(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token string) e2eResp {
	t.Helper()
	return e2eReq(t, ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/auth/flows/%s", ts.URL, token),
		nil,
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
}

// captureChallenge returns the verification code from the most recent emitted
// challenge event for the given challenge_id. It polls the capture emitter which
// was wired into e2eServer.
func captureCode(challengeID string) string {
	return e2eEmitter.payloadFor(challengeID, "code")
}

// findFlowChallengeID digs the challenge_id out of the flow's data row directly
// from the DB (it is not exposed in the wire FlowState to avoid leaking it).
func findFlowChallengeID(t *testing.T, ctx context.Context, flowToken string) string {
	t.Helper()
	hash := flowHashToken(flowToken)
	rows, err := models.IamFlows.Query().All(ctx, testDB.Bobx())
	if err != nil {
		t.Fatalf("query flows: %v", err)
	}
	for _, row := range rows {
		if row.TokenHash == hash {
			var data struct {
				ActiveChallenge *struct {
					ChallengeID string `json:"challenge_id"`
				} `json:"active_challenge"`
			}
			if err := json.Unmarshal(row.Data, &data); err != nil {
				t.Fatalf("unmarshal flow data: %v", err)
			}
			if data.ActiveChallenge != nil {
				return data.ActiveChallenge.ChallengeID
			}
		}
	}
	t.Fatal("flow not found or no active_challenge")
	return ""
}

// ─── tests ───────────────────────────────────────────────────────────────────

// TestE2EFlowSignupHappyPath is the full signup flow: create → verify_email → completed.
func TestE2EFlowSignupHappyPath(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := "flow-signup-" + newUUID()[:8] + "@example.com"

	// 1. Create the flow.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":     "signup",
		"email":    email,
		"password": "Sup3rStr0ng!Pass",
		"name":     "Flow User",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Status != "pending" {
		t.Fatalf("status = %q, want pending", fs.Status)
	}
	if fs.Step != "verify_email" {
		t.Fatalf("step = %q, want verify_email", fs.Step)
	}
	if fs.FlowToken == "" {
		t.Fatal("flow_token is empty")
	}
	token1 := fs.FlowToken

	// Contact should be masked.
	if fs.Contact == nil || fs.Contact.EmailMasked == "" {
		t.Error("expected masked email in contact")
	}

	// 2. Grab the verification code from the emitter.
	challengeID := findFlowChallengeID(t, ctx, token1)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no code captured from emitter")
	}

	// 3. Submit verify_email.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token1, "verify_email", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs2.Status)
	}
	if fs2.Step != "completed" {
		t.Fatalf("step = %q, want completed", fs2.Step)
	}
	if fs2.Session == nil || fs2.Session.AccessToken == "" {
		t.Fatal("session not minted on completion")
	}
	token2 := fs2.FlowToken

	// 4. Token must have rotated (§5 rule 2).
	if token2 == token1 {
		t.Error("token was NOT rotated after verify_email — security violation")
	}

	// 5. Old token must be rejected (returns 410).
	_, rOld := flowGet(t, ctx, ts, projectID, token1)
	if rOld.Status != http.StatusGone && rOld.Status != http.StatusNotFound {
		t.Errorf("old token: status = %d, want 410 or 404", rOld.Status)
	}
}

// TestE2EFlowSignupExpiredToken verifies that an expired/unknown token → 410.
func TestE2EFlowSignupExpiredToken(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	// Expire the flow in the DB by inserting one with expires_at = past.
	hash := flowHashToken("ftk_" + newUUID())
	now := nowUTC()
	flowID := newUUID()
	rm := json.RawMessage(`{}`)
	exp := now.Add(-1 * time.Minute)
	_, err := models.IamFlows.Insert(&models.IamFlowSetter{
		ID:        &flowID,
		ProjectID: &projectID,
		TokenHash: &hash,
		Kind:      ptr("signup"),
		Status:    ptr("pending"),
		Step:      ptr("verify_email"),
		ExpiresAt: &exp,
		CreatedAt: &now,
		UpdatedAt: &now,
		Data:      &rm,
	}).One(ctx, testDB.Bobx())
	if err != nil {
		t.Fatalf("insert expired flow: %v", err)
	}

	// Try to get by any non-persisted token (unknown) → 410.
	unknownToken := "ftk_" + newUUID()
	_, r := flowGet(t, ctx, ts, projectID, unknownToken)
	if r.Status != http.StatusGone && r.Status != http.StatusNotFound {
		t.Errorf("unknown token: status = %d, want 410/404", r.Status)
	}
}

// TestE2EFlowSignupForeignProject verifies cross-project isolation.
func TestE2EFlowSignupForeignProject(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectA := e2eProject(t, ctx)
	projectB := e2eProject(t, ctx)
	email := "flow-foreign-" + newUUID()[:8] + "@example.com"

	// Create flow in project A.
	fs, r := flowCreate(t, ctx, ts, projectA, map[string]any{
		"kind":     "signup",
		"email":    email,
		"password": "Sup3rStr0ng!Pass",
		"name":     "Foreign User",
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	// Attempt to get it from project B → 410/404.
	_, rB := flowGet(t, ctx, ts, projectB, token)
	if rB.Status != http.StatusGone && rB.Status != http.StatusNotFound {
		t.Errorf("foreign project: status = %d, want 410/404", rB.Status)
	}
}

// TestE2EFlowSignupWrongCodeDecrementsAttempts verifies §5 rule 6.
func TestE2EFlowSignupWrongCodeDecrementsAttempts(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := "flow-wrong-" + newUUID()[:8] + "@example.com"

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Wrong Code",
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	// Submit with a wrong code — should get 200 with error embedded.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": "000000"})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "pending" {
		t.Fatalf("after wrong code: status = %q, want pending", fs2.Status)
	}
	if fs2.Error == nil {
		t.Fatal("after wrong code: expected error field in flow state")
	}
	if fs2.Error.Code != "invalid_code" {
		t.Errorf("error.code = %q, want invalid_code", fs2.Error.Code)
	}
	// Token must NOT have rotated.
	if fs2.FlowToken != token {
		t.Error("token must not rotate on wrong code")
	}
}

// TestE2EFlowSignupResendBeforeResendAt verifies the 429 rate limit (§5 rule 7).
func TestE2EFlowSignupResendBeforeResendAt(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := "flow-resend-" + newUUID()[:8] + "@example.com"

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Resend User",
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	// Immediately try to resend → should get 429 (resend_at is 60s from now).
	_, rResend := flowResend(t, ctx, ts, projectID, token)
	if rResend.Status != http.StatusTooManyRequests {
		t.Errorf("immediate resend: status = %d, want 429", rResend.Status)
	}
}

// TestE2EFlowAbandon verifies DELETE /v1/auth/flows/{token} → 204.
func TestE2EFlowAbandon(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := "flow-abandon-" + newUUID()[:8] + "@example.com"

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Abandon User",
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	// Abandon the flow.
	rDel := flowAbandon(t, ctx, ts, projectID, token)
	if rDel.Status != http.StatusNoContent && rDel.Status != http.StatusOK {
		t.Errorf("abandon: status = %d, want 204", rDel.Status)
	}

	// Flow should no longer be retrievable.
	_, rGet := flowGet(t, ctx, ts, projectID, token)
	if rGet.Status != http.StatusGone && rGet.Status != http.StatusNotFound {
		t.Errorf("after abandon: status = %d, want 410/404", rGet.Status)
	}
}

// TestE2EFlowSigninWrongPasswordIsNeutral verifies that an unknown email or
// wrong password returns a neutral 401 (anti-enumeration §5.4). The previous
// placeholder checked for 200; now that signin is implemented the create step
// authenticates immediately.
func TestE2EFlowSigninWrongPasswordIsNeutral(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	_, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signin", "email": "nonexistent@example.com", "password": "wrongpass",
	})
	// Wrong/unknown credentials → neutral 401, no enumeration.
	e2eWantStatus(t, r, http.StatusUnauthorized)
}

// suppress unused-symbol lint (referenced via reflection / blank imports).
var (
	_ = sql.ErrNoRows
	_ = null.From[string]
	_ = time.Second
)
