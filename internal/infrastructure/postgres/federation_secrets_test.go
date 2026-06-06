package postgres

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

func newTestCipher(t *testing.T) Cipher {
	t.Helper()
	c, err := NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestFedConnSecretsRoundTrip(t *testing.T) {
	c := newTestCipher(t)
	conn := &domain.Connection{
		Config: &domain.FederationConnectionConfig{
			Saml: &domain.FederationSamlConfig{SPPrivateKeyPEM: "saml-private-key"},
			Oidc: &domain.FederationOidcConfig{ClientSecret: "oidc-client-secret"},
		},
	}
	if err := fedEncryptConnSecrets(c, conn); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(conn.Config.Saml.SPPrivateKeyPEM, cipherPrefix) {
		t.Errorf("SP key not encrypted: %q", conn.Config.Saml.SPPrivateKeyPEM)
	}
	if !strings.HasPrefix(conn.Config.Oidc.ClientSecret, cipherPrefix) {
		t.Errorf("client secret not encrypted: %q", conn.Config.Oidc.ClientSecret)
	}
	if err := fedDecryptConnSecrets(c, conn); err != nil {
		t.Fatal(err)
	}
	if conn.Config.Saml.SPPrivateKeyPEM != "saml-private-key" {
		t.Errorf("SP key round trip failed: %q", conn.Config.Saml.SPPrivateKeyPEM)
	}
	if conn.Config.Oidc.ClientSecret != "oidc-client-secret" {
		t.Errorf("client secret round trip failed: %q", conn.Config.Oidc.ClientSecret)
	}
}

func TestFedConnSecretsNilSafe(t *testing.T) {
	c := newTestCipher(t)
	// nil Config and one-sided configs must not panic.
	for _, conn := range []*domain.Connection{
		{},
		{Config: &domain.FederationConnectionConfig{}},
		{Config: &domain.FederationConnectionConfig{Saml: &domain.FederationSamlConfig{}}},
	} {
		if err := fedEncryptConnSecrets(c, conn); err != nil {
			t.Fatal(err)
		}
		if err := fedDecryptConnSecrets(c, conn); err != nil {
			t.Fatal(err)
		}
	}
}
