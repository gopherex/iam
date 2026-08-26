package postgres

// Outbox emitter: the bridge between the aggregate adapters and the
// transactional outbox. Adapters depend on the small Emitter port; the concrete
// outboxEmitter wraps the pg-outbox facade.
//
// Transactionality is automatic: pg-outbox's enqueue Executor is db.TxDB (a
// ctx-aware executor that resolves the ambient pgtx transaction from ctx). When
// an adapter calls emitter.Emit(ctx, …) from inside withTx/withTxRet, the
// INSERT into outbox_messages joins that same transaction — so the event is
// recorded iff the business mutation commits. A failed Emit aborts the whole
// transaction (atomic emit).

import (
	"context"
	"encoding/json"
	"strings"

	outbox "github.com/gopherex/pg-outbox"

	"github.com/gopherex/iam/internal/domain"
)

// Emitter is the port the Postgres adapters use to publish domain events. It is
// consumed inside the adapters' transactions; see outboxEmitter for the wiring.
type Emitter interface {
	Emit(ctx context.Context, ev domain.Event) error
}

// outboxEmitter maps domain.Event onto an outbox.Message and enqueues it.
type outboxEmitter struct{ ob *outbox.Outbox }

// NewOutboxEmitter builds the outbox-backed Emitter over an initialized
// *outbox.Outbox (constructed with db.TxDB as its enqueue executor).
func NewOutboxEmitter(ob *outbox.Outbox) *outboxEmitter { return &outboxEmitter{ob: ob} }

var _ Emitter = (*outboxEmitter)(nil)

// Emit serializes the event and enqueues it on the caller's transaction.
func (e *outboxEmitter) Emit(ctx context.Context, event domain.Event) error {
	if event.ID == "" {
		event.ID = newUUID()
	}

	if event.Version == 0 {
		event.Version = 1
	}

	if event.OccurredAt.IsZero() {
		event.OccurredAt = nowUTC()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return e.ob.Enqueue(ctx, outbox.Message{
		ID:           event.ID,
		Topic:        "iam." + aggregateOf(event.Type),
		MessageType:  event.Type,
		PartitionKey: event.ProjectID,
		Payload:      payload,
		ContentType:  "application/json",
		Headers: map[string]string{
			"project_id":  event.ProjectID,
			"environment": event.Environment,
			"event_id":    event.ID,
		},
	})
}

// aggregateOf returns the aggregate prefix of an event type ("api_key.created"
// -> "api_key"); the topic groups events per aggregate.
func aggregateOf(eventType string) string {
	if i := strings.IndexByte(eventType, '.'); i >= 0 {
		return eventType[:i]
	}

	return eventType
}
