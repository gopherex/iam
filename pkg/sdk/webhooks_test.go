package sdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestWebhookVerifier(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("test-secret"))
	verifier, err := NewWebhookVerifier(WebhookVerifierConfig{SigningSecret: secret, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"id":"evt_1","type":"session.revoked","version":1,"occurred_at":"2026-07-14T12:00:00Z","project_id":"p1","environment":"live","data":{"session_id":"s1","user_id":"u1","project_id":"p1"}}`)
	headers := signedWebhookHeaders("test-secret", "evt_1", now.Unix(), body)
	event, err := verifier.Verify(headers, body)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	data, err := event.SessionRevokedData()
	if err != nil || data.SessionID != "s1" || data.UserID != "u1" || data.ProjectID != "p1" {
		t.Fatalf("data=%+v err=%v", data, err)
	}

	t.Run("changed body", func(t *testing.T) {
		if _, err := verifier.Verify(headers, append([]byte(nil), append(body, ' ')...)); err != ErrWebhookInvalidSignature {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("wrong secret", func(t *testing.T) {
		bad, _ := NewWebhookVerifier(WebhookVerifierConfig{SigningSecret: "wrong", Now: func() time.Time { return now }})
		if _, err := bad.Verify(headers, body); err != ErrWebhookInvalidSignature {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("missing header", func(t *testing.T) {
		h := headers.Clone()
		h.Del("webhook-id")
		if _, err := verifier.Verify(h, body); err != ErrWebhookMalformedHeaders {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("malformed timestamp", func(t *testing.T) {
		h := headers.Clone()
		h.Set("webhook-timestamp", "nope")
		if _, err := verifier.Verify(h, body); err != ErrWebhookMalformedHeaders {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("old timestamp", func(t *testing.T) {
		h := signedWebhookHeaders("test-secret", "evt_1", now.Add(-6*time.Minute).Unix(), body)
		if _, err := verifier.Verify(h, body); err != ErrWebhookStaleTimestamp {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("future timestamp", func(t *testing.T) {
		h := signedWebhookHeaders("test-secret", "evt_1", now.Add(6*time.Minute).Unix(), body)
		if _, err := verifier.Verify(h, body); err != ErrWebhookStaleTimestamp {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("one valid signature", func(t *testing.T) {
		h := headers.Clone()
		h.Set("webhook-signature", "v1,invalid v1,"+signature("test-secret", "evt_1", now.Unix(), body))
		if _, err := verifier.Verify(h, body); err != nil {
			t.Fatalf("Verify: %v", err)
		}
	})
}

func signedWebhookHeaders(secret, id string, timestamp int64, body []byte) http.Header {
	headers := make(http.Header)
	headers.Set("webhook-id", id)
	headers.Set("webhook-timestamp", fmt.Sprint(timestamp))
	headers.Set("webhook-signature", "v1,"+signature(secret, id, timestamp, body))
	return headers
}

func signature(secret, id string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%s.%d.", id, timestamp)
	_, _ = mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
