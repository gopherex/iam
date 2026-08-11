package domain

import "time"

// AdminHook is a blocking auth hook: a signed HTTP callback invoked at an auth
// decision point (before_sign_in / before_token_issue / before_user_create). A
// non-2xx reply denies the action unless FailOpen is set. SigningSecret is
// returned only once, on create.
type AdminHook struct {
	ID            string
	Type          string
	URL           string
	TimeoutMs     int
	Enabled       bool
	FailOpen      bool
	SigningSecret string
}

// AdminJob is an async background job (bulk import, audit/data export) an admin
// enqueues and polls.
type AdminJob struct {
	ID     string
	Type   string
	Status string
	// Progress and Result mirror the job's data envelope.
	Progress int
	Result   map[string]any
	Error    string
}

// AdminRiskRule is a declarative risk rule an admin defines. Condition is a
// free-form expression evaluated by the risk engine; Action is what to do when
// it matches (require_step_up / block / notify / allow).
type AdminRiskRule struct {
	ID        string
	Name      string
	Condition string
	Action    string
	Enabled   bool
}

// AdminBlock is a manual rate-limit block on a subject (ip / email / phone /
// asn). A request whose subject matches an unexpired block is refused.
type AdminBlock struct {
	ID        string
	Type      string
	Value     string
	Reason    string
	ExpiresAt time.Time
}
