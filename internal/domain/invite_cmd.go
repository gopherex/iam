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
	// CreatedBy is the inviting user's account id for member invitations;
	// empty for admin/system invites (panel, approvals, integrations).
	CreatedBy string
	// CreatedByEmail is the resolved inviter address (admin listings only).
	CreatedByEmail string
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
	// CreatedBy marks a member (user-driven) invitation: the invite occupies
	// one of that user's invite-cap slots. Empty = admin/system invite (the
	// admin API, access-request approvals, the telegram bot), uncapped.
	CreatedBy string
}

// InviteCreated is the create result carrying the one-time raw token.
type InviteCreated struct {
	Invite
	Token string
}

// InviteQuota is a user's member-invitation budget: Cap is the effective
// limit (user override or project default), Used the occupied slots
// (pending-unexpired + accepted), Left the creatable count.
type InviteQuota struct {
	Cap  int
	Used int
	Left int
}

// WithCreatorEmail is the admin-listing enrichment: the inviter's account id
// resolved to their email (best-effort — the account may be gone).
func (i Invite) WithCreatorEmail(email string) Invite {
	i.CreatedByEmail = email

	return i
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
