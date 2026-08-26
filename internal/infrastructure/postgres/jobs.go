package postgres

import (
	"context"
	"encoding/json"
)

// Async job status values (iam_jobs.status).
const (
	jobStatusPending   = "pending"
	jobStatusRunning   = "running"
	jobStatusCompleted = "completed"
	jobStatusFailed    = "failed"
	jobStatusCancelled = "cancelled"
)

// jobData is the iam_jobs.data envelope: the input spec, progress, and a result
// reference or error once the worker has run it.
// jobFieldStatus is the status key inside a job's envelope.
const jobFieldStatus = "status"

type jobData struct {
	Spec     map[string]any `json:"spec,omitempty"`
	Progress int            `json:"progress,omitempty"`
	Result   map[string]any `json:"result,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// createJob inserts a pending async job and returns its id and status. The jobs
// worker drains pending jobs and runs them. Uses the pool for a standalone
// insert so the job survives even if a surrounding request transaction rolls
// back (the caller has already decided to enqueue the work).
func createJob(ctx context.Context, db *DB, projectID, typ string, spec map[string]any) (string, string, error) {
	id := newUUID()

	raw, err := json.Marshal(jobData{Spec: spec})
	if err != nil {
		return "", "", err
	}

	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO iam_jobs (id, project_id, type, status, data) VALUES ($1, $2, $3, $4, $5)`,
		id, projectID, typ, jobStatusPending, raw); err != nil {
		return "", "", err
	}

	return id, jobStatusPending, nil
}
