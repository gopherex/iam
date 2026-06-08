package postgres

// pgAuthenticator is the Postgres-backed implementation of api.Authenticator:
// it validates each security scheme's credential and resolves the calling
// domain.Principal that the ogen SecurityHandler places in the request context.
//
// Token model: every minted credential is one of
//   - a signed RS256 JWT (jwx Signer), carrying "pid" (project), "typ" and the
//     scheme-specific claims (sid/sub/scope/jti/act). The project is read from
//     the *unverified* claims only to select which project's signing keys to
//     verify against — a forged "pid" cannot verify because the signing key is
//     private to the project, so this is not a cross-tenant trust hole. The
//     authorization tenant boundary (project_id in the path) is enforced
//     separately by pkg/api's requireProjectAdmin/requireOperator guards.
//   - an opaque secret (API key, SCIM token, app-client secret) whose sha256
//     hash is stored; we hash the presented secret and look the row up by hash,
//     taking the tenant from the stored row.
//   - the configured master key (operator), compared in constant time.
//
// All verification failures collapse to domain.ErrUnauthorized so the handler
// renders a uniform 401 without leaking which check failed.

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// authDefaultEnv is the environment whose signing keys mint and verify tokens
// until per-environment resolution is wired end-to-end (all signers currently
// emit "live").
const authDefaultEnv = "live"

// pgAuthenticator validates credentials and resolves principals.
type pgAuthenticator struct {
	db        *DB
	masterKey string
}

// NewAuthenticator builds the Postgres-backed Authenticator. masterKey is the
// platform operator credential (empty disables the masterKey scheme).
func NewAuthenticator(db *DB, masterKey string) *pgAuthenticator {
	return &pgAuthenticator{db: db, masterKey: masterKey}
}

var _ api.Authenticator = (*pgAuthenticator)(nil)

// verifyJWT reads the project from the token's unverified claims, then verifies
// the signature/expiry against that project's signing keys and returns the
// verified claims.
func (a *pgAuthenticator) verifyJWT(ctx context.Context, token string) (map[string]any, error) {
	claims := a.db.Signer().UnverifiedClaims(token)
	if claims == nil {
		return nil, domain.ErrUnauthorized
	}
	pid, _ := claims["pid"].(string)
	if pid == "" {
		return nil, domain.ErrUnauthorized
	}
	env := authDefaultEnv
	if e, ok := claims["env"].(string); ok && e != "" {
		env = e
	}
	verified, err := a.db.Signer().Verify(ctx, pid, env, token)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	return verified, nil
}

// User validates a bearerAuth end-user access token: a verified typ=access JWT
// naming a still-live session.
func (a *pgAuthenticator) User(ctx context.Context, token string) (*domain.Principal, error) {
	claims, err := a.verifyJWT(ctx, token)
	if err != nil {
		return nil, err
	}
	if typ, _ := claims["typ"].(string); typ != "access" {
		return nil, domain.ErrUnauthorized
	}
	sid := claimStr(claims, "sid")
	if sid != "" {
		// A revoked/expired session must fail even while the JWT is unexpired.
		row, err := models.FindIamSession(ctx, a.db.Bobx(), sid)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
		if v, ok := row.ExpiresAt.Get(); ok && nowUTC().After(v) {
			return nil, domain.ErrUnauthorized
		}
	}
	return &domain.Principal{
		Kind:        domain.PrincipalUser,
		AccountID:   claimStr(claims, "sub"),
		ProjectID:   claimStr(claims, "pid"),
		Environment: claimEnv(claims),
		SessionID:   sid,
		ClientID:    claimStr(claims, "aud"),
		AAL:         claimInt(claims, "aal"),
		Scopes:      claimScopes(claims),
	}, nil
}

// Admin validates an adminToken: a verified typ=admin JWT. Admin tokens are
// revocable — a matching iam_admin_tokens row must exist and be neither revoked
// nor expired. Impersonation tokens are intentionally rejected here; their only
// valid use is the single-use redeem endpoint.
func (a *pgAuthenticator) Admin(ctx context.Context, token string) (*domain.Principal, error) {
	claims, err := a.verifyJWT(ctx, token)
	if err != nil {
		return nil, err
	}
	typ, _ := claims["typ"].(string)
	if typ != "admin" {
		return nil, domain.ErrUnauthorized
	}
	row, err := models.IamAdminTokens.Query(
		sm.Where(models.IamAdminTokens.Columns.Hash.EQ(psql.Arg(sha256Hex(token)))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	var tok domain.OperatorAdminToken
	if err := unmarshal(row.Data, &tok); err == nil && tok.Revoked {
		return nil, domain.ErrUnauthorized
	}
	if v, ok := row.ExpiresAt.Get(); ok && nowUTC().After(v) {
		return nil, domain.ErrUnauthorized
	}
	return &domain.Principal{
		Kind:        domain.PrincipalAdmin,
		AccountID:   claimStr(claims, "sub"),
		ProjectID:   claimStr(claims, "pid"),
		Environment: claimEnv(claims),
		ClientID:    claimStr(claims, "act"),
		Scopes:      claimScopes(claims),
	}, nil
}

// Master validates the masterKey operator credential against the configured key
// in constant time. An unset key disables the scheme.
func (a *pgAuthenticator) Master(_ context.Context, token string) (*domain.Principal, error) {
	if a.masterKey == "" || token == "" {
		return nil, domain.ErrUnauthorized
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.masterKey)) != 1 {
		return nil, domain.ErrUnauthorized
	}
	return &domain.Principal{Kind: domain.PrincipalOperator, Environment: authDefaultEnv}, nil
}

// Service validates a serviceToken: either a verified typ=service JWT (service
// account) or an opaque API key ("iak_*.<random>") looked up by sha256 hash.
func (a *pgAuthenticator) Service(ctx context.Context, token string) (*domain.Principal, error) {
	if claims, err := a.verifyJWT(ctx, token); err == nil {
		if typ, _ := claims["typ"].(string); typ == "service" {
			return &domain.Principal{
				Kind:        domain.PrincipalService,
				AccountID:   claimStr(claims, "sub"),
				ProjectID:   claimStr(claims, "pid"),
				Environment: claimEnv(claims),
				Scopes:      claimScopes(claims),
			}, nil
		}
	}
	// Opaque API key.
	row, err := models.IamAPIKeys.Query(
		sm.Where(models.IamAPIKeys.Columns.Hash.EQ(psql.Arg(machineIDHash(token)))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if row.Disabled {
		return nil, domain.ErrUnauthorized
	}
	if v, ok := row.ExpiresAt.Get(); ok && nowUTC().After(v) {
		return nil, domain.ErrUnauthorized
	}
	var key domain.APIKey
	_ = unmarshal(row.Data, &key)
	return &domain.Principal{
		Kind:        domain.PrincipalService,
		ProjectID:   row.ProjectID,
		Environment: authDefaultEnv,
		Scopes:      key.Scopes,
	}, nil
}

// SCIM validates a scimToken: an opaque provisioning credential looked up by
// sha256 hash, scoped to its connection.
func (a *pgAuthenticator) SCIM(ctx context.Context, token string) (*domain.Principal, error) {
	row, err := models.IamScimTokens.Query(
		sm.Where(models.IamScimTokens.Columns.Hash.EQ(psql.Arg(fedHashToken(token)))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	var tok domain.ScimToken
	if err := unmarshal(row.Data, &tok); err == nil {
		if !tok.ExpiresAt.IsZero() && nowUTC().After(tok.ExpiresAt) {
			return nil, domain.ErrUnauthorized
		}
	}
	return &domain.Principal{
		Kind:         domain.PrincipalSCIM,
		ProjectID:    row.ProjectID,
		Environment:  authDefaultEnv,
		ConnectionID: row.ConnectionID,
	}, nil
}

// Client validates clientSecretBasic: an app-client id plus one of its opaque
// secrets, compared by sha256 hash.
func (a *pgAuthenticator) Client(ctx context.Context, clientID, secret string) (*domain.Principal, error) {
	app, err := models.FindIamAppClient(ctx, a.db.Bobx(), clientID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	rows, err := models.IamAppSecrets.Query(
		sm.Where(models.IamAppSecrets.Columns.AppID.EQ(psql.Arg(clientID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	want := adminSHA256(secret)
	matched := false
	for _, r := range rows {
		if subtle.ConstantTimeCompare([]byte(r.Hash), []byte(want)) == 1 {
			matched = true
			break
		}
	}
	if !matched {
		return nil, domain.ErrUnauthorized
	}
	return &domain.Principal{
		Kind:        domain.PrincipalClient,
		ProjectID:   app.ProjectID,
		Environment: authDefaultEnv,
		ClientID:    clientID,
	}, nil
}

// OAuth2 validates an oauth2 bearer access token (OIDC-issued): a verified JWT.
func (a *pgAuthenticator) OAuth2(ctx context.Context, token string) (*domain.Principal, error) {
	claims, err := a.verifyJWT(ctx, token)
	if err != nil {
		return nil, err
	}
	if typ, _ := claims["typ"].(string); typ != "access" {
		return nil, domain.ErrUnauthorized
	}
	return &domain.Principal{
		Kind:        domain.PrincipalUser,
		AccountID:   claimStr(claims, "sub"),
		ProjectID:   claimStr(claims, "pid"),
		Environment: claimEnv(claims),
		ClientID:    claimStr(claims, "aud"),
		Scopes:      claimScopes(claims),
	}, nil
}

// ----- claim helpers -----

func claimStr(claims map[string]any, key string) string {
	s, _ := claims[key].(string)
	return s
}

// claimEnv returns the token's environment from its "env" claim, falling back to
// authDefaultEnv for tokens minted before env tagging.
func claimEnv(claims map[string]any) string {
	if e, _ := claims["env"].(string); e != "" {
		return e
	}
	return authDefaultEnv
}

func claimInt(claims map[string]any, key string) int {
	switch v := claims[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// claimScopes normalises the "scope"/"scp" claim, which may arrive as a
// space-delimited string or a JSON array.
func claimScopes(claims map[string]any) []string {
	v, ok := claims["scope"]
	if !ok {
		v = claims["scp"]
	}
	switch s := v.(type) {
	case string:
		if s == "" {
			return nil
		}
		return strings.Fields(s)
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	case []string:
		return s
	}
	return nil
}

var _ = time.Time{}
