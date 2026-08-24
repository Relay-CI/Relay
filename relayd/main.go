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
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	relayd [--cloud-agent] [--cloud-hub]
	relayd version
	relayd service <...>

Examples:
	relayd
	relayd --port 9090
	relayd --addr 0.0.0.0:9090
	relayd --cloud-agent
	relayd --cloud-hub
	relayd --data-dir /var/lib/relayd --socket /var/lib/relayd/relay.sock
`

type relaydRunConfig struct {
	Addr          string
	DataDir       string
	SocketPath    string
	DisableSocket bool
	CloudAgent    bool
	CloudHub      bool
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
		case a == "--cloud-agent":
			cfg.CloudAgent = true
		case a == "--cloud-hub":
			cfg.CloudHub = true
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
	App             string    `json:"app"`
	RepoURL         string    `json:"repo_url"`
	Branch          string    `json:"branch"`
	CommitSHA       string    `json:"commit_sha"`
	Env             DeployEnv `json:"env"`
	WorkspaceEnv    DeployEnv `json:"workspace_env,omitempty"`
	WorkspaceBranch string    `json:"workspace_branch,omitempty"`

	// Optional overrides; otherwise resolved from detected plan / relay.config.json
	InstallCmd string `json:"install_cmd"`
	BuildCmd   string `json:"build_cmd"`
	StartCmd   string `json:"start_cmd"`

	ServicePort int `json:"service_port"`
	HostPort    int `json:"host_port"`
	// Internal marker so persisted explicit preview ports are not silently re-assigned.
	HostPortExplicit bool     `json:"-"`
	PublicHost       string   `json:"public_host"`
	PublicHosts      []string `json:"public_hosts,omitempty"`

	Mode        string `json:"mode"`         // "traefik" or "port"
	TrafficMode string `json:"traffic_mode"` // "edge" or "session"
	Source      string `json:"source"`       // "git" or "sync"
	Engine      string `json:"engine"`       // "docker" or "station"/"vessel"; overrides stored app state when set

	CommitMessage string `json:"commit_message,omitempty"`
	DeployedBy    string `json:"-"` // username of the user who triggered the deploy
	GitToken      string `json:"-"` // injected at clone time, never serialized
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

type AppSecret struct {
	App    string `json:"app"`
	Env    string `json:"env"`
	Branch string `json:"branch"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

type SyncStartRequest struct {
	App          string    `json:"app"`
	Branch       string    `json:"branch"`
	Env          DeployEnv `json:"env"`
	BaseVersion  string    `json:"base_version,omitempty"`
	LocalChanged bool      `json:"local_changed,omitempty"`
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
	Role         string // "owner", "admin", "deployer", "viewer"
}

type UserPermission struct {
	UserID string `json:"-"`
	App    string `json:"app"`
	Env    string `json:"env"`
	Role   string `json:"role"`
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

	db          *sql.DB
	analyticsDB *sql.DB
	relayDB     *relayDBManager

	apiToken  string
	secretKey []byte // 32-byte AES-256 key for encrypting secrets at rest; nil = no encryption

	buildpacks []Buildpack

	building sync.Map // key: "app__env__branch" → bool; guards per-lane build dedup

	buildCancelsMu sync.Mutex
	buildCancels   map[string]context.CancelFunc // deployID → active cancel func

	webhookRateMu sync.Mutex
	webhookHits   map[string][]time.Time // repoURL → recent trigger timestamps

	rolloutWatchMu sync.Mutex
	rolloutWatches map[string]struct{}

	eventsMu    sync.RWMutex
	eventsChans map[chan []byte]struct{}

	runtime        ContainerRuntime
	stationRuntime ContainerRuntime

	acmeMu       sync.Mutex
	acmeServer   *http.Server
	acmeListener net.Listener
	socketPath   string
	cloud        *cloudController
}

// ---------------------- Preview URL helper ----------------------

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

// acmeDisabled returns true when the operator has disabled the lightweight
// ACME HTTP-01 listener via admin settings or RELAY_DISABLE_ACME=true.
func (s *Server) acmeDisabled() bool {
	if v := strings.TrimSpace(os.Getenv("RELAY_DISABLE_ACME")); v == "true" || v == "1" {
		return true
	}
	return s.serverConfigGet("acme_disabled") == "true"
}

type customHostRule struct {
	ID                  string `json:"id,omitempty"`
	Host                string `json:"host"`
	Action              string `json:"action"`
	RedirectURL         string `json:"redirect_url,omitempty"`
	RedirectCode        int    `json:"redirect_code,omitempty"`
	PreservePath        bool   `json:"preserve_path,omitempty"`
	UpstreamURL         string `json:"upstream_url,omitempty"`
	ResponseStatus      int    `json:"response_status,omitempty"`
	ResponseBody        string `json:"response_body,omitempty"`
	ResponseContentType string `json:"response_content_type,omitempty"`
}

func normalizeCustomHostRules(rules []customHostRule) []customHostRule {
	out := make([]customHostRule, 0, len(rules))
	for _, rule := range rules {
		rule.ID = strings.TrimSpace(rule.ID)
		rule.Host = normalizedHostname(rule.Host)
		rule.Action = strings.TrimSpace(rule.Action)
		rule.RedirectURL = strings.TrimSpace(rule.RedirectURL)
		rule.UpstreamURL = strings.TrimSpace(rule.UpstreamURL)
		rule.ResponseContentType = strings.TrimSpace(rule.ResponseContentType)
		if rule.RedirectCode == 0 {
			rule.RedirectCode = http.StatusFound
		}
		if rule.ResponseStatus == 0 {
			rule.ResponseStatus = http.StatusOK
		}
		out = append(out, rule)
	}
	return out
}

func validateAbsoluteHTTPURL(raw string, field string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s must be a valid URL", field)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("%s must include a host", field)
	}
	return nil
}

func validateCustomHostRule(rule customHostRule, relayPort int) error {
	if err := validateProxyHostname(rule.Host, "custom_host_rules.host"); err != nil {
		return err
	}
	switch rule.Action {
	case "redirect":
		if err := validateAbsoluteHTTPURL(rule.RedirectURL, "custom_host_rules.redirect_url"); err != nil {
			return err
		}
		switch rule.RedirectCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		default:
			return fmt.Errorf("custom_host_rules.redirect_code must be 301, 302, 307, or 308")
		}
	case "reverse_proxy":
		if err := validateAbsoluteHTTPURL(rule.UpstreamURL, "custom_host_rules.upstream_url"); err != nil {
			return err
		}
	case "static_response":
		if rule.ResponseStatus < 100 || rule.ResponseStatus > 599 {
			return fmt.Errorf("custom_host_rules.response_status must be between 100 and 599")
		}
	case "relay_dashboard":
		if relayPort <= 0 {
			return fmt.Errorf("relay dashboard host rules require Relay to listen on a TCP port")
		}
	default:
		return fmt.Errorf("custom_host_rules.action must be redirect, reverse_proxy, static_response, or relay_dashboard")
	}
	return nil
}

func encodeCustomHostRules(rules []customHostRule) string {
	rules = normalizeCustomHostRules(rules)
	if len(rules) == 0 {
		return ""
	}
	data, err := json.Marshal(rules)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseCustomHostRules(raw string) []customHostRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var rules []customHostRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil
	}
	return normalizeCustomHostRules(rules)
}

func (s *Server) serverCustomHostRules() []customHostRule {
	return parseCustomHostRules(s.serverConfigGet("custom_host_rules"))
}

type doctorCheck struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Detail  string `json:"detail,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type doctorReport struct {
	GeneratedAt       int64                  `json:"generated_at"`
	HTTPAddr          string                 `json:"http_addr"`
	SocketPath        string                 `json:"socket_path,omitempty"`
	BaseDomain        string                 `json:"base_domain,omitempty"`
	DashboardHost     string                 `json:"dashboard_host,omitempty"`
	ManagedExampleURL string                 `json:"managed_example_url,omitempty"`
	WebhookURL        string                 `json:"webhook_url,omitempty"`
	Checks            map[string]doctorCheck `json:"checks"`
}

func doctorOK(summary string, detail ...string) doctorCheck {
	check := doctorCheck{Status: "ok", Summary: summary}
	if len(detail) > 0 {
		check.Detail = detail[0]
	}
	return check
}

func doctorWarn(summary string, detail string, hint string) doctorCheck {
	return doctorCheck{Status: "warn", Summary: summary, Detail: detail, Hint: hint}
}

func doctorError(summary string, detail string, hint string) doctorCheck {
	return doctorCheck{Status: "error", Summary: summary, Detail: detail, Hint: hint}
}

func doctorInfo(summary string, detail string) doctorCheck {
	return doctorCheck{Status: "info", Summary: summary, Detail: detail}
}

func ensureDirWritable(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return errors.New("directory path is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".relay-doctor-write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return err
	}
	_ = os.Remove(probe)
	return nil
}

func dockerServerVersion() (string, error) {
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	version := strings.TrimSpace(string(out))
	if version == "" {
		return "", errors.New("docker returned an empty server version")
	}
	return version, nil
}

func resolveHostIPs(host string) ([]string, error) {
	if strings.TrimSpace(host) == "" {
		return nil, errors.New("host is empty")
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0, len(ips))
	for _, ip := range ips {
		if strings.TrimSpace(ip) != "" {
			filtered = append(filtered, ip)
		}
	}
	if len(filtered) == 0 {
		return nil, errors.New("host resolved without IP records")
	}
	return filtered, nil
}

func httpsReachable(host string) error {
	if strings.TrimSpace(host) == "" {
		return errors.New("host is empty")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://" + host + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTPS /health returned %d", resp.StatusCode)
	}
	return nil
}

func webhookProbeURL(httpAddr string, dashboardHost string) string {
	if host := strings.TrimSpace(dashboardHost); host != "" {
		return "https://" + host + "/api/webhooks/github"
	}
	port := listenAddrPort(httpAddr)
	if port == 0 || port == 80 {
		return "http://127.0.0.1/api/webhooks/github"
	}
	return fmt.Sprintf("http://127.0.0.1:%d/api/webhooks/github", port)
}

func (s *Server) buildDoctorReport() doctorReport {
	baseDomain := s.serverBaseDomain()
	dashboardHost := s.serverDashboardHost()
	managedExampleURL := ""
	if baseDomain != "" {
		managedExampleURL = "https://" + fmt.Sprintf("relay-doctor-main.%s", baseDomain)
	}
	report := doctorReport{
		GeneratedAt:       time.Now().UnixMilli(),
		HTTPAddr:          s.httpAddr,
		SocketPath:        s.socketPath,
		BaseDomain:        baseDomain,
		DashboardHost:     dashboardHost,
		ManagedExampleURL: managedExampleURL,
		WebhookURL:        webhookProbeURL(s.httpAddr, dashboardHost),
		Checks:            map[string]doctorCheck{},
	}

	if err := ensureDirWritable(s.dataDir); err != nil {
		report.Checks["data_dir"] = doctorError("Data directory is not writable", err.Error(), "Check ownership, permissions, and the configured RELAY_DATA_DIR path.")
	} else {
		report.Checks["data_dir"] = doctorOK("Data directory is writable", s.dataDir)
	}

	if s.secretKey == nil {
		report.Checks["secret_key"] = doctorWarn("Secrets are not encrypted at rest", "RELAY_SECRET_KEY is not configured.", "Set RELAY_SECRET_KEY before production use.")
	} else {
		report.Checks["secret_key"] = doctorOK("Secrets encryption is enabled")
	}

	if version, err := dockerServerVersion(); err != nil {
		report.Checks["docker"] = doctorError("Docker is unavailable", err.Error(), "Install Docker, start the daemon, and confirm the relayd user can talk to it.")
	} else {
		report.Checks["docker"] = doctorOK("Docker daemon is reachable", "server version "+version)
	}

	if s.socketPath == "" {
		report.Checks["socket"] = doctorInfo("Unix socket is disabled", "Set RELAY_SOCKET or omit --no-socket to enable the local CLI transport.")
	} else if fi, err := os.Stat(s.socketPath); err != nil {
		report.Checks["socket"] = doctorWarn("Unix socket path is configured but not present", err.Error(), "Restart relayd or check RELAY_SOCKET.")
	} else {
		report.Checks["socket"] = doctorOK("Unix socket is present", fi.Mode().String())
	}

	if s.acmeDisabled() {
		report.Checks["acme"] = doctorWarn("ACME helper listener is disabled", "Relay will not bind port 80 for HTTP-01 challenges.", "Leave it disabled only if another ACME/TLS layer already owns port 80.")
	} else if s.acmeListener == nil {
		report.Checks["acme"] = doctorWarn("ACME helper listener is not active", "Relay could not confirm a live port 80 listener.", "Ensure port 80 is open or let the Caddy proxy handle HTTP-01 challenges.")
	} else {
		report.Checks["acme"] = doctorOK("ACME helper listener is active", s.acmeListener.Addr().String())
	}

	if baseDomain == "" {
		report.Checks["base_domain"] = doctorInfo("Managed base domain is not configured", "Set Base Domain if you want Relay to auto-assign app subdomains.")
	} else {
		testHost := "relay-doctor-main." + baseDomain
		if ips, err := resolveHostIPs(testHost); err != nil {
			report.Checks["base_domain"] = doctorWarn("Wildcard DNS could not be verified", err.Error(), "Point *.your-domain at this server before relying on managed subdomains.")
		} else {
			report.Checks["base_domain"] = doctorOK("Wildcard-style DNS resolves", strings.Join(ips, ", "))
		}
	}

	if dashboardHost == "" {
		report.Checks["dashboard_host"] = doctorInfo("Dashboard host is not configured", "Set Dashboard Host if you want Relay itself behind managed HTTPS.")
	} else {
		if ips, err := resolveHostIPs(dashboardHost); err != nil {
			report.Checks["dashboard_host"] = doctorWarn("Dashboard host does not resolve yet", err.Error(), "Create an A or CNAME record for the dashboard host.")
		} else if err := httpsReachable(dashboardHost); err != nil {
			check := doctorWarn("Dashboard DNS resolves but HTTPS is not healthy yet", err.Error(), "Wait for Caddy to issue a certificate or check ports 80/443 and public routing.")
			check.Detail += " | " + strings.Join(ips, ", ")
			report.Checks["dashboard_host"] = check
		} else {
			report.Checks["dashboard_host"] = doctorOK("Dashboard host resolves and HTTPS is reachable", strings.Join(ips, ", "))
		}
	}

	if baseDomain == "" && dashboardHost == "" {
		report.Checks["caddy_proxy"] = doctorInfo("Global proxy is optional right now", "Set a base domain or dashboard host to make Relay manage HTTPS routes.")
	} else if s.runtime.IsRunning("relay-global-proxy") {
		report.Checks["caddy_proxy"] = doctorOK("Global Caddy proxy is running", "relay-global-proxy")
	} else {
		report.Checks["caddy_proxy"] = doctorWarn("Global Caddy proxy is not running", "Relay could not find the relay-global-proxy container.", "Check Docker health and re-save the server domain settings.")
	}

	report.Checks["webhook"] = doctorInfo("GitHub webhook endpoint", report.WebhookURL)
	return report
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	writeJSON(w, 200, s.buildDoctorReport())
}

// handleServerConfig handles GET/POST /api/server/config.
func (s *Server) handleServerConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, map[string]any{
			"base_domain":               s.serverBaseDomain(),
			"dashboard_host":            s.serverDashboardHost(),
			"acme_disabled":             s.serverConfigGet("acme_disabled"),
			"custom_host_rules":         s.serverCustomHostRules(),
			"theme_name":                s.serverConfigGet("theme_name"),
			"theme_css":                 s.serverConfigGet("theme_css"),
			"plugin_mutations_enabled":  s.pluginMutationsEnabled(),
			"doctor":                    s.buildDoctorReport(),
			"image_retention_per_lane":  s.imageRetentionPerLaneSetting(),
			"unused_image_max_age_days": s.unusedImageMaxAgeDaysSetting(),
			"log_retention_days":        s.logRetentionDaysSetting(),
			"build_cache_keep_gb":       s.buildCacheKeepGBSetting(),
		})
	case http.MethodPost:
		var body struct {
			BaseDomain            *string           `json:"base_domain"`
			DashboardHost         *string           `json:"dashboard_host"`
			ACMEDisabled          *string           `json:"acme_disabled"`
			CustomHostRules       *[]customHostRule `json:"custom_host_rules"`
			ThemeName             *string           `json:"theme_name"`
			ThemeCSS              *string           `json:"theme_css"`
			ImageRetentionPerLane *int              `json:"image_retention_per_lane"`
			UnusedImageMaxAgeDays *int              `json:"unused_image_max_age_days"`
			LogRetentionDays      *int              `json:"log_retention_days"`
			BuildCacheKeepGB      *int              `json:"build_cache_keep_gb"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid JSON")
			return
		}
		// Only validate and update fields that were actually sent in the request.
		updates := map[string]string{}
		proxyChanged := false
		if body.BaseDomain != nil {
			v := strings.TrimSpace(*body.BaseDomain)
			if err := validateProxyHostname(v, "base_domain"); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			updates["base_domain"] = v
			proxyChanged = true
		}
		if body.DashboardHost != nil {
			v := strings.TrimSpace(*body.DashboardHost)
			if err := validateProxyHostname(v, "dashboard_host"); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			updates["dashboard_host"] = v
			proxyChanged = true
		}
		if body.ACMEDisabled != nil {
			v := ""
			if *body.ACMEDisabled == "true" {
				v = "true"
			}
			updates["acme_disabled"] = v
			proxyChanged = true
		}
		if body.CustomHostRules != nil {
			rules := normalizeCustomHostRules(*body.CustomHostRules)
			relayPort := listenAddrPort(s.httpAddr)
			for _, rule := range rules {
				if err := validateCustomHostRule(rule, relayPort); err != nil {
					httpError(w, 400, err.Error())
					return
				}
			}
			updates["custom_host_rules"] = encodeCustomHostRules(rules)
			proxyChanged = true
		}
		if body.ThemeName != nil {
			updates["theme_name"] = strings.TrimSpace(*body.ThemeName)
		}
		if body.ThemeCSS != nil {
			updates["theme_css"] = *body.ThemeCSS
		}
		if body.ImageRetentionPerLane != nil {
			if *body.ImageRetentionPerLane < 0 {
				httpError(w, 400, "image_retention_per_lane must be >= 0")
				return
			}
			updates["image_retention_per_lane"] = strconv.Itoa(*body.ImageRetentionPerLane)
		}
		if body.UnusedImageMaxAgeDays != nil {
			if *body.UnusedImageMaxAgeDays < 0 {
				httpError(w, 400, "unused_image_max_age_days must be >= 0")
				return
			}
			updates["unused_image_max_age_days"] = strconv.Itoa(*body.UnusedImageMaxAgeDays)
		}
		if body.LogRetentionDays != nil {
			if *body.LogRetentionDays < 0 {
				httpError(w, 400, "log_retention_days must be >= 0")
				return
			}
			updates["log_retention_days"] = strconv.Itoa(*body.LogRetentionDays)
		}
		if body.BuildCacheKeepGB != nil {
			if *body.BuildCacheKeepGB < 0 {
				httpError(w, 400, "build_cache_keep_gb must be >= 0")
				return
			}
			updates["build_cache_keep_gb"] = strconv.Itoa(*body.BuildCacheKeepGB)
		}
		previous := map[string]string{
			"base_domain":       s.serverConfigGet("base_domain"),
			"dashboard_host":    s.serverConfigGet("dashboard_host"),
			"acme_disabled":     s.serverConfigGet("acme_disabled"),
			"custom_host_rules": s.serverConfigGet("custom_host_rules"),
		}
		tx, err := s.db.Begin()
		if err != nil {
			httpError(w, 500, "db begin failed: "+err.Error())
			return
		}
		for key, value := range updates {
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
		if proxyChanged {
			s.startACMEListener()
			if err := s.ensureGlobalProxy(); err != nil {
				_ = s.writeServerConfig(previous)
				httpError(w, 500, "failed to refresh global proxy: "+err.Error())
				return
			}
		}
		writeJSON(w, 200, map[string]any{
			"base_domain":               s.serverConfigGet("base_domain"),
			"dashboard_host":            s.serverConfigGet("dashboard_host"),
			"acme_disabled":             s.serverConfigGet("acme_disabled"),
			"custom_host_rules":         s.serverCustomHostRules(),
			"theme_name":                s.serverConfigGet("theme_name"),
			"theme_css":                 s.serverConfigGet("theme_css"),
			"plugin_mutations_enabled":  s.pluginMutationsEnabled(),
			"doctor":                    s.buildDoctorReport(),
			"image_retention_per_lane":  s.imageRetentionPerLaneSetting(),
			"unused_image_max_age_days": s.unusedImageMaxAgeDaysSetting(),
			"log_retention_days":        s.logRetentionDaysSetting(),
			"build_cache_keep_gb":       s.buildCacheKeepGBSetting(),
		})
	default:
		httpError(w, 405, "method not allowed")
	}
}

// handlePublicTheme handles GET /api/public/theme — no authentication required.
// Returns the active theme name and any custom CSS so the UI can apply it before login.
func (s *Server) handlePublicTheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	writeJSON(w, 200, map[string]string{
		"theme_name": s.serverConfigGet("theme_name"),
		"theme_css":  s.serverConfigGet("theme_css"),
	})
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

// ---------------------- Projects API handler ----------------------

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	var sess *UserSession
	if s.hasUsers() {
		sess = s.validateUserSession(r)
		if sess == nil {
			httpError(w, 401, "unauthorized")
			return
		}
	}

	const projectsQueryWithResources = `SELECT app, env, branch, COALESCE(engine,''), mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(public_hosts,''), COALESCE(active_slot,''), COALESCE(standby_slot,''), COALESCE(drain_until,0), COALESCE(traffic_mode,''), COALESCE(access_policy,''), COALESCE(ip_allowlist,''), COALESCE(expires_at,0), COALESCE(webhook_secret,''), COALESCE(notification_webhooks,''), COALESCE(traffic_split_percent,100), COALESCE(rollout_min_requests,25), COALESCE(rollout_error_percent,5), COALESCE(rollout_assess_seconds,300), COALESCE(rollout_started_at,0), COALESCE(rollout_deploy_id,''), COALESCE(rollout_status,''), repo_url, COALESCE(stopped,0), COALESCE(cpu_limit,''), COALESCE(mem_limit,''), COALESCE(resource_mode,''), COALESCE(buildpack_kind,'')
		FROM app_state ORDER BY app, env, branch`
	const projectsQueryLegacy = `SELECT app, env, branch, COALESCE(engine,''), mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(public_hosts,''), COALESCE(active_slot,''), COALESCE(standby_slot,''), COALESCE(drain_until,0), COALESCE(traffic_mode,''), COALESCE(access_policy,''), COALESCE(ip_allowlist,''), COALESCE(expires_at,0), COALESCE(webhook_secret,''), COALESCE(notification_webhooks,''), COALESCE(traffic_split_percent,100), COALESCE(rollout_min_requests,25), COALESCE(rollout_error_percent,5), COALESCE(rollout_assess_seconds,300), COALESCE(rollout_started_at,0), COALESCE(rollout_deploy_id,''), COALESCE(rollout_status,''), repo_url, COALESCE(stopped,0)
		FROM app_state ORDER BY app, env, branch`
	rows, err := s.db.Query(projectsQueryWithResources)
	hasResourceColumns := true
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no such column") {
		rows, err = s.db.Query(projectsQueryLegacy)
		hasResourceColumns = false
	}
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	type ProjectEnv struct {
		App                  string   `json:"app"`
		Env                  string   `json:"env"`
		Branch               string   `json:"branch"`
		Engine               string   `json:"engine,omitempty"`
		Mode                 string   `json:"mode"`
		HostPort             int      `json:"host_port"`
		HostPortExplicit     bool     `json:"host_port_explicit,omitempty"`
		ServicePort          int      `json:"service_port"`
		PublicHost           string   `json:"public_host"`
		PublicHosts          []string `json:"public_hosts,omitempty"`
		ActiveSlot           string   `json:"active_slot,omitempty"`
		StandbySlot          string   `json:"standby_slot,omitempty"`
		DrainUntil           int64    `json:"drain_until,omitempty"`
		TrafficMode          string   `json:"traffic_mode,omitempty"`
		AccessPolicy         string   `json:"access_policy,omitempty"`
		IPAllowlist          string   `json:"ip_allowlist,omitempty"`
		ExpiresAt            int64    `json:"expires_at,omitempty"`
		WebhookSecret        string   `json:"webhook_secret,omitempty"`
		NotificationWebhooks string   `json:"notification_webhooks,omitempty"`
		TrafficSplitPercent  int      `json:"traffic_split_percent,omitempty"`
		RolloutMinRequests   int      `json:"rollout_min_requests,omitempty"`
		RolloutErrorPercent  float64  `json:"rollout_error_percent,omitempty"`
		RolloutAssessSeconds int      `json:"rollout_assess_seconds,omitempty"`
		RolloutStartedAt     int64    `json:"rollout_started_at,omitempty"`
		RolloutDeployID      string   `json:"rollout_deploy_id,omitempty"`
		RolloutStatus        string   `json:"rollout_status,omitempty"`
		RepoURL              string   `json:"repo_url"`
		Stopped              bool     `json:"stopped,omitempty"`
		AccessRole           string   `json:"access_role,omitempty"`
		CPULimit             string   `json:"cpu_limit,omitempty"`
		MemLimit             string   `json:"mem_limit,omitempty"`
		ResourceMode         string   `json:"resource_mode,omitempty"`
		BuildpackKind        string   `json:"buildpack_kind,omitempty"`
	}

	type ProjectInfo struct {
		Name     string           `json:"name"`
		Envs     []ProjectEnv     `json:"envs"`
		Services []ProjectService `json:"services"`
	}

	byApp := map[string]*ProjectInfo{}
	for rows.Next() {
		var pe ProjectEnv
		var publicHostsRaw string
		if hasResourceColumns {
			if err := rows.Scan(&pe.App, &pe.Env, &pe.Branch, &pe.Engine, &pe.Mode,
				&pe.HostPort, &pe.HostPortExplicit, &pe.ServicePort, &pe.PublicHost, &publicHostsRaw, &pe.ActiveSlot, &pe.StandbySlot, &pe.DrainUntil, &pe.TrafficMode, &pe.AccessPolicy, &pe.IPAllowlist, &pe.ExpiresAt, &pe.WebhookSecret, &pe.NotificationWebhooks, &pe.TrafficSplitPercent, &pe.RolloutMinRequests, &pe.RolloutErrorPercent, &pe.RolloutAssessSeconds, &pe.RolloutStartedAt, &pe.RolloutDeployID, &pe.RolloutStatus, &pe.RepoURL, &pe.Stopped, &pe.CPULimit, &pe.MemLimit, &pe.ResourceMode, &pe.BuildpackKind); err != nil {
				continue
			}
		} else {
			if err := rows.Scan(&pe.App, &pe.Env, &pe.Branch, &pe.Engine, &pe.Mode,
				&pe.HostPort, &pe.HostPortExplicit, &pe.ServicePort, &pe.PublicHost, &publicHostsRaw, &pe.ActiveSlot, &pe.StandbySlot, &pe.DrainUntil, &pe.TrafficMode, &pe.AccessPolicy, &pe.IPAllowlist, &pe.ExpiresAt, &pe.WebhookSecret, &pe.NotificationWebhooks, &pe.TrafficSplitPercent, &pe.RolloutMinRequests, &pe.RolloutErrorPercent, &pe.RolloutAssessSeconds, &pe.RolloutStartedAt, &pe.RolloutDeployID, &pe.RolloutStatus, &pe.RepoURL, &pe.Stopped); err != nil {
				continue
			}
		}
		pe.PublicHosts = parsePublicHosts(publicHostsRaw)
		pe.PublicHost, pe.PublicHosts = canonicalizePublicHosts(pe.PublicHost, pe.PublicHosts)
		pe.Engine = firstNonEmptyEngine(pe.Engine)
		pe.TrafficMode = firstNonEmpty(normalizeTrafficMode(pe.TrafficMode), s.lanePolicy(DeployEnv(pe.Env)).DefaultTrafficMode)
		pe.AccessPolicy = firstNonEmpty(normalizeAccessPolicy(pe.AccessPolicy), s.lanePolicy(DeployEnv(pe.Env)).DefaultAccessPolicy)
		if sess != nil {
			pe.AccessRole = s.effectiveLaneRole(sess, pe.App, DeployEnv(pe.Env))
			if !roleAtLeast(pe.AccessRole, "viewer") {
				continue
			}
		}
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
				if sess != nil && !roleAtLeast(s.effectiveLaneRole(sess, ps.Project, DeployEnv(ps.Env)), "viewer") {
					continue
				}
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
		if len(v.Envs) == 0 && len(v.Services) == 0 {
			continue
		}
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

var errLaneNotFound = errors.New("lane not found")

func seedAllBaselineLanesEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("RELAY_SEED_BASELINE_LANES"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func baselineLaneSeedTargets(preferredEnv DeployEnv) []DeployEnv {
	preferredEnv = normalizeDeployEnv(string(preferredEnv))
	if preferredEnv == "" {
		preferredEnv = EnvProd
	}
	if seedAllBaselineLanesEnabled() {
		return baselineDeployEnvs()
	}
	return []DeployEnv{preferredEnv}
}

func (s *Server) ensureBaselineLanes(app string, branch string, preferredEnv DeployEnv, preferredRepoURL string, preferredEngine string) error {
	app = strings.TrimSpace(app)
	branch = strings.TrimSpace(branch)
	preferredEnv = normalizeDeployEnv(string(preferredEnv))
	preferredRepoURL = strings.TrimSpace(preferredRepoURL)
	if app == "" || branch == "" || s == nil || s.db == nil {
		return nil
	}

	baseEngine := firstNonEmptyEngine(preferredEngine)
	if baseEngine == "" {
		baseEngine = detectDefaultEngine()
	}

	for _, env := range baselineLaneSeedTargets(preferredEnv) {
		if existing, err := s.getAppState(app, env, branch); err == nil && existing != nil {
			continue
		}
		policy := s.lanePolicy(env)
		st := &AppState{
			App:          app,
			Env:          env,
			Branch:       branch,
			RepoURL:      "",
			Engine:       baseEngine,
			Mode:         policy.DefaultMode,
			TrafficMode:  policy.DefaultTrafficMode,
			AccessPolicy: policy.DefaultAccessPolicy,
			HostPort:     policy.DefaultHostPort,
			ServicePort:  3000,
		}
		if preferredRepoURL != "" && env == preferredEnv {
			st.RepoURL = preferredRepoURL
		}
		s.constrainAppState(st)
		if err := s.saveAppState(st); err != nil {
			return err
		}
	}

	// Keep unused baseline lanes fully off by default when only one lane is
	// seeded for a project (e.g. first deploy to prod).
	if !seedAllBaselineLanesEnabled() {
		for _, env := range baselineDeployEnvs() {
			if env == preferredEnv {
				continue
			}
			st, err := s.getAppState(app, env, branch)
			if err != nil || st == nil {
				continue
			}
			if strings.TrimSpace(st.CurrentImage) != "" || st.Stopped {
				continue
			}
			s.stopDockerAppLane(app, env, branch)
			_ = s.stopStationLane(app, env, branch)
			s.stopLaneServices(app, env, branch)
			st.Stopped = true
			s.constrainAppState(st)
			_ = s.saveAppState(st)
		}
	}
	return nil
}

func (s *Server) projectHasActiveWork(app string) bool {
	app = strings.TrimSpace(app)
	if app == "" {
		return false
	}

	found := false
	s.building.Range(func(k, _ any) bool {
		if strings.HasPrefix(k.(string), app+"__") {
			found = true
			return false
		}
		return true
	})
	if found {
		return true
	}

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

func (s *Server) appLaneHasActiveWork(app string, env DeployEnv, branch string) bool {
	app = strings.TrimSpace(app)
	branch = strings.TrimSpace(branch)
	if !validDeployTarget(app, env, branch) {
		return false
	}

	lockKey := fmt.Sprintf("%s__%s__%s", app, env, branch)
	if _, ok := s.building.Load(lockKey); ok {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, deploy := range s.deploys {
		if deploy == nil || deploy.App != app || deploy.Env != env || deploy.Branch != branch {
			continue
		}
		if deploy.Status == StatusQueued || deploy.Status == StatusRunning {
			return true
		}
	}
	return false
}

func (s *Server) deleteLaneData(app string, env DeployEnv, branch string) (map[string]int64, []string, error) {
	app = strings.TrimSpace(app)
	branch = strings.TrimSpace(branch)
	if !validDeployTarget(app, env, branch) {
		return nil, nil, fmt.Errorf("app, branch, env required")
	}

	var lane projectLaneRecord
	lane.Env = env
	lane.Branch = branch
	hasLane := true
	if err := s.db.QueryRow(
		`SELECT COALESCE(current_image,''), COALESCE(previous_image,'')
		 FROM app_state WHERE app=? AND env=? AND branch=?`,
		app, string(env), branch,
	).Scan(&lane.CurrentImage, &lane.PreviousImage); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			hasLane = false
		} else {
			return nil, nil, fmt.Errorf("load lane state: %w", err)
		}
	}

	services := make([]ProjectService, 0, 8)
	serviceRows, err := s.db.Query(
		`SELECT project, name, type, branch, env, container, network, volume, env_key, env_val, COALESCE(image,''), COALESCE(port,0), COALESCE(host_port,0), COALESCE(spec_hash,'')
		 FROM project_services WHERE project=? AND env=? AND branch=?`,
		app, string(env), branch,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load lane services: %w", err)
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
		 FROM deploys WHERE app=? AND env=? AND branch=?`,
		app, string(env), branch,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load lane deploys: %w", err)
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
		`SELECT id, repo_dir, staging_dir FROM sync_sessions WHERE app=? AND env=? AND branch=?`,
		app, string(env), branch,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load lane sync sessions: %w", err)
	}
	for sessionRows.Next() {
		var rec projectSyncRecord
		if err := sessionRows.Scan(&rec.ID, &rec.RepoDir, &rec.StagingDir); err != nil {
			continue
		}
		sessions = append(sessions, rec)
	}
	sessionRows.Close()

	if !hasLane && len(services) == 0 && len(deploys) == 0 && len(sessions) == 0 {
		return nil, nil, errLaneNotFound
	}

	warnings := make([]string, 0, 8)
	images := map[string]struct{}{}
	networks := map[string]struct{}{}
	workspaces := map[string]struct{}{}
	logs := map[string]struct{}{}
	edgeConfigs := map[string]struct{}{}
	volumes := map[string]struct{}{}

	s.runtime.Remove(appBaseContainerName(app, env, branch))
	s.runtime.Remove(appSlotContainerName(app, env, branch, "blue"))
	s.runtime.Remove(appSlotContainerName(app, env, branch, "green"))
	_ = s.stopStationLane(app, env, branch)
	networks[appNetworkName(app, env, branch)] = struct{}{}
	workspaces[filepath.Join(s.workspacesDir, fmt.Sprintf("%s__%s__%s", safe(app), safe(string(env)), safe(branch)))] = struct{}{}
	edgeConfigs[s.edgeProxyConfigPath(app, env, branch)] = struct{}{}
	if lane.CurrentImage != "" {
		images[lane.CurrentImage] = struct{}{}
	}
	if lane.PreviousImage != "" {
		images[lane.PreviousImage] = struct{}{}
	}

	for _, svc := range services {
		s.removeProjectServiceContainers(svc)
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
		return nil, warnings, fmt.Errorf("begin delete lane transaction: %w", err)
	}

	execDelete := func(query string, args ...any) (int64, error) {
		res, err := tx.Exec(query, args...)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return n, nil
	}

	deployRequestCount, err := execDelete(`DELETE FROM deploy_requests WHERE app=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete deploy requests: %w", err)
	}
	deployCount, err := execDelete(`DELETE FROM deploys WHERE app=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete deploys: %w", err)
	}
	appStateCount, err := execDelete(`DELETE FROM app_state WHERE app=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete app state: %w", err)
	}
	secretCount, err := execDelete(`DELETE FROM app_secrets WHERE app=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete app secrets: %w", err)
	}
	credentialCount, err := execDelete(`DELETE FROM service_credentials WHERE project=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete service credentials: %w", err)
	}
	serviceStateCount, err := execDelete(`DELETE FROM project_services WHERE project=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete service state: %w", err)
	}
	serviceSpecCount, err := execDelete(`DELETE FROM project_service_specs WHERE project=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete service specs: %w", err)
	}
	sessionCount, err := execDelete(`DELETE FROM sync_sessions WHERE app=? AND env=? AND branch=?`, app, string(env), branch)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete sync sessions: %w", err)
	}
	promotionCount, err := execDelete(
		`DELETE FROM promotions
		  WHERE app=? AND ((source_env=? AND source_branch=?) OR (target_env=? AND target_branch=?))`,
		app, string(env), branch, string(env), branch,
	)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete promotions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, warnings, fmt.Errorf("commit delete lane transaction: %w", err)
	}

	s.mu.Lock()
	for id, deploy := range s.deploys {
		if deploy != nil && deploy.App == app && deploy.Env == env && deploy.Branch == branch {
			delete(s.deploys, id)
		}
	}
	s.mu.Unlock()

	s.syncMu.Lock()
	for id, sess := range s.syncSessions {
		if sess != nil && sess.App == app && sess.Env == env && sess.Branch == branch {
			delete(s.syncSessions, id)
		}
	}
	s.syncMu.Unlock()

	s.building.Delete(fmt.Sprintf("%s__%s__%s", app, env, branch))

	s.broadcastSnapshot()
	go func() { _ = s.ensureGlobalProxy() }()

	return map[string]int64{
		"lanes":               appStateCount,
		"deploys":             deployCount,
		"deploy_requests":     deployRequestCount,
		"secrets":             secretCount,
		"service_credentials": credentialCount,
		"service_states":      serviceStateCount,
		"service_specs":       serviceSpecCount,
		"sync_sessions":       sessionCount,
		"promotions":          promotionCount,
		"runtime_services":    int64(len(services)),
	}, warnings, nil
}

func (s *Server) handleLaneDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}

	var req LaneDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	req.Env = normalizeDeployEnv(string(req.Env))
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}
	if _, ok := s.requireLaneAccess(w, r, req.App, req.Env, "deployer"); !ok {
		return
	}
	if strings.TrimSpace(req.Confirm) != "DELETE" {
		httpError(w, 400, "confirm must be DELETE")
		return
	}
	if s.appLaneHasActiveWork(req.App, req.Env, req.Branch) {
		httpError(w, 409, "cannot delete a lane while deploys are queued or running")
		return
	}

	summary, warnings, err := s.deleteLaneData(req.App, req.Env, req.Branch)
	if err != nil {
		if errors.Is(err, errLaneNotFound) {
			httpError(w, 404, "lane not found")
			return
		}
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{
		"status":   "deleted",
		"app":      req.App,
		"env":      req.Env,
		"branch":   req.Branch,
		"summary":  summary,
		"warnings": warnings,
	})
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
		s.removeProjectServiceContainers(svc)
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
	credentialCount, err := execDelete(`DELETE FROM service_credentials WHERE project=?`, app)
	if err != nil {
		_ = tx.Rollback()
		return nil, warnings, fmt.Errorf("delete service credentials: %w", err)
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

	s.building.Range(func(k, _ any) bool {
		if strings.HasPrefix(k.(string), app+"__") {
			s.building.Delete(k)
		}
		return true
	})

	s.broadcastSnapshot()
	go func() { _ = s.ensureGlobalProxy() }()

	return map[string]int64{
		"lanes":               appStateCount,
		"deploys":             deployCount,
		"deploy_requests":     deployRequestCount,
		"secrets":             secretCount,
		"service_credentials": credentialCount,
		"service_states":      serviceStateCount,
		"service_specs":       serviceSpecCount,
		"sync_sessions":       sessionCount,
		"runtime_services":    int64(len(services)),
	}, warnings, nil
}

// ---------------------- Main ----------------------

func main() {
	setupMemoryLimits()

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

	// Only the long-running server should have startup side effects —
	// `relayd version`/`relayd service` must not provision swapfiles or
	// spawn samplers.
	_ = maybeEnableAutoSwap()
	startMemorySampler()

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
	sqliteStore, err := openSQLiteStore(dbPath)
	if err != nil {
		panic(err)
	}
	db := sqliteStore.Primary
	analyticsDB := sqliteStore.Analytics
	relayDBKey, err := loadOrCreateRelayDBKey(dataDir)
	if err != nil {
		panic(fmt.Errorf("initialize RelayDB credential key: %w", err))
	}

	// Derive a 32-byte AES-256 key from RELAY_SECRET_KEY for encrypting secrets at rest.
	var secretKey []byte
	if rawKey := strings.TrimSpace(os.Getenv("RELAY_SECRET_KEY")); rawKey != "" {
		h := sha256.Sum256([]byte(rawKey))
		secretKey = h[:]
	}

	s := &Server{
		deploys:               make(map[string]*Deploy),
		queue:                 make(chan DeployJob, 1000),
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
		analyticsDB:           analyticsDB,
		relayDB:               newRelayDBManager(db, relayDBKey, hostTotalMemMB),
		apiToken:              apiToken,
		secretKey:             secretKey,
		buildpacks:            defaultBuildpacks(),
		buildCancels:          make(map[string]context.CancelFunc),
		webhookHits:           make(map[string][]time.Time),
		rolloutWatches:        make(map[string]struct{}),
		eventsChans:           make(map[chan []byte]struct{}),
		runtime:               &DockerRuntime{},
		stationRuntime:        newStationRuntime(dataDir),
	}
	s.corsOrigins, s.allowAllCORS = parseAllowedOrigins(os.Getenv("RELAY_CORS_ORIGINS"))
	if err := s.initCloud(runCfg); err != nil {
		panic(err)
	}

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
	_ = s.resumeRolloutWatches()
	_, _ = s.repairLegacyAppHostPortsFromRuntime()

	// Restore global domain proxy state from DB
	go func() { _ = s.ensureGlobalProxy() }()
	go superviseWorker("lane-expiry", s.runLaneExpiryWorker)
	go s.runHousekeepingWorker()
	go runGuarded("warm-images", s.warmBuildpackBaseImages)

	// Start worker pool: deploy jobs are I/O-bound (git, image pull/push,
	// container ops), so we can run more workers than CPU cores. On very small
	// hosts, cap the pool so two simultaneous Docker builds don't race for the
	// same RAM budget and OOM-kill each other.
	n := runtime.NumCPU()
	if n < 2 {
		n = 2
	}
	if total := hostTotalMemMB(); total > 0 && total <= 1024 {
		n = 1 // single build at a time on hosts ≤ 1 GB
	} else if total > 0 && total <= 2200 && n > 2 {
		n = 2 // cap at 2 on small hosts so builds don't race for RAM
	}
	s.worker(n)

	mux := http.NewServeMux()
	s.registerNetworkControlRoutes(mux)
	mux.Handle("/.well-known/acme-challenge/",
		http.StripPrefix("/.well-known/acme-challenge/",
			http.FileServer(http.Dir(s.acmeWebroot))))

	// UI
	uiRoot, _ := fs.Sub(uiFS, "ui")
	uiHandler := uiAssetHandler(uiRoot)
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", uiHandler))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dashboard/", http.StatusFound)
			return
		}
		// Fallback for SPA or other files if they are in root
		uiHandler.ServeHTTP(w, r)
	})

	// Unix-domain socket (local control transport)
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
	s.socketPath = socketPath
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
			// UI assets stay on TCP; local control routes share one registration module.
			sockMux := http.NewServeMux()
			s.registerLocalControlRoutes(sockMux)
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
	go superviseWorker("log-tailer", s.startLogTailer)
	if s.cloud != nil && s.cloud.agentEnabled {
		go superviseWorker("cloud-agent", s.runCloudAgentLoop)
	}
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
		req.Env = normalizeDeployEnv(string(req.Env))
		req.WorkspaceEnv = normalizeDeployEnv(string(req.WorkspaceEnv))
		req.WorkspaceBranch = strings.TrimSpace(req.WorkspaceBranch)
		req.Source = strings.TrimSpace(req.Source)
		if req.Source == "" {
			req.Source = "git"
		}
		if req.Source == "git" && strings.TrimSpace(req.RepoURL) == "" {
			if st, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && st != nil {
				req.RepoURL = strings.TrimSpace(st.RepoURL)
			}
		}
		if err := validateDeployRequest(req); err != nil {
			httpError(w, 400, err.Error())
			return
		}
		if req.Source == "git" && strings.TrimSpace(req.RepoURL) == "" {
			httpError(w, 400, "repo_url required for git deploys")
			return
		}
		if _, ok := s.requireLaneAccess(w, r, req.App, req.Env, "deployer"); !ok {
			return
		}
		if err := s.ensureBaselineLanes(req.App, req.Branch, req.Env, req.RepoURL, req.Engine); err != nil {
			httpError(w, 500, "failed to seed baseline lanes: "+err.Error())
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
		s.emitDeployNotificationsByID(deployID)

		s.queue <- DeployJob{ID: deployID, Req: req}
		s.broadcastSnapshot()
		writeJSON(w, 201, deploy)

	case http.MethodGet:
		out, err := s.listDeploysFromDB()
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if s.hasUsers() {
			sess := s.validateUserSession(r)
			if sess == nil {
				httpError(w, 401, "unauthorized")
				return
			}
			filtered := make([]*Deploy, 0, len(out))
			for _, deploy := range out {
				if deploy == nil {
					continue
				}
				if roleAtLeast(s.effectiveLaneRole(sess, deploy.App, deploy.Env), "viewer") {
					filtered = append(filtered, deploy)
				}
			}
			out = filtered
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
	if _, ok := s.requireLaneAccess(w, r, d.App, d.Env, "viewer"); !ok {
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) handleLogsByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
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
	Available bool   `json:"available"`
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
		if target.Kind == "app" && target.Live && target.Available {
			return target.ID
		}
	}
	for _, target := range targets {
		if target.Kind == "app" && target.Available {
			return target.ID
		}
	}
	for _, target := range targets {
		if target.Running {
			return target.ID
		}
	}
	for _, target := range targets {
		if target.Available {
			return target.ID
		}
	}
	return ""
}

func runtimeLogTargetStatus(runtime ContainerRuntime, container string) (running bool, available bool) {
	running = runtime.IsRunning(container)
	if running {
		return true, true
	}
	return false, runtime.ContainerExists(container)
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

	activeSlot := s.currentActiveSlotWithRuntime(s.runtime, app, env, branch, st)
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
		target.Running, target.Available = runtimeLogTargetStatus(s.runtime, target.Container)
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
		if s.runtime.ContainerExists(name) {
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
			if strings.EqualFold(svc.Type, serviceTypeRelayDB) {
				add(RuntimeLogTarget{
					ID:        "service:" + svc.Name + ":pool",
					Label:     fmt.Sprintf("RelayDB pool: %s", svc.Name),
					Kind:      "service",
					Container: relayDBPoolContainerName(svc.Container),
					Service:   svc.Name,
					Image:     getenv("RELAY_PGBOUNCER_IMAGE", defaultPgBouncerImage),
				})
			}
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
			return nil, targets, lane, errors.New(runtimeLogOfflineReason(lane, nil))
		}
	}
	for _, target := range targets {
		if target.ID == id {
			t := target
			if !t.Available {
				return &t, targets, lane, errors.New(runtimeLogOfflineReason(lane, &t))
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
	if !selected.Available || !runtime.ContainerExists(selected.Container) {
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
		s.broadcastEvent("update", []byte(`{}`))
	}()
}

func (s *Server) buildSnapshotJSONForSession(sess *UserSession) ([]byte, error) {
	rows, err := s.db.Query(
		`SELECT app, env, branch, COALESCE(engine,''), mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(public_hosts,''), COALESCE(active_slot,''), COALESCE(standby_slot,''), COALESCE(drain_until,0), COALESCE(traffic_mode,''), COALESCE(access_policy,''), COALESCE(ip_allowlist,''), COALESCE(expires_at,0), COALESCE(webhook_secret,''), COALESCE(notification_webhooks,''), COALESCE(traffic_split_percent,100), COALESCE(rollout_min_requests,25), COALESCE(rollout_error_percent,5), COALESCE(rollout_assess_seconds,300), COALESCE(rollout_started_at,0), COALESCE(rollout_deploy_id,''), COALESCE(rollout_status,''), repo_url, COALESCE(stopped,0)
		FROM app_state ORDER BY app, env, branch`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type snapEnv struct {
		App                  string   `json:"app"`
		Env                  string   `json:"env"`
		Branch               string   `json:"branch"`
		Engine               string   `json:"engine,omitempty"`
		Mode                 string   `json:"mode"`
		HostPort             int      `json:"host_port"`
		HostPortExplicit     bool     `json:"host_port_explicit,omitempty"`
		ServicePort          int      `json:"service_port"`
		PublicHost           string   `json:"public_host"`
		PublicHosts          []string `json:"public_hosts,omitempty"`
		ActiveSlot           string   `json:"active_slot,omitempty"`
		StandbySlot          string   `json:"standby_slot,omitempty"`
		DrainUntil           int64    `json:"drain_until,omitempty"`
		TrafficMode          string   `json:"traffic_mode,omitempty"`
		AccessPolicy         string   `json:"access_policy,omitempty"`
		IPAllowlist          string   `json:"ip_allowlist,omitempty"`
		ExpiresAt            int64    `json:"expires_at,omitempty"`
		WebhookSecret        string   `json:"webhook_secret,omitempty"`
		NotificationWebhooks string   `json:"notification_webhooks,omitempty"`
		TrafficSplitPercent  int      `json:"traffic_split_percent,omitempty"`
		RolloutMinRequests   int      `json:"rollout_min_requests,omitempty"`
		RolloutErrorPercent  float64  `json:"rollout_error_percent,omitempty"`
		RolloutAssessSeconds int      `json:"rollout_assess_seconds,omitempty"`
		RolloutStartedAt     int64    `json:"rollout_started_at,omitempty"`
		RolloutDeployID      string   `json:"rollout_deploy_id,omitempty"`
		RolloutStatus        string   `json:"rollout_status,omitempty"`
		RepoURL              string   `json:"repo_url"`
		Stopped              bool     `json:"stopped,omitempty"`
		Running              bool     `json:"running,omitempty"`
		AccessRole           string   `json:"access_role,omitempty"`
	}
	type snapProject struct {
		Name     string           `json:"name"`
		Envs     []snapEnv        `json:"envs"`
		Services []ProjectService `json:"services"`
	}

	byApp := map[string]*snapProject{}
	for rows.Next() {
		var pe snapEnv
		var publicHostsRaw string
		if err := rows.Scan(&pe.App, &pe.Env, &pe.Branch, &pe.Engine, &pe.Mode,
			&pe.HostPort, &pe.HostPortExplicit, &pe.ServicePort, &pe.PublicHost, &publicHostsRaw, &pe.ActiveSlot, &pe.StandbySlot, &pe.DrainUntil, &pe.TrafficMode, &pe.AccessPolicy, &pe.IPAllowlist, &pe.ExpiresAt, &pe.WebhookSecret, &pe.NotificationWebhooks, &pe.TrafficSplitPercent, &pe.RolloutMinRequests, &pe.RolloutErrorPercent, &pe.RolloutAssessSeconds, &pe.RolloutStartedAt, &pe.RolloutDeployID, &pe.RolloutStatus, &pe.RepoURL, &pe.Stopped); err != nil {
			continue
		}
		pe.PublicHosts = parsePublicHosts(publicHostsRaw)
		pe.PublicHost, pe.PublicHosts = canonicalizePublicHosts(pe.PublicHost, pe.PublicHosts)
		pe.Engine = firstNonEmptyEngine(pe.Engine)
		pe.TrafficMode = firstNonEmpty(normalizeTrafficMode(pe.TrafficMode), s.lanePolicy(DeployEnv(pe.Env)).DefaultTrafficMode)
		pe.AccessPolicy = firstNonEmpty(normalizeAccessPolicy(pe.AccessPolicy), s.lanePolicy(DeployEnv(pe.Env)).DefaultAccessPolicy)
		pe.Running = s.appLaneRunning(pe.App, DeployEnv(pe.Env), pe.Branch)
		pe.AccessRole = "owner"
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
		if sess != nil {
			filteredEnvs := make([]snapEnv, 0, len(v.Envs))
			for _, env := range v.Envs {
				role := s.effectiveLaneRole(sess, v.Name, DeployEnv(env.Env))
				if !roleAtLeast(role, "viewer") {
					continue
				}
				env.AccessRole = role
				filteredEnvs = append(filteredEnvs, env)
			}
			filteredServices := make([]ProjectService, 0, len(v.Services))
			for _, svc := range v.Services {
				if roleAtLeast(s.effectiveLaneRole(sess, v.Name, DeployEnv(svc.Env)), "viewer") {
					filteredServices = append(filteredServices, svc)
				}
			}
			v.Envs = filteredEnvs
			v.Services = filteredServices
			if len(v.Envs) == 0 && len(v.Services) == 0 {
				continue
			}
		}
		projects = append(projects, v)
	}

	deploys, err := s.listDeploysFromDB()
	if err != nil {
		return nil, err
	}
	if sess != nil {
		filtered := make([]*Deploy, 0, len(deploys))
		for _, deploy := range deploys {
			if deploy == nil {
				continue
			}
			if roleAtLeast(s.effectiveLaneRole(sess, deploy.App, deploy.Env), "viewer") {
				filtered = append(filtered, deploy)
			}
		}
		deploys = filtered
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
	var sess *UserSession
	if s.hasUsers() {
		sess = s.validateUserSession(r)
		if sess == nil {
			httpError(w, 401, "unauthorized")
			return
		}
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

	if snap, err := s.buildSnapshotJSONForSession(sess); err == nil {
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
			if strings.HasPrefix(string(msg), "event: update") {
				if snap, err := s.buildSnapshotJSONForSession(sess); err == nil {
					fmt.Fprintf(w, "event: update\ndata: %s\n\n", snap)
					flusher.Flush()
				}
				continue
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
	if _, ok := s.requireLaneAccess(w, r, d.App, d.Env, "viewer"); !ok {
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
	req.Env = normalizeDeployEnv(string(req.Env))
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}
	if _, ok := s.requireLaneAccess(w, r, req.App, req.Env, "deployer"); !ok {
		return
	}

	// Workspace conflict check: if the caller has a known base version, verify
	// it still matches the server's current workspace. A mismatch means someone
	// else deployed in between and the caller needs to run "relay pull" first.
	if req.BaseVersion != "" {
		if st, err := s.getAppState(req.App, req.Env, req.Branch); err == nil && st != nil && st.RepoHash != "" {
			if req.BaseVersion != st.RepoHash {
				// If the caller has changed its local workspace since the last saved
				// base version, treat the local tree as the source of truth and allow
				// the sync to proceed without forcing a separate retry.
				if !req.LocalChanged {
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

func isIgnoredWorkspaceEnvPath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(rel), "./"))
	if rel == "" {
		return false
	}
	base := strings.ToLower(strings.TrimSpace(path.Base(rel)))
	if base == ".env" {
		return true
	}
	if !strings.HasPrefix(base, ".env.") {
		return false
	}
	suffix := strings.TrimPrefix(base, ".env.")
	if suffix == "" {
		return true
	}
	switch suffix {
	case "example", "sample", "template", "defaults":
		return false
	}
	if strings.HasSuffix(suffix, ".example") || strings.HasSuffix(suffix, ".sample") || strings.HasSuffix(suffix, ".template") {
		return false
	}
	return true
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
		if isIgnoredWorkspaceEnvPath(p) {
			continue
		}
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
	if isIgnoredWorkspaceEnvPath(rel) {
		httpError(w, 400, "environment files are ignored during workspace sync; add variables in admin secrets")
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
	if err := s.saveSessionToDB(sess); err != nil {
		fmt.Fprintf(os.Stderr, "saveSessionToDB %s: %v\n", sess.ID, err)
	}

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
		if isIgnoredWorkspaceEnvPath(name) {
			wn, err := io.Copy(io.Discard, tr)
			if err != nil {
				httpError(w, 400, "invalid bundle entry")
				return
			}
			totalWritten += wn
			if totalWritten > remaining {
				httpError(w, 413, "bundle exceeds session remaining quota")
				return
			}
			continue
		}

		if hdr.Size > (remaining - totalWritten) {
			httpError(w, 413, "bundle entry exceeds session remaining quota")
			return
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			mustMkdir(filepath.Join(sess.StagingDir, filepath.FromSlash(name)))
			continue
		case tar.TypeReg:
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
			if err := s.saveAppState(st); err != nil {
				fmt.Fprintf(os.Stderr, "handleSyncComplete saveAppState %s: %v\n", sess.App, err)
			}
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
	if err := s.ensureBaselineLanes(sess.App, sess.Branch, sess.Env, "", req.Engine); err != nil {
		httpError(w, 500, "failed to seed baseline lanes: "+err.Error())
		return
	}

	s.mu.Lock()
	s.deploys[deployID] = deploy
	s.mu.Unlock()

	if err := s.saveDeployToDB(deploy, req); err != nil {
		fmt.Fprintf(os.Stderr, "saveDeployToDB %s: %v\n", deployID, err)
	}
	s.emitDeployNotificationsByID(deployID)
	s.auditLog(deployedBy, "deploy.trigger", sess.App, fmt.Sprintf("build #%d env=%s branch=%s", buildNum, sess.Env, sess.Branch))
	s.queue <- DeployJob{ID: deployID, Req: req}
	s.broadcastSnapshot()

	s.syncMu.Lock()
	delete(s.syncSessions, sessionID)
	s.syncMu.Unlock()
	if err := s.deleteSessionFromDB(sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "deleteSessionFromDB %s: %v\n", sessionID, err)
	}

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
	// Drop repos whose hits have all aged out, so the map never accumulates
	// one permanent entry per repo URL ever seen.
	if len(s.webhookHits) > 64 {
		for k, v := range s.webhookHits {
			if k == repoURL {
				continue
			}
			if len(v) == 0 || !v[len(v)-1].After(cutoff) {
				delete(s.webhookHits, k)
			}
		}
	}
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

	// Read body once for signature verification and JSON decode. Bound the
	// read so a hostile or misbehaving sender can't balloon the daemon's heap;
	// GitHub itself caps hook payloads at 25 MB and real ones are far smaller.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, 400, "failed to read body")
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

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

	query := `SELECT app, env, branch, mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(webhook_secret,''), COALESCE(git_token,'') FROM app_state WHERE repo_url IN (` +
		strings.Repeat("?,", len(repoURLs)-1) + `?) AND branch=? AND (stopped IS NULL OR stopped=0)`
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
		var envS, webhookSecret, gitToken string
		if err := rows.Scan(&st.App, &envS, &st.Branch, &st.Mode, &st.HostPort, &st.HostPortExplicit, &st.ServicePort, &st.PublicHost, &webhookSecret, &gitToken); err != nil {
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

		if !s.webhookAllowed(payload.Repository.CloneURL) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"status": "rate_limited", "triggered": 0})
			return
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
			GitToken:         gitToken,
		}

		s.mu.Lock()
		s.deploys[deployID] = deploy
		s.mu.Unlock()

		if err := s.saveDeployToDB(deploy, req); err != nil {
			fmt.Fprintf(os.Stderr, "saveDeployToDB %s: %v\n", deployID, err)
		}
		s.emitDeployNotificationsByID(deployID)
		shortSHA := payload.After
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
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
	App          string    `json:"app"`
	Branch       string    `json:"branch"`
	Env          DeployEnv `json:"env"`
	SourceEnv    DeployEnv `json:"source_env,omitempty"`
	SourceBranch string    `json:"source_branch,omitempty"`
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
	req.Env = normalizeDeployEnv(string(req.Env))
	req.SourceEnv = normalizeDeployEnv(string(req.SourceEnv))
	if req.SourceEnv != "" && strings.TrimSpace(req.SourceBranch) == "" {
		req.SourceBranch = req.Branch
	}
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}
	if req.SourceEnv == "" && strings.TrimSpace(req.SourceBranch) != "" {
		httpError(w, 400, "source_env is required when source_branch is set")
		return
	}
	if req.SourceEnv != "" && !validDeployTarget(req.App, req.SourceEnv, req.SourceBranch) {
		httpError(w, 400, "source_env and source_branch must target an existing lane")
		return
	}
	if _, ok := s.requireLaneAccess(w, r, req.App, req.Env, "deployer"); !ok {
		return
	}
	if req.SourceEnv != "" {
		if _, ok := s.requireLaneAccess(w, r, req.App, req.SourceEnv, "deployer"); !ok {
			return
		}
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

// handleRolloutAbort lets a user manually flip an in-progress canary/blue-green
// rollout back to the previously-active slot immediately, instead of waiting
// out rollout_assess_seconds for the automated health check in
// startRolloutWatch. It's the fast path (an existing container swap, not a
// full redeploy) — handleRollback (POST /api/deploys/rollback) is the
// separate, heavier "redeploy a past image" flow and doesn't touch an
// in-flight traffic split the way this does.
func (s *Server) handleRolloutAbort(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req AppActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	req.Env = normalizeDeployEnv(string(req.Env))
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}
	if _, ok := s.requireLaneAccess(w, r, req.App, req.Env, "deployer"); !ok {
		return
	}
	st, err := s.getAppState(req.App, req.Env, req.Branch)
	if err != nil || st == nil {
		httpError(w, 404, "app not found")
		return
	}
	if normalizeActiveSlot(st.StandbySlot) == "" || strings.TrimSpace(st.RolloutStatus) != "monitoring" {
		httpError(w, 400, "no active rollout to abort")
		return
	}
	if err := s.rollbackCanary(req.App, req.Env, req.Branch, "manually aborted by "+requestActorLabel(s, r)); err != nil {
		httpError(w, 500, "rollback failed: "+err.Error())
		return
	}
	st, err = s.getAppState(req.App, req.Env, req.Branch)
	if err != nil || st == nil {
		writeJSON(w, 200, map[string]string{"status": "rolled_back"})
		return
	}
	writeJSON(w, 200, st)
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
	if deploy, err := s.getDeployFromDB(id); err == nil && deploy != nil {
		if _, ok := s.requireLaneAccess(w, r, deploy.App, deploy.Env, "deployer"); !ok {
			return
		}
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

type LaneDeleteRequest struct {
	App     string    `json:"app"`
	Branch  string    `json:"branch"`
	Env     DeployEnv `json:"env"`
	Confirm string    `json:"confirm"`
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
	req.Env = normalizeDeployEnv(string(req.Env))
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
	req.Env = normalizeDeployEnv(string(req.Env))
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
			companions, companionErr := s.orchestrateCompanions(
				logf,
				req.App,
				req.Env,
				req.Branch,
				desiredServices,
				false,
				true,
				companionFailureFatal,
			)
			if companionErr != nil {
				httpError(w, 500, companionErr.Error())
				return
			}
			networkName = companions.NetworkName
			for key, value := range companions.Environment {
				extraEnv[key] = value
			}
		}
		s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, extraEnv, nil)
		if err := s.runContainer(nil, st.App, st.Env, st.Branch, st.CurrentImage, st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, networkName, extraEnv); err != nil {
			httpError(w, 500, err.Error())
			return
		}
	} else {
		s.stopDockerAppLane(req.App, req.Env, req.Branch)
		s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, extraEnv, nil)
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
	req.Env = normalizeDeployEnv(string(req.Env))
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
			companions, companionErr := s.orchestrateCompanions(
				logf,
				req.App,
				req.Env,
				req.Branch,
				desiredServices,
				false,
				true,
				companionFailureFatal,
			)
			if companionErr != nil {
				httpError(w, 500, companionErr.Error())
				return
			}
			networkName = companions.NetworkName
			for key, value := range companions.Environment {
				extraEnv[key] = value
			}
		}
		s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, extraEnv, nil)
		if err := s.runContainer(nil, st.App, st.Env, st.Branch, st.CurrentImage, st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, networkName, extraEnv); err != nil {
			httpError(w, 500, err.Error())
			return
		}
	} else {
		s.stopDockerAppLane(req.App, req.Env, req.Branch)
		s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, extraEnv, nil)
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
			App                  string    `json:"app"`
			Env                  DeployEnv `json:"env"`
			Branch               string    `json:"branch"`
			RepoURL              *string   `json:"repo_url"`
			ProjectRoot          *string   `json:"project_root"`
			BuildContext         *string   `json:"build_context"`
			Dockerfile           *string   `json:"dockerfile"`
			Engine               *string   `json:"engine"`
			Mode                 *string   `json:"mode"`
			TrafficMode          *string   `json:"traffic_mode"`
			AccessPolicy         *string   `json:"access_policy"`
			IPAllowlist          *string   `json:"ip_allowlist"`
			HostPort             *int      `json:"host_port"`
			ServicePort          *int      `json:"service_port"`
			PublicHost           *string   `json:"public_host"`
			PublicHosts          *[]string `json:"public_hosts"`
			WebhookSecret        *string   `json:"webhook_secret"`
			NotificationWebhooks *string   `json:"notification_webhooks"`
			TrafficSplitPercent  *int      `json:"traffic_split_percent"`
			RolloutMinRequests   *int      `json:"rollout_min_requests"`
			RolloutErrorPercent  *float64  `json:"rollout_error_percent"`
			RolloutAssessSeconds *int      `json:"rollout_assess_seconds"`
			CPULimit             *string   `json:"cpu_limit"`
			MemLimit             *string   `json:"mem_limit"`
			ResourceMode         *string   `json:"resource_mode"`
			Volumes              *[]string `json:"volumes"`
			GitToken             *string   `json:"git_token"`
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
		if _, ok := s.requireLaneAccess(w, r, body.App, body.Env, "deployer"); !ok {
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
		if body.ProjectRoot != nil {
			value, err := cleanRepoRelativePath(*body.ProjectRoot)
			if err != nil {
				httpError(w, 400, "project_root "+err.Error())
				return
			}
			st.ProjectRoot = value
		}
		if body.BuildContext != nil {
			value, err := cleanRepoRelativePath(*body.BuildContext)
			if err != nil {
				httpError(w, 400, "build_context "+err.Error())
				return
			}
			st.BuildContext = value
		}
		if body.Dockerfile != nil {
			value, err := cleanRepoRelativePath(*body.Dockerfile)
			if err != nil {
				httpError(w, 400, "dockerfile "+err.Error())
				return
			}
			st.Dockerfile = value
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
		if body.PublicHosts != nil {
			primary, hosts := canonicalizePublicHosts(st.PublicHost, *body.PublicHosts)
			if err := validateAppPublicHosts(hosts, normalizedRequestHost(r), s.serverDashboardHost()); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			st.PublicHost = primary
			st.PublicHosts = hosts
		}
		st.PublicHost, st.PublicHosts = canonicalizePublicHosts(st.PublicHost, st.PublicHosts)
		if body.WebhookSecret != nil {
			st.WebhookSecret = strings.TrimSpace(*body.WebhookSecret)
		}
		if body.NotificationWebhooks != nil {
			st.NotificationWebhooks = normalizeNotificationWebhooks(*body.NotificationWebhooks)
		}
		if body.TrafficSplitPercent != nil {
			if *body.TrafficSplitPercent < 1 || *body.TrafficSplitPercent > 100 {
				httpError(w, 400, "traffic_split_percent must be between 1 and 100")
				return
			}
			st.TrafficSplitPercent = *body.TrafficSplitPercent
		}
		if body.RolloutMinRequests != nil {
			if *body.RolloutMinRequests < 1 {
				httpError(w, 400, "rollout_min_requests must be at least 1")
				return
			}
			st.RolloutMinRequests = *body.RolloutMinRequests
		}
		if body.RolloutErrorPercent != nil {
			if *body.RolloutErrorPercent <= 0 || *body.RolloutErrorPercent > 100 {
				httpError(w, 400, "rollout_error_percent must be greater than 0 and at most 100")
				return
			}
			st.RolloutErrorPercent = *body.RolloutErrorPercent
		}
		if body.RolloutAssessSeconds != nil {
			if *body.RolloutAssessSeconds < 30 {
				httpError(w, 400, "rollout_assess_seconds must be at least 30")
				return
			}
			st.RolloutAssessSeconds = *body.RolloutAssessSeconds
		}
		if body.CPULimit != nil {
			st.CPULimit = strings.TrimSpace(*body.CPULimit)
		}
		if body.MemLimit != nil {
			st.MemLimit = strings.TrimSpace(*body.MemLimit)
		}
		if body.ResourceMode != nil {
			mode := strings.TrimSpace(*body.ResourceMode)
			if mode != "" && mode != "manual" && mode != "auto" {
				httpError(w, 400, "resource_mode must be manual or auto")
				return
			}
			st.ResourceMode = mode
		}
		if body.Volumes != nil {
			cleaned := make([]string, 0, len(*body.Volumes))
			for _, v := range *body.Volumes {
				if t := strings.TrimSpace(v); t != "" {
					cleaned = append(cleaned, t)
				}
			}
			st.Volumes = cleaned
		}
		if body.GitToken != nil {
			st.GitToken = strings.TrimSpace(*body.GitToken)
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
			proxyNeedsRefresh = len(st.PublicHosts) > 0
		} else {
			proxyNeedsRefresh = !samePublicHosts(previousState.PublicHosts, st.PublicHosts) || previousState.HostPort != st.HostPort
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
		if _, ok := s.requireLaneAccess(w, r, app, e, "viewer"); !ok {
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
		// Marshal AppState then inject git_token_set so the token itself is never in responses.
		stBytes, _ := json.Marshal(st)
		var stMap map[string]any
		_ = json.Unmarshal(stBytes, &stMap)
		stMap["git_token_set"] = st.GitToken != ""
		writeJSON(w, 200, stMap)

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
		if _, ok := s.requireLaneAccess(w, r, app, env, "viewer"); !ok {
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
		if _, ok := s.requireLaneAccess(w, r, body.App, body.Env, "deployer"); !ok {
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
			logf := func(string, ...any) {}
			if _, err := s.orchestrateCompanions(logf, body.App, body.Env, body.Branch, []ServiceConfig{cfg}, false, false, companionFailureFatal); err != nil {
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
		if _, ok := s.requireLaneAccess(w, r, app, DeployEnv(env), "deployer"); !ok {
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
	if _, ok := s.requireLaneAccess(w, r, body.App, body.Env, "deployer"); !ok {
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
	logf := func(string, ...any) {}
	if _, err := s.orchestrateCompanions(logf, body.App, body.Env, body.Branch, []ServiceConfig{*selected}, true, false, companionFailureFatal); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	s.broadcastSnapshot()
	writeJSON(w, 200, map[string]string{"status": "restarted"})
}

func validSecretEnvKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		ch := key[i]
		isAlpha := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
		isDigit := ch >= '0' && ch <= '9'
		if i == 0 {
			if !isAlpha && ch != '_' {
				return false
			}
			continue
		}
		if !isAlpha && !isDigit && ch != '_' {
			return false
		}
	}
	return true
}

func parseRawSecretEnvText(raw string) (map[string]string, error) {
	out := map[string]string{}
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			return nil, fmt.Errorf("line %d must be KEY=VALUE", idx+1)
		}
		key := strings.TrimSpace(line[:eq])
		if !validSecretEnvKey(key) {
			return nil, fmt.Errorf("line %d has invalid key %q", idx+1, key)
		}
		value := strings.TrimSpace(line[eq+1:])
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no KEY=VALUE pairs found")
	}
	return out, nil
}

func (s *Server) handleAppSecrets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app := r.URL.Query().Get("app")
		env := normalizeDeployEnv(r.URL.Query().Get("env"))
		branch := r.URL.Query().Get("branch")
		if !validDeployTarget(app, env, branch) {
			httpError(w, 400, "app, env, and branch required")
			return
		}
		if _, ok := s.requireLaneAccess(w, r, app, env, "viewer"); !ok {
			return
		}

		query := `SELECT app, env, branch, key, value FROM app_secrets WHERE app=?`
		args := []any{app}
		if env != "" {
			query += ` AND (env=? OR env='')`
			args = append(args, string(env))
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
		var body struct {
			App     string `json:"app"`
			Env     string `json:"env"`
			Branch  string `json:"branch"`
			Key     string `json:"key"`
			Value   string `json:"value"`
			Raw     string `json:"raw"`
			Replace bool   `json:"replace"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		body.Env = string(normalizeDeployEnv(body.Env))
		body.App = strings.TrimSpace(body.App)
		body.Branch = strings.TrimSpace(body.Branch)
		body.Key = strings.TrimSpace(body.Key)
		raw := strings.TrimSpace(body.Raw)
		if body.App == "" || !isKnownDeployEnv(DeployEnv(body.Env)) || body.Branch == "" {
			httpError(w, 400, "app, env, and branch required")
			return
		}
		if _, ok := s.requireLaneAccess(w, r, body.App, DeployEnv(body.Env), "deployer"); !ok {
			return
		}
		actor := ""
		if userSess := s.validateUserSession(r); userSess != nil {
			actor = userSess.Username
		}
		if raw != "" {
			pairs, err := parseRawSecretEnvText(raw)
			if err != nil {
				httpError(w, 400, "invalid raw env: "+err.Error())
				return
			}
			tx, err := s.db.Begin()
			if err != nil {
				httpError(w, 500, "failed to start secret transaction: "+err.Error())
				return
			}
			if body.Replace {
				if _, err := tx.Exec(`DELETE FROM app_secrets WHERE app=? AND env=? AND branch=?`, body.App, body.Env, body.Branch); err != nil {
					_ = tx.Rollback()
					httpError(w, 500, "failed to replace secrets: "+err.Error())
					return
				}
			}
			keys := make([]string, 0, len(pairs))
			for key := range pairs {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				encrypted := s.encryptSecret(pairs[key])
				if _, err := tx.Exec(`INSERT OR REPLACE INTO app_secrets (app, env, branch, key, value) VALUES (?, ?, ?, ?, ?)`,
					body.App, body.Env, body.Branch, key, encrypted); err != nil {
					_ = tx.Rollback()
					httpError(w, 500, "failed to save secret: "+err.Error())
					return
				}
			}
			if err := tx.Commit(); err != nil {
				httpError(w, 500, "failed to commit secrets: "+err.Error())
				return
			}
			s.auditLog(actor, "secret.bulk_set", body.App, fmt.Sprintf("count=%d env=%s branch=%s replace=%v", len(keys), body.Env, body.Branch, body.Replace))
			writeJSON(w, 200, map[string]any{"ok": true, "count": len(keys), "replace": body.Replace})
			return
		}
		if body.Key == "" {
			httpError(w, 400, "key required")
			return
		}
		if !validSecretEnvKey(body.Key) {
			httpError(w, 400, "key must match [A-Za-z_][A-Za-z0-9_]*")
			return
		}
		encrypted := s.encryptSecret(body.Value)
		if _, err := s.db.Exec(`INSERT OR REPLACE INTO app_secrets (app, env, branch, key, value) VALUES (?, ?, ?, ?, ?)`,
			body.App, body.Env, body.Branch, body.Key, encrypted); err != nil {
			httpError(w, 500, "failed to save secret: "+err.Error())
			return
		}
		// Audit secret writes (log key name only, never the value)
		s.auditLog(actor, "secret.set", body.App, fmt.Sprintf("key=%s env=%s branch=%s", body.Key, body.Env, body.Branch))
		writeJSON(w, 200, map[string]bool{"ok": true})

	case http.MethodDelete:
		app := r.URL.Query().Get("app")
		env := normalizeDeployEnv(r.URL.Query().Get("env"))
		branch := r.URL.Query().Get("branch")
		key := r.URL.Query().Get("key")
		if !validDeployTarget(app, env, branch) || key == "" {
			httpError(w, 400, "app, env, branch, and key required")
			return
		}
		if _, ok := s.requireLaneAccess(w, r, app, env, "deployer"); !ok {
			return
		}

		_, err := s.db.Exec(`DELETE FROM app_secrets WHERE app=? AND env=? AND branch=? AND key=?`,
			app, string(env), branch, key)
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

func (s *Server) mergeLaneSecretsIntoEnv(app string, env DeployEnv, branch string, target map[string]string, log func(string, ...any)) int {
	if target == nil {
		return 0
	}
	secs, err := s.getAppSecrets(app, env, branch)
	if err != nil {
		if log != nil {
			log("warning: could not load admin secrets: %v", err)
		}
		return 0
	}
	for k, v := range secs {
		target[k] = v
	}
	return len(secs)
}

func scrubWorkspaceEnvFiles(repoDir string) (int, error) {
	removed := 0
	walkErr := filepath.Walk(repoDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info == nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(repoDir, p)
		if relErr != nil {
			return nil
		}
		if !isIgnoredWorkspaceEnvPath(rel) {
			return nil
		}
		if rmErr := os.Remove(p); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return rmErr
		}
		removed++
		return nil
	})
	if walkErr != nil {
		return removed, walkErr
	}
	return removed, nil
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

	// If already building this specific app/env/branch, retry later without
	// blocking a goroutine on a full queue.
	if _, loaded := s.building.LoadOrStore(lockKey, true); loaded {
		s.requeueDeployLater(job, 5*time.Second)
		return
	}
	defer s.building.Delete(lockKey)

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
		if req.GitToken == "" {
			req.GitToken = state.GitToken
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
	s.resolvePortConflict(s.runtimeForEngine(engine), &req, log)

	// Resolve workspace path.
	workspaceEnv := normalizeDeployEnv(string(req.WorkspaceEnv))
	workspaceBranch := strings.TrimSpace(req.WorkspaceBranch)
	if workspaceBranch == "" {
		workspaceEnv = req.Env
		workspaceBranch = req.Branch
	}
	workspace := filepath.Join(s.workspacesDir, fmt.Sprintf("%s__%s__%s", safe(req.App), safe(string(req.Env)), safe(req.Branch)))
	repoDir := s.workspaceRepoDir(req.App, workspaceEnv, workspaceBranch)
	if req.Source == "workspace" && !fileExists(repoDir) {
		failDeploy(s, d, fmt.Errorf("workspace not found"), fmt.Sprintf("workspace source %s/%s is not available on this server", workspaceEnv, workspaceBranch))
		return
	}
	if req.Source != "workspace" {
		mustMkdir(repoDir)
	}

	// If source is git, ensure repo is up to date
	if (req.Source == "git" || strings.TrimSpace(job.PromoteImage) != "") && req.RepoURL != "" {
		repoDir = filepath.Join(workspace, "repo")
		mustMkdir(repoDir)
		log("source is git: preparing repo from %s [%s]", req.RepoURL, req.Branch)

		// Build an authenticated clone URL if a git token is set. The token is
		// injected as the URL userinfo so git uses it for HTTPS auth without
		// needing a credential helper. req.RepoURL is kept clean for logs.
		cloneURL := req.RepoURL
		if req.GitToken != "" {
			if u, err := url.Parse(req.RepoURL); err == nil && (u.Scheme == "https" || u.Scheme == "http") {
				u.User = url.UserPassword("x-access-token", req.GitToken)
				cloneURL = u.String()
			}
		}

		if !fileExists(filepath.Join(repoDir, ".git")) {
			log("cloning repository...")
			_ = os.RemoveAll(repoDir)
			if err := runCmdLoggedCtx(buildCtx, workspace, logf, "git", "clone", "--depth", "1", "--branch", req.Branch, cloneURL, "repo"); err != nil {
				failDeploy(s, d, err, "git clone failed: "+err.Error())
				return
			}
		} else {
			log("updating existing repository...")
			if err := runCmdLoggedCtx(buildCtx, repoDir, logf, "git", "fetch", "--depth=1", "origin", req.Branch); err != nil {
				log("fetch failed, attempting clean clone: %v", err)
				_ = os.RemoveAll(repoDir)
				if err := runCmdLoggedCtx(buildCtx, workspace, logf, "git", "clone", "--depth", "1", "--branch", req.Branch, cloneURL, "repo"); err != nil {
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
	if removedEnvFiles, scrubErr := scrubWorkspaceEnvFiles(repoDir); scrubErr != nil {
		log("warning: failed to sanitize workspace env files: %v", scrubErr)
	} else if removedEnvFiles > 0 {
		log("ignored %d workspace env file(s); use admin secrets instead", removedEnvFiles)
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
		s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, extraEnv, log)
		var runErr error
		if rollbackEngine == EngineStation {
			runErr = s.runStationApp(log, req, job.RollbackImage, extraEnv)
		} else {
			runErr = s.swapContainer(log, req, job.RollbackImage, "", extraEnv)
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
		latestState, _ := s.getAppState(req.App, req.Env, req.Branch)
		if latestState == nil {
			latestState = state
		}
		nextState := buildRollbackLaneState(rollbackLaneTransition{
			Request:       req,
			Current:       latestState,
			Engine:        rollbackEngine,
			CurrentImage:  job.RollbackImage,
			PreviousImage: prevCurrent,
			ActiveSlotFallback: s.currentActiveSlotWithRuntime(
				s.runtimeForEngine(rollbackEngine),
				req.App,
				req.Env,
				req.Branch,
				latestState,
			),
		})
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

	effectiveProjectRoot := ""
	effectiveBuildContext := ""
	effectiveDockerfile := ""
	if state != nil {
		effectiveProjectRoot = strings.TrimSpace(state.ProjectRoot)
		effectiveBuildContext = strings.TrimSpace(state.BuildContext)
		effectiveDockerfile = strings.TrimSpace(state.Dockerfile)
	}
	if effectiveProjectRoot == "" && cfg != nil {
		effectiveProjectRoot = strings.TrimSpace(cfg.ProjectRoot)
	}
	if effectiveBuildContext == "" && cfg != nil {
		effectiveBuildContext = strings.TrimSpace(cfg.BuildContext)
	}
	if effectiveDockerfile == "" && cfg != nil {
		effectiveDockerfile = strings.TrimSpace(cfg.Dockerfile)
	}

	projectRootDir, _, err := resolveRepoRelativePath(repoDir, effectiveProjectRoot)
	if err != nil {
		end := time.Now()
		d.Status = StatusFailed
		d.EndedAt = &end
		d.Error = "invalid project_root: " + err.Error()
		_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
		return
	}
	buildContextDir := projectRootDir
	if effectiveBuildContext != "" {
		buildContextDir, _, err = resolveRepoRelativePath(repoDir, effectiveBuildContext)
		if err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "invalid build_context: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
	}

	// If the repo supplies a Dockerfile (explicitly in relay.config.json or
	// directly in the selected app root/build context), treat it as the source of truth and bypass
	// buildpack detection.
	var plan BuildPlan
	customDockerfile := ""
	if effectiveDockerfile != "" {
		customDockerfile = effectiveDockerfile
	} else {
		customDockerfile = detectUserDockerfile(repoDir, projectRootDir, buildContextDir)
	}
	if customDockerfile != "" {
		cfgServicePort := 0
		cfgBuildImage := ""
		cfgRunImage := ""
		cfgInstallCmd := ""
		cfgBuildCmd := ""
		cfgStartCmd := ""
		if cfg != nil {
			cfgServicePort = cfg.ServicePort
			cfgBuildImage = cfgStr(cfg, "BuildImage")
			cfgRunImage = cfgStr(cfg, "RunImage")
			cfgInstallCmd = cfg.InstallCmd
			cfgBuildCmd = cfg.BuildCmd
			cfgStartCmd = cfg.StartCmd
		}
		// Resolve dockerfile path and ensure it's inside repoDir
		df := customDockerfile
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
			Kind:            "custom-dockerfile",
			ServicePort:     firstNonZero(req.ServicePort, cfgServicePort),
			BuildImage:      firstNonEmpty(cfgBuildImage, ""),
			RunImage:        firstNonEmpty(cfgRunImage, ""),
			InstallCmd:      firstNonEmpty(cfgInstallCmd, req.InstallCmd),
			BuildCmd:        firstNonEmpty(cfgBuildCmd, req.BuildCmd),
			StartCmd:        firstNonEmpty(cfgStartCmd, req.StartCmd),
			WriteDockerfile: nil,
			Verify:          nil,
		}
	} else {
		// Select buildpack (ConfigBuildpack has priority if cfg exists). Keep
		// scanning after the first match so an ambiguous repo (e.g. both
		// go.mod and package.json at the root) is visible in the build log
		// instead of being silently resolved by list order.
		var pack Buildpack
		var matchedNames []string
		for _, bp := range s.buildpacks {
			ok := bp.Detect(projectRootDir, cfg)
			log("buildpack detect: %s -> %v", bp.Name(), ok)
			if ok {
				matchedNames = append(matchedNames, bp.Name())
				if pack == nil {
					pack = bp
				}
			}
		}
		if len(matchedNames) > 1 {
			log("warning: %d buildpacks matched (%s) — selected %q by priority order; set \"kind\" in relay.config.json to pin this explicitly if that's wrong", len(matchedNames), strings.Join(matchedNames, ", "), pack.Name())
		}
		if pack == nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "no buildpack matched (add relay.config.json or a recognizable project file)." + monorepoSubdirHint(projectRootDir, s.buildpacks, cfg)
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}

		log("selected buildpack: %s", pack.Name())

		var err error
		plan, err = pack.Plan(req, projectRootDir, cfg)
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
		if err := plan.WriteDockerfile(projectRootDir); err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "dockerfile error: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
		defer cleanupGeneratedBuildpackFiles(projectRootDir)
	}

	dockerfilePath := filepath.Join(projectRootDir, "Dockerfile")
	if customDockerfile != "" {
		dockerfilePath = customDockerfile
		if !filepath.IsAbs(dockerfilePath) {
			dockerfilePath = filepath.Join(repoDir, dockerfilePath)
		}
	}

	buildEnv := map[string]string{}
	buildSecretCount := s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, buildEnv, log)
	buildEnvB64 := encodeBuildEnvArgValue(buildEnv)
	if buildSecretCount > 0 {
		log("build env: injected %d lane secret(s) into build steps", buildSecretCount)
	}

	repoHash := buildInputFingerprint(buildContextDir, buildEnv)
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
		stationDockerfilePath, cleanupBuildDockerfile, err := prepareBuildDockerfileWithEnv(buildContextDir, dockerfilePath, buildEnvB64, true)
		if err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "station build prep failed: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
		defer cleanupBuildDockerfile()
		if _, err := s.buildStationSnapshot(buildCtx, buildContextDir, stationDockerfilePath, artifactRef, logf); err != nil {
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

		// Pre-build disk check: if available space is below 2 GB, trigger an
		// immediate housekeeping pass before the build fills the disk. Docker
		// image layers can quietly consume tens of GB without an explicit prune,
		// and a full disk turns a successful build into a hard failure.
		if avail := diskAvailableMB(s.dataDir); avail > 0 && avail < 2048 {
			log("pre-build: only %d MB disk available — running housekeeping pass", avail)
			s.pruneDeployImages()
			// BuildKit build cache is the largest silent consumer and is not
			// touched by image pruning. Try a bounded prune (keeps the newest
			// RELAY_BUILD_CACHE_KEEP_GB) first — that's usually enough to clear
			// the low-disk condition without throwing away cache that would
			// otherwise make the next build of the SAME app fast. Only fall
			// back to wiping the entire cache if that wasn't enough.
			s.pruneBuildCache(false)
			if after := diskAvailableMB(s.dataDir); after == 0 || after < 2048 {
				s.pruneBuildCache(true)
			}
			if after := diskAvailableMB(s.dataDir); after > 0 {
				log("pre-build: %d MB available after prune", after)
			}
		}

		log("docker build starting...")
		if total := hostTotalMemMB(); total > 0 && total <= 2200 && hostSwapTotalMB() == 0 {
			// Last-chance swap provisioning: covers hosts where startup ran
			// before the operator fixed permissions or where the swapfile
			// was removed since. Failures are remembered, so this does not
			// redo expensive work on every build.
			swapErr := maybeEnableAutoSwap()
			if swapMB := hostSwapTotalMB(); swapMB > 0 {
				log("auto-swap: %d MB of swap active before build", swapMB)
			} else {
				if swapErr != nil {
					log("auto-swap: %v", swapErr)
				}
				// Wording must not contain the literal "(Turbopack)" — the
				// OOM hint below greps this same log for that marker.
				log("warning: host has %d MB RAM and no swap — Node build heap is capped at %s MB, but native bundler memory ignores that cap and can be OOM-killed", total, nodeBuildHeapMB())
				log("hint: add a 2 GB swapfile to make OOM kills effectively impossible: fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile")
			}
		}

		artifactRef = fmt.Sprintf("relay/%s:%s-%s-%s",
			safe(req.App),
			safe(string(req.Env)),
			safe(req.Branch),
			d.ID,
		)

		buildDockerfilePath := ""
		if customDockerfile != "" {
			buildDockerfilePath = dockerfilePath
		}
		wrappedDockerfilePath, cleanupBuildDockerfile, err := prepareBuildDockerfileWithEnv(buildContextDir, buildDockerfilePath, buildEnvB64, false)
		if err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			d.Error = "docker build prep failed: " + err.Error()
			_ = s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", "")
			return
		}
		defer cleanupBuildDockerfile()
		var buildArgs map[string]string
		if buildEnvB64 != "" {
			buildArgs = map[string]string{relayBuildEnvArg: buildEnvB64}
		}
		if err := s.runtime.Build(buildCtx, artifactRef, buildContextDir, wrappedDockerfilePath, buildArgs, logf, plan.Kind); err != nil {
			end := time.Now()
			d.Status = StatusFailed
			d.EndedAt = &end
			errMsg := err.Error()
			if strings.Contains(strings.ToLower(errMsg), "no space left on device") {
				log("build host out of disk space — reclaiming build cache and dangling images so the next deploy can proceed")
				s.pruneDeployImages()
				s.pruneBuildCache(true)
				if after := diskAvailableMB(s.dataDir); after > 0 {
					log("post-failure prune: %d MB disk available now — retry the deploy", after)
				}
			}
			if deployLogLooksLikeOOMKill(d.LogPath, errMsg) {
				heapMB := nodeBuildHeapMB()
				log("hint: build exited 137/SIGKILL. This is usually the kernel or Docker killing the build for memory pressure.")
				log("hint: Relay caps Node build heap at %s MB by default; lower RELAY_NODE_BUILD_HEAP_MB or add swap/use a larger host if Next still gets killed.", heapMB)
				// "(Turbopack)" is Next's own banner ("▲ Next.js 16.x (Turbopack)").
				// Matching the parenthesized form avoids false positives from
				// relayd's own warning lines in the same log.
				if tail, _ := readFileTail(d.LogPath, 256*1024); strings.Contains(tail, "(Turbopack)") {
					// Turbopack allocates in a native Rust binary outside the
					// V8 heap, so NODE_OPTIONS caps cannot bound it.
					log("hint: this build uses Next's Turbopack engine, whose native memory ignores Node heap caps. Swap is the reliable fix (relayd enables it automatically on small hosts when running as root), or build with `next build --webpack`.")
				}
				if hostSwapTotalMB() == 0 {
					log("hint: this host has no swap configured — see docs/small-host-memory.md")
				}
				errMsg = "build process killed with exit 137 (likely out of memory): " + errMsg
			}
			d.Error = "docker build failed: " + errMsg
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
		if appStopped {
			for _, service := range desiredServices {
				service.Stopped = true
				desiredMap[service.Name] = service
			}
			log("app lane is kept offline; deploy will build and update the saved image without starting containers")
			s.reconcileProjectServices(log, req.App, req.Env, req.Branch, desiredMap)
		} else {
			companions, _ := s.orchestrateCompanions(
				log,
				req.App,
				req.Env,
				req.Branch,
				desiredServices,
				false,
				true,
				companionFailureWarning,
			)
			networkName = companions.NetworkName
			for key, value := range companions.Environment {
				extraEnv[key] = value
			}
		}
		runtimeSecretCount := s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, extraEnv, log)
		if runtimeSecretCount > 0 {
			log("runtime env: injected %d lane secret(s) into containers", runtimeSecretCount)
		}

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
		runtimeSecretCount := s.mergeLaneSecretsIntoEnv(req.App, req.Env, req.Branch, extraEnv, log)
		if runtimeSecretCount > 0 {
			log("runtime env: injected %d lane secret(s) into containers", runtimeSecretCount)
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
	var configuredVolumes []string
	if cfg != nil {
		configuredVolumes = cfg.Volumes
	}
	nextState := buildSuccessfulLaneState(successfulLaneTransition{
		Request:           req,
		Current:           currentState,
		Previous:          prev,
		Engine:            engine,
		BuildpackKind:     plan.Kind,
		CurrentImage:      artifactRef,
		PreviousImage:     prevImg,
		RepoHash:          repoHash,
		ConfiguredVolumes: configuredVolumes,
		ActiveSlotFallback: s.currentActiveSlotWithRuntime(
			s.runtimeForEngine(engine),
			req.App,
			req.Env,
			req.Branch,
			prev,
		),
		DeployID: d.ID,
		Stopped:  appStopped,
	})
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
		_ = plan.Cleanup(projectRootDir)
	}

	// Refresh global domain proxy after every successful deploy
	go func() {
		if err := s.ensureGlobalProxy(); err != nil {
			log("global proxy update: %v", err)
		}
	}()

	log("deploy success. image=%s", artifactRef)
}

func (s *Server) requeueDeployLater(job DeployJob, delay time.Duration) {
	// One goroutine per requeue attempt; retries up to 10 times with the same
	// delay before giving up. The old recursive time.AfterFunc approach spawned
	// a new timer on every queue-full check — creating an unbounded chain of
	// goroutines if the queue stayed saturated for a long time.
	go func() {
		for attempt := 0; attempt < 10; attempt++ {
			time.Sleep(delay)
			select {
			case s.queue <- job:
				return
			default:
			}
		}
		fmt.Printf("deploy %s: requeue abandoned after 10 attempts — queue saturated\n", job.ID)
	}()
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

func logRuntimeContainerDiagnostics(runtime ContainerRuntime, log func(string, ...any), name string) {
	if log == nil {
		return
	}
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
		return
	}
	if _, ok := runtime.(*DockerRuntime); ok {
		state, stateErr := exec.Command("docker", "inspect", "--format", "{{json .State}}", name).CombinedOutput()
		if stateErr != nil {
			log("container %s inspect failed: %v (%s)", name, stateErr, strings.TrimSpace(string(state)))
		} else {
			log("container %s state: %s", name, strings.TrimSpace(string(state)))
		}
		logs, logsErr := exec.Command("docker", "logs", "--tail", "200", name).CombinedOutput()
		if logsErr != nil {
			log("container %s logs unavailable: %v (%s)", name, logsErr, strings.TrimSpace(string(logs)))
		} else if trimmed := strings.TrimSpace(string(logs)); trimmed != "" {
			log("container output:\n%s", trimmed)
		} else {
			log("container log is empty")
		}
	}
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
		} else if attempts > 0 && runtime.ContainerExists(name) {
			if log != nil {
				log("container %s exited before becoming ready", name)
			}
			logRuntimeContainerDiagnostics(runtime, log, name)
			return fmt.Errorf("container %s exited before becoming ready on port %d", name, port)
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
	if _, ok := runtime.(*DockerRuntime); ok {
		logRuntimeContainerDiagnostics(runtime, log, name)
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

func (s *Server) writeEdgeProxyConfig(app string, env DeployEnv, branch string, activeSlot string, standbySlot string, servicePort int, trafficMode string, splitPercent int) (string, error) {
	p := s.edgeProxyConfigPath(app, env, branch)
	activeUpstream := fmt.Sprintf("%s:%d", appSlotContainerName(app, env, branch, activeSlot), firstNonZero(servicePort, 3000))
	standbyMode := normalizeActiveSlot(standbySlot)
	trafficMode = firstNonEmpty(normalizeTrafficMode(trafficMode), "edge")
	splitPercent = normalizeTrafficSplitPercent(splitPercent)
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
	// This hop (Caddy -> this nginx -> app container) is always plain HTTP,
	// so $scheme below is always "http" regardless of what the public
	// client actually used. An incoming X-Forwarded-Proto (set correctly by
	// Caddy, or passed through from an upstream CDN like Cloudflare) must
	// win over that — otherwise every app behind Relay is told every
	// request is HTTP even when it wasn't, and any app that redirects
	// http->https based on that header redirects forever (client is on
	// https, gets told it's http, redirects to https, loops — this is
	// exactly what "Flexible" SSL mode on Cloudflare triggers).
	conf.WriteString("  map $http_x_forwarded_proto $relay_xfp {\n")
	conf.WriteString("    default $http_x_forwarded_proto;\n")
	conf.WriteString("    '' $scheme;\n")
	conf.WriteString("  }\n")
	conf.WriteString("  resolver 127.0.0.11 ipv6=off valid=5s;\n")
	conf.WriteString("  client_max_body_size 0;\n")
	conf.WriteString("  large_client_header_buffers 4 16k;\n")
	conf.WriteString("  error_log /dev/stderr;\n")
	conf.WriteString("  log_format relay_json escape=json '{\"ts\":$msec,\"status\":$status,\"host\":\"$host\",\"method\":\"$request_method\",\"path\":\"$request_uri\",\"slot\":\"$relay_target_slot\",\"request_id\":\"$request_id\"}';\n")
	conf.WriteString("  access_log /dev/stdout relay_json;\n")
	conf.WriteString("  map $arg___relay_target $relay_slot_override {\n")
	conf.WriteString("    default \"\";\n")
	conf.WriteString(fmt.Sprintf("    ~*^(new|live|active|%s)$ %s;\n", activeSlot, activeSlot))
	if standbyMode != "" {
		conf.WriteString(fmt.Sprintf("    ~*^(old|previous|standby|draining|%s)$ %s;\n", standbyMode, standbyMode))
	}
	conf.WriteString("  }\n")
	if standbyMode != "" && splitPercent < 100 {
		conf.WriteString("  split_clients $request_id $relay_weighted_slot {\n")
		conf.WriteString(fmt.Sprintf("    %d%% %s;\n", splitPercent, activeSlot))
		conf.WriteString(fmt.Sprintf("    * %s;\n", standbyMode))
		conf.WriteString("  }\n")
	}
	if trafficMode == "session" {
		cookieName := edgeCookieName(app, env, branch)
		conf.WriteString(fmt.Sprintf("  map $cookie_%s $relay_slot_cookie {\n", cookieName))
		if standbyMode != "" && splitPercent < 100 {
			conf.WriteString("    default $relay_weighted_slot;\n")
		} else {
			conf.WriteString(fmt.Sprintf("    default %s;\n", activeSlot))
		}
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
		conf.WriteString("      access_log off;\n")
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
	conf.WriteString("      proxy_set_header X-Forwarded-Proto $relay_xfp;\n")
	conf.WriteString("      proxy_set_header Upgrade $http_upgrade;\n")
	conf.WriteString("      proxy_set_header Connection $connection_upgrade;\n")
	conf.WriteString("      proxy_request_buffering off;\n")
	conf.WriteString("      proxy_read_timeout 300s;\n")
	conf.WriteString("      proxy_send_timeout 300s;\n")
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
	if st, err := s.getAppState(app, env, branch); err == nil && st != nil {
		if st.ResourceMode != "auto" {
			spec.CPULimit = st.CPULimit
			spec.MemLimit = st.MemLimit
		}
		for _, v := range st.Volumes {
			if expanded := expandAppVolume(app, string(env), branch, v); expanded != "" {
				spec.Volumes = append(spec.Volumes, expanded)
			}
		}
	}
	if spec.MemLimit == "" {
		// No explicit limit: cap the container to a host-sized default so one
		// leaky app can't OOM the whole host (relayd, builds, and the other
		// apps included). Docker's default memory-swap allowance (2x the cap)
		// lets the app spill to swap before the kernel kills anything.
		if mb := defaultAppMemLimitMB(); mb > 0 {
			spec.MemLimit = fmt.Sprintf("%dm", mb)
		}
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

func (s *Server) ensureEdgeProxy(log func(string, ...any), app string, env DeployEnv, branch string, networkName string, activeSlot string, standbySlot string, servicePort int, hostPort int, mode string, trafficMode string, publicHost string, splitPercent int, recreate bool) error {
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
	configPath, err := s.writeEdgeProxyConfig(app, env, branch, activeSlot, standbySlot, servicePort, trafficMode, splitPercent)
	if err != nil {
		return fmt.Errorf("write edge proxy config: %w", err)
	}
	volumes := []string{
		fmt.Sprintf("%s:/etc/nginx/nginx.conf:ro", dockerPath(configPath)),
	}
	if err := validateEdgeProxyLogPaths(configPath, volumes); err != nil {
		return fmt.Errorf("edge proxy config validation failed: %w", err)
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
			Volumes:       volumes,
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
		log("reloading edge proxy %s -> active=%s standby=%s traffic=%s split=%d%%", containerName, appSlotContainerName(app, env, branch, activeSlot), normalizeActiveSlot(standbySlot), firstNonEmpty(normalizeTrafficMode(trafficMode), "edge"), normalizeTrafficSplitPercent(splitPercent))
	}
	out, reloadErr := s.runtime.Exec(containerName, []string{"nginx", "-s", "reload"})
	if reloadErr != nil {
		return fmt.Errorf("edge proxy reload failed: %v (%s)", reloadErr, strings.TrimSpace(string(out)))
	}
	return nil
}

func edgeProxyLogDestinations(configText string) []string {
	paths := make([]string, 0, 4)
	scanner := bufio.NewScanner(strings.NewReader(configText))
	scanner.Buffer(make([]byte, 0, 4*1024), 128*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		directive := fields[0]
		if directive != "access_log" && directive != "error_log" {
			continue
		}
		dest := strings.TrimSuffix(strings.TrimSpace(fields[1]), ";")
		if dest == "" || dest == "off" {
			continue
		}
		paths = append(paths, filepath.ToSlash(dest))
	}
	return paths
}

func edgeProxyVolumeTarget(volume string) string {
	spec := strings.TrimSpace(volume)
	if spec == "" {
		return ""
	}
	idx := strings.LastIndex(spec, ":/")
	if idx < 0 {
		return ""
	}
	target := strings.TrimSpace(spec[idx+1:])
	if optIdx := strings.LastIndex(target, ":"); optIdx > 0 {
		opt := target[optIdx+1:]
		if !strings.Contains(opt, "/") && !strings.Contains(opt, "\\") {
			target = target[:optIdx]
		}
	}
	return filepath.ToSlash(strings.TrimSpace(target))
}

func validateEdgeProxyLogPaths(configPath string, volumes []string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read nginx config: %w", err)
	}
	dests := edgeProxyLogDestinations(string(data))
	if len(dests) == 0 {
		return nil
	}
	mountedTargets := make(map[string]struct{}, len(volumes))
	for _, vol := range volumes {
		if target := edgeProxyVolumeTarget(vol); target != "" {
			mountedTargets[target] = struct{}{}
		}
	}
	missing := make([]string, 0, len(dests))
	seen := map[string]struct{}{}
	for _, dest := range dests {
		if !strings.HasPrefix(dest, "/") || strings.HasPrefix(dest, "/dev/") {
			continue
		}
		if _, ok := mountedTargets[dest]; ok {
			continue
		}
		if _, dup := seen[dest]; dup {
			continue
		}
		seen[dest] = struct{}{}
		missing = append(missing, dest)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("nginx config references file log path(s) without matching mounts: %s; use /dev/stdout and /dev/stderr or add matching bind mounts", strings.Join(missing, ", "))
	}
	return nil
}

// retireStandbySlot drops oldSlot from the edge proxy's routing map (via a
// graceful `nginx -s reload`, which stops new connections from being routed
// there without dropping ones already in flight) and only removes its
// container once that's confirmed applied. Removing the container first (the
// previous order) left a window where a sticky-session or in-flight request
// could resolve to a container that had already been torn down — a
// Cloudflare-visible 502 during every slot switch. A short pause after the
// reload gives nginx's already-established connections to the old upstream
// a moment to finish before the process under them disappears.
func (s *Server) retireStandbySlot(app string, env DeployEnv, branch string, activeSlot string, oldSlot string, servicePort int, hostPort int, mode string, trafficMode string, publicHost string) {
	name := appSlotContainerName(app, env, branch, oldSlot)
	_ = s.ensureEdgeProxy(nil, app, env, branch, appNetworkName(app, env, branch), activeSlot, "", servicePort, hostPort, mode, trafficMode, publicHost, 100, false)
	time.Sleep(2 * time.Second)
	s.runtime.Remove(name)
	if st, err := s.getAppState(app, env, branch); err == nil && st != nil {
		if normalizeActiveSlot(st.ActiveSlot) == normalizeActiveSlot(activeSlot) && normalizeActiveSlot(st.StandbySlot) == normalizeActiveSlot(oldSlot) {
			st.StandbySlot = ""
			st.DrainUntil = 0
			_ = s.saveAppState(st)
			s.broadcastSnapshot()
		}
	}
}

func (s *Server) cleanupStandbySlotAfter(app string, env DeployEnv, branch string, activeSlot string, oldSlot string, servicePort int, hostPort int, mode string, trafficMode string, publicHost string, wait time.Duration) {
	if wait <= 0 {
		s.retireStandbySlot(app, env, branch, activeSlot, oldSlot, servicePort, hostPort, mode, trafficMode, publicHost)
		return
	}
	go func() {
		time.Sleep(wait)
		s.retireStandbySlot(app, env, branch, activeSlot, oldSlot, servicePort, hostPort, mode, trafficMode, publicHost)
	}()
}

type edgeProxyAccessLogEntry struct {
	Ts     float64 `json:"ts"`
	Status int     `json:"status"`
	Slot   string  `json:"slot"`
}

func edgeProxyLogJSONPayload(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "{") {
		return trimmed
	}
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	payload := strings.TrimSpace(parts[1])
	if strings.HasPrefix(payload, "{") {
		return payload
	}
	return ""
}

func (s *Server) assessRolloutLog(app string, env DeployEnv, branch string, activeSlot string, sinceMillis int64) (int, int, error) {
	containerName := appBaseContainerName(app, env, branch)
	if !s.runtime.IsRunning(containerName) {
		return 0, 0, fmt.Errorf("edge proxy %s is not running", containerName)
	}

	since := ""
	if sinceMillis > 0 {
		since = strconv.FormatInt(sinceMillis/1000, 10)
	}
	// 8s (was 3s): scanning up to 20000 log lines can legitimately take longer
	// than 3s under host load, and a cutoff scan used to be reported as a
	// clean, error-free result (see below) — silently undercounting requests
	// and letting a genuinely failing canary sail through the health check.
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	pr, err := s.runtime.LogStream(ctx, containerName, 20000, since)
	if err != nil {
		return 0, 0, err
	}
	if pr == nil {
		return 0, 0, fmt.Errorf("edge proxy log stream unavailable")
	}
	defer pr.Close()

	total := 0
	errorCount := 0
	scanner := bufio.NewScanner(pr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		payload := edgeProxyLogJSONPayload(scanner.Text())
		if payload == "" {
			continue
		}
		var entry edgeProxyAccessLogEntry
		if err := json.Unmarshal([]byte(payload), &entry); err != nil {
			continue
		}
		tsMillis := int64(entry.Ts * 1000)
		if sinceMillis > 0 && tsMillis < sinceMillis {
			continue
		}
		if normalizeActiveSlot(entry.Slot) != normalizeActiveSlot(activeSlot) {
			continue
		}
		total++
		if entry.Status >= 500 {
			errorCount++
		}
	}
	if err := scanner.Err(); err != nil {
		// A deadline exceeded mid-scan means the read was cut off before it
		// reached the end of the window — total/errorCount are a partial,
		// potentially misleading sample, not a clean result. Treat that as a
		// failed assessment so the caller doesn't graduate a canary on
		// undercounted data. io.ErrClosedPipe / context.Canceled without a
		// deadline are the normal "reader closed after we were done" case
		// and are still safe to treat as success.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return total, errorCount, fmt.Errorf("log scan timed out before finishing: %w", err)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, io.ErrClosedPipe) {
			return total, errorCount, nil
		}
		return total, errorCount, err
	}
	return total, errorCount, nil
}

func rolloutWatchKey(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("%s__%s__%s", app, env, branch)
}

func (s *Server) startRolloutWatch(app string, env DeployEnv, branch string) {
	key := rolloutWatchKey(app, env, branch)
	s.rolloutWatchMu.Lock()
	if _, ok := s.rolloutWatches[key]; ok {
		s.rolloutWatchMu.Unlock()
		return
	}
	s.rolloutWatches[key] = struct{}{}
	s.rolloutWatchMu.Unlock()

	go func() {
		defer func() {
			s.rolloutWatchMu.Lock()
			delete(s.rolloutWatches, key)
			s.rolloutWatchMu.Unlock()
		}()

		st, err := s.getAppState(app, env, branch)
		if err != nil || st == nil {
			return
		}
		wait := time.Duration(normalizeRolloutAssessSeconds(st.RolloutAssessSeconds)) * time.Second
		time.Sleep(wait)

		st, err = s.getAppState(app, env, branch)
		if err != nil || st == nil {
			return
		}
		if normalizeActiveSlot(st.StandbySlot) == "" || strings.TrimSpace(st.RolloutStatus) != "monitoring" {
			return
		}

		total, errors, logErr := s.assessRolloutLog(app, env, branch, st.ActiveSlot, st.RolloutStartedAt)
		// A failed/inconclusive assessment (log stream unreachable, scan
		// timed out, container not running) must never be treated the same
		// as "no errors seen" — that would auto-promote a canary relay
		// never actually verified. Retry once after a short backoff before
		// giving up, since most causes here (a slow log read, a momentarily
		// busy proxy) are transient.
		if logErr != nil {
			time.Sleep(10 * time.Second)
			total, errors, logErr = s.assessRolloutLog(app, env, branch, st.ActiveSlot, st.RolloutStartedAt)
		}
		errorPercent := 0.0
		if total > 0 {
			errorPercent = (float64(errors) / float64(total)) * 100
		}
		minRequests := normalizeRolloutMinRequests(st.RolloutMinRequests)
		threshold := normalizeRolloutErrorPercent(st.RolloutErrorPercent)

		if logErr == nil && total >= minRequests && errorPercent > threshold {
			_ = s.rollbackCanary(app, env, branch, fmt.Sprintf("%.2f%% 5xx over %d requests", errorPercent, total))
			return
		}
		if logErr != nil {
			// Still couldn't get a trustworthy read after the retry. Hold —
			// leave RolloutStatus as "monitoring" rather than blindly
			// graduating an unverified canary. relayd re-arms this watch on
			// its next restart (resumeRolloutWatches); the split otherwise
			// stays as-is until then.
			s.auditLog("relay-rollout", "rollout.assess_failed", app, fmt.Sprintf("env=%s branch=%s requests=%d err=%v — holding at current split, not graduating", env, branch, total, logErr))
			return
		}
		_ = s.graduateCanary(app, env, branch, total, errorPercent, logErr)
	}()
}

func (s *Server) graduateCanary(app string, env DeployEnv, branch string, total int, errorPercent float64, assessErr error) error {
	st, err := s.getAppState(app, env, branch)
	if err != nil || st == nil {
		return err
	}
	if normalizeActiveSlot(st.StandbySlot) == "" {
		return nil
	}
	st.TrafficSplitPercent = 100
	st.RolloutStartedAt = 0
	st.RolloutStatus = "graduated"
	_ = s.saveRolloutGraduation(st)
	s.broadcastSnapshot()
	s.auditLog("relay-rollout", "rollout.graduate", app, fmt.Sprintf("env=%s branch=%s requests=%d error_percent=%.2f err=%v", env, branch, total, errorPercent, assessErr))
	s.cleanupStandbySlotAfter(app, env, branch, st.ActiveSlot, st.StandbySlot, st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, st.PublicHost, rolloutDrainDuration())
	return nil
}

func (s *Server) rollbackCanary(app string, env DeployEnv, branch string, detail string) error {
	st, err := s.getAppState(app, env, branch)
	if err != nil || st == nil {
		return err
	}
	oldSlot := normalizeActiveSlot(st.StandbySlot)
	newSlot := normalizeActiveSlot(st.ActiveSlot)
	if oldSlot == "" || newSlot == "" {
		return nil
	}
	if err := s.ensureEdgeProxy(nil, app, env, branch, appNetworkName(app, env, branch), oldSlot, "", st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, st.PublicHost, 100, false); err != nil {
		return err
	}
	s.runtime.Remove(appSlotContainerName(app, env, branch, newSlot))
	st.ActiveSlot = oldSlot
	st.StandbySlot = ""
	st.DrainUntil = 0
	st.TrafficSplitPercent = 100
	st.RolloutStartedAt = 0
	st.RolloutStatus = "rolled_back"
	if st.PreviousImage != "" {
		st.CurrentImage = st.PreviousImage
	}
	_ = s.saveRolloutRollback(st)
	s.broadcastSnapshot()
	s.auditLog("relay-rollout", "rollout.rollback", app, fmt.Sprintf("env=%s branch=%s %s", env, branch, detail))
	return nil
}

func (s *Server) resumeRolloutWatches() error {
	rows, err := s.db.Query(
		`SELECT app, env, branch
		 FROM app_state
		 WHERE COALESCE(standby_slot,'') != ''
		   AND COALESCE(rollout_status,'') = 'monitoring'`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var app string
		var env string
		var branch string
		if err := rows.Scan(&app, &env, &branch); err != nil {
			continue
		}
		s.startRolloutWatch(app, DeployEnv(env), branch)
	}
	return nil
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

func (s *Server) analyticsStore() *sql.DB {
	if s.analyticsDB != nil {
		return s.analyticsDB
	}
	return s.db
}

// startLogTailer reads new lines appended to the Caddy access log, inserts
// analytics events into SQLite, and resolves country codes in the background.
func (s *Server) startLogTailer() {
	logPath := filepath.Join(s.caddyLogsDir, "access.log")
	adb := s.analyticsStore()

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
			s.insertAnalyticsEvents(adb, newEvents)
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

func (s *Server) insertAnalyticsEvents(adb *sql.DB, entries []caddyLogEntry) {
	tx, err := adb.Begin()
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
	adb := s.analyticsStore()
	// Collect distinct IPs that have no country assigned yet.
	rows, err := adb.Query(`
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

	// Filter out already-cached IPs in a single IN query.
	uncached := ips[:0]
	if len(ips) > 0 {
		placeholders := make([]string, len(ips))
		args := make([]any, len(ips))
		for i, ip := range ips {
			placeholders[i] = "?"
			args[i] = ip
		}
		cachedRows, err2 := adb.Query(
			`SELECT ip FROM ip_country_cache WHERE ip IN (`+strings.Join(placeholders, ",")+`) AND country_code != ''`,
			args...)
		cached := map[string]bool{}
		if err2 == nil {
			for cachedRows.Next() {
				var ip string
				if cachedRows.Scan(&ip) == nil {
					cached[ip] = true
				}
			}
			cachedRows.Close()
		}
		for _, ip := range ips {
			if !cached[ip] {
				uncached = append(uncached, ip)
			}
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
			_, _ = adb.Exec(
				`INSERT OR REPLACE INTO ip_country_cache(ip,country_code,country_name,updated_at) VALUES(?,?,?,?)`,
				r.Query, r.CountryCode, r.Country, now)
		}
		// Small back-off to stay under the 45 req/min rate limit.
		if i+100 < len(uncached) {
			time.Sleep(1500 * time.Millisecond)
		}
	}

	// Apply cached countries to analytics_events rows that are still empty.
	_, _ = adb.Exec(`
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

type adminOpsComparisonWindow struct {
	Seconds int64 `json:"seconds"`
}

type adminOpsTrafficWindow struct {
	Requests      int64   `json:"requests"`
	ServerErrors  int64   `json:"server_errors"`
	ClientErrors  int64   `json:"client_errors"`
	Bandwidth     int64   `json:"bandwidth_bytes"`
	ServerErrRate float64 `json:"server_error_rate"`
}

type adminOpsDeploySummary struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Source           string `json:"source,omitempty"`
	CreatedAt        string `json:"created_at"`
	StartedAt        string `json:"started_at,omitempty"`
	EndedAt          string `json:"ended_at,omitempty"`
	BuildNumber      int    `json:"build_number,omitempty"`
	BuildDurationMS  int64  `json:"build_duration_ms,omitempty"`
	CommitSHA        string `json:"commit_sha,omitempty"`
	CommitMessage    string `json:"commit_message,omitempty"`
	DeployedBy       string `json:"deployed_by,omitempty"`
	ImageTag         string `json:"image_tag,omitempty"`
	PreviousImageTag string `json:"previous_image_tag,omitempty"`
}

type adminOpsDeployDelta struct {
	Current              adminOpsDeploySummary    `json:"current"`
	Previous             *adminOpsDeploySummary   `json:"previous,omitempty"`
	Window               adminOpsComparisonWindow `json:"window"`
	CurrentTraffic       *adminOpsTrafficWindow   `json:"current_traffic,omitempty"`
	PreviousTraffic      *adminOpsTrafficWindow   `json:"previous_traffic,omitempty"`
	BuildDurationDeltaMS int64                    `json:"build_duration_delta_ms,omitempty"`
	ServerErrRateDelta   float64                  `json:"server_error_rate_delta,omitempty"`
	RequestDelta         int64                    `json:"request_delta,omitempty"`
	BandwidthDelta       int64                    `json:"bandwidth_delta_bytes,omitempty"`
	AnalyticsAvailable   bool                     `json:"analytics_available"`
	AnalyticsNote        string                   `json:"analytics_note,omitempty"`
}

type adminOpsContainerUsage struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Kind          string  `json:"kind"`
	Container     string  `json:"container"`
	Running       bool    `json:"running"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsageBytes int64   `json:"mem_usage_bytes"`
	MemLimitBytes int64   `json:"mem_limit_bytes"`
	MemPercent    float64 `json:"mem_percent"`
	StorageBytes  int64   `json:"storage_bytes"`
	NetRxBytes    int64   `json:"net_rx_bytes"`
	NetTxBytes    int64   `json:"net_tx_bytes"`
	BlockRead     int64   `json:"block_read_bytes"`
	BlockWrite    int64   `json:"block_write_bytes"`
}

type adminOpsLaneUsage struct {
	CPUPercent        float64                  `json:"cpu_percent"`
	MemUsageBytes     int64                    `json:"mem_usage_bytes"`
	MemLimitBytes     int64                    `json:"mem_limit_bytes"`
	MemPercent        float64                  `json:"mem_percent"`
	StorageBytes      int64                    `json:"storage_bytes"`
	NetRxBytes        int64                    `json:"net_rx_bytes"`
	NetTxBytes        int64                    `json:"net_tx_bytes"`
	BlockReadBytes    int64                    `json:"block_read_bytes"`
	BlockWriteBytes   int64                    `json:"block_write_bytes"`
	RunningContainers int                      `json:"running_containers"`
	ContainerCount    int                      `json:"container_count"`
	Measured          bool                     `json:"measured"`
	Note              string                   `json:"note,omitempty"`
	Targets           []adminOpsContainerUsage `json:"targets"`
}

type adminOpsLane struct {
	App           string               `json:"app"`
	Env           string               `json:"env"`
	Branch        string               `json:"branch"`
	Engine        string               `json:"engine"`
	PublicHost    string               `json:"public_host,omitempty"`
	HostPort      int                  `json:"host_port,omitempty"`
	RepoURL       string               `json:"repo_url,omitempty"`
	Stopped       bool                 `json:"stopped"`
	CurrentImage  string               `json:"current_image,omitempty"`
	PreviousImage string               `json:"previous_image,omitempty"`
	Usage         adminOpsLaneUsage    `json:"usage"`
	Latest        *adminOpsDeployDelta `json:"latest,omitempty"`
}

type adminOpsApp struct {
	App         string            `json:"app"`
	LaneCount   int               `json:"lane_count"`
	OnlineLanes int               `json:"online_lanes"`
	Usage       adminOpsLaneUsage `json:"usage"`
	Lanes       []adminOpsLane    `json:"lanes"`
}

type adminOpsSummary struct {
	AppCount      int     `json:"app_count"`
	LaneCount     int     `json:"lane_count"`
	OnlineLanes   int     `json:"online_lanes"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsageBytes int64   `json:"mem_usage_bytes"`
	MemLimitBytes int64   `json:"mem_limit_bytes"`
	MemPercent    float64 `json:"mem_percent"`
	StorageBytes  int64   `json:"storage_bytes"`
}

type adminOpsResponse struct {
	GeneratedAt int64           `json:"generated_at"`
	Summary     adminOpsSummary `json:"summary"`
	Apps        []adminOpsApp   `json:"apps"`
	Daemon      adminOpsDaemon  `json:"daemon"`
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	adb := s.analyticsStore()
	period := r.URL.Query().Get("period")
	app := strings.TrimSpace(r.URL.Query().Get("app"))
	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	env := DeployEnv(strings.TrimSpace(r.URL.Query().Get("env")))
	var sess *UserSession
	if s.hasUsers() {
		sess = s.validateUserSession(r)
		if sess == nil {
			httpError(w, 401, "unauthorized")
			return
		}
	}

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
		if sess != nil {
			allowed := false
			if env != "" {
				allowed = roleAtLeast(s.effectiveLaneRole(sess, app, env), "viewer")
			} else {
				rows, err := s.db.Query(`SELECT DISTINCT env FROM app_state WHERE app=?`, app)
				if err == nil {
					for rows.Next() {
						var laneEnv string
						if rows.Scan(&laneEnv) == nil && roleAtLeast(s.effectiveLaneRole(sess, app, DeployEnv(laneEnv)), "viewer") {
							allowed = true
							break
						}
					}
					rows.Close()
				}
			}
			if !allowed {
				httpError(w, 403, "insufficient app access")
				return
			}
		}
		hosts, err := s.analyticsHostsForSelection(app, env, branch)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if len(hosts) == 0 {
			writeJSON(w, 200, analyticsResponse{PeriodLabel: periodLabel})
			return
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(hosts)), ",")
		hostFilter = ` AND host IN (` + placeholders + `)`
		for _, host := range hosts {
			hostArgs = append(hostArgs, host)
		}
	}

	// Total requests.
	var total int64
	_ = adb.QueryRow(`SELECT COUNT(*) FROM analytics_events WHERE ts>=?`+hostFilter, hostArgs...).Scan(&total)

	// By country.
	countryRows, _ := adb.Query(
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
	statusRows, _ := adb.Query(
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
	hourRows, _ := adb.Query(
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
	hostRows, _ := adb.Query(
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

type dockerStatsRow struct {
	CPUPercent float64
	MemUsage   int64
	MemLimit   int64
	MemPercent float64
	NetRx      int64
	NetTx      int64
	BlockRead  int64
	BlockWrite int64
}

type dockerStorageRow struct {
	Running    bool
	SizeRW     int64
	SizeRootFS int64
}

func parseDockerPercent(raw string) float64 {
	value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "%"))
	if value == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(value, 64)
	return f
}

func parseDockerBytes(raw string) int64 {
	value := strings.TrimSpace(raw)
	if value == "" || value == "--" || value == "0" || value == "0B" {
		return 0
	}
	value = strings.ReplaceAll(value, " ", "")
	splitAt := -1
	for i, r := range value {
		if !(r == '.' || r == '-' || (r >= '0' && r <= '9')) {
			splitAt = i
			break
		}
	}
	if splitAt <= 0 {
		n, _ := strconv.ParseFloat(value, 64)
		return int64(n)
	}
	numPart := value[:splitAt]
	unitPart := strings.ToUpper(strings.TrimSpace(value[splitAt:]))
	num, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0
	}
	units := map[string]float64{
		"B":  1,
		"KB": 1000, "MB": 1000 * 1000, "GB": 1000 * 1000 * 1000, "TB": 1000 * 1000 * 1000 * 1000,
		"KIB": 1024, "MIB": 1024 * 1024, "GIB": 1024 * 1024 * 1024, "TIB": 1024 * 1024 * 1024 * 1024,
	}
	multiplier, ok := units[unitPart]
	if !ok {
		multiplier = 1
	}
	return int64(num * multiplier)
}

func parseDockerIOPair(raw string) (int64, int64) {
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseDockerBytes(parts[0]), parseDockerBytes(parts[1])
}

func collectDockerStats(containers []string) map[string]dockerStatsRow {
	out := map[string]dockerStatsRow{}
	if len(containers) == 0 {
		return out
	}
	args := []string{
		"stats", "--no-stream",
		"--format", "{{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}",
	}
	args = append(args, containers...)
	cmd := exec.Command("docker", args...)
	data, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}
		memParts := strings.Split(fields[2], "/")
		memUsage := int64(0)
		memLimit := int64(0)
		if len(memParts) == 2 {
			memUsage = parseDockerBytes(memParts[0])
			memLimit = parseDockerBytes(memParts[1])
		}
		netRx, netTx := parseDockerIOPair(fields[4])
		blockRead, blockWrite := parseDockerIOPair(fields[5])
		out[strings.TrimSpace(fields[0])] = dockerStatsRow{
			CPUPercent: parseDockerPercent(fields[1]),
			MemUsage:   memUsage,
			MemLimit:   memLimit,
			MemPercent: parseDockerPercent(fields[3]),
			NetRx:      netRx,
			NetTx:      netTx,
			BlockRead:  blockRead,
			BlockWrite: blockWrite,
		}
	}
	return out
}

func collectDockerStorage(containers []string) map[string]dockerStorageRow {
	out := map[string]dockerStorageRow{}
	if len(containers) == 0 {
		return out
	}
	args := []string{
		"inspect", "--size",
		"--format", "{{.Name}}\t{{.State.Running}}\t{{.SizeRw}}\t{{.SizeRootFs}}",
	}
	args = append(args, containers...)
	cmd := exec.Command("docker", args...)
	data, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		name := strings.TrimPrefix(strings.TrimSpace(fields[0]), "/")
		sizeRW, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		sizeRootFS, _ := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64)
		out[name] = dockerStorageRow{
			Running:    strings.EqualFold(strings.TrimSpace(fields[1]), "true"),
			SizeRW:     sizeRW,
			SizeRootFS: sizeRootFS,
		}
	}
	return out
}

func deployBuildDurationMS(d *Deploy) int64 {
	if d == nil || d.StartedAt == nil || d.EndedAt == nil {
		return 0
	}
	return d.EndedAt.Sub(*d.StartedAt).Milliseconds()
}

func toAdminDeploySummary(d *Deploy) adminOpsDeploySummary {
	if d == nil {
		return adminOpsDeploySummary{}
	}
	return adminOpsDeploySummary{
		ID:        d.ID,
		Status:    string(d.Status),
		Source:    d.Source,
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
		StartedAt: func() string {
			if d.StartedAt != nil {
				return d.StartedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		EndedAt: func() string {
			if d.EndedAt != nil {
				return d.EndedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		BuildNumber:      d.BuildNumber,
		BuildDurationMS:  deployBuildDurationMS(d),
		CommitSHA:        d.CommitSHA,
		CommitMessage:    d.CommitMessage,
		DeployedBy:       d.DeployedBy,
		ImageTag:         d.ImageTag,
		PreviousImageTag: d.PrevImage,
	}
}

func analyticsWindowHost(host string, hostPort int) []string {
	candidates := []string{}
	appendIf := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, existing := range candidates {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		candidates = append(candidates, v)
	}
	appendIf(host)
	if hostPort > 0 {
		appendIf(fmt.Sprintf("127.0.0.1:%d", hostPort))
		appendIf(fmt.Sprintf("localhost:%d", hostPort))
	}
	return candidates
}

func analyticsWindowStats(adb *sql.DB, hosts []string, from int64, to int64) (*adminOpsTrafficWindow, bool) {
	if adb == nil || len(hosts) == 0 || to <= from {
		return nil, false
	}
	placeholders := make([]string, len(hosts))
	args := make([]any, 0, len(hosts)+2)
	args = append(args, from, to)
	for i, host := range hosts {
		placeholders[i] = "?"
		args = append(args, host)
	}
	row := adb.QueryRow(
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN status>=500 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status>=400 AND status<500 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(bytes),0)
		   FROM analytics_events
		  WHERE ts>=? AND ts<? AND host IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	var reqs, serverErrors, clientErrors, bytesSent int64
	if err := row.Scan(&reqs, &serverErrors, &clientErrors, &bytesSent); err != nil {
		return nil, false
	}
	window := &adminOpsTrafficWindow{
		Requests:      reqs,
		ServerErrors:  serverErrors,
		ClientErrors:  clientErrors,
		Bandwidth:     bytesSent,
		ServerErrRate: 0,
	}
	if reqs > 0 {
		window.ServerErrRate = float64(serverErrors) / float64(reqs) * 100
	}
	return window, true
}

func (s *Server) latestSuccessfulDeployPair(app string, env DeployEnv, branch string) (*Deploy, *Deploy, error) {
	rows, err := s.db.Query(
		`SELECT d.id, d.app, d.repo_url, d.branch, d.commit_sha, d.env, d.status, d.created_at, d.started_at, d.ended_at, d.error, d.log_path, d.image_tag, d.previous_image_tag, COALESCE(d.preview_url,''), COALESCE(r.source,''), COALESCE(d.build_number,0), COALESCE(d.deployed_by,''), COALESCE(d.commit_message,'')
		   FROM deploys d
		   LEFT JOIN deploy_requests r ON r.id=d.id
		  WHERE d.app=? AND d.env=? AND d.branch=? AND d.status=?
		  ORDER BY d.created_at DESC
		  LIMIT 2`,
		app, string(env), branch, string(StatusSuccess),
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []*Deploy{}
	for rows.Next() {
		d, scanErr := scanDeployRow(rows)
		if scanErr == nil {
			items = append(items, d)
		}
	}
	if len(items) == 0 {
		return nil, nil, nil
	}
	if len(items) == 1 {
		return items[0], nil, nil
	}
	return items[0], items[1], nil
}

func (s *Server) buildAdminOpsResponse() adminOpsResponse {
	resp := adminOpsResponse{
		GeneratedAt: time.Now().UnixMilli(),
		Apps:        []adminOpsApp{},
		Daemon:      collectDaemonStats(),
	}
	rows, err := s.db.Query(`SELECT app, env, branch FROM app_state ORDER BY app, env, branch`)
	if err != nil {
		return resp
	}
	defer rows.Close()

	type laneRef struct {
		App    string
		Env    DeployEnv
		Branch string
	}
	laneRefs := []laneRef{}
	for rows.Next() {
		var lane laneRef
		var env string
		if scanErr := rows.Scan(&lane.App, &env, &lane.Branch); scanErr == nil {
			lane.Env = DeployEnv(env)
			laneRefs = append(laneRefs, lane)
		}
	}

	type laneBundle struct {
		state    *AppState
		targets  []RuntimeLogTarget
		laneData RuntimeLogLaneState
		services []ProjectService
	}
	bundles := map[string]laneBundle{}
	containerNames := []string{}
	seenContainers := map[string]struct{}{}
	for _, ref := range laneRefs {
		st, err := s.getAppState(ref.App, ref.Env, ref.Branch)
		if err != nil || st == nil {
			continue
		}
		targets, laneData, _ := s.runtimeLogTargets(ref.App, ref.Env, ref.Branch)
		services, _ := s.getProjectServices(ref.App, string(ref.Env), ref.Branch)
		key := ref.App + "__" + string(ref.Env) + "__" + ref.Branch
		bundles[key] = laneBundle{state: st, targets: targets, laneData: laneData, services: services}
		for _, target := range targets {
			if target.Container == "" {
				continue
			}
			if _, ok := seenContainers[target.Container]; ok {
				continue
			}
			seenContainers[target.Container] = struct{}{}
			containerNames = append(containerNames, target.Container)
		}
	}

	statsByContainer := collectDockerStats(containerNames)
	storageByContainer := collectDockerStorage(containerNames)

	apps := map[string]*adminOpsApp{}
	adb := s.analyticsStore()
	nowSec := time.Now().Unix()
	for _, ref := range laneRefs {
		key := ref.App + "__" + string(ref.Env) + "__" + ref.Branch
		bundle, ok := bundles[key]
		if !ok || bundle.state == nil {
			continue
		}
		if _, ok := apps[ref.App]; !ok {
			apps[ref.App] = &adminOpsApp{
				App:   ref.App,
				Usage: adminOpsLaneUsage{Targets: []adminOpsContainerUsage{}},
				Lanes: []adminOpsLane{},
			}
		}
		appEntry := apps[ref.App]

		laneUsage := adminOpsLaneUsage{Targets: []adminOpsContainerUsage{}}
		if firstNonEmptyEngine(bundle.state.Engine) != EngineDocker {
			laneUsage.Note = "Live resource sampling currently supports Docker lanes."
		}
		for _, target := range bundle.targets {
			stat := statsByContainer[target.Container]
			size := storageByContainer[target.Container]
			targetUsage := adminOpsContainerUsage{
				ID:            target.ID,
				Label:         target.Label,
				Kind:          target.Kind,
				Container:     target.Container,
				Running:       target.Running,
				CPUPercent:    stat.CPUPercent,
				MemUsageBytes: stat.MemUsage,
				MemLimitBytes: stat.MemLimit,
				MemPercent:    stat.MemPercent,
				StorageBytes:  size.SizeRW + size.SizeRootFS,
				NetRxBytes:    stat.NetRx,
				NetTxBytes:    stat.NetTx,
				BlockRead:     stat.BlockRead,
				BlockWrite:    stat.BlockWrite,
			}
			laneUsage.Targets = append(laneUsage.Targets, targetUsage)
			laneUsage.ContainerCount++
			if target.Running {
				laneUsage.RunningContainers++
			}
			laneUsage.CPUPercent += targetUsage.CPUPercent
			laneUsage.MemUsageBytes += targetUsage.MemUsageBytes
			laneUsage.MemLimitBytes += targetUsage.MemLimitBytes
			laneUsage.StorageBytes += targetUsage.StorageBytes
			laneUsage.NetRxBytes += targetUsage.NetRxBytes
			laneUsage.NetTxBytes += targetUsage.NetTxBytes
			laneUsage.BlockReadBytes += targetUsage.BlockRead
			laneUsage.BlockWriteBytes += targetUsage.BlockWrite
			if target.Running && (targetUsage.CPUPercent > 0 || targetUsage.MemUsageBytes > 0 || targetUsage.StorageBytes > 0) {
				laneUsage.Measured = true
			}
		}
		if laneUsage.MemLimitBytes > 0 {
			laneUsage.MemPercent = float64(laneUsage.MemUsageBytes) / float64(laneUsage.MemLimitBytes) * 100
		}

		var latestDelta *adminOpsDeployDelta
		latest, previous, _ := s.latestSuccessfulDeployPair(ref.App, ref.Env, ref.Branch)
		if latest != nil {
			delta := &adminOpsDeployDelta{
				Current:            toAdminDeploySummary(latest),
				AnalyticsAvailable: false,
			}
			if previous != nil {
				prevSummary := toAdminDeploySummary(previous)
				delta.Previous = &prevSummary
				delta.BuildDurationDeltaMS = delta.Current.BuildDurationMS - prevSummary.BuildDurationMS
				latestAnchor := latest.CreatedAt.Unix()
				if latest.EndedAt != nil {
					latestAnchor = latest.EndedAt.Unix()
				}
				prevAnchor := previous.CreatedAt.Unix()
				if previous.EndedAt != nil {
					prevAnchor = previous.EndedAt.Unix()
				}
				windowSec := int64(0)
				if latestAnchor > prevAnchor {
					windowSec = latestAnchor - prevAnchor
				}
				if live := nowSec - latestAnchor; live > 0 && (windowSec == 0 || live < windowSec) {
					windowSec = live
				}
				if windowSec > 7*24*3600 {
					windowSec = 7 * 24 * 3600
				}
				if windowSec >= 300 {
					delta.Window.Seconds = windowSec
					hosts := analyticsWindowHost(bundle.state.PublicHost, bundle.state.HostPort)
					currentTraffic, currentOK := analyticsWindowStats(adb, hosts, latestAnchor, latestAnchor+windowSec)
					previousTraffic, prevOK := analyticsWindowStats(adb, hosts, prevAnchor, prevAnchor+windowSec)
					if currentOK && prevOK {
						delta.CurrentTraffic = currentTraffic
						delta.PreviousTraffic = previousTraffic
						delta.ServerErrRateDelta = currentTraffic.ServerErrRate - previousTraffic.ServerErrRate
						delta.RequestDelta = currentTraffic.Requests - previousTraffic.Requests
						delta.BandwidthDelta = currentTraffic.Bandwidth - previousTraffic.Bandwidth
						delta.AnalyticsAvailable = true
					} else {
						delta.AnalyticsNote = "Traffic comparison is only available for lanes with proxy analytics history."
					}
				} else {
					delta.AnalyticsNote = "Relay needs more post-deploy traffic time before it can compare this lane."
				}
			} else {
				delta.AnalyticsNote = "No previous successful deploy exists for this lane yet."
			}
			latestDelta = delta
		}

		lane := adminOpsLane{
			App:           ref.App,
			Env:           string(ref.Env),
			Branch:        ref.Branch,
			Engine:        firstNonEmptyEngine(bundle.state.Engine),
			PublicHost:    bundle.state.PublicHost,
			HostPort:      bundle.state.HostPort,
			RepoURL:       bundle.state.RepoURL,
			Stopped:       bundle.state.Stopped,
			CurrentImage:  bundle.state.CurrentImage,
			PreviousImage: bundle.state.PreviousImage,
			Usage:         laneUsage,
			Latest:        latestDelta,
		}
		appEntry.Lanes = append(appEntry.Lanes, lane)
		appEntry.LaneCount++
		if laneUsage.RunningContainers > 0 && !lane.Stopped {
			appEntry.OnlineLanes++
		}
		appEntry.Usage.CPUPercent += laneUsage.CPUPercent
		appEntry.Usage.MemUsageBytes += laneUsage.MemUsageBytes
		appEntry.Usage.MemLimitBytes += laneUsage.MemLimitBytes
		appEntry.Usage.StorageBytes += laneUsage.StorageBytes
		appEntry.Usage.RunningContainers += laneUsage.RunningContainers
		appEntry.Usage.ContainerCount += laneUsage.ContainerCount
		if laneUsage.Measured {
			appEntry.Usage.Measured = true
		}
	}

	appNames := make([]string, 0, len(apps))
	for name := range apps {
		appNames = append(appNames, name)
	}
	sort.Strings(appNames)
	for _, name := range appNames {
		app := apps[name]
		sort.Slice(app.Lanes, func(i, j int) bool {
			if app.Lanes[i].Env == app.Lanes[j].Env {
				return app.Lanes[i].Branch < app.Lanes[j].Branch
			}
			return app.Lanes[i].Env < app.Lanes[j].Env
		})
		if app.Usage.MemLimitBytes > 0 {
			app.Usage.MemPercent = float64(app.Usage.MemUsageBytes) / float64(app.Usage.MemLimitBytes) * 100
		}
		resp.Apps = append(resp.Apps, *app)
		resp.Summary.AppCount++
		resp.Summary.LaneCount += app.LaneCount
		resp.Summary.OnlineLanes += app.OnlineLanes
		resp.Summary.CPUPercent += app.Usage.CPUPercent
		resp.Summary.MemUsageBytes += app.Usage.MemUsageBytes
		resp.Summary.MemLimitBytes += app.Usage.MemLimitBytes
		resp.Summary.StorageBytes += app.Usage.StorageBytes
	}
	if resp.Summary.MemLimitBytes > 0 {
		resp.Summary.MemPercent = float64(resp.Summary.MemUsageBytes) / float64(resp.Summary.MemLimitBytes) * 100
	}
	return resp
}

func (s *Server) handleAdminOps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	writeJSON(w, 200, s.buildAdminOpsResponse())
}

func (s *Server) startACMEListener() {
	s.acmeMu.Lock()
	defer s.acmeMu.Unlock()

	if s.acmeServer != nil {
		_ = s.acmeServer.Close()
		s.acmeServer = nil
	}
	if s.acmeListener != nil {
		_ = s.acmeListener.Close()
		s.acmeListener = nil
	}

	if s.acmeDisabled() {
		fmt.Println("ACME HTTP listener disabled")
		return
	}

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
	// Redirect plain-HTTP traffic to HTTPS only when HTTPS is actually
	// configured.  Without this guard, local/dev installs with no domain
	// configured get stuck in an infinite redirect loop.
	// Enable by setting RELAY_HTTPS_REDIRECT=1 or by configuring a base
	// domain or dashboard host (which implies TLS is in use via Caddy).
	acmeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpsRedirect := (getenv("RELAY_HTTPS_REDIRECT", "") != "" ||
			s.serverBaseDomain() != "" ||
			s.serverDashboardHost() != "") && !isHTTPSRequest(r)
		if httpsRedirect {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		// No HTTPS configured: pass non-challenge traffic through without redirect.
		http.NotFound(w, r)
	})

	srv := &http.Server{
		Handler:           acmeMux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	s.acmeListener = ln
	s.acmeServer = srv
	fmt.Println("ACME HTTP listener on", acmeAddr)
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "ACME listener error: %v\n", err)
		}
		s.acmeMu.Lock()
		if s.acmeServer == srv {
			s.acmeServer = nil
		}
		if s.acmeListener == ln {
			s.acmeListener = nil
		}
		s.acmeMu.Unlock()
	}()
}

func customHostRuleBlock(rule customHostRule, relayPort int) (string, error) {
	switch rule.Action {
	case "redirect":
		target := rule.RedirectURL
		if rule.PreservePath {
			target += "{uri}"
		}
		return fmt.Sprintf("\tredir %s %d\n", target, rule.RedirectCode), nil
	case "reverse_proxy":
		return fmt.Sprintf("\treverse_proxy %s\n", rule.UpstreamURL), nil
	case "static_response":
		var block strings.Builder
		if ct := strings.TrimSpace(rule.ResponseContentType); ct != "" {
			block.WriteString(fmt.Sprintf("\theader Content-Type %s\n", strconv.Quote(ct)))
		}
		block.WriteString(fmt.Sprintf("\trespond %s %d\n", strconv.Quote(rule.ResponseBody), rule.ResponseStatus))
		return block.String(), nil
	case "relay_dashboard":
		if relayPort <= 0 {
			return "", fmt.Errorf("relay dashboard host rules require Relay to listen on a TCP port")
		}
		return fmt.Sprintf("\treverse_proxy host.docker.internal:%d\n", relayPort), nil
	default:
		return "", fmt.Errorf("unsupported custom host rule action: %s", rule.Action)
	}
}

func (s *Server) ensureGlobalProxy() error {
	if _, err := s.repairLegacyAppHostPortsFromRuntime(); err != nil {
		return err
	}

	rows, err := s.db.Query(`SELECT app, env, branch, COALESCE(engine,'docker'), public_host, COALESCE(public_hosts,''), COALESCE(host_port, 0) FROM app_state WHERE (public_host != '' OR COALESCE(public_hosts,'') != '') AND COALESCE(stopped,0)=0`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type route struct {
		host  string
		block string
	}
	routeIndex := map[string]int{}
	var routes []route
	dockerNetworks := map[string]struct{}{}
	setRoute := func(host string, block string) {
		host = strings.TrimSpace(host)
		if host == "" || strings.TrimSpace(block) == "" {
			return
		}
		if idx, ok := routeIndex[host]; ok {
			routes[idx].block = block
			return
		}
		routeIndex[host] = len(routes)
		routes = append(routes, route{host: host, block: block})
	}
	for rows.Next() {
		var app string
		var env DeployEnv
		var branch string
		var engine string
		var publicHost string
		var publicHostsRaw string
		var hostPort int
		if err := rows.Scan(&app, &env, &branch, &engine, &publicHost, &publicHostsRaw, &hostPort); err != nil {
			continue
		}
		primary, hosts := canonicalizePublicHosts(publicHost, parsePublicHosts(publicHostsRaw))
		if primary == "" {
			continue
		}
		upstream := ""
		if firstNonEmptyEngine(engine) == EngineDocker {
			upstream = fmt.Sprintf("%s:3000", appBaseContainerName(app, env, branch))
			dockerNetworks[appNetworkName(app, env, branch)] = struct{}{}
		} else if hostPort > 0 {
			upstream = fmt.Sprintf("host.docker.internal:%d", hostPort)
		}
		if upstream == "" {
			continue
		}
		for _, host := range hosts {
			setRoute(host, fmt.Sprintf("\treverse_proxy %s\n", upstream))
		}
	}
	if dashboardHost := strings.TrimSpace(s.serverDashboardHost()); dashboardHost != "" {
		if relayPort := listenAddrPort(s.httpAddr); relayPort > 0 {
			setRoute(dashboardHost, fmt.Sprintf("\treverse_proxy host.docker.internal:%d\n", relayPort))
		}
	}
	relayPort := listenAddrPort(s.httpAddr)
	for _, rule := range s.serverCustomHostRules() {
		block, err := customHostRuleBlock(rule, relayPort)
		if err != nil {
			return err
		}
		setRoute(rule.Host, block)
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
	// DNS-01 challenge support: hosts fronted by a tunnel/CDN (Cloudflare
	// Tunnel, etc.) never let Let's Encrypt reach the origin directly, so
	// HTTP-01 and TLS-ALPN-01 both fail. DNS-01 proves ownership via a TXT
	// record instead and works regardless of what's in front of the origin.
	// Requires a Caddy image built with the cloudflare-dns plugin, set via
	// RELAY_CADDY_IMAGE (see docker/caddy/Dockerfile).
	cfToken := strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN"))
	for _, r := range routes {
		cf.WriteString(strings.TrimSpace(r.host) + " {\n")
		if cfToken != "" {
			cf.WriteString("\ttls {\n\t\tdns cloudflare {env.CLOUDFLARE_API_TOKEN}\n\t}\n")
		}
		cf.WriteString(r.block)
		cf.WriteString("}\n\n")
	}
	if err := os.WriteFile(configPath, []byte(cf.String()), 0644); err != nil {
		return err
	}
	connectDockerNetworks := func() error {
		networks := make([]string, 0, len(dockerNetworks))
		for network := range dockerNetworks {
			networks = append(networks, network)
		}
		sort.Strings(networks)
		var connectErrs []error
		for _, network := range networks {
			if err := s.runtime.NetworkConnect(containerName, network); err != nil {
				connectErrs = append(connectErrs, fmt.Errorf("connect global proxy to %s: %w", network, err))
			}
		}
		return errors.Join(connectErrs...)
	}

	image := getenv("RELAY_CADDY_IMAGE", "caddy:alpine")
	// The image and Cloudflare token are only applied at container creation
	// (Docker has no live "docker update --env" for a running container), so
	// track what the container was last created with and force a recreate
	// when either changes — otherwise flipping on CLOUDFLARE_API_TOKEN would
	// silently do nothing until someone manually removes the container.
	markerPath := filepath.Join(configDir, "container.marker")
	desiredMarker := fmt.Sprintf("image=%s cf=%t", image, cfToken != "")
	if s.runtime.IsRunning(containerName) {
		if prev, _ := os.ReadFile(markerPath); string(prev) != desiredMarker {
			s.runtime.Remove(containerName)
		}
	}
	if err := os.WriteFile(markerPath, []byte(desiredMarker), 0644); err != nil {
		return err
	}

	if !s.runtime.IsRunning(containerName) {
		s.runtime.Remove(containerName)
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
		if cfToken != "" {
			spec.Env = append(spec.Env, "CLOUDFLARE_API_TOKEN="+cfToken)
		}
		if err := s.runtime.RunDetached(spec); err != nil {
			return fmt.Errorf("global proxy start: %w", err)
		}
		return connectDockerNetworks()
	}

	connectErr := connectDockerNetworks()
	out, reloadErr := s.runtime.Exec(containerName, []string{"caddy", "reload", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"})
	if reloadErr != nil {
		return errors.Join(connectErr, fmt.Errorf("global proxy reload: %v (%s)", reloadErr, strings.TrimSpace(string(out))))
	}
	return connectErr
}

// repairLegacyAppHostPortsFromRuntime backfills missing persisted host ports for
// upgraded installs by reading the live edge-proxy container port binding.
func (s *Server) repairLegacyAppHostPortsFromRuntime() (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}

	type candidate struct {
		app    string
		env    DeployEnv
		branch string
		engine string
	}

	rows, err := s.db.Query(`
		SELECT app, env, branch, COALESCE(engine,'')
		FROM app_state
		WHERE COALESCE(stopped,0)=0 AND COALESCE(host_port,0)=0`)
	if err != nil {
		return 0, err
	}

	var candidates []candidate
	for rows.Next() {
		var app string
		var envS string
		var branch string
		var engine string
		if err := rows.Scan(&app, &envS, &branch, &engine); err != nil {
			continue
		}

		candidates = append(candidates, candidate{
			app:    app,
			env:    normalizeDeployEnv(envS),
			branch: branch,
			engine: engine,
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}

	repaired := 0
	for _, candidate := range candidates {
		runtime := s.runtimeForEngine(candidate.engine)
		if runtime == nil {
			continue
		}

		containerName := appBaseContainerName(candidate.app, candidate.env, candidate.branch)
		if !runtime.IsRunning(containerName) {
			continue
		}
		published := runtime.PublishedPort(containerName, 3000)
		if published <= 0 {
			continue
		}

		if _, err := s.db.Exec(
			`UPDATE app_state
			 SET host_port=?, updated_at=?
			 WHERE app=? AND env=? AND branch=? AND COALESCE(host_port,0)=0`,
			published, time.Now().UnixMilli(), candidate.app, string(candidate.env), candidate.branch,
		); err != nil {
			return repaired, err
		}
		repaired++
	}
	return repaired, nil
}

func (s *Server) currentActiveSlotWithRuntime(runtime ContainerRuntime, app string, env DeployEnv, branch string, state *AppState) string {
	stored := ""
	if state != nil {
		stored = normalizeActiveSlot(state.ActiveSlot)
		if stored != "" && runtime.IsRunning(appSlotContainerName(app, env, branch, stored)) {
			return stored
		}
	}
	for _, slot := range []string{"blue", "green"} {
		if runtime.IsRunning(appSlotContainerName(app, env, branch, slot)) {
			return slot
		}
	}
	return stored
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

	// Resolve port conflict for restart/runContainer paths that bypass runDeploy.
	// (runDeploy calls resolvePortConflict earlier; the second call is a no-op when the port is free.)
	if !req.HostPortExplicit {
		conflictReq := req
		conflictReq.HostPort = hostPort
		s.resolvePortConflict(s.runtime, &conflictReq, log)
		hostPort = conflictReq.HostPort
	}

	state, _ := s.getAppState(req.App, req.Env, req.Branch)
	activeSlot := s.currentActiveSlot(req.App, req.Env, req.Branch, state)
	nextSlot := nextActiveSlot(activeSlot)
	candidateName := appSlotContainerName(req.App, req.Env, req.Branch, nextSlot)

	// nextSlot is the OTHER of only two slot names (blue/green), reused by
	// every deploy. If it's still running, it can only be a previous
	// deploy's standby that's mid-drain (this deploy started before that
	// one's rolloutDrainDuration elapsed) — runSlotContainer would
	// otherwise force-remove it unconditionally to make room, abruptly
	// cutting off any sticky-session user still pinned there with zero
	// grace. Finish its drain safely first: drop it from the edge proxy's
	// routing map (so nothing new can be sent to it) before removing it,
	// instead of yanking it out from under active traffic.
	if s.runtime.IsRunning(candidateName) {
		if log != nil {
			log("slot %s still draining a previous deploy; finishing that drain before reuse", nextSlot)
		}
		s.retireStandbySlot(req.App, req.Env, req.Branch, activeSlot, nextSlot, servicePort, hostPort, mode, trafficMode, req.PublicHost)
	}

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
	splitPercent := defaultTrafficSplitPercent()
	if state != nil && normalizeActiveSlot(activeSlot) != "" {
		splitPercent = state.TrafficSplitPercent
	}
	if err := s.ensureEdgeProxy(log, req.App, req.Env, req.Branch, networkName, nextSlot, activeSlot, servicePort, hostPort, mode, trafficMode, req.PublicHost, splitPercent, recreateEdge); err != nil {
		if log != nil {
			log("edge proxy failed: %v", err)
		}
		s.runtime.Remove(candidateName)
		return err
	}

	if activeSlot != "" && activeSlot != nextSlot {
		canary := splitPercent > 0 && splitPercent < 100
		drainUntil := time.Now().Add(rolloutDrainDuration()).UnixMilli()
		if canary {
			drainUntil = 0
		}
		oldName := appSlotContainerName(req.App, req.Env, req.Branch, activeSlot)
		if log != nil {
			if canary {
				log("routing %d%% of traffic to new slot %s while keeping %s hot", splitPercent, nextSlot, activeSlot)
			} else {
				log("draining previous slot %s for %s", oldName, rolloutDrainDuration())
			}
		}
		if state != nil {
			state.ActiveSlot = nextSlot
			state.StandbySlot = activeSlot
			state.DrainUntil = drainUntil
			state.TrafficMode = trafficMode
			state.TrafficSplitPercent = splitPercent
			state.HostPort = hostPort
			state.ServicePort = servicePort
			if canary {
				state.RolloutStartedAt = time.Now().UnixMilli()
				state.RolloutDeployID = ""
				state.RolloutStatus = "monitoring"
			} else {
				state.RolloutStartedAt = 0
				state.RolloutDeployID = ""
				state.RolloutStatus = "graduated"
			}
			_ = s.saveAppState(state)
			s.broadcastSnapshot()
		}
		if canary {
			s.startRolloutWatch(req.App, req.Env, req.Branch)
		} else {
			s.cleanupStandbySlotAfter(req.App, req.Env, req.Branch, nextSlot, activeSlot, servicePort, hostPort, mode, trafficMode, req.PublicHost, rolloutDrainDuration())
		}
	} else if state != nil {
		state.ActiveSlot = nextSlot
		state.StandbySlot = ""
		state.DrainUntil = 0
		state.TrafficMode = trafficMode
		state.TrafficSplitPercent = defaultTrafficSplitPercent()
		state.HostPort = hostPort
		state.ServicePort = servicePort
		state.RolloutStartedAt = 0
		state.RolloutDeployID = ""
		state.RolloutStatus = ""
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

func deployLogLooksLikeOOMKill(logPath string, errMsg string) bool {
	combined := strings.ToLower(errMsg)
	if logPath != "" {
		if tail, err := readFileTail(logPath, 256*1024); err == nil {
			combined += "\n" + strings.ToLower(tail)
		}
	}
	return strings.Contains(combined, "exit code: 137") ||
		strings.Contains(combined, "exit status 137") ||
		strings.Contains(combined, "\nkilled\n") ||
		strings.Contains(combined, " killed\n") ||
		strings.Contains(combined, "\nkilled")
}

func readFileTail(path string, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	offset := int64(0)
	if st.Size() > maxBytes {
		offset = st.Size() - maxBytes
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return "", err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
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
			// No users in DB: always send the user through account setup regardless
			// of whether token.txt exists. Token auth still works for API/CLI
			// bearer-token calls; it just doesn't appear as a login option here.
			writeJSON(w, 200, map[string]any{
				"setup_required": true,
				"legacy_mode":    false,
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
	role = normalizeUserRole(role)
	for _, allowed := range allowedRoles {
		if role == normalizeUserRole(allowed) {
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

func (s *Server) userPermissions(userID string) ([]UserPermission, error) {
	rows, err := s.db.Query(
		`SELECT user_id, app, env, role FROM user_permissions WHERE user_id=? ORDER BY app, env`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]UserPermission, 0, 8)
	for rows.Next() {
		var perm UserPermission
		if err := rows.Scan(&perm.UserID, &perm.App, &perm.Env, &perm.Role); err != nil {
			continue
		}
		perm.App = strings.TrimSpace(perm.App)
		perm.Env = string(normalizeDeployEnv(perm.Env))
		perm.Role = normalizeUserRole(perm.Role)
		if perm.App == "" {
			continue
		}
		if perm.Env == "" {
			perm.Env = "*"
		}
		out = append(out, perm)
	}
	return out, nil
}

func normalizeUserPermissions(perms []UserPermission) []UserPermission {
	seen := map[string]UserPermission{}
	for _, perm := range perms {
		perm.App = strings.TrimSpace(perm.App)
		perm.Env = strings.TrimSpace(perm.Env)
		if perm.App == "" {
			continue
		}
		if perm.Env == "" {
			perm.Env = "*"
		}
		if perm.Env != "*" {
			perm.Env = string(normalizeDeployEnv(perm.Env))
		}
		if perm.Env == "" {
			continue
		}
		perm.Role = normalizeUserRole(perm.Role)
		key := perm.App + "\x00" + perm.Env
		if existing, ok := seen[key]; !ok || roleRank(perm.Role) > roleRank(existing.Role) {
			seen[key] = perm
		}
	}
	out := make([]UserPermission, 0, len(seen))
	for _, perm := range seen {
		out = append(out, perm)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].App == out[j].App {
			return out[i].Env < out[j].Env
		}
		return out[i].App < out[j].App
	})
	return out
}

func (s *Server) replaceUserPermissions(userID string, perms []UserPermission) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM user_permissions WHERE user_id=?`, userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	perms = normalizeUserPermissions(perms)
	for _, perm := range perms {
		if _, err := tx.Exec(
			`INSERT INTO user_permissions (user_id, app, env, role) VALUES (?, ?, ?, ?)`,
			userID, perm.App, perm.Env, perm.Role,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Server) userHasScopedPermissions(userID string) bool {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM user_permissions WHERE user_id=?`, userID).Scan(&count)
	return count > 0
}

func (s *Server) scopedLaneRole(userID string, app string, env DeployEnv) string {
	rows, err := s.db.Query(
		`SELECT role FROM user_permissions
		 WHERE user_id=?
		   AND (app=? OR app='*')
		   AND (env=? OR env='*')`,
		userID, app, string(env),
	)
	if err != nil {
		return ""
	}
	defer rows.Close()
	best := ""
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			continue
		}
		role = normalizeUserRole(role)
		if roleRank(role) > roleRank(best) {
			best = role
		}
	}
	return best
}

func (s *Server) effectiveLaneRole(sess *UserSession, app string, env DeployEnv) string {
	if sess == nil {
		return ""
	}
	if normalizeUserRole(sess.Role) == "owner" {
		return "owner"
	}
	if role := s.scopedLaneRole(sess.UserID, app, env); role != "" {
		return role
	}
	if s.userHasScopedPermissions(sess.UserID) {
		return ""
	}
	return normalizeUserRole(sess.Role)
}

func (s *Server) laneAccessAllowed(sess *UserSession, app string, env DeployEnv, requiredRole string) bool {
	if sess == nil {
		return false
	}
	return roleAtLeast(s.effectiveLaneRole(sess, app, env), requiredRole)
}

func (s *Server) requireLaneAccess(w http.ResponseWriter, r *http.Request, app string, env DeployEnv, requiredRole string) (*UserSession, bool) {
	if !s.hasUsers() {
		return nil, true
	}
	sess := s.validateUserSession(r)
	if sess == nil {
		httpError(w, 401, "unauthorized")
		return nil, false
	}
	if !s.laneAccessAllowed(sess, app, env, requiredRole) {
		httpError(w, 403, "insufficient app access")
		return sess, false
	}
	return sess, true
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
		"cli_redirect": fmt.Sprintf("http://127.0.0.1:%d/callback?code=%s&user=%s&role=%s",
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
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Relay-Token, X-Requested-With")
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
			`SELECT id, username, role, COALESCE(created_at,0) FROM users ORDER BY created_at ASC`,
		)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		defer rows.Close()
		type userRow struct {
			ID          string           `json:"id"`
			Username    string           `json:"username"`
			Role        string           `json:"role"`
			CreatedAt   int64            `json:"created_at"`
			Permissions []UserPermission `json:"permissions,omitempty"`
		}
		out := []userRow{}
		for rows.Next() {
			var u userRow
			if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
				continue
			}
			u.Role = normalizeUserRole(u.Role)
			u.Permissions, _ = s.userPermissions(u.ID)
			out = append(out, u)
		}
		writeJSON(w, 200, out)

	case http.MethodPost:
		var body struct {
			Username    string           `json:"username"`
			Password    string           `json:"password"`
			Role        string           `json:"role"`
			Permissions []UserPermission `json:"permissions"`
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
		role := normalizeUserRole(body.Role)
		if role == "viewer" && strings.TrimSpace(body.Role) == "" {
			role = "deployer"
		}
		hash, err := hashPassword(body.Password)
		if err != nil {
			httpError(w, 500, "failed to hash password")
			return
		}
		id := newID()
		tx, err := s.db.Begin()
		if err != nil {
			httpError(w, 500, "db error")
			return
		}
		if _, err = tx.Exec(
			`INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
			id, body.Username, hash, role, time.Now().UnixMilli(),
		); err != nil {
			_ = tx.Rollback()
			httpError(w, 409, "username already exists")
			return
		}
		perms := normalizeUserPermissions(body.Permissions)
		for i := range perms {
			perms[i].UserID = id
		}
		for _, perm := range perms {
			if _, err = tx.Exec(
				`INSERT INTO user_permissions (user_id, app, env, role) VALUES (?, ?, ?, ?)`,
				id, perm.App, perm.Env, perm.Role,
			); err != nil {
				_ = tx.Rollback()
				httpError(w, 500, "failed to save permissions")
				return
			}
		}
		if err = tx.Commit(); err != nil {
			httpError(w, 500, "db error")
			return
		}
		s.auditLog(sess.Username, "user.create", body.Username, "role="+role)
		writeJSON(w, 200, map[string]any{"id": id, "username": body.Username, "role": role, "permissions": perms})

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
			Role        *string          `json:"role"`
			Permissions []UserPermission `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		var username string
		var currentRole string
		if err := s.db.QueryRow(`SELECT username, role FROM users WHERE id=?`, userID).Scan(&username, &currentRole); err != nil {
			httpError(w, 404, "user not found")
			return
		}
		nextRole := normalizeUserRole(currentRole)
		if body.Role != nil {
			nextRole = normalizeUserRole(*body.Role)
			if !validUserRole(nextRole) {
				httpError(w, 400, "role must be owner, admin, deployer, or viewer")
				return
			}
			if _, err := s.db.Exec(`UPDATE users SET role=? WHERE id=?`, nextRole, userID); err != nil {
				httpError(w, 500, err.Error())
				return
			}
		}
		for i := range body.Permissions {
			body.Permissions[i].UserID = userID
		}
		if err := s.replaceUserPermissions(userID, body.Permissions); err != nil {
			httpError(w, 500, "failed to update permissions")
			return
		}
		s.auditLog(sess.Username, "user.role", username, "role="+nextRole)
		writeJSON(w, 200, map[string]any{"id": userID, "role": nextRole, "permissions": normalizeUserPermissions(body.Permissions)})

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

func deployNotificationEvent(status DeployStatus) string {
	switch status {
	case StatusQueued:
		return "deploy.queued"
	case StatusRunning:
		return "deploy.started"
	case StatusSuccess:
		return "deploy.succeeded"
	case StatusFailed:
		return "deploy.failed"
	default:
		return "deploy.updated"
	}
}

func (s *Server) emitDeployNotificationsByID(deployID string) {
	deploy, err := s.getDeployFromDB(deployID)
	if err != nil || deploy == nil {
		return
	}
	st, _ := s.getAppState(deploy.App, deploy.Env, deploy.Branch)
	if st == nil {
		return
	}
	webhooks := notificationWebhookList(st.NotificationWebhooks)
	if len(webhooks) == 0 {
		return
	}
	payload := map[string]any{
		"event": deployNotificationEvent(deploy.Status),
		"deploy": map[string]any{
			"id":             deploy.ID,
			"app":            deploy.App,
			"env":            deploy.Env,
			"branch":         deploy.Branch,
			"status":         deploy.Status,
			"commit_sha":     deploy.CommitSHA,
			"commit_message": deploy.CommitMessage,
			"build_number":   deploy.BuildNumber,
			"preview_url":    deploy.PreviewURL,
			"deployed_by":    deploy.DeployedBy,
			"created_at":     deploy.CreatedAt.UnixMilli(),
			"started_at":     timePtrToMillis(deploy.StartedAt),
			"ended_at":       timePtrToMillis(deploy.EndedAt),
			"error":          deploy.Error,
		},
		"lane": map[string]any{
			"public_host":           st.PublicHost,
			"traffic_split_percent": st.TrafficSplitPercent,
			"rollout_status":        st.RolloutStatus,
			"active_slot":           st.ActiveSlot,
			"standby_slot":          st.StandbySlot,
		},
	}
	body, _ := json.Marshal(payload)
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		for _, endpoint := range webhooks {
			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
			if err != nil {
				fmt.Fprintf(os.Stderr, "notification webhook: build request for %s: %v\n", endpoint, err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "notification webhook: post to %s: %v\n", endpoint, err)
				continue
			}
			if resp.StatusCode >= 400 {
				fmt.Fprintf(os.Stderr, "notification webhook: %s returned %d\n", endpoint, resp.StatusCode)
			}
			_ = resp.Body.Close()
		}
	}()
}

func (s *Server) updateDeployStatus(id string, status DeployStatus, errMsg string, started, ended *time.Time, imageTag, prevImg string) error {
	_, err := s.db.Exec(
		`UPDATE deploys SET status=?, error=?, started_at=?, ended_at=?, image_tag=?, previous_image_tag=? WHERE id=?`,
		string(status), errMsg, timePtrToMillis(started), timePtrToMillis(ended), imageTag, prevImg, id,
	)
	if err == nil {
		s.broadcastSnapshot()
		s.emitDeployNotificationsByID(id)
	}
	return err
}

func failDeploy(s *Server, d *Deploy, err error, msg string) {
	end := time.Now()
	d.Status = StatusFailed
	d.EndedAt = &end
	d.Error = msg
	if dbErr := s.updateDeployStatus(d.ID, d.Status, d.Error, d.StartedAt, d.EndedAt, "", ""); dbErr != nil {
		fmt.Fprintf(os.Stderr, "failDeploy updateDeployStatus %s: %v\n", d.ID, dbErr)
	}
}

func scanDeployRow(scanner interface {
	Scan(dest ...any) error
}) (*Deploy, error) {
	var d Deploy
	var env, st string
	var created, started, ended sql.NullInt64
	var errTxt sql.NullString
	var img, prev sql.NullString
	var purl string
	if err := scanner.Scan(&d.ID, &d.App, &d.RepoURL, &d.Branch, &d.CommitSHA, &env, &st, &created, &started, &ended, &errTxt, &d.LogPath, &img, &prev, &purl, &d.Source, &d.BuildNumber, &d.DeployedBy, &d.CommitMessage); err != nil {
		return nil, err
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
	return &d, nil
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
		d, err := scanDeployRow(rows)
		if err != nil {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *Server) getDeployFromDB(id string) (*Deploy, error) {
	row := s.db.QueryRow(`SELECT d.id, d.app, d.repo_url, d.branch, d.commit_sha, d.env, d.status, d.created_at, d.started_at, d.ended_at, d.error, d.log_path, d.image_tag, d.previous_image_tag, COALESCE(d.preview_url,''), COALESCE(r.source,''), COALESCE(d.build_number,0), COALESCE(d.deployed_by,''), COALESCE(d.commit_message,'')
		FROM deploys d
		LEFT JOIN deploy_requests r ON r.id = d.id
		WHERE d.id=?`, id)

	d, err := scanDeployRow(row)
	if err != nil {
		return nil, err
	}
	return d, nil
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
		return fmt.Errorf("env must be one of: preview, dev, staging, prod (or production; ddev alias accepted)")
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
	if req.WorkspaceBranch != "" && !isKnownDeployEnv(normalizeDeployEnv(string(req.WorkspaceEnv))) {
		return fmt.Errorf("workspace_env required when workspace_branch is set")
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

func defaultTrafficSplitPercent() int {
	return 100
}

func defaultRolloutMinRequests() int {
	return 25
}

func defaultRolloutErrorPercent() float64 {
	return 5
}

func defaultRolloutAssessSeconds() int {
	return 300
}

func normalizeTrafficSplitPercent(value int) int {
	switch {
	case value <= 0:
		return defaultTrafficSplitPercent()
	case value > 100:
		return 100
	default:
		return value
	}
}

func normalizeRolloutMinRequests(value int) int {
	if value <= 0 {
		return defaultRolloutMinRequests()
	}
	return value
}

func normalizeRolloutErrorPercent(value float64) float64 {
	if value <= 0 {
		return defaultRolloutErrorPercent()
	}
	if value > 100 {
		return 100
	}
	return value
}

func normalizeRolloutAssessSeconds(value int) int {
	if value <= 0 {
		return defaultRolloutAssessSeconds()
	}
	return value
}

func normalizeNotificationWebhooks(value string) string {
	lines := strings.Split(value, "\n")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parsed, err := url.Parse(line)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		line = parsed.String()
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func notificationWebhookList(value string) []string {
	value = normalizeNotificationWebhooks(value)
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

func validUserRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner", "admin", "deployer", "viewer":
		return true
	default:
		return false
	}
}

func normalizeUserRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if !validUserRole(role) {
		return "viewer"
	}
	return role
}

func roleRank(role string) int {
	switch normalizeUserRole(role) {
	case "viewer":
		return 1
	case "deployer":
		return 2
	case "admin":
		return 3
	case "owner":
		return 4
	default:
		return 0
	}
}

func roleAtLeast(role string, required string) bool {
	if required == "" {
		return true
	}
	return roleRank(role) >= roleRank(required)
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
	pkg, raw, ok := readPackageManifest(repoDir)
	if !ok {
		return false
	}
	if pkg == nil {
		return strings.Contains(strings.ToLower(raw), `"`+strings.ToLower(dep)+`"`)
	}
	_, ok = pkg.Dependencies[dep]
	if ok {
		return true
	}
	_, ok = pkg.DevDependencies[dep]
	return ok
}

type packageManifest struct {
	PackageManager  string            `json:"packageManager"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]any    `json:"dependencies"`
	DevDependencies map[string]any    `json:"devDependencies"`
}

func readPackageManifest(repoDir string) (*packageManifest, string, bool) {
	p := filepath.Join(repoDir, "package.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, "", false
	}
	var pkg packageManifest
	if err := json.Unmarshal(b, &pkg); err != nil {
		return nil, string(b), true
	}
	return &pkg, string(b), true
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
