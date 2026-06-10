package api

// CoreAuthFlows is the port for the server-side resumable auth flow engine
// (§9 architecture). Each method is one atomic port call; the adapter owns
// its transaction. Create returns the plain-text flow_token (never persisted);
// Get/Submit/Resend/Abandon look up by the token supplied in the request.

import (
	"context"
	"strings"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

// CoreAuthFlows is the port the CoreAuthFlowService orchestrates.
type CoreAuthFlows interface {
	Create(ctx context.Context, cmd domain.FlowCreateCmd) (*domain.FlowState, error)
	Get(ctx context.Context, cmd domain.FlowGetCmd) (*domain.FlowState, error)
	Submit(ctx context.Context, cmd domain.FlowSubmitCmd) (*domain.FlowState, error)
	Resend(ctx context.Context, cmd domain.FlowResendCmd) (*domain.FlowState, error)
	Abandon(ctx context.Context, cmd domain.FlowAbandonCmd) error
}

// CoreAuthFlowDeps are the ports the CoreAuthFlowService orchestrates.
type CoreAuthFlowDeps struct {
	Flows CoreAuthFlows
}

// CoreAuthFlowService implements the flow-related operations in the CoreAuth
// ogen handler group. It maps HTTP ↔ port and builds the oas.FlowState response.
type CoreAuthFlowService struct {
	oas.UnimplementedHandler
	deps CoreAuthFlowDeps
}

// NewCoreAuthFlowService builds the flow service from its dependencies.
func NewCoreAuthFlowService(deps CoreAuthFlowDeps) *CoreAuthFlowService {
	return &CoreAuthFlowService{deps: deps}
}

var _ oas.CoreAuthHandler = (*CoreAuthFlowService)(nil)

// PostV1AuthFlows creates a new server-side resumable auth flow.
func (s *CoreAuthFlowService) PostV1AuthFlows(ctx context.Context, req *oas.FlowCreateRequest, params oas.PostV1AuthFlowsParams) (*oas.FlowState, error) {
	fs, err := s.deps.Flows.Create(ctx, domain.FlowCreateCmd{
		ProjectID:    params.XClientID,
		Kind:         domain.FlowKind(req.Kind),
		Email:        req.Email.Or(""),
		Password:     req.Password.Or(""),
		Name:         req.Name.Or(""),
		CaptchaToken: req.CaptchaToken.Or(""),
	})
	if err != nil {
		return nil, err
	}
	return oasFlowState(fs), nil
}

// GetV1AuthFlowsByFlowToken retrieves a live flow by its opaque token.
func (s *CoreAuthFlowService) GetV1AuthFlowsByFlowToken(ctx context.Context, params oas.GetV1AuthFlowsByFlowTokenParams) (*oas.FlowState, error) {
	fs, err := s.deps.Flows.Get(ctx, domain.FlowGetCmd{
		ProjectID: params.XClientID,
		FlowToken: params.FlowToken,
	})
	if err != nil {
		return nil, err
	}
	return oasFlowState(fs), nil
}

// GetV1AuthFlowsCurrent is Phase 3 (cookie-bound flow). Until then returns 404.
func (s *CoreAuthFlowService) GetV1AuthFlowsCurrent(_ context.Context, _ oas.GetV1AuthFlowsCurrentParams) (*oas.FlowState, error) {
	return nil, domain.ErrNotFound
}

// PostV1AuthFlowsByFlowTokenSubmit advances the flow state machine.
func (s *CoreAuthFlowService) PostV1AuthFlowsByFlowTokenSubmit(ctx context.Context, req *oas.FlowSubmitRequest, params oas.PostV1AuthFlowsByFlowTokenSubmitParams) (*oas.FlowState, error) {
	payload := make(map[string]string)
	if p, ok := req.Payload.Get(); ok {
		for k, raw := range p {
			// Each value is a jx.Raw (JSON bytes). For string scalars we strip
			// the surrounding JSON quotes. Non-string values are passed as-is.
			s2 := string(raw)
			if len(s2) >= 2 && s2[0] == '"' && s2[len(s2)-1] == '"' {
				s2 = s2[1 : len(s2)-1]
			}
			payload[k] = s2
		}
	}
	fs, err := s.deps.Flows.Submit(ctx, domain.FlowSubmitCmd{
		ProjectID: params.XClientID,
		FlowToken: params.FlowToken,
		Action:    req.Action,
		Payload:   payload,
	})
	if err != nil {
		return nil, err
	}
	return oasFlowState(fs), nil
}

// PostV1AuthFlowsByFlowTokenResend re-issues the active challenge.
func (s *CoreAuthFlowService) PostV1AuthFlowsByFlowTokenResend(ctx context.Context, params oas.PostV1AuthFlowsByFlowTokenResendParams) (*oas.FlowState, error) {
	fs, err := s.deps.Flows.Resend(ctx, domain.FlowResendCmd{
		ProjectID: params.XClientID,
		FlowToken: params.FlowToken,
	})
	if err != nil {
		return nil, err
	}
	return oasFlowState(fs), nil
}

// DeleteV1AuthFlowsByFlowToken abandons a live flow.
func (s *CoreAuthFlowService) DeleteV1AuthFlowsByFlowToken(ctx context.Context, params oas.DeleteV1AuthFlowsByFlowTokenParams) error {
	return s.deps.Flows.Abandon(ctx, domain.FlowAbandonCmd{
		ProjectID: params.XClientID,
		FlowToken: params.FlowToken,
	})
}

// ─── mapper ───────────────────────────────────────────────────────────────────

// oasFlowState maps a domain.FlowState onto the wire oas.FlowState.
func oasFlowState(fs *domain.FlowState) *oas.FlowState {
	f := fs.Flow
	out := &oas.FlowState{
		FlowToken:   fs.FlowToken,
		Kind:        oas.FlowStateKind(f.Kind),
		Status:      oas.FlowStateStatus(f.Status),
		Step:        oas.FlowStateStep(f.Step),
		NextActions: flowNextActions(f),
		ExpiresAt:   oas.NewOptTimestamp(oas.Timestamp(f.ExpiresAt)),
	}
	// Masked contact (§5 rule 10).
	if f.Contact.Email != "" || f.Contact.Phone != "" {
		fc := oas.FlowContact{}
		if f.Contact.Email != "" {
			fc.EmailMasked = oas.NewOptString(maskEmail(f.Contact.Email))
		}
		if f.Contact.Phone != "" {
			fc.PhoneMasked = oas.NewOptString(maskPhone(f.Contact.Phone))
		}
		out.Contact = oas.NewOptFlowContact(fc)
	}
	// Active challenge (if any).
	if ac := f.ActiveChallenge; ac != nil {
		out.Challenge = oas.NewOptFlowChallenge(oas.FlowChallenge{
			Channel:      oas.NewOptString(ac.Channel),
			ExpiresAt:    oas.NewOptTimestamp(oas.Timestamp(ac.ExpiresAt)),
			ResendAt:     oas.NewOptTimestamp(oas.Timestamp(ac.ResendAt)),
			AttemptsLeft: oas.NewOptInt(ac.AttemptsLeft),
		})
	}
	// Consents.
	if len(f.ConsentsRequired) > 0 {
		refs := make([]oas.ConsentDocRef, 0, len(f.ConsentsRequired))
		for _, c := range f.ConsentsRequired {
			refs = append(refs, oas.ConsentDocRef{
				Key:     oas.NewOptString(c.Key),
				Version: oas.NewOptString(c.Version),
			})
		}
		out.ConsentsRequired = refs
	}
	// Error (within a still-pending flow).
	if fe := f.Error; fe != nil {
		out.Error = oas.NewOptFlowError(oas.FlowError{
			Code:    oas.NewOptString(fe.Code),
			Message: oas.NewOptString(fe.Message),
		})
	}
	// Session (completed only).
	if fs.Session != nil {
		out.Session = oas.NewOptSessionTokens(oasSessionTokens(fs.Session))
	}
	return out
}

// flowNextActions returns the machine-readable set of actions the client may
// take in the current step (§6).
func flowNextActions(f *domain.Flow) []string {
	if f.Status != domain.FlowStatusPending {
		return nil
	}
	var actions []string
	switch f.Step {
	case domain.FlowStepCollectCredentials:
		actions = []string{"submit"}
	case domain.FlowStepVerifyEmail, domain.FlowStepVerifyPhone:
		actions = []string{"verify_email"}
		if f.ActiveChallenge != nil {
			actions = append(actions, "resend")
		}
	case domain.FlowStepSetPassword:
		actions = []string{"set_password"}
	case domain.FlowStepMFARequired:
		actions = []string{"verify_mfa"}
	case domain.FlowStepAcceptConsents:
		actions = []string{"accept_consents"}
	case domain.FlowStepRequestAccess:
		actions = []string{"submit_access_request"}
	}
	return actions
}

// maskEmail returns a***@b.ru-style masked email (§5 rule 10).
func maskEmail(email string) string {
	at := strings.IndexByte(email, '@')
	if at < 0 {
		return "***"
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 1 {
		return "*" + domain
	}
	return string(local[0]) + "***" + domain
}

// maskPhone masks a phone number showing only the last 2 digits.
func maskPhone(phone string) string {
	if len(phone) <= 2 {
		return "***"
	}
	return "***" + phone[len(phone)-2:]
}

// WithCoreAuthFlows adds the CoreAuthFlowService to the Service, replacing
// the default CoreAuthService for the flow-related operations. The option merges
// the flow handler methods into the CoreAuth group using the composite pattern:
// CoreAuthService handles the non-flow ops; CoreAuthFlowService handles flows.
func WithCoreAuthFlows(flowDeps CoreAuthFlowDeps) Option {
	return func(s *Service) {
		s.CoreAuthHandler = &coreAuthComposite{
			CoreAuthHandler:     s.CoreAuthHandler,
			CoreAuthFlowService: NewCoreAuthFlowService(flowDeps),
		}
	}
}

// coreAuthComposite combines the existing CoreAuth handler (auth/password/email/…)
// with the flow handler. The flow methods win; everything else delegates.
type coreAuthComposite struct {
	oas.CoreAuthHandler
	*CoreAuthFlowService
}

// The six flow methods are served by CoreAuthFlowService; every other CoreAuth
// method is served by the embedded oas.CoreAuthHandler (typically *CoreAuthService).

func (c *coreAuthComposite) PostV1AuthFlows(ctx context.Context, req *oas.FlowCreateRequest, params oas.PostV1AuthFlowsParams) (*oas.FlowState, error) {
	return c.CoreAuthFlowService.PostV1AuthFlows(ctx, req, params)
}
func (c *coreAuthComposite) GetV1AuthFlowsByFlowToken(ctx context.Context, params oas.GetV1AuthFlowsByFlowTokenParams) (*oas.FlowState, error) {
	return c.CoreAuthFlowService.GetV1AuthFlowsByFlowToken(ctx, params)
}
func (c *coreAuthComposite) GetV1AuthFlowsCurrent(ctx context.Context, params oas.GetV1AuthFlowsCurrentParams) (*oas.FlowState, error) {
	return c.CoreAuthFlowService.GetV1AuthFlowsCurrent(ctx, params)
}
func (c *coreAuthComposite) PostV1AuthFlowsByFlowTokenSubmit(ctx context.Context, req *oas.FlowSubmitRequest, params oas.PostV1AuthFlowsByFlowTokenSubmitParams) (*oas.FlowState, error) {
	return c.CoreAuthFlowService.PostV1AuthFlowsByFlowTokenSubmit(ctx, req, params)
}
func (c *coreAuthComposite) PostV1AuthFlowsByFlowTokenResend(ctx context.Context, params oas.PostV1AuthFlowsByFlowTokenResendParams) (*oas.FlowState, error) {
	return c.CoreAuthFlowService.PostV1AuthFlowsByFlowTokenResend(ctx, params)
}
func (c *coreAuthComposite) DeleteV1AuthFlowsByFlowToken(ctx context.Context, params oas.DeleteV1AuthFlowsByFlowTokenParams) error {
	return c.CoreAuthFlowService.DeleteV1AuthFlowsByFlowToken(ctx, params)
}

// ConsentDocRef is re-exported from oas for convenience, since it's used in the mapper.
// (oas.ConsentDocRef is the generated type)
