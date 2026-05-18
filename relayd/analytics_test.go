package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnalyticsHostsForSelectionIncludesPortModeLaneHosts(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.saveAppState(&AppState{
		App:      "demo",
		Env:      EnvPreview,
		Branch:   "feature",
		RepoURL:  "https://example.com/demo.git",
		Mode:     "port",
		HostPort: 43123,
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	hosts, err := s.analyticsHostsForSelection("demo", EnvPreview, "feature")
	if err != nil {
		t.Fatalf("analyticsHostsForSelection: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected localhost and 127.0.0.1 host candidates, got %v", hosts)
	}
	if hosts[0] != "127.0.0.1:43123" || hosts[1] != "localhost:43123" {
		t.Fatalf("unexpected host candidates: %v", hosts)
	}
}

func TestHandleAnalyticsFiltersToSelectedLane(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.saveAppState(&AppState{
		App:      "demo",
		Env:      EnvPreview,
		Branch:   "feature",
		RepoURL:  "https://example.com/demo.git",
		Mode:     "port",
		HostPort: 43123,
	}); err != nil {
		t.Fatalf("save preview app state: %v", err)
	}
	if err := s.saveAppState(&AppState{
		App:        "demo",
		Env:        EnvProd,
		Branch:     "main",
		RepoURL:    "https://example.com/demo.git",
		Mode:       "edge",
		PublicHost: "demo.example.com",
	}); err != nil {
		t.Fatalf("save prod app state: %v", err)
	}
	now := time.Now().Unix()
	if _, err := s.db.Exec(
		`INSERT INTO analytics_events(ts, host, method, path, status, bytes, remote_ip) VALUES
		 (?, ?, 'GET', '/', 200, 10, '127.0.0.1'),
		 (?, ?, 'GET', '/', 200, 10, '127.0.0.1'),
		 (?, ?, 'GET', '/', 500, 10, '127.0.0.1')`,
		now, "localhost:43123",
		now, "127.0.0.1:43123",
		now, "demo.example.com",
	); err != nil {
		t.Fatalf("insert analytics events: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/analytics?app=demo&env=preview&branch=feature&period=30d", nil)
	rr := httptest.NewRecorder()
	s.handleAnalytics(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp analyticsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.TotalRequests != 2 {
		t.Fatalf("expected only preview-lane requests, got %d", resp.TotalRequests)
	}
	if len(resp.ByHost) != 2 {
		t.Fatalf("expected two preview hosts, got %+v", resp.ByHost)
	}
	for _, host := range resp.ByHost {
		if host.Host == "demo.example.com" {
			t.Fatalf("unexpected prod host in lane-scoped analytics: %+v", resp.ByHost)
		}
	}
}
