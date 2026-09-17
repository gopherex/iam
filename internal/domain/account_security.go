package domain

import "time"

// SecurityScope is supplied by the transport, never by a request body.
type SecurityScope struct {
	ProjectID   string
	Environment string
	AccountID   string
	SessionID   string
	ActorID     string
}

type SecurityPolicy struct {
	Version                     int               `json:"version"`
	Mode                        string            `json:"mode"`
	ContinueURL                 string            `json:"continue_url"`
	ClientURLs                  map[string]string `json:"client_urls"`
	Notify                      bool              `json:"notify"`
	RequireNewDeviceProof       bool              `json:"require_new_device_proof"`
	FlowTTLSeconds              int               `json:"flow_ttl_seconds"`
	TrustTTLSeconds             int               `json:"trust_ttl_seconds"`
	RetentionDays               int               `json:"retention_days"`
	FailureThreshold            int               `json:"failure_threshold"`
	FailureWindowSeconds        int               `json:"failure_window_seconds"`
	NotificationCooldownSeconds int               `json:"notification_cooldown_seconds"`
}

type SecurityDevice struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	FirstSeenAt  time.Time  `json:"first_seen_at"`
	LastSeenAt   time.Time  `json:"last_seen_at"`
	TrustedUntil *time.Time `json:"trusted_until,omitempty"`
	Revoked      bool       `json:"revoked"`
	Current      bool       `json:"current"`
	SessionIDs   []string   `json:"session_ids"`
}

type SecurityDeviceRegistration struct {
	Device      SecurityDevice `json:"device"`
	DeviceToken string         `json:"device_token"`
}

type SecurityEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	At        time.Time `json:"at"`
	Outcome   string    `json:"outcome"`
	SessionID string    `json:"session_id,omitempty"`
	DeviceID  string    `json:"device_id,omitempty"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type SecurityIncident struct {
	ID                string          `json:"id"`
	AccountID         string          `json:"account_id"`
	Status            string          `json:"status"`
	Severity          string          `json:"severity"`
	Reasons           []string        `json:"reasons"`
	Events            []SecurityEvent `json:"events"`
	SessionID         string          `json:"session_id,omitempty"`
	DeviceID          string          `json:"device_id,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	PolicyVersion     int             `json:"policy_version"`
	RevokedSessionIDs []string        `json:"revoked_session_ids"`
	Resolution        string          `json:"resolution,omitempty"`
}

type SecurityCaseMessage struct {
	At      time.Time `json:"at"`
	Author  string    `json:"author"`
	Message string    `json:"message"`
}

type SecurityCase struct {
	ID            string                `json:"id"`
	AccountID     string                `json:"account_id,omitempty"`
	Status        string                `json:"status"`
	ContactMasked string                `json:"contact_masked"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	Messages      []SecurityCaseMessage `json:"messages"`
}

type SecurityDelivery struct {
	ID              string    `json:"id"`
	IncidentID      string    `json:"incident_id,omitempty"`
	CaseID          string    `json:"case_id,omitempty"`
	Channel         string    `json:"channel"`
	RecipientMasked string    `json:"recipient_masked"`
	Status          string    `json:"status"`
	Attempts        int       `json:"attempts"`
	LastError       string    `json:"last_error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SecurityFlowState carries only the restricted flow capability, never a session.
type SecurityFlowState struct {
	Deletion          *AccountDeletion      `json:"deletion,omitempty"`
	Sessions          []SecuritySession     `json:"sessions,omitempty"`
	Challenge         map[string]any        `json:"challenge,omitempty"`
	FlowToken         string                `json:"flow_token,omitempty"`
	Kind              string                `json:"kind"`
	Status            string                `json:"status"`
	Step              string                `json:"step"`
	Version           int                   `json:"version"`
	NextActions       []string              `json:"next_actions"`
	ExpiresAt         time.Time             `json:"expires_at"`
	Incident          *SecurityIncident     `json:"incident,omitempty"`
	Case              *SecurityCase         `json:"case,omitempty"`
	ContactMasked     string                `json:"contact_masked,omitempty"`
	AttemptsLeft      int                   `json:"attempts_left"`
	ResendAt          *time.Time            `json:"resend_at,omitempty"`
	Factors           []SecurityProofFactor `json:"factors,omitempty"`
	AllSessions       bool                  `json:"all_sessions"`
	RevokedSessionIDs []string              `json:"revoked_session_ids"`
	Outcome           string                `json:"outcome,omitempty"`
	ErrorCode         string                `json:"error_code,omitempty"`
}

type SecurityFlowInput struct {
	ExchangeKey       string         `json:"exchange_key,omitempty"`
	RequestID         string         `json:"request_id,omitempty"`
	DeviceID          string         `json:"device_id,omitempty"`
	Credential        map[string]any `json:"credential,omitempty"`
	Action            string         `json:"action,omitempty"`
	IncidentID        string         `json:"incident_id,omitempty"`
	SessionID         string         `json:"session_id,omitempty"`
	ContinuationToken string         `json:"continuation_token,omitempty"`
	Identifier        string         `json:"identifier,omitempty"`
	Contact           string         `json:"contact,omitempty"`
	Code              string         `json:"code,omitempty"`
	Password          string         `json:"password,omitempty"`
	NewPassword       string         `json:"new_password,omitempty"`
	FactorID          string         `json:"factor_id,omitempty"`
	Message           string         `json:"message,omitempty"`
	AllSessions       bool           `json:"all_sessions"`
	Version           int            `json:"version"`
}

type SecurityCaseDecision struct {
	Action   string `json:"action"`
	Message  string `json:"message"`
	Evidence string `json:"evidence"`
}

type SecurityProofFactor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Hint string `json:"hint"`
}

type SecuritySession struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
