//go:build embed

package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// dist holds the built SPA assets (web/dist), embedded into the binary.
// Produced by `yarn build`; this file only compiles under `-tags embed`.
//
//go:embed all:dist
var dist embed.FS

// Handler serves the embedded admin SPA: real assets are served directly
// (immutable, content-hashed under /assets), and any unknown path falls back to
// index.html so client-side routes (/projects/…) resolve.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, statErr := fs.Stat(sub, p); statErr != nil {
			// Not a real asset — serve the SPA shell for client-side routing.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}
