package postgres

// Postgres adapter for the project invitation system.
//
// An invite carries an opaque token (`inv_` prefix, ≥256-bit random). Only
// sha256(token) hex lives in iam_invites.token_hash; the raw token is returned
// to the admin response exactly once at creation. Admins create (optionally
// email-bound + emailed), list, and revoke invites. The signup flow redeems an
// invite when the project registration mode is invite_only (see
// coreauth_flows_pg.go for the redeem/accept path which shares inviteHashToken).

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

const (
	inviteTokenPrefix  = "inv_"
	inviteDefaultTTL   = 7 * 24 * time.Hour
	inviteStatusPend   = "pending"
	inviteStatusAccept = "accepted"
	inviteStatusRevoke = "revoked"
)

// inviteMintToken mints a new opaque invite token (`inv_` prefix, ≥256-bit).
func inviteMintToken() (token, hash string, err error) {
	b := make([]byte, 32) // 256 bits
	if _, err = rand.Read(b); err != nil {
		return
	}
	token = inviteTokenPrefix + hex.EncodeToString(b)
	hash = inviteHashToken(token)
	return
}

// inviteHashToken returns sha256(token) in hex. Shared with the flow redeem path.
func inviteHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// pgInvites is the Postgres-backed api.AdminInvites adapter.
type pgInvites struct {
	db      *DB
	emitter Emitter
}

// NewPgInvites builds the invite admin adapter.
func NewPgInvites(db *DB, emitter Emitter) *pgInvites {
	return &pgInvites{db: db, emitter: emitter}
}

var _ api.AdminInvites = (*pgInvites)(nil)

// inviteToDomain maps a model row onto domain.Invite.
func inviteToDomain(row *models.IamInvite) domain.Invite {
	inv := domain.Invite{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
	}
	if email, ok := row.Email.Get(); ok {
		inv.Email = email
	}
	if exp, ok := row.ExpiresAt.Get(); ok {
		inv.ExpiresAt = exp
	}
	return inv
}

// Create mints a token, inserts the invite, and (when email-bound) emits an
// invite.created event so the notification layer sends the invitation email.
// The raw token is returned exactly once.
func (a *pgInvites) Create(ctx context.Context, cmd domain.InviteCreateCmd) (*domain.InviteCreated, error) {
	token, hash, err := inviteMintToken()
	if err != nil {
		return nil, fmt.Errorf("invite create: mint token: %w", err)
	}
	now := nowUTC()
	expires := cmd.ExpiresAt
	if expires.IsZero() {
		expires = now.Add(inviteDefaultTTL)
	}
	env := coreAuthDefaultEnv
	if cmd.Environment != "" {
		env = cmd.Environment
	}
	id := newUUID()

	created, err := withTxRet(ctx, a.db, func(ctx context.Context) (*domain.InviteCreated, error) {
		emptyData := json.RawMessage(`{}`)
		setter := &models.IamInviteSetter{
			ID:          &id,
			ProjectID:   &cmd.ProjectID,
			Environment: &env,
			TokenHash:   &hash,
			Status:      ptr(inviteStatusPend),
			ExpiresAt:   ptr(null.From(expires)),
			CreatedAt:   &now,
			UpdatedAt:   &now,
			Data:        &emptyData,
		}
		if cmd.Email != "" {
			setter.Email = ptr(null.From(cmd.Email))
		}
		if _, ierr := models.IamInvites.Insert(setter).One(ctx, a.db.Bobx()); ierr != nil {
			return nil, ierr
		}
		out := &domain.InviteCreated{
			Invite: domain.Invite{
				ID:        id,
				ProjectID: cmd.ProjectID,
				Email:     cmd.Email,
				Status:    inviteStatusPend,
				ExpiresAt: expires,
				CreatedAt: now,
			},
			Token: token,
		}
		// Only email-bound invites trigger a send.
		if cmd.Email != "" {
			payload := map[string]any{
				"to":           cmd.Email,
				"invite_token": token,
			}
			if cmd.RedirectTo != "" {
				payload["redirect_to"] = cmd.RedirectTo
			}
			if eerr := a.emitter.Emit(ctx, domain.Event{
				Type:        "invite.created",
				ProjectID:   cmd.ProjectID,
				Environment: env,
				AggregateID: id,
				Payload:     payload,
			}); eerr != nil {
				return nil, eerr
			}
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// List returns the project's invitations (most recent first).
func (a *pgInvites) List(ctx context.Context, cmd domain.InviteListCmd) ([]domain.Invite, error) {
	rows, err := models.IamInvites.Query(
		sm.Where(models.IamInvites.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
		sm.Where(models.IamInvites.Columns.Environment.EQ(psql.Arg(adminEnv(cmd.Environment)))),
		sm.OrderBy(models.IamInvites.Columns.CreatedAt).Desc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Invite, 0, len(rows))
	for _, row := range rows {
		out = append(out, inviteToDomain(row))
	}
	return out, nil
}

// Revoke marks a pending invitation revoked. Tenant-scoped; a foreign or missing
// invite yields ErrNotFound.
func (a *pgInvites) Revoke(ctx context.Context, cmd domain.InviteRevokeCmd) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamInvite(ctx, a.db.Bobx(), cmd.InviteID)
		if err != nil {
			if adminIsNotFound(err) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != cmd.ProjectID || row.Environment != adminEnv(cmd.Environment) {
			return domain.ErrNotFound
		}
		now := nowUTC()
		if err := row.Update(ctx, a.db.Bobx(), &models.IamInviteSetter{
			Status:    ptr(inviteStatusRevoke),
			UpdatedAt: &now,
		}); err != nil {
			return err
		}
		return nil
	})
}
