// Code scaffolded for IAM handler groups.
//
// OAuthSocialService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type OAuthSocialAccounts interface {
	EnabledProviders(ctx context.Context, projectID string) ([]domain.OAuthProvider, error)
	CompleteLogin(ctx context.Context, projectID, provider, code string) (*domain.Account, *domain.Session, error)
	Link(ctx context.Context, accountID, provider, code string) error
	Unlink(ctx context.Context, accountID, identityID string) error
	Exchange(ctx context.Context, cmd domain.OAuthSocialExchangeCmd) (*domain.Account, *domain.Session, error)
}

type OAuthSocialDeps struct{ Accounts OAuthSocialAccounts }

// OAuthSocialService implements the OAuthSocialHandler slice of oas.Handler.
type OAuthSocialService struct {
	oas.UnimplementedHandler
	deps OAuthSocialDeps
}

// NewOAuthSocialService builds the OAuthSocial service from its dependencies.
func NewOAuthSocialService(deps OAuthSocialDeps) *OAuthSocialService {
	return &OAuthSocialService{deps: deps}
}

var _ oas.Handler = (*OAuthSocialService)(nil)

func (s *OAuthSocialService) GetV1AuthOauthByProviderCallback(ctx context.Context, params oas.GetV1AuthOauthByProviderCallbackParams) (r *oas.GetV1AuthOauthByProviderCallbackFound, _ error) {
	panic("implement me")
}

func (s *OAuthSocialService) GetV1AuthOauthByProviderLinkCallback(ctx context.Context, params oas.GetV1AuthOauthByProviderLinkCallbackParams) (r *oas.GetV1AuthOauthByProviderLinkCallbackFound, _ error) {
	panic("implement me")
}

func (s *OAuthSocialService) GetV1AuthOauthByProviderLinkStart(ctx context.Context, params oas.GetV1AuthOauthByProviderLinkStartParams) (r *oas.GetV1AuthOauthByProviderLinkStartFound, _ error) {
	panic("implement me")
}

func (s *OAuthSocialService) GetV1AuthOauthByProviderStart(ctx context.Context, params oas.GetV1AuthOauthByProviderStartParams) (r *oas.GetV1AuthOauthByProviderStartFound, _ error) {
	panic("implement me")
}

func (s *OAuthSocialService) GetV1AuthOauthProviders(ctx context.Context, params oas.GetV1AuthOauthProvidersParams) (*oas.GetV1AuthOauthProvidersOK, error) {
	providers, err := s.deps.Accounts.EnabledProviders(ctx, params.XClientID)
	if err != nil {
		return nil, err
	}
	items := make([]oas.GetV1AuthOauthProvidersOKProvidersItem, 0, len(providers))
	for _, p := range providers {
		items = append(items, oasOAuthProvider(p))
	}
	return &oas.GetV1AuthOauthProvidersOK{Providers: items}, nil
}

func (s *OAuthSocialService) PostV1AuthOauthByProviderUnlink(ctx context.Context, req *oas.PostV1AuthOauthByProviderUnlinkReq, params oas.PostV1AuthOauthByProviderUnlinkParams) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.Unlink(ctx, p.AccountID, req.IdentityID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *OAuthSocialService) PostV1AuthOauthExchange(ctx context.Context, req *oas.PostV1AuthOauthExchangeReq, params oas.PostV1AuthOauthExchangeParams) (*oas.AuthResult, error) {
	cmd := domain.OAuthSocialExchangeCmd{
		ProjectID:    params.XClientID,
		Code:         req.Code,
		CodeVerifier: req.CodeVerifier.Or(""),
	}
	acct, sess, err := s.deps.Accounts.Exchange(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

// oasOAuthProvider maps a domain OAuth provider to its OAS list item shape.
func oasOAuthProvider(p domain.OAuthProvider) oas.GetV1AuthOauthProvidersOKProvidersItem {
	return oas.GetV1AuthOauthProvidersOKProvidersItem{
		ID:     oas.NewOptString(p.ID),
		Name:   oas.NewOptString(p.Name),
		Scopes: p.Scopes,
	}
}
