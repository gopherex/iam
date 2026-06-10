package domain

import "time"

// Invite is a project invitation issued by an admin. The raw token is never
// stored; only sha256(token) lives in iam_invites.token_hash. The plain-text
// token is surfaced to the admin exactly once at creation.
type Invite struct {
	ID        string
	ProjectID string
	Email     string // empty when the invite is not email-bound
	Status    string // pending | accepted | revoked
	ExpiresAt time.Time
	CreatedAt time.Time
}

// InviteCreateCmd creates a new invitation. When Email is set the invite is
// email-bound (signup must use the same email) and a notification is sent. The
// raw token is returned in the InviteCreated result, never persisted.
type InviteCreateCmd struct {
	ProjectID   string
	Environment string
	Email       string    // optional; empty → open invite, no email sent
	ExpiresAt   time.Time // zero → default TTL
	RedirectTo  string    // optional base for the email link
}

// InviteCreated is the create result carrying the one-time raw token.
type InviteCreated struct {
	Invite
	Token string
}

// InviteListCmd lists invitations for a project.
type InviteListCmd struct {
	ProjectID   string
	Environment string
}

// InviteRevokeCmd revokes a pending invitation.
type InviteRevokeCmd struct {
	ProjectID   string
	Environment string
	InviteID    string
}
