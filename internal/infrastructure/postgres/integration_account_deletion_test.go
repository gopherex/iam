//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

func deletionFixture(t *testing.T) (context.Context, *pgAccountStore, domain.SecurityScope, *domain.Session) {
	t.Helper()
	ctx, _, scope, session := securityFixture(t)
	return ctx, NewPgAccountStore(testDB, e2eEmitter), scope, session
}

func TestAccountDeletionScheduleCancel(t *testing.T) {
	ctx, store, scope, session := deletionFixture(t)
	input := domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}
	if _, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "wrong"}, deletionRequest); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatal(err)
	}
	state, err := store.DeletionStatus(ctx, scope)
	if err != nil || state.Status != "none" || state.GraceDays != 7 {
		t.Fatalf("state after wrong proof: %+v %v", state, err)
	}
	if _, err := store.SetDeletionPolicy(ctx, scope, domain.AccountDeletionPolicy{GraceDays: 7}); err != nil {
		t.Fatal(err)
	}
	state, err = store.MutateDeletion(ctx, scope, input, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != deletionPending || state.GraceDays != 7 || state.DeleteAt.Sub(*state.RequestedAt) != 7*deletionDay {
		t.Fatalf("bad schedule: %+v", state)
	}
	if _, err := NewAuthenticator(testDB, "").User(ctx, session.AccessToken); err != nil {
		t.Fatalf("schedule ended access: %v", err)
	}
	account, err := store.Get(ctx, scope.ProjectID, scope.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if account.Status != "active" || account.Deletion == nil {
		t.Fatal("deletion replaced account access status")
	}
	if _, err := store.UpdateProfile(ctx, domain.ProfileUpdateCmd{ProjectID: scope.ProjectID, AccountID: scope.AccountID, Name: "Still active"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetDeletionPolicy(ctx, scope, domain.AccountDeletionPolicy{GraceDays: 1}); err != nil {
		t.Fatal(err)
	}
	again, err := store.MutateDeletion(ctx, scope, input, deletionRequest)
	if err != nil || again.RequestID != state.RequestID || !again.DeleteAt.Equal(*state.DeleteAt) {
		t.Fatalf("request extended deadline: %+v %v", again, err)
	}
	cancelled, err := store.MutateDeletion(ctx, scope, input, deletionCancel)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != deletionCancelled || cancelled.CancelledAt == nil {
		t.Fatal(cancelled)
	}
	if err := store.SweepDeletions(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := NewAuthenticator(testDB, "").User(ctx, session.AccessToken); err != nil {
		t.Fatal(err)
	}
	history, err := store.DeletionHistory(ctx, scope)
	if err != nil || len(history.Events) != 2 {
		t.Fatalf("history: %+v %v", history, err)
	}
	fresh, err := store.MutateDeletion(ctx, scope, input, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.RequestID == state.RequestID || fresh.GraceDays != 1 {
		t.Fatal("fresh request did not use current policy")
	}
}

func expireDeletion(t *testing.T, ctx context.Context, scope domain.SecurityScope) {
	t.Helper()
	past := time.Now().Add(-time.Second)
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_account_deletions SET delete_at=$1,data=jsonb_set(data,'{delete_at}',to_jsonb($1::timestamptz)) WHERE user_id=$2`, past, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_users SET data=jsonb_set(data,'{deletion,delete_at}',to_jsonb($1::timestamptz)) WHERE id=$2`, past, scope.AccountID); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeletionDueAndCleanup(t *testing.T) {
	ctx, store, scope, session := deletionFixture(t)
	input := domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}
	if _, err := store.MutateDeletion(ctx, scope, input, deletionRequest); err != nil {
		t.Fatal(err)
	}
	expireDeletion(t, ctx, scope)
	if _, err := NewAuthenticator(testDB, "").User(ctx, session.AccessToken); err == nil {
		t.Fatal("access survived deadline before worker ran")
	}
	if _, err := store.MutateDeletion(ctx, scope, input, deletionCancel); err == nil {
		t.Fatal("late cancellation accepted")
	}
	var wait sync.WaitGroup
	failures := make([]error, 2)
	for i := range failures {
		wait.Go(func() { failures[i] = store.SweepDeletions(ctx) })
	}
	wait.Wait()
	for _, err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"iam_credentials", "iam_sessions", "iam_refresh_tokens", "iam_flows", "iam_security_deliveries"} {
		var count int
		if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE user_id=$1`, scope.AccountID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("remaining %s: %d", table, count)
		}
	}
	if _, err := store.Get(ctx, scope.ProjectID, scope.AccountID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatal(err)
	}
	state, err := store.DeletionStatus(ctx, scope)
	if err != nil || state.Status != deletionDeleted {
		t.Fatalf("state: %+v %v", state, err)
	}
	history, err := store.DeletionHistory(ctx, scope)
	if err != nil || len(history.Events) != 2 {
		t.Fatalf("duplicate finalization: %+v %v", history, err)
	}
}

func TestAccountDeletionScopeAndConcurrentRequest(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	var wait sync.WaitGroup
	results := make([]*domain.AccountDeletion, 2)
	failures := make([]error, 2)
	for i := range results {
		wait.Go(func() {
			results[i], failures[i] = store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest)
		})
	}
	wait.Wait()
	for _, err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if results[0].RequestID != results[1].RequestID {
		t.Fatal("concurrent request created two deadlines")
	}
	other := scope
	other.ProjectID = e2eProject(t, ctx)
	if _, err := store.MutateDeletion(ctx, other, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionCancel); err == nil {
		t.Fatal("cross-project cancellation")
	}
	foreign := scope
	foreign.Environment = "test"
	state, err := store.DeletionStatus(api.WithEnvironment(ctx, "test"), foreign)
	if err == nil && state.Status == deletionPending {
		t.Fatal("cross-environment state leak")
	}
}

func TestAccountDeletionHTTPPasswordAndProfile(t *testing.T) {
	ctx, _, scope, session := deletionFixture(t)
	server := e2eServer(t)
	headers := e2eBearer(session.AccessToken)
	failed := e2eReq(t, ctx, http.MethodDelete, server.URL+"/v1/users/me", map[string]any{"password": "wrong"}, headers)
	e2eWantStatus(t, failed, http.StatusUnauthorized)
	result := e2eReq(t, ctx, http.MethodDelete, server.URL+"/v1/users/me", map[string]any{"password": "Sup3rStr0ng!Pass"}, headers)
	e2eWantStatus(t, result, http.StatusOK)
	var state domain.AccountDeletion
	if err := json.Unmarshal(result.Body, &state); err != nil {
		t.Fatal(err)
	}
	if state.Status != deletionPending {
		t.Fatal(string(result.Body))
	}
	result = e2eReq(t, ctx, http.MethodGet, server.URL+"/v1/users/me", nil, headers)
	e2eWantStatus(t, result, http.StatusOK)
	var body struct {
		User struct {
			Deletion domain.AccountDeletion `json:"deletion"`
		} `json:"user"`
	}
	e2eDecode(t, result, &body)
	if body.User.Deletion.RequestID != state.RequestID {
		t.Fatal("profile omitted deletion state")
	}
	result = e2eReq(t, ctx, http.MethodPost, server.URL+"/v1/users/me/deletion/cancel", map[string]any{"password": "Sup3rStr0ng!Pass"}, headers)
	e2eWantStatus(t, result, http.StatusOK)
	_ = scope
}

func TestAccountDeletionMFAProof(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	security := NewPgSecurity(testDB, e2eEmitter)
	account, err := store.Get(ctx, scope.ProjectID, scope.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	factor := e2eActiveEmailFactor(t, ctx, scope.ProjectID, scope.AccountID, account.PrimaryEmail)
	input := domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}
	_, err = store.MutateDeletion(ctx, scope, input, deletionRequest)
	var denial *domain.Error
	if !errors.As(err, &denial) || denial.Code != domain.ErrStepUpRequired.Code {
		t.Fatal(err)
	}
	token := denial.Details["security_flow_token"].(string)
	flow, err := security.Flow(ctx, scope, token, domain.SecurityFlowInput{}, "get")
	if err != nil {
		t.Fatal(err)
	}
	flow = securitySubmit(t, ctx, security, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	flow = securitySubmit(t, ctx, security, scope, flow, domain.SecurityFlowInput{Action: "select_factor", FactorID: factor})
	flow = securitySubmit(t, ctx, security, scope, flow, domain.SecurityFlowInput{Action: "verify_mfa", FactorID: factor, Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	if len(flow.NextActions) != 2 || flow.NextActions[0] != "authorize_deletion" {
		t.Fatal(flow.NextActions)
	}
	flow = securitySubmit(t, ctx, security, scope, flow, domain.SecurityFlowInput{Action: "authorize_deletion"})
	input.ProofToken = flow.FlowToken
	if _, err := store.MutateDeletion(ctx, scope, input, deletionRequest); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MutateDeletion(ctx, scope, input, deletionCancel); err == nil {
		t.Fatal("request proof authorized cancellation")
	}
}

func TestAccountDeletionPasswordlessWithDetectionDisabled(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	security := NewPgSecurity(testDB, e2eEmitter)
	policy, err := security.Policy(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	policy.Mode = securityDisabled
	if _, err := security.SetPolicy(ctx, scope, *policy); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `DELETE FROM iam_credentials WHERE user_id=$1 AND type='password'`, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	_, err = store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{}, deletionRequest)
	var denial *domain.Error
	if !errors.As(err, &denial) || denial.Code != domain.ErrStepUpRequired.Code {
		t.Fatal(err)
	}
	flow, err := security.Flow(ctx, scope, denial.Details["security_flow_token"].(string), domain.SecurityFlowInput{}, "get")
	if err != nil {
		t.Fatal(err)
	}
	flow = securitySubmit(t, ctx, security, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	flow = securitySubmit(t, ctx, security, scope, flow, domain.SecurityFlowInput{Action: "authorize_deletion"})
	state, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{ProofToken: flow.FlowToken}, deletionRequest)
	if err != nil || state.GraceDays != 7 {
		t.Fatalf("state: %+v %v", state, err)
	}
}

type deletionFailEmitter struct{}

func (deletionFailEmitter) Emit(_ context.Context, event domain.Event) error {
	if event.Type == domain.WebhookEventUserDeleted {
		return domain.ErrConflict
	}
	return nil
}

func TestAccountDeletionCleanupRollback(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	if _, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest); err != nil {
		t.Fatal(err)
	}
	expireDeletion(t, ctx, scope)
	failing := NewPgAccountStore(testDB, deletionFailEmitter{})
	if err := failing.finishDeletion(ctx, scope); err == nil {
		t.Fatal("expected finalization failure")
	}
	state, err := store.DeletionStatus(ctx, scope)
	if err != nil || state.Status != deletionPending {
		t.Fatalf("lost retryable state: %+v %v", state, err)
	}
	var sessions int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_sessions WHERE user_id=$1`, scope.AccountID).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions == 0 {
		t.Fatal("failed transaction committed partial cleanup")
	}
	if _, err := store.Get(ctx, scope.ProjectID, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	if err := store.finishDeletion(ctx, scope); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeletionAdminAuthorization(t *testing.T) {
	ctx, _, scope, session := deletionFixture(t)
	server := e2eServer(t)
	endpoint := server.URL + "/v1/projects/" + scope.ProjectID + "/admin/account-deletion-policy"
	for _, headers := range []map[string]string{e2eMaster(), e2eBearer(deletionAdminToken(t, ctx, scope.ProjectID))} {
		result := e2eReq(t, ctx, http.MethodGet, endpoint, nil, headers)
		e2eWantStatus(t, result, http.StatusOK)
		result = e2eReq(t, ctx, http.MethodPut, endpoint, map[string]any{"grace_days": 14}, headers)
		e2eWantStatus(t, result, http.StatusOK)
	}
	for _, headers := range []map[string]string{nil, e2eBearer("invalid"), e2eBearer(session.AccessToken)} {
		result := e2eReq(t, ctx, http.MethodPut, endpoint, map[string]any{"grace_days": 1}, headers)
		e2eWantStatus(t, result, http.StatusUnauthorized)
	}
	other := e2eProject(t, ctx)
	result := e2eReq(t, ctx, http.MethodGet, endpoint, nil, e2eBearer(deletionAdminToken(t, ctx, other)))
	e2eWantStatus(t, result, http.StatusForbidden)
}

func deletionAdminToken(t *testing.T, ctx context.Context, project string) string {
	t.Helper()
	token, _, err := NewPgOperator(testDB, nopEmitter{}).MintAdminToken(ctx, domain.OperatorAdminTokenCmd{ProjectID: project, Name: "deletion-test", Scopes: []string{"admin:ui"}, ExpiresAt: nowUTC().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestAccountDeletionSurvivesStaleAccountUpdate(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	var old []byte
	if err := testDB.Pool.QueryRow(ctx, `SELECT data FROM iam_users WHERE id=$1`, scope.AccountID).Scan(&old); err != nil {
		t.Fatal(err)
	}
	state, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_users SET data=$1 WHERE id=$2`, old, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	account, err := store.Get(ctx, scope.ProjectID, scope.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if account.Deletion == nil || account.Deletion.RequestID != state.RequestID {
		t.Fatal("stale update erased deletion state")
	}
}

func TestAccountDeletionOrphanWorkers(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	if _, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest); err != nil {
		t.Fatal(err)
	}
	expireDeletion(t, ctx, scope)
	if _, err := testDB.Pool.Exec(ctx, `DELETE FROM iam_users WHERE id=$1`, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Go(func() { errs[i] = store.finishDeletion(ctx, scope) })
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	history, err := store.DeletionHistory(ctx, scope)
	if err != nil || len(history.Events) != 2 {
		t.Fatalf("orphan finalized more than once: %+v %v", history, err)
	}
}
