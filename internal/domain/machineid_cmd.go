package domain

import "time"

// Machine-identity command/value types for the service-account and API-key
// slices of the machine identity aggregate. Names are prefixed with MachineID*
// to avoid collisions with command types owned by other service slices.

// MachineIDServiceAccountListCmd queries service accounts in a project.
type MachineIDServiceAccountListCmd struct {
	ProjectID string
	Cursor    string
	Limit     int
}

// MachineIDServiceAccountPage is a cursor-paginated slice of service accounts.
type MachineIDServiceAccountPage struct {
	Items      []*ServiceAccount
	NextCursor string
	HasMore    bool
}

// MachineIDServiceAccountPatchCmd updates a service account's scopes/disabled state.
type MachineIDServiceAccountPatchCmd struct {
	ProjectID        string
	ServiceAccountID string
	Scopes           []string
	Disabled         *bool
}

// MachineIDSecretCmd mints a new client secret for a service account.
type MachineIDSecretCmd struct {
	ProjectID        string
	ServiceAccountID string
	Name             string
	ExpiresAt        *time.Time
}

// MachineIDSecret is the result of creating a service-account client secret.
// ClientSecret is the plaintext value, returned exactly once at creation.
type MachineIDSecret struct {
	SecretID     string
	ClientID     string
	ClientSecret string
}

// MachineIDAPIKeyPatchCmd updates API-key metadata/scopes.
type MachineIDAPIKeyPatchCmd struct {
	ProjectID string
	KeyID     string
	Name      string
	Scopes    []string
	Disabled  *bool
}
