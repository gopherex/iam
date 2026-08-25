//go:build integration

package postgres

// integration_e2e_admin_iac_test.go — HTTP e2e tests for the desired-state
// (infrastructure-as-code) admin surface:
//
//   GET /v1/projects/{id}/admin/config     — every configuration document
//   PUT /v1/projects/{id}/admin/config     — apply them all, atomically
//   PUT /v1/projects/{id}/admin/clients    — reconcile app clients
//
// Both PUTs accept ?dry_run=true (the `plan` half: return the change set, write
// nothing) and the clients PUT accepts ?prune=true.

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

// iacConfigChange mirrors one entry of ConfigApplyResult.changes.
type iacConfigChange struct {
	Document string         `json:"document"`
	Action   string         `json:"action"`
	Before   map[string]any `json:"before"`
	After    map[string]any `json:"after"`
}

type iacConfigApplyResult struct {
	DryRun  bool              `json:"dry_run"`
	Changes []iacConfigChange `json:"changes"`
	Config  struct {
		Auth           map[string]any `json:"auth"`
		PasswordPolicy map[string]any `json:"password_policy"`
		SessionPolicy  map[string]any `json:"session_policy"`
	} `json:"config"`
}

// iacChangeFor returns the change reported for one document, or nil.
func iacChangeFor(res iacConfigApplyResult, document string) *iacConfigChange {
	for i := range res.Changes {
		if res.Changes[i].Document == document {
			return &res.Changes[i]
		}
	}
	return nil
}

// TestE2EAdminConfigBundleGet checks that the aggregate read returns every
// document, including ones that were never written.
func TestE2EAdminConfigBundleGet(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	configURL := fmt.Sprintf("%s/v1/projects/%s/admin/config", ts.URL, projectID)

	r := e2eReq(t, ctx, http.MethodGet, configURL, nil, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)

	var bundle map[string]any
	e2eDecode(t, r, &bundle)
	for _, key := range []string{"auth", "password_policy", "session_policy", "mfa_policy", "rate_limits"} {
		if _, ok := bundle[key]; !ok {
			t.Errorf("config bundle missing document %q; body: %s", key, r.Body)
		}
	}
}

// TestE2EAdminConfigBundleDryRun is the `plan` half: the change set comes back
// but the database is untouched.
func TestE2EAdminConfigBundleDryRun(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	configURL := fmt.Sprintf("%s/v1/projects/%s/admin/config", ts.URL, projectID)

	desired := map[string]any{
		"auth":            map[string]any{"methods": []string{"email"}},
		"password_policy": map[string]any{"min_length": 14},
	}

	r := e2eReq(t, ctx, http.MethodPut, configURL+"?dry_run=true", desired, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)

	var res iacConfigApplyResult
	e2eDecode(t, r, &res)
	if !res.DryRun {
		t.Errorf("dry_run = false, want true; body: %s", r.Body)
	}

	change := iacChangeFor(res, "password_policy")
	if change == nil {
		t.Fatalf("no change reported for password_policy; body: %s", r.Body)
	}
	if change.Action != "create" {
		t.Errorf("password_policy action = %q, want create", change.Action)
	}
	if change.Before != nil {
		t.Errorf("password_policy before = %v, want null for a create", change.Before)
	}
	if got := change.After["min_length"]; fmt.Sprint(got) != "14" {
		t.Errorf("password_policy after.min_length = %v, want 14", got)
	}

	// Nothing may have been written.
	rGet := e2eReq(t, ctx, http.MethodGet, configURL, nil, e2eBearer(token))
	e2eWantStatus(t, rGet, http.StatusOK)

	var after struct {
		PasswordPolicy map[string]any `json:"password_policy"`
	}
	e2eDecode(t, rGet, &after)
	if v, ok := after.PasswordPolicy["min_length"]; ok && fmt.Sprint(v) == "14" {
		t.Fatalf("dry run wrote password_policy.min_length; body: %s", rGet.Body)
	}
}

// TestE2EAdminConfigBundleApply applies several documents at once and checks
// they are all readable afterwards, then that re-applying reports no change.
func TestE2EAdminConfigBundleApply(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	configURL := fmt.Sprintf("%s/v1/projects/%s/admin/config", ts.URL, projectID)

	desired := map[string]any{
		"auth":            map[string]any{"methods": []string{"email"}},
		"password_policy": map[string]any{"min_length": 14},
		"session_policy":  map[string]any{"access_ttl": 3600},
	}

	r := e2eReq(t, ctx, http.MethodPut, configURL, desired, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)

	var res iacConfigApplyResult
	e2eDecode(t, r, &res)
	if res.DryRun {
		t.Errorf("dry_run = true on a real apply; body: %s", r.Body)
	}
	if len(res.Changes) != 3 {
		t.Errorf("changes = %d, want 3; body: %s", len(res.Changes), r.Body)
	}

	rGet := e2eReq(t, ctx, http.MethodGet, configURL, nil, e2eBearer(token))
	e2eWantStatus(t, rGet, http.StatusOK)

	var stored struct {
		PasswordPolicy map[string]any `json:"password_policy"`
		SessionPolicy  map[string]any `json:"session_policy"`
	}
	e2eDecode(t, rGet, &stored)
	if got := stored.PasswordPolicy["min_length"]; fmt.Sprint(got) != "14" {
		t.Errorf("stored password_policy.min_length = %v, want 14; body: %s", got, rGet.Body)
	}
	if got := stored.SessionPolicy["access_ttl"]; fmt.Sprint(got) != "3600" {
		t.Errorf("stored session_policy.access_ttl = %v, want 3600; body: %s", got, rGet.Body)
	}

	// Applying the identical desired state again is a no-op.
	rAgain := e2eReq(t, ctx, http.MethodPut, configURL, desired, e2eBearer(token))
	e2eWantStatus(t, rAgain, http.StatusOK)

	var again iacConfigApplyResult
	e2eDecode(t, rAgain, &again)
	for _, c := range again.Changes {
		if c.Action != "unchanged" {
			t.Errorf("re-apply reported %s on %s, want unchanged; body: %s", c.Action, c.Document, rAgain.Body)
		}
	}
}

// TestE2EAdminConfigBundleAtomic checks that one invalid document rejects the
// whole bundle: the valid documents beside it must not land.
func TestE2EAdminConfigBundleAtomic(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	configURL := fmt.Sprintf("%s/v1/projects/%s/admin/config", ts.URL, projectID)

	// min_length is well past any sane bound; auth beside it is valid.
	bad := map[string]any{
		"auth":            map[string]any{"methods": []string{"email"}},
		"password_policy": map[string]any{"min_length": 100000},
	}

	r := e2eReq(t, ctx, http.MethodPut, configURL, bad, e2eBearer(token))
	if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 422/400 for an invalid document; body: %s", r.Status, r.Body)
	}

	rGet := e2eReq(t, ctx, http.MethodGet, configURL, nil, e2eBearer(token))
	e2eWantStatus(t, rGet, http.StatusOK)

	var stored struct {
		Auth map[string]any `json:"auth"`
	}
	e2eDecode(t, rGet, &stored)
	if len(stored.Auth) != 0 {
		t.Fatalf("rejected bundle still wrote auth = %v; body: %s", stored.Auth, rGet.Body)
	}
}

type iacClientChange struct {
	ID     string         `json:"id"`
	Action string         `json:"action"`
	Before map[string]any `json:"before"`
	After  map[string]any `json:"after"`
}

type iacClientsApplyResult struct {
	DryRun  bool              `json:"dry_run"`
	Prune   bool              `json:"prune"`
	Changes []iacClientChange `json:"changes"`
}

// e2eListClientIDs returns the ids of the project's app clients.
func e2eListClientIDs(t *testing.T, ctx context.Context, ts, projectID, token string) map[string]bool {
	t.Helper()
	r := e2eReq(t, ctx, http.MethodGet, fmt.Sprintf("%s/v1/projects/%s/admin/apps", ts, projectID), nil, e2eBearer(token))
	e2eWantStatus(t, r, http.StatusOK)
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	e2eDecode(t, r, &resp)
	out := map[string]bool{}
	for _, c := range resp.Data {
		out[c.ID] = true
	}
	return out
}

// TestE2EAdminClientsApply covers the desired-state client surface: dry run
// writes nothing, ids supplied by the caller are honoured, a second identical
// apply is a no-op, prune is off by default and deletes when asked.
func TestE2EAdminClientsApply(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)
	clientsURL := fmt.Sprintf("%s/v1/projects/%s/admin/clients", ts.URL, projectID)

	// A client that exists before the apply and is absent from the desired list.
	strayID := e2eAppClient(t, ctx, ts, projectID, token, "https://stray.example.com/cb")

	const wantedID = "iac-client-1"
	desired := map[string]any{
		"clients": []map[string]any{{
			"id":            wantedID,
			"name":          "IaC client",
			"type":          "spa",
			"redirect_uris": []string{"https://app.example.com/cb"},
		}},
	}

	t.Run("dry_run_writes_nothing", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPut, clientsURL+"?dry_run=true", desired, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		var res iacClientsApplyResult
		e2eDecode(t, r, &res)
		if !res.DryRun {
			t.Errorf("dry_run = false, want true; body: %s", r.Body)
		}
		if len(res.Changes) != 1 || res.Changes[0].Action != "create" {
			t.Fatalf("changes = %+v, want a single create; body: %s", res.Changes, r.Body)
		}

		if e2eListClientIDs(t, ctx, ts.URL, projectID, token)[wantedID] {
			t.Fatal("dry run created the client")
		}
	})

	t.Run("apply_creates_with_supplied_id", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPut, clientsURL, desired, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		var res iacClientsApplyResult
		e2eDecode(t, r, &res)
		if len(res.Changes) != 1 || res.Changes[0].Action != "create" || res.Changes[0].ID != wantedID {
			t.Fatalf("changes = %+v, want create of %q; body: %s", res.Changes, wantedID, r.Body)
		}

		ids := e2eListClientIDs(t, ctx, ts.URL, projectID, token)
		if !ids[wantedID] {
			t.Fatal("apply did not create the client")
		}
		if !ids[strayID] {
			t.Fatal("apply deleted a client absent from the list without prune")
		}
	})

	t.Run("reapply_is_unchanged", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPut, clientsURL, desired, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		var res iacClientsApplyResult
		e2eDecode(t, r, &res)
		if len(res.Changes) != 1 || res.Changes[0].Action != "unchanged" {
			t.Fatalf("changes = %+v, want a single unchanged; body: %s", res.Changes, r.Body)
		}
	})

	t.Run("apply_updates_in_place", func(t *testing.T) {
		updated := map[string]any{
			"clients": []map[string]any{{
				"id":            wantedID,
				"name":          "IaC client renamed",
				"type":          "spa",
				"redirect_uris": []string{"https://app.example.com/cb", "https://app.example.com/cb2"},
			}},
		}
		r := e2eReq(t, ctx, http.MethodPut, clientsURL, updated, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		var res iacClientsApplyResult
		e2eDecode(t, r, &res)
		if len(res.Changes) != 1 || res.Changes[0].Action != "update" {
			t.Fatalf("changes = %+v, want a single update; body: %s", res.Changes, r.Body)
		}
		if res.Changes[0].Before["name"] != "IaC client" {
			t.Errorf("before.name = %v, want the previous name", res.Changes[0].Before["name"])
		}
		if res.Changes[0].After["name"] != "IaC client renamed" {
			t.Errorf("after.name = %v, want the new name", res.Changes[0].After["name"])
		}
	})

	t.Run("prune_dry_run_reports_delete_without_deleting", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPut, clientsURL+"?prune=true&dry_run=true", desired, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		var res iacClientsApplyResult
		e2eDecode(t, r, &res)
		var deletes int
		for _, c := range res.Changes {
			if c.Action == "delete" && c.ID == strayID {
				deletes++
			}
		}
		if deletes != 1 {
			t.Fatalf("prune dry run reported %d deletes of the stray client, want 1; body: %s", deletes, r.Body)
		}
		if !e2eListClientIDs(t, ctx, ts.URL, projectID, token)[strayID] {
			t.Fatal("prune dry run deleted the stray client")
		}
	})

	t.Run("prune_deletes_absent_clients", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPut, clientsURL+"?prune=true", desired, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		ids := e2eListClientIDs(t, ctx, ts.URL, projectID, token)
		if ids[strayID] {
			t.Fatal("prune did not delete the client absent from the list")
		}
		if !ids[wantedID] {
			t.Fatal("prune deleted a client that is in the list")
		}
	})
}

// TestE2EAdminUserRoles covers the admin role surface end to end: replace a
// user's roles, read them back, and reject a malformed label.
func TestE2EAdminUserRoles(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	// Create a user through the runtime sign-up so the id is a real account.
	rSignUp := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/auth/sign-up",
		map[string]any{"email": fmt.Sprintf("roles+%s@example.com", newUUID()), "password": "Sup3rStr0ng!Pass"},
		map[string]string{"X-Client-Id": projectID, "X-Environment": "live"},
	)
	e2eWantStatus(t, rSignUp, http.StatusOK)

	var signUp struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	e2eDecode(t, rSignUp, &signUp)
	if signUp.User.ID == "" {
		t.Fatalf("sign-up returned no user id: %s", rSignUp.Body)
	}

	rolesURL := fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/roles", ts.URL, projectID, signUp.User.ID)

	t.Run("starts_empty", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodGet, rolesURL, nil, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		var body struct {
			Roles []string `json:"roles"`
		}
		e2eDecode(t, r, &body)
		if len(body.Roles) != 0 {
			t.Fatalf("roles = %v, want empty", body.Roles)
		}
	})

	t.Run("put_replaces_and_sorts", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPut, rolesURL,
			map[string]any{"roles": []string{"viewer", "ops", "viewer"}}, e2eBearer(token))
		e2eWantStatus(t, r, http.StatusOK)

		var body struct {
			Roles []string `json:"roles"`
		}
		e2eDecode(t, r, &body)
		if want := []string{"ops", "viewer"}; !reflect.DeepEqual(body.Roles, want) {
			t.Fatalf("roles = %v, want %v", body.Roles, want)
		}

		rGet := e2eReq(t, ctx, http.MethodGet, rolesURL, nil, e2eBearer(token))
		e2eWantStatus(t, rGet, http.StatusOK)
		e2eDecode(t, rGet, &body)
		if want := []string{"ops", "viewer"}; !reflect.DeepEqual(body.Roles, want) {
			t.Fatalf("roles after GET = %v, want %v", body.Roles, want)
		}
	})

	t.Run("rejects_malformed_role", func(t *testing.T) {
		r := e2eReq(t, ctx, http.MethodPut, rolesURL,
			map[string]any{"roles": []string{"not a role"}}, e2eBearer(token))
		if r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Fatalf("status = %d, want 422/400 for a malformed role; body: %s", r.Status, r.Body)
		}
	})
}
