package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aarondl/opt/null"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// pgTestMode backs the /v1/test/* endpoints used by the SDK/test harness to
// drive deterministic tests. Every operation is refused unless the environment
// is non-live (see the handler gate) — test mode must never touch live data.
type pgTestMode struct {
	db      *DB
	emitter Emitter
}

// NewPgTestMode builds the test-mode adapter.
func NewPgTestMode(db *DB, emitter Emitter) *pgTestMode {
	return &pgTestMode{db: db, emitter: emitter}
}

// testModeResetTables are the runtime tables cleared by Reset, in FK-free order
// (the schema has no foreign keys). Each is wiped for the (project, env).
var testModeResetTables = []string{
	"iam_refresh_tokens", "iam_sessions", "iam_credentials", "iam_recovery_codes",
	"iam_factors", "iam_identities", "iam_challenges", "iam_flows", "iam_auth_codes",
	"iam_device_codes", "iam_activity", "iam_users",
}

// tables that carry an environment column (wiped by project+env); the rest are
// wiped by project only.
var testModeEnvScoped = map[string]bool{
	"iam_refresh_tokens": true, "iam_sessions": true, "iam_credentials": true,
	"iam_recovery_codes": true, "iam_factors": true, "iam_identities": true,
	"iam_challenges": true, "iam_flows": true, "iam_auth_codes": true,
	"iam_device_codes": true, "iam_users": true,
}

// Reset deletes the project's runtime data for the environment. Destructive; the
// handler restricts it to non-live environments.
func (a *pgTestMode) Reset(ctx context.Context, projectID, env string) (int64, error) {
	var total int64

	for _, table := range testModeResetTables {
		query := `DELETE FROM ` + table + ` WHERE project_id = $1`
		args := []any{projectID}

		if testModeEnvScoped[table] {
			query += ` AND environment = $2`

			args = append(args, env)
		}

		tag, err := a.db.Pool.Exec(ctx, query, args...)
		if err != nil {
			return total, err
		}

		total += tag.RowsAffected()
	}

	return total, nil
}

// Seed creates fixture data. Currently supports a single test user from
// {"email": "...", "name": "..."} so tests have a known account.
func (a *pgTestMode) Seed(ctx context.Context, projectID, env string, spec map[string]any) error {
	email, _ := spec["email"].(string)
	if email == "" {
		return nil
	}

	name, _ := spec["name"].(string)

	return a.db.withTx(ctx, func(ctx context.Context) error {
		acc := domain.Account{
			ID: newUUID(), ProjectID: projectID, Kind: coreAuthKindHuman, Status: coreAuthStatusActive,
			PrimaryEmail: email, Name: name, EmailVerified: true, CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
		}

		raw, err := marshal(acc)
		if err != nil {
			return err
		}

		rawData := json.RawMessage(raw)
		now := nowUTC()
		emailPtr := ptr(null.From(email))

		_, err = models.IamUsers.Insert(&models.IamUserSetter{
			ID: &acc.ID, ProjectID: &acc.ProjectID, Environment: &env, Kind: ptr(acc.Kind),
			Status: ptr(acc.Status), PrimaryEmail: emailPtr, CreatedAt: &now, UpdatedAt: &now, Data: &rawData,
		}).One(ctx, a.db.Bobx())
		if isUniqueViolation(err) {
			return nil // idempotent seed
		}

		return err
	})
}

// Clock stores a test-clock offset for the (project, env) so time-dependent
// assertions are deterministic. The offset is persisted as a config doc; wiring
// it into every nowUTC read is deferred, so this records intent and is a safe
// no-op on the data path today.
func (a *pgTestMode) Clock(ctx context.Context, projectID, env string, advanceSeconds int, reset bool) error {
	doc := map[string]any{"advance_seconds": advanceSeconds}
	if reset {
		doc = map[string]any{"advance_seconds": 0}
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	_, err = a.db.Pool.Exec(ctx,
		`INSERT INTO iam_config (project_id, environment, key, data) VALUES ($1, $2, 'test_clock', $3)
		 ON CONFLICT (project_id, environment, key) DO UPDATE SET data = $3, updated_at = now()`,
		projectID, env, raw)

	return err
}

// Messages returns captured out-of-band deliveries (OTP codes, magic links) for
// the environment so tests can complete flows without a real mailbox. It reads
// delivery events recorded in iam_events; channel/to filter the result.
func (a *pgTestMode) Messages(ctx context.Context, projectID, env, channel, to string) ([]map[string]any, error) {
	rows, err := a.db.Pool.Query(ctx,
		`SELECT type, data, created_at FROM iam_events
		 WHERE project_id = $1 AND environment = $2 AND type LIKE 'auth.%'
		 ORDER BY created_at DESC LIMIT 200`, projectID, env)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)

	for rows.Next() {
		var (
			typ string
			raw []byte
			ts  any
		)

		if err := rows.Scan(&typ, &raw, &ts); err != nil {
			return nil, err
		}

		var payload map[string]any

		_ = json.Unmarshal(raw, &payload)

		msg := map[string]any{"type": typ, "payload": payload}
		if !messageMatches(payload, channel, to) {
			continue
		}

		out = append(out, msg)
	}

	return out, rows.Err()
}

func messageMatches(payload map[string]any, channel, to string) bool {
	if channel != "" {
		if c, _ := payload["channel"].(string); c != "" && c != channel {
			return false
		}
	}

	if to != "" {
		if t, _ := payload["to"].(string); t != "" && t != to {
			return false
		}
	}

	return true
}

// ClockOffset reads an environment's test-clock offset.
//
// It is deliberately refused for live: the offset exists so a test can move
// time, and live time is not a thing a caller gets to move. Reading it costs one
// indexed lookup on a request that already named a non-live environment.
func (a *pgTestMode) ClockOffset(
	ctx context.Context, projectID, environment string,
) (time.Duration, error) {
	if environment == "" || environment == testModeLiveEnvironment {
		return 0, nil
	}

	var raw []byte

	err := a.db.Pool.QueryRow(ctx,
		`SELECT data FROM iam_config WHERE project_id = $1 AND environment = $2 AND key = 'test_clock'`,
		projectID, environment).Scan(&raw)
	if err != nil {
		// No document is the common case: the environment has no offset.
		return 0, nil //nolint:nilerr // an unset clock is not an error.
	}

	var doc struct {
		AdvanceSeconds int `json:"advance_seconds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return 0, nil //nolint:nilerr // an unreadable clock is an unset clock.
	}

	if doc.AdvanceSeconds <= 0 {
		return 0, nil
	}

	return time.Duration(doc.AdvanceSeconds) * time.Second, nil
}

// testModeLiveEnvironment is the environment test mode may never touch.
const testModeLiveEnvironment = "live"
