package postgres

// Postgres adapter for the MachineIdentities port (iam_service_accounts +
// iam_api_keys). Follows the gold pattern in reference.go:
//   - each aggregate is stored as a `data jsonb` envelope; the lookup columns
//     (project_id, prefix, hash, disabled, expires_at) are populated from the
//     struct purely so queries can filter without decoding the blob;
//   - every mutation is wrapped in withTx / withTxRet (serializable + retry);
//     reads run directly on db.Bobx();
//   - the tenant boundary is project_id: a row whose project_id does not match
//     the requested one is reported as not-found.
//
// Crypto: service-account secrets and API-key secrets are opaque random tokens
// minted with crypto/rand; only their sha256 hash is persisted, never the
// plaintext. The plaintext is returned exactly once at creation. JWT access
// tokens are not signed here — MintToken returns an opaque bearer token and the
// signing step is marked with a TODO.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// PgMachineIdentities is the Postgres-backed MachineIdentities adapter.
type PgMachineIdentities struct{ db *DB }

// NewPgMachineIdentities builds the adapter over a *DB.
func NewPgMachineIdentities(db *DB) *PgMachineIdentities { return &PgMachineIdentities{db: db} }

// Port assertion: the adapter must satisfy api.MachineIdentities.
var _ api.MachineIdentities = (*PgMachineIdentities)(nil)

// ===== envelopes ===================================================

// saEnvelope is the persisted shape of a service account: the public domain
// aggregate plus its issued client secrets (which the domain struct does not
// carry). Only secret *hashes* live here, never plaintext.
type saEnvelope struct {
	domain.ServiceAccount
	Secrets []saSecret `json:"secrets,omitempty"`
}

// saSecret is one client secret of a service account. Hash is sha256(plaintext);
// Revoked marks a secret that can no longer mint tokens.
type saSecret struct {
	SecretID  string     `json:"secret_id"`
	ClientID  string     `json:"client_id"`
	Name      string     `json:"name"`
	Hash      string     `json:"hash"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Revoked   bool       `json:"revoked,omitempty"`
}

// ===== local helpers ===============================================

// machineIDRandomToken returns a URL-safe opaque token from crypto/rand.
func machineIDRandomToken(nbytes int) (string, error) {
	buf := make([]byte, nbytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// machineIDHash returns the hex sha256 of an opaque token; only this is stored.
func machineIDHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// loadServiceAccount fetches and tenant-checks a service-account envelope.
func (a *PgMachineIdentities) loadServiceAccount(ctx context.Context, projectID, id string) (*models.IamServiceAccount, *saEnvelope, error) {
	row, err := models.FindIamServiceAccount(ctx, a.db.Bobx(), id)
	if err != nil {
		return nil, nil, translatePgErr("service_account", err)
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, nil, domain.ErrNotFound
	}
	var env saEnvelope
	if err := unmarshal(row.Data, &env); err != nil {
		return nil, nil, err
	}
	return row, &env, nil
}

// saSetterData re-marshals an envelope into a setter Data field.
func saSetterData(env *saEnvelope) (*json.RawMessage, error) {
	raw, err := marshal(env)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	return &rm, nil
}

// ===== service accounts ============================================

// CreateServiceAccount creates a service account in the command's project.
func (a *PgMachineIdentities) CreateServiceAccount(ctx context.Context, cmd domain.ServiceAccountCmd) (*domain.ServiceAccount, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.ServiceAccount, error) {
		sa := domain.ServiceAccount{
			ID:        newUUID(),
			ProjectID: cmd.ProjectID,
			Name:      cmd.Name,
			Scopes:    cmd.Scopes,
			Disabled:  false,
		}
		env := saEnvelope{ServiceAccount: sa}
		data, err := saSetterData(&env)
		if err != nil {
			return nil, err
		}
		setter := &models.IamServiceAccountSetter{
			ID:        &sa.ID,
			ProjectID: &sa.ProjectID,
			Name:      &sa.Name,
			Disabled:  &sa.Disabled,
			Data:      data,
		}
		if _, err := models.IamServiceAccounts.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		// TODO outbox event: service_account.created
		return &sa, nil
	})
}

// ListServiceAccounts returns a cursor page of service accounts in a project.
// The cursor is the id of the last item; rows are ordered by id ascending and
// one extra row is fetched to compute HasMore.
func (a *PgMachineIdentities) ListServiceAccounts(ctx context.Context, cmd domain.MachineIDServiceAccountListCmd) (*domain.MachineIDServiceAccountPage, error) {
	limit := cmd.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	mods := []bob.Mod[*dialect.SelectQuery]{
		sm.Where(models.IamServiceAccounts.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
	}
	if cmd.Cursor != "" {
		mods = append(mods, sm.Where(models.IamServiceAccounts.Columns.ID.GT(psql.Arg(cmd.Cursor))))
	}
	mods = append(mods,
		sm.OrderBy(models.IamServiceAccounts.Columns.ID).Asc(),
		sm.Limit(limit+1),
	)
	rows, err := models.IamServiceAccounts.Query(mods...).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}

	page := &domain.MachineIDServiceAccountPage{Items: make([]*domain.ServiceAccount, 0, len(rows))}
	if len(rows) > limit {
		page.HasMore = true
		rows = rows[:limit]
	}
	for _, row := range rows {
		var env saEnvelope
		if err := unmarshal(row.Data, &env); err != nil {
			return nil, err
		}
		sa := env.ServiceAccount
		page.Items = append(page.Items, &sa)
	}
	if page.HasMore && len(page.Items) > 0 {
		page.NextCursor = page.Items[len(page.Items)-1].ID
	}
	return page, nil
}

// GetServiceAccount fetches one service account, enforcing the tenant boundary.
func (a *PgMachineIdentities) GetServiceAccount(ctx context.Context, projectID, serviceAccountID string) (*domain.ServiceAccount, error) {
	_, env, err := a.loadServiceAccount(ctx, projectID, serviceAccountID)
	if err != nil {
		return nil, err
	}
	sa := env.ServiceAccount
	return &sa, nil
}

// UpdateServiceAccount patches scopes / disabled state of a service account.
func (a *PgMachineIdentities) UpdateServiceAccount(ctx context.Context, cmd domain.MachineIDServiceAccountPatchCmd) (*domain.ServiceAccount, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.ServiceAccount, error) {
		row, env, err := a.loadServiceAccount(ctx, cmd.ProjectID, cmd.ServiceAccountID)
		if err != nil {
			return nil, err
		}
		if cmd.Scopes != nil {
			env.Scopes = cmd.Scopes
		}
		if cmd.Disabled != nil {
			env.Disabled = *cmd.Disabled
		}
		data, err := saSetterData(env)
		if err != nil {
			return nil, err
		}
		setter := &models.IamServiceAccountSetter{
			Disabled:  &env.Disabled,
			Data:      data,
			UpdatedAt: ptr(nowUTC()),
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		// TODO outbox event: service_account.updated
		sa := env.ServiceAccount
		return &sa, nil
	})
}

// DeleteServiceAccount removes a service account in the given project.
func (a *PgMachineIdentities) DeleteServiceAccount(ctx context.Context, projectID, serviceAccountID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.loadServiceAccount(ctx, projectID, serviceAccountID)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: service_account.deleted
		return nil
	})
}

// CreateServiceAccountSecret mints a new client secret for a service account.
// The plaintext secret is returned once; only its sha256 hash is persisted.
func (a *PgMachineIdentities) CreateServiceAccountSecret(ctx context.Context, cmd domain.MachineIDSecretCmd) (*domain.MachineIDSecret, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.MachineIDSecret, error) {
		row, env, err := a.loadServiceAccount(ctx, cmd.ProjectID, cmd.ServiceAccountID)
		if err != nil {
			return nil, err
		}
		plaintext, err := machineIDRandomToken(32)
		if err != nil {
			return nil, err
		}
		secretID := newUUID()
		// ClientID identifies the service account for the client-credentials
		// flow; the service-account id doubles as the client id.
		clientID := env.ID
		sec := saSecret{
			SecretID:  secretID,
			ClientID:  clientID,
			Name:      cmd.Name,
			Hash:      machineIDHash(plaintext),
			CreatedAt: nowUTC(),
			ExpiresAt: cmd.ExpiresAt,
		}
		env.Secrets = append(env.Secrets, sec)
		data, err := saSetterData(env)
		if err != nil {
			return nil, err
		}
		setter := &models.IamServiceAccountSetter{Data: data, UpdatedAt: ptr(nowUTC())}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		// TODO outbox event: service_account.secret.created
		return &domain.MachineIDSecret{
			SecretID:     secretID,
			ClientID:     clientID,
			ClientSecret: plaintext,
		}, nil
	})
}

// RevokeServiceAccountSecret marks a service-account client secret revoked.
func (a *PgMachineIdentities) RevokeServiceAccountSecret(ctx context.Context, projectID, serviceAccountID, secretID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, env, err := a.loadServiceAccount(ctx, projectID, serviceAccountID)
		if err != nil {
			return err
		}
		found := false
		for i := range env.Secrets {
			if env.Secrets[i].SecretID == secretID {
				env.Secrets[i].Revoked = true
				found = true
				break
			}
		}
		if !found {
			return domain.ErrNotFound
		}
		data, err := saSetterData(env)
		if err != nil {
			return err
		}
		setter := &models.IamServiceAccountSetter{Data: data, UpdatedAt: ptr(nowUTC())}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return err
		}
		// TODO outbox event: service_account.secret.revoked
		return nil
	})
}

// MintToken issues a signed RS256 JWT access token (jwx Signer) for a service
// account, carrying the SA subject + its scopes.
func (a *PgMachineIdentities) MintToken(ctx context.Context, projectID, serviceAccountID string) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		_, env, err := a.loadServiceAccount(ctx, projectID, serviceAccountID)
		if err != nil {
			return "", err
		}
		if env.Disabled {
			return "", domain.ErrForbidden
		}
		token, err := a.db.Signer().Sign(ctx, projectID, "live", map[string]any{
			"iss":   projectID,
			"sub":   serviceAccountID,
			"pid":   projectID,
			"typ":   "service",
			"scope": env.Scopes,
		}, time.Hour)
		if err != nil {
			return "", err
		}
		// TODO outbox event: service_account.token.minted
		return token, nil
	})
}

// ===== api keys ====================================================

// keyEnvelope is the persisted shape of an API key (the public domain
// aggregate). The secret hash lives in the hash column, not the envelope.
type keyEnvelope struct {
	domain.APIKey
}

// loadAPIKey fetches and tenant-checks an API key.
func (a *PgMachineIdentities) loadAPIKey(ctx context.Context, projectID, keyID string) (*models.IamAPIKey, *domain.APIKey, error) {
	row, err := models.FindIamAPIKey(ctx, a.db.Bobx(), keyID)
	if err != nil {
		return nil, nil, translatePgErr("api_key", err)
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, nil, domain.ErrNotFound
	}
	var key domain.APIKey
	if err := unmarshal(row.Data, &key); err != nil {
		return nil, nil, err
	}
	return row, &key, nil
}

// mintAPIKeySecret builds a plaintext secret of the form "<prefix>.<random>"
// and returns the plaintext, the prefix and the sha256 hash to persist.
func mintAPIKeySecret() (plaintext, prefix, hash string, err error) {
	p, err := machineIDRandomToken(6)
	if err != nil {
		return "", "", "", err
	}
	prefix = "iak_" + p
	body, err := machineIDRandomToken(32)
	if err != nil {
		return "", "", "", err
	}
	plaintext = prefix + "." + body
	return plaintext, prefix, machineIDHash(plaintext), nil
}

// CreateAPIKey creates an API key and returns the plaintext secret exactly once.
func (a *PgMachineIdentities) CreateAPIKey(ctx context.Context, cmd domain.APIKeyCmd) (*domain.APIKey, string, error) {
	type result struct {
		key    *domain.APIKey
		secret string
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		plaintext, prefix, hash, err := mintAPIKeySecret()
		if err != nil {
			return result{}, err
		}
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
			return result{}, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamAPIKeySetter{
			ID:        &key.ID,
			ProjectID: &key.ProjectID,
			Prefix:    &key.Prefix,
			Hash:      &hash,
			Disabled:  &key.Disabled,
			Data:      &rm,
		}
		if _, err := models.IamAPIKeys.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return result{}, domain.ErrConflict
			}
			return result{}, err
		}
		// TODO outbox event: api_key.created
		return result{key: &key, secret: plaintext}, nil
	})
	if err != nil {
		return nil, "", err
	}
	return res.key, res.secret, nil
}

// ListAPIKeys returns every API key in a project (tenant-scoped).
func (a *PgMachineIdentities) ListAPIKeys(ctx context.Context, projectID string) ([]*domain.APIKey, error) {
	rows, err := models.IamAPIKeys.Query(
		sm.Where(models.IamAPIKeys.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.OrderBy(models.IamAPIKeys.Columns.CreatedAt).Asc(),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]*domain.APIKey, 0, len(rows))
	for _, row := range rows {
		var key domain.APIKey
		if err := unmarshal(row.Data, &key); err != nil {
			return nil, err
		}
		k := key
		out = append(out, &k)
	}
	return out, nil
}

// UpdateAPIKey patches API-key metadata / scopes / disabled state.
func (a *PgMachineIdentities) UpdateAPIKey(ctx context.Context, cmd domain.MachineIDAPIKeyPatchCmd) (*domain.APIKey, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.APIKey, error) {
		row, key, err := a.loadAPIKey(ctx, cmd.ProjectID, cmd.KeyID)
		if err != nil {
			return nil, err
		}
		if cmd.Name != "" {
			key.Name = cmd.Name
		}
		if cmd.Scopes != nil {
			key.Scopes = cmd.Scopes
		}
		if cmd.Disabled != nil {
			key.Disabled = *cmd.Disabled
		}
		raw, err := marshal(keyEnvelope{APIKey: *key})
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamAPIKeySetter{Disabled: &key.Disabled, Data: &rm}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		// TODO outbox event: api_key.updated
		return key, nil
	})
}

// RotateAPIKey mints a fresh secret for an existing key and returns the new
// plaintext exactly once; the prefix and hash columns are replaced.
func (a *PgMachineIdentities) RotateAPIKey(ctx context.Context, projectID, keyID string) (*domain.APIKey, string, error) {
	type result struct {
		key    *domain.APIKey
		secret string
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		row, key, err := a.loadAPIKey(ctx, projectID, keyID)
		if err != nil {
			return result{}, err
		}
		plaintext, prefix, hash, err := mintAPIKeySecret()
		if err != nil {
			return result{}, err
		}
		key.Prefix = prefix
		raw, err := marshal(keyEnvelope{APIKey: *key})
		if err != nil {
			return result{}, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamAPIKeySetter{Prefix: &prefix, Hash: &hash, Data: &rm}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return result{}, err
		}
		// TODO outbox event: api_key.rotated
		return result{key: key, secret: plaintext}, nil
	})
	if err != nil {
		return nil, "", err
	}
	return res.key, res.secret, nil
}

// RevokeAPIKey deletes an API key in the given project.
func (a *PgMachineIdentities) RevokeAPIKey(ctx context.Context, projectID, keyID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.loadAPIKey(ctx, projectID, keyID)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: api_key.revoked
		return nil
	})
}
