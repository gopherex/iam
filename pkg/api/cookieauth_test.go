package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookieAuthMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		cookie     string
		wantAuth   string
	}{
		{name: "cookie promoted to bearer", cookie: "tok123", wantAuth: "Bearer tok123"},
		{name: "existing Authorization untouched", authHeader: "Bearer real", cookie: "tok123", wantAuth: "Bearer real"},
		{name: "no cookie no header", wantAuth: ""},
		{name: "empty cookie ignored", cookie: "", wantAuth: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var seenAuth string
			h := CookieAuthMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				seenAuth = r.Header.Get("Authorization")
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: tc.cookie})
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if seenAuth != tc.wantAuth {
				t.Errorf("Authorization = %q, want %q", seenAuth, tc.wantAuth)
			}
		})
	}
}
