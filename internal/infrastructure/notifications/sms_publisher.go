package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gopherex/xlog"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/infrastructure/postgres"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// smsHTTPTimeout bounds every outbound SMS gateway call so a stuck provider
// cannot wedge the outbox consumer.
const smsHTTPTimeout = 10 * time.Second

// gatewayErrSnippetMaxBytes bounds how much of a failed gateway response body
// is quoted back into the error.
const gatewayErrSnippetMaxBytes = 512

// smsJob is the SMS analog of emailJob: a template key + the resolved phone
// recipient. The body is rendered from inline defaults (defaultSMSText) since
// SMS copy is short and few — no DB-backed catalog for v1.
type smsJob struct {
	TemplateID string
	To         string
	Locale     string
	Data       map[string]any
}

// smsJobFromEvent mirrors emailJobFromEvent (email_publisher.go) but only for
// SMS-channel delivery events. Returns (smsJob{}, false) for everything else so
// the email path stays untouched.
func smsJobFromEvent(ev eventEnvelope) (smsJob, bool) {
	data := payloadData(ev.Payload)

	job := smsJob{Locale: stringValue(ev.Payload, "locale"), Data: data}
	switch ev.Type {
	case "config.test_sms_requested":
		job.TemplateID = stringValue(ev.Payload, "template_id")
		if job.TemplateID == "" {
			job.TemplateID = "otp"
		}

		job.To = stringValue(ev.Payload, "to")
	case "auth.otp.started":
		if stringValue(ev.Payload, "channel") != "sms" {
			return smsJob{}, false
		}

		job.TemplateID = "otp"
		job.To = phoneRecipient(ev.Payload)
	case "mfa.challenge.created":
		if stringValue(ev.Payload, "channel") != "sms" {
			return smsJob{}, false
		}

		job.TemplateID = "mfa_sms"
		job.To = phoneRecipient(ev.Payload)
	case "phone.verification.requested":
		if stringValue(ev.Payload, "channel") != "sms" {
			return smsJob{}, false
		}

		job.TemplateID = "phone_verification"
		if stringValue(ev.Payload, "purpose") == "change" {
			job.TemplateID = "phone_change"
		}

		job.To = phoneRecipient(ev.Payload)
	default:
		return smsJob{}, false
	}

	job.Data["to"] = job.To
	job.Data["phone"] = job.To
	job.Data["template_id"] = job.TemplateID

	return job, true
}

// phoneRecipient picks the SMS target from to/contact/subject — unlike the email
// recipient() it does NOT require an '@'; instead it skips email-looking values.
func phoneRecipient(payload map[string]any) string {
	for _, key := range []string{"to", "contact", "subject"} {
		v := strings.TrimSpace(stringValue(payload, key))
		if v != "" && !strings.Contains(v, "@") {
			return v
		}
	}

	return ""
}

// publishSMS resolves the locale, loads the enabled SMS provider, renders the
// body and sends. Fail-soft on "not configured" (log + nil so the outbox acks),
// fail-hard on a misconfigured provider or send error (so operators see retries).
func (p *Publisher) publishSMS(ctx context.Context, ev eventEnvelope, job smsJob) error {
	if job.To == "" {
		p.log.Info("sms skipped: event has no recipient",
			xlog.String("event", ev.Type), xlog.String("project_id", ev.ProjectID))

		return nil
	}

	provider, err := p.smsProvider(ctx, ev.ProjectID)
	if err != nil {
		if errors.Is(err, errNoSMSProvider) {
			// Default state: no SMS provider configured. Preserve the historical
			// no-op behavior — ack the message, do not retry forever.
			p.log.Info("sms skipped: no enabled sms provider",
				xlog.String("event", ev.Type), xlog.String("project_id", ev.ProjectID))

			return nil
		}

		return err
	}

	job.Locale = p.resolveLocale(ctx, ev, job.Locale)

	text, err := renderText(defaultSMSText(job.TemplateID, job.Locale), job.Data)
	if err != nil {
		return err
	}

	if strings.TrimSpace(text) == "" {
		p.log.Info("sms skipped: empty body",
			xlog.String("event", ev.Type), xlog.String("template_id", job.TemplateID))

		return nil
	}

	if err := provider.send(ctx, job.To, text); err != nil {
		return err
	}
	// Do not log the phone number or body (PII / OTP code) at info level.
	p.log.Info("sms sent",
		xlog.String("event", ev.Type),
		xlog.String("project_id", ev.ProjectID),
		xlog.String("template_id", job.TemplateID),
		xlog.String("provider", provider.Type),
	)

	return nil
}

// errNoSMSProvider signals the fail-soft "no enabled sms provider" case so
// publishSMS can ack the outbox message instead of retrying forever.
var errNoSMSProvider = errors.New("notifications: no enabled sms provider")

// SMS config-validation and delivery failures. Each provider-specific
// requirement gets its own sentinel so callers could errors.Is a specific
// cause; errUnsupportedSMSProviderType is shared by the two places (decode
// time, send time) that reject an unknown provider type.
var (
	errUnsupportedSMSProviderType = errors.New("notifications: unsupported sms provider type")
	errTwilioInvalidURL           = errors.New("notifications: twilio url must be a valid https URL")
	errTwilioMissingFields        = errors.New("notifications: twilio requires account_sid, auth_token and from")
	errGenericSMSMissingURL       = errors.New("notifications: generic sms provider requires url")
	errGenericSMSInvalidURL       = errors.New("notifications: generic sms url must be a valid https URL")
	errAWSSNSMissingRegion        = errors.New("notifications: aws_sns requires region (or an explicit endpoint)")
	errAWSSNSInvalidEndpoint      = errors.New("notifications: aws_sns endpoint must be a valid https URL")
	errAWSSNSMissingCredentials   = errors.New("notifications: aws_sns requires access_key_id and secret_access_key")
	errSMSGatewayResponse         = errors.New("notifications: sms gateway returned an error")
)

type smsConfig struct {
	Type      string // "generic" | "twilio" | "aws_sns"
	URL       string // generic: webhook URL; twilio: Messages endpoint (derived)
	From      string // sender id / phone
	Username  string // twilio: account SID  | generic: basic-auth user (optional)
	Password  string // twilio: auth token   | generic: basic-auth pass (optional)
	AuthToken string // generic: bearer token (optional)
	// aws_sns (AWS SNS or any SNS-compatible endpoint, e.g. Yandex Cloud):
	Region          string // e.g. "ru-central1" / "us-east-1"
	AccessKeyID     string // service-account static access key id
	SecretAccessKey string // service-account static secret
	Endpoint        string // override host (Yandex: https://notifications.yandexcloud.net); default AWS
	HTTPClient      *http.Client
}

// smsProvider loads the enabled kind=sms provider for the project, decrypts its
// config and returns a ready smsConfig. Returns errNoSMSProvider when none is
// usable so the caller can fail-soft.
func (p *Publisher) smsProvider(ctx context.Context, projectID string) (*smsConfig, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg("sms"))),
		sm.Where(models.IamProviders.Columns.Enabled.EQ(psql.Arg(true))),
	).All(ctx, p.db.Bobx())
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		var d providerData
		if len(row.Data) > 0 {
			if err := json.Unmarshal(row.Data, &d); err != nil {
				return nil, err
			}
		}

		typ := row.Provider
		if d.Type != "" {
			typ = d.Type
		}

		typ = strings.ToLower(strings.TrimSpace(typ))
		if !domain.SMSProviderTypes.Has(typ) {
			continue
		}

		cfg, err := decodeSMSConfig(p.db.Cipher, typ, d.Config)
		if err != nil {
			return nil, err
		}

		return cfg, nil
	}

	return nil, errNoSMSProvider
}

// decodeSMSConfig reads clear keys (from, url, account_sid, username) and
// decrypts the secret keys (api_key/auth_token/token/secret/password) via the
// supplied cipher — the admin path stores those keys encrypted. Returns an error
// for a misconfigured provider (missing required fields) so callers fail-hard.
func decodeSMSConfig(cipher postgres.Cipher, typ string, raw map[string]json.RawMessage) (*smsConfig, error) {
	cfg := &smsConfig{
		Type:       typ,
		URL:        strings.TrimSpace(rawString(raw, "url")),
		From:       strings.TrimSpace(rawString(raw, "from")),
		Username:   strings.TrimSpace(rawString(raw, "username")),
		HTTPClient: &http.Client{Timeout: smsHTTPTimeout},
	}

	if err := decryptSMSSecrets(cipher, raw, cfg); err != nil {
		return nil, err
	}

	var err error

	switch typ {
	case "twilio":
		err = finishTwilioSMSConfig(raw, cfg)
	case "generic":
		err = finishGenericSMSConfig(cfg)
	case "aws_sns":
		err = finishAWSSNSConfig(cipher, raw, cfg)
	default:
		err = fmt.Errorf("%w: %q", errUnsupportedSMSProviderType, typ)
	}

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// decryptSMSSecrets decrypts every secret key the write path may have stored
// (the set MUST cover every spelling postgres.providerSecretKeys encrypts, so
// a secret stored under any of them is decrypted rather than sent verbatim).
// Every token-like secret populates both the bearer slot (AuthToken, generic)
// and the basic-auth password slot (Password, twilio) so it works regardless
// of provider type; "password" fills the password slot only.
func decryptSMSSecrets(cipher postgres.Cipher, raw map[string]json.RawMessage, cfg *smsConfig) error {
	for _, key := range []string{"password", "auth_token", "token", "api_key", "apikey", "secret", "secret_key", "client_secret", "access_token"} {
		v := rawString(raw, key)
		if v == "" {
			continue
		}

		dec, err := cipher.Decrypt(v)
		if err != nil {
			return err
		}

		switch key {
		case "password":
			if cfg.Password == "" {
				cfg.Password = dec
			}
		default: // every other recognized secret: bearer + basic-pass fallback
			if cfg.AuthToken == "" {
				cfg.AuthToken = dec
			}

			if cfg.Password == "" {
				cfg.Password = dec
			}
		}
	}

	return nil
}

// finishTwilioSMSConfig fills in the Twilio Messages API endpoint from
// account_sid when no explicit url was set, and validates the result.
func finishTwilioSMSConfig(raw map[string]json.RawMessage, cfg *smsConfig) error {
	sid := strings.TrimSpace(rawString(raw, "account_sid"))
	if sid == "" {
		sid = cfg.Username
	}

	cfg.Username = sid
	if cfg.URL == "" {
		cfg.URL = "https://api.twilio.com/2010-04-01/Accounts/" + url.PathEscape(sid) + "/Messages.json"
	}
	// Never send the basic-auth credential over cleartext http, even if the
	// operator overrode the endpoint (downgrade protection, same as generic).
	if u, err := url.Parse(cfg.URL); err != nil || u.Scheme != "https" || u.Host == "" {
		return errTwilioInvalidURL
	}

	if sid == "" || cfg.Password == "" || cfg.From == "" {
		return errTwilioMissingFields
	}

	return nil
}

// finishGenericSMSConfig validates the operator-supplied generic endpoint.
func finishGenericSMSConfig(cfg *smsConfig) error {
	if cfg.URL == "" {
		return errGenericSMSMissingURL
	}
	// Never send credentials over cleartext http (downgrade protection).
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errGenericSMSInvalidURL
	}

	return nil
}

// finishAWSSNSConfig decrypts the AWS credential pair (falling back to the
// generic secret slot decryptSMSSecrets already populated when
// secret_access_key isn't set separately), derives the regional endpoint
// when none is configured, and validates the result.
func finishAWSSNSConfig(cipher postgres.Cipher, raw map[string]json.RawMessage, cfg *smsConfig) error {
	cfg.Region = strings.TrimSpace(rawString(raw, "region"))
	cfg.Endpoint = strings.TrimSpace(rawString(raw, "endpoint"))

	if v := rawString(raw, "access_key_id"); v != "" {
		dec, err := cipher.Decrypt(v)
		if err != nil {
			return err
		}

		cfg.AccessKeyID = dec
	}

	if v := rawString(raw, "secret_access_key"); v != "" {
		dec, err := cipher.Decrypt(v)
		if err != nil {
			return err
		}

		cfg.SecretAccessKey = dec
	} else if cfg.SecretAccessKey == "" {
		cfg.SecretAccessKey = cfg.Password
	}

	if cfg.Endpoint == "" {
		if cfg.Region == "" {
			return errAWSSNSMissingRegion
		}

		cfg.Endpoint = "https://sns." + cfg.Region + ".amazonaws.com"
	}

	if u, err := url.Parse(cfg.Endpoint); err != nil || u.Scheme != "https" || u.Host == "" {
		return errAWSSNSInvalidEndpoint
	}

	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return errAWSSNSMissingCredentials
	}

	if cfg.Region == "" {
		cfg.Region = "ru-central1"
	}

	return nil
}

func (c *smsConfig) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}

	return &http.Client{Timeout: smsHTTPTimeout}
}

// send dispatches to the concrete sender for the configured provider type.
func (c *smsConfig) send(ctx context.Context, to, text string) error {
	switch c.Type {
	case "twilio":
		return c.sendTwilio(ctx, to, text)
	case "generic":
		return c.sendGeneric(ctx, to, text)
	case "aws_sns":
		return c.sendAwsSNS(ctx, to, text)
	default:
		return fmt.Errorf("%w: %q", errUnsupportedSMSProviderType, c.Type)
	}
}

// sendGeneric POSTs {to,text,from} JSON to the operator-configured webhook,
// optionally with bearer or basic auth. Non-2xx is an error.
func (c *smsConfig) sendGeneric(ctx context.Context, to, text string) error {
	body, err := json.Marshal(map[string]string{"to": to, "text": text, "from": c.From})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	} else if c.Username != "" || c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	return c.do(req)
}

// sendTwilio POSTs an x-www-form-urlencoded From/To/Body to the Twilio Messages
// endpoint using HTTP Basic auth (account SID : auth token).
func (c *smsConfig) sendTwilio(ctx context.Context, to, text string) error {
	form := url.Values{}
	form.Set("From", c.From)
	form.Set("To", to)
	form.Set("Body", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.Username, c.Password)

	return c.do(req)
}

func (c *smsConfig) do(req *http.Request) error {
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, gatewayErrSnippetMaxBytes))
		return fmt.Errorf("%w %d: %s", errSMSGatewayResponse, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	return nil
}

// defaultSMSText returns the inline built-in SMS body for a template key. Bodies
// are short and rendered with renderText so {{.code}} etc. interpolate. Russian
// copy is provided for "ru"; everything else falls back to English.
func defaultSMSText(key, locale string) string {
	ru := strings.HasPrefix(strings.ToLower(locale), "ru")

	switch key {
	case "otp", "mfa_sms":
		if ru {
			return "{{.code}} — ваш код подтверждения."
		}

		return "{{.code}} is your verification code."
	case "phone_verification":
		if ru {
			return "{{.code}} — код подтверждения номера телефона."
		}

		return "{{.code}} is your phone verification code."
	case "phone_change":
		if ru {
			return "{{.code}} — код для смены номера телефона."
		}

		return "{{.code}} is your phone change code."
	default:
		if ru {
			return "{{.code}} — ваш код."
		}

		return "{{.code}} is your code."
	}
}
