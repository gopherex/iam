//go:build integration

package postgres

// integration_e2e_gooidc_test.go — compatibility with the OIDC client library
// real relying parties actually use.
//
// go-oidc is what ArgoCD, oauth2-proxy and kube-oidc-proxy discover and verify
// with. It was rejecting this provider outright with "oidc: issuer did not
// match", because the discovery document advertised a RELATIVE issuer. Driving
// the library itself here is the only way to keep that honest: an assertion we
// write ourselves can agree with a document no standards client would accept.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	oidclib "github.com/coreos/go-oidc/v3/oidc"
)

// TestE2EGoOIDCDiscoversAndVerifies runs a standards client end to end: discover
// the provider, complete an authorization code flow through the hosted screens,
// and verify the returned id_token against the published JWKS.
func TestE2EGoOIDCDiscoversAndVerifies(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	const redirectURI = "https://rp.example.com/callback"

	rApp := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID),
		map[string]any{
			"name":          "Relying Party",
			"type":          "web",
			"redirect_uris": []string{redirectURI},
		}, e2eBearer(adminToken))
	e2eWantStatus(t, rApp, http.StatusCreated)

	var app struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
	}
	e2eDecode(t, rApp, &app)

	rSecret := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/apps/%s/secrets", ts.URL, projectID, app.App.ID),
		map[string]any{"name": "e2e"}, e2eBearer(adminToken))
	e2eWantStatus(t, rSecret, http.StatusCreated)

	var secret struct {
		ClientSecret string `json:"client_secret"`
	}
	e2eDecode(t, rSecret, &secret)

	// 1. Discovery. go-oidc fetches <issuer>/.well-known/openid-configuration and
	//    refuses the provider unless the document's `issuer` is byte-identical to
	//    the URL it asked for. This is the check that used to fail.
	issuer := fmt.Sprintf("%s/p/%s/e/live", ts.URL, projectID)

	provider, err := oidclib.NewProvider(ctx, issuer)
	if err != nil {
		t.Fatalf("go-oidc discovery against %s: %v", issuer, err)
	}

	var claims struct {
		AuthorizationEndpoint string   `json:"authorization_endpoint"`
		TokenEndpoint         string   `json:"token_endpoint"`
		ScopesSupported       []string `json:"scopes_supported"`
	}
	if err := provider.Claims(&claims); err != nil {
		t.Fatalf("read discovery claims: %v", err)
	}
	if !strings.HasPrefix(claims.AuthorizationEndpoint, "http") {
		t.Fatalf("authorization_endpoint = %q is not absolute", claims.AuthorizationEndpoint)
	}

	// 2. Sign in and consent through the hosted screens.
	email := fmt.Sprintf("rp-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)
	tenant := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

	challenge := challengeFor(pkceVerifier)
	authorizeURL := fmt.Sprintf(
		"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s"+
			"&code_challenge=%s&code_challenge_method=S256",
		app.App.ID, url.QueryEscape(redirectURI), url.QueryEscape("openid email"),
		url.QueryEscape(challenge))

	status, body := b.do(t, ctx, http.MethodGet, authorizeURL, nil, nil)
	if status != http.StatusFound {
		t.Fatalf("authorize: status %d, body %s", status, body)
	}

	interactionID := strings.TrimPrefix(b.lastLocation, "/oauth/interaction/")

	status, body = b.do(t, ctx, http.MethodPost, "/v1/auth/flows", map[string]any{
		"kind":        "signin",
		"email":       email,
		"password":    "Sup3rStr0ng!Pass",
		"cookie_mode": true,
	}, tenant)
	if status != http.StatusOK {
		t.Fatalf("sign in: status %d, body %s", status, body)
	}

	status, body = b.do(t, ctx, http.MethodPost,
		"/v1/oauth/interaction/"+interactionID+"/login", map[string]any{},
		b.csrf(t, ctx, projectID))
	if status != http.StatusOK {
		t.Fatalf("interaction login: status %d, body %s", status, body)
	}

	status, body = b.do(t, ctx, http.MethodPost,
		"/v1/oauth/interaction/"+interactionID+"/consent",
		map[string]any{"granted_scopes": []string{"openid", "email"}, "remember": true},
		b.csrf(t, ctx, projectID))
	if status != http.StatusOK {
		t.Fatalf("consent: status %d, body %s", status, body)
	}

	var consent struct {
		RedirectTo string `json:"redirect_to"`
	}
	if err := json.Unmarshal(body, &consent); err != nil {
		t.Fatalf("decode consent: %v", err)
	}

	parsed, err := url.Parse(consent.RedirectTo)
	if err != nil {
		t.Fatalf("parse consent redirect: %v", err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in %q", consent.RedirectTo)
	}

	// 3. Exchange with client_secret_basic, the method go-oidc's oauth2 config
	//    uses by default.
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkceVerifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claims.TokenEndpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(app.App.ID, secret.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token exchange: status %d, body %s", resp.StatusCode, raw)
	}

	var tokens struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(raw, &tokens); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}
	if tokens.IDToken == "" {
		t.Fatalf("no id_token in %s", raw)
	}

	// 4. Verify the id_token the way a relying party does: signature against the
	//    published JWKS, issuer and audience against what it discovered.
	verified, err := provider.Verifier(&oidclib.Config{ClientID: app.App.ID}).Verify(ctx, tokens.IDToken)
	if err != nil {
		t.Fatalf("go-oidc id_token verification: %v", err)
	}

	if verified.Issuer != issuer {
		t.Fatalf("verified issuer = %q, want %q", verified.Issuer, issuer)
	}

	// oauth2-proxy (and most relying parties) key the signed-in user off the
	// email claim and refuse a token without one when the email scope was
	// granted.
	var idClaims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified any    `json:"email_verified"`
	}
	if err := verified.Claims(&idClaims); err != nil {
		t.Fatalf("read id_token claims: %v", err)
	}
	if idClaims.Subject == "" {
		t.Error("id_token has no sub")
	}
	if idClaims.Email != email {
		t.Errorf("id_token email = %q, want %q (the email scope was granted)", idClaims.Email, email)
	}
}
