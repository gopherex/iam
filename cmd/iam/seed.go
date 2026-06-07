package main

import (
	"context"
	"errors"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/infrastructure/postgres"
	"github.com/gopherex/xlog"
)

// seedRoot ensures a root project exists so a fresh deployment is immediately
// usable from the admin panel: the operator (master key) signs in and already
// has a project to manage. Idempotent — it does nothing once any project exists.
func seedRoot(ctx context.Context, db *postgres.DB, emitter postgres.Emitter, log *xlog.Logger) error {
	op := postgres.NewPgOperator(db, emitter)
	projects, err := op.ListProjects(ctx)
	if err != nil {
		return err
	}
	if len(projects) > 0 {
		return nil // already seeded / not empty
	}
	proj, err := op.CreateProject(ctx, domain.ProjectCmd{Name: "Root", Slug: "root", DefaultLocale: "en"})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil
		}
		return err
	}
	// The admin panel signs in with the master key (operator) and mints its own
	// per-project admin tokens on demand, so the seed creates only the project —
	// no admin token is minted or logged here (avoids a secret in the logs).
	log.Info("seeded root project", xlog.String("project_id", proj.ID))
	return nil
}
