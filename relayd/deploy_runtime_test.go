package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"testing"
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

type mockRuntime struct {
	running   map[string]bool
	published map[string]int
}

func (m *mockRuntime) RunDetached(ContainerSpec) error               { return nil }
func (m *mockRuntime) Remove(string)                                 {}
func (m *mockRuntime) IsRunning(name string) bool                    { return m.running[name] }
func (m *mockRuntime) ContainerIP(string) string                     { return "" }
func (m *mockRuntime) PublishedPort(name string, _ int) int          { return m.published[name] }
func (m *mockRuntime) Exec(string, []string) ([]byte, error)         { return nil, nil }
func (m *mockRuntime) NetworkConnect(string, string) error           { return nil }
func (m *mockRuntime) EnsureNetwork(string) error                    { return nil }
func (m *mockRuntime) RemoveNetwork(string)                          {}
func (m *mockRuntime) RemoveVolume(string)                           {}
func (m *mockRuntime) Pull(string) error                             { return nil }
func (m *mockRuntime) Build(string, string, string, io.Writer) error { return nil }
func (m *mockRuntime) RemoveImage(string)                            {}
func (m *mockRuntime) ListImages(string) ([]string, error)           { return nil, nil }
func (m *mockRuntime) LogStream(context.Context, string, int, string) (io.ReadCloser, error) {
	return nil, nil
}

func newPreviewPortTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "relayd-test.db"))
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
		db:      db,
		runtime: &mockRuntime{running: map[string]bool{}, published: map[string]int{}},
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
