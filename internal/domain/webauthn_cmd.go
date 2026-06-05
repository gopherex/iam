package domain

// WebAuthnRenameCredentialCmd renames a stored passkey/WebAuthn credential. The
// AccountID scopes the operation to the caller, preventing cross-account access.
type WebAuthnRenameCredentialCmd struct {
	AccountID    string
	CredentialID string
	Name         string
}

// WebAuthnCeremonyData is the persisted state of an in-flight WebAuthn ceremony.
// It is marshalled into the iam_challenges `data` jsonb column. PublicKey holds
// the publicKey credential options surfaced to the browser (also mirrored on the
// returned Challenge), while Session is the opaque marshalled go-webauthn
// SessionData (challenge bytes, RP id, allowed credentials, user verification)
// the library requires to verify the matching Finish* response. AccountID scopes
// a registration ceremony to its owner.
type WebAuthnCeremonyData struct {
	PublicKey map[string]any `json:"publicKey,omitempty"`
	Session   []byte         `json:"session,omitempty"`
	AccountID string         `json:"accountId,omitempty"`
}

// WebAuthnStoredCredential is the persisted form of a registered passkey. The
// public, display-facing fields live on Credential; Library is the opaque
// marshalled go-webauthn Credential (id, COSE public key, sign count, AAGUID,
// flags, attestation) the library needs to verify subsequent assertions. It is
// marshalled into the iam_webauthn_credentials `data` jsonb column.
type WebAuthnStoredCredential struct {
	Credential WebAuthnCredential `json:"credential"`
	Library    []byte             `json:"library,omitempty"`
}
