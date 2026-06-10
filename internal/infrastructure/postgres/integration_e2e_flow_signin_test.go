//go:build integration

package postgres

// integration_e2e_flow_signin_test.go — HTTP e2e tests for the SIGNIN kind of
// the server-side resumable auth flow engine.
//
// Coverage:
//   (1) No-MFA happy path: create returns status=completed + session immediately.
//   (2) MFA happy path: create returns step=mfa_required + flow_token;
//       submit mfa with captured code → completed + session;
//       old token is rejected after rotation.
//   (3) Wrong password → neutral 401/invalid_credentials (anti-enumeration §5.4).
//   (4) Wrong MFA code → attemptsLeft decremented, status stays pending.

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// TestE2EFlowSigninNoMFA verifies the no-MFA path: create immediately returns
// status=completed with a session (single round-trip).
func TestE2EFlowSigninNoMFA(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("flow-signin-nomfa-%s@example.com", newUUID()[:8])

	// Seed the account (no MFA factor enrolled).
	registerUser(t, ctx, projectID, email)

	// Create the signin flow.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":     "signin",
		"email":    email,
		"password": "Sup3rStr0ng!Pass",
	})
	e2eWantStatus(t, r, http.StatusOK)

	if fs.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs.Status)
	}
	if fs.Step != "completed" {
		t.Fatalf("step = %q, want completed", fs.Step)
	}
	if fs.FlowToken == "" {
		t.Fatal("flow_token is empty")
	}
	if fs.Session == nil || fs.Session.AccessToken == "" {
		t.Fatal("session not returned on no-MFA signin completion")
	}
}

// TestE2EFlowSigninWithMFA verifies the two-step MFA path:
//  1. create → step=mfa_required (no session yet, challenge issued)
//  2. submit mfa with captured code → completed + session
//  3. old token is rejected after rotation (§5 rule 2)
func TestE2EFlowSigninWithMFA(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("flow-signin-mfa-%s@example.com", newUUID()[:8])

	// Seed account + active email MFA factor.
	acct, _ := registerUser(t, ctx, projectID, email)
	e2eActiveEmailFactor(t, ctx, projectID, acct.ID, email)

	// 1. Create the signin flow — should gate at mfa_required.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":     "signin",
		"email":    email,
		"password": "Sup3rStr0ng!Pass",
	})
	e2eWantStatus(t, r, http.StatusOK)

	if fs.Status != "pending" {
		t.Fatalf("status = %q, want pending", fs.Status)
	}
	if fs.Step != "mfa_required" {
		t.Fatalf("step = %q, want mfa_required", fs.Step)
	}
	if fs.FlowToken == "" {
		t.Fatal("flow_token is empty")
	}
	if fs.Session != nil {
		t.Error("session must NOT be set at mfa_required step")
	}
	token1 := fs.FlowToken

	// 2. Retrieve the challenge id and the emitted code.
	challengeID := findFlowChallengeID(t, ctx, token1)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no MFA code captured from emitter")
	}

	// 3. Submit the MFA code.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token1, "mfa", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)

	if fs2.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs2.Status)
	}
	if fs2.Step != "completed" {
		t.Fatalf("step = %q, want completed", fs2.Step)
	}
	if fs2.Session == nil || fs2.Session.AccessToken == "" {
		t.Fatal("session not minted after MFA verification")
	}
	token2 := fs2.FlowToken

	// 4. Token must have rotated (§5 rule 2).
	if token2 == token1 {
		t.Error("token was NOT rotated after MFA verify — security violation")
	}

	// 5. Old token must be rejected (returns 410).
	_, rOld := flowGet(t, ctx, ts, projectID, token1)
	if rOld.Status != http.StatusGone && rOld.Status != http.StatusNotFound {
		t.Errorf("old token: status = %d, want 410 or 404", rOld.Status)
	}
}

// TestE2EFlowSigninWrongPassword verifies that an invalid password (or unknown
// email) returns a neutral 401 — no enumeration of account existence (§5.4).
func TestE2EFlowSigninWrongPassword(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	// Neither a registered account nor a correct password.
	_, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":     "signin",
		"email":    "no-such-user-" + newUUID()[:8] + "@example.com",
		"password": "WrongPassword!",
	})
	// Must return 401 with no additional enumeration signal.
	e2eWantStatus(t, r, http.StatusUnauthorized)
}

// TestE2EFlowSigninWrongMFACodeDecrementsAttempts verifies §5 rule 6: a wrong
// MFA code decrements AttemptsLeft and leaves the flow pending.
func TestE2EFlowSigninWrongMFACodeDecrementsAttempts(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("flow-signin-badmfa-%s@example.com", newUUID()[:8])

	acct, _ := registerUser(t, ctx, projectID, email)
	e2eActiveEmailFactor(t, ctx, projectID, acct.ID, email)

	// Get to mfa_required.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":     "signin",
		"email":    email,
		"password": "Sup3rStr0ng!Pass",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "mfa_required" {
		t.Fatalf("step = %q, want mfa_required", fs.Step)
	}
	token := fs.FlowToken

	// Submit a wrong code.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token, "mfa", map[string]any{"code": "000000"})
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
	// Token must NOT have rotated on a wrong code.
	if fs2.FlowToken != token {
		t.Error("token must not rotate on wrong MFA code")
	}
	// Attempts must be decremented.
	if fs2.Challenge == nil {
		t.Fatal("expected challenge in flow state after wrong code")
	}
	if fs2.Challenge.AttemptsLeft >= 5 {
		t.Errorf("attempts_left = %d, expected decrement from 5", fs2.Challenge.AttemptsLeft)
	}
}
