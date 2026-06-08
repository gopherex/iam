package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gopherex/iam/pkg/sdk"
	sdkgrpc "github.com/gopherex/iam/pkg/sdk/grpc"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptorStoresPrincipal(t *testing.T) {
	auth := &fakeAuth{principal: &sdk.Principal{ProjectID: "proj_123"}}
	interceptor := sdkgrpc.UnaryServerInterceptor(auth)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid-token"))

	resp, err := interceptor(ctx, nil, &googlegrpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, func(ctx context.Context, req any) (any, error) {
		principal, ok := sdk.PrincipalFrom(ctx)
		if !ok {
			t.Fatal("principal missing from context")
		}
		return principal.ProjectID, nil
	})
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
	if resp != "proj_123" {
		t.Fatalf("resp = %#v, want proj_123", resp)
	}
	if auth.token != "valid-token" {
		t.Fatalf("auth token = %q, want valid-token", auth.token)
	}
}

func TestUnaryServerInterceptorRejectsMissingToken(t *testing.T) {
	interceptor := sdkgrpc.UnaryServerInterceptor(&fakeAuth{})
	_, err := interceptor(context.Background(), nil, &googlegrpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not run")
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status = %v, want Unauthenticated", status.Code(err))
	}
}

func TestStreamServerInterceptorStoresPrincipal(t *testing.T) {
	auth := &fakeAuth{principal: &sdk.Principal{ProjectID: "proj_123"}}
	interceptor := sdkgrpc.StreamServerInterceptor(auth)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid-token"))

	err := interceptor(nil, fakeServerStream{ctx: ctx}, &googlegrpc.StreamServerInfo{FullMethod: "/test.Service/Stream"}, func(srv any, stream googlegrpc.ServerStream) error {
		principal, ok := sdk.PrincipalFrom(stream.Context())
		if !ok {
			t.Fatal("principal missing from context")
		}
		if principal.ProjectID != "proj_123" {
			t.Fatalf("project = %q, want proj_123", principal.ProjectID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
	if auth.token != "valid-token" {
		t.Fatalf("auth token = %q, want valid-token", auth.token)
	}
}

type fakeAuth struct {
	principal *sdk.Principal
	token     string
}

func (a *fakeAuth) Authenticate(_ context.Context, token string) (*sdk.Principal, error) {
	a.token = token
	if a.principal == nil {
		return nil, sdk.ErrInvalidToken
	}
	if token == "" {
		return nil, sdk.ErrMissingToken
	}
	if token != "valid-token" {
		return nil, sdk.ErrInvalidToken
	}
	return a.principal, nil
}

var errNotImplemented = errors.New("not implemented")

type fakeServerStream struct {
	googlegrpc.ServerStream
	ctx context.Context
}

func (s fakeServerStream) Context() context.Context {
	return s.ctx
}

func (s fakeServerStream) SetHeader(metadata.MD) error {
	return nil
}

func (s fakeServerStream) SendHeader(metadata.MD) error {
	return nil
}

func (s fakeServerStream) SetTrailer(metadata.MD) {
}

func (s fakeServerStream) SendMsg(any) error {
	return errNotImplemented
}

func (s fakeServerStream) RecvMsg(any) error {
	return errNotImplemented
}
