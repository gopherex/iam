//go:build integration

package postgres

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/go-faster/jx"

	"github.com/gopherex/iam/internal/domain"
)

// Signer must encrypt the signing-key private PEM at rest and still verify.
func TestSignerEncryptsPrivateKeyAtRest(t *testing.T) {
	ctx := context.Background()
	testDB.UseCipher(testCipher(t))
	defer testDB.UseCipher(NewIdentityCipher())

	projectID := newUUID()
	signer := testDB.Signer()
	token, err := signer.Sign(ctx, projectID, "live", map[string]any{
		"sub": "u1", "pid": projectID, "typ": "access", "env": "live",
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	var pem string
	if err := testDB.Pool.QueryRow(ctx,
		`SELECT private_pem FROM iam_signing_keys WHERE project_id=$1 LIMIT 1`, projectID,
	).Scan(&pem); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(pem, cipherPrefix) {
		t.Fatalf("private_pem not encrypted at rest: %.16s", pem)
	}
	if strings.Contains(pem, "PRIVATE KEY") {
		t.Fatal("plaintext PEM leaked at rest")
	}

	claims, err := signer.Verify(ctx, projectID, "live", token)
	if err != nil {
		t.Fatalf("verify (must decrypt under the hood): %v", err)
	}
	if claims["sub"] != "u1" {
		t.Errorf("sub = %v, want u1", claims["sub"])
	}
}

// PlatformCsrf round trip: issue, verify (reusable), reject wrong client / bad token.
func TestCsrfIssueAndVerify(t *testing.T) {
	ctx := context.Background()
	pl := NewPgPlatform(testDB)

	tok, err := pl.IssueCsrfToken(ctx, "client-1")
	if err != nil {
		t.Fatal(err)
	}
	if tok.Token == "" {
		t.Fatal("empty csrf token")
	}
	if err := pl.VerifyCsrfToken(ctx, "client-1", tok.Token); err != nil {
		t.Fatalf("verify valid: %v", err)
	}
	if err := pl.VerifyCsrfToken(ctx, "client-1", tok.Token); err != nil {
		t.Fatalf("verify reuse (synchronizer token is reusable in TTL): %v", err)
	}
	if err := pl.VerifyCsrfToken(ctx, "client-2", tok.Token); err == nil {
		t.Fatal("expected failure for wrong client")
	}
	if err := pl.VerifyCsrfToken(ctx, "client-1", "not-a-token"); err == nil {
		t.Fatal("expected failure for bad token")
	}
}

// Provider config secret keys must be encrypted at rest and decrypted on read.
func TestProviderConfigSecretsAtRest(t *testing.T) {
	ctx := context.Background()
	testDB.UseCipher(testCipher(t))
	defer testDB.UseCipher(NewIdentityCipher())

	cfg := NewPgAdminConfig(testDB, nopEmitter{})
	projectID := newUUID()
	cmd := domain.AdminProviderCmd{
		ProjectID: projectID,
		Type:      "smtp",
		Enabled:   true,
		Config: map[string]jx.Raw{
			"host":     jx.Raw(`"smtp.example.com"`),
			"password": jx.Raw(`"p@ssw0rd"`),
		},
	}
	p, err := cfg.createProvider(ctx, "email", cmd)
	if err != nil {
		t.Fatal(err)
	}

	var data []byte
	if err := testDB.Pool.QueryRow(ctx,
		`SELECT data FROM iam_providers WHERE id=$1`, p.ID,
	).Scan(&data); err != nil {
		t.Fatal(err)
	}
	// jx.Raw config values are base64-encoded inside the JSONB envelope (a
	// pre-existing quirk of marshalling []byte). The at-rest check is therefore:
	// the column must NOT contain the base64 of the plaintext value — if it does,
	// the secret was stored unencrypted.
	plainB64 := base64.StdEncoding.EncodeToString([]byte(`"p@ssw0rd"`))
	if strings.Contains(string(data), plainB64) {
		t.Fatalf("password stored unencrypted at rest: %s", data)
	}
	if strings.Contains(string(data), "p@ssw0rd") {
		t.Fatalf("plaintext password at rest: %s", data)
	}

	list, err := cfg.listProviders(ctx, projectID, "email")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 provider, got %d", len(list))
	}
	if got := string(list[0].Config["password"]); got != `"p@ssw0rd"` {
		t.Errorf("decrypted password = %s, want \"p@ssw0rd\"", got)
	}
	if got := string(list[0].Config["host"]); got != `"smtp.example.com"` {
		t.Errorf("host mangled: %s", got)
	}
}
