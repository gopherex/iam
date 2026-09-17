// Package webhooksecret decodes signing secrets shared by IAM and its SDK.
package webhooksecret

import (
	"encoding/base64"
	"errors"
	"strings"
)

var (
	ErrRequired = errors.New("webhook signing secret is required")
	ErrInvalid  = errors.New("invalid webhook signing secret")
)

// Decode returns the HMAC key: decoded bytes for whsec_ secrets, or the value
// itself for legacy secrets. Malformed prefixed secrets never fall back to raw.
func Decode(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrRequired
	}

	if encoded, ok := strings.CutPrefix(value, "whsec_"); ok {
		key, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(key) == 0 {
			return nil, ErrInvalid
		}

		return key, nil
	}

	return []byte(value), nil
}
