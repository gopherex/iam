package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
)

// The throttle commits even for incorrect passwords; a rejected mutation cannot
// roll back the account-wide attempt budget.
func (a *pgAccountStore) deletionPassword(
	ctx context.Context,
	scope domain.SecurityScope,
	password string,
) (bool, error) {
	var (
		hasPassword bool
		denial      error
	)

	err := a.db.withTx(ctx, func(ctx context.Context) error {
		allowed, err := NewPgSecurity(a.db, a.emitter).throttle(
			ctx,
			scope,
			scope.AccountID,
			"account_deletion",
			securityProofBurst,
			securityProofRateWindow)
		if err != nil {
			return err
		}

		if !allowed {
			denial = domain.ErrRateLimited
			return nil
		}

		var secret string

		err = a.db.TxDB.QueryRow(
			ctx,
			`SELECT secret FROM iam_credentials WHERE project_id=$1 AND environment=$2 AND user_id=$3 AND type='password'`,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID).Scan(&secret)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		if err != nil {
			return securityStoreError(err)
		}

		hasPassword = true

		if !coreAuthCheckPassword(secret, password) {
			denial = domain.ErrInvalidCredentials
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return hasPassword, denial
}

func (a *pgAccountStore) deletionNeedsProof(
	ctx context.Context,
	scope domain.SecurityScope,
	hasPassword bool,
) (bool, error) {
	if !hasPassword {
		return true, nil
	}

	return NewPgSecurity(a.db, a.emitter).priorMFARequired(
		ctx,
		&securityFlow{
			ProjectID:   scope.ProjectID,
			Environment: scope.Environment,
			AccountID:   scope.AccountID,
			Cutoff:      nowIn(ctx),
		})
}

func (a *pgAccountStore) requireDeletionProof(
	ctx context.Context,
	scope domain.SecurityScope,
	in domain.AccountDeletionInput,
	action string,
	hasPassword bool,
) error {
	needed, err := a.deletionNeedsProof(ctx, scope, hasPassword)
	if err != nil {
		return err
	}

	if !needed || in.ProofToken != "" {
		return nil
	}

	var token string

	err = a.db.withTx(ctx, func(ctx context.Context) error {
		security := NewPgSecurity(a.db, a.emitter)

		flow, capability, err := security.newFlow(ctx, scope, deletionConfigKey)
		if err != nil {
			return err
		}

		if err := security.startReview(
			ctx,
			scope,
			flow,
			domain.SecurityFlowInput{SessionID: scope.SessionID}); err != nil {
			return err
		}

		flow.Purpose = action
		flow.State.ExpiresAt = nowIn(ctx).Add(deletionProofTTL)
		token = capability

		return security.saveFlow(ctx, flow, capability)
	})
	if err != nil {
		return err
	}

	return domain.ErrStepUpRequired.WithDetails(map[string]any{"security_flow_token": token, "purpose": action})
}

func (a *pgAccountStore) consumeDeletionProof(
	ctx context.Context,
	scope domain.SecurityScope,
	token, action string,
	hasPassword bool,
) error {
	needed, err := a.deletionNeedsProof(ctx, scope, hasPassword)
	if err != nil {
		return err
	}

	if !needed {
		return nil
	}

	security := NewPgSecurity(a.db, a.emitter)

	flow, err := security.loadFlow(ctx, scope, token)
	if err != nil {
		return err
	}

	if flow.CurrentToken != token || flow.Purpose != action || flow.AccountID != scope.AccountID ||
		flow.TargetSession != scope.SessionID || !flow.Proved || flow.SupportGrant ||
		flow.State.Status != securityCompleted || flow.State.Outcome != deletionAuthorized {
		return domain.ErrStepUpRequired
	}

	if err := security.validateFlowAuthority(ctx, flow); err != nil {
		return err
	}

	flow.State.Status = "consumed"
	flow.State.Version++

	return security.saveFlow(ctx, flow, token)
}

func isDeletionProof(flow *securityFlow) bool {
	return flow.Purpose == deletionRequest || flow.Purpose == deletionCancel
}

func authorizeDeletionProof(flow *securityFlow, in domain.SecurityFlowInput) error {
	if !flow.Proved || flow.SupportGrant || in.Action != "authorize_deletion" {
		return domain.ErrBadRequest
	}

	flow.State.Status = securityCompleted
	flow.State.Step = securityCompleted
	flow.State.Outcome = deletionAuthorized

	return nil
}

// Recheck under the mutation transaction: a password changed while the first
// check ran cannot authorize deletion. The lock protects this credential until commit.
func (a *pgAccountStore) recheckDeletionPassword(
	ctx context.Context,
	scope domain.SecurityScope,
	password string,
	hadPassword bool,
) error {
	var secret string

	err := a.db.TxDB.QueryRow(
		ctx,
		`SELECT secret FROM iam_credentials WHERE project_id=$1 AND environment=$2 AND user_id=$3
AND type='password' FOR SHARE`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID).Scan(&secret)
	if errors.Is(err, pgx.ErrNoRows) {
		if hadPassword {
			return domain.ErrInvalidCredentials
		}

		return nil
	}

	if err != nil {
		return securityStoreError(err)
	}

	if !hadPassword || !coreAuthCheckPassword(secret, password) {
		return domain.ErrInvalidCredentials
	}

	return nil
}
