//go:build integration

package postgres

// Integration e2e tests for mfa_policy ENFORCEMENT at runtime.
//
// What is enforced (see mfa_policy_pg.go):
//   - allowed_factors gates ENROLLMENT of new factors. The DB<->policy name
//     mapping (email->email_otp, recovery->backup_codes) is respected.
//   - A missing mfa_policy doc, or an unset/empty allowed_factors list, allows
//     every factor (backward compatible).
//   - Challenge/verify of a factor enrolled BEFORE a policy tightening keeps
//     working (no lockout); only enrollment is gated.
//   - required_for_admins is loaded but does NOT hard-block a 0-factor account
//     (IAM has no admin-role subject); guarded here against an accidental
//     lockout regression.

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// seedMFAPolicy writes the mfa_policy doc for the project in the default
// runtime env (= mfaDefaultEnv = "live").
func seedMFAPolicy(t *testing.T, ctx context.Context, projectID string, doc map[string]any) {
	t.Helper()
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "mfa_policy", doc)
}

// mfaEnrollEndpoints maps a logical factor to its enroll endpoint + a minimal
// valid request body, so a single policy can be asserted across all surfaces.
type mfaEnrollSurface struct {
	name string
	path string
	body map[string]any
}

func mfaEnrollSurfaces() []mfaEnrollSurface {
	return []mfaEnrollSurface{
		{"totp", "/v1/auth/mfa/totp/enroll", map[string]any{}},
		{"sms", "/v1/auth/mfa/sms/enroll", map[string]any{"phone": "+14155550100"}},
		{"email", "/v1/auth/mfa/email/enroll", map[string]any{"email": "factor@test.com"}},
		{"webauthn", "/v1/auth/mfa/webauthn/enroll/options", map[string]any{"name": "key"}},
		{"recovery", "/v1/auth/mfa/recovery-codes/generate", map[string]any{}},
	}
}

// TestE2EMFAPolicyAllowedFactorsGate: allowed_factors=["totp"] permits the totp
// enroll surface and denies the four others with mfa_factor_not_allowed (403).
func TestE2EMFAPolicyAllowedFactorsGate(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-pol-totp-%s@test.com", newUUID()))
	seedMFAPolicy(t, ctx, projectID, map[string]any{"allowed_factors": []string{"totp"}})

	for _, s := range mfaEnrollSurfaces() {
		s := s
		t.Run(s.name, func(t *testing.T) {
			r := e2eReq(t, ctx, http.MethodPost, ts.URL+s.path, s.body, e2eBearer(sess.AccessToken))
			if s.name == "totp" {
				e2eWantStatus(t, r, http.StatusOK)
				return
			}
			if r.Status != http.StatusForbidden {
				t.Fatalf("%s: want 403 (mfa_factor_not_allowed), got %d\nbody: %s", s.name, r.Status, r.Body)
			}
		})
	}
}

// TestE2EMFAPolicyEmailOtpMapping: allowed_factors=["email_otp"] permits the
// DB "email" factor enroll and denies totp. Confirms the email_otp->email map.
func TestE2EMFAPolicyEmailOtpMapping(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-pol-email-%s@test.com", newUUID()))
	seedMFAPolicy(t, ctx, projectID, map[string]any{"allowed_factors": []string{"email_otp"}})

	t.Run("email enroll permitted", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/mfa/email/enroll",
			map[string]any{"email": fmt.Sprintf("factor-%s@test.com", newUUID())}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
	})
	t.Run("totp denied", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/mfa/totp/enroll",
			map[string]any{}, e2eBearer(sess.AccessToken))
		if r.Status != http.StatusForbidden {
			t.Fatalf("want 403, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EMFAPolicyBackupCodesMapping: allowed_factors=["backup_codes"] permits
// recovery-codes generation and denies totp. Confirms the backup_codes->recovery
// map.
func TestE2EMFAPolicyBackupCodesMapping(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-pol-bc-%s@test.com", newUUID()))
	seedMFAPolicy(t, ctx, projectID, map[string]any{"allowed_factors": []string{"backup_codes"}})

	t.Run("recovery-codes permitted", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/mfa/recovery-codes/generate",
			map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
	})
	t.Run("totp denied", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/mfa/totp/enroll",
			map[string]any{}, e2eBearer(sess.AccessToken))
		if r.Status != http.StatusForbidden {
			t.Fatalf("want 403, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EMFAPolicyDefaultAllowsAll: with no mfa_policy doc, every enroll surface
// succeeds (backward compatibility).
func TestE2EMFAPolicyDefaultAllowsAll(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-pol-default-%s@test.com", newUUID()))
	// No seedMFAPolicy => no doc.

	for _, s := range mfaEnrollSurfaces() {
		s := s
		t.Run(s.name, func(t *testing.T) {
			body := s.body
			if s.name == "email" {
				body = map[string]any{"email": fmt.Sprintf("factor-%s@test.com", newUUID())}
			}
			r := e2eReq(t, ctx, http.MethodPost, ts.URL+s.path, body, e2eBearer(sess.AccessToken))
			e2eWantStatus(t, r, http.StatusOK)
		})
	}
}

// TestE2EMFAPolicyEmptyAllowedAllowsAll: allowed_factors=[] (present but empty)
// is treated as unset => allows all (matches FactorAllowed, and the configspec
// write-time validation blocks only the empty+required lockout combo).
func TestE2EMFAPolicyEmptyAllowedAllowsAll(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-pol-empty-%s@test.com", newUUID()))
	seedMFAPolicy(t, ctx, projectID, map[string]any{"allowed_factors": []string{}})

	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/mfa/totp/enroll",
		map[string]any{}, e2eBearer(sess.AccessToken))
	e2eWantStatus(t, r, http.StatusOK)
}

// TestE2EMFAPolicyPreEnrolledStillUsable: a factor enrolled BEFORE the policy
// tightens must remain usable for login. Enroll an active email factor, then set
// allowed_factors=["webauthn"]; the password sign-in must still gate on MFA
// (issue a flow_token), proving the tightening did not lock the user out.
func TestE2EMFAPolicyPreEnrolledStillUsable(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("mfa-pol-preenroll-%s@test.com", newUUID())
	acc, _ := registerUser(t, ctx, projectID, email)
	e2eActiveEmailFactor(t, ctx, projectID, acc.ID, email)

	// Tighten the policy AFTER the factor is active.
	seedMFAPolicy(t, ctx, projectID, map[string]any{"allowed_factors": []string{"webauthn"}})

	// Sign-in must still require the existing (now-disallowed-for-enroll) factor.
	flowToken := e2eSignInFlowToken(t, ctx, ts.URL, projectID, email)
	if flowToken == "" {
		t.Fatal("expected a flow_token: tightening allowed_factors must not lock out a pre-enrolled factor")
	}
}

// TestE2EMFAPolicyRequiredForAdminsNoLockout: required_for_admins=true with a
// 0-factor account must NOT block password login (IAM has no admin-role subject;
// see §1). Guards against an accidental hard-block regression.
func TestE2EMFAPolicyRequiredForAdminsNoLockout(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("mfa-pol-reqadmin-%s@test.com", newUUID())
	registerUser(t, ctx, projectID, email)
	seedMFAPolicy(t, ctx, projectID, map[string]any{
		"required_for_admins": true,
		"allowed_factors":     []string{"totp"},
	})

	// Password sign-in for a 0-factor account must succeed (no MFA gate, no block).
	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-in/password",
		map[string]any{"email": email, "password": "Sup3rStr0ng!Pass"},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	e2eWantStatus(t, r, http.StatusOK)
	var step struct {
		ResultType string `json:"result_type"`
		Session    struct {
			AccessToken string `json:"access_token"`
		} `json:"session"`
	}
	e2eDecode(t, r, &step)
	// A 0-factor account must authenticate directly (no MFA gate, no hard block).
	if step.ResultType != "authenticated" || step.Session.AccessToken == "" {
		t.Fatalf("0-factor account must log in without MFA when required_for_admins=true; got result_type=%q\nbody: %s", step.ResultType, r.Body)
	}
}

// TestMFALoadPolicyNotFound: the loader returns the zero value (allow-all) when
// no mfa_policy row exists, and round-trips a seeded doc.
func TestMFALoadPolicyNotFound(t *testing.T) {
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	mfa := NewPgMFAAccounts(testDB, e2eEmitter)

	t.Run("missing row => zero (allow all)", func(t *testing.T) {
		pol, err := mfa.mfaLoadPolicy(ctx, projectID)
		if err != nil {
			t.Fatalf("mfaLoadPolicy: %v", err)
		}
		if pol.AllowedFactors != nil {
			t.Errorf("want nil AllowedFactors, got %v", pol.AllowedFactors)
		}
		if !pol.FactorAllowed("totp") || !pol.FactorAllowed("webauthn") {
			t.Error("zero policy must allow all factors")
		}
	})

	t.Run("seeded doc round-trips", func(t *testing.T) {
		seedMFAPolicy(t, ctx, projectID, map[string]any{
			"allowed_factors":     []string{"email_otp"},
			"required_for_admins": true,
		})
		pol, err := mfa.mfaLoadPolicy(ctx, projectID)
		if err != nil {
			t.Fatalf("mfaLoadPolicy: %v", err)
		}
		if !pol.FactorAllowed("email") {
			t.Error("email_otp policy must allow db email factor")
		}
		if pol.FactorAllowed("totp") {
			t.Error("email_otp policy must deny totp")
		}
		if pol.RequiredForAdmins == nil || !*pol.RequiredForAdmins {
			t.Error("required_for_admins should round-trip as true")
		}
	})
}
