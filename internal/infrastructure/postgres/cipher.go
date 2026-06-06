package postgres

// Secrets at rest: reversible secrets that the service must read back (signing-key
// private PEMs, TOTP shared secrets, external IdP client secrets) are encrypted
// with AES-256-GCM before they touch a column, and decrypted on read. Hashed
// credentials (passwords, API-key/SCIM/app-secret sha256) are NOT handled here —
// they are already one-way.
//
// Format: an encrypted value is "enc:v1:<base64(nonce || ciphertext)>". Decrypt
// recognises that prefix; any value without it is treated as legacy plaintext and
// returned as-is, so encryption can be enabled on an existing database and rows
// migrate lazily (re-encrypted on their next write).

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

// cipherPrefix tags an AES-256-GCM v1 ciphertext.
const cipherPrefix = "enc:v1:"

// Cipher encrypts and decrypts reversible secrets at rest.
type Cipher interface {
	// Encrypt returns the enc:v1 ciphertext of plaintext.
	Encrypt(plaintext string) (string, error)
	// Decrypt returns the plaintext for an enc:v1 value, or the input unchanged
	// when it is legacy plaintext (no enc: prefix).
	Decrypt(value string) (string, error)
}

// identityCipher is the no-op cipher used when no encryption key is configured:
// values are stored as-is. Decrypt still transparently handles any pre-existing
// enc:v1 values it cannot read by erroring, so a key cannot be silently dropped.
type identityCipher struct{}

// NewIdentityCipher returns a passthrough Cipher (no encryption).
func NewIdentityCipher() Cipher { return identityCipher{} }

func (identityCipher) Encrypt(plaintext string) (string, error) { return plaintext, nil }

func (identityCipher) Decrypt(value string) (string, error) {
	if strings.HasPrefix(value, cipherPrefix) {
		return "", errors.New("postgres: encrypted secret found but no encryption key is configured")
	}
	return value, nil
}

// aesCipher is the AES-256-GCM Cipher.
type aesCipher struct{ aead cipher.AEAD }

// NewCipher builds an AES-256-GCM Cipher from a base64-encoded 32-byte key.
// An empty key yields the identity (passthrough) cipher.
func NewCipher(keyB64 string) (Cipher, error) {
	if keyB64 == "" {
		return NewIdentityCipher(), nil
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, errors.New("postgres: encryption key must be base64-encoded")
	}
	if len(key) != 32 {
		return nil, errors.New("postgres: encryption key must decode to 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &aesCipher{aead: aead}, nil
}

func (c *aesCipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return cipherPrefix + base64.StdEncoding.EncodeToString(ct), nil
}

func (c *aesCipher) Decrypt(value string) (string, error) {
	if !strings.HasPrefix(value, cipherPrefix) {
		return value, nil // legacy plaintext
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, cipherPrefix))
	if err != nil {
		return "", err
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("postgres: ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
