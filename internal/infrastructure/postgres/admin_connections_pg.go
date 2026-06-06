package postgres

// pgAdminConnections is the Postgres-backed adapter for api.AdminConnections —
// the admin-facing CRUD surface over the same iam_sso_connections / iam_domains
// tables that the federation adapter owns.
//
// Persistence follows the package gold pattern: the domain aggregate is
// marshalled into the `data jsonb` envelope; typed columns (project_id, type,
// status, name, domain, connection_id) are lookup-only. Every mutation runs
// inside withTx / withTxRet. The tenant boundary is project_id on every query.
//
// Shared helpers from federation_pg.go are reused directly:
//   - fedConnSetter       — builds an IamSsoConnectionSetter from a Connection.
//   - fedConnectionFromRow — unmarshals an IamSsoConnection row.
//   - fedDomainFromRow    — unmarshals an IamDomain row.
//   - fedRandomToken      — mints a crypto-random opaque token (hex).
//   - fedHashToken        — sha256 hex digest of a token.
//
// Domain-verification tokens are persisted in the connection's `data` envelope
// (field VerifyToken on domain.Domain) and returned as the DNS TXT record value
// in AdminDomainRegistration.

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aarondl/opt/null"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// ===========================================================================
// pgAdminConnections
// ===========================================================================

// pgAdminConnections provides admin-facing CRUD over iam_sso_connections and
// iam_domains. It is scoped to a project on every operation (tenant boundary).
type pgAdminConnections struct {
	db      *DB
	emitter Emitter
}

// NewPgAdminConnections builds the Postgres-backed AdminConnections adapter.
func NewPgAdminConnections(db *DB, emitter Emitter) *pgAdminConnections {
	return &pgAdminConnections{db: db, emitter: emitter}
}

var _ api.AdminConnections = (*pgAdminConnections)(nil)

// ---------------------------------------------------------------------------
// Connection CRUD
// ---------------------------------------------------------------------------

func (a *pgAdminConnections) List(ctx context.Context, projectID string) ([]domain.Connection, error) {
	rows, err := models.IamSsoConnections.Query(
		sm.Where(models.IamSsoConnections.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Connection, 0, len(rows))
	for _, row := range rows {
		c, err := fedConnectionFromRow(a.db.Cipher, row)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

func (a *pgAdminConnections) Get(ctx context.Context, projectID, connID string) (*domain.Connection, error) {
	row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), connID)
	if err != nil {
		if errors.Is(translatePgErr("connection", err), ErrNotFound) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, domain.ErrConnectionNotFound
	}
	return fedConnectionFromRow(a.db.Cipher, row)
}

func (a *pgAdminConnections) Create(ctx context.Context, cmd domain.AdminConnectionCmd) (*domain.Connection, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Connection, error) {
		conn := &domain.Connection{
			ID:          newUUID(),
			ProjectID:   cmd.ProjectID,
			Type:        cmd.Type,
			Name:        cmd.Name,
			Status:      "active",
			Domains:     cmd.Domains,
			ExternalRef: cmd.ExternalRef,
		}
		setter, err := fedConnSetter(a.db.Cipher, conn)
		if err != nil {
			return nil, err
		}
		if _, err := models.IamSsoConnections.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "connection.created",
			ProjectID:   conn.ProjectID,
			Environment: "",
			AggregateID: conn.ID,
			Payload:     conn,
		}); err != nil {
			return nil, err
		}
		return conn, nil
	})
}

// Update applies a JSON merge patch to the connection aggregate.
//
// Patch strategy (read-modify-write inside the serializable tx):
//  1. Load the current row (tenant boundary check).
//  2. Unmarshal into domain.Connection via fedConnectionFromRow.
//  3. Apply recognised top-level keys from the patch map onto the struct.
//     Only keys that are present in the patch are touched; absent keys keep
//     their current value.  The same logic as fedApplyConnectionPatch is used
//     so that admin and federation patches behave identically.
//  4. Re-marshal the modified aggregate and write it back with fedConnSetter.
func (a *pgAdminConnections) Update(ctx context.Context, projectID, connID string, patch map[string]any) (*domain.Connection, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Connection, error) {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), connID)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return nil, domain.ErrConnectionNotFound
			}
			return nil, err
		}
		if row.ProjectID != projectID { // tenant boundary
			return nil, domain.ErrConnectionNotFound
		}
		conn, err := fedConnectionFromRow(a.db.Cipher, row)
		if err != nil {
			return nil, err
		}
		// Apply the merge patch onto the aggregate (shared helper, same semantics
		// as the federation update path).
		fedApplyConnectionPatch(conn, patch)
		setter, err := fedConnSetter(a.db.Cipher, conn)
		if err != nil {
			return nil, err
		}
		setter.ID = nil        // never re-set the pk on update
		setter.ProjectID = nil // never change the tenant column
		setter.UpdatedAt = ptr(nowUTC())
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "connection.updated",
			ProjectID:   conn.ProjectID,
			Environment: "",
			AggregateID: conn.ID,
			Payload:     conn,
		}); err != nil {
			return nil, err
		}
		return conn, nil
	})
}

func (a *pgAdminConnections) Delete(ctx context.Context, projectID, connID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), connID)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return domain.ErrConnectionNotFound
			}
			return err
		}
		if row.ProjectID != projectID { // tenant boundary
			return domain.ErrConnectionNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "connection.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: connID,
			Payload:     map[string]any{"id": connID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Domain CRUD + verification
// ---------------------------------------------------------------------------

func (a *pgAdminConnections) ListDomains(ctx context.Context, projectID string) ([]domain.Domain, error) {
	rows, err := models.IamDomains.Query(
		sm.Where(models.IamDomains.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Domain, 0, len(rows))
	for _, row := range rows {
		d, err := fedDomainFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, nil
}

// CreateDomain registers a new email domain for a project, generating a DNS
// verification token. The token is stored in the data envelope (field
// "verify_token") so VerifyDomain can check it later; it is returned to the
// caller as the TXT record value they must publish under
// _iam-verify.<domain>.
func (a *pgAdminConnections) CreateDomain(ctx context.Context, cmd domain.AdminDomainCmd) (*domain.AdminDomainRegistration, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminDomainRegistration, error) {
		// Mint a random verification token (plaintext, URL-safe hex); we store it
		// in the envelope rather than a separate column, so VerifyDomain can
		// compare it out-of-band when the caller asks us to confirm the record.
		verifyToken, err := fedRandomToken()
		if err != nil {
			return nil, err
		}

		dom := &domain.Domain{
			ID:           newUUID(),
			ProjectID:    cmd.ProjectID,
			Domain:       cmd.Domain,
			Status:       "pending",
			ConnectionID: cmd.ConnectionID,
		}

		// The domain aggregate is stored in the envelope; the verify token is an
		// extra field that lives only in the envelope (not on domain.Domain to
		// keep the struct lean).  We marshal the struct first and then merge in
		// the token.
		rawDom, err := marshal(dom)
		if err != nil {
			return nil, err
		}
		var envelope map[string]any
		if err := json.Unmarshal(rawDom, &envelope); err != nil {
			return nil, err
		}
		envelope["verify_token"] = verifyToken
		rawEnv, err := json.Marshal(envelope)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(rawEnv)

		setter := &models.IamDomainSetter{
			ID:        &dom.ID,
			ProjectID: &dom.ProjectID,
			Domain:    ptr(dom.Domain),
			Status:    ptr(dom.Status),
			Data:      &rm,
		}
		if cmd.ConnectionID != "" {
			v := null.From(cmd.ConnectionID)
			setter.ConnectionID = &v
		}

		if _, err := models.IamDomains.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrDomainTaken
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "domain.created",
			ProjectID:   dom.ProjectID,
			Environment: "",
			AggregateID: dom.ID,
			Payload:     dom,
		}); err != nil {
			return nil, err
		}
		return &domain.AdminDomainRegistration{
			Domain:                  dom,
			VerificationRecordType:  "TXT",
			VerificationRecordName:  "_iam-verify." + cmd.Domain,
			VerificationRecordValue: verifyToken,
		}, nil
	})
}

func (a *pgAdminConnections) DeleteDomain(ctx context.Context, projectID, domainID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamDomain(ctx, a.db.Bobx(), domainID)
		if err != nil {
			if errors.Is(translatePgErr("domain", err), ErrNotFound) {
				return domain.ErrDomainNotFound
			}
			return err
		}
		if row.ProjectID != projectID { // tenant boundary
			return domain.ErrDomainNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "domain.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: domainID,
			Payload:     map[string]any{"id": domainID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// VerifyDomain marks a domain as verified.  The actual DNS lookup is performed
// out of band by the caller; here we simply flip the persisted state to
// "verified" and record the timestamp — mirroring the same pattern used by
// pgFederationConnections.VerifyDomain.
func (a *pgAdminConnections) VerifyDomain(ctx context.Context, projectID, domainID string) (*domain.Domain, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Domain, error) {
		row, err := models.FindIamDomain(ctx, a.db.Bobx(), domainID)
		if err != nil {
			if errors.Is(translatePgErr("domain", err), ErrNotFound) {
				return nil, domain.ErrDomainNotFound
			}
			return nil, err
		}
		if row.ProjectID != projectID { // tenant boundary
			return nil, domain.ErrDomainNotFound
		}
		dom, err := fedDomainFromRow(row)
		if err != nil {
			return nil, err
		}
		dom.Status = "verified"
		raw, err := marshal(dom)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		verifiedAt := null.From(nowUTC())
		setter := &models.IamDomainSetter{
			Status:     ptr("verified"),
			VerifiedAt: &verifiedAt,
			Data:       &rm,
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "domain.verified",
			ProjectID:   dom.ProjectID,
			Environment: "",
			AggregateID: dom.ID,
			Payload:     dom,
		}); err != nil {
			return nil, err
		}
		return dom, nil
	})
}
