// Code scaffolded for IAM handler groups.
//
// PlatformService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

// PlatformConfig serves unauthenticated bootstrap config for a client.
type PlatformConfig interface {
	PublicConfig(ctx context.Context, projectID, clientID string) (*domain.PublicConfig, error)
}

// PlatformDeps are the ports the Platform service orchestrates.
type PlatformDeps struct{ Config PlatformConfig }

// PlatformService implements the PlatformHandler slice of oas.Handler.
type PlatformService struct {
	oas.UnimplementedHandler
	deps PlatformDeps
}

// NewPlatformService builds the Platform service from its dependencies.
func NewPlatformService(deps PlatformDeps) *PlatformService { return &PlatformService{deps: deps} }

var _ oas.Handler = (*PlatformService)(nil)

func (s *PlatformService) GetV1ConfigPublic(ctx context.Context, params oas.GetV1ConfigPublicParams) (r *oas.PublicConfig, _ error) {
	panic("implement me")
}

func (s *PlatformService) GetV1Csrf(ctx context.Context, params oas.GetV1CsrfParams) (r *oas.GetV1CsrfOK, _ error) {
	panic("implement me")
}

func (s *PlatformService) GetV1Health(ctx context.Context) (r *oas.GetV1HealthOK, _ error) {
	panic("implement me")
}

func (s *PlatformService) GetV1HealthLive(ctx context.Context) (r *oas.GetV1HealthLiveOK, _ error) {
	panic("implement me")
}

func (s *PlatformService) GetV1HealthReady(ctx context.Context) (r oas.GetV1HealthReadyRes, _ error) {
	panic("implement me")
}
