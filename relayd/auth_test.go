package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func createUserSessionForTest(t *testing.T, s *Server, username string, role string) string {
	t.Helper()

	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID := newID()
	if _, err := s.db.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, username, hash, role, time.Now().UnixMilli(),
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	token, err := s.createUserSession(userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return token
}

func TestHandleDashboardSessionReturnsSetupAndLegacyModeWithoutUsers(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.apiToken = "legacy-token"

	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	rec := httptest.NewRecorder()

	s.handleDashboardSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["setup_required"] != true {
		t.Fatalf("expected setup_required=true, got %#v", body)
	}
	if body["legacy_mode"] != false {
		t.Fatalf("expected legacy_mode=false (setup forced), got %#v", body)
	}
}

func TestHandleDashboardSessionAuthenticatesLegacyTokenWithoutUsers(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.apiToken = "legacy-token"

	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.Header.Set("Authorization", "Bearer legacy-token")
	rec := httptest.NewRecorder()

	s.handleDashboardSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["authenticated"] != true || body["legacy_mode"] != true {
		t.Fatalf("expected authenticated legacy response, got %#v", body)
	}
}

func TestAuthByMethodBlocksViewerWrites(t *testing.T) {
	s := newPreviewPortTestServer(t)
	token := createUserSessionForTest(t, s, "viewer-user", "viewer")

	handler := s.authByMethod(nil, []string{"owner", "deployer"})(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	readReq := httptest.NewRequest(http.MethodGet, "/api/deploys", nil)
	readReq.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: token})
	readRec := httptest.NewRecorder()
	handler(readRec, readReq)
	if readRec.Code != http.StatusNoContent {
		t.Fatalf("expected viewer GET to pass, got %d", readRec.Code)
	}

	writeReq := httptest.NewRequest(http.MethodPost, "/api/deploys", strings.NewReader(`{}`))
	writeReq.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: token})
	writeRec := httptest.NewRecorder()
	handler(writeRec, writeReq)
	if writeRec.Code != http.StatusForbidden {
		t.Fatalf("expected viewer POST to be forbidden, got %d", writeRec.Code)
	}
}

func TestAuthWithRolesBlocksNonOwners(t *testing.T) {
	s := newPreviewPortTestServer(t)
	token := createUserSessionForTest(t, s, "deploy-user", "deployer")

	handler := s.authWithRoles("owner")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/server/config", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: token})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected non-owner to be forbidden, got %d", rec.Code)
	}
}

func TestHandleServerConfigRejectsInvalidDashboardHost(t *testing.T) {
	s := newPreviewPortTestServer(t)
	ownerToken := createUserSessionForTest(t, s, "owner-user", "owner")

	req := httptest.NewRequest(http.MethodPost, "/api/server/config", strings.NewReader(`{"dashboard_host":"admin.example.com { reverse_proxy 127.0.0.1:9999 }"}`))
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: ownerToken})
	rec := httptest.NewRecorder()

	s.authWithRoles("owner")(s.handleServerConfig)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid hostname to be rejected, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandleAuthCLIStartUsesExistingSession(t *testing.T) {
	s := newPreviewPortTestServer(t)
	ownerToken := createUserSessionForTest(t, s, "owner-user", "owner")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/cli/start", strings.NewReader(`{"cli_port":52702}`))
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: ownerToken})
	rec := httptest.NewRecorder()

	s.handleAuthCLIStart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	redirect, _ := body["cli_redirect"].(string)
	if !strings.Contains(redirect, "http://127.0.0.1:52702/callback?code=") {
		t.Fatalf("expected cli redirect for callback port, got %#v", body)
	}
}

func TestHandleEdgeAuthzDefaultsDevLaneToRelayLogin(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.saveAppState(&AppState{App: "demo", Env: EnvDev, Branch: "main"}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/edge/authz?app=demo&env=dev&branch=main", nil)
	rec := httptest.NewRecorder()
	s.handleEdgeAuthz(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated dev lane to require relay login, got %d", rec.Code)
	}

	token := createUserSessionForTest(t, s, "owner-user", "owner")
	authedReq := httptest.NewRequest(http.MethodGet, "/api/edge/authz?app=demo&env=dev&branch=main", nil)
	authedReq.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: token})
	authedRec := httptest.NewRecorder()
	s.handleEdgeAuthz(authedRec, authedReq)
	if authedRec.Code != http.StatusNoContent {
		t.Fatalf("expected authenticated dev lane request to pass, got %d", authedRec.Code)
	}
}

func TestHandleEdgeAuthzAllowsIPAllowlist(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.saveAppState(&AppState{
		App:          "demo",
		Env:          EnvStaging,
		Branch:       "main",
		AccessPolicy: AccessPolicyIPAllowlist,
		IPAllowlist:  "203.0.113.0/24",
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/api/edge/authz?app=demo&env=staging&branch=main", nil)
	allowedReq.Header.Set("X-Forwarded-For", "203.0.113.25")
	allowedRec := httptest.NewRecorder()
	s.handleEdgeAuthz(allowedRec, allowedReq)
	if allowedRec.Code != http.StatusNoContent {
		t.Fatalf("expected allowlisted IP to pass, got %d", allowedRec.Code)
	}

	blockedReq := httptest.NewRequest(http.MethodGet, "/api/edge/authz?app=demo&env=staging&branch=main", nil)
	blockedReq.Header.Set("X-Forwarded-For", "198.51.100.10")
	blockedRec := httptest.NewRecorder()
	s.handleEdgeAuthz(blockedRec, blockedReq)
	if blockedRec.Code != http.StatusForbidden {
		t.Fatalf("expected non-allowlisted IP to be blocked, got %d", blockedRec.Code)
	}
}
