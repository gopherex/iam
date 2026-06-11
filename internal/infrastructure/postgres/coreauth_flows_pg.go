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
	"errors"
	"fmt"
	"strings"
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
	cfg      *configReader
}

// NewPgCoreAuthFlows builds the flow adapter. accounts must be the same
// pgCoreAuth adapter already registered for the CoreAuth group. cfg is the
// shared runtime config reader (registration policy lives in the auth doc); a
// nil cfg falls back to a fresh default-TTL reader.
func NewPgCoreAuthFlows(db *DB, emitter Emitter, accounts api.CoreAuthAccounts, cfg *configReader) *pgCoreAuthFlows {
	if cfg == nil {
		cfg = NewConfigReader(db, 0)
	}
	return &pgCoreAuthFlows{db: db, emitter: emitter, accounts: accounts, cfg: cfg}
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
	Locale           string                            `json:"locale,omitempty"`
	RedirectTo       string                            `json:"redirect_to,omitempty"`
	Contact          domain.FlowContact                `json:"contact"`
	Collected        domain.FlowCollected              `json:"collected"`
	ActiveChallenge  *domain.FlowActiveChallenge       `json:"active_challenge,omitempty"`
	ConsentsRequired []domain.FlowConsentRef           `json:"consents_required,omitempty"`
	ConsentsAccepted []domain.AccountConsentAcceptance `json:"consents_accepted,omitempty"`
	RegistrationMode string                            `json:"registration_mode,omitempty"`
	PasswordStrategy string                            `json:"password_strategy,omitempty"`
	Error            *domain.FlowError                 `json:"error,omitempty"`
	Method           string                            `json:"method,omitempty"`
	AvailableMethods []string                          `json:"available_methods,omitempty"`
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
	// Environment boundary: a flow created in one environment is invisible from
	// another (test/live isolation). The request env is resolved from ctx.
	env, err := effectiveEnv(ctx, a.db, projectID, coreAuthDefaultEnv)
	if err != nil {
		return nil, nil, err
	}
	if flowRowEnv(row) != env {
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
		Environment:      flowRowEnv(row),
		Kind:             domain.FlowKind(row.Kind),
		Status:           domain.FlowStatus(row.Status),
		Step:             domain.FlowStep(row.Step),
		ExpiresAt:        row.ExpiresAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		Locale:           data.Locale,
		RedirectTo:       data.RedirectTo,
		Contact:          data.Contact,
		Collected:        data.Collected,
		ActiveChallenge:  data.ActiveChallenge,
		ConsentsRequired: data.ConsentsRequired,
		ConsentsAccepted: data.ConsentsAccepted,
		RegistrationMode: data.RegistrationMode,
		PasswordStrategy: data.PasswordStrategy,
		Error:            data.Error,
		Method:           data.Method,
		AvailableMethods: data.AvailableMethods,
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
		Locale:           f.Locale,
		RedirectTo:       f.RedirectTo,
		Contact:          f.Contact,
		Collected:        f.Collected,
		ActiveChallenge:  f.ActiveChallenge,
		ConsentsRequired: f.ConsentsRequired,
		ConsentsAccepted: f.ConsentsAccepted,
		RegistrationMode: f.RegistrationMode,
		PasswordStrategy: f.PasswordStrategy,
		Error:            f.Error,
		Method:           f.Method,
		AvailableMethods: f.AvailableMethods,
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
		Locale:           f.Locale,
		RedirectTo:       f.RedirectTo,
		Contact:          f.Contact,
		Collected:        f.Collected,
		ActiveChallenge:  f.ActiveChallenge,
		ConsentsRequired: f.ConsentsRequired,
		ConsentsAccepted: f.ConsentsAccepted,
		RegistrationMode: f.RegistrationMode,
		PasswordStrategy: f.PasswordStrategy,
		Error:            f.Error,
		Method:           f.Method,
		AvailableMethods: f.AvailableMethods,
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

// flowRowEnv returns the environment a flow row belongs to, defaulting to the
// runtime default for rows written before env tagging.
func flowRowEnv(row *models.IamFlow) string {
	if row.Environment == "" {
		return coreAuthDefaultEnv
	}
	return row.Environment
}

// flowInsert creates a new iam_flows row.
func (a *pgCoreAuthFlows) flowInsert(ctx context.Context, f *domain.Flow, hash string, data flowData) error {
	if data.Locale == "" {
		data.Locale = f.Locale
	}
	if data.RedirectTo == "" {
		data.RedirectTo = f.RedirectTo
	}
	rm, err := flowDataRM(data)
	if err != nil {
		return err
	}
	flowEnv := f.Environment
	if flowEnv == "" {
		flowEnv = coreAuthDefaultEnv
	}
	setter := &models.IamFlowSetter{
		ID:          &f.ID,
		ProjectID:   &f.ProjectID,
		Environment: &flowEnv,
		TokenHash:   &hash,
		Kind:        ptr(string(f.Kind)),
		Status:      ptr(string(f.Status)),
		Step:        ptr(string(f.Step)),
		ExpiresAt:   &f.ExpiresAt,
		CreatedAt:   &f.CreatedAt,
		UpdatedAt:   &f.UpdatedAt,
		Data:        &rm,
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
	env, err := effectiveEnv(ctx, a.db, cmd.ProjectID, coreAuthDefaultEnv)
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	f := &domain.Flow{
		ID:          newUUID(),
		ProjectID:   cmd.ProjectID,
		Environment: env,
		Kind:        cmd.Kind,
		Status:      domain.FlowStatusPending,
		Step:        domain.FlowStepCollectCredentials,
		ExpiresAt:   now.Add(flowTTL),
		CreatedAt:   now,
		UpdatedAt:   now,
		Locale:      cmd.Locale,
		RedirectTo:  cmd.RedirectTo,
		Contact:     domain.FlowContact{Email: cmd.Email, Phone: cmd.Phone},
		Collected: domain.FlowCollected{
			Name:        cmd.Name,
			HasPassword: cmd.Password != "",
		},
		ConsentsAccepted: cmd.Consents,
		Method:           cmd.Method,
	}
	var state *domain.FlowState
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

// emitFlowContinue fires a best-effort flow proof email for still-pending
// email-bearing flows (signup/recovery). The notification layer turns
// flow_token+token into <app_base_url>/continue?flow=…&token=… so clients can
// resume the flow and auto-submit the one-time proof. A send/emit failure must
// not fail flow creation, so the error is swallowed.
func (a *pgCoreAuthFlows) emitFlowContinue(ctx context.Context, state *domain.FlowState, redirectTo, locale string) {
	f := state.Flow
	if f.Status != domain.FlowStatusPending || f.Contact.Email == "" {
		return
	}
	if f.Kind != domain.FlowKindSignup && f.Kind != domain.FlowKindRecovery {
		return
	}
	ac := f.ActiveChallenge
	if ac == nil || ac.Channel != "email" || ac.Code == "" || ac.Token == "" {
		return
	}
	if redirectTo == "" {
		redirectTo = f.RedirectTo
	}
	payload := map[string]any{
		"flow_token":   state.FlowToken,
		"token":        ac.Token,
		"code":         ac.Code,
		"challenge_id": ac.ChallengeID,
		"kind":         string(f.Kind),
		"purpose":      string(f.Kind),
		"to":           f.Contact.Email,
		"contact":      f.Contact.Email,
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
		Environment: f.Environment,
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

func flowVerificationSecret(payload map[string]string) (code string, token string) {
	code = strings.TrimSpace(payload["code"])
	token = strings.TrimSpace(payload["token"])
	return code, token
}

func flowVerifyConsumeCmd(projectID, accountID, challengeID, code, token string) domain.CoreAuthVerifyConsumeCmd {
	cmd := domain.CoreAuthVerifyConsumeCmd{
		ProjectID:   projectID,
		AccountID:   accountID,
		ChallengeID: challengeID,
	}
	if code != "" {
		cmd.Code = code
	} else {
		// In flow mode the link token must match the active challenge, not just
		// any unconsumed token in the project.
		cmd.Token = token
	}
	return cmd
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
	// Re-issue the active challenge by its channel / the flow's method. Email is
	// the default (signup verify, recovery email, signup); sms re-issues an OTP;
	// magic_link re-issues a fresh link.
	ch, channel, err := a.flowReissueChallenge(ctx, f, ac)
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	f.ActiveChallenge = &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      channel,
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
		Code:         ch.Code,
		Token:        ch.Token,
	}
	f.Error = nil
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		return a.flowSave(ctx, row, f)
	}); err != nil {
		return nil, err
	}
	state := &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}
	a.emitFlowContinue(ctx, state, f.RedirectTo, f.Locale)
	return state, nil
}

// flowReissueChallenge re-issues the flow's active challenge using the channel
// recorded on the challenge (falling back to the flow method). It returns the
// new challenge id, channel and expiry. Email re-issues a verification (or
// password_reset for recovery) code; sms re-issues an OTP via the passwordless
// adapter; magic_link re-issues a fresh link.
func (a *pgCoreAuthFlows) flowReissueChallenge(ctx context.Context, f *domain.Flow, ac *domain.FlowActiveChallenge) (*domain.Challenge, string, error) {
	switch {
	case ac.Channel == "sms" || f.Method == domain.FlowMethodPhoneOTP:
		core, ok := a.accounts.(*pgCoreAuth)
		if !ok {
			return nil, "", fmt.Errorf("flow resend: accounts is not *pgCoreAuth")
		}
		pl := NewPgPasswordlessAccounts(a.db, a.emitter, a.cfg, core)
		purpose := "signin"
		if f.Kind == domain.FlowKindRecovery {
			purpose = "recovery"
		}
		ch, serr := pl.StartOTP(ctx, f.ProjectID, f.Contact.Phone, "sms", purpose, f.Locale)
		if serr != nil {
			return nil, "", serr
		}
		return ch, "sms", nil
	case f.Method == domain.FlowMethodMagicLink:
		core, ok := a.accounts.(*pgCoreAuth)
		if !ok {
			return nil, "", fmt.Errorf("flow resend: accounts is not *pgCoreAuth")
		}
		pl := NewPgPasswordlessAccounts(a.db, a.emitter, a.cfg, core)
		ch, serr := pl.StartMagicLink(ctx, f.ProjectID, f.Contact.Email, "", f.Locale)
		if serr != nil {
			return nil, "", serr
		}
		return ch, "email", nil
	default:
		if f.Kind == domain.FlowKindRecovery {
			ch, serr := a.flowIssueRecoveryEmailChallenge(ctx, f)
			if serr != nil {
				return nil, "", serr
			}
			return ch, "email", nil
		}
		// Email verification (signup email path).
		ch, serr := a.accounts.StartEmailVerification(ctx, domain.CoreAuthVerifyStartCmd{
			ProjectID:     f.ProjectID,
			AccountID:     f.UserID,
			Contact:       f.Contact.Email,
			Locale:        f.Locale,
			SuppressEmail: true,
		})
		if serr != nil {
			return nil, "", serr
		}
		return ch, "email", nil
	}
}

func (a *pgCoreAuthFlows) flowIssueRecoveryEmailChallenge(ctx context.Context, f *domain.Flow) (*domain.Challenge, error) {
	if f.UserID == "" {
		return &domain.Challenge{ID: newUUID(), Type: "password_reset", ExpiresAt: nowUTC().Add(coreAuthChallengeTTL)}, nil
	}
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("flow recovery challenge: accounts is not *pgCoreAuth")
	}
	code, err := coreAuthRandomCode()
	if err != nil {
		return nil, fmt.Errorf("flow recovery challenge: random code: %w", err)
	}
	token, err := coreAuthRandomToken()
	if err != nil {
		return nil, fmt.Errorf("flow recovery challenge: random token: %w", err)
	}
	now := nowUTC()
	data := coreAuthChallengeData{
		ID:          newUUID(),
		ProjectID:   f.ProjectID,
		Environment: f.Environment,
		Type:        "password_reset",
		Purpose:     "reset",
		AccountID:   f.UserID,
		Subject:     f.Contact.Email,
		CodeHash:    coreAuthSHA256(code),
		TokenHash:   coreAuthSHA256(token),
		RedirectTo:  f.RedirectTo,
		Locale:      f.Locale,
		ExpiresAt:   now.Add(coreAuthChallengeTTL),
		CreatedAt:   now,
	}
	var ch *domain.Challenge
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		inserted, insErr := pgCA.coreAuthInsertChallenge(ctx, data)
		if insErr != nil {
			return insErr
		}
		ch = inserted
		return nil
	}); err != nil {
		return nil, err
	}
	ch.Code = code
	ch.Token = token
	return ch, nil
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
// Config is resolved under the request's effective environment so test/live can
// carry different registration policies.
func (a *pgCoreAuthFlows) flowAuthConfig(ctx context.Context, projectID string) (mode, passwordStrategy string) {
	cfg, err := a.cfg.AuthConfig(ctx, projectID)
	if err != nil {
		return "", ""
	}
	return cfg.RegistrationMode, cfg.PasswordStrategy
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
		ConsentsAccepted: f.ConsentsAccepted,
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
		// Redeem the supplied invite. A valid+pending+unexpired token (matching the
		// bound email for email-bound invites) lets the signup proceed and is marked
		// accepted on the success path below. Absent token → invite_required; a bad
		// token → invite_invalid. Both keep the flow blocked.
		if cmd.InviteToken == "" {
			return a.flowPersistAtStep(ctx, f, domain.FlowStepBlocked,
				&domain.FlowError{Code: "invite_required", Message: "An invitation is required to sign up."})
		}
		inviteRow, ok, err := a.flowFindRedeemableInvite(ctx, f.ProjectID, cmd.InviteToken, cmd.Email)
		if err != nil {
			return nil, err
		}
		if !ok {
			return a.flowPersistAtStep(ctx, f, domain.FlowStepBlocked,
				&domain.FlowError{Code: "invite_invalid", Message: "The invitation is invalid or expired."})
		}
		// Valid invite: proceed with the normal signup path, marking the invite
		// accepted in the same (ambient) transaction once the user is created.
		return a.advanceSignupCreateAccepted(ctx, f, cmd, inviteRow)
	case "request_access":
		return a.flowPersistAtStep(ctx, f, domain.FlowStepRequestAccess, nil)
	}

	return a.flowSignupRegisterAndPersist(ctx, f, cmd, pwStrategy)
}

// advanceSignupCreateAccepted runs the invite_only success path: it marks the
// redeemed invite accepted (same ambient transaction; a later register error
// rolls it back) and then proceeds with the normal signup.
func (a *pgCoreAuthFlows) advanceSignupCreateAccepted(ctx context.Context, f *domain.Flow, cmd domain.FlowCreateCmd, inviteRow *models.IamInvite) (*domain.FlowState, error) {
	if err := a.flowMarkInviteAccepted(ctx, inviteRow); err != nil {
		return nil, err
	}
	return a.flowSignupRegisterAndPersist(ctx, f, cmd, f.PasswordStrategy)
}

// flowSignupRegisterAndPersist registers the user, issues the email challenge,
// and persists the flow at verify_email. Shared by the open and invite_only
// (accepted) paths.
func (a *pgCoreAuthFlows) flowSignupRegisterAndPersist(ctx context.Context, f *domain.Flow, cmd domain.FlowCreateCmd, pwStrategy string) (*domain.FlowState, error) {
	// 1. Register the user. With the after_verify password strategy the account is
	// created WITHOUT a password (set later at the set_password step).
	password := cmd.Password
	if pwStrategy == "after_verify" {
		password = ""
	}
	acct, _, err := a.accounts.Register(ctx, domain.RegisterCmd{
		ProjectID:       f.ProjectID,
		Email:           cmd.Email,
		Password:        password,
		Name:            cmd.Name,
		Locale:          cmd.Locale,
		SkipConsentGate: true, // flow enforces consent at the accept_consents step
	})
	if err != nil {
		return nil, err
	}
	f.UserID = acct.ID
	f.Step = domain.FlowStepVerifyEmail

	// Snapshot the project's required consents onto the flow so the post-identity
	// accept_consents gate (flowCompleteOrGateConsents) knows what to enforce.
	f.ConsentsRequired = a.flowRequiredConsents(ctx, f.ProjectID, cmd.Locale)

	// 2. Issue email verification challenge (auto-issued per §7).
	ch, err := a.accounts.StartEmailVerification(ctx, domain.CoreAuthVerifyStartCmd{
		ProjectID:     f.ProjectID,
		AccountID:     acct.ID,
		Contact:       acct.PrimaryEmail,
		Locale:        cmd.Locale,
		SuppressEmail: true,
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
		Code:         ch.Code,
		Token:        ch.Token,
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
		ConsentsRequired: f.ConsentsRequired,
		ConsentsAccepted: f.ConsentsAccepted,
		RegistrationMode: f.RegistrationMode,
		PasswordStrategy: f.PasswordStrategy,
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// flowRequiredConsents resolves the project's required consent documents to one
// (key, version) ref per key under the request environment, picking the document
// that best matches the requested locale. An absent/empty config yields nil —
// the flow then completes without an accept_consents step (pre-gate behaviour).
// A config read error is swallowed to nil here, consistent with flowAuthConfig.
func (a *pgCoreAuthFlows) flowRequiredConsents(ctx context.Context, projectID, locale string) []domain.FlowConsentRef {
	docs, err := a.cfg.ConsentConfig(ctx, projectID)
	if err != nil || len(docs) == 0 {
		return nil
	}
	defLocale, _ := a.cfg.AuthConfig(ctx, projectID)
	required := resolveRequiredConsents(docs, locale, defLocale.DefaultLocale)
	if len(required) == 0 {
		return nil
	}
	out := make([]domain.FlowConsentRef, 0, len(required))
	for _, r := range required {
		out = append(out, domain.FlowConsentRef{Key: r.Key, Version: r.Version})
	}
	return out
}

// flowFindRedeemableInvite looks up an invite by its raw token and validates it
// for (project, request env, status=pending, unexpired, email match for
// email-bound invites). Returns (row, true, nil) when redeemable; (nil, false,
// nil) for any validation miss; (nil, false, err) only on an unexpected DB error.
func (a *pgCoreAuthFlows) flowFindRedeemableInvite(ctx context.Context, projectID, rawToken, email string) (*models.IamInvite, bool, error) {
	env, err := effectiveEnv(ctx, a.db, projectID, coreAuthDefaultEnv)
	if err != nil {
		return nil, false, err
	}
	hash := inviteHashToken(rawToken)
	rows, err := models.IamInvites.Query(
		sm.Where(models.IamInvites.Columns.TokenHash.EQ(psql.Arg(hash))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	row := rows[0]
	if row.ProjectID != projectID {
		return nil, false, nil
	}
	if row.Environment != env {
		return nil, false, nil
	}
	if row.Status != inviteStatusPend {
		return nil, false, nil
	}
	if exp, ok := row.ExpiresAt.Get(); ok && nowUTC().After(exp) {
		return nil, false, nil
	}
	// Email-bound invites must match the signup email.
	if bound, ok := row.Email.Get(); ok && bound != "" && bound != email {
		return nil, false, nil
	}
	return row, true, nil
}

// flowMarkInviteAccepted sets status=accepted + accepted_at=now on a redeemed
// invite, joining the caller's (ambient) transaction.
func (a *pgCoreAuthFlows) flowMarkInviteAccepted(ctx context.Context, row *models.IamInvite) error {
	now := nowUTC()
	return row.Update(ctx, a.db.Bobx(), &models.IamInviteSetter{
		Status:     ptr(inviteStatusAccept),
		AcceptedAt: ptr(null.From(now)),
		UpdatedAt:  &now,
	})
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
	case domain.FlowStepAcceptConsents:
		if cmd.Action != "accept_consents" {
			return nil, domain.ErrBadRequest.WithMessage("expected action accept_consents")
		}
		return a.signupAcceptConsents(ctx, row, f, cmd)
	default:
		return nil, domain.ErrBadRequest.WithMessage(fmt.Sprintf("unexpected step %q for signup", f.Step))
	}
}

// signupVerifyEmail verifies the email OTP code or link token. On success it
// rotates the token, completes the flow, and returns a session (§5 rules 2, 8).
func (a *pgCoreAuthFlows) signupVerifyEmail(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	ac := f.ActiveChallenge
	if ac == nil {
		return nil, domain.ErrBadRequest.WithMessage("no active email challenge")
	}
	code, token := flowVerificationSecret(cmd.Payload)
	if code == "" && token == "" {
		return nil, domain.ErrBadRequest.WithMessage("code or token is required")
	}
	if ac.AttemptsLeft <= 0 {
		return nil, domain.ErrChallengeInvalid.WithMessage("challenge exhausted; please resend")
	}

	// VerifyEmail consumes the challenge and marks the account email_verified.
	acct, sess, err := a.accounts.VerifyEmail(ctx, flowVerifyConsumeCmd(f.ProjectID, f.UserID, ac.ChallengeID, code, token))
	if err != nil {
		// Wrong code/token: decrement attempts, embed error in flow, stay pending (§5 rule 6).
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

	// Identity proven: gate on required consents, otherwise complete + rotate.
	return a.flowCompleteOrGateConsents(ctx, row, f, cmd, sess)
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
		if err := pgCA.coreAuthEnforcePasswordPolicy(ctx, acc.ProjectID, password); err != nil {
			return nil, err
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

	// Password set: gate on required consents, otherwise complete + rotate.
	return a.flowCompleteOrGateConsents(ctx, row, f, cmd, sess)
}

func flowRequiredConsentDocs(f *domain.Flow) []consentRequiredDoc {
	required := make([]consentRequiredDoc, 0, len(f.ConsentsRequired))
	for _, r := range f.ConsentsRequired {
		required = append(required, consentRequiredDoc{Key: r.Key, Version: r.Version})
	}
	return required
}

func mergeConsentAcceptances(lists ...[]domain.AccountConsentAcceptance) []domain.AccountConsentAcceptance {
	var out []domain.AccountConsentAcceptance
	seen := make(map[domain.FlowConsentRef]struct{})
	for _, list := range lists {
		for _, acc := range list {
			ref := domain.FlowConsentRef{Key: acc.Key, Version: acc.Version}
			if _, ok := seen[ref]; ok {
				continue
			}
			seen[ref] = struct{}{}
			out = append(out, acc)
		}
	}
	return out
}

func (a *pgCoreAuthFlows) signupRevokeProvisionalSession(ctx context.Context, f *domain.Flow, sess *domain.Session) error {
	if sess == nil {
		return nil
	}
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil
	}
	if err := pgCA.coreAuthRevokeSession(ctx, f.ProjectID, sess.ID); err != nil && !errors.Is(err, domain.ErrSessionNotFound) {
		return err
	}
	return nil
}

func (a *pgCoreAuthFlows) signupRecordAcceptedConsents(ctx context.Context, f *domain.Flow, accepted []domain.AccountConsentAcceptance) error {
	accepted = mergeConsentAcceptances(accepted)
	if len(accepted) == 0 {
		return nil
	}
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return fmt.Errorf("signup record consents: accounts is not *pgCoreAuth")
	}
	localeByKey := pgCA.coreAuthConsentLocales(ctx, f.ProjectID, "")
	env := f.Environment
	if env == "" {
		env = coreAuthDefaultEnv
	}
	configured, _ := a.cfg.ConsentConfig(ctx, f.ProjectID)
	allowedConsent := make(map[string]struct{}, len(configured))
	for _, d := range configured {
		allowedConsent[d.Key+"\x00"+d.Version] = struct{}{}
	}
	now := nowUTC()
	recorded := false
	for _, acc := range accepted {
		if _, ok := allowedConsent[acc.Key+"\x00"+acc.Version]; !ok {
			continue
		}
		setter := &models.IamConsentSetter{
			ID:          ptr(newUUID()),
			ProjectID:   ptr(f.ProjectID),
			Environment: &env,
			UserID:      ptr(f.UserID),
			DocKey:      ptr(acc.Key),
			Version:     ptr(acc.Version),
			AcceptedAt:  ptr(now),
		}
		if loc := localeByKey[acc.Key]; loc != "" {
			setter.Locale = ptr(null.From(loc))
		}
		if _, err := models.IamConsents.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return fmt.Errorf("signup record consents: insert consent: %w", err)
		}
		recorded = true
	}
	if !recorded {
		return nil
	}
	return a.emitter.Emit(ctx, domain.Event{
		Type:        "account.consents_accepted",
		ProjectID:   f.ProjectID,
		Environment: env,
		AggregateID: f.UserID,
		Payload:     map[string]any{"account_id": f.UserID, "project_id": f.ProjectID, "accepted": accepted},
	})
}

func (a *pgCoreAuthFlows) signupCompleteWithExistingSession(ctx context.Context, row *models.IamFlow, f *domain.Flow, accepted []domain.AccountConsentAcceptance, sess *domain.Session) (*domain.FlowState, error) {
	f.ConsentsRequired = nil
	f.ConsentsAccepted = nil
	f.Status = domain.FlowStatusCompleted
	f.Step = domain.FlowStepCompleted
	f.Error = nil
	newToken, err := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		if err := a.signupRecordAcceptedConsents(ctx, f, accepted); err != nil {
			return "", err
		}
		return a.flowRotate(ctx, row, f)
	})
	if err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: newToken, Flow: f, Session: sess}, nil
}

func (a *pgCoreAuthFlows) signupCompleteWithFreshSession(ctx context.Context, row *models.IamFlow, f *domain.Flow, accepted []domain.AccountConsentAcceptance) (*domain.FlowState, error) {
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("signup complete with consents: accounts is not *pgCoreAuth")
	}
	f.ConsentsRequired = nil
	f.ConsentsAccepted = nil
	f.Status = domain.FlowStatusCompleted
	f.Step = domain.FlowStepCompleted
	f.Error = nil
	type result struct {
		token string
		sess  *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		if err := a.signupRecordAcceptedConsents(ctx, f, accepted); err != nil {
			return result{}, err
		}
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), f.UserID)
		if err != nil {
			return result{}, fmt.Errorf("signup complete with consents: load user: %w", err)
		}
		acc, err := coreAuthLoadAccount(userRow, f.ProjectID)
		if err != nil {
			return result{}, fmt.Errorf("signup complete with consents: parse account: %w", err)
		}
		sess, err := pgCA.coreAuthMintSession(ctx, acc, "", []string{"pwd"}, 1)
		if err != nil {
			return result{}, fmt.Errorf("signup complete with consents: mint session: %w", err)
		}
		token, err := a.flowRotate(ctx, row, f)
		if err != nil {
			return result{}, err
		}
		return result{token: token, sess: sess}, nil
	})
	if err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: res.token, Flow: f, Session: res.sess}, nil
}

// flowCompleteOrGateConsents is the shared post-identity terminus for signup. If
// the flow still has unaccepted required consents it halts at accept_consents
// (no token rotation, no session surfaced — the user is not fully registered
// yet); otherwise it records accepted consents, completes the flow, rotates the
// token (§5 rule 2), and surfaces a session.
func (a *pgCoreAuthFlows) flowCompleteOrGateConsents(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd, sess *domain.Session) (*domain.FlowState, error) {
	required := flowRequiredConsentDocs(f)
	if len(required) > 0 {
		if missing := missingRequiredConsents(required, f.ConsentsAccepted); len(missing) == 0 {
			if err := a.signupRevokeProvisionalSession(ctx, f, sess); err != nil {
				return nil, err
			}
			return a.signupCompleteWithFreshSession(ctx, row, f, f.ConsentsAccepted)
		}
		// The identity step (verify_email / set_password) already minted a session,
		// but the user is NOT fully registered until required consents are accepted.
		// Revoke that session now so no valid (even if unsurfaced) session/refresh
		// token lingers for an unregistered user; accept_consents mints the real one.
		if err := a.signupRevokeProvisionalSession(ctx, f, sess); err != nil {
			return nil, err
		}
		f.Step = domain.FlowStepAcceptConsents
		f.Error = nil
		if err := a.db.withTx(ctx, func(ctx context.Context) error {
			return a.flowSave(ctx, row, f)
		}); err != nil {
			return nil, err
		}
		return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
	}
	return a.signupCompleteWithExistingSession(ctx, row, f, f.ConsentsAccepted, sess)
}

// signupAcceptConsents handles the accept_consents step: it validates that every
// required consent has been accepted (exact key+version), records the
// acceptances in iam_consents, emits account.consents_accepted, then completes
// the flow with a fresh session (mirroring signupSetPassword's mint+rotate).
//
// The submit payload carries acceptances under payload["accept"] as a JSON array
// of {"key","version"} objects (e.g. payload: {"accept":[{"key":"tos",
// "version":"2026-06-01"}]}). Legacy clients that sent payload["consents"] are
// accepted as an alias. A client that can only send string scalars may instead
// pass the array as a JSON-encoded string; both forms are accepted.
// Validation is against f.ConsentsRequired (server truth); a client cannot
// bypass the gate by omitting or faking entries.
func (a *pgCoreAuthFlows) signupAcceptConsents(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	if f.UserID == "" {
		return nil, domain.ErrBadRequest.WithMessage("no verified user for signup")
	}
	rawAccept := cmd.Payload["accept"]
	if rawAccept == "" {
		rawAccept = cmd.Payload["consents"]
	}
	submitted, err := parseConsentAccept(rawAccept)
	if err != nil {
		return nil, err
	}
	accepted := mergeConsentAcceptances(f.ConsentsAccepted, submitted)

	required := flowRequiredConsentDocs(f)
	if missing := missingRequiredConsents(required, accepted); len(missing) > 0 {
		return nil, domain.ErrConsentRequired.WithDetails(consentRefDetails(missing))
	}
	return a.signupCompleteWithFreshSession(ctx, row, f, accepted)
}
