package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var stationBuildMu sync.Mutex

// runCommandCaptured executes cmd with stdout/stderr redirected to a temporary
// file instead of anonymous pipes. Detached vessel subprocesses on Windows can
// keep pipe handles open after the launcher exits, which makes CombinedOutput
// hang even though the container/proxy has already started.
func runCommandCaptured(cmd *exec.Cmd) ([]byte, error) {
	f, err := os.CreateTemp("", "relay-cmd-*.log")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	defer func() {
		_ = f.Close()
		_ = os.Remove(path)
	}()

	cmd.Stdout = f
	cmd.Stderr = f
	runErr := cmd.Run()
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(f)
	if readErr != nil {
		return data, readErr
	}
	return data, runErr
}

// ─── WSL2 keep-alive ─────────────────────────────────────────────────────────

var wslKeepAliveOnce sync.Once

// startWSLKeepAlive runs a background goroutine that pings the WSL2 distro
// every 20 seconds so the Hyper-V VM stays running between deploys.
// Docker does the same thing with the docker-desktop distro.
// Without this, the first wsl.exe call after an idle period takes 10–30 s to
// cold-boot the VM, which shows up as the long "[build] delegating to WSL2"
// pause at the start of every deploy.
func startWSLKeepAlive(distro string) {
	wslKeepAliveOnce.Do(func() {
		go func() {
			for {
				time.Sleep(20 * time.Second)
				c := exec.Command("wsl.exe", "-d", distro, "--", "true")
				setCmdHideWindow(c)
				_ = c.Run()
			}
		}()
	})
}

// winToWSLPath converts a Windows absolute path to the /mnt/X form used inside
// WSL2 (e.g. C:\foo\bar → /mnt/c/foo/bar).
func winToWSLPath(p string) string {
	if len(p) >= 2 && p[1] == ':' {
		drive := strings.ToLower(string(p[0]))
		rest := strings.ReplaceAll(p[2:], "\\", "/")
		return "/mnt/" + drive + rest
	}
	return strings.ReplaceAll(p, "\\", "/")
}

// shqSimple wraps s in single quotes for safe POSIX shell interpolation.
func shqSimple(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// wslBuildDirFromBreadcrumb reads the .wsl-build-dir breadcrumb that
// BuildDockerfile writes to buildDir, returning the WSL2-internal path.
func wslBuildDirFromBreadcrumb(buildDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(buildDir, ".wsl-build-dir"))
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(string(data))
	if dir == "" {
		return "", fmt.Errorf("empty wsl-build-dir breadcrumb")
	}
	return dir, nil
}

// wslSaveSnapshot runs `vessel snapshot save` inside WSL2 so the snapshot
// is created by hardlinks within ext4 — effectively instant.
func wslSaveSnapshot(distro, snapshotName, wslBuildDir string, logw io.Writer) error {
	fmt.Fprintf(logw, "[vessel] saving snapshot in WSL2 filesystem...\n")
	// Remove any existing WSL2-side snapshot first (vessel save refuses to overwrite).
	rmCmd := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c",
		"rm -rf /tmp/relay-native/snapshots/"+shqSimple(snapshotName))
	setCmdHideWindow(rmCmd)
	_ = rmCmd.Run()
	cmd := exec.Command("wsl.exe", "-d", distro, "--",
		"/usr/local/bin/vessel", "snapshot", "save", snapshotName, wslBuildDir)
	setCmdHideWindow(cmd)
	cmd.Stdout = logw
	cmd.Stderr = logw
	return cmd.Run()
}

// wslWriteWindowsManifestStub creates a minimal Windows-side snapshot
// directory containing only the manifest JSON.  This lets loadStationManifest
// work without a full Windows-side rootfs copy.
func wslWriteWindowsManifestStub(snapshotName, buildDir string) error {
	manifestData, err := os.ReadFile(filepath.Join(buildDir, "vessel-manifest.json"))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	winSnapDir := stationSnapshotDir(snapshotName)
	_ = os.RemoveAll(winSnapDir)
	if err := os.MkdirAll(winSnapDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(filepath.Join(winSnapDir, "vessel-manifest.json"), manifestData, 0644)
}

// syncSnapshotToWSL2 copies the named snapshot from the Windows store into
// WSL2's own /tmp/relay-native/snapshots directory so containers can start
// from native ext4 instead of the slow /mnt/c/ 9P path.
// It is best-effort; failures are logged but do not abort the deploy.
// Used as the slow fallback when the WSL2-native fast path is unavailable.
func syncSnapshotToWSL2(distro, snapshotName string, logw io.Writer) {
	winSnapDir := stationSnapshotDir(snapshotName)
	wslSrc := winToWSLPath(winSnapDir)
	wslDst := "/tmp/relay-native/snapshots/" + snapshotName
	fmt.Fprintf(logw, "[vessel] syncing snapshot to WSL2 filesystem...\n")
	cmd := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c",
		"rm -rf "+shqSimple(wslDst)+" && mkdir -p /tmp/relay-native/snapshots && cp -a "+shqSimple(wslSrc)+" "+shqSimple(wslDst))
	setCmdHideWindow(cmd)
	cmd.Stdout = logw
	cmd.Stderr = logw
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(logw, "[vessel] warn: WSL2 snapshot sync failed: %v\n", err)
	}
}

type stationManifest struct {
	Cmd        []string          `json:"cmd"`
	Entrypoint []string          `json:"entrypoint,omitempty"`
	Env        map[string]string `json:"env"`
	Port       int               `json:"port"`
	WorkDir    string            `json:"workdir"`
}

type stationContainerRecord struct {
	ID      string    `json:"id"`
	App     string    `json:"app"`
	PID     int       `json:"pid"`
	Port    int       `json:"port"`
	NetMode string    `json:"net_mode,omitempty"`
	IP      string    `json:"ip,omitempty"`
	Image   string    `json:"image,omitempty"`
	Started time.Time `json:"started"`
}

type stationProxyRecord struct {
	App       string `json:"app"`
	ProxyPort int    `json:"proxy_port"`
	PID       int    `json:"pid,omitempty"`
}

type StationRuntime struct {
	volumeBaseDir string
}

func stationBinaryName() string {
	if runtime.GOOS == "windows" {
		return "station.exe"
	}
	return "vessel"
}

func stationStateBaseDir() string {
	return filepath.Join(os.TempDir(), "relay-native")
}

func stationSnapshotStoreDir() string {
	return filepath.Join(stationStateBaseDir(), "snapshots")
}

func stationSnapshotDir(name string) string {
	return filepath.Join(stationSnapshotStoreDir(), name)
}

func stationSnapshotManifestPath(name string) string {
	return filepath.Join(stationSnapshotDir(name), "vessel-manifest.json")
}

func vesselContainerConfigPath(id string) string {
	return filepath.Join(stationStateBaseDir(), "containers", id, "config.json")
}

func vesselContainerLogPath(id string) string {
	return filepath.Join(stationStateBaseDir(), "containers", id, "output.log")
}

func vesselProxyConfigPath(app string) string {
	return filepath.Join(stationStateBaseDir(), "proxies", app, "proxy.json")
}

func vesselProxyLogPath(app string) string {
	return filepath.Join(stationStateBaseDir(), "proxies", app, "proxy.log")
}

func newStationRuntime(dataDir string) *StationRuntime {
	base := filepath.Join(stationStateBaseDir(), "volumes")
	if strings.TrimSpace(dataDir) != "" {
		base = filepath.Join(dataDir, "vessel-volumes")
	}
	// Start the vessel daemon in the background so it is warm before the first
	// deploy.  The daemon process itself keeps the WSL2 VM alive — no separate
	// keep-alive ping loop is needed.
	if runtime.GOOS == "windows" {
		startStationAgentBackground()
	}
	return &StationRuntime{volumeBaseDir: base}
}

func stationAppName(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("relay__%s__%s__%s__app", safe(app), safe(string(env)), safe(branch))
}

func vesselProxyName(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("relay__%s__%s__%s__proxy", safe(app), safe(string(env)), safe(branch))
}

func stationSnapshotName(app string, env DeployEnv, branch, deployID string) string {
	return fmt.Sprintf("relay__%s__%s__%s__%s", safe(app), safe(string(env)), safe(branch), safe(deployID))
}

func stationSnapshotPrefix(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("relay__%s__%s__%s__", safe(app), safe(string(env)), safe(branch))
}

func vesselSourceDirCandidates() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 6)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}

	add(os.Getenv("RELAY_VESSEL_SOURCE_DIR"))
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		add(filepath.Join(exeDir, "..", "vessel"))
		add(filepath.Join(exeDir, "vessel"))
	}
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "..", "vessel"))
		add(filepath.Join(wd, "vessel"))
	}
	return out
}

func validVesselSourceDir(dir string) bool {
	return fileExists(filepath.Join(dir, "go.mod")) && fileExists(filepath.Join(dir, "main.go"))
}

func findVesselSourceDir() string {
	for _, dir := range vesselSourceDirCandidates() {
		if validVesselSourceDir(dir) {
			return dir
		}
	}
	return ""
}

func vesselSourceNewerThanBinary(sourceDir, binaryPath string) bool {
	info, err := os.Stat(binaryPath)
	if err != nil {
		return true
	}
	binaryMod := info.ModTime()
	newer := false
	_ = filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".go") || base == "go.mod" || base == "go.sum" {
			if info.ModTime().After(binaryMod) {
				newer = true
			}
		}
		return nil
	})
	return newer
}

func ensureVesselBinary() (string, error) {
	if override := strings.TrimSpace(os.Getenv("RELAY_VESSEL_BIN")); override != "" {
		if fileExists(override) {
			return filepath.Abs(override)
		}
		return "", fmt.Errorf("RELAY_VESSEL_BIN points to a missing file: %s", override)
	}

	if sourceDir := findVesselSourceDir(); sourceDir != "" {
		binaryPath := filepath.Join(sourceDir, stationBinaryName())
		stationBuildMu.Lock()
		defer stationBuildMu.Unlock()
		if vesselSourceNewerThanBinary(sourceDir, binaryPath) {
			cmd := exec.Command("go", "build", "-o", binaryPath, ".")
			cmd.Dir = sourceDir
			setCmdHideWindow(cmd)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("build vessel from %s: %v (%s)", sourceDir, err, strings.TrimSpace(string(out)))
			}
		}
		return binaryPath, nil
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), stationBinaryName())
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("vessel source/binary not found; keep vessel beside relayd or set RELAY_VESSEL_BIN")
}

func loadStationManifest(snapshotName string) (*stationManifest, error) {
	data, err := os.ReadFile(stationSnapshotManifestPath(snapshotName))
	if err != nil {
		return nil, err
	}
	var manifest stationManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Env == nil {
		manifest.Env = map[string]string{}
	}
	return &manifest, nil
}

func vesselCommand(manifest *stationManifest) ([]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("missing vessel manifest")
	}
	cmd := make([]string, 0, len(manifest.Entrypoint)+len(manifest.Cmd))
	cmd = append(cmd, manifest.Entrypoint...)
	cmd = append(cmd, manifest.Cmd...)
	if len(cmd) == 0 {
		return nil, fmt.Errorf("vessel snapshot has no entrypoint/cmd")
	}
	return cmd, nil
}

func mergeVesselEnv(manifestEnv, extraEnv map[string]string) []string {
	merged := map[string]string{}
	for key, value := range manifestEnv {
		merged[key] = value
	}
	for key, value := range extraEnv {
		merged[key] = value
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, merged[key]))
	}
	return pairs
}

func stationContainerRunning(rec *stationContainerRecord) bool {
	return rec != nil && rec.PID > 0 && pidAlive(rec.PID)
}

func loadVesselContainersByApp(appName string) ([]stationContainerRecord, error) {
	matches, err := filepath.Glob(filepath.Join(stationStateBaseDir(), "containers", "*", "config.json"))
	if err != nil {
		return nil, err
	}
	out := make([]stationContainerRecord, 0, len(matches))
	for _, match := range matches {
		data, readErr := os.ReadFile(match)
		if readErr != nil {
			continue
		}
		var rec stationContainerRecord
		if json.Unmarshal(data, &rec) != nil {
			continue
		}
		if rec.App != appName {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Started.After(out[j].Started)
	})
	return out, nil
}

func latestStationContainerByApp(appName string) (*stationContainerRecord, error) {
	records, err := loadVesselContainersByApp(appName)
	if err != nil || len(records) == 0 {
		return nil, err
	}
	for i := range records {
		if stationContainerRunning(&records[i]) {
			return &records[i], nil
		}
	}
	return &records[0], nil
}

func loadVesselContainerByID(id string) (*stationContainerRecord, error) {
	data, err := os.ReadFile(vesselContainerConfigPath(id))
	if err != nil {
		return nil, err
	}
	var rec stationContainerRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func loadVesselProxyRecord(app string) *stationProxyRecord {
	data, err := os.ReadFile(vesselProxyConfigPath(app))
	if err != nil {
		return nil
	}
	var rec stationProxyRecord
	if json.Unmarshal(data, &rec) != nil {
		return nil
	}
	return &rec
}

type vesselPortBinding struct {
	HostPort      int
	ContainerPort int
}

func parseVesselPortBinding(binding string) vesselPortBinding {
	binding = strings.TrimSpace(binding)
	if binding == "" {
		return vesselPortBinding{}
	}
	parts := strings.Split(binding, ":")
	last := strings.TrimSpace(parts[len(parts)-1])
	containerPort, _ := strconv.Atoi(last)
	if len(parts) == 1 {
		return vesselPortBinding{ContainerPort: containerPort}
	}
	hostPart := strings.TrimSpace(strings.Join(parts[:len(parts)-1], ":"))
	if hostPart == "" {
		return vesselPortBinding{ContainerPort: containerPort}
	}
	hostPortText := hostPart
	if idx := strings.LastIndex(hostPart, ":"); idx >= 0 {
		hostPortText = strings.TrimSpace(hostPart[idx+1:])
	}
	hostPort, _ := strconv.Atoi(hostPortText)
	return vesselPortBinding{
		HostPort:      hostPort,
		ContainerPort: containerPort,
	}
}

func vesselPortFromEnv(envs []string) int {
	for _, pair := range envs {
		if !strings.HasPrefix(pair, "PORT=") {
			continue
		}
		port, _ := strconv.Atoi(strings.TrimPrefix(pair, "PORT="))
		if port > 0 {
			return port
		}
	}
	return 0
}

func vesselSpecPort(spec ContainerSpec) int {
	for _, binding := range spec.PortBindings {
		if port := parseVesselPortBinding(binding).ContainerPort; port > 0 {
			return port
		}
	}
	return vesselPortFromEnv(spec.Env)
}

func vesselFindFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

func vesselEnvWithResolvedPort(envs []string, port int) []string {
	if port <= 0 {
		return append([]string{}, envs...)
	}
	out := make([]string, 0, len(envs)+1)
	replaced := false
	for _, pair := range envs {
		if strings.HasPrefix(pair, "PORT=") {
			if !replaced {
				out = append(out, fmt.Sprintf("PORT=%d", port))
				replaced = true
			}
			continue
		}
		out = append(out, pair)
	}
	if !replaced {
		out = append(out, fmt.Sprintf("PORT=%d", port))
	}
	return out
}

func vesselResolvedRunPort(spec ContainerSpec) (int, []string, error) {
	envs := append([]string{}, spec.Env...)
	bridgeMode := spec.Network != "" && runtime.GOOS != "windows"
	for _, binding := range spec.PortBindings {
		parsed := parseVesselPortBinding(binding)
		if parsed.ContainerPort <= 0 {
			continue
		}
		if bridgeMode {
			return parsed.ContainerPort, vesselEnvWithResolvedPort(envs, parsed.ContainerPort), nil
		}
		hostPort := parsed.HostPort
		if hostPort <= 0 {
			var err error
			hostPort, err = vesselFindFreePort()
			if err != nil {
				return 0, nil, err
			}
		}
		return hostPort, vesselEnvWithResolvedPort(envs, hostPort), nil
	}
	return vesselPortFromEnv(envs), envs, nil
}

func stationRuntimeLogPath(name string) string {
	if rec := loadVesselProxyRecord(name); rec != nil {
		return vesselProxyLogPath(name)
	}
	rec, err := latestStationContainerByApp(name)
	if err != nil || rec == nil {
		return ""
	}
	return vesselContainerLogPath(rec.ID)
}

func vesselSlotUpstream(runtime ContainerRuntime, name string, port int) string {
	if ip := strings.TrimSpace(runtime.ContainerIP(name)); ip != "" {
		return net.JoinHostPort(ip, strconv.Itoa(firstNonZero(port, 3000)))
	}
	hostPort := runtime.PublishedPort(name, port)
	if hostPort <= 0 {
		hostPort = firstNonZero(port, 3000)
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(hostPort))
}

func stripWSLNulls(b []byte) string {
	out := make([]byte, 0, len(b))
	for _, ch := range b {
		if ch != 0 && ch != '\r' {
			out = append(out, ch)
		}
	}
	return string(out)
}

func vesselWSLDistro() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	wslListCmd := exec.Command("wsl.exe", "--list", "--quiet")
	setCmdHideWindow(wslListCmd)
	out, err := wslListCmd.CombinedOutput()
	if err != nil {
		return "station-linux"
	}
	var fallback string
	for _, line := range strings.Split(stripWSLNulls(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if fallback == "" {
			fallback = line
		}
		name := strings.ToLower(line)
		if name == "docker-desktop" || name == "docker-desktop-data" {
			continue
		}
		return line
	}
	if fallback != "" {
		return fallback
	}
	return "station-linux"
}

func (r *StationRuntime) probeBridgeAddress(ip string, port int) bool {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), 2*time.Second)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

func (r *StationRuntime) bridgeReadyStable(name string, min time.Duration) bool {
	rec, err := latestStationContainerByApp(name)
	if err != nil || rec == nil {
		return false
	}
	if strings.TrimSpace(rec.IP) == "" || !stationContainerRunning(rec) {
		return false
	}
	return time.Since(rec.Started) >= min
}

func (r *StationRuntime) readyByLog(name string) bool {
	path := stationRuntimeLogPath(name)
	if strings.TrimSpace(path) == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}
	if len(data) > 64*1024 {
		data = data[len(data)-64*1024:]
	}
	text := strings.ToLower(string(data))
	readyPos := -1
	for _, marker := range []string{"ready in", "listening on", "server started", "started server", "http://localhost:"} {
		if idx := strings.LastIndex(text, marker); idx > readyPos {
			readyPos = idx
		}
	}
	if readyPos < 0 {
		return false
	}
	fatalPos := -1
	for _, marker := range []string{"error: spawn:", "cannot find module", "module_not_found", "enoent:", "panic:"} {
		if idx := strings.LastIndex(text, marker); idx > fatalPos {
			fatalPos = idx
		}
	}
	return readyPos > fatalPos
}

func waitForPort(log func(string, ...any), port int, timeout time.Duration, label string) error {
	deadline := time.Now().Add(timeout)
	for attempts := 0; time.Now().Before(deadline); attempts++ {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 2*time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if log != nil && attempts > 0 && attempts%10 == 0 {
			log("waiting for %s on port %d", label, port)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s did not become ready on port %d within %s", label, port, timeout)
}

func vesselExec(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	setCmdHideWindow(cmd)
	out, err := runCommandCaptured(cmd)
	if err != nil {
		return string(out), fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func (r *StationRuntime) binary() (string, error) {
	return ensureVesselBinary()
}

func (r *StationRuntime) command(bin string, args ...string) *exec.Cmd {
	cmd := exec.Command(bin, args...)
	setCmdHideWindow(cmd)
	if base := strings.TrimSpace(r.volumeBaseDir); base != "" {
		cmd.Env = append(os.Environ(), "VESSEL_VOLUME_BASE="+base)
	}
	return cmd
}

func (r *StationRuntime) run(args ...string) ([]byte, error) {
	bin, err := r.binary()
	if err != nil {
		return nil, err
	}
	cmd := r.command(bin, args...)
	return runCommandCaptured(cmd)
}

func (r *StationRuntime) runWithTimeout(timeout time.Duration, args ...string) ([]byte, error) {
	bin, err := r.binary()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	setCmdHideWindow(cmd)
	if base := strings.TrimSpace(r.volumeBaseDir); base != "" {
		cmd.Env = append(os.Environ(), "VESSEL_VOLUME_BASE="+base)
	}
	return runCommandCaptured(cmd)
}

func (r *StationRuntime) volumePath(name string) string {
	base := r.volumeBaseDir
	if strings.TrimSpace(base) == "" {
		base = filepath.Join(stationStateBaseDir(), "volumes")
	}
	return filepath.Join(base, safe(name))
}

// runDetachedViaAgent tries to start the container through the WSL2 daemon.
// Returns (true, nil) on success, (true, err) on a hard error that should be
// returned immediately, or (false, nil) when the agent is unavailable and the
// caller should fall through to the legacy wsl.exe path.
func runDetachedViaAgent(spec ContainerSpec) (bool, error) {
	agent, err := getStationAgent()
	if err != nil || agent == nil {
		return false, nil
	}
	netMode := ""
	if len(spec.PortBindings) > 0 {
		netMode = "host" // published ports must bind in the WSL host namespace
	}
	port, envs, portErr := vesselResolvedRunPort(spec)
	if portErr != nil {
		return true, fmt.Errorf("resolve vessel port for %s: %w", spec.Name, portErr)
	}
	req := agentRunContainerReq{
		App:        spec.Name,
		Image:      spec.Image,
		Command:    spec.Command,
		Env:        envs,
		Volumes:    spec.Volumes,
		ExtraHosts: spec.ExtraHosts,
		NetMode:    netMode,
		Restart:    spec.RestartPolicy,
		Port:       port,
	}
	if _, runErr := agent.RunContainer(req); runErr == nil {
		return true, nil
	}
	resetStationAgent()
	return false, nil
}

func (r *StationRuntime) RunDetached(spec ContainerSpec) error {
	// ── Windows fast path: delegate to the long-lived WSL2 daemon ────────────
	if ok, err := runDetachedViaAgent(spec); ok {
		return err
	}

	// ── Legacy path (non-Windows or daemon unavailable) ───────────────────────
	args := []string{"run", "--app", spec.Name}
	if spec.RestartPolicy != "" {
		args = append(args, "--restart", spec.RestartPolicy)
	}
	if spec.Network != "" && runtime.GOOS != "windows" {
		// On Windows, app containers run via WSL2 while the edge proxy runs on
		// the host process. Using bridge mode yields WSL-only 10.88.x.x upstreams
		// that the host proxy cannot dial reliably, causing 502 responses.
		// Keep Linux bridge mode on non-Windows platforms.
		args = append(args, "--net", "bridge")
	} else if runtime.GOOS == "windows" && len(spec.PortBindings) > 0 {
		// Published ports must bind in the WSL host namespace so Windows can
		// reach them via localhost forwarding. The Linux-side auto-bridge mode
		// keeps the listener inside an internal namespace that Windows cannot dial.
		args = append(args, "--net", "host")
	}
	port, envs, err := vesselResolvedRunPort(spec)
	if err != nil {
		return fmt.Errorf("resolve vessel port for %s: %w", spec.Name, err)
	}
	if port > 0 {
		args = append(args, "--port", strconv.Itoa(port))
	}
	for _, envPair := range envs {
		args = append(args, "--env", envPair)
	}
	for _, volume := range spec.Volumes {
		args = append(args, "--volume", volume)
	}
	for _, host := range spec.ExtraHosts {
		args = append(args, "--add-host", host)
	}
	if spec.Image != "" {
		args = append(args, "--image", spec.Image)
	}
	args = append(args, spec.Command...)

	out, err := r.runWithTimeout(2*time.Minute, args...)
	if err != nil {
		return fmt.Errorf("vessel run %s: %v (%s)", spec.Name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *StationRuntime) Remove(name string) {
	// Agent fast path: single HTTP call instead of multiple wsl.exe spawns.
	// getStationAgent() returns (nil, nil) on non-Windows.
	if agent, err := getStationAgent(); err == nil && agent != nil {
		agent.StopApp(name)
		return
	}
	// Legacy path.
	if bin, err := r.binary(); err == nil {
		_, _ = vesselExec(bin, "proxy", "stop", name)
		records, _ := loadVesselContainersByApp(name)
		for _, rec := range records {
			_, _ = vesselExec(bin, "stop", rec.ID)
		}
	}
}

func (r *StationRuntime) IsRunning(name string) bool {
	if rec, err := latestStationContainerByApp(name); err == nil && stationContainerRunning(rec) {
		return true
	}
	if proxy := loadVesselProxyRecord(name); proxy != nil && proxy.PID > 0 && pidAlive(proxy.PID) {
		return true
	}
	return false
}

func (r *StationRuntime) ContainerIP(name string) string {
	rec, err := latestStationContainerByApp(name)
	if err != nil || rec == nil {
		return ""
	}
	return strings.TrimSpace(rec.IP)
}

func (r *StationRuntime) PublishedPort(name string, containerPort int) int {
	if proxy := loadVesselProxyRecord(name); proxy != nil {
		return proxy.ProxyPort
	}
	rec, err := latestStationContainerByApp(name)
	if err != nil || rec == nil {
		return 0
	}
	if strings.TrimSpace(rec.IP) != "" && strings.EqualFold(strings.TrimSpace(rec.NetMode), "bridge") {
		return 0
	}
	return rec.Port
}

func (r *StationRuntime) Exec(container string, cmd []string) ([]byte, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("vessel exec requires a command")
	}
	target := strings.TrimSpace(container)
	if target == "" {
		return nil, fmt.Errorf("vessel exec requires a container name")
	}
	if rec, err := loadVesselContainerByID(target); err == nil && stationContainerRunning(rec) {
		target = rec.ID
	} else if rec, err := latestStationContainerByApp(target); err == nil && rec != nil && stationContainerRunning(rec) {
		target = rec.ID
	} else if proxy := loadVesselProxyRecord(target); proxy != nil && proxy.PID > 0 && pidAlive(proxy.PID) {
		return nil, fmt.Errorf("vessel exec does not support proxy process %s", container)
	} else {
		return nil, fmt.Errorf("no running vessel container for %s", container)
	}
	args := append([]string{"exec", target}, cmd...)
	return r.run(args...)
}

func (r *StationRuntime) NetworkConnect(container, network string) error {
	return nil
}

func (r *StationRuntime) EnsureNetwork(name string) error {
	return nil
}

func (r *StationRuntime) RemoveNetwork(name string) {}

func (r *StationRuntime) RemoveVolume(name string) {
	if strings.TrimSpace(name) == "" {
		return
	}
	_ = os.RemoveAll(r.volumePath(name))
}

func (r *StationRuntime) Build(tag, contextDir, dockerfilePath string, logw io.Writer) error {
	bin, err := r.binary()
	if err != nil {
		return err
	}
	df := dockerfilePath
	if strings.TrimSpace(df) == "" {
		df = filepath.Join(contextDir, "Dockerfile")
	}
	buildDir := filepath.Join(stationStateBaseDir(), "runtime-builds", safe(tag))
	_ = os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(buildDir)
	buildCmd := r.command(bin, "build-dockerfile", df, contextDir, buildDir)
	buildCmd.Dir = contextDir
	buildCmd.Stdout = logw
	buildCmd.Stderr = logw
	if err := buildCmd.Run(); err != nil {
		return err
	}
	saveCmd := r.command(bin, "snapshot", "save", tag, buildDir)
	saveCmd.Dir = contextDir
	saveCmd.Stdout = logw
	saveCmd.Stderr = logw
	return saveCmd.Run()
}

func (r *StationRuntime) RemoveImage(ref string) {
	if strings.TrimSpace(ref) == "" {
		return
	}
	_ = os.RemoveAll(stationSnapshotDir(ref))
}

func (r *StationRuntime) ListImages(repo string) ([]string, error) {
	entries, err := os.ReadDir(stationSnapshotStoreDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func tailLines(text string, tail int) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if tail > 0 && len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}
	return lines
}

func followFileStream(ctx context.Context, path string, tail int) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		var data []byte
		var err error
		for start := time.Now(); ; {
			data, err = os.ReadFile(path)
			if err == nil {
				break
			}
			if time.Since(start) > 10*time.Second {
				_ = pw.CloseWithError(fmt.Errorf("log not found"))
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
		}

		for _, line := range tailLines(string(data), tail) {
			if strings.TrimSpace(line) == "" {
				continue
			}
			_, _ = fmt.Fprintln(pw, line)
		}

		f, err := os.Open(path)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		defer f.Close()
		if _, err := f.Seek(int64(len(data)), io.SeekStart); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		reader := bufio.NewReaderSize(f, 64*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line, readErr := reader.ReadString('\n')
			if len(line) > 0 {
				trimmed := strings.TrimRight(line, "\r\n")
				if trimmed != "" {
					_, _ = fmt.Fprintln(pw, trimmed)
				}
			}
			if readErr == io.EOF {
				time.Sleep(250 * time.Millisecond)
				continue
			}
			if readErr != nil {
				_ = pw.CloseWithError(readErr)
				return
			}
		}
	}()
	return pr, nil
}

func (r *StationRuntime) LogStream(ctx context.Context, name string, tail int, since string) (io.ReadCloser, error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		path := stationRuntimeLogPath(name)
		if path != "" {
			return followFileStream(ctx, path, tail)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("log not found")
		}
		if ctx != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (s *Server) buildStationSnapshot(repoDir, dockerfilePath, snapshotName string, logw io.Writer) (*stationManifest, error) {
	// ── Fast path: delegate entirely to the long-lived WSL2 daemon ───────────
	// The daemon builds and saves the snapshot in WSL2-native ext4 storage in
	// one round-trip, returning the manifest over HTTP.  No Windows-side rootfs
	// copy, no manifest stub, no separate wsl.exe snapshot-save call.
	if runtime.GOOS == "windows" {
		if agent, err := getStationAgent(); err == nil && agent != nil {
			fmt.Fprintf(logw, "[build] delegating to vessel daemon (WSL2)\n")
			m, agentErr := agent.BuildDockerfile(dockerfilePath, repoDir, snapshotName, logw)
			if agentErr == nil {
				return m, nil
			}
			fmt.Fprintf(logw, "[vessel] warn: daemon build failed (%v); falling back to wsl.exe path\n", agentErr)
			resetStationAgent()
		}
	}

	// ── Legacy path (non-Windows or daemon unavailable) ───────────────────────
	bin, err := ensureVesselBinary()
	if err != nil {
		return nil, err
	}
	vr, ok := s.runtimeForEngine(EngineStation).(*StationRuntime)
	if !ok || vr == nil {
		vr = newStationRuntime(s.dataDir)
	}

	buildDir := filepath.Join(s.dataDir, "vessel-builds", snapshotName)
	_ = os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("create vessel build dir: %w", err)
	}
	defer os.RemoveAll(buildDir)

	buildCmd := vr.command(bin, "build-dockerfile", dockerfilePath, repoDir, buildDir)
	buildCmd.Dir = repoDir
	buildCmd.Stdout = logw
	buildCmd.Stderr = logw
	if err := buildCmd.Run(); err != nil {
		return nil, fmt.Errorf("vessel build-dockerfile failed: %w", err)
	}

	// Windows WSL2 fast-save: if the build wrote output to a WSL2-internal
	// directory, save the snapshot there via hardlinks (ext4 → ext4, instant).
	if runtime.GOOS == "windows" {
		if wslBuildDir, err := wslBuildDirFromBreadcrumb(buildDir); err == nil {
			if distro := vesselWSLDistro(); distro != "" {
				if snapErr := wslSaveSnapshot(distro, snapshotName, wslBuildDir, logw); snapErr == nil {
					if serr := wslWriteWindowsManifestStub(snapshotName, buildDir); serr != nil {
						fmt.Fprintf(logw, "[vessel] warn: manifest stub: %v\n", serr)
					}
					cleanCmd := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c",
						"rm -rf "+shqSimple(wslBuildDir))
					setCmdHideWindow(cleanCmd)
					_ = cleanCmd.Run()
					return loadStationManifest(snapshotName)
				}
				fmt.Fprintf(logw, "[vessel] warn: WSL2 fast snapshot save failed; falling back to Windows save\n")
			}
		}
	}

	// Standard path: save snapshot on Windows, then sync to WSL2.
	_ = os.RemoveAll(stationSnapshotDir(snapshotName))
	saveCmd := vr.command(bin, "snapshot", "save", snapshotName, buildDir)
	saveCmd.Dir = repoDir
	saveCmd.Stdout = logw
	saveCmd.Stderr = logw
	if err := saveCmd.Run(); err != nil {
		return nil, fmt.Errorf("vessel snapshot save failed: %w", err)
	}
	if runtime.GOOS == "windows" {
		if distro := vesselWSLDistro(); distro != "" {
			syncSnapshotToWSL2(distro, snapshotName, logw)
		}
	}

	return loadStationManifest(snapshotName)
}

func (s *Server) stopStationLane(app string, env DeployEnv, branch string) error {
	runtime := s.runtimeForEngine(EngineStation)
	for _, name := range []string{
		appBaseContainerName(app, env, branch),
		appSlotContainerName(app, env, branch, "blue"),
		appSlotContainerName(app, env, branch, "green"),
		stationAppName(app, env, branch),
		vesselProxyName(app, env, branch),
	} {
		runtime.Remove(name)
	}
	return nil
}

// vesselProxyParams bundles the arguments shared by all proxy-related helpers,
// keeping individual function signatures under the 7-parameter limit.
type vesselProxyParams struct {
	app         string
	env         DeployEnv
	branch      string
	activeSlot  string
	standbySlot string
	servicePort int
	hostPort    int
	mode        string
	trafficMode string
	publicHost  string
	recreate    bool
}

// ensureVesselEdgeProxyViaAgent attempts to start or swap the edge proxy
// through the long-lived WSL2 daemon.  Returns (true, err) when the agent
// handled the request, or (false, nil) to fall through to the legacy path.
func (s *Server) ensureVesselEdgeProxyViaAgent(log func(string, ...any), vrt ContainerRuntime, p vesselProxyParams) (bool, error) {
	agent, agentErr := getStationAgent()
	if agentErr != nil || agent == nil {
		return false, nil
	}
	proxyName := appBaseContainerName(p.app, p.env, p.branch)
	targetPort := firstNonZero(p.hostPort, defaultHostPort(p.env))
	activeUpstream := vesselSlotUpstream(vrt, appSlotContainerName(p.app, p.env, p.branch, p.activeSlot), p.servicePort)

	var proxyOpErr error
	if p.recreate || !vrt.IsRunning(proxyName) {
		proxyOpErr = agentProxyStart(agent, vrt, proxyName, activeUpstream, targetPort, p)
	} else {
		proxyOpErr = agentProxySwap(agent, vrt, proxyName, activeUpstream, p)
	}
	if proxyOpErr != nil {
		resetStationAgent()
		return false, nil
	}
	if p.recreate {
		return true, s.waitForRuntimeContainerReady(vrt, log, proxyName, targetPort, 10*time.Second)
	}
	return true, nil
}

func agentProxyStart(agent *stationAgent, vrt ContainerRuntime, proxyName, activeUpstream string, targetPort int, p vesselProxyParams) error {
	vrt.Remove(proxyName)
	req := agentProxyReq{
		App:            proxyName,
		Port:           targetPort,
		ActiveUpstream: activeUpstream,
		ActiveSlot:     p.activeSlot,
		TrafficMode:    firstNonEmpty(normalizeTrafficMode(p.trafficMode), "edge"),
		CookieName:     edgeCookieName(p.app, p.env, p.branch),
	}
	if p.standbySlot != "" {
		req.StandbyUpstream = vesselSlotUpstream(vrt, appSlotContainerName(p.app, p.env, p.branch, p.standbySlot), p.servicePort)
		req.StandbySlot = p.standbySlot
	}
	if strings.ToLower(strings.TrimSpace(p.mode)) == "traefik" && strings.TrimSpace(p.publicHost) != "" {
		req.PublicHost = strings.TrimSpace(p.publicHost)
	}
	_, err := agent.ProxyStart(req)
	return err
}

func agentProxySwap(agent *stationAgent, vrt ContainerRuntime, proxyName, activeUpstream string, p vesselProxyParams) error {
	req := agentProxyReq{
		App:            proxyName,
		ActiveUpstream: activeUpstream,
		ActiveSlot:     p.activeSlot,
		TrafficMode:    firstNonEmpty(normalizeTrafficMode(p.trafficMode), "edge"),
		CookieName:     edgeCookieName(p.app, p.env, p.branch),
	}
	if p.standbySlot != "" {
		req.StandbyUpstream = vesselSlotUpstream(vrt, appSlotContainerName(p.app, p.env, p.branch, p.standbySlot), p.servicePort)
		req.StandbySlot = p.standbySlot
	} else {
		req.ClearStandby = true
	}
	if strings.ToLower(strings.TrimSpace(p.mode)) == "traefik" && strings.TrimSpace(p.publicHost) != "" {
		req.PublicHost = strings.TrimSpace(p.publicHost)
	} else if !p.recreate {
		req.ClearPublicHost = true
	}
	return agent.ProxySwap(req)
}

func (s *Server) ensureVesselEdgeProxy(log func(string, ...any), app string, env DeployEnv, branch string, activeSlot string, standbySlot string, servicePort int, hostPort int, mode string, trafficMode string, publicHost string, recreate bool) error {
	runtime := s.runtimeForEngine(EngineStation)
	activeSlot = normalizeActiveSlot(activeSlot)
	standbySlot = normalizeActiveSlot(standbySlot)
	if standbySlot != "" && !runtime.IsRunning(appSlotContainerName(app, env, branch, standbySlot)) {
		standbySlot = ""
		if st, err := s.getAppState(app, env, branch); err == nil && st != nil && normalizeActiveSlot(st.ActiveSlot) == activeSlot {
			st.StandbySlot = ""
			st.DrainUntil = 0
			_ = s.saveAppState(st)
			s.broadcastSnapshot()
		}
	}

	// ── Agent fast path ───────────────────────────────────────────────────────
	if done, err := s.ensureVesselEdgeProxyViaAgent(log, runtime, vesselProxyParams{
		app: app, env: env, branch: branch,
		activeSlot: activeSlot, standbySlot: standbySlot,
		servicePort: servicePort, hostPort: hostPort,
		mode: mode, trafficMode: trafficMode, publicHost: publicHost,
		recreate: recreate,
	}); done {
		return err
	}

	bin, err := ensureVesselBinary()
	if err != nil {
		return err
	}

	proxyName := appBaseContainerName(app, env, branch)
	activeUpstream := vesselSlotUpstream(runtime, appSlotContainerName(app, env, branch, activeSlot), servicePort)
	args := []string{"proxy"}
	if recreate || !runtime.IsRunning(proxyName) {
		runtime.Remove(proxyName)
		args = append(args, "start", "--app", proxyName, "--port", strconv.Itoa(firstNonZero(hostPort, defaultHostPort(env))))
	} else {
		args = append(args, "swap", "--app", proxyName)
	}
	args = append(args,
		"--active-upstream", activeUpstream,
		"--active-slot", activeSlot,
		"--traffic-mode", firstNonEmpty(normalizeTrafficMode(trafficMode), "edge"),
		"--cookie-name", edgeCookieName(app, env, branch),
	)
	if standbySlot != "" {
		args = append(args,
			"--standby-upstream", vesselSlotUpstream(runtime, appSlotContainerName(app, env, branch, standbySlot), servicePort),
			"--standby-slot", standbySlot,
		)
	} else {
		args = append(args, "--clear-standby")
	}
	if strings.ToLower(strings.TrimSpace(mode)) == "traefik" && strings.TrimSpace(publicHost) != "" {
		args = append(args, "--public-host", strings.TrimSpace(publicHost))
	} else if !recreate {
		args = append(args, "--clear-public-host")
	}

	if out, err := vesselExec(bin, args...); err != nil {
		return fmt.Errorf("vessel proxy update failed: %w", err)
	} else if log != nil && strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			log(line)
		}
	}
	if recreate {
		return s.waitForRuntimeContainerReady(runtime, log, proxyName, firstNonZero(hostPort, defaultHostPort(env)), 10*time.Second)
	}
	return nil
}

func (s *Server) cleanupVesselStandbySlotAfter(app string, env DeployEnv, branch string, activeSlot string, oldSlot string, servicePort int, hostPort int, mode string, trafficMode string, publicHost string, wait time.Duration) {
	runtime := s.runtimeForEngine(EngineStation)
	name := appSlotContainerName(app, env, branch, oldSlot)
	cleanup := func() {
		runtime.Remove(name)
		_ = s.ensureVesselEdgeProxy(nil, app, env, branch, activeSlot, "", servicePort, hostPort, mode, trafficMode, publicHost, false)
		if st, err := s.getAppState(app, env, branch); err == nil && st != nil {
			if normalizeActiveSlot(st.ActiveSlot) == normalizeActiveSlot(activeSlot) && normalizeActiveSlot(st.StandbySlot) == normalizeActiveSlot(oldSlot) {
				st.StandbySlot = ""
				st.DrainUntil = 0
				_ = s.saveAppState(st)
				s.broadcastSnapshot()
			}
		}
	}
	if wait <= 0 {
		cleanup()
		return
	}
	go func() {
		time.Sleep(wait)
		cleanup()
	}()
}

func (s *Server) runStationApp(log func(string, ...any), req DeployRequest, snapshotName string, extraEnv map[string]string) error {
	runtime := s.runtimeForEngine(EngineStation)
	servicePort := firstNonZero(req.ServicePort, 3000)
	hostPort := firstNonZero(req.HostPort, defaultHostPort(req.Env))
	mode := firstNonEmpty(strings.ToLower(strings.TrimSpace(req.Mode)), "port")
	trafficMode := firstNonEmpty(normalizeTrafficMode(req.TrafficMode), "edge")
	networkName := appNetworkName(req.App, req.Env, req.Branch)
	if err := runtime.EnsureNetwork(networkName); err != nil {
		return fmt.Errorf("ensure vessel network: %w", err)
	}

	state, _ := s.getAppState(req.App, req.Env, req.Branch)
	activeSlot := s.currentActiveSlotWithRuntime(runtime, req.App, req.Env, req.Branch, state)
	nextSlot := nextActiveSlot(activeSlot)
	candidateName := appSlotContainerName(req.App, req.Env, req.Branch, nextSlot)

	if err := s.runSlotContainerWithRuntime(runtime, log, req.App, req.Env, req.Branch, nextSlot, snapshotName, servicePort, networkName, extraEnv); err != nil {
		return err
	}
	if err := s.waitForRuntimeContainerReady(runtime, log, candidateName, servicePort, rolloutReadyTimeout()); err != nil {
		runtime.Remove(candidateName)
		return err
	}

	recreateProxy := !runtime.IsRunning(appBaseContainerName(req.App, req.Env, req.Branch)) || activeSlot == ""
	if state != nil {
		prevMode := firstNonEmpty(strings.ToLower(strings.TrimSpace(state.Mode)), "port")
		prevTrafficMode := firstNonEmpty(normalizeTrafficMode(state.TrafficMode), "edge")
		prevHostPort := firstNonZero(state.HostPort, defaultHostPort(req.Env))
		if prevMode != mode || prevHostPort != hostPort || prevTrafficMode != trafficMode || strings.TrimSpace(state.PublicHost) != strings.TrimSpace(req.PublicHost) {
			recreateProxy = true
		}
	}
	if !recreateProxy && edgeProxyPublishedPortChanged(runtime, req.App, req.Env, req.Branch, hostPort, mode) {
		recreateProxy = true
	}
	if err := s.ensureVesselEdgeProxy(log, req.App, req.Env, req.Branch, nextSlot, activeSlot, servicePort, hostPort, mode, trafficMode, req.PublicHost, recreateProxy); err != nil {
		runtime.Remove(candidateName)
		return err
	}

	if activeSlot != "" && activeSlot != nextSlot {
		drainUntil := time.Now().Add(rolloutDrainDuration()).UnixMilli()
		if state != nil {
			state.ActiveSlot = nextSlot
			state.StandbySlot = activeSlot
			state.DrainUntil = drainUntil
			state.TrafficMode = trafficMode
			_ = s.saveAppState(state)
			s.broadcastSnapshot()
		}
		s.cleanupVesselStandbySlotAfter(req.App, req.Env, req.Branch, nextSlot, activeSlot, servicePort, hostPort, mode, trafficMode, req.PublicHost, rolloutDrainDuration())
	} else if state != nil {
		state.ActiveSlot = nextSlot
		state.StandbySlot = ""
		state.DrainUntil = 0
		state.TrafficMode = trafficMode
		_ = s.saveAppState(state)
		s.broadcastSnapshot()
	}
	return nil
}

func pruneStationSnapshots(app string, env DeployEnv, branch string, keep ...string) error {
	prefix := stationSnapshotPrefix(app, env, branch)
	// Prune WSL2-native snapshots via the daemon (primary store on Windows).
	if runtime.GOOS == "windows" {
		if agent, err := getStationAgent(); err == nil && agent != nil {
			agent.PruneSnapshots(prefix, keep)
		}
	}
	// Also prune Windows-side store (fallback when daemon was unavailable).
	return pruneLocalStationSnapshots(prefix, keep)
}

// pruneLocalStationSnapshots removes entries from the Windows-side snapshot
// store that match prefix and are not in the keep list.
func pruneLocalStationSnapshots(prefix string, keep []string) error {
	entries, err := os.ReadDir(stationSnapshotStoreDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	keepSet := make(map[string]struct{}, len(keep))
	for _, name := range keep {
		if name = strings.TrimSpace(name); name != "" {
			keepSet[name] = struct{}{}
		}
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !strings.HasPrefix(name, prefix) {
			continue
		}
		if _, ok := keepSet[name]; !ok {
			_ = os.RemoveAll(stationSnapshotDir(name))
		}
	}
	return nil
}

func streamFileTailSSE(w http.ResponseWriter, r *http.Request, path string, tail int, targetJSON string) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	var data []byte
	var err error
	for start := time.Now(); ; {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
		if time.Since(start) > 10*time.Second {
			return fmt.Errorf("log not found")
		}
		select {
		case <-r.Context().Done():
			return r.Context().Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	startIdx := 0
	if tail > 0 && len(lines) > tail {
		startIdx = len(lines) - tail
	}

	if targetJSON != "" {
		fmt.Fprintf(w, "event: runtime-target\ndata: %s\n\n", targetJSON)
	}
	fmt.Fprint(w, ": stream connected\n\n")
	flusher.Flush()
	for _, line := range lines[startIdx:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", line)
	}
	flusher.Flush()

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("log not found")
	}
	defer f.Close()
	if _, err := f.Seek(int64(len(data)), io.SeekStart); err != nil {
		return err
	}

	reader := bufio.NewReaderSize(f, 64*1024)
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		default:
		}

		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			if trimmed != "" {
				fmt.Fprintf(w, "data: %s\n\n", trimmed)
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (s *Server) stationRuntimeLogTargets(app string, env DeployEnv, branch string, st *AppState) ([]RuntimeLogTarget, RuntimeLogLaneState, error) {
	runtime := s.runtimeForEngine(EngineStation)
	activeSlot := ""
	standbySlot := ""
	if st != nil {
		activeSlot = normalizeActiveSlot(st.ActiveSlot)
		standbySlot = normalizeActiveSlot(st.StandbySlot)
	}
	if activeSlot == "" {
		activeSlot = s.currentActiveSlotWithRuntime(runtime, app, env, branch, st)
	}

	targets := make([]RuntimeLogTarget, 0, 6)
	lane := RuntimeLogLaneState{
		AppStopped:  st != nil && st.Stopped,
		ActiveSlot:  activeSlot,
		StandbySlot: standbySlot,
	}
	seen := map[string]struct{}{}
	add := func(target RuntimeLogTarget) {
		if target.ID == "" || target.Container == "" {
			return
		}
		if _, ok := seen[target.ID]; ok {
			return
		}
		target.Engine = EngineStation
		target.Running = runtime.IsRunning(target.Container)
		targets = append(targets, target)
		seen[target.ID] = struct{}{}
	}

	if activeSlot != "" {
		add(RuntimeLogTarget{
			ID:        "live",
			Label:     fmt.Sprintf("Live app (%s)", activeSlot),
			Kind:      "app",
			Container: appSlotContainerName(app, env, branch, activeSlot),
			Live:      true,
			Slot:      activeSlot,
			Image: func() string {
				if st != nil {
					return st.CurrentImage
				}
				return ""
			}(),
		})
	}
	if standbySlot != "" && standbySlot != activeSlot {
		add(RuntimeLogTarget{
			ID:        "standby",
			Label:     fmt.Sprintf("Standby app (%s)", standbySlot),
			Kind:      "app",
			Container: appSlotContainerName(app, env, branch, standbySlot),
			Slot:      standbySlot,
			Image: func() string {
				if st != nil {
					return st.PreviousImage
				}
				return ""
			}(),
		})
	}
	for _, slot := range []string{"blue", "green"} {
		if slot == activeSlot || slot == standbySlot {
			continue
		}
		name := appSlotContainerName(app, env, branch, slot)
		if runtime.IsRunning(name) {
			add(RuntimeLogTarget{
				ID:        "slot:" + slot,
				Label:     fmt.Sprintf("App slot (%s)", slot),
				Kind:      "app",
				Container: name,
				Slot:      slot,
				Image: func() string {
					if st != nil {
						return st.CurrentImage
					}
					return ""
				}(),
			})
		}
	}
	add(RuntimeLogTarget{
		ID:        "proxy",
		Label:     "Edge proxy",
		Kind:      "proxy",
		Container: appBaseContainerName(app, env, branch),
	})
	services, err := s.getProjectServices(app, string(env), branch)
	if err == nil {
		sort.Slice(services, func(i, j int) bool {
			return services[i].Name < services[j].Name
		})
		for _, svc := range services {
			add(RuntimeLogTarget{
				ID:        "service:" + svc.Name,
				Label:     fmt.Sprintf("Service: %s", svc.Name),
				Kind:      "service",
				Container: svc.Container,
				Service:   svc.Name,
				Image:     svc.Image,
			})
		}
	}
	for _, target := range targets {
		if !target.Running {
			continue
		}
		lane.HasRunningTarget = true
		if target.Kind == "app" {
			lane.AppRunning = true
		}
	}

	switch {
	case lane.AppStopped:
		lane.OfflineReason = "This app lane is currently off. Start or redeploy it to resume runtime logs."
	case !lane.AppRunning && activeSlot != "":
		lane.OfflineReason = fmt.Sprintf("Relay cannot find a running Vessel container for the live %s app slot.", activeSlot)
	case !lane.AppRunning:
		lane.OfflineReason = "Relay cannot find a running Vessel app container for this lane."
	}

	return targets, lane, nil
}

