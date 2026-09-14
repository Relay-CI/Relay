package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicUpdateReportsOpaqueVersionChange(t *testing.T) {
	s := newPreviewPortTestServer(t)
	state := &AppState{App: "demo", Env: EnvProd, Branch: "main", CurrentImage: "relay/demo:secret-internal-tag", AccessPolicy: AccessPolicyPublic}
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	version := laneReleaseVersion(state.App, state.Env, state.Branch, state.CurrentImage)
	req := httptest.NewRequest(http.MethodGet, "/api/public/update?app=demo&env=prod&branch=main&current=old", nil)
	rec := httptest.NewRecorder()
	s.handlePublicUpdate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	bodyText := rec.Body.String()
	var got map[string]any
	if err := json.Unmarshal([]byte(bodyText), &got); err != nil {
		t.Fatal(err)
	}
	if got["version"] != version || got["update_available"] != true {
		t.Fatalf("unexpected response: %#v", got)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("public update endpoint must be browser-callable")
	}
	if bytes := bodyText; bytes == "" || strings.Contains(bytes, state.CurrentImage) {
		t.Fatalf("response leaked internal image reference: %s", bytes)
	}
}

func TestPublicUpdateKeepsProtectedLanePrivate(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.saveAppState(&AppState{App: "demo", Env: EnvDev, Branch: "main", CurrentImage: "private", AccessPolicy: AccessPolicyRelayLogin}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/public/update?app=demo&env=dev&branch=main", nil)
	rec := httptest.NewRecorder()
	s.handlePublicUpdate(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
