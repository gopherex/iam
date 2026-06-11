package postgres

// consent_gate_pg.go enforces the consent.required gate shared by the two signup
// paths:
//
//   - flow signup  (coreauth_flows_pg.go): after the identity step the flow halts
//     at step=accept_consents until every required document is accepted.
//   - non-flow POST /v1/auth/signup (coreauth_pg.go Register): the request is
//     rejected up-front when a required document is missing.
//
// Both paths read the consent config through the shared configReader (env-scoped,
// cached, tolerant). The config domain types and their fail-CLOSED admin-write
// validation live in internal/domain/configspec.go — this file only wires the
// dormant runtime gate and does not re-validate.
//
// Backward-compat contract: when no consent doc is configured (or none is
// required:true) the gate is a no-op, so projects without consent behave exactly
// as before.

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gopherex/iam/internal/domain"
)

// parseConsentAccept decodes the flow accept_consents payload value into a slice
// of acceptances. The value is the raw bytes of payload["accept"]: either a JSON
// array ([{"key","version"},…]) or a JSON-encoded string wrapping that array
// (for clients restricted to string scalars). An empty value yields nil (no
// acceptances), letting the gate report every requirement as missing.
func parseConsentAccept(raw string) ([]domain.AccountConsentAcceptance, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var accepted []domain.AccountConsentAcceptance
	if err := json.Unmarshal([]byte(raw), &accepted); err == nil {
		return accepted, nil
	}
	// Fallback: the value is a JSON string that itself encodes the array.
	var inner string
	if err := json.Unmarshal([]byte(raw), &inner); err == nil {
		if err := json.Unmarshal([]byte(inner), &accepted); err == nil {
			return accepted, nil
		}
	}
	return nil, domain.ErrBadRequest.WithMessage("accept must be a JSON array of {key,version}")
}

// consentRequiredDoc is one required consent document resolved for a given
// locale: the (key, version) the user must accept plus the resolved locale to
// stamp on the acceptance row.
type consentRequiredDoc struct {
	Key     string
	Version string
	Locale  string
}

// resolveRequiredConsents reduces the configured documents to one required entry
// per key, picking the document whose locale best matches the requested locale.
//
// Resolution order per key (mirrors the repo locale chain req→…→en→first):
//  1. exact match on the requested locale (when non-empty),
//  2. the project default locale,
//  3. a locale-less document,
//  4. the first required document seen for that key.
//
// Non-required documents never gate. Returns nil when nothing is required.
func resolveRequiredConsents(docs []domain.ConsentDocumentSpec, requestedLocale, defaultLocale string) []consentRequiredDoc {
	// Group the required documents by key, preserving first-seen order of keys.
	type cand struct {
		version string
		locale  string
	}
	byKey := make(map[string][]cand)
	var order []string
	for _, d := range docs {
		if d.Required == nil || !*d.Required {
			continue
		}
		if d.Key == "" || d.Version == "" {
			continue
		}
		loc := ""
		if d.Locale != nil {
			loc = *d.Locale
		}
		if _, ok := byKey[d.Key]; !ok {
			order = append(order, d.Key)
		}
		byKey[d.Key] = append(byKey[d.Key], cand{version: d.Version, locale: loc})
	}
	if len(order) == 0 {
		return nil
	}

	pick := func(cands []cand) cand {
		// 1. exact requested locale
		if requestedLocale != "" {
			for _, c := range cands {
				if c.locale == requestedLocale {
					return c
				}
			}
		}
		// 2. project default locale
		if defaultLocale != "" {
			for _, c := range cands {
				if c.locale == defaultLocale {
					return c
				}
			}
		}
		// 3. locale-less document
		for _, c := range cands {
			if c.locale == "" {
				return c
			}
		}
		// 4. first one
		return cands[0]
	}

	out := make([]consentRequiredDoc, 0, len(order))
	for _, key := range order {
		c := pick(byKey[key])
		out = append(out, consentRequiredDoc{Key: key, Version: c.version, Locale: c.locale})
	}
	return out
}

// missingRequiredConsents returns the required documents that are NOT covered by
// the supplied acceptances. An acceptance covers a requirement only on an exact
// (key, version) match — accepting an older version does not satisfy a newer
// requirement. Validation is against the server-resolved requirement set, so a
// tampered client cannot bypass the gate by claiming extra acceptances.
func missingRequiredConsents(required []consentRequiredDoc, accepted []domain.AccountConsentAcceptance) []consentRequiredDoc {
	if len(required) == 0 {
		return nil
	}
	have := make(map[domain.FlowConsentRef]struct{}, len(accepted))
	for _, a := range accepted {
		have[domain.FlowConsentRef{Key: a.Key, Version: a.Version}] = struct{}{}
	}
	var missing []consentRequiredDoc
	for _, r := range required {
		if _, ok := have[domain.FlowConsentRef{Key: r.Key, Version: r.Version}]; !ok {
			missing = append(missing, r)
		}
	}
	return missing
}

// consentRefDetails renders missing requirements as a details payload for the
// 403 consent_required error.
func consentRefDetails(missing []consentRequiredDoc) map[string]any {
	refs := make([]map[string]string, 0, len(missing))
	for _, m := range missing {
		refs = append(refs, map[string]string{"key": m.Key, "version": m.Version})
	}
	return map[string]any{"missing": refs}
}

// coreAuthCheckRequiredConsents enforces the consent gate for the non-flow
// POST /v1/auth/signup path. It loads the consent config for the request env and
// rejects with domain.ErrConsentRequired when any required document is not
// present (exact key+version) in the supplied acceptances.
//
// Fail policy: not-found/empty config → no requirement (nil); a genuine config
// read error is propagated (fail-closed) so a transient error never silently
// skips a configured legal gate.
func (a *pgCoreAuth) coreAuthCheckRequiredConsents(ctx context.Context, projectID, requestedLocale string, accepted []domain.AccountConsentAcceptance) error {
	docs, err := a.cfg.ConsentConfig(ctx, projectID)
	if err != nil {
		return err
	}
	defLocale, _ := a.cfg.AuthConfig(ctx, projectID)
	required := resolveRequiredConsents(docs, requestedLocale, defLocale.DefaultLocale)
	missing := missingRequiredConsents(required, accepted)
	if len(missing) > 0 {
		return domain.ErrConsentRequired.WithDetails(consentRefDetails(missing))
	}
	return nil
}

// coreAuthConsentLocales builds a key→resolved-locale map for stamping accepted
// consent rows. The locale per key follows the same resolution chain as the
// gate. Best-effort: a read error yields an empty map (locale left null).
func (a *pgCoreAuth) coreAuthConsentLocales(ctx context.Context, projectID, requestedLocale string) map[string]string {
	docs, err := a.cfg.ConsentConfig(ctx, projectID)
	if err != nil || len(docs) == 0 {
		return map[string]string{}
	}
	defLocale, _ := a.cfg.AuthConfig(ctx, projectID)
	out := make(map[string]string)
	for _, r := range resolveConsentLocales(docs, requestedLocale, defLocale.DefaultLocale) {
		if r.Locale != "" {
			out[r.Key] = r.Locale
		}
	}
	return out
}

// resolveConsentLocales resolves the best locale per key across ALL documents
// (not just required ones), so non-required acceptances still record a locale.
func resolveConsentLocales(docs []domain.ConsentDocumentSpec, requestedLocale, defaultLocale string) []consentRequiredDoc {
	all := make([]domain.ConsentDocumentSpec, 0, len(docs))
	yes := true
	for _, d := range docs {
		dd := d
		dd.Required = &yes // treat every doc as a candidate for locale resolution
		all = append(all, dd)
	}
	return resolveRequiredConsents(all, requestedLocale, defaultLocale)
}
