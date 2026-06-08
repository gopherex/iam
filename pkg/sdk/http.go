package sdk

import (
	"encoding/json"
	"net/http"
)

// HTTPTokenExtractor extracts an IAM token from an HTTP request.
type HTTPTokenExtractor func(*http.Request) (string, bool)

// HTTPErrorHandler handles middleware authentication failures.
type HTTPErrorHandler func(http.ResponseWriter, *http.Request, error)

// HTTPMiddlewareOptions customizes HTTP middleware behavior.
type HTTPMiddlewareOptions struct {
	TokenExtractor HTTPTokenExtractor
	ErrorHandler   HTTPErrorHandler
}

// Middleware authenticates requests and stores Principal in request context.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return HTTPMiddleware(v, next)
}

// MiddlewareWithOptions returns configurable HTTP authentication middleware.
func (v *Verifier) MiddlewareWithOptions(opts HTTPMiddlewareOptions) func(http.Handler) http.Handler {
	return HTTPMiddlewareWithOptions(v, opts)
}

// HTTPMiddleware authenticates requests and stores Principal in request context.
func HTTPMiddleware(auth Authenticator, next http.Handler) http.Handler {
	return HTTPMiddlewareWithOptions(auth, HTTPMiddlewareOptions{})(next)
}

// HTTPMiddlewareWithOptions returns configurable HTTP authentication middleware.
func HTTPMiddlewareWithOptions(auth Authenticator, opts HTTPMiddlewareOptions) func(http.Handler) http.Handler {
	extract := opts.TokenExtractor
	if extract == nil {
		extract = BearerToken
	}
	handleError := opts.ErrorHandler
	if handleError == nil {
		handleError = defaultHTTPErrorHandler
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extract(r)
			if !ok {
				handleError(w, r, ErrMissingToken)
				return
			}
			principal, err := auth.Authenticate(r.Context(), token)
			if err != nil {
				handleError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func defaultHTTPErrorHandler(w http.ResponseWriter, _ *http.Request, _ error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="iam"`)
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
