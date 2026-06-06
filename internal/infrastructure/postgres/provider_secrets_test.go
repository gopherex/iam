package postgres

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/go-faster/jx"
)

func TestProviderConfigCryptRoundTrip(t *testing.T) {
	c, err := NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	cfg := map[string]jx.Raw{
		"host":     jx.Raw(`"smtp.example.com"`), // not secret -> untouched
		"port":     jx.Raw(`587`),                // non-string -> untouched
		"password": jx.Raw(`"s3cr3t"`),           // secret -> encrypted
		"api_key":  jx.Raw(`"ak_live_123"`),      // secret -> encrypted
	}
	enc, err := encryptProviderConfig(c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if string(enc["host"]) != `"smtp.example.com"` || string(enc["port"]) != `587` {
		t.Errorf("non-secret keys mutated: %v %v", string(enc["host"]), string(enc["port"]))
	}
	if !strings.Contains(string(enc["password"]), cipherPrefix) {
		t.Errorf("password not encrypted: %s", enc["password"])
	}
	if strings.Contains(string(enc["password"]), "s3cr3t") {
		t.Error("plaintext password leaked")
	}

	dec, err := decryptProviderConfig(c, enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(dec["password"]) != `"s3cr3t"` {
		t.Errorf("password round trip failed: %s", dec["password"])
	}
	if string(dec["api_key"]) != `"ak_live_123"` {
		t.Errorf("api_key round trip failed: %s", dec["api_key"])
	}
}

func TestProviderConfigCryptNil(t *testing.T) {
	c, _ := NewCipher("")
	out, err := encryptProviderConfig(c, nil)
	if err != nil || out != nil {
		t.Fatalf("nil config: got %v %v", out, err)
	}
}
