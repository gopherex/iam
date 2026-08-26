package postgres

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
)

const (
	webhookHTTPTimeout       = 10 * time.Second
	webhookSecretOverlap     = 24 * time.Hour
	webhookResponseBodyLimit = 16 << 10
	webhookDefaultPageLimit  = 50
	webhookMaxPageLimit      = 200
	webhookMaxRedirects      = 3
)

type webhookData struct {
	URL                      string    `json:"url"`
	Events                   []string  `json:"events"`
	Description              string    `json:"description,omitempty"`
	SigningSecret            string    `json:"signing_secret"`
	PreviousSigningSecret    string    `json:"previous_signing_secret,omitempty"`
	PreviousSecretValidUntil time.Time `json:"previous_secret_valid_until,omitempty"`
}

// PgWebhooks owns webhook configuration, the durable public-event archive and
// delivery attempts. It is shared by the admin API and the outbox publisher so
// manual retries use exactly the same signing and HTTP path as normal delivery.
type PgWebhooks struct {
	db         *DB
	httpClient *http.Client
}

func NewPgWebhooks(db *DB, client *http.Client) *PgWebhooks {
	if client == nil {
		client = newWebhookHTTPClient(webhookHTTPTimeout)
	}

	return &PgWebhooks{db: db, httpClient: client}
}

// newWebhookHTTPClient builds the delivery client with SSRF protection: a dialer
// Control hook rejects connections whose resolved IP is private, link-local,
// unique-local, CGNAT, unspecified or multicast — so a webhook URL (or a redirect
// it returns) cannot be used to reach cloud metadata (169.254.169.254), internal
// services, or the cluster network. The check runs at connect time on the
// actually-resolved IP, closing the DNS-rebinding TOCTOU that host-string
// validation alone leaves open. Loopback stays reachable as the documented
// http-development escape hatch (see validateWebhookURL). Redirects are capped.
// NewOutboundHTTPClient exposes the hardened outbound client (SSRF guards,
// redirect limits, fixed timeout) to other outbound senders in this service,
// so a second delivery path cannot quietly be less careful than webhooks.
func NewOutboundHTTPClient(timeout time.Duration) *http.Client {
	return newWebhookHTTPClient(timeout)
}

func newWebhookHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}

			if ip := net.ParseIP(host); ip != nil && isBlockedWebhookIP(ip) {
				return fmt.Errorf("webhook: refusing to connect to non-public address %s", ip)
			}

			return nil
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= webhookMaxRedirects {
				return errors.New("webhook: too many redirects")
			}

			return nil
		},
	}
}

// isBlockedWebhookIP reports whether ip is a non-public destination a webhook
// must never reach. Loopback is intentionally allowed (dev escape hatch).
func isBlockedWebhookIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return false
	}

	if ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsPrivate() {
		return true
	}

	// RFC 6598 carrier-grade NAT (100.64.0.0/10) is not covered by IsPrivate.
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
		return true
	}

	return false
}

func normalizeWebhookLimit(limit int) int {
	if limit <= 0 {
		return webhookDefaultPageLimit
	}

	if limit > webhookMaxPageLimit {
		return webhookMaxPageLimit
	}

	return limit
}

func newWebhookSigningSecret() (string, error) {
	token, err := coreAuthRandomToken()
	if err != nil {
		return "", err
	}

	return "whsec_" + base64.StdEncoding.EncodeToString([]byte(token)), nil
}

func validateWebhookURL(raw string) error {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return domain.ErrValidation.WithMessage("webhook url must be an absolute URL")
	}

	if u.User != nil || u.Fragment != "" {
		return domain.ErrValidation.WithMessage("webhook url must not contain userinfo or a fragment")
	}

	host := u.Hostname()
	if u.Scheme == "https" {
		return nil
	}

	if u.Scheme == "http" && (strings.EqualFold(host, "localhost") || isLoopbackIP(host)) {
		return nil
	}

	return domain.ErrValidation.WithMessage("webhook url must use https (http is allowed only for loopback development)")
}

func isLoopbackIP(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateWebhookEvents(events []string) error {
	if len(events) == 0 {
		return domain.ErrValidation.WithMessage("at least one webhook event is required")
	}

	seen := make(map[string]struct{}, len(events))
	for _, eventType := range events {
		if eventType != "*" && !slices.Contains(domain.SupportedWebhookEvents, eventType) {
			return domain.ErrValidation.WithDetails(map[string]any{
				"event":     eventType,
				"supported": domain.SupportedWebhookEvents,
			}).WithMessage("unsupported webhook event")
		}

		if _, ok := seen[eventType]; ok {
			return domain.ErrValidation.WithMessage("duplicate webhook event: " + eventType)
		}

		seen[eventType] = struct{}{}
	}

	return nil
}

func (a *PgWebhooks) decodeWebhookData(raw []byte) (webhookData, error) {
	var data webhookData
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, err
	}

	var err error

	data.SigningSecret, err = a.db.Cipher.Decrypt(data.SigningSecret)
	if err != nil {
		return data, err
	}

	if data.PreviousSigningSecret != "" {
		data.PreviousSigningSecret, err = a.db.Cipher.Decrypt(data.PreviousSigningSecret)
		if err != nil {
			return data, err
		}
	}

	return data, nil
}

func (a *PgWebhooks) encodeWebhookData(data webhookData) ([]byte, error) {
	var err error

	data.SigningSecret, err = a.db.Cipher.Encrypt(data.SigningSecret)
	if err != nil {
		return nil, err
	}

	if data.PreviousSigningSecret != "" {
		data.PreviousSigningSecret, err = a.db.Cipher.Encrypt(data.PreviousSigningSecret)
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(data)
}

func (a *PgWebhooks) scanWebhook(row pgx.Row) (*domain.Webhook, error) {
	var (
		out domain.Webhook
		raw []byte
	)
	if err := row.Scan(&out.ID, &out.ProjectID, &out.Environment, &out.Enabled, &out.CreatedAt, &out.UpdatedAt, &raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound.WithMessage("webhook not found")
		}

		return nil, err
	}

	data, err := a.decodeWebhookData(raw)
	if err != nil {
		return nil, err
	}

	out.URL = data.URL
	out.Events = data.Events
	out.Description = data.Description
	out.SigningSecret = data.SigningSecret
	out.PreviousSigningSecret = data.PreviousSigningSecret
	out.PreviousSecretValidUntil = data.PreviousSecretValidUntil

	return &out, nil
}

const webhookSelect = `SELECT id, project_id, environment, enabled, created_at, updated_at, data FROM iam_webhooks`

func (a *PgWebhooks) Get(ctx context.Context, projectID, environment, id string) (*domain.Webhook, error) {
	return a.scanWebhook(a.db.TxDB.QueryRow(ctx, webhookSelect+` WHERE id = $1 AND project_id = $2 AND environment = $3`, id, projectID, adminEnv(environment)))
}

func (a *PgWebhooks) List(ctx context.Context, cmd domain.WebhookListCmd) ([]domain.Webhook, string, bool, error) {
	limit := normalizeWebhookLimit(cmd.Limit)
	env := adminEnv(cmd.Environment)
	args := []any{cmd.ProjectID, env, limit + 1}

	query := webhookSelect + ` WHERE project_id = $1 AND environment = $2`
	if cmd.Cursor != "" {
		query += ` AND (created_at, id) < COALESCE((SELECT created_at, id FROM iam_webhooks WHERE id = $4 AND project_id = $1 AND environment = $2), ('infinity'::timestamptz, '~'::text))`

		args = append(args, cmd.Cursor)
	}

	query += ` ORDER BY created_at DESC, id DESC LIMIT $3`

	rows, err := a.db.TxDB.Query(ctx, query, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	out := make([]domain.Webhook, 0, limit+1)

	for rows.Next() {
		webhook, err := a.scanWebhook(rows)
		if err != nil {
			return nil, "", false, err
		}

		out = append(out, *webhook)
	}

	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}

	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}

	next := ""
	if hasMore && len(out) > 0 {
		next = out[len(out)-1].ID
	}

	return out, next, hasMore, nil
}

func (a *PgWebhooks) Create(ctx context.Context, cmd domain.WebhookCreateCmd) (*domain.Webhook, string, error) {
	if err := validateWebhookURL(cmd.URL); err != nil {
		return nil, "", err
	}

	if err := validateWebhookEvents(cmd.Events); err != nil {
		return nil, "", err
	}

	env := adminEnv(cmd.Environment)
	if cmd.IdempotencyKey != "" {
		existing, err := a.scanWebhook(a.db.TxDB.QueryRow(ctx, webhookSelect+` WHERE project_id = $1 AND environment = $2 AND idempotency_key = $3`, cmd.ProjectID, env, cmd.IdempotencyKey))
		if err == nil {
			return existing, existing.SigningSecret, nil
		}

		if !errors.Is(err, domain.ErrNotFound) {
			return nil, "", err
		}
	}

	secret, err := newWebhookSigningSecret()
	if err != nil {
		return nil, "", err
	}

	raw, err := a.encodeWebhookData(webhookData{
		URL: cmd.URL, Events: slices.Clone(cmd.Events), Description: cmd.Description, SigningSecret: secret,
	})
	if err != nil {
		return nil, "", err
	}

	id := newUUID()
	now := nowUTC()

	webhook, err := a.scanWebhook(a.db.TxDB.QueryRow(ctx, `
		INSERT INTO iam_webhooks (id, project_id, environment, idempotency_key, enabled, created_at, updated_at, data)
		VALUES ($1, $2, $3, $4, $5, $6, $6, $7)
		ON CONFLICT DO NOTHING
		RETURNING id, project_id, environment, enabled, created_at, updated_at, data`,
		id, cmd.ProjectID, env, cmd.IdempotencyKey, cmd.Enabled, now, raw))
	if errors.Is(err, domain.ErrNotFound) && cmd.IdempotencyKey != "" {
		// A concurrent request may have won the partial unique index after the
		// preflight lookup. Return that exact resource and its one-time secret.
		existing, getErr := a.scanWebhook(a.db.TxDB.QueryRow(ctx, webhookSelect+` WHERE project_id = $1 AND environment = $2 AND idempotency_key = $3`, cmd.ProjectID, env, cmd.IdempotencyKey))
		if getErr != nil {
			return nil, "", getErr
		}

		return existing, existing.SigningSecret, nil
	}

	if err != nil {
		return nil, "", translatePgErr("webhook", err)
	}

	return webhook, secret, nil
}

func (a *PgWebhooks) Update(ctx context.Context, cmd domain.WebhookUpdateCmd) (*domain.Webhook, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.Webhook, error) {
		webhook, err := a.Get(ctx, cmd.ProjectID, cmd.Environment, cmd.ID)
		if err != nil {
			return nil, err
		}

		if cmd.URL != nil {
			if err := validateWebhookURL(*cmd.URL); err != nil {
				return nil, err
			}

			webhook.URL = *cmd.URL
		}

		if cmd.Events != nil {
			if err := validateWebhookEvents(*cmd.Events); err != nil {
				return nil, err
			}

			webhook.Events = slices.Clone(*cmd.Events)
		}

		if cmd.Description != nil {
			webhook.Description = *cmd.Description
		}

		if cmd.Enabled != nil {
			webhook.Enabled = *cmd.Enabled
		}

		raw, err := a.encodeWebhookData(webhookData{
			URL: webhook.URL, Events: webhook.Events, Description: webhook.Description,
			SigningSecret: webhook.SigningSecret, PreviousSigningSecret: webhook.PreviousSigningSecret,
			PreviousSecretValidUntil: webhook.PreviousSecretValidUntil,
		})
		if err != nil {
			return nil, err
		}

		now := nowUTC()

		return a.scanWebhook(a.db.TxDB.QueryRow(ctx, `
			UPDATE iam_webhooks SET enabled = $1, updated_at = $2, data = $3
			WHERE id = $4 AND project_id = $5 AND environment = $6
			RETURNING id, project_id, environment, enabled, created_at, updated_at, data`,
			webhook.Enabled, now, raw, cmd.ID, cmd.ProjectID, adminEnv(cmd.Environment)))
	})
}

func (a *PgWebhooks) Delete(ctx context.Context, projectID, environment, id string) error {
	return a.db.withTx(ctx, func(ctx context.Context) error {
		if _, err := a.db.TxDB.Exec(ctx, `DELETE FROM iam_webhook_deliveries WHERE webhook_id = $1 AND project_id = $2 AND environment = $3`, id, projectID, adminEnv(environment)); err != nil {
			return err
		}

		result, err := a.db.TxDB.Exec(ctx, `DELETE FROM iam_webhooks WHERE id = $1 AND project_id = $2 AND environment = $3`, id, projectID, adminEnv(environment))
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return domain.ErrNotFound.WithMessage("webhook not found")
		}

		return nil
	})
}

func (a *PgWebhooks) RotateSecret(ctx context.Context, projectID, environment, id string) (string, error) {
	return withTxRet(ctx, a.db, func(ctx context.Context) (string, error) {
		webhook, err := a.Get(ctx, projectID, environment, id)
		if err != nil {
			return "", err
		}

		secret, err := newWebhookSigningSecret()
		if err != nil {
			return "", err
		}

		now := nowUTC()

		raw, err := a.encodeWebhookData(webhookData{
			URL: webhook.URL, Events: webhook.Events, Description: webhook.Description,
			SigningSecret: secret, PreviousSigningSecret: webhook.SigningSecret,
			PreviousSecretValidUntil: now.Add(webhookSecretOverlap),
		})
		if err != nil {
			return "", err
		}

		result, err := a.db.TxDB.Exec(ctx, `UPDATE iam_webhooks SET data = $1, updated_at = $2 WHERE id = $3 AND project_id = $4 AND environment = $5`, raw, now, id, projectID, adminEnv(environment))
		if err != nil {
			return "", err
		}

		if result.RowsAffected() == 0 {
			return "", domain.ErrNotFound.WithMessage("webhook not found")
		}

		return secret, nil
	})
}

func publicEventFromDomain(ev domain.Event) (domain.PublicEvent, string, bool) {
	if !slices.Contains(domain.SupportedWebhookEvents, ev.Type) {
		return domain.PublicEvent{}, "", false
	}

	if ev.Version == 0 {
		ev.Version = 1
	}

	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = nowUTC()
	}

	if ev.Environment == "" {
		ev.Environment = "live"
	}

	data := map[string]any{}
	userID := ""

	switch ev.Type {
	case domain.WebhookEventSessionRevoked:
		payload := sessionRevokedPayload(ev.Payload)
		if payload.SessionID == "" || payload.UserID == "" || payload.ProjectID != ev.ProjectID {
			return domain.PublicEvent{}, "", false
		}

		userID = payload.UserID
		data = map[string]any{
			"session_id": payload.SessionID,
			"user_id":    payload.UserID,
			"project_id": payload.ProjectID,
		}
	case domain.WebhookEventUserBanned:
		account := accountFromPayload(ev.Payload)

		userID = account.ID
		if userID == "" {
			userID = ev.AggregateID
		}

		data = map[string]any{"user_id": userID, "status": "banned"}
	case domain.WebhookEventUserDeleted:
		data = publicMap(ev.Payload)

		userID = firstString(data, "user_id", "id")
		if userID == "" {
			userID = ev.AggregateID
		}

		data = map[string]any{"user_id": userID}
	case domain.WebhookEventEmailChanged:
		account := accountFromPayload(ev.Payload)

		userID = account.ID
		if userID == "" {
			userID = ev.AggregateID
		}

		data = map[string]any{
			"user_id":        userID,
			"email":          account.PrimaryEmail,
			"email_verified": account.EmailVerified,
		}
	}

	return domain.PublicEvent{
		ID: ev.ID, Type: ev.Type, Version: ev.Version, OccurredAt: ev.OccurredAt,
		ProjectID: ev.ProjectID, Environment: ev.Environment, Data: data,
	}, userID, true
}

func sessionRevokedPayload(value any) domain.SessionRevokedPayload {
	switch payload := value.(type) {
	case domain.SessionRevokedPayload:
		return payload
	case *domain.SessionRevokedPayload:
		if payload != nil {
			return *payload
		}
	}

	data := publicMap(value)

	return domain.SessionRevokedPayload{
		SessionID: firstString(data, "session_id"),
		UserID:    firstString(data, "user_id"),
		ProjectID: firstString(data, "project_id"),
	}
}

func accountFromPayload(payload any) domain.Account {
	switch value := payload.(type) {
	case domain.Account:
		return value
	case *domain.Account:
		if value != nil {
			return *value
		}
	}

	return domain.Account{}
}

func publicMap(payload any) map[string]any {
	if value, ok := payload.(map[string]any); ok {
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[key] = item
		}

		return out
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return map[string]any{}
	}

	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		return map[string]any{}
	}

	return out
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			return value
		}
	}

	return ""
}

func (a *PgWebhooks) archiveEvent(ctx context.Context, event domain.PublicEvent, userID string) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = a.db.Pool.Exec(ctx, `
		INSERT INTO iam_events (id, project_id, environment, aggregate_id, user_id, type, published, created_at, data)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8)
		ON CONFLICT (id) DO NOTHING`, event.ID, event.ProjectID, event.Environment,
		firstString(event.Data, "session_id", "user_id"), userID, event.Type, event.OccurredAt, raw)

	return err
}

func webhookMatches(events []string, eventType string) bool {
	return slices.Contains(events, "*") || slices.Contains(events, eventType)
}

func (a *PgWebhooks) enabledForEvent(ctx context.Context, event domain.PublicEvent) ([]domain.Webhook, error) {
	rows, err := a.db.Pool.Query(ctx, webhookSelect+` WHERE project_id = $1 AND environment = $2 AND enabled = true`, event.ProjectID, event.Environment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Webhook

	for rows.Next() {
		webhook, err := a.scanWebhook(rows)
		if err != nil {
			return nil, err
		}

		if webhookMatches(webhook.Events, event.Type) {
			out = append(out, *webhook)
		}
	}

	return out, rows.Err()
}

func (a *PgWebhooks) ensureDelivery(ctx context.Context, webhook domain.Webhook, event domain.PublicEvent) (*domain.WebhookDelivery, error) {
	_, err := a.db.Pool.Exec(ctx, `
		INSERT INTO iam_webhook_deliveries (id, project_id, environment, webhook_id, event_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6, $6)
		ON CONFLICT (webhook_id, event_id) DO NOTHING`,
		newUUID(), event.ProjectID, event.Environment, webhook.ID, event.ID, nowUTC())
	if err != nil {
		return nil, err
	}

	return a.getDeliveryByPair(ctx, webhook.ID, event.ID)
}

func (a *PgWebhooks) getDeliveryByPair(ctx context.Context, webhookID, eventID string) (*domain.WebhookDelivery, error) {
	return scanDelivery(a.db.Pool.QueryRow(ctx, deliverySelect+` WHERE d.webhook_id = $1 AND d.event_id = $2`, webhookID, eventID))
}

const deliverySelect = `SELECT d.id, d.project_id, d.environment, d.webhook_id, d.event_id, e.type,
	d.status, d.attempt_count, d.next_attempt_at, d.last_attempt_at, d.delivered_at,
	d.response_status, d.response_body, d.last_error, d.created_at, d.updated_at
	FROM iam_webhook_deliveries d JOIN iam_events e ON e.id = d.event_id`

// rowScanner is the read half of a pgx row, so a scan helper can be given
// either a single row or one pulled off a Rows cursor.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanDelivery(row rowScanner) (*domain.WebhookDelivery, error) {
	var (
		out          domain.WebhookDelivery
		responseBody *string
		lastError    *string
	)
	if err := row.Scan(&out.ID, &out.ProjectID, &out.Environment, &out.WebhookID, &out.EventID, &out.EventType,
		&out.Status, &out.AttemptCount, &out.NextAttemptAt, &out.LastAttemptAt, &out.DeliveredAt,
		&out.ResponseStatus, &responseBody, &lastError, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound.WithMessage("webhook delivery not found")
		}

		return nil, err
	}

	if responseBody != nil {
		out.ResponseBody = *responseBody
	}

	if lastError != nil {
		out.LastError = *lastError
	}

	return &out, nil
}

func (a *PgWebhooks) loadDeliveryParts(ctx context.Context, deliveryID string) (*domain.WebhookDelivery, *domain.Webhook, domain.PublicEvent, error) {
	delivery, err := scanDelivery(a.db.Pool.QueryRow(ctx, deliverySelect+` WHERE d.id = $1`, deliveryID))
	if err != nil {
		return nil, nil, domain.PublicEvent{}, err
	}

	webhook, err := a.Get(ctx, delivery.ProjectID, delivery.Environment, delivery.WebhookID)
	if err != nil {
		return nil, nil, domain.PublicEvent{}, err
	}

	var (
		raw   []byte
		event domain.PublicEvent
	)

	if err := a.db.Pool.QueryRow(ctx, `SELECT data FROM iam_events WHERE id = $1`, delivery.EventID).Scan(&raw); err != nil {
		return nil, nil, domain.PublicEvent{}, err
	}

	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, nil, domain.PublicEvent{}, err
	}

	return delivery, webhook, event, nil
}

func webhookSignature(secret, eventID string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%s.%d.", eventID, timestamp)
	_, _ = mac.Write(body)

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (a *PgWebhooks) deliver(ctx context.Context, deliveryID string, force bool) (*domain.WebhookDelivery, error) {
	delivery, webhook, event, err := a.loadDeliveryParts(ctx, deliveryID)
	if err != nil {
		return nil, err
	}

	if delivery.Status == "succeeded" && !force {
		return delivery, nil
	}

	req, err := buildWebhookRequest(ctx, webhook, event)
	if err != nil {
		return nil, err
	}

	attemptedAt := nowUTC()
	status, responseBody, requestErr := a.sendWebhookRequest(req)
	attempts := delivery.AttemptCount + 1

	if requestErr == nil {
		return a.recordWebhookSuccess(ctx, delivery.ID, attempts, attemptedAt, status, responseBody)
	}

	return a.recordWebhookFailure(ctx, delivery.ID, attempts, attemptedAt, status, responseBody, requestErr)
}

// buildWebhookRequest signs event's JSON body (current secret, plus the
// previous one while it is still within its grace window — so a secret
// rotation doesn't 401 in-flight deliveries) and assembles the signed POST.
func buildWebhookRequest(ctx context.Context, webhook *domain.Webhook, event domain.PublicEvent) (*http.Request, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	timestamp := nowUTC().Unix()

	signatures := []string{"v1," + webhookSignature(webhook.SigningSecret, event.ID, timestamp, body)}
	if webhook.PreviousSigningSecret != "" && nowIn(ctx).Before(webhook.PreviousSecretValidUntil) {
		signatures = append(signatures, "v1,"+webhookSignature(webhook.PreviousSigningSecret, event.ID, timestamp, body))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gopherex-iam-webhooks/1")
	req.Header.Set("Webhook-Id", event.ID)
	req.Header.Set("Webhook-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Webhook-Signature", strings.Join(signatures, " "))

	return req, nil
}

// sendWebhookRequest performs req and reports the outcome: status/responseBody
// are populated whenever a response came back at all (even a non-2xx one, so
// the caller can persist it for debugging), and requestErr is set for a
// transport failure, a body-read failure, or a non-2xx status.
func (a *PgWebhooks) sendWebhookRequest(req *http.Request) (*int, string, error) {
	response, requestErr := a.httpClient.Do(req)
	if response == nil {
		return nil, "", requestErr
	}

	status := response.StatusCode

	limited, readErr := io.ReadAll(io.LimitReader(response.Body, webhookResponseBodyLimit+1))
	_ = response.Body.Close()

	if len(limited) > webhookResponseBodyLimit {
		limited = limited[:webhookResponseBodyLimit]
	}

	responseBody := string(limited)

	if requestErr == nil && readErr != nil {
		requestErr = readErr
	}

	if requestErr == nil && (response.StatusCode < 200 || response.StatusCode >= 300) {
		requestErr = fmt.Errorf("webhook returned HTTP %d", response.StatusCode)
	}

	return &status, responseBody, requestErr
}

// recordWebhookSuccess marks a delivery succeeded and returns its fresh row.
func (a *PgWebhooks) recordWebhookSuccess(ctx context.Context, deliveryID string, attempts int, attemptedAt time.Time, status *int, responseBody string) (*domain.WebhookDelivery, error) {
	if _, err := a.db.Pool.Exec(ctx, `UPDATE iam_webhook_deliveries
		SET status = 'succeeded', attempt_count = $1, next_attempt_at = NULL, last_attempt_at = $2,
			delivered_at = $2, response_status = $3, response_body = $4, last_error = NULL, updated_at = $2
		WHERE id = $5`, attempts, attemptedAt, status, responseBody, deliveryID); err != nil {
		return nil, err
	}

	return scanDelivery(a.db.Pool.QueryRow(ctx, deliverySelect+` WHERE d.id = $1`, deliveryID))
}

// recordWebhookFailure schedules the next retry with capped exponential
// backoff (1s, 2s, 4s, ... capped at 5m) and returns the fresh row alongside
// requestErr (the caller reports it so pg-outbox retries the delivery job).
func (a *PgWebhooks) recordWebhookFailure(ctx context.Context, deliveryID string, attempts int, attemptedAt time.Time, status *int, responseBody string, requestErr error) (*domain.WebhookDelivery, error) {
	delay := time.Second << min(attempts-1, 8)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}

	message := requestErr.Error()

	if _, err := a.db.Pool.Exec(ctx, `UPDATE iam_webhook_deliveries
		SET status = 'failed', attempt_count = $1, next_attempt_at = $2, last_attempt_at = $3,
			response_status = $4, response_body = $5, last_error = $6, updated_at = $3
		WHERE id = $7`, attempts, attemptedAt.Add(delay), attemptedAt, status, responseBody, message, deliveryID); err != nil {
		return nil, errors.Join(requestErr, err)
	}

	updated, err := scanDelivery(a.db.Pool.QueryRow(ctx, deliverySelect+` WHERE d.id = $1`, deliveryID))
	if err != nil {
		return nil, errors.Join(requestErr, err)
	}

	return updated, requestErr
}

// PublishEvent archives and delivers one safe public event. Returning an error
// asks pg-outbox to retry with its exponential backoff. Succeeded deliveries
// are skipped on the retry, avoiding duplicate sends to healthy endpoints.
func (a *PgWebhooks) PublishEvent(ctx context.Context, ev domain.Event) error {
	event, userID, ok := publicEventFromDomain(ev)
	if !ok {
		return nil
	}

	if err := a.archiveEvent(ctx, event, userID); err != nil {
		return err
	}

	webhooks, err := a.enabledForEvent(ctx, event)
	if err != nil {
		return err
	}

	var deliveryErrs []error

	for _, webhook := range webhooks {
		delivery, err := a.ensureDelivery(ctx, webhook, event)
		if err == nil {
			_, err = a.deliver(ctx, delivery.ID, false)
		}

		if err != nil {
			deliveryErrs = append(deliveryErrs, fmt.Errorf("webhook %s: %w", webhook.ID, err))
		}
	}

	if len(deliveryErrs) > 0 {
		return errors.Join(deliveryErrs...)
	}

	_, err = a.db.Pool.Exec(ctx, `UPDATE iam_events SET published = true WHERE id = $1`, event.ID)

	return err
}

func (a *PgWebhooks) ListDeliveries(ctx context.Context, cmd domain.WebhookDeliveryListCmd) ([]domain.WebhookDelivery, error) {
	limit := normalizeWebhookLimit(cmd.Limit)
	args := []any{cmd.ProjectID, adminEnv(cmd.Environment)}
	query := deliverySelect + ` WHERE d.project_id = $1 AND d.environment = $2`

	if cmd.WebhookID != "" {
		args = append(args, cmd.WebhookID)
		query += fmt.Sprintf(" AND d.webhook_id = $%d", len(args))
	}

	if cmd.Status != "" {
		if cmd.Status != "pending" && cmd.Status != "succeeded" && cmd.Status != "failed" {
			return nil, domain.ErrValidation.WithMessage("invalid webhook delivery status")
		}

		args = append(args, cmd.Status)
		query += fmt.Sprintf(" AND d.status = $%d", len(args))
	}

	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY d.created_at DESC, d.id DESC LIMIT $%d", len(args))

	rows, err := a.db.TxDB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.WebhookDelivery

	for rows.Next() {
		delivery, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}

		out = append(out, *delivery)
	}

	return out, rows.Err()
}

func (a *PgWebhooks) RetryDelivery(ctx context.Context, projectID, environment, deliveryID string) (*domain.WebhookDelivery, error) {
	delivery, err := scanDelivery(a.db.TxDB.QueryRow(ctx, deliverySelect+` WHERE d.id = $1 AND d.project_id = $2 AND d.environment = $3`, deliveryID, projectID, adminEnv(environment)))
	if err != nil {
		return nil, err
	}

	_, err = a.db.TxDB.Exec(ctx, `UPDATE iam_webhook_deliveries SET status = 'pending', next_attempt_at = NULL, updated_at = $1 WHERE id = $2`, nowUTC(), delivery.ID)
	if err != nil {
		return nil, err
	}

	updated, err := a.deliver(ctx, delivery.ID, true)
	if updated != nil {
		// Manual retries are diagnostic operations: the persisted failed status
		// is the useful result even when the receiver still returns an error.
		return updated, nil
	}

	return nil, err
}

func (a *PgWebhooks) Test(ctx context.Context, projectID, environment, webhookID, eventType string) (*domain.WebhookDelivery, error) {
	if eventType == "" {
		eventType = domain.WebhookEventSessionRevoked
	}

	if !slices.Contains(domain.SupportedWebhookEvents, eventType) {
		return nil, domain.ErrValidation.WithMessage("unsupported webhook event")
	}

	webhook, err := a.Get(ctx, projectID, environment, webhookID)
	if err != nil {
		return nil, err
	}

	event := domain.PublicEvent{
		ID: newUUID(), Type: eventType, Version: 1, OccurredAt: nowUTC(),
		ProjectID: projectID, Environment: adminEnv(environment),
		Data: map[string]any{"test": true},
	}
	if err := a.archiveEvent(ctx, event, ""); err != nil {
		return nil, err
	}

	delivery, err := a.ensureDelivery(ctx, *webhook, event)
	if err != nil {
		return nil, err
	}

	updated, err := a.deliver(ctx, delivery.ID, true)
	if updated != nil {
		return updated, nil
	}

	return nil, err
}

func (a *PgWebhooks) ListEvents(ctx context.Context, cmd domain.WebhookEventListCmd) (*domain.WebhookEventPage, error) {
	limit := normalizeWebhookLimit(cmd.Limit)
	args := []any{cmd.ProjectID, adminEnv(cmd.Environment)}
	query := `SELECT data FROM iam_events WHERE project_id = $1 AND environment = $2`

	if cmd.Type != "" {
		args = append(args, cmd.Type)
		query += fmt.Sprintf(" AND type = $%d", len(args))
	}

	if cmd.UserID != "" {
		args = append(args, cmd.UserID)
		query += fmt.Sprintf(" AND user_id = $%d", len(args))
	}

	if cmd.Cursor != "" {
		args = append(args, cmd.Cursor)
		query += fmt.Sprintf(" AND (created_at, id) < COALESCE((SELECT created_at, id FROM iam_events WHERE id = $%d), ('infinity'::timestamptz, '~'::text))", len(args))
	}

	args = append(args, limit+1)
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := a.db.TxDB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	page := &domain.WebhookEventPage{Data: make([]domain.PublicEvent, 0, limit+1)}

	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}

		var event domain.PublicEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, err
		}

		page.Data = append(page.Data, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	page.HasMore = len(page.Data) > limit
	if page.HasMore {
		page.Data = page.Data[:limit]
		page.NextCursor = page.Data[len(page.Data)-1].ID
	}

	return page, nil
}

func (a *PgWebhooks) ReplayEvent(ctx context.Context, projectID, environment, eventID, webhookID string) ([]domain.WebhookDelivery, error) {
	var raw []byte
	if err := a.db.TxDB.QueryRow(ctx, `SELECT data FROM iam_events WHERE id = $1 AND project_id = $2 AND environment = $3`, eventID, projectID, adminEnv(environment)).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound.WithMessage("event not found")
		}

		return nil, err
	}

	var event domain.PublicEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, err
	}

	var webhooks []domain.Webhook

	if webhookID != "" {
		webhook, err := a.Get(ctx, projectID, environment, webhookID)
		if err != nil {
			return nil, err
		}

		webhooks = []domain.Webhook{*webhook}
	} else {
		var err error

		webhooks, err = a.enabledForEvent(ctx, event)
		if err != nil {
			return nil, err
		}
	}

	var out []domain.WebhookDelivery

	for _, webhook := range webhooks {
		delivery, err := a.ensureDelivery(ctx, webhook, event)
		if err == nil {
			delivery, err = a.deliver(ctx, delivery.ID, true)
		}

		if err != nil {
			return out, err
		}

		out = append(out, *delivery)
	}

	return out, nil
}
