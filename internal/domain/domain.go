// Package domain holds the IAM domain model: aggregate roots and the command
// value-objects services pass to the aggregate ports.
//
// Architecture (decided 2026-06-05):
//   - An aggregate root is the transactional consistency boundary. Each port
//     method is ONE atomic business operation; the persistence adapter opens
//     the transaction INSIDE the method — services never touch pgtx.
//   - API services (pkg/api) are pure orchestration: map oas -> domain, call
//     one or more aggregate-port methods, map domain -> oas. They hold only
//     interfaces (consumer-defined, next to each service).
//   - Cross-aggregate consistency is out of scope here (events/outbox come
//     later); a single call mutates a single aggregate.
//
// Fields are representative; they grow as operations are implemented.
package domain

import "time"

// ===== Account aggregate =====
// Root of the user: profile, credentials, linked identities, MFA factors,
// passkeys, sessions, consents. All of these are consistency-bound to it.

type Account struct {
	ID            string
	ProjectID     string
	Kind          string // human | guest | system
	Status        string // active | suspended | banned | deactivated
	PrimaryEmail  string
	EmailVerified bool
	PrimaryPhone  string
	PhoneVerified bool
	Name          string
	Locale        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Session struct {
	ID           string
	AccountID    string
	ProjectID    string
	ClientID     string
	AMR          []string
	AAL          int
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	CreatedAt    time.Time
	// Device / management metadata (stored in the iam_sessions data envelope;
	// snake-cased so it round-trips with the rename/trust paths). Powers
	// self-managed-session UIs.
	DeviceName   string    `json:"device_name,omitempty"`
	IP           string    `json:"ip,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	Fingerprint  string    `json:"fingerprint,omitempty"`
	Trusted      bool      `json:"trusted,omitempty"`
	LastActiveAt time.Time `json:"last_active_at,omitempty"`
	// Current is a transient view flag (this session == the caller's) — never
	// persisted.
	Current bool `json:"-"`
}

type Identity struct {
	ID                string
	Type              string // password | oauth | saml | oidc | passkey
	Provider          string
	ProviderAccountID string
	Email             string
}

type Factor struct {
	ID     string
	Type   string // totp | sms | email | webauthn
	Status string // pending | active
	Hint   string
	// OTPAuthURL is the otpauth:// provisioning URL returned to the caller at
	// TOTP enrollment time so it can be rendered as a QR code. It is transient
	// provisioning material, not a stored credential (the shared secret lives
	// in iam_factors.secret), so it is excluded from the persisted aggregate.
	OTPAuthURL string `json:"-"`
}

type WebAuthnCredential struct {
	ID         string
	Name       string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

type Challenge struct {
	ID        string
	Type      string
	ExpiresAt time.Time
	// PublicKey carries WebAuthn / provisioning material when relevant.
	PublicKey map[string]any
}

// ===== Connection aggregate (federation) =====
// SSO connection plus the domains and SCIM provisioning bound to it.

type Connection struct {
	ID          string
	ProjectID   string
	Type        string // saml | oidc
	Name        string
	Status      string
	Domains     []string
	ExternalRef string
	// Config carries the protocol-specific provider settings (SAML IdP metadata
	// / certificate, OIDC issuer / client credentials). It is persisted whole in
	// the connection's jsonb envelope; see FederationSamlConfig / FederationOidcConfig.
	Config *FederationConnectionConfig `json:",omitempty"`
}

type Domain struct {
	ID           string
	ProjectID    string
	Domain       string
	Status       string // pending | verified
	ConnectionID string
}

// ===== Machine identity aggregate =====

type ServiceAccount struct {
	ID        string
	ProjectID string
	Name      string
	Scopes    []string
	Disabled  bool
}

type APIKey struct {
	ID        string
	ProjectID string
	Name      string
	Scopes    []string
	Prefix    string
	Disabled  bool
}

// ===== App client aggregate (admin) =====

type AppClient struct {
	ID           string
	ProjectID    string
	Name         string
	Type         string // spa | native | web | machine
	Environment  string
	RedirectURIs []string
	// AllowedOrigins are the browser origins (scheme://host[:port]) permitted to
	// make cross-origin calls to IAM for this client. The CORS layer reflects
	// ACAO for any origin in the union of all clients' AllowedOrigins.
	AllowedOrigins []string
}

// ===== Project aggregate (admin / operator) =====
// Project plus its environments, signing keys, auth config, webhooks, etc.

type Project struct {
	ID               string
	Name             string
	Slug             string
	DefaultLocale    string
	SupportedLocales []string
	Environments     []string
	CreatedAt        time.Time
}

type Environment struct {
	ProjectID string
	Name      string // live | test | staging
	Issuer    string
	CreatedAt time.Time
}

// ===== OIDC provider aggregate =====

type Grant struct {
	ID        string
	AccountID string
	ClientID  string
	Scopes    []string
	GrantedAt time.Time
}

type Interaction struct {
	ID          string
	ClientID    string
	Scopes      []string
	RedirectURI string
	Nonce       string
	SessionID   string
}

// ===== Read models =====

type OAuthProvider struct {
	ID     string
	Name   string
	Scopes []string
}

type PublicConfig struct {
	ProjectName      string
	Methods          []string
	Providers        []OAuthProvider
	Locales          []string
	DefaultLocale    string
	ConsentDocuments []ConsentDocument
	Registration     *RegistrationInfo
}

// RegistrationInfo is the public view of a project's signup policy, so the app
// can pre-block /register and show/hide the password field.
type RegistrationInfo struct {
	Mode             string // open | invite_only | request_access | closed
	PasswordStrategy string // password_first | after_verify
}

type ConsentDocument struct {
	Key      string
	Version  string
	Title    string
	Body     string
	Locale   string
	Required bool
	URL      string
}

// ===== Commands (multi-field inputs to aggregate operations) =====

type RegisterCmd struct {
	ProjectID   string
	Environment string
	Email       string
	Phone       string
	Password    string
	Name        string
	Locale      string
	Consents    []AccountConsentAcceptance
	// SkipConsentGate bypasses the up-front consent.required enforcement in
	// Register. The resumable signup flow sets this because it enforces consent
	// at its own accept_consents step (after identity proof), not at user
	// creation. The non-flow POST /v1/auth/sign-up path leaves it false.
	SkipConsentGate bool
}

type ProfileUpdateCmd struct {
	ProjectID string
	AccountID string
	Name      string
	AvatarURL string
	Locale    string
}

type ConnectionCmd struct {
	ProjectID string
	Type      string
	Name      string
	Domains   []string
}

type ServiceAccountCmd struct {
	ProjectID string
	Name      string
	Scopes    []string
}

type APIKeyCmd struct {
	ProjectID string
	Name      string
	Scopes    []string
}

type AppClientCmd struct {
	ProjectID      string
	Environment    string
	Name           string
	Type           string
	RedirectURIs   []string
	AllowedOrigins []string
}

type ProjectCmd struct {
	Name          string
	Slug          string
	DefaultLocale string
}

type EnvironmentCmd struct {
	ProjectID string
	Name      string
}
