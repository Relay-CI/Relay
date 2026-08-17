package main

import (
	"bytes"
	"database/sql"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRelayDBProfileAutoUsesHostMemory(t *testing.T) {
	tests := []struct {
		hostMB int
		want   string
	}{
		{hostMB: 2048, want: "starter"},
		{hostMB: 8192, want: "balanced"},
		{hostMB: 32768, want: "throughput"},
	}
	for _, tt := range tests {
		if got := resolveRelayDBProfile("auto", tt.hostMB).Name; got != tt.want {
			t.Fatalf("auto profile for %d MB: got %q, want %q", tt.hostMB, got, tt.want)
		}
	}
}

func TestRelayDBPostgresCommandMatchesResourceProfile(t *testing.T) {
	profile := resolveRelayDBProfile("balanced", 0)
	command := strings.Join(relayDBPostgresCommand(profile), " ")
	for _, want := range []string{
		"shared_buffers=512MB",
		"effective_cache_size=1536MB",
		"max_connections=100",
		"wal_compression=on",
		"jit=off",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("expected command to contain %q: %s", want, command)
		}
	}
}

func TestRelayDBEnsureBuildsPooledStableEncryptedDatabase(t *testing.T) {
	db := newRelayDBTestDB(t)
	key := bytes.Repeat([]byte{0x42}, 32)
	manager := newRelayDBManager(db, key, func() int { return 2048 })

	runtime := newRelayDBTestRuntime()
	target := relayDBTarget{
		App: "chat", Env: "prod", Branch: "main", Name: "db",
		Network: "relay-chat-prod-main", PrimaryName: "relay__chat__prod__main__svc__db",
		VolumeName: "relay__chat__prod__main__svc__db_data",
	}
	svc := ServiceConfig{Name: "db", Type: serviceTypeRelayDB, Profile: "starter"}

	first, err := manager.Ensure(runtime, target, svc, false, func(string, ...any) {})
	if err != nil {
		t.Fatalf("ensure RelayDB: %v", err)
	}
	if first.EnvKey != "DATABASE_URL" {
		t.Fatalf("env key: got %q", first.EnvKey)
	}
	if len(runtime.specs) != 2 {
		t.Fatalf("expected primary and pooler specs, got %d", len(runtime.specs))
	}
	primary, pooler := runtime.specs[0], runtime.specs[1]
	if primary.Image != "postgres:17" {
		t.Fatalf("primary image: got %q", primary.Image)
	}
	if primary.MemLimit != "512m" || primary.CPULimit != "1" {
		t.Fatalf("starter resource limits: memory=%q cpu=%q", primary.MemLimit, primary.CPULimit)
	}
	if pooler.Image != defaultPgBouncerImage {
		t.Fatalf("pooler image: got %q", pooler.Image)
	}
	if !envContains(pooler.Env, "POOL_MODE=transaction") || !envContains(pooler.Env, "MAX_CLIENT_CONN=500") || !envContains(pooler.Env, "MAX_PREPARED_STATEMENTS=100") {
		t.Fatalf("pooler environment missing transaction profile: %#v", pooler.Env)
	}
	parsed, err := url.Parse(first.URL)
	if err != nil {
		t.Fatalf("parse app URL: %v", err)
	}
	if parsed.Hostname() != relayDBPoolContainerName(target.PrimaryName) {
		t.Fatalf("app URL bypassed pooler: %q", parsed.Hostname())
	}
	password, ok := parsed.User.Password()
	if !ok || len(password) < 24 {
		t.Fatalf("expected generated password in app URL")
	}

	var stored string
	if err := db.QueryRow(`SELECT password FROM service_credentials WHERE project='chat' AND env='prod' AND branch='main' AND name='db'`).Scan(&stored); err != nil {
		t.Fatalf("load stored credential: %v", err)
	}
	if !strings.HasPrefix(stored, "enc:") || strings.Contains(stored, password) {
		t.Fatalf("credential was not encrypted at rest")
	}

	second, err := manager.Ensure(runtime, target, svc, false, func(string, ...any) {})
	if err != nil {
		t.Fatalf("ensure existing RelayDB: %v", err)
	}
	if second.URL != first.URL {
		t.Fatalf("credential changed across ensure calls")
	}
	if len(runtime.specs) != 2 {
		t.Fatalf("healthy RelayDB was recreated")
	}
}

func TestRelayDBCredentialCannotBeReadWithWrongKey(t *testing.T) {
	db := newRelayDBTestDB(t)
	target := relayDBTarget{App: "video", Env: "prod", Branch: "main", Name: "db"}
	first := newRelayDBManager(db, bytes.Repeat([]byte{1}, 32), nil)
	if _, _, _, err := first.loadOrCreateCredential(target); err != nil {
		t.Fatalf("create credential: %v", err)
	}
	second := newRelayDBManager(db, bytes.Repeat([]byte{2}, 32), nil)
	if _, _, _, err := second.loadOrCreateCredential(target); err == nil {
		t.Fatal("expected wrong encryption key to fail")
	}
}

func TestStartProjectServiceDoesNotPersistRelayDBURL(t *testing.T) {
	db := newRelayDBTestDB(t)
	runtime := newRelayDBTestRuntime()
	manager := newRelayDBManager(db, bytes.Repeat([]byte{3}, 32), func() int { return 2048 })
	server := &Server{db: db, runtime: runtime, relayDB: manager}

	envKey, envValue, err := server.startProjectService(
		func(string, ...any) {}, "chat", "prod", "main",
		ServiceConfig{Name: "db", Type: serviceTypeRelayDB}, "relay-chat-prod-main", false,
	)
	if err != nil {
		t.Fatalf("start RelayDB companion: %v", err)
	}
	if envKey != "DATABASE_URL" || !strings.Contains(envValue, "__pool") {
		t.Fatalf("unexpected app connection: %s=%s", envKey, envValue)
	}
	state, err := server.getProjectService("chat", "prod", "main", "db")
	if err != nil {
		t.Fatalf("load RelayDB state: %v", err)
	}
	if state.EnvVal != "" {
		t.Fatal("RelayDB URL with credentials was persisted in project service state")
	}
}

func newRelayDBTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "relaydb-test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateDB(db); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}

type relayDBTestRuntime struct {
	*mockRuntime
	specs []ContainerSpec
}

func newRelayDBTestRuntime() *relayDBTestRuntime {
	return &relayDBTestRuntime{
		mockRuntime: &mockRuntime{
			running:   map[string]bool{},
			exists:    map[string]bool{},
			published: map[string]int{},
		},
	}
}

func (r *relayDBTestRuntime) RunDetached(spec ContainerSpec) error {
	r.specs = append(r.specs, spec)
	r.running[spec.Name] = true
	r.exists[spec.Name] = true
	return nil
}

func (r *relayDBTestRuntime) Remove(name string) {
	delete(r.running, name)
	delete(r.exists, name)
}

func envContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
