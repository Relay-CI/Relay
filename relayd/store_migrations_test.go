package main

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func openMigrationFixture(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "fixture.db"))
	if err != nil {
		t.Fatalf("open migration fixture: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func fixtureHasTable(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
		t.Fatalf("inspect table %s: %v", table, err)
	}
	return count == 1
}

func fixtureColumns(t *testing.T, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("inspect columns for %s: %v", table, err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan column for %s: %v", table, err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns for %s: %v", table, err)
	}
	return columns
}

func TestMigrateDBCreatesFreshSchemaAndIsIdempotent(t *testing.T) {
	db := openMigrationFixture(t)
	if err := migrateDB(db); err != nil {
		t.Fatalf("fresh migration: %v", err)
	}
	if err := migrateDB(db); err != nil {
		t.Fatalf("second migration: %v", err)
	}

	for _, table := range []string{
		"deploys",
		"deploy_requests",
		"app_state",
		"sync_sessions",
		"project_services",
		"project_service_specs",
		"service_credentials",
		"users",
		"analytics_events",
		"promotions",
		"connected_servers",
	} {
		if !fixtureHasTable(t, db, table) {
			t.Errorf("fresh schema is missing table %s", table)
		}
	}

	appStateColumns := fixtureColumns(t, db, "app_state")
	for _, column := range []string{
		"public_hosts",
		"host_port_explicit",
		"volumes",
		"buildpack_kind",
		"git_token",
		"rollout_status",
		"cpu_limit",
		"mem_limit",
		"resource_mode",
	} {
		if !appStateColumns[column] {
			t.Errorf("fresh app_state schema is missing column %s", column)
		}
	}
}

func TestMigrateDBUpgradesLegacyFixtureWithoutLosingRows(t *testing.T) {
	db := openMigrationFixture(t)
	legacy := []string{
		`CREATE TABLE app_state (
			app TEXT,
			env TEXT,
			branch TEXT,
			repo_url TEXT,
			current_image TEXT,
			previous_image TEXT,
			mode TEXT,
			host_port INTEGER,
			service_port INTEGER,
			public_host TEXT,
			updated_at INTEGER,
			PRIMARY KEY (app, env, branch)
		)`,
		`INSERT INTO app_state (app, env, branch, public_host) VALUES ('demo', 'prod', 'main', 'demo.example.com')`,
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'deployer'
		)`,
		`INSERT INTO users (id, username, password_hash, role) VALUES ('user-1', 'owner', 'hash', 'owner')`,
	}
	for _, statement := range legacy {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed legacy fixture: %v", err)
		}
	}

	if err := migrateDB(db); err != nil {
		t.Fatalf("upgrade legacy fixture: %v", err)
	}

	appStateColumns := fixtureColumns(t, db, "app_state")
	for _, column := range []string{"public_hosts", "volumes", "buildpack_kind", "git_token", "rollout_status"} {
		if !appStateColumns[column] {
			t.Errorf("legacy app_state was not upgraded with %s", column)
		}
	}
	if !fixtureColumns(t, db, "users")["created_at"] {
		t.Error("legacy users table was not upgraded with created_at")
	}

	var publicHost string
	if err := db.QueryRow(`SELECT public_host FROM app_state WHERE app='demo' AND env='prod' AND branch='main'`).Scan(&publicHost); err != nil {
		t.Fatalf("read legacy app row after migration: %v", err)
	}
	if publicHost != "demo.example.com" {
		t.Fatalf("legacy app row changed: public_host=%q", publicHost)
	}
	var username string
	if err := db.QueryRow(`SELECT username FROM users WHERE id='user-1'`).Scan(&username); err != nil {
		t.Fatalf("read legacy user after migration: %v", err)
	}
	if username != "owner" {
		t.Fatalf("legacy user row changed: username=%q", username)
	}
}

func TestOpenSQLiteStoreMigratesBeforeOpeningAnalyticsPool(t *testing.T) {
	store, err := openSQLiteStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if !fixtureHasTable(t, store.Primary, "app_state") {
		t.Fatal("primary pool did not receive migrations")
	}
	if !fixtureHasTable(t, store.Analytics, "analytics_events") {
		t.Fatal("analytics pool opened before migrations were visible")
	}
}
