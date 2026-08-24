package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func applyMigrationFixture(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	fixturePath := filepath.Join("testdata", "migrations", name)
	contents, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read migration fixture %s: %v", name, err)
	}
	if _, err := db.Exec(string(contents)); err != nil {
		t.Fatalf("apply migration fixture %s: %v", name, err)
	}
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

func fixtureMigrationVersions(t *testing.T, db *sql.DB) []int {
	t.Helper()
	rows, err := db.Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("read migration versions: %v", err)
	}
	defer rows.Close()
	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			t.Fatalf("scan migration version: %v", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migration versions: %v", err)
	}
	return versions
}

func TestMigrateDBCreatesFreshSchemaAndIsIdempotent(t *testing.T) {
	db := openMigrationFixture(t)
	if err := migrateDB(db); err != nil {
		t.Fatalf("fresh migration: %v", err)
	}
	if err := migrateDB(db); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	wantVersions := []int{1, 2, 3, 4, 5, 6, 7, 8}
	if got := fixtureMigrationVersions(t, db); !reflect.DeepEqual(got, wantVersions) {
		t.Fatalf("migration ledger = %v, want %v", got, wantVersions)
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
		"github_connections",
		"github_projects",
		"github_deliveries",
		"github_previews",
		"github_app_config",
		"github_installations",
		"github_installation_repositories",
		"github_check_runs",
		"github_app_states",
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

	githubProjectColumns := fixtureColumns(t, db, "github_projects")
	for _, column := range []string{"auth_mode", "installation_id", "repository_id"} {
		if !githubProjectColumns[column] {
			t.Errorf("fresh github_projects schema is missing column %s", column)
		}
	}
}

func TestMigrateDBCompletesPartiallyUpgradedConnectedServersFixture(t *testing.T) {
	db := openMigrationFixture(t)
	applyMigrationFixture(t, db, "partial_connected_servers.sql")

	if err := migrateDB(db); err != nil {
		t.Fatalf("upgrade partial fixture: %v", err)
	}
	columns := fixtureColumns(t, db, "connected_servers")
	for _, column := range []string{"ws_session_id", "agent_version", "last_disconnect_at"} {
		if !columns[column] {
			t.Errorf("partial connected_servers fixture was not upgraded with %s", column)
		}
	}
	var serverName string
	if err := db.QueryRow(`SELECT server_name FROM connected_servers WHERE id='server-1'`).Scan(&serverName); err != nil {
		t.Fatalf("read partial fixture row after migration: %v", err)
	}
	if serverName != "primary" {
		t.Fatalf("partial fixture row changed: server_name=%q", serverName)
	}
}

func TestMigrateGitHubAppPreservesVersionSevenTokenProjects(t *testing.T) {
	db := openMigrationFixture(t)
	applyMigrationFixture(t, db, "github_token_v7.sql")

	if err := migrateDB(db); err != nil {
		t.Fatalf("upgrade GitHub token fixture: %v", err)
	}
	var repo, authMode, secret string
	var installationID, repositoryID int64
	if err := db.QueryRow(
		`SELECT repo_full_name, auth_mode, installation_id, repository_id, webhook_secret_enc
		 FROM github_projects WHERE app='widget'`,
	).Scan(&repo, &authMode, &installationID, &repositoryID, &secret); err != nil {
		t.Fatalf("read migrated GitHub token project: %v", err)
	}
	if repo != "acme/widget" || authMode != "token" || installationID != 0 || repositoryID != 0 || secret != "enc:legacy-webhook-secret" {
		t.Fatalf("version seven project changed during App expansion: repo=%q mode=%q installation=%d repository=%d secret=%q", repo, authMode, installationID, repositoryID, secret)
	}
	if got := fixtureMigrationVersions(t, db); !reflect.DeepEqual(got, []int{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatalf("migration ledger = %v", got)
	}
}

func TestSQLiteMigrationFailureRollsBackSchemaAndLedger(t *testing.T) {
	db := openMigrationFixture(t)
	migrations := []schemaMigration{{
		version: 1,
		name:    "deliberate failure",
		up: func(tx *sql.Tx) error {
			if _, err := tx.Exec(`CREATE TABLE should_roll_back (id INTEGER PRIMARY KEY)`); err != nil {
				return err
			}
			return errors.New("stop migration")
		},
	}}

	if err := runSQLiteMigrations(db, migrations); err == nil {
		t.Fatal("failed migration unexpectedly succeeded")
	}
	if fixtureHasTable(t, db, "should_roll_back") {
		t.Fatal("failed migration left its schema change behind")
	}
	if got := fixtureMigrationVersions(t, db); len(got) != 0 {
		t.Fatalf("failed migration was recorded in ledger: %v", got)
	}
}

func TestSQLiteMigrationRejectsUnknownSchemaVersion(t *testing.T) {
	db := openMigrationFixture(t)
	if _, err := db.Exec(`CREATE TABLE schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at INTEGER NOT NULL
	)`); err != nil {
		t.Fatalf("create migration ledger: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (999, 'future', 0)`); err != nil {
		t.Fatalf("seed future migration: %v", err)
	}

	if err := runSQLiteMigrations(db, sqliteSchemaMigrations); err == nil {
		t.Fatal("unknown schema version was accepted")
	}
}

func TestMigrateDBUpgradesLegacyFixtureWithoutLosingRows(t *testing.T) {
	db := openMigrationFixture(t)
	applyMigrationFixture(t, db, "legacy_app_state_users.sql")

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
