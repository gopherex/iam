package postgres

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound / ErrConflict are the storage-level sentinels the service layer
// translates to its API errors. (IAM has no proto layer; records are stored as
// plain JSON in the `data` column.)
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

func ptr[T any](v T) *T { return &v }

// marshal serializes a record to a JSONB blob.
func marshal(v any) ([]byte, error) { return json.Marshal(v) }

// unmarshal decodes a JSONB blob into v.
func unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

// translatePgErr maps pgx.ErrNoRows onto ErrNotFound; everything else passes
// through wrapped with the resource name.
func translatePgErr(resource string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", resource, ErrNotFound)
	}
	return err
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505), translated into ErrConflict by callers.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// newUUID mints a fresh record id.
func newUUID() string { return uuid.NewString() }
