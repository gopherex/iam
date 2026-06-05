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

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

type adminUsers interface {
	List(ctx context.Context, projectID string) ([]domain.Account, error)
	Get(ctx context.Context, projectID, accountID string) (*domain.Account, error)
	Create(ctx context.Context, cmd domain.RegisterCmd) (*domain.Account, error)
	Ban(ctx context.Context, projectID, accountID string) error
	Delete(ctx context.Context, projectID, accountID string) error
}

type adminApps interface {
	List(ctx context.Context, projectID string) ([]domain.AppClient, error)
	Create(ctx context.Context, cmd domain.AppClientCmd) (*domain.AppClient, error)
	Get(ctx context.Context, projectID, appID string) (*domain.AppClient, error)
	Delete(ctx context.Context, projectID, appID string) error
}

// AdminDeps are the per-project administration ports. Config (auth/policy/
// providers/webhooks/keys/risk/jobs) is added as those surfaces are implemented.
type AdminDeps struct {
	Users adminUsers
	Apps  adminApps
}

// AdminService implements the AdminHandler slice of oas.Handler.
type AdminService struct {
	oas.UnimplementedHandler
	deps AdminDeps
}

// NewAdminService builds the Admin service from its dependencies.
func NewAdminService(deps AdminDeps) *AdminService { return &AdminService{deps: deps} }

var _ oas.Handler = (*AdminService)(nil)

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminEmailProvidersById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminEmailProvidersByIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminEmailProvidersByIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminJwksByKeyId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminJwksByKeyIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminJwksByKeyIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminSmsProvidersById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminSmsProvidersByIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminSmsProvidersByIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminTokenProfilesById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminTokenProfilesByIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminTokenProfilesByIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionIdParams) (r oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAccessRequests(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAccessRequestsParams) (r oas.GetV1ProjectsByProjectIdAdminAccessRequestsRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminApps(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAppsParams) (r oas.GetV1ProjectsByProjectIdAdminAppsRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAppsByAppIdParams) (r oas.GetV1ProjectsByProjectIdAdminAppsByAppIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigAuth(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigAuthParams) (r oas.GetV1ProjectsByProjectIdAdminConfigAuthRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigPasswordPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigPasswordPolicyParams) (r oas.GetV1ProjectsByProjectIdAdminConfigPasswordPolicyRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigSessionPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigSessionPolicyParams) (r oas.GetV1ProjectsByProjectIdAdminConfigSessionPolicyRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConsents(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConsentsParams) (r oas.GetV1ProjectsByProjectIdAdminConsentsRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminEmailProviders(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminEmailProvidersParams) (r oas.GetV1ProjectsByProjectIdAdminEmailProvidersRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminEmailTemplates(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminEmailTemplatesParams) (r oas.GetV1ProjectsByProjectIdAdminEmailTemplatesRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminFeatures(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminFeaturesParams) (r oas.GetV1ProjectsByProjectIdAdminFeaturesRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminI18nByLocale(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminI18nByLocaleParams) (r oas.GetV1ProjectsByProjectIdAdminI18nByLocaleRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminJwks(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminJwksParams) (r oas.GetV1ProjectsByProjectIdAdminJwksRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminSmsProviders(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSmsProvidersParams) (r oas.GetV1ProjectsByProjectIdAdminSmsProvidersRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminTokenProfiles(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminTokenProfilesParams) (r oas.GetV1ProjectsByProjectIdAdminTokenProfilesRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsers(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersParams) (r oas.GetV1ProjectsByProjectIdAdminUsersRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdParams) (r oas.GetV1ProjectsByProjectIdAdminUsersByUserIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdIdentities(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesParams) (r oas.GetV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesRes, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdSessions(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsParams) (r oas.GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdReq, params oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdParams) (r oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigAuth(ctx context.Context, req *oas.AuthConfig, params oas.PatchV1ProjectsByProjectIdAdminConfigAuthParams) (r oas.PatchV1ProjectsByProjectIdAdminConfigAuthRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigPasswordPolicy(ctx context.Context, req *oas.PasswordPolicy, params oas.PatchV1ProjectsByProjectIdAdminConfigPasswordPolicyParams) (r oas.PatchV1ProjectsByProjectIdAdminConfigPasswordPolicyRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigSessionPolicy(ctx context.Context, req *oas.SessionPolicy, params oas.PatchV1ProjectsByProjectIdAdminConfigSessionPolicyParams) (r oas.PatchV1ProjectsByProjectIdAdminConfigSessionPolicyRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminEmailProvidersById(ctx context.Context, req *oas.EmailProvider, params oas.PatchV1ProjectsByProjectIdAdminEmailProvidersByIdParams) (r oas.PatchV1ProjectsByProjectIdAdminEmailProvidersByIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminEmailTemplatesById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdReq, params oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdParams) (r oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminSmsProvidersById(ctx context.Context, req *oas.SmsProvider, params oas.PatchV1ProjectsByProjectIdAdminSmsProvidersByIdParams) (r oas.PatchV1ProjectsByProjectIdAdminSmsProvidersByIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminTokenProfilesById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdReq, params oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdParams) (r oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdReq, params oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdParams) (r oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAccessRequestsByIdApprove(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveReq, params oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveParams) (r oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAccessRequestsByIdDeny(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyReq, params oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyParams) (r oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminApps(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAppsReq, params oas.PostV1ProjectsByProjectIdAdminAppsParams) (r oas.PostV1ProjectsByProjectIdAdminAppsRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAppsByAppIdSecrets(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsReq, params oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsParams) (r oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailProviders(ctx context.Context, req *oas.EmailProvider, params oas.PostV1ProjectsByProjectIdAdminEmailProvidersParams) (r oas.PostV1ProjectsByProjectIdAdminEmailProvidersRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewReq, params oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewParams) (r oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTest(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTestReq, params oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTestParams) (r oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTestRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminJwksByKeyIdActivate(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminJwksByKeyIdActivateParams) (r oas.PostV1ProjectsByProjectIdAdminJwksByKeyIdActivateRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminJwksRotate(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminJwksRotateReq, params oas.PostV1ProjectsByProjectIdAdminJwksRotateParams) (r oas.PostV1ProjectsByProjectIdAdminJwksRotateRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminSmsProviders(ctx context.Context, req *oas.SmsProvider, params oas.PostV1ProjectsByProjectIdAdminSmsProvidersParams) (r oas.PostV1ProjectsByProjectIdAdminSmsProvidersRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminTokenProfiles(ctx context.Context, req *oas.TokenProfile, params oas.PostV1ProjectsByProjectIdAdminTokenProfilesParams) (r oas.PostV1ProjectsByProjectIdAdminTokenProfilesRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreview(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewReq, params oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewParams) (r oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsers(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersReq, params oas.PostV1ProjectsByProjectIdAdminUsersParams) (r oas.PostV1ProjectsByProjectIdAdminUsersRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdAnonymize(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminUsersByUserIdAnonymizeReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdAnonymizeParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdAnonymizeRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdBan(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdExport(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdExportParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdExportRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonate(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdPassword(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdPasswordReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdPasswordParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdPasswordRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevoke(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdUnban(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdUnbanParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdUnbanRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmail(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmailParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmailRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhone(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhoneParams) (r oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhoneRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminConsents(ctx context.Context, req *oas.ConsentConfig, params oas.PutV1ProjectsByProjectIdAdminConsentsParams) (r oas.PutV1ProjectsByProjectIdAdminConsentsRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminFeatures(ctx context.Context, req oas.PutV1ProjectsByProjectIdAdminFeaturesReq, params oas.PutV1ProjectsByProjectIdAdminFeaturesParams) (r oas.PutV1ProjectsByProjectIdAdminFeaturesRes, _ error) {
	panic("implement me")
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminI18nByLocale(ctx context.Context, req oas.PutV1ProjectsByProjectIdAdminI18nByLocaleReq, params oas.PutV1ProjectsByProjectIdAdminI18nByLocaleParams) (r oas.PutV1ProjectsByProjectIdAdminI18nByLocaleRes, _ error) {
	panic("implement me")
}
