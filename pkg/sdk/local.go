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
	if issuer == "" && projectID != "" && environment != "" {
		issuer = issuerFor(projectID, environment)
	}
	jwksURL := strings.TrimSpace(config.JWKSURL)
	if jwksURL == "" {
		if projectID == "" {
			return nil, errors.New("iam sdk: project id is required for local jwks url")
		}
		if environment == "" {
			return nil, errors.New("iam sdk: environment is required for local jwks url")
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
		return &VerifyResult{
			Valid: false,
			Error: "invalid_token",
		}, nil
	}
	claims, err := tokenClaims(tok)
	if err != nil {
		return nil, err
	}
	if err := v.validateClaims(claims); err != nil {
		return &VerifyResult{
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
	return jwt.Parse([]byte(token), jwt.WithKeySet(set), jwt.WithValidate(true))
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
		return nil, err
	}
	v.keySet = set
	v.expiresAt = now.Add(v.cacheTTL)
	return set, nil
}

func (v *LocalVerifier) validateClaims(claims Claims) error {
	if v.tokenType != "" && claimString(claims, "typ") != v.tokenType {
		return errors.New("invalid_token_type")
	}
	if v.issuer != "" && claimString(claims, "iss") != v.issuer {
		return errors.New("invalid_issuer")
	}
	if v.projectID != "" {
		pid := claimString(claims, "pid")
		if pid == "" {
			pid, _ = parseIssuer(claimString(claims, "iss"))
		}
		if pid != v.projectID {
			return errors.New("invalid_project")
		}
	}
	if v.environment != "" {
		env := claimString(claims, "env")
		if env == "" {
			_, env = parseIssuer(claimString(claims, "iss"))
		}
		if env != v.environment {
			return errors.New("invalid_environment")
		}
	}
	if v.audience != "" && !claimContains(claims, "aud", v.audience) && !claimContains(claims, "client_id", v.audience) {
		return errors.New("invalid_audience")
	}
	return nil
}

func tokenClaims(tok jwt.Token) (Claims, error) {
	buf, err := json.Marshal(tok)
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(buf, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func buildJWKSURL(baseURL, projectID, environment string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", errors.New("iam sdk: base url is required for local jwks url")
	}
	return url.JoinPath(baseURL, "p", projectID, "e", environment, ".well-known", "jwks.json")
}

func issuerFor(projectID, environment string) string {
	return "/p/" + projectID + "/e/" + environment
}

func parseIssuer(issuer string) (projectID, environment string) {
	parts := strings.Split(issuer, "/")
	if len(parts) == 5 && parts[0] == "" && parts[1] == "p" && parts[3] == "e" {
		return parts[2], parts[4]
	}
	return "", ""
}

func claimContains(claims Claims, key string, want string) bool {
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
