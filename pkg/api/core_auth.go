// Code scaffolded for IAM handler groups.
//
// CoreAuthService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

// CoreAuthAccounts is the Core Auth slice of the Account aggregate. Each method
// is one atomic operation; the adapter owns its transaction.
type CoreAuthAccounts interface {
	Register(ctx context.Context, cmd domain.RegisterCmd) (*domain.Account, *domain.Session, error)
	AuthenticatePassword(ctx context.Context, projectID, email, password string) (*domain.Account, *domain.Session, error)
	Refresh(ctx context.Context, refreshToken string) (*domain.Account, *domain.Session, error)
	ExchangeCode(ctx context.Context, code, verifier string) (*domain.Account, *domain.Session, error)
	CreateGuest(ctx context.Context, projectID string) (*domain.Account, *domain.Session, error)
	GetSession(ctx context.Context, sessionID string) (*domain.Account, *domain.Session, error)
	SignOut(ctx context.Context, sessionID string, everywhere bool) error
}

// CoreAuthDeps are the ports the Core Auth service orchestrates.
type CoreAuthDeps struct{ Accounts CoreAuthAccounts }

// CoreAuthService implements the CoreAuthHandler slice of oas.Handler.
type CoreAuthService struct {
	oas.UnimplementedHandler
	deps CoreAuthDeps
}

// NewCoreAuthService builds the CoreAuth service from its dependencies.
func NewCoreAuthService(deps CoreAuthDeps) *CoreAuthService { return &CoreAuthService{deps: deps} }

var _ oas.Handler = (*CoreAuthService)(nil)

func (s *CoreAuthService) GetV1AuthEmailChangeCancel(ctx context.Context, params oas.GetV1AuthEmailChangeCancelParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) GetV1AuthEmailVerificationCallback(ctx context.Context, params oas.GetV1AuthEmailVerificationCallbackParams) (r *oas.GetV1AuthEmailVerificationCallbackFound, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) GetV1AuthSession(ctx context.Context) (*oas.GetV1AuthSessionOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, sess, err := s.deps.Accounts.GetSession(ctx, p.SessionID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1AuthSessionOK{
		User:    oas.NewOptUser(oasUser(acct)),
		Session: oas.NewOptSession(oasSession(sess)),
	}, nil
}

func (s *CoreAuthService) GetV1TokensCurrent(ctx context.Context) (r *oas.GetV1TokensCurrentOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthAccessRequests(ctx context.Context, req *oas.PostV1AuthAccessRequestsReq, params oas.PostV1AuthAccessRequestsParams) (r *oas.PostV1AuthAccessRequestsOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailChangeStart(ctx context.Context, req *oas.PostV1AuthEmailChangeStartReq) (r *oas.Challenge, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailChangeVerify(ctx context.Context, req *oas.PostV1AuthEmailChangeVerifyReq) (r *oas.PostV1AuthEmailChangeVerifyOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailVerificationStart(ctx context.Context, req *oas.PostV1AuthEmailVerificationStartReq, params oas.PostV1AuthEmailVerificationStartParams) (r *oas.Challenge, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthEmailVerificationVerify(ctx context.Context, req *oas.PostV1AuthEmailVerificationVerifyReq, params oas.PostV1AuthEmailVerificationVerifyParams) (r *oas.AuthResult, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthGuest(ctx context.Context, req *oas.PostV1AuthGuestReq, params oas.PostV1AuthGuestParams) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.CreateGuest(ctx, params.XClientID)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthPasswordChange(ctx context.Context, req *oas.PasswordChangeRequest) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordCheck(ctx context.Context, req *oas.PostV1AuthPasswordCheckReq, params oas.PostV1AuthPasswordCheckParams) (r *oas.PostV1AuthPasswordCheckOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordForgot(ctx context.Context, req *oas.PasswordForgotRequest, params oas.PostV1AuthPasswordForgotParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordReset(ctx context.Context, req *oas.PasswordResetRequest, params oas.PostV1AuthPasswordResetParams) (r *oas.AuthResult, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPasswordVerify(ctx context.Context, req *oas.PostV1AuthPasswordVerifyReq) (r *oas.PostV1AuthPasswordVerifyOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneChangeStart(ctx context.Context, req *oas.PostV1AuthPhoneChangeStartReq) (r *oas.Challenge, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneChangeVerify(ctx context.Context, req *oas.PostV1AuthPhoneChangeVerifyReq) (r *oas.PostV1AuthPhoneChangeVerifyOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneVerificationStart(ctx context.Context, req *oas.PostV1AuthPhoneVerificationStartReq, params oas.PostV1AuthPhoneVerificationStartParams) (r *oas.Challenge, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthPhoneVerificationVerify(ctx context.Context, req *oas.PostV1AuthPhoneVerificationVerifyReq, params oas.PostV1AuthPhoneVerificationVerifyParams) (r oas.PhoneVerifyResult, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSessionStepUp(ctx context.Context, req *oas.PostV1AuthSessionStepUpReq) (r oas.StepUpResult, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSessionSwitchGroup(ctx context.Context, req *oas.PostV1AuthSessionSwitchGroupReq) (r *oas.AuthResult, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSignInPassword(ctx context.Context, req *oas.PasswordSignInRequest, params oas.PostV1AuthSignInPasswordParams) (oas.AuthResultOrNextStep, error) {
	acct, sess, err := s.deps.Accounts.AuthenticatePassword(ctx, params.XClientID, req.Email.Or(""), req.Password)
	if err != nil {
		return oas.AuthResultOrNextStep{}, err
	}
	return oas.NewAuthResultAuthResultOrNextStep(*authResult(acct, sess)), nil
}

func (s *CoreAuthService) PostV1AuthSignOut(ctx context.Context, req oas.OptPostV1AuthSignOutReq) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	everywhere := false
	if v, ok := req.Get(); ok {
		everywhere = v.Everywhere.Or(false)
	}
	if err := s.deps.Accounts.SignOut(ctx, p.SessionID, everywhere); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *CoreAuthService) PostV1AuthSignOutAll(ctx context.Context, req oas.OptPostV1AuthSignOutAllReq) (r *oas.PostV1AuthSignOutAllOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1AuthSignUp(ctx context.Context, req *oas.SignUpRequest, params oas.PostV1AuthSignUpParams) (*oas.AuthResult, error) {
	cmd := domain.RegisterCmd{
		ProjectID: params.XClientID,
		Email:     req.Email.Or(""),
		Phone:     req.Phone.Or(""),
		Password:  req.Password.Or(""),
		Name:      req.Name.Or(""),
	}
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	acct, sess, err := s.deps.Accounts.Register(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthTokenExchange(ctx context.Context, req *oas.CodeExchangeRequest, params oas.PostV1AuthTokenExchangeParams) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.ExchangeCode(ctx, req.Code, req.CodeVerifier.Or(""))
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthTokenRefresh(ctx context.Context, req oas.OptRefreshRequest, params oas.PostV1AuthTokenRefreshParams) (*oas.AuthResult, error) {
	rt := ""
	if v, ok := req.Get(); ok {
		rt = v.RefreshToken.Or("")
	}
	if rt == "" {
		return nil, domain.ErrInvalidToken.WithMessage("refresh_token is required")
	}
	acct, sess, err := s.deps.Accounts.Refresh(ctx, rt)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1ChallengesCaptchaVerify(ctx context.Context, req *oas.PostV1ChallengesCaptchaVerifyReq) (r *oas.PostV1ChallengesCaptchaVerifyOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1TokensIntrospect(ctx context.Context, req *oas.PostV1TokensIntrospectReq) (r *oas.PostV1TokensIntrospectOK, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1TokensRevoke(ctx context.Context, req *oas.PostV1TokensRevokeReq) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *CoreAuthService) PostV1TokensVerify(ctx context.Context, req *oas.PostV1TokensVerifyReq) (r *oas.PostV1TokensVerifyOK, _ error) {
	panic("implement me")
}
