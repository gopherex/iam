package domain

import "time"

// Event is a domain event emitted transactionally through the outbox: the
// adapter that mutates an aggregate enqueues the event inside the same database
// transaction, so the event is durably recorded iff the mutation commits.
//
// Payload carries a full snapshot of the affected aggregate (the subscriber is
// an in-infrastructure consumer that wants the state in-context, not a public
// API). Delivery events (auth.otp.started, auth.magiclink.started, …) carry the
// plaintext secret needed to dispatch the message — that plaintext lands in the
// outbox at rest and is the reason outbox payloads are an encryption target.
type Event struct {
	// ID is the unique event id (uuid). Consumers deduplicate on it because
	// outbox delivery is at-least-once.
	ID string
	// Type is the event name, "<aggregate>.<verb>" (e.g. "api_key.created").
	Type string
	// ProjectID is the owning tenant; used as the outbox partition key so events
	// for one project stay ordered.
	ProjectID string
	// Environment is the project environment (live/…); empty when not applicable.
	Environment string
	// AggregateID is the id of the affected aggregate instance.
	AggregateID string
	// OccurredAt is when the event happened (UTC).
	OccurredAt time.Time
	// Payload is the aggregate snapshot (or, for delivery events, the dispatch
	// envelope). JSON-serialised into the outbox message body.
	Payload any
}
