package webhooksecret_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gopherex/iam/internal/webhooksecret"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, secret string
		key          []byte
		err          error
	}{
		{name: "standard", secret: "whsec_c2VjcmV0", key: []byte("secret")},
		{name: "binary key", secret: "whsec_AP+A", key: []byte{0, 255, 128}},
		{name: "legacy", secret: "legacy-secret", key: []byte("legacy-secret")},
		{name: "legacy base64 remains literal", secret: "c2VjcmV0", key: []byte("c2VjcmV0")},
		{name: "whitespace", secret: " \twhsec_c2VjcmV0\n", key: []byte("secret")},
		{name: "empty", err: webhooksecret.ErrRequired},
		{name: "blank", secret: " \t", err: webhooksecret.ErrRequired},
		{name: "empty encoded", secret: "whsec_", err: webhooksecret.ErrInvalid},
		{name: "malformed", secret: "whsec_not-base64!", err: webhooksecret.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key, err := webhooksecret.Decode(tc.secret)
			if !errors.Is(err, tc.err) {
				t.Fatalf("error=%v want=%v", err, tc.err)
			}

			if !bytes.Equal(key, tc.key) {
				t.Fatal("decoded key mismatch")
			}
		})
	}
}
