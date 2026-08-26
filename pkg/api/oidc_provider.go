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
	"html/template"
	"net/url"
	"strings"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type OIDCGrants interface {
	// ResolveInteraction returns the context the hosted login/consent UI renders:
	// the requesting application, the tenant, and the project's locales.
	ResolveInteraction(ctx context.Context, interactionID string) (*domain.OIDCInteractionContext, error)
	// CompleteLogin binds the interaction to the caller. sessionID lets the
	// adapter verify the interaction belongs to this session (anti-hijack)
	// before completing.
	CompleteLogin(ctx context.Context, interactionID, accountID, sessionID string) error
	// Consent records the resource-owner's consent decision and returns the
	// redirect target the user-agent should follow next.
	Consent(ctx context.Context, cmd domain.OIDCConsentCmd) (*domain.OIDCAuthorizeResult, error)
	// Reject cancels the interaction and returns the redirect target carrying
	// the OAuth2 error back to the client. It is a public operation.
	Reject(ctx context.Context, cmd domain.OIDCRejectCmd) (*domain.OIDCAuthorizeResult, error)
	ListGrants(ctx context.Context, accountID string) ([]domain.Grant, error)
	RevokeGrant(ctx context.Context, accountID, grantID string) error

	// Authorize handles the front-channel authorization request and returns the
	// redirect URL the user-agent must follow next. Public operation.
	Authorize(ctx context.Context, cmd domain.OIDCAuthorizeCmd) (*domain.OIDCAuthorizeResult, error)
	// Logout ends the session named by the id_token_hint and returns where to
	// send the browser, plus whether to clear its session cookies. Public.
	Logout(ctx context.Context, cmd domain.OIDCLogoutCmd) (*domain.OIDCLogoutResult, error)
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

	// RegisterClient / ReadClient / UpdateClient / DeleteClient are dynamic
	// client registration (RFC 7591) and the management it implies (RFC 7592).
	RegisterClient(ctx context.Context, cmd domain.OIDCClientRegisterCmd) (*domain.OIDCClientRegistration, error)
	ReadClient(ctx context.Context, clientID string) (*domain.OIDCClientRegistration, error)
	UpdateClient(ctx context.Context, cmd domain.OIDCClientUpdateCmd) (*domain.OIDCClientRegistration, error)
	DeleteClient(ctx context.Context, clientID string) error
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

func (s *OIDCProviderService) GetOauth2Authorize(
	ctx context.Context, params oas.GetOauth2AuthorizeParams,
) (oas.GetOauth2AuthorizeRes, error) {
	// Public front-channel operation: the client is identified by client_id.
	res, err := s.deps.Grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:            params.ClientID,
		ResponseType:        params.ResponseType.Or(""),
		RedirectURI:         params.RedirectURI.Or(""),
		Scope:               params.Scope.Or(""),
		State:               params.State.Or(""),
		CodeChallenge:       params.CodeChallenge.Or(""),
		CodeChallengeMethod: params.CodeChallengeMethod.Or(""),
		Nonce:               params.Nonce.Or(""),
		Prompt:              params.Prompt.Or(""),
		MaxAge:              params.MaxAge.Or(0),
		ResponseMode:        params.ResponseMode.Or(""),
		RequestURI:          params.RequestURI.Or(""),
		Request:             params.Request.Or(""),
	})
	if err != nil {
		return nil, err
	}

	// form_post answers with a document, not a redirect: the response parameters
	// go in a POST body, so they never reach the address bar or the Referer of
	// whatever the client renders next.
	if res.FormPost != nil {
		return &oas.GetOauth2AuthorizeOK{Data: strings.NewReader(renderFormPost(res.FormPost))}, nil
	}

	return &oas.GetOauth2AuthorizeFound{Location: optURI(res.RedirectTo)}, nil
}

func (s *OIDCProviderService) GetOauth2Logout(ctx context.Context, params oas.GetOauth2LogoutParams) (r *oas.GetOauth2LogoutFound, _ error) {
	// Public RP-initiated logout.
	res, err := s.deps.Grants.Logout(ctx, domain.OIDCLogoutCmd{
		IDTokenHint:           params.IDTokenHint.Or(""),
		PostLogoutRedirectURI: params.PostLogoutRedirectURI.Or(""),
		State:                 params.State.Or(""),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.GetOauth2LogoutFound{Location: optURI(res.RedirectURL)}
	if res.ClearSessionCookies {
		out.SetCookie = ClearSessionCookies()
	}

	return out, nil
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

func (s *OIDCProviderService) GetV1OauthGrants(ctx context.Context, _ oas.GetV1OauthGrantsParams) (*oas.GetV1OauthGrantsOK, error) {
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

	out := &oas.GetV1OauthInteractionByInteractionIdOK{
		RequestedScopes: in.Interaction.Scopes,
		ProjectID:       oas.NewOptString(in.ProjectID),
		Environment:     oas.NewOptString(in.Environment),
		Client: oas.NewOptGetV1OauthInteractionByInteractionIdOKClient(
			oas.GetV1OauthInteractionByInteractionIdOKClient{
				ID:   oas.NewOptString(in.Interaction.ClientID),
				Name: oas.NewOptString(in.ClientName),
				Type: oas.NewOptString(in.ClientType),
			}),
		SupportedLocales: in.SupportedLocales,
	}
	out.Stage = oas.NewOptGetV1OauthInteractionByInteractionIdOKStage(
		oas.GetV1OauthInteractionByInteractionIdOKStage(in.Stage))

	if in.DefaultLocale != "" {
		out.DefaultLocale = oas.NewOptString(in.DefaultLocale)
	}

	if !in.ExpiresAt.IsZero() {
		out.ExpiresAt = oas.NewOptTimestamp(oas.Timestamp(in.ExpiresAt))
	}

	return out, nil
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
	// Client-authenticated pushed authorization request (RFC 9126). A client may
	// only push a request for itself: the client_id in the body must be the one
	// that authenticated, or a client could lodge a request under another's
	// identity and have it honored at the authorization endpoint.
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	if p.ClientID != "" && req.ClientID != p.ClientID {
		return nil, domain.ErrInvalidClient.WithMessage("client_id does not match the authenticated client")
	}

	parRedirect := req.RedirectURI.Or(url.URL{})

	res, perr := s.deps.Grants.PushAuthorizationRequest(ctx, domain.OIDCParCmd{
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
	})
	if perr != nil {
		return nil, perr
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
	// The token endpoint accepts four kinds of client: client_secret_basic (the
	// transport authenticated it), client_secret_post and private_key_jwt (the
	// body does), and a public client with PKCE (none of them). Transport
	// authentication is therefore optional here, and the adapter decides — the
	// tenant comes from whichever identity is actually proven.
	var (
		projectID             string
		environment           string
		authenticatedClientID string
	)

	if p, ok := PrincipalFrom(ctx); ok && p != nil {
		projectID, environment = p.ProjectID, oidcEnv(p)
		// A service account authenticating with its client secret is a
		// client-credentials client; the transport proved it either way.
		if p.Kind == domain.PrincipalClient || p.Kind == domain.PrincipalService {
			authenticatedClientID = p.ClientID
		}
	}

	res, err := s.deps.Grants.Token(ctx, domain.OIDCTokenCmd{
		ProjectID:             projectID,
		Env:                   environment,
		AuthenticatedClientID: authenticatedClientID,
		ClientAssertionType:   req.ClientAssertionType.Or(""),
		ClientAssertion:       req.ClientAssertion.Or(""),
		GrantType:             req.GrantType.Or(""),
		Code:                  req.Code.Or(""),
		RedirectURI:           req.RedirectURI.Or(""),
		CodeVerifier:          req.CodeVerifier.Or(""),
		RefreshToken:          req.RefreshToken.Or(""),
		ClientID:              req.ClientID.Or(""),
		ClientSecret:          req.ClientSecret.Or(""),
		Scope:                 req.Scope.Or(""),
		DeviceCode:            req.DeviceCode.Or(""),
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

	res, err := s.deps.Grants.Consent(ctx, domain.OIDCConsentCmd{
		InteractionID: params.InteractionID,
		AccountID:     p.AccountID,
		SessionID:     p.SessionID,
		GrantedScopes: req.GrantedScopes,
		Remember:      req.Remember.Or(false),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.PostV1OauthInteractionByInteractionIdConsentOK{
		RedirectTo: oas.NewOptString(res.RedirectTo),
	}
	if res.FormPost != nil {
		out.FormPost = oas.NewOptFormPostResponse(oasFormPost(res.FormPost))
	}

	return out, nil
}

func (s *OIDCProviderService) PostV1OauthInteractionByInteractionIdLogin(ctx context.Context, _ oas.OptPostV1OauthInteractionByInteractionIdLoginReq, params oas.PostV1OauthInteractionByInteractionIdLoginParams) (*oas.PostV1OauthInteractionByInteractionIdLoginOK, error) {
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

	res, err := s.deps.Grants.Reject(ctx, cmd)
	if err != nil {
		return nil, err
	}

	out := &oas.PostV1OauthInteractionByInteractionIdRejectOK{
		RedirectTo: oas.NewOptString(res.RedirectTo),
	}
	if res.FormPost != nil {
		out.FormPost = oas.NewOptFormPostResponse(oasFormPost(res.FormPost))
	}

	return out, nil
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
func oasOAuthGrant(grant domain.Grant) oas.OAuthGrant {
	return oas.OAuthGrant{
		ID: oas.NewOptString(grant.ID),
		Client: oas.NewOptOAuthGrantClient(oas.OAuthGrantClient{
			ID: oas.NewOptString(grant.ClientID),
		}),
		Scopes:    grant.Scopes,
		GrantedAt: oas.NewOptTimestamp(oas.Timestamp(grant.GrantedAt)),
	}
}

// ----- dynamic client registration (RFC 7591 / 7592) -----

// oidcRegistrationFrom reads RFC 7591 metadata off the wire.
func oidcRegistrationFrom(req *oas.ClientRegistration) domain.OIDCClientRegistration {
	return domain.OIDCClientRegistration{
		ClientName:              req.ClientName.Or(""),
		RedirectURIs:            req.RedirectUris,
		ApplicationType:         string(req.ApplicationType.Or("")),
		TokenEndpointAuthMethod: string(req.TokenEndpointAuthMethod.Or("")),
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		Scope:                   splitSpaceDelimited(req.Scope.Or("")),
		JWKS:                    req.Jwks.Or(""),
		JWKSURI:                 req.JwksURI.Or(""),
		PostLogoutRedirectURIs:  req.PostLogoutRedirectUris,
		BackchannelLogoutURI:    req.BackchannelLogoutURI.Or(""),
	}
}

// oasClientRegistration renders registration metadata back onto the wire. The
// secret and the registration access token are present only right after
// registration — afterwards only their digests exist.
func oasClientRegistration(reg *domain.OIDCClientRegistration) *oas.ClientRegistrationResponse {
	out := &oas.ClientRegistrationResponse{
		ClientName:             oas.NewOptString(reg.ClientName),
		RedirectUris:           reg.RedirectURIs,
		GrantTypes:             reg.GrantTypes,
		ResponseTypes:          reg.ResponseTypes,
		PostLogoutRedirectUris: reg.PostLogoutRedirectURIs,
		ClientID:               oas.NewOptString(reg.ClientID),
		ClientIDIssuedAt:       oas.NewOptInt(int(reg.IssuedAt.Unix())),
	}

	if reg.ApplicationType != "" {
		out.ApplicationType = oas.NewOptClientRegistrationResponseApplicationType(
			oas.ClientRegistrationResponseApplicationType(reg.ApplicationType))
	}

	if reg.TokenEndpointAuthMethod != "" {
		out.TokenEndpointAuthMethod = oas.NewOptClientRegistrationResponseTokenEndpointAuthMethod(
			oas.ClientRegistrationResponseTokenEndpointAuthMethod(reg.TokenEndpointAuthMethod))
	}

	if len(reg.Scope) > 0 {
		out.Scope = oas.NewOptString(strings.Join(reg.Scope, " "))
	}

	if reg.JWKS != "" {
		out.Jwks = oas.NewOptNilString(reg.JWKS)
	}

	if reg.JWKSURI != "" {
		out.JwksURI = oas.NewOptNilString(reg.JWKSURI)
	}

	if reg.BackchannelLogoutURI != "" {
		out.BackchannelLogoutURI = oas.NewOptNilString(reg.BackchannelLogoutURI)
	}

	if reg.ClientSecret != "" {
		out.ClientSecret = oas.NewOptNilString(reg.ClientSecret)
	}

	if reg.RegistrationAccessToken != "" {
		out.RegistrationAccessToken = oas.NewOptNilString(reg.RegistrationAccessToken)
	}

	if reg.RegistrationClientURI != "" {
		out.RegistrationClientURI = oas.NewOptNilString(reg.RegistrationClientURI)
	}

	return out
}

// splitSpaceDelimited splits an OAuth space-delimited list.
func splitSpaceDelimited(v string) []string {
	return strings.Fields(v)
}

func (s *OIDCProviderService) PostOauth2Register(
	ctx context.Context, req *oas.ClientRegistration, params oas.PostOauth2RegisterParams,
) (*oas.ClientRegistrationResponse, error) {
	// The initial access token is a project-admin token: it is what says which
	// project the new client belongs to. Registration is never open here.
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	reg, err := s.deps.Grants.RegisterClient(ctx, domain.OIDCClientRegisterCmd{
		ProjectID:    p.ProjectID,
		Environment:  params.XEnvironment.Or(""),
		Registration: oidcRegistrationFrom(req),
	})
	if err != nil {
		return nil, err
	}

	return oasClientRegistration(reg), nil
}

func (s *OIDCProviderService) GetOauth2RegisterByClientId(
	ctx context.Context, params oas.GetOauth2RegisterByClientIdParams,
) (*oas.ClientRegistrationResponse, error) {
	if err := requireRegistrationOwner(ctx, params.ClientID); err != nil {
		return nil, err
	}

	reg, err := s.deps.Grants.ReadClient(ctx, params.ClientID)
	if err != nil {
		return nil, err
	}

	return oasClientRegistration(reg), nil
}

func (s *OIDCProviderService) PutOauth2RegisterByClientId(
	ctx context.Context, req *oas.ClientRegistration, params oas.PutOauth2RegisterByClientIdParams,
) (*oas.ClientRegistrationResponse, error) {
	if err := requireRegistrationOwner(ctx, params.ClientID); err != nil {
		return nil, err
	}

	p, _ := PrincipalFrom(ctx)

	reg, err := s.deps.Grants.UpdateClient(ctx, domain.OIDCClientUpdateCmd{
		ClientID:     params.ClientID,
		ProjectID:    p.ProjectID,
		Registration: oidcRegistrationFrom(req),
	})
	if err != nil {
		return nil, err
	}

	return oasClientRegistration(reg), nil
}

func (s *OIDCProviderService) DeleteOauth2RegisterByClientId(
	ctx context.Context, params oas.DeleteOauth2RegisterByClientIdParams,
) (*oas.Ok, error) {
	if err := requireRegistrationOwner(ctx, params.ClientID); err != nil {
		return nil, err
	}

	if err := s.deps.Grants.DeleteClient(ctx, params.ClientID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

// requireRegistrationOwner checks that the registration access token presented
// belongs to the client being managed. The token authorizes exactly one client;
// without this check any registered client could manage any other.
func requireRegistrationOwner(ctx context.Context, clientID string) error {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}

	if p.ClientID == "" || p.ClientID != clientID {
		return domain.ErrClientNotFound
	}

	return nil
}

// ----- form_post response mode -----

// oasFormPost renders a form-post response onto the wire.
func oasFormPost(post *domain.OIDCFormPost) oas.FormPostResponse {
	fields := make(oas.FormPostResponseFields, len(post.Fields))
	for name, value := range post.Fields {
		fields[name] = value
	}

	return oas.FormPostResponse{Action: post.Action, Fields: fields}
}

// formPostTemplate is the page a form_post authorization response is delivered
// as. It submits itself; the noscript button is the fallback for the small
// number of user-agents that would otherwise strand the user on a blank page.
const formPostTemplate = `<!DOCTYPE html>
<html>
<head><title>Submitting…</title></head>
<body onload="document.forms[0].submit()">
<form method="post" action="{{.Action}}">
{{range $name, $value := .Fields}}<input type="hidden" name="{{$name}}" value="{{$value}}"/>
{{end}}<noscript><button type="submit">Continue</button></noscript>
</form>
</body>
</html>
`

// renderFormPost builds the self-submitting document. Every value is escaped by
// html/template: the fields carry a state the client chose, and a client that
// put markup in its own state must not end up executing it here.
func renderFormPost(post *domain.OIDCFormPost) string {
	doc, err := template.New("form_post").Parse(formPostTemplate)
	if err != nil {
		return ""
	}

	var out strings.Builder
	if err := doc.Execute(&out, post); err != nil {
		// Executing a fixed template over strings cannot fail in practice; if it
		// somehow does, an empty document is safer than a partial one.
		return ""
	}

	return out.String()
}
