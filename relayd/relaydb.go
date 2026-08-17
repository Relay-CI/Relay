package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	serviceTypeRelayDB       = "relaydb"
	defaultRelayDBVersion    = "17"
	defaultPgBouncerImage    = "edoburu/pgbouncer:v1.25.2-p0"
	relayDBCredentialKeyFile = "relaydb.key"
)

type relayDBProfile struct {
	Name                 string
	MemoryMB             int
	CPUs                 string
	SharedBuffersMB      int
	EffectiveCacheMB     int
	WorkMemMB            int
	MaintenanceWorkMemMB int
	MaxDBConnections     int
	MaxClientConnections int
	DefaultPoolSize      int
	ReservePoolSize      int
}

type relayDBTarget struct {
	App           string
	Env           string
	Branch        string
	Name          string
	Network       string
	PrimaryName   string
	VolumeName    string
	HostPort      int
	CustomVolumes []string
}

type relayDBConnection struct {
	EnvKey       string
	URL          string
	PrimaryName  string
	PoolerName   string
	VolumeName   string
	PrimaryImage string
	Port         int
	Profile      relayDBProfile
}

// relayDBManager is the deep module for a RelayDB workload. Its interface is
// one database intent; credential persistence, topology, tuning, readiness,
// and the app connection contract stay inside the implementation.
type relayDBManager struct {
	db           *sql.DB
	key          []byte
	hostMemoryMB func() int
}

func newRelayDBManager(db *sql.DB, key []byte, hostMemoryMB func() int) *relayDBManager {
	return &relayDBManager{db: db, key: key, hostMemoryMB: hostMemoryMB}
}

func relayDBPoolContainerName(primary string) string {
	return primary + "__pool"
}

func normalizeRelayDBProfile(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "starter", "balanced", "throughput":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "auto"
	}
}

func resolveRelayDBProfile(value string, hostMemoryMB int) relayDBProfile {
	name := normalizeRelayDBProfile(value)
	if name == "auto" {
		switch {
		case hostMemoryMB >= 32768:
			name = "throughput"
		case hostMemoryMB >= 8192:
			name = "balanced"
		default:
			name = "starter"
		}
	}
	switch name {
	case "throughput":
		return relayDBProfile{
			Name: "throughput", MemoryMB: 8192, CPUs: "4",
			SharedBuffersMB: 2048, EffectiveCacheMB: 6144,
			WorkMemMB: 16, MaintenanceWorkMemMB: 512,
			MaxDBConnections: 200, MaxClientConnections: 10000,
			DefaultPoolSize: 100, ReservePoolSize: 20,
		}
	case "balanced":
		return relayDBProfile{
			Name: "balanced", MemoryMB: 2048, CPUs: "2",
			SharedBuffersMB: 512, EffectiveCacheMB: 1536,
			WorkMemMB: 8, MaintenanceWorkMemMB: 128,
			MaxDBConnections: 100, MaxClientConnections: 2000,
			DefaultPoolSize: 50, ReservePoolSize: 10,
		}
	default:
		return relayDBProfile{
			Name: "starter", MemoryMB: 512, CPUs: "1",
			SharedBuffersMB: 128, EffectiveCacheMB: 384,
			WorkMemMB: 4, MaintenanceWorkMemMB: 64,
			MaxDBConnections: 50, MaxClientConnections: 500,
			DefaultPoolSize: 20, ReservePoolSize: 5,
		}
	}
}

func relayDBPostgresCommand(profile relayDBProfile) []string {
	return []string{
		"postgres",
		"-c", fmt.Sprintf("shared_buffers=%dMB", profile.SharedBuffersMB),
		"-c", fmt.Sprintf("effective_cache_size=%dMB", profile.EffectiveCacheMB),
		"-c", fmt.Sprintf("work_mem=%dMB", profile.WorkMemMB),
		"-c", fmt.Sprintf("maintenance_work_mem=%dMB", profile.MaintenanceWorkMemMB),
		"-c", fmt.Sprintf("max_connections=%d", profile.MaxDBConnections),
		"-c", "wal_compression=on",
		"-c", "checkpoint_completion_target=0.9",
		"-c", "random_page_cost=1.1",
		"-c", "effective_io_concurrency=200",
		"-c", "jit=off",
	}
}

func relayDBDefaultHealth() *ServiceHealth {
	return &ServiceHealth{
		Test:               "pg_isready -U relay -d relay",
		IntervalSeconds:    5,
		TimeoutSeconds:     3,
		Retries:            12,
		StartPeriodSeconds: 10,
	}
}

func relayDBPoolHealth() *ServiceHealth {
	return &ServiceHealth{
		Test:               "pg_isready -h 127.0.0.1 -p 5432 -U relay -d relay",
		IntervalSeconds:    5,
		TimeoutSeconds:     3,
		Retries:            12,
		StartPeriodSeconds: 5,
	}
}

func (m *relayDBManager) Ensure(runtime ContainerRuntime, target relayDBTarget, svc ServiceConfig, force bool, log func(string, ...any)) (relayDBConnection, error) {
	svc = normalizeServiceConfig(svc)
	profile := resolveRelayDBProfile(svc.Profile, m.hostMemory())
	username, password, databaseName, err := m.loadOrCreateCredential(target)
	if err != nil {
		return relayDBConnection{}, fmt.Errorf("RelayDB credentials: %w", err)
	}

	version := firstNonEmpty(strings.TrimSpace(svc.Version), defaultRelayDBVersion)
	primaryImage := firstNonEmpty(strings.TrimSpace(svc.Image), "postgres:"+version)
	poolerImage := firstNonEmpty(strings.TrimSpace(svc.PoolerImage), getenv("RELAY_PGBOUNCER_IMAGE", defaultPgBouncerImage))
	poolerName := relayDBPoolContainerName(target.PrimaryName)
	backendURL := postgresConnectionURL(username, password, target.PrimaryName, 5432, databaseName, target)
	appURL := postgresConnectionURL(username, password, poolerName, 5432, databaseName, target)

	connection := relayDBConnection{
		EnvKey:       relayDBEnvKey(svc.Name),
		URL:          appURL,
		PrimaryName:  target.PrimaryName,
		PoolerName:   poolerName,
		VolumeName:   target.VolumeName,
		PrimaryImage: primaryImage,
		Port:         5432,
		Profile:      profile,
	}

	if !force && runtime.IsRunning(target.PrimaryName) && runtime.IsRunning(poolerName) {
		_ = runtime.NetworkConnect(target.PrimaryName, target.Network)
		_ = runtime.NetworkConnect(poolerName, target.Network)
		if err := waitForRelayDBReady(runtime, poolerName, 5432, 10*time.Second); err == nil {
			return connection, nil
		}
	}

	runtime.Remove(poolerName)
	runtime.Remove(target.PrimaryName)

	primaryEnv := []string{
		"POSTGRES_USER=" + username,
		"POSTGRES_PASSWORD=" + password,
		"POSTGRES_DB=" + databaseName,
		"POSTGRES_INITDB_ARGS=--data-checksums --encoding=UTF8",
	}
	for key, value := range svc.Env {
		if isRelayDBManagedEnv(key) {
			continue
		}
		primaryEnv = append(primaryEnv, key+"="+value)
	}
	volumes := append([]string(nil), target.CustomVolumes...)
	if len(volumes) == 0 {
		volumes = []string{target.VolumeName + ":/var/lib/postgresql/data"}
	}
	command := relayDBPostgresCommand(profile)
	if strings.TrimSpace(svc.Command) != "" {
		command = strings.Fields(svc.Command)
	}
	health := svc.Health
	if health == nil || strings.TrimSpace(health.Test) == "" {
		health = relayDBDefaultHealth()
	}
	primarySpec := ContainerSpec{
		Name:          target.PrimaryName,
		Image:         primaryImage,
		Network:       target.Network,
		RestartPolicy: "unless-stopped",
		Env:           primaryEnv,
		Volumes:       volumes,
		HealthArgs:    healthArgs(health),
		Command:       command,
		CPULimit:      profile.CPUs,
		MemLimit:      strconv.Itoa(profile.MemoryMB) + "m",
	}
	if log != nil {
		log("starting RelayDB primary (profile=%s memory=%dMB image=%s)", profile.Name, profile.MemoryMB, primaryImage)
	}
	if err := runtime.RunDetached(primarySpec); err != nil {
		return relayDBConnection{}, fmt.Errorf("start RelayDB primary: %w", err)
	}
	if err := waitForRelayDBReady(runtime, target.PrimaryName, 5432, 60*time.Second); err != nil {
		runtime.Remove(target.PrimaryName)
		return relayDBConnection{}, fmt.Errorf("RelayDB primary readiness: %w", err)
	}

	poolerEnv := []string{
		"DATABASE_URL=" + backendURL,
		"POOL_MODE=transaction",
		"MAX_CLIENT_CONN=" + strconv.Itoa(profile.MaxClientConnections),
		"DEFAULT_POOL_SIZE=" + strconv.Itoa(profile.DefaultPoolSize),
		"MIN_POOL_SIZE=" + strconv.Itoa(min(5, profile.DefaultPoolSize)),
		"RESERVE_POOL_SIZE=" + strconv.Itoa(profile.ReservePoolSize),
		"MAX_PREPARED_STATEMENTS=100",
		"SERVER_RESET_QUERY=DISCARD ALL",
		"IGNORE_STARTUP_PARAMETERS=extra_float_digits",
	}
	poolerSpec := ContainerSpec{
		Name:          poolerName,
		Image:         poolerImage,
		Network:       target.Network,
		RestartPolicy: "unless-stopped",
		Env:           poolerEnv,
		HealthArgs:    healthArgs(relayDBPoolHealth()),
		CPULimit:      "0.5",
		MemLimit:      "128m",
	}
	if target.HostPort > 0 {
		poolerSpec.PortBindings = []string{fmt.Sprintf("%d:5432", target.HostPort)}
	}
	if log != nil {
		log("starting RelayDB pooler (mode=transaction max_clients=%d image=%s)", profile.MaxClientConnections, poolerImage)
	}
	if err := runtime.RunDetached(poolerSpec); err != nil {
		runtime.Remove(target.PrimaryName)
		return relayDBConnection{}, fmt.Errorf("start RelayDB pooler: %w", err)
	}
	if err := waitForRelayDBReady(runtime, poolerName, 5432, 30*time.Second); err != nil {
		runtime.Remove(poolerName)
		runtime.Remove(target.PrimaryName)
		return relayDBConnection{}, fmt.Errorf("RelayDB pooler readiness: %w", err)
	}
	return connection, nil
}

func (m *relayDBManager) hostMemory() int {
	if m.hostMemoryMB == nil {
		return 0
	}
	return m.hostMemoryMB()
}

func isRelayDBManagedEnv(key string) bool {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_INITDB_ARGS":
		return true
	default:
		return false
	}
}

func relayDBEnvKey(name string) string {
	if strings.EqualFold(strings.TrimSpace(name), "db") {
		return "DATABASE_URL"
	}
	return strings.ToUpper(safe(strings.TrimSpace(name))) + "_URL"
}

func postgresConnectionURL(username, password, host string, port int, databaseName string, target relayDBTarget) string {
	u := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   "/" + databaseName,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	q.Set("application_name", safe(target.App+"_"+target.Env+"_"+target.Branch))
	u.RawQuery = q.Encode()
	return u.String()
}

func waitForRelayDBReady(runtime ContainerRuntime, name string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	lastProbe := ""
	for time.Now().Before(deadline) {
		if runtime.IsRunning(name) {
			out, err := runtime.Exec(name, []string{
				"pg_isready", "-h", "127.0.0.1", "-p", strconv.Itoa(port), "-U", "relay", "-d", "relay",
			})
			if err == nil {
				return nil
			}
			lastProbe = strings.TrimSpace(string(out))
		} else if runtime.ContainerExists(name) {
			return fmt.Errorf("container %s exited before accepting connections", name)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if lastProbe != "" {
		return fmt.Errorf("container %s did not become ready on port %d within %s: %s", name, port, timeout, lastProbe)
	}
	return fmt.Errorf("container %s did not become ready on port %d within %s", name, port, timeout)
}

func (m *relayDBManager) loadOrCreateCredential(target relayDBTarget) (string, string, string, error) {
	const username = "relay"
	const databaseName = "relay"
	var stored string
	err := m.db.QueryRow(
		`SELECT password FROM service_credentials WHERE project=? AND env=? AND branch=? AND name=?`,
		target.App, target.Env, target.Branch, target.Name,
	).Scan(&stored)
	if err == nil {
		password, decErr := decryptRelayDBSecret(m.key, stored)
		return username, password, databaseName, decErr
	}
	if err != sql.ErrNoRows {
		return "", "", "", err
	}

	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", err
	}
	password := base64.RawURLEncoding.EncodeToString(raw)
	encrypted, err := encryptRelayDBSecret(m.key, password)
	if err != nil {
		return "", "", "", err
	}
	_, err = m.db.Exec(
		`INSERT OR IGNORE INTO service_credentials (project, env, branch, name, username, password, database_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		target.App, target.Env, target.Branch, target.Name, username, encrypted, databaseName, time.Now().UnixMilli(),
	)
	if err != nil {
		return "", "", "", err
	}
	if err := m.db.QueryRow(
		`SELECT password FROM service_credentials WHERE project=? AND env=? AND branch=? AND name=?`,
		target.App, target.Env, target.Branch, target.Name,
	).Scan(&stored); err != nil {
		return "", "", "", err
	}
	password, err = decryptRelayDBSecret(m.key, stored)
	return username, password, databaseName, err
}

func loadOrCreateRelayDBKey(dataDir string) ([]byte, error) {
	path := filepath.Join(dataDir, relayDBCredentialKeyFile)
	if raw, err := os.ReadFile(path); err == nil {
		decoded, decErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
		if decErr != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("invalid RelayDB key file %s", path)
		}
		return decoded, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	encoded := base64.RawStdEncoding.EncodeToString(key) + "\n"
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return nil, err
	}
	return key, nil
}

func encryptRelayDBSecret(key []byte, plaintext string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("RelayDB encryption key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "enc:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptRelayDBSecret(key []byte, ciphertext string) (string, error) {
	if len(key) != 32 || !strings.HasPrefix(ciphertext, "enc:") {
		return "", fmt.Errorf("RelayDB credential is not encrypted with the active key")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:"))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("RelayDB credential ciphertext is truncated")
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
