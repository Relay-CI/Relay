CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at INTEGER NOT NULL
);

INSERT INTO schema_migrations(version, name, applied_at) VALUES
    (1, 'core tables', 0),
    (2, 'lane and deploy columns', 0),
    (3, 'authentication tables', 0),
    (4, 'analytics audit and promotions', 0),
    (5, 'connected servers', 0),
    (6, 'lane policies', 0),
    (7, 'github delivery workflow', 0);

CREATE TABLE lane_policies (
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
);

CREATE TABLE github_projects (
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
);

INSERT INTO github_projects (
    app, connection_id, repo_full_name, clone_url, html_url,
    production_branch, preview_enabled, production_enabled,
    webhook_id, webhook_secret_enc, status_context, created_at, updated_at
) VALUES (
    'widget', 'default', 'acme/widget', 'https://github.com/acme/widget.git',
    'https://github.com/acme/widget', 'main', 1, 1, 42,
    'enc:legacy-webhook-secret', 'relay/preview', 100, 100
);
