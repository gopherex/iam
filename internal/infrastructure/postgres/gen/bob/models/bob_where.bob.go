// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

var (
	SelectWhere     = Where[*dialect.SelectQuery]()
	UpdateWhere     = Where[*dialect.UpdateQuery]()
	DeleteWhere     = Where[*dialect.DeleteQuery]()
	OnConflictWhere = Where[*clause.ConflictClause]() // Used in ON CONFLICT DO UPDATE
)

func Where[Q psql.Filterable]() struct {
	IamUsers               iamUserWhere[Q]
	IamCredentials         iamCredentialWhere[Q]
	IamIdentities          iamIdentityWhere[Q]
	IamSessions            iamSessionWhere[Q]
	IamRefreshTokens       iamRefreshTokenWhere[Q]
	IamFactors             iamFactorWhere[Q]
	IamWebauthnCredentials iamWebauthnCredentialWhere[Q]
	IamRecoveryCodes       iamRecoveryCodeWhere[Q]
	IamChallenges          iamChallengeWhere[Q]
	IamFlows               iamFlowWhere[Q]
	IamConsents            iamConsentWhere[Q]
	IamServiceAccounts     iamServiceAccountWhere[Q]
	IamAPIKeys             iamAPIKeyWhere[Q]
	IamAppClients          iamAppClientWhere[Q]
	IamAppSecrets          iamAppSecretWhere[Q]
	IamSsoConnections      iamSsoConnectionWhere[Q]
	IamDomains             iamDomainWhere[Q]
	IamScimTokens          iamScimTokenWhere[Q]
	IamScimResources       iamScimResourceWhere[Q]
	IamOauthGrants         iamOauthGrantWhere[Q]
	IamInteractions        iamInteractionWhere[Q]
	IamAuthCodes           iamAuthCodeWhere[Q]
	IamParRequests         iamParRequestWhere[Q]
	IamDeviceCodes         iamDeviceCodeWhere[Q]
	IamProjects            iamProjectWhere[Q]
	IamEnvironments        iamEnvironmentWhere[Q]
	IamSigningKeys         iamSigningKeyWhere[Q]
	IamTokenProfiles       iamTokenProfileWhere[Q]
	IamAdminTokens         iamAdminTokenWhere[Q]
	IamConfigs             iamConfigWhere[Q]
	IamProviders           iamProviderWhere[Q]
	IamEmailTemplates      iamEmailTemplateWhere[Q]
	IamWebhooks            iamWebhookWhere[Q]
	IamHooks               iamHookWhere[Q]
	IamJobs                iamJobWhere[Q]
	IamAuditLogs           iamAuditLogWhere[Q]
	IamAccessRequests      iamAccessRequestWhere[Q]
	IamRiskRules           iamRiskRuleWhere[Q]
	IamBlocks              iamBlockWhere[Q]
	IamActivities          iamActivityWhere[Q]
	IamEvents              iamEventWhere[Q]
} {
	return struct {
		IamUsers               iamUserWhere[Q]
		IamCredentials         iamCredentialWhere[Q]
		IamIdentities          iamIdentityWhere[Q]
		IamSessions            iamSessionWhere[Q]
		IamRefreshTokens       iamRefreshTokenWhere[Q]
		IamFactors             iamFactorWhere[Q]
		IamWebauthnCredentials iamWebauthnCredentialWhere[Q]
		IamRecoveryCodes       iamRecoveryCodeWhere[Q]
		IamChallenges          iamChallengeWhere[Q]
		IamFlows               iamFlowWhere[Q]
		IamConsents            iamConsentWhere[Q]
		IamServiceAccounts     iamServiceAccountWhere[Q]
		IamAPIKeys             iamAPIKeyWhere[Q]
		IamAppClients          iamAppClientWhere[Q]
		IamAppSecrets          iamAppSecretWhere[Q]
		IamSsoConnections      iamSsoConnectionWhere[Q]
		IamDomains             iamDomainWhere[Q]
		IamScimTokens          iamScimTokenWhere[Q]
		IamScimResources       iamScimResourceWhere[Q]
		IamOauthGrants         iamOauthGrantWhere[Q]
		IamInteractions        iamInteractionWhere[Q]
		IamAuthCodes           iamAuthCodeWhere[Q]
		IamParRequests         iamParRequestWhere[Q]
		IamDeviceCodes         iamDeviceCodeWhere[Q]
		IamProjects            iamProjectWhere[Q]
		IamEnvironments        iamEnvironmentWhere[Q]
		IamSigningKeys         iamSigningKeyWhere[Q]
		IamTokenProfiles       iamTokenProfileWhere[Q]
		IamAdminTokens         iamAdminTokenWhere[Q]
		IamConfigs             iamConfigWhere[Q]
		IamProviders           iamProviderWhere[Q]
		IamEmailTemplates      iamEmailTemplateWhere[Q]
		IamWebhooks            iamWebhookWhere[Q]
		IamHooks               iamHookWhere[Q]
		IamJobs                iamJobWhere[Q]
		IamAuditLogs           iamAuditLogWhere[Q]
		IamAccessRequests      iamAccessRequestWhere[Q]
		IamRiskRules           iamRiskRuleWhere[Q]
		IamBlocks              iamBlockWhere[Q]
		IamActivities          iamActivityWhere[Q]
		IamEvents              iamEventWhere[Q]
	}{
		IamUsers:               buildIamUserWhere[Q](IamUsers.Columns),
		IamCredentials:         buildIamCredentialWhere[Q](IamCredentials.Columns),
		IamIdentities:          buildIamIdentityWhere[Q](IamIdentities.Columns),
		IamSessions:            buildIamSessionWhere[Q](IamSessions.Columns),
		IamRefreshTokens:       buildIamRefreshTokenWhere[Q](IamRefreshTokens.Columns),
		IamFactors:             buildIamFactorWhere[Q](IamFactors.Columns),
		IamWebauthnCredentials: buildIamWebauthnCredentialWhere[Q](IamWebauthnCredentials.Columns),
		IamRecoveryCodes:       buildIamRecoveryCodeWhere[Q](IamRecoveryCodes.Columns),
		IamChallenges:          buildIamChallengeWhere[Q](IamChallenges.Columns),
		IamFlows:               buildIamFlowWhere[Q](IamFlows.Columns),
		IamConsents:            buildIamConsentWhere[Q](IamConsents.Columns),
		IamServiceAccounts:     buildIamServiceAccountWhere[Q](IamServiceAccounts.Columns),
		IamAPIKeys:             buildIamAPIKeyWhere[Q](IamAPIKeys.Columns),
		IamAppClients:          buildIamAppClientWhere[Q](IamAppClients.Columns),
		IamAppSecrets:          buildIamAppSecretWhere[Q](IamAppSecrets.Columns),
		IamSsoConnections:      buildIamSsoConnectionWhere[Q](IamSsoConnections.Columns),
		IamDomains:             buildIamDomainWhere[Q](IamDomains.Columns),
		IamScimTokens:          buildIamScimTokenWhere[Q](IamScimTokens.Columns),
		IamScimResources:       buildIamScimResourceWhere[Q](IamScimResources.Columns),
		IamOauthGrants:         buildIamOauthGrantWhere[Q](IamOauthGrants.Columns),
		IamInteractions:        buildIamInteractionWhere[Q](IamInteractions.Columns),
		IamAuthCodes:           buildIamAuthCodeWhere[Q](IamAuthCodes.Columns),
		IamParRequests:         buildIamParRequestWhere[Q](IamParRequests.Columns),
		IamDeviceCodes:         buildIamDeviceCodeWhere[Q](IamDeviceCodes.Columns),
		IamProjects:            buildIamProjectWhere[Q](IamProjects.Columns),
		IamEnvironments:        buildIamEnvironmentWhere[Q](IamEnvironments.Columns),
		IamSigningKeys:         buildIamSigningKeyWhere[Q](IamSigningKeys.Columns),
		IamTokenProfiles:       buildIamTokenProfileWhere[Q](IamTokenProfiles.Columns),
		IamAdminTokens:         buildIamAdminTokenWhere[Q](IamAdminTokens.Columns),
		IamConfigs:             buildIamConfigWhere[Q](IamConfigs.Columns),
		IamProviders:           buildIamProviderWhere[Q](IamProviders.Columns),
		IamEmailTemplates:      buildIamEmailTemplateWhere[Q](IamEmailTemplates.Columns),
		IamWebhooks:            buildIamWebhookWhere[Q](IamWebhooks.Columns),
		IamHooks:               buildIamHookWhere[Q](IamHooks.Columns),
		IamJobs:                buildIamJobWhere[Q](IamJobs.Columns),
		IamAuditLogs:           buildIamAuditLogWhere[Q](IamAuditLogs.Columns),
		IamAccessRequests:      buildIamAccessRequestWhere[Q](IamAccessRequests.Columns),
		IamRiskRules:           buildIamRiskRuleWhere[Q](IamRiskRules.Columns),
		IamBlocks:              buildIamBlockWhere[Q](IamBlocks.Columns),
		IamActivities:          buildIamActivityWhere[Q](IamActivities.Columns),
		IamEvents:              buildIamEventWhere[Q](IamEvents.Columns),
	}
}
