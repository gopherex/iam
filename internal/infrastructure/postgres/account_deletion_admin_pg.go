package postgres

import (
	"context"
	"strings"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

// AdminCancelDeletion shares the worker's account lock. Administrative authority
// replaces the user's password proof; cancellation never changes a ban or guard.
func (a *pgAccountStore) AdminCancelDeletion(
	ctx context.Context,
	scope domain.SecurityScope,
	input domain.AdminAccountDeletionCancelInput,
) (*domain.AccountDeletion, error) {
	if scope.ActorID == "" {
		return nil, domain.ErrUnauthorized
	}

	input.Reason = strings.TrimSpace(input.Reason)
	if input.RequestID == "" || input.Reason == "" {
		return nil, domain.ErrValidation.WithMessage("Request ID and cancellation reason are required")
	}

	scope, err := NewPgSecurity(a.db, a.emitter).scope(ctx, scope)
	if err != nil {
		return nil, err
	}

	ctx = api.WithEnvironment(ctx, scope.Environment)

	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AccountDeletion, error) {
		if err := a.lockDeletionAccount(ctx, scope); err != nil {
			return nil, err
		}

		state, err := a.DeletionStatus(ctx, scope)
		if err != nil {
			return nil, err
		}

		if state.RequestID != input.RequestID {
			return nil, domain.ErrConflict.WithMessage("The deletion request changed; refresh the user details")
		}

		if state.Status == deletionCancelled {
			return state, nil
		}

		if state.Status != deletionPending || state.DeleteAt == nil || !nowIn(ctx).Before(*state.DeleteAt) {
			return nil, domain.ErrConflict.WithMessage("The account deletion deadline has passed")
		}

		return a.changeDeletion(ctx, scope, state, deletionCancel, input.Reason)
	})
}
