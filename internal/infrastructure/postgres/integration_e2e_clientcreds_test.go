//go:build integration

package postgres

// integration_e2e_clientcreds_test.go — the client-credentials grant.
//
// A service account is an OAuth client of the client-credentials kind, so the
// thing worth proving is that it works the way every OAuth library already
// expects: id and secret at /oauth2/token, a bearer back, and that bearer
// accepted on the runtime API.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// e2eServiceAccount creates a service account with the given scopes and returns
// its id and a fresh secret.
func e2eServiceAccount(
	t *testing.T, ctx context.Context, ts *httptest.Server, projectID, token string, scopes ...string,
) (string, string) {
	t.Helper()

	base := fmt.Sprintf("%s/v1/projects/%s/admin/service-accounts", ts.URL, projectID)

	r := e2eReq(t, ctx, http.MethodPost, base,
		map[string]any{"name": "e2e-sa", "scopes": scopes}, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)

	var created struct {
		ServiceAccount struct {
			ID string `json:"id"`
		} `json:"service_account"`
	}
	e2eDecode(t, r, &created)

	said := created.ServiceAccount.ID
	if said == "" {
		t.Fatalf("create service account: no id, body %s", r.Body)
	}

	sec := e2eReq(t, ctx, http.MethodPost, base+"/"+said+"/secrets",
		map[string]any{"name": "e2e"}, e2eBearer(token))
	e2eWantStatus(t, sec, http.StatusCreated)

	var issued struct {
		Secret       string `json:"secret"`
		ClientSecret string `json:"client_secret"`
	}
	e2eDecode(t, sec, &issued)

	secret := issued.Secret
	if secret == "" {
		secret = issued.ClientSecret
	}

	if secret == "" {
		t.Fatalf("create secret: nothing returned, body %s", sec.Body)
	}

	return said, secret
}

// e2eTokenForm posts a form to the token endpoint, optionally with basic auth.
func e2eTokenForm(
	t *testing.T, ctx context.Context, ts *httptest.Server, form url.Values, basicID, basicSecret string,
) (int, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/oauth2/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if basicID != "" {
		req.SetBasicAuth(basicID, basicSecret)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, raw
}

func TestE2EClientCredentialsGrant(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	said, secret := e2eServiceAccount(t, ctx, ts, projectID, adminToken, "users:read", "users:write")

	// Discovery has to advertise it, or a client has no way to know it can ask.
	t.Run("advertised_in_discovery", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet,
			fmt.Sprintf("%s/p/%s/e/live/.well-known/openid-configuration", ts.URL, projectID), nil, nil)
		e2eWantStatus(t, r, http.StatusOK)

		var doc struct {
			GrantTypesSupported []string `json:"grant_types_supported"`
		}
		e2eDecode(t, r, &doc)

		found := false

		for _, g := range doc.GrantTypesSupported {
			if g == "client_credentials" {
				found = true
			}
		}

		if !found {
			t.Fatalf("grant_types_supported = %v, want client_credentials", doc.GrantTypesSupported)
		}
	})

	t.Run("client_secret_basic", func(t *testing.T) {
		status, body := e2eTokenForm(t, ctx, ts,
			url.Values{"grant_type": {"client_credentials"}}, said, secret)
		if status != http.StatusOK {
			t.Fatalf("token: status %d, body %s", status, body)
		}

		var tok struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   int    `json:"expires_in"`
			Scope       string `json:"scope"`
		}
		if err := json.Unmarshal(body, &tok); err != nil {
			t.Fatalf("decode token: %v", err)
		}

		if tok.AccessToken == "" || tok.TokenType != "Bearer" || tok.ExpiresIn <= 0 {
			t.Fatalf("incomplete token response: %s", body)
		}

		if tok.Scope != "users:read users:write" {
			t.Errorf("scope = %q, want the account's own scopes", tok.Scope)
		}

		// The token has to be accepted where a service credential is accepted.
		r := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/config/public", nil, map[string]string{
			"Authorization": "Bearer " + tok.AccessToken,
			"X-Client-Id":   projectID,
		})
		e2eWantStatus(t, r, http.StatusOK)
	})

	t.Run("client_secret_post", func(t *testing.T) {
		status, body := e2eTokenForm(t, ctx, ts, url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {said},
			"client_secret": {secret},
		}, "", "")
		if status != http.StatusOK {
			t.Fatalf("token: status %d, body %s", status, body)
		}
	})

	// The request says how LITTLE to grant, never how much.
	t.Run("scope_can_narrow", func(t *testing.T) {
		status, body := e2eTokenForm(t, ctx, ts, url.Values{
			"grant_type": {"client_credentials"},
			"scope":      {"users:read"},
		}, said, secret)
		if status != http.StatusOK {
			t.Fatalf("token: status %d, body %s", status, body)
		}

		var tok struct {
			Scope string `json:"scope"`
		}
		_ = json.Unmarshal(body, &tok)

		if tok.Scope != "users:read" {
			t.Errorf("scope = %q, want the narrowed one", tok.Scope)
		}
	})

	t.Run("scope_cannot_widen", func(t *testing.T) {
		status, body := e2eTokenForm(t, ctx, ts, url.Values{
			"grant_type": {"client_credentials"},
			"scope":      {"users:read billing:admin"},
		}, said, secret)
		if status < 400 {
			t.Fatalf("status = %d, want a 4xx — an ungranted scope was issued: %s", status, body)
		}
	})

	t.Run("wrong_secret_is_refused", func(t *testing.T) {
		status, _ := e2eTokenForm(t, ctx, ts,
			url.Values{"grant_type": {"client_credentials"}}, said, secret+"x")
		if status < 400 {
			t.Fatalf("status = %d, want a 4xx for a wrong secret", status)
		}
	})

	// A disabled account must stop minting immediately, not at token expiry.
	t.Run("disabled_account_is_refused", func(t *testing.T) {
		did, dsecret := e2eServiceAccount(t, ctx, ts, projectID, adminToken, "users:read")

		r := e2eReq(t, ctx, http.MethodPatch,
			fmt.Sprintf("%s/v1/projects/%s/admin/service-accounts/%s", ts.URL, projectID, did),
			map[string]any{"disabled": true}, e2eBearer(adminToken))
		e2eWantStatus(t, r, http.StatusOK)

		status, _ := e2eTokenForm(t, ctx, ts,
			url.Values{"grant_type": {"client_credentials"}}, did, dsecret)
		if status < 400 {
			t.Fatalf("status = %d, want a 4xx for a disabled service account", status)
		}
	})
}
