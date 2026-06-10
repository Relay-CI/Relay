package main

import (
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestUIAssetHandlerServesPrecompressed(t *testing.T) {
	root := fstest.MapFS{
		"_next/static/chunks/app.js":    {Data: []byte("console.log('full source')")},
		"_next/static/chunks/app.js.gz": {Data: []byte("gzip-bytes")},
		"_next/static/chunks/app.js.br": {Data: []byte("br-bytes")},
		"index.html":                    {Data: []byte("<html></html>")},
	}
	h := uiAssetHandler(root)

	req := httptest.NewRequest("GET", "/_next/static/chunks/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Content-Encoding"); got != "br" {
		t.Fatalf("Content-Encoding = %q, want br", got)
	}
	if rec.Body.String() != "br-bytes" {
		t.Fatalf("expected brotli payload, got %q", rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("hashed assets must be immutable, got %q", got)
	}
	if got := rec.Header().Get("Content-Type"); got == "" || got == "application/gzip" {
		t.Fatalf("Content-Type must reflect the original asset, got %q", got)
	}

	// gzip-only client gets the gzip variant.
	req = httptest.NewRequest("GET", "/_next/static/chunks/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}

	// No Accept-Encoding: identity bytes.
	req = httptest.NewRequest("GET", "/_next/static/chunks/app.js", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("identity response must have no Content-Encoding, got %q", got)
	}
	if rec.Body.String() != "console.log('full source')" {
		t.Fatalf("expected original payload, got %q", rec.Body.String())
	}

	// HTML revalidates instead of caching forever.
	req = httptest.NewRequest("GET", "/index.html", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("html Cache-Control = %q, want no-cache", got)
	}
}
