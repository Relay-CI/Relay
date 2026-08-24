package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------- Multi-service / Project config ----------------------

// ProjectConfig is read from relay.json at the root of the repo.
// If present, Relay starts companion services (databases, caches) before
// launching the app container, wiring everything on a shared Docker network.
type ProjectConfig struct {
	Project  string          `json:"project"`
	Services []ServiceConfig `json:"services"`
}

// ServiceConfig describes a single companion inside a project.
// Type is one of: "app", "relaydb", "postgres", "mysql", "redis", "mongo".
type ServiceConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Version     string            `json:"version"`           // e.g. "17" for postgres:17
	Profile     string            `json:"profile,omitempty"` // RelayDB: auto, starter, balanced, throughput
	Port        int               `json:"port,omitempty"`    // override default container port
	HostPort    int               `json:"host_port,omitempty"`
	Image       string            `json:"image,omitempty"`
	PoolerImage string            `json:"pooler_image,omitempty"`
	Command     string            `json:"command,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Volumes     []string          `json:"volumes,omitempty"`
	Health      *ServiceHealth    `json:"health,omitempty"`
	Stopped     bool              `json:"stopped,omitempty"`
	Disabled    bool              `json:"disabled,omitempty"`
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
	EnvVal    string `json:"-"` // connection credentials must never leave relayd
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

type companionFailurePolicy int

const (
	companionFailureFatal companionFailurePolicy = iota
	companionFailureWarning
)

type companionOrchestrationResult struct {
	NetworkName string
	Environment map[string]string
	Desired     map[string]ServiceConfig
}

func desiredCompanionMap(services []ServiceConfig) map[string]ServiceConfig {
	desired := make(map[string]ServiceConfig, len(services))
	for _, service := range services {
		service = normalizeServiceConfig(service)
		if service.Name != "" {
			desired[service.Name] = service
		}
	}
	return desired
}

func companionNetworkName(app string, env DeployEnv, branch string) string {
	return fmt.Sprintf("relay-%s-%s-%s", safe(app), safe(string(env)), safe(branch))
}

func (s *Server) orchestrateCompanions(
	log func(string, ...any),
	app string,
	env DeployEnv,
	branch string,
	services []ServiceConfig,
	force bool,
	reconcile bool,
	failurePolicy companionFailurePolicy,
) (companionOrchestrationResult, error) {
	if log == nil {
		log = func(string, ...any) {}
	}
	result := companionOrchestrationResult{
		Environment: map[string]string{},
		Desired:     desiredCompanionMap(services),
	}
	runnable := make([]ServiceConfig, 0, len(services))
	for _, service := range services {
		service = normalizeServiceConfig(service)
		if serviceShouldRun(service) {
			runnable = append(runnable, service)
		}
	}
	if len(runnable) == 0 {
		if reconcile {
			s.reconcileProjectServices(log, app, env, branch, result.Desired)
		}
		return result, nil
	}

	result.NetworkName = companionNetworkName(app, env, branch)
	log("setting up project network: %s", result.NetworkName)
	if err := s.runtime.EnsureNetwork(result.NetworkName); err != nil {
		if failurePolicy == companionFailureFatal {
			return result, err
		}
		log("warning: could not create network: %v", err)
		result.NetworkName = ""
		if reconcile {
			s.reconcileProjectServices(log, app, env, branch, result.Desired)
		}
		return result, nil
	}

	for _, service := range runnable {
		key, value, err := s.startProjectService(log, app, string(env), branch, service, result.NetworkName, force)
		if err != nil {
			if failurePolicy == companionFailureFatal {
				return result, err
			}
			log("warning: service %s failed: %v", service.Name, err)
			continue
		}
		if key != "" && value != "" {
			result.Environment[key] = value
		}
	}
	if reconcile {
		s.reconcileProjectServices(log, app, env, branch, result.Desired)
	}
	return result, nil
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
	if svc.Type == serviceTypeRelayDB {
		svc.Profile = normalizeRelayDBProfile(svc.Profile)
	} else {
		svc.Profile = strings.TrimSpace(svc.Profile)
	}
	svc.Image = strings.TrimSpace(svc.Image)
	svc.PoolerImage = strings.TrimSpace(svc.PoolerImage)
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
	case serviceTypeRelayDB, "postgres", "mysql", "redis", "mongo", "worker", "custom":
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
	case serviceTypeRelayDB:
		if version == "" {
			version = defaultRelayDBVersion
		}
		return "postgres:" + version
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
	case serviceTypeRelayDB:
		return 5432
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
	case serviceTypeRelayDB:
		return relayDBEnvKey(svc.Name),
			fmt.Sprintf("postgresql://relay@%s:%d/relay?sslmode=disable", host, port)
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
		s.removeProjectServiceContainers(*running)
	}
	s.deleteProjectServiceState(app, env, branch, name)
}

func (s *Server) removeProjectServiceContainers(svc ProjectService) {
	if strings.EqualFold(svc.Type, serviceTypeRelayDB) {
		s.runtime.Remove(relayDBPoolContainerName(svc.Container))
		if s.stationRuntime != nil {
			s.stationRuntime.Remove(relayDBPoolContainerName(svc.Container))
		}
	}
	s.runtime.Remove(svc.Container)
	if s.stationRuntime != nil {
		s.stationRuntime.Remove(svc.Container)
	}
}

func (s *Server) appLaneRunning(app string, env DeployEnv, branch string) bool {
	names := []string{
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
	if svc.Type == serviceTypeRelayDB {
		if s.relayDB == nil {
			return "", "", fmt.Errorf("RelayDB manager is not initialized")
		}
		if runtime.IsRunning(containerName) && !force {
			if current, currentErr := s.getProjectService(app, env, branch, svc.Name); currentErr == nil && current != nil && current.SpecHash != serviceConfigHash(svc) {
				force = true
			}
		}
		connection, ensureErr := s.relayDB.Ensure(runtime, relayDBTarget{
			App:           app,
			Env:           env,
			Branch:        branch,
			Name:          svc.Name,
			Network:       networkName,
			PrimaryName:   containerName,
			VolumeName:    volumeName,
			HostPort:      svc.HostPort,
			CustomVolumes: svc.Volumes,
		}, svc, force, log)
		if ensureErr != nil {
			return "", "", ensureErr
		}
		if saveErr := s.saveProjectService(&ProjectService{
			Project: app, Name: svc.Name, Type: svc.Type, Branch: branch, Env: env,
			Container: connection.PrimaryName, Network: networkName, Volume: connection.VolumeName,
			EnvKey: connection.EnvKey, EnvVal: "", Image: connection.PrimaryImage,
			Port: connection.Port, HostPort: svc.HostPort, SpecHash: serviceConfigHash(svc),
		}); saveErr != nil {
			return "", "", fmt.Errorf("save RelayDB state: %w", saveErr)
		}
		log("RelayDB %s ready: profile=%s pooled_url=%s", svc.Name, connection.Profile.Name, connection.EnvKey)
		return connection.EnvKey, connection.URL, nil
	}
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
