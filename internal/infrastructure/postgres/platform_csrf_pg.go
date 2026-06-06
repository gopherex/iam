package postgres

// CSRF token issuance for cookie-mode clients.
//
// IssueCsrfToken mints a cryptographically random token (32 bytes, hex-encoded)
// bound to the clientID, persists it as a short-lived iam_challenges row of
// type "csrf_token" (same table/pattern as OAuth state in oauthsocial_state.go),
// and returns the raw token to the caller for embedding in a cookie/header.
//
// Storage approach: iam_challenges is the repo's canonical home for short-lived
// one-time tokens.  The raw token is stored in code_hash (sha256 hex) so that a
// future verify step can look it up in constant time without storing the secret
// in plain text.  clientID is kept in subject for auditing.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"

	"github.com/aarondl/opt/null"
)

// csrfTokenTTL is how long an issued CSRF token remains valid.
const csrfTokenTTL = 1 * time.Hour

var _ api.PlatformCsrf = (*pgPlatform)(nil)

// IssueCsrfToken generates a random CSRF token bound to clientID, persists it
// with a TTL, and returns *domain.PlatformCsrfToken carrying the raw token.
func (a *pgPlatform) IssueCsrfToken(ctx context.Context, clientID string) (*domain.PlatformCsrfToken, error) {
	raw, err := csrfRandomToken()
	if err != nil {
		return nil, err
	}

	id := newUUID()
	typ := "csrf_token"
	sub := null.From(clientID)
	hash := csrfHashToken(raw)
	ch := null.From(hash)
	exp := nowUTC().Add(csrfTokenTTL)
	data := json.RawMessage(`{}`)

	// project_id is NOT NULL; resolve it from the requesting client when the
	// client is registered, otherwise scope the token to the platform ("").
	projectID := ""
	if app, err := models.FindIamAppClient(ctx, a.db.Bobx(), clientID); err == nil {
		projectID = app.ProjectID
	}

	err = a.db.withTx(ctx, func(ctx context.Context) error {
		setter := &models.IamChallengeSetter{
			ID:        &id,
			ProjectID: &projectID,
			Type:      &typ,
			Subject:   &sub,
			CodeHash:  &ch,
			ExpiresAt: &exp,
			Data:      &data,
		}
		_, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx())
		return err
	})
	if err != nil {
		return nil, err
	}

	return &domain.PlatformCsrfToken{Token: raw}, nil
}

// VerifyCsrfToken validates a CSRF token issued to clientID: it must exist, be
// of type csrf_token, be bound to the same client (subject) and be unexpired.
// Reusable within its TTL (synchronizer-token pattern — not consumed on verify).
func (a *pgPlatform) VerifyCsrfToken(ctx context.Context, clientID, token string) error {
	if clientID == "" || token == "" {
		return domain.ErrInvalidCsrf
	}
	row, err := models.IamChallenges.Query(
		sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg("csrf_token"))),
		sm.Where(models.IamChallenges.Columns.CodeHash.EQ(psql.Arg(csrfHashToken(token)))),
	).One(ctx, a.db.Bobx())
	if err != nil {
		return domain.ErrInvalidCsrf
	}
	sub, ok := row.Subject.Get()
	if !ok || subtle.ConstantTimeCompare([]byte(sub), []byte(clientID)) != 1 {
		return domain.ErrInvalidCsrf
	}
	if nowUTC().After(row.ExpiresAt) {
		return domain.ErrInvalidCsrf
	}
	return nil
}

// csrfRandomToken returns a URL-safe opaque token drawn from crypto/rand.
// Only its sha256 hash is ever persisted.
func csrfRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// csrfHashToken returns the hex-encoded sha256 of the raw token.
func csrfHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
