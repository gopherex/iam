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

// ─── multichannel: phone_otp ──────────────────────────────────────────────────

// TestE2EFlowSigninPhoneOTP drives the phone_otp method end-to-end:
//
//	create{method:phone_otp,phone} → step=verify_phone (no session)
//	submit{verify_otp,code}        → completed + session, token rotated.
func TestE2EFlowSigninPhoneOTP(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	phone := fmt.Sprintf("+1415556%04d", 1000+len(t.Name())%9000)

	// 1. Create the signin flow on the phone_otp channel.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":   "signin",
		"method": "phone_otp",
		"phone":  phone,
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Status != "pending" {
		t.Fatalf("status = %q, want pending", fs.Status)
	}
	if fs.Step != "verify_phone" {
		t.Fatalf("step = %q, want verify_phone", fs.Step)
	}
	if fs.Session != nil {
		t.Error("session must NOT be set at verify_phone step")
	}
	token1 := fs.FlowToken

	// 2. Recover the challenge id from the flow row and the emitted code.
	challengeID := findFlowChallengeID(t, ctx, token1)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no OTP code captured from emitter")
	}

	// 3. Submit the code with the verify_otp action.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token1, "verify_otp", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs2.Status)
	}
	if fs2.Session == nil || fs2.Session.AccessToken == "" {
		t.Fatal("session not minted after phone OTP verify")
	}
	if fs2.FlowToken == token1 {
		t.Error("token was NOT rotated after phone OTP verify")
	}
}

// TestE2EFlowSigninPhoneOTPNoProvider verifies the SMS-provider pre-flight: a
// project with no enabled sms provider fails fast on create (no flow row minted).
func TestE2EFlowSigninPhoneOTPNoProvider(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	// Deliberately do NOT enable an sms provider.
	phone := fmt.Sprintf("+1415557%04d", 1000+len(t.Name())%9000)

	_, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":   "signin",
		"method": "phone_otp",
		"phone":  phone,
	})
	if r.Status == http.StatusOK {
		t.Fatalf("expected non-200 when no sms provider enabled, got 200: %s", r.Body)
	}
}

// TestE2EFlowSigninPhoneOTPWrongCode verifies wrong-code handling on phone_otp:
// attempts decremented, flow stays pending, no rotation.
func TestE2EFlowSigninPhoneOTPWrongCode(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	phone := fmt.Sprintf("+1415558%04d", 1000+len(t.Name())%9000)

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":   "signin",
		"method": "phone_otp",
		"phone":  phone,
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "verify_phone" {
		t.Fatalf("step = %q, want verify_phone", fs.Step)
	}
	token := fs.FlowToken

	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token, "verify_otp", map[string]any{"code": "000000"})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "pending" {
		t.Fatalf("after wrong code: status = %q, want pending", fs2.Status)
	}
	if fs2.Error == nil || fs2.Error.Code != "invalid_code" {
		t.Fatalf("after wrong code: expected error.code=invalid_code, got %+v", fs2.Error)
	}
	if fs2.FlowToken != token {
		t.Error("token must not rotate on wrong OTP code")
	}
	if fs2.Challenge == nil || fs2.Challenge.AttemptsLeft >= 5 {
		t.Errorf("attempts_left not decremented: %+v", fs2.Challenge)
	}
}

// ─── multichannel: magic_link ─────────────────────────────────────────────────

// TestE2EFlowSigninMagicLink drives the magic_link method end-to-end:
//
//	create{method:magic_link,email} → step=verify_email (no session)
//	submit{verify_email,token}      → completed + session, token rotated.
func TestE2EFlowSigninMagicLink(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("flow-signin-magic-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	// 1. Create the signin flow on the magic_link channel.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":   "signin",
		"method": "magic_link",
		"email":  email,
		"locale": "ru",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Status != "pending" {
		t.Fatalf("status = %q, want pending", fs.Status)
	}
	if fs.Step != "verify_email" {
		t.Fatalf("step = %q, want verify_email", fs.Step)
	}
	if fs.Session != nil {
		t.Error("session must NOT be set at verify_email step")
	}
	token1 := fs.FlowToken

	// 2. Recover the challenge id + opaque magic-link token from the emitter.
	challengeID := findFlowChallengeID(t, ctx, token1)
	mlToken := e2eEmitter.payloadFor(challengeID, "token")
	if mlToken == "" {
		t.Fatalf("magic-link token not captured for challenge %s", challengeID)
	}
	if got := e2eEmitter.payloadFor(challengeID, "locale"); got != "ru" {
		t.Fatalf("magic-link locale = %q, want ru", got)
	}

	// 3. Submit the token with the verify_email action.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token1, "verify_email", map[string]any{"token": mlToken})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs2.Status)
	}
	if fs2.Session == nil || fs2.Session.AccessToken == "" {
		t.Fatal("session not minted after magic-link verify")
	}
	if fs2.FlowToken == token1 {
		t.Error("token was NOT rotated after magic-link verify")
	}
}

// TestE2EFlowSigninMagicLinkBadToken verifies an invalid magic-link token keeps
// the flow pending with an invalid_code error (no rotation).
func TestE2EFlowSigninMagicLinkBadToken(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("flow-signin-magicbad-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":   "signin",
		"method": "magic_link",
		"email":  email,
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"token": "ml_bogus_token"})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "pending" {
		t.Fatalf("after bad token: status = %q, want pending", fs2.Status)
	}
	if fs2.Error == nil || fs2.Error.Code != "invalid_code" {
		t.Fatalf("after bad token: expected error.code=invalid_code, got %+v", fs2.Error)
	}
	if fs2.FlowToken != token {
		t.Error("token must not rotate on bad magic-link token")
	}
}

// ─── multichannel: switch_method ──────────────────────────────────────────────

// TestE2EFlowSigninSwitchMethod starts on magic_link and switches to phone_otp
// mid-flow without rotating the token, then completes via the new channel.
func TestE2EFlowSigninSwitchMethod(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	email := fmt.Sprintf("flow-signin-switch-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)
	phone := fmt.Sprintf("+1415559%04d", 1000+len(t.Name())%9000)

	// 1. Start on magic_link → verify_email.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":   "signin",
		"method": "magic_link",
		"email":  email,
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "verify_email" {
		t.Fatalf("step = %q, want verify_email", fs.Step)
	}
	token := fs.FlowToken

	// 2. Switch to phone_otp in place — token must NOT rotate.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token, "switch_method", map[string]any{
		"method": "phone_otp",
		"phone":  phone,
	})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Step != "verify_phone" {
		t.Fatalf("after switch: step = %q, want verify_phone", fs2.Step)
	}
	if fs2.FlowToken != token {
		t.Error("token must NOT rotate on switch_method")
	}

	// 3. Complete via the new sms channel.
	challengeID := findFlowChallengeID(t, ctx, token)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no OTP code captured after switch_method")
	}
	fs3, r3 := flowSubmit(t, ctx, ts, projectID, token, "verify_otp", map[string]any{"code": code})
	e2eWantStatus(t, r3, http.StatusOK)
	if fs3.Status != "completed" {
		t.Fatalf("status = %q, want completed", fs3.Status)
	}
	if fs3.Session == nil || fs3.Session.AccessToken == "" {
		t.Fatal("session not minted after switched phone OTP verify")
	}
}
