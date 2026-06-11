package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// configspec.go is the single source of truth for validating project-config
// documents (the iam_config jsonb envelopes: auth, password_policy,
// session_policy, mfa_policy, rate_limits, consent, features) plus notification
// provider docs.
//
// Design contract (read before editing):
//   - FAIL-CLOSED. Any unknown key or unsupported value is REJECTED with
//     ErrValidation (HTTP 422). We never silently drop or default away a value
//     the admin supplied — that would let the API advertise behaviour the
//     runtime cannot honour.
//   - Each typed struct mirrors the EXACT jsonb field names used in storage
//     (the *_pg.go / oas struct tags), so Validate() parses the same bytes the
//     adapter persists/reads. The canonical key is the lowercase snake_case
//     tag.
//   - Registries below enumerate ONLY values that are implemented end-to-end
//     RIGHT NOW. To add a new value you must (a) wire it end-to-end in the
//     runtime, then (b) add it to the registry here. The registry is the gate.
//   - No dependencies beyond the standard library and this package.

// ---------------------------------------------------------------------------
// Canonical registries (single source of truth)
// ---------------------------------------------------------------------------

// stringSet is an ordered, membership-checked registry. Insertion order is
// preserved for stable, human-readable "allowed" error details.
type stringSet struct {
	order []string
	set   map[string]struct{}
}

func newStringSet(values ...string) stringSet {
	s := stringSet{order: append([]string(nil), values...), set: make(map[string]struct{}, len(values))}
	for _, v := range values {
		s.set[v] = struct{}{}
	}
	return s
}

// Has reports whether v is a member.
func (s stringSet) Has(v string) bool { _, ok := s.set[v]; return ok }

// List returns the members in registration order (for error details / public-config filtering).
func (s stringSet) List() []string { return append([]string(nil), s.order...) }

// SupportedAuthMethods is the canonical set of values accepted in the auth doc
// `methods[]` array. Only methods implemented end-to-end are listed:
//   - email      — password + email OTP/verification (flow engine + direct endpoints)
//   - oauth      — social/OIDC via iam_providers rows (a methods entry is decorative)
//   - passkey    — WebAuthn ceremony endpoints + iam_webauthn_credentials
//   - magic_link — /v1/auth/magic-link/{start,verify}
//   - phone      — phone OTP login/signup over /v1/auth/otp/{start,verify}
//     (channel=sms|whatsapp; resolves/creates by primary_phone)
//
// `username` (as a first-class login method) is intentionally ABSENT: no
// route/handler/credential exists. To add a method: implement it end-to-end,
// then add it here.
var SupportedAuthMethods = newStringSet("email", "oauth", "passkey", "magic_link", "phone")

// SupportedMFAFactors is the canonical set of values accepted in the mfa_policy
// doc `allowed_factors[]`. All five are implemented end-to-end. Note two policy
// values differ from the internal/DB names:
//   - email_otp   maps to DB factor type "email"
//   - backup_codes maps to the iam_recovery_codes subsystem
//
// To add a factor: implement enroll+verify end-to-end, then add it here.
var SupportedMFAFactors = newStringSet("totp", "sms", "email_otp", "webauthn", "backup_codes")

// RegistrationModes is the canonical set for auth.registration.mode. All four
// are enforced in the signup flow engine. To add a mode: implement the flow
// branch, then add it here.
var RegistrationModes = newStringSet("open", "invite_only", "request_access", "closed")

// PasswordStrategies is the canonical set for auth.registration.password_strategy.
// Both are realized. To add a strategy: implement the flow branch, then add it here.
var PasswordStrategies = newStringSet("password_first", "after_verify")

// RateLimitActions is the canonical set for rate_limits rules `action`. The
// runtime limiter has NO concept of "action" today, so the set is EMPTY and any
// non-empty action is rejected. To support actions: implement action-aware
// limiting, then populate this set.
var RateLimitActions = newStringSet()

// RateLimitBy is the canonical set for rate_limits rules `by` (the counter
// subject). The runtime limiter keys exclusively by client IP, so "ip" is the
// only realized value. To add a dimension (user/session/...): implement keying
// in the limiter, then add it here.
var RateLimitBy = newStringSet("ip")

// RateLimitEndpoints is the canonical set of endpoints the hardcoded limiter
// router actually classifies. A rule targeting any other path could never be
// enforced, so it is rejected. This MUST stay in sync with the switch lists in
// pkg/api/ratelimit.go (guest + sensitive + auth buckets). To add an endpoint:
// add it to the router classification, then add it here.
var RateLimitEndpoints = newStringSet(
	// guest bucket
	"/v1/auth/guest",
	// sensitive bucket
	"/v1/auth/sign-in/password",
	"/v1/auth/password/forgot",
	"/v1/auth/password/reset",
	"/v1/auth/password/verify",
	"/v1/auth/email/verification/start",
	"/v1/auth/email/verification/verify",
	"/v1/auth/phone/verification/start",
	"/v1/auth/phone/verification/verify",
	"/v1/auth/otp/start",
	"/v1/auth/otp/verify",
	"/v1/auth/magic-link/start",
	"/v1/auth/magic-link/verify",
	"/v1/auth/mfa/challenge",
	"/v1/auth/mfa/verify",
	"/v1/auth/webauthn/login/options",
	"/v1/auth/webauthn/login/verify",
	"/v1/auth/webauthn/register/options",
	"/v1/auth/webauthn/register/verify",
	"/v1/challenges/captcha/verify",
	// auth bucket
	"/v1/auth/sign-up",
	"/v1/auth/token/refresh",
	"/v1/auth/token/exchange",
	"/v1/auth/oauth/exchange",
	"/v1/auth/access-requests",
)

// EmailProviderTypes is the canonical set for iam_providers (kind=email) `type`.
// Only "smtp" is realized: the runtime sender skips every other type silently.
// To add a sender (ses/sendgrid/...): implement the sender, then add it here.
var EmailProviderTypes = newStringSet("smtp")

// SMSProviderTypes is the canonical set for iam_providers (kind=sms) `type`.
// "generic" (HTTP webhook), "twilio", and "aws_sns" (AWS SNS / any SNS-compatible
// endpoint such as Yandex Cloud Notifications via a custom endpoint) are realized.
// The runtime sender skips every other type silently. To add a sender: implement
// the sender, then add it here.
var SMSProviderTypes = newStringSet("generic", "twilio", "aws_sns")

// FeatureKeys is the canonical namespace for the features doc (map[string]bool).
// Only keys backed by a working subsystem are listed. To add a feature: wire the
// toggle into the runtime, then add the key here.
var FeatureKeys = newStringSet(
	"password",
	"otp",
	"phone_login",
	"magic_link",
	"webauthn",
	"oauth",
	"mfa",
	"guest",
	"email_verification",
	"phone_verification",
	"email_change",
	"phone_change",
	"resumable_flows",
	"impersonation",
	"consent",
	"access_requests",
	"step_up",
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// strictUnmarshal decodes raw JSON into v rejecting unknown fields, so an
// unexpected key in the doc fails closed rather than being silently dropped.
func strictUnmarshal(raw []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

// validateAbsoluteHTTPURL ensures s is an absolute http(s) URL (used for
// app_base_url and consent document URLs).
func validateAbsoluteHTTPURL(field, s string) error {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || !u.IsAbs() || u.Host == "" {
		return ErrValidation.WithMessage(field + " must be an absolute http(s) URL")
	}
	if sc := strings.ToLower(u.Scheme); sc != "http" && sc != "https" {
		return ErrValidation.WithMessage(field + " must use http or https")
	}
	return nil
}

// ---------------------------------------------------------------------------
// auth doc — iam_config key="auth"
// ---------------------------------------------------------------------------

// AuthRegistrationConfig mirrors the nested `registration` object in the auth
// doc. Tags match coreauth_flows_pg.go / platform_pg.go / oas RegistrationConfig.
type AuthRegistrationConfig struct {
	Mode             *string `json:"mode,omitempty"`
	PasswordStrategy *string `json:"password_strategy,omitempty"`
}

// AuthConfigSpec is the typed mirror of the iam_config key="auth" document.
// Field tags are the exact stored jsonb keys. `locales` and `supported_locales`
// both appear in the wild (oas writes supported_locales, platform_pg reads
// locales); both are accepted and reconciled in Validate.
type AuthConfigSpec struct {
	Methods          []string                `json:"methods,omitempty"`
	Registration     *AuthRegistrationConfig `json:"registration,omitempty"`
	AppBaseURL       *string                 `json:"app_base_url,omitempty"`
	DefaultLocale    *string                 `json:"default_locale,omitempty"`
	SupportedLocales []string                `json:"supported_locales,omitempty"`
	Locales          []string                `json:"locales,omitempty"`
}

// ParseAuthConfig strictly decodes the auth doc, rejecting unknown top-level keys.
func ParseAuthConfig(raw []byte) (AuthConfigSpec, error) {
	var c AuthConfigSpec
	if err := strictUnmarshal(raw, &c); err != nil {
		return c, ErrValidation.WithMessage("invalid auth config: " + err.Error())
	}
	return c, nil
}

// Validate enforces the auth-doc rules fail-closed.
func (c AuthConfigSpec) Validate() error {
	seen := make(map[string]struct{}, len(c.Methods))
	for _, m := range c.Methods {
		if !SupportedAuthMethods.Has(m) {
			return ErrValidation.WithDetails(map[string]any{
				"field":   "methods",
				"value":   m,
				"allowed": SupportedAuthMethods.List(),
			}).WithMessage("unsupported auth method: " + m)
		}
		if _, dup := seen[m]; dup {
			return ErrValidation.WithMessage("duplicate auth method: " + m)
		}
		seen[m] = struct{}{}
	}

	if r := c.Registration; r != nil {
		if r.Mode != nil && *r.Mode != "" && !RegistrationModes.Has(*r.Mode) {
			return ErrValidation.WithDetails(map[string]any{
				"field":   "registration.mode",
				"value":   *r.Mode,
				"allowed": RegistrationModes.List(),
			}).WithMessage("unsupported registration mode: " + *r.Mode)
		}
		if r.PasswordStrategy != nil && *r.PasswordStrategy != "" && !PasswordStrategies.Has(*r.PasswordStrategy) {
			return ErrValidation.WithDetails(map[string]any{
				"field":   "registration.password_strategy",
				"value":   *r.PasswordStrategy,
				"allowed": PasswordStrategies.List(),
			}).WithMessage("unsupported password strategy: " + *r.PasswordStrategy)
		}
	}

	if c.AppBaseURL != nil && strings.TrimSpace(*c.AppBaseURL) != "" {
		if err := validateAbsoluteHTTPURL("app_base_url", *c.AppBaseURL); err != nil {
			return err
		}
	}

	// default_locale, if set, must belong to the supported-locale list when that
	// list is non-empty. Honour either jsonb key.
	locales := c.SupportedLocales
	if len(locales) == 0 {
		locales = c.Locales
	}
	if c.DefaultLocale != nil && strings.TrimSpace(*c.DefaultLocale) != "" && len(locales) > 0 {
		ok := false
		for _, l := range locales {
			if l == *c.DefaultLocale {
				ok = true
				break
			}
		}
		if !ok {
			return ErrValidation.WithDetails(map[string]any{
				"field":             "default_locale",
				"value":             *c.DefaultLocale,
				"supported_locales": locales,
			}).WithMessage("default_locale must be one of the supported locales")
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// password_policy doc — iam_config key="password_policy"
// ---------------------------------------------------------------------------

// PasswordPolicySpec mirrors the password_policy jsonb doc. Tags match
// coreauth_pg.go coreAuthPasswordPolicy and oas PasswordPolicy.
type PasswordPolicySpec struct {
	MinLength      *int  `json:"min_length,omitempty"`
	BreachedCheck  *bool `json:"breached_check,omitempty"`
	History        *int  `json:"history,omitempty"`
	ZxcvbnMinScore *int  `json:"zxcvbn_min_score,omitempty"`
}

// ParsePasswordPolicy strictly decodes the password_policy doc.
func ParsePasswordPolicy(raw []byte) (PasswordPolicySpec, error) {
	var c PasswordPolicySpec
	if err := strictUnmarshal(raw, &c); err != nil {
		return c, ErrValidation.WithMessage("invalid password_policy: " + err.Error())
	}
	return c, nil
}

// Validate enforces password-policy rules fail-closed. breached_check=true and
// history>0 are rejected because their engines are not implemented — we refuse
// to let admins enable a no-op security control.
func (c PasswordPolicySpec) Validate() error {
	if c.MinLength != nil {
		if *c.MinLength < 1 || *c.MinLength > 256 {
			return ErrValidation.WithMessage("min_length must be between 1 and 256")
		}
	}
	if c.ZxcvbnMinScore != nil {
		if *c.ZxcvbnMinScore < 0 || *c.ZxcvbnMinScore > 4 {
			return ErrValidation.WithMessage("zxcvbn_min_score must be between 0 and 4")
		}
	}
	if c.BreachedCheck != nil && *c.BreachedCheck {
		return ErrValidation.WithMessage("breached_check not supported")
	}
	if c.History != nil {
		if *c.History < 0 || *c.History > 50 {
			return ErrValidation.WithMessage("history must be between 0 and 50")
		}
		if *c.History > 0 {
			return ErrValidation.WithMessage("password history not supported")
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// session_policy doc — iam_config key="session_policy"
// ---------------------------------------------------------------------------

// SessionPolicySpec mirrors the session_policy jsonb doc. Tags match oas
// SessionPolicy. All TTLs are seconds.
type SessionPolicySpec struct {
	AccessTTL       *int  `json:"access_ttl,omitempty"`
	RefreshTTL      *int  `json:"refresh_ttl,omitempty"`
	IdleTimeout     *int  `json:"idle_timeout,omitempty"`
	AbsoluteTimeout *int  `json:"absolute_timeout,omitempty"`
	ReuseDetection  *bool `json:"reuse_detection,omitempty"`
}

// ParseSessionPolicy strictly decodes the session_policy doc.
func ParseSessionPolicy(raw []byte) (SessionPolicySpec, error) {
	var c SessionPolicySpec
	if err := strictUnmarshal(raw, &c); err != nil {
		return c, ErrValidation.WithMessage("invalid session_policy: " + err.Error())
	}
	return c, nil
}

// Validate enforces positivity, sane upper bounds, and cross-field ordering.
func (c SessionPolicySpec) Validate() error {
	const (
		maxAccess   = 86400    // 1 day
		maxRefresh  = 31536000 // 365 days
		maxAbsolute = 31536000 // 365 days
	)
	check := func(field string, v *int, max int) error {
		if v == nil {
			return nil
		}
		if *v <= 0 {
			return ErrValidation.WithMessage(field + " must be > 0")
		}
		if *v > max {
			return ErrValidation.WithDetails(map[string]any{"field": field, "value": *v, "max": max}).
				WithMessage(field + " exceeds maximum")
		}
		return nil
	}
	if err := check("access_ttl", c.AccessTTL, maxAccess); err != nil {
		return err
	}
	if err := check("refresh_ttl", c.RefreshTTL, maxRefresh); err != nil {
		return err
	}
	if err := check("idle_timeout", c.IdleTimeout, maxRefresh); err != nil {
		return err
	}
	if err := check("absolute_timeout", c.AbsoluteTimeout, maxAbsolute); err != nil {
		return err
	}

	if c.AccessTTL != nil && c.RefreshTTL != nil && *c.AccessTTL >= *c.RefreshTTL {
		return ErrValidation.WithDetails(map[string]any{"access_ttl": *c.AccessTTL, "refresh_ttl": *c.RefreshTTL}).
			WithMessage("access_ttl must be less than refresh_ttl")
	}
	if c.AccessTTL != nil && c.IdleTimeout != nil && *c.AccessTTL > *c.IdleTimeout {
		return ErrValidation.WithDetails(map[string]any{"access_ttl": *c.AccessTTL, "idle_timeout": *c.IdleTimeout}).
			WithMessage("access_ttl must not exceed idle_timeout")
	}
	if c.IdleTimeout != nil && c.AbsoluteTimeout != nil && *c.IdleTimeout > *c.AbsoluteTimeout {
		return ErrValidation.WithDetails(map[string]any{"idle_timeout": *c.IdleTimeout, "absolute_timeout": *c.AbsoluteTimeout}).
			WithMessage("idle_timeout must not exceed absolute_timeout")
	}
	if c.IdleTimeout != nil && c.RefreshTTL != nil && *c.IdleTimeout > *c.RefreshTTL {
		return ErrValidation.WithDetails(map[string]any{"idle_timeout": *c.IdleTimeout, "refresh_ttl": *c.RefreshTTL}).
			WithMessage("idle_timeout must not exceed refresh_ttl")
	}
	return nil
}

// ---------------------------------------------------------------------------
// mfa_policy doc — iam_config key="mfa_policy"
// ---------------------------------------------------------------------------

// MFAPolicySpec mirrors the mfa_policy jsonb doc. Tags match oas MfaPolicy.
type MFAPolicySpec struct {
	RequiredForAdmins *bool    `json:"required_for_admins,omitempty"`
	AllowedFactors    []string `json:"allowed_factors,omitempty"`
	RememberDevice    *bool    `json:"remember_device,omitempty"`
}

// ParseMFAPolicy strictly decodes the mfa_policy doc.
func ParseMFAPolicy(raw []byte) (MFAPolicySpec, error) {
	var c MFAPolicySpec
	if err := strictUnmarshal(raw, &c); err != nil {
		return c, ErrValidation.WithMessage("invalid mfa_policy: " + err.Error())
	}
	return c, nil
}

// Validate enforces the factor enum, rejects duplicates, and blocks the
// "MFA required but no factors offered" lockout configuration.
func (c MFAPolicySpec) Validate() error {
	seen := make(map[string]struct{}, len(c.AllowedFactors))
	for _, f := range c.AllowedFactors {
		if !SupportedMFAFactors.Has(f) {
			return ErrValidation.WithDetails(map[string]any{
				"field":   "allowed_factors",
				"value":   f,
				"allowed": SupportedMFAFactors.List(),
			}).WithMessage("allowed_factors: unknown factor " + f)
		}
		if _, dup := seen[f]; dup {
			return ErrValidation.WithMessage("allowed_factors: duplicate factor " + f)
		}
		seen[f] = struct{}{}
	}
	// AllowedFactors present-but-empty + required => lockout.
	if c.AllowedFactors != nil && len(c.AllowedFactors) == 0 &&
		c.RequiredForAdmins != nil && *c.RequiredForAdmins {
		return ErrValidation.WithMessage("allowed_factors cannot be empty when MFA is required")
	}
	return nil
}

// MFAPolicyFactorName maps an internal/DB factor type to the value used in the
// mfa_policy `allowed_factors[]` doc (see SupportedMFAFactors). Two names differ
// from the DB representation:
//   - DB "email"             -> policy "email_otp"
//   - the recovery-codes subsystem ("recovery"/"backup_codes") -> "backup_codes"
//
// All other factors (totp/sms/webauthn) map 1:1.
func MFAPolicyFactorName(dbFactorType string) string {
	switch dbFactorType {
	case "email":
		return "email_otp"
	case "recovery", "backup_codes":
		return "backup_codes"
	default:
		return dbFactorType
	}
}

// FactorAllowed reports whether the given internal/DB factor type may be
// enrolled under this policy. An unset (nil) or empty AllowedFactors means the
// policy does not gate enrollment — every implemented factor is allowed
// (backward compatible: a project with no mfa_policy doc behaves as before).
func (c MFAPolicySpec) FactorAllowed(dbFactorType string) bool {
	if len(c.AllowedFactors) == 0 {
		return true
	}
	name := MFAPolicyFactorName(dbFactorType)
	for _, f := range c.AllowedFactors {
		if f == name {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// rate_limits doc — iam_config key="rate_limits"
// ---------------------------------------------------------------------------

// RateLimitRuleSpec mirrors one element of rate_limits.rules[]. Tags match oas
// RateLimitRule.
type RateLimitRuleSpec struct {
	Endpoint      *string `json:"endpoint,omitempty"`
	Action        *string `json:"action,omitempty"`
	Limit         *int    `json:"limit,omitempty"`
	WindowSeconds *int    `json:"window_seconds,omitempty"`
	By            *string `json:"by,omitempty"`
}

// RateLimitsSpec mirrors the rate_limits jsonb doc.
type RateLimitsSpec struct {
	Rules []RateLimitRuleSpec `json:"rules,omitempty"`
}

// ParseRateLimits strictly decodes the rate_limits doc, rejecting unknown keys
// at both the top level and inside each rule.
func ParseRateLimits(raw []byte) (RateLimitsSpec, error) {
	var c RateLimitsSpec
	if err := strictUnmarshal(raw, &c); err != nil {
		return c, ErrValidation.WithMessage("invalid rate_limits: " + err.Error())
	}
	return c, nil
}

// Validate enforces fail-closed rules: by must be "ip", endpoint must be a
// realized path, action must be empty (unsupported), limit/window positive.
func (c RateLimitsSpec) Validate() error {
	const (
		maxLimit  = 1000000
		maxWindow = 86400
	)
	type tuple struct{ endpoint, by string }
	seen := make(map[tuple]struct{}, len(c.Rules))
	for i, r := range c.Rules {
		at := func(s string) string { return fmt.Sprintf("rules[%d].%s", i, s) }

		if r.By == nil || strings.TrimSpace(*r.By) == "" {
			return ErrValidation.WithMessage(at("by") + " is required")
		}
		if !RateLimitBy.Has(*r.By) {
			return ErrValidation.WithDetails(map[string]any{
				"field":   at("by"),
				"value":   *r.By,
				"allowed": RateLimitBy.List(),
			}).WithMessage(at("by") + ": unsupported subject")
		}

		if r.Endpoint == nil || strings.TrimSpace(*r.Endpoint) == "" {
			return ErrValidation.WithMessage(at("endpoint") + " is required")
		}
		if !RateLimitEndpoints.Has(*r.Endpoint) {
			return ErrValidation.WithDetails(map[string]any{
				"field": at("endpoint"),
				"value": *r.Endpoint,
			}).WithMessage(at("endpoint") + ": unsupported endpoint")
		}

		if r.Action != nil && strings.TrimSpace(*r.Action) != "" {
			// RateLimitActions is empty: actions are unsupported.
			if !RateLimitActions.Has(*r.Action) {
				return ErrValidation.WithMessage(at("action") + " not supported yet")
			}
		}

		if r.Limit == nil || *r.Limit < 1 {
			return ErrValidation.WithMessage(at("limit") + " must be >= 1")
		}
		if *r.Limit > maxLimit {
			return ErrValidation.WithMessage(at("limit") + " exceeds maximum")
		}

		if r.WindowSeconds == nil || *r.WindowSeconds < 1 {
			return ErrValidation.WithMessage(at("window_seconds") + " must be >= 1")
		}
		if *r.WindowSeconds > maxWindow {
			return ErrValidation.WithMessage(at("window_seconds") + " exceeds maximum")
		}

		t := tuple{*r.Endpoint, *r.By}
		if _, dup := seen[t]; dup {
			return ErrValidation.WithDetails(map[string]any{
				"endpoint": *r.Endpoint,
				"by":       *r.By,
			}).WithMessage("duplicate (endpoint, by) rule")
		}
		seen[t] = struct{}{}
	}
	return nil
}

// ---------------------------------------------------------------------------
// consent doc — iam_config key="consent"
// ---------------------------------------------------------------------------

// ConsentDocumentSpec mirrors one consent document. Tags match the lowercase
// oas ConsentDocument / platform jsonb keys.
type ConsentDocumentSpec struct {
	Key      string  `json:"key"`
	Version  string  `json:"version"`
	Title    *string `json:"title,omitempty"`
	Body     *string `json:"body,omitempty"`
	Locale   *string `json:"locale,omitempty"`
	Required *bool   `json:"required,omitempty"`
	URL      *string `json:"url,omitempty"`
}

// ConsentConfigSpec mirrors the consent jsonb doc.
type ConsentConfigSpec struct {
	Documents []ConsentDocumentSpec `json:"documents,omitempty"`
}

// ParseConsentConfig strictly decodes the consent doc.
func ParseConsentConfig(raw []byte) (ConsentConfigSpec, error) {
	var c ConsentConfigSpec
	if err := strictUnmarshal(raw, &c); err != nil {
		return c, ErrValidation.WithMessage("invalid consent config: " + err.Error())
	}
	return c, nil
}

// Validate enforces presentability and uniqueness rules fail-closed.
func (c ConsentConfigSpec) Validate() error {
	const (
		maxDocs = 200
		maxBody = 64 * 1024
	)
	if len(c.Documents) > maxDocs {
		return ErrValidation.WithMessage("too many consent documents")
	}
	type tuple struct{ key, locale, version string }
	seen := make(map[tuple]struct{}, len(c.Documents))
	for i, d := range c.Documents {
		at := func(s string) string { return fmt.Sprintf("documents[%d].%s", i, s) }

		key := strings.TrimSpace(d.Key)
		if key == "" {
			return ErrValidation.WithMessage(at("key") + " is required")
		}
		version := strings.TrimSpace(d.Version)
		if version == "" {
			return ErrValidation.WithMessage(at("version") + " is required")
		}

		required := d.Required != nil && *d.Required
		if required {
			if d.Title == nil || strings.TrimSpace(*d.Title) == "" {
				return ErrValidation.WithMessage(at("title") + " is required for a required consent")
			}
			hasBody := d.Body != nil && strings.TrimSpace(*d.Body) != ""
			hasURL := d.URL != nil && strings.TrimSpace(*d.URL) != ""
			if !hasBody && !hasURL {
				return ErrValidation.WithMessage(at("body") + " or " + at("url") + " is required for a required consent")
			}
		}

		if d.Body != nil && len(*d.Body) > maxBody {
			return ErrValidation.WithMessage(at("body") + " exceeds maximum size")
		}
		if d.Locale != nil && strings.TrimSpace(*d.Locale) == "" && *d.Locale != "" {
			return ErrValidation.WithMessage(at("locale") + " must not be blank")
		}
		if d.URL != nil && strings.TrimSpace(*d.URL) != "" {
			if err := validateAbsoluteHTTPURL(at("url"), *d.URL); err != nil {
				return err
			}
		}

		locale := ""
		if d.Locale != nil {
			locale = *d.Locale
		}
		t := tuple{key, locale, version}
		if _, dup := seen[t]; dup {
			return ErrValidation.WithDetails(map[string]any{
				"key": key, "locale": locale, "version": version,
			}).WithMessage("duplicate consent document (key, locale, version)")
		}
		seen[t] = struct{}{}
	}
	return nil
}

// ---------------------------------------------------------------------------
// features doc — iam_config key="features"
// ---------------------------------------------------------------------------

// FeaturesSpec mirrors the features jsonb doc (a flat map[string]bool).
type FeaturesSpec map[string]bool

// ParseFeatures decodes the features doc.
func ParseFeatures(raw []byte) (FeaturesSpec, error) {
	var c FeaturesSpec
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, ErrValidation.WithMessage("invalid features: " + err.Error())
	}
	return c, nil
}

// Validate rejects any key outside the canonical FeatureKeys registry.
func (c FeaturesSpec) Validate() error {
	var unknown []string
	for k := range c {
		if !FeatureKeys.Has(k) {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return ErrValidation.WithDetails(map[string]any{"unknown_features": unknown}).
			WithMessage("unknown feature(s): " + strings.Join(unknown, ", "))
	}
	return nil
}

// ---------------------------------------------------------------------------
// notification provider doc — iam_providers.data
// ---------------------------------------------------------------------------

// ProviderConfigSpec validates an email/sms provider's type against the
// realized registry for the given kind. kind is "email" or "sms"; typ is the
// provider type string (e.g. "smtp"). Comparison is case-insensitive and the
// canonical form is lowercase.
type ProviderConfigSpec struct {
	Kind string // "email" | "sms"
	Type string
}

// Validate rejects unsupported provider types fail-closed, checking the type
// against the realized registry for the given kind (email/sms).
func (c ProviderConfigSpec) Validate() error {
	typ := strings.ToLower(strings.TrimSpace(c.Type))
	if typ == "" {
		return ErrValidation.WithMessage("provider type is required")
	}
	switch strings.ToLower(strings.TrimSpace(c.Kind)) {
	case "email":
		if !EmailProviderTypes.Has(typ) {
			return ErrValidation.WithDetails(map[string]any{
				"field": "type", "value": typ, "allowed": EmailProviderTypes.List(),
			}).WithMessage("unsupported email provider type: " + typ)
		}
	case "sms":
		if !SMSProviderTypes.Has(typ) {
			return ErrValidation.WithDetails(map[string]any{
				"field": "type", "value": typ, "allowed": SMSProviderTypes.List(),
			}).WithMessage("unsupported sms provider type: " + typ)
		}
	default:
		return ErrValidation.WithMessage("unknown provider kind: " + c.Kind)
	}
	return nil
}
