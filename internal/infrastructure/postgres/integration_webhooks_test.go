//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

func TestWebhookDeliveryLifecycle(t *testing.T) {
	ctx := context.Background()
	var fail atomic.Bool
	var calls atomic.Int32
	var lastSignature atomic.Value
	projectID := newUUID()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastSignature.Store(r.Header.Get("Webhook-Signature"))
		var event domain.PublicEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("decode event: %v", err)
		}
		if event.ID == "" || event.Version != 1 {
			t.Errorf("bad event envelope: %+v", event)
		}
		// The Test webhook action delivers a synthetic {"test": true} payload with
		// the event type being exercised; only assert the real session.revoked
		// envelope emitted by PublishEvent.
		if event.Type == domain.WebhookEventSessionRevoked && event.Data["test"] != true {
			if event.Data["session_id"] == "" || event.Data["user_id"] == "" || event.Data["project_id"] != projectID {
				t.Errorf("bad session.revoked data: %#v", event.Data)
			}
		}
		if fail.Load() {
			http.Error(w, "retry", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	oldCipher := testDB.Cipher
	testDB.UseCipher(testCipher(t))
	defer testDB.UseCipher(oldCipher)

	service := NewPgWebhooks(testDB, server.Client())
	idempotencyKey := newUUID()
	created, secret, err := service.Create(ctx, domain.WebhookCreateCmd{
		ProjectID: projectID, Environment: "live", URL: server.URL,
		Events: []string{domain.WebhookEventSessionRevoked}, Enabled: true,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("missing signing secret")
	}
	again, againSecret, err := service.Create(ctx, domain.WebhookCreateCmd{
		ProjectID: projectID, Environment: "live", URL: server.URL,
		Events: []string{domain.WebhookEventSessionRevoked}, Enabled: true,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil || again.ID != created.ID || againSecret != secret {
		t.Fatalf("idempotent create mismatch: webhook=%+v secret_equal=%v err=%v", again, againSecret == secret, err)
	}
	var stored string
	if err := testDB.Pool.QueryRow(ctx, `SELECT data::text FROM iam_webhooks WHERE id = $1`, created.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, secret) {
		t.Fatal("signing secret stored in plaintext")
	}

	eventID := newUUID()
	if err := service.PublishEvent(ctx, domain.Event{
		ID: eventID, Type: domain.WebhookEventSessionRevoked, ProjectID: projectID,
		Environment: "live", AggregateID: "s1", OccurredAt: time.Now().UTC(),
		Payload: domain.SessionRevokedPayload{SessionID: "s1", UserID: "u1", ProjectID: projectID},
	}); err != nil {
		t.Fatal(err)
	}
	deliveries, err := service.ListDeliveries(ctx, domain.WebhookDeliveryListCmd{ProjectID: projectID, Environment: "live"})
	if err != nil || len(deliveries) != 1 || deliveries[0].Status != "succeeded" {
		t.Fatalf("deliveries=%+v err=%v", deliveries, err)
	}
	if signature, _ := lastSignature.Load().(string); !strings.Contains(signature, "v1,") {
		t.Fatalf("bad signature header: %q", signature)
	}

	newSecret, err := service.RotateSecret(ctx, projectID, "live", created.ID)
	if err != nil || newSecret == secret || newSecret == "" {
		t.Fatalf("rotate secret: changed=%v err=%v", newSecret != secret, err)
	}
	if _, err := service.Test(ctx, projectID, "live", created.ID, domain.WebhookEventSessionRevoked); err != nil {
		t.Fatal(err)
	}
	if signature, _ := lastSignature.Load().(string); strings.Count(signature, "v1,") != 2 {
		t.Fatalf("rotation overlap must emit two signatures: %q", signature)
	}

	fail.Store(true)
	failedEventID := newUUID()
	err = service.PublishEvent(ctx, domain.Event{
		ID: failedEventID, Type: domain.WebhookEventSessionRevoked, ProjectID: projectID,
		Environment: "live", AggregateID: "s2", OccurredAt: time.Now().UTC(),
		Payload: domain.SessionRevokedPayload{SessionID: "s2", UserID: "u1", ProjectID: projectID},
	})
	if err == nil {
		t.Fatal("failed endpoint must ask outbox for a retry")
	}
	failed, err := service.getDeliveryByPair(ctx, created.ID, failedEventID)
	if err != nil || failed.Status != "failed" || failed.AttemptCount != 1 || failed.NextAttemptAt == nil {
		t.Fatalf("failed delivery=%+v err=%v", failed, err)
	}
	fail.Store(false)
	retried, err := service.RetryDelivery(ctx, projectID, "live", failed.ID)
	if err != nil || retried.Status != "succeeded" || retried.AttemptCount != 2 {
		t.Fatalf("retried=%+v err=%v", retried, err)
	}

	page, err := service.ListEvents(ctx, domain.WebhookEventListCmd{ProjectID: projectID, Environment: "live", UserID: "u1"})
	if err != nil || len(page.Data) != 2 {
		t.Fatalf("events=%+v err=%v", page, err)
	}
	if calls.Load() < 4 {
		t.Fatal(fmt.Sprintf("expected delivery calls, got %d", calls.Load()))
	}
}
