package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

const (
	hookDefaultTimeoutMs = 3000
	hookMaxTimeoutMs     = 10000
	hookMaxResponseBytes = 8 << 10
)

// hookData is the iam_hooks.data envelope. The signing secret is AES-GCM
// encrypted at rest; fail_open decides whether an unreachable/erroring hook
// blocks (false, the secure default) or is skipped (true).
type hookData struct {
	URL           string `json:"url"`
	TimeoutMs     int    `json:"timeout_ms,omitempty"`
	SigningSecret string `json:"signing_secret,omitempty"`
	FailOpen      bool   `json:"fail_open,omitempty"`
}

// pgHooks is the admin blocking-hooks adapter plus the runtime invoker. A hook
// is a signed HTTP callback the project registers at an auth decision point
// (before_sign_in / before_token_issue / before_user_create); a non-2xx reply
// denies the action unless the hook is configured fail_open.
type pgHooks struct {
	db         *DB
	emitter    Emitter
	httpClient *http.Client
}

// NewPgHooks builds the hooks adapter with an SSRF-guarded delivery client.
func NewPgHooks(db *DB, emitter Emitter) *pgHooks {
	return &pgHooks{db: db, emitter: emitter, httpClient: newWebhookHTTPClient(hookMaxTimeoutMs * time.Millisecond)}
}

func (a *pgHooks) List(ctx context.Context, projectID string) ([]domain.AdminHook, error) {
	rows, err := models.IamHooks.Query(
		sm.Where(models.IamHooks.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return nil, fmt.Errorf("list hooks: %w", err)
	}

	out := make([]domain.AdminHook, 0, len(rows))
	for _, row := range rows {
		out = append(out, hookToDomain(row))
	}

	return out, nil
}

func (a *pgHooks) Get(ctx context.Context, projectID, id string) (*domain.AdminHook, error) {
	row, err := models.FindIamHook(ctx, a.db.Bobx(), id)
	if err != nil || row.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}

	h := hookToDomain(row)

	return &h, nil
}

func (a *pgHooks) Create(ctx context.Context, projectID string, hook domain.AdminHook) (domain.AdminHook, error) {
	if err := validateWebhookURL(hook.URL); err != nil {
		return domain.AdminHook{}, err
	}

	secret, err := newWebhookSigningSecret()
	if err != nil {
		return domain.AdminHook{}, err
	}

	encSecret, err := a.db.Cipher.Encrypt(secret)
	if err != nil {
		return domain.AdminHook{}, err
	}

	id := newUUID()

	raw, err := json.Marshal(hookData{URL: hook.URL, TimeoutMs: clampHookTimeout(hook.TimeoutMs), SigningSecret: encSecret, FailOpen: hook.FailOpen})
	if err != nil {
		return domain.AdminHook{}, fmt.Errorf("hook create: marshal data: %w", err)
	}

	rm := json.RawMessage(raw)

	if _, err := models.IamHooks.Insert(&models.IamHookSetter{
		ID: &id, ProjectID: &projectID, Type: &hook.Type, Enabled: &hook.Enabled, Data: &rm,
	}).One(ctx, a.db.Bobx()); err != nil {
		return domain.AdminHook{}, fmt.Errorf("hook create: insert: %w", err)
	}

	hook.ID = id
	hook.SigningSecret = secret // returned once, on create

	return hook, nil
}

func (a *pgHooks) Update(ctx context.Context, projectID, id string, hook domain.AdminHook) (domain.AdminHook, error) {
	row, err := models.FindIamHook(ctx, a.db.Bobx(), id)
	if err != nil || row.ProjectID != projectID {
		return domain.AdminHook{}, domain.ErrNotFound
	}

	if hook.URL != "" {
		if err := validateWebhookURL(hook.URL); err != nil {
			return domain.AdminHook{}, err
		}
	}

	var cur hookData

	_ = json.Unmarshal(row.Data, &cur)

	if hook.URL != "" {
		cur.URL = hook.URL
	}

	if hook.TimeoutMs > 0 {
		cur.TimeoutMs = clampHookTimeout(hook.TimeoutMs)
	}

	cur.FailOpen = hook.FailOpen

	raw, err := json.Marshal(cur)
	if err != nil {
		return domain.AdminHook{}, fmt.Errorf("hook update: marshal data: %w", err)
	}

	rawData := json.RawMessage(raw)
	now := nowUTC()

	typ := row.Type
	if hook.Type != "" {
		typ = hook.Type
	}

	if err := row.Update(ctx, a.db.Bobx(), &models.IamHookSetter{
		Type: &typ, Enabled: &hook.Enabled, Data: &rawData, UpdatedAt: &now,
	}); err != nil {
		return domain.AdminHook{}, err
	}

	return hookToDomain(row), nil
}

func (a *pgHooks) Delete(ctx context.Context, projectID, id string) error {
	n, err := models.IamHooks.Delete(
		dm.Where(models.IamHooks.Columns.ID.EQ(psql.Arg(id))),
		dm.Where(models.IamHooks.Columns.ProjectID.EQ(psql.Arg(projectID))),
	).Exec(ctx, a.db.Bobx())
	if err != nil {
		return fmt.Errorf("hook delete: %w", err)
	}

	if n == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Test sends a payload to the hook and returns the observed status, body and
// latency without applying any auth decision.
func (a *pgHooks) Test(ctx context.Context, projectID, id string, payload []byte) (int, string, int, error) {
	row, err := models.FindIamHook(ctx, a.db.Bobx(), id)
	if err != nil || row.ProjectID != projectID {
		return 0, "", 0, domain.ErrNotFound
	}

	var cfg hookData

	_ = json.Unmarshal(row.Data, &cfg)

	if len(payload) == 0 {
		payload = []byte(`{"test":true}`)
	}

	started := nowUTC()
	status, body, _ := a.call(ctx, cfg, payload)
	dur := int(nowUTC().Sub(started).Milliseconds())

	return status, body, dur, nil
}

// InvokeHooks runs every enabled hook of hookType for the project as a blocking
// gate. It returns allowed=false when any hook denies (non-2xx) or errors and is
// not fail_open. A project with no hooks of the type is allowed. This is the
// runtime seam wired into the auth decision points.
func (a *pgHooks) InvokeHooks(ctx context.Context, projectID, hookType string, payload []byte) (bool, error) {
	rows, err := models.IamHooks.Query(
		sm.Where(models.IamHooks.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamHooks.Columns.Type.EQ(psql.Arg(hookType))),
		sm.Where(models.IamHooks.Columns.Enabled.EQ(psql.Arg(true))),
	).All(ctx, a.db.Bobx())
	if err != nil {
		return false, fmt.Errorf("invoke hooks: list: %w", err)
	}

	for _, row := range rows {
		var cfg hookData

		_ = json.Unmarshal(row.Data, &cfg)

		status, _, callErr := a.call(ctx, cfg, payload)
		ok := callErr == nil && status >= httpStatusSuccessMin && status < httpStatusSuccessMax

		if !ok && !cfg.FailOpen {
			return false, nil // fail closed: deny the action
		}
	}

	return true, nil
}

// call performs the signed HTTP POST to the hook, returning status, body and any
// transport error.
func (a *pgHooks) call(ctx context.Context, cfg hookData, payload []byte) (int, string, error) {
	timeout := time.Duration(clampHookTimeout(cfg.TimeoutMs)) * time.Millisecond

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", fmt.Errorf("hook call: build request: %w", err)
	}

	secret := cfg.SigningSecret
	if dec, decErr := a.db.Cipher.Decrypt(secret); decErr == nil {
		secret = dec
	}

	ts := nowUTC().Unix()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gopherex-iam-hooks/1")
	req.Header.Set("Webhook-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("Webhook-Signature", "v1,"+webhookSignature(secret, "", ts, payload))

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("hook call: send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, hookMaxResponseBytes))

	return resp.StatusCode, string(body), nil
}

func clampHookTimeout(timeoutMs int) int {
	if timeoutMs <= 0 {
		return hookDefaultTimeoutMs
	}

	if timeoutMs > hookMaxTimeoutMs {
		return hookMaxTimeoutMs
	}

	return timeoutMs
}

func hookToDomain(row *models.IamHook) domain.AdminHook {
	var cfg hookData

	_ = json.Unmarshal(row.Data, &cfg)

	return domain.AdminHook{
		ID: row.ID, Type: row.Type, URL: cfg.URL,
		TimeoutMs: clampHookTimeout(cfg.TimeoutMs), Enabled: row.Enabled, FailOpen: cfg.FailOpen,
	}
}
