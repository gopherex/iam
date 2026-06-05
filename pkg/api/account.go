// Code scaffolded for IAM handler groups.
//
// AccountService is pure orchestration: it holds aggregate-port interfaces (deps) and
// nothing else. It embeds oas.UnimplementedHandler so any operation it does not
// override returns not-implemented, and panics on every v1.0.0 operation until
// written. Each port method is atomic in its adapter — services never open a
// transaction.

package api

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type AccountStore interface {
	Get(ctx context.Context, projectID, accountID string) (*domain.Account, error)
	UpdateProfile(ctx context.Context, cmd domain.ProfileUpdateCmd) (*domain.Account, error)
	Delete(ctx context.Context, projectID, accountID string) error
	ListSessions(ctx context.Context, accountID string) ([]domain.Session, error)
	RevokeSession(ctx context.Context, accountID, sessionID string) error
	ListIdentities(ctx context.Context, accountID string) ([]domain.Identity, error)
}

type AccountDeps struct{ Accounts AccountStore }

// AccountService implements the AccountHandler slice of oas.Handler.
type AccountService struct {
	oas.UnimplementedHandler
	deps AccountDeps
}

// NewAccountService builds the Account service from its dependencies.
func NewAccountService(deps AccountDeps) *AccountService { return &AccountService{deps: deps} }

var _ oas.Handler = (*AccountService)(nil)

func (s *AccountService) DeleteV1AuthIdentitiesByIdentityId(ctx context.Context, params oas.DeleteV1AuthIdentitiesByIdentityIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AccountService) DeleteV1Sessions(ctx context.Context, req oas.OptDeleteV1SessionsReq) (r *oas.DeleteV1SessionsOK, _ error) {
	panic("implement me")
}

func (s *AccountService) DeleteV1SessionsBySessionId(ctx context.Context, params oas.DeleteV1SessionsBySessionIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AccountService) DeleteV1UsersMe(ctx context.Context, req oas.OptDeleteV1UsersMeReq) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1AccountCapabilities(ctx context.Context) (r *oas.GetV1AccountCapabilitiesOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1AuthIdentities(ctx context.Context) (r *oas.GetV1AuthIdentitiesOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1Sessions(ctx context.Context) (r *oas.GetV1SessionsOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1SessionsCurrent(ctx context.Context) (r *oas.GetV1SessionsCurrentOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMe(ctx context.Context) (r *oas.GetV1UsersMeOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMeActivity(ctx context.Context, params oas.GetV1UsersMeActivityParams) (r *oas.GetV1UsersMeActivityOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMeConsents(ctx context.Context) (r *oas.GetV1UsersMeConsentsOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMeExportByJobId(ctx context.Context, params oas.GetV1UsersMeExportByJobIdParams) (r *oas.GetV1UsersMeExportByJobIdOK, _ error) {
	panic("implement me")
}

func (s *AccountService) PatchV1SessionsBySessionId(ctx context.Context, req *oas.PatchV1SessionsBySessionIdReq, params oas.PatchV1SessionsBySessionIdParams) (r *oas.PatchV1SessionsBySessionIdOK, _ error) {
	panic("implement me")
}

func (s *AccountService) PatchV1UsersMe(ctx context.Context, req *oas.PatchV1UsersMeReq) (r *oas.PatchV1UsersMeOK, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1AuthIdentitiesMergeConfirm(ctx context.Context, req *oas.PostV1AuthIdentitiesMergeConfirmReq) (r *oas.PostV1AuthIdentitiesMergeConfirmOK, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1AuthIdentitiesMergeStart(ctx context.Context, req *oas.PostV1AuthIdentitiesMergeStartReq) (r *oas.PostV1AuthIdentitiesMergeStartOK, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1SessionsBySessionIdTrust(ctx context.Context, req *oas.PostV1SessionsBySessionIdTrustReq, params oas.PostV1SessionsBySessionIdTrustParams) (r *oas.PostV1SessionsBySessionIdTrustOK, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1UsersMeConsents(ctx context.Context, req *oas.PostV1UsersMeConsentsReq) (r *oas.PostV1UsersMeConsentsOK, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1UsersMeExport(ctx context.Context) (r *oas.PostV1UsersMeExportOK, _ error) {
	panic("implement me")
}
