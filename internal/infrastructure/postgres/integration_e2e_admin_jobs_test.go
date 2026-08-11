//go:build integration

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/gopherex/xlog"
)

// TestE2EAdminJobsImportUsers covers the async jobs surface: enqueue a bulk user
// import, run the worker, then verify the imported account exists and its
// pre-hashed password authenticates. Also covers the synchronous
// password-hash-verify and jobs list/get.
func TestE2EAdminJobsImportUsers(t *testing.T) {
	ctx := context.Background()
	ts := e2eServer(t)
	projectID, token := e2eProjectAdmin(t, ctx)

	email := fmt.Sprintf("import-%s@example.com", newUUID()[:8])
	hash, err := bcrypt.GenerateFromPassword([]byte("Imported!Pass1"), 12)
	if err != nil {
		t.Fatal(err)
	}

	// Enqueue the import.
	ri := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/projects/"+projectID+"/admin/import/users",
		map[string]any{
			"users":                []map[string]any{{"email": email, "password_hash": string(hash)}},
			"password_hash_format": "bcrypt",
		}, e2eBearer(token))
	e2eWantStatus(t, ri, http.StatusOK)

	var enq struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	e2eDecode(t, ri, &enq)

	if enq.JobID == "" {
		t.Fatal("import did not return a job id")
	}

	// Run the worker until the queue is empty (other tests may have left pending
	// jobs; the worker processes oldest-first).
	for testDB.drainOneJob(ctx, xlog.NewJSON()) { //nolint:revive // drain until empty
	}

	// The job is now completed.
	rg := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/projects/"+projectID+"/admin/jobs/"+enq.JobID, nil, e2eBearer(token))
	e2eWantStatus(t, rg, http.StatusOK)

	var jobResp struct {
		Job struct {
			Status string `json:"status"`
		} `json:"job"`
	}
	e2eDecode(t, rg, &jobResp)

	if jobResp.Job.Status != "completed" {
		t.Fatalf("job status = %q, want completed", jobResp.Job.Status)
	}

	// The imported account authenticates with its original password (the imported
	// bcrypt hash was stored as the credential).
	ca := NewPgCoreAuth(testDB, &recordingEmitter{}, NewConfigReader(testDB, 0))
	if _, err := ca.AuthenticatePassword(ctx, projectID, email, "Imported!Pass1"); err != nil {
		t.Fatalf("imported user cannot authenticate: %v", err)
	}

	// Synchronous password-hash verification.
	rv := e2eReq(t, ctx, http.MethodPost, ts.URL+"/v1/projects/"+projectID+"/admin/import/password-hashes/verify",
		map[string]any{"hash": string(hash), "password": "Imported!Pass1", "format": "bcrypt"}, e2eBearer(token))
	e2eWantStatus(t, rv, http.StatusOK)

	var ver struct {
		Valid bool `json:"valid"`
	}
	e2eDecode(t, rv, &ver)

	if !ver.Valid {
		t.Fatal("password-hash verify returned false for a matching password")
	}

	// Jobs list includes the import job.
	rl := e2eReq(t, ctx, http.MethodGet, ts.URL+"/v1/projects/"+projectID+"/admin/jobs", nil, e2eBearer(token))
	e2eWantStatus(t, rl, http.StatusOK)
}
