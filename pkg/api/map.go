package api

import (
	"net/url"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

// optURI wraps a redirect target string as an oas.OptURI for 302 Found
// responses; an unparseable URL yields an unset value.
func optURI(raw string) oas.OptURI {
	u, err := url.Parse(raw)
	if err != nil {
		return oas.OptURI{}
	}
	return oas.NewOptURI(*u)
}

// oas <-> domain mappers. The wire types (oas) stay at the edge; services map
// to/from domain at the boundary so business logic never depends on the
// generated contract.

func oasUser(a *domain.Account) oas.User {
	u := oas.User{
		ID:            a.ID,
		Kind:          oas.UserKind(a.Kind),
		Status:        oas.UserStatus(a.Status),
		EmailVerified: oas.NewOptBool(a.EmailVerified),
	}
	if a.PrimaryEmail != "" {
		u.PrimaryEmail = oas.NewOptNilString(a.PrimaryEmail)
	}
	if a.PrimaryPhone != "" {
		u.PrimaryPhone = oas.NewOptNilString(a.PrimaryPhone)
	}
	// Surface the editable profile (name/locale) so PATCH /v1/users/me and reads
	// reflect the stored values. Locale is only set when non-empty: an empty
	// string fails the oas locale pattern on response validation.
	if a.Name != "" || a.Locale != "" {
		var prof oas.CoreProfile
		if a.Name != "" {
			prof.Name = oas.NewOptNilString(a.Name)
		}
		if a.Locale != "" {
			prof.Locale = oas.NewOptString(a.Locale)
		}
		u.Profile = oas.NewOptCoreProfile(prof)
	}
	return u
}

func oasSessionTokens(s *domain.Session) oas.SessionTokens {
	t := oas.SessionTokens{
		AccessToken: s.AccessToken,
		ExpiresIn:   s.ExpiresIn,
		TokenType:   "Bearer",
	}
	if s.RefreshToken != "" {
		t.RefreshToken = oas.NewOptString(s.RefreshToken)
	}
	return t
}

func oasSession(s *domain.Session) oas.Session {
	return oas.Session{
		ID:       s.ID,
		UserID:   oas.NewOptString(s.AccountID),
		ClientID: oas.NewOptString(s.ClientID),
		Amr:      s.AMR,
		Aal:      oas.NewOptSessionAal(oas.SessionAal(s.AAL)),
	}
}

// authResult builds the authenticated AuthResult from a freshly issued session.
func authResult(a *domain.Account, s *domain.Session) *oas.AuthResult {
	return &oas.AuthResult{
		ResultType: oas.AuthResultResultTypeAuthenticated,
		User:       oasUser(a),
		Session:    oasSessionTokens(s),
	}
}
