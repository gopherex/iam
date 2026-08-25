package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gopherex/xlog"
)

// gcDefaultInterval is how often the collector sweeps expired runtime rows.
const gcDefaultInterval = 10 * time.Minute

// gcSweep is one DELETE the garbage collector runs each pass. Every runtime
// table below carries an expires_at that is only ever filtered read-side, so
// without this the rows accumulate forever (iam_challenges is the highest-churn:
// one row per OTP / magic-link / verification send).
type gcSweep struct {
	table string
	sql   string
}

var gcSweeps = []gcSweep{
	{"iam_challenges", `DELETE FROM iam_challenges WHERE expires_at < now()`},
	{"iam_flows", `DELETE FROM iam_flows WHERE expires_at < now()`},
	{"iam_auth_codes", `DELETE FROM iam_auth_codes WHERE expires_at < now()`},
	{"iam_interactions", `DELETE FROM iam_interactions WHERE expires_at IS NOT NULL AND expires_at < now()`},
	{"iam_device_codes", `DELETE FROM iam_device_codes WHERE expires_at < now()`},
	{"iam_par_requests", `DELETE FROM iam_par_requests WHERE expires_at < now()`},
	// A revoked token stops needing a denylist entry once it has expired on its
	// own; keeping it would grow the table without adding any protection.
	{"iam_revoked_tokens", `DELETE FROM iam_revoked_tokens WHERE expires_at < now()`},
	// Sessions/refresh tokens have a nullable expires_at; a revoked-but-unexpired
	// refresh token is kept for reuse detection until it expires, so only prune
	// past-expiry rows (they can no longer be used or detected against).
	{"iam_sessions", `DELETE FROM iam_sessions WHERE expires_at IS NOT NULL AND expires_at < now()`},
	{"iam_refresh_tokens", `DELETE FROM iam_refresh_tokens WHERE expires_at IS NOT NULL AND expires_at < now()`},
}

// RunGarbageCollector periodically deletes expired runtime rows. It runs one
// sweep shortly after start (to trim any backlog a fresh deploy inherits) and
// then every interval, and blocks until ctx is cancelled. Wire it as a
// background worker alongside the outbox relay.
func (db *DB) RunGarbageCollector(ctx context.Context, interval time.Duration, log *xlog.Logger) {
	if interval <= 0 {
		interval = gcDefaultInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	db.gcSweepAll(ctx, log)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			db.gcSweepAll(ctx, log)
		}
	}
}

func (db *DB) gcSweepAll(ctx context.Context, log *xlog.Logger) {
	for _, s := range gcSweeps {
		tag, err := db.Pool.Exec(ctx, s.sql)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			log.Warn("gc sweep failed", xlog.String("table", s.table), xlog.Error("err", err))

			continue
		}

		if n := tag.RowsAffected(); n > 0 {
			log.Info("gc swept expired rows",
				xlog.String("table", s.table), xlog.String("rows", strconv.FormatInt(n, 10)))
		}
	}

	db.gcRetentionSweep(ctx, log)
}

// gcRetentionSweep prunes audit logs and events past each project's configured
// retention window (iam_config key=retention). audit_log_retention_days /
// event_retention_days of 0 or absent means "keep" (no pruning).
func (db *DB) gcRetentionSweep(ctx context.Context, log *xlog.Logger) {
	rows, err := db.Pool.Query(ctx, `SELECT project_id, data FROM iam_config WHERE key = 'retention'`)
	if err != nil {
		if ctx.Err() == nil {
			log.Warn("gc retention scan failed", xlog.Error("err", err))
		}

		return
	}

	type policy struct {
		project   string
		auditDays int
		eventDays int
	}

	var policies []policy

	for rows.Next() {
		var (
			pid string
			raw []byte
		)

		if err := rows.Scan(&pid, &raw); err != nil {
			continue
		}

		var d struct {
			AuditLogRetentionDays *int `json:"audit_log_retention_days"`
			EventRetentionDays    *int `json:"event_retention_days"`
		}

		_ = json.Unmarshal(raw, &d)

		p := policy{project: pid}
		if d.AuditLogRetentionDays != nil {
			p.auditDays = *d.AuditLogRetentionDays
		}

		if d.EventRetentionDays != nil {
			p.eventDays = *d.EventRetentionDays
		}

		policies = append(policies, p)
	}

	rows.Close()

	for _, p := range policies {
		if ctx.Err() != nil {
			return
		}

		if p.auditDays > 0 {
			_, _ = db.Pool.Exec(ctx,
				`DELETE FROM iam_audit_logs WHERE project_id = $1 AND at < now() - make_interval(days => $2)`,
				p.project, p.auditDays)
		}

		if p.eventDays > 0 {
			_, _ = db.Pool.Exec(ctx,
				`DELETE FROM iam_events WHERE project_id = $1 AND created_at < now() - make_interval(days => $2)`,
				p.project, p.eventDays)
		}
	}
}
