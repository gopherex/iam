//go:build integration

package postgres

// integration_roles_test.go — role assignments (iam_user_roles) and the OIDC
// `groups` claim they feed.

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// rolesTestUser creates a project and a user in it, returning both ids.
func rolesTestUser(t *testing.T, ctx context.Context) (projectID, userID string) {
	t.Helper()
	op := NewPgOperator(testDB, nopEmitter{})
	proj, err := op.CreateProject(ctx, domain.ProjectCmd{Name: "roles " + newUUID()[:8]})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	users := NewPgAdminUsers(testDB, nopEmitter{})
	acc, err := users.Create(ctx, domain.RegisterCmd{
		ProjectID:   proj.ID,
		Environment: "live",
		Email:       "roles+" + newUUID() + "@example.com",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return proj.ID, acc.ID
}

func TestRolesSetAndList(t *testing.T) {
	ctx := context.Background()
	projectID, userID := rolesTestUser(t, ctx)
	roles := NewPgRoles(testDB, nopEmitter{})

	// A desired-state write: duplicates collapse and the result is sorted, so
	// the `groups` claim is byte-stable across tokens.
	got, err := roles.SetRoles(ctx, domain.AdminUserRolesSetCmd{
		ProjectID: projectID, Environment: "live", UserID: userID,
		Roles: []string{"ops", "viewer", "ops"},
	})
	if err != nil {
		t.Fatalf("SetRoles: %v", err)
	}
	if want := []string{"ops", "viewer"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SetRoles = %v, want %v", got, want)
	}

	listed, err := roles.ListRoles(ctx, domain.AdminUserRolesCmd{
		ProjectID: projectID, Environment: "live", UserID: userID,
	})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if want := []string{"ops", "viewer"}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListRoles = %v, want %v", listed, want)
	}

	// Replacing removes what the new list omits.
	if _, err := roles.SetRoles(ctx, domain.AdminUserRolesSetCmd{
		ProjectID: projectID, Environment: "live", UserID: userID,
		Roles: []string{"viewer"},
	}); err != nil {
		t.Fatalf("SetRoles replace: %v", err)
	}

	listed, err = roles.ListRoles(ctx, domain.AdminUserRolesCmd{
		ProjectID: projectID, Environment: "live", UserID: userID,
	})
	if err != nil {
		t.Fatalf("ListRoles after replace: %v", err)
	}
	if want := []string{"viewer"}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListRoles after replace = %v, want %v", listed, want)
	}
}

// TestRolesAreEnvironmentScoped: the same person can hold different roles in
// live and test, and one environment's assignment must not leak into the other.
func TestRolesAreEnvironmentScoped(t *testing.T) {
	ctx := context.Background()
	projectID, userID := rolesTestUser(t, ctx)
	roles := NewPgRoles(testDB, nopEmitter{})

	if _, err := roles.SetRoles(ctx, domain.AdminUserRolesSetCmd{
		ProjectID: projectID, Environment: "live", UserID: userID, Roles: []string{"ops"},
	}); err != nil {
		t.Fatalf("SetRoles live: %v", err)
	}

	other, err := roles.ListRoles(ctx, domain.AdminUserRolesCmd{
		ProjectID: projectID, Environment: "test", UserID: userID,
	})
	if err != nil {
		t.Fatalf("ListRoles test: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("roles leaked into the test environment: %v", other)
	}
}

func TestRolesRejectInvalidValues(t *testing.T) {
	ctx := context.Background()
	projectID, userID := rolesTestUser(t, ctx)
	roles := NewPgRoles(testDB, nopEmitter{})

	for _, bad := range []string{"", "   ", "has space", "wild*card", "quote\"d"} {
		if _, err := roles.SetRoles(ctx, domain.AdminUserRolesSetCmd{
			ProjectID: projectID, Environment: "live", UserID: userID, Roles: []string{bad},
		}); err == nil {
			t.Errorf("SetRoles(%q) = nil error, want a validation failure", bad)
		}
	}
}

// TestOIDCGroupsClaim is the point of the feature: a client that is granted the
// `groups` scope gets the user's IAM roles in both the access token and the
// id_token, and a client that is not granted it gets no claim at all.
func TestOIDCGroupsClaim(t *testing.T) {
	ctx := context.Background()
	projectID, userID := rolesTestUser(t, ctx)

	if _, err := NewPgRoles(testDB, nopEmitter{}).SetRoles(ctx, domain.AdminUserRolesSetCmd{
		ProjectID: projectID, Environment: "live", UserID: userID,
		Roles: []string{"ops", "platform:admin"},
	}); err != nil {
		t.Fatalf("SetRoles: %v", err)
	}

	grants := NewPgOIDCGrants(testDB, nopEmitter{})

	claimsOf := func(t *testing.T, token string) map[string]any {
		t.Helper()
		claims := testDB.Signer().UnverifiedClaims(token)
		if claims == nil {
			t.Fatal("token has no readable claims")
		}
		return claims
	}

	groupsOf := func(t *testing.T, claims map[string]any) []string {
		t.Helper()
		raw, ok := claims["groups"]
		if !ok {
			return nil
		}
		buf, err := json.Marshal(raw)
		if err != nil {
			t.Fatalf("marshal groups claim: %v", err)
		}
		var out []string
		if err := json.Unmarshal(buf, &out); err != nil {
			t.Fatalf("groups claim is not an array of strings: %s", buf)
		}
		return out
	}

	t.Run("granted_scope_yields_roles", func(t *testing.T) {
		resp, err := grants.mintTokenResponse(ctx, oidcTokenSubject{
			projectID: projectID,
			env:       "live",
			subject:   userID,
			clientID:  "client-groups",
			scopes:    []string{"openid", "groups"},
		})
		if err != nil {
			t.Fatalf("mintTokenResponse: %v", err)
		}

		access, _ := resp["access_token"].(string)
		if access == "" {
			t.Fatal("no access_token in response")
		}
		want := []string{"ops", "platform:admin"}
		if got := groupsOf(t, claimsOf(t, access)); !reflect.DeepEqual(got, want) {
			t.Errorf("access token groups = %v, want %v", got, want)
		}

		idToken, _ := resp["id_token"].(string)
		if idToken == "" {
			t.Fatal("no id_token in response")
		}
		if got := groupsOf(t, claimsOf(t, idToken)); !reflect.DeepEqual(got, want) {
			t.Errorf("id_token groups = %v, want %v", got, want)
		}
	})

	t.Run("scope_absent_yields_no_claim", func(t *testing.T) {
		resp, err := grants.mintTokenResponse(ctx, oidcTokenSubject{
			projectID: projectID,
			env:       "live",
			subject:   userID,
			clientID:  "client-nogroups",
			scopes:    []string{"openid"},
		})
		if err != nil {
			t.Fatalf("mintTokenResponse: %v", err)
		}

		for _, key := range []string{"access_token", "id_token"} {
			token, _ := resp[key].(string)
			if token == "" {
				t.Fatalf("no %s in response", key)
			}
			if _, present := claimsOf(t, token)["groups"]; present {
				t.Errorf("%s carries a groups claim without the groups scope", key)
			}
		}
	})

	// A user with no assignments still gets the claim when the scope is granted,
	// so a relying party can tell "asked and has none" from "did not ask".
	t.Run("granted_scope_with_no_roles_yields_empty_list", func(t *testing.T) {
		_, otherUser := rolesTestUser(t, ctx)
		resp, err := grants.mintTokenResponse(ctx, oidcTokenSubject{
			projectID: projectID,
			env:       "live",
			subject:   otherUser,
			clientID:  "client-groups",
			scopes:    []string{"openid", "groups"},
		})
		if err != nil {
			t.Fatalf("mintTokenResponse: %v", err)
		}

		access, _ := resp["access_token"].(string)
		claims := claimsOf(t, access)
		raw, ok := claims["groups"]
		if !ok {
			t.Fatal("granted groups scope produced no groups claim")
		}
		if got := groupsOf(t, claims); len(got) != 0 {
			t.Fatalf("groups = %v (%T), want an empty list", got, raw)
		}
	})

	// Roles are IAM's, not the client's: a role the client asks for but nobody
	// assigned must not appear.
	t.Run("client_cannot_inject_a_role", func(t *testing.T) {
		resp, err := grants.mintTokenResponse(ctx, oidcTokenSubject{
			projectID: projectID,
			env:       "live",
			subject:   userID,
			clientID:  "client-groups",
			scopes:    []string{"openid", "groups", "admin"},
		})
		if err != nil {
			t.Fatalf("mintTokenResponse: %v", err)
		}

		access, _ := resp["access_token"].(string)
		for _, role := range groupsOf(t, claimsOf(t, access)) {
			if role == "admin" {
				t.Fatal("a scope the client sent turned into a role")
			}
		}
	})
}
