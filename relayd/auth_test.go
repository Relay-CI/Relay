package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestControlMiddlewareAddsSecurityHeaders(t *testing.T) {
	s := &Server{}
	handler := s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "https://relay.example/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	for _, name := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Permissions-Policy", "Strict-Transport-Security"} {
		if rec.Header().Get(name) == "" {
			t.Fatalf("missing security header %s", name)
		}
	}
}

func TestConcurrentFirstOwnerSetupCreatesOnlyOneOwner(t *testing.T) {
	s := newPreviewPortTestServer(t)
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for _, username := range []string{"first-owner", "second-owner"} {
		wg.Add(1)
		go func(username string) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(`{"username":"`+username+`","password":"password123"}`))
			rec := httptest.NewRecorder()
			s.handleAuthSetup(rec, req)
			statuses <- rec.Code
		}(username)
	}
	wg.Wait()
	close(statuses)
	ok, conflict := 0, 0
	for status := range statuses {
		if status == http.StatusOK {
			ok++
		}
		if status == http.StatusConflict {
			conflict++
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("expected one owner and one conflict, got ok=%d conflict=%d", ok, conflict)
	}
}

func TestVerifyPasswordAcceptsLegacyHashAndFlagsUpgrade(t *testing.T) {
	legacy := checkLegacyPasswordFixture(t, "correct horse")
	valid, upgrade := verifyPassword("correct horse", legacy)
	if !valid || !upgrade {
		t.Fatalf("legacy hash: valid=%v upgrade=%v, want valid=true upgrade=true", valid, upgrade)
	}
	if valid, _ := verifyPassword("wrong password", legacy); valid {
		t.Fatal("legacy hash accepted wrong password")
	}
}

func TestVerifyPasswordAcceptsArgon2idAndSkipsUpgrade(t *testing.T) {
	hash, err := hashPassword("correct horse")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	valid, upgrade := verifyPassword("correct horse", hash)
	if !valid || upgrade {
		t.Fatalf("argon2id hash: valid=%v upgrade=%v, want valid=true upgrade=false", valid, upgrade)
	}
	if valid, _ := verifyPassword("wrong password", hash); valid {
		t.Fatal("argon2id hash accepted wrong password")
	}
}

func checkLegacyPasswordFixture(t *testing.T, password string) string {
	t.Helper()
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i + 1)
	}
	h := pbkdf2HMACSHA256([]byte(password), salt, pwHashIter)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(h)
}

func TestLoginThrottledAfterRepeatedFailures(t *testing.T) {
	s := newPreviewPortTestServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(`{"username":"owner","password":"password123"}`))
	setupRec := httptest.NewRecorder()
	s.handleAuthSetup(setupRec, setupReq)
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup: status=%d body=%s", setupRec.Code, setupRec.Body.String())
	}

	var lastCode int
	for i := 0; i < loginMaxPerUser+2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"owner","password":"wrong"}`))
		req.RemoteAddr = "203.0.113.5:9999"
		rec := httptest.NewRecorder()
		s.handleAuthLogin(rec, req)
		lastCode = rec.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected final attempt throttled with 429, got %d", lastCode)
	}

}

func TestLoginThrottledPerIPAcrossUsernames(t *testing.T) {
	s := newPreviewPortTestServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(`{"username":"owner","password":"password123"}`))
	setupRec := httptest.NewRecorder()
	s.handleAuthSetup(setupRec, setupReq)
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup: status=%d body=%s", setupRec.Code, setupRec.Body.String())
	}

	var lastCode int
	for i := 0; i < loginMaxPerIP+2; i++ {
		body := `{"username":"guess-` + strconv.Itoa(i) + `","password":"wrong"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.7:9999"
		rec := httptest.NewRecorder()
		s.handleAuthLogin(rec, req)
		lastCode = rec.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected per-IP throttle across usernames, got %d", lastCode)
	}
}

func TestDashboardLoginUsesShortIdleSessionKind(t *testing.T) {
	s := newPreviewPortTestServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(`{"username":"owner","password":"password123"}`))
	setupRec := httptest.NewRecorder()
	s.handleAuthSetup(setupRec, setupReq)
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup: status=%d body=%s", setupRec.Code, setupRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"owner","password":"password123"}`))
	req.RemoteAddr = "203.0.113.6:9999"
	rec := httptest.NewRecorder()
	s.handleAuthLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cookieToken string
	for _, c := range rec.Result().Cookies() {
		if c.Name == dashboardSessionCookie {
			cookieToken = c.Value
		}
	}
	if cookieToken == "" {
		t.Fatal("login did not set dashboard session cookie")
	}
	var kind string
	if err := s.db.QueryRow(`SELECT kind FROM user_sessions WHERE token=?`, storedSessionToken(cookieToken)).Scan(&kind); err != nil {
		t.Fatalf("read session kind: %v", err)
	}
	if kind != "dashboard" {
		t.Fatalf("dashboard login session kind = %q, want %q", kind, "dashboard")
	}
}

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

func TestHandleServerConfigSetsRelaySecretKey(t *testing.T) {
	s := newPreviewPortTestServer(t)
	ownerToken := createUserSessionForTest(t, s, "owner-user", "owner")

	req := httptest.NewRequest(http.MethodPost, "/api/server/config", strings.NewReader(`{"relay_secret_key":"dashboard-passphrase"}`))
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: ownerToken})
	rec := httptest.NewRecorder()

	s.authWithRoles("owner")(s.handleServerConfig)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected secret key save to pass, got %d (%s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["relay_secret_key_configured"] != true {
		t.Fatalf("expected relay_secret_key_configured=true, got %#v", body)
	}
	if body["relay_secret_key_source"] != "dashboard" {
		t.Fatalf("expected dashboard key source, got %#v", body)
	}
	if got := s.serverConfigGet("relay_secret_key"); got != "dashboard-passphrase" {
		t.Fatalf("expected persisted dashboard key, got %q", got)
	}
	if encrypted := s.encryptSecret("token"); !strings.HasPrefix(encrypted, "enc:") {
		t.Fatalf("expected newly configured key to encrypt immediately, got %q", encrypted)
	}
}

func TestHandleServerConfigRejectsUnsafeRelaySecretKeyRotation(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.setRelaySecretKey("first-passphrase", "dashboard")
	encrypted := s.encryptSecret("admin-token")
	if _, err := s.db.Exec(
		`INSERT OR REPLACE INTO app_secrets (app, env, branch, key, value) VALUES (?, ?, ?, ?, ?)`,
		"demo", "prod", "main", "TOKEN", encrypted,
	); err != nil {
		t.Fatalf("insert encrypted secret: %v", err)
	}
	ownerToken := createUserSessionForTest(t, s, "owner-user", "owner")

	req := httptest.NewRequest(http.MethodPost, "/api/server/config", strings.NewReader(`{"relay_secret_key":"different-passphrase"}`))
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: ownerToken})
	rec := httptest.NewRecorder()

	s.authWithRoles("owner")(s.handleServerConfig)(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected unsafe rotation to be rejected, got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := s.decryptSecret(encrypted); got != "admin-token" {
		t.Fatalf("expected old key to remain active, got %q", got)
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

func TestScopedLaneAccessAllowsOnlyGrantedEnv(t *testing.T) {
	s := newPreviewPortTestServer(t)
	token := createUserSessionForTest(t, s, "scoped-user", "deployer")

	var userID string
	if err := s.db.QueryRow(`SELECT id FROM users WHERE username=?`, "scoped-user").Scan(&userID); err != nil {
		t.Fatalf("lookup user: %v", err)
	}
	if _, err := s.db.Exec(
		`INSERT INTO user_permissions (user_id, app, env, role) VALUES (?, ?, ?, ?)`,
		userID, "demo", "staging", "deployer",
	); err != nil {
		t.Fatalf("insert permission: %v", err)
	}

	if role := s.effectiveLaneRole(&UserSession{UserID: userID, Username: "scoped-user", Role: "deployer"}, "demo", EnvStaging); role != "deployer" {
		t.Fatalf("expected staging role deployer, got %q", role)
	}
	if role := s.effectiveLaneRole(&UserSession{UserID: userID, Username: "scoped-user", Role: "deployer"}, "demo", EnvProd); role != "" {
		t.Fatalf("expected prod role to be denied, got %q", role)
	}

	handler := s.authByMethod(nil, []string{"owner", "admin", "deployer"})(func(w http.ResponseWriter, r *http.Request) {
		_, ok := s.requireLaneAccess(w, r, "demo", EnvStaging, "deployer")
		if ok {
			w.WriteHeader(http.StatusNoContent)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/apps/restart", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: token})
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected staging write to pass, got %d", rec.Code)
	}
}
