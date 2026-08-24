package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type AppState struct {
	App                  string    `json:"app"`
	Env                  DeployEnv `json:"env"`
	Branch               string    `json:"branch"`
	RepoURL              string    `json:"repo_url"`
	ProjectRoot          string    `json:"project_root,omitempty"`
	BuildContext         string    `json:"build_context,omitempty"`
	Dockerfile           string    `json:"dockerfile,omitempty"`
	Engine               string    `json:"engine,omitempty"`
	CurrentImage         string    `json:"current_image,omitempty"`
	PreviousImage        string    `json:"previous_image,omitempty"`
	Mode                 string    `json:"mode"`
	HostPort             int       `json:"host_port"`
	HostPortExplicit     bool      `json:"host_port_explicit,omitempty"`
	ServicePort          int       `json:"service_port"`
	PublicHost           string    `json:"public_host"`
	PublicHosts          []string  `json:"public_hosts,omitempty"`
	ActiveSlot           string    `json:"active_slot,omitempty"`
	StandbySlot          string    `json:"standby_slot,omitempty"`
	DrainUntil           int64     `json:"drain_until,omitempty"`
	TrafficMode          string    `json:"traffic_mode"`
	AccessPolicy         string    `json:"access_policy,omitempty"`
	IPAllowlist          string    `json:"ip_allowlist,omitempty"`
	ExpiresAt            int64     `json:"expires_at,omitempty"`
	RepoHash             string    `json:"repo_hash,omitempty"`
	WebhookSecret        string    `json:"webhook_secret,omitempty"`
	NotificationWebhooks string    `json:"notification_webhooks,omitempty"`
	TrafficSplitPercent  int       `json:"traffic_split_percent,omitempty"`
	RolloutMinRequests   int       `json:"rollout_min_requests,omitempty"`
	RolloutErrorPercent  float64   `json:"rollout_error_percent,omitempty"`
	RolloutAssessSeconds int       `json:"rollout_assess_seconds,omitempty"`
	RolloutStartedAt     int64     `json:"rollout_started_at,omitempty"`
	RolloutDeployID      string    `json:"rollout_deploy_id,omitempty"`
	RolloutStatus        string    `json:"rollout_status,omitempty"`
	Stopped              bool      `json:"stopped,omitempty"`
	CPULimit             string    `json:"cpu_limit,omitempty"`
	MemLimit             string    `json:"mem_limit,omitempty"`
	ResourceMode         string    `json:"resource_mode,omitempty"`
	Volumes              []string  `json:"volumes,omitempty"`
	BuildpackKind        string    `json:"buildpack_kind,omitempty"` // e.g. "sveltekit", "django" — from the last successful build's BuildPlan.Kind
	GitToken             string    `json:"-"`                        // stored in DB but never in API responses
}

type successfulLaneTransition struct {
	Request            DeployRequest
	Current            *AppState
	Previous           *AppState
	Engine             string
	BuildpackKind      string
	CurrentImage       string
	PreviousImage      string
	RepoHash           string
	ConfiguredVolumes  []string
	ActiveSlotFallback string
	DeployID           string
	Stopped            bool
}

type rollbackLaneTransition struct {
	Request            DeployRequest
	Current            *AppState
	Engine             string
	CurrentImage       string
	PreviousImage      string
	ActiveSlotFallback string
}

func cloneLaneState(primary, fallback *AppState) AppState {
	var state AppState
	if fallback != nil {
		state = *fallback
		state.PublicHosts = append([]string(nil), fallback.PublicHosts...)
		state.Volumes = append([]string(nil), fallback.Volumes...)
	}
	if primary != nil {
		state = *primary
		state.PublicHosts = append([]string(nil), primary.PublicHosts...)
		state.Volumes = append([]string(nil), primary.Volumes...)
	}
	return state
}

func buildSuccessfulLaneState(in successfulLaneTransition) *AppState {
	state := cloneLaneState(in.Current, in.Previous)
	req := in.Request
	state.App = req.App
	state.Env = req.Env
	state.Branch = req.Branch
	if strings.TrimSpace(req.RepoURL) != "" {
		state.RepoURL = req.RepoURL
	}
	state.Engine = in.Engine
	state.BuildpackKind = in.BuildpackKind
	state.CurrentImage = in.CurrentImage
	state.PreviousImage = in.PreviousImage
	state.Mode = firstNonEmpty(req.Mode, firstNonEmpty(state.Mode, "port"))
	state.TrafficMode = firstNonEmpty(normalizeTrafficMode(req.TrafficMode), firstNonEmpty(normalizeTrafficMode(state.TrafficMode), "edge"))
	state.HostPort = firstNonZero(req.HostPort, firstNonZero(state.HostPort, defaultHostPort(req.Env)))
	if req.HostPort > 0 {
		state.HostPortExplicit = persistedHostPortExplicit(req, in.Current, in.Previous)
	}
	state.ServicePort = firstNonZero(req.ServicePort, state.ServicePort)
	if strings.TrimSpace(req.PublicHost) != "" {
		state.PublicHost = req.PublicHost
	}
	state.PublicHosts = persistedPublicHosts(req, in.Current, in.Previous)
	if normalizeActiveSlot(state.ActiveSlot) == "" {
		state.ActiveSlot = in.ActiveSlotFallback
	}
	state.RepoHash = in.RepoHash
	if len(in.ConfiguredVolumes) > 0 {
		state.Volumes = append([]string(nil), in.ConfiguredVolumes...)
	}
	if state.TrafficSplitPercent == 0 {
		state.TrafficSplitPercent = defaultTrafficSplitPercent()
	}
	if state.RolloutMinRequests == 0 {
		state.RolloutMinRequests = defaultRolloutMinRequests()
	}
	if state.RolloutErrorPercent == 0 {
		state.RolloutErrorPercent = defaultRolloutErrorPercent()
	}
	if state.RolloutAssessSeconds == 0 {
		state.RolloutAssessSeconds = defaultRolloutAssessSeconds()
	}
	if state.RolloutStatus == "monitoring" || state.RolloutDeployID == "" {
		state.RolloutDeployID = in.DeployID
	}
	state.Stopped = in.Stopped
	return &state
}

func buildRollbackLaneState(in rollbackLaneTransition) *AppState {
	state := cloneLaneState(in.Current, nil)
	req := in.Request
	state.App = req.App
	state.Env = req.Env
	state.Branch = req.Branch
	if state.RepoURL == "" {
		state.RepoURL = req.RepoURL
	}
	state.Engine = in.Engine
	state.CurrentImage = in.CurrentImage
	state.PreviousImage = in.PreviousImage
	state.Mode = firstNonEmpty(req.Mode, firstNonEmpty(state.Mode, "port"))
	state.TrafficMode = firstNonEmpty(normalizeTrafficMode(req.TrafficMode), firstNonEmpty(normalizeTrafficMode(state.TrafficMode), "edge"))
	state.HostPort = firstNonZero(req.HostPort, firstNonZero(state.HostPort, defaultHostPort(req.Env)))
	if req.HostPort > 0 {
		state.HostPortExplicit = persistedHostPortExplicit(req, in.Current)
	}
	state.ServicePort = firstNonZero(req.ServicePort, state.ServicePort)
	if strings.TrimSpace(req.PublicHost) != "" {
		state.PublicHost = req.PublicHost
	}
	state.PublicHosts = persistedPublicHosts(req, in.Current)
	if normalizeActiveSlot(state.ActiveSlot) == "" {
		state.ActiveSlot = in.ActiveSlotFallback
	}
	state.Stopped = false
	return &state
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

func persistedHostPortExplicit(req DeployRequest, states ...*AppState) bool {
	if req.HostPort <= 0 {
		return false
	}
	if req.HostPortExplicit {
		return true
	}
	for _, state := range states {
		if state != nil && state.HostPortExplicit && req.HostPort == state.HostPort {
			return true
		}
	}
	return false
}

// Public host aliases are lane configuration, not deploy input. Most deploy
// triggers only carry the primary host, so preserve the full configured set
// whenever a state replacement is written.
func persistedPublicHosts(req DeployRequest, states ...*AppState) []string {
	hosts := req.PublicHosts
	if len(hosts) == 0 {
		for _, state := range states {
			if state != nil && len(state.PublicHosts) > 0 {
				hosts = state.PublicHosts
				break
			}
		}
	}
	_, hosts = canonicalizePublicHosts(req.PublicHost, hosts)
	return hosts
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

// resolvePortConflict ensures req.HostPort is not taken by a process or container
// other than our own edge proxy. If it is, the next free port is chosen and logged.
// Called for all envs; assignPreviewHostPort handles preview-specific auto-assignment.
func (s *Server) resolvePortConflict(runtime ContainerRuntime, req *DeployRequest, log func(string, ...any)) {
	if req == nil || req.HostPortExplicit {
		return
	}
	preferred := firstNonZero(req.HostPort, defaultHostPort(req.Env))
	if preferred <= 0 {
		return
	}
	// If our own edge proxy already owns a port, preserve it to avoid
	// recreating the nginx container (which causes a brief traffic gap).
	edgeContainer := appBaseContainerName(req.App, req.Env, req.Branch)
	if runtime != nil && runtime.IsRunning(edgeContainer) {
		existingPort := runtime.PublishedPort(edgeContainer, 3000)
		if existingPort == preferred {
			return // Our proxy already owns the preferred port.
		}
		if existingPort > 0 {
			// Our proxy is live on a different port; keep it to avoid a
			// recreate-induced downtime window while nginx restarts.
			req.HostPort = existingPort
			return
		}
	}
	if hostPortAvailable(preferred) {
		req.HostPort = preferred
		return
	}
	// Port in use by something else; find nearest free one.
	if chosen := firstAvailableHostPort(preferred+1, 256); chosen > 0 {
		if log != nil {
			log("host port %d already in use; auto-assigned %d", preferred, chosen)
		}
		req.HostPort = chosen
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

func parsePublicHosts(raw string) []string {
	return normalizePublicHosts(strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '\t', ' ':
			return true
		default:
			return false
		}
	}))
}

func normalizePublicHosts(hosts []string) []string {
	if len(hosts) == 0 {
		return nil
	}
	out := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		normalized := normalizedHostname(host)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeAnalyticsHosts(hosts []string) []string {
	if len(hosts) == 0 {
		return nil
	}
	out := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		normalized := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodePublicHosts(hosts []string) string {
	return strings.Join(normalizePublicHosts(hosts), "\n")
}

func encodeVolumes(vols []string) string {
	out := make([]string, 0, len(vols))
	for _, v := range vols {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return strings.Join(out, "\n")
}

func parseVolumes(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, "\n")
	out := make([]string, 0, len(parts))
	for _, v := range parts {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// expandAppVolume resolves a volume entry from relay.config.json into a
// Docker -v argument. A bare container path like "/data" becomes a
// relay-managed named volume so data survives deploys and rollbacks.
// Explicit bindings ("name:/data" or "/host/path:/data") pass through as-is.
func expandAppVolume(app, env, branch, v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.Contains(v, ":") {
		return v
	}
	target := v
	if !strings.HasPrefix(target, "/") {
		target = "/" + target
	}
	pathPart := safe(strings.TrimPrefix(target, "/"))
	if pathPart == "" {
		pathPart = "data"
	}
	return fmt.Sprintf("relay__%s__%s__%s__%s:%s", safe(app), safe(env), safe(branch), pathPart, target)
}

func canonicalizePublicHosts(primary string, hosts []string) (string, []string) {
	combined := make([]string, 0, len(hosts)+1)
	if strings.TrimSpace(primary) != "" {
		combined = append(combined, primary)
	}
	combined = append(combined, hosts...)
	normalized := normalizePublicHosts(combined)
	if len(normalized) == 0 {
		return "", nil
	}
	return normalized[0], normalized
}

func samePublicHosts(a, b []string) bool {
	a = normalizePublicHosts(a)
	b = normalizePublicHosts(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validateAppPublicHosts(hosts []string, requestHost string, dashboardHost string) error {
	for _, host := range normalizePublicHosts(hosts) {
		if err := validateProxyHostname(host, "public_hosts"); err != nil {
			return err
		}
		if host != "" && (host == normalizedHostname(requestHost) || host == normalizedHostname(dashboardHost)) {
			return fmt.Errorf("public_hosts cannot include the Relay dashboard host; use a different subdomain for apps")
		}
	}
	return nil
}

func (s *Server) publicHostsForApp(app string) ([]string, error) {
	rows, err := s.db.Query(`SELECT public_host, COALESCE(public_hosts,'') FROM app_state WHERE app=?`, app)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []string
	for rows.Next() {
		var publicHost string
		var publicHostsRaw string
		if err := rows.Scan(&publicHost, &publicHostsRaw); err != nil {
			continue
		}
		if publicHost != "" {
			hosts = append(hosts, publicHost)
		}
		hosts = append(hosts, parsePublicHosts(publicHostsRaw)...)
	}
	return normalizePublicHosts(hosts), rows.Err()
}

func (s *Server) analyticsHostsForSelection(app string, env DeployEnv, branch string) ([]string, error) {
	if app == "" {
		return nil, nil
	}
	if env == "" || strings.TrimSpace(branch) == "" {
		return s.publicHostsForApp(app)
	}
	st, err := s.getAppState(app, env, branch)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, nil
	}
	hosts := append([]string{}, analyticsWindowHost(st.PublicHost, st.HostPort)...)
	hosts = append(hosts, st.PublicHosts...)
	return normalizeAnalyticsHosts(hosts), nil
}

func (s *Server) saveAppState(st *AppState) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO app_state
		(app, env, branch, repo_url, project_root, build_context, dockerfile, engine, current_image, previous_image, mode, host_port, host_port_explicit, service_port, public_host, public_hosts, active_slot, standby_slot, drain_until, traffic_mode, access_policy, ip_allowlist, repo_hash, expires_at, webhook_secret, notification_webhooks, traffic_split_percent, rollout_min_requests, rollout_error_percent, rollout_assess_seconds, rollout_started_at, rollout_deploy_id, rollout_status, stopped, cpu_limit, mem_limit, resource_mode, volumes, buildpack_kind, git_token, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		st.App, string(st.Env), st.Branch, st.RepoURL, st.ProjectRoot, st.BuildContext, st.Dockerfile, firstNonEmptyEngine(st.Engine), st.CurrentImage, st.PreviousImage, st.Mode,
		st.HostPort, st.HostPortExplicit, st.ServicePort, st.PublicHost, encodePublicHosts(st.PublicHosts), normalizeActiveSlot(st.ActiveSlot), normalizeActiveSlot(st.StandbySlot), st.DrainUntil, firstNonEmpty(normalizeTrafficMode(st.TrafficMode), "edge"), firstNonEmpty(normalizeAccessPolicy(st.AccessPolicy), s.lanePolicy(st.Env).DefaultAccessPolicy), normalizeIPAllowlist(st.IPAllowlist), st.RepoHash, st.ExpiresAt, st.WebhookSecret, st.NotificationWebhooks, st.TrafficSplitPercent, st.RolloutMinRequests, st.RolloutErrorPercent, st.RolloutAssessSeconds, st.RolloutStartedAt, st.RolloutDeployID, st.RolloutStatus, st.Stopped, strings.TrimSpace(st.CPULimit), strings.TrimSpace(st.MemLimit), strings.TrimSpace(st.ResourceMode), encodeVolumes(st.Volumes), strings.TrimSpace(st.BuildpackKind), st.GitToken, time.Now().UnixMilli(),
	)
	return err
}

// saveRolloutGraduation persists only the columns rollout graduation touches,
// as a targeted UPDATE rather than saveAppState's full-row INSERT OR REPLACE.
// graduateCanary runs on a background timer and can land after a concurrent
// deploy or config save already wrote a newer full row for this lane; a
// full-row REPLACE here would silently clobber every other column (e.g. a
// just-deployed current_image) with this goroutine's now-stale copy. Only
// touching the columns actually being changed makes the two writers
// commutative instead of last-write-wins.
func (s *Server) saveRolloutGraduation(st *AppState) error {
	_, err := s.db.Exec(
		`UPDATE app_state SET traffic_split_percent=?, rollout_started_at=?, rollout_status=?, updated_at=?
		 WHERE app=? AND env=? AND branch=?`,
		st.TrafficSplitPercent, st.RolloutStartedAt, st.RolloutStatus, time.Now().UnixMilli(),
		st.App, string(st.Env), st.Branch,
	)
	return err
}

// saveRolloutRollback is saveRolloutGraduation's counterpart for rollbackCanary
// — same rationale, scoped to the columns a rollback actually changes.
func (s *Server) saveRolloutRollback(st *AppState) error {
	_, err := s.db.Exec(
		`UPDATE app_state SET active_slot=?, standby_slot=?, drain_until=?, traffic_split_percent=?, rollout_started_at=?, rollout_status=?, current_image=?, updated_at=?
		 WHERE app=? AND env=? AND branch=?`,
		normalizeActiveSlot(st.ActiveSlot), normalizeActiveSlot(st.StandbySlot), st.DrainUntil, st.TrafficSplitPercent, st.RolloutStartedAt, st.RolloutStatus, st.CurrentImage, time.Now().UnixMilli(),
		st.App, string(st.Env), st.Branch,
	)
	return err
}

func (s *Server) getAppState(app string, env DeployEnv, branch string) (*AppState, error) {
	row := s.db.QueryRow(`SELECT app, env, branch, repo_url, COALESCE(project_root,''), COALESCE(build_context,''), COALESCE(dockerfile,''), COALESCE(engine,''), current_image, previous_image, mode, host_port, COALESCE(host_port_explicit,0), service_port, public_host, COALESCE(public_hosts,''), COALESCE(active_slot,''), COALESCE(standby_slot,''), COALESCE(drain_until,0), COALESCE(traffic_mode,''), COALESCE(access_policy,''), COALESCE(ip_allowlist,''), COALESCE(repo_hash,''), COALESCE(expires_at,0), COALESCE(webhook_secret,''), COALESCE(notification_webhooks,''), COALESCE(traffic_split_percent,100), COALESCE(rollout_min_requests,25), COALESCE(rollout_error_percent,5), COALESCE(rollout_assess_seconds,300), COALESCE(rollout_started_at,0), COALESCE(rollout_deploy_id,''), COALESCE(rollout_status,''), COALESCE(stopped,0), COALESCE(cpu_limit,''), COALESCE(mem_limit,''), COALESCE(resource_mode,''), COALESCE(volumes,''), COALESCE(buildpack_kind,''), COALESCE(git_token,'')
		FROM app_state WHERE app=? AND env=? AND branch=?`, app, string(env), branch)

	var st AppState
	var envS string
	var publicHostsRaw string
	var volumesRaw string
	if err := row.Scan(&st.App, &envS, &st.Branch, &st.RepoURL, &st.ProjectRoot, &st.BuildContext, &st.Dockerfile, &st.Engine, &st.CurrentImage, &st.PreviousImage, &st.Mode, &st.HostPort, &st.HostPortExplicit, &st.ServicePort, &st.PublicHost, &publicHostsRaw, &st.ActiveSlot, &st.StandbySlot, &st.DrainUntil, &st.TrafficMode, &st.AccessPolicy, &st.IPAllowlist, &st.RepoHash, &st.ExpiresAt, &st.WebhookSecret, &st.NotificationWebhooks, &st.TrafficSplitPercent, &st.RolloutMinRequests, &st.RolloutErrorPercent, &st.RolloutAssessSeconds, &st.RolloutStartedAt, &st.RolloutDeployID, &st.RolloutStatus, &st.Stopped, &st.CPULimit, &st.MemLimit, &st.ResourceMode, &volumesRaw, &st.BuildpackKind, &st.GitToken); err != nil {
		return nil, err
	}
	st.Env = DeployEnv(envS)
	st.PublicHosts = parsePublicHosts(publicHostsRaw)
	st.Volumes = parseVolumes(volumesRaw)
	s.constrainAppState(&st)
	return &st, nil
}
