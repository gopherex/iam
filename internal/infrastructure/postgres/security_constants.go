package postgres

import "time"

// Internal security protocol values shared by detection, flows and delivery.
//
//nolint:gosec // Public protocol and event names; no credential material.
const (
	securitySetPassword           = "set_password"
	securitySucceeded             = "succeeded"
	securityAbandon               = "abandon"
	securityAccountID             = "account_id"
	securityAccountSecured        = "account_secured"
	securityApproved              = "approved"
	securityAwaitingSupport       = "awaiting_support"
	securityCases                 = "cases"
	securityCompleted             = "completed"
	securityDeliveries            = "deliveries"
	securityDevices               = "devices"
	securityDisabled              = "disabled"
	securityEmail                 = "email"
	securityEnforce               = "enforce"
	securityIncidents             = "incidents"
	securityNeedsInformation      = "needs_information"
	securityPasskey               = "passkey"
	securityPending               = "pending"
	securityRateLimited           = "rate_limited"
	securityRecovery              = "recovery"
	securityRequestSupport        = "request_support"
	securityResend                = "resend"
	securityRestoreAccess         = "restore_access"
	securityReview                = "review"
	securityCodeTemplate          = "security_code"
	securityRecoveryTemplate      = "security_recovery"
	securitySessionCreated        = "session.created"
	securitySessionDeviceMismatch = "session.device_mismatch"

	securityFlowTokenPrefix    = "sft_"
	securitySignInApproved     = "sign_in_approved"
	securitySignin             = "signin"
	securitySMS                = "sms"
	securitySupportGrant       = "support_grant"
	securityTokenReuseDetected = "token.reuse_detected"
	securityTrust              = "trust"
	securityVerifyContact      = "verify_contact"
	securityVerifyIdentity     = "verify_identity"
	securityVerifyMFA          = "verify_mfa"
	securityVerifyRecoveryCode = "verify_recovery_code"
)

const (
	securityTokenBytes                  = 32
	securityCodeAttempts                = 5
	securityCodeTTL                     = 10 * time.Minute
	securityProofRateWindow             = 15 * time.Minute
	securityRetryGrace                  = 30 * time.Second
	securityContinuationTTL             = 30 * time.Minute
	securityTOTPReplayWindow            = 90 * time.Second
	securityPersistenceTimeout          = 5 * time.Second
	securitySessionHistoryLimit         = 1000
	securityEventHistoryLimit           = 100
	securityMessageHistoryLimit         = 100
	securityPhoneVisibleDigits          = 4
	securityDefaultFlowSeconds          = 1800
	securityDefaultTrustSeconds         = 2592000
	securityDefaultRetentionDays        = 90
	securityDefaultFailureThreshold     = 5
	securityDefaultFailureWindowSeconds = 900
	securityDefaultNotificationCooldown = 1800
	securityDeliveryBurst               = 5
	securityFlowStartBurst              = 10
	securityProofBurst                  = 20
)
