//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

const lockoutTestPassword = "Sup3rStr0ng!Pass"

// TestAccountLockoutAfterFailures verifies that consecutive wrong-password
// attempts lock the account, after which even the correct password is refused.
func TestAccountLockoutAfterFailures(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	ca := NewPgCoreAuth(testDB, &recordingEmitter{}, NewConfigReader(testDB, time.Minute))
	acct := registerForPolicy(t, ctx, ca, projectID)

	for i := range coreAuthMaxLoginFailures {
		_, err := ca.AuthenticatePassword(ctx, projectID, acct.PrimaryEmail, "wrong-password")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: got %v, want invalid_credentials", i+1, err)
		}
	}

	// Threshold reached: the account is locked and the correct password is
	// refused with account_locked (not tested).
	if _, err := ca.AuthenticatePassword(ctx, projectID, acct.PrimaryEmail, lockoutTestPassword); !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("after %d failures: got %v, want account_locked", coreAuthMaxLoginFailures, err)
	}
}

// TestAccountLockoutResetsOnSuccess verifies that a successful login before the
// threshold clears the accumulated failure counter.
func TestAccountLockoutResetsOnSuccess(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()
	ca := NewPgCoreAuth(testDB, &recordingEmitter{}, NewConfigReader(testDB, time.Minute))
	acct := registerForPolicy(t, ctx, ca, projectID)

	// One short of the threshold, then a success resets the counter.
	for range coreAuthMaxLoginFailures - 1 {
		if _, err := ca.AuthenticatePassword(ctx, projectID, acct.PrimaryEmail, "wrong-password"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("pre-reset wrong attempt: unexpected %v", err)
		}
	}

	if _, err := ca.AuthenticatePassword(ctx, projectID, acct.PrimaryEmail, lockoutTestPassword); err != nil {
		t.Fatalf("correct password should succeed and reset: %v", err)
	}

	// After the reset another threshold-1 failures must still NOT lock.
	for range coreAuthMaxLoginFailures - 1 {
		if _, err := ca.AuthenticatePassword(ctx, projectID, acct.PrimaryEmail, "wrong-password"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("post-reset wrong attempt: unexpected %v", err)
		}
	}

	if _, err := ca.AuthenticatePassword(ctx, projectID, acct.PrimaryEmail, lockoutTestPassword); err != nil {
		t.Fatalf("counter was not reset — locked too early: %v", err)
	}
}
