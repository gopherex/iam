// Code scaffolded for IAM handler groups.
//
// OIDCProviderService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type OIDCGrants interface {
	ResolveInteraction(ctx context.Context, interactionID string) (*domain.Interaction, error)
	CompleteLogin(ctx context.Context, interactionID, accountID string) error
	ListGrants(ctx context.Context, accountID string) ([]domain.Grant, error)
	RevokeGrant(ctx context.Context, accountID, grantID string) error
}

type OIDCProviderDeps struct{ Grants OIDCGrants }

// OIDCProviderService implements the OIDCProviderHandler slice of oas.Handler.
type OIDCProviderService struct {
	oas.UnimplementedHandler
	deps OIDCProviderDeps
}

// NewOIDCProviderService builds the OIDCProvider service from its dependencies.
func NewOIDCProviderService(deps OIDCProviderDeps) *OIDCProviderService {
	return &OIDCProviderService{deps: deps}
}

var _ oas.Handler = (*OIDCProviderService)(nil)

func (s *OIDCProviderService) DeleteV1OauthGrantsByGrantId(ctx context.Context, params oas.DeleteV1OauthGrantsByGrantIdParams) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Grants.RevokeGrant(ctx, p.AccountID, params.GrantID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *OIDCProviderService) GetOauth2Authorize(ctx context.Context, params oas.GetOauth2AuthorizeParams) (r *oas.GetOauth2AuthorizeFound, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) GetOauth2Logout(ctx context.Context, params oas.GetOauth2LogoutParams) (r *oas.GetOauth2LogoutFound, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) GetOauth2Userinfo(ctx context.Context) (r oas.GetOauth2UserinfoOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) GetPByProjectIdEByEnvWellKnownJwksJson(ctx context.Context, params oas.GetPByProjectIdEByEnvWellKnownJwksJsonParams) (r oas.GetPByProjectIdEByEnvWellKnownJwksJsonOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) GetPByProjectIdEByEnvWellKnownOpenidConfiguration(ctx context.Context, params oas.GetPByProjectIdEByEnvWellKnownOpenidConfigurationParams) (r oas.GetPByProjectIdEByEnvWellKnownOpenidConfigurationOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) GetV1Device(ctx context.Context, params oas.GetV1DeviceParams) (r *oas.GetV1DeviceOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) GetV1OauthGrants(ctx context.Context, params oas.GetV1OauthGrantsParams) (*oas.GetV1OauthGrantsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	grants, err := s.deps.Grants.ListGrants(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.OAuthGrant, 0, len(grants))
	for i := range grants {
		data = append(data, oasOAuthGrant(grants[i]))
	}
	return &oas.GetV1OauthGrantsOK{Data: data}, nil
}

func (s *OIDCProviderService) GetV1OauthInteractionByInteractionId(ctx context.Context, params oas.GetV1OauthInteractionByInteractionIdParams) (*oas.GetV1OauthInteractionByInteractionIdOK, error) {
	in, err := s.deps.Grants.ResolveInteraction(ctx, params.InteractionID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1OauthInteractionByInteractionIdOK{
		RequestedScopes: in.Scopes,
	}, nil
}

func (s *OIDCProviderService) PostOauth2BackchannelLogout(ctx context.Context, req *oas.PostOauth2BackchannelLogoutReq) error {
	panic("implement me")
}

func (s *OIDCProviderService) PostOauth2DeviceAuthorization(ctx context.Context, req *oas.PostOauth2DeviceAuthorizationReq) (r *oas.PostOauth2DeviceAuthorizationOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) PostOauth2Introspect(ctx context.Context, req *oas.PostOauth2IntrospectReq) (r *oas.PostOauth2IntrospectOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) PostOauth2Par(ctx context.Context, req *oas.PushedAuthorizationRequest) (r *oas.PostOauth2ParCreated, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) PostOauth2Revoke(ctx context.Context, req *oas.PostOauth2RevokeReq) error {
	panic("implement me")
}

func (s *OIDCProviderService) PostOauth2Token(ctx context.Context, req *oas.PostOauth2TokenReq) (r oas.PostOauth2TokenOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) PostV1DeviceApprove(ctx context.Context, req *oas.PostV1DeviceApproveReq) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) PostV1DeviceDeny(ctx context.Context, req *oas.PostV1DeviceDenyReq) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) PostV1OauthInteractionByInteractionIdConsent(ctx context.Context, req *oas.PostV1OauthInteractionByInteractionIdConsentReq, params oas.PostV1OauthInteractionByInteractionIdConsentParams) (r *oas.PostV1OauthInteractionByInteractionIdConsentOK, _ error) {
	panic("implement me")
}

func (s *OIDCProviderService) PostV1OauthInteractionByInteractionIdLogin(ctx context.Context, req oas.OptPostV1OauthInteractionByInteractionIdLoginReq, params oas.PostV1OauthInteractionByInteractionIdLoginParams) (*oas.PostV1OauthInteractionByInteractionIdLoginOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Grants.CompleteLogin(ctx, params.InteractionID, p.AccountID); err != nil {
		return nil, err
	}
	return &oas.PostV1OauthInteractionByInteractionIdLoginOK{}, nil
}

func (s *OIDCProviderService) PostV1OauthInteractionByInteractionIdReject(ctx context.Context, req oas.OptPostV1OauthInteractionByInteractionIdRejectReq, params oas.PostV1OauthInteractionByInteractionIdRejectParams) (r *oas.PostV1OauthInteractionByInteractionIdRejectOK, _ error) {
	panic("implement me")
}

// oasOAuthGrant maps a domain Grant to its oas representation.
func oasOAuthGrant(g domain.Grant) oas.OAuthGrant {
	return oas.OAuthGrant{
		ID: oas.NewOptString(g.ID),
		Client: oas.NewOptOAuthGrantClient(oas.OAuthGrantClient{
			ID: oas.NewOptString(g.ClientID),
		}),
		Scopes:    g.Scopes,
		GrantedAt: oas.NewOptTimestamp(oas.Timestamp(g.GrantedAt)),
	}
}
