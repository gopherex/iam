package domain

import (
	"time"

	"github.com/go-faster/jx"
)

// AdminConfigDoc is an opaque, fully-typed configuration document carried
// between the API layer and the adapter as a map of raw JSON fields. Services
// never interpret it: the adapter validates/persists, the API layer round-trips
// it to/from the matching generated oas type. Used for auth / password-policy /
// session-policy / consent configuration objects.
type AdminConfigDoc = map[string]jx.Raw

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

// ----- Configuration (auth / password / session / consent) -----

// AdminConfigGetCmd selects a configuration document for read.
type AdminConfigGetCmd struct {
	ProjectID   string
	Environment string
}

// AdminConfigUpdateCmd carries a (partial or full) configuration document to
// apply. Doc is the wire object the admin sent, opaque to the service.
type AdminConfigUpdateCmd struct {
	ProjectID   string
	Environment string
	Doc         AdminConfigDoc
}

// ----- Notification providers (email / sms) -----

// AdminProvider is a configured notification provider (email or SMS). Config is
// an opaque settings bag the adapter interprets.
type AdminProvider struct {
	ID      string
	Type    string
	Config  map[string]jx.Raw
	Enabled bool
}

// AdminProviderCmd creates or replaces a notification provider.
type AdminProviderCmd struct {
	ProjectID   string
	Environment string
	ID          string // empty on create
	Type        string
	Config      map[string]jx.Raw
	Enabled     bool
}

// AdminProviderDeleteCmd removes a notification provider.
type AdminProviderDeleteCmd struct {
	ProjectID   string
	Environment string
	ID          string
}

// ----- Email templates / i18n / features -----

// AdminTemplateUpdateCmd patches an email template.
type AdminTemplateUpdateCmd struct {
	ProjectID   string
	Environment string
	TemplateID  string
	Patch       map[string]jx.Raw
}

// AdminTemplatePreviewCmd renders an email template with sample data.
type AdminTemplatePreviewCmd struct {
	ProjectID   string
	Environment string
	TemplateID  string
	Locale      string
	Data        map[string]jx.Raw
}

// AdminTemplatePreview is a rendered email template.
type AdminTemplatePreview struct {
	Subject string
	HTML    string
	Text    string
}

// AdminTemplateSendTestCmd sends a rendered template to a test address.
type AdminTemplateSendTestCmd struct {
	ProjectID   string
	Environment string
	TemplateID  string
	To          string
	Locale      string
	Data        map[string]jx.Raw
}

// AdminI18nUpdateCmd replaces the i18n bundle for a locale.
type AdminI18nUpdateCmd struct {
	ProjectID   string
	Environment string
	Locale      string
	Messages    map[string]jx.Raw
}

// AdminFeaturesUpdateCmd toggles project feature flags.
type AdminFeaturesUpdateCmd struct {
	ProjectID   string
	Environment string
	Features    map[string]bool
}

// ----- Signing keys (JWKS) / token profiles -----

// AdminSigningKey is a JWKS signing key in its administrative form.
type AdminSigningKey struct {
	Kid       string
	Alg       string
	Use       string
	Status    string
	CreatedAt time.Time
}

// AdminJWKSRotateCmd rotates the signing key set, optionally activating the new
// key immediately.
type AdminJWKSRotateCmd struct {
	ProjectID   string
	Environment string
	Activate    bool
}

// AdminTokenProfile is a token-shaping profile.
type AdminTokenProfile struct {
	ID             string
	Name           string
	Audience       string
	AccessTTL      int
	RefreshTTL     int
	ClaimsTemplate map[string]jx.Raw
}

// AdminTokenProfileCmd creates or replaces a token profile.
type AdminTokenProfileCmd struct {
	ProjectID   string
	Environment string
	ID          string // empty on create
	Profile     AdminConfigDoc
}

// AdminTokenProfilePreviewCmd renders the claims a profile would produce for a
// user.
type AdminTokenProfilePreviewCmd struct {
	ProjectID   string
	Environment string
	ProfileID   string
	UserID      string
}

// ----- Access requests -----

// AdminAccessRequestListCmd lists pending access requests.
type AdminAccessRequestListCmd struct {
	ProjectID   string
	Environment string
	Status      string
	Cursor      string
}

// AdminAccessRequestPage is a page of access requests.
type AdminAccessRequestPage struct {
	Items      []CoreAuthAccessRequest
	NextCursor string
	HasMore    bool
}

// AdminAccessRequestDecisionCmd approves or denies an access request.
type AdminAccessRequestDecisionCmd struct {
	ProjectID   string
	Environment string
	RequestID   string
	ActorID     string
	Reason      string
}
