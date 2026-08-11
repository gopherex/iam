package domain

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
