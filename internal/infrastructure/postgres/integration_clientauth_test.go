//go:build integration

package postgres

// integration_clientauth_test.go — private_key_jwt (RFC 7523 §3).
//
// The client proves itself by signing a short-lived assertion with a key only it
// holds, so no shared secret exists to leak from our storage or from the
// client's configuration. FAPI requires it; before this the parameters were not
// even accepted.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/gopherex/iam/internal/domain"
)

// clientKeyPair is a client's signing key plus the public JWKS it publishes.
type clientKeyPair struct {
	private jwk.Key
	jwks    string
}

func newClientKeyPair(t *testing.T) clientKeyPair {
	t.Helper()

	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	priv, err := jwk.Import(raw)
	if err != nil {
		t.Fatalf("import key: %v", err)
	}
	if err := priv.Set(jwk.KeyIDKey, "client-key-1"); err != nil {
		t.Fatalf("set kid: %v", err)
	}
	if err := priv.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		t.Fatalf("set alg: %v", err)
	}

	pub, err := jwk.PublicKeyOf(priv)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}

	set := jwk.NewSet()
	if err := set.AddKey(pub); err != nil {
		t.Fatalf("add key: %v", err)
	}

	encoded, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	return clientKeyPair{private: priv, jwks: string(encoded)}
}

// assertion builds a client assertion. Fields can be overridden to exercise the
// checks individually.
func (k clientKeyPair) assertion(t *testing.T, clientID, audience string, mutate func(jwt.Token)) string {
	t.Helper()

	tok, err := jwt.NewBuilder().
		Issuer(clientID).
		Subject(clientID).
		Audience([]string{audience}).
		JwtID(newUUID()).
		IssuedAt(nowUTC()).
		Expiration(nowUTC().Add(time.Minute)).
		Build()
	if err != nil {
		t.Fatalf("build assertion: %v", err)
	}

	if mutate != nil {
		mutate(tok)
	}

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), k.private))
	if err != nil {
		t.Fatalf("sign assertion: %v", err)
	}

	return string(signed)
}

// assertionFixture is a confidential client that authenticates by assertion.
type assertionFixture struct {
	pkceFixture
	keys     clientKeyPair
	audience string
}

func newAssertionFixture(t *testing.T, ctx context.Context) assertionFixture {
	t.Helper()
	f := newPKCEFixture(t, ctx, "web")
	keys := newClientKeyPair(t)

	if _, err := NewPgAdminApps(testDB, nopEmitter{}).Update(ctx, f.projectID, "live", f.clientID,
		map[string]any{"jwks": keys.jwks}); err != nil {
		t.Fatalf("publish client jwks: %v", err)
	}

	return assertionFixture{
		pkceFixture: f,
		keys:        keys,
		audience:    testDB.PublicURL + "/oauth2/token",
	}
}

// exchangeWithAssertion redeems an authorization code using private_key_jwt.
func (f assertionFixture) exchangeWithAssertion(
	ctx context.Context, t *testing.T, code, assertion string,
) (map[string]any, error) {
	t.Helper()
	return f.grants.Token(ctx, domain.OIDCTokenCmd{
		GrantType:           "authorization_code",
		Code:                code,
		RedirectURI:         f.redirectURI,
		CodeVerifier:        pkceVerifier,
		ClientAssertionType: oidcClientAssertionType,
		ClientAssertion:     assertion,
	})
}

// TestPrivateKeyJWTAuthenticatesAClient: the happy path — no secret anywhere.
func TestPrivateKeyJWTAuthenticatesAClient(t *testing.T) {
	ctx := context.Background()
	f := newAssertionFixture(t, ctx)

	code := f.authorizeAndConsent(t, ctx, challengeFor(pkceVerifier), "S256")

	resp, err := f.exchangeWithAssertion(ctx, t, code,
		f.keys.assertion(t, f.clientID, f.audience, nil))
	if err != nil {
		t.Fatalf("exchange with a client assertion: %v", err)
	}

	if access, _ := resp["access_token"].(string); access == "" {
		t.Fatalf("no access token: %v", resp)
	}
}

// TestPrivateKeyJWTIsSingleUse: a captured assertion must not be a second
// authentication.
func TestPrivateKeyJWTIsSingleUse(t *testing.T) {
	ctx := context.Background()
	f := newAssertionFixture(t, ctx)

	assertion := f.keys.assertion(t, f.clientID, f.audience, nil)

	code := f.authorizeAndConsent(t, ctx, challengeFor(pkceVerifier), "S256")
	if _, err := f.exchangeWithAssertion(ctx, t, code, assertion); err != nil {
		t.Fatalf("first exchange: %v", err)
	}

	code = f.authorizeAndConsent(t, ctx, challengeFor(pkceVerifier), "S256")
	if _, err := f.exchangeWithAssertion(ctx, t, code, assertion); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("replayed assertion = %v, want unauthorized", err)
	}
}

// TestPrivateKeyJWTRejections covers what must not authenticate.
func TestPrivateKeyJWTRejections(t *testing.T) {
	ctx := context.Background()
	f := newAssertionFixture(t, ctx)
	other := newClientKeyPair(t)

	tests := []struct {
		name      string
		assertion func(t *testing.T) string
	}{
		{
			// Signed by a key the client never published.
			name: "unknown key",
			assertion: func(t *testing.T) string {
				return other.assertion(t, f.clientID, f.audience, nil)
			},
		},
		{
			// Addressed to somebody else — a captured assertion must not be
			// replayable against a different endpoint.
			name: "wrong audience",
			assertion: func(t *testing.T) string {
				return f.keys.assertion(t, f.clientID, "https://elsewhere.example.com/token", nil)
			},
		},
		{
			// iss and sub must both be the client (RFC 7523 §3).
			name: "subject is not the issuer",
			assertion: func(t *testing.T) string {
				return f.keys.assertion(t, f.clientID, f.audience, func(tok jwt.Token) {
					_ = tok.Set(jwt.SubjectKey, "somebody-else")
				})
			},
		},
		{
			// A long-lived assertion is a bearer credential in all but name.
			name: "expires too far ahead",
			assertion: func(t *testing.T) string {
				return f.keys.assertion(t, f.clientID, f.audience, func(tok jwt.Token) {
					_ = tok.Set(jwt.ExpirationKey, nowUTC().Add(24*time.Hour))
				})
			},
		},
		{
			name: "already expired",
			assertion: func(t *testing.T) string {
				return f.keys.assertion(t, f.clientID, f.audience, func(tok jwt.Token) {
					_ = tok.Set(jwt.ExpirationKey, nowUTC().Add(-time.Minute))
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := f.authorizeAndConsent(t, ctx, challengeFor(pkceVerifier), "S256")

			_, err := f.exchangeWithAssertion(ctx, t, code, tt.assertion(t))
			if !errors.Is(err, domain.ErrUnauthorized) {
				t.Fatalf("exchange = %v, want unauthorized", err)
			}
		})
	}
}

// TestPrivateKeyJWTNeedsPublishedKeys: a client that published none cannot be
// authenticated this way, and is told so rather than failing obscurely.
func TestPrivateKeyJWTNeedsPublishedKeys(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "web")
	keys := newClientKeyPair(t)

	code := f.authorizeAndConsent(t, ctx, challengeFor(pkceVerifier), "S256")

	_, err := f.grants.Token(ctx, domain.OIDCTokenCmd{
		GrantType:           "authorization_code",
		Code:                code,
		RedirectURI:         f.redirectURI,
		CodeVerifier:        pkceVerifier,
		ClientAssertionType: oidcClientAssertionType,
		ClientAssertion:     keys.assertion(t, f.clientID, testDB.PublicURL+"/oauth2/token", nil),
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("exchange = %v, want unauthorized", err)
	}
}

// requestObject builds a signed authorization request (RFC 9101).
func (k clientKeyPair) requestObject(t *testing.T, clientID string, params map[string]any) string {
	t.Helper()

	builder := jwt.NewBuilder().Issuer(clientID).Audience([]string{"iam"})
	for key, value := range params {
		builder = builder.Claim(key, value)
	}

	tok, err := builder.Build()
	if err != nil {
		t.Fatalf("build request object: %v", err)
	}

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), k.private))
	if err != nil {
		t.Fatalf("sign request object: %v", err)
	}

	return string(signed)
}

// TestRequestObjectWinsOverTheQuery is the whole point of JAR: a query string an
// attacker can rewrite must not be able to change what the client asked for.
func TestRequestObjectWinsOverTheQuery(t *testing.T) {
	ctx := context.Background()
	f := newAssertionFixture(t, ctx)

	object := f.keys.requestObject(t, f.clientID, map[string]any{
		"client_id":             f.clientID,
		"response_type":         "code",
		"redirect_uri":          f.redirectURI,
		"scope":                 "openid",
		"state":                 "from-the-object",
		"code_challenge":        challengeFor(pkceVerifier),
		"code_challenge_method": "S256",
	})

	// The query says something else entirely; the object must win.
	redirect, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:     f.clientID,
		Request:      object,
		ResponseType: "token",
		RedirectURI:  "https://evil.example.net/cb",
		Scope:        "openid groups",
		State:        "from-the-query",
	})
	if err != nil {
		t.Fatalf("Authorize with a request object: %v", err)
	}

	const prefix = "/oauth/interaction/"
	if len(redirect) <= len(prefix) || redirect[:len(prefix)] != prefix {
		t.Fatalf("Authorize returned %q, want an interaction — the query's response_type=token should have been ignored", redirect)
	}

	in, err := f.grants.ResolveInteraction(ctx, redirect[len(prefix):])
	if err != nil {
		t.Fatalf("ResolveInteraction: %v", err)
	}

	if in.Interaction.RedirectURI != f.redirectURI {
		t.Fatalf("redirect_uri = %q, want the object's %q", in.Interaction.RedirectURI, f.redirectURI)
	}
	if in.Interaction.State != "from-the-object" {
		t.Fatalf("state = %q, want the object's value", in.Interaction.State)
	}
	if in.Interaction.CodeChallenge != challengeFor(pkceVerifier) {
		t.Fatal("the object's PKCE challenge did not survive")
	}
}

// TestRequestObjectRejections: an object that does not verify, or speaks for a
// different client, is refused outright rather than partially honored.
func TestRequestObjectRejections(t *testing.T) {
	ctx := context.Background()
	f := newAssertionFixture(t, ctx)
	other := newClientKeyPair(t)

	base := map[string]any{
		"client_id":             f.clientID,
		"response_type":         "code",
		"redirect_uri":          f.redirectURI,
		"scope":                 "openid",
		"code_challenge":        challengeFor(pkceVerifier),
		"code_challenge_method": "S256",
	}

	t.Run("signed by an unknown key", func(t *testing.T) {
		_, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
			ClientID: f.clientID,
			Request:  other.requestObject(t, f.clientID, base),
		})
		if !errors.Is(err, domain.ErrBadRequest) {
			t.Fatalf("Authorize = %v, want bad_request", err)
		}
	})

	t.Run("names a different client", func(t *testing.T) {
		params := map[string]any{}
		for k, v := range base {
			params[k] = v
		}
		params["client_id"] = "someone-else"

		_, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
			ClientID: f.clientID,
			Request:  f.keys.requestObject(t, f.clientID, params),
		})
		if !errors.Is(err, domain.ErrBadRequest) {
			t.Fatalf("Authorize = %v, want bad_request", err)
		}
	})
}
