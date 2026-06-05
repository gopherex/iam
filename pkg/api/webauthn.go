// Code scaffolded for IAM handler groups.
//
// WebAuthnService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type webAuthnAccounts interface {
	BeginLogin(ctx context.Context, projectID, email string) (*domain.Challenge, error)
	FinishLogin(ctx context.Context, challengeID string, credential map[string]any) (*domain.Account, *domain.Session, error)
	BeginRegistration(ctx context.Context, accountID string) (*domain.Challenge, error)
	FinishRegistration(ctx context.Context, accountID, challengeID string, credential map[string]any) (*domain.WebAuthnCredential, error)
	ListCredentials(ctx context.Context, accountID string) ([]domain.WebAuthnCredential, error)
	RemoveCredential(ctx context.Context, accountID, credentialID string) error
}

type WebAuthnDeps struct{ Accounts webAuthnAccounts }

// WebAuthnService implements the WebAuthnHandler slice of oas.Handler.
type WebAuthnService struct {
	oas.UnimplementedHandler
	deps WebAuthnDeps
}

// NewWebAuthnService builds the WebAuthn service from its dependencies.
func NewWebAuthnService(deps WebAuthnDeps) *WebAuthnService { return &WebAuthnService{deps: deps} }

var _ oas.Handler = (*WebAuthnService)(nil)
