package postgres

import (
	"encoding/base64"
	"strings"
	"testing"
)

func testKey() string {
	return base64.StdEncoding.EncodeToString(make([]byte, 32)) // all-zero 32-byte key
}

func TestAESCipherRoundTrip(t *testing.T) {
	c, err := NewCipher(testKey())
	if err != nil {
		t.Fatal(err)
	}
	plain := "-----BEGIN RSA PRIVATE KEY-----\nsecret\n-----END RSA PRIVATE KEY-----"
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, cipherPrefix) {
		t.Fatalf("ciphertext missing prefix: %q", enc)
	}
	if strings.Contains(enc, "secret") {
		t.Fatal("plaintext leaked into ciphertext")
	}
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != plain {
		t.Fatalf("round trip mismatch: got %q", dec)
	}
}

func TestAESCipherNonceVaries(t *testing.T) {
	c, _ := NewCipher(testKey())
	a, _ := c.Encrypt("x")
	b, _ := c.Encrypt("x")
	if a == b {
		t.Fatal("same plaintext produced identical ciphertext (nonce reuse)")
	}
}

func TestCipherLegacyPlaintextPassthrough(t *testing.T) {
	c, _ := NewCipher(testKey())
	// A value without the enc: prefix is treated as legacy plaintext.
	got, err := c.Decrypt("legacy-plaintext")
	if err != nil || got != "legacy-plaintext" {
		t.Fatalf("legacy passthrough failed: %q %v", got, err)
	}
}

func TestIdentityCipher(t *testing.T) {
	c, err := NewCipher("") // empty key -> identity
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := c.Encrypt("plain")
	if enc != "plain" {
		t.Fatalf("identity should passthrough, got %q", enc)
	}
	// Identity must refuse to silently lose an encrypted value.
	if _, err := c.Decrypt(cipherPrefix + "abc"); err == nil {
		t.Fatal("identity decrypt of enc:v1 value should error")
	}
}

func TestNewCipherBadKey(t *testing.T) {
	if _, err := NewCipher("not-base64!!!"); err == nil {
		t.Fatal("expected error for non-base64 key")
	}
	if _, err := NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 16))); err == nil {
		t.Fatal("expected error for 16-byte key (need 32)")
	}
}
