package postgres

// OAuthSocial adapter — satisfies api.OAuthSocialAccounts.
//
// Spans three envelopes:
//   - iam_providers (kind=oauth, enabled=true) — the read model behind
//     EnabledProviders.
//   - iam_identities — the provider links (Link / Unlink / CompleteLogin's
//     identity upsert).
//   - iam_users + iam_sessions — the account a social login resolves to and the
//     session it mints.
//
// The provider code-exchange / redirect-URL handshake with the upstream IdP is
// implemented with golang.org/x/oauth2: an oauth2.Config is assembled from the
// iam_providers row (kind=oauth, config in the data envelope), AuthCodeURL drives
// the browser redirect and Exchange + a userinfo fetch resolves the external
// id/email. We persist the identity link, account and session, and mint a real
// signed JWT access token via the project Signer.

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"golang.org/x/oauth2"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

const (
	// oauthSocialDefaultEnv is the environment whose signing key mints the access
	// token for a social login session.
	oauthSocialDefaultEnv = "live"

	// timeSecondDur is time.Second, used to derive ExpiresIn (seconds) from the
	// access-token TTL.
	timeSecondDur = time.Second

	// oauthSocialAccessTTL / oauthSocialRefreshTTL bound the minted access and
	// refresh JWTs.
	oauthSocialAccessTTL  = time.Hour
	oauthSocialRefreshTTL = 30 * 24 * time.Hour

	// oauthSocialExchangeCodeTTL bounds the single-use exchange code that maps a
	// minted session to the ?code= redirect handed back by CompleteLoginRedirect.
	oauthSocialExchangeCodeTTL = 5 * time.Minute
)

// pgOAuthSocial is the Postgres-backed OAuthSocialAccounts adapter.
type pgOAuthSocial struct {
	db      *DB
	emitter Emitter
}

// NewPgOAuthSocial builds the OAuthSocial adapter over the connection bundle.
func NewPgOAuthSocial(db *DB, emitter Emitter) *pgOAuthSocial {
	return &pgOAuthSocial{db: db, emitter: emitter}
}

var _ api.OAuthSocialAccounts = (*pgOAuthSocial)(nil)

// oauthProviderData is the iam_providers `data` jsonb envelope for an OAuth
// provider: the display name, requested scopes and the upstream client config
// (credentials + endpoints) used to build the golang.org/x/oauth2 Config. The
// id/project/kind/provider live in the lookup columns.
//
// The endpoint fields may live either at the top level or nested under a
// `config` object (the shape AdminConfig persists); we accept both.
type oauthProviderData struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`

	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"userinfo_url"`
	RedirectURL  string `json:"redirect_url"`

	Config *oauthProviderData `json:"config,omitempty"`
}

// resolved folds the optional nested `config` envelope into the top level so
// callers see a single flat view of the provider's OAuth settings.
func (d oauthProviderData) resolved() oauthProviderData {
	out := d
	out.Config = nil
	if d.Config != nil {
		c := *d.Config
		if out.Name == "" {
			out.Name = c.Name
		}
		if len(out.Scopes) == 0 {
			out.Scopes = c.Scopes
		}
		if out.ClientID == "" {
			out.ClientID = c.ClientID
		}
		if out.ClientSecret == "" {
			out.ClientSecret = c.ClientSecret
		}
		if out.AuthURL == "" {
			out.AuthURL = c.AuthURL
		}
		if out.TokenURL == "" {
			out.TokenURL = c.TokenURL
		}
		if out.UserInfoURL == "" {
			out.UserInfoURL = c.UserInfoURL
		}
		if out.RedirectURL == "" {
			out.RedirectURL = c.RedirectURL
		}
	}
	return out
}

// oauthUserInfo is the subset of an OAuth/OIDC userinfo response we consume: the
// stable external subject id and the (provider-asserted) email. Different IdPs
// spell the id field differently, so we accept the common variants.
type oauthUserInfo struct {
	Sub      string `json:"sub"`
	ID       any    `json:"id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Verified *bool  `json:"email_verified"`
}

// externalID returns the stable provider account id, preferring `sub` (OIDC),
// then `id` (e.g. GitHub/Google v1), then `user_id`.
func (u oauthUserInfo) externalID() string {
	if u.Sub != "" {
		return u.Sub
	}
	switch v := u.ID.(type) {
	case string:
		if v != "" {
			return v
		}
	case float64:
		if v != 0 {
			return jsonNumberString(v)
		}
	case json.Number:
		if s := v.String(); s != "" {
			return s
		}
	}
	return u.UserID
}

// jsonNumberString renders a numeric id without scientific notation.
func jsonNumberString(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

// EnabledProviders lists the enabled OAuth providers configured for a project.
// Tenant boundary: only rows whose project_id matches are returned.
func (a *pgOAuthSocial) EnabledProviders(ctx context.Context, projectID string) ([]domain.OAuthProvider, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg("oauth"))),
		sm.Where(models.IamProviders.Columns.Enabled.EQ(psql.Arg(true))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.OAuthProvider, 0, len(rows))
	for _, row := range rows {
		var raw oauthProviderData
		if err := unmarshal(row.Data, &raw); err != nil {
			return nil, err
		}
		d := raw.resolved()
		name := d.Name
		if name == "" {
			name = row.Provider
		}
		out = append(out, domain.OAuthProvider{
			ID:     row.Provider,
			Name:   name,
			Scopes: d.Scopes,
		})
	}
	return out, nil
}

// CompleteLogin resolves the OAuth callback `code` into an account + session.
//
// It swaps `code` at the provider token endpoint (golang.org/x/oauth2 Exchange)
// and fetches the userinfo claims (provider account id + email), then upserts the
// iam_identities link, resolves/creates the iam_users account, and mints an
// iam_sessions row.
func (a *pgOAuthSocial) CompleteLogin(ctx context.Context, projectID, provider, code string) (*domain.Account, *domain.Session, error) {
	if projectID == "" || provider == "" || code == "" {
		return nil, nil, domain.ErrBadRequest
	}

	// Exchange the code at the provider token endpoint and fetch the userinfo
	// claims (provider account id + email) BEFORE opening the serializable tx —
	// the upstream round-trip must not hold the transaction open.
	cfg, d, err := a.loadOAuthConfig(ctx, projectID, provider, "")
	if err != nil {
		return nil, nil, err
	}
	providerAccountID, email, err := a.oauthExchange(ctx, cfg, d, code, "")
	if err != nil {
		return nil, nil, err
	}

	type result struct {
		acct *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// Find an existing identity for this provider account (tenant-scoped).
		ident, err := a.findIdentity(ctx, projectID, provider, providerAccountID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return result{}, err
		}

		var acct *domain.Account
		if errors.Is(err, domain.ErrNotFound) {
			// No link yet: provision a new account and link the identity.
			acct, err = a.createSocialAccount(ctx, projectID, email)
			if err != nil {
				return result{}, err
			}
			if err := a.insertIdentity(ctx, &domain.Identity{
				ID:                newUUID(),
				Type:              "oauth",
				Provider:          provider,
				ProviderAccountID: providerAccountID,
				Email:             email,
			}, projectID, acct.ID); err != nil {
				return result{}, err
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "identity.linked",
				ProjectID:   projectID,
				AggregateID: acct.ID,
				Payload:     acct,
			}); err != nil {
				return result{}, err
			}
		} else {
			acct, err = a.loadAccount(ctx, projectID, ident.UserID)
			if err != nil {
				return result{}, err
			}
		}

		sess, err := a.mintSession(ctx, acct)
		if err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "session.created",
			ProjectID:   acct.ProjectID,
			AggregateID: sess.ID,
			Payload:     sess,
		}); err != nil {
			return result{}, err
		}
		return result{acct: acct, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acct, res.sess, nil
}

// Link attaches a provider identity to an already authenticated account.
//
// As with CompleteLogin the upstream code-exchange runs via golang.org/x/oauth2
// (Exchange + userinfo) to resolve the provider account, then we persist the
// link to iam_identities for the account.
func (a *pgOAuthSocial) Link(ctx context.Context, accountID, provider, code string) error {
	if accountID == "" || provider == "" || code == "" {
		return domain.ErrBadRequest
	}

	// The account row carries the tenant; we link inside its project. Resolve it
	// up front so the upstream code-exchange runs outside the serializable tx.
	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		return translatePgErr("user", err)
	}
	projectID := row.ProjectID

	cfg, d, err := a.loadOAuthConfig(ctx, projectID, provider, "")
	if err != nil {
		return err
	}
	providerAccountID, email, err := a.oauthExchange(ctx, cfg, d, code, "")
	if err != nil {
		return err
	}

	return a.db.withTx(ctx, func(ctx context.Context) error {
		// Reject if this provider account is already linked elsewhere.
		if existing, err := a.findIdentity(ctx, projectID, provider, providerAccountID); err == nil {
			if existing.UserID == accountID {
				return domain.ErrAlreadyLinked
			}
			return domain.ErrIdentityExists
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		if err := a.insertIdentity(ctx, &domain.Identity{
			ID:                newUUID(),
			Type:              "oauth",
			Provider:          provider,
			ProviderAccountID: providerAccountID,
			Email:             email,
		}, projectID, accountID); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "identity.linked",
			ProjectID:   projectID,
			AggregateID: accountID,
			Payload:     map[string]any{"account_id": accountID, "provider": provider, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// Unlink removes a provider identity from an account. The identity must belong
// to the account (tenant + ownership boundary), else not-found.
func (a *pgOAuthSocial) Unlink(ctx context.Context, accountID, identityID string) error {
	if accountID == "" || identityID == "" {
		return domain.ErrBadRequest
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamIdentity(ctx, a.db.Bobx(), identityID)
		if err != nil {
			return translatePgErr("identity", err)
		}
		if row.UserID != accountID { // ownership boundary
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "identity.unlinked",
			ProjectID:   row.ProjectID,
			AggregateID: identityID,
			Payload:     map[string]any{"id": identityID, "project_id": row.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// oauthExchangeCodeData is the iam_auth_codes data envelope for a social-login
// exchange code: the minted session plus the optional PKCE challenge bound to
// the code at issue time. Stored whole in the code's data jsonb column.
type oauthExchangeCodeData struct {
	Session       *domain.Session `json:"session"`
	CodeChallenge string          `json:"code_challenge,omitempty"`
}

// Exchange resolves a one-time exchange code (issued by CompleteLoginRedirect)
// into the account + session it authenticated, scoped to the command's project.
// The code is looked up by sha256 hash in iam_auth_codes; missing / consumed /
// expired codes map to domain.ErrInvalidToken. The code is consumed (single-use)
// before the stored session is returned. When a PKCE code_challenge was stored
// with the code, the supplied CodeVerifier must hash (S256) to it; a flow that
// carried no challenge skips PKCE.
func (a *pgOAuthSocial) Exchange(ctx context.Context, cmd domain.OAuthSocialExchangeCmd) (*domain.Account, *domain.Session, error) {
	if cmd.ProjectID == "" || cmd.Code == "" {
		return nil, nil, domain.ErrBadRequest
	}
	type result struct {
		acct *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		hash := fedHashToken(cmd.Code)
		rows, err := models.IamAuthCodes.Query(
			sm.Where(models.IamAuthCodes.Columns.CodeHash.EQ(psql.Arg(hash))),
			sm.Where(models.IamAuthCodes.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Limit(1),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return result{}, err
		}
		if len(rows) == 0 {
			return result{}, domain.ErrInvalidToken
		}
		row := rows[0]
		if row.Consumed {
			return result{}, domain.ErrInvalidToken
		}
		if !row.ExpiresAt.IsZero() && row.ExpiresAt.Before(nowUTC()) {
			return result{}, domain.ErrInvalidToken
		}
		var data oauthExchangeCodeData
		if err := unmarshal(row.Data, &data); err != nil {
			return result{}, err
		}
		if data.Session == nil {
			return result{}, domain.ErrInvalidToken
		}
		if data.CodeChallenge != "" {
			if oauthPKCEChallengeS256(cmd.CodeVerifier) != data.CodeChallenge {
				return result{}, domain.ErrInvalidToken
			}
		}
		// Mark consumed (single-use) before handing back the session.
		consumed := true
		if err := row.Update(ctx, a.db.Bobx(), &models.IamAuthCodeSetter{Consumed: &consumed}); err != nil {
			return result{}, err
		}
		acct, err := a.loadAccount(ctx, cmd.ProjectID, row.UserID.GetOrZero())
		if err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oauth.social.exchanged",
			ProjectID:   cmd.ProjectID,
			AggregateID: data.Session.ID,
			Payload:     data.Session,
		}); err != nil {
			return result{}, err
		}
		return result{acct: acct, sess: data.Session}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acct, res.sess, nil
}

// oauthPKCEChallengeS256 derives the S256 PKCE code_challenge from a verifier:
// base64url(sha256(verifier)) without padding (RFC 7636).
func oauthPKCEChallengeS256(verifier string) string {
	if verifier == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// StartLogin builds the provider authorize URL for a browser redirect, looking
// up the provider client_id / authorize endpoint from iam_providers and
// appending state, the optional PKCE challenge, prompt and login_hint.
func (a *pgOAuthSocial) StartLogin(ctx context.Context, cmd domain.OAuthSocialStartCmd) (string, error) {
	if cmd.ProjectID == "" || cmd.Provider == "" {
		return "", domain.ErrBadRequest
	}
	cfg, d, err := a.loadOAuthConfig(ctx, cmd.ProjectID, cmd.Provider, cmd.RedirectTo)
	if err != nil {
		return "", err
	}
	// Persist the CSRF state bound to a validated redirect (anti-CSRF + closes
	// open redirect: only the stored, validated target is used at callback).
	redirect := oauthSafeRedirect(cmd.RedirectTo, d.RedirectURL)
	if err := a.storeState(ctx, cmd.ProjectID, cmd.Provider, cmd.State, redirect, ""); err != nil {
		return "", err
	}
	opts := a.authCodeOpts(cmd.CodeChallenge, cmd.Prompt, cmd.LoginHint)
	return cfg.AuthCodeURL(cmd.State, opts...), nil
}

// authCodeOpts assembles the AuthCodeURL options shared by StartLogin and
// StartLink: an S256 PKCE challenge, an explicit prompt and an OIDC login_hint.
func (a *pgOAuthSocial) authCodeOpts(codeChallenge, prompt, loginHint string) []oauth2.AuthCodeOption {
	var opts []oauth2.AuthCodeOption
	if codeChallenge != "" {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}
	if prompt != "" {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", prompt))
	}
	if loginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", loginHint))
	}
	return opts
}

// CompleteLoginRedirect handles the provider callback and returns the product
// redirect URL plus an optional Set-Cookie value (cookie mode).
func (a *pgOAuthSocial) CompleteLoginRedirect(ctx context.Context, cmd domain.OAuthSocialCallbackCmd) (domain.OAuthSocialCallbackResult, error) {
	if cmd.Error != "" {
		return domain.OAuthSocialCallbackResult{}, domain.ErrProviderError.WithMessage(cmd.Error)
	}
	if cmd.ProjectID == "" || cmd.Provider == "" || cmd.Code == "" {
		return domain.OAuthSocialCallbackResult{}, domain.ErrBadRequest
	}

	// Verify + consume the CSRF state BEFORE exchanging the code; the stored,
	// validated redirect is used (never the request's raw redirect_to).
	redirect, _, err := a.consumeState(ctx, cmd.ProjectID, cmd.Provider, cmd.State)
	if err != nil {
		return domain.OAuthSocialCallbackResult{}, err
	}

	// Exchange the code (PKCE-protected when a verifier is supplied) for the
	// userinfo claims, then resolve/create the account and mint the session.
	cfg, d, err := a.loadOAuthConfig(ctx, cmd.ProjectID, cmd.Provider, cmd.RedirectTo)
	if err != nil {
		return domain.OAuthSocialCallbackResult{}, err
	}
	providerAccountID, email, err := a.oauthExchange(ctx, cfg, d, cmd.Code, cmd.CodeVerifier)
	if err != nil {
		return domain.OAuthSocialCallbackResult{}, err
	}

	sess, err := a.resolveLoginAndMint(ctx, cmd.ProjectID, cmd.Provider, providerAccountID, email)
	if err != nil {
		return domain.OAuthSocialCallbackResult{}, err
	}

	// Persist a single-use exchange code mapping to the minted session so the SPA
	// can complete sign-in via Exchange (token mode); cookie mode uses SetCookie.
	code, err := a.storeExchangeCode(ctx, cmd.ProjectID, sess)
	if err != nil {
		return domain.OAuthSocialCallbackResult{}, err
	}

	if redirect == "" {
		redirect = d.RedirectURL
	}
	redirect = oauthAppendCode(redirect, code)
	return domain.OAuthSocialCallbackResult{
		RedirectURL: redirect,
		SetCookie:   sessionCookieHeader(sess.AccessToken, oauthSocialAccessTTL),
	}, nil
}

// oauthAppendCode appends ?code=<code> (or &code=<code>) to a redirect URL,
// URL-encoding the code. A blank redirect is returned unchanged.
func oauthAppendCode(redirect, code string) string {
	if redirect == "" {
		return redirect
	}
	sep := "?"
	if strings.Contains(redirect, "?") {
		sep = "&"
	}
	return redirect + sep + "code=" + url.QueryEscape(code)
}

// storeExchangeCode mints a single-use exchange code and persists it in
// iam_auth_codes (sha256(code) -> minted session JSON, user_id = account id,
// project-scoped, expires_at = now+5m). The opaque plaintext code is returned
// for the ?code= redirect; only its hash is stored. Mirrors federation's
// fedProvisionAndStoreCode without re-provisioning (the session is already
// minted by resolveLoginAndMint).
func (a *pgOAuthSocial) storeExchangeCode(ctx context.Context, projectID string, sess *domain.Session) (string, error) {
	code, err := fedRandomToken()
	if err != nil {
		return "", err
	}
	err = a.db.withTx(ctx, func(ctx context.Context) error {
		raw, err := marshal(oauthExchangeCodeData{Session: sess})
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		uid := null.From(sess.AccountID)
		setter := &models.IamAuthCodeSetter{
			ID:        ptr(newUUID()),
			ProjectID: &projectID,
			CodeHash:  ptr(fedHashToken(code)),
			UserID:    &uid,
			ExpiresAt: ptr(nowUTC().Add(oauthSocialExchangeCodeTTL)),
			Data:      &rm,
		}
		if _, err := models.IamAuthCodes.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "oauth.social.exchange_code_issued",
			ProjectID:   projectID,
			AggregateID: sess.AccountID,
			Payload:     sess,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return code, nil
}

// resolveLoginAndMint upserts the identity link for an exchanged provider
// account, resolves/creates the iam_users account and mints an iam_sessions row,
// all inside one serializable tx. Shared by CompleteLogin and the redirect flow.
func (a *pgOAuthSocial) resolveLoginAndMint(ctx context.Context, projectID, provider, providerAccountID, email string) (*domain.Session, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Session, error) {
		ident, err := a.findIdentity(ctx, projectID, provider, providerAccountID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		var acct *domain.Account
		if errors.Is(err, domain.ErrNotFound) {
			acct, err = a.createSocialAccount(ctx, projectID, email)
			if err != nil {
				return nil, err
			}
			if err := a.insertIdentity(ctx, &domain.Identity{
				ID:                newUUID(),
				Type:              "oauth",
				Provider:          provider,
				ProviderAccountID: providerAccountID,
				Email:             email,
			}, projectID, acct.ID); err != nil {
				return nil, err
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "identity.linked",
				ProjectID:   projectID,
				AggregateID: acct.ID,
				Payload:     acct,
			}); err != nil {
				return nil, err
			}
		} else {
			acct, err = a.loadAccount(ctx, projectID, ident.UserID)
			if err != nil {
				return nil, err
			}
		}
		sess, err := a.mintSession(ctx, acct)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "session.created",
			ProjectID:   acct.ProjectID,
			AggregateID: sess.ID,
			Payload:     sess,
		}); err != nil {
			return nil, err
		}
		return sess, nil
	})
}

// StartLink builds the provider authorize URL for an account-link flow. The
// authenticated AccountID is carried by the caller's signed `state`.
func (a *pgOAuthSocial) StartLink(ctx context.Context, cmd domain.OAuthSocialLinkStartCmd) (string, error) {
	if cmd.AccountID == "" || cmd.Provider == "" {
		return "", domain.ErrBadRequest
	}
	projectID := cmd.ProjectID
	if projectID == "" {
		// Fall back to the account's tenant when the caller did not supply it.
		row, err := models.FindIamUser(ctx, a.db.Bobx(), cmd.AccountID)
		if err != nil {
			return "", translatePgErr("user", err)
		}
		projectID = row.ProjectID
	}
	cfg, d, err := a.loadOAuthConfig(ctx, projectID, cmd.Provider, cmd.RedirectTo)
	if err != nil {
		return "", err
	}
	// Bind the CSRF state to the validated redirect AND the linking account.
	redirect := oauthSafeRedirect(cmd.RedirectTo, d.RedirectURL)
	if err := a.storeState(ctx, projectID, cmd.Provider, cmd.State, redirect, cmd.AccountID); err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(cmd.State), nil
}

// CompleteLink handles the link callback: it exchanges the code for the provider
// account and attaches the identity to the authenticated AccountID, then returns
// the product redirect URL. The signed `state` carrying the AccountID is
// validated by the caller; AccountID is taken from the principal here.
func (a *pgOAuthSocial) CompleteLink(ctx context.Context, cmd domain.OAuthSocialLinkCallbackCmd) (string, error) {
	if cmd.Error != "" {
		return "", domain.ErrProviderError.WithMessage(cmd.Error)
	}
	if cmd.AccountID == "" || cmd.Provider == "" || cmd.Code == "" {
		return "", domain.ErrBadRequest
	}

	// The account row carries the tenant; resolve it before the upstream exchange.
	row, err := models.FindIamUser(ctx, a.db.Bobx(), cmd.AccountID)
	if err != nil {
		return "", translatePgErr("user", err)
	}
	projectID := row.ProjectID
	if cmd.ProjectID != "" && cmd.ProjectID != projectID { // tenant boundary
		return "", domain.ErrForbidden
	}

	// Verify + consume the CSRF state; it must have been started by THIS account.
	stateRedirect, stateAccount, err := a.consumeState(ctx, projectID, cmd.Provider, cmd.State)
	if err != nil {
		return "", err
	}
	if stateAccount != "" && stateAccount != cmd.AccountID {
		return "", domain.ErrForbidden
	}

	cfg, d, err := a.loadOAuthConfig(ctx, projectID, cmd.Provider, cmd.RedirectTo)
	if err != nil {
		return "", err
	}
	providerAccountID, email, err := a.oauthExchange(ctx, cfg, d, cmd.Code, cmd.CodeVerifier)
	if err != nil {
		return "", err
	}

	err = a.db.withTx(ctx, func(ctx context.Context) error {
		if existing, err := a.findIdentity(ctx, projectID, cmd.Provider, providerAccountID); err == nil {
			if existing.UserID == cmd.AccountID {
				return domain.ErrAlreadyLinked
			}
			return domain.ErrIdentityExists
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if err := a.insertIdentity(ctx, &domain.Identity{
			ID:                newUUID(),
			Type:              "oauth",
			Provider:          cmd.Provider,
			ProviderAccountID: providerAccountID,
			Email:             email,
		}, projectID, cmd.AccountID); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "identity.linked",
			ProjectID:   projectID,
			AggregateID: cmd.AccountID,
			Payload:     map[string]any{"account_id": cmd.AccountID, "provider": cmd.Provider, "project_id": projectID},
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	redirect := stateRedirect
	if redirect == "" {
		redirect = d.RedirectURL
	}
	return redirect, nil
}

// ===== local helpers (oauth-prefixed where they touch package scope) =====

// loadOAuthConfig loads the enabled iam_providers row for (project, provider,
// kind=oauth) and assembles the golang.org/x/oauth2 Config from its data
// envelope. redirectOverride, when non-empty, wins over the persisted
// RedirectURL (the callback URL is request/deployment specific). A missing
// provider maps to ErrNotFound; a provider lacking client/endpoint config maps
// to ErrProviderError.
func (a *pgOAuthSocial) loadOAuthConfig(ctx context.Context, projectID, provider, redirectOverride string) (*oauth2.Config, *oauthProviderData, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Provider.EQ(psql.Arg(provider))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg("oauth"))),
		sm.Where(models.IamProviders.Columns.Enabled.EQ(psql.Arg(true))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, domain.ErrNotFound
	}
	var raw oauthProviderData
	if err := unmarshal(rows[0].Data, &raw); err != nil {
		return nil, nil, err
	}
	d := raw.resolved()
	if d.ClientID == "" || d.AuthURL == "" || d.TokenURL == "" {
		return nil, nil, domain.ErrProviderError.WithMessage("oauth provider misconfigured: missing client_id/auth_url/token_url")
	}
	redirect := d.RedirectURL
	if redirectOverride != "" {
		redirect = redirectOverride
	}
	cfg := &oauth2.Config{
		ClientID:     d.ClientID,
		ClientSecret: d.ClientSecret,
		RedirectURL:  redirect,
		Scopes:       d.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  d.AuthURL,
			TokenURL: d.TokenURL,
		},
	}
	return cfg, &d, nil
}

// oauthExchange swaps the authorization code for a token (golang.org/x/oauth2's
// Exchange) and fetches the userinfo endpoint with that token to resolve the
// external account id + email. The optional codeVerifier carries the PKCE
// verifier paired with the StartLogin challenge. Upstream/transport failures map
// to ErrProviderError.
func (a *pgOAuthSocial) oauthExchange(ctx context.Context, cfg *oauth2.Config, d *oauthProviderData, code, codeVerifier string) (providerAccountID, email string, err error) {
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.VerifierOption(codeVerifier))
	}
	tok, err := cfg.Exchange(ctx, code, opts...)
	if err != nil {
		return "", "", domain.ErrProviderError.WithMessage("oauth code exchange failed")
	}
	if d.UserInfoURL == "" {
		return "", "", domain.ErrProviderError.WithMessage("oauth provider misconfigured: missing userinfo_url")
	}
	info, err := a.fetchUserInfo(ctx, cfg, tok, d.UserInfoURL)
	if err != nil {
		return "", "", err
	}
	providerAccountID = info.externalID()
	if providerAccountID == "" {
		return "", "", domain.ErrProviderError.WithMessage("oauth userinfo missing subject id")
	}
	return providerAccountID, info.Email, nil
}

// fetchUserInfo GETs the provider userinfo endpoint using the exchanged token
// (cfg.Client attaches the bearer token and auto-refreshes) and decodes the
// claims. Non-2xx / transport / decode failures map to ErrProviderError.
func (a *pgOAuthSocial) fetchUserInfo(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token, userInfoURL string) (oauthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return oauthUserInfo{}, domain.ErrProviderError.WithMessage("oauth userinfo request build failed")
	}
	resp, err := cfg.Client(ctx, tok).Do(req)
	if err != nil {
		return oauthUserInfo{}, domain.ErrProviderError.WithMessage("oauth userinfo fetch failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return oauthUserInfo{}, domain.ErrProviderError.WithMessage("oauth userinfo read failed")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthUserInfo{}, domain.ErrProviderError.WithMessage("oauth userinfo returned non-2xx")
	}
	var info oauthUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return oauthUserInfo{}, domain.ErrProviderError.WithMessage("oauth userinfo decode failed")
	}
	return info, nil
}

// findIdentity loads the OAuth identity for a (project, provider, providerAccountID)
// triple, mapping no-rows to domain.ErrNotFound. Tenant-scoped by project_id.
func (a *pgOAuthSocial) findIdentity(ctx context.Context, projectID, provider, providerAccountID string) (*models.IamIdentity, error) {
	rows, err := models.IamIdentities.Query(
		sm.Where(models.IamIdentities.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamIdentities.Columns.Provider.EQ(psql.Arg(provider))),
		sm.Where(models.IamIdentities.Columns.ProviderAccountID.EQ(psql.Arg(providerAccountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	return rows[0], nil
}

// insertIdentity writes a provider link row for an account. Lookup columns carry
// the provider correlation; the domain Identity is stored in the data envelope.
func (a *pgOAuthSocial) insertIdentity(ctx context.Context, ident *domain.Identity, projectID, userID string) error {
	raw, err := marshal(ident)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamIdentitySetter{
		ID:        &ident.ID,
		ProjectID: &projectID,
		UserID:    &userID,
		Type:      ptr(ident.Type),
		Data:      &rm,
	}
	if ident.Provider != "" {
		v := null.From(ident.Provider)
		setter.Provider = &v
	}
	if ident.ProviderAccountID != "" {
		v := null.From(ident.ProviderAccountID)
		setter.ProviderAccountID = &v
	}
	if ident.Email != "" {
		v := null.From(ident.Email)
		setter.Email = &v
	}
	if _, err := models.IamIdentities.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrIdentityExists
		}
		return err
	}
	return nil
}

// createSocialAccount provisions a new account for a first-time social login.
func (a *pgOAuthSocial) createSocialAccount(ctx context.Context, projectID, email string) (*domain.Account, error) {
	acct := &domain.Account{
		ID:            newUUID(),
		ProjectID:     projectID,
		Kind:          "human",
		Status:        "active",
		PrimaryEmail:  email,
		EmailVerified: email != "", // provider-asserted email is treated as verified
		CreatedAt:     nowUTC(),
		UpdatedAt:     nowUTC(),
	}
	raw, err := marshal(acct)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamUserSetter{
		ID:        &acct.ID,
		ProjectID: &acct.ProjectID,
		Kind:      ptr(acct.Kind),
		Status:    ptr(acct.Status),
		Data:      &rm,
	}
	if acct.PrimaryEmail != "" {
		v := null.From(acct.PrimaryEmail)
		setter.PrimaryEmail = &v
	}
	if _, err := models.IamUsers.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrEmailExists
		}
		return nil, err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "user.created",
		ProjectID:   acct.ProjectID,
		AggregateID: acct.ID,
		Payload:     acct,
	}); err != nil {
		return nil, err
	}
	return acct, nil
}

// loadAccount reads the account aggregate from iam_users, tenant-scoped.
func (a *pgOAuthSocial) loadAccount(ctx context.Context, projectID, userID string) (*domain.Account, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), userID)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, domain.ErrUserNotFound
	}
	var acct domain.Account
	if err := unmarshal(row.Data, &acct); err != nil {
		return nil, err
	}
	return &acct, nil
}

// mintSession creates a session row for an account and returns it with a signed
// RS256 JWT access token (minted by the project Signer) plus a refresh token
// signed by the same key.
func (a *pgOAuthSocial) mintSession(ctx context.Context, acct *domain.Account) (*domain.Session, error) {
	sessionID := newUUID()

	signEnv, err := resolveSignEnv(ctx, a.db, acct.ProjectID, oauthSocialDefaultEnv)
	if err != nil {
		return nil, err
	}
	access, err := a.db.Signer().Sign(ctx, acct.ProjectID, signEnv, map[string]any{
		"iss": acct.ProjectID,
		"sub": acct.ID,
		"sid": sessionID,
		"pid": acct.ProjectID,
		"aal": 1,
		"amr": []string{"oauth"},
		"typ": "access",
		"env": signEnv,
	}, oauthSocialAccessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := a.db.Signer().Sign(ctx, acct.ProjectID, signEnv, map[string]any{
		"iss": acct.ProjectID,
		"sub": acct.ID,
		"sid": sessionID,
		"pid": acct.ProjectID,
		"typ": "refresh",
		"env": signEnv,
	}, oauthSocialRefreshTTL)
	if err != nil {
		return nil, err
	}
	sess := &domain.Session{
		ID:           sessionID,
		AccountID:    acct.ID,
		ProjectID:    acct.ProjectID,
		AMR:          []string{"oauth"},
		AAL:          1,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(oauthSocialAccessTTL / timeSecondDur),
		CreatedAt:    nowUTC(),
	}
	raw, err := marshal(sess)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamSessionSetter{
		ID:        &sess.ID,
		ProjectID: &sess.ProjectID,
		UserID:    &sess.AccountID,
		Aal:       ptr(int32(sess.AAL)),
		Data:      &rm,
	}
	if _, err := models.IamSessions.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		return nil, err
	}
	return sess, nil
}
