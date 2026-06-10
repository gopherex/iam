package domain

// BuiltinEmailTemplate is a system-provided email template: the default copy the
// notification renderer falls back to when a project has not customised the
// template for a given key. The admin API lists, previews, and test-sends these
// so operators can see and exercise every template (and verify SMTP) before
// overriding any of them.
type BuiltinEmailTemplate struct {
	Key     string
	Name    string
	Subject string
	Text    string
	HTML    string
}

// BuiltinEmailTemplates is the canonical catalogue, keyed by the template ids the
// notification layer emits. Slice order is the admin display order. Keep the copy
// in sync with the renderer's fallbacks.
var BuiltinEmailTemplates = []BuiltinEmailTemplate{
	{
		Key:     "email_verification",
		Name:    "Email verification",
		Subject: "Verify your email",
		Text:    "Use code {{.code}} or open {{.link}} to verify your email.",
	},
	{
		Key:     "otp",
		Name:    "Sign-in code (OTP)",
		Subject: "Your sign-in code",
		Text:    "Your code is {{.code}}.",
	},
	{
		Key:     "magic_link",
		Name:    "Magic link",
		Subject: "Your sign-in link",
		Text:    "Open this link to sign in: {{.link}}",
		HTML:    `<p>Open this link to sign in: <a href="{{.link}}">{{.link}}</a></p>`,
	},
	{
		Key:     "email_change",
		Name:    "Email change",
		Subject: "Confirm your new email",
		Text:    "Use code {{.code}} or open {{.link}} to confirm your new email.",
	},
	{
		Key:     "password_reset",
		Name:    "Password reset",
		Subject: "Reset your password",
		Text:    "Use code {{.code}} or open {{.link}} to reset your password.",
	},
	{
		Key:     "mfa_email",
		Name:    "MFA code",
		Subject: "Your MFA code",
		Text:    "Your MFA code is {{.code}}.",
	},
	{
		Key:     "flow_continue",
		Name:    "Continue on another device",
		Subject: "Continue where you left off",
		Text:    "Continue on this or another device: {{.continue_url}}",
		HTML:    `<p>Continue where you left off: <a href="{{.continue_url}}">{{.continue_url}}</a></p>`,
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
