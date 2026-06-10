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

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

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
		return nil, fmt.Errorf("recovery flow: accounts adapter is not *pgCoreAuth")
	}

	now := nowUTC()
	f.Step = domain.FlowStepVerifyEmail

	// Try to locate the account. We handle the two paths identically on the
	// wire (anti-enumeration). The error branch still emits the same shape.
	var ac *domain.FlowActiveChallenge

	userRow, err := pgCA.coreAuthFindUserByEmail(ctx, cmd.ProjectID, cmd.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("recovery create: lookup: %w", err)
	}

	if err == nil {
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
			Locale:      cmd.Locale,
			ExpiresAt:   now.Add(coreAuthChallengeTTL),
			CreatedAt:   now,
		}

		if err := a.db.withTx(ctx, func(ctx context.Context) error {
			if _, insErr := pgCA.coreAuthInsertChallenge(ctx, ch); insErr != nil {
				return insErr
			}
			return pgCA.emitter.Emit(ctx, domain.Event{
				Type:        "password.reset_requested",
				ProjectID:   cmd.ProjectID,
				Environment: f.Environment,
				AggregateID: acc.ID,
				Payload: map[string]any{
					"code":         code,
					"token":        token,
					"account_id":   acc.ID,
					"challenge_id": ch.ID,
					"contact":      ch.Subject,
					"to":           ch.Subject,
					"locale":       cmd.Locale,
					"purpose":      ch.Purpose,
				},
			})
		}); err != nil {
			return nil, fmt.Errorf("recovery create: issue challenge: %w", err)
		}

		ac = &domain.FlowActiveChallenge{
			ChallengeID:  ch.ID,
			Channel:      "email",
			ExpiresAt:    ch.ExpiresAt,
			ResendAt:     now.Add(flowResendCooloff),
			AttemptsLeft: flowMaxAttempts,
		}
	} else {
		// Unknown email: synthesise a fake descriptor (random ID, no DB row).
		// The client gets identical shape; any code submitted will fail.
		ac = &domain.FlowActiveChallenge{
			ChallengeID:  newUUID(), // dangling — no DB row
			Channel:      "email",
			ExpiresAt:    now.Add(coreAuthChallengeTTL),
			ResendAt:     now.Add(flowResendCooloff),
			AttemptsLeft: flowMaxAttempts,
		}
	}

	f.ActiveChallenge = ac

	token, hash, err := flowMintToken()
	if err != nil {
		return nil, fmt.Errorf("recovery create: mint token: %w", err)
	}
	if err := a.flowInsert(ctx, f, hash, flowData{
		Contact:         f.Contact,
		Collected:       f.Collected,
		ActiveChallenge: f.ActiveChallenge,
	}); err != nil {
		return nil, fmt.Errorf("recovery create: insert flow: %w", err)
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// ─── advance ─────────────────────────────────────────────────────────────────

// advanceRecovery routes Submit actions to the correct step handler.
func advanceRecovery(ctx context.Context, a *pgCoreAuthFlows, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	switch f.Step {
	case domain.FlowStepVerifyEmail:
		if cmd.Action != "verify_email" {
			return nil, domain.ErrBadRequest.WithMessage("expected action verify_email")
		}
		return a.recoveryVerifyEmail(ctx, row, f, cmd)
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

// recoveryVerifyEmail handles the OTP verification step. On a correct code the
// password_reset challenge is consumed and the flow advances to set_password.
// Wrong code decrements attempts and embeds an error — the flow stays pending
// and the token does NOT rotate (§5 rule 6). Non-existent-user flows always
// fail identically (anti-enumeration §5.4).
func (a *pgCoreAuthFlows) recoveryVerifyEmail(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
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

	// Type-assert for internal challenge access. Safe: same package.
	pgCA, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("recovery verify_email: accounts is not *pgCoreAuth")
	}

	// Attempt to load and consume the challenge inside a transaction.
	// We use coreAuthConsumeChallenge because it validates, marks consumed, and
	// returns the challenge data (including account_id). If the challenge_id is
	// dangling (fake) or the code is wrong, we treat it as invalid_code.
	type consumeResult struct {
		accountID string
	}
	res, consumeErr := withTxRet(ctx, a.db, func(ctx context.Context) (consumeResult, error) {
		_, data, err := pgCA.coreAuthConsumeChallenge(ctx, f.ProjectID, domain.CoreAuthVerifyConsumeCmd{
			ProjectID:   f.ProjectID,
			ChallengeID: ac.ChallengeID,
			Code:        code,
		}, "password_reset")
		if err != nil {
			return consumeResult{}, err
		}
		return consumeResult{accountID: data.AccountID}, nil
	})

	if consumeErr != nil {
		// Wrong code or challenge not found / consumed: decrement attempts, embed error.
		ac.AttemptsLeft--
		f.Error = &domain.FlowError{Code: "invalid_code", Message: "The verification code is incorrect."}
		_ = a.db.withTx(ctx, func(ctx context.Context) error {
			return a.flowSave(ctx, row, f)
		})
		return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
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
		return nil, fmt.Errorf("recovery set_password: accounts is not *pgCoreAuth")
	}

	type setResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (setResult, error) {
		// Load the account so we can mint a session.
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), f.UserID)
		if err != nil {
			return setResult{}, fmt.Errorf("recovery set_password: load user: %w", err)
		}
		acc, err := coreAuthLoadAccount(userRow, f.ProjectID)
		if err != nil {
			return setResult{}, fmt.Errorf("recovery set_password: parse account: %w", err)
		}

		// Hash and write the new password credential (§5 rule 5: never stored in data).
		hash, err := coreAuthHashPassword(password)
		if err != nil {
			return setResult{}, fmt.Errorf("recovery set_password: hash password: %w", err)
		}
		if err := pgCA.coreAuthUpsertPasswordCredential(ctx, acc.ProjectID, acc.ID, hash); err != nil {
			return setResult{}, fmt.Errorf("recovery set_password: upsert credential: %w", err)
		}

		// Revoke all existing sessions for safety (mirrors ResetPassword behaviour).
		if _, err := pgCA.coreAuthSignOutAll(ctx, acc.ProjectID, acc.ID, ""); err != nil {
			return setResult{}, fmt.Errorf("recovery set_password: sign out all: %w", err)
		}

		// Mint a fresh session.
		sess, err := pgCA.coreAuthMintSession(ctx, acc, "", []string{"pwd"}, 1)
		if err != nil {
			return setResult{}, fmt.Errorf("recovery set_password: mint session: %w", err)
		}

		if err := pgCA.emitter.Emit(ctx, domain.Event{
			Type:        "password.reset",
			ProjectID:   acc.ProjectID,
			Environment: f.Environment,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return setResult{}, err
		}
		return setResult{acc: acc, sess: sess}, nil
	})
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

	return &domain.FlowState{FlowToken: newToken, Flow: f, Session: res.sess}, nil
}
