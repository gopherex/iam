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

func TestEmailJobFromEventFlowContinue(t *testing.T) {
	job, ok := emailJobFromEvent(eventEnvelope{
		Type:    "auth.flow.continue",
		Payload: map[string]any{"to": "user@example.com", "flow_token": "ftk_abc", "kind": "signup"},
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

func TestFlowContinueURL(t *testing.T) {
	if got := flowContinueURL("https://app.example.com", "ftk_abc"); got != "https://app.example.com/continue?flow=ftk_abc" {
		t.Fatalf("url = %q", got)
	}
	if flowContinueURL("", "ftk_abc") != "" {
		t.Fatal("empty base must yield empty link")
	}
	if flowContinueURL("https://app.example.com", "") != "" {
		t.Fatal("empty token must yield empty link")
	}
	if flowContinueURL("not-a-url", "ftk_abc") != "" {
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
