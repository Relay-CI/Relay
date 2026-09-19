package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRuntimeHTTPReadinessPathRequiresSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready":
			w.WriteHeader(http.StatusNoContent)
		case "/redirect":
			http.Redirect(w, r, "/ready", http.StatusFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	parsed, _ := url.Parse(server.URL)
	if !runtimeHTTPReady(parsed.Host) {
		t.Fatal("legacy process-level probe changed")
	}
	if !runtimeHTTPReadyOnPath(parsed.Host, "/ready") {
		t.Fatal("healthy readiness path rejected")
	}
	if runtimeHTTPReadyOnPath(parsed.Host, "/broken") {
		t.Fatal("HTTP 500 passed application readiness")
	}
	if runtimeHTTPReadyOnPath(parsed.Host, "/redirect") {
		t.Fatal("redirect passed application readiness")
	}
}
