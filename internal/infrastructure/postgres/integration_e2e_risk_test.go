//go:build integration

package postgres

// integration_e2e_risk_test.go — adaptive step-up from risk rules.
//
// The point of the narrow signal set is that a rule an administrator writes
// either evaluates or is refused when it is written. So the tests are: a bad
// rule is refused, a good one fires, and firing changes the sign-in.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func e2eRiskRule(
	t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token string, body map[string]any,
) e2eResp {
	t.Helper()

	return e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/risk/rules", ts.URL, projectID), body, e2eBearer(token))
}

func TestE2ERiskRulesAreValidated(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	t.Run("unknown_signal_is_refused", func(t *testing.T) {
		r := e2eRiskRule(t, ctx, ts, projectID, adminToken, map[string]any{
			"name": "impossible travel", "signal": "impossible_travel", "action": "require_step_up",
		})
		if r.Status < 400 {
			t.Fatalf("status = %d, want a 4xx — a rule that could never fire was stored", r.Status)
		}
	})

	t.Run("unknown_action_is_refused", func(t *testing.T) {
		r := e2eRiskRule(t, ctx, ts, projectID, adminToken, map[string]any{
			"name": "nope", "signal": "new_ip", "action": "quarantine",
		})
		if r.Status < 400 {
			t.Fatalf("status = %d, want a 4xx for an unknown action", r.Status)
		}
	})

	t.Run("known_signal_is_accepted", func(t *testing.T) {
		r := e2eRiskRule(t, ctx, ts, projectID, adminToken, map[string]any{
			"name": "unfamiliar address", "signal": "new_ip", "action": "notify", "enabled": true,
		})
		e2eWantStatus(t, r, http.StatusCreated)
	})

	// The 1.4 spelling still works: a rule written then must keep evaluating.
	t.Run("condition_is_still_accepted", func(t *testing.T) {
		r := e2eRiskRule(t, ctx, ts, projectID, adminToken, map[string]any{
			"name": "legacy", "condition": "new_device", "action": "notify",
		})
		e2eWantStatus(t, r, http.StatusCreated)

		list := e2eReq(t, ctx, http.MethodGet,
			fmt.Sprintf("%s/v1/projects/%s/admin/risk/rules", ts.URL, projectID), nil, e2eBearer(adminToken))
		e2eWantStatus(t, list, http.StatusOK)

		if !containsSignal(t, list.Body, "new_device") {
			t.Errorf("a rule written with `condition` does not read back with a signal: %s", list.Body)
		}
	})
}

// TestE2ERiskRuleBlocksASignIn: `block` has to actually refuse.
func TestE2ERiskRuleBlocksASignIn(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	// Every sign-in in the test arrives from an address the account has never
	// used, because the account has no sessions yet.
	r := e2eRiskRule(t, ctx, ts, projectID, adminToken, map[string]any{
		"name": "unknown address", "signal": "new_ip", "action": "block", "enabled": true,
	})
	e2eWantStatus(t, r, http.StatusCreated)

	email := fmt.Sprintf("risk-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)
	tenant := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

	status, body := b.do(t, ctx, http.MethodPost, "/v1/auth/flows", map[string]any{
		"kind": "signin", "email": email, "password": "Sup3rStr0ng!Pass",
	}, tenant)
	if status < 400 {
		t.Fatalf("sign in: status %d, want a 4xx — the blocking rule did not fire: %s", status, body)
	}

	// And the firing is on the record.
	events := e2eReq(t, ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/projects/%s/admin/risk/events", ts.URL, projectID), nil, e2eBearer(adminToken))
	e2eWantStatus(t, events, http.StatusOK)

	if !containsSignal(t, events.Body, "new_ip") {
		t.Errorf("no risk event recorded for the rule that fired: %s", events.Body)
	}
}

// TestE2ERiskRuleDisabledDoesNothing: an administrator switching a rule off has
// to be able to rely on that.
func TestE2ERiskRuleDisabledDoesNothing(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	r := e2eRiskRule(t, ctx, ts, projectID, adminToken, map[string]any{
		"name": "off", "signal": "new_ip", "action": "block", "enabled": false,
	})
	e2eWantStatus(t, r, http.StatusCreated)

	email := fmt.Sprintf("risk-off-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)

	status, body := b.do(t, ctx, http.MethodPost, "/v1/auth/flows", map[string]any{
		"kind": "signin", "email": email, "password": "Sup3rStr0ng!Pass",
	}, map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	if status != http.StatusOK {
		t.Fatalf("sign in: status %d, body %s — a disabled rule blocked it", status, body)
	}
}

// containsSignal reports whether a JSON body mentions the given signal value.
func containsSignal(t *testing.T, body []byte, signal string) bool {
	t.Helper()

	var any1 any
	if err := json.Unmarshal(body, &any1); err != nil {
		t.Fatalf("decode: %v", err)
	}

	return jsonMentions(any1, signal)
}

// jsonMentions walks a decoded document looking for a string value.
func jsonMentions(v any, want string) bool {
	switch t := v.(type) {
	case string:
		return t == want
	case []any:
		for _, item := range t {
			if jsonMentions(item, want) {
				return true
			}
		}
	case map[string]any:
		for _, item := range t {
			if jsonMentions(item, want) {
				return true
			}
		}
	}

	return false
}
