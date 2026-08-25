//go:build integration

package postgres

// integration_e2e_cookie_session_test.go — the browser session an OIDC provider
// needs.
//
// Until cookie_mode existed, no login path handed a browser an iam_session
// cookie: password sign-in and the whole flow engine returned tokens in the
// response body, and only the social / magic-link CALLBACKS (which are browser
// redirects) set cookies. Without a session cookie the provider's hosted pages
// cannot tell that the user is already signed in, so every relying party
// re-prompts for a password and it stops being single sign-on.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// browser is an HTTP client with a cookie jar — the thing e2eReq deliberately is
// not, since the rest of the suite exercises the token-mode API.
type browser struct {
	client *http.Client
	ts     *httptest.Server
	// lastLocation is the Location header of the most recent response, kept
	// because redirects are deliberately not followed.
	lastLocation string
}

func newBrowser(t *testing.T, ts *httptest.Server) *browser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return &browser{
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		ts: ts,
	}
}

func (b *browser) do(t *testing.T, ctx context.Context, method, path string, body any, headers map[string]string) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, b.ts.URL+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b.lastLocation = resp.Header.Get("Location")
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

// csrf returns the header set a cookie-mode POST needs: the project it is bound
// to plus a freshly issued synchronizer token.
func (b *browser) csrf(t *testing.T, ctx context.Context, projectID string) map[string]string {
	t.Helper()
	headers := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}
	status, body := b.do(t, ctx, http.MethodGet, "/v1/csrf", nil, headers)
	if status != http.StatusOK {
		t.Fatalf("GET /v1/csrf: status %d, body %s", status, body)
	}
	var out struct {
		CsrfToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode csrf: %v", err)
	}
	if out.CsrfToken == "" {
		t.Fatalf("empty csrf token: %s", body)
	}
	headers["X-Csrf-Token"] = out.CsrfToken
	return headers
}

// cookie returns the value the jar holds for name, or "".
func (b *browser) cookie(name string) string {
	u, err := url.Parse(b.ts.URL)
	if err != nil {
		return ""
	}
	for _, c := range b.client.Jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// TestE2ECookieModeFlowEstablishesBrowserSession: a flow created with
// cookie_mode=true hands the completed session back as HttpOnly cookies, and the
// browser is then authenticated by cookie alone.
func TestE2ECookieModeFlowEstablishesBrowserSession(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("cookie-mode-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)
	headers := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

	status, body := b.do(t, ctx, http.MethodPost, "/v1/auth/flows", map[string]any{
		"kind":        "signin",
		"email":       email,
		"password":    "Sup3rStr0ng!Pass",
		"cookie_mode": true,
	}, headers)
	if status != http.StatusOK {
		t.Fatalf("create flow: status %d, body %s", status, body)
	}

	var fs struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &fs); err != nil {
		t.Fatalf("decode flow state: %v", err)
	}
	if fs.Status != "completed" {
		t.Fatalf("status = %q, want completed (body %s)", fs.Status, body)
	}

	if b.cookie("iam_session") == "" {
		t.Fatal("cookie_mode flow completed without setting iam_session")
	}

	// The session cookie alone must authenticate the browser: this is exactly
	// what the provider's "already signed in" screen relies on.
	status, body = b.do(t, ctx, http.MethodGet, "/v1/auth/session", nil, headers)
	if status != http.StatusOK {
		t.Fatalf("GET /v1/auth/session by cookie: status %d, body %s", status, body)
	}
}

// TestE2ETokenModeFlowSetsNoSessionCookie: a programmatic client is unaffected —
// it never asked for cookie mode, so the session stays in the body.
func TestE2ETokenModeFlowSetsNoSessionCookie(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := fmt.Sprintf("token-mode-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)
	status, body := b.do(t, ctx, http.MethodPost, "/v1/auth/flows", map[string]any{
		"kind":     "signin",
		"email":    email,
		"password": "Sup3rStr0ng!Pass",
	}, map[string]string{"X-Client-Id": projectID, "X-Environment": "live"})
	if status != http.StatusOK {
		t.Fatalf("create flow: status %d, body %s", status, body)
	}

	if got := b.cookie("iam_session"); got != "" {
		t.Fatalf("token-mode flow set iam_session = %q; the session belongs in the body", got)
	}

	var fs struct {
		Session *struct {
			AccessToken string `json:"access_token"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &fs); err != nil {
		t.Fatalf("decode flow state: %v", err)
	}
	if fs.Session == nil || fs.Session.AccessToken == "" {
		t.Fatalf("token-mode flow returned no session in the body: %s", body)
	}
}

// TestE2ESessionCookieTTLFollowsPolicy: the cookie's Max-Age comes from the
// project's session_policy, not from a constant compiled into the transport.
func TestE2ESessionCookieTTLFollowsPolicy(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	email := fmt.Sprintf("cookie-ttl-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	const (
		accessTTL  = 1800
		refreshTTL = 7 * 24 * 3600
	)

	r := e2eReq(t, ctx, http.MethodPut,
		fmt.Sprintf("%s/v1/projects/%s/admin/config", ts.URL, projectID),
		map[string]any{"session_policy": map[string]any{"access_ttl": accessTTL, "refresh_ttl": refreshTTL}},
		e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/v1/auth/flows",
		strings.NewReader(fmt.Sprintf(
			`{"kind":"signin","email":%q,"password":"Sup3rStr0ng!Pass","cookie_mode":true}`, email)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", projectID)
	req.Header.Set("X-Environment", "live")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("create flow: status %d, body %s", resp.StatusCode, raw)
	}

	want := map[string]int{"iam_session": accessTTL, "iam_refresh": refreshTTL}
	seen := map[string]int{}
	for _, c := range resp.Cookies() {
		if _, ok := want[c.Name]; ok {
			seen[c.Name] = c.MaxAge
		}
	}

	for name, ttl := range want {
		got, ok := seen[name]
		if !ok {
			t.Errorf("no %s cookie was set", name)
			continue
		}
		if got != ttl {
			t.Errorf("%s Max-Age = %d, want %d (the configured session_policy)", name, got, ttl)
		}
	}
}
