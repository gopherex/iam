package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-faster/jx"
	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
)

const (
	auditDefaultLimit = 50
	auditMaxLimit     = 200
)

// pgAudit is the Postgres-backed admin audit-log reader plus the writer used by
// the auditing emitter. Rows live in iam_audit_logs (project-scoped, append-only
// via the emitter; pruned by the retention policy / GC).
type pgAudit struct {
	db      *DB
	emitter Emitter
}

// NewPgAudit builds the audit adapter.
func NewPgAudit(db *DB, emitter Emitter) *pgAudit {
	return &pgAudit{db: db, emitter: emitter}
}

// record inserts one audit row. Called by the auditing emitter on the caller's
// transaction, so it commits iff the audited mutation commits.
func (a *pgAudit) record(ctx context.Context, projectID, typ, actorID, targetID string, data json.RawMessage) error {
	if len(data) == 0 {
		data = json.RawMessage(`{}`)
	}

	_, err := a.db.TxDB.Exec(ctx,
		`INSERT INTO iam_audit_logs (id, project_id, type, actor_id, target_id, at, data)
		 VALUES ($1, $2, $3, $4, $5, now(), $6)`,
		newUUID(), projectID, typ, nullIfEmpty(actorID), nullIfEmpty(targetID), data)
	if err != nil {
		return fmt.Errorf("record audit log: %w", err)
	}

	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}

	return s
}

// List returns a page of audit entries (newest first) plus the next cursor.
func (a *pgAudit) List(ctx context.Context, cmd domain.AuditLogListCmd) ([]domain.AuditLogEntry, string, bool, error) {
	query, args, limit := buildAuditListQuery(cmd)

	rows, err := a.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	return scanAuditListPage(rows, limit)
}

// buildAuditListQuery turns the list filters (project, type/actor/target,
// keyset cursor) into a bounded SQL query, clamping the page size and asking
// for one extra row so the caller can tell whether a next page exists.
func buildAuditListQuery(cmd domain.AuditLogListCmd) (string, []any, int) {
	limit := cmd.Limit
	if limit <= 0 {
		limit = auditDefaultLimit
	}

	if limit > auditMaxLimit {
		limit = auditMaxLimit
	}

	conds := []string{"project_id = $1"}
	args := []any{cmd.ProjectID}

	add := func(col, val string) {
		if val != "" {
			args = append(args, val)
			conds = append(conds, fmt.Sprintf("%s = $%d", col, len(args)))
		}
	}

	add("type", cmd.Type)
	add("actor_id", cmd.ActorID)
	add("target_id", cmd.TargetID)

	if cmd.Cursor != "" {
		at, id, ok := decodeAuditCursor(cmd.Cursor)
		if ok {
			args = append(args, at)
			atArg := len(args)
			args = append(args, id)
			idArg := len(args)
			conds = append(conds, fmt.Sprintf("(at, id) < ($%d, $%d)", atArg, idArg))
		}
	}

	args = append(args, limit+1)

	query := `SELECT id, type, actor_id, target_id, at, data FROM iam_audit_logs WHERE ` +
		strings.Join(conds, " AND ") +
		fmt.Sprintf(" ORDER BY at DESC, id DESC LIMIT $%d", len(args))

	return query, args, limit
}

// scanAuditListPage reads up to limit+1 rows, trims the lookahead row, and
// derives the next-page cursor from the last row kept.
func scanAuditListPage(rows pgx.Rows, limit int) ([]domain.AuditLogEntry, string, bool, error) {
	out := make([]domain.AuditLogEntry, 0, limit)

	for rows.Next() {
		var (
			e             domain.AuditLogEntry
			actor, target *string
			raw           []byte
		)

		if err := rows.Scan(&e.ID, &e.Type, &actor, &target, &e.At, &raw); err != nil {
			return nil, "", false, fmt.Errorf("scan audit log: %w", err)
		}

		if actor != nil {
			e.ActorID = *actor
		}

		if target != nil {
			e.TargetID = *target
		}

		e.Data = decodeAuditData(raw)
		out = append(out, e)
	}

	if err := rows.Err(); err != nil {
		return nil, "", false, fmt.Errorf("list audit logs: rows: %w", err)
	}

	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}

	next := ""

	if hasMore && len(out) > 0 {
		last := out[len(out)-1]
		next = encodeAuditCursor(last.At, last.ID)
	}

	return out, next, hasMore, nil
}

// Get returns a single audit entry by id, scoped to the project.
func (a *pgAudit) Get(ctx context.Context, projectID, id string) (*domain.AuditLogEntry, error) {
	var (
		e             domain.AuditLogEntry
		actor, target *string
		raw           []byte
	)

	err := a.db.Pool.QueryRow(ctx,
		`SELECT id, type, actor_id, target_id, at, data FROM iam_audit_logs WHERE id = $1 AND project_id = $2`,
		id, projectID).Scan(&e.ID, &e.Type, &actor, &target, &e.At, &raw)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	if actor != nil {
		e.ActorID = *actor
	}

	if target != nil {
		e.TargetID = *target
	}

	e.Data = decodeAuditData(raw)

	return &e, nil
}

// CreateExport records an export job over the requested window and returns its
// id and status. The job is drained by the jobs worker.
func (a *pgAudit) CreateExport(ctx context.Context, cmd domain.AuditExportCmd) (string, string, error) {
	format := cmd.Format
	if format == "" {
		format = "json"
	}

	spec := map[string]any{
		"kind":   "audit_export",
		"format": format,
	}
	if !cmd.From.IsZero() {
		spec["from"] = cmd.From.UTC().Format(time.RFC3339)
	}

	if !cmd.To.IsZero() {
		spec["to"] = cmd.To.UTC().Format(time.RFC3339)
	}

	return createJob(ctx, a.db, cmd.ProjectID, "audit_export", spec)
}

func decodeAuditData(raw []byte) map[string]jx.Raw {
	if len(raw) == 0 {
		return nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}

	return jsonToRaw(m)
}

func encodeAuditCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(at.UnixNano(), 10) + "|" + id))
}

// auditCursorParts is the "<unix_nano>|<id>" cursor's field count.
const auditCursorParts = 2

func decodeAuditCursor(cur string) (time.Time, string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(cur)
	if err != nil {
		return time.Time{}, "", false
	}

	parts := strings.SplitN(string(b), "|", auditCursorParts)
	if len(parts) != auditCursorParts {
		return time.Time{}, "", false
	}

	ns, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", false
	}

	return time.Unix(0, ns).UTC(), parts[1], true
}
