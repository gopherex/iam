//go:build integration

// Shared HTTP end-to-end test harness. e2eServer wires the SAME handler graph
// and request pipeline that cmd/iam/main.go builds (every feature group over the
// Postgres adapters, security handler, ErrorHandler, X-Environment + CSRF +
// cookie-auth middleware) over the shared testcontainers Postgres (testDB).
//
// Group test files (integration_e2e_<group>_test.go) reuse these helpers and the
// e2eClient request helper; they must NOT redefine any helper declared here.
package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/internal/oas"
	"github.com/gopherex/iam/pkg/api"
)

// e2eEnableSMSProvider seeds an enabled kind=sms provider for projectID so the
// phone-OTP pre-flight (which requires an enabled SMS provider before minting a
// challenge) passes. Delivery itself is captured by the e2e emitter, so the
// config need not point at a real gateway.
func e2eEnableSMSProvider(t *testing.T, ctx context.Context, projectID string) {
	t.Helper()
	id := newUUID()
	kind, prov := "sms", "generic"
	enabled := true
	data := json.RawMessage(`{"type":"generic","config":{"url":"https://sms.example.test/send"}}`)
	if _, err := models.IamProviders.Insert(&models.IamProviderSetter{
		ID:        &id,
		ProjectID: &projectID,
		Kind:      &kind,
		Provider:  &prov,
		Enabled:   &enabled,
		Data:      &data,
	}).One(ctx, testDB.Bobx()); err != nil {
		t.Fatalf("seed sms provider: %v", err)
	}
}

// e2eCaptureEmitter records emitted domain events so tests can recover the
// plaintext OTP code / magic-link token that production only delivers out of
// band (email/SMS). Keyed lookups are by challenge_id from the event payload.
type e2eCaptureEmitter struct {
	mu     sync.Mutex
	events []domain.Event
}

func (c *e2eCaptureEmitter) Emit(_ context.Context, e domain.Event) error {
	c.mu.Lock()
	c.events = append(c.events, e)
	c.mu.Unlock()
	return nil
}

// payloadFor returns the value of field in the most recent event whose payload
// carries the given challenge_id, or "" if none.
func (c *e2eCaptureEmitter) payloadFor(challengeID, field string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.events) - 1; i >= 0; i-- {
		p, ok := c.events[i].Payload.(map[string]any)
		if !ok {
			continue
		}
		if cid, _ := p["challenge_id"].(string); cid == challengeID {
			if v, ok := p[field].(string); ok {
				return v
			}
		}
	}
	return ""
}

// e2eEmitter is the shared capture emitter wired into the harness server. Tests
// run serially (-p 1); lookups key on the unique challenge_id so cross-test
// events don't collide.
var e2eEmitter = &e2eCaptureEmitter{}

// e2eServer assembles the full IAM handler + middleware pipeline over testDB and
// returns a live httptest server (closed via t.Cleanup). Rate-limit / CORS /
// security-header middleware are intentionally omitted: they are not under test
// and would make assertions flaky.
func e2eServer(t *testing.T) *httptest.Server {
	t.Helper()
	// Wrap the capture emitter with auditing so privileged mutations write audit
	// rows (as production does); the capture emitter is still the inner sink, so
	// payloadFor lookups keep working.
	em := Emitter(NewAuditingEmitter(testDB, e2eEmitter))
	platform := NewPgPlatform(testDB)
	coreAuth := NewPgCoreAuth(testDB, em, nil)

	handler := api.New(
		api.WithPlatform(api.NewPlatformService(api.PlatformDeps{
			Config: platform,
			Csrf:   platform,
		})),
		api.WithCoreAuth(api.NewCoreAuthService(api.CoreAuthDeps{
			Accounts: coreAuth,
			Tokens:   coreAuth,
			MFA:      NewPgMFAAccounts(testDB, em, nil),
		})),
		api.WithCoreAuthFlows(api.CoreAuthFlowDeps{
			Flows: NewPgCoreAuthFlows(testDB, em, coreAuth, nil),
		}),
		api.WithPasswordless(api.NewPasswordlessService(api.PasswordlessDeps{
			Accounts: NewPgPasswordlessAccounts(testDB, em, NewConfigReader(testDB, time.Second), coreAuth),
		})),
		api.WithOAuthSocial(api.NewOAuthSocialService(api.OAuthSocialDeps{
			Accounts: NewPgOAuthSocial(testDB, em, nil),
		})),
		api.WithWebAuthn(api.NewWebAuthnService(api.WebAuthnDeps{
			Accounts: NewPgWebAuthnAccounts(testDB, em, nil),
		})),
		api.WithMFA(api.NewMFAService(api.MFADeps{
			Accounts: NewPgMFAAccounts(testDB, em, nil),
		})),
		api.WithAccount(api.NewAccountService(api.AccountDeps{
			Accounts: NewPgAccountStore(testDB, em),
		})),
		api.WithMachineIdentity(api.NewMachineIdentityService(api.MachineIdentityDeps{
			Keys: NewPgMachineIdentities(testDB, em),
		})),
		api.WithFederation(api.NewFederationService(api.FederationDeps{
			Connections: NewPgFederationConnections(testDB, em),
			Runtime:     NewPgFederationRuntime(testDB, em),
			Scim:        NewPgFederationScim(testDB, em),
		})),
		api.WithOIDCProvider(api.NewOIDCProviderService(api.OIDCProviderDeps{
			Grants: NewPgOIDCGrants(testDB, em, NewConfigReader(testDB, time.Second)),
		})),
		api.WithAdmin(api.NewAdminService(api.AdminDeps{
			Users:           NewPgAdminUsers(testDB, em),
			Apps:            NewPgAdminApps(testDB, em),
			ServiceAccounts: NewPgAdminServiceAccounts(testDB, em),
			APIKeys:         NewPgAdminAPIKeys(testDB, em),
			Connections:     NewPgAdminConnections(testDB, em),
			Config:          NewPgAdminConfig(testDB, em),
			Roles:           NewPgRoles(testDB, em),
			Keys:            NewPgAdminKeys(testDB, em),
			AccessRequests:  NewPgAdminAccessRequests(testDB, em),
			Invites:         NewPgInvites(testDB, em),
			Grants:          NewPgOIDCGrants(testDB, em, NewConfigReader(testDB, time.Second)),
			Audit:           NewPgAudit(testDB, em),
			Jobs:            NewPgJobs(testDB, em),
			Hooks:           NewPgHooks(testDB, em),
			Risk:            NewPgRisk(testDB, em),
			TestMode:        NewPgTestMode(testDB, em),
		})),
		api.WithOperator(api.NewOperatorService(api.OperatorDeps{
			Projects: NewPgOperator(testDB, em),
		})),
	)

	auth := NewAuthenticator(testDB, e2eMasterKey)
	srv, err := oas.NewServer(handler, api.NewSecurityHandler(auth), oas.WithErrorHandler(api.ErrorHandler))
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	pipeline := api.RequestMetaMiddleware(
		api.EnvironmentMiddleware(
			api.CSRFMiddleware(platform)(
				api.CookieAuthMiddleware(
					api.SoftAuthMiddleware(auth)(srv)))))
	ts := httptest.NewServer(pipeline)
	t.Cleanup(ts.Close)
	// Point the deployment's public base URL at the live harness server, exactly
	// as an operator points service.http.public_url at their origin. Only then is
	// the advertised issuer a real prefix of the discovery URL tests fetch.
	prevPublicURL := testDB.PublicURL
	testDB.UsePublicURL(ts.URL)
	t.Cleanup(func() { testDB.PublicURL = prevPublicURL })
	return ts
}

// e2eProject creates a fresh empty project and returns its id.
func e2eProject(t *testing.T, ctx context.Context) string {
	t.Helper()
	op := NewPgOperator(testDB, nopEmitter{})
	proj, err := op.CreateProject(ctx, domain.ProjectCmd{Name: "E2E " + newUUID()[:8]})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return proj.ID
}

// e2eProjectAdmin creates a fresh project and mints a project-admin bearer token
// with the given scopes (defaults to admin:ui when none are passed).
func e2eProjectAdmin(t *testing.T, ctx context.Context, scopes ...string) (projectID, token string) {
	t.Helper()
	projectID = e2eProject(t, ctx)
	if len(scopes) == 0 {
		scopes = []string{"admin:ui"}
	}
	op := NewPgOperator(testDB, nopEmitter{})
	tok, _, err := op.MintAdminToken(ctx, domain.OperatorAdminTokenCmd{
		ProjectID: projectID,
		Name:      "e2e",
		Scopes:    scopes,
		ExpiresAt: nowUTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("mint admin token: %v", err)
	}
	return projectID, tok
}

// e2eAppClient registers an app client through the admin API and returns its
// id. redirectURIs become the client's registered redirect_uri allow-list, which
// the authorization endpoint matches exactly.
func e2eAppClient(t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token string, redirectURIs ...string) string {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodPost, fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID),
		map[string]any{
			"name":          "e2e-client",
			"type":          "spa",
			"redirect_uris": redirectURIs,
		},
		e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)
	var resp struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
	}
	e2eDecode(t, r, &resp)
	if resp.App.ID == "" {
		t.Fatalf("create app client: empty id, body: %s", r.Body)
	}
	return resp.App.ID
}

// e2eInteractionCount returns how many authorization interactions exist for a
// client id. Tests assert it stays at zero for requests that must be refused
// before anything is persisted.
func e2eInteractionCount(t *testing.T, ctx context.Context, clientID string) int {
	t.Helper()
	var n int
	if err := testDB.Pool.QueryRow(ctx,
		`SELECT count(*) FROM iam_interactions WHERE client_id = $1`, clientID).Scan(&n); err != nil {
		t.Fatalf("count interactions: %v", err)
	}
	return n
}

// e2eResp is the decoded outcome of an HTTP request: the status code and the raw
// body (decode JSON into a typed struct with e2eDecode).
type e2eResp struct {
	Status int
	Body   []byte
}

// e2eReq performs an HTTP request against the harness server. method/url are
// required; body (if non-nil) is JSON-encoded; headers are applied verbatim
// (e.g. {"Authorization": "Bearer "+tok, "X-Environment": "live"}).
func e2eReq(t *testing.T, ctx context.Context, method, url string, body any, headers map[string]string) e2eResp {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return e2eResp{Status: resp.StatusCode, Body: raw}
}

// e2eBearer is the header map for an admin/bearer-authenticated request.
func e2eBearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token, "X-Environment": "live"}
}

// e2eMasterKey is the operator master key the harness server is built with;
// send it as a bearer token to authenticate operator (cross-project) endpoints.
const e2eMasterKey = "e2e-master-key"

// e2eMaster is the header map for an operator (master-key) authenticated request.
func e2eMaster() map[string]string { return e2eBearer(e2eMasterKey) }

// e2eDecode unmarshals an e2eResp body into dst, failing the test on error.
func e2eDecode(t *testing.T, r e2eResp, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode body (status %d): %v\nbody: %s", r.Status, err, r.Body)
	}
}

// e2eWantStatus asserts the response status, dumping the body on mismatch.
func e2eWantStatus(t *testing.T, r e2eResp, want int) {
	t.Helper()
	if r.Status != want {
		t.Fatalf("status = %d, want %d\nbody: %s", r.Status, want, r.Body)
	}
}
