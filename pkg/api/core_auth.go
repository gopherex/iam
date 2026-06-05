// Code scaffolded for IAM handler groups. Each XxxService embeds
// oas.UnimplementedHandler (so non-1.0.0 / unwritten ops auto-return
// not-implemented) and panics on every v1.0.0 op until implemented.

package api

import (
	"context"

	"github.com/gopherex/iam/internal/oas"
)

// CoreAuthService implements the CoreAuthHandler slice of oas.Handler.
type CoreAuthService struct{ oas.UnimplementedHandler }

var _ oas.Handler = (*CoreAuthService)(nil)

func (s *CoreAuthService) GetV1AuthEmailChangeCancel(ctx context.Context, params oas.GetV1AuthEmailChangeCancelParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) GetV1AuthEmailVerificationCallback(ctx context.Context, params oas.GetV1AuthEmailVerificationCallbackParams) (r *oas.GetV1AuthEmailVerificationCallbackFound, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) GetV1AuthSession(ctx context.Context) (r oas.GetV1AuthSessionRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) GetV1TokensCurrent(ctx context.Context) (r oas.GetV1TokensCurrentRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthAccessRequests(ctx context.Context, req *oas.PostV1AuthAccessRequestsReq, params oas.PostV1AuthAccessRequestsParams) (r *oas.PostV1AuthAccessRequestsOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailChangeStart(ctx context.Context, req *oas.PostV1AuthEmailChangeStartReq) (r oas.PostV1AuthEmailChangeStartRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailChangeVerify(ctx context.Context, req *oas.PostV1AuthEmailChangeVerifyReq) (r oas.PostV1AuthEmailChangeVerifyRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailVerificationStart(ctx context.Context, req *oas.PostV1AuthEmailVerificationStartReq, params oas.PostV1AuthEmailVerificationStartParams) (r oas.PostV1AuthEmailVerificationStartRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailVerificationVerify(ctx context.Context, req *oas.PostV1AuthEmailVerificationVerifyReq, params oas.PostV1AuthEmailVerificationVerifyParams) (r oas.PostV1AuthEmailVerificationVerifyRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthGuest(ctx context.Context, req *oas.PostV1AuthGuestReq, params oas.PostV1AuthGuestParams) (r oas.PostV1AuthGuestRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordChange(ctx context.Context, req *oas.PasswordChangeRequest) (r oas.PostV1AuthPasswordChangeRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordCheck(ctx context.Context, req *oas.PostV1AuthPasswordCheckReq, params oas.PostV1AuthPasswordCheckParams) (r *oas.PostV1AuthPasswordCheckOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordForgot(ctx context.Context, req *oas.PasswordForgotRequest, params oas.PostV1AuthPasswordForgotParams) (r oas.PostV1AuthPasswordForgotRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordReset(ctx context.Context, req *oas.PasswordResetRequest, params oas.PostV1AuthPasswordResetParams) (r oas.PostV1AuthPasswordResetRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordVerify(ctx context.Context, req *oas.PostV1AuthPasswordVerifyReq) (r oas.PostV1AuthPasswordVerifyRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneChangeStart(ctx context.Context, req *oas.PostV1AuthPhoneChangeStartReq) (r oas.PostV1AuthPhoneChangeStartRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneChangeVerify(ctx context.Context, req *oas.PostV1AuthPhoneChangeVerifyReq) (r oas.PostV1AuthPhoneChangeVerifyRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneVerificationStart(ctx context.Context, req *oas.PostV1AuthPhoneVerificationStartReq, params oas.PostV1AuthPhoneVerificationStartParams) (r oas.PostV1AuthPhoneVerificationStartRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneVerificationVerify(ctx context.Context, req *oas.PostV1AuthPhoneVerificationVerifyReq, params oas.PostV1AuthPhoneVerificationVerifyParams) (r oas.PostV1AuthPhoneVerificationVerifyRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSessionStepUp(ctx context.Context, req *oas.PostV1AuthSessionStepUpReq) (r oas.PostV1AuthSessionStepUpRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSessionSwitchGroup(ctx context.Context, req *oas.PostV1AuthSessionSwitchGroupReq) (r oas.PostV1AuthSessionSwitchGroupRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSignInPassword(ctx context.Context, req *oas.PasswordSignInRequest, params oas.PostV1AuthSignInPasswordParams) (r oas.PostV1AuthSignInPasswordRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSignOut(ctx context.Context, req oas.OptPostV1AuthSignOutReq) (r oas.PostV1AuthSignOutRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSignOutAll(ctx context.Context, req oas.OptPostV1AuthSignOutAllReq) (r oas.PostV1AuthSignOutAllRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSignUp(ctx context.Context, req *oas.SignUpRequest, params oas.PostV1AuthSignUpParams) (r oas.PostV1AuthSignUpRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthTokenExchange(ctx context.Context, req *oas.CodeExchangeRequest, params oas.PostV1AuthTokenExchangeParams) (r oas.PostV1AuthTokenExchangeRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthTokenRefresh(ctx context.Context, req oas.OptRefreshRequest, params oas.PostV1AuthTokenRefreshParams) (r oas.PostV1AuthTokenRefreshRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1ChallengesCaptchaVerify(ctx context.Context, req *oas.PostV1ChallengesCaptchaVerifyReq) (r *oas.PostV1ChallengesCaptchaVerifyOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1TokensIntrospect(ctx context.Context, req *oas.PostV1TokensIntrospectReq) (r oas.PostV1TokensIntrospectRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1TokensRevoke(ctx context.Context, req *oas.PostV1TokensRevokeReq) (r oas.PostV1TokensRevokeRes, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1TokensVerify(ctx context.Context, req *oas.PostV1TokensVerifyReq) (r oas.PostV1TokensVerifyRes, _ error) {
	panic("implement me")
}
