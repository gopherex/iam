package postgres

// Subject data export.
//
// Two endpoints ask for the same thing: an administrator answering a
// subject-access request (admin/users/{id}/export) and the person themselves
// (/v1/users/me/export). Both enqueue a job; this is what the worker does with
// it.
//
// The export is everything IAM holds about one person that is theirs to see:
// their profile, the identities linked to it, the sessions it holds, the
// consents it recorded, the OAuth grants it gave, and its security activity. It
// deliberately excludes credential material — a password hash, a TOTP seed or a
// refresh token are about the account, not facts about the person, and handing
// them out would turn an access request into a credential leak.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// errExportNoSubject is a job asking to export nobody: the spec lost its subject
// between enqueue and drain, which is a bug rather than a bad request.
var errExportNoSubject = errors.New("data export: no subject in the job spec")

// Export result keys and the content type of the rendered document.
const (
	exportKeyCount   = "count"
	exportKeyType    = "content_type"
	exportKeyURL     = "download_url"
	exportContentype = "application/json"
)

// exportRowLimit bounds each collection in an export. A subject-access request
// wants the record, not an unbounded dump that cannot be delivered.
const exportRowLimit = 1000

// processDataExport renders one person's data. It backs both the admin and the
// self-service export job types.
func (db *DB) processDataExport(ctx context.Context, projectID string, spec map[string]any) (map[string]any, error) {
	accountID := exportSubject(spec)
	if accountID == "" {
		return nil, errExportNoSubject
	}

	out := map[string]any{payloadAccountID: accountID, "exported_at": nowUTC()}

	profile, err := exportRow(ctx, db,
		`SELECT data FROM iam_users WHERE project_id = $1 AND id = $2`, projectID, accountID)
	if err != nil {
		return nil, err
	}

	out["profile"] = profile

	collections := []struct {
		name  string
		query string
	}{
		{"identities", `SELECT data FROM iam_identities WHERE project_id = $1 AND user_id = $2
		                 ORDER BY created_at DESC LIMIT $3`},
		{"sessions", `SELECT data FROM iam_sessions WHERE project_id = $1 AND user_id = $2
		               ORDER BY created_at DESC LIMIT $3`},
		{"grants", `SELECT data FROM iam_oauth_grants WHERE project_id = $1 AND user_id = $2
		             ORDER BY granted_at DESC LIMIT $3`},
		{"activity", `SELECT data FROM iam_activity WHERE project_id = $1 AND user_id = $2
		               ORDER BY at DESC LIMIT $3`},
	}

	for _, c := range collections {
		rows, err := exportRows(ctx, db, c.query, projectID, accountID)
		if err != nil {
			return nil, err
		}

		out[c.name] = rows
	}

	blob, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("data export: encode: %w", err)
	}

	return map[string]any{
		exportKeyCount: 1,
		exportKeyType:  exportContentype,
		exportKeyURL:   "data:" + exportContentype + ";base64," + base64Std(blob),
	}, nil
}

// exportSubject reads the subject id out of a job spec. The two creators name it
// differently, and both spellings are already in flight.
func exportSubject(spec map[string]any) string {
	for _, key := range []string{eventFieldUserID, payloadAccountID, "subject_id"} {
		if v, ok := spec[key].(string); ok && v != "" {
			return v
		}
	}

	return ""
}

// exportRow reads a single jsonb envelope, or nil when there is none.
func exportRow(ctx context.Context, db *DB, query string, args ...any) (map[string]any, error) {
	var raw []byte
	if err := db.Pool.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return nil, fmt.Errorf("data export: read subject: %w", err)
	}

	var out map[string]any

	_ = json.Unmarshal(raw, &out)

	return out, nil
}

// exportRows reads a bounded collection of jsonb envelopes.
func exportRows(ctx context.Context, db *DB, query, projectID, accountID string) ([]map[string]any, error) {
	rows, err := db.Pool.Query(ctx, query, projectID, accountID, exportRowLimit)
	if err != nil {
		return nil, fmt.Errorf("data export: query: %w", err)
	}
	defer rows.Close()

	out := make([]map[string]any, 0)

	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("data export: scan: %w", err)
		}

		var item map[string]any

		_ = json.Unmarshal(raw, &item)

		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("data export: read: %w", err)
	}

	return out, nil
}
