package postgres

import (
	"context"
	"crypto/hmac"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/webhooksecret"
	iamsdk "github.com/gopherex/iam/pkg/sdk"
)

func TestWebhookSignature(t *testing.T) {
	t.Parallel()

	body := []byte(`{"id":"evt-1"}`)

	got := webhookSignature([]byte("secret"), "evt-1", 123, body)
	if got == "" {
		t.Fatal("signature is empty")
	}

	if !hmac.Equal([]byte(got), []byte(webhookSignature([]byte("secret"), "evt-1", 123, body))) {
		t.Fatal("signature is not deterministic")
	}

	if hmac.Equal([]byte(got), []byte(webhookSignature([]byte("other"), "evt-1", 123, body))) {
		t.Fatal("signature does not depend on the secret")
	}
}

func TestPublicSessionRevokedPayloadIsExact(t *testing.T) {
	t.Parallel()

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

	if _, _, published := publicEventFromDomain(domain.Event{
		Type: domain.WebhookEventSessionRevoked, ProjectID: "p1",
		Payload: domain.SessionRevokedPayload{SessionID: "s1", ProjectID: "p1"},
	}); published {
		t.Fatal("session.revoked without user_id became public")
	}
}

func TestValidateWebhookURL(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

	if _, _, published := publicEventFromDomain(domain.Event{Type: "auth.otp.started", Payload: map[string]any{"code": "123456"}}); published {
		t.Fatal("credential-bearing internal event became public")
	}
}

func TestWebhookRequestVerifiesWithSDK(t *testing.T) {
	t.Parallel()

	generated, err := newWebhookSigningSecret()
	if err != nil {
		t.Fatal(err)
	}

	previous, err := newWebhookSigningSecret()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, current, previous string
		expired                 bool
	}{
		{name: "issued secret", current: generated},
		{name: "legacy secret", current: "legacy-secret"},
		{name: "rotation", current: generated, previous: previous},
		{name: "legacy rotation", current: generated, previous: "previous-legacy-secret"},
		{name: "expired previous secret", current: generated, previous: previous, expired: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			until := time.Now().Add(time.Hour)
			if tc.expired {
				until = time.Now().Add(-time.Second)
			}

			event := domain.PublicEvent{ID: "evt-sdk", Type: domain.WebhookEventSessionRevoked, Version: 1, ProjectID: "p1", Environment: "live", OccurredAt: time.Now().UTC(), Data: map[string]any{"session_id": "s1", "user_id": "u1", "project_id": "p1"}}

			request, err := buildWebhookRequest(context.Background(), &domain.Webhook{URL: "https://receiver.example/webhook", SigningSecret: tc.current, PreviousSigningSecret: tc.previous, PreviousSecretValidUntil: until}, event)
			if err != nil {
				t.Fatal(err)
			}

			defer request.Body.Close()

			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}

			current, err := iamsdk.NewWebhookVerifier(iamsdk.WebhookVerifierConfig{SigningSecret: tc.current})
			if err != nil {
				t.Fatal(err)
			}

			received, err := current.Verify(request.Header, body)
			if err != nil {
				t.Fatalf("IAM request rejected by SDK: %v", err)
			}

			if received.ID != event.ID {
				t.Fatal("event ID changed")
			}

			if _, err := current.Verify(request.Header, append(body, ' ')); !errors.Is(err, iamsdk.ErrWebhookInvalidSignature) {
				t.Fatalf("tampered payload: %v", err)
			}

			want := 1
			if tc.previous != "" && !tc.expired {
				want = 2
			}

			if tc.previous != "" {
				verifyPreviousWebhookSecret(t, tc.previous, request.Header, body, tc.expired)
			}

			if got := len(strings.Fields(request.Header.Get("Webhook-Signature"))); got != want {
				t.Fatalf("signatures=%d want=%d", got, want)
			}
		})
	}
}

func TestWebhookRequestRejectsMalformedSecret(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, current, previous string
		expired                 bool
		want                    error
	}{
		{name: "empty current", want: webhooksecret.ErrRequired},
		{name: "invalid current", current: "whsec_invalid!", want: webhooksecret.ErrInvalid},
		{name: "invalid previous", current: "legacy", previous: "whsec_invalid!", want: webhooksecret.ErrInvalid},
		{name: "expired invalid previous ignored", current: "legacy", previous: "whsec_invalid!", expired: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			until := time.Now().Add(time.Hour)
			if tc.expired {
				until = time.Now().Add(-time.Second)
			}

			req, err := buildWebhookRequest(context.Background(), &domain.Webhook{URL: "https://receiver.example/hook", SigningSecret: tc.current, PreviousSigningSecret: tc.previous, PreviousSecretValidUntil: until}, domain.PublicEvent{ID: "event"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}

			if req != nil {
				req.Body.Close()
			}
		})
	}
}

func verifyPreviousWebhookSecret(t *testing.T, secret string, headers http.Header, body []byte, expired bool) {
	t.Helper()

	verifier, err := iamsdk.NewWebhookVerifier(iamsdk.WebhookVerifierConfig{SigningSecret: secret})
	if err != nil {
		t.Fatal(err)
	}

	var want error
	if expired {
		want = iamsdk.ErrWebhookInvalidSignature
	}

	if _, err := verifier.Verify(headers, body); !errors.Is(err, want) {
		t.Fatalf("previous signature error=%v want=%v", err, want)
	}
}
