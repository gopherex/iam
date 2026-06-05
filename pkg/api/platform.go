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

func (s *PlatformService) GetV1ConfigPublic(ctx context.Context, params oas.GetV1ConfigPublicParams) (*oas.PublicConfig, error) {
	cfg, err := s.deps.Config.PublicConfig(ctx, params.XClientID, params.XClientID)
	if err != nil {
		return nil, err
	}
	return oasPublicConfig(cfg), nil
}

func (s *PlatformService) GetV1Csrf(ctx context.Context, params oas.GetV1CsrfParams) (r *oas.GetV1CsrfOK, _ error) {
	panic("implement me")
}

func (s *PlatformService) GetV1Health(ctx context.Context) (*oas.GetV1HealthOK, error) {
	return &oas.GetV1HealthOK{Status: oas.NewOptString("ok")}, nil
}

func (s *PlatformService) GetV1HealthLive(ctx context.Context) (*oas.GetV1HealthLiveOK, error) {
	return &oas.GetV1HealthLiveOK{Status: oas.NewOptString("ok")}, nil
}

func (s *PlatformService) GetV1HealthReady(ctx context.Context) (*oas.GetV1HealthReadyOK, error) {
	return &oas.GetV1HealthReadyOK{Status: oas.NewOptString("ok")}, nil
}

// oasPublicConfig maps the domain bootstrap config to the oas wire type.
func oasPublicConfig(c *domain.PublicConfig) *oas.PublicConfig {
	r := &oas.PublicConfig{
		Project:       oas.NewOptPublicConfigProject(oas.PublicConfigProject{Name: oas.NewOptString(c.ProjectName)}),
		Methods:       c.Methods,
		Locales:       c.Locales,
		DefaultLocale: oas.NewOptString(c.DefaultLocale),
	}
	for _, p := range c.Providers {
		r.Providers = append(r.Providers, oas.PublicConfigProvidersItem{
			ID:   oas.NewOptString(p.ID),
			Name: oas.NewOptString(p.Name),
		})
	}
	return r
}
