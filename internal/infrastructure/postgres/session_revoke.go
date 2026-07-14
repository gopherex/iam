package postgres

import (
	"context"
	"encoding/json"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
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
		return err
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
