package postgres

import (
	"bytes"
	"context"
	"encoding/json"
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
		return nil, err
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

func (a *pgHooks) Create(ctx context.Context, projectID string, h domain.AdminHook) (domain.AdminHook, error) {
	if err := validateWebhookURL(h.URL); err != nil {
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

	raw, err := json.Marshal(hookData{URL: h.URL, TimeoutMs: clampHookTimeout(h.TimeoutMs), SigningSecret: encSecret, FailOpen: h.FailOpen})
	if err != nil {
		return domain.AdminHook{}, err
	}

	rm := json.RawMessage(raw)

	if _, err := models.IamHooks.Insert(&models.IamHookSetter{
		ID: &id, ProjectID: &projectID, Type: &h.Type, Enabled: &h.Enabled, Data: &rm,
	}).One(ctx, a.db.Bobx()); err != nil {
		return domain.AdminHook{}, err
	}

	h.ID = id
	h.SigningSecret = secret // returned once, on create

	return h, nil
}

func (a *pgHooks) Update(ctx context.Context, projectID, id string, h domain.AdminHook) (domain.AdminHook, error) {
	row, err := models.FindIamHook(ctx, a.db.Bobx(), id)
	if err != nil || row.ProjectID != projectID {
		return domain.AdminHook{}, domain.ErrNotFound
	}

	if h.URL != "" {
		if err := validateWebhookURL(h.URL); err != nil {
			return domain.AdminHook{}, err
		}
	}

	var cur hookData
	_ = json.Unmarshal(row.Data, &cur)

	if h.URL != "" {
		cur.URL = h.URL
	}

	if h.TimeoutMs > 0 {
		cur.TimeoutMs = clampHookTimeout(h.TimeoutMs)
	}

	cur.FailOpen = h.FailOpen

	raw, err := json.Marshal(cur)
	if err != nil {
		return domain.AdminHook{}, err
	}

	rm := json.RawMessage(raw)
	now := nowUTC()
	typ := row.Type
	if h.Type != "" {
		typ = h.Type
	}

	if err := row.Update(ctx, a.db.Bobx(), &models.IamHookSetter{
		Type: &typ, Enabled: &h.Enabled, Data: &rm, UpdatedAt: &now,
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
		return err
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

	var d hookData
	_ = json.Unmarshal(row.Data, &d)

	if len(payload) == 0 {
		payload = []byte(`{"test":true}`)
	}

	started := nowUTC()
	status, body, _ := a.call(ctx, d, payload)
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
		return false, err
	}

	for _, row := range rows {
		var d hookData
		_ = json.Unmarshal(row.Data, &d)

		status, _, callErr := a.call(ctx, d, payload)
		ok := callErr == nil && status >= 200 && status < 300

		if !ok && !d.FailOpen {
			return false, nil // fail closed: deny the action
		}
	}

	return true, nil
}

// call performs the signed HTTP POST to the hook, returning status, body and any
// transport error.
func (a *pgHooks) call(ctx context.Context, d hookData, payload []byte) (int, string, error) {
	timeout := time.Duration(clampHookTimeout(d.TimeoutMs)) * time.Millisecond
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, d.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}

	secret := d.SigningSecret
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
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, hookMaxResponseBytes))

	return resp.StatusCode, string(body), nil
}

func clampHookTimeout(ms int) int {
	if ms <= 0 {
		return hookDefaultTimeoutMs
	}

	if ms > hookMaxTimeoutMs {
		return hookMaxTimeoutMs
	}

	return ms
}

func hookToDomain(row *models.IamHook) domain.AdminHook {
	var d hookData
	_ = json.Unmarshal(row.Data, &d)

	return domain.AdminHook{
		ID: row.ID, Type: row.Type, URL: d.URL,
		TimeoutMs: clampHookTimeout(d.TimeoutMs), Enabled: row.Enabled, FailOpen: d.FailOpen,
	}
}
