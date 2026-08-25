//go:build integration

package postgres

// integration_e2e_dcr_test.go — dynamic client registration (RFC 7591) and the
// client management it implies (RFC 7592).
//
// The two things worth proving here are the ones a wrong implementation gets
// wrong quietly:
//   - registration is not open. In a multi-tenant server an open endpoint lets
//     anybody create a client inside somebody else's project.
//   - the registration access token authorises exactly one client. If it did
//     not, any registered client could read or delete any other.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// dcrRegistration is the subset of RFC 7591 metadata the tests assert on.
type dcrRegistration struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
	RegistrationAccessToken string   `json:"registration_access_token"`
	RegistrationClientURI   string   `json:"registration_client_uri"`
}

func TestE2EDynamicClientRegistration(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	_, token := e2eProjectAdmin(t, ctx)

	registerURL := ts.URL + "/oauth2/register"

	// A public client authenticates with nothing and must therefore be handed no
	// secret — PKCE is what protects it instead.
	t.Run("public_client_gets_no_secret", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, registerURL, map[string]any{
			"client_name":                "dcr-spa",
			"redirect_uris":              []string{"https://spa.example.com/cb"},
			"token_endpoint_auth_method": "none",
			"scope":                      "openid profile",
		}, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)

		var got dcrRegistration
		e2eDecode(t, r, &got)

		if got.ClientID == "" {
			t.Fatalf("no client_id, body: %s", r.Body)
		}
		if got.ClientSecret != "" {
			t.Errorf("public client was issued a secret")
		}
		if got.RegistrationAccessToken == "" {
			t.Errorf("no registration_access_token — the client cannot manage itself")
		}
		if want := registerURL + "/" + got.ClientID; got.RegistrationClientURI != want {
			t.Errorf("registration_client_uri = %q, want %q", got.RegistrationClientURI, want)
		}
	})

	t.Run("confidential_client_gets_a_secret", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, registerURL, map[string]any{
			"client_name":   "dcr-web",
			"redirect_uris": []string{"https://web.example.com/cb"},
		}, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)

		var got dcrRegistration
		e2eDecode(t, r, &got)

		if got.ClientSecret == "" {
			t.Errorf("confidential client got no secret, body: %s", r.Body)
		}
		if got.TokenEndpointAuthMethod != "client_secret_basic" {
			t.Errorf("token_endpoint_auth_method = %q, want the RFC 7591 default",
				got.TokenEndpointAuthMethod)
		}
	})

	// Without an initial access token there is no project to register into, so
	// this must not fall back to "some" project.
	t.Run("refuses_without_initial_access_token", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, registerURL, map[string]any{
			"client_name":   "dcr-anon",
			"redirect_uris": []string{"https://anon.example.com/cb"},
		}, nil)
		if r.Status != http.StatusUnauthorized && r.Status != http.StatusForbidden {
			t.Fatalf("status = %d, want 401/403 — registration must not be open", r.Status)
		}
	})

	t.Run("rejects_a_relative_redirect_uri", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, registerURL, map[string]any{
			"client_name":   "dcr-bad",
			"redirect_uris": []string{"/cb"},
		}, e2eBearer(token))
		if r.Status < 400 {
			t.Fatalf("status = %d, want a 4xx for a relative redirect_uri", r.Status)
		}
	})

	t.Run("manage_with_the_registration_access_token", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPost, registerURL, map[string]any{
			"client_name":                "dcr-managed",
			"redirect_uris":              []string{"https://managed.example.com/cb"},
			"token_endpoint_auth_method": "none",
		}, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusCreated)

		var reg dcrRegistration
		e2eDecode(t, r, &reg)

		self := fmt.Sprintf("%s/%s", registerURL, reg.ClientID)
		auth := e2eBearer(reg.RegistrationAccessToken)

		read := e2eReq(t, ctx, http.MethodGet, self, nil, auth)
		e2eWantStatus(t, read, http.StatusOK)

		var got dcrRegistration
		e2eDecode(t, read, &got)

		if got.ClientID != reg.ClientID {
			t.Errorf("read returned client_id %q, want %q", got.ClientID, reg.ClientID)
		}
		// The secret is issued once; a read must never hand it out again.
		if got.ClientSecret != "" {
			t.Errorf("read re-issued a client_secret")
		}

		// RFC 7592 §2.2: an update replaces the metadata. The redirect URI left
		// out of the request has to be gone afterwards.
		upd := e2eReq(t, ctx, http.MethodPut, self, map[string]any{
			"client_name":                "dcr-renamed",
			"redirect_uris":              []string{"https://managed.example.com/other"},
			"token_endpoint_auth_method": "none",
		}, auth)
		e2eWantStatus(t, upd, http.StatusOK)

		var after dcrRegistration
		e2eDecode(t, upd, &after)

		if after.ClientName != "dcr-renamed" {
			t.Errorf("client_name = %q, want the updated one", after.ClientName)
		}
		if len(after.RedirectURIs) != 1 || after.RedirectURIs[0] != "https://managed.example.com/other" {
			t.Errorf("redirect_uris = %v — update did not replace them", after.RedirectURIs)
		}

		del := e2eReq(t, ctx, http.MethodDelete, self, nil, auth)
		e2eWantStatus(t, del, http.StatusOK)

		gone := e2eReq(t, ctx, http.MethodGet, self, nil, auth)
		if gone.Status < 400 {
			t.Fatalf("status = %d after delete, want a 4xx", gone.Status)
		}
	})

	// The registration access token names one client. Presenting it against a
	// different client id must fail, or the token is a master key.
	t.Run("token_manages_only_its_own_client", func(t *testing.T) {
		mk := func(name, uri string) dcrRegistration {
			t.Helper()
			r := e2eReq(t, ctx, http.MethodPost, registerURL, map[string]any{
				"client_name":                name,
				"redirect_uris":              []string{uri},
				"token_endpoint_auth_method": "none",
			}, e2eBearer(token))
			e2eWantStatus(t, r, http.StatusCreated)
			var reg dcrRegistration
			e2eDecode(t, r, &reg)
			return reg
		}

		mine := mk("dcr-mine", "https://mine.example.com/cb")
		theirs := mk("dcr-theirs", "https://theirs.example.com/cb")

		r := e2eReq(t, ctx, http.MethodGet,
			fmt.Sprintf("%s/%s", registerURL, theirs.ClientID), nil,
			e2eBearer(mine.RegistrationAccessToken))
		if r.Status < 400 {
			t.Fatalf("status = %d — a registration token read somebody else's client", r.Status)
		}
	})

	// A client created through the admin API has no registration token and is
	// managed through the admin API; the RFC 7592 endpoints must not see it.
	t.Run("admin_created_client_is_not_dcr_managed", func(t *testing.T) {
		projectID, adminTok := e2eProjectAdmin(t, ctx)
		clientID := e2eAppClient(t, ctx, ts, projectID, adminTok, "https://admin.example.com/cb")

		r := e2eReq(t, ctx, http.MethodGet,
			fmt.Sprintf("%s/%s", registerURL, clientID), nil, e2eBearer(adminTok))
		if r.Status < 400 {
			t.Fatalf("status = %d — an admin-created client was readable as a DCR client", r.Status)
		}
	})
}

// TestE2EDynamicallyRegisteredClientCompletesCodeFlow is the test that matters:
// a client nobody configured by hand registers itself, and the credentials it
// was handed carry it through a full authorization code exchange. Registration
// that produces a client the token endpoint then refuses is worse than none.
func TestE2EDynamicallyRegisteredClientCompletesCodeFlow(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	const redirectURI = "https://dcr-app.example.com/callback"

	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/oauth2/register", map[string]any{
		"client_name":   "DCR Demo",
		"redirect_uris": []string{redirectURI},
		"scope":         "openid email",
	}, e2eBearer(adminToken))
	e2eWantStatus(t, r, http.StatusCreated)

	var reg dcrRegistration
	e2eDecode(t, r, &reg)

	if reg.ClientSecret == "" {
		t.Fatalf("registration returned no client_secret: %s", r.Body)
	}

	email := fmt.Sprintf("dcr-flow-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)

	challenge := challengeFor(pkceVerifier)
	authorizeURL := fmt.Sprintf(
		"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=dcr-1"+
			"&code_challenge=%s&code_challenge_method=S256",
		reg.ClientID, url.QueryEscape(redirectURI), url.QueryEscape("openid email"),
		url.QueryEscape(challenge))

	status, body := b.do(t, ctx, http.MethodGet, authorizeURL, nil, nil)
	if status != http.StatusFound {
		t.Fatalf("authorize: status %d, body %s", status, body)
	}

	const prefix = "/oauth/interaction/"
	if !strings.HasPrefix(b.lastLocation, prefix) {
		t.Fatalf("authorize redirected to %q, want an interaction handle", b.lastLocation)
	}

	interactionID := strings.TrimPrefix(b.lastLocation, prefix)
	tenant := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

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
		"/v1/oauth/interaction/"+interactionID+"/login", map[string]any{}, b.csrf(t, ctx, projectID))
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
		t.Fatalf("parse consent redirect %q: %v", consent.RedirectTo, err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("consent redirect %q carries no code", consent.RedirectTo)
	}

	// The secret handed out at registration is what authenticates the exchange.
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkceVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/oauth2/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(reg.ClientID, reg.ClientSecret)

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
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(raw, &tokens); err != nil {
		t.Fatalf("decode token response: %v", err)
	}

	if tokens.AccessToken == "" || tokens.IDToken == "" {
		t.Fatalf("token response is missing tokens: %s", raw)
	}
}

// TestE2EApplyPreservesRegistrationToken: the desired-state apply describes
// configuration, not credentials. If it dropped the registration access token,
// running IaC would revoke every dynamically registered client's ability to
// manage itself — invisibly, since nothing in the diff would mention it.
func TestE2EApplyPreservesRegistrationToken(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	registerURL := ts.URL + "/oauth2/register"

	r := e2eReq(t, ctx, http.MethodPost, registerURL, map[string]any{
		"client_name":                "dcr-iac",
		"redirect_uris":              []string{"https://iac.example.com/cb"},
		"token_endpoint_auth_method": "none",
	}, e2eBearer(adminToken))
	e2eWantStatus(t, r, http.StatusCreated)

	var reg dcrRegistration
	e2eDecode(t, r, &reg)

	// The console shows where the client came from.
	list := e2eReq(t, ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts.URL, projectID), nil, e2eBearer(adminToken))
	e2eWantStatus(t, list, http.StatusOK)

	var apps struct {
		Apps []struct {
			ID                    string `json:"id"`
			DynamicallyRegistered bool   `json:"dynamically_registered"`
		} `json:"data"`
	}
	e2eDecode(t, list, &apps)

	found := false

	for _, app := range apps.Apps {
		if app.ID == reg.ClientID {
			found = true

			if !app.DynamicallyRegistered {
				t.Errorf("dynamically_registered = false for a self-registered client")
			}
		}
	}

	if !found {
		t.Fatalf("registered client %q is not in the admin list: %s", reg.ClientID, list.Body)
	}

	// Now run an apply that redescribes the same client.
	apply := e2eReq(t, ctx, http.MethodPut,
		fmt.Sprintf("%s/v1/projects/%s/admin/clients", ts.URL, projectID),
		map[string]any{
			"clients": []map[string]any{{
				"id":            reg.ClientID,
				"name":          "dcr-iac",
				"type":          "spa",
				"redirect_uris": []string{"https://iac.example.com/cb"},
			}},
		}, e2eBearer(adminToken))
	e2eWantStatus(t, apply, http.StatusOK)

	// The registration access token must still manage the client.
	read := e2eReq(t, ctx, http.MethodGet,
		fmt.Sprintf("%s/%s", registerURL, reg.ClientID), nil,
		e2eBearer(reg.RegistrationAccessToken))
	e2eWantStatus(t, read, http.StatusOK)
}
