// Code scaffolded for IAM handler groups.
//
// MachineIdentityService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type MachineIdentities interface {
	CreateServiceAccount(ctx context.Context, cmd domain.ServiceAccountCmd) (*domain.ServiceAccount, error)
	MintToken(ctx context.Context, projectID, serviceAccountID string) (string, error)
	CreateAPIKey(ctx context.Context, cmd domain.APIKeyCmd) (*domain.APIKey, string, error)
	RevokeAPIKey(ctx context.Context, projectID, keyID string) error
}

type MachineIdentityDeps struct{ Keys MachineIdentities }

// MachineIdentityService implements the MachineIdentityHandler slice of oas.Handler.
type MachineIdentityService struct {
	oas.UnimplementedHandler
	deps MachineIdentityDeps
}

// NewMachineIdentityService builds the MachineIdentity service from its dependencies.
func NewMachineIdentityService(deps MachineIdentityDeps) *MachineIdentityService {
	return &MachineIdentityService{deps: deps}
}

var _ oas.Handler = (*MachineIdentityService)(nil)
