package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func configureRollingTestLane(t *testing.T, s *Server, port int) {
	t.Helper()
	rt := s.runtime.(*mockRuntime)
	name := appSlotContainerName("demo", EnvProd, "main", "blue")
	rt.running[name], rt.published[name] = true, port
	if err := s.saveAppState(&AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker,
		Mode: "port", TrafficMode: "session", ActiveSlot: "blue", ServicePort: 3000}); err != nil {
		t.Fatal(err)
	}
}

func TestRollingHTMLStreamsBeforeUpstreamFinishes(t *testing.T) {
	s := newPreviewPortTestServer(t)
	started, release := make(chan struct{}), make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("ETag", `"original-html"`)
		io.WriteString(w, "<!doctype html><html><head></head><body>first chunk")
		w.(http.Flusher).Flush()
		close(started)
		<-release
		io.WriteString(w, "last chunk</body></html>")
	}))
	defer upstream.Close()
	parsed, _ := url.Parse(upstream.URL)
	port, _ := strconv.Atoi(parsed.Port())
	configureRollingTestLane(t, s, port)
	token, err := s.edgeSessionToken("demo", EnvProd, "main")
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Relay-Edge-Token", token)
		r.Header.Set("X-Relay-Lane-App", "demo")
		r.Header.Set("X-Relay-Lane-Env", "prod")
		r.Header.Set("X-Relay-Lane-Branch", "main")
		r.Header.Set("X-Relay-Original-Uri", "/")
		s.handleEdgeSessionProxy(w, r)
	}))
	defer proxy.Close()
	first := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		resp, err := proxy.Client().Get(proxy.URL)
		if err != nil {
			first <- err
			return
		}
		defer resp.Body.Close()
		_, err = io.ReadFull(resp.Body, make([]byte, 1))
		first <- err
		io.Copy(io.Discard, resp.Body)
	}()
	<-started
	streamed := false
	select {
	case err := <-first:
		streamed = err == nil
	case <-time.After(time.Second):
	}
	close(release)
	<-finished
	if !streamed {
		t.Fatal("rolling proxy buffered streamed HTML until upstream finished")
	}
}

func TestRollingPreservesImmutableAssetBrowserCache(t *testing.T) {
	s := newPreviewPortTestServer(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		io.WriteString(w, "console.log('asset')")
	}))
	defer upstream.Close()
	parsed, _ := url.Parse(upstream.URL)
	port, _ := strconv.Atoi(parsed.Port())
	configureRollingTestLane(t, s, port)
	w := testEdgeProxyRequest(t, s, nil, "/_next/static/chunks/hashed.js", http.MethodGet, nil)
	cache := w.Header().Get("Cache-Control")
	if w.Code != 200 || !strings.Contains(cache, "immutable") || !strings.Contains(cache, "private") || strings.Contains(cache, "public") {
		t.Fatalf("immutable asset browser cache lost or shared with visitor cookies: %d %s", w.Code, cache)
	}
}

type countingPublishedRuntime struct {
	*mockRuntime
	inspections int
}

func (r *countingPublishedRuntime) PublishedPort(name string, port int) int {
	r.inspections++
	return r.mockRuntime.PublishedPort(name, port)
}

func TestRollingReusesSlotEndpointAcrossRequests(t *testing.T) {
	s := newPreviewPortTestServer(t)
	_, port := testEdgeVersionServer(t, "v1")
	configureRollingTestLane(t, s, port)
	rt := &countingPublishedRuntime{mockRuntime: s.runtime.(*mockRuntime)}
	s.runtime = rt
	var cookie *http.Cookie
	for i := 0; i < 5; i++ {
		w := testEdgeProxyRequest(t, s, cookie, "/asset.js", http.MethodGet, nil)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		cookie = w.Result().Cookies()[0]
	}
	if rt.inspections != 1 {
		t.Fatalf("5 requests spawned %d runtime inspections; want 1", rt.inspections)
	}
}

func TestRetireStandbyDoesNothingWhenLaneWasDeleted(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)
	proxy := appBaseContainerName("demo", EnvProd, "main")
	rt.running[proxy] = true
	s.retireStandbySlot("demo", EnvProd, "main", "green", "blue", 3000, 3000, "port", "session", "")
	if len(rt.events) != 0 {
		t.Fatalf("stale drain mutated deleted lane: %v", rt.events)
	}
}

func TestChangingCanarySplitRefreshesLiveProxy(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)
	proxy := appBaseContainerName("demo", EnvProd, "main")
	rt.running[proxy] = true
	rt.published[proxy] = 3000
	for _, slot := range []string{"blue", "green"} {
		rt.running[appSlotContainerName("demo", EnvProd, "main", slot)] = true
	}
	st := &AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, Mode: "port",
		HostPort: 3000, ServicePort: 3000, ActiveSlot: "green", StandbySlot: "blue", TrafficMode: "canary", TrafficSplitPercent: 10, RolloutStatus: "monitoring"}
	if err := s.saveAppState(st); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/apps/config", strings.NewReader(`{"app":"demo","env":"prod","branch":"main","traffic_split_percent":50}`))
	w := httptest.NewRecorder()
	s.handleAppConfig(w, r)
	if w.Code != 200 {
		t.Fatalf("config update: %d %s", w.Code, w.Body.String())
	}
	config, err := os.ReadFile(s.edgeProxyConfigPath("demo", EnvProd, "main"))
	if err != nil || !strings.Contains(string(config), "50% green;") {
		t.Fatalf("live proxy did not apply new split: %s, %v", config, err)
	}
	for _, event := range rt.events {
		if strings.HasPrefix(event, "remove:") {
			t.Fatalf("split change restarted a container: %s", event)
		}
	}
}

func TestRollingRequestsAlreadyInFlightSurviveModeChange(t *testing.T) {
	s := newPreviewPortTestServer(t)
	_, port := testEdgeVersionServer(t, "v1")
	configureRollingTestLane(t, s, port)
	st, _ := s.getAppState("demo", EnvProd, "main")
	st.TrafficMode = "edge"
	if err := s.saveAppState(st); err != nil {
		t.Fatal(err)
	}
	// An old nginx worker may already have accepted this request before the
	// graceful policy reload. It must still reach the currently active app.
	w := testEdgeProxyRequest(t, s, nil, "/api/data", http.MethodGet, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "v1:/api/data") {
		t.Fatalf("policy change dropped in-flight request: %d %s", w.Code, w.Body.String())
	}
}

func TestCanaryPolicyCanBeConfiguredBeforeNextDeployment(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)
	rt.running[appBaseContainerName("demo", EnvProd, "main")] = true
	rt.published[appBaseContainerName("demo", EnvProd, "main")] = 3000
	rt.running[appSlotContainerName("demo", EnvProd, "main", "blue")] = true
	st := &AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, Mode: "port", ActiveSlot: "blue", TrafficMode: "edge", TrafficSplitPercent: 100}
	if err := s.saveAppState(st); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/apps/config", strings.NewReader(`{"app":"demo","env":"prod","branch":"main","traffic_mode":"canary","traffic_split_percent":10}`))
	s.handleAppConfig(w, r)
	if w.Code != 200 {
		t.Fatalf("config update: %d %s", w.Code, w.Body.String())
	}
	got, err := s.getAppState("demo", EnvProd, "main")
	if err != nil || got.TrafficSplitPercent != 10 {
		t.Fatalf("configured canary split reset before deployment: %#v %v", got, err)
	}
}

func TestRollingStreamInjectionHandlesSplitHeadTags(t *testing.T) {
	script := `<script src="` + edgePresenceScriptPath + `" defer></script>`
	for _, html := range []string{
		"<!doctype html><html><head></head><body>page</body></html>",
		"<HTML><HEAD data-title='test'></HEAD><BODY>page</BODY></HTML>",
		"<html><body>headless page</body></html>",
	} {
		body := &edgePresenceBody{ReadCloser: io.NopCloser(iotest.OneByteReader(strings.NewReader(html)))}
		got, err := io.ReadAll(body)
		if err != nil || strings.Count(string(got), script) != 1 || strings.Replace(string(got), script, "", 1) != html {
			t.Fatalf("stream injection lost/duplicated bytes: %q %v", got, err)
		}
	}
}

func TestRollingEndpointCacheInvalidatedBeforeSlotReuse(t *testing.T) {
	s := newPreviewPortTestServer(t)
	_, firstPort := testEdgeVersionServer(t, "v1")
	_, secondPort := testEdgeVersionServer(t, "v2")
	configureRollingTestLane(t, s, firstPort)
	first := testEdgeProxyRequest(t, s, nil, "/api/version", http.MethodGet, nil)
	if !strings.Contains(first.Body.String(), "v1:") {
		t.Fatal(first.Body.String())
	}
	name := appSlotContainerName("demo", EnvProd, "main", "blue")
	s.runtime.(*mockRuntime).published[name] = secondPort
	if err := s.runSlotContainerWithRuntime(s.runtime, nil, "demo", EnvProd, "main", "blue", "new-image", 3000, "network", nil); err != nil {
		t.Fatal(err)
	}
	next := testEdgeProxyRequest(t, s, first.Result().Cookies()[0], "/api/version", http.MethodGet, nil)
	if next.Code != 200 || !strings.Contains(next.Body.String(), "v2:") {
		t.Fatalf("reused slot retained stale endpoint: %d %s", next.Code, next.Body.String())
	}
}

func TestEdgeProxyBindingChangesOnlyForListeningSocketChanges(t *testing.T) {
	previous := &AppState{Env: EnvProd, Mode: "traefik", HostPort: 3000, PublicHost: "ethrealm.com"}
	for _, tc := range []struct {
		mode string
		port int
		host string
		want bool
	}{
		{"traefik", 3000, "www.ethrealm.com", false},
		{"traefik", 3001, "ethrealm.com", true},
		{"port", 3000, "ethrealm.com", true},
		{"traefik", 3000, "", true},
	} {
		if got := edgeProxyBindingChanged(previous, tc.mode, tc.port, tc.host); got != tc.want {
			t.Fatalf("binding change %+v: got %t", tc, got)
		}
	}
}

func TestRollingAssetsKeepRequestedCompression(t *testing.T) {
	s := newPreviewPortTestServer(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			compressed := gzip.NewWriter(w)
			io.WriteString(compressed, "compressed javascript")
			compressed.Close()
			return
		}
		io.WriteString(w, "uncompressed javascript")
	}))
	defer upstream.Close()
	parsed, _ := url.Parse(upstream.URL)
	port, _ := strconv.Atoi(parsed.Port())
	configureRollingTestLane(t, s, port)
	token, err := s.edgeSessionToken("demo", EnvProd, "main")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/edge/session-proxy", nil)
	for key, value := range map[string]string{"X-Relay-Edge-Token": token, "X-Relay-Lane-App": "demo", "X-Relay-Lane-Env": "prod", "X-Relay-Lane-Branch": "main", "X-Relay-Original-Uri": "/_next/static/chunks/app.js", "Accept-Encoding": "gzip"} {
		r.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	s.handleEdgeSessionProxy(w, r)
	if w.Code != 200 || w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("rolling mode disabled asset compression: %d %v", w.Code, w.Header())
	}
	decoded, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer decoded.Close()
	data, err := io.ReadAll(decoded)
	if err != nil || string(data) != "compressed javascript" {
		t.Fatalf("compressed asset corrupted: %q %v", data, err)
	}
}

func TestCanaryGraduationUpdatesLiveTrafficBeforeDrain(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)
	rt.running[appBaseContainerName("demo", EnvProd, "main")] = true
	rt.running[appSlotContainerName("demo", EnvProd, "main", "blue")] = true
	rt.running[appSlotContainerName("demo", EnvProd, "main", "green")] = true
	st := &AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, Mode: "port", HostPort: 3000,
		ServicePort: 3000, ActiveSlot: "green", StandbySlot: "blue", TrafficMode: "canary", TrafficSplitPercent: 10, RolloutStatus: "monitoring"}
	if err := s.saveAppState(st); err != nil {
		t.Fatal(err)
	}
	if _, err := s.writeEdgeProxyConfig(st.App, st.Env, st.Branch, "green", "blue", 3000, "canary", 10); err != nil {
		t.Fatal(err)
	}
	if err := s.graduateCanary(st.App, st.Env, st.Branch, 100, 0, nil); err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile(s.edgeProxyConfigPath(st.App, st.Env, st.Branch))
	if err != nil || strings.Contains(string(config), "split_clients") {
		t.Fatalf("graduated state still serves old canary split: %v", err)
	}
	for _, event := range rt.events {
		if strings.HasPrefix(event, "remove:") {
			t.Fatalf("graduation interrupted draining container: %s", event)
		}
	}
}
