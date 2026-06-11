package postgres

// Passwordless adapter — OTP codes and magic links over the iam_challenges
// envelope. A challenge stores only the sha256 hash of the opaque code/token
// (never the plaintext), the project boundary, the subject (email/phone) and a
// single-use `consumed` flag. Verify resolves (or provisions) the user behind
// the subject and mints a session.
//
// Crypto: the OTP code and the magic-link token are minted with crypto/rand and
// returned to the caller in plaintext exactly once (via the Challenge); only
// their sha256 hash is persisted in code_hash. The session's access token is a
// signed RS256 JWT (project Signer); the refresh token stays opaque (revocable).

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"regexp"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// pgPasswordlessAccounts persists OTP / magic-link challenges and, on verify,
// resolves-or-creates the user + a fresh session.
type pgPasswordlessAccounts struct {
	db      *DB
	emitter Emitter
	cfg     *configReader
	core    *pgCoreAuth
}

// NewPgPasswordlessAccounts builds the Passwordless adapter over db. cfg is the
// runtime config reader used to enforce registration mode and consent gating on
// auto-provisioned passwordless signups; core mints sessions through the shared
// path so passwordless logins honor session_policy and get a rotatable refresh
// token.
func NewPgPasswordlessAccounts(db *DB, emitter Emitter, cfg *configReader, core *pgCoreAuth) *pgPasswordlessAccounts {
	return &pgPasswordlessAccounts{db: db, emitter: emitter, cfg: cfg, core: core}
}

var _ api.PasswordlessAccounts = (*pgPasswordlessAccounts)(nil)

// ===== lifetimes =====

const (
	otpTTL         = 10 * time.Minute
	magicLinkTTL   = 30 * time.Minute
	otpCodeLen     = 6  // numeric digits
	magicBytes     = 32 // raw entropy for the opaque magic-link token
	maxOTPAttempts = 5  // failed code submissions before the challenge locks
)

// challengeEnvelope is the aggregate stored in the data jsonb column. The
// queryable columns (project_id, type, subject, code_hash, expires_at,
// consumed) mirror it for lookups.
type challengeEnvelope struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Environment string    `json:"environment"` // request env (live | test | …); scopes resolve/create
	Type        string    `json:"type"`        // otp | email (magic link)
	Channel     string    `json:"channel"`     // email | sms
	Purpose     string    `json:"purpose"`     // login | signin | signup | recovery | verify | ...
	Subject     string    `json:"subject"`     // identifier being challenged
	RedirectTo  string    `json:"redirect_to,omitempty"`
	CodeHash    string    `json:"code_hash"`
	ExpiresAt   time.Time `json:"expires_at"`
	Consumed    bool      `json:"consumed"`
	Attempts    int       `json:"attempts"` // failed verify count; caps brute-force on the code
	CreatedAt   time.Time `json:"created_at"`
}

// e164 is the E.164 phone shape the SMS/WhatsApp OTP channels require on the
// free-form OtpStartRequest.identifier (the OAS leaves it an unconstrained
// string; phone-shaped channels validate it here). Mirrors the pattern the
// phone-verification/change bodies enforce: a leading '+', a non-zero first
// digit, then up to 14 more digits.
var e164 = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// isPhoneChannel reports whether the channel resolves a user by phone number
// (sms or whatsapp) rather than by email. An empty channel is email (back-compat).
func isPhoneChannel(channel string) bool {
	return channel == "sms" || channel == "whatsapp"
}

// ===== OTP =====

func (a *pgPasswordlessAccounts) StartOTP(ctx context.Context, projectID, identifier, channel, purpose string) (*domain.Challenge, error) {
	if projectID == "" || identifier == "" {
		return nil, domain.ErrBadRequest
	}
	// Only channels with a working end-to-end delivery path are accepted. An
	// empty channel is email (back-compat). whatsapp has no delivery path yet, so
	// reject it rather than mint a dead challenge the client can never complete.
	switch channel {
	case "", "email", "sms":
	default:
		return nil, domain.ErrBadRequest.WithMessage("unsupported otp channel")
	}
	// Phone-shaped channels resolve the user by phone number, so the free-form
	// identifier MUST be a valid E.164 number (the OAS does not constrain it).
	if isPhoneChannel(channel) && !e164.MatchString(identifier) {
		return nil, domain.ErrBadRequest.WithMessage("identifier must be a valid E.164 phone number for sms")
	}
	// Pre-flight: an SMS OTP is useless without an enabled SMS provider. Fail
	// fast with a clear error instead of committing a challenge whose code can
	// never be delivered (the outbox SMS path fail-soft acks on no provider).
	if channel == "sms" {
		ok, err := providerEnabled(ctx, a.db, projectID, "sms")
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.ErrValidation.WithMessage("an enabled sms provider is required to send sms otp")
		}
	}
	// Scope the challenge (and the user it later resolves/creates) to the request
	// environment, so a phone/email is distinct across live/test/staging.
	reqEnv, err := effectiveEnv(ctx, a.db, projectID, runtimeDefaultEnv)
	if err != nil {
		return nil, err
	}
	code, err := randomNumericCode(otpCodeLen)
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	env := &challengeEnvelope{
		ID:          newUUID(),
		ProjectID:   projectID,
		Environment: reqEnv,
		Type:        "otp",
		Channel:     channel,
		Purpose:     purpose,
		Subject:     identifier,
		CodeHash:    hashToken(code),
		ExpiresAt:   now.Add(otpTTL),
		Consumed:    false,
		CreatedAt:   now,
	}
	// Persist the challenge and enqueue the delivery event atomically: the OTP
	// code must reach the outbox iff the challenge row commits (nested withTx
	// joins the ambient tx via pgtx).
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		if err := a.insertChallenge(ctx, env); err != nil {
			return err
		}
		return a.emitter.Emit(ctx, domain.Event{
			Type:        "auth.otp.started",
			ProjectID:   env.ProjectID,
			Environment: env.Environment,
			AggregateID: env.ID,
			Payload: map[string]any{
				"code":         code,
				"channel":      env.Channel,
				"purpose":      env.Purpose,
				"account_id":   env.Subject,
				"contact":      env.Subject,
				"to":           env.Subject,
				"challenge_id": env.ID,
			},
		})
	}); err != nil {
		return nil, err
	}
	return challengeFromEnvelope(env), nil
}

func (a *pgPasswordlessAccounts) VerifyOTP(ctx context.Context, challengeID, code string) (*domain.Account, *domain.Session, error) {
	// Per-challenge brute-force cap. The mismatch bump is committed in its OWN
	// transaction (not the verify tx, which rolls back on the returned error), so
	// the attempt count actually persists. Once the cap is hit the challenge is
	// consumed (locked) and further tries fail closed.
	env, row, err := a.loadChallengeForVerify(ctx, challengeID, "otp")
	if err != nil {
		return nil, nil, err
	}
	if env.Attempts >= maxOTPAttempts {
		_ = a.consume(ctx, row) // lock out a maxed-out challenge
		return nil, nil, domain.ErrRateLimited
	}
	if !constantTimeMatch(env.CodeHash, hashToken(code)) {
		if berr := a.bumpAttempts(ctx, row); berr != nil {
			return nil, nil, berr
		}
		return nil, nil, domain.ErrInvalidOTP
	}

	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (*verifyResult, error) {
		// Re-load inside the tx: another concurrent verify may have consumed it,
		// and we must consume atomically with provisioning + session mint.
		env, row, err := a.loadChallengeForVerify(ctx, challengeID, "otp")
		if err != nil {
			return nil, err
		}
		if !constantTimeMatch(env.CodeHash, hashToken(code)) {
			return nil, domain.ErrInvalidOTP
		}
		if err := a.consume(ctx, row); err != nil {
			return nil, err
		}
		acct, sess, err := a.resolveAndSession(ctx, env)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "auth.otp.verified",
			ProjectID:   acct.ProjectID,
			Environment: env.envScope(),
			AggregateID: acct.ID,
			Payload:     map[string]any{"account_id": acct.ID, "session_id": sess.ID},
		}); err != nil {
			return nil, err
		}
		return &verifyResult{acct: acct, sess: sess}, nil
	})
	return unpackVerify(res, err)
}

// bumpAttempts increments the challenge's failed-verify counter in its own
// committed transaction, locking (consuming) it once the cap is reached.
func (a *pgPasswordlessAccounts) bumpAttempts(ctx context.Context, row *models.IamChallenge) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		var env challengeEnvelope
		if err := unmarshal(row.Data, &env); err != nil {
			return err
		}
		env.Attempts++
		raw, err := marshal(&env)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamChallengeSetter{Data: &rm}
		if env.Attempts >= maxOTPAttempts {
			setter.Consumed = ptr(true)
		}
		return row.Update(ctx, a.db.Bobx(), setter)
	})
}

// ===== Magic link =====

func (a *pgPasswordlessAccounts) StartMagicLink(ctx context.Context, projectID, email, redirectTo string) (*domain.Challenge, error) {
	if projectID == "" || email == "" {
		return nil, domain.ErrBadRequest
	}
	reqEnv, err := effectiveEnv(ctx, a.db, projectID, runtimeDefaultEnv)
	if err != nil {
		return nil, err
	}
	token, err := randomOpaqueToken(magicBytes)
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	env := &challengeEnvelope{
		ID:          newUUID(),
		ProjectID:   projectID,
		Environment: reqEnv,
		Type:        "email",
		Channel:     "email",
		Purpose:     "login",
		Subject:     email,
		RedirectTo:  redirectTo,
		CodeHash:    hashToken(token),
		ExpiresAt:   now.Add(magicLinkTTL),
		Consumed:    false,
		CreatedAt:   now,
	}
	// Atomic persist + delivery enqueue (nested withTx joins the ambient tx).
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		if err := a.insertChallenge(ctx, env); err != nil {
			return err
		}
		return a.emitter.Emit(ctx, domain.Event{
			Type:        "auth.magiclink.started",
			ProjectID:   env.ProjectID,
			Environment: env.Environment,
			AggregateID: env.ID,
			Payload: map[string]any{
				"token":        token,
				"account_id":   env.Subject,
				"contact":      env.Subject,
				"to":           env.Subject,
				"redirect_to":  env.RedirectTo,
				"purpose":      env.Purpose,
				"challenge_id": env.ID,
			},
		})
	}); err != nil {
		return nil, err
	}
	ch := challengeFromEnvelope(env)
	// Hand the opaque token back to the caller once; the link is built upstream.
	ch.PublicKey = map[string]any{"token": token, "redirect_to": redirectTo}
	return ch, nil
}

func (a *pgPasswordlessAccounts) VerifyMagicLink(ctx context.Context, token string) (*domain.Account, *domain.Session, error) {
	if token == "" {
		return nil, nil, domain.ErrInvalidToken
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (*verifyResult, error) {
		// Magic-link verify carries only the opaque token: look the challenge up
		// by its sha256 hash (the column is indexable and never stores plaintext).
		row, err := a.findUnconsumedByHash(ctx, "email", hashToken(token))
		if err != nil {
			return nil, err
		}
		var env challengeEnvelope
		if err := unmarshal(row.Data, &env); err != nil {
			return nil, err
		}
		if env.Consumed {
			return nil, domain.ErrTokenUsed
		}
		if nowUTC().After(env.ExpiresAt) {
			return nil, domain.ErrChallengeExpired
		}
		if err := a.consume(ctx, row); err != nil {
			return nil, err
		}
		acct, sess, err := a.resolveAndSession(ctx, &env)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "auth.magiclink.verified",
			ProjectID:   acct.ProjectID,
			Environment: env.envScope(),
			AggregateID: acct.ID,
			Payload:     map[string]any{"account_id": acct.ID, "session_id": sess.ID},
		}); err != nil {
			return nil, err
		}
		return &verifyResult{acct: acct, sess: sess}, nil
	})
	return unpackVerify(res, err)
}

// ===== persistence helpers =====

// insertChallenge writes the envelope plus its lookup columns.
func (a *pgPasswordlessAccounts) insertChallenge(ctx context.Context, env *challengeEnvelope) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		raw, err := marshal(env)
		if err != nil {
			return err
		}
		rm := json.RawMessage(raw)
		subject := null.From(env.Subject)
		codeHash := null.From(env.CodeHash)
		environment := env.Environment
		if environment == "" {
			environment = runtimeDefaultEnv
		}
		setter := &models.IamChallengeSetter{
			ID:          &env.ID,
			ProjectID:   &env.ProjectID,
			Environment: &environment,
			Type:        &env.Type,
			Subject:     &subject,
			CodeHash:    &codeHash,
			ExpiresAt:   &env.ExpiresAt,
			Consumed:    ptr(false),
			CreatedAt:   &env.CreatedAt,
			Data:        &rm,
		}
		if _, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return translatePgErr("challenge", err)
		}
		return nil
	})
}

// loadChallengeForVerify fetches a challenge by id, enforces type + single-use +
// expiry, and returns the decoded envelope and the row (for consume).
func (a *pgPasswordlessAccounts) loadChallengeForVerify(ctx context.Context, challengeID, typ string) (*challengeEnvelope, *models.IamChallenge, error) {
	row, err := models.FindIamChallenge(ctx, a.db.Bobx(), challengeID)
	if err != nil {
		if isNoRows(err) {
			return nil, nil, domain.ErrChallengeInvalid
		}
		return nil, nil, err
	}
	if row.Type != typ {
		return nil, nil, domain.ErrChallengeInvalid
	}
	var env challengeEnvelope
	if err := unmarshal(row.Data, &env); err != nil {
		return nil, nil, err
	}
	if env.Consumed || row.Consumed {
		return nil, nil, domain.ErrTokenUsed
	}
	if nowUTC().After(env.ExpiresAt) {
		return nil, nil, domain.ErrChallengeExpired
	}
	return &env, row, nil
}

// findUnconsumedByHash resolves a challenge of the given type by its code_hash
// lookup column (used by magic-link verify, which has no challenge id).
func (a *pgPasswordlessAccounts) findUnconsumedByHash(ctx context.Context, typ, codeHash string) (*models.IamChallenge, error) {
	row, err := models.IamChallenges.Query(
		sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg(typ))),
		sm.Where(models.IamChallenges.Columns.CodeHash.EQ(psql.Arg(codeHash))),
		sm.Where(models.IamChallenges.Columns.Consumed.EQ(psql.Arg(false))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}
	return row, nil
}

// consume flips the single-use flag and mirrors it into the envelope, so a
// replayed code/token fails on the next verify.
func (a *pgPasswordlessAccounts) consume(ctx context.Context, row *models.IamChallenge) error {
	var env challengeEnvelope
	if err := unmarshal(row.Data, &env); err != nil {
		return err
	}
	env.Consumed = true
	raw, err := marshal(&env)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamChallengeSetter{Consumed: ptr(true), Data: &rm}
	if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
		return err
	}
	return nil
}

// resolveAndSession finds the user behind the challenge subject within the
// project+environment (creating one if the subject is fresh and the purpose
// permits signup) and mints a session that records the verification channel.
func (a *pgPasswordlessAccounts) resolveAndSession(ctx context.Context, env *challengeEnvelope) (*domain.Account, *domain.Session, error) {
	acct, err := a.resolveOrCreateUser(ctx, env)
	if err != nil {
		return nil, nil, err
	}
	if acct.Status == "suspended" {
		return nil, nil, domain.ErrAccountSuspended
	}
	if acct.Status == "banned" {
		return nil, nil, domain.ErrAccountBanned
	}
	sess, err := a.createSession(ctx, acct, env.Channel, env.envScope())
	if err != nil {
		return nil, nil, err
	}
	return acct, sess, nil
}

// envScope returns the request environment recorded on the challenge, defaulting
// to the runtime default so older challenges (minted before env was persisted)
// keep resolving against the live data set.
func (env *challengeEnvelope) envScope() string {
	if env.Environment == "" {
		return runtimeDefaultEnv
	}
	return env.Environment
}

// allowsSignup reports whether the challenge purpose permits provisioning a new
// user on a subject miss. Only "verify" must NOT create (it asserts control of
// an already-known identity, so an unknown subject is a not-found). Passwordless
// OTP is otherwise a transparent sign-in-or-sign-up: "signin", "signup",
// "login", "recovery" and the empty/back-compat purpose all provision on a miss.
// This preserves the long-standing email-OTP contract (an unknown email starts a
// passwordless signup) while still letting a caller opt into a strict lookup via
// purpose=verify.
func allowsSignup(purpose string) bool {
	switch purpose {
	case "verify":
		return false
	default:
		return true
	}
}

// resolveOrCreateUser resolves the user behind the challenge subject (by email
// for the email/magic-link channels, by phone for the sms/whatsapp channels),
// scoped to the challenge environment. On a miss it provisions a fresh active
// human user when the purpose permits signup, otherwise returns ErrUserNotFound.
func (a *pgPasswordlessAccounts) resolveOrCreateUser(ctx context.Context, env *challengeEnvelope) (*domain.Account, error) {
	if isPhoneChannel(env.Channel) {
		return a.resolveOrCreateByPhone(ctx, env)
	}
	return a.resolveOrCreateByEmail(ctx, env)
}

// resolveOrCreateByEmail handles the email / magic-link channels.
func (a *pgPasswordlessAccounts) resolveOrCreateByEmail(ctx context.Context, env *challengeEnvelope) (*domain.Account, error) {
	scope := env.envScope()
	row, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(env.ProjectID))),
		sm.Where(models.IamUsers.Columns.Environment.EQ(psql.Arg(scope))),
		sm.Where(models.IamUsers.Columns.PrimaryEmail.EQ(psql.Arg(env.Subject))),
	).One(ctx, a.db.Bobx())
	if err == nil {
		var acc domain.Account
		if uerr := unmarshal(row.Data, &acc); uerr != nil {
			return nil, uerr
		}
		// Magic-link / email-OTP proves control of the address.
		if !acc.EmailVerified {
			acc.EmailVerified = true
			if uerr := a.persistAccount(ctx, &acc); uerr != nil {
				return nil, uerr
			}
		}
		return &acc, nil
	}
	if !isNoRows(err) {
		return nil, err
	}
	if !allowsSignup(env.Purpose) {
		return nil, domain.ErrUserNotFound
	}
	// Registration mode / required consent gate: passwordless auto-signup is only
	// allowed for open registration with no required consents; otherwise the user
	// must register via the flow (which enforces both). Fail-closed.
	if ok, gerr := a.passwordlessSignupAllowed(ctx, env.ProjectID); gerr != nil {
		return nil, gerr
	} else if !ok {
		return nil, domain.ErrUserNotFound
	}

	now := nowUTC()
	acc := &domain.Account{
		ID:            newUUID(),
		ProjectID:     env.ProjectID,
		Kind:          "human",
		Status:        "active",
		PrimaryEmail:  env.Subject,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	email := null.From(acc.PrimaryEmail)
	setter := &models.IamUserSetter{PrimaryEmail: &email}
	if err := a.insertUser(ctx, acc, scope, setter); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrEmailExists
		}
		return nil, err
	}
	return acc, nil
}

// resolveOrCreateByPhone handles the sms / whatsapp channels: lookup/create by
// (project, environment, primary_phone) — the env-scoped unique index.
func (a *pgPasswordlessAccounts) resolveOrCreateByPhone(ctx context.Context, env *challengeEnvelope) (*domain.Account, error) {
	scope := env.envScope()
	row, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(env.ProjectID))),
		sm.Where(models.IamUsers.Columns.Environment.EQ(psql.Arg(scope))),
		sm.Where(models.IamUsers.Columns.PrimaryPhone.EQ(psql.Arg(env.Subject))),
	).One(ctx, a.db.Bobx())
	if err == nil {
		var acc domain.Account
		if uerr := unmarshal(row.Data, &acc); uerr != nil {
			return nil, uerr
		}
		// Phone-OTP proves control of the number.
		if !acc.PhoneVerified {
			acc.PhoneVerified = true
			if uerr := a.persistAccount(ctx, &acc); uerr != nil {
				return nil, uerr
			}
		}
		return &acc, nil
	}
	if !isNoRows(err) {
		return nil, err
	}
	if !allowsSignup(env.Purpose) {
		return nil, domain.ErrUserNotFound
	}
	// Registration mode / required consent gate: passwordless auto-signup is only
	// allowed for open registration with no required consents; otherwise the user
	// must register via the flow (which enforces both). Fail-closed.
	if ok, gerr := a.passwordlessSignupAllowed(ctx, env.ProjectID); gerr != nil {
		return nil, gerr
	} else if !ok {
		return nil, domain.ErrUserNotFound
	}

	now := nowUTC()
	acc := &domain.Account{
		ID:            newUUID(),
		ProjectID:     env.ProjectID,
		Kind:          "human",
		Status:        "active",
		PrimaryPhone:  env.Subject,
		PhoneVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	phone := null.From(acc.PrimaryPhone)
	setter := &models.IamUserSetter{PrimaryPhone: &phone}
	if err := a.insertUser(ctx, acc, scope, setter); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrPhoneExists
		}
		return nil, err
	}
	return acc, nil
}

// insertUser persists a freshly provisioned passwordless user. The caller
// supplies the channel-specific lookup column(s) on setter (primary_email or
// primary_phone); insertUser fills the shared columns + data envelope, scopes
// the row to environment, writes it and emits user.created.
func (a *pgPasswordlessAccounts) insertUser(ctx context.Context, acc *domain.Account, environment string, setter *models.IamUserSetter) error {
	raw, err := marshal(acc)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter.ID = &acc.ID
	setter.ProjectID = &acc.ProjectID
	setter.Environment = &environment
	setter.Kind = ptr(acc.Kind)
	setter.Status = ptr(acc.Status)
	setter.Data = &rm
	if _, err := models.IamUsers.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		return err
	}
	return a.emitter.Emit(ctx, domain.Event{
		Type:        "user.created",
		ProjectID:   acc.ProjectID,
		Environment: environment,
		AggregateID: acc.ID,
		Payload:     acc,
	})
}

// persistAccount writes back a mutated account envelope.
func (a *pgPasswordlessAccounts) persistAccount(ctx context.Context, acc *domain.Account) error {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), acc.ID)
	if err != nil {
		return translatePgErr("user", err)
	}
	raw, err := marshal(acc)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamUserSetter{Data: &rm, UpdatedAt: ptr(nowUTC())}
	if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
		return err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "user.updated",
		ProjectID:   acc.ProjectID,
		Environment: row.Environment,
		AggregateID: acc.ID,
		Payload:     acc,
	}); err != nil {
		return err
	}
	return nil
}

// createSession mints an authenticated session for acct through the shared core
// path, so passwordless logins honor session_policy (access/refresh TTLs,
// refresh horizon), persist a ROTATABLE refresh token in iam_refresh_tokens, and
// record device metadata — exactly like password/OAuth logins. AMR records "otp"
// plus the verification channel (sms) so downstream MFA/AAL policy can tell a
// phone login from an email one. The session is bound to the challenge's
// environment by overriding it on the context the core minter reads.
func (a *pgPasswordlessAccounts) createSession(ctx context.Context, acct *domain.Account, channel, env string) (*domain.Session, error) {
	amr := []string{"otp"}
	if isPhoneChannel(channel) {
		amr = append(amr, "sms")
	}
	if env != "" {
		ctx = api.WithEnvironment(ctx, env)
	}
	return a.core.coreAuthMintSession(ctx, acct, "", amr, 1)
}

// ===== local helpers (aggregate-prefixed) =====

// verifyResult bundles the (account, session) pair so it can flow through the
// single generic return of withTxRet.
type verifyResult struct {
	acct *domain.Account
	sess *domain.Session
}

// unpackVerify adapts (*verifyResult, error) into the port's triple return.
func unpackVerify(r *verifyResult, err error) (*domain.Account, *domain.Session, error) {
	if err != nil {
		return nil, nil, err
	}
	return r.acct, r.sess, nil
}

// hashToken returns the hex sha256 of an opaque token/code; only this is stored.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// constantTimeMatch compares two hex hashes without leaking timing.
func constantTimeMatch(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// randomNumericCode mints an n-digit numeric OTP from crypto/rand.
func randomNumericCode(n int) (string, error) {
	const digits = "0123456789"
	buf := make([]byte, n)
	for i := range buf {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		buf[i] = digits[idx.Int64()]
	}
	return string(buf), nil
}

// randomOpaqueToken mints a hex opaque token from nbytes of crypto/rand entropy.
func randomOpaqueToken(nbytes int) (string, error) {
	buf := make([]byte, nbytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// challengeFromEnvelope maps the stored envelope into the domain Challenge the
// API returns (never carrying the plaintext code/token by default).
func challengeFromEnvelope(env *challengeEnvelope) *domain.Challenge {
	return &domain.Challenge{
		ID:        env.ID,
		Type:      env.Type,
		ExpiresAt: env.ExpiresAt,
	}
}

// providerEnabled reports whether the project has at least one enabled provider
// of the given kind (e.g. "sms", "email"). Used for delivery pre-flight checks.
func providerEnabled(ctx context.Context, db *DB, projectID, kind string) (bool, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg(kind))),
		sm.Where(models.IamProviders.Columns.Enabled.EQ(psql.Arg(true))),
	).All(ctx, db.Bobx())
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

// passwordlessSignupAllowed reports whether a fresh user may be auto-provisioned
// on a passwordless (OTP/magic-link) subject miss. It is fail-CLOSED: a config
// read error blocks provisioning. Auto-signup is permitted only when the project
// registration mode is open (or unset) AND no required consent documents are
// configured — otherwise the user must register through the resumable flow,
// which enforces the mode (invite_only/request_access/closed) and presents the
// consent step. This closes the passwordless bypass of both gates.
func (a *pgPasswordlessAccounts) passwordlessSignupAllowed(ctx context.Context, projectID string) (bool, error) {
	if a.cfg == nil {
		return true, nil
	}
	auth, err := a.cfg.AuthConfig(ctx, projectID)
	if err != nil {
		return false, err
	}
	if auth.RegistrationMode != "" && auth.RegistrationMode != "open" {
		return false, nil
	}
	consents, err := a.cfg.ConsentConfig(ctx, projectID)
	if err != nil {
		return false, err
	}
	for _, d := range consents {
		if d.Required != nil && *d.Required {
			return false, nil
		}
	}
	return true, nil
}

// isNoRows reports whether err is a bob/pgx no-rows result.
func isNoRows(err error) bool {
	return translatePgErr("", err) != err
}
