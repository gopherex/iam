//go:build integration

package postgres

// integration_e2e_formpost_test.go — the form_post response mode.
//
// The point of form_post is that the authorization response never appears in a
// URL: not in the address bar, not in browser history, not in the Referer of
// whatever the client renders next. So the things worth asserting are that the
// response really arrives as a form, that the form carries the same parameters
// a redirect would have, and that no code leaks into a Location header.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestE2EFormPostResponseMode(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	const redirectURI = "https://formpost.example.com/callback"

	clientID := e2eAppClient(t, ctx, ts, projectID, adminToken, redirectURI)

	// Discovery has to say so, or a client has no way to know it can ask.
	t.Run("advertised_in_discovery", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet,
			fmt.Sprintf("%s/p/%s/e/live/.well-known/openid-configuration", ts.URL, projectID), nil, nil)
		e2eWantStatus(t, r, http.StatusOK)

		var doc struct {
			ResponseModesSupported []string `json:"response_modes_supported"`
		}
		e2eDecode(t, r, &doc)

		if !containsString(doc.ResponseModesSupported, "form_post") {
			t.Fatalf("response_modes_supported = %v, want form_post among them", doc.ResponseModesSupported)
		}
	})

	t.Run("unknown_mode_is_still_refused", func(t *testing.T) {
		b := newBrowser(t, ts)
		status, body := b.do(t, ctx, http.MethodGet, fmt.Sprintf(
			"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=openid"+
				"&response_mode=carrier_pigeon&code_challenge=%s&code_challenge_method=S256",
			clientID, url.QueryEscape(redirectURI), url.QueryEscape(challengeFor(pkceVerifier))), nil, nil)
		if status != http.StatusFound {
			t.Fatalf("authorize: status %d, body %s", status, body)
		}

		if !strings.Contains(b.lastLocation, "unsupported_response_mode") {
			t.Fatalf("redirected to %q, want unsupported_response_mode", b.lastLocation)
		}
	})

	email := fmt.Sprintf("formpost-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)

	authorizeURL := fmt.Sprintf(
		"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=fp-1"+
			"&response_mode=form_post&code_challenge=%s&code_challenge_method=S256",
		clientID, url.QueryEscape(redirectURI), url.QueryEscape("openid email"),
		url.QueryEscape(challengeFor(pkceVerifier)))

	status, body := b.do(t, ctx, http.MethodGet, authorizeURL, nil, nil)
	if status != http.StatusFound {
		t.Fatalf("authorize: status %d, body %s", status, body)
	}

	// The redirect to our own interaction UI is not the authorization response,
	// so it stays a redirect whatever mode the client asked for.
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

	// Consent answers the hosted UI with a form to submit, not a URL to follow.
	status, body = b.do(t, ctx, http.MethodPost,
		"/v1/oauth/interaction/"+interactionID+"/consent",
		map[string]any{"granted_scopes": []string{"openid", "email"}, "remember": true},
		b.csrf(t, ctx, projectID))
	if status != http.StatusOK {
		t.Fatalf("consent: status %d, body %s", status, body)
	}

	var consent struct {
		RedirectTo string `json:"redirect_to"`
		FormPost   *struct {
			Action string            `json:"action"`
			Fields map[string]string `json:"fields"`
		} `json:"form_post"`
	}
	if err := json.Unmarshal(body, &consent); err != nil {
		t.Fatalf("decode consent: %v", err)
	}

	if consent.FormPost == nil {
		t.Fatalf("consent answered with %s — no form_post, so the code would travel in a URL", body)
	}

	if consent.FormPost.Action != redirectURI {
		t.Errorf("form action = %q, want the registered %q", consent.FormPost.Action, redirectURI)
	}

	// The whole point: nothing carrying the code may be handed back as a URL.
	if strings.Contains(consent.RedirectTo, "code=") {
		t.Errorf("redirect_to = %q still carries the code", consent.RedirectTo)
	}

	code := consent.FormPost.Fields["code"]
	if code == "" {
		t.Fatalf("form fields carry no code: %v", consent.FormPost.Fields)
	}

	if got := consent.FormPost.Fields["state"]; got != "fp-1" {
		t.Errorf("state field = %q, want the original fp-1", got)
	}

	// RFC 9207 travels in the form too, or a client checking `iss` breaks the
	// moment it switches response mode.
	if consent.FormPost.Fields["iss"] == "" {
		t.Errorf("form fields carry no iss: %v", consent.FormPost.Fields)
	}

	// And the code is a real one.
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkceVerifier},
		"client_id":     {clientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/oauth2/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token exchange: status %d", resp.StatusCode)
	}
}

// TestE2EFormPostSilentAuthorizeReturnsADocument: when an existing session and
// grant already answer the request, there is no hosted UI in the loop — the
// authorization endpoint itself has to produce the form.
func TestE2EFormPostSilentAuthorizeReturnsADocument(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	const redirectURI = "https://formpost-silent.example.com/callback"

	clientID := e2eAppClient(t, ctx, ts, projectID, adminToken, redirectURI)

	email := fmt.Sprintf("formpost-silent-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)
	tenant := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

	authorize := func(mode string) (int, string) {
		t.Helper()

		u := fmt.Sprintf(
			"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=fp-2"+
				"&code_challenge=%s&code_challenge_method=S256",
			clientID, url.QueryEscape(redirectURI), url.QueryEscape("openid email"),
			url.QueryEscape(challengeFor(pkceVerifier)))
		if mode != "" {
			u += "&response_mode=" + mode
		}

		status, body := b.do(t, ctx, http.MethodGet, u, nil, nil)

		return status, string(body)
	}

	// First pass: sign in and consent, so the grant is remembered.
	status, page := authorize("")
	if status != http.StatusFound {
		t.Fatalf("authorize: status %d, body %s", status, page)
	}

	interactionID := strings.TrimPrefix(b.lastLocation, "/oauth/interaction/")

	step := func(path string, body any, headers map[string]string) {
		t.Helper()

		st, raw := b.do(t, ctx, http.MethodPost, path, body, headers)
		if st != http.StatusOK {
			t.Fatalf("%s: status %d, body %s", path, st, raw)
		}
	}

	step("/v1/auth/flows", map[string]any{
		"kind": "signin", "email": email, "password": "Sup3rStr0ng!Pass", "cookie_mode": true,
	}, tenant)
	step("/v1/oauth/interaction/"+interactionID+"/login", map[string]any{}, b.csrf(t, ctx, projectID))
	step("/v1/oauth/interaction/"+interactionID+"/consent",
		map[string]any{"granted_scopes": []string{"openid", "email"}, "remember": true},
		b.csrf(t, ctx, projectID))

	// Second pass: the session and the grant answer it, so no UI is involved.
	status, page = authorize("form_post")
	if status != http.StatusOK {
		t.Fatalf("silent authorize in form_post mode: status %d, want 200 with a document; body %s",
			status, page)
	}

	if !strings.Contains(page, `action="`+redirectURI+`"`) {
		t.Errorf("document does not post to the registered redirect_uri: %s", page)
	}

	if !strings.Contains(page, `name="code"`) {
		t.Errorf("document carries no code field: %s", page)
	}

	if b.lastLocation != "" {
		t.Errorf("form_post response also set a Location: %q", b.lastLocation)
	}
}

