package postgres

// Client-credentials authentication for service accounts.
//
// A service account is an OAuth client of the client-credentials kind: it has an
// id, it holds secrets, and it asks for a token in its own name. Modeling it as
// anything else means inventing a second, non-standard way to do what RFC 6749
// §4.4 already describes — and every integration library already implements.
//
// So the service-account id IS the client id, its secrets are client secrets,
// and `POST /oauth2/token` with `grant_type=client_credentials` is how it gets a
// token. Nothing here is specific to the token endpoint; the same verification
// backs `client_secret_basic` on any client-authenticated call.

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// serviceTokenTTL is how long a client-credentials token lives. Short, because
// the holder can always ask for another one — it authenticates with a secret it
// keeps, not with a user's presence.
const serviceTokenTTL = time.Hour

// Token-response keys and the values a client-credentials response carries.
const (
	respAccessToken    = "access_token"
	respTokenType      = "token_type"
	respExpiresIn      = "expires_in"
	respScope          = "scope"
	tokenTypeBearer    = "Bearer"
	tokenTypeService   = "service"
	eventServiceMinted = "service_account.token.minted"
)

// resolvedServiceAccount is a verified service-account credential.
type resolvedServiceAccount struct {
	ID          string
	ProjectID   string
	Environment string
	Scopes      []string
}

// loadServiceAccountCredential resolves an already-authenticated service
// account. It still refuses a disabled one: authentication happened at the
// transport, but whether the account may act is decided here.
func loadServiceAccountCredential(ctx context.Context, db *DB, clientID string) (*resolvedServiceAccount, error) {
	row, err := models.FindIamServiceAccount(ctx, db.Bobx(), clientID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	var env saEnvelope
	if err := unmarshal(row.Data, &env); err != nil {
		return nil, domain.ErrUnauthorized
	}

	if env.Disabled {
		return nil, domain.ErrUnauthorized
	}

	return resolvedFrom(ctx, db, row, &env)
}

// resolvedFrom builds the resolved credential shared by both entry points.
func resolvedFrom(
	ctx context.Context, db *DB, row *models.IamServiceAccount, env *saEnvelope,
) (*resolvedServiceAccount, error) {
	// Service accounts are not environment-scoped rows; they act in the project's
	// effective default environment.
	environment, err := effectiveEnv(ctx, db, row.ProjectID, authDefaultEnv)
	if err != nil {
		return nil, err
	}

	return &resolvedServiceAccount{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		Environment: environment,
		Scopes:      env.Scopes,
	}, nil
}

// verifyServiceAccountSecret checks a service account's client secret.
//
// The lookup is by id rather than by secret hash: a secret belongs to exactly
// one account, and matching the presented secret against that account's own
// secrets keeps a hash collision across tenants from ever being a question.
func verifyServiceAccountSecret(
	ctx context.Context, db *DB, clientID, secret string,
) (*resolvedServiceAccount, error) {
	if clientID == "" || secret == "" {
		return nil, domain.ErrUnauthorized
	}

	row, err := models.FindIamServiceAccount(ctx, db.Bobx(), clientID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	var env saEnvelope
	if err := unmarshal(row.Data, &env); err != nil {
		return nil, domain.ErrUnauthorized
	}

	if env.Disabled {
		return nil, domain.ErrUnauthorized
	}

	if !serviceAccountSecretMatches(env.Secrets, secret) {
		return nil, domain.ErrUnauthorized
	}

	return resolvedFrom(ctx, db, row, &env)
}

// serviceAccountSecretMatches compares the presented secret against every live
// secret the account holds, without short-circuiting: an account may hold
// several at once so one can be rotated in before the other is deleted, and the
// reply time must not say which of them was close.
func serviceAccountSecretMatches(secrets []saSecret, presented string) bool {
	given := machineIDHash(presented)
	now := nowUTC()
	matched := 0

	for _, s := range secrets {
		if s.Revoked {
			continue
		}

		if s.ExpiresAt != nil && !s.ExpiresAt.IsZero() && now.After(*s.ExpiresAt) {
			continue
		}

		matched |= subtle.ConstantTimeCompare([]byte(given), []byte(s.Hash))
	}

	return matched == 1
}

// narrowServiceScopes resolves the scopes a client-credentials token carries.
//
// A requested scope must be one the account already holds: the request says how
// little to grant, never how much. Asking for something outside the grant is
// refused rather than trimmed, so a misconfigured caller fails loudly instead of
// receiving a token that quietly does less than it asked for.
func narrowServiceScopes(granted []string, requested string) ([]string, error) {
	wanted := splitScopes(requested)
	if len(wanted) == 0 {
		return granted, nil
	}

	allowed := make(map[string]struct{}, len(granted))
	for _, s := range granted {
		allowed[s] = struct{}{}
	}

	for _, s := range wanted {
		if _, ok := allowed[s]; !ok {
			return nil, domain.ErrValidation.WithDetails(map[string]any{
				respScope: s, "granted": granted,
			}).WithMessage("invalid_scope: the service account was not granted " + s)
		}
	}

	return wanted, nil
}

// tokenClientCredentialsGrant mints an access token for a verified service
// account (RFC 6749 §4.4). There is no refresh token and no id_token: there is
// no user to represent, and a client that can authenticate can simply ask again.
func (a *pgOIDCGrants) tokenClientCredentialsGrant(
	ctx context.Context, cmd domain.OIDCTokenCmd,
) (map[string]any, error) {
	clientID := firstNonEmpty(cmd.AuthenticatedClientID, cmd.ClientID)

	// client_secret_basic proves the client at the transport, and leaves no
	// secret in the body to prove it with a second time. Re-verifying would
	// reject exactly the method most OAuth clients use by default.
	var (
		account *resolvedServiceAccount
		err     error
	)

	if cmd.AuthenticatedClientID != "" && cmd.AuthenticatedClientID == clientID {
		account, err = loadServiceAccountCredential(ctx, a.db, clientID)
	} else {
		account, err = verifyServiceAccountSecret(ctx, a.db, clientID, cmd.ClientSecret)
	}

	if err != nil {
		return nil, err
	}

	scopes, err := narrowServiceScopes(account.Scopes, cmd.Scope)
	if err != nil {
		return nil, err
	}

	issuer := oidcIssuer(a.db.PublicURL, account.ProjectID, account.Environment)

	token, err := a.db.Signer().Sign(ctx, account.ProjectID, account.Environment, map[string]any{
		claimIssuer:      issuer,
		claimSubject:     account.ID,
		claimProjectID:   account.ProjectID,
		claimEnvironment: account.Environment,
		claimClientID:    account.ID,
		claimTokenType:   tokenTypeService,
		claimScope:       scopes,
	}, serviceTokenTTL)
	if err != nil {
		return nil, err
	}

	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        eventServiceMinted,
		ProjectID:   account.ProjectID,
		Environment: account.Environment,
		AggregateID: account.ID,
		Payload:     map[string]any{claimClientID: account.ID, payloadScopes: scopes},
	}); err != nil {
		return nil, err
	}

	return map[string]any{
		respAccessToken: token,
		respTokenType:   tokenTypeBearer,
		respExpiresIn:   int(serviceTokenTTL / time.Second),
		respScope:       joinScopes(scopes),
	}, nil
}
