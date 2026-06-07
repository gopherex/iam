//go:build !embed

// Package web serves the admin SPA. The default build does NOT embed the SPA
// assets (so `go build ./...` needs no prior `yarn build`); build the binary
// with `make build` (which runs `go build -tags embed` after building the SPA)
// to embed and serve it.
package web

import "net/http"

// Handler returns a placeholder explaining the SPA was not embedded.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("admin SPA not embedded — build with `make build` (go build -tags embed)\n"))
	})
}
