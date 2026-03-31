//go:build linux

package main

// Station daemon — persistent HTTP service running inside WSL2.
//
// relayd on Windows starts this once via a detached wsl.exe call and then
// communicates over 127.0.0.1:<port> (WSL2 auto-forwards TCP to Windows) for
// all build and snapshot operations.  This eliminates the per-operation
// wsl.exe spawn cost and the WSL VM cold-boot lag.
//
// Protocol for POST /build-dockerfile (streaming):
//   L:<text>   — a log line from the build (written to the caller's logw)
//   M:<json>   — final manifest on success (JSON of BuildManifest)
//   E:<msg>    — fatal build error
//
// All other endpoints use plain JSON request/response bodies.

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	hdrContentType = "Content-Type"
	mimeJSON       = "application/json"
	msgNotAllowed  = "method not allowed"
	msgBadRequest  = "bad request: "
	msgAppRequired = "app is required"
)

// ─── request / response types ─────────────────────────────────────────────────

type daemonBuildReq struct {
	Dockerfile   string `json:"dockerfile"`
	ContextDir   string `json:"context_dir"`
	SnapshotName string `json:"snapshot_name"`
}

type daemonSnapshotSaveReq struct {
	Name   string `json:"name"`
	SrcDir string `json:"src_dir"`
}

type daemonPruneReq struct {
	Prefix string   `json:"prefix"`
	Keep   []string `json:"keep"`
}

// ─── container run / stop ─────────────────────────────────────────────────────

type daemonRunContainerReq struct {
	App        string   `json:"app"`
	Image      string   `json:"image"`
	Command    []string `json:"command,omitempty"`
	Env        []string `json:"env,omitempty"` // KEY=VALUE pairs
	Volumes    []string `json:"volumes,omitempty"`
	ExtraHosts []string `json:"extra_hosts,omitempty"`
	NetMode    string   `json:"net_mode,omitempty"`
	Restart    string   `json:"restart,omitempty"`
	Port       int      `json:"port,omitempty"`
}

type daemonRunContainerResp struct {
	ID   string `json:"id"`
	PID  int    `json:"pid"`
	Port int    `json:"port"`
	IP   string `json:"ip,omitempty"`
}

type daemonStopAppReq struct {
	App string `json:"app"`
}

// ─── proxy ────────────────────────────────────────────────────────────────────

type daemonProxyReq struct {
	App             string `json:"app"`
	Port            int    `json:"port,omitempty"`
	ActiveUpstream  string `json:"active_upstream"`
	StandbyUpstream string `json:"standby_upstream,omitempty"`
	ActiveSlot      string `json:"active_slot,omitempty"`
	StandbySlot     string `json:"standby_slot,omitempty"`
	TrafficMode     string `json:"traffic_mode,omitempty"`
	CookieName      string `json:"cookie_name,omitempty"`
	PublicHost      string `json:"public_host,omitempty"`
	ClearStandby    bool   `json:"clear_standby,omitempty"`
	ClearPublicHost bool   `json:"clear_public_host,omitempty"`
}

type daemonProxyStartResp struct {
	Port int `json:"port"`
	PID  int `json:"pid"`
}

// ─── entry point ──────────────────────────────────────────────────────────────

// cmdDaemon starts the station HTTP daemon.  portFile receives the chosen port
// number so the Windows side can read it via a single wslRun call and dial
// directly over TCP.
func cmdDaemon(portFile string) {
	if portFile == "" {
		portFile = "/tmp/relay-station-agent.port"
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "[station-daemon] listen:", err)
		os.Exit(1)
	}
	port := l.Addr().(*net.TCPAddr).Port

	_ = os.MkdirAll(filepath.Dir(portFile), 0755)
	if writeErr := os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0644); writeErr != nil {
		fmt.Fprintf(os.Stderr, "[station-daemon] write port file: %v\n", writeErr)
	}
	fmt.Printf("[station-daemon] pid=%d port=%d\n", os.Getpid(), port)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", daemonHealth)
	mux.HandleFunc("/build-dockerfile", daemonBuild)
	mux.HandleFunc("/snapshot/save", daemonSnapshotSave)
	mux.HandleFunc("/snapshot/", daemonSnapshotByName) // manifest + delete + exists
	mux.HandleFunc("/snapshots", daemonListSnapshots)
	mux.HandleFunc("/snapshots/prune", daemonPruneSnapshots)
	mux.HandleFunc("/container/run", daemonContainerRun)
	mux.HandleFunc("/container/stop", daemonContainerStop)
	mux.HandleFunc("/proxy/start", daemonProxyStart)
	mux.HandleFunc("/proxy/swap", daemonProxySwap)
	mux.HandleFunc("/proxy/stop", daemonProxyStop)

	if err := http.Serve(l, mux); err != nil {
		fmt.Fprintln(os.Stderr, "[station-daemon] serve:", err)
	}
}

// ─── handlers ─────────────────────────────────────────────────────────────────

func daemonHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(hdrContentType, mimeJSON)
	_, _ = io.WriteString(w, `{"ok":true}`)
}

// daemonBuild handles POST /build-dockerfile.
// Builds the Dockerfile inside WSL2 native storage, saves the snapshot, and
// streams progress back to the caller using the L:/M:/E: line protocol.
func daemonBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonBuildReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgBadRequest+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Dockerfile == "" || req.ContextDir == "" || req.SnapshotName == "" {
		http.Error(w, "dockerfile, context_dir and snapshot_name required", http.StatusBadRequest)
		return
	}

	w.Header().Set(hdrContentType, "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	daemonFlush(w)

	// Build to a temp dir inside the WSL2 filesystem so all I/O stays on ext4.
	outDir := filepath.Join(stateBaseDir(), "daemon-builds", randID())
	_ = os.RemoveAll(outDir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		daemonErr(w, "create build dir: "+err.Error())
		return
	}
	defer os.RemoveAll(outDir)

	logw := &daemonLineWriter{w: w, prefix: "L:"}
	logf := func(format string, args ...any) {
		_, _ = fmt.Fprintf(logw, format+"\n", args...)
	}

	// Pre-sync the build context from the slow 9P /mnt/ bridge to WSL2-native
	// ext4 storage so that Dockerfile COPY instructions read at full speed.
	// rsync makes this incremental on repeated builds; cp -a is the fallback.
	contextDir := req.ContextDir
	if strings.HasPrefix(req.ContextDir, "/mnt/") {
		if synced, syncErr := syncContextToNative(req.ContextDir, logw); syncErr == nil {
			contextDir = synced
		} else {
			fmt.Fprintf(logw, "[context-sync] warn: %v — using original path\n", syncErr)
		}
	}

	manifest, err := BuildDockerfile(req.Dockerfile, contextDir, outDir, logf, logw)
	if err != nil {
		daemonErr(w, "build: "+err.Error())
		return
	}

	// Atomically replace any prior snapshot with the new rootfs.
	_ = os.RemoveAll(snapshotPath(req.SnapshotName))
	if err := os.MkdirAll(snapshotStore(), 0755); err != nil {
		daemonErr(w, "snapshot store: "+err.Error())
		return
	}
	if err := hardlinkCopy(outDir, snapshotPath(req.SnapshotName)); err != nil {
		daemonErr(w, "snapshot save: "+err.Error())
		return
	}

	data, _ := json.Marshal(manifest)
	_, _ = fmt.Fprintf(w, "M:%s\n", data)
	daemonFlush(w)
}

func daemonSnapshotSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonSnapshotSaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_ = os.RemoveAll(snapshotPath(req.Name))
	if err := os.MkdirAll(snapshotStore(), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := hardlinkCopy(req.SrcDir, snapshotPath(req.Name)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// daemonSnapshotByName handles:
//   GET    /snapshot/<name>            — 200 if exists, 404 if not
//   GET    /snapshot/<name>/manifest   — return manifest JSON
//   DELETE /snapshot/<name>            — remove snapshot
func daemonSnapshotByName(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/snapshot/")

	if name, ok := strings.CutSuffix(tail, "/manifest"); ok {
		data, err := os.ReadFile(filepath.Join(snapshotPath(name), "station-manifest.json"))
		if err != nil {
			http.Error(w, "manifest not found", http.StatusNotFound)
			return
		}
		w.Header().Set(hdrContentType, mimeJSON)
		_, _ = w.Write(data)
		return
	}

	name := tail
	if name == "" {
		http.Error(w, "snapshot name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		_ = os.RemoveAll(snapshotPath(name))
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		if _, err := os.Stat(snapshotPath(name)); err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
	}
}

func daemonListSnapshots(w http.ResponseWriter, _ *http.Request) {
	entries, _ := os.ReadDir(snapshotStore())
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	w.Header().Set(hdrContentType, mimeJSON)
	_ = json.NewEncoder(w).Encode(map[string][]string{"names": names})
}

func daemonPruneSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonPruneReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	keepSet := make(map[string]struct{}, len(req.Keep))
	for _, k := range req.Keep {
		if k = strings.TrimSpace(k); k != "" {
			keepSet[k] = struct{}{}
		}
	}
	entries, _ := os.ReadDir(snapshotStore())
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, req.Prefix) {
			continue
		}
		if _, ok := keepSet[name]; ok {
			continue
		}
		_ = os.RemoveAll(snapshotPath(name))
	}
	w.WriteHeader(http.StatusOK)
}

// ─── streaming helpers ────────────────────────────────────────────────────────

// daemonLineWriter prefixes every written line with prefix and flushes after
// each write so the Windows client sees live build output.
type daemonLineWriter struct {
	w      http.ResponseWriter
	prefix string
}

func (d *daemonLineWriter) Write(p []byte) (int, error) {
	lines := strings.SplitAfter(string(p), "\n")
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			continue
		}
		if _, err := fmt.Fprintf(d.w, "%s%s\n", d.prefix, trimmed); err != nil {
			return 0, err
		}
	}
	daemonFlush(d.w)
	return len(p), nil
}

func daemonFlush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func daemonErr(w http.ResponseWriter, msg string) {
	_, _ = fmt.Fprintf(w, "E:%s\n", strings.ReplaceAll(msg, "\n", " "))
	daemonFlush(w)
}

// ─── context pre-sync ────────────────────────────────────────────────────────

// syncContextToNative copies src (a /mnt/ 9P path) to WSL2-native storage so
// Dockerfile COPY instructions read from ext4 instead of the slow 9P bridge.
// Uses rsync for incremental copies when available, cp -a as fallback.
func syncContextToNative(src string, logw io.Writer) (string, error) {
	key := strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(strings.TrimPrefix(src, "/"))
	dst := filepath.Join(stateBaseDir(), "build-contexts", key)
	fmt.Fprintf(logw, "[context-sync] syncing %s → %s\n", src, dst)

	if _, err := exec.LookPath("rsync"); err == nil {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return "", fmt.Errorf("mkdir build-context dst: %w", err)
		}
		cmd := exec.Command("rsync", "-a", "--delete", src+"/", dst+"/")
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("rsync: %v (%s)", err, strings.TrimSpace(string(out)))
		}
		return dst, nil
	}

	// Fallback: full copy each time (cp -a available everywhere on Linux).
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "", fmt.Errorf("mkdir parent: %w", err)
	}
	_ = os.RemoveAll(dst)
	cmd := exec.Command("cp", "-a", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("cp: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return dst, nil
}

// ─── container run / stop ────────────────────────────────────────────────────

func daemonContainerRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonRunContainerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgBadRequest+err.Error(), http.StatusBadRequest)
		return
	}
	if req.App == "" {
		http.Error(w, msgAppRequired, http.StatusBadRequest)
		return
	}
	if req.Image == "" {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}
	resp, err := runContainerForDaemon(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(hdrContentType, mimeJSON)
	_ = json.NewEncoder(w).Encode(resp)
}

// runContainerForDaemon is the error-returning equivalent of cmdRun, used by
// the daemon HTTP handler so errors become HTTP responses rather than os.Exit.
func runContainerForDaemon(req daemonRunContainerReq) (*daemonRunContainerResp, error) {
	id := randID()
	rec, err := buildContainerRecord(id, req)
	if err != nil {
		return nil, err
	}

	if err := prepareNetworkEnv(rec); err != nil {
		releaseImageRootfs(rec)
		return nil, fmt.Errorf("network: %w", err)
	}
	if err := os.MkdirAll(containerDir(id), 0755); err != nil {
		releaseImageRootfs(rec)
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	lf, createErr := os.Create(logPath(id))
	if createErr != nil {
		releaseImageRootfs(rec)
		return nil, fmt.Errorf("create log: %w", createErr)
	}

	pid, spawnErr := doSpawn(rec, false, lf)
	_ = lf.Close()
	if spawnErr != nil {
		_ = os.RemoveAll(containerDir(id))
		releaseImageRootfs(rec)
		return nil, fmt.Errorf("spawn: %w", spawnErr)
	}

	rec.PID = pid
	if err := saveRecord(rec); err != nil {
		return nil, fmt.Errorf("save state: %w", err)
	}
	if req.Restart == "always" {
		rec.SupervisorPID = supervise(rec)
		if rec.SupervisorPID > 0 {
			_ = saveRecord(rec)
		}
	}
	return &daemonRunContainerResp{ID: id, PID: pid, Port: rec.Port, IP: rec.IP}, nil
}

// buildContainerRecord prepares a ContainerRecord from a run request without
// spawning the process — allocates port, loads manifest, builds env map.
func buildContainerRecord(id string, req daemonRunContainerReq) (*ContainerRecord, error) {
	extraEnv, cmdArgs, workDir, err := resolveImageMeta(req)
	if err != nil {
		return nil, err
	}

	dir, isOverlay, err := prepareImageRootfs(req.Image, id)
	if err != nil {
		return nil, fmt.Errorf("image rootfs: %w", err)
	}

	cleanupRootfs := func() { releaseImageRootfs(&ContainerRecord{WorkDir: imageWorkPath(id, isOverlay), OverlayActive: isOverlay}) }

	port := req.Port
	if port == 0 {
		if req.App != "" {
			port, err = allocPort(req.App)
		} else {
			port, err = findFreePort()
		}
		if err != nil {
			cleanupRootfs()
			return nil, fmt.Errorf("port alloc: %w", err)
		}
	}

	resolvedVolumes, err := resolveVolumeSpecs(req.Volumes)
	if err != nil {
		cleanupRootfs()
		return nil, fmt.Errorf("volumes: %w", err)
	}

	env := buildContainerEnv(id, req.App, port, workDir, resolvedVolumes, req.ExtraHosts, extraEnv)
	return &ContainerRecord{
		ID:            id,
		App:           req.App,
		Dir:           dir,
		Command:       cmdArgs,
		Port:          port,
		Env:           env,
		UserEnv:       extraEnv,
		Started:       time.Now(),
		RestartPolicy: req.Restart,
		Volumes:       req.Volumes,
		ExtraHosts:    req.ExtraHosts,
		NetMode:       req.NetMode,
		Image:         req.Image,
		ContainerCwd:  workDir,
		NetworkKey:    containerNetworkKey(req.App, id),
		OverlayActive: isOverlay,
		WorkDir:       imageWorkPath(id, isOverlay),
	}, nil
}

// resolveImageMeta loads the image manifest and merges req.Env on top.
func resolveImageMeta(req daemonRunContainerReq) (extraEnv map[string]string, cmdArgs []string, workDir string, err error) {
	manifest, loadErr := loadImageManifest(req.Image)
	if loadErr != nil {
		return nil, nil, "", fmt.Errorf("image manifest: %w", loadErr)
	}
	extraEnv = make(map[string]string, len(manifest.Env))
	for k, v := range manifest.Env {
		extraEnv[k] = v
	}
	workDir = strings.TrimSpace(manifest.WorkDir)
	cmdArgs = append([]string(nil), req.Command...)
	if len(cmdArgs) == 0 {
		cmdArgs = append(manifest.Entrypoint, manifest.Cmd...)
	}
	if len(cmdArgs) == 0 {
		return nil, nil, "", fmt.Errorf("image %q has no default command", req.Image)
	}
	// Per-request env overrides take precedence over manifest env.
	for _, pair := range req.Env {
		if k, v, ok := splitEnvPair(pair); ok {
			extraEnv[k] = v
		}
	}
	return extraEnv, cmdArgs, workDir, nil
}

// buildContainerEnv constructs the full env map injected into the container.
func buildContainerEnv(id, app string, port int, workDir string, volumes, extraHosts []string, extra map[string]string) map[string]string {
	env := map[string]string{"PORT": strconv.Itoa(port), "CONTAINER_ID": id}
	if app != "" {
		env["APP_NAME"] = app
	}
	if len(volumes) > 0 {
		env["CONTAINER_VOLUMES"] = strings.Join(volumes, ",")
	}
	if len(extraHosts) > 0 {
		env["CONTAINER_EXTRA_HOSTS"] = strings.Join(extraHosts, ",")
	}
	if workDir != "" {
		env["CONTAINER_WORKDIR"] = workDir
	}
	if len(extra) > 0 {
		keys := mergeContainerExtraEnv(env, extra)
		env["CONTAINER_FORWARD_ENV"] = strings.Join(keys, ",")
	}
	return env
}

func daemonContainerStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonStopAppReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgBadRequest+err.Error(), http.StatusBadRequest)
		return
	}
	if req.App == "" {
		http.Error(w, msgAppRequired, http.StatusBadRequest)
		return
	}

	// Stop the edge proxy for this app if running.
	if rec := loadSlotRecord(req.App); rec != nil {
		if rec.PID > 0 && pidAlive(rec.PID) {
			_ = killProcess(rec.PID)
		}
		_ = os.RemoveAll(proxyDir(req.App))
	}

	// Stop all containers belonging to this app.
	for _, rec := range allRecords() {
		if rec.App != req.App {
			continue
		}
		if rec.SupervisorPID > 0 && pidAlive(rec.SupervisorPID) {
			_ = killProcess(rec.SupervisorPID)
		}
		if rec.PID > 0 && pidAlive(rec.PID) {
			_ = killProcess(rec.PID)
		}
		if rec.App != "" {
			releasePort(rec.App)
		}
		_ = os.RemoveAll(containerDir(rec.ID))
		teardownContainerNetwork(rec)
		releaseImageRootfs(rec)
	}
	w.WriteHeader(http.StatusOK)
}

// ─── proxy start / swap / stop ───────────────────────────────────────────────

func daemonProxyStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonProxyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgBadRequest+err.Error(), http.StatusBadRequest)
		return
	}
	if req.App == "" {
		http.Error(w, msgAppRequired, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.ActiveUpstream) == "" {
		http.Error(w, "active_upstream is required", http.StatusBadRequest)
		return
	}

	port := req.Port
	if port == 0 {
		var err error
		port, err = findFreePort()
		if err != nil {
			http.Error(w, "find free port: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	rec := normalizeSlotRecord(&SlotRecord{
		App:             req.App,
		ProxyPort:       port,
		ActiveUpstream:  strings.TrimSpace(req.ActiveUpstream),
		StandbyUpstream: strings.TrimSpace(req.StandbyUpstream),
		ActiveSlot:      firstProxyValue(normalizeProxySlot(req.ActiveSlot), "blue"),
		StandbySlot:     normalizeProxySlot(req.StandbySlot),
		TrafficMode:     firstProxyValue(normalizeProxyTrafficMode(req.TrafficMode), "edge"),
		CookieName:      firstProxyValue(strings.TrimSpace(req.CookieName), "station_slot"),
		PublicHost:      strings.TrimSpace(req.PublicHost),
	})
	if err := saveSlotRecord(rec); err != nil {
		http.Error(w, "save proxy config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	self, err := os.Executable()
	if err != nil {
		http.Error(w, "resolve executable: "+err.Error(), http.StatusInternalServerError)
		return
	}
	pid, err := startDetachedProcess(self, []string{proxyMagic, req.App})
	if err != nil {
		http.Error(w, "start proxy daemon: "+err.Error(), http.StatusInternalServerError)
		return
	}
	rec.PID = pid
	_ = saveSlotRecord(rec)

	w.Header().Set(hdrContentType, mimeJSON)
	_ = json.NewEncoder(w).Encode(daemonProxyStartResp{Port: port, PID: pid})
}

func daemonProxySwap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonProxyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgBadRequest+err.Error(), http.StatusBadRequest)
		return
	}
	if req.App == "" {
		http.Error(w, msgAppRequired, http.StatusBadRequest)
		return
	}
	rec := loadSlotRecord(req.App)
	if rec == nil {
		http.Error(w, "no proxy for app "+req.App, http.StatusNotFound)
		return
	}
	applyProxySwap(rec, req)
	if err := saveSlotRecord(rec); err != nil {
		http.Error(w, "save proxy config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// applyProxySwap mutates rec in-place according to the swap request fields.
func applyProxySwap(rec *SlotRecord, req daemonProxyReq) {
	if s := strings.TrimSpace(req.ActiveUpstream); s != "" {
		rec.ActiveUpstream = s
	}
	applyProxyStandby(rec, req)
	if slot := normalizeProxySlot(req.ActiveSlot); slot != "" {
		rec.ActiveSlot = slot
	}
	if slot := normalizeProxySlot(req.StandbySlot); slot != "" && !req.ClearStandby {
		rec.StandbySlot = slot
	}
	if mode := normalizeProxyTrafficMode(req.TrafficMode); mode != "" {
		rec.TrafficMode = mode
	}
	if cookie := strings.TrimSpace(req.CookieName); cookie != "" {
		rec.CookieName = cookie
	}
	applyProxyPublicHost(rec, req)
}

func applyProxyStandby(rec *SlotRecord, req daemonProxyReq) {
	if req.ClearStandby {
		rec.StandbyUpstream = ""
		rec.StandbySlot = ""
		return
	}
	if s := strings.TrimSpace(req.StandbyUpstream); s != "" {
		rec.StandbyUpstream = s
		rec.StandbySlot = firstProxyValue(normalizeProxySlot(req.StandbySlot), rec.StandbySlot, nextProxySlot(rec.ActiveSlot))
	}
}

func applyProxyPublicHost(rec *SlotRecord, req daemonProxyReq) {
	if req.ClearPublicHost {
		rec.PublicHost = ""
	} else if s := strings.TrimSpace(req.PublicHost); s != "" {
		rec.PublicHost = s
	}
}

func daemonProxyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req daemonStopAppReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgBadRequest+err.Error(), http.StatusBadRequest)
		return
	}
	if req.App == "" {
		http.Error(w, msgAppRequired, http.StatusBadRequest)
		return
	}

	rec := loadSlotRecord(req.App)
	if rec != nil && rec.PID > 0 && pidAlive(rec.PID) {
		_ = killProcess(rec.PID)
	}
	_ = os.RemoveAll(proxyDir(req.App))
	w.WriteHeader(http.StatusOK)
}


