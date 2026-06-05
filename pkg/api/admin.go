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

type AdminUsers interface {
	List(ctx context.Context, projectID string) ([]domain.Account, error)
	Get(ctx context.Context, projectID, accountID string) (*domain.Account, error)
	Create(ctx context.Context, cmd domain.RegisterCmd) (*domain.Account, error)
	Ban(ctx context.Context, projectID, accountID string) error
	Delete(ctx context.Context, projectID, accountID string) error
}

type AdminApps interface {
	List(ctx context.Context, projectID string) ([]domain.AppClient, error)
	Create(ctx context.Context, cmd domain.AppClientCmd) (*domain.AppClient, error)
	Get(ctx context.Context, projectID, appID string) (*domain.AppClient, error)
	Delete(ctx context.Context, projectID, appID string) error
}

// AdminDeps are the per-project administration ports. Config (auth/policy/
// providers/webhooks/keys/risk/jobs) is added as those surfaces are implemented.
type AdminDeps struct {
	Users AdminUsers
	Apps  AdminApps
}

// AdminService implements the AdminHandler slice of oas.Handler.
type AdminService struct {
	oas.UnimplementedHandler
	deps AdminDeps
}

// NewAdminService builds the Admin service from its dependencies.
func NewAdminService(deps AdminDeps) *AdminService { return &AdminService{deps: deps} }

var _ oas.Handler = (*AdminService)(nil)

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminAppsByAppIdSecretsBySecretIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminEmailProvidersById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminEmailProvidersByIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminJwksByKeyId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminJwksByKeyIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminSmsProvidersById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminSmsProvidersByIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminTokenProfilesById(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminTokenProfilesByIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesByIdentityIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) DeleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionId(ctx context.Context, params oas.DeleteV1ProjectsByProjectIdAdminUsersByUserIdSessionsBySessionIdParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAccessRequests(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAccessRequestsParams) (r *oas.GetV1ProjectsByProjectIdAdminAccessRequestsOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminApps(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAppsParams) (r *oas.GetV1ProjectsByProjectIdAdminAppsOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminAppsByAppIdParams) (r *oas.GetV1ProjectsByProjectIdAdminAppsByAppIdOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigAuth(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigAuthParams) (r *oas.AuthConfig, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigPasswordPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigPasswordPolicyParams) (r *oas.PasswordPolicy, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConfigSessionPolicy(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConfigSessionPolicyParams) (r *oas.SessionPolicy, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminConsents(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminConsentsParams) (r *oas.ConsentConfig, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminEmailProviders(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminEmailProvidersParams) (r *oas.GetV1ProjectsByProjectIdAdminEmailProvidersOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminEmailTemplates(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminEmailTemplatesParams) (r oas.GetV1ProjectsByProjectIdAdminEmailTemplatesOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminFeatures(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminFeaturesParams) (r oas.GetV1ProjectsByProjectIdAdminFeaturesOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminI18nByLocale(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminI18nByLocaleParams) (r oas.GetV1ProjectsByProjectIdAdminI18nByLocaleOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminJwks(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminJwksParams) (r *oas.GetV1ProjectsByProjectIdAdminJwksOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminSmsProviders(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminSmsProvidersParams) (r *oas.GetV1ProjectsByProjectIdAdminSmsProvidersOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminTokenProfiles(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminTokenProfilesParams) (r *oas.GetV1ProjectsByProjectIdAdminTokenProfilesOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsers(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersParams) (r *oas.GetV1ProjectsByProjectIdAdminUsersOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdParams) (r *oas.GetV1ProjectsByProjectIdAdminUsersByUserIdOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdIdentities(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesParams) (r *oas.GetV1ProjectsByProjectIdAdminUsersByUserIdIdentitiesOK, _ error) {
	panic("implement me")
}

func (s *AdminService) GetV1ProjectsByProjectIdAdminUsersByUserIdSessions(ctx context.Context, params oas.GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsParams) (r *oas.GetV1ProjectsByProjectIdAdminUsersByUserIdSessionsOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminAppsByAppId(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdReq, params oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdParams) (r *oas.PatchV1ProjectsByProjectIdAdminAppsByAppIdOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigAuth(ctx context.Context, req *oas.AuthConfig, params oas.PatchV1ProjectsByProjectIdAdminConfigAuthParams) (r *oas.AuthConfig, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigPasswordPolicy(ctx context.Context, req *oas.PasswordPolicy, params oas.PatchV1ProjectsByProjectIdAdminConfigPasswordPolicyParams) (r *oas.PasswordPolicy, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminConfigSessionPolicy(ctx context.Context, req *oas.SessionPolicy, params oas.PatchV1ProjectsByProjectIdAdminConfigSessionPolicyParams) (r *oas.SessionPolicy, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminEmailProvidersById(ctx context.Context, req *oas.EmailProvider, params oas.PatchV1ProjectsByProjectIdAdminEmailProvidersByIdParams) (r *oas.EmailProvider, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminEmailTemplatesById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdReq, params oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdParams) (r oas.PatchV1ProjectsByProjectIdAdminEmailTemplatesByIdOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminSmsProvidersById(ctx context.Context, req *oas.SmsProvider, params oas.PatchV1ProjectsByProjectIdAdminSmsProvidersByIdParams) (r *oas.SmsProvider, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminTokenProfilesById(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdReq, params oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdParams) (r *oas.PatchV1ProjectsByProjectIdAdminTokenProfilesByIdOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PatchV1ProjectsByProjectIdAdminUsersByUserId(ctx context.Context, req oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdReq, params oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdParams) (r *oas.PatchV1ProjectsByProjectIdAdminUsersByUserIdOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAccessRequestsByIdApprove(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveReq, params oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveParams) (r oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdApproveOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAccessRequestsByIdDeny(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyReq, params oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyParams) (r *oas.PostV1ProjectsByProjectIdAdminAccessRequestsByIdDenyOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminApps(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAppsReq, params oas.PostV1ProjectsByProjectIdAdminAppsParams) (r *oas.PostV1ProjectsByProjectIdAdminAppsCreated, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminAppsByAppIdSecrets(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsReq, params oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsParams) (r *oas.PostV1ProjectsByProjectIdAdminAppsByAppIdSecretsCreated, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailProviders(ctx context.Context, req *oas.EmailProvider, params oas.PostV1ProjectsByProjectIdAdminEmailProvidersParams) (r *oas.EmailProvider, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreview(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewReq, params oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewParams) (r *oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdPreviewOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTest(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTestReq, params oas.PostV1ProjectsByProjectIdAdminEmailTemplatesByIdSendTestParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminJwksByKeyIdActivate(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminJwksByKeyIdActivateParams) (r *oas.PostV1ProjectsByProjectIdAdminJwksByKeyIdActivateOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminJwksRotate(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminJwksRotateReq, params oas.PostV1ProjectsByProjectIdAdminJwksRotateParams) (r *oas.PostV1ProjectsByProjectIdAdminJwksRotateOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminSmsProviders(ctx context.Context, req *oas.SmsProvider, params oas.PostV1ProjectsByProjectIdAdminSmsProvidersParams) (r *oas.SmsProvider, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminTokenProfiles(ctx context.Context, req *oas.TokenProfile, params oas.PostV1ProjectsByProjectIdAdminTokenProfilesParams) (r *oas.PostV1ProjectsByProjectIdAdminTokenProfilesCreated, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreview(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewReq, params oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewParams) (r *oas.PostV1ProjectsByProjectIdAdminTokenProfilesByIdPreviewOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsers(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersReq, params oas.PostV1ProjectsByProjectIdAdminUsersParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersCreated, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdAnonymize(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminUsersByUserIdAnonymizeReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdAnonymizeParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdBan(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdBanOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdExport(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdExportParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdExportOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonate(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdImpersonateOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdPassword(ctx context.Context, req *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdPasswordReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdPasswordParams) (r *oas.Ok, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevoke(ctx context.Context, req oas.OptPostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeReq, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdSessionsRevokeOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdUnban(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdUnbanParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdUnbanOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmail(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmailParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyEmailOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhone(ctx context.Context, params oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhoneParams) (r *oas.PostV1ProjectsByProjectIdAdminUsersByUserIdVerifyPhoneOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminConsents(ctx context.Context, req *oas.ConsentConfig, params oas.PutV1ProjectsByProjectIdAdminConsentsParams) (r *oas.ConsentConfig, _ error) {
	panic("implement me")
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminFeatures(ctx context.Context, req oas.PutV1ProjectsByProjectIdAdminFeaturesReq, params oas.PutV1ProjectsByProjectIdAdminFeaturesParams) (r oas.PutV1ProjectsByProjectIdAdminFeaturesOK, _ error) {
	panic("implement me")
}

func (s *AdminService) PutV1ProjectsByProjectIdAdminI18nByLocale(ctx context.Context, req oas.PutV1ProjectsByProjectIdAdminI18nByLocaleReq, params oas.PutV1ProjectsByProjectIdAdminI18nByLocaleParams) (r oas.PutV1ProjectsByProjectIdAdminI18nByLocaleOK, _ error) {
	panic("implement me")
}
