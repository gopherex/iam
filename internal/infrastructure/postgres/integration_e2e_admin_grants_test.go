//go:build integration

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// TestE2EAdminUserGrants covers the admin view over a user's OAuth consent
// grants: list, then revoke by id, then list again (empty).
func TestE2EAdminUserGrants(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	acct, _ := registerUser(t, ctx, projectID, fmt.Sprintf("grants-%s@example.com", newUUID()[:8]))

	// Seed a consent grant for the user.
	grantID := newUUID()
	grants := NewPgOIDCGrants(testDB, e2eEmitter, nil)
	if err := grants.persistGrant(ctx, projectID, &domain.Grant{
		ID:        grantID,
		AccountID: acct.ID,
		ClientID:  "client-" + newUUID()[:8],
		Scopes:    []string{"openid", "profile"},
		GrantedAt: nowUTC(),
	}); err != nil {
		t.Fatalf("seed grant: %v", err)
	}

	base := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/grants", ts.URL, projectID, acct.ID)

	// List → one grant.
	r := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)

	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	e2eDecode(t, r, &list)

	if len(list.Data) != 1 || list.Data[0].ID != grantID {
		t.Fatalf("list grants = %+v, want one grant %s", list.Data, grantID)
	}

	// Unauthenticated → 401.
	rNoAuth := e2eReq(t, ctx, http.MethodGet, base, nil, map[string]string{"X-Environment": "live"})
	e2eWantStatus(t, rNoAuth, http.StatusUnauthorized)

	// Revoke by id → 200.
	rDel := e2eReq(t, ctx, http.MethodDelete, base+"/"+grantID, nil, e2eBearer(token))
	e2eWantStatus(t, rDel, http.StatusOK)

	// List again → empty.
	r2 := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, r2, http.StatusOK)

	var list2 struct {
		Data []any `json:"data"`
	}
	e2eDecode(t, r2, &list2)

	if len(list2.Data) != 0 {
		t.Fatalf("after revoke: %d grants, want 0", len(list2.Data))
	}
}
