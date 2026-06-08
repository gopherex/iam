// Package grpc contains IAM authentication interceptors for gRPC servers.
package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/gopherex/iam/pkg/sdk"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TokenExtractor extracts a bearer token from an incoming gRPC context.
type TokenExtractor func(context.Context) (string, bool)

type options struct {
	tokenExtractor TokenExtractor
}

// Option customizes gRPC interceptors.
type Option func(*options)

// WithTokenExtractor overrides the default authorization metadata extractor.
func WithTokenExtractor(extractor TokenExtractor) Option {
	return func(o *options) {
		o.tokenExtractor = extractor
	}
}

// BearerToken extracts authorization: Bearer token from incoming metadata.
func BearerToken(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	for _, value := range md.Get("authorization") {
		scheme, token, ok := strings.Cut(value, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") {
			continue
		}
		token = strings.TrimSpace(token)
		return token, token != ""
	}
	return "", false
}

// UnaryServerInterceptor authenticates unary gRPC calls and stores Principal in
// the handler context.
func UnaryServerInterceptor(auth sdk.Authenticator, opts ...Option) googlegrpc.UnaryServerInterceptor {
	cfg := applyOptions(opts)
	return func(ctx context.Context, req any, info *googlegrpc.UnaryServerInfo, handler googlegrpc.UnaryHandler) (any, error) {
		principal, err := authenticate(ctx, auth, cfg.tokenExtractor)
		if err != nil {
			return nil, err
		}
		_ = info
		return handler(sdk.WithPrincipal(ctx, principal), req)
	}
}

// StreamServerInterceptor authenticates streaming gRPC calls and stores
// Principal in the stream context.
func StreamServerInterceptor(auth sdk.Authenticator, opts ...Option) googlegrpc.StreamServerInterceptor {
	cfg := applyOptions(opts)
	return func(srv any, stream googlegrpc.ServerStream, info *googlegrpc.StreamServerInfo, handler googlegrpc.StreamHandler) error {
		principal, err := authenticate(stream.Context(), auth, cfg.tokenExtractor)
		if err != nil {
			return err
		}
		_ = info
		return handler(srv, wrappedServerStream{
			ServerStream: stream,
			ctx:          sdk.WithPrincipal(stream.Context(), principal),
		})
	}
}

func applyOptions(opts []Option) options {
	cfg := options{tokenExtractor: BearerToken}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.tokenExtractor == nil {
		cfg.tokenExtractor = BearerToken
	}
	return cfg
}

func authenticate(ctx context.Context, auth sdk.Authenticator, extractor TokenExtractor) (*sdk.Principal, error) {
	token, ok := extractor(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing bearer token")
	}
	principal, err := auth.Authenticate(ctx, token)
	if err != nil {
		if errors.Is(err, sdk.ErrMissingToken) || errors.Is(err, sdk.ErrInvalidToken) {
			return nil, status.Error(codes.Unauthenticated, "invalid bearer token")
		}
		return nil, status.Error(codes.Internal, "iam authentication failed")
	}
	return principal, nil
}

type wrappedServerStream struct {
	googlegrpc.ServerStream
	ctx context.Context
}

func (s wrappedServerStream) Context() context.Context {
	return s.ctx
}
