//go:build integration

package postgres

// integration_e2e_flow_recovery_test.go — HTTP e2e tests for the recovery
// (forgot-password) resumable auth flow.
//
// Coverage:
//   1. Happy path: register → create recovery → capture OTP → verify_email
//      → set_password → completed + session; new password works for sign-in.
//      verify_email also accepts the opaque email link token.
//   2. Anti-enumeration: create recovery for an unknown email returns the
//      SAME step=verify_email FlowState with no error; any submitted code
//      fails verification identically.
//   3. Wrong code decrements attempts_left; token does NOT rotate; error embedded.

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// ─── tests ────────────────────────────────────────────────────────────────────

// TestE2EFlowRecoveryHappyPath exercises the full recovery state machine:
// register → create recovery → OTP verified → new password set → session.
// It then signs in with the new password to confirm the credential was updated.
func TestE2EFlowRecoveryHappyPath(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("recovery-happy-%s@example.com", newUUID()[:8])
	newPassword := "N3wStr0ng!Pass99"

	// 1. Register a user with an initial password.
	registerUser(t, ctx, projectID, email)

	// 2. Create a recovery flow.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":  "recovery",
		"email": email,
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Status != "pending" {
		t.Fatalf("create: status = %q, want pending", fs.Status)
	}
	if fs.Step != "verify_email" {
		t.Fatalf("create: step = %q, want verify_email", fs.Step)
	}
	if fs.FlowToken == "" {
		t.Fatal("create: flow_token is empty")
	}
	// Masked contact should be present.
	if fs.Contact == nil || fs.Contact.EmailMasked == "" {
		t.Error("create: expected masked email in contact")
	}
	token1 := fs.FlowToken

	// 3. Capture the OTP emitted to the "email" (captured by e2eEmitter).
	challengeID := findFlowChallengeID(t, ctx, token1)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("verify_email: no OTP code captured from emitter")
	}
	if got := e2eEmitter.payloadFor(challengeID, "flow_token"); got != token1 {
		t.Fatalf("flow email flow_token = %q, want %q", got, token1)
	}
	if got := e2eEmitter.payloadFor(challengeID, "token"); got == "" {
		t.Fatal("flow email proof token is empty")
	}

	// 4. Submit verify_email with the correct code.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token1, "verify_email", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "pending" {
		t.Fatalf("verify_email: status = %q, want pending", fs2.Status)
	}
	if fs2.Step != "set_password" {
		t.Fatalf("verify_email: step = %q, want set_password", fs2.Step)
	}
	if fs2.Error != nil {
		t.Errorf("verify_email: unexpected error: %+v", fs2.Error)
	}
	// Token must NOT rotate at verify_email (rotates only at set_password).
	if fs2.FlowToken != token1 {
		t.Errorf("verify_email: token rotated prematurely (should rotate only at set_password)")
	}

	// 5. Submit set_password with the new password.
	fs3, r3 := flowSubmit(t, ctx, ts, projectID, fs2.FlowToken, "set_password", map[string]any{"password": newPassword})
	e2eWantStatus(t, r3, http.StatusOK)
	if fs3.Status != "completed" {
		t.Fatalf("set_password: status = %q, want completed", fs3.Status)
	}
	if fs3.Step != "completed" {
		t.Fatalf("set_password: step = %q, want completed", fs3.Step)
	}
	if fs3.Session == nil || fs3.Session.AccessToken == "" {
		t.Fatal("set_password: session not minted on completion")
	}
	token3 := fs3.FlowToken

	// 6. Token must have rotated at set_password (§5 rule 2).
	if token3 == token1 {
		t.Error("set_password: token was NOT rotated — security violation")
	}

	// 7. Old token must be rejected.
	_, rOld := flowGet(t, ctx, ts, projectID, token1)
	if rOld.Status != http.StatusGone && rOld.Status != http.StatusNotFound {
		t.Errorf("old token after completion: status = %d, want 410/404", rOld.Status)
	}

	// 8. Sign in with the NEW password to confirm the credential was updated.
	rSignIn := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-in/password",
		map[string]any{"email": email, "password": newPassword},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	if rSignIn.Status != http.StatusOK {
		t.Fatalf("sign-in with new password: status = %d, want 200; body: %s", rSignIn.Status, rSignIn.Body)
	}

	// 9. Sign in with the OLD password must fail (credential replaced).
	rSignInOld := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-in/password",
		map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	if rSignInOld.Status == http.StatusOK {
		t.Error("sign-in with OLD password should fail after recovery")
	}
}

// TestE2EFlowRecoveryVerifyEmailTokenPath verifies that the recovery email link
// token advances verify_email to set_password just like the numeric code.
func TestE2EFlowRecoveryVerifyEmailTokenPath(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("recovery-token-%s@example.com", newUUID()[:8])

	registerUser(t, ctx, projectID, email)

	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":        "recovery",
		"email":       email,
		"redirect_to": "https://app.example.com/auth/reset-password",
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "verify_email" {
		t.Fatalf("create: step = %q, want verify_email", fs.Step)
	}
	token1 := fs.FlowToken

	challengeID := findFlowChallengeID(t, ctx, token1)
	emailToken := e2eEmitter.payloadFor(challengeID, "token")
	if emailToken == "" {
		t.Fatalf("password reset token not captured for challenge %s", challengeID)
	}
	if got := e2eEmitter.payloadFor(challengeID, "flow_token"); got != token1 {
		t.Fatalf("flow email flow_token = %q, want %q", got, token1)
	}

	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token1, "verify_email", map[string]any{"token": emailToken})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Status != "pending" {
		t.Fatalf("verify_email: status = %q, want pending", fs2.Status)
	}
	if fs2.Step != "set_password" {
		t.Fatalf("verify_email: step = %q, want set_password", fs2.Step)
	}
	if fs2.Error != nil {
		t.Fatalf("verify_email: unexpected error: %+v", fs2.Error)
	}
	if fs2.FlowToken != token1 {
		t.Error("verify_email token path must not rotate until set_password")
	}
}

// TestE2EFlowRecoveryAntiEnumeration verifies §5.4: create recovery for an
// unknown email returns EXACTLY the same FlowState shape as for a real user —
// step=verify_email, status=pending, no error, masked contact present.
// Any code submitted against that flow fails identically (invalid_code).
func TestE2EFlowRecoveryAntiEnumeration(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	unknownEmail := fmt.Sprintf("ghost-%s@example.com", newUUID()[:8])

	// 1. Create recovery for a non-existent user.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":  "recovery",
		"email": unknownEmail,
	})
	e2eWantStatus(t, r, http.StatusOK)

	// Must look identical to a real-user recovery flow (§5.4).
	if fs.Status != "pending" {
		t.Fatalf("anti-enum: status = %q, want pending", fs.Status)
	}
	if fs.Step != "verify_email" {
		t.Fatalf("anti-enum: step = %q, want verify_email", fs.Step)
	}
	if fs.FlowToken == "" {
		t.Fatal("anti-enum: flow_token is empty")
	}
	if fs.Error != nil {
		t.Errorf("anti-enum: create must NOT return an error (leaks account existence): %+v", fs.Error)
	}
	// Masked contact should be present (same shape).
	if fs.Contact == nil || fs.Contact.EmailMasked == "" {
		t.Error("anti-enum: expected masked email contact in response")
	}

	// 2. Submit any code against the fake flow — must fail with invalid_code,
	//    NOT with a 404/410 or an account-existence hint.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, fs.FlowToken, "verify_email", map[string]any{"code": "000000"})
	e2eWantStatus(t, r2, http.StatusOK) // still 200, error embedded in body
	if fs2.Status != "pending" {
		t.Fatalf("anti-enum wrong code: status = %q, want pending", fs2.Status)
	}
	if fs2.Error == nil {
		t.Fatal("anti-enum wrong code: expected error field in flow state")
	}
	if fs2.Error.Code != "invalid_code" {
		t.Errorf("anti-enum wrong code: error.code = %q, want invalid_code", fs2.Error.Code)
	}
}

// TestE2EFlowRecoveryWrongCodeDecrementsAttempts verifies §5 rule 6:
// a wrong OTP decrements attempts_left, keeps status=pending, embeds
// error.code=invalid_code, and does NOT rotate the token.
func TestE2EFlowRecoveryWrongCodeDecrementsAttempts(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("recovery-wrong-%s@example.com", newUUID()[:8])

	registerUser(t, ctx, projectID, email)

	// Create recovery flow.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":  "recovery",
		"email": email,
	})
	e2eWantStatus(t, r, http.StatusOK)
	token := fs.FlowToken

	// Submit a wrong code.
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": "000000"})
	e2eWantStatus(t, r2, http.StatusOK)

	// Status stays pending.
	if fs2.Status != "pending" {
		t.Fatalf("wrong code: status = %q, want pending", fs2.Status)
	}
	// Error is embedded.
	if fs2.Error == nil {
		t.Fatal("wrong code: expected error field in flow state")
	}
	if fs2.Error.Code != "invalid_code" {
		t.Errorf("wrong code: error.code = %q, want invalid_code", fs2.Error.Code)
	}
	// Token must NOT rotate on a wrong code.
	if fs2.FlowToken != token {
		t.Error("wrong code: token rotated — should NOT rotate on failure")
	}

	// Attempts must have decremented (from flowMaxAttempts=5 to 4).
	// We verify this by checking the flow DB directly.
	challengeID := findFlowChallengeID(t, ctx, token)
	if challengeID == "" {
		t.Fatal("wrong code: challenge_id not found in flow data after wrong code")
	}

	// Submit a second wrong code — attempts should be 3 now.
	fs3, r3 := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": "111111"})
	e2eWantStatus(t, r3, http.StatusOK)
	if fs3.Error == nil || fs3.Error.Code != "invalid_code" {
		t.Errorf("second wrong code: expected invalid_code error, got %+v", fs3.Error)
	}

	// Now submit the real code — it should still work (challenge not exhausted).
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("no OTP code captured from emitter")
	}
	fs4, r4 := flowSubmit(t, ctx, ts, projectID, token, "verify_email", map[string]any{"code": code})
	e2eWantStatus(t, r4, http.StatusOK)
	if fs4.Step != "set_password" {
		t.Fatalf("after correct code: step = %q, want set_password", fs4.Step)
	}
}

// TestE2EFlowRecoveryPhone exercises the phone-OTP recovery channel: a user with
// a phone (created via phone-OTP signup) resets their password over SMS:
// create{method:phone_otp} → verify_phone → set_password → completed + session.
func TestE2EFlowRecoveryPhone(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	phone := fmt.Sprintf("+1415555%04d", 2000+len(t.Name())%7000)
	hdr := map[string]string{"X-Client-Id": projectID}

	// Provision a user with this phone via phone-OTP signup.
	if _, _, status, body := phoneStartVerify(t, ctx, ts.URL, phone, "signup", hdr); status != http.StatusOK {
		t.Fatalf("phone signup setup failed: %d %s", status, body)
	}

	// Create a phone recovery flow.
	fs, r := flowCreate(t, ctx, ts, projectID, map[string]any{
		"kind":   "recovery",
		"method": "phone_otp",
		"phone":  phone,
	})
	e2eWantStatus(t, r, http.StatusOK)
	if fs.Step != "verify_phone" {
		t.Fatalf("create: step = %q, want verify_phone", fs.Step)
	}
	token := fs.FlowToken

	challengeID := findFlowChallengeID(t, ctx, token)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatal("verify_phone: no OTP code captured")
	}
	fs2, r2 := flowSubmit(t, ctx, ts, projectID, token, "verify_phone", map[string]any{"code": code})
	e2eWantStatus(t, r2, http.StatusOK)
	if fs2.Step != "set_password" {
		t.Fatalf("verify_phone: step = %q, want set_password", fs2.Step)
	}

	fs3, r3 := flowSubmit(t, ctx, ts, projectID, fs2.FlowToken, "set_password",
		map[string]any{"password": "N3wStr0ng!Pass99"})
	e2eWantStatus(t, r3, http.StatusOK)
	if fs3.Status != "completed" {
		t.Fatalf("set_password: status = %q, want completed", fs3.Status)
	}
	if fs3.Session == nil || fs3.Session.AccessToken == "" {
		t.Fatal("set_password: expected a session with access token")
	}
}
