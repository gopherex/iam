package domain

import "time"

const (
	WebhookEventSessionRevoked = "session.revoked"
	WebhookEventUserBanned     = "user.banned"
	WebhookEventUserDeleted    = "user.deleted"
	WebhookEventEmailChanged   = "email.changed"
)

// SupportedWebhookEvents is the deliberately small public event catalogue.
// Internal delivery events may contain credentials or one-time proofs and must
// never become subscribable merely because they pass through the same outbox.
var SupportedWebhookEvents = []string{
	WebhookEventSessionRevoked,
	WebhookEventUserBanned,
	WebhookEventUserDeleted,
	WebhookEventEmailChanged,
}

type Webhook struct {
	ID                       string    `json:"id"`
	ProjectID                string    `json:"project_id"`
	Environment              string    `json:"environment"`
	URL                      string    `json:"url"`
	Events                   []string  `json:"events"`
	Description              string    `json:"description,omitempty"`
	Enabled                  bool      `json:"enabled"`
	SigningSecret            string    `json:"-"`
	PreviousSigningSecret    string    `json:"-"`
	PreviousSecretValidUntil time.Time `json:"-"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type WebhookCreateCmd struct {
	ProjectID      string
	Environment    string
	URL            string
	Events         []string
	Description    string
	Enabled        bool
	IdempotencyKey string
}

type WebhookUpdateCmd struct {
	ProjectID   string
	Environment string
	ID          string
	URL         *string
	Events      *[]string
	Description *string
	Enabled     *bool
}

type WebhookListCmd struct {
	ProjectID   string
	Environment string
	Cursor      string
	Limit       int
}

type WebhookDeliveryListCmd struct {
	ProjectID   string
	Environment string
	WebhookID   string
	Status      string
	Limit       int
}

type WebhookDelivery struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	Environment    string     `json:"environment"`
	WebhookID      string     `json:"webhook_id"`
	EventID        string     `json:"event_id"`
	EventType      string     `json:"event_type"`
	Status         string     `json:"status"`
	AttemptCount   int        `json:"attempt_count"`
	NextAttemptAt  *time.Time `json:"next_attempt_at,omitempty"`
	LastAttemptAt  *time.Time `json:"last_attempt_at,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ResponseStatus *int       `json:"response_status,omitempty"`
	ResponseBody   string     `json:"response_body,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PublicEvent is the versioned, credential-free body delivered to consumers.
type PublicEvent struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Version     int            `json:"version"`
	OccurredAt  time.Time      `json:"occurred_at"`
	ProjectID   string         `json:"project_id"`
	Environment string         `json:"environment"`
	Data        map[string]any `json:"data"`
}

type WebhookEventListCmd struct {
	ProjectID   string
	Environment string
	Type        string
	UserID      string
	Cursor      string
	Limit       int
}

type WebhookEventPage struct {
	Data       []PublicEvent
	NextCursor string
	HasMore    bool
}
