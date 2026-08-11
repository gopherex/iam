package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/gopherex/xlog"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

const (
	jobsWorkerInterval = 5 * time.Second
	auditExportMaxRows = 50000
)

// RunJobsWorker drains pending async jobs (audit exports, user imports) one at a
// time and runs them to completion. Blocks until ctx is cancelled. Wire it as a
// background worker alongside the GC and webhook-retry workers.
func (db *DB) RunJobsWorker(ctx context.Context, interval time.Duration, log *xlog.Logger) {
	if interval <= 0 {
		interval = jobsWorkerInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for db.drainOneJob(ctx, log) {
			}
		}
	}
}

// drainOneJob claims and runs a single pending job. Returns true if a job was
// processed (so the caller loops until the queue is empty).
func (db *DB) drainOneJob(ctx context.Context, log *xlog.Logger) bool {
	var (
		id, typ, projectID string
		raw                []byte
	)

	// Atomic claim: one pending job, skip rows another instance is running.
	err := db.Pool.QueryRow(ctx,
		`UPDATE iam_jobs SET status = $1, updated_at = now()
		 WHERE id = (
		     SELECT id FROM iam_jobs WHERE status = $2 ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, type, project_id, data`,
		jobStatusRunning, jobStatusPending).Scan(&id, &typ, &projectID, &raw)
	if err != nil {
		return false // no pending job (ErrNoRows) or a transient error
	}

	var d jobData

	_ = json.Unmarshal(raw, &d)

	var (
		result map[string]any
		perr   error
	)

	switch typ {
	case "audit_export":
		result, perr = db.processAuditExport(ctx, projectID, d.Spec)
	case "import_users":
		result, perr = db.processImportUsers(ctx, projectID, d.Spec)
	default:
		perr = fmt.Errorf("unknown job type %q", typ)
	}

	db.finishJob(ctx, id, d, result, perr, log)

	return true
}

func (db *DB) finishJob(ctx context.Context, id string, d jobData, result map[string]any, perr error, log *xlog.Logger) {
	status := jobStatusCompleted
	if perr != nil {
		status = jobStatusFailed
		d.Error = perr.Error()

		log.Warn("job failed", xlog.String("job_id", id), xlog.Error("err", perr))
	}

	d.Result = result

	raw, err := json.Marshal(d)
	if err != nil {
		return
	}

	if _, err := db.Pool.Exec(ctx,
		`UPDATE iam_jobs SET status = $1, data = $2, updated_at = now() WHERE id = $3 AND status = $4`,
		status, raw, id, jobStatusRunning); err != nil {
		log.Warn("job finalize failed", xlog.String("job_id", id), xlog.Error("err", err))
	}
}

// processAuditExport materializes the project's audit logs (optionally within a
// time window) as a JSON array and records the row count. The bytes are stored
// in the job result so exports/{job_id} can hand them back.
func (db *DB) processAuditExport(ctx context.Context, projectID string, spec map[string]any) (map[string]any, error) {
	conds := []string{"project_id = $1"}
	args := []any{projectID}

	if from, ok := specTime(spec, "from"); ok {
		args = append(args, from)
		conds = append(conds, fmt.Sprintf("at >= $%d", len(args)))
	}

	if to, ok := specTime(spec, "to"); ok {
		args = append(args, to)
		conds = append(conds, fmt.Sprintf("at <= $%d", len(args)))
	}

	args = append(args, auditExportMaxRows)

	query := `SELECT id, type, actor_id, target_id, at, data FROM iam_audit_logs WHERE ` +
		condsJoin(conds) + fmt.Sprintf(" ORDER BY at DESC LIMIT $%d", len(args))

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type entry struct {
		ID       string          `json:"id"`
		Type     string          `json:"type"`
		ActorID  *string         `json:"actor_id"`
		TargetID *string         `json:"target_id"`
		At       time.Time       `json:"at"`
		Data     json.RawMessage `json:"data"`
	}

	var out []entry

	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.Type, &e.ActorID, &e.TargetID, &e.At, &e.Data); err != nil {
			return nil, err
		}

		out = append(out, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	blob, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"count":        len(out),
		"content_type": "application/json",
		"download_url": "data:application/json;base64," + base64Std(blob),
	}, nil
}

// processImportUsers creates an account per entry in the spec's "users" list.
// A pre-hashed bcrypt password ("$2...") is stored as-is; any other format is
// rejected for that row. Duplicate emails and invalid rows are counted, not
// fatal — the job completes with per-row tallies.
func (db *DB) processImportUsers(ctx context.Context, projectID string, spec map[string]any) (map[string]any, error) {
	usersAny, _ := spec["users"].([]any)
	format, _ := spec["password_hash_format"].(string)

	var processed, failed int

	rowErrors := make([]string, 0)

	for _, ua := range usersAny {
		u, ok := ua.(map[string]any)
		if !ok {
			failed++

			continue
		}

		email, _ := u["email"].(string)
		name, _ := u["name"].(string)
		hash, _ := u["password_hash"].(string)

		if err := db.importOneUser(ctx, projectID, email, name, hash, format); err != nil {
			failed++

			if len(rowErrors) < 100 {
				rowErrors = append(rowErrors, email+": "+err.Error())
			}

			continue
		}

		processed++
	}

	return map[string]any{
		"processed": processed,
		"failed":    failed,
		"errors":    rowErrors,
	}, nil
}

func (db *DB) importOneUser(ctx context.Context, projectID, email, name, hash, format string) error {
	if err := domain.ValidateEmail(email); err != nil {
		return err
	}
	// A supplied password hash must be bcrypt — the only format the auth engine
	// verifies. Reject anything else rather than store an unusable credential.
	if hash != "" {
		if format != "" && format != "bcrypt" {
			return fmt.Errorf("unsupported password_hash_format %q", format)
		}

		if len(hash) < 4 || hash[0] != '$' || hash[1] != '2' {
			return domain.ErrValidation.WithMessage("password_hash must be bcrypt ($2...)")
		}
	}

	return db.withTx(ctx, func(ctx context.Context) error {
		env, err := effectiveEnv(ctx, db, projectID, coreAuthDefaultEnv)
		if err != nil {
			return err
		}

		acc := domain.Account{
			ID: newUUID(), ProjectID: projectID, Kind: coreAuthKindHuman,
			Status: "active", PrimaryEmail: email, Name: name,
			CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
		}

		rawAcc, err := marshal(acc)
		if err != nil {
			return err
		}

		rmAcc := json.RawMessage(rawAcc)
		now := nowUTC()
		emailPtr := ptr(null.From(email))

		if _, err := models.IamUsers.Insert(&models.IamUserSetter{
			ID: &acc.ID, ProjectID: &acc.ProjectID, Environment: &env,
			Kind: ptr(acc.Kind), Status: ptr(acc.Status), PrimaryEmail: emailPtr,
			CreatedAt: &now, UpdatedAt: &now, Data: &rmAcc,
		}).One(ctx, db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}

			return err
		}

		if hash == "" {
			return nil
		}

		credID := newUUID()
		credType := coreAuthCredentialPassword

		credRaw, err := marshal(coreAuthCredential{
			ID: credID, ProjectID: projectID, UserID: acc.ID, Type: credType, Hash: hash,
			CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return err
		}

		rmCred := json.RawMessage(credRaw)
		if _, err := models.IamCredentials.Insert(&models.IamCredentialSetter{
			ID: &credID, ProjectID: &projectID, Environment: &env, UserID: &acc.ID,
			Type: &credType, Secret: &hash, CreatedAt: &now, UpdatedAt: &now, Data: &rmCred,
		}).One(ctx, db.Bobx()); err != nil {
			return err
		}

		return nil
	})
}

func specTime(spec map[string]any, key string) (time.Time, bool) {
	s, ok := spec[key].(string)
	if !ok || s == "" {
		return time.Time{}, false
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}

func condsJoin(conds []string) string {
	out := ""

	var outSb309 strings.Builder
	for i, c := range conds {
		if i > 0 {
			outSb309.WriteString(" AND ")
		}

		outSb309.WriteString(c)
	}
	out += outSb309.String()

	return out
}

func base64Std(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
