package postgres

// Retention is measured from closure/update. Keep active work and records that
// explain it; deleting an open case must never strand a recovery guard.
const (
	securityRetentionUpdated = "COALESCE((records.data->>'updated_at')::timestamptz,records.created_at)"
	securityRetentionIdle    = `
AND NOT EXISTS(SELECT 1 FROM iam_security_guards g WHERE g.project_id=records.project_id
 AND g.environment=records.environment
 AND g.user_id=records.user_id)
AND NOT EXISTS(SELECT 1 FROM iam_flows f WHERE f.project_id=records.project_id
 AND f.environment=records.environment
 AND f.user_id=records.user_id
 AND f.status='pending'
 AND f.expires_at>now())`
)

type securityRetentionSweep struct{ table, predicate, age string }

func securityRetentionSweeps() []securityRetentionSweep {
	return []securityRetentionSweep{
		{
			"iam_security_incidents", "records.status='resolved'" + securityRetentionIdle,
			securityRetentionUpdated,
		},
		{
			"iam_security_cases", "records.status IN ('completed','rejected')" + securityRetentionIdle,
			securityRetentionUpdated,
		},
		{
			"iam_security_deliveries", `NOT EXISTS(SELECT 1 FROM iam_security_cases c
 WHERE c.id=records.data->>'case_id'
 AND c.project_id=records.project_id
 AND c.environment=records.environment)
AND NOT EXISTS(SELECT 1 FROM iam_security_incidents i
 WHERE i.id=records.data->>'incident_id'
 AND i.project_id=records.project_id
 AND i.environment=records.environment)`,
			securityRetentionUpdated,
		},
		{
			"iam_security_case_decisions", `NOT EXISTS(SELECT 1 FROM iam_security_cases c
 WHERE c.id=records.case_id
 AND c.project_id=records.project_id
 AND c.environment=records.environment)`,
			"records.created_at",
		},
	}
}
