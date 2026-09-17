package api

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

const securityPageSize = 30

// AccountSecurityStore owns every security transition and its transaction.
type AccountSecurityStore interface {
	Policy(ctx context.Context, scope domain.SecurityScope) (*domain.SecurityPolicy, error)
	SetPolicy(
		ctx context.Context,
		scope domain.SecurityScope,
		policy domain.SecurityPolicy) (*domain.SecurityPolicy, error)
	List(
		ctx context.Context,
		scope domain.SecurityScope,
		kind, cursor string,
		limit int) (SecurityPage, error)
	Incident(ctx context.Context, scope domain.SecurityScope, id string) (*domain.SecurityIncident, error)
	RegisterDevice(
		ctx context.Context, scope domain.SecurityScope, name string,
	) (*domain.SecurityDeviceRegistration, error)
	UpdateDevice(
		ctx context.Context,
		scope domain.SecurityScope,
		id, action, name, proof string) (*domain.SecurityDevice, error)
	Start(
		ctx context.Context,
		scope domain.SecurityScope,
		input domain.SecurityFlowInput,
		mode string) (*domain.SecurityFlowState, error)
	Flow(
		ctx context.Context,
		scope domain.SecurityScope,
		token string,
		input domain.SecurityFlowInput,
		mode string) (*domain.SecurityFlowState, error)
	Case(ctx context.Context, scope domain.SecurityScope, id string) (*domain.SecurityCase, error)
	DecideCase(
		ctx context.Context,
		scope domain.SecurityScope,
		id string,
		decision domain.SecurityCaseDecision) (*domain.SecurityCase, error)
	RetryDelivery(ctx context.Context, scope domain.SecurityScope, id string) (*domain.SecurityDelivery, error)
}

type SecurityPage struct {
	Data       any    `json:"data"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type AccountSecurityService struct {
	oas.UnimplementedHandler
	Store AccountSecurityStore
}

func NewAccountSecurityService(store AccountSecurityStore) *AccountSecurityService {
	return &AccountSecurityService{Store: store}
}

func WithAccountSecurity(h oas.AccountSecurityHandler) Option {
	return func(s *Service) { s.AccountSecurityHandler = h }
}

func securityResponse[T any](v any, err error) (*T, error) {
	if err != nil {
		return nil, fmt.Errorf("account security conversion: %w", err)
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("account security conversion: %w", err)
	}

	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("decode security result: %w", err)
	}

	return &out, nil
}

func securityInput[T any](v any) (T, error) {
	var out T

	b, err := json.Marshal(v)
	if err != nil {
		return out, fmt.Errorf("encode security input: %w", err)
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("decode security input: %w", err)
	}

	return out, nil
}

//nolint:nestif // Apply the distinct public, user, admin and recovery-decision authorization rules together.
func securityScope(ctx context.Context, projectID, mode string) (domain.SecurityScope, error) {
	s := domain.SecurityScope{ProjectID: projectID, Environment: EnvironmentFromContext(ctx)}
	if mode == "public" {
		return s, nil
	}

	p, err := requirePrincipal(ctx)
	if err != nil {
		return s, err
	}

	if mode == "admin" || mode == "decision" {
		if _, err := requireProjectAdmin(ctx, projectID); err != nil {
			return s, err
		}

		if mode == "decision" &&
			p.Kind != domain.PrincipalOperator &&
			!slices.Contains(p.Scopes, "security:recovery") {
			return s, domain.ErrForbidden.WithMessage("security:recovery permission is required")
		}

		s.ActorID = securityActorID(p)
	} else {
		if p.Kind != domain.PrincipalUser || p.ProjectID != projectID {
			return s, domain.ErrForbidden
		}

		s.AccountID, s.SessionID = p.AccountID, p.SessionID
		if s.Environment != "" && s.Environment != p.Environment {
			return s, domain.ErrForbidden
		}

		s.Environment = p.Environment
	}

	return s, nil
}

func (s *AccountSecurityService) ListSecurityIncidents(
	ctx context.Context,
	p oas.ListSecurityIncidentsParams,
) (*oas.SecurityIncidentList, error) {
	scope, err := securityScope(ctx, p.XClientID, "user")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityIncidentList](
		s.Store.List(
			ctx,
			scope,
			"incidents",
			p.Cursor.Or(""),
			p.Limit.Or(securityPageSize)))
}

func (s *AccountSecurityService) GetSecurityIncident(
	ctx context.Context,
	p oas.GetSecurityIncidentParams,
) (*oas.SecurityIncident, error) {
	scope, err := securityScope(ctx, p.XClientID, "user")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityIncident](s.Store.Incident(ctx, scope, p.IncidentID))
}

func (s *AccountSecurityService) ListSecurityDevices(
	ctx context.Context,
	p oas.ListSecurityDevicesParams,
) (*oas.SecurityDeviceList, error) {
	scope, err := securityScope(ctx, p.XClientID, "user")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityDeviceList](
		s.Store.List(
			ctx,
			scope,
			"devices",
			p.Cursor.Or(""),
			p.Limit.Or(securityPageSize)))
}

func (s *AccountSecurityService) RegisterSecurityDevice(
	ctx context.Context,
	req *oas.SecurityDeviceInput,
	p oas.RegisterSecurityDeviceParams,
) (*oas.SecurityDeviceRegistration, error) {
	scope, err := securityScope(ctx, p.XClientID, "user")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityDeviceRegistration](s.Store.RegisterDevice(ctx, scope, req.Name.Or("")))
}

func (s *AccountSecurityService) UpdateSecurityDevice(
	ctx context.Context,
	req *oas.SecurityDeviceInput,
	p oas.UpdateSecurityDeviceParams,
) (*oas.SecurityDevice, error) {
	scope, err := securityScope(ctx, p.XClientID, "user")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityDevice](
		s.Store.UpdateDevice(
			ctx,
			scope,
			p.DeviceID,
			string(req.Action.Or("")),
			req.Name.Or(""),
			req.ProofToken.Or("")))
}

func (s *AccountSecurityService) StartSecurityFlow(
	ctx context.Context,
	req *oas.SecurityFlowInput,
	p oas.StartSecurityFlowParams,
) (*oas.SecurityFlowState, error) {
	scope, err := securityScope(ctx, p.XClientID, "user")
	if err != nil {
		return nil, err
	}

	in, err := securityInput[domain.SecurityFlowInput](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityFlowState](s.Store.Start(ctx, scope, in, "review"))
}

func (s *AccountSecurityService) ExchangeSecurityContinuation(
	ctx context.Context,
	req *oas.SecurityFlowInput,
	p oas.ExchangeSecurityContinuationParams,
) (*oas.SecurityFlowState, error) {
	scope, err := securityScope(ctx, p.XClientID, "public")
	if err != nil {
		return nil, err
	}

	in, err := securityInput[domain.SecurityFlowInput](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityFlowState](s.Store.Start(ctx, scope, in, "exchange"))
}

func (s *AccountSecurityService) StartSecurityRecovery(
	ctx context.Context,
	req *oas.SecurityFlowInput,
	p oas.StartSecurityRecoveryParams,
) (*oas.SecurityFlowState, error) {
	scope, err := securityScope(ctx, p.XClientID, "public")
	if err != nil {
		return nil, err
	}

	in, err := securityInput[domain.SecurityFlowInput](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityFlowState](s.Store.Start(ctx, scope, in, "recovery"))
}

func (s *AccountSecurityService) GetSecurityFlow(
	ctx context.Context,
	p oas.GetSecurityFlowParams,
) (*oas.SecurityFlowState, error) {
	scope, err := securityScope(ctx, p.XClientID, "public")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityFlowState](
		s.Store.Flow(ctx, scope, p.XSecurityToken, domain.SecurityFlowInput{}, "get"))
}

func (s *AccountSecurityService) SubmitSecurityFlow(
	ctx context.Context,
	req *oas.SecurityFlowInput,
	p oas.SubmitSecurityFlowParams,
) (*oas.SecurityFlowState, error) {
	scope, err := securityScope(ctx, p.XClientID, "public")
	if err != nil {
		return nil, err
	}

	in, err := securityInput[domain.SecurityFlowInput](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityFlowState](s.Store.Flow(ctx, scope, p.XSecurityToken, in, "submit"))
}

func (s *AccountSecurityService) ResendSecurityFlow(
	ctx context.Context,
	p oas.ResendSecurityFlowParams,
) (*oas.SecurityFlowState, error) {
	scope, err := securityScope(ctx, p.XClientID, "public")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityFlowState](
		s.Store.Flow(
			ctx,
			scope,
			p.XSecurityToken,
			domain.SecurityFlowInput{},
			"resend"))
}

func (s *AccountSecurityService) GetSecurityPolicy(
	ctx context.Context,
	p oas.GetSecurityPolicyParams,
) (*oas.SecurityPolicy, error) {
	scope, err := securityScope(ctx, p.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityPolicy](s.Store.Policy(ctx, scope))
}

func (s *AccountSecurityService) PutSecurityPolicy(
	ctx context.Context,
	req *oas.SecurityPolicy,
	p oas.PutSecurityPolicyParams,
) (*oas.SecurityPolicy, error) {
	scope, err := securityScope(ctx, p.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	in, err := securityInput[domain.SecurityPolicy](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityPolicy](s.Store.SetPolicy(ctx, scope, in))
}

func (s *AccountSecurityService) AdminListSecurityIncidents(
	ctx context.Context,
	p oas.AdminListSecurityIncidentsParams,
) (*oas.SecurityIncidentList, error) {
	scope, err := securityScope(ctx, p.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityIncidentList](
		s.Store.List(
			ctx,
			scope,
			"incidents",
			p.Cursor.Or(""),
			p.Limit.Or(securityPageSize)))
}

func (s *AccountSecurityService) ListSecurityCases(
	ctx context.Context,
	p oas.ListSecurityCasesParams,
) (*oas.SecurityCaseList, error) {
	scope, err := securityScope(ctx, p.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityCaseList](
		s.Store.List(
			ctx,
			scope,
			"cases",
			p.Cursor.Or(""),
			p.Limit.Or(securityPageSize)))
}

func (s *AccountSecurityService) GetSecurityCase(
	ctx context.Context,
	p oas.GetSecurityCaseParams,
) (*oas.SecurityCase, error) {
	scope, err := securityScope(ctx, p.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityCase](s.Store.Case(ctx, scope, p.CaseID))
}

func (s *AccountSecurityService) DecideSecurityCase(
	ctx context.Context,
	req *oas.SecurityCaseDecision,
	p oas.DecideSecurityCaseParams,
) (*oas.SecurityCase, error) {
	scope, err := securityScope(ctx, p.ProjectID, "decision")
	if err != nil {
		return nil, err
	}

	in, err := securityInput[domain.SecurityCaseDecision](req)
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityCase](s.Store.DecideCase(ctx, scope, p.CaseID, in))
}

func (s *AccountSecurityService) ListSecurityDeliveries(
	ctx context.Context,
	p oas.ListSecurityDeliveriesParams,
) (*oas.SecurityDeliveryList, error) {
	scope, err := securityScope(ctx, p.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityDeliveryList](
		s.Store.List(
			ctx,
			scope,
			"deliveries",
			p.Cursor.Or(""),
			p.Limit.Or(securityPageSize)))
}

func (s *AccountSecurityService) RetrySecurityDelivery(
	ctx context.Context,
	p oas.RetrySecurityDeliveryParams,
) (*oas.SecurityDelivery, error) {
	scope, err := securityScope(ctx, p.ProjectID, "admin")
	if err != nil {
		return nil, err
	}

	return securityResponse[oas.SecurityDelivery](s.Store.RetryDelivery(ctx, scope, p.DeliveryID))
}

func securityActorID(p *domain.Principal) string {
	if p.CredentialID != "" {
		return p.CredentialID
	}

	if p.AccountID != "" {
		return p.AccountID
	}

	return string(p.Kind) + ":" + p.ClientID
}
