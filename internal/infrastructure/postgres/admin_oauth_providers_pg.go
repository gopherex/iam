package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// OAuth social providers are stored in iam_providers (kind=oauth) in exactly the
// oauthProviderData shape the runtime (oauthsocial_pg.loadOAuthConfig) reads, so
// the admin surface and the login runtime share one source of truth. The client
// secret is AES-GCM encrypted at rest (the runtime decrypts on read) and is
// never returned on list/read.

const adminOAuthProviderKind = "oauth"

// ListOAuthProviders returns the project's configured OAuth providers, secret
// omitted.
func (a *pgAdminConfig) ListOAuthProviders(ctx context.Context, projectID string) ([]domain.AdminOAuthProvider, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg(adminOAuthProviderKind))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, fmt.Errorf("list oauth providers: %w", err)
	}

	out := make([]domain.AdminOAuthProvider, 0, len(rows))
	for _, row := range rows {
		var providerData oauthProviderData

		_ = json.Unmarshal(row.Data, &providerData)

		out = append(out, domain.AdminOAuthProvider{
			ID:       row.ID,
			Provider: row.Provider,
			ClientID: providerData.ClientID,
			Scopes:   providerData.Scopes,
			Enabled:  row.Enabled,
		})
	}

	return out, nil
}

// CreateOAuthProvider inserts a new provider, encrypting the client secret.
func (a *pgAdminConfig) CreateOAuthProvider(
	ctx context.Context,
	projectID string,
	p domain.AdminOAuthProvider,
) (domain.AdminOAuthProvider, error) {
	if p.Provider == "" {
		return domain.AdminOAuthProvider{}, domain.ErrValidation.WithMessage("provider is required")
	}

	return withTxRet(ctx, a.db, func(ctx context.Context) (domain.AdminOAuthProvider, error) {
		id := p.ID
		if id == "" {
			id = newUUID()
		}

		enc, err := a.db.Cipher.Encrypt(p.ClientSecret)
		if err != nil {
			return domain.AdminOAuthProvider{}, err
		}

		//nolint:gosec // ClientSecret here is enc, the ciphertext, not the
		// plaintext gosec's field-name pattern match assumes.
		raw, err := json.Marshal(oauthProviderData{
			Name: p.Provider, ClientID: p.ClientID, ClientSecret: enc, Scopes: p.Scopes,
		})
		if err != nil {
			return domain.AdminOAuthProvider{}, fmt.Errorf("marshal provider config: %w", err)
		}

		rm := json.RawMessage(raw)
		kind := adminOAuthProviderKind

		if _, err := models.IamProviders.Insert(&models.IamProviderSetter{
			ID: &id, ProjectID: &projectID, Kind: &kind, Provider: &p.Provider,
			Enabled: &p.Enabled, Data: &rm,
		}).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return domain.AdminOAuthProvider{}, domain.ErrConflict
			}

			return domain.AdminOAuthProvider{}, fmt.Errorf("insert provider: %w", err)
		}

		p.ID = id
		p.ClientSecret = ""

		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "config.oauth_provider_created",
			ProjectID:   projectID,
			AggregateID: id,
			Payload:     map[string]any{"id": id, "provider": p.Provider},
		}); err != nil {
			return domain.AdminOAuthProvider{}, err
		}

		return p, nil
	})
}

// UpdateOAuthProvider replaces a provider by id. An empty ClientSecret keeps the
// stored one (secret is write-only, so the client cannot read it to resend).
func (a *pgAdminConfig) UpdateOAuthProvider(
	ctx context.Context,
	projectID, id string,
	p domain.AdminOAuthProvider,
) (domain.AdminOAuthProvider, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (domain.AdminOAuthProvider, error) {
		row, err := models.FindIamProvider(ctx, a.db.Bobx(), id)
		if err != nil || row.ProjectID != projectID || row.Kind != adminOAuthProviderKind {
			return domain.AdminOAuthProvider{}, domain.ErrNotFound
		}

		var cur oauthProviderData

		_ = json.Unmarshal(row.Data, &cur)

		secret := cur.ClientSecret
		if p.ClientSecret != "" {
			enc, encErr := a.db.Cipher.Encrypt(p.ClientSecret)
			if encErr != nil {
				return domain.AdminOAuthProvider{}, encErr
			}

			secret = enc
		}

		provider := p.Provider
		if provider == "" {
			provider = row.Provider
		}

		//nolint:gosec // ClientSecret here is already the ciphertext (either
		// re-encrypted above or carried over from the stored row), not the
		// plaintext gosec's field-name pattern match assumes.
		raw, err := json.Marshal(oauthProviderData{
			Name: provider, ClientID: p.ClientID, ClientSecret: secret, Scopes: p.Scopes,
		})
		if err != nil {
			return domain.AdminOAuthProvider{}, fmt.Errorf("marshal provider config: %w", err)
		}

		rm := json.RawMessage(raw)
		now := nowUTC()

		if err := row.Update(ctx, a.db.Bobx(), &models.IamProviderSetter{
			Provider: &provider, Enabled: &p.Enabled, Data: &rm, UpdatedAt: &now,
		}); err != nil {
			return domain.AdminOAuthProvider{}, err
		}

		if err := a.emitter.Emit(ctx, domain.Event{
			Type: "config.oauth_provider_updated", ProjectID: projectID, AggregateID: id,
			Payload: map[string]any{"id": id, "provider": provider},
		}); err != nil {
			return domain.AdminOAuthProvider{}, err
		}

		return domain.AdminOAuthProvider{
			ID: id, Provider: provider, ClientID: p.ClientID, Scopes: p.Scopes, Enabled: p.Enabled,
		}, nil
	})
}

// DeleteOAuthProvider removes a provider by id (project-scoped).
func (a *pgAdminConfig) DeleteOAuthProvider(ctx context.Context, projectID, id string) error {
	n, err := models.IamProviders.Delete(
		dm.Where(models.IamProviders.Columns.ID.EQ(psql.Arg(id))),
		dm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		dm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg(adminOAuthProviderKind))),
	).Exec(ctx, a.db.Bobx())
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}

	if n == 0 {
		return domain.ErrNotFound
	}

	return nil
}
