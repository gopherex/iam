package domain

import "time"

// Command value-objects for the Admin (per-project administration) surface.
// Names are prefixed with the aggregate to avoid collisions with other
// services' command types. Each maps onto exactly one aggregate-port method.

// ----- Users -----

// AdminUserUpdateCmd is a partial profile update applied by a project admin.
type AdminUserUpdateCmd struct {
	ProjectID string
	AccountID string
	Name      string
	Locale    string
}

// AdminUserBanCmd bans a user, optionally with a reason and an expiry.
type AdminUserBanCmd struct {
	ProjectID string
	AccountID string
	Reason    string
	Until     time.Time
}

// AdminUserPasswordCmd sets a user's password out-of-band.
type AdminUserPasswordCmd struct {
	ProjectID      string
	AccountID      string
	Password       string
	RevokeSessions bool
}

// AdminUserImpersonateCmd mints a time-boxed impersonation link.
type AdminUserImpersonateCmd struct {
	ProjectID       string
	AccountID       string
	ActorID         string // the admin performing the impersonation
	Reason          string
	DurationSeconds int
}

// AdminImpersonation is the result of an impersonation request.
type AdminImpersonation struct {
	URL       string
	ExpiresAt time.Time
}

// AdminUserSessionsRevokeCmd revokes a user's sessions, optionally keeping one.
type AdminUserSessionsRevokeCmd struct {
	ProjectID       string
	AccountID       string
	ExceptSessionID string
	Reason          string
}

// AdminUserAnonymizeCmd anonymizes (GDPR erase) a user.
type AdminUserAnonymizeCmd struct {
	ProjectID string
	AccountID string
	Reason    string
}

// ----- Service accounts -----

// AdminServiceAccountUpdateCmd updates a service account's scopes / state.
type AdminServiceAccountUpdateCmd struct {
	ProjectID        string
	ServiceAccountID string
	Scopes           []string
	Disabled         bool
}

// AdminServiceAccountSecretCmd issues a new secret for a service account.
type AdminServiceAccountSecretCmd struct {
	ProjectID        string
	ServiceAccountID string
	Name             string
	ExpiresAt        time.Time
}

// AdminSecret is a freshly minted machine credential. The plaintext secret is
// returned exactly once.
type AdminSecret struct {
	SecretID     string
	ClientID     string
	ClientSecret string
}

// ----- API keys -----

// AdminAPIKeyCmd creates a project API key.
type AdminAPIKeyCmd struct {
	ProjectID string
	Name      string
	Scopes    []string
	ExpiresAt time.Time
}

// AdminAPIKeyUpdateCmd updates an API key's name / scopes / state.
type AdminAPIKeyUpdateCmd struct {
	ProjectID string
	KeyID     string
	Name      string
	Scopes    []string
	Disabled  bool
}

// AdminAPIKeySecret is an API key plus its one-time plaintext secret.
type AdminAPIKeySecret struct {
	Key    *APIKey
	Secret string
}

// ----- SSO connections / domains -----

// AdminConnectionCmd creates an SSO connection.
type AdminConnectionCmd struct {
	ProjectID   string
	Type        string
	Name        string
	Domains     []string
	ExternalRef string
}

// AdminDomainCmd registers a verification domain.
type AdminDomainCmd struct {
	ProjectID    string
	Domain       string
	ConnectionID string
}

// AdminDomainRegistration is a domain plus the DNS record proving ownership.
type AdminDomainRegistration struct {
	Domain                  *Domain
	VerificationRecordType  string
	VerificationRecordName  string
	VerificationRecordValue string
}
