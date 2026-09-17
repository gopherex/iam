package postgres

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

type securityFlow struct {
	Contacts          securityIncidentPrivate  `json:"contacts"`
	RecoveryPhone     string                   `json:"recovery_phone"`
	TargetDeviceID    string                   `json:"target_device_id"`
	CurrentToken      string                   `json:"current_token"`
	PreviousTokenHash string                   `json:"previous_token_hash"`
	PreviousInputHash string                   `json:"previous_input_hash"`
	PreviousUntil     time.Time                `json:"previous_until"`
	Protected         bool                     `json:"protected"`
	FactorID          string                   `json:"factor_id"`
	FactorRecipient   string                   `json:"factor_recipient"`
	WebAuthnSession   json.RawMessage          `json:"webauthn_session"`
	Purpose           string                   `json:"purpose"`
	ID                string                   `json:"id"`
	ProjectID         string                   `json:"project_id"`
	Environment       string                   `json:"environment"`
	AccountID         string                   `json:"account_id"`
	TargetSession     string                   `json:"target_session"`
	Recipient         string                   `json:"recipient"` // encrypted, including in iam_flows.data
	Identifier        string                   `json:"identifier"`
	CaseID            string                   `json:"case_id"`
	IncidentID        string                   `json:"incident_id"`
	CodeHash          string                   `json:"code_hash"`
	CodeExpires       time.Time                `json:"code_expires"`
	Cutoff            time.Time                `json:"cutoff"`
	Proved            bool                     `json:"proved"`
	SupportGrant      bool                     `json:"support_grant"`
	MFARequired       bool                     `json:"mfa_required"`
	ContactVerified   bool                     `json:"contact_verified"`
	State             domain.SecurityFlowState `json:"state"`
}

type securityIncidentPrivate struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	MFARequired bool   `json:"mfa_required"`
}

func securityFlowScope(f *securityFlow) domain.SecurityScope {
	return domain.SecurityScope{
		ProjectID:   f.ProjectID,
		Environment: f.Environment,
		AccountID:   f.AccountID,
	}
}

func (s *pgSecurity) newFlow(
	ctx context.Context,
	scope domain.SecurityScope,
	kind string,
) (*securityFlow, string, error) {
	p, err := s.Policy(ctx, scope)
	if err != nil {
		return nil, "", securityStoreError(err)
	}

	if p.Mode == securityDisabled && kind != deletionConfigKey {
		return nil, "", domain.ErrForbidden.WithMessage("Account security is disabled")
	}

	token, err := accountRandomToken(securityTokenBytes)
	if err != nil {
		return nil, "", securityStoreError(err)
	}

	token = securityFlowTokenPrefix + token
	f := &securityFlow{
		ID:          newUUID(),
		ProjectID:   scope.ProjectID,
		Environment: scope.Environment,
		AccountID:   scope.AccountID,
		Cutoff:      nowIn(ctx),
		State: domain.SecurityFlowState{
			Kind:              "security_" + kind,
			Status:            securityPending,
			Step:              securityVerifyContact,
			Version:           1,
			NextActions:       []string{},
			AttemptsLeft:      securityCodeAttempts,
			ExpiresAt:         nowIn(ctx).Add(time.Duration(p.FlowTTLSeconds) * time.Second),
			RevokedSessionIDs: []string{},
		},
	}

	return f, token, nil
}

func (s *pgSecurity) saveFlow(ctx context.Context, f *securityFlow, token string) error {
	f.CurrentToken = token
	f.State.FlowToken = ""
	f.State.Sessions = nil
	f.State.Factors = nil

	secret, err := s.encrypt(f)
	if err != nil {
		return securityStoreError(err)
	}

	raw, err := json.Marshal(
		map[string]string{
			"security_payload":    secret,
			"previous_token_hash": f.PreviousTokenHash,
		})
	if err != nil {
		return securityStoreError(err)
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`INSERT INTO iam_flows(id,project_id,environment,token_hash,kind,status,step,user_id,expires_at,data)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	ON CONFLICT(id) DO UPDATE SET
token_hash=EXCLUDED.token_hash,status=EXCLUDED.status,step=EXCLUDED.step,data=EXCLUDED.data,updated_at=now()`,
		f.ID,
		f.ProjectID,
		f.Environment,
		accountHashToken(token),
		f.State.Kind,
		f.State.Status,
		f.State.Step,
		f.AccountID,
		f.State.ExpiresAt,
		raw)

	return securityStoreError(err)
}

func (s *pgSecurity) loadFlow(ctx context.Context, scope domain.SecurityScope, token string) (*securityFlow, error) {
	if !strings.HasPrefix(token, securityFlowTokenPrefix) {
		return nil, domain.ErrFlowNotFound
	}

	var raw []byte

	err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_flows WHERE project_id=$1 AND environment=$2 AND (token_hash=$3
OR data->>'previous_token_hash'=$3) AND kind IN ('security_review','security_recovery','security_account_deletion')`,
		scope.ProjectID,
		scope.Environment,
		accountHashToken(token)).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrFlowNotFound
	}

	if err != nil {
		return nil, securityStoreError(err)
	}

	var envelope map[string]string
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, securityStoreError(err)
	}

	var f securityFlow
	if err := s.decrypt(envelope["security_payload"], &f); err != nil {
		return nil, securityStoreError(err)
	}

	if f.ID == "" || f.ProjectID != scope.ProjectID || f.Environment != scope.Environment {
		return nil, domain.ErrFlowNotFound
	}

	if !nowIn(ctx).Before(f.State.ExpiresAt) {
		return nil, domain.ErrFlowExpired
	}

	return &f, nil
}

func (s *pgSecurity) account(ctx context.Context, scope domain.SecurityScope) (*domain.Account, error) {
	var raw []byte

	err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_users WHERE id=$1 AND project_id=$2 AND environment=$3`,
		scope.AccountID,
		scope.ProjectID,
		scope.Environment).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, securityStoreError(err)
	}

	var a domain.Account
	if err := json.Unmarshal(raw, &a); err != nil { //nolint:musttag // Preserve stored field names.
		return nil, securityStoreError(err)
	}

	return &a, nil
}

//nolint:funlen,cyclop,nestif // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) startReview(
	ctx context.Context,
	scope domain.SecurityScope,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	a, err := s.account(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	f.Contacts = securityVerifiedContacts(a)

	if in.IncidentID != "" {
		var (
			inc     domain.SecurityIncident
			private securityIncidentPrivate
		)

		if err := s.load(ctx, scope, securityIncidents, in.IncidentID, &inc, &private); err != nil {
			return securityStoreError(err)
		}

		if inc.Status == "resolved" {
			return domain.ErrConflict.WithMessage("Incident is already resolved")
		}

		f.Contacts = private
		f.IncidentID = inc.ID
		f.TargetSession = inc.SessionID
		f.Cutoff = inc.CreatedAt

		f.Recipient = private.Email
		if f.Recipient == "" {
			f.Recipient = private.Phone
		}

		f.MFARequired = private.MFARequired
	} else {
		f.TargetSession = in.SessionID
		if a.EmailVerified {
			f.Recipient = a.PrimaryEmail
		} else if a.PhoneVerified {
			f.Recipient = a.PrimaryPhone
		}
	}

	if f.TargetSession != "" && f.IncidentID == "" {
		var exists bool

		err := s.db.TxDB.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM iam_sessions WHERE id=$1 AND project_id=$2 AND environment=$3
AND user_id=$4)`,
			f.TargetSession,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID).Scan(&exists)
		if err != nil {
			return securityStoreError(err)
		}

		if !exists {
			return domain.ErrSessionNotFound
		}
	}

	if in.DeviceID != "" {
		var device domain.SecurityDevice
		if err := s.load(ctx, scope, securityDevices, in.DeviceID, &device, nil); err != nil {
			return securityStoreError(err)
		}

		if device.Revoked || !slices.Contains(device.SessionIDs, scope.SessionID) {
			return domain.ErrForbidden
		}

		f.Purpose = securityTrust
		f.TargetDeviceID = in.DeviceID
		f.TargetSession = scope.SessionID
	}

	activeMFA, err := s.priorMFARequired(ctx, f)
	if err != nil {
		return err
	}

	f.MFARequired = f.MFARequired || activeMFA
	f.State.Step = securityVerifyIdentity

	return s.issueCode(ctx, f)
}

func (s *pgSecurity) Start(
	ctx context.Context,
	scope domain.SecurityScope,
	in domain.SecurityFlowInput,
	mode string,
) (*domain.SecurityFlowState, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	return withTxRet(ctx, s.db, func(ctx context.Context) (*domain.SecurityFlowState, error) {
		return s.startSecurityFlow(ctx, scope, in, mode)
	})
}

//nolint:cyclop // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) startSecurityFlow(
	ctx context.Context, scope domain.SecurityScope, in domain.SecurityFlowInput, mode string,
) (*domain.SecurityFlowState, error) {
	subject := scope.AccountID
	if subject == "" {
		subject = domain.RequestMetaFromContext(ctx).IP
	}

	ok, err := s.throttle(
		ctx,
		scope,
		subject,
		"flow_start",
		securityFlowStartBurst,
		securityProofRateWindow)
	if err != nil {
		return nil, securityStoreError(err)
	}

	if !ok {
		return nil, domain.ErrRateLimited
	}

	if mode == securityExchangeMode {
		if in.ExchangeKey != "" && (len(in.ExchangeKey) < 32 || len(in.ExchangeKey) > 128) {
			return nil, domain.ErrValidation
		}

		if replay, err := s.replayContinuation(ctx, scope, in); !errors.Is(err, ErrNotFound) {
			return replay, err
		}
	}

	kind := securityReview
	if mode == securityRecovery {
		kind = securityRecovery
	}

	f, token, err := s.newFlow(ctx, scope, kind)
	if err != nil {
		return nil, securityStoreError(err)
	}

	switch mode {
	case securityReview:
		if scope.AccountID == "" {
			return nil, domain.ErrUnauthorized
		}

		if err := s.startReview(ctx, scope, f, in); err != nil {
			return nil, securityStoreError(err)
		}
	case securityRecovery:
		if err := s.startRecoveryContact(ctx, f, in); err != nil {
			return nil, securityStoreError(err)
		}
	case securityExchangeMode:
		if err := s.exchange(ctx, scope, f, in.ContinuationToken); err != nil {
			return nil, securityStoreError(err)
		}
	default:
		return nil, domain.ErrBadRequest
	}

	if err := s.saveFlow(ctx, f, token); err != nil {
		return nil, securityStoreError(err)
	}

	if mode == securityExchangeMode && in.ExchangeKey != "" {
		if err := s.rememberContinuation(ctx, scope, in, token); err != nil {
			return nil, err
		}
	}

	return s.viewFlow(ctx, f, token)
}

//nolint:funlen,nestif // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) exchange(
	ctx context.Context,
	scope domain.SecurityScope,
	f *securityFlow,
	token string,
) error {
	if len(token) < securityTokenBytes {
		return domain.ErrInvalidToken
	}

	var purpose string

	err := s.db.TxDB.QueryRow(
		ctx,
		`UPDATE iam_security_continuations SET consumed=true WHERE token_hash=$1 AND project_id=$2
AND environment=$3 AND NOT consumed AND expires_at>$4 RETURNING user_id,incident_id,case_id,purpose`,
		accountHashToken(token),
		scope.ProjectID,
		scope.Environment,
		nowIn(ctx)).Scan(&f.AccountID, &f.IncidentID, &f.CaseID, &purpose)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidToken
	}

	if err != nil {
		return securityStoreError(err)
	}

	if purpose == securityReview {
		return s.startReview(
			ctx,
			securityFlowScope(f),
			f,
			domain.SecurityFlowInput{IncidentID: f.IncidentID})
	}

	var (
		c       domain.SecurityCase
		private securityCasePrivate
	)

	if err := s.load(ctx, scope, securityCases, f.CaseID, &c, &private); err != nil {
		return securityStoreError(err)
	}

	f.State.Kind = securityRecoveryTemplate
	f.Recipient = private.Contact

	if purpose == securitySupportGrant {
		if c.Status != securityApproved || c.AccountID == "" {
			return domain.ErrInvalidToken
		}

		f.AccountID = c.AccountID
		f.Proved = true

		f.SupportGrant = true
		if _, err := s.db.TxDB.Exec(
			ctx,
			`UPDATE iam_security_cases SET data=jsonb_set(data,'{grant_flow_id}',to_jsonb($1::text))
WHERE id=$2 AND project_id=$3 AND environment=$4`,
			f.ID,
			f.CaseID,
			f.ProjectID,
			f.Environment); err != nil {
			return securityStoreError(err)
		}

		f.State.Step = securityReview

		f.State.Case = &c
		if private.Protected {
			f.State.AllSessions = private.AllSessions
			f.TargetSession = private.TargetSession
			f.State.RevokedSessionIDs = private.RevokedSessionIDs

			f.State.Step = securityRestoreAccess
			if _, err := s.db.TxDB.Exec(
				ctx,
				`UPDATE iam_security_guards SET flow_id=$1 WHERE project_id=$2 AND environment=$3 AND
user_id=$4`,
				f.ID,
				f.ProjectID,
				f.Environment,
				f.AccountID); err != nil {
				return securityStoreError(err)
			}
		}
	} else {
		f.State.Step = securityAwaitingSupport
		f.State.Case = &c
	}

	return nil
}

func (s *pgSecurity) issueCode(ctx context.Context, f *securityFlow) error {
	if f.Recipient == "" {
		f.State.Step = securityVerifyMFA
		return nil
	}

	code, err := coreAuthRandomCode()
	if err != nil {
		return securityStoreError(err)
	}

	f.CodeHash = accountHashToken(code)
	f.CodeExpires = nowIn(ctx).Add(securityCodeTTL)
	f.State.AttemptsLeft = securityCodeAttempts
	resend := nowIn(ctx).Add(time.Minute)
	f.State.ResendAt = &resend
	f.State.ContactMasked = securityMask(f.Recipient)

	return s.queueDelivery(
		ctx,
		securityFlowScope(f),
		securityDeliveryPayload{To: f.Recipient, Template: securityCodeTemplate, Code: code},
		f.IncidentID,
		f.CaseID,
		"code:"+f.ID+":"+newUUID())
}

//nolint:funlen,cyclop // Keep the server action table together for audit.
func (s *pgSecurity) viewFlow(ctx context.Context, f *securityFlow, token string) (*domain.SecurityFlowState, error) {
	v := f.State
	if v.Case != nil {
		c := *v.Case
		c.AccountID = ""
		v.Case = &c
	}

	v.FlowToken = token

	v.NextActions = []string{}
	if v.Status != securityPending {
		return &v, nil
	}

	switch v.Step {
	case securityVerifyContact, securityVerifyIdentity:
		v.NextActions = []string{
			"verify_code",
			securityResend,
			securityRequestSupport,
			securityAbandon,
		}
	case securityVerifyMFA:
		v.NextActions = []string{
			"select_factor",
			securityVerifyMFA,
			securityVerifyRecoveryCode,
			securityRequestSupport,
			securityAbandon,
		}

		factors, err := s.proofFactors(ctx, f)
		if err != nil {
			return nil, securityStoreError(err)
		}

		v.Factors = factors

	case "support_required":
		v.NextActions = []string{securityRequestSupport, securityAbandon}
	case securityReview:
		if isDeletionProof(f) {
			v.NextActions = []string{"authorize_deletion", securityAbandon}
			return &v, nil
		}

		if f.Purpose == securityTrust {
			v.NextActions = []string{"trust_device", securityAbandon}
			return &v, nil
		}

		if f.TargetSession == "" {
			sessions, err := s.reviewSessions(ctx, f)
			if err != nil {
				return nil, securityStoreError(err)
			}

			v.Sessions = sessions
		}

		v.NextActions = []string{"report_activity", securityAbandon}
		if !f.SupportGrant {
			v.NextActions = append(v.NextActions, "confirm_activity")
		}

		if f.IncidentID != "" {
			inc, err := s.Incident(ctx, securityFlowScope(f), f.IncidentID)
			if err != nil {
				return nil, securityStoreError(err)
			}

			v.Incident = inc
		}
	case securityRestoreAccess:
		actions, err := s.restoreActions(ctx, f)
		if err != nil {
			return nil, securityStoreError(err)
		}

		v.NextActions = actions
	case securityAwaitingSupport:
		v.NextActions = []string{"support_message", securityAbandon}

		c, err := s.Case(
			ctx,
			domain.SecurityScope{ProjectID: f.ProjectID, Environment: f.Environment},
			f.CaseID)
		if err != nil {
			return nil, securityStoreError(err)
		}

		v.Case = c

		v.Case.AccountID = ""
		if c.Status == "rejected" {
			v.Status = securityCompleted
			v.Outcome = "support_rejected"
			v.NextActions = []string{}
		}
	}

	return s.viewRecoveryDeletion(ctx, f, &v)
}

func (s *pgSecurity) Flow(
	ctx context.Context,
	scope domain.SecurityScope,
	token string,
	in domain.SecurityFlowInput,
	mode string,
) (*domain.SecurityFlowState, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	if mode == "get" {
		f, err := s.loadFlow(ctx, scope, token)
		if err != nil {
			return nil, securityStoreError(err)
		}

		if f.CurrentToken != token {
			return nil, domain.ErrConflict.WithMessage("Retry the last submitted action")
		}

		return s.viewFlow(ctx, f, token)
	}

	return withTxRet(ctx, s.db, func(ctx context.Context) (*domain.SecurityFlowState, error) {
		return s.mutateSecurityFlow(ctx, scope, token, in, mode)
	})
}

func (s *pgSecurity) advance(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	if in.Action == "cancel_deletion" {
		return s.cancelRecoveryDeletion(ctx, f, in.RequestID)
	}

	if in.Action == securityAbandon {
		if f.State.Step == securityRestoreAccess {
			return domain.ErrBadRequest
		}

		f.State.Status = "aborted"

		return nil
	}

	if in.Action == securityRequestSupport {
		return s.requestSupport(ctx, f, in)
	}

	switch f.State.Step {
	case securityVerifyIdentity, securityVerifyContact:
		return s.advanceContactProof(ctx, f, in)
	case securityVerifyMFA:
		return s.advanceFactorProof(ctx, f, in)
	case securityReview:
		return s.advanceReview(ctx, f, in)
	case securityRestoreAccess:
		return s.advanceRestore(ctx, f, in)
	case securityAwaitingSupport:
		return s.advanceSupportMessage(ctx, f, in)
	default:
		return domain.ErrBadRequest
	}
}

//nolint:funlen // Keep the ordered security transition together for audit.
func (s *pgSecurity) advanceContactProof(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	scope := securityFlowScope(f)

	if in.Action != "verify_code" {
		return domain.ErrBadRequest
	}

	if f.State.AttemptsLeft <= 0 || !nowIn(ctx).Before(f.CodeExpires) {
		f.State.ErrorCode = "challenge_expired"
		return nil
	}

	ok, err := s.throttle(
		ctx,
		scope,
		f.AccountID+":"+f.Recipient,
		"proof",
		securityProofBurst,
		securityProofRateWindow)
	if err != nil {
		return securityStoreError(err)
	}

	if !ok {
		f.State.ErrorCode = securityRateLimited
		return nil
	}

	if subtle.ConstantTimeCompare([]byte(accountHashToken(in.Code)), []byte(f.CodeHash)) != 1 {
		f.State.AttemptsLeft--
		f.State.ErrorCode = "invalid_code"

		return nil
	}

	f.CodeHash = ""
	f.ContactVerified = true

	f.State.ResendAt = nil
	if f.State.Step == securityVerifyContact {
		f.Proved = false
		return s.openCase(ctx, f, in.Message)
	}

	var active int

	err = s.db.TxDB.QueryRow(
		ctx,
		`SELECT count(*) FROM iam_factors WHERE project_id=$1 AND environment=$2 AND user_id=$3
AND status='active'`,
		f.ProjectID,
		f.Environment,
		f.AccountID).Scan(&active)
	if err != nil {
		return securityStoreError(err)
	}

	if active > 0 || f.MFARequired {
		f.MFARequired = true
		f.State.Step = securityVerifyMFA
		f.State.AttemptsLeft = securityCodeAttempts

		return nil
	}

	f.Proved = true

	f.State.Step = securityReview
	if f.Purpose == securitySignin {
		f.State.Status = securityCompleted
		f.State.Step = securityCompleted
		f.State.Outcome = securitySignInApproved
	}

	return nil
}

func (s *pgSecurity) advanceFactorProof(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	if in.Action == "select_factor" {
		return s.selectFactor(ctx, f, in.FactorID)
	}

	if in.Action != securityVerifyMFA && in.Action != securityVerifyRecoveryCode {
		return domain.ErrBadRequest
	}

	if f.State.AttemptsLeft <= 0 {
		f.State.ErrorCode = "attempts_exhausted"
		return nil
	}

	ok, err := s.verifyFactor(ctx, f, in)
	if err != nil {
		return securityStoreError(err)
	}

	if !ok {
		f.State.AttemptsLeft--
		f.State.ErrorCode = "invalid_factor"

		return nil
	}

	f.Proved = true

	f.State.Step = securityReview
	if f.Purpose == securitySignin {
		f.State.Status = securityCompleted
		f.State.Step = securityCompleted
		f.State.Outcome = securitySignInApproved
	}

	return nil
}

func (s *pgSecurity) advanceReview(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	if isDeletionProof(f) {
		return authorizeDeletionProof(f, in)
	}

	if f.Purpose == securityTrust {
		return s.trustReviewedDevice(ctx, f, in)
	}

	scope := securityFlowScope(f)
	if !f.Proved {
		return domain.ErrStepUpRequired
	}

	switch in.Action {
	case "confirm_activity":
		if f.SupportGrant {
			return domain.ErrBadRequest
		}

		return s.complete(ctx, f, "activity_confirmed")
	case "report_activity":
		if f.TargetSession == "" && !in.AllSessions {
			if err := s.selectReviewSession(ctx, f, in.SessionID); err != nil {
				return securityStoreError(err)
			}
		}

		f.State.AllSessions = in.AllSessions
		if err := s.protect(ctx, scope, f); err != nil {
			return securityStoreError(err)
		}

		f.State.Step = securityRestoreAccess
	default:
		return domain.ErrBadRequest
	}

	return nil
}

func (s *pgSecurity) advanceRestore(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	scope := securityFlowScope(f)

	actions, err := s.restoreActions(ctx, f)
	if err != nil {
		return securityStoreError(err)
	}

	if !slices.Contains(actions, in.Action) || !f.Proved {
		return domain.ErrBadRequest
	}

	if in.Action != securitySetPassword {
		return s.restoreWithoutPassword(ctx, f, in)
	}

	coreAuth := NewPgCoreAuth(s.db, s.emitter, nil)
	if err := coreAuth.coreAuthEnforcePasswordPolicy(ctx, f.ProjectID, in.NewPassword); err != nil {
		return securityStoreError(err)
	}

	hash, err := coreAuthHashPassword(in.NewPassword)
	if err != nil {
		return securityStoreError(err)
	}

	if err := coreAuth.coreAuthUpsertPasswordCredential(
		api.WithEnvironment(ctx, f.Environment),
		f.ProjectID,
		f.AccountID,
		hash); err != nil {
		return securityStoreError(err)
	}

	if err := s.invalidateRecoveryCredentials(ctx, f); err != nil {
		return securityStoreError(err)
	}

	if err := s.restoreAccountContacts(ctx, f); err != nil {
		return securityStoreError(err)
	}

	if err := s.emit(
		ctx,
		scope,
		"security.credentials.restored",
		f.AccountID,
		map[string]any{securityAccountID: f.AccountID}); err != nil {
		return securityStoreError(err)
	}

	return s.complete(ctx, f, securityAccountSecured)
}

func (s *pgSecurity) advanceSupportMessage(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	if in.Action != "support_message" {
		return domain.ErrBadRequest
	}

	return s.caseMessage(ctx, f, in.Message)
}

func (s *pgSecurity) verifyFactor(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) (bool, error) {
	scope := securityFlowScope(f)

	ok, err := s.throttle(
		ctx,
		scope,
		f.AccountID,
		"mfa_proof",
		securityProofBurst,
		securityProofRateWindow)
	if err != nil || !ok {
		return false, securityStoreError(err)
	}

	if in.Action == securityVerifyRecoveryCode {
		var id string

		err = s.db.TxDB.QueryRow(
			ctx,
			`UPDATE iam_recovery_codes SET used=true WHERE project_id=$1 AND environment=$2 AND
user_id=$3 AND hash=$4 AND NOT used AND created_at<=$5 RETURNING id`,
			f.ProjectID,
			f.Environment,
			f.AccountID,
			accountHashToken(in.Code),
			f.Cutoff).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return err == nil, securityStoreError(err)
	}

	if f.FactorID != "" && (in.FactorID == securityPasskey || f.CodeHash != "") {
		return s.verifySelectedFactor(ctx, f, in)
	}

	var secret string

	err = s.db.TxDB.QueryRow(
		ctx,
		`SELECT secret FROM iam_factors WHERE id=$1 AND project_id=$2 AND environment=$3 AND
user_id=$4 AND type='totp' AND status='active' AND created_at<=$5`,
		in.FactorID,
		f.ProjectID,
		f.Environment,
		f.AccountID,
		f.Cutoff).Scan(&secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, securityStoreError(err)
	}

	secret, err = s.db.Cipher.Decrypt(secret)
	if err != nil {
		return false, securityStoreError(err)
	}

	if !totp.Validate(in.Code, secret) {
		return false, nil
	}

	return s.throttle(
		ctx,
		scope,
		f.AccountID+":"+in.FactorID+":"+in.Code,
		"used_totp",
		1,
		securityTOTPReplayWindow)
}

func (s *pgSecurity) complete(ctx context.Context, f *securityFlow, outcome string) error {
	scope := securityFlowScope(f)
	if outcome == securityAccountSecured {
		if _, err := s.db.TxDB.Exec(
			ctx,
			`UPDATE iam_users SET security_recovered_at=$1, data=data-'security_recovered_at'
WHERE id=$2 AND project_id=$3 AND environment=$4`,
			nowIn(ctx),
			f.AccountID,
			f.ProjectID,
			f.Environment); err != nil {
			return securityStoreError(err)
		}

		if _, err := s.db.TxDB.Exec(
			ctx,
			`DELETE FROM iam_security_guards WHERE project_id=$1 AND environment=$2 AND user_id=$3
AND flow_id=$4`,
			scope.ProjectID,
			scope.Environment,
			scope.AccountID,
			f.ID); err != nil {
			return securityStoreError(err)
		}
	}

	if f.IncidentID != "" {
		inc, err := s.Incident(ctx, scope, f.IncidentID)
		if err != nil {
			return securityStoreError(err)
		}

		inc.Status = "resolved"
		inc.Resolution = outcome

		inc.UpdatedAt = nowIn(ctx)
		if err := s.save(ctx, scope, securityIncidents, inc.ID, inc.Status, inc); err != nil {
			return securityStoreError(err)
		}
	}

	if f.CaseID != "" && f.SupportGrant {
		c, err := s.Case(ctx, scope, f.CaseID)
		if err != nil {
			return securityStoreError(err)
		}

		c.Status = securityCompleted

		c.UpdatedAt = nowIn(ctx)
		if err := s.save(ctx, scope, securityCases, c.ID, c.Status, c); err != nil {
			return securityStoreError(err)
		}
	}

	f.State.Status = securityCompleted
	f.State.Step = securityCompleted
	f.State.Outcome = outcome

	if err := s.notifyStatus(ctx, f, outcome); err != nil {
		return securityStoreError(err)
	}

	return s.emit(
		ctx,
		scope,
		"security.incident.resolved",
		f.IncidentID,
		map[string]any{
			securityAccountID: f.AccountID,
			"incident_id":     f.IncidentID,
			"resolution":      outcome,
		})
}

//nolint:gosec // Only an encrypted request digest is retained for idempotency.
func securityRequestHash(in domain.SecurityFlowInput, mode string) (string, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return "", securityStoreError(err)
	}

	return accountHashToken(mode + ":" + string(raw)), nil
}

func (s *pgSecurity) validateFlowAuthority(ctx context.Context, f *securityFlow) error {
	if f.SupportGrant {
		var valid bool
		if err := s.db.TxDB.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM iam_security_cases WHERE id=$1 AND project_id=$2 AND environment=$3
AND status='approved' AND data->>'grant_flow_id'=$4)`,
			f.CaseID,
			f.ProjectID,
			f.Environment,
			f.ID).Scan(&valid); err != nil {
			return securityStoreError(err)
		}

		if !valid {
			return domain.ErrInvalidToken
		}
	}

	if f.AccountID != "" {
		var recovered *time.Time
		if err := s.db.TxDB.QueryRow(
			ctx,
			`SELECT security_recovered_at FROM iam_users WHERE id=$1 AND
project_id=$2 AND environment=$3`,
			f.AccountID,
			f.ProjectID,
			f.Environment).Scan(&recovered); err != nil {
			return securityStoreError(err)
		}

		if recovered != nil && recovered.After(f.Cutoff) {
			return domain.ErrFlowExpired
		}
	}

	return nil
}

func (s *pgSecurity) proofFactors(ctx context.Context, f *securityFlow) ([]domain.SecurityProofFactor, error) {
	v := domain.SecurityFlowState{}

	rows, err := s.db.TxDB.Query(
		ctx,
		`SELECT id,type,data FROM iam_factors WHERE project_id=$1 AND environment=$2 AND user_id=$3
AND status='active' AND created_at<=$4`,
		f.ProjectID,
		f.Environment,
		f.AccountID,
		f.Cutoff)
	if err != nil {
		return nil, securityStoreError(err)
	}

	for rows.Next() {
		var (
			id, typ string
			raw     []byte
		)

		if err := rows.Scan(&id, &typ, &raw); err != nil {
			rows.Close()
			return nil, securityStoreError(err)
		}

		var factor domain.Factor
		if err := json.Unmarshal(raw, &factor); err != nil { //nolint:musttag // Preserve stored field names.
			rows.Close()
			return nil, securityStoreError(err)
		}

		if typ == securityEmail || typ == securitySMS {
			factor.Hint = securityMask(factor.Hint)
		}

		if typ != "webauthn" {
			v.Factors = append(
				v.Factors,
				domain.SecurityProofFactor{ID: id, Type: typ, Hint: factor.Hint})
		}
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		return nil, securityStoreError(err)
	}

	var passkey bool
	if err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM iam_webauthn_credentials WHERE project_id=$1 AND environment=$2
AND user_id=$3 AND created_at<=$4)`,
		f.ProjectID,
		f.Environment,
		f.AccountID,
		f.Cutoff).Scan(&passkey); err != nil {
		return nil, securityStoreError(err)
	}

	if passkey {
		v.Factors = append(
			v.Factors,
			domain.SecurityProofFactor{ID: securityPasskey, Type: "webauthn", Hint: "Passkey"})
	}

	return v.Factors, nil
}

func (s *pgSecurity) requestSupport(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	if in.Identifier != "" {
		f.Identifier = strings.TrimSpace(in.Identifier)
	}

	if in.Contact != "" && in.Contact != f.Recipient {
		address, err := mail.ParseAddress(in.Contact)
		if err != nil || address.Address != in.Contact {
			return domain.ErrValidation
		}

		f.Recipient = in.Contact
		f.ContactVerified = false
		f.SupportGrant = false
		f.State.Kind = securityRecoveryTemplate
		f.State.Step = securityVerifyContact
		f.Proved = false

		return s.issueCode(ctx, f)
	}

	if !f.ContactVerified && !f.Proved {
		return domain.ErrStepUpRequired.WithMessage("Verify a contact for support first")
	}

	address, err := mail.ParseAddress(f.Recipient)
	if err != nil || address.Address != f.Recipient {
		return domain.ErrValidation.WithMessage("An available email is required for support")
	}

	return s.openCase(ctx, f, in.Message)
}

func (s *pgSecurity) resendCode(ctx context.Context, f *securityFlow) error {
	scope := securityFlowScope(f)

	if (f.State.Step != securityVerifyIdentity &&
		f.State.Step != securityVerifyContact) ||
		f.State.ResendAt == nil {
		return domain.ErrBadRequest
	}

	if nowIn(ctx).Before(*f.State.ResendAt) {
		return domain.ErrFlowResendTooSoon
	}

	ok, err := s.throttle(
		ctx,
		scope,
		f.Recipient,
		securityResend,
		securityDeliveryBurst,
		time.Hour)
	if err != nil {
		return securityStoreError(err)
	}

	if !ok {
		f.State.ErrorCode = securityRateLimited
	} else if err := s.issueCode(ctx, f); err != nil {
		return securityStoreError(err)
	}

	return nil
}

func (s *pgSecurity) trustReviewedDevice(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	if in.Action != "trust_device" || !f.Proved {
		return domain.ErrBadRequest
	}

	if _, err := s.UpdateDevice(
		ctx,
		securityFlowScope(f),
		f.TargetDeviceID,
		securityTrust,
		"",
		f.CurrentToken); err != nil {
		return securityStoreError(err)
	}

	f.State.Status = securityCompleted
	f.State.Step = securityCompleted
	f.State.Outcome = "device_trusted"

	return nil
}

func (s *pgSecurity) selectReviewSession(ctx context.Context, f *securityFlow, id string) error {
	sessions, err := s.reviewSessions(ctx, f)
	if err != nil {
		return securityStoreError(err)
	}

	for _, session := range sessions {
		if session.ID == id {
			f.TargetSession = session.ID
		}
	}

	if len(sessions) > 0 && f.TargetSession == "" {
		return domain.ErrValidation.WithMessage("Select an affected session or all sessions")
	}

	return nil
}

func (s *pgSecurity) mutateSecurityFlow(
	ctx context.Context,
	scope domain.SecurityScope,
	token string,
	in domain.SecurityFlowInput,
	mode string,
) (*domain.SecurityFlowState, error) {
	flowToken := token

	f, err := s.loadFlow(ctx, scope, flowToken)
	if err != nil {
		return nil, securityStoreError(err)
	}

	requestHash, err := securityRequestHash(in, mode)
	if err != nil {
		return nil, securityStoreError(err)
	}

	if f.CurrentToken != flowToken {
		if f.PreviousTokenHash != accountHashToken(flowToken) ||
			f.PreviousInputHash != requestHash ||
			!nowIn(ctx).Before(f.PreviousUntil) {
			return nil, domain.ErrConflict
		}

		return s.viewFlow(ctx, f, f.CurrentToken)
	}

	if f.State.Status != securityPending {
		return s.viewFlow(ctx, f, flowToken)
	}

	if err := s.validateFlowAuthority(ctx, f); err != nil {
		return nil, securityStoreError(err)
	}

	f.State.ErrorCode = ""
	if mode == securityResend {
		if err := s.resendCode(ctx, f); err != nil {
			return nil, securityStoreError(err)
		}
	} else {
		if in.Version != f.State.Version {
			return nil, domain.ErrConflict.WithMessage("Refresh the flow before submitting this action")
		}

		if err := s.advance(ctx, f, in); err != nil {
			return nil, securityStoreError(err)
		}
	}

	f.State.Version++

	next, err := accountRandomToken(securityTokenBytes)
	if err != nil {
		return nil, securityStoreError(err)
	}

	f.PreviousTokenHash = accountHashToken(flowToken)
	f.PreviousInputHash = requestHash
	f.PreviousUntil = nowIn(ctx).Add(securityRetryGrace)

	flowToken = securityFlowTokenPrefix + next
	if err := s.saveFlow(ctx, f, flowToken); err != nil {
		return nil, securityStoreError(err)
	}

	return s.viewFlow(ctx, f, flowToken)
}

func (s *pgSecurity) startRecoveryContact(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	address, err := mail.ParseAddress(in.Contact)
	if err != nil || address.Address != in.Contact || len(in.Contact) > 254 {
		return domain.ErrValidation.WithMessage("A valid contact email is required")
	}

	if strings.TrimSpace(in.Identifier) == "" {
		return domain.ErrValidation.WithMessage("Account email or phone is required")
	}

	f.Recipient = in.Contact

	f.Identifier = strings.TrimSpace(in.Identifier)
	if err := s.issueCode(ctx, f); err != nil {
		return securityStoreError(err)
	}

	return nil
}

func (s *pgSecurity) priorMFARequired(ctx context.Context, f *securityFlow) (bool, error) {
	var required bool

	err := s.db.TxDB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam_factors
 WHERE project_id=$1 AND environment=$2 AND user_id=$3 AND status='active' AND created_at<=$4)`,
		f.ProjectID, f.Environment, f.AccountID, f.Cutoff).Scan(&required)

	return required, securityStoreError(err)
}
