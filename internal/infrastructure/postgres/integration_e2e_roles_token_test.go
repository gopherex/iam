//go:build integration

package postgres

// integration_e2e_roles_token_test.go — the roles claim in core-auth access
// tokens (opt-in via session_policy.access_token_claims) and the
// revoke-sessions-on-role-change fence.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// jwtPayload decodes the JWT payload without verification (test-only).
func jwtPayload(t *testing.T, token string) map[string]any {
	t.Helper()

	payload, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[1])
	if err != nil {
		t.Fatalf("decode jwt payload: %v", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("parse jwt payload: %v", err)
	}

	return claims
}

func TestE2ERolesClaimAndSessionRevocation(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	patchSessionPolicy := func(claims []string) {
		t.Helper()
		body := map[string]any{}
		if claims != nil {
			body["access_token_claims"] = claims
		}
		r := e2eReq(t, ctx, http.MethodPatch,
			ts.URL+"/v1/projects/"+projectID+"/admin/config/session-policy", body,
			e2eBearer(adminToken))
		e2eWantStatus(t, r, http.StatusOK)
	}

	setRoles := func(userID string, roles ...string) {
		t.Helper()
		r := e2eReq(t, ctx, http.MethodPut,
			ts.URL+"/v1/projects/"+projectID+"/admin/users/"+userID+"/roles",
			map[string]any{"roles": roles}, e2eBearer(adminToken))
		e2eWantStatus(t, r, http.StatusOK)
	}

	// A user with a password, signed in via the direct endpoints.
	signup := func(email string) (string, string) {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-up",
			map[string]any{"email": email, "password": "Sup3rStr0ng!"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live", "Content-Type": "application/json"})
		e2eWantStatus(t, r, http.StatusOK)

		var body struct {
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(r.Body, &body); err != nil {
			t.Fatalf("decode sign-up: %v", err)
		}

		return body.User.ID, body.Session.AccessToken
	}

	// signIn issues a fresh session for an existing user (the roles bearer).
	signIn := func(email string) string {
		r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-in/password",
			map[string]any{"email": email, "password": "Sup3rStr0ng!"},
			map[string]string{"X-Client-Id": projectID, "X-Environment": "live", "Content-Type": "application/json"})
		e2eWantStatus(t, r, http.StatusOK)

		var body struct {
			Session struct {
				AccessToken string `json:"access_token"`
			} `json:"session"`
		}
		if err := json.Unmarshal(r.Body, &body); err != nil {
			t.Fatalf("decode sign-in: %v", err)
		}

		return body.Session.AccessToken
	}

	bearerEmail := "rt-" + newUUID()[:8] + "@example.com"
	userID, _ := signup(bearerEmail)
	setRoles(userID, "ops")

	// 1. Flag off → no roles claim even with roles assigned.
	patchSessionPolicy(nil)
	if _, ok := jwtPayload(t, signIn(bearerEmail))["roles"]; ok {
		t.Fatal("roles claim must be absent while session_policy opts out")
	}

	// 2. Flag on → the roles bearer's token carries [ops]; a user without
	// roles emits nothing.
	patchSessionPolicy([]string{"roles"})
	tokenWithOps := signIn(bearerEmail)
	if got := jwtPayload(t, tokenWithOps)["roles"]; got == nil {
		t.Fatal("roles claim missing while session_policy opts in")
	} else {
		arr, _ := got.([]any)
		if len(arr) != 1 || arr[0] != "ops" {
			t.Fatalf("roles = %v, want [ops]", got)
		}
	}

	_, rolelessToken := signup("rl-" + newUUID()[:8] + "@example.com")
	if _, ok := jwtPayload(t, rolelessToken)["roles"]; ok {
		t.Fatal("empty role set must not emit the claim")
	}

	// 3. GET /v1/tokens/current mirrors the claim.
	me := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/tokens/current", nil,
		map[string]string{"Authorization": "Bearer " + tokenWithOps, "X-Client-Id": projectID})
	e2eWantStatus(t, me, http.StatusOK)

	var current struct {
		Claims struct {
			Roles []string `json:"roles"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(me.Body, &current); err != nil {
		t.Fatalf("decode current claims: %v", err)
	}
	if len(current.Claims.Roles) != 1 || current.Claims.Roles[0] != "ops" {
		t.Fatalf("current claims roles = %v", current.Claims.Roles)
	}

	// 4. Changing roles revokes sessions: the old token turns unauthorized.
	setRoles(userID, "viewer")
	old := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/users/me", nil,
		map[string]string{"Authorization": "Bearer " + tokenWithOps, "X-Client-Id": projectID})
	e2eWantStatus(t, old, http.StatusUnauthorized)

	// 5. Re-saving the same roles must NOT revoke (no churn on no-op saves).
	keepToken := signIn(bearerEmail)
	setRoles(userID, "viewer", "viewer") // NormalizeRoles dedupes → same set

	keep := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/users/me", nil,
		map[string]string{"Authorization": "Bearer " + keepToken, "X-Client-Id": projectID})
	e2eWantStatus(t, keep, http.StatusOK)

	// 6. Admin users listing carries the role sets.
	list := e2eReq(t, ctx, http.MethodGet,
		ts.URL+"/v1/projects/"+projectID+"/admin/users", nil, e2eBearer(adminToken))
	e2eWantStatus(t, list, http.StatusOK)

	var users struct {
		Data []struct {
			ID    string   `json:"id"`
			Roles []string `json:"roles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(list.Body, &users); err != nil {
		t.Fatalf("decode users: %v", err)
	}

	found := false
	for _, u := range users.Data {
		if u.ID == userID {
			found = true
			if len(u.Roles) != 1 || u.Roles[0] != "viewer" {
				t.Fatalf("listed roles = %v, want [viewer]", u.Roles)
			}
		}
	}
	if !found {
		t.Fatal("user missing from admin listing")
	}
}
