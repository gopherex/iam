//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

func securityFixture(t *testing.T) (context.Context, *pgSecurity, domain.SecurityScope, *domain.Session) {
	t.Helper()
	ctx := context.Background()
	project := e2eProject(t, ctx)
	a, sess := registerUser(t, ctx, project, "security-"+newUUID()+"@example.com")
	a.EmailVerified = true
	raw, _ := json.Marshal(a)
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_users SET data=$1 WHERE id=$2`, raw, a.ID); err != nil {
		t.Fatal(err)
	}
	scope := domain.SecurityScope{ProjectID: project, Environment: "live", AccountID: a.ID, SessionID: sess.ID}
	store := NewPgSecurity(testDB, e2eEmitter)
	p := securityDefaultPolicy()
	p.Mode = "enforce"
	p.ContinueURL = "https://app.example.com/security"
	if _, err := store.SetPolicy(ctx, scope, p); err != nil {
		t.Fatal(err)
	}
	return ctx, store, scope, sess
}

func TestSecurityConcurrentProtection(t *testing.T) {
	ctx, store, scope, session := securityFixture(t)
	flow, err := store.Start(ctx, scope, domain.SecurityFlowInput{SessionID: session.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	input := domain.SecurityFlowInput{Action: "report_activity", AllSessions: true, Version: flow.Version}
	results := make([]*domain.SecurityFlowState, 2)
	failures := make([]error, 2)
	var wait sync.WaitGroup
	for i := range results {
		wait.Go(func() { results[i], failures[i] = store.Flow(ctx, scope, flow.FlowToken, input, "submit") })
	}
	wait.Wait()
	for _, err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if results[0].FlowToken != results[1].FlowToken || results[0].Version != flow.Version+1 {
		t.Fatal("concurrent retry ran protection twice")
	}
}

func TestSecurityDeviceTrustIsExplicit(t *testing.T) {
	ctx, store, scope, _ := securityFixture(t)
	device, err := store.RegisterDevice(ctx, scope, "Test browser")
	if err != nil {
		t.Fatal(err)
	}
	if device.Device.TrustedUntil != nil {
		t.Fatal("registration silently trusted the device")
	}
	flow, err := store.Start(ctx, scope, domain.SecurityFlowInput{DeviceID: device.Device.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	if len(flow.NextActions) != 2 || flow.NextActions[0] != "trust_device" {
		t.Fatal(flow.NextActions)
	}
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "trust_device"})
	if flow.Outcome != "device_trusted" {
		t.Fatal(flow.Outcome)
	}
	var trusted bool
	if err := testDB.Pool.QueryRow(ctx, `SELECT data->>'trusted_until' IS NOT NULL FROM iam_security_devices WHERE id=$1`, device.Device.ID).Scan(&trusted); err != nil {
		t.Fatal(err)
	}
	if !trusted {
		t.Fatal("trust not persisted")
	}
}

func TestSecuritySignInProof(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
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
	ca := NewPgCoreAuth(testDB, NewSecurityEmitter(testDB, e2eEmitter), nil)
	mint := func(ctx context.Context) (*domain.Session, error) {
		return withTxRet(ctx, testDB, func(ctx context.Context) (*domain.Session, error) {
			return ca.coreAuthMintSession(ctx, account, "", []string{"pwd"}, 1)
		})
	}
	_, err = mint(ctx)
	var denial *domain.Error
	if !errors.As(err, &denial) || denial.Code != domain.ErrStepUpRequired.Code {
		t.Fatalf("expected proof: %v", err)
	}
	token, ok := denial.Details["security_flow_token"].(string)
	if !ok {
		t.Fatal("missing proof capability")
	}
	flow, err := s.Flow(ctx, scope, token, domain.SecurityFlowInput{}, "get")
	if err != nil {
		t.Fatal(err)
	}
	before := flow.FlowToken
	input := domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code, Version: flow.Version}
	flow = securitySubmit(t, ctx, s, scope, flow, input)
	if flow.Outcome != "sign_in_approved" || flow.FlowToken == before {
		t.Fatalf("proof/rotation: %+v", flow)
	}
	replay, err := s.Flow(ctx, scope, before, input, "submit")
	if err != nil || replay.FlowToken != flow.FlowToken {
		t.Fatalf("lost response retry: %v", err)
	}
	if _, err := s.Flow(ctx, scope, before, domain.SecurityFlowInput{}, "get"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("old token remains readable: %v", err)
	}
	ctx = domain.WithRequestMeta(ctx, domain.RequestMeta{SecurityProof: flow.FlowToken})
	if _, err := mint(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := mint(ctx); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("proof reused: %v", err)
	}
}

func TestSecurityGuardAndEnvironment(t *testing.T) {
	ctx, s, scope, session := securityFixture(t)
	flow, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: session.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	foreign := scope
	foreign.Environment = "test"
	if _, err := s.Flow(api.WithEnvironment(ctx, "test"), foreign, flow.FlowToken, domain.SecurityFlowInput{}, "get"); err == nil {
		t.Fatal("cross-environment capability accepted")
	}
	flow = securitySubmit(t, ctx, s, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	flow = securitySubmit(t, ctx, s, scope, flow, domain.SecurityFlowInput{Action: "report_activity", AllSessions: true})
	if err := securityAccountGuard(ctx, testDB, scope.ProjectID, scope.AccountID); !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("guard: %v", err)
	}
	ca := NewPgCoreAuth(testDB, NewSecurityEmitter(testDB, e2eEmitter), nil)
	err = ca.ChangePassword(ctx, domain.CoreAuthPasswordChangeCmd{AccountID: scope.AccountID, CurrentPassword: "Sup3rStr0ng!Pass", NewPassword: "ChangedPass123!"})
	if !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("legacy password change bypassed guard: %v", err)
	}
	if _, err := s.Flow(ctx, scope, flow.FlowToken, domain.SecurityFlowInput{Action: "abandon", Version: flow.Version}, "submit"); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("protected flow abandoned: %v", err)
	}
	flow = securitySubmit(t, ctx, s, scope, flow, domain.SecurityFlowInput{Action: "set_password", NewPassword: "FinishedPass123!"})
	if err := securityAccountGuard(ctx, testDB, scope.ProjectID, scope.AccountID); err != nil {
		t.Fatal(err)
	}
}

func TestSecurityDetectionModesAndDelivery(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
	p, err := s.Policy(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	p.Mode = "observe"
	p.Notify = true
	if _, err := s.SetPolicy(ctx, scope, *p); err != nil {
		t.Fatal(err)
	}
	ctx = domain.WithRequestMeta(ctx, domain.RequestMeta{IP: "203.0.113.9", UserAgent: "security-test"})
	for range p.FailureThreshold {
		if err := s.recordFailure(ctx, scope.ProjectID, scope.AccountID, "password_failures"); err != nil {
			t.Fatal(err)
		}
	}
	var incidents, deliveries int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_security_incidents WHERE project_id=$1`, scope.ProjectID).Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 {
		t.Fatalf("incidents %d", incidents)
	}
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_security_deliveries WHERE project_id=$1`, scope.ProjectID).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	if deliveries != 0 {
		t.Fatal("observe sent a notification")
	}
	p.Mode = "enforce"
	if _, err := s.SetPolicy(ctx, scope, *p); err != nil {
		t.Fatal(err)
	}
	private := securityIncidentPrivate{Email: "security-target@example.com"}
	err = s.db.withTx(ctx, func(ctx context.Context) error {
		return s.recordIncident(ctx, scope, "token.reuse_detected", "", "", "high", "blocked", []string{"refresh_reuse"}, private, p)
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := securityLastPayload(t, ctx, scope, "security_alert")
	if !strings.Contains(payload.Link, "#iam_security=") || strings.Contains(payload.Link, "?iam_security=") {
		t.Fatal(payload.Link)
	}
	continuation, err := s.Start(ctx, scope, domain.SecurityFlowInput{ContinuationToken: payload.Continuation}, "exchange")
	if err != nil {
		t.Fatal(err)
	}
	if continuation.Step != "verify_identity" {
		t.Fatal("link alone proved identity")
	}
	if _, err := s.Start(ctx, scope, domain.SecurityFlowInput{ContinuationToken: payload.Continuation}, "exchange"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("continuation replay: %v", err)
	}
}

func securityLastPayload(t *testing.T, ctx context.Context, scope domain.SecurityScope, template string) securityDeliveryPayload {
	t.Helper()
	rows, err := testDB.Pool.Query(ctx, `SELECT private_data FROM iam_security_deliveries WHERE project_id=$1 AND environment=$2 ORDER BY created_at DESC`, scope.ProjectID, scope.Environment)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var p securityDeliveryPayload
		if err := NewPgSecurity(testDB, nil).decrypt(raw, &p); err != nil {
			t.Fatal(err)
		}
		if p.Template == template {
			return p
		}
	}
	t.Fatalf("no %s delivery", template)
	return securityDeliveryPayload{}
}

func securitySubmit(t *testing.T, ctx context.Context, s *pgSecurity, scope domain.SecurityScope, f *domain.SecurityFlowState, in domain.SecurityFlowInput) *domain.SecurityFlowState {
	t.Helper()
	in.Version = f.Version
	out, err := s.Flow(ctx, scope, f.FlowToken, in, "submit")
	if err != nil {
		t.Fatalf("submit %s: %v", in.Action, err)
	}
	return out
}

func TestSecuritySelectiveRecovery(t *testing.T) {
	for _, all := range []bool{false, true} {
		t.Run(fmt.Sprint(all), func(t *testing.T) {
			ctx, s, scope, first := securityFixture(t)
			ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
			a, err := s.account(ctx, scope)
			if err != nil {
				t.Fatal(err)
			}
			second, err := withTxRet(ctx, testDB, func(ctx context.Context) (*domain.Session, error) {
				return ca.coreAuthMintSession(ctx, a, "", []string{"pwd"}, 1)
			})
			if err != nil {
				t.Fatal(err)
			}
			f, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: first.ID}, "review")
			if err != nil {
				t.Fatal(err)
			}
			code := securityLastPayload(t, ctx, scope, "security_code").Code
			wrong := securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "verify_code", Code: "wrong"})
			if wrong.AttemptsLeft != 4 || wrong.ErrorCode != "invalid_code" {
				t.Fatalf("bad proof did not consume attempt: %+v", wrong)
			}
			f = securitySubmit(t, ctx, s, scope, wrong, domain.SecurityFlowInput{Action: "verify_code", Code: code})
			if f.Step != "review" {
				t.Fatalf("step %s", f.Step)
			}
			f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "report_activity", AllSessions: all})
			if f.Step != "restore_access" || f.AllSessions != all {
				t.Fatalf("protection selection lost: %+v", f)
			}
			f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "set_password", NewPassword: "N3wSecur3!Pass123"})
			if f.Status != "completed" || f.Outcome != "account_secured" {
				t.Fatalf("not completed: %+v", f)
			}
			var count int
			if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_sessions WHERE user_id=$1`, scope.AccountID).Scan(&count); err != nil {
				t.Fatal(err)
			}
			want := 1
			if all {
				want = 0
			}
			if count != want {
				t.Fatalf("sessions=%d want=%d", count, want)
			}
			introspection, err := ca.Introspect(ctx, scope.ProjectID, second.AccessToken)
			if err != nil {
				t.Fatal(err)
			}
			if introspection.Active == all {
				t.Fatalf("unrelated session active=%v all=%v", introspection.Active, all)
			}
			b, _ := json.Marshal(f)
			if strings.Contains(string(b), "access_token") || strings.Contains(string(b), "refresh_token") {
				t.Fatal("recovery issued ordinary tokens")
			}
		})
	}
}

func TestSecurityScopeAndReplay(t *testing.T) {
	ctx, s, scope, sess := securityFixture(t)
	f, err := s.Start(ctx, scope, domain.SecurityFlowInput{SessionID: sess.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	other := scope
	other.ProjectID = e2eProject(t, ctx)
	if _, err := s.Flow(ctx, other, f.FlowToken, domain.SecurityFlowInput{}, "get"); !errors.Is(err, domain.ErrFlowNotFound) {
		t.Fatalf("foreign project: %v", err)
	}
	code := securityLastPayload(t, ctx, scope, "security_code").Code
	prior := f.Version
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "verify_code", Code: code})
	if _, err := s.Flow(ctx, scope, f.FlowToken, domain.SecurityFlowInput{Action: "report_activity", Version: prior}, "submit"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale action: %v", err)
	}
	f = securitySubmit(t, ctx, s, scope, f, domain.SecurityFlowInput{Action: "confirm_activity"})
	if f.Outcome != "activity_confirmed" {
		t.Fatal(f.Outcome)
	}
	var count int
	_ = testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_sessions WHERE id=$1`, sess.ID).Scan(&count)
	if count != 1 {
		t.Fatal("confirm revoked session")
	}
}

func TestSecuritySupportGrant(t *testing.T) {
	ctx, s, scope, _ := securityFixture(t)
	a, err := s.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	public := scope
	public.AccountID = ""
	public.SessionID = ""
	f, err := s.Start(ctx, public, domain.SecurityFlowInput{Identifier: a.PrimaryEmail, Contact: "new-contact@example.com"}, "recovery")
	if err != nil {
		t.Fatal(err)
	}
	code := securityLastPayload(t, ctx, scope, "security_code").Code
	f = securitySubmit(t, ctx, s, public, f, domain.SecurityFlowInput{Action: "verify_code", Code: code, Message: "Lost all factors"})
	if f.Step != "awaiting_support" || f.Case == nil {
		t.Fatalf("missing case: %+v", f)
	}
	admin := public
	admin.ActorID = "test-reviewer"
	_, err = s.DecideCase(ctx, admin, f.Case.ID, domain.SecurityCaseDecision{Action: "approve", Message: "Ownership verified", Evidence: "Independent verification case 123"})
	if err != nil {
		t.Fatal(err)
	}
	token := securityLastPayload(t, ctx, scope, "security_recovery").Continuation
	grant, err := s.Start(ctx, public, domain.SecurityFlowInput{ContinuationToken: token}, "exchange")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(ctx, public, domain.SecurityFlowInput{ContinuationToken: token}, "exchange"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("grant replay: %v", err)
	}
	grant = securitySubmit(t, ctx, s, public, grant, domain.SecurityFlowInput{Action: "report_activity", AllSessions: true})
	grant = securitySubmit(t, ctx, s, public, grant, domain.SecurityFlowInput{Action: "set_password", NewPassword: "Restor3d!Secure123"})
	if grant.Outcome != "account_secured" {
		t.Fatal(grant.Outcome)
	}
	a, err = s.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if a.PrimaryEmail != "new-contact@example.com" || !a.EmailVerified {
		t.Fatal("support contact not installed")
	}
}

func TestSecurityHTTPContract(t *testing.T) {
	ctx, s, scope, sess := securityFixture(t)
	_ = s
	ts := e2eServer(t)
	headers := e2eBearer(sess.AccessToken)
	headers["X-Client-Id"] = scope.ProjectID
	r := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/security/flows", map[string]any{"session_id": sess.ID}, headers)
	e2eWantStatus(t, r, http.StatusOK)
	var f domain.SecurityFlowState
	if err := json.Unmarshal(r.Body, &f); err != nil {
		t.Fatal(err)
	}
	if f.Step != "verify_identity" || f.FlowToken == "" {
		t.Fatalf("bad response: %s", r.Body)
	}
	r = e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/security/flows/current", nil, map[string]string{"X-Client-Id": scope.ProjectID, "X-Security-Token": f.FlowToken})
	e2eWantStatus(t, r, http.StatusOK)
	r = e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/security/incidents", nil, headers)
	e2eWantStatus(t, r, http.StatusOK)
}

func TestSecurityRestoresVerifiedContacts(t *testing.T) {
	ctx, store, scope, session := securityFixture(t)
	original, err := store.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	flow, err := store.Start(ctx, scope, domain.SecurityFlowInput{SessionID: session.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_users SET primary_email='attacker@example.com',
 primary_phone='+15555550123',data=data || '{"PrimaryEmail":"attacker@example.com","EmailVerified":true,
 "PrimaryPhone":"+15555550123","PhoneVerified":true}'::jsonb WHERE id=$1`, scope.AccountID); err != nil {
		t.Fatal(err)
	}
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "report_activity"})
	securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "set_password", NewPassword: "RestoredPass123!"})
	restored, err := store.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if restored.PrimaryEmail != original.PrimaryEmail || !restored.EmailVerified || restored.PrimaryPhone != "" || restored.PhoneVerified {
		t.Fatalf("contacts not restored: %+v", restored)
	}
}

func TestSecurityDirectReviewRequiresExistingMFA(t *testing.T) {
	ctx, store, scope, session := securityFixture(t)
	account, err := store.account(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	factor := e2eActiveEmailFactor(t, ctx, scope.ProjectID, scope.AccountID, account.PrimaryEmail)
	flow, err := store.Start(ctx, scope, domain.SecurityFlowInput{SessionID: session.ID}, "review")
	if err != nil {
		t.Fatal(err)
	}
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	if flow.Step != "verify_mfa" {
		t.Fatalf("existing MFA bypassed: %s", flow.Step)
	}
	if _, err := store.Flow(ctx, scope, flow.FlowToken, domain.SecurityFlowInput{Action: "report_activity", Version: flow.Version}, "submit"); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("report before MFA: %v", err)
	}
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "select_factor", FactorID: factor})
	flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "verify_mfa", FactorID: factor, Code: securityLastPayload(t, ctx, scope, "security_code").Code})
	if flow.Step != "review" {
		t.Fatal(flow.Step)
	}
}

func TestSecuritySupportKeepsSelectedScope(t *testing.T) {
	for _, all := range []bool{false, true} {
		t.Run(fmt.Sprint(all), func(t *testing.T) {
			ctx, store, scope, session := securityFixture(t)
			account, err := store.account(ctx, scope)
			if err != nil {
				t.Fatal(err)
			}
			coreAuth := NewPgCoreAuth(testDB, e2eEmitter, nil)
			other, err := withTxRet(ctx, testDB, func(ctx context.Context) (*domain.Session, error) {
				return coreAuth.coreAuthMintSession(ctx, account, "", nil, 1)
			})
			if err != nil {
				t.Fatal(err)
			}
			public := scope
			public.AccountID = ""
			public.SessionID = ""
			// Open the support case before protection, exercising reuse of its private scope.
			initial, err := store.Start(ctx, public, domain.SecurityFlowInput{Identifier: account.PrimaryEmail, Contact: account.PrimaryEmail}, "recovery")
			if err != nil {
				t.Fatal(err)
			}
			initial = securitySubmit(t, ctx, store, public, initial, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
			flow, err := store.Start(ctx, scope, domain.SecurityFlowInput{SessionID: session.ID}, "review")
			if err != nil {
				t.Fatal(err)
			}
			flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
			flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "report_activity", AllSessions: all})
			flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "request_support", Message: "Cannot complete recovery"})
			if flow.Case == nil || flow.Case.ID != initial.Case.ID {
				t.Fatal("existing case not resumed")
			}
			admin := public
			admin.ActorID = "support-reviewer"
			if _, err := store.DecideCase(ctx, admin, flow.Case.ID, domain.SecurityCaseDecision{Action: "approve", Message: "Verified", Evidence: "Independent verification"}); err != nil {
				t.Fatal(err)
			}
			grant, err := store.Start(ctx, public, domain.SecurityFlowInput{ContinuationToken: securityLastPayload(t, ctx, scope, "security_recovery").Continuation}, "exchange")
			if err != nil {
				t.Fatal(err)
			}
			if grant.Step != "restore_access" || grant.AllSessions != all {
				t.Fatalf("scope changed: %+v", grant)
			}
			securitySubmit(t, ctx, store, public, grant, domain.SecurityFlowInput{Action: "set_password", NewPassword: "SupportRestored123!"})
			var remains bool
			if err := testDB.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam_sessions WHERE id=$1)`, other.ID).Scan(&remains); err != nil {
				t.Fatal(err)
			}
			if remains == all {
				t.Fatal("support changed session selection")
			}
		})
	}
}

func TestSecurityDeviceRevokeUsesActualSessions(t *testing.T) {
	ctx, store, scope, _ := securityFixture(t)
	registered, err := store.RegisterDevice(ctx, scope, "Device")
	if err != nil {
		t.Fatal(err)
	}
	// Display history may have aged out this session. Revocation still uses session ownership.
	if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_security_devices SET data=jsonb_set(data,'{session_ids}','[]'::jsonb) WHERE id=$1`, registered.Device.ID); err != nil {
		t.Fatal(err)
	}
	revoked, err := store.UpdateDevice(ctx, scope, registered.Device.ID, "revoke", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !revoked.Current || !revoked.Revoked {
		t.Fatal("current device was not identified")
	}
	var remaining bool
	if err := testDB.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam_sessions WHERE id=$1)`, scope.SessionID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining {
		t.Fatal("device retained session outside display history")
	}
}
