package postgres

// WebAuthn / passkey adapter — persists WebAuthn credentials (iam_webauthn_credentials)
// and the short-lived ceremony challenges (iam_challenges) used during the
// registration (attestation) and login (assertion) flows.
//
// The cryptographic attestation (registration) and assertion (login)
// verification is delegated to github.com/go-webauthn/webauthn: BeginRegistration
// / BeginLogin mint the publicKey options + a SessionData snapshot we persist in
// iam_challenges.data; FinishRegistration / FinishLogin replay that SessionData
// against the browser's PublicKeyCredential response to verify the signature, the
// challenge, the origin, and (on login) the signature counter for clone
// detection. We never hand-roll the COSE / CBOR crypto.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// challengeTTL bounds how long a WebAuthn ceremony challenge stays valid.
const webauthnChallengeTTL = 5 * time.Minute

// webauthnSignerEnv is the signing-key environment the minted access tokens are
// issued under (matches the default-env convention used by the other adapters).
const webauthnSignerEnv = "live"

// pgWebAuthnAccounts is the Postgres adapter for the WebAuthn account ports.
type pgWebAuthnAccounts struct{ db *DB }

// NewPgWebAuthnAccounts builds the WebAuthn adapter.
func NewPgWebAuthnAccounts(db *DB) *pgWebAuthnAccounts { return &pgWebAuthnAccounts{db: db} }

// Port assertion — keeps the adapter honest against the pkg/api contract.
var _ api.WebAuthnAccounts = (*pgWebAuthnAccounts)(nil)

// ----- relying-party configuration -----
//
// RP config derivation choice: the Relying Party is scoped per project. The
// effective RP ID (a registrable domain) is taken from the project's `data`
// jsonb (key "webauthnRpId", falling back to "rpId"); the permitted origins from
// "webauthnRpOrigins"/"rpOrigins". When a project has not been configured we fall
// back to a localhost RP so single-host / dev deployments work out of the box.
// This keeps the crypto verification (challenge + origin binding) anchored to a
// real domain without requiring a new column.
const (
	webauthnDefaultRPID     = "localhost"
	webauthnDefaultRPOrigin = "http://localhost"
)

// rpConfigFor loads the project and builds the go-webauthn instance bound to its
// Relying Party identity (id + permitted origins).
func (a *pgWebAuthnAccounts) rpConfigFor(ctx context.Context, projectID string) (*gowebauthn.WebAuthn, error) {
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

// webauthnRPFromProject extracts the RP id, display name and permitted origins
// from the project envelope, applying the localhost defaults when unset.
func webauthnRPFromProject(row *models.IamProject) (rpID, displayName string, origins []string) {
	rpID = webauthnDefaultRPID
	origins = []string{webauthnDefaultRPOrigin}
	displayName = row.Name
	if displayName == "" {
		displayName = row.Slug
	}

	var meta map[string]any
	if len(row.Data) > 0 {
		_ = json.Unmarshal(row.Data, &meta)
	}
	if id := webauthnMetaString(meta, "webauthnRpId", "rpId"); id != "" {
		rpID = id
	}
	if o := webauthnMetaStrings(meta, "webauthnRpOrigins", "rpOrigins"); len(o) > 0 {
		origins = o
	} else if rpID != webauthnDefaultRPID {
		// Derive a single https origin from the configured RP id.
		origins = []string{"https://" + rpID}
	}
	return rpID, displayName, origins
}

func webauthnMetaString(meta map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := meta[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func webauthnMetaStrings(meta map[string]any, keys ...string) []string {
	for _, k := range keys {
		raw, ok := meta[k].([]any)
		if !ok {
			continue
		}
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

// ----- go-webauthn.User adapter -----
//
// webauthnUser adapts the domain account plus its stored credentials onto the
// go-webauthn User interface the library consumes for both ceremonies.
type webauthnUser struct {
	account *domain.Account
	creds   []gowebauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte          { return []byte(u.account.ID) }
func (u *webauthnUser) WebAuthnName() string        { return webauthnUserName(u.account) }
func (u *webauthnUser) WebAuthnDisplayName() string { return webauthnDisplayName(u.account) }
func (u *webauthnUser) WebAuthnCredentials() []gowebauthn.Credential {
	return u.creds
}

func webauthnUserName(a *domain.Account) string {
	if a.PrimaryEmail != "" {
		return a.PrimaryEmail
	}
	return a.ID
}

func webauthnDisplayName(a *domain.Account) string {
	if a.Name != "" {
		return a.Name
	}
	return webauthnUserName(a)
}

// loadWebauthnUser reads the account row + its stored library credentials and
// adapts them onto the go-webauthn User interface.
func (a *pgWebAuthnAccounts) loadWebauthnUser(ctx context.Context, accountID string) (*webauthnUser, error) {
	userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
	if err != nil {
		return nil, translatePgErr("user", err)
	}
	var acct domain.Account
	if err := unmarshal(userRow.Data, &acct); err != nil {
		return nil, err
	}
	creds, err := a.loadLibraryCredentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return &webauthnUser{account: &acct, creds: creds}, nil
}

// loadLibraryCredentials rehydrates the go-webauthn Credential records persisted
// for an account from the credential `data` envelopes.
func (a *pgWebAuthnAccounts) loadLibraryCredentials(ctx context.Context, accountID string) ([]gowebauthn.Credential, error) {
	rows, err := models.IamWebauthnCredentials.Query(
		sm.Where(models.IamWebauthnCredentials.Columns.UserID.EQ(psql.Arg(accountID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, err
	}
	out := make([]gowebauthn.Credential, 0, len(rows))
	for _, row := range rows {
		var stored domain.WebAuthnStoredCredential
		if err := unmarshal(row.Data, &stored); err != nil {
			return nil, err
		}
		if len(stored.Library) == 0 {
			continue
		}
		var lib gowebauthn.Credential
		if err := json.Unmarshal(stored.Library, &lib); err != nil {
			return nil, err
		}
		out = append(out, lib)
	}
	return out, nil
}

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

// consumeChallenge flips the single-use flag so a challenge can't be replayed.
func (a *pgWebAuthnAccounts) consumeChallenge(ctx context.Context, row *models.IamChallenge) error {
	return row.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: ptr(true)})
}

// insertCeremony persists a WebAuthn ceremony: the publicKey options surfaced to
// the browser plus the opaque go-webauthn SessionData replayed on Finish*. The
// code_hash column keys on the library challenge value for lookup; the Challenge
// aggregate returned to the caller mirrors the publicKey options.
func (a *pgWebAuthnAccounts) insertCeremony(ctx context.Context, projectID, ctype string, publicKey map[string]any, session *gowebauthn.SessionData, accountID string) (*domain.Challenge, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Challenge, error) {
		sessionRaw, err := json.Marshal(session)
		if err != nil {
			return nil, err
		}
		cer := domain.WebAuthnCeremonyData{
			PublicKey: publicKey,
			Session:   sessionRaw,
			AccountID: accountID,
		}
		data, err := marshal(cer)
		if err != nil {
			return nil, err
		}
		ch := &domain.Challenge{
			ID:        newUUID(),
			Type:      ctype,
			ExpiresAt: nowUTC().Add(webauthnChallengeTTL),
			PublicKey: publicKey,
		}
		rm := json.RawMessage(data)
		hash := null.From(webauthnHash(session.Challenge))
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

// loadCeremony fetches a ceremony challenge scoped to projectID, enforcing TTL +
// the single-use invariant, and rehydrates the go-webauthn SessionData.
func (a *pgWebAuthnAccounts) loadCeremony(ctx context.Context, projectID, challengeID, ctype string) (*models.IamChallenge, *domain.WebAuthnCeremonyData, *gowebauthn.SessionData, error) {
	row, err := models.FindIamChallenge(ctx, a.db.Bobx(), challengeID)
	if err != nil {
		return nil, nil, nil, translatePgErr("challenge", err)
	}
	if row.ProjectID != projectID || row.Type != ctype { // tenant + ceremony boundary
		return nil, nil, nil, domain.ErrChallengeInvalid
	}
	if row.Consumed {
		return nil, nil, nil, domain.ErrChallengeInvalid
	}
	if nowUTC().After(row.ExpiresAt) {
		return nil, nil, nil, domain.ErrChallengeExpired
	}
	var cer domain.WebAuthnCeremonyData
	if err := unmarshal(row.Data, &cer); err != nil {
		return nil, nil, nil, err
	}
	var session gowebauthn.SessionData
	if err := json.Unmarshal(cer.Session, &session); err != nil {
		return nil, nil, nil, domain.ErrChallengeInvalid
	}
	return row, &cer, &session, nil
}

// webauthnOptionsMap marshals a go-webauthn options payload (CredentialCreation
// or CredentialAssertion response) into the publicKey map surfaced to the client.
func webauthnOptionsMap(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// webauthnCredentialReader re-marshals the browser credential map and returns a
// reader the go-webauthn protocol parsers consume.
func webauthnCredentialReader(credential map[string]any) (*bytes.Reader, error) {
	raw, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(raw), nil
}

// ----- credential persistence helpers -----

// loadCredential fetches a credential by id, enforcing the account boundary
// (a row owned by another user is treated as not-found). It decodes the stored
// wrapper so the opaque library material is preserved across mutations.
func (a *pgWebAuthnAccounts) loadCredential(ctx context.Context, accountID, credentialID string) (*models.IamWebauthnCredential, *domain.WebAuthnStoredCredential, error) {
	row, err := models.FindIamWebauthnCredential(ctx, a.db.Bobx(), credentialID)
	if err != nil {
		return nil, nil, translatePgErr("webauthn_credential", err)
	}
	if row.UserID != accountID { // ownership boundary
		return nil, nil, domain.ErrNotFound
	}
	var stored domain.WebAuthnStoredCredential
	if err := unmarshal(row.Data, &stored); err != nil {
		return nil, nil, err
	}
	return row, &stored, nil
}

// ----- api.WebAuthnAccounts -----

// BeginLogin starts an assertion ceremony for the given email within a project.
// The publicKey options carry the allowed credentials and the fresh challenge.
func (a *pgWebAuthnAccounts) BeginLogin(ctx context.Context, projectID, email string) (*domain.Challenge, error) {
	w, err := a.rpConfigFor(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Resolve the account for the email so the library can scope allowed
	// credentials. A missing account (or one without passkeys) is surfaced as
	// invalid credentials to avoid leaking account existence to anonymous
	// callers. (Identified, non-discoverable login.)
	if email == "" {
		return nil, domain.ErrInvalidCredentials
	}
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
	var acct domain.Account
	if err := unmarshal(rows[0].Data, &acct); err != nil {
		return nil, err
	}
	creds, err := a.loadLibraryCredentials(ctx, acct.ID)
	if err != nil {
		return nil, err
	}
	if len(creds) == 0 {
		return nil, domain.ErrInvalidCredentials
	}
	user := &webauthnUser{account: &acct, creds: creds}

	// BeginLogin mints the publicKey request options + the SessionData snapshot
	// (challenge, RP id, allowed credentials, user verification) the library
	// replays in FinishLogin to verify the assertion signature.
	assertion, session, err := w.BeginLogin(user)
	if err != nil {
		return nil, domain.ErrProviderError
	}
	publicKey, err := webauthnOptionsMap(assertion.Response)
	if err != nil {
		return nil, err
	}
	return a.insertCeremony(ctx, projectID, "webauthn_login", publicKey, session, acct.ID)
}

// FinishLogin verifies the assertion with go-webauthn and, on success, mints a
// session. The library replays the persisted SessionData against the browser's
// PublicKeyCredential response: it checks the challenge, the origin, the RP id,
// the credential public-key signature and the signature counter (clone
// detection). We persist the bumped sign count and mint a signed access token.
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

		_, cer, session, err := a.loadCeremony(ctx, projectID, challengeID, "webauthn_login")
		if err != nil {
			return loginResult{}, err
		}

		w, err := a.rpConfigFor(ctx, projectID)
		if err != nil {
			return loginResult{}, err
		}

		// The ceremony was bound to a specific account at BeginLogin time.
		user, err := a.loadWebauthnUser(ctx, cer.AccountID)
		if err != nil {
			return loginResult{}, err
		}

		// verify with WebAuthn signing/attestation — parse the browser
		// assertion and validate its signature, challenge, origin and signature
		// counter against the stored credential public key via go-webauthn.
		reader, err := webauthnCredentialReader(credential)
		if err != nil {
			return loginResult{}, err
		}
		parsed, err := protocol.ParseCredentialRequestResponseBody(reader)
		if err != nil {
			return loginResult{}, domain.ErrMFAInvalid
		}
		validated, err := w.ValidateLogin(user, *session, parsed)
		if err != nil {
			return loginResult{}, domain.ErrMFAInvalid
		}

		// Persist the verified credential's bumped sign count + last-used. The
		// library credential id is the raw byte id; our row keys on the base64url
		// credential id surfaced to the client.
		credID := base64.RawURLEncoding.EncodeToString(validated.ID)
		credRow, err := models.FindIamWebauthnCredential(ctx, a.db.Bobx(), credID)
		if err != nil {
			return loginResult{}, translatePgErr("webauthn_credential", err)
		}
		if credRow.ProjectID != projectID || credRow.UserID != cer.AccountID {
			return loginResult{}, domain.ErrMFAInvalid
		}
		var stored domain.WebAuthnStoredCredential
		if err := unmarshal(credRow.Data, &stored); err != nil {
			return loginResult{}, err
		}
		now := nowUTC()
		stored.Credential.LastUsedAt = now
		libRaw, err := json.Marshal(validated)
		if err != nil {
			return loginResult{}, err
		}
		stored.Library = libRaw
		storedRaw, err := marshal(stored)
		if err != nil {
			return loginResult{}, err
		}
		rmStored := json.RawMessage(storedRaw)
		used := null.From(now)
		pk := null.From(validated.PublicKey)
		if err := credRow.Update(ctx, a.db.Bobx(), &models.IamWebauthnCredentialSetter{
			SignCount:  ptr(int64(validated.Authenticator.SignCount)),
			PublicKey:  &pk,
			LastUsedAt: &used,
			Data:       &rmStored,
		}); err != nil {
			return loginResult{}, err
		}

		acct := user.account

		if err := a.consumeChallenge(ctx, row); err != nil {
			return loginResult{}, err
		}

		// Mint the session. The access token is a signed RS256 JWT from the
		// project's active signing key (jwx); the refresh token stays opaque.
		sessionID := newUUID()
		accessToken, err := a.db.Signer().Sign(ctx, projectID, webauthnSignerEnv, map[string]any{
			"iss": projectID,
			"sub": acct.ID,
			"sid": sessionID,
			"pid": projectID,
			"aal": 2,
			"amr": []string{"webauthn"},
			"typ": "access",
		}, time.Hour)
		if err != nil {
			return loginResult{}, err
		}
		refresh, err := webauthnRandomChallenge()
		if err != nil {
			return loginResult{}, err
		}
		sess := &domain.Session{
			ID:           sessionID,
			AccountID:    acct.ID,
			ProjectID:    projectID,
			AMR:          []string{"webauthn"},
			AAL:          2,
			AccessToken:  accessToken,
			RefreshToken: refresh,
			ExpiresIn:    int(time.Hour.Seconds()),
			CreatedAt:    now,
		}
		// TODO outbox event: webauthn.login.succeeded
		return loginResult{acct: acct, sess: sess}, nil
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
	projectID := userRow.ProjectID

	w, err := a.rpConfigFor(ctx, projectID)
	if err != nil {
		return nil, err
	}
	user, err := a.loadWebauthnUser(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Exclude the user's already-registered passkeys so the authenticator does
	// not create a duplicate credential.
	exclusions := make([]protocol.CredentialDescriptor, 0, len(user.creds))
	for i := range user.creds {
		exclusions = append(exclusions, user.creds[i].Descriptor())
	}

	creation, session, err := w.BeginRegistration(user, gowebauthn.WithExclusions(exclusions))
	if err != nil {
		return nil, domain.ErrProviderError
	}
	publicKey, err := webauthnOptionsMap(creation.Response)
	if err != nil {
		return nil, err
	}
	return a.insertCeremony(ctx, projectID, "webauthn_register", publicKey, session, accountID)
}

// FinishRegistration verifies the attestation with go-webauthn and persists the
// new credential. The library validates the attestation object + clientDataJSON
// against the persisted SessionData (challenge, RP id, origin) and returns the
// verified credential id, COSE public key and sign count, which we store.
func (a *pgWebAuthnAccounts) FinishRegistration(ctx context.Context, accountID, challengeID string, credential map[string]any) (*domain.WebAuthnCredential, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.WebAuthnCredential, error) {
		userRow, err := models.FindIamUser(ctx, a.db.Bobx(), accountID)
		if err != nil {
			return nil, translatePgErr("user", err)
		}
		projectID := userRow.ProjectID

		row, cer, session, err := a.loadCeremony(ctx, projectID, challengeID, "webauthn_register")
		if err != nil {
			return nil, err
		}
		if cer.AccountID != accountID {
			return nil, domain.ErrChallengeInvalid
		}

		w, err := a.rpConfigFor(ctx, projectID)
		if err != nil {
			return nil, err
		}
		user, err := a.loadWebauthnUser(ctx, accountID)
		if err != nil {
			return nil, err
		}

		// The optional display name is a UI-only attribute supplied alongside the
		// credential; pull it out before the protocol parse (which ignores it).
		name, _ := credential["name"].(string)
		if name == "" {
			name = "Passkey"
		}

		// verify with WebAuthn signing/attestation — parse the browser
		// attestation response and validate it (challenge, origin, RP id,
		// attestation statement) against the stored SessionData via go-webauthn.
		reader, err := webauthnCredentialReader(credential)
		if err != nil {
			return nil, err
		}
		parsed, err := protocol.ParseCredentialCreationResponseBody(reader)
		if err != nil {
			return nil, domain.ErrMFAInvalid
		}
		libCred, err := w.CreateCredential(user, *session, parsed)
		if err != nil {
			return nil, domain.ErrMFAInvalid
		}

		// The credential id surfaced to the client is the base64url raw id.
		credID := base64.RawURLEncoding.EncodeToString(libCred.ID)
		now := nowUTC()
		cred := domain.WebAuthnCredential{
			ID:        credID,
			Name:      name,
			CreatedAt: now,
		}
		libRaw, err := json.Marshal(libCred)
		if err != nil {
			return nil, err
		}
		stored := domain.WebAuthnStoredCredential{Credential: cred, Library: libRaw}
		data, err := marshal(stored)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(data)
		pubKey := null.From(libCred.PublicKey)
		setter := &models.IamWebauthnCredentialSetter{
			ID:           &credID,
			ProjectID:    &projectID,
			UserID:       &accountID,
			CredentialID: &credID,
			PublicKey:    &pubKey,
			SignCount:    ptr(int64(libCred.Authenticator.SignCount)),
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
		return &cred, nil
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
		var stored domain.WebAuthnStoredCredential
		if err := unmarshal(row.Data, &stored); err != nil {
			return nil, err
		}
		cred := stored.Credential
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
		row, stored, err := a.loadCredential(ctx, cmd.AccountID, cmd.CredentialID)
		if err != nil {
			return nil, err
		}
		stored.Credential.Name = cmd.Name
		data, err := marshal(stored)
		if err != nil {
			return nil, err
		}
		rm := json.RawMessage(data)
		if err := row.Update(ctx, a.db.Bobx(), &models.IamWebauthnCredentialSetter{Data: &rm}); err != nil {
			return nil, err
		}
		cred := stored.Credential
		if row.LastUsedAt.IsValue() {
			cred.LastUsedAt = row.LastUsedAt.GetOrZero()
		}
		// TODO outbox event: webauthn.credential.renamed
		return &cred, nil
	})
}
