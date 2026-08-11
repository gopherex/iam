//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gopherex/xlog"
)

// TestGCSweepDeletesExpired verifies the garbage collector deletes past-expiry
// runtime rows while leaving still-valid ones untouched.
func TestGCSweepDeletesExpired(t *testing.T) {
	ctx := context.Background()
	proj := "gc-" + newUUID()[:8]

	expired := newUUID()
	valid := newUUID()

	insert := `INSERT INTO iam_challenges (id, project_id, type, expires_at, data) VALUES ($1, $2, 'email', $3, '{}'::jsonb)`
	if _, err := testDB.Pool.Exec(ctx, insert, expired, proj, nowUTC().Add(-time.Hour)); err != nil {
		t.Fatalf("insert expired: %v", err)
	}

	if _, err := testDB.Pool.Exec(ctx, insert, valid, proj, nowUTC().Add(time.Hour)); err != nil {
		t.Fatalf("insert valid: %v", err)
	}

	testDB.gcSweepAll(ctx, xlog.NewJSON())

	if n := countChallenge(t, ctx, expired); n != 0 {
		t.Errorf("expired challenge not swept: count = %d", n)
	}

	if n := countChallenge(t, ctx, valid); n != 1 {
		t.Errorf("valid challenge wrongly swept: count = %d", n)
	}
}

func countChallenge(t *testing.T, ctx context.Context, id string) int {
	t.Helper()

	var n int
	if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_challenges WHERE id = $1`, id).Scan(&n); err != nil {
		t.Fatalf("count challenge: %v", err)
	}

	return n
}
