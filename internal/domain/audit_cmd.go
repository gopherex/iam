package domain

import (
	"time"

	"github.com/go-faster/jx"
)

// AuditLogEntry is one recorded privileged action (an admin/operator/service
// principal mutating a tenant's data).
type AuditLogEntry struct {
	ID       string
	Type     string
	ActorID  string
	TargetID string
	At       time.Time
	Data     map[string]jx.Raw
}

// AuditLogListCmd filters and paginates the audit log. Empty filter fields are
// ignored. Cursor is an opaque keyset token from a prior page.
type AuditLogListCmd struct {
	ProjectID string
	ActorID   string
	TargetID  string
	Type      string
	Cursor    string
	Limit     int
}

// AuditExportCmd requests an audit-log export job over an optional time window.
type AuditExportCmd struct {
	ProjectID string
	From      time.Time
	To        time.Time
	Format    string
}
