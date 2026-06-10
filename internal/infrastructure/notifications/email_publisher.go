package notifications

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	outbox "github.com/gopherex/pg-outbox"
	"github.com/gopherex/xlog"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/infrastructure/postgres"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

const defaultLocale = "en"

type Publisher struct {
	db  *postgres.DB
	log *xlog.Logger
}

func NewPublisher(db *postgres.DB, log *xlog.Logger) *Publisher {
	return &Publisher{db: db, log: log}
}

func (p *Publisher) Publish(ctx context.Context, msgs []outbox.Message) error {
	for _, msg := range msgs {
		if err := p.publishOne(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

type eventEnvelope struct {
	Type        string         `json:"type"`
	ProjectID   string         `json:"project_id"`
	Environment string         `json:"environment"`
	AggregateID string         `json:"aggregate_id"`
	Payload     map[string]any `json:"payload"`
}

type emailJob struct {
	TemplateID string
	To         string
	Locale     string
	Data       map[string]any
}

func (p *Publisher) publishOne(ctx context.Context, msg outbox.Message) error {
	var ev eventEnvelope
	if err := json.Unmarshal(msg.Payload, &ev); err != nil {
		return err
	}
	job, ok := emailJobFromEvent(ev)
	if !ok {
		p.log.Info("would publish",
			xlog.String("id", msg.ID),
			xlog.String("topic", msg.Topic),
			xlog.String("type", msg.MessageType),
		)
		return nil
	}
	if job.To == "" {
		return fmt.Errorf("notifications: email event %s has no recipient", ev.Type)
	}
	// Flow continue email: build the cross-device deep-link from a per-tenant
	// base — the per-flow redirect_to when its origin is allowed, else the
	// project's configured app_base_url. With neither, the feature is disabled.
	if job.TemplateID == "flow_continue" {
		base := p.resolveContinueBase(ctx, ev)
		if base == "" {
			p.log.Info("flow continue email skipped: no app base URL for project", xlog.String("project_id", ev.ProjectID))
			return nil
		}
		link := flowContinueURL(base, stringValue(ev.Payload, "flow_token"))
		if link == "" {
			return nil
		}
		job.Data["continue_url"] = link
		job.Data["link"] = link
	}
	provider, err := p.smtpProvider(ctx, ev.ProjectID)
	if err != nil {
		return err
	}
	rendered, err := p.renderTemplate(ctx, ev.ProjectID, job)
	if err != nil {
		return err
	}
	if err := provider.send(job.To, rendered); err != nil {
		return err
	}
	p.log.Info("email sent",
		xlog.String("event", ev.Type),
		xlog.String("project_id", ev.ProjectID),
		xlog.String("template_id", job.TemplateID),
		xlog.String("to", job.To),
	)
	return nil
}

func emailJobFromEvent(ev eventEnvelope) (emailJob, bool) {
	data := payloadData(ev.Payload)
	locale := stringValue(ev.Payload, "locale")
	if locale == "" {
		locale = defaultLocale
	}
	job := emailJob{Locale: locale, Data: data}
	switch ev.Type {
	case "config.test_email_requested":
		job.TemplateID = stringValue(ev.Payload, "template_id")
		job.To = stringValue(ev.Payload, "to")
	case "auth.otp.started":
		if stringValue(ev.Payload, "channel") != "email" {
			return emailJob{}, false
		}
		job.TemplateID = "otp"
		job.To = recipient(ev.Payload)
	case "auth.magiclink.started":
		job.TemplateID = "magic_link"
		job.To = recipient(ev.Payload)
	case "email.verification.requested":
		job.TemplateID = "email_verification"
		if stringValue(ev.Payload, "purpose") == "change" {
			job.TemplateID = "email_change"
		}
		job.To = recipient(ev.Payload)
	case "password.reset_requested":
		job.TemplateID = "password_reset"
		job.To = recipient(ev.Payload)
	case "mfa.challenge.created":
		if stringValue(ev.Payload, "channel") != "email" {
			return emailJob{}, false
		}
		job.TemplateID = "mfa_email"
		job.To = recipient(ev.Payload)
	case "auth.flow.continue":
		// Cross-device "continue your sign-up" deep-link. continue_url is built in
		// publishOne from the configured app base URL + flow_token.
		job.TemplateID = "flow_continue"
		job.To = recipient(ev.Payload)
	default:
		return emailJob{}, false
	}
	if link := linkWithToken(stringValue(ev.Payload, "redirect_to"), stringValue(ev.Payload, "token")); link != "" {
		job.Data["link"] = link
		job.Data["magic_link"] = link
		job.Data["reset_url"] = link
		job.Data["verification_url"] = link
	}
	job.Data["to"] = job.To
	job.Data["email"] = job.To
	job.Data["template_id"] = job.TemplateID
	return job, true
}

func payloadData(payload map[string]any) map[string]any {
	out := map[string]any{}
	if raw, ok := payload["template_data"].(map[string]any); ok {
		for k, v := range raw {
			out[k] = v
		}
	}
	for k, v := range payload {
		out[k] = v
	}
	return out
}

func recipient(payload map[string]any) string {
	for _, key := range []string{"to", "email", "contact", "subject", "account_id"} {
		v := stringValue(payload, key)
		if strings.Contains(v, "@") {
			return v
		}
	}
	return ""
}

func stringValue(m map[string]any, key string) string {
	switch v := m[key].(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

// resolveContinueBase picks the per-tenant base for a flow-continue deep-link:
// the per-flow redirect_to when its origin matches the project's configured
// app_base_url, otherwise app_base_url itself. A redirect_to from a foreign
// origin is dropped (it would phish a valid flow token to an attacker host).
// Returns "" when the project has no app_base_url configured.
func (p *Publisher) resolveContinueBase(ctx context.Context, ev eventEnvelope) string {
	base := p.projectAppBaseURL(ctx, ev.ProjectID, ev.Environment)
	if redirectTo := stringValue(ev.Payload, "redirect_to"); redirectTo != "" {
		if base != "" && sameOrigin(redirectTo, base) {
			return redirectTo
		}
		p.log.Info("flow continue: redirect_to origin not allowed; using app_base_url",
			xlog.String("project_id", ev.ProjectID))
	}
	return base
}

// projectAppBaseURL reads app_base_url from the project+env "auth" config doc.
// Returns "" when unset or unreadable.
func (p *Publisher) projectAppBaseURL(ctx context.Context, projectID, env string) string {
	if env == "" {
		env = "live"
	}
	row, err := models.IamConfigs.Query(
		sm.Where(models.IamConfigs.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamConfigs.Columns.Environment.EQ(psql.Arg(env))),
		sm.Where(models.IamConfigs.Columns.Key.EQ(psql.Arg("auth"))),
	).One(ctx, p.db.Bobx())
	if err != nil || len(row.Data) == 0 {
		return ""
	}
	var doc map[string]any
	if json.Unmarshal(row.Data, &doc) != nil {
		return ""
	}
	s, _ := doc["app_base_url"].(string)
	return strings.TrimSpace(s)
}

// sameOrigin reports whether two URLs share scheme + host (incl. port).
func sameOrigin(a, b string) bool {
	ua, err1 := url.Parse(a)
	ub, err2 := url.Parse(b)
	if err1 != nil || err2 != nil || ua.Scheme == "" || ua.Host == "" || ub.Scheme == "" || ub.Host == "" {
		return false
	}
	return strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host)
}

// flowContinueURL builds the cross-device resume deep-link
// <base>/continue?flow=<flow_token>. Returns "" on a bad base or empty token.
func flowContinueURL(rawBase, flowToken string) string {
	if rawBase == "" || flowToken == "" {
		return ""
	}
	u, err := url.Parse(rawBase)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/continue"
	q := u.Query()
	q.Set("flow", flowToken)
	u.RawQuery = q.Encode()
	return u.String()
}

func linkWithToken(rawBase, token string) string {
	if rawBase == "" || token == "" {
		return ""
	}
	u, err := url.Parse(rawBase)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	if u.Host == "" {
		return ""
	}
	q := u.Query()
	if q.Get("token") == "" {
		q.Set("token", token)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

type smtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	Secure   bool
	StartTLS bool
}

type providerData struct {
	Type   string                     `json:"type"`
	Config map[string]json.RawMessage `json:"config"`
}

func (p *Publisher) smtpProvider(ctx context.Context, projectID string) (*smtpConfig, error) {
	rows, err := models.IamProviders.Query(
		sm.Where(models.IamProviders.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamProviders.Columns.Kind.EQ(psql.Arg("email"))),
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
		if !strings.EqualFold(typ, "smtp") {
			continue
		}
		cfg, err := p.decodeSMTPConfig(d.Config)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}
	return nil, errors.New("notifications: enabled smtp provider is required")
}

func (p *Publisher) decodeSMTPConfig(raw map[string]json.RawMessage) (*smtpConfig, error) {
	cfg := &smtpConfig{
		Host:     rawString(raw, "host"),
		Port:     rawInt(raw, "port", 587),
		Username: rawString(raw, "username"),
		Password: rawString(raw, "password"),
		From:     rawString(raw, "from"),
		FromName: rawString(raw, "from_name"),
		Secure:   rawBool(raw, "secure") || rawBool(raw, "ssl"),
		StartTLS: rawBool(raw, "start_tls") || rawBool(raw, "tls") || !rawBool(raw, "secure"),
	}
	if cfg.Port == 465 {
		cfg.Secure = true
	}
	if cfg.From == "" {
		cfg.From = cfg.Username
	}
	for _, key := range []string{"password", "api_key", "secret", "token", "access_token", "auth_token"} {
		if v := rawString(raw, key); v != "" {
			dec, err := p.db.Cipher.Decrypt(v)
			if err != nil {
				return nil, err
			}
			if key == "password" || cfg.Password == "" {
				cfg.Password = dec
			}
		}
	}
	if cfg.Host == "" || cfg.From == "" {
		return nil, errors.New("notifications: smtp host and from are required")
	}
	return cfg, nil
}

type renderedEmail struct {
	Subject string
	HTML    string
	Text    string
}

func (p *Publisher) renderTemplate(ctx context.Context, projectID string, job emailJob) (renderedEmail, error) {
	body, err := p.templateBody(ctx, projectID, job.TemplateID, job.Locale)
	if err != nil {
		return renderedEmail{}, err
	}
	subject, err := renderText(body["subject"], job.Data)
	if err != nil {
		return renderedEmail{}, err
	}
	html, err := renderHTML(body["html"], job.Data)
	if err != nil {
		return renderedEmail{}, err
	}
	text, err := renderText(body["text"], job.Data)
	if err != nil {
		return renderedEmail{}, err
	}
	return renderedEmail{Subject: subject, HTML: html, Text: text}, nil
}

func (p *Publisher) templateBody(ctx context.Context, projectID, key, locale string) (map[string]string, error) {
	row, err := models.IamEmailTemplates.Query(
		sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(key))),
		sm.Where(models.IamEmailTemplates.Columns.Locale.EQ(psql.Arg(locale))),
	).One(ctx, p.db.Bobx())
	if err != nil && locale != defaultLocale {
		row, err = models.IamEmailTemplates.Query(
			sm.Where(models.IamEmailTemplates.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamEmailTemplates.Columns.Key.EQ(psql.Arg(key))),
			sm.Where(models.IamEmailTemplates.Columns.Locale.EQ(psql.Arg(defaultLocale))),
		).One(ctx, p.db.Bobx())
	}
	if err != nil {
		return defaultTemplate(key), nil
	}
	out := map[string]string{}
	if len(row.Data) > 0 {
		_ = json.Unmarshal(row.Data, &out)
	}
	if out["subject"] == "" && out["html"] == "" && out["text"] == "" {
		return defaultTemplate(key), nil
	}
	return out, nil
}

func renderHTML(src string, data map[string]any) (string, error) {
	if src == "" {
		return "", nil
	}
	tpl, err := htmltemplate.New("email").Option("missingkey=zero").Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderText(src string, data map[string]any) (string, error) {
	if src == "" {
		return "", nil
	}
	tpl, err := template.New("email").Option("missingkey=zero").Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func defaultTemplate(key string) map[string]string {
	switch key {
	case "otp":
		return map[string]string{"subject": "Your sign-in code", "text": "Your code is {{.code}}."}
	case "magic_link":
		return map[string]string{"subject": "Your sign-in link", "text": "Open this link to sign in: {{.link}}", "html": `<p>Open this link to sign in: <a href="{{.link}}">{{.link}}</a></p>`}
	case "email_change":
		return map[string]string{"subject": "Confirm your new email", "text": "Use code {{.code}} or open {{.link}} to confirm your new email."}
	case "password_reset":
		return map[string]string{"subject": "Reset your password", "text": "Use code {{.code}} or open {{.link}} to reset your password."}
	case "mfa_email":
		return map[string]string{"subject": "Your MFA code", "text": "Your MFA code is {{.code}}."}
	case "flow_continue":
		return map[string]string{"subject": "Continue where you left off", "text": "Continue on this or another device: {{.continue_url}}", "html": `<p>Continue where you left off: <a href="{{.continue_url}}">{{.continue_url}}</a></p>`}
	default:
		return map[string]string{"subject": "Verify your email", "text": "Use code {{.code}} or open {{.link}} to verify your email."}
	}
}

func (c *smtpConfig) send(to string, msg renderedEmail) error {
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	from := mail.Address{Name: c.FromName, Address: c.From}
	rcpt := mail.Address{Address: to}
	headers := map[string]string{
		"From":         from.String(),
		"To":           rcpt.String(),
		"Subject":      mime.QEncoding.Encode("utf-8", msg.Subject),
		"Date":         time.Now().Format(time.RFC1123Z),
		"MIME-Version": "1.0",
	}
	body := msg.Text
	if msg.HTML != "" {
		headers["Content-Type"] = `text/html; charset="utf-8"`
		body = msg.HTML
	} else {
		headers["Content-Type"] = `text/plain; charset="utf-8"`
	}
	var raw bytes.Buffer
	for k, v := range headers {
		raw.WriteString(k)
		raw.WriteString(": ")
		raw.WriteString(v)
		raw.WriteString("\r\n")
	}
	raw.WriteString("\r\n")
	raw.WriteString(body)

	client, err := c.connect(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if c.Username != "" || c.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(c.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw.Bytes()); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (c *smtpConfig) connect(addr string) (*smtp.Client, error) {
	if c.Secure {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn, c.Host)
	}
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}
	if c.StartTLS {
		if err := client.StartTLS(&tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}); err != nil {
			_ = client.Close()
			return nil, err
		}
	}
	return client, nil
}

func rawString(raw map[string]json.RawMessage, key string) string {
	var s string
	if b, ok := raw[key]; ok {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func rawBool(raw map[string]json.RawMessage, key string) bool {
	var b bool
	if v, ok := raw[key]; ok {
		_ = json.Unmarshal(v, &b)
	}
	return b
}

func rawInt(raw map[string]json.RawMessage, key string, fallback int) int {
	if b, ok := raw[key]; ok {
		var n int
		if err := json.Unmarshal(b, &n); err == nil && n > 0 {
			return n
		}
		var s string
		if err := json.Unmarshal(b, &s); err == nil {
			if parsed, err := strconv.Atoi(s); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return fallback
}
