package postgres

// Provider config secrets at rest. iam_providers stores an untyped config blob
// (map[string]jx.Raw) for each notification provider — SMTP/SMS/IdP settings
// that include credentials under conventional key names. We encrypt the string
// values of those known secret keys with the DB cipher on write and decrypt them
// on read; non-secret keys (host, port, from, region, …) stay in clear so the
// config remains queryable/debuggable.

import (
	"encoding/json"
	"strings"

	"github.com/go-faster/jx"
)

// providerSecretKeys are the config keys whose values are treated as secrets and
// encrypted at rest. Matched case-insensitively.
var providerSecretKeys = map[string]struct{}{
	"password":      {},
	"api_key":       {},
	"apikey":        {},
	"secret":        {},
	"secret_key":    {},
	"client_secret": {},
	"auth_token":    {},
	"token":         {},
	"private_key":   {},
	"access_key":    {},
	"access_key_id": {},
	"access_token":  {},
	"sasl_password": {},
}

// providerConfigCrypt applies transform (Encrypt or Decrypt) to the string value
// of every recognised secret key, leaving other keys and non-string values
// untouched. The input map is not mutated; a new map is returned.
func providerConfigCrypt(cfg map[string]jx.Raw, transform func(string) (string, error)) (map[string]jx.Raw, error) {
	if cfg == nil {
		return nil, nil
	}
	out := make(map[string]jx.Raw, len(cfg))
	for k, v := range cfg {
		if _, ok := providerSecretKeys[strings.ToLower(k)]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil { // only transform JSON strings
				t, err := transform(s)
				if err != nil {
					return nil, err
				}
				b, err := json.Marshal(t)
				if err != nil {
					return nil, err
				}
				out[k] = jx.Raw(b)
				continue
			}
		}
		out[k] = v
	}
	return out, nil
}

func encryptProviderConfig(c Cipher, cfg map[string]jx.Raw) (map[string]jx.Raw, error) {
	return providerConfigCrypt(cfg, c.Encrypt)
}

func decryptProviderConfig(c Cipher, cfg map[string]jx.Raw) (map[string]jx.Raw, error) {
	return providerConfigCrypt(cfg, c.Decrypt)
}

// rawToJSON / jsonToRaw bridge the domain config map (jx.Raw) and the at-rest
// storage map (json.RawMessage). jx.Raw is a plain []byte alias with no
// MarshalJSON, so json.Marshal would base64-encode each value; json.RawMessage
// marshals verbatim. Storing as json.RawMessage keeps iam_providers.data real
// JSON so every reader (admin API and the notification publisher alike) gets the
// clear config values, not base64.
func rawToJSON(in map[string]jx.Raw) map[string]json.RawMessage {
	if in == nil {
		return nil
	}
	out := make(map[string]json.RawMessage, len(in))
	for k, v := range in {
		out[k] = json.RawMessage(v)
	}
	return out
}

func jsonToRaw(in map[string]json.RawMessage) map[string]jx.Raw {
	if in == nil {
		return nil
	}
	out := make(map[string]jx.Raw, len(in))
	for k, v := range in {
		out[k] = jx.Raw(v)
	}
	return out
}
