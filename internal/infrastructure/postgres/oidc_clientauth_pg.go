package postgres

// Client authentication by assertion (private_key_jwt, RFC 7523 §3).
//
// The client proves who it is by signing a short-lived JWT with a key only it
// holds, instead of presenting a secret both sides know. Nothing shareable ever
// travels, so there is no secret to leak from our storage or from the client's
// configuration — which is why FAPI requires it.
//
// client_secret_jwt (HMAC over the shared secret) is deliberately NOT supported:
// we store only the sha256 of a client secret, and keeping it reversible so we
// could compute an HMAC would weaken storage for a method that is strictly
// weaker than this one.

import (
	"context"
	"errors"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/gopherex/iam/internal/domain"
)

// oidcClientAssertionType is the only assertion type RFC 7523 defines for client
// authentication.
const oidcClientAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

// oidcAssertionMaxLifetime bounds how far ahead a client assertion may expire.
// A long-lived assertion is a bearer credential in all but name.
const oidcAssertionMaxLifetime = 5 * time.Minute

// errAssertionRejected is the base for every assertion failure. The client is
// told the assertion did not authenticate it, never which check it failed —
// naming the check would help someone forging one.
var errAssertionRejected = errors.New("client assertion rejected")

// verifyClientAssertion authenticates a client from its signed assertion and
// returns the client id it proved.
//
// audience is what the assertion must be addressed to: the endpoint's URL or the
// issuer. Accepting either follows RFC 7523 §3, where deployments differ.
func (a *pgOIDCGrants) verifyClientAssertion(
	ctx context.Context, projectID, env, assertionType, assertion, audience string,
) (string, string, string, error) {
	if assertion == "" {
		return "", "", "", nil // not using assertion auth
	}

	if assertionType != oidcClientAssertionType {
		return "", "", "", domain.ErrUnauthorized.WithMessage("unsupported client_assertion_type")
	}
	// Peek to learn WHICH client is claiming to sign this, so we know whose keys
	// to verify against. Nothing from the peek is trusted.
	peek := a.db.Signer().UnverifiedClaims(assertion)
	if peek == nil {
		return "", "", "", domain.ErrUnauthorized.WithMessage(errAssertionRejected.Error())
	}

	clientID := peekString(peek, claimIssuer)
	if clientID == "" || clientID != peekString(peek, claimSubject) {
		// RFC 7523 §3: for client authentication iss and sub are both the client.
		return "", "", "", domain.ErrUnauthorized.WithMessage(errAssertionRejected.Error())
	}

	clientRow, app, err := a.resolveAuthorizeClient(ctx, clientID)
	if err != nil {
		return "", "", "", domain.ErrUnauthorized.WithMessage(errAssertionRejected.Error())
	}
	// The client's own row decides the tenant. When the caller already knows one
	// — it authenticated some other way too — they must agree, or a client from
	// another project could act inside this one.
	assertedProject, assertedEnv := clientRow.ProjectID, clientRow.Environment
	if projectID != "" && projectID != assertedProject {
		return "", "", "", domain.ErrUnauthorized.WithMessage(errAssertionRejected.Error())
	}

	keys, err := a.clientKeys.keysFor(ctx, app)
	if err != nil {
		return "", "", "", domain.ErrUnauthorized.WithMessage(
			"the client has published no keys to verify an assertion against")
	}

	jti, expires, err := parseClientAssertion(assertion, clientID, audience, keys)
	if err != nil {
		return "", "", "", err
	}

	if env == "" {
		env = assertedEnv
	}
	// Single use. A replayed assertion is a captured one.
	fresh, err := claimOnce(ctx, a.db, assertedProject, env, oidcAssertionJTI(clientID, jti), expires)
	if err != nil {
		return "", "", "", err
	}

	if !fresh {
		return "", "", "", domain.ErrUnauthorized.WithMessage(errAssertionRejected.Error())
	}

	return clientID, assertedProject, assertedEnv, nil
}

// parseClientAssertion verifies the assertion's signature and its claim set,
// returning the id that makes it single-use and when it expires.
func parseClientAssertion(assertion, clientID, audience string, keys jwk.Set) (string, time.Time, error) {
	rejected := domain.ErrUnauthorized.WithMessage(errAssertionRejected.Error())

	tok, err := jwt.Parse([]byte(assertion),
		jwt.WithKeySet(keys),
		jwt.WithValidate(true),
		jwt.WithAudience(audience),
		jwt.WithIssuer(clientID),
		jwt.WithRequiredClaim("exp"),
		jwt.WithRequiredClaim("jti"),
	)
	if err != nil {
		return "", time.Time{}, rejected
	}

	expires, ok := tok.Expiration()
	if !ok || expires.After(nowUTC().Add(oidcAssertionMaxLifetime)) {
		// A long-lived assertion is a bearer credential; captured once, reusable
		// for as long as its author felt like.
		return "", time.Time{}, rejected
	}

	var jti string
	if err := tok.Get("jti", &jti); err != nil || jti == "" {
		return "", time.Time{}, rejected
	}

	return jti, expires, nil
}

// oidcAssertionJTI namespaces an assertion's jti so it cannot collide with a
// token we issued, and so two clients choosing the same jti do not block each
// other.
func oidcAssertionJTI(clientID, jti string) string {
	return "assertion:" + clientID + ":" + jti
}

// oidcAssertionAlgorithms are the signing algorithms a client assertion or a
// request object may use, advertised in the discovery document. `none` is
// excluded by construction — an unsigned assertion proves nothing.
func oidcAssertionAlgorithms() []string {
	return []string{
		jwa.RS256().String(), jwa.RS384().String(), jwa.RS512().String(),
		jwa.ES256().String(), jwa.ES384().String(), jwa.PS256().String(),
	}
}
