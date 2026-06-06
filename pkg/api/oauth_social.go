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
	// StartLogin builds the provider authorize URL for a browser redirect.
	StartLogin(ctx context.Context, cmd domain.OAuthSocialStartCmd) (string, error)
	// CompleteLoginRedirect handles the provider callback and returns the
	// product redirect URL plus an optional Set-Cookie value (cookie mode).
	CompleteLoginRedirect(ctx context.Context, cmd domain.OAuthSocialCallbackCmd) (domain.OAuthSocialCallbackResult, error)
	// StartLink builds the provider authorize URL for an account-link flow.
	StartLink(ctx context.Context, cmd domain.OAuthSocialLinkStartCmd) (string, error)
	// CompleteLink handles the link callback and returns the product redirect URL.
	CompleteLink(ctx context.Context, cmd domain.OAuthSocialLinkCallbackCmd) (string, error)
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

// GetV1AuthOauthByProviderCallback handles the provider callback (public,
// security: []) and redirects the browser back to the product, optionally
// setting session cookies in cookie mode.
func (s *OAuthSocialService) GetV1AuthOauthByProviderCallback(ctx context.Context, params oas.GetV1AuthOauthByProviderCallbackParams) (r *oas.GetV1AuthOauthByProviderCallbackFound, _ error) {
	res, err := s.deps.Accounts.CompleteLoginRedirect(ctx, domain.OAuthSocialCallbackCmd{
		Provider: params.Provider,
		Code:     params.Code.Or(""),
		State:    params.State.Or(""),
		Error:    params.Error.Or(""),
	})
	if err != nil {
		return nil, err
	}
	out := &oas.GetV1AuthOauthByProviderCallbackFound{Location: optURI(res.RedirectURL)}
	if len(res.SetCookie) > 0 {
		out.SetCookie = res.SetCookie
	}
	return out, nil
}

// GetV1AuthOauthByProviderLinkCallback handles the account-link callback
// (public, security: []) and redirects the browser back to the product.
func (s *OAuthSocialService) GetV1AuthOauthByProviderLinkCallback(ctx context.Context, params oas.GetV1AuthOauthByProviderLinkCallbackParams) (r *oas.GetV1AuthOauthByProviderLinkCallbackFound, _ error) {
	url, err := s.deps.Accounts.CompleteLink(ctx, domain.OAuthSocialLinkCallbackCmd{
		Provider: params.Provider,
	})
	if err != nil {
		return nil, err
	}
	return &oas.GetV1AuthOauthByProviderLinkCallbackFound{Location: optURI(url)}, nil
}

// GetV1AuthOauthByProviderLinkStart begins linking a provider to the current
// user; the account comes from the authenticated principal, never the request.
func (s *OAuthSocialService) GetV1AuthOauthByProviderLinkStart(ctx context.Context, params oas.GetV1AuthOauthByProviderLinkStartParams) (r *oas.GetV1AuthOauthByProviderLinkStartFound, _ error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	url, err := s.deps.Accounts.StartLink(ctx, domain.OAuthSocialLinkStartCmd{
		AccountID:  p.AccountID,
		Provider:   params.Provider,
		RedirectTo: params.RedirectTo,
		State:      params.State.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.GetV1AuthOauthByProviderLinkStartFound{Location: optURI(url)}, nil
}

// GetV1AuthOauthByProviderStart begins a browser-driven social login (public,
// security: []) and redirects to the provider's authorize endpoint.
func (s *OAuthSocialService) GetV1AuthOauthByProviderStart(ctx context.Context, params oas.GetV1AuthOauthByProviderStartParams) (r *oas.GetV1AuthOauthByProviderStartFound, _ error) {
	url, err := s.deps.Accounts.StartLogin(ctx, domain.OAuthSocialStartCmd{
		Provider:      params.Provider,
		RedirectTo:    params.RedirectTo,
		State:         params.State.Or(""),
		CodeChallenge: params.CodeChallenge.Or(""),
		Prompt:        params.Prompt.Or(""),
		LoginHint:     params.LoginHint.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.GetV1AuthOauthByProviderStartFound{Location: optURI(url)}, nil
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
