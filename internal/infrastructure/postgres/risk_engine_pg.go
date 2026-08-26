package postgres

// The risk evaluator.
//
// A risk rule pairs a named signal with what to do when it fires. Signals are a
// closed set (domain.RiskSignals), not an expression language: a rule an
// administrator writes either evaluates or is refused when it is written, which
// is the difference between a security control and a note about one.
//
// Evaluation runs the moment the first factor verifies and before a session is
// minted — the only point where "step up" is still a decision rather than the
// retraction of a session already handed out, and the only point where "no
// earlier session came from this address" is still true of this sign-in.

import (
	"context"
	"fmt"

	"github.com/gopherex/iam/internal/domain"
)

// Risk actions, in the order they win. A block ends the sign-in; a step-up
// changes it; notify and allow only record.
const (
	riskActionBlock    = "block"
	riskActionStepUp   = "require_step_up"
	riskActionNotify   = "notify"
	riskActionAllow    = "allow"
	riskEventRuleFired = "risk.rule.fired"
)

// riskDecision is what evaluation concluded.
type riskDecision struct {
	Action string
	Rule   string
	Signal string
}

// evaluateSignInRisk runs a project's enabled rules against one sign-in.
//
// The strongest action wins, and `allow` is an explicit override: a rule saying
// a situation is fine beats one saying it is suspicious, so an administrator can
// carve an exception without deleting the rule it is an exception to.
func evaluateSignInRisk(
	ctx context.Context, db *DB, emitter Emitter,
	projectID, environment, accountID string, priorFailures int,
) (riskDecision, error) {
	rules, err := NewPgRisk(db, emitter).ListRules(ctx, projectID)
	if err != nil {
		// Risk evaluation must not be able to take sign-in down with it: a rule
		// store that cannot be read leaves the sign-in to the controls that do not
		// depend on it (lockout, blocks, rate limits).
		//nolint:nilerr // deliberately non-fatal; see above.
		return riskDecision{}, nil
	}

	if len(rules) == 0 {
		return riskDecision{}, nil
	}

	signals, err := signInSignals(ctx, db, projectID, accountID, priorFailures)
	if err != nil {
		return riskDecision{}, err
	}

	decision := riskDecision{}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		signal := rule.EffectiveSignal()
		if !signals[signal] {
			continue
		}

		if rule.Action == riskActionAllow {
			return riskDecision{Action: riskActionAllow, Rule: rule.Name, Signal: signal}, nil
		}

		if riskActionRank(rule.Action) > riskActionRank(decision.Action) {
			decision = riskDecision{Action: rule.Action, Rule: rule.Name, Signal: signal}
		}
	}

	if decision.Action == "" {
		return decision, nil
	}

	if err := recordRiskEvent(ctx, db, projectID, environment, accountID, decision); err != nil {
		return decision, err
	}

	return decision, nil
}

// recordRiskEvent writes the record `admin/risk/events` reads back.
//
// It is written on the pool rather than through the ambient transaction on
// purpose: a `block` decision aborts the sign-in, which rolls that transaction
// back — and an event that disappears exactly when a rule refused somebody is
// the one you most needed. An auto-commit insert survives the rollback.
func recordRiskEvent(
	ctx context.Context, db *DB, projectID, environment, accountID string, decision riskDecision,
) error {
	payload, err := marshal(map[string]any{
		payloadAccountID: accountID,
		"rule":           decision.Rule,
		"signal":         decision.Signal,
		"action":         decision.Action,
	})
	if err != nil {
		return err
	}

	if _, err = db.Pool.Exec(ctx,
		`INSERT INTO iam_events (id, project_id, environment, aggregate_id, user_id, type, published, data)
		 VALUES ($1, $2, $3, $4, $4, $5, true, $6)`,
		newUUID(), projectID, environment, accountID, riskEventRuleFired, payload); err != nil {
		return fmt.Errorf("record risk event: %w", err)
	}

	return nil
}

// riskActionRanks orders actions by how much they restrict the sign-in, so the
// strongest one a project's rules ask for is the one that takes effect.
//
//nolint:gochecknoglobals,mnd // a fixed ordering, not state or measurements.
var riskActionRanks = map[string]int{
	riskActionBlock:  3,
	riskActionStepUp: 2,
	riskActionNotify: 1,
}

// riskActionRank orders actions by how much they restrict the sign-in.
func riskActionRank(action string) int {
	return riskActionRanks[action]
}

// signInSignals evaluates every signal once, so a project with several rules
// pays for the lookups once rather than per rule.
func signInSignals(
	ctx context.Context, db *DB, projectID, accountID string, priorFailures int,
) (map[string]bool, error) {
	meta := domain.RequestMetaFromContext(ctx)
	out := map[string]bool{"recent_failures": priorFailures > 0}

	// A client that sends no fingerprint would otherwise look like a new device
	// on every request; absence of evidence is not the signal.
	if meta.Fingerprint != "" {
		seen, err := sessionSeen(ctx, db, projectID, accountID, "fingerprint", meta.Fingerprint)
		if err != nil {
			return nil, err
		}

		out["new_device"] = !seen
	}

	if meta.IP != "" {
		seen, err := sessionSeen(ctx, db, projectID, accountID, "ip", meta.IP)
		if err != nil {
			return nil, err
		}

		out["new_ip"] = !seen
	}

	return out, nil
}

// sessionSeen reports whether any session this user already holds carries the
// given value in its device envelope.
func sessionSeen(
	ctx context.Context, db *DB, projectID, accountID, field, value string,
) (bool, error) {
	var n int

	err := db.TxDB.QueryRow(ctx,
		`SELECT count(*) FROM iam_sessions
		  WHERE project_id = $1 AND user_id = $2 AND data ->> $3 = $4
		  LIMIT 1`,
		projectID, accountID, field, value).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("risk: count sessions: %w", err)
	}

	return n > 0, nil
}
