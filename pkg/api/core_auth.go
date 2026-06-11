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
	AuthenticatePassword(ctx context.Context, projectID, email, password string) (*domain.CoreAuthPasswordResult, error)
	Refresh(ctx context.Context, refreshToken string) (*domain.Account, *domain.Session, error)
	ExchangeCode(ctx context.Context, code, verifier string) (*domain.Account, *domain.Session, error)
	RedeemImpersonation(ctx context.Context, token, clientID string) (*domain.Account, *domain.Session, error)
	CreateGuest(ctx context.Context, projectID string) (*domain.Account, *domain.Session, error)
	GetSession(ctx context.Context, sessionID string) (*domain.Account, *domain.Session, error)
	SignOut(ctx context.Context, sessionID string, everywhere bool) error
	SignOutAll(ctx context.Context, accountID, exceptSessionID string) (int, error)

	// Email verification / change.
	StartEmailVerification(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error)
	VerifyEmail(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, *domain.Session, error)
	VerifyEmailCallback(ctx context.Context, cmd domain.CoreAuthEmailVerificationCallbackCmd) (*domain.CoreAuthEmailVerificationCallbackResult, error)
	VerifyCaptcha(ctx context.Context, projectID, provider, token, action string) (*domain.CoreAuthCaptchaVerifyResult, error)
	StartEmailChange(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error)
	VerifyEmailChange(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, error)
	CancelEmailChange(ctx context.Context, token string) error

	// Phone verification / change.
	StartPhoneVerification(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error)
	VerifyPhone(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, *domain.Session, error)
	StartPhoneChange(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error)
	VerifyPhoneChange(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, error)

	// Password lifecycle.
	ForgotPassword(ctx context.Context, cmd domain.CoreAuthPasswordForgotCmd) error
	ResetPassword(ctx context.Context, cmd domain.CoreAuthPasswordResetCmd) (*domain.Account, *domain.Session, error)
	ChangePassword(ctx context.Context, cmd domain.CoreAuthPasswordChangeCmd) error
	CheckPassword(ctx context.Context, projectID, password string) (*domain.CoreAuthPasswordCheckResult, error)
	VerifyPassword(ctx context.Context, cmd domain.CoreAuthPasswordChangeCmd) (*domain.CoreAuthPasswordVerifyResult, error)

	// Session.
	StepUp(ctx context.Context, cmd domain.CoreAuthStepUpCmd) (*domain.CoreAuthStepUpResult, error)
	SwitchGroup(ctx context.Context, accountID, sessionID, groupID string) (*domain.Account, *domain.Session, error)

	// Access requests.
	CreateAccessRequest(ctx context.Context, cmd domain.CoreAuthAccessRequestCmd) (*domain.CoreAuthAccessRequest, error)
}

// CoreAuthTokens is the Core Auth slice of token introspection / verification.
// Each method is one atomic operation; the adapter owns its transaction.
type CoreAuthTokens interface {
	Introspect(ctx context.Context, projectID, token string) (*domain.CoreAuthTokenIntrospection, error)
	Verify(ctx context.Context, projectID, token, audience string) (*domain.CoreAuthTokenVerification, error)
	Revoke(ctx context.Context, cmd domain.CoreAuthRevokeCmd) error
	CurrentClaims(ctx context.Context, sessionID string) (map[string]any, error)
}

// CoreAuthDeps are the ports the Core Auth service orchestrates.
type CoreAuthDeps struct {
	Accounts CoreAuthAccounts
	Tokens   CoreAuthTokens
	MFA      CoreAuthMFA
}

// CoreAuthMFA issues the step-up challenge when password sign-in needs a second
// factor. The returned challenge id is the flow_token the client presents to
// mfa/verify or recovery-codes/verify to finish authentication.
type CoreAuthMFA interface {
	Challenge(ctx context.Context, accountID, factorID string) (*domain.Challenge, error)
}

// CoreAuthService implements the CoreAuthHandler slice of oas.Handler.
type CoreAuthService struct {
	oas.UnimplementedHandler
	deps CoreAuthDeps
}

// NewCoreAuthService builds the CoreAuth service from its dependencies.
func NewCoreAuthService(deps CoreAuthDeps) *CoreAuthService { return &CoreAuthService{deps: deps} }

var _ oas.Handler = (*CoreAuthService)(nil)

func (s *CoreAuthService) GetV1AuthEmailChangeCancel(ctx context.Context, params oas.GetV1AuthEmailChangeCancelParams) (*oas.Ok, error) {
	// Public op (security: []): the opaque token identifies the pending change.
	if err := s.deps.Accounts.CancelEmailChange(ctx, params.Token); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *CoreAuthService) GetV1AuthEmailVerificationCallback(ctx context.Context, params oas.GetV1AuthEmailVerificationCallbackParams) (r *oas.GetV1AuthEmailVerificationCallbackFound, _ error) {
	// Public op (security: []): the opaque token identifies the pending
	// verification. The port consumes it and returns where to redirect the
	// browser plus an optional session cookie.
	res, err := s.deps.Accounts.VerifyEmailCallback(ctx, domain.CoreAuthEmailVerificationCallbackCmd{
		Token:      params.Token,
		RedirectTo: params.RedirectTo.Or(""),
	})
	if err != nil {
		return nil, err
	}
	out := &oas.GetV1AuthEmailVerificationCallbackFound{Location: optURI(res.RedirectURL)}
	if res.SetCookie != "" {
		out.SetCookie = []string{res.SetCookie}
	}
	return out, nil
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

func (s *CoreAuthService) GetV1TokensCurrent(ctx context.Context) (*oas.GetV1TokensCurrentOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	claims, err := s.deps.Tokens.CurrentClaims(ctx, p.SessionID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1TokensCurrentOK{
		Claims: oas.NewOptGetV1TokensCurrentOKClaims(oasRawMap[oas.GetV1TokensCurrentOKClaims](claims)),
	}, nil
}

func (s *CoreAuthService) PostV1AuthAccessRequests(ctx context.Context, req *oas.PostV1AuthAccessRequestsReq, params oas.PostV1AuthAccessRequestsParams) (*oas.PostV1AuthAccessRequestsOK, error) {
	// Public op (security: []): the project is taken from X-Client-Id.
	ar, err := s.deps.Accounts.CreateAccessRequest(ctx, domain.CoreAuthAccessRequestCmd{
		ProjectID:    params.XClientID,
		Email:        req.Email,
		Reason:       req.Reason.Or(""),
		CaptchaToken: req.CaptchaToken.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthAccessRequestsOK{
		Request: oas.NewOptAccessRequest(oasCoreAuthAccessRequest(ar)),
	}, nil
}

func (s *CoreAuthService) PostV1AuthEmailChangeStart(ctx context.Context, req *oas.PostV1AuthEmailChangeStartReq) (*oas.Challenge, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ch, err := s.deps.Accounts.StartEmailChange(ctx, domain.CoreAuthVerifyStartCmd{
		ProjectID:  p.ProjectID,
		AccountID:  p.AccountID,
		Contact:    req.NewEmail,
		RedirectTo: req.RedirectTo.Or(""),
		Locale:     req.Locale.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return oasChallenge(ch), nil
}

func (s *CoreAuthService) PostV1AuthEmailChangeVerify(ctx context.Context, req *oas.PostV1AuthEmailChangeVerifyReq) (*oas.PostV1AuthEmailChangeVerifyOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.deps.Accounts.VerifyEmailChange(ctx, domain.CoreAuthVerifyConsumeCmd{
		ProjectID:   p.ProjectID,
		AccountID:   p.AccountID,
		ChallengeID: req.ChallengeID.Or(""),
		Code:        req.Code.Or(""),
		Token:       req.Token.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthEmailChangeVerifyOK{User: oas.NewOptUser(oasUser(acct))}, nil
}

func (s *CoreAuthService) PostV1AuthEmailVerificationStart(ctx context.Context, req *oas.PostV1AuthEmailVerificationStartReq, params oas.PostV1AuthEmailVerificationStartParams) (*oas.Challenge, error) {
	// Public op (security: []): project from X-Client-Id; email from the body.
	ch, err := s.deps.Accounts.StartEmailVerification(ctx, domain.CoreAuthVerifyStartCmd{
		ProjectID:  params.XClientID,
		Contact:    req.Email.Or(""),
		RedirectTo: req.RedirectTo.Or(""),
		Locale:     req.Locale.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return oasChallenge(ch), nil
}

func (s *CoreAuthService) PostV1AuthEmailVerificationVerify(ctx context.Context, req *oas.PostV1AuthEmailVerificationVerifyReq, params oas.PostV1AuthEmailVerificationVerifyParams) (*oas.AuthResult, error) {
	// Public op (security: []): a successful verify mints a session.
	acct, sess, err := s.deps.Accounts.VerifyEmail(ctx, domain.CoreAuthVerifyConsumeCmd{
		ProjectID:   params.XClientID,
		ChallengeID: req.ChallengeID.Or(""),
		Code:        req.Code.Or(""),
		Token:       req.Token.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthGuest(ctx context.Context, req *oas.PostV1AuthGuestReq, params oas.PostV1AuthGuestParams) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.CreateGuest(ctx, params.XClientID)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthPasswordChange(ctx context.Context, req *oas.PasswordChangeRequest) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.ChangePassword(ctx, domain.CoreAuthPasswordChangeCmd{
		AccountID:           p.AccountID,
		SessionID:           p.SessionID,
		CurrentPassword:     req.CurrentPassword.Or(""),
		NewPassword:         req.NewPassword,
		RevokeOtherSessions: req.RevokeOtherSessions.Or(false),
	}); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *CoreAuthService) PostV1AuthPasswordCheck(ctx context.Context, req *oas.PostV1AuthPasswordCheckReq, params oas.PostV1AuthPasswordCheckParams) (*oas.PostV1AuthPasswordCheckOK, error) {
	// Public op (security: []): project from X-Client-Id.
	res, err := s.deps.Accounts.CheckPassword(ctx, params.XClientID, req.Password)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthPasswordCheckOK{
		Valid:      oas.NewOptBool(res.Valid),
		Score:      oas.NewOptInt(res.Score),
		Violations: res.Violations,
	}, nil
}

func (s *CoreAuthService) PostV1AuthPasswordForgot(ctx context.Context, req *oas.PasswordForgotRequest, params oas.PostV1AuthPasswordForgotParams) (*oas.Ok, error) {
	// Public op (security: []): project from X-Client-Id.
	if err := s.deps.Accounts.ForgotPassword(ctx, domain.CoreAuthPasswordForgotCmd{
		ProjectID:    params.XClientID,
		Email:        req.Email,
		RedirectTo:   req.RedirectTo.Or(""),
		Locale:       req.Locale.Or(""),
		CaptchaToken: req.CaptchaToken.Or(""),
	}); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *CoreAuthService) PostV1AuthPasswordReset(ctx context.Context, req *oas.PasswordResetRequest, params oas.PostV1AuthPasswordResetParams) (*oas.AuthResult, error) {
	// Public op (security: []): the reset token/challenge identifies the account;
	// a successful reset mints a session.
	acct, sess, err := s.deps.Accounts.ResetPassword(ctx, domain.CoreAuthPasswordResetCmd{
		ProjectID:   params.XClientID,
		Token:       req.Token.Or(""),
		ChallengeID: req.ChallengeID.Or(""),
		Code:        req.Code.Or(""),
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthPasswordVerify(ctx context.Context, req *oas.PostV1AuthPasswordVerifyReq) (*oas.PostV1AuthPasswordVerifyOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.deps.Accounts.VerifyPassword(ctx, domain.CoreAuthPasswordChangeCmd{
		AccountID:       p.AccountID,
		SessionID:       p.SessionID,
		CurrentPassword: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthPasswordVerifyOK{
		Ok:  oas.NewOptBool(res.OK),
		Aal: oas.NewOptInt(res.AAL),
		Amr: res.AMR,
	}, nil
}

func (s *CoreAuthService) PostV1AuthPhoneChangeStart(ctx context.Context, req *oas.PostV1AuthPhoneChangeStartReq) (*oas.Challenge, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ch, err := s.deps.Accounts.StartPhoneChange(ctx, domain.CoreAuthVerifyStartCmd{
		ProjectID: p.ProjectID,
		AccountID: p.AccountID,
		Contact:   req.NewPhone,
		Channel:   string(req.Channel.Or("")),
	})
	if err != nil {
		return nil, err
	}
	return oasChallenge(ch), nil
}

func (s *CoreAuthService) PostV1AuthPhoneChangeVerify(ctx context.Context, req *oas.PostV1AuthPhoneChangeVerifyReq) (*oas.PostV1AuthPhoneChangeVerifyOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.deps.Accounts.VerifyPhoneChange(ctx, domain.CoreAuthVerifyConsumeCmd{
		ProjectID:   p.ProjectID,
		AccountID:   p.AccountID,
		ChallengeID: req.ChallengeID,
		Code:        req.Code,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthPhoneChangeVerifyOK{User: oas.NewOptUser(oasUser(acct))}, nil
}

func (s *CoreAuthService) PostV1AuthPhoneVerificationStart(ctx context.Context, req *oas.PostV1AuthPhoneVerificationStartReq, params oas.PostV1AuthPhoneVerificationStartParams) (*oas.Challenge, error) {
	// Public op (security: []): project from X-Client-Id.
	ch, err := s.deps.Accounts.StartPhoneVerification(ctx, domain.CoreAuthVerifyStartCmd{
		ProjectID: params.XClientID,
		Contact:   req.Phone,
		Channel:   string(req.Channel.Or("")),
		Locale:    req.Locale.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return oasChallenge(ch), nil
}

func (s *CoreAuthService) PostV1AuthPhoneVerificationVerify(ctx context.Context, req *oas.PostV1AuthPhoneVerificationVerifyReq, params oas.PostV1AuthPhoneVerificationVerifyParams) (oas.PhoneVerifyResult, error) {
	// Public op (security: []): a successful verify mints a session.
	acct, sess, err := s.deps.Accounts.VerifyPhone(ctx, domain.CoreAuthVerifyConsumeCmd{
		ProjectID:   params.XClientID,
		ChallengeID: req.ChallengeID,
		Code:        req.Code,
	})
	if err != nil {
		return oas.PhoneVerifyResult{}, err
	}
	return oas.NewAuthResultPhoneVerifyResult(*authResult(acct, sess)), nil
}

func (s *CoreAuthService) PostV1AuthSessionStepUp(ctx context.Context, req *oas.PostV1AuthSessionStepUpReq) (oas.StepUpResult, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return oas.StepUpResult{}, err
	}
	cmd := domain.CoreAuthStepUpCmd{
		AccountID:   p.AccountID,
		SessionID:   p.SessionID,
		Purpose:     req.Purpose,
		RequiredAAL: int(req.RequiredAal.Or(0)),
	}
	if v, ok := req.MaxAgeSeconds.Get(); ok {
		cmd.MaxAgeSeconds = v
		cmd.HasMaxAge = true
	}
	res, err := s.deps.Accounts.StepUp(ctx, cmd)
	if err != nil {
		return oas.StepUpResult{}, err
	}
	if res.Satisfied {
		return oas.NewOkResultStepUpResult(oas.OkResult{
			Ok:         oas.NewOptBool(true),
			ResultType: oas.OkResultResultTypeOk,
		}), nil
	}
	next := oas.AuthNextStep{ResultType: oas.AuthNextStepResultTypeNextStep}
	next.NextStep.SetTo(oas.NextStepStepUp)
	if res.Challenge != nil {
		next.FlowToken = oas.NewOptString(res.Challenge.ID)
	}
	return oas.NewAuthNextStepStepUpResult(next), nil
}

func (s *CoreAuthService) PostV1AuthSessionSwitchGroup(ctx context.Context, req *oas.PostV1AuthSessionSwitchGroupReq) (*oas.AuthResult, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	acct, sess, err := s.deps.Accounts.SwitchGroup(ctx, p.AccountID, p.SessionID, req.GroupID)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthSignInPassword(ctx context.Context, req *oas.PasswordSignInRequest, params oas.PostV1AuthSignInPasswordParams) (oas.AuthResultOrNextStep, error) {
	res, err := s.deps.Accounts.AuthenticatePassword(ctx, params.XClientID, req.Email.Or(""), req.Password)
	if err != nil {
		return oas.AuthResultOrNextStep{}, err
	}
	if res.MFARequired {
		// Issue a step-up challenge for the account's primary factor; its id is the
		// flow_token the client presents to mfa/verify or recovery-codes/verify.
		ch, err := s.deps.MFA.Challenge(ctx, res.Account.ID, primaryFactorID(res.Factors))
		if err != nil {
			return oas.AuthResultOrNextStep{}, err
		}
		next := oas.AuthNextStep{
			ResultType: oas.AuthNextStepResultTypeNextStep,
			FlowToken:  oas.NewOptString(ch.ID),
			Factors:    oasFactors(res.Factors),
		}
		next.NextStep.SetTo(oas.NextStepMfaRequired)
		return oas.NewAuthNextStepAuthResultOrNextStep(next), nil
	}
	return oas.NewAuthResultAuthResultOrNextStep(*authResult(res.Account, res.Session)), nil
}

// primaryFactorID picks the factor to challenge first at sign-in: prefer factors
// that need no out-of-band delivery (TOTP, WebAuthn), else the first active one.
func primaryFactorID(factors []domain.Factor) string {
	for _, f := range factors {
		if f.Type == "totp" || f.Type == "webauthn" {
			return f.ID
		}
	}
	if len(factors) > 0 {
		return factors[0].ID
	}
	return ""
}

// oasFactors maps domain factors to their wire form for the MFA next-step.
func oasFactors(factors []domain.Factor) []oas.Factor {
	out := make([]oas.Factor, 0, len(factors))
	for _, f := range factors {
		item := oas.Factor{
			ID:     oas.NewOptString(f.ID),
			Type:   oas.NewOptFactorType(oas.FactorType(f.Type)),
			Status: oas.NewOptFactorStatus(oas.FactorStatus(f.Status)),
		}
		if f.Hint != "" {
			item.Hint = oas.NewOptNilString(f.Hint)
		}
		out = append(out, item)
	}
	return out
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

func (s *CoreAuthService) PostV1AuthSignOutAll(ctx context.Context, req oas.OptPostV1AuthSignOutAllReq) (*oas.PostV1AuthSignOutAllOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	except := ""
	if v, ok := req.Get(); ok && v.ExceptCurrent.Or(false) {
		except = p.SessionID
	}
	n, err := s.deps.Accounts.SignOutAll(ctx, p.AccountID, except)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthSignOutAllOK{RevokedCount: oas.NewOptInt(n)}, nil
}

func (s *CoreAuthService) PostV1AuthSignUp(ctx context.Context, req *oas.SignUpRequest, params oas.PostV1AuthSignUpParams) (*oas.AuthResult, error) {
	consents := make([]domain.AccountConsentAcceptance, 0, len(req.Consents))
	for _, c := range req.Consents {
		consents = append(consents, domain.AccountConsentAcceptance{Key: c.Key, Version: c.Version})
	}
	cmd := domain.RegisterCmd{
		ProjectID: params.XClientID,
		Email:     req.Email.Or(""),
		Phone:     req.Phone.Or(""),
		Password:  req.Password.Or(""),
		Name:      req.Name.Or(""),
		Locale:    req.Locale.Or(""),
		Consents:  consents,
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

func (s *CoreAuthService) PostV1AuthImpersonateRedeem(ctx context.Context, req *oas.PostV1AuthImpersonateRedeemReq, params oas.PostV1AuthImpersonateRedeemParams) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.RedeemImpersonation(ctx, req.Token, params.XClientID)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *CoreAuthService) PostV1AuthTokenRefresh(ctx context.Context, req oas.OptRefreshRequest, params oas.PostV1AuthTokenRefreshParams) (*oas.AuthResultHeaders, error) {
	rt := ""
	if v, ok := req.Get(); ok {
		rt = v.RefreshToken.Or("")
	}
	// Cookie mode: when the body omits the token, take it from the refresh cookie.
	cookieMode := false
	if rt == "" {
		if v, ok := params.IamRefresh.Get(); ok && v != "" {
			rt = v
			cookieMode = true
		}
	}
	if rt == "" {
		return nil, domain.ErrInvalidToken.WithMessage("refresh_token is required")
	}
	acct, sess, err := s.deps.Accounts.Refresh(ctx, rt)
	if err != nil {
		return nil, err
	}
	out := &oas.AuthResultHeaders{Response: *authResult(acct, sess)}
	// Rotate both cookies for a cookie-mode refresh; token-mode callers get the
	// rotated tokens in the body only.
	if cookieMode {
		out.SetCookie = SessionCookies(sess.AccessToken, sess.RefreshToken, cookieAccessTTL, cookieRefreshTTL)
	}
	return out, nil
}

func (s *CoreAuthService) PostV1ChallengesCaptchaVerify(ctx context.Context, req *oas.PostV1ChallengesCaptchaVerifyReq) (r *oas.PostV1ChallengesCaptchaVerifyOK, _ error) {
	// Public op (security: []): verify a CAPTCHA token against the configured
	// provider. The adapter resolves the project from request context.
	res, err := s.deps.Accounts.VerifyCaptcha(ctx, "", req.Provider, req.Token, req.Action.Or(""))
	if err != nil {
		return nil, err
	}
	return &oas.PostV1ChallengesCaptchaVerifyOK{
		Valid: oas.NewOptBool(res.Valid),
		Score: oas.NewOptFloat64(res.Score),
	}, nil
}

func (s *CoreAuthService) PostV1TokensIntrospect(ctx context.Context, req *oas.PostV1TokensIntrospectReq) (*oas.PostV1TokensIntrospectOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	intro, err := s.deps.Tokens.Introspect(ctx, p.ProjectID, req.Token)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1TokensIntrospectOK{
		Active:          oas.NewOptBool(intro.Active),
		AdditionalProps: oasRawMap[oas.PostV1TokensIntrospectOKAdditional](intro.Claims),
	}, nil
}

func (s *CoreAuthService) PostV1TokensRevoke(ctx context.Context, req *oas.PostV1TokensRevokeReq) (*oas.Ok, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	if err := s.deps.Tokens.Revoke(ctx, domain.CoreAuthRevokeCmd{
		Token:     req.Token.Or(""),
		SessionID: req.SessionID.Or(""),
		Reason:    req.Reason.Or(""),
	}); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *CoreAuthService) PostV1TokensVerify(ctx context.Context, req *oas.PostV1TokensVerifyReq) (*oas.PostV1TokensVerifyOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.deps.Tokens.Verify(ctx, p.ProjectID, req.Token, req.Audience.Or(""))
	if err != nil {
		return nil, err
	}
	out := &oas.PostV1TokensVerifyOK{
		Valid:  oas.NewOptBool(v.Valid),
		Claims: oas.NewOptPostV1TokensVerifyOKClaims(oasRawMap[oas.PostV1TokensVerifyOKClaims](v.Claims)),
	}
	if v.Error != "" {
		out.Error = oas.NewOptNilString(v.Error)
	}
	return out, nil
}

// ----- service-local mappers -----

// oasCoreAuthAccessRequest maps a domain access request to its wire form.
func oasCoreAuthAccessRequest(ar *domain.CoreAuthAccessRequest) oas.AccessRequest {
	out := oas.AccessRequest{
		ID:    oas.NewOptString(ar.ID),
		Email: oas.NewOptString(ar.Email),
	}
	if ar.Reason != "" {
		out.Reason = oas.NewOptNilString(ar.Reason)
	}
	if ar.Status != "" {
		out.Status = oas.NewOptAccessRequestStatus(oas.AccessRequestStatus(ar.Status))
	}
	return out
}
