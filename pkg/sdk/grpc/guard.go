package grpc

// Claim guards for gRPC mirror the HTTP guards in pkg/sdk: they enforce scope
// grants and authenticator-strength (AAL) that IAM minted onto the token. They
// are NOT ReBAC authorization. Each guard reads the Principal placed in context
// by UnaryServerInterceptor/StreamServerInterceptor; chain it after the auth
// interceptor. Missing principal -> Unauthenticated; insufficient claim ->
// PermissionDenied.

import (
	"context"

	"github.com/gopherex/iam/pkg/sdk"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func check(ctx context.Context, ok func(*sdk.Principal) bool) error {
	principal, present := sdk.PrincipalFrom(ctx)
	if !present {
		return status.Error(codes.Unauthenticated, "missing principal")
	}
	if !ok(principal) {
		return status.Error(codes.PermissionDenied, "insufficient claims")
	}
	return nil
}

func unaryGuard(ok func(*sdk.Principal) bool) googlegrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *googlegrpc.UnaryServerInfo, handler googlegrpc.UnaryHandler) (any, error) {
		if err := check(ctx, ok); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func streamGuard(ok func(*sdk.Principal) bool) googlegrpc.StreamServerInterceptor {
	return func(srv any, stream googlegrpc.ServerStream, _ *googlegrpc.StreamServerInfo, handler googlegrpc.StreamHandler) error {
		if err := check(stream.Context(), ok); err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

// RequireScopesUnary admits a unary call only when the principal holds every
// listed scope. Chain after UnaryServerInterceptor.
func RequireScopesUnary(scopes ...string) googlegrpc.UnaryServerInterceptor {
	return unaryGuard(func(p *sdk.Principal) bool { return p.HasAllScopes(scopes...) })
}

// RequireScopesStream is the streaming counterpart of RequireScopesUnary.
func RequireScopesStream(scopes ...string) googlegrpc.StreamServerInterceptor {
	return streamGuard(func(p *sdk.Principal) bool { return p.HasAllScopes(scopes...) })
}

// RequireAnyScopeUnary admits a unary call when the principal holds at least one
// of the listed scopes.
func RequireAnyScopeUnary(scopes ...string) googlegrpc.UnaryServerInterceptor {
	return unaryGuard(func(p *sdk.Principal) bool { return p.HasAnyScope(scopes...) })
}

// RequireAnyScopeStream is the streaming counterpart of RequireAnyScopeUnary.
func RequireAnyScopeStream(scopes ...string) googlegrpc.StreamServerInterceptor {
	return streamGuard(func(p *sdk.Principal) bool { return p.HasAnyScope(scopes...) })
}

// RequireAALUnary admits a unary call only when the principal's assurance level
// is at least min (e.g. RequireAALUnary(2) demands a step-up/MFA session).
func RequireAALUnary(min int) googlegrpc.UnaryServerInterceptor {
	return unaryGuard(func(p *sdk.Principal) bool { return p.MeetsAAL(min) })
}

// RequireAALStream is the streaming counterpart of RequireAALUnary.
func RequireAALStream(min int) googlegrpc.StreamServerInterceptor {
	return streamGuard(func(p *sdk.Principal) bool { return p.MeetsAAL(min) })
}
