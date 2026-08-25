package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gopherex/iam/internal/domain"
)

// SoftAuthMiddleware resolves the caller's session without requiring one.
//
// Public operations declare no security, so the generated server never runs a
// security handler for them and no principal reaches the handler — which is
// exactly wrong for /oauth2/authorize: it is public because the user may not be
// signed in yet, but when they ARE signed in the endpoint must notice. Without
// that, `prompt=none` can never succeed, an already-authenticated user is asked
// for their password again at every relying party, and single sign-on is single
// in name only.
//
// A failed or absent credential is not an error here: the request proceeds
// unauthenticated, and the operation's own security (if it declares any) still
// decides. Place it INSIDE CookieAuthMiddleware, which is what promotes the
// session cookie to the bearer header this reads.
func SoftAuthMiddleware(a Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			p, err := a.User(r.Context(), token)
			if err != nil || p == nil {
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(withSoftPrincipal(r.Context(), p)))
		})
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}

	return strings.TrimSpace(header[len(prefix):])
}

const softPrincipalKey ctxKey = iota + 1

func withSoftPrincipal(ctx context.Context, p *domain.Principal) context.Context {
	return context.WithValue(ctx, softPrincipalKey, p)
}

// SoftPrincipalFrom returns the caller's session on a PUBLIC operation, if they
// have one. Handlers that require authentication must keep using PrincipalFrom —
// this one is deliberately allowed to be absent.
func SoftPrincipalFrom(ctx context.Context) (*domain.Principal, bool) {
	p, ok := ctx.Value(softPrincipalKey).(*domain.Principal)
	return p, ok && p != nil
}
