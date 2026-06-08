// Package sdk is the ergonomic Go SDK for IAM resource servers.
//
// Server applications use this package to verify tokens received from clients.
// The SDK calls IAM's token verification endpoint with the server's own
// service/admin credential and stores the verified principal in request
// contexts for downstream handlers.
package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-faster/jx"

	"github.com/gopherex/iam/internal/oas"
)

var (
	// ErrMissingToken is returned when an incoming request has no bearer token.
	ErrMissingToken = errors.New("iam sdk: missing bearer token")
	// ErrInvalidToken is returned when IAM rejects the incoming bearer token.
	ErrInvalidToken = errors.New("iam sdk: invalid bearer token")
)

// Claims is the decoded claim set returned by IAM token verification.
type Claims map[string]any

// Principal is the resource-server view of an authenticated IAM subject.
type Principal struct {
	Subject     string
	AccountID   string
	UserID      string
	ProjectID   string
	Environment string
	SessionID   string
	ClientID    string
	Scopes      []string
	AAL         int
	AMR         []string
	Claims      Claims
}

// VerifyResult is IAM's token-verification response after decoding raw claims.
type VerifyResult struct {
	Valid     bool
	Error     string
	Claims    Claims
	Principal Principal
}

// Config configures a server-side IAM SDK client.
type Config struct {
	// BaseURL is the IAM API base URL, for example https://iam.example.com.
	BaseURL string
	// Credential is the server's IAM credential used to call verification APIs.
	// In production this is usually a service token or project-admin token.
	Credential string
	// HTTPClient optionally overrides the HTTP client used for IAM API calls.
	HTTPClient *http.Client
}

// Client wraps the generated IAM OpenAPI client with server-side conveniences.
type Client struct {
	oas *oas.Client
}

// Authenticator verifies a bearer token and returns a Principal on success.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*Principal, error)
}

// Warmer can verify that an authenticator's external dependencies are ready.
type Warmer interface {
	Warm(ctx context.Context) error
}

// Refresher forces an authenticator to refresh cached verification state.
type Refresher interface {
	Refresh(ctx context.Context) error
}

// Warm initializes authenticator verification state when supported.
func Warm(ctx context.Context, auth Authenticator) error {
	if w, ok := auth.(Warmer); ok {
		return w.Warm(ctx)
	}
	return nil
}

// Refresh forces authenticator verification state refresh when supported.
func Refresh(ctx context.Context, auth Authenticator) error {
	if r, ok := auth.(Refresher); ok {
		return r.Refresh(ctx)
	}
	return nil
}

// New creates a server-side IAM SDK client.
func New(config Config) (*Client, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return nil, errors.New("iam sdk: base url is required")
	}
	credential := strings.TrimSpace(config.Credential)
	if credential == "" {
		return nil, errors.New("iam sdk: credential is required")
	}
	opts := []oas.ClientOption(nil)
	if config.HTTPClient != nil {
		opts = append(opts, oas.WithClient(config.HTTPClient))
	}
	c, err := oas.NewClient(baseURL, staticSecuritySource{token: credential}, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{oas: c}, nil
}

// Verifier verifies end-user/client tokens received by a resource server.
type Verifier struct {
	client   *Client
	audience string
}

// VerifierOption customizes a Verifier.
type VerifierOption func(*Verifier)

// WithAudience asks IAM to validate the token audience.
func WithAudience(audience string) VerifierOption {
	return func(v *Verifier) {
		v.audience = strings.TrimSpace(audience)
	}
}

// NewVerifier creates a token verifier from Config.
func NewVerifier(config Config, opts ...VerifierOption) (*Verifier, error) {
	c, err := New(config)
	if err != nil {
		return nil, err
	}
	v := &Verifier{client: c}
	for _, opt := range opts {
		if opt != nil {
			opt(v)
		}
	}
	return v, nil
}

// Verify verifies token via IAM /v1/tokens/verify.
func (v *Verifier) Verify(ctx context.Context, token string) (*VerifyResult, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrMissingToken
	}
	req := &oas.PostV1TokensVerifyReq{Token: token}
	if v.audience != "" {
		req.Audience = oas.NewOptString(v.audience)
	}
	res, err := v.client.oas.PostV1TokensVerify(ctx, req)
	if err != nil {
		return nil, err
	}
	claims := Claims{}
	if raw, ok := res.Claims.Get(); ok {
		claims = rawMapToClaims(map[string]jx.Raw(raw))
	}
	out := &VerifyResult{
		Valid:  res.Valid.Or(false),
		Error:  res.Error.Or(""),
		Claims: claims,
	}
	out.Principal = principalFromClaims(claims)
	return out, nil
}

// Introspect verifies token activity via IAM /v1/tokens/introspect.
func (v *Verifier) Introspect(ctx context.Context, token string) (*VerifyResult, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrMissingToken
	}
	res, err := v.client.oas.PostV1TokensIntrospect(ctx, &oas.PostV1TokensIntrospectReq{Token: token})
	if err != nil {
		return nil, err
	}
	claims := rawMapToClaims(map[string]jx.Raw(res.AdditionalProps))
	out := &VerifyResult{
		Valid:  res.Active.Or(false),
		Claims: claims,
	}
	out.Principal = principalFromClaims(claims)
	return out, nil
}

// Authenticate verifies token and returns a Principal on success.
func (v *Verifier) Authenticate(ctx context.Context, token string) (*Principal, error) {
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

// Warm is a no-op for remote verification; IAM is contacted on Verify.
func (v *Verifier) Warm(context.Context) error {
	return nil
}

// Refresh is a no-op for remote verification; IAM holds verification state.
func (v *Verifier) Refresh(context.Context) error {
	return nil
}

// BearerToken extracts an Authorization: Bearer token from an HTTP request.
func BearerToken(r *http.Request) (string, bool) {
	value := r.Header.Get("Authorization")
	scheme, token, ok := strings.Cut(value, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

type contextKey struct{}

// WithPrincipal stores principal in ctx.
func WithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

// PrincipalFrom returns the principal stored by SDK HTTP middleware or gRPC
// interceptors.
func PrincipalFrom(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(contextKey{}).(*Principal)
	return p, ok && p != nil
}

func rawMapToClaims(raw map[string]jx.Raw) Claims {
	claims := make(Claims, len(raw))
	for k, v := range raw {
		var out any
		if err := json.Unmarshal(v, &out); err != nil {
			claims[k] = string(v)
			continue
		}
		claims[k] = out
	}
	return claims
}

func principalFromClaims(claims Claims) Principal {
	subject := claimString(claims, "sub")
	projectID := claimString(claims, "pid")
	environment := claimString(claims, "env")
	if projectID == "" || environment == "" {
		issuerProjectID, issuerEnvironment := parseIssuer(claimString(claims, "iss"))
		if projectID == "" {
			projectID = issuerProjectID
		}
		if environment == "" {
			environment = issuerEnvironment
		}
	}
	return Principal{
		Subject:     subject,
		AccountID:   subject,
		UserID:      subject,
		ProjectID:   projectID,
		Environment: environment,
		SessionID:   claimString(claims, "sid"),
		ClientID:    firstNonEmpty(claimString(claims, "aud"), claimString(claims, "client_id")),
		Scopes:      claimStringSlice(claims, "scope", "scp"),
		AAL:         claimInt(claims, "aal"),
		AMR:         claimStringSlice(claims, "amr"),
		Claims:      claims,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func claimString(claims Claims, key string) string {
	switch v := claims[key].(type) {
	case string:
		return v
	case []any:
		if len(v) == 0 {
			return ""
		}
		s, _ := v[0].(string)
		return s
	default:
		return ""
	}
}

func claimInt(claims Claims, key string) int {
	switch v := claims[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func claimStringSlice(claims Claims, keys ...string) []string {
	for _, key := range keys {
		switch v := claims[key].(type) {
		case string:
			if v == "" {
				continue
			}
			return strings.Fields(v)
		case []string:
			return append([]string(nil), v...)
		case []any:
			out := make([]string, 0, len(v))
			for _, item := range v {
				s, ok := item.(string)
				if ok && s != "" {
					out = append(out, s)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

type staticSecuritySource struct {
	token string
}

func (s staticSecuritySource) AdminToken(context.Context, oas.OperationName, *oas.Client) (oas.AdminToken, error) {
	return oas.AdminToken{Token: s.token}, nil
}

func (s staticSecuritySource) BearerAuth(context.Context, oas.OperationName, *oas.Client) (oas.BearerAuth, error) {
	return oas.BearerAuth{Token: s.token}, nil
}

func (s staticSecuritySource) ClientSecretBasic(context.Context, oas.OperationName, *oas.Client) (oas.ClientSecretBasic, error) {
	return oas.ClientSecretBasic{Username: s.token}, nil
}

func (s staticSecuritySource) MasterKey(context.Context, oas.OperationName, *oas.Client) (oas.MasterKey, error) {
	return oas.MasterKey{Token: s.token}, nil
}

func (s staticSecuritySource) OAuth2(context.Context, oas.OperationName, *oas.Client) (oas.OAuth2, error) {
	return oas.OAuth2{Token: s.token}, nil
}

func (s staticSecuritySource) ScimToken(context.Context, oas.OperationName, *oas.Client) (oas.ScimToken, error) {
	return oas.ScimToken{Token: s.token}, nil
}

func (s staticSecuritySource) ServiceToken(context.Context, oas.OperationName, *oas.Client) (oas.ServiceToken, error) {
	return oas.ServiceToken{Token: s.token}, nil
}
