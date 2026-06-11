//go:build integration

package postgres

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

// registerUser creates a fresh account + session in a new project.
func registerUser(t *testing.T, ctx context.Context, projectID, email string) (*domain.Account, *domain.Session) {
	t.Helper()
	ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
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

	auth := NewAuthenticator(testDB, "")
	if _, err := auth.Admin(ctx, token); err == nil {
		t.Fatal("impersonation token must not authenticate as adminToken")
	}

	ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
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
	if _, err := auth.Admin(ctx, token); err == nil {
		t.Fatal("redeemed impersonation token must not authenticate as adminToken")
	}
}

func TestAdminTokenMintUsesRequestMetadata(t *testing.T) {
	ctx := context.Background()
	operator := NewPgOperator(testDB, nopEmitter{})
	project, err := operator.CreateProject(ctx, domain.ProjectCmd{Name: "Admin Token Metadata"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	wantExpires := nowUTC().Add(2 * time.Hour).Truncate(time.Second)
	token, expiresAt, err := operator.MintAdminToken(ctx, domain.OperatorAdminTokenCmd{
		ProjectID: project.ID,
		Name:      "console",
		Scopes:    []string{"admin:ui", "users:read"},
		ExpiresAt: wantExpires,
	})
	if err != nil {
		t.Fatalf("mint admin token: %v", err)
	}
	if token == "" {
		t.Fatal("empty admin token")
	}
	if !expiresAt.Equal(wantExpires) {
		t.Fatalf("expiresAt = %s, want %s", expiresAt, wantExpires)
	}

	auth := NewAuthenticator(testDB, "")
	p, err := auth.Admin(ctx, token)
	if err != nil {
		t.Fatalf("admin auth: %v", err)
	}
	if p.ProjectID != project.ID {
		t.Fatalf("principal project = %q, want %q", p.ProjectID, project.ID)
	}
	if !slices.Equal(p.Scopes, []string{"admin:ui", "users:read"}) {
		t.Fatalf("scopes = %v", p.Scopes)
	}

	tokens, err := operator.ListAdminTokens(ctx, project.ID)
	if err != nil {
		t.Fatalf("list admin tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("len(tokens) = %d, want 1", len(tokens))
	}
	got := tokens[0]
	if got.Name != "console" {
		t.Fatalf("name = %q, want console", got.Name)
	}
	if !slices.Equal(got.Scopes, []string{"admin:ui", "users:read"}) {
		t.Fatalf("listed scopes = %v", got.Scopes)
	}
	if !got.ExpiresAt.Equal(wantExpires) {
		t.Fatalf("listed expiresAt = %s, want %s", got.ExpiresAt, wantExpires)
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
