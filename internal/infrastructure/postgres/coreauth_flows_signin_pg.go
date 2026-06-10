package postgres

// coreauth_flows_signin_pg.go — SIGNIN kind of the server-side resumable auth
// flow engine (§7 signin machine, §5 security).
//
// State machine:
//
//	create{email,password}
//	  → AuthenticatePassword (wrong → ErrInvalidCredentials, anti-enumeration §5.4)
//	  → !MFARequired → completed (flowInsert at completed, session returned)
//	  → MFARequired  → mfa_required (flowInsert, challenge issued, no session yet)
//
//	submit{action:"mfa", payload:{code}} at step mfa_required
//	  → mfa.Verify(challengeID, code)
//	  → success → completed, token rotated (§5 rule 2), session returned
//	  → wrong   → AttemptsLeft--, flowSave, pending + error{invalid_code}
//
// Security mapping (§5):
//   - §5.1 token: flowMintToken (≥256-bit, ftk_ prefix, only hash stored).
//   - §5.2 rotation: flowRotate on the session-minting (MFA verify) step.
//     For the no-MFA path a single token is minted and the flow is immediately
//     completed — no rotation is needed as there is no subsequent transition.
//   - §5.3 tenant+TTL: enforced by flowLoad in the caller (Submit) path.
//   - §5.4 anti-enumeration: AuthenticatePassword returns ErrInvalidCredentials
//     for both unknown-user and wrong-password; we propagate it unchanged.
//   - §5.5 no raw secrets: password never stored in data; has_password=true only.
//   - §5.6 attempts: AttemptsLeft tracked in ActiveChallenge, decremented on wrong code.
//   - §5.8 session once: session returned only when status=completed.

import (
	"context"
	"fmt"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

func init() {
	flowCreators[domain.FlowKindSignin] = createSignin
	flowAdvancers[domain.FlowKindSignin] = advanceSignin
}

// ─── create ──────────────────────────────────────────────────────────────────

// createSignin is the flowCreateFn for SIGNIN. It immediately authenticates the
// password credential; if MFA is required it issues a challenge and persists at
// step=mfa_required; otherwise it completes the flow in a single round-trip.
func createSignin(ctx context.Context, a *pgCoreAuthFlows, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	// 1. Verify password. Returns ErrInvalidCredentials for both unknown-user
	//    and wrong-password — propagate as-is (§5.4 anti-enumeration).
	result, err := a.accounts.AuthenticatePassword(ctx, f.ProjectID, cmd.Email, cmd.Password)
	if err != nil {
		return nil, err // includes ErrInvalidCredentials
	}

	// 2. No MFA required: complete the flow immediately. This is the first and
	//    only token issued, so no rotation is needed. Persist at completed status.
	if !result.MFARequired {
		f.UserID = result.Account.ID
		f.Status = domain.FlowStatusCompleted
		f.Step = domain.FlowStepCompleted

		token, hash, err := flowMintToken()
		if err != nil {
			return nil, fmt.Errorf("flow signin create (no-mfa): mint token: %w", err)
		}
		if err := a.flowInsert(ctx, f, hash, flowData{
			Contact:   f.Contact,
			Collected: f.Collected,
		}); err != nil {
			return nil, err
		}
		return &domain.FlowState{FlowToken: token, Flow: f, Session: result.Session}, nil
	}

	// 3. MFA required: pick the primary factor and issue a challenge.
	primaryFactorID := mfaPrimaryFactorID(result.Factors)
	if primaryFactorID == "" {
		// Shouldn't happen: AuthenticatePassword only sets MFARequired when
		// there are active factors; guard defensively.
		return nil, domain.ErrMFAInvalid
	}

	mfa := NewPgMFAAccounts(a.db, a.emitter)
	ch, err := mfa.Challenge(ctx, result.Account.ID, primaryFactorID)
	if err != nil {
		return nil, fmt.Errorf("flow signin create: issue mfa challenge: %w", err)
	}

	now := nowUTC()
	f.UserID = result.Account.ID
	f.Step = domain.FlowStepMFARequired
	f.ActiveChallenge = &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      ch.Type, // "email" | "totp" | "webauthn" | "sms"
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
	}

	// 4. Mint the initial flow token and persist the pending row.
	token, hash, err := flowMintToken()
	if err != nil {
		return nil, fmt.Errorf("flow signin create (mfa): mint token: %w", err)
	}
	if err := a.flowInsert(ctx, f, hash, flowData{
		Contact:         f.Contact,
		Collected:       f.Collected,
		ActiveChallenge: f.ActiveChallenge,
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// ─── advance ─────────────────────────────────────────────────────────────────

// advanceSignin is the flowAdvanceFn for SIGNIN.
func advanceSignin(ctx context.Context, a *pgCoreAuthFlows, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	switch f.Step {
	case domain.FlowStepMFARequired:
		if cmd.Action != "mfa" {
			return nil, domain.ErrBadRequest.WithMessage(`expected action "mfa" at step mfa_required`)
		}
		return a.signinVerifyMFA(ctx, row, f, cmd)
	default:
		return nil, domain.ErrBadRequest.WithMessage(fmt.Sprintf("unexpected step %q for signin", f.Step))
	}
}

// signinVerifyMFA verifies the MFA code via mfa.Verify. On success it rotates
// the flow token (§5 rule 2) and completes the flow, returning the session.
// On a wrong code it decrements AttemptsLeft, embeds error{invalid_code}, and
// returns the pending FlowState without a session (mirrors signupVerifyEmail).
func (a *pgCoreAuthFlows) signinVerifyMFA(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	ac := f.ActiveChallenge
	if ac == nil {
		return nil, domain.ErrBadRequest.WithMessage("no active MFA challenge")
	}
	code := cmd.Payload["code"]
	if code == "" {
		return nil, domain.ErrBadRequest.WithMessage("code is required")
	}
	if ac.AttemptsLeft <= 0 {
		return nil, domain.ErrChallengeInvalid.WithMessage("challenge exhausted; please resend")
	}

	mfa := NewPgMFAAccounts(a.db, a.emitter)
	_, sess, err := mfa.Verify(ctx, ac.ChallengeID, code)
	if err != nil {
		// Wrong code: decrement attempts, embed error, stay pending (§5 rule 6).
		ac.AttemptsLeft--
		f.Error = &domain.FlowError{Code: "invalid_code", Message: "The MFA code is incorrect."}
		// Best-effort save; return the flow state even if the save fails.
		_ = a.db.withTx(ctx, func(ctx context.Context) error {
			return a.flowSave(ctx, row, f)
		})
		return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
	}

	// Success: complete the flow and rotate the token (§5 rule 2).
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
	return &domain.FlowState{FlowToken: newToken, Flow: f, Session: sess}, nil
}
