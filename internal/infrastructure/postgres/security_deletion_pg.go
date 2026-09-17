package postgres

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
)

func recoveryDeletionAllowed(f *securityFlow) bool {
	return f.Proved && f.AccountID != "" && !isDeletionProof(f) &&
		f.Purpose != securityTrust && f.Purpose != securitySignin
}

// Ownership proof grants this explicit action without issuing a normal session
// or clearing the recovery guard. The worker uses the same account lock.
func (s *pgSecurity) cancelRecoveryDeletion(ctx context.Context, f *securityFlow, requestID string) error {
	if !recoveryDeletionAllowed(f) {
		return domain.ErrStepUpRequired
	}

	if requestID == "" {
		return domain.ErrValidation
	}

	account := NewPgAccountStore(s.db, s.emitter)
	scope := securityFlowScope(f)

	if err := account.lockDeletionAccount(ctx, scope); err != nil {
		return err
	}

	if err := s.validateFlowAuthority(ctx, f); err != nil {
		return err
	}

	state, err := account.DeletionStatus(ctx, scope)
	if err != nil {
		return err
	}

	if state.RequestID != requestID {
		return domain.ErrConflict.WithMessage("The deletion request changed")
	}

	if state.Status == deletionCancelled {
		return nil
	}

	if state.Status != deletionPending || state.DeleteAt == nil || !nowIn(ctx).Before(*state.DeleteAt) {
		return domain.ErrConflict.WithMessage("The account deletion deadline has passed")
	}

	_, err = account.changeDeletion(ctx, scope, state, deletionCancel,
		"Cancelled by the user during verified account recovery")

	return err
}

func (s *pgSecurity) viewRecoveryDeletion(
	ctx context.Context, f *securityFlow, view *domain.SecurityFlowState,
) (*domain.SecurityFlowState, error) {
	if !recoveryDeletionAllowed(f) {
		return view, nil
	}

	deletion, err := NewPgAccountStore(s.db, s.emitter).DeletionStatus(ctx, securityFlowScope(f))
	if err != nil {
		return nil, err
	}

	if deletion.Status != "none" {
		view.Deletion = deletion
	}

	if deletion.Status == deletionPending && deletion.DeleteAt != nil && nowIn(ctx).Before(*deletion.DeleteAt) {
		view.NextActions = append(view.NextActions, "cancel_deletion")
	}

	return view, nil
}
