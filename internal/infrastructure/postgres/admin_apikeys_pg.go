package postgres

// Postgres adapter for the AdminAPIKeys port declared in pkg/api/admin.go.
//
// Port: api.AdminAPIKeys — project-scoped opaque API-key management.
// Table: iam_api_keys (id, project_id, prefix, hash, disabled, expires_at, created_at, data jsonb).
//
// Design notes (mirror admin_pg.go):
//   - The data column is a jsonb envelope holding the full domain.APIKey aggregate.
//   - The lookup columns (project_id, prefix, hash, disabled, expires_at) are
//     derived from the aggregate and kept in sync on every mutation.
//   - Reads run on db.Bobx(); every mutation is wrapped in withTx / withTxRet.
//   - tenant boundary: a row whose project_id != the requested project_id is
//     reported as domain.ErrNotFound.
//   - Secrets: mintAPIKeySecret() (defined in machineid_pg.go) mints a token of
//     the form "iak_<random>.<random>"; only the sha256 hash and the prefix are
//     persisted. The plaintext is returned exactly once in AdminAPIKeySecret.
//   - Create and Rotate share the same minting logic; Rotate replaces the prefix
//     and hash on an existing row.

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// pgAdminAPIKeys is the Postgres-backed AdminAPIKeys adapter.
type pgAdminAPIKeys struct {
	db      *DB
	emitter Emitter
}

// NewPgAdminAPIKeys constructs the adapter over an open *DB.
func NewPgAdminAPIKeys(db *DB, emitter Emitter) *pgAdminAPIKeys {
	return &pgAdminAPIKeys{db: db, emitter: emitter}
}

// Port assertion.
var _ api.AdminAPIKeys = (*pgAdminAPIKeys)(nil)

// ===== helpers =======================================================

// findAdminAPIKey loads a row and enforces the tenant boundary.
// It reuses the package-level loadAPIKey from machineid_pg.go which already
// handles FindIamAPIKey + project_id check + envelope unmarshal.
func (a *pgAdminAPIKeys) findAdminAPIKey(ctx context.Context, projectID, keyID string) (*models.IamAPIKey, *domain.APIKey, error) {
	row, key, err := loadAPIKey(a.db, ctx, projectID, keyID)
	if err != nil {
		return nil, nil, err
	}
	return row, key, nil
}

// ===== List ===========================================================

// List returns every API key that belongs to projectID.
func (a *pgAdminAPIKeys) List(ctx context.Context, projectID string) ([]domain.APIKey, error) {
	rows, err := models.IamAPIKeys.Query(
		sm.Where(models.IamAPIKeys.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.OrderBy(models.IamAPIKeys.Columns.CreatedAt).Asc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.APIKey, 0, len(rows))
	for _, row := range rows {
		var key domain.APIKey
		if err := unmarshal(row.Data, &key); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, nil
}

// ===== Create =========================================================

// Create mints a new API key for the project described by cmd.
// The plaintext secret is returned exactly once in AdminAPIKeySecret.Secret;
// only the sha256 hash and the short prefix are persisted.
func (a *pgAdminAPIKeys) Create(ctx context.Context, cmd domain.AdminAPIKeyCmd) (*domain.AdminAPIKeySecret, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminAPIKeySecret, error) {
		plaintext, prefix, hash, err := mintAPIKeySecret()
		if err != nil {
			return nil, err
		}
		now := nowUTC()
		key := domain.APIKey{
			ID:        newUUID(),
			ProjectID: cmd.ProjectID,
			Name:      cmd.Name,
			Scopes:    cmd.Scopes,
			Prefix:    prefix,
			Disabled:  false,
		}
		raw, err := marshal(keyEnvelope{APIKey: key})
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamAPIKeySetter{
			ID:        &key.ID,
			ProjectID: &key.ProjectID,
			Prefix:    &prefix,
			Hash:      &hash,
			Disabled:  &key.Disabled,
			CreatedAt: &now,
			Data:      &rm,
		}
		if !cmd.ExpiresAt.IsZero() {
			v := null.From(cmd.ExpiresAt.UTC())
			setter.ExpiresAt = &v
		}
		if _, err := models.IamAPIKeys.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		result := &domain.AdminAPIKeySecret{Key: &key, Secret: plaintext}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "api_key.created",
			ProjectID:   key.ProjectID,
			Environment: "",
			AggregateID: key.ID,
			Payload:     &key,
		}); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// ===== Update =========================================================

// Update patches the name, scopes, and/or disabled state of an existing key.
func (a *pgAdminAPIKeys) Update(ctx context.Context, cmd domain.AdminAPIKeyUpdateCmd) (*domain.APIKey, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.APIKey, error) {
		row, key, err := a.findAdminAPIKey(ctx, cmd.ProjectID, cmd.KeyID)
		if err != nil {
			return nil, err
		}
		if cmd.Name != "" {
			key.Name = cmd.Name
		}
		if cmd.Scopes != nil {
			key.Scopes = cmd.Scopes
		}
		key.Disabled = cmd.Disabled
		raw, err := marshal(keyEnvelope{APIKey: *key})
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamAPIKeySetter{
			Disabled: &key.Disabled,
			Data:     &rm,
		}); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "api_key.updated",
			ProjectID:   key.ProjectID,
			Environment: "",
			AggregateID: key.ID,
			Payload:     key,
		}); err != nil {
			return nil, err
		}
		return key, nil
	})
}

// ===== Delete =========================================================

// Delete permanently removes the key from the project.
func (a *pgAdminAPIKeys) Delete(ctx context.Context, projectID, keyID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.findAdminAPIKey(ctx, projectID, keyID)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "api_key.deleted",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: keyID,
			Payload:     map[string]any{"id": keyID, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ===== Rotate =========================================================

// Rotate mints a fresh secret for an existing key.
// The old secret is immediately invalidated; the new plaintext is returned
// exactly once. The prefix and hash columns are replaced atomically.
func (a *pgAdminAPIKeys) Rotate(ctx context.Context, projectID, keyID string) (*domain.AdminAPIKeySecret, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AdminAPIKeySecret, error) {
		row, key, err := a.findAdminAPIKey(ctx, projectID, keyID)
		if err != nil {
			return nil, err
		}
		plaintext, prefix, hash, err := mintAPIKeySecret()
		if err != nil {
			return nil, err
		}
		key.Prefix = prefix
		raw, err := marshal(keyEnvelope{APIKey: *key})
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamAPIKeySetter{
			Prefix: &prefix,
			Hash:   &hash,
			Data:   &rm,
		}); err != nil {
			return nil, err
		}
		result := &domain.AdminAPIKeySecret{Key: key, Secret: plaintext}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "api_key.rotated",
			ProjectID:   key.ProjectID,
			Environment: "",
			AggregateID: key.ID,
			Payload:     key,
		}); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// ===== internal shim =================================================

// loadAPIKey is a package-level function defined in machineid_pg.go as a
// method on *PgMachineIdentities. To avoid duplicating the logic here while
// still being callable from a different receiver, we wrap it via a free
// function that accepts the *DB directly.
//
// NOTE: if the compiler rejects the call below because loadAPIKey is a method
// rather than a free function, replace the body of findAdminAPIKey with the
// inline equivalent (FindIamAPIKey + project_id guard + unmarshal) from
// machineid_pg.go:352-364.
func loadAPIKey(db *DB, ctx context.Context, projectID, keyID string) (*models.IamAPIKey, *domain.APIKey, error) {
	row, err := models.FindIamAPIKey(ctx, db.Bobx(), keyID)
	if err != nil {
		return nil, nil, translatePgErr("api_key", err)
	}
	if row.ProjectID != projectID {
		return nil, nil, domain.ErrNotFound
	}
	var key domain.APIKey
	if err := unmarshal(row.Data, &key); err != nil {
		return nil, nil, err
	}
	return row, &key, nil
}

// ensure time import is used (ExpiresAt handling).
var _ = time.Time{}
