package postgres

import (
	"context"
	"errors"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// pgPlatform serves unauthenticated bootstrap config assembled from three
// envelopes: iam_projects (name + supported locales), iam_config (key=auth,
// the project auth policy: enabled methods + default locale) and iam_providers
// (the enabled social/OIDC providers). It is a read-only adapter: no mutation,
// every query is scoped to the requested project (tenant boundary).
type pgPlatform struct{ db *DB }

// NewPgPlatform builds the Postgres-backed PlatformConfig adapter.
func NewPgPlatform(db *DB) *pgPlatform { return &pgPlatform{db: db} }

var _ api.PlatformConfig = (*pgPlatform)(nil)

// platformDefaultEnvironment is the environment whose auth config bootstraps a
// public client. iam_config is keyed (project_id, environment, key).
const platformDefaultEnvironment = "live"

// platformAuthConfig mirrors the auth-policy fields persisted in the
// iam_config(key=auth) data envelope. Columns on iam_config are lookup-only;
// the policy itself lives in the jsonb blob.
type platformAuthConfig struct {
	Methods       []string `json:"methods"`
	Locales       []string `json:"locales"`
	DefaultLocale string   `json:"default_locale"`
}

type platformConsentConfig struct {
	Documents []domain.ConsentDocument `json:"documents"`
}

// PublicConfig assembles domain.PublicConfig for the (projectID) tenant. The
// clientID is accepted for interface parity / future per-client overrides; the
// current bootstrap is project-wide. A missing project is a not-found.
func (a *pgPlatform) PublicConfig(ctx context.Context, projectID, clientID string) (*domain.PublicConfig, error) {
	_ = clientID // reserved for per-client config overrides

	// Project envelope: name + supported locales (tenant boundary).
	projRow, err := models.FindIamProject(ctx, a.db.Bobx(), projectID)
	if err != nil {
		// FindIamProject returns pgx.ErrNoRows when absent; translatePgErr maps
		// that onto the package ErrNotFound sentinel.
		if errors.Is(translatePgErr("project", err), ErrNotFound) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}

	cfg := &domain.PublicConfig{ProjectName: projRow.Name}

	var proj domain.Project
	if len(projRow.Data) > 0 {
		if err := unmarshal(projRow.Data, &proj); err != nil {
			return nil, err
		}
		cfg.ProjectName = proj.Name
		cfg.Locales = proj.SupportedLocales
		cfg.DefaultLocale = proj.DefaultLocale
	}

	// Auth config envelope: iam_config(project_id, environment=live, key=auth).
	authRow, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(platformDefaultEnvironment))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("auth"))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if !errors.Is(translatePgErr("config", err), ErrNotFound) {
			return nil, err
		}
		// No explicit auth policy yet: leave defaults derived from the project.
	} else if len(authRow.Data) > 0 {
		var ac platformAuthConfig
		if err := unmarshal(authRow.Data, &ac); err != nil {
			return nil, err
		}
		cfg.Methods = ac.Methods
		if len(ac.Locales) > 0 {
			cfg.Locales = ac.Locales
		}
		if ac.DefaultLocale != "" {
			cfg.DefaultLocale = ac.DefaultLocale
		}
	}

	// Providers envelope: only enabled providers for the project are surfaced.
	provRows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Enabled.EQ(psql.Arg(true))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	for _, p := range provRows {
		cfg.Providers = append(cfg.Providers, platformProviderToDomain(p))
	}

	consentRow, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(platformDefaultEnvironment))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("consent"))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if !errors.Is(translatePgErr("config", err), ErrNotFound) {
			return nil, err
		}
	} else if len(consentRow.Data) > 0 {
		var cc platformConsentConfig
		if err := unmarshal(consentRow.Data, &cc); err != nil {
			return nil, err
		}
		cfg.ConsentDocuments = cc.Documents
	}

	return cfg, nil
}

// platformOAuthProvider mirrors the provider read-model fields persisted in the
// iam_providers data envelope (name + granted scopes). Columns are lookup-only.
type platformOAuthProvider struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// platformProviderToDomain maps an iam_providers envelope row to the public
// OAuthProvider read model, preferring the jsonb display name over the
// provider key column.
func platformProviderToDomain(row *models.IamProvider) domain.OAuthProvider {
	out := domain.OAuthProvider{ID: row.ID, Name: row.Provider}
	if len(row.Data) > 0 {
		var p platformOAuthProvider
		if err := unmarshal(row.Data, &p); err == nil {
			if p.Name != "" {
				out.Name = p.Name
			}
			out.Scopes = p.Scopes
		}
	}
	return out
}
