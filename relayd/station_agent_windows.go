//go:build windows

package main

// stationAgent — Windows-side client for the long-lived vessel daemon inside WSL2.
//
// Architecture:
//   relayd (Windows)  ──TCP──►  vessel daemon (WSL2 Linux)
//                     127.0.0.1:<port>     /tmp/relay-station-agent.port
//
// WSL2 automatically forwards TCP ports bound inside the Linux VM to the
// same address on Windows, so the Windows HTTP client dials normally.
//
// Lifecycle:
//   1. ensureStationAgent() is called lazily on first WSL2 build/snapshot op.
//   2. Agent reads /tmp/relay-station-agent.port via a single wslRun call.
//   3. If the port is missing or stale, the agent starts the daemon via a
//      detached wsl.exe invocation and polls until /health responds.
//   4. All subsequent calls reuse the HTTP connection pool — no wsl.exe spawn.
//
// Fallback: if the agent cannot be initialised the caller gracefully falls
// back to the legacy wsl.exe-per-operation path already in runtime_vessel.go.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	stationAgentPortFile   = "/tmp/relay-station-agent.port"
	stationAgentStartupMax = 30 * time.Second
	stationAgentProbeTO    = 2 * time.Second

	agentContentType = "Content-Type"
	agentMIMEJSON    = "application/json"
	agentSnapshotPfx = "/snapshot/"
)

// stationAgent is the per-relayd-process singleton that tracks the daemon port
// and owns a pooled HTTP client.
type stationAgent struct {
	distro string
	port   int
	http   *http.Client
}

var (
	agentMu   sync.Mutex
	agentOnce sync.Once
	agentSing *stationAgent
	agentErr  error
)

// getStationAgent returns the initialized singleton, starting the daemon when
// needed.  Returns (nil, err) if the agent cannot be initialised; the caller
// should fall back to the legacy wsl.exe path in that case.
func getStationAgent() (*stationAgent, error) {
	agentOnce.Do(func() {
		distro := vesselWSLDistro()
		if distro == "" {
			agentErr = fmt.Errorf("no WSL2 distro available")
			return
		}
		a := &stationAgent{
			distro: distro,
			http: &http.Client{
				Timeout: 0, // no global timeout; callers use per-request contexts
				Transport: &http.Transport{
					DialContext:         (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
					MaxIdleConnsPerHost: 4,
				},
			},
		}
		agentErr = a.ensureRunning()
		if agentErr == nil {
			agentSing = a
		}
	})
	return agentSing, agentErr
}

// resetStationAgent drops the cached agent so the next getStationAgent call
// re-initialises from scratch.  Call after unrecoverable daemon errors.
func resetStationAgent() {
	agentMu.Lock()
	defer agentMu.Unlock()
	agentOnce = sync.Once{}
	agentSing = nil
	agentErr = nil
}

// ─── agent lifecycle ──────────────────────────────────────────────────────────

// ensureRunning reads the saved port and verifies the daemon is healthy, or
// starts the daemon if it is missing/stale.
func (a *stationAgent) ensureRunning() error {
	// Install station binary in WSL2 first (no-op if already installed).
	if err := a.ensureWSLBinary(); err != nil {
		return fmt.Errorf("vessel WSL2 binary: %w", err)
	}

	// Try reading the port file written by a running daemon.
	if port := a.readPortFile(); port > 0 {
		a.port = port
		if a.healthy() {
			return nil
		}
	}
	return a.startDaemon()
}

// ensureWSLBinary calls vessel.exe wsl-ensure to install/verify the Linux
// binary inside WSL2 without duplicating wslEnsureVessel logic here.
func (a *stationAgent) ensureWSLBinary() error {
	bin, err := ensureVesselBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "wsl-ensure")
	setCmdHideWindow(cmd)
	out, err := runCommandCaptured(cmd)
	if err != nil {
		return fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// readPortFile reads the daemon port from /tmp/relay-station-agent.port inside WSL2.
func (a *stationAgent) readPortFile() int {
	out, err := agentWSLRun(a.distro, "cat "+stationAgentPortFile+" 2>/dev/null")
	if err != nil {
		return 0
	}
	port, _ := strconv.Atoi(strings.TrimSpace(out))
	return port
}

// healthy sends GET /health to the daemon and returns true on 200 OK.
func (a *stationAgent) healthy() bool {
	if a.port <= 0 {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), stationAgentProbeTO)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.url("/health"), nil)
	resp, err := a.http.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// startDaemon launches vessel daemon inside WSL2 and waits for /health.
func (a *stationAgent) startDaemon() error {
	// Remove stale port file so we can detect when the new daemon writes its port.
	_, _ = agentWSLRun(a.distro, "rm -f "+stationAgentPortFile)

	launchCmd := fmt.Sprintf(
		"nohup %s daemon --port-file=%s >> /tmp/relay-station-agent.log 2>&1 &",
		wslStationBin, stationAgentPortFile,
	)
	if _, err := agentWSLRun(a.distro, launchCmd); err != nil {
		return fmt.Errorf("launch vessel daemon in WSL2: %w", err)
	}

	deadline := time.Now().Add(stationAgentStartupMax)
	for time.Now().Before(deadline) {
		if port := a.readPortFile(); port > 0 {
			a.port = port
			if a.healthy() {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("vessel daemon did not become healthy within %s", stationAgentStartupMax)
}

func (a *stationAgent) url(path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", a.port, path)
}

// agentProxyStartResp is the response from POST /proxy/start.
// Defined here rather than in vessel_agent_types.go because it is only
// consumed by the Windows agent implementation.
type agentProxyStartResp struct {
	Port int `json:"port"`
	PID  int `json:"pid"`
}

// ─── build ────────────────────────────────────────────────────────────────────

type agentBuildReq struct {
	Dockerfile   string `json:"dockerfile"`
	ContextDir   string `json:"context_dir"`
	SnapshotName string `json:"snapshot_name"`
}

// BuildDockerfile delegates a Dockerfile build to the WSL2 daemon.  Log lines
// are streamed to logw as they arrive; the manifest is returned on success.
// The daemon saves the snapshot in WSL2-native storage (ext4) — no Windows-side
// rootfs copy or manifest stub is needed.
func (a *stationAgent) BuildDockerfile(dockerfile, contextDir, snapshotName string, logw io.Writer) (*stationManifest, error) {
	body, _ := json.Marshal(agentBuildReq{
		Dockerfile:   toWSLPath(dockerfile),
		ContextDir:   toWSLPath(contextDir),
		SnapshotName: snapshotName,
	})
	ctx, cancel := context.WithTimeout(context.Background(), stationAgentBuildTimeout())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url("/build-dockerfile"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set(agentContentType, agentMIMEJSON)
	longClient := &http.Client{Transport: a.http.Transport}
	resp, err := longClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vessel agent build: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vessel agent build: status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	return readBuildStream(resp.Body, logw)
}

func stationAgentBuildTimeout() time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAY_STATION_AGENT_BUILD_TIMEOUT_SECONDS")))
	if err != nil || secs <= 0 {
		secs = 600
	}
	return time.Duration(secs) * time.Second
}

// readBuildStream reads the L:/M:/E: line protocol from the daemon build
// response, forwarding log lines to logw and returning the manifest on success.
func readBuildStream(body io.Reader, logw io.Writer) (*stationManifest, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "L:"):
			if logw != nil {
				_, _ = fmt.Fprintln(logw, line[2:])
			}
		case strings.HasPrefix(line, "M:"):
			var m stationManifest
			if err := json.Unmarshal([]byte(line[2:]), &m); err != nil {
				return nil, fmt.Errorf("decode build manifest: %w", err)
			}
			if m.Env == nil {
				m.Env = map[string]string{}
			}
			return &m, nil
		case strings.HasPrefix(line, "E:"):
			return nil, fmt.Errorf("%s", line[2:])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read build stream: %w", err)
	}
	return nil, fmt.Errorf("vessel agent build: stream ended without result")
}

// ─── snapshot operations ──────────────────────────────────────────────────────

// GetManifest returns the manifest for a snapshot stored in WSL2.
func (a *stationAgent) GetManifest(snapshotName string) (*stationManifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		a.url(agentSnapshotPfx+snapshotName+"/manifest"), nil)
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("snapshot %q not found in WSL2", snapshotName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get manifest: HTTP %d", resp.StatusCode)
	}
	var m stationManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	if m.Env == nil {
		m.Env = map[string]string{}
	}
	return &m, nil
}

// SnapshotExists returns true when the named snapshot is present in WSL2.
func (a *stationAgent) SnapshotExists(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.url(agentSnapshotPfx+name), nil)
	resp, err := a.http.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// DeleteSnapshot removes a WSL2-native snapshot by name.
func (a *stationAgent) DeleteSnapshot(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, a.url(agentSnapshotPfx+name), nil)
	resp, err := a.http.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

type agentPruneReq struct {
	Prefix string   `json:"prefix"`
	Keep   []string `json:"keep"`
}

// PruneSnapshots removes WSL2-side snapshots matching prefix, preserving those
// in keep.  Best-effort — failures are silently ignored.
func (a *stationAgent) PruneSnapshots(prefix string, keep []string) {
	body, _ := json.Marshal(agentPruneReq{Prefix: prefix, Keep: keep})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url("/snapshots/prune"), bytes.NewReader(body))
	req.Header.Set(agentContentType, agentMIMEJSON)
	resp, err := a.http.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// ─── container run / stop ────────────────────────────────────────────────────

// RunContainer starts a container from a snapshot image inside WSL2.
// Returns the container record on success.
func (a *stationAgent) RunContainer(req agentRunContainerReq) (*agentRunContainerResp, error) {
	body, _ := json.Marshal(req)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url("/container/run"), bytes.NewReader(body))
	httpReq.Header.Set(agentContentType, agentMIMEJSON)
	resp, err := a.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vessel agent run container: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vessel agent run container: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	var result agentRunContainerResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode run container response: %w", err)
	}
	return &result, nil
}

// StopApp stops all containers and the edge proxy for an app inside WSL2.
func (a *stationAgent) StopApp(name string) {
	body, _ := json.Marshal(struct {
		App string `json:"app"`
	}{App: name})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url("/container/stop"), bytes.NewReader(body))
	req.Header.Set(agentContentType, agentMIMEJSON)
	if resp, err := a.http.Do(req); err == nil {
		resp.Body.Close()
	}
}

// ─── proxy ────────────────────────────────────────────────────────────────────

// ProxyStart starts (or recreates) the edge proxy for an app inside WSL2.
// Returns the proxy listen port on success.
func (a *stationAgent) ProxyStart(req agentProxyReq) (int, error) {
	body, _ := json.Marshal(req)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url("/proxy/start"), bytes.NewReader(body))
	httpReq.Header.Set(agentContentType, agentMIMEJSON)
	resp, err := a.http.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("vessel agent proxy start: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("vessel agent proxy start: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	var result agentProxyStartResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode proxy start response: %w", err)
	}
	return result.Port, nil
}

// ProxySwap hot-swaps the upstream(s) on a running proxy inside WSL2.
func (a *stationAgent) ProxySwap(req agentProxyReq) error {
	body, _ := json.Marshal(req)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url("/proxy/swap"), bytes.NewReader(body))
	httpReq.Header.Set(agentContentType, agentMIMEJSON)
	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("vessel agent proxy swap: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vessel agent proxy swap: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// ProxyStop stops the edge proxy for an app inside WSL2.
func (a *stationAgent) ProxyStop(name string) {
	body, _ := json.Marshal(struct {
		App string `json:"app"`
	}{App: name})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url("/proxy/stop"), bytes.NewReader(body))
	req.Header.Set(agentContentType, agentMIMEJSON)
	if resp, err := a.http.Do(req); err == nil {
		resp.Body.Close()
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// agentWSLRun executes a short shell command inside the WSL2 distro.
// It is a local copy of the wslRun pattern so vessel_agent_windows.go does not
// depend on runtime_vessel.go being compiled (avoids a circular init order).
func agentWSLRun(distro, cmd string) (string, error) {
	c := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c", cmd)
	setCmdHideWindow(c)
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	// Strip null bytes that wsl.exe emits on some Windows builds (UTF-16 LE).
	filtered := make([]byte, 0, len(out))
	for _, b := range out {
		if b != 0x00 && b != 0x0D {
			filtered = append(filtered, b)
		}
	}
	return string(filtered), nil
}

// toWSLPath converts a Windows absolute path to its /mnt/X form for WSL2.
// Duplicate of the one in vessel/wsl_windows.go — kept local to avoid a
// shared-package dependency.
func toWSLPath(p string) string {
	if len(p) >= 2 && p[1] == ':' {
		drive := strings.ToLower(string(p[0]))
		rest := strings.ReplaceAll(p[2:], "\\", "/")
		return "/mnt/" + drive + rest
	}
	return strings.ReplaceAll(p, "\\", "/")
}

// wslStationBin is the canonical path of the station binary inside WSL2.
// Matches the constant in vessel/wsl_windows.go.
const wslStationBin = "/usr/local/bin/vessel"

// stationAgentHealthy is a convenience wrapper used by the engine detector.
// Returns true if the vessel daemon in the default WSL2 distro responds within
// a short probe timeout.
func stationAgentHealthy() bool {
	a, err := getStationAgent()
	return err == nil && a != nil && a.healthy()
}

// wslAvailableForAgent reports whether WSL2 and a usable distro are present.
// Lightweight check used by the engine auto-selector without full agent init.
func wslAvailableForAgent() bool {
	distro := vesselWSLDistro()
	return distro != "" && distro != "station-linux"
}

// startStationAgentBackground kicks off the agent initialisation in a goroutine
// so relayd can warm the daemon in parallel with other startup work.
func startStationAgentBackground() {
	go func() { _, _ = getStationAgent() }()
}

// Ensure os is available for stderr writes in ensureWSLBinary.
var _ = os.Stderr
