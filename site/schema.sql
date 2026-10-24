-- Relay Cloud Connect
-- PostgreSQL schema for the hosted control plane used by /site/cloud.
-- Apply with: psql "$DATABASE_URL" -f schema.sql

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS accounts (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email          TEXT NOT NULL UNIQUE,
  full_name      TEXT NOT NULL,
  auth_provider  TEXT NOT NULL DEFAULT 'email',
  github_login   TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workspaces (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_account_id    UUID REFERENCES accounts (id) ON DELETE SET NULL,
  name                TEXT NOT NULL,
  slug                TEXT NOT NULL UNIQUE,
  plan                TEXT NOT NULL DEFAULT 'starter',
  region              TEXT NOT NULL DEFAULT 'us-east-1',
  websocket_endpoint  TEXT NOT NULL DEFAULT 'wss://relay.com/connect',
  log_retention_days  INTEGER NOT NULL DEFAULT 7,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workspace_memberships (
  workspace_id     UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  account_id       UUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  role             TEXT NOT NULL,
  invited_by       UUID REFERENCES accounts (id) ON DELETE SET NULL,
  joined_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_active_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (workspace_id, account_id)
);

CREATE TABLE IF NOT EXISTS workspace_entitlements (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id  UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  feature_key   TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'enabled',
  quota         INTEGER,
  notes         TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, feature_key)
);

CREATE TABLE IF NOT EXISTS sso_connections (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id   UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  provider       TEXT NOT NULL,
  domain         TEXT NOT NULL,
  status         TEXT NOT NULL DEFAULT 'pending',
  enforced       BOOLEAN NOT NULL DEFAULT FALSE,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_connections_unique
  ON sso_connections (workspace_id, provider, domain);

CREATE TABLE IF NOT EXISTS servers (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id          UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  name                  TEXT NOT NULL,
  region                TEXT NOT NULL,
  environment           TEXT NOT NULL DEFAULT 'production',
  connected             BOOLEAN NOT NULL DEFAULT FALSE,
  connection_mode       TEXT NOT NULL DEFAULT 'websocket',
  websocket_session_id  TEXT,
  agent_version         TEXT NOT NULL DEFAULT 'relayd 0.0.0',
  cpu_percent           NUMERIC(5, 2) NOT NULL DEFAULT 0,
  memory_percent        NUMERIC(5, 2) NOT NULL DEFAULT 0,
  active_alerts         INTEGER NOT NULL DEFAULT 0,
  last_seen_at          TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, name)
);

CREATE TABLE IF NOT EXISTS apps (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id       UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  server_id          UUID REFERENCES servers (id) ON DELETE SET NULL,
  name               TEXT NOT NULL,
  runtime            TEXT NOT NULL,
  repository         TEXT,
  environment        TEXT NOT NULL DEFAULT 'production',
  status             TEXT NOT NULL DEFAULT 'healthy',
  release_channel    TEXT NOT NULL DEFAULT 'stable',
  desired_instances  INTEGER NOT NULL DEFAULT 1,
  live_instances     INTEGER NOT NULL DEFAULT 0,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, name, environment)
);

CREATE TABLE IF NOT EXISTS deployments (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id      UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  app_id            UUID REFERENCES apps (id) ON DELETE SET NULL,
  server_id         UUID REFERENCES servers (id) ON DELETE SET NULL,
  version           TEXT NOT NULL,
  source_ref        TEXT,
  status            TEXT NOT NULL,
  trigger_kind      TEXT NOT NULL DEFAULT 'manual',
  triggered_by      TEXT NOT NULL,
  duration_seconds  INTEGER,
  started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at       TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS log_streams (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id         UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  server_id            UUID REFERENCES servers (id) ON DELETE SET NULL,
  app_id               UUID REFERENCES apps (id) ON DELETE SET NULL,
  retention_days       INTEGER NOT NULL DEFAULT 7,
  monthly_gb_ingested  NUMERIC(10, 2) NOT NULL DEFAULT 0,
  last_ingested_at     TIMESTAMPTZ,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metric_snapshots (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id          UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  server_id             UUID NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
  captured_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  cpu_percent           NUMERIC(5, 2) NOT NULL DEFAULT 0,
  memory_percent        NUMERIC(5, 2) NOT NULL DEFAULT 0,
  disk_percent          NUMERIC(5, 2) NOT NULL DEFAULT 0,
  websocket_latency_ms  INTEGER NOT NULL DEFAULT 0,
  requests_per_minute   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS remote_actions (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id   UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  server_id      UUID REFERENCES servers (id) ON DELETE SET NULL,
  app_id         UUID REFERENCES apps (id) ON DELETE SET NULL,
  action_kind    TEXT NOT NULL,
  status         TEXT NOT NULL DEFAULT 'queued',
  requested_by   TEXT NOT NULL,
  note           TEXT,
  requested_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at   TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS interest_leads (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email             TEXT NOT NULL UNIQUE,
  team_name         TEXT,
  fleet_size        INTEGER,
  needs             TEXT,
  status            TEXT NOT NULL DEFAULT 'new',
  priority          TEXT NOT NULL DEFAULT 'medium',
  owner_name        TEXT,
  review_notes      TEXT,
  customer_token    UUID NOT NULL DEFAULT gen_random_uuid(),
  last_contacted_at TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE interest_leads
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'new';

ALTER TABLE interest_leads
  ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'medium';

ALTER TABLE interest_leads
  ADD COLUMN IF NOT EXISTS owner_name TEXT;

ALTER TABLE interest_leads
  ADD COLUMN IF NOT EXISTS review_notes TEXT;

ALTER TABLE interest_leads
  ADD COLUMN IF NOT EXISTS customer_token UUID;

ALTER TABLE interest_leads
  ADD COLUMN IF NOT EXISTS last_contacted_at TIMESTAMPTZ;

UPDATE interest_leads
SET customer_token = gen_random_uuid()
WHERE customer_token IS NULL;

ALTER TABLE interest_leads
  ALTER COLUMN customer_token SET DEFAULT gen_random_uuid();

ALTER TABLE interest_leads
  ALTER COLUMN customer_token SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_workspace_memberships_workspace
  ON workspace_memberships (workspace_id, role);

CREATE INDEX IF NOT EXISTS idx_workspace_entitlements_workspace
  ON workspace_entitlements (workspace_id, feature_key);

CREATE INDEX IF NOT EXISTS idx_sso_connections_workspace
  ON sso_connections (workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_servers_workspace_connected
  ON servers (workspace_id, connected, last_seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_apps_workspace_environment
  ON apps (workspace_id, environment, status);

CREATE INDEX IF NOT EXISTS idx_deployments_workspace_started
  ON deployments (workspace_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_log_streams_workspace
  ON log_streams (workspace_id, last_ingested_at DESC);

CREATE INDEX IF NOT EXISTS idx_metric_snapshots_server_captured
  ON metric_snapshots (server_id, captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_remote_actions_workspace_requested
  ON remote_actions (workspace_id, requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_interest_leads_created
  ON interest_leads (created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_interest_leads_customer_token
  ON interest_leads (customer_token);

DROP TRIGGER IF EXISTS trg_accounts_updated_at ON accounts;
CREATE TRIGGER trg_accounts_updated_at
BEFORE UPDATE ON accounts
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS trg_workspaces_updated_at ON workspaces;
CREATE TRIGGER trg_workspaces_updated_at
BEFORE UPDATE ON workspaces
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS trg_workspace_entitlements_updated_at ON workspace_entitlements;
CREATE TRIGGER trg_workspace_entitlements_updated_at
BEFORE UPDATE ON workspace_entitlements
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS trg_sso_connections_updated_at ON sso_connections;
CREATE TRIGGER trg_sso_connections_updated_at
BEFORE UPDATE ON sso_connections
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS trg_servers_updated_at ON servers;
CREATE TRIGGER trg_servers_updated_at
BEFORE UPDATE ON servers
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS trg_apps_updated_at ON apps;
CREATE TRIGGER trg_apps_updated_at
BEFORE UPDATE ON apps
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS trg_interest_leads_updated_at ON interest_leads;
CREATE TRIGGER trg_interest_leads_updated_at
BEFORE UPDATE ON interest_leads
FOR EACH ROW
EXECUTE FUNCTION touch_updated_at();

INSERT INTO accounts (email, full_name, auth_provider, github_login)
VALUES
  ('ali@relay.dev', 'Ali Rahman', 'github', 'alir'),
  ('sara@relay.dev', 'Sara Dawson', 'email', NULL),
  ('nina@relay.dev', 'Nina Patel', 'google', NULL)
ON CONFLICT (email) DO NOTHING;

WITH owner AS (
  SELECT id
  FROM accounts
  WHERE email = 'ali@relay.dev'
)
INSERT INTO workspaces (owner_account_id, name, slug, plan, region, websocket_endpoint, log_retention_days)
SELECT
  owner.id,
  'Relay Cloud Connect',
  'relay-cloud-connect',
  'scale',
  'us-east-1',
  'wss://relay.com/connect',
  30
FROM owner
ON CONFLICT (slug) DO NOTHING;

INSERT INTO workspace_memberships (workspace_id, account_id, role)
SELECT w.id, a.id, membership.role
FROM workspaces w
JOIN (
  VALUES
    ('ali@relay.dev', 'owner'),
    ('sara@relay.dev', 'operator'),
    ('nina@relay.dev', 'observer')
) AS membership (email, role) ON TRUE
JOIN accounts a ON a.email = membership.email
WHERE w.slug = 'relay-cloud-connect'
ON CONFLICT (workspace_id, account_id) DO NOTHING;

INSERT INTO workspace_entitlements (workspace_id, feature_key, status, quota, notes)
SELECT
  w.id,
  item.feature_key,
  item.status,
  item.quota,
  item.notes
FROM workspaces w
JOIN (
  VALUES
    ('teams', 'enabled', 10, 'Multi-user workspace access'),
    ('rbac', 'enabled', 4, 'Admin, operator, deployer, observer'),
    ('log_retention', 'metered', 30, 'Retention window in days'),
    ('advanced_features', 'enabled', NULL, 'Remote actions, alerts, monitoring exports'),
    ('sso', 'enabled', NULL, 'SAML and SCIM for company domains')
) AS item (feature_key, status, quota, notes) ON TRUE
WHERE w.slug = 'relay-cloud-connect'
ON CONFLICT (workspace_id, feature_key) DO NOTHING;

INSERT INTO sso_connections (workspace_id, provider, domain, status, enforced)
SELECT w.id, 'okta', 'relay.dev', 'active', TRUE
FROM workspaces w
WHERE w.slug = 'relay-cloud-connect'
ON CONFLICT DO NOTHING;

INSERT INTO servers (
  workspace_id,
  name,
  region,
  environment,
  connected,
  connection_mode,
  websocket_session_id,
  agent_version,
  cpu_percent,
  memory_percent,
  active_alerts,
  last_seen_at
)
SELECT
  w.id,
  item.name,
  item.region,
  item.environment,
  item.connected,
  'websocket',
  item.websocket_session_id,
  item.agent_version,
  item.cpu_percent,
  item.memory_percent,
  item.active_alerts,
  NOW() - item.minutes_ago * INTERVAL '1 minute'
FROM workspaces w
JOIN (
  VALUES
    ('nyc-edge-1', 'New York', 'production', TRUE,  'ws_01', 'relayd 0.8.4', 38.0, 64.0, 0, 1),
    ('fra-build-2', 'Frankfurt', 'staging',  TRUE,  'ws_02', 'relayd 0.8.4', 57.0, 71.0, 1, 2),
    ('lon-preview-1', 'London', 'preview',   TRUE,  'ws_03', 'relayd 0.8.3', 21.0, 44.0, 0, 1),
    ('home-lab', 'Dubai', 'dev', FALSE, NULL, 'relayd 0.8.1', 0.0, 0.0, 1, 37)
) AS item (name, region, environment, connected, websocket_session_id, agent_version, cpu_percent, memory_percent, active_alerts, minutes_ago) ON TRUE
WHERE w.slug = 'relay-cloud-connect'
ON CONFLICT (workspace_id, name) DO NOTHING;

INSERT INTO apps (
  workspace_id,
  server_id,
  name,
  runtime,
  repository,
  environment,
  status,
  release_channel,
  desired_instances,
  live_instances
)
SELECT
  w.id,
  s.id,
  item.name,
  item.runtime,
  item.repository,
  item.environment,
  item.status,
  item.release_channel,
  item.desired_instances,
  item.live_instances
FROM workspaces w
JOIN (
  VALUES
    ('nyc-edge-1', 'dashboard-web', 'Next.js', 'github.com/relay/dashboard', 'production', 'healthy', 'stable', 3, 3),
    ('nyc-edge-1', 'billing-api', 'Go', 'github.com/relay/billing-api', 'production', 'healthy', 'stable', 2, 2),
    ('fra-build-2', 'ops-runner', 'Node', 'github.com/relay/ops-runner', 'staging', 'degraded', 'candidate', 2, 1),
    ('lon-preview-1', 'preview-gateway', 'Rust', 'github.com/relay/preview-gateway', 'preview', 'healthy', 'preview', 1, 1)
) AS item (server_name, name, runtime, repository, environment, status, release_channel, desired_instances, live_instances) ON TRUE
JOIN servers s ON s.workspace_id = w.id AND s.name = item.server_name
WHERE w.slug = 'relay-cloud-connect'
ON CONFLICT (workspace_id, name, environment) DO NOTHING;

INSERT INTO deployments (
  workspace_id,
  app_id,
  server_id,
  version,
  source_ref,
  status,
  trigger_kind,
  triggered_by,
  duration_seconds,
  started_at,
  finished_at
)
SELECT
  w.id,
  a.id,
  s.id,
  item.version,
  item.source_ref,
  item.status,
  item.trigger_kind,
  item.triggered_by,
  item.duration_seconds,
  NOW() - item.started_minutes_ago * INTERVAL '1 minute',
  CASE
    WHEN item.finished_minutes_ago IS NULL THEN NULL
    ELSE NOW() - item.finished_minutes_ago * INTERVAL '1 minute'
  END
FROM workspaces w
JOIN (
  VALUES
    ('ops-runner', 'fra-build-2', 'ops-2026.05.01-8', 'refs/heads/main', 'running', 'manual', 'ali@relay.dev', NULL, 6, NULL),
    ('dashboard-web', 'nyc-edge-1', 'web-2026.05.01-4', 'refs/tags/web-2026.05.01-4', 'succeeded', 'git', 'ci@github', 143, 22, 19),
    ('billing-api', 'nyc-edge-1', 'api-2026.05.01-2', 'refs/heads/release', 'succeeded', 'manual', 'sara@relay.dev', 98, 46, 44)
) AS item (app_name, server_name, version, source_ref, status, trigger_kind, triggered_by, duration_seconds, started_minutes_ago, finished_minutes_ago) ON TRUE
JOIN apps a ON a.workspace_id = w.id AND a.name = item.app_name
JOIN servers s ON s.workspace_id = w.id AND s.name = item.server_name
WHERE w.slug = 'relay-cloud-connect'
  AND NOT EXISTS (
    SELECT 1
    FROM deployments existing
    WHERE existing.workspace_id = w.id
      AND existing.app_id = a.id
      AND existing.version = item.version
  );

INSERT INTO metric_snapshots (
  workspace_id,
  server_id,
  captured_at,
  cpu_percent,
  memory_percent,
  disk_percent,
  websocket_latency_ms,
  requests_per_minute
)
SELECT
  w.id,
  s.id,
  NOW() - item.minutes_ago * INTERVAL '1 minute',
  item.cpu_percent,
  item.memory_percent,
  item.disk_percent,
  item.websocket_latency_ms,
  item.requests_per_minute
FROM workspaces w
JOIN (
  VALUES
    ('nyc-edge-1', 1, 38.0, 64.0, 52.0, 42, 1280),
    ('fra-build-2', 2, 57.0, 71.0, 61.0, 88, 740),
    ('lon-preview-1', 1, 21.0, 44.0, 40.0, 61, 290),
    ('home-lab', 37, 0.0, 0.0, 20.0, 0, 0)
) AS item (server_name, minutes_ago, cpu_percent, memory_percent, disk_percent, websocket_latency_ms, requests_per_minute) ON TRUE
JOIN servers s ON s.workspace_id = w.id AND s.name = item.server_name
WHERE w.slug = 'relay-cloud-connect'
  AND NOT EXISTS (
    SELECT 1
    FROM metric_snapshots existing
    WHERE existing.workspace_id = w.id
      AND existing.server_id = s.id
      AND existing.requests_per_minute = item.requests_per_minute
  );

INSERT INTO remote_actions (
  workspace_id,
  server_id,
  app_id,
  action_kind,
  status,
  requested_by,
  note,
  requested_at,
  completed_at
)
SELECT
  w.id,
  s.id,
  a.id,
  item.action_kind,
  item.status,
  item.requested_by,
  item.note,
  NOW() - item.requested_minutes_ago * INTERVAL '1 minute',
  CASE
    WHEN item.completed_minutes_ago IS NULL THEN NULL
    ELSE NOW() - item.completed_minutes_ago * INTERVAL '1 minute'
  END
FROM workspaces w
JOIN (
  VALUES
    ('fra-build-2', 'ops-runner', 'deploy', 'running', 'ali@relay.dev', 'Promote candidate build to staging', 5, NULL),
    ('nyc-edge-1', 'billing-api', 'restart', 'queued', 'nina@relay.dev', 'Drain unhealthy worker after alert', 9, NULL),
    ('home-lab', NULL, 'logs', 'queued', 'marc@relay.dev', 'Fetch buffered logs after reconnect', 14, NULL)
) AS item (server_name, app_name, action_kind, status, requested_by, note, requested_minutes_ago, completed_minutes_ago) ON TRUE
JOIN servers s ON s.workspace_id = w.id AND s.name = item.server_name
LEFT JOIN apps a ON a.workspace_id = w.id AND a.name = item.app_name
WHERE w.slug = 'relay-cloud-connect'
  AND NOT EXISTS (
    SELECT 1
    FROM remote_actions existing
    WHERE existing.workspace_id = w.id
      AND existing.server_id = s.id
      AND existing.action_kind = item.action_kind
      AND existing.requested_by = item.requested_by
      AND COALESCE(existing.note, '') = COALESCE(item.note, '')
  );

INSERT INTO interest_leads (
  email,
  team_name,
  fleet_size,
  needs,
  status,
  priority,
  owner_name,
  review_notes,
  last_contacted_at
)
VALUES
  (
    'ops@northstar.dev',
    'Northstar Platform',
    12,
    'RBAC for operators, 30-day log retention, and Okta SSO before the summer cutover.',
    'qualified',
    'high',
    'Ali',
    'Strong platform fit. Wants pilot on staging first.',
    NOW() - INTERVAL '55 minutes'
  ),
  (
    'infra@aperture.ai',
    'Aperture Infra',
    28,
    'Remote deploy approvals, customer-owned servers, and longer log retention for incident review.',
    'reviewing',
    'urgent',
    'Sara',
    'Multi-region fleet and strong enterprise expansion path.',
    NOW() - INTERVAL '2 hours'
  ),
  (
    'hello@loworbit.io',
    'Low Orbit',
    4,
    'A cleaner way to deploy to customer boxes without asking for port forwarding.',
    'new',
    'medium',
    NULL,
    NULL,
    NULL
  )
ON CONFLICT (email)
DO UPDATE SET
  team_name = EXCLUDED.team_name,
  fleet_size = EXCLUDED.fleet_size,
  needs = EXCLUDED.needs,
  status = EXCLUDED.status,
  priority = EXCLUDED.priority,
  owner_name = EXCLUDED.owner_name,
  review_notes = EXCLUDED.review_notes,
  last_contacted_at = EXCLUDED.last_contacted_at,
  updated_at = NOW();
