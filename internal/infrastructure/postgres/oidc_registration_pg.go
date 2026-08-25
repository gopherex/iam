package postgres

// Dynamic client registration (RFC 7591) and client management (RFC 7592).
//
// Registration is NOT open. IAM is multi-tenant, and an open endpoint would let
// anyone create clients inside somebody else's project; the project-admin token
// is the initial access token the RFC allows the server to require, and it is
// also what says which project the new client belongs to.
//
// The client is handed a registration access token once. It is the only
// credential that can read, update or delete that client, and only that one —
// which is why it is stored as a digest beside the client rather than anywhere
// a lookup could confuse it with a different credential.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// oidcRegistrationTokenBytes is the entropy of a registration access token.
const oidcRegistrationTokenBytes = 32

// Client types, as stored. RFC 7591's `application_type` and
// `token_endpoint_auth_method` between them decide which one a registration
// lands on.
const (
	clientTypeWeb    = "web"
	clientTypeSPA    = "spa"
	clientTypeNative = "native"
)

// oidcAuthMethodNone is the RFC 7591 value for a client that authenticates with
// nothing — a public client, protected by PKCE instead of a secret.
const oidcAuthMethodNone = "none"

// oidcAuthMethodBasic is the RFC 7591 default when a client says nothing.
const oidcAuthMethodBasic = "client_secret_basic"

// oidcSecretMetaName labels the secret row a registration mints.
const oidcSecretMetaName = "name"

// RegisterClient creates a client from RFC 7591 metadata.
func (a *pgOIDCGrants) RegisterClient(
	ctx context.Context, cmd domain.OIDCClientRegisterCmd,
) (*domain.OIDCClientRegistration, error) {
	reg := cmd.Registration
	if err := validateRegistrationURIs(reg); err != nil {
		return nil, err
	}

	token, tokenHash, err := adminRandomToken(oidcRegistrationTokenBytes)
	if err != nil {
		return nil, err
	}

	app := &domain.AppClient{
		ID:                      newUUID(),
		ProjectID:               cmd.ProjectID,
		Environment:             adminEnv(cmd.Environment),
		Name:                    reg.ClientName,
		Type:                    oidcRegistrationClientType(reg),
		RedirectURIs:            reg.RedirectURIs,
		PostLogoutRedirectURIs:  reg.PostLogoutRedirectURIs,
		BackchannelLogoutURI:    reg.BackchannelLogoutURI,
		Scopes:                  reg.Scope,
		JWKS:                    reg.JWKS,
		JWKSURI:                 reg.JWKSURI,
		RegistrationTokenHash:   tokenHash,
		TokenEndpointAuthMethod: oidcRegistrationAuthMethod(reg),
	}

	var secret string

	err = a.db.withTx(ctx, func(ctx context.Context) error {
		if err := a.writeRegisteredClient(ctx, app, true); err != nil {
			return err
		}
		// A confidential client needs something to authenticate with. A public
		// one gets nothing on purpose: it cannot keep a secret, which is what
		// PKCE is for.
		if oidcIsConfidentialClient(app.Type) && app.TokenEndpointAuthMethod != oidcAuthMethodPrivateKeyJWT {
			issued, hash, serr := adminRandomToken(oidcRegistrationTokenBytes)
			if serr != nil {
				return serr
			}

			secret = issued

			if err := a.storeClientSecret(ctx, app, hash); err != nil {
				return err
			}
		}

		return a.emitter.Emit(ctx, domain.Event{
			Type:        "app_client.registered",
			ProjectID:   app.ProjectID,
			Environment: app.Environment,
			AggregateID: app.ID,
			Payload:     map[string]any{claimClientID: app.ID, "client_name": app.Name},
		})
	})
	if err != nil {
		return nil, err
	}

	out := oidcRegistrationOf(app, a.db.PublicURL)
	out.ClientSecret = secret
	out.RegistrationAccessToken = token

	return &out, nil
}

// ReadClient returns a registered client's metadata.
func (a *pgOIDCGrants) ReadClient(ctx context.Context, clientID string) (*domain.OIDCClientRegistration, error) {
	_, app, err := a.resolveRegisteredClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	out := oidcRegistrationOf(app, a.db.PublicURL)

	return &out, nil
}

// UpdateClient replaces a registered client's metadata. RFC 7592 §2.2 makes this
// a replacement: a field the caller leaves out is cleared, not kept.
func (a *pgOIDCGrants) UpdateClient(
	ctx context.Context, cmd domain.OIDCClientUpdateCmd,
) (*domain.OIDCClientRegistration, error) {
	row, app, err := a.resolveRegisteredClient(ctx, cmd.ClientID)
	if err != nil {
		return nil, err
	}

	if row.ProjectID != cmd.ProjectID {
		return nil, domain.ErrClientNotFound
	}

	reg := cmd.Registration
	if err := validateRegistrationURIs(reg); err != nil {
		return nil, err
	}

	updated := *app
	updated.Name = reg.ClientName
	updated.RedirectURIs = reg.RedirectURIs
	updated.PostLogoutRedirectURIs = reg.PostLogoutRedirectURIs
	updated.BackchannelLogoutURI = reg.BackchannelLogoutURI
	updated.Scopes = reg.Scope
	updated.JWKS = reg.JWKS
	updated.JWKSURI = reg.JWKSURI

	if method := oidcRegistrationAuthMethod(reg); method != "" {
		updated.TokenEndpointAuthMethod = method
	}

	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		return a.writeRegisteredClient(ctx, &updated, false)
	}); err != nil {
		return nil, err
	}

	out := oidcRegistrationOf(&updated, a.db.PublicURL)

	return &out, nil
}

// DeleteClient removes a registered client.
func (a *pgOIDCGrants) DeleteClient(ctx context.Context, clientID string) error {
	row, _, err := a.resolveRegisteredClient(ctx, clientID)
	if err != nil {
		return err
	}

	return a.db.withTx(ctx, func(ctx context.Context) error {
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}

		return a.emitter.Emit(ctx, domain.Event{
			Type:        "app_client.deleted",
			ProjectID:   row.ProjectID,
			Environment: row.Environment,
			AggregateID: row.ID,
			Payload:     map[string]any{claimClientID: row.ID},
		})
	})
}

// resolveRegisteredClient loads a client that was registered dynamically. A
// client created through the admin API has no registration token and is not
// manageable this way — its owner manages it through the admin API instead.
func (a *pgOIDCGrants) resolveRegisteredClient(
	ctx context.Context, clientID string,
) (*models.IamAppClient, *domain.AppClient, error) {
	row, err := models.FindIamAppClient(ctx, a.db.Bobx(), clientID)
	if err != nil {
		return nil, nil, domain.ErrClientNotFound
	}

	var app domain.AppClient
	if err := unmarshal(row.Data, &app); err != nil {
		return nil, nil, err
	}

	if app.RegistrationTokenHash == "" {
		return nil, nil, domain.ErrClientNotFound
	}

	return row, &app, nil
}

// writeRegisteredClient inserts or updates the client row. Assumes an ambient
// transaction.
func (a *pgOIDCGrants) writeRegisteredClient(ctx context.Context, app *domain.AppClient, create bool) error {
	raw, err := marshal(app)
	if err != nil {
		return err
	}

	data := json.RawMessage(raw)

	if create {
		if _, ierr := models.IamAppClients.Insert(&models.IamAppClientSetter{
			ID:          &app.ID,
			ProjectID:   &app.ProjectID,
			Environment: ptr(app.Environment),
			Name:        ptr(app.Name),
			Type:        ptr(app.Type),
			Data:        &data,
		}).One(ctx, a.db.Bobx()); ierr != nil {
			if isUniqueViolation(ierr) {
				return domain.ErrConflict
			}

			return fmt.Errorf("insert app client: %w", ierr)
		}

		return nil
	}

	row, err := models.FindIamAppClient(ctx, a.db.Bobx(), app.ID)
	if err != nil {
		return domain.ErrClientNotFound
	}

	return row.Update(ctx, a.db.Bobx(), &models.IamAppClientSetter{
		Name:      ptr(app.Name),
		Type:      ptr(app.Type),
		Data:      &data,
		UpdatedAt: ptr(nowUTC()),
	})
}

// storeClientSecret records a freshly issued client secret. Assumes an ambient
// transaction.
func (a *pgOIDCGrants) storeClientSecret(ctx context.Context, app *domain.AppClient, hash string) error {
	meta, err := marshal(map[string]any{oidcSecretMetaName: "registration"})
	if err != nil {
		return err
	}

	data := json.RawMessage(meta)

	if _, err = models.IamAppSecrets.Insert(&models.IamAppSecretSetter{
		ID:        ptr(newUUID()),
		ProjectID: &app.ProjectID,
		AppID:     &app.ID,
		Hash:      &hash,
		Data:      &data,
	}).One(ctx, a.db.Bobx()); err != nil {
		return fmt.Errorf("insert app secret: %w", err)
	}

	return nil
}

// validateRegistrationURIs checks every URI a client submits. They all end up
// somewhere a browser or the server is sent, so a relative or malformed one is
// refused rather than normalized into something nobody meant.
func validateRegistrationURIs(reg domain.OIDCClientRegistration) error {
	if len(reg.RedirectURIs) == 0 {
		return domain.ErrValidation.WithMessage("redirect_uris is required")
	}

	groups := []struct {
		field string
		uris  []string
	}{
		{"redirect_uris", reg.RedirectURIs},
		{"post_logout_redirect_uris", reg.PostLogoutRedirectURIs},
		{"backchannel_logout_uri", nonEmptyURIs(reg.BackchannelLogoutURI)},
		{"jwks_uri", nonEmptyURIs(reg.JWKSURI)},
	}

	for _, group := range groups {
		for _, uri := range group.uris {
			if err := domain.ValidateAbsoluteHTTPURL(group.field, uri); err != nil {
				return err
			}
		}
	}

	return nil
}

// nonEmptyURIs wraps an optional single URI so it validates like a list.
func nonEmptyURIs(uri string) []string {
	if uri == "" {
		return nil
	}

	return []string{uri}
}

// oidcAuthMethodPrivateKeyJWT is the one auth method that needs no secret.
const oidcAuthMethodPrivateKeyJWT = "private_key_jwt"

// oidcRegistrationClientType maps RFC 7591 metadata onto our client types. A
// client that authenticates with nothing is public; `application_type=native`
// says which kind of public client it is.
func oidcRegistrationClientType(reg domain.OIDCClientRegistration) string {
	if reg.ApplicationType == clientTypeNative {
		return clientTypeNative
	}

	if oidcRegistrationAuthMethod(reg) == oidcAuthMethodNone {
		return clientTypeSPA
	}

	return clientTypeWeb
}

// oidcRegistrationAuthMethod defaults to the RFC 7591 default when unset.
func oidcRegistrationAuthMethod(reg domain.OIDCClientRegistration) string {
	if reg.TokenEndpointAuthMethod != "" {
		return reg.TokenEndpointAuthMethod
	}

	return oidcAuthMethodBasic
}

// oidcRegistrationOf renders a stored client back as RFC 7591 metadata.
func oidcRegistrationOf(app *domain.AppClient, publicURL string) domain.OIDCClientRegistration {
	applicationType := clientTypeWeb
	if app.Type == clientTypeNative {
		applicationType = clientTypeNative
	}

	return domain.OIDCClientRegistration{
		ClientID:                app.ID,
		ClientName:              app.Name,
		RedirectURIs:            app.RedirectURIs,
		ApplicationType:         applicationType,
		TokenEndpointAuthMethod: app.TokenEndpointAuthMethod,
		GrantTypes:              []string{string(oidc.GrantTypeCode), string(oidc.GrantTypeRefreshToken)},
		ResponseTypes:           []string{oidcResponseTypeCode},
		Scope:                   app.Scopes,
		JWKS:                    app.JWKS,
		JWKSURI:                 app.JWKSURI,
		PostLogoutRedirectURIs:  app.PostLogoutRedirectURIs,
		BackchannelLogoutURI:    app.BackchannelLogoutURI,
		IssuedAt:                nowUTC(),
		RegistrationClientURI:   strings.TrimRight(publicURL, "/") + "/oauth2/register/" + app.ID,
	}
}
