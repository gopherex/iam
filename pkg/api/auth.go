package api

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

// Authenticator validates a credential and resolves the calling principal. The
// adapter implements it (JWT verification, session/token lookup); pkg/api only
// juggles the interface. One method per security scheme.
type Authenticator interface {
	User(ctx context.Context, token string) (*domain.Principal, error)              // bearerAuth
	Admin(ctx context.Context, token string) (*domain.Principal, error)             // adminToken
	Master(ctx context.Context, token string) (*domain.Principal, error)            // masterKey
	Service(ctx context.Context, token string) (*domain.Principal, error)           // serviceToken / API key
	SCIM(ctx context.Context, token string) (*domain.Principal, error)              // scimToken
	Client(ctx context.Context, clientID, secret string) (*domain.Principal, error) // clientSecretBasic
	OAuth2(ctx context.Context, token string) (*domain.Principal, error)            // oauth2
}

// ----- principal in context -----

type ctxKey int

const principalKey ctxKey = iota

func withPrincipal(ctx context.Context, p *domain.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFrom returns the authenticated principal placed in ctx by the
// SecurityHandler, if any.
func PrincipalFrom(ctx context.Context) (*domain.Principal, bool) {
	p, ok := ctx.Value(principalKey).(*domain.Principal)
	return p, ok
}

// requirePrincipal returns the principal or domain.ErrUnauthorized.
func requirePrincipal(ctx context.Context) (*domain.Principal, error) {
	if p, ok := PrincipalFrom(ctx); ok && p != nil {
		return p, nil
	}
	return nil, domain.ErrUnauthorized
}

// ----- ogen SecurityHandler -----

// NewSecurityHandler wires an Authenticator into the ogen SecurityHandler.
// Pass it to oas.NewServer(handler, api.NewSecurityHandler(auth), …).
func NewSecurityHandler(a Authenticator) oas.SecurityHandler {
	return securityHandler{a: a}
}

type securityHandler struct{ a Authenticator }

func (h securityHandler) auth(ctx context.Context, p *domain.Principal, err error) (context.Context, error) {
	if err != nil {
		return ctx, err // ogen wraps as SecurityError -> ErrorHandler -> 401
	}
	return withPrincipal(ctx, p), nil
}

func (h securityHandler) HandleBearerAuth(ctx context.Context, _ oas.OperationName, t oas.BearerAuth) (context.Context, error) {
	p, err := h.a.User(ctx, t.Token)
	return h.auth(ctx, p, err)
}

func (h securityHandler) HandleAdminToken(ctx context.Context, _ oas.OperationName, t oas.AdminToken) (context.Context, error) {
	p, err := h.a.Admin(ctx, t.Token)
	return h.auth(ctx, p, err)
}

func (h securityHandler) HandleMasterKey(ctx context.Context, _ oas.OperationName, t oas.MasterKey) (context.Context, error) {
	p, err := h.a.Master(ctx, t.Token)
	return h.auth(ctx, p, err)
}

func (h securityHandler) HandleServiceToken(ctx context.Context, _ oas.OperationName, t oas.ServiceToken) (context.Context, error) {
	p, err := h.a.Service(ctx, t.Token)
	return h.auth(ctx, p, err)
}

func (h securityHandler) HandleScimToken(ctx context.Context, _ oas.OperationName, t oas.ScimToken) (context.Context, error) {
	p, err := h.a.SCIM(ctx, t.Token)
	return h.auth(ctx, p, err)
}

func (h securityHandler) HandleClientSecretBasic(ctx context.Context, _ oas.OperationName, t oas.ClientSecretBasic) (context.Context, error) {
	p, err := h.a.Client(ctx, t.Username, t.Password)
	return h.auth(ctx, p, err)
}

func (h securityHandler) HandleOAuth2(ctx context.Context, _ oas.OperationName, t oas.OAuth2) (context.Context, error) {
	p, err := h.a.OAuth2(ctx, t.Token)
	return h.auth(ctx, p, err)
}
