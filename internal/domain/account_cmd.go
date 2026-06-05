package domain

import "time"

// Account command/value types for the user-self slice of the Account aggregate.
// Names are prefixed with the aggregate (Account*) to avoid collisions with
// command types owned by other service slices.

// AccountActivityEvent is a single audit/activity record on the account.
type AccountActivityEvent struct {
	ID     string
	Type   string
	IP     string
	Device string
	At     time.Time
}

// AccountActivityCmd queries the account activity log.
type AccountActivityCmd struct {
	AccountID string
	Type      string
	Cursor    string
	Limit     int
}

// AccountActivityPage is a cursor-paginated slice of activity events.
type AccountActivityPage struct {
	Events     []AccountActivityEvent
	NextCursor string
	HasMore    bool
}

// AccountConsent is a recorded consent acceptance on the account.
type AccountConsent struct {
	Key        string
	Version    string
	URL        string
	AcceptedAt time.Time
	Locale     string
}

// AccountConsentAcceptance records acceptance of one consent document.
type AccountConsentAcceptance struct {
	Key     string
	Version string
}

// AccountAcceptConsentsCmd accepts one or more consent documents.
type AccountAcceptConsentsCmd struct {
	AccountID string
	Accept    []AccountConsentAcceptance
}

// AccountExportJob describes a pending or finished data-export job.
type AccountExportJob struct {
	JobID       string
	Status      string
	DownloadURL string
}

// AccountRenameSessionCmd sets a human-friendly device name on a session.
type AccountRenameSessionCmd struct {
	AccountID  string
	SessionID  string
	DeviceName string
}

// AccountTrustSessionCmd marks a session as trusted for a duration.
type AccountTrustSessionCmd struct {
	AccountID       string
	SessionID       string
	DurationSeconds int
}

// AccountRevokeSessionsCmd bulk-revokes sessions for the account.
type AccountRevokeSessionsCmd struct {
	AccountID       string
	ExceptSessionID string // if set, this session is kept
	ExceptCurrent   bool
}

// AccountMergeStartCmd begins linking/merging another identity into the account.
type AccountMergeStartCmd struct {
	AccountID        string
	TargetIdentifier string
}

// AccountMergeConfirmCmd confirms a pending identity merge with a challenge code.
type AccountMergeConfirmCmd struct {
	AccountID   string
	ChallengeID string
	Code        string
}
