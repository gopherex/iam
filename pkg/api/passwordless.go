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
	StartOTP(ctx context.Context, projectID, identifier, channel, purpose string) (*domain.Challenge, error)
	VerifyOTP(ctx context.Context, challengeID, code string) (*domain.Account, *domain.Session, error)
	StartMagicLink(ctx context.Context, projectID, email, redirectTo string) (*domain.Challenge, error)
	VerifyMagicLink(ctx context.Context, token string) (*domain.Account, *domain.Session, error)
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

func (s *PasswordlessService) PostV1AuthOtpStart(ctx context.Context, req *oas.OtpStartRequest, params oas.PostV1AuthOtpStartParams) (*oas.Challenge, error) {
	ch, err := s.deps.Accounts.StartOTP(ctx, params.XClientID, req.Identifier, string(req.Channel), string(req.Purpose))
	if err != nil {
		return nil, err
	}
	return oasChallenge(ch), nil
}

func (s *PasswordlessService) PostV1AuthOtpVerify(ctx context.Context, req *oas.OtpVerifyRequest, params oas.PostV1AuthOtpVerifyParams) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.VerifyOTP(ctx, req.ChallengeID, req.Code)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *PasswordlessService) PostV1AuthMagicLinkStart(ctx context.Context, req *oas.MagicLinkStartRequest, params oas.PostV1AuthMagicLinkStartParams) (*oas.Challenge, error) {
	ch, err := s.deps.Accounts.StartMagicLink(ctx, params.XClientID, req.Email, req.RedirectTo)
	if err != nil {
		return nil, err
	}
	return oasChallenge(ch), nil
}

func (s *PasswordlessService) PostV1AuthMagicLinkVerify(ctx context.Context, req *oas.MagicLinkVerifyRequest, params oas.PostV1AuthMagicLinkVerifyParams) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.VerifyMagicLink(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}
