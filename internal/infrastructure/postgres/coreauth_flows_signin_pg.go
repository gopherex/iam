package postgres

// coreauth_flows_signin_pg.go — SIGNIN kind of the server-side resumable auth
// flow engine (§7 signin machine, §5 security). Multichannel: a `method`
// selects the primary factor at create time; the flow wraps the already-built
// single-method adapters (password, phone-OTP, magic-link). The state machine
// also lets the client switch to an alternate method mid-flow (switch_method).
//
// Methods:
//
//	password (default)
//	  create{email,password} → AuthenticatePassword (wrong → ErrInvalidCredentials,
//	    anti-enumeration §5.4)
//	  → !MFARequired → completed (session returned)
//	  → MFARequired  → mfa_required (challenge issued, no session yet)
//	  submit{mfa,code} → mfa.Verify → completed (token rotated, session)
//
//	phone_otp
//	  create{phone}  → preflight enabled SMS provider → StartOTP(sms, signin)
//	    → verify_phone (no session yet)
//	  submit{verify_otp,code} → VerifyOTP → completed (token rotated, session)
//
//	magic_link
//	  create{email}  → StartMagicLink(login) (link delivered out-of-band)
//	    → verify_email (no session yet)
//	  submit{verify_email,token} → VerifyMagicLink → completed (token rotated, session)
//
//	switch_method (any pending signin step)
//	  submit{switch_method,method[,phone]} → re-issues the requested method's
//	    challenge in place, overwriting ActiveChallenge (flowSave, no rotation).
//
// Security mapping (§5):
//   - §5.1 token: flowMintToken (≥256-bit, ftk_ prefix, only hash stored).
//   - §5.2 rotation: flowRotate on the session-minting step (mfa/verify_otp/
//     verify_email-magic-link). switch_method and wrong-code use flowSave (no rotation).
//   - §5.3 tenant+TTL: enforced by flowLoad in the caller (Submit) path.
//   - §5.4 anti-enumeration: password propagates ErrInvalidCredentials; phone_otp
//     and magic_link mint challenges uniformly and resolve the user only on verify.
//   - §5.5 no raw secrets: password never stored in data; has_password=true only.
//   - §5.6 attempts: AttemptsLeft tracked in ActiveChallenge, decremented on wrong code.
//   - §5.8 session once: session returned only when status=completed.
//
// MFA layering: password gates MFA before completing (parity with the legacy
// single-channel signin). phone_otp and magic_link are passwordless first-factor
// logins that the underlying adapters mint at AAL1 directly — consistent with the
// standalone passwordless endpoints — so they complete without a second factor.

import (
	"context"
	"errors"
	"fmt"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

func init() {
	flowCreators[domain.FlowKindSignin] = createSignin
	flowAdvancers[domain.FlowKindSignin] = advanceSignin
}

// signinPasswordless builds a passwordless adapter sharing the flow adapter's db,
// emitter and config, with the core auth adapter as the session minter. Mirrors
// the in-place construction of NewPgMFAAccounts used by signinVerifyMFA.
func (a *pgCoreAuthFlows) signinPasswordless() (*pgPasswordlessAccounts, error) {
	core, ok := a.accounts.(*pgCoreAuth)
	if !ok {
		return nil, fmt.Errorf("signin flow: accounts adapter is not *pgCoreAuth")
	}
	return NewPgPasswordlessAccounts(a.db, a.emitter, a.cfg, core), nil
}

// ─── create ──────────────────────────────────────────────────────────────────

// createSignin is the flowCreateFn for SIGNIN. It dispatches on cmd.Method.
func createSignin(ctx context.Context, a *pgCoreAuthFlows, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	method := cmd.Method
	if method == "" {
		method = domain.FlowMethodPassword
	}
	f.Method = method
	switch method {
	case domain.FlowMethodPassword:
		return a.createSigninPassword(ctx, f, cmd)
	case domain.FlowMethodPhoneOTP:
		return a.createSigninPhoneOTP(ctx, f, cmd)
	case domain.FlowMethodMagicLink:
		return a.createSigninMagicLink(ctx, f, cmd)
	default:
		return nil, domain.ErrBadRequest.WithMessage(fmt.Sprintf("unsupported signin method %q", method))
	}
}

// createSigninPassword authenticates the password credential; if MFA is required
// it issues a challenge and persists at mfa_required, otherwise it completes the
// flow in a single round-trip.
func (a *pgCoreAuthFlows) createSigninPassword(ctx context.Context, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
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
		f.AvailableMethods = a.signinAvailableMethods(ctx, result.Account, result.Factors)

		token, hash, err := flowMintToken()
		if err != nil {
			return nil, fmt.Errorf("flow signin create (no-mfa): mint token: %w", err)
		}
		if err := a.flowInsert(ctx, f, hash, flowData{
			Contact:          f.Contact,
			Collected:        f.Collected,
			Method:           f.Method,
			AvailableMethods: f.AvailableMethods,
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
	f.AvailableMethods = a.signinAvailableMethods(ctx, result.Account, result.Factors)
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
		Contact:          f.Contact,
		Collected:        f.Collected,
		ActiveChallenge:  f.ActiveChallenge,
		Method:           f.Method,
		AvailableMethods: f.AvailableMethods,
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// createSigninPhoneOTP issues an SMS OTP challenge (purpose=signin) and persists
// the flow at verify_phone. The SMS provider is preflighted inside StartOTP, so
// a missing provider fails fast with ErrValidation before any flow row is written.
func (a *pgCoreAuthFlows) createSigninPhoneOTP(ctx context.Context, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	if cmd.Phone == "" {
		return nil, domain.ErrBadRequest.WithMessage("phone is required for phone_otp signin")
	}
	pl, err := a.signinPasswordless()
	if err != nil {
		return nil, err
	}
	ch, err := pl.StartOTP(ctx, f.ProjectID, cmd.Phone, "sms", "signin")
	if err != nil {
		return nil, err // ErrValidation (no provider) / ErrBadRequest (bad E.164)
	}
	return a.signinPersistChallenge(ctx, f, domain.FlowStepVerifyPhone, "sms", ch)
}

// createSigninMagicLink issues a magic-link challenge (purpose=login). The link
// is delivered out-of-band; the flow halts at verify_email until the client
// submits the token carried by the link.
func (a *pgCoreAuthFlows) createSigninMagicLink(ctx context.Context, f *domain.Flow, cmd domain.FlowCreateCmd) (*domain.FlowState, error) {
	if cmd.Email == "" {
		return nil, domain.ErrBadRequest.WithMessage("email is required for magic_link signin")
	}
	pl, err := a.signinPasswordless()
	if err != nil {
		return nil, err
	}
	ch, err := pl.StartMagicLink(ctx, f.ProjectID, cmd.Email, cmd.RedirectTo)
	if err != nil {
		return nil, err
	}
	return a.signinPersistChallenge(ctx, f, domain.FlowStepVerifyEmail, "email", ch)
}

// signinPersistChallenge embeds a passwordless challenge into the flow's
// ActiveChallenge and persists a fresh pending row at the given step.
func (a *pgCoreAuthFlows) signinPersistChallenge(ctx context.Context, f *domain.Flow, step domain.FlowStep, channel string, ch *domain.Challenge) (*domain.FlowState, error) {
	now := nowUTC()
	f.Step = step
	f.ActiveChallenge = &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      channel,
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
	}
	token, hash, err := flowMintToken()
	if err != nil {
		return nil, fmt.Errorf("flow signin create (%s): mint token: %w", f.Method, err)
	}
	if err := a.flowInsert(ctx, f, hash, flowData{
		Contact:          f.Contact,
		Collected:        f.Collected,
		ActiveChallenge:  f.ActiveChallenge,
		Method:           f.Method,
		AvailableMethods: f.AvailableMethods,
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: token, Flow: f}, nil
}

// signinAvailableMethods computes the alternate methods the client may switch to
// for a resolved account. Only called from the password-create paths, so the
// active method is always password and is excluded from the result. magic_link
// is offered when an email is on file; phone_otp when a phone is on file.
// Best-effort; a nil account yields nil.
func (a *pgCoreAuthFlows) signinAvailableMethods(_ context.Context, acc *domain.Account, _ []domain.Factor) []string {
	if acc == nil {
		return nil
	}
	var out []string
	if acc.PrimaryEmail != "" {
		out = append(out, domain.FlowMethodMagicLink)
	}
	if acc.PrimaryPhone != "" {
		out = append(out, domain.FlowMethodPhoneOTP)
	}
	return out
}

// ─── advance ─────────────────────────────────────────────────────────────────

// advanceSignin is the flowAdvanceFn for SIGNIN.
func advanceSignin(ctx context.Context, a *pgCoreAuthFlows, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	// switch_method is valid at any pending signin step: re-issue an alternate
	// method's challenge in place (no rotation).
	if cmd.Action == "switch_method" {
		return a.signinSwitchMethod(ctx, row, f, cmd)
	}
	switch f.Step {
	case domain.FlowStepMFARequired:
		if cmd.Action != "mfa" {
			return nil, domain.ErrBadRequest.WithMessage(`expected action "mfa" at step mfa_required`)
		}
		return a.signinVerifyMFA(ctx, row, f, cmd)
	case domain.FlowStepVerifyPhone:
		if cmd.Action != "verify_otp" {
			return nil, domain.ErrBadRequest.WithMessage(`expected action "verify_otp" at step verify_phone`)
		}
		return a.signinVerifyOTP(ctx, row, f, cmd)
	case domain.FlowStepVerifyEmail:
		if cmd.Action != "verify_email" {
			return nil, domain.ErrBadRequest.WithMessage(`expected action "verify_email" at step verify_email`)
		}
		return a.signinVerifyMagicLink(ctx, row, f, cmd)
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
		return a.signinWrongCode(ctx, row, f, cmd, "The MFA code is incorrect.")
	}

	return a.signinCompleteWithSession(ctx, row, f, sess)
}

// signinVerifyOTP verifies the SMS OTP via the passwordless adapter. On success
// it completes the flow with the minted session and rotates the token.
func (a *pgCoreAuthFlows) signinVerifyOTP(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	ac := f.ActiveChallenge
	if ac == nil {
		return nil, domain.ErrBadRequest.WithMessage("no active otp challenge")
	}
	code := cmd.Payload["code"]
	if code == "" {
		return nil, domain.ErrBadRequest.WithMessage("code is required")
	}
	if ac.AttemptsLeft <= 0 {
		return nil, domain.ErrChallengeInvalid.WithMessage("challenge exhausted; please resend")
	}
	pl, err := a.signinPasswordless()
	if err != nil {
		return nil, err
	}
	_, sess, err := pl.VerifyOTP(ctx, ac.ChallengeID, code)
	if err != nil {
		// ErrRateLimited (challenge locked) is surfaced as-is so the client backs off.
		if errors.Is(err, domain.ErrRateLimited) {
			return nil, err
		}
		return a.signinWrongCode(ctx, row, f, cmd, "The verification code is incorrect.")
	}
	return a.signinCompleteWithSession(ctx, row, f, sess)
}

// signinVerifyMagicLink consumes the opaque magic-link token submitted by the
// client and, on success, completes the flow with the minted session.
func (a *pgCoreAuthFlows) signinVerifyMagicLink(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	token := cmd.Payload["token"]
	if token == "" {
		return nil, domain.ErrBadRequest.WithMessage("token is required")
	}
	pl, err := a.signinPasswordless()
	if err != nil {
		return nil, err
	}
	_, sess, err := pl.VerifyMagicLink(ctx, token)
	if err != nil {
		return a.signinWrongCode(ctx, row, f, cmd, "The magic link is invalid or expired.")
	}
	return a.signinCompleteWithSession(ctx, row, f, sess)
}

// signinSwitchMethod re-issues a different method's challenge in place: it
// overwrites ActiveChallenge and the active method, then persists with flowSave
// (no rotation — switching does not grant privilege). The requested method must
// be one of the flow's AvailableMethods (or the empty default password).
func (a *pgCoreAuthFlows) signinSwitchMethod(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd) (*domain.FlowState, error) {
	method := cmd.Payload["method"]
	if method == "" {
		return nil, domain.ErrBadRequest.WithMessage("method is required for switch_method")
	}
	if method == f.Method {
		return nil, domain.ErrBadRequest.WithMessage("already using that method")
	}
	switch method {
	case domain.FlowMethodPhoneOTP:
		phone := cmd.Payload["phone"]
		if phone == "" {
			phone = f.Contact.Phone
		}
		if phone == "" {
			return nil, domain.ErrBadRequest.WithMessage("phone is required to switch to phone_otp")
		}
		pl, err := a.signinPasswordless()
		if err != nil {
			return nil, err
		}
		ch, err := pl.StartOTP(ctx, f.ProjectID, phone, "sms", "signin")
		if err != nil {
			return nil, err
		}
		f.Contact.Phone = phone
		return a.signinSwitchPersist(ctx, row, f, cmd.FlowToken, method, domain.FlowStepVerifyPhone, "sms", ch)
	case domain.FlowMethodMagicLink:
		if f.Contact.Email == "" {
			return nil, domain.ErrBadRequest.WithMessage("email is required to switch to magic_link")
		}
		pl, err := a.signinPasswordless()
		if err != nil {
			return nil, err
		}
		ch, err := pl.StartMagicLink(ctx, f.ProjectID, f.Contact.Email, cmd.Payload["redirect_to"])
		if err != nil {
			return nil, err
		}
		return a.signinSwitchPersist(ctx, row, f, cmd.FlowToken, method, domain.FlowStepVerifyEmail, "email", ch)
	default:
		return nil, domain.ErrBadRequest.WithMessage(fmt.Sprintf("cannot switch to method %q", method))
	}
}

// signinSwitchPersist updates the flow's method, step and ActiveChallenge in
// place (no token rotation, §5 rule 2 — switching grants no privilege) and
// returns the still-valid flow token unchanged.
func (a *pgCoreAuthFlows) signinSwitchPersist(ctx context.Context, row *models.IamFlow, f *domain.Flow, flowToken, method string, step domain.FlowStep, channel string, ch *domain.Challenge) (*domain.FlowState, error) {
	now := nowUTC()
	f.Method = method
	f.Step = step
	f.Error = nil
	f.ActiveChallenge = &domain.FlowActiveChallenge{
		ChallengeID:  ch.ID,
		Channel:      channel,
		ExpiresAt:    ch.ExpiresAt,
		ResendAt:     now.Add(flowResendCooloff),
		AttemptsLeft: flowMaxAttempts,
	}
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		return a.flowSave(ctx, row, f)
	}); err != nil {
		return nil, err
	}
	return &domain.FlowState{FlowToken: flowToken, Flow: f}, nil
}

// signinWrongCode decrements AttemptsLeft, embeds error{invalid_code}, and
// returns the pending FlowState without a session (mirrors signupVerifyEmail).
func (a *pgCoreAuthFlows) signinWrongCode(ctx context.Context, row *models.IamFlow, f *domain.Flow, cmd domain.FlowSubmitCmd, msg string) (*domain.FlowState, error) {
	if f.ActiveChallenge != nil {
		f.ActiveChallenge.AttemptsLeft--
	}
	f.Error = &domain.FlowError{Code: "invalid_code", Message: msg}
	_ = a.db.withTx(ctx, func(ctx context.Context) error {
		return a.flowSave(ctx, row, f)
	})
	return &domain.FlowState{FlowToken: cmd.FlowToken, Flow: f}, nil
}

// signinCompleteWithSession completes the flow, rotates the token (§5 rule 2 —
// session-minting step) and surfaces the session.
func (a *pgCoreAuthFlows) signinCompleteWithSession(ctx context.Context, row *models.IamFlow, f *domain.Flow, sess *domain.Session) (*domain.FlowState, error) {
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
