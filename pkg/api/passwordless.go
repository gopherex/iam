// Code scaffolded for IAM handler groups.
//
// PasswordlessService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type PasswordlessAccounts interface {
	StartOTP(ctx context.Context, projectID, identifier, channel, purpose, locale string) (*domain.Challenge, error)
	VerifyOTP(ctx context.Context, challengeID, code string) (*domain.Account, *domain.Session, error)
	StartMagicLink(ctx context.Context, projectID, email, redirectTo, locale string) (*domain.Challenge, error)
	VerifyMagicLink(ctx context.Context, token string) (*domain.Account, *domain.Session, error)
	// VerifyMagicLinkCallback consumes the link token like VerifyMagicLink and
	// additionally returns a redirect target sanitized against the project's
	// app_base_url (open-redirect safe), for the browser GET callback leg.
	VerifyMagicLinkCallback(
		ctx context.Context, token, redirectTo string,
	) (*domain.Account, *domain.Session, string, error)
}

type PasswordlessDeps struct{ Accounts PasswordlessAccounts }

// PasswordlessService implements the PasswordlessHandler slice of oas.Handler.
type PasswordlessService struct {
	oas.UnimplementedHandler
	deps PasswordlessDeps
}

// NewPasswordlessService builds the Passwordless service from its dependencies.
func NewPasswordlessService(deps PasswordlessDeps) *PasswordlessService {
	return &PasswordlessService{deps: deps}
}

var _ oas.Handler = (*PasswordlessService)(nil)

func (s *PasswordlessService) PostV1AuthOtpStart(
	ctx context.Context,
	req *oas.OtpStartRequest,
	params oas.PostV1AuthOtpStartParams,
) (*oas.Challenge, error) {
	ch, err := s.deps.Accounts.StartOTP(
		ctx, params.XClientID, req.Identifier, string(req.Channel), string(req.Purpose), req.Locale.Or(""),
	)
	if err != nil {
		return nil, err
	}

	return oasChallenge(ch), nil
}

func (s *PasswordlessService) PostV1AuthOtpVerify(
	ctx context.Context,
	req *oas.OtpVerifyRequest,
	_ oas.PostV1AuthOtpVerifyParams,
) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.VerifyOTP(ctx, req.ChallengeID, req.Code)
	if err != nil {
		return nil, err
	}

	return authResult(acct, sess), nil
}

func (s *PasswordlessService) PostV1AuthMagicLinkStart(
	ctx context.Context,
	req *oas.MagicLinkStartRequest,
	params oas.PostV1AuthMagicLinkStartParams,
) (*oas.Challenge, error) {
	ch, err := s.deps.Accounts.StartMagicLink(ctx, params.XClientID, req.Email, req.RedirectTo, req.Locale.Or(""))
	if err != nil {
		return nil, err
	}

	return oasChallenge(ch), nil
}

func (s *PasswordlessService) PostV1AuthMagicLinkVerify(
	ctx context.Context,
	req *oas.MagicLinkVerifyRequest,
	_ oas.PostV1AuthMagicLinkVerifyParams,
) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.VerifyMagicLink(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	return authResult(acct, sess), nil
}

// GetV1AuthMagicLinkCallback is the browser GET leg of a magic link: it consumes
// the token, mints a session, sets the session cookies and redirects the browser
// to the (sanitized) target. Mirrors the email-verification callback.
func (s *PasswordlessService) GetV1AuthMagicLinkCallback(
	ctx context.Context,
	params oas.GetV1AuthMagicLinkCallbackParams,
) (*oas.GetV1AuthMagicLinkCallbackFound, error) {
	_, sess, redirect, err := s.deps.Accounts.VerifyMagicLinkCallback(ctx, params.Token, params.RedirectTo.Or(""))
	if err != nil {
		return nil, err
	}

	out := &oas.GetV1AuthMagicLinkCallbackFound{Location: optURI(redirect)}
	out.SetCookie = SessionCookiesFor(sess)

	return out, nil
}
