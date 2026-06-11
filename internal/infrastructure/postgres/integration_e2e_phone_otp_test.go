//go:build integration

package postgres

// integration_e2e_phone_otp_test.go — HTTP end-to-end tests for phone-OTP
// primary auth (login + signup) over /v1/auth/otp/{start,verify} with
// channel=sms. Exercises the phone-aware resolve/create path added to the
// passwordless adapter: signup-by-phone, signin reuse, purpose gating,
// E.164 validation, AMR=sms, replay, and environment isolation.
//
// The plaintext OTP code only reaches the SMS delivery channel in production;
// the harness capture emitter (e2eEmitter) records it from the
// auth.otp.started event so the happy path can be exercised.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// phoneStartVerify drives a full sms-channel OTP start+verify and returns the
// decoded session access token. headers must carry X-Client-Id (and optionally
// X-Environment).
func phoneStartVerify(t *testing.T, ctx context.Context, baseURL, phone, purpose string, headers map[string]string) (accessToken, accountID string, verifyStatus int, verifyBody []byte) {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodPost, baseURL+"/v1/auth/otp/start",
		map[string]any{"identifier": phone, "channel": "sms", "purpose": purpose},
		headers)
	e2eWantStatus(t, r, http.StatusOK)
	var start map[string]any
	e2eDecode(t, r, &start)
	challengeID, _ := start["challenge_id"].(string)
	if challengeID == "" {
		t.Fatalf("no challenge_id in phone OTP start response: %s", r.Body)
	}
	code := e2eEmitter.payloadFor(challengeID, "code")
	if code == "" {
		t.Fatalf("OTP code not captured for challenge %s", challengeID)
	}
	// Sanity: the started event must carry the sms channel + phone recipient.
	if ch := e2eEmitter.payloadFor(challengeID, "channel"); ch != "sms" {
		t.Fatalf("started event channel = %q, want sms", ch)
	}
	if to := e2eEmitter.payloadFor(challengeID, "to"); to != phone {
		t.Fatalf("started event to = %q, want %q", to, phone)
	}

	vr := e2eReq(t, ctx, http.MethodPost, baseURL+"/v1/auth/otp/verify",
		map[string]any{"challenge_id": challengeID, "code": code}, headers)
	var body struct {
		Session struct {
			AccessToken string `json:"access_token"`
		} `json:"session"`
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if vr.Status == http.StatusOK {
		e2eDecode(t, vr, &body)
	}
	id := body.Account.ID
	if id == "" {
		id = body.User.ID
	}
	return body.Session.AccessToken, id, vr.Status, vr.Body
}

// TestE2EPhoneOTPSignupRoundTrip: unknown phone + purpose=signup mints a session
// and the access token carries amr containing "sms".
func TestE2EPhoneOTPSignupRoundTrip(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	phone := fmt.Sprintf("+1415555%04d", 1000+len(t.Name())%9000)
	hdr := map[string]string{"X-Client-Id": projectID}

	access, _, status, body := phoneStartVerify(t, ctx, ts.URL, phone, "signup", hdr)
	if status != http.StatusOK {
		t.Fatalf("phone signup verify status = %d, body=%s", status, body)
	}
	if access == "" {
		t.Fatalf("expected access_token after phone OTP verify, body=%s", body)
	}

	// AMR must record the sms channel (in addition to otp). Read it back from
	// the live token claims.
	r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/tokens/current", nil, e2eBearer(access))
	e2eWantStatus(t, r, http.StatusOK)
	var claims struct {
		Claims map[string]json.RawMessage `json:"claims"`
	}
	e2eDecode(t, r, &claims)
	var amr []string
	if raw, ok := claims.Claims["amr"]; ok {
		_ = json.Unmarshal(raw, &amr)
	}
	if !containsStr(amr, "otp") || !containsStr(amr, "sms") {
		t.Fatalf("amr = %v, want to contain both otp and sms", amr)
	}
}

// TestE2EPhoneOTPSigninReuse: a second sms-OTP on the same phone resolves the
// same account (no duplicate user).
func TestE2EPhoneOTPSigninReuse(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	phone := "+14155557777"
	hdr := map[string]string{"X-Client-Id": projectID}

	_, firstID, status, body := phoneStartVerify(t, ctx, ts.URL, phone, "signup", hdr)
	if status != http.StatusOK || firstID == "" {
		t.Fatalf("first phone signup failed: status=%d id=%q body=%s", status, firstID, body)
	}
	_, secondID, status2, body2 := phoneStartVerify(t, ctx, ts.URL, phone, "signin", hdr)
	if status2 != http.StatusOK {
		t.Fatalf("second phone signin failed: status=%d body=%s", status2, body2)
	}
	if secondID != firstID {
		t.Fatalf("signin resolved a different account: first=%q second=%q", firstID, secondID)
	}
}

// TestE2EPhoneOTPVerifyUnknown: purpose=verify on an unknown phone must NOT
// create a user — verify returns 404 (user_not_found). (purpose=signin, like the
// email-OTP contract, is sign-in-or-sign-up and would provision instead.)
func TestE2EPhoneOTPVerifyUnknown(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	phone := "+14155558888"
	hdr := map[string]string{"X-Client-Id": projectID}

	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/start",
		map[string]any{"identifier": phone, "channel": "sms", "purpose": "verify"}, hdr)
	e2eWantStatus(t, r, http.StatusOK)
	var start map[string]any
	e2eDecode(t, r, &start)
	challengeID, _ := start["challenge_id"].(string)
	code := e2eEmitter.payloadFor(challengeID, "code")
	if code == "" {
		t.Fatalf("OTP code not captured for challenge %s", challengeID)
	}
	vr := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/verify",
		map[string]any{"challenge_id": challengeID, "code": code}, hdr)
	if vr.Status != http.StatusNotFound {
		t.Fatalf("verify-on-unknown-phone status = %d, want 404\nbody: %s", vr.Status, vr.Body)
	}
}

// TestE2EPhoneOTPInvalidIdentifier: a non-E.164 identifier on the sms channel is
// rejected at start with 400.
func TestE2EPhoneOTPInvalidIdentifier(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	hdr := map[string]string{"X-Client-Id": projectID}

	for _, bad := range []string{"not-a-phone", "415-555-1234", "001234", "+0123", "user@example.com"} {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/start",
			map[string]any{"identifier": bad, "channel": "sms", "purpose": "signup"}, hdr)
		if r.Status != http.StatusBadRequest {
			t.Errorf("sms start with identifier %q status = %d, want 400\nbody: %s", bad, r.Status, r.Body)
		}
	}
}

// TestE2EPhoneOTPReplay: re-verifying a consumed challenge fails.
func TestE2EPhoneOTPReplay(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	e2eEnableSMSProvider(t, ctx, projectID)
	phone := "+14155559999"
	hdr := map[string]string{"X-Client-Id": projectID}

	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/start",
		map[string]any{"identifier": phone, "channel": "sms", "purpose": "signup"}, hdr)
	e2eWantStatus(t, r, http.StatusOK)
	var start map[string]any
	e2eDecode(t, r, &start)
	challengeID, _ := start["challenge_id"].(string)
	code := e2eEmitter.payloadFor(challengeID, "code")

	first := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/verify",
		map[string]any{"challenge_id": challengeID, "code": code}, hdr)
	e2eWantStatus(t, first, http.StatusOK)

	second := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/verify",
		map[string]any{"challenge_id": challengeID, "code": code}, hdr)
	if second.Status == http.StatusOK {
		t.Fatalf("replay of consumed phone OTP unexpectedly succeeded: %s", second.Body)
	}
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
