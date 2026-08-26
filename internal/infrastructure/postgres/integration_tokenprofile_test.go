//go:build integration

package postgres

// integration_tokenprofile_test.go — token profiles applied at minting.
//
// A profile that is stored, previewable and never applied is a setting an
// operator believes is in effect. These tests are about the two halves of
// "applied": the profile changes the token, and it cannot change the claims the
// provider owns.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// decodeJWTClaims reads a JWT's payload without verifying it — the signature is
// covered elsewhere; here only the claim set matters.
func decodeJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %q", token)
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("parse payload: %v", err)
	}

	return claims
}

func TestE2ETokenProfileAppliedAtMinting(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	// A profile with a different audience, a much shorter lifetime, one extra
	// claim, and one that tries to rewrite the subject.
	rp := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/token-profiles", ts.URL, projectID),
		map[string]any{
			"name": "api", "audience": "https://api.example.com", "access_ttl": 120,
			"claims_template": map[string]any{"tier": "gold", "sub": "not-you"},
		}, e2eBearer(adminToken))
	e2eWantStatus(t, rp, http.StatusCreated)

	var created struct {
		Profile struct {
			ID string `json:"id"`
		} `json:"profile"`
	}
	e2eDecode(t, rp, &created)

	if created.Profile.ID == "" {
		t.Fatalf("no profile id: %s", rp.Body)
	}

	const redirectURI = "https://profile-app.example.com/callback"

	clientID := e2eAppClient(t, ctx, ts, projectID, adminToken, redirectURI)

	bind := e2eReq(t, ctx, http.MethodPatch,
		fmt.Sprintf("%s/v1/projects/%s/admin/apps/%s", ts.URL, projectID, clientID),
		map[string]any{"token_profile_id": created.Profile.ID}, e2eBearer(adminToken))
	e2eWantStatus(t, bind, http.StatusOK)

	email := fmt.Sprintf("profile-%s@example.com", newUUID()[:8])
	registerUser(t, ctx, projectID, email)

	b := newBrowser(t, ts)
	tenant := map[string]string{"X-Client-Id": projectID, "X-Environment": "live"}

	authorizeURL := fmt.Sprintf(
		"/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=tp-1"+
			"&code_challenge=%s&code_challenge_method=S256",
		clientID, url.QueryEscape(redirectURI), url.QueryEscape("openid email"),
		url.QueryEscape(challengeFor(pkceVerifier)))

	status, body := b.do(t, ctx, http.MethodGet, authorizeURL, nil, nil)
	if status != http.StatusFound {
		t.Fatalf("authorize: status %d, body %s", status, body)
	}

	interactionID := strings.TrimPrefix(b.lastLocation, "/oauth/interaction/")

	step := func(path string, payload any, headers map[string]string) []byte {
		t.Helper()

		st, raw := b.do(t, ctx, http.MethodPost, path, payload, headers)
		if st != http.StatusOK {
			t.Fatalf("%s: status %d, body %s", path, st, raw)
		}

		return raw
	}

	step("/v1/auth/flows", map[string]any{
		"kind": "signin", "email": email, "password": "Sup3rStr0ng!Pass", "cookie_mode": true,
	}, tenant)
	step("/v1/oauth/interaction/"+interactionID+"/login", map[string]any{}, b.csrf(t, ctx, projectID))

	raw := step("/v1/oauth/interaction/"+interactionID+"/consent",
		map[string]any{"granted_scopes": []string{"openid", "email"}, "remember": true},
		b.csrf(t, ctx, projectID))

	var consent struct {
		RedirectTo string `json:"redirect_to"`
	}
	if err := json.Unmarshal(raw, &consent); err != nil {
		t.Fatalf("decode consent: %v", err)
	}

	parsed, err := url.Parse(consent.RedirectTo)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {parsed.Query().Get("code")},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkceVerifier},
		"client_id":     {clientID},
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/oauth2/token",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	defer resp.Body.Close()

	tokenBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token exchange: status %d, body %s", resp.StatusCode, tokenBody)
	}

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(tokenBody, &tok); err != nil {
		t.Fatalf("decode token: %v", err)
	}

	if tok.ExpiresIn != 120 {
		t.Errorf("expires_in = %d, want the profile's 120", tok.ExpiresIn)
	}

	claims := decodeJWTClaims(t, tok.AccessToken)

	if !audienceContains(claims["aud"], "https://api.example.com") {
		t.Errorf("aud = %v, want the profile's audience", claims["aud"])
	}

	if claims["tier"] != "gold" {
		t.Errorf("tier = %v, want the template's claim", claims["tier"])
	}

	// The template must not be able to rewrite a claim the provider owns.
	if claims["sub"] == "not-you" {
		t.Errorf("the claims template rewrote `sub` — that is an impersonation primitive")
	}
}

// audienceContains reports whether an `aud` claim names the given audience; it
// serializes as a string or an array depending on the token.
func audienceContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	}

	return false
}
