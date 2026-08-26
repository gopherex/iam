package postgres

// Token profiles at minting.
//
// A profile is a named set of token settings for one audience: what `aud` says,
// how long the token lives, and a fixed set of extra claims. A client is bound
// to at most one, so the profile answers "what does a token for THIS
// integration look like" — which is the question an operator actually has when
// one relying party needs a five-minute token and another needs an hour.
//
// It deliberately cannot express anything about the individual user. Per-user
// values come from the user's roles and the `groups` scope; a template that
// could reach into the subject would be a second, weaker authorization system.

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// oidcTokenProfile is the resolved profile applied to one mint.
type oidcTokenProfile struct {
	Audience   string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Claims     map[string]any
}

// resolveClientTokenProfile loads the profile a client is bound to, if any.
//
// A missing or deleted profile is not an error: the binding is configuration,
// and a token that cannot be minted because somebody removed a profile would
// take an integration down for a reason nobody would look for. The mint falls
// back to the project defaults instead.
func (a *pgOIDCGrants) resolveClientTokenProfile(ctx context.Context, clientID string) *oidcTokenProfile {
	if clientID == "" {
		return nil
	}

	clientRow, err := models.FindIamAppClient(ctx, a.db.Bobx(), clientID)
	if err != nil {
		return nil
	}

	var app domain.AppClient
	if err := unmarshal(clientRow.Data, &app); err != nil {
		return nil
	}

	if app.TokenProfileID == "" {
		return nil
	}

	profileRow, err := models.FindIamTokenProfile(ctx, a.db.Bobx(), app.TokenProfileID)
	if err != nil || profileRow.ProjectID != clientRow.ProjectID {
		return nil
	}

	profile := adminTokenProfileToDomain(profileRow)

	out := &oidcTokenProfile{
		Audience:   profile.Audience,
		AccessTTL:  time.Duration(profile.AccessTTL) * time.Second,
		RefreshTTL: time.Duration(profile.RefreshTTL) * time.Second,
	}

	if len(profile.ClaimsTemplate) > 0 {
		out.Claims = make(map[string]any, len(profile.ClaimsTemplate))

		for name, raw := range profile.ClaimsTemplate {
			var value any
			if err := json.Unmarshal([]byte(raw), &value); err != nil {
				continue
			}

			out.Claims[name] = value
		}
	}

	return out
}

// applyTokenProfile folds a profile into a mint. Everything it does not set is
// left as the project's default, so a profile that only shortens a lifetime does
// only that.
func applyTokenProfile(
	profile *oidcTokenProfile, claims map[string]any, accessTTL, refreshTTL time.Duration,
) (time.Duration, time.Duration) {
	if profile == nil {
		return accessTTL, refreshTTL
	}

	if profile.AccessTTL > 0 {
		accessTTL = profile.AccessTTL
	}

	if profile.RefreshTTL > 0 {
		refreshTTL = profile.RefreshTTL
	}
	// Template claims are applied first and the standard claims win, so a
	// template cannot restate `sub`, `iss` or `exp` as something else.
	if profile.Audience != "" {
		claims[claimAudience] = profile.Audience
	}

	return accessTTL, refreshTTL
}

// tokenProfileClaims returns the template claims that may be added to a token:
// the ones that do not collide with a claim the provider owns.
func tokenProfileClaims(profile *oidcTokenProfile) map[string]any {
	if profile == nil {
		return nil
	}

	out := make(map[string]any, len(profile.Claims))

	for name, value := range profile.Claims {
		if oidcReservedClaims[name] {
			continue
		}

		out[name] = value
	}

	return out
}

// oidcReservedClaims are the claims the provider owns. A template naming one is
// ignored for that claim rather than rejected: the rest of the profile is still
// what the operator meant, and silently letting a template rewrite `sub` would
// be an impersonation primitive.
//
//nolint:gochecknoglobals // a fixed set, not state.
var oidcReservedClaims = map[string]bool{
	claimIssuer: true, claimSubject: true, claimAudience: true,
	claimExpiresAt: true, claimTokenID: true, claimTokenType: true,
	claimProjectID: true, claimEnvironment: true, claimSessionID: true,
	claimClientID: true, claimScope: true, claimGroups: true,
	claimAAL: true, claimAMR: true, "iat": true, "nbf": true,
}
