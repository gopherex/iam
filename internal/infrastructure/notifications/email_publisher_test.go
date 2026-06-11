package notifications

import (
	"strings"
	"testing"
)

func TestEmailJobFromEventMagicLink(t *testing.T) {
	job, ok := emailJobFromEvent(eventEnvelope{
		Type: "auth.magiclink.started",
		Payload: map[string]any{
			"to":          "user@example.com",
			"token":       "tok_123",
			"redirect_to": "https://app.example.com/callback",
		},
	})
	if !ok {
		t.Fatal("expected email job")
	}
	if job.TemplateID != "magic_link" {
		t.Fatalf("template = %q", job.TemplateID)
	}
	if job.To != "user@example.com" {
		t.Fatalf("to = %q", job.To)
	}
	if got := job.Data["link"]; got != "https://app.example.com/callback?token=tok_123" {
		t.Fatalf("link = %v", got)
	}
}

func TestEmailJobFromEventFlowContinue(t *testing.T) {
	job, ok := emailJobFromEvent(eventEnvelope{
		Type:    "auth.flow.continue",
		Payload: map[string]any{"to": "user@example.com", "flow_token": "ftk_abc", "token": "proof_123", "code": "260129", "kind": "signup"},
	})
	if !ok {
		t.Fatal("expected email job")
	}
	if job.TemplateID != "flow_continue" {
		t.Fatalf("template = %q", job.TemplateID)
	}
	if job.To != "user@example.com" {
		t.Fatalf("to = %q", job.To)
	}
	if job.Data["token"] != "proof_123" {
		t.Fatalf("token = %v", job.Data["token"])
	}
	if job.Data["code"] != "260129" {
		t.Fatalf("code = %v", job.Data["code"])
	}
}

func TestSameOrigin(t *testing.T) {
	allow := []struct{ a, b string }{
		{"https://app.example.com/continue", "https://app.example.com"},
		{"https://app.example.com/x", "https://app.example.com/y"},
		{"https://APP.example.com", "https://app.example.com"}, // host case-insensitive
		{"https://app.example.com:8443/x", "https://app.example.com:8443"},
	}
	for _, c := range allow {
		if !sameOrigin(c.a, c.b) {
			t.Errorf("sameOrigin(%q,%q) = false, want true", c.a, c.b)
		}
	}
	deny := []struct{ a, b string }{
		{"https://evil.com", "https://app.example.com"},             // foreign host (phishing)
		{"http://app.example.com", "https://app.example.com"},       // scheme mismatch
		{"https://app.example.com:9000", "https://app.example.com"}, // port mismatch
		{"", "https://app.example.com"},                             // empty
		{"not-a-url", "https://app.example.com"},                    // unparseable
	}
	for _, c := range deny {
		if sameOrigin(c.a, c.b) {
			t.Errorf("sameOrigin(%q,%q) = true, want false", c.a, c.b)
		}
	}
}

func TestDefaultTemplateLocale(t *testing.T) {
	en := defaultTemplate("otp", "en")
	if en["subject"] != "Your sign-in code" {
		t.Fatalf("en subject = %q", en["subject"])
	}
	ru := defaultTemplate("otp", "ru")
	if ru["subject"] != "Код для входа" {
		t.Fatalf("ru subject = %q", ru["subject"])
	}
	// Region subtag falls back to base language.
	if defaultTemplate("otp", "ru-RU")["subject"] != ru["subject"] {
		t.Fatal("ru-RU should fall back to ru")
	}
	// Unknown locale falls back to English.
	if defaultTemplate("otp", "de")["subject"] != en["subject"] {
		t.Fatal("unknown locale should fall back to en")
	}
	// Unknown key falls back to email_verification (still localized).
	if defaultTemplate("nope", "ru")["subject"] == "" {
		t.Fatal("unknown key must still yield a subject")
	}
}

func TestFlowContinueURL(t *testing.T) {
	if got := flowContinueURL("https://app.example.com", "ftk_abc", "proof_123"); got != "https://app.example.com/continue?flow=ftk_abc&token=proof_123" {
		t.Fatalf("url = %q", got)
	}
	if got := flowContinueURL("https://app.example.com", "ftk_abc", ""); got != "https://app.example.com/continue?flow=ftk_abc" {
		t.Fatalf("url without proof = %q", got)
	}
	if flowContinueURL("", "ftk_abc", "proof_123") != "" {
		t.Fatal("empty base must yield empty link")
	}
	if flowContinueURL("https://app.example.com", "", "proof_123") != "" {
		t.Fatal("empty token must yield empty link")
	}
	if flowContinueURL("not-a-url", "ftk_abc", "proof_123") != "" {
		t.Fatal("bad base must yield empty link")
	}
}

func TestEmailJobFromEventIgnoresSMSOTP(t *testing.T) {
	_, ok := emailJobFromEvent(eventEnvelope{
		Type:    "auth.otp.started",
		Payload: map[string]any{"channel": "sms", "to": "+15555550123", "code": "123456"},
	})
	if ok {
		t.Fatal("sms otp should not become an email job")
	}
}

func TestRenderTemplate(t *testing.T) {
	got, err := renderText("Code: {{.code}}", map[string]any{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Code: 123456" {
		t.Fatalf("rendered = %q", got)
	}
}

func TestCodeLinkDefaultTemplates(t *testing.T) {
	cases := []struct {
		key          string
		noLinkText   string
		withLinkText string
	}{
		{
			key:          "email_verification",
			noLinkText:   "Введите код 260129, чтобы подтвердить почту.",
			withLinkText: "Введите код 260129, чтобы подтвердить почту.\nИли откройте ссылку: https://app.example.com/verify?token=tok_123",
		},
		{
			key:          "email_change",
			noLinkText:   "Введите код 260129, чтобы подтвердить новую почту.",
			withLinkText: "Введите код 260129, чтобы подтвердить новую почту.\nИли откройте ссылку: https://app.example.com/verify?token=tok_123",
		},
		{
			key:          "password_reset",
			noLinkText:   "Введите код 260129, чтобы сбросить пароль.",
			withLinkText: "Введите код 260129, чтобы сбросить пароль.\nИли откройте ссылку: https://app.example.com/verify?token=tok_123",
		},
	}
	for _, tc := range cases {
		t.Run(tc.key+"/text_without_link", func(t *testing.T) {
			tpl := defaultTemplate(tc.key, "ru")
			got, err := renderText(tpl["text"], map[string]any{"code": "260129"})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.noLinkText {
				t.Fatalf("rendered without link = %q", got)
			}
			if strings.Contains(got, "<no value>") {
				t.Fatalf("rendered without link contains missing value marker: %q", got)
			}
		})
		t.Run(tc.key+"/text_with_link", func(t *testing.T) {
			tpl := defaultTemplate(tc.key, "ru")
			got, err := renderText(tpl["text"], map[string]any{
				"code": "260129",
				"link": "https://app.example.com/verify?token=tok_123",
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.withLinkText {
				t.Fatalf("rendered with link = %q", got)
			}
		})
		t.Run(tc.key+"/html_without_link", func(t *testing.T) {
			tpl := defaultTemplate(tc.key, "ru")
			got, err := renderHTML(tpl["html"], map[string]any{"code": "260129"})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, "<strong>260129</strong>") {
				t.Fatalf("rendered html without link must contain code: %q", got)
			}
			if strings.Contains(got, "<a ") || strings.Contains(got, "<no value>") {
				t.Fatalf("rendered html without link must not contain link/missing marker: %q", got)
			}
		})
		t.Run(tc.key+"/html_with_link", func(t *testing.T) {
			tpl := defaultTemplate(tc.key, "ru")
			got, err := renderHTML(tpl["html"], map[string]any{
				"code": "260129",
				"link": "https://app.example.com/verify?token=tok_123",
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, "<strong>260129</strong>") || !strings.Contains(got, "https://app.example.com/verify?token=tok_123") {
				t.Fatalf("rendered html with link must contain code and link: %q", got)
			}
		})
	}
}

func TestFlowContinueDefaultTemplate(t *testing.T) {
	tpl := defaultTemplate("flow_continue", "ru")
	got, err := renderText(tpl["text"], map[string]any{"code": "260129"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Введите код 260129, чтобы продолжить." {
		t.Fatalf("rendered without link = %q", got)
	}
	if strings.Contains(got, "<no value>") {
		t.Fatalf("rendered without link contains missing value marker: %q", got)
	}

	got, err = renderText(tpl["text"], map[string]any{
		"code":         "260129",
		"continue_url": "https://app.example.com/continue?flow=ftk_abc&token=proof_123",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "Введите код 260129, чтобы продолжить.\nИли откройте ссылку: https://app.example.com/continue?flow=ftk_abc&token=proof_123"
	if got != want {
		t.Fatalf("rendered with link = %q", got)
	}

	html, err := renderHTML(tpl["html"], map[string]any{
		"code":         "260129",
		"continue_url": "https://app.example.com/continue?flow=ftk_abc&token=proof_123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<strong>260129</strong>") || !strings.Contains(html, "https://app.example.com/continue?flow=ftk_abc&amp;token=proof_123") {
		t.Fatalf("rendered html with link must contain code and escaped link: %q", html)
	}
}

func TestMagicLinkDefaultTemplate(t *testing.T) {
	tpl := defaultTemplate("magic_link", "ru")
	got, err := renderText(tpl["text"], map[string]any{
		"link": "https://app.example.com/auth/magic?token=tok_123",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "Откройте ссылку, чтобы войти: https://app.example.com/auth/magic?token=tok_123"
	if got != want {
		t.Fatalf("rendered text = %q", got)
	}

	html, err := renderHTML(tpl["html"], map[string]any{
		"link": "https://app.example.com/auth/magic?token=tok_123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "https://app.example.com/auth/magic?token=tok_123") {
		t.Fatalf("rendered html must contain link: %q", html)
	}
}

func TestInviteDefaultTemplate(t *testing.T) {
	tpl := defaultTemplate("invite", "ru")
	got, err := renderText(tpl["text"], map[string]any{
		"invite_token": "inv_abc",
		"invite_url":   "https://app.example.com/invite?token=inv_abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "Вас пригласили. Примите приглашение: https://app.example.com/invite?token=inv_abc (или используйте код inv_abc)"
	if got != want {
		t.Fatalf("rendered text = %q", got)
	}

	html, err := renderHTML(tpl["html"], map[string]any{
		"invite_token": "inv_abc",
		"invite_url":   "https://app.example.com/invite?token=inv_abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "https://app.example.com/invite?token=inv_abc") || !strings.Contains(html, "<strong>inv_abc</strong>") {
		t.Fatalf("rendered html must contain link and code: %q", html)
	}
}

func TestOTPDefaultTemplate(t *testing.T) {
	tpl := defaultTemplate("otp", "ru")
	html, err := renderHTML(tpl["html"], map[string]any{"code": "260129"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<strong>260129</strong>") {
		t.Fatalf("rendered html must contain code: %q", html)
	}
}
