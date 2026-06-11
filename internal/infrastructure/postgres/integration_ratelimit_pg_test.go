//go:build integration

package postgres

// Integration tests for the Postgres-backed RateLimitConfigReader adapter
// (ratelimit_pg.go). They run against the shared testcontainers Postgres, seed
// the iam_config key="rate_limits" doc directly (reusing seedConfig), and assert
// the adapter's read/parse/filter and env-scoping behaviour. The middleware that
// consumes these rules is covered by pkg/api/ratelimit_test.go; this pins the DB
// read half end-to-end.

import (
	"context"
	"testing"
	"time"
)

// TestPgRateLimitsNoContext: an empty client id (no project context) yields no
// rules so the middleware falls back to the hardcoded defaults.
func TestPgRateLimitsNoContext(t *testing.T) {
	ctx := context.Background()
	a := NewPgRateLimits(testDB)

	rules, err := a.RateLimitRules(ctx, "", "live")
	if err != nil {
		t.Fatalf("RateLimitRules: %v", err)
	}
	if rules != nil {
		t.Fatalf("no-context rules = %v, want nil", rules)
	}
}

// TestPgRateLimitsNoDoc: a project with no rate_limits doc yields no rules.
func TestPgRateLimitsNoDoc(t *testing.T) {
	ctx := context.Background()
	a := NewPgRateLimits(testDB)

	rules, err := a.RateLimitRules(ctx, newUUID(), "live")
	if err != nil {
		t.Fatalf("RateLimitRules: %v", err)
	}
	if rules != nil {
		t.Fatalf("no-doc rules = %v, want nil", rules)
	}
}

// TestPgRateLimitsParsesStoredRule: a stored, valid ip rule on a realized
// endpoint is returned with the window mapped seconds->Duration.
func TestPgRateLimitsParsesStoredRule(t *testing.T) {
	ctx := context.Background()
	a := NewPgRateLimits(testDB)
	projectID := newUUID()

	seedConfig(t, ctx, projectID, "live", "rate_limits", map[string]any{
		"rules": []map[string]any{
			{"endpoint": "/v1/auth/sign-in/password", "limit": 5, "window_seconds": 60, "by": "ip"},
		},
	})

	rules, err := a.RateLimitRules(ctx, projectID, "live")
	if err != nil {
		t.Fatalf("RateLimitRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules = %v, want exactly 1", rules)
	}
	got := rules[0]
	if got.Endpoint != "/v1/auth/sign-in/password" || got.Limit != 5 || got.By != "ip" {
		t.Fatalf("rule = %+v, want sign-in/password limit=5 by=ip", got)
	}
	if got.Window != 60*time.Second {
		t.Fatalf("rule window = %v, want 60s", got.Window)
	}
}

// TestPgRateLimitsDropsNonEnforceable: rules that are not by=ip or whose endpoint
// is not a realized/classified path are dropped defensively, even if they slip
// past the write-path validation.
func TestPgRateLimitsDropsNonEnforceable(t *testing.T) {
	ctx := context.Background()
	a := NewPgRateLimits(testDB)
	projectID := newUUID()

	seedConfig(t, ctx, projectID, "live", "rate_limits", map[string]any{
		"rules": []map[string]any{
			{"endpoint": "/v1/auth/otp/start", "limit": 3, "window_seconds": 30, "by": "ip"},
			// by != ip -> dropped
			{"endpoint": "/v1/auth/otp/verify", "limit": 3, "window_seconds": 30, "by": "user"},
			// unknown endpoint -> dropped
			{"endpoint": "/v1/auth/not-a-real-endpoint", "limit": 9, "window_seconds": 30, "by": "ip"},
		},
	})

	rules, err := a.RateLimitRules(ctx, projectID, "live")
	if err != nil {
		t.Fatalf("RateLimitRules: %v", err)
	}
	if len(rules) != 1 || rules[0].Endpoint != "/v1/auth/otp/start" {
		t.Fatalf("rules = %+v, want only the by=ip realized endpoint", rules)
	}
}

// TestPgRateLimitsEnvScoped: a doc stored under "live" is invisible to a "preview"
// lookup (and vice-versa); each environment reads its own override.
func TestPgRateLimitsEnvScoped(t *testing.T) {
	ctx := context.Background()
	a := NewPgRateLimits(testDB)
	projectID := newUUID()

	seedConfig(t, ctx, projectID, "live", "rate_limits", map[string]any{
		"rules": []map[string]any{
			{"endpoint": "/v1/auth/sign-in/password", "limit": 5, "window_seconds": 60, "by": "ip"},
		},
	})

	// preview has no doc -> nil (defaults).
	preview, err := a.RateLimitRules(ctx, projectID, "preview")
	if err != nil {
		t.Fatalf("RateLimitRules preview: %v", err)
	}
	if preview != nil {
		t.Fatalf("preview rules = %v, want nil (env-isolated)", preview)
	}

	// empty env defaults to the runtime default env ("live") and sees the doc.
	def, err := a.RateLimitRules(ctx, projectID, "")
	if err != nil {
		t.Fatalf("RateLimitRules default-env: %v", err)
	}
	if len(def) != 1 {
		t.Fatalf("default-env rules = %v, want the live doc's single rule", def)
	}
}
