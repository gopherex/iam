//go:build integration

package postgres

import (
	"context"
	"net/http"
	"testing"
)

// TestE2EAdminSendTestSMS covers the admin SMS test-send endpoint: 422 when no
// SMS provider is enabled, 200 once one exists.
func TestE2EAdminSendTestSMS(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx, "project")
	url := ts.URL + "/v1/projects/" + projectID + "/admin/sms-providers/send-test"
	body := map[string]any{"to": "+14155552671", "template_id": "otp"}

	// No provider yet → 422.
	r := e2eReq(t, ctx, http.MethodPost, url, body, e2eBearer(token))
	if r.Status != http.StatusUnprocessableEntity {
		t.Fatalf("no provider: status = %d, want 422; body=%s", r.Status, r.Body)
	}

	// With an enabled SMS provider → 200.
	e2eEnableSMSProvider(t, ctx, projectID)
	r2 := e2eReq(t, ctx, http.MethodPost, url, body, e2eBearer(token))
	if r2.Status != http.StatusOK {
		t.Fatalf("with provider: status = %d, want 200; body=%s", r2.Status, r2.Body)
	}
}

// TestE2EFlowSigninPasskeyOAuthValidation covers create-time validation for the
// passkey and oauth signin methods (happy-path assertion/redirect exchange needs
// WebAuthn / OAuth-provider fixtures and is covered by their dedicated suites).
func TestE2EFlowSigninPasskeyOAuthValidation(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)

	// passkey without email → 400.
	_, r := flowCreate(t, ctx, ts, projectID, map[string]any{"kind": "signin", "method": "passkey"})
	if r.Status != http.StatusBadRequest {
		t.Fatalf("passkey no-email: status = %d, want 400; body=%s", r.Status, r.Body)
	}

	// oauth without provider → 400.
	_, r2 := flowCreate(t, ctx, ts, projectID, map[string]any{"kind": "signin", "method": "oauth"})
	if r2.Status != http.StatusBadRequest {
		t.Fatalf("oauth no-provider: status = %d, want 400; body=%s", r2.Status, r2.Body)
	}
}
