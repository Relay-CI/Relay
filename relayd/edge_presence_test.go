package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testEdgeProxyRequest(t *testing.T, s *Server, cookie *http.Cookie, path, method string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	token, err := s.edgeSessionToken("demo", EnvProd, "main")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, edgeSessionProxyURL(8080, "127.0.0.1", "demo", EnvProd, "main"), body)
	req.Header.Set("X-Relay-Edge-Token", token)
	req.Header.Set("X-Relay-Lane-App", "demo")
	req.Header.Set("X-Relay-Lane-Env", string(EnvProd))
	req.Header.Set("X-Relay-Lane-Branch", "main")
	req.Header.Set("X-Relay-Original-Uri", path)
	req.Header.Set("X-Forwarded-Host", "demo.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	s.handleEdgeSessionProxy(w, req)
	return w
}

func testEdgeVersionServer(t *testing.T, version string) (*httptest.Server, int) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body>%s</body></html>", version)
			return
		}
		_, _ = io.WriteString(w, version+":"+r.URL.RequestURI())
	}))
	t.Cleanup(server.Close)
	parsed, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(parsed.Port())
	return server, port
}

func TestEdgeSessionDeploymentPreservesVisitorAcrossRefreshAndAPIRequests(t *testing.T) {
	s := newPreviewPortTestServer(t)
	_, bluePort := testEdgeVersionServer(t, "v1")
	_, greenPort := testEdgeVersionServer(t, "v2")
	runtime := s.runtime.(*mockRuntime)
	blue := appSlotContainerName("demo", EnvProd, "main", "blue")
	green := appSlotContainerName("demo", EnvProd, "main", "green")
	runtime.published[blue], runtime.published[green] = bluePort, greenPort
	runtime.running[blue], runtime.running[green] = true, true
	state := &AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, Mode: "port", TrafficMode: "session", ActiveSlot: "blue", ServicePort: 3000}
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	first := testEdgeProxyRequest(t, s, nil, "/", http.MethodGet, nil)
	if first.Code != 200 || !strings.Contains(first.Body.String(), "v1") {
		t.Fatalf("first visit: %d %s", first.Code, first.Body.String())
	}
	if !strings.Contains(first.Body.String(), edgePresenceScriptPath) {
		t.Fatal("heartbeat script was not injected")
	}
	cookies := first.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("invalid session cookie: %#v", cookies)
	}
	state.ActiveSlot, state.StandbySlot = "green", "blue"
	state.DrainUntil = time.Now().Add(time.Hour).UnixMilli()
	state.RolloutStatus = "draining"
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	old := testEdgeProxyRequest(t, s, cookies[0], "/", http.MethodGet, nil)
	if old.Code != 200 || !strings.Contains(old.Body.String(), "v1") {
		t.Fatalf("refresh changed old visitor: %d %s", old.Code, old.Body.String())
	}
	api := testEdgeProxyRequest(t, s, cookies[0], "/api/post?id=7", http.MethodPost, strings.NewReader("upload"))
	if api.Code != 200 || api.Body.String() != "v1:/api/post?id=7" {
		t.Fatalf("API changed old visitor: %d %s", api.Code, api.Body.String())
	}
	newVisitor := testEdgeProxyRequest(t, s, nil, "/", http.MethodGet, nil)
	if newVisitor.Code != 200 || !strings.Contains(newVisitor.Body.String(), "v2") {
		t.Fatalf("new visitor did not get v2: %d %s", newVisitor.Code, newVisitor.Body.String())
	}
	if newVisitor.Result().Cookies()[0].Value == cookies[0].Value {
		t.Fatal("visitors share session identity")
	}
	if got, err := s.activeOldSessions(state, time.Now()); err != nil || got != 1 {
		t.Fatalf("old session count = %d, %v", got, err)
	}
}

func TestEdgeSessionProxyRequiresLaneToken(t *testing.T) {
	s := newPreviewPortTestServer(t)
	req := httptest.NewRequest(http.MethodGet, edgeSessionProxyURL(8080, "127.0.0.1", "demo", EnvProd, "main"), nil)
	req.Header.Set("X-Relay-Lane-App", "demo")
	req.Header.Set("X-Relay-Lane-Env", string(EnvProd))
	req.Header.Set("X-Relay-Lane-Branch", "main")
	req.Header.Set("X-Relay-Original-Uri", "/")
	w := httptest.NewRecorder()
	s.handleEdgeSessionProxy(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned route returned %d", w.Code)
	}
}

func TestEdgePresenceScriptServedRegardlessOfTrafficMode(t *testing.T) {
	s := newPreviewPortTestServer(t)
	// No app state saved — the handler would normally return 503 for session routes.
	// The presence.js script should still be served because it is mode-independent:
	// browsers may have cached HTML with the injected tag from a prior session deploy.
	w := testEdgeProxyRequest(t, s, nil, edgePresenceScriptPath, http.MethodGet, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for presence.js with no session state, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/javascript") {
		t.Fatalf("expected application/javascript content-type, got %q", ct)
	}
}

func TestEdgeSessionPresenceScriptLocationInAllNginxModes(t *testing.T) {
	for _, mode := range []string{"edge", "session", "canary"} {
		s := &Server{dataDir: t.TempDir(), httpAddr: ":8080"}
		configPath, err := s.writeEdgeProxyConfig("demo", EnvPreview, "main", "blue", "", 3000, mode, 100)
		if err != nil {
			t.Fatalf("mode=%s: write edge proxy config: %v", mode, err)
		}
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("mode=%s: read config: %v", mode, err)
		}
		text := string(data)
		if !strings.Contains(text, "location = /__relay/presence.js") {
			t.Fatalf("mode=%s: expected presence.js location block in all traffic modes, got:\n%s", mode, text)
		}
	}
}

func TestEdgePresenceWritesAreCoalescedWithoutEarlyExpiry(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.ensureEdgePresenceSchema(); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	if err := s.touchEdgePresence("demo", EnvProd, "main", "visitor", "blue", now); err != nil {
		t.Fatal(err)
	}
	first, err := s.edgePresenceSession("demo", EnvProd, "main", "visitor")
	if err != nil || first.LastSeen < now+edgePresenceWriteLease.Milliseconds() {
		t.Fatalf("presence lease not reserved: %#v, %v", first, err)
	}
	if err := s.touchEdgePresence("demo", EnvProd, "main", "visitor", "blue", now+1000); err != nil {
		t.Fatal(err)
	}
	second, err := s.edgePresenceSession("demo", EnvProd, "main", "visitor")
	if err != nil || second.LastSeen != first.LastSeen {
		t.Fatalf("heartbeat rewrote within lease: %#v, %v", second, err)
	}
	if err := s.touchEdgePresence("demo", EnvProd, "main", "visitor", "blue", first.LastSeen+1); err != nil {
		t.Fatal(err)
	}
	third, err := s.edgePresenceSession("demo", EnvProd, "main", "visitor")
	if err != nil || third.LastSeen <= second.LastSeen {
		t.Fatalf("presence did not renew after lease: %#v, %v", third, err)
	}
}

func TestEdgeSessionHeartbeatsKeepOtherTabsAliveThenExpire(t *testing.T) {
	s := newPreviewPortTestServer(t)
	_, bluePort := testEdgeVersionServer(t, "v1")
	_, greenPort := testEdgeVersionServer(t, "v2")
	runtime := s.runtime.(*mockRuntime)
	runtime.published[appSlotContainerName("demo", EnvProd, "main", "blue")] = bluePort
	runtime.published[appSlotContainerName("demo", EnvProd, "main", "green")] = greenPort
	state := &AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, Mode: "port", TrafficMode: "session", ActiveSlot: "blue", ServicePort: 3000}
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	first := testEdgeProxyRequest(t, s, nil, "/", http.MethodGet, nil)
	cookie := first.Result().Cookies()[0]
	state.ActiveSlot, state.StandbySlot, state.RolloutStatus = "green", "blue", "draining"
	state.DrainUntil = time.Now().Add(time.Hour).UnixMilli()
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	// Two tabs share the visitor cookie. Either tab can renew presence.
	for _, tab := range []string{"tab-one", "tab-two"} {
		beat := testEdgeProxyRequest(t, s, cookie, edgePresencePath, http.MethodPost, strings.NewReader(tab))
		if beat.Code != http.StatusNoContent || len(beat.Result().Cookies()) != 1 {
			t.Fatalf("heartbeat %s: %d", tab, beat.Code)
		}
	}
	// Simulate tab one closing and a missed heartbeat window; tab two's next
	// heartbeat alone keeps this visitor on the old slot.
	_, err := s.db.Exec(`UPDATE edge_presence SET last_seen_at=? WHERE id=?`, time.Now().Add(-time.Minute).UnixMilli(), cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	beat := testEdgeProxyRequest(t, s, cookie, edgePresencePath, http.MethodPost, strings.NewReader("tab-two"))
	if beat.Code != http.StatusNoContent {
		t.Fatalf("remaining tab heartbeat: %d", beat.Code)
	}
	if old := testEdgeProxyRequest(t, s, cookie, "/", http.MethodGet, nil); !strings.Contains(old.Body.String(), "v1") {
		t.Fatalf("remaining tab lost old version: %s", old.Body.String())
	}
	_, err = s.db.Exec(`UPDATE edge_presence SET last_seen_at=? WHERE id=?`, time.Now().Add(-edgePresenceTTL()-time.Second).UnixMilli(), cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.activeOldSessions(state, time.Now()); err != nil || got != 0 {
		t.Fatalf("expired session counted: %d %v", got, err)
	}
	if next := testEdgeProxyRequest(t, s, cookie, "/", http.MethodGet, nil); !strings.Contains(next.Body.String(), "v2") {
		t.Fatalf("expired visitor did not move to v2: %s", next.Body.String())
	}
}

func TestEdgeSessionDrainWaitsForActiveUploadAtMaximum(t *testing.T) {
	s := newPreviewPortTestServer(t)
	started, release := make(chan struct{}), make(chan struct{})
	oldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/upload" {
			close(started)
			<-release
		}
		_, _ = io.WriteString(w, "v1")
	}))
	t.Cleanup(oldServer.Close)
	parsed, _ := url.Parse(oldServer.URL)
	bluePort, _ := strconv.Atoi(parsed.Port())
	_, greenPort := testEdgeVersionServer(t, "v2")
	runtime := s.runtime.(*mockRuntime)
	blue := appSlotContainerName("demo", EnvProd, "main", "blue")
	green := appSlotContainerName("demo", EnvProd, "main", "green")
	proxy := appBaseContainerName("demo", EnvProd, "main")
	runtime.published[blue], runtime.published[green] = bluePort, greenPort
	runtime.running[blue], runtime.running[green], runtime.running[proxy] = true, true, true
	state := &AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, Mode: "port", TrafficMode: "session", ActiveSlot: "blue", ServicePort: 3000, HostPort: 3000}
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	cookie := testEdgeProxyRequest(t, s, nil, "/", http.MethodGet, nil).Result().Cookies()[0]
	state.ActiveSlot, state.StandbySlot, state.RolloutStatus = "green", "blue", "draining"
	state.DrainUntil = time.Now().Add(time.Hour).UnixMilli()
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		testEdgeProxyRequest(t, s, cookie, "/upload", http.MethodPost, strings.NewReader("large upload"))
		close(done)
	}()
	<-started
	state.DrainUntil = time.Now().Add(-time.Second).UnixMilli()
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	if retired := s.sessionDrainTick("demo", EnvProd, "main", "green", "blue"); retired {
		t.Fatal("removed old slot during upload")
	}
	if s.activeEdgeRequests("demo", EnvProd, "main", "blue") != 1 {
		t.Fatal("upload was not counted")
	}
	if next := testEdgeProxyRequest(t, s, cookie, "/", http.MethodGet, nil); !strings.Contains(next.Body.String(), "v2") {
		t.Fatalf("maximum drain did not move future requests to v2: %s", next.Body.String())
	}
	for _, event := range runtime.events {
		if event == "remove:"+blue {
			t.Fatal("old slot removed before upload completed")
		}
	}
	close(release)
	<-done
	if retired := s.sessionDrainTick("demo", EnvProd, "main", "green", "blue"); !retired {
		t.Fatal("old slot remained after upload finished")
	}
	found := false
	for _, event := range runtime.events {
		if event == "remove:"+blue {
			found = true
		}
	}
	if !found {
		t.Fatal("old slot was not removed")
	}
}

func TestEdgeSessionDrainRemovesIdleVersionBeforeMaximum(t *testing.T) {
	s := newPreviewPortTestServer(t)
	_, bluePort := testEdgeVersionServer(t, "v1")
	_, greenPort := testEdgeVersionServer(t, "v2")
	runtime := s.runtime.(*mockRuntime)
	blue := appSlotContainerName("demo", EnvProd, "main", "blue")
	green := appSlotContainerName("demo", EnvProd, "main", "green")
	proxy := appBaseContainerName("demo", EnvProd, "main")
	runtime.published[blue], runtime.published[green] = bluePort, greenPort
	runtime.running[blue], runtime.running[green], runtime.running[proxy] = true, true, true
	state := &AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, Mode: "port", TrafficMode: "session", ActiveSlot: "blue", ServicePort: 3000, HostPort: 3000}
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	cookie := testEdgeProxyRequest(t, s, nil, "/", http.MethodGet, nil).Result().Cookies()[0]
	state.ActiveSlot, state.StandbySlot, state.RolloutStatus = "green", "blue", "draining"
	state.DrainUntil = time.Now().Add(time.Hour).UnixMilli()
	if err := s.saveAppState(state); err != nil {
		t.Fatal(err)
	}
	if retired := s.sessionDrainTick("demo", EnvProd, "main", "green", "blue"); retired {
		t.Fatal("removed version with a present visitor")
	}
	if _, err := s.db.Exec(`UPDATE edge_presence SET last_seen_at=? WHERE id=?`, time.Now().Add(-edgePresenceTTL()-time.Second).UnixMilli(), cookie.Value); err != nil {
		t.Fatal(err)
	}
	if retired := s.sessionDrainTick("demo", EnvProd, "main", "green", "blue"); !retired {
		t.Fatal("idle version was not removed")
	}
	removed := false
	for _, event := range runtime.events {
		if event == "remove:"+blue {
			removed = true
		}
	}
	if !removed {
		t.Fatal("old container was not removed")
	}
	updated, err := s.getAppState("demo", EnvProd, "main")
	if err != nil || updated.StandbySlot != "" {
		t.Fatalf("standby state not cleared: %#v, %v", updated, err)
	}
}
