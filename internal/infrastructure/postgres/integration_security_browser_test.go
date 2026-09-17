//go:build integration

package postgres

import (
	"encoding/json"
	"fmt"
	"github.com/gopherex/iam/internal/domain"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Run after building web/dist. Playwright stays a development-only dependency.
func TestSecurityBrowser(t *testing.T) {
	module := os.Getenv("IAM_PLAYWRIGHT_MODULE")
	if module == "" {
		t.Skip("set IAM_PLAYWRIGHT_MODULE to the installed playwright module")
	}
	for _, mode := range []string{"single", "all", "passkey"} {
		all := mode != "single"
		t.Run(mode, func(t *testing.T) {
			ctx, store, scope, _ := securityFixture(t)
			account, err := store.account(ctx, scope)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "passkey" {
				if _, err := NewPgAccountStore(testDB, e2eEmitter).MutateDeletion(ctx, scope, domain.AccountDeletionInput{Password: "Sup3rStr0ng!Pass"}, deletionRequest); err != nil {
					t.Fatal(err)
				}
			}
			apiServer := e2eServer(t)
			upstream, err := url.Parse(apiServer.URL)
			if err != nil {
				t.Fatal(err)
			}
			proxy := httputil.NewSingleHostReverseProxy(upstream)
			root, err := filepath.Abs("../../..")
			if err != nil {
				t.Fatal(err)
			}
			files := http.FileServer(http.Dir(filepath.Join(root, "web/dist")))
			codePath := "/__test_code/" + newUUID()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == codePath:
					_, _ = w.Write([]byte(securityLastPayload(t, ctx, scope, "security_code").Code))
				case strings.HasPrefix(r.URL.Path, "/v1/"):
					proxy.ServeHTTP(w, r)
				case strings.HasPrefix(r.URL.Path, "/security/"):
					http.ServeFile(w, r, filepath.Join(root, "web/dist/index.html"))
				default:
					files.ServeHTTP(w, r)
				}
			}))
			defer server.Close()
			browserURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
			flowToken := ""
			if mode == "passkey" {
				raw, err := json.Marshal(map[string]any{"webauthnRpId": "localhost", "webauthnRpOrigins": []string{browserURL}})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := testDB.Pool.Exec(ctx, `UPDATE iam_projects SET data=data || $1::jsonb WHERE id=$2`, raw, scope.ProjectID); err != nil {
					t.Fatal(err)
				}
				if _, err := testDB.Pool.Exec(ctx, `INSERT INTO iam_config(project_id,environment,key,data) VALUES($1,'live','auth','{"methods":["passkey"]}') ON CONFLICT(project_id,environment,key) DO UPDATE SET data=EXCLUDED.data`, scope.ProjectID); err != nil {
					t.Fatal(err)
				}
				flow, err := store.Start(ctx, scope, domain.SecurityFlowInput{SessionID: scope.SessionID}, "review")
				if err != nil {
					t.Fatal(err)
				}
				flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "verify_code", Code: securityLastPayload(t, ctx, scope, "security_code").Code})
				flow = securitySubmit(t, ctx, store, scope, flow, domain.SecurityFlowInput{Action: "report_activity", AllSessions: true})
				flowToken = flow.FlowToken
			}
			cmd := exec.CommandContext(ctx, "node", filepath.Join(root, "web/test/security-browser.mjs"))
			cmd.Env = append(os.Environ(), "IAM_BROWSER_URL="+browserURL+"/security/"+scope.ProjectID+"/live", "IAM_BROWSER_FLOW_TOKEN="+flowToken, "IAM_BROWSER_PROJECT="+scope.ProjectID, "IAM_BROWSER_EMAIL="+account.PrimaryEmail, "IAM_BROWSER_CODE_URL="+server.URL+codePath, "IAM_BROWSER_ALL="+fmt.Sprint(all), "IAM_PLAYWRIGHT_MODULE="+module)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("browser: %v\n%s", err, output)
			}
			if mode == "passkey" {
				var credentials int
				if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_webauthn_credentials WHERE user_id=$1`, scope.AccountID).Scan(&credentials); err != nil {
					t.Fatal(err)
				}
				if credentials != 1 {
					t.Fatalf("passkey count %d", credentials)
				}
			}
			if mode != "passkey" {
				deletion, err := NewPgAccountStore(testDB, e2eEmitter).DeletionStatus(ctx, scope)
				if err != nil || deletion.Status != deletionCancelled {
					t.Fatalf("browser did not cancel deletion: %+v %v", deletion, err)
				}
			}
			var sessions int
			if err := testDB.Pool.QueryRow(ctx, `SELECT count(*) FROM iam_sessions WHERE project_id=$1 AND user_id=$2`, scope.ProjectID, scope.AccountID).Scan(&sessions); err != nil {
				t.Fatal(err)
			}
			want := 1
			if all {
				want = 0
			}
			if sessions != want {
				t.Fatalf("browser left %d sessions; want %d", sessions, want)
			}
		})
	}
}
