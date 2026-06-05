// Code scaffolded for IAM handler groups. Each XxxService embeds
// oas.UnimplementedHandler (so non-1.0.0 / unwritten ops auto-return
// not-implemented) and panics on every v1.0.0 op until implemented.

package api

import (
	"context"

	"github.com/gopherex/iam/internal/oas"
)

// AccountService implements the AccountHandler slice of oas.Handler.
type AccountService struct{ oas.UnimplementedHandler }

var _ oas.Handler = (*AccountService)(nil)

func (s *AccountService) DeleteV1AuthIdentitiesByIdentityId(ctx context.Context, params oas.DeleteV1AuthIdentitiesByIdentityIdParams) (r oas.DeleteV1AuthIdentitiesByIdentityIdRes, _ error) {
	panic("implement me")
}

func (s *AccountService) DeleteV1Sessions(ctx context.Context, req oas.OptDeleteV1SessionsReq) (r oas.DeleteV1SessionsRes, _ error) {
	panic("implement me")
}

func (s *AccountService) DeleteV1SessionsBySessionId(ctx context.Context, params oas.DeleteV1SessionsBySessionIdParams) (r oas.DeleteV1SessionsBySessionIdRes, _ error) {
	panic("implement me")
}

func (s *AccountService) DeleteV1UsersMe(ctx context.Context, req oas.OptDeleteV1UsersMeReq) (r oas.DeleteV1UsersMeRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1AccountCapabilities(ctx context.Context) (r oas.GetV1AccountCapabilitiesRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1AuthIdentities(ctx context.Context) (r oas.GetV1AuthIdentitiesRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1Sessions(ctx context.Context) (r oas.GetV1SessionsRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1SessionsCurrent(ctx context.Context) (r oas.GetV1SessionsCurrentRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMe(ctx context.Context) (r oas.GetV1UsersMeRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMeActivity(ctx context.Context, params oas.GetV1UsersMeActivityParams) (r oas.GetV1UsersMeActivityRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMeConsents(ctx context.Context) (r oas.GetV1UsersMeConsentsRes, _ error) {
	panic("implement me")
}

func (s *AccountService) GetV1UsersMeExportByJobId(ctx context.Context, params oas.GetV1UsersMeExportByJobIdParams) (r oas.GetV1UsersMeExportByJobIdRes, _ error) {
	panic("implement me")
}

func (s *AccountService) PatchV1SessionsBySessionId(ctx context.Context, req *oas.PatchV1SessionsBySessionIdReq, params oas.PatchV1SessionsBySessionIdParams) (r oas.PatchV1SessionsBySessionIdRes, _ error) {
	panic("implement me")
}

func (s *AccountService) PatchV1UsersMe(ctx context.Context, req *oas.PatchV1UsersMeReq) (r oas.PatchV1UsersMeRes, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1AuthIdentitiesMergeConfirm(ctx context.Context, req *oas.PostV1AuthIdentitiesMergeConfirmReq) (r oas.PostV1AuthIdentitiesMergeConfirmRes, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1AuthIdentitiesMergeStart(ctx context.Context, req *oas.PostV1AuthIdentitiesMergeStartReq) (r oas.PostV1AuthIdentitiesMergeStartRes, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1SessionsBySessionIdTrust(ctx context.Context, req *oas.PostV1SessionsBySessionIdTrustReq, params oas.PostV1SessionsBySessionIdTrustParams) (r oas.PostV1SessionsBySessionIdTrustRes, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1UsersMeConsents(ctx context.Context, req *oas.PostV1UsersMeConsentsReq) (r oas.PostV1UsersMeConsentsRes, _ error) {
	panic("implement me")
}

func (s *AccountService) PostV1UsersMeExport(ctx context.Context) (r oas.PostV1UsersMeExportRes, _ error) {
	panic("implement me")
}
