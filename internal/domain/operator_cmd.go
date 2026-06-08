package domain

import "time"

// OperatorProjectPatchCmd is the operator-scoped partial update of a project.
// Only the named scalar fields are mutable through /mgmt; empty strings are
// treated as "no change" by the adapter.
type OperatorProjectPatchCmd struct {
	ProjectID     string
	Name          string
	Slug          string
	DefaultLocale string
}

// OperatorAdminToken is a read model describing an admin token minted for a
// project, returned by the /mgmt admin-tokens listing.
type OperatorAdminToken struct {
	ID        string
	ProjectID string
	Name      string
	Scopes    []string
	CreatedAt time.Time
	ExpiresAt time.Time
	Revoked   bool
}

// OperatorAdminTokenCmd carries the operator request to mint a project-admin
// token. Empty ExpiresAt means the adapter applies its default TTL.
type OperatorAdminTokenCmd struct {
	ProjectID string
	Name      string
	Scopes    []string
	ExpiresAt time.Time
}

// OperatorConfigCmd carries a free-form declarative project configuration
// document for the config plan/apply operations. Config is intentionally
// schemaless at this layer; the adapter validates it.
type OperatorConfigCmd struct {
	ProjectID string
	Config    map[string]any
}

// OperatorFeaturesCmd is the operator-scoped feature-flag patch for a project.
type OperatorFeaturesCmd struct {
	ProjectID string
	Features  map[string]bool
}
