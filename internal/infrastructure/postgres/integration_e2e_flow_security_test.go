//go:build integration

package postgres

// integration_e2e_flow_security_test.go — adversarial HTTP e2e tests for the
// server-side resumable auth flow engine (§9).
//
// Each sub-test maps to a specific §5 security rule in
// docs/design/resumable-auth-flows.md.
//
// Coverage:
//  1. Token rotation kills the old token after every privilege transition (§5 rule 2).
//     - signup: verify_email → completed
//     - signin: mfa_required → completed
//     - recovery: set_password → completed
//  2. Tenant isolation: cross-project token → 410/404 (§5 rule 3).
//  3. Unknown / malformed token → 410/404 (§5 rule 3).
//  4. Expired flow → 410/404 (§5 rule 3); direct DB insert with past expires_at.
//  5. Anti-enumeration:
//     - recovery: unknown email returns same step=verify_email shape (§5 rule 4).
//     - signin: unknown email returns same outcome as known email + wrong password (§5 rule 4).
//  6. Attempts lockout: flowMaxAttempts wrong codes exhaust the challenge;
//     further submissions (including a correct code) are rejected (§5 rule 6).
//  7. Resend rate-limit: immediate resend → 429 (§5 rule 7).
//  8. Abandon: DELETE → 204; subsequent GET/submit/resend → 410/404 (§5 rule 8).
//  9. No session before second factor for MFA-enrolled signin (§5 rules 8, 9).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// ─── 1. Token rotation kills the old token ────────────────────────────────────

// TestE2EFlowSecurityTokenRotation_Signup verifies that after signup
// verify_email→completed the OLD token is immediately dead and the new
// (completed) token is also terminal (§5 rule 2).
func TestE2EFlowSecurityTokenRotation_Signup(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-rot-signup-%s@example.com", newUUID()[:8])

	// Create → verify_email.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Sec Rotation",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "verify_email" {
		t.Fatalf("step = %q, want verify_email", fs.Step)
	}
	oldToken := fs.FlowToken

	// Submit correct code → completed; token must rotate.
	challengeID := findFlowChallengeID(t, ctx, oldToken)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no code captured from emitter")
	}
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, oldToken, "verify_email", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs2.Status)
	}
	newToken := fs2.FlowToken
	if newToken == oldToken {
		t.Error("§5 rule 2 VIOLATION: token was NOT rotated after verify_email→completed")
	}
	if fs2.Session == nil || fs2.Session.AccessToken == "" {
		t.Fatal("§5 rule 8: session not minted at completion")
	}

	// OLD token must be dead on GET and submit.
	_, rOldGet := flowGet(t, ctx, ts, projectID, oldToken)
	if rOldGet.Status != http.StatusGone && rOldGet.Status != http.StatusNotFound {
		t.Errorf("§5 rule 2 VIOLATION: old token GET returned %d, want 410/404", rOldGet.Status)
	}
	_, rOldSub := flowSubmit(t, ctx, ts, projectID, oldToken, "verify_email", map[string]any{"code": "000000"})
	if rOldSub.Status != http.StatusGone && rOldSub.Status != http.StatusNotFound {
		t.Errorf("§5 rule 2 VIOLATION: old token submit returned %d, want 410/404", rOldSub.Status)
	}

	// NEW token is terminal (status=completed → status≠pending → flowLoad rejects it).
	// Submitting on it must not re-mint a session.
	_, rNewGet := flowGet(t, ctx, ts, projectID, newToken)
	if rNewGet.Status != http.StatusGone && rNewGet.Status != http.StatusNotFound {
		t.Errorf("§5 rule 8 VIOLATION: completed token GET returned %d, want 410/404", rNewGet.Status)
	}
	_, rNewSub := flowSubmit(t, ctx, ts, projectID, newToken, "verify_email", map[string]any{"code": "000000"})
	if rNewSub.Status != http.StatusGone && rNewSub.Status != http.StatusNotFound {
		t.Errorf("§5 rule 8 VIOLATION: completed token submit returned %d, want 410/404", rNewSub.Status)
	}
}

// TestE2EFlowSecurityTokenRotation_Signin verifies MFA mfa_required→completed
// rotates the token and kills the old one (§5 rule 2).
func TestE2EFlowSecurityTokenRotation_Signin(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-rot-signin-%s@example.com", newUUID()[:8])

	acct, _ := registerUser(t, ctx, projectID, email)
	e2eActiveEmailFactor(t, ctx, projectID, acct.ID, email)

	// Create → mfa_required (no session).
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signin", "email": email, "password": "Sup3rStr0ng!Pass",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "mfa_required" {
		t.Fatalf("step = %q, want mfa_required", fs.Step)
	}
	oldToken := fs.FlowToken

	// Submit correct MFA code → completed; token must rotate.
	challengeID := findFlowChallengeID(t, ctx, oldToken)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no MFA code captured from emitter")
	}
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, oldToken, "mfa", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs2.Status)
	}
	newToken := fs2.FlowToken
	if newToken == oldToken {
		t.Error("§5 rule 2 VIOLATION: token was NOT rotated after mfa_required→completed")
	}

	// OLD token dead.
	_, rOld := flowGet(t, ctx, ts, projectID, oldToken)
	if rOld.Status != http.StatusGone && rOld.Status != http.StatusNotFound {
		t.Errorf("§5 rule 2 VIOLATION: old signin token GET returned %d, want 410/404", rOld.Status)
	}

	// NEW token terminal.
	_, rNew := flowGet(t, ctx, ts, projectID, newToken)
	if rNew.Status != http.StatusGone && rNew.Status != http.StatusNotFound {
		t.Errorf("§5 rule 8 VIOLATION: completed signin token GET returned %d, want 410/404", rNew.Status)
	}
}

// TestE2EFlowSecurityTokenRotation_Recovery verifies that set_password→completed
// rotates the token and kills the old one (§5 rule 2).
func TestE2EFlowSecurityTokenRotation_Recovery(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-rot-recovery-%s@example.com", newUUID()[:8])

	registerUser(t, ctx, projectID, email)

	// Create → verify_email.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "recovery", "email": email,
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "verify_email" {
		t.Fatalf("step = %q, want verify_email", fs.Step)
	}
	token1 := fs.FlowToken

	// Verify email (token should NOT rotate here — rotates at set_password).
	challengeID := findFlowChallengeID(t, ctx, token1)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no OTP code captured from emitter")
	}
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token1, "verify_email", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Step != "set_password" {
		t.Fatalf("step = %q, want set_password", fs2.Step)
	}

	// set_password → completed: token MUST rotate.
	fs3, r3 := flowSubmit(t, ctx, ts, projectID, fs2.FlowToken, "set_password", map[string]any{"password": "N3wStr0ng!Pass99"})
	e2eWantStatus(t, r3, http.StatusOK)
	if fs3.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs3.Status)
	}
	token3 := fs3.FlowToken
	if token3 == token1 {
		t.Error("§5 rule 2 VIOLATION: token was NOT rotated after recovery set_password→completed")
	}
	if fs3.Session == nil || fs3.Session.AccessToken == "" {
		t.Fatal("§5 rule 8: session not minted at completion")
	}

	// Old token (token1) must be dead after rotation.
	_, rOld := flowGet(t, ctx, ts, projectID, token1)
	if rOld.Status != http.StatusGone && rOld.Status != http.StatusNotFound {
		t.Errorf("§5 rule 2 VIOLATION: old recovery token GET returned %d, want 410/404", rOld.Status)
	}

	// New completed token is also terminal.
	_, rNew := flowGet(t, ctx, ts, projectID, token3)
	if rNew.Status != http.StatusGone && rNew.Status != http.StatusNotFound {
		t.Errorf("§5 rule 8 VIOLATION: completed recovery token GET returned %d, want 410/404", rNew.Status)
	}
}

// ─── 2. Tenant isolation ─────────────────────────────────────────────────────

// TestE2EFlowSecurityTenantIsolation verifies that a flow_token from project A
// is rejected by all endpoints when X-Client-Id is project B (§5 rule 3).
func TestE2EFlowSecurityTenantIsolation(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectA := e2eProject(t, ctx)
	projectB := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-tenant-%s@example.com", newUUID()[:8])

	// Create a live flow in project A.
	fs, r := flowCreate(t, ctx, ts, projectA, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "TenantA",
	})
	e2eWantStatus(t, r, http.StatusOK)
	tokenA := fs.FlowToken

	// GET with project B → must be 410/404 (tenant boundary, §5 rule 3).
	_, rGet := flowGet(t, ctx, ts, projectB, tokenA)
	if rGet.Status != http.StatusGone && rGet.Status != http.StatusNotFound {
		t.Errorf("§5 rule 3 VIOLATION: cross-tenant GET returned %d, want 410/404; body: %s",
			rGet.Status, rGet.Body)
	}

	// Submit with project B → must also be rejected.
	_, rSub := flowSubmit(t, ctx, ts, projectB, tokenA, "verify_email", map[string]any{"code": "000000"})
	if rSub.Status != http.StatusGone && rSub.Status != http.StatusNotFound {
		t.Errorf("§5 rule 3 VIOLATION: cross-tenant submit returned %d, want 410/404; body: %s",
			rSub.Status, rSub.Body)
	}

	// Resend with project B → must be rejected.
	_, rResend := flowResend(t, ctx, ts, projectB, tokenA)
	if rResend.Status != http.StatusGone && rResend.Status != http.StatusNotFound {
		t.Errorf("§5 rule 3 VIOLATION: cross-tenant resend returned %d, want 410/404; body: %s",
			rResend.Status, rResend.Body)
	}
}

// ─── 3. Unknown / malformed token ────────────────────────────────────────────

// TestE2EFlowSecurityUnknownToken verifies that unknown or malformed flow_tokens
// return 410/404 from GET and submit (§5 rule 3).
func TestE2EFlowSecurityUnknownToken(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	type tc struct {
		name  string
		token string
	}
	cases := []tc{
		{"valid_prefix_unknown_suffix", "ftk_" + newUUID()},
		{"no_prefix", "not-a-flow-token-" + newUUID()[:8]},
		{"prefix_only", "ftk_"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			_, rGet := flowGet(t, ctx, ts, projectID, c.token)
			if rGet.Status != http.StatusGone && rGet.Status != http.StatusNotFound {
				t.Errorf("§5 rule 3: unknown token GET returned %d, want 410/404; body: %s",
					rGet.Status, rGet.Body)
			}
			_, rSub := flowSubmit(t, ctx, ts, projectID, c.token, "verify_email", map[string]any{"code": "x"})
			if rSub.Status != http.StatusGone && rSub.Status != http.StatusNotFound {
				t.Errorf("§5 rule 3: unknown token submit returned %d, want 410/404; body: %s",
					rSub.Status, rSub.Body)
			}
		})
	}
}

// ─── 4. Expired flow ─────────────────────────────────────────────────────────

// TestE2EFlowSecurityExpiredFlow verifies that a flow with expires_at in the
// past is treated as not-found (§5 rule 3). Direct DB insert is used to create
// an expired row without waiting 30 minutes (flowTTL).
//
// Note: flowTTL=30m cannot be shortened via the public API. Sleeping 30m is
// impractical in CI. We insert a row directly with expires_at = now-1h, which
// is the same technique used by TestE2EFlowSignupExpiredToken.
func TestE2EFlowSecurityExpiredFlow(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	// Insert a synthetic expired row with expires_at = now - 1h.
	expiredToken := "ftk_" + newUUID()
	expiredHash := flowHashToken(expiredToken)
	now := nowUTC()
	exp := now.Add(-time.Hour)
	flowID := newUUID()
	rm := json.RawMessage(`{}`)
	_, err := models.IamFlows.Insert(&models.IamFlowSetter{
		ID:        &flowID,
		ProjectID: &projectID,
		TokenHash: &expiredHash,
		Kind:      ptr("signup"),
		Status:    ptr("pending"),
		Step:      ptr("verify_email"),
		ExpiresAt: &exp,
		CreatedAt: &now,
		UpdatedAt: &now,
		Data:      &rm,
	}).One(ctx, testDB.Bobx())
	if err != nil {
		t.Fatalf("insert expired flow row: %v", err)
	}

	// All three access verbs must return 410/404 for the expired token.
	_, rGet := flowGet(t, ctx, ts, projectID, expiredToken)
	if rGet.Status != http.StatusGone && rGet.Status != http.StatusNotFound {
		t.Errorf("§5 rule 3: expired token GET returned %d, want 410/404", rGet.Status)
	}

	_, rSub := flowSubmit(t, ctx, ts, projectID, expiredToken, "verify_email", map[string]any{"code": "000000"})
	if rSub.Status != http.StatusGone && rSub.Status != http.StatusNotFound {
		t.Errorf("§5 rule 3: expired token submit returned %d, want 410/404", rSub.Status)
	}

	_, rResend := flowResend(t, ctx, ts, projectID, expiredToken)
	if rResend.Status != http.StatusGone && rResend.Status != http.StatusNotFound {
		t.Errorf("§5 rule 3: expired token resend returned %d, want 410/404", rResend.Status)
	}
}

// ─── 5. Anti-enumeration ──────────────────────────────────────────────────────

// TestE2EFlowSecurityAntiEnumeration_Recovery verifies §5 rule 4: creating a
// recovery flow for an unknown email returns the SAME FlowState shape as for a
// known user (status=pending, step=verify_email, masked contact, no error).
// Any code submitted against the ghost flow returns invalid_code — never a
// 404/410 or an account-existence hint.
func TestE2EFlowSecurityAntiEnumeration_Recovery(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	realEmail := fmt.Sprintf("sec-ae-real-%s@example.com", newUUID()[:8])
	unknownEmail := fmt.Sprintf("sec-ae-ghost-%s@example.com", newUUID()[:8])

	// Register a real user to have a reference shape.
	registerUser(t, ctx, projectID, realEmail)

	// Create recovery for known user.
	fsReal, rReal := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "recovery", "email": realEmail,
	})
	e2eWantStatus(t, rReal, http.StatusOK)

	// Create recovery for unknown user.
	fsGhost, rGhost := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "recovery", "email": unknownEmail,
	})
	e2eWantStatus(t, rGhost, http.StatusOK)

	// The ghost shape must match the real shape (§5 rule 4).
	if fsGhost.Status != "pending" {
		t.Errorf("§5.4 VIOLATION: unknown email recovery status = %q, want pending", fsGhost.Status)
	}
	if fsGhost.Step != "verify_email" {
		t.Errorf("§5.4 VIOLATION: unknown email recovery step = %q, want verify_email", fsGhost.Step)
	}
	if fsGhost.FlowToken == "" {
		t.Error("§5.4 VIOLATION: unknown email recovery: flow_token is empty")
	}
	if fsGhost.Error != nil {
		t.Errorf("§5.4 VIOLATION: create MUST NOT return error for unknown email (leaks existence): %+v", fsGhost.Error)
	}
	if fsGhost.Contact == nil || fsGhost.Contact.EmailMasked == "" {
		t.Error("§5.4 VIOLATION: unknown email recovery: masked contact must be present")
	}
	// Structural match between real and ghost.
	if fsReal.Status != fsGhost.Status {
		t.Errorf("§5.4 VIOLATION: status mismatch real=%q ghost=%q", fsReal.Status, fsGhost.Status)
	}
	if fsReal.Step != fsGhost.Step {
		t.Errorf("§5.4 VIOLATION: step mismatch real=%q ghost=%q", fsReal.Step, fsGhost.Step)
	}

	// Submitting any code to the ghost flow must return 200 with invalid_code —
	// NOT 404/410 or an existence signal.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, fsGhost.FlowToken, "verify_email", map[string]any{"code": "000000"})
	e2eWantStatus(t, r2, http.StatusOK) // 200 with embedded error — not 404/410
	if fs2.Status != "pending" {
		t.Errorf("§5.4: ghost flow wrong code: status = %q, want pending", fs2.Status)
	}
	if fs2.Error == nil {
		t.Fatal("§5.4: ghost flow wrong code: expected error field in FlowState")
	}
	if fs2.Error.Code != "invalid_code" {
		t.Errorf("§5.4: ghost flow wrong code: error.code = %q, want invalid_code", fs2.Error.Code)
	}
}

// TestE2EFlowSecurityAntiEnumeration_Signin verifies §5 rule 4: creating a
// signin flow for an unknown email must return the SAME HTTP status and error
// code as a known email with a wrong password — the response must not reveal
// which case it is.
func TestE2EFlowSecurityAntiEnumeration_Signin(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	realEmail := fmt.Sprintf("sec-ae-signin-real-%s@example.com", newUUID()[:8])
	unknownEmail := fmt.Sprintf("sec-ae-signin-ghost-%s@example.com", newUUID()[:8])

	// Register a real user.
	registerUser(t, ctx, projectID, realEmail)

	// Known email + wrong password.
	_, rKnownWrong := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signin", "email": realEmail, "password": "WrongPassword!",
	})

	// Unknown email + any password.
	_, rUnknown := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signin", "email": unknownEmail, "password": "AnyPassword!",
	})

	// HTTP status must be identical — different codes leak account existence (§5 rule 4).
	if rKnownWrong.Status != rUnknown.Status {
		t.Errorf("§5.4 VIOLATION: known-wrong status=%d vs unknown status=%d — different responses leak account existence",
			rKnownWrong.Status, rUnknown.Status)
	}

	// Neither should return 200 OK (credentials rejected).
	if rKnownWrong.Status == http.StatusOK {
		t.Error("§5.4: wrong password for known email must not return 200")
	}
	if rUnknown.Status == http.StatusOK {
		t.Error("§5.4: unknown email must not return 200")
	}
}

// ─── 6. Attempts lockout ─────────────────────────────────────────────────────

// TestE2EFlowSecurityAttemptsLockout verifies §5 rule 6: submitting
// flowMaxAttempts wrong codes exhausts the challenge. Each wrong attempt:
//   - keeps status=pending
//   - embeds a stable machine code in error.code
//   - decrements attempts_left
//
// After exhaustion, further submissions (including a correct code) are rejected
// without advancing the flow.
func TestE2EFlowSecurityAttemptsLockout(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-lockout-%s@example.com", newUUID()[:8])

	// Signup auto-issues a verify_email challenge on create.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Lockout User",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "verify_email" {
		t.Fatalf("step = %q, want verify_email", fs.Step)
	}
	token := fs.FlowToken
	challengeID := findFlowChallengeID(t, ctx, token)

	prevLeft := flowMaxAttempts
	for i := 1; i <= flowMaxAttempts; i++ {
		// Use distinct high codes to avoid accidental collision with the real 6-digit code.
		wrongCode := fmt.Sprintf("%06d", 900000+i)
		fsi, ri := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": wrongCode})
		e2eWantStatus(t, ri, http.StatusOK)

		if fsi.Status != "pending" {
			t.Errorf("attempt %d: status = %q, want pending (§5 rule 6)", i, fsi.Status)
		}
		if fsi.Error == nil {
			t.Errorf("attempt %d: expected error field (§5 rule 6)", i)
		} else if fsi.Error.Code != "invalid_code" {
			t.Errorf("attempt %d: error.code = %q, want stable code invalid_code (§5 rule 6)", i, fsi.Error.Code)
		}
		if fsi.FlowToken != token {
			t.Errorf("attempt %d: token rotated on wrong code — must not rotate (§5 rule 6)", i)
		}
		if fsi.Challenge != nil {
			wantLeft := prevLeft - 1
			if fsi.Challenge.AttemptsLeft != wantLeft {
				t.Errorf("attempt %d: attempts_left = %d, want %d (§5 rule 6)",
					i, fsi.Challenge.AttemptsLeft, wantLeft)
			}
			prevLeft = fsi.Challenge.AttemptsLeft
		}
	}

	// After exhaustion: a subsequent wrong code must not advance (§5 rule 6).
	fsExh, _ := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": "999999"})
	if fsExh.Status == "completed" {
		t.Error("§5 rule 6 VIOLATION: flow completed after challenge exhaustion")
	}
	// Must carry an error (either embedded or HTTP-level).
	if fsExh.Status == "pending" && fsExh.Error == nil {
		t.Error("§5 rule 6: expected error after challenge exhaustion, got pending with no error")
	}

	// After exhaustion: a CORRECT code must also be rejected (challenge is dead/consumed).
	correctCode := captureCode(challengeID)
	if correctCode != "" {
		fsCorr, _ := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": correctCode})
		if fsCorr.Status == "completed" {
			t.Error("§5 rule 6 VIOLATION: correct code accepted after challenge exhaustion")
		}
	}
}

// ─── 7. Resend rate-limit ────────────────────────────────────────────────────

// TestE2EFlowSecurityResendRateLimit verifies §5 rule 7: an immediate resend
// before resend_at → 429. The response must expose resend_at so the client
// knows when to retry.
func TestE2EFlowSecurityResendRateLimit(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-resend-%s@example.com", newUUID()[:8])

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Resend Sec",
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	// Immediately resend — resend_at is flowResendCooloff (60s) in the future.
	_, rResend := flowResend(t, ctx, ts, projectID, token)
	if rResend.Status != http.StatusTooManyRequests {
		t.Errorf("§5 rule 7: immediate resend returned %d, want 429; body: %s",
			rResend.Status, rResend.Body)
	}

	// A second immediate resend also 429, not 500.
	_, rResend2 := flowResend(t, ctx, ts, projectID, token)
	if rResend2.Status != http.StatusTooManyRequests {
		t.Errorf("§5 rule 7: second immediate resend returned %d, want 429", rResend2.Status)
	}
}

// ─── 8. Abandon ──────────────────────────────────────────────────────────────

// TestE2EFlowSecurityAbandon verifies §5 rules 3/8: DELETE → 204 (or 200);
// subsequent GET, submit, and resend on the abandoned token → 410/404.
// A second DELETE must be idempotent (not 500).
func TestE2EFlowSecurityAbandon(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-abandon-%s@example.com", newUUID()[:8])

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Abandon Sec",
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	// DELETE the flow → 204.
	rDel := flowAbandon(t, ctx, ts, projectID, token)
	if rDel.Status != http.StatusNoContent && rDel.Status != http.StatusOK {
		t.Fatalf("abandon: DELETE returned %d, want 204; body: %s", rDel.Status, rDel.Body)
	}

	// GET on abandoned token → 410/404.
	_, rGet := flowGet(t, ctx, ts, projectID, token)
	if rGet.Status != http.StatusGone && rGet.Status != http.StatusNotFound {
		t.Errorf("§5 rule 8: after abandon GET returned %d, want 410/404", rGet.Status)
	}

	// Submit on abandoned token → 410/404.
	_, rSub := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": "000000"})
	if rSub.Status != http.StatusGone && rSub.Status != http.StatusNotFound {
		t.Errorf("§5 rule 8: after abandon submit returned %d, want 410/404", rSub.Status)
	}

	// Resend on abandoned token → 410/404.
	_, rResend := flowResend(t, ctx, ts, projectID, token)
	if rResend.Status != http.StatusGone && rResend.Status != http.StatusNotFound {
		t.Errorf("§5 rule 8: after abandon resend returned %d, want 410/404", rResend.Status)
	}

	// Second DELETE must be idempotent (not 500).
	rDel2 := flowAbandon(t, ctx, ts, projectID, token)
	if rDel2.Status == http.StatusInternalServerError {
		t.Errorf("§5 rule 8: idempotent second abandon returned 500; body: %s", rDel2.Body)
	}
}

// ─── 9. No session before second factor (signin with MFA) ───────────────────

// TestE2EFlowSecurityNoSessionBeforeMFA verifies §5 rules 8 + 9: for an
// MFA-enrolled account the create response and any pre-verify GET carry no
// session. A wrong MFA code also carries no session. The session appears ONLY
// when status=completed (after the second factor is verified).
func TestE2EFlowSecurityNoSessionBeforeMFA(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("sec-nomfa-session-%s@example.com", newUUID()[:8])

	acct, _ := registerUser(t, ctx, projectID, email)
	e2eActiveEmailFactor(t, ctx, projectID, acct.ID, email)

	// 1. Create → mfa_required. Session must be absent.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind": "signin", "email": email, "password": "Sup3rStr0ng!Pass",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Status != "pending" {
		t.Fatalf("status = %q, want pending", fs.Status)
	}
	if fs.Step != "mfa_required" {
		t.Fatalf("step = %q, want mfa_required", fs.Step)
	}
	if fs.Session != nil {
		t.Error("§5 rule 8/9 VIOLATION: session present in create response at mfa_required")
	}
	token := fs.FlowToken

	// 2. GET at mfa_required. Session must still be absent.
	fsGet, rGet := flowGet(t, ctx, ts, projectID, token)
	e2eWantStatus(t, rGet, http.StatusOK)
	if fsGet.Session != nil {
		t.Error("§5 rule 8/9 VIOLATION: session present in GET response at mfa_required")
	}
	if fsGet.Step != "mfa_required" {
		t.Errorf("GET step = %q, want mfa_required", fsGet.Step)
	}

	// 3. Wrong MFA code → session still absent, status stays pending.
	fsBad, rBad := flowSubmit(t, ctx, ts, projectID, token, "mfa", map[string]any{"code": "000000"})
	e2eWantStatus(t, rBad, http.StatusOK)
	if fsBad.Session != nil {
		t.Error("§5 rule 8/9 VIOLATION: session present after wrong MFA code")
	}
	if fsBad.Status != "pending" {
		t.Errorf("after wrong MFA code: status = %q, want pending", fsBad.Status)
	}
	if fsBad.Error == nil || fsBad.Error.Code != "invalid_code" {
		t.Errorf("after wrong MFA code: expected error.code=invalid_code, got %v", fsBad.Error)
	}

	// 4. Correct MFA code → session appears NOW, status=completed.
	challengeID := findFlowChallengeID(t, ctx, token)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no MFA code captured from emitter")
	}
	fsOK, rOK := flowSubmit(t, ctx, ts, projectID, token, "mfa", map[string]any{"code": code})
	e2eWantStatus(t, rOK, http.StatusOK)
	if fsOK.Status != "completed" {
		t.Fatalf("after correct MFA: status = %q, want completed", fsOK.Status)
	}
	if fsOK.Session == nil || fsOK.Session.AccessToken == "" {
		t.Fatal("§5 rule 8: session NOT present after MFA verify — should be minted at completion")
	}
}
