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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	oidcAccessTTL = 10 * time.Minute
	// oidcRefreshTTL is the lifetime of an issued refresh token.
	oidcRefreshTTL = 30 * 24 * time.Hour
)

// JWT claim names minted and inspected across the core-auth, federation and
// OIDC token paths. Naming them once keeps the spelling identical everywhere a
// token is built or read.
const (
	claimIssuer      = "iss"
	claimSubject     = "sub"
	claimSessionID   = "sid"
	claimTokenID     = "jti"
	claimProjectID   = "pid"
	claimAudience    = "aud"
	claimAAL         = "aal"
	claimAMR         = "amr"
	claimTokenType   = "typ"
	claimEnvironment = "env"
	claimClientID    = "client_id"
	claimScope       = "scope"
	claimExpiresAt   = "exp"
	// claimGroups carries the user's IAM roles. It is emitted only when the
	// `groups` scope was granted, and its values come from the role assignments
	// in iam_user_roles — never from anything the client sent.
	claimGroups = "groups"
)

// oidcScopeGroups asks for the user's IAM roles in the `groups` claim of the
// access and id token. Relying parties (ArgoCD, Grafana, ...) map that claim
// onto their own permissions.
const oidcScopeGroups = "groups"

// oidcClaimsSupported is the claims_supported list of the discovery document:
// what a client can expect to find in a token from this provider.
func oidcClaimsSupported() []string {
	return []string{
		claimIssuer, claimSubject, claimAudience, claimExpiresAt, "iat", "auth_time",
		"nonce", "at_hash", "azp", claimSessionID, claimScope, claimClientID,
		claimTokenType, claimEnvironment, claimGroups,
	}
}

// Values of the `typ` claim. IAM mints three token kinds and each verify path
// checks the kind it expects, so a refresh token can never be presented as an
// access token.
const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

// oidcIssuer returns the canonical issuer for project/env: the deployment's
// absolute public base URL followed by the tenant path. OIDC Discovery 1.0 §3
// requires the issuer to be an absolute https URL that is a literal prefix of
// the URL the discovery document is served from — go-oidc and every other
// conforming client verify exactly that, so the base MUST be the configured
// service.http.public_url and never anything derived from a request header.
//
// base is expected normalized (absolute, no trailing slash); a caller that
// passes an empty base gets an empty issuer, which fails closed everywhere it
// is compared.
func oidcIssuer(base, projectID, env string) string {
	if base == "" {
		return ""
	}

	return fmt.Sprintf("%s/p/%s/e/%s", base, projectID, env)
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
	// sessionID ties the grant to the IAM browser session it was authorized
	// from. It is what makes an issued token killable: revoking the session
	// revokes its refresh tokens, and back-channel logout names it as `sid`.
	sessionID string
	// authTime is when the user actually authenticated, for the `auth_time`
	// claim and the max_age check.
	authTime time.Time
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
	// clientKeys resolves the public keys a client signs its assertions and
	// request objects with.
	clientKeys *clientKeyCache
	// cfg resolves the project's auth config (locales) for the hosted
	// login/consent pages. A nil cfg falls back to no declared locales.
	cfg *configReader
}

// NewPgOIDCGrants builds the OIDC-provider adapter over db.
func NewPgOIDCGrants(db *DB, emitter Emitter, cfg *configReader) *pgOIDCGrants {
	return &pgOIDCGrants{db: db, emitter: emitter, cfg: cfg, clientKeys: newClientKeyCache()}
}

var _ api.OIDCGrants = (*pgOIDCGrants)(nil)

// oidcInteractionEnvelope is the data-jsonb shape for an interaction: the
// public domain.Interaction fields plus the account bound at login time
// (iam_interactions has no account lookup column).
type oidcInteractionEnvelope struct {
	domain.Interaction
	AccountID string `json:"account_id,omitempty"`
	// AuthTime is when the session that claimed this interaction authenticated.
	// It becomes the id_token's `auth_time` and is what a max_age request is
	// measured against.
	AuthTime time.Time `json:"auth_time,omitempty"`
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

// oidcInteractionExpired reports whether a pending interaction has aged out. An
// expired interaction is treated as gone: the user has to restart the flow
// rather than resume one whose parameters may no longer reflect the client.
func oidcInteractionExpired(ctx context.Context, row *models.IamInteraction) bool {
	exp, ok := row.ExpiresAt.Get()
	return ok && !exp.IsZero() && exp.Before(nowIn(ctx))
}

// ResolveInteraction returns the pending interaction by id. No tenant filter is
// applied here because the interaction id is itself an unguessable handle, and
// the endpoint is public: the UI needs the requested scopes before the user has
// logged in. Whoever may ACT on the interaction is decided by the session
// binding in CompleteLogin/Consent, not here.
func (a *pgOIDCGrants) ResolveInteraction(
	ctx context.Context, interactionID string,
) (*domain.OIDCInteractionContext, error) {
	row, err := models.FindIamInteraction(ctx, a.db.Bobx(), interactionID)
	if err != nil {
		return nil, translatePgErr("interaction", err)
	}

	if oidcInteractionExpired(ctx, row) {
		return nil, domain.ErrFlowExpired
	}

	var env oidcInteractionEnvelope
	if err := unmarshal(row.Data, &env); err != nil {
		return nil, err
	}

	in := env.Interaction

	// An interaction nobody has claimed still needs a login; one that carries a
	// bound session and account only needs the decision.
	stage := domain.OIDCStageLogin
	if row.SessionID.GetOrZero() != "" && env.AccountID != "" {
		stage = domain.OIDCStageConsent
	}

	out := &domain.OIDCInteractionContext{
		Interaction: in,
		ProjectID:   row.ProjectID,
		Environment: row.Environment,
		Stage:       stage,
	}
	if exp, ok := row.ExpiresAt.Get(); ok {
		out.ExpiresAt = exp
	}
	// The application's own name: a consent screen that cannot say WHICH app is
	// asking for these scopes is not a consent screen.
	if in.ClientID != "" {
		if clientRow, err := models.FindIamAppClient(ctx, a.db.Bobx(), in.ClientID); err == nil {
			out.ClientName = clientRow.Name
			out.ClientType = clientRow.Type
		}
	}

	if a.cfg != nil && row.ProjectID != "" {
		authCfg, err := a.cfg.AuthConfig(ctx, row.ProjectID)
		if err != nil {
			return nil, err
		}

		out.DefaultLocale = authCfg.DefaultLocale
		out.SupportedLocales = authCfg.SupportedLocales
	}

	return out, nil
}

// CompleteLogin binds an authenticated account and session to the interaction.
//
// This is where an interaction acquires an owner. It is created unbound by the
// public authorization endpoint, so the FIRST authenticated caller claims it and
// every later step must present the same session. Without that claim the
// interaction id — which travels through the user-agent, and therefore through
// history, logs and referrers — would be the only thing needed to drive somebody
// else's authorization to completion and hand their client a code bound to the
// attacker's account.
//
// A caller with no session (an admin token, an API key, a client credential)
// cannot own an interactive flow at all: its empty session id used to match the
// unbound interaction's NULL, which made the check vacuous.
func (a *pgOIDCGrants) CompleteLogin(ctx context.Context, interactionID, accountID, sessionID string) error {
	if sessionID == "" || accountID == "" {
		return domain.ErrForbidden
	}

	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamInteraction(ctx, a.db.Bobx(), interactionID)
		if err != nil {
			return translatePgErr("interaction", err)
		}

		if oidcInteractionExpired(ctx, row) {
			return domain.ErrFlowExpired
		}

		if bound := row.SessionID.GetOrZero(); bound != "" && bound != sessionID {
			return domain.ErrForbidden
		}
		// Persist the resolved account into the interaction envelope alongside the
		// domain.Interaction fields (iam_interactions has no account column).
		var env oidcInteractionEnvelope
		if err := unmarshal(row.Data, &env); err != nil {
			return err
		}

		env.AccountID = accountID
		env.SessionID = sessionID
		env.AuthTime = nowUTC()

		raw, err := marshal(&env)
		if err != nil {
			return err
		}

		rm := json.RawMessage(raw)

		boundSession := null.From(sessionID)

		setter := &models.IamInteractionSetter{Data: &rm, SessionID: &boundSession}
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
//
// Consent is only meaningful from the person who logged in: the interaction must
// already be claimed (CompleteLogin), the caller must present the session that
// claimed it, and the consenting account must be the one that was bound. An
// unclaimed interaction is refused outright — otherwise anyone holding the
// interaction id could mint a code for their own account and hand the client a
// session that is not the user's.
// Consent turns an approved interaction into an authorization response. Like
// Authorize, the response mode decides only how the parameters travel: in
// form_post mode the hosted UI is handed a form to submit rather than a URL to
// follow.
func (a *pgOIDCGrants) Consent(
	ctx context.Context, cmd domain.OIDCConsentCmd,
) (*domain.OIDCAuthorizeResult, error) {
	if cmd.SessionID == "" || cmd.AccountID == "" {
		return nil, domain.ErrForbidden
	}

	mode := ""

	target, err := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		row, env, err := a.claimedInteraction(ctx, cmd)
		if err != nil {
			return "", err
		}

		in := env.Interaction

		if cmd.Remember && in.ClientID != "" {
			if err := a.rememberGrant(ctx, row.ProjectID, in.ClientID, cmd); err != nil {
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

		mode = in.ResponseMode

		return a.consentedCode(ctx, cmd, row, env)
	})
	if err != nil {
		return nil, err
	}

	return oidcAuthorizeResult(target, mode), nil
}

// consentedCode mints the authorization code a consent decided on and renders
// the response that carries it. The code is opaque and sha256-hashed at rest,
// so it stays revocable; the token endpoint resolves the principal from the
// row's columns and the scopes from its envelope. Assumes an ambient
// transaction.
func (a *pgOIDCGrants) consentedCode(
	ctx context.Context,
	cmd domain.OIDCConsentCmd,
	row *models.IamInteraction,
	env oidcInteractionEnvelope,
) (string, error) {
	in := env.Interaction

	scopes := cmd.GrantedScopes
	if len(scopes) == 0 {
		scopes = in.Scopes
	}

	code, authCodeRow, err := a.issueAuthorizationCode(ctx, authCodeRequest{
		projectID:   row.ProjectID,
		environment: row.Environment,
		accountID:   cmd.AccountID,
		sessionID:   cmd.SessionID,
		authTime:    env.AuthTime,
		interaction: in,
		scopes:      scopes,
	})
	if err != nil {
		return "", err
	}

	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        eventTokenIssued,
		ProjectID:   row.ProjectID,
		Environment: "",
		AggregateID: authCodeRow.ID,
		Payload: map[string]any{
			payloadGrantType: string(oidc.GrantTypeCode),
			claimClientID:    in.ClientID,
			payloadAccountID: cmd.AccountID,
			payloadScopes:    scopes,
		},
	}); err != nil {
		return "", err
	}

	return oidcAuthorizationResponse(in.RedirectURI, code, in.State, in.ResponseMode,
		oidcIssuer(a.db.PublicURL, row.ProjectID, row.Environment)), nil
}

// claimedInteraction loads the interaction a consent refers to and checks that
// the caller is the one who claimed it. Both the session and the account have to
// match: the interaction handle travels in a URL, so possession of it is not on
// its own permission to decide the request.
func (a *pgOIDCGrants) claimedInteraction(
	ctx context.Context, cmd domain.OIDCConsentCmd,
) (*models.IamInteraction, oidcInteractionEnvelope, error) {
	var env oidcInteractionEnvelope

	row, err := models.FindIamInteraction(ctx, a.db.Bobx(), cmd.InteractionID)
	if err != nil {
		return nil, env, translatePgErr("interaction", err)
	}

	if oidcInteractionExpired(ctx, row) {
		return nil, env, domain.ErrFlowExpired
	}

	if row.SessionID.GetOrZero() != cmd.SessionID {
		return nil, env, domain.ErrForbidden
	}

	if err := unmarshal(row.Data, &env); err != nil {
		return nil, env, err
	}

	if env.AccountID == "" || env.AccountID != cmd.AccountID {
		return nil, env, domain.ErrForbidden
	}

	return row, env, nil
}

// rememberGrant records a consent the user asked to be remembered, so the next
// authorization request from the same client is answered without a screen.
// Assumes an ambient transaction.
func (a *pgOIDCGrants) rememberGrant(
	ctx context.Context, projectID, clientID string, cmd domain.OIDCConsentCmd,
) error {
	grant := domain.Grant{
		ID:        newUUID(),
		AccountID: cmd.AccountID,
		ClientID:  clientID,
		Scopes:    cmd.GrantedScopes,
		GrantedAt: nowUTC(),
	}
	if err := a.persistGrant(ctx, projectID, &grant); err != nil {
		return err
	}

	return a.emitter.Emit(ctx, domain.Event{
		Type:        "oidc.grant.created",
		ProjectID:   projectID,
		Environment: "",
		AggregateID: grant.ID,
		Payload:     &grant,
	})
}

// authCodeRequest is everything one authorization code is minted from.
type authCodeRequest struct {
	projectID   string
	environment string
	accountID   string
	sessionID   string
	authTime    time.Time
	interaction domain.Interaction
	scopes      []string
}

// oidcAuthCodeBytes is the entropy of an authorization code.
const oidcAuthCodeBytes = 32

// oidcAuthCodeTTL bounds an authorization code. It is redeemed immediately by
// the client, so it stays short (RFC 6749 §4.1.2 recommends under ten minutes).
const oidcAuthCodeTTL = 10 * time.Minute

// issueAuthorizationCode mints a one-time authorization code for a decided
// request. Both paths that can decide one — the consent screen, and a silent
// request satisfied by an existing session and grant — go through here, so a
// code minted without UI carries exactly the same bindings (PKCE, redirect_uri,
// nonce, session, auth time) as one minted with it. Assumes an ambient
// transaction.
func (a *pgOIDCGrants) issueAuthorizationCode(
	ctx context.Context, req authCodeRequest,
) (string, *models.IamAuthCode, error) {
	code, err := oidcRandToken(oidcAuthCodeBytes)
	if err != nil {
		return "", nil, err
	}

	raw, err := marshal(authCodeData{
		Scopes:              req.scopes,
		RedirectURI:         req.interaction.RedirectURI,
		Nonce:               req.interaction.Nonce,
		CodeChallenge:       req.interaction.CodeChallenge,
		CodeChallengeMethod: req.interaction.CodeChallengeMethod,
		SessionID:           req.sessionID,
		AuthTime:            req.authTime,
	})
	if err != nil {
		return "", nil, err
	}

	data := json.RawMessage(raw)
	uid := null.From(req.accountID)
	cid := null.From(req.interaction.ClientID)

	env := req.environment
	if env == "" {
		if env, err = effectiveEnv(ctx, a.db, req.projectID); err != nil {
			return "", nil, err
		}
	}

	row, err := models.IamAuthCodes.Insert(&models.IamAuthCodeSetter{
		ID:          ptr(newUUID()),
		ProjectID:   &req.projectID,
		Environment: &env,
		CodeHash:    ptr(oidcHashToken(code)),
		ClientID:    &cid,
		UserID:      &uid,
		ExpiresAt:   ptr(nowUTC().Add(oidcAuthCodeTTL)),
		Data:        &data,
	}).One(ctx, a.db.Bobx())
	if err != nil {
		if isUniqueViolation(err) {
			return "", nil, domain.ErrConflict
		}

		return "", nil, fmt.Errorf("insert authorization code: %w", err)
	}

	return code, row, nil
}

// oidcAuthorizationResponse renders the success response in the requested mode.
// `fragment` keeps the code out of the Referer header and out of server logs at
// the redirect target, which is why clients ask for it.
func oidcAuthorizationResponse(redirectURI, code, state, mode, issuer string) string {
	params := url.Values{}
	params.Set("code", code)

	if state != "" {
		params.Set("state", state)
	}
	// RFC 9207: the response says which provider issued it. A client registered
	// with more than one OP can otherwise be tricked into redeeming a code at the
	// wrong one — the code was honest, the destination was not.
	if issuer != "" {
		params.Set("iss", issuer)
	}

	if mode == oidcResponseModeFragment {
		return redirectURI + "#" + params.Encode()
	}
	// form_post is split back apart by oidcFormPostOf, which reads the query —
	// so build one here and let the caller move it into a body.

	sep := "?"
	if strings.Contains(redirectURI, "?") {
		sep = "&"
	}

	return redirectURI + sep + params.Encode()
}

// Reject cancels the interaction and returns the redirect carrying the OAuth2
// error back to the client. Public operation (no session binding).
func (a *pgOIDCGrants) Reject(
	ctx context.Context, cmd domain.OIDCRejectCmd,
) (*domain.OIDCAuthorizeResult, error) {
	mode := ""

	target, err := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		row, err := models.FindIamInteraction(ctx, a.db.Bobx(), cmd.InteractionID)
		if err != nil {
			return "", translatePgErr("interaction", err)
		}

		if oidcInteractionExpired(ctx, row) {
			return "", domain.ErrFlowExpired
		}
		// Canceling is public, because the user may want to back out before they
		// have signed in and there is no session to check then. Once someone HAS
		// claimed the interaction, only they may cancel it — otherwise anyone who
		// learned the id could end a stranger's sign-in.
		if bound := row.SessionID.GetOrZero(); bound != "" {
			principal, ok := api.SoftPrincipalFrom(ctx)
			if !ok || principal.SessionID != bound {
				return "", domain.ErrForbidden
			}
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

		mode = in.ResponseMode

		return oidcErrorRedirect(in.RedirectURI, errCode, cmd.ErrorDescription, in.State,
			oidcIssuer(a.db.PublicURL, row.ProjectID, row.Environment)), nil
	})
	if err != nil {
		return nil, err
	}

	return oidcAuthorizeResult(target, mode), nil
}

// ===== grants =====

// persistGrant upserts a remembered grant for (project, user, client). Helper
// shared by Consent; assumes an ambient transaction.
func (a *pgOIDCGrants) persistGrant(ctx context.Context, projectID string, g *domain.Grant) error {
	env, err := effectiveEnv(ctx, a.db, projectID)
	if err != nil {
		return err
	}

	raw, err := marshal(g)
	if err != nil {
		return err
	}

	rm := json.RawMessage(raw)

	setter := &models.IamOauthGrantSetter{
		ID:          &g.ID,
		ProjectID:   &projectID,
		Environment: &env,
		UserID:      &g.AccountID,
		ClientID:    &g.ClientID,
		GrantedAt:   ptr(g.GrantedAt),
		Data:        &rm,
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

// oidcResponseTypeCode is the only response_type IAM supports (authorization
// code flow); the discovery document advertises exactly this one.
const oidcResponseTypeCode = "code"

// oidcAuthorizeTTL is how long an authorization interaction stays resolvable
// before the user has to restart the flow.
const oidcAuthorizeTTL = 10 * time.Minute

// resolveAuthorizeClient loads the app client behind client_id. An unknown or
// disabled client is ErrInvalidClient: RFC 6749 §4.1.2.1 forbids redirecting the
// user-agent in that case, because nothing about the request has been proven to
// belong to a registered client yet.
func (a *pgOIDCGrants) resolveAuthorizeClient(
	ctx context.Context, clientID string,
) (*models.IamAppClient, *domain.AppClient, error) {
	if clientID == "" {
		return nil, nil, domain.ErrInvalidClient
	}

	row, err := models.FindIamAppClient(ctx, a.db.Bobx(), clientID)
	if err != nil {
		if isStorageNotFound(translatePgErr("app_client", err)) {
			return nil, nil, domain.ErrInvalidClient
		}

		return nil, nil, err
	}

	var app domain.AppClient
	if err := unmarshal(row.Data, &app); err != nil {
		return nil, nil, err
	}

	if app.Disabled {
		return nil, nil, domain.ErrInvalidClient
	}

	return row, &app, nil
}

// authorizeRedirectURI returns the registered redirect_uri the request asked
// for. The comparison is exact against the client's registration — no prefix or
// substring matching, which is how open redirectors are built (RFC 9700 §2.1).
// A client with exactly one registered URI may omit the parameter.
func authorizeRedirectURI(app *domain.AppClient, requested string) (string, error) {
	if requested == "" {
		if len(app.RedirectURIs) == 1 {
			return app.RedirectURIs[0], nil
		}

		return "", domain.ErrInvalidRedirectURI
	}

	for _, registered := range app.RedirectURIs {
		if registered == requested {
			return requested, nil
		}
	}

	return "", domain.ErrInvalidRedirectURI
}

// PKCE (RFC 7636). S256 is the only method IAM implements, and the only one the
// discovery document advertises: `plain` offers no protection against an
// intercepted authorization code, which is the whole point of the exchange.
// OAuth2 error codes returned on the authorization redirect (RFC 6749 §4.1.2.1).
const (
	oidcErrInvalidRequest          = "invalid_request"
	oidcErrUnsupportedResponseType = "unsupported_response_type"
)

const (
	oidcCodeChallengeMethodS256 = "S256"
	// Length bounds of a code_verifier, straight from RFC 7636 §4.1. The S256
	// challenge is BASE64URL of a 32-byte digest, so it is always 43 characters;
	// the same bounds are applied to it for a cheap early reject.
	oidcCodeVerifierMinLen = 43
	oidcCodeVerifierMaxLen = 128
)

// oidcUnreserved reports whether r is in the RFC 7636 `unreserved` set that both
// the code_verifier and the code_challenge are drawn from.
func oidcUnreserved(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-', r == '.', r == '_', r == '~':
		return true
	default:
		return false
	}
}

// oidcValidPKCEValue reports whether s is a syntactically valid code_verifier or
// code_challenge: the RFC 7636 length bounds and character set.
func oidcValidPKCEValue(s string) bool {
	if len(s) < oidcCodeVerifierMinLen || len(s) > oidcCodeVerifierMaxLen {
		return false
	}

	for _, r := range s {
		if !oidcUnreserved(r) {
			return false
		}
	}

	return true
}

// oidcCodeChallengeFor returns BASE64URL-ENCODE(SHA256(verifier)) — the S256
// transformation of RFC 7636 §4.6, unpadded as the spec requires.
func oidcCodeChallengeFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// oidcCheckPKCERequest validates the PKCE parameters of an authorization
// request. It returns an OAuth2 error code and description when the request must
// be bounced back to the client, or empty strings when it is acceptable.
//
// PKCE is REQUIRED for public clients (spa / native): they hold no secret, so
// the authorization code is the only thing standing between an attacker who
// intercepts the redirect and a token. Confidential clients authenticate with a
// secret at the token endpoint and may omit it — but if they send a challenge it
// is enforced exactly the same way.
func oidcCheckPKCERequest(app *domain.AppClient, challenge, method string) (string, string) {
	if challenge == "" {
		if method != "" {
			return oidcErrInvalidRequest, "code_challenge_method was sent without a code_challenge"
		}

		if !oidcIsConfidentialClient(app.Type) {
			return oidcErrInvalidRequest, "code_challenge is required for public clients (PKCE, RFC 7636)"
		}

		return "", ""
	}

	if method != oidcCodeChallengeMethodS256 {
		return oidcErrInvalidRequest, "code_challenge_method must be S256"
	}

	if !oidcValidPKCEValue(challenge) {
		return oidcErrInvalidRequest, "code_challenge is not a valid RFC 7636 challenge"
	}

	return "", ""
}

// oidcVerifyPKCE checks a presented code_verifier against the challenge bound to
// the authorization code at authorize time.
//
// Both directions matter. A code issued WITH a challenge cannot be redeemed
// without the matching verifier — that is the attack PKCE exists to stop. A code
// issued WITHOUT one cannot be redeemed with a verifier either: accepting that
// would let an attacker who captured a code strip the challenge and downgrade
// the exchange (RFC 7636 §4.6).
func oidcVerifyPKCE(challenge, method, verifier string) error {
	if challenge == "" {
		if verifier != "" {
			return domain.ErrInvalidGrant.WithMessage(
				"code_verifier was presented for a code issued without a code_challenge")
		}

		return nil
	}

	if verifier == "" {
		return domain.ErrInvalidGrant.WithMessage("code_verifier is required for this authorization code")
	}

	if method != oidcCodeChallengeMethodS256 {
		// Only S256 is ever stored; anything else means a corrupted or forged code.
		return domain.ErrInvalidGrant.WithMessage("unsupported code_challenge_method on the authorization code")
	}

	if !oidcValidPKCEValue(verifier) {
		return domain.ErrInvalidGrant.WithMessage("code_verifier is not a valid RFC 7636 verifier")
	}

	if subtle.ConstantTimeCompare([]byte(oidcCodeChallengeFor(verifier)), []byte(challenge)) != 1 {
		return domain.ErrInvalidGrant.WithMessage("code_verifier does not match the code_challenge")
	}

	return nil
}

// oidcErrorRedirect builds the OAuth2 error redirect back to an already
// validated redirect_uri (RFC 6749 §4.1.2.1). state is echoed verbatim when the
// client sent one, so the client can match the response to its request.
func oidcErrorRedirect(redirectURI, code, description, state, issuer string) string {
	params := url.Values{}
	params.Set("error", code)

	if description != "" {
		params.Set("error_description", description)
	}

	if state != "" {
		params.Set("state", state)
	}
	// RFC 9207: name the issuer on error responses too. A client talking to
	// several providers otherwise cannot tell which one refused it, which is half
	// of the mix-up attack the parameter exists to stop.
	if issuer != "" {
		params.Set("iss", issuer)
	}

	sep := "?"
	if strings.Contains(redirectURI, "?") {
		sep = "&"
	}

	return redirectURI + sep + params.Encode()
}

// Prompt values (OIDC Core §3.1.2.1).
const (
	oidcPromptNone          = "none"
	oidcPromptLogin         = "login"
	oidcPromptConsent       = "consent"
	oidcPromptSelectAccount = "select_account"
)

// Event type and payload keys shared by the paths that issue tokens. They are
// part of the webhook contract, so they are named once rather than retyped.
const (
	// #nosec G101 -- an event type name, not a credential.
	eventTokenIssued = "oidc.token.issued"
	payloadGrantType = "grant_type"
	payloadAccountID = "account_id"
	payloadScopes    = "scopes"
)

// oidcResponseModeQuery / oidcResponseModeFragment are the response modes the
// discovery document advertises.
const (
	oidcResponseModeQuery    = "query"
	oidcResponseModeFragment = "fragment"
	oidcResponseModeFormPost = "form_post"
)

// oidcResponseModeSupported reports whether a requested response mode is one we
// can actually deliver. An empty value means the default, `query`.
func oidcResponseModeSupported(mode string) bool {
	switch mode {
	case "", oidcResponseModeQuery, oidcResponseModeFragment, oidcResponseModeFormPost:
		return true
	default:
		return false
	}
}

// authorizeIntent is what the authorization request asks us to do about the
// user's existing session, once prompt and max_age have been read.
type authorizeIntent struct {
	// silent forbids showing any UI: the request either succeeds against an
	// existing session and grant, or comes back as an error (prompt=none).
	silent bool
	// forceLogin re-authenticates even when a valid session exists.
	forceLogin bool
	// forceConsent shows the consent screen even when the scopes were granted
	// before.
	forceConsent bool
}

// parseAuthorizePrompt reads the prompt parameter. `none` is exclusive: asking
// for no UI and for a login screen in the same breath is a contradiction the
// spec makes an error rather than guessing at.
func parseAuthorizePrompt(prompt string) (authorizeIntent, string) {
	var intent authorizeIntent

	values := strings.Fields(prompt)
	for _, v := range values {
		switch v {
		case oidcPromptNone:
			intent.silent = true
		case oidcPromptLogin, oidcPromptSelectAccount:
			intent.forceLogin = true
		case oidcPromptConsent:
			intent.forceConsent = true
		default:
			return intent, "prompt value " + v + " is not supported"
		}
	}

	if intent.silent && len(values) > 1 {
		return intent, "prompt=none cannot be combined with other prompt values"
	}

	return intent, ""
}

// authorizeScopesAllowed checks the request against the client's scope
// allow-list. An empty list means the project has not restricted the client, and
// a scope outside a non-empty list is refused rather than trimmed: silently
// issuing a narrower token than asked for produces failures far from the cause.
func authorizeScopesAllowed(app *domain.AppClient, scopes []string) (string, bool) {
	if len(app.Scopes) == 0 {
		return "", true
	}

	for _, scope := range scopes {
		if !containsString(app.Scopes, scope) {
			return scope, false
		}
	}

	return "", true
}

// oidcSessionAuthTime returns when a live session authenticated, and whether it
// is still live at all.
func (a *pgOIDCGrants) oidcSessionAuthTime(ctx context.Context, projectID, sessionID string) (time.Time, bool) {
	if sessionID == "" {
		return time.Time{}, false
	}

	row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
	if err != nil || row.ProjectID != projectID {
		return time.Time{}, false
	}

	if exp, ok := row.ExpiresAt.Get(); ok && !exp.IsZero() && exp.Before(nowIn(ctx)) {
		return time.Time{}, false
	}

	return row.CreatedAt, true
}

// authorizeSilently answers an authorization request from what we already know:
// a live IAM session, and a remembered grant covering the scopes asked for. It
// reports whether it decided the request.
//
// This is what makes it single sign-on rather than repeated sign-on. It is also
// where max_age is enforced: a session older than the client is willing to rely
// on is not good enough, however valid it still is.
func (a *pgOIDCGrants) authorizeSilently(
	ctx context.Context,
	cmd domain.OIDCAuthorizeCmd,
	app *domain.AppClient,
	clientRow *models.IamAppClient,
	redirectURI string,
	scopes []string,
) (string, bool, error) {
	principal, ok := api.SoftPrincipalFrom(ctx)
	if !ok || principal.AccountID == "" || principal.SessionID == "" {
		return "", false, nil
	}

	if principal.ProjectID != clientRow.ProjectID {
		return "", false, nil // a session in another tenant is not this one's
	}

	authTime, live := a.oidcSessionAuthTime(ctx, clientRow.ProjectID, principal.SessionID)
	if !live {
		return "", false, nil
	}

	if cmd.MaxAge > 0 && nowUTC().Sub(authTime) > time.Duration(cmd.MaxAge)*time.Second {
		return "", false, nil // too old to rely on; re-authenticate
	}

	if consentErr := a.oidcVerifyConsent(ctx, clientRow.ProjectID, principal.AccountID, cmd.ClientID); consentErr != nil {
		//nolint:nilerr // No remembered grant is not a failure: it means the user
		// has to decide, which is the interaction the caller falls through to.
		return "", false, nil
	}

	interaction := domain.Interaction{
		ClientID:            cmd.ClientID,
		Scopes:              scopes,
		RedirectURI:         redirectURI,
		Nonce:               cmd.Nonce,
		CodeChallenge:       cmd.CodeChallenge,
		CodeChallengeMethod: cmd.CodeChallengeMethod,
		State:               cmd.State,
	}

	code, err := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		issued, row, ierr := a.issueAuthorizationCode(ctx, authCodeRequest{
			projectID:   clientRow.ProjectID,
			environment: clientRow.Environment,
			accountID:   principal.AccountID,
			sessionID:   principal.SessionID,
			authTime:    authTime,
			interaction: interaction,
			scopes:      scopes,
		})
		if ierr != nil {
			return "", ierr
		}

		return issued, a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.token.issued",
			ProjectID:   clientRow.ProjectID,
			Environment: clientRow.Environment,
			AggregateID: row.ID,
			Payload: map[string]any{
				"grant_type": "authorization_code",
				"client_id":  cmd.ClientID,
				"account_id": principal.AccountID,
				"scopes":     scopes,
				"silent":     true,
			},
		})
	})
	if err != nil {
		return "", false, err
	}

	return oidcAuthorizationResponse(redirectURI, code, cmd.State, cmd.ResponseMode,
		oidcIssuer(a.db.PublicURL, clientRow.ProjectID, clientRow.Environment)), true, nil
}

// silentFailure names what a prompt=none request was missing: somewhere to get a
// user from, or a decision the user has not made. Telling the client which lets
// it choose between a redirect and a prompt.
func (a *pgOIDCGrants) silentFailure(ctx context.Context, cmd domain.OIDCAuthorizeCmd) string {
	principal, ok := api.SoftPrincipalFrom(ctx)
	if !ok || principal.AccountID == "" {
		return "login_required"
	}

	if cmd.MaxAge > 0 {
		if authTime, live := a.oidcSessionAuthTime(ctx, principal.ProjectID, principal.SessionID); live &&
			nowUTC().Sub(authTime) > time.Duration(cmd.MaxAge)*time.Second {
			return "login_required"
		}
	}

	return "consent_required"
}

// Authorize builds a front-channel interaction for the request and returns the
// redirect to the login/consent UI. Public operation.
//
// Validation order follows RFC 6749 §4.1.2.1, and it is the order that matters:
// the client and its redirect_uri are resolved FIRST and any failure there is
// reported as a 400 without a redirect, because an unverified redirect_uri must
// never receive traffic. Only once both check out does a bad request parameter
// become a redirect carrying `error` and the caller's `state`. Nothing is
// persisted until the client is known, so an unknown client_id can no longer
// mint interaction rows.
// Authorize answers an authorization request. It resolves the request down to a
// single URL and then, in form_post mode, turns the response destined for the
// client into a form instead. The split is deliberate: every path below builds
// the same parameters either way, so the response mode changes only how they
// travel, never what they say.
func (a *pgOIDCGrants) Authorize(
	ctx context.Context, cmd domain.OIDCAuthorizeCmd,
) (*domain.OIDCAuthorizeResult, error) {
	// Resolve first: a pushed request (RFC 9126) or a signed request object
	// (RFC 9101) replaces the query parameters, and the response mode is one of
	// the things they replace — so the mode that decides how the response
	// travels has to be read after that, not before.
	resolved, err := a.resolveAuthorizeRequest(ctx, cmd)
	if err != nil {
		return nil, err
	}

	target, err := a.authorize(ctx, resolved)
	if err != nil {
		return nil, err
	}

	return oidcAuthorizeResult(target, resolved.ResponseMode), nil
}

// resolveAuthorizeRequest applies the two mechanisms that can supply an
// authorization request from somewhere other than the query string.
//
// A pushed request supplies the whole request: only client_id is read from the
// query alongside it, so a query parameter cannot override what the client
// lodged over an authenticated back channel. A signed request object replaces
// the parameters it carries, so nothing in the browser's URL can alter what the
// client asked for.
func (a *pgOIDCGrants) resolveAuthorizeRequest(
	ctx context.Context, cmd domain.OIDCAuthorizeCmd,
) (domain.OIDCAuthorizeCmd, error) {
	if cmd.Request != "" {
		resolved, err := a.consumeRequestObject(ctx, cmd)
		if err != nil {
			return cmd, err
		}

		cmd = resolved
	}

	if cmd.RequestURI != "" {
		pushed, err := a.consumePushedRequest(ctx, cmd.RequestURI, cmd.ClientID)
		if err != nil {
			return cmd, err
		}

		cmd = pushed
	}

	return cmd, nil
}

// oidcAuthorizeResult converts a resolved target into the result the API layer
// renders. Only an absolute URL is a response to the client; the interaction
// redirect is relative and stays a redirect whatever the client asked for.
func oidcAuthorizeResult(target, mode string) *domain.OIDCAuthorizeResult {
	if mode != oidcResponseModeFormPost || !oidcIsAbsoluteURL(target) {
		return &domain.OIDCAuthorizeResult{RedirectTo: target}
	}

	post, ok := oidcFormPostOf(target)
	if !ok {
		return &domain.OIDCAuthorizeResult{RedirectTo: target}
	}

	return &domain.OIDCAuthorizeResult{FormPost: post}
}

// oidcIsAbsoluteURL distinguishes a client redirect_uri from the interaction
// path we serve ourselves.
func oidcIsAbsoluteURL(raw string) bool {
	return strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://")
}

// oidcFormPostOf splits a response URL into the form the browser posts. The URL
// was built in query mode, so its parameters are exactly the response
// parameters — the form carries the same set, just in a body.
func oidcFormPostOf(raw string) (*domain.OIDCFormPost, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}

	fields := make(map[string]string, len(parsed.Query()))
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			fields[key] = values[0]
		}
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""

	return &domain.OIDCFormPost{Action: parsed.String(), Fields: fields}, true
}

// authorizeRequestError checks everything about a request that can be decided
// from the request itself, once the client and its redirect_uri are known. It
// returns the OAuth2 error code and description to send back, or empty strings
// when the request is well-formed.
func authorizeRequestError(
	app *domain.AppClient, cmd domain.OIDCAuthorizeCmd, scopes []string,
) (string, string) {
	if cmd.ResponseType != oidcResponseTypeCode {
		return oidcErrUnsupportedResponseType, "only response_type=code is supported"
	}

	if len(scopes) == 0 {
		return oidcErrInvalidRequest, "scope is required"
	}

	if errCode, desc := oidcCheckPKCERequest(app, cmd.CodeChallenge, cmd.CodeChallengeMethod); errCode != "" {
		return errCode, desc
	}

	if bad, ok := authorizeScopesAllowed(app, scopes); !ok {
		return "invalid_scope", "the client is not allowed the scope " + bad
	}

	if !oidcResponseModeSupported(cmd.ResponseMode) {
		return "unsupported_response_mode",
			"only query, fragment and form_post response modes are supported"
	}

	return "", ""
}

// authorize resolves a request whose parameters are already settled down to a
// single URL: either the interaction UI, or the client's redirect_uri carrying
// the response.
func (a *pgOIDCGrants) authorize(ctx context.Context, cmd domain.OIDCAuthorizeCmd) (string, error) {
	clientRow, app, err := a.resolveAuthorizeClient(ctx, cmd.ClientID)
	if err != nil {
		return "", err
	}

	redirectURI, err := authorizeRedirectURI(app, cmd.RedirectURI)
	if err != nil {
		return "", err
	}

	// From here the redirect target is a URI this client registered, so request
	// errors travel back to the client instead of being shown to the user.
	issuer := oidcIssuer(a.db.PublicURL, clientRow.ProjectID, clientRow.Environment)

	scopes := splitScopes(cmd.Scope)
	if errCode, desc := authorizeRequestError(app, cmd, scopes); errCode != "" {
		return oidcErrorRedirect(redirectURI, errCode, desc, cmd.State, issuer), nil
	}

	intent, promptErr := parseAuthorizePrompt(cmd.Prompt)
	if promptErr != "" {
		return oidcErrorRedirect(redirectURI, oidcErrInvalidRequest, promptErr, cmd.State, issuer), nil
	}

	// Single sign-on happens here: a caller who already holds a valid IAM session
	// and has granted these scopes before is answered with a code, not with a
	// login screen. It is also the only way prompt=none can succeed, since that
	// forbids showing any UI at all.
	if !intent.forceLogin && !intent.forceConsent {
		redirect, decided, err := a.authorizeSilently(ctx, cmd, app, clientRow, redirectURI, scopes)
		if err != nil {
			return "", err
		}

		if decided {
			return redirect, nil
		}
	}

	if intent.silent {
		// No UI allowed, and the request could not be satisfied from what we
		// already know. Say which of the two things is missing.
		return oidcErrorRedirect(redirectURI, a.silentFailure(ctx, cmd), "", cmd.State, issuer), nil
	}

	return a.startInteraction(ctx, cmd, clientRow, redirectURI, scopes)
}

// startInteraction persists the pending authorization request and returns the
// path the user-agent is sent to. Everything the token endpoint later checks —
// PKCE, redirect_uri, nonce, state — is bound here, so the browser cannot alter
// it between the screen and the code.
func (a *pgOIDCGrants) startInteraction(
	ctx context.Context,
	cmd domain.OIDCAuthorizeCmd,
	clientRow *models.IamAppClient,
	redirectURI string,
	scopes []string,
) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		in := domain.Interaction{
			ID:          newUUID(),
			ClientID:    cmd.ClientID,
			Scopes:      scopes,
			RedirectURI: redirectURI,
			Nonce:       cmd.Nonce,
			// Bind PKCE to the interaction; Consent copies it onto the code it
			// mints, and the token endpoint checks the verifier against it.
			CodeChallenge:       cmd.CodeChallenge,
			CodeChallengeMethod: cmd.CodeChallengeMethod,
			State:               cmd.State,
			Prompt:              cmd.Prompt,
			ResponseMode:        cmd.ResponseMode,
		}

		raw, err := marshal(&in)
		if err != nil {
			return "", err
		}

		rm := json.RawMessage(raw)
		cid := null.From(cmd.ClientID)
		exp := null.From(nowUTC().Add(oidcAuthorizeTTL))

		setter := &models.IamInteractionSetter{
			ID:          &in.ID,
			ProjectID:   ptr(clientRow.ProjectID),
			Environment: ptr(clientRow.Environment),
			ClientID:    &cid,
			ExpiresAt:   &exp,
			Data:        &rm,
		}
		if _, err := models.IamInteractions.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return "", err
		}

		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.interaction.created",
			ProjectID:   clientRow.ProjectID,
			Environment: clientRow.Environment,
			AggregateID: in.ID,
			Payload:     &in,
		}); err != nil {
			return "", err
		}

		return "/oauth/interaction/" + in.ID, nil
	})
}

// Logout terminates an RP-initiated logout. Validating the id_token_hint
// signature is the token subsystem's job; we return the post-logout redirect.
func (a *pgOIDCGrants) Logout(ctx context.Context, cmd domain.OIDCLogoutCmd) (*domain.OIDCLogoutResult, error) {
	out := &domain.OIDCLogoutResult{RedirectURL: "/"}

	if cmd.IDTokenHint == "" {
		// Without a hint we know neither whose session to end nor which client
		// registered the redirect the caller is asking for. OpenID Connect
		// RP-Initiated Logout 1.0 §2 says the URI SHOULD NOT be honored then, so
		// it is ignored rather than followed.
		return out, nil
	}

	peek := a.db.Signer().UnverifiedClaims(cmd.IDTokenHint)
	if peek == nil {
		return nil, domain.ErrInvalidToken
	}

	projectID, env := oidcParseIssuer(a.db.PublicURL, peekString(peek, claimIssuer))
	if projectID == "" {
		return nil, domain.ErrInvalidToken
	}

	claims, err := a.db.Signer().Verify(ctx, projectID, env, cmd.IDTokenHint)
	if err != nil {
		return nil, err
	}

	sub := peekString(claims, claimSubject)
	sid := peekString(claims, claimSessionID)
	clientID := peekAudience(claims)

	// End the session for real. Revoking it invalidates its refresh tokens
	// through the shared primitive, and emits session.revoked — which is what
	// drives back-channel logout to the other clients holding a grant on it.
	terminated, err := a.terminateSession(ctx, projectID, sid)
	if err != nil {
		return nil, err
	}

	if terminated {
		// The browser must lose its session cookie too, or the very next
		// authorization request signs the user straight back in.
		out.ClearSessionCookies = true
	}

	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "oidc.session.logout",
		ProjectID:   projectID,
		Environment: env,
		AggregateID: sub,
		Payload: map[string]any{
			claimSubject:        sub,
			eventFieldSessionID: sid,
			"project_id":        projectID,
			claimEnvironment:    env,
			"client_id":         clientID,
		},
	}); err != nil {
		return nil, err
	}

	out.RedirectURL = a.postLogoutRedirect(ctx, clientID, cmd)

	return out, nil
}

// postLogoutRedirect resolves where to send the browser after logout. An
// unregistered target is refused rather than followed: post_logout_redirect_uri
// is attacker-controlled input, and honoring it unchecked turns every logout
// link into an open redirect.
func (a *pgOIDCGrants) postLogoutRedirect(ctx context.Context, clientID string, cmd domain.OIDCLogoutCmd) string {
	if cmd.PostLogoutRedirectURI == "" || clientID == "" {
		return "/"
	}

	row, err := models.FindIamAppClient(ctx, a.db.Bobx(), clientID)
	if err != nil {
		return "/"
	}

	var app domain.AppClient
	if err := unmarshal(row.Data, &app); err != nil {
		return "/"
	}

	if !containsString(app.PostLogoutRedirectURIs, cmd.PostLogoutRedirectURI) {
		return "/"
	}

	if cmd.State == "" {
		return cmd.PostLogoutRedirectURI
	}

	sep := "?"
	if strings.Contains(cmd.PostLogoutRedirectURI, "?") {
		sep = "&"
	}

	return cmd.PostLogoutRedirectURI + sep + url.Values{"state": {cmd.State}}.Encode()
}

// terminateSession revokes the IAM session an id_token names, reporting whether
// there was one to revoke. A session that is already gone is not an error — the
// caller asked for it to be over, and it is.
func (a *pgOIDCGrants) terminateSession(ctx context.Context, projectID, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, nil
	}

	return withTxRet(ctx, a.db, func(ctx context.Context) (bool, error) {
		row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
		if err != nil {
			if isStorageNotFound(translatePgErr("session", err)) {
				return false, nil
			}

			return false, err
		}

		if row.ProjectID != projectID {
			return false, nil // another tenant's session is not ours to end
		}

		if err := revokeSessionRecord(ctx, a.db, a.emitter, row, "rp_initiated_logout"); err != nil {
			return false, err
		}

		return true, nil
	})
}

// peekAudience reads the `aud` claim, which is a string OR an array of strings
// (RFC 7519 §4.1.3) — an id_token minted for one client still serializes it as
// an array, so reading it as a plain string silently yields nothing.
func peekAudience(claims map[string]any) string {
	switch v := claims[claimAudience].(type) {
	case string:
		return v
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				return s
			}
		}
	}

	return ""
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

	projectID, env := oidcParseIssuer(a.db.PublicURL, peekString(peek, claimIssuer))
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

// Token dispatches an /oauth2/token request. Code/refresh-token validation looks
// up the persisted hashes; mintTokenResponse then mints real RS256-signed access,
// id and (for offline_access) refresh tokens via the project Signer.
func (a *pgOIDCGrants) Token(ctx context.Context, cmd domain.OIDCTokenCmd) (map[string]any, error) {
	// private_key_jwt: a client that signed an assertion has proven itself
	// without any shared secret, so from here it is treated exactly like one the
	// transport authenticated.
	if cmd.ClientAssertion != "" {
		assertedClientID, assertedProject, assertedEnv, err := a.verifyClientAssertion(
			ctx, cmd.ProjectID, cmd.Env, cmd.ClientAssertionType, cmd.ClientAssertion,
			a.db.PublicURL+"/oauth2/token")
		if err != nil {
			return nil, err
		}

		cmd.AuthenticatedClientID = assertedClientID
		if cmd.ClientID == "" {
			cmd.ClientID = assertedClientID
		}
		// An assertion client authenticates nowhere else, so the tenant comes from
		// the client it proved itself to be.
		if cmd.ProjectID == "" {
			cmd.ProjectID = assertedProject
		}

		if cmd.Env == "" {
			cmd.Env = assertedEnv
		}
	}
	// A public client (PKCE, no secret) authenticates with nothing at the
	// transport, so it arrives with no tenant. RFC 6749 §3.2.1 requires it to
	// send client_id; the client's own row supplies the rest. Refusing here would
	// turn away exactly the clients PKCE exists for.
	if cmd.ProjectID == "" && cmd.ClientID != "" {
		if clientRow, _, err := a.resolveAuthorizeClient(ctx, cmd.ClientID); err == nil {
			cmd.ProjectID = clientRow.ProjectID
			if cmd.Env == "" {
				cmd.Env = clientRow.Environment
			}
		}
	}

	// The refresh grant runs outside the shared transaction on purpose. Spending
	// the presented token — and, when it turns out to have been replayed, burning
	// the session it belonged to — must COMMIT even though the exchange itself
	// then fails. Inside one transaction the rejection would roll the burn back
	// and reuse detection would detect and then forget.
	if cmd.GrantType == "refresh_token" {
		return a.tokenRefreshGrant(ctx, cmd)
	}
	// Client credentials mints a token for the calling service account itself.
	// There is no user, no code and no session to bind, so it shares nothing with
	// the grants below but the endpoint.
	if cmd.GrantType == string(oidc.GrantTypeClientCredentials) {
		return a.tokenClientCredentialsGrant(ctx, cmd)
	}

	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		switch cmd.GrantType {
		case "authorization_code":
			return a.tokenAuthorizationCodeGrant(ctx, cmd)
		case "device_code":
			return a.tokenDeviceCodeGrant(ctx, cmd)
		default:
			return nil, domain.ErrUnsupportedGrant
		}
	})
}

// tokenAuthorizationCodeGrant exchanges an authorization_code for tokens (RFC
// 6749 §4.1.3), enforcing: client_secret for a confidential client not
// already authenticated at the transport (H-01), the redirect_uri matching
// what was stored at authorize time (H-03), PKCE before the code is consumed
// so a failed exchange doesn't burn a valid code, and prior consent (M-02).
func (a *pgOIDCGrants) tokenAuthorizationCodeGrant(ctx context.Context, cmd domain.OIDCTokenCmd) (map[string]any, error) {
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

	if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowIn(ctx)) {
		return nil, domain.ErrTokenExpired
	}

	effectiveClientID := firstNonEmpty(row.ClientID.GetOrZero(), cmd.ClientID, cmd.AuthenticatedClientID)
	if cmd.AuthenticatedClientID != effectiveClientID {
		if err := a.oidcVerifyClientSecret(ctx, effectiveClientID, cmd.ClientSecret); err != nil {
			return nil, err
		}
	}

	// Parse the code data envelope for redirect_uri, nonce, and scopes.
	codeData, err := parseAuthCodeData(row.Data)
	if err != nil {
		return nil, err
	}

	if cmd.RedirectURI != "" || codeData.RedirectURI != "" {
		if subtle.ConstantTimeCompare([]byte(cmd.RedirectURI), []byte(codeData.RedirectURI)) != 1 {
			return nil, domain.ErrUnauthorized
		}
	}

	if err := oidcVerifyPKCE(codeData.CodeChallenge, codeData.CodeChallengeMethod, cmd.CodeVerifier); err != nil {
		return nil, err
	}

	if row.ProjectID != "" && row.UserID.GetOrZero() != "" && effectiveClientID != "" {
		if err := a.oidcVerifyConsent(ctx, row.ProjectID, row.UserID.GetOrZero(), effectiveClientID); err != nil {
			return nil, err
		}
	}

	consumed := true
	if err := row.Update(ctx, a.db.Bobx(), &models.IamAuthCodeSetter{Consumed: &consumed}); err != nil {
		return nil, err
	}

	codeEnv := row.Environment
	if codeEnv == "" {
		codeEnv = oidcDefaultEnv
	}

	tokenSubj := oidcTokenSubject{
		projectID: row.ProjectID,
		env:       codeEnv,
		subject:   row.UserID.GetOrZero(),
		clientID:  effectiveClientID,
		nonce:     codeData.Nonce,
		scopes:    splitScopesFromData(row.Data),
		sessionID: codeData.SessionID,
		authTime:  codeData.AuthTime,
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "oidc.token.issued",
		ProjectID:   row.ProjectID,
		Environment: codeEnv,
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
}

// tokenDeviceCodeGrant exchanges a device_code for tokens once the user has
// approved it on the verification page (RFC 8628 §3.5): approved mints
// tokens, denied is terminal, and anything else (still pending, or expired)
// tells the device to keep polling or give up per the RFC's error codes.
func (a *pgOIDCGrants) tokenDeviceCodeGrant(ctx context.Context, cmd domain.OIDCTokenCmd) (map[string]any, error) {
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
		if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowIn(ctx)) {
			return nil, domain.ErrTokenExpired
		}
		// RFC 8628: still pending.
		return nil, domain.ErrBadRequest
	}
}

// tokenRefreshGrant exchanges a refresh token, rotating it. It manages its own
// transactions: see Token for why the spend must not share one with the mint.
func (a *pgOIDCGrants) tokenRefreshGrant(ctx context.Context, cmd domain.OIDCTokenCmd) (map[string]any, error) {
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

	sub, tokenClientID, scopes, err := a.verifyRefreshToken(ctx, projectID, env, cmd.RefreshToken)
	if err != nil {
		return nil, err
	}
	// Who is asking, before the record has a chance to answer for them.
	presenter := firstNonEmpty(cmd.AuthenticatedClientID, cmd.ClientID, tokenClientID)

	// Spend the presented token in its own transaction, and take the grant from
	// the RECORD rather than the token's own claims: rotation limits the damage
	// of a leak, and a replayed token burns the whole session. The spend (and the
	// burn) must commit even when this exchange is then rejected.
	var stored oidcRefreshData

	row, err := withTxRet(ctx, a.db, func(ctx context.Context) (*models.IamRefreshToken, error) {
		redeemed, data, rerr := a.oidcRedeemRefreshToken(ctx, projectID, env, cmd.RefreshToken)
		stored = data

		return redeemed, rerr
	})
	if err != nil {
		if rerr := a.reportReuse(ctx, err, row, stored, env); rerr != nil {
			return nil, rerr
		}

		return nil, err
	}

	if len(stored.Scopes) > 0 {
		scopes = stored.Scopes
	}

	// A refresh token belongs to the client it was issued to. Compare against
	// the presenter, not against a value the record just supplied.
	if stored.ClientID != "" && presenter != stored.ClientID {
		return nil, domain.ErrInvalidGrant.WithMessage("refresh token was issued to a different client")
	}

	effectiveClientID := firstNonEmpty(stored.ClientID, presenter)

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
		nonce:     stored.Nonce,
		sessionID: row.SessionID,
	})
}

// reportReuse records a detected refresh-token replay. The burn itself already
// committed (see oidcRedeemRefreshToken); this runs outside the rolled-back
// transaction so the security event is not lost with it.
func (a *pgOIDCGrants) reportReuse(
	ctx context.Context, err error, row *models.IamRefreshToken, stored oidcRefreshData, env string,
) error {
	if !errors.Is(err, domain.ErrTokenUsed) || row == nil {
		return nil
	}

	return a.emitter.Emit(ctx, domain.Event{
		Type:        "oidc.token.reuse_detected",
		ProjectID:   row.ProjectID,
		Environment: env,
		AggregateID: row.SessionID,
		Payload: map[string]any{
			eventFieldSessionID: row.SessionID,
			eventFieldUserID:    row.UserID,
			"client_id":         stored.ClientID,
		},
	})
}

// denyAccessToken adds a presented access token to the revocation denylist and
// returns its jti. A token that does not verify for this tenant is left alone —
// RFC 7009 makes an unmatched token a no-op success, not an error.
func (a *pgOIDCGrants) denyAccessToken(ctx context.Context, cmd domain.OIDCRevokeCmd) (string, error) {
	verified, verifyErr := a.db.Signer().Verify(ctx, cmd.ProjectID, cmd.Env, cmd.Token)
	if verifyErr != nil {
		//nolint:nilerr // RFC 7009: a token we cannot match is a no-op success,
		// not a failure — telling the caller which tokens exist would be an oracle.
		return "", nil
	}

	jti := peekString(verified, claimTokenID)

	var expiresAt time.Time
	if exp, ok := verified["exp"].(float64); ok {
		expiresAt = time.Unix(int64(exp), 0).UTC()
	}

	return jti, denyToken(ctx, a.db, cmd.ProjectID, cmd.Env, jti, expiresAt)
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

// oidcParseIssuer extracts (projectID, env) from an absolute issuer of the form
// "<base>/p/<projectID>/e/<env>". It returns empty strings unless the issuer is
// exactly that — in particular an issuer whose origin/base path is not this
// deployment's is rejected rather than parsed, so a token minted by a foreign
// issuer can never route to one of our tenants.
func oidcParseIssuer(base, iss string) (string, string) {
	if base == "" || iss == "" {
		return "", ""
	}

	rest, ok := strings.CutPrefix(iss, base+"/")
	if !ok {
		return "", ""
	}

	parts := strings.Split(rest, "/")
	// "p" / <projectID> / "e" / <env>
	if len(parts) == 4 && parts[0] == "p" && parts[2] == "e" && parts[1] != "" && parts[3] != "" {
		return parts[1], parts[3]
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

	issuer := oidcIssuer(a.db.PublicURL, sub.projectID, env)
	now := nowUTC()

	// Token lifetimes come from the project's session_policy, like every other
	// token IAM issues. They used to be constants compiled in here, so a project
	// could configure the lifetime of its core-auth sessions and not of the
	// tokens its own OIDC provider handed out.
	accessTTL, refreshTTL := oidcAccessTTL, oidcRefreshTTL

	if a.cfg != nil {
		if sp, perr := a.cfg.SessionPolicy(ctx, sub.projectID); perr == nil {
			accessTTL, refreshTTL = sp.AccessTTL, sp.RefreshTTL
		}
	}
	// A client bound to a token profile gets that profile's audience, lifetimes
	// and extra claims. Everything the profile leaves unset stays as the
	// project's default.
	profile := a.resolveClientTokenProfile(ctx, sub.clientID)

	// The `groups` scope projects the user's IAM role assignments into the token.
	// The values are read from storage, never taken from the request: a client
	// can ask for the scope, but it cannot ask for a role.
	// A granted scope always yields the claim, empty list included: "asked and
	// has no roles" must be distinguishable from "did not ask".
	var groups []string

	if oidcHasScope(sub.scopes, oidcScopeGroups) {
		roles, err := userRoles(ctx, a.db, sub.projectID, env, sub.subject)
		if err != nil {
			return nil, err
		}

		groups = roles
		if groups == nil {
			groups = []string{}
		}
	}

	access, accessTTL, refreshTTL, err := a.mintOIDCAccessToken(ctx, sub, env, issuer, profile, groups, accessTTL, refreshTTL)
	if err != nil {
		return nil, err
	}

	resp := oidc.AccessTokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		// accessTTL cannot be non-positive here: it starts at the compiled-in
		// default, session_policy's own validation rejects <= 0, and a token
		// profile only ever overrides it with a value > 0. Sign loss on the
		// int64->uint64 conversion gosec is warning about needs a negative
		// input, which none of those paths can produce.
		ExpiresIn: uint64(accessTTL / time.Second), //nolint:gosec // accessTTL is always > 0, see comment above
		Scope:     oidc.SpaceDelimitedArray(sub.scopes),
	}

	// id_token: only for openid requests. Built from the zitadel IDTokenClaims
	// struct (correct field names), then signed by OUR key via the Signer.
	// Profile / email claims for the scopes the client was granted.
	scopeClaims, err := a.oidcScopeClaims(ctx, sub.subject, sub.scopes)
	if err != nil {
		return nil, err
	}

	if oidcHasScope(sub.scopes, "openid") {
		idToken, err := a.mintIDToken(ctx, sub, env, issuer, access, now, groups, scopeClaims, accessTTL)
		if err != nil {
			return nil, err
		}

		resp.IDToken = idToken
	}

	// refresh_token: signed, rotatable JWT for offline_access requests.
	if oidcHasScope(sub.scopes, "offline_access") {
		refresh, err := a.mintOIDCRefreshToken(ctx, sub, env, issuer, refreshTTL)
		if err != nil {
			return nil, err
		}

		resp.RefreshToken = refresh
	}

	return oidcClaimsMap(resp)
}

// mintOIDCAccessToken builds and signs the access-token JWT: the token
// profile's claim template goes in first so a standard claim always wins
// over it, then applyTokenProfile may override the TTLs, then groups/session
// are layered on. Returns the (possibly profile-adjusted) TTLs alongside the
// token since the refresh token minted afterward must use the same values.
func (a *pgOIDCGrants) mintOIDCAccessToken(ctx context.Context, sub oidcTokenSubject, env, issuer string, profile *oidcTokenProfile, groups []string, accessTTL, refreshTTL time.Duration) (string, time.Duration, time.Duration, error) {
	accessClaims := tokenProfileClaims(profile)
	if accessClaims == nil {
		accessClaims = map[string]any{}
	}

	for name, value := range map[string]any{
		claimIssuer:    issuer,
		claimSubject:   sub.subject,
		claimAudience:  sub.clientID,
		claimClientID:  sub.clientID,
		claimScope:     joinScopes(sub.scopes),
		claimTokenType: tokenTypeAccess,
		// jti names this token so revocation can name it back; sid ties it to the
		// session, so revoking the session kills it and back-channel logout can
		// tell relying parties which one ended.
		claimTokenID: newUUID(),
		// pid/env identify the tenant whose key signed this token. The verifier
		// selects the signing key by them, so a token without pid cannot be
		// verified at all — including at our own userinfo endpoint, which is
		// where relying parties resolve claims the id_token did not carry.
		claimProjectID:   sub.projectID,
		claimEnvironment: env,
	} {
		accessClaims[name] = value
	}

	accessTTL, refreshTTL = applyTokenProfile(profile, accessClaims, accessTTL, refreshTTL)

	if groups != nil {
		accessClaims[claimGroups] = groups
	}

	if sub.sessionID != "" {
		accessClaims[claimSessionID] = sub.sessionID
	}

	access, err := a.db.Signer().Sign(ctx, sub.projectID, env, accessClaims, accessTTL)
	if err != nil {
		return "", 0, 0, err
	}

	return access, accessTTL, refreshTTL, nil
}

// mintOIDCRefreshToken builds, signs, and records a rotatable refresh-token
// JWT for an offline_access grant. Recording it is what makes rotation,
// revocation, and replay detection possible — a refresh token that exists
// only as a signature supports none of those.
func (a *pgOIDCGrants) mintOIDCRefreshToken(ctx context.Context, sub oidcTokenSubject, env, issuer string, refreshTTL time.Duration) (string, error) {
	refreshClaims := map[string]any{
		claimIssuer:      issuer,
		claimSubject:     sub.subject,
		claimAudience:    sub.clientID,
		claimClientID:    sub.clientID,
		claimScope:       joinScopes(sub.scopes),
		claimTokenType:   tokenTypeRefresh,
		claimTokenID:     newUUID(),
		claimProjectID:   sub.projectID,
		claimEnvironment: env,
	}
	if sub.sessionID != "" {
		refreshClaims[claimSessionID] = sub.sessionID
	}

	refresh, err := a.db.Signer().Sign(ctx, sub.projectID, env, refreshClaims, refreshTTL)
	if err != nil {
		return "", err
	}

	if err := a.storeOIDCRefreshToken(ctx, sub, env, refresh, refreshTTL); err != nil {
		return "", err
	}

	return refresh, nil
}

// mintIDToken builds an OIDC id_token for sub using the zitadel IDTokenClaims
// struct for correct claim names, sets the access-token hash (at_hash), and
// signs it with OUR project key via the Signer. groups is the resolved role list
// (nil unless the `groups` scope was granted) and is carried alongside the
// standard claims.
func (a *pgOIDCGrants) mintIDToken(
	ctx context.Context, sub oidcTokenSubject, env, issuer, accessToken string, now time.Time,
	groups []string, scopeClaims map[string]any, ttl time.Duration,
) (string, error) {
	idc := oidc.NewIDTokenClaims(
		issuer,
		sub.subject,
		[]string{sub.clientID},
		now.Add(ttl),
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
	claims[claimEnvironment] = env

	if groups != nil {
		claims[claimGroups] = groups
	}

	for k, v := range scopeClaims {
		claims[k] = v
	}

	if sub.sessionID != "" {
		claims[claimSessionID] = sub.sessionID
	}

	if !sub.authTime.IsZero() {
		claims["auth_time"] = sub.authTime.Unix()
	}

	return a.db.Signer().Sign(ctx, sub.projectID, env, claims, ttl)
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
	if err == nil {
		// A revoked token is inactive, whatever its signature says.
		denied, derr := tokenDenied(ctx, a.db, peekString(claims, claimTokenID))
		if derr != nil {
			return nil, derr
		}

		if denied {
			return inactive, nil
		}
	}

	if err != nil {
		return inactive, nil
	}

	if iss, _ := claims[claimIssuer].(string); iss != oidcIssuer(a.db.PublicURL, cmd.ProjectID, env) {
		return inactive, nil // issuer does not match the request tenant
	}

	return oidcClaimsMap(oidcIntrospectionFromClaims(claims))
}

// oidcIntrospectionFromClaims maps a verified token's claims onto the RFC
// 7662 introspection response. Every field but Active/TokenType is optional
// on the claim set (a refresh token, for instance, carries no scope), so each
// is copied only when present.
func oidcIntrospectionFromClaims(claims map[string]any) oidc.IntrospectionResponse {
	resp := oidc.IntrospectionResponse{Active: true, TokenType: "Bearer"}

	if v, ok := claims["sub"].(string); ok {
		resp.Subject = v
	}

	if v, ok := claims["client_id"].(string); ok {
		resp.ClientID = v
	}

	if v, ok := claims[claimIssuer].(string); ok {
		resp.Issuer = v
	}

	if v, ok := claims["aud"].(string); ok && v != "" {
		resp.Audience = oidc.Audience{v}
	}

	if v, ok := claims["scope"].(string); ok && v != "" {
		resp.Scope = oidc.SpaceDelimitedArray(splitScopes(v))
	}

	if v, ok := claims["exp"].(float64); ok {
		resp.Expiration = oidc.FromTime(time.Unix(int64(v), 0))
	}

	if v, ok := claims["iat"].(float64); ok {
		resp.IssuedAt = oidc.FromTime(time.Unix(int64(v), 0))
	}

	if v, ok := claims["nbf"].(float64); ok {
		resp.NotBefore = oidc.FromTime(time.Unix(int64(v), 0))
	}

	return resp
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

		aggregateID := ""
		if len(rows) > 0 {
			aggregateID = rows[0].ID
		}
		// A refresh token is a record, so revoking it is a write. RFC 7009 asks
		// the server to accept either token type here regardless of the hint.
		refreshRows, err := models.IamRefreshTokens.Query(
			sm.Where(models.IamRefreshTokens.Columns.Hash.EQ(psql.Arg(hash))),
			sm.Where(models.IamRefreshTokens.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Limit(1),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return fmt.Errorf("read refresh token: %w", err)
		}

		if len(refreshRows) > 0 {
			var data oidcRefreshData
			if len(refreshRows[0].Data) > 0 {
				if uerr := unmarshal(refreshRows[0].Data, &data); uerr != nil {
					return uerr
				}
			}

			if err := a.markRefreshRevoked(ctx, refreshRows[0], data); err != nil {
				return err
			}

			aggregateID = refreshRows[0].ID
		}
		// An access token is verified offline and can only be stopped by naming it.
		jti, derr := a.denyAccessToken(ctx, cmd)
		if derr != nil {
			return derr
		}

		if aggregateID == "" {
			aggregateID = jti
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
// oidcParTTL is the lifetime of a pushed authorization request. RFC 9126 §2.2
// recommends keeping it short — the client redeems it immediately.
const oidcParTTL = 90 * time.Second

// oidcParRequestURIPrefix is the RFC 9126 URN namespace for a pushed request.
const oidcParRequestURIPrefix = "urn:ietf:params:oauth:request_uri:"

func (a *pgOIDCGrants) PushAuthorizationRequest(ctx context.Context, cmd domain.OIDCParCmd) (*domain.OIDCParResult, error) {
	// RFC 9126 §2.1: the pushed request must be validated as the authorization
	// endpoint would validate it. Doing it here means a request that could never
	// be authorized is refused now, instead of after the user has been walked
	// through a login screen. It also means the request_uri, once issued, always
	// names a request whose client and redirect_uri are already known good.
	clientRow, app, err := a.resolveAuthorizeClient(ctx, cmd.ClientID)
	if err != nil {
		return nil, err
	}

	redirectURI, err := authorizeRedirectURI(app, cmd.RedirectURI)
	if err != nil {
		return nil, err
	}

	if cmd.ResponseType != oidcResponseTypeCode {
		return nil, domain.ErrBadRequest.WithMessage("only response_type=code is supported")
	}

	if len(splitScopes(cmd.Scope)) == 0 {
		return nil, domain.ErrBadRequest.WithMessage("scope is required")
	}

	if errCode, desc := oidcCheckPKCERequest(app, cmd.CodeChallenge, cmd.CodeChallengeMethod); errCode != "" {
		return nil, domain.ErrBadRequest.WithMessage(desc)
	}

	// Store the resolved redirect_uri so redeeming the request cannot depend on
	// re-deriving it later.
	cmd.RedirectURI = redirectURI

	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.OIDCParResult, error) {
		opaque, err := oidcRandToken(32)
		if err != nil {
			return nil, err
		}

		requestURI := oidcParRequestURIPrefix + opaque

		raw, err := marshal(&cmd)
		if err != nil {
			return nil, err
		}

		rm := json.RawMessage(raw)

		cid := null.From(cmd.ClientID)
		setter := &models.IamParRequestSetter{
			ID:         ptr(newUUID()),
			ProjectID:  ptr(clientRow.ProjectID),
			RequestURI: &requestURI,
			ClientID:   &cid,
			ExpiresAt:  ptr(nowUTC().Add(oidcParTTL)),
			Data:       &rm,
		}

		parRow, err := models.IamParRequests.Insert(setter).One(ctx, a.db.Bobx())
		if err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}

			return nil, err
		}

		result := &domain.OIDCParResult{RequestURI: requestURI, ExpiresIn: int(oidcParTTL / time.Second)}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.par.created",
			ProjectID:   clientRow.ProjectID,
			Environment: clientRow.Environment,
			AggregateID: parRow.ID,
			Payload:     result,
		}); err != nil {
			return nil, err
		}

		return result, nil
	})
}

// consumePushedRequest redeems a request_uri and returns the authorization
// request that was pushed under it.
//
// The request is single-use: it is deleted as it is read, so a request_uri that
// leaks through the user-agent cannot be replayed. It is also bound to the
// client that pushed it — a different client presenting somebody else's
// request_uri gets nothing.
func (a *pgOIDCGrants) consumePushedRequest(
	ctx context.Context, requestURI, clientID string,
) (domain.OIDCAuthorizeCmd, error) {
	var out domain.OIDCAuthorizeCmd

	return withTxRet(ctx, a.db, func(ctx context.Context) (domain.OIDCAuthorizeCmd, error) {
		rows, err := models.IamParRequests.Query(
			sm.Where(models.IamParRequests.Columns.RequestURI.EQ(psql.Arg(requestURI))),
			sm.Limit(1),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return out, fmt.Errorf("read pushed request: %w", err)
		}

		if len(rows) == 0 {
			return out, domain.ErrInvalidRequestURI
		}

		row := rows[0]

		// Consume first: an expired or mismatched request is dropped too, so a
		// leaked request_uri cannot be probed repeatedly.
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return out, err
		}

		if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowIn(ctx)) {
			return out, domain.ErrInvalidRequestURI
		}

		if bound := row.ClientID.GetOrZero(); bound != "" && bound != clientID {
			return out, domain.ErrInvalidRequestURI
		}

		var pushed domain.OIDCParCmd
		if err := unmarshal(row.Data, &pushed); err != nil {
			return out, err
		}

		return domain.OIDCAuthorizeCmd{
			ClientID:            pushed.ClientID,
			ResponseType:        pushed.ResponseType,
			RedirectURI:         pushed.RedirectURI,
			Scope:               pushed.Scope,
			State:               pushed.State,
			CodeChallenge:       pushed.CodeChallenge,
			CodeChallengeMethod: pushed.CodeChallengeMethod,
			Nonce:               pushed.Nonce,
			Prompt:              pushed.Prompt,
			ResponseMode:        pushed.ResponseMode,
		}, nil
	})
}

// insertDeviceCodeRow persists the pending device-code row: device_code is
// stored as a hash (only the caller ever sees the plaintext), the
// OIDCDevicePending view goes into the data envelope for the verification UI.
func insertDeviceCodeRow(ctx context.Context, db *DB, clientRow *models.IamAppClient, deviceCode, userCode string, expiresAt time.Time, pending domain.OIDCDevicePending) (*models.IamDeviceCode, error) {
	raw, err := marshal(&pending)
	if err != nil {
		return nil, err
	}

	rm := json.RawMessage(raw)
	setter := &models.IamDeviceCodeSetter{
		ID:          ptr(newUUID()),
		ProjectID:   ptr(clientRow.ProjectID),
		Environment: ptr(clientRow.Environment),
		DeviceCode:  ptr(oidcHashToken(deviceCode)),
		UserCode:    &userCode,
		Status:      ptr("pending"),
		ExpiresAt:   &expiresAt,
		Data:        &rm,
	}

	row, err := models.IamDeviceCodes.Insert(setter).One(ctx, db.Bobx())
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrConflict
		}

		return nil, err
	}

	return row, nil
}

// DeviceAuthorization starts a device authorization grant (RFC 8628). The
// device_code is stored as a hash; the plaintext device_code and user_code are
// returned to the client exactly once.
func (a *pgOIDCGrants) DeviceAuthorization(ctx context.Context, cmd domain.OIDCDeviceAuthorizationCmd) (*domain.OIDCDeviceAuthorization, error) {
	// Resolve the client so the pending grant is stored under its real tenant.
	// The verification page looks the user_code up by PROJECT (that is all a
	// signed-in browser knows), so a row keyed by client id can never be found —
	// which made the whole user half of the device grant unreachable.
	clientRow, _, err := a.resolveAuthorizeClient(ctx, cmd.ClientID)
	if err != nil {
		return nil, err
	}

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

		// RFC 8628 §3.2: the device prints verification_uri for a human to type
		// into another device, so it has to be an absolute URL — a path is
		// useless on a TV screen. It points at the hosted verification page and
		// carries the tenant the page needs to resolve the code.
		verificationURI := a.db.PublicURL + "/oauth/device"
		verificationQuery := url.Values{
			"user_code": {userCode},
			"client_id": {clientRow.ProjectID},
		}

		out := &domain.OIDCDeviceAuthorization{
			DeviceCode:              deviceCode,
			UserCode:                userCode,
			VerificationURI:         verificationURI,
			VerificationURIComplete: verificationURI + "?" + verificationQuery.Encode(),
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

		deviceRow, err := insertDeviceCodeRow(ctx, a.db, clientRow, deviceCode, userCode, expiresAt, pending)
		if err != nil {
			return nil, err
		}

		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oidc.device.authorized",
			ProjectID:   clientRow.ProjectID,
			Environment: clientRow.Environment,
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
	// JWT was verified by the auth middleware), so the subject is settled. A
	// relying party that falls back to userinfo because the id_token was thin
	// must find the same profile/email claims here, or it still cannot name the
	// user. The claim names come from the OIDC UserInfo struct so they stay
	// spec-correct.
	out, err := oidcClaimsMap(&oidc.UserInfo{Subject: accountID})
	if err != nil {
		return nil, err
	}
	// Only the scopes the presented access token actually carries: userinfo must
	// not become a way around the consent the user gave. A token granted plain
	// `openid` gets a subject and nothing else.
	var granted []string
	if p, ok := api.PrincipalFrom(ctx); ok && p != nil {
		granted = p.Scopes
	}

	extra, err := a.oidcScopeClaims(ctx, accountID, granted)
	if err != nil {
		return nil, err
	}

	for k, v := range extra {
		out[k] = v
	}

	return out, nil
}

// oidcScopeClaims resolves the standard OIDC profile/email claims a relying
// party expects for the scopes it was granted (OIDC Core §5.4).
//
// It matters more than it looks: oauth2-proxy, Grafana and friends identify the
// signed-in person by the `email` claim and refuse a token without one. An
// id_token carrying nothing but `sub` authenticates a subject nobody can name.
//
// Only granted scopes contribute — a client that asked for neither gets neither.
func (a *pgOIDCGrants) oidcScopeClaims(ctx context.Context, accountID string, scopes []string) (map[string]any, error) {
	out := map[string]any{}

	wantEmail := oidcHasScope(scopes, oidc.ScopeEmail)
	wantProfile := oidcHasScope(scopes, oidc.ScopeProfile)

	if accountID == "" || (!wantEmail && !wantProfile) {
		return out, nil
	}

	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		if isStorageNotFound(translatePgErr("user", err)) {
			// A subject with no account row discloses nothing rather than failing
			// the token: the token itself is still valid for what it asserts.
			return out, nil
		}

		return nil, err
	}

	var acct domain.Account
	if err := unmarshal(row.Data, &acct); err != nil {
		return nil, err
	}

	if wantEmail && acct.PrimaryEmail != "" {
		out["email"] = acct.PrimaryEmail
		out["email_verified"] = acct.EmailVerified
	}

	if wantProfile {
		if acct.Name != "" {
			out["name"] = acct.Name
		}

		if acct.Locale != "" {
			out["locale"] = acct.Locale
		}

		if acct.PrimaryPhone != "" {
			out["phone_number"] = acct.PrimaryPhone
			out["phone_number_verified"] = acct.PhoneVerified
		}

		if !acct.UpdatedAt.IsZero() {
			out["updated_at"] = acct.UpdatedAt.Unix()
		}
	}

	return out, nil
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
		if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowIn(ctx)) {
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
// names) and marshaled to the generic map the oas layer emits. The signing
// algorithm advertised matches the Signer (RS256).
func (a *pgOIDCGrants) OpenIDConfiguration(ctx context.Context, projectID, env string) (map[string]any, error) {
	root := a.db.PublicURL
	// issuer is the deployment base + tenant path, which is exactly the prefix of
	// the URL this document is served from
	// (<issuer>/.well-known/openid-configuration). Conforming clients (go-oidc and
	// friends) reject the document outright when the two disagree.
	issuer := oidcIssuer(root, projectID, env)
	cfg := &oidc.DiscoveryConfiguration{
		Issuer:                      issuer,
		AuthorizationEndpoint:       root + "/oauth2/authorize",
		TokenEndpoint:               root + "/oauth2/token",
		UserinfoEndpoint:            root + "/oauth2/userinfo",
		JwksURI:                     issuer + "/.well-known/jwks.json",
		IntrospectionEndpoint:       root + "/oauth2/introspect",
		RevocationEndpoint:          root + "/oauth2/revoke",
		DeviceAuthorizationEndpoint: root + "/oauth2/device_authorization",
		EndSessionEndpoint:          root + "/oauth2/logout",
		ResponseTypesSupported:      []string{oidcResponseTypeCode},
		ResponseModesSupported: []string{
			oidcResponseModeQuery, oidcResponseModeFragment, oidcResponseModeFormPost,
		},
		GrantTypesSupported: []oidc.GrantType{
			oidc.GrantTypeCode, oidc.GrantTypeRefreshToken, oidc.GrantTypeDeviceCode,
			oidc.GrantTypeClientCredentials,
		},
		SubjectTypesSupported: []string{"public"},
		ScopesSupported: []string{
			oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail, oidc.ScopeOfflineAccess, oidcScopeGroups,
		},
		ClaimsSupported:                  oidcClaimsSupported(),
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []oidc.AuthMethod{
			oidc.AuthMethodBasic, oidc.AuthMethodPost, oidc.AuthMethodNone,
			oidc.AuthMethodPrivateKeyJWT,
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
	m["pushed_authorization_request_endpoint"] = root + "/oauth2/par"
	// The prompt values the authorization endpoint acts on. The struct in this
	// lib version has no field for them; advertising them is only honest now that
	// they are honored rather than ignored.
	// RFC 9207: we name ourselves on every authorization response, so clients can
	// and should check it.
	m["authorization_response_iss_parameter_supported"] = true
	// Algorithms accepted on a client assertion and on a signed request object.
	m["token_endpoint_auth_signing_alg_values_supported"] = oidcAssertionAlgorithms()
	// Signed request objects are accepted by value; the by-reference form is PAR,
	// whose request_uri we issue ourselves.
	m["request_parameter_supported"] = true
	m["request_object_signing_alg_values_supported"] = oidcAssertionAlgorithms()
	// RFC 7591. Registration is not open: the endpoint takes a project-admin
	// token as the initial access token, which is also what says which project
	// the new client lands in.
	m["registration_endpoint"] = root + "/oauth2/register"

	m["prompt_values_supported"] = []string{
		oidcPromptNone, oidcPromptLogin, oidcPromptConsent, oidcPromptSelectAccount,
	}

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

	var outSb1304 strings.Builder

	for i, s := range scopes {
		if i > 0 {
			outSb1304.WriteString(" ")
		}

		outSb1304.WriteString(s)
	}

	out += outSb1304.String()

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
	// PKCE parameters copied from the authorization request. An empty challenge
	// means the code was issued without PKCE, which the token endpoint treats as
	// "a verifier must NOT be presented".
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
	// SessionID / AuthTime come from the interaction the code was minted from.
	SessionID string    `json:"session_id,omitempty"`
	AuthTime  time.Time `json:"auth_time,omitempty"`
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
// confidential client, verifies the supplied secret against its stored sha256
// hashes using constant-time comparison.
//
// The secrets live in iam_app_secrets — a client may hold several so one can be
// rotated before the old one is dropped — and any of them authenticates the
// client. A hash inside the client's own data envelope is also accepted, for
// clients provisioned that way.
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

	given := sha256.Sum256([]byte(clientSecret))
	givenHex := hex.EncodeToString(given[:])

	var data struct {
		ClientSecretHash string `json:"client_secret_hash"`
	}
	if err := json.Unmarshal(row.Data, &data); err == nil && data.ClientSecretHash != "" {
		if subtle.ConstantTimeCompare([]byte(givenHex), []byte(data.ClientSecretHash)) == 1 {
			return nil
		}
	}

	secrets, err := models.IamAppSecrets.Query(
		sm.Where(models.IamAppSecrets.Columns.ProjectID.EQ(psql.Arg(row.ProjectID))),
		sm.Where(models.IamAppSecrets.Columns.AppID.EQ(psql.Arg(clientID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return fmt.Errorf("read client secrets: %w", err)
	}

	// Compare against every issued secret without short-circuiting, so the reply
	// time does not reveal which one (if any) was close.
	matched := 0
	for _, secret := range secrets {
		matched |= subtle.ConstantTimeCompare([]byte(givenHex), []byte(secret.Hash))
	}

	if matched != 1 {
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
