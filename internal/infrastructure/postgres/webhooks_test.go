package postgres

import (
	"crypto/hmac"
	"reflect"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

func TestWebhookSignature(t *testing.T) {
	body := []byte(`{"id":"evt-1"}`)
	got := webhookSignature("secret", "evt-1", 123, body)
	if got == "" {
		t.Fatal("signature is empty")
	}
	if !hmac.Equal([]byte(got), []byte(webhookSignature("secret", "evt-1", 123, body))) {
		t.Fatal("signature is not deterministic")
	}
	if hmac.Equal([]byte(got), []byte(webhookSignature("other", "evt-1", 123, body))) {
		t.Fatal("signature does not depend on the secret")
	}
}

func TestPublicSessionRevokedPayloadIsExact(t *testing.T) {
	event, userID, ok := publicEventFromDomain(domain.Event{
		ID: "evt-1", Type: domain.WebhookEventSessionRevoked, ProjectID: "p1",
		Payload: domain.SessionRevokedPayload{SessionID: "s1", UserID: "u1", ProjectID: "p1"},
	})
	if !ok || userID != "u1" {
		t.Fatalf("event not normalized: ok=%v user=%q", ok, userID)
	}
	want := map[string]any{"session_id": "s1", "user_id": "u1", "project_id": "p1"}
	if !reflect.DeepEqual(event.Data, want) {
		t.Fatalf("data = %#v, want %#v", event.Data, want)
	}
	if _, _, ok := publicEventFromDomain(domain.Event{
		Type: domain.WebhookEventSessionRevoked, ProjectID: "p1",
		Payload: domain.SessionRevokedPayload{SessionID: "s1", ProjectID: "p1"},
	}); ok {
		t.Fatal("session.revoked without user_id became public")
	}
}

func TestValidateWebhookURL(t *testing.T) {
	for _, valid := range []string{"https://hooks.example.com/iam", "http://127.0.0.1:8080/hook", "http://localhost/hook"} {
		if err := validateWebhookURL(valid); err != nil {
			t.Fatalf("%s: %v", valid, err)
		}
	}
	for _, invalid := range []string{"http://example.com/hook", "https://user:pass@example.com/hook", "/relative", "ftp://example.com/hook"} {
		if err := validateWebhookURL(invalid); err == nil {
			t.Fatalf("%s: expected validation error", invalid)
		}
	}
}

func TestPublicEventSanitizesPayload(t *testing.T) {
	now := time.Now().UTC()
	event, userID, ok := publicEventFromDomain(domain.Event{
		ID: "evt-1", Type: domain.WebhookEventEmailChanged, ProjectID: "p1", Environment: "live",
		AggregateID: "u1", OccurredAt: now,
		Payload: &domain.Account{ID: "u1", PrimaryEmail: "new@example.com", EmailVerified: true},
	})
	if !ok || userID != "u1" {
		t.Fatalf("event not normalized: ok=%v user=%q", ok, userID)
	}
	if event.Data["email"] != "new@example.com" || event.Data["email_verified"] != true {
		t.Fatalf("unexpected data: %#v", event.Data)
	}
	if _, exists := event.Data["PasswordHash"]; exists {
		t.Fatal("internal account fields leaked")
	}

	if _, _, ok := publicEventFromDomain(domain.Event{Type: "auth.otp.started", Payload: map[string]any{"code": "123456"}}); ok {
		t.Fatal("credential-bearing internal event became public")
	}
}
