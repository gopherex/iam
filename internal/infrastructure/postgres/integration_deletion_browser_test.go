//go:build integration

package postgres

import (
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

func TestAccountDeletionBrowser(t *testing.T) {
	module := os.Getenv("IAM_PLAYWRIGHT_MODULE")
	if module == "" {
		t.Skip("set IAM_PLAYWRIGHT_MODULE after building web/dist")
	}
	ctx, store, scope, _ := deletionFixture(t)
	account, err := store.Get(ctx, scope.ProjectID, scope.AccountID)
	if err != nil {
		t.Fatal(err)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/"), strings.HasPrefix(r.URL.Path, "/mgmt/"):
			proxy.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/security/"), strings.HasPrefix(r.URL.Path, "/projects/"):
			http.ServeFile(w, r, filepath.Join(root, "web/dist/index.html"))
		default:
			files.ServeHTTP(w, r)
		}
	}))
	defer server.Close()
	cmd := exec.CommandContext(ctx, "node", filepath.Join(root, "web/test/account-deletion-browser.mjs"))
	cmd.Env = append(os.Environ(), "IAM_BROWSER_URL="+server.URL, "IAM_BROWSER_PROJECT="+scope.ProjectID,
		"IAM_BROWSER_EMAIL="+account.PrimaryEmail, "IAM_BROWSER_MASTER_KEY="+e2eMasterKey, "IAM_PLAYWRIGHT_MODULE="+module)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser: %v\n%s", err, output)
	}
	state, err := store.DeletionStatus(ctx, scope)
	if err != nil || state.Status != deletionCancelled {
		t.Fatalf("state: %+v %v", state, err)
	}
}
