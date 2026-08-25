//go:build integration

package postgres

// integration_interaction_binding_test.go — who is allowed to drive an
// authorization interaction to completion.
//
// An interaction is created unbound by the public /oauth2/authorize endpoint and
// its id travels through the user-agent. The first authenticated caller claims
// it; everyone else must be refused, or the id alone would be enough to hand a
// client an authorization code bound to somebody else's account.

import (
	"context"
	"errors"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// startInteraction runs an authorization request and returns the interaction id.
func (f pkceFixture) startInteraction(t *testing.T, ctx context.Context) string {
	t.Helper()

	redirect, err := f.grants.Authorize(ctx, domain.OIDCAuthorizeCmd{
		ClientID:            f.clientID,
		ResponseType:        "code",
		RedirectURI:         f.redirectURI,
		Scope:               "openid",
		CodeChallenge:       challengeFor(pkceVerifier),
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	const prefix = "/oauth/interaction/"
	if len(redirect) <= len(prefix) || redirect[:len(prefix)] != prefix {
		t.Fatalf("Authorize returned %q, want an interaction handle", redirect)
	}

	return redirect[len(prefix):]
}

// TestInteractionConsentRequiresLogin: an interaction nobody has logged into
// cannot be consented. This is the case a sessionless principal (admin token,
// API key, client credential) used to satisfy, because its empty session id
// compared equal to the interaction's unset one.
func TestInteractionConsentRequiresLogin(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	interactionID := f.startInteraction(t, ctx)

	t.Run("sessionless_principal_refused", func(t *testing.T) {
		_, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
			InteractionID: interactionID,
			AccountID:     f.userID,
			SessionID:     "", // admin token / API key / client credential
			GrantedScopes: []string{"openid"},
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("consent without a session = %v, want forbidden", err)
		}
	})

	t.Run("unclaimed_interaction_refused", func(t *testing.T) {
		_, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
			InteractionID: interactionID,
			AccountID:     f.userID,
			SessionID:     "sess-never-logged-in",
			GrantedScopes: []string{"openid"},
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("consent on an unclaimed interaction = %v, want forbidden", err)
		}
	})
}

// TestInteractionLoginClaimsAndLocks: the first authenticated session takes
// ownership, and a second session cannot take it over or consent on it.
func TestInteractionLoginClaimsAndLocks(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	interactionID := f.startInteraction(t, ctx)

	const victimSession = "sess-victim"

	if err := f.grants.CompleteLogin(ctx, interactionID, f.userID, victimSession); err != nil {
		t.Fatalf("CompleteLogin: %v", err)
	}

	t.Run("another_session_cannot_take_over", func(t *testing.T) {
		err := f.grants.CompleteLogin(ctx, interactionID, f.userID, "sess-attacker")
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("second login = %v, want forbidden", err)
		}
	})

	t.Run("another_session_cannot_consent", func(t *testing.T) {
		_, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
			InteractionID: interactionID,
			AccountID:     f.userID,
			SessionID:     "sess-attacker",
			GrantedScopes: []string{"openid"},
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("consent from another session = %v, want forbidden", err)
		}
	})

	// Same session, different account: the code must be bound to the person who
	// actually authenticated, not to whoever the caller names.
	t.Run("another_account_on_the_same_session_cannot_consent", func(t *testing.T) {
		_, other := rolesTestUser(t, ctx)

		_, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
			InteractionID: interactionID,
			AccountID:     other,
			SessionID:     victimSession,
			GrantedScopes: []string{"openid"},
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("consent for a different account = %v, want forbidden", err)
		}
	})

	t.Run("owning_session_consents", func(t *testing.T) {
		redirect, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
			InteractionID: interactionID,
			AccountID:     f.userID,
			SessionID:     victimSession,
			GrantedScopes: []string{"openid"},
		})
		if err != nil {
			t.Fatalf("consent from the owning session: %v", err)
		}
		if redirect == "" {
			t.Fatal("consent returned no redirect")
		}
	})
}

// TestInteractionLoginRejectsSessionlessPrincipal: a credential with no browser
// session cannot claim an interactive flow at all.
func TestInteractionLoginRejectsSessionlessPrincipal(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	interactionID := f.startInteraction(t, ctx)

	if err := f.grants.CompleteLogin(ctx, interactionID, f.userID, ""); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("login without a session = %v, want forbidden", err)
	}

	if err := f.grants.CompleteLogin(ctx, interactionID, "", "sess-1"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("login without an account = %v, want forbidden", err)
	}
}

// TestInteractionExpires: a stale interaction is gone rather than resumable.
func TestInteractionExpires(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	interactionID := f.startInteraction(t, ctx)

	if _, err := testDB.Pool.Exec(ctx,
		`UPDATE iam_interactions SET expires_at = now() - interval '1 minute' WHERE id = $1`,
		interactionID); err != nil {
		t.Fatalf("age the interaction: %v", err)
	}

	if _, err := f.grants.ResolveInteraction(ctx, interactionID); !errors.Is(err, domain.ErrFlowExpired) {
		t.Fatalf("resolve expired = %v, want flow_expired", err)
	}

	if err := f.grants.CompleteLogin(ctx, interactionID, f.userID, "sess-1"); !errors.Is(err, domain.ErrFlowExpired) {
		t.Fatalf("login on expired = %v, want flow_expired", err)
	}

	if _, err := f.grants.Consent(ctx, domain.OIDCConsentCmd{
		InteractionID: interactionID, AccountID: f.userID, SessionID: "sess-1",
		GrantedScopes: []string{"openid"},
	}); !errors.Is(err, domain.ErrFlowExpired) {
		t.Fatalf("consent on expired = %v, want flow_expired", err)
	}
}

// TestInteractionContextRendersTheScreen: the hosted login/consent page arrives
// holding only the interaction id, and everything it must display has to come
// back from this one call — which application is asking, for what, in whose
// project and in which language.
func TestInteractionContextRendersTheScreen(t *testing.T) {
	ctx := context.Background()
	f := newPKCEFixture(t, ctx, "spa")
	interactionID := f.startInteraction(t, ctx)

	got, err := f.grants.ResolveInteraction(ctx, interactionID)
	if err != nil {
		t.Fatalf("ResolveInteraction: %v", err)
	}

	if got.ClientName != "pkce client" {
		t.Errorf("client name = %q, want the registered name", got.ClientName)
	}
	if got.ClientType != "spa" {
		t.Errorf("client type = %q, want spa", got.ClientType)
	}
	if got.ProjectID != f.projectID {
		t.Errorf("project_id = %q, want %q", got.ProjectID, f.projectID)
	}
	if got.Environment != "live" {
		t.Errorf("environment = %q, want live", got.Environment)
	}
	if got.ExpiresAt.IsZero() {
		t.Error("expires_at is zero; the page cannot tell the user their window")
	}
	if len(got.Interaction.Scopes) == 0 {
		t.Error("no requested scopes; nothing to consent to")
	}

	// Nobody has signed in yet.
	if got.Stage != domain.OIDCStageLogin {
		t.Fatalf("stage = %q, want login", got.Stage)
	}

	// Once a session claims it, only the decision is missing.
	if err := f.grants.CompleteLogin(ctx, interactionID, f.userID, "sess-"+newUUID()); err != nil {
		t.Fatalf("CompleteLogin: %v", err)
	}

	got, err = f.grants.ResolveInteraction(ctx, interactionID)
	if err != nil {
		t.Fatalf("ResolveInteraction after login: %v", err)
	}
	if got.Stage != domain.OIDCStageConsent {
		t.Fatalf("stage = %q, want consent", got.Stage)
	}
}
