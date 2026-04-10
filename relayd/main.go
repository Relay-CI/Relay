// main.go
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

const relaydServiceUsage = `relayd service — manage systemd unit (Linux)

Usage:
	relayd service unit [--name relayd] [--user root] [--group root] [--data-dir /var/lib/relayd] [--addr :8080] [--bin /usr/local/bin/relayd]
	relayd service install [--name relayd] [--user root] [--group root] [--data-dir /var/lib/relayd] [--addr :8080] [--bin /usr/local/bin/relayd]

Examples:
	relayd service unit
	sudo relayd service install --user relay --group relay --data-dir /var/lib/relayd
`

const relaydRunUsage = `relayd — Relay agent

Usage:
	relayd [--addr :8080] [--port 8080] [--data-dir ./data] [--socket /path/to/relay.sock] [--no-socket]
	relayd version
	relayd service <...>

Examples:
	relayd
	relayd --port 9090
	relayd --addr 0.0.0.0:9090
	relayd --data-dir /var/lib/relayd --socket /var/lib/relayd/relay.sock
`

type relaydRunConfig struct {
	Addr          string
	DataDir       string
	SocketPath    string
	DisableSocket bool
	ShowVersion   bool
}

func parseRelaydRunArgs(args []string) (relaydRunConfig, error) {
	var cfg relaydRunConfig
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		switch {
		case a == "version" || a == "--version" || a == "-version":
			cfg.ShowVersion = true
		case a == "--help" || a == "-h" || a == "help":
			return cfg, fmt.Errorf("%s", relaydRunUsage)
		case a == "--no-socket":
			cfg.DisableSocket = true
		case a == "--addr":
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("--addr requires a value")
			}
			cfg.Addr = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--addr="):
			cfg.Addr = strings.TrimSpace(strings.TrimPrefix(a, "--addr="))
		case a == "--port":
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("--port requires a value")
			}
			port := strings.TrimSpace(args[i])
			if _, err := strconv.Atoi(port); err != nil {
				return cfg, fmt.Errorf("invalid --port value %q", port)
			}
			cfg.Addr = ":" + port
		case strings.HasPrefix(a, "--port="):
			port := strings.TrimSpace(strings.TrimPrefix(a, "--port="))
			if _, err := strconv.Atoi(port); err != nil {
				return cfg, fmt.Errorf("invalid --port value %q", port)
			}
			cfg.Addr = ":" + port
		case a == "--data-dir":
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("--data-dir requires a value")
			}
			cfg.DataDir = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--data-dir="):
			cfg.DataDir = strings.TrimSpace(strings.TrimPrefix(a, "--data-dir="))
		case a == "--socket":
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("--socket requires a value")
			}
			cfg.SocketPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--socket="):
			cfg.SocketPath = strings.TrimSpace(strings.TrimPrefix(a, "--socket="))
		default:
			return cfg, fmt.Errorf("unknown argument %q", a)
		}
	}
	return cfg, nil
}

//go:embed ui/*
var uiFS embed.FS

const dashboardSessionCookie = "relay_session"

var (
	relaydVersion   = "dev"
	relaydCommit    = "unknown"
	relaydBuildDate = "unknown"
)

func relaydVersionLine() string {
	return fmt.Sprintf("relayd %s (commit=%s built=%s os=%s arch=%s)", relaydVersion, relaydCommit, relaydBuildDate, runtime.GOOS, runtime.GOARCH)
}

func stationBinaryVersion() (version string, binaryPath string, err error) {
	bin, err := ensureVesselBinary()
	if err != nil {
		return "", "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--version")
	setCmdHideWindow(cmd)
	out, runErr := runCommandCaptured(cmd)
	if runErr != nil {
		return "", bin, runErr
	}
	return strings.TrimSpace(string(out)), bin, nil
}

// ctxKey is used to attach per-connection metadata injected by ConnContext.
type ctxKey int

const (
	// ctxKeySocket marks requests that arrived over the Unix domain socket.
	// The socket is protected by filesystem permissions (0600), so these
	// connections are considered pre-authenticated and do not need a token.
	ctxKeySocket ctxKey = iota
)

type DeployEnv string

const (
	EnvPreview DeployEnv = "preview"
	EnvProd    DeployEnv = "prod"
	EnvStaging DeployEnv = "staging"
	EnvDev     DeployEnv = "dev"
)

type DeployStatus string

const (
	StatusQueued  DeployStatus = "queued"
	StatusRunning DeployStatus = "running"
	StatusFailed  DeployStatus = "failed"
	StatusSuccess DeployStatus = "success"
)

func isActiveDeployStatus(status DeployStatus) bool {
	switch string(status) {
	case "queued", "running", "building":
		return true
	default:
		return false
	}
}

type DeployRequest struct {
	App       string    `json:"app"`
	RepoURL   string    `json:"repo_url"`
	Branch    string    `json:"branch"`
	CommitSHA string    `json:"commit_sha"`
	Env       DeployEnv `json:"env"`

	// Optional overrides; otherwise resolved from detected plan / relay.config.json
	InstallCmd string `json:"install_cmd"`
	BuildCmd   string `json:"build_cmd"`
	StartCmd   string `json:"start_cmd"`

	ServicePort int `json:"service_port"`
	HostPort    int `json:"host_port"`
	// Internal marker so persisted explicit preview ports are not silently re-assigned.
	HostPortExplicit bool   `json:"-"`
	PublicHost       string `json:"public_host"`

	Mode        string `json:"mode"`         // "traefik" or "port"
	TrafficMode string `json:"traffic_mode"` // "edge" or "session"
	Source      string `json:"source"`       // "git" or "sync"
	Engine      string `json:"engine"`       // "docker" or "station"/"vessel"; overrides stored app state when set

	CommitMessage string `json:"commit_message,omitempty"`
	DeployedBy    string `json:"-"` // username of the user who triggered the deploy
}

type Deploy struct {
	ID            string       `json:"id"`
	App           string       `json:"app"`
	RepoURL       string       `json:"repo_url"`
	Branch        string       `json:"branch"`
	CommitSHA     string       `json:"commit_sha"`
	Env           DeployEnv    `json:"env"`
	Status        DeployStatus `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
	StartedAt     *time.Time   `json:"started_at,omitempty"`
	EndedAt       *time.Time   `json:"ended_at,omitempty"`
	Error         string       `json:"error,omitempty"`
	LogPath       string       `json:"log_path"`
	ImageTag      string       `json:"image_tag,omitempty"`
	PrevImage     string       `json:"previous_image_tag,omitempty"`
	PreviewURL    string       `json:"preview_url,omitempty"`
	Source        string       `json:"source,omitempty"`
	BuildNumber   int          `json:"build_number,omitempty"`
	DeployedBy    string       `json:"deployed_by,omitempty"`
	CommitMessage string       `json:"commit_message,omitempty"`
}

type DeployJob struct {
	ID            string
	Req           DeployRequest
	Rollback      bool
	RollbackImage string
	PromoteImage  string
}

type AppState struct {
	App              string    `json:"app"`
	Env              DeployEnv `json:"env"`
	Branch           string    `json:"branch"`
	RepoURL          string    `json:"repo_url"`
	Engine           string    `json:"engine,omitempty"`
	CurrentImage     string    `json:"current_image,omitempty"`
	PreviousImage    string    `json:"previous_image,omitempty"`
	Mode             string    `json:"mode"`
	HostPort         int       `json:"host_port"`
	HostPortExplicit bool      `json:"host_port_explicit,omitempty"`
	ServicePort      int       `json:"service_port"`
	PublicHost       string    `json:"public_host"`
	ActiveSlot       string    `json:"active_slot,omitempty"`
	StandbySlot      string    `json:"standby_slot,omitempty"`
	DrainUntil       int64     `json:"drain_until,omitempty"`
	TrafficMode      string    `json:"traffic_mode"`
	AccessPolicy     string    `json:"access_policy,omitempty"`
	IPAllowlist      string    `json:"ip_allowlist,omitempty"`
	ExpiresAt        int64     `json:"expires_at,omitempty"`
	RepoHash         string    `json:"repo_hash,omitempty"`
	WebhookSecret    string    `json:"webhook_secret,omitempty"`
	Stopped          bool      `json:"stopped,omitempty"`
}

// ---------------------- Multi-service / Project config ----------------------

// ProjectConfig is read from relay.json at the root of the repo.
// If present, Relay starts companion services (databases, caches) before
// launching the app container, wiring everything on a shared Docker network.
type ProjectConfig struct {
	Project  string          `json:"project"`
	Services []ServiceConfig `json:"services"`
}

// ServiceConfig describes a single service inside a project.
// Type is one of: "app", "postgres", "mysql", "redis", "mongo".
type ServiceConfig struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Version  string            `json:"version"`        // e.g. "16" for postgres:16
	Port     int               `json:"port,omitempty"` // override default container port
	HostPort int               `json:"host_port,omitempty"`
	Image    string            `json:"image,omitempty"`
	Command  string            `json:"command,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Volumes  []string          `json:"volumes,omitempty"`
	Health   *ServiceHealth    `json:"health,omitempty"`
	Stopped  bool              `json:"stopped,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
}

// ProjectService tracks a running companion (database) service in SQLite.
type ProjectService struct {
	Project   string `json:"project"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Branch    string `json:"branch"`
	Env       string `json:"env"`
	Container string `json:"container"`
	Network   string `json:"network"`
	Volume    string `json:"volume"`
	EnvKey    string `json:"env_key"`
	EnvVal    string `json:"env_val"`
	Image     string `json:"image,omitempty"`
	Port      int    `json:"port,omitempty"`
	HostPort  int    `json:"host_port,omitempty"`
	SpecHash  string `json:"spec_hash,omitempty"`
	Running   bool   `json:"running"`
}

type ServiceHealth struct {
	Test               string `json:"test,omitempty"`
	IntervalSeconds    int    `json:"interval_seconds,omitempty"`
	TimeoutSeconds     int    `json:"timeout_seconds,omitempty"`
	Retries            int    `json:"retries,omitempty"`
	StartPeriodSeconds int    `json:"start_period_seconds,omitempty"`
}

type ServiceSpecRecord struct {
	Project   string
	Env       string
	Branch    string
	Name      string
	Config    ServiceConfig
	UpdatedAt int64
}

type AppSecret struct {
	App    string `json:"app"`
	Env    string `json:"env"`
	Branch string `json:"branch"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

type SyncStartRequest struct {
	App         string    `json:"app"`
	Branch      string    `json:"branch"`
	Env         DeployEnv `json:"env"`
	BaseVersion string    `json:"base_version,omitempty"`
}

type SyncStartResponse struct {
	SessionID        string `json:"session_id"`
	WorkspaceVersion string `json:"workspace_version,omitempty"`
}

type ManifestFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Mtime    int64  `json:"mtime"`
	Sha256   string `json:"sha256,omitempty"`
	Hash     string `json:"hash,omitempty"`
	HashAlgo string `json:"hash_algo,omitempty"`
}

type SyncPlanRequest struct {
	Files []ManifestFile `json:"files"`
}

type SyncPlanResponse struct {
	Need   []string `json:"need"`
	Delete []string `json:"delete"`
}

type SyncDeleteRequest struct {
	Paths []string `json:"paths"`
}

type SyncSession struct {
	ID            string
	App           string
	Branch        string
	Env           DeployEnv
	RepoDir       string
	StagingDir    string
	CreatedAt     time.Time
	DeleteList    []string
	UploadedBytes int64
	MaxBytes      int64
	uploadMu      sync.Mutex
}

// ─── User auth ───────────────────────────────────────────────────────────────

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Role         string // "owner", "deployer", "viewer"
}

type UserSession struct {
	Token    string
	UserID   string
	Username string
	Role     string
}

type AuthCode struct {
	Code     string
	UserID   string
	Username string
	Role     string
}

// ─── Container runtime abstraction ───────────────────────────────────────────

// ContainerRuntime abstracts Docker-backed container operations used by relayd.
// Per-lane engine selection can dispatch around this for experimental backends.
type ContainerRuntime interface {
	// RunDetached starts a container in the background.
	RunDetached(spec ContainerSpec) error
	// Remove stops and deletes a container (ignores not-found errors).
	Remove(name string)
	// IsRunning reports whether the named container is currently running.
	IsRunning(name string) bool
	// ContainerIP returns the first IP address found in the container's networks.
	ContainerIP(name string) string
	// PublishedPort returns the host-side port mapped to the given container port.
	PublishedPort(name string, containerPort int) int
	// Exec runs a command inside a running container and returns combined output.
	Exec(container string, cmd []string) ([]byte, error)
	// NetworkConnect attaches a running container to an additional network.
	NetworkConnect(container, network string) error
	// EnsureNetwork creates a bridge network if it does not already exist.
	EnsureNetwork(name string) error
	// RemoveNetwork deletes a network (ignores errors).
	RemoveNetwork(name string)
	// RemoveVolume deletes a named volume (ignores errors).
	RemoveVolume(name string)
	// Pull pulls an image from a registry (ignores errors).
	Pull(image string) error
	// Build builds an image from contextDir tagged as tag.
	// dockerfilePath may be "" to use the default Dockerfile in contextDir.
	Build(ctx context.Context, tag, contextDir, dockerfilePath string, logw io.Writer) error
	// RemoveImage removes an image by reference (ignores errors).
	RemoveImage(ref string)
	// ListImages returns all image refs matching the given repository name.
	ListImages(repo string) ([]string, error)
	// LogStream opens a streaming log reader for the named container.
	LogStream(ctx context.Context, name string, tail int, since string) (io.ReadCloser, error)
}

// ContainerSpec describes a container to launch via ContainerRuntime.RunDetached.
type ContainerSpec struct {
	Name          string
	Image         string
	Network       string
	RestartPolicy string   // "always", "unless-stopped", or "" (no --restart flag)
	Env           []string // "KEY=VALUE" pairs
	Volumes       []string // "source:target[:options]" bindings
	ExtraHosts    []string // "name:ip" aliases to inject via /etc/hosts
	PortBindings  []string // "hostSpec:containerPort" e.g. "127.0.0.1::3000"
	HealthArgs    []string // pre-computed --health-* flag pairs (from healthArgs())
	Command       []string // optional command override
}

// DockerRuntime implements ContainerRuntime by calling the local Docker CLI.
type DockerRuntime struct{}

func (r *DockerRuntime) RunDetached(spec ContainerSpec) error {
	args := []string{"run", "-d", "--name", spec.Name}
	if spec.RestartPolicy != "" {
		args = append(args, "--restart="+spec.RestartPolicy)
	}
	if spec.Network != "" {
		args = append(args, "--network", spec.Network)
	}
	for _, e := range spec.Env {
		args = append(args, "-e", e)
	}
	for _, v := range spec.Volumes {
		args = append(args, "-v", v)
	}
	for _, host := range spec.ExtraHosts {
		args = append(args, "--add-host", host)
	}
	for _, p := range spec.PortBindings {
		args = append(args, "-p", p)
	}
	args = append(args, spec.HealthArgs...)
	args = append(args, spec.Image)
	args = append(args, spec.Command...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run %s: %v — %s", spec.Name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *DockerRuntime) Remove(name string) {
	_ = exec.Command("docker", "rm", "-f", name).Run()
}

func (r *DockerRuntime) IsRunning(name string) bool {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", name).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func (r *DockerRuntime) ContainerIP(name string) string {
	out, err := exec.Command("docker", "inspect", "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", name).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r *DockerRuntime) PublishedPort(name string, containerPort int) int {
	format := fmt.Sprintf("{{with index .NetworkSettings.Ports %q}}{{(index . 0).HostPort}}{{end}}", fmt.Sprintf("%d/tcp", firstNonZero(containerPort, 3000)))
	out, err := exec.Command("docker", "inspect", "--format", format, name).CombinedOutput()
	if err != nil {
		return 0
	}
	port, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return port
}

func (r *DockerRuntime) Exec(container string, cmd []string) ([]byte, error) {
	args := append([]string{"exec", container}, cmd...)
	return exec.Command("docker", args...).CombinedOutput()
}

func (r *DockerRuntime) NetworkConnect(container, network string) error {
	return exec.Command("docker", "network", "connect", network, container).Run()
}

func (r *DockerRuntime) EnsureNetwork(name string) error {
	out, _ := exec.Command("docker", "network", "inspect", name).CombinedOutput()
	if strings.Contains(string(out), `"Name"`) {
		return nil
	}
	out, err := exec.Command("docker", "network", "create", "--driver", "bridge", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker network create: %v — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *DockerRuntime) RemoveNetwork(name string) {
	_ = exec.Command("docker", "network", "rm", name).Run()
}

func (r *DockerRuntime) RemoveVolume(name string) {
	if strings.TrimSpace(name) == "" {
		return
	}
	_ = exec.Command("docker", "volume", "rm", "-f", name).Run()
}

func (r *DockerRuntime) Pull(image string) error {
	if strings.TrimSpace(image) == "" {
		return nil
	}
	return exec.Command("docker", "pull", image).Run()
}

func (r *DockerRuntime) Build(ctx context.Context, tag, contextDir, dockerfilePath string, logw io.Writer) error {
	args := []string{"build", "-t", tag}
	if dockerfilePath != "" {
		args = append(args, "-f", dockerfilePath)
	}
	args = append(args, ".")
	return runCmdLoggedEnvCtx(ctx, contextDir, logw, []string{"DOCKER_BUILDKIT=1"}, "docker", args...)
}

func (r *DockerRuntime) RemoveImage(ref string) {
	if strings.TrimSpace(ref) == "" {
		return
	}
	_ = exec.Command("docker", "image", "rm", "-f", ref).Run()
}

func (r *DockerRuntime) ListImages(repo string) ([]string, error) {
	out, err := exec.Command("docker", "images", repo, "--format", "{{.Repository}}:{{.Tag}}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list images %s: %v (%s)", repo, err, strings.TrimSpace(string(out)))
	}
	var result []string
	for _, line := range strings.Split(string(out), "\n") {
		img := strings.TrimSpace(line)
		if img != "" && img != "<none>:<none>" {
			result = append(result, img)
		}
	}
	return result, nil
}

func (r *DockerRuntime) LogStream(ctx context.Context, name string, tail int, since string) (io.ReadCloser, error) {
	args := []string{"logs", "--timestamps", "--tail", strconv.Itoa(tail), "-f"}
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, name)
	cmd := exec.CommandContext(ctx, "docker", args...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return nil, err
	}
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
	}()
	return pr, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type Server struct {
	mu      sync.RWMutex
	deploys map[string]*Deploy

	queue chan DeployJob

	syncMu       sync.Mutex
	syncSessions map[string]*SyncSession

	dataDir               string
	workspacesDir         string
	logsDir               string
	pluginsDir            string
	acmeWebroot           string
	caddyLogsDir          string
	httpAddr              string
	corsOrigins           map[string]struct{}
	allowAllCORS          bool
	enablePluginMutations bool

	db *sql.DB

	apiToken  string
	secretKey []byte // 32-byte AES-256 key for encrypting secrets at rest; nil = no encryption

	buildpacks []Buildpack

	buildLock sync.Mutex
	building  map[string]bool // app__env__branch

	buildCancelsMu sync.Mutex
	buildCancels   map[string]context.CancelFunc // deployID → active cancel func

	webhookRateMu sync.Mutex
	webhookHits   map[string][]time.Time // repoURL → recent trigger timestamps

	eventsMu    sync.RWMutex
	eventsChans map[chan []byte]struct{}

	runtime        ContainerRuntime
	stationRuntime ContainerRuntime
}

// ---------------------- Config ----------------------

type RelayConfig struct {
	Kind        string `json:"kind"`         // optional hint; else auto-detect
	BuildImage  string `json:"build_image"`  // docker image to run install/build
	RunImage    string `json:"run_image"`    // runtime image; if empty, defaults per pack
	ServicePort int    `json:"service_port"` // container port
	InstallCmd  string `json:"install_cmd"`
	BuildCmd    string `json:"build_cmd"`
	StartCmd    string `json:"start_cmd"`
	Dockerfile  string `json:"dockerfile"` // if set, agent uses this dockerfile as-is
}

func readRelayConfig(repoDir string) (*RelayConfig, error) {
	p := filepath.Join(repoDir, "relay.config.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c RelayConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// ---------------------- Buildpacks ----------------------

type BuildPlan struct {
	Kind        string
	ServicePort int

	BuildImage string
	RunImage   string

	InstallCmd string
	BuildCmd   string
	StartCmd   string

	// Create / overwrite Dockerfile unless Config.Dockerfile is used
	WriteDockerfile func(repoDir string) error

	Verify  func(repoDir string) error
	Cleanup func(repoDir string) error
}

// Buildpack interface for automatic detection and plan generation
type Buildpack interface {
	Name() string
	Detect(repoDir string, cfg *RelayConfig) bool
	Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error)
}

type PluginDetectRules struct {
	Kind              string   `json:"kind,omitempty"`
	Kinds             []string `json:"kinds,omitempty"`
	FilesAny          []string `json:"files_any,omitempty"`
	FilesAll          []string `json:"files_all,omitempty"`
	DirsAny           []string `json:"dirs_any,omitempty"`
	DirsAll           []string `json:"dirs_all,omitempty"`
	PackageDepsAny    []string `json:"package_deps_any,omitempty"`
	PackageDepsAll    []string `json:"package_deps_all,omitempty"`
	FileExtensionsAny []string `json:"file_extensions_any,omitempty"`
	FileExtensionsAll []string `json:"file_extensions_all,omitempty"`
}

type PluginPlanSpec struct {
	Kind               string   `json:"kind"`
	ServicePort        int      `json:"service_port,omitempty"`
	BuildImage         string   `json:"build_image,omitempty"`
	RunImage           string   `json:"run_image,omitempty"`
	InstallCmd         string   `json:"install_cmd,omitempty"`
	BuildCmd           string   `json:"build_cmd,omitempty"`
	StartCmd           string   `json:"start_cmd,omitempty"`
	DockerfileTemplate string   `json:"dockerfile_template"`
	WriteDefaultConf   bool     `json:"write_default_conf,omitempty"`
	WasmMime           bool     `json:"wasm_mime,omitempty"`
	CleanupPaths       []string `json:"cleanup_paths,omitempty"`
}

type BuildpackPlugin struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	DetectRules PluginDetectRules `json:"detect"`
	PlanSpec    PluginPlanSpec    `json:"plan"`
}

type PluginBuildpack struct {
	plugin *BuildpackPlugin
}

// defaultBuildpacks returns the ordered list of buildpacks we support.
func defaultBuildpacks() []Buildpack {
	return []Buildpack{
		&NodeNextStandaloneBuildpack{},
		&NodeNextBuildpack{},
		&NodeViteBuildpack{},
		&ExpoWebBuildpack{},
		&SprintUIBuildpack{},
		&NodeGenericBuildpack{},
		&GoBuildpack{},
		&DotnetBuildpack{},
		&PythonBuildpack{},
		&JavaBuildpack{},
		&RustBuildpack{},
		&CCppBuildpack{},
		&WasmStaticBuildpack{},
		&StaticBuildpack{},
	}
}

func (b *PluginBuildpack) Name() string { return b.plugin.Name }

func (b *PluginBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	rules := b.plugin.DetectRules
	if cfg != nil && cfg.Kind != "" {
		want := strings.ToLower(strings.TrimSpace(cfg.Kind))
		if want != "" {
			if strings.EqualFold(rules.Kind, want) {
				return true
			}
			for _, k := range rules.Kinds {
				if strings.EqualFold(k, want) {
					return true
				}
			}
		}
	}
	if !pluginDetectRuleMatch(repoDir, rules) {
		return false
	}
	return true
}

func (b *PluginBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	spec := b.plugin.PlanSpec
	if strings.TrimSpace(spec.DockerfileTemplate) == "" {
		return BuildPlan{}, fmt.Errorf("plugin %s has empty dockerfile_template", b.plugin.Name)
	}
	servicePort := spec.ServicePort
	if servicePort == 0 {
		servicePort = 3000
	}
	if req.ServicePort != 0 {
		servicePort = req.ServicePort
	} else if cfg != nil && cfg.ServicePort != 0 {
		servicePort = cfg.ServicePort
	}
	plan := BuildPlan{
		Kind:        firstNonEmpty(spec.Kind, b.plugin.Name),
		ServicePort: servicePort,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), spec.BuildImage),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), spec.RunImage),
		InstallCmd:  firstNonEmpty(req.InstallCmd, spec.InstallCmd),
		BuildCmd:    firstNonEmpty(req.BuildCmd, spec.BuildCmd),
		StartCmd:    firstNonEmpty(req.StartCmd, spec.StartCmd),
	}
	plan.WriteDockerfile = func(repoDir string) error {
		rendered, err := renderPluginDockerfile(b.plugin, plan)
		if err != nil {
			return err
		}
		if spec.WriteDefaultConf {
			return writeStaticDockerArtifacts(repoDir, rendered, spec.WasmMime)
		}
		return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(rendered), 0644)
	}
	plan.Cleanup = func(repoDir string) error {
		for _, rel := range spec.CleanupPaths {
			rel = filepath.Clean(rel)
			if rel == "." || strings.HasPrefix(rel, "..") {
				continue
			}
			_ = os.RemoveAll(filepath.Join(repoDir, rel))
		}
		_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
		if spec.WriteDefaultConf {
			marker := filepath.Join(repoDir, ".relay_default_conf_created")
			if fileExists(marker) {
				_ = os.Remove(filepath.Join(repoDir, "default.conf"))
				_ = os.Remove(marker)
			}
		}
		return nil
	}
	return plan, nil
}

func pluginDetectRuleMatch(repoDir string, rules PluginDetectRules) bool {
	if len(rules.FilesAny) == 0 && len(rules.FilesAll) == 0 &&
		len(rules.DirsAny) == 0 && len(rules.DirsAll) == 0 &&
		len(rules.PackageDepsAny) == 0 && len(rules.PackageDepsAll) == 0 &&
		len(rules.FileExtensionsAny) == 0 && len(rules.FileExtensionsAll) == 0 {
		return false
	}
	if len(rules.FilesAny) > 0 && !anyPathExists(repoDir, false, rules.FilesAny) {
		return false
	}
	if len(rules.FilesAll) > 0 && !allPathExists(repoDir, false, rules.FilesAll) {
		return false
	}
	if len(rules.DirsAny) > 0 && !anyPathExists(repoDir, true, rules.DirsAny) {
		return false
	}
	if len(rules.DirsAll) > 0 && !allPathExists(repoDir, true, rules.DirsAll) {
		return false
	}
	if len(rules.PackageDepsAny) > 0 && !anyPackageDep(repoDir, rules.PackageDepsAny) {
		return false
	}
	if len(rules.PackageDepsAll) > 0 && !allPackageDeps(repoDir, rules.PackageDepsAll) {
		return false
	}
	if len(rules.FileExtensionsAny) > 0 && !anyFileExt(repoDir, rules.FileExtensionsAny) {
		return false
	}
	if len(rules.FileExtensionsAll) > 0 && !allFileExt(repoDir, rules.FileExtensionsAll) {
		return false
	}
	return true
}

func renderPluginDockerfile(plugin *BuildpackPlugin, plan BuildPlan) (string, error) {
	tpl, err := template.New(plugin.Name).Funcs(template.FuncMap{
		"shellJSON":  shellJSON,
		"shellForm":  shellForm,
		"quoteForSh": quoteForSh,
		"shQuote":    shQuote,
	}).Parse(plugin.PlanSpec.DockerfileTemplate)
	if err != nil {
		return "", fmt.Errorf("plugin %s template parse error: %w", plugin.Name, err)
	}
	data := map[string]any{
		"Name":        plugin.Name,
		"Description": plugin.Description,
		"Kind":        plan.Kind,
		"ServicePort": plan.ServicePort,
		"BuildImage":  plan.BuildImage,
		"RunImage":    plan.RunImage,
		"InstallCmd":  plan.InstallCmd,
		"BuildCmd":    plan.BuildCmd,
		"StartCmd":    plan.StartCmd,
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("plugin %s template exec error: %w", plugin.Name, err)
	}
	return buf.String(), nil
}

type NodeNextBuildpack struct{}

type NodeNextStandaloneBuildpack struct{}

func (b *NodeNextStandaloneBuildpack) Name() string { return "next-standalone" }
func (b *NodeNextStandaloneBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "next-standalone") {
		return true
	}
	return isNextStandaloneEnabled(repoDir)
}
func (b *NodeNextStandaloneBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	forced := &RelayConfig{Kind: "next-standalone"}
	if cfg != nil {
		copyCfg := *cfg
		copyCfg.Kind = "next-standalone"
		forced = &copyCfg
	}
	return (&NodeNextBuildpack{}).Plan(req, repoDir, forced)
}

func (b *NodeNextBuildpack) Name() string { return "node-next" }
func (b *NodeNextBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "next-standalone") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "next.config.js")) ||
		fileExists(filepath.Join(repoDir, "next.config.mjs")) ||
		fileExists(filepath.Join(repoDir, "next.config.cjs")) ||
		fileExists(filepath.Join(repoDir, "next.config.ts")) ||
		fileExists(filepath.Join(repoDir, "next.config.mts"))
}
func (b *NodeNextBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim")

	standalone := isNextStandaloneEnabled(repoDir)
	if cfg != nil && strings.EqualFold(cfg.Kind, "next-standalone") {
		standalone = true
	}

	port := 3000
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, nodeDefaultBuildCmd(repoDir))

	if standalone {
		return BuildPlan{
			Kind:        "next-standalone",
			ServicePort: port,
			BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
			RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
			InstallCmd:  install,
			BuildCmd:    build,
			StartCmd:    `node server.js`,
			// Verification happens inside the built image; host checks are unreliable.
			Verify: nil,
			WriteDockerfile: func(repoDir string) error {
				// Multi-stage: install with a cached package store, build with a persistent .next cache,
				// then copy the standalone runtime output into the final image.
				df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
RUN mkdir -p /app/public/uploads && chown -R node:node /app || true
USER node
EXPOSE 3000
CMD ["node","server.js"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, install), nodeRunStepWithCaches(repoDir, build, "/app/.next/cache"), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg))

				return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
			},
			Cleanup: func(repoDir string) error {
				_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
				return nil
			},
		}, nil
	}

	// Classic Next.js (no standalone)
	start := firstNonEmpty(req.StartCmd, nextClassicStartCmd(repoDir))
	return BuildPlan{
		Kind:        "next-classic",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			// Keep dev deps in the build stage, derive production deps from that cached install,
			// and reuse a persistent .next cache between builds.
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s
RUN rm -rf node_modules

%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
EXPOSE 3000
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, install), nodeRunStepWithCaches(repoDir, build, "/app/.next/cache"), nodeProdDepsStage(repoDir), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), shellJSON(start))

			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type NodeViteBuildpack struct{}

func (b *NodeViteBuildpack) Name() string { return "node-vite" }
func (b *NodeViteBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "vite-static") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "vite.config.ts")) ||
		fileExists(filepath.Join(repoDir, "vite.config.js")) ||
		fileExists(filepath.Join(repoDir, "vite.config.mjs"))
}
func (b *NodeViteBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, nodeDefaultBuildCmd(repoDir))

	return BuildPlan{
		Kind:        "vite-static",
		ServicePort: 80,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    "",
		// Build runs inside Docker; host filesystem checks are unreliable.
		Verify: nil,
		WriteDockerfile: func(repoDir string) error {
			// Multi-stage: build with node using a cached package store, then serve with nginx.
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s

FROM %s
COPY --from=builder /app/dist /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, install), nodeRunStepWithCaches(repoDir, build), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg))

			if err := os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644); err != nil {
				return err
			}
			// Only write default.conf if the repo doesn't provide one already
			defPath := filepath.Join(repoDir, "default.conf")
			marker := filepath.Join(repoDir, ".relay_default_conf_created")
			if !fileExists(defPath) {
				defaultConf := `server {
			listen 80;
			server_name _;
			root /usr/share/nginx/html;
			index index.html;
			location / {
				try_files $uri $uri/ /index.html;
			}
			}
`
				if err := os.WriteFile(defPath, []byte(defaultConf), 0644); err != nil {
					return err
				}
				_ = os.WriteFile(marker, []byte("1"), 0644)
				return nil
			}
			return nil
		},
		Cleanup: func(repoDir string) error {
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.RemoveAll(filepath.Join(repoDir, "dist"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			marker := filepath.Join(repoDir, ".relay_default_conf_created")
			if fileExists(marker) {
				_ = os.Remove(filepath.Join(repoDir, "default.conf"))
				_ = os.Remove(marker)
			}
			return nil
		},
	}, nil
}

type ExpoWebBuildpack struct{}

func (b *ExpoWebBuildpack) Name() string { return "expo-web" }
func (b *ExpoWebBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "expo-web") {
		return true
	}
	if !fileExists(filepath.Join(repoDir, "package.json")) {
		return false
	}
	if !hasPackageDependency(repoDir, "expo") {
		return false
	}
	return fileExists(filepath.Join(repoDir, "app.json")) ||
		fileExists(filepath.Join(repoDir, "app.config.js")) ||
		fileExists(filepath.Join(repoDir, "app.config.ts"))
}
func (b *ExpoWebBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, `sh -lc "npx expo export --platform web --output-dir dist || npx expo export -p web --output-dir dist || npx expo export --platform web"`)

	return BuildPlan{
		Kind:        "expo-web",
		ServicePort: 80,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s

FROM %s
COPY --from=builder /app/dist /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, install), nodeRunStepWithCaches(repoDir, build), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg))

			return writeStaticDockerArtifacts(repoDir, df, false)
		},
		Cleanup: func(repoDir string) error {
			return cleanupStaticDockerArtifacts(repoDir, true)
		},
	}, nil
}

type NodeGenericBuildpack struct{}

type SprintUIBuildpack struct{}

func (b *SprintUIBuildpack) Name() string { return "sprint-ui" }
func (b *SprintUIBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "sprint-ui") {
		return true
	}
	if !fileExists(filepath.Join(repoDir, "package.json")) {
		return false
	}
	if !fileExists(filepath.Join(repoDir, "config.sui")) {
		return false
	}
	return fileExists(filepath.Join(repoDir, ".sprint", "build.js")) ||
		fileExists(filepath.Join(repoDir, ".sprint", "server.js"))
}
func (b *SprintUIBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, `npm run build`)
	start := firstNonEmpty(req.StartCmd, `if [ -f build/server.mjs ]; then exec node build/server.mjs; else exec serve -s build -l ${PORT:-3000}; fi`)
	port := firstNonZero(req.ServicePort, 3000)

	return BuildPlan{
		Kind:        "sprint-ui",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
WORKDIR /app
COPY . .
%s
RUN rm -rf node_modules

%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=%d
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
RUN npm install -g serve@14
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, install), nodeRunStepWithCaches(repoDir, build), nodeProdDepsStage(repoDir), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, shellJSON(start))

			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.RemoveAll(filepath.Join(repoDir, "build"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

func (b *NodeGenericBuildpack) Name() string { return "node-generic" }
func (b *NodeGenericBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "node") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "package.json"))
}
func (b *NodeGenericBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, nodeDefaultBuildCmd(repoDir))
	start := firstNonEmpty(req.StartCmd, nodeDefaultStartCmd(repoDir))
	port := firstNonZero(req.ServicePort, 3000)

	return BuildPlan{
		Kind:        "node",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			// Keep dev deps in the builder and derive a production-only dependency set for runtime.
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s
RUN rm -rf node_modules

%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=%d
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, install), nodeRunStepWithCaches(repoDir, build), nodeProdDepsStage(repoDir), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, shellJSON(start))

			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type GoBuildpack struct{}

func (b *GoBuildpack) Name() string { return "go" }
func (b *GoBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "go") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "go.mod"))
}
func (b *GoBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_GO_IMAGE", "golang:1.22")
	runImg := getenv("RELAY_GO_RUN_IMAGE", "gcr.io/distroless/base-debian12")

	port := firstNonZero(req.ServicePort, 8080)
	install := "" // go mod download included in build
	build := firstNonEmpty(req.BuildCmd, fmt.Sprintf(`sh -lc "go mod download && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags=\"-s -w\" -o /out/app ./"`))
	start := firstNonEmpty(req.StartCmd, `/app/app`)

	return BuildPlan{
		Kind:        "go",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
		WORKDIR /src
		COPY go.mod go.sum ./
		RUN go mod download
		COPY . .
		RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/app ./

		FROM %s
		WORKDIR /app
		COPY --from=builder /out/app /app/app
		ENV PORT=%d
		EXPOSE %d
		USER 65532:65532
		ENTRYPOINT ["/app/app"]
		`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		// Verify is not performed on host; docker build will fail if build step fails.
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type DotnetBuildpack struct{}

func (b *DotnetBuildpack) Name() string { return "dotnet" }
func (b *DotnetBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "dotnet") {
		return true
	}
	// Detect common .NET solution/project files
	m, _ := filepath.Glob(filepath.Join(repoDir, "*.sln"))
	if len(m) > 0 {
		return true
	}
	m, _ = filepath.Glob(filepath.Join(repoDir, "*.csproj"))
	return len(m) > 0
}
func (b *DotnetBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_DOTNET_SDK_IMAGE", "mcr.microsoft.com/dotnet/sdk:8.0")
	runImg := getenv("RELAY_DOTNET_ASPNET_IMAGE", "mcr.microsoft.com/dotnet/aspnet:8.0")

	port := firstNonZero(req.ServicePort, 8080)
	install := "" // restore in build
	project := dotnetPickEntry(repoDir)
	if project == "" {
		return BuildPlan{}, fmt.Errorf("dotnet: no .sln or .csproj found")
	}

	build := firstNonEmpty(req.BuildCmd, fmt.Sprintf(`sh -lc "dotnet restore %s && dotnet publish %s -c Release -o /out"`, shQuote(project), shQuote(project)))
	start := firstNonEmpty(req.StartCmd, `dotnet app.dll`)

	return BuildPlan{
		Kind:        "dotnet",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY . .
RUN dotnet restore %s
RUN dotnet publish %s -c Release -o /out

FROM %s
WORKDIR /app
COPY --from=builder /out ./
ENV ASPNETCORE_URLS=http://0.0.0.0:%d
EXPOSE %d
CMD ["sh","-lc","dll=$(ls *.dll | head -n1); echo Running $dll; dotnet $dll"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), shQuote(project), shQuote(project), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		// No host-side Verify; docker build ensures publish succeeded.
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type PythonBuildpack struct{}

func (b *PythonBuildpack) Name() string { return "python" }
func (b *PythonBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "python") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "requirements.txt")) ||
		fileExists(filepath.Join(repoDir, "pyproject.toml")) ||
		fileExists(filepath.Join(repoDir, "Pipfile"))
}
func (b *PythonBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_PY_IMAGE", "python:3.12")
	runImg := getenv("RELAY_PY_RUN_IMAGE", "python:3.12-slim")

	port := firstNonZero(req.ServicePort, 8000)

	install := firstNonEmpty(req.InstallCmd, pythonInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, "") // optional
	start := firstNonEmpty(req.StartCmd, pythonDefaultStart(repoDir))

	return BuildPlan{
		Kind:        "python",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			install := firstNonEmpty(req.InstallCmd, pythonInstallCmd(repoDir))
			if strings.TrimSpace(install) == "" {
				install = `sh -lc "pip install --no-cache-dir -U pip"`
			}

			df := fmt.Sprintf(`FROM %s
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV PORT=%d
COPY . .
RUN %s
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, install, port, shellForm(start))
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type JavaBuildpack struct{}

func (b *JavaBuildpack) Name() string { return "java" }
func (b *JavaBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "java") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "pom.xml")) || fileExists(filepath.Join(repoDir, "build.gradle")) || fileExists(filepath.Join(repoDir, "build.gradle.kts"))
}
func (b *JavaBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_JAVA_BUILD_IMAGE", "maven:3.9-eclipse-temurin-21")
	runImg := getenv("RELAY_JAVA_RUN_IMAGE", "eclipse-temurin:21-jre")

	port := firstNonZero(req.ServicePort, 8080)
	install := "" // maven/gradle handles
	build := firstNonEmpty(req.BuildCmd, javaDefaultBuild(repoDir))
	start := firstNonEmpty(req.StartCmd, `java -jar app.jar`)

	return BuildPlan{
		Kind:        "java",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			isMaven := fileExists(filepath.Join(repoDir, "pom.xml"))
			isGradle := fileExists(filepath.Join(repoDir, "build.gradle")) || fileExists(filepath.Join(repoDir, "build.gradle.kts"))
			if isMaven {
				df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY pom.xml mvnw* .
COPY .mvn .mvn
COPY src ./src
RUN mvn -q -DskipTests package

FROM %s
WORKDIR /app
COPY --from=builder /src/target/*.jar /app/app.jar
ENV PORT=%d
EXPOSE %d
CMD ["java","-jar","/app/app.jar"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
				return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
			} else if isGradle {
				gradleCmd := "gradle build -x test"
				if fileExists(filepath.Join(repoDir, "gradlew")) {
					gradleCmd = "./gradlew build -x test"
				}
				df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY . .
RUN %s

FROM %s
WORKDIR /app
COPY --from=builder /src/build/libs/*.jar /app/app.jar
ENV PORT=%d
EXPOSE %d
CMD ["java","-jar","/app/app.jar"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), gradleCmd, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
				return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
			} else {
				return fmt.Errorf("java: no pom.xml or build.gradle found")
			}
		},
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type RustBuildpack struct{}

func (b *RustBuildpack) Name() string { return "rust" }
func (b *RustBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "rust") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "Cargo.toml"))
}
func (b *RustBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_RUST_IMAGE", "rust:1.77")
	runImg := getenv("RELAY_RUST_RUN_IMAGE", "debian:bookworm-slim")

	port := firstNonZero(req.ServicePort, 8080)
	// Let the Dockerfile perform the cargo build and extraction; avoid noisy default copy here.
	build := firstNonEmpty(req.BuildCmd, "")
	start := firstNonEmpty(req.StartCmd, `/app/app`)

	return BuildPlan{
		Kind:        "rust",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY Cargo.toml Cargo.lock ./
RUN cargo fetch
COPY . .
RUN cargo build --release && set -eu; \
  bin=$(find target/release -maxdepth 1 -type f -executable \
    ! -name "*.d" ! -name "*.rlib" ! -name "*.so" ! -name "*.a" | head -n1); \
  test -n "$bin"; mkdir -p /out; cp "$bin" /out/app

FROM %s
WORKDIR /app
COPY --from=builder /out/app /app/app
ENV PORT=%d
EXPOSE %d
USER 65532:65532
ENTRYPOINT ["/app/app"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)

			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type CCppBuildpack struct{}

func (b *CCppBuildpack) Name() string { return "c-cpp" }
func (b *CCppBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "c") || strings.EqualFold(cfg.Kind, "cpp") || strings.EqualFold(cfg.Kind, "c-cpp")) {
		return true
	}
	if fileExists(filepath.Join(repoDir, "CMakeLists.txt")) || fileExists(filepath.Join(repoDir, "Makefile")) {
		return true
	}
	return len(findFilesByExt(repoDir, ".c", ".cc", ".cpp", ".cxx")) > 0
}
func (b *CCppBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_CC_IMAGE", "debian:bookworm")
	runImg := getenv("RELAY_CC_RUN_IMAGE", "debian:bookworm-slim")
	port := firstNonZero(req.ServicePort, 8080)
	install := firstNonEmpty(req.InstallCmd, `sh -lc "apt-get update && apt-get install -y --no-install-recommends build-essential cmake pkg-config && rm -rf /var/lib/apt/lists/*"`)
	build := firstNonEmpty(req.BuildCmd, cCppDefaultBuildCmd(repoDir))
	start := firstNonEmpty(req.StartCmd, `/app/app`)

	return BuildPlan{
		Kind:        "c-cpp",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
RUN %s
COPY . .
RUN %s

FROM %s
WORKDIR /app
COPY --from=builder /out/app /app/app
ENV PORT=%d
EXPOSE %d
CMD ["/app/app"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), install, build, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.RemoveAll(filepath.Join(repoDir, "build"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type WasmStaticBuildpack struct{}

func (b *WasmStaticBuildpack) Name() string { return "wasm-static" }
func (b *WasmStaticBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "wasm-static") {
		return true
	}
	if !hasWasmAssets(repoDir) {
		return false
	}
	return fileExists(filepath.Join(repoDir, "index.html")) || fileExists(filepath.Join(repoDir, "public", "index.html"))
}
func (b *WasmStaticBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	return BuildPlan{
		Kind:        "wasm-static",
		ServicePort: 80,
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		WriteDockerfile: func(repoDir string) error {
			root := "."
			if fileExists(filepath.Join(repoDir, "public", "index.html")) {
				root = "public"
			}
			df := fmt.Sprintf(`FROM %s
COPY %s /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), root)
			return writeStaticDockerArtifacts(repoDir, df, true)
		},
		Cleanup: func(repoDir string) error {
			return cleanupStaticDockerArtifacts(repoDir, false)
		},
	}, nil
}

type StaticBuildpack struct{}

func (b *StaticBuildpack) Name() string { return "static" }
func (b *StaticBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "static") {
		return true
	}
	// If there's an index.html at root and no package.json/go.mod/etc, treat as static
	if fileExists(filepath.Join(repoDir, "index.html")) && !fileExists(filepath.Join(repoDir, "package.json")) && !fileExists(filepath.Join(repoDir, "go.mod")) {
		return true
	}
	// common "public" static folder
	if fileExists(filepath.Join(repoDir, "public", "index.html")) && !fileExists(filepath.Join(repoDir, "package.json")) && !fileExists(filepath.Join(repoDir, "go.mod")) {
		return true
	}
	return false
}
func (b *StaticBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	port := 80
	return BuildPlan{
		Kind:        "static",
		ServicePort: port,
		BuildImage:  "",
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		WriteDockerfile: func(repoDir string) error {
			// serve ./public if present else serve repo root
			root := "."
			if fileExists(filepath.Join(repoDir, "public", "index.html")) {
				root = "public"
			}
			df := fmt.Sprintf(`FROM %s
COPY %s /usr/share/nginx/html
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), root)
			return writeStaticDockerArtifacts(repoDir, df, false)
		},
		Cleanup: func(repoDir string) error {
			return cleanupStaticDockerArtifacts(repoDir, false)
		},
	}, nil
}

// ---------------------- Multi-service helpers ----------------------

// readProjectConfig reads relay.json from the repo root.
func readProjectConfig(repoDir string) (*ProjectConfig, error) {
	p := filepath.Join(repoDir, "relay.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c ProjectConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if len(c.Services) == 0 {
		return nil, fmt.Errorf("relay.json has no services")
	}
	return &c, nil
}

func normalizeServiceConfig(svc ServiceConfig) ServiceConfig {
	svc.Name = strings.TrimSpace(svc.Name)
	svc.Type = strings.ToLower(strings.TrimSpace(svc.Type))
	svc.Version = strings.TrimSpace(svc.Version)
	svc.Image = strings.TrimSpace(svc.Image)
	svc.Command = strings.TrimSpace(svc.Command)
	if svc.Env == nil {
		svc.Env = map[string]string{}
	}
	nextVolumes := make([]string, 0, len(svc.Volumes))
	for _, vol := range svc.Volumes {
		vol = strings.TrimSpace(vol)
		if vol != "" {
			nextVolumes = append(nextVolumes, vol)
		}
	}
	svc.Volumes = nextVolumes
	if svc.Health != nil {
		svc.Health.Test = strings.TrimSpace(svc.Health.Test)
	}
	return svc
}

func serviceShouldRun(svc ServiceConfig) bool {
	svc = normalizeServiceConfig(svc)
	return svc.Name != "" && !svc.Disabled && !svc.Stopped && !strings.EqualFold(svc.Type, "app")
}

func serviceConfigHash(svc ServiceConfig) string {
	svc = normalizeServiceConfig(svc)
	b, _ := json.Marshal(svc)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func defaultServiceType(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "postgres", "mysql", "redis", "mongo", "worker", "custom":
		return strings.ToLower(strings.TrimSpace(kind))
	default:
		return "custom"
	}
}

// serviceImageName returns the Docker image to use for a companion service.
func serviceImageName(svc ServiceConfig) string {
	if strings.TrimSpace(svc.Image) != "" {
		return strings.TrimSpace(svc.Image)
	}
	version := strings.TrimSpace(svc.Version)
	switch strings.ToLower(svc.Type) {
	case "postgres":
		if version == "" {
			version = "16"
		}
		return "postgres:" + version
	case "mysql":
		if version == "" {
			version = "8"
		}
		return "mysql:" + version
	case "redis":
		if version == "" {
			version = "7"
		}
		return "redis:" + version + "-alpine"
	case "mongo":
		if version == "" {
			version = "7"
		}
		return "mongo:" + version
	case "worker":
		return ""
	case "custom":
		return ""
	default:
		if version == "" {
			return svc.Type
		}
		return svc.Type + ":" + version
	}
}

// serviceDefaultPort returns the default container port for a service type.
func serviceDefaultPort(svcType string) int {
	switch strings.ToLower(svcType) {
	case "postgres":
		return 5432
	case "mysql":
		return 3306
	case "redis":
		return 6379
	case "mongo":
		return 27017
	case "worker", "custom":
		return 0
	default:
		return 5432
	}
}

func serviceHostAliasName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if label == "" {
			return ""
		}
		for i, r := range label {
			isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
			isDigit := r >= '0' && r <= '9'
			if !isLetter && !isDigit && r != '-' {
				return ""
			}
			if (i == 0 || i == len(label)-1) && r == '-' {
				return ""
			}
		}
	}
	return name
}

func serviceEndpointForRuntime(runtime ContainerRuntime, svc ServiceConfig, containerName string, port int) (string, int) {
	if _, ok := runtime.(*StationRuntime); ok {
		if alias := serviceHostAliasName(svc.Name); alias != "" && strings.TrimSpace(runtime.ContainerIP(containerName)) != "" {
			return alias, port
		}
		if ip := strings.TrimSpace(runtime.ContainerIP(containerName)); ip != "" {
			return ip, port
		}
		if published := runtime.PublishedPort(containerName, port); published > 0 {
			return "127.0.0.1", published
		}
	}
	return containerName, port
}

func (s *Server) serviceHostAliasesForRuntime(runtime ContainerRuntime, app string, env DeployEnv, branch string) []string {
	if _, ok := runtime.(*StationRuntime); !ok {
		return nil
	}
	services, err := s.getProjectServices(app, string(env), branch)
	if err != nil || len(services) == 0 {
		return nil
	}
	aliases := make([]string, 0, len(services))
	seen := map[string]struct{}{}
	for _, svc := range services {
		alias := serviceHostAliasName(svc.Name)
		if alias == "" || !runtime.IsRunning(svc.Container) {
			continue
		}
		ip := strings.TrimSpace(runtime.ContainerIP(svc.Container))
		if ip == "" {
			continue
		}
		entry := alias + ":" + ip
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		aliases = append(aliases, entry)
	}
	sort.Strings(aliases)
	return aliases
}

// serviceEnvInfo returns the env var name and connection URL for a companion service.
func serviceEnvInfo(svc ServiceConfig, host string, port int) (key, val string) {
	svc = normalizeServiceConfig(svc)
	if port == 0 {
		port = svc.Port
	}
	if port == 0 {
		port = serviceDefaultPort(svc.Type)
	}
	switch strings.ToLower(svc.Type) {
	case "postgres":
		return strings.ToUpper(svc.Name) + "_URL",
			fmt.Sprintf("postgres://relay:relay@%s:%d/relay?sslmode=disable", host, port)
	case "mysql":
		return strings.ToUpper(svc.Name) + "_URL",
			fmt.Sprintf("mysql://relay:relay@%s:%d/relay", host, port)
	case "redis":
		return strings.ToUpper(svc.Name) + "_URL",
			fmt.Sprintf("redis://%s:%d", host, port)
	case "mongo":
		return strings.ToUpper(svc.Name) + "_URL",
			fmt.Sprintf("mongodb://relay:relay@%s:%d/relay", host, port)
	case "custom":
		if port > 0 {
			return strings.ToUpper(svc.Name) + "_URL",
				fmt.Sprintf("http://%s:%d", host, port)
		}
		return strings.ToUpper(svc.Name) + "_HOST", host
	default:
		return strings.ToUpper(svc.Name) + "_HOST", host
	}
}

func serviceBaseArgs(svc ServiceConfig, volumeName string) []string {
	if len(svc.Volumes) > 0 {
		args := make([]string, 0, len(svc.Volumes)*2+8)
		for _, mount := range svc.Volumes {
			args = append(args, "-v", mount)
		}
		switch strings.ToLower(svc.Type) {
		case "postgres":
			args = append(args, "-e", "POSTGRES_USER=relay", "-e", "POSTGRES_PASSWORD=relay", "-e", "POSTGRES_DB=relay")
		case "mysql":
			args = append(args, "-e", "MYSQL_ROOT_PASSWORD=relay", "-e", "MYSQL_USER=relay", "-e", "MYSQL_PASSWORD=relay", "-e", "MYSQL_DATABASE=relay")
		case "mongo":
			args = append(args, "-e", "MONGO_INITDB_ROOT_USERNAME=relay", "-e", "MONGO_INITDB_ROOT_PASSWORD=relay", "-e", "MONGO_INITDB_DATABASE=relay")
		}
		return args
	}
	switch strings.ToLower(svc.Type) {
	case "postgres":
		return []string{
			"-e", "POSTGRES_USER=relay",
			"-e", "POSTGRES_PASSWORD=relay",
			"-e", "POSTGRES_DB=relay",
			"-v", volumeName + ":/var/lib/postgresql/data",
		}
	case "mysql":
		return []string{
			"-e", "MYSQL_ROOT_PASSWORD=relay",
			"-e", "MYSQL_USER=relay",
			"-e", "MYSQL_PASSWORD=relay",
			"-e", "MYSQL_DATABASE=relay",
			"-v", volumeName + ":/var/lib/mysql",
		}
	case "redis":
		return []string{"-v", volumeName + ":/data"}
	case "mongo":
		return []string{
			"-e", "MONGO_INITDB_ROOT_USERNAME=relay",
			"-e", "MONGO_INITDB_ROOT_PASSWORD=relay",
			"-e", "MONGO_INITDB_DATABASE=relay",
			"-v", volumeName + ":/data/db",
		}
	default:
		return nil
	}
}

func splitVolumeSpec(spec string) (source string, remainder string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", ""
	}
	if len(spec) >= 2 && spec[1] == ':' {
		if idx := strings.Index(spec[2:], ":"); idx >= 0 {
			return spec[:idx+2], spec[idx+2:]
		}
		return spec, ""
	}
	if idx := strings.Index(spec, ":"); idx >= 0 {
		return spec[:idx], spec[idx:]
	}
	return spec, ""
}

func isNamedVolumeSource(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	if filepath.IsAbs(source) {
		return false
	}
	return !strings.ContainsAny(source, `/\`)
}

func (s *Server) stationVolumeMount(spec string) string {
	source, remainder := splitVolumeSpec(spec)
	if !isNamedVolumeSource(source) {
		return spec
	}
	hostPath := filepath.Join(s.dataDir, "station-volumes", safe(source))
	mustMkdir(hostPath)
	return hostPath + remainder
}

func healthArgs(h *ServiceHealth) []string {
	if h == nil || strings.TrimSpace(h.Test) == "" {
		return nil
	}
	args := []string{"--health-cmd", strings.TrimSpace(h.Test)}
	if h.IntervalSeconds > 0 {
		args = append(args, "--health-interval", fmt.Sprintf("%ds", h.IntervalSeconds))
	}
	if h.TimeoutSeconds > 0 {
		args = append(args, "--health-timeout", fmt.Sprintf("%ds", h.TimeoutSeconds))
	}
	if h.Retries > 0 {
		args = append(args, "--health-retries", strconv.Itoa(h.Retries))
	}
	if h.StartPeriodSeconds > 0 {
		args = append(args, "--health-start-period", fmt.Sprintf("%ds", h.StartPeriodSeconds))
	}
	return args
}

func (s *Server) getProjectService(app, env, branch, name string) (*ProjectService, error) {
	row := s.db.QueryRow(
		`SELECT project, name, type, branch, env, container, network, volume, env_key, env_val, COALESCE(image,''), COALESCE(port,0), COALESCE(host_port,0), COALESCE(spec_hash,'')
		FROM project_services WHERE project=? AND env=? AND branch=? AND name=?`,
		app, env, branch, name,
	)
	var ps ProjectService
	if err := row.Scan(&ps.Project, &ps.Name, &ps.Type, &ps.Branch, &ps.Env, &ps.Container, &ps.Network, &ps.Volume, &ps.EnvKey, &ps.EnvVal, &ps.Image, &ps.Port, &ps.HostPort, &ps.SpecHash); err != nil {
		return nil, err
	}
	return &ps, nil
}

func (s *Server) deleteProjectServiceState(app, env, branch, name string) {
	_, _ = s.db.Exec(`DELETE FROM project_services WHERE project=? AND env=? AND branch=? AND name=?`, app, env, branch, name)
}

func (s *Server) stopProjectServiceRuntime(app, env, branch, name string) {
	if running, err := s.getProjectService(app, env, branch, name); err == nil && running != nil {
		s.runtime.Remove(running.Container)
		if s.stationRuntime != nil {
			s.stationRuntime.Remove(running.Container)
		}
	}
	s.deleteProjectServiceState(app, env, branch, name)
}

func (s *Server) appLaneRunning(app string, env DeployEnv, branch string) bool {
	names := []string{
		appBaseContainerName(app, env, branch),
		appSlotContainerName(app, env, branch, "blue"),
		appSlotContainerName(app, env, branch, "green"),
		stationAppName(app, env, branch),
	}
	for _, rt := range []ContainerRuntime{s.runtime, s.stationRuntime} {
		if rt == nil {
			continue
		}
		for _, name := range names {
			if rt.IsRunning(name) {
				return true
			}
		}
	}
	return false
}

// startProjectService ensures a companion service container is running on the given network.
// Returns the env key+value to inject into the app container.
func (s *Server) startProjectService(
	log func(string, ...any),
	app, env, branch string,
	svc ServiceConfig,
	networkName string,
	force bool,
) (envKey, envVal string, err error) {
	svc = normalizeServiceConfig(svc)
	runtime := s.runtime // companion services always run through the Docker runtime
	containerName := fmt.Sprintf("relay__%s__%s__%s__svc__%s", safe(app), safe(env), safe(branch), safe(svc.Name))
	volumeName := containerName + "_data"
	image := serviceImageName(svc)
	port := svc.Port
	if port == 0 {
		port = serviceDefaultPort(svc.Type)
	}
	hostPort := svc.HostPort

	envKey, envVal = serviceEnvInfo(svc, containerName, port)
	specHash := serviceConfigHash(svc)

	// Check if already running.
	if runtime.IsRunning(containerName) && !force {
		if current, err := s.getProjectService(app, env, branch, svc.Name); err == nil && current.SpecHash == specHash {
			log("service %s already running (%s)", svc.Name, containerName)
			_ = runtime.NetworkConnect(containerName, networkName)
			if current.EnvKey != "" || current.EnvVal != "" {
				return current.EnvKey, current.EnvVal, nil
			}
			host, resolvedPort := serviceEndpointForRuntime(runtime, svc, containerName, port)
			envKey, envVal = serviceEnvInfo(svc, host, resolvedPort)
			return envKey, envVal, nil
		}
		log("service %s config changed, recreating (%s)", svc.Name, containerName)
	}
	if image == "" {
		return envKey, envVal, fmt.Errorf("service %s needs an image", svc.Name)
	}

	// Remove stale stopped container.
	s.runtime.Remove(containerName)
	if s.stationRuntime != nil {
		s.stationRuntime.Remove(containerName)
	}

	// Build the service ContainerSpec from per-type base args and user overrides.
	var envs, volumes []string
	baseArgs := serviceBaseArgs(svc, volumeName)
	for i := 0; i+1 < len(baseArgs); i += 2 {
		switch baseArgs[i] {
		case "-e":
			envs = append(envs, baseArgs[i+1])
		case "-v":
			volumes = append(volumes, baseArgs[i+1])
		}
	}
	for key, value := range svc.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}
	var ports []string
	if hostPort > 0 && port > 0 {
		ports = append(ports, fmt.Sprintf("%d:%d", hostPort, port))
	}
	var cmd []string
	if svc.Command != "" {
		cmd = strings.Fields(svc.Command)
	}
	spec := ContainerSpec{
		Name:          containerName,
		Image:         image,
		Network:       networkName,
		RestartPolicy: "unless-stopped",
		Env:           envs,
		Volumes:       volumes,
		PortBindings:  ports,
		HealthArgs:    healthArgs(svc.Health),
		Command:       cmd,
	}

	log("starting companion service %s (image=%s container=%s)", svc.Name, image, containerName)
	if err := runtime.RunDetached(spec); err != nil {
		return envKey, envVal, err
	}
	host, resolvedPort := serviceEndpointForRuntime(runtime, svc, containerName, port)
	envKey, envVal = serviceEnvInfo(svc, host, resolvedPort)

	// Save service state to DB.
	_ = s.saveProjectService(&ProjectService{
		Project:   app,
		Name:      svc.Name,
		Type:      svc.Type,
		Branch:    branch,
		Env:       env,
		Container: containerName,
		Network:   networkName,
		Volume:    volumeName,
		EnvKey:    envKey,
		EnvVal:    envVal,
		Image:     image,
		Port:      port,
		HostPort:  hostPort,
		SpecHash:  specHash,
	})

	log("service %s started: %s=%s", svc.Name, envKey, envVal)
	return envKey, envVal, nil
}

func (s *Server) saveProjectService(ps *ProjectService) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO project_services
		(project, name, type, branch, env, container, network, volume, env_key, env_val, image, port, host_port, spec_hash, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ps.Project, ps.Name, ps.Type, ps.Branch, ps.Env,
		ps.Container, ps.Network, ps.Volume, ps.EnvKey, ps.EnvVal, ps.Image, ps.Port, ps.HostPort, ps.SpecHash,
		time.Now().UnixMilli(),
	)
	return err
}

func (s *Server) getProjectServices(app, env, branch string) ([]ProjectService, error) {
	rows, err := s.db.Query(
		`SELECT project, name, type, branch, env, container, network, volume, env_key, env_val, COALESCE(image,''), COALESCE(port,0), COALESCE(host_port,0), COALESCE(spec_hash,'')
		FROM project_services WHERE project=? AND env=? AND branch=?`,
		app, env, branch,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectService
	for rows.Next() {
		var ps ProjectService
		if err := rows.Scan(&ps.Project, &ps.Name, &ps.Type, &ps.Branch, &ps.Env,
			&ps.Container, &ps.Network, &ps.Volume, &ps.EnvKey, &ps.EnvVal, &ps.Image, &ps.Port, &ps.HostPort, &ps.SpecHash); err != nil {
			continue
		}
		out = append(out, ps)
	}
	return out, nil
}

func (s *Server) saveServiceSpec(app, env, branch string, svc ServiceConfig) error {
	svc = normalizeServiceConfig(svc)
	raw, _ := json.Marshal(svc)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO project_service_specs
		(project, env, branch, name, config_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		app, env, branch, svc.Name, string(raw), time.Now().UnixMilli(),
	)
	return err
}

func (s *Server) deleteServiceSpec(app, env, branch, name string) error {
	_, err := s.db.Exec(`DELETE FROM project_service_specs WHERE project=? AND env=? AND branch=? AND name=?`, app, env, branch, name)
	return err
}

func (s *Server) getServiceSpecs(app, env, branch string) ([]ServiceSpecRecord, error) {
	rows, err := s.db.Query(
		`SELECT project, env, branch, name, config_json, updated_at
		FROM project_service_specs WHERE project=? AND env=? AND branch=?
		ORDER BY name`,
		app, env, branch,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServiceSpecRecord
	for rows.Next() {
		var rec ServiceSpecRecord
		var raw string
		if err := rows.Scan(&rec.Project, &rec.Env, &rec.Branch, &rec.Name, &raw, &rec.UpdatedAt); err != nil {
			continue
		}
		if err := json.Unmarshal([]byte(raw), &rec.Config); err != nil {
			continue
		}
		rec.Config = normalizeServiceConfig(rec.Config)
		out = append(out, rec)
	}
	return out, nil
}

func (s *Server) resolveCompanionSpecs(app string, env DeployEnv, branch, repoDir string) ([]ServiceConfig, error) {
	merged := map[string]ServiceConfig{}
	if projectCfg, err := readProjectConfig(repoDir); err == nil && projectCfg != nil {
		for _, svc := range projectCfg.Services {
			svc = normalizeServiceConfig(svc)
			if svc.Name == "" {
				continue
			}
			merged[svc.Name] = svc
		}
	}
	specs, err := s.getServiceSpecs(app, string(env), branch)
	if err == nil {
		for _, rec := range specs {
			svc := normalizeServiceConfig(rec.Config)
			if svc.Name == "" {
				continue
			}
			if svc.Disabled {
				delete(merged, svc.Name)
				continue
			}
			merged[svc.Name] = svc
		}
	}
	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ServiceConfig, 0, len(names))
	for _, name := range names {
		svc := merged[name]
		if svc.Disabled || strings.EqualFold(svc.Type, "app") {
			continue
		}
		out = append(out, svc)
	}
	return out, nil
}

func (s *Server) reconcileProjectServices(log func(string, ...any), app string, env DeployEnv, branch string, desired map[string]ServiceConfig) {
	current, err := s.getProjectServices(app, string(env), branch)
	if err != nil {
		return
	}
	for _, svc := range current {
		want, ok := desired[svc.Name]
		if ok && serviceShouldRun(want) {
			continue
		}
		if log != nil {
			if ok {
				log("stopping companion service %s because it is kept off", svc.Name)
			} else {
				log("removing stale companion service %s", svc.Name)
			}
		}
		s.stopProjectServiceRuntime(app, string(env), branch, svc.Name)
	}
}

// ---------------------- Preview URL helper ----------------------

// autoPreviewHost reports whether the active lane policy would auto-assign
// a managed hostname when RELAY_BASE_DOMAIN is configured.
func autoPreviewHost(app, branch string) string {
	base := strings.TrimSpace(os.Getenv("RELAY_BASE_DOMAIN"))
	if !laneNeedsManagedHost(defaultLanePolicy(EnvPreview), base) {
		return ""
	}
	return fmt.Sprintf("%s-%s.%s", safe(app), safe(branch), base)
}

// serverConfigGet reads a single key from the server_config table.
// Returns "" if the key doesn't exist or on any error.
func (s *Server) serverConfigGet(key string) string {
	var val string
	_ = s.db.QueryRow(`SELECT value FROM server_config WHERE key=?`, key).Scan(&val)
	return strings.TrimSpace(val)
}

// serverBaseDomain returns the effective base domain: DB value first, then env var.
func (s *Server) serverBaseDomain() string {
	if base := s.serverConfigGet("base_domain"); base != "" {
		return base
	}
	return strings.TrimSpace(os.Getenv("RELAY_BASE_DOMAIN"))
}

func (s *Server) serverDashboardHost() string {
	if host := s.serverConfigGet("dashboard_host"); host != "" {
		return host
	}
	return strings.TrimSpace(os.Getenv("RELAY_DASHBOARD_HOST"))
}

func listenAddrPort(addr string) int {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0
	}
	_, port, err := net.SplitHostPort(addr)
	if err == nil {
		p, _ := strconv.Atoi(port)
		return p
	}
	if strings.HasPrefix(addr, ":") {
		p, _ := strconv.Atoi(strings.TrimPrefix(addr, ":"))
		return p
	}
	return 0
}

// autoPreviewHostFull resolves the effective managed hostname for the given lane.
func (s *Server) autoPreviewHostFull(env DeployEnv, app, branch, existingHost string) string {
	return s.managedLaneHost(env, app, branch, existingHost)
}

// handleServerConfig handles GET/POST /api/server/config.
func (s *Server) handleServerConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, map[string]string{
			"base_domain":    s.serverBaseDomain(),
			"dashboard_host": s.serverDashboardHost(),
		})
	case http.MethodPost:
		var body struct {
			BaseDomain    string `json:"base_domain"`
			DashboardHost string `json:"dashboard_host"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid JSON")
			return
		}
		baseDomain := strings.TrimSpace(body.BaseDomain)
		dashboardHost := strings.TrimSpace(body.DashboardHost)
		if err := validateProxyHostname(baseDomain, "base_domain"); err != nil {
			httpError(w, 400, err.Error())
			return
		}
		if err := validateProxyHostname(dashboardHost, "dashboard_host"); err != nil {
			httpError(w, 400, err.Error())
			return
		}
		previous := map[string]string{
			"base_domain":    s.serverConfigGet("base_domain"),
			"dashboard_host": s.serverConfigGet("dashboard_host"),
		}
		tx, err := s.db.Begin()
		if err != nil {
			httpError(w, 500, "db begin failed: "+err.Error())
			return
		}
		for key, value := range map[string]string{
			"base_domain":    baseDomain,
			"dashboard_host": dashboardHost,
		} {
			if _, err := tx.Exec(
				`INSERT INTO server_config (key, value) VALUES (?, ?)
				 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
				key, value,
			); err != nil {
				_ = tx.Rollback()
				httpError(w, 500, "db write failed: "+err.Error())
				return
			}
		}
		if err := tx.Commit(); err != nil {
			httpError(w, 500, "db write failed: "+err.Error())
			return
		}
		if err := s.ensureGlobalProxy(); err != nil {
			_ = s.writeServerConfig(previous)
			httpError(w, 500, "failed to refresh global proxy: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{
			"base_domain":    baseDomain,
			"dashboard_host": dashboardHost,
		})
	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) writeServerConfig(values map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for key, value := range values {
		if _, err := tx.Exec(
			`INSERT INTO server_config (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			key, strings.TrimSpace(value),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ---------------------- Buildpack Plugins API ----------------------

func (s *Server) pluginBuildpacksDir() string {
	return filepath.Join(s.pluginsDir, "buildpacks")
}

func (s *Server) reloadBuildpacks() error {
	plugins, err := s.loadPluginBuildpacks()
	if err != nil {
		return err
	}
	packs := make([]Buildpack, 0, len(plugins)+len(defaultBuildpacks()))
	for _, plugin := range plugins {
		packs = append(packs, &PluginBuildpack{plugin: plugin})
	}
	packs = append(packs, defaultBuildpacks()...)
	s.buildpacks = packs
	return nil
}

func (s *Server) loadPluginBuildpacks() ([]*BuildpackPlugin, error) {
	dir := s.pluginBuildpacksDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var plugins []*BuildpackPlugin
	for _, ent := range ents {
		if ent.IsDir() || !strings.HasSuffix(strings.ToLower(ent.Name()), ".json") {
			continue
		}
		p := filepath.Join(dir, ent.Name())
		plugin, err := readBuildpackPluginFile(p)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	sort.SliceStable(plugins, func(i, j int) bool {
		if plugins[i].Priority == plugins[j].Priority {
			return plugins[i].Name < plugins[j].Name
		}
		return plugins[i].Priority > plugins[j].Priority
	})
	return plugins, nil
}

func readBuildpackPluginFile(p string) (*BuildpackPlugin, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var plugin BuildpackPlugin
	if err := json.Unmarshal(b, &plugin); err != nil {
		return nil, err
	}
	if err := validateBuildpackPlugin(&plugin); err != nil {
		return nil, err
	}
	return &plugin, nil
}

func validateBuildpackPlugin(plugin *BuildpackPlugin) error {
	name := safe(plugin.Name)
	if name == "" || name == "x" {
		return fmt.Errorf("plugin name required")
	}
	plugin.Name = name
	plugin.PlanSpec.Kind = firstNonEmpty(strings.TrimSpace(plugin.PlanSpec.Kind), plugin.Name)
	if plugin.PlanSpec.ServicePort < 0 {
		return fmt.Errorf("service_port must be >= 0")
	}
	if strings.TrimSpace(plugin.PlanSpec.DockerfileTemplate) == "" {
		return fmt.Errorf("dockerfile_template required")
	}
	rules := plugin.DetectRules
	if strings.TrimSpace(rules.Kind) == "" &&
		len(rules.Kinds) == 0 &&
		len(rules.FilesAny) == 0 &&
		len(rules.FilesAll) == 0 &&
		len(rules.DirsAny) == 0 &&
		len(rules.DirsAll) == 0 &&
		len(rules.PackageDepsAny) == 0 &&
		len(rules.PackageDepsAll) == 0 &&
		len(rules.FileExtensionsAny) == 0 &&
		len(rules.FileExtensionsAll) == 0 {
		return fmt.Errorf("plugin %s must define at least one detect rule", plugin.Name)
	}
	return nil
}

func (s *Server) handleBuildpackPlugins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		plugins, err := s.loadPluginBuildpacks()
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, plugins)
	case http.MethodPost:
		if !s.pluginMutationsEnabled() {
			httpError(w, 403, "plugin mutations are disabled on this server")
			return
		}
		var plugin BuildpackPlugin
		if err := json.NewDecoder(r.Body).Decode(&plugin); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		if err := validateBuildpackPlugin(&plugin); err != nil {
			httpError(w, 400, err.Error())
			return
		}
		path := filepath.Join(s.pluginBuildpacksDir(), safe(plugin.Name)+".json")
		body, _ := json.MarshalIndent(plugin, "", "  ")
		if err := os.WriteFile(path, body, 0644); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if err := s.reloadBuildpacks(); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 201, plugin)
	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) handleBuildpackPluginByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/plugins/buildpacks/")
	name = safe(path.Base(strings.TrimSpace(name)))
	if name == "" || name == "x" {
		httpError(w, 400, "plugin name required")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if !s.pluginMutationsEnabled() {
			httpError(w, 403, "plugin mutations are disabled on this server")
			return
		}
		if err := os.Remove(filepath.Join(s.pluginBuildpacksDir(), name+".json")); err != nil {
			if os.IsNotExist(err) {
				httpError(w, 404, "plugin not found")
				return
			}
			httpError(w, 500, err.Error())
			return
		}
		if err := s.reloadBuildpacks(); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"deleted": name})
	default:
		httpError(w, 405, "method not allowed")
	}
}

// ---------------------- Projects API handler ----------------------

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}

	rows, err := s.db.Query(
		`SELECT app, env, branch, COALESCE(engine,''), mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(active_slot,''), COALESCE(standby_slot,''), COALESCE(drain_until,0), COALESCE(traffic_mode,''), COALESCE(access_policy,''), COALESCE(ip_allowlist,''), COALESCE(expires_at,0), COALESCE(webhook_secret,''), repo_url, COALESCE(stopped,0)
		FROM app_state ORDER BY app, env, branch`,
	)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	type ProjectEnv struct {
		App              string `json:"app"`
		Env              string `json:"env"`
		Branch           string `json:"branch"`
		Engine           string `json:"engine,omitempty"`
		Mode             string `json:"mode"`
		HostPort         int    `json:"host_port"`
		HostPortExplicit bool   `json:"host_port_explicit,omitempty"`
		ServicePort      int    `json:"service_port"`
		PublicHost       string `json:"public_host"`
		ActiveSlot       string `json:"active_slot,omitempty"`
		StandbySlot      string `json:"standby_slot,omitempty"`
		DrainUntil       int64  `json:"drain_until,omitempty"`
		TrafficMode      string `json:"traffic_mode,omitempty"`
		AccessPolicy     string `json:"access_policy,omitempty"`
		IPAllowlist      string `json:"ip_allowlist,omitempty"`
		ExpiresAt        int64  `json:"expires_at,omitempty"`
		WebhookSecret    string `json:"webhook_secret,omitempty"`
		RepoURL          string `json:"repo_url"`
		Stopped          bool   `json:"stopped,omitempty"`
	}

	type ProjectInfo struct {
		Name     string           `json:"name"`
		Envs     []ProjectEnv     `json:"envs"`
		Services []ProjectService `json:"services"`
	}

	byApp := map[string]*ProjectInfo{}
	for rows.Next() {
		var pe ProjectEnv
		if err := rows.Scan(&pe.App, &pe.Env, &pe.Branch, &pe.Engine, &pe.Mode,
			&pe.HostPort, &pe.HostPortExplicit, &pe.ServicePort, &pe.PublicHost, &pe.ActiveSlot, &pe.StandbySlot, &pe.DrainUntil, &pe.TrafficMode, &pe.AccessPolicy, &pe.IPAllowlist, &pe.ExpiresAt, &pe.WebhookSecret, &pe.RepoURL, &pe.Stopped); err != nil {
			continue
		}
		pe.Engine = firstNonEmptyEngine(pe.Engine)
		pe.TrafficMode = firstNonEmpty(normalizeTrafficMode(pe.TrafficMode), s.lanePolicy(DeployEnv(pe.Env)).DefaultTrafficMode)
		pe.AccessPolicy = firstNonEmpty(normalizeAccessPolicy(pe.AccessPolicy), s.lanePolicy(DeployEnv(pe.Env)).DefaultAccessPolicy)
		if _, ok := byApp[pe.App]; !ok {
			byApp[pe.App] = &ProjectInfo{Name: pe.App, Envs: []ProjectEnv{}, Services: []ProjectService{}}
		}
		byApp[pe.App].Envs = append(byApp[pe.App].Envs, pe)
	}

	// Attach services to each project.
	for _, proj := range byApp {
		svcs, _ := s.db.Query(
			`SELECT project, name, type, branch, env, container, network, volume, env_key, env_val, COALESCE(image,''), COALESCE(port,0), COALESCE(host_port,0), COALESCE(spec_hash,'')
			FROM project_services WHERE project=?`, proj.Name,
		)
		if svcs != nil {
			for svcs.Next() {
				var ps ProjectService
				_ = svcs.Scan(&ps.Project, &ps.Name, &ps.Type, &ps.Branch, &ps.Env,
					&ps.Container, &ps.Network, &ps.Volume, &ps.EnvKey, &ps.EnvVal, &ps.Image, &ps.Port, &ps.HostPort, &ps.SpecHash)
				if ps.Container != "" {
					ps.Running = s.runtime.IsRunning(ps.Container)
				}
				proj.Services = append(proj.Services, ps)
			}
			svcs.Close()
		}
	}

	out := make([]*ProjectInfo, 0, len(byApp))
	for _, v := range byApp {
		out = append(out, v)
	}
	writeJSON(w, 200, out)
}

type projectLaneRecord struct {
	Env           DeployEnv
	Branch        string
	CurrentImage  string
	PreviousImage string
}

type projectDeployRecord struct {
	ID        string
	LogPath   string
	ImageTag  string
	PrevImage string
}

type projectSyncRecord struct {
	ID         string
	RepoDir    string
	StagingDir string
}

func (s *Server) projectHasActiveWork(app string) bool {
	app = strings.TrimSpace(app)
	if app == "" {
		return false
	}

	s.buildLock.Lock()
	for key, active := range s.building {
		if active && strings.HasPrefix(key, app+"__") {
			s.buildLock.Unlock()
			return true
		}
	}
	s.buildLock.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, deploy := range s.deploys {
		if deploy == nil || deploy.App != app {
			continue
		}
		if deploy.Status == StatusQueued || deploy.Status == StatusRunning {
			return true
		}
	}
	return false
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}

	var body ProjectDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	body.App = strings.TrimSpace(body.App)
	if body.App == "" {
		httpError(w, 400, "app required")
		return
	}
	if s.projectHasActiveWork(body.App) {
		httpError(w, 409, "cannot delete a project while deploys are queued or running")
		return
	}

	summary, warnings, err := s.deleteProjectData(body.App)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{
		"status":   "deleted",
		"app":      body.App,
		"summary":  summary,
		"warnings": warnings,
	})
}

func (s *Server) deleteProjectData(app string) (map[string]int64, []string, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return nil, nil, fmt.Errorf("app required")
	}

	lanes := make([]projectLaneRecord, 0, 4)
	laneRows, err := s.db.Query(
		`SELECT env, branch, COALESCE(current_image,''), COALESCE(previous_image,'')
		FROM app_state WHERE app=?`,
		app,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load project lanes: %w", err)
	}
	for laneRows.Next() {
		var rec projectLaneRecord
		var envS string
		if err := laneRows.Scan(&envS, &rec.Branch, &rec.CurrentImage, &rec.PreviousImage); err != nil {
			continue
		}
		rec.Env = DeployEnv(envS)
		lanes = append(lanes, rec)
	}
	laneRows.Close()

	services := make([]ProjectService, 0, 8)
	serviceRows, err := s.db.Query(
		`SELECT project, name, type, branch, env, container, network, volume, env_key, env_val, COALESCE(image,''), COALESCE(port,0), COALESCE(host_port,0), COALESCE(spec_hash,'')
		FROM project_services WHERE project=?`,
		app,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load project services: %w", err)
	}
	for serviceRows.Next() {
		var ps ProjectService
		if err := serviceRows.Scan(&ps.Project, &ps.Name, &ps.Type, &ps.Branch, &ps.Env, &ps.Container, &ps.Network, &ps.Volume, &ps.EnvKey, &ps.EnvVal, &ps.Image, &ps.Port, &ps.HostPort, &ps.SpecHash); err != nil {
			continue
		}
		services = append(services, ps)
	}
	serviceRows.Close()

	deploys := make([]projectDeployRecord, 0, 16)
	deployRows, err := s.db.Query(
		`SELECT id, COALESCE(log_path,''), COALESCE(image_tag,''), COALESCE(previous_image_tag,'')
		FROM deploys WHERE app=?`,
		app,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load project deploys: %w", err)
	}
	for deployRows.Next() {
		var rec projectDeployRecord
		if err := deployRows.Scan(&rec.ID, &rec.LogPath, &rec.ImageTag, &rec.PrevImage); err != nil {
			continue
		}
		deploys = append(deploys, rec)
	}
	deployRows.Close()

	sessions := make([]projectSyncRecord, 0, 4)
	sessionRows, err := s.db.Query(
		`SELECT id, repo_dir, staging_dir FROM sync_sessions WHERE app=?`,
		app,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load project sync sessions: %w", err)
	}
	for sessionRows.Next() {
		var rec projectSyncRecord
		if err := sessionRows.Scan(&rec.ID, &rec.RepoDir, &rec.StagingDir); err != nil {
			continue
		}
		sessions = append(sessions, rec)
	}
	sessionRows.Close()

	if len(lanes) == 0 && len(services) == 0 && len(deploys) == 0 && len(sessions) == 0 {
		return nil, nil, fmt.Errorf("project not found")
	}

	warnings := make([]string, 0, 8)
	images := map[string]struct{}{}
	networks := map[string]struct{}{}
	workspaces := map[string]struct{}{}
	logs := map[string]struct{}{}
	edgeConfigs := map[string]struct{}{}
	volumes := map[string]struct{}{}

	for _, lane := range lanes {
		s.runtime.Remove(appBaseContainerName(app, lane.Env, lane.Branch))
		s.runtime.Remove(appSlotContainerName(app, lane.Env, lane.Branch, "blue"))
		s.runtime.Remove(appSlotContainerName(app, lane.Env, lane.Branch, "green"))
		_ = s.stopStationLane(app, lane.Env, lane.Branch)
		networks[appNetworkName(app, lane.Env, lane.Branch)] = struct{}{}
		workspaces[filepath.Join(s.workspacesDir, fmt.Sprintf("%s__%s__%s", safe(app), safe(string(lane.Env)), safe(lane.Branch)))] = struct{}{}
		edgeConfigs[s.edgeProxyConfigPath(app, lane.Env, lane.Branch)] = struct{}{}
		if lane.CurrentImage != "" {
			images[lane.CurrentImage] = struct{}{}
		}
		if lane.PreviousImage != "" {
			images[lane.PreviousImage] = struct{}{}
		}
	}

	for _, svc := range services {
		s.runtime.Remove(svc.Container)
		if svc.Network != "" {
			networks[svc.Network] = struct{}{}
		}
		if svc.Volume != "" {
			volumes[svc.Volume] = struct{}{}
		}
	}

	for _, dep := range deploys {
		if dep.LogPath != "" {
			logs[dep.LogPath] = struct{}{}
		}
		if dep.ImageTag != "" {
			images[dep.ImageTag] = struct{}{}
		}
		if dep.PrevImage != "" {
			images[dep.PrevImage] = struct{}{}
		}
	}

	for _, sess := range sessions {
		if pathWithinBase(s.workspacesDir, sess.RepoDir) {
			workspaces[filepath.Dir(sess.RepoDir)] = struct{}{}
		}
		if pathWithinBase(s.workspacesDir, sess.StagingDir) {
			workspaces[filepath.Dir(sess.StagingDir)] = struct{}{}
		}
	}

	for path := range logs {
		if !pathWithinBase(s.logsDir, path) {
			warnings = append(warnings, fmt.Sprintf("skipped log cleanup outside logs dir: %s", path))
			continue
		}
		_ = os.Remove(path)
	}

	for path := range workspaces {
		if !pathWithinBase(s.workspacesDir, path) {
			warnings = append(warnings, fmt.Sprintf("skipped workspace cleanup outside workspaces dir: %s", path))
			continue
		}
		_ = os.RemoveAll(path)
	}

	edgeProxyDir := filepath.Join(s.dataDir, "edge-proxy")
	for path := range edgeConfigs {
		if !pathWithinBase(edgeProxyDir, path) {
			warnings = append(warnings, fmt.Sprintf("skipped edge proxy cleanup outside edge-proxy dir: %s", path))
			continue
		}
		_ = os.Remove(path)
	}

	for volume := range volumes {
		s.runtime.RemoveVolume(volume)
		if s.stationRuntime != nil {
			s.stationRuntime.RemoveVolume(volume)
		}
	}
	for network := range networks {
		s.runtime.RemoveNetwork(network)
		if s.stationRuntime != nil {
			s.stationRuntime.RemoveNetwork(network)
		}
	}
	for image := range images {
		s.runtime.RemoveImage(image)
		_ = os.RemoveAll(stationSnapshotDir(image))
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, warnings, fmt.Errorf("begin delete transaction: %w", err)
	}

	execDelete := func(query string, args ...any) (int64, error) {
		res, err := tx.Exec(query, args...)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return n, nil
	}

	deployRequestCount, err := execDelete(`DELETE FROM deploy_requests WHERE app=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete deploy requests: %w", err)
	}
	deployCount, err := execDelete(`DELETE FROM deploys WHERE app=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete deploys: %w", err)
	}
	appStateCount, err := execDelete(`DELETE FROM app_state WHERE app=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete app state: %w", err)
	}
	secretCount, err := execDelete(`DELETE FROM app_secrets WHERE app=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete app secrets: %w", err)
	}
	serviceStateCount, err := execDelete(`DELETE FROM project_services WHERE project=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete service state: %w", err)
	}
	serviceSpecCount, err := execDelete(`DELETE FROM project_service_specs WHERE project=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete service specs: %w", err)
	}
	sessionCount, err := execDelete(`DELETE FROM sync_sessions WHERE app=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete sync sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, warnings, fmt.Errorf("commit delete transaction: %w", err)
	}

	s.mu.Lock()
	for id, deploy := range s.deploys {
		if deploy != nil && deploy.App == app {
			delete(s.deploys, id)
		}
	}
	s.mu.Unlock()

	s.syncMu.Lock()
	for id, sess := range s.syncSessions {
		if sess != nil && sess.App == app {
			delete(s.syncSessions, id)
		}
	}
	s.syncMu.Unlock()

	s.buildLock.Lock()
	for key := range s.building {
		if strings.HasPrefix(key, app+"__") {
			delete(s.building, key)
		}
	}
	s.buildLock.Unlock()

	s.broadcastSnapshot()
	go func() { _ = s.ensureGlobalProxy() }()

	return map[string]int64{
		"lanes":            appStateCount,
		"deploys":          deployCount,
		"deploy_requests":  deployRequestCount,
		"secrets":          secretCount,
		"service_states":   serviceStateCount,
		"service_specs":    serviceSpecCount,
		"sync_sessions":    sessionCount,
		"runtime_services": int64(len(services)),
	}, warnings, nil
}

// ---------------------- Main ----------------------

func main() {
	runArgs := os.Args[1:]
	if len(os.Args) > 1 {
		switch runArgs[0] {
		case "service":
			if err := handleServiceCommand(runArgs[1:]); err != nil {
				dieService(err.Error())
			}
			return
		case "version", "--version", "-version":
			fmt.Println(relaydVersionLine())
			return
		}
	}

	runCfg, err := parseRelaydRunArgs(runArgs)
	if err != nil {
		if strings.Contains(err.Error(), "Usage:\n\trelayd") {
			fmt.Println(err.Error())
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, relaydRunUsage)
		os.Exit(2)
	}
	if runCfg.ShowVersion {
		fmt.Println(relaydVersionLine())
		return
	}

	dataDir := getenv("RELAY_DATA_DIR", "./data")
	if runCfg.DataDir != "" {
		dataDir = runCfg.DataDir
	}
	apiToken := getenv("RELAY_TOKEN", "")

	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		panic(fmt.Errorf("failed to abs data dir: %w", err))
	}
	dataDir = absDataDir

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		panic(fmt.Errorf("failed to create data dir: %w", err))
	}

	if apiToken == "" {
		var created bool
		apiToken, created = loadOrCreateToken(dataDir)
		if created {
			fmt.Println("Relay token (save this):", apiToken)
		}
	}

	dbPath := filepath.Join(dataDir, "relay.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA busy_timeout=5000;")
	db.SetMaxOpenConns(1)
	if err := migrateDB(db); err != nil {
		panic(err)
	}

	// Derive a 32-byte AES-256 key from RELAY_SECRET_KEY for encrypting secrets at rest.
	var secretKey []byte
	if rawKey := strings.TrimSpace(os.Getenv("RELAY_SECRET_KEY")); rawKey != "" {
		h := sha256.Sum256([]byte(rawKey))
		secretKey = h[:]
	}

	s := &Server{
		deploys:               make(map[string]*Deploy),
		queue:                 make(chan DeployJob, 200),
		syncSessions:          make(map[string]*SyncSession),
		dataDir:               dataDir,
		workspacesDir:         filepath.Join(dataDir, "workspaces"),
		logsDir:               filepath.Join(dataDir, "logs"),
		pluginsDir:            filepath.Join(dataDir, "plugins"),
		acmeWebroot:           filepath.Join(dataDir, "acme-webroot"),
		caddyLogsDir:          filepath.Join(dataDir, "caddy-logs"),
		httpAddr:              getenv("RELAY_ADDR", ":8080"),
		corsOrigins:           map[string]struct{}{},
		allowAllCORS:          false,
		enablePluginMutations: getenvBool("RELAY_ENABLE_PLUGIN_MUTATIONS", false),
		db:                    db,
		apiToken:              apiToken,
		secretKey:             secretKey,
		buildpacks:            defaultBuildpacks(),
		building:              make(map[string]bool),
		buildCancels:          make(map[string]context.CancelFunc),
		webhookHits:           make(map[string][]time.Time),
		eventsChans:           make(map[chan []byte]struct{}),
		runtime:               &DockerRuntime{},
		stationRuntime:        newStationRuntime(dataDir),
	}
	s.corsOrigins, s.allowAllCORS = parseAllowedOrigins(os.Getenv("RELAY_CORS_ORIGINS"))

	mustMkdir(s.workspacesDir)
	mustMkdir(s.logsDir)
	mustMkdir(s.pluginsDir)
	mustMkdir(s.acmeWebroot)
	mustMkdir(s.caddyLogsDir)
	mustMkdir(s.pluginBuildpacksDir())
	if err := s.reloadBuildpacks(); err != nil {
		panic(err)
	}

	_ = s.loadSessionsFromDB()
	_ = s.loadDeploysFromDB()
	_ = s.reconcileStaleDeploysOnStartup()

	// Restore global domain proxy state from DB
	go func() { _ = s.ensureGlobalProxy() }()
	go s.runLaneExpiryWorker()

	// Start worker pool: use half of available CPUs (at least 1)
	n := runtime.NumCPU() / 2
	if n < 1 {
		n = 1
	}
	s.worker(n)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.Handle("/.well-known/acme-challenge/",
		http.StripPrefix("/.well-known/acme-challenge/",
			http.FileServer(http.Dir(s.acmeWebroot))))

	// UI
	uiRoot, _ := fs.Sub(uiFS, "ui")
	uiHandler := http.FileServer(http.FS(uiRoot))
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", uiHandler))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dashboard/", http.StatusFound)
			return
		}
		// Fallback for SPA or other files if they are in root
		uiHandler.ServeHTTP(w, r)
	})

	// API (auth)
	authAny := s.auth
	authOwner := s.authWithRoles("owner")
	authDeployer := s.authWithRoles("owner", "deployer")
	authReadOnly := s.authByMethod(nil, nil)
	authReadDeployerWrite := s.authByMethod(nil, []string{"owner", "deployer"})
	mux.HandleFunc("/api/auth/session", s.handleDashboardSession)
	mux.HandleFunc("/api/auth/setup", s.handleAuthSetup)
	mux.HandleFunc("/api/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/api/auth/me", s.handleAuthMe)
	mux.HandleFunc("/api/auth/cli/start", s.handleAuthCLIStart)
	mux.HandleFunc("/api/auth/cli/exchange", s.handleAuthCLIExchange)
	mux.HandleFunc("/api/deploys", authReadDeployerWrite(s.handleDeploys))
	mux.HandleFunc("/api/deploys/cancel/", authDeployer(s.handleDeployCancel))
	mux.HandleFunc("/api/deploys/", authAny(s.handleDeployByID))
	mux.HandleFunc("/api/deploys/rollback", authDeployer(s.handleRollback))
	mux.HandleFunc("/api/apps/start", authDeployer(s.handleAppStart))
	mux.HandleFunc("/api/apps/stop", authDeployer(s.handleAppStop))
	mux.HandleFunc("/api/apps/restart", authDeployer(s.handleAppRestart))
	mux.HandleFunc("/api/apps/config", authReadDeployerWrite(s.handleAppConfig))
	mux.HandleFunc("/api/apps/signed-link", authDeployer(s.handleSignedLink))
	mux.HandleFunc("/api/server/config", authOwner(s.handleServerConfig))
	mux.HandleFunc("/api/apps/companions", authReadDeployerWrite(s.handleAppCompanions))
	mux.HandleFunc("/api/apps/companions/restart", authDeployer(s.handleCompanionRestart))
	mux.HandleFunc("/api/apps/secrets", authDeployer(s.handleAppSecrets))
	mux.HandleFunc("/api/plugins/buildpacks", authOwner(s.handleBuildpackPlugins))
	mux.HandleFunc("/api/plugins/buildpacks/", authOwner(s.handleBuildpackPluginByName))

	mux.HandleFunc("/api/logs/", authAny(s.handleLogsByID))
	mux.HandleFunc("/api/logs/stream/", authAny(s.handleLogsStream))
	mux.HandleFunc("/api/runtime/logs/targets", authAny(s.handleRuntimeLogTargets))
	mux.HandleFunc("/api/runtime/logs/stream", authAny(s.handleRuntimeLogStream))
	mux.HandleFunc("/api/events", authAny(s.handleEvents))

	// Sync
	mux.HandleFunc("/api/sync/start", authDeployer(s.handleSyncStart))
	mux.HandleFunc("/api/sync/plan/", authDeployer(s.handleSyncPlan))
	mux.HandleFunc("/api/sync/upload/", authDeployer(s.handleSyncUpload))
	mux.HandleFunc("/api/sync/bundle/", authDeployer(s.handleSyncBundle))
	mux.HandleFunc("/api/sync/delete/", authDeployer(s.handleSyncDelete))
	mux.HandleFunc("/api/sync/finish/", authDeployer(s.handleSyncFinish))
	mux.HandleFunc("/api/sync/pull/", authDeployer(s.handleSyncPull))

	// Projects (multi-service view)
	mux.HandleFunc("/api/projects", authReadOnly(s.handleProjects))
	mux.HandleFunc("/api/projects/delete", authOwner(s.handleProjectDelete))
	mux.HandleFunc("/api/promotions", authReadDeployerWrite(s.handlePromotions))
	mux.HandleFunc("/api/promotions/approve", authOwner(s.handlePromotionApprove))

	// Webhooks (unauthenticated, use provider-specific secrets if configured)
	mux.HandleFunc("/api/webhooks/github", s.handleGithubWebhook)
	mux.HandleFunc("/api/edge/authz", s.handleEdgeAuthz)
	mux.HandleFunc("/api/analytics", authAny(s.handleAnalytics))

	// User management (owner only, enforced inside handlers)
	mux.HandleFunc("/api/users", authOwner(s.handleUsers))
	mux.HandleFunc("/api/users/", authOwner(s.handleUserByID))

	// Audit log
	mux.HandleFunc("/api/audit", authOwner(s.handleAuditLog))

	// ── Unix-domain socket (local API transport) ──────────────────────────────
	// The socket is the recommended transport for the relay CLI when both the
	// CLI and server run on the same machine.  No HTTP overhead, no token needed
	// (filesystem ACL provides authentication), and traffic never hits the
	// network stack.  Set RELAY_SOCKET="" to disable.
	socketPath := getenv("RELAY_SOCKET", filepath.Join(dataDir, "relay.sock"))
	if runCfg.DisableSocket {
		socketPath = ""
	} else if runCfg.SocketPath != "" {
		socketPath = runCfg.SocketPath
	}
	if socketPath != "" {
		// Remove stale socket from a previous run.
		_ = os.Remove(socketPath)
		ln, sockErr := net.Listen("unix", socketPath)
		if sockErr != nil {
			fmt.Fprintf(os.Stderr, "warn: unix socket unavailable (%v); API reachable via HTTP only\n", sockErr)
		} else {
			if err := os.Chmod(socketPath, 0600); err != nil {
				fmt.Fprintf(os.Stderr, "warn: could not chmod socket: %v\n", err)
			}
			// The socket mux mirrors the full API mux (all /api/ routes).
			// We do NOT serve the UI files on the socket; that stays on TCP.
			sockMux := http.NewServeMux()
			sockMux.HandleFunc("/health", s.handleHealth)
			sockMux.HandleFunc("/api/version", s.handleVersion)
			sockMux.HandleFunc("/api/auth/session", s.handleDashboardSession)
			sockMux.HandleFunc("/api/auth/setup", s.handleAuthSetup)
			sockMux.HandleFunc("/api/auth/login", s.handleAuthLogin)
			sockMux.HandleFunc("/api/auth/me", s.handleAuthMe)
			sockMux.HandleFunc("/api/auth/cli/start", s.handleAuthCLIStart)
			sockMux.HandleFunc("/api/auth/cli/exchange", s.handleAuthCLIExchange)
			sockMux.HandleFunc("/api/deploys", authReadDeployerWrite(s.handleDeploys))
			sockMux.HandleFunc("/api/deploys/cancel/", authDeployer(s.handleDeployCancel))
			sockMux.HandleFunc("/api/deploys/", authAny(s.handleDeployByID))
			sockMux.HandleFunc("/api/deploys/rollback", authDeployer(s.handleRollback))
			sockMux.HandleFunc("/api/apps/start", authDeployer(s.handleAppStart))
			sockMux.HandleFunc("/api/apps/stop", authDeployer(s.handleAppStop))
			sockMux.HandleFunc("/api/apps/restart", authDeployer(s.handleAppRestart))
			sockMux.HandleFunc("/api/apps/config", authReadDeployerWrite(s.handleAppConfig))
			sockMux.HandleFunc("/api/apps/signed-link", authDeployer(s.handleSignedLink))
			sockMux.HandleFunc("/api/server/config", authOwner(s.handleServerConfig))
			sockMux.HandleFunc("/api/apps/companions", authReadDeployerWrite(s.handleAppCompanions))
			sockMux.HandleFunc("/api/apps/companions/restart", authDeployer(s.handleCompanionRestart))
			sockMux.HandleFunc("/api/apps/secrets", authDeployer(s.handleAppSecrets))
			sockMux.HandleFunc("/api/plugins/buildpacks", authOwner(s.handleBuildpackPlugins))
			sockMux.HandleFunc("/api/plugins/buildpacks/", authOwner(s.handleBuildpackPluginByName))
			sockMux.HandleFunc("/api/logs/", authAny(s.handleLogsByID))
			sockMux.HandleFunc("/api/logs/stream/", authAny(s.handleLogsStream))
			sockMux.HandleFunc("/api/runtime/logs/targets", authAny(s.handleRuntimeLogTargets))
			sockMux.HandleFunc("/api/runtime/logs/stream", authAny(s.handleRuntimeLogStream))
			sockMux.HandleFunc("/api/events", authAny(s.handleEvents))
			sockMux.HandleFunc("/api/sync/start", authDeployer(s.handleSyncStart))
			sockMux.HandleFunc("/api/sync/plan/", authDeployer(s.handleSyncPlan))
			sockMux.HandleFunc("/api/sync/upload/", authDeployer(s.handleSyncUpload))
			sockMux.HandleFunc("/api/sync/bundle/", authDeployer(s.handleSyncBundle))
			sockMux.HandleFunc("/api/sync/delete/", authDeployer(s.handleSyncDelete))
			sockMux.HandleFunc("/api/sync/finish/", authDeployer(s.handleSyncFinish))
			sockMux.HandleFunc("/api/sync/pull/", authDeployer(s.handleSyncPull))
			sockMux.HandleFunc("/api/projects", authReadOnly(s.handleProjects))
			sockMux.HandleFunc("/api/projects/delete", authOwner(s.handleProjectDelete))
			sockMux.HandleFunc("/api/promotions", authReadDeployerWrite(s.handlePromotions))
			sockMux.HandleFunc("/api/promotions/approve", authOwner(s.handlePromotionApprove))
			sockMux.HandleFunc("/api/webhooks/github", s.handleGithubWebhook)
			sockMux.HandleFunc("/api/edge/authz", s.handleEdgeAuthz)
			sockMux.HandleFunc("/api/users", authOwner(s.handleUsers))
			sockMux.HandleFunc("/api/users/", authOwner(s.handleUserByID))
			sockMux.HandleFunc("/api/audit", authOwner(s.handleAuditLog))
			socketServer := &http.Server{
				Handler:           sockMux,
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       60 * time.Second,
				IdleTimeout:       300 * time.Second,
				// ConnContext stamps every connection with ctxKeySocket=true so
				// the auth middleware can skip token validation.
				ConnContext: func(ctx context.Context, c net.Conn) context.Context {
					return context.WithValue(ctx, ctxKeySocket, true)
				},
			}
			fmt.Println("Relay API socket :", socketPath)
			go func() {
				if err := socketServer.Serve(ln); err != nil && err != http.ErrServerClosed {
					fmt.Fprintf(os.Stderr, "socket server error: %v\n", err)
				}
			}()
		}
	}

	// ── TCP HTTP server (UI + full API for remote / webhook access) ───────────
	addr := getenv("RELAY_ADDR", ":8080")
	if runCfg.Addr != "" {
		addr = runCfg.Addr
	}
	s.httpAddr = addr
	// Start the lightweight ACME / HTTP-01 listener on :80 (best-effort; if
	// the port is already taken by Caddy via Docker, it logs a notice and
	// continues without it).
	go s.startACMEListener()
	// Tail Caddy access logs and aggregate per-request analytics.
	go s.startLogTailer()
	httpServer := &http.Server{
		Addr:    addr,
		Handler: s.withCORS(mux),
		// ReadHeaderTimeout guards against Slowloris-style attacks.
		ReadHeaderTimeout: 10 * time.Second,
		// ReadTimeout applies to reading the entire request body.
		// Large file uploads via /api/sync/upload set no per-request deadline
		// override, so keep this generous.
		ReadTimeout: 120 * time.Second,
		// Do NOT set WriteTimeout – it would kill long-lived SSE log streams.
		// Individual handlers use http.ResponseController to extend deadlines.
		IdleTimeout: 120 * time.Second,
	}
	fmt.Println("Relay Agent listening on", addr)
	if err := httpServer.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "address already in use") {
			fmt.Fprintln(os.Stderr, "error: cannot start relayd because the listen address is already in use:", addr)
			fmt.Fprintln(os.Stderr, "fix:")
			fmt.Fprintln(os.Stderr, "  1) stop the existing process using this port")
			fmt.Fprintln(os.Stderr, "  2) or start relayd on another port, for example:")
			fmt.Fprintln(os.Stderr, "     relayd --port 9090")
			fmt.Fprintln(os.Stderr, "     RELAY_ADDR=:9090 relayd")
			os.Exit(1)
		}
		panic(err)
	}
}

type relaydServiceConfig struct {
	Name    string
	User    string
	Group   string
	DataDir string
	Addr    string
	BinPath string
}

func defaultServiceConfig() relaydServiceConfig {
	binPath, _ := os.Executable()
	if binPath == "" {
		binPath = "relayd"
	}
	return relaydServiceConfig{
		Name:    "relayd",
		User:    "root",
		Group:   "root",
		DataDir: "/var/lib/relayd",
		Addr:    ":8080",
		BinPath: binPath,
	}
}

func handleServiceCommand(args []string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("service command is only supported on Linux")
	}
	if len(args) == 0 {
		fmt.Print(relaydServiceUsage)
		return nil
	}

	mode := strings.ToLower(strings.TrimSpace(args[0]))
	cfg, err := parseServiceFlags(args[1:])
	if err != nil {
		return err
	}

	switch mode {
	case "unit":
		fmt.Print(renderSystemdUnit(cfg))
		return nil
	case "install":
		return installSystemdUnit(cfg)
	default:
		return fmt.Errorf("unknown service subcommand %q\n\n%s", mode, relaydServiceUsage)
	}
}

func parseServiceFlags(args []string) (relaydServiceConfig, error) {
	cfg := defaultServiceConfig()
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		if a == "" {
			continue
		}
		switch {
		case a == "--name" && i+1 < len(args):
			cfg.Name = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--name="):
			cfg.Name = strings.TrimSpace(strings.TrimPrefix(a, "--name="))
		case a == "--user" && i+1 < len(args):
			cfg.User = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--user="):
			cfg.User = strings.TrimSpace(strings.TrimPrefix(a, "--user="))
		case a == "--group" && i+1 < len(args):
			cfg.Group = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--group="):
			cfg.Group = strings.TrimSpace(strings.TrimPrefix(a, "--group="))
		case a == "--data-dir" && i+1 < len(args):
			cfg.DataDir = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--data-dir="):
			cfg.DataDir = strings.TrimSpace(strings.TrimPrefix(a, "--data-dir="))
		case a == "--addr" && i+1 < len(args):
			cfg.Addr = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--addr="):
			cfg.Addr = strings.TrimSpace(strings.TrimPrefix(a, "--addr="))
		case a == "--bin" && i+1 < len(args):
			cfg.BinPath = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--bin="):
			cfg.BinPath = strings.TrimSpace(strings.TrimPrefix(a, "--bin="))
		default:
			return cfg, fmt.Errorf("unknown option %q", a)
		}
	}

	if cfg.Name == "" || cfg.User == "" || cfg.Group == "" || cfg.DataDir == "" || cfg.BinPath == "" {
		return cfg, fmt.Errorf("name, user, group, data-dir, and bin must be non-empty")
	}
	return cfg, nil
}

func renderSystemdUnit(cfg relaydServiceConfig) string {
	return fmt.Sprintf(`[Unit]
Description=Relay Agent
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s
Environment=RELAY_DATA_DIR=%s
Environment=RELAY_ADDR=%s
ExecStart=%s
Restart=always
RestartSec=2
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`, cfg.User, cfg.Group, cfg.DataDir, cfg.DataDir, cfg.Addr, cfg.BinPath)
}

func installSystemdUnit(cfg relaydServiceConfig) error {
	unitPath := filepath.Join("/etc/systemd/system", cfg.Name+".service")
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(unitPath, []byte(renderSystemdUnit(cfg)), 0644); err != nil {
		return fmt.Errorf("write unit file %s: %w (run with sudo)", unitPath, err)
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %v\n%s", err, string(out))
	}
	if out, err := exec.Command("systemctl", "enable", "--now", cfg.Name).CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable --now %s failed: %v\n%s", cfg.Name, err, string(out))
	}
	fmt.Printf("installed and started systemd service: %s\n", cfg.Name)
	fmt.Printf("status: systemctl status %s\n", cfg.Name)
	fmt.Printf("logs  : journalctl -u %s -f\n", cfg.Name)
	return nil
}

func dieService(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}

// ---------------------- HTTP Handlers ----------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}

	stationVersion := ""
	stationPath := ""
	stationErr := ""
	if ver, path, err := stationBinaryVersion(); err == nil {
		stationVersion = ver
		stationPath = path
	} else {
		stationErr = err.Error()
	}

	writeJSON(w, 200, map[string]any{
		"component":           "relayd",
		"version":             relaydVersion,
		"commit":              relaydCommit,
		"build_date":          relaydBuildDate,
		"goos":                runtime.GOOS,
		"goarch":              runtime.GOARCH,
		"station_version":     stationVersion,
		"station_binary":      stationPath,
		"station_version_err": stationErr,
	})
}

func (s *Server) handleDeploys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req DeployRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		if err := validateDeployRequest(req); err != nil {
			httpError(w, 400, err.Error())
			return
		}

		deployID := newID()
		logPath := filepath.Join(s.logsDir, deployID+".log")
		deploy := &Deploy{
			ID:        deployID,
			App:       req.App,
			RepoURL:   req.RepoURL,
			Branch:    req.Branch,
			CommitSHA: req.CommitSHA,
			Env:       req.Env,
			Status:    StatusQueued,
			CreatedAt: time.Now(),
			LogPath:   logPath,
		}

		s.mu.Lock()
		s.deploys[deployID] = deploy
		s.mu.Unlock()

		if err := s.saveDeployToDB(deploy, req); err != nil {
			httpError(w, 500, "failed to persist deploy: "+err.Error())
			return
		}

		s.queue <- DeployJob{ID: deployID, Req: req}
		s.broadcastSnapshot()
		writeJSON(w, 201, deploy)

	case http.MethodGet:
		out, err := s.listDeploysFromDB()
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, out)

	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) handleDeployByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/deploys/")
	if id == "" {
		httpError(w, 400, "missing id")
		return
	}

	d, err := s.getDeployFromDB(id)
	if err != nil {
		httpError(w, 404, "not found")
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) handleLogsByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	if id == "" {
		httpError(w, 400, "missing id")
		return
	}

	d, err := s.getDeployFromDB(id)
	if err != nil {
		s.mu.RLock()
		d = s.deploys[id]
		s.mu.RUnlock()
		if d == nil {
			httpError(w, 404, "not found")
			return
		}
	}

	f, err := os.Open(d.LogPath)
	if err != nil {
		httpError(w, 404, "log not found")
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.Copy(w, f)
}

type RuntimeLogTarget struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Kind      string `json:"kind"`
	Container string `json:"container"`
	Running   bool   `json:"running"`
	Live      bool   `json:"live,omitempty"`
	Slot      string `json:"slot,omitempty"`
	Service   string `json:"service,omitempty"`
	Image     string `json:"image,omitempty"`
	Engine    string `json:"engine,omitempty"`
}

type RuntimeLogLaneState struct {
	AppStopped       bool   `json:"app_stopped"`
	AppRunning       bool   `json:"app_running"`
	HasRunningTarget bool   `json:"has_running_target"`
	ActiveSlot       string `json:"active_slot,omitempty"`
	StandbySlot      string `json:"standby_slot,omitempty"`
	OfflineReason    string `json:"offline_reason,omitempty"`
}

func runtimeLogDefaultTarget(targets []RuntimeLogTarget) string {
	for _, target := range targets {
		if target.Kind == "app" && target.Live && target.Running {
			return target.ID
		}
	}
	for _, target := range targets {
		if target.Kind == "app" && target.Running {
			return target.ID
		}
	}
	for _, target := range targets {
		if target.Running {
			return target.ID
		}
	}
	return ""
}

func runtimeLogOfflineReason(lane RuntimeLogLaneState, target *RuntimeLogTarget) string {
	if target != nil && target.Label != "" && lane.HasRunningTarget {
		return fmt.Sprintf("%s is offline right now. Choose a running target to inspect logs.", target.Label)
	}
	if lane.OfflineReason != "" {
		return lane.OfflineReason
	}
	if target != nil && target.Label != "" {
		return fmt.Sprintf("%s is offline right now.", target.Label)
	}
	return "No running runtime log targets are available for this lane."
}

func runtimeLogDockerError(line string, lane RuntimeLogLaneState, target *RuntimeLogTarget) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "no such container") || strings.Contains(lower, "container") && strings.Contains(lower, "not found") {
		return runtimeLogOfflineReason(lane, target)
	}
	if strings.Contains(lower, "is not running") {
		return runtimeLogOfflineReason(lane, target)
	}
	return ""
}

func (s *Server) runtimeLogTargets(app string, env DeployEnv, branch string) ([]RuntimeLogTarget, RuntimeLogLaneState, error) {
	st, err := s.getAppState(app, env, branch)
	if err != nil {
		return nil, RuntimeLogLaneState{}, err
	}
	if firstNonEmptyEngine(st.Engine) == EngineStation {
		return s.stationRuntimeLogTargets(app, env, branch, st)
	}

	activeSlot := normalizeActiveSlot(st.ActiveSlot)
	if activeSlot == "" {
		activeSlot = s.currentActiveSlot(app, env, branch, st)
	}
	standbySlot := normalizeActiveSlot(st.StandbySlot)

	targets := make([]RuntimeLogTarget, 0, 6)
	seen := map[string]struct{}{}
	add := func(target RuntimeLogTarget) {
		if target.ID == "" || target.Container == "" {
			return
		}
		if _, ok := seen[target.ID]; ok {
			return
		}
		target.Running = s.runtime.IsRunning(target.Container)
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
			Image:     st.CurrentImage,
		})
	}
	if standbySlot != "" && standbySlot != activeSlot {
		add(RuntimeLogTarget{
			ID:        "standby",
			Label:     fmt.Sprintf("Standby app (%s)", standbySlot),
			Kind:      "app",
			Container: appSlotContainerName(app, env, branch, standbySlot),
			Slot:      standbySlot,
			Image:     st.PreviousImage,
		})
	}
	for _, slot := range []string{"blue", "green"} {
		if slot == activeSlot || slot == standbySlot {
			continue
		}
		name := appSlotContainerName(app, env, branch, slot)
		if s.runtime.IsRunning(name) {
			add(RuntimeLogTarget{
				ID:        "slot:" + slot,
				Label:     fmt.Sprintf("App slot (%s)", slot),
				Kind:      "app",
				Container: name,
				Slot:      slot,
				Image:     st.CurrentImage,
			})
		}
	}

	add(RuntimeLogTarget{
		ID:        "proxy",
		Label:     "Edge proxy",
		Kind:      "proxy",
		Container: appBaseContainerName(app, env, branch),
		Image:     getenv("RELAY_NGINX_IMAGE", "nginx:alpine"),
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

	lane := RuntimeLogLaneState{
		AppStopped:  st.Stopped,
		ActiveSlot:  activeSlot,
		StandbySlot: standbySlot,
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
		lane.OfflineReason = fmt.Sprintf("Relay cannot find a running container for the live %s app slot.", activeSlot)
	case !lane.AppRunning:
		lane.OfflineReason = "Relay cannot find a running app container for this lane."
	case !lane.HasRunningTarget:
		lane.OfflineReason = "No runtime containers are currently running for this lane."
	}

	return targets, lane, nil
}

func (s *Server) resolveRuntimeLogTarget(app string, env DeployEnv, branch string, id string) (*RuntimeLogTarget, []RuntimeLogTarget, RuntimeLogLaneState, error) {
	targets, lane, err := s.runtimeLogTargets(app, env, branch)
	if err != nil {
		return nil, nil, RuntimeLogLaneState{}, err
	}
	if len(targets) == 0 {
		return nil, nil, lane, fmt.Errorf("no runtime log targets available")
	}
	if strings.TrimSpace(id) == "" {
		id = runtimeLogDefaultTarget(targets)
		if id == "" {
			return nil, targets, lane, fmt.Errorf(runtimeLogOfflineReason(lane, nil))
		}
	}
	for _, target := range targets {
		if target.ID == id {
			t := target
			if !t.Running {
				return &t, targets, lane, fmt.Errorf(runtimeLogOfflineReason(lane, &t))
			}
			return &t, targets, lane, nil
		}
	}
	return nil, targets, lane, fmt.Errorf("runtime log target %q not found", id)
}

func writeSSEFrame(w io.Writer, flusher http.Flusher, eventName string, payload string) {
	if eventName != "" {
		_, _ = fmt.Fprintf(w, "event: %s\n", eventName)
	}
	for _, line := range strings.Split(payload, "\n") {
		_, _ = fmt.Fprintf(w, "data: %s\n", line)
	}
	_, _ = fmt.Fprint(w, "\n")
	flusher.Flush()
}

func (s *Server) handleRuntimeLogTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}

	app := strings.TrimSpace(r.URL.Query().Get("app"))
	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	env := DeployEnv(strings.TrimSpace(r.URL.Query().Get("env")))
	if app == "" || branch == "" || env == "" {
		httpError(w, 400, "app, env, and branch are required")
		return
	}

	targets, lane, err := s.runtimeLogTargets(app, env, branch)
	if err != nil {
		httpError(w, 404, err.Error())
		return
	}

	writeJSON(w, 200, map[string]any{
		"targets":        targets,
		"default_target": runtimeLogDefaultTarget(targets),
		"lane":           lane,
	})
}

func (s *Server) handleRuntimeLogStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}

	app := strings.TrimSpace(r.URL.Query().Get("app"))
	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	env := DeployEnv(strings.TrimSpace(r.URL.Query().Get("env")))
	targetID := strings.TrimSpace(r.URL.Query().Get("target"))
	if app == "" || branch == "" || env == "" {
		httpError(w, 400, "app, env, and branch are required")
		return
	}

	selected, _, lane, err := s.resolveRuntimeLogTarget(app, env, branch, targetID)
	if err != nil {
		lowerErr := strings.ToLower(err.Error())
		status := 404
		if strings.Contains(lowerErr, "offline") || strings.Contains(lowerErr, "currently off") || strings.Contains(lowerErr, "no running") || strings.Contains(lowerErr, "cannot find a running") {
			status = 409
		}
		httpError(w, status, err.Error())
		return
	}
	runtime := s.runtimeForEngine(selected.Engine)
	if !runtime.IsRunning(selected.Container) {
		httpError(w, 409, runtimeLogOfflineReason(lane, selected))
		return
	}

	tail := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("tail")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			if parsed < 0 {
				tail = 0
			} else if parsed > 2000 {
				tail = 2000
			} else {
				tail = parsed
			}
		}
	}

	since := strings.TrimSpace(r.URL.Query().Get("since"))
	ctx := r.Context()
	pr, logErr := runtime.LogStream(ctx, selected.Container, tail, since)
	if logErr != nil {
		httpError(w, 500, "cannot start log stream: "+logErr.Error())
		return
	}
	defer pr.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpError(w, 500, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	targetJSON, _ := json.Marshal(selected)
	writeSSEFrame(w, flusher, "runtime-target", string(targetJSON))

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	lines := make(chan string, 64)
	scanErrs := make(chan error, 1)
	streamErr := ""
	go func() {
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
		scanErrs <- scanner.Err()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case line, ok := <-lines:
			if ok {
				if streamErr == "" {
					streamErr = runtimeLogDockerError(line, lane, selected)
				}
				if streamErr != "" {
					continue
				}
				writeSSEFrame(w, flusher, "", line)
				continue
			}
			scanErr := <-scanErrs
			status := map[string]any{"status": "complete", "target": selected.ID}
			if streamErr != "" {
				status["status"] = "error"
				status["error"] = streamErr
			} else if scanErr != nil {
				status["status"] = "error"
				status["error"] = scanErr.Error()
			}
			payload, _ := json.Marshal(status)
			writeSSEFrame(w, flusher, "runtime-status", string(payload))
			return
		}
	}
}

// ── SSE event broadcaster ─────────────────────────────────────────────────────

func (s *Server) subscribeEvents() chan []byte {
	ch := make(chan []byte, 32)
	s.eventsMu.Lock()
	s.eventsChans[ch] = struct{}{}
	s.eventsMu.Unlock()
	return ch
}

func (s *Server) unsubscribeEvents(ch chan []byte) {
	s.eventsMu.Lock()
	delete(s.eventsChans, ch)
	s.eventsMu.Unlock()
}

func (s *Server) broadcastEvent(eventType string, payload []byte) {
	msg := make([]byte, 0, len(eventType)+len(payload)+16)
	msg = append(msg, "event: "...)
	msg = append(msg, eventType...)
	msg = append(msg, "\ndata: "...)
	msg = append(msg, payload...)
	msg = append(msg, '\n', '\n')
	s.eventsMu.RLock()
	for ch := range s.eventsChans {
		select {
		case ch <- msg:
		default: // slow client — drop frame, do not block
		}
	}
	s.eventsMu.RUnlock()
}

// broadcastSnapshot builds the current projects+deploys snapshot in a goroutine
// and fans it out as an "update" SSE event to all connected dashboard clients.
func (s *Server) broadcastSnapshot() {
	go func() {
		snap, err := s.buildSnapshotJSON()
		if err != nil {
			return
		}
		s.broadcastEvent("update", snap)
	}()
}

// buildSnapshotJSON returns the projects+deploys payload for the /api/events endpoint.
func (s *Server) buildSnapshotJSON() ([]byte, error) {
	rows, err := s.db.Query(
		`SELECT app, env, branch, COALESCE(engine,''), mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(active_slot,''), COALESCE(standby_slot,''), COALESCE(drain_until,0), COALESCE(traffic_mode,''), COALESCE(access_policy,''), COALESCE(ip_allowlist,''), COALESCE(expires_at,0), COALESCE(webhook_secret,''), repo_url, COALESCE(stopped,0)
		FROM app_state ORDER BY app, env, branch`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type snapEnv struct {
		App              string `json:"app"`
		Env              string `json:"env"`
		Branch           string `json:"branch"`
		Engine           string `json:"engine,omitempty"`
		Mode             string `json:"mode"`
		HostPort         int    `json:"host_port"`
		HostPortExplicit bool   `json:"host_port_explicit,omitempty"`
		ServicePort      int    `json:"service_port"`
		PublicHost       string `json:"public_host"`
		ActiveSlot       string `json:"active_slot,omitempty"`
		StandbySlot      string `json:"standby_slot,omitempty"`
		DrainUntil       int64  `json:"drain_until,omitempty"`
		TrafficMode      string `json:"traffic_mode,omitempty"`
		AccessPolicy     string `json:"access_policy,omitempty"`
		IPAllowlist      string `json:"ip_allowlist,omitempty"`
		ExpiresAt        int64  `json:"expires_at,omitempty"`
		WebhookSecret    string `json:"webhook_secret,omitempty"`
		RepoURL          string `json:"repo_url"`
		Stopped          bool   `json:"stopped,omitempty"`
		Running          bool   `json:"running,omitempty"`
	}
	type snapProject struct {
		Name     string           `json:"name"`
		Envs     []snapEnv        `json:"envs"`
		Services []ProjectService `json:"services"`
	}

	byApp := map[string]*snapProject{}
	for rows.Next() {
		var pe snapEnv
		if err := rows.Scan(&pe.App, &pe.Env, &pe.Branch, &pe.Engine, &pe.Mode,
			&pe.HostPort, &pe.HostPortExplicit, &pe.ServicePort, &pe.PublicHost, &pe.ActiveSlot, &pe.StandbySlot, &pe.DrainUntil, &pe.TrafficMode, &pe.AccessPolicy, &pe.IPAllowlist, &pe.ExpiresAt, &pe.WebhookSecret, &pe.RepoURL, &pe.Stopped); err != nil {
			continue
		}
		pe.Engine = firstNonEmptyEngine(pe.Engine)
		pe.TrafficMode = firstNonEmpty(normalizeTrafficMode(pe.TrafficMode), s.lanePolicy(DeployEnv(pe.Env)).DefaultTrafficMode)
		pe.AccessPolicy = firstNonEmpty(normalizeAccessPolicy(pe.AccessPolicy), s.lanePolicy(DeployEnv(pe.Env)).DefaultAccessPolicy)
		pe.Running = s.appLaneRunning(pe.App, DeployEnv(pe.Env), pe.Branch)
		if _, ok := byApp[pe.App]; !ok {
			byApp[pe.App] = &snapProject{Name: pe.App, Envs: []snapEnv{}, Services: []ProjectService{}}
		}
		byApp[pe.App].Envs = append(byApp[pe.App].Envs, pe)
	}
	rows.Close()

	for _, proj := range byApp {
		svcs, _ := s.db.Query(
			`SELECT project, name, type, branch, env, container, network, volume, env_key, env_val, COALESCE(image,''), COALESCE(port,0), COALESCE(host_port,0), COALESCE(spec_hash,'')
			FROM project_services WHERE project=?`, proj.Name,
		)
		if svcs != nil {
			for svcs.Next() {
				var ps ProjectService
				_ = svcs.Scan(&ps.Project, &ps.Name, &ps.Type, &ps.Branch, &ps.Env,
					&ps.Container, &ps.Network, &ps.Volume, &ps.EnvKey, &ps.EnvVal, &ps.Image, &ps.Port, &ps.HostPort, &ps.SpecHash)
				proj.Services = append(proj.Services, ps)
			}
			svcs.Close()
		}
	}

	projects := make([]*snapProject, 0, len(byApp))
	for _, v := range byApp {
		projects = append(projects, v)
	}

	deploys, err := s.listDeploysFromDB()
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		Projects interface{} `json:"projects"`
		Deploys  interface{} `json:"deploys"`
	}{Projects: projects, Deploys: deploys})
}

// handleEvents streams project/deploy state updates to the dashboard via SSE.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpError(w, 500, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if snap, err := s.buildSnapshotJSON(); err == nil {
		fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", snap)
		flusher.Flush()
	}

	ch := s.subscribeEvents()
	defer s.unsubscribeEvents(ch)

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}

func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/logs/stream/")
	if id == "" {
		httpError(w, 400, "missing id")
		return
	}

	s.mu.RLock()
	d := s.deploys[id]
	s.mu.RUnlock()
	if d == nil {
		if dbd, err := s.getDeployFromDB(id); err == nil {
			d = dbd
		}
	}
	if d == nil {
		httpError(w, 404, "not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpError(w, 500, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	from := int64(0)
	if q := r.URL.Query().Get("from"); q != "" {
		if v, err := strconv.ParseInt(q, 10, 64); err == nil && v >= 0 {
			from = v
		}
	}

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()

	var f *os.File
	var err error
	for start := time.Now(); ; {
		select {
		case <-ctx.Done():
			return
		default:
		}
		f, err = os.Open(d.LogPath)
		if err == nil {
			break
		}
		if time.Since(start) > 30*time.Second {
			httpError(w, 404, "log not found")
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	defer f.Close()

	_, _ = f.Seek(from, io.SeekStart)
	br := bufio.NewReaderSize(f, 64*1024)
	terminalEOFs := 0

	fmt.Fprint(w, ": stream connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		default:
		}

		line, readErr := br.ReadString('\n')
		if len(line) > 0 {
			terminalEOFs = 0
			rawLen := int64(len(line))
			trimmed := strings.TrimRight(line, "\r\n")
			if trimmed != "" {
				fmt.Fprintf(w, "data: %s\n\n", trimmed)
				flusher.Flush()
			}
			from += rawLen
		}

		if readErr != nil {
			if readErr == io.EOF {
				latest := d
				if dbd, err := s.getDeployFromDB(id); err == nil && dbd != nil {
					latest = dbd
					d = dbd
				}
				if isTerminalDeployStatus(latest.Status) {
					terminalEOFs++
					if terminalEOFs >= 4 {
						fmt.Fprintf(w, "event: deploy-status\ndata: {\"status\":\"%s\"}\n\n", latest.Status)
						flusher.Flush()
						return
					}
				}
				time.Sleep(250 * time.Millisecond)
				continue
			}
			return
		}
	}
}

func isTerminalDeployStatus(status DeployStatus) bool {
	return status == StatusSuccess || status == StatusFailed
}

func (s *Server) pruneAppImages(app string, env DeployEnv, branch string, keep ...string) error {
	repo := fmt.Sprintf("relay/%s", safe(app))
	tagPrefix := fmt.Sprintf("%s-%s-", safe(string(env)), safe(branch))
	keepSet := map[string]struct{}{}
	for _, img := range keep {
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}
		keepSet[img] = struct{}{}
	}

	images, err := s.runtime.ListImages(repo)
	if err != nil {
		return fmt.Errorf("list app images failed: %v", err)
	}

	for _, img := range images {
		if _, ok := keepSet[img]; ok {
			continue
		}
		if !strings.HasPrefix(img, repo+":") {
			continue
		}
		tag := strings.TrimPrefix(img, repo+":")
		if !strings.HasPrefix(tag, tagPrefix) {
			continue
		}
		s.runtime.RemoveImage(img)
	}
	return nil
}

func (s *Server) handleSyncStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req SyncStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}

	// Workspace conflict check: if the caller has a known base version, verify
	// it still matches the server's current workspace. A mismatch means someone
	// else deployed in between and the caller needs to run "relay pull" first.
	if req.BaseVersion != "" {
		if st, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && st != nil && st.RepoHash != "" {
			if req.BaseVersion != st.RepoHash {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":          "workspace behind server",
					"server_version": st.RepoHash,
					"hint":           "run: relay pull",
				})
				return
			}
		}
	}

	// Capture current workspace version to return to client (before this deploy).
	wsVersion := ""
	if st, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && st != nil {
		wsVersion = st.RepoHash
	}

	sessionID := newID()
	workspace := filepath.Join(s.workspacesDir, fmt.Sprintf("%s__%s__%s", safe(req.App), safe(string(req.Env)), safe(req.Branch)))
	repoDir := filepath.Join(workspace, "repo")
	stagingDir := filepath.Join(workspace, "staging", sessionID)

	mustMkdir(repoDir)
	mustMkdir(stagingDir)

	maxBytes := int64(524288000)
	if v := os.Getenv("RELAY_MAX_UPLOAD_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxBytes = n
		}
	}

	sess := &SyncSession{
		ID:            sessionID,
		App:           req.App,
		Branch:        req.Branch,
		Env:           req.Env,
		RepoDir:       repoDir,
		StagingDir:    stagingDir,
		CreatedAt:     time.Now(),
		DeleteList:    []string{},
		UploadedBytes: 0,
		MaxBytes:      maxBytes,
	}

	s.syncMu.Lock()
	s.syncSessions[sessionID] = sess
	s.syncMu.Unlock()

	_ = s.saveSessionToDB(sess)
	writeJSON(w, 200, SyncStartResponse{SessionID: sessionID, WorkspaceVersion: wsVersion})
}

func (s *Server) handleSyncPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sync/plan/")
	sess := s.getSession(sessionID)
	if sess == nil {
		httpError(w, 404, "session not found")
		return
	}

	var req SyncPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}

	client := map[string]ManifestFile{}
	for _, f := range req.Files {
		if f.Path == "" {
			continue
		}
		p := filepath.ToSlash(strings.TrimPrefix(f.Path, "./"))
		client[p] = f
	}

	serverPaths := map[string]bool{}
	need := make([]string, 0)
	deleteList := make([]string, 0)

	syncIgnoreTop := map[string]bool{
		"node_modules": true,
		".git":         true,
		".next":        true,
		"dist":         true,
		".turbo":       true,
		"coverage":     true,
		".relay":       true,
		"cache":        true,
		"bin":          true,
		"obj":          true,
		"target":       true,
	}

	_ = filepath.Walk(sess.RepoDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(sess.RepoDir, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		parts := strings.Split(rel, "/")
		if len(parts) > 0 && syncIgnoreTop[parts[0]] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}
		serverPaths[rel] = true

		cf, ok := client[rel]
		if !ok {
			deleteList = append(deleteList, rel)
			return nil
		}

		serverSize := info.Size()
		serverMtime := info.ModTime().UnixMilli()

		algo := strings.ToLower(strings.TrimSpace(cf.HashAlgo))
		expected := strings.TrimSpace(cf.Hash)
		if expected == "" {
			algo = "sha256"
			expected = cf.Sha256
		}

		if expected != "" {
			sh, err := fileHashByAlgo(p, algo)
			if err != nil || sh != expected {
				need = append(need, rel)
			}
			return nil
		}

		if cf.Size == serverSize && cf.Mtime == serverMtime {
			return nil
		}
		need = append(need, rel)
		return nil
	})

	for rel := range client {
		if !serverPaths[rel] {
			need = append(need, rel)
		}
	}

	sort.Strings(need)
	sort.Strings(deleteList)
	writeJSON(w, 200, SyncPlanResponse{Need: need, Delete: deleteList})
}

func (s *Server) handleSyncUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httpError(w, 405, "method not allowed")
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sync/upload/")
	sess := s.getSession(sessionID)
	if sess == nil {
		httpError(w, 404, "session not found")
		return
	}

	rel := r.URL.Query().Get("path")
	if rel == "" {
		httpError(w, 400, "missing path")
		return
	}
	rel = filepath.ToSlash(strings.TrimPrefix(rel, "./"))
	if !isSafeRelPath(rel) {
		httpError(w, 400, "invalid path")
		return
	}

	sess.uploadMu.Lock()
	defer sess.uploadMu.Unlock()

	remaining := sess.MaxBytes - sess.UploadedBytes
	if remaining <= 0 {
		httpError(w, 413, "session upload quota exceeded")
		return
	}
	if r.ContentLength > 0 {
		if r.ContentLength > remaining {
			httpError(w, 413, "upload exceeds session remaining quota")
			return
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, remaining)
	}

	dst := filepath.Join(sess.StagingDir, filepath.FromSlash(rel))
	mustMkdir(filepath.Dir(dst))

	f, err := os.Create(dst)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	defer f.Close()

	n, err := io.Copy(f, r.Body)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}

	sess.UploadedBytes += n
	_ = s.saveSessionToDB(sess)

	writeJSON(w, 200, map[string]string{"ok": "true"})
}

func (s *Server) handleSyncBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httpError(w, 405, "method not allowed")
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sync/bundle/")
	sess := s.getSession(sessionID)
	if sess == nil {
		httpError(w, 404, "session not found")
		return
	}

	sess.uploadMu.Lock()
	defer sess.uploadMu.Unlock()

	remaining := sess.MaxBytes - sess.UploadedBytes
	if remaining <= 0 {
		httpError(w, 413, "session upload quota exceeded")
		return
	}
	if r.ContentLength > 0 {
		if r.ContentLength > remaining {
			httpError(w, 413, "bundle exceeds session remaining quota")
			return
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, remaining)
	}

	dec, err := zstd.NewReader(r.Body)
	if err != nil {
		httpError(w, 400, "invalid zstd stream")
		return
	}
	defer dec.Close()

	tr := tar.NewReader(dec)
	totalWritten := int64(0)
	count := 0

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			httpError(w, 400, "invalid tar stream")
			return
		}

		name := filepath.ToSlash(strings.TrimPrefix(hdr.Name, "./"))
		if name == "" || !isSafeRelPath(name) {
			httpError(w, 400, "invalid path in bundle")
			return
		}

		if hdr.Size > (remaining - totalWritten) {
			httpError(w, 413, "bundle entry exceeds session remaining quota")
			return
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			mustMkdir(filepath.Join(sess.StagingDir, filepath.FromSlash(name)))
			continue
		case tar.TypeReg, tar.TypeRegA:
		default:
			continue
		}

		dst := filepath.Join(sess.StagingDir, filepath.FromSlash(name))
		mustMkdir(filepath.Dir(dst))

		f, err := os.Create(dst)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		wn, err := io.Copy(f, tr)
		_ = f.Close()
		if err != nil {
			if strings.Contains(err.Error(), "http: request body too large") {
				httpError(w, 413, "upload exceeds session remaining quota")
				return
			}
			httpError(w, 500, err.Error())
			return
		}
		totalWritten += wn
		if totalWritten > remaining {
			httpError(w, 413, "bundle exceeds session remaining quota")
			return
		}
		count++
	}

	sess.UploadedBytes += totalWritten
	_ = s.saveSessionToDB(sess)

	writeJSON(w, 200, map[string]any{"ok": true, "files": count, "bytes": totalWritten})
}

func (s *Server) handleSyncDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sync/delete/")
	sess := s.getSession(sessionID)
	if sess == nil {
		httpError(w, 404, "session not found")
		return
	}

	var req SyncDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}

	paths := make([]string, 0, len(req.Paths))
	for _, p := range req.Paths {
		p = filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(p), "./"))
		if p == "" || !isSafeRelPath(p) {
			continue
		}
		paths = append(paths, p)
	}

	if len(paths) == 0 {
		writeJSON(w, 200, map[string]string{"ok": "true"})
		return
	}

	s.syncMu.Lock()
	sess.DeleteList = append(sess.DeleteList, paths...)
	_ = s.saveSessionToDB(sess)
	s.syncMu.Unlock()

	writeJSON(w, 200, map[string]string{"ok": "true"})
}

func (s *Server) handleSyncFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sync/finish/")
	sess := s.getSession(sessionID)
	if sess == nil {
		httpError(w, 404, "session not found")
		return
	}

	if err := applySyncUpdates(sess); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	// Snapshot the workspace fingerprint now (after files are applied, before
	// the async build starts) so we can return it to the client.
	newWorkspaceVersion := repoFingerprint(sess.RepoDir)

	var body struct {
		Mode        string `json:"mode"`
		HostPort    int    `json:"host_port"`
		ServicePort int    `json:"service_port"`
		PublicHost  string `json:"public_host"`
		Source      string `json:"source"`
		InstallCmd  string `json:"install_cmd"`
		BuildCmd    string `json:"build_cmd"`
		StartCmd    string `json:"start_cmd"`
		Engine      string `json:"engine,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Persist engine preference to app state when provided so that runDeploy
	// picks it up even if no prior app state exists (first deploy of a new project).
	if body.Engine != "" {
		if eng := normalizeEngine(body.Engine); eng != "" {
			st, _ := s.getAppState(sess.App, sess.Env, sess.Branch)
			if st == nil {
				st = &AppState{App: sess.App, Env: sess.Env, Branch: sess.Branch}
			}
			st.Engine = eng
			_ = s.saveAppState(st)
		}
	}

	// Capture who triggered this deploy from the authenticated session.
	deployedBy := ""
	if userSess := s.validateUserSession(r); userSess != nil {
		deployedBy = userSess.Username
	}

	deployID := newID()
	logPath := filepath.Join(s.logsDir, deployID+".log")
	buildNum := s.nextBuildNumber(sess.App)
	deploy := &Deploy{
		ID:          deployID,
		App:         sess.App,
		RepoURL:     "",
		Branch:      sess.Branch,
		Env:         sess.Env,
		Status:      StatusQueued,
		CreatedAt:   time.Now(),
		LogPath:     logPath,
		BuildNumber: buildNum,
		DeployedBy:  deployedBy,
	}

	mode := strings.ToLower(strings.TrimSpace(body.Mode))

	source := strings.TrimSpace(body.Source)
	if source == "" {
		source = "sync"
	}

	req := DeployRequest{
		App:              sess.App,
		RepoURL:          "",
		Branch:           sess.Branch,
		Env:              sess.Env,
		Source:           source,
		Mode:             mode,
		HostPort:         body.HostPort,
		HostPortExplicit: body.HostPort > 0,
		ServicePort:      body.ServicePort,
		PublicHost:       strings.TrimSpace(body.PublicHost),
		InstallCmd:       body.InstallCmd,
		BuildCmd:         body.BuildCmd,
		StartCmd:         body.StartCmd,
		Engine:           normalizeEngine(body.Engine),
		DeployedBy:       deployedBy,
	}

	s.mu.Lock()
	s.deploys[deployID] = deploy
	s.mu.Unlock()

	_ = s.saveDeployToDB(deploy, req)
	s.auditLog(deployedBy, "deploy.trigger", sess.App, fmt.Sprintf("build #%d env=%s branch=%s", buildNum, sess.Env, sess.Branch))
	s.queue <- DeployJob{ID: deployID, Req: req}
	s.broadcastSnapshot()

	s.syncMu.Lock()
	delete(s.syncSessions, sessionID)
	s.syncMu.Unlock()
	_ = s.deleteSessionFromDB(sessionID)

	writeJSON(w, 200, map[string]any{
		"id":                deploy.ID,
		"app":               deploy.App,
		"branch":            deploy.Branch,
		"env":               deploy.Env,
		"status":            deploy.Status,
		"created_at":        deploy.CreatedAt,
		"build_number":      deploy.BuildNumber,
		"workspace_version": newWorkspaceVersion,
	})
}

// webhookAllowed returns true if the repo has not exceeded the rate limit
// (5 triggers per 60 seconds). Call sites hold no locks.
func (s *Server) webhookAllowed(repoURL string) bool {
	now := time.Now()
	cutoff := now.Add(-60 * time.Second)
	s.webhookRateMu.Lock()
	defer s.webhookRateMu.Unlock()
	hits := s.webhookHits[repoURL]
	var recent []time.Time
	for _, t := range hits {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 5 {
		s.webhookHits[repoURL] = recent
		return false
	}
	s.webhookHits[repoURL] = append(recent, now)
	return true
}

func (s *Server) handleGithubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}

	// Read body once for signature verification and JSON decode.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, 400, "failed to read body")
		return
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Global webhook secret (fallback if no per-app secret matches).
	globalSecret := strings.TrimSpace(os.Getenv("RELAY_GITHUB_WEBHOOK_SECRET"))
	if globalSecret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifyGithubSig256([]byte(globalSecret), body, sig) {
			httpError(w, 401, "invalid signature")
			return
		}
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "" {
		httpError(w, 400, "missing X-GitHub-Event")
		return
	}
	if event != "push" {
		writeJSON(w, 200, map[string]string{"status": "ignored", "reason": "not a push event"})
		return
	}

	var payload struct {
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Repository struct {
			CloneURL string `json:"clone_url"`
			HTMLURL  string `json:"html_url"`
		} `json:"repository"`
		HeadCommit struct {
			Message string `json:"message"`
		} `json:"head_commit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, 400, "invalid json")
		return
	}

	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	// Try to match HTTPS clone URL or HTML URL
	repoURLs := []string{payload.Repository.CloneURL, payload.Repository.HTMLURL}
	// Also try appending/removing .git
	for _, u := range repoURLs {
		if strings.HasSuffix(u, ".git") {
			repoURLs = append(repoURLs, strings.TrimSuffix(u, ".git"))
		} else {
			repoURLs = append(repoURLs, u+".git")
		}
	}

	query := `SELECT app, env, branch, mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(webhook_secret,'') FROM app_state WHERE repo_url IN (` +
		strings.Repeat("?,", len(repoURLs)-1) + `?) AND branch=?`
	args := make([]any, 0, len(repoURLs)+1)
	for _, u := range repoURLs {
		args = append(args, u)
	}
	args = append(args, branch)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		httpError(w, 500, "db error: "+err.Error())
		return
	}
	defer rows.Close()

	sig := r.Header.Get("X-Hub-Signature-256")

	count := 0
	matched := 0
	secretProtected := 0
	for rows.Next() {
		var st AppState
		var envS, webhookSecret string
		if err := rows.Scan(&st.App, &envS, &st.Branch, &st.Mode, &st.HostPort, &st.HostPortExplicit, &st.ServicePort, &st.PublicHost, &webhookSecret); err != nil {
			continue
		}
		st.Env = DeployEnv(envS)
		matched++

		// Per-app secret check: if the app has a webhook secret configured,
		// verify the request signature against it (overrides global check).
		if globalSecret == "" {
			if webhookSecret == "" {
				continue
			}
			secretProtected++
			if !verifyGithubSig256([]byte(webhookSecret), body, sig) {
				continue
			}
		}

		deployID := newID()
		commitMsg := strings.SplitN(strings.TrimSpace(payload.HeadCommit.Message), "\n", 2)[0]
		buildNum := s.nextBuildNumber(st.App)
		deploy := &Deploy{
			ID:            deployID,
			App:           st.App,
			RepoURL:       payload.Repository.CloneURL,
			Branch:        st.Branch,
			CommitSHA:     payload.After,
			Env:           st.Env,
			Status:        StatusQueued,
			CreatedAt:     time.Now(),
			LogPath:       filepath.Join(s.logsDir, deployID+".log"),
			BuildNumber:   buildNum,
			CommitMessage: commitMsg,
		}

		req := DeployRequest{
			App:              st.App,
			RepoURL:          payload.Repository.CloneURL,
			Branch:           st.Branch,
			CommitSHA:        payload.After,
			Env:              st.Env,
			Source:           "git",
			Mode:             st.Mode,
			HostPort:         st.HostPort,
			HostPortExplicit: st.HostPortExplicit,
			ServicePort:      st.ServicePort,
			PublicHost:       st.PublicHost,
			CommitMessage:    commitMsg,
		}

		s.mu.Lock()
		s.deploys[deployID] = deploy
		s.mu.Unlock()

		_ = s.saveDeployToDB(deploy, req)
		shortSHA := payload.After
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		if !s.webhookAllowed(payload.Repository.CloneURL) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"status": "rate_limited", "triggered": 0})
			return
		}
		s.auditLog("github", "deploy.trigger", st.App, fmt.Sprintf("build #%d commit=%s env=%s", buildNum, shortSHA, st.Env))
		s.queue <- DeployJob{ID: deployID, Req: req}
		s.broadcastSnapshot()
		count++
	}

	if globalSecret == "" && matched > 0 && secretProtected == 0 {
		httpError(w, 401, "github webhook secret required")
		return
	}
	if globalSecret == "" && secretProtected > 0 && count == 0 {
		httpError(w, 401, "invalid signature")
		return
	}

	writeJSON(w, 200, map[string]any{"status": "ok", "triggered": count})
}

func verifyGithubSig256(secret, body []byte, sigHeader string) bool {
	// header format: "sha256=<hex>"
	const prefix = "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return false
	}
	gotHex := strings.TrimSpace(sigHeader[len(prefix):])
	got, err := hex.DecodeString(gotHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	exp := mac.Sum(nil)

	return subtle.ConstantTimeCompare(got, exp) == 1
}

// ---------------------- Rollback ----------------------

type RollbackRequest struct {
	App    string    `json:"app"`
	Branch string    `json:"branch"`
	Env    DeployEnv `json:"env"`
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}
	deployID, err := s.enqueueRollbackForLane(req, requestActorLabel(s, r), "rollback")
	if err != nil {
		httpError(w, 400, err.Error())
		return
	}
	deploy, err := s.getDeployFromDB(deployID)
	if err != nil {
		httpError(w, 500, "rollback queued but deploy could not be loaded")
		return
	}
	writeJSON(w, 200, deploy)
}

func (s *Server) handleDeployCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/deploys/cancel/")
	if id == "" {
		httpError(w, 400, "missing deploy id")
		return
	}
	s.buildCancelsMu.Lock()
	cancel, ok := s.buildCancels[id]
	s.buildCancelsMu.Unlock()
	if !ok {
		httpError(w, 404, "no active build for this deploy")
		return
	}
	cancel()
	writeJSON(w, 200, map[string]string{"status": "cancelled"})
}

// ---------------------- App control (start/stop/restart) ----------------------

type AppActionRequest struct {
	App    string    `json:"app"`
	Branch string    `json:"branch"`
	Env    DeployEnv `json:"env"`
}

type ProjectDeleteRequest struct {
	App string `json:"app"`
}

func (s *Server) handleAppStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req AppActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}

	s.stopDockerAppLane(req.App, req.Env, req.Branch)
	_ = s.stopStationLane(req.App, req.Env, req.Branch)
	s.stopLaneServices(req.App, req.Env, req.Branch)
	if st, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && st != nil {
		st.Stopped = true
		s.constrainAppState(st)
		_ = s.saveAppState(st)
	}
	msg := "stopped"
	writeJSON(w, 200, map[string]string{"status": "stopped", "output": msg})
}

func (s *Server) handleAppStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req AppActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}

	st, err := s.getAppState(req.App, req.Env, req.Branch)
	if err != nil || st == nil || st.CurrentImage == "" {
		httpError(w, 400, "no image found for app")
		return
	}
	s.constrainAppState(st)
	engine := firstNonEmptyEngine(st.Engine)

	networkName := ""
	extraEnv := map[string]string{}
	logf := func(string, ...any) {}
	controlReq := DeployRequest{
		App:              st.App,
		Branch:           st.Branch,
		Env:              st.Env,
		HostPort:         st.HostPort,
		HostPortExplicit: st.HostPortExplicit,
		ServicePort:      st.ServicePort,
		PublicHost:       st.PublicHost,
		Mode:             st.Mode,
		TrafficMode:      st.TrafficMode,
	}
	s.assignPreviewHostPort(s.runtimeForEngine(engine), &controlReq, st, nil)
	st.HostPort = firstNonZero(controlReq.HostPort, st.HostPort)
	if engine == EngineDocker {
		_ = s.stopStationLane(req.App, req.Env, req.Branch)
		desiredServices, err := s.resolveCompanionSpecs(req.App, req.Env, req.Branch, s.workspaceRepoDir(req.App, req.Env, req.Branch))
		if err == nil && len(desiredServices) > 0 {
			desiredMap := map[string]ServiceConfig{}
			for _, svc := range desiredServices {
				desiredMap[svc.Name] = svc
			}
			networkName = fmt.Sprintf("relay-%s-%s-%s", safe(req.App), safe(string(req.Env)), safe(req.Branch))
			if err := s.runtime.EnsureNetwork(networkName); err != nil {
				httpError(w, 500, err.Error())
				return
			}
			for _, svc := range desiredServices {
				if !serviceShouldRun(svc) {
					continue
				}
				k, v, svcErr := s.startProjectService(logf, req.App, string(req.Env), req.Branch, svc, networkName, false)
				if svcErr != nil {
					httpError(w, 500, svcErr.Error())
					return
				}
				if k != "" && v != "" {
					extraEnv[k] = v
				}
			}
			s.reconcileProjectServices(nil, req.App, req.Env, req.Branch, desiredMap)
		}
		if err := s.runContainer(nil, st.App, st.Env, st.Branch, st.CurrentImage, st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, networkName, extraEnv); err != nil {
			httpError(w, 500, err.Error())
			return
		}
	} else {
		s.stopDockerAppLane(req.App, req.Env, req.Branch)
		if secs, err := s.getAppSecrets(req.App, req.Env, req.Branch); err == nil {
			for k, v := range secs {
				extraEnv[k] = v
			}
		}
		if err := s.runStationApp(logf, DeployRequest{
			App:              st.App,
			Env:              st.Env,
			Branch:           st.Branch,
			ServicePort:      st.ServicePort,
			HostPort:         st.HostPort,
			HostPortExplicit: st.HostPortExplicit,
			Mode:             st.Mode,
			TrafficMode:      st.TrafficMode,
			PublicHost:       st.PublicHost,
		}, st.CurrentImage, extraEnv); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if current, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && current != nil {
			st.ActiveSlot = current.ActiveSlot
			st.StandbySlot = current.StandbySlot
			st.DrainUntil = current.DrainUntil
			st.TrafficMode = current.TrafficMode
		}
	}
	st.Stopped = false
	s.constrainAppState(st)
	s.refreshLaneExpiry(st, time.Now())
	_ = s.saveAppState(st)
	_ = s.setLatestDeployPreviewURL(st.App, st.Env, st.Branch, previewURLFromConfig(st.Mode, st.PublicHost, st.HostPort))

	writeJSON(w, 200, map[string]string{"status": "started"})
}

func (s *Server) handleAppRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req AppActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}

	st, err := s.getAppState(req.App, req.Env, req.Branch)
	if err != nil || st == nil || st.CurrentImage == "" {
		httpError(w, 400, "no image found for app")
		return
	}
	s.constrainAppState(st)
	engine := firstNonEmptyEngine(st.Engine)
	controlReq := DeployRequest{
		App:              st.App,
		Branch:           st.Branch,
		Env:              st.Env,
		HostPort:         st.HostPort,
		HostPortExplicit: st.HostPortExplicit,
		ServicePort:      st.ServicePort,
		PublicHost:       st.PublicHost,
		Mode:             st.Mode,
		TrafficMode:      st.TrafficMode,
	}
	s.assignPreviewHostPort(s.runtimeForEngine(engine), &controlReq, st, nil)
	st.HostPort = firstNonZero(controlReq.HostPort, st.HostPort)

	networkName := ""
	extraEnv := map[string]string{}
	logf := func(string, ...any) {}
	if engine == EngineDocker {
		_ = s.stopStationLane(req.App, req.Env, req.Branch)
		desiredServices, err := s.resolveCompanionSpecs(req.App, req.Env, req.Branch, s.workspaceRepoDir(req.App, req.Env, req.Branch))
		if err == nil && len(desiredServices) > 0 {
			desiredMap := map[string]ServiceConfig{}
			for _, svc := range desiredServices {
				desiredMap[svc.Name] = svc
			}
			networkName = fmt.Sprintf("relay-%s-%s-%s", safe(req.App), safe(string(req.Env)), safe(req.Branch))
			if err := s.runtime.EnsureNetwork(networkName); err != nil {
				httpError(w, 500, err.Error())
				return
			}
			for _, svc := range desiredServices {
				if !serviceShouldRun(svc) {
					continue
				}
				k, v, svcErr := s.startProjectService(logf, req.App, string(req.Env), req.Branch, svc, networkName, false)
				if svcErr != nil {
					httpError(w, 500, svcErr.Error())
					return
				}
				if k != "" && v != "" {
					extraEnv[k] = v
				}
			}
			s.reconcileProjectServices(nil, req.App, req.Env, req.Branch, desiredMap)
		}
		if err := s.runContainer(nil, st.App, st.Env, st.Branch, st.CurrentImage, st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, networkName, extraEnv); err != nil {
			httpError(w, 500, err.Error())
			return
		}
	} else {
		s.stopDockerAppLane(req.App, req.Env, req.Branch)
		if secs, err := s.getAppSecrets(req.App, req.Env, req.Branch); err == nil {
			for k, v := range secs {
				extraEnv[k] = v
			}
		}
		if err := s.runStationApp(logf, DeployRequest{
			App:              st.App,
			Env:              st.Env,
			Branch:           st.Branch,
			ServicePort:      st.ServicePort,
			HostPort:         st.HostPort,
			HostPortExplicit: st.HostPortExplicit,
			Mode:             st.Mode,
			TrafficMode:      st.TrafficMode,
			PublicHost:       st.PublicHost,
		}, st.CurrentImage, extraEnv); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if current, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && current != nil {
			st.ActiveSlot = current.ActiveSlot
			st.StandbySlot = current.StandbySlot
			st.DrainUntil = current.DrainUntil
			st.TrafficMode = current.TrafficMode
		}
	}
	st.Stopped = false
	s.constrainAppState(st)
	s.refreshLaneExpiry(st, time.Now())
	_ = s.saveAppState(st)
	_ = s.setLatestDeployPreviewURL(st.App, st.Env, st.Branch, previewURLFromConfig(st.Mode, st.PublicHost, st.HostPort))
	writeJSON(w, 200, map[string]string{"status": "restarted"})
}

func (s *Server) handleAppConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			App           string    `json:"app"`
			Env           DeployEnv `json:"env"`
			Branch        string    `json:"branch"`
			RepoURL       *string   `json:"repo_url"`
			Engine        *string   `json:"engine"`
			Mode          *string   `json:"mode"`
			TrafficMode   *string   `json:"traffic_mode"`
			AccessPolicy  *string   `json:"access_policy"`
			IPAllowlist   *string   `json:"ip_allowlist"`
			HostPort      *int      `json:"host_port"`
			ServicePort   *int      `json:"service_port"`
			PublicHost    *string   `json:"public_host"`
			WebhookSecret *string   `json:"webhook_secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		body.Env = normalizeDeployEnv(string(body.Env))
		if !validDeployTarget(body.App, body.Env, body.Branch) {
			httpError(w, 400, "app, branch, env required")
			return
		}

		var previousState *AppState
		st, _ := s.getAppState(body.App, body.Env, body.Branch)
		if st != nil {
			copyState := *st
			previousState = &copyState
		}
		if st == nil {
			st = &AppState{
				App:    body.App,
				Env:    body.Env,
				Branch: body.Branch,
				Engine: detectDefaultEngine(),
			}
		}
		st.Engine = firstNonEmptyEngine(st.Engine)

		if body.RepoURL != nil {
			st.RepoURL = strings.TrimSpace(*body.RepoURL)
		}
		if body.Engine != nil {
			engine := normalizeEngine(*body.Engine)
			if engine == "" {
				httpError(w, 400, "engine must be docker or station")
				return
			}
			st.Engine = engine
		}
		if body.Mode != nil {
			mode := strings.ToLower(strings.TrimSpace(*body.Mode))
			if mode != "" && mode != "port" && mode != "traefik" {
				httpError(w, 400, "mode must be port or traefik")
				return
			}
			st.Mode = mode
		}
		if body.TrafficMode != nil {
			trafficMode := normalizeTrafficMode(*body.TrafficMode)
			if trafficMode == "" {
				httpError(w, 400, "traffic_mode must be edge or session")
				return
			}
			st.TrafficMode = trafficMode
		}
		if body.AccessPolicy != nil {
			accessPolicy := normalizeAccessPolicy(*body.AccessPolicy)
			if accessPolicy == "" {
				httpError(w, 400, "access_policy must be public, relay-login, signed-link, or ip-allowlist")
				return
			}
			st.AccessPolicy = accessPolicy
		}
		if body.IPAllowlist != nil {
			st.IPAllowlist = normalizeIPAllowlist(*body.IPAllowlist)
		}
		if body.HostPort != nil {
			if *body.HostPort < 0 {
				httpError(w, 400, "host_port must be >= 0")
				return
			}
			st.HostPort = *body.HostPort
			st.HostPortExplicit = *body.HostPort > 0
		}
		if body.ServicePort != nil {
			if *body.ServicePort < 0 {
				httpError(w, 400, "service_port must be >= 0")
				return
			}
			st.ServicePort = *body.ServicePort
		}
		if body.PublicHost != nil {
			publicHost := strings.TrimSpace(*body.PublicHost)
			if err := validateProxyHostname(publicHost, "public_host"); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			if publicHost != "" && (normalizedHostname(publicHost) == normalizedRequestHost(r) || normalizedHostname(publicHost) == normalizedHostname(s.serverDashboardHost())) {
				httpError(w, 400, "public_host cannot match the Relay dashboard host; use a different subdomain for apps")
				return
			}
			st.PublicHost = publicHost
		}
		if body.WebhookSecret != nil {
			st.WebhookSecret = strings.TrimSpace(*body.WebhookSecret)
		}
		if normalizeAccessPolicy(st.AccessPolicy) == AccessPolicyIPAllowlist && strings.TrimSpace(st.IPAllowlist) == "" {
			httpError(w, 400, "ip_allowlist is required when access_policy is ip-allowlist")
			return
		}
		policy := s.lanePolicy(st.Env)
		if normalizeAccessPolicy(st.AccessPolicy) == "" {
			st.AccessPolicy = policy.DefaultAccessPolicy
		}
		s.constrainAppState(st)
		if err := s.saveAppState(st); err != nil {
			httpError(w, 500, "failed to save app state: "+err.Error())
			return
		}
		proxyNeedsRefresh := false
		if previousState == nil {
			proxyNeedsRefresh = strings.TrimSpace(st.PublicHost) != ""
		} else {
			proxyNeedsRefresh = strings.TrimSpace(previousState.PublicHost) != strings.TrimSpace(st.PublicHost) || previousState.HostPort != st.HostPort
		}
		if proxyNeedsRefresh {
			if err := s.ensureGlobalProxy(); err != nil {
				if previousState != nil {
					_ = s.saveAppState(previousState)
				} else {
					_, _ = s.db.Exec(`DELETE FROM app_state WHERE app=? AND env=? AND branch=?`, st.App, string(st.Env), st.Branch)
				}
				httpError(w, 500, "failed to refresh global proxy: "+err.Error())
				return
			}
		}
		s.broadcastSnapshot()
		writeJSON(w, 200, st)

	case http.MethodGet:
		app := r.URL.Query().Get("app")
		env := r.URL.Query().Get("env")
		branch := r.URL.Query().Get("branch")
		if app == "" {
			httpError(w, 400, "app required")
			return
		}
		e := normalizeDeployEnv(env)
		if !isKnownDeployEnv(e) {
			writeJSON(w, 200, map[string]any{})
			return
		}
		st, err := s.getAppState(app, e, branch)
		if err != nil {
			// Not found or other DB error: return empty object so UI can continue.
			writeJSON(w, 200, map[string]any{})
			return
		}
		if st == nil {
			writeJSON(w, 200, map[string]any{})
			return
		}
		writeJSON(w, 200, st)

	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) handleAppCompanions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app := r.URL.Query().Get("app")
		env := DeployEnv(r.URL.Query().Get("env"))
		branch := r.URL.Query().Get("branch")
		env = normalizeDeployEnv(string(env))
		if !validDeployTarget(app, env, branch) {
			httpError(w, 400, "app, branch, env required")
			return
		}
		specs, err := s.getServiceSpecs(app, string(env), branch)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		repoSpecs, _ := s.resolveCompanionSpecs(app, env, branch, s.workspaceRepoDir(app, env, branch))
		running, _ := s.getProjectServices(app, string(env), branch)
		runningByName := map[string]ProjectService{}
		for _, svc := range running {
			runningByName[svc.Name] = svc
		}
		type companionItem struct {
			Config    ServiceConfig   `json:"config"`
			Running   *ProjectService `json:"running,omitempty"`
			Managed   bool            `json:"managed"`
			Source    string          `json:"source,omitempty"`
			UpdatedAt int64           `json:"updated_at,omitempty"`
		}
		merged := map[string]companionItem{}
		for _, svc := range repoSpecs {
			item := companionItem{Config: svc, Managed: false, Source: "repo"}
			if running, ok := runningByName[svc.Name]; ok {
				run := running
				item.Running = &run
			}
			merged[svc.Name] = item
		}
		for _, rec := range specs {
			item := companionItem{
				Config:    rec.Config,
				Managed:   true,
				Source:    "agent",
				UpdatedAt: rec.UpdatedAt,
			}
			if existing, ok := merged[rec.Config.Name]; ok && existing.Source == "repo" {
				item.Source = "merged"
			}
			if running, ok := runningByName[rec.Config.Name]; ok {
				run := running
				item.Running = &run
			}
			merged[rec.Config.Name] = item
		}
		names := make([]string, 0, len(merged))
		for name := range merged {
			names = append(names, name)
		}
		sort.Strings(names)
		items := make([]companionItem, 0, len(names))
		for _, name := range names {
			items = append(items, merged[name])
		}
		writeJSON(w, 200, items)
	case http.MethodPost:
		var body struct {
			App    string        `json:"app"`
			Env    DeployEnv     `json:"env"`
			Branch string        `json:"branch"`
			Config ServiceConfig `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		body.Env = normalizeDeployEnv(string(body.Env))
		if !validDeployTarget(body.App, body.Env, body.Branch) {
			httpError(w, 400, "app, branch, env required")
			return
		}
		cfg := normalizeServiceConfig(body.Config)
		if cfg.Name == "" {
			httpError(w, 400, "companion name required")
			return
		}
		if cfg.Type == "" {
			cfg.Type = "custom"
		}
		cfg.Type = defaultServiceType(cfg.Type)
		if (cfg.Type == "custom" || cfg.Type == "worker") && serviceImageName(cfg) == "" {
			httpError(w, 400, "custom and worker companions need an image")
			return
		}
		if err := s.saveServiceSpec(body.App, string(body.Env), body.Branch, cfg); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if cfg.Stopped || cfg.Disabled {
			s.stopProjectServiceRuntime(body.App, string(body.Env), body.Branch, cfg.Name)
		} else if s.appLaneRunning(body.App, body.Env, body.Branch) && serviceShouldRun(cfg) {
			networkName := fmt.Sprintf("relay-%s-%s-%s", safe(body.App), safe(string(body.Env)), safe(body.Branch))
			if err := s.runtime.EnsureNetwork(networkName); err != nil {
				httpError(w, 500, err.Error())
				return
			}
			logf := func(string, ...any) {}
			if _, _, err := s.startProjectService(logf, body.App, string(body.Env), body.Branch, cfg, networkName, false); err != nil {
				httpError(w, 500, err.Error())
				return
			}
		}
		s.broadcastSnapshot()
		writeJSON(w, 200, cfg)
	case http.MethodDelete:
		app := r.URL.Query().Get("app")
		env := string(normalizeDeployEnv(r.URL.Query().Get("env")))
		branch := r.URL.Query().Get("branch")
		name := r.URL.Query().Get("name")
		if app == "" || branch == "" || name == "" || !isKnownDeployEnv(DeployEnv(env)) {
			httpError(w, 400, "app, branch, env, name required")
			return
		}
		keepDisabled := false
		if cfg, err := readProjectConfig(s.workspaceRepoDir(app, DeployEnv(env), branch)); err == nil && cfg != nil {
			for _, repoSvc := range cfg.Services {
				if normalizeServiceConfig(repoSvc).Name == name {
					keepDisabled = true
					break
				}
			}
		}
		if keepDisabled {
			_ = s.saveServiceSpec(app, env, branch, ServiceConfig{Name: name, Type: "custom", Disabled: true})
		} else {
			_ = s.deleteServiceSpec(app, env, branch, name)
		}
		s.stopProjectServiceRuntime(app, env, branch, name)
		s.broadcastSnapshot()
		writeJSON(w, 200, map[string]string{"status": "deleted"})
	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) handleCompanionRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var body struct {
		App    string    `json:"app"`
		Env    DeployEnv `json:"env"`
		Branch string    `json:"branch"`
		Name   string    `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	body.Env = normalizeDeployEnv(string(body.Env))
	if body.App == "" || body.Branch == "" || body.Name == "" || !isKnownDeployEnv(body.Env) {
		httpError(w, 400, "app, branch, env, name required")
		return
	}
	var selected *ServiceConfig
	specs, err := s.resolveCompanionSpecs(body.App, body.Env, body.Branch, s.workspaceRepoDir(body.App, body.Env, body.Branch))
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	for _, rec := range specs {
		if rec.Name == body.Name {
			cfg := rec
			selected = &cfg
			break
		}
	}
	if selected == nil {
		httpError(w, 404, "companion not found")
		return
	}
	if selected.Stopped {
		httpError(w, 409, "companion is kept off")
		return
	}
	networkName := fmt.Sprintf("relay-%s-%s-%s", safe(body.App), safe(string(body.Env)), safe(body.Branch))
	if err := s.runtime.EnsureNetwork(networkName); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	logf := func(string, ...any) {}
	if _, _, err := s.startProjectService(logf, body.App, string(body.Env), body.Branch, *selected, networkName, true); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	s.broadcastSnapshot()
	writeJSON(w, 200, map[string]string{"status": "restarted"})
}

func (s *Server) handleAppSecrets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app := r.URL.Query().Get("app")
		env := r.URL.Query().Get("env")
		branch := r.URL.Query().Get("branch")
		if app == "" {
			httpError(w, 400, "app required")
			return
		}

		query := `SELECT app, env, branch, key, value FROM app_secrets WHERE app=?`
		args := []any{app}
		if env != "" {
			query += ` AND (env=? OR env='')`
			args = append(args, env)
		}
		if branch != "" {
			query += ` AND (branch=? OR branch='')`
			args = append(args, branch)
		}

		rows, err := s.db.Query(query, args...)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		defer rows.Close()

		out := []AppSecret{}
		for rows.Next() {
			var sec AppSecret
			if err := rows.Scan(&sec.App, &sec.Env, &sec.Branch, &sec.Key, &sec.Value); err != nil {
				continue
			}
			sec.Value = s.decryptSecret(sec.Value)
			out = append(out, sec)
		}
		writeJSON(w, 200, out)

	case http.MethodPost:
		var sec AppSecret
		if err := json.NewDecoder(r.Body).Decode(&sec); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		if sec.App == "" || sec.Key == "" {
			httpError(w, 400, "app and key required")
			return
		}

		encrypted := s.encryptSecret(sec.Value)
		_, err := s.db.Exec(`INSERT OR REPLACE INTO app_secrets (app, env, branch, key, value) VALUES (?, ?, ?, ?, ?)`,
			sec.App, sec.Env, sec.Branch, sec.Key, encrypted)
		if err != nil {
			httpError(w, 500, "failed to save secret: "+err.Error())
			return
		}
		// Audit secret writes (log key name only, never the value)
		actor := ""
		if userSess := s.validateUserSession(r); userSess != nil {
			actor = userSess.Username
		}
		s.auditLog(actor, "secret.set", sec.App, fmt.Sprintf("key=%s env=%s branch=%s", sec.Key, sec.Env, sec.Branch))
		writeJSON(w, 200, map[string]bool{"ok": true})

	case http.MethodDelete:
		app := r.URL.Query().Get("app")
		env := r.URL.Query().Get("env")
		branch := r.URL.Query().Get("branch")
		key := r.URL.Query().Get("key")
		if app == "" || key == "" {
			httpError(w, 400, "app and key required")
			return
		}

		_, err := s.db.Exec(`DELETE FROM app_secrets WHERE app=? AND env=? AND branch=? AND key=?`,
			app, env, branch, key)
		if err != nil {
			httpError(w, 500, "failed to delete secret: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"ok": true})

	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) getAppSecrets(app string, env DeployEnv, branch string) (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value, env, branch FROM app_secrets WHERE app=?`, app)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]string)

	type entry struct {
		val         string
		specificity int
	}
	entries := make(map[string]entry)

	for rows.Next() {
		var k, v, e, b string
		if err := rows.Scan(&k, &v, &e, &b); err != nil {
			continue
		}
		v = s.decryptSecret(v)

		spec := 0
		if e == string(env) {
			spec += 2
		} else if e != "" {
			continue
		}

		if b == branch {
			spec += 1
		} else if b != "" {
			continue
		}

		if cur, ok := entries[k]; !ok || spec >= cur.specificity {
			entries[k] = entry{val: v, specificity: spec}
		}
	}

	for k, e := range entries {
		res[k] = e.val
	}
	return res, nil
}

// ---------------------- Worker / Deploy pipeline ----------------------

func (s *Server) worker(n int) {
	if n <= 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		go func() {
			for job := range s.queue {
				s.runDeploy(job)
			}
		}()
	}
}

func (s *Server) runDeploy(job DeployJob) {
	req := job.Req
	lockKey := fmt.Sprintf("%s__%s__%s", req.App, req.Env, req.Branch)
	var d *Deploy

	defer func() {
		if r := recover(); r != nil {
			if d != nil {
				end := time.Now()
				if d.StartedAt == nil {
					started := end
					d.StartedAt = &started
				}
				d.Status = StatusFailed
				d.EndedAt = &end
				d.Error = fmt.Sprintf("deploy worker panic: %v", r)
				_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, d.ImageTag, d.PrevImage)
			}
		}
	}()

	s.buildLock.Lock()
	if s.building[lockKey] {
		s.buildLock.Unlock()
		// If already building this specific app/env/branch, retry later without
		// blocking a goroutine on a full queue.
		s.requeueDeployLater(job, 5*time.Second)
		return
	}
	s.building[lockKey] = true
	s.buildLock.Unlock()
	defer func() {
		s.buildLock.Lock()
		delete(s.building, lockKey)
		s.buildLock.Unlock()
	}()

	// Register a cancel func so the admin UI can abort an active build.
	buildCtx, buildCancel := context.WithCancel(context.Background())
	defer buildCancel()
	s.buildCancelsMu.Lock()
	s.buildCancels[job.ID] = buildCancel
	s.buildCancelsMu.Unlock()
	defer func() {
		s.buildCancelsMu.Lock()
		delete(s.buildCancels, job.ID)
		s.buildCancelsMu.Unlock()
	}()

	// Load deploy record (in-memory + db)
	s.mu.RLock()
	d = s.deploys[job.ID]
	s.mu.RUnlock()
	if d == nil {
		if dbd, err := s.getDeployFromDB(job.ID); err == nil {
			d = dbd
		}
	}
	if d == nil {
		return
	}

	// Pull missing config from persistent AppState
	state, _ := s.getAppState(req.App, req.Env, req.Branch)
	if state != nil {
		if req.RepoURL == "" {
			req.RepoURL = state.RepoURL
		}
		if req.Mode == "" {
			req.Mode = state.Mode
		}
		if req.HostPort == 0 {
			req.HostPort = state.HostPort
			req.HostPortExplicit = state.HostPortExplicit
		} else if req.HostPort == state.HostPort {
			req.HostPortExplicit = req.HostPortExplicit || state.HostPortExplicit
		}
		if req.ServicePort == 0 {
			req.ServicePort = state.ServicePort
		}
		if req.PublicHost == "" {
			req.PublicHost = state.PublicHost
		}
		if req.TrafficMode == "" {
			req.TrafficMode = state.TrafficMode
		}
		// Auto-detect source if RepoURL is present but source isn't "sync"
		if req.Source == "" && req.RepoURL != "" {
			req.Source = "git"
		}
	}
	engine := detectDefaultEngine()
	if req.Engine != "" {
		engine = firstNonEmptyEngine(req.Engine)
	} else if state != nil {
		engine = firstNonEmptyEngine(state.Engine)
	}
	constrainDeployRequestForEngine(engine, &req)
	s.applyLaneDefaultsToDeployRequest(&req)

	// Mark running
	now := time.Now()
	d.Status = StatusRunning
	d.StartedAt = &now
	_ = s.updateDeployStatus(d.ID, d.Status, "", d.StartedAt, nil, "", "")

	logf, _ := os.OpenFile(d.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logf == nil {
		// last resort
		d.Status = StatusFailed
		end := time.Now()
		d.EndedAt = &end
		d.Error = "failed to open log file"
		_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
		return
	}
	defer logf.Close()
	log := func(format string, args ...any) {
		fmt.Fprintf(logf, format+"\n", args...)
	}
	s.assignPreviewHostPort(s.runtimeForEngine(engine), &req, state, log)

	// Resolve workspace path
	workspace := filepath.Join(s.workspacesDir, fmt.Sprintf("%s__%s__%s", safe(req.App), safe(string(req.Env)), safe(req.Branch)))
	repoDir := filepath.Join(workspace, "repo")
	mustMkdir(repoDir)

	// If source is git, ensure repo is up to date
	if (req.Source == "git" || strings.TrimSpace(job.PromoteImage) != "") && req.RepoURL != "" {
		log("source is git: preparing repo from %s [%s]", req.RepoURL, req.Branch)
		if !fileExists(filepath.Join(repoDir, ".git")) {
			log("cloning repository...")
			// Clone into a temp dir then move or just clone into repoDir
			// Since repoDir exists (mustMkdir), we might need to remove it or clone inside
			_ = os.RemoveAll(repoDir)
			if err := runCmdLoggedCtx(buildCtx, workspace, logf, "git", "clone", "--depth", "1", "--branch", req.Branch, req.RepoURL, "repo"); err != nil {
				failDeploy(s, d, err, "git clone failed: "+err.Error())
				return
			}
		} else {
			log("updating existing repository...")
			if err := runCmdLoggedCtx(buildCtx, repoDir, logf, "git", "fetch", "origin", req.Branch); err != nil {
				log("fetch failed, attempting clean clone: %v", err)
				_ = os.RemoveAll(repoDir)
				if err := runCmdLoggedCtx(buildCtx, workspace, logf, "git", "clone", "--depth", "1", "--branch", req.Branch, req.RepoURL, "repo"); err != nil {
					failDeploy(s, d, err, "git clone failed after retry: "+err.Error())
					return
				}
			} else {
				if err := runCmdLoggedCtx(buildCtx, repoDir, logf, "git", "reset", "--hard", "origin/"+req.Branch); err != nil {
					failDeploy(s, d, err, "git reset failed: "+err.Error())
					return
				}
			}
		}
	}

	// If rollback: we only re-run container with previous image
	if job.Rollback {
		log("rollback requested -> image=%s", job.RollbackImage)
		state, _ := s.getAppState(req.App, req.Env, req.Branch)
		prevCurrent := ""
		prevPrevious := ""
		if state != nil {
			prevCurrent = state.CurrentImage
			prevPrevious = state.PreviousImage
		}
		rollbackEngine := firstNonEmptyEngine(func() string {
			if state != nil {
				return state.Engine
			}
			return engine
		}())
		if rollbackEngine == EngineDocker {
			_ = s.stopStationLane(req.App, req.Env, req.Branch)
		} else {
			s.stopDockerAppLane(req.App, req.Env, req.Branch)
		}
		extraEnv := map[string]string{}
		if rollbackEngine == EngineStation {
			if secs, err := s.getAppSecrets(req.App, req.Env, req.Branch); err == nil {
				for k, v := range secs {
					extraEnv[k] = v
				}
			}
		}
		var runErr error
		if rollbackEngine == EngineStation {
			runErr = s.runStationApp(log, req, job.RollbackImage, extraEnv)
		} else {
			runErr = s.swapContainer(log, req, job.RollbackImage, "", nil)
		}
		if runErr != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = runErr.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}

		end := time.Now()
		d.Status = StatusSuccess
		d.EndedAt = &end
		d.ImageTag = job.RollbackImage
		d.PrevImage = prevCurrent
		_ = s.updateDeployStatus(d.ID, d.Status, "", d.StartedAt, d.EndedAt, d.ImageTag, d.PrevImage)
		nextState := &AppState{
			App:    req.App,
			Env:    req.Env,
			Branch: req.Branch,
			RepoURL: func() string {
				if state != nil {
					return state.RepoURL
				}
				return req.RepoURL
			}(),
			Engine:        rollbackEngine,
			CurrentImage:  job.RollbackImage,
			PreviousImage: prevCurrent,
			Mode:          firstNonEmpty(req.Mode, "port"),
			TrafficMode: firstNonEmpty(normalizeTrafficMode(req.TrafficMode), func() string {
				if state != nil {
					return state.TrafficMode
				}
				return "edge"
			}()),
			HostPort:         firstNonZero(req.HostPort, defaultHostPort(req.Env)),
			HostPortExplicit: persistedHostPortExplicit(req, state),
			ServicePort:      req.ServicePort,
			PublicHost:       req.PublicHost,
			ActiveSlot: func() string {
				if current, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && current != nil && normalizeActiveSlot(current.ActiveSlot) != "" {
					return current.ActiveSlot
				}
				return s.currentActiveSlotWithRuntime(s.runtimeForEngine(rollbackEngine), req.App, req.Env, req.Branch, state)
			}(),
			StandbySlot: func() string {
				if current, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && current != nil {
					return current.StandbySlot
				}
				return ""
			}(),
			DrainUntil: func() int64 {
				if current, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && current != nil {
					return current.DrainUntil
				}
				return 0
			}(),
			RepoHash: func() string {
				if state != nil {
					return state.RepoHash
				}
				return ""
			}(),
			WebhookSecret: func() string {
				if state != nil {
					return state.WebhookSecret
				}
				return ""
			}(),
			AccessPolicy: func() string {
				if state != nil {
					return state.AccessPolicy
				}
				return ""
			}(),
			IPAllowlist: func() string {
				if state != nil {
					return state.IPAllowlist
				}
				return ""
			}(),
			Stopped: false,
		}
		s.constrainAppState(nextState)
		s.refreshLaneExpiry(nextState, time.Now())
		_ = s.saveAppState(nextState)
		if err := s.pruneRuntimeArtifacts(rollbackEngine, req.App, req.Env, req.Branch, job.RollbackImage, prevCurrent); err != nil {
			log("warning: image prune failed after rollback: %v", err)
		} else if prevPrevious != "" && prevPrevious != job.RollbackImage && prevPrevious != prevCurrent {
			log("removed superseded backup image(s) for %s/%s/%s", req.App, req.Env, req.Branch)
		}
		return
	}

	// Detect relay.config.json (optional)
	var cfg *RelayConfig
	if c, err := readRelayConfig(repoDir); err == nil {
		cfg = c
	}

	// If user supplied a Dockerfile in cfg, validate and bypass buildpack detection.
	var plan BuildPlan
	if cfg != nil && strings.TrimSpace(cfg.Dockerfile) != "" {
		// Resolve dockerfile path and ensure it's inside repoDir
		df := cfg.Dockerfile
		if !filepath.IsAbs(df) {
			df = filepath.Join(repoDir, df)
		}
		absDF, err := filepath.Abs(df)
		if err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "invalid dockerfile path: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
		absRepo, _ := filepath.Abs(repoDir)
		rel, err := filepath.Rel(absRepo, absDF)
		if err != nil || strings.HasPrefix(rel, "..") {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "dockerfile must be inside the repo directory"
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
		// Create a minimal plan using cfg values; we will not write Dockerfile (user provided it)
		plan = BuildPlan{
			Kind:            "custom",
			ServicePort:     firstNonZero(req.ServicePort, cfg.ServicePort),
			BuildImage:      firstNonEmpty(cfgStr(cfg, "BuildImage"), ""),
			RunImage:        firstNonEmpty(cfgStr(cfg, "RunImage"), ""),
			InstallCmd:      firstNonEmpty(cfg.InstallCmd, req.InstallCmd),
			BuildCmd:        firstNonEmpty(cfg.BuildCmd, req.BuildCmd),
			StartCmd:        firstNonEmpty(cfg.StartCmd, req.StartCmd),
			WriteDockerfile: nil,
			Verify:          nil,
		}
	} else {
		// Select buildpack (ConfigBuildpack has priority if cfg exists)
		var pack Buildpack
		for _, bp := range s.buildpacks {
			ok := bp.Detect(repoDir, cfg)
			log("buildpack detect: %s -> %v", bp.Name(), ok)
			if ok {
				pack = bp
				break
			}
		}
		if pack == nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "no buildpack matched (add relay.config.json or a recognizable project file)"
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}

		log("selected buildpack: %s", pack.Name())

		var err error
		plan, err = pack.Plan(req, repoDir, cfg)
		if err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "plan error: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
		if plan.Kind == "next-classic" {
			log("optimization hint: enable output: \"standalone\" in next.config.* for a smaller runtime image and faster deploys")
		}
	}

	// Merge service port: request wins, then config, then plan default
	if req.ServicePort != 0 {
		plan.ServicePort = req.ServicePort
	} else if cfg != nil && cfg.ServicePort != 0 {
		plan.ServicePort = cfg.ServicePort
	}

	// Generate dockerfile (unless plan says not to)
	if plan.WriteDockerfile != nil {
		if err := plan.WriteDockerfile(repoDir); err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "dockerfile error: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
	}

	dockerfilePath := filepath.Join(repoDir, "Dockerfile")
	if cfg != nil && strings.TrimSpace(cfg.Dockerfile) != "" {
		dockerfilePath = cfg.Dockerfile
		if !filepath.IsAbs(dockerfilePath) {
			dockerfilePath = filepath.Join(repoDir, dockerfilePath)
		}
	}

	repoHash := repoFingerprint(repoDir)
	artifactRef := ""
	reusedArtifact := false
	if strings.TrimSpace(job.PromoteImage) != "" {
		artifactRef = strings.TrimSpace(job.PromoteImage)
		reusedArtifact = true
		log("promotion requested: reusing artifact %s", artifactRef)
	} else if s.canReuseRuntimeArtifact(state, engine, repoHash) {
		artifactRef = state.CurrentImage
		reusedArtifact = true
		log("build reuse: unchanged build inputs; reusing %s", artifactRef)
	} else if engine == EngineStation {
		artifactRef = stationSnapshotName(req.App, req.Env, req.Branch, d.ID)
		log("station build starting...")
		if _, err := s.buildStationSnapshot(buildCtx, repoDir, dockerfilePath, artifactRef, logf); err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "station build failed: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
	} else {
		// Build step. If relay.config.json specified a Dockerfile path,
		// pass it to the build so advanced users can provide custom files.
		log("docker build starting...")

		artifactRef = fmt.Sprintf("relay/%s:%s-%s-%s",
			safe(req.App),
			safe(string(req.Env)),
			safe(req.Branch),
			d.ID,
		)

		buildDockerfilePath := ""
		if cfg != nil && strings.TrimSpace(cfg.Dockerfile) != "" {
			buildDockerfilePath = dockerfilePath
		}
		if err := s.runtime.Build(buildCtx, artifactRef, repoDir, buildDockerfilePath, logf); err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "docker build failed: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
	}

	// Verify hook
	if plan.Verify != nil {
		if err := plan.Verify(repoDir); err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "verify failed: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
	}

	prev := state
	appStopped := prev != nil && prev.Stopped

	networkName := ""
	extraEnv := map[string]string{}
	desiredServices, _ := s.resolveCompanionSpecs(req.App, req.Env, req.Branch, repoDir)
	desiredMap := map[string]ServiceConfig{}
	if req.ServicePort == 0 {
		req.ServicePort = plan.ServicePort
	}
	if req.PublicHost == "" {
		if host := s.autoPreviewHostFull(req.Env, req.App, req.Branch, req.PublicHost); host != "" {
			req.PublicHost = host
			req.Mode = "traefik"
			log("auto lane URL: %s", host)
		}
	}

	if engine == EngineDocker {
		for _, svc := range desiredServices {
			if appStopped {
				stoppedSvc := svc
				stoppedSvc.Stopped = true
				desiredMap[svc.Name] = stoppedSvc
				continue
			}
			desiredMap[svc.Name] = svc
		}
		if appStopped {
			log("app lane is kept offline; deploy will build and update the saved image without starting containers")
		} else if len(desiredServices) > 0 {
			networkName = fmt.Sprintf("relay-%s-%s-%s", safe(req.App), safe(string(req.Env)), safe(req.Branch))
			log("setting up project network: %s", networkName)
			if netErr := s.runtime.EnsureNetwork(networkName); netErr != nil {
				log("warning: could not create network: %v", netErr)
				networkName = ""
			} else {
				for _, svc := range desiredServices {
					if !serviceShouldRun(svc) {
						log("service %s is configured to stay off", svc.Name)
						continue
					}
					k, v, svcErr := s.startProjectService(log, req.App, string(req.Env), req.Branch, svc, networkName, false)
					if svcErr != nil {
						log("warning: service %s failed: %v", svc.Name, svcErr)
					} else if k != "" && v != "" {
						extraEnv[k] = v
					}
				}
			}
		}
		s.reconcileProjectServices(log, req.App, req.Env, req.Branch, desiredMap)

		if !appStopped {
			_ = s.stopStationLane(req.App, req.Env, req.Branch)
			if err := s.swapContainer(log, req, artifactRef, networkName, extraEnv); err != nil {
				end := time.Now()
				d.Status = StatusFailed
				d.EndedAt = &end
				d.Error = "run failed: " + err.Error()
				_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
				return
			}
		}
	} else {
		s.stopDockerAppLane(req.App, req.Env, req.Branch)
		for _, svc := range desiredServices {
			if appStopped {
				stoppedSvc := svc
				stoppedSvc.Stopped = true
				desiredMap[svc.Name] = stoppedSvc
				continue
			}
			desiredMap[svc.Name] = svc
		}
		if appStopped {
			log("app lane is kept offline; deploy will build and update the saved station snapshot without starting containers")
		} else if len(desiredServices) > 0 {
			log("station engine does not support companion services; skipping %d service(s)", len(desiredServices))
		}
		s.reconcileProjectServices(log, req.App, req.Env, req.Branch, desiredMap)
		if secs, err := s.getAppSecrets(req.App, req.Env, req.Branch); err == nil {
			for k, v := range secs {
				extraEnv[k] = v
			}
		}
		if !appStopped {
			if err := s.runStationApp(log, req, artifactRef, extraEnv); err != nil {
				end := time.Now()
				d.Status = StatusFailed
				d.EndedAt = &end
				d.Error = "run failed: " + err.Error()
				_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
				return
			}
		}
		if current, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && current != nil {
			req.TrafficMode = current.TrafficMode
			req.Mode = current.Mode
			req.PublicHost = current.PublicHost
		}
	}

	// Persist app state + deploy success
	prevImg := previousDeployImage(prev, artifactRef, reusedArtifact)
	currentState, _ := s.getAppState(req.App, req.Env, req.Branch)
	nextState := &AppState{
		App:           req.App,
		Env:           req.Env,
		Branch:        req.Branch,
		RepoURL:       req.RepoURL,
		Engine:        engine,
		CurrentImage:  artifactRef,
		PreviousImage: prevImg,
		Mode:          firstNonEmpty(req.Mode, "port"),
		TrafficMode: firstNonEmpty(normalizeTrafficMode(req.TrafficMode), func() string {
			if prev != nil {
				return prev.TrafficMode
			}
			return "edge"
		}()),
		HostPort:         firstNonZero(req.HostPort, defaultHostPort(req.Env)),
		HostPortExplicit: persistedHostPortExplicit(req, prev),
		ServicePort:      req.ServicePort,
		PublicHost:       req.PublicHost,
		ActiveSlot: func() string {
			if currentState != nil {
				return currentState.ActiveSlot
			}
			return s.currentActiveSlotWithRuntime(s.runtimeForEngine(engine), req.App, req.Env, req.Branch, prev)
		}(),
		StandbySlot: func() string {
			if currentState != nil {
				return currentState.StandbySlot
			}
			return ""
		}(),
		DrainUntil: func() int64 {
			if currentState != nil {
				return currentState.DrainUntil
			}
			return 0
		}(),
		RepoHash: repoHash,
		WebhookSecret: func() string {
			if currentState != nil {
				return currentState.WebhookSecret
			}
			if prev != nil {
				return prev.WebhookSecret
			}
			return ""
		}(),
		AccessPolicy: func() string {
			if currentState != nil {
				return currentState.AccessPolicy
			}
			if prev != nil {
				return prev.AccessPolicy
			}
			return ""
		}(),
		IPAllowlist: func() string {
			if currentState != nil {
				return currentState.IPAllowlist
			}
			if prev != nil {
				return prev.IPAllowlist
			}
			return ""
		}(),
		Stopped: appStopped,
	}
	s.constrainAppState(nextState)
	s.refreshLaneExpiry(nextState, time.Now())
	_ = s.saveAppState(nextState)

	end := time.Now()
	d.Status = StatusSuccess
	d.EndedAt = &end
	d.ImageTag = artifactRef
	d.PrevImage = prevImg
	_ = s.updateDeployStatus(d.ID, d.Status, "", d.StartedAt, d.EndedAt, d.ImageTag, d.PrevImage)

	// Store a preview URL on the deploy record so the dashboard can show it
	// without guessing the host port.
	if !appStopped && req.PublicHost != "" {
		d.PreviewURL = "https://" + req.PublicHost
	} else if !appStopped && firstNonEmpty(req.Mode, "port") == "port" {
		hostPort := firstNonZero(req.HostPort, defaultHostPort(req.Env))
		if hostPort > 0 {
			d.PreviewURL = fmt.Sprintf("http://127.0.0.1:%d", hostPort)
		}
	}
	if d.PreviewURL != "" {
		_ = s.setDeployPreviewURL(d.ID, d.PreviewURL)
		log("preview URL: %s", d.PreviewURL)
	}

	keepArtifacts := []string{artifactRef}
	if prevImg != "" && prevImg != artifactRef {
		keepArtifacts = append(keepArtifacts, prevImg)
	}
	if err := s.pruneRuntimeArtifacts(engine, req.App, req.Env, req.Branch, keepArtifacts...); err != nil {
		log("warning: image prune failed: %v", err)
	}

	// Cleanup hook
	if plan.Cleanup != nil {
		_ = plan.Cleanup(repoDir)
	}

	// Refresh global domain proxy after every successful deploy
	go func() {
		if err := s.ensureGlobalProxy(); err != nil {
			log("global proxy update: %v", err)
		}
	}()

	log("deploy success. image=%s", artifactRef)
}

func defaultHostPort(env DeployEnv) int {
	return defaultLanePolicy(env).DefaultHostPort
}

func previewHostPortExplicit(req DeployRequest, state *AppState) bool {
	if req.HostPort <= 0 {
		return false
	}
	if req.HostPortExplicit {
		return true
	}
	if state == nil {
		return true
	}
	return state.HostPortExplicit && req.HostPort == state.HostPort
}

func persistedHostPortExplicit(req DeployRequest, state *AppState) bool {
	if req.HostPort <= 0 {
		return false
	}
	if req.HostPortExplicit {
		return true
	}
	return state != nil && state.HostPortExplicit && req.HostPort == state.HostPort
}

func shouldAutoAssignPreviewHostPort(req DeployRequest, state *AppState) bool {
	policy := defaultLanePolicy(req.Env)
	if policy.Env == EnvProd {
		return false
	}
	if strings.TrimSpace(req.PublicHost) != "" {
		return false
	}
	if laneNeedsManagedHost(policy, strings.TrimSpace(os.Getenv("RELAY_BASE_DOMAIN"))) {
		return false
	}
	if firstNonEmpty(strings.ToLower(strings.TrimSpace(req.Mode)), "port") != "port" {
		return false
	}
	if previewHostPortExplicit(req, state) {
		return false
	}
	if state == nil {
		return req.HostPort == 0
	}
	return req.HostPort == 0 || req.HostPort == state.HostPort
}

func firstAvailableHostPort(start int, span int) int {
	if start <= 0 {
		return 0
	}
	if span <= 0 {
		span = 1
	}
	for offset := 0; offset < span; offset++ {
		port := start + offset
		if hostPortAvailable(port) {
			return port
		}
	}
	return 0
}

func hostPortAvailable(port int) bool {
	if port <= 0 || port > 65535 {
		return false
	}
	for _, host := range []string{"127.0.0.1", ""} {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return false
		}
		_ = ln.Close()
	}
	return true
}

func (s *Server) assignPreviewHostPort(runtime ContainerRuntime, req *DeployRequest, state *AppState, log func(string, ...any)) {
	if req == nil {
		return
	}
	if s.lanePolicy(req.Env).Env == EnvProd {
		return
	}
	if strings.TrimSpace(req.PublicHost) != "" {
		return
	}
	if s.autoPreviewHostFull(req.Env, req.App, req.Branch, "") != "" {
		return
	}
	if firstNonEmpty(strings.ToLower(strings.TrimSpace(req.Mode)), "port") != "port" {
		return
	}

	preferred := firstNonZero(req.HostPort, defaultHostPort(req.Env))
	if s.previewHostPortUsable(runtime, req.App, req.Env, req.Branch, preferred) {
		req.HostPort = preferred
		return
	}
	if !shouldAutoAssignPreviewHostPort(*req, state) {
		return
	}

	if chosen := s.firstAvailablePreviewHostPort(runtime, req.App, req.Env, req.Branch, preferred, 256); chosen > 0 {
		req.HostPort = chosen
		if chosen != preferred && log != nil {
			log("preview host port %d unavailable; using %d", preferred, chosen)
		}
	}
}

func (s *Server) firstAvailablePreviewHostPort(runtime ContainerRuntime, app string, env DeployEnv, branch string, start int, span int) int {
	if start <= 0 {
		return 0
	}
	if span <= 0 {
		span = 1
	}
	for offset := 0; offset < span; offset++ {
		port := start + offset
		if s.previewHostPortUsable(runtime, app, env, branch, port) {
			return port
		}
	}
	return 0
}

func (s *Server) previewHostPortUsable(runtime ContainerRuntime, app string, env DeployEnv, branch string, port int) bool {
	if port <= 0 {
		return false
	}
	if runtime != nil {
		containerName := appBaseContainerName(app, env, branch)
		if runtime.IsRunning(containerName) && runtime.PublishedPort(containerName, 3000) == port {
			return true
		}
	}
	if s.previewHostPortReservedByOtherApp(app, env, branch, port) {
		return false
	}
	return hostPortAvailable(port)
}

func (s *Server) previewHostPortReservedByOtherApp(app string, env DeployEnv, branch string, port int) bool {
	if s == nil || s.db == nil || defaultLanePolicy(env).Env == EnvProd || port <= 0 {
		return false
	}
	rows, err := s.db.Query(
		`SELECT app, env, branch, COALESCE(mode,''), host_port, COALESCE(public_host,'')
		FROM app_state
		WHERE env=? AND host_port=?`,
		string(env), port,
	)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var rowApp string
		var rowEnv string
		var rowBranch string
		var rowMode string
		var rowHostPort int
		var rowPublicHost string
		if err := rows.Scan(&rowApp, &rowEnv, &rowBranch, &rowMode, &rowHostPort, &rowPublicHost); err != nil {
			continue
		}
		if rowApp == app && rowEnv == string(env) && rowBranch == branch {
			continue
		}
		if firstNonEmpty(strings.ToLower(strings.TrimSpace(rowMode)), "port") != "port" {
			continue
		}
		if strings.TrimSpace(rowPublicHost) != "" {
			continue
		}
		return true
	}
	return false
}

func edgeProxyPublishedPortChanged(runtime ContainerRuntime, app string, env DeployEnv, branch string, hostPort int, mode string, publicHost string) bool {
	if runtime == nil {
		return false
	}
	containerName := appBaseContainerName(app, env, branch)
	if !runtime.IsRunning(containerName) {
		return false
	}
	published := runtime.PublishedPort(containerName, 3000)
	if firstNonEmpty(strings.ToLower(strings.TrimSpace(mode)), "port") == "port" {
		return published != firstNonZero(hostPort, defaultHostPort(env))
	}
	if strings.TrimSpace(publicHost) != "" {
		return published != firstNonZero(hostPort, defaultHostPort(env))
	}
	return published != 0
}

func (s *Server) requeueDeployLater(job DeployJob, delay time.Duration) {
	time.AfterFunc(delay, func() {
		select {
		case s.queue <- job:
		default:
			s.requeueDeployLater(job, delay)
		}
	})
}

func (s *Server) workspaceRepoDir(app string, env DeployEnv, branch string) string {
	return filepath.Join(s.workspacesDir, fmt.Sprintf("%s__%s__%s", safe(app), safe(string(env)), safe(branch)), "repo")
}

func appBaseContainerName(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("relay__%s__%s__%s", safe(app), safe(string(env)), safe(branch))
}

func appSlotContainerName(app string, env DeployEnv, branch string, slot string) string {
	return appBaseContainerName(app, env, branch) + "__" + normalizeActiveSlot(firstNonEmpty(slot, "blue"))
}

func appNetworkName(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("relay-%s-%s-%s", safe(app), safe(string(env)), safe(branch))
}

func normalizeActiveSlot(slot string) string {
	switch strings.ToLower(strings.TrimSpace(slot)) {
	case "blue":
		return "blue"
	case "green":
		return "green"
	default:
		return ""
	}
}

func normalizeTrafficMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "session":
		return "session"
	case "edge":
		return "edge"
	default:
		return ""
	}
}

func nextActiveSlot(slot string) string {
	if normalizeActiveSlot(slot) == "blue" {
		return "green"
	}
	return "blue"
}

func dockerPath(p string) string {
	return filepath.ToSlash(filepath.Clean(p))
}

func rolloutReadyTimeout() time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAY_ROLLOUT_READY_TIMEOUT_SECONDS")))
	if err != nil || secs <= 0 {
		secs = 60
	}
	return time.Duration(secs) * time.Second
}

func rolloutDrainDuration() time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAY_ROLLOUT_DRAIN_SECONDS")))
	if err != nil || secs < 0 {
		secs = 30
	}
	return time.Duration(secs) * time.Second
}

func (s *Server) waitForRuntimeContainerReady(runtime ContainerRuntime, log func(string, ...any), name string, port int, timeout time.Duration) error {
	port = firstNonZero(port, 3000)
	deadline := time.Now().Add(timeout)
	if log != nil {
		log("checking readiness for %s on port %d (timeout %s)", name, port, timeout)
	}
	for attempts := 0; time.Now().Before(deadline); attempts++ {
		if vr, ok := runtime.(*StationRuntime); ok {
			if vr.readyByLog(name) || vr.bridgeReadyStable(name, 8*time.Second) {
				return nil
			}
		}
		if runtime.IsRunning(name) {
			if hostPort := runtime.PublishedPort(name, port); hostPort > 0 {
				conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(hostPort)), 2*time.Second)
				if err == nil {
					_ = conn.Close()
					return nil
				}
			}
			if ip := runtime.ContainerIP(name); ip != "" {
				conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), 2*time.Second)
				if err == nil {
					_ = conn.Close()
					return nil
				}
				if vr, ok := runtime.(*StationRuntime); ok {
					if vr.probeBridgeAddress(ip, port) {
						return nil
					}
				}
			}
		}
		if log != nil && attempts > 0 && attempts%10 == 0 {
			log("waiting for %s to accept traffic on port %d", name, port)
		}
		time.Sleep(500 * time.Millisecond)
	}
	// Timed out — collect diagnostics so the deploy log shows what happened.
	if log != nil {
		if _, ok := runtime.(*StationRuntime); ok {
			if rec, _ := latestStationContainerByApp(name); rec != nil {
				log("container %s: pid=%d running=%v port=%d", name, rec.PID, stationContainerRunning(rec), rec.Port)
			} else {
				log("container %s: no record found in state dir", name)
			}
			if path := stationRuntimeLogPath(name); path != "" {
				if data, readErr := os.ReadFile(path); readErr == nil {
					if len(data) == 0 {
						log("container log is empty")
					} else {
						if len(data) > 4096 {
							data = data[len(data)-4096:]
						}
						log("container output:\n%s", strings.TrimSpace(string(data)))
					}
				} else {
					log("container log unreadable: %v", readErr)
				}
			} else {
				log("container log not found (no state record?)")
			}
		}
	}
	return fmt.Errorf("container %s did not become ready on port %d within %s", name, port, timeout)
}

func (s *Server) waitForContainerReady(log func(string, ...any), name string, port int, timeout time.Duration) error {
	return s.waitForRuntimeContainerReady(s.runtime, log, name, port, timeout)
}

func (s *Server) edgeProxyConfigPath(app string, env DeployEnv, branch string) string {
	dir := filepath.Join(s.dataDir, "edge-proxy")
	mustMkdir(dir)
	return filepath.Join(dir, fmt.Sprintf("%s__%s__%s.nginx.conf", safe(app), safe(string(env)), safe(branch)))
}

func edgeCookieName(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("relay_slot_%s_%s_%s", safe(app), safe(string(env)), safe(branch))
}

func (s *Server) writeEdgeProxyConfig(app string, env DeployEnv, branch string, activeSlot string, standbySlot string, servicePort int, trafficMode string) (string, error) {
	p := s.edgeProxyConfigPath(app, env, branch)
	activeUpstream := fmt.Sprintf("%s:%d", appSlotContainerName(app, env, branch, activeSlot), firstNonZero(servicePort, 3000))
	standbyMode := normalizeActiveSlot(standbySlot)
	trafficMode = firstNonEmpty(normalizeTrafficMode(trafficMode), "edge")
	relayPort := listenAddrPort(s.httpAddr)
	authURL := ""
	if relayPort > 0 {
		authURL = edgeAuthProxyURL(relayPort, app, env, branch, "host.docker.internal")
	}
	var conf strings.Builder
	conf.WriteString("worker_processes auto;\n")
	conf.WriteString("events { worker_connections 1024; }\n")
	conf.WriteString("http {\n")
	conf.WriteString("  map $http_upgrade $connection_upgrade {\n")
	conf.WriteString("    default upgrade;\n")
	conf.WriteString("    '' close;\n")
	conf.WriteString("  }\n")
	conf.WriteString("  resolver 127.0.0.11 ipv6=off valid=5s;\n")
	conf.WriteString("  map $arg___relay_target $relay_slot_override {\n")
	conf.WriteString("    default \"\";\n")
	conf.WriteString(fmt.Sprintf("    ~*^(new|live|active|%s)$ %s;\n", activeSlot, activeSlot))
	if standbyMode != "" {
		conf.WriteString(fmt.Sprintf("    ~*^(old|previous|standby|draining|%s)$ %s;\n", standbyMode, standbyMode))
	}
	conf.WriteString("  }\n")
	if trafficMode == "session" {
		cookieName := edgeCookieName(app, env, branch)
		conf.WriteString(fmt.Sprintf("  map $cookie_%s $relay_slot_cookie {\n", cookieName))
		conf.WriteString(fmt.Sprintf("    default %s;\n", activeSlot))
		conf.WriteString(fmt.Sprintf("    ~^%s$ %s;\n", activeSlot, activeSlot))
		if standbyMode != "" {
			conf.WriteString(fmt.Sprintf("    ~^%s$ %s;\n", standbyMode, standbyMode))
		}
		conf.WriteString("  }\n")
		conf.WriteString("  map $relay_slot_override $relay_target_slot {\n")
		conf.WriteString("    default $relay_slot_cookie;\n")
		conf.WriteString(fmt.Sprintf("    ~^%s$ %s;\n", activeSlot, activeSlot))
		if standbyMode != "" {
			conf.WriteString(fmt.Sprintf("    ~^%s$ %s;\n", standbyMode, standbyMode))
		}
		conf.WriteString("  }\n")
	} else {
		conf.WriteString("  map $relay_slot_override $relay_target_slot {\n")
		conf.WriteString(fmt.Sprintf("    default %s;\n", activeSlot))
		conf.WriteString(fmt.Sprintf("    ~^%s$ %s;\n", activeSlot, activeSlot))
		if standbyMode != "" {
			conf.WriteString(fmt.Sprintf("    ~^%s$ %s;\n", standbyMode, standbyMode))
		}
		conf.WriteString("  }\n")
	}
	conf.WriteString(fmt.Sprintf("  map $relay_target_slot $relay_upstream {\n    default %s;\n", activeUpstream))
	conf.WriteString(fmt.Sprintf("    ~^%s$ %s;\n", activeSlot, activeUpstream))
	if standbyMode != "" {
		conf.WriteString(fmt.Sprintf("    ~^%s$ %s:%d;\n", standbyMode, appSlotContainerName(app, env, branch, standbyMode), firstNonZero(servicePort, 3000)))
	}
	conf.WriteString("  }\n")
	conf.WriteString("  server {\n")
	conf.WriteString("    listen 3000;\n")
	conf.WriteString("    location = /__relay/health {\n")
	conf.WriteString("      access_log off;\n")
	conf.WriteString("      add_header Content-Type text/plain;\n")
	conf.WriteString("      return 200 'ok';\n")
	conf.WriteString("    }\n")
	if authURL != "" {
		conf.WriteString("    location = /__relay/authz {\n")
		conf.WriteString("      internal;\n")
		conf.WriteString("      proxy_pass_request_body off;\n")
		conf.WriteString("      proxy_set_header Content-Length \"\";\n")
		conf.WriteString("      proxy_set_header Cookie $http_cookie;\n")
		conf.WriteString("      proxy_set_header Authorization $http_authorization;\n")
		conf.WriteString("      proxy_set_header X-Relay-Token $http_x_relay_token;\n")
		conf.WriteString("      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		conf.WriteString("      proxy_set_header X-Forwarded-Host $host;\n")
		conf.WriteString("      proxy_set_header X-Original-Uri $request_uri;\n")
		conf.WriteString("      proxy_pass " + authURL + ";\n")
		conf.WriteString("    }\n")
		conf.WriteString("    location @relay_unauthorized {\n")
		conf.WriteString("      add_header Content-Type text/plain always;\n")
		conf.WriteString("      return 401 'relay login required';\n")
		conf.WriteString("    }\n")
		conf.WriteString("    location @relay_forbidden {\n")
		conf.WriteString("      add_header Content-Type text/plain always;\n")
		conf.WriteString("      return 403 'request blocked';\n")
		conf.WriteString("    }\n")
	}
	conf.WriteString("    location / {\n")
	if authURL != "" {
		conf.WriteString("      auth_request /__relay/authz;\n")
		conf.WriteString("      error_page 401 = @relay_unauthorized;\n")
		conf.WriteString("      error_page 403 = @relay_forbidden;\n")
	}
	conf.WriteString("      proxy_http_version 1.1;\n")
	conf.WriteString("      proxy_set_header Host $host;\n")
	conf.WriteString("      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	conf.WriteString("      proxy_set_header X-Forwarded-Proto $scheme;\n")
	conf.WriteString("      proxy_set_header Upgrade $http_upgrade;\n")
	conf.WriteString("      proxy_set_header Connection $connection_upgrade;\n")
	conf.WriteString("      proxy_read_timeout 300s;\n")
	conf.WriteString("      proxy_connect_timeout 5s;\n")
	conf.WriteString("      add_header X-Relay-Target $relay_target_slot always;\n")
	conf.WriteString(fmt.Sprintf("      add_header X-Relay-Traffic-Mode \"%s\" always;\n", trafficMode))
	if trafficMode == "session" {
		conf.WriteString(fmt.Sprintf("      add_header Set-Cookie \"%s=$relay_target_slot; Path=/; Max-Age=86400; SameSite=Lax\" always;\n", edgeCookieName(app, env, branch)))
	}
	conf.WriteString("      proxy_pass http://$relay_upstream;\n")
	conf.WriteString("    }\n")
	conf.WriteString("  }\n")
	conf.WriteString("}\n")
	if err := os.WriteFile(p, []byte(conf.String()), 0644); err != nil {
		return "", err
	}
	return p, nil
}

func (s *Server) runSlotContainerWithRuntime(runtime ContainerRuntime, log func(string, ...any), app string, env DeployEnv, branch string, slot string, image string, servicePort int, networkName string, extraEnv map[string]string) error {
	containerName := appSlotContainerName(app, env, branch, slot)
	runtime.Remove(containerName)

	secs, _ := s.getAppSecrets(app, env, branch)
	allEnv := make(map[string]string)
	for k, v := range extraEnv {
		allEnv[k] = v
	}
	for k, v := range secs {
		allEnv[k] = v
	}

	keys := make([]string, 0, len(allEnv))
	for k := range allEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	envPairs := []string{fmt.Sprintf("PORT=%d", firstNonZero(servicePort, 3000))}
	for _, k := range keys {
		envPairs = append(envPairs, fmt.Sprintf("%s=%s", k, allEnv[k]))
	}

	spec := ContainerSpec{
		Name:          containerName,
		Image:         image,
		Network:       networkName,
		RestartPolicy: "always",
		Env:           envPairs,
		ExtraHosts:    s.serviceHostAliasesForRuntime(runtime, app, env, branch),
		PortBindings:  []string{fmt.Sprintf("127.0.0.1::%d", firstNonZero(servicePort, 3000))},
	}
	if log != nil {
		log("runtime run candidate: %s on %s", containerName, networkName)
	}
	t0 := time.Now()
	err := runtime.RunDetached(spec)
	if log != nil {
		log("station run completed in %s", time.Since(t0).Round(time.Millisecond))
	}
	return err
}

func (s *Server) runSlotContainer(log func(string, ...any), app string, env DeployEnv, branch string, slot string, image string, servicePort int, networkName string, extraEnv map[string]string) error {
	return s.runSlotContainerWithRuntime(s.runtime, log, app, env, branch, slot, image, servicePort, networkName, extraEnv)
}

func (s *Server) ensureEdgeProxy(log func(string, ...any), app string, env DeployEnv, branch string, networkName string, activeSlot string, standbySlot string, servicePort int, hostPort int, mode string, trafficMode string, publicHost string, recreate bool) error {
	activeSlot = normalizeActiveSlot(activeSlot)
	standbySlot = normalizeActiveSlot(standbySlot)
	if standbySlot != "" && !s.runtime.IsRunning(appSlotContainerName(app, env, branch, standbySlot)) {
		if log != nil {
			log("standby slot %s is no longer running; clearing stale old target", standbySlot)
		}
		standbySlot = ""
		if st, err := s.getAppState(app, env, branch); err == nil && st != nil {
			if normalizeActiveSlot(st.ActiveSlot) == activeSlot {
				st.StandbySlot = ""
				st.DrainUntil = 0
				_ = s.saveAppState(st)
				s.broadcastSnapshot()
			}
		}
	}
	configPath, err := s.writeEdgeProxyConfig(app, env, branch, activeSlot, standbySlot, servicePort, trafficMode)
	if err != nil {
		return fmt.Errorf("write edge proxy config: %w", err)
	}

	containerName := appBaseContainerName(app, env, branch)
	if recreate || !s.runtime.IsRunning(containerName) {
		s.runtime.Remove(containerName)
		portBindings := []string{}
		normMode := strings.ToLower(strings.TrimSpace(mode))
		if normMode == "port" {
			portBindings = []string{fmt.Sprintf("%d:3000", hostPort)}
		} else if strings.TrimSpace(publicHost) != "" {
			portBindings = []string{fmt.Sprintf("127.0.0.1:%d:3000", hostPort)}
		}
		spec := ContainerSpec{
			Name:          containerName,
			Image:         getenv("RELAY_NGINX_IMAGE", "nginx:alpine"),
			Network:       networkName,
			RestartPolicy: "always",
			Volumes:       []string{fmt.Sprintf("%s:/etc/nginx/nginx.conf:ro", dockerPath(configPath))},
			PortBindings:  portBindings,
			ExtraHosts:    []string{"host.docker.internal:host-gateway"},
		}
		if log != nil {
			log("runtime run edge: %s", containerName)
		}
		if runErr := s.runtime.RunDetached(spec); runErr != nil {
			return fmt.Errorf("edge proxy start failed: %w", runErr)
		}
		return s.waitForContainerReady(log, containerName, 3000, 15*time.Second)
	}

	if log != nil {
		log("reloading edge proxy %s -> active=%s standby=%s traffic=%s", containerName, appSlotContainerName(app, env, branch, activeSlot), normalizeActiveSlot(standbySlot), firstNonEmpty(normalizeTrafficMode(trafficMode), "edge"))
	}
	out, reloadErr := s.runtime.Exec(containerName, []string{"nginx", "-s", "reload"})
	if reloadErr != nil {
		return fmt.Errorf("edge proxy reload failed: %v (%s)", reloadErr, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Server) cleanupStandbySlotAfter(app string, env DeployEnv, branch string, activeSlot string, oldSlot string, servicePort int, hostPort int, mode string, trafficMode string, publicHost string, wait time.Duration) {
	name := appSlotContainerName(app, env, branch, oldSlot)
	if wait <= 0 {
		s.runtime.Remove(name)
		_ = s.ensureEdgeProxy(nil, app, env, branch, appNetworkName(app, env, branch), activeSlot, "", servicePort, hostPort, mode, trafficMode, publicHost, false)
		if st, err := s.getAppState(app, env, branch); err == nil && st != nil {
			if normalizeActiveSlot(st.ActiveSlot) == normalizeActiveSlot(activeSlot) && normalizeActiveSlot(st.StandbySlot) == normalizeActiveSlot(oldSlot) {
				st.StandbySlot = ""
				st.DrainUntil = 0
				_ = s.saveAppState(st)
				s.broadcastSnapshot()
			}
		}
		return
	}
	go func() {
		time.Sleep(wait)
		s.runtime.Remove(name)
		_ = s.ensureEdgeProxy(nil, app, env, branch, appNetworkName(app, env, branch), activeSlot, "", servicePort, hostPort, mode, trafficMode, publicHost, false)
		if st, err := s.getAppState(app, env, branch); err == nil && st != nil {
			if normalizeActiveSlot(st.ActiveSlot) == normalizeActiveSlot(activeSlot) && normalizeActiveSlot(st.StandbySlot) == normalizeActiveSlot(oldSlot) {
				st.StandbySlot = ""
				st.DrainUntil = 0
				_ = s.saveAppState(st)
				s.broadcastSnapshot()
			}
		}
	}()
}

// ensureGlobalProxy maintains a Caddy container that routes public_host domains
// to their respective per-app edge proxy ports, handling TLS automatically.
// It is called (in a goroutine) after any deploy or app config change.
// startACMEListener binds a lightweight HTTP server on RELAY_ACME_ADDR (default
// ":80") that:
//   - serves /.well-known/acme-challenge/* from the acme-webroot directory so
//     that external ACME clients (certbot, acme.sh) can complete HTTP-01
//     challenges without needing nginx or any other web server;
//   - redirects all other HTTP traffic to the same URL over HTTPS.
//
// If the address is already in use (e.g. Caddy is running with -p 80:80 via
// Docker), the bind silently fails and Caddy handles challenges itself.
// ─────────────────────────────────────────────────────────────────────────────
// Analytics: Caddy access-log tailer + IP geolocation + query handler
// ─────────────────────────────────────────────────────────────────────────────

// caddyLogEntry is the subset of Caddy's JSON access-log format we care about.
type caddyLogEntry struct {
	Ts      float64 `json:"ts"`
	Request struct {
		RemoteIP string `json:"remote_ip"`
		Method   string `json:"method"`
		Host     string `json:"host"`
		URI      string `json:"uri"`
	} `json:"request"`
	Status int   `json:"status"`
	Size   int64 `json:"size"`
}

// startLogTailer reads new lines appended to the Caddy access log, inserts
// analytics events into SQLite, and resolves country codes in the background.
func (s *Server) startLogTailer() {
	logPath := filepath.Join(s.caddyLogsDir, "access.log")

	// Persist the byte offset between restarts so we don't re-process old events.
	getOffset := func() int64 {
		var v string
		_ = s.db.QueryRow(`SELECT value FROM server_config WHERE key='analytics_log_offset'`).Scan(&v)
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	saveOffset := func(off int64) {
		_, _ = s.db.Exec(`INSERT OR REPLACE INTO server_config(key,value) VALUES('analytics_log_offset',?)`, strconv.FormatInt(off, 10))
	}

	offset := getOffset()

	for {
		f, err := os.Open(logPath)
		if err != nil {
			// File doesn't exist yet (Caddy not started). Wait and retry.
			time.Sleep(5 * time.Second)
			continue
		}

		fi, _ := f.Stat()
		// If the file shrank (roll / truncation), reset to beginning.
		if offset > fi.Size() {
			offset = 0
			saveOffset(0)
		}
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			f.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(f)
		var newEvents []caddyLogEntry
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			var entry caddyLogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			// Skip internal Caddy admin / health check lines.
			if entry.Ts == 0 || entry.Request.Host == "" {
				continue
			}
			newEvents = append(newEvents, entry)
		}

		newOffset, _ := f.Seek(0, io.SeekCurrent)
		f.Close()

		if len(newEvents) > 0 {
			s.insertAnalyticsEvents(newEvents)
			saveOffset(newOffset)
			offset = newOffset
			// Kick off a non-blocking country resolver pass.
			go s.resolveAnalyticsCountries()
		} else {
			offset = newOffset
		}

		time.Sleep(2 * time.Second)
	}
}

func (s *Server) insertAnalyticsEvents(entries []caddyLogEntry) {
	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare(`INSERT INTO analytics_events(ts,host,method,path,status,bytes,remote_ip) VALUES(?,?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()
	for _, e := range entries {
		ts := int64(e.Ts)
		uri := e.Request.URI
		if len(uri) > 256 {
			uri = uri[:256]
		}
		if _, err := stmt.Exec(ts, e.Request.Host, e.Request.Method, uri, e.Status, e.Size, e.Request.RemoteIP); err != nil {
			continue
		}
	}
	tx.Commit()
}

// ipAPIBatchResponse is one item from the ip-api.com /batch response.
type ipAPIBatchResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Query       string `json:"query"`
}

// resolveAnalyticsCountries batch-resolves remote IPs to country codes using
// ip-api.com (free, no key, 45 req/min).  Results are cached in ip_country_cache.
func (s *Server) resolveAnalyticsCountries() {
	// Collect distinct IPs that have no country assigned yet.
	rows, err := s.db.Query(`
		SELECT DISTINCT remote_ip FROM analytics_events
		WHERE country_code='' AND remote_ip!=''
		LIMIT 500`)
	if err != nil {
		return
	}
	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err == nil && ip != "" {
			ips = append(ips, ip)
		}
	}
	rows.Close()
	if len(ips) == 0 {
		return
	}

	// Filter out already-cached IPs.
	uncached := ips[:0]
	for _, ip := range ips {
		var code string
		_ = s.db.QueryRow(`SELECT country_code FROM ip_country_cache WHERE ip=?`, ip).Scan(&code)
		if code == "" {
			uncached = append(uncached, ip)
		}
	}

	// Process in batches of up to 100 (ip-api.com limit).
	for i := 0; i < len(uncached); i += 100 {
		end := i + 100
		if end > len(uncached) {
			end = len(uncached)
		}
		batch := uncached[i:end]

		type req struct {
			Query  string `json:"query"`
			Fields string `json:"fields"`
		}
		reqs := make([]req, len(batch))
		for j, ip := range batch {
			reqs[j] = req{Query: ip, Fields: "status,country,countryCode,query"}
		}
		body, _ := json.Marshal(reqs)
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Post(
			"http://ip-api.com/batch", "application/json", bytes.NewReader(body))
		if err != nil {
			continue
		}
		var results []ipAPIBatchResponse
		_ = json.NewDecoder(resp.Body).Decode(&results)
		resp.Body.Close()

		now := time.Now().Unix()
		for _, r := range results {
			if r.Status != "success" || r.Query == "" {
				continue
			}
			_, _ = s.db.Exec(
				`INSERT OR REPLACE INTO ip_country_cache(ip,country_code,country_name,updated_at) VALUES(?,?,?,?)`,
				r.Query, r.CountryCode, r.Country, now)
		}
		// Small back-off to stay under the 45 req/min rate limit.
		if i+100 < len(uncached) {
			time.Sleep(1500 * time.Millisecond)
		}
	}

	// Apply cached countries to analytics_events rows that are still empty.
	_, _ = s.db.Exec(`
		UPDATE analytics_events
		SET country_code = (SELECT country_code FROM ip_country_cache WHERE ip=remote_ip),
		    country_name = (SELECT country_name FROM ip_country_cache WHERE ip=remote_ip)
		WHERE country_code='' AND remote_ip!=''
		  AND EXISTS (SELECT 1 FROM ip_country_cache WHERE ip=remote_ip)`)
}

type analyticsResponse struct {
	TotalRequests int64              `json:"total_requests"`
	PeriodLabel   string             `json:"period"`
	ByCountry     []analyticsCountry `json:"by_country"`
	ByStatus      []analyticsStatus  `json:"by_status"`
	ByHour        []analyticsHour    `json:"by_hour"`
	ByHost        []analyticsHost    `json:"by_host"`
}
type analyticsCountry struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
type analyticsStatus struct {
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}
type analyticsHour struct {
	Ts    int64 `json:"ts"`
	Count int64 `json:"count"`
}
type analyticsHost struct {
	Host  string `json:"host"`
	Count int64  `json:"count"`
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	app := strings.TrimSpace(r.URL.Query().Get("app"))

	var since int64
	var periodLabel string
	now := time.Now().Unix()
	switch period {
	case "24h":
		since = now - 86400
		periodLabel = "24h"
	case "30d":
		since = now - 30*86400
		periodLabel = "30d"
	default:
		since = now - 7*86400
		periodLabel = "7d"
	}

	hostFilter := ""
	hostArgs := []any{since}
	if app != "" {
		// Match events where the host belongs to the given app.
		hostFilter = ` AND host IN (SELECT public_host FROM app_state WHERE app=? AND public_host!='')`
		hostArgs = append(hostArgs, app)
	}

	// Total requests.
	var total int64
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM analytics_events WHERE ts>=?`+hostFilter, hostArgs...).Scan(&total)

	// By country.
	countryRows, _ := s.db.Query(
		`SELECT COALESCE(NULLIF(country_code,''),'??') AS cc,
		        COALESCE(NULLIF(country_name,''),'Unknown') AS cn,
		        COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY cc ORDER BY cnt DESC LIMIT 30`,
		hostArgs...)
	var byCountry []analyticsCountry
	if countryRows != nil {
		for countryRows.Next() {
			var c analyticsCountry
			_ = countryRows.Scan(&c.Code, &c.Name, &c.Count)
			byCountry = append(byCountry, c)
		}
		countryRows.Close()
	}

	// By status class (group into 2xx, 3xx, 4xx, 5xx).
	statusRows, _ := s.db.Query(
		`SELECT (status/100)*100 AS sc, COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY sc ORDER BY sc`,
		hostArgs...)
	var byStatus []analyticsStatus
	if statusRows != nil {
		for statusRows.Next() {
			var st analyticsStatus
			_ = statusRows.Scan(&st.Status, &st.Count)
			byStatus = append(byStatus, st)
		}
		statusRows.Close()
	}

	// By hour (bucket ts to nearest hour).
	bucketSize := int64(3600)
	if period == "30d" {
		bucketSize = 86400 // daily buckets for 30-day view
	}
	hourRows, _ := s.db.Query(
		`SELECT (ts/?)*? AS bucket, COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY bucket ORDER BY bucket`,
		append([]any{bucketSize, bucketSize}, hostArgs...)...)
	var byHour []analyticsHour
	if hourRows != nil {
		for hourRows.Next() {
			var h analyticsHour
			_ = hourRows.Scan(&h.Ts, &h.Count)
			byHour = append(byHour, h)
		}
		hourRows.Close()
	}

	// By host.
	hostRows, _ := s.db.Query(
		`SELECT host, COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY host ORDER BY cnt DESC LIMIT 20`,
		hostArgs...)
	var byHost []analyticsHost
	if hostRows != nil {
		for hostRows.Next() {
			var h analyticsHost
			_ = hostRows.Scan(&h.Host, &h.Count)
			byHost = append(byHost, h)
		}
		hostRows.Close()
	}

	resp := analyticsResponse{
		TotalRequests: total,
		PeriodLabel:   periodLabel,
		ByCountry:     byCountry,
		ByStatus:      byStatus,
		ByHour:        byHour,
		ByHost:        byHost,
	}
	if resp.ByCountry == nil {
		resp.ByCountry = []analyticsCountry{}
	}
	if resp.ByStatus == nil {
		resp.ByStatus = []analyticsStatus{}
	}
	if resp.ByHour == nil {
		resp.ByHour = []analyticsHour{}
	}
	if resp.ByHost == nil {
		resp.ByHost = []analyticsHost{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) startACMEListener() {
	acmeAddr := getenv("RELAY_ACME_ADDR", ":80")
	ln, err := net.Listen("tcp", acmeAddr)
	if err != nil {
		// Port already claimed (e.g. by Caddy's Docker port binding) — that's
		// fine; Caddy will handle ACME challenges on its own.
		fmt.Fprintf(os.Stderr, "info: ACME HTTP listener could not bind %s (%v); challenges will be served by Caddy when Docker is available\n", acmeAddr, err)
		return
	}

	acmeMux := http.NewServeMux()
	// Serve challenge tokens from the local webroot directory.
	acmeMux.Handle("/.well-known/acme-challenge/",
		http.StripPrefix("/.well-known/acme-challenge/",
			http.FileServer(http.Dir(s.acmeWebroot))))
	// Redirect all other plain-HTTP traffic to HTTPS.
	acmeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})

	srv := &http.Server{
		Handler:           acmeMux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Println("ACME HTTP listener on", acmeAddr)
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "ACME listener error: %v\n", err)
		}
	}()
}

func (s *Server) ensureGlobalProxy() error {
	rows, err := s.db.Query(`SELECT public_host, COALESCE(host_port, 0) FROM app_state WHERE public_host != '' AND COALESCE(stopped,0)=0`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type route struct {
		host     string
		hostPort int
	}
	var routes []route
	for rows.Next() {
		var r route
		if err := rows.Scan(&r.host, &r.hostPort); err != nil {
			continue
		}
		if r.hostPort <= 0 || strings.TrimSpace(r.host) == "" {
			continue
		}
		routes = append(routes, r)
	}
	if dashboardHost := strings.TrimSpace(s.serverDashboardHost()); dashboardHost != "" {
		if relayPort := listenAddrPort(s.httpAddr); relayPort > 0 {
			routes = append(routes, route{host: dashboardHost, hostPort: relayPort})
		}
	}

	containerName := "relay-global-proxy"
	if len(routes) == 0 {
		s.runtime.Remove(containerName)
		return nil
	}

	configDir := filepath.Join(s.dataDir, "global-proxy")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	dataPath := filepath.Join(configDir, "data")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "Caddyfile")

	var cf strings.Builder
	// Always emit a global options block so we can configure access logging
	// (and optionally the ACME email for Let's Encrypt account registration).
	cf.WriteString("{\n")
	if email := strings.TrimSpace(os.Getenv("RELAY_ACME_EMAIL")); email != "" {
		cf.WriteString(fmt.Sprintf("\temail %s\n", email))
	}
	cf.WriteString("\tlog {\n")
	cf.WriteString("\t\toutput file /logs/access.log {\n")
	cf.WriteString("\t\t\troll_size 50mb\n")
	cf.WriteString("\t\t\troll_keep 3\n")
	cf.WriteString("\t\t}\n")
	cf.WriteString("\t\tformat json\n")
	cf.WriteString("\t}\n")
	cf.WriteString("}\n\n")
	for _, r := range routes {
		cf.WriteString(strings.TrimSpace(r.host) + " {\n")
		cf.WriteString(fmt.Sprintf("\treverse_proxy host.docker.internal:%d\n", r.hostPort))
		cf.WriteString("}\n\n")
	}
	if err := os.WriteFile(configPath, []byte(cf.String()), 0644); err != nil {
		return err
	}

	if !s.runtime.IsRunning(containerName) {
		s.runtime.Remove(containerName)
		image := getenv("RELAY_CADDY_IMAGE", "caddy:alpine")
		// Ensure the image is available before attempting to run
		_ = s.runtime.Pull(image)
		spec := ContainerSpec{
			Name:          containerName,
			Image:         image,
			RestartPolicy: "always",
			Volumes: []string{
				fmt.Sprintf("%s:/etc/caddy/Caddyfile:ro", dockerPath(configPath)),
				fmt.Sprintf("%s:/data", dockerPath(dataPath)),
				fmt.Sprintf("%s:/logs", dockerPath(s.caddyLogsDir)),
			},
			PortBindings: []string{"80:80", "443:443", "443:443/udp"},
			ExtraHosts:   []string{"host.docker.internal:host-gateway"},
		}
		if err := s.runtime.RunDetached(spec); err != nil {
			return fmt.Errorf("global proxy start: %w", err)
		}
		return nil
	}

	out, reloadErr := s.runtime.Exec(containerName, []string{"caddy", "reload", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"})
	if reloadErr != nil {
		return fmt.Errorf("global proxy reload: %v (%s)", reloadErr, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Server) currentActiveSlotWithRuntime(runtime ContainerRuntime, app string, env DeployEnv, branch string, state *AppState) string {
	if state != nil && normalizeActiveSlot(state.ActiveSlot) != "" {
		return normalizeActiveSlot(state.ActiveSlot)
	}
	for _, slot := range []string{"blue", "green"} {
		if runtime.IsRunning(appSlotContainerName(app, env, branch, slot)) {
			return slot
		}
	}
	return ""
}

func (s *Server) currentActiveSlot(app string, env DeployEnv, branch string, state *AppState) string {
	return s.currentActiveSlotWithRuntime(s.runtime, app, env, branch, state)
}

func (s *Server) swapContainer(log func(string, ...any), req DeployRequest, imageTag string, networkName string, extraEnv map[string]string) error {
	servicePort := firstNonZero(req.ServicePort, 3000)
	hostPort := firstNonZero(req.HostPort, defaultHostPort(req.Env))
	mode := firstNonEmpty(strings.ToLower(strings.TrimSpace(req.Mode)), "port")
	trafficMode := firstNonEmpty(normalizeTrafficMode(req.TrafficMode), "edge")
	if networkName == "" {
		networkName = appNetworkName(req.App, req.Env, req.Branch)
	}
	if err := s.runtime.EnsureNetwork(networkName); err != nil {
		return fmt.Errorf("ensure app network: %w", err)
	}

	state, _ := s.getAppState(req.App, req.Env, req.Branch)
	activeSlot := s.currentActiveSlot(req.App, req.Env, req.Branch, state)
	nextSlot := nextActiveSlot(activeSlot)
	candidateName := appSlotContainerName(req.App, req.Env, req.Branch, nextSlot)

	if err := s.runSlotContainer(log, req.App, req.Env, req.Branch, nextSlot, imageTag, servicePort, networkName, extraEnv); err != nil {
		return err
	}
	if err := s.waitForContainerReady(log, candidateName, servicePort, rolloutReadyTimeout()); err != nil {
		s.runtime.Remove(candidateName)
		return err
	}

	recreateEdge := !s.runtime.IsRunning(appBaseContainerName(req.App, req.Env, req.Branch)) || activeSlot == ""
	if state != nil {
		prevMode := firstNonEmpty(strings.ToLower(strings.TrimSpace(state.Mode)), "port")
		prevTrafficMode := firstNonEmpty(normalizeTrafficMode(state.TrafficMode), "edge")
		prevHostPort := firstNonZero(state.HostPort, defaultHostPort(req.Env))
		if prevMode != mode || prevHostPort != hostPort || prevTrafficMode != trafficMode || state.PublicHost != req.PublicHost {
			recreateEdge = true
		}
	}
	if !recreateEdge && edgeProxyPublishedPortChanged(s.runtime, req.App, req.Env, req.Branch, hostPort, mode, req.PublicHost) {
		recreateEdge = true
	}
	if err := s.ensureEdgeProxy(log, req.App, req.Env, req.Branch, networkName, nextSlot, activeSlot, servicePort, hostPort, mode, trafficMode, req.PublicHost, recreateEdge); err != nil {
		if log != nil {
			log("edge proxy failed: %v", err)
		}
		s.runtime.Remove(candidateName)
		return err
	}

	if activeSlot != "" && activeSlot != nextSlot {
		drainUntil := time.Now().Add(rolloutDrainDuration()).UnixMilli()
		oldName := appSlotContainerName(req.App, req.Env, req.Branch, activeSlot)
		if log != nil {
			log("draining previous slot %s for %s", oldName, rolloutDrainDuration())
		}
		if state != nil {
			state.ActiveSlot = nextSlot
			state.StandbySlot = activeSlot
			state.DrainUntil = drainUntil
			state.TrafficMode = trafficMode
			_ = s.saveAppState(state)
			s.broadcastSnapshot()
		}
		s.cleanupStandbySlotAfter(req.App, req.Env, req.Branch, nextSlot, activeSlot, servicePort, hostPort, mode, trafficMode, req.PublicHost, rolloutDrainDuration())
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

func (s *Server) runContainer(log func(string, ...any), app string, env DeployEnv, branch string, image string, servicePort int, hostPort int, mode string, trafficMode string, networkName string, extraEnv map[string]string) error {
	return s.swapContainer(log, DeployRequest{
		App:              app,
		Branch:           branch,
		Env:              env,
		ServicePort:      servicePort,
		HostPort:         hostPort,
		HostPortExplicit: false,
		Mode:             mode,
		TrafficMode:      trafficMode,
	}, image, networkName, extraEnv)
}

func runCmdLogged(dir string, logw io.Writer, name string, args ...string) error {
	return runCmdLoggedEnv(dir, logw, nil, name, args...)
}

func runCmdLoggedEnv(dir string, logw io.Writer, extraEnv []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = logw
	cmd.Stderr = logw
	return cmd.Run()
}

func runCmdLoggedCtx(ctx context.Context, dir string, logw io.Writer, name string, args ...string) error {
	return runCmdLoggedEnvCtx(ctx, dir, logw, nil, name, args...)
}

func runCmdLoggedEnvCtx(ctx context.Context, dir string, logw io.Writer, extraEnv []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = logw
	cmd.Stderr = logw
	return cmd.Run()
}

// ---------------------- Auth + CORS ----------------------

func (s *Server) validateToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.apiToken)) == 1
}

func (s *Server) requestToken(r *http.Request) (string, string) {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:]), "bearer"
	}
	if token := strings.TrimSpace(r.Header.Get("X-Relay-Token")); token != "" {
		return token, "header"
	}
	if c, err := r.Cookie(dashboardSessionCookie); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value), "cookie"
	}
	if r.Method == http.MethodGet && (strings.HasPrefix(r.URL.Path, "/api/logs/stream/") || strings.HasPrefix(r.URL.Path, "/api/runtime/logs/stream")) {
		if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
			return token, "query"
		}
	}
	return "", ""
}

func isHTTPSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func (s *Server) setDashboardSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	cookie := &http.Cookie{
		Name:     dashboardSessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPSRequest(r),
		MaxAge:   60 * 60 * 12,
	}
	cookie.Domain = s.sessionCookieDomainForRequest(r)
	http.SetCookie(w, cookie)
}

func (s *Server) clearDashboardSessionCookie(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     dashboardSessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPSRequest(r),
		MaxAge:   -1,
	}
	cookie.Domain = s.sessionCookieDomainForRequest(r)
	http.SetCookie(w, cookie)
}

func isSameOriginRequest(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := r.Host
	if fh := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); fh != "" {
		host = fh
	}
	return strings.EqualFold(u.Host, host)
}

func normalizedRequestHost(r *http.Request) string {
	host := strings.TrimSpace(r.Host)
	if fh := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); fh != "" {
		host = fh
	}
	host = strings.TrimSpace(strings.Split(host, ",")[0])
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	return strings.ToLower(host)
}

func normalizedHostname(value string) string {
	host := strings.TrimSpace(value)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	return strings.ToLower(host)
}

func validateProxyHostname(value string, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.Contains(value, "://") {
		return fmt.Errorf("%s must be a hostname only, without a URL scheme", field)
	}
	if strings.ContainsAny(value, "/\\{}[]()\"'`\r\n\t ") {
		return fmt.Errorf("%s must be a plain hostname", field)
	}
	if strings.Contains(value, ":") {
		return fmt.Errorf("%s must not include a port", field)
	}
	host := normalizedHostname(value)
	if host == "" {
		return fmt.Errorf("%s must be a valid hostname", field)
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 {
			return fmt.Errorf("%s must be a valid hostname", field)
		}
		for i, ch := range label {
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
				continue
			}
			if ch == '-' && i > 0 && i < len(label)-1 {
				continue
			}
			return fmt.Errorf("%s must be a valid hostname", field)
		}
	}
	return nil
}

func requiresSameOrigin(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func parseAllowedOrigins(raw string) (map[string]struct{}, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]struct{}{}, false
	}
	origins := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == "*" {
			return map[string]struct{}{}, true
		}
		u, err := url.Parse(part)
		if err != nil || u.Scheme == "" || u.Host == "" {
			continue
		}
		origins[strings.ToLower(u.Scheme+"://"+u.Host)] = struct{}{}
	}
	return origins, false
}

func getenvBool(k string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func (s *Server) isOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if s.allowAllCORS {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	normalized := strings.ToLower(u.Scheme + "://" + u.Host)
	if _, ok := s.corsOrigins[normalized]; ok {
		return true
	}
	return isSameOriginRequest(r)
}

func (s *Server) pluginMutationsEnabled() bool {
	return s.enablePluginMutations
}

func (s *Server) handleDashboardSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !s.hasUsers() {
			token, _ := s.requestToken(r)
			if s.validateToken(token) {
				writeJSON(w, 200, map[string]any{"authenticated": true, "legacy_mode": true})
				return
			}
			writeJSON(w, 200, map[string]any{
				"setup_required": true,
				"legacy_mode":    strings.TrimSpace(s.apiToken) != "",
			})
			return
		}
		sess := s.validateUserSession(r)
		if sess == nil {
			writeJSON(w, 200, map[string]any{"authenticated": false})
			return
		}
		writeJSON(w, 200, map[string]any{"authenticated": true, "username": sess.Username, "role": sess.Role})
	case http.MethodPost:
		if s.hasUsers() {
			httpError(w, 410, "use POST /api/auth/login")
			return
		}
		// Legacy token mode
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		if !s.validateToken(body.Token) {
			httpError(w, 403, "invalid token")
			return
		}
		s.setDashboardSessionCookie(w, r, strings.TrimSpace(body.Token))
		writeJSON(w, 200, map[string]any{"authenticated": true})
	case http.MethodDelete:
		if s.hasUsers() {
			token, _ := s.requestToken(r)
			if token != "" {
				_, _ = s.db.Exec(`DELETE FROM user_sessions WHERE token=?`, token)
			}
		}
		s.clearDashboardSessionCookie(w, r)
		writeJSON(w, 200, map[string]any{"authenticated": false})
	default:
		httpError(w, 405, "method not allowed")
	}
}

func userRoleAllowed(role string, allowedRoles []string) bool {
	if len(allowedRoles) == 0 {
		return true
	}
	for _, allowed := range allowedRoles {
		if role == allowed {
			return true
		}
	}
	return false
}

func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func (s *Server) authorize(next http.HandlerFunc, allowedRoles []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Requests arriving over the Unix socket are pre-authenticated by
		// filesystem ACL (socket mode 0600, owned by the server process user).
		if r.Context().Value(ctxKeySocket) == true {
			next(w, r)
			return
		}
		if s.hasUsers() {
			// New user-session auth.
			sess := s.validateUserSession(r)
			if sess == nil {
				httpError(w, 401, "unauthorized")
				return
			}
			_, source := s.requestToken(r)
			if source == "cookie" && requiresSameOrigin(r.Method) && !isSameOriginRequest(r) {
				httpError(w, 403, "origin check failed")
				return
			}
			if !userRoleAllowed(sess.Role, allowedRoles) {
				httpError(w, 403, "insufficient role")
				return
			}
			next(w, r)
			return
		}
		// Legacy token auth (backward compat for installs without user accounts).
		token, source := s.requestToken(r)
		if token == "" {
			httpError(w, 401, "missing token")
			return
		}
		if !s.validateToken(token) {
			httpError(w, 403, "invalid token")
			return
		}
		if source == "cookie" && requiresSameOrigin(r.Method) && !isSameOriginRequest(r) {
			httpError(w, 403, "origin check failed")
			return
		}
		next(w, r)
	}
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return s.authorize(next, nil)
}

func (s *Server) authWithRoles(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return s.authorize(next, allowedRoles)
	}
}

func (s *Server) authByMethod(readRoles []string, writeRoles []string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			allowedRoles := writeRoles
			if isReadOnlyMethod(r.Method) {
				allowedRoles = readRoles
			}
			s.authorize(next, allowedRoles)(w, r)
		}
	}
}

// ─── User auth helpers ────────────────────────────────────────────────────────

func (s *Server) hasUsers() bool {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count > 0
}

func (s *Server) validateUserSession(r *http.Request) *UserSession {
	token, _ := s.requestToken(r)
	if token == "" {
		return nil
	}
	var sess UserSession
	var expiresAt int64
	err := s.db.QueryRow(
		`SELECT us.token, us.user_id, u.username, u.role, us.expires_at
		 FROM user_sessions us JOIN users u ON u.id = us.user_id
		 WHERE us.token=? AND us.expires_at>?`,
		token, time.Now().UnixMilli(),
	).Scan(&sess.Token, &sess.UserID, &sess.Username, &sess.Role, &expiresAt)
	if err != nil {
		return nil
	}
	return &sess
}

func (s *Server) createUserSession(userID string) (string, error) {
	token := newID() + newID() // 64 hex chars
	_, err := s.db.Exec(
		`INSERT INTO user_sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		token, userID, time.Now().UnixMilli(), time.Now().Add(30*24*time.Hour).UnixMilli(),
	)
	return token, err
}

// ─── Auth endpoints ───────────────────────────────────────────────────────────

// POST /api/auth/setup — create first owner account (only when no users exist).
func (s *Server) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	if s.hasUsers() {
		httpError(w, 409, "already set up")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if body.Username == "" || len(body.Password) < 8 {
		httpError(w, 400, "username required; password must be at least 8 characters")
		return
	}
	hash, err := hashPassword(body.Password)
	if err != nil {
		httpError(w, 500, "failed to hash password")
		return
	}
	id := newID()
	if _, err = s.db.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, 'owner', ?)`,
		id, body.Username, hash, time.Now().UnixMilli(),
	); err != nil {
		httpError(w, 500, "failed to create user")
		return
	}
	token, err := s.createUserSession(id)
	if err != nil {
		httpError(w, 500, "user created but session failed")
		return
	}
	s.setDashboardSessionCookie(w, r, token)
	writeJSON(w, 200, map[string]any{"username": body.Username, "role": "owner"})
}

// POST /api/auth/login
func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	if !s.hasUsers() {
		writeJSON(w, 200, map[string]any{"setup_required": true})
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		CLIPort  int    `json:"cli_port,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	var u User
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, role FROM users WHERE username=?`,
		strings.TrimSpace(body.Username),
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role)
	if err != nil || !checkPassword(body.Password, u.PasswordHash) {
		time.Sleep(300 * time.Millisecond) // slow down brute-force
		httpError(w, 403, "invalid credentials")
		return
	}
	token, err := s.createUserSession(u.ID)
	if err != nil {
		httpError(w, 500, "session error")
		return
	}
	s.setDashboardSessionCookie(w, r, token)
	resp := map[string]any{"username": u.Username, "role": u.Role}
	// CLI browser flow: if cli_port given, generate a short-lived auth code
	// that the CLI can exchange for a bearer token.
	if body.CLIPort > 0 {
		if cliResp, err := s.createCLIAuthResponse(u.ID, u.Username, u.Role, body.CLIPort); err == nil {
			for k, v := range cliResp {
				resp[k] = v
			}
		}
	}
	writeJSON(w, 200, resp)
}

// GET /api/auth/me
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	if !s.hasUsers() {
		writeJSON(w, 200, map[string]any{"setup_required": true})
		return
	}
	sess := s.validateUserSession(r)
	if sess == nil {
		writeJSON(w, 200, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, 200, map[string]any{"authenticated": true, "username": sess.Username, "role": sess.Role})
}

func (s *Server) createCLIAuthResponse(userID string, username string, role string, cliPort int) (map[string]any, error) {
	if cliPort <= 0 || cliPort > 65535 {
		return nil, fmt.Errorf("invalid cli port")
	}
	code := newID() + newID()
	if _, err := s.db.Exec(
		`INSERT INTO auth_codes (code, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		code, userID, time.Now().UnixMilli(), time.Now().Add(5*time.Minute).UnixMilli(),
	); err != nil {
		return nil, err
	}
	return map[string]any{
		"cli_code": code,
		"cli_redirect": fmt.Sprintf("http://localhost:%d/callback?code=%s&user=%s&role=%s",
			cliPort, code, url.QueryEscape(username), url.QueryEscape(role)),
	}, nil
}

// POST /api/auth/cli/start — mint a short-lived auth code from an existing signed-in browser session.
func (s *Server) handleAuthCLIStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	if !s.hasUsers() {
		writeJSON(w, 200, map[string]any{"setup_required": true})
		return
	}
	sess := s.validateUserSession(r)
	if sess == nil {
		httpError(w, 401, "unauthorized")
		return
	}
	var body struct {
		CLIPort int `json:"cli_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	resp, err := s.createCLIAuthResponse(sess.UserID, sess.Username, sess.Role, body.CLIPort)
	if err != nil {
		httpError(w, 400, err.Error())
		return
	}
	resp["username"] = sess.Username
	resp["role"] = sess.Role
	writeJSON(w, 200, resp)
}

// POST /api/auth/cli/exchange — exchange a short-lived auth code for a bearer token.
func (s *Server) handleAuthCLIExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		httpError(w, 400, "invalid request")
		return
	}
	var userID string
	var expiresAt int64
	if err := s.db.QueryRow(
		`SELECT user_id, expires_at FROM auth_codes WHERE code=?`, body.Code,
	).Scan(&userID, &expiresAt); err != nil || time.Now().UnixMilli() > expiresAt {
		httpError(w, 403, "invalid or expired code")
		return
	}
	_, _ = s.db.Exec(`DELETE FROM auth_codes WHERE code=?`, body.Code)
	var u User
	if err := s.db.QueryRow(
		`SELECT id, username, role FROM users WHERE id=?`, userID,
	).Scan(&u.ID, &u.Username, &u.Role); err != nil {
		httpError(w, 500, "user lookup failed")
		return
	}
	token, err := s.createUserSession(u.ID)
	if err != nil {
		httpError(w, 500, "session error")
		return
	}
	writeJSON(w, 200, map[string]any{"token": token, "username": u.Username, "role": u.Role})
}

// GET /api/sync/pull/{app}/{env}/{branch} — download current workspace as tar.
func (s *Server) handleSyncPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/sync/pull/"), "/", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		httpError(w, 400, "usage: /api/sync/pull/{app}/{env}/{branch}")
		return
	}
	appName, envStr, branch := parts[0], parts[1], parts[2]
	workspace := filepath.Join(s.workspacesDir, fmt.Sprintf("%s__%s__%s", safe(appName), safe(envStr), safe(branch)))
	repoDir := filepath.Join(workspace, "repo")
	if _, err := os.Stat(repoDir); err != nil {
		httpError(w, 404, "no workspace found for this app/env/branch — deploy first")
		return
	}
	wsVersion := ""
	if st, err := s.getAppState(appName, DeployEnv(envStr), branch); err == nil && st != nil {
		wsVersion = st.RepoHash
	}
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("X-Workspace-Version", wsVersion)
	tw := tar.NewWriter(w)
	_ = filepath.Walk(repoDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(repoDir, p)
		rel = filepath.ToSlash(rel)
		if shouldIgnoreRepoPath(rel) {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}
		_ = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     rel,
			Mode:     0644,
			Size:     int64(len(data)),
			ModTime:  info.ModTime(),
		})
		_, _ = tw.Write(data)
		return nil
	})
	_ = tw.Close()
}

// ─── Password hashing (PBKDF2-style, stdlib only) ─────────────────────────────

const pwHashIter = 100_000

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	h := pbkdf2HMACSHA256([]byte(password), salt, pwHashIter)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(h), nil
}

func checkPassword(password, stored string) bool {
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err2 := hex.DecodeString(parts[1])
	if err2 != nil {
		return false
	}
	got := pbkdf2HMACSHA256([]byte(password), salt, pwHashIter)
	return subtle.ConstantTimeCompare(got, expected) == 1
}

// pbkdf2HMACSHA256 is a single-block PBKDF2 with HMAC-SHA256 as the PRF.
// Using stdlib only (no golang.org/x/crypto dependency).
func pbkdf2HMACSHA256(password, salt []byte, iter int) []byte {
	mac := hmac.New(sha256.New, password)
	mac.Write(salt)
	mac.Write([]byte{0, 0, 0, 1})
	u := mac.Sum(nil)
	out := make([]byte, len(u))
	copy(out, u)
	for i := 1; i < iter; i++ {
		mac.Reset()
		mac.Write(u)
		u = mac.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return out
}

func (s *Server) withCORS(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && s.isOriginAllowed(r) {
			if s.allowAllCORS {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Relay-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			if origin != "" && !s.isOriginAllowed(r) {
				httpError(w, 403, "origin not allowed")
				return
			}
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// ---------------------- Sync helpers ----------------------

func (s *Server) getSession(id string) *SyncSession {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	return s.syncSessions[id]
}

func applySyncUpdates(sess *SyncSession) error {
	// Delete files requested
	for _, rel := range sess.DeleteList {
		rel = filepath.ToSlash(strings.TrimPrefix(rel, "./"))
		if rel == "" || !isSafeRelPath(rel) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(sess.RepoDir, filepath.FromSlash(rel)))
	}

	// Copy staging into repo (overwrite)
	err := filepath.Walk(sess.StagingDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(sess.StagingDir, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		dst := filepath.Join(sess.RepoDir, filepath.FromSlash(rel))
		if info.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		// file
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		return copyFile(p, dst, info.Mode())
	})
	if err != nil {
		return err
	}

	// Remove staging dir
	_ = os.RemoveAll(sess.StagingDir)
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	_ = os.Chmod(dst, mode)
	return nil
}

// ---------------------- DB ----------------------

func migrateDB(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS deploys (
			id TEXT PRIMARY KEY,
			app TEXT,
			repo_url TEXT,
			branch TEXT,
			commit_sha TEXT,
			env TEXT,
			status TEXT,
			created_at INTEGER,
			started_at INTEGER,
			ended_at INTEGER,
			error TEXT,
			log_path TEXT,
			image_tag TEXT,
			previous_image_tag TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS deploy_requests (
			id TEXT PRIMARY KEY,
			app TEXT,
			repo_url TEXT,
			branch TEXT,
			commit_sha TEXT,
			env TEXT,
			install_cmd TEXT,
			build_cmd TEXT,
			start_cmd TEXT,
			service_port INTEGER,
			host_port INTEGER,
			public_host TEXT,
			mode TEXT,
			source TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS app_state (
			app TEXT,
			env TEXT,
			branch TEXT,
			repo_url TEXT,
			engine TEXT DEFAULT 'docker',
			current_image TEXT,
			previous_image TEXT,
			mode TEXT,
			host_port INTEGER,
			host_port_explicit INTEGER DEFAULT 0,
			service_port INTEGER,
			public_host TEXT,
			active_slot TEXT,
			standby_slot TEXT,
			drain_until INTEGER,
			traffic_mode TEXT,
			access_policy TEXT,
			ip_allowlist TEXT,
			repo_hash TEXT,
			expires_at INTEGER DEFAULT 0,
			stopped INTEGER DEFAULT 0,
			updated_at INTEGER,
			PRIMARY KEY (app, env, branch)
		);`,
		`CREATE TABLE IF NOT EXISTS sync_sessions (
			id TEXT PRIMARY KEY,
			app TEXT,
			branch TEXT,
			env TEXT,
			repo_dir TEXT,
			staging_dir TEXT,
			created_at INTEGER,
			delete_list TEXT,
			uploaded_bytes INTEGER,
			max_bytes INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS app_secrets (
			app TEXT,
			env TEXT,
			branch TEXT,
			key TEXT,
			value TEXT,
			PRIMARY KEY (app, env, branch, key)
		);`,
		`CREATE TABLE IF NOT EXISTS project_services (
			project TEXT,
			name TEXT,
			type TEXT,
			branch TEXT,
			env TEXT,
			container TEXT,
			network TEXT,
			volume TEXT,
			env_key TEXT,
			env_val TEXT,
			image TEXT,
			port INTEGER,
			host_port INTEGER,
			spec_hash TEXT,
			updated_at INTEGER,
			PRIMARY KEY (project, name, branch, env)
		);`,
		`CREATE TABLE IF NOT EXISTS project_service_specs (
			project TEXT,
			env TEXT,
			branch TEXT,
			name TEXT,
			config_json TEXT,
			updated_at INTEGER,
			PRIMARY KEY (project, env, branch, name)
		);`,
	}
	for _, st := range stmts {
		if _, err := db.Exec(st); err != nil {
			return err
		}
	}
	// Best-effort schema upgrades for existing databases.
	_, _ = db.Exec(`ALTER TABLE deploys ADD COLUMN preview_url TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE deploys ADD COLUMN build_number INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE deploys ADD COLUMN deployed_by TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE deploys ADD COLUMN commit_message TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN webhook_secret TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN engine TEXT DEFAULT 'docker'`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN active_slot TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN standby_slot TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN drain_until INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN traffic_mode TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN access_policy TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN ip_allowlist TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN expires_at INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN stopped INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN host_port_explicit INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN image TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN port INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN host_port INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN spec_hash TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN repo_hash TEXT DEFAULT ''`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS server_config (key TEXT PRIMARY KEY, value TEXT)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'deployer',
		created_at INTEGER
	)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS user_sessions (
		token TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		created_at INTEGER,
		expires_at INTEGER
	)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS auth_codes (
		code TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		created_at INTEGER,
		expires_at INTEGER
	)`)
	// Analytics tables
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS analytics_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts INTEGER NOT NULL,
		host TEXT NOT NULL DEFAULT '',
		method TEXT NOT NULL DEFAULT '',
		path TEXT NOT NULL DEFAULT '',
		status INTEGER NOT NULL DEFAULT 0,
		bytes INTEGER NOT NULL DEFAULT 0,
		remote_ip TEXT NOT NULL DEFAULT '',
		country_code TEXT NOT NULL DEFAULT '',
		country_name TEXT NOT NULL DEFAULT ''
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_analytics_ts ON analytics_events(ts)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS ip_country_cache (
		ip TEXT PRIMARY KEY,
		country_code TEXT NOT NULL DEFAULT '',
		country_name TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL DEFAULT 0
	)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts INTEGER NOT NULL,
		actor TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL DEFAULT '',
		target TEXT NOT NULL DEFAULT '',
		detail TEXT NOT NULL DEFAULT ''
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(ts)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS promotions (
		id TEXT PRIMARY KEY,
		app TEXT NOT NULL,
		source_env TEXT NOT NULL,
		source_branch TEXT NOT NULL,
		source_deploy_id TEXT DEFAULT '',
		source_image TEXT DEFAULT '',
		target_env TEXT NOT NULL,
		target_branch TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending_approval',
		approval_required INTEGER NOT NULL DEFAULT 0,
		requested_by TEXT DEFAULT '',
		requested_at INTEGER NOT NULL DEFAULT 0,
		approved_by TEXT DEFAULT '',
		approved_at INTEGER NOT NULL DEFAULT 0,
		target_deploy_id TEXT DEFAULT '',
		rollback_deploy_id TEXT DEFAULT '',
		health_status TEXT DEFAULT '',
		health_detail TEXT DEFAULT ''
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_promotions_app_ts ON promotions(app, requested_at DESC)`)
	if err := seedLanePolicies(db); err != nil {
		return err
	}
	return nil
}

// ─── Audit log ───────────────────────────────────────────────────────────────

func (s *Server) auditLog(actor, action, target, detail string) {
	_, _ = s.db.Exec(
		`INSERT INTO audit_log (ts, actor, action, target, detail) VALUES (?, ?, ?, ?, ?)`,
		time.Now().UnixMilli(), actor, action, target, detail,
	)
}

// GET /api/audit — returns recent audit entries (owner only)
func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	limit := 200
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	rows, err := s.db.Query(
		`SELECT id, ts, actor, action, target, detail FROM audit_log ORDER BY ts DESC LIMIT ?`, limit,
	)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	defer rows.Close()
	type entry struct {
		ID     int64  `json:"id"`
		TS     int64  `json:"ts"`
		Actor  string `json:"actor"`
		Action string `json:"action"`
		Target string `json:"target"`
		Detail string `json:"detail"`
	}
	out := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.TS, &e.Actor, &e.Action, &e.Target, &e.Detail); err != nil {
			continue
		}
		out = append(out, e)
	}
	writeJSON(w, 200, out)
}

// ─── Secrets encryption ───────────────────────────────────────────────────────

// encryptSecret encrypts plaintext using AES-256-GCM. Returns the plaintext
// unchanged if no secret key is configured or on any error.
func (s *Server) encryptSecret(plaintext string) string {
	if len(s.secretKey) != 32 {
		return plaintext
	}
	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return plaintext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plaintext
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return plaintext
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "enc:" + base64.StdEncoding.EncodeToString(ct)
}

// decryptSecret decrypts a value previously encrypted by encryptSecret.
// Returns the value unchanged if it is not encrypted or on any error.
func (s *Server) decryptSecret(ciphertext string) string {
	if len(s.secretKey) != 32 || !strings.HasPrefix(ciphertext, "enc:") {
		return ciphertext
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext[4:])
	if err != nil {
		return ciphertext
	}
	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return ciphertext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ciphertext
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return ciphertext
	}
	plaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return ciphertext
	}
	return string(plaintext)
}

// ─── Build numbering ─────────────────────────────────────────────────────────

// nextBuildNumber returns the next sequential build number for an app.
func (s *Server) nextBuildNumber(app string) int {
	var n int
	_ = s.db.QueryRow(
		`SELECT COALESCE(MAX(build_number), 0) + 1 FROM deploys WHERE app=?`, app,
	).Scan(&n)
	if n < 1 {
		n = 1
	}
	return n
}

// ─── User management API ──────────────────────────────────────────────────────

// GET  /api/users — list users (owner only)
// POST /api/users — create user (owner only)
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if !s.hasUsers() {
		httpError(w, 404, "user accounts not enabled")
		return
	}
	sess := s.validateUserSession(r)
	if sess == nil {
		httpError(w, 401, "unauthorized")
		return
	}
	if sess.Role != "owner" {
		httpError(w, 403, "owner role required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.Query(
			`SELECT id, username, role, created_at FROM users ORDER BY created_at ASC`,
		)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		defer rows.Close()
		type userRow struct {
			ID        string `json:"id"`
			Username  string `json:"username"`
			Role      string `json:"role"`
			CreatedAt int64  `json:"created_at"`
		}
		out := []userRow{}
		for rows.Next() {
			var u userRow
			if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
				continue
			}
			out = append(out, u)
		}
		writeJSON(w, 200, out)

	case http.MethodPost:
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		body.Username = strings.TrimSpace(body.Username)
		if body.Username == "" || len(body.Password) < 8 {
			httpError(w, 400, "username required; password must be at least 8 characters")
			return
		}
		role := body.Role
		if role != "owner" && role != "deployer" && role != "viewer" {
			role = "deployer"
		}
		hash, err := hashPassword(body.Password)
		if err != nil {
			httpError(w, 500, "failed to hash password")
			return
		}
		id := newID()
		if _, err = s.db.Exec(
			`INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
			id, body.Username, hash, role, time.Now().UnixMilli(),
		); err != nil {
			httpError(w, 409, "username already exists")
			return
		}
		s.auditLog(sess.Username, "user.create", body.Username, "role="+role)
		writeJSON(w, 200, map[string]any{"id": id, "username": body.Username, "role": role})

	default:
		httpError(w, 405, "method not allowed")
	}
}

// DELETE /api/users/{id} — delete a user (owner only)
// PATCH  /api/users/{id} — change role (owner only)
func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	if !s.hasUsers() {
		httpError(w, 404, "user accounts not enabled")
		return
	}
	sess := s.validateUserSession(r)
	if sess == nil {
		httpError(w, 401, "unauthorized")
		return
	}
	if sess.Role != "owner" {
		httpError(w, 403, "owner role required")
		return
	}
	userID := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if userID == "" {
		httpError(w, 400, "user id required")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if userID == sess.UserID {
			httpError(w, 400, "cannot delete your own account")
			return
		}
		var username string
		_ = s.db.QueryRow(`SELECT username FROM users WHERE id=?`, userID).Scan(&username)
		if _, err := s.db.Exec(`DELETE FROM users WHERE id=?`, userID); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		_, _ = s.db.Exec(`DELETE FROM user_sessions WHERE user_id=?`, userID)
		s.auditLog(sess.Username, "user.delete", username, "")
		writeJSON(w, 200, map[string]bool{"ok": true})

	case http.MethodPatch:
		var body struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		if body.Role != "owner" && body.Role != "deployer" && body.Role != "viewer" {
			httpError(w, 400, "role must be owner, deployer, or viewer")
			return
		}
		var username string
		_ = s.db.QueryRow(`SELECT username FROM users WHERE id=?`, userID).Scan(&username)
		if _, err := s.db.Exec(`UPDATE users SET role=? WHERE id=?`, body.Role, userID); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		s.auditLog(sess.Username, "user.role", username, "role="+body.Role)
		writeJSON(w, 200, map[string]any{"id": userID, "role": body.Role})

	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) setDeployPreviewURL(id, url string) error {
	_, err := s.db.Exec(`UPDATE deploys SET preview_url=? WHERE id=?`, url, id)
	return err
}

func (s *Server) setLatestDeployPreviewURL(app string, env DeployEnv, branch string, previewURL string) error {
	_, err := s.db.Exec(
		`UPDATE deploys
		 SET preview_url=?
		 WHERE id = (
		   SELECT id FROM deploys
		   WHERE app=? AND env=? AND branch=?
		   ORDER BY created_at DESC
		   LIMIT 1
		 )`,
		previewURL, app, string(env), branch,
	)
	return err
}

func previewURLFromConfig(mode string, publicHost string, hostPort int) string {
	if strings.TrimSpace(publicHost) != "" {
		return "https://" + strings.TrimSpace(publicHost)
	}
	if firstNonEmpty(strings.ToLower(strings.TrimSpace(mode)), "port") == "port" && hostPort > 0 {
		return fmt.Sprintf("http://127.0.0.1:%d", hostPort)
	}
	return ""
}

func (s *Server) saveDeployToDB(d *Deploy, req DeployRequest) error {
	d.Source = req.Source
	if req.CommitMessage != "" {
		d.CommitMessage = req.CommitMessage
	}
	if req.DeployedBy != "" {
		d.DeployedBy = req.DeployedBy
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO deploys
		(id, app, repo_url, branch, commit_sha, env, status, created_at, started_at, ended_at, error, log_path, image_tag, previous_image_tag, preview_url, build_number, deployed_by, commit_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.App, d.RepoURL, d.Branch, d.CommitSHA, string(d.Env), string(d.Status),
		d.CreatedAt.UnixMilli(),
		timePtrToMillis(d.StartedAt),
		timePtrToMillis(d.EndedAt),
		d.Error, d.LogPath, d.ImageTag, d.PrevImage, d.PreviewURL,
		d.BuildNumber, d.DeployedBy, d.CommitMessage,
	)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO deploy_requests
		(id, app, repo_url, branch, commit_sha, env, install_cmd, build_cmd, start_cmd, service_port, host_port, public_host, mode, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, req.App, req.RepoURL, req.Branch, req.CommitSHA, string(req.Env),
		req.InstallCmd, req.BuildCmd, req.StartCmd,
		req.ServicePort, req.HostPort, req.PublicHost, req.Mode, req.Source,
	)
	return err
}

func (s *Server) updateDeployStatus(id string, status DeployStatus, errMsg string, started, ended *time.Time, imageTag, prevImg string) error {
	_, err := s.db.Exec(
		`UPDATE deploys SET status=?, error=?, started_at=?, ended_at=?, image_tag=?, previous_image_tag=? WHERE id=?`,
		string(status), errMsg, timePtrToMillis(started), timePtrToMillis(ended), imageTag, prevImg, id,
	)
	if err == nil {
		s.broadcastSnapshot()
	}
	return err
}

func failDeploy(s *Server, d *Deploy, err error, msg string) {
	end := time.Now()
	d.Status = StatusFailed
	d.EndedAt = &end
	d.Error = msg
	_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
}

func (s *Server) listDeploysFromDB() ([]*Deploy, error) {
	rows, err := s.db.Query(`SELECT d.id, d.app, d.repo_url, d.branch, d.commit_sha, d.env, d.status, d.created_at, d.started_at, d.ended_at, d.error, d.log_path, d.image_tag, d.previous_image_tag, COALESCE(d.preview_url,''), COALESCE(r.source,''), COALESCE(d.build_number,0), COALESCE(d.deployed_by,''), COALESCE(d.commit_message,'')
		FROM deploys d
		LEFT JOIN deploy_requests r ON r.id = d.id
		ORDER BY d.created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Deploy{}
	for rows.Next() {
		var d Deploy
		var env, st string
		var created, started, ended sql.NullInt64
		var errTxt sql.NullString
		var img, prev sql.NullString

		var purl string
		if err := rows.Scan(&d.ID, &d.App, &d.RepoURL, &d.Branch, &d.CommitSHA, &env, &st, &created, &started, &ended, &errTxt, &d.LogPath, &img, &prev, &purl, &d.Source, &d.BuildNumber, &d.DeployedBy, &d.CommitMessage); err != nil {
			continue
		}
		d.Env = DeployEnv(env)
		d.Status = DeployStatus(st)
		d.CreatedAt = millisToTime(created.Int64)
		d.PreviewURL = purl

		if started.Valid {
			t := millisToTime(started.Int64)
			d.StartedAt = &t
		}
		if ended.Valid {
			t := millisToTime(ended.Int64)
			d.EndedAt = &t
		}
		if errTxt.Valid {
			d.Error = errTxt.String
		}
		if img.Valid {
			d.ImageTag = img.String
		}
		if prev.Valid {
			d.PrevImage = prev.String
		}
		out = append(out, &d)
	}
	return out, nil
}

func (s *Server) getDeployFromDB(id string) (*Deploy, error) {
	row := s.db.QueryRow(`SELECT d.id, d.app, d.repo_url, d.branch, d.commit_sha, d.env, d.status, d.created_at, d.started_at, d.ended_at, d.error, d.log_path, d.image_tag, d.previous_image_tag, COALESCE(d.preview_url,''), COALESCE(r.source,''), COALESCE(d.build_number,0), COALESCE(d.deployed_by,''), COALESCE(d.commit_message,'')
		FROM deploys d
		LEFT JOIN deploy_requests r ON r.id = d.id
		WHERE d.id=?`, id)

	var d Deploy
	var env, st, purl string
	var created, started, ended sql.NullInt64
	var errTxt sql.NullString
	var img, prev sql.NullString

	if err := row.Scan(&d.ID, &d.App, &d.RepoURL, &d.Branch, &d.CommitSHA, &env, &st, &created, &started, &ended, &errTxt, &d.LogPath, &img, &prev, &purl, &d.Source, &d.BuildNumber, &d.DeployedBy, &d.CommitMessage); err != nil {
		return nil, err
	}
	d.PreviewURL = purl
	d.Env = DeployEnv(env)
	d.Status = DeployStatus(st)
	d.CreatedAt = millisToTime(created.Int64)
	if started.Valid {
		t := millisToTime(started.Int64)
		d.StartedAt = &t
	}
	if ended.Valid {
		t := millisToTime(ended.Int64)
		d.EndedAt = &t
	}
	if errTxt.Valid {
		d.Error = errTxt.String
	}
	if img.Valid {
		d.ImageTag = img.String
	}
	if prev.Valid {
		d.PrevImage = prev.String
	}
	return &d, nil
}

func (s *Server) loadDeploysFromDB() error {
	deploys, err := s.listDeploysFromDB()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range deploys {
		s.deploys[d.ID] = d
	}
	return nil
}

func (s *Server) reconcileStaleDeploysOnStartup() error {
	s.mu.RLock()
	active := make([]*Deploy, 0)
	for _, d := range s.deploys {
		if d != nil && isActiveDeployStatus(d.Status) {
			active = append(active, d)
		}
	}
	s.mu.RUnlock()

	if len(active) == 0 {
		return nil
	}

	now := time.Now()
	for _, d := range active {
		d.Status = StatusFailed
		if d.StartedAt == nil {
			started := now
			d.StartedAt = &started
		}
		ended := now
		d.EndedAt = &ended
		if strings.TrimSpace(d.Error) == "" {
			d.Error = "deploy interrupted by relay restart"
		}
		if err := s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, d.ImageTag, d.PrevImage); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) saveAppState(st *AppState) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO app_state
		(app, env, branch, repo_url, engine, current_image, previous_image, mode, host_port, host_port_explicit, service_port, public_host, active_slot, standby_slot, drain_until, traffic_mode, access_policy, ip_allowlist, repo_hash, expires_at, webhook_secret, stopped, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		st.App, string(st.Env), st.Branch, st.RepoURL, firstNonEmptyEngine(st.Engine), st.CurrentImage, st.PreviousImage, st.Mode,
		st.HostPort, st.HostPortExplicit, st.ServicePort, st.PublicHost, normalizeActiveSlot(st.ActiveSlot), normalizeActiveSlot(st.StandbySlot), st.DrainUntil, firstNonEmpty(normalizeTrafficMode(st.TrafficMode), "edge"), firstNonEmpty(normalizeAccessPolicy(st.AccessPolicy), s.lanePolicy(st.Env).DefaultAccessPolicy), normalizeIPAllowlist(st.IPAllowlist), st.RepoHash, st.ExpiresAt, st.WebhookSecret, st.Stopped, time.Now().UnixMilli(),
	)
	return err
}

func (s *Server) getAppState(app string, env DeployEnv, branch string) (*AppState, error) {
	row := s.db.QueryRow(`SELECT app, env, branch, repo_url, COALESCE(engine,''), current_image, previous_image, mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(active_slot,''), COALESCE(standby_slot,''), COALESCE(drain_until,0), COALESCE(traffic_mode,''), COALESCE(access_policy,''), COALESCE(ip_allowlist,''), COALESCE(repo_hash,''), COALESCE(expires_at,0), COALESCE(webhook_secret,''), COALESCE(stopped,0)
		FROM app_state WHERE app=? AND env=? AND branch=?`, app, string(env), branch)

	var st AppState
	var envS string
	if err := row.Scan(&st.App, &envS, &st.Branch, &st.RepoURL, &st.Engine, &st.CurrentImage, &st.PreviousImage, &st.Mode, &st.HostPort, &st.HostPortExplicit, &st.ServicePort, &st.PublicHost, &st.ActiveSlot, &st.StandbySlot, &st.DrainUntil, &st.TrafficMode, &st.AccessPolicy, &st.IPAllowlist, &st.RepoHash, &st.ExpiresAt, &st.WebhookSecret, &st.Stopped); err != nil {
		return nil, err
	}
	st.Env = DeployEnv(envS)
	s.constrainAppState(&st)
	return &st, nil
}

func (s *Server) saveSessionToDB(sess *SyncSession) error {
	delJSON, _ := json.Marshal(sess.DeleteList)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO sync_sessions
		(id, app, branch, env, repo_dir, staging_dir, created_at, delete_list, uploaded_bytes, max_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.App, sess.Branch, string(sess.Env), sess.RepoDir, sess.StagingDir,
		sess.CreatedAt.UnixMilli(), string(delJSON), sess.UploadedBytes, sess.MaxBytes,
	)
	return err
}

func (s *Server) deleteSessionFromDB(id string) error {
	_, err := s.db.Exec(`DELETE FROM sync_sessions WHERE id=?`, id)
	return err
}

func (s *Server) loadSessionsFromDB() error {
	rows, err := s.db.Query(`SELECT id, app, branch, env, repo_dir, staging_dir, created_at, delete_list, uploaded_bytes, max_bytes FROM sync_sessions`)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	for rows.Next() {
		var sess SyncSession
		var envS string
		var created int64
		var delStr sql.NullString
		if err := rows.Scan(&sess.ID, &sess.App, &sess.Branch, &envS, &sess.RepoDir, &sess.StagingDir, &created, &delStr, &sess.UploadedBytes, &sess.MaxBytes); err != nil {
			continue
		}
		sess.Env = DeployEnv(envS)
		sess.CreatedAt = millisToTime(created)
		if delStr.Valid && delStr.String != "" {
			_ = json.Unmarshal([]byte(delStr.String), &sess.DeleteList)
		} else {
			sess.DeleteList = []string{}
		}
		s.syncSessions[sess.ID] = &sess
	}
	return nil
}

// ---------------------- Helpers ----------------------

func validateDeployRequest(req DeployRequest) error {
	if req.App == "" {
		return fmt.Errorf("app required")
	}
	if req.Branch == "" {
		return fmt.Errorf("branch required")
	}
	req.Env = normalizeDeployEnv(string(req.Env))
	if !isKnownDeployEnv(req.Env) {
		return fmt.Errorf("env must be one of: preview, dev, staging, prod")
	}
	if req.Mode != "" {
		m := strings.ToLower(strings.TrimSpace(req.Mode))
		if m != "port" && m != "traefik" {
			return fmt.Errorf("mode must be port or traefik")
		}
	}
	if req.TrafficMode != "" {
		tm := normalizeTrafficMode(req.TrafficMode)
		if tm == "" {
			return fmt.Errorf("traffic_mode must be edge or session")
		}
	}
	if req.HostPort < 0 {
		return fmt.Errorf("host_port must be >= 0")
	}
	if req.ServicePort < 0 {
		return fmt.Errorf("service_port must be >= 0")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}

func getenv(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

func mustMkdir(p string) {
	_ = os.MkdirAll(p, 0755)
}

func safe(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "x"
	}
	return string(out)
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func loadOrCreateToken(dataDir string) (string, bool) {
	p := filepath.Join(dataDir, "token.txt")
	if b, err := os.ReadFile(p); err == nil {
		t := strings.TrimSpace(string(b))
		if t != "" {
			return t, false
		}
	}
	// create
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	t := hex.EncodeToString(b)
	_ = os.WriteFile(p, []byte(t+"\n"), 0600)
	return t, true
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func firstNonZero(v, def int) int {
	if v != 0 {
		return v
	}
	return def
}

func timePtrToMillis(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UnixMilli()
}

func millisToTime(ms int64) time.Time {
	return time.UnixMilli(ms)
}

func isSafeRelPath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	// No absolute paths
	if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") {
		return false
	}
	// Clean using path (slash-separated) to avoid platform-specific tricks
	clean := path.Clean(rel)
	if clean == "." {
		return false
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	// Block null bytes or other control chars
	if strings.ContainsAny(clean, "\x00") {
		return false
	}
	return true
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func pathWithinBase(baseDir string, target string) bool {
	if strings.TrimSpace(baseDir) == "" || strings.TrimSpace(target) == "" {
		return false
	}
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func hasPackageDependency(repoDir string, dep string) bool {
	p := filepath.Join(repoDir, "package.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]any `json:"dependencies"`
		DevDependencies map[string]any `json:"devDependencies"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		return strings.Contains(strings.ToLower(string(b)), `"`+strings.ToLower(dep)+`"`)
	}
	_, ok := pkg.Dependencies[dep]
	if ok {
		return true
	}
	_, ok = pkg.DevDependencies[dep]
	return ok
}

func findFilesByExt(repoDir string, exts ...string) []string {
	allowed := map[string]struct{}{}
	for _, ext := range exts {
		allowed[strings.ToLower(ext)] = struct{}{}
	}
	var out []string
	_ = filepath.Walk(repoDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if _, ok := allowed[ext]; !ok {
			return nil
		}
		rel, relErr := filepath.Rel(repoDir, p)
		if relErr == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func readFirstExisting(repoDir string, names ...string) (string, string) {
	for _, name := range names {
		p := filepath.Join(repoDir, name)
		if b, err := os.ReadFile(p); err == nil {
			return name, string(b)
		}
	}
	return "", ""
}

func hasWasmAssets(repoDir string) bool {
	return len(findFilesByExt(repoDir, ".wasm")) > 0
}

func anyPathExists(repoDir string, wantDir bool, paths []string) bool {
	for _, p := range paths {
		if pathExists(repoDir, p, wantDir) {
			return true
		}
	}
	return false
}

func allPathExists(repoDir string, wantDir bool, paths []string) bool {
	for _, p := range paths {
		if !pathExists(repoDir, p, wantDir) {
			return false
		}
	}
	return true
}

func pathExists(repoDir string, rel string, wantDir bool) bool {
	clean := filepath.Clean(rel)
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, "..") {
		return false
	}
	p := filepath.Join(repoDir, clean)
	st, err := os.Stat(p)
	if err != nil {
		return false
	}
	if wantDir {
		return st.IsDir()
	}
	return !st.IsDir()
}

func anyPackageDep(repoDir string, deps []string) bool {
	for _, dep := range deps {
		if hasPackageDependency(repoDir, dep) {
			return true
		}
	}
	return false
}

func allPackageDeps(repoDir string, deps []string) bool {
	for _, dep := range deps {
		if !hasPackageDependency(repoDir, dep) {
			return false
		}
	}
	return true
}

func anyFileExt(repoDir string, exts []string) bool {
	for _, ext := range exts {
		if len(findFilesByExt(repoDir, ext)) > 0 {
			return true
		}
	}
	return false
}

func allFileExt(repoDir string, exts []string) bool {
	for _, ext := range exts {
		if len(findFilesByExt(repoDir, ext)) == 0 {
			return false
		}
	}
	return true
}

func fileHashByAlgo(p string, algo string) (string, error) {
	algo = strings.ToLower(strings.TrimSpace(algo))
	switch algo {
	case "", "sha256":
		return sha256File(p)
	default:
		return "", fmt.Errorf("unsupported hash algo: %s", algo)
	}
}

func sha256File(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func repoFingerprint(repoDir string) string {
	// Content-based fingerprint of build inputs. This is used to reuse a prior
	// artifact when the repo contents are unchanged, so size/modtime alone are
	// not strong enough.
	h := sha256.New()
	_, _ = h.Write([]byte(runtime.GOOS))
	_ = filepath.Walk(repoDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		rel, _ := filepath.Rel(repoDir, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if shouldIgnoreRepoPath(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}
		_, _ = h.Write([]byte(rel))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{0})
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// ---------------------- Minimal defaults for buildpacks ----------------------
// These keep your buildpacks compiling even if you haven’t added custom logic yet.
// You can refine them later.

func cfgStr(cfg *RelayConfig, field string) string {
	if cfg == nil {
		return ""
	}
	switch field {
	case "BuildImage":
		return strings.TrimSpace(cfg.BuildImage)
	case "RunImage":
		return strings.TrimSpace(cfg.RunImage)
	default:
		return ""
	}
}

func quoteForSh(s string) string {
	// wrap string in single quotes safely: ' becomes '"'"'
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func shQuote(s string) string { return quoteForSh(s) }

func shellJSON(cmd string) string {
	// Dockerfile JSON CMD expects ["sh","-lc","..."] form for arbitrary strings
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		cmd = "sleep 3600"
	}
	b, _ := json.Marshal([]string{"sh", "-lc", cmd})
	return string(b)
}

func shellForm(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		cmd = "sleep 3600"
	}
	// shell form: sh -lc "..."
	return fmt.Sprintf(`["sh","-lc",%s]`, mustJSON(cmd))
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// Node helpers
var (
	nextConfigStandaloneLiteralRe   = regexp.MustCompile(`\boutput\s*:\s*(?:"standalone"|'standalone')`)
	nextConfigStandaloneAssignVarRe = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:"standalone"|'standalone')`)
)

func nodePackageManager(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if fileExists(filepath.Join(repoDir, "yarn.lock")) {
		return "yarn"
	}
	return "npm"
}

func nodeInstallCmd(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return "corepack enable && pnpm install --frozen-lockfile --prefer-offline"
	case "yarn":
		return "corepack enable && yarn install --frozen-lockfile --silent"
	default:
		return "npm ci --prefer-offline --no-audit --no-fund"
	}
}

func nodeProdInstallCmd(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return "corepack enable && pnpm install --frozen-lockfile --prod --prefer-offline"
	case "yarn":
		return "corepack enable && yarn install --frozen-lockfile --production=true --silent"
	default:
		return "npm ci --omit=dev --prefer-offline --no-audit --no-fund"
	}
}

func nodeProdDepsStage(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return fmt.Sprintf(`FROM deps AS prod-deps
RUN pnpm prune --prod || true
RUN mkdir -p /app/node_modules
`)
	case "yarn":
		return fmt.Sprintf(`FROM deps AS prod-deps
RUN rm -rf node_modules
%s
RUN mkdir -p /app/node_modules
`, nodeRunStepWithCaches(repoDir, nodeProdInstallCmd(repoDir)))
	default:
		return `FROM deps AS prod-deps
RUN npm prune --omit=dev --no-audit --no-fund || true
RUN mkdir -p /app/node_modules
`
	}
}

func nodePackageCacheDir(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return "/root/.local/share/pnpm/store"
	case "yarn":
		return "/usr/local/share/.cache/yarn"
	default:
		return "/root/.npm"
	}
}

func nodeRunStepWithCaches(repoDir string, cmd string, extraTargets ...string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}

	var mounts []string
	if target := strings.TrimSpace(nodePackageCacheDir(repoDir)); target != "" {
		mounts = append(mounts, fmt.Sprintf("--mount=type=cache,target=%s", target))
	}
	for _, target := range extraTargets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		mounts = append(mounts, fmt.Sprintf("--mount=type=cache,target=%s", target))
	}
	if len(mounts) == 0 {
		return fmt.Sprintf("RUN %s", cmd)
	}
	return fmt.Sprintf("RUN %s %s", strings.Join(mounts, " "), cmd)
}

func nodeDefaultBuildCmd(repoDir string) string {
	// Pick package manager based on lockfile
	if fileExists(filepath.Join(repoDir, "pnpm-lock.yaml")) {
		return "pnpm run build"
	}
	if fileExists(filepath.Join(repoDir, "yarn.lock")) {
		return "yarn build"
	}
	return "npm run build"
}

func nodeDefaultStartCmd(repoDir string) string {
	// Default to start script matching package manager
	if fileExists(filepath.Join(repoDir, "pnpm-lock.yaml")) {
		return "pnpm start"
	}
	if fileExists(filepath.Join(repoDir, "yarn.lock")) {
		return "yarn start"
	}
	return "npm start"
}

func nodePackageScript(repoDir string, name string) string {
	b, err := os.ReadFile(filepath.Join(repoDir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		return ""
	}
	return strings.TrimSpace(pkg.Scripts[name])
}

func nextClassicStartCmd(repoDir string) string {
	if nodePackageScript(repoDir, "start") != "" {
		return nodeDefaultStartCmd(repoDir)
	}
	return `exec ./node_modules/.bin/next start --hostname 0.0.0.0 --port ${PORT:-3000}`
}

func isNextStandaloneEnabled(repoDir string) bool {
	configs := []string{"next.config.js", "next.config.mjs", "next.config.cjs", "next.config.ts", "next.config.mts"}
	for _, c := range configs {
		p := filepath.Join(repoDir, c)
		if b, err := os.ReadFile(p); err == nil {
			if nextConfigEnablesStandalone(string(b)) {
				return true
			}
		}
	}
	return false
}

func nextConfigEnablesStandalone(src string) bool {
	cleaned := stripJSComments(src)
	if nextConfigStandaloneLiteralRe.MatchString(cleaned) {
		return true
	}
	for _, match := range nextConfigStandaloneAssignVarRe.FindAllStringSubmatch(cleaned, -1) {
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if nextConfigUsesOutputVariable(cleaned, name) {
			return true
		}
		if name == "output" && nextConfigHasOutputShorthand(cleaned) {
			return true
		}
	}
	return false
}

func nextConfigUsesOutputVariable(src string, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	re := regexp.MustCompile(fmt.Sprintf(`\boutput\s*:\s*%s\b`, regexp.QuoteMeta(name)))
	return re.MatchString(src)
}

func nextConfigHasOutputShorthand(src string) bool {
	for i := 0; i < len(src); i++ {
		if !isJSIdentifierStart(src[i]) {
			continue
		}
		j := i + 1
		for j < len(src) && isJSIdentifierPart(src[j]) {
			j++
		}
		if src[i:j] != "output" {
			i = j - 1
			continue
		}
		prev := prevNonSpaceByte(src, i-1)
		next := nextNonSpaceByte(src, j)
		if (prev == '{' || prev == ',') && (next == '}' || next == ',') {
			return true
		}
		i = j - 1
	}
	return false
}

func prevNonSpaceByte(src string, idx int) byte {
	for idx >= 0 {
		if !isJSSpace(src[idx]) {
			return src[idx]
		}
		idx--
	}
	return 0
}

func nextNonSpaceByte(src string, idx int) byte {
	for idx < len(src) {
		if !isJSSpace(src[idx]) {
			return src[idx]
		}
		idx++
	}
	return 0
}

func isJSSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isJSIdentifierStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isJSIdentifierPart(ch byte) bool {
	return isJSIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func stripJSComments(src string) string {
	const (
		jsStateNormal = iota
		jsStateSingleQuote
		jsStateDoubleQuote
		jsStateTemplate
		jsStateLineComment
		jsStateBlockComment
	)
	var out strings.Builder
	out.Grow(len(src))
	state := jsStateNormal
	for i := 0; i < len(src); i++ {
		ch := src[i]
		switch state {
		case jsStateNormal:
			if ch == '/' && i+1 < len(src) {
				switch src[i+1] {
				case '/':
					state = jsStateLineComment
					i++
					continue
				case '*':
					state = jsStateBlockComment
					i++
					continue
				}
			}
			out.WriteByte(ch)
			switch ch {
			case '\'':
				state = jsStateSingleQuote
			case '"':
				state = jsStateDoubleQuote
			case '`':
				state = jsStateTemplate
			}
		case jsStateLineComment:
			if ch == '\n' || ch == '\r' {
				out.WriteByte(ch)
				state = jsStateNormal
			}
		case jsStateBlockComment:
			if ch == '*' && i+1 < len(src) && src[i+1] == '/' {
				i++
				state = jsStateNormal
				continue
			}
			if ch == '\n' || ch == '\r' {
				out.WriteByte(ch)
			}
		case jsStateSingleQuote:
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
				continue
			}
			if ch == '\'' {
				state = jsStateNormal
			}
		case jsStateDoubleQuote:
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
				continue
			}
			if ch == '"' {
				state = jsStateNormal
			}
		case jsStateTemplate:
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
				continue
			}
			if ch == '`' {
				state = jsStateNormal
			}
		}
	}
	return out.String()
}

// Python helpers
func pythonInstallCmd(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "requirements.txt")) {
		return `sh -lc "pip install --no-cache-dir -r requirements.txt"`
	}
	if fileExists(filepath.Join(repoDir, "pyproject.toml")) {
		return `sh -lc "pip install --no-cache-dir ."`
	}
	if fileExists(filepath.Join(repoDir, "Pipfile")) {
		return `sh -lc "pip install --no-cache-dir pipenv && pipenv install --system --deploy"`
	}
	return ""
}

func pythonEntryModule(repoDir string) string {
	for _, name := range []string{"main.py", "app.py"} {
		if fileExists(filepath.Join(repoDir, name)) {
			return strings.TrimSuffix(name, ".py")
		}
	}
	return "main"
}

func pythonFramework(repoDir string) string {
	_, content := readFirstExisting(repoDir, "main.py", "app.py")
	if content != "" {
		s := strings.ToLower(content)
		switch {
		case strings.Contains(s, "fastapi"):
			return "fastapi"
		case strings.Contains(s, "flask"):
			return "flask"
		}
	}
	for _, name := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if b, err := os.ReadFile(filepath.Join(repoDir, name)); err == nil {
			s := strings.ToLower(string(b))
			switch {
			case strings.Contains(s, "fastapi"), strings.Contains(s, "uvicorn"):
				return "fastapi"
			case strings.Contains(s, "flask"), strings.Contains(s, "gunicorn"):
				return "flask"
			}
		}
	}
	return "generic"
}

func pythonDefaultStart(repoDir string) string {
	module := pythonEntryModule(repoDir)
	switch pythonFramework(repoDir) {
	case "fastapi":
		return fmt.Sprintf(`sh -lc "python -m uvicorn %s:app --host 0.0.0.0 --port ${PORT:-8000}"`, module)
	case "flask":
		return fmt.Sprintf(`sh -lc "FLASK_APP=%s flask run --host 0.0.0.0 --port ${PORT:-8000}"`, module)
	}
	if fileExists(filepath.Join(repoDir, "main.py")) {
		return `sh -lc "python main.py"`
	}
	if fileExists(filepath.Join(repoDir, "app.py")) {
		return `sh -lc "python app.py"`
	}
	return `sh -lc "python -m http.server ${PORT:-8000} --bind 0.0.0.0"`
}

// Java helpers
func javaDefaultBuild(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "pom.xml")) {
		return `sh -lc "mvn -q -DskipTests package && cp target/*.jar app.jar"`
	}
	// gradle
	if fileExists(filepath.Join(repoDir, "build.gradle")) || fileExists(filepath.Join(repoDir, "build.gradle.kts")) {
		return `sh -lc "./gradlew -q build -x test && cp build/libs/*.jar app.jar"`
	}
	return `sh -lc "echo 'no build tool detected'; exit 1"`
}

// .NET helpers
func dotnetPickEntry(repoDir string) string {
	// Prefer .sln
	if m, _ := filepath.Glob(filepath.Join(repoDir, "*.sln")); len(m) > 0 {
		return filepath.Base(m[0])
	}
	if m, _ := filepath.Glob(filepath.Join(repoDir, "*.csproj")); len(m) > 0 {
		return filepath.Base(m[0])
	}
	return ""
}

func cCppDefaultBuildCmd(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "CMakeLists.txt")) {
		return `sh -lc "cmake -S . -B build -DCMAKE_BUILD_TYPE=Release && cmake --build build --config Release -j && bin=$(find build -type f -perm /111 ! -name '*.so' ! -name '*.a' | head -n1); test -n \"$bin\"; mkdir -p /out; cp \"$bin\" /out/app"`
	}
	if fileExists(filepath.Join(repoDir, "Makefile")) {
		return `sh -lc "make && bin=$(find . -maxdepth 4 -type f -perm /111 ! -name '*.so' ! -name '*.a' ! -path './.git/*' | head -n1); test -n \"$bin\"; mkdir -p /out; cp \"$bin\" /out/app"`
	}
	if len(findFilesByExt(repoDir, ".cc", ".cpp", ".cxx")) > 0 {
		return `sh -lc "mkdir -p /out && g++ -O2 -std=c++17 -o /out/app $(find . -maxdepth 4 -type f \( -name '*.cc' -o -name '*.cpp' -o -name '*.cxx' \) | sort)"`
	}
	if len(findFilesByExt(repoDir, ".c")) > 0 {
		return `sh -lc "mkdir -p /out && gcc -O2 -o /out/app $(find . -maxdepth 4 -type f -name '*.c' | sort)"`
	}
	return `sh -lc "echo 'no C/C++ sources found'; exit 1"`
}

func writeStaticDockerArtifacts(repoDir string, dockerfile string, includeWasmMime bool) error {
	if err := os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		return err
	}
	defPath := filepath.Join(repoDir, "default.conf")
	marker := filepath.Join(repoDir, ".relay_default_conf_created")
	if fileExists(defPath) {
		return nil
	}
	var conf strings.Builder
	conf.WriteString("server {\n")
	conf.WriteString("  listen 80;\n")
	conf.WriteString("  server_name _;\n")
	conf.WriteString("  root /usr/share/nginx/html;\n")
	conf.WriteString("  index index.html;\n")
	if includeWasmMime {
		conf.WriteString("  types {\n")
		conf.WriteString("    application/wasm wasm;\n")
		conf.WriteString("  }\n")
	}
	conf.WriteString("  location / {\n")
	conf.WriteString("    try_files $uri $uri/ /index.html;\n")
	conf.WriteString("  }\n")
	conf.WriteString("}\n")
	if err := os.WriteFile(defPath, []byte(conf.String()), 0644); err != nil {
		return err
	}
	return os.WriteFile(marker, []byte("1"), 0644)
}

func cleanupStaticDockerArtifacts(repoDir string, removeNodeArtifacts bool) error {
	if removeNodeArtifacts {
		_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
		_ = os.RemoveAll(filepath.Join(repoDir, "dist"))
	}
	_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
	marker := filepath.Join(repoDir, ".relay_default_conf_created")
	if fileExists(marker) {
		_ = os.Remove(filepath.Join(repoDir, "default.conf"))
		_ = os.Remove(marker)
	}
	return nil
}
