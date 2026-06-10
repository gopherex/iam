package domain

// BuiltinEmailCopy is the localized copy of a built-in email template.
type BuiltinEmailCopy struct {
	Subject string
	Text    string
	HTML    string
}

// BuiltinEmailTemplate is a system-provided email template: the default copy the
// notification renderer falls back to when a project has not customised the
// template for a given key/locale. The admin API lists, previews, and test-sends
// these so operators can see and exercise every template (and verify SMTP)
// before overriding any of them. Copy is keyed by locale; "en" is the canonical
// fallback and must always be present.
type BuiltinEmailTemplate struct {
	Key     string
	Name    string
	Locales map[string]BuiltinEmailCopy
}

// builtinEmailLocaleFallback is the locale used when neither the requested locale
// nor any other resolution yields a built-in translation.
const builtinEmailLocaleFallback = "en"

// BuiltinEmailTemplates is the canonical catalogue, keyed by the template ids the
// notification layer emits. Slice order is the admin display order. Keep the copy
// in sync with the renderer fallbacks.
var BuiltinEmailTemplates = []BuiltinEmailTemplate{
	{
		Key:  "email_verification",
		Name: "Email verification",
		Locales: map[string]BuiltinEmailCopy{
			"en": {Subject: "Verify your email", Text: "Use code {{.code}} or open {{.link}} to verify your email."},
			"ru": {Subject: "Подтвердите вашу почту", Text: "Введите код {{.code}} или откройте {{.link}}, чтобы подтвердить почту."},
		},
	},
	{
		Key:  "otp",
		Name: "Sign-in code (OTP)",
		Locales: map[string]BuiltinEmailCopy{
			"en": {Subject: "Your sign-in code", Text: "Your code is {{.code}}."},
			"ru": {Subject: "Код для входа", Text: "Ваш код: {{.code}}."},
		},
	},
	{
		Key:  "magic_link",
		Name: "Magic link",
		Locales: map[string]BuiltinEmailCopy{
			"en": {Subject: "Your sign-in link", Text: "Open this link to sign in: {{.link}}", HTML: `<p>Open this link to sign in: <a href="{{.link}}">{{.link}}</a></p>`},
			"ru": {Subject: "Ссылка для входа", Text: "Откройте ссылку, чтобы войти: {{.link}}", HTML: `<p>Откройте ссылку, чтобы войти: <a href="{{.link}}">{{.link}}</a></p>`},
		},
	},
	{
		Key:  "email_change",
		Name: "Email change",
		Locales: map[string]BuiltinEmailCopy{
			"en": {Subject: "Confirm your new email", Text: "Use code {{.code}} or open {{.link}} to confirm your new email."},
			"ru": {Subject: "Подтвердите новую почту", Text: "Введите код {{.code}} или откройте {{.link}}, чтобы подтвердить новую почту."},
		},
	},
	{
		Key:  "password_reset",
		Name: "Password reset",
		Locales: map[string]BuiltinEmailCopy{
			"en": {Subject: "Reset your password", Text: "Use code {{.code}} or open {{.link}} to reset your password."},
			"ru": {Subject: "Сброс пароля", Text: "Введите код {{.code}} или откройте {{.link}}, чтобы сбросить пароль."},
		},
	},
	{
		Key:  "mfa_email",
		Name: "MFA code",
		Locales: map[string]BuiltinEmailCopy{
			"en": {Subject: "Your MFA code", Text: "Your MFA code is {{.code}}."},
			"ru": {Subject: "Код двухфакторной аутентификации", Text: "Ваш код подтверждения: {{.code}}."},
		},
	},
	{
		Key:  "flow_continue",
		Name: "Continue on another device",
		Locales: map[string]BuiltinEmailCopy{
			"en": {Subject: "Continue where you left off", Text: "Continue on this or another device: {{.continue_url}}", HTML: `<p>Continue where you left off: <a href="{{.continue_url}}">{{.continue_url}}</a></p>`},
			"ru": {Subject: "Продолжите с того же места", Text: "Продолжите на этом или другом устройстве: {{.continue_url}}", HTML: `<p>Продолжите с того же места: <a href="{{.continue_url}}">{{.continue_url}}</a></p>`},
		},
	},
}

// BuiltinEmailTemplateByKey returns the catalogue entry for key, or nil when key
// is not a known built-in template.
func BuiltinEmailTemplateByKey(key string) *BuiltinEmailTemplate {
	for i := range BuiltinEmailTemplates {
		if BuiltinEmailTemplates[i].Key == key {
			return &BuiltinEmailTemplates[i]
		}
	}
	return nil
}

// Copy returns the localized copy for locale, falling back to the base language
// (e.g. "ru-RU" -> "ru") and then to "en".
func (t *BuiltinEmailTemplate) Copy(locale string) BuiltinEmailCopy {
	if c, ok := t.Locales[locale]; ok {
		return c
	}
	if base := baseLocale(locale); base != "" && base != locale {
		if c, ok := t.Locales[base]; ok {
			return c
		}
	}
	return t.Locales[builtinEmailLocaleFallback]
}

// baseLocale reduces a BCP-47 tag to its primary subtag ("ru-RU" -> "ru").
func baseLocale(locale string) string {
	for i := 0; i < len(locale); i++ {
		if locale[i] == '-' || locale[i] == '_' {
			return locale[:i]
		}
	}
	return locale
}

// FirstNonEmptyLocale returns the first non-empty locale from candidates, or "en".
func FirstNonEmptyLocale(candidates ...string) string {
	for _, c := range candidates {
		if c != "" {
			return c
		}
	}
	return builtinEmailLocaleFallback
}
