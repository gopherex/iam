package postgres

// Federation adapter — Postgres persistence for the SSO/SCIM federation ports
// (api.FederationConnections, api.FederationRuntime, api.FederationScim).
//
// Tables it owns:
//   - iam_sso_connections : SSO connection aggregate (saml | oidc).
//   - iam_domains         : verified email domains routed to a connection.
//   - iam_scim_tokens     : SCIM provisioning bearer credentials (hash only).
//   - iam_scim_resources  : SCIM Users / Groups (connection-scoped).
//
// Persistence follows the package gold pattern (reference.go): the domain
// aggregate is marshalled into the `data jsonb` envelope, the typed columns are
// lookup-only, every mutation runs inside db.withTx / withTxRet, reads run on
// db.Bobx(). The tenant boundary is project_id on every query; SCIM resources
// are additionally scoped to their connection_id (a row whose connection/project
// does not match the request is treated as not-found).
//
// Crypto policy: SCIM token secrets and SSO exchange/auth codes are opaque
// tokens minted with crypto/rand; only their sha256 hash is persisted. JWT /
// SAML / OIDC token minting and signature verification are NOT implemented here
// (marked // TODO: sign/verify with signing key) — the runtime legs persist what
// they can and return generated opaque material.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// ===========================================================================
// shared federation helpers
// ===========================================================================

// fedRandomToken mints an opaque, URL-safe random token (crypto/rand). It is the
// plaintext secret handed back to the caller exactly once; only its hash is
// stored.
func fedRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// fedHashToken is the sha256 hex digest persisted in hash columns; the plaintext
// opaque token is never stored.
func fedHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// fedConnectionFromRow rebuilds a domain.Connection from its envelope. The jsonb
// blob is authoritative; lookup columns backfill anything the blob omits.
func fedConnectionFromRow(row *models.IamSsoConnection) (*domain.Connection, error) {
	var c domain.Connection
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &c); err != nil {
			return nil, err
		}
	}
	c.ID = row.ID
	c.ProjectID = row.ProjectID
	if c.Type == "" {
		c.Type = row.Type
	}
	if c.Status == "" {
		c.Status = row.Status
	}
	if c.Name == "" {
		c.Name = row.Name
	}
	if c.ExternalRef == "" {
		if v, ok := row.ExternalRef.Get(); ok {
			c.ExternalRef = v
		}
	}
	return &c, nil
}

// fedDomainFromRow rebuilds a domain.Domain from its envelope.
func fedDomainFromRow(row *models.IamDomain) (*domain.Domain, error) {
	var d domain.Domain
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &d); err != nil {
			return nil, err
		}
	}
	d.ID = row.ID
	d.ProjectID = row.ProjectID
	if d.Domain == "" {
		d.Domain = row.Domain
	}
	if d.Status == "" {
		d.Status = row.Status
	}
	if d.ConnectionID == "" {
		if v, ok := row.ConnectionID.Get(); ok {
			d.ConnectionID = v
		}
	}
	return &d, nil
}

// fedTokenFromRow rebuilds a domain.ScimToken from its envelope (secret excluded;
// only the hash lives in the hash column).
func fedTokenFromRow(row *models.IamScimToken) (*domain.ScimToken, error) {
	var t domain.ScimToken
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &t); err != nil {
			return nil, err
		}
	}
	t.ID = row.ID
	t.ProjectID = row.ProjectID
	t.ConnectionID = row.ConnectionID
	return &t, nil
}

// fedConnSetter builds an insert/update setter from a connection aggregate: the
// lookup columns are projected from the struct, the aggregate is stored whole in
// the jsonb envelope.
func fedConnSetter(c *domain.Connection) (*models.IamSsoConnectionSetter, error) {
	raw, err := marshal(c)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamSsoConnectionSetter{
		ID:        &c.ID,
		ProjectID: &c.ProjectID,
		Type:      ptr(c.Type),
		Status:    ptr(c.Status),
		Name:      ptr(c.Name),
		Data:      &rm,
	}
	if c.ExternalRef != "" {
		v := null.From(c.ExternalRef)
		setter.ExternalRef = &v
	}
	return setter, nil
}

// ===========================================================================
// FederationConnections
// ===========================================================================

// pgFederationConnections persists the SSO connection aggregate plus the domains
// and SCIM tokens bound to it. Every method is scoped to a project (tenant
// boundary); a row whose project_id does not match is treated as not-found.
type pgFederationConnections struct{ db *DB }

// NewPgFederationConnections builds the Postgres-backed FederationConnections adapter.
func NewPgFederationConnections(db *DB) *pgFederationConnections {
	return &pgFederationConnections{db: db}
}

var _ api.FederationConnections = (*pgFederationConnections)(nil)

func (a *pgFederationConnections) CreateConnection(ctx context.Context, cmd domain.ConnectionCmd) (*domain.Connection, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Connection, error) {
		conn := &domain.Connection{
			ID:        newUUID(),
			ProjectID: cmd.ProjectID,
			Type:      cmd.Type,
			Name:      cmd.Name,
			Status:    "active",
			Domains:   cmd.Domains,
		}
		setter, err := fedConnSetter(conn)
		if err != nil {
			return nil, err
		}
		if _, err := models.IamSsoConnections.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		// TODO outbox event: federation.connection.created
		return conn, nil
	})
}

func (a *pgFederationConnections) GetConnection(ctx context.Context, projectID, id string) (*domain.Connection, error) {
	row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), id)
	if err != nil {
		if errors.Is(translatePgErr("connection", err), ErrNotFound) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	if row.ProjectID != projectID { // tenant boundary
		return nil, domain.ErrConnectionNotFound
	}
	return fedConnectionFromRow(row)
}

func (a *pgFederationConnections) ListConnections(ctx context.Context, projectID string) ([]domain.Connection, error) {
	rows, err := models.IamSsoConnections.Query(
		sm.Where(models.IamSsoConnections.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Connection, 0, len(rows))
	for _, row := range rows {
		c, err := fedConnectionFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

func (a *pgFederationConnections) UpdateConnection(ctx context.Context, cmd domain.FederationConnectionUpdateCmd) (*domain.Connection, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Connection, error) {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), cmd.ID)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return nil, domain.ErrConnectionNotFound
			}
			return nil, err
		}
		if row.ProjectID != cmd.ProjectID {
			return nil, domain.ErrConnectionNotFound
		}
		conn, err := fedConnectionFromRow(row)
		if err != nil {
			return nil, err
		}
		fedApplyConnectionPatch(conn, cmd.Patch)
		setter, err := fedConnSetter(conn)
		if err != nil {
			return nil, err
		}
		setter.ID = nil // never re-set the pk on update
		setter.ProjectID = nil
		setter.UpdatedAt = ptr(nowUTC())
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		// TODO outbox event: federation.connection.updated
		return conn, nil
	})
}

// fedApplyConnectionPatch applies the supplied merge-patch fields onto the
// connection aggregate (only keys the caller provided are touched).
func fedApplyConnectionPatch(c *domain.Connection, patch map[string]any) {
	if patch == nil {
		return
	}
	if v, ok := patch["name"].(string); ok {
		c.Name = v
	}
	if v, ok := patch["status"].(string); ok {
		c.Status = v
	}
	if v, ok := patch["type"].(string); ok {
		c.Type = v
	}
	if v, ok := patch["external_ref"].(string); ok {
		c.ExternalRef = v
	}
	if raw, ok := patch["domains"].([]any); ok {
		doms := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				doms = append(doms, s)
			}
		}
		c.Domains = doms
	} else if doms, ok := patch["domains"].([]string); ok {
		c.Domains = doms
	}
}

func (a *pgFederationConnections) DeleteConnection(ctx context.Context, projectID, id string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), id)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return domain.ErrConnectionNotFound
			}
			return err
		}
		if row.ProjectID != projectID {
			return domain.ErrConnectionNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: federation.connection.deleted
		return nil
	})
}

func (a *pgFederationConnections) TestConnection(ctx context.Context, projectID, id string) (string, error) {
	conn, err := a.GetConnection(ctx, projectID, id)
	if err != nil {
		return "", err
	}
	// The SSO test leg drives the provider login flow and validates the round
	// trip; the protocol crypto (AuthnRequest signing / OIDC client assertion)
	// is not implemented here.
	// TODO: sign/verify with signing key — build a real provider test URL.
	return "/v1/sso/" + conn.Type + "/" + conn.ID + "/start?test=1", nil
}

func (a *pgFederationConnections) RotateConnectionCertificate(ctx context.Context, projectID, id string) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), id)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return "", domain.ErrConnectionNotFound
			}
			return "", err
		}
		if row.ProjectID != projectID {
			return "", domain.ErrConnectionNotFound
		}
		conn, err := fedConnectionFromRow(row)
		if err != nil {
			return "", err
		}
		// A real rotation generates a new SAML signing keypair / SP certificate
		// and stores the private key in the signing-key store. We persist a fresh
		// opaque reference and return the (placeholder) public certificate.
		// TODO: sign/verify with signing key — mint a real X.509 keypair.
		certRef, err := fedRandomToken()
		if err != nil {
			return "", err
		}
		conn.ExternalRef = certRef
		setter, err := fedConnSetter(conn)
		if err != nil {
			return "", err
		}
		setter.ID = nil
		setter.ProjectID = nil
		setter.UpdatedAt = ptr(nowUTC())
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return "", err
		}
		// TODO outbox event: federation.connection.certificate_rotated
		return certRef, nil
	})
}

func (a *pgFederationConnections) AddDomain(ctx context.Context, projectID, connectionID, name string) (*domain.Domain, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Domain, error) {
		dom := &domain.Domain{
			ID:           newUUID(),
			ProjectID:    projectID,
			Domain:       name,
			Status:       "pending",
			ConnectionID: connectionID,
		}
		raw, err := marshal(dom)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamDomainSetter{
			ID:        &dom.ID,
			ProjectID: &dom.ProjectID,
			Domain:    ptr(dom.Domain),
			Status:    ptr(dom.Status),
			Data:      &rm,
		}
		if connectionID != "" {
			v := null.From(connectionID)
			setter.ConnectionID = &v
		}
		if _, err := models.IamDomains.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrDomainTaken
			}
			return nil, err
		}
		// TODO outbox event: federation.domain.added
		return dom, nil
	})
}

func (a *pgFederationConnections) VerifyDomain(ctx context.Context, projectID, domainID string) (*domain.Domain, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Domain, error) {
		row, err := models.FindIamDomain(ctx, a.db.Bobx(), domainID)
		if err != nil {
			if errors.Is(translatePgErr("domain", err), ErrNotFound) {
				return nil, domain.ErrDomainNotFound
			}
			return nil, err
		}
		if row.ProjectID != projectID {
			return nil, domain.ErrDomainNotFound
		}
		dom, err := fedDomainFromRow(row)
		if err != nil {
			return nil, err
		}
		// Verification of the DNS challenge is performed out of band; here we
		// flip the persisted state to verified.
		dom.Status = "verified"
		raw, err := marshal(dom)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		verifiedAt := null.From(nowUTC())
		setter := &models.IamDomainSetter{
			Status:     ptr("verified"),
			VerifiedAt: &verifiedAt,
			Data:       &rm,
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		// TODO outbox event: federation.domain.verified
		return dom, nil
	})
}

func (a *pgFederationConnections) ListDomains(ctx context.Context, projectID string) ([]domain.Domain, error) {
	rows, err := models.IamDomains.Query(
		sm.Where(models.IamDomains.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Domain, 0, len(rows))
	for _, row := range rows {
		d, err := fedDomainFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, nil
}

func (a *pgFederationConnections) DeleteDomain(ctx context.Context, projectID, domainID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamDomain(ctx, a.db.Bobx(), domainID)
		if err != nil {
			if errors.Is(translatePgErr("domain", err), ErrNotFound) {
				return domain.ErrDomainNotFound
			}
			return err
		}
		if row.ProjectID != projectID {
			return domain.ErrDomainNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: federation.domain.deleted
		return nil
	})
}

func (a *pgFederationConnections) CreateScimToken(ctx context.Context, cmd domain.FederationScimTokenCmd) (*domain.ScimToken, string, error) {
	type result struct {
		tok    *domain.ScimToken
		secret string
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// Verify the connection exists and is in the requested project before
		// minting a credential scoped to it (tenant boundary).
		connRow, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), cmd.ConnectionID)
		if err != nil {
			if errors.Is(translatePgErr("connection", err), ErrNotFound) {
				return result{}, domain.ErrConnectionNotFound
			}
			return result{}, err
		}
		if connRow.ProjectID != cmd.ProjectID {
			return result{}, domain.ErrConnectionNotFound
		}

		secret, err := fedRandomToken()
		if err != nil {
			return result{}, err
		}
		tok := &domain.ScimToken{
			ID:           newUUID(),
			ProjectID:    cmd.ProjectID,
			ConnectionID: cmd.ConnectionID,
			Name:         cmd.Name,
			ExpiresAt:    cmd.ExpiresAt,
		}
		raw, err := marshal(tok)
		if err != nil {
			return result{}, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimTokenSetter{
			ID:           &tok.ID,
			ProjectID:    &tok.ProjectID,
			ConnectionID: &tok.ConnectionID,
			Hash:         ptr(fedHashToken(secret)), // store only the hash, never plaintext
			Data:         &rm,
		}
		if _, err := models.IamScimTokens.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return result{}, domain.ErrConflict
			}
			return result{}, err
		}
		// TODO outbox event: federation.scim_token.created
		return result{tok: tok, secret: secret}, nil
	})
	if err != nil {
		return nil, "", err
	}
	return res.tok, res.secret, nil
}

func (a *pgFederationConnections) ListScimTokens(ctx context.Context, projectID, connectionID string) ([]domain.ScimToken, error) {
	rows, err := models.IamScimTokens.Query(
		sm.Where(models.IamScimTokens.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamScimTokens.Columns.ConnectionID.EQ(psql.Arg(connectionID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScimToken, 0, len(rows))
	for _, row := range rows {
		t, err := fedTokenFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, nil
}

func (a *pgFederationConnections) DeleteScimToken(ctx context.Context, projectID, connectionID, tokenID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamScimToken(ctx, a.db.Bobx(), tokenID)
		if err != nil {
			if errors.Is(translatePgErr("scim_token", err), ErrNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if row.ProjectID != projectID || row.ConnectionID != connectionID {
			return domain.ErrNotFound
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: federation.scim_token.deleted
		return nil
	})
}

func (a *pgFederationConnections) ResolveConnection(ctx context.Context, projectID, email string) (*domain.Connection, error) {
	// Route the email's domain to its verified connection: find the domain row
	// for the (project, host), then load the connection it binds to.
	host := fedEmailDomain(email)
	if host == "" {
		return nil, domain.ErrConnectionNotFound
	}
	domRow, err := models.IamDomains.Query(
		sm.Where(models.IamDomains.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamDomains.Columns.Domain.EQ(psql.Arg(host))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if errors.Is(translatePgErr("domain", err), ErrNotFound) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	connID, ok := domRow.ConnectionID.Get()
	if !ok || connID == "" {
		return nil, domain.ErrConnectionNotFound
	}
	return a.GetConnection(ctx, projectID, connID)
}

// fedEmailDomain extracts the host portion of an email address (lowercased by
// the caller's expectation; stored domains are matched verbatim).
func fedEmailDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return ""
}

// ===========================================================================
// FederationRuntime — SSO outbound/inbound legs + exchange
// ===========================================================================

// pgFederationRuntime drives the OIDC / SAML authentication legs. The protocol
// crypto (AuthnRequest signing, OIDC code exchange, assertion signature
// verification, JWT minting) is NOT implemented here: each such line is marked
// // TODO: sign/verify with signing key. Exchange resolves a short-lived opaque
// exchange code to the authenticated account + session.
type pgFederationRuntime struct{ db *DB }

// NewPgFederationRuntime builds the Postgres-backed FederationRuntime adapter.
func NewPgFederationRuntime(db *DB) *pgFederationRuntime {
	return &pgFederationRuntime{db: db}
}

var _ api.FederationRuntime = (*pgFederationRuntime)(nil)

// fedConnByID loads a connection by id without a project filter (the runtime
// legs are reached by connection id from the public surface).
func (a *pgFederationRuntime) fedConnByID(ctx context.Context, connectionID string) (*models.IamSsoConnection, error) {
	row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), connectionID)
	if err != nil {
		if errors.Is(translatePgErr("connection", err), ErrNotFound) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	return row, nil
}

func (a *pgFederationRuntime) OidcStart(ctx context.Context, cmd domain.FederationSsoStartCmd) (*domain.FederationSsoRedirect, error) {
	if _, err := a.fedConnByID(ctx, cmd.ConnectionID); err != nil {
		return nil, err
	}
	// Build the OIDC authorization-code redirect with PKCE + state. The PKCE
	// verifier / nonce persistence and provider metadata discovery are not
	// implemented here.
	// TODO: sign/verify with signing key — build a real OIDC authorize URL.
	state, err := fedRandomToken()
	if err != nil {
		return nil, err
	}
	_ = state
	return &domain.FederationSsoRedirect{
		URL: "/v1/sso/oidc/" + cmd.ConnectionID + "/start?redirect_to=" + cmd.RedirectTo,
	}, nil
}

func (a *pgFederationRuntime) OidcCallback(ctx context.Context, cmd domain.FederationSsoCallbackCmd) (*domain.FederationSsoRedirect, error) {
	if _, err := a.fedConnByID(ctx, cmd.ConnectionID); err != nil {
		return nil, err
	}
	// Exchange the authorization code at the provider token endpoint, verify the
	// id_token signature, provision/link the account and mint a session. None of
	// the provider crypto is implemented here.
	// TODO: sign/verify with signing key — verify id_token + exchange code.
	// TODO outbox event: federation.sso.oidc_callback
	exchangeCode, err := fedRandomToken()
	if err != nil {
		return nil, err
	}
	return &domain.FederationSsoRedirect{
		URL: "/v1/sso/exchange?code=" + exchangeCode,
	}, nil
}

func (a *pgFederationRuntime) SamlLogin(ctx context.Context, cmd domain.FederationSsoStartCmd) (*domain.FederationSsoRedirect, error) {
	if _, err := a.fedConnByID(ctx, cmd.ConnectionID); err != nil {
		return nil, err
	}
	// Build and sign the SAML AuthnRequest and redirect to the IdP SSO endpoint.
	// TODO: sign/verify with signing key — build + sign the SAML AuthnRequest.
	return &domain.FederationSsoRedirect{
		URL: "/v1/sso/saml/" + cmd.ConnectionID + "/login?redirect_to=" + cmd.RedirectTo,
	}, nil
}

func (a *pgFederationRuntime) SamlAcs(ctx context.Context, cmd domain.FederationSamlAcsCmd) (*domain.FederationSsoRedirect, error) {
	if _, err := a.fedConnByID(ctx, cmd.ConnectionID); err != nil {
		return nil, err
	}
	// Validate the SAML Response signature, extract the assertion, provision/link
	// the account and mint a session. The XML-DSig verification is not
	// implemented here.
	// TODO: sign/verify with signing key — verify the SAML assertion signature.
	// TODO outbox event: federation.sso.saml_acs
	exchangeCode, err := fedRandomToken()
	if err != nil {
		return nil, err
	}
	return &domain.FederationSsoRedirect{
		URL: "/v1/sso/exchange?code=" + exchangeCode,
	}, nil
}

func (a *pgFederationRuntime) SamlSlo(ctx context.Context, connectionID string) (*domain.FederationSsoRedirect, error) {
	if _, err := a.fedConnByID(ctx, connectionID); err != nil {
		return nil, err
	}
	// Build and sign the SAML LogoutRequest / LogoutResponse.
	// TODO: sign/verify with signing key — build + sign the SAML LogoutRequest.
	// TODO outbox event: federation.sso.saml_slo
	return &domain.FederationSsoRedirect{
		URL: "/v1/sso/saml/" + connectionID + "/slo",
	}, nil
}

func (a *pgFederationRuntime) SamlMetadata(ctx context.Context, connectionID string) ([]byte, error) {
	row, err := a.fedConnByID(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	// Render the SP metadata XML (entityID, ACS URL, signing certificate). The
	// embedded signing certificate comes from the connection's signing key.
	// TODO: sign/verify with signing key — embed the real SP signing certificate.
	xml := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="` + row.ID + `">` +
		`<SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">` +
		`<AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" ` +
		`Location="/v1/sso/saml/` + row.ID + `/acs" index="0"/>` +
		`</SPSSODescriptor></EntityDescriptor>`
	return []byte(xml), nil
}

func (a *pgFederationRuntime) Exchange(ctx context.Context, projectID, code string) (*domain.Account, *domain.Session, error) {
	type result struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// The exchange code is an opaque single-use token persisted (hashed) by
		// the callback leg and resolved here to the user it authenticated. The
		// code store is not part of this adapter's tables; until it lands we
		// cannot resolve the principal.
		// TODO: sign/verify with signing key — resolve the exchange code to a user.
		hash := fedHashToken(code)
		_ = hash

		// Resolution of the exchange code to an account is not yet wired (no code
		// table in this adapter); treat any code as unresolved.
		return result{}, domain.ErrInvalidCredentials

		// Once the code resolves to (projectID, userID), the session is minted:
		//   acc := loaded account
		//   sess := &domain.Session{ID: newUUID(), AccountID: acc.ID, ProjectID: projectID, ...}
		//   persist into iam_sessions; mint the access/refresh tokens.
		//   // TODO: sign/verify with signing key — mint the access/id token JWT.
		//   // TODO outbox event: federation.sso.exchanged
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// ===========================================================================
// FederationScim — connection-scoped SCIM v2 provisioning
// ===========================================================================

// pgFederationScim persists SCIM Users and Groups as free-form attribute maps in
// iam_scim_resources. Every operation is scoped to the connection it is invoked
// for (and, transitively, the connection's project): a resource that does not
// belong to the requested connection is treated as not-found. The SCIM schema
// semantics (id assignment, meta block, list envelope) are owned here.
type pgFederationScim struct{ db *DB }

// NewPgFederationScim builds the Postgres-backed FederationScim adapter.
func NewPgFederationScim(db *DB) *pgFederationScim {
	return &pgFederationScim{db: db}
}

var _ api.FederationScim = (*pgFederationScim)(nil)

const (
	scimResourceTypeUser  = "User"
	scimResourceTypeGroup = "Group"
)

// fedScimProjectForConnection resolves the project a connection belongs to so
// SCIM rows can be persisted with the right tenant boundary. A missing
// connection is a not-found.
func (a *pgFederationScim) fedScimProjectForConnection(ctx context.Context, connectionID string) (string, error) {
	row, err := models.FindIamSsoConnection(ctx, a.db.Bobx(), connectionID)
	if err != nil {
		if errors.Is(translatePgErr("connection", err), ErrNotFound) {
			return "", domain.ErrConnectionNotFound
		}
		return "", err
	}
	return row.ProjectID, nil
}

// fedScimResourceFromRow decodes the stored SCIM attribute map and stamps the
// canonical id / meta so the wire representation is self-consistent.
func fedScimResourceFromRow(row *models.IamScimResource) (map[string]any, error) {
	attrs := map[string]any{}
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &attrs); err != nil {
			return nil, err
		}
	}
	attrs["id"] = row.ID
	if v, ok := row.ExternalID.Get(); ok && v != "" {
		attrs["externalId"] = v
	}
	meta, _ := attrs["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["resourceType"] = row.ResourceType
	meta["created"] = row.CreatedAt.UTC().Format(time.RFC3339)
	meta["lastModified"] = row.UpdatedAt.UTC().Format(time.RFC3339)
	attrs["meta"] = meta
	return attrs, nil
}

// fedScimExternalID pulls the SCIM externalId out of an attribute map, if present.
func fedScimExternalID(attrs map[string]any) string {
	if v, ok := attrs["externalId"].(string); ok {
		return v
	}
	return ""
}

// fedScimListEnvelope wraps resources in the SCIM v2 ListResponse envelope.
func fedScimListEnvelope(resources []map[string]any, startIndex, count int) map[string]any {
	if startIndex <= 0 {
		startIndex = 1
	}
	return map[string]any{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": len(resources),
		"startIndex":   startIndex,
		"itemsPerPage": len(resources),
		"Resources":    resources,
	}
}

// fedScimList is the shared list path for Users and Groups (scoped by connection
// + resource type).
func (a *pgFederationScim) fedScimList(ctx context.Context, q domain.FederationScimListQuery, resourceType string) (map[string]any, error) {
	rows, err := models.IamScimResources.Query(
		sm.Where(models.IamScimResources.Columns.ConnectionID.EQ(psql.Arg(q.ConnectionID))),
		sm.Where(models.IamScimResources.Columns.ResourceType.EQ(psql.Arg(resourceType))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	resources := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m, err := fedScimResourceFromRow(row)
		if err != nil {
			return nil, err
		}
		resources = append(resources, m)
	}
	return fedScimListEnvelope(resources, q.StartIndex, q.Count), nil
}

// fedScimGet loads a single resource scoped to its connection + type.
func (a *pgFederationScim) fedScimGet(ctx context.Context, connectionID, resourceID, resourceType string) (*models.IamScimResource, error) {
	row, err := models.FindIamScimResource(ctx, a.db.Bobx(), resourceID)
	if err != nil {
		if errors.Is(translatePgErr("scim_resource", err), ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if row.ConnectionID != connectionID || row.ResourceType != resourceType { // tenant boundary
		return nil, domain.ErrNotFound
	}
	return row, nil
}

// fedScimCreate inserts a new SCIM resource on a connection (id minted here).
func (a *pgFederationScim) fedScimCreate(ctx context.Context, cmd domain.FederationScimWriteCmd, resourceType string) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		projectID, err := a.fedScimProjectForConnection(ctx, cmd.ConnectionID)
		if err != nil {
			return nil, err
		}
		id := newUUID()
		attrs := map[string]any{}
		for k, v := range cmd.Attributes {
			attrs[k] = v
		}
		attrs["id"] = id

		raw, err := marshal(attrs)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimResourceSetter{
			ID:           &id,
			ProjectID:    &projectID,
			ConnectionID: &cmd.ConnectionID,
			ResourceType: ptr(resourceType),
			Data:         &rm,
		}
		if ext := fedScimExternalID(cmd.Attributes); ext != "" {
			v := null.From(ext)
			setter.ExternalID = &v
		}
		if _, err := models.IamScimResources.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		// TODO outbox event: federation.scim.resource_created
		row, err := models.FindIamScimResource(ctx, a.db.Bobx(), id)
		if err != nil {
			return nil, err
		}
		return fedScimResourceFromRow(row)
	})
}

// fedScimReplace overwrites a resource's attributes wholesale (PUT semantics).
func (a *pgFederationScim) fedScimReplace(ctx context.Context, cmd domain.FederationScimWriteCmd, resourceType string) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		row, err := a.fedScimGet(ctx, cmd.ConnectionID, cmd.ResourceID, resourceType)
		if err != nil {
			return nil, err
		}
		attrs := map[string]any{}
		for k, v := range cmd.Attributes {
			attrs[k] = v
		}
		attrs["id"] = row.ID

		raw, err := marshal(attrs)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimResourceSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}
		if ext := fedScimExternalID(cmd.Attributes); ext != "" {
			v := null.From(ext)
			setter.ExternalID = &v
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		// TODO outbox event: federation.scim.resource_replaced
		return fedScimResourceFromRow(row)
	})
}

// fedScimPatch applies a shallow attribute merge (the supplied keys overwrite the
// stored ones; this adapter does not interpret SCIM PATCH op/path grammar).
func (a *pgFederationScim) fedScimPatch(ctx context.Context, cmd domain.FederationScimPatchCmd, resourceType string) (map[string]any, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (map[string]any, error) {
		row, err := a.fedScimGet(ctx, cmd.ConnectionID, cmd.ResourceID, resourceType)
		if err != nil {
			return nil, err
		}
		attrs := map[string]any{}
		if len(row.Data) > 0 {
			if err := unmarshal(row.Data, &attrs); err != nil {
				return nil, err
			}
		}
		for k, v := range cmd.Patch {
			attrs[k] = v
		}
		attrs["id"] = row.ID

		raw, err := marshal(attrs)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamScimResourceSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}
		if ext := fedScimExternalID(attrs); ext != "" {
			v := null.From(ext)
			setter.ExternalID = &v
		}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		// TODO outbox event: federation.scim.resource_patched
		return fedScimResourceFromRow(row)
	})
}

// fedScimDelete removes a resource scoped to its connection + type.
func (a *pgFederationScim) fedScimDelete(ctx context.Context, connectionID, resourceID, resourceType string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := a.fedScimGet(ctx, connectionID, resourceID, resourceType)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: federation.scim.resource_deleted
		return nil
	})
}

// --- Users ---

func (a *pgFederationScim) ListUsers(ctx context.Context, q domain.FederationScimListQuery) (map[string]any, error) {
	return a.fedScimList(ctx, q, scimResourceTypeUser)
}

func (a *pgFederationScim) GetUser(ctx context.Context, connectionID, scimUserID string) (map[string]any, error) {
	row, err := a.fedScimGet(ctx, connectionID, scimUserID, scimResourceTypeUser)
	if err != nil {
		return nil, err
	}
	return fedScimResourceFromRow(row)
}

func (a *pgFederationScim) CreateUser(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimCreate(ctx, cmd, scimResourceTypeUser)
}

func (a *pgFederationScim) ReplaceUser(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimReplace(ctx, cmd, scimResourceTypeUser)
}

func (a *pgFederationScim) PatchUser(ctx context.Context, cmd domain.FederationScimPatchCmd) (map[string]any, error) {
	return a.fedScimPatch(ctx, cmd, scimResourceTypeUser)
}

func (a *pgFederationScim) DeleteUser(ctx context.Context, connectionID, scimUserID string) error {
	return a.fedScimDelete(ctx, connectionID, scimUserID, scimResourceTypeUser)
}

// --- Groups ---

func (a *pgFederationScim) ListGroups(ctx context.Context, q domain.FederationScimListQuery) (map[string]any, error) {
	return a.fedScimList(ctx, q, scimResourceTypeGroup)
}

func (a *pgFederationScim) GetGroup(ctx context.Context, connectionID, groupID string) (map[string]any, error) {
	row, err := a.fedScimGet(ctx, connectionID, groupID, scimResourceTypeGroup)
	if err != nil {
		return nil, err
	}
	return fedScimResourceFromRow(row)
}

func (a *pgFederationScim) CreateGroup(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimCreate(ctx, cmd, scimResourceTypeGroup)
}

func (a *pgFederationScim) ReplaceGroup(ctx context.Context, cmd domain.FederationScimWriteCmd) (map[string]any, error) {
	return a.fedScimReplace(ctx, cmd, scimResourceTypeGroup)
}

func (a *pgFederationScim) PatchGroup(ctx context.Context, cmd domain.FederationScimPatchCmd) (map[string]any, error) {
	return a.fedScimPatch(ctx, cmd, scimResourceTypeGroup)
}

func (a *pgFederationScim) DeleteGroup(ctx context.Context, connectionID, groupID string) error {
	return a.fedScimDelete(ctx, connectionID, groupID, scimResourceTypeGroup)
}
