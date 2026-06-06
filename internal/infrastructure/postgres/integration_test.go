//go:build integration

// Integration tests run against a real Postgres started via testcontainers.
// Build/run with: go test -tags=integration ./internal/infrastructure/postgres/...
// (or `make test-db`). They are excluded from the default `go test ./...`.
package postgres

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/gopherex/iam/internal/domain"
)

// testDB is the shared connection to the throwaway Postgres container. Tests
// isolate by using fresh UUID project ids, so a single migrated database is
// reused across the package.
var testDB *DB

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:17",
		tcpostgres.WithDatabase("iam"),
		tcpostgres.WithUsername("iam"),
		tcpostgres.WithPassword("iam"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(90*time.Second)),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("connection string: " + err.Error())
	}
	db, err := Connect(ctx, dsn)
	if err != nil {
		panic("connect: " + err.Error())
	}
	if err := db.Migrate(ctx); err != nil {
		panic("migrate: " + err.Error())
	}
	testDB = db

	code := m.Run()

	db.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

// nopEmitter is a no-op Emitter for adapter tests; outbox emission is exercised
// separately (it is not the subject of these tests).
type nopEmitter struct{}

func (nopEmitter) Emit(_ context.Context, _ domain.Event) error { return nil }

// testCipher returns an AES-256-GCM cipher with an all-zero key for round-trip
// assertions.
func testCipher(t *testing.T) Cipher {
	t.Helper()
	c, err := NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
