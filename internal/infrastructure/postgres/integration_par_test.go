//go:build integration

package postgres

// integration_par_test.go — pushed authorization requests (RFC 9126).
//
// A pushed request is lodged over the authenticated back channel and redeemed
// once at /oauth2/authorize. Until this was wired the request_uri was written
// and never read, so the endpoint advertised in the discovery document did
// nothing and every pushed parameter — including code_challenge — was dropped.

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

func (f pkceFixture) push(ctx context.Context, mutate func(*domain.OIDCParCmd)) (*domain.OIDCParResult, error) {
	cmd := domain.OIDCParCmd{
		ClientID:            f.clientID,
		ResponseType:        "code",
		RedirectURI:         f.redirectURI,
		Scope:               "openid",
		State:               "st-par",
		CodeChallenge:       challengeFor(pkceVerifier),
		CodeChallengeMethod: "S256",
	}
	if mutate != nil {
		mutate(&cmd)
	}

	return f.grants.PushAuthorizationRequest(ctx, cmd)
}

// TestPARRoundTrip: a pushed request is redeemed at the authorization endpoint,
// and the parameters that reach the interaction are the pushed ones.
func TestPARRoundTrip(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	res, err := f.push(ctx, nil)
	if err != nil {
		t.Fatalf("PushAuthorizationRequest: %v", err)
	}

	if !strings.HasPrefix(res.RequestURI, "urn:ietf:params:oauth:request_uri:") {
		t.Fatalf("request_uri = %q, want the RFC 9126 URN form", res.RequestURI)
	}

	// Only client_id and request_uri are sent; everything else must come from the
	// pushed request. A junk redirect_uri in the query must not be honoured.
	redirect, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:    f.clientID,
		RequestURI:  res.RequestURI,
		RedirectURI: "https://evil.example.net/cb",
	})
	if err != nil {
		t.Fatalf("Authorize with request_uri: %v", err)
	}

	const prefix = "/oauth/interaction/"
	if !strings.HasPrefix(redirect, prefix) {
		t.Fatalf("Authorize returned %q, want an interaction handle", redirect)
	}

	in, err := f.grants.ResolveInteraction(ctx, strings.TrimPrefix(redirect, prefix))
	if err != nil {
		t.Fatalf("ResolveInteraction: %v", err)
	}

	if in.Interaction.RedirectURI != f.redirectURI {
		t.Fatalf("interaction redirect_uri = %q, want the pushed %q", in.Interaction.RedirectURI, f.redirectURI)
	}

	if in.Interaction.CodeChallenge != challengeFor(pkceVerifier) || in.Interaction.CodeChallengeMethod != "S256" {
		t.Fatalf("PKCE did not survive the push: challenge=%q method=%q",
			in.Interaction.CodeChallenge, in.Interaction.CodeChallengeMethod)
	}
}

// TestPARRequestURIIsSingleUse: the request_uri travels through the user-agent,
// so redeeming it twice must fail.
func TestPARRequestURIIsSingleUse(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	res, err := f.push(ctx, nil)
	if err != nil {
		t.Fatalf("PushAuthorizationRequest: %v", err)
	}

	if _, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID: f.clientID, RequestURI: res.RequestURI,
	}); err != nil {
		t.Fatalf("first redemption: %v", err)
	}

	_, err = f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID: f.clientID, RequestURI: res.RequestURI,
	})
	if !errors.Is(err, domain.ErrInvalidRequestURI) {
		t.Fatalf("second redemption = %v, want invalid_request_uri", err)
	}
}

func TestPARRequestURIRejections(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	t.Run("unknown_request_uri", func(t *testing.T) {
		_, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
			ClientID:   f.clientID,
			RequestURI: "urn:ietf:params:oauth:request_uri:nope",
		})
		if !errors.Is(err, domain.ErrInvalidRequestURI) {
			t.Fatalf("unknown request_uri = %v, want invalid_request_uri", err)
		}
	})

	t.Run("expired_request_uri", func(t *testing.T) {
		res, err := f.push(ctx, nil)
		if err != nil {
			t.Fatalf("PushAuthorizationRequest: %v", err)
		}

		if _, err := testDB.Pool.Exec(ctx,
			`UPDATE iam_par_requests SET expires_at = now() - interval '1 minute' WHERE request_uri = $1`,
			res.RequestURI); err != nil {
			t.Fatalf("age the pushed request: %v", err)
		}

		_, err = f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
			ClientID: f.clientID, RequestURI: res.RequestURI,
		})
		if !errors.Is(err, domain.ErrInvalidRequestURI) {
			t.Fatalf("expired request_uri = %v, want invalid_request_uri", err)
		}
	})

	// A request_uri belongs to the client that pushed it.
	t.Run("another_client_cannot_redeem", func(t *testing.T) {
		res, err := f.push(ctx, nil)
		if err != nil {
			t.Fatalf("PushAuthorizationRequest: %v", err)
		}

		other := newPKCEFixture(t, ctx, "spa")

		_, err = f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
			ClientID: other.clientID, RequestURI: res.RequestURI,
		})
		if !errors.Is(err, domain.ErrInvalidRequestURI) {
			t.Fatalf("redemption by another client = %v, want invalid_request_uri", err)
		}
	})
}

// TestPARValidatesAtPushTime: a request that could never be authorized is
// refused when it is pushed, not after the user has been sent to a login screen.
func TestPARValidatesAtPushTime(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	tests := []struct {
		name   string
		mutate func(*domain.OIDCParCmd)
		want   error
	}{
		{
			name:   "unknown client",
			mutate: func(c *domain.OIDCParCmd) { c.ClientID = "nope" },
			want:   domain.ErrInvalidClient,
		},
		{
			name:   "unregistered redirect_uri",
			mutate: func(c *domain.OIDCParCmd) { c.RedirectURI = "https://evil.example.net/cb" },
			want:   domain.ErrInvalidRedirectURI,
		},
		{
			name:   "unsupported response_type",
			mutate: func(c *domain.OIDCParCmd) { c.ResponseType = "token" },
			want:   domain.ErrBadRequest,
		},
		{
			name:   "missing scope",
			mutate: func(c *domain.OIDCParCmd) { c.Scope = "" },
			want:   domain.ErrBadRequest,
		},
		{
			name: "public client without PKCE",
			mutate: func(c *domain.OIDCParCmd) {
				c.CodeChallenge, c.CodeChallengeMethod = "", ""
			},
			want: domain.ErrBadRequest,
		},
		{
			name:   "plain PKCE method",
			mutate: func(c *domain.OIDCParCmd) { c.CodeChallengeMethod = "plain" },
			want:   domain.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.push(ctx, tt.mutate); !errors.Is(err, tt.want) {
				t.Fatalf("push = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestPARPushedRequestCompletesTheFlow: the code minted from a pushed request
// still requires the verifier that matches the pushed challenge.
func TestPARPushedRequestCompletesTheFlow(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")

	res, err := f.push(ctx, nil)
	if err != nil {
		t.Fatalf("PushAuthorizationRequest: %v", err)
	}

	redirect, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID: f.clientID, RequestURI: res.RequestURI,
	})
	if err != nil {
		t.Fatalf("Authorize with request_uri: %v", err)
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
		Remember:      true,
	})
	if err != nil {
		t.Fatalf("Consent: %v", err)
	}

	parsed, err := url.Parse(consent)
	if err != nil {
		t.Fatalf("parse consent redirect: %v", err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("consent redirect %q carries no code", consent)
	}

	if _, err := f.exchange(ctx, code, "ZZZftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); !errors.Is(err, domain.ErrInvalidGrant) {
		t.Fatalf("exchange with a wrong verifier = %v, want invalid_grant", err)
	}

	if _, err := f.exchange(ctx, code, pkceVerifier); err != nil {
		t.Fatalf("exchange with the pushed challenge's verifier: %v", err)
	}
}
