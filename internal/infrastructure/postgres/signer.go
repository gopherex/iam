package postgres

// Signer mints and verifies the project/environment JWTs (access tokens, id
// tokens, service/admin tokens) and publishes the JWKS. Keys live in
// iam_signing_keys (RSA private PEM); a key is generated on first use. Backed by
// github.com/lestrrat-go/jwx/v3.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// Signer is the JWT signer/verifier over a project+environment's signing keys.
type Signer struct{ db *DB }

// Signer returns the JWT signer for this connection bundle.
func (db *DB) Signer() *Signer { return &Signer{db: db} }

func (s *Signer) keysFor(ctx context.Context, projectID, env string) ([]*models.IamSigningKey, error) {
	return models.IamSigningKeys.Query(
		sm.Where(models.IamSigningKeys.Columns.ProjectID.EQ(psql.Arg(projectID))),
		sm.Where(models.IamSigningKeys.Columns.Environment.EQ(psql.Arg(env))),
	).All(ctx, s.db.Bobx())
}

// activeKey returns the active signing key (kid + private key) for project/env,
// generating and persisting a fresh RSA-2048 key if none is active.
func (s *Signer) activeKey(ctx context.Context, projectID, env string) (string, *rsa.PrivateKey, error) {
	rows, err := s.keysFor(ctx, projectID, env)
	if err != nil {
		return "", nil, err
	}
	for _, r := range rows {
		if r.Status != "active" {
			continue
		}
		pemStr, ok := r.PrivatePem.Get()
		if !ok || pemStr == "" {
			continue
		}
		decPem, err := s.db.Cipher.Decrypt(pemStr)
		if err != nil {
			return "", nil, err
		}
		priv, err := parsePrivatePEM(decPem)
		if err != nil {
			return "", nil, err
		}
		return r.Kid, priv, nil
	}
	// none active — generate one inside a tx.
	var kid string
	var priv *rsa.PrivateKey
	err = s.db.withTx(ctx, func(ctx context.Context) error {
		var genErr error
		priv, genErr = rsa.GenerateKey(rand.Reader, 2048)
		if genErr != nil {
			return genErr
		}
		kid = newUUID()
		pemStr := encodePrivatePEM(priv)
		encPem, encErr := s.db.Cipher.Encrypt(pemStr)
		if encErr != nil {
			return encErr
		}
		pv := null.From(encPem)
		raw := json.RawMessage(`{}`)
		setter := &models.IamSigningKeySetter{
			Kid:         &kid,
			ProjectID:   &projectID,
			Environment: &env,
			Alg:         ptr("RS256"),
			Use:         ptr("sig"),
			Status:      ptr("active"),
			PrivatePem:  &pv,
			Data:        &raw,
		}
		if _, err := models.IamSigningKeys.Insert(setter).One(ctx, s.db.Bobx()); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	return kid, priv, nil
}

// Sign mints a signed RS256 JWT for project/env with the given claims and TTL.
// iss/aud/sub should be passed as claims by the caller.
func (s *Signer) Sign(ctx context.Context, projectID, env string, claims map[string]any, ttl time.Duration) (string, error) {
	kid, priv, err := s.activeKey(ctx, projectID, env)
	if err != nil {
		return "", err
	}
	now := nowUTC()
	b := jwt.NewBuilder().IssuedAt(now).Expiration(now.Add(ttl)).NotBefore(now)
	for k, v := range claims {
		b = b.Claim(k, v)
	}
	tok, err := b.Build()
	if err != nil {
		return "", err
	}
	key, err := jwk.Import(priv)
	if err != nil {
		return "", err
	}
	if err := key.Set(jwk.KeyIDKey, kid); err != nil {
		return "", err
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		return "", err
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), key))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

// Verify parses and validates a token against the project/env public keys,
// returning its claims. An invalid/expired token maps to domain.ErrInvalidToken.
func (s *Signer) Verify(ctx context.Context, projectID, env, token string) (map[string]any, error) {
	set, err := s.publicSet(ctx, projectID, env)
	if err != nil {
		return nil, err
	}
	tok, err := jwt.Parse([]byte(token), jwt.WithKeySet(set), jwt.WithValidate(true))
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	return tokenClaims(tok)
}

// UnverifiedClaims decodes a JWT's claims WITHOUT verifying the signature. Use
// only where the result is non-authoritative — e.g. routing an idempotent
// revoke by its sid claim. Returns nil on a malformed token.
func (s *Signer) UnverifiedClaims(token string) map[string]any {
	tok, err := jwt.Parse([]byte(token), jwt.WithVerify(false), jwt.WithValidate(false))
	if err != nil {
		return nil
	}
	m, err := tokenClaims(tok)
	if err != nil {
		return nil
	}
	return m
}

// JWKS returns the public JWK set for project/env as a generic map (the
// /.well-known/jwks.json body).
func (s *Signer) JWKS(ctx context.Context, projectID, env string) (map[string]any, error) {
	set, err := s.publicSet(ctx, projectID, env)
	if err != nil {
		return nil, err
	}
	buf, err := json.Marshal(set)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// publicSet builds a jwk.Set of the non-retired public keys (active + inactive,
// so recently-rotated tokens still verify).
func (s *Signer) publicSet(ctx context.Context, projectID, env string) (jwk.Set, error) {
	rows, err := s.keysFor(ctx, projectID, env)
	if err != nil {
		return nil, err
	}
	set := jwk.NewSet()
	for _, r := range rows {
		if r.Status == "retired" {
			continue
		}
		pemStr, ok := r.PrivatePem.Get()
		if !ok || pemStr == "" {
			continue
		}
		decPem, err := s.db.Cipher.Decrypt(pemStr)
		if err != nil {
			continue
		}
		priv, err := parsePrivatePEM(decPem)
		if err != nil {
			continue
		}
		pub, err := jwk.PublicKeyOf(priv)
		if err != nil {
			continue
		}
		if err := pub.Set(jwk.KeyIDKey, r.Kid); err != nil {
			slog.Error("webauthn: failed to set key ID on JWKS public key", "err", err, "kid", r.Kid)
			continue
		}
		if err := pub.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
			slog.Error("webauthn: failed to set algorithm on JWKS public key", "err", err, "kid", r.Kid)
			continue
		}
		if err := pub.Set(jwk.KeyUsageKey, "sig"); err != nil {
			slog.Error("webauthn: failed to set key usage on JWKS public key", "err", err, "kid", r.Kid)
			continue
		}
		if err := set.AddKey(pub); err != nil {
			slog.Error("webauthn: failed to add public key to JWKS set", "err", err, "kid", r.Kid)
			continue
		}
	}
	return set, nil
}

// tokenClaims renders a token's full claim set as a generic map.
func tokenClaims(tok jwt.Token) (map[string]any, error) {
	buf, err := json.Marshal(tok)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func parsePrivatePEM(s string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		return nil, domain.ErrInternal
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func encodePrivatePEM(k *rsa.PrivateKey) string {
	der := x509.MarshalPKCS1PrivateKey(k)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
}

// newRSAKeyPEM generates a fresh RSA-2048 private key and returns its PEM. Used
// by the admin signing-key create/rotate path.
func newRSAKeyPEM() (string, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	return encodePrivatePEM(k), nil
}
