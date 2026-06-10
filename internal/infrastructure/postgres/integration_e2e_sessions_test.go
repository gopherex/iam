//go:build integration

package postgres

import (
	"context"
	"net/http"
	"testing"
)

// TestE2ESelfManagedSessions verifies a session captures device metadata at
// sign-in (IP / User-Agent) and that the account session list exposes it plus
// the `current` marker — the foundation for an in-app session manager.
func TestE2ESelfManagedSessions(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := "sess-" + newUUID()[:8] + "@example.com"

	// Sign up over HTTP so the request-meta middleware captures the device.
	hdr := map[string]string{
		"X-Client-Id":          projectID,
		"X-Environment":        "live",
		"User-Agent":           "ACME-App/2.1 (iPhone; iOS 18)",
		"X-Forwarded-For":      "203.0.113.7, 10.0.0.1",
		"X-Device-Fingerprint": "fp-abc123",
	}
	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-up",
		map[string]any{"email": email, "password": "Sup3rStr0ng!Pass", "name": "Sess"}, hdr)
	e2eWantStatus(t, r, http.StatusOK)
	var signup struct {
		Session struct {
			AccessToken string `json:"access_token"`
		} `json:"session"`
	}
	e2eDecode(t, r, &signup)
	if signup.Session.AccessToken == "" {
		t.Fatalf("no access token, body: %s", r.Body)
	}
	bearer := map[string]string{"Authorization": "Bearer " + signup.Session.AccessToken, "X-Environment": "live"}

	t.Run("list sessions exposes device metadata + current", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/sessions", nil, bearer)
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Data []struct {
				ID           string `json:"id"`
				IP           string `json:"ip"`
				UserAgent    string `json:"user_agent"`
				Current      bool   `json:"current"`
				LastActiveAt string `json:"last_active_at"`
			} `json:"data"`
		}
		e2eDecode(t, r, &body)
		if len(body.Data) == 0 {
			t.Fatal("expected at least one session")
		}
		var cur *struct {
			ID           string `json:"id"`
			IP           string `json:"ip"`
			UserAgent    string `json:"user_agent"`
			Current      bool   `json:"current"`
			LastActiveAt string `json:"last_active_at"`
		}
		for i := range body.Data {
			if body.Data[i].Current {
				cur = &body.Data[i]
			}
		}
		if cur == nil {
			t.Fatalf("no current session flagged; body: %s", r.Body)
		}
		if cur.IP != "203.0.113.7" {
			t.Errorf("ip = %q, want 203.0.113.7 (first X-Forwarded-For hop)", cur.IP)
		}
		if cur.UserAgent != "ACME-App/2.1 (iPhone; iOS 18)" {
			t.Errorf("user_agent = %q", cur.UserAgent)
		}
		if cur.LastActiveAt == "" {
			t.Error("last_active_at not set")
		}
	})

	t.Run("current session endpoint", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/sessions/current", nil, bearer)
		e2eWantStatus(t, r, http.StatusOK)
		var body struct {
			Session struct {
				Current   bool   `json:"current"`
				IP        string `json:"ip"`
				UserAgent string `json:"user_agent"`
			} `json:"session"`
		}
		e2eDecode(t, r, &body)
		if !body.Session.Current {
			t.Error("current session must be flagged current=true")
		}
		if body.Session.IP != "203.0.113.7" {
			t.Errorf("current ip = %q", body.Session.IP)
		}
	})
}
