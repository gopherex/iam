package api

import (
	"strings"
	"testing"
	"time"
)

func TestSessionCookies(t *testing.T) {
	cookies := SessionCookies("acc-tok", "ref-tok", time.Hour, 24*time.Hour)
	if len(cookies) != 2 {
		t.Fatalf("want 2 cookies, got %d", len(cookies))
	}
	access, refresh := cookies[0], cookies[1]

	if !strings.HasPrefix(access, SessionCookieName+"=acc-tok") {
		t.Errorf("access cookie: %q", access)
	}
	if !strings.HasPrefix(refresh, RefreshCookieName+"=ref-tok") {
		t.Errorf("refresh cookie: %q", refresh)
	}
	// access on every path; refresh scoped to the refresh endpoint.
	if !strings.Contains(access, "Path=/;") {
		t.Errorf("access path: %q", access)
	}
	if !strings.Contains(refresh, "Path=/v1/auth/token/refresh") {
		t.Errorf("refresh path: %q", refresh)
	}
	for _, c := range cookies {
		if !strings.Contains(c, "HttpOnly") || !strings.Contains(c, "Secure") || !strings.Contains(c, "SameSite=Lax") {
			t.Errorf("cookie missing security attrs: %q", c)
		}
	}
}
