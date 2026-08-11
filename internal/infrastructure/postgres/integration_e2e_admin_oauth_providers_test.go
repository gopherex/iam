//go:build integration

package postgres

import (
	"context"
	"net/http"
	"testing"
)

// TestE2EAdminOAuthProviders covers the admin OAuth social-provider config CRUD.
// The client secret is write-only: it is accepted on create/update but never
// returned on read.
func TestE2EAdminOAuthProviders(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	base := ts.URL + "/v1/projects/" + projectID + "/admin/oauth-providers"

	// Create.
	r := e2eReq(t, ctx, http.MethodPost, base, map[string]any{
		"provider":      "google",
		"client_id":     "client-abc",
		"client_secret": "super-secret",
		"scopes":        []string{"openid", "email"},
		"enabled":       true,
	}, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusCreated)

	var created struct {
		ID           string `json:"id"`
		Provider     string `json:"provider"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	e2eDecode(t, r, &created)

	if created.ID == "" || created.Provider != "google" || created.ClientID != "client-abc" {
		t.Fatalf("create response = %+v", created)
	}

	if created.ClientSecret != "" {
		t.Fatal("client_secret must not be returned (write-only)")
	}

	// List → one provider, still no secret.
	rl := e2eReq(t, ctx, http.MethodGet, base, nil, e2eBearer(token))
	e2eWantStatus(t, rl, http.StatusOK)

	var list struct {
		Data []struct {
			ID           string `json:"id"`
			ClientSecret string `json:"client_secret"`
		} `json:"data"`
	}
	e2eDecode(t, rl, &list)

	if len(list.Data) != 1 || list.Data[0].ID != created.ID {
		t.Fatalf("list = %+v, want the created provider", list.Data)
	}

	if list.Data[0].ClientSecret != "" {
		t.Fatal("list leaked client_secret")
	}

	// Update (disable, new scopes; secret omitted keeps the stored one).
	ru := e2eReq(t, ctx, http.MethodPatch, base+"/"+created.ID, map[string]any{
		"provider":  "google",
		"client_id": "client-abc",
		"scopes":    []string{"openid"},
		"enabled":   false,
	}, e2eBearer(token))
	e2eWantStatus(t, ru, http.StatusOK)

	// Delete.
	rd := e2eReq(t, ctx, http.MethodDelete, base+"/"+created.ID, nil, e2eBearer(token))
	e2eWantStatus(t, rd, http.StatusOK)

	// Deleting again → 404.
	rd2 := e2eReq(t, ctx, http.MethodDelete, base+"/"+created.ID, nil, e2eBearer(token))
	e2eWantStatus(t, rd2, http.StatusNotFound)

	// Unauthenticated → 401.
	rNoAuth := e2eReq(t, ctx, http.MethodGet, base, nil, map[string]string{"X-Environment": "live"})
	e2eWantStatus(t, rNoAuth, http.StatusUnauthorized)
}
