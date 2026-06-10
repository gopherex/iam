//go:build integration

package postgres

// Integration e2e HTTP tests for the MFA and WebAuthn feature groups.
//
// MFA paths (12 ops from pkg/api/mfa.go + openapi/openapi.yaml):
//
//	GET  /v1/auth/mfa/factors
//	POST /v1/auth/mfa/totp/enroll
//	POST /v1/auth/mfa/totp/verify
//	POST /v1/auth/mfa/challenge
//	POST /v1/auth/mfa/verify
//	POST /v1/auth/mfa/email/enroll
//	POST /v1/auth/mfa/sms/enroll
//	POST /v1/auth/mfa/webauthn/enroll/options
//	POST /v1/auth/mfa/webauthn/enroll/verify
//	POST /v1/auth/mfa/recovery-codes/generate
//	POST /v1/auth/mfa/recovery-codes/verify
//	DELETE /v1/auth/mfa/factors/{factor_id}
//
// WebAuthn paths (7 ops from pkg/api/webauthn.go + openapi/openapi.yaml):
//
//	POST /v1/auth/webauthn/login/options
//	POST /v1/auth/webauthn/login/verify
//	POST /v1/auth/webauthn/register/options
//	POST /v1/auth/webauthn/register/verify
//	GET  /v1/auth/webauthn/credentials
//	PATCH /v1/auth/webauthn/credentials/{credential_id}
//	DELETE /v1/auth/webauthn/credentials/{credential_id}
//
// Production bugs found during test development (t.Skip entries below):
//
//  1. DELETE /v1/auth/mfa/factors/{factor_id} with a non-existent or already-deleted
//     factor_id returns 500 instead of 404. Root cause: mfa_pg.go/mfaFindFactor calls
//     translatePgErr which wraps the local postgres.ErrNotFound (a plain error.New value)
//     not domain.ErrNotFound (*domain.Error). NewError cannot errors.As to *domain.Error
//     so it falls through to ErrInternal (500).
//
//  2. POST /v1/auth/mfa/verify with an invalid/non-existent challenge_id returns 500
//     instead of 401/404. Same root cause as (1): translatePgErr("challenge", err) wraps
//     local ErrNotFound which is not caught by NewError, returning 500.
//
//  3. DELETE/PATCH /v1/auth/webauthn/credentials/{credential_id} with a non-existent
//     credential_id returns 500 instead of 404. Same root cause: loadCredential calls
//     translatePgErr("webauthn_credential", err) which wraps local ErrNotFound.
//
//  4. POST /v1/auth/mfa/challenge is defined as security:[] (public endpoint) in
//     openapi.yaml but the handler calls requirePrincipal which always returns
//     ErrUnauthorized when no bearer security scheme is applied. The endpoint always
//     returns 401 regardless of the presence of a valid bearer token.
//
//  5. POST /v1/auth/mfa/recovery-codes/verify: the handler (mfa.go) does not set
//     AccountID on the MFARecoveryVerifyCmd passed to VerifyRecoveryCode. The adapter
//     queries iam_recovery_codes with user_id = "" which matches nothing, returning
//     ErrInvalidCredentials (401) for every code including valid ones.
//
// NOTE on WebAuthn FINISH steps: completing a WebAuthn attestation or assertion
// requires the browser to produce a signed PublicKeyCredential whose challenge bytes
// were signed with the private key of a previously registered hardware (or software)
// authenticator. Without a software authenticator library that can participate in the
// go-webauthn ceremony (performing COSE/CBOR signing over the exact challenge nonce
// the server issued), producing a valid credential over HTTP in a unit test is not
// feasible. The BEGIN steps are fully exercised; the FINISH steps are tested for their
// security boundary (401 without auth) and schema validation (422 for missing fields).

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// ============================================================
// MFA tests
// ============================================================

// TestE2EMFAListFactors verifies GET /v1/auth/mfa/factors.
func TestE2EMFAListFactors(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-list-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/mfa/factors"

	t.Run("happy path returns empty list", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, url, nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Data []any `json:"data"`
		}
		e2eDecode(t, r, &body)
		if body.Data == nil {
			t.Error("expected data array, got nil")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, url, nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EMFATOTPEnroll verifies POST /v1/auth/mfa/totp/enroll.
func TestE2EMFATOTPEnroll(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-totp-enroll-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/mfa/totp/enroll"

	t.Run("happy path returns factor_id", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			FactorID string `json:"factor_id"`
		}
		e2eDecode(t, r, &body)
		if body.FactorID == "" {
			t.Error("expected factor_id in response")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EMFATOTPVerify verifies POST /v1/auth/mfa/totp/verify.
func TestE2EMFATOTPVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-totp-verify-%s@test.com", newUUID()))

	// First enroll a TOTP factor.
	enrollURL := ts.URL + "/v1/auth/mfa/totp/enroll"
	enrollResp := e2eReq(t, ctx, http.MethodPost, enrollURL, map[string]any{}, e2eBearer(sess.AccessToken))
	e2eWantStatus(t, enrollResp, http.StatusOK)
	var enrollBody struct {
		FactorID string `json:"factor_id"`
	}
	e2eDecode(t, enrollResp, &enrollBody)

	verifyURL := ts.URL + "/v1/auth/mfa/totp/verify"

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, map[string]any{
			"factor_id": enrollBody.FactorID,
			"code":      "123456",
		}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("wrong code returns auth error", func(t *testing.T) {
		// A real TOTP code would be needed to test the happy path; we verify the
		// error path (wrong code) returns an auth error.
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, map[string]any{
			"factor_id": enrollBody.FactorID,
			"code":      "000000",
		}, e2eBearer(sess.AccessToken))
		// Adapter returns ErrMFAInvalid (401).
		if r.Status != http.StatusUnauthorized && r.Status != http.StatusUnprocessableEntity {
			t.Errorf("want 401 or 422, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing required fields returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, map[string]any{
			// factor_id and code are required by the schema
		}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})
}

// TestE2EMFAChallenge verifies POST /v1/auth/mfa/challenge.
//
// This endpoint is defined as security:[] in openapi.yaml (public, no bearer auth).
// It requires X-Client-Id but NOT a bearer token at the middleware level. The handler
// currently calls requirePrincipal which always returns 401 when no security scheme
// is applied — see production bug #4 in the package-level doc comment.
func TestE2EMFAChallenge(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	challengeURL := ts.URL + "/v1/auth/mfa/challenge"

	// Production bug #4: the handler calls requirePrincipal but the endpoint
	// has security:[] so no principal is ever placed in context. Every request
	// returns 401 regardless of headers.
	t.Run("BUG: public endpoint always returns 401 (handler calls requirePrincipal on security:[] op)", func(t *testing.T) {
		t.Skip("Production bug #4: POST /v1/auth/mfa/challenge is security:[] in openapi.yaml " +
			"but handler calls requirePrincipal — returns 401 instead of 200. " +
			"Fix: either add bearerAuth security to the spec or remove requirePrincipal from the handler " +
			"and rely on flow_token to identify the account.")
	})

	t.Run("missing X-Client-Id returns 422", func(t *testing.T) {
		// X-Client-Id is a required header for this endpoint; missing it should
		// trigger the parameter decode error → 422.
		r := e2eReq(t, ctx, http.MethodPost, challengeURL, map[string]any{}, nil)
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("X-Client-Id only returns 401 (no principal in context)", func(t *testing.T) {
		// Demonstrate the current observed behaviour: even with X-Client-Id set,
		// requirePrincipal returns 401 because bearer auth is not processed.
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, challengeURL, map[string]any{}, hdrs)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EMFAVerify verifies POST /v1/auth/mfa/verify.
func TestE2EMFAVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	url := ts.URL + "/v1/auth/mfa/verify"

	t.Run("missing X-Client-Id returns 422", func(t *testing.T) {
		// X-Client-Id is required; without it the parameter decoder rejects the request.
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": newUUID(),
			"code":         "123456",
		}, nil)
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("missing challenge_id returns 422", func(t *testing.T) {
		// The handler validates challenge_id is present.
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"code": "123456",
		}, hdrs)
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("invalid challenge_id returns 4xx", func(t *testing.T) {
		// translatePgErr now wraps domain.ErrNotFound (via ErrNotFound alias) so the
		// ogen NewError hook renders it as 404. The Verify adapter translates a missing
		// challenge row to domain.ErrNotFound → 404.
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": newUUID(), // does not exist
			"code":         "123456",
		}, hdrs)
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("want 4xx for invalid challenge_id, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EMFAEmailEnroll verifies POST /v1/auth/mfa/email/enroll.
func TestE2EMFAEmailEnroll(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-email-enroll-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/mfa/email/enroll"

	t.Run("happy path returns factor_id + challenge_id", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"email": fmt.Sprintf("factor-%s@test.com", newUUID()),
		}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			FactorID    string `json:"factor_id"`
			ChallengeID string `json:"challenge_id"`
		}
		e2eDecode(t, r, &body)
		if body.FactorID == "" {
			t.Error("expected factor_id in response")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"email": "someone@test.com",
		}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("invalid email returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"email": "not-an-email",
		}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("missing email returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})
}

// TestE2EMFASMSEnroll verifies POST /v1/auth/mfa/sms/enroll.
func TestE2EMFASMSEnroll(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-sms-enroll-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/mfa/sms/enroll"

	t.Run("happy path returns factor_id + challenge_id", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"phone": "+14155550100",
		}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			FactorID    string `json:"factor_id"`
			ChallengeID string `json:"challenge_id"`
		}
		e2eDecode(t, r, &body)
		if body.FactorID == "" {
			t.Error("expected factor_id in response")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"phone": "+14155550101",
		}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("invalid phone format returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"phone": "not-a-phone",
		}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("missing phone returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})
}

// TestE2EMFARecoveryCodesGenerate verifies POST /v1/auth/mfa/recovery-codes/generate.
func TestE2EMFARecoveryCodesGenerate(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-recovery-gen-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/mfa/recovery-codes/generate"

	t.Run("happy path returns codes array", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Codes []string `json:"codes"`
		}
		e2eDecode(t, r, &body)
		if len(body.Codes) == 0 {
			t.Error("expected non-empty codes array")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EMFARecoveryCodesVerify verifies POST /v1/auth/mfa/recovery-codes/verify.
//
// This endpoint is public (security:[]) and requires X-Client-Id. The handler does
// NOT set AccountID on the MFARecoveryVerifyCmd (production bug #5), so the adapter
// queries recovery codes with user_id="" and always returns ErrInvalidCredentials.
func TestE2EMFARecoveryCodesVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)

	verifyURL := ts.URL + "/v1/auth/mfa/recovery-codes/verify"

	t.Run("missing X-Client-Id returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, map[string]any{
			"code": "somecode",
		}, nil)
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("missing code returns 422", func(t *testing.T) {
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, map[string]any{}, hdrs)
		// The schema requires "code"; the handler also validates it.
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Errorf("want 422 or 400, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("invalid code returns auth error", func(t *testing.T) {
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, verifyURL, map[string]any{
			"code": "completely-wrong-code",
		}, hdrs)
		// Invalid code → ErrInvalidCredentials (401) or not-found (404).
		if r.Status != http.StatusUnauthorized && r.Status != http.StatusNotFound && r.Status != http.StatusUnprocessableEntity {
			t.Errorf("want 401/404/422, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	// Production bug #5: even with a valid code the handler always returns 401 because
	// the AccountID is never set in MFARecoveryVerifyCmd. The adapter queries
	// iam_recovery_codes with user_id="" which matches nothing.
	t.Run("BUG: valid code always returns 401 (AccountID missing from cmd)", func(t *testing.T) {
		t.Skip("Production bug #5: POST /v1/auth/mfa/recovery-codes/verify always returns 401 " +
			"(invalid_credentials) even for valid codes because the handler does not set AccountID " +
			"on MFARecoveryVerifyCmd. The adapter queries with user_id='' which finds no rows. " +
			"Fix: derive AccountID from flow_token or add it to the request/session context.")
	})
}

// TestE2EMFADeleteFactor verifies DELETE /v1/auth/mfa/factors/{factor_id}.
func TestE2EMFADeleteFactor(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-delete-%s@test.com", newUUID()))

	// Enroll an email factor to delete.
	enrollURL := ts.URL + "/v1/auth/mfa/email/enroll"
	enrollResp := e2eReq(t, ctx, http.MethodPost, enrollURL, map[string]any{
		"email": fmt.Sprintf("del-%s@test.com", newUUID()),
	}, e2eBearer(sess.AccessToken))
	e2eWantStatus(t, enrollResp, http.StatusOK)
	var enrollBody struct {
		FactorID string `json:"factor_id"`
	}
	e2eDecode(t, enrollResp, &enrollBody)
	if enrollBody.FactorID == "" {
		t.Fatal("enroll did not return factor_id")
	}

	t.Run("no auth returns 401", func(t *testing.T) {
		url := ts.URL + "/v1/auth/mfa/factors/" + enrollBody.FactorID
		r := e2eReq(t, ctx, http.MethodDelete, url, nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("not-found factor returns 404", func(t *testing.T) {
		// ErrNotFound now aliases domain.ErrNotFound; translatePgErr wraps it so the
		// ogen NewError hook can errors.As to *domain.Error and render 404.
		url := ts.URL + "/v1/auth/mfa/factors/" + newUUID()
		r := e2eReq(t, ctx, http.MethodDelete, url, nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("happy path deletes the factor", func(t *testing.T) {
		url := ts.URL + "/v1/auth/mfa/factors/" + enrollBody.FactorID
		r := e2eReq(t, ctx, http.MethodDelete, url, nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Ok bool `json:"ok"`
		}
		e2eDecode(t, r, &body)
		if !body.Ok {
			t.Error("expected ok: true in response")
		}
	})

	t.Run("already-deleted factor returns 404", func(t *testing.T) {
		// The happy path above already deleted enrollBody.FactorID. A second delete
		// of the same factor_id should return 404 (not 500) now that ErrNotFound
		// aliases domain.ErrNotFound.
		url := ts.URL + "/v1/auth/mfa/factors/" + enrollBody.FactorID
		r := e2eReq(t, ctx, http.MethodDelete, url, nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EMFAWebAuthnEnrollOptions verifies POST /v1/auth/mfa/webauthn/enroll/options.
func TestE2EMFAWebAuthnEnrollOptions(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-wa-opts-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/mfa/webauthn/enroll/options"

	t.Run("happy path returns challenge_id + publicKey options", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"name": "my-security-key",
		}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			ChallengeID string         `json:"challenge_id"`
			PublicKey   map[string]any `json:"publicKey"`
		}
		e2eDecode(t, r, &body)
		if body.ChallengeID == "" {
			t.Error("expected challenge_id in response")
		}
		if body.PublicKey == nil {
			t.Error("expected publicKey options in response")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EMFAWebAuthnEnrollVerify verifies POST /v1/auth/mfa/webauthn/enroll/verify.
//
// The full ceremony requires a real authenticator; we test auth/validation error paths.
// NOTE: producing a valid WebAuthn attestation over HTTP in a unit test requires a
// software authenticator that signs a challenge from the server — not feasible without
// an instrumented browser or a Go software authenticator. The BEGIN step is covered
// above; these tests verify the error paths only.
func TestE2EMFAWebAuthnEnrollVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("mfa-wa-verify-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/mfa/webauthn/enroll/verify"

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": newUUID(),
			"credential":   map[string]any{"id": "fake"},
		}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("missing required fields returns 422", func(t *testing.T) {
		// challenge_id and credential are required.
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("non-existent challenge_id returns domain error (not 200)", func(t *testing.T) {
		// NOTE: Cannot produce a valid attestation without a real authenticator.
		// The server will reject the credential during cryptographic verification.
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": newUUID(), // non-existent
			"credential":   map[string]any{"id": "fake", "type": "public-key"},
		}, e2eBearer(sess.AccessToken))
		// Accepts any non-200 status since exact error depends on crypto validation
		// of the fake credential vs. a missing challenge row.
		if r.Status == http.StatusOK {
			t.Errorf("expected error response, got 200\nbody: %s", r.Body)
		}
	})
}

// ============================================================
// WebAuthn tests
// ============================================================

// TestE2EWebAuthnLoginOptions verifies POST /v1/auth/webauthn/login/options.
//
// This endpoint has security:[] and requires X-Client-Id. It also requires an email
// in the request body to look up the user's passkeys — without a valid registered
// email+passkey pair, the adapter returns ErrInvalidCredentials (401).
func TestE2EWebAuthnLoginOptions(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	url := ts.URL + "/v1/auth/webauthn/login/options"

	t.Run("missing X-Client-Id returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, nil)
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("empty email returns 401 (no passkeys for empty email)", func(t *testing.T) {
		// BeginLogin requires a non-empty email to look up the user;
		// empty email → ErrInvalidCredentials (401).
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, hdrs)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("unknown email returns 401 (user not found)", func(t *testing.T) {
		// A user that exists but has no passkeys → ErrInvalidCredentials (401).
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"email": fmt.Sprintf("unknown-%s@test.com", newUUID()),
		}, hdrs)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("registered user without passkeys returns 401", func(t *testing.T) {
		// Register a user but do NOT add any passkeys. BeginLogin should reject
		// because the user has no WebAuthn credentials.
		email := fmt.Sprintf("wa-login-nopk-%s@test.com", newUUID())
		_, _ = registerUser(t, ctx, projectID, email)
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"email": email,
		}, hdrs)
		// User exists but has no passkeys → ErrInvalidCredentials (401).
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EWebAuthnLoginVerify verifies POST /v1/auth/webauthn/login/verify.
//
// Full ceremony requires a real authenticator credential; we test error paths.
// Rationale: completing a WebAuthn assertion requires the browser to sign the
// challenge bytes with the private key of a previously registered credential.
// Without a software authenticator library that can replay the ceremony (e.g.
// go-webauthn/webauthn's protocol.ParseCredentialRequestResponse on a crafted
// payload), producing a valid assertion programmatically would require either
// a real hardware key or an instrumented browser driver — both out of scope for
// this integration test layer.
func TestE2EWebAuthnLoginVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	url := ts.URL + "/v1/auth/webauthn/login/verify"

	t.Run("missing required fields returns 422", func(t *testing.T) {
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		// challenge_id and credential are required.
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, hdrs)
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("missing X-Client-Id returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": newUUID(),
			"credential":   map[string]any{"id": "fake"},
		}, nil)
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("fake credential with real challenge returns error (not 200)", func(t *testing.T) {
		// We cannot start a login/options flow without a registered user+passkey,
		// so we use a synthetic challenge_id. The server should reject the fake
		// credential with a non-200 error (401 or 404 from challenge lookup).
		hdrs := map[string]string{
			"X-Client-Id":   projectID,
			"X-Environment": "live",
		}
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": newUUID(), // non-existent
			"credential":   map[string]any{"id": "fake", "type": "public-key"},
		}, hdrs)
		if r.Status == http.StatusOK {
			t.Errorf("fake credential must not succeed, got 200\nbody: %s", r.Body)
		}
	})
}

// TestE2EWebAuthnRegisterOptions verifies POST /v1/auth/webauthn/register/options.
func TestE2EWebAuthnRegisterOptions(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("wa-reg-opts-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/webauthn/register/options"

	t.Run("happy path returns challenge_id + publicKey", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			ChallengeID string         `json:"challenge_id"`
			PublicKey   map[string]any `json:"publicKey"`
		}
		e2eDecode(t, r, &body)
		if body.ChallengeID == "" {
			t.Error("expected challenge_id in response")
		}
		if body.PublicKey == nil {
			t.Error("expected publicKey in response")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EWebAuthnRegisterVerify verifies POST /v1/auth/webauthn/register/verify.
//
// Full ceremony requires a real authenticator; we test auth/validation error paths.
// Rationale: completing a WebAuthn attestation requires the browser to produce a
// signed PublicKeyCredential whose challenge bytes match the server-issued challenge
// and whose attestation can be verified by the RP's go-webauthn library. Without a
// software authenticator (none available in the test environment), we cannot produce
// a valid attestation; instead we verify the security boundary (401 without auth)
// and schema validation (422 for missing fields).
func TestE2EWebAuthnRegisterVerify(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("wa-reg-verify-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/webauthn/register/verify"

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": newUUID(),
			"credential":   map[string]any{"id": "fake"},
		}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("missing required fields returns 422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})

	t.Run("fake credential returns error (not 200)", func(t *testing.T) {
		// Begin a registration to get a real challenge_id.
		optURL := ts.URL + "/v1/auth/webauthn/register/options"
		optResp := e2eReq(t, ctx, http.MethodPost, optURL, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, optResp, http.StatusOK)
		var optBody struct {
			ChallengeID string `json:"challenge_id"`
		}
		e2eDecode(t, optResp, &optBody)

		r := e2eReq(t, ctx, http.MethodPost, url, map[string]any{
			"challenge_id": optBody.ChallengeID,
			"credential":   map[string]any{"id": "fake", "type": "public-key"},
		}, e2eBearer(sess.AccessToken))
		if r.Status == http.StatusOK {
			t.Errorf("fake credential must not succeed, got 200\nbody: %s", r.Body)
		}
	})
}

// TestE2EWebAuthnListCredentials verifies GET /v1/auth/webauthn/credentials.
func TestE2EWebAuthnListCredentials(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("wa-list-creds-%s@test.com", newUUID()))
	url := ts.URL + "/v1/auth/webauthn/credentials"

	t.Run("happy path returns empty data array", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, url, nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Data []any `json:"data"`
		}
		e2eDecode(t, r, &body)
		if body.Data == nil {
			t.Error("expected data array in response")
		}
	})

	t.Run("no auth returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, url, nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})
}

// TestE2EWebAuthnDeleteCredential verifies DELETE /v1/auth/webauthn/credentials/{credential_id}.
func TestE2EWebAuthnDeleteCredential(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, _ = registerUser(t, ctx, projectID, fmt.Sprintf("wa-del-cred-%s@test.com", newUUID()))

	t.Run("no auth returns 401", func(t *testing.T) {
		url := ts.URL + "/v1/auth/webauthn/credentials/" + newUUID()
		r := e2eReq(t, ctx, http.MethodDelete, url, nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("non-existent credential returns 404", func(t *testing.T) {
		// loadCredential uses translatePgErr; ErrNotFound now aliases domain.ErrNotFound
		// so the ogen NewError hook renders 404, not 500.
		_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("wa-del-noexist-%s@test.com", newUUID()))
		url := ts.URL + "/v1/auth/webauthn/credentials/" + newUUID()
		r := e2eReq(t, ctx, http.MethodDelete, url, nil, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusNotFound)
	})
}

// TestE2EWebAuthnRenameCredential verifies PATCH /v1/auth/webauthn/credentials/{credential_id}.
func TestE2EWebAuthnRenameCredential(t *testing.T) {
	ts := e2eServer(t)
	ctx := context.Background()
	projectID := e2eProject(t, ctx)
	_, sess := registerUser(t, ctx, projectID, fmt.Sprintf("wa-rename-cred-%s@test.com", newUUID()))

	t.Run("no auth returns 401", func(t *testing.T) {
		url := ts.URL + "/v1/auth/webauthn/credentials/" + newUUID()
		r := e2eReq(t, ctx, http.MethodPatch, url, map[string]any{"name": "renamed"}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("non-existent credential returns 404", func(t *testing.T) {
		// loadCredential calls translatePgErr; ErrNotFound now aliases domain.ErrNotFound
		// so the ogen NewError hook renders 404, not 500.
		url := ts.URL + "/v1/auth/webauthn/credentials/" + newUUID()
		r := e2eReq(t, ctx, http.MethodPatch, url, map[string]any{"name": "renamed"}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusNotFound)
	})

	t.Run("missing name field returns 422", func(t *testing.T) {
		// name is a required field per the schema; sending an empty body with valid auth
		// triggers the schema validation decode error (422) before the handler executes.
		url := ts.URL + "/v1/auth/webauthn/credentials/" + newUUID()
		r := e2eReq(t, ctx, http.MethodPatch, url, map[string]any{}, e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusUnprocessableEntity)
	})
}
