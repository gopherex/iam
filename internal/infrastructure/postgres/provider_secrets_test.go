package postgres

import (
	"encoding/base64"
	"encoding/json"
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

// TestProviderConfigStoredAsRealJSON guards the iam_providers.data storage path:
// adminProviderData.Config must serialise to real JSON, not base64. jx.Raw is a
// bare []byte alias, so storing the config map as map[string]jx.Raw made
// json.Marshal base64-encode every value — the admin API round-tripped it back
// (symmetric base64) but the notification publisher, which reads the column via
// map[string]json.RawMessage, got base64 garbage for host/port/from and could
// never connect to SMTP. Config must stay clear for every reader.
func TestProviderConfigStoredAsRealJSON(t *testing.T) {
	c, err := NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	encCfg, err := encryptProviderConfig(c, map[string]jx.Raw{
		"host":     jx.Raw(`"postbox.cloud.yandex.net"`),
		"port":     jx.Raw(`587`),
		"password": jx.Raw(`"s3cr3t"`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write exactly like createProvider/updateProvider.
	raw, err := json.Marshal(adminProviderData{Type: "smtp", Config: rawToJSON(encCfg)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"host":"postbox.cloud.yandex.net"`) {
		t.Fatalf("host not stored as real JSON (base64 regression): %s", raw)
	}

	// Read like the notification publisher: map[string]json.RawMessage, verbatim.
	var pub struct {
		Type   string                     `json:"type"`
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(raw, &pub); err != nil {
		t.Fatal(err)
	}
	var host string
	if err := json.Unmarshal(pub.Config["host"], &host); err != nil {
		t.Fatalf("publisher cannot read host: %v", err)
	}
	if host != "postbox.cloud.yandex.net" {
		t.Fatalf("publisher read host = %q, want clear value", host)
	}
}

func TestProviderConfigCryptNil(t *testing.T) {
	c, _ := NewCipher("")
	out, err := encryptProviderConfig(c, nil)
	if err != nil || out != nil {
		t.Fatalf("nil config: got %v %v", out, err)
	}
}
