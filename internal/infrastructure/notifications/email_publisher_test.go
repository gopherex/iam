package notifications

import (
	"strings"
	"testing"
)

func TestEmailJobFromEventMagicLink(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	_, ok := emailJobFromEvent(eventEnvelope{
		Type:    "auth.otp.started",
		Payload: map[string]any{"channel": "sms", "to": "+15555550123", "code": "123456"},
	})
	if ok {
		t.Fatal("sms otp should not become an email job")
	}
}

func TestRenderTemplate(t *testing.T) {
	t.Parallel()

	got, err := renderText("Code: {{.code}}", map[string]any{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}

	if got != "Code: 123456" {
		t.Fatalf("rendered = %q", got)
	}
}

func TestCodeLinkDefaultTemplates(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
			t.Parallel()

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
			t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestEmailJobFromEventAccessRequestDecision(t *testing.T) {
	t.Parallel()

	// domain.CoreAuthAccessRequest has no json tags, so the envelope carries
	// capitalized keys — the recipient lookup must accept that spelling.
	payload := map[string]any{
		"ID":        "ar_123",
		"ProjectID": "prj_1",
		"Email":     "user@example.com",
		"Status":    "approved",
		"Locale":    "ru",
	}

	job, ok := emailJobFromEvent(eventEnvelope{Type: "access_request.approved", Payload: payload})
	if !ok {
		t.Fatal("expected email job")
	}

	if job.TemplateID != "access_request_approved" {
		t.Fatalf("template = %q", job.TemplateID)
	}

	if job.To != "user@example.com" {
		t.Fatalf("to = %q", job.To)
	}

	if job.Locale != "ru" {
		t.Fatalf("locale = %q, want ru (requester's submission locale)", job.Locale)
	}

	job, ok = emailJobFromEvent(eventEnvelope{
		Type:    "access_request.denied",
		Payload: map[string]any{"Email": "user@example.com", "Reason": "not in team"},
	})
	if !ok {
		t.Fatal("expected email job")
	}

	if job.TemplateID != "access_request_denied" {
		t.Fatalf("template = %q", job.TemplateID)
	}

	if job.Data["reason"] != "not in team" {
		t.Fatalf("reason = %v", job.Data["reason"])
	}
}

func TestEmailJobFromEventAccessRequestNoRecipient(t *testing.T) {
	t.Parallel()

	job, ok := emailJobFromEvent(eventEnvelope{
		Type:    "access_request.approved",
		Payload: map[string]any{"ID": "ar_123", "Status": "approved"},
	})
	if !ok {
		t.Fatal("expected email job (recipient checked by publishOne)")
	}

	if job.To != "" {
		t.Fatalf("to = %q, want empty", job.To)
	}
}

func TestAccessDecisionDefaultTemplates(t *testing.T) {
	t.Parallel()

	approved := defaultTemplate("access_request_approved", "ru")

	got, err := renderText(approved["text"], map[string]any{"link": "https://app.example.com"})
	if err != nil {
		t.Fatal(err)
	}

	want := "Хорошая новость — ваша заявка на доступ одобрена.\nПродолжите регистрацию: https://app.example.com"
	if got != want {
		t.Fatalf("approved text = %q", got)
	}

	// Without a configured app base URL the email still sends, just link-less.
	got, err = renderText(approved["text"], nil)
	if err != nil {
		t.Fatal(err)
	}

	if got != "Хорошая новость — ваша заявка на доступ одобрена." || strings.Contains(got, "<no value>") {
		t.Fatalf("approved text without link = %q", got)
	}

	denied := defaultTemplate("access_request_denied", "ru")

	got, err = renderText(denied["text"], map[string]any{"reason": "не входит в команду"})
	if err != nil {
		t.Fatal(err)
	}

	want = "Ваша заявка на доступ отклонена.\nПричина: не входит в команду"
	if got != want {
		t.Fatalf("denied text = %q", got)
	}

	got, err = renderText(denied["text"], nil)
	if err != nil {
		t.Fatal(err)
	}

	if got != "Ваша заявка на доступ отклонена." || strings.Contains(got, "<no value>") {
		t.Fatalf("denied text without reason = %q", got)
	}
}

func TestIsAccessDecisionEvent(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"access_request.approved", "access_request.denied"} {
		if !isAccessDecisionEvent(typ) {
			t.Errorf("isAccessDecisionEvent(%q) = false", typ)
		}
	}

	for _, typ := range []string{"access_request.created", "invite.created", ""} {
		if isAccessDecisionEvent(typ) {
			t.Errorf("isAccessDecisionEvent(%q) = true", typ)
		}
	}
}

func TestBuildMessageMultipart(t *testing.T) {
	t.Parallel()

	c := &smtpConfig{From: "noreply@example.com", FromName: "Example"}
	msg := renderedEmail{
		Subject: "Подтвердите почту",
		Text:    "Код: 260129",
		HTML:    `<p>Код: <strong>260129</strong></p>`,
	}

	raw := c.buildMessage("user@example.org", msg)

	s := string(raw)

	// Deterministic header order.
	for _, pair := range [][2]string{
		{"From:", "To:"},
		{"To:", "Subject:"},
		{"Subject:", "Date:"},
		{"Date:", "Message-ID:"},
		{"Message-ID:", "MIME-Version:"},
	} {
		if !strings.Contains(s, pair[0]+" ") || !strings.Contains(s, pair[1]+" ") ||
			strings.Index(s, pair[0]) > strings.Index(s, pair[1]) {
			t.Fatalf("headers %q before %q violated:\n%s", pair[0], pair[1], s)
		}
	}

	// Message-ID in the From domain, angle-bracketed.
	if !strings.Contains(s, "@example.com>\r\n") {
		t.Fatalf("message-id must end in @example.com>:\n%s", s)
	}

	// Both parts, text first, QP-encoded (Cyrillic must not appear raw).
	if !strings.Contains(s, `multipart/alternative; boundary="`) {
		t.Fatalf("expected multipart/alternative:\n%s", s)
	}

	if ti, hi := strings.Index(s, `text/plain`), strings.Index(s, `text/html`); ti < 0 || hi < 0 || ti > hi {
		t.Fatalf("text part must precede html part:\n%s", s)
	}

	if got := strings.Count(s, "quoted-printable"); got != 2 {
		t.Fatalf("quoted-printable count = %d, want 2:\n%s", got, s)
	}

	if strings.Contains(s, "Код:") {
		t.Fatalf("raw UTF-8 body leaked (must be QP-encoded):\n%s", s)
	}
}

func TestBuildMessageSinglePart(t *testing.T) {
	t.Parallel()

	c := &smtpConfig{From: "noreply@example.com"}

	raw := c.buildMessage("user@example.org", renderedEmail{Subject: "s", Text: "plain body"})
	if s := string(raw); !strings.Contains(s, `Content-Type: text/plain`) || strings.Contains(s, "multipart") {
		t.Fatalf("text-only message:\n%s", s)
	}

	raw = c.buildMessage("user@example.org", renderedEmail{Subject: "s", HTML: "<p>html</p>"})
	if s := string(raw); !strings.Contains(s, `Content-Type: text/html`) || strings.Contains(s, "multipart") {
		t.Fatalf("html-only message:\n%s", s)
	}
}

func TestMessageIDDomain(t *testing.T) {
	t.Parallel()

	if id := (&smtpConfig{From: "noreply@example.com"}).messageID(); !strings.HasPrefix(id, "<") || !strings.HasSuffix(id, "@example.com>") {
		t.Fatalf("id = %q", id)
	}

	if id := (&smtpConfig{}).messageID(); !strings.HasSuffix(id, "@localhost>") {
		t.Fatalf("id without from = %q", id)
	}
}

func TestOTPDefaultTemplate(t *testing.T) {
	t.Parallel()

	tpl := defaultTemplate("otp", "ru")

	html, err := renderHTML(tpl["html"], map[string]any{"code": "260129"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(html, "<strong>260129</strong>") {
		t.Fatalf("rendered html must contain code: %q", html)
	}
}
