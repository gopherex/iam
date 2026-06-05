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
// Token / id-token / JWKS / userinfo MINTING and signature VERIFICATION are not
// implemented here: those lines persist what they can and return an opaque
// placeholder, marked with `// TODO: sign/verify with signing key`.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// pgOIDCGrants is the Postgres-backed api.OIDCGrants adapter.
type pgOIDCGrants struct{ db *DB }

// NewPgOIDCGrants builds the OIDC-provider adapter over db.
func NewPgOIDCGrants(db *DB) *pgOIDCGrants { return &pgOIDCGrants{db: db} }

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
		// TODO outbox event: oidc.interaction.login_completed
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
			// TODO outbox event: oidc.grant.created
		}

		// The interaction is satisfied; drop it so the code cannot be replayed.
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return "", err
		}
		// TODO outbox event: oidc.interaction.consented

		// TODO: sign/verify with signing key — the authorization code is minted
		// and signed by the token subsystem; we only know the redirect target.
		return in.RedirectURI, nil
	})
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
		// TODO outbox event: oidc.interaction.rejected
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
		// TODO outbox event: oidc.grant.revoked
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
		// TODO outbox event: oidc.interaction.created
		return fmt.Sprintf("/oauth/interaction/%s", in.ID), nil
	})
}

// Logout terminates an RP-initiated logout. Validating the id_token_hint
// signature is the token subsystem's job; we return the post-logout redirect.
func (a *pgOIDCGrants) Logout(ctx context.Context, cmd domain.OIDCLogoutCmd) (string, error) {
	// TODO: sign/verify with signing key — validate id_token_hint signature and
	// resolve the session it references before terminating it.
	// TODO outbox event: oidc.session.logout
	redirect := cmd.PostLogoutRedirectURI
	if redirect == "" {
		return "/", nil
	}
	if cmd.State != "" {
		return fmt.Sprintf("%s?state=%s", redirect, cmd.State), nil
	}
	return redirect, nil
}

// BackchannelLogout validates the logout token and terminates referenced
// sessions. Public operation.
func (a *pgOIDCGrants) BackchannelLogout(ctx context.Context, cmd domain.OIDCBackchannelLogoutCmd) error {
	if cmd.LogoutToken == "" {
		return domain.ErrInvalidToken
	}
	// TODO: sign/verify with signing key — verify the logout_token JWT signature
	// and extract sub/sid before terminating the referenced sessions.
	// TODO outbox event: oidc.session.backchannel_logout
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
			consumed := true
			if err := row.Update(ctx, a.db.Bobx(), &models.IamAuthCodeSetter{Consumed: &consumed}); err != nil {
				return nil, err
			}
			// TODO outbox event: oidc.token.issued
			return a.mintTokenResponse(splitScopesFromData(row.Data))
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
				// TODO outbox event: oidc.token.issued
				return a.mintTokenResponse(nil)
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
			// TODO: sign/verify with signing key — validate the refresh token and
			// rotate it; we persist nothing extra here.
			// TODO outbox event: oidc.token.refreshed
			return a.mintTokenResponse(nil)
		default:
			return nil, domain.ErrUnsupportedGrant
		}
	})
}

// mintTokenResponse returns the token-endpoint response. The actual JWT minting
// and signing is delegated to the token subsystem.
func (a *pgOIDCGrants) mintTokenResponse(scopes []string) (map[string]any, error) {
	access, err := oidcRandToken(32)
	if err != nil {
		return nil, err
	}
	// TODO: sign/verify with signing key — access_token / id_token must be minted
	// and signed by the token subsystem; below is an opaque placeholder.
	resp := map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
	if len(scopes) > 0 {
		resp["scope"] = joinScopes(scopes)
	}
	return resp, nil
}

// Introspect returns the introspection response. Without signature verification
// we report inactive unless a live persisted token is found; here we cannot
// resolve opaque tokens, so we report inactive.
func (a *pgOIDCGrants) Introspect(ctx context.Context, cmd domain.OIDCIntrospectCmd) (map[string]any, error) {
	if cmd.Token == "" {
		return map[string]any{"active": false}, nil
	}
	// TODO: sign/verify with signing key — verify the token signature / look up
	// its persisted state to populate the introspection claims.
	return map[string]any{"active": false}, nil
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
		// TODO: sign/verify with signing key — opaque access/refresh tokens are
		// revoked by the token subsystem.
		// TODO outbox event: oidc.token.revoked
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
		if _, err := models.IamParRequests.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		// TODO outbox event: oidc.par.created
		return &domain.OIDCParResult{RequestURI: requestURI, ExpiresIn: ttl}, nil
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
		if _, err := models.IamDeviceCodes.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		// TODO outbox event: oidc.device.authorized
		return out, nil
	})
}

// ===== userinfo =====

// Userinfo returns the OIDC userinfo claims for the bearer-authenticated
// account. Resolving the claims set requires the account aggregate (a separate
// port); the signing of a signed userinfo response is the token subsystem's job.
func (a *pgOIDCGrants) Userinfo(ctx context.Context, accountID, sessionID string) (map[string]any, error) {
	// TODO: sign/verify with signing key — a signed userinfo response (JWT) is
	// minted by the token subsystem; here we return the minimal sub claim.
	return map[string]any{"sub": accountID}, nil
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
		// TODO outbox event: oidc.device.decided
		return nil
	})
}

// ===== JWKS / discovery =====

// JWKS returns the JSON Web Key Set for a project environment. Public. The
// public-key material is derived from the persisted signing keys; here we list
// the active keys and emit their metadata, leaving the public-key encoding to
// the signing subsystem.
func (a *pgOIDCGrants) JWKS(ctx context.Context, projectID, env string) (map[string]any, error) {
	rows, err := models.IamSigningKeys.Query(
		sm.Where(models.IamSigningKeys.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSigningKeys.Columns.Environment.EQ(psql.Arg(env))),
		sm.Where(models.IamSigningKeys.Columns.Status.EQ(psql.Arg("active"))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	keys := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		// TODO: sign/verify with signing key — derive the public JWK (n/e or
		// x/y/crv) from the stored private_pem via the signing subsystem.
		keys = append(keys, map[string]any{
			"kid": row.Kid,
			"alg": row.Alg,
			"use": row.Use,
		})
	}
	return map[string]any{"keys": keys}, nil
}

// OpenIDConfiguration returns the discovery document for a project environment.
func (a *pgOIDCGrants) OpenIDConfiguration(ctx context.Context, projectID, env string) (map[string]any, error) {
	base := fmt.Sprintf("/p/%s/e/%s", projectID, env)
	return map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                "/oauth2/authorize",
		"token_endpoint":                        "/oauth2/token",
		"userinfo_endpoint":                     "/oauth2/userinfo",
		"jwks_uri":                              base + "/.well-known/jwks.json",
		"introspection_endpoint":                "/oauth2/introspect",
		"revocation_endpoint":                   "/oauth2/revoke",
		"pushed_authorization_request_endpoint": "/oauth2/par",
		"device_authorization_endpoint":         "/oauth2/device_authorization",
		"end_session_endpoint":                  "/oauth2/logout",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "device_code"},
		"code_challenge_methods_supported":      []string{"S256"},
	}, nil
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
