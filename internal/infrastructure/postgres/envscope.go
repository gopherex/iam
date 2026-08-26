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
	"strings"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// runtimeDefaultEnv is the environment a public request operates in when no
// X-Environment header is supplied. It MUST stay "live" so existing clients
// (which send nothing, or send "live") keep hitting the live data set.
const runtimeDefaultEnv = "live"

// effectiveEnv resolves the environment a public request operates in for
// projectID: the requested X-Environment when it names a real environment of
// the project, or runtimeDefaultEnv when none was requested (or the request
// asked for it directly, which needs no lookup). An unknown requested
// environment is rejected with domain.ErrBadRequest so a client cannot
// conjure rows under an arbitrary environment name.
func effectiveEnv(ctx context.Context, db *DB, projectID string) (string, error) {
	// An authenticated principal is bound to the environment its credential was
	// minted in (the token's env claim). The X-Environment header must not
	// override it, or a token minted for one environment could operate on
	// another's data — a cross-environment downgrade. Reject an explicit
	// mismatch; otherwise scope to the principal's own environment.
	if p, ok := api.PrincipalFrom(ctx); ok && p != nil && p.Environment != "" {
		reqEnv := api.EnvironmentFromContext(ctx)
		if reqEnv != "" && !strings.EqualFold(reqEnv, p.Environment) {
			return "", domain.ErrForbidden.WithMessage("environment mismatch: token is scoped to a different environment")
		}

		return p.Environment, nil
	}

	env := api.EnvironmentFromContext(ctx)
	if env == "" || env == runtimeDefaultEnv {
		return runtimeDefaultEnv, nil
	}

	if _, err := models.FindIamEnvironment(ctx, db.Bobx(), projectID, env); err != nil {
		return "", domain.ErrBadRequest.WithMessage("unknown environment: " + env)
	}

	return env, nil
}
