// Code scaffolded for IAM handler groups.
//
// MFAService is pure orchestration: it holds aggregate-port interfaces (deps) and
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

type MFAAccounts interface {
	ListFactors(ctx context.Context, accountID string) ([]domain.Factor, error)
	EnrollTOTP(ctx context.Context, accountID string) (*domain.Factor, error)
	Challenge(ctx context.Context, accountID, factorID string) (*domain.Challenge, error)
	Verify(ctx context.Context, challengeID, code string) (*domain.Account, *domain.Session, error)
	GenerateRecoveryCodes(ctx context.Context, accountID string) ([]string, error)
	RemoveFactor(ctx context.Context, accountID, factorID string) error
}

type MFADeps struct{ Accounts MFAAccounts }

// MFAService implements the MFAHandler slice of oas.Handler.
type MFAService struct {
	oas.UnimplementedHandler
	deps MFADeps
}

// NewMFAService builds the MFA service from its dependencies.
func NewMFAService(deps MFADeps) *MFAService { return &MFAService{deps: deps} }

var _ oas.Handler = (*MFAService)(nil)

func (s *MFAService) DeleteV1AuthMfaFactorsByFactorId(ctx context.Context, params oas.DeleteV1AuthMfaFactorsByFactorIdParams) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.RemoveFactor(ctx, p.AccountID, params.FactorID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *MFAService) GetV1AuthMfaFactors(ctx context.Context) (*oas.GetV1AuthMfaFactorsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	factors, err := s.deps.Accounts.ListFactors(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	out := make([]oas.Factor, 0, len(factors))
	for i := range factors {
		out = append(out, oasFactor(&factors[i]))
	}
	return &oas.GetV1AuthMfaFactorsOK{Data: out}, nil
}

func (s *MFAService) PostV1AuthMfaChallenge(ctx context.Context, req oas.OptPostV1AuthMfaChallengeReq, params oas.PostV1AuthMfaChallengeParams) (*oas.Challenge, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	factorID := ""
	if v, ok := req.Get(); ok {
		factorID = v.FactorID.Or("")
	}
	ch, err := s.deps.Accounts.Challenge(ctx, p.AccountID, factorID)
	if err != nil {
		return nil, err
	}
	return oasChallenge(ch), nil
}

func (s *MFAService) PostV1AuthMfaEmailEnroll(ctx context.Context, req *oas.PostV1AuthMfaEmailEnrollReq) (r *oas.PostV1AuthMfaEmailEnrollOK, _ error) {
	panic("implement me")
}

func (s *MFAService) PostV1AuthMfaRecoveryCodesGenerate(ctx context.Context, req oas.OptPostV1AuthMfaRecoveryCodesGenerateReq) (*oas.PostV1AuthMfaRecoveryCodesGenerateOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	codes, err := s.deps.Accounts.GenerateRecoveryCodes(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthMfaRecoveryCodesGenerateOK{Codes: codes}, nil
}

func (s *MFAService) PostV1AuthMfaRecoveryCodesVerify(ctx context.Context, req *oas.PostV1AuthMfaRecoveryCodesVerifyReq, params oas.PostV1AuthMfaRecoveryCodesVerifyParams) (r *oas.AuthResult, _ error) {
	panic("implement me")
}

func (s *MFAService) PostV1AuthMfaSmsEnroll(ctx context.Context, req *oas.PostV1AuthMfaSmsEnrollReq) (r *oas.PostV1AuthMfaSmsEnrollOK, _ error) {
	panic("implement me")
}

func (s *MFAService) PostV1AuthMfaTotpEnroll(ctx context.Context, req oas.OptPostV1AuthMfaTotpEnrollReq) (*oas.PostV1AuthMfaTotpEnrollOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	factor, err := s.deps.Accounts.EnrollTOTP(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthMfaTotpEnrollOK{
		FactorID: oas.NewOptString(factor.ID),
	}, nil
}

func (s *MFAService) PostV1AuthMfaTotpVerify(ctx context.Context, req *oas.PostV1AuthMfaTotpVerifyReq) (r *oas.PostV1AuthMfaTotpVerifyOK, _ error) {
	panic("implement me")
}

func (s *MFAService) PostV1AuthMfaVerify(ctx context.Context, req *oas.PostV1AuthMfaVerifyReq, params oas.PostV1AuthMfaVerifyParams) (*oas.AuthResult, error) {
	challengeID := req.ChallengeID.Or("")
	if challengeID == "" {
		return nil, domain.ErrValidation.WithMessage("challenge_id is required")
	}
	acct, sess, err := s.deps.Accounts.Verify(ctx, challengeID, req.Code.Or(""))
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *MFAService) PostV1AuthMfaWebauthnEnrollOptions(ctx context.Context, req oas.OptPostV1AuthMfaWebauthnEnrollOptionsReq) (r *oas.PostV1AuthMfaWebauthnEnrollOptionsOK, _ error) {
	panic("implement me")
}

func (s *MFAService) PostV1AuthMfaWebauthnEnrollVerify(ctx context.Context, req *oas.PostV1AuthMfaWebauthnEnrollVerifyReq) (r *oas.PostV1AuthMfaWebauthnEnrollVerifyOK, _ error) {
	panic("implement me")
}

// oasFactor maps a domain Factor to its wire representation.
func oasFactor(f *domain.Factor) oas.Factor {
	out := oas.Factor{
		ID: oas.NewOptString(f.ID),
	}
	if f.Type != "" {
		out.Type = oas.NewOptFactorType(oas.FactorType(f.Type))
	}
	if f.Status != "" {
		out.Status = oas.NewOptFactorStatus(oas.FactorStatus(f.Status))
	}
	if f.Hint != "" {
		out.Hint = oas.NewOptNilString(f.Hint)
	}
	return out
}

// oasChallenge maps a domain Challenge to its wire representation.
func oasChallenge(c *domain.Challenge) *oas.Challenge {
	out := &oas.Challenge{
		ChallengeID: c.ID,
		ExpiresAt:   oas.Timestamp(c.ExpiresAt),
	}
	if c.Type != "" {
		out.Type = oas.NewOptString(c.Type)
	}
	return out
}
