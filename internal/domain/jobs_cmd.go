package domain

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
