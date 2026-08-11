package postgres

import (
	"context"
	"encoding/json"

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
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
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

		out = append(out, domain.AdminRiskRule{ID: id, Name: d.Name, Condition: d.Condition, Action: d.Action, Enabled: enabled})
	}

	return out, rows.Err()
}

func (a *pgRisk) CreateRule(ctx context.Context, projectID string, r domain.AdminRiskRule) (domain.AdminRiskRule, error) {
	id := newUUID()

	raw, err := json.Marshal(riskRuleData{Name: r.Name, Condition: r.Condition, Action: r.Action})
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
	raw, err := json.Marshal(riskRuleData{Name: r.Name, Condition: r.Condition, Action: r.Action})
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

// ListEvents returns recorded risk events. There is no dedicated risk-events
// store yet (the evaluation engine that would populate it is future work), so
// this returns an empty page rather than 501.
func (a *pgRisk) ListEvents(_ context.Context, _ string) ([]map[string]any, error) {
	return []map[string]any{}, nil
}
