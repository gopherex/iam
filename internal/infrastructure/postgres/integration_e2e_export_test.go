//go:build integration

package postgres

// integration_e2e_export_test.go — subject data export.
//
// An access or erasure request is the reason this exists, so the test is the
// shape of that request: ask for the export, wait for the worker, and check that
// what comes back describes the person and does not hand out their credentials.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gopherex/xlog"
)

// e2eAwaitExport polls an export job until it leaves the running state.
func e2eAwaitExport(t *testing.T, ctx context.Context, url string, headers map[string]string) (string, string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		r := e2eReq(t, ctx, http.MethodGet, url, nil, headers)
		if r.Status != http.StatusOK {
			t.Fatalf("export status: %d, body %s", r.Status, r.Body)
		}

		var out struct {
			Status      string `json:"status"`
			DownloadURL string `json:"download_url"`
		}
		e2eDecode(t, r, &out)

		if out.Status != "" && out.Status != "running" && out.Status != "pending" {
			return out.Status, out.DownloadURL
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("export job did not finish: %s", url)

	return "", ""
}

// e2eDecodeExport turns a data: URL back into the exported document.
func e2eDecodeExport(t *testing.T, downloadURL string) map[string]any {
	t.Helper()

	const prefix = "data:application/json;base64,"
	if !strings.HasPrefix(downloadURL, prefix) {
		t.Fatalf("download_url is not an inline document: %q", downloadURL)
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(downloadURL, prefix))
	if err != nil {
		t.Fatalf("decode export: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse export: %v", err)
	}

	return doc
}

func TestE2EAdminUserExport(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, adminToken := e2eProjectAdmin(t, ctx)

	email := fmt.Sprintf("export-%s@example.com", newUUID()[:8])
	acct, _ := registerUser(t, ctx, projectID, email)
	userID := acct.ID

	start := e2eReq(t, ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/projects/%s/admin/users/%s/export", ts.URL, projectID, userID),
		nil, e2eBearer(adminToken))
	e2eWantStatus(t, start, http.StatusOK)

	var job struct {
		JobID string `json:"job_id"`
	}
	e2eDecode(t, start, &job)

	if job.JobID == "" {
		t.Fatalf("no job id: %s", start.Body)
	}

	// Drain the queue in-line rather than waiting on the service's worker: the
	// harness does not start one.
	drainJobs(ctx, t)

	status, url := e2eAwaitExport(t, ctx,
		fmt.Sprintf("%s/v1/projects/%s/admin/exports/%s", ts.URL, projectID, job.JobID),
		e2eBearer(adminToken))
	if status != "completed" {
		t.Fatalf("export status = %q, want completed", status)
	}

	doc := e2eDecodeExport(t, url)

	if doc["account_id"] != userID {
		t.Errorf("export is for %v, want %s", doc["account_id"], userID)
	}

	for _, key := range []string{"profile", "identities", "sessions", "grants", "activity"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("export has no %q section", key)
		}
	}

	// An access request must never become a credential leak.
	blob, _ := json.Marshal(doc)
	for _, forbidden := range []string{"password_hash", "$2a$", "refresh_token", "totp_secret"} {
		if strings.Contains(string(blob), forbidden) {
			t.Errorf("export carries %q — credential material must stay out", forbidden)
		}
	}
}

// drainJobs runs the jobs queue to empty. The harness does not start the
// service's worker, so a test that enqueues work drains it itself.
func drainJobs(ctx context.Context, t *testing.T) {
	t.Helper()

	log := xlog.NewJSON(xlog.WithLevel(xlog.ErrorLevel))
	for testDB.drainOneJob(ctx, log) { //nolint:revive,staticcheck // drain until empty
	}
}
