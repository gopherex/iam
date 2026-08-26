package domain

import (
	"strings"
	"time"
)

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

// Error-detail keys: which field was wrong, what it held, and what it accepts.
const (
	detailField   = "field"
	detailValue   = "value"
	detailAllowed = "allowed"
)

// AdminRiskRule is a declarative risk rule an admin defines. Condition is a
// free-form expression evaluated by the risk engine; Action is what to do when
// it matches (require_step_up / block / notify / allow).
type AdminRiskRule struct {
	ID   string
	Name string
	// Signal names the condition the rule fires on, from RiskSignals. Condition
	// is its released spelling and is accepted as an alias on the way in, so a
	// rule written against the older shape keeps working.
	Signal    string
	Condition string
	Action    string
	Enabled   bool
}

// EffectiveSignal is the signal this rule fires on, preferring the current
// field and falling back to the released one.
func (r AdminRiskRule) EffectiveSignal() string {
	if r.Signal != "" {
		return r.Signal
	}

	return r.Condition
}

// Validate rejects a rule that could never fire: an unknown signal or an
// unknown action. Storing one would be a control an administrator believes is
// in place and is not.
func (r AdminRiskRule) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return ErrValidation.WithMessage("name is required")
	}

	signal := r.EffectiveSignal()
	if !RiskSignals.Has(signal) {
		return ErrValidation.WithDetails(map[string]any{
			detailField: "signal", detailValue: signal, detailAllowed: RiskSignals.List(),
		}).WithMessage("unknown risk signal")
	}

	if !RiskActions.Has(r.Action) {
		return ErrValidation.WithDetails(map[string]any{
			detailField: "action", detailValue: r.Action, detailAllowed: RiskActions.List(),
		}).WithMessage("unknown risk action")
	}

	return nil
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
