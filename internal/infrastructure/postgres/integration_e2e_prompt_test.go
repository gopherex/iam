//go:build integration

package postgres

// integration_e2e_prompt_test.go — how an authorization request treats a session
// that already exists: silent single sign-on, prompt, max_age, the client's
// scope allow-list and response_mode.
//
// /oauth2/authorize used to ignore prompt entirely, so prompt=none rendered a
// login page inside the hidden iframe an SPA uses for silent renewal, and an
// already-signed-in user was asked for a password once per relying party.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// promptFixture is a browser with a live IAM session plus a registered client.
type promptFixture struct {
	browser     *browser
	baseURL     string
	projectID   string
	adminToken  string
	clientID    string
	redirectURI string
	email       string
}

const promptRedirectURI = "https://rp.example.com/cb"

// promptSetup registers a client, a user, and signs the browser in.
func promptSetup(t *testing.T, ctx context.Context) (promptFixture, func(query string) (int, string)) {
	t.Helper()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	rApp := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID),
		map[string]any{
			"name":          "Prompt RP",
			"type":          "spa",
			"redirect_uris": []string{promptRedirectURI},
		}, e2eBearer(adminToken))
	e2eWantStatus(t, rApp, http.StatusCreated)

	var app struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
	}
	e2eDecode(t, rApp, &app)

	email := fmt.Sprintf("prompt-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)
	tenant := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

	status, body := b.do(t, ctx, http.MethodPost, "/v1/auth/flows", map[string]any{
		"kind":        "signin",
		"email":       email,
		"password":    "Sup3rStr0ng!Pass",
		"cookie_mode": true,
	}, tenant)
	if status != http.StatusOK {
		t.Fatalf("sign in: status %d, body %s", status, body)
	}

	f := promptFixture{
		browser:     b,
		baseURL:     ts.URL,
		projectID:   projectID,
		adminToken:  adminToken,
		clientID:    app.App.ID,
		redirectURI: promptRedirectURI,
		email:       email,
	}

	// authorize runs a request with the browser's session attached and returns
	// the status plus the Location it was sent to.
	authorize := func(extra string) (int, string) {
		u := fmt.Sprintf(
			"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s"+
				"&code_challenge=%s&code_challenge_method=S256&state=st%s",
			f.clientID, url.QueryEscape(f.redirectURI), url.QueryEscape("openid"),
			url.QueryEscape(challengeFor(pkceVerifier)), extra)
		status, _ := b.do(t, ctx, http.MethodGet, u, nil, nil)
		return status, b.lastLocation
	}

	return f, authorize
}

// grantConsent walks the interaction once so the user has a remembered grant.
func (f promptFixture) grantConsent(t *testing.T, ctx context.Context, interactionID string) {
	t.Helper()
	status, body := f.browser.do(t, ctx, http.MethodPost,
		"/v1/oauth/interaction/"+interactionID+"/login", map[string]any{},
		f.browser.csrf(t, ctx, f.projectID))
	if status != http.StatusOK {
		t.Fatalf("interaction login: status %d, body %s", status, body)
	}

	status, body = f.browser.do(t, ctx, http.MethodPost,
		"/v1/oauth/interaction/"+interactionID+"/consent",
		map[string]any{"granted_scopes": []string{"openid"}, "remember": true},
		f.browser.csrf(t, ctx, f.projectID))
	if status != http.StatusOK {
		t.Fatalf("consent: status %d, body %s", status, body)
	}
}

// TestAuthorizeSilentSSO: once the user has granted the scopes, a later
// authorization request from the same browser is answered with a code — no
// login screen, no consent screen. This is the difference between single
// sign-on and signing on once per application.
func TestAuthorizeSilentSSO(t *testing.T) {
	ctx := context.Background()
	f, authorize := promptSetup(t, ctx)

	status, location := authorize("")
	if status != http.StatusFound || !strings.HasPrefix(location, "/oauth/interaction/") {
		t.Fatalf("first authorize = %d %q, want an interaction", status, location)
	}

	f.grantConsent(t, ctx, strings.TrimPrefix(location, "/oauth/interaction/"))

	status, location = authorize("")
	if status != http.StatusFound {
		t.Fatalf("second authorize = %d, want 302", status)
	}

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse %q: %v", location, err)
	}

	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != f.redirectURI {
		t.Fatalf("second authorize went to %q, want a code at %q", location, f.redirectURI)
	}
	if parsed.Query().Get("code") == "" {
		t.Fatalf("no code in %q", location)
	}
	if parsed.Query().Get("state") != "st" {
		t.Fatalf("state missing from %q", location)
	}
}

// TestAuthorizePromptNone covers silent renewal: the case an SPA runs in a
// hidden iframe, where showing UI is the same as failing.
func TestAuthorizePromptNone(t *testing.T) {
	ctx := context.Background()
	f, authorize := promptSetup(t, ctx)

	// Signed in, but nothing granted yet.
	status, location := authorize("&prompt=none")
	if status != http.StatusFound {
		t.Fatalf("prompt=none = %d, want 302", status)
	}

	parsed, _ := url.Parse(location)
	if got := parsed.Query().Get("error"); got != "consent_required" {
		t.Fatalf("error = %q, want consent_required (location %q)", got, location)
	}

	// Grant, then it succeeds without any UI.
	status, location = authorize("")
	f.grantConsent(t, ctx, strings.TrimPrefix(location, "/oauth/interaction/"))

	status, location = authorize("&prompt=none")
	if status != http.StatusFound {
		t.Fatalf("prompt=none after consent = %d, want 302", status)
	}

	parsed, _ = url.Parse(location)
	if parsed.Query().Get("code") == "" {
		t.Fatalf("prompt=none produced no code after consent: %q", location)
	}
}

// TestAuthorizePromptNoneWithoutSession: no session means login_required, and
// still no UI.
func TestAuthorizePromptNoneWithoutSession(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	rApp := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID),
		map[string]any{
			"name":          "Anonymous RP",
			"type":          "spa",
			"redirect_uris": []string{promptRedirectURI},
		}, e2eBearer(adminToken))
	e2eWantStatus(t, rApp, http.StatusCreated)

	var app struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
	}
	e2eDecode(t, rApp, &app)

	b := newBrowser(t, ts) // no session
	u := fmt.Sprintf(
		"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=openid"+
			"&code_challenge=%s&code_challenge_method=S256&prompt=none",
		app.App.ID, url.QueryEscape(promptRedirectURI), url.QueryEscape(challengeFor(pkceVerifier)))

	status, _ := b.do(t, ctx, http.MethodGet, u, nil, nil)
	if status != http.StatusFound {
		t.Fatalf("status = %d, want 302", status)
	}

	parsed, _ := url.Parse(b.lastLocation)
	if got := parsed.Query().Get("error"); got != "login_required" {
		t.Fatalf("error = %q, want login_required (location %q)", got, b.lastLocation)
	}
	if strings.HasPrefix(b.lastLocation, "/oauth/interaction/") {
		t.Fatal("prompt=none created an interaction; it must never show UI")
	}
}

// TestAuthorizePromptLoginForcesInteraction: a client that wants fresh
// authentication gets it, even though the session would have satisfied the
// request silently.
func TestAuthorizePromptLoginForcesInteraction(t *testing.T) {
	ctx := context.Background()
	f, authorize := promptSetup(t, ctx)

	_, location := authorize("")
	f.grantConsent(t, ctx, strings.TrimPrefix(location, "/oauth/interaction/"))

	// Without prompt this would be silent.
	_, location = authorize("&prompt=login")
	if !strings.HasPrefix(location, "/oauth/interaction/") {
		t.Fatalf("prompt=login went to %q, want an interaction", location)
	}
}

// TestAuthorizePromptConsentForcesTheScreen: a granted scope is re-confirmed
// when the client insists.
func TestAuthorizePromptConsentForcesTheScreen(t *testing.T) {
	ctx := context.Background()
	f, authorize := promptSetup(t, ctx)

	_, location := authorize("")
	f.grantConsent(t, ctx, strings.TrimPrefix(location, "/oauth/interaction/"))

	_, location = authorize("&prompt=consent")
	if !strings.HasPrefix(location, "/oauth/interaction/") {
		t.Fatalf("prompt=consent went to %q, want an interaction", location)
	}
}

// TestAuthorizePromptNoneIsExclusive: asking for no UI and for a login screen at
// once is a contradiction, not a preference.
func TestAuthorizePromptNoneIsExclusive(t *testing.T) {
	ctx := context.Background()
	_, authorize := promptSetup(t, ctx)

	_, location := authorize("&prompt=none%20login")

	parsed, _ := url.Parse(location)
	if got := parsed.Query().Get("error"); got != "invalid_request" {
		t.Fatalf("error = %q, want invalid_request (location %q)", got, location)
	}
}

// TestAuthorizeMaxAgeForcesReauthentication: a session older than the client is
// willing to rely on is not good enough, however valid it still is.
func TestAuthorizeMaxAgeForcesReauthentication(t *testing.T) {
	ctx := context.Background()
	f, authorize := promptSetup(t, ctx)

	_, location := authorize("")
	f.grantConsent(t, ctx, strings.TrimPrefix(location, "/oauth/interaction/"))

	// Age the session past what max_age=1 will accept.
	if _, err := testDB.Pool.Exec(ctx,
		`UPDATE iam_sessions SET created_at = now() - interval '1 hour' WHERE project_id = $1`,
		f.projectID); err != nil {
		t.Fatalf("age the session: %v", err)
	}

	_, location = authorize("&max_age=1")
	if !strings.HasPrefix(location, "/oauth/interaction/") {
		t.Fatalf("max_age=1 went to %q, want re-authentication", location)
	}

	// And with prompt=none it cannot re-authenticate, so it must say so.
	_, location = authorize("&max_age=1&prompt=none")

	parsed, _ := url.Parse(location)
	if got := parsed.Query().Get("error"); got != "login_required" {
		t.Fatalf("error = %q, want login_required (location %q)", got, location)
	}
}

// TestAuthorizeScopeAllowList: a client is refused a scope the project did not
// allow it, loudly rather than by quietly narrowing the token.
func TestAuthorizeScopeAllowList(t *testing.T) {
	ctx := context.Background()
	f, _ := promptSetup(t, ctx)

	r := e2eReq(t, ctx, http.MethodPatch,
		fmt.Sprintf("%s/v1/projects/%s/admin/apps/%s", f.baseURL, f.projectID, f.clientID),
		map[string]any{"scopes": []string{"openid"}}, e2eBearer(f.adminToken))
	e2eWantStatus(t, r, http.StatusOK)

	u := fmt.Sprintf(
		"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s"+
			"&code_challenge=%s&code_challenge_method=S256",
		f.clientID, url.QueryEscape(f.redirectURI), url.QueryEscape("openid groups"),
		url.QueryEscape(challengeFor(pkceVerifier)))

	status, _ := f.browser.do(t, ctx, http.MethodGet, u, nil, nil)
	if status != http.StatusFound {
		t.Fatalf("status = %d, want 302", status)
	}

	parsed, _ := url.Parse(f.browser.lastLocation)
	if got := parsed.Query().Get("error"); got != "invalid_scope" {
		t.Fatalf("error = %q, want invalid_scope (location %q)", got, f.browser.lastLocation)
	}
}

// TestAuthorizeResponseModeFragment: the code comes back in the fragment, which
// keeps it out of the Referer header and the redirect target's logs.
func TestAuthorizeResponseModeFragment(t *testing.T) {
	ctx := context.Background()
	f, authorize := promptSetup(t, ctx)

	_, location := authorize("")
	f.grantConsent(t, ctx, strings.TrimPrefix(location, "/oauth/interaction/"))

	_, location = authorize("&response_mode=fragment")
	if !strings.Contains(location, "#") {
		t.Fatalf("response_mode=fragment returned %q, want a fragment", location)
	}
	if strings.Contains(strings.SplitN(location, "#", 2)[0], "code=") {
		t.Fatalf("the code is in the query despite response_mode=fragment: %q", location)
	}

	frag := strings.SplitN(location, "#", 2)[1]
	values, err := url.ParseQuery(frag)
	if err != nil {
		t.Fatalf("parse fragment: %v", err)
	}
	if values.Get("code") == "" {
		t.Fatalf("no code in fragment %q", frag)
	}
}

// TestAuthorizeResponseNamesTheIssuer covers RFC 9207: every authorization
// response says which provider produced it. A client registered with more than
// one OP can otherwise be steered into redeeming an honest code at the wrong
// provider — the mix-up attack the parameter exists to stop.
func TestAuthorizeResponseNamesTheIssuer(t *testing.T) {
	ctx := context.Background()
	f, authorize := promptSetup(t, ctx)

	wantIssuer := fmt.Sprintf("%s/p/%s/e/live", f.baseURL, f.projectID)

	// Error response.
	_, location := authorize("&prompt=none%20login")

	parsed, _ := url.Parse(location)
	if got := parsed.Query().Get("iss"); got != wantIssuer {
		t.Fatalf("error response iss = %q, want %q", got, wantIssuer)
	}

	// Success response.
	_, location = authorize("")
	f.grantConsent(t, ctx, strings.TrimPrefix(location, "/oauth/interaction/"))

	_, location = authorize("")

	parsed, _ = url.Parse(location)
	if got := parsed.Query().Get("iss"); got != wantIssuer {
		t.Fatalf("success response iss = %q, want %q", got, wantIssuer)
	}
	if parsed.Query().Get("code") == "" {
		t.Fatalf("no code alongside iss: %q", location)
	}
}

// TestDiscoveryAdvertisesIssParameter: a client only checks `iss` if the
// metadata tells it to.
func TestDiscoveryAdvertisesIssParameter(t *testing.T) {
	ctx := context.Background()
	f, _ := promptSetup(t, ctx)

	r := e2eReq(t, ctx, http.MethodGet,
		fmt.Sprintf("%s/p/%s/e/live/.well-known/openid-configuration", f.baseURL, f.projectID), nil, nil)
	e2eWantStatus(t, r, http.StatusOK)

	var doc map[string]any
	e2eDecode(t, r, &doc)

	if supported, _ := doc["authorization_response_iss_parameter_supported"].(bool); !supported {
		t.Fatalf("authorization_response_iss_parameter_supported = %v, want true", doc["authorization_response_iss_parameter_supported"])
	}
}
