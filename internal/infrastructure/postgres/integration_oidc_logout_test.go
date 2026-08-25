//go:build integration

package postgres

// integration_oidc_logout_test.go — RP-initiated logout and back-channel logout.
//
// Logout used to validate the id_token_hint, emit an event and end nothing: the
// session, its tokens and the browser cookie all survived. It also followed
// whatever post_logout_redirect_uri it was handed. And back-channel logout was
// advertised in the discovery document while no logout token was ever sent.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// logoutFixture is a project with a client, a user and a live session that has
// OIDC tokens issued against it.
type logoutFixture struct {
	pkceFixture
	sessionID string
	idToken   string
}

func newLogoutFixture(t *testing.T, ctx context.Context) logoutFixture {
	t.Helper()
	f := newPKCEFixture(t, ctx, "web")

	// A real session row, so logout has something to end.
	acct, session, err := NewPgCoreAuth(testDB, nopEmitter{}, nil).Register(ctx, domain.RegisterCmd{
		ProjectID: f.projectID,
		Email:     "logout+" + newUUID() + "@example.com",
		Password:  "Sup3rStr0ng!Pass",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	f.userID = acct.ID

	resp, err := f.grants.mintTokenResponse(ctx, oidcTokenSubject{
		projectID: f.projectID,
		env:       "live",
		subject:   acct.ID,
		clientID:  f.clientID,
		scopes:    []string{"openid", "offline_access"},
		sessionID: session.ID,
	})
	if err != nil {
		t.Fatalf("mintTokenResponse: %v", err)
	}

	idToken, _ := resp["id_token"].(string)
	if idToken == "" {
		t.Fatal("no id_token to log out with")
	}

	return logoutFixture{pkceFixture: f, sessionID: session.ID, idToken: idToken}
}

func sessionExists(t *testing.T, ctx context.Context, sessionID string) bool {
	t.Helper()
	_, err := models.FindIamSession(ctx, testDB.Bobx(), sessionID)
	return err == nil
}

// TestOIDCLogoutEndsTheSession: the point of logging out.
func TestOIDCLogoutEndsTheSession(t *testing.T) {
	ctx := context.Background()
	f := newLogoutFixture(t, ctx)

	if !sessionExists(t, ctx, f.sessionID) {
		t.Fatal("fixture session missing before logout")
	}

	res, err := f.grants.Logout(ctx, domain.OIDCLogoutCmd{IDTokenHint: f.idToken})
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if sessionExists(t, ctx, f.sessionID) {
		t.Fatal("the session survived the logout")
	}

	if !res.ClearSessionCookies {
		t.Fatal("logout did not ask to clear the browser session cookies")
	}
}

// TestOIDCLogoutRejectsUnregisteredRedirect: post_logout_redirect_uri is
// attacker-controlled input; following it unchecked makes every logout link an
// open redirect.
func TestOIDCLogoutRejectsUnregisteredRedirect(t *testing.T) {
	ctx := context.Background()
	f := newLogoutFixture(t, ctx)

	res, err := f.grants.Logout(ctx, domain.OIDCLogoutCmd{
		IDTokenHint:           f.idToken,
		PostLogoutRedirectURI: "https://evil.example.net/landing",
		State:                 "st",
	})
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if res.RedirectURL != "/" {
		t.Fatalf("redirect = %q; an unregistered post-logout URI must not be followed", res.RedirectURL)
	}
}

// TestOIDCLogoutHonoursRegisteredRedirect: and a registered one is followed,
// with the client's state echoed back.
func TestOIDCLogoutHonoursRegisteredRedirect(t *testing.T) {
	ctx := context.Background()
	f := newLogoutFixture(t, ctx)

	const target = "https://app.example.com/signed-out"

	if _, err := NewPgAdminApps(testDB, nopEmitter{}).Update(ctx, f.projectID, "live", f.clientID,
		map[string]any{"post_logout_redirect_uris": []any{target}}); err != nil {
		t.Fatalf("register post-logout uri: %v", err)
	}

	res, err := f.grants.Logout(ctx, domain.OIDCLogoutCmd{
		IDTokenHint:           f.idToken,
		PostLogoutRedirectURI: target,
		State:                 "st-9",
	})
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	parsed, err := url.Parse(res.RedirectURL)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}

	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != target {
		t.Fatalf("redirect = %q, want the registered %q", got, target)
	}

	if got := parsed.Query().Get("state"); got != "st-9" {
		t.Fatalf("state = %q, want st-9", got)
	}
}

// TestOIDCLogoutWithoutHintChangesNothing: no hint means we know neither whose
// session to end nor who registered the redirect being asked for.
func TestOIDCLogoutWithoutHintChangesNothing(t *testing.T) {
	ctx := context.Background()
	f := newLogoutFixture(t, ctx)

	res, err := f.grants.Logout(ctx, domain.OIDCLogoutCmd{
		PostLogoutRedirectURI: "https://evil.example.net/landing",
	})
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if res.RedirectURL != "/" {
		t.Fatalf("redirect = %q, want / without an id_token_hint", res.RedirectURL)
	}

	if !sessionExists(t, ctx, f.sessionID) {
		t.Fatal("a hintless logout ended a session it could not identify")
	}
}

// TestBackchannelLogoutDelivered: the relying party is actually told, with a
// logout token it can verify.
func TestBackchannelLogoutDelivered(t *testing.T) {
	ctx := context.Background()
	f := newLogoutFixture(t, ctx)

	var (
		mu       sync.Mutex
		received []string
	)

	rp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		received = append(received, r.PostFormValue("logout_token"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer rp.Close()

	if _, err := NewPgAdminApps(testDB, nopEmitter{}).Update(ctx, f.projectID, "live", f.clientID,
		map[string]any{"backchannel_logout_uri": rp.URL}); err != nil {
		t.Fatalf("register backchannel uri: %v", err)
	}

	if err := DeliverBackchannelLogout(ctx, testDB, NewOutboundHTTPClient(5*time.Second),
		f.projectID, "live", f.sessionID, f.userID); err != nil {
		t.Fatalf("DeliverBackchannelLogout: %v", err)
	}

	mu.Lock()
	got := append([]string(nil), received...)
	mu.Unlock()

	if len(got) != 1 {
		t.Fatalf("relying party received %d logout tokens, want 1", len(got))
	}

	claims := testDB.Signer().UnverifiedClaims(got[0])
	if claims == nil {
		t.Fatal("logout token has no readable claims")
	}

	// aud is a string or an array of strings (RFC 7519 §4.1.3).
	if aud := audienceOf(claims); aud != f.clientID {
		t.Errorf("aud = %v, want the client %q", claims["aud"], f.clientID)
	}
	if sid, _ := claims["sid"].(string); sid != f.sessionID {
		t.Errorf("sid = %v, want the ended session", claims["sid"])
	}
	if sub, _ := claims["sub"].(string); sub != f.userID {
		t.Errorf("sub = %v, want the user", claims["sub"])
	}
	if _, present := claims["nonce"]; present {
		t.Error("a logout token must not carry nonce; that is how it is told from an id_token")
	}

	events, _ := claims["events"].(map[string]any)
	if _, ok := events[backchannelLogoutEventType]; !ok {
		raw, _ := json.Marshal(claims["events"])
		t.Errorf("events = %s, want the backchannel-logout event", raw)
	}
}

// TestBackchannelLogoutSkipsClientsWithoutURI: a client that did not ask to be
// notified is not contacted, and that is not a failure.
func TestBackchannelLogoutSkipsClientsWithoutURI(t *testing.T) {
	ctx := context.Background()
	f := newLogoutFixture(t, ctx)

	if err := DeliverBackchannelLogout(ctx, testDB, NewOutboundHTTPClient(5*time.Second),
		f.projectID, "live", f.sessionID, f.userID); err != nil {
		t.Fatalf("delivery to a client with no backchannel URI: %v", err)
	}
}

// audienceOf reads an `aud` claim in either of its permitted shapes.
func audienceOf(claims map[string]any) string {
	switch v := claims["aud"].(type) {
	case string:
		return v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
