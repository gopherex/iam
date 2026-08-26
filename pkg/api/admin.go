// Code scaffolded for IAM handler groups.
//
// AdminService is pure orchestration: it holds aggregate-port interfaces (deps) and
// nothing else. It embeds oas.UnimplementedHandler so any operation it does not
// override returns not-implemented, and panics on every v1.0.0 operation until
// written. Each port method is atomic in its adapter — services never open a
// transaction.

package api

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-faster/jx"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type AdminUsers interface {
	List(ctx context.Context, projectID, environment string) ([]domain.Account, error)
	Get(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error)
	Create(ctx context.Context, cmd domain.RegisterCmd) (*domain.Account, error)
	Update(ctx context.Context, cmd domain.AdminUserUpdateCmd) (*domain.Account, error)
	Ban(ctx context.Context, projectID, environment, accountID string) error
	BanWith(ctx context.Context, cmd domain.AdminUserBanCmd) (*domain.Account, error)
	Unban(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error)
	Delete(ctx context.Context, projectID, environment, accountID string) error
	VerifyEmail(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error)
	VerifyPhone(ctx context.Context, projectID, environment, accountID string) (*domain.Account, error)
	SetPassword(ctx context.Context, cmd domain.AdminUserPasswordCmd) error
	Anonymize(ctx context.Context, cmd domain.AdminUserAnonymizeCmd) error
	Export(ctx context.Context, projectID, environment, accountID string) (jobID string, err error)
	Impersonate(ctx context.Context, cmd domain.AdminUserImpersonateCmd) (*domain.AdminImpersonation, error)
	ResetMFA(ctx context.Context, projectID, environment, accountID string, factorIDs []string) (removed int, err error)
	ListIdentities(ctx context.Context, projectID, environment, accountID string) ([]domain.Identity, error)
	DeleteIdentity(ctx context.Context, projectID, environment, accountID, identityID string) error
	ListSessions(ctx context.Context, projectID, environment, accountID string) ([]domain.Session, error)
	DeleteSession(ctx context.Context, projectID, environment, accountID, sessionID string) error
	RevokeSessions(ctx context.Context, cmd domain.AdminUserSessionsRevokeCmd) (revoked int, err error)
}

type AdminApps interface {
	List(ctx context.Context, projectID, environment string) ([]domain.AppClient, error)
	Create(ctx context.Context, cmd domain.AppClientCmd) (*domain.AppClient, error)
	Get(ctx context.Context, projectID, environment, appID string) (*domain.AppClient, error)
	Update(ctx context.Context, projectID, environment, appID string, patch map[string]any) (*domain.AppClient, error)
	Delete(ctx context.Context, projectID, environment, appID string) error
	AddSecret(ctx context.Context, projectID, environment, appID, name string) (*domain.AdminSecret, error)
	DeleteSecret(ctx context.Context, projectID, environment, appID, secretID string) error
	// Apply reconciles the environment's clients against a desired-state list in
	// one transaction, optionally pruning clients the list omits.
	Apply(ctx context.Context, cmd domain.AdminAppsApplyCmd) (*domain.AdminAppsApplyResult, error)
}

// AdminServiceAccounts is the machine-identity slice exposed to project admins.
type AdminServiceAccounts interface {
	List(ctx context.Context, projectID string) ([]domain.ServiceAccount, error)
	Get(ctx context.Context, projectID, saID string) (*domain.ServiceAccount, error)
	Create(ctx context.Context, cmd domain.ServiceAccountCmd) (*domain.ServiceAccount, error)
	Update(ctx context.Context, cmd domain.AdminServiceAccountUpdateCmd) (*domain.ServiceAccount, error)
	Delete(ctx context.Context, projectID, saID string) error
	AddSecret(ctx context.Context, cmd domain.AdminServiceAccountSecretCmd) (*domain.AdminSecret, error)
	DeleteSecret(ctx context.Context, projectID, saID, secretID string) error
}

// AdminAPIKeys is the project API-key administration slice.
type AdminAPIKeys interface {
	List(ctx context.Context, projectID string) ([]domain.APIKey, error)
	Create(ctx context.Context, cmd domain.AdminAPIKeyCmd) (*domain.AdminAPIKeySecret, error)
	Update(ctx context.Context, cmd domain.AdminAPIKeyUpdateCmd) (*domain.APIKey, error)
	Delete(ctx context.Context, projectID, keyID string) error
	Rotate(ctx context.Context, projectID, keyID string) (*domain.AdminAPIKeySecret, error)
}

// AdminConnections is the federation (SSO connections + domains) admin slice.
type AdminConnections interface {
	List(ctx context.Context, projectID string) ([]domain.Connection, error)
	Get(ctx context.Context, projectID, connID string) (*domain.Connection, error)
	Create(ctx context.Context, cmd domain.AdminConnectionCmd) (*domain.Connection, error)
	Update(ctx context.Context, projectID, connID string, patch map[string]any) (*domain.Connection, error)
	Delete(ctx context.Context, projectID, connID string) error
	ListDomains(ctx context.Context, projectID string) ([]domain.Domain, error)
	CreateDomain(ctx context.Context, cmd domain.AdminDomainCmd) (*domain.AdminDomainRegistration, error)
	DeleteDomain(ctx context.Context, projectID, domainID string) error
	VerifyDomain(ctx context.Context, projectID, domainID string) (*domain.Domain, error)
}

// AdminConfig is the project-configuration slice: auth / password-policy /
// session-policy / consent documents plus feature flags and i18n bundles. Each
// document is carried opaquely as a domain.AdminConfigDoc the adapter validates
// and persists.
// AdminRoles is the role-assignment slice. Roles are IAM-owned labels scoped to
// a project environment; they are the only source of the OIDC `groups` claim.
type AdminRoles interface {
	ListRoles(ctx context.Context, cmd domain.AdminUserRolesCmd) ([]string, error)
	SetRoles(ctx context.Context, cmd domain.AdminUserRolesSetCmd) ([]string, error)
}

type AdminConfig interface {
	// GetConfigBundle reads every configuration document at once; ApplyConfig
	// writes a whole bundle atomically. Together they are the desired-state
	// (plan/apply) surface over the per-document endpoints below.
	GetConfigBundle(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigBundle, error)
	ApplyConfig(ctx context.Context, cmd domain.AdminConfigApplyCmd) (*domain.AdminConfigApplyResult, error)

	GetAuthConfig(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error)
	UpdateAuthConfig(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error)
	GetPasswordPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error)
	UpdatePasswordPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error)
	GetSessionPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error)
	UpdateSessionPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error)
	GetRateLimits(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error)
	UpdateRateLimits(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error)
	GetMfaPolicy(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error)
	UpdateMfaPolicy(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error)
	GetConsent(ctx context.Context, cmd domain.AdminConfigGetCmd) (domain.AdminConfigDoc, error)
	PutConsent(ctx context.Context, cmd domain.AdminConfigUpdateCmd) (domain.AdminConfigDoc, error)

	GetFeatures(ctx context.Context, cmd domain.AdminConfigGetCmd) (map[string]bool, error)
	PutFeatures(ctx context.Context, cmd domain.AdminFeaturesUpdateCmd) (map[string]bool, error)

	GetI18n(ctx context.Context, cmd domain.AdminConfigGetCmd, locale string) (map[string]jx.Raw, error)
	PutI18n(ctx context.Context, cmd domain.AdminI18nUpdateCmd) (map[string]jx.Raw, error)

	// Email / SMS providers.
	ListEmailProviders(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminProvider, error)
	CreateEmailProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error)
	UpdateEmailProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error)
	DeleteEmailProvider(ctx context.Context, cmd domain.AdminProviderDeleteCmd) error
	ListSmsProviders(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminProvider, error)
	CreateSmsProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error)
	UpdateSmsProvider(ctx context.Context, cmd domain.AdminProviderCmd) (*domain.AdminProvider, error)
	DeleteSmsProvider(ctx context.Context, cmd domain.AdminProviderDeleteCmd) error
	ListOAuthProviders(ctx context.Context, projectID string) ([]domain.AdminOAuthProvider, error)
	CreateOAuthProvider(ctx context.Context, projectID string, p domain.AdminOAuthProvider) (domain.AdminOAuthProvider, error)
	UpdateOAuthProvider(ctx context.Context, projectID, id string, p domain.AdminOAuthProvider) (domain.AdminOAuthProvider, error)
	DeleteOAuthProvider(ctx context.Context, projectID, id string) error
	GetRetentionPolicy(ctx context.Context, projectID string) ([]byte, error)
	PutRetentionPolicy(ctx context.Context, projectID string, raw []byte) error

	// Email templates.
	ListEmailTemplates(ctx context.Context, cmd domain.AdminConfigGetCmd) (map[string]jx.Raw, error)
	UpdateEmailTemplate(ctx context.Context, cmd domain.AdminTemplateUpdateCmd) (map[string]jx.Raw, error)
	PreviewEmailTemplate(ctx context.Context, cmd domain.AdminTemplatePreviewCmd) (*domain.AdminTemplatePreview, error)
	SendTestEmail(ctx context.Context, cmd domain.AdminTemplateSendTestCmd) error
	SendTestSMS(ctx context.Context, cmd domain.AdminTemplateSendTestCmd) error
}

// AdminKeys is the signing-key (JWKS) + token-profile administration slice.
type AdminKeys interface {
	ListSigningKeys(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminSigningKey, error)
	DeleteSigningKey(ctx context.Context, cmd domain.AdminConfigGetCmd, kid string) error
	RotateSigningKeys(ctx context.Context, cmd domain.AdminJWKSRotateCmd) (*domain.AdminSigningKey, error)
	ActivateSigningKey(ctx context.Context, cmd domain.AdminConfigGetCmd, kid string) (*domain.AdminSigningKey, error)

	ListTokenProfiles(ctx context.Context, cmd domain.AdminConfigGetCmd) ([]domain.AdminTokenProfile, error)
	CreateTokenProfile(ctx context.Context, cmd domain.AdminTokenProfileCmd) (*domain.AdminTokenProfile, error)
	UpdateTokenProfile(ctx context.Context, cmd domain.AdminTokenProfileCmd) (*domain.AdminTokenProfile, error)
	DeleteTokenProfile(ctx context.Context, cmd domain.AdminConfigGetCmd, profileID string) error
	PreviewTokenProfile(ctx context.Context, cmd domain.AdminTokenProfilePreviewCmd) (map[string]jx.Raw, error)
}

// AdminAccessRequests is the access-request moderation slice.
type AdminAccessRequests interface {
	List(ctx context.Context, cmd domain.AdminAccessRequestListCmd) (*domain.AdminAccessRequestPage, error)
	Approve(ctx context.Context, cmd domain.AdminAccessRequestDecisionCmd) (map[string]jx.Raw, error)
	Deny(ctx context.Context, cmd domain.AdminAccessRequestDecisionCmd) (*domain.CoreAuthAccessRequest, error)
}

// AdminInvites is the project invitation administration slice.
type AdminInvites interface {
	Create(ctx context.Context, cmd domain.InviteCreateCmd) (*domain.InviteCreated, error)
	List(ctx context.Context, cmd domain.InviteListCmd) ([]domain.Invite, error)
	Revoke(ctx context.Context, cmd domain.InviteRevokeCmd) error
}

// AdminWebhooks is the project-scoped public event delivery subsystem.
type AdminWebhooks interface {
	List(ctx context.Context, cmd domain.WebhookListCmd) ([]domain.Webhook, string, bool, error)
	Get(ctx context.Context, projectID, environment, id string) (*domain.Webhook, error)
	Create(ctx context.Context, cmd domain.WebhookCreateCmd) (*domain.Webhook, string, error)
	Update(ctx context.Context, cmd domain.WebhookUpdateCmd) (*domain.Webhook, error)
	Delete(ctx context.Context, projectID, environment, id string) error
	RotateSecret(ctx context.Context, projectID, environment, id string) (string, error)
	Test(ctx context.Context, projectID, environment, webhookID, eventType string) (*domain.WebhookDelivery, error)
	ListDeliveries(ctx context.Context, cmd domain.WebhookDeliveryListCmd) ([]domain.WebhookDelivery, error)
	RetryDelivery(ctx context.Context, projectID, environment, deliveryID string) (*domain.WebhookDelivery, error)
	ListEvents(ctx context.Context, cmd domain.WebhookEventListCmd) (*domain.WebhookEventPage, error)
	ReplayEvent(ctx context.Context, projectID, environment, eventID, webhookID string) ([]domain.WebhookDelivery, error)
}

// AdminDeps are the per-project administration ports.
type AdminDeps struct {
	Users           AdminUsers
	Apps            AdminApps
	ServiceAccounts AdminServiceAccounts
	APIKeys         AdminAPIKeys
	Connections     AdminConnections
	Config          AdminConfig
	Roles           AdminRoles
	Keys            AdminKeys
	AccessRequests  AdminAccessRequests
	Invites         AdminInvites
	Webhooks        AdminWebhooks
	Grants          AdminGrants
	Audit           AdminAudit
	Jobs            AdminJobs
	Hooks           AdminHooks
	Risk            AdminRisk
	TestMode        AdminTestMode
}

// AdminTestMode backs the /v1/test/* endpoints (seed / reset / clock / messages)
// used by the SDK and test harness. Optional — nil disables test mode.
type AdminTestMode interface {
	Seed(ctx context.Context, projectID, env string, spec map[string]any) error
	Reset(ctx context.Context, projectID, env string) (int64, error)
	Clock(ctx context.Context, projectID, env string, advanceSeconds int, reset bool) error
	Messages(ctx context.Context, projectID, env, channel, to string) ([]map[string]any, error)
}

// AdminRisk manages declarative risk rules, risk events, and manual rate-limit
// blocks.
type AdminRisk interface {
	ListRules(ctx context.Context, projectID string) ([]domain.AdminRiskRule, error)
	CreateRule(ctx context.Context, projectID string, r domain.AdminRiskRule) (domain.AdminRiskRule, error)
	UpdateRule(ctx context.Context, projectID, id string, r domain.AdminRiskRule) (domain.AdminRiskRule, error)
	DeleteRule(ctx context.Context, projectID, id string) error
	CreateBlock(ctx context.Context, projectID, env string, b domain.AdminBlock) (domain.AdminBlock, error)
	DeleteBlock(ctx context.Context, projectID, id string) error
	ListEvents(ctx context.Context, projectID string) ([]map[string]any, error)
}

// AdminHooks manages blocking auth hooks (signed HTTP callbacks at auth decision
// points) and test-fires them.
type AdminHooks interface {
	List(ctx context.Context, projectID string) ([]domain.AdminHook, error)
	Get(ctx context.Context, projectID, id string) (*domain.AdminHook, error)
	Create(ctx context.Context, projectID string, h domain.AdminHook) (domain.AdminHook, error)
	Update(ctx context.Context, projectID, id string, h domain.AdminHook) (domain.AdminHook, error)
	Delete(ctx context.Context, projectID, id string) error
	Test(ctx context.Context, projectID, id string, payload []byte) (status int, body string, durationMs int, err error)
}

// AdminJobs manages async background jobs (bulk import, exports) and verifies
// imported password hashes.
type AdminJobs interface {
	List(ctx context.Context, projectID string, limit int) ([]domain.AdminJob, error)
	Get(ctx context.Context, projectID, id string) (*domain.AdminJob, error)
	Cancel(ctx context.Context, projectID, id string) error
	CreateImportUsers(ctx context.Context, projectID string, users []map[string]jx.Raw, format string, sendInvites bool) (jobID string, status string, err error)
	VerifyPasswordHash(hash, password, format string) (bool, error)
}

// AdminAudit reads the tenant audit log and enqueues export jobs.
type AdminAudit interface {
	List(ctx context.Context, cmd domain.AuditLogListCmd) ([]domain.AuditLogEntry, string, bool, error)
	Get(ctx context.Context, projectID, id string) (*domain.AuditLogEntry, error)
	CreateExport(ctx context.Context, cmd domain.AuditExportCmd) (jobID string, status string, err error)
}

// AdminGrants is the admin view over a user's OAuth consent grants. It reuses the
// runtime grants store (grants are keyed by the account's unique id).
type AdminGrants interface {
	ListGrants(ctx context.Context, accountID string) ([]domain.Grant, error)
	RevokeGrant(ctx context.Context, accountID, grantID string) error
}

// AdminService implements the AdminHandler slice of oas.Handler.
type AdminService struct {
	oas.UnimplementedHandler
	deps AdminDeps
}

// NewAdminService builds the Admin service from its dependencies.
func NewAdminService(deps AdminDeps) *AdminService { return &AdminService{deps: deps} }

var _ oas.Handler = (*AdminService)(nil)

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminWebhooksById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminWebhooksByIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Webhooks.Delete(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.ID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminWebhooks(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminWebhooksParams) (*oas.GetV1ProjectsByProjectIdAdminWebhooksOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	webhooks, next, hasMore, err := s.deps.Webhooks.List(ctx, domain.WebhookListCmd{
		ProjectID: params.ProjectID, Environment: params.XEnvironment.Or("live"),
		Cursor: params.Cursor.Or(""), Limit: params.Limit.Or(50),
	})
	if err != nil {
		return nil, err
	}

	data := make([]oas.Webhook, 0, len(webhooks))
	for i := range webhooks {
		data = append(data, oasWebhook(&webhooks[i]))
	}

	out := &oas.GetV1ProjectsByProjectIdAdminWebhooksOK{Data: data, HasMore: oas.NewOptBool(hasMore)}
	if next != "" {
		out.NextCursor = oas.NewOptNilString(next)
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminWebhooksById(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminWebhooksByIdParams) (*oas.GetV1ProjectsByProjectIdAdminWebhooksByIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	webhook, err := s.deps.Webhooks.Get(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.ID)
	if err != nil {
		return nil, err
	}

	return &oas.GetV1ProjectsByProjectIdAdminWebhooksByIdOK{Webhook: oas.NewOptWebhook(oasWebhook(webhook))}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminWebhooks(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminWebhooksReq, params oas.PostV1ProjectsByProjectIdAdminWebhooksParams) (*oas.PostV1ProjectsByProjectIdAdminWebhooksCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	webhook, secret, err := s.deps.Webhooks.Create(ctx, domain.WebhookCreateCmd{
		ProjectID: params.ProjectID, Environment: params.XEnvironment.Or("live"),
		URL: req.URL, Events: req.Events, Description: req.Description.Or(""),
		Enabled: req.Enabled.Or(true), IdempotencyKey: params.IdempotencyKey.Or(""),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminWebhooksCreated{
		Webhook: oas.NewOptWebhook(oasWebhook(webhook)), SigningSecret: oas.NewOptString(secret),
	}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminWebhooksById(ctx context.Context, req *oas.PatchV1ProjectsByProjectIdAdminWebhooksByIdReq, params oas.PatchV1ProjectsByProjectIdAdminWebhooksByIdParams) (*oas.PatchV1ProjectsByProjectIdAdminWebhooksByIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd, err := webhookUpdateCmd(params.ProjectID, params.XEnvironment.Or("live"), params.ID, req)
	if err != nil {
		return nil, err
	}

	webhook, err := s.deps.Webhooks.Update(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminWebhooksByIdOK{Webhook: oas.NewOptWebhook(oasWebhook(webhook))}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminWebhooksByIdRotateSecret(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminWebhooksByIdRotateSecretParams) (*oas.PostV1ProjectsByProjectIdAdminWebhooksByIdRotateSecretOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	secret, err := s.deps.Webhooks.RotateSecret(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.ID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminWebhooksByIdRotateSecretOK{SigningSecret: oas.NewOptString(secret)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminWebhooksByIdTest(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminWebhooksByIdTestReq, params oas.PostV1ProjectsByProjectIdAdminWebhooksByIdTestParams) (*oas.PostV1ProjectsByProjectIdAdminWebhooksByIdTestOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	delivery, err := s.deps.Webhooks.Test(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.ID, req.EventType.Or(""))
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminWebhooksByIdTestOK{Delivery: oasWebhookDelivery(delivery)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminWebhookDeliveries(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminWebhookDeliveriesParams) (*oas.GetV1ProjectsByProjectIdAdminWebhookDeliveriesOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	deliveries, err := s.deps.Webhooks.ListDeliveries(ctx, domain.WebhookDeliveryListCmd{
		ProjectID: params.ProjectID, Environment: params.XEnvironment.Or("live"),
		WebhookID: params.WebhookID.Or(""), Status: string(params.Status.Or("")), Limit: 100,
	})
	if err != nil {
		return nil, err
	}

	data := make([]oas.WebhookDelivery, 0, len(deliveries))
	for i := range deliveries {
		data = append(data, oasWebhookDelivery(&deliveries[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminWebhookDeliveriesOK{Data: data}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminWebhookDeliveriesByDeliveryIdRetry(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminWebhookDeliveriesByDeliveryIdRetryParams) (*oas.PostV1ProjectsByProjectIdAdminWebhookDeliveriesByDeliveryIdRetryOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	delivery, err := s.deps.Webhooks.RetryDelivery(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.DeliveryID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminWebhookDeliveriesByDeliveryIdRetryOK{Delivery: oasWebhookDelivery(delivery)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminEvents(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminEventsParams) (*oas.GetV1ProjectsByProjectIdAdminEventsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	page, err := s.deps.Webhooks.ListEvents(ctx, domain.WebhookEventListCmd{
		ProjectID: params.ProjectID, Environment: params.XEnvironment.Or("live"),
		Type: params.Type.Or(""), UserID: params.UserID.Or(""), Cursor: params.Cursor.Or(""), Limit: params.Limit.Or(50),
	})
	if err != nil {
		return nil, err
	}

	data := make([]oas.Event, 0, len(page.Data))
	for _, event := range page.Data {
		data = append(data, oasPublicEvent(event))
	}

	out := &oas.GetV1ProjectsByProjectIdAdminEventsOK{Data: data, HasMore: oas.NewOptBool(page.HasMore)}
	if page.NextCursor != "" {
		out.NextCursor = oas.NewOptNilString(page.NextCursor)
	}

	return out, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEventsByEventIdReplay(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminEventsByEventIdReplayReq, params oas.PostV1ProjectsByProjectIdAdminEventsByEventIdReplayParams) (*oas.PostV1ProjectsByProjectIdAdminEventsByEventIdReplayOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	webhookID := ""
	if value, ok := req.Get(); ok {
		webhookID = value.WebhookID.Or("")
	}

	deliveries, err := s.deps.Webhooks.ReplayEvent(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.EventID, webhookID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.WebhookDelivery, 0, len(deliveries))
	for i := range deliveries {
		data = append(data, oasWebhookDelivery(&deliveries[i]))
	}

	return &oas.PostV1ProjectsByProjectIdAdminEventsByEventIdReplayOK{Deliveries: data}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Apps.Delete(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.AppID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Apps.DeleteSecret(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.AppID, params.SecretID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminEmailProvidersById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminEmailProvidersByIdParams) (r *oas.Ok, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Config.DeleteEmailProvider(ctx, domain.AdminProviderDeleteCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          params.ID,
	}); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminJwksByKeyId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminJwksByKeyIdParams) (r *oas.Ok, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Keys.DeleteSigningKey(ctx, adminCfg(params.ProjectID, params.XEnvironment), params.KeyID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminSmsProvidersById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminSmsProvidersByIdParams) (r *oas.Ok, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Config.DeleteSmsProvider(ctx, domain.AdminProviderDeleteCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          params.ID,
	}); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminTokenProfilesById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminTokenProfilesByIdParams) (r *oas.Ok, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Keys.DeleteTokenProfile(ctx, adminCfg(params.ProjectID, params.XEnvironment), params.ID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Users.Delete(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Users.DeleteIdentity(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID, params.IdentityID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Users.DeleteSession(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID, params.SessionID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAccessRequests(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAccessRequestsParams) (r *oas.GetV1ProjectsByProjectIdAdminAccessRequestsOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	page, err := s.deps.AccessRequests.List(ctx, domain.AdminAccessRequestListCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Status:      params.Status.Or(""),
		Cursor:      params.Cursor.Or(""),
	})
	if err != nil {
		return nil, err
	}

	data := make([]oas.AccessRequest, 0, len(page.Items))
	for i := range page.Items {
		data = append(data, oasAdminAccessRequest(&page.Items[i]))
	}

	out := &oas.GetV1ProjectsByProjectIdAdminAccessRequestsOK{
		Data:    data,
		HasMore: oas.NewOptBool(page.HasMore),
	}
	if page.NextCursor != "" {
		out.NextCursor = oas.NewOptNilString(page.NextCursor)
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminApps(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAppsParams) (*oas.GetV1ProjectsByProjectIdAdminAppsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	apps, err := s.deps.Apps.List(ctx, params.ProjectID, params.XEnvironment.Or("live"))
	if err != nil {
		return nil, err
	}

	data := make([]oas.AppClient, 0, len(apps))
	for i := range apps {
		data = append(data, oasAppClient(&apps[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminAppsOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAppsByAppIdParams) (*oas.GetV1ProjectsByProjectIdAdminAppsByAppIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	app, err := s.deps.Apps.Get(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.AppID)
	if err != nil {
		return nil, err
	}

	return &oas.GetV1ProjectsByProjectIdAdminAppsByAppIdOK{
		App: oas.NewOptAppClient(oasAppClient(app)),
	}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigAuth(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigAuthParams) (r *oas.AuthConfig, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.GetAuthConfig(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	out := &oas.AuthConfig{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigPasswordPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigPasswordPolicyParams) (r *oas.PasswordPolicy, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.GetPasswordPolicy(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	out := &oas.PasswordPolicy{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigSessionPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigSessionPolicyParams) (r *oas.SessionPolicy, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.GetSessionPolicy(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	out := &oas.SessionPolicy{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConsents(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConsentsParams) (r *oas.ConsentConfig, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.GetConsent(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	out := &oas.ConsentConfig{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminEmailProviders(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminEmailProvidersParams) (r *oas.GetV1ProjectsByProjectIdAdminEmailProvidersOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	provs, err := s.deps.Config.ListEmailProviders(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	data := make([]oas.EmailProvider, 0, len(provs))
	for i := range provs {
		data = append(data, oasAdminEmailProvider(&provs[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminEmailProvidersOK{Data: data}, nil
}

func oasAdminOAuthProvider(p domain.AdminOAuthProvider) oas.OAuthProviderConfig {
	out := oas.OAuthProviderConfig{
		Provider: p.Provider,
		Scopes:   p.Scopes,
		Enabled:  oas.NewOptBool(p.Enabled),
	}
	if p.ID != "" {
		out.ID = oas.NewOptString(p.ID)
	}

	if p.ClientID != "" {
		out.ClientID = oas.NewOptString(p.ClientID)
	}
	// ClientSecret is write-only; never echoed back.
	return out
}

func adminOAuthProviderFromReq(req *oas.OAuthProviderConfig) domain.AdminOAuthProvider {
	return domain.AdminOAuthProvider{
		ID:           req.ID.Or(""),
		Provider:     req.Provider,
		ClientID:     req.ClientID.Or(""),
		ClientSecret: req.ClientSecret.Or(""),
		Scopes:       req.Scopes,
		Enabled:      req.Enabled.Or(false),
	}
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminOauthProviders(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminOauthProvidersParams) (*oas.GetV1ProjectsByProjectIdAdminOauthProvidersOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	provs, err := s.deps.Config.ListOAuthProviders(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.OAuthProviderConfig, 0, len(provs))
	for i := range provs {
		data = append(data, oasAdminOAuthProvider(provs[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminOauthProvidersOK{Data: data}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminOauthProviders(ctx context.Context, req *oas.OAuthProviderConfig, params oas.PostV1ProjectsByProjectIdAdminOauthProvidersParams) (*oas.OAuthProviderConfig, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prov, err := s.deps.Config.CreateOAuthProvider(ctx, params.ProjectID, adminOAuthProviderFromReq(req))
	if err != nil {
		return nil, err
	}

	out := oasAdminOAuthProvider(prov)

	return &out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminOauthProvidersById(ctx context.Context, req *oas.OAuthProviderConfig, params oas.PatchV1ProjectsByProjectIdAdminOauthProvidersByIdParams) (*oas.OAuthProviderConfig, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prov, err := s.deps.Config.UpdateOAuthProvider(ctx, params.ProjectID, params.ID, adminOAuthProviderFromReq(req))
	if err != nil {
		return nil, err
	}

	out := oasAdminOAuthProvider(prov)

	return &out, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminOauthProvidersById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminOauthProvidersByIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Config.DeleteOAuthProvider(ctx, params.ProjectID, params.ID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func oasAuditLog(e domain.AuditLogEntry) oas.AuditLog {
	out := oas.AuditLog{
		ID:   oas.NewOptString(e.ID),
		Type: oas.NewOptString(e.Type),
		At:   oas.NewOptTimestamp(oas.Timestamp(e.At)),
	}
	if e.ActorID != "" {
		out.ActorID = oas.NewOptNilString(e.ActorID)
	}

	if e.TargetID != "" {
		out.TargetID = oas.NewOptNilString(e.TargetID)
	}

	if e.Data != nil {
		out.Data = oas.NewOptAuditLogData(oas.AuditLogData(e.Data))
	}

	return out
}

// adminTestGate authorizes a /v1/test/* call: an authenticated principal is
// required and the environment must be non-live — test mode must never touch
// live data. Returns the caller's project and the resolved (non-live) env.
func (s *AdminService) adminTestGate(ctx context.Context, env string) (string, string, error) {
	if s.deps.TestMode == nil {
		return "", "", domain.ErrNotImplemented
	}

	p, err := requirePrincipal(ctx)
	if err != nil {
		return "", "", err
	}
	// Test mode (seed / reset-wipe / clock / messages) is a privileged
	// administrative capability, NEVER an end-user one: an ordinary user or
	// OAuth-client token must not be able to wipe or reseed an environment.
	//nolint:exhaustive // an allowlist, not an oversight: a future
	// PrincipalKind is meant to fall into the default and be rejected until
	// someone deliberately adds it here.
	switch p.Kind {
	case domain.PrincipalAdmin, domain.PrincipalOperator, domain.PrincipalService:
	default:
		return "", "", domain.ErrForbidden.WithMessage("test mode requires an admin, operator or service credential")
	}
	// Normalize before the live comparison so whitespace/case variants
	// (" live", "LIVE") cannot slip past the guard.
	env = strings.ToLower(strings.TrimSpace(env))
	if env == "" || env == "live" {
		return "", "", domain.ErrForbidden.WithMessage("test mode is not available in the live environment")
	}

	if p.ProjectID == "" {
		return "", "", domain.ErrForbidden.WithMessage("test mode requires a project-scoped credential")
	}

	return p.ProjectID, env, nil
}

func (s *AdminService) PostV1TestSeed(ctx context.Context, req oas.PostV1TestSeedReq, params oas.PostV1TestSeedParams) (*oas.Ok, error) {
	projectID, env, err := s.adminTestGate(ctx, params.XEnvironment.Or(""))
	if err != nil {
		return nil, err
	}

	rawMap := make(map[string]json.RawMessage, len(req))
	for k, v := range req {
		rawMap[k] = json.RawMessage(v)
	}

	blob, _ := json.Marshal(rawMap)

	spec := map[string]any{}
	_ = json.Unmarshal(blob, &spec)

	if err := s.deps.TestMode.Seed(ctx, projectID, env, spec); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1TestReset(ctx context.Context, params oas.PostV1TestResetParams) (oas.PostV1TestResetOK, error) {
	projectID, env, err := s.adminTestGate(ctx, params.XEnvironment.Or(""))
	if err != nil {
		return nil, err
	}

	deleted, err := s.deps.TestMode.Reset(ctx, projectID, env)
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(map[string]any{"ok": true, "deleted": deleted})

	out := oas.PostV1TestResetOK{}
	_ = out.UnmarshalJSON(raw)

	return out, nil
}

func (s *AdminService) PostV1TestClock(ctx context.Context, req *oas.PostV1TestClockReq, params oas.PostV1TestClockParams) (*oas.Ok, error) {
	projectID, env, err := s.adminTestGate(ctx, params.XEnvironment.Or(""))
	if err != nil {
		return nil, err
	}

	if err := s.deps.TestMode.Clock(ctx, projectID, env, req.AdvanceSeconds.Or(0), req.Reset.Or(false)); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) GetV1TestMessages(ctx context.Context, params oas.GetV1TestMessagesParams) (oas.GetV1TestMessagesOK, error) {
	projectID, env, err := s.adminTestGate(ctx, params.XEnvironment.Or(""))
	if err != nil {
		return nil, err
	}

	msgs, err := s.deps.TestMode.Messages(ctx, projectID, env, params.Channel.Or(""), params.To.Or(""))
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(map[string]any{"data": msgs})

	out := oas.GetV1TestMessagesOK{}
	_ = out.UnmarshalJSON(raw)

	return out, nil
}

func oasRiskRule(r domain.AdminRiskRule) oas.RiskRule {
	signal := r.EffectiveSignal()
	out := oas.RiskRule{
		ID:        oas.NewOptString(r.ID),
		Name:      r.Name,
		Condition: oas.NewOptString(signal),
		Enabled:   oas.NewOptBool(r.Enabled),
	}

	var sig oas.RiskRuleSignal
	if err := sig.UnmarshalText([]byte(signal)); err == nil {
		out.Signal = oas.NewOptRiskRuleSignal(sig)
	}

	var act oas.RiskRuleAction
	if err := act.UnmarshalText([]byte(r.Action)); err == nil {
		out.Action = oas.NewOptRiskRuleAction(act)
	}

	return out
}

func riskRuleFromReq(req *oas.RiskRule) domain.AdminRiskRule {
	return domain.AdminRiskRule{
		ID:        req.ID.Or(""),
		Name:      req.Name,
		Signal:    string(req.Signal.Or("")),
		Condition: req.Condition.Or(""), //nolint:staticcheck // the released alias, still accepted.
		Action:    string(req.Action.Or("")),
		Enabled:   req.Enabled.Or(true),
	}
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminRiskRules(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminRiskRulesParams) (*oas.GetV1ProjectsByProjectIdAdminRiskRulesOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	rules, err := s.deps.Risk.ListRules(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.RiskRule, 0, len(rules))
	for i := range rules {
		data = append(data, oasRiskRule(rules[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminRiskRulesOK{Data: data, HasMore: oas.NewOptBool(false)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminRiskRules(ctx context.Context, req *oas.RiskRule, params oas.PostV1ProjectsByProjectIdAdminRiskRulesParams) (*oas.RiskRule, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	created, err := s.deps.Risk.CreateRule(ctx, params.ProjectID, riskRuleFromReq(req))
	if err != nil {
		return nil, err
	}

	out := oasRiskRule(created)

	return &out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminRiskRulesByRuleId(ctx context.Context, req *oas.RiskRule, params oas.PatchV1ProjectsByProjectIdAdminRiskRulesByRuleIdParams) (*oas.RiskRule, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	updated, err := s.deps.Risk.UpdateRule(ctx, params.ProjectID, params.RuleID, riskRuleFromReq(req))
	if err != nil {
		return nil, err
	}

	out := oasRiskRule(updated)

	return &out, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminRiskRulesByRuleId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminRiskRulesByRuleIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Risk.DeleteRule(ctx, params.ProjectID, params.RuleID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminRiskEvents(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminRiskEventsParams) (oas.GetV1ProjectsByProjectIdAdminRiskEventsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	events, err := s.deps.Risk.ListEvents(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(map[string]any{"data": events, "has_more": false})

	out := oas.GetV1ProjectsByProjectIdAdminRiskEventsOK{}
	_ = out.UnmarshalJSON(raw)

	return out, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminRateLimitBlocks(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminRateLimitBlocksReq, params oas.PostV1ProjectsByProjectIdAdminRateLimitBlocksParams) (oas.PostV1ProjectsByProjectIdAdminRateLimitBlocksOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	b := domain.AdminBlock{Type: string(req.Type), Value: req.Value, Reason: req.Reason.Or("")}
	if v, ok := req.ExpiresAt.Get(); ok {
		b.ExpiresAt = v
	}

	created, err := s.deps.Risk.CreateBlock(ctx, params.ProjectID, params.XEnvironment.Or(""), b)
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(map[string]any{"id": created.ID, "type": created.Type, "value": created.Value})

	out := oas.PostV1ProjectsByProjectIdAdminRateLimitBlocksOK{}
	_ = out.UnmarshalJSON(raw)

	return out, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminRateLimitBlocksByBlockId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminRateLimitBlocksByBlockIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Risk.DeleteBlock(ctx, params.ProjectID, params.BlockID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func oasHook(h domain.AdminHook) oas.Hook {
	out := oas.Hook{
		ID:        oas.NewOptString(h.ID),
		URL:       oas.NewOptString(h.URL),
		TimeoutMs: oas.NewOptInt(h.TimeoutMs),
		Enabled:   oas.NewOptBool(h.Enabled),
	}

	var ht oas.HookType
	if err := ht.UnmarshalText([]byte(h.Type)); err == nil {
		out.Type = oas.NewOptHookType(ht)
	}

	return out
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminHooks(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminHooksParams) (*oas.GetV1ProjectsByProjectIdAdminHooksOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	hooks, err := s.deps.Hooks.List(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.Hook, 0, len(hooks))
	for i := range hooks {
		data = append(data, oasHook(hooks[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminHooksOK{Data: data, HasMore: oas.NewOptBool(false)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminHooks(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminHooksReq, params oas.PostV1ProjectsByProjectIdAdminHooksParams) (*oas.PostV1ProjectsByProjectIdAdminHooksCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	created, err := s.deps.Hooks.Create(ctx, params.ProjectID, domain.AdminHook{
		Type: req.Type, URL: req.URL, TimeoutMs: req.TimeoutMs.Or(0), Enabled: req.Enabled.Or(true),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminHooksCreated{
		Hook:          oas.NewOptHook(oasHook(created)),
		SigningSecret: oas.NewOptString(created.SigningSecret),
	}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminHooksById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminHooksByIdReq, params oas.PatchV1ProjectsByProjectIdAdminHooksByIdParams) (*oas.PatchV1ProjectsByProjectIdAdminHooksByIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cur, err := s.deps.Hooks.Get(ctx, params.ProjectID, params.ID)
	if err != nil {
		return nil, err
	}

	patch := decodeHookPatch(req)
	merged := *cur

	if patch.Type != nil {
		merged.Type = *patch.Type
	}

	if patch.URL != nil {
		merged.URL = *patch.URL
	}

	if patch.TimeoutMs != nil {
		merged.TimeoutMs = *patch.TimeoutMs
	}

	if patch.Enabled != nil {
		merged.Enabled = *patch.Enabled
	}

	if patch.FailOpen != nil {
		merged.FailOpen = *patch.FailOpen
	}

	updated, err := s.deps.Hooks.Update(ctx, params.ProjectID, params.ID, merged)
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminHooksByIdOK{Hook: oas.NewOptHook(oasHook(updated))}, nil
}

type hookPatch struct {
	Type      *string `json:"type"`
	URL       *string `json:"url"`
	TimeoutMs *int    `json:"timeout_ms"`
	Enabled   *bool   `json:"enabled"`
	FailOpen  *bool   `json:"fail_open"`
}

func decodeHookPatch(req oas.PatchV1ProjectsByProjectIdAdminHooksByIdReq) hookPatch {
	m := make(map[string]json.RawMessage, len(req))
	for k, v := range req {
		m[k] = json.RawMessage(v)
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return hookPatch{}
	}

	var p hookPatch

	_ = json.Unmarshal(raw, &p)

	return p
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminHooksById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminHooksByIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Hooks.Delete(ctx, params.ProjectID, params.ID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminHooksByIdTest(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminHooksByIdTestReq, params oas.PostV1ProjectsByProjectIdAdminHooksByIdTestParams) (*oas.PostV1ProjectsByProjectIdAdminHooksByIdTestOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	var payload []byte

	if v, ok := req.Get(); ok {
		if p, ok := v.Payload.Get(); ok {
			m := make(map[string]json.RawMessage, len(p))
			for k, raw := range p {
				m[k] = json.RawMessage(raw)
			}

			payload, _ = json.Marshal(m)
		}
	}

	status, body, durMs, err := s.deps.Hooks.Test(ctx, params.ProjectID, params.ID, payload)
	if err != nil {
		return nil, err
	}

	out := &oas.PostV1ProjectsByProjectIdAdminHooksByIdTestOK{
		Status:     oas.NewOptInt(status),
		DurationMs: oas.NewOptInt(durMs),
	}

	bodyJSON, _ := json.Marshal(body)
	resp := oas.PostV1ProjectsByProjectIdAdminHooksByIdTestOKResponse{"body": jx.Raw(bodyJSON)}
	out.Response = oas.NewOptPostV1ProjectsByProjectIdAdminHooksByIdTestOKResponse(resp)

	return out, nil
}

func oasJob(j domain.AdminJob) oas.Job {
	out := oas.Job{
		ID:       oas.NewOptString(j.ID),
		Type:     oas.NewOptString(j.Type),
		Progress: oas.NewOptJobProgress(oas.JobProgress{Processed: oas.NewOptInt(j.Progress)}),
	}

	var st oas.JobStatus
	if err := st.UnmarshalText([]byte(jobAPIStatus(j.Status))); err == nil {
		out.Status = oas.NewOptJobStatus(st)
	}

	return out
}

// jobAPIStatus maps the internal "pending" status onto the API's "running"
// (queued and in-progress both read as running to the client).
func jobAPIStatus(s string) string {
	if s == "pending" {
		return "running"
	}

	return s
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminJobs(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminJobsParams) (*oas.GetV1ProjectsByProjectIdAdminJobsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	jobs, err := s.deps.Jobs.List(ctx, params.ProjectID, 0)
	if err != nil {
		return nil, err
	}

	data := make([]oas.Job, 0, len(jobs))
	for i := range jobs {
		data = append(data, oasJob(jobs[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminJobsOK{Data: data, HasMore: oas.NewOptBool(false)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminJobsByJobId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminJobsByJobIdParams) (*oas.GetV1ProjectsByProjectIdAdminJobsByJobIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	job, err := s.deps.Jobs.Get(ctx, params.ProjectID, params.JobID)
	if err != nil {
		return nil, err
	}

	return &oas.GetV1ProjectsByProjectIdAdminJobsByJobIdOK{Job: oas.NewOptJob(oasJob(*job))}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminJobsByJobIdCancel(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminJobsByJobIdCancelParams) (*oas.PostV1ProjectsByProjectIdAdminJobsByJobIdCancelOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Jobs.Cancel(ctx, params.ProjectID, params.JobID); err != nil {
		return nil, err
	}

	job, err := s.deps.Jobs.Get(ctx, params.ProjectID, params.JobID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminJobsByJobIdCancelOK{Job: oas.NewOptJob(oasJob(*job))}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminExportsByJobId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminExportsByJobIdParams) (*oas.GetV1ProjectsByProjectIdAdminExportsByJobIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	job, err := s.deps.Jobs.Get(ctx, params.ProjectID, params.JobID)
	if err != nil {
		return nil, err
	}

	out := &oas.GetV1ProjectsByProjectIdAdminExportsByJobIdOK{Status: oas.NewOptString(jobAPIStatus(job.Status))}
	if job.Result != nil {
		if url, ok := job.Result["download_url"].(string); ok && url != "" {
			out.DownloadURL = oas.NewOptNilString(url)
		}
	}

	return out, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminImportUsers(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminImportUsersReq, params oas.PostV1ProjectsByProjectIdAdminImportUsersParams) (*oas.PostV1ProjectsByProjectIdAdminImportUsersOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	users := make([]map[string]jx.Raw, 0, len(req.Users))
	for _, u := range req.Users {
		users = append(users, map[string]jx.Raw(u))
	}

	jobID, status, err := s.deps.Jobs.CreateImportUsers(ctx, params.ProjectID, users, req.PasswordHashFormat.Or(""), req.SendInvites.Or(false))
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminImportUsersOK{
		JobID:  oas.NewOptString(jobID),
		Status: oas.NewOptString(jobAPIStatus(status)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminImportPasswordHashesVerify(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminImportPasswordHashesVerifyReq, params oas.PostV1ProjectsByProjectIdAdminImportPasswordHashesVerifyParams) (*oas.PostV1ProjectsByProjectIdAdminImportPasswordHashesVerifyOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	valid, err := s.deps.Jobs.VerifyPasswordHash(req.Hash, req.Password, req.Format)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminImportPasswordHashesVerifyOK{Valid: oas.NewOptBool(valid)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminRetentionPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminRetentionPolicyParams) (*oas.RetentionPolicy, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	raw, err := s.deps.Config.GetRetentionPolicy(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	var out oas.RetentionPolicy
	if len(raw) > 0 {
		if err := out.UnmarshalJSON(raw); err != nil {
			return nil, err
		}
	}

	return &out, nil
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminRetentionPolicy(ctx context.Context, req *oas.RetentionPolicy, params oas.PutV1ProjectsByProjectIdAdminRetentionPolicyParams) (*oas.RetentionPolicy, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	raw, err := req.MarshalJSON()
	if err != nil {
		return nil, err
	}

	if err := s.deps.Config.PutRetentionPolicy(ctx, params.ProjectID, raw); err != nil {
		return nil, err
	}

	return req, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAuditLogs(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAuditLogsParams) (*oas.GetV1ProjectsByProjectIdAdminAuditLogsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	entries, next, hasMore, err := s.deps.Audit.List(ctx, domain.AuditLogListCmd{
		ProjectID: params.ProjectID,
		ActorID:   params.ActorID.Or(""),
		TargetID:  params.TargetID.Or(""),
		Type:      params.Type.Or(""),
		Cursor:    params.Cursor.Or(""),
		Limit:     params.Limit.Or(0),
	})
	if err != nil {
		return nil, err
	}

	data := make([]oas.AuditLog, 0, len(entries))
	for i := range entries {
		data = append(data, oasAuditLog(entries[i]))
	}

	out := &oas.GetV1ProjectsByProjectIdAdminAuditLogsOK{Data: data, HasMore: oas.NewOptBool(hasMore)}
	if next != "" {
		out.NextCursor = oas.NewOptNilString(next)
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAuditLogsByAuditId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAuditLogsByAuditIdParams) (*oas.GetV1ProjectsByProjectIdAdminAuditLogsByAuditIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	entry, err := s.deps.Audit.Get(ctx, params.ProjectID, params.AuditID)
	if err != nil {
		return nil, err
	}

	return &oas.GetV1ProjectsByProjectIdAdminAuditLogsByAuditIdOK{AuditLog: oas.NewOptAuditLog(oasAuditLog(*entry))}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAuditExport(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAuditExportReq, params oas.PostV1ProjectsByProjectIdAdminAuditExportParams) (*oas.PostV1ProjectsByProjectIdAdminAuditExportOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AuditExportCmd{ProjectID: params.ProjectID, Format: req.Format.Or("json")}
	if v, ok := req.From.Get(); ok {
		cmd.From = v
	}

	if v, ok := req.To.Get(); ok {
		cmd.To = v
	}

	jobID, status, err := s.deps.Audit.CreateExport(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminAuditExportOK{
		JobID:  oas.NewOptString(jobID),
		Status: oas.NewOptString(status),
	}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminEmailTemplates(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminEmailTemplatesParams) (r oas.GetV1ProjectsByProjectIdAdminEmailTemplatesOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	tmpls, err := s.deps.Config.ListEmailTemplates(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	return oas.GetV1ProjectsByProjectIdAdminEmailTemplatesOK(tmpls), nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminFeatures(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminFeaturesParams) (r oas.GetV1ProjectsByProjectIdAdminFeaturesOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	features, err := s.deps.Config.GetFeatures(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	return oas.GetV1ProjectsByProjectIdAdminFeaturesOK(features), nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminI18nByLocale(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminI18nByLocaleParams) (r oas.GetV1ProjectsByProjectIdAdminI18nByLocaleOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	msgs, err := s.deps.Config.GetI18n(ctx, adminCfg(params.ProjectID, params.XEnvironment), params.Locale)
	if err != nil {
		return nil, err
	}

	return oas.GetV1ProjectsByProjectIdAdminI18nByLocaleOK(msgs), nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminJwks(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminJwksParams) (r *oas.GetV1ProjectsByProjectIdAdminJwksOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	keys, err := s.deps.Keys.ListSigningKeys(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	data := make([]oas.SigningKey, 0, len(keys))
	for i := range keys {
		data = append(data, oasAdminSigningKey(&keys[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminJwksOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminSmsProviders(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSmsProvidersParams) (r *oas.GetV1ProjectsByProjectIdAdminSmsProvidersOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	provs, err := s.deps.Config.ListSmsProviders(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	data := make([]oas.SmsProvider, 0, len(provs))
	for i := range provs {
		data = append(data, oasAdminSmsProvider(&provs[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminSmsProvidersOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminTokenProfiles(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminTokenProfilesParams) (r *oas.GetV1ProjectsByProjectIdAdminTokenProfilesOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	profiles, err := s.deps.Keys.ListTokenProfiles(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	data := make([]oas.TokenProfile, 0, len(profiles))
	for i := range profiles {
		data = append(data, oasAdminTokenProfile(&profiles[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminTokenProfilesOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsers(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersParams) (*oas.GetV1ProjectsByProjectIdAdminUsersOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	accts, err := s.deps.Users.List(ctx, params.ProjectID, params.XEnvironment.Or("live"))
	if err != nil {
		return nil, err
	}

	data := make([]oas.User, 0, len(accts))
	for i := range accts {
		data = append(data, oasUser(&accts[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminUsersOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdParams) (*oas.GetV1ProjectsByProjectIdAdminUsersByUserIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	acct, err := s.deps.Users.Get(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID)
	if err != nil {
		return nil, err
	}

	return &oas.GetV1ProjectsByProjectIdAdminUsersByUserIdOK{
		User: oas.NewOptUser(oasUser(acct)),
	}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdIdentities(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesParams) (*oas.GetV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	ids, err := s.deps.Users.ListIdentities(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.Identity, 0, len(ids))
	for i := range ids {
		data = append(data, oasIdentity(&ids[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdGrants(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdGrantsParams) (*oas.GetV1ProjectsByProjectIdAdminUsersByUserIdGrantsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	grants, err := s.deps.Grants.ListGrants(ctx, params.UserID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.OAuthGrant, 0, len(grants))
	for i := range grants {
		data = append(data, oasOAuthGrant(grants[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminUsersByUserIdGrantsOK{Data: data}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserIdGrantsByGrantId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdGrantsByGrantIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Grants.RevokeGrant(ctx, params.UserID, params.GrantID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdSessions(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsParams) (*oas.GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	sessions, err := s.deps.Users.ListSessions(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.Session, 0, len(sessions))
	for i := range sessions {
		data = append(data, oasSession(&sessions[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsOK{Data: data}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdReq, params oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdParams) (*oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	app, err := s.deps.Apps.Update(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.AppID, oasRawPatch(req))
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdOK{
		App: oas.NewOptAppClient(oasAppClient(app)),
	}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigAuth(ctx context.Context, req *oas.AuthConfig, params oas.PatchV1ProjectsByProjectIdAdminConfigAuthParams) (r *oas.AuthConfig, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.UpdateAuthConfig(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Doc:         oasEncodeConfig(req),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.AuthConfig{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigPasswordPolicy(ctx context.Context, req *oas.PasswordPolicy, params oas.PatchV1ProjectsByProjectIdAdminConfigPasswordPolicyParams) (r *oas.PasswordPolicy, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.UpdatePasswordPolicy(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Doc:         oasEncodeConfig(req),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.PasswordPolicy{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigSessionPolicy(ctx context.Context, req *oas.SessionPolicy, params oas.PatchV1ProjectsByProjectIdAdminConfigSessionPolicyParams) (r *oas.SessionPolicy, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.UpdateSessionPolicy(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Doc:         oasEncodeConfig(req),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.SessionPolicy{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigRateLimits(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigRateLimitsParams) (r *oas.RateLimits, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.GetRateLimits(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	out := &oas.RateLimits{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigMfaPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigMfaPolicyParams) (r *oas.MfaPolicy, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.GetMfaPolicy(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	out := &oas.MfaPolicy{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigMfaPolicy(ctx context.Context, req *oas.MfaPolicy, params oas.PatchV1ProjectsByProjectIdAdminConfigMfaPolicyParams) (r *oas.MfaPolicy, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.UpdateMfaPolicy(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Doc:         oasEncodeConfig(req),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.MfaPolicy{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigRateLimits(ctx context.Context, req *oas.RateLimits, params oas.PatchV1ProjectsByProjectIdAdminConfigRateLimitsParams) (r *oas.RateLimits, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.UpdateRateLimits(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Doc:         oasEncodeConfig(req),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.RateLimits{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminEmailProvidersById(ctx context.Context, req *oas.EmailProvider, params oas.PatchV1ProjectsByProjectIdAdminEmailProvidersByIdParams) (r *oas.EmailProvider, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prov, err := s.deps.Config.UpdateEmailProvider(ctx, domain.AdminProviderCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          params.ID,
		Type:        req.Type,
		Config:      map[string]jx.Raw(req.Config.Or(nil)),
		Enabled:     req.Enabled.Or(false),
	})
	if err != nil {
		return nil, err
	}

	out := oasAdminEmailProvider(prov)

	return &out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminEmailTemplatesById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdReq, params oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdParams) (r oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	tmpl, err := s.deps.Config.UpdateEmailTemplate(ctx, domain.AdminTemplateUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		TemplateID:  params.ID,
		Patch:       map[string]jx.Raw(req),
	})
	if err != nil {
		return nil, err
	}

	return oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdOK(tmpl), nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminSmsProvidersById(ctx context.Context, req *oas.SmsProvider, params oas.PatchV1ProjectsByProjectIdAdminSmsProvidersByIdParams) (r *oas.SmsProvider, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prov, err := s.deps.Config.UpdateSmsProvider(ctx, domain.AdminProviderCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          params.ID,
		Type:        req.Type,
		Config:      map[string]jx.Raw(req.Config.Or(nil)),
		Enabled:     req.Enabled.Or(false),
	})
	if err != nil {
		return nil, err
	}

	out := oasAdminSmsProvider(prov)

	return &out, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminTokenProfilesById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdReq, params oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdParams) (r *oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prof, err := s.deps.Keys.UpdateTokenProfile(ctx, domain.AdminTokenProfileCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          params.ID,
		Profile:     domain.AdminConfigDoc(req),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdOK{
		Profile: oas.NewOptTokenProfile(oasAdminTokenProfile(prof)),
	}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdReq, params oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdParams) (*oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AdminUserUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or("live"),
		AccountID:   params.UserID,
		Name:        oasRawString(req, "name"),
		Locale:      oasRawString(req, "locale"),
	}

	acct, err := s.deps.Users.Update(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdOK{
		User: oas.NewOptUser(oasUser(acct)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAccessRequestsByIdApprove(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveReq, params oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveParams) (r oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveOK, _ error) {
	p, err := requireProjectAdmin(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	res, err := s.deps.AccessRequests.Approve(ctx, domain.AdminAccessRequestDecisionCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		RequestID:   params.ID,
		ActorID:     p.AccountID,
	})
	if err != nil {
		return nil, err
	}

	return oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveOK(res), nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAccessRequestsByIdDeny(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyReq, params oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyParams) (r *oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyOK, _ error) {
	p, err := requireProjectAdmin(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	cmd := domain.AdminAccessRequestDecisionCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		RequestID:   params.ID,
		ActorID:     p.AccountID,
	}
	if v, ok := req.Get(); ok {
		cmd.Reason = v.Reason.Or("")
	}

	ar, err := s.deps.AccessRequests.Deny(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyOK{
		Request: oas.NewOptAccessRequest(oasAdminAccessRequest(ar)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminApps(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAppsReq, params oas.PostV1ProjectsByProjectIdAdminAppsParams) (*oas.PostV1ProjectsByProjectIdAdminAppsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AppClientCmd{
		ProjectID:      params.ProjectID,
		Environment:    params.XEnvironment.Or("live"),
		Name:           req.Name,
		Type:           string(req.Type),
		RedirectURIs:   req.RedirectUris,
		AllowedOrigins: req.AllowedOrigins,
		Disabled:       req.Disabled.Or(false),

		PostLogoutRedirectURIs: req.PostLogoutRedirectUris,
		BackchannelLogoutURI:   req.BackchannelLogoutURI.Or(""),
		TokenProfileID:         req.TokenProfileID.Or(""),
		Scopes:                 req.Scopes,
		JWKS:                   req.Jwks.Or(""),
		JWKSURI:                req.JwksURI.Or(""),
	}

	app, err := s.deps.Apps.Create(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminAppsCreated{
		App: oas.NewOptAppClient(oasAppClient(app)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAppsByAppIdSecrets(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsReq, params oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsParams) (*oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	secret, err := s.deps.Apps.AddSecret(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.AppID, req.Name)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsCreated{
		SecretID:     oas.NewOptString(secret.SecretID),
		ClientSecret: oas.NewOptString(secret.ClientSecret),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailProviders(ctx context.Context, req *oas.EmailProvider, params oas.PostV1ProjectsByProjectIdAdminEmailProvidersParams) (r *oas.EmailProvider, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prov, err := s.deps.Config.CreateEmailProvider(ctx, domain.AdminProviderCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          req.ID.Or(""),
		Type:        req.Type,
		Config:      map[string]jx.Raw(req.Config.Or(nil)),
		Enabled:     req.Enabled.Or(false),
	})
	if err != nil {
		return nil, err
	}

	out := oasAdminEmailProvider(prov)

	return &out, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewReq, params oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewParams) (r *oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AdminTemplatePreviewCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		TemplateID:  params.ID,
	}
	if v, ok := req.Get(); ok {
		cmd.Locale = v.Locale.Or("")
		cmd.Data = map[string]jx.Raw(v.Data.Or(nil))
	}

	prev, err := s.deps.Config.PreviewEmailTemplate(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewOK{
		Subject: oas.NewOptString(prev.Subject),
		HTML:    oas.NewOptString(prev.HTML),
		Text:    oas.NewOptString(prev.Text),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTest(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTestReq, params oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTestParams) (r *oas.Ok, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Config.SendTestEmail(ctx, domain.AdminTemplateSendTestCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		TemplateID:  params.ID,
		To:          req.To,
		Locale:      req.Locale.Or(""),
		Data:        map[string]jx.Raw(req.Data.Or(nil)),
	}); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminSmsProvidersSendTest(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminSmsProvidersSendTestReq, params oas.PostV1ProjectsByProjectIdAdminSmsProvidersSendTestParams) (r *oas.Ok, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Config.SendTestSMS(ctx, domain.AdminTemplateSendTestCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		TemplateID:  req.TemplateID.Or(""),
		To:          req.To,
		Locale:      req.Locale.Or(""),
		Data:        map[string]jx.Raw(req.Data.Or(nil)),
	}); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminJwksByKeyIdActivate(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminJwksByKeyIdActivateParams) (r *oas.PostV1ProjectsByProjectIdAdminJwksByKeyIdActivateOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	key, err := s.deps.Keys.ActivateSigningKey(ctx, adminCfg(params.ProjectID, params.XEnvironment), params.KeyID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminJwksByKeyIdActivateOK{
		Key: oas.NewOptSigningKey(oasAdminSigningKey(key)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminJwksRotate(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminJwksRotateReq, params oas.PostV1ProjectsByProjectIdAdminJwksRotateParams) (r *oas.PostV1ProjectsByProjectIdAdminJwksRotateOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AdminJWKSRotateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
	}
	if v, ok := req.Get(); ok {
		cmd.Activate = v.Activate.Or(false)
	}

	key, err := s.deps.Keys.RotateSigningKeys(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminJwksRotateOK{
		Key: oas.NewOptSigningKey(oasAdminSigningKey(key)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminSmsProviders(ctx context.Context, req *oas.SmsProvider, params oas.PostV1ProjectsByProjectIdAdminSmsProvidersParams) (r *oas.SmsProvider, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prov, err := s.deps.Config.CreateSmsProvider(ctx, domain.AdminProviderCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          req.ID.Or(""),
		Type:        req.Type,
		Config:      map[string]jx.Raw(req.Config.Or(nil)),
		Enabled:     req.Enabled.Or(false),
	})
	if err != nil {
		return nil, err
	}

	out := oasAdminSmsProvider(prov)

	return &out, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminTokenProfiles(ctx context.Context, req *oas.TokenProfile, params oas.PostV1ProjectsByProjectIdAdminTokenProfilesParams) (r *oas.PostV1ProjectsByProjectIdAdminTokenProfilesCreated, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	prof, err := s.deps.Keys.CreateTokenProfile(ctx, domain.AdminTokenProfileCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ID:          req.ID.Or(""),
		Profile:     oasEncodeConfig(req),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminTokenProfilesCreated{
		Profile: oas.NewOptTokenProfile(oasAdminTokenProfile(prof)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreview(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewReq, params oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewParams) (r *oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	claims, err := s.deps.Keys.PreviewTokenProfile(ctx, domain.AdminTokenProfilePreviewCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		ProfileID:   params.ID,
		UserID:      req.UserID,
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewOK{
		Claims: oas.NewOptPostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewOKClaims(
			oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewOKClaims(claims)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsers(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersReq, params oas.PostV1ProjectsByProjectIdAdminUsersParams) (*oas.PostV1ProjectsByProjectIdAdminUsersCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.RegisterCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or("live"),
		Email:       req.Email.Or(""),
		Phone:       req.Phone.Or(""),
		Password:    req.Password.Or(""),
	}

	acct, err := s.deps.Users.Create(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersCreated{
		User: oas.NewOptUser(oasUser(acct)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdAnonymize(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminUsersByUserIdAnonymizeReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdAnonymizeParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AdminUserAnonymizeCmd{ProjectID: params.ProjectID, Environment: params.XEnvironment.Or("live"), AccountID: params.UserID}
	if v, ok := req.Get(); ok {
		cmd.Reason = v.Reason.Or("")
	}

	if err := s.deps.Users.Anonymize(ctx, cmd); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdBan(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AdminUserBanCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or("live"),
		AccountID:   params.UserID,
		Reason:      req.Reason.Or(""),
		Until:       req.Until.Or(time.Time{}),
	}

	acct, err := s.deps.Users.BanWith(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanOK{
		User: oas.NewOptUser(oasUser(acct)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdExport(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdExportParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdExportOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	jobID, err := s.deps.Users.Export(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdExportOK{
		JobID: oas.NewOptString(jobID),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonate(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateOK, error) {
	p, err := requireProjectAdmin(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	res, err := s.deps.Users.Impersonate(ctx, domain.AdminUserImpersonateCmd{
		ProjectID:       params.ProjectID,
		Environment:     params.XEnvironment.Or("live"),
		AccountID:       params.UserID,
		ActorID:         p.AccountID,
		Reason:          req.Reason,
		DurationSeconds: req.DurationSeconds,
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateOK{
		ImpersonationURL: oas.NewOptString(res.URL),
		ExpiresAt:        oas.NewOptTimestamp(oas.Timestamp(res.ExpiresAt)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdMfaReset(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminUsersByUserIdMfaResetReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdMfaResetParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdMfaResetOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	var factorIDs []string

	if v, ok := req.Get(); ok {
		if ids, ok := v.FactorIds.Get(); ok {
			factorIDs = ids
		}
	}

	removed, err := s.deps.Users.ResetMFA(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID, factorIDs)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdMfaResetOK{
		RemovedCount: oas.NewOptInt(removed),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdPassword(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdPasswordReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdPasswordParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	err := s.deps.Users.SetPassword(ctx, domain.AdminUserPasswordCmd{
		ProjectID:      params.ProjectID,
		Environment:    params.XEnvironment.Or("live"),
		AccountID:      params.UserID,
		Password:       req.Password,
		RevokeSessions: req.RevokeSessions.Or(false),
	})
	if err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevoke(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	cmd := domain.AdminUserSessionsRevokeCmd{ProjectID: params.ProjectID, Environment: params.XEnvironment.Or("live"), AccountID: params.UserID}
	if v, ok := req.Get(); ok {
		cmd.ExceptSessionID = v.ExceptSessionID.Or("")
		cmd.Reason = v.Reason.Or("")
	}

	revoked, err := s.deps.Users.RevokeSessions(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeOK{
		RevokedCount: oas.NewOptInt(revoked),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdUnban(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdUnbanParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdUnbanOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	acct, err := s.deps.Users.Unban(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdUnbanOK{
		User: oas.NewOptUser(oasUser(acct)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmail(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmailParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmailOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	acct, err := s.deps.Users.VerifyEmail(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmailOK{
		User: oas.NewOptUser(oasUser(acct)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhone(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhoneParams) (*oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhoneOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	acct, err := s.deps.Users.VerifyPhone(ctx, params.ProjectID, params.XEnvironment.Or("live"), params.UserID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhoneOK{
		User: oas.NewOptUser(oasUser(acct)),
	}, nil
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminConsents(ctx context.Context, req *oas.ConsentConfig, params oas.PutV1ProjectsByProjectIdAdminConsentsParams) (r *oas.ConsentConfig, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doc, err := s.deps.Config.PutConsent(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Doc:         oasEncodeConfig(req),
	})
	if err != nil {
		return nil, err
	}

	out := &oas.ConsentConfig{}
	if err := oasDecodeConfig(doc, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminFeatures(ctx context.Context, req oas.PutV1ProjectsByProjectIdAdminFeaturesReq, params oas.PutV1ProjectsByProjectIdAdminFeaturesParams) (r oas.PutV1ProjectsByProjectIdAdminFeaturesOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	features, err := s.deps.Config.PutFeatures(ctx, domain.AdminFeaturesUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Features:    map[string]bool(req),
	})
	if err != nil {
		return nil, err
	}

	return oas.PutV1ProjectsByProjectIdAdminFeaturesOK(features), nil
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminI18nByLocale(ctx context.Context, req oas.PutV1ProjectsByProjectIdAdminI18nByLocaleReq, params oas.PutV1ProjectsByProjectIdAdminI18nByLocaleParams) (r oas.PutV1ProjectsByProjectIdAdminI18nByLocaleOK, _ error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	msgs, err := s.deps.Config.PutI18n(ctx, domain.AdminI18nUpdateCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Locale:      params.Locale,
		Messages:    map[string]jx.Raw(req),
	})
	if err != nil {
		return nil, err
	}

	return oas.PutV1ProjectsByProjectIdAdminI18nByLocaleOK(msgs), nil
}

// ===== Service accounts =====

func (s *AdminService) GetV1ProjectsByProjectIdAdminServiceAccounts(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminServiceAccountsParams) (*oas.GetV1ProjectsByProjectIdAdminServiceAccountsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	sas, err := s.deps.ServiceAccounts.List(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.ServiceAccount, 0, len(sas))
	for i := range sas {
		data = append(data, oasServiceAccount(&sas[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminServiceAccountsOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminServiceAccountsBySaId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminServiceAccountsBySaIdParams) (*oas.GetV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	sa, err := s.deps.ServiceAccounts.Get(ctx, params.ProjectID, params.SaID)
	if err != nil {
		return nil, err
	}

	return &oas.GetV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK{
		ServiceAccount: oas.NewOptServiceAccount(oasServiceAccount(sa)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminServiceAccounts(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminServiceAccountsReq, params oas.PostV1ProjectsByProjectIdAdminServiceAccountsParams) (*oas.PostV1ProjectsByProjectIdAdminServiceAccountsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	sa, err := s.deps.ServiceAccounts.Create(ctx, domain.ServiceAccountCmd{
		ProjectID: params.ProjectID,
		Name:      req.Name,
		Scopes:    req.Scopes,
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminServiceAccountsCreated{
		ServiceAccount: oas.NewOptServiceAccount(oasServiceAccount(sa)),
	}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminServiceAccountsBySaId(ctx context.Context, req *oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdReq, params oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdParams) (*oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	sa, err := s.deps.ServiceAccounts.Update(ctx, domain.AdminServiceAccountUpdateCmd{
		ProjectID:        params.ProjectID,
		ServiceAccountID: params.SaID,
		Scopes:           req.Scopes,
		Disabled:         req.Disabled.Or(false),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminServiceAccountsBySaIdOK{
		ServiceAccount: oas.NewOptServiceAccount(oasServiceAccount(sa)),
	}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.ServiceAccounts.Delete(ctx, params.ProjectID, params.SaID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecrets(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsReq, params oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsParams) (*oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	secret, err := s.deps.ServiceAccounts.AddSecret(ctx, domain.AdminServiceAccountSecretCmd{
		ProjectID:        params.ProjectID,
		ServiceAccountID: params.SaID,
		Name:             req.Name,
		ExpiresAt:        req.ExpiresAt.Or(time.Time{}),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsCreated{
		SecretID:     oas.NewOptString(secret.SecretID),
		ClientID:     oas.NewOptString(secret.ClientID),
		ClientSecret: oas.NewOptString(secret.ClientSecret),
	}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsBySecretId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminServiceAccountsBySaIdSecretsBySecretIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.ServiceAccounts.DeleteSecret(ctx, params.ProjectID, params.SaID, params.SecretID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

// ===== API keys =====

func (s *AdminService) GetV1ProjectsByProjectIdAdminApiKeys(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminApiKeysParams) (*oas.GetV1ProjectsByProjectIdAdminApiKeysOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	keys, err := s.deps.APIKeys.List(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.ApiKey, 0, len(keys))
	for i := range keys {
		data = append(data, oasApiKey(&keys[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminApiKeysOK{Data: data}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminApiKeys(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminApiKeysReq, params oas.PostV1ProjectsByProjectIdAdminApiKeysParams) (*oas.PostV1ProjectsByProjectIdAdminApiKeysCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	res, err := s.deps.APIKeys.Create(ctx, domain.AdminAPIKeyCmd{
		ProjectID: params.ProjectID,
		Name:      req.Name,
		Scopes:    req.Scopes,
		ExpiresAt: req.ExpiresAt.Or(time.Time{}),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminApiKeysCreated{
		APIKey: oas.NewOptApiKey(oasApiKey(res.Key)),
		Secret: oas.NewOptString(res.Secret),
	}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminApiKeysByKeyId(ctx context.Context, req *oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdReq, params oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdParams) (*oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	key, err := s.deps.APIKeys.Update(ctx, domain.AdminAPIKeyUpdateCmd{
		ProjectID: params.ProjectID,
		KeyID:     params.KeyID,
		Name:      req.Name.Or(""),
		Scopes:    req.Scopes,
		Disabled:  req.Disabled.Or(false),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminApiKeysByKeyIdOK{
		APIKey: oas.NewOptApiKey(oasApiKey(key)),
	}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminApiKeysByKeyId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminApiKeysByKeyIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.APIKeys.Delete(ctx, params.ProjectID, params.KeyID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotate(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotateParams) (*oas.PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotateOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	res, err := s.deps.APIKeys.Rotate(ctx, params.ProjectID, params.KeyID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminApiKeysByKeyIdRotateOK{
		APIKey: oas.NewOptApiKey(oasApiKey(res.Key)),
		Secret: oas.NewOptString(res.Secret),
	}, nil
}

// ===== SSO connections =====

func (s *AdminService) GetV1ProjectsByProjectIdAdminSsoConnections(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSsoConnectionsParams) (*oas.GetV1ProjectsByProjectIdAdminSsoConnectionsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	conns, err := s.deps.Connections.List(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.SSOConnection, 0, len(conns))
	for i := range conns {
		data = append(data, oasConnection(&conns[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminSsoConnectionsOK{Data: data}, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminSsoConnectionsById(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdParams) (*oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	conn, err := s.deps.Connections.Get(ctx, params.ProjectID, params.ID)
	if err != nil {
		return nil, err
	}

	return &oas.GetV1ProjectsByProjectIdAdminSsoConnectionsByIdOK{
		Connection: oas.NewOptSSOConnection(oasConnection(conn)),
	}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminSsoConnections(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminSsoConnectionsReq, params oas.PostV1ProjectsByProjectIdAdminSsoConnectionsParams) (*oas.PostV1ProjectsByProjectIdAdminSsoConnectionsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	conn, err := s.deps.Connections.Create(ctx, domain.AdminConnectionCmd{
		ProjectID:   params.ProjectID,
		Type:        string(req.Type),
		Name:        req.Name,
		Domains:     req.Domains,
		ExternalRef: req.ExternalRef.Or(""),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminSsoConnectionsCreated{
		Connection: oas.NewOptSSOConnection(oasConnection(conn)),
	}, nil
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminSsoConnectionsById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminSsoConnectionsByIdReq, params oas.PatchV1ProjectsByProjectIdAdminSsoConnectionsByIdParams) (*oas.PatchV1ProjectsByProjectIdAdminSsoConnectionsByIdOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	conn, err := s.deps.Connections.Update(ctx, params.ProjectID, params.ID, oasRawPatch(req))
	if err != nil {
		return nil, err
	}

	return &oas.PatchV1ProjectsByProjectIdAdminSsoConnectionsByIdOK{
		Connection: oas.NewOptSSOConnection(oasConnection(conn)),
	}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminSsoConnectionsById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminSsoConnectionsByIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Connections.Delete(ctx, params.ProjectID, params.ID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

// ===== Verification domains =====

func (s *AdminService) GetV1ProjectsByProjectIdAdminDomains(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminDomainsParams) (*oas.GetV1ProjectsByProjectIdAdminDomainsOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	doms, err := s.deps.Connections.ListDomains(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	data := make([]oas.Domain, 0, len(doms))
	for i := range doms {
		data = append(data, oasDomain(&doms[i]))
	}

	return &oas.GetV1ProjectsByProjectIdAdminDomainsOK{Data: data}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminDomains(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminDomainsReq, params oas.PostV1ProjectsByProjectIdAdminDomainsParams) (*oas.PostV1ProjectsByProjectIdAdminDomainsCreated, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	reg, err := s.deps.Connections.CreateDomain(ctx, domain.AdminDomainCmd{
		ProjectID:    params.ProjectID,
		Domain:       req.Domain,
		ConnectionID: req.ConnectionID.Or(""),
	})
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminDomainsCreated{
		Domain: oas.NewOptDomain(oasDomain(reg.Domain)),
		VerificationRecord: oas.NewOptPostV1ProjectsByProjectIdAdminDomainsCreatedVerificationRecord(
			oas.PostV1ProjectsByProjectIdAdminDomainsCreatedVerificationRecord{
				Type:  oas.NewOptString(reg.VerificationRecordType),
				Name:  oas.NewOptString(reg.VerificationRecordName),
				Value: oas.NewOptString(reg.VerificationRecordValue),
			}),
	}, nil
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminDomainsByDomainId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminDomainsByDomainIdParams) (*oas.Ok, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	if err := s.deps.Connections.DeleteDomain(ctx, params.ProjectID, params.DomainID); err != nil {
		return nil, err
	}

	return &oas.Ok{Ok: oas.NewOptBool(true)}, nil
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerify(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerifyParams) (*oas.PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerifyOK, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	dom, err := s.deps.Connections.VerifyDomain(ctx, params.ProjectID, params.DomainID)
	if err != nil {
		return nil, err
	}

	return &oas.PostV1ProjectsByProjectIdAdminDomainsByDomainIdVerifyOK{
		Domain: oas.NewOptDomain(oasDomain(dom)),
	}, nil
}

// oasRawString extracts a JSON string field from a map[string]jx.Raw patch
// body, returning "" when absent or not a string.
func oasRawString[T ~map[string]jx.Raw](m T, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}

	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}

	return v
}

// oasRawPatch decodes a map[string]jx.Raw patch body into a generic
// map[string]any the domain layer can apply field-by-field.
func oasRawPatch[T ~map[string]jx.Raw](m T) map[string]any {
	out := make(map[string]any, len(m))
	for k, raw := range m {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			continue
		}

		out[k] = v
	}

	return out
}

func oasWebhook(w *domain.Webhook) oas.Webhook {
	out := oas.Webhook{
		ID:      w.ID,
		URL:     w.URL,
		Events:  w.Events,
		Enabled: w.Enabled,
	}
	if w.Description != "" {
		out.Description = oas.NewOptString(w.Description)
	}

	if w.Environment != "" {
		out.Environment = oas.NewOptString(w.Environment)
	}

	if !w.CreatedAt.IsZero() {
		out.CreatedAt = oas.NewOptTimestamp(oas.Timestamp(w.CreatedAt))
	}

	if !w.UpdatedAt.IsZero() {
		out.UpdatedAt = oas.NewOptTimestamp(oas.Timestamp(w.UpdatedAt))
	}

	return out
}

func oasPublicEvent(event domain.PublicEvent) oas.Event {
	data := make(map[string]any, len(event.Data))
	for key, value := range event.Data {
		data[key] = value
	}

	return oas.Event{
		ID:          event.ID,
		Type:        event.Type,
		Version:     event.Version,
		CreatedAt:   oas.Timestamp(event.OccurredAt),
		ProjectID:   event.ProjectID,
		Environment: event.Environment,
		Data:        oasRawMap[oas.EventData](data),
	}
}

func oasWebhookDelivery(delivery *domain.WebhookDelivery) oas.WebhookDelivery {
	out := oas.WebhookDelivery{
		ID:           delivery.ID,
		WebhookID:    delivery.WebhookID,
		EventID:      delivery.EventID,
		EventType:    delivery.EventType,
		Status:       oas.WebhookDeliveryStatus(delivery.Status),
		AttemptCount: delivery.AttemptCount,
		CreatedAt:    oas.Timestamp(delivery.CreatedAt),
		UpdatedAt:    oas.Timestamp(delivery.UpdatedAt),
	}
	if delivery.NextAttemptAt != nil {
		out.NextAttemptAt = oas.NewOptNilTimestamp(oas.Timestamp(*delivery.NextAttemptAt))
	}

	if delivery.LastAttemptAt != nil {
		out.LastAttemptAt = oas.NewOptNilTimestamp(oas.Timestamp(*delivery.LastAttemptAt))
	}

	if delivery.DeliveredAt != nil {
		out.DeliveredAt = oas.NewOptNilTimestamp(oas.Timestamp(*delivery.DeliveredAt))
	}

	if delivery.ResponseStatus != nil {
		out.ResponseStatus = oas.NewOptNilInt(*delivery.ResponseStatus)
	}

	if delivery.ResponseBody != "" {
		out.ResponseBody = oas.NewOptNilString(delivery.ResponseBody)
	}

	if delivery.LastError != "" {
		out.LastError = oas.NewOptNilString(delivery.LastError)
	}

	return out
}

func webhookUpdateCmd(projectID, environment, id string, req *oas.PatchV1ProjectsByProjectIdAdminWebhooksByIdReq) (domain.WebhookUpdateCmd, error) {
	cmd := domain.WebhookUpdateCmd{ProjectID: projectID, Environment: environment, ID: id}
	if value, ok := req.URL.Get(); ok {
		cmd.URL = &value
	}

	if req.Events != nil {
		events := append([]string(nil), req.Events...)
		cmd.Events = &events
	}

	if value, ok := req.Description.Get(); ok {
		cmd.Description = &value
	}

	if value, ok := req.Enabled.Get(); ok {
		cmd.Enabled = &value
	}

	return cmd, nil
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdRoles(
	ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdRolesParams,
) (*oas.UserRoles, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	roles, err := s.deps.Roles.ListRoles(ctx, domain.AdminUserRolesCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		UserID:      params.UserID,
	})
	if err != nil {
		return nil, err
	}

	return &oas.UserRoles{Roles: rolesOrEmpty(roles)}, nil
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminUsersByUserIdRoles(
	ctx context.Context, req *oas.UserRoles, params oas.PutV1ProjectsByProjectIdAdminUsersByUserIdRolesParams,
) (*oas.UserRoles, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	roles, err := s.deps.Roles.SetRoles(ctx, domain.AdminUserRolesSetCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		UserID:      params.UserID,
		Roles:       req.Roles,
	})
	if err != nil {
		return nil, err
	}

	return &oas.UserRoles{Roles: rolesOrEmpty(roles)}, nil
}

// rolesOrEmpty keeps `roles` a JSON array rather than null when a user has none.
func rolesOrEmpty(roles []string) []string {
	if roles == nil {
		return []string{}
	}

	return roles
}

// ----- desired-state (IaC) apply -----

// oasProjectConfigToBundle converts the wire bundle into per-document config
// docs. Only the documents the caller actually sent end up in the bundle; the
// adapter leaves the rest untouched.
func oasProjectConfigToBundle(req *oas.ProjectConfig) domain.AdminConfigBundle {
	out := domain.AdminConfigBundle{}

	if v, ok := req.Auth.Get(); ok {
		out[domain.ConfigDocAuth] = oasEncodeConfig(&v)
	}

	if v, ok := req.PasswordPolicy.Get(); ok {
		out[domain.ConfigDocPasswordPolicy] = oasEncodeConfig(&v)
	}

	if v, ok := req.SessionPolicy.Get(); ok {
		out[domain.ConfigDocSessionPolicy] = oasEncodeConfig(&v)
	}

	if v, ok := req.MfaPolicy.Get(); ok {
		out[domain.ConfigDocMFAPolicy] = oasEncodeConfig(&v)
	}

	if v, ok := req.RateLimits.Get(); ok {
		out[domain.ConfigDocRateLimits] = oasEncodeConfig(&v)
	}

	return out
}

// oasProjectConfig renders a config bundle back onto the wire type. An unset
// document decodes into its zero value, so the response always carries every
// document key.
func oasProjectConfig(bundle domain.AdminConfigBundle) (oas.ProjectConfig, error) {
	var out oas.ProjectConfig

	var authCfg oas.AuthConfig
	if err := oasDecodeConfig(bundle[domain.ConfigDocAuth], &authCfg); err != nil {
		return out, err
	}

	out.Auth = oas.NewOptAuthConfig(authCfg)

	var passwordPolicy oas.PasswordPolicy
	if err := oasDecodeConfig(bundle[domain.ConfigDocPasswordPolicy], &passwordPolicy); err != nil {
		return out, err
	}

	out.PasswordPolicy = oas.NewOptPasswordPolicy(passwordPolicy)

	var sessionPolicy oas.SessionPolicy
	if err := oasDecodeConfig(bundle[domain.ConfigDocSessionPolicy], &sessionPolicy); err != nil {
		return out, err
	}

	out.SessionPolicy = oas.NewOptSessionPolicy(sessionPolicy)

	var mfaPolicy oas.MfaPolicy
	if err := oasDecodeConfig(bundle[domain.ConfigDocMFAPolicy], &mfaPolicy); err != nil {
		return out, err
	}

	out.MfaPolicy = oas.NewOptMfaPolicy(mfaPolicy)

	var rateLimits oas.RateLimits
	if err := oasDecodeConfig(bundle[domain.ConfigDocRateLimits], &rateLimits); err != nil {
		return out, err
	}

	out.RateLimits = oas.NewOptRateLimits(rateLimits)

	return out, nil
}

// oasConfigApplyResult renders a bulk-config change set onto the wire type.
func oasConfigApplyResult(res *domain.AdminConfigApplyResult) (*oas.ConfigApplyResult, error) {
	cfg, err := oasProjectConfig(res.Config)
	if err != nil {
		return nil, err
	}

	out := &oas.ConfigApplyResult{
		DryRun:  oas.NewOptBool(res.DryRun),
		Config:  oas.NewOptProjectConfig(cfg),
		Changes: make([]oas.ConfigDocumentChange, 0, len(res.Changes)),
	}

	for _, c := range res.Changes {
		change := oas.ConfigDocumentChange{
			Document: oas.NewOptConfigDocumentChangeDocument(oas.ConfigDocumentChangeDocument(c.Document)),
			Action:   oas.NewOptConfigDocumentChangeAction(oas.ConfigDocumentChangeAction(c.Action)),
			After:    oas.NewOptConfigDocumentChangeAfter(oas.ConfigDocumentChangeAfter(c.After)),
		}
		if c.Before == nil {
			change.Before.SetToNull()
		} else {
			change.Before = oas.NewOptNilConfigDocumentChangeBefore(oas.ConfigDocumentChangeBefore(c.Before))
		}

		out.Changes = append(out.Changes, change)
	}

	return out, nil
}

// oasAppClientsApplyResult renders a desired-state client change set onto the
// wire type.
func oasAppClientsApplyResult(res *domain.AdminAppsApplyResult) *oas.AppClientApplyResult {
	out := &oas.AppClientApplyResult{
		DryRun:  oas.NewOptBool(res.DryRun),
		Prune:   oas.NewOptBool(res.Prune),
		Changes: make([]oas.AppClientChange, 0, len(res.Changes)),
	}

	for _, c := range res.Changes {
		change := oas.AppClientChange{
			ID:     oas.NewOptString(c.ID),
			Action: oas.NewOptAppClientChangeAction(oas.AppClientChangeAction(c.Action)),
		}
		if c.Before == nil {
			change.Before.SetToNull()
		} else {
			change.Before = oas.NewOptNilAppClient(oasAppClient(c.Before))
		}

		if c.After == nil {
			change.After.SetToNull()
		} else {
			change.After = oas.NewOptNilAppClient(oasAppClient(c.After))
		}

		out.Changes = append(out.Changes, change)
	}

	return out
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfig(
	ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigParams,
) (*oas.ProjectConfig, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	bundle, err := s.deps.Config.GetConfigBundle(ctx, adminCfg(params.ProjectID, params.XEnvironment))
	if err != nil {
		return nil, err
	}

	out, err := oasProjectConfig(bundle)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminConfig(
	ctx context.Context, req *oas.ProjectConfig, params oas.PutV1ProjectsByProjectIdAdminConfigParams,
) (*oas.ConfigApplyResult, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	res, err := s.deps.Config.ApplyConfig(ctx, domain.AdminConfigApplyCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Docs:        oasProjectConfigToBundle(req),
		DryRun:      params.DryRun.Or(false),
	})
	if err != nil {
		return nil, err
	}

	return oasConfigApplyResult(res)
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminClients(
	ctx context.Context, req *oas.AppClientDesiredState, params oas.PutV1ProjectsByProjectIdAdminClientsParams,
) (*oas.AppClientApplyResult, error) {
	if _, err := requireProjectAdmin(ctx, params.ProjectID); err != nil {
		return nil, err
	}

	clients := make([]domain.AdminAppClientDesired, 0, len(req.Clients))
	for i := range req.Clients {
		c := &req.Clients[i]
		clients = append(clients, domain.AdminAppClientDesired{
			ID:             c.ID.Or(""),
			Name:           c.Name.Or(""),
			Type:           string(c.Type.Or("")),
			RedirectURIs:   c.RedirectUris,
			AllowedOrigins: c.AllowedOrigins,
			Disabled:       c.Disabled.Or(false),

			PostLogoutRedirectURIs: c.PostLogoutRedirectUris,
			BackchannelLogoutURI:   c.BackchannelLogoutURI.Or(""),
			TokenProfileID:         c.TokenProfileID.Or(""),
			Scopes:                 c.Scopes,
			JWKS:                   c.Jwks.Or(""),
			JWKSURI:                c.JwksURI.Or(""),
		})
	}

	res, err := s.deps.Apps.Apply(ctx, domain.AdminAppsApplyCmd{
		ProjectID:   params.ProjectID,
		Environment: params.XEnvironment.Or(""),
		Clients:     clients,
		Prune:       params.Prune.Or(false),
		DryRun:      params.DryRun.Or(false),
	})
	if err != nil {
		return nil, err
	}

	return oasAppClientsApplyResult(res), nil
}

// oasAppClient maps a domain AppClient onto the generated oas.AppClient.
func oasAppClient(a *domain.AppClient) oas.AppClient {
	out := oas.AppClient{
		ID:             oas.NewOptString(a.ID),
		Name:           oas.NewOptString(a.Name),
		Environment:    oas.NewOptString(a.Environment),
		RedirectUris:   a.RedirectURIs,
		AllowedOrigins: a.AllowedOrigins,
		Disabled:       oas.NewOptBool(a.Disabled),
		// A dynamically registered client holds a registration access token and
		// can rewrite its own metadata; the console says so rather than letting
		// an operator wonder where the client came from.
		DynamicallyRegistered: oas.NewOptBool(a.RegistrationTokenHash != ""),

		PostLogoutRedirectUris: a.PostLogoutRedirectURIs,
		Scopes:                 a.Scopes,
	}
	if a.JWKS != "" {
		out.Jwks = oas.NewOptNilString(a.JWKS)
	}

	if a.JWKSURI != "" {
		out.JwksURI = oas.NewOptNilString(a.JWKSURI)
	}

	if a.BackchannelLogoutURI != "" {
		out.BackchannelLogoutURI = oas.NewOptNilString(a.BackchannelLogoutURI)
	}

	if a.TokenProfileID != "" {
		out.TokenProfileID = oas.NewOptNilString(a.TokenProfileID)
	}

	if a.Type != "" {
		out.Type = oas.NewOptAppClientType(oas.AppClientType(a.Type))
	}

	return out
}

// ----- config document round-tripping -----

// jxEncoder/jxDecoder are the two halves of the ogen JSON codec every generated
// schema type implements. We use them to round-trip fully-typed configuration
// objects through the opaque domain.AdminConfigDoc carried to the adapter.
type (
	jxEncoder interface{ Encode(e *jx.Encoder) }
	jxDecoder interface{ Decode(d *jx.Decoder) error }
)

// oasEncodeConfig encodes a typed oas value and re-parses it into a flat map of
// raw JSON fields (the domain.AdminConfigDoc shape).
func oasEncodeConfig(v jxEncoder) domain.AdminConfigDoc {
	var e jx.Encoder
	v.Encode(&e)

	out := domain.AdminConfigDoc{}

	d := jx.DecodeBytes(e.Bytes())
	if err := d.Obj(func(d *jx.Decoder, key string) error {
		raw, err := d.Raw()
		if err != nil {
			return err
		}

		out[key] = raw

		return nil
	}); err != nil {
		return domain.AdminConfigDoc{}
	}

	return out
}

// oasDecodeConfig rebuilds a JSON object from a domain.AdminConfigDoc and
// decodes it into the supplied typed oas value.
func oasDecodeConfig(doc domain.AdminConfigDoc, dst jxDecoder) error {
	var e jx.Encoder
	e.ObjStart()

	for k, raw := range doc {
		e.FieldStart(k)
		e.Raw(raw)
	}

	e.ObjEnd()

	return dst.Decode(jx.DecodeBytes(e.Bytes()))
}

// oasAdminAccessRequest maps a domain access request to its wire form.
func oasAdminAccessRequest(ar *domain.CoreAuthAccessRequest) oas.AccessRequest {
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

// oasAdminProvider maps a domain notification provider onto oas.EmailProvider.
func oasAdminEmailProvider(p *domain.AdminProvider) oas.EmailProvider {
	out := oas.EmailProvider{
		ID:      oas.NewOptString(p.ID),
		Type:    p.Type,
		Enabled: oas.NewOptBool(p.Enabled),
	}
	if len(p.Config) > 0 {
		out.Config = oas.NewOptEmailProviderConfig(oas.EmailProviderConfig(p.Config))
	}

	return out
}

// oasAdminSmsProvider maps a domain notification provider onto oas.SmsProvider.
func oasAdminSmsProvider(p *domain.AdminProvider) oas.SmsProvider {
	out := oas.SmsProvider{
		ID:      oas.NewOptString(p.ID),
		Type:    p.Type,
		Enabled: oas.NewOptBool(p.Enabled),
	}
	if len(p.Config) > 0 {
		out.Config = oas.NewOptSmsProviderConfig(oas.SmsProviderConfig(p.Config))
	}

	return out
}

// oasAdminSigningKey maps a domain signing key onto oas.SigningKey.
func oasAdminSigningKey(k *domain.AdminSigningKey) oas.SigningKey {
	out := oas.SigningKey{
		Kid: oas.NewOptString(k.Kid),
		Alg: oas.NewOptString(k.Alg),
		Use: oas.NewOptString(k.Use),
	}
	if k.Status != "" {
		out.Status = oas.NewOptSigningKeyStatus(oas.SigningKeyStatus(k.Status))
	}

	if !k.CreatedAt.IsZero() {
		out.CreatedAt = oas.NewOptTimestamp(oas.Timestamp(k.CreatedAt))
	}

	return out
}

// oasAdminTokenProfile maps a domain token profile onto oas.TokenProfile.
func oasAdminTokenProfile(p *domain.AdminTokenProfile) oas.TokenProfile {
	out := oas.TokenProfile{
		ID:       oas.NewOptString(p.ID),
		Name:     oas.NewOptString(p.Name),
		Audience: oas.NewOptString(p.Audience),
	}
	if p.AccessTTL != 0 {
		out.AccessTTL = oas.NewOptInt(p.AccessTTL)
	}

	if p.RefreshTTL != 0 {
		out.RefreshTTL = oas.NewOptInt(p.RefreshTTL)
	}

	if len(p.ClaimsTemplate) > 0 {
		out.ClaimsTemplate = oas.NewOptTokenProfileClaimsTemplate(oas.TokenProfileClaimsTemplate(p.ClaimsTemplate))
	}

	return out
}

// adminCfg builds the get-command shared by every config read.
func adminCfg(projectID string, env oas.OptString) domain.AdminConfigGetCmd {
	return domain.AdminConfigGetCmd{ProjectID: projectID, Environment: env.Or("")}
}
