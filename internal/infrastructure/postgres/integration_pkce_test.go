//go:build integration

package postgres

// integration_pkce_test.go — PKCE (RFC 7636) end to end: the challenge sent to
// /oauth2/authorize is bound to the issued authorization code, and the token
// endpoint refuses to exchange that code without the matching verifier.

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// pkceFixture is a project with one registered app client and one user, plus the
// grants adapter under test.
type pkceFixture struct {
	grants       *pgOIDCGrants
	projectID    string
	userID       string
	clientID     string
	clientSecret string
	redirectURI  string
}

const pkceRedirectURI = "https://app.example.com/cb"

func newPKCEFixture(t *testing.T, ctx context.Context, clientType string) pkceFixture {
	t.Helper()

	op := NewPgOperator(testDB, nopEmitter{})
	proj, err := op.CreateProject(ctx, domain.ProjectCmd{Name: "pkce " + newUUID()[:8]})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	app, err := NewPgAdminApps(testDB, nopEmitter{}).Create(ctx, domain.AppClientCmd{
		ProjectID:    proj.ID,
		Environment:  "live",
		Name:         "pkce client",
		Type:         clientType,
		RedirectURIs: []string{pkceRedirectURI},
	})
	if err != nil {
		t.Fatalf("create app client: %v", err)
	}

	acc, err := NewPgAdminUsers(testDB, nopEmitter{}).Create(ctx, domain.RegisterCmd{
		ProjectID:   proj.ID,
		Environment: "live",
		Email:       "pkce+" + newUUID() + "@example.com",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	f := pkceFixture{
		grants:      NewPgOIDCGrants(testDB, nopEmitter{}, nil),
		projectID:   proj.ID,
		userID:      acc.ID,
		clientID:    app.ID,
		redirectURI: pkceRedirectURI,
	}

	// Confidential clients authenticate with a secret at the token endpoint.
	if clientType == "web" || clientType == "machine" {
		secret, err := NewPgAdminApps(testDB, nopEmitter{}).AddSecret(ctx, proj.ID, "live", app.ID, "e2e")
		if err != nil {
			t.Fatalf("add client secret: %v", err)
		}
		f.clientSecret = secret.ClientSecret
	}

	return f
}

// authorizeAndConsent drives authorize -> consent and returns the authorization
// code the client would receive on its redirect_uri.
func (f pkceFixture) authorizeAndConsent(t *testing.T, ctx context.Context, challenge, method string) string {
	t.Helper()

	redirect, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:            f.clientID,
		ResponseType:        "code",
		RedirectURI:         f.redirectURI,
		Scope:               "openid",
		CodeChallenge:       challenge,
		CodeChallengeMethod: method,
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	const prefix = "/oauth/interaction/"
	if len(redirect) <= len(prefix) || redirect[:len(prefix)] != prefix {
		t.Fatalf("Authorize returned %q, want an interaction handle", redirect)
	}

	interactionID := redirect[len(prefix):]

	// The interaction is created unbound; the authenticated user claims it at
	// login, and consent has to come from that same session and account.
	sessionID := "sess-" + newUUID()
	if err := f.grants.CompleteLogin(ctx, interactionID, f.userID, sessionID); err != nil {
		t.Fatalf("CompleteLogin: %v", err)
	}

	consentRedirect, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
		InteractionID: interactionID,
		AccountID:     f.userID,
		SessionID:     sessionID,
		GrantedScopes: []string{"openid"},
		Remember:      true,
	})
	if err != nil {
		t.Fatalf("Consent: %v", err)
	}

	parsed, err := url.Parse(consentRedirect)
	if err != nil {
		t.Fatalf("parse consent redirect %q: %v", consentRedirect, err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("consent redirect %q carries no code", consentRedirect)
	}

	return code
}

func (f pkceFixture) exchange(ctx context.Context, code, verifier string) (map[string]any, error) {
	return f.grants.Token(ctx, domain.OIDCTokenCmd{
		ProjectID:    f.projectID,
		Env:          "live",
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  f.redirectURI,
		CodeVerifier: verifier,
		ClientID:     f.clientID,
		ClientSecret: f.clientSecret,
	})
}

// TestPKCEExchangeRequiresMatchingVerifier is the attack PKCE exists to stop: an
// authorization code intercepted on the redirect is worthless without the
// verifier that never left the client.
func TestPKCEExchangeRequiresMatchingVerifier(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	challenge := challengeFor(pkceVerifier)

	t.Run("matching_verifier_exchanges", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, challenge, "S256")

		resp, err := f.exchange(ctx, code, pkceVerifier)
		if err != nil {
			t.Fatalf("exchange with the matching verifier: %v", err)
		}

		if access, _ := resp["access_token"].(string); access == "" {
			t.Fatalf("exchange returned no access_token: %v", resp)
		}
	})

	t.Run("missing_verifier_rejected", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, challenge, "S256")

		if _, err := f.exchange(ctx, code, ""); !errors.Is(err, domain.ErrInvalidGrant) {
			t.Fatalf("exchange without a verifier = %v, want invalid_grant", err)
		}
	})

	t.Run("wrong_verifier_rejected", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, challenge, "S256")

		other := "ZZZftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		if _, err := f.exchange(ctx, code, other); !errors.Is(err, domain.ErrInvalidGrant) {
			t.Fatalf("exchange with a wrong verifier = %v, want invalid_grant", err)
		}
	})

	// A rejected exchange must not burn the code: the legitimate client still
	// holds the real verifier and has to be able to finish its flow.
	t.Run("rejected_exchange_leaves_the_code_usable", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, challenge, "S256")

		if _, err := f.exchange(ctx, code, ""); err == nil {
			t.Fatal("exchange without a verifier succeeded")
		}

		if _, err := f.exchange(ctx, code, pkceVerifier); err != nil {
			t.Fatalf("exchange with the correct verifier after a failed attempt: %v", err)
		}
	})
}

// TestPKCEAuthorizeRequiresChallengeForPublicClients: a public client holds no
// secret, so an authorization request without PKCE is refused — bounced back to
// the client's registered redirect_uri as RFC 6749 §4.1.2.1 requires.
func TestPKCEAuthorizeRequiresChallengeForPublicClients(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	redirect, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:     f.clientID,
		ResponseType: "code",
		RedirectURI:  f.redirectURI,
		Scope:        "openid",
		State:        "st-1",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatalf("parse redirect %q: %v", redirect, err)
	}

	if got := parsed.Query().Get("error"); got != "invalid_request" {
		t.Fatalf("error = %q, want invalid_request (redirect was %q)", got, redirect)
	}

	if got := parsed.Query().Get("state"); got != "st-1" {
		t.Fatalf("state = %q, want st-1", got)
	}

	if n := e2eInteractionCount(t, ctx, f.clientID); n != 0 {
		t.Fatalf("a PKCE-less request created %d interaction(s); want 0", n)
	}
}

// TestPKCEConfidentialClientMayOmitChallenge: a confidential client
// authenticates with a secret at the token endpoint, so PKCE stays optional for
// it — but a code issued without a challenge cannot be redeemed WITH a verifier
// (downgrade protection, RFC 7636 §4.6).
func TestPKCEConfidentialClientMayOmitChallenge(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "web")

	t.Run("exchange_without_pkce", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, "", "")

		resp, err := f.grants.Token(ctx, domain.OIDCTokenCmd{
			ProjectID:    f.projectID,
			Env:          "live",
			GrantType:    "authorization_code",
			Code:         code,
			RedirectURI:  f.redirectURI,
			ClientID:     f.clientID,
			ClientSecret: f.clientSecret,
		})
		if err != nil {
			t.Fatalf("exchange without PKCE: %v", err)
		}

		if access, _ := resp["access_token"].(string); access == "" {
			t.Fatalf("exchange returned no access_token: %v", resp)
		}
	})

	t.Run("verifier_for_a_pkce_less_code_rejected", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, "", "")

		if _, err := f.exchange(ctx, code, pkceVerifier); !errors.Is(err, domain.ErrInvalidGrant) {
			t.Fatalf("exchange = %v, want invalid_grant", err)
		}
	})
}

// TestOIDCConfidentialClientSecret: the token endpoint authenticates a
// confidential client against the secrets issued for it (iam_app_secrets), and
// refuses a wrong or missing one.
func TestOIDCConfidentialClientSecret(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "web")

	t.Run("issued_secret_authenticates", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, "", "")

		if _, err := f.grants.Token(ctx, domain.OIDCTokenCmd{
			ProjectID: f.projectID, Env: "live", GrantType: "authorization_code",
			Code: code, RedirectURI: f.redirectURI,
			ClientID: f.clientID, ClientSecret: f.clientSecret,
		}); err != nil {
			t.Fatalf("exchange with the issued secret: %v", err)
		}
	})

	t.Run("wrong_secret_rejected", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, "", "")

		if _, err := f.grants.Token(ctx, domain.OIDCTokenCmd{
			ProjectID: f.projectID, Env: "live", GrantType: "authorization_code",
			Code: code, RedirectURI: f.redirectURI,
			ClientID: f.clientID, ClientSecret: "not-the-secret",
		}); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("exchange with a wrong secret = %v, want unauthorized", err)
		}
	})

	t.Run("missing_secret_rejected", func(t *testing.T) {
		code := f.authorizeAndConsent(t, ctx, "", "")

		if _, err := f.grants.Token(ctx, domain.OIDCTokenCmd{
			ProjectID: f.projectID, Env: "live", GrantType: "authorization_code",
			Code: code, RedirectURI: f.redirectURI, ClientID: f.clientID,
		}); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("exchange without a secret = %v, want unauthorized", err)
		}
	})

	// Rotation: a second secret is issued alongside the first, and both work
	// until the old one is dropped.
	t.Run("rotated_secret_also_authenticates", func(t *testing.T) {
		second, err := NewPgAdminApps(testDB, nopEmitter{}).AddSecret(ctx, f.projectID, "live", f.clientID, "rotated")
		if err != nil {
			t.Fatalf("add second secret: %v", err)
		}

		for name, secret := range map[string]string{"old": f.clientSecret, "new": second.ClientSecret} {
			code := f.authorizeAndConsent(t, ctx, "", "")
			if _, err := f.grants.Token(ctx, domain.OIDCTokenCmd{
				ProjectID: f.projectID, Env: "live", GrantType: "authorization_code",
				Code: code, RedirectURI: f.redirectURI,
				ClientID: f.clientID, ClientSecret: secret,
			}); err != nil {
				t.Fatalf("exchange with the %s secret: %v", name, err)
			}
		}
	})
}

// TestOIDCScopeClaimsFollowConsent: profile and email claims appear only for the
// scopes the client was granted. A relying party needs `email` to name the user,
// and a client that was not granted it must not receive it anyway.
func TestOIDCScopeClaimsFollowConsent(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	// Give the account something to disclose.
	if _, err := NewPgAdminUsers(testDB, nopEmitter{}).Update(ctx, domain.AdminUserUpdateCmd{
		ProjectID:   f.projectID,
		Environment: "live",
		AccountID:   f.userID,
		Name:        "Ada Lovelace",
	}); err != nil {
		t.Fatalf("set user name: %v", err)
	}

	claimsFor := func(t *testing.T, scopes []string) map[string]any {
		t.Helper()
		resp, err := f.grants.mintTokenResponse(ctx, oidcTokenSubject{
			projectID: f.projectID,
			env:       "live",
			subject:   f.userID,
			clientID:  f.clientID,
			scopes:    scopes,
		})
		if err != nil {
			t.Fatalf("mintTokenResponse: %v", err)
		}
		idToken, _ := resp["id_token"].(string)
		if idToken == "" {
			t.Fatalf("no id_token for scopes %v", scopes)
		}
		claims := testDB.Signer().UnverifiedClaims(idToken)
		if claims == nil {
			t.Fatal("id_token has no readable claims")
		}
		return claims
	}

	t.Run("email_scope_discloses_email", func(t *testing.T) {
		claims := claimsFor(t, []string{"openid", "email"})
		if got, _ := claims["email"].(string); got == "" {
			t.Fatalf("no email claim with the email scope: %v", claims)
		}
		if _, ok := claims["email_verified"]; !ok {
			t.Error("email disclosed without email_verified")
		}
		if _, ok := claims["name"]; ok {
			t.Error("name disclosed without the profile scope")
		}
	})

	t.Run("profile_scope_discloses_name", func(t *testing.T) {
		claims := claimsFor(t, []string{"openid", "profile"})
		if got, _ := claims["name"].(string); got != "Ada Lovelace" {
			t.Fatalf("name claim = %q, want the account name", got)
		}
		if _, ok := claims["email"]; ok {
			t.Error("email disclosed without the email scope")
		}
	})

	t.Run("bare_openid_discloses_nothing", func(t *testing.T) {
		claims := claimsFor(t, []string{"openid"})
		for _, key := range []string{"email", "email_verified", "name", "phone_number"} {
			if _, ok := claims[key]; ok {
				t.Errorf("%s disclosed to a client granted only openid", key)
			}
		}
	})
}

// TestOIDCAuthorizationResponseCarriesState: RFC 6749 §4.1.2 requires `state`
// back on the authorization response. Relying parties use it as their CSRF
// defence — oauth2-proxy refuses the callback outright without it — so dropping
// it makes the provider unusable no matter how correct everything else is.
func TestOIDCAuthorizationResponseCarriesState(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	const state = "opaque-client-state-123"

	redirect, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:            f.clientID,
		ResponseType:        "code",
		RedirectURI:         f.redirectURI,
		Scope:               "openid",
		State:               state,
		CodeChallenge:       challengeFor(pkceVerifier),
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	interactionID := strings.TrimPrefix(redirect, "/oauth/interaction/")

	sessionID := "sess-" + newUUID()
	if err := f.grants.CompleteLogin(ctx, interactionID, f.userID, sessionID); err != nil {
		t.Fatalf("CompleteLogin: %v", err)
	}

	consent, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
		InteractionID: interactionID,
		AccountID:     f.userID,
		SessionID:     sessionID,
		GrantedScopes: []string{"openid"},
	})
	if err != nil {
		t.Fatalf("Consent: %v", err)
	}

	parsed, err := url.Parse(consent)
	if err != nil {
		t.Fatalf("parse consent redirect: %v", err)
	}

	if got := parsed.Query().Get("state"); got != state {
		t.Fatalf("state = %q, want %q — the client will reject this callback", got, state)
	}
	if parsed.Query().Get("code") == "" {
		t.Fatal("no code alongside the state")
	}
}

// TestOIDCAccessTokenIsSelfVerifiable: the provider must accept its own tokens.
// The verifier selects a signing key by the `pid` claim, and OIDC-minted tokens
// carried none — so /oauth2/userinfo answered 401 to a token this very service
// had just issued, and any relying party resolving claims there gave up.
func TestOIDCAccessTokenIsSelfVerifiable(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	resp, err := f.grants.mintTokenResponse(ctx, oidcTokenSubject{
		projectID: f.projectID,
		env:       "live",
		subject:   f.userID,
		clientID:  f.clientID,
		scopes:    []string{"openid", "email"},
	})
	if err != nil {
		t.Fatalf("mintTokenResponse: %v", err)
	}

	access, _ := resp["access_token"].(string)
	if access == "" {
		t.Fatal("no access_token")
	}

	principal, err := NewAuthenticator(testDB, e2eMasterKey).OAuth2(ctx, access)
	if err != nil {
		t.Fatalf("the provider rejected its own access token: %v", err)
	}

	if principal.ProjectID != f.projectID {
		t.Errorf("principal project = %q, want %q", principal.ProjectID, f.projectID)
	}
	if principal.AccountID != f.userID {
		t.Errorf("principal account = %q, want %q", principal.AccountID, f.userID)
	}
	if len(principal.Scopes) == 0 {
		t.Error("principal carries no scopes; userinfo cannot gate its claims")
	}
}
