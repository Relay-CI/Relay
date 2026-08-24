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
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN webhook_secret TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN notification_webhooks TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN traffic_split_percent INTEGER DEFAULT 100`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN rollout_min_requests INTEGER DEFAULT 25`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN rollout_error_percent REAL DEFAULT 5`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN rollout_assess_seconds INTEGER DEFAULT 300`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN rollout_started_at INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN rollout_deploy_id TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN rollout_status TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN stopped INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN host_port_explicit INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN project_root TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN build_context TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN dockerfile TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN cpu_limit TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN mem_limit TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN resource_mode TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN public_hosts TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN buildpack_kind TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN image TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN port INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN host_port INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE project_services ADD COLUMN spec_hash TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN repo_hash TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN volumes TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE app_state ADD COLUMN git_token TEXT DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN created_at INTEGER`)
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
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS user_permissions (
		user_id TEXT NOT NULL,
		app TEXT NOT NULL,
		env TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		PRIMARY KEY (user_id, app, env)
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
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_analytics_ip ON analytics_events(remote_ip)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_analytics_cc ON analytics_events(country_code, remote_ip)`)
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
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS cloud_agent_tokens (
		token TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL DEFAULT 'default',
		server_name TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL DEFAULT 0
	)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS connected_servers (
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
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_connected_servers_status_heartbeat
		ON connected_servers(status, last_heartbeat_at DESC)`)
	_, _ = db.Exec(`ALTER TABLE connected_servers ADD COLUMN ws_session_id TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE connected_servers ADD COLUMN agent_version TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE connected_servers ADD COLUMN last_disconnect_at INTEGER NOT NULL DEFAULT 0`)
	if err := seedLanePolicies(db); err != nil {
		return err
	}
	return nil
}
