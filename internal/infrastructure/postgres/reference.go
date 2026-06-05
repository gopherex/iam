package postgres

// REFERENCE ADAPTER — the worked pattern every port adapter follows.
//
// Persistence rules for this package:
//   - Prefer the generated bob query builders (internal/.../gen/bob/models).
//     Use the sqld(c) typed funcs (db.q(), gen/db) ONLY for super-hot paths.
//   - Wrap every MUTATION in db.withTx / withTxRet (serializable + mandatory
//     retry). Reads may run directly on db.Bobx().
//   - Marshal the domain aggregate into the `data jsonb` column (marshal/
//     unmarshal helpers); envelope columns are for lookups only.
//   - Translate pg errors: translatePgErr maps no-rows -> domain not-found;
//     isUniqueViolation maps 23505 -> the domain conflict.
//   - Every place that would emit a domain event is marked `// TODO outbox
//     event: <name>` (no outbox logic yet).
//
// This file is illustrative; it does not implement a pkg/api port. Real
// adapters live in their own files and assert the interface, e.g.
// `var _ api.AccountStore = (*pgAccountStore)(nil)`.

import (
	"context"
	"encoding/json"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

func ptr[T any](v T) *T { return &v }

// pgUsersReference shows the bob + pgtx pattern over the iam_users envelope.
type pgUsersReference struct{ db *DB }

func (a *pgUsersReference) create(ctx context.Context, u *domain.Account) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		raw, err := marshal(u)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamUserSetter{
			ID:        &u.ID,
			ProjectID: &u.ProjectID,
			Kind:      ptr(u.Kind),
			Status:    ptr(u.Status),
			Data:      &rm,
		}
		if u.PrimaryEmail != "" {
			v := null.From(u.PrimaryEmail)
			setter.PrimaryEmail = &v
		}
		if _, err := models.IamUsers.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return domain.ErrEmailExists
			}
			return err
		}
		// TODO outbox event: user.created
		return nil
	})
}

func (a *pgUsersReference) get(ctx context.Context, projectID, id string) (*domain.Account, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), id)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, domain.ErrUserNotFound
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

func (a *pgUsersReference) list(ctx context.Context, projectID string) ([]domain.Account, error) {
	rows, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Account, 0, len(rows))
	for _, row := range rows {
		var acc domain.Account
		if err := unmarshal(row.Data, &acc); err != nil {
			return nil, err
		}
		out = append(out, acc)
	}
	return out, nil
}

func (a *pgUsersReference) update(ctx context.Context, u *domain.Account) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamUser(ctx, a.db.Bobx(), u.ID)
		if err != nil {
			return translatePgErr("user", err)
		}
		raw, err := marshal(u)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamUserSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return err
		}
		// TODO outbox event: user.updated
		return nil
	})
}

func (a *pgUsersReference) delete(ctx context.Context, projectID, id string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamUser(ctx, a.db.Bobx(), id)
		if err != nil {
			return translatePgErr("user", err)
		}
		if row.ProjectID != projectID {
			return domain.ErrUserNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: user.deleted
		return nil
	})
}
