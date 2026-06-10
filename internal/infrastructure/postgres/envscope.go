package postgres

// Request-environment resolution for the runtime (public auth) data path.
//
// Stripe-like test/live/staging isolation keys every runtime row on
// (project_id, environment, …). The environment a request operates in is the
// X-Environment header, lifted into ctx by api.EnvironmentMiddleware. The public
// handlers run with no fixed environment, so the persistence layer resolves it
// from ctx here: a request without the header (or with the project's default
// "live") operates in the default environment for back-compat; a request naming
// another environment is scoped to it, but only after the environment is
// confirmed to exist on the project so a caller cannot conjure rows (and, via
// the Signer, signing keys) for an arbitrary environment name.

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// runtimeDefaultEnv is the environment a public request operates in when no
// X-Environment header is supplied. It MUST stay "live" so existing clients
// (which send nothing, or send "live") keep hitting the live data set.
const runtimeDefaultEnv = "live"

// effectiveEnv resolves the environment a public request operates in for
// projectID: the requested X-Environment when it names a real environment of the
// project, or fallback when none was requested (or the request asked for the
// fallback itself, which needs no lookup). An unknown requested environment is
// rejected with domain.ErrBadRequest so a client cannot create rows under an
// arbitrary environment name.
func effectiveEnv(ctx context.Context, db *DB, projectID, fallback string) (string, error) {
	env := api.EnvironmentFromContext(ctx)
	if env == "" || env == fallback {
		return fallback, nil
	}
	if _, err := models.FindIamEnvironment(ctx, db.Bobx(), projectID, env); err != nil {
		return "", domain.ErrBadRequest.WithMessage("unknown environment: " + env)
	}
	return env, nil
}
