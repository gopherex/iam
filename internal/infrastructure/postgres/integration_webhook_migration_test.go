//go:build integration

package postgres

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/infrastructure/postgres/migrations"
)

// TestWebhookMigrationPreservesPopulatedIAMUsers applies the webhook migration
// over the previous release's bootstrap schema with an existing user row. This
// guards the production upgrade contract: webhook work must remain additive and
// must not rebuild or rewrite identity data.
func TestWebhookMigrationPreservesPopulatedIAMUsers(t *testing.T) {
	ctx := context.Background()
	database := "upgrade_" + strings.ReplaceAll(newUUID(), "-", "")[:12]
	if _, err := testDB.Pool.Exec(ctx, `CREATE DATABASE `+fmt.Sprintf("%q", database)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = testDB.Pool.Exec(context.Background(), `DROP DATABASE IF EXISTS `+fmt.Sprintf("%q", database)+` WITH (FORCE)`)
	})

	dsn, err := url.Parse(testDSN)
	if err != nil {
		t.Fatal(err)
	}
	dsn.Path = "/" + database
	db, err := Connect(ctx, dsn.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	bootstrap, err := migrations.FS.ReadFile("20260610202345893_bootstrap.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, string(bootstrap)); err != nil {
		t.Fatal(err)
	}
	original := `{"id":"user-existing","marker":"must-survive"}`
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO iam_users (id, project_id, environment, primary_email, data)
		VALUES ('user-existing', 'project-existing', 'live', 'existing@example.com', $1::jsonb)`, original); err != nil {
		t.Fatal(err)
	}

	upgrade, err := migrations.FS.ReadFile("20260712135920777_webhook_delivery.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := strings.SplitN(string(upgrade), "-- sqld:down", 2)[0]
	up = strings.TrimPrefix(up, "-- sqld:up")
	if _, err := db.Pool.Exec(ctx, up); err != nil {
		t.Fatal(err)
	}

	var email, data string
	if err := db.Pool.QueryRow(ctx, `SELECT primary_email, data::text FROM iam_users WHERE id = 'user-existing'`).Scan(&email, &data); err != nil {
		t.Fatal(err)
	}
	if email != "existing@example.com" || !strings.Contains(data, "must-survive") {
		t.Fatalf("existing user changed: email=%q data=%s", email, data)
	}
	var deliveriesTable string
	if err := db.Pool.QueryRow(ctx, `SELECT to_regclass('public.iam_webhook_deliveries')::text`).Scan(&deliveriesTable); err != nil {
		t.Fatal(err)
	}
	if deliveriesTable != "iam_webhook_deliveries" {
		t.Fatalf("delivery table missing: %q", deliveriesTable)
	}
}
