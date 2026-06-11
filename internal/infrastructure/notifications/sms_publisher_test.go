package notifications

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/infrastructure/postgres"
)

func TestSMSJobFromEventOTP(t *testing.T) {
	job, ok := smsJobFromEvent(eventEnvelope{
		Type: "auth.otp.started",
		Payload: map[string]any{
			"channel": "sms",
			"code":    "123456",
			"to":      "+15551234567",
			"contact": "+15551234567",
		},
	})
	if !ok {
		t.Fatal("expected sms job")
	}
	if job.TemplateID != "otp" {
		t.Fatalf("template = %q", job.TemplateID)
	}
	if job.To != "+15551234567" {
		t.Fatalf("to = %q", job.To)
	}
	if job.Data["code"] != "123456" {
		t.Fatalf("code = %v", job.Data["code"])
	}
}

func TestSMSJobFromEventMFA(t *testing.T) {
	job, ok := smsJobFromEvent(eventEnvelope{
		Type:    "mfa.challenge.created",
		Payload: map[string]any{"channel": "sms", "code": "999", "to": "+1999"},
	})
	if !ok {
		t.Fatal("expected sms job")
	}
	if job.TemplateID != "mfa_sms" {
		t.Fatalf("template = %q", job.TemplateID)
	}
	if job.To != "+1999" {
		t.Fatalf("to = %q", job.To)
	}
}

func TestSMSJobFromEventPhoneVerification(t *testing.T) {
	verify, ok := smsJobFromEvent(eventEnvelope{
		Type:    "phone.verification.requested",
		Payload: map[string]any{"channel": "sms", "code": "1", "contact": "+1", "purpose": "verify"},
	})
	if !ok || verify.TemplateID != "phone_verification" {
		t.Fatalf("verify template = %q ok=%v", verify.TemplateID, ok)
	}
	change, ok := smsJobFromEvent(eventEnvelope{
		Type:    "phone.verification.requested",
		Payload: map[string]any{"channel": "sms", "code": "1", "contact": "+1", "purpose": "change"},
	})
	if !ok || change.TemplateID != "phone_change" {
		t.Fatalf("change template = %q ok=%v", change.TemplateID, ok)
	}
}

func TestSMSJobFromEventIgnoresEmailChannel(t *testing.T) {
	if _, ok := smsJobFromEvent(eventEnvelope{
		Type:    "auth.otp.started",
		Payload: map[string]any{"channel": "email", "to": "user@example.com"},
	}); ok {
		t.Fatal("email-channel otp must not produce an sms job")
	}
	if _, ok := smsJobFromEvent(eventEnvelope{
		Type:    "auth.magiclink.started",
		Payload: map[string]any{"to": "user@example.com"},
	}); ok {
		t.Fatal("magic link must not produce an sms job")
	}
}

func TestPhoneRecipient(t *testing.T) {
	cases := []struct {
		payload map[string]any
		want    string
	}{
		{map[string]any{"to": "+15551112222"}, "+15551112222"},
		{map[string]any{"contact": "+1999"}, "+1999"},
		{map[string]any{"to": "user@example.com", "contact": "+1888"}, "+1888"},
		{map[string]any{"subject": "+1777"}, "+1777"},
		{map[string]any{"to": "user@example.com"}, ""},
		{map[string]any{}, ""},
	}
	for _, c := range cases {
		if got := phoneRecipient(c.payload); got != c.want {
			t.Errorf("phoneRecipient(%v) = %q, want %q", c.payload, got, c.want)
		}
	}
}

func TestDecodeSMSConfigGeneric(t *testing.T) {
	cipher := postgres.NewIdentityCipher()
	raw := func(m map[string]any) map[string]json.RawMessage {
		out := map[string]json.RawMessage{}
		for k, v := range m {
			b, _ := json.Marshal(v)
			out[k] = b
		}
		return out
	}

	cfg, err := decodeSMSConfig(cipher, "generic", raw(map[string]any{
		"url":        "https://gw.example.com/send",
		"from":       "IAM",
		"auth_token": "secret-token",
	}))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.URL != "https://gw.example.com/send" || cfg.AuthToken != "secret-token" {
		t.Fatalf("cfg = %+v", cfg)
	}

	if _, err := decodeSMSConfig(cipher, "generic", raw(map[string]any{"from": "IAM"})); err == nil {
		t.Fatal("expected error: missing url")
	}
	if _, err := decodeSMSConfig(cipher, "generic", raw(map[string]any{"url": "http://insecure.example.com"})); err == nil {
		t.Fatal("expected error: non-https url rejected")
	}
}

func TestDecodeSMSConfigTwilio(t *testing.T) {
	cipher := postgres.NewIdentityCipher()
	raw := func(m map[string]any) map[string]json.RawMessage {
		out := map[string]json.RawMessage{}
		for k, v := range m {
			b, _ := json.Marshal(v)
			out[k] = b
		}
		return out
	}

	cfg, err := decodeSMSConfig(cipher, "twilio", raw(map[string]any{
		"account_sid": "ACxxx",
		"auth_token":  "tok",
		"from":        "+15550000000",
	}))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Username != "ACxxx" || cfg.Password != "tok" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if !strings.Contains(cfg.URL, "/Accounts/ACxxx/Messages.json") {
		t.Fatalf("url = %q", cfg.URL)
	}

	if _, err := decodeSMSConfig(cipher, "twilio", raw(map[string]any{"account_sid": "ACxxx", "from": "+1"})); err == nil {
		t.Fatal("expected error: missing auth_token")
	}
}

func TestSMSSendGeneric(t *testing.T) {
	var gotAuth, gotCT string
	var body map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &smsConfig{Type: "generic", URL: srv.URL, From: "IAM", AuthToken: "tok123"}
	if err := cfg.send(context.Background(), "+1555", "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotAuth != "Bearer tok123" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if !strings.HasPrefix(gotCT, "application/json") {
		t.Fatalf("content-type = %q", gotCT)
	}
	if body["to"] != "+1555" || body["text"] != "hello" || body["from"] != "IAM" {
		t.Fatalf("body = %v", body)
	}

	// non-2xx → error
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	cfg.URL = bad.URL
	if err := cfg.send(context.Background(), "+1", "x"); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestSMSSendTwilio(t *testing.T) {
	var gotUser, gotPass, gotCT string
	var ok bool
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok = r.BasicAuth()
		gotCT = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		form = r.PostForm
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	cfg := &smsConfig{Type: "twilio", URL: srv.URL, From: "+15550000000", Username: "ACxxx", Password: "tok"}
	if err := cfg.send(context.Background(), "+15551112222", "body text"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !ok || gotUser != "ACxxx" || gotPass != "tok" {
		t.Fatalf("basic auth = %q/%q ok=%v", gotUser, gotPass, ok)
	}
	if !strings.HasPrefix(gotCT, "application/x-www-form-urlencoded") {
		t.Fatalf("content-type = %q", gotCT)
	}
	if form.Get("From") != "+15550000000" || form.Get("To") != "+15551112222" || form.Get("Body") != "body text" {
		t.Fatalf("form = %v", form)
	}
}

func TestDefaultSMSText(t *testing.T) {
	en, err := renderText(defaultSMSText("otp", "en"), map[string]any{"code": "424242"})
	if err != nil {
		t.Fatalf("render en: %v", err)
	}
	if !strings.Contains(en, "424242") {
		t.Fatalf("en body = %q", en)
	}
	ru, err := renderText(defaultSMSText("otp", "ru"), map[string]any{"code": "424242"})
	if err != nil {
		t.Fatalf("render ru: %v", err)
	}
	if !strings.Contains(ru, "424242") || !strings.Contains(ru, "код") {
		t.Fatalf("ru body = %q", ru)
	}
}
