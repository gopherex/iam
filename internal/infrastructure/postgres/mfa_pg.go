package postgres

// Postgres adapter for the MFA aggregate. Backs api.MFAAccounts over three
// envelopes:
//
//   - iam_factors        — enrolled authenticator factors (totp/sms/email/webauthn);
//                          the shared TOTP/webauthn secret lives in the `secret`
//                          column, the domain.Factor aggregate in `data`.
//   - iam_recovery_codes — single-use recovery codes, stored only as sha256 hashes.
//   - iam_challenges      — pending verification challenges; `subject` carries the
//                          owning account id, `data` carries flow material
//                          (factor id, delivery code hash, webauthn options).
//
// Persistence follows the gold pattern (reference.go): bob query builders,
// every mutation wrapped in withTx/withTxRet, aggregate marshalled into `data`,
// envelope columns used only for lookups, and every query bounded by project_id.
//
// Secrets/codes/tokens are minted with crypto/rand and only ever persisted as
// sha256 hashes (recovery codes, opaque flow tokens) or as the raw TOTP secret
// the authenticator app provisioned. Plaintext recovery codes are returned to
// the caller exactly once at generation time and never stored.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// mfaTOTPIssuer is the issuer label embedded in provisioned otpauth URLs and
// shown by authenticator apps.
const mfaTOTPIssuer = "gopherex-iam"

// mfaChallengeTTL bounds how long a freshly issued challenge stays verifiable.
const mfaChallengeTTL = 5 * time.Minute

// mfaRecoveryCodeCount is the size of a freshly minted recovery-code batch.
const mfaRecoveryCodeCount = 10

// mfaMaxFactorsPerAccount is the maximum number of MFA factors an account may
// have enrolled (M-14: prevents unbounded factor creation).
const mfaMaxFactorsPerAccount = 10

// pgMFAAccounts is the Postgres-backed api.MFAAccounts adapter.
type pgMFAAccounts struct {
	db      *DB
	emitter Emitter
}

// NewPgMFAAccounts builds the MFA aggregate adapter over a *DB.
func NewPgMFAAccounts(db *DB, emitter Emitter) *pgMFAAccounts {
	return &pgMFAAccounts{db: db, emitter: emitter}
}

var _ api.MFAAccounts = (*pgMFAAccounts)(nil)

// ===== challenge data envelope =====

// mfaChallengeData is the jsonb payload carried in iam_challenges.data. It holds
// the flow material the verify step needs but that has no dedicated column.
//
// Session carries the opaque marshalled go-webauthn SessionData (challenge bytes,
// RP id, user verification) minted by BeginRegistration during a WebAuthn-factor
// enrollment; EnrollWebAuthnVerify replays it via w.CreateCredential to verify
// the attestation. It is the MFA-flow counterpart of domain.WebAuthnCeremonyData
// (we keep the session alongside the existing mfa flow fields rather than swap to
// a second envelope so the shared mfaInsertChallenge/mfaConsumeChallenge helpers
// still apply).
type mfaChallengeData struct {
	FactorID      string         `json:"factor_id,omitempty"`
	FlowTokenHash string         `json:"flow_token_hash,omitempty"`
	PublicKey     map[string]any `json:"public_key,omitempty"`
	Session       []byte         `json:"session,omitempty"`
}

// ===== helpers (mfa-prefixed) =====

// mfaSha256Hex returns the hex sha256 of s — the only form codes/tokens are stored in.
func mfaSha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// mfaRandomBytes draws n cryptographically-random bytes.
func mfaRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// mfaGenerateTOTPKey provisions a fresh TOTP key via the RFC 6238 library. The
// returned *otp.Key exposes the base32 shared secret (key.Secret(), stored in
// iam_factors.secret) and the otpauth:// provisioning URL (key.URL(), surfaced
// to the caller for QR rendering). accountName is the per-user label shown in
// the authenticator app (the account's primary email when available).
func mfaGenerateTOTPKey(accountName string) (*otp.Key, error) {
	if accountName == "" {
		accountName = "user"
	}
	return totp.Generate(totp.GenerateOpts{
		Issuer:      mfaTOTPIssuer,
		AccountName: accountName,
	})
}

// mfaNewOpaqueToken mints a hex opaque token (e.g. a flow / delivery code).
func mfaNewOpaqueToken(nbytes int) (string, error) {
	b, err := mfaRandomBytes(nbytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// mfaFactorFromRow rebuilds a domain.Factor from its envelope row.
func mfaFactorFromRow(row *models.IamFactor) (domain.Factor, error) {
	var f domain.Factor
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &f); err != nil {
			return domain.Factor{}, err
		}
	}
	// Envelope columns are authoritative for the lookup fields.
	f.ID = row.ID
	if f.Type == "" {
		f.Type = row.Type
	}
	if f.Status == "" {
		f.Status = row.Status
	}
	return f, nil
}

// mfaLoadAccount loads the owning account aggregate, enforcing the tenant boundary.
func (a *pgMFAAccounts) mfaLoadAccount(ctx context.Context, projectID, accountID string) (*domain.Account, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	if row.ProjectID != projectID {
		return nil, domain.ErrUserNotFound
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

// mfaFindFactor loads a factor by id within an account, enforcing the boundary.
func (a *pgMFAAccounts) mfaFindFactor(ctx context.Context, accountID, factorID string) (*models.IamFactor, error) {
	row, err := models.FindIamFactor(ctx, a.db.Bobx(), factorID)
	if err != nil {
		return nil, translatePgErr("factor", err)
	}
	if row.UserID != accountID {
		return nil, domain.ErrNotFound
	}
	return row, nil
}

// mfaCountFactors returns the number of factors enrolled for an account.
func (a *pgMFAAccounts) mfaCountFactors(ctx context.Context, accountID string) (int, error) {
	rows, err := models.IamFactors.Query(
		sm.Where(models.IamFactors.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// ===== WebAuthn-factor ceremony helpers (mfa-prefixed) =====
//
// A WebAuthn MFA factor is enrolled through the same attestation ceremony the
// passkey adapter uses (webauthn_pg.go): BeginRegistration mints the publicKey
// creation options + a SessionData snapshot we persist in the challenge; the
// verify step replays that SessionData via go-webauthn's CreateCredential to
// validate the attestation (challenge, origin, RP id, attestation statement)
// before the factor is activated. We never hand-roll the COSE/CBOR crypto.

// mfaRPConfigFor builds the per-project go-webauthn Relying Party instance for a
// WebAuthn-factor ceremony. It reuses webauthnRPFromProject (the shared per-project
// RP derivation in webauthn_pg.go) so the RP id + permitted origins match the
// passkey adapter exactly.
func (a *pgMFAAccounts) mfaRPConfigFor(ctx context.Context, projectID string) (*gowebauthn.WebAuthn, error) {
	row, err := models.FindIamProject(ctx, a.db.Bobx(), projectID)
	if err != nil {
		return nil, translatePgErr("project", err)
	}
	rpID, displayName, origins := webauthnRPFromProject(row)
	w, err := gowebauthn.New(&gowebauthn.Config{
		RPID:          rpID,
		RPDisplayName: displayName,
		RPOrigins:     origins,
	})
	if err != nil {
		return nil, domain.ErrProviderError
	}
	return w, nil
}

// mfaLoadWebauthnUser reads the account aggregate and adapts it onto the
// go-webauthn User interface. A WebAuthn MFA factor is independent of the passkey
// credentials, so the ceremony starts the user with no existing credentials (the
// library only needs the user handle, name and display name for registration).
func (a *pgMFAAccounts) mfaLoadWebauthnUser(ctx context.Context, accountID string) (*webauthnUser, error) {
	userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	var acct domain.Account
	if err := unmarshal(userRow.Data, &acct); err != nil {
		return nil, err
	}
	acct.ID = userRow.ID
	acct.ProjectID = userRow.ProjectID
	return &webauthnUser{account: &acct, creds: nil}, nil
}

// ===== api.MFAAccounts =====

// ListFactors returns every factor enrolled for the account.
func (a *pgMFAAccounts) ListFactors(ctx context.Context, accountID string) ([]domain.Factor, error) {
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
		out = append(out, f)
	}
	return out, nil
}

// EnrollTOTP provisions a pending TOTP factor and returns it; the activation
// happens later via VerifyTOTP once the user proves possession of the secret.
func (a *pgMFAAccounts) EnrollTOTP(ctx context.Context, accountID string) (*domain.Factor, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Factor, error) {
		// The owning account fixes the tenant and supplies the label the
		// authenticator app shows for this enrollment.
		acc, err := a.mfaResolveAccount(ctx, accountID)
		if err != nil {
			return nil, err
		}
		count, err := a.mfaCountFactors(ctx, accountID)
		if err != nil {
			return nil, err
		}
		if count >= mfaMaxFactorsPerAccount {
			return nil, domain.ErrConflict.WithMessage("maximum number of MFA factors reached")
		}
		// Provision the shared secret + otpauth URL via the RFC 6238 library;
		// key.Secret() is the base32 secret stored in iam_factors.secret and
		// key.URL() is the otpauth:// URL the caller renders as a QR code.
		key, err := mfaGenerateTOTPKey(acc.PrimaryEmail)
		if err != nil {
			return nil, err
		}
		f := domain.Factor{
			ID:         newUUID(),
			Type:       "totp",
			Status:     "pending",
			Hint:       "authenticator app",
			OTPAuthURL: key.URL(),
		}
		if err := a.mfaInsertFactorFor(ctx, acc.ProjectID, accountID, &f, key.Secret()); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.factor.enrolled",
			ProjectID:   acc.ProjectID,
			Environment: "",
			AggregateID: f.ID,
			Payload:     f,
		}); err != nil {
			return nil, err
		}
		return &f, nil
	})
}

// Challenge issues a verification challenge for an existing factor.
func (a *pgMFAAccounts) Challenge(ctx context.Context, accountID, factorID string) (*domain.Challenge, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Challenge, error) {
		projectID, err := a.mfaResolveProject(ctx, accountID)
		if err != nil {
			return nil, err
		}
		factorRow, err := a.mfaFindFactor(ctx, accountID, factorID)
		if err != nil {
			return nil, err
		}
		factor, err := mfaFactorFromRow(factorRow)
		if err != nil {
			return nil, err
		}
		ch := domain.Challenge{
			ID:        newUUID(),
			Type:      factor.Type,
			ExpiresAt: nowUTC().Add(mfaChallengeTTL),
		}
		data := mfaChallengeData{FactorID: factor.ID}
		// Delivery factors (email/sms) carry a one-time code sent out of band;
		// Verify matches its sha256 against FlowTokenHash. TOTP/WebAuthn need no
		// delivery — the code comes from the authenticator.
		deliver := factor.Type == "email" || factor.Type == "sms"
		var code string
		if deliver {
			code, err = mfaNewOpaqueToken(4) // 8 hex chars
			if err != nil {
				return nil, err
			}
			data.FlowTokenHash = mfaSha256Hex(code)
		}
		if err := a.mfaInsertChallenge(ctx, projectID, accountID, &ch, data); err != nil {
			return nil, err
		}
		payload := map[string]any{
			"channel":      factor.Type,
			"factor_id":    factor.ID,
			"challenge_id": ch.ID,
		}
		if deliver {
			payload["code"] = code
			payload["to"] = factor.Hint
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.challenge.created",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: ch.ID,
			Payload:     payload,
		}); err != nil {
			return nil, err
		}
		return &ch, nil
	})
}

// mfaAccountFromFlow resolves the account behind a login flow_token (the id of
// the step-up challenge minted at password sign-in). It enforces the tenant
// boundary and rejects consumed/expired tokens, so a second factor can only be
// completed against a live, password-backed flow.
func (a *pgMFAAccounts) mfaAccountFromFlow(ctx context.Context, projectID, flowToken string) (string, error) {
	if flowToken == "" {
		return "", domain.ErrChallengeInvalid
	}
	row, err := models.FindIamChallenge(ctx, a.db.Bobx(), flowToken)
	if err != nil {
		return "", domain.ErrChallengeInvalid
	}
	if row.ProjectID != projectID || row.Consumed {
		return "", domain.ErrChallengeInvalid
	}
	if nowUTC().After(row.ExpiresAt) {
		return "", domain.ErrChallengeExpired
	}
	accountID := row.Subject.GetOrZero()
	if accountID == "" {
		return "", domain.ErrChallengeInvalid
	}
	return accountID, nil
}

// ChallengeWithFlow issues a fresh challenge for a different factor mid-login,
// identifying the account from the flow_token rather than a session principal
// (the endpoint is public — the user is not yet authenticated). When factorID is
// empty the account's primary factor is chosen.
func (a *pgMFAAccounts) ChallengeWithFlow(ctx context.Context, projectID, flowToken, factorID string) (*domain.Challenge, error) {
	accountID, err := a.mfaAccountFromFlow(ctx, projectID, flowToken)
	if err != nil {
		return nil, err
	}
	if factorID == "" {
		factors, err := a.ListFactors(ctx, accountID)
		if err != nil {
			return nil, err
		}
		factorID = mfaPrimaryFactorID(factors)
		if factorID == "" {
			return nil, domain.ErrMFAInvalid
		}
	}
	return a.Challenge(ctx, accountID, factorID)
}

// mfaPrimaryFactorID prefers factors needing no out-of-band delivery.
func mfaPrimaryFactorID(factors []domain.Factor) string {
	for _, f := range factors {
		if f.Status == "active" && (f.Type == "totp" || f.Type == "webauthn") {
			return f.ID
		}
	}
	for _, f := range factors {
		if f.Status == "active" {
			return f.ID
		}
	}
	return ""
}

// Verify consumes a challenge and, on success, returns the authenticated account
// plus a fresh session. TOTP codes are checked with the RFC 6238 library.
func (a *pgMFAAccounts) Verify(ctx context.Context, challengeID, code string) (*domain.Account, *domain.Session, error) {
	type result struct {
		acc  *domain.Account
		sess *domain.Session
	}
	out, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		row, err := models.FindIamChallenge(ctx, a.db.Bobx(), challengeID)
		if err != nil {
			return result{}, translatePgErr("challenge", err)
		}
		if row.Consumed {
			return result{}, domain.ErrChallengeInvalid
		}
		if nowUTC().After(row.ExpiresAt) {
			return result{}, domain.ErrChallengeExpired
		}
		accountID := row.Subject.GetOrZero()
		var data mfaChallengeData
		if len(row.Data) > 0 {
			if err := unmarshal(row.Data, &data); err != nil {
				return result{}, err
			}
		}
		// Code verification:
		//   - delivery factors (email/sms) compare the sha256 of the supplied code
		//     against the stored code hash;
		//   - TOTP validates the supplied code against the factor's shared
		//     secret with the RFC 6238 library.
		switch row.Type {
		case "email", "sms":
			if data.FlowTokenHash == "" || subtle.ConstantTimeCompare([]byte(mfaSha256Hex(code)), []byte(data.FlowTokenHash)) != 1 {
				return result{}, domain.ErrMFAInvalid
			}
		default:
			// TOTP: load the factor's shared secret and check the supplied code
			// with the RFC 6238 library. An invalid/expired code is ErrMFAInvalid.
			if data.FactorID == "" {
				return result{}, domain.ErrMFAInvalid
			}
			factor, err := a.mfaFindFactor(ctx, accountID, data.FactorID)
			if err != nil {
				return result{}, err
			}
			if factor.Status != "active" {
				return result{}, domain.ErrMFAInvalid
			}
			secret, err := a.db.Cipher.Decrypt(factor.Secret)
			if err != nil {
				return result{}, domain.ErrMFAInvalid
			}
			if !totp.Validate(code, secret) {
				return result{}, domain.ErrMFAInvalid
			}
		}
		if err := a.mfaConsumeChallenge(ctx, row); err != nil {
			return result{}, err
		}
		acc, err := a.mfaLoadAccount(ctx, row.ProjectID, accountID)
		if err != nil {
			return result{}, err
		}
		sess, err := a.mfaMintSession(ctx, acc)
		if err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.challenge.verified",
			ProjectID:   row.ProjectID,
			Environment: mfaDefaultEnv,
			AggregateID: challengeID,
			Payload:     sess,
		}); err != nil {
			return result{}, err
		}
		return result{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out.acc, out.sess, nil
}

// GenerateRecoveryCodes mints a fresh batch of single-use recovery codes,
// replacing any prior unused ones. Only the sha256 hashes are persisted; the
// plaintext codes are returned to the caller exactly once.
func (a *pgMFAAccounts) GenerateRecoveryCodes(ctx context.Context, accountID string) ([]string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) ([]string, error) {
		projectID, err := a.mfaResolveProject(ctx, accountID)
		if err != nil {
			return nil, err
		}
		// Invalidate the previous batch so only the newest codes work.
		if _, err := models.IamRecoveryCodes.Delete(
			dm.Where(models.IamRecoveryCodes.Columns.UserID.EQ(psql.Arg(accountID))),
			dm.Where(models.IamRecoveryCodes.Columns.ProjectID.EQ(psql.Arg(projectID))),
		).Exec(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		codes := make([]string, 0, mfaRecoveryCodeCount)
		for i := 0; i < mfaRecoveryCodeCount; i++ {
			code, err := mfaNewOpaqueToken(16) // 32 hex chars, 128 bits
			if err != nil {
				return nil, err
			}
			codes = append(codes, code)
			setter := &models.IamRecoveryCodeSetter{
				ID:        ptr(newUUID()),
				ProjectID: ptr(projectID),
				UserID:    ptr(accountID),
				Hash:      ptr(mfaSha256Hex(code)),
				Used:      ptr(false),
				CreatedAt: ptr(nowUTC()),
			}
			if _, err := models.IamRecoveryCodes.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
				return nil, err
			}
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.recovery_codes.generated",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: accountID,
			Payload:     map[string]any{"account_id": accountID, "project_id": projectID, "count": len(codes)},
		}); err != nil {
			return nil, err
		}
		return codes, nil
	})
}

// RemoveFactor deletes a factor (and is a no-op-safe boundary check otherwise).
func (a *pgMFAAccounts) RemoveFactor(ctx context.Context, accountID, factorID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, err := a.mfaFindFactor(ctx, accountID, factorID)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.factor.removed",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: row.ID,
			Payload:     map[string]any{"id": row.ID, "project_id": row.ProjectID},
		}); err != nil {
			return err
		}
		return nil
	})
}

// EnrollEmail enrolls an email factor and issues a delivery challenge carrying
// the sha256 of a freshly minted one-time code.
func (a *pgMFAAccounts) EnrollEmail(ctx context.Context, cmd domain.MFAEmailEnrollCmd) (*domain.Factor, *domain.Challenge, error) {
	if err := domain.ValidateEmail(cmd.Email); err != nil {
		return nil, nil, err
	}
	return a.mfaEnrollDelivery(ctx, cmd.AccountID, "email", cmd.Email)
}

// EnrollSMS enrolls an SMS factor and issues a delivery challenge carrying the
// sha256 of a freshly minted one-time code.
func (a *pgMFAAccounts) EnrollSMS(ctx context.Context, cmd domain.MFASmsEnrollCmd) (*domain.Factor, *domain.Challenge, error) {
	if err := domain.ValidatePhone(cmd.Phone); err != nil {
		return nil, nil, err
	}
	return a.mfaEnrollDelivery(ctx, cmd.AccountID, "sms", cmd.Phone)
}

// VerifyTOTP activates a pending TOTP factor after the code check passes.
func (a *pgMFAAccounts) VerifyTOTP(ctx context.Context, cmd domain.MFATotpVerifyCmd) (*domain.Factor, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Factor, error) {
		row, err := a.mfaFindFactor(ctx, cmd.AccountID, cmd.FactorID)
		if err != nil {
			return nil, err
		}
		// Prove possession of the secret before activating: check the code
		// against the stored shared secret with the RFC 6238 library.
		secret, err := a.db.Cipher.Decrypt(row.Secret)
		if err != nil {
			return nil, domain.ErrMFAInvalid
		}
		if !totp.Validate(cmd.Code, secret) {
			return nil, domain.ErrMFAInvalid
		}

		f, err := mfaFactorFromRow(row)
		if err != nil {
			return nil, err
		}
		f.Status = "active"
		raw, err := marshal(&f)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(raw)
		setter := &models.IamFactorSetter{Status: ptr("active"), Data: &rm}
		if err := row.Update(ctx, a.db.Bobx(), setter); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.factor.activated",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: f.ID,
			Payload:     f,
		}); err != nil {
			return nil, err
		}
		return &f, nil
	})
}

// VerifyRecoveryCode consumes a single-use recovery code and, on success,
// returns the authenticated account plus a fresh session.
func (a *pgMFAAccounts) VerifyRecoveryCode(ctx context.Context, cmd domain.MFARecoveryVerifyCmd) (*domain.Account, *domain.Session, error) {
	type result struct {
		acc  *domain.Account
		sess *domain.Session
	}
	out, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		// A recovery code is a second factor: it is accepted only against a live
		// login flow_token (password already verified). The account is taken from
		// the flow, never from the request, so a code cannot be redeemed standalone.
		accountID, err := a.mfaAccountFromFlow(ctx, cmd.ProjectID, cmd.FlowToken)
		if err != nil {
			return result{}, err
		}
		hash := mfaSha256Hex(cmd.Code)
		row, err := models.IamRecoveryCodes.Query(
			sm.Where(models.IamRecoveryCodes.Columns.ProjectID.EQ(psql.Arg(cmd.ProjectID))),
			sm.Where(models.IamRecoveryCodes.Columns.UserID.EQ(psql.Arg(accountID))),
			sm.Where(models.IamRecoveryCodes.Columns.Hash.EQ(psql.Arg(hash))),
			sm.Where(models.IamRecoveryCodes.Columns.Used.EQ(psql.Arg(false))),
		).One(ctx, a.db.Bobx())
		if err != nil {
			return result{}, domain.ErrInvalidCredentials
		}
		// Burn the code so it cannot be replayed.
		if err := row.Update(ctx, a.db.Bobx(), &models.IamRecoveryCodeSetter{Used: ptr(true)}); err != nil {
			return result{}, err
		}
		acc, err := a.mfaLoadAccount(ctx, cmd.ProjectID, row.UserID)
		if err != nil {
			return result{}, err
		}
		sess, err := a.mfaMintSession(ctx, acc)
		if err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.recovery_code.consumed",
			ProjectID:   cmd.ProjectID,
			Environment: mfaDefaultEnv,
			AggregateID: row.UserID,
			Payload:     sess,
		}); err != nil {
			return result{}, err
		}
		return result{acc: acc, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out.acc, out.sess, nil
}

// EnrollWebAuthnOptions issues a WebAuthn enrollment challenge carrying the
// publicKey creation options the authenticator needs. It runs a real go-webauthn
// attestation ceremony: BeginRegistration mints the publicKey options + the
// SessionData snapshot (challenge, RP id, user verification) we persist in the
// challenge row so EnrollWebAuthnVerify can verify the matching attestation.
func (a *pgMFAAccounts) EnrollWebAuthnOptions(ctx context.Context, cmd domain.MFAWebAuthnEnrollOptionsCmd) (*domain.Challenge, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Challenge, error) {
		projectID, err := a.mfaResolveProject(ctx, cmd.AccountID)
		if err != nil {
			return nil, err
		}

		// Build the per-project Relying Party + the user handle the library binds
		// the ceremony to.
		w, err := a.mfaRPConfigFor(ctx, projectID)
		if err != nil {
			return nil, err
		}
		user, err := a.mfaLoadWebauthnUser(ctx, cmd.AccountID)
		if err != nil {
			return nil, err
		}

		// BeginRegistration mints the publicKey creation options surfaced to the
		// authenticator + the opaque SessionData replayed on verify.
		creation, session, err := w.BeginRegistration(user)
		if err != nil {
			return nil, domain.ErrProviderError
		}
		publicKey, err := webauthnOptionsMap(creation.Response)
		if err != nil {
			return nil, err
		}
		sessionRaw, err := json.Marshal(session)
		if err != nil {
			return nil, err
		}

		ch := domain.Challenge{
			ID:        newUUID(),
			Type:      "webauthn",
			ExpiresAt: nowUTC().Add(mfaChallengeTTL),
			PublicKey: publicKey,
		}
		if err := a.mfaInsertChallenge(ctx, projectID, cmd.AccountID, &ch, mfaChallengeData{
			PublicKey: publicKey,
			Session:   sessionRaw,
		}); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.webauthn.options_issued",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: ch.ID,
			Payload:     ch,
		}); err != nil {
			return nil, err
		}
		return &ch, nil
	})
}

// EnrollWebAuthnVerify validates the attestation and activates the WebAuthn factor.
// It replays the SessionData persisted by EnrollWebAuthnOptions through go-webauthn's
// CreateCredential, which verifies the attestation object + clientDataJSON against
// the ceremony (challenge, RP id, origin, attestation statement). On success the
// challenge is consumed and the factor activated, persisting the verified library
// credential in the factor secret column so subsequent assertions can be checked.
func (a *pgMFAAccounts) EnrollWebAuthnVerify(ctx context.Context, cmd domain.MFAWebAuthnEnrollVerifyCmd) (*domain.Factor, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Factor, error) {
		row, err := models.FindIamChallenge(ctx, a.db.Bobx(), cmd.ChallengeID)
		if err != nil {
			return nil, translatePgErr("challenge", err)
		}
		if row.Subject.GetOrZero() != cmd.AccountID {
			return nil, domain.ErrChallengeInvalid
		}
		if row.Consumed {
			return nil, domain.ErrChallengeInvalid
		}
		if nowUTC().After(row.ExpiresAt) {
			return nil, domain.ErrChallengeExpired
		}

		// Rehydrate the ceremony SessionData persisted at options time.
		var data mfaChallengeData
		if len(row.Data) > 0 {
			if err := unmarshal(row.Data, &data); err != nil {
				return nil, err
			}
		}
		if len(data.Session) == 0 {
			return nil, domain.ErrChallengeInvalid
		}
		var session gowebauthn.SessionData
		if err := json.Unmarshal(data.Session, &session); err != nil {
			return nil, domain.ErrChallengeInvalid
		}

		// Rebuild the per-project Relying Party + the bound user handle.
		w, err := a.mfaRPConfigFor(ctx, row.ProjectID)
		if err != nil {
			return nil, err
		}
		user, err := a.mfaLoadWebauthnUser(ctx, cmd.AccountID)
		if err != nil {
			return nil, err
		}

		// verify with WebAuthn signing/attestation — marshal the browser credential
		// map, parse it, then validate the attestation (challenge, origin, RP id,
		// attestation statement) against the stored SessionData via go-webauthn.
		credRaw, err := json.Marshal(cmd.Credential)
		if err != nil {
			return nil, err
		}
		parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(credRaw))
		if err != nil {
			return nil, domain.ErrMFAInvalid
		}
		libCred, err := w.CreateCredential(user, session, parsed)
		if err != nil {
			return nil, domain.ErrMFAInvalid
		}

		if err := a.mfaConsumeChallenge(ctx, row); err != nil {
			return nil, err
		}

		// Persist the verified library credential in the factor secret column so the
		// authenticator material (id, COSE public key, sign count) is retained for
		// subsequent assertions; the credential id is the base64url raw id.
		libJSON, err := json.Marshal(libCred)
		if err != nil {
			return nil, err
		}
		// Label the factor with the supplied name, falling back to the base64url
		// credential id, then a generic hint.
		hint := "security key"
		if name, _ := cmd.Credential["name"].(string); name != "" {
			hint = name
		} else if id := base64.RawURLEncoding.EncodeToString(libCred.ID); id != "" {
			hint = id
		}
		f := domain.Factor{
			ID:     newUUID(),
			Type:   "webauthn",
			Status: "active",
			Hint:   hint,
		}
		if err := a.mfaInsertFactorFor(ctx, row.ProjectID, cmd.AccountID, &f, string(libJSON)); err != nil {
			return nil, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.factor.enrolled",
			ProjectID:   row.ProjectID,
			Environment: "",
			AggregateID: f.ID,
			Payload:     f,
		}); err != nil {
			return nil, err
		}
		return &f, nil
	})
}

// ===== mutation helpers (run inside an ambient withTx/withTxRet) =====

// mfaResolveProject returns the project id that owns the account, enforcing the
// account exists. It doubles as the tenant guard for enroll/challenge paths.
func (a *pgMFAAccounts) mfaResolveProject(ctx context.Context, accountID string) (string, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		return "", translatePgErr("user", err)
	}
	return row.ProjectID, nil
}

// mfaResolveAccount loads the owning account aggregate by id (project resolved
// from the row). It is the enroll-path counterpart to mfaResolveProject when the
// caller also needs account material such as the primary email for the TOTP
// authenticator label.
func (a *pgMFAAccounts) mfaResolveAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	row, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	var acc domain.Account
	if err := unmarshal(row.Data, &acc); err != nil {
		return nil, err
	}
	// Envelope is authoritative for the tenant + identifier.
	acc.ID = row.ID
	acc.ProjectID = row.ProjectID
	return &acc, nil
}

// mfaInsertChallenge persists a challenge envelope. subject carries the owning
// account; data carries the flow material (factor id, code hash, webauthn opts).
func (a *pgMFAAccounts) mfaInsertChallenge(ctx context.Context, projectID, accountID string, ch *domain.Challenge, data mfaChallengeData) error {
	raw, err := marshal(data)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	subject := null.From(accountID)
	setter := &models.IamChallengeSetter{
		ID:        ptr(ch.ID),
		ProjectID: ptr(projectID),
		Type:      ptr(ch.Type),
		Subject:   &subject,
		ExpiresAt: ptr(ch.ExpiresAt),
		Consumed:  ptr(false),
		CreatedAt: ptr(nowUTC()),
		Data:      &rm,
	}
	if data.FlowTokenHash != "" {
		ch := null.From(data.FlowTokenHash)
		setter.CodeHash = &ch
	}
	if _, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		return err
	}
	return nil
}

// mfaConsumeChallenge marks a challenge as single-use consumed.
func (a *pgMFAAccounts) mfaConsumeChallenge(ctx context.Context, row *models.IamChallenge) error {
	return row.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: ptr(true)})
}

// mfaEnrollDelivery is the shared enroll path for email/sms factors: it creates a
// pending factor and an accompanying delivery challenge whose one-time code is
// stored only as a sha256 hash (the plaintext code would be delivered out-of-band).
func (a *pgMFAAccounts) mfaEnrollDelivery(ctx context.Context, accountID, factorType, hint string) (*domain.Factor, *domain.Challenge, error) {
	type result struct {
		factor *domain.Factor
		ch     *domain.Challenge
	}
	out, err := withTxRet(ctx, a.db, func(ctx context.Context) (result, error) {
		projectID, err := a.mfaResolveProject(ctx, accountID)
		if err != nil {
			return result{}, err
		}
		count, err := a.mfaCountFactors(ctx, accountID)
		if err != nil {
			return result{}, err
		}
		if count >= mfaMaxFactorsPerAccount {
			return result{}, domain.ErrConflict.WithMessage("maximum number of MFA factors reached")
		}
		f := domain.Factor{
			ID:     newUUID(),
			Type:   factorType,
			Status: "pending",
			Hint:   hint,
		}
		if err := a.mfaInsertFactorFor(ctx, projectID, accountID, &f, ""); err != nil {
			return result{}, err
		}
		code, err := mfaNewOpaqueToken(4) // 8 hex chars delivered out-of-band
		if err != nil {
			return result{}, err
		}
		ch := domain.Challenge{
			ID:        newUUID(),
			Type:      factorType,
			ExpiresAt: nowUTC().Add(mfaChallengeTTL),
		}
		if err := a.mfaInsertChallenge(ctx, projectID, accountID, &ch, mfaChallengeData{
			FactorID:      f.ID,
			FlowTokenHash: mfaSha256Hex(code),
		}); err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.factor.enrolled",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: f.ID,
			Payload:     f,
		}); err != nil {
			return result{}, err
		}
		if err := a.emitter.Emit(ctx, domain.Event{
			Type:        "mfa.challenge.created",
			ProjectID:   projectID,
			Environment: "",
			AggregateID: ch.ID,
			Payload: map[string]any{
				"code":         code,
				"channel":      factorType,
				"factor_id":    f.ID,
				"challenge_id": ch.ID,
				"to":           hint,
				"contact":      hint,
			},
		}); err != nil {
			return result{}, err
		}
		return result{factor: &f, ch: &ch}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out.factor, out.ch, nil
}

// mfaInsertFactorFor persists a factor envelope for an explicit account id.
func (a *pgMFAAccounts) mfaInsertFactorFor(ctx context.Context, projectID, accountID string, f *domain.Factor, secret string) error {
	raw, err := marshal(f)
	if err != nil {
		return err
	}
	rm := json.RawMessage(raw)
	encSecret, err := a.db.Cipher.Encrypt(secret)
	if err != nil {
		return err
	}
	setter := &models.IamFactorSetter{
		ID:        ptr(f.ID),
		ProjectID: ptr(projectID),
		UserID:    ptr(accountID),
		Type:      ptr(f.Type),
		Status:    ptr(f.Status),
		Secret:    ptr(encSecret),
		CreatedAt: ptr(nowUTC()),
		Data:      &rm,
	}
	if _, err := models.IamFactors.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

// mfaDefaultEnv is the environment whose signing key mints the MFA session's
// access-token JWT.
const mfaDefaultEnv = "live"

// mfaAccessTTL bounds the minted access-token JWT.
const mfaAccessTTL = 30 * time.Minute

// mfaMintSession produces a session for a freshly verified (AAL2) account. The
// access token is a signed RS256 JWT minted by the project Signer (jwx, carrying
// the session sid); the refresh token stays an opaque random handle.
func (a *pgMFAAccounts) mfaMintSession(ctx context.Context, acc *domain.Account) (*domain.Session, error) {
	sessionID := newUUID()
	signEnv, err := resolveSignEnv(ctx, a.db, acc.ProjectID, mfaDefaultEnv)
	if err != nil {
		return nil, err
	}
	access, err := a.db.Signer().Sign(ctx, acc.ProjectID, signEnv, map[string]any{
		"iss": acc.ProjectID,
		"sub": acc.ID,
		"sid": sessionID,
		"pid": acc.ProjectID,
		"aal": 2,
		"amr": []string{"mfa"},
		"typ": "access",
		"env": signEnv,
	}, mfaAccessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := mfaNewOpaqueToken(32)
	if err != nil {
		return nil, err
	}
	return &domain.Session{
		ID:           sessionID,
		AccountID:    acc.ID,
		ProjectID:    acc.ProjectID,
		AMR:          []string{"mfa"},
		AAL:          2,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(mfaAccessTTL / time.Second),
		CreatedAt:    nowUTC(),
	}, nil
}
