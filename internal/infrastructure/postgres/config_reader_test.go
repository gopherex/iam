//go:build integration

package postgres

// Integration tests for the runtime config reader. They run against the shared
// testcontainers Postgres (see integration_test.go) and seed iam_config rows
// directly, then assert the reader's defaulting, tolerant parsing, env scoping,
// and TTL-cache behaviour.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
	"github.com/gopherex/iam/pkg/api"
)

// seedConfig upserts an iam_config doc for (projectID, env, key).
func seedConfig(t *testing.T, ctx context.Context, projectID, env, key string, doc any) {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal config doc: %v", err)
	}
	rm := json.RawMessage(raw)
	now := nowUTC()
	// Try update first; insert if no row exists yet.
	existing, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(env))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg(key))),
	).One(ctx, testDB.Bobx())
	if err == nil {
		if err := existing.Update(ctx, testDB.Bobx(), &models.IamConfigSetter{
			Data:      &rm,
			UpdatedAt: &now,
		}); err != nil {
			t.Fatalf("update config %q: %v", key, err)
		}
		return
	}
	if _, err := models.IamConfigs.Insert(&models.IamConfigSetter{
		ProjectID:   &projectID,
		Environment: ptr(env),
		Key:         ptr(key),
		Data:        &rm,
		UpdatedAt:   &now,
	}).One(ctx, testDB.Bobx()); err != nil {
		t.Fatalf("insert config %q: %v", key, err)
	}
}

// TestConfigReaderDefaults verifies that an absent doc yields the legacy
// hardcoded defaults for every accessor.
func TestConfigReaderDefaults(t *testing.T) {
	ctx := context.Background()
	r := NewConfigReader(testDB, time.Minute)
	projectID := newUUID()

	pp, err := r.PasswordPolicy(ctx, projectID)
	if err != nil {
		t.Fatalf("password policy: %v", err)
	}
	if pp.MinLength != defaultPasswordMinLength {
		t.Errorf("MinLength = %d, want %d", pp.MinLength, defaultPasswordMinLength)
	}

	sp, err := r.SessionPolicy(ctx, projectID)
	if err != nil {
		t.Fatalf("session policy: %v", err)
	}
	if sp.AccessTTL != coreAuthAccessTTL {
		t.Errorf("AccessTTL = %v, want %v", sp.AccessTTL, coreAuthAccessTTL)
	}
	if sp.RefreshTTL != coreAuthRefreshTTL {
		t.Errorf("RefreshTTL = %v, want %v", sp.RefreshTTL, coreAuthRefreshTTL)
	}
	if sp.IdleTimeout != 0 || sp.AbsoluteTimeout != 0 {
		t.Errorf("idle/absolute = %v/%v, want 0/0", sp.IdleTimeout, sp.AbsoluteTimeout)
	}

	ac, err := r.AuthConfig(ctx, projectID)
	if err != nil {
		t.Fatalf("auth config: %v", err)
	}
	if ac.RegistrationMode != "" || ac.PasswordStrategy != "" {
		t.Errorf("auth defaults non-empty: mode=%q strategy=%q", ac.RegistrationMode, ac.PasswordStrategy)
	}

	mp, err := r.MFAPolicy(ctx, projectID)
	if err != nil {
		t.Fatalf("mfa policy: %v", err)
	}
	if mp.RequiredForAdmins || len(mp.AllowedFactors) != 0 {
		t.Errorf("mfa defaults non-zero: %+v", mp)
	}
}

// TestConfigReaderPasswordPolicyParse covers parsing + the min_length clamp.
func TestConfigReaderPasswordPolicyParse(t *testing.T) {
	ctx := context.Background()
	r := NewConfigReader(testDB, time.Minute)

	projectID := newUUID()
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "password_policy", map[string]any{"min_length": 12})
	pp, err := r.PasswordPolicy(ctx, projectID)
	if err != nil {
		t.Fatalf("password policy: %v", err)
	}
	if pp.MinLength != 12 {
		t.Errorf("MinLength = %d, want 12", pp.MinLength)
	}

	// min_length 0 must clamp back to the default floor.
	zeroProject := newUUID()
	seedConfig(t, ctx, zeroProject, runtimeDefaultEnv, "password_policy", map[string]any{"min_length": 0})
	pp, err = r.PasswordPolicy(ctx, zeroProject)
	if err != nil {
		t.Fatalf("password policy (zero): %v", err)
	}
	if pp.MinLength != defaultPasswordMinLength {
		t.Errorf("clamped MinLength = %d, want %d", pp.MinLength, defaultPasswordMinLength)
	}
}

// TestConfigReaderSessionPolicyMapping verifies seconds->Duration and ordering.
func TestConfigReaderSessionPolicyMapping(t *testing.T) {
	ctx := context.Background()
	r := NewConfigReader(testDB, time.Minute)
	projectID := newUUID()

	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "session_policy", map[string]any{
		"access_ttl":       60,
		"refresh_ttl":      3600,
		"idle_timeout":     1800,
		"absolute_timeout": 7200,
		"reuse_detection":  true,
	})
	sp, err := r.SessionPolicy(ctx, projectID)
	if err != nil {
		t.Fatalf("session policy: %v", err)
	}
	if sp.AccessTTL != 60*time.Second {
		t.Errorf("AccessTTL = %v, want 60s", sp.AccessTTL)
	}
	if sp.RefreshTTL != 3600*time.Second {
		t.Errorf("RefreshTTL = %v, want 3600s", sp.RefreshTTL)
	}
	if sp.IdleTimeout != 1800*time.Second {
		t.Errorf("IdleTimeout = %v, want 1800s", sp.IdleTimeout)
	}
	if sp.AbsoluteTimeout != 7200*time.Second {
		t.Errorf("AbsoluteTimeout = %v, want 7200s", sp.AbsoluteTimeout)
	}
	if !sp.ReuseDetection {
		t.Error("ReuseDetection = false, want true")
	}
	if sp.AccessTTL >= sp.RefreshTTL {
		t.Errorf("access >= refresh: %v >= %v", sp.AccessTTL, sp.RefreshTTL)
	}
}

// TestConfigReaderTolerantDecode proves an unknown extra key does NOT brick the
// read: known fields still parse (runtime is fail-open-to-defaults).
func TestConfigReaderTolerantDecode(t *testing.T) {
	ctx := context.Background()
	r := NewConfigReader(testDB, time.Minute)
	projectID := newUUID()

	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "password_policy", map[string]any{
		"min_length":     16,
		"unknown_future": "ignored",
	})
	pp, err := r.PasswordPolicy(ctx, projectID)
	if err != nil {
		t.Fatalf("password policy: %v", err)
	}
	if pp.MinLength != 16 {
		t.Errorf("MinLength = %d, want 16 (unknown key must not break parsing)", pp.MinLength)
	}
}

// TestConfigReaderCache verifies a second read within the TTL does not re-hit the
// DB (a config change is invisible until the window elapses or invalidate runs).
func TestConfigReaderCache(t *testing.T) {
	ctx := context.Background()
	r := NewConfigReader(testDB, time.Hour) // long TTL so the change stays cached
	projectID := newUUID()

	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "password_policy", map[string]any{"min_length": 10})
	if pp, _ := r.PasswordPolicy(ctx, projectID); pp.MinLength != 10 {
		t.Fatalf("warm MinLength = %d, want 10", pp.MinLength)
	}

	// Change the underlying doc; the cached value must persist.
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "password_policy", map[string]any{"min_length": 99})
	if pp, _ := r.PasswordPolicy(ctx, projectID); pp.MinLength != 10 {
		t.Errorf("cached MinLength = %d, want 10 (still cached)", pp.MinLength)
	}

	// After invalidation the new value is read.
	r.invalidate(projectID, runtimeDefaultEnv, "password_policy")
	if pp, _ := r.PasswordPolicy(ctx, projectID); pp.MinLength != 99 {
		t.Errorf("post-invalidate MinLength = %d, want 99", pp.MinLength)
	}
}

// TestConfigReaderEnvIsolation proves test/live docs return distinct values.
func TestConfigReaderEnvIsolation(t *testing.T) {
	ctx := context.Background()
	r := NewConfigReader(testDB, time.Minute)

	op := NewPgOperator(testDB, nopEmitter{})
	project, err := op.CreateProject(ctx, domain.ProjectCmd{Name: "config-reader-env"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := op.CreateEnvironment(ctx, domain.EnvironmentCmd{ProjectID: project.ID, Name: "test"}); err != nil {
		t.Fatalf("create env: %v", err)
	}

	seedConfig(t, ctx, project.ID, "live", "password_policy", map[string]any{"min_length": 8})
	seedConfig(t, ctx, project.ID, "test", "password_policy", map[string]any{"min_length": 20})

	liveCtx := api.WithEnvironment(ctx, "live")
	testCtx := api.WithEnvironment(ctx, "test")

	if pp, _ := r.PasswordPolicy(liveCtx, project.ID); pp.MinLength != 8 {
		t.Errorf("live MinLength = %d, want 8", pp.MinLength)
	}
	if pp, _ := r.PasswordPolicy(testCtx, project.ID); pp.MinLength != 20 {
		t.Errorf("test MinLength = %d, want 20", pp.MinLength)
	}
}

// TestConfigReaderUnknownEnvPropagates verifies that an unknown X-Environment is
// a client error and is NOT silently defaulted away.
func TestConfigReaderUnknownEnvPropagates(t *testing.T) {
	ctx := context.Background()
	r := NewConfigReader(testDB, time.Minute)

	op := NewPgOperator(testDB, nopEmitter{})
	project, err := op.CreateProject(ctx, domain.ProjectCmd{Name: "config-reader-badenv"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	badCtx := api.WithEnvironment(ctx, "does-not-exist")
	if _, err := r.PasswordPolicy(badCtx, project.ID); err == nil {
		t.Fatal("expected unknown-environment error to propagate")
	}
}

// TestSessionPolicyEnforcedOnMint is the regression guard for the NET-NEW
// behaviour: a session_policy doc must drive the minted session's access TTL
// (ExpiresIn) and refresh lifetime, instead of the legacy hardcoded constants.
func TestSessionPolicyEnforcedOnMint(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()

	// access_ttl 120s, refresh_ttl 600s (well below the 30m/30d defaults).
	seedConfig(t, ctx, projectID, runtimeDefaultEnv, "session_policy", map[string]any{
		"access_ttl":  120,
		"refresh_ttl": 600,
	})

	cfg := NewConfigReader(testDB, time.Minute)
	ca := NewPgCoreAuth(testDB, nopEmitter{}, cfg)
	_, sess, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     "session-policy@example.com",
		Password:  "Sup3rStr0ng!Pass",
		Name:      "SP",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if sess.ExpiresIn != 120 {
		t.Errorf("ExpiresIn = %d, want 120 (from session_policy.access_ttl)", sess.ExpiresIn)
	}
}

// TestSessionPolicyDefaultMintUnchanged asserts back-compat: with no
// session_policy doc, a minted session keeps the legacy access TTL (30m).
func TestSessionPolicyDefaultMintUnchanged(t *testing.T) {
	ctx := context.Background()
	projectID := newUUID()

	cfg := NewConfigReader(testDB, time.Minute)
	ca := NewPgCoreAuth(testDB, nopEmitter{}, cfg)
	_, sess, err := ca.Register(ctx, domain.RegisterCmd{
		ProjectID: projectID,
		Email:     "session-default@example.com",
		Password:  "Sup3rStr0ng!Pass",
		Name:      "SD",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if want := int(coreAuthAccessTTL / time.Second); sess.ExpiresIn != want {
		t.Errorf("ExpiresIn = %d, want %d (legacy default)", sess.ExpiresIn, want)
	}
}
