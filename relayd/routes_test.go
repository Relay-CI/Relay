package main

import (
	"net/http"
	"net/url"
	"testing"
)

func TestControlRoutesAreSharedAcrossNetworkAndSocketTransports(t *testing.T) {
	s := &Server{}
	networkMux := http.NewServeMux()
	localMux := http.NewServeMux()
	s.registerNetworkControlRoutes(networkMux)
	s.registerLocalControlRoutes(localMux)

	shared := []string{
		"/health",
		"/api/deploys",
		"/api/apps/config",
		"/api/apps/companions",
		"/api/plugins/buildpacks",
		"/api/runtime/logs/targets",
		"/api/sync/start",
		"/api/projects",
		"/api/promotions",
		"/api/webhooks/github",
		"/api/edge/authz",
		"/api/doctor",
		"/api/users",
		"/api/audit",
	}
	for _, path := range shared {
		assertRouteRegistered(t, networkMux, path)
		assertRouteRegistered(t, localMux, path)
	}

	for _, path := range []string{"/api/deploys/rollout-abort", "/api/analytics"} {
		assertRouteRegistered(t, networkMux, path)
		assertExactRouteMissing(t, localMux, path)
	}
}

func assertRouteRegistered(t *testing.T, mux *http.ServeMux, path string) {
	t.Helper()
	_, pattern := mux.Handler(&http.Request{URL: &url.URL{Path: path}})
	if pattern == "" {
		t.Fatalf("expected route %s to be registered", path)
	}
}

func assertExactRouteMissing(t *testing.T, mux *http.ServeMux, path string) {
	t.Helper()
	_, pattern := mux.Handler(&http.Request{URL: &url.URL{Path: path}})
	if pattern == path {
		t.Fatalf("expected exact route %s to remain unavailable", path)
	}
}
