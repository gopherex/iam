// Code scaffolded for IAM handler groups. Each XxxService embeds
// oas.UnimplementedHandler (so non-1.0.0 / unwritten ops auto-return
// not-implemented) and panics on every v1.0.0 op until implemented.

package api

import (
	"context"

	"github.com/gopherex/iam/internal/oas"
)

// PlatformService implements the PlatformHandler slice of oas.Handler.
type PlatformService struct{ oas.UnimplementedHandler }

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
