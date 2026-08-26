package grpc

import (
	"context"
	"testing"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gopherex/iam/pkg/sdk"
)

func ctxWith(p *sdk.Principal) context.Context {
	ctx := context.Background()
	if p != nil {
		ctx = sdk.WithPrincipal(ctx, p)
	}

	return ctx
}

func invokeUnary(ctx context.Context, interceptor googlegrpc.UnaryServerInterceptor) (bool, error) {
	called := false
	_, err := interceptor(ctx, nil, &googlegrpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})

	return called, err
}

func wantCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if status.Code(err) != want {
		t.Fatalf("want code %v, got %v (err=%v)", want, status.Code(err), err)
	}
}

func TestRequireScopesUnary(t *testing.T) {
	t.Parallel()

	p := &sdk.Principal{Scopes: []string{"billing:read"}}

	called, err := invokeUnary(ctxWith(p), RequireScopesUnary("billing:read"))
	if !called || err != nil {
		t.Fatalf("granted: called=%v err=%v", called, err)
	}

	called, err = invokeUnary(ctxWith(p), RequireScopesUnary("billing:write"))
	if called {
		t.Fatal("missing scope must not reach handler")
	}

	wantCode(t, err, codes.PermissionDenied)

	called, err = invokeUnary(ctxWith(nil), RequireScopesUnary("billing:read"))
	if called {
		t.Fatal("no principal must not reach handler")
	}

	wantCode(t, err, codes.Unauthenticated)
}

func TestRequireAALUnary(t *testing.T) {
	t.Parallel()

	called, err := invokeUnary(ctxWith(&sdk.Principal{AAL: 2}), RequireAALUnary(2))
	if !called || err != nil {
		t.Fatalf("aal met: called=%v err=%v", called, err)
	}

	called, err = invokeUnary(ctxWith(&sdk.Principal{AAL: 1}), RequireAALUnary(2))
	if called {
		t.Fatal("aal short must not reach handler")
	}

	wantCode(t, err, codes.PermissionDenied)
}

func TestRequireAnyScopeUnary(t *testing.T) {
	t.Parallel()

	p := &sdk.Principal{Scopes: []string{"billing:read"}}
	if called, err := invokeUnary(ctxWith(p), RequireAnyScopeUnary("admin", "billing:read")); !called || err != nil {
		t.Fatalf("any match: called=%v err=%v", called, err)
	}

	if called, err := invokeUnary(ctxWith(p), RequireAnyScopeUnary("admin", "root")); called || status.Code(err) != codes.PermissionDenied {
		t.Fatalf("any miss: called=%v err=%v", called, err)
	}
}
