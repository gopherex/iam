package notifications

import "testing"

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
	got, err := render("Code: {{.code}}", map[string]any{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Code: 123456" {
		t.Fatalf("rendered = %q", got)
	}
}
