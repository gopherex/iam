package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// revokeSessionRecord is the single revocation primitive used by runtime,
// account and admin paths. Keeping it here prevents event-name/payload drift and
// guarantees refresh tokens are invalidated before the session disappears.
// The caller must already be inside a transaction.
func revokeSessionRecord(ctx context.Context, db *DB, emitter Emitter, row *models.IamSession, _ string) error {
	refreshTokens, err := models.IamRefreshTokens.Query(
		sm.Where(models.IamRefreshTokens.Columns.ProjectID.EQ(psql.Arg(row.ProjectID))),
		sm.Where(models.IamRefreshTokens.Columns.SessionID.EQ(psql.Arg(row.ID))),
		sm.Where(models.IamRefreshTokens.Columns.Revoked.EQ(psql.Arg(false))),
	).All(ctx, db.Bobx())
	if err != nil {
		return fmt.Errorf("list refresh tokens: %w", err)
	}

	for _, refresh := range refreshTokens {
		var data coreAuthRefreshToken
		if len(refresh.Data) > 0 {
			if err := unmarshal(refresh.Data, &data); err != nil {
				return err
			}
		}

		data.Revoked = true

		raw, err := marshal(data)
		if err != nil {
			return err
		}

		rm := json.RawMessage(raw)
		if err := refresh.Update(ctx, db.Bobx(), &models.IamRefreshTokenSetter{Revoked: ptr(true), Data: &rm}); err != nil {
			return err
		}
	}

	if err := row.Delete(ctx, db.Bobx()); err != nil {
		return err
	}

	return emitter.Emit(ctx, domain.Event{
		Type:        domain.WebhookEventSessionRevoked,
		ProjectID:   row.ProjectID,
		Environment: row.Environment,
		AggregateID: row.ID,
		Payload: domain.SessionRevokedPayload{
			SessionID: row.ID,
			UserID:    row.UserID,
			ProjectID: row.ProjectID,
		},
	})
}

// revokeUserSessions revokes every live session of a user (project+env
// scoped) except exceptID, emitting the session.revoked events downstream
// consumers (backchannel logout, webhooks) already subscribe to. Used by the
// admin paths: ban, and role changes (a stale role must not ride a live
// token). Must run inside the caller's transaction.
func revokeUserSessions(
	ctx context.Context, db *DB, emitter Emitter, projectID, env, userID, exceptID, reason string,
) (int, error) {
	rows, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSessions.Columns.Environment.EQ(psql.Arg(adminEnv(env)))),
		sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(userID))),
	).All(ctx, db.Bobx())
	if err != nil {
		return 0, fmt.Errorf("list sessions: %w", err)
	}

	revoked := 0

	for _, row := range rows {
		if exceptID != "" && row.ID == exceptID {
			continue
		}

		if err := revokeSessionRecord(ctx, db, emitter, row, reason); err != nil {
			return revoked, err
		}

		revoked++
	}

	return revoked, nil
}
