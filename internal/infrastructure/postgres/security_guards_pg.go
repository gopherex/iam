package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

func securityAccountGuard(ctx context.Context, db *DB, projectID, accountID string) error {
	env, err := effectiveEnv(ctx, db, projectID)
	if err != nil {
		return securityStoreError(err)
	}

	if err := accountDeletionAccess(ctx, db, projectID, env, accountID); err != nil {
		return err
	}

	var guarded bool
	if err := db.TxDB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM iam_security_guards WHERE project_id=$1 AND environment=$2
AND user_id=$3)`,
		projectID,
		env,
		accountID).Scan(&guarded); err != nil {
		return securityStoreError(err)
	}

	if guarded {
		return domain.ErrAccountLocked.WithMessage("Complete the account security recovery flow")
	}

	return nil
}

func (s *pgSecurity) guardSignIn(ctx context.Context, acc *domain.Account, aal int) error {
	if err := securityAccountGuard(ctx, s.db, acc.ProjectID, acc.ID); err != nil {
		return securityStoreError(err)
	}

	scope, err := s.scope(
		ctx,
		domain.SecurityScope{
			ProjectID:   acc.ProjectID,
			AccountID:   acc.ID,
			Environment: api.EnvironmentFromContext(ctx),
		})
	if err != nil {
		return securityStoreError(err)
	}

	p, err := s.Policy(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	if p.Mode != securityEnforce || !p.RequireNewDeviceProof || aal >= 2 {
		return nil
	}

	meta := domain.RequestMetaFromContext(ctx)
	if meta.SecurityProof != "" {
		return s.consumeSignInProof(ctx, scope, acc.ID, meta.SecurityProof)
	}

	if meta.DeviceToken != "" {
		var known bool

		err = s.db.TxDB.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM iam_security_devices WHERE project_id=$1 AND environment=$2
AND user_id=$3 AND token_hash=$4 AND NOT (data->>'revoked')::boolean)`,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID,
			accountHashToken(meta.DeviceToken)).Scan(&known)
		if err != nil {
			return securityStoreError(err)
		}

		if known {
			return nil
		}
	}

	var exists bool
	if err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM iam_users WHERE id=$1 AND created_at<$2) OR EXISTS(SELECT
1 FROM iam_sessions WHERE user_id=$1)`,
		acc.ID,
		nowIn(ctx).Add(-time.Minute)).Scan(&exists); err != nil {
		return securityStoreError(err)
	}

	if !exists {
		return nil
	} // initial registration has no previous account to prove

	return s.requireSignInProof(ctx, scope, meta)
}

func (s *pgSecurity) recordFailure(ctx context.Context, projectID, accountID, kind string) error {
	scope, err := s.scope(
		ctx,
		domain.SecurityScope{
			ProjectID:   projectID,
			AccountID:   accountID,
			Environment: api.EnvironmentFromContext(ctx),
		})
	if err != nil {
		return securityStoreError(err)
	}

	p, err := s.Policy(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	if p.Mode == securityDisabled {
		return nil
	}

	independent, cancel := context.WithTimeout(context.Background(), securityPersistenceTimeout)
	defer cancel()

	independent = api.WithEnvironment(independent, scope.Environment)
	independent = domain.WithRequestMeta(independent, domain.RequestMetaFromContext(ctx))

	return s.db.withTx(independent, func(ctx context.Context) error { //nolint:contextcheck // Separate commit.
		ok, err := s.throttle(
			ctx,
			scope,
			accountID,
			kind,
			p.FailureThreshold-1,
			time.Duration(p.FailureWindowSeconds)*time.Second)
		if err != nil {
			return securityStoreError(err)
		}

		if ok {
			return nil
		}

		a, err := s.account(ctx, scope)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}

		if err != nil {
			return securityStoreError(err)
		}

		private := securityVerifiedContacts(a)

		required, err := s.priorMFARequired(ctx, &securityFlow{
			ProjectID: scope.ProjectID, Environment: scope.Environment, AccountID: accountID, Cutoff: nowIn(ctx),
		})
		if err != nil {
			return err
		}

		private.MFARequired = required || kind == "mfa_failures"

		return s.recordIncident(
			ctx,
			scope,
			kind,
			"",
			"",
			"warning",
			"blocked",
			[]string{kind},
			private,
			p)
	})
}

func (s *pgSecurity) requireSignInProof(
	_ context.Context,
	scope domain.SecurityScope,
	meta domain.RequestMeta,
) error {
	var err error

	// The failed sign-in rolls its transaction back. Persist this proof flow on
	// an independent transaction so the returned continuation actually exists.
	independent, cancel := context.WithTimeout(context.Background(), securityPersistenceTimeout)
	defer cancel()

	independent = api.WithEnvironment(independent, scope.Environment)
	independent = domain.WithRequestMeta(independent, meta)

	var token string

	err = s.db.withTx(independent, func(ctx context.Context) error { //nolint:contextcheck // Separate commit.
		ok, err := s.throttle(
			ctx,
			scope,
			scope.AccountID,
			"signin_proof",
			securityDeliveryBurst,
			securityProofRateWindow)
		if err != nil {
			return securityStoreError(err)
		}

		if !ok {
			return domain.ErrRateLimited
		}

		f, t, err := s.newFlow(ctx, scope, securityReview)
		if err != nil {
			return securityStoreError(err)
		}

		token = t

		f.Purpose = securitySignin
		if err := s.startReview(ctx, scope, f, domain.SecurityFlowInput{}); err != nil {
			return securityStoreError(err)
		}

		return s.saveFlow(ctx, f, t)
	})
	if err != nil {
		return securityStoreError(err)
	}

	return domain.ErrStepUpRequired.WithDetails(map[string]any{"security_flow_token": token})
}

func (s *pgSecurity) consumeSignInProof(
	ctx context.Context,
	scope domain.SecurityScope,
	accountID, token string,
) error {
	f, err := s.loadFlow(ctx, scope, token)
	if err != nil {
		return securityStoreError(err)
	}

	if f.CurrentToken != token ||
		f.AccountID != accountID ||
		f.Purpose != securitySignin ||
		!f.Proved ||
		f.State.Outcome != securitySignInApproved {
		return domain.ErrInvalidToken
	}

	if err := s.validateFlowAuthority(ctx, f); err != nil {
		return err
	}

	f.Proved = false
	f.State.Outcome = "sign_in_proof_consumed"

	return s.saveFlow(ctx, f, token)
}
