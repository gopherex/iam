package sdk

import (
	"context"
	"fmt"
	"net/http"
)

// HybridVerifier verifies locally first and falls back to remote verification
// when the local path cannot authenticate a token.
type HybridVerifier struct {
	local  *LocalVerifier
	remote *Verifier
}

// NewHybridVerifier wires local and remote verification paths together.
func NewHybridVerifier(local *LocalVerifier, remote *Verifier) *HybridVerifier {
	return &HybridVerifier{
		local:  local,
		remote: remote,
	}
}

// Verify tries local JWKS verification first, then falls back to IAM remote
// verification when local verification fails with a generic invalid_token.
func (v *HybridVerifier) Verify(ctx context.Context, token string) (*VerifyResult, error) {
	res, err := v.local.Verify(ctx, token)
	if err == nil && res.Valid {
		return res, nil
	}
	if err == nil && res.Error != "" && res.Error != "invalid_token" {
		return res, nil
	}
	return v.remote.Verify(ctx, token)
}

// Authenticate verifies token and returns a Principal on success.
func (v *HybridVerifier) Authenticate(ctx context.Context, token string) (*Principal, error) {
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

// Warm fetches local JWKS immediately. Remote verification stays lazy.
func (v *HybridVerifier) Warm(ctx context.Context) error {
	return v.local.Warm(ctx)
}

// Refresh forces local JWKS refresh.
func (v *HybridVerifier) Refresh(ctx context.Context) error {
	return v.local.Refresh(ctx)
}

// Middleware authenticates HTTP requests with hybrid verification.
func (v *HybridVerifier) Middleware(next http.Handler) http.Handler {
	return HTTPMiddleware(v, next)
}

// MiddlewareWithOptions returns configurable HTTP authentication middleware.
func (v *HybridVerifier) MiddlewareWithOptions(opts HTTPMiddlewareOptions) func(http.Handler) http.Handler {
	return HTTPMiddlewareWithOptions(v, opts)
}
