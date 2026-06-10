//go:build integration

package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// flowCookieName mirrors api.FlowCookieName; kept as a literal so the test does
// not import pkg/api.
const flowCookieName = "iam_flow"

// rawFlow performs a request against the flow endpoints exposing cookies and
// headers (e2eReq hides them). Returns status, the response's Set-Cookie cookies,
// and the body.
func rawFlow(t *testing.T, ctx context.Context, method, url, projectID string, body any, reqCookies map[string]string) (int, []*http.Cookie, []byte) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(raw)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Client-Id", projectID)
	req.Header.Set("X-Environment", "live")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range reqCookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, _ := readAll(resp)
	return resp.StatusCode, resp.Cookies(), raw
}

func readAll(resp *http.Response) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// TestE2EFlowCookieSetAndResume verifies the iam_flow httpOnly cookie is set on
// create and that GET /flows/current resumes the flow from it.
func TestE2EFlowCookieSetAndResume(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := "cookie-" + newUUID()[:8] + "@example.com"

	status, cookies, body := rawFlow(t, ctx, http.MethodPost, ts.URL+"/v1/auth/flows", projectID,
		map[string]any{"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Cookie"}, nil)
	if status != http.StatusOK {
		t.Fatalf("create status = %d, body: %s", status, body)
	}
	ck := findCookie(cookies, flowCookieName)
	if ck == nil || ck.Value == "" {
		t.Fatal("expected iam_flow cookie on create")
	}
	if !ck.HttpOnly {
		t.Error("iam_flow cookie must be HttpOnly")
	}
	if ck.Path != "/v1/auth/flows" {
		t.Errorf("iam_flow cookie path = %q, want /v1/auth/flows", ck.Path)
	}
	// The cookie carries the flow_token (same value as the body's flow_token).
	var created struct {
		FlowToken string `json:"flow_token"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if ck.Value != created.FlowToken {
		t.Errorf("cookie value != flow_token")
	}

	t.Run("current with cookie resumes the flow", func(t *testing.T) {
		st, _, b := rawFlow(t, ctx, http.MethodGet, ts.URL+"/v1/auth/flows/current", projectID, nil,
			map[string]string{flowCookieName: ck.Value})
		if st != http.StatusOK {
			t.Fatalf("current status = %d, body: %s", st, b)
		}
		var fs struct {
			Kind string `json:"kind"`
			Step string `json:"step"`
		}
		if err := json.Unmarshal(b, &fs); err != nil {
			t.Fatalf("decode current: %v", err)
		}
		if fs.Kind != "signup" || fs.Step != "verify_email" {
			t.Errorf("current = %s/%s, want signup/verify_email", fs.Kind, fs.Step)
		}
	})

	t.Run("current without cookie returns 404", func(t *testing.T) {
		st, _, _ := rawFlow(t, ctx, http.MethodGet, ts.URL+"/v1/auth/flows/current", projectID, nil, nil)
		if st != http.StatusNotFound && st != http.StatusGone {
			t.Errorf("current no cookie status = %d, want 404/410", st)
		}
	})

	t.Run("current with garbage cookie returns 404", func(t *testing.T) {
		st, _, _ := rawFlow(t, ctx, http.MethodGet, ts.URL+"/v1/auth/flows/current", projectID, nil,
			map[string]string{flowCookieName: "ftk_garbage"})
		if st != http.StatusNotFound && st != http.StatusGone {
			t.Errorf("current garbage cookie status = %d, want 404/410", st)
		}
	})
}

// TestE2EFlowCookieClearedOnCompletion verifies the cookie is cleared when the
// flow completes, and the stale cookie no longer resolves a flow.
func TestE2EFlowCookieClearedOnCompletion(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID := e2eProject(t, ctx)
	email := "cookieclr-" + newUUID()[:8] + "@example.com"

	status, cookies, body := rawFlow(t, ctx, http.MethodPost, ts.URL+"/v1/auth/flows", projectID,
		map[string]any{"kind": "signup", "email": email, "password": "Sup3rStr0ng!Pass", "name": "Clr"}, nil)
	if status != http.StatusOK {
		t.Fatalf("create status = %d, body: %s", status, body)
	}
	ck := findCookie(cookies, flowCookieName)
	if ck == nil {
		t.Fatal("no iam_flow cookie on create")
	}
	token := ck.Value

	// Complete the flow: verify the emailed code.
	challengeID := findFlowChallengeID(t, ctx, token)
	code := captureCode(challengeID)
	if code == "" {
		t.Fatalf("no code captured for challenge %s", challengeID)
	}
	st, doneCookies, b := rawFlow(t, ctx, http.MethodPost,
		ts.URL+"/v1/auth/flows/"+token+"/submit", projectID,
		map[string]any{"action": "verify_email", "payload": map[string]any{"code": code}}, nil)
	if st != http.StatusOK {
		t.Fatalf("submit status = %d, body: %s", st, b)
	}
	var fs struct {
		Status  string `json:"status"`
		Session *struct {
			AccessToken string `json:"access_token"`
		} `json:"session"`
	}
	if err := json.Unmarshal(b, &fs); err != nil {
		t.Fatalf("decode submit: %v", err)
	}
	if fs.Status != "completed" || fs.Session == nil || fs.Session.AccessToken == "" {
		t.Fatalf("expected completed+session, got status=%s body=%s", fs.Status, b)
	}
	// Cookie must be cleared (MaxAge<0 or empty value).
	cleared := findCookie(doneCookies, flowCookieName)
	if cleared == nil {
		t.Fatal("expected a Set-Cookie clearing iam_flow on completion")
	}
	if cleared.Value != "" && cleared.MaxAge >= 0 {
		t.Errorf("iam_flow cookie not cleared: value=%q maxage=%d", cleared.Value, cleared.MaxAge)
	}

	// Stale cookie (old token) no longer resolves a flow.
	st2, _, _ := rawFlow(t, ctx, http.MethodGet, ts.URL+"/v1/auth/flows/current", projectID, nil,
		map[string]string{flowCookieName: token})
	if st2 != http.StatusNotFound && st2 != http.StatusGone {
		t.Errorf("current with completed-flow cookie status = %d, want 404/410", st2)
	}
}
