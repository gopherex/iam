package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const defaultJWKSCacheTTL = 5 * time.Minute

var (
	// ErrProjectIDRequired is returned when a local JWKS URL must be derived
	// but no project id was configured.
	ErrProjectIDRequired = errors.New("iam sdk: project id is required for local jwks url")
	// ErrEnvironmentRequired is returned when a local JWKS URL must be derived
	// but no environment was configured.
	ErrEnvironmentRequired = errors.New("iam sdk: environment is required for local jwks url")
	// ErrLocalBaseURLRequired is returned when building a local JWKS URL with
	// no base URL configured.
	ErrLocalBaseURLRequired = errors.New("iam sdk: base url is required for local jwks url")
	// ErrInvalidTokenType, ErrInvalidIssuer, ErrInvalidProject,
	// ErrInvalidEnvironment, ErrInvalidAudience are validateClaims' rejection
	// reasons; VerifyResult.Error carries err.Error() verbatim as the
	// machine-readable code a caller sees.
	ErrInvalidTokenType   = errors.New("invalid_token_type")
	ErrInvalidIssuer      = errors.New("invalid_issuer")
	ErrInvalidProject     = errors.New("invalid_project")
	ErrInvalidEnvironment = errors.New("invalid_environment")
	ErrInvalidAudience    = errors.New("invalid_audience")
)

// LocalConfig configures local JWT verification using IAM's public JWKS.
type LocalConfig struct {
	// BaseURL is the IAM public base URL. It is used to build JWKSURL when
	// JWKSURL is not provided.
	BaseURL string
	// JWKSURL optionally overrides the public JWKS endpoint.
	JWKSURL string
	// ProjectID scopes verification to a single IAM project.
	ProjectID string
	// Environment scopes verification to a single IAM environment.
	Environment string
	// Issuer optionally overrides the expected iss claim. When empty and
	// ProjectID/Environment are set, it defaults to /p/{project_id}/e/{env}.
	Issuer string
	// Audience optionally validates aud/client_id.
	Audience string
	// TokenType validates typ. Empty defaults to access.
	TokenType string
	// CacheTTL controls how long fetched JWKS keys are reused.
	CacheTTL time.Duration
	// HTTPClient optionally overrides the HTTP client used for JWKS fetches.
	HTTPClient *http.Client
}

// LocalVerifier verifies IAM JWTs locally using the project's JWKS.
type LocalVerifier struct {
	jwksURL     string
	projectID   string
	environment string
	issuer      string
	audience    string
	tokenType   string
	cacheTTL    time.Duration
	httpClient  *http.Client

	mu        sync.RWMutex
	keySet    jwk.Set
	expiresAt time.Time
}

// NewLocalVerifier creates a local JWT verifier that does not call IAM token
// verification APIs.
func NewLocalVerifier(config LocalConfig) (*LocalVerifier, error) {
	projectID := strings.TrimSpace(config.ProjectID)
	environment := strings.TrimSpace(config.Environment)

	issuer := strings.TrimSpace(config.Issuer)
	if issuer == "" {
		issuer = issuerFor(config.BaseURL, projectID, environment)
	}

	jwksURL := strings.TrimSpace(config.JWKSURL)
	if jwksURL == "" {
		if projectID == "" {
			return nil, ErrProjectIDRequired
		}

		if environment == "" {
			return nil, ErrEnvironmentRequired
		}

		var err error

		jwksURL, err = buildJWKSURL(config.BaseURL, projectID, environment)
		if err != nil {
			return nil, err
		}
	}

	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultJWKSCacheTTL
	}

	tokenType := strings.TrimSpace(config.TokenType)
	if tokenType == "" {
		tokenType = "access"
	}

	return &LocalVerifier{
		jwksURL:     jwksURL,
		projectID:   projectID,
		environment: environment,
		issuer:      issuer,
		audience:    strings.TrimSpace(config.Audience),
		tokenType:   tokenType,
		cacheTTL:    cacheTTL,
		httpClient:  config.HTTPClient,
	}, nil
}

// Verify verifies token locally against the cached JWKS.
func (v *LocalVerifier) Verify(ctx context.Context, token string) (*VerifyResult, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrMissingToken
	}

	tok, err := v.parse(ctx, token, false)
	if err != nil {
		tok, err = v.parse(ctx, token, true)
	}

	if err != nil {
		return &VerifyResult{ //nolint:nilerr // an invalid token verifies as invalid, not an error
			Valid: false,
			Error: "invalid_token",
		}, nil
	}

	claims, err := tokenClaims(tok)
	if err != nil {
		return nil, err
	}

	if err := v.validateClaims(claims); err != nil {
		return &VerifyResult{ //nolint:nilerr // a claim-validation failure verifies as invalid, not an error
			Valid:  false,
			Error:  err.Error(),
			Claims: claims,
		}, nil
	}

	out := &VerifyResult{
		Valid:  true,
		Claims: claims,
	}
	out.Principal = principalFromClaims(claims)

	return out, nil
}

// Authenticate verifies token and returns a Principal on success.
func (v *LocalVerifier) Authenticate(ctx context.Context, token string) (*Principal, error) {
	res, err := v.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	if !res.Valid {
		if res.Error != "" {
			return nil, fmt.Errorf("%w: %s", ErrInvalidToken, res.Error)
		}

		return nil, ErrInvalidToken
	}

	return &res.Principal, nil
}

// Warm fetches JWKS immediately so startup can fail before serving traffic.
func (v *LocalVerifier) Warm(ctx context.Context) error {
	return v.Refresh(ctx)
}

// Refresh forces JWKS cache refresh.
func (v *LocalVerifier) Refresh(ctx context.Context) error {
	_, err := v.keySetFor(ctx, true)
	return err
}

// Middleware authenticates HTTP requests with local JWT verification.
func (v *LocalVerifier) Middleware(next http.Handler) http.Handler {
	return HTTPMiddleware(v, next)
}

// MiddlewareWithOptions returns configurable HTTP authentication middleware.
func (v *LocalVerifier) MiddlewareWithOptions(opts HTTPMiddlewareOptions) func(http.Handler) http.Handler {
	return HTTPMiddlewareWithOptions(v, opts)
}

func (v *LocalVerifier) parse(ctx context.Context, token string, forceRefresh bool) (jwt.Token, error) {
	set, err := v.keySetFor(ctx, forceRefresh)
	if err != nil {
		return nil, err
	}

	tok, err := jwt.Parse([]byte(token), jwt.WithKeySet(set), jwt.WithValidate(true))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return tok, nil
}

func (v *LocalVerifier) keySetFor(ctx context.Context, forceRefresh bool) (jwk.Set, error) {
	now := time.Now()

	v.mu.RLock()

	if !forceRefresh && v.keySet != nil && now.Before(v.expiresAt) {
		set := v.keySet
		v.mu.RUnlock()

		return set, nil
	}

	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	now = time.Now()
	if !forceRefresh && v.keySet != nil && now.Before(v.expiresAt) {
		return v.keySet, nil
	}

	opts := []jwk.FetchOption(nil)
	if v.httpClient != nil {
		opts = append(opts, jwk.WithHTTPClient(v.httpClient))
	}

	set, err := jwk.Fetch(ctx, v.jwksURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}

	v.keySet = set
	v.expiresAt = now.Add(v.cacheTTL)

	return set, nil
}

func (v *LocalVerifier) validateClaims(claims Claims) error {
	if v.tokenType != "" && claimString(claims, "typ") != v.tokenType {
		return ErrInvalidTokenType
	}

	if v.issuer != "" && claimString(claims, "iss") != v.issuer {
		return ErrInvalidIssuer
	}

	if v.projectID != "" {
		pid := claimString(claims, "pid")
		if pid == "" {
			pid, _ = parseIssuer(claimString(claims, "iss"))
		}

		if pid != v.projectID {
			return ErrInvalidProject
		}
	}

	if v.environment != "" {
		env := claimString(claims, "env")
		if env == "" {
			_, env = parseIssuer(claimString(claims, "iss"))
		}

		if env != v.environment {
			return ErrInvalidEnvironment
		}
	}

	if v.audience != "" && !claimContains(claims, "aud", v.audience) && !claimContains(claims, "client_id", v.audience) {
		return ErrInvalidAudience
	}

	return nil
}

func tokenClaims(tok jwt.Token) (Claims, error) {
	buf, err := json.Marshal(tok)
	if err != nil {
		return nil, fmt.Errorf("marshal token: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(buf, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	return claims, nil
}

func buildJWKSURL(baseURL, projectID, environment string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", ErrLocalBaseURLRequired
	}

	jwksURL, err := url.JoinPath(baseURL, "p", projectID, "e", environment, ".well-known", "jwks.json")
	if err != nil {
		return "", fmt.Errorf("build jwks url: %w", err)
	}

	return jwksURL, nil
}

// issuerFor builds the canonical absolute issuer for a tenant:
// "<baseURL>/p/<projectID>/e/<environment>". IAM issues nothing else — the
// issuer is an absolute URL that prefixes the tenant's discovery document
// (OIDC Discovery 1.0 §3). Returns "" when baseURL is unknown, which leaves the
// issuer check disabled rather than pinning a wrong value.
func issuerFor(baseURL, projectID, environment string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || projectID == "" || environment == "" {
		return ""
	}

	return baseURL + "/p/" + projectID + "/e/" + environment
}

// issuerTenantSegments is the length of the tenant suffix every IAM issuer path
// ends with: "p", <projectID>, "e", <environment>.
const issuerTenantSegments = 4

// parseIssuer recovers the project id and environment from an absolute IAM
// issuer of the form "<base>/p/<projectID>/e/<environment>", returning empty
// strings for anything else. The deployment base is not known here, so the
// tenant path is matched as the tail of the issuer's path; callers only reach
// this after the token's signature has been verified against a pinned JWKS.
func parseIssuer(issuer string) (string, string) {
	u, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || !u.IsAbs() || u.Host == "" {
		return "", ""
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < issuerTenantSegments {
		return "", ""
	}

	tail := parts[len(parts)-issuerTenantSegments:]
	if tail[0] != "p" || tail[2] != "e" || tail[1] == "" || tail[3] == "" {
		return "", ""
	}

	return tail[1], tail[3]
}

func claimContains(claims Claims, key, want string) bool {
	switch v := claims[key].(type) {
	case string:
		if v == want {
			return true
		}

		for _, item := range strings.Fields(v) {
			if item == want {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if item == want {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	}

	return false
}
