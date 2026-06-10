//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// TestActivityTrackingBumpsLastActive verifies the authenticator bumps a
// session's last_active_at on an authenticated request when it is stale beyond
// the throttle, and leaves it alone within the throttle window.
func TestActivityTrackingBumpsLastActive(t *testing.T) {
	ctx := context.Background()
	ca := NewPgCoreAuth(testDB, nopEmitter{})
	auth := NewAuthenticator(testDB, "")
	projectID := newUUID()
	_, sess, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     "act-" + newUUID()[:8] + "@example.com",
		Password:  "Sup3rStr0ng!Pass",
		Name:      "Act",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	backdate := func(d time.Duration) {
		t.Helper()
		row, err := models.FindIamSession(ctx, testDB.Bobx(), sess.ID)
		if err != nil {
			t.Fatalf("find session: %v", err)
		}
		old := nowUTC().Add(-d)
		if err := row.Update(ctx, testDB.Bobx(), &models.IamSessionSetter{LastActiveAt: &old}); err != nil {
			t.Fatalf("backdate: %v", err)
		}
	}
	lastActive := func() time.Time {
		t.Helper()
		row, err := models.FindIamSession(ctx, testDB.Bobx(), sess.ID)
		if err != nil {
			t.Fatalf("find session: %v", err)
		}
		return row.LastActiveAt
	}

	// Stale (2h) → authenticating bumps last_active to ~now.
	backdate(2 * time.Hour)
	if _, err := auth.User(ctx, sess.AccessToken); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if d := nowUTC().Sub(lastActive()); d > time.Minute {
		t.Errorf("last_active not bumped on stale session: %s ago", d)
	}

	// Within the throttle (30s) → authenticating does NOT rewrite it.
	backdate(30 * time.Second)
	if _, err := auth.User(ctx, sess.AccessToken); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if d := nowUTC().Sub(lastActive()); d < 20*time.Second {
		t.Errorf("last_active bumped within throttle window (should be ~30s ago, got %s)", d)
	}
}
