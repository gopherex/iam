// Code scaffolded for IAM handler groups.
//
// WebAuthnService is pure orchestration: it holds aggregate-port interfaces (deps) and
// nothing else. It embeds oas.UnimplementedHandler so any operation it does not
// override returns not-implemented, and panics on every v1.0.0 operation until
// written. Each port method is atomic in its adapter — services never open a
// transaction.

package api

import (
	"context"
	"encoding/json"

	"github.com/go-faster/jx"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type WebAuthnAccounts interface {
	BeginLogin(ctx context.Context, projectID, email string) (*domain.Challenge, error)
	FinishLogin(ctx context.Context, challengeID string, credential map[string]any) (*domain.Account, *domain.Session, error)
	BeginRegistration(ctx context.Context, accountID string) (*domain.Challenge, error)
	FinishRegistration(ctx context.Context, accountID, challengeID string, credential map[string]any) (*domain.WebAuthnCredential, error)
	ListCredentials(ctx context.Context, accountID string) ([]domain.WebAuthnCredential, error)
	RemoveCredential(ctx context.Context, accountID, credentialID string) error
	RenameCredential(ctx context.Context, cmd domain.WebAuthnRenameCredentialCmd) (*domain.WebAuthnCredential, error)
}

type WebAuthnDeps struct{ Accounts WebAuthnAccounts }

// WebAuthnService implements the WebAuthnHandler slice of oas.Handler.
type WebAuthnService struct {
	oas.UnimplementedHandler
	deps WebAuthnDeps
}

// NewWebAuthnService builds the WebAuthn service from its dependencies.
func NewWebAuthnService(deps WebAuthnDeps) *WebAuthnService { return &WebAuthnService{deps: deps} }

var _ oas.Handler = (*WebAuthnService)(nil)

func (s *WebAuthnService) PostV1AuthWebauthnLoginOptions(ctx context.Context, req oas.OptPostV1AuthWebauthnLoginOptionsReq, params oas.PostV1AuthWebauthnLoginOptionsParams) (*oas.PostV1AuthWebauthnLoginOptionsOK, error) {
	email := ""
	if v, ok := req.Get(); ok {
		email = v.Email.Or("")
	}
	ch, err := s.deps.Accounts.BeginLogin(ctx, params.XClientID, email)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthWebauthnLoginOptionsOK{
		ChallengeID: oas.NewOptString(ch.ID),
		PublicKey:   oas.NewOptPostV1AuthWebauthnLoginOptionsOKPublicKey(oasRawMap[oas.PostV1AuthWebauthnLoginOptionsOKPublicKey](ch.PublicKey)),
	}, nil
}

func (s *WebAuthnService) PostV1AuthWebauthnLoginVerify(ctx context.Context, req *oas.PostV1AuthWebauthnLoginVerifyReq, params oas.PostV1AuthWebauthnLoginVerifyParams) (*oas.AuthResult, error) {
	acct, sess, err := s.deps.Accounts.FinishLogin(ctx, req.ChallengeID, anyMap(req.Credential))
	if err != nil {
		return nil, err
	}
	return authResult(acct, sess), nil
}

func (s *WebAuthnService) PostV1AuthWebauthnRegisterOptions(ctx context.Context, req oas.OptPostV1AuthWebauthnRegisterOptionsReq) (*oas.PostV1AuthWebauthnRegisterOptionsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ch, err := s.deps.Accounts.BeginRegistration(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthWebauthnRegisterOptionsOK{
		ChallengeID: oas.NewOptString(ch.ID),
		PublicKey:   oas.NewOptPostV1AuthWebauthnRegisterOptionsOKPublicKey(oasRawMap[oas.PostV1AuthWebauthnRegisterOptionsOKPublicKey](ch.PublicKey)),
	}, nil
}

func (s *WebAuthnService) PostV1AuthWebauthnRegisterVerify(ctx context.Context, req *oas.PostV1AuthWebauthnRegisterVerifyReq) (*oas.PostV1AuthWebauthnRegisterVerifyOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	cred, err := s.deps.Accounts.FinishRegistration(ctx, p.AccountID, req.ChallengeID, anyMap(req.Credential))
	if err != nil {
		return nil, err
	}
	return &oas.PostV1AuthWebauthnRegisterVerifyOK{
		Credential: oas.NewOptWebAuthnCredential(oasWebAuthnCredential(*cred)),
	}, nil
}

func (s *WebAuthnService) GetV1AuthWebauthnCredentials(ctx context.Context) (*oas.GetV1AuthWebauthnCredentialsOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	creds, err := s.deps.Accounts.ListCredentials(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	data := make([]oas.WebAuthnCredential, 0, len(creds))
	for _, c := range creds {
		data = append(data, oasWebAuthnCredential(c))
	}
	return &oas.GetV1AuthWebauthnCredentialsOK{Data: data}, nil
}

func (s *WebAuthnService) DeleteV1AuthWebauthnCredentialsByCredentialId(ctx context.Context, params oas.DeleteV1AuthWebauthnCredentialsByCredentialIdParams) (*oas.Ok, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.deps.Accounts.RemoveCredential(ctx, p.AccountID, params.CredentialID); err != nil {
		return nil, err
	}
	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *WebAuthnService) PatchV1AuthWebauthnCredentialsByCredentialId(ctx context.Context, req *oas.PatchV1AuthWebauthnCredentialsByCredentialIdReq, params oas.PatchV1AuthWebauthnCredentialsByCredentialIdParams) (*oas.PatchV1AuthWebauthnCredentialsByCredentialIdOK, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	cred, err := s.deps.Accounts.RenameCredential(ctx, domain.WebAuthnRenameCredentialCmd{
		AccountID:    p.AccountID,
		CredentialID: params.CredentialID,
		Name:         req.Name,
	})
	if err != nil {
		return nil, err
	}
	return &oas.PatchV1AuthWebauthnCredentialsByCredentialIdOK{
		Credential: oas.NewOptWebAuthnCredential(oasWebAuthnCredential(*cred)),
	}, nil
}

// ----- service-local mappers -----

// oasWebAuthnCredential maps a domain credential onto the wire type.
func oasWebAuthnCredential(c domain.WebAuthnCredential) oas.WebAuthnCredential {
	cred := oas.WebAuthnCredential{}
	if c.ID != "" {
		cred.ID = oas.NewOptString(c.ID)
	}
	if c.Name != "" {
		cred.Name = oas.NewOptString(c.Name)
	}
	if !c.CreatedAt.IsZero() {
		cred.CreatedAt = oas.NewOptTimestamp(oas.Timestamp(c.CreatedAt))
	}
	if !c.LastUsedAt.IsZero() {
		cred.LastUsedAt = oas.NewOptTimestamp(oas.Timestamp(c.LastUsedAt))
	}
	return cred
}

// oasRawMap encodes a domain map into a wire map of raw JSON values (the
// generated publicKey option types are all map[string]jx.Raw aliases).
func oasRawMap[T ~map[string]jx.Raw](m map[string]any) T {
	out := make(T, len(m))
	for k, v := range m {
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		out[k] = jx.Raw(b)
	}
	return out
}

// anyMap decodes a wire map of raw JSON values into a domain map.
func anyMap[T ~map[string]jx.Raw](m T) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		var dst any
		if err := json.Unmarshal([]byte(v), &dst); err != nil {
			continue
		}
		out[k] = dst
	}
	return out
}
