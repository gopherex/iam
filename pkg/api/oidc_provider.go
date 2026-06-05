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
	"net/url"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type OIDCGrants interface {
	ResolveInteraction(ctx context.Context, interactionID string) (*domain.Interaction, error)
	// CompleteLogin binds the interaction to the caller. sessionID lets the
	// adapter verify the interaction belongs to this session (anti-hijack)
	// before completing.
	CompleteLogin(ctx context.Context, interactionID, accountID, sessionID string) error
	// Consent records the resource-owner's consent decision and returns the
	// redirect target the user-agent should follow next.
	Consent(ctx context.Context, cmd domain.OIDCConsentCmd) (string, error)
	// Reject cancels the interaction and returns the redirect target carrying
	// the OAuth2 error back to the client. It is a public operation.
	Reject(ctx context.Context, cmd domain.OIDCRejectCmd) (string, error)
	ListGrants(ctx context.Context, accountID string) ([]domain.Grant, error)
	RevokeGrant(ctx context.Context, accountID, grantID string) error

	// Authorize handles the front-channel authorization request and returns the
	// redirect URL the user-agent must follow next. Public operation.
	Authorize(ctx context.Context, cmd domain.OIDCAuthorizeCmd) (string, error)
	// Logout terminates the RP-initiated logout and returns the post-logout
	// redirect URL. Public operation.
	Logout(ctx context.Context, cmd domain.OIDCLogoutCmd) (string, error)
	// BackchannelLogout validates the logout token and terminates the referenced
	// sessions. Public operation.
	BackchannelLogout(ctx context.Context, cmd domain.OIDCBackchannelLogoutCmd) error

	// Token dispatches an /oauth2/token request and returns the raw token
	// response map. Client-authenticated.
	Token(ctx context.Context, cmd domain.OIDCTokenCmd) (map[string]any, error)
	// Introspect returns the introspection response map. Client-authenticated.
	Introspect(ctx context.Context, cmd domain.OIDCIntrospectCmd) (map[string]any, error)
	// Revoke revokes a token. Client-authenticated.
	Revoke(ctx context.Context, cmd domain.OIDCRevokeCmd) error
	// PushAuthorizationRequest stores a PAR and returns its request_uri.
	// Client-authenticated.
	PushAuthorizationRequest(ctx context.Context, cmd domain.OIDCParCmd) (*domain.OIDCParResult, error)
	// DeviceAuthorization starts a device authorization grant (RFC 8628).
	// Client-authenticated.
	DeviceAuthorization(ctx context.Context, cmd domain.OIDCDeviceAuthorizationCmd) (*domain.OIDCDeviceAuthorization, error)

	// Userinfo returns the OIDC userinfo claims for the bearer-authenticated
	// account. accountID/sessionID come from the principal.
	Userinfo(ctx context.Context, accountID, sessionID string) (map[string]any, error)

	// ResolveDevice returns the pending device authorization for a user-facing
	// code, scoped to the requesting client's project. Public operation.
	ResolveDevice(ctx context.Context, code domain.OIDCDeviceUserCode) (*domain.OIDCDevicePending, error)
	// ApproveDevice approves a pending device authorization on behalf of the
	// authenticated user.
	ApproveDevice(ctx context.Context, cmd domain.OIDCDeviceDecisionCmd) error
	// DenyDevice denies a pending device authorization on behalf of the
	// authenticated user.
	DenyDevice(ctx context.Context, cmd domain.OIDCDeviceDecisionCmd) error

	// JWKS returns the JSON Web Key Set for a project environment. Public.
	JWKS(ctx context.Context, projectID, env string) (map[string]any, error)
	// OpenIDConfiguration returns the discovery document for a project
	// environment. Public.
	OpenIDConfiguration(ctx context.Context, projectID, env string) (map[string]any, error)
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
	// Public front-channel operation: the client is identified by client_id.
	redirectTo, err := s.deps.Grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:      params.ClientID,
		ResponseType:  string(params.ResponseType),
		RedirectURI:   params.RedirectURI,
		Scope:         params.Scope,
		State:         params.State.Or(""),
		CodeChallenge: params.CodeChallenge.Or(""),
		Nonce:         params.Nonce.Or(""),
		Prompt:        params.Prompt.Or(""),
		RequestURI:    params.RequestURI.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.GetOauth2AuthorizeFound{Location: optURI(redirectTo)}, nil
}

func (s *OIDCProviderService) GetOauth2Logout(ctx context.Context, params oas.GetOauth2LogoutParams) (r *oas.GetOauth2LogoutFound, _ error) {
	// Public RP-initiated logout.
	redirectTo, err := s.deps.Grants.Logout(ctx, domain.OIDCLogoutCmd{
		IDTokenHint:           params.IDTokenHint.Or(""),
		PostLogoutRedirectURI: params.PostLogoutRedirectURI.Or(""),
		State:                 params.State.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.GetOauth2LogoutFound{Location: optURI(redirectTo)}, nil
}

func (s *OIDCProviderService) GetOauth2Userinfo(ctx context.Context) (r oas.GetOauth2UserinfoOK, _ error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	claims, err := s.deps.Grants.Userinfo(ctx, p.AccountID, p.SessionID)
	if err != nil {
		return nil, err
	}
	return oasRawMap[oas.GetOauth2UserinfoOK](claims), nil
}

func (s *OIDCProviderService) GetPByProjectIdEByEnvWellKnownJwksJson(ctx context.Context, params oas.GetPByProjectIdEByEnvWellKnownJwksJsonParams) (r oas.GetPByProjectIdEByEnvWellKnownJwksJsonOK, _ error) {
	// Public discovery endpoint scoped by project/env path params.
	jwks, err := s.deps.Grants.JWKS(ctx, params.ProjectID, params.Env)
	if err != nil {
		return nil, err
	}
	return oasRawMap[oas.GetPByProjectIdEByEnvWellKnownJwksJsonOK](jwks), nil
}

func (s *OIDCProviderService) GetPByProjectIdEByEnvWellKnownOpenidConfiguration(ctx context.Context, params oas.GetPByProjectIdEByEnvWellKnownOpenidConfigurationParams) (r oas.GetPByProjectIdEByEnvWellKnownOpenidConfigurationOK, _ error) {
	// Public discovery endpoint scoped by project/env path params.
	cfg, err := s.deps.Grants.OpenIDConfiguration(ctx, params.ProjectID, params.Env)
	if err != nil {
		return nil, err
	}
	return oasRawMap[oas.GetPByProjectIdEByEnvWellKnownOpenidConfigurationOK](cfg), nil
}

func (s *OIDCProviderService) GetV1Device(ctx context.Context, params oas.GetV1DeviceParams) (r *oas.GetV1DeviceOK, _ error) {
	// Public verification UI lookup: project comes from the X-Client-Id header.
	pending, err := s.deps.Grants.ResolveDevice(ctx, domain.OIDCDeviceUserCode{
		ProjectID: params.XClientID,
		UserCode:  params.UserCode,
	})
	if err != nil {
		return nil, err
	}
	return oasOIDCDevicePending(pending), nil
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
	// Public back-channel logout: the logout_token carries the subject/sessions.
	return s.deps.Grants.BackchannelLogout(ctx, domain.OIDCBackchannelLogoutCmd{
		LogoutToken: req.LogoutToken.Or(""),
	})
}

func (s *OIDCProviderService) PostOauth2DeviceAuthorization(ctx context.Context, req *oas.PostOauth2DeviceAuthorizationReq) (r *oas.PostOauth2DeviceAuthorizationOK, _ error) {
	// Client-authenticated device authorization request.
	auth, err := s.deps.Grants.DeviceAuthorization(ctx, domain.OIDCDeviceAuthorizationCmd{
		ClientID: req.ClientID.Or(""),
		Scope:    req.Scope.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostOauth2DeviceAuthorizationOK{
		DeviceCode:              oas.NewOptString(auth.DeviceCode),
		UserCode:                oas.NewOptString(auth.UserCode),
		VerificationURI:         oas.NewOptString(auth.VerificationURI),
		VerificationURIComplete: oas.NewOptString(auth.VerificationURIComplete),
		ExpiresIn:               oas.NewOptInt(auth.ExpiresIn),
		Interval:                oas.NewOptInt(auth.Interval),
	}, nil
}

func (s *OIDCProviderService) PostOauth2Introspect(ctx context.Context, req *oas.PostOauth2IntrospectReq) (r *oas.PostOauth2IntrospectOK, _ error) {
	// Client-authenticated token introspection (RFC 7662). The verifying tenant
	// is the authenticated client's project — never the token's self-asserted
	// issuer (cross-tenant confusion).
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.deps.Grants.Introspect(ctx, domain.OIDCIntrospectCmd{
		ProjectID: p.ProjectID,
		Env:       oidcEnv(p),
		Token:     req.Token.Or(""),
	})
	if err != nil {
		return nil, err
	}
	out := &oas.PostOauth2IntrospectOK{
		AdditionalProps: oasRawMap[oas.PostOauth2IntrospectOKAdditional](res),
	}
	if active, ok := res["active"].(bool); ok {
		out.Active = oas.NewOptBool(active)
		delete(out.AdditionalProps, "active")
	}
	return out, nil
}

func (s *OIDCProviderService) PostOauth2Par(ctx context.Context, req *oas.PushedAuthorizationRequest) (r *oas.PostOauth2ParCreated, _ error) {
	// Client-authenticated pushed authorization request (RFC 9126).
	parRedirect := req.RedirectURI.Or(url.URL{})
	res, err := s.deps.Grants.PushAuthorizationRequest(ctx, domain.OIDCParCmd{
		ResponseType:        req.ResponseType,
		ClientID:            req.ClientID,
		RedirectURI:         parRedirect.String(),
		Scope:               req.Scope.Or(""),
		State:               req.State.Or(""),
		CodeChallenge:       req.CodeChallenge.Or(""),
		CodeChallengeMethod: string(req.CodeChallengeMethod.Or("")),
		Nonce:               req.Nonce.Or(""),
		ResponseMode:        req.ResponseMode.Or(""),
		Prompt:              req.Prompt.Or(""),
		LoginHint:           req.LoginHint.Or(""),
		Request:             req.Request.Or(""),
		ClientAssertionType: req.ClientAssertionType.Or(""),
		ClientAssertion:     req.ClientAssertion.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostOauth2ParCreated{
		RequestURI: oas.NewOptString(res.RequestURI),
		ExpiresIn:  oas.NewOptInt(res.ExpiresIn),
	}, nil
}

func (s *OIDCProviderService) PostOauth2Revoke(ctx context.Context, req *oas.PostOauth2RevokeReq) error {
	// Client-authenticated token revocation (RFC 7009).
	p, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	return s.deps.Grants.Revoke(ctx, domain.OIDCRevokeCmd{
		ProjectID:     p.ProjectID,
		Env:           oidcEnv(p),
		Token:         req.Token.Or(""),
		TokenTypeHint: req.TokenTypeHint.Or(""),
	})
}

// oidcEnv resolves the environment for a client/principal, defaulting to live.
func oidcEnv(p *domain.Principal) string {
	if p.Environment != "" {
		return p.Environment
	}
	return "live"
}

func (s *OIDCProviderService) PostOauth2Token(ctx context.Context, req *oas.PostOauth2TokenReq) (r oas.PostOauth2TokenOK, _ error) {
	// Client-authenticated token endpoint (RFC 6749 + extensions). The tenant is
	// the authenticated client's project, not the token's issuer.
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.deps.Grants.Token(ctx, domain.OIDCTokenCmd{
		ProjectID:    p.ProjectID,
		Env:          oidcEnv(p),
		GrantType:    req.GrantType.Or(""),
		Code:         req.Code.Or(""),
		RedirectURI:  req.RedirectURI.Or(""),
		CodeVerifier: req.CodeVerifier.Or(""),
		RefreshToken: req.RefreshToken.Or(""),
		ClientID:     req.ClientID.Or(""),
		ClientSecret: req.ClientSecret.Or(""),
		DeviceCode:   req.DeviceCode.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return oasRawMap[oas.PostOauth2TokenOK](res), nil
}

func (s *OIDCProviderService) PostV1DeviceApprove(ctx context.Context, req *oas.PostV1DeviceApproveReq) (r *oas.Ok, _ error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Grants.ApproveDevice(ctx, domain.OIDCDeviceDecisionCmd{
		ProjectID: p.ProjectID,
		UserCode:  req.UserCode,
		AccountID: p.AccountID,
		SessionID: p.SessionID,
	}); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *OIDCProviderService) PostV1DeviceDeny(ctx context.Context, req *oas.PostV1DeviceDenyReq) (r *oas.Ok, _ error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Grants.DenyDevice(ctx, domain.OIDCDeviceDecisionCmd{
		ProjectID: p.ProjectID,
		UserCode:  req.UserCode,
		AccountID: p.AccountID,
		SessionID: p.SessionID,
	}); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *OIDCProviderService) PostV1OauthInteractionByInteractionIdConsent(ctx context.Context, req *oas.PostV1OauthInteractionByInteractionIdConsentReq, params oas.PostV1OauthInteractionByInteractionIdConsentParams) (*oas.PostV1OauthInteractionByInteractionIdConsentOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	redirectTo, err := s.deps.Grants.Consent(ctx, domain.OIDCConsentCmd{
		InteractionID: params.InteractionID,
		AccountID:     p.AccountID,
		SessionID:     p.SessionID,
		GrantedScopes: req.GrantedScopes,
		Remember:      req.Remember.Or(false),
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1OauthInteractionByInteractionIdConsentOK{
		RedirectTo: oas.NewOptString(redirectTo),
	}, nil
}

func (s *OIDCProviderService) PostV1OauthInteractionByInteractionIdLogin(ctx context.Context, req oas.OptPostV1OauthInteractionByInteractionIdLoginReq, params oas.PostV1OauthInteractionByInteractionIdLoginParams) (*oas.PostV1OauthInteractionByInteractionIdLoginOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Grants.CompleteLogin(ctx, params.InteractionID, p.AccountID, p.SessionID); err != nil {
		return nil, err
	}
	return &oas.PostV1OauthInteractionByInteractionIdLoginOK{}, nil
}

func (s *OIDCProviderService) PostV1OauthInteractionByInteractionIdReject(ctx context.Context, req oas.OptPostV1OauthInteractionByInteractionIdRejectReq, params oas.PostV1OauthInteractionByInteractionIdRejectParams) (*oas.PostV1OauthInteractionByInteractionIdRejectOK, error) {
	cmd := domain.OIDCRejectCmd{InteractionID: params.InteractionID}
	if v, ok := req.Get(); ok {
		cmd.Error = v.Error.Or("")
		cmd.ErrorDescription = v.ErrorDescription.Or("")
	}
	redirectTo, err := s.deps.Grants.Reject(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1OauthInteractionByInteractionIdRejectOK{
		RedirectTo: oas.NewOptString(redirectTo),
	}, nil
}

// oasOIDCDevicePending maps a pending device authorization to its oas
// verification-UI representation. The client descriptor is a freeform map.
func oasOIDCDevicePending(p *domain.OIDCDevicePending) *oas.GetV1DeviceOK {
	out := &oas.GetV1DeviceOK{Scopes: p.Scopes}
	if len(p.ClientMap) > 0 {
		out.Client = oas.NewOptGetV1DeviceOKClient(oasRawMap[oas.GetV1DeviceOKClient](p.ClientMap))
	}
	if !p.ExpiresAt.IsZero() {
		out.ExpiresAt = oas.NewOptTimestamp(oas.Timestamp(p.ExpiresAt))
	}
	return out
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
