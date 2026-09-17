package domain

import "time"

// AccountDeletion is independent of Account.Status: pending accounts stay active.
type AccountDeletion struct {
	OK          bool       `json:"ok"`
	RequestID   string     `json:"request_id,omitempty"`
	Status      string     `json:"status"`
	RequestedAt *time.Time `json:"requested_at,omitempty"`
	DeleteAt    *time.Time `json:"delete_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	GraceDays   int        `json:"grace_days"`
}

type AccountDeletionPolicy struct {
	GraceDays int `json:"grace_days"`
}

type AccountDeletionInput struct {
	Password   string `json:"password,omitempty"`
	ProofToken string `json:"proof_token,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type AdminAccountDeletionCancelInput struct {
	RequestID string `json:"request_id"`
	Reason    string `json:"reason"`
}

type AccountDeletionEvent struct {
	ActorID   string    `json:"actor_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Type      string    `json:"type"`
	At        time.Time `json:"at"`
	RequestID string    `json:"request_id"`
}

type AccountDeletionHistory struct {
	Deletion AccountDeletion        `json:"deletion"`
	Events   []AccountDeletionEvent `json:"events"`
}
