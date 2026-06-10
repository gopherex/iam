package postgres

// Postgres adapter for the server-side resumable auth flow engine (§9).
//
// Security invariants (§5):
//   - flow_token: ≥256-bit random, `ftk_` prefix; ONLY sha256(token) in DB.
//   - Every read is scoped by project_id + status=pending + expires_at>now.
//   - Expired/foreign/completed → domain.ErrFlowNotFound (no enumeration).
//   - Token rotated on every privilege step (new token + hash written, old token dead).
//   - Passwords never stored in data; only `has_password: true`.
//   - Attempts tracked in active_challenge; wrong code returns error without reset.
//   - resend_at gates re-issue; early resend → domain.ErrFlowResendTooSoon.
//   - Constant-time token comparison via crypto/subtle.
//
// The per-kind `advance` seam: Submit dispatches to advanceSignup (or a stub for
// the others). Each advance function has signature flowAdvanceFn (see below).
// Next-task agents implement their kind and set the corresponding entry in
// flowAdvancers.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// ─── constants ───────────────────────────────────────────────────────────────

const (
	flowTTL           = 30 * time.Minute
	flowTokenPrefix   = "ftk_"
	flowResendCooloff = 60 * time.Second
	flowMaxAttempts   = 5
	flowStatusPending = "pending"
)

// ─── adapter struct ──────────────────────────────────────────────────────────

// pgCoreAuthFlows is the Postgres-backed api.CoreAuthFlows adapter.
type pgCoreAuthFlows struct {
	db       *DB
	emitter  Emitter
	accounts api.CoreAuthAccounts
}

// NewPgCoreAuthFlows builds the flow adapter. accounts must be the same
// pgCoreAuth adapter already registered for the CoreAuth group.
func NewPgCoreAuthFlows(db *DB, emitter Emitter, accounts api.CoreAuthAccounts) *pgCoreAuthFlows {
	return &pgCoreAuthFlows{db: db, emitter: emitter, accounts: accounts}
}

var _ api.CoreAuthFlows = (*pgCoreAuthFlows)(nil)

// ─── token helpers ───────────────────────────────────────────────────────────

// flowMintToken mints a new opaque flow_token (≥256-bit random, `ftk_` prefix).
func flowMintToken() (token, hash string, err error) {
	b := make([]byte, 32) // 256 bits
	if _, err = rand.Read(b); err != nil {
		return
	}
	token = flowTokenPrefix + hex.EncodeToString(b)
	hash = flowHashToken(token)
	return
}

// flowHashToken returns sha256(token) in hex.
func flowHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// flowCompareToken reports whether the supplied token matches the stored hash in
// constant time (§5 rule 1). Kept for callers that want explicit compare.
func flowCompareToken(token, storedHash string) bool {
	h := flowHashToken(token)
	return subtle.ConstantTimeCompare([]byte(h), []byte(storedHash)) == 1
}

// ─── data envelope ──────────────────────────────────────────────────────────

// flowData is the jsonb payload in iam_flows.data.
type flowData struct {
	Contact          domain.FlowContact          `json:"contact"`
	Collected        domain.FlowCollected        `json:"collected"`
	ActiveChallenge  *domain.FlowActiveChallenge `json:"active_challenge,omitempty"`
	ConsentsRequired []domain.FlowConsentRef     `json:"consents_required,omitempty"`
	RegistrationMode string                      `json:"registration_mode,omitempty"`
	PasswordStrategy string                      `json:"password_strategy,omitempty"`
	Error            *domain.FlowError           `json:"error,omitempty"`
}

// flowDataRM serializes a flowData into a json.RawMessage for the setter.
func flowDataRM(d flowData) (json.RawMessage, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// ─── CRUD helpers ────────────────────────────────────────────────────────────

// flowLoad finds a live flow by project+token, enforcing tenant boundary, TTL,
// and status=pending (§5 rule 3). Returns ErrFlowNotFound for any miss.
func (a *pgCoreAuthFlows) flowLoad(ctx context.Context, projectID, token string) (*models.IamFlow, *domain.Flow, error) {
	// Hash the incoming token before the DB call.
	hash := flowHashToken(token)
	rows, err := models.IamFlows.Query(
		sm.Where(models.IamFlows.Columns.TokenHash.EQ(psql.Arg(hash))),
	).All(ctx, a.db.Bobx())
	if err != nil || len(rows) == 0 {
		return nil, nil, domain.ErrFlowNotFound
	}
	row := rows[0]
	// Tenant boundary.
	if row.ProjectID != projectID {
		return nil, nil, domain.ErrFlowNotFound
	}
	// Status must be pending.
	if row.Status != flowStatusPending {
		return nil, nil, domain.ErrFlowNotFound
	}
	// TTL.
	if nowUTC().After(row.ExpiresAt) {
		return nil, nil, domain.ErrFlowExpired
	}
	f, err := flowUnmarshalRow(row)
	if err != nil {
		return nil, nil, err
	}
	return row, f, nil
}

// flowUnmarshalRow converts a model row into a domain.Flow.
func flowUnmarshalRow(row *models.IamFlow) (*domain.Flow, error) {
	var data flowData
	if err := unmarshal(row.Data, &data); err != nil {
		return nil, err
	}
	f := &domain.Flow{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		Kind:             domain.FlowKind(row.Kind),
		Status:           domain.FlowStatus(row.Status),
		Step:             domain.FlowStep(row.Step),
		ExpiresAt:        row.ExpiresAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		Contact:          data.Contact,
		Collected:        data.Collected,
		ActiveChallenge:  data.ActiveChallenge,
		ConsentsRequired: data.ConsentsRequired,
		RegistrationMode: data.RegistrationMode,
		PasswordStrategy: data.PasswordStrategy,
		Error:            data.Error,
	}
	if uid, ok := row.UserID.Get(); ok {
		f.UserID = uid
	}
	return f, nil
}

// flowSave persists updated flow fields using the existing row (no token rotation).
func (a *pgCoreAuthFlows) flowSave(ctx context.Context, row *models.IamFlow, f *domain.Flow) error {
	now := nowUTC()
	rm, err := flowDataRM(flowData{
		Contact:          f.Contact,
		Collected:        f.Collected,
		ActiveChallenge:  f.ActiveChallenge,
		ConsentsRequired: f.ConsentsRequired,
		RegistrationMode: f.RegistrationMode,
		PasswordStrategy: f.PasswordStrategy,
		Error:            f.Error,
	})
	if err != nil {
		return err
	}
	setter := &models.IamFlowSetter{
		Status:    ptr(string(f.Status)),
		Step:      ptr(string(f.Step)),
		UpdatedAt: &now,
		Data:      &rm,
	}
	if f.UserID != "" {
		setter.UserID = ptr(null.From(f.UserID))
	}
	f.UpdatedAt = now
	return row.Update(ctx, a.db.Bobx(), setter)
}

// flowRotate replaces the token_hash (token rotation on privilege step, §5 rule 2).
// Returns the new plain-text token.
func (a *pgCoreAuthFlows) flowRotate(ctx context.Context, row *models.IamFlow, f *domain.Flow) (string, error) {
	newToken, newHash, err := flowMintToken()
	if err != nil {
		return "", fmt.Errorf("flow rotate: mint token: %w", err)
	}
	now := nowUTC()
	rm, err := flowDataRM(flowData{
		Contact:          f.Contact,
		Collected:        f.Collected,
		ActiveChallenge:  f.ActiveChallenge,
		ConsentsRequired: f.ConsentsRequired,
		RegistrationMode: f.RegistrationMode,
		PasswordStrategy: f.PasswordStrategy,
		Error:            f.Error,
	})
	if err != nil {
		return "", err
	}
	setter := &models.IamFlowSetter{
		TokenHash: &newHash,
		Status:    ptr(string(f.Status)),
		Step:      ptr(string(f.Step)),
		UpdatedAt: &now,
		Data:      &rm,
	}
	if f.UserID != "" {
		setter.UserID = ptr(null.From(f.UserID))
	}
	if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
		return "", err
	}
	f.UpdatedAt = now
	return newToken, nil
}

// flowInsert creates a new iam_flows row.
func (a *pgCoreAuthFlows) flowInsert(ctx context.Context, f *domain.Flow, hash string, data flowData) error {
	rm, err := flowDataRM(data)
	if err != nil {
		return err
	}
	setter := &models.IamFlowSetter{
		ID:        &f.ID,
		ProjectID: &f.ProjectID,
		TokenHash: &hash,
		Kind:      ptr(string(f.Kind)),
		Status:    ptr(string(f.Status)),
		Step:      ptr(string(f.Step)),
		ExpiresAt: &f.ExpiresAt,
		CreatedAt: &f.CreatedAt,
		UpdatedAt: &f.UpdatedAt,
		Data:      &rm,
	}
	if f.UserID != "" {
		uid := null.From(f.UserID)
		setter.UserID = &uid
	}
	_, err = models.IamFlows.Insert(setter).One(ctx, a.db.Bobx())
	return err
}

// ─── Create ──────────────────────────────────────────────────────────────────

// flowCreateFn is the per-kind create seam (mirrors flowAdvanceFn for Submit).
// Each kind may process the create-time credentials immediately (e.g. signin
// verifies the password, recovery issues an OTP). Registered in flowCreators;
// per-kind files override their entry from an init().
type flowCreateFn func(ctx context.Context, a *pgCoreAuthFlows, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error)

// flowCreators maps each FlowKind to its create handler. signup is wired here;
// signin/recovery/email_change are registered by their own files (init()), and
// fall back to flowCreateCollect (persist at collect_credentials) until then.
var flowCreators = map[domain.FlowKind]flowCreateFn{
	domain.FlowKindSignup: func(ctx context.Context, a *pgCoreAuthFlows, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
		return a.advanceSignupCreate(ctx, f, cmd)
	},
}

func (a *pgCoreAuthFlows) Create(ctx context.Context, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	now := nowUTC()
	f := &domain.Flow{
		ID:        newUUID(),
		ProjectID: cmd.ProjectID,
		Kind:      cmd.Kind,
		Status:    domain.FlowStatusPending,
		Step:      domain.FlowStepCollectCredentials,
		ExpiresAt: now.Add(flowTTL),
		CreatedAt: now,
		UpdatedAt: now,
		Contact:   domain.FlowContact{Email: cmd.Email},
		Collected: domain.FlowCollected{
			Name:        cmd.Name,
			HasPassword: cmd.Password != "",
		},
	}
	var state *domain.FlowState
	var err error
	if create, ok := flowCreators[cmd.Kind]; ok {
		state, err = create(ctx, a, f, cmd)
	} else {
		state, err = a.flowCreateCollect(ctx, f)
	}
	if err != nil {
		return nil, err
	}
	a.emitFlowContinue(ctx, state, cmd.RedirectTo, cmd.Locale)
	return state, nil
}

// emitFlowContinue fires a best-effort cross-device "continue" deep-link email
// for still-pending email-bearing flows (signup/recovery). The notification
// layer turns flow_token into <app_base_url>/continue?flow=… ; a send/emit
// failure must not fail flow creation, so the error is swallowed.
func (a *pgCoreAuthFlows) emitFlowContinue(ctx context.Context, state *domain.FlowState, redirectTo, locale string) {
	f := state.Flow
	if f.Status != domain.FlowStatusPending || f.Contact.Email == "" {
		return
	}
	if f.Kind != domain.FlowKindSignup && f.Kind != domain.FlowKindRecovery {
		return
	}
	payload := map[string]any{
		"flow_token": state.FlowToken,
		"kind":       string(f.Kind),
		"to":         f.Contact.Email,
		"contact":    f.Contact.Email,
	}
	// Per-flow base override; the notification layer validates its origin against
	// the project before honouring it, falling back to app_base_url.
	if redirectTo != "" {
		payload["redirect_to"] = redirectTo
	}
	// Requested language; the notification layer falls back to account/project
	// default when empty.
	if locale != "" {
		payload["locale"] = locale
	}
	_ = a.emitter.Emit(ctx, domain.Event{
		Type:        "auth.flow.continue",
		ProjectID:   f.ProjectID,
		AggregateID: f.ID,
		Payload:     payload,
	})
}

// flowCreateCollect persists a fresh flow at collect_credentials with a new token.
// Default create behavior for kinds that have not registered a create handler.
func (a *pgCoreAuthFlows) flowCreateCollect(ctx context.Context, f *domain.Flow) (*domain.FlowState, error) {
	token, hash, err := flowMintToken()
	if err != nil {
		return nil, fmt.Errorf("flow create: %w", err)
	}
	if err := a.flowInsert(ctx, f, hash, flowData{
		Contact:   f.Contact,
		Collected: f.Collected,
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// ─── Get ─────────────────────────────────────────────────────────────────────

func (a *pgCoreAuthFlows) Get(ctx context.Context, cmd domain.FlowGetCmd) (*domain.FlowState, error) {
	_, f, err := a.flowLoad(ctx, cmd.ProjectID, cmd.FlowToken)
	if err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
}

// ─── Submit ──────────────────────────────────────────────────────────────────

// flowAdvanceFn is the per-kind advance seam. Next-task agents implement one
// per kind and register it in the flowAdvancers map.
//
// Signature:
//
//		func(ctx context.Context, a *pgCoreAuthFlows, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error)
//
//	  - ctx: request context
//	  - a: the adapter (access to db, emitter, accounts)
//	  - row: raw model row for row.Update calls
//	  - f: loaded domain.Flow
//	  - cmd: the submit command (Action + Payload)
//
// Must return the committed FlowState with the (possibly new) token in FlowToken.
// Return a domain.Error to surface a client-visible error. The token in cmd is
// still valid if the step did not require rotation.
type flowAdvanceFn func(ctx context.Context, a *pgCoreAuthFlows, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error)

// flowAdvancers maps each FlowKind to its advance function. signin/recovery/
// email_change stubs return ErrNotImplemented; next tasks replace them.
var flowAdvancers = map[domain.FlowKind]flowAdvanceFn{
	domain.FlowKindSignup:      advanceSignup,
	domain.FlowKindSignin:      advanceNotImplemented,
	domain.FlowKindRecovery:    advanceNotImplemented,
	domain.FlowKindEmailChange: advanceNotImplemented,
}

func advanceNotImplemented(_ context.Context, _ *pgCoreAuthFlows, _ *models.IamFlow, _ *domain.Flow, _ domain.FlowSubmitCmd) (*domain.FlowState, error) {
	return nil, domain.ErrNotImplemented
}

func (a *pgCoreAuthFlows) Submit(ctx context.Context, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	row, f, err := a.flowLoad(ctx, cmd.ProjectID, cmd.FlowToken)
	if err != nil {
		return nil, err
	}
	advance, ok := flowAdvancers[f.Kind]
	if !ok {
		return nil, domain.ErrNotImplemented
	}
	return advance(ctx, a, row, f, cmd)
}

// ─── Resend ──────────────────────────────────────────────────────────────────

func (a *pgCoreAuthFlows) Resend(ctx context.Context, cmd domain.FlowResendCmd) (*domain.FlowState, error) {
	row, f, err := a.flowLoad(ctx, cmd.ProjectID, cmd.FlowToken)
	if err != nil {
		return nil, err
	}
	ac := f.ActiveChallenge
	if ac == nil {
		return nil, domain.ErrBadRequest.WithMessage("no active challenge to resend")
	}
	// Rate-limit: check resend_at (§5 rule 7).
	if nowUTC().Before(ac.ResendAt) {
		return nil, domain.ErrFlowResendTooSoon.WithDetails(map[string]any{
			"resend_at": ac.ResendAt.Unix(),
		})
	}
	// Re-issue email verification challenge.
	ch, err := a.accounts.StartEmailVerification(ctx, domain.CoreAuthVerifyStartCmd{
		ProjectID: f.ProjectID,
		AccountID: f.UserID,
		Contact:   f.Contact.Email,
	})
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	f.ActiveChallenge = &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      "email",
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
	}
	f.Error = nil
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		return a.flowSave(ctx, row, f)
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
}

// ─── Abandon ─────────────────────────────────────────────────────────────────

func (a *pgCoreAuthFlows) Abandon(ctx context.Context, cmd domain.FlowAbandonCmd) error {
	row, f, err := a.flowLoad(ctx, cmd.ProjectID, cmd.FlowToken)
	if err != nil {
		// Already gone — idempotent.
		return nil
	}
	f.Status = domain.FlowStatusAborted
	return a.db.withTx(ctx, func(ctx context.Context) error {
		return a.flowSave(ctx, row, f)
	})
}

// ─── Signup advance ──────────────────────────────────────────────────────────

// advanceSignupCreate is called during Create for signup. It registers the user,
// issues an email challenge, and persists the flow at step=verify_email.
// Returns a FlowState with the initial token (new token from flowMintToken).
// flowAuthConfig reads the project's signup policy (registration mode + password
// strategy) from the auth config doc. Empty strings mean "unset" → open/default.
// Reads the live environment (runtime is single-env until A2 wiring).
func (a *pgCoreAuthFlows) flowAuthConfig(ctx context.Context, projectID string) (mode, passwordStrategy string) {
	row, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(coreAuthDefaultEnv))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("auth"))),
	).One(ctx, a.db.Bobx())
	if err != nil || len(row.Data) == 0 {
		return "", ""
	}
	var doc struct {
		Registration struct {
			Mode             string `json:"mode"`
			PasswordStrategy string `json:"password_strategy"`
		} `json:"registration"`
	}
	if unmarshal(row.Data, &doc) != nil {
		return "", ""
	}
	return doc.Registration.Mode, doc.Registration.PasswordStrategy
}

// flowPersistAtStep mints a token and persists a fresh flow halted at a terminal
// or waiting step (blocked / request_access), with no challenge issued.
func (a *pgCoreAuthFlows) flowPersistAtStep(ctx context.Context, f *domain.Flow, step domain.FlowStep, ferr *domain.FlowError) (*domain.FlowState, error) {
	f.Step = step
	f.Error = ferr
	token, hash, err := flowMintToken()
	if err != nil {
		return nil, fmt.Errorf("flow persist at %s: mint token: %w", step, err)
	}
	if err := a.flowInsert(ctx, f, hash, flowData{
		Contact:          f.Contact,
		Collected:        f.Collected,
		RegistrationMode: f.RegistrationMode,
		PasswordStrategy: f.PasswordStrategy,
		Error:            ferr,
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

func (a *pgCoreAuthFlows) advanceSignupCreate(ctx context.Context, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	// 0. Enforce the project's registration policy (read from auth config).
	mode, pwStrategy := a.flowAuthConfig(ctx, f.ProjectID)
	f.RegistrationMode = mode
	f.PasswordStrategy = pwStrategy
	switch mode {
	case "closed":
		return a.flowPersistAtStep(ctx, f, domain.FlowStepBlocked,
			&domain.FlowError{Code: "registration_closed", Message: "Registration is closed."})
	case "invite_only":
		// Invite redemption lands in Phase 1b; until then, no public signup.
		return a.flowPersistAtStep(ctx, f, domain.FlowStepBlocked,
			&domain.FlowError{Code: "invite_required", Message: "An invitation is required to sign up."})
	case "request_access":
		return a.flowPersistAtStep(ctx, f, domain.FlowStepRequestAccess, nil)
	}

	// 1. Register the user. With the after_verify password strategy the account is
	// created WITHOUT a password (set later at the set_password step).
	password := cmd.Password
	if pwStrategy == "after_verify" {
		password = ""
	}
	acct, _, err := a.accounts.Register(ctx, domain.RegisterCmd{
		ProjectID: f.ProjectID,
		Email:     cmd.Email,
		Password:  password,
		Name:      cmd.Name,
		Locale:    cmd.Locale,
	})
	if err != nil {
		return nil, err
	}
	f.UserID = acct.ID
	f.Step = domain.FlowStepVerifyEmail

	// 2. Issue email verification challenge (auto-issued per §7).
	ch, err := a.accounts.StartEmailVerification(ctx, domain.CoreAuthVerifyStartCmd{
		ProjectID: f.ProjectID,
		AccountID: acct.ID,
		Contact:   acct.PrimaryEmail,
		Locale:    cmd.Locale,
	})
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	f.ActiveChallenge = &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      "email",
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
	}

	// 3. Mint token and persist the flow row.
	token, hash, err := flowMintToken()
	if err != nil {
		return nil, fmt.Errorf("flow signup create: mint token: %w", err)
	}
	if err := a.flowInsert(ctx, f, hash, flowData{
		Contact:          f.Contact,
		Collected:        f.Collected,
		ActiveChallenge:  f.ActiveChallenge,
		RegistrationMode: f.RegistrationMode,
		PasswordStrategy: f.PasswordStrategy,
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// advanceSignup handles Submit for signup flows.
func advanceSignup(ctx context.Context, a *pgCoreAuthFlows, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	switch f.Step {
	case domain.FlowStepVerifyEmail:
		if cmd.Action != "verify_email" {
			return nil, domain.ErrBadRequest.WithMessage("expected action verify_email")
		}
		return a.signupVerifyEmail(ctx, row, f, cmd)
	case domain.FlowStepSetPassword:
		if cmd.Action != "set_password" {
			return nil, domain.ErrBadRequest.WithMessage("expected action set_password")
		}
		return a.signupSetPassword(ctx, row, f, cmd)
	default:
		return nil, domain.ErrBadRequest.WithMessage(fmt.Sprintf("unexpected step %q for signup", f.Step))
	}
}

// signupVerifyEmail verifies the email OTP code. On success it rotates the token,
// completes the flow, and returns a session (§5 rules 2, 8).
func (a *pgCoreAuthFlows) signupVerifyEmail(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	ac := f.ActiveChallenge
	if ac == nil {
		return nil, domain.ErrBadRequest.WithMessage("no active email challenge")
	}
	code := cmd.Payload["code"]
	if code == "" {
		return nil, domain.ErrBadRequest.WithMessage("code is required")
	}
	if ac.AttemptsLeft <= 0 {
		return nil, domain.ErrChallengeInvalid.WithMessage("challenge exhausted; please resend")
	}

	// VerifyEmail consumes the challenge and marks the account email_verified.
	acct, sess, err := a.accounts.VerifyEmail(ctx, domain.CoreAuthVerifyConsumeCmd{
		ProjectID:   f.ProjectID,
		AccountID:   f.UserID,
		ChallengeID: ac.ChallengeID,
		Code:        code,
	})
	if err != nil {
		// Wrong code: decrement attempts, embed error in flow, stay pending (§5 rule 6).
		ac.AttemptsLeft--
		f.Error = &domain.FlowError{Code: "invalid_code", Message: "The verification code is incorrect."}
		// Best-effort save; ignore error to return the flow state.
		_ = a.db.withTx(ctx, func(ctx context.Context) error {
			return a.flowSave(ctx, row, f)
		})
		return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
	}

	// Email verified.
	f.ActiveChallenge = nil
	f.Error = nil
	f.UserID = acct.ID

	// after_verify strategy: the account has no password yet — advance to the
	// set_password step instead of completing. The session minted by VerifyEmail
	// is not surfaced; set_password mints the real one.
	if f.PasswordStrategy == "after_verify" {
		f.Step = domain.FlowStepSetPassword
		if err := a.db.withTx(ctx, func(ctx context.Context) error {
			return a.flowSave(ctx, row, f)
		}); err != nil {
			return nil, err
		}
		return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
	}

	// Success: complete the flow, rotate the token (§5 rule 2).
	f.Status = domain.FlowStatusCompleted
	f.Step = domain.FlowStepCompleted

	newToken, err := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		return a.flowRotate(ctx, row, f)
	})
	if err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: newToken, Flow: f, Session: sess}, nil
}

// signupSetPassword handles the set_password step for the after_verify strategy:
// it writes the account's first password credential, completes the flow, rotates
// the token (privilege step §5 rule 2), and returns a fresh session.
func (a *pgCoreAuthFlows) signupSetPassword(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	if f.UserID == "" {
		return nil, domain.ErrBadRequest.WithMessage("no verified user for signup")
	}
	password := cmd.Payload["password"]
	if password == "" {
		return nil, domain.ErrBadRequest.WithMessage("password is required")
	}
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("signup set_password: accounts is not *pgCoreAuth")
	}

	sess, err := withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Session, error) {
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), f.UserID)
		if err != nil {
			return nil, fmt.Errorf("signup set_password: load user: %w", err)
		}
		acc, err := coreAuthLoadAccount(userRow, f.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("signup set_password: parse account: %w", err)
		}
		hash, err := coreAuthHashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("signup set_password: hash password: %w", err)
		}
		if err := pgCA.coreAuthUpsertPasswordCredential(ctx, acc.ProjectID, acc.ID, hash); err != nil {
			return nil, fmt.Errorf("signup set_password: upsert credential: %w", err)
		}
		s, err := pgCA.coreAuthMintSession(ctx, acc, "", []string{"pwd"}, 1)
		if err != nil {
			return nil, fmt.Errorf("signup set_password: mint session: %w", err)
		}
		return s, nil
	})
	if err != nil {
		return nil, err
	}

	f.Status = domain.FlowStatusCompleted
	f.Step = domain.FlowStepCompleted
	f.Error = nil
	newToken, err := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		return a.flowRotate(ctx, row, f)
	})
	if err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: newToken, Flow: f, Session: sess}, nil
}
