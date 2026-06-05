// Code scaffolded for IAM handler groups.
//
// FederationService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type FederationConnections interface {
	CreateConnection(ctx context.Context, cmd domain.ConnectionCmd) (*domain.Connection, error)
	GetConnection(ctx context.Context, projectID, id string) (*domain.Connection, error)
	ListConnections(ctx context.Context, projectID string) ([]domain.Connection, error)
	AddDomain(ctx context.Context, projectID, connectionID, name string) (*domain.Domain, error)
	VerifyDomain(ctx context.Context, projectID, domainID string) (*domain.Domain, error)
}

type FederationDeps struct{ Connections FederationConnections }

// FederationService implements the FederationHandler slice of oas.Handler.
type FederationService struct {
	oas.UnimplementedHandler
	deps FederationDeps
}

// NewFederationService builds the Federation service from its dependencies.
func NewFederationService(deps FederationDeps) *FederationService {
	return &FederationService{deps: deps}
}

var _ oas.Handler = (*FederationService)(nil)
