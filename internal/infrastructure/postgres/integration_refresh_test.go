//go:build integration

package postgres

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// TestRefreshDeviceMismatch verifies refresh denies + revokes when the presented
// device fingerprint differs from the one bound at sign-in (token-theft signal),
// and allows a matching or absent fingerprint.
func TestRefreshDeviceMismatch(t *testing.T) {
	ctx := context.Background()
	ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
	projectID := newUUID()
	acct, _, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     "dev-" + newUUID()[:8] + "@example.com",
		Password:  "Sup3rStr0ng!Pass",
		Name:      "Dev",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	ctxA := domain.WithRequestMeta(ctx, domain.RequestMeta{Fingerprint: "fp-A", IP: "1.1.1.1", UserAgent: "UA-A"})
	sess, err := ca.coreAuthMintSession(ctxA, acct, "client-1", []string{"pwd"}, 1)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	// Same fingerprint → refresh works.
	_, sess2, err := ca.Refresh(ctxA, sess.RefreshToken)
	if err != nil {
		t.Fatalf("refresh same device: %v", err)
	}

	// Different fingerprint → denied as device mismatch.
	ctxB := domain.WithRequestMeta(ctx, domain.RequestMeta{Fingerprint: "fp-B"})
	if _, _, err := ca.Refresh(ctxB, sess2.RefreshToken); !errors.Is(err, domain.ErrSessionDeviceMismatch) {
		t.Fatalf("expected device_mismatch, got %v", err)
	}

	// The session is revoked by the mismatch: even the matching device can't refresh now.
	if _, _, err := ca.Refresh(ctxA, sess2.RefreshToken); err == nil {
		t.Error("session should be revoked after a device mismatch")
	}
}

// TestRefreshNoFingerprintSkipsCheck: when the client sends no fingerprint the
// device check is skipped (best-effort binding), so refresh still works.
func TestRefreshNoFingerprintSkipsCheck(t *testing.T) {
	ctx := context.Background()
	ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
	projectID := newUUID()
	acct, _, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     "nofp-" + newUUID()[:8] + "@example.com",
		Password:  "Sup3rStr0ng!Pass",
		Name:      "NoFP",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	ctxA := domain.WithRequestMeta(ctx, domain.RequestMeta{Fingerprint: "fp-A"})
	sess, err := ca.coreAuthMintSession(ctxA, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	// No fingerprint on refresh → cannot enforce → allowed.
	if _, _, err := ca.Refresh(ctx, sess.RefreshToken); err != nil {
		t.Fatalf("refresh without fingerprint should be allowed: %v", err)
	}
}

// TestRefreshPreservesSessionContext is the regression for the refresh bug:
// rotating tokens must keep the SAME session (id, AAL/MFA elevation, AMR, client)
// rather than rebuilding a fresh AAL1 session on every refresh.
func TestRefreshPreservesSessionContext(t *testing.T) {
	ctx := context.Background()
	ca := NewPgCoreAuth(testDB, nopEmitter{}, nil)
	projectID := newUUID()
	acct, _, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     "refresh-" + newUUID()[:8] + "@example.com",
		Password:  "Sup3rStr0ng!Pass",
		Name:      "Refresh",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Mint an AAL2 (MFA-elevated) session bound to a specific client.
	sess, err := ca.coreAuthMintSession(ctx, acct, "client-1", []string{"pwd", "otp"}, 2)
	if err != nil {
		t.Fatalf("mint session: %v", err)
	}

	_, sess2, err := ca.Refresh(ctx, sess.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if sess2.ID != sess.ID {
		t.Errorf("session id changed on refresh: %q -> %q (should be stable)", sess.ID, sess2.ID)
	}
	if sess2.AAL != 2 {
		t.Errorf("AAL downgraded on refresh: got %d, want 2 (MFA elevation lost)", sess2.AAL)
	}
	if sess2.ClientID != "client-1" {
		t.Errorf("client lost on refresh: got %q, want client-1", sess2.ClientID)
	}
	if !slices.Contains(sess2.AMR, "otp") {
		t.Errorf("AMR lost on refresh: got %v, want it to contain otp", sess2.AMR)
	}
	if sess2.RefreshToken == "" || sess2.RefreshToken == sess.RefreshToken {
		t.Error("refresh token must be rotated to a new value")
	}
	if sess2.AccessToken == sess.AccessToken {
		t.Error("access token must be re-minted")
	}

	// Old refresh token is single-use: reuse must be rejected (rotation revoked it).
	if _, _, err := ca.Refresh(ctx, sess.RefreshToken); err == nil {
		t.Error("reusing the old refresh token must fail")
	}

	// The new token works and still preserves the context.
	_, sess3, err := ca.Refresh(ctx, sess2.RefreshToken)
	if err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if sess3.ID != sess.ID || sess3.AAL != 2 {
		t.Errorf("second refresh lost context: id=%q aal=%d", sess3.ID, sess3.AAL)
	}
}
