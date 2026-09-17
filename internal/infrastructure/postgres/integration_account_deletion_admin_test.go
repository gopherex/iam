//go:build integration

package postgres

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

func TestAccountDeletionAdminCancel(t *testing.T) {
	ctx, store, scope, session := deletionFixture(t)
	state, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	server := e2eServer(t)
	endpoint := server.URL + "/v1/projects/" + scope.ProjectID + "/admin/users/" + scope.AccountID + "/deletion/cancel"
	body := map[string]any{"request_id": state.RequestID, "reason": "User requested cancellation through support"}
	for _, headers := range []map[string]string{nil, e2eBearer(session.AccessToken), e2eBearer("invalid")} {
		e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, headers), http.StatusUnauthorized)
	}
	other, otherToken := e2eProjectAdmin(t, ctx)
	_ = other
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, e2eBearer(otherToken)), http.StatusForbidden)
	admin := e2eBearer(deletionAdminToken(t, ctx, scope.ProjectID))
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, map[string]any{"request_id": state.RequestID, "reason": " "}, admin), http.StatusUnprocessableEntity)
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, map[string]any{"request_id": "stale-request", "reason": "Support"}, admin), http.StatusConflict)
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, admin), http.StatusOK)
	// The operator may retry the same cancellation without duplicate events.
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, e2eMaster()), http.StatusOK)
	history, err := store.DeletionHistory(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if history.Deletion.Status != deletionCancelled || len(history.Events) != 2 {
		t.Fatalf("history: %+v", history)
	}
	event := history.Events[0]
	if event.ActorID == "" || event.ActorID == scope.AccountID || event.Reason != body["reason"] {
		t.Fatalf("missing admin attribution: %+v", event)
	}
	if _, err := NewAuthenticator(testDB, "").User(ctx, session.AccessToken); err != nil {
		t.Fatal(err)
	}
	state, err = store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	// A stale dialog cannot cancel a replacement request.
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, admin), http.StatusConflict)
	expireDeletion(t, ctx, scope)
	body["request_id"] = state.RequestID
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, admin), http.StatusConflict)
}

func TestAccountDeletionAdminCancelPreservesRestrictions(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	state, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_users SET status='banned',data=jsonb_set(data,'{Status}','"banned"') WHERE id=$1`, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	scope.ActorID = "support-admin"
	if _, err := store.AdminCancelDeletion(ctx, scope, domain.AdminAccountDeletionCancelInput{RequestID: state.RequestID, Reason: "Preserve account pending investigation"}); err != nil {
		t.Fatal(err)
	}
	account, err := store.Get(ctx, scope.ProjectID, scope.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if account.Status != "banned" {
		t.Fatal("cancellation removed account restriction")
	}
}

func TestAccountDeletionEmailReuse(t *testing.T) {
	ctx, store, scope, oldSession := deletionFixture(t)
	old, err := store.Get(ctx, scope.ProjectID, scope.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `INSERT INTO iam_user_roles(project_id,environment,user_id,role) VALUES($1,$2,$3,'sensitive-role')`, scope.ProjectID, scope.Environment, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest); err != nil {
		t.Fatal(err)
	}
	server := e2eServer(t)
	headers := map[string]string{"X-Client-Id": scope.ProjectID, "X-Environment": scope.Environment}
	body := map[string]any{"email": old.PrimaryEmail, "password": "BrandNew!Password123"}
	endpoint := server.URL + "/v1/auth/sign-up"
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, headers), http.StatusConflict)
	expireDeletion(t, ctx, scope)
	// The address is reserved until the actual cleanup transaction commits.
	e2eWantStatus(t, e2eReq(t, ctx, http.MethodPost, endpoint, body, headers), http.StatusConflict)
	if err := store.finishDeletion(ctx, scope); err != nil {
		t.Fatal(err)
	}
	result := e2eReq(t, ctx, http.MethodPost, endpoint, body, headers)
	e2eWantStatus(t, result, http.StatusOK)
	var response struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	e2eDecode(t, result, &response)
	if response.User.ID == "" || response.User.ID == scope.AccountID {
		t.Fatal("registration reused deleted identity")
	}
	fresh, err := store.Get(ctx, scope.ProjectID, response.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.EmailVerified || fresh.Deletion != nil {
		t.Fatal("new account inherited verification or deletion state")
	}
	var roles int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_user_roles WHERE user_id IN ($1,$2)`, scope.AccountID, fresh.ID).Scan(&roles); err != nil {
		t.Fatal(err)
	}
	if roles != 0 {
		t.Fatal("deleted roles survived or transferred")
	}
	if _, err := NewAuthenticator(testDB, "").User(ctx, oldSession.AccessToken); err == nil {
		t.Fatal("old session authorizes reused email")
	}
	if _, err := store.Get(ctx, scope.ProjectID, scope.AccountID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatal(err)
	}
	// Reprocessing the old request cannot touch the new account at the same email.
	if err := store.finishDeletion(ctx, scope); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, scope.ProjectID, fresh.ID); err != nil {
		t.Fatal(err)
	}
}
