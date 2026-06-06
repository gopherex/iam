//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// AdminAPIKeys CRUD + tenancy boundary against a real database.
func TestAdminAPIKeyCRUDAndTenancy(t *testing.T) {
	ctx := context.Background()
	keys := NewPgAdminAPIKeys(testDB, nopEmitter{})
	projectA := newUUID()
	projectB := newUUID()

	// Create.
	created, err := keys.Create(ctx, domain.AdminAPIKeyCmd{
		ProjectID: projectA,
		Name:      "ci",
		Scopes:    []string{"read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Secret == "" || created.Key == nil {
		t.Fatal("create returned no secret/key")
	}
	keyID := created.Key.ID

	// List is project-scoped.
	listA, err := keys.List(ctx, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(listA) != 1 || listA[0].ID != keyID {
		t.Fatalf("project A list = %+v", listA)
	}
	listB, err := keys.List(ctx, projectB)
	if err != nil {
		t.Fatal(err)
	}
	if len(listB) != 0 {
		t.Fatalf("project B must not see project A keys: %+v", listB)
	}

	// Rotate returns a fresh secret.
	rotated, err := keys.Rotate(ctx, projectA, keyID)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.Secret == created.Secret {
		t.Fatal("rotate did not change the secret")
	}

	// Update.
	updated, err := keys.Update(ctx, domain.AdminAPIKeyUpdateCmd{
		ProjectID: projectA,
		KeyID:     keyID,
		Name:      "ci-renamed",
		Disabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "ci-renamed" || !updated.Disabled {
		t.Fatalf("update not applied: %+v", updated)
	}

	// Tenancy: project B cannot delete project A's key.
	if err := keys.Delete(ctx, projectB, keyID); err == nil {
		t.Fatal("cross-tenant delete must fail")
	}

	// Owner can delete.
	if err := keys.Delete(ctx, projectA, keyID); err != nil {
		t.Fatal(err)
	}
	after, err := keys.List(ctx, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Fatalf("key not deleted: %+v", after)
	}
}
