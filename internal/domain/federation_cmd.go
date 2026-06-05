package domain

import "time"

// FederationConnectionUpdateCmd is a partial update of an SSO connection. The
// Patch map carries only the fields the caller supplied (decoded from the
// free-form JSON merge-patch body); the adapter applies them atomically.
type FederationConnectionUpdateCmd struct {
	ProjectID string
	ID        string
	Patch     map[string]any
}

// FederationScimTokenCmd mints a new SCIM provisioning bearer token scoped to a
// single SSO connection.
type FederationScimTokenCmd struct {
	ProjectID    string
	ConnectionID string
	Name         string
	ExpiresAt    time.Time
}

// ScimToken is a provisioning credential bound to an SSO connection. The secret
// is returned only at creation time and is never persisted in clear.
type ScimToken struct {
	ID           string
	ProjectID    string
	ConnectionID string
	Name         string
	ExpiresAt    time.Time
}
