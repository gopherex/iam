// Code scaffolded for IAM handler groups.
//
// MachineIdentityService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type MachineIdentities interface {
	CreateServiceAccount(ctx context.Context, cmd domain.ServiceAccountCmd) (*domain.ServiceAccount, error)
	MintToken(ctx context.Context, projectID, serviceAccountID string) (string, error)
	CreateAPIKey(ctx context.Context, cmd domain.APIKeyCmd) (*domain.APIKey, string, error)
	RevokeAPIKey(ctx context.Context, projectID, keyID string) error
}

type MachineIdentityDeps struct{ Keys MachineIdentities }

// MachineIdentityService implements the MachineIdentityHandler slice of oas.Handler.
type MachineIdentityService struct {
	oas.UnimplementedHandler
	deps MachineIdentityDeps
}

// NewMachineIdentityService builds the MachineIdentity service from its dependencies.
func NewMachineIdentityService(deps MachineIdentityDeps) *MachineIdentityService {
	return &MachineIdentityService{deps: deps}
}

var _ oas.Handler = (*MachineIdentityService)(nil)

// PostV1ProjectsByProjectIdAdminServiceAccounts creates a service account in a project.
func (s *MachineIdentityService) PostV1ProjectsByProjectIdAdminServiceAccounts(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminServiceAccountsReq, params oas.PostV1ProjectsByProjectIdAdminServiceAccountsParams) (*oas.PostV1ProjectsByProjectIdAdminServiceAccountsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	sa, err := s.deps.Keys.CreateServiceAccount(ctx, domain.ServiceAccountCmd{
		ProjectID: params.ProjectID,
		Name:      req.Name,
		Scopes:    req.Scopes,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ProjectsByProjectIdAdminServiceAccountsCreated{
		ServiceAccount: oas.NewOptServiceAccount(oasServiceAccount(sa)),
	}, nil
}

// PostV1ServiceAccountsTokens mints an access token for the calling service account.
func (s *MachineIdentityService) PostV1ServiceAccountsTokens(ctx context.Context, req *oas.PostV1ServiceAccountsTokensReq) (*oas.PostV1ServiceAccountsTokensOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	token, err := s.deps.Keys.MintToken(ctx, p.ProjectID, p.AccountID)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ServiceAccountsTokensOK{
		AccessToken: oas.NewOptString(token),
		TokenType:   oas.NewOptString("Bearer"),
	}, nil
}

// PostV1ProjectsByProjectIdAdminApiKeys creates an API key in a project.
func (s *MachineIdentityService) PostV1ProjectsByProjectIdAdminApiKeys(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminApiKeysReq, params oas.PostV1ProjectsByProjectIdAdminApiKeysParams) (*oas.PostV1ProjectsByProjectIdAdminApiKeysCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	key, secret, err := s.deps.Keys.CreateAPIKey(ctx, domain.APIKeyCmd{
		ProjectID: params.ProjectID,
		Name:      req.Name,
		Scopes:    req.Scopes,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ProjectsByProjectIdAdminApiKeysCreated{
		APIKey: oas.NewOptApiKey(oasApiKey(key)),
		Secret: oas.NewOptString(secret),
	}, nil
}

// DeleteV1ProjectsByProjectIdAdminApiKeysByKeyId revokes an API key.
func (s *MachineIdentityService) DeleteV1ProjectsByProjectIdAdminApiKeysByKeyId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminApiKeysByKeyIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	if err := s.deps.Keys.RevokeAPIKey(ctx, params.ProjectID, params.KeyID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

// oasServiceAccount maps a domain ServiceAccount to its oas representation.
func oasServiceAccount(sa *domain.ServiceAccount) oas.ServiceAccount {
	return oas.ServiceAccount{
		ID:       oas.NewOptString(sa.ID),
		Name:     oas.NewOptString(sa.Name),
		Scopes:   sa.Scopes,
		Disabled: oas.NewOptBool(sa.Disabled),
	}
}

// oasApiKey maps a domain APIKey to its oas representation.
func oasApiKey(k *domain.APIKey) oas.ApiKey {
	return oas.ApiKey{
		ID:       oas.NewOptString(k.ID),
		Name:     oas.NewOptString(k.Name),
		Scopes:   k.Scopes,
		Prefix:   oas.NewOptString(k.Prefix),
		Disabled: oas.NewOptBool(k.Disabled),
	}
}
