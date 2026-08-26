package postgres

// coreauth_flows_recovery_pg.go — recovery (forgot-password) kind for the
// server-side resumable auth flow engine (§7).
//
// State machine:
//   create{email} → step=verify_email (always; anti-enumeration §5.4)
//   submit{verify_email, code}        → step=set_password (if correct OTP)
//   submit{set_password, password}    → status=completed + session
//
// Anti-enumeration (§5.4): create ALWAYS returns the same FlowState shape
// regardless of whether the email maps to a real account. Internally:
//   - real user   → "password_reset" challenge row inserted, code emitted
//   - unknown email → no DB row; fake challenge descriptor with a random ID
//
// At verify_email: wrong-code for a non-existent user and wrong-code for a
// real user both return identical {"error":"invalid_code"} responses. The
// challenge lookup for a fake ID fails silently and is treated as invalid_code.
//
// Security §5 mapping:
//   §5.1 token ≥256-bit random "ftk_" — flowMintToken
//   §5.2 rotation — flowRotate at set_password (privilege step)
//   §5.3 tenant+TTL — flowLoad (called by Submit in the engine)
//   §5.4 anti-enumeration — uniform create response; see above
//   §5.5 no raw password in data — password passed directly to hash/upsert
//   §5.6 attempts/lockout — AttemptsLeft in ActiveChallenge
//   §5.7 resend rate-limit — ResendAt in ActiveChallenge (engine Resend method)
//   §5.8 session on completion — flowRotate → FlowState.Session
//   §5.10 masking — contact shown as masked email only

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// The flow kinds register themselves so a kind lives entirely in its own file:
// adding one means adding a file, not editing a dispatch table somewhere else.
//
//nolint:gochecknoinits // a registration table, populated once at load.
func init() {
	flowCreators[domain.FlowKindRecovery] = createRecovery
	flowAdvancers[domain.FlowKindRecovery] = advanceRecovery
}

// ─── create ──────────────────────────────────────────────────────────────────

// createRecovery handles POST /v1/auth/flows with kind=recovery.
// Anti-enumeration contract: always persists at step=verify_email and returns
// the same FlowState shape. The difference between a real and a fake user
// is invisible to the caller.
func createRecovery(ctx context.Context, a *pgCoreAuthFlows, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	// Type-assert to access internal pgCoreAuth helpers. Both adapters live in
	// the same postgres package; this assertion is safe within the package.
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("recovery flow: %w", errAccountsNotPgCoreAuth)
	}

	// Channel dispatch: phone-OTP recovery mirrors the email path with an SMS
	// challenge; default is the email reset.
	if cmd.Method == "phone_otp" {
		return a.createRecoveryPhone(ctx, pgCA, f, cmd)
	}

	now := nowUTC()
	f.Method = coreAuthChallengeEmail
	f.Step = domain.FlowStepVerifyEmail

	ac, err := recoveryEmailChallenge(ctx, a, pgCA, f, cmd, now)
	if err != nil {
		return nil, err
	}

	return finalizeRecoveryFlow(ctx, a, f, ac, flowData{
		Contact:   f.Contact,
		Collected: f.Collected,
	}, "recovery create")
}

// recoveryEmailChallenge resolves the challenge descriptor for an email
// recovery: a real account gets a persisted "password_reset" challenge and
// the email is queued; an unknown one gets a dangling fake descriptor so the
// two paths are indistinguishable on the wire (anti-enumeration, §5.4). Sets
// f.UserID as a side effect when the account is real.
func recoveryEmailChallenge(ctx context.Context, a *pgCoreAuthFlows, pgCA *pgCoreAuth, f *domain.Flow, cmd domain.FlowCreateCmd, now time.Time) (*domain.FlowActiveChallenge, error) {
	userRow, err := pgCA.coreAuthFindUserByEmail(ctx, cmd.ProjectID, cmd.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("recovery create: lookup: %w", err)
	}

	if err != nil {
		// Unknown email: synthesize a fake descriptor (random ID, no DB row).
		// The client gets identical shape; any code submitted will fail.
		return &domain.FlowActiveChallenge{ //nolint:nilerr // anti-enumeration, see func doc
			ChallengeID:  newUUID(), // dangling — no DB row
			Channel:      coreAuthChallengeEmail,
			ExpiresAt:    now.Add(coreAuthChallengeTTL),
			ResendAt:     now.Add(flowResendCooloff),
			AttemptsLeft: flowMaxAttempts,
		}, nil
	}

	// Real user: issue a password_reset challenge.
	acc, loadErr := coreAuthLoadAccount(userRow, cmd.ProjectID)
	if loadErr != nil {
		return nil, fmt.Errorf("recovery create: load account: %w", loadErr)
	}

	f.UserID = acc.ID

	code, codeErr := coreAuthRandomCode()
	if codeErr != nil {
		return nil, fmt.Errorf("recovery create: random code: %w", codeErr)
	}

	token, tokenErr := coreAuthRandomToken()
	if tokenErr != nil {
		return nil, fmt.Errorf("recovery create: random token: %w", tokenErr)
	}

	ch := coreAuthChallengeData{
		ID:          newUUID(),
		ProjectID:   cmd.ProjectID,
		Environment: f.Environment,
		Type:        "password_reset",
		Purpose:     "reset",
		AccountID:   acc.ID,
		Subject:     cmd.Email,
		CodeHash:    coreAuthSHA256(code),
		TokenHash:   coreAuthSHA256(token),
		RedirectTo:  cmd.RedirectTo,
		Locale:      cmd.Locale,
		ExpiresAt:   now.Add(coreAuthChallengeTTL),
		CreatedAt:   now,
	}

	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		_, insErr := pgCA.coreAuthInsertChallenge(ctx, ch)

		return insErr
	}); err != nil {
		return nil, fmt.Errorf("recovery create: issue challenge: %w", err)
	}

	return &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      coreAuthChallengeEmail,
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
		Code:         code,
		Token:        token,
	}, nil
}

// finalizeRecoveryFlow stores the resolved active challenge, mints the flow
// token, and persists the flow row — the shared tail of both recovery
// channels (email, phone_otp).
func finalizeRecoveryFlow(ctx context.Context, a *pgCoreAuthFlows, f *domain.Flow, ac *domain.FlowActiveChallenge, fd flowData, errPrefix string) (*domain.FlowState, error) {
	f.ActiveChallenge = ac
	fd.ActiveChallenge = ac

	token, hash, err := flowMintToken()
	if err != nil {
		return nil, fmt.Errorf("%s: mint token: %w", errPrefix, err)
	}

	if err := a.flowInsert(ctx, f, hash, fd); err != nil {
		return nil, fmt.Errorf("%s: insert flow: %w", errPrefix, err)
	}

	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// ─── create: phone-OTP channel ────────────────────────────────────────────────

// createRecoveryPhone mirrors createRecovery over an SMS challenge. Same
// anti-enumeration contract: a real phone gets a "phone" reset challenge; an
// unknown phone gets a dangling fake descriptor. Requires an enabled SMS
// provider (pre-flight) so the code can actually be delivered.
func (a *pgCoreAuthFlows) createRecoveryPhone(ctx context.Context, pgCA *pgCoreAuth, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	now := nowUTC()
	f.Method = "phone_otp"
	f.Step = domain.FlowStepVerifyPhone

	phone := strings.TrimSpace(cmd.Phone)
	if !e164.MatchString(phone) {
		return nil, domain.ErrBadRequest.WithMessage("phone must be a valid E.164 number")
	}

	ok, err := providerEnabled(ctx, a.db, cmd.ProjectID, "sms")
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, domain.ErrValidation.WithMessage("an enabled sms provider is required for phone recovery")
	}

	ac, err := recoveryPhoneChallenge(ctx, pgCA, f, cmd, phone, now)
	if err != nil {
		return nil, err
	}

	return finalizeRecoveryFlow(ctx, a, f, ac, flowData{
		Contact:   f.Contact,
		Collected: f.Collected,
		Method:    f.Method,
	}, "recovery create phone")
}

// issuePhoneResetChallenge persists ch and queues the SMS carrying its code,
// atomically so a delivery failure leaves no dangling challenge row.
func issuePhoneResetChallenge(ctx context.Context, pgCA *pgCoreAuth, ch coreAuthChallengeData, environment, accountID, phone, locale, code string) error {
	return pgCA.db.withTx(ctx, func(ctx context.Context) error {
		if _, insErr := pgCA.coreAuthInsertChallenge(ctx, ch); insErr != nil {
			return insErr
		}

		return pgCA.emitter.Emit(ctx, domain.Event{
			Type:        "phone.verification.requested",
			ProjectID:   ch.ProjectID,
			Environment: environment,
			AggregateID: accountID,
			Payload: map[string]any{
				"code":         code,
				"channel":      "sms",
				"purpose":      "reset",
				"account_id":   accountID,
				"challenge_id": ch.ID,
				"contact":      phone,
				"to":           phone,
				"locale":       locale,
			},
		})
	})
}

// recoveryPhoneChallenge is recoveryEmailChallenge's SMS counterpart: a real
// phone gets a persisted "phone" reset challenge and an SMS queued, an
// unknown one gets a dangling fake descriptor. Sets f.UserID as a side effect
// when the account is real.
func recoveryPhoneChallenge(ctx context.Context, pgCA *pgCoreAuth, f *domain.Flow, cmd domain.FlowCreateCmd, phone string, now time.Time) (*domain.FlowActiveChallenge, error) {
	userRow, err := pgCA.coreAuthFindUserByPhone(ctx, cmd.ProjectID, phone)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("recovery create phone: lookup: %w", err)
	}

	if err != nil {
		return &domain.FlowActiveChallenge{ //nolint:nilerr // anti-enumeration, see func doc
			ChallengeID:  newUUID(), // dangling — no DB row
			Channel:      "sms",
			ExpiresAt:    now.Add(coreAuthChallengeTTL),
			ResendAt:     now.Add(flowResendCooloff),
			AttemptsLeft: flowMaxAttempts,
		}, nil
	}

	acc, loadErr := coreAuthLoadAccount(userRow, cmd.ProjectID)
	if loadErr != nil {
		return nil, fmt.Errorf("recovery create phone: load account: %w", loadErr)
	}

	f.UserID = acc.ID

	code, codeErr := coreAuthRandomCode()
	if codeErr != nil {
		return nil, fmt.Errorf("recovery create phone: random code: %w", codeErr)
	}

	token, tokenErr := coreAuthRandomToken()
	if tokenErr != nil {
		return nil, fmt.Errorf("recovery create phone: random token: %w", tokenErr)
	}

	ch := coreAuthChallengeData{
		ID:          newUUID(),
		ProjectID:   cmd.ProjectID,
		Environment: f.Environment,
		Type:        "phone",
		Purpose:     "reset",
		AccountID:   acc.ID,
		Subject:     phone,
		CodeHash:    coreAuthSHA256(code),
		TokenHash:   coreAuthSHA256(token),
		Locale:      cmd.Locale,
		ExpiresAt:   now.Add(coreAuthChallengeTTL),
		CreatedAt:   now,
	}

	if err := issuePhoneResetChallenge(ctx, pgCA, ch, f.Environment, acc.ID, phone, cmd.Locale, code); err != nil {
		return nil, fmt.Errorf("recovery create phone: issue challenge: %w", err)
	}

	return &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      "sms",
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
	}, nil
}

// ─── advance ─────────────────────────────────────────────────────────────────

// advanceRecovery routes Submit actions to the correct step handler.
func advanceRecovery(ctx context.Context, a *pgCoreAuthFlows, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	//nolint:exhaustive // only the three steps a recovery flow can be
	// submitted from advance here; every other step is a caller error, which
	// the default case reports as such.
	switch f.Step {
	case domain.FlowStepVerifyEmail:
		if cmd.Action != flowActionVerifyEmail {
			return nil, domain.ErrBadRequest.WithMessage("expected action verify_email")
		}

		return a.recoveryVerifyEmail(ctx, row, f, cmd)
	case domain.FlowStepVerifyPhone:
		if cmd.Action != "verify_phone" {
			return nil, domain.ErrBadRequest.WithMessage("expected action verify_phone")
		}

		return a.recoveryVerifyPhone(ctx, row, f, cmd)
	case domain.FlowStepSetPassword:
		if cmd.Action != "set_password" {
			return nil, domain.ErrBadRequest.WithMessage("expected action set_password")
		}

		return a.recoverySetPassword(ctx, row, f, cmd)
	default:
		return nil, domain.ErrBadRequest.WithMessage(fmt.Sprintf("unexpected step %q for recovery", f.Step))
	}
}

// ─── step: verify_email ───────────────────────────────────────────────────────

// recoveryVerifyEmail handles the email verification step. On a correct code or
// link token the password_reset challenge is consumed and the flow advances to
// set_password. Wrong secrets decrement attempts and embed an error — the flow
// stays pending and the token does NOT rotate (§5 rule 6). Non-existent-user
// flows always fail identically (anti-enumeration §5.4).
func (a *pgCoreAuthFlows) recoveryVerifyEmail(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	activeChallenge := f.ActiveChallenge
	if activeChallenge == nil {
		return nil, domain.ErrBadRequest.WithMessage("no active email challenge")
	}

	code, token := flowVerificationSecret(cmd.Payload)
	if code == "" && token == "" {
		return nil, domain.ErrBadRequest.WithMessage("code or token is required")
	}

	if activeChallenge.AttemptsLeft <= 0 {
		return nil, domain.ErrChallengeInvalid.WithMessage("challenge exhausted; please resend")
	}

	// Type-assert for internal challenge access. Safe: same package.
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("recovery verify_email: %w", errAccountsNotPgCoreAuth)
	}

	// Attempt to load and consume the challenge inside a transaction.
	// We use coreAuthConsumeChallenge because it validates, marks consumed, and
	// returns the challenge data (including account_id). If the challenge_id is
	// dangling (fake) or the code is wrong, we treat it as invalid_code.
	type consumeResult struct {
		accountID string
	}

	res, consumeErr := withTxRet(ctx, a.db, func(ctx context.Context) (consumeResult, error) {
		_, data, err := pgCA.coreAuthConsumeChallenge(ctx, f.ProjectID, flowVerifyConsumeCmd(f.ProjectID, "", activeChallenge.ChallengeID, code, token), "password_reset")
		if err != nil {
			return consumeResult{}, err
		}

		return consumeResult{accountID: data.AccountID}, nil
	})
	if consumeErr != nil {
		// Wrong code/token or challenge not found / consumed: decrement attempts, embed error.
		activeChallenge.AttemptsLeft--
		f.Error = &domain.FlowError{Code: "invalid_code", Message: "The verification code is incorrect."}
		_ = a.db.withTx(ctx, func(ctx context.Context) error {
			return a.flowSave(ctx, row, f)
		})

		return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil //nolint:nilerr // wrong code stays pending, see above
	}

	// Code verified — advance to set_password. Do NOT rotate yet (token rotates
	// only on the privilege-granting set_password step, §5 rule 2).
	f.UserID = res.accountID
	f.Step = domain.FlowStepSetPassword
	f.ActiveChallenge = nil
	f.Error = nil

	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		return a.flowSave(ctx, row, f)
	}); err != nil {
		return nil, err
	}

	return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
}

// ─── step: verify_phone ───────────────────────────────────────────────────────

// recoveryVerifyPhone is recoveryVerifyEmail over an SMS "phone" challenge. On a
// correct code it advances to set_password; wrong code decrements attempts and
// stays pending (anti-enumeration identical to the email path).
func (a *pgCoreAuthFlows) recoveryVerifyPhone(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	activeChallenge := f.ActiveChallenge
	if activeChallenge == nil {
		return nil, domain.ErrBadRequest.WithMessage("no active phone challenge")
	}

	code := cmd.Payload["code"]
	if code == "" {
		return nil, domain.ErrBadRequest.WithMessage("code is required")
	}

	if activeChallenge.AttemptsLeft <= 0 {
		return nil, domain.ErrChallengeInvalid.WithMessage("challenge exhausted; please resend")
	}

	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("recovery verify_phone: %w", errAccountsNotPgCoreAuth)
	}

	res, consumeErr := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		_, data, err := pgCA.coreAuthConsumeChallenge(ctx, f.ProjectID, domain.CoreAuthVerifyConsumeCmd{
			ProjectID:   f.ProjectID,
			ChallengeID: activeChallenge.ChallengeID,
			Code:        code,
		}, "phone")
		if err != nil {
			return "", err
		}

		return data.AccountID, nil
	})
	if consumeErr != nil {
		activeChallenge.AttemptsLeft--
		f.Error = &domain.FlowError{Code: "invalid_code", Message: "The verification code is incorrect."}
		_ = a.db.withTx(ctx, func(ctx context.Context) error {
			return a.flowSave(ctx, row, f)
		})

		return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil //nolint:nilerr // wrong code stays pending, see above
	}

	f.UserID = res
	f.Step = domain.FlowStepSetPassword
	f.ActiveChallenge = nil
	f.Error = nil

	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		return a.flowSave(ctx, row, f)
	}); err != nil {
		return nil, err
	}

	return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
}

// ─── step: set_password ───────────────────────────────────────────────────────

// recoverySetPassword handles the new-password step. On success the flow is
// completed, the token is ROTATED (new token → new session; old token dead),
// and a session is returned in FlowState (§5 rules 2, 8).
//
// The password is passed directly to bcrypt via coreAuthHashPassword and then
// written with coreAuthUpsertPasswordCredential — it is NEVER stored in flow
// data (§5 rule 5).
func (a *pgCoreAuthFlows) recoverySetPassword(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	if f.UserID == "" {
		// Should not happen if the state machine is followed correctly.
		return nil, domain.ErrBadRequest.WithMessage("no verified user for recovery")
	}

	password := cmd.Payload["password"]
	if password == "" {
		return nil, domain.ErrBadRequest.WithMessage("password is required")
	}

	// Type-assert for internal session/credential helpers. Same package.
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("recovery set_password: %w", errAccountsNotPgCoreAuth)
	}

	sess, err := recoveryResetPassword(ctx, a, pgCA, f, password)
	if err != nil {
		return nil, err
	}

	// Complete the flow and ROTATE the token (§5 rule 2 — session-minting step).
	f.Status = domain.FlowStatusCompleted
	f.Step = domain.FlowStepCompleted
	f.ActiveChallenge = nil
	f.Error = nil

	newToken, err := withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		return a.flowRotate(ctx, row, f)
	})
	if err != nil {
		return nil, err
	}

	// Annotate collected (password was set).
	f.Collected.HasPassword = true

	return &domain.FlowState{FlowToken: newToken, Flow: f, Session: sess}, nil
}

// recoveryResetPassword loads the verified user, enforces the password
// policy, writes the new credential, revokes existing sessions, and mints a
// fresh one — all inside one transaction so a mid-way failure leaves the old
// credential and sessions intact.
func recoveryResetPassword(ctx context.Context, a *pgCoreAuthFlows, pgCA *pgCoreAuth, f *domain.Flow, password string) (*domain.Session, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Session, error) {
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), f.UserID)
		if err != nil {
			return nil, fmt.Errorf("recovery set_password: load user: %w", err)
		}

		acc, err := coreAuthLoadAccount(userRow, f.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("recovery set_password: parse account: %w", err)
		}

		if err := pgCA.coreAuthEnforcePasswordPolicy(ctx, acc.ProjectID, password); err != nil {
			return nil, err
		}

		// Hash and write the new password credential (§5 rule 5: never stored in data).
		hash, err := coreAuthHashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("recovery set_password: hash password: %w", err)
		}

		if err := pgCA.coreAuthUpsertPasswordCredential(ctx, acc.ProjectID, acc.ID, hash); err != nil {
			return nil, fmt.Errorf("recovery set_password: upsert credential: %w", err)
		}

		// Revoke all existing sessions for safety (mirrors ResetPassword behavior).
		if _, err := pgCA.coreAuthSignOutAll(ctx, acc.ProjectID, acc.ID, ""); err != nil {
			return nil, fmt.Errorf("recovery set_password: sign out all: %w", err)
		}

		// Mint a fresh session.
		sess, err := pgCA.coreAuthMintSession(ctx, acc, "", []string{"pwd"}, 1)
		if err != nil {
			return nil, fmt.Errorf("recovery set_password: mint session: %w", err)
		}

		if err := pgCA.emitter.Emit(ctx, domain.Event{
			Type:        "password.reset",
			ProjectID:   acc.ProjectID,
			Environment: f.Environment,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}

		return sess, nil
	})
}
