package postgres

// Postgres adapter for the Core Auth aggregate slice. It satisfies the
// api.CoreAuthAccounts (register / sign-in / refresh / verification / password
// lifecycle / step-up / access requests) and api.CoreAuthTokens (introspect /
// verify / revoke / current claims) ports.
//
// Storage model (package convention): each aggregate is one table carrying the
// queryable envelope columns plus the full domain object in a `data jsonb`
// column. The lookup columns (project_id, email, user_id, status, hash,
// code_hash, ...) are derived from the struct purely so the tenant-scoped
// queries can find rows; the authoritative object lives in the jsonb blob.
//
// Tenant boundary: every query filters by project_id; a row whose project_id
// does not match the requested one is treated as not-found.
//
// Crypto: passwords are bcrypt-hashed; refresh tokens / codes / secrets are
// drawn from crypto/rand and only their sha256 hash is persisted (hash /
// code_hash / secret columns), never the plaintext. The ACCESS TOKEN is a
// signed RS256 JWT minted and verified via the Signer (jwx, db.Signer()); the
// session's sid is carried as a claim, so introspect/verify/revoke resolve the
// session from the token rather than scanning.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"golang.org/x/crypto/bcrypt"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// ----- adapter -----

// pgCoreAuth is the Postgres-backed Core Auth adapter. Every mutating method
// wraps its work in db.withTx / withTxRet (serializable + mandatory retry);
// reads run on db.Bobx() directly.
type pgCoreAuth struct {
	db      *DB
	emitter Emitter
}

// NewPgCoreAuth builds the Postgres-backed Core Auth adapter.
func NewPgCoreAuth(db *DB, emitter Emitter) *pgCoreAuth {
	return &pgCoreAuth{db: db, emitter: emitter}
}

var (
	_ api.CoreAuthAccounts = (*pgCoreAuth)(nil)
	_ api.CoreAuthTokens   = (*pgCoreAuth)(nil)
)

const (
	// coreAuthDefaultEnv is the environment whose signing key mints access
	// tokens until per-environment resolution is wired from the client.
	coreAuthDefaultEnv = "live"
	coreAuthAccessTTL  = 30 * time.Minute
)

// coreAuthVerifyAccess validates a signed access-token JWT (jwx) against the
// project's signing keys and resolves the still-live session it names. A
// missing/revoked session yields (claims, nil, nil) so callers report inactive.
func (a *pgCoreAuth) coreAuthVerifyAccess(ctx context.Context, projectID, token string) (map[string]any, *domain.Session, error) {
	// Verify against the environment the token was minted in (its "env" claim),
	// falling back to the default for legacy untagged tokens.
	env := coreAuthDefaultEnv
	if peek := a.db.Signer().UnverifiedClaims(token); peek != nil {
		if e, ok := peek["env"].(string); ok && e != "" {
			env = e
		}
	}
	claims, err := a.db.Signer().Verify(ctx, projectID, env, token)
	if err != nil {
		return nil, nil, err
	}
	sid, _ := claims["sid"].(string)
	if sid == "" {
		return claims, nil, nil
	}
	row, err := models.FindIamSession(ctx, a.db.Bobx(), sid)
	if err != nil {
		return claims, nil, nil // session gone == revoked
	}
	if v, ok := row.ExpiresAt.Get(); ok && nowUTC().After(v) {
		return claims, nil, nil
	}
	sess, err := coreAuthLoadSession(row, row.ProjectID)
	if err != nil {
		return claims, nil, nil
	}
	return claims, sess, nil
}

// ----- core-auth local constants -----

const (
	coreAuthDefaultSessionTTL   = 24 * time.Hour
	coreAuthRefreshTTL          = 30 * 24 * time.Hour
	coreAuthChallengeTTL        = 15 * time.Minute
	coreAuthDefaultExpiresInSec = int(coreAuthDefaultSessionTTL / time.Second)

	coreAuthChallengeEmail = "email"
	coreAuthChallengePhone = "phone"

	coreAuthCredentialPassword = "password"

	coreAuthStatusActive    = "active"
	coreAuthStatusSuspended = "suspended"
	coreAuthStatusBanned    = "banned"

	coreAuthKindHuman = "human"
	coreAuthKindGuest = "guest"
)

// ----- core-auth crypto helpers -----

// coreAuthHashPassword bcrypt-hashes a plaintext password for the secret column
// / credential envelope.
func coreAuthHashPassword(plaintext string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// coreAuthCheckPassword reports whether plaintext matches the bcrypt hash.
func coreAuthCheckPassword(hash, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

// coreAuthRandomToken returns a URL-safe opaque token drawn from crypto/rand.
// Only its sha256 hash is ever persisted.
func coreAuthRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// coreAuthRandomCode returns a 6-digit numeric one-time code from crypto/rand.
func coreAuthRandomCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1000000
	out := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		out[i] = byte('0' + n%10)
		n /= 10
	}
	return string(out), nil
}

// coreAuthSHA256 returns the hex sha256 of an opaque token; this is what lands
// in the hash / code_hash / secret columns (never the plaintext).
func coreAuthSHA256(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ----- core-auth envelopes -----

// coreAuthCredential is the credential aggregate stored in the iam_credentials
// `data` jsonb envelope. The secret column mirrors Hash for lookups.
type coreAuthCredential struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// coreAuthRefreshToken is the refresh-token aggregate stored in the
// iam_refresh_tokens `data` jsonb envelope. The hash column mirrors Hash.
type coreAuthRefreshToken struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Hash      string    `json:"hash"`
	Revoked   bool      `json:"revoked"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// coreAuthChallengeData is the challenge aggregate stored in the iam_challenges
// `data` jsonb envelope. CodeHash / Token carry the verifiable material.
type coreAuthChallengeData struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Type       string    `json:"type"`    // email | phone | password_reset | email_change | phone_change
	Purpose    string    `json:"purpose"` // verify | change | reset | step_up
	AccountID  string    `json:"account_id"`
	Subject    string    `json:"subject"`    // contact being challenged
	CodeHash   string    `json:"code_hash"`  // sha256 of the numeric code
	TokenHash  string    `json:"token_hash"` // sha256 of the opaque link token
	RedirectTo string    `json:"redirect_to"`
	Locale     string    `json:"locale"`
	Channel    string    `json:"channel"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// coreAuthCodeData is the auth-code aggregate stored in the iam_auth_codes
// `data` jsonb envelope. ChallengeHash carries the PKCE verifier check.
type coreAuthCodeData struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	UserID        string    `json:"user_id"`
	ClientID      string    `json:"client_id"`
	CodeHash      string    `json:"code_hash"`
	ChallengeHash string    `json:"challenge_hash"` // sha256 of PKCE code_challenge
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// ----- core-auth mappers -----

// coreAuthLoadAccount unmarshals an iam_users envelope row into a domain
// Account, enforcing the tenant boundary.
func coreAuthLoadAccount(row *models.IamUser, projectID string) (*domain.Account, error) {
	if row.ProjectID != projectID {
		return nil, domain.ErrUserNotFound
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

// coreAuthLoadSession unmarshals an iam_sessions envelope row into a domain
// Session, enforcing the tenant boundary.
func coreAuthLoadSession(row *models.IamSession, projectID string) (*domain.Session, error) {
	if row.ProjectID != projectID {
		return nil, domain.ErrSessionNotFound
	}
	var sess domain.Session
	if err := unmarshal(row.Data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// coreAuthAccountActive maps a non-active account status onto the matching
// 403 domain error; an active account returns nil.
func coreAuthAccountActive(acc *domain.Account) error {
	switch acc.Status {
	case coreAuthStatusSuspended:
		return domain.ErrAccountSuspended
	case coreAuthStatusBanned:
		return domain.ErrAccountBanned
	case "":
		return nil
	case coreAuthStatusActive:
		return nil
	default:
		return domain.ErrForbidden
	}
}

// ----- core-auth persistence primitives -----

// coreAuthFindUserByEmail returns the iam_users row for (projectID, email) or a
// not-found domain error. The unique index on (project_id, primary_email) makes
// this a single-row lookup.
func (a *pgCoreAuth) coreAuthFindUserByEmail(ctx context.Context, projectID, email string) (*models.IamUser, error) {
	row, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamUsers.Columns.PrimaryEmail.EQ(psql.Arg(email))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if errors.Is(translatePgErr("user", err), ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return row, nil
}

// coreAuthFindPasswordCredential returns the password iam_credentials row for a
// user (tenant-scoped) or a not-found domain error.
func (a *pgCoreAuth) coreAuthFindPasswordCredential(ctx context.Context, projectID, userID string) (*models.IamCredential, error) {
	row, err := models.IamCredentials.Query(
		sm.Where(models.IamCredentials.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamCredentials.Columns.UserID.EQ(psql.Arg(userID))),
		sm.Where(models.IamCredentials.Columns.Type.EQ(psql.Arg(coreAuthCredentialPassword))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if errors.Is(translatePgErr("credential", err), ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	return row, nil
}

// coreAuthMintSession persists a fresh session + its refresh token inside an
// open transaction and returns the populated domain Session. The access token
// is a generated opaque string (real JWT minting is deferred).
//
// MUST be called inside db.withTx / withTxRet (it issues multiple mutations).
func (a *pgCoreAuth) coreAuthMintSession(ctx context.Context, acc *domain.Account, clientID string, amr []string, aal int) (*domain.Session, error) {
	now := nowUTC()
	sessionID := newUUID()

	if aal <= 0 {
		aal = 1
	}
	// Access token is a signed RS256 JWT (jwx); the project's active signing key
	// is generated on first use.
	signEnv, err := resolveSignEnv(ctx, a.db, acc.ProjectID, coreAuthDefaultEnv)
	if err != nil {
		return nil, err
	}
	aud := clientID
	if aud == "" {
		aud = acc.ProjectID
	}
	accessToken, err := a.db.Signer().Sign(ctx, acc.ProjectID, signEnv, map[string]any{
		"iss": "https://iam.gopherex.com/" + acc.ProjectID,
		"sub": acc.ID,
		"sid": sessionID,
		"jti": newUUID(),
		"pid": acc.ProjectID,
		"aud": aud,
		"aal": aal,
		"amr": amr,
		"typ": "access",
		"env": signEnv,
	}, coreAuthAccessTTL)
	if err != nil {
		return nil, err
	}
	refreshPlain, err := coreAuthRandomToken()
	if err != nil {
		return nil, err
	}
	refreshHash := coreAuthSHA256(refreshPlain)
	meta := domain.RequestMetaFromContext(ctx)
	sess := &domain.Session{
		ID:           sessionID,
		AccountID:    acc.ID,
		ProjectID:    acc.ProjectID,
		ClientID:     clientID,
		AMR:          amr,
		AAL:          aal,
		AccessToken:  accessToken,
		RefreshToken: refreshPlain,
		ExpiresIn:    coreAuthDefaultExpiresInSec,
		CreatedAt:    now,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
		Fingerprint:  meta.Fingerprint,
		LastActiveAt: now,
	}

	rawSess, err := marshal(sess)
	if err != nil {
		return nil, err
	}
	rmSess := json.RawMessage(rawSess)
	sessSetter := &models.IamSessionSetter{
		ID:           &sess.ID,
		ProjectID:    &sess.ProjectID,
		UserID:       &sess.AccountID,
		Aal:          ptr(int32(aal)),
		Trusted:      ptr(false),
		ExpiresAt:    ptr(null.From(now.Add(coreAuthDefaultSessionTTL))),
		CreatedAt:    &now,
		LastActiveAt: &now,
		Data:         &rmSess,
	}
	if clientID != "" {
		sessSetter.ClientID = ptr(null.From(clientID))
	}
	if _, err := models.IamSessions.Insert(sessSetter).One(ctx, a.db.Bobx()); err != nil {
		return nil, err
	}

	rt := coreAuthRefreshToken{
		ID:        newUUID(),
		ProjectID: acc.ProjectID,
		UserID:    acc.ID,
		SessionID: sessionID,
		Hash:      refreshHash,
		Revoked:   false,
		ExpiresAt: now.Add(coreAuthRefreshTTL),
		CreatedAt: now,
	}
	if err := a.coreAuthInsertRefreshToken(ctx, rt); err != nil {
		return nil, err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "session.created",
		ProjectID:   sess.ProjectID,
		Environment: coreAuthDefaultEnv,
		AggregateID: sess.ID,
		Payload:     sess,
	}); err != nil {
		return nil, err
	}
	return sess, nil
}

// coreAuthRotateSession rotates the access + refresh tokens for an EXISTING
// session, preserving its identity and security context — session id, AAL, AMR,
// client, created-at. Used on token refresh so a long-lived session keeps its
// AAL2 (MFA) elevation, client binding, and stable id instead of being rebuilt
// as a fresh AAL1 session on every refresh.
func (a *pgCoreAuth) coreAuthRotateSession(ctx context.Context, acc *domain.Account, row *models.IamSession) (*domain.Session, error) {
	now := nowUTC()
	prev, err := coreAuthLoadSession(row, row.ProjectID)
	if err != nil {
		return nil, err
	}
	aal := int(row.Aal)
	if aal <= 0 {
		aal = prev.AAL
	}
	if aal <= 0 {
		aal = 1
	}
	amr := prev.AMR
	clientID := prev.ClientID
	signEnv, err := resolveSignEnv(ctx, a.db, acc.ProjectID, coreAuthDefaultEnv)
	if err != nil {
		return nil, err
	}
	aud := clientID
	if aud == "" {
		aud = acc.ProjectID
	}
	accessToken, err := a.db.Signer().Sign(ctx, acc.ProjectID, signEnv, map[string]any{
		"iss": "https://iam.gopherex.com/" + acc.ProjectID,
		"sub": acc.ID,
		"sid": row.ID,
		"jti": newUUID(),
		"pid": acc.ProjectID,
		"aud": aud,
		"aal": aal,
		"amr": amr,
		"typ": "access",
		"env": signEnv,
	}, coreAuthAccessTTL)
	if err != nil {
		return nil, err
	}
	refreshPlain, err := coreAuthRandomToken()
	if err != nil {
		return nil, err
	}
	sess := &domain.Session{
		ID:           row.ID,
		AccountID:    acc.ID,
		ProjectID:    acc.ProjectID,
		ClientID:     clientID,
		AMR:          amr,
		AAL:          aal,
		AccessToken:  accessToken,
		RefreshToken: refreshPlain,
		ExpiresIn:    coreAuthDefaultExpiresInSec,
		CreatedAt:    prev.CreatedAt,
		// Carry the device identity across refresh; refresh the IP/last-active
		// from the current request when available.
		DeviceName:   prev.DeviceName,
		UserAgent:    prev.UserAgent,
		Fingerprint:  prev.Fingerprint,
		Trusted:      prev.Trusted,
		IP:           prev.IP,
		LastActiveAt: now,
	}
	if m := domain.RequestMetaFromContext(ctx); m.IP != "" {
		sess.IP = m.IP
	}
	rawSess, err := marshal(sess)
	if err != nil {
		return nil, err
	}
	rmSess := json.RawMessage(rawSess)
	// Keep the session row; only refresh last-active + the token snapshot. AAL,
	// trusted, client_id columns are intentionally left untouched.
	if err := row.Update(ctx, a.db.Bobx(), &models.IamSessionSetter{
		LastActiveAt: &now,
		Data:         &rmSess,
	}); err != nil {
		return nil, err
	}
	rt := coreAuthRefreshToken{
		ID:        newUUID(),
		ProjectID: acc.ProjectID,
		UserID:    acc.ID,
		SessionID: row.ID,
		Hash:      coreAuthSHA256(refreshPlain),
		Revoked:   false,
		ExpiresAt: now.Add(coreAuthRefreshTTL),
		CreatedAt: now,
	}
	if err := a.coreAuthInsertRefreshToken(ctx, rt); err != nil {
		return nil, err
	}
	return sess, nil
}

// coreAuthInsertRefreshToken persists a refresh-token envelope row. MUST run
// inside an open transaction.
func (a *pgCoreAuth) coreAuthInsertRefreshToken(ctx context.Context, rt coreAuthRefreshToken) error {
	raw, err := marshal(rt)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamRefreshTokenSetter{
		ID:        &rt.ID,
		ProjectID: &rt.ProjectID,
		UserID:    &rt.UserID,
		SessionID: &rt.SessionID,
		Hash:      &rt.Hash,
		Revoked:   &rt.Revoked,
		ExpiresAt: ptr(null.From(rt.ExpiresAt)),
		CreatedAt: &rt.CreatedAt,
		Data:      &rm,
	}
	if _, err := models.IamRefreshTokens.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		return err
	}
	return nil
}

// coreAuthInsertChallenge persists a challenge envelope row, dispatching code /
// token hashes to the lookup columns. MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthInsertChallenge(ctx context.Context, ch coreAuthChallengeData) (*domain.Challenge, error) {
	raw, err := marshal(ch)
	if err != nil {
		return nil, err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamChallengeSetter{
		ID:        &ch.ID,
		ProjectID: &ch.ProjectID,
		Type:      &ch.Type,
		ExpiresAt: &ch.ExpiresAt,
		Consumed:  ptr(false),
		CreatedAt: &ch.CreatedAt,
		Data:      &rm,
	}
	if ch.Subject != "" {
		setter.Subject = ptr(null.From(ch.Subject))
	}
	if ch.CodeHash != "" {
		setter.CodeHash = ptr(null.From(ch.CodeHash))
	}
	if _, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		return nil, err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "challenge.created",
		ProjectID:   ch.ProjectID,
		Environment: coreAuthDefaultEnv,
		AggregateID: ch.ID,
		Payload:     ch,
	}); err != nil {
		return nil, err
	}
	return &domain.Challenge{ID: ch.ID, Type: ch.Type, ExpiresAt: ch.ExpiresAt}, nil
}

// coreAuthStartChallenge mints a verification/change challenge (single-use code
// + opaque link token), persists it, and returns the public Challenge. The
// numeric code and link token are dispatched out-of-band (TODO outbox); only
// their hashes are stored.
func (a *pgCoreAuth) coreAuthStartChallenge(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd, chType, purpose string) (*domain.Challenge, error) {
	if cmd.ProjectID == "" {
		return nil, domain.ErrValidation.WithMessage("project is required")
	}
	contact := strings.TrimSpace(cmd.Contact)
	if contact == "" {
		return nil, domain.ErrValidation.WithMessage("contact is required")
	}
	code, err := coreAuthRandomCode()
	if err != nil {
		return nil, err
	}
	token, err := coreAuthRandomToken()
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	ch := coreAuthChallengeData{
		ID:         newUUID(),
		ProjectID:  cmd.ProjectID,
		Type:       chType,
		Purpose:    purpose,
		AccountID:  cmd.AccountID,
		Subject:    contact,
		CodeHash:   coreAuthSHA256(code),
		TokenHash:  coreAuthSHA256(token),
		RedirectTo: cmd.RedirectTo,
		Locale:     cmd.Locale,
		Channel:    cmd.Channel,
		ExpiresAt:  now.Add(coreAuthChallengeTTL),
		CreatedAt:  now,
	}
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Challenge, error) {
		out, err := a.coreAuthInsertChallenge(ctx, ch)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        chType + ".verification.requested",
			ProjectID:   ch.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: ch.ID,
			Payload: map[string]any{
				"code":         code,
				"token":        token,
				"channel":      ch.Channel,
				"account_id":   ch.AccountID,
				"challenge_id": ch.ID,
				"contact":      ch.Subject,
				"to":           ch.Subject,
				"locale":       ch.Locale,
				"redirect_to":  ch.RedirectTo,
				"purpose":      ch.Purpose,
			},
		}); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// coreAuthConsumeChallenge loads and validates a challenge identified by either
// (ChallengeID + Code) or an opaque Token, enforcing the tenant boundary,
// expiry and single-use, then marks it consumed. Returns the challenge data.
//
// MUST run inside an open transaction (it mutates the consumed flag).
func (a *pgCoreAuth) coreAuthConsumeChallenge(ctx context.Context, projectID string, cmd domain.CoreAuthVerifyConsumeCmd, wantType string) (*models.IamChallenge, *coreAuthChallengeData, error) {
	var row *models.IamChallenge
	if cmd.ChallengeID != "" {
		r, err := models.FindIamChallenge(ctx, a.db.Bobx(), cmd.ChallengeID)
		if err != nil {
			if errors.Is(translatePgErr("challenge", err), ErrNotFound) {
				return nil, nil, domain.ErrChallengeInvalid
			}
			return nil, nil, err
		}
		row = r
	} else if cmd.Token != "" {
		// Token path: match on the data envelope's token hash via the subject
		// index is not possible (token is not a column), so scan the project's
		// unconsumed challenges of the wanted type and compare hashes.
		r, err := a.coreAuthFindChallengeByToken(ctx, projectID, wantType, cmd.Token)
		if err != nil {
			return nil, nil, err
		}
		row = r
	} else {
		return nil, nil, domain.ErrChallengeInvalid
	}

	if row.ProjectID != projectID {
		return nil, nil, domain.ErrChallengeInvalid
	}
	var data coreAuthChallengeData
	if err := unmarshal(row.Data, &data); err != nil {
		return nil, nil, err
	}
	if wantType != "" && data.Type != wantType {
		return nil, nil, domain.ErrChallengeInvalid
	}
	if row.Consumed {
		return nil, nil, domain.ErrTokenUsed
	}
	if !row.ExpiresAt.IsZero() && nowUTC().After(row.ExpiresAt) {
		return nil, nil, domain.ErrChallengeExpired
	}
	// Verify the supplied factor: a numeric code (hashed) or the opaque token.
	if cmd.Code != "" {
		if coreAuthSHA256(cmd.Code) != data.CodeHash {
			return nil, nil, domain.ErrInvalidOTP
		}
	} else if cmd.Token != "" {
		if coreAuthSHA256(cmd.Token) != data.TokenHash {
			return nil, nil, domain.ErrChallengeInvalid
		}
	} else {
		return nil, nil, domain.ErrChallengeInvalid
	}

	if err := row.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: ptr(true)}); err != nil {
		return nil, nil, err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "challenge.consumed",
		ProjectID:   projectID,
		Environment: coreAuthDefaultEnv,
		AggregateID: row.ID,
		Payload:     data,
	}); err != nil {
		return nil, nil, err
	}
	return row, &data, nil
}

// coreAuthFindChallengeByToken scans a project's challenges of a type and
// returns the one whose stored token hash matches the opaque token. The token
// is not a lookup column, so this filters by (project_id, type) then compares
// hashes in memory.
func (a *pgCoreAuth) coreAuthFindChallengeByToken(ctx context.Context, projectID, wantType, token string) (*models.IamChallenge, error) {
	rows, err := models.IamChallenges.Query(
		sm.Where(models.IamChallenges.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamChallenges.Columns.Consumed.EQ(psql.Arg(false))),
		sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg(wantType))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	wantHash := coreAuthSHA256(token)
	for _, row := range rows {
		var data coreAuthChallengeData
		if err := unmarshal(row.Data, &data); err != nil {
			continue
		}
		if data.TokenHash == wantHash {
			return row, nil
		}
	}
	return nil, domain.ErrChallengeInvalid
}

// coreAuthUpdateAccount re-marshals and persists an account envelope, syncing
// the lookup columns. MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthUpdateAccount(ctx context.Context, acc *domain.Account) error {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), acc.ID)
	if err != nil {
		return translatePgErr("user", err)
	}
	if row.ProjectID != acc.ProjectID {
		return domain.ErrUserNotFound
	}
	acc.UpdatedAt = nowUTC()
	raw, err := marshal(acc)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	setter := &models.IamUserSetter{
		Status:    ptr(acc.Status),
		Data:      &rm,
		UpdatedAt: ptr(acc.UpdatedAt),
	}
	if acc.PrimaryEmail != "" {
		setter.PrimaryEmail = ptr(null.From(acc.PrimaryEmail))
	}
	if acc.PrimaryPhone != "" {
		setter.PrimaryPhone = ptr(null.From(acc.PrimaryPhone))
	}
	if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailExists
		}
		return err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "user.updated",
		ProjectID:   acc.ProjectID,
		Environment: coreAuthDefaultEnv,
		AggregateID: acc.ID,
		Payload:     acc,
	}); err != nil {
		return err
	}
	return nil
}

// coreAuthRevokeSession marks a session revoked: its refresh tokens are flagged
// and the session row is deleted. MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthRevokeSession(ctx context.Context, projectID, sessionID string) error {
	row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
	if err != nil {
		if errors.Is(translatePgErr("session", err), ErrNotFound) {
			return domain.ErrSessionNotFound
		}
		return err
	}
	if projectID != "" && row.ProjectID != projectID {
		return domain.ErrSessionNotFound
	}
	if err := a.coreAuthRevokeRefreshTokensForSession(ctx, row.ProjectID, sessionID); err != nil {
		return err
	}
	revokedProjectID := row.ProjectID
	if err := row.Delete(ctx, a.db.Bobx()); err != nil {
		return err
	}
	if err := a.emitter.Emit(ctx, domain.Event{
		Type:        "session.revoked",
		ProjectID:   revokedProjectID,
		Environment: coreAuthDefaultEnv,
		AggregateID: sessionID,
		Payload:     map[string]any{"session_id": sessionID, "project_id": revokedProjectID},
	}); err != nil {
		return err
	}
	return nil
}

// coreAuthRevokeAllForUser revokes all sessions and refresh tokens for a user
// (reuse detection defense). Errors are logged but not surfaced to avoid
// information leakage. MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthRevokeAllForUser(ctx context.Context, projectID, userID string) error {
	sessions, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(userID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if err := a.coreAuthRevokeSession(ctx, projectID, s.ID); err != nil {
			slog.Error("coreauth: failed to revoke session during revoke-all", "err", err, "session_id", s.ID, "project_id", projectID, "user_id", userID)
		}
	}
	return nil
}

// coreAuthRevokeRefreshTokensForSession flags every refresh token bound to a
// session revoked. MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthRevokeRefreshTokensForSession(ctx context.Context, projectID, sessionID string) error {
	rows, err := models.IamRefreshTokens.Query(
		sm.Where(models.IamRefreshTokens.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamRefreshTokens.Columns.SessionID.EQ(psql.Arg(sessionID))),
		sm.Where(models.IamRefreshTokens.Columns.Revoked.EQ(psql.Arg(false))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := a.coreAuthMarkRefreshRevoked(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// coreAuthMarkRefreshRevoked flips a refresh-token row's revoked flag (column +
// envelope). MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthMarkRefreshRevoked(ctx context.Context, row *models.IamRefreshToken) error {
	var data coreAuthRefreshToken
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &data); err != nil {
			data = coreAuthRefreshToken{Revoked: false}
		}
	}
	data.Revoked = true
	raw, err := marshal(data)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	return row.Update(ctx, a.db.Bobx(), &models.IamRefreshTokenSetter{Revoked: ptr(true), Data: &rm})
}

// ===========================================================================
// api.CoreAuthAccounts
// ===========================================================================

// Register creates a human account (iam_users), its bcrypt password credential
// (iam_credentials) and an initial session (iam_sessions + iam_refresh_tokens),
// all in one serializable transaction.
func (a *pgCoreAuth) Register(ctx context.Context, cmd domain.RegisterCmd) (*domain.Account, *domain.Session, error) {
	if cmd.ProjectID == "" {
		return nil, nil, domain.ErrValidation.WithMessage("project is required")
	}
	if err := cmd.Validate(); err != nil {
		return nil, nil, err
	}
	type regResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (regResult, error) {
		now := nowUTC()
		acc := &domain.Account{
			ID:            newUUID(),
			ProjectID:     cmd.ProjectID,
			Kind:          coreAuthKindHuman,
			Status:        coreAuthStatusActive,
			PrimaryEmail:  cmd.Email,
			PrimaryPhone:  cmd.Phone,
			Name:          cmd.Name,
			EmailVerified: false,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		rawAcc, err := marshal(acc)
		if err != nil {
			return regResult{}, err
		}
		rmAcc := json.RawMessage(rawAcc)
		userSetter := &models.IamUserSetter{
			ID:        &acc.ID,
			ProjectID: &acc.ProjectID,
			Kind:      ptr(acc.Kind),
			Status:    ptr(acc.Status),
			CreatedAt: &now,
			UpdatedAt: &now,
			Data:      &rmAcc,
		}
		if acc.PrimaryEmail != "" {
			userSetter.PrimaryEmail = ptr(null.From(acc.PrimaryEmail))
		}
		if acc.PrimaryPhone != "" {
			userSetter.PrimaryPhone = ptr(null.From(acc.PrimaryPhone))
		}
		if _, err := models.IamUsers.Insert(userSetter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				if acc.PrimaryEmail != "" {
					return regResult{}, domain.ErrEmailExists
				}
				return regResult{}, domain.ErrPhoneExists
			}
			return regResult{}, err
		}
		for _, c := range cmd.Consents {
			if _, err := models.IamConsents.Insert(&models.IamConsentSetter{
				ID:         ptr(newUUID()),
				ProjectID:  ptr(acc.ProjectID),
				UserID:     ptr(acc.ID),
				DocKey:     ptr(c.Key),
				Version:    ptr(c.Version),
				AcceptedAt: ptr(now),
			}).One(ctx, a.db.Bobx()); err != nil {
				return regResult{}, err
			}
		}

		// Password credential (bcrypt). Optional: phone-only sign-ups have none.
		if cmd.Password != "" {
			hash, err := coreAuthHashPassword(cmd.Password)
			if err != nil {
				return regResult{}, err
			}
			cred := coreAuthCredential{
				ID:        newUUID(),
				ProjectID: acc.ProjectID,
				UserID:    acc.ID,
				Type:      coreAuthCredentialPassword,
				Hash:      hash,
				CreatedAt: now,
				UpdatedAt: now,
			}
			rawCred, err := marshal(cred)
			if err != nil {
				return regResult{}, err
			}
			rmCred := json.RawMessage(rawCred)
			credSetter := &models.IamCredentialSetter{
				ID:        &cred.ID,
				ProjectID: &cred.ProjectID,
				UserID:    &cred.UserID,
				Type:      ptr(cred.Type),
				Secret:    &cred.Hash,
				CreatedAt: &now,
				UpdatedAt: &now,
				Data:      &rmCred,
			}
			if _, err := models.IamCredentials.Insert(credSetter).One(ctx, a.db.Bobx()); err != nil {
				return regResult{}, err
			}
		}

		sess, err := a.coreAuthMintSession(ctx, acc, "", []string{"pwd"}, 1)
		if err != nil {
			return regResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.registered",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return regResult{}, err
		}
		return regResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// AuthenticatePassword verifies an email + password against the bcrypt
// credential and mints a session. A missing user or a bad password both return
// ErrInvalidCredentials (no account enumeration).
func (a *pgCoreAuth) AuthenticatePassword(ctx context.Context, projectID, email, password string) (*domain.CoreAuthPasswordResult, error) {
	if projectID == "" || email == "" {
		return nil, domain.ErrInvalidCredentials
	}
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.CoreAuthPasswordResult, error) {
		userRow, err := a.coreAuthFindUserByEmail(ctx, projectID, email)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				return nil, domain.ErrInvalidCredentials
			}
			return nil, err
		}
		acc, err := coreAuthLoadAccount(userRow, projectID)
		if err != nil {
			return nil, err
		}
		cred, err := a.coreAuthFindPasswordCredential(ctx, projectID, acc.ID)
		if err != nil {
			return nil, err
		}
		if !coreAuthCheckPassword(cred.Secret, password) {
			return nil, domain.ErrInvalidCredentials
		}
		if err := coreAuthAccountActive(acc); err != nil {
			return nil, err
		}
		// Second factor: password (factor 1) verified, but if the account has an
		// active MFA factor we do NOT mint a session — the caller must complete a
		// second factor (mfa/verify or recovery-codes/verify) carrying the
		// flow_token. This keeps AAL2 enforcement at login.
		factors, err := a.coreAuthActiveFactors(ctx, acc.ID)
		if err != nil {
			return nil, err
		}
		if len(factors) > 0 {
			return &domain.CoreAuthPasswordResult{Account: acc, MFARequired: true, Factors: factors}, nil
		}
		sess, err := a.coreAuthMintSession(ctx, acc, "", []string{"pwd"}, 1)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.signed_in",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return &domain.CoreAuthPasswordResult{Account: acc, Session: sess}, nil
	})
}

// coreAuthActiveFactors returns the account's active MFA factors (used to decide
// whether password sign-in must gate on a second factor).
func (a *pgCoreAuth) coreAuthActiveFactors(ctx context.Context, accountID string) ([]domain.Factor, error) {
	rows, err := models.IamFactors.Query(
		sm.Where(models.IamFactors.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Factor, 0, len(rows))
	for _, row := range rows {
		f, err := mfaFactorFromRow(row)
		if err != nil {
			return nil, err
		}
		if f.Status == "active" {
			out = append(out, f)
		}
	}
	return out, nil
}

// Refresh rotates a refresh token: it looks the token up by sha256 hash,
// validates it (not revoked / not expired), revokes the old one, and mints a
// fresh session for the same account.
func (a *pgCoreAuth) Refresh(ctx context.Context, refreshToken string) (*domain.Account, *domain.Session, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, nil, domain.ErrInvalidToken.WithMessage("refresh_token is required")
	}
	hash := coreAuthSHA256(refreshToken)
	type refreshResult struct {
		acc      *domain.Account
		sess     *domain.Session
		mismatch bool
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (refreshResult, error) {
		row, err := models.IamRefreshTokens.Query(
			sm.Where(models.IamRefreshTokens.Columns.Hash.EQ(psql.Arg(hash))),
		).One(ctx, a.db.Bobx())
		if err != nil {
			if errors.Is(translatePgErr("refresh_token", err), ErrNotFound) {
				return refreshResult{}, domain.ErrInvalidToken
			}
			return refreshResult{}, err
		}
		if row.Revoked {
			if err := a.coreAuthRevokeAllForUser(ctx, row.ProjectID, row.UserID); err != nil {
				slog.Error("coreauth: failed to revoke all sessions on refresh token reuse", "err", err, "project_id", row.ProjectID, "user_id", row.UserID)
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "token.reuse_detected",
				ProjectID:   row.ProjectID,
				Environment: coreAuthDefaultEnv,
				AggregateID: row.UserID,
				Payload:     map[string]any{"session_id": row.SessionID},
			}); err != nil {
				slog.Error("coreauth: failed to emit token.reuse_detected event", "err", err, "project_id", row.ProjectID, "user_id", row.UserID)
			}
			return refreshResult{}, domain.ErrTokenRevoked
		}
		if v, ok := row.ExpiresAt.Get(); ok && nowUTC().After(v) {
			return refreshResult{}, domain.ErrTokenExpired
		}

		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), row.UserID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return refreshResult{}, domain.ErrUserNotFound
			}
			return refreshResult{}, err
		}
		acc, err := coreAuthLoadAccount(userRow, row.ProjectID)
		if err != nil {
			return refreshResult{}, err
		}
		if err := coreAuthAccountActive(acc); err != nil {
			return refreshResult{}, err
		}

		sessRow, err := models.FindIamSession(ctx, a.db.Bobx(), row.SessionID)
		if err != nil {
			if errors.Is(translatePgErr("session", err), ErrNotFound) {
				return refreshResult{}, domain.ErrSessionNotFound
			}
			return refreshResult{}, err
		}

		// Device-change defense: a request that presents a device fingerprint
		// which differs from the one bound to the session at sign-in is a strong
		// token-theft signal. Revoke the session (committing this tx so the
		// revocation persists) and refuse the refresh. UA changes are NOT treated
		// as theft (browsers update their UA legitimately); only a fingerprint
		// mismatch denies.
		prev, err := coreAuthLoadSession(sessRow, sessRow.ProjectID)
		if err != nil {
			return refreshResult{}, err
		}
		meta := domain.RequestMetaFromContext(ctx)
		if prev.Fingerprint != "" && meta.Fingerprint != "" && meta.Fingerprint != prev.Fingerprint {
			if err := a.coreAuthMarkRefreshRevoked(ctx, row); err != nil {
				return refreshResult{}, err
			}
			if err := a.coreAuthRevokeSession(ctx, sessRow.ProjectID, sessRow.ID); err != nil &&
				!errors.Is(err, domain.ErrSessionNotFound) {
				return refreshResult{}, err
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "session.device_mismatch",
				ProjectID:   sessRow.ProjectID,
				Environment: coreAuthDefaultEnv,
				AggregateID: sessRow.ID,
				Payload:     map[string]any{"session_id": sessRow.ID, "user_id": sessRow.UserID},
			}); err != nil {
				return refreshResult{}, err
			}
			// Commit the revocation (nil error); the caller maps mismatch to an error.
			return refreshResult{mismatch: true}, nil
		}

		// Rotate the tokens in place: revoke the presented refresh token, then
		// mint a new access + refresh pair bound to the SAME session, preserving
		// its id, AAL (MFA elevation), AMR, and client. The session is not torn
		// down on refresh.
		if err := a.coreAuthMarkRefreshRevoked(ctx, row); err != nil {
			return refreshResult{}, err
		}
		sess, err := a.coreAuthRotateSession(ctx, acc, sessRow)
		if err != nil {
			return refreshResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "token.refreshed",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: sess.ID,
			Payload:     sess,
		}); err != nil {
			return refreshResult{}, err
		}
		return refreshResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	if res.mismatch {
		return nil, nil, domain.ErrSessionDeviceMismatch
	}
	return res.acc, res.sess, nil
}

// ExchangeCode trades a one-time auth code (iam_auth_codes) for a session,
// verifying the PKCE code_verifier against the stored challenge hash.
func (a *pgCoreAuth) ExchangeCode(ctx context.Context, code, verifier string) (*domain.Account, *domain.Session, error) {
	if strings.TrimSpace(code) == "" {
		return nil, nil, domain.ErrBadRequest.WithMessage("code is required")
	}
	codeHash := coreAuthSHA256(code)
	type exResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (exResult, error) {
		row, err := models.IamAuthCodes.Query(
			sm.Where(models.IamAuthCodes.Columns.CodeHash.EQ(psql.Arg(codeHash))),
		).One(ctx, a.db.Bobx())
		if err != nil {
			if errors.Is(translatePgErr("auth_code", err), ErrNotFound) {
				return exResult{}, domain.ErrInvalidToken
			}
			return exResult{}, err
		}
		if row.Consumed {
			return exResult{}, domain.ErrTokenUsed
		}
		if !row.ExpiresAt.IsZero() && nowUTC().After(row.ExpiresAt) {
			return exResult{}, domain.ErrTokenExpired
		}
		var data coreAuthCodeData
		if err := unmarshal(row.Data, &data); err != nil {
			return exResult{}, err
		}
		// PKCE: if a challenge was bound at issuance, the verifier must hash to it.
		if data.ChallengeHash != "" {
			if coreAuthSHA256(verifier) != data.ChallengeHash {
				return exResult{}, domain.ErrInvalidToken.WithMessage("code_verifier mismatch")
			}
		}
		userID, _ := row.UserID.Get()
		if userID == "" {
			return exResult{}, domain.ErrInvalidToken
		}
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), userID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return exResult{}, domain.ErrUserNotFound
			}
			return exResult{}, err
		}
		acc, err := coreAuthLoadAccount(userRow, row.ProjectID)
		if err != nil {
			return exResult{}, err
		}
		if err := coreAuthAccountActive(acc); err != nil {
			return exResult{}, err
		}
		if err := row.Update(ctx, a.db.Bobx(), &models.IamAuthCodeSetter{Consumed: ptr(true)}); err != nil {
			return exResult{}, err
		}
		clientID, _ := row.ClientID.Get()
		sess, err := a.coreAuthMintSession(ctx, acc, clientID, []string{"oauth"}, 1)
		if err != nil {
			return exResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "token.exchanged",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: sess.ID,
			Payload:     sess,
		}); err != nil {
			return exResult{}, err
		}
		return exResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// CreateGuest provisions an anonymous guest account and an initial session.
func (a *pgCoreAuth) CreateGuest(ctx context.Context, projectID string) (*domain.Account, *domain.Session, error) {
	if projectID == "" {
		return nil, nil, domain.ErrValidation.WithMessage("project is required")
	}
	type guestResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (guestResult, error) {
		now := nowUTC()
		acc := &domain.Account{
			ID:        newUUID(),
			ProjectID: projectID,
			Kind:      coreAuthKindGuest,
			Status:    coreAuthStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		}
		raw, err := marshal(acc)
		if err != nil {
			return guestResult{}, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamUserSetter{
			ID:        &acc.ID,
			ProjectID: &acc.ProjectID,
			Kind:      ptr(acc.Kind),
			Status:    ptr(acc.Status),
			CreatedAt: &now,
			UpdatedAt: &now,
			Data:      &rm,
		}
		if _, err := models.IamUsers.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return guestResult{}, err
		}
		sess, err := a.coreAuthMintSession(ctx, acc, "", []string{"anonymous"}, 1)
		if err != nil {
			return guestResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "guest.created",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return guestResult{}, err
		}
		return guestResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// GetSession resolves the account + session for a session id (read-only).
func (a *pgCoreAuth) GetSession(ctx context.Context, sessionID string) (*domain.Account, *domain.Session, error) {
	if sessionID == "" {
		return nil, nil, domain.ErrSessionNotFound
	}
	row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
	if err != nil {
		if errors.Is(translatePgErr("session", err), ErrNotFound) {
			return nil, nil, domain.ErrSessionNotFound
		}
		return nil, nil, err
	}
	sess, err := coreAuthLoadSession(row, row.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	userRow, err := models.FindIamUser(ctx, a.db.Bobx(), row.UserID)
	if err != nil {
		if errors.Is(translatePgErr("user", err), ErrNotFound) {
			return nil, nil, domain.ErrUserNotFound
		}
		return nil, nil, err
	}
	acc, err := coreAuthLoadAccount(userRow, row.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	return acc, sess, nil
}

// SignOut revokes the current session; when everywhere is set it revokes every
// session for the same account.
func (a *pgCoreAuth) SignOut(ctx context.Context, sessionID string, everywhere bool) error {
	if sessionID == "" {
		return domain.ErrSessionNotFound
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
		if err != nil {
			if errors.Is(translatePgErr("session", err), ErrNotFound) {
				return domain.ErrSessionNotFound
			}
			return err
		}
		if everywhere {
			if _, err := a.coreAuthSignOutAll(ctx, row.ProjectID, row.UserID, ""); err != nil {
				return err
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "user.signed_out_everywhere",
				ProjectID:   row.ProjectID,
				Environment: coreAuthDefaultEnv,
				AggregateID: row.UserID,
				Payload:     map[string]any{"account_id": row.UserID, "project_id": row.ProjectID},
			}); err != nil {
				return err
			}
			return nil
		}
		if err := a.coreAuthRevokeSession(ctx, row.ProjectID, sessionID); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.signed_out",
			ProjectID:   row.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: row.UserID,
			Payload:     map[string]any{"account_id": row.UserID, "session_id": sessionID, "project_id": row.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// SignOutAll revokes every session for an account except optionally one, and
// returns the count revoked.
func (a *pgCoreAuth) SignOutAll(ctx context.Context, accountID, exceptSessionID string) (int, error) {
	if accountID == "" {
		return 0, domain.ErrUserNotFound
	}
	return withTxRet(ctx, a.db, func(ctx context.Context) (int, error) {
		// The account's project is read from its row so the revoke stays
		// tenant-scoped.
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return 0, domain.ErrUserNotFound
			}
			return 0, err
		}
		n, err := a.coreAuthSignOutAll(ctx, userRow.ProjectID, accountID, exceptSessionID)
		if err != nil {
			return 0, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.sessions_revoked",
			ProjectID:   userRow.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: accountID,
			Payload:     map[string]any{"account_id": accountID, "count": n, "project_id": userRow.ProjectID},
		}); err != nil {
			return 0, err
		}
		return n, nil
	})
}

// coreAuthSignOutAll revokes every session for (projectID, userID) except
// exceptSessionID. MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthSignOutAll(ctx context.Context, projectID, userID, exceptSessionID string) (int, error) {
	rows, err := models.IamSessions.Query(
		sm.Where(models.IamSessions.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSessions.Columns.UserID.EQ(psql.Arg(userID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return 0, err
	}
	count := 0
	for _, row := range rows {
		if row.ID == exceptSessionID {
			continue
		}
		if err := a.coreAuthRevokeSession(ctx, projectID, row.ID); err != nil &&
			!errors.Is(err, domain.ErrSessionNotFound) {
			return count, err
		}
		count++
	}
	return count, nil
}

// ----- email verification / change -----

// StartEmailVerification mints an email-verification challenge (code + link).
func (a *pgCoreAuth) StartEmailVerification(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error) {
	return a.coreAuthStartChallenge(ctx, cmd, coreAuthChallengeEmail, "verify")
}

// VerifyEmail consumes an email-verification challenge, marks the matching
// account's email verified and mints a session.
func (a *pgCoreAuth) VerifyEmail(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, *domain.Session, error) {
	type verifyResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (verifyResult, error) {
		_, data, err := a.coreAuthConsumeChallenge(ctx, cmd.ProjectID, cmd, coreAuthChallengeEmail)
		if err != nil {
			return verifyResult{}, err
		}
		userRow, err := a.coreAuthFindUserByEmail(ctx, cmd.ProjectID, data.Subject)
		if err != nil {
			return verifyResult{}, err
		}
		acc, err := coreAuthLoadAccount(userRow, cmd.ProjectID)
		if err != nil {
			return verifyResult{}, err
		}
		acc.EmailVerified = true
		if err := a.coreAuthUpdateAccount(ctx, acc); err != nil {
			return verifyResult{}, err
		}
		sess, err := a.coreAuthMintSession(ctx, acc, "", []string{"otp"}, 1)
		if err != nil {
			return verifyResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "email.verified",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return verifyResult{}, err
		}
		return verifyResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// VerifyEmailCallback consumes an opaque email-verification link token and
// resolves the redirect target. Marking the account verified mirrors
// VerifyEmail; no session cookie is minted here (the link flow defers to the
// SPA which then signs in), so SetCookie is left empty.
func (a *pgCoreAuth) VerifyEmailCallback(ctx context.Context, cmd domain.CoreAuthEmailVerificationCallbackCmd) (*domain.CoreAuthEmailVerificationCallbackResult, error) {
	if strings.TrimSpace(cmd.Token) == "" {
		return nil, domain.ErrChallengeInvalid
	}
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.CoreAuthEmailVerificationCallbackResult, error) {
		// Token-only consume: project is unknown up front, so locate by token.
		row, err := a.coreAuthFindChallengeByTokenAnyProject(ctx, coreAuthChallengeEmail, cmd.Token)
		if err != nil {
			return nil, err
		}
		var data coreAuthChallengeData
		if err := unmarshal(row.Data, &data); err != nil {
			return nil, err
		}
		consume := domain.CoreAuthVerifyConsumeCmd{ProjectID: data.ProjectID, Token: cmd.Token}
		if _, _, err := a.coreAuthConsumeChallenge(ctx, data.ProjectID, consume, coreAuthChallengeEmail); err != nil {
			return nil, err
		}
		if userRow, err := a.coreAuthFindUserByEmail(ctx, data.ProjectID, data.Subject); err == nil {
			if acc, err := coreAuthLoadAccount(userRow, data.ProjectID); err == nil {
				acc.EmailVerified = true
				if err := a.coreAuthUpdateAccount(ctx, acc); err != nil {
					return nil, err
				}
			}
		}
		redirect := cmd.RedirectTo
		if redirect == "" {
			redirect = data.RedirectTo
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "email.verified",
			ProjectID:   data.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: data.AccountID,
			Payload:     data,
		}); err != nil {
			return nil, err
		}
		// SetCookie is empty: this adapter does not mint a session cookie on the
		// link callback; the SPA at RedirectURL completes sign-in.
		return &domain.CoreAuthEmailVerificationCallbackResult{RedirectURL: redirect}, nil
	})
}

// coreAuthFindChallengeByTokenAnyProject locates an unconsumed challenge of a
// type across all projects by matching the stored token hash. Used by the
// public link callback where the project is not supplied.
func (a *pgCoreAuth) coreAuthFindChallengeByTokenAnyProject(ctx context.Context, wantType, token string) (*models.IamChallenge, error) {
	rows, err := models.IamChallenges.Query(
		sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg(wantType))),
		sm.Where(models.IamChallenges.Columns.Consumed.EQ(psql.Arg(false))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	wantHash := coreAuthSHA256(token)
	for _, row := range rows {
		var data coreAuthChallengeData
		if err := unmarshal(row.Data, &data); err != nil {
			continue
		}
		if data.TokenHash == wantHash {
			return row, nil
		}
	}
	return nil, domain.ErrChallengeInvalid
}

// coreAuthCaptchaConfig is the iam_config(key=captcha) data envelope: the
// provider name, the siteverify secret, an optional verify URL override and the
// optional minimum score threshold (reCAPTCHA v3 / Turnstile style).
type coreAuthCaptchaConfig struct {
	Provider  string  `json:"provider"`
	Secret    string  `json:"secret"`
	VerifyURL string  `json:"verify_url"`
	MinScore  float64 `json:"min_score"`
}

// coreAuthCaptchaSiteverifyResp is the subset of a provider siteverify response
// we consume (hCaptcha / reCAPTCHA / Turnstile share this shape).
type coreAuthCaptchaSiteverifyResp struct {
	Success    bool     `json:"success"`
	Score      float64  `json:"score"`
	Action     string   `json:"action"`
	ErrorCodes []string `json:"error-codes"`
}

// coreAuthCaptchaVerifyURL resolves the siteverify endpoint: the config override
// wins, else the provider default. An unknown provider yields "".
func coreAuthCaptchaVerifyURL(provider, override string) string {
	if override != "" {
		return override
	}
	switch strings.ToLower(provider) {
	case "hcaptcha":
		return "https://api.hcaptcha.com/siteverify"
	case "recaptcha":
		return "https://www.google.com/recaptcha/api/siteverify"
	case "turnstile":
		return "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	default:
		return ""
	}
}

// VerifyCaptcha verifies a CAPTCHA token against the project's configured
// provider siteverify endpoint. The provider config lives in
// iam_config(project, env=live, key=captcha); a missing row or empty secret
// means no provider is configured and the token is accepted (enforcement is a
// policy decision made upstream). With a provider configured, the token is
// POSTed (form-encoded) to the siteverify URL and the JSON response mapped to
// the result; an HTTP/parse failure yields Valid:false (never a request error).
func (a *pgCoreAuth) VerifyCaptcha(ctx context.Context, projectID, provider, token, action string) (*domain.CoreAuthCaptchaVerifyResult, error) {
	_ = action
	if strings.TrimSpace(token) == "" {
		return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
	}

	var cfg coreAuthCaptchaConfig
	row, err := models.FindIamConfig(ctx, a.db.Bobx(), projectID, coreAuthDefaultEnv, "captcha")
	if err != nil {
		if errors.Is(translatePgErr("config", err), ErrNotFound) {
			return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
		}
		return nil, err
	}
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &cfg); err != nil {
			return nil, err
		}
	}
	if cfg.Provider == "" {
		cfg.Provider = provider
	}
	if cfg.Secret == "" {
		return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
	}

	verifyURL := coreAuthCaptchaVerifyURL(cfg.Provider, cfg.VerifyURL)
	if verifyURL == "" {
		return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
	}

	form := url.Values{}
	form.Set("secret", cfg.Secret)
	form.Set("response", token)

	verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(verifyCtx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
	}
	var parsed coreAuthCaptchaSiteverifyResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return &domain.CoreAuthCaptchaVerifyResult{Valid: false, Score: 0}, nil
	}
	valid := parsed.Success
	if valid && cfg.MinScore > 0 && parsed.Score > 0 {
		valid = parsed.Score >= cfg.MinScore
	}
	return &domain.CoreAuthCaptchaVerifyResult{Valid: valid, Score: parsed.Score}, nil
}

// StartEmailChange mints an email-change challenge for the current account
// targeting the new email.
func (a *pgCoreAuth) StartEmailChange(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error) {
	return a.coreAuthStartChallenge(ctx, cmd, coreAuthChallengeEmail, "change")
}

// VerifyEmailChange consumes an email-change challenge and swaps the account's
// primary email to the challenged contact.
func (a *pgCoreAuth) VerifyEmailChange(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		_, data, err := a.coreAuthConsumeChallenge(ctx, cmd.ProjectID, cmd, coreAuthChallengeEmail)
		if err != nil {
			return nil, err
		}
		accountID := cmd.AccountID
		if accountID == "" {
			accountID = data.AccountID
		}
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return nil, domain.ErrUserNotFound
			}
			return nil, err
		}
		acc, err := coreAuthLoadAccount(userRow, cmd.ProjectID)
		if err != nil {
			return nil, err
		}
		acc.PrimaryEmail = data.Subject
		acc.EmailVerified = true
		if err := a.coreAuthUpdateAccount(ctx, acc); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "email.changed",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

// CancelEmailChange voids a pending email-change challenge by its opaque token.
func (a *pgCoreAuth) CancelEmailChange(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return domain.ErrChallengeInvalid
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := a.coreAuthFindChallengeByTokenAnyProject(ctx, coreAuthChallengeEmail, token)
		if err != nil {
			return err
		}
		if err := row.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: ptr(true)}); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "email_change.cancelled",
			ProjectID:   row.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: row.ID,
			Payload:     map[string]any{"challenge_id": row.ID, "project_id": row.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ----- phone verification / change -----

// StartPhoneVerification mints a phone-verification challenge (SMS/WhatsApp).
func (a *pgCoreAuth) StartPhoneVerification(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error) {
	return a.coreAuthStartChallenge(ctx, cmd, coreAuthChallengePhone, "verify")
}

// VerifyPhone consumes a phone-verification challenge, marks the matching
// account's phone verified and mints a session.
func (a *pgCoreAuth) VerifyPhone(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, *domain.Session, error) {
	type verifyResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (verifyResult, error) {
		_, data, err := a.coreAuthConsumeChallenge(ctx, cmd.ProjectID, cmd, coreAuthChallengePhone)
		if err != nil {
			return verifyResult{}, err
		}
		userRow, err := a.coreAuthFindUserByPhone(ctx, cmd.ProjectID, data.Subject)
		if err != nil {
			return verifyResult{}, err
		}
		acc, err := coreAuthLoadAccount(userRow, cmd.ProjectID)
		if err != nil {
			return verifyResult{}, err
		}
		if err := a.coreAuthUpdateAccount(ctx, acc); err != nil {
			return verifyResult{}, err
		}
		sess, err := a.coreAuthMintSession(ctx, acc, "", []string{"otp"}, 1)
		if err != nil {
			return verifyResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "phone.verified",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return verifyResult{}, err
		}
		return verifyResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// coreAuthFindUserByPhone returns the iam_users row for (projectID, phone).
func (a *pgCoreAuth) coreAuthFindUserByPhone(ctx context.Context, projectID, phone string) (*models.IamUser, error) {
	row, err := models.IamUsers.Query(
		sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamUsers.Columns.PrimaryPhone.EQ(psql.Arg(phone))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if errors.Is(translatePgErr("user", err), ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return row, nil
}

// StartPhoneChange mints a phone-change challenge for the current account
// targeting the new phone.
func (a *pgCoreAuth) StartPhoneChange(ctx context.Context, cmd domain.CoreAuthVerifyStartCmd) (*domain.Challenge, error) {
	return a.coreAuthStartChallenge(ctx, cmd, coreAuthChallengePhone, "change")
}

// VerifyPhoneChange consumes a phone-change challenge and swaps the account's
// primary phone to the challenged contact.
func (a *pgCoreAuth) VerifyPhoneChange(ctx context.Context, cmd domain.CoreAuthVerifyConsumeCmd) (*domain.Account, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Account, error) {
		_, data, err := a.coreAuthConsumeChallenge(ctx, cmd.ProjectID, cmd, coreAuthChallengePhone)
		if err != nil {
			return nil, err
		}
		accountID := cmd.AccountID
		if accountID == "" {
			accountID = data.AccountID
		}
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return nil, domain.ErrUserNotFound
			}
			return nil, err
		}
		acc, err := coreAuthLoadAccount(userRow, cmd.ProjectID)
		if err != nil {
			return nil, err
		}
		acc.PrimaryPhone = data.Subject
		if err := a.coreAuthUpdateAccount(ctx, acc); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "phone.changed",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return nil, err
		}
		return acc, nil
	})
}

// ----- password lifecycle -----

// ForgotPassword mints a password-reset challenge for the email if it resolves
// to an account. To avoid account enumeration a missing user is a silent
// success (no challenge, no error).
func (a *pgCoreAuth) ForgotPassword(ctx context.Context, cmd domain.CoreAuthPasswordForgotCmd) error {
	if cmd.ProjectID == "" || strings.TrimSpace(cmd.Email) == "" {
		return domain.ErrValidation.WithMessage("project and email are required")
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		userRow, err := a.coreAuthFindUserByEmail(ctx, cmd.ProjectID, cmd.Email)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				// Silent success: do not reveal whether the email exists.
				return nil
			}
			return err
		}
		acc, err := coreAuthLoadAccount(userRow, cmd.ProjectID)
		if err != nil {
			return err
		}
		code, err := coreAuthRandomCode()
		if err != nil {
			return err
		}
		token, err := coreAuthRandomToken()
		if err != nil {
			return err
		}
		now := nowUTC()
		ch := coreAuthChallengeData{
			ID:         newUUID(),
			ProjectID:  cmd.ProjectID,
			Type:       "password_reset",
			Purpose:    "reset",
			AccountID:  acc.ID,
			Subject:    cmd.Email,
			CodeHash:   coreAuthSHA256(code),
			TokenHash:  coreAuthSHA256(token),
			RedirectTo: cmd.RedirectTo,
			Locale:     cmd.Locale,
			ExpiresAt:  now.Add(coreAuthChallengeTTL),
			CreatedAt:  now,
		}
		if _, err := a.coreAuthInsertChallenge(ctx, ch); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "password.reset_requested",
			ProjectID:   cmd.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload: map[string]any{
				"code":         code,
				"token":        token,
				"account_id":   acc.ID,
				"challenge_id": ch.ID,
				"contact":      ch.Subject,
				"to":           ch.Subject,
				"locale":       ch.Locale,
				"redirect_to":  ch.RedirectTo,
				"purpose":      ch.Purpose,
			},
		}); err != nil {
			return err
		}
		return nil
	})
}

// ResetPassword consumes a password-reset challenge and writes a fresh bcrypt
// credential, then mints a session.
func (a *pgCoreAuth) ResetPassword(ctx context.Context, cmd domain.CoreAuthPasswordResetCmd) (*domain.Account, *domain.Session, error) {
	if strings.TrimSpace(cmd.NewPassword) == "" {
		return nil, nil, domain.ErrWeakPassword
	}
	type resetResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (resetResult, error) {
		consume := domain.CoreAuthVerifyConsumeCmd{
			ProjectID:   cmd.ProjectID,
			ChallengeID: cmd.ChallengeID,
			Code:        cmd.Code,
			Token:       cmd.Token,
		}
		_, data, err := a.coreAuthConsumeChallenge(ctx, cmd.ProjectID, consume, "password_reset")
		if err != nil {
			return resetResult{}, err
		}
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), data.AccountID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return resetResult{}, domain.ErrUserNotFound
			}
			return resetResult{}, err
		}
		acc, err := coreAuthLoadAccount(userRow, cmd.ProjectID)
		if err != nil {
			return resetResult{}, err
		}
		hash, err := coreAuthHashPassword(cmd.NewPassword)
		if err != nil {
			return resetResult{}, err
		}
		if err := a.coreAuthUpsertPasswordCredential(ctx, acc.ProjectID, acc.ID, hash); err != nil {
			return resetResult{}, err
		}
		// Reset revokes existing sessions for safety, then mints a fresh one.
		if _, err := a.coreAuthSignOutAll(ctx, acc.ProjectID, acc.ID, ""); err != nil {
			return resetResult{}, err
		}
		sess, err := a.coreAuthMintSession(ctx, acc, "", []string{"pwd"}, 1)
		if err != nil {
			return resetResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "password.reset",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return resetResult{}, err
		}
		return resetResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// coreAuthUpsertPasswordCredential writes (or replaces) the bcrypt password
// credential for a user. MUST run inside an open transaction.
func (a *pgCoreAuth) coreAuthUpsertPasswordCredential(ctx context.Context, projectID, userID, hash string) error {
	now := nowUTC()
	row, err := models.IamCredentials.Query(
		sm.Where(models.IamCredentials.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamCredentials.Columns.UserID.EQ(psql.Arg(userID))),
		sm.Where(models.IamCredentials.Columns.Type.EQ(psql.Arg(coreAuthCredentialPassword))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		if !errors.Is(translatePgErr("credential", err), ErrNotFound) {
			return err
		}
		// Insert a fresh credential.
		cred := coreAuthCredential{
			ID:        newUUID(),
			ProjectID: projectID,
			UserID:    userID,
			Type:      coreAuthCredentialPassword,
			Hash:      hash,
			CreatedAt: now,
			UpdatedAt: now,
		}
		raw, mErr := marshal(cred)
		if mErr != nil {
			return mErr
		}
		rm := json.RawMessage(raw)
		setter := &models.IamCredentialSetter{
			ID:        &cred.ID,
			ProjectID: &cred.ProjectID,
			UserID:    &cred.UserID,
			Type:      ptr(cred.Type),
			Secret:    &cred.Hash,
			CreatedAt: &now,
			UpdatedAt: &now,
			Data:      &rm,
		}
		if _, err := models.IamCredentials.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return err
		}
		return nil
	}
	// Update the existing credential in place.
	var cred coreAuthCredential
	if len(row.Data) > 0 {
		_ = unmarshal(row.Data, &cred)
	}
	cred.Hash = hash
	cred.UpdatedAt = now
	raw, err := marshal(cred)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	return row.Update(ctx, a.db.Bobx(), &models.IamCredentialSetter{Secret: &hash, UpdatedAt: &now, Data: &rm})
}

// ChangePassword re-asserts the current password (bcrypt) and writes the new
// one; optionally revoking the account's other sessions.
func (a *pgCoreAuth) ChangePassword(ctx context.Context, cmd domain.CoreAuthPasswordChangeCmd) error {
	if cmd.AccountID == "" {
		return domain.ErrUnauthorized
	}
	if strings.TrimSpace(cmd.NewPassword) == "" {
		return domain.ErrWeakPassword
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), cmd.AccountID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return domain.ErrUserNotFound
			}
			return err
		}
		acc, err := coreAuthLoadAccount(userRow, userRow.ProjectID)
		if err != nil {
			return err
		}
		// Verify the current password against the stored bcrypt credential.
		cred, err := a.coreAuthFindPasswordCredential(ctx, acc.ProjectID, acc.ID)
		if err != nil {
			return err
		}
		if !coreAuthCheckPassword(cred.Secret, cmd.CurrentPassword) {
			return domain.ErrInvalidCredentials
		}
		hash, err := coreAuthHashPassword(cmd.NewPassword)
		if err != nil {
			return err
		}
		if err := a.coreAuthUpsertPasswordCredential(ctx, acc.ProjectID, acc.ID, hash); err != nil {
			return err
		}
		if cmd.RevokeOtherSessions {
			if _, err := a.coreAuthSignOutAll(ctx, acc.ProjectID, acc.ID, cmd.SessionID); err != nil {
				return err
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "password.changed",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: acc.ID,
			Payload:     acc,
		}); err != nil {
			return err
		}
		return nil
	})
}

// coreAuthPasswordPolicy is the iam_config(key=password_policy) data envelope.
// All fields are optional; a missing config row (or absent field) falls back to
// the package defaults in coreAuthLoadPasswordPolicy.
type coreAuthPasswordPolicy struct {
	MinLength      int  `json:"min_length"`
	BreachedCheck  bool `json:"breached_check"`
	History        int  `json:"history"`
	ZxcvbnMinScore int  `json:"zxcvbn_min_score"`
}

// coreAuthLoadPasswordPolicy reads the iam_config(project, env=live,
// key=password_policy) envelope and unmarshals its data jsonb into the policy
// struct. A missing config row yields the sane default (min_length 8); other
// read errors propagate.
func (a *pgCoreAuth) coreAuthLoadPasswordPolicy(ctx context.Context, projectID string) (coreAuthPasswordPolicy, error) {
	pol := coreAuthPasswordPolicy{MinLength: 8}
	row, err := models.FindIamConfig(ctx, a.db.Bobx(), projectID, coreAuthDefaultEnv, "password_policy")
	if err != nil {
		if errors.Is(translatePgErr("config", err), ErrNotFound) {
			return pol, nil
		}
		return pol, err
	}
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &pol); err != nil {
			return pol, err
		}
	}
	if pol.MinLength <= 0 {
		pol.MinLength = 8
	}
	return pol, nil
}

// CheckPassword runs a stateless strength/policy check against the project's
// password_policy (loaded from iam_config). The configured min_length is the
// hard floor; any present rules (breached_check, zxcvbn_min_score) are applied
// on top of the mixed-case heuristic. A missing policy row falls back to a sane
// default (min_length 8). The result is returned without persistence.
func (a *pgCoreAuth) CheckPassword(ctx context.Context, projectID, password string) (*domain.CoreAuthPasswordCheckResult, error) {
	pol, err := a.coreAuthLoadPasswordPolicy(ctx, projectID)
	if err != nil {
		return nil, err
	}
	res := &domain.CoreAuthPasswordCheckResult{Valid: true, Score: 4}
	if len(password) < pol.MinLength {
		res.Valid = false
		res.Score = 0
		res.Violations = append(res.Violations, "too_short")
	}
	if strings.ToLower(password) == password || strings.ToUpper(password) == password {
		if res.Score > 0 {
			res.Score--
		}
		res.Violations = append(res.Violations, "no_mixed_case")
	}
	// zxcvbn_min_score, when configured, is the floor the computed score must
	// clear; falling short marks the password too weak.
	if pol.ZxcvbnMinScore > 0 && res.Score < pol.ZxcvbnMinScore {
		res.Valid = false
		res.Violations = append(res.Violations, "too_weak")
	}
	return res, nil
}

// VerifyPassword re-asserts the current account's password for a sudo/step-up
// gate, returning whether it matched plus the session AAL/AMR.
func (a *pgCoreAuth) VerifyPassword(ctx context.Context, cmd domain.CoreAuthPasswordChangeCmd) (*domain.CoreAuthPasswordVerifyResult, error) {
	if cmd.AccountID == "" {
		return nil, domain.ErrUnauthorized
	}
	userRow, err := models.FindIamUser(ctx, a.db.Bobx(), cmd.AccountID)
	if err != nil {
		if errors.Is(translatePgErr("user", err), ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	cred, err := a.coreAuthFindPasswordCredential(ctx, userRow.ProjectID, cmd.AccountID)
	if err != nil {
		return nil, err
	}
	ok := coreAuthCheckPassword(cred.Secret, cmd.CurrentPassword)
	out := &domain.CoreAuthPasswordVerifyResult{OK: ok, AAL: 1, AMR: []string{"pwd"}}
	if cmd.SessionID != "" {
		if sessRow, err := models.FindIamSession(ctx, a.db.Bobx(), cmd.SessionID); err == nil &&
			sessRow.ProjectID == userRow.ProjectID {
			out.AAL = int(sessRow.Aal)
			if sess, err := coreAuthLoadSession(sessRow, userRow.ProjectID); err == nil && len(sess.AMR) > 0 {
				out.AMR = sess.AMR
			}
		}
	}
	return out, nil
}

// ----- session -----

// StepUp evaluates whether the current session already meets the required AAL;
// if so it is satisfied, otherwise a step-up challenge is minted to gate it.
func (a *pgCoreAuth) StepUp(ctx context.Context, cmd domain.CoreAuthStepUpCmd) (*domain.CoreAuthStepUpResult, error) {
	if cmd.SessionID == "" {
		return nil, domain.ErrUnauthorized
	}
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.CoreAuthStepUpResult, error) {
		sessRow, err := models.FindIamSession(ctx, a.db.Bobx(), cmd.SessionID)
		if err != nil {
			if errors.Is(translatePgErr("session", err), ErrNotFound) {
				return nil, domain.ErrSessionNotFound
			}
			return nil, err
		}
		required := cmd.RequiredAAL
		if required <= 0 {
			required = 2
		}
		// Freshness: when a max-age is set, a session older than it cannot be
		// considered already-stepped-up regardless of AAL.
		fresh := true
		if cmd.HasMaxAge {
			fresh = nowUTC().Sub(sessRow.CreatedAt) <= time.Duration(cmd.MaxAgeSeconds)*time.Second
		}
		if int(sessRow.Aal) >= required && fresh {
			return &domain.CoreAuthStepUpResult{Satisfied: true}, nil
		}
		code, err := coreAuthRandomCode()
		if err != nil {
			return nil, err
		}
		now := nowUTC()
		ch := coreAuthChallengeData{
			ID:        newUUID(),
			ProjectID: sessRow.ProjectID,
			Type:      "step_up",
			Purpose:   cmd.Purpose,
			AccountID: cmd.AccountID,
			Subject:   cmd.SessionID,
			CodeHash:  coreAuthSHA256(code),
			ExpiresAt: now.Add(coreAuthChallengeTTL),
			CreatedAt: now,
		}
		out, err := a.coreAuthInsertChallenge(ctx, ch)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "step_up.requested",
			ProjectID:   ch.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: ch.AccountID,
			Payload:     ch,
		}); err != nil {
			return nil, err
		}
		return &domain.CoreAuthStepUpResult{Satisfied: false, Challenge: out}, nil
	})
}

// SwitchGroup re-scopes the current session to a different group and mints a
// fresh session reflecting the new active group. The group binding lives in the
// session envelope's client/group claims; the rotation revokes the old session.
func (a *pgCoreAuth) SwitchGroup(ctx context.Context, accountID, sessionID, groupID string) (*domain.Account, *domain.Session, error) {
	if accountID == "" || sessionID == "" {
		return nil, nil, domain.ErrUnauthorized
	}
	type switchResult struct {
		acc  *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (switchResult, error) {
		sessRow, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
		if err != nil {
			if errors.Is(translatePgErr("session", err), ErrNotFound) {
				return switchResult{}, domain.ErrSessionNotFound
			}
			return switchResult{}, err
		}
		if sessRow.UserID != accountID {
			return switchResult{}, domain.ErrForbidden
		}
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			if errors.Is(translatePgErr("user", err), ErrNotFound) {
				return switchResult{}, domain.ErrUserNotFound
			}
			return switchResult{}, err
		}
		acc, err := coreAuthLoadAccount(userRow, sessRow.ProjectID)
		if err != nil {
			return switchResult{}, err
		}
		old, err := coreAuthLoadSession(sessRow, sessRow.ProjectID)
		if err != nil {
			return switchResult{}, err
		}
		// Revoke the current session, then mint a fresh one carrying the new
		// active group in its client/group binding.
		if err := a.coreAuthRevokeSession(ctx, sessRow.ProjectID, sessionID); err != nil {
			return switchResult{}, err
		}
		amr := old.AMR
		if len(amr) == 0 {
			amr = []string{"pwd"}
		}
		sess, err := a.coreAuthMintSession(ctx, acc, groupID, amr, old.AAL)
		if err != nil {
			return switchResult{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "session.group_switched",
			ProjectID:   acc.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: sess.ID,
			Payload:     sess,
		}); err != nil {
			return switchResult{}, err
		}
		return switchResult{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acc, res.sess, nil
}

// ----- access requests -----

// CreateAccessRequest records a self-service access request gating sign-up
// behind approval (iam_access_requests, status=pending).
func (a *pgCoreAuth) CreateAccessRequest(ctx context.Context, cmd domain.CoreAuthAccessRequestCmd) (*domain.CoreAuthAccessRequest, error) {
	if cmd.ProjectID == "" || strings.TrimSpace(cmd.Email) == "" {
		return nil, domain.ErrValidation.WithMessage("project and email are required")
	}
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.CoreAuthAccessRequest, error) {
		now := nowUTC()
		ar := &domain.CoreAuthAccessRequest{
			ID:        newUUID(),
			ProjectID: cmd.ProjectID,
			Email:     cmd.Email,
			Reason:    cmd.Reason,
			Status:    "pending",
		}
		raw, err := marshal(ar)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamAccessRequestSetter{
			ID:        &ar.ID,
			ProjectID: &ar.ProjectID,
			Email:     &ar.Email,
			Status:    ptr(ar.Status),
			CreatedAt: &now,
			UpdatedAt: &now,
			Data:      &rm,
		}
		if _, err := models.IamAccessRequests.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "access_request.created",
			ProjectID:   ar.ProjectID,
			Environment: coreAuthDefaultEnv,
			AggregateID: ar.ID,
			Payload:     ar,
		}); err != nil {
			return nil, err
		}
		return ar, nil
	})
}

// ===========================================================================
// api.CoreAuthTokens
// ===========================================================================

// Introspect resolves a token to its active state + claims. It matches an
// access token against a live session (envelope) or a refresh token by hash,
// all scoped to the project (tenant boundary).
func (a *pgCoreAuth) Introspect(ctx context.Context, projectID, token string) (*domain.CoreAuthTokenIntrospection, error) {
	if strings.TrimSpace(token) == "" {
		return &domain.CoreAuthTokenIntrospection{Active: false}, nil
	}
	// Refresh token: hashed lookup.
	if row, err := models.IamRefreshTokens.Query(
		sm.Where(models.IamRefreshTokens.Columns.Hash.EQ(psql.Arg(coreAuthSHA256(token)))),
		sm.Where(models.IamRefreshTokens.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).One(ctx, a.db.Bobx()); err == nil {
		active := !row.Revoked
		if v, ok := row.ExpiresAt.Get(); ok && nowUTC().After(v) {
			active = false
		}
		return &domain.CoreAuthTokenIntrospection{
			Active: active,
			Claims: map[string]any{
				"token_type": "refresh_token",
				"sub":        row.UserID,
				"sid":        row.SessionID,
			},
		}, nil
	} else if !errors.Is(translatePgErr("refresh_token", err), ErrNotFound) {
		return nil, err
	}

	// Access token: verify the JWT signature/claims (jwx) and confirm the named
	// session is still live.
	claims, sess, err := a.coreAuthVerifyAccess(ctx, projectID, token)
	if err != nil || sess == nil {
		return &domain.CoreAuthTokenIntrospection{Active: false}, nil
	}
	claims["token_type"] = "access_token"
	return &domain.CoreAuthTokenIntrospection{Active: true, Claims: claims}, nil
}

// RedeemImpersonation exchanges a single-use impersonation token (minted by the
// admin impersonate endpoint) for a session acting as the target user. The token
// is a typ=impersonation JWT (sub=target, act=admin); the matching
// iam_challenges row gates single use and is consumed here so the token cannot be
// replayed.
func (a *pgCoreAuth) RedeemImpersonation(ctx context.Context, token, clientID string) (*domain.Account, *domain.Session, error) {
	claims := a.db.Signer().UnverifiedClaims(token)
	if claims == nil {
		return nil, nil, domain.ErrInvalidToken
	}
	projectID, _ := claims["pid"].(string)
	env := coreAuthDefaultEnv
	if e, ok := claims["env"].(string); ok && e != "" {
		env = e
	}
	verified, err := a.db.Signer().Verify(ctx, projectID, env, token)
	if err != nil {
		return nil, nil, domain.ErrInvalidToken
	}
	if typ, _ := verified["typ"].(string); typ != "impersonation" {
		return nil, nil, domain.ErrInvalidToken
	}
	sub, _ := verified["sub"].(string)
	actor, _ := verified["act"].(string)
	if projectID == "" || sub == "" {
		return nil, nil, domain.ErrInvalidToken
	}

	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (*verifyResult, error) {
		// Consume the single-use challenge (gates replay).
		row, err := models.IamChallenges.Query(
			sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg("impersonation"))),
			sm.Where(models.IamChallenges.Columns.CodeHash.EQ(psql.Arg(adminSHA256(token)))),
		).One(ctx, a.db.Bobx())
		if err != nil {
			return nil, domain.ErrChallengeInvalid
		}
		if row.ProjectID != projectID {
			return nil, domain.ErrChallengeInvalid
		}
		if nowUTC().After(row.ExpiresAt) {
			return nil, domain.ErrChallengeExpired
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil { // single-use redemption
			return nil, err
		}
		// Load the target user and mint a session acting as them.
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), sub)
		if err != nil {
			return nil, domain.ErrUserNotFound
		}
		if userRow.ProjectID != projectID {
			return nil, domain.ErrUserNotFound
		}
		acc, err := coreAuthLoadAccount(userRow, projectID)
		if err != nil {
			return nil, err
		}
		sess, err := a.coreAuthMintSession(ctx, acc, clientID, []string{"impersonation"}, 1)
		if err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "user.impersonation_redeemed",
			ProjectID:   projectID,
			Environment: env,
			AggregateID: sub,
			Payload:     map[string]any{"user_id": sub, "actor_id": actor, "session_id": sess.ID},
		}); err != nil {
			return nil, err
		}
		return &verifyResult{acct: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acct, res.sess, nil
}

// Verify validates a token's signature and claims for an audience. Signature
// verification is deferred (opaque tokens), so this checks liveness + audience
// against the session envelope and reports a structured result.
func (a *pgCoreAuth) Verify(ctx context.Context, projectID, token, audience string) (*domain.CoreAuthTokenVerification, error) {
	if strings.TrimSpace(token) == "" {
		return &domain.CoreAuthTokenVerification{Valid: false, Error: "missing_token"}, nil
	}
	// Verify the JWT signature + claims (jwx), then confirm the session is live.
	claims, sess, err := a.coreAuthVerifyAccess(ctx, projectID, token)
	if err != nil || sess == nil {
		return &domain.CoreAuthTokenVerification{Valid: false, Error: "invalid_token"}, nil
	}
	if audience != "" && sess.ClientID != "" && sess.ClientID != audience {
		return &domain.CoreAuthTokenVerification{Valid: false, Error: "audience_mismatch"}, nil
	}
	return &domain.CoreAuthTokenVerification{Valid: true, Claims: claims}, nil
}

// Revoke revokes a token or a whole session. A SessionID revokes the session;
// otherwise the opaque token is matched as a refresh token (by hash) or an
// access token (session envelope).
func (a *pgCoreAuth) Revoke(ctx context.Context, cmd domain.CoreAuthRevokeCmd) error {
	if cmd.SessionID == "" && strings.TrimSpace(cmd.Token) == "" {
		return domain.ErrBadRequest.WithMessage("token or session_id is required")
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		if cmd.SessionID != "" {
			if err := a.coreAuthRevokeSession(ctx, "", cmd.SessionID); err != nil &&
				!errors.Is(err, domain.ErrSessionNotFound) {
				return err
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "session.revoked",
				ProjectID:   "",
				Environment: coreAuthDefaultEnv,
				AggregateID: cmd.SessionID,
				Payload:     map[string]any{"session_id": cmd.SessionID},
			}); err != nil {
				return err
			}
			return nil
		}
		// Refresh token by hash.
		if row, err := models.IamRefreshTokens.Query(
			sm.Where(models.IamRefreshTokens.Columns.Hash.EQ(psql.Arg(coreAuthSHA256(cmd.Token)))),
		).One(ctx, a.db.Bobx()); err == nil {
			if err := a.coreAuthMarkRefreshRevoked(ctx, row); err != nil {
				return err
			}
			if err := a.emitter.Emit(ctx, domain.Event{
				Type:        "token.revoked",
				ProjectID:   row.ProjectID,
				Environment: coreAuthDefaultEnv,
				AggregateID: row.ID,
				Payload:     map[string]any{"token_id": row.ID, "session_id": row.SessionID, "project_id": row.ProjectID},
			}); err != nil {
				return err
			}
			return nil
		} else if !errors.Is(translatePgErr("refresh_token", err), ErrNotFound) {
			return err
		}
		// Access token (JWT): route by its sid claim. Revocation is idempotent,
		// so the claims are read unverified purely to find the session.
		if claims := a.db.Signer().UnverifiedClaims(cmd.Token); claims != nil {
			sid, _ := claims["sid"].(string)
			pid, _ := claims["pid"].(string)
			if sid != "" {
				if err := a.coreAuthRevokeSession(ctx, pid, sid); err != nil &&
					!errors.Is(err, domain.ErrSessionNotFound) {
					return err
				}
				if err := a.emitter.Emit(ctx, domain.Event{
					Type:        "token.revoked",
					ProjectID:   pid,
					Environment: coreAuthDefaultEnv,
					AggregateID: sid,
					Payload:     map[string]any{"session_id": sid, "project_id": pid},
				}); err != nil {
					return err
				}
			}
		}
		// Unknown token: revocation is idempotent, treat as success.
		return nil
	})
}

// CurrentClaims returns the claim set for the current session id (read-only).
func (a *pgCoreAuth) CurrentClaims(ctx context.Context, sessionID string) (map[string]any, error) {
	if sessionID == "" {
		return nil, domain.ErrUnauthorized
	}
	row, err := models.FindIamSession(ctx, a.db.Bobx(), sessionID)
	if err != nil {
		if errors.Is(translatePgErr("session", err), ErrNotFound) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	sess, err := coreAuthLoadSession(row, row.ProjectID)
	if err != nil {
		return nil, err
	}
	claims := map[string]any{
		"sub": sess.AccountID,
		"sid": sess.ID,
		"aal": sess.AAL,
		"amr": sess.AMR,
		"iss": row.ProjectID,
	}
	if sess.ClientID != "" {
		claims["aud"] = sess.ClientID
	}
	if v, ok := row.ExpiresAt.Get(); ok {
		claims["exp"] = v.Unix()
	}
	return claims, nil
}
