package domain

import "context"

// RequestMeta is the per-request device/network context captured by transport
// middleware and read by the session-minting path so sessions carry the
// originating device (for self-managed-session UIs and theft detection).
type RequestMeta struct {
	IP          string
	UserAgent   string
	Fingerprint string
}

type requestMetaKey struct{}

// WithRequestMeta returns a context carrying the request metadata.
func WithRequestMeta(ctx context.Context, m RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaKey{}, m)
}

// RequestMetaFromContext returns the request metadata, or the zero value when
// none was set (non-HTTP callers, internal mints).
func RequestMetaFromContext(ctx context.Context) RequestMeta {
	m, _ := ctx.Value(requestMetaKey{}).(RequestMeta)
	return m
}
