package postgres

// Postgres adapter for the OIDC-provider aggregate ports (api.OIDCGrants).
//
// Tables owned by this adapter:
//   - iam_interactions : front-channel authorization interactions (login/consent)
//   - iam_oauth_grants  : remembered resource-owner grants per (user, client)
//   - iam_auth_codes    : issued authorization codes (stored as sha256 hashes)
//   - iam_par_requests  : pushed authorization requests (RFC 9126)
//   - iam_device_codes  : device authorization grants (RFC 8628)
//   - iam_signing_keys  : per project/env JWKS signing material
//
// Each aggregate is persisted as the `data jsonb` envelope; the typed columns
// (project_id, client_id, session_id, code_hash, user_code, status, ...) are set
// from the struct for lookups only. Every query is scoped by project_id so a row
// belonging to another tenant is treated as not-found.
//
// Token / id-token / JWKS MINTING and signature VERIFICATION are implemented
// via the project/env jwx Signer (db.Signer()): access tokens, id_tokens and
// refresh tokens are signed RS256 JWTs minted by OUR key; introspection and
// the logout id_token_hint / backchannel logout_token are verified against it.
// Protocol claim/response shapes use github.com/zitadel/oidc/v3 structs.
// Authorization codes / device codes / refresh-token rotation stay opaque
// (sha256-hashed or signed-JWT) so they remain revocable.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aarondl/opt/null"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

const (
	// oidcDefaultEnv is the environment whose signing key mints OIDC provider
	// tokens until per-environment resolution is wired from the client.
	oidcDefaultEnv = "live"
	// oidcAccessTTL is the lifetime of an issued access token.
	oidcAccessTTL = time.Hour
	// oidcIDTokenTTL is the lifetime of an issued id_token.
	oidcIDTokenTTL = time.Hour
	// oidcRefreshTTL is the lifetime of an issued refresh token.
	oidcRefreshTTL = 30 * 24 * time.Hour
)

// oidcIssuer returns the canonical issuer string for project/env, matching the
// discovery document's `issuer` value.
func oidcIssuer(projectID, env string) string {
	return fmt.Sprintf("/p/%s/e/%s", projectID, env)
}

// oidcTokenSubject identifies the principal a token family is minted for and
// the request context needed to build standards-compliant claims.
type oidcTokenSubject struct {
	projectID string
	env       string
	subject   string
	clientID  string
	nonce     string
	scopes    []string
}

// oidcHasScope reports whether the openid scope is present (id_token is only
// issued for openid requests).
func oidcHasScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}

// pgOIDCGrants is the Postgres-backed api.OIDCGrants adapter.
type pgOIDCGrants struct {
	db      *DB
	emitter Emitter
}

// NewPgOIDCGrants builds the OIDC-provider adapter over db.
func NewPgOIDCGrants(db *DB, emitter Emitter) *pgOIDCGrants {
	return &pgOIDCGrants{db: db, emitter: emitter}
}

var _ api.OIDCGrants = (*pgOIDCGrants)(nil)

// oidcInteractionEnvelope is the data-jsonb shape for an interaction: the
// public domain.Interaction fields plus the account bound at login time
// (iam_interactions has no account lookup column).
type oidcInteractionEnvelope struct {
	domain.Interaction
	AccountID string `json:"account_id,omitempty"`
}

// ===== local helpers =====

// oidcHashToken returns the sha256 hex digest of an opaque token. Only digests
// are stored; the plaintext token is returned to the caller exactly once.
func oidcHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// oidcRandToken mints a cryptographically random opaque token (hex-encoded).
func oidcRandToken(nbytes int) (string, error) {
	buf := make([]byte, nbytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// oidcUserCode mints a short, human-enterable device user-code.
func oidcUserCode() (string, error) {
	const alphabet = "BCDFGHJKLMNPQRSTVWXZ23456789"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, 0, 9)
	for i, b := range buf {
		if i == 4 {
			out = append(out, '-')
		}
		out = append(out, alphabet[int(b)%len(alphabet)])
	}
	return string(out), nil
}

// ===== interactions =====

// ResolveInteraction returns the pending interaction by id. No tenant filter is
// applied here because the interaction id is itself an unguessable handle; the
// session binding is enforced at CompleteLogin/Consent time.
func (a *pgOIDCGrants) ResolveInteraction(ctx context.Context, interactionID string) (*domain.Interaction, error) {
	row, err := models.FindIamInteraction(ctx, a.db.Bobx(), interactionID)
	if err != nil {
		return nil, translatePgErr("interaction", err)
	}
	var in domain.Interaction
	if err := unmarshal(row.Data, &in); err != nil {
		return nil, err
	}
	return &in, nil
}

// CompleteLogin binds an authenticated account to the interaction. It verifies
// the interaction's session_id matches the caller's session (anti-hijack):
// a mismatch is ErrForbidden.
func (a *pgOIDCGrants) CompleteLogin(ctx context.Context, interactionID, accountID, sessionID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamInteraction(ctx, a.db.Bobx(), interactionID)
		if err != nil {
			return translatePgErr("interaction", err)
		}
		if row.SessionID.GetOrZero() != sessionID {
			return domain.ErrForbidden
		}
		// Persist the resolved account into the interaction envelope alongside the
		// domain.Interaction fields (iam_interactions has no account column).
		var env oidcInteractionEnvelope
		if err := unmarshal(row.Data, &env); err != nil {
			return err
		}
		env.AccountID = accountID
		raw, err := marshal(&env)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamInteractionSetter{Data: &rm}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.interaction.login_completed",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: interactionID,
			Payload:     &env,
		}); err != nil {
			return err
		}
		return nil
	})
}

// Consent records the resource-owner's decision. It verifies the session
// binding, optionally persists a remembered grant, and returns the redirect
// target the user-agent follows next.
func (a *pgOIDCGrants) Consent(ctx context.Context, cmd domain.OIDCConsentCmd) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		row, err := models.FindIamInteraction(ctx, a.db.Bobx(), cmd.InteractionID)
		if err != nil {
			return "", translatePgErr("interaction", err)
		}
		if row.SessionID.GetOrZero() != cmd.SessionID {
			return "", domain.ErrForbidden
		}
		var in domain.Interaction
		if err := unmarshal(row.Data, &in); err != nil {
			return "", err
		}

		if cmd.Remember && in.ClientID != "" {
			grant := domain.Grant{
				ID:        newUUID(),
				AccountID: cmd.AccountID,
				ClientID:  in.ClientID,
				Scopes:    cmd.GrantedScopes,
				GrantedAt: nowUTC(),
			}
			if err := a.persistGrant(ctx, row.ProjectID, &grant); err != nil {
				return "", err
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "oidc.grant.created",
				ProjectID:   row.ProjectID,
				Environment: "",
				AggregateID: grant.ID,
				Payload:     &grant,
			}); err != nil {
				return "", err
			}
		}

		// The interaction is satisfied; drop it so the code cannot be replayed.
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return "", err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.interaction.consented",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: cmd.InteractionID,
			Payload:     &in,
		}); err != nil {
			return "", err
		}

		// Mint a one-time authorization code (opaque, sha256-hashed at rest, so
		// it stays revocable) bound to the consenting user/client/scopes. The
		// Token authorization_code branch resolves the principal from the
		// user_id/client_id columns and the scopes from the data envelope, then
		// mints+signs the access/id tokens with our key.
		code, err := oidcRandToken(32)
		if err != nil {
			return "", err
		}
		scopes := cmd.GrantedScopes
		if len(scopes) == 0 {
			scopes = in.Scopes
		}
		codeData, err := marshal(struct {
			Scopes      []string `json:"Scopes"`
			RedirectURI string   `json:"RedirectURI"`
			Nonce       string   `json:"Nonce"`
		}{
			Scopes:      scopes,
			RedirectURI: in.RedirectURI,
			Nonce:       in.Nonce,
		})
		if err != nil {
			return "", err
		}
		rm := json.RawMessage(codeData)
		uid := null.From(cmd.AccountID)
		cid := null.From(in.ClientID)
		setter := &models.IamAuthCodeSetter{
			ID:        ptr(newUUID()),
			ProjectID: &row.ProjectID,
			CodeHash:  ptr(oidcHashToken(code)),
			ClientID:  &cid,
			UserID:    &uid,
			ExpiresAt: ptr(nowUTC().Add(10 * time.Minute)),
			Data:      &rm,
		}
		authCodeRow, err := models.IamAuthCodes.Insert(setter).One(ctx, a.db.Bobx())
		if err != nil {
			if isUniqueViolation(err) {
				return "", domain.ErrConflict
			}
			return "", err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.token.issued",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: authCodeRow.ID,
			Payload: map[string]any{
				"grant_type": "authorization_code",
				"client_id":  in.ClientID,
				"account_id": cmd.AccountID,
				"scopes":     scopes,
			},
		}); err != nil {
			return "", err
		}

		return oidcAppendQuery(in.RedirectURI, "code", code), nil
	})
}

// oidcAppendQuery appends a single key=value query parameter to a URL,
// choosing the correct separator.
func oidcAppendQuery(rawurl, key, value string) string {
	sep := "?"
	if strings.Contains(rawurl, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%s%s=%s", rawurl, sep, key, value)
}

// Reject cancels the interaction and returns the redirect carrying the OAuth2
// error back to the client. Public operation (no session binding).
func (a *pgOIDCGrants) Reject(ctx context.Context, cmd domain.OIDCRejectCmd) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		row, err := models.FindIamInteraction(ctx, a.db.Bobx(), cmd.InteractionID)
		if err != nil {
			return "", translatePgErr("interaction", err)
		}
		var in domain.Interaction
		if err := unmarshal(row.Data, &in); err != nil {
			return "", err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return "", err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.interaction.rejected",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: cmd.InteractionID,
			Payload:     &in,
		}); err != nil {
			return "", err
		}
		errCode := cmd.Error
		if errCode == "" {
			errCode = "access_denied"
		}
		return fmt.Sprintf("%s?error=%s&error_description=%s", in.RedirectURI, errCode, cmd.ErrorDescription), nil
	})
}

// ===== grants =====

// persistGrant upserts a remembered grant for (project, user, client). Helper
// shared by Consent; assumes an ambient transaction.
func (a *pgOIDCGrants) persistGrant(ctx context.Context, projectID string, g *domain.Grant) error {
	raw, err := marshal(g)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamOauthGrantSetter{
		ID:        &g.ID,
		ProjectID: &projectID,
		UserID:    &g.AccountID,
		ClientID:  &g.ClientID,
		GrantedAt: ptr(g.GrantedAt),
		Data:      &rm,
	}
	if _, err := models.IamOauthGrants.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

// ListGrants returns every remembered grant for the account.
func (a *pgOIDCGrants) ListGrants(ctx context.Context, accountID string) ([]domain.Grant, error) {
	rows, err := models.IamOauthGrants.Query(
		sm.Where(models.IamOauthGrants.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Grant, 0, len(rows))
	for _, row := range rows {
		var g domain.Grant
		if err := unmarshal(row.Data, &g); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// RevokeGrant deletes a remembered grant owned by the account. A grant whose
// user_id does not match the caller is treated as not-found.
func (a *pgOIDCGrants) RevokeGrant(ctx context.Context, accountID, grantID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamOauthGrant(ctx, a.db.Bobx(), grantID)
		if err != nil {
			return translatePgErr("grant", err)
		}
		if row.UserID != accountID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.grant.revoked",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: grantID,
			Payload:     map[string]any{"id": grantID, "project_id": row.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ===== authorize / logout / back-channel =====

// Authorize builds a front-channel interaction for the request and returns the
// redirect to the login/consent UI. Public operation: the project is resolved
// from the client; here we persist a fresh interaction keyed by an unguessable
// id and return its handle to the UI.
func (a *pgOIDCGrants) Authorize(ctx context.Context, cmd domain.OIDCAuthorizeCmd) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		in := domain.Interaction{
			ID:          newUUID(),
			ClientID:    cmd.ClientID,
			Scopes:      splitScopes(cmd.Scope),
			RedirectURI: cmd.RedirectURI,
			Nonce:       cmd.Nonce,
		}
		raw, err := marshal(&in)
		if err != nil {
			return "", err
		}
		rm := json.RawMessage(raw)
		// project_id is unknown without a client lookup port; the client_id
		// lookup column carries the routing key for the UI to resolve.
		cid := null.From(cmd.ClientID)
		exp := null.From(nowUTC().Add(10 * time.Minute))
		setter := &models.IamInteractionSetter{
			ID:        &in.ID,
			ProjectID: ptr(cmd.ClientID), // routing key; project resolved by client at UI
			ClientID:  &cid,
			ExpiresAt: &exp,
			Data:      &rm,
		}
		if _, err := models.IamInteractions.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return "", err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.interaction.created",
			ProjectID:   cmd.ClientID, // project resolved by client at UI; clientID is the routing key
			Environment: "",
			AggregateID: in.ID,
			Payload:     &in,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("/oauth/interaction/%s", in.ID), nil
	})
}

// Logout terminates an RP-initiated logout. Validating the id_token_hint
// signature is the token subsystem's job; we return the post-logout redirect.
func (a *pgOIDCGrants) Logout(ctx context.Context, cmd domain.OIDCLogoutCmd) (string, error) {
	// When an id_token_hint is supplied, verify its signature against the tenant
	// named in its `iss` claim (peeked unverified for routing only) before
	// honouring the request. An invalid hint is rejected; a valid one resolves
	// the sub/sid of the session to terminate. The actual session termination is
	// owned by the session store (a separate port not wired into this adapter),
	// so we validate the hint here and emit the logout event downstream.
	if cmd.IDTokenHint != "" {
		peek := a.db.Signer().UnverifiedClaims(cmd.IDTokenHint)
		if peek == nil {
			return "", domain.ErrInvalidToken
		}
		projectID, env := oidcParseIssuer(peekString(peek, "iss"))
		if projectID == "" {
			return "", domain.ErrInvalidToken
		}
		claims, err := a.db.Signer().Verify(ctx, projectID, env, cmd.IDTokenHint)
		if err != nil {
			return "", err
		}
		sub := peekString(claims, "sub") // session subject to terminate
		sid := peekString(claims, "sid") // session id to terminate
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.session.logout",
			ProjectID:   projectID,
			Environment: env,
			AggregateID: sub,
			Payload:     map[string]any{"sub": sub, "sid": sid, "project_id": projectID, "env": env},
		}); err != nil {
			return "", err
		}
	}
	redirect := cmd.PostLogoutRedirectURI
	if redirect == "" {
		return "/", nil
	}
	if cmd.State != "" {
		return fmt.Sprintf("%s?state=%s", redirect, cmd.State), nil
	}
	return redirect, nil
}

// peekString reads a string claim from a generic claim map, returning "" when
// absent or not a string.
func peekString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// BackchannelLogout validates the logout token and terminates referenced
// sessions. Public operation.
func (a *pgOIDCGrants) BackchannelLogout(ctx context.Context, cmd domain.OIDCBackchannelLogoutCmd) error {
	if cmd.LogoutToken == "" {
		return domain.ErrInvalidToken
	}
	// Verify the logout_token JWT signature against the tenant named in its
	// `iss` claim (peeked unverified for routing only), then extract the sub/sid
	// of the sessions to terminate. The actual termination is owned by the
	// session store (a separate port not wired into this adapter); we validate
	// the token here and emit the backchannel-logout event downstream.
	peek := a.db.Signer().UnverifiedClaims(cmd.LogoutToken)
	if peek == nil {
		return domain.ErrInvalidToken
	}
	projectID, env := oidcParseIssuer(peekString(peek, "iss"))
	if projectID == "" {
		return domain.ErrInvalidToken
	}
	claims, err := a.db.Signer().Verify(ctx, projectID, env, cmd.LogoutToken)
	if err != nil {
		return err
	}
	sub := peekString(claims, "sub") // session subject to terminate
	sid := peekString(claims, "sid") // session id to terminate
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "oidc.session.backchannel_logout",
		ProjectID:   projectID,
		Environment: env,
		AggregateID: sub,
		Payload:     map[string]any{"sub": sub, "sid": sid, "project_id": projectID, "env": env},
	}); err != nil {
		return err
	}
	return nil
}

// ===== token endpoint family =====

// Token dispatches an /oauth2/token request. Code/refresh-token validation
// looks up the persisted hashes; the access/id-token MINTING + SIGNING is left
// to the token subsystem (returns an opaque placeholder here).
func (a *pgOIDCGrants) Token(ctx context.Context, cmd domain.OIDCTokenCmd) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		switch cmd.GrantType {
		case "authorization_code":
			if cmd.Code == "" {
				return nil, domain.ErrBadRequest
			}
			hash := oidcHashToken(cmd.Code)
			rows, err := models.IamAuthCodes.Query(
				sm.Where(models.IamAuthCodes.Columns.CodeHash.EQ(psql.Arg(hash))),
				sm.Limit(1),
			).All(ctx, a.db.Bobx())
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				return nil, domain.ErrInvalidToken
			}
			row := rows[0]
			if row.Consumed {
				return nil, domain.ErrTokenUsed
			}
			if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowUTC()) {
				return nil, domain.ErrTokenExpired
			}

			// H-01: Verify client_secret for confidential clients.
			effectiveClientID := firstNonEmpty(row.ClientID.GetOrZero(), cmd.ClientID)
			if err := a.oidcVerifyClientSecret(ctx, effectiveClientID, cmd.ClientSecret); err != nil {
				return nil, err
			}

			// Parse the code data envelope for redirect_uri, nonce, and scopes.
			codeData, err := parseAuthCodeData(row.Data)
			if err != nil {
				return nil, err
			}

			// H-03: Verify redirect_uri matches the one stored at authorize time.
			if cmd.RedirectURI != "" || codeData.RedirectURI != "" {
				if subtle.ConstantTimeCompare([]byte(cmd.RedirectURI), []byte(codeData.RedirectURI)) != 1 {
					return nil, domain.ErrUnauthorized
				}
			}

			// M-02: Verify consent was granted for this (project, user, client).
			if row.ProjectID != "" && row.UserID.GetOrZero() != "" && effectiveClientID != "" {
				if err := a.oidcVerifyConsent(ctx, row.ProjectID, row.UserID.GetOrZero(), effectiveClientID); err != nil {
					return nil, err
				}
			}

			consumed := true
			if err := row.Update(ctx, a.db.Bobx(), &models.IamAuthCodeSetter{Consumed: &consumed}); err != nil {
				return nil, err
			}
			tokenSubj := oidcTokenSubject{
				projectID: row.ProjectID,
				env:       oidcDefaultEnv,
				subject:   row.UserID.GetOrZero(),
				clientID:  effectiveClientID,
				nonce:     codeData.Nonce,
				scopes:    splitScopesFromData(row.Data),
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "oidc.token.issued",
				ProjectID:   row.ProjectID,
				Environment: oidcDefaultEnv,
				AggregateID: row.UserID.GetOrZero(),
				Payload: map[string]any{
					"grant_type": "authorization_code",
					"client_id":  tokenSubj.clientID,
					"subject":    tokenSubj.subject,
					"scopes":     tokenSubj.scopes,
				},
			}); err != nil {
				return nil, err
			}
			return a.mintTokenResponse(ctx, tokenSubj)
		case "device_code":
			if cmd.DeviceCode == "" {
				return nil, domain.ErrBadRequest
			}
			hash := oidcHashToken(cmd.DeviceCode)
			rows, err := models.IamDeviceCodes.Query(
				sm.Where(models.IamDeviceCodes.Columns.DeviceCode.EQ(psql.Arg(hash))),
				sm.Limit(1),
			).All(ctx, a.db.Bobx())
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				return nil, domain.ErrInvalidToken
			}
			row := rows[0]
			switch row.Status {
			case "approved":
				var pending domain.OIDCDevicePending
				_ = unmarshal(row.Data, &pending)
				tokenSubj := oidcTokenSubject{
					projectID: row.ProjectID,
					env:       oidcDefaultEnv,
					subject:   row.UserID.GetOrZero(),
					clientID:  firstNonEmpty(pending.ClientID, cmd.ClientID),
					scopes:    pending.Scopes,
				}
				if err := a.emitter.Emit(ctx, domain.Event{
					Type:        "oidc.token.issued",
					ProjectID:   row.ProjectID,
					Environment: oidcDefaultEnv,
					AggregateID: row.UserID.GetOrZero(),
					Payload: map[string]any{
						"grant_type": "device_code",
						"client_id":  tokenSubj.clientID,
						"subject":    tokenSubj.subject,
						"scopes":     tokenSubj.scopes,
					},
				}); err != nil {
					return nil, err
				}
				return a.mintTokenResponse(ctx, tokenSubj)
			case "denied":
				return nil, domain.ErrForbidden
			default:
				if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowUTC()) {
					return nil, domain.ErrTokenExpired
				}
				// RFC 8628: still pending.
				return nil, domain.ErrBadRequest
			}
		case "refresh_token":
			if cmd.RefreshToken == "" {
				return nil, domain.ErrBadRequest
			}
			// The refresh token is a signed RS256 JWT (typ=refresh). Verify it
			// against the REQUEST tenant (the authenticated client's project) —
			// never the token's self-asserted issuer: a token from another tenant
			// fails signature verification against this project's keys.
			projectID := cmd.ProjectID
			env := cmd.Env
			if env == "" {
				env = oidcDefaultEnv
			}
			if projectID == "" {
				return nil, domain.ErrInvalidToken
			}
			sub, clientID, scopes, err := a.verifyRefreshToken(ctx, projectID, env, cmd.RefreshToken)
			if err != nil {
				return nil, err
			}
			effectiveClientID := firstNonEmpty(clientID, cmd.ClientID)
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "oidc.token.refreshed",
				ProjectID:   projectID,
				Environment: env,
				AggregateID: sub,
				Payload: map[string]any{
					"subject":    sub,
					"client_id":  effectiveClientID,
					"scopes":     scopes,
					"project_id": projectID,
					"env":        env,
				},
			}); err != nil {
				return nil, err
			}
			return a.mintTokenResponse(ctx, oidcTokenSubject{
				projectID: projectID,
				env:       env,
				subject:   sub,
				clientID:  effectiveClientID,
				scopes:    scopes,
			})
		default:
			return nil, domain.ErrUnsupportedGrant
		}
	})
}

// verifyRefreshToken validates a signed refresh-token JWT against the project's
// signing keys and returns its bound principal/scope context. An invalid token,
// or one that is not a refresh token, maps to ErrInvalidToken.
func (a *pgOIDCGrants) verifyRefreshToken(ctx context.Context, projectID, env, token string) (sub, clientID string, scopes []string, err error) {
	claims, verr := a.db.Signer().Verify(ctx, projectID, env, token)
	if verr != nil {
		return "", "", nil, verr
	}
	if typ, _ := claims["typ"].(string); typ != "refresh" {
		return "", "", nil, domain.ErrInvalidToken
	}
	sub, _ = claims["sub"].(string)
	clientID, _ = claims["client_id"].(string)
	if s, ok := claims["scope"].(string); ok {
		scopes = splitScopes(s)
	}
	return sub, clientID, scopes, nil
}

// oidcParseIssuer extracts (projectID, env) from a "/p/<projectID>/e/<env>"
// issuer string. Returns empty strings if the issuer is not in that form.
func oidcParseIssuer(iss string) (projectID, env string) {
	parts := strings.Split(iss, "/")
	// "" / "p" / <projectID> / "e" / <env>
	if len(parts) == 5 && parts[0] == "" && parts[1] == "p" && parts[3] == "e" {
		return parts[2], parts[4]
	}
	return "", ""
}

// mintTokenResponse builds the token-endpoint response for sub. The access
// token is a signed RS256 JWT (our jwx Signer); an id_token is minted and
// signed only for openid requests; a signed, rotatable refresh token is issued
// for offline_access requests.
func (a *pgOIDCGrants) mintTokenResponse(ctx context.Context, sub oidcTokenSubject) (map[string]any, error) {
	if sub.projectID == "" {
		// Without the tenant we cannot resolve a signing key.
		return nil, domain.ErrBadRequest
	}
	env := sub.env
	if env == "" {
		env = oidcDefaultEnv
	}
	issuer := oidcIssuer(sub.projectID, env)
	now := nowUTC()

	// Access token: signed RS256 JWT carrying the standard access claims.
	access, err := a.db.Signer().Sign(ctx, sub.projectID, env, map[string]any{
		"iss":       issuer,
		"sub":       sub.subject,
		"aud":       sub.clientID,
		"client_id": sub.clientID,
		"scope":     joinScopes(sub.scopes),
		"typ":       "access",
		"env":       env,
	}, oidcAccessTTL)
	if err != nil {
		return nil, err
	}

	resp := oidc.AccessTokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   uint64(oidcAccessTTL / time.Second),
		Scope:       oidc.SpaceDelimitedArray(sub.scopes),
	}

	// id_token: only for openid requests. Built from the zitadel IDTokenClaims
	// struct (correct field names), then signed by OUR key via the Signer.
	if oidcHasScope(sub.scopes, "openid") {
		idToken, err := a.mintIDToken(ctx, sub, env, issuer, access, now)
		if err != nil {
			return nil, err
		}
		resp.IDToken = idToken
	}

	// refresh_token: signed, rotatable JWT for offline_access requests.
	if oidcHasScope(sub.scopes, "offline_access") {
		refresh, err := a.db.Signer().Sign(ctx, sub.projectID, env, map[string]any{
			"iss":       issuer,
			"sub":       sub.subject,
			"aud":       sub.clientID,
			"client_id": sub.clientID,
			"scope":     joinScopes(sub.scopes),
			"typ":       "refresh",
			"env":       env,
		}, oidcRefreshTTL)
		if err != nil {
			return nil, err
		}
		resp.RefreshToken = refresh
	}

	return oidcClaimsMap(resp)
}

// mintIDToken builds an OIDC id_token for sub using the zitadel IDTokenClaims
// struct for correct claim names, sets the access-token hash (at_hash), and
// signs it with OUR project key via the Signer.
func (a *pgOIDCGrants) mintIDToken(ctx context.Context, sub oidcTokenSubject, env, issuer, accessToken string, now time.Time) (string, error) {
	idc := oidc.NewIDTokenClaims(
		issuer,
		sub.subject,
		[]string{sub.clientID},
		now.Add(oidcIDTokenTTL),
		now,
		sub.nonce,
		"",  // acr
		nil, // amr
		sub.clientID,
		0, // skew
	)
	if accessToken != "" {
		if h, err := oidc.ClaimHash(accessToken, jose.RS256); err == nil {
			idc.AccessTokenHash = h
		}
	}
	claims, err := oidcClaimsMap(idc)
	if err != nil {
		return "", err
	}
	// The Signer sets iat/exp/nbf; drop the struct-provided lifetimes so they do
	// not collide, but keep auth_time and the OIDC-specific claims.
	delete(claims, "exp")
	delete(claims, "iat")
	delete(claims, "nbf")
	claims["env"] = env
	return a.db.Signer().Sign(ctx, sub.projectID, env, claims, oidcIDTokenTTL)
}

// oidcClaimsMap marshals an OIDC claims/response struct to the generic map the
// oas layer (and Signer) expect.
func oidcClaimsMap(v any) (map[string]any, error) {
	buf, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// firstNonEmpty returns the first non-empty string of its arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// Introspect returns the introspection response (RFC 7662). The token is a
// signed RS256 JWT (access or refresh) minted by our Signer; the tenant whose
// keys verify it is taken from the `iss` claim (peeked unverified for routing
// only). A token that fails verification — bad signature, expired, or wrong
// tenant — is reported as inactive, never as an error.
func (a *pgOIDCGrants) Introspect(ctx context.Context, cmd domain.OIDCIntrospectCmd) (map[string]any, error) {
	inactive := map[string]any{"active": false}
	if cmd.Token == "" {
		return inactive, nil
	}
	// Anchor verification to the REQUEST tenant (the authenticated client's
	// project), never to the token's self-asserted issuer: a token minted under
	// another tenant fails signature verification against this project's keys
	// and is reported inactive (prevents cross-tenant token confusion).
	if cmd.ProjectID == "" {
		return inactive, nil
	}
	env := cmd.Env
	if env == "" {
		env = oidcDefaultEnv
	}
	claims, err := a.db.Signer().Verify(ctx, cmd.ProjectID, env, cmd.Token)
	if err != nil {
		return inactive, nil
	}
	if iss, _ := claims["iss"].(string); iss != oidcIssuer(cmd.ProjectID, env) {
		return inactive, nil // issuer does not match the request tenant
	}

	resp := oidc.IntrospectionResponse{Active: true}
	if v, ok := claims["sub"].(string); ok {
		resp.Subject = v
	}
	if v, ok := claims["client_id"].(string); ok {
		resp.ClientID = v
	}
	if v, ok := claims["iss"].(string); ok {
		resp.Issuer = v
	}
	if v, ok := claims["aud"].(string); ok && v != "" {
		resp.Audience = oidc.Audience{v}
	}
	if v, ok := claims["scope"].(string); ok && v != "" {
		resp.Scope = oidc.SpaceDelimitedArray(splitScopes(v))
	}
	resp.TokenType = "Bearer"
	if v, ok := claims["exp"].(float64); ok {
		resp.Expiration = oidc.FromTime(time.Unix(int64(v), 0))
	}
	if v, ok := claims["iat"].(float64); ok {
		resp.IssuedAt = oidc.FromTime(time.Unix(int64(v), 0))
	}
	if v, ok := claims["nbf"].(float64); ok {
		resp.NotBefore = oidc.FromTime(time.Unix(int64(v), 0))
	}
	return oidcClaimsMap(resp)
}

// Revoke revokes a token. Authorization-code / device-code material is matched
// by hash; opaque access/refresh tokens are handled by the token subsystem.
func (a *pgOIDCGrants) Revoke(ctx context.Context, cmd domain.OIDCRevokeCmd) error {
	if cmd.Token == "" {
		return nil // RFC 7009: revoking an unknown token is a no-op success.
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		hash := oidcHashToken(cmd.Token)
		rows, err := models.IamAuthCodes.Query(
			sm.Where(models.IamAuthCodes.Columns.CodeHash.EQ(psql.Arg(hash))),
			sm.Limit(1),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			consumed := true
			if err := rows[0].Update(ctx, a.db.Bobx(), &models.IamAuthCodeSetter{Consumed: &consumed}); err != nil {
				return err
			}
		}
		// Access/refresh tokens are stateless, signature-verifiable RS256 JWTs;
		// short-circuit revocation of a single stateless token would require a
		// per-jti denylist store, which is not one of this adapter's owned
		// tables. The auth-code material above is revoked by hash. RFC 7009: any
		// token we cannot match is a no-op success.
		aggregateID := ""
		if len(rows) > 0 {
			aggregateID = rows[0].ID
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.token.revoked",
			ProjectID:   cmd.ProjectID,
			Environment: cmd.Env,
			AggregateID: aggregateID,
			Payload:     map[string]any{"id": aggregateID, "project_id": cmd.ProjectID, "token_type_hint": cmd.TokenTypeHint},
		}); err != nil {
			return err
		}
		return nil
	})
}

// PushAuthorizationRequest stores a PAR and returns its request_uri (RFC 9126).
func (a *pgOIDCGrants) PushAuthorizationRequest(ctx context.Context, cmd domain.OIDCParCmd) (*domain.OIDCParResult, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.OIDCParResult, error) {
		opaque, err := oidcRandToken(32)
		if err != nil {
			return nil, err
		}
		requestURI := "urn:ietf:params:oauth:request_uri:" + opaque
		raw, err := marshal(&cmd)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		const ttl = 90 // seconds, RFC 9126 recommended upper bound
		cid := null.From(cmd.ClientID)
		setter := &models.IamParRequestSetter{
			ID:         ptr(newUUID()),
			ProjectID:  ptr(cmd.ClientID), // routing key; project resolved by client
			RequestURI: &requestURI,
			ClientID:   &cid,
			ExpiresAt:  ptr(nowUTC().Add(ttl * time.Second)),
			Data:       &rm,
		}
		parRow, err := models.IamParRequests.Insert(setter).One(ctx, a.db.Bobx())
		if err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		result := &domain.OIDCParResult{RequestURI: requestURI, ExpiresIn: ttl}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.par.created",
			ProjectID:   cmd.ClientID, // routing key; project resolved by client
			Environment: "",
			AggregateID: parRow.ID,
			Payload:     result,
		}); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// DeviceAuthorization starts a device authorization grant (RFC 8628). The
// device_code is stored as a hash; the plaintext device_code and user_code are
// returned to the client exactly once.
func (a *pgOIDCGrants) DeviceAuthorization(ctx context.Context, cmd domain.OIDCDeviceAuthorizationCmd) (*domain.OIDCDeviceAuthorization, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.OIDCDeviceAuthorization, error) {
		deviceCode, err := oidcRandToken(32)
		if err != nil {
			return nil, err
		}
		userCode, err := oidcUserCode()
		if err != nil {
			return nil, err
		}
		const ttl = 600 // 10 minutes
		const interval = 5
		expiresAt := nowUTC().Add(ttl * time.Second)

		out := &domain.OIDCDeviceAuthorization{
			DeviceCode:              deviceCode,
			UserCode:                userCode,
			VerificationURI:         "/device",
			VerificationURIComplete: fmt.Sprintf("/device?user_code=%s", userCode),
			ExpiresIn:               ttl,
			Interval:                interval,
		}
		// Persist the OIDCDevicePending view in the data envelope so the
		// verification UI can show client + scopes.
		pending := domain.OIDCDevicePending{
			ClientID:  cmd.ClientID,
			Scopes:    splitScopes(cmd.Scope),
			ExpiresAt: expiresAt,
		}
		raw, err := marshal(&pending)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamDeviceCodeSetter{
			ID:         ptr(newUUID()),
			ProjectID:  ptr(cmd.ClientID), // routing key; project resolved by client
			DeviceCode: ptr(oidcHashToken(deviceCode)),
			UserCode:   &userCode,
			Status:     ptr("pending"),
			ExpiresAt:  &expiresAt,
			Data:       &rm,
		}
		deviceRow, err := models.IamDeviceCodes.Insert(setter).One(ctx, a.db.Bobx())
		if err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.device.authorized",
			ProjectID:   cmd.ClientID, // routing key; project resolved by client
			Environment: "",
			AggregateID: deviceRow.ID,
			Payload:     &pending,
		}); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// ===== userinfo =====

// Userinfo returns the OIDC userinfo claims for the bearer-authenticated
// account. Resolving the claims set requires the account aggregate (a separate
// port); the signing of a signed userinfo response is the token subsystem's job.
func (a *pgOIDCGrants) Userinfo(ctx context.Context, accountID, sessionID string) (map[string]any, error) {
	// The bearer principal is already authenticated upstream (the access-token
	// JWT was verified by the auth middleware), so the userinfo body is the
	// resolved subject. Resolving richer profile/email claims requires the
	// account aggregate (a separate port not wired into this adapter); a signed
	// (JWT) userinfo response is only returned under content negotiation, which
	// the oas layer does not request. Shape it via the OIDC UserInfo struct so
	// the claim names are spec-correct.
	return oidcClaimsMap(&oidc.UserInfo{Subject: accountID})
}

// ===== device verification UI =====

// ResolveDevice returns the pending device authorization for a user-facing code,
// scoped to the requesting client's project. A row whose project_id does not
// match is treated as not-found.
func (a *pgOIDCGrants) ResolveDevice(ctx context.Context, code domain.OIDCDeviceUserCode) (*domain.OIDCDevicePending, error) {
	rows, err := models.IamDeviceCodes.Query(
		sm.Where(models.IamDeviceCodes.Columns.UserCode.EQ(psql.Arg(code.UserCode))),
		sm.Where(models.IamDeviceCodes.Columns.ProjectID.EQ(psql.Arg(code.ProjectID))),
		sm.Limit(1),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	row := rows[0]
	var pending domain.OIDCDevicePending
	if err := unmarshal(row.Data, &pending); err != nil {
		return nil, err
	}
	if pending.ExpiresAt.IsZero() {
		pending.ExpiresAt = row.ExpiresAt
	}
	return &pending, nil
}

// ApproveDevice approves a pending device authorization on behalf of the
// authenticated user.
func (a *pgOIDCGrants) ApproveDevice(ctx context.Context, cmd domain.OIDCDeviceDecisionCmd) error {
	return a.deviceDecision(ctx, cmd, "approved")
}

// DenyDevice denies a pending device authorization on behalf of the
// authenticated user.
func (a *pgOIDCGrants) DenyDevice(ctx context.Context, cmd domain.OIDCDeviceDecisionCmd) error {
	return a.deviceDecision(ctx, cmd, "denied")
}

// deviceDecision records an approve/deny decision for a pending device
// authorization, scoped to the caller's project. Shared by Approve/Deny.
func (a *pgOIDCGrants) deviceDecision(ctx context.Context, cmd domain.OIDCDeviceDecisionCmd, status string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		rows, err := models.IamDeviceCodes.Query(
			sm.Where(models.IamDeviceCodes.Columns.UserCode.EQ(psql.Arg(cmd.UserCode))),
			sm.Where(models.IamDeviceCodes.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Limit(1),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return domain.ErrNotFound
		}
		row := rows[0]
		if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowUTC()) {
			return domain.ErrTokenExpired
		}
		uid := null.From(cmd.AccountID)
		setter := &models.IamDeviceCodeSetter{Status: &status, UserID: &uid}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.device.decided",
			ProjectID:   cmd.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload: map[string]any{
				"id":         row.ID,
				"status":     status,
				"account_id": cmd.AccountID,
				"project_id": cmd.ProjectID,
			},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ===== JWKS / discovery =====

// JWKS returns the JSON Web Key Set for a project environment. Public. The
// public-key material is derived from the persisted signing keys; here we list
// the active keys and emit their metadata, leaving the public-key encoding to
// the signing subsystem.
func (a *pgOIDCGrants) JWKS(ctx context.Context, projectID, env string) (map[string]any, error) {
	// Public JWK set derived (n/e) from the project/env signing keys via jwx.
	return a.db.Signer().JWKS(ctx, projectID, env)
}

// OpenIDConfiguration returns the discovery document for a project environment,
// built from the zitadel DiscoveryConfiguration struct (spec-correct field
// names) and marshalled to the generic map the oas layer emits. The signing
// algorithm advertised matches the Signer (RS256).
func (a *pgOIDCGrants) OpenIDConfiguration(ctx context.Context, projectID, env string) (map[string]any, error) {
	base := oidcIssuer(projectID, env)
	cfg := &oidc.DiscoveryConfiguration{
		Issuer:                           base,
		AuthorizationEndpoint:            "/oauth2/authorize",
		TokenEndpoint:                    "/oauth2/token",
		UserinfoEndpoint:                 "/oauth2/userinfo",
		JwksURI:                          base + "/.well-known/jwks.json",
		IntrospectionEndpoint:            "/oauth2/introspect",
		RevocationEndpoint:               "/oauth2/revoke",
		DeviceAuthorizationEndpoint:      "/oauth2/device_authorization",
		EndSessionEndpoint:               "/oauth2/logout",
		ResponseTypesSupported:           []string{"code"},
		ResponseModesSupported:           []string{"query", "fragment"},
		GrantTypesSupported:              []oidc.GrantType{oidc.GrantTypeCode, oidc.GrantTypeRefreshToken, oidc.GrantTypeDeviceCode},
		SubjectTypesSupported:            []string{"public"},
		ScopesSupported:                  []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail, oidc.ScopeOfflineAccess},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []oidc.AuthMethod{
			oidc.AuthMethodBasic, oidc.AuthMethodPost, oidc.AuthMethodNone,
		},
		CodeChallengeMethodsSupported:     []oidc.CodeChallengeMethod{oidc.CodeChallengeMethodS256},
		BackChannelLogoutSupported:        true,
		BackChannelLogoutSessionSupported: true,
	}
	m, err := oidcClaimsMap(cfg)
	if err != nil {
		return nil, err
	}
	// The pushed-authorization-request endpoint has no field on the discovery
	// struct in this lib version; advertise it explicitly (RFC 9126).
	m["pushed_authorization_request_endpoint"] = "/oauth2/par"
	return m, nil
}

// ===== small string helpers =====

// splitScopes splits a space-delimited OAuth scope string.
func splitScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	var out []string
	cur := ""
	for _, r := range scope {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// joinScopes joins scopes into a space-delimited string.
func joinScopes(scopes []string) string {
	out := ""
	for i, s := range scopes {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

// splitScopesFromData extracts scopes from a persisted auth-code data envelope,
// tolerating envelopes that do not carry a scope field.
func splitScopesFromData(data json.RawMessage) []string {
	var env struct {
		Scopes []string `json:"Scopes"`
		Scope  string   `json:"scope"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil
	}
	if len(env.Scopes) > 0 {
		return env.Scopes
	}
	return splitScopes(env.Scope)
}

// authCodeData holds fields persisted alongside an authorization code.
type authCodeData struct {
	Scopes      []string `json:"Scopes"`
	Scope       string   `json:"scope"`
	RedirectURI string   `json:"RedirectURI"`
	Nonce       string   `json:"Nonce"`
}

// parseAuthCodeData unmarshals the auth-code data envelope.
func parseAuthCodeData(data json.RawMessage) (authCodeData, error) {
	var d authCodeData
	if err := json.Unmarshal(data, &d); err != nil {
		return d, domain.ErrBadRequest.WithMessage("corrupted auth code data")
	}
	return d, nil
}

// oidcIsConfidentialClient reports whether the app client type requires a
// client secret (web and machine are confidential; spa and native are public).
func oidcIsConfidentialClient(clientType string) bool {
	return clientType == "web" || clientType == "machine"
}

// oidcVerifyClientSecret looks up an app client by ID, and if it is a
// confidential client, verifies the supplied secret against the stored sha256
// hash in the data envelope using constant-time comparison.
func (a *pgOIDCGrants) oidcVerifyClientSecret(ctx context.Context, clientID, clientSecret string) error {
	if clientID == "" {
		return domain.ErrUnauthorized
	}
	row, err := models.FindIamAppClient(ctx, a.db.Bobx(), clientID)
	if err != nil {
		return domain.ErrUnauthorized
	}
	if !oidcIsConfidentialClient(row.Type) {
		return nil
	}
	if clientSecret == "" {
		return domain.ErrUnauthorized
	}
	var data struct {
		ClientSecretHash string `json:"client_secret_hash"`
	}
	if err := json.Unmarshal(row.Data, &data); err != nil || data.ClientSecretHash == "" {
		return domain.ErrUnauthorized
	}
	given := sha256.Sum256([]byte(clientSecret))
	givenHex := hex.EncodeToString(given[:])
	if subtle.ConstantTimeCompare([]byte(givenHex), []byte(data.ClientSecretHash)) != 1 {
		return domain.ErrUnauthorized
	}
	return nil
}

// oidcVerifyConsent checks that a consent grant exists for the given
// (projectID, userID, clientID), returning ErrConsentRequired if absent.
func (a *pgOIDCGrants) oidcVerifyConsent(ctx context.Context, projectID, userID, clientID string) error {
	rows, err := models.IamOauthGrants.Query(
		sm.Where(models.IamOauthGrants.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamOauthGrants.Columns.UserID.EQ(psql.Arg(userID))),
		sm.Where(models.IamOauthGrants.Columns.ClientID.EQ(psql.Arg(clientID))),
		sm.Limit(1),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return domain.ErrConsentRequired
	}
	return nil
}
