package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShouldAutoAssignPreviewHostPort(t *testing.T) {
	baseReq := DeployRequest{
		App:    "demo",
		Branch: "main",
		Env:    EnvPreview,
	}

	if !shouldAutoAssignPreviewHostPort(baseReq, nil) {
		t.Fatalf("expected first preview deploy in port mode to auto-assign host port")
	}
	if shouldAutoAssignPreviewHostPort(DeployRequest{
		App:        "demo",
		Branch:     "main",
		Env:        EnvPreview,
		PublicHost: "demo-main.example.com",
	}, nil) {
		t.Fatalf("public-host deploy should not auto-assign host port")
	}
	if shouldAutoAssignPreviewHostPort(DeployRequest{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		HostPort: 4444,
	}, nil) {
		t.Fatalf("explicit host-port deploy should not auto-assign host port")
	}
	if shouldAutoAssignPreviewHostPort(DeployRequest{
		App:    "demo",
		Branch: "main",
		Env:    EnvPreview,
		Mode:   "traefik",
	}, nil) {
		t.Fatalf("traefik deploy should not auto-assign host port")
	}
	if shouldAutoAssignPreviewHostPort(DeployRequest{
		App:    "demo",
		Branch: "main",
		Env:    EnvProd,
	}, nil) {
		t.Fatalf("prod deploy should not auto-assign preview host port")
	}
	if !shouldAutoAssignPreviewHostPort(DeployRequest{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		HostPort: 3007,
	}, &AppState{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		HostPort: 3007,
	}) {
		t.Fatalf("inherited preview host port should auto-reassign if it becomes unavailable")
	}
	if shouldAutoAssignPreviewHostPort(DeployRequest{
		App:              "demo",
		Branch:           "main",
		Env:              EnvPreview,
		HostPort:         3010,
		HostPortExplicit: true,
	}, &AppState{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		HostPort: 3007,
	}) {
		t.Fatalf("explicit preview host port override should not auto-reassign")
	}

	t.Setenv("RELAY_BASE_DOMAIN", "preview.example.com")
	if shouldAutoAssignPreviewHostPort(baseReq, nil) {
		t.Fatalf("auto-host preview deploy should not auto-assign host port")
	}
}

func TestDeployLogLooksLikeOOMKillDetectsExit137(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "deploy.log")
	if err := os.WriteFile(logPath, []byte("Creating an optimized production build ...\n94.63 Killed\nexit code: 137\n"), 0644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if !deployLogLooksLikeOOMKill(logPath, "exit status 1") {
		t.Fatal("expected exit 137 log to be classified as likely OOM kill")
	}
	if deployLogLooksLikeOOMKill("", "exit status 1") {
		t.Fatal("plain exit status 1 should not be classified as OOM kill")
	}
}

func TestHostPortAvailable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if hostPortAvailable(port) {
		_ = ln.Close()
		t.Fatalf("occupied port %d reported as available", port)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	if !hostPortAvailable(port) {
		t.Fatalf("released port %d reported as unavailable", port)
	}
}

func TestDefaultBuildpacksPreferNextStandalone(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `export default { output: "standalone" }`)

	var selected Buildpack
	for _, bp := range defaultBuildpacks() {
		if bp.Detect(repoDir, nil) {
			selected = bp
			break
		}
	}
	if selected == nil {
		t.Fatalf("expected a buildpack match")
	}
	if selected.Name() != "next-standalone" {
		t.Fatalf("expected next-standalone first, got %q", selected.Name())
	}
}

func TestAssignPreviewHostPortReassignsInheritedBusyStatePort(t *testing.T) {
	s := newPreviewPortTestServer(t)
	base := mustFindConsecutiveFreePorts(t, 41000, 2)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	req := DeployRequest{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		Mode:     "port",
		HostPort: base,
	}
	state := &AppState{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		Mode:     "port",
		HostPort: base,
	}

	s.assignPreviewHostPort(s.runtime, &req, state, nil)
	if req.HostPort != base+1 {
		t.Fatalf("expected busy inherited port %d to move to %d, got %d", base, base+1, req.HostPort)
	}
}

func TestAssignPreviewHostPortKeepsExplicitBusyPort(t *testing.T) {
	s := newPreviewPortTestServer(t)
	base := mustFindConsecutiveFreePorts(t, 42000, 2)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	req := DeployRequest{
		App:              "demo",
		Branch:           "main",
		Env:              EnvPreview,
		Mode:             "port",
		HostPort:         base,
		HostPortExplicit: true,
	}

	s.assignPreviewHostPort(s.runtime, &req, nil, nil)
	if req.HostPort != base {
		t.Fatalf("expected explicit busy port %d to be preserved, got %d", base, req.HostPort)
	}
}

func TestAssignPreviewHostPortKeepsPersistedExplicitStatePort(t *testing.T) {
	s := newPreviewPortTestServer(t)
	base := mustFindConsecutiveFreePorts(t, 43000, 2)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	req := DeployRequest{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		Mode:     "port",
		HostPort: base,
	}
	state := &AppState{
		App:              "demo",
		Branch:           "main",
		Env:              EnvPreview,
		Mode:             "port",
		HostPort:         base,
		HostPortExplicit: true,
	}

	s.assignPreviewHostPort(s.runtime, &req, state, nil)
	if req.HostPort != base {
		t.Fatalf("expected persisted explicit port %d to be preserved, got %d", base, req.HostPort)
	}
}

func TestAssignPreviewHostPortSkipsReservedPorts(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if _, err := s.db.Exec(
		`INSERT INTO app_state (app, env, branch, mode, host_port, public_host) VALUES (?, ?, ?, ?, ?, ?)`,
		"other", string(EnvPreview), "main", "port", defaultHostPort(EnvPreview), "",
	); err != nil {
		t.Fatalf("insert reserved state: %v", err)
	}

	req := DeployRequest{
		App:    "demo",
		Branch: "main",
		Env:    EnvPreview,
		Mode:   "port",
	}

	s.assignPreviewHostPort(s.runtime, &req, nil, nil)
	if req.HostPort == defaultHostPort(EnvPreview) {
		t.Fatalf("expected reserved preview port %d to be skipped", defaultHostPort(EnvPreview))
	}
	if s.previewHostPortReservedByOtherApp(req.App, req.Env, req.Branch, req.HostPort) {
		t.Fatalf("assigned port %d is still reserved by another preview app", req.HostPort)
	}
	if !hostPortAvailable(req.HostPort) {
		t.Fatalf("assigned port %d is not actually available", req.HostPort)
	}
}

func TestAssignPreviewHostPortKeepsCurrentProxyPort(t *testing.T) {
	s := newPreviewPortTestServer(t)
	containerName := appBaseContainerName("demo", EnvPreview, "main")
	runtime := &mockRuntime{
		running:   map[string]bool{containerName: true},
		published: map[string]int{containerName: 3555},
	}

	req := DeployRequest{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		Mode:     "port",
		HostPort: 3555,
	}
	state := &AppState{
		App:      "demo",
		Branch:   "main",
		Env:      EnvPreview,
		Mode:     "port",
		HostPort: 3555,
	}

	s.assignPreviewHostPort(runtime, &req, state, nil)
	if req.HostPort != 3555 {
		t.Fatalf("expected current proxy port 3555 to be preserved, got %d", req.HostPort)
	}
}

func TestEdgeProxyPublishedPortChanged(t *testing.T) {
	containerName := appBaseContainerName("demo", EnvPreview, "main")
	runtime := &mockRuntime{
		running:   map[string]bool{containerName: true},
		published: map[string]int{containerName: 3003},
	}

	if !edgeProxyPublishedPortChanged(runtime, "demo", EnvPreview, "main", 3002, "port", "") {
		t.Fatalf("expected published port mismatch to require edge proxy recreation")
	}
	if edgeProxyPublishedPortChanged(runtime, "demo", EnvPreview, "main", 3003, "port", "") {
		t.Fatalf("matching published port should not require edge proxy recreation")
	}
}

func TestSaveAndLoadAppStatePersistsHostPortExplicit(t *testing.T) {
	s := newPreviewPortTestServer(t)
	st := &AppState{
		App:              "demo",
		Env:              EnvPreview,
		Branch:           "main",
		Engine:           EngineDocker,
		Mode:             "port",
		HostPort:         3555,
		HostPortExplicit: true,
		ServicePort:      3000,
		PublicHost:       "",
	}

	if err := s.saveAppState(st); err != nil {
		t.Fatalf("save app state: %v", err)
	}
	got, err := s.getAppState(st.App, st.Env, st.Branch)
	if err != nil {
		t.Fatalf("get app state: %v", err)
	}
	if !got.HostPortExplicit {
		t.Fatalf("expected host_port_explicit to persist")
	}
}

func TestSaveAndLoadAppStatePersistsPublicHosts(t *testing.T) {
	s := newPreviewPortTestServer(t)
	st := &AppState{
		App:         "demo",
		Env:         EnvProd,
		Branch:      "main",
		Engine:      EngineDocker,
		Mode:        "traefik",
		HostPort:    3555,
		ServicePort: 3000,
		PublicHost:  "demo.example.com",
		PublicHosts: []string{"demo.example.com", "www.demo.example.com", "demo.example.com"},
	}

	if err := s.saveAppState(st); err != nil {
		t.Fatalf("save app state: %v", err)
	}
	got, err := s.getAppState(st.App, st.Env, st.Branch)
	if err != nil {
		t.Fatalf("get app state: %v", err)
	}
	if got.PublicHost != "demo.example.com" {
		t.Fatalf("expected primary host to persist, got %q", got.PublicHost)
	}
	if len(got.PublicHosts) != 2 {
		t.Fatalf("expected 2 unique public hosts, got %#v", got.PublicHosts)
	}
	if got.PublicHosts[0] != "demo.example.com" || got.PublicHosts[1] != "www.demo.example.com" {
		t.Fatalf("unexpected public hosts: %#v", got.PublicHosts)
	}
}

func TestPersistedPublicHostsKeepsAliasesAcrossStateReplacement(t *testing.T) {
	current := &AppState{
		PublicHost:  "demo.example.com",
		PublicHosts: []string{"demo.example.com", "www.demo.example.com"},
	}
	previous := &AppState{
		PublicHost:  "old.example.com",
		PublicHosts: []string{"old.example.com", "www.old.example.com"},
	}

	hosts := persistedPublicHosts(DeployRequest{PublicHost: "demo.example.com"}, current, previous)
	if got := strings.Join(hosts, ","); got != "demo.example.com,www.demo.example.com" {
		t.Fatalf("redeploy lost configured aliases: %q", got)
	}

	hosts = persistedPublicHosts(DeployRequest{PublicHost: "demo.example.com"}, nil, current)
	if got := strings.Join(hosts, ","); got != "demo.example.com,www.demo.example.com" {
		t.Fatalf("rollback fallback lost configured aliases: %q", got)
	}

	hosts = persistedPublicHosts(DeployRequest{
		PublicHost:  "new.example.com",
		PublicHosts: []string{"new.example.com", "www.new.example.com"},
	}, current)
	if got := strings.Join(hosts, ","); got != "new.example.com,www.new.example.com" {
		t.Fatalf("explicit deploy hosts should win over persisted aliases: %q", got)
	}
}

func TestRepairLegacyAppHostPortsFromRuntime(t *testing.T) {
	s := newPreviewPortTestServer(t)
	containerName := appBaseContainerName("myhltv", EnvProd, "main")
	s.runtime = &mockRuntime{
		running:   map[string]bool{containerName: true},
		published: map[string]int{containerName: 3002},
	}

	if err := s.saveAppState(&AppState{
		App:        "myhltv",
		Env:        EnvProd,
		Branch:     "main",
		Engine:     EngineDocker,
		Mode:       "traefik",
		HostPort:   0,
		PublicHost: "myhltv.example.com",
		Stopped:    false,
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	repaired, err := s.repairLegacyAppHostPortsFromRuntime()
	if err != nil {
		t.Fatalf("repair host ports: %v", err)
	}
	if repaired != 1 {
		t.Fatalf("expected 1 repaired row, got %d", repaired)
	}

	got, err := s.getAppState("myhltv", EnvProd, "main")
	if err != nil {
		t.Fatalf("get repaired app state: %v", err)
	}
	if got.HostPort != 3002 {
		t.Fatalf("expected repaired host port 3002, got %d", got.HostPort)
	}
}

func TestEnsureGlobalProxyRepairsLegacyHostPortsBeforeWritingConfig(t *testing.T) {
	s := newPreviewPortTestServer(t)
	containerName := appBaseContainerName("myhltv", EnvProd, "main")
	s.runtime = &mockRuntime{
		running:   map[string]bool{containerName: true},
		published: map[string]int{containerName: 3002},
	}

	if err := s.saveAppState(&AppState{
		App:        "myhltv",
		Env:        EnvProd,
		Branch:     "main",
		Engine:     EngineDocker,
		Mode:       "traefik",
		HostPort:   0,
		PublicHost: "myhltv.example.com",
		Stopped:    false,
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	if err := s.ensureGlobalProxy(); err != nil {
		t.Fatalf("ensure global proxy: %v", err)
	}

	configPath := filepath.Join(s.dataDir, "global-proxy", "Caddyfile")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read caddyfile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "myhltv.example.com {\n\treverse_proxy "+containerName+":3000\n}") {
		t.Fatalf("expected repaired host route in caddyfile, got:\n%s", text)
	}

	got, err := s.getAppState("myhltv", EnvProd, "main")
	if err != nil {
		t.Fatalf("get repaired app state: %v", err)
	}
	if got.HostPort != 3002 {
		t.Fatalf("expected persisted host port 3002, got %d", got.HostPort)
	}
}

func TestEnsureGlobalProxyWritesAllPublicHostAliases(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)

	if err := s.saveAppState(&AppState{
		App:         "myhltv",
		Env:         EnvProd,
		Branch:      "main",
		Engine:      EngineDocker,
		Mode:        "traefik",
		HostPort:    3002,
		PublicHost:  "myhltv.example.com",
		PublicHosts: []string{"myhltv.example.com", "www.myhltv.example.com"},
		Stopped:     false,
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	if err := s.ensureGlobalProxy(); err != nil {
		t.Fatalf("ensure global proxy: %v", err)
	}

	configPath := filepath.Join(s.dataDir, "global-proxy", "Caddyfile")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read caddyfile: %v", err)
	}
	text := string(data)
	upstream := appBaseContainerName("myhltv", EnvProd, "main") + ":3000"
	if !strings.Contains(text, "myhltv.example.com {\n\treverse_proxy "+upstream+"\n}") {
		t.Fatalf("expected primary host route in caddyfile, got:\n%s", text)
	}
	if !strings.Contains(text, "www.myhltv.example.com {\n\treverse_proxy "+upstream+"\n}") {
		t.Fatalf("expected alias host route in caddyfile, got:\n%s", text)
	}
	if !rt.hasNetworkConnection("relay-global-proxy", appNetworkName("myhltv", EnvProd, "main")) {
		t.Fatalf("expected global proxy to join app network, got connections: %v", rt.networkConnections)
	}
}

func TestEnsureGlobalProxyKeepsStationLaneOnHostPort(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)

	if err := s.saveAppState(&AppState{
		App:        "desktop-app",
		Env:        EnvProd,
		Branch:     "main",
		Engine:     EngineStation,
		Mode:       "traefik",
		HostPort:   3010,
		PublicHost: "desktop.example.com",
		Stopped:    false,
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	if err := s.ensureGlobalProxy(); err != nil {
		t.Fatalf("ensure global proxy: %v", err)
	}

	configPath := filepath.Join(s.dataDir, "global-proxy", "Caddyfile")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read caddyfile: %v", err)
	}
	if !strings.Contains(string(data), "desktop.example.com {\n\treverse_proxy host.docker.internal:3010\n}") {
		t.Fatalf("expected station host-port route in caddyfile, got:\n%s", data)
	}
	if len(rt.networkConnections) != 0 {
		t.Fatalf("station lane should not attach caddy to a docker network, got: %v", rt.networkConnections)
	}
}

func TestEnsureGlobalProxyWritesCustomRedirectRule(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.writeServerConfig(map[string]string{
		"custom_host_rules": `[{"host":"relay.example.com","action":"redirect","redirect_url":"https://example.com","redirect_code":301,"preserve_path":true}]`,
	}); err != nil {
		t.Fatalf("write server config: %v", err)
	}

	if err := s.ensureGlobalProxy(); err != nil {
		t.Fatalf("ensure global proxy: %v", err)
	}

	configPath := filepath.Join(s.dataDir, "global-proxy", "Caddyfile")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read caddyfile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "relay.example.com {\n\tredir https://example.com{uri} 301\n}") {
		t.Fatalf("expected redirect rule in caddyfile, got:\n%s", text)
	}
}

func TestAppLaneRunningIgnoresProxyOnlyContainer(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)

	app := "demo"
	env := EnvPreview
	branch := "main"

	rt.running[appBaseContainerName(app, env, branch)] = true
	if s.appLaneRunning(app, env, branch) {
		t.Fatalf("edge proxy alone should not count as an app lane")
	}

	rt.running[appSlotContainerName(app, env, branch, "blue")] = true
	if !s.appLaneRunning(app, env, branch) {
		t.Fatalf("running app slot should count as an app lane")
	}
}

func TestCurrentActiveSlotPrefersRunningSlotOverStaleState(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)

	app := "demo"
	env := EnvPreview
	branch := "main"

	rt.running[appSlotContainerName(app, env, branch, "green")] = true

	state := &AppState{
		App:        app,
		Env:        env,
		Branch:     branch,
		ActiveSlot: "blue",
	}

	if got := s.currentActiveSlotWithRuntime(rt, app, env, branch, state); got != "green" {
		t.Fatalf("expected running slot green to override stale state, got %q", got)
	}
}

func TestRuntimeLogTargetsPreferAvailableAppLogsOverProxy(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)

	app := "demo"
	env := EnvPreview
	branch := "main"
	live := appSlotContainerName(app, env, branch, "blue")
	proxy := appBaseContainerName(app, env, branch)

	if err := s.saveAppState(&AppState{
		App:          app,
		Env:          env,
		Branch:       branch,
		Engine:       EngineDocker,
		ActiveSlot:   "blue",
		CurrentImage: "demo:latest",
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	rt.exists[live] = true
	rt.exists[proxy] = true
	rt.running[proxy] = true

	targets, lane, err := s.runtimeLogTargets(app, env, branch)
	if err != nil {
		t.Fatalf("runtimeLogTargets: %v", err)
	}
	if lane.AppRunning {
		t.Fatalf("expected app lane to report no running app container")
	}
	if !lane.HasRunningTarget {
		t.Fatalf("expected running proxy target to keep lane targets online")
	}
	if got := runtimeLogDefaultTarget(targets); got != "live" {
		t.Fatalf("expected crashed live app logs to be default target, got %q", got)
	}

	selected, _, _, err := s.resolveRuntimeLogTarget(app, env, branch, "live")
	if err != nil {
		t.Fatalf("resolveRuntimeLogTarget: %v", err)
	}
	if selected == nil || selected.ID != "live" {
		t.Fatalf("expected live target, got %#v", selected)
	}
	if selected.Running {
		t.Fatalf("expected live target to be stopped")
	}
	if !selected.Available {
		t.Fatalf("expected live target logs to remain available")
	}
}

func TestRuntimeLogTargetsUseRunningSlotAsLiveTargetWhenStateIsStale(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := s.runtime.(*mockRuntime)

	app := "demo"
	env := EnvPreview
	branch := "main"
	green := appSlotContainerName(app, env, branch, "green")

	if err := s.saveAppState(&AppState{
		App:          app,
		Env:          env,
		Branch:       branch,
		Engine:       EngineDocker,
		ActiveSlot:   "blue",
		CurrentImage: "demo:latest",
	}); err != nil {
		t.Fatalf("save app state: %v", err)
	}

	rt.exists[green] = true
	rt.running[green] = true

	targets, _, err := s.runtimeLogTargets(app, env, branch)
	if err != nil {
		t.Fatalf("runtimeLogTargets: %v", err)
	}
	if got := runtimeLogDefaultTarget(targets); got != "live" {
		t.Fatalf("expected live target to follow running slot, got %q", got)
	}
	for _, target := range targets {
		if target.ID == "live" && target.Slot != "green" {
			t.Fatalf("expected live target to point at green slot, got %q", target.Slot)
		}
	}
}

// retireStandbySlot must drop the old slot from the edge proxy's routing
// map (nginx reload) BEFORE removing its container — removing first (the
// previous order) left a window where an in-flight or sticky-session
// request could resolve to a container that no longer existed, surfacing as
// a Cloudflare-visible 502 during every deploy's slot switch.
func TestRetireStandbySlotReloadsBeforeRemoving(t *testing.T) {
	s := newPreviewPortTestServer(t)
	app, env, branch := "demo", EnvPreview, "main"
	edgeName := appBaseContainerName(app, env, branch)
	oldSlotName := appSlotContainerName(app, env, branch, "blue")
	rt := s.runtime.(*mockRuntime)
	rt.running[edgeName] = true

	s.retireStandbySlot(app, env, branch, "green", "blue", 3000, 0, "port", "edge", "")

	var reloadIdx, removeIdx = -1, -1
	for i, ev := range rt.events {
		if strings.HasPrefix(ev, "exec:"+edgeName+":") && strings.Contains(ev, "reload") {
			reloadIdx = i
		}
		if ev == "remove:"+oldSlotName {
			removeIdx = i
		}
	}
	if reloadIdx == -1 {
		t.Fatalf("expected an nginx reload exec against %s, got events: %v", edgeName, rt.events)
	}
	if removeIdx == -1 {
		t.Fatalf("expected the old slot container %s to be removed, got events: %v", oldSlotName, rt.events)
	}
	if removeIdx < reloadIdx {
		t.Fatalf("expected reload (index %d) before remove (index %d), got events: %v", reloadIdx, removeIdx, rt.events)
	}
}

func TestWaitForRuntimeContainerReadyFailsFastForExitedContainer(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := &mockRuntime{
		running:   map[string]bool{},
		exists:    map[string]bool{"demo": true},
		published: map[string]int{},
	}

	start := time.Now()
	err := s.waitForRuntimeContainerReady(rt, nil, "demo", 3000, 5*time.Second)
	if err == nil {
		t.Fatalf("expected exited container readiness failure")
	}
	if !strings.Contains(err.Error(), "exited before becoming ready") {
		t.Fatalf("unexpected readiness error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("expected fast failure for exited container, took %s", elapsed)
	}
}

type mockRuntime struct {
	running            map[string]bool
	exists             map[string]bool
	published          map[string]int
	networkConnections [][2]string
	events             []string
}

func (m *mockRuntime) RunDetached(ContainerSpec) error { return nil }
func (m *mockRuntime) Remove(name string)              { m.events = append(m.events, "remove:"+name) }
func (m *mockRuntime) ContainerExists(name string) bool {
	if m.exists == nil {
		return m.running[name]
	}
	return m.exists[name] || m.running[name]
}
func (m *mockRuntime) IsRunning(name string) bool           { return m.running[name] }
func (m *mockRuntime) ContainerIP(string) string            { return "" }
func (m *mockRuntime) PublishedPort(name string, _ int) int { return m.published[name] }
func (m *mockRuntime) Exec(name string, args []string) ([]byte, error) {
	m.events = append(m.events, "exec:"+name+":"+strings.Join(args, " "))
	return nil, nil
}
func (m *mockRuntime) NetworkConnect(container, network string) error {
	m.networkConnections = append(m.networkConnections, [2]string{container, network})
	return nil
}
func (m *mockRuntime) hasNetworkConnection(container, network string) bool {
	for _, connection := range m.networkConnections {
		if connection[0] == container && connection[1] == network {
			return true
		}
	}
	return false
}
func (m *mockRuntime) EnsureNetwork(string) error { return nil }
func (m *mockRuntime) RemoveNetwork(string)       {}
func (m *mockRuntime) RemoveVolume(string)        {}
func (m *mockRuntime) Pull(string) error          { return nil }
func (m *mockRuntime) Build(context.Context, string, string, string, map[string]string, io.Writer, string) error {
	return nil
}
func (m *mockRuntime) RemoveImage(string)                  {}
func (m *mockRuntime) ListImages(string) ([]string, error) { return nil, nil }
func (m *mockRuntime) LogStream(context.Context, string, int, string) (io.ReadCloser, error) {
	return nil, nil
}

func newPreviewPortTestServer(t *testing.T) *Server {
	t.Helper()
	dataDir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "relayd-test.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := migrateDB(db); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return &Server{
		dataDir:      dataDir,
		caddyLogsDir: filepath.Join(dataDir, "caddy-logs"),
		db:           db,
		runtime:      &mockRuntime{running: map[string]bool{}, exists: map[string]bool{}, published: map[string]int{}},
	}
}

func mustFindConsecutiveFreePorts(t *testing.T, start int, count int) int {
	t.Helper()
	for port := start; port < 65535-count; port++ {
		allFree := true
		for offset := 0; offset < count; offset++ {
			if !hostPortAvailable(port + offset) {
				allFree = false
				break
			}
		}
		if allFree {
			return port
		}
	}
	t.Fatalf("no run of %d consecutive free ports found from %d", count, start)
	return 0
}
