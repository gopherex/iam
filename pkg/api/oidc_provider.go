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

type oidcGrants interface {
	ResolveInteraction(ctx context.Context, interactionID string) (*domain.Interaction, error)
	CompleteLogin(ctx context.Context, interactionID, accountID string) error
	ListGrants(ctx context.Context, accountID string) ([]domain.Grant, error)
	RevokeGrant(ctx context.Context, accountID, grantID string) error
}

type OIDCProviderDeps struct{ Grants oidcGrants }

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
