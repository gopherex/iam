//go:build integration

package postgres

// integration_e2e_clock_test.go — the test clock.
//
// Half of what an auth server does is time-dependent, and none of it is testable
// if the only way to reach an expiry is to wait for it. The clock is what makes
// that reachable, so the tests are: it expires things, and it cannot touch live.

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestE2ETestClockExpiresAChallenge(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)
	e2eCreateEnvironment(t, ctx, projectID, "test")

	testEnv := map[string]string{"X-Client-Id": projectID, "X-Environment": "test"}

	// A code that is valid right now.
	start := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/start",
		map[string]any{"identifier": "clock@example.com", "channel": "email", "purpose": "signin"},
		testEnv)
	e2eWantStatus(t, start, http.StatusOK)

	var challenge struct {
		ChallengeID string `json:"challenge_id"`
	}
	e2eDecode(t, start, &challenge)

	if challenge.ChallengeID == "" {
		t.Fatalf("no challenge: %s", start.Body)
	}

	// Move the environment's clock past the challenge's lifetime.
	adv := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/clock",
		map[string]any{"advance_seconds": 3600},
		map[string]string{
			"Authorization": "Bearer " + adminToken,
			"X-Environment": "test",
			"X-Client-Id":   projectID,
		})
	e2eWantStatus(t, adv, http.StatusOK)

	// The code is now past its expiry, so verifying it must fail as expired
	// rather than as a wrong code.
	verify := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/otp/verify",
		map[string]any{"challenge_id": challenge.ChallengeID, "code": "000000"}, testEnv)
	if verify.Status < 400 {
		t.Fatalf("verify after advancing the clock: status %d, want a 4xx", verify.Status)
	}

	if !containsAny(string(verify.Body), "expired", "flow_expired", "challenge_expired") {
		t.Errorf("error was %s — advancing the clock did not expire the challenge", verify.Body)
	}
}

// TestE2ETestClockRefusesLive: the clock exists so tests can move time. Live
// time is not a thing a caller gets to move.
func TestE2ETestClockRefusesLive(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)
	e2eCreateEnvironment(t, ctx, projectID, "test")

	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/clock",
		map[string]any{"advance_seconds": 3600},
		map[string]string{
			"Authorization": "Bearer " + adminToken,
			"X-Environment": "live",
			"X-Client-Id":   projectID,
		})
	if r.Status < 400 {
		t.Fatalf("status = %d, want a 4xx — the live clock was moved", r.Status)
	}

	// And an offset stored for one environment must not leak into another.
	adv := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/clock",
		map[string]any{"advance_seconds": 3600},
		map[string]string{
			"Authorization": "Bearer " + adminToken,
			"X-Environment": "test",
			"X-Client-Id":   projectID,
		})
	e2eWantStatus(t, adv, http.StatusOK)

	live := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/config/public", nil,
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	e2eWantStatus(t, live, http.StatusOK)
}

// TestE2ETestClockResets: an offset has to be removable, or an environment is
// stuck in the future.
func TestE2ETestClockResets(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)
	e2eCreateEnvironment(t, ctx, projectID, "test")

	headers := map[string]string{
		"Authorization": "Bearer " + adminToken,
		"X-Environment": "test",
		"X-Client-Id":   projectID,
	}

	// Well under the admin token's own hour: the clock moves everything in the
	// environment, the caller's credential included.
	for _, body := range []map[string]any{{"advance_seconds": 1800}, {"reset": true}} {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/test/clock", body, headers)
		e2eWantStatus(t, r, http.StatusOK)
	}

	offset, err := NewPgTestMode(testDB, nopEmitter{}).ClockOffset(ctx, projectID, "test")
	if err != nil {
		t.Fatalf("read offset: %v", err)
	}

	if offset != 0 {
		t.Fatalf("offset after reset = %v, want 0", offset)
	}
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, wants ...string) bool {
	for _, w := range wants {
		if strings.Contains(s, w) {
			return true
		}
	}

	return false
}
