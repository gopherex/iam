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
}

// NewPgPasswordlessAccounts builds the Passwordless adapter over db.
func NewPgPasswordlessAccounts(db *DB, emitter Emitter) *pgPasswordlessAccounts {
	return &pgPasswordlessAccounts{db: db, emitter: emitter}
}

var _ api.PasswordlessAccounts = (*pgPasswordlessAccounts)(nil)

// ===== lifetimes =====

const (
	otpTTL       = 10 * time.Minute
	magicLinkTTL = 30 * time.Minute
	otpCodeLen   = 6  // numeric digits
	magicBytes   = 32 // raw entropy for the opaque magic-link token
)

// challengeEnvelope is the aggregate stored in the data jsonb column. The
// queryable columns (project_id, type, subject, code_hash, expires_at,
// consumed) mirror it for lookups.
type challengeEnvelope struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Type       string    `json:"type"`    // otp | email (magic link)
	Channel    string    `json:"channel"` // email | sms
	Purpose    string    `json:"purpose"` // login | signup | recovery | ...
	Subject    string    `json:"subject"` // identifier being challenged
	RedirectTo string    `json:"redirect_to,omitempty"`
	CodeHash   string    `json:"code_hash"`
	ExpiresAt  time.Time `json:"expires_at"`
	Consumed   bool      `json:"consumed"`
	CreatedAt  time.Time `json:"created_at"`
}

// ===== OTP =====

func (a *pgPasswordlessAccounts) StartOTP(ctx context.Context, projectID, identifier, channel, purpose string) (*domain.Challenge, error) {
	if projectID == "" || identifier == "" {
		return nil, domain.ErrBadRequest
	}
	code, err := randomNumericCode(otpCodeLen)
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	env := &challengeEnvelope{
		ID:        newUUID(),
		ProjectID: projectID,
		Type:      "otp",
		Channel:   channel,
		Purpose:   purpose,
		Subject:   identifier,
		CodeHash:  hashToken(code),
		ExpiresAt: now.Add(otpTTL),
		Consumed:  false,
		CreatedAt: now,
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
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (*verifyResult, error) {
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
			Environment: "",
			AggregateID: acct.ID,
			Payload:     map[string]any{"account_id": acct.ID, "session_id": sess.ID},
		}); err != nil {
			return nil, err
		}
		return &verifyResult{acct: acct, sess: sess}, nil
	})
	return unpackVerify(res, err)
}

// ===== Magic link =====

func (a *pgPasswordlessAccounts) StartMagicLink(ctx context.Context, projectID, email, redirectTo string) (*domain.Challenge, error) {
	if projectID == "" || email == "" {
		return nil, domain.ErrBadRequest
	}
	token, err := randomOpaqueToken(magicBytes)
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	env := &challengeEnvelope{
		ID:         newUUID(),
		ProjectID:  projectID,
		Type:       "email",
		Channel:    "email",
		Purpose:    "login",
		Subject:    email,
		RedirectTo: redirectTo,
		CodeHash:   hashToken(token),
		ExpiresAt:  now.Add(magicLinkTTL),
		Consumed:   false,
		CreatedAt:  now,
	}
	// Atomic persist + delivery enqueue (nested withTx joins the ambient tx).
	if err := a.db.withTx(ctx, func(ctx context.Context) error {
		if err := a.insertChallenge(ctx, env); err != nil {
			return err
		}
		return a.emitter.Emit(ctx, domain.Event{
			Type:        "auth.magiclink.started",
			ProjectID:   env.ProjectID,
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
			Environment: "",
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
		setter := &models.IamChallengeSetter{
			ID:        &env.ID,
			ProjectID: &env.ProjectID,
			Type:      &env.Type,
			Subject:   &subject,
			CodeHash:  &codeHash,
			ExpiresAt: &env.ExpiresAt,
			Consumed:  ptr(false),
			CreatedAt: &env.CreatedAt,
			Data:      &rm,
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
// project (creating one if the subject is a fresh email) and mints a session.
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
	sess, err := a.createSession(ctx, acct)
	if err != nil {
		return nil, nil, err
	}
	return acct, sess, nil
}

// resolveOrCreateUser looks the subject up by the email lookup column; an
// unknown email is provisioned as a new active human user (passwordless signup).
func (a *pgPasswordlessAccounts) resolveOrCreateUser(ctx context.Context, env *challengeEnvelope) (*domain.Account, error) {
	row, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(env.ProjectID))),
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
	raw, err := marshal(acc)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	email := null.From(acc.PrimaryEmail)
	setter := &models.IamUserSetter{
		ID:           &acc.ID,
		ProjectID:    &acc.ProjectID,
		Kind:         ptr(acc.Kind),
		Status:       ptr(acc.Status),
		PrimaryEmail: &email,
		Data:         &rm,
	}
	if _, err := models.IamUsers.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrEmailExists
		}
		return nil, err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "user.created",
		ProjectID:   acc.ProjectID,
		Environment: "",
		AggregateID: acc.ID,
		Payload:     acc,
	}); err != nil {
		return nil, err
	}
	return acc, nil
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
		Environment: "",
		AggregateID: acc.ID,
		Payload:     acc,
	}); err != nil {
		return err
	}
	return nil
}

// createSession mints an authenticated session for acct. The access token is a
// signed RS256 JWT (jwx Signer); the refresh token stays opaque (revocable).
func (a *pgPasswordlessAccounts) createSession(ctx context.Context, acct *domain.Account) (*domain.Session, error) {
	now := nowUTC()
	const expiresIn = 3600
	sessionID := newUUID()
	signEnv, err := resolveSignEnv(ctx, a.db, acct.ProjectID, "live")
	if err != nil {
		return nil, err
	}
	access, err := a.db.Signer().Sign(ctx, acct.ProjectID, signEnv, map[string]any{
		"iss": acct.ProjectID,
		"sub": acct.ID,
		"sid": sessionID,
		"pid": acct.ProjectID,
		"aal": 1,
		"amr": []string{"otp"},
		"typ": "access",
		"env": signEnv,
	}, time.Duration(expiresIn)*time.Second)
	if err != nil {
		return nil, err
	}
	refresh, err := randomOpaqueToken(32)
	if err != nil {
		return nil, err
	}
	sess := &domain.Session{
		ID:           sessionID,
		AccountID:    acct.ID,
		ProjectID:    acct.ProjectID,
		AMR:          []string{"otp"},
		AAL:          1,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    expiresIn,
		CreatedAt:    now,
	}
	raw, err := marshal(sess)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	expiresAt := null.From(now.Add(time.Duration(expiresIn) * time.Second))
	setter := &models.IamSessionSetter{
		ID:           &sess.ID,
		ProjectID:    &sess.ProjectID,
		UserID:       &sess.AccountID,
		Aal:          ptr(int32(sess.AAL)),
		Trusted:      ptr(false),
		ExpiresAt:    &expiresAt,
		CreatedAt:    &now,
		LastActiveAt: &now,
		Data:         &rm,
	}
	if _, err := models.IamSessions.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		return nil, translatePgErr("session", err)
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "auth.session.created",
		ProjectID:   sess.ProjectID,
		Environment: "",
		AggregateID: sess.ID,
		Payload:     sess,
	}); err != nil {
		return nil, err
	}
	return sess, nil
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

// isNoRows reports whether err is a bob/pgx no-rows result.
func isNoRows(err error) bool {
	return translatePgErr("", err) != err
}
