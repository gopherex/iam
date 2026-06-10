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
type AdminConfig interface {
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

	// Email templates.
	ListEmailTemplates(ctx context.Context, cmd domain.AdminConfigGetCmd) (map[string]jx.Raw, error)
	UpdateEmailTemplate(ctx context.Context, cmd domain.AdminTemplateUpdateCmd) (map[string]jx.Raw, error)
	PreviewEmailTemplate(ctx context.Context, cmd domain.AdminTemplatePreviewCmd) (*domain.AdminTemplatePreview, error)
	SendTestEmail(ctx context.Context, cmd domain.AdminTemplateSendTestCmd) error
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

// AdminDeps are the per-project administration ports.
type AdminDeps struct {
	Users           AdminUsers
	Apps            AdminApps
	ServiceAccounts AdminServiceAccounts
	APIKeys         AdminAPIKeys
	Connections     AdminConnections
	Config          AdminConfig
	Keys            AdminKeys
	AccessRequests  AdminAccessRequests
	Invites         AdminInvites
}

// AdminService implements the AdminHandler slice of oas.Handler.
type AdminService struct {
	oas.UnimplementedHandler
	deps AdminDeps
}

// NewAdminService builds the Admin service from its dependencies.
func NewAdminService(deps AdminDeps) *AdminService { return &AdminService{deps: deps} }

var _ oas.Handler = (*AdminService)(nil)

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
		ProjectID:    params.ProjectID,
		Environment:  params.XEnvironment.Or("live"),
		Name:         req.Name,
		Type:         string(req.Type),
		RedirectURIs: req.RedirectUris,
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

// oasAppClient maps a domain AppClient onto the generated oas.AppClient.
func oasAppClient(a *domain.AppClient) oas.AppClient {
	out := oas.AppClient{
		ID:           oas.NewOptString(a.ID),
		Name:         oas.NewOptString(a.Name),
		Environment:  oas.NewOptString(a.Environment),
		RedirectUris: a.RedirectURIs,
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
type jxEncoder interface{ Encode(e *jx.Encoder) }
type jxDecoder interface{ Decode(d *jx.Decoder) error }

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
		out[key] = jx.Raw(raw)
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
