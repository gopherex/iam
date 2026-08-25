package postgres

// Signed request objects (JAR, RFC 9101).
//
// The whole authorization request arrives as a JWT the client signed, instead of
// as query parameters the browser carries. Everything in a query string is
// modifiable by whatever sits between the client and the user's browser; a
// signed object is not. When one is present its parameters WIN over the query,
// because the point is that the query cannot change the request.
//
// This is the by-value form. The by-reference form of RFC 9101 (a `request_uri`
// pointing at a client-hosted URL) is deliberately not accepted: `request_uri`
// here means a pushed request (RFC 9126), which achieves the same protection by
// lodging the request over the client's authenticated back channel and never
// requires us to fetch from an address the browser chose.

import (
	"context"

	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/gopherex/iam/internal/domain"
)

// consumeRequestObject verifies a signed request object and returns the
// authorization request it carries.
//
// client_id must be present in the query as well and must match the object
// (RFC 9101 §5): we have to know whose key to verify against before we can trust
// anything inside.
func (a *pgOIDCGrants) consumeRequestObject(
	ctx context.Context, cmd domain.OIDCAuthorizeCmd,
) (domain.OIDCAuthorizeCmd, error) {
	_, app, err := a.resolveAuthorizeClient(ctx, cmd.ClientID)
	if err != nil {
		return cmd, err
	}

	keys, err := a.clientKeys.keysFor(ctx, app)
	if err != nil {
		return cmd, domain.ErrBadRequest.WithMessage(
			"the client has published no keys to verify a request object against")
	}

	tok, err := jwt.Parse([]byte(cmd.Request),
		jwt.WithKeySet(keys),
		jwt.WithValidate(true),
	)
	if err != nil {
		return cmd, domain.ErrBadRequest.WithMessage("the request object did not verify")
	}
	// The object speaks for one client only, and it is the one that signed it.
	if issuer, ok := tok.Issuer(); ok && issuer != cmd.ClientID {
		return cmd, domain.ErrBadRequest.WithMessage("the request object was issued for a different client")
	}

	out := cmd
	out.Request = ""

	if err := mergeRequestObject(&out, tok, cmd.ClientID); err != nil {
		return cmd, err
	}

	return out, nil
}

// mergeRequestObject copies the object's parameters over the query's. The
// direction is the point: an attacker who can rewrite the URL must not be able
// to change what the client asked for.
func mergeRequestObject(out *domain.OIDCAuthorizeCmd, tok jwt.Token, clientID string) error {
	str := func(name string) string {
		var v string
		if err := tok.Get(name, &v); err != nil {
			return ""
		}

		return v
	}

	if v := str("client_id"); v != "" && v != clientID {
		return domain.ErrBadRequest.WithMessage("the request object names a different client")
	}

	for _, field := range []struct {
		name string
		set  func(string)
	}{
		{"response_type", func(v string) { out.ResponseType = v }},
		{"redirect_uri", func(v string) { out.RedirectURI = v }},
		{claimScope, func(v string) { out.Scope = v }},
		{"state", func(v string) { out.State = v }},
		{"nonce", func(v string) { out.Nonce = v }},
		{"code_challenge", func(v string) { out.CodeChallenge = v }},
		{"code_challenge_method", func(v string) { out.CodeChallengeMethod = v }},
		{"prompt", func(v string) { out.Prompt = v }},
		{"response_mode", func(v string) { out.ResponseMode = v }},
	} {
		if v := str(field.name); v != "" {
			field.set(v)
		}
	}

	var maxAge int
	if err := tok.Get("max_age", &maxAge); err == nil && maxAge > 0 {
		out.MaxAge = maxAge
	}

	return nil
}
