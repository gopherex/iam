//go:build integration

package postgres

// Integration tests for session_policy enforcement at refresh time: idle and
// absolute timeouts, the absolute cap on the refresh-token horizon, and the
// reuse_detection gate. They run against the shared testcontainers Postgres and
// drive the real Refresh/mint paths, backdating session columns to trip the
// time-based checks deterministically.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// recordingEmitter captures emitted events for assertions.
type recordingEmitter struct {
	mu     sync.Mutex
	events []domain.Event
}

func (e *recordingEmitter) Emit(_ context.Context, ev domain.Event) error {
	e.mu.Lock()
	e.events = append(e.events, ev)
	e.mu.Unlock()
	return nil
}

func (e *recordingEmitter) count(typ string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for _, ev := range e.events {
		if ev.Type == typ {
			n++
		}
	}
	return n
}

// backdateSession moves a session's created_at / last_active_at into the past so
// a time-based policy check trips deterministically.
func backdateSession(t *testing.T, ctx context.Context, sessionID string, created, lastActive time.Time) {
	t.Helper()
	row, err := models.FindIamSession(ctx, testDB.Bobx(), sessionID)
	if err != nil {
		t.Fatalf("find session: %v", err)
	}
	if err := row.Update(ctx, testDB.Bobx(), &models.IamSessionSetter{
		CreatedAt:    &created,
		LastActiveAt: &lastActive,
	}); err != nil {
		t.Fatalf("backdate session: %v", err)
	}
}

func registerForPolicy(t *testing.T, ctx context.Context, ca *pgCoreAuth, projectID string) *domain.Account {
	t.Helper()
	acct, _, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     "sp-" + newUUID()[:8] + "@example.com",
		Password:  "Sup3rStr0ng!Pass",
		Name:      "SP",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return acct
}

// TestSessionPolicyIdleTimeout: a session idle beyond idle_timeout is refused
// (ErrSessionExpired) and revoked, even though the refresh token itself is still
// within its refresh_ttl.
func TestSessionPolicyIdleTimeout(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "session_policy", map[string]any{
		"access_ttl":   60,
		"refresh_ttl":  99999,
		"idle_timeout": 1,
	})
	cfg := NewConfigReader(testDB, time.Minute)
	ca := NewPgCoreAuth(testDB, nopEmitter{}, cfg)
	acct := registerForPolicy(t, ctx, ca, projectID)

	sess, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	// Push last-active well past the 1s idle window.
	past := nowUTC().Add(-time.Hour)
	backdateSession(t, ctx, sess.ID, past, past)

	if _, _, err := ca.Refresh(ctx, sess.RefreshToken); !errors.Is(err, domain.ErrSessionExpired) {
		t.Fatalf("expected session_expired (idle), got %v", err)
	}
	// Session is revoked: a subsequent refresh on the same token also fails.
	if _, _, err := ca.Refresh(ctx, sess.RefreshToken); err == nil {
		t.Error("session should be revoked after idle timeout")
	}
}

// TestSessionPolicyAbsoluteTimeout: a session older than absolute_timeout is
// refused even if active recently (within idle and refresh windows).
func TestSessionPolicyAbsoluteTimeout(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "session_policy", map[string]any{
		"access_ttl":       60,
		"refresh_ttl":      99999,
		"idle_timeout":     99999,
		"absolute_timeout": 1,
	})
	cfg := NewConfigReader(testDB, time.Minute)
	ca := NewPgCoreAuth(testDB, nopEmitter{}, cfg)
	acct := registerForPolicy(t, ctx, ca, projectID)

	sess, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	// created_at far in the past, but last_active just now (still within idle).
	backdateSession(t, ctx, sess.ID, nowUTC().Add(-time.Hour), nowUTC())

	if _, _, err := ca.Refresh(ctx, sess.RefreshToken); !errors.Is(err, domain.ErrSessionExpired) {
		t.Fatalf("expected session_expired (absolute), got %v", err)
	}
}

// TestSessionPolicyAbsoluteCapsRefreshHorizon: with absolute_timeout shorter
// than refresh_ttl, a rotation near the absolute deadline issues a refresh token
// whose expiry is capped at the absolute deadline (created_at + absolute), not
// now + refresh_ttl.
func TestSessionPolicyAbsoluteCapsRefreshHorizon(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "session_policy", map[string]any{
		"access_ttl":       60,
		"refresh_ttl":      99999,
		"idle_timeout":     99999,
		"absolute_timeout": 600, // 10m absolute lifetime
	})
	cfg := NewConfigReader(testDB, time.Minute)
	ca := NewPgCoreAuth(testDB, nopEmitter{}, cfg)
	acct := registerForPolicy(t, ctx, ca, projectID)

	sess, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	// created_at 9 minutes ago: 1 minute of absolute lifetime remains.
	created := nowUTC().Add(-9 * time.Minute)
	backdateSession(t, ctx, sess.ID, created, nowUTC())

	_, sess2, err := ca.Refresh(ctx, sess.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	// The new refresh-token row must expire at created+600s, not now+99999s.
	rtRow, err := models.IamRefreshTokens.Query(
		sm.Where(models.IamRefreshTokens.Columns.Hash.EQ(psql.Arg(coreAuthSHA256(sess2.RefreshToken)))),
	).One(ctx, testDB.Bobx())
	if err != nil {
		t.Fatalf("find rotated refresh token: %v", err)
	}
	deadline := created.Add(600 * time.Second)
	if got, ok := rtRow.ExpiresAt.Get(); !ok || got.Sub(deadline).Abs() > 5*time.Second {
		t.Errorf("refresh expiry = %v, want ~%v (absolute cap)", got, deadline)
	}
}

// TestSessionPolicyReuseDetectionOn (default-on via doc): presenting a rotated
// (revoked) refresh token revokes ALL of the user's sessions and emits
// token.reuse_detected.
func TestSessionPolicyReuseDetectionOn(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "session_policy", map[string]any{
		"access_ttl":      60,
		"refresh_ttl":     99999,
		"reuse_detection": true,
	})
	cfg := NewConfigReader(testDB, time.Minute)
	rec := &recordingEmitter{}
	ca := NewPgCoreAuth(testDB, rec, cfg)
	acct := registerForPolicy(t, ctx, ca, projectID)

	// Two independent sessions for the same user.
	sA, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint A: %v", err)
	}
	sB, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint B: %v", err)
	}

	// Rotate A so its original token becomes revoked.
	if _, _, err := ca.Refresh(ctx, sA.RefreshToken); err != nil {
		t.Fatalf("rotate A: %v", err)
	}
	// Present the old (revoked) A token: reuse detected.
	if _, _, err := ca.Refresh(ctx, sA.RefreshToken); !errors.Is(err, domain.ErrTokenRevoked) {
		t.Fatalf("expected token_revoked, got %v", err)
	}
	if rec.count("token.reuse_detected") != 1 {
		t.Errorf("token.reuse_detected emitted %d times, want 1", rec.count("token.reuse_detected"))
	}
	// Cascade revoked ALL of the user's sessions, including the untouched B.
	if _, _, err := ca.Refresh(ctx, sB.RefreshToken); err == nil {
		t.Error("session B should be revoked by the reuse cascade")
	}
}

// TestSessionPolicyReuseDetectionOff: with reuse_detection=false a revoked token
// is still REJECTED (no replay), but OTHER sessions survive and no
// token.reuse_detected event fires.
func TestSessionPolicyReuseDetectionOff(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "session_policy", map[string]any{
		"access_ttl":      60,
		"refresh_ttl":     99999,
		"reuse_detection": false,
	})
	cfg := NewConfigReader(testDB, time.Minute)
	rec := &recordingEmitter{}
	ca := NewPgCoreAuth(testDB, rec, cfg)
	acct := registerForPolicy(t, ctx, ca, projectID)

	sA, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint A: %v", err)
	}
	sB, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint B: %v", err)
	}

	if _, _, err := ca.Refresh(ctx, sA.RefreshToken); err != nil {
		t.Fatalf("rotate A: %v", err)
	}
	// Revoked token still rejected (no replay downgrade).
	if _, _, err := ca.Refresh(ctx, sA.RefreshToken); !errors.Is(err, domain.ErrTokenRevoked) {
		t.Fatalf("expected token_revoked, got %v", err)
	}
	// No cascade: B still works.
	if _, _, err := ca.Refresh(ctx, sB.RefreshToken); err != nil {
		t.Errorf("session B should survive with reuse_detection off: %v", err)
	}
	if rec.count("token.reuse_detected") != 0 {
		t.Errorf("token.reuse_detected emitted %d times, want 0", rec.count("token.reuse_detected"))
	}
}

// TestSessionPolicyDefaultsRefreshUnchanged: with no doc, refresh works within
// the legacy window and idle/absolute enforcement is inert (back-compat).
func TestSessionPolicyDefaultsRefreshUnchanged(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	cfg := NewConfigReader(testDB, time.Minute)
	ca := NewPgCoreAuth(testDB, nopEmitter{}, cfg)
	acct := registerForPolicy(t, ctx, ca, projectID)

	sess, err := ca.coreAuthMintSession(ctx, acct, "", nil, 1)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	// Even a long-idle session refreshes fine: no idle/absolute timeout by default.
	past := nowUTC().Add(-365 * 24 * time.Hour)
	backdateSession(t, ctx, sess.ID, past, past)
	if _, _, err := ca.Refresh(ctx, sess.RefreshToken); err != nil {
		t.Fatalf("default refresh should succeed regardless of age: %v", err)
	}
}
