package domain

// MFA command/value types. Names are prefixed with the MFA aggregate to avoid
// collisions with other services' command types.

// MFAEmailEnrollCmd enrolls an email factor and issues a delivery challenge.
type MFAEmailEnrollCmd struct {
	AccountID string
	Email     string
}

// MFASmsEnrollCmd enrolls an SMS factor and issues a delivery challenge.
type MFASmsEnrollCmd struct {
	AccountID string
	Phone     string
}

// MFATotpVerifyCmd activates a pending TOTP factor by verifying a code.
type MFATotpVerifyCmd struct {
	AccountID string
	FactorID  string
	Code      string
}

// MFAWebAuthnEnrollOptionsCmd requests creation options for a new WebAuthn
// factor enrollment.
type MFAWebAuthnEnrollOptionsCmd struct {
	AccountID string
	Name      string
}

// MFAWebAuthnEnrollVerifyCmd verifies the attestation produced by the
// authenticator and activates the WebAuthn factor.
type MFAWebAuthnEnrollVerifyCmd struct {
	AccountID   string
	ChallengeID string
	Credential  map[string]any
}

// MFARecoveryVerifyCmd consumes a recovery code to complete an MFA flow.
type MFARecoveryVerifyCmd struct {
	ProjectID string
	AccountID string
	FlowToken string
	Code      string
}
