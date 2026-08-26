package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

// pgRisk is the admin risk-rules + rate-limit-blocks adapter. Risk rules are
// declarative and stored in iam_risk_rules (the evaluation engine is a separate
// concern); blocks are stored in iam_blocks and enforced at the auth entry
// points via IsBlocked.
type pgRisk struct {
	db      *DB
	emitter Emitter
}

// NewPgRisk builds the risk adapter.
func NewPgRisk(db *DB, emitter Emitter) *pgRisk {
	return &pgRisk{db: db, emitter: emitter}
}

type riskRuleData struct {
	Name string `json:"name"`
	// Signal is the current field; Condition is the spelling released in 1.4 and
	// is still read so a rule written then keeps evaluating.
	Signal    string `json:"signal,omitempty"`
	Condition string `json:"condition,omitempty"`
	Action    string `json:"action"`
}

// riskRuleDataOf renders a rule for storage, writing the signal under the
// current name and keeping the released one in step.
func riskRuleDataOf(r domain.AdminRiskRule) riskRuleData {
	signal := r.EffectiveSignal()

	return riskRuleData{Name: r.Name, Signal: signal, Condition: signal, Action: r.Action}
}

func (a *pgRisk) ListRules(ctx context.Context, projectID string) ([]domain.AdminRiskRule, error) {
	rows, err := a.db.Pool.Query(ctx,
		`SELECT id, enabled, data FROM iam_risk_rules WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AdminRiskRule

	for rows.Next() {
		var (
			id      string
			enabled bool
			raw     []byte
		)

		if err := rows.Scan(&id, &enabled, &raw); err != nil {
			return nil, err
		}

		var d riskRuleData

		_ = json.Unmarshal(raw, &d)

		out = append(out, domain.AdminRiskRule{
			ID: id, Name: d.Name, Signal: d.Signal, Condition: d.Condition,
			Action: d.Action, Enabled: enabled,
		})
	}

	return out, rows.Err()
}

func (a *pgRisk) CreateRule(ctx context.Context, projectID string, r domain.AdminRiskRule) (domain.AdminRiskRule, error) {
	if err := r.Validate(); err != nil {
		return domain.AdminRiskRule{}, err
	}

	id := newUUID()

	raw, err := json.Marshal(riskRuleDataOf(r))
	if err != nil {
		return domain.AdminRiskRule{}, err
	}

	if _, err := a.db.Pool.Exec(ctx,
		`INSERT INTO iam_risk_rules (id, project_id, enabled, data) VALUES ($1, $2, $3, $4)`,
		id, projectID, r.Enabled, raw); err != nil {
		return domain.AdminRiskRule{}, err
	}

	r.ID = id

	return r, nil
}

func (a *pgRisk) UpdateRule(ctx context.Context, projectID, id string, r domain.AdminRiskRule) (domain.AdminRiskRule, error) {
	if err := r.Validate(); err != nil {
		return domain.AdminRiskRule{}, err
	}

	raw, err := json.Marshal(riskRuleDataOf(r))
	if err != nil {
		return domain.AdminRiskRule{}, err
	}

	n, err := a.db.Pool.Exec(ctx,
		`UPDATE iam_risk_rules SET enabled = $1, data = $2, updated_at = now() WHERE id = $3 AND project_id = $4`,
		r.Enabled, raw, id, projectID)
	if err != nil {
		return domain.AdminRiskRule{}, err
	}

	if n.RowsAffected() == 0 {
		return domain.AdminRiskRule{}, domain.ErrNotFound
	}

	r.ID = id

	return r, nil
}

func (a *pgRisk) DeleteRule(ctx context.Context, projectID, id string) error {
	n, err := a.db.Pool.Exec(ctx, `DELETE FROM iam_risk_rules WHERE id = $1 AND project_id = $2`, id, projectID)
	if err != nil {
		return err
	}

	if n.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

type blockData struct {
	Type   string `json:"type"`
	Reason string `json:"reason,omitempty"`
}

func (a *pgRisk) CreateBlock(ctx context.Context, projectID, env string, b domain.AdminBlock) (domain.AdminBlock, error) {
	id := newUUID()

	raw, err := json.Marshal(blockData{Type: b.Type, Reason: b.Reason})
	if err != nil {
		return domain.AdminBlock{}, err
	}

	var expires any
	if !b.ExpiresAt.IsZero() {
		expires = b.ExpiresAt
	}

	if _, err := a.db.Pool.Exec(ctx,
		`INSERT INTO iam_blocks (id, project_id, environment, subject, expires_at, data) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, projectID, adminEnv(env), b.Value, expires, raw); err != nil {
		return domain.AdminBlock{}, err
	}

	b.ID = id

	return b, nil
}

func (a *pgRisk) DeleteBlock(ctx context.Context, projectID, id string) error {
	n, err := a.db.Pool.Exec(ctx, `DELETE FROM iam_blocks WHERE id = $1 AND project_id = $2`, id, projectID)
	if err != nil {
		return err
	}

	if n.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// IsBlocked reports whether any of the subjects (an IP, email, phone, …) matches
// an unexpired block for the project. Used at the auth entry points.
func (a *pgRisk) IsBlocked(ctx context.Context, projectID string, subjects ...string) (bool, error) {
	nonEmpty := make([]string, 0, len(subjects))
	for _, s := range subjects {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}

	if len(nonEmpty) == 0 {
		return false, nil
	}

	var n int

	err := a.db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM iam_blocks
		 WHERE project_id = $1 AND subject = ANY($2) AND (expires_at IS NULL OR expires_at > now())`,
		projectID, nonEmpty).Scan(&n)
	if err != nil {
		return false, err
	}

	return n > 0, nil
}

// ListEvents returns the risk events recorded for a project: one row per rule
// that fired, newest first. They are ordinary emitted events, so a webhook
// consumer and this endpoint see the same thing.
func (a *pgRisk) ListEvents(ctx context.Context, projectID string) ([]map[string]any, error) {
	rows, err := a.db.Pool.Query(ctx,
		`SELECT id, data, created_at FROM iam_events
		  WHERE project_id = $1 AND type = $2
		  ORDER BY created_at DESC LIMIT $3`,
		projectID, riskEventRuleFired, riskEventPageSize)
	if err != nil {
		return nil, fmt.Errorf("risk: query events: %w", err)
	}
	defer rows.Close()

	out := make([]map[string]any, 0)

	for rows.Next() {
		var (
			id        string
			raw       []byte
			createdAt time.Time
		)

		if err := rows.Scan(&id, &raw, &createdAt); err != nil {
			return nil, fmt.Errorf("risk: scan event: %w", err)
		}

		var payload map[string]any

		_ = json.Unmarshal(raw, &payload)

		out = append(out, map[string]any{"id": id, "at": createdAt, "payload": payload})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("risk: read events: %w", err)
	}

	return out, nil
}

// riskEventPageSize bounds the events endpoint. It is a review surface, not an
// export; audit/export is where a full history comes from.
const riskEventPageSize = 200
