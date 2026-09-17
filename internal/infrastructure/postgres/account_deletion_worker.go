package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/gopherex/xlog"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

const (
	deletionWorkerInterval = time.Minute
	deletionWorkerBatch    = 100
)

// RunDeletionWorker resumes committed deletion requests after restarts. Multiple
// replicas serialize on the account row; access is denied at the deadline even
// if cleanup is temporarily failing.
func (a *pgAccountStore) RunDeletionWorker(ctx context.Context, log *xlog.Logger) {
	ticker := time.NewTicker(deletionWorkerInterval)
	defer ticker.Stop()

	for {
		if err := a.SweepDeletions(ctx); err != nil && ctx.Err() == nil {
			log.Warn("account deletion sweep failed", xlog.Error("err", err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *pgAccountStore) SweepDeletions(ctx context.Context) error {
	rows, err := a.db.Pool.Query(
		ctx,
		`SELECT project_id,environment,user_id FROM iam_account_deletions
 WHERE status='pending' AND delete_at<=$1 ORDER BY delete_at LIMIT $2`,
		nowIn(ctx),
		deletionWorkerBatch)
	if err != nil {
		return securityStoreError(err)
	}

	scopes := []domain.SecurityScope{}

	for rows.Next() {
		var scope domain.SecurityScope
		if err := rows.Scan(&scope.ProjectID, &scope.Environment, &scope.AccountID); err != nil {
			rows.Close()
			return securityStoreError(err)
		}

		scopes = append(scopes, scope)
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		return securityStoreError(err)
	}

	var failures []error

	for _, scope := range scopes {
		if err := a.finishDeletion(api.WithEnvironment(ctx, scope.Environment), scope); err != nil {
			failures = append(failures, err)
		}
	}

	return errors.Join(failures...)
}

func (a *pgAccountStore) finishDeletion(ctx context.Context, scope domain.SecurityScope) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		// Same lock order as request/cancel. An administrative deletion may already
		// have removed the user, in which case cleanup still removes orphaned records.
		err := a.lockDeletionAccount(ctx, scope)
		if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
			return err
		}
		// The request remains the serialization point if an administrator already
		// removed the user. Two workers must not finalize an orphan twice.
		if _, err := a.db.TxDB.Exec(ctx,
			`SELECT user_id FROM iam_account_deletions
WHERE project_id=$1 AND environment=$2 AND user_id=$3 FOR UPDATE`,
			scope.ProjectID, scope.Environment, scope.AccountID); err != nil {
			return securityStoreError(err)
		}

		state, err := a.DeletionStatus(ctx, scope)
		if err != nil {
			return err
		}

		if state.Status != deletionPending || nowIn(ctx).Before(*state.DeleteAt) {
			return nil
		}

		if err := a.purgeDeletedAccount(ctx, scope); err != nil {
			return err
		}

		state.Status = deletionDeleted
		if err := a.saveDeletion(ctx, scope, state); err != nil {
			return err
		}

		return a.recordDeletion(ctx, scope, state, domain.WebhookEventUserDeleted, "")
	})
}

func (a *pgAccountStore) purgeDeletedAccount(ctx context.Context, scope domain.SecurityScope) error {
	core := NewPgCoreAuth(a.db, a.emitter, nil)
	if _, err := core.coreAuthSignOutAll(ctx, scope.ProjectID, scope.AccountID, ""); err != nil {
		return err
	}

	if err := a.purgeDeletionEnvelopes(ctx, scope); err != nil {
		return err
	}

	tables := []string{
		"iam_credentials",
		"iam_identities",
		"iam_refresh_tokens",
		"iam_factors",
		"iam_webauthn_credentials",
		"iam_recovery_codes",
		"iam_consents",
		"iam_user_roles",
		"iam_oauth_grants",
		"iam_auth_codes",
		"iam_device_codes",
		"iam_flows",
		"iam_activity",
		"iam_security_devices",
		"iam_security_guards",
		"iam_security_incidents",
		"iam_security_continuations",
		"iam_security_cases",
		"iam_security_deliveries",
	}
	for _, table := range tables {
		_, err := a.db.TxDB.Exec(
			ctx,
			`DELETE FROM `+table+` WHERE project_id=$1 AND environment=$2 AND user_id=$3`,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID)
		if err != nil {
			return securityStoreError(err)
		}
	}

	_, err := a.db.TxDB.Exec(
		ctx,
		`DELETE FROM iam_users WHERE id=$1 AND project_id=$2 AND environment=$3`,
		scope.AccountID,
		scope.ProjectID,
		scope.Environment)

	return securityStoreError(err)
}

func (a *pgAccountStore) purgeDeletionEnvelopes(ctx context.Context, scope domain.SecurityScope) error {
	statements := []string{
		`DELETE FROM iam_security_case_decisions WHERE project_id=$1 AND environment=$2 AND case_id IN
 (SELECT id FROM iam_security_cases WHERE project_id=$1 AND environment=$2 AND user_id=$3)`,
		`DELETE FROM iam_challenges WHERE project_id=$1 AND environment=$2 AND
 (subject=$3 OR data->>'account_id'=$3 OR data->>'AccountID'=$3 OR subject IN
 (SELECT primary_email FROM iam_users WHERE id=$3 UNION SELECT primary_phone FROM iam_users WHERE id=$3))`,
		`DELETE FROM iam_interactions WHERE project_id=$1 AND environment=$2 AND
 (data->>'account_id'=$3 OR data->>'user_id'=$3 OR data->>'AccountID'=$3)`,
	}
	for _, statement := range statements {
		if _, err := a.db.TxDB.Exec(
			ctx,
			statement,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID); err != nil {
			return securityStoreError(err)
		}
	}
	// These two legacy tables have no environment column; globally unique user
	// IDs provide ownership, and project scope is still enforced.
	for _, statement := range []string{
		`DELETE FROM iam_scim_resources WHERE project_id=$1 AND user_id=$2`,
		`DELETE FROM iam_jobs WHERE project_id=$1 AND (data->>'account_id'=$2 OR data->>'user_id'=$2)`,
	} {
		if _, err := a.db.TxDB.Exec(ctx, statement, scope.ProjectID, scope.AccountID); err != nil {
			return securityStoreError(err)
		}
	}

	return nil
}
