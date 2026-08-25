package sdk

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrincipalClaimChecks(t *testing.T) {
	p := &Principal{Scopes: []string{"billing:read", "billing:write"}, AAL: 2, AMR: []string{"pwd", "otp"}}

	if !p.HasScope("billing:read") || p.HasScope("admin") {
		t.Fatal("HasScope wrong")
	}

	if !p.HasAllScopes("billing:read", "billing:write") || p.HasAllScopes("billing:read", "admin") {
		t.Fatal("HasAllScopes wrong")
	}

	if !p.HasAnyScope("admin", "billing:read") || p.HasAnyScope("admin", "root") {
		t.Fatal("HasAnyScope wrong")
	}

	if !p.HasAnyScope() { // no constraint
		t.Fatal("HasAnyScope() empty should be true")
	}

	if !p.HasAMR("otp") || p.HasAMR("webauthn") {
		t.Fatal("HasAMR wrong")
	}

	if !p.MeetsAAL(2) || p.MeetsAAL(3) {
		t.Fatal("MeetsAAL wrong")
	}

	var nilp *Principal
	if nilp.HasScope("x") || nilp.MeetsAAL(1) || nilp.HasAMR("x") {
		t.Fatal("nil principal must fail closed")
	}
}

// chain wraps a 200-OK handler with an auth stub that injects principal, then
// the guard under test — mirroring real middleware order.
func guardChain(principal *Principal, gd func(http.Handler) http.Handler) http.Handler {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	authStub := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if principal != nil {
				ctx = WithPrincipal(ctx, principal)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	return authStub(gd(ok))
}

func status(t *testing.T, h http.Handler) int {
	t.Helper()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	return rec.Code
}

func TestRequireScopesHTTP(t *testing.T) {
	p := &Principal{Scopes: []string{"billing:read"}}

	if c := status(t, guardChain(p, RequireScopes([]string{"billing:read"}))); c != http.StatusOK {
		t.Fatalf("granted scope: want 200, got %d", c)
	}

	if c := status(t, guardChain(p, RequireScopes([]string{"billing:write"}))); c != http.StatusForbidden {
		t.Fatalf("missing scope: want 403, got %d", c)
	}
	// No principal in context -> 401, not 403.
	if c := status(t, guardChain(nil, RequireScopes([]string{"billing:read"}))); c != http.StatusUnauthorized {
		t.Fatalf("no principal: want 401, got %d", c)
	}
}

func TestRequireAnyScopeHTTP(t *testing.T) {
	p := &Principal{Scopes: []string{"billing:read"}}
	if c := status(t, guardChain(p, RequireAnyScope([]string{"admin", "billing:read"}))); c != http.StatusOK {
		t.Fatalf("any-scope match: want 200, got %d", c)
	}

	if c := status(t, guardChain(p, RequireAnyScope([]string{"admin", "root"}))); c != http.StatusForbidden {
		t.Fatalf("any-scope miss: want 403, got %d", c)
	}
}

func TestRequireAALHTTP(t *testing.T) {
	if c := status(t, guardChain(&Principal{AAL: 2}, RequireAAL(2))); c != http.StatusOK {
		t.Fatalf("aal met: want 200, got %d", c)
	}

	if c := status(t, guardChain(&Principal{AAL: 1}, RequireAAL(2))); c != http.StatusForbidden {
		t.Fatalf("aal short: want 403, got %d", c)
	}
}

func TestPrincipalGroups(t *testing.T) {
	t.Parallel()

	p := &Principal{Groups: []string{"ops", "platform:admin"}}

	if !p.InGroup("ops") || !p.InGroup("platform:admin") {
		t.Fatal("InGroup missed a role the principal holds")
	}

	if p.InGroup("viewer") || p.InGroup("") {
		t.Fatal("InGroup matched a role the principal does not hold")
	}

	if !p.InAnyGroup("viewer", "ops") {
		t.Fatal("InAnyGroup should match on any one role")
	}

	if p.InAnyGroup("viewer", "auditor") {
		t.Fatal("InAnyGroup matched with no overlap")
	}

	// No constraint means no constraint.
	if !p.InAnyGroup() {
		t.Fatal("InAnyGroup with no arguments should not constrain")
	}

	// A nil principal is unauthenticated, not privileged.
	var nilp *Principal
	if nilp.InGroup("ops") || nilp.InAnyGroup("ops") {
		t.Fatal("a nil principal must hold no roles")
	}
}

func TestPrincipalGroupsFromClaims(t *testing.T) {
	t.Parallel()

	p := principalFromClaims(Claims{
		"sub":    "acct_1",
		"pid":    "proj_1",
		"groups": []any{"ops", "viewer"},
	})

	if !p.InGroup("ops") || !p.InGroup("viewer") {
		t.Fatalf("groups claim was not surfaced: %+v", p.Groups)
	}
}
