package postgres

// Postgres adapter for the per-project service-account administration port
// declared in pkg/api/admin.go:
//
//   - pgAdminServiceAccounts -> api.AdminServiceAccounts (iam_service_accounts + iam_app_secrets)
//
// The service account aggregate is persisted as a `data jsonb` envelope; the
// typed columns (project_id, name, disabled) are lookup-only and derived from
// the marshalled struct. Every query is scoped by project_id (the tenant
// boundary). Reads run on db.Bobx(); every mutation is wrapped in
// db.withTx / withTxRet (serializable + mandatory retry).
//
// Secrets are minted with adminRandomToken and persisted only as a sha256 hash;
// the plaintext is returned exactly once inside domain.AdminSecret. The
// iam_app_secrets table is reused for service-account secrets (app_id = sa.ID).

import (
	"context"
	"encoding/json"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// =====================================================================
// AdminServiceAccounts — iam_service_accounts + iam_app_secrets
// =====================================================================

type pgAdminServiceAccounts struct{ db *DB }

// NewPgAdminServiceAccounts builds the Postgres-backed AdminServiceAccounts adapter.
func NewPgAdminServiceAccounts(db *DB) *pgAdminServiceAccounts {
	return &pgAdminServiceAccounts{db: db}
}

var _ api.AdminServiceAccounts = (*pgAdminServiceAccounts)(nil)

// findSA loads a service account row enforcing the tenant boundary.
func (a *pgAdminServiceAccounts) findSA(ctx context.Context, projectID, saID string) (*models.IamServiceAccount, *domain.ServiceAccount, error) {
	row, err := models.FindIamServiceAccount(ctx, a.db.Bobx(), saID)
	if err != nil {
		if adminIsNotFound(translatePgErr("service_account", err)) {
			return nil, nil, domain.ErrNotFound
		}
		return nil, nil, err
	}
	if row.ProjectID != projectID {
		return nil, nil, domain.ErrNotFound
	}
	var sa domain.ServiceAccount
	if err := unmarshal(row.Data, &sa); err != nil {
		return nil, nil, err
	}
	return row, &sa, nil
}

// List returns all service accounts for a project.
func (a *pgAdminServiceAccounts) List(ctx context.Context, projectID string) ([]domain.ServiceAccount, error) {
	rows, err := models.IamServiceAccounts.Query(
		sm.Where(models.IamServiceAccounts.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.ServiceAccount, 0, len(rows))
	for _, row := range rows {
		var sa domain.ServiceAccount
		if err := unmarshal(row.Data, &sa); err != nil {
			return nil, err
		}
		out = append(out, sa)
	}
	return out, nil
}

// Get returns a single service account scoped by project.
func (a *pgAdminServiceAccounts) Get(ctx context.Context, projectID, saID string) (*domain.ServiceAccount, error) {
	_, sa, err := a.findSA(ctx, projectID, saID)
	return sa, err
}

// Create inserts a new service account for the project.
func (a *pgAdminServiceAccounts) Create(ctx context.Context, cmd domain.ServiceAccountCmd) (*domain.ServiceAccount, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.ServiceAccount, error) {
		sa := &domain.ServiceAccount{
			ID:        newUUID(),
			ProjectID: cmd.ProjectID,
			Name:      cmd.Name,
			Scopes:    cmd.Scopes,
			Disabled:  false,
		}
		raw, err := marshal(sa)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if _, err := models.IamServiceAccounts.Insert(&models.IamServiceAccountSetter{
			ID:        &sa.ID,
			ProjectID: &sa.ProjectID,
			Name:      &sa.Name,
			Disabled:  &sa.Disabled,
			Data:      &rm,
		}).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		// TODO outbox event: service_account.created
		return sa, nil
	})
}

// Update applies a partial update (scopes / disabled flag) to a service account.
func (a *pgAdminServiceAccounts) Update(ctx context.Context, cmd domain.AdminServiceAccountUpdateCmd) (*domain.ServiceAccount, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.ServiceAccount, error) {
		row, sa, err := a.findSA(ctx, cmd.ProjectID, cmd.ServiceAccountID)
		if err != nil {
			return nil, err
		}
		if cmd.Scopes != nil {
			sa.Scopes = cmd.Scopes
		}
		sa.Disabled = cmd.Disabled
		raw, err := marshal(sa)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamServiceAccountSetter{
			Disabled:  &sa.Disabled,
			Data:      &rm,
			UpdatedAt: ptr(nowUTC()),
		}); err != nil {
			return nil, err
		}
		// TODO outbox event: service_account.updated
		return sa, nil
	})
}

// Delete removes a service account and its secrets for a project.
func (a *pgAdminServiceAccounts) Delete(ctx context.Context, projectID, saID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.findSA(ctx, projectID, saID)
		if err != nil {
			return err
		}
		// Cascade the service account's secrets (stored in iam_app_secrets, app_id = saID).
		secrets, err := models.IamAppSecrets.Query(
			sm.Where(models.IamAppSecrets.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamAppSecrets.Columns.AppID.EQ(psql.Arg(saID))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return err
		}
		for _, s := range secrets {
			if err := s.Delete(ctx, a.db.Bobx()); err != nil {
				return err
			}
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: service_account.deleted
		return nil
	})
}

// AddSecret mints a new client secret for the service account. The plaintext is
// returned once; only its sha256 hash is persisted (iam_app_secrets.app_id = saID).
func (a *pgAdminServiceAccounts) AddSecret(ctx context.Context, cmd domain.AdminServiceAccountSecretCmd) (*domain.AdminSecret, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminSecret, error) {
		// Tenant boundary.
		if _, _, err := a.findSA(ctx, cmd.ProjectID, cmd.ServiceAccountID); err != nil {
			return nil, err
		}
		// Mint an opaque secret; persist only its sha256 hash.
		secret, hash, err := adminRandomToken(32)
		if err != nil {
			return nil, err
		}
		secretID := newUUID()
		meta := map[string]any{"name": cmd.Name}
		if !cmd.ExpiresAt.IsZero() {
			meta["expires_at"] = cmd.ExpiresAt.UTC()
		}
		mraw, err := marshal(meta)
		if err != nil {
			return nil, err
		}
		mrm := json.RawMessage(mraw)
		if _, err := models.IamAppSecrets.Insert(&models.IamAppSecretSetter{
			ID:        &secretID,
			ProjectID: &cmd.ProjectID,
			AppID:     &cmd.ServiceAccountID,
			Hash:      &hash,
			Data:      &mrm,
		}).One(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		// TODO outbox event: service_account.secret_created
		return &domain.AdminSecret{
			SecretID:     secretID,
			ClientID:     cmd.ServiceAccountID,
			ClientSecret: secret,
		}, nil
	})
}

// DeleteSecret removes a single secret from a service account.
func (a *pgAdminServiceAccounts) DeleteSecret(ctx context.Context, projectID, saID, secretID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamAppSecret(ctx, a.db.Bobx(), secretID)
		if err != nil {
			if adminIsNotFound(translatePgErr("app_secret", err)) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.AppID != saID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: service_account.secret_deleted
		return nil
	})
}
