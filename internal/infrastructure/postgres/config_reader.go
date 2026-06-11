package postgres

// config_reader.go is the runtime reader for the project-config documents stored
// in iam_config (auth, password_policy, session_policy, mfa_policy, ...). It is
// the single component every enforcement stage consults at request time.
//
// Design contract (read before editing):
//   - Fail-OPEN-to-defaults. The admin write path (admin_pg.go + domain
//     configspec.Validate) is the fail-CLOSED gate: a doc only lands in storage
//     after strict validation. At runtime we must never brick auth on a config
//     read/parse error, so this reader decodes TOLERANTLY (plain json.Unmarshal,
//     NOT the strict ParseX), applies defensive clamps, and returns
//     fully-defaulted Effective* value types — enforcement code never sees a nil
//     or an absent doc.
//   - When a doc is ABSENT (or empty), the returned Effective* carries the exact
//     hardcoded defaults the codebase used before this reader existed, so
//     behaviour is byte-identical until an admin sets a doc.
//   - Reads are env-scoped through effectiveEnv (test/live carry distinct config)
//     and memoized with a short TTL cache, copying the mechanics of
//     pkg/api/cors.go originCache (RWMutex + exp + stale-on-error fail-safe),
//     keyed by (projectID, environment, key).

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// ---------------------------------------------------------------------------
// Defaults — the pre-reader hardcoded values. Keep in sync with the package
// constants they mirror so an absent doc behaves exactly as before.
// ---------------------------------------------------------------------------

const (
	// defaultPasswordMinLength mirrors the old coreAuthLoadPasswordPolicy floor.
	defaultPasswordMinLength = 8
	// configReaderDefaultTTL is the cache window when none is supplied.
	configReaderDefaultTTL = 30 * time.Second
)

// ---------------------------------------------------------------------------
// Effective* — runtime, fully-defaulted views. Value types (no pointers): the
// reader has already applied defaults, so enforcement code is defaults-free.
// ---------------------------------------------------------------------------

// EffectivePasswordPolicy is the runtime view of password_policy.
type EffectivePasswordPolicy struct {
	MinLength      int  // default 8
	ZxcvbnMinScore int  // default 0
	BreachedCheck  bool // default false (write path rejects true today)
	History        int  // default 0
}

// EffectiveSessionPolicy is the runtime view of session_policy. Defaults equal
// the legacy package constants so an absent doc yields identical token lifetimes.
type EffectiveSessionPolicy struct {
	AccessTTL       time.Duration // default coreAuthAccessTTL (30m)
	RefreshTTL      time.Duration // default coreAuthRefreshTTL (30d)
	IdleTimeout     time.Duration // default 0 = disabled
	AbsoluteTimeout time.Duration // default 0 = disabled
	ReuseDetection  bool          // default false
}

// EffectiveMFAPolicy is the runtime view of mfa_policy.
type EffectiveMFAPolicy struct {
	RequiredForAdmins bool
	AllowedFactors    []string
	RememberDevice    bool
}

// EffectiveAuthConfig is the runtime view of the auth doc. RegistrationMode==""
// preserves the flowAuthConfig contract: empty means open/default.
type EffectiveAuthConfig struct {
	Methods          []string
	RegistrationMode string
	PasswordStrategy string
	AppBaseURL       string
	DefaultLocale    string
	SupportedLocales []string
}

// ---------------------------------------------------------------------------
// configReader
// ---------------------------------------------------------------------------

// cacheKey identifies one cached config doc within a project+environment.
type cacheKey struct {
	projectID string
	env       string
	key       string
}

// cacheEntry holds the raw jsonb bytes (nil == doc absent) and its expiry. Raw
// bytes are cached and parsed per call: parsing is cheap and avoids a
// type-specific cache.
type cacheEntry struct {
	raw []byte
	exp time.Time
}

// configReader reads typed, defaulted project-config views at runtime with a
// per-(project, env, key) TTL cache.
type configReader struct {
	db  *DB
	ttl time.Duration

	mu      sync.RWMutex
	entries map[cacheKey]cacheEntry
}

// NewConfigReader builds the shared runtime config reader. ttl<=0 defaults to
// 30s.
func NewConfigReader(db *DB, ttl time.Duration) *configReader {
	if ttl <= 0 {
		ttl = configReaderDefaultTTL
	}
	return &configReader{
		db:      db,
		ttl:     ttl,
		entries: make(map[cacheKey]cacheEntry),
	}
}

// invalidate drops the cached entry for (projectID, env, key) so the next read
// re-fetches. Unused by v1 consumers; provided for a future "config.updated"
// subscriber that wants sub-TTL propagation.
func (r *configReader) invalidate(projectID, env, key string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	delete(r.entries, cacheKey{projectID, env, key})
	r.mu.Unlock()
}

// rawDoc resolves the request environment and returns the (cached) raw jsonb for
// (project, env, key). A returned nil slice with a nil error means the doc is
// absent — callers apply defaults. An environment-resolution error (unknown
// X-Environment) is a CLIENT error and is propagated; a config read error is
// swallowed in favour of the last-good (or absent) cached value, copying the
// stale-on-error behaviour of pkg/api/cors.go.
func (r *configReader) rawDoc(ctx context.Context, projectID, key string) ([]byte, error) {
	// Env resolution is per-request and not cached: it is a cheap lookup and the
	// client error it surfaces must always propagate.
	env, err := effectiveEnv(ctx, r.db, projectID, runtimeDefaultEnv)
	if err != nil {
		return nil, err
	}
	return r.rawDocForEnv(ctx, projectID, env, key)
}

// rawDocForEnv is rawDoc with the environment supplied explicitly, for callers
// that must read config in the environment of a token/session rather than the
// request (e.g. refresh, where the policy must follow the token's environment so
// a client cannot select a weaker environment's policy via X-Environment).
func (r *configReader) rawDocForEnv(ctx context.Context, projectID, env, key string) ([]byte, error) {
	return r.loadDoc(ctx, projectID, env, key, false)
}

// rawDocStrict resolves the request environment and reads the doc fail-CLOSED: a
// genuine DB read error is PROPAGATED rather than swallowed to a stale/absent
// value. Used by security gates (e.g. consent.required) where a transient read
// failure must never silently skip the control.
func (r *configReader) rawDocStrict(ctx context.Context, projectID, key string) ([]byte, error) {
	env, err := effectiveEnv(ctx, r.db, projectID, runtimeDefaultEnv)
	if err != nil {
		return nil, err
	}
	return r.loadDoc(ctx, projectID, env, key, true)
}

// loadDoc is the cached read core. failClosed selects the read-error policy:
// false (default) keeps the last-good/absent value for a short window so
// non-security enforcement uses defaults rather than failing; true propagates
// the error so the caller can deny.
func (r *configReader) loadDoc(ctx context.Context, projectID, env, key string, failClosed bool) ([]byte, error) {
	if env == "" {
		env = runtimeDefaultEnv
	}
	ck := cacheKey{projectID: projectID, env: env, key: key}

	r.mu.RLock()
	ent, ok := r.entries[ck]
	if ok && time.Now().Before(ent.exp) {
		raw := ent.raw
		r.mu.RUnlock()
		return raw, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if ent, ok := r.entries[ck]; ok && time.Now().Before(ent.exp) { // another goroutine refreshed
		return ent.raw, nil
	}

	raw, err := r.fetch(ctx, projectID, env, key)
	if err != nil {
		if failClosed {
			return nil, err
		}
		// Keep the stale entry (if any); extend its exp briefly to avoid hammering
		// the DB. With no prior entry, fall back to "absent" (nil) for the same
		// short window so enforcement uses defaults rather than failing.
		prev := r.entries[ck]
		prev.exp = time.Now().Add(5 * time.Second)
		r.entries[ck] = prev
		return prev.raw, nil
	}
	r.entries[ck] = cacheEntry{raw: raw, exp: time.Now().Add(r.ttl)}
	return raw, nil
}

// fetch reads the iam_config row for (project, env, key), returning a nil slice
// when the row is absent or empty. Only a genuine read error propagates.
func (r *configReader) fetch(ctx context.Context, projectID, env, key string) ([]byte, error) {
	row, err := models.FindIamConfig(ctx, r.db.Bobx(), projectID, env, key)
	if err != nil {
		if errors.Is(translatePgErr("config", err), ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if len(row.Data) == 0 {
		return nil, nil
	}
	// Copy out of the row so the cached slice is not aliased to bob internals.
	out := make([]byte, len(row.Data))
	copy(out, row.Data)
	return out, nil
}

// ---------------------------------------------------------------------------
// Typed accessors
// ---------------------------------------------------------------------------

// PasswordPolicy returns the effective password policy for the project under the
// request environment. An absent doc yields MinLength=8 (the legacy default).
func (r *configReader) PasswordPolicy(ctx context.Context, projectID string) (EffectivePasswordPolicy, error) {
	eff := EffectivePasswordPolicy{MinLength: defaultPasswordMinLength}
	raw, err := r.rawDoc(ctx, projectID, "password_policy")
	if err != nil {
		return eff, err
	}
	if len(raw) == 0 {
		return eff, nil
	}
	var spec domain.PasswordPolicySpec
	if unmarshal(raw, &spec) != nil {
		return eff, nil // tolerant: fall back to defaults on a malformed doc
	}
	if spec.MinLength != nil && *spec.MinLength > 0 {
		eff.MinLength = *spec.MinLength
	}
	if spec.ZxcvbnMinScore != nil && *spec.ZxcvbnMinScore >= 0 {
		eff.ZxcvbnMinScore = *spec.ZxcvbnMinScore
	}
	if spec.BreachedCheck != nil {
		eff.BreachedCheck = *spec.BreachedCheck
	}
	if spec.History != nil && *spec.History > 0 {
		eff.History = *spec.History
	}
	return eff, nil
}

// SessionPolicy returns the effective session policy. Defaults equal the legacy
// constants (access 30m, refresh 30d, idle/absolute disabled). Values are
// clamped defensively: non-positive TTLs fall back to the default, and a refresh
// TTL that does not exceed the access TTL is ignored (write-time validation
// already enforces the ordering, but a legacy/hand-edited doc must not invert).
func (r *configReader) SessionPolicy(ctx context.Context, projectID string) (EffectiveSessionPolicy, error) {
	raw, err := r.rawDoc(ctx, projectID, "session_policy")
	if err != nil {
		return defaultSessionPolicy(), err
	}
	return parseSessionPolicy(raw), nil
}

// SessionPolicyForEnv is SessionPolicy bound to an explicit environment (the
// token's/session's own env), so refresh-time policy can't be weakened by the
// request's X-Environment header.
func (r *configReader) SessionPolicyForEnv(ctx context.Context, projectID, env string) (EffectiveSessionPolicy, error) {
	raw, err := r.rawDocForEnv(ctx, projectID, env, "session_policy")
	if err != nil {
		return defaultSessionPolicy(), err
	}
	return parseSessionPolicy(raw), nil
}

func defaultSessionPolicy() EffectiveSessionPolicy {
	return EffectiveSessionPolicy{AccessTTL: coreAuthAccessTTL, RefreshTTL: coreAuthRefreshTTL}
}

func parseSessionPolicy(raw []byte) EffectiveSessionPolicy {
	eff := defaultSessionPolicy()
	if len(raw) == 0 {
		return eff
	}
	var spec domain.SessionPolicySpec
	if unmarshal(raw, &spec) != nil {
		return eff
	}
	if spec.AccessTTL != nil && *spec.AccessTTL > 0 {
		eff.AccessTTL = time.Duration(*spec.AccessTTL) * time.Second
	}
	if spec.RefreshTTL != nil && *spec.RefreshTTL > 0 {
		eff.RefreshTTL = time.Duration(*spec.RefreshTTL) * time.Second
	}
	if spec.IdleTimeout != nil && *spec.IdleTimeout > 0 {
		eff.IdleTimeout = time.Duration(*spec.IdleTimeout) * time.Second
	}
	if spec.AbsoluteTimeout != nil && *spec.AbsoluteTimeout > 0 {
		eff.AbsoluteTimeout = time.Duration(*spec.AbsoluteTimeout) * time.Second
	}
	if spec.ReuseDetection != nil {
		eff.ReuseDetection = *spec.ReuseDetection
	}
	// Defensive ordering: a refresh TTL must outlast the access TTL.
	if eff.RefreshTTL <= eff.AccessTTL {
		eff.RefreshTTL = coreAuthRefreshTTL
		if eff.RefreshTTL <= eff.AccessTTL {
			eff.AccessTTL = coreAuthAccessTTL
		}
	}
	return eff
}

// MFAPolicy returns the effective MFA policy. An absent doc yields the zero
// policy (no factors required/offered).
func (r *configReader) MFAPolicy(ctx context.Context, projectID string) (EffectiveMFAPolicy, error) {
	var eff EffectiveMFAPolicy
	raw, err := r.rawDoc(ctx, projectID, "mfa_policy")
	if err != nil {
		return eff, err
	}
	if len(raw) == 0 {
		return eff, nil
	}
	var spec domain.MFAPolicySpec
	if unmarshal(raw, &spec) != nil {
		return eff, nil
	}
	if spec.RequiredForAdmins != nil {
		eff.RequiredForAdmins = *spec.RequiredForAdmins
	}
	if spec.RememberDevice != nil {
		eff.RememberDevice = *spec.RememberDevice
	}
	eff.AllowedFactors = append([]string(nil), spec.AllowedFactors...)
	return eff, nil
}

// ConsentConfig returns the project's consent documents for the request
// environment. An absent/empty/malformed doc yields a nil slice — i.e. "no
// consent configured", which preserves pre-gate behaviour (no required-consent
// step). Reads are env-scoped + cached like every other accessor. A genuine DB
// read error is swallowed to defaults (nil) here, matching the other typed
// accessors; the gate callers decide their own fail-open/closed policy on the
// returned (slice, error) pair.
func (r *configReader) ConsentConfig(ctx context.Context, projectID string) ([]domain.ConsentDocumentSpec, error) {
	// Fail-CLOSED: consent.required is a legal gate, so a transient read error
	// must propagate (the gate denies) rather than silently skip the step.
	raw, err := r.rawDocStrict(ctx, projectID, "consent")
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var spec domain.ConsentConfigSpec
	if unmarshal(raw, &spec) != nil {
		return nil, nil // tolerant: a malformed doc gates nothing
	}
	return spec.Documents, nil
}

// AuthConfig returns the effective auth doc. Empty RegistrationMode/
// PasswordStrategy mean "unset" (open/default), preserving the flowAuthConfig
// contract. supported_locales and the legacy locales key are reconciled.
func (r *configReader) AuthConfig(ctx context.Context, projectID string) (EffectiveAuthConfig, error) {
	var eff EffectiveAuthConfig
	raw, err := r.rawDoc(ctx, projectID, "auth")
	if err != nil {
		return eff, err
	}
	if len(raw) == 0 {
		return eff, nil
	}
	var spec domain.AuthConfigSpec
	if unmarshal(raw, &spec) != nil {
		return eff, nil
	}
	eff.Methods = append([]string(nil), spec.Methods...)
	if spec.Registration != nil {
		if spec.Registration.Mode != nil {
			eff.RegistrationMode = *spec.Registration.Mode
		}
		if spec.Registration.PasswordStrategy != nil {
			eff.PasswordStrategy = *spec.Registration.PasswordStrategy
		}
	}
	if spec.AppBaseURL != nil {
		eff.AppBaseURL = *spec.AppBaseURL
	}
	if spec.DefaultLocale != nil {
		eff.DefaultLocale = *spec.DefaultLocale
	}
	if len(spec.SupportedLocales) > 0 {
		eff.SupportedLocales = append([]string(nil), spec.SupportedLocales...)
	} else if len(spec.Locales) > 0 {
		eff.SupportedLocales = append([]string(nil), spec.Locales...)
	}
	return eff, nil
}
