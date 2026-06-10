// ui_assets.go — serving for the embedded dashboard: long-lived caching for
// content-hashed Next.js assets and precompressed (.br/.gz) variants emitted
// by the dashboard build, so the ~800 KB of JS arrives as ~200 KB once and is
// then cached by the browser entirely.
package main

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

var compressibleUIExt = map[string]bool{
	".js":   true,
	".css":  true,
	".html": true,
	".svg":  true,
	".json": true,
	".txt":  true,
	".xml":  true,
	".map":  true,
}

func uiAssetHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		ext := path.Ext(p)

		if strings.HasPrefix(p, "_next/static/") {
			// Everything under _next/static is content-hashed by Next.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// HTML and other entry points must revalidate so dashboard
			// updates take effect right after a relayd upgrade.
			w.Header().Set("Cache-Control", "no-cache")
		}

		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && compressibleUIExt[ext] {
			w.Header().Add("Vary", "Accept-Encoding")
			accept := r.Header.Get("Accept-Encoding")
			for _, cand := range []struct{ enc, suffix string }{{"br", ".br"}, {"gzip", ".gz"}} {
				if !strings.Contains(accept, cand.enc) {
					continue
				}
				compressed := p + cand.suffix
				f, err := root.Open(compressed)
				if err != nil {
					continue
				}
				f.Close()
				if ct := mime.TypeByExtension(ext); ct != "" {
					w.Header().Set("Content-Type", ct)
				}
				w.Header().Set("Content-Encoding", cand.enc)
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/" + compressed
				fileServer.ServeHTTP(w, r2)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
