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
// not implemented here (no HTTP client / IdP secrets in this layer); those legs
// carry a TODO. We persist the identity link, account and session, and return a
// generated opaque session token (real JWT minting is a signing-key TODO).

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// pgOAuthSocial is the Postgres-backed OAuthSocialAccounts adapter.
type pgOAuthSocial struct{ db *DB }

// NewPgOAuthSocial builds the OAuthSocial adapter over the connection bundle.
func NewPgOAuthSocial(db *DB) *pgOAuthSocial { return &pgOAuthSocial{db: db} }

var _ api.OAuthSocialAccounts = (*pgOAuthSocial)(nil)

// oauthProviderData is the iam_providers `data` jsonb envelope for an OAuth
// provider: the display name and requested scopes. The id/project/kind/provider
// live in the lookup columns.
type oauthProviderData struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// oauthRandToken returns a 32-byte crypto/rand opaque token, hex-encoded. Used
// for the session access/refresh tokens we hand back until real JWT minting
// lands.
func oauthRandToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
		var d oauthProviderData
		if err := unmarshal(row.Data, &d); err != nil {
			return nil, err
		}
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
// The upstream code-exchange (swap `code` at the provider token endpoint, fetch
// the userinfo claims) is not implemented here. Once we have the provider
// account id + email we upsert the iam_identities link, resolve/create the
// iam_users account, and mint an iam_sessions row.
func (a *pgOAuthSocial) CompleteLogin(ctx context.Context, projectID, provider, code string) (*domain.Account, *domain.Session, error) {
	if projectID == "" || provider == "" || code == "" {
		return nil, nil, domain.ErrBadRequest
	}
	type result struct {
		acct *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// TODO: exchange `code` at the provider token endpoint and fetch the
		// userinfo claims (provider account id + email). No IdP HTTP client in
		// this layer yet.
		providerAccountID := code // placeholder identity correlation until exchange lands
		email := ""

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
			// TODO outbox event: identity.linked
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
		// TODO outbox event: session.created
		return result{acct: acct, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acct, res.sess, nil
}

// Link attaches a provider identity to an already authenticated account.
//
// As with CompleteLogin the upstream code-exchange is a TODO; we persist the
// link to iam_identities for the account.
func (a *pgOAuthSocial) Link(ctx context.Context, accountID, provider, code string) error {
	if accountID == "" || provider == "" || code == "" {
		return domain.ErrBadRequest
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		// The account row carries the tenant; we link inside its project.
		row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			return translatePgErr("user", err)
		}
		projectID := row.ProjectID

		// TODO: exchange `code` at the provider token endpoint for the provider
		// account id + email.
		providerAccountID := code // placeholder until exchange lands

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
		}, projectID, accountID); err != nil {
			return err
		}
		// TODO outbox event: identity.linked
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
		// TODO outbox event: identity.unlinked
		return nil
	})
}

// Exchange swaps a one-time OAuth authorization code (PKCE-protected) for an
// account + session, scoped to the command's project.
func (a *pgOAuthSocial) Exchange(ctx context.Context, cmd domain.OAuthSocialExchangeCmd) (*domain.Account, *domain.Session, error) {
	if cmd.ProjectID == "" || cmd.Code == "" {
		return nil, nil, domain.ErrBadRequest
	}
	type result struct {
		acct *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// TODO: validate the one-time code (and PKCE CodeVerifier) against the
		// stored authorization grant, then resolve the linked identity. No grant
		// store wired here yet; we treat the code as the identity correlator.
		providerAccountID := cmd.Code

		ident, err := a.findIdentityAnyProvider(ctx, cmd.ProjectID, providerAccountID)
		if err != nil {
			return result{}, err
		}
		acct, err := a.loadAccount(ctx, cmd.ProjectID, ident.UserID)
		if err != nil {
			return result{}, err
		}
		sess, err := a.mintSession(ctx, acct)
		if err != nil {
			return result{}, err
		}
		// TODO outbox event: session.created
		return result{acct: acct, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acct, res.sess, nil
}

// StartLogin builds the provider authorize URL for a browser redirect.
func (a *pgOAuthSocial) StartLogin(ctx context.Context, cmd domain.OAuthSocialStartCmd) (string, error) {
	// TODO: build the provider authorize URL — look up the provider client_id /
	// authorize endpoint from iam_providers, append state/PKCE/prompt/login_hint
	// and the callback redirect_uri. No IdP metadata client in this layer yet.
	return "", domain.ErrServiceUnavailable.WithMessage("oauth authorize URL building not implemented")
}

// CompleteLoginRedirect handles the provider callback and returns the product
// redirect URL plus an optional Set-Cookie value (cookie mode).
func (a *pgOAuthSocial) CompleteLoginRedirect(ctx context.Context, cmd domain.OAuthSocialCallbackCmd) (domain.OAuthSocialCallbackResult, error) {
	if cmd.Error != "" {
		return domain.OAuthSocialCallbackResult{}, domain.ErrProviderError.WithMessage(cmd.Error)
	}
	// TODO: validate `state` against the stored PAR/interaction, run
	// CompleteLogin to mint the session, then build the product redirect URL and
	// (cookie mode) the Set-Cookie header. State store and redirect resolution
	// are not wired in this layer yet.
	return domain.OAuthSocialCallbackResult{}, domain.ErrServiceUnavailable.WithMessage("oauth callback redirect not implemented")
}

// StartLink builds the provider authorize URL for an account-link flow.
func (a *pgOAuthSocial) StartLink(ctx context.Context, cmd domain.OAuthSocialLinkStartCmd) (string, error) {
	// TODO: build the provider authorize URL for the link flow (carries the
	// authenticated AccountID in signed state). Same IdP-metadata dependency as
	// StartLogin.
	return "", domain.ErrServiceUnavailable.WithMessage("oauth link authorize URL building not implemented")
}

// CompleteLink handles the link callback and returns the product redirect URL.
func (a *pgOAuthSocial) CompleteLink(ctx context.Context, cmd domain.OAuthSocialLinkCallbackCmd) (string, error) {
	// TODO: validate signed state to recover the AccountID, run Link to attach
	// the identity, then return the product redirect URL. State store and
	// redirect resolution not wired in this layer yet.
	return "", domain.ErrServiceUnavailable.WithMessage("oauth link callback not implemented")
}

// ===== local helpers (oauth-prefixed where they touch package scope) =====

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

// findIdentityAnyProvider loads an OAuth identity by provider account id within a
// project, regardless of provider. Used by Exchange where the grant carries the
// linked identity but the provider is implicit.
func (a *pgOAuthSocial) findIdentityAnyProvider(ctx context.Context, projectID, providerAccountID string) (*models.IamIdentity, error) {
	rows, err := models.IamIdentities.Query(
		sm.Where(models.IamIdentities.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamIdentities.Columns.Type.EQ(psql.Arg("oauth"))),
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
	// TODO outbox event: user.created
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

// mintSession creates a session row for an account and returns it with an opaque
// access/refresh token. Real JWT access/id token minting is a signing-key TODO.
func (a *pgOAuthSocial) mintSession(ctx context.Context, acct *domain.Account) (*domain.Session, error) {
	access, err := oauthRandToken()
	if err != nil {
		return nil, err
	}
	refresh, err := oauthRandToken()
	if err != nil {
		return nil, err
	}
	// TODO: sign/verify with signing key — mint a real JWT access/id token here
	// instead of the opaque random token.
	sess := &domain.Session{
		ID:           newUUID(),
		AccountID:    acct.ID,
		ProjectID:    acct.ProjectID,
		AMR:          []string{"oauth"},
		AAL:          1,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    3600,
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
