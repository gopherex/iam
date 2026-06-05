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
	ListServiceAccounts(ctx context.Context, cmd domain.MachineIDServiceAccountListCmd) (*domain.MachineIDServiceAccountPage, error)
	GetServiceAccount(ctx context.Context, projectID, serviceAccountID string) (*domain.ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, cmd domain.MachineIDServiceAccountPatchCmd) (*domain.ServiceAccount, error)
	DeleteServiceAccount(ctx context.Context, projectID, serviceAccountID string) error
	CreateServiceAccountSecret(ctx context.Context, cmd domain.MachineIDSecretCmd) (*domain.MachineIDSecret, error)
	RevokeServiceAccountSecret(ctx context.Context, projectID, serviceAccountID, secretID string) error
	MintToken(ctx context.Context, projectID, serviceAccountID string) (string, error)
	CreateAPIKey(ctx context.Context, cmd domain.APIKeyCmd) (*domain.APIKey, string, error)
	ListAPIKeys(ctx context.Context, projectID string) ([]*domain.APIKey, error)
	UpdateAPIKey(ctx context.Context, cmd domain.MachineIDAPIKeyPatchCmd) (*domain.APIKey, error)
	RotateAPIKey(ctx context.Context, projectID, keyID string) (*domain.APIKey, string, error)
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

// GetV1ProjectsByProjectIdAdminServiceAccounts lists service accounts in a project.
func (s *MachineIdentityService) GetV1ProjectsByProjectIdAdminServiceAccounts(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminServiceAccountsParams) (*oas.GetV1ProjectsByProjectIdAdminServiceAccountsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	page, err := s.deps.Keys.ListServiceAccounts(ctx, domain.MachineIDServiceAccountListCmd{
		ProjectID: params.ProjectID,
		Cursor:    params.Cursor.Or(""),
		Limit:     params.Limit.Or(0),
	})
	if err != nil {
		return nil, err
	}
	data := make([]oas.ServiceAccount, 0, len(page.Items))
	for _, sa := range page.Items {
		data = append(data, oasServiceAccount(sa))
	}
	return &oas.GetV1ProjectsByProjectIdAdminServiceAccountsOK{
		Data:       data,
		NextCursor: oas.NewOptNilString(page.NextCursor),
		HasMore:    oas.NewOptBool(page.HasMore),
	}, nil
}

// GetV1ProjectsByProjectIdAdminServiceAccountsBySaId fetches one service account.
func (s *MachineIdentityService) GetV1ProjectsByProjectIdAdminServiceAccountsBySaId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminServiceAccountsBySaIdParams) (*oas.GetV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	sa, err := s.deps.Keys.GetServiceAccount(ctx, params.ProjectID, params.SaID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK{
		ServiceAccount: oas.NewOptServiceAccount(oasServiceAccount(sa)),
	}, nil
}

// PatchV1ProjectsByProjectIdAdminServiceAccountsBySaId updates a service account.
func (s *MachineIdentityService) PatchV1ProjectsByProjectIdAdminServiceAccountsBySaId(ctx context.Context, req *oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdReq, params oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdParams) (*oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	cmd := domain.MachineIDServiceAccountPatchCmd{
		ProjectID:        params.ProjectID,
		ServiceAccountID: params.SaID,
		Scopes:           req.Scopes,
	}
	if v, ok := req.Disabled.Get(); ok {
		cmd.Disabled = &v
	}
	sa, err := s.deps.Keys.UpdateServiceAccount(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK{
		ServiceAccount: oas.NewOptServiceAccount(oasServiceAccount(sa)),
	}, nil
}

// DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaId deletes a service account.
func (s *MachineIdentityService) DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	if err := s.deps.Keys.DeleteServiceAccount(ctx, params.ProjectID, params.SaID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

// PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecrets mints a client secret.
func (s *MachineIdentityService) PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecrets(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsReq, params oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsParams) (*oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	cmd := domain.MachineIDSecretCmd{
		ProjectID:        params.ProjectID,
		ServiceAccountID: params.SaID,
		Name:             req.Name,
	}
	if v, ok := req.ExpiresAt.Get(); ok {
		cmd.ExpiresAt = &v
	}
	secret, err := s.deps.Keys.CreateServiceAccountSecret(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsCreated{
		SecretID:     oas.NewOptString(secret.SecretID),
		ClientID:     oas.NewOptString(secret.ClientID),
		ClientSecret: oas.NewOptString(secret.ClientSecret),
	}, nil
}

// DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsBySecretId revokes a secret.
func (s *MachineIdentityService) DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsBySecretId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsBySecretIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	if err := s.deps.Keys.RevokeServiceAccountSecret(ctx, params.ProjectID, params.SaID, params.SecretID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

// GetV1ProjectsByProjectIdAdminApiKeys lists API keys in a project.
func (s *MachineIdentityService) GetV1ProjectsByProjectIdAdminApiKeys(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminApiKeysParams) (*oas.GetV1ProjectsByProjectIdAdminApiKeysOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	keys, err := s.deps.Keys.ListAPIKeys(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.ApiKey, 0, len(keys))
	for _, k := range keys {
		data = append(data, oasApiKey(k))
	}
	return &oas.GetV1ProjectsByProjectIdAdminApiKeysOK{Data: data}, nil
}

// PatchV1ProjectsByProjectIdAdminApiKeysByKeyId updates API-key metadata/scopes.
func (s *MachineIdentityService) PatchV1ProjectsByProjectIdAdminApiKeysByKeyId(ctx context.Context, req *oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdReq, params oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdParams) (*oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	cmd := domain.MachineIDAPIKeyPatchCmd{
		ProjectID: params.ProjectID,
		KeyID:     params.KeyID,
		Name:      req.Name.Or(""),
		Scopes:    req.Scopes,
	}
	if v, ok := req.Disabled.Get(); ok {
		cmd.Disabled = &v
	}
	key, err := s.deps.Keys.UpdateAPIKey(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdOK{
		APIKey: oas.NewOptApiKey(oasApiKey(key)),
	}, nil
}

// PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotate rotates the key secret.
func (s *MachineIdentityService) PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotate(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotateParams) (*oas.PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotateOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}
	key, secret, err := s.deps.Keys.RotateAPIKey(ctx, params.ProjectID, params.KeyID)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotateOK{
		APIKey: oas.NewOptApiKey(oasApiKey(key)),
		Secret: oas.NewOptString(secret),
	}, nil
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
