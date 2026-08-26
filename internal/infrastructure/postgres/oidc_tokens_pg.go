package postgres

// OIDC token lifecycle: the state that makes a stateless token killable.
//
// Access tokens are signed JWTs a resource server verifies offline, and refresh
// tokens were the same — which meant RFC 7009 revocation could not touch either,
// and a leaked refresh token stayed usable for its whole lifetime with nothing
// able to stop it. Two records fix that without giving up offline verification:
//
//   - iam_refresh_tokens holds a row per issued OIDC refresh token (by sha256),
//     the same table core-auth uses, so rotation, reuse detection and "revoke
//     the session" all work on OIDC grants too;
//   - iam_revoked_tokens names access tokens by jti until they expire anyway, so
//     the verification path has something to check.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// oidcRefreshData is the data envelope of an OIDC refresh-token row. It records
// what the token may be exchanged for, so a rotation cannot quietly widen the
// grant.
type oidcRefreshData struct {
	ClientID string   `json:"client_id,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	Nonce    string   `json:"nonce,omitempty"`
	// Kind marks the row as issued by the OIDC provider rather than core-auth,
	// so the two lifecycles stay distinguishable in one table.
	Kind string `json:"kind,omitempty"`
	// Revoked mirrors the column for the shared session-revocation helper.
	Revoked bool `json:"revoked,omitempty"`
}

const oidcRefreshKind = "oidc"

// storeOIDCRefreshToken records a freshly minted refresh token. Assumes an
// ambient transaction.
func (a *pgOIDCGrants) storeOIDCRefreshToken(
	ctx context.Context, sub oidcTokenSubject, env, token string, ttl time.Duration,
) error {
	raw, err := marshal(oidcRefreshData{
		ClientID: sub.clientID,
		Scopes:   sub.scopes,
		Nonce:    sub.nonce,
		Kind:     oidcRefreshKind,
	})
	if err != nil {
		return err
	}

	data := json.RawMessage(raw)
	expires := null.From(nowUTC().Add(ttl))

	if _, err := models.IamRefreshTokens.Insert(&models.IamRefreshTokenSetter{
		ID:          ptr(newUUID()),
		ProjectID:   ptr(sub.projectID),
		Environment: ptr(env),
		UserID:      ptr(sub.subject),
		SessionID:   ptr(sub.sessionID),
		Hash:        ptr(oidcHashToken(token)),
		ExpiresAt:   &expires,
		Data:        &data,
	}).One(ctx, a.db.Bobx()); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}

	return nil
}

// oidcRedeemRefreshToken consumes a presented refresh token and returns what it
// was issued for.
//
// Rotation is unconditional: the presented token is spent whether or not the
// exchange succeeds afterwards. Presenting an already-spent token is the signal
// that it leaked — the legitimate holder would have moved on to the rotated one
// — so the whole session's tokens are revoked rather than merely refusing this
// one (RFC 9700 §4.14.2).
func (a *pgOIDCGrants) oidcRedeemRefreshToken(
	ctx context.Context, projectID, env, token string,
) (*models.IamRefreshToken, oidcRefreshData, error) {
	var data oidcRefreshData

	rows, err := models.IamRefreshTokens.Query(
		sm.Where(models.IamRefreshTokens.Columns.Hash.EQ(psql.Arg(oidcHashToken(token)))),
		sm.Where(models.IamRefreshTokens.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Limit(1),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, data, fmt.Errorf("read refresh token: %w", err)
	}

	if len(rows) == 0 {
		// Either never issued by us, or already swept after expiry.
		return nil, data, domain.ErrInvalidToken
	}

	row := rows[0]
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &data); err != nil {
			return nil, data, err
		}
	}

	if row.Revoked {
		// Reuse: burn the family the token belonged to. This has to survive the
		// rejection we are about to return, so it deliberately runs on the pool
		// rather than in the caller's transaction — otherwise the rollback would
		// undo the very defense the detection triggered.
		if err := burnSessionRefreshTokens(ctx, a.db, row.ProjectID, row.SessionID); err != nil {
			return nil, data, err
		}

		return row, data, domain.ErrTokenUsed
	}

	if exp, ok := row.ExpiresAt.Get(); ok && !exp.IsZero() && exp.Before(nowIn(ctx)) {
		return nil, data, domain.ErrTokenExpired
	}

	if err := a.markRefreshRevoked(ctx, row, data); err != nil {
		return nil, data, err
	}

	return row, data, nil
}

// markRefreshRevoked spends one refresh-token row.
func (a *pgOIDCGrants) markRefreshRevoked(
	ctx context.Context, row *models.IamRefreshToken, data oidcRefreshData,
) error {
	data.Revoked = true

	raw, err := marshal(data)
	if err != nil {
		return err
	}

	rm := json.RawMessage(raw)
	if err := row.Update(ctx, a.db.Bobx(), &models.IamRefreshTokenSetter{
		Revoked: ptr(true), Data: &rm,
	}); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

// burnSessionRefreshTokens revokes every live refresh token of a session outside
// any ambient transaction, so the revocation commits even when the operation
// that triggered it goes on to fail.
func burnSessionRefreshTokens(ctx context.Context, db *DB, projectID, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	if _, err := db.Pool.Exec(ctx,
		`UPDATE iam_refresh_tokens
		    SET revoked = true,
		        data = jsonb_set(coalesce(data, '{}'::jsonb), '{revoked}', 'true'::jsonb)
		  WHERE project_id = $1 AND session_id = $2 AND revoked = false`,
		projectID, sessionID,
	); err != nil {
		return fmt.Errorf("burn session refresh tokens: %w", err)
	}

	return nil
}

// denyToken adds an access token's jti to the denylist until it expires. A token
// with no jti or no expiry cannot be named, and is left alone rather than
// silently pretended to be revoked.
func denyToken(ctx context.Context, db *DB, projectID, env, jti string, expiresAt time.Time) error {
	if jti == "" || expiresAt.IsZero() {
		return nil
	}

	if _, err := db.TxDB.Exec(ctx,
		`INSERT INTO iam_revoked_tokens (jti, project_id, environment, expires_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (jti) DO NOTHING`,
		jti, projectID, adminEnv(env), expiresAt,
	); err != nil {
		return fmt.Errorf("deny token: %w", err)
	}

	return nil
}

// claimOnce reserves a jti until it expires, reporting whether this caller got
// it. It shares the denylist table with revocation because the question is the
// same one — "has this token id been used up" — and a client assertion is
// single-use by definition (RFC 7523 §3): replaying one is how a captured
// assertion becomes a second authentication.
func claimOnce(ctx context.Context, db *DB, projectID, env, jti string, expiresAt time.Time) (bool, error) {
	if jti == "" || expiresAt.IsZero() {
		return false, nil
	}

	tag, err := db.TxDB.Exec(ctx,
		`INSERT INTO iam_revoked_tokens (jti, project_id, environment, expires_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (jti) DO NOTHING`,
		jti, projectID, adminEnv(env), expiresAt,
	)
	if err != nil {
		return false, fmt.Errorf("claim token id: %w", err)
	}

	return tag.RowsAffected() == 1, nil
}

// tokenDenied reports whether a token's jti has been revoked. Called on every
// verification, so it is a single primary-key lookup.
func tokenDenied(ctx context.Context, db *DB, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}

	var found int
	if err := db.TxDB.QueryRow(ctx,
		`SELECT 1 FROM iam_revoked_tokens WHERE jti = $1 AND expires_at > now()`, jti,
	).Scan(&found); err != nil {
		if isNoRows(err) {
			return false, nil
		}

		return false, fmt.Errorf("check revoked token: %w", err)
	}

	return true, nil
}
