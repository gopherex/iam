package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// ptr is a small generic helper for the *T fields all over these specs.
func ptr[T any](v T) *T { return &v }

// assertValidate runs spec.Validate() and asserts whether it should pass and,
// when it should fail, that the error is errors.Is(ErrValidation).
func assertValidate(t *testing.T, name string, err error, wantErr bool) {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Fatalf("%s: expected ErrValidation, got nil", name)
		}
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%s: expected errors.Is(err, ErrValidation), got %v", name, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("%s: expected no error, got %v", name, err)
	}
}

// ---------------------------------------------------------------------------
// AuthConfigSpec
// ---------------------------------------------------------------------------

func TestAuthConfigSpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    domain.AuthConfigSpec
		wantErr bool
	}{
		{
			name:    "empty is valid",
			spec:    domain.AuthConfigSpec{},
			wantErr: false,
		},
		{
			name:    "all supported methods",
			spec:    domain.AuthConfigSpec{Methods: []string{"email", "oauth", "passkey", "magic_link"}},
			wantErr: false,
		},
		{
			name:    "unknown method rejected",
			spec:    domain.AuthConfigSpec{Methods: []string{"email", "username"}},
			wantErr: true,
		},
		{
			name:    "webauthn is not an auth method (passkey is canonical)",
			spec:    domain.AuthConfigSpec{Methods: []string{"webauthn"}},
			wantErr: true,
		},
		{
			name:    "duplicate method rejected",
			spec:    domain.AuthConfigSpec{Methods: []string{"email", "email"}},
			wantErr: true,
		},
		{
			name:    "valid registration mode + strategy",
			spec:    domain.AuthConfigSpec{Registration: &domain.AuthRegistrationConfig{Mode: ptr("invite_only"), PasswordStrategy: ptr("after_verify")}},
			wantErr: false,
		},
		{
			name:    "empty registration mode tolerated",
			spec:    domain.AuthConfigSpec{Registration: &domain.AuthRegistrationConfig{Mode: ptr("")}},
			wantErr: false,
		},
		{
			name:    "unknown registration mode rejected",
			spec:    domain.AuthConfigSpec{Registration: &domain.AuthRegistrationConfig{Mode: ptr("self_serve")}},
			wantErr: true,
		},
		{
			name:    "unknown password strategy rejected",
			spec:    domain.AuthConfigSpec{Registration: &domain.AuthRegistrationConfig{PasswordStrategy: ptr("never")}},
			wantErr: true,
		},
		{
			name:    "valid absolute https app_base_url",
			spec:    domain.AuthConfigSpec{AppBaseURL: ptr("https://app.example.com")},
			wantErr: false,
		},
		{
			name:    "blank app_base_url tolerated",
			spec:    domain.AuthConfigSpec{AppBaseURL: ptr("   ")},
			wantErr: false,
		},
		{
			name:    "relative app_base_url rejected",
			spec:    domain.AuthConfigSpec{AppBaseURL: ptr("/login")},
			wantErr: true,
		},
		{
			name:    "non-http scheme app_base_url rejected",
			spec:    domain.AuthConfigSpec{AppBaseURL: ptr("ftp://app.example.com")},
			wantErr: true,
		},
		{
			name:    "default_locale in supported_locales",
			spec:    domain.AuthConfigSpec{DefaultLocale: ptr("en"), SupportedLocales: []string{"en", "ru"}},
			wantErr: false,
		},
		{
			name:    "default_locale honours legacy locales key",
			spec:    domain.AuthConfigSpec{DefaultLocale: ptr("ru"), Locales: []string{"en", "ru"}},
			wantErr: false,
		},
		{
			name:    "default_locale not in supported list rejected",
			spec:    domain.AuthConfigSpec{DefaultLocale: ptr("de"), SupportedLocales: []string{"en", "ru"}},
			wantErr: true,
		},
		{
			name:    "default_locale with no list tolerated",
			spec:    domain.AuthConfigSpec{DefaultLocale: ptr("de")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

func TestParseAuthConfig_RejectsUnknownFields(t *testing.T) {
	t.Parallel()
	_, err := domain.ParseAuthConfig([]byte(`{"methods":["email"],"bogus":true}`))
	assertValidate(t, "unknown top-level field", err, true)

	got, err := domain.ParseAuthConfig([]byte(`{"methods":["email","passkey"]}`))
	if err != nil {
		t.Fatalf("valid doc should parse: %v", err)
	}
	if len(got.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %v", got.Methods)
	}
}

// ---------------------------------------------------------------------------
// PasswordPolicySpec
// ---------------------------------------------------------------------------

func TestPasswordPolicySpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    domain.PasswordPolicySpec
		wantErr bool
	}{
		{"empty valid", domain.PasswordPolicySpec{}, false},
		{"min_length lower bound", domain.PasswordPolicySpec{MinLength: ptr(1)}, false},
		{"min_length upper bound", domain.PasswordPolicySpec{MinLength: ptr(256)}, false},
		{"min_length zero rejected", domain.PasswordPolicySpec{MinLength: ptr(0)}, true},
		{"min_length too large rejected", domain.PasswordPolicySpec{MinLength: ptr(257)}, true},
		{"zxcvbn 0 ok", domain.PasswordPolicySpec{ZxcvbnMinScore: ptr(0)}, false},
		{"zxcvbn 4 ok", domain.PasswordPolicySpec{ZxcvbnMinScore: ptr(4)}, false},
		{"zxcvbn negative rejected", domain.PasswordPolicySpec{ZxcvbnMinScore: ptr(-1)}, true},
		{"zxcvbn 5 rejected", domain.PasswordPolicySpec{ZxcvbnMinScore: ptr(5)}, true},
		{"breached_check false ok", domain.PasswordPolicySpec{BreachedCheck: ptr(false)}, false},
		{"breached_check true rejected (unimplemented)", domain.PasswordPolicySpec{BreachedCheck: ptr(true)}, true},
		{"history 0 ok", domain.PasswordPolicySpec{History: ptr(0)}, false},
		{"history >0 rejected (unimplemented)", domain.PasswordPolicySpec{History: ptr(5)}, true},
		{"history negative rejected", domain.PasswordPolicySpec{History: ptr(-1)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// SessionPolicySpec
// ---------------------------------------------------------------------------

func TestSessionPolicySpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    domain.SessionPolicySpec
		wantErr bool
	}{
		{"empty valid", domain.SessionPolicySpec{}, false},
		{
			"coherent ttls",
			domain.SessionPolicySpec{AccessTTL: ptr(900), IdleTimeout: ptr(3600), RefreshTTL: ptr(86400), AbsoluteTimeout: ptr(604800)},
			false,
		},
		{"access_ttl zero rejected", domain.SessionPolicySpec{AccessTTL: ptr(0)}, true},
		{"access_ttl negative rejected", domain.SessionPolicySpec{AccessTTL: ptr(-1)}, true},
		{"access_ttl exceeds max rejected", domain.SessionPolicySpec{AccessTTL: ptr(86401)}, true},
		{"refresh_ttl exceeds max rejected", domain.SessionPolicySpec{RefreshTTL: ptr(31536001)}, true},
		{
			"access >= refresh rejected",
			domain.SessionPolicySpec{AccessTTL: ptr(3600), RefreshTTL: ptr(3600)},
			true,
		},
		{
			"access > idle rejected",
			domain.SessionPolicySpec{AccessTTL: ptr(3600), IdleTimeout: ptr(1800)},
			true,
		},
		{
			"idle > absolute rejected",
			domain.SessionPolicySpec{IdleTimeout: ptr(7200), AbsoluteTimeout: ptr(3600)},
			true,
		},
		{
			"idle > refresh rejected",
			domain.SessionPolicySpec{IdleTimeout: ptr(7200), RefreshTTL: ptr(3600)},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// MFAPolicySpec
// ---------------------------------------------------------------------------

func TestMFAPolicySpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    domain.MFAPolicySpec
		wantErr bool
	}{
		{"empty valid", domain.MFAPolicySpec{}, false},
		{
			"all supported factors",
			domain.MFAPolicySpec{AllowedFactors: []string{"totp", "sms", "email_otp", "webauthn", "backup_codes"}},
			false,
		},
		{
			"unknown factor rejected",
			domain.MFAPolicySpec{AllowedFactors: []string{"totp", "yubikey"}},
			true,
		},
		{
			"email is not a policy factor (email_otp is canonical)",
			domain.MFAPolicySpec{AllowedFactors: []string{"email"}},
			true,
		},
		{
			"duplicate factor rejected",
			domain.MFAPolicySpec{AllowedFactors: []string{"totp", "totp"}},
			true,
		},
		{
			"empty factors + required => lockout rejected",
			domain.MFAPolicySpec{AllowedFactors: []string{}, RequiredForAdmins: ptr(true)},
			true,
		},
		{
			"factors present + required ok",
			domain.MFAPolicySpec{AllowedFactors: []string{"totp"}, RequiredForAdmins: ptr(true)},
			false,
		},
		{
			"nil factors + required ok (not present-but-empty)",
			domain.MFAPolicySpec{RequiredForAdmins: ptr(true)},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// RateLimitsSpec
// ---------------------------------------------------------------------------

func TestRateLimitsSpec_Validate(t *testing.T) {
	t.Parallel()
	validRule := func() domain.RateLimitRuleSpec {
		return domain.RateLimitRuleSpec{
			Endpoint:      ptr("/v1/auth/sign-in/password"),
			By:            ptr("ip"),
			Limit:         ptr(10),
			WindowSeconds: ptr(60),
		}
	}
	with := func(mut func(*domain.RateLimitRuleSpec)) domain.RateLimitsSpec {
		r := validRule()
		mut(&r)
		return domain.RateLimitsSpec{Rules: []domain.RateLimitRuleSpec{r}}
	}
	tests := []struct {
		name    string
		spec    domain.RateLimitsSpec
		wantErr bool
	}{
		{"empty valid", domain.RateLimitsSpec{}, false},
		{"valid rule", domain.RateLimitsSpec{Rules: []domain.RateLimitRuleSpec{validRule()}}, false},
		{"missing by rejected", with(func(r *domain.RateLimitRuleSpec) { r.By = nil }), true},
		{"unsupported by rejected", with(func(r *domain.RateLimitRuleSpec) { r.By = ptr("user") }), true},
		{"missing endpoint rejected", with(func(r *domain.RateLimitRuleSpec) { r.Endpoint = nil }), true},
		{"unsupported endpoint rejected", with(func(r *domain.RateLimitRuleSpec) { r.Endpoint = ptr("/v1/nope") }), true},
		{"action rejected (unimplemented)", with(func(r *domain.RateLimitRuleSpec) { r.Action = ptr("block") }), true},
		{"limit zero rejected", with(func(r *domain.RateLimitRuleSpec) { r.Limit = ptr(0) }), true},
		{"limit nil rejected", with(func(r *domain.RateLimitRuleSpec) { r.Limit = nil }), true},
		{"limit over max rejected", with(func(r *domain.RateLimitRuleSpec) { r.Limit = ptr(1000001) }), true},
		{"window zero rejected", with(func(r *domain.RateLimitRuleSpec) { r.WindowSeconds = ptr(0) }), true},
		{"window nil rejected", with(func(r *domain.RateLimitRuleSpec) { r.WindowSeconds = nil }), true},
		{"window over max rejected", with(func(r *domain.RateLimitRuleSpec) { r.WindowSeconds = ptr(86401) }), true},
		{
			"duplicate (endpoint,by) rejected",
			domain.RateLimitsSpec{Rules: []domain.RateLimitRuleSpec{validRule(), validRule()}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// ConsentConfigSpec
// ---------------------------------------------------------------------------

func TestConsentConfigSpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    domain.ConsentConfigSpec
		wantErr bool
	}{
		{"empty valid", domain.ConsentConfigSpec{}, false},
		{
			"minimal optional doc valid",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos", Version: "1"}}},
			false,
		},
		{
			"missing key rejected",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Version: "1"}}},
			true,
		},
		{
			"missing version rejected",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos"}}},
			true,
		},
		{
			"required without title rejected",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos", Version: "1", Required: ptr(true), Body: ptr("text")}}},
			true,
		},
		{
			"required without body or url rejected",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos", Version: "1", Required: ptr(true), Title: ptr("Terms")}}},
			true,
		},
		{
			"required with title + body valid",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos", Version: "1", Required: ptr(true), Title: ptr("Terms"), Body: ptr("text")}}},
			false,
		},
		{
			"required with title + url valid",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos", Version: "1", Required: ptr(true), Title: ptr("Terms"), URL: ptr("https://example.com/tos")}}},
			false,
		},
		{
			"required with relative url rejected",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos", Version: "1", Required: ptr(true), Title: ptr("Terms"), URL: ptr("/tos")}}},
			true,
		},
		{
			"duplicate (key,locale,version) rejected",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{
				{Key: "tos", Version: "1", Locale: ptr("en")},
				{Key: "tos", Version: "1", Locale: ptr("en")},
			}},
			true,
		},
		{
			"same key different locale valid",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{
				{Key: "tos", Version: "1", Locale: ptr("en")},
				{Key: "tos", Version: "1", Locale: ptr("ru")},
			}},
			false,
		},
		{
			"oversized body rejected",
			domain.ConsentConfigSpec{Documents: []domain.ConsentDocumentSpec{{Key: "tos", Version: "1", Body: ptr(strings.Repeat("x", 64*1024+1))}}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// FeaturesSpec
// ---------------------------------------------------------------------------

func TestFeaturesSpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    domain.FeaturesSpec
		wantErr bool
	}{
		{"empty valid", domain.FeaturesSpec{}, false},
		{"known features valid", domain.FeaturesSpec{"password": true, "mfa": false, "consent": true}, false},
		{"unknown key rejected", domain.FeaturesSpec{"password": true, "telepathy": true}, true},
		{"unknown key with false value still rejected", domain.FeaturesSpec{"telepathy": false}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// ProviderConfigSpec
// ---------------------------------------------------------------------------

func TestProviderConfigSpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    domain.ProviderConfigSpec
		wantErr bool
	}{
		{"email smtp valid", domain.ProviderConfigSpec{Kind: "email", Type: "smtp"}, false},
		{"email smtp case-insensitive valid", domain.ProviderConfigSpec{Kind: "EMAIL", Type: "SMTP"}, false},
		{"email ses rejected (unimplemented)", domain.ProviderConfigSpec{Kind: "email", Type: "ses"}, true},
		{"email empty type rejected", domain.ProviderConfigSpec{Kind: "email", Type: ""}, true},
		{"sms rejected wholesale", domain.ProviderConfigSpec{Kind: "sms", Type: "twilio"}, true},
		{"unknown kind rejected", domain.ProviderConfigSpec{Kind: "push", Type: "fcm"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertValidate(t, tt.name, tt.spec.Validate(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// Public-config method filtering.
//
// The actual filter lives in postgres.pgPlatform.PublicConfig, which requires a
// live DB and so cannot be exercised by a pure unit test here. That filter is
// nothing more than `domain.SupportedAuthMethods.Has(m)` applied over the stored
// methods slice (preserving order). This test pins that registry contract — the
// exact predicate the filter depends on — so a registry change that would break
// the filter is caught at the domain layer. The end-to-end filter path is
// covered by the HTTP e2e suite.
// ---------------------------------------------------------------------------

func TestSupportedAuthMethods_FilterContract(t *testing.T) {
	t.Parallel()
	stored := []string{"email", "username", "passkey", "phone", "magic_link", "oauth"}
	want := []string{"email", "passkey", "magic_link", "oauth"}

	var got []string
	for _, m := range stored {
		if domain.SupportedAuthMethods.Has(m) {
			got = append(got, m)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("filtered methods = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("filtered methods = %v, want %v (order preserved)", got, want)
		}
	}
}
