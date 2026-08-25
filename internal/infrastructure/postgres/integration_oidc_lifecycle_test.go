//go:build integration

package postgres

// integration_oidc_lifecycle_test.go — the state that makes an issued token
// killable: RFC 7009 revocation, refresh rotation and reuse detection.
//
// Before this, /oauth2/revoke returned 200 and revoked nothing but the
// authorization code, and an OIDC refresh token was a bare signature: replayable
// for its whole lifetime with nothing able to stop it.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-faster/jx"

	"github.com/gopherex/iam/internal/domain"
)

// mustConfigDoc renders a plain map into the raw-JSON doc shape the config
// adapter persists.
func mustConfigDoc(t *testing.T, fields map[string]any) domain.AdminConfigDoc {
	t.Helper()
	doc := domain.AdminConfigDoc{}
	for k, v := range fields {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", k, err)
		}
		doc[k] = jx.Raw(raw)
	}
	return doc
}

// oidcGrant mints a token set the way the authorization code path does, with a
// session behind it.
func (f pkceFixture) oidcGrant(t *testing.T, ctx context.Context, sessionID string, scopes ...string) map[string]any {
	t.Helper()
	if len(scopes) == 0 {
		scopes = []string{"openid", "offline_access"}
	}
	resp, err := f.grants.mintTokenResponse(ctx, oidcTokenSubject{
		projectID: f.projectID,
		env:       "live",
		subject:   f.userID,
		clientID:  f.clientID,
		scopes:    scopes,
		sessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("mintTokenResponse: %v", err)
	}
	return resp
}

func (f pkceFixture) refreshWith(ctx context.Context, token string) (map[string]any, error) {
	return f.grants.Token(ctx, domain.OIDCTokenCmd{
		ProjectID:             f.projectID,
		Env:                   "live",
		GrantType:             "refresh_token",
		RefreshToken:          token,
		AuthenticatedClientID: f.clientID,
	})
}

// TestOIDCRefreshRotates: each exchange spends the presented token and hands
// back a new one.
func TestOIDCRefreshRotates(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	first := f.oidcGrant(t, ctx, "sess-"+newUUID())
	refresh, _ := first["refresh_token"].(string)
	if refresh == "" {
		t.Fatal("offline_access produced no refresh token")
	}

	next, err := f.refreshWith(ctx, refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	rotated, _ := next["refresh_token"].(string)
	if rotated == "" {
		t.Fatal("refresh returned no new refresh token")
	}
	if rotated == refresh {
		t.Fatal("the refresh token was not rotated; a leak stays usable forever")
	}
	if access, _ := next["access_token"].(string); access == "" {
		t.Fatal("refresh returned no access token")
	}
}

// TestOIDCRefreshReuseBurnsTheSession: replaying a spent token is the signal it
// leaked — the legitimate holder has moved on — so the whole session's tokens
// die rather than just the replayed one (RFC 9700 §4.14.2).
func TestOIDCRefreshReuseBurnsTheSession(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	sessionID := "sess-" + newUUID()

	first := f.oidcGrant(t, ctx, sessionID)
	refresh, _ := first["refresh_token"].(string)

	next, err := f.refreshWith(ctx, refresh)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	rotated, _ := next["refresh_token"].(string)

	// Replay the spent one.
	if _, err := f.refreshWith(ctx, refresh); !errors.Is(err, domain.ErrTokenUsed) {
		t.Fatalf("replay = %v, want token_already_used", err)
	}

	// And the rotated one is dead too: the family was burned.
	if _, err := f.refreshWith(ctx, rotated); err == nil {
		t.Fatal("the rotated token still works after a replay was detected")
	}
}

// TestOIDCRevokeKillsRefreshToken: RFC 7009 on a refresh token must actually
// stop it.
func TestOIDCRevokeKillsRefreshToken(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	grant := f.oidcGrant(t, ctx, "sess-"+newUUID())
	refresh, _ := grant["refresh_token"].(string)

	if err := f.grants.Revoke(ctx, domain.OIDCRevokeCmd{
		ProjectID: f.projectID, Env: "live", Token: refresh, TokenTypeHint: "refresh_token",
	}); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if _, err := f.refreshWith(ctx, refresh); err == nil {
		t.Fatal("a revoked refresh token still bought new tokens")
	}
}

// TestOIDCRevokeKillsAccessToken: an access token is verified offline, so
// revocation only means something if the verification path consults a denylist.
func TestOIDCRevokeKillsAccessToken(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	grant := f.oidcGrant(t, ctx, "sess-"+newUUID())
	access, _ := grant["access_token"].(string)

	auth := NewAuthenticator(testDB, e2eMasterKey)
	if _, err := auth.OAuth2(ctx, access); err != nil {
		t.Fatalf("freshly minted access token rejected: %v", err)
	}

	if err := f.grants.Revoke(ctx, domain.OIDCRevokeCmd{
		ProjectID: f.projectID, Env: "live", Token: access, TokenTypeHint: "access_token",
	}); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if _, err := auth.OAuth2(ctx, access); !errors.Is(err, domain.ErrTokenRevoked) {
		t.Fatalf("revoked access token = %v, want token_revoked", err)
	}

	// Introspection must agree: a revoked token is not active.
	res, err := f.grants.Introspect(ctx, domain.OIDCIntrospectCmd{
		ProjectID: f.projectID, Env: "live", Token: access,
	})
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}

	if active, _ := res["active"].(bool); active {
		t.Fatal("introspection reports a revoked token as active")
	}
}

// TestOIDCRefreshBoundToItsClient: a token issued to one client cannot be
// redeemed by another, even one that authenticated correctly as itself.
func TestOIDCRefreshBoundToItsClient(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	other := newPKCEFixture(t, ctx, "spa")

	grant := f.oidcGrant(t, ctx, "sess-"+newUUID())
	refresh, _ := grant["refresh_token"].(string)

	_, err := f.grants.Token(ctx, domain.OIDCTokenCmd{
		ProjectID:             f.projectID,
		Env:                   "live",
		GrantType:             "refresh_token",
		RefreshToken:          refresh,
		AuthenticatedClientID: other.clientID,
	})
	if !errors.Is(err, domain.ErrInvalidGrant) {
		t.Fatalf("cross-client refresh = %v, want invalid_grant", err)
	}
}

// TestOIDCTokenTTLFollowsSessionPolicy: the provider's own tokens honour the
// project's configured lifetimes, not constants compiled into the adapter.
func TestOIDCTokenTTLFollowsSessionPolicy(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	cfgAdmin := NewPgAdminConfig(testDB, nopEmitter{})
	if _, err := cfgAdmin.UpdateSessionPolicy(ctx, domain.AdminConfigUpdateCmd{
		ProjectID:   f.projectID,
		Environment: "live",
		Doc:         mustConfigDoc(t, map[string]any{"access_ttl": 1200, "refresh_ttl": 7 * 24 * 3600}),
	}); err != nil {
		t.Fatalf("set session policy: %v", err)
	}

	grants := NewPgOIDCGrants(testDB, nopEmitter{}, NewConfigReader(testDB, 0))

	resp, err := grants.mintTokenResponse(ctx, oidcTokenSubject{
		projectID: f.projectID,
		env:       "live",
		subject:   f.userID,
		clientID:  f.clientID,
		scopes:    []string{"openid"},
		sessionID: "sess-" + newUUID(),
	})
	if err != nil {
		t.Fatalf("mintTokenResponse: %v", err)
	}

	expiresIn, _ := resp["expires_in"].(float64)
	if expiresIn == 0 {
		if v, ok := resp["expires_in"].(uint64); ok {
			expiresIn = float64(v)
		}
	}

	if int(expiresIn) != 1200 {
		t.Fatalf("expires_in = %v, want the configured access_ttl 1200", resp["expires_in"])
	}
}
