// Code scaffolded for IAM handler groups.
//
// MFAService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type mfaAccounts interface {
	ListFactors(ctx context.Context, accountID string) ([]domain.Factor, error)
	EnrollTOTP(ctx context.Context, accountID string) (*domain.Factor, error)
	Challenge(ctx context.Context, accountID, factorID string) (*domain.Challenge, error)
	Verify(ctx context.Context, challengeID, code string) (*domain.Account, *domain.Session, error)
	GenerateRecoveryCodes(ctx context.Context, accountID string) ([]string, error)
	RemoveFactor(ctx context.Context, accountID, factorID string) error
}

type MFADeps struct{ Accounts mfaAccounts }

// MFAService implements the MFAHandler slice of oas.Handler.
type MFAService struct {
	oas.UnimplementedHandler
	deps MFADeps
}

// NewMFAService builds the MFA service from its dependencies.
func NewMFAService(deps MFADeps) *MFAService { return &MFAService{deps: deps} }

var _ oas.Handler = (*MFAService)(nil)
