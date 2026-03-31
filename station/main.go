// station — a slim container runtime experiment for relay.
//
// Cross-platform: Linux uses kernel namespaces + chroot for real isolation.
// Windows uses process groups + working-directory isolation.
// Goal: faster startup and lighter footprint than Docker as a future relay backend.
//
// Build:
//   Linux:   go build -o station .
//   Windows: go build -o station.exe .

package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ─── types ────────────────────────────────────────────────────────────────────

// ContainerRecord is persisted for each tracked process.
// Dir = rootfs (Linux chroot) or working directory (Windows).
type ContainerRecord struct {
	ID            string            `json:"id"`
	App           string            `json:"app"` // stable name, used for port reuse
	Dir           string            `json:"dir"`
	Command       []string          `json:"command"`
	PID           int               `json:"pid"`
	SupervisorPID int               `json:"supervisor_pid,omitempty"`
	Port          int               `json:"port"` // allocated host port (0 = none)
	Env           map[string]string `json:"env"`  // extra env injected into process
	UserEnv       map[string]string `json:"user_env,omitempty"`
	Started       time.Time         `json:"started"`
	RestartPolicy string            `json:"restart_policy,omitempty"` // "" | "always"
	Volumes       []string          `json:"volumes,omitempty"`        // ["host:container", ...]
	ExtraHosts    []string          `json:"extra_hosts,omitempty"`    // ["name:ip", ...]
	NetMode       string            `json:"net_mode,omitempty"`       // "" | "host" | "bridge"
	IP            string            `json:"ip,omitempty"`             // container IP (bridge mode)
	Image         string            `json:"image,omitempty"`          // snapshot name (used as base layer)
	WorkDir       string            `json:"work_dir,omitempty"`       // cleanup root for overlay or hardlink copy
	ContainerCwd  string            `json:"container_cwd,omitempty"`
	NetworkKey    string            `json:"network_key,omitempty"`
	OverlayActive bool              `json:"overlay_active,omitempty"` // WorkDir/merged is an overlay mount
}

// ─── state storage ────────────────────────────────────────────────────────────

func stateBaseDir() string {
	return filepath.Join(os.TempDir(), "relay-native")
}

func containerDir(id string) string {
	return filepath.Join(stateBaseDir(), "containers", id)
}

func configPath(id string) string {
	return filepath.Join(containerDir(id), "config.json")
}

func logPath(id string) string {
	return filepath.Join(containerDir(id), "output.log")
}

func saveRecord(rec *ContainerRecord) error {
	if err := os.MkdirAll(containerDir(rec.ID), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(rec.ID), data, 0644)
}

func loadRecord(id string) (*ContainerRecord, error) {
	data, err := os.ReadFile(configPath(id))
	if err != nil {
		return nil, fmt.Errorf("container %q not found", id)
	}
	var rec ContainerRecord
	return &rec, json.Unmarshal(data, &rec)
}

func allRecords() []*ContainerRecord {
	matches, _ := filepath.Glob(filepath.Join(stateBaseDir(), "containers", "*", "config.json"))
	var out []*ContainerRecord
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		var rec ContainerRecord
		if json.Unmarshal(data, &rec) == nil {
			out = append(out, &rec)
		}
	}
	return out
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func randID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		die("resolve path %q: %v", p, err)
	}
	return abs
}

// parseFlags pulls --app, --port, --restart, --volume / -v, --net, --env,
// --add-host, --workdir, and
// --image out of args, returning the values and the remaining positional args.
func parseFlags(args []string) (appName string, port int, restart string, volumes []string, netMode string, image string, extraEnv map[string]string, extraHosts []string, workdir string, rest []string) {
	extraEnv = map[string]string{}
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--app" && i+1 < len(args):
			appName = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--app="):
			appName = strings.TrimPrefix(args[i], "--app=")
		case args[i] == "--port" && i+1 < len(args):
			port, _ = strconv.Atoi(args[i+1])
			i++
		case strings.HasPrefix(args[i], "--port="):
			port, _ = strconv.Atoi(strings.TrimPrefix(args[i], "--port="))
		case args[i] == "--restart" && i+1 < len(args):
			restart = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--restart="):
			restart = strings.TrimPrefix(args[i], "--restart=")
		case (args[i] == "--volume" || args[i] == "-v") && i+1 < len(args):
			volumes = append(volumes, args[i+1])
			i++
		case strings.HasPrefix(args[i], "--volume="):
			volumes = append(volumes, strings.TrimPrefix(args[i], "--volume="))
		case args[i] == "--net" && i+1 < len(args):
			netMode = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--net="):
			netMode = strings.TrimPrefix(args[i], "--net=")
		case args[i] == "--image" && i+1 < len(args):
			image = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--image="):
			image = strings.TrimPrefix(args[i], "--image=")
		case (args[i] == "--env" || args[i] == "-e") && i+1 < len(args):
			if k, v, ok := splitEnvPair(args[i+1]); ok {
				extraEnv[k] = v
			}
			i++
		case strings.HasPrefix(args[i], "--env="):
			if k, v, ok := splitEnvPair(strings.TrimPrefix(args[i], "--env=")); ok {
				extraEnv[k] = v
			}
		case args[i] == "--add-host" && i+1 < len(args):
			extraHosts = append(extraHosts, args[i+1])
			i++
		case strings.HasPrefix(args[i], "--add-host="):
			extraHosts = append(extraHosts, strings.TrimPrefix(args[i], "--add-host="))
		case args[i] == "--workdir" && i+1 < len(args):
			workdir = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--workdir="):
			workdir = strings.TrimPrefix(args[i], "--workdir=")
		default:
			rest = append(rest, args[i])
		}
	}
	return
}

func splitEnvPair(value string) (string, string, bool) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), parts[1], true
}

func isProtectedContainerEnvKey(key string) bool {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "PORT", "APP_NAME", "CONTAINER_ID", "CONTAINER_VOLUMES", "CONTAINER_EXTRA_HOSTS", "CONTAINER_WORKDIR", "CONTAINER_FORWARD_ENV":
		return true
	default:
		return false
	}
}

func mergeContainerExtraEnv(env map[string]string, extraEnv map[string]string) []string {
	keys := make([]string, 0, len(extraEnv))
	for k, v := range extraEnv {
		key := strings.TrimSpace(k)
		if key == "" || isProtectedContainerEnvKey(key) {
			continue
		}
		env[key] = v
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ─── entry point ──────────────────────────────────────────────────────────────

func main() {
	if isContainerInit() {
		runContainerInit()
		os.Exit(1)
	}
	if isProxyRun() {
		runProxyDaemon()
		os.Exit(0)
	}
	if isBuildRunInit() {
		runBuildInit()
		os.Exit(0)
	}
	if maybeRunDesktopUI() {
		return
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		appName, port, restart, volumes, netMode, image, extraEnv, extraHosts, workdir, rest := parseFlags(os.Args[2:])
		if len(rest) < 2 && image == "" {
			die("usage: station run [--app <name>] [--port <n>] [--restart always] [--env K=V] [--add-host name:ip] [--workdir <dir>] [--volume h:c] [--net bridge] [--image <snapshot>] <dir> <cmd> [args...]")
		}
		if image != "" {
			// --image mode: rest is the command; the root filesystem comes from the snapshot
			cmdRun(appName, port, restart, volumes, netMode, image, extraEnv, extraHosts, workdir, "", rest, false)
		} else {
			cmdRun(appName, port, restart, volumes, netMode, "", extraEnv, extraHosts, workdir, rest[0], rest[1:], false)
		}

	case "run-fg":
		appName, port, restart, volumes, netMode, image, extraEnv, extraHosts, workdir, rest := parseFlags(os.Args[2:])
		if len(rest) < 2 && image == "" {
			die("usage: station run-fg ...")
		}
		if image != "" {
			cmdRun(appName, port, restart, volumes, netMode, image, extraEnv, extraHosts, workdir, "", rest, true)
		} else {
			cmdRun(appName, port, restart, volumes, netMode, "", extraEnv, extraHosts, workdir, rest[0], rest[1:], true)
		}

	case "build":
		if len(os.Args) < 4 {
			die("usage: station build [--app <name>] <dir> <cmd> [args...]")
		}
		appName, _, _, _, _, _, _, _, _, rest := parseFlags(os.Args[2:])
		if len(rest) < 2 {
			die("usage: station build [--app <name>] <dir> <cmd> [args...]")
		}
		cmdBuild(appName, rest[0], rest[1:])

	case "build-logs":
		if len(os.Args) < 3 {
			die("usage: station build-logs <build-id>")
		}
		cmdBuildLogs(os.Args[2])

	case "exec":
		if len(os.Args) < 3 {
			die("usage: station exec <id> [cmd [args...]]")
		}
		cmdExec(os.Args[2], os.Args[3:])

	case "snapshot":
		if len(os.Args) < 3 {
			die("usage: station snapshot <save|load|list|delete> [args...]")
		}
		switch os.Args[2] {
		case "save":
			if len(os.Args) < 5 {
				die("usage: station snapshot save <name> <dir>")
			}
			cmdSnapshotSave(os.Args[3], os.Args[4])
		case "load":
			if len(os.Args) < 5 {
				die("usage: station snapshot load <name> <dest-dir>")
			}
			cmdSnapshotLoad(os.Args[3], os.Args[4])
		case "list", "ls":
			cmdSnapshotList()
		case "delete", "rm":
			if len(os.Args) < 4 {
				die("usage: station snapshot delete <name>")
			}
			cmdSnapshotDelete(os.Args[3])
		default:
			die("unknown snapshot subcommand %q (save|load|list|delete)", os.Args[2])
		}

	case "proxy":
		if len(os.Args) < 3 {
			cmdProxyList()
			return
		}
		switch os.Args[2] {
		case "start":
			cmdProxyStart(parseProxyArgs(os.Args[3:]))
		case "swap":
			cfg := parseProxyArgs(os.Args[3:])
			if cfg.App == "" {
				die("usage: station proxy swap --app <app> [--active-upstream <host:port>]")
			}
			cmdProxySwap(cfg)
		case "stop":
			if len(os.Args) < 4 {
				die("usage: station proxy stop <app>")
			}
			cmdProxyStop(os.Args[3])
		case "list", "ls":
			cmdProxyList()
		default:
			die("unknown proxy subcommand %q (start|stop|swap|list)", os.Args[2])
		}

	case "list", "ps":
		cmdList()

	case "stop":
		if len(os.Args) < 3 {
			die("usage: station stop <id>")
		}
		cmdStop(os.Args[2])

	case "logs":
		if len(os.Args) < 3 {
			die("usage: station logs <id>")
		}
		cmdLogs(os.Args[2])

	case "status":
		if len(os.Args) < 3 {
			die("usage: station status <id>")
		}
		cmdStatus(os.Args[2])

	case "port":
		if len(os.Args) < 3 {
			cmdPortList()
			return
		}
		switch os.Args[2] {
		case "list", "ls":
			cmdPortList()
		case "free":
			if len(os.Args) < 4 {
				die("usage: station port free <app>")
			}
			cmdPortFree(os.Args[3])
		default:
			die("unknown port subcommand %q (list | free)", os.Args[2])
		}

	case "image":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: station image <pull|list|rm> [image]")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "pull":
			if len(os.Args) < 4 {
				die("usage: station image pull <image>")
			}
			cmdImagePull(os.Args[3])
		case "list", "ls":
			cmdImageList()
		case "rm", "remove", "delete":
			if len(os.Args) < 4 {
				die("usage: station image rm <image>")
			}
			cmdImageRemove(os.Args[3])
		default:
			die("unknown image subcommand: %s", os.Args[2])
		}

	case "build-dockerfile":
		// build-dockerfile <dockerfile> <contextDir> <outDir>
		// Builds a rootfs from a Dockerfile and writes station-manifest.json.
		if len(os.Args) < 5 {
			die("usage: station build-dockerfile <dockerfile> <context-dir> <out-dir>")
		}
		df, ctxDir, outDir := os.Args[2], os.Args[3], os.Args[4]
		logf := func(f string, a ...any) { fmt.Printf(f+"\n", a...) }
		m, err := BuildDockerfile(df, ctxDir, outDir, logf, os.Stdout)
		if err != nil {
			die("build-dockerfile: %v", err)
		}
		fmt.Printf("\nbuild complete → %s\n", outDir)
		if len(m.Cmd) > 0 {
			fmt.Printf("cmd: %v\n", m.Cmd)
		}
		if m.Port > 0 {
			fmt.Printf("port: %d\n", m.Port)
		}

	case "daemon":
		// Long-lived HTTP service inside WSL2.  relayd starts this once and
		// routes all build/snapshot calls over TCP instead of spawning wsl.exe
		// for each operation.
		// Usage: station daemon [--port-file=<path>]
		portFile := ""
		for _, arg := range os.Args[2:] {
			if strings.HasPrefix(arg, "--port-file=") {
				portFile = strings.TrimPrefix(arg, "--port-file=")
			}
		}
		cmdDaemon(portFile)

	case "wsl-ensure":
		// Ensure the Linux station binary is installed in the default WSL2 distro.
		// Called by relayd's station agent before starting the daemon so the
		// agent does not need to duplicate the wslEnsureStation logic itself.
		platformWslEnsure()

	case "wsl-warm":
		// Pre-warm the WSL2 VM and start the keepalive.  Add this to Windows
		// Task Scheduler → At log on → "station wsl-warm" to eliminate the
		// cold-start penalty on first use after each reboot.
		platformWslWarm()

	case "setup-rootfs":
		dest := "rootfs"
		if len(os.Args) >= 3 {
			dest = os.Args[2]
		}
		if err := platformSetupRootfs(dest); err != nil {
			die("%v", err)
		}

	case "_net-alloc":
		// Internal plumbing: allocate (or reuse) a bridge IP for a container ID
		// and print it to stdout. Called by the Windows host into WSL2 before
		// spawning a bridge container so both sides agree on the IP.
		if len(os.Args) < 3 {
			os.Exit(1)
		}
		netAllocCmd(os.Args[2])

	case "_supervise":
		// Internal: run the persistent restart-on-crash loop for a container.
		// Launched as a detached child by supervise() in supervisor.go so it
		// outlives the short-lived "station run" command.
		if len(os.Args) < 3 {
			die("usage: station _supervise <container-id>")
		}
		cmdSuperviseDaemon(os.Args[2])

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf(`station — slim container runtime for relay (%s)

Container commands:
  run [--app <n>] [--port <p>] <dir> <cmd> [args...]  Run detached
  run-fg [--app <n>] [--port <p>] <dir> <cmd> [args...] Run foreground
  list                                                   List containers
  stop <id>                                              Stop container
  logs <id>                                              Container output
  status <id>                                            Container details

Build commands:
  build [--app <n>] <dir> <cmd> [args...]            Run a build step (npm ci, go build…)
  build-logs <build-id>                              Show output of a past build
  build-dockerfile <dockerfile> <ctx> <out-dir>      Build rootfs from Dockerfile (no Docker)

Image commands:
  image pull   <image>   Pull an OCI image from Docker Hub
  image list             List cached images
  image rm     <image>   Remove a cached image

Port commands:
  port              List app → port allocations
  port free <app>   Release a port allocation

Other:
  setup-rootfs [dest]  %s
  wsl-warm             Pre-warm WSL2 VM + start keepalive (add to Windows startup)
  Windows desktop      Double-click vessel.exe in Explorer to open Station Desktop

Flags (run / run-fg):
  --app <name>        Stable app name — port is reused across redeploys
  --port <n>          Pin to a specific port instead of auto-allocating
  --restart always    Restart on crash (process supervisor)
  --volume h:c        Bind-mount a host path or named managed volume at c
  -v h:c              Alias for --volume
  --add-host n:ip     Add a static hostname inside the container (/etc/hosts)
  --net bridge        Attach to station0 bridge (10.88.0.0/16), requires root
  --net host          Share host network (default)
  --image <name>      Run from a snapshot (overlay on Linux, hardlink fallback)

Proxy / blue-green commands:
  proxy start --app <n> --port <p> --upstream <host:port>  Start reverse proxy
  proxy swap  --app <n> --upstream <host:port>             Hot-swap upstream
  proxy stop  <app>                                        Stop proxy
  proxy list                                               List proxies

Snapshot commands:
  snapshot save   <name> <dir>   Commit rootfs as a reusable image
  snapshot load   <name> <dest>  Materialise a working copy from snapshot
  snapshot list                  List saved images
  snapshot delete <name>         Remove image

`, platformName(), platformSetupRootfsHint())
}

// ─── run command ──────────────────────────────────────────────────────────────

func cmdRun(appName string, pinPort int, restart string, volumes []string, netMode string, image string, extraEnv map[string]string, extraHosts []string, containerWorkDir string, dir string, cmdArgs []string, foreground bool) {
	id := randID()

	if extraEnv == nil {
		extraEnv = map[string]string{}
	}
	if image != "" {
		manifest, err := loadImageManifest(image)
		if err != nil {
			die("image manifest: %v", err)
		}
		if containerWorkDir == "" && strings.TrimSpace(manifest.WorkDir) != "" {
			containerWorkDir = strings.TrimSpace(manifest.WorkDir)
		}
		for key, value := range manifest.Env {
			if _, exists := extraEnv[key]; exists {
				continue
			}
			extraEnv[key] = value
		}
		if len(cmdArgs) == 0 {
			cmdArgs = append(cmdArgs, manifest.Entrypoint...)
			cmdArgs = append(cmdArgs, manifest.Cmd...)
		}
		if len(cmdArgs) == 0 {
			die("image %q has no default command", image)
		}
	}

	// If --image is set, prepare an overlay mount (Linux) or hardlink copy (other).
	var imageRootfsDir string
	var imageIsOverlay bool
	if image != "" && !platformSkipImagePrep(image) {
		var err error
		imageRootfsDir, imageIsOverlay, err = prepareImageRootfs(image, id)
		if err != nil {
			die("image: %v", err)
		}
		if dir == "" {
			dir = imageRootfsDir
		}
	}

	// Platform hook: on Windows, if dir is actually an exe file, auto-correct.
	dir, cmdArgs = resolveDir(dir, cmdArgs)
	absDir := mustAbs(dir)
	if _, err := os.Stat(absDir); err != nil {
		die("dir %q: %v", absDir, err)
	}

	// Port allocation: pin > named app (stable) > ephemeral
	port := pinPort
	if port == 0 && appName != "" {
		var err error
		port, err = allocPort(appName)
		if err != nil {
			die("port alloc: %v", err)
		}
	} else if port == 0 {
		var err error
		port, err = findFreePort()
		if err != nil {
			die("find free port: %v", err)
		}
	}

	// Inject PORT into the process environment so the app can bind to it.
	resolvedVolumes, err := resolveVolumeSpecs(volumes)
	if err != nil {
		die("volumes: %v", err)
	}
	env := map[string]string{
		"PORT":         strconv.Itoa(port),
		"CONTAINER_ID": id,
	}
	if appName != "" {
		env["APP_NAME"] = appName
	}
	// Volume paths for containerInit.
	if len(resolvedVolumes) > 0 {
		env["CONTAINER_VOLUMES"] = strings.Join(resolvedVolumes, ",")
	}
	if len(extraHosts) > 0 {
		env["CONTAINER_EXTRA_HOSTS"] = strings.Join(extraHosts, ",")
	}
	if containerWorkDir != "" {
		env["CONTAINER_WORKDIR"] = containerWorkDir
	}
	if len(extraEnv) > 0 {
		keys := mergeContainerExtraEnv(env, extraEnv)
		env["CONTAINER_FORWARD_ENV"] = strings.Join(keys, ",")
	}

	rec := &ContainerRecord{
		ID: id, App: appName, Dir: absDir,
		Command: cmdArgs, Port: port, Env: env,
		UserEnv:       extraEnv,
		Started:       time.Now(),
		RestartPolicy: restart,
		Volumes:       volumes,
		ExtraHosts:    extraHosts,
		NetMode:       netMode,
		Image:         image,
		ContainerCwd:  containerWorkDir,
		NetworkKey:    containerNetworkKey(appName, id),
		OverlayActive: imageIsOverlay,
	}

	// If --image was used, record the cleanup root for releaseImageRootfs.
	if image != "" {
		rec.WorkDir = imageWorkPath(id, imageIsOverlay)
	}

	// Allocate a container IP and inject CONTAINER_NET_* env vars (Linux bridge only).
	if err := prepareNetworkEnv(rec); err != nil {
		die("network: %v", err)
	}

	var logWriter io.Writer = os.Stdout
	var logFile *os.File

	if !foreground {
		if err := os.MkdirAll(containerDir(id), 0755); err != nil {
			die("create state dir: %v", err)
		}
		lf, err := os.Create(logPath(id))
		if err != nil {
			die("create log: %v", err)
		}
		logFile = lf
		logWriter = lf
	}

	pid, err := doSpawn(rec, foreground, logWriter)
	if logFile != nil {
		_ = logFile.Close()
	}
	if err != nil {
		if !foreground {
			_ = os.RemoveAll(containerDir(id))
		}
		die("spawn: %v", err)
	}

	if foreground {
		return
	}

	rec.PID = pid
	if err := saveRecord(rec); err != nil {
		die("save state: %v", err)
	}

	// Start process supervisor if restart policy is "always".
	if restart == "always" {
		rec.SupervisorPID = supervise(rec)
		if rec.SupervisorPID > 0 {
			_ = saveRecord(rec)
		}
	}

	app := appName
	if app == "" {
		app = id
	}
	fmt.Printf("container %s started  app=%s  pid=%d  port=%d\n", id, app, pid, port)
	if rec.IP != "" {
		fmt.Printf("  ip:     %s\n", rec.IP)
	}
	if restart == "always" {
		fmt.Printf("  restart: always (supervisor running)\n")
	}
	fmt.Printf("  logs:   station logs %s\n", id)
	fmt.Printf("  stop:   station stop %s\n", id)
}

// ─── other commands ───────────────────────────────────────────────────────────

func cmdList() {
	recs := allRecords()
	if len(recs) == 0 {
		fmt.Println("no containers")
		return
	}
	fmt.Printf("%-10s  %-12s  %-7s  %-6s  %-7s  %-20s  %s\n",
		"ID", "APP", "STATUS", "PORT", "PID", "STARTED", "COMMAND")
	for _, rec := range recs {
		status := "stopped"
		if pidAlive(rec.PID) {
			status = "running"
		}
		cmd := strings.Join(rec.Command, " ")
		if len(cmd) > 35 {
			cmd = cmd[:32] + "..."
		}
		app := rec.App
		if app == "" {
			app = "-"
		}
		portStr := "-"
		if rec.Port > 0 {
			portStr = strconv.Itoa(rec.Port)
		}
		fmt.Printf("%-10s  %-12s  %-7s  %-6s  %-7s  %-20s  %s\n",
			rec.ID, app, status, portStr, strconv.Itoa(rec.PID),
			rec.Started.Format("01-02 15:04:05"), cmd)
	}
}

func cmdStop(id string) {
	wasRunning, err := stopContainer(id)
	if err != nil {
		die("%v", err)
	}
	if wasRunning {
		fmt.Printf("container %s stopped\n", id)
		return
	}
	fmt.Printf("container %s is not running\n", id)
}

func stopContainer(id string) (bool, error) {
	rec, err := loadRecord(id)
	if err != nil {
		return false, err
	}
	wasRunning := pidAlive(rec.PID)
	if !pidAlive(rec.PID) {
	} else {
		if err := killProcess(rec.PID); err != nil {
			return false, fmt.Errorf("kill %d: %v", rec.PID, err)
		}
	}
	if rec.SupervisorPID > 0 && pidAlive(rec.SupervisorPID) {
		_ = killProcess(rec.SupervisorPID)
	}
	// Release stable port allocation so the next deploy can reuse the same port.
	if rec.App != "" {
		releasePort(rec.App)
	}
	// Remove the record first so the supervisor goroutine (if any) sees it
	// gone and stops attempting restarts.
	_ = os.RemoveAll(containerDir(id))
	// Clean up bridge networking (veth pair). No-op on non-Linux or host-mode.
	teardownContainerNetwork(rec)
	// Release overlay mount or hardlink working copy created by --image.
	releaseImageRootfs(rec)
	return wasRunning, nil
}

func cmdLogs(id string) {
	data, err := os.ReadFile(logPath(id))
	if err != nil {
		die("no log for %s: %v", id, err)
	}
	_, _ = os.Stdout.Write(data)
}

// buildEnv merges the host environment with the per-container overrides so
// every key in extra overwrites the host value. Used by both spawn files.
func buildEnv(extra map[string]string) []string {
	env := os.Environ()
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

func cmdStatus(id string) {
	rec, err := loadRecord(id)
	if err != nil {
		die("%v", err)
	}
	status := "stopped"
	if pidAlive(rec.PID) {
		status = "running"
	}
	fmt.Printf("ID:       %s\n", rec.ID)
	fmt.Printf("App:      %s\n", rec.App)
	fmt.Printf("Status:   %s\n", status)
	fmt.Printf("PID:      %d\n", rec.PID)
	fmt.Printf("Port:     %d\n", rec.Port)
	if rec.IP != "" {
		fmt.Printf("IP:       %s\n", rec.IP)
	}
	if rec.NetMode != "" {
		fmt.Printf("Net:      %s\n", rec.NetMode)
	}
	if rec.RestartPolicy != "" {
		fmt.Printf("Restart:  %s\n", rec.RestartPolicy)
	}
	if len(rec.Volumes) > 0 {
		fmt.Printf("Volumes:  %s\n", strings.Join(rec.Volumes, ", "))
	}
	if len(rec.ExtraHosts) > 0 {
		fmt.Printf("Hosts:    %s\n", strings.Join(rec.ExtraHosts, ", "))
	}
	if rec.Image != "" {
		fmt.Printf("Image:    %s\n", rec.Image)
	}
	fmt.Printf("Dir:      %s\n", rec.Dir)
	fmt.Printf("Command:  %s\n", strings.Join(rec.Command, " "))
	fmt.Printf("Started:  %s\n", rec.Started.Format(time.RFC3339))
	fmt.Printf("Platform: %s\n", platformName())
}

// cmdExec enters an existing running container's namespaces.
func cmdExec(target string, shellArgs []string) {
	rec, err := resolveExecTarget(target)
	if err != nil {
		die("%v", err)
	}
	if err := platformExec(rec, shellArgs); err != nil {
		die("%v", err)
	}
}

func resolveExecTarget(target string) (*ContainerRecord, error) {
	if rec, err := loadRecord(target); err == nil {
		if !pidAlive(rec.PID) {
			return nil, fmt.Errorf("container %s is not running (pid %d)", rec.ID, rec.PID)
		}
		return rec, nil
	}
	records := allRecords()
	sort.Slice(records, func(i, j int) bool {
		return records[i].Started.After(records[j].Started)
	})
	for _, rec := range records {
		if rec.App != target {
			continue
		}
		if !pidAlive(rec.PID) {
			continue
		}
		return rec, nil
	}
	return nil, fmt.Errorf("container %q not found", target)
}

func containerNetworkKey(appName, id string) string {
	if strings.TrimSpace(appName) != "" {
		return strings.TrimSpace(appName)
	}
	return strings.TrimSpace(id)
}


