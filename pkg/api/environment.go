package api

import (
	"context"
	"net/http"
)

// EnvironmentHeader is the request header that selects the project environment
// (live / staging / …) a token is minted in. It mirrors the X-Environment OpenAPI
// parameter; the middleware lifts it into the request context so the persistence
// layer can pick the right signing keys without threading it through every port.
const EnvironmentHeader = "X-Environment"

type envCtxKey struct{}

// WithEnvironment stores the requested environment in ctx.
func WithEnvironment(ctx context.Context, env string) context.Context {
	return context.WithValue(ctx, envCtxKey{}, env)
}

// EnvironmentFromContext returns the requested environment, or "" when unset
// (callers fall back to their default environment).
func EnvironmentFromContext(ctx context.Context) string {
	e, _ := ctx.Value(envCtxKey{}).(string)
	return e
}

// EnvironmentMiddleware lifts the X-Environment header into the request context.
// A missing header leaves the context unset (default environment applies). The
// value is validated against the project's environments at mint time, so an
// unknown environment here is harmless until it is actually used.
func EnvironmentMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if env := r.Header.Get(EnvironmentHeader); env != "" {
			r = r.WithContext(WithEnvironment(r.Context(), env))
		}
		next.ServeHTTP(w, r)
	})
}
