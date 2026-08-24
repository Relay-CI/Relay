package main

import (
	"database/sql"
	"fmt"
	"time"
)

type schemaMigration struct {
	version int
	name    string
	up      func(*sql.Tx) error
}

var sqliteSchemaMigrations = []schemaMigration{
	{version: 1, name: "core tables", up: createCoreTables},
	{version: 2, name: "lane and deploy columns", up: migrateLaneAndDeployColumns},
	{version: 3, name: "authentication tables", up: migrateAuthenticationTables},
	{version: 4, name: "analytics audit and promotions", up: migrateOperationsTables},
	{version: 5, name: "connected servers", up: migrateConnectedServerTables},
	{version: 6, name: "lane policies", up: migrateLanePolicyTables},
	{version: 7, name: "github delivery workflow", up: migrateGitHubWorkflowTables},
	{version: 8, name: "github app installations", up: migrateGitHubAppTables},
}

func migrateDB(db *sql.DB) error {
	if err := runSQLiteMigrations(db, sqliteSchemaMigrations); err != nil {
		return err
	}
	if err := seedLanePolicies(db); err != nil {
		return fmt.Errorf("seed lane policies: %w", err)
	}
	return nil
}

func runSQLiteMigrations(db *sql.DB, migrations []schemaMigration) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema migration ledger: %w", err)
	}

	applied, err := appliedSQLiteMigrations(db)
	if err != nil {
		return err
	}
	knownVersions := make(map[int]struct{}, len(migrations))
	previousVersion := 0
	for _, migration := range migrations {
		if migration.version <= previousVersion {
			return fmt.Errorf("SQLite migrations must be ordered with unique positive versions: %d follows %d", migration.version, previousVersion)
		}
		knownVersions[migration.version] = struct{}{}
		previousVersion = migration.version
	}
	for version := range applied {
		if _, ok := knownVersions[version]; !ok {
			return fmt.Errorf("SQLite schema version %d is newer than or unknown to this relayd build", version)
		}
	}
	for _, migration := range migrations {
		if _, ok := applied[migration.version]; ok {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin SQLite migration %d (%s): %w", migration.version, migration.name, err)
		}
		if err := migration.up(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply SQLite migration %d (%s): %w", migration.version, migration.name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			migration.version,
			migration.name,
			time.Now().UTC().UnixMilli(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record SQLite migration %d (%s): %w", migration.version, migration.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit SQLite migration %d (%s): %w", migration.version, migration.name, err)
		}
	}
	return nil
}

func appliedSQLiteMigrations(db *sql.DB) (map[int]string, error) {
	rows, err := db.Query(`SELECT version, name FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read SQLite migration ledger: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]string)
	for rows.Next() {
		var version int
		var name string
		if err := rows.Scan(&version, &name); err != nil {
			return nil, fmt.Errorf("scan SQLite migration ledger: %w", err)
		}
		applied[version] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SQLite migration ledger: %w", err)
	}
	return applied, nil
}

type sqliteColumn struct {
	table      string
	name       string
	definition string
}

func migrateLaneAndDeployColumns(tx *sql.Tx) error {
	columns := []sqliteColumn{
		{table: "deploys", name: "preview_url", definition: "TEXT DEFAULT ''"},
		{table: "deploys", name: "build_number", definition: "INTEGER DEFAULT 0"},
		{table: "deploys", name: "deployed_by", definition: "TEXT DEFAULT ''"},
		{table: "deploys", name: "commit_message", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "webhook_secret", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "engine", definition: "TEXT DEFAULT 'docker'"},
		{table: "app_state", name: "active_slot", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "standby_slot", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "drain_until", definition: "INTEGER DEFAULT 0"},
		{table: "app_state", name: "traffic_mode", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "access_policy", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "ip_allowlist", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "expires_at", definition: "INTEGER DEFAULT 0"},
		{table: "app_state", name: "notification_webhooks", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "traffic_split_percent", definition: "INTEGER DEFAULT 100"},
		{table: "app_state", name: "rollout_min_requests", definition: "INTEGER DEFAULT 25"},
		{table: "app_state", name: "rollout_error_percent", definition: "REAL DEFAULT 5"},
		{table: "app_state", name: "rollout_assess_seconds", definition: "INTEGER DEFAULT 300"},
		{table: "app_state", name: "rollout_started_at", definition: "INTEGER DEFAULT 0"},
		{table: "app_state", name: "rollout_deploy_id", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "rollout_status", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "stopped", definition: "INTEGER DEFAULT 0"},
		{table: "app_state", name: "host_port_explicit", definition: "INTEGER DEFAULT 0"},
		{table: "app_state", name: "project_root", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "build_context", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "dockerfile", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "cpu_limit", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "mem_limit", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "resource_mode", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "public_hosts", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "buildpack_kind", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "repo_hash", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "volumes", definition: "TEXT DEFAULT ''"},
		{table: "app_state", name: "git_token", definition: "TEXT DEFAULT ''"},
		{table: "project_services", name: "image", definition: "TEXT DEFAULT ''"},
		{table: "project_services", name: "port", definition: "INTEGER DEFAULT 0"},
		{table: "project_services", name: "host_port", definition: "INTEGER DEFAULT 0"},
		{table: "project_services", name: "spec_hash", definition: "TEXT DEFAULT ''"},
	}
	for _, column := range columns {
		if err := ensureSQLiteColumn(tx, column); err != nil {
			return err
		}
	}
	return nil
}

func ensureSQLiteColumn(tx *sql.Tx, column sqliteColumn) error {
	rows, err := tx.Query(`PRAGMA table_info(` + column.table + `)`)
	if err != nil {
		return fmt.Errorf("inspect %s.%s: %w", column.table, column.name, err)
	}
	foundTable := false
	foundColumn := false
	for rows.Next() {
		foundTable = true
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan %s columns: %w", column.table, err)
		}
		if name == column.name {
			foundColumn = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate %s columns: %w", column.table, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close %s column inspection: %w", column.table, err)
	}
	if !foundTable {
		return fmt.Errorf("table %s does not exist", column.table)
	}
	if foundColumn {
		return nil
	}
	if _, err := tx.Exec(`ALTER TABLE ` + column.table + ` ADD COLUMN ` + column.name + ` ` + column.definition); err != nil {
		return fmt.Errorf("add %s.%s: %w", column.table, column.name, err)
	}
	return nil
}

func migrateAuthenticationTables(tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS server_config (key TEXT PRIMARY KEY, value TEXT)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'deployer',
			created_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			created_at INTEGER,
			expires_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS auth_codes (
			code TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			created_at INTEGER,
			expires_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS user_permissions (
			user_id TEXT NOT NULL,
			app TEXT NOT NULL,
			env TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'viewer',
			PRIMARY KEY (user_id, app, env)
		)`,
	}
	if err := execSQLiteStatements(tx, statements); err != nil {
		return err
	}
	return ensureSQLiteColumn(tx, sqliteColumn{table: "users", name: "created_at", definition: "INTEGER"})
}

func migrateOperationsTables(tx *sql.Tx) error {
	return execSQLiteStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS analytics_events (
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
		)`,
		`CREATE INDEX IF NOT EXISTS idx_analytics_ts ON analytics_events(ts)`,
		`CREATE INDEX IF NOT EXISTS idx_analytics_ip ON analytics_events(remote_ip)`,
		`CREATE INDEX IF NOT EXISTS idx_analytics_cc ON analytics_events(country_code, remote_ip)`,
		`CREATE TABLE IF NOT EXISTS ip_country_cache (
			ip TEXT PRIMARY KEY,
			country_code TEXT NOT NULL DEFAULT '',
			country_name TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ts INTEGER NOT NULL,
			actor TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL DEFAULT '',
			target TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(ts)`,
		`CREATE TABLE IF NOT EXISTS promotions (
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
		)`,
		`CREATE INDEX IF NOT EXISTS idx_promotions_app_ts ON promotions(app, requested_at DESC)`,
	})
}

func migrateConnectedServerTables(tx *sql.Tx) error {
	if err := execSQLiteStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS cloud_agent_tokens (
			token TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL DEFAULT 'default',
			server_name TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS connected_servers (
			id TEXT PRIMARY KEY,
			token TEXT NOT NULL UNIQUE,
			workspace_id TEXT NOT NULL DEFAULT 'default',
			server_name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'offline',
			ws_session_id TEXT NOT NULL DEFAULT '',
			agent_version TEXT NOT NULL DEFAULT '',
			last_heartbeat_at INTEGER NOT NULL DEFAULT 0,
			last_disconnect_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
	}); err != nil {
		return err
	}
	for _, column := range []sqliteColumn{
		{table: "connected_servers", name: "ws_session_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "connected_servers", name: "agent_version", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "connected_servers", name: "last_disconnect_at", definition: "INTEGER NOT NULL DEFAULT 0"},
	} {
		if err := ensureSQLiteColumn(tx, column); err != nil {
			return err
		}
	}
	return execSQLiteStatements(tx, []string{
		`CREATE INDEX IF NOT EXISTS idx_connected_servers_status_heartbeat
			ON connected_servers(status, last_heartbeat_at DESC)`,
	})
}

func migrateLanePolicyTables(tx *sql.Tx) error {
	return execSQLiteStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS lane_policies (
			env TEXT PRIMARY KEY,
			display_name TEXT DEFAULT '',
			default_mode TEXT DEFAULT '',
			default_traffic_mode TEXT DEFAULT '',
			default_access_policy TEXT DEFAULT '',
			default_host_port INTEGER DEFAULT 0,
			auto_subdomain INTEGER DEFAULT 0,
			random_subdomain INTEGER DEFAULT 0,
			retention_hours INTEGER DEFAULT 0,
			promote_to TEXT DEFAULT ''
		)`,
	})
}

func migrateGitHubWorkflowTables(tx *sql.Tx) error {
	return execSQLiteStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS github_connections (
			id TEXT PRIMARY KEY,
			login TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			html_url TEXT NOT NULL DEFAULT '',
			token_enc TEXT NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS github_projects (
			app TEXT PRIMARY KEY,
			connection_id TEXT NOT NULL DEFAULT 'default',
			repo_full_name TEXT NOT NULL UNIQUE,
			clone_url TEXT NOT NULL,
			html_url TEXT NOT NULL DEFAULT '',
			production_branch TEXT NOT NULL DEFAULT 'main',
			preview_enabled INTEGER NOT NULL DEFAULT 1,
			production_enabled INTEGER NOT NULL DEFAULT 1,
			webhook_id INTEGER NOT NULL DEFAULT 0,
			webhook_secret_enc TEXT NOT NULL,
			status_context TEXT NOT NULL DEFAULT 'relay/preview',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_github_projects_repo ON github_projects(repo_full_name)`,
		`CREATE TABLE IF NOT EXISTS github_deliveries (
			delivery_id TEXT PRIMARY KEY,
			event TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL DEFAULT '',
			repo_full_name TEXT NOT NULL DEFAULT '',
			outcome TEXT NOT NULL DEFAULT 'received',
			received_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_github_deliveries_received ON github_deliveries(received_at DESC)`,
		`CREATE TABLE IF NOT EXISTS github_previews (
			repo_full_name TEXT NOT NULL,
			pr_number INTEGER NOT NULL,
			app TEXT NOT NULL,
			branch TEXT NOT NULL,
			status_repo_full_name TEXT NOT NULL DEFAULT '',
			head_sha TEXT NOT NULL DEFAULT '',
			deploy_id TEXT NOT NULL DEFAULT '',
			preview_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'queued',
			updated_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (repo_full_name, branch)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_github_previews_app_updated ON github_previews(app, updated_at DESC)`,
	})
}

func migrateGitHubAppTables(tx *sql.Tx) error {
	for _, column := range []sqliteColumn{
		{table: "github_projects", name: "auth_mode", definition: "TEXT NOT NULL DEFAULT 'token'"},
		{table: "github_projects", name: "installation_id", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "github_projects", name: "repository_id", definition: "INTEGER NOT NULL DEFAULT 0"},
	} {
		if err := ensureSQLiteColumn(tx, column); err != nil {
			return err
		}
	}
	return execSQLiteStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS github_app_config (
			id TEXT PRIMARY KEY,
			app_id INTEGER NOT NULL,
			client_id TEXT NOT NULL DEFAULT '',
			app_slug TEXT NOT NULL DEFAULT '',
			app_name TEXT NOT NULL DEFAULT '',
			owner_login TEXT NOT NULL DEFAULT '',
			private_key_enc TEXT NOT NULL,
			webhook_secret_enc TEXT NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS github_installations (
			installation_id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL DEFAULT 0,
			account_login TEXT NOT NULL DEFAULT '',
			account_type TEXT NOT NULL DEFAULT '',
			repository_selection TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			suspended_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS github_installation_repositories (
			installation_id INTEGER NOT NULL,
			repository_id INTEGER NOT NULL,
			full_name TEXT NOT NULL,
			clone_url TEXT NOT NULL DEFAULT '',
			html_url TEXT NOT NULL DEFAULT '',
			default_branch TEXT NOT NULL DEFAULT 'main',
			private INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1,
			updated_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (installation_id, repository_id)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_github_installation_repositories_name
			ON github_installation_repositories(installation_id, full_name)`,
		`CREATE TABLE IF NOT EXISTS github_check_runs (
			deploy_id TEXT PRIMARY KEY,
			installation_id INTEGER NOT NULL,
			repository_id INTEGER NOT NULL,
			check_run_id INTEGER NOT NULL,
			head_sha TEXT NOT NULL DEFAULT '',
			check_name TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS github_app_states (
			state_hash TEXT PRIMARY KEY,
			purpose TEXT NOT NULL,
			actor TEXT NOT NULL DEFAULT '',
			expires_at INTEGER NOT NULL,
			used_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_github_app_states_expiry ON github_app_states(expires_at)`,
	})
}

func execSQLiteStatements(tx *sql.Tx, statements []string) error {
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("execute schema statement: %w", err)
		}
	}
	return nil
}
