package domain

import "time"

// Flow is the server-side resumable auth flow aggregate. Clients hold only the
// opaque FlowToken; all state lives here (§4 data model).
type Flow struct {
	ID        string
	ProjectID string
	Kind      FlowKind
	Status    FlowStatus
	Step      FlowStep
	UserID    string    // set once the user is created / resolved
	ExpiresAt time.Time // whole-flow TTL (30m)
	CreatedAt time.Time
	UpdatedAt time.Time

	// Data fields (stored in the jsonb `data` column).
	Contact          FlowContact
	Collected        FlowCollected
	ActiveChallenge  *FlowActiveChallenge
	ConsentsRequired []FlowConsentRef
	RegistrationMode string // open | request_access | invite_only | closed
	Error            *FlowError
}

// FlowKind is the auth flow category, selecting the state machine.
type FlowKind string

const (
	FlowKindSignup      FlowKind = "signup"
	FlowKindSignin      FlowKind = "signin"
	FlowKindRecovery    FlowKind = "recovery"
	FlowKindEmailChange FlowKind = "email_change"
)

// FlowStatus is the top-level lifecycle state.
type FlowStatus string

const (
	FlowStatusPending   FlowStatus = "pending"
	FlowStatusCompleted FlowStatus = "completed"
	FlowStatusExpired   FlowStatus = "expired"
	FlowStatusAborted   FlowStatus = "aborted"
)

// FlowStep is the current step the client must action.
type FlowStep string

const (
	FlowStepCollectCredentials FlowStep = "collect_credentials"
	FlowStepVerifyEmail        FlowStep = "verify_email"
	FlowStepVerifyPhone        FlowStep = "verify_phone"
	FlowStepSetPassword        FlowStep = "set_password"
	FlowStepMFARequired        FlowStep = "mfa_required"
	FlowStepStepUp             FlowStep = "step_up"
	FlowStepAcceptConsents     FlowStep = "accept_consents"
	FlowStepRequestAccess      FlowStep = "request_access"
	FlowStepAwaitingApproval   FlowStep = "awaiting_approval"
	FlowStepCompleted          FlowStep = "completed"
	FlowStepBlocked            FlowStep = "blocked"
)

// FlowContact holds the raw contact data (server-only; clients see only masked
// versions via FlowState).
type FlowContact struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// FlowCollected holds the non-secret credentials collected so far. Passwords are
// bcrypt-hashed at the point of consumption; only `has_password` is recorded
// here (§5 rule 5).
type FlowCollected struct {
	Name        string `json:"name,omitempty"`
	HasPassword bool   `json:"has_password,omitempty"`
}

// FlowActiveChallenge is the inline challenge metadata embedded in the flow's
// data jsonb. The authoritative challenge lives in iam_challenges; this carries
// only the display / gate fields.
type FlowActiveChallenge struct {
	ChallengeID  string    `json:"challenge_id"`
	Channel      string    `json:"channel"` // email | sms | whatsapp
	ExpiresAt    time.Time `json:"expires_at"`
	ResendAt     time.Time `json:"resend_at"`
	AttemptsLeft int       `json:"attempts_left"`
}

// FlowConsentRef is a consent document reference embedded in the flow data.
type FlowConsentRef struct {
	Key     string `json:"key"`
	Version string `json:"version"`
}

// FlowError carries the last stable error code visible to the client (§8). The
// flow stays pending; the client branches on Code.
type FlowError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FlowState is the full view returned to the caller. It carries the plain-text
// token (returned from the engine exactly once per rotation, never persisted).
type FlowState struct {
	FlowToken string
	Flow      *Flow
	Session   *Session // non-nil only when status=completed
}

// ─── commands ─────────────────────────────────────────────────────────────────

// FlowCreateCmd creates a new server-side resumable auth flow.
type FlowCreateCmd struct {
	ProjectID    string
	Kind         FlowKind
	Email        string
	Password     string
	Name         string
	CaptchaToken string
}

// FlowGetCmd retrieves a live flow by its opaque token (project-scoped).
type FlowGetCmd struct {
	ProjectID string
	FlowToken string
}

// FlowSubmitCmd submits the current step, advancing the state machine.
type FlowSubmitCmd struct {
	ProjectID string
	FlowToken string
	Action    string
	Payload   map[string]string
}

// FlowResendCmd re-issues the active challenge on a live flow.
type FlowResendCmd struct {
	ProjectID string
	FlowToken string
}

// FlowAbandonCmd marks a live flow as aborted.
type FlowAbandonCmd struct {
	ProjectID string
	FlowToken string
}
