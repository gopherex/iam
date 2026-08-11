//go:build integration

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestE2EMagicLinkCallback exercises the browser GET leg of a magic link: it
// consumes the token, sets session cookies and 302-redirects to the sanitized
// target.
func TestE2EMagicLinkCallback(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("ml-cb-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	// Start a magic link and recover the emitted token.
	rs := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/magic-link/start",
		map[string]any{"email": email, "purpose": "signin", "redirect_to": "/welcome"},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	e2eWantStatus(t, rs, http.StatusOK)

	var chal struct {
		ChallengeID string `json:"challenge_id"`
	}
	e2eDecode(t, rs, &chal)

	token := e2eEmitter.payloadFor(chal.ChallengeID, "token")
	if token == "" {
		t.Fatalf("no magic-link token captured for challenge %s", chal.ChallengeID)
	}

	// GET the callback with a non-following client so the 302 is observable.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		ts.URL+"/v1/auth/magic-link/callback?token="+token+"&redirect_to=/welcome", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("callback status = %d, want 302", resp.StatusCode)
	}

	if loc := resp.Header.Get("Location"); !strings.Contains(loc, "/welcome") {
		t.Fatalf("Location = %q, want it to contain /welcome", loc)
	}

	if len(resp.Header.Values("Set-Cookie")) == 0 {
		t.Fatal("callback did not set session cookies")
	}

	// The token is single-use: a second callback fails.
	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		ts.URL+"/v1/auth/magic-link/callback?token="+token, nil)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}

	resp2.Body.Close()

	if resp2.StatusCode == http.StatusFound {
		t.Fatal("magic-link token was reusable via the callback")
	}
}
