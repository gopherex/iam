//go:build integration

package postgres

// integration_federation_session_test.go — the login session an enterprise SSO
// sign-in produces.
//
// Federation used to sign its own access and refresh tokens instead of going
// through the shared minting seam, so the refresh token was never recorded in
// iam_refresh_tokens. /v1/auth/token/refresh looks a refresh token up by hash
// and rejects what it cannot find, so an SSO session could not be extended at
// all: it expired with its access token, ten minutes in, and the 30-day refresh
// cookie the browser was holding was dead on arrival.

import (
	"context"
	"errors"
	"testing"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// fedSession provisions an SSO login the way a SAML/OIDC callback does.
func fedSession(t *testing.T, ctx context.Context, projectID string) *domain.Session {
	t.Helper()
	runtime := NewPgFederationRuntime(testDB, nopEmitter{}, NewConfigReader(testDB, 0))

	acct, err := runtime.fedCreateAccount(ctx, projectID, "sso+"+newUUID()+"@example.com")
	if err != nil {
		t.Fatalf("provision account: %v", err)
	}

	sess, err := runtime.fedMintSession(ctx, acct, "saml")
	if err != nil {
		t.Fatalf("fedMintSession: %v", err)
	}

	return sess
}

// TestFederationSessionCanRefresh is the bug: an SSO session has to be
// extendable like any other.
func TestFederationSessionCanRefresh(t *testing.T) {
	ctx := context.Background()
	projectID := e2eProject(t, ctx)

	sess := fedSession(t, ctx, projectID)
	if sess.RefreshToken == "" {
		t.Fatal("SSO login produced no refresh token")
	}

	_, rotated, err := NewPgCoreAuth(testDB, nopEmitter{}, nil).Refresh(ctx, sess.RefreshToken)
	if err != nil {
		t.Fatalf("refreshing an SSO session: %v", err)
	}

	if rotated.AccessToken == "" {
		t.Fatal("refresh returned no access token")
	}
	if rotated.RefreshToken == sess.RefreshToken {
		t.Fatal("the refresh token was not rotated")
	}
}

// TestFederationSessionIsRecorded: the refresh token exists as a row, which is
// what makes it rotatable, revocable and detectable when replayed.
func TestFederationSessionIsRecorded(t *testing.T) {
	ctx := context.Background()
	projectID := e2eProject(t, ctx)

	sess := fedSession(t, ctx, projectID)

	rows, err := models.IamRefreshTokens.Query(
		sm.Where(models.IamRefreshTokens.Columns.SessionID.EQ(psql.Arg(sess.ID))),
	).All(ctx, testDB.Bobx())
	if err != nil {
		t.Fatalf("read refresh tokens: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("SSO session has %d refresh-token rows, want 1", len(rows))
	}
}

// TestFederationSessionRevocationKillsRefresh: revoking the session invalidates
// its refresh token, which is only possible because the token is a record.
func TestFederationSessionRevocationKillsRefresh(t *testing.T) {
	ctx := context.Background()
	projectID := e2eProject(t, ctx)

	sess := fedSession(t, ctx, projectID)

	if _, err := NewPgAdminUsers(testDB, nopEmitter{}).RevokeSessions(ctx, domain.AdminUserSessionsRevokeCmd{
		ProjectID:   projectID,
		Environment: "live",
		AccountID:   sess.AccountID,
	}); err != nil {
		t.Fatalf("revoke sessions: %v", err)
	}

	_, _, err := NewPgCoreAuth(testDB, nopEmitter{}, nil).Refresh(ctx, sess.RefreshToken)
	if err == nil {
		t.Fatal("a revoked SSO session still refreshed")
	}
	if !errors.Is(err, domain.ErrTokenRevoked) && !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("refresh after revocation = %v, want a rejection", err)
	}
}

// TestFederationSessionTTLFollowsPolicy: lifetimes come from the project's
// session_policy, not from constants that used to live in the SSO adapter.
func TestFederationSessionTTLFollowsPolicy(t *testing.T) {
	ctx := context.Background()
	projectID := e2eProject(t, ctx)

	if _, err := NewPgAdminConfig(testDB, nopEmitter{}).UpdateSessionPolicy(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   projectID,
		Environment: "live",
		Doc:         mustConfigDoc(t, map[string]any{"access_ttl": 900, "refresh_ttl": 3 * 24 * 3600}),
	}); err != nil {
		t.Fatalf("set session policy: %v", err)
	}

	sess := fedSession(t, ctx, projectID)

	if sess.ExpiresIn != 900 {
		t.Errorf("access lifetime = %d, want the configured 900", sess.ExpiresIn)
	}
	if sess.RefreshExpiresIn != 3*24*3600 {
		t.Errorf("refresh lifetime = %d, want the configured %d", sess.RefreshExpiresIn, 3*24*3600)
	}
}
