package postgres

// Mint-environment resolution. A token is signed with a project+environment
// signing key; the environment comes from the X-Environment request header
// (lifted into ctx by api.EnvironmentMiddleware). Because Signer.activeKey
// auto-creates a signing key on first use for any project/env pair, a
// client-supplied environment MUST be validated against the project's declared
// environments before it is used — otherwise a caller could spawn signing keys
// for arbitrary environment names. The per-feature default (almost always
// "live") is trusted without a lookup.

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// resolveSignEnv returns the environment to mint a token in for projectID: the
// requested X-Environment when it names a real environment of the project, or
// fallback when none was requested. An unknown requested environment is rejected
// with domain.ErrBadRequest so callers cannot mint against an arbitrary env.
func resolveSignEnv(ctx context.Context, db *DB, projectID, fallback string) (string, error) {
	env := api.EnvironmentFromContext(ctx)
	if env == "" || env == fallback {
		return fallback, nil
	}
	if _, err := models.FindIamEnvironment(ctx, db.Bobx(), projectID, env); err != nil {
		return "", domain.ErrBadRequest.WithMessage("unknown environment: " + env)
	}
	return env, nil
}
