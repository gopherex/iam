package postgres

// Role assignments (iam_user_roles).
//
// A role is a plain label owned by IAM and scoped to a project ENVIRONMENT, so
// the same person can be an operator in test and a viewer in live. Roles are the
// only source of the OIDC `groups` claim: a client can ask for the `groups`
// scope, but the values always come from what an admin assigned here — never
// from anything the client sent.
//
// The table is a plain join table rather than the usual jsonb envelope: the read
// on the token path is "the roles of this user in this environment", and that is
// exactly one index lookup here.

import (
	"context"
	"fmt"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

// eventFieldUserID is the user key inside emitted event payloads.
const eventFieldUserID = "user_id"

// pgRoles is the Postgres-backed role-assignment store. It is shared by the
// admin API and the token-minting path.
type pgRoles struct {
	db      *DB
	emitter Emitter
}

// NewPgRoles builds the role-assignment adapter over db.
func NewPgRoles(db *DB, emitter Emitter) *pgRoles {
	return &pgRoles{db: db, emitter: emitter}
}

var _ api.AdminRoles = (*pgRoles)(nil)

// ListRoles returns the roles assigned to a user in one project environment,
// sorted, so the `groups` claim is stable across tokens.
func (a *pgRoles) ListRoles(ctx context.Context, cmd domain.AdminUserRolesCmd) ([]string, error) {
	return userRoles(ctx, a.db, cmd.ProjectID, adminEnv(cmd.Environment), cmd.UserID)
}

// SetRoles replaces a user's roles with exactly cmd.Roles, in one transaction.
// It is a desired-state write: a role missing from the list is unassigned.
func (a *pgRoles) SetRoles(ctx context.Context, cmd domain.AdminUserRolesSetCmd) ([]string, error) {
	roles, err := domain.NormalizeRoles(cmd.Roles)
	if err != nil {
		return nil, err
	}

	env := adminEnv(cmd.Environment)

	return withTxRet(ctx, a.db, func(ctx context.Context) ([]string, error) {
		if _, err := a.db.TxDB.Exec(ctx,
			`DELETE FROM iam_user_roles WHERE project_id = $1 AND environment = $2 AND user_id = $3`,
			cmd.ProjectID, env, cmd.UserID,
		); err != nil {
			return nil, fmt.Errorf("clear user roles: %w", err)
		}

		for _, role := range roles {
			if _, err := a.db.TxDB.Exec(ctx,
				`INSERT INTO iam_user_roles (project_id, environment, user_id, role) VALUES ($1, $2, $3, $4)`,
				cmd.ProjectID, env, cmd.UserID, role,
			); err != nil {
				return nil, fmt.Errorf("assign user role %q: %w", role, err)
			}
		}

		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.roles.updated",
			ProjectID:   cmd.ProjectID,
			Environment: env,
			AggregateID: cmd.UserID,
			Payload:     map[string]any{eventFieldUserID: cmd.UserID, "roles": roles},
		}); err != nil {
			return nil, err
		}

		return roles, nil
	})
}

// userRoles reads a user's roles. It is a package-level helper (not a method) so
// the OIDC token path can project the `groups` claim without depending on the
// admin adapter.
func userRoles(ctx context.Context, db *DB, projectID, env, userID string) ([]string, error) {
	if projectID == "" || userID == "" {
		return nil, nil
	}

	rows, err := db.TxDB.Query(ctx,
		`SELECT role FROM iam_user_roles
		  WHERE project_id = $1 AND environment = $2 AND user_id = $3
		  ORDER BY role`,
		projectID, env, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("read user roles: %w", err)
	}
	defer rows.Close()

	var out []string

	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}

		out = append(out, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read user roles: %w", err)
	}

	return out, nil
}

// deleteUserRoles drops every role assignment for a user. Called when the user
// is deleted so the assignments cannot outlive them and be inherited by a
// recreated id. Assumes an ambient transaction.
func deleteUserRoles(ctx context.Context, db *DB, projectID, env, userID string) error {
	if _, err := db.TxDB.Exec(ctx,
		`DELETE FROM iam_user_roles WHERE project_id = $1 AND environment = $2 AND user_id = $3`,
		projectID, env, userID,
	); err != nil {
		return fmt.Errorf("delete user roles: %w", err)
	}

	return nil
}
