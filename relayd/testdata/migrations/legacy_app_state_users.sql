CREATE TABLE app_state (
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
);

INSERT INTO app_state (app, env, branch, public_host)
VALUES ('demo', 'prod', 'main', 'demo.example.com');

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'deployer'
);

INSERT INTO users (id, username, password_hash, role)
VALUES ('user-1', 'owner', 'hash', 'owner');
