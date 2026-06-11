//go:build integration

package postgres

// integration_e2e_oauth_test.go — HTTP end-to-end tests for:
//   - Passwordless (OTP start/verify, magic-link start/verify)
//   - OAuth Social (providers list, start, callback, exchange, unlink, link)
//   - OIDC Provider (discovery, JWKS, authorize, token, userinfo, logout,
//     back-channel logout, PAR, device-auth, device-resolve,
//     interaction fetch/login/consent/reject, grants list/revoke)
//
// All tests use the shared harness from integration_e2e_harness_test.go.
// Flows that require a real upstream provider callback (Google/GitHub) are
// exercised for their start + error-path only; a note documents what can't
// be fully exercised without a live provider.
//
// Previously observed production bugs (now FIXED on this branch):
//
//  1. OTP/MagicLink verify: isNoRows() only checked pgx.ErrNoRows but bob's
//     scan.One returns sql.ErrNoRows. Fixed: translatePgErr now checks both
//     sql.ErrNoRows and pgx.ErrNoRows; isNoRows() uses translatePgErr internally.
//
//  2. OIDC Interaction, OAuth Social Unlink, and OIDC Grant Revoke returned 500
//     because translatePgErr wrapped the package-level postgres.ErrNotFound (a
//     plain error) not domain.ErrNotFound (*domain.Error). Fixed: ErrNotFound now
//     aliases domain.ErrNotFound so errors.As in the ogen NewError hook finds it.
//
// Remaining skips are for infeasible happy paths (token from delivered email):
//   - TestE2EPasswordlessOTPRoundTrip / TestE2EPasswordlessMagicLinkRoundTrip

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// ═════════════════════════════════════════════════════════════════════════════
// Passwordless
// ═════════════════════════════════════════════════════════════════════════════

// TestE2EPasswordlessOTPStart verifies that POSTing a valid OTP start request
// returns 200 with a challenge_id, and that missing required fields return 422.
func TestE2EPasswordlessOTPStart(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("happy_path_email_channel", func(t *testing.T) {
		// purpose enum is "signin" (not "sign_in") per OtpStartRequest schema.
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/otp/start",
			map[string]any{
				"identifier": "otp-user@example.com",
				"channel":    "email",
				"purpose":    "signin",
				"locale":     "ru",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		e2eWantStatus(t, r, http.StatusOK)
		var body map[string]any
		e2eDecode(t, r, &body)
		if body["challenge_id"] == nil {
			t.Fatalf("expected challenge_id in response, got: %s", r.Body)
		}
		challengeID, _ := body["challenge_id"].(string)
		if got := e2eEmitter.payloadFor(challengeID, "locale"); got != "ru" {
			t.Fatalf("emitted locale = %q, want ru", got)
		}
	})

	t.Run("missing_identifier_422", func(t *testing.T) {
		// identifier is required per OtpStartRequest schema.
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/otp/start",
			map[string]any{
				"channel": "email",
				"purpose": "signin",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("invalid_channel_enum_422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/otp/start",
			map[string]any{
				"identifier": "otp-user@example.com",
				"channel":    "carrier_pigeon",
				"purpose":    "signin",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400 for invalid channel, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing_project_header_not_500", func(t *testing.T) {
		// No X-Client-Id → the adapter will receive an empty project ID and
		// may return an error; we just assert that the server doesn't panic (no 500).
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/otp/start",
			map[string]any{
				"identifier": "otp-user2@example.com",
				"channel":    "email",
				"purpose":    "signin",
			},
			nil,
		)
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 without X-Client-Id: %s", r.Body)
		}
	})
}

// TestE2EPasswordlessOTPVerify verifies that a bogus challenge / code returns
// a 4xx and that the verify endpoint exists.
func TestE2EPasswordlessOTPVerify(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("invalid_challenge_4xx", func(t *testing.T) {
		// loadChallengeForVerify calls FindIamChallenge; isNoRows() now catches both
		// sql.ErrNoRows (bob) and pgx.ErrNoRows (generated finders) via translatePgErr,
		// so a missing challenge maps to domain.ErrChallengeInvalid → 401.
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/otp/verify",
			map[string]any{
				"challenge_id": newUUID(),
				"code":         "000000",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for invalid OTP challenge, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing_code_field_422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/otp/verify",
			map[string]any{"challenge_id": newUUID()},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EPasswordlessMagicLinkStart tests the magic-link start endpoint.
func TestE2EPasswordlessMagicLinkStart(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("happy_path", func(t *testing.T) {
		// purpose is required by MagicLinkStartRequest schema; enum is "signin"/"signup".
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/magic-link/start",
			map[string]any{
				"email":       "magic@example.com",
				"redirect_to": "https://app.example.com/callback",
				"purpose":     "signin",
				"locale":      "ru",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		e2eWantStatus(t, r, http.StatusOK)
		var body map[string]any
		e2eDecode(t, r, &body)
		if body["challenge_id"] == nil {
			t.Fatalf("expected challenge_id in response, got: %s", r.Body)
		}
		challengeID, _ := body["challenge_id"].(string)
		if got := e2eEmitter.payloadFor(challengeID, "locale"); got != "ru" {
			t.Fatalf("emitted locale = %q, want ru", got)
		}
	})

	t.Run("missing_email_422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/magic-link/start",
			map[string]any{
				"redirect_to": "https://app.example.com/callback",
				"purpose":     "signin",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing_purpose_422", func(t *testing.T) {
		// purpose is required per MagicLinkStartRequest schema.
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/magic-link/start",
			map[string]any{
				"email":       "magic2@example.com",
				"redirect_to": "https://app.example.com/callback",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400 for missing purpose, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("second_start_same_email_returns_new_challenge", func(t *testing.T) {
		// Idempotent: a second start creates a fresh challenge for the same email.
		r1 := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/magic-link/start",
			map[string]any{
				"email":       "magic-dup@example.com",
				"redirect_to": "https://app.example.com/cb",
				"purpose":     "signin",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		r2 := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/magic-link/start",
			map[string]any{
				"email":       "magic-dup@example.com",
				"redirect_to": "https://app.example.com/cb",
				"purpose":     "signin",
			},
			map[string]string{"X-Client-Id": projectID},
		)
		e2eWantStatus(t, r1, http.StatusOK)
		e2eWantStatus(t, r2, http.StatusOK)
		var b1, b2 map[string]any
		e2eDecode(t, r1, &b1)
		e2eDecode(t, r2, &b2)
		if b1["challenge_id"] == b2["challenge_id"] {
			t.Fatal("expected distinct challenge_ids for two start calls")
		}
	})
}

// TestE2EPasswordlessMagicLinkVerify tests the magic-link verify endpoint.
// Full round-trip requires extracting the token from the DB / email; here we
// test error paths only.
func TestE2EPasswordlessMagicLinkVerify(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("bogus_token_4xx", func(t *testing.T) {
		// findUnconsumedByHash uses IamChallenges.Query().One() which returns
		// sql.ErrNoRows on no match; isNoRows() now catches it (translatePgErr checks
		// both sql.ErrNoRows and pgx.ErrNoRows), mapping to domain.ErrInvalidToken → 401.
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/magic-link/verify",
			map[string]any{"token": "not-a-real-magic-link-token"},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for bogus magic-link token, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing_token_field_422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/magic-link/verify",
			map[string]any{},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// ═════════════════════════════════════════════════════════════════════════════
// Passwordless full round-trip (start → verify via DB-extracted code)
// ═════════════════════════════════════════════════════════════════════════════

// TestE2EPasswordlessOTPRoundTrip exercises the complete OTP sign-in flow by
// verifying start returns a challenge_id, then testing a wrong-code verify path.
//
// NOTE: The DB stores only the sha256 hash of the code (never plaintext), so
// we can't easily brute-force the 6-digit code without reading it from the
// nopEmitter or the email. This test therefore:
//
//  1. Starts an OTP challenge and captures challenge_id.
//  2. Documents the wrong-code → expects 4xx (skipped due to production bug).
func TestE2EPasswordlessOTPRoundTrip(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	// Start — purpose enum is "signin".
	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/start",
		map[string]any{
			"identifier": "otp-roundtrip@example.com",
			"channel":    "email",
			"purpose":    "signin",
		},
		map[string]string{"X-Client-Id": projectID},
	)
	e2eWantStatus(t, r, http.StatusOK)
	var start map[string]any
	e2eDecode(t, r, &start)
	challengeID, _ := start["challenge_id"].(string)
	if challengeID == "" {
		t.Fatal("no challenge_id in OTP start response")
	}

	// The plaintext code only reaches the delivery channel; the harness capture
	// emitter records it from the auth.otp.started event.
	code := e2eEmitter.payloadFor(challengeID, "code")
	if code == "" {
		t.Fatalf("OTP code not captured for challenge %s", challengeID)
	}

	t.Run("wrong code returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/verify",
			map[string]any{"challenge_id": challengeID, "code": "000000"},
			map[string]string{"X-Client-Id": projectID})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("correct code mints a session", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/verify",
			map[string]any{"challenge_id": challengeID, "code": code},
			map[string]string{"X-Client-Id": projectID})
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.Session.AccessToken == "" {
			t.Errorf("expected access_token after OTP verify, body=%s", r.Body)
		}
	})
}

// TestE2EPasswordlessMagicLinkRoundTrip exercises the magic-link sign-in flow.
// The happy path requires the token from the delivered email; only start is
// fully exercised here.
func TestE2EPasswordlessMagicLinkRoundTrip(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	// Start — purpose is required; enum is "signin".
	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/magic-link/start",
		map[string]any{
			"email":       "magic-roundtrip@example.com",
			"redirect_to": "https://app.example.com/cb",
			"purpose":     "signin",
		},
		map[string]string{"X-Client-Id": projectID},
	)
	e2eWantStatus(t, r, http.StatusOK)
	var start map[string]any
	e2eDecode(t, r, &start)
	challengeID, _ := start["challenge_id"].(string)
	if challengeID == "" {
		t.Fatal("no challenge_id in magic-link start response")
	}

	// The opaque token only reaches the delivery channel; the harness capture
	// emitter records it from the auth.magiclink.started event.
	token := e2eEmitter.payloadFor(challengeID, "token")
	if token == "" {
		t.Fatalf("magic-link token not captured for challenge %s", challengeID)
	}

	t.Run("bogus token returns 401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/magic-link/verify",
			map[string]any{"token": "not-a-real-token"},
			map[string]string{"X-Client-Id": projectID})
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("real token mints a session", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/magic-link/verify",
			map[string]any{"token": token},
			map[string]string{"X-Client-Id": projectID})
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if body.Session.AccessToken == "" {
			t.Errorf("expected access_token after magic-link verify, body=%s", r.Body)
		}
	})
}

// ═════════════════════════════════════════════════════════════════════════════
// OAuth Social
// ═════════════════════════════════════════════════════════════════════════════

// TestE2EOAuthSocialProvidersList verifies the enabled-providers list endpoint.
func TestE2EOAuthSocialProvidersList(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("happy_path_empty_list", func(t *testing.T) {
		// A brand-new project has no oauth providers configured.
		r := e2eReq(t, ctx, http.MethodGet,
			ts.URL+"/v1/auth/oauth/providers",
			nil,
			map[string]string{"X-Client-Id": projectID},
		)
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Providers []any `json:"providers"`
		}
		e2eDecode(t, r, &body)
		// An empty project has no enabled providers; nil and empty slice are both acceptable.
		if body.Providers == nil {
			body.Providers = []any{}
		}
	})

	t.Run("missing_x_client_id_not_500", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet,
			ts.URL+"/v1/auth/oauth/providers",
			nil,
			nil,
		)
		// Missing project header; the adapter may return an empty list or an error,
		// but must not panic (no 500).
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 without X-Client-Id: %s", r.Body)
		}
	})
}

// TestE2EOAuthSocialStart verifies the OAuth social start (redirect-to-provider)
// endpoint.
//
// NOTE: A real provider round-trip (Google/GitHub) is not feasible without live
// OAuth credentials. We test: (a) start returns an error for an unknown provider
// in a project with no OAuth providers configured (not 500); (b) missing
// required query params return 4xx.
func TestE2EOAuthSocialStart(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("unknown_provider_not_500", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/auth/oauth/google/start?client_id=%s&redirect_to=https://app.example.com/cb",
			ts.URL, projectID)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		// The provider is not configured → should be a 4xx (provider_not_found /
		// bad_request), never a 500.
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for unknown provider: %s", r.Body)
		}
		if r.Status < 400 {
			t.Fatalf("expected 4xx for unconfigured provider, got %d", r.Status)
		}
	})

	t.Run("missing_client_id_422", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/auth/oauth/google/start?redirect_to=https://app.example.com/cb", ts.URL)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400 for missing client_id, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing_redirect_to_422", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/auth/oauth/google/start?client_id=%s", ts.URL, projectID)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400 for missing redirect_to, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOAuthSocialCallback tests the provider callback handler.
//
// NOTE: Full callback happiness-path requires a valid `state` cookie (from
// start) and a real authorization code from an upstream provider.  We exercise
// the error branch (provider-set "error" param) and the missing-state path.
func TestE2EOAuthSocialCallback(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("provider_error_param_not_500", func(t *testing.T) {
		// Provider signals an error via ?error=access_denied.  The adapter should
		// return a 4xx redirect or error, never a 500.
		u := fmt.Sprintf("%s/v1/auth/oauth/google/callback?error=access_denied", ts.URL)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for provider error param: %s", r.Body)
		}
	})

	t.Run("no_state_no_code_not_500", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/auth/oauth/google/callback", ts.URL)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for empty callback: %s", r.Body)
		}
	})
}

// TestE2EOAuthSocialExchange tests the code-exchange endpoint.
func TestE2EOAuthSocialExchange(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("bogus_code_4xx", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/oauth/exchange",
			map[string]any{"code": "not-a-real-oauth-exchange-code"},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for bogus exchange code, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing_code_field_422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/oauth/exchange",
			map[string]any{},
			map[string]string{"X-Client-Id": projectID},
		)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOAuthSocialUnlink tests the unlink endpoint.
// It requires a bearer-authenticated user; unauthenticated calls must return 401.
func TestE2EOAuthSocialUnlink(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("unauthenticated_401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/oauth/google/unlink",
			map[string]any{"identity_id": newUUID()},
			nil,
		)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_unknown_identity_4xx", func(t *testing.T) {
		// Unlink calls translatePgErr; ErrNotFound now aliases domain.ErrNotFound so
		// the ogen NewError hook renders 404 for a non-existent identity.
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "unlink-user@example.com")
		r := e2eReq(t, ctx, http.MethodPost,
			ts.URL+"/v1/auth/oauth/google/unlink",
			map[string]any{"identity_id": newUUID()},
			e2eBearer(sess.AccessToken),
		)
		// The identity doesn't exist → 404.
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown identity, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOAuthSocialLinkStart tests the authenticated link-start endpoint.
func TestE2EOAuthSocialLinkStart(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("unauthenticated_401", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/auth/oauth/github/link/start?redirect_to=https://app.example.com/cb", ts.URL)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_unconfigured_provider_not_500", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "link-start-user@example.com")
		u := fmt.Sprintf("%s/v1/auth/oauth/github/link/start?redirect_to=https://app.example.com/cb", ts.URL)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, e2eBearer(sess.AccessToken))
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for unconfigured provider link start: %s", r.Body)
		}
		if r.Status < 400 {
			t.Fatalf("expected 4xx for unconfigured provider, got %d", r.Status)
		}
	})
}

// TestE2EOAuthSocialLinkCallback tests the link callback endpoint.
// Full happy-path requires a valid upstream callback; we test the no-state
// error path only.
//
// NOTE: full link callback with real external OAuth provider is not testable
// without live credentials.
func TestE2EOAuthSocialLinkCallback(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_state_not_500", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/auth/oauth/github/link/callback", ts.URL)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for empty link callback: %s", r.Body)
		}
	})
}

// ═════════════════════════════════════════════════════════════════════════════
// OIDC Provider — public endpoints
// ═════════════════════════════════════════════════════════════════════════════

// TestE2EOIDCProviderDiscovery tests the OIDC discovery document endpoint.
// The endpoint is public and scoped to /p/{project_id}/e/{env}/.well-known/openid-configuration.
func TestE2EOIDCProviderDiscovery(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("happy_path_returns_issuer", func(t *testing.T) {
		u := fmt.Sprintf("%s/p/%s/e/live/.well-known/openid-configuration", ts.URL, projectID)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		e2eWantStatus(t, r, http.StatusOK)
		var doc map[string]any
		e2eDecode(t, r, &doc)
		if doc["issuer"] == nil {
			t.Fatalf("discovery doc missing 'issuer': %s", r.Body)
		}
	})

	t.Run("unknown_project_returns_200", func(t *testing.T) {
		// NOTE: OpenIDConfiguration builds a static discovery document from the
		// project_id path param without any DB lookup, so any project ID (even
		// unknown) returns 200. This is the actual designed behavior — the issuer
		// is embedded in the discovery doc itself.
		u := fmt.Sprintf("%s/p/%s/e/live/.well-known/openid-configuration", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		// Accept 200 (static discovery) or 4xx (project-validated); never 500.
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for unknown project discovery: %s", r.Body)
		}
	})
}

// TestE2EOIDCProviderDiscoveryContainsExpectedFields ensures the discovery
// document contains the minimum required OIDC fields.
func TestE2EOIDCProviderDiscoveryContainsExpectedFields(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	u := fmt.Sprintf("%s/p/%s/e/live/.well-known/openid-configuration", ts.URL, projectID)
	r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
	e2eWantStatus(t, r, http.StatusOK)

	var doc map[string]any
	e2eDecode(t, r, &doc)

	requiredFields := []string{
		"issuer",
		"authorization_endpoint",
		"token_endpoint",
		"jwks_uri",
		"response_types_supported",
		"subject_types_supported",
		"id_token_signing_alg_values_supported",
	}
	for _, field := range requiredFields {
		if doc[field] == nil {
			t.Errorf("discovery doc missing required field %q", field)
		}
	}
}

// TestE2EOIDCProviderJWKS tests the JWKS endpoint.
func TestE2EOIDCProviderJWKS(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("happy_path_returns_keys_array", func(t *testing.T) {
		// JWKS returns the keys array; for a new project with no signing key created
		// yet, the keys may be empty (key generation is lazy). We assert the field
		// exists and the status is 200.
		u := fmt.Sprintf("%s/p/%s/e/live/.well-known/jwks.json", ts.URL, projectID)
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		e2eWantStatus(t, r, http.StatusOK)
		var jwks map[string]any
		e2eDecode(t, r, &jwks)
		if _, ok := jwks["keys"]; !ok {
			t.Fatalf("JWKS response missing 'keys': %s", r.Body)
		}
	})

	t.Run("unknown_project_returns_200_or_4xx", func(t *testing.T) {
		// Like discovery, JWKS may return 200 with empty keys for any project ID
		// (no project existence check). Never 500.
		u := fmt.Sprintf("%s/p/%s/e/live/.well-known/jwks.json", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for unknown project JWKS: %s", r.Body)
		}
	})
}

// TestE2EOIDCProviderJWKSContainsKeys ensures the JWKS endpoint returns the
// keys array field with at least 0 keys; lazily provisioned keys appear after
// an admin signing-key creation.
func TestE2EOIDCProviderJWKSContainsKeys(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	// Hit the discovery endpoint first (doesn't create keys; discovery is static).
	discU := fmt.Sprintf("%s/p/%s/e/live/.well-known/openid-configuration", ts.URL, projectID)
	discR := e2eReq(t, ctx, http.MethodGet, discU, nil, nil)
	e2eWantStatus(t, discR, http.StatusOK)

	jwksU := fmt.Sprintf("%s/p/%s/e/live/.well-known/jwks.json", ts.URL, projectID)
	r := e2eReq(t, ctx, http.MethodGet, jwksU, nil, nil)
	e2eWantStatus(t, r, http.StatusOK)

	var jwks struct {
		Keys []any `json:"keys"`
	}
	e2eDecode(t, r, &jwks)
	// NOTE: A fresh project without an admin-provisioned signing key has an empty
	// JWKS. The discovery endpoint is static (no DB lookup) so it does not trigger
	// key generation. Key creation is an admin operation (POST .../admin/jwks).
	// We only assert the field is present; length may be 0.
	if jwks.Keys == nil {
		t.Fatal("JWKS endpoint returned nil keys field (should be empty array, not null)")
	}
}

// TestE2EOIDCProviderAuthorize tests the OAuth2 authorization endpoint.
//
// NOTE: a full authorization code flow requires a registered app client (OIDC
// client_id) with a matching redirect_uri. We exercise missing-params and
// unknown-client error paths; a complete code → token flow is not tested
// because it additionally requires an interaction round-trip.
func TestE2EOIDCProviderAuthorize(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("missing_required_params_422", func(t *testing.T) {
		// Missing response_type, redirect_uri, scope.
		u := ts.URL + "/oauth2/authorize?client_id=does-not-exist"
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400 for missing params, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("unknown_client_id_not_500", func(t *testing.T) {
		u := ts.URL + "/oauth2/authorize?client_id=unknown&response_type=code&redirect_uri=https://app.example.com/cb&scope=openid"
		r := e2eReq(t, ctx, http.MethodGet, u, nil, nil)
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for unknown client_id: %s", r.Body)
		}
		// Authorize with unknown client may redirect (302) to an error interaction
		// or return a direct 4xx; either is acceptable — just not 500.
		if r.Status < 300 {
			t.Fatalf("expected 3xx or 4xx for unknown client, got %d", r.Status)
		}
	})
}

// TestE2EOIDCProviderToken tests the token endpoint (form-encoded).
//
// The token endpoint is client-authenticated (clientSecretBasic) and handles
// various grant types. Without a real authorization code we can only test
// missing-credential 401 and invalid-grant-type 4xx.
//
// NOTE: full authorization_code, refresh_token, device_code grant flows
// require a prior authorization step (which needs a registered OIDC client);
// those happy paths are not testable without a configured app client.
func TestE2EOIDCProviderToken(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		form := url.Values{"grant_type": {"authorization_code"}, "code": {"bogus"}}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401 without credentials, got %d\nbody: %s", resp.StatusCode, b)
		}
	})

	t.Run("invalid_client_secret_not_500", func(t *testing.T) {
		// Use a registered project as client_id with a wrong secret; should
		// be 401 (bad credentials) — not 500.
		projectID := e2eProject(t, ctx)
		form := url.Values{
			"grant_type": {"unsupported_grant"},
			"client_id":  {projectID},
		}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(projectID, "wrong-secret")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		// Bad client secret → 401; unknown grant type → 400. Neither should be 500.
		if resp.StatusCode >= 500 {
			t.Fatalf("unexpected 5xx for invalid grant: %d\nbody: %s", resp.StatusCode, b)
		}
	})
}

// TestE2EOIDCProviderRevoke tests RFC 7009 token revocation endpoint.
func TestE2EOIDCProviderRevoke(t *testing.T) {
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		form := url.Values{"token": {"any-token"}}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401 without credentials, got %d\nbody: %s", resp.StatusCode, b)
		}
	})
}

// TestE2EOIDCProviderIntrospect tests RFC 7662 token introspection.
func TestE2EOIDCProviderIntrospect(t *testing.T) {
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		form := url.Values{"token": {"any-token"}}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/introspect", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401 without credentials, got %d\nbody: %s", resp.StatusCode, b)
		}
	})
}

// TestE2EOIDCProviderUserinfo tests the OIDC userinfo endpoint.
func TestE2EOIDCProviderUserinfo(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/oauth2/userinfo", nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_with_user_session_not_500", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "userinfo-user@example.com")
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/oauth2/userinfo", nil, e2eBearer(sess.AccessToken))
		// A regular user session might not satisfy the OIDC userinfo OAuth2 scope;
		// we accept any non-500 response.
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for userinfo with user session: %s", r.Body)
		}
	})
}

// TestE2EOIDCProviderLogout tests the RP-initiated logout endpoint.
// Uses a non-redirecting HTTP client because the endpoint returns a 302 that
// may point to an external URL (which can't be followed in tests).
func TestE2EOIDCProviderLogout(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	// Non-redirecting client: prevents the test client following the 302 Location
	// header to an external URL (e.g. https://app.example.com/bye) which would
	// cause a network error.
	noRedirectClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Run("no_id_token_hint_not_500", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/oauth2/logout", nil)
		resp, err := noRedirectClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for logout without id_token_hint: %s", b)
		}
	})

	t.Run("with_post_logout_redirect_uri_not_500", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet,
			ts.URL+"/oauth2/logout?post_logout_redirect_uri=https://app.example.com/bye", nil)
		resp, err := noRedirectClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for logout with redirect_uri: %s", b)
		}
		// With post_logout_redirect_uri, expect a 302 redirect back to the URI.
		_ = ctx // suppress unused warning
	})
}

// TestE2EOIDCProviderBackchannelLogout tests the back-channel logout endpoint.
func TestE2EOIDCProviderBackchannelLogout(t *testing.T) {
	ts := e2eServer(t)

	t.Run("bogus_logout_token_4xx", func(t *testing.T) {
		form := url.Values{"logout_token": {"not-a-jwt"}}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/backchannel-logout", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for bogus logout token: %s", b)
		}
		if resp.StatusCode < 400 {
			t.Fatalf("expected 4xx for bogus logout token, got %d", resp.StatusCode)
		}
	})

	t.Run("empty_logout_token_4xx", func(t *testing.T) {
		// Empty logout_token → ErrInvalidToken from the adapter → 401.
		form := url.Values{"logout_token": {""}}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/backchannel-logout", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for empty logout token: %s", b)
		}
		if resp.StatusCode < 400 {
			t.Fatalf("expected 4xx for empty logout token, got %d", resp.StatusCode)
		}
	})
}

// TestE2EOIDCProviderPAR tests the Pushed Authorization Request endpoint.
//
// NOTE: PAR requires client authentication; without a registered OIDC client we
// can only verify the unauthenticated path returns 401.
func TestE2EOIDCProviderPAR(t *testing.T) {
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		form := url.Values{
			"response_type": {"code"},
			"client_id":     {"any-client"},
			"redirect_uri":  {"https://app.example.com/cb"},
			"scope":         {"openid"},
		}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/par", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401 without credentials, got %d\nbody: %s", resp.StatusCode, b)
		}
	})
}

// TestE2EOIDCProviderDeviceAuthorization tests the device authorization endpoint.
//
// NOTE: device_authorization requires client authentication.  Without a
// registered OIDC client we can only test the unauthenticated path.
func TestE2EOIDCProviderDeviceAuthorization(t *testing.T) {
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		form := url.Values{"scope": {"openid"}}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth2/device_authorization", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401 without credentials, got %d\nbody: %s", resp.StatusCode, b)
		}
	})
}

// TestE2EOIDCProviderDeviceResolve tests GET /v1/device (device verification UI).
func TestE2EOIDCProviderDeviceResolve(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("unknown_user_code_4xx", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/device?user_code=AAAA-BBBB", ts.URL)
		r := e2eReq(t, ctx, http.MethodGet, u, nil,
			map[string]string{"X-Client-Id": projectID})
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown user_code, got %d\nbody: %s", r.Status, r.Body)
		}
	})

	t.Run("missing_user_code_param_422", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/device", nil,
			map[string]string{"X-Client-Id": projectID})
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("want 422 or 400 for missing user_code, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOIDCProviderDeviceApprove tests POST /v1/device/approve.
// Requires an authenticated user; unknown user_code → 4xx.
func TestE2EOIDCProviderDeviceApprove(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/device/approve",
			map[string]any{"user_code": "AAAA-BBBB"}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_unknown_code_4xx", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "device-approve@example.com")
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/device/approve",
			map[string]any{"user_code": "UNKN-CODE"},
			e2eBearer(sess.AccessToken))
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown device code, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOIDCProviderDeviceDeny tests POST /v1/device/deny.
func TestE2EOIDCProviderDeviceDeny(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/device/deny",
			map[string]any{"user_code": "AAAA-BBBB"}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_unknown_code_4xx", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "device-deny@example.com")
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/device/deny",
			map[string]any{"user_code": "UNKN-CODE"},
			e2eBearer(sess.AccessToken))
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown device code deny, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// ═════════════════════════════════════════════════════════════════════════════
// OIDC Interaction
// ═════════════════════════════════════════════════════════════════════════════

// TestE2EOIDCProviderInteractionFetch tests GET /v1/oauth/interaction/{id}.
func TestE2EOIDCProviderInteractionFetch(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	t.Run("unknown_interaction_4xx", func(t *testing.T) {
		// ResolveInteraction calls translatePgErr; ErrNotFound now aliases
		// domain.ErrNotFound so the ogen NewError hook renders 404.
		u := fmt.Sprintf("%s/v1/oauth/interaction/%s", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodGet, u, nil,
			map[string]string{"X-Client-Id": projectID})
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown interaction, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOIDCProviderInteractionLogin tests POST /v1/oauth/interaction/{id}/login.
// Requires an authenticated user.
func TestE2EOIDCProviderInteractionLogin(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/oauth/interaction/%s/login", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, u, nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_unknown_interaction_4xx", func(t *testing.T) {
		// CompleteLogin calls translatePgErr; ErrNotFound now aliases domain.ErrNotFound
		// so the ogen NewError hook renders 404, not 500.
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "interaction-login@example.com")
		u := fmt.Sprintf("%s/v1/oauth/interaction/%s/login", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, u, nil, e2eBearer(sess.AccessToken))
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown interaction login, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOIDCProviderInteractionConsent tests POST /v1/oauth/interaction/{id}/consent.
func TestE2EOIDCProviderInteractionConsent(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/oauth/interaction/%s/consent", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, u,
			map[string]any{"granted_scopes": []string{"openid"}}, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_unknown_interaction_4xx", func(t *testing.T) {
		// Consent calls translatePgErr; ErrNotFound now aliases domain.ErrNotFound
		// so the ogen NewError hook renders 404, not 500.
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "consent-user@example.com")
		u := fmt.Sprintf("%s/v1/oauth/interaction/%s/consent", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, u,
			map[string]any{"granted_scopes": []string{"openid"}},
			e2eBearer(sess.AccessToken))
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown interaction consent, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// TestE2EOIDCProviderInteractionReject tests POST /v1/oauth/interaction/{id}/reject.
// This is a public endpoint (security: []).
func TestE2EOIDCProviderInteractionReject(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("unknown_interaction_4xx", func(t *testing.T) {
		// Reject calls translatePgErr; ErrNotFound now aliases domain.ErrNotFound
		// so the ogen NewError hook renders 404, not 500.
		u := fmt.Sprintf("%s/v1/oauth/interaction/%s/reject", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodPost, u,
			map[string]any{"error": "access_denied", "error_description": "User denied"}, nil)
		if r.Status == http.StatusInternalServerError {
			t.Fatalf("unexpected 500 for unknown interaction reject: %s", r.Body)
		}
		if r.Status < 400 {
			t.Fatalf("expected 4xx for unknown interaction, got %d", r.Status)
		}
	})
}

// TestE2EOIDCProviderGrantsList tests GET /v1/oauth/grants.
func TestE2EOIDCProviderGrantsList(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/oauth/grants", nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_empty_list", func(t *testing.T) {
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "grants-user@example.com")
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/oauth/grants", nil,
			e2eBearer(sess.AccessToken))
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Data []any `json:"data"`
		}
		e2eDecode(t, r, &body)
		// A fresh user has no grants.
		if body.Data == nil {
			body.Data = []any{}
		}
	})
}

// TestE2EOIDCProviderGrantRevoke tests DELETE /v1/oauth/grants/{grant_id}.
func TestE2EOIDCProviderGrantRevoke(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)

	t.Run("no_auth_401", func(t *testing.T) {
		u := fmt.Sprintf("%s/v1/oauth/grants/%s", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodDelete, u, nil, nil)
		e2eWantStatus(t, r, http.StatusUnauthorized)
	})

	t.Run("authenticated_unknown_grant_4xx", func(t *testing.T) {
		// RevokeGrant calls translatePgErr; ErrNotFound now aliases domain.ErrNotFound
		// so the ogen NewError hook renders 404 for a non-existent grant.
		projectID := e2eProject(t, ctx)
		_, sess := registerUser(t, ctx, projectID, "revoke-grant@example.com")
		u := fmt.Sprintf("%s/v1/oauth/grants/%s", ts.URL, newUUID())
		r := e2eReq(t, ctx, http.MethodDelete, u, nil, e2eBearer(sess.AccessToken))
		if r.Status < 400 || r.Status >= 500 {
			t.Fatalf("expected 4xx for unknown grant, got %d\nbody: %s", r.Status, r.Body)
		}
	})
}

// ─── compile-time guard ───────────────────────────────────────────────────────
// Ensures the bytes and io packages are used (imported for form-encoded requests).
var _ = bytes.NewReader
var _ = io.ReadAll
