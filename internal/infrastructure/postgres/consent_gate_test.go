package postgres

// consent_gate_test.go — unit tests for the pure consent-gate helpers (locale
// resolution + required-set matching). No DB required.

import (
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

func TestResolveRequiredConsents_LocalePreference(t *testing.T) {
	docs := []domain.ConsentDocumentSpec{
		{Key: "tos", Version: "v-en", Locale: ptr("en"), Required: ptr(true)},
		{Key: "tos", Version: "v-ru", Locale: ptr("ru"), Required: ptr(true)},
		{Key: "tos", Version: "v-none", Required: ptr(true)},
		{Key: "marketing", Version: "m1", Locale: ptr("en"), Required: ptr(false)}, // not required
	}

	tests := []struct {
		name       string
		requested  string
		def        string
		wantTosVer string
		wantNumReq int
	}{
		{"exact requested locale", "ru", "en", "v-ru", 1},
		{"falls back to project default", "fr", "en", "v-en", 1},
		{"falls back to locale-less", "fr", "de", "v-none", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveRequiredConsents(docs, tc.requested, tc.def)
			if len(got) != tc.wantNumReq {
				t.Fatalf("required count = %d, want %d (%+v)", len(got), tc.wantNumReq, got)
			}
			if got[0].Key != "tos" || got[0].Version != tc.wantTosVer {
				t.Fatalf("tos resolved = %s/%s, want tos/%s", got[0].Key, got[0].Version, tc.wantTosVer)
			}
		})
	}
}

func TestResolveRequiredConsents_NoneRequired(t *testing.T) {
	docs := []domain.ConsentDocumentSpec{
		{Key: "marketing", Version: "m1", Required: ptr(false)},
		{Key: "analytics", Version: "a1"}, // required nil
	}
	if got := resolveRequiredConsents(docs, "en", "en"); got != nil {
		t.Fatalf("want nil for no required docs, got %+v", got)
	}
}

func TestMissingRequiredConsents(t *testing.T) {
	required := []consentRequiredDoc{
		{Key: "tos", Version: "v1"},
		{Key: "privacy", Version: "p1"},
	}

	t.Run("all accepted exactly", func(t *testing.T) {
		accepted := []domain.AccountConsentAcceptance{
			{Key: "tos", Version: "v1"},
			{Key: "privacy", Version: "p1"},
			{Key: "marketing", Version: "m1"}, // extra, ignored
		}
		if m := missingRequiredConsents(required, accepted); len(m) != 0 {
			t.Fatalf("missing = %+v, want none", m)
		}
	})

	t.Run("version mismatch counts as missing", func(t *testing.T) {
		accepted := []domain.AccountConsentAcceptance{
			{Key: "tos", Version: "v0"}, // wrong version
			{Key: "privacy", Version: "p1"},
		}
		m := missingRequiredConsents(required, accepted)
		if len(m) != 1 || m[0].Key != "tos" {
			t.Fatalf("missing = %+v, want [tos]", m)
		}
	})

	t.Run("empty acceptances → all missing", func(t *testing.T) {
		if m := missingRequiredConsents(required, nil); len(m) != 2 {
			t.Fatalf("missing = %+v, want 2", m)
		}
	})

	t.Run("nothing required → nothing missing", func(t *testing.T) {
		if m := missingRequiredConsents(nil, nil); m != nil {
			t.Fatalf("missing = %+v, want nil", m)
		}
	})
}
