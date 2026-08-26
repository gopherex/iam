package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-faster/jx"
	"golang.org/x/crypto/bcrypt"

	"github.com/gopherex/iam/internal/domain"
)

// pgJobs is the admin async-jobs adapter (list/get/cancel) over iam_jobs. Jobs
// are created via createJob and drained by the jobs worker (jobs_worker.go).
type pgJobs struct {
	db      *DB
	emitter Emitter
}

// NewPgJobs builds the jobs adapter.
func NewPgJobs(db *DB, emitter Emitter) *pgJobs {
	return &pgJobs{db: db, emitter: emitter}
}

// List returns the project's jobs, newest first (bounded).
func (a *pgJobs) List(ctx context.Context, projectID string, limit int) ([]domain.AdminJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := a.db.Pool.Query(ctx,
		`SELECT id, type, status, data FROM iam_jobs WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2`,
		projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	out := make([]domain.AdminJob, 0, limit)

	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}

		out = append(out, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan jobs: %w", err)
	}

	return out, nil
}

// Get returns a single job by id, project-scoped.
func (a *pgJobs) Get(ctx context.Context, projectID, id string) (*domain.AdminJob, error) {
	row := a.db.Pool.QueryRow(ctx,
		`SELECT id, type, status, data FROM iam_jobs WHERE id = $1 AND project_id = $2`, id, projectID)

	job, err := scanJob(row)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return &job, nil
}

// Cancel marks a still-pending/running job cancelled. Completed/failed jobs are
// left unchanged (nothing to cancel).
func (a *pgJobs) Cancel(ctx context.Context, projectID, id string) error {
	n, err := a.db.Pool.Exec(ctx,
		`UPDATE iam_jobs SET status = $1, updated_at = now()
		 WHERE id = $2 AND project_id = $3 AND status IN ($4, $5)`,
		jobStatusCancelled, id, projectID, jobStatusPending, jobStatusRunning)
	if err != nil {
		return fmt.Errorf("cancel job: %w", err)
	}

	if n.RowsAffected() == 0 {
		// Either not found or in a terminal state.
		if _, gErr := a.Get(ctx, projectID, id); gErr != nil {
			return domain.ErrNotFound
		}
	}

	return nil
}

// CreateImportUsers enqueues a bulk user-import job. The users list is stored in
// the job spec (converting jx.Raw values so they persist as raw JSON, not
// base64) and processed by the jobs worker.
func (a *pgJobs) CreateImportUsers(
	ctx context.Context,
	projectID string,
	users []map[string]jx.Raw,
	format string,
	sendInvites bool,
) (string, string, error) {
	converted := make([]any, 0, len(users))
	for _, u := range users {
		converted = append(converted, rawToJSON(u))
	}

	spec := map[string]any{
		"users":                converted,
		"password_hash_format": format,
		"send_invites":         sendInvites,
	}

	return createJob(ctx, a.db, projectID, "import_users", spec)
}

// VerifyPasswordHash reports whether password matches hash for the given format.
// Only bcrypt is supported (the format the auth engine can verify).
func (a *pgJobs) VerifyPasswordHash(hash, password, format string) (bool, error) {
	if format != "" && format != "bcrypt" {
		return false, domain.ErrValidation.WithMessage("unsupported password hash format: " + format)
	}

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil, nil
}

func scanJob(r rowScanner) (domain.AdminJob, error) {
	var (
		job domain.AdminJob
		raw []byte
	)

	if err := r.Scan(&job.ID, &job.Type, &job.Status, &raw); err != nil {
		return domain.AdminJob{}, err
	}

	if len(raw) > 0 {
		var d jobData
		if err := json.Unmarshal(raw, &d); err == nil {
			job.Progress = d.Progress
			job.Result = d.Result
			job.Error = d.Error
		}
	}

	return job, nil
}
