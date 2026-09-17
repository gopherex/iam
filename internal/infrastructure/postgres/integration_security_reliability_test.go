//go:build integration

package postgres

import (
	"context"
	"errors"
	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/xlog"
	"sync"
	"testing"
	"time"
)

func TestSecurityReliabilityRecoveryMarkerSurvivesAccountUpdate(t *testing.T) {
	ctx, s, scope, sess := securityFixture(t)
	prove := func() *domain.SecurityFlowState {
		f, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: sess.ID}, "review")
		if err != nil {
			t.Fatal(err)
		}
		return securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	}
	old := prove()
	current := prove()
	current = securitySubmit(t, ctx, s, scope, current, domain.SecurityFlowInput{Action: "report_activity"})
	securitySubmit(t, ctx, s, scope, current, domain.SecurityFlowInput{Action: "set_password", NewPassword: "N3wSecur3!Pass123"})
	_, err := s.Flow(ctx, scope, old.FlowToken, domain.SecurityFlowInput{Version: old.Version, Action: "report_activity"}, "submit")
	if !errors.Is(err, domain.ErrFlowExpired) {
		t.Fatalf("setup: old flow must be expired after recovery: %v", err)
	}
	acc, err := s.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
	_, err = withTxRet(ctx, testDB, func(ctx context.Context) (bool, error) { return true, ca.coreAuthUpdateAccount(ctx, acc) })
	if err != nil {
		t.Fatal(err)
	}
	renewed, err := s.Flow(ctx, scope, old.FlowToken, domain.SecurityFlowInput{Version: old.Version, Action: "report_activity"}, "submit")
	if !errors.Is(err, domain.ErrFlowExpired) {
		t.Fatalf("old proof accepted after account update: %+v %v", renewed, err)
	}
}
func TestSecurityReliabilityDeletionDeliveryRetryAfterOneHour(t *testing.T) {
	ctx, store, scope, _ := deletionFixture(t)
	if _, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := testDB.Pool.QueryRow(ctx, `SELECT id FROM iam_security_deliveries WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, scope.AccountID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_security_deliveries SET created_at=$1,data=jsonb_set(data,'{created_at}',to_jsonb($1::timestamptz)) WHERE id=$2`, old, id); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPgSecurity(testDB, e2eEmitter).RetryDelivery(ctx, scope, id); err != nil {
		t.Errorf("BUG: non-expiring deletion notification cannot be retried after 1h: %v", err)
	}
}
func TestSecurityReliabilityDeletionDeadlineIntrospectionAndOIDCRefresh(t *testing.T) {
	ctx, store, scope, sess := deletionFixture(t)
	oidc := NewPgOIDCGrants(testDB, nopEmitter{}, nil)
	initial, err := withTxRet(ctx, testDB, func(ctx context.Context) (map[string]any, error) {
		return oidc.mintTokenResponse(ctx, oidcTokenSubject{projectID: scope.ProjectID, env: scope.Environment, subject: scope.AccountID, sessionID: sess.ID, scopes: []string{"openid", "offline_access"}})
	})
	if err != nil {
		t.Fatalf("mint setup: %v", err)
	}
	refresh, ok := initial["refresh_token"].(string)
	if !ok || refresh == "" {
		t.Fatal("setup: no refresh")
	}
	if _, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest); err != nil {
		t.Fatal(err)
	}
	expireDeletion(t, ctx, scope)
	if _, err := NewAuthenticator(testDB, "").User(ctx, sess.AccessToken); err == nil {
		t.Fatal("setup: expected normal auth rejected")
	}
	ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
	for name, token := range map[string]string{"core_access": sess.AccessToken, "core_refresh": sess.RefreshToken, "oidc_access": initial["access_token"].(string), "oidc_refresh": refresh} {
		result, err := ca.Introspect(ctx, scope.ProjectID, token)
		if err != nil {
			t.Fatal(err)
		}
		if result.Active {
			t.Errorf("BUG: %s introspects active after deletion deadline", name)
		}
	}
	result, err := oidc.tokenRefreshGrant(ctx, domain.OIDCTokenCmd{ProjectID: scope.ProjectID, Env: scope.Environment, RefreshToken: refresh})
	if err == nil && result["access_token"] != "" {
		t.Error("BUG: OIDC minted new access token after deletion deadline")
	} else {
		t.Logf("refresh rejected: %v", err)
	}
}

func TestSecurityReliabilityDeletionDuringRecovery(t *testing.T) {
	ctx, s, scope, sess := securityFixture(t)
	store := NewPgAccountStore(testDB, e2eEmitter)
	in := domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}
	deletion, err := store.MutateDeletion(ctx, scope, in, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	f, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: sess.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	if f.Deletion != nil {
		t.Fatal("deletion disclosed before proof")
	}
	_, err = s.Flow(ctx, scope, f.FlowToken, domain.SecurityFlowInput{Version: f.Version, Action: "cancel_deletion", RequestID: deletion.RequestID}, "submit")
	if !errors.Is(err, domain.ErrStepUpRequired) {
		t.Fatalf("unproved cancellation: %v", err)
	}
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	if f.Deletion == nil || f.Deletion.RequestID != deletion.RequestID {
		t.Fatal("missing deletion after ownership proof")
	}
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "report_activity"})
	if f.Deletion.Status != deletionPending {
		t.Fatal("recovery cancelled deletion without user choice")
	}
	_, err = s.Flow(ctx, scope, f.FlowToken, domain.SecurityFlowInput{Version: f.Version, Action: "cancel_deletion", RequestID: newUUID()}, "submit")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale request: %v", err)
	}
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "cancel_deletion", RequestID: deletion.RequestID})
	if f.Deletion.Status != deletionCancelled || f.Step != "restore_access" || f.AllSessions {
		t.Fatalf("unexpected state: %+v", f)
	}
	var guards int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_security_guards WHERE user_id=$1`, scope.AccountID).Scan(&guards); err != nil {
		t.Fatal(err)
	}
	if guards != 1 {
		t.Fatal("cancellation removed recovery guard")
	}
	history, err := store.DeletionHistory(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Events) != 2 || history.Events[0].ActorID != scope.AccountID {
		t.Fatalf("cancellation audit: %+v", history)
	}
	expireDeletion(t, ctx, scope)
	if err := store.SweepDeletions(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.account(ctx, scope); err != nil {
		t.Fatalf("cancelled account removed: %v", err)
	}
}
func TestSecurityReliabilityRetentionPreservesOpenSupportCase(t *testing.T) {
	ctx, s, scope, sess := securityFixture(t)
	policy, err := s.Policy(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	policy.RetentionDays = 1
	if _, err := s.SetPolicy(ctx, scope, *policy); err != nil {
		t.Fatal(err)
	}
	f, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: sess.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "report_activity"})
	internal, err := s.loadFlow(ctx, scope, f.FlowToken)
	if err != nil {
		t.Fatal(err)
	}
	_, err = withTxRet(ctx, testDB, func(ctx context.Context) (bool, error) { return true, s.openCase(ctx, internal, "Still recovering") })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_security_cases SET created_at=$1 WHERE id=$2`, time.Now().Add(-48*time.Hour), internal.CaseID); err != nil {
		t.Fatal(err)
	}
	testDB.gcSecurityRetention(ctx, xlog.NewJSON())
	var cases, guards int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_security_cases WHERE id=$1`, internal.CaseID).Scan(&cases); err != nil {
		t.Fatal(err)
	}
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_security_guards WHERE user_id=$1`, scope.AccountID).Scan(&guards); err != nil {
		t.Fatal(err)
	}
	if cases != 1 || guards != 1 {
		t.Fatalf("active recovery lost: cases=%d guards=%d", cases, guards)
	}
}

func reliabilityContinuation(t *testing.T, ctx context.Context, s *pgSecurity, scope domain.SecurityScope) string {
	t.Helper()
	account, err := s.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	f, err := s.Start(ctx, scope, domain.SecurityFlowInput{Identifier: account.PrimaryEmail, Contact: account.PrimaryEmail}, "recovery")
	if err != nil {
		t.Fatal(err)
	}
	securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	return securityLastPayload(t, ctx, scope, securityRecoveryTemplate).Continuation
}
func TestSecurityReliabilityContinuationRetry(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
	token := reliabilityContinuation(t, ctx, s, scope)
	key, err := accountRandomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	input := domain.SecurityFlowInput{ContinuationToken: token, ExchangeKey: key}
	first, err := s.Start(ctx, scope, input, "exchange")
	if err != nil {
		t.Fatal(err)
	}
	retry, err := s.Start(ctx, scope, input, "exchange")
	if err != nil || retry.FlowToken != first.FlowToken {
		t.Fatalf("lost response retry: %+v %v", retry, err)
	}
	for _, wrong := range []string{"", newUUID()} {
		bad := input
		bad.ExchangeKey = wrong
		if _, err := s.Start(ctx, scope, bad, "exchange"); !errors.Is(err, domain.ErrInvalidToken) {
			t.Fatalf("consumed continuation with wrong secret: %v", err)
		}
	}
	other := scope
	other.Environment = "test"
	if _, err := s.Start(ctx, other, input, "exchange"); err == nil {
		t.Fatal("cross-environment retry")
	}
	securitySubmit(t, ctx, s, scope, first, domain.SecurityFlowInput{Action: "support_message", Message: "More context"})
	if _, err := s.Start(ctx, scope, input, "exchange"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("retry followed rotation: %v", err)
	}
}
func TestSecurityReliabilityContinuationRetryExpires(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
	input := domain.SecurityFlowInput{ContinuationToken: reliabilityContinuation(t, ctx, s, scope), ExchangeKey: newUUID()}
	if _, err := s.Start(ctx, scope, input, "exchange"); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := testDB.Pool.QueryRow(ctx, `SELECT exchange_data FROM iam_security_continuations WHERE token_hash=$1`, accountHashToken(input.ContinuationToken)).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var replay securityExchange
	if err := s.decrypt(raw, &replay); err != nil {
		t.Fatal(err)
	}
	replay.Until = time.Now().Add(-time.Second)
	raw, err := s.encrypt(replay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_security_continuations SET exchange_data=$1 WHERE token_hash=$2`, raw, accountHashToken(input.ContinuationToken)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(ctx, scope, input, "exchange"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("retry outside grace: %v", err)
	}
}
func TestSecurityReliabilityRecoveryCancelRejectsDeadline(t *testing.T) {
	ctx, s, scope, sess := securityFixture(t)
	store := NewPgAccountStore(testDB, e2eEmitter)
	deletion, err := store.MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest)
	if err != nil {
		t.Fatal(err)
	}
	f, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: sess.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	expireDeletion(t, ctx, scope)
	_, err = s.Flow(ctx, scope, f.FlowToken, domain.SecurityFlowInput{Action: "cancel_deletion", RequestID: deletion.RequestID, Version: f.Version}, "submit")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("late cancellation: %v", err)
	}
}

func TestSecurityReliabilityConcurrentExchange(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
	input := domain.SecurityFlowInput{ContinuationToken: reliabilityContinuation(t, ctx, s, scope), ExchangeKey: newUUID()}
	var wait sync.WaitGroup
	result := make([]*domain.SecurityFlowState, 2)
	failures := make([]error, 2)
	for i := range result {
		wait.Go(func() { result[i], failures[i] = s.Start(ctx, scope, input, "exchange") })
	}
	wait.Wait()
	for _, err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if result[0].FlowToken != result[1].FlowToken {
		t.Fatal("duplicate exchange created two flows")
	}
}
func TestSecurityReliabilityExpiredCodeCannotBeRetried(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
	if _, err := s.Start(ctx, scope, domain.SecurityFlowInput{}, "review"); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := testDB.Pool.QueryRow(ctx, `SELECT id FROM iam_security_deliveries WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, scope.AccountID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_security_deliveries SET data=jsonb_set(data,'{created_at}',to_jsonb($1::timestamptz)) WHERE id=$2`, time.Now().Add(-11*time.Minute), id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RetryDelivery(ctx, scope, id); !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("expired code retried: %v", err)
	}
}
func TestSecurityReliabilityClosedCaseRetention(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
	policy, err := s.Policy(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	policy.RetentionDays = 1
	if _, err := s.SetPolicy(ctx, scope, *policy); err != nil {
		t.Fatal(err)
	}
	reliabilityContinuation(t, ctx, s, scope)
	var id string
	if err := testDB.Pool.QueryRow(ctx, `SELECT id FROM iam_security_cases WHERE user_id=$1`, scope.AccountID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_flows SET expires_at=$1 WHERE user_id=$2`, time.Now().Add(-time.Hour), scope.AccountID); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_security_cases SET status='completed',created_at=$1 WHERE id=$2`, old, id); err != nil {
		t.Fatal(err)
	}
	testDB.gcSecurityRetention(ctx, xlog.NewJSON())
	if _, err := s.Case(ctx, scope, id); err != nil {
		t.Fatalf("recently closed case removed based on creation: %v", err)
	}
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_security_cases SET data=jsonb_set(data,'{updated_at}',to_jsonb($1::timestamptz)) WHERE id=$2`, old, id); err != nil {
		t.Fatal(err)
	}
	testDB.gcSecurityRetention(ctx, xlog.NewJSON())
	var count int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_security_cases WHERE id=$1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("closed case retained past retention")
	}
}

func TestSecurityReliabilityRecoveryInvalidatesSignInProof(t *testing.T) {
	ctx, s, scope, session := securityFixture(t)
	policy, err := s.Policy(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	policy.RequireNewDeviceProof = true
	if _, err := s.SetPolicy(ctx, scope, *policy); err != nil {
		t.Fatal(err)
	}
	account, err := s.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	auth := NewPgCoreAuth(testDB, NewSecurityEmitter(testDB, e2eEmitter), nil)
	_, err = withTxRet(ctx, testDB, func(ctx context.Context) (*domain.Session, error) {
		return auth.coreAuthMintSession(ctx, account, "", []string{"pwd"}, 1)
	})
	var denial *domain.Error
	if !errors.As(err, &denial) || denial.Code != domain.ErrStepUpRequired.Code {
		t.Fatalf("expected sign-in proof: %v", err)
	}
	token, ok := denial.Details["security_flow_token"].(string)
	if !ok {
		t.Fatal("missing sign-in proof")
	}
	proof, err := s.Flow(ctx, scope, token, domain.SecurityFlowInput{}, "get")
	if err != nil {
		t.Fatal(err)
	}
	proof = securitySubmit(t, ctx, s, scope, proof, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	recovery, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: session.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	recovery = securitySubmit(t, ctx, s, scope, recovery, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	recovery = securitySubmit(t, ctx, s, scope, recovery, domain.SecurityFlowInput{Action: "report_activity"})
	securitySubmit(t, ctx, s, scope, recovery, domain.SecurityFlowInput{Action: "set_password", NewPassword: "N3wSecur3!Pass123"})
	err = testDB.withTx(ctx, func(ctx context.Context) error {
		return s.consumeSignInProof(ctx, scope, scope.AccountID, proof.FlowToken)
	})
	if !errors.Is(err, domain.ErrFlowExpired) {
		t.Fatalf("old sign-in proof survived recovery: %v", err)
	}
}
