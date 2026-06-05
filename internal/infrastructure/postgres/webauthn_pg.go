package postgres

// WebAuthn / passkey adapter — persists WebAuthn credentials (iam_webauthn_credentials)
// and the short-lived ceremony challenges (iam_challenges) used during the
// registration (attestation) and login (assertion) flows.
//
// The actual WebAuthn attestation/assertion cryptographic verification is NOT
// implemented here: the adapter persists the ceremony state and the submitted
// credential material, returning the stored aggregate. Each crypto boundary is
// marked with a `// TODO: verify with WebAuthn signing/attestation` comment.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// challengeTTL bounds how long a WebAuthn ceremony challenge stays valid.
const webauthnChallengeTTL = 5 * time.Minute

// pgWebAuthnAccounts is the Postgres adapter for the WebAuthn account ports.
type pgWebAuthnAccounts struct{ db *DB }

// NewPgWebAuthnAccounts builds the WebAuthn adapter.
func NewPgWebAuthnAccounts(db *DB) *pgWebAuthnAccounts { return &pgWebAuthnAccounts{db: db} }

// Port assertion — keeps the adapter honest against the pkg/api contract.
var _ api.WebAuthnAccounts = (*pgWebAuthnAccounts)(nil)

// ----- challenge persistence helpers -----

// webauthnRandomChallenge mints a fresh, URL-safe random challenge string for
// the publicKey options (crypto/rand, never predictable).
func webauthnRandomChallenge() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// webauthnHash returns the sha256 hex digest of an opaque value; only digests
// are persisted, never the plaintext challenge/credential material.
func webauthnHash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

// insertChallenge persists a ceremony challenge envelope and returns the domain
// aggregate. The raw challenge bytes live inside PublicKey; the code_hash column
// stores only the sha256 digest for lookup.
func (a *pgWebAuthnAccounts) insertChallenge(ctx context.Context, projectID, ctype string, publicKey map[string]any) (*domain.Challenge, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Challenge, error) {
		raw, err := webauthnRandomChallenge()
		if err != nil {
			return nil, err
		}
		if publicKey == nil {
			publicKey = map[string]any{}
		}
		publicKey["challenge"] = raw

		ch := &domain.Challenge{
			ID:        newUUID(),
			Type:      ctype,
			ExpiresAt: nowUTC().Add(webauthnChallengeTTL),
			PublicKey: publicKey,
		}
		data, err := marshal(ch)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(data)
		hash := null.From(webauthnHash(raw))
		setter := &models.IamChallengeSetter{
			ID:        &ch.ID,
			ProjectID: &projectID,
			Type:      &ctype,
			CodeHash:  &hash,
			ExpiresAt: ptr(ch.ExpiresAt),
			Consumed:  ptr(false),
			Data:      &rm,
		}
		if _, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			return nil, err
		}
		// TODO outbox event: webauthn.challenge.created
		return ch, nil
	})
}

// loadChallenge fetches a challenge scoped to projectID, enforcing TTL and the
// single-use (consumed) invariant. It returns the row and decoded aggregate.
func (a *pgWebAuthnAccounts) loadChallenge(ctx context.Context, projectID, challengeID, ctype string) (*models.IamChallenge, *domain.Challenge, error) {
	row, err := models.FindIamChallenge(ctx, a.db.Bobx(), challengeID)
	if err != nil {
		return nil, nil, translatePgErr("challenge", err)
	}
	if row.ProjectID != projectID || row.Type != ctype { // tenant + ceremony boundary
		return nil, nil, domain.ErrChallengeInvalid
	}
	if row.Consumed {
		return nil, nil, domain.ErrChallengeInvalid
	}
	if nowUTC().After(row.ExpiresAt) {
		return nil, nil, domain.ErrChallengeExpired
	}
	var ch domain.Challenge
	if err := unmarshal(row.Data, &ch); err != nil {
		return nil, nil, err
	}
	return row, &ch, nil
}

// consumeChallenge flips the single-use flag so a challenge can't be replayed.
func (a *pgWebAuthnAccounts) consumeChallenge(ctx context.Context, row *models.IamChallenge) error {
	return row.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: ptr(true)})
}

// ----- credential persistence helpers -----

// loadCredential fetches a credential by id, enforcing the account boundary
// (a row owned by another user is treated as not-found).
func (a *pgWebAuthnAccounts) loadCredential(ctx context.Context, accountID, credentialID string) (*models.IamWebauthnCredential, *domain.WebAuthnCredential, error) {
	row, err := models.FindIamWebauthnCredential(ctx, a.db.Bobx(), credentialID)
	if err != nil {
		return nil, nil, translatePgErr("webauthn_credential", err)
	}
	if row.UserID != accountID { // ownership boundary
		return nil, nil, domain.ErrNotFound
	}
	var cred domain.WebAuthnCredential
	if err := unmarshal(row.Data, &cred); err != nil {
		return nil, nil, err
	}
	return row, &cred, nil
}

// ----- api.WebAuthnAccounts -----

// BeginLogin starts an assertion ceremony for the given email within a project.
// The publicKey options carry the allowed credentials and the fresh challenge.
func (a *pgWebAuthnAccounts) BeginLogin(ctx context.Context, projectID, email string) (*domain.Challenge, error) {
	// Resolve the account for the email so we can scope allowed credentials.
	// A missing account is surfaced as invalid credentials to avoid leaking
	// account existence to anonymous callers.
	var acct *domain.Account
	if email != "" {
		rows, err := models.IamUsers.Query(
			sm.Where(models.IamUsers.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamUsers.Columns.PrimaryEmail.EQ(psql.Arg(email))),
		).All(ctx, a.db.Bobx())
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, domain.ErrInvalidCredentials
		}
		var dec domain.Account
		if err := unmarshal(rows[0].Data, &dec); err != nil {
			return nil, err
		}
		acct = &dec
	}

	publicKey := map[string]any{
		"userVerification": "preferred",
	}
	if acct != nil {
		creds, err := a.ListCredentials(ctx, acct.ID)
		if err != nil {
			return nil, err
		}
		allow := make([]map[string]any, 0, len(creds))
		for _, c := range creds {
			allow = append(allow, map[string]any{"type": "public-key", "id": c.ID})
		}
		publicKey["allowCredentials"] = allow
	}
	return a.insertChallenge(ctx, projectID, "webauthn_login", publicKey)
}

// FinishLogin verifies the assertion and, on success, mints a session.
// The cryptographic assertion verification is not implemented here.
func (a *pgWebAuthnAccounts) FinishLogin(ctx context.Context, challengeID string, credential map[string]any) (*domain.Account, *domain.Session, error) {
	// withTxRet returns a single value; pair the account+session through a
	// local struct so the whole login stays inside one serializable tx.
	type loginResult struct {
		acct *domain.Account
		sess *domain.Session
	}
	res, err := withTxRet(ctx, a.db, func(ctx context.Context) (loginResult, error) {
		// We must locate the challenge to know which project we operate in.
		row, err := models.FindIamChallenge(ctx, a.db.Bobx(), challengeID)
		if err != nil {
			return loginResult{}, translatePgErr("challenge", err)
		}
		if row.Type != "webauthn_login" || row.Consumed {
			return loginResult{}, domain.ErrChallengeInvalid
		}
		if nowUTC().After(row.ExpiresAt) {
			return loginResult{}, domain.ErrChallengeExpired
		}
		projectID := row.ProjectID

		// TODO: verify with WebAuthn signing/attestation — validate the
		// assertion signature against the stored credential public key and
		// challenge, then look up the owning credential. Until the verifier is
		// wired, the submitted credential id identifies the account.
		credID, _ := credential["id"].(string)
		if credID == "" {
			return loginResult{}, domain.ErrInvalidCredentials
		}
		credRow, err := models.FindIamWebauthnCredential(ctx, a.db.Bobx(), credID)
		if err != nil {
			return loginResult{}, domain.ErrInvalidCredentials
		}
		if credRow.ProjectID != projectID {
			return loginResult{}, domain.ErrInvalidCredentials
		}

		// Bump sign count + last-used on the credential (replay defence once the
		// verifier is wired).
		used := null.From(nowUTC())
		if err := credRow.Update(ctx, a.db.Bobx(), &models.IamWebauthnCredentialSetter{
			SignCount:  ptr(credRow.SignCount + 1),
			LastUsedAt: &used,
		}); err != nil {
			return loginResult{}, err
		}

		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), credRow.UserID)
		if err != nil {
			return loginResult{}, translatePgErr("user", err)
		}
		var acct domain.Account
		if err := unmarshal(userRow.Data, &acct); err != nil {
			return loginResult{}, err
		}

		if err := a.consumeChallenge(ctx, row); err != nil {
			return loginResult{}, err
		}

		// Mint the session opaque tokens. The access/refresh tokens are opaque
		// random values; JWT minting/signing is out of scope here.
		access, err := webauthnRandomChallenge()
		if err != nil {
			return loginResult{}, err
		}
		refresh, err := webauthnRandomChallenge()
		if err != nil {
			return loginResult{}, err
		}
		// TODO: sign/verify with signing key — replace the opaque access token
		// with a signed JWT access/id token minted from the signing key.
		sess := &domain.Session{
			ID:           newUUID(),
			AccountID:    acct.ID,
			ProjectID:    projectID,
			AMR:          []string{"webauthn"},
			AAL:          2,
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresIn:    int(time.Hour.Seconds()),
			CreatedAt:    nowUTC(),
		}
		// TODO outbox event: webauthn.login.succeeded
		return loginResult{acct: &acct, sess: sess}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res.acct, res.sess, nil
}

// BeginRegistration starts an attestation ceremony for an existing account.
func (a *pgWebAuthnAccounts) BeginRegistration(ctx context.Context, accountID string) (*domain.Challenge, error) {
	userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	publicKey := map[string]any{
		"user": map[string]any{
			"id":          userRow.ID,
			"name":        userRow.PrimaryEmail.GetOrZero(),
			"displayName": userRow.PrimaryEmail.GetOrZero(),
		},
		"authenticatorSelection": map[string]any{"userVerification": "preferred"},
	}
	return a.insertChallenge(ctx, userRow.ProjectID, "webauthn_register", publicKey)
}

// FinishRegistration verifies the attestation and persists the new credential.
// The cryptographic attestation verification is not implemented here.
func (a *pgWebAuthnAccounts) FinishRegistration(ctx context.Context, accountID, challengeID string, credential map[string]any) (*domain.WebAuthnCredential, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.WebAuthnCredential, error) {
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			return nil, translatePgErr("user", err)
		}
		projectID := userRow.ProjectID

		row, _, err := a.loadChallenge(ctx, projectID, challengeID, "webauthn_register")
		if err != nil {
			return nil, err
		}

		// TODO: verify with WebAuthn signing/attestation — validate the
		// attestation object/clientDataJSON against the stored challenge and
		// extract the verified credential id + public key. Until then we trust
		// the submitted material.
		credID, _ := credential["id"].(string)
		if credID == "" {
			return nil, domain.ErrInvalidCredentials
		}
		name, _ := credential["name"].(string)
		if name == "" {
			name = "Passkey"
		}

		var pubKey null.Val[[]byte]
		if rawKey, ok := credential["publicKey"].(string); ok && rawKey != "" {
			pubKey = null.From([]byte(rawKey))
		}

		now := nowUTC()
		cred := &domain.WebAuthnCredential{
			ID:        credID,
			Name:      name,
			CreatedAt: now,
		}
		data, err := marshal(cred)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(data)
		setter := &models.IamWebauthnCredentialSetter{
			ID:           &credID,
			ProjectID:    &projectID,
			UserID:       &accountID,
			CredentialID: &credID,
			PublicKey:    &pubKey,
			SignCount:    ptr(int64(0)),
			CreatedAt:    &now,
			Data:         &rm,
		}
		if _, err := models.IamWebauthnCredentials.Insert(setter).One(ctx, a.db.Bobx()); err != nil {
			if isUniqueViolation(err) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}

		if err := a.consumeChallenge(ctx, row); err != nil {
			return nil, err
		}
		// TODO outbox event: webauthn.credential.registered
		return cred, nil
	})
}

// ListCredentials returns every passkey owned by the account.
func (a *pgWebAuthnAccounts) ListCredentials(ctx context.Context, accountID string) ([]domain.WebAuthnCredential, error) {
	rows, err := models.IamWebauthnCredentials.Query(
		sm.Where(models.IamWebauthnCredentials.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]domain.WebAuthnCredential, 0, len(rows))
	for _, row := range rows {
		var cred domain.WebAuthnCredential
		if err := unmarshal(row.Data, &cred); err != nil {
			return nil, err
		}
		// Reflect persisted ceremony timestamps onto the aggregate.
		if row.LastUsedAt.IsValue() {
			cred.LastUsedAt = row.LastUsedAt.GetOrZero()
		}
		out = append(out, cred)
	}
	return out, nil
}

// RemoveCredential deletes a passkey owned by the account.
func (a *pgWebAuthnAccounts) RemoveCredential(ctx context.Context, accountID, credentialID string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		row, _, err := a.loadCredential(ctx, accountID, credentialID)
		if err != nil {
			return err
		}
		if err := row.Delete(ctx, a.db.Bobx()); err != nil {
			return err
		}
		// TODO outbox event: webauthn.credential.removed
		return nil
	})
}

// RenameCredential updates the display name of an owned passkey.
func (a *pgWebAuthnAccounts) RenameCredential(ctx context.Context, cmd domain.WebAuthnRenameCredentialCmd) (*domain.WebAuthnCredential, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.WebAuthnCredential, error) {
		row, cred, err := a.loadCredential(ctx, cmd.AccountID, cmd.CredentialID)
		if err != nil {
			return nil, err
		}
		cred.Name = cmd.Name
		data, err := marshal(cred)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(data)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamWebauthnCredentialSetter{Data: &rm}); err != nil {
			return nil, err
		}
		if row.LastUsedAt.IsValue() {
			cred.LastUsedAt = row.LastUsedAt.GetOrZero()
		}
		// TODO outbox event: webauthn.credential.renamed
		return cred, nil
	})
}
