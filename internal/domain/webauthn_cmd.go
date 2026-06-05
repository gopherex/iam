package domain

// WebAuthnRenameCredentialCmd renames a stored passkey/WebAuthn credential. The
// AccountID scopes the operation to the caller, preventing cross-account access.
type WebAuthnRenameCredentialCmd struct {
	AccountID    string
	CredentialID string
	Name         string
}
