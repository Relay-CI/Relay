package main

import (
	"database/sql"
	"fmt"
	"time"
)

type SQLiteStore struct {
	Primary   *sql.DB
	Analytics *sql.DB
}

// sqliteDSN builds a DSN that applies pragmas on every pooled connection
// (Exec-based PRAGMAs only reach one connection of the pool).
func sqliteDSN(path string) string {
	return "file:" + path +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=cache_size(-4096)"
}

func openSQLiteStore(path string) (*SQLiteStore, error) {
	primary, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open primary SQLite store: %w", err)
	}
	_, _ = primary.Exec("PRAGMA journal_mode=WAL;")
	primary.SetMaxOpenConns(12)
	primary.SetMaxIdleConns(6)
	primary.SetConnMaxLifetime(30 * time.Minute)
	if err := migrateDB(primary); err != nil {
		_ = primary.Close()
		return nil, fmt.Errorf("migrate SQLite store: %w", err)
	}

	analytics, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		_ = primary.Close()
		return nil, fmt.Errorf("open analytics SQLite store: %w", err)
	}
	_, _ = analytics.Exec("PRAGMA journal_mode=WAL;")
	analytics.SetMaxOpenConns(3)
	analytics.SetMaxIdleConns(2)
	analytics.SetConnMaxLifetime(30 * time.Minute)

	return &SQLiteStore{Primary: primary, Analytics: analytics}, nil
}

func (store *SQLiteStore) Close() error {
	if store == nil {
		return nil
	}
	if store.Analytics != nil {
		_ = store.Analytics.Close()
	}
	if store.Primary != nil {
		return store.Primary.Close()
	}
	return nil
}

func createCoreTables(tx *sql.Tx) error {
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
			project_root TEXT DEFAULT '',
			build_context TEXT DEFAULT '',
			dockerfile TEXT DEFAULT '',
			engine TEXT DEFAULT 'docker',
			current_image TEXT,
			previous_image TEXT,
			mode TEXT,
			host_port INTEGER,
			host_port_explicit INTEGER DEFAULT 0,
			service_port INTEGER,
			public_host TEXT,
			public_hosts TEXT DEFAULT '',
			active_slot TEXT,
			standby_slot TEXT,
			drain_until INTEGER,
			traffic_mode TEXT,
			access_policy TEXT,
			ip_allowlist TEXT,
			repo_hash TEXT,
			expires_at INTEGER DEFAULT 0,
			webhook_secret TEXT DEFAULT '',
			notification_webhooks TEXT DEFAULT '',
			traffic_split_percent INTEGER DEFAULT 100,
			rollout_min_requests INTEGER DEFAULT 25,
			rollout_error_percent REAL DEFAULT 5,
			rollout_assess_seconds INTEGER DEFAULT 300,
			rollout_started_at INTEGER DEFAULT 0,
			rollout_deploy_id TEXT DEFAULT '',
			rollout_status TEXT DEFAULT '',
			stopped INTEGER DEFAULT 0,
			cpu_limit TEXT DEFAULT '',
			mem_limit TEXT DEFAULT '',
			resource_mode TEXT DEFAULT '',
			volumes TEXT DEFAULT '',
			buildpack_kind TEXT DEFAULT '',
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
		`CREATE TABLE IF NOT EXISTS service_credentials (
			project TEXT NOT NULL,
			env TEXT NOT NULL,
			branch TEXT NOT NULL,
			name TEXT NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			database_name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			PRIMARY KEY (project, env, branch, name)
		);`,
	}
	for _, st := range stmts {
		if _, err := tx.Exec(st); err != nil {
			return fmt.Errorf("create core table: %w", err)
		}
	}
	return nil
}
