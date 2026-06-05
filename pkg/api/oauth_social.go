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

type oauthSocialAccounts interface {
	EnabledProviders(ctx context.Context, projectID string) ([]domain.OAuthProvider, error)
	CompleteLogin(ctx context.Context, projectID, provider, code string) (*domain.Account, *domain.Session, error)
	Link(ctx context.Context, accountID, provider, code string) error
	Unlink(ctx context.Context, accountID, identityID string) error
}

type OAuthSocialDeps struct{ Accounts oauthSocialAccounts }

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
