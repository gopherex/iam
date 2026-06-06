package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gopherex/iam/internal/domain"
)

// fakeCsrf accepts the token "good" for client "c1"; everything else fails.
type fakeCsrf struct{ calls int }

func (f *fakeCsrf) VerifyCsrfToken(_ context.Context, clientID, token string) error {
	f.calls++
	if clientID == "c1" && token == "good" {
		return nil
	}
	return domain.ErrInvalidCsrf
}

func sessionCookie() *http.Cookie {
	return &http.Cookie{Name: SessionCookieName, Value: "x"}
}

func TestCSRFMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		authHeader bool
		cookie     bool
		clientID   string
		csrf       string
		wantStatus int
		wantNext   bool
		wantVerify bool
	}{
		{name: "safe GET passes", method: http.MethodGet, cookie: true, wantStatus: 200, wantNext: true},
		{name: "bearer request passes", method: http.MethodPost, authHeader: true, cookie: true, wantStatus: 200, wantNext: true},
		{name: "no session cookie passes", method: http.MethodPost, wantStatus: 200, wantNext: true},
		{name: "cookie + missing headers rejected", method: http.MethodPost, cookie: true, wantStatus: 403, wantNext: false},
		{name: "cookie + valid token passes", method: http.MethodPost, cookie: true, clientID: "c1", csrf: "good", wantStatus: 200, wantNext: true, wantVerify: true},
		{name: "cookie + bad token rejected", method: http.MethodPost, cookie: true, clientID: "c1", csrf: "bad", wantStatus: 403, wantNext: false, wantVerify: true},
		{name: "cookie + wrong client rejected", method: http.MethodPost, cookie: true, clientID: "c2", csrf: "good", wantStatus: 403, wantNext: false, wantVerify: true},
		{name: "safe HEAD passes", method: http.MethodHead, cookie: true, wantStatus: 200, wantNext: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCsrf{}
			nextCalled := false
			h := CSRFMiddleware(f)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tc.method, "/v1/anything", nil)
			if tc.authHeader {
				req.Header.Set("Authorization", "Bearer tok")
			}
			if tc.cookie {
				req.AddCookie(sessionCookie())
			}
			if tc.clientID != "" {
				req.Header.Set("X-Client-ID", tc.clientID)
			}
			if tc.csrf != "" {
				req.Header.Set("X-CSRF-Token", tc.csrf)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if nextCalled != tc.wantNext {
				t.Errorf("next called = %v, want %v", nextCalled, tc.wantNext)
			}
			if (f.calls > 0) != tc.wantVerify {
				t.Errorf("verify called = %v, want %v", f.calls > 0, tc.wantVerify)
			}
		})
	}
}
