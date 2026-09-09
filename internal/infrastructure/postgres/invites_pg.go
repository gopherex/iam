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
	b := make([]byte, randomTokenBytes)
	if _, err = rand.Read(b); err != nil {
		return token, hash, fmt.Errorf("mint invite token: %w", err)
	}

	token = inviteTokenPrefix + hex.EncodeToString(b)
	hash = inviteHashToken(token)

	return token, hash, nil
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
	cfg     *configReader
}

// NewPgInvites builds the invite adapter. cfg reads the auth doc for the
// member-invite default policy; nil falls back to a default-TTL reader.
func NewPgInvites(db *DB, emitter Emitter, cfg *configReader) *pgInvites {
	if cfg == nil {
		cfg = NewConfigReader(db, 0)
	}

	return &pgInvites{db: db, emitter: emitter, cfg: cfg}
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

// emitInviteCreated notifies the notification layer to send the invitation
// email; invite_url is built downstream from the app base URL (or the
// per-invite redirect_to) + raw token.
func (a *pgInvites) emitInviteCreated(ctx context.Context, cmd domain.InviteCreateCmd, env, id, token string) error {
	payload := map[string]any{
		"to":           cmd.Email,
		"invite_token": token,
	}
	if cmd.RedirectTo != "" {
		payload["redirect_to"] = cmd.RedirectTo
	}

	if cmd.CreatedBy != "" {
		payload["created_by"] = cmd.CreatedBy
	}

	return a.emitter.Emit(ctx, domain.Event{
		Type:        "invite.created",
		ProjectID:   cmd.ProjectID,
		Environment: env,
		AggregateID: id,
		Payload:     payload,
	})
}

// Create mints a token, inserts the invite, and (when email-bound) emits an
// invite.created event so the notification layer sends the invitation email.
// The raw token is returned exactly once. A member invite (CreatedBy != "")
// must be email-bound and passes the creator's invite-cap check first.
func (a *pgInvites) Create(ctx context.Context, cmd domain.InviteCreateCmd) (*domain.InviteCreated, error) {
	if cmd.CreatedBy != "" && cmd.Email == "" {
		return nil, domain.ErrValidation.WithMessage("member invitations must be email-bound")
	}

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
		// Member invites serialize on the creator: the advisory xact lock plus
		// the in-transaction slot count close the check-then-insert race that
		// would otherwise let N concurrent creates overshoot the cap.
		if cmd.CreatedBy != "" {
			if err := lockInviteQuota(ctx, a.db, cmd.CreatedBy); err != nil {
				return nil, err
			}

			if _, _, err := a.checkQuotaLocked(ctx, cmd.ProjectID, env, cmd.CreatedBy); err != nil {
				return nil, err
			}
		}

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

		if cmd.CreatedBy != "" {
			setter.CreatedBy = ptr(null.From(cmd.CreatedBy))
		}

		if _, ierr := models.IamInvites.Insert(setter).One(ctx, a.db.Bobx()); ierr != nil {
			return nil, fmt.Errorf("invite create: insert: %w", ierr)
		}

		// Only email-bound invites trigger a send.
		if cmd.Email != "" {
			if eerr := a.emitInviteCreated(ctx, cmd, env, id, token); eerr != nil {
				return nil, eerr
			}
		}

		return &domain.InviteCreated{
			Invite: domain.Invite{
				ID:        id,
				ProjectID: cmd.ProjectID,
				Email:     cmd.Email,
				Status:    inviteStatusPend,
				ExpiresAt: expires,
				CreatedAt: now,
			},
			Token: token,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

// errInviteQuota codes the cap-exhausted terminal failure: a state conflict
// (the caller asks for a new resource slot that the quota forbids), not an
// authorization problem — the right itself is intact.
var errInviteQuota = domain.ErrConflict.WithMessage("invite quota exceeded")

// inviteSlotStatuses are the statuses that occupy a creator's invite-cap slot.
const inviteSlotStatuses = "('pending','accepted')"

// lockInviteQuota takes the per-user transactional advisory lock serializing
// member-invite creation for one user.
func lockInviteQuota(ctx context.Context, db *DB, userID string) error {
	if _, err := db.TxDB.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('iam_invite_quota:' || $1))`, userID,
	); err != nil {
		return fmt.Errorf("invite quota lock: %w", err)
	}

	return nil
}

// effectiveInviteCap resolves the user's cap: the per-user invite_cap override
// from the account envelope (missing → project default; 0 → right revoked),
// else the auth-doc member_invites default when enabled.
func (a *pgInvites) effectiveInviteCap(ctx context.Context, projectID, userID string) (int, error) {
	if row, err := models.FindIamUser(ctx, a.db.Bobx(), userID); err == nil && len(row.Data) > 0 {
		var env struct {
			InviteCap *int `json:"invite_cap"`
		}
		if json.Unmarshal(row.Data, &env) == nil && env.InviteCap != nil {
			return *env.InviteCap, nil
		}
	}

	cfg, err := a.cfg.AuthConfig(ctx, projectID)
	if err != nil {
		return 0, err
	}

	if !cfg.MemberInvitesEnabled {
		return 0, nil
	}

	return cfg.MemberInvitesDefaultCap, nil
}

// usedInviteSlots counts the creator's occupied slots: pending-unexpired plus
// accepted. Revoked and expired-pending rows are dead — they free the slot.
func (a *pgInvites) usedInviteSlots(ctx context.Context, projectID, env, userID string) (int, error) {
	var used int
	if err := a.db.Pool.QueryRow(ctx, `
		SELECT count(*) FROM iam_invites
		WHERE project_id = $1 AND environment = $2 AND created_by = $3
		  AND status IN `+inviteSlotStatuses+`
		  AND (expires_at IS NULL OR expires_at > now())`,
		projectID, env, userID,
	).Scan(&used); err != nil {
		return 0, fmt.Errorf("count invite slots: %w", err)
	}

	return used, nil
}

// checkQuotaLocked enforces the cap; must run after lockInviteQuota in the
// same transaction. Returns (effective cap, used slots).
func (a *pgInvites) checkQuotaLocked(ctx context.Context, projectID, env, userID string) (int, int, error) {
	inviteCap, err := a.effectiveInviteCap(ctx, projectID, userID)
	if err != nil {
		return 0, 0, err
	}

	if inviteCap <= 0 {
		return 0, 0, domain.ErrForbidden.WithMessage("member invitations are not permitted")
	}

	used, err := a.usedInviteSlots(ctx, projectID, env, userID)
	if err != nil {
		return 0, 0, err
	}

	if used >= inviteCap {
		return inviteCap, used, errInviteQuota
	}

	return inviteCap, used, nil
}

// InviteQuota reports the caller's member-invitation budget. Used counts the
// occupied slots (pending-unexpired + accepted); Left is the creatable count.
func (a *pgInvites) InviteQuota(ctx context.Context, projectID, env, userID string) (domain.InviteQuota, error) {
	inviteCap, err := a.effectiveInviteCap(ctx, projectID, userID)
	if err != nil {
		return domain.InviteQuota{}, err
	}

	used, err := a.usedInviteSlots(ctx, projectID, env, userID)
	if err != nil {
		return domain.InviteQuota{}, err
	}

	left := max(inviteCap-used, 0)

	return domain.InviteQuota{Cap: inviteCap, Used: used, Left: left}, nil
}

// ListByCreator returns the invites a user created (newest first). Member
// scope: only their own rows, any status.
func (a *pgInvites) ListByCreator(ctx context.Context, projectID, env, userID string) ([]domain.Invite, error) {
	rows, err := models.IamInvites.Query(
		sm.Where(models.IamInvites.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamInvites.Columns.Environment.EQ(psql.Arg(env))),
		sm.Where(models.IamInvites.Columns.CreatedBy.EQ(psql.Arg(userID))),
		sm.OrderBy(models.IamInvites.Columns.CreatedAt).Desc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, fmt.Errorf("invite list by creator: %w", err)
	}

	out := make([]domain.Invite, 0, len(rows))
	for _, row := range rows {
		out = append(out, inviteToDomain(row))
	}

	return out, nil
}

// RevokeOwn marks the creator's own pending invitation revoked (freeing the
// slot). Someone else's or already-decided rows are ErrNotFound (no probing).
func (a *pgInvites) RevokeOwn(ctx context.Context, projectID, env, inviteID, userID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamInvite(ctx, a.db.Bobx(), inviteID)
		if err != nil {
			if adminIsNotFound(err) {
				return domain.ErrNotFound
			}

			return err
		}

		creator, hasCreator := row.CreatedBy.Get()
		if row.ProjectID != projectID || row.Environment != env || !hasCreator || creator != userID {
			return domain.ErrNotFound
		}

		if row.Status != inviteStatusPend {
			return domain.ErrConflict
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

// List returns the project's invitations (most recent first).
func (a *pgInvites) List(ctx context.Context, cmd domain.InviteListCmd) ([]domain.Invite, error) {
	rows, err := models.IamInvites.Query(
		sm.Where(models.IamInvites.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
		sm.Where(models.IamInvites.Columns.Environment.EQ(psql.Arg(adminEnv(cmd.Environment)))),
		sm.OrderBy(models.IamInvites.Columns.CreatedAt).Desc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, fmt.Errorf("invite list: %w", err)
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
