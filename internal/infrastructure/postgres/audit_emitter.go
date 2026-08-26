package postgres

import (
	"context"
	"encoding/json"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

// auditingEmitter wraps an Emitter and records an audit-log row whenever a
// privileged actor (admin / operator / service / scim principal) emits an event
// — i.e. an administrative mutation, not ordinary runtime traffic. The audit
// write joins the caller's transaction so it is durable iff the mutation
// commits, and it is best-effort: an audit failure never fails the mutation.
//
// Only the event type, target and actor are recorded, plus the actor kind — NOT
// the event payload, which for some aggregates carries secrets.
type auditingEmitter struct {
	audit *pgAudit
	inner Emitter
}

// NewAuditingEmitter wraps inner so privileged mutations are audited.
func NewAuditingEmitter(db *DB, inner Emitter) *auditingEmitter {
	return &auditingEmitter{audit: NewPgAudit(db, inner), inner: inner}
}

var _ Emitter = (*auditingEmitter)(nil)

func (e *auditingEmitter) Emit(ctx context.Context, event domain.Event) error {
	if p, ok := api.PrincipalFrom(ctx); ok && p != nil && auditableActor(p.Kind) && event.ProjectID != "" {
		// Best-effort like record() below: the marshaled value is a fixed
		// literal (a PrincipalKind string), so this cannot actually fail.
		data, _ := json.Marshal(map[string]any{"actor_kind": string(p.Kind)}) //nolint:errchkjson // literal string value, cannot fail
		_ = e.audit.record(ctx, event.ProjectID, event.Type, auditActorID(p), event.AggregateID, data)
	}

	return e.inner.Emit(ctx, event)
}

func auditableActor(k domain.PrincipalKind) bool {
	switch k {
	case domain.PrincipalAdmin, domain.PrincipalOperator, domain.PrincipalService, domain.PrincipalSCIM:
		return true
	case domain.PrincipalUser, domain.PrincipalClient:
		return false
	default:
		return false
	}
}

func auditActorID(p *domain.Principal) string {
	if p.AccountID != "" {
		return p.AccountID
	}

	return string(p.Kind)
}
