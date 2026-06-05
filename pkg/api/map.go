package api

import (
	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

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
