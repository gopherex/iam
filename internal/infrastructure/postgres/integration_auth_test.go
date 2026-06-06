//go:build integration

package postgres

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// registerUser creates a fresh account + session in a new project.
func registerUser(t *testing.T, ctx context.Context, projectID, email string) (*domain.Account, *domain.Session) {
	t.Helper()
	ca := NewPgCoreAuth(testDB, nopEmitter{})
	acct, sess, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     email,
		Password:  "Sup3rStr0ng!Pass",
		Name:      "Test User",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return acct, sess
}

// Authenticator resolves a bearer access token to its principal.
func TestAuthenticatorResolvesUser(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	acct, sess := registerUser(t, ctx, projectID, "auth-user@example.com")

	auth := NewAuthenticator(testDB, "")
	p, err := auth.User(ctx, sess.AccessToken)
	if err != nil {
		t.Fatalf("authenticate user: %v", err)
	}
	if p.Kind != domain.PrincipalUser {
		t.Errorf("kind = %v, want user", p.Kind)
	}
	if p.AccountID != acct.ID {
		t.Errorf("account = %q, want %q", p.AccountID, acct.ID)
	}
	if p.ProjectID != projectID {
		t.Errorf("project = %q, want %q", p.ProjectID, projectID)
	}
	if p.Environment != "live" {
		t.Errorf("env = %q, want live", p.Environment)
	}

	// A garbage token must be rejected.
	if _, err := auth.User(ctx, "not.a.jwt"); err == nil {
		t.Fatal("expected rejection of a bogus token")
	}
}

// Impersonation redeem mints a session as the target user and is single-use.
func TestImpersonationRedeemSingleUse(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	target, _ := registerUser(t, ctx, projectID, "target@example.com")

	admin := NewPgAdminUsers(testDB, nopEmitter{})
	imp, err := admin.Impersonate(ctx, domain.AdminUserImpersonateCmd{
		ProjectID:       projectID,
		AccountID:       target.ID,
		ActorID:         "admin-1",
		Reason:          "support",
		DurationSeconds: 300,
	})
	if err != nil {
		t.Fatalf("impersonate: %v", err)
	}
	token := impersonationToken(t, imp.URL)

	ca := NewPgCoreAuth(testDB, nopEmitter{})
	acct, sess, err := ca.RedeemImpersonation(ctx, token, "client-1")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if acct.ID != target.ID {
		t.Errorf("redeemed account = %q, want target %q", acct.ID, target.ID)
	}
	if sess == nil || sess.AccessToken == "" {
		t.Fatal("redeem did not mint a session")
	}

	// Single-use: a second redemption must fail (the challenge was consumed).
	if _, _, err := ca.RedeemImpersonation(ctx, token, "client-1"); err == nil {
		t.Fatal("expected single-use failure on second redeem")
	}
}

func impersonationToken(t *testing.T, raw string) string {
	t.Helper()
	i := strings.IndexByte(raw, '?')
	if i < 0 {
		t.Fatalf("no query in impersonation url: %q", raw)
	}
	q, err := url.ParseQuery(raw[i+1:])
	if err != nil {
		t.Fatal(err)
	}
	tok := q.Get("token")
	if tok == "" {
		t.Fatalf("no token in impersonation url: %q", raw)
	}
	return tok
}
