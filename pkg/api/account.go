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

func (s *AccountService) DeleteV1SessionsBySessionId(ctx context.Context, params oas.DeleteV1SessionsBySessionIdParams) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.RevokeSession(ctx, p.AccountID, params.SessionID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AccountService) DeleteV1UsersMe(ctx context.Context, req oas.OptDeleteV1UsersMeReq) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.Delete(ctx, p.ProjectID, p.AccountID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AccountService) GetV1AccountCapabilities(ctx context.Context) (r *oas.GetV1AccountCapabilitiesOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1AuthIdentities(ctx context.Context) (*oas.GetV1AuthIdentitiesOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ids, err := s.deps.Accounts.ListIdentities(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.Identity, 0, len(ids))
	for i := range ids {
		data = append(data, oasIdentity(&ids[i]))
	}
	return &oas.GetV1AuthIdentitiesOK{Data: data}, nil
}

func (s *AccountService) GetV1Sessions(ctx context.Context) (*oas.GetV1SessionsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := s.deps.Accounts.ListSessions(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.Session, 0, len(sessions))
	for i := range sessions {
		data = append(data, oasSession(&sessions[i]))
	}
	return &oas.GetV1SessionsOK{Data: data}, nil
}

func (s *AccountService) GetV1SessionsCurrent(ctx context.Context) (r *oas.GetV1SessionsCurrentOK, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMe(ctx context.Context) (*oas.GetV1UsersMeOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.deps.Accounts.Get(ctx, p.ProjectID, p.AccountID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1UsersMeOK{User: oas.NewOptUser(oasUser(acct))}, nil
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

func (s *AccountService) PatchV1UsersMe(ctx context.Context, req *oas.PatchV1UsersMeReq) (*oas.PatchV1UsersMeOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.deps.Accounts.UpdateProfile(ctx, domain.ProfileUpdateCmd{
		ProjectID: p.ProjectID,
		AccountID: p.AccountID,
		Name:      req.Name.Or(""),
		AvatarURL: req.AvatarURL.Or(""),
		Locale:    req.Locale.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.PatchV1UsersMeOK{User: oas.NewOptUser(oasUser(acct))}, nil
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

// oasIdentity maps a domain Identity to its wire representation.
func oasIdentity(i *domain.Identity) oas.Identity {
	id := oas.Identity{
		ID:   i.ID,
		Type: oas.IdentityType(i.Type),
	}
	if i.Provider != "" {
		id.Provider = oas.NewOptNilString(i.Provider)
	}
	if i.ProviderAccountID != "" {
		id.ProviderAccountID = oas.NewOptNilString(i.ProviderAccountID)
	}
	if i.Email != "" {
		id.Email = oas.NewOptNilString(i.Email)
	}
	return id
}
