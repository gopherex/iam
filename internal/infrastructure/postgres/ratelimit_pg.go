package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// pgRateLimits reads a project's per-environment rate_limits doc (iam_config
// key="rate_limits") so the runtime limiter can override the hardcoded defaults.
// It is a read-only adapter scoped to (project_id, environment); a missing doc
// (or a missing/empty project context) yields no rules, in which case the
// middleware falls back to the built-in defaults.
type pgRateLimits struct{ db *DB }

// NewPgRateLimits builds the Postgres-backed RateLimitConfigReader adapter.
func NewPgRateLimits(db *DB) *pgRateLimits { return &pgRateLimits{db: db} }

var _ api.RateLimitConfigReader = (*pgRateLimits)(nil)

// RateLimitRules returns the project's effective, enforceable rate-limit rules
// for the requested environment. clientID is the X-Client-ID (the project id);
// env is the raw X-Environment header ("" => runtime default "live"). The
// middleware runs before EnvironmentMiddleware, so env is passed explicitly
// rather than resolved from ctx.
//
// Returns (nil, nil) when there is no project context or no stored doc — the
// caller then applies the hardcoded defaults. Only rules whose endpoint is a
// realized, classified path and whose subject is "ip" are returned; any other
// rule is dropped defensively (the write path already validates, configspec.go).
func (a *pgRateLimits) RateLimitRules(ctx context.Context, clientID, env string) ([]api.RateLimitRule, error) {
	if clientID == "" {
		return nil, nil // no project context -> defaults
	}
	if env == "" {
		env = runtimeDefaultEnv
	}

	row, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(clientID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(env))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("rate_limits"))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if errors.Is(translatePgErr("config", err), ErrNotFound) {
			return nil, nil // no doc -> defaults
		}
		return nil, err
	}
	if len(row.Data) == 0 {
		return nil, nil
	}

	spec, err := domain.ParseRateLimits(row.Data)
	if err != nil {
		return nil, err
	}

	rules := make([]api.RateLimitRule, 0, len(spec.Rules))
	for _, r := range spec.Rules {
		if r.Endpoint == nil || r.Limit == nil || r.WindowSeconds == nil || r.By == nil {
			continue
		}
		if *r.By != "ip" || !domain.RateLimitEndpoints.Has(*r.Endpoint) {
			continue
		}
		rules = append(rules, api.RateLimitRule{
			Endpoint: *r.Endpoint,
			Limit:    *r.Limit,
			Window:   time.Duration(*r.WindowSeconds) * time.Second,
			By:       *r.By,
		})
	}
	return rules, nil
}
