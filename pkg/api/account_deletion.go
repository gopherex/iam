package api

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type AccountDeletionStore interface {
	AdminCancelDeletion(
		ctx context.Context,
		scope domain.SecurityScope,
		input domain.AdminAccountDeletionCancelInput,
	) (*domain.AccountDeletion, error)
	DeletionStatus(ctx context.Context, scope domain.SecurityScope) (*domain.AccountDeletion, error)
	MutateDeletion(
		ctx context.Context,
		scope domain.SecurityScope,
		input domain.AccountDeletionInput,
		action string) (*domain.AccountDeletion, error)
	DeletionPolicy(ctx context.Context, scope domain.SecurityScope) (*domain.AccountDeletionPolicy, error)
	SetDeletionPolicy(
		ctx context.Context,
		scope domain.SecurityScope,
		policy domain.AccountDeletionPolicy) (*domain.AccountDeletionPolicy, error)
	DeletionHistory(ctx context.Context, scope domain.SecurityScope) (*domain.AccountDeletionHistory, error)
}

func (s *AccountService) deletionScope(ctx context.Context) (domain.SecurityScope, error) {
	if s.deps.Deletion == nil {
		return domain.SecurityScope{}, domain.ErrNotImplemented
	}

	principal, err := requirePrincipal(ctx)
	if err != nil {
		return domain.SecurityScope{}, err
	}

	return securityScope(ctx, principal.ProjectID, "user")
}

func (s *AccountService) DeleteV1UsersMe(
	ctx context.Context,
	req *oas.AccountDeletionInput,
) (*oas.AccountDeletion, error) {
	return s.mutateDeletion(ctx, req, "deletion_request")
}

func (s *AccountService) CancelAccountDeletion(
	ctx context.Context,
	req *oas.AccountDeletionInput,
) (*oas.AccountDeletion, error) {
	return s.mutateDeletion(ctx, req, "deletion_cancel")
}

func (s *AccountService) mutateDeletion(
	ctx context.Context,
	req *oas.AccountDeletionInput,
	action string,
) (*oas.AccountDeletion, error) {
	scope, err := s.deletionScope(ctx)
	if err != nil {
		return nil, err
	}

	in, err := securityInput[domain.AccountDeletionInput](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.AccountDeletion](s.deps.Deletion.MutateDeletion(ctx, scope, in, action))
}

func (s *AccountService) GetAccountDeletion(ctx context.Context) (*oas.AccountDeletion, error) {
	scope, err := s.deletionScope(ctx)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.AccountDeletion](s.deps.Deletion.DeletionStatus(ctx, scope))
}

func (s *AccountService) GetAccountDeletionPolicy(
	ctx context.Context,
	params oas.GetAccountDeletionPolicyParams,
) (*oas.AccountDeletionPolicy, error) {
	scope, err := securityScope(ctx, params.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	if s.deps.Deletion == nil {
		return nil, domain.ErrNotImplemented
	}

	return securityResponse[oas.AccountDeletionPolicy](s.deps.Deletion.DeletionPolicy(ctx, scope))
}

func (s *AccountService) PutAccountDeletionPolicy(
	ctx context.Context,
	req *oas.AccountDeletionPolicy,
	params oas.PutAccountDeletionPolicyParams,
) (*oas.AccountDeletionPolicy, error) {
	scope, err := securityScope(ctx, params.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	if s.deps.Deletion == nil {
		return nil, domain.ErrNotImplemented
	}

	policy, err := securityInput[domain.AccountDeletionPolicy](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.AccountDeletionPolicy](s.deps.Deletion.SetDeletionPolicy(ctx, scope, policy))
}

func (s *AccountService) GetAdminAccountDeletion(
	ctx context.Context,
	params oas.GetAdminAccountDeletionParams,
) (*oas.AccountDeletionHistory, error) {
	scope, err := securityScope(ctx, params.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	if s.deps.Deletion == nil {
		return nil, domain.ErrNotImplemented
	}

	scope.AccountID = params.UserID

	return securityResponse[oas.AccountDeletionHistory](s.deps.Deletion.DeletionHistory(ctx, scope))
}

func (s *AccountService) CancelAdminAccountDeletion(
	ctx context.Context,
	req *oas.AdminAccountDeletionCancelInput,
	params oas.CancelAdminAccountDeletionParams,
) (*oas.AccountDeletion, error) {
	scope, err := securityScope(ctx, params.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	if s.deps.Deletion == nil {
		return nil, domain.ErrNotImplemented
	}

	scope.AccountID = params.UserID

	input, err := securityInput[domain.AdminAccountDeletionCancelInput](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.AccountDeletion](s.deps.Deletion.AdminCancelDeletion(ctx, scope, input))
}
