#!/usr/bin/env node
/**
 * relay-mcp — Model Context Protocol server for Relay
 *
 * Connect to relayd via:
 *   RELAY_SOCKET=/path/to/relay.sock   Unix socket (no token needed — filesystem ACL is auth)
 *   RELAY_URL=http://server:8080       HTTP (requires RELAY_TOKEN)
 *   RELAY_TOKEN=your-token
 *
 * Usage in MCP config:
 *   {
 *     "mcpServers": {
 *       "relay": {
 *         "command": "npx",
 *         "args": ["-y", "@relay-org/relay-mcp"],
 *         "env": {
 *           "RELAY_URL": "http://your-server:8080",
 *           "RELAY_TOKEN": "your-token"
 *         }
 *       }
 *     }
 *   }
 */
"use strict";

const { McpServer } = require("@modelcontextprotocol/sdk/server/mcp.js");
const { StdioServerTransport } = require("@modelcontextprotocol/sdk/server/stdio.js");
const { z } = require("zod");
const { readConfig, createRelayRequest } = require("./transport.js");
const { version } = require("./package.json");

let config;
try {
  config = readConfig();
} catch (error) {
  process.stderr.write(`[relay-mcp] ${error.message}\n`);
  process.exit(1);
}
const relayRequest = createRelayRequest(config);
if (!config.socketPath && !config.token) {
  process.stderr.write("[relay-mcp] RELAY_TOKEN is not set. Authenticated HTTP tools require a token for this server.\n");
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function qs(params) {
  const pairs = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== null && v !== ""
  );
  if (!pairs.length) return "";
  return "?" + pairs.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`).join("&");
}

function fmt(obj) {
  return JSON.stringify(obj, null, 2);
}

function text(obj) {
  return { content: [{ type: "text", text: fmt(obj) }] };
}

// ─── MCP server ───────────────────────────────────────────────────────────────

const server = new McpServer({
  name: "relay",
  version,
});

// Help clients distinguish inspection from operations that change the server.
function registerTool(name, description, inputSchema, handler) {
  const readOnly = name.startsWith("get_") || name.startsWith("list_");
  return server.registerTool(name, {
    description,
    inputSchema,
    annotations: {
      readOnlyHint: readOnly,
      destructiveHint: !readOnly,
      idempotentHint: readOnly,
      openWorldHint: true,
    },
  }, async (args, extra) => {
    try { return await handler(args, extra); }
    catch (error) {
      return { isError: true, content: [{ type: "text", text: error.message || String(error) }] };
    }
  });
}

registerTool(
  "get_doctor",
  "Check agent setup and diagnostics, including Docker, persistent storage, DNS, TLS, and proxy health. Read-only; does not change server settings.",
  {},
  async () => text(await relayRequest("GET", "/api/doctor"))
);

// ── Projects ───────────────────────────────────────────────────────────────────

registerTool(
  "list_projects",
  "List all projects and their environments known to the Relay agent.",
  {},
  async () => text(await relayRequest("GET", "/api/projects"))
);

// ── Deploys ───────────────────────────────────────────────────────────────────

registerTool(
  "list_deploys",
  "List recent deploys. Filter by app, env, and/or branch. Returns build numbers, statuses, deployed_by, commit messages, and timestamps.",
  {
    app:    z.string().optional().describe("App name"),
    env:    z.string().optional().describe("Environment: dev, staging, prod, etc."),
    branch: z.string().optional().describe("Branch name"),
    limit:  z.number().int().min(1).max(200).default(20).describe("Max results"),
  },
  async ({ app, env, branch, limit }) =>
    text(await relayRequest("GET", `/api/deploys${qs({ app, env, branch, limit })}`))
);

registerTool(
  "get_deploy",
  "Get a single deploy record by ID including status, logs reference, image tag, and commit info.",
  { id: z.string().describe("Deploy ID") },
  async ({ id }) =>
    text(await relayRequest("GET", `/api/deploys/${encodeURIComponent(id)}`))
);

registerTool(
  "get_deploy_logs",
  "Fetch the stored build and deploy logs for a given deploy ID.",
  { id: z.string().describe("Deploy ID") },
  async ({ id }) => {
    const data = await relayRequest("GET", `/api/logs/${encodeURIComponent(id)}`);
    const out = typeof data.text === "string" ? data.text : fmt(data);
    return { content: [{ type: "text", text: out }] };
  }
);

registerTool(
  "cancel_deploy",
  "Cancel an in-progress deploy by ID.",
  { id: z.string().describe("Deploy ID to cancel") },
  async ({ id }) =>
    text(await relayRequest("POST", `/api/deploys/cancel/${encodeURIComponent(id)}`))
);

registerTool(
  "rollback",
  "Roll back the most recent deploy to the previous image for an app/env/branch.",
  {
    app:    z.string().describe("App name"),
    env:    z.string().describe("Environment"),
    branch: z.string().default("main").describe("Branch"),
  },
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/deploys/rollback", { app, env, branch }))
);

// ── App control ───────────────────────────────────────────────────────────────

const appParams = {
  app:    z.string().describe("App name"),
  env:    z.string().describe("Environment"),
  branch: z.string().default("main").describe("Branch"),
};

registerTool(
  "start_app",
  "Start a stopped app container without triggering a new build.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/start", { app, env, branch }))
);

registerTool(
  "stop_app",
  "Stop a running app container.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/stop", { app, env, branch }))
);

registerTool(
  "restart_app",
  "Restart a running app container in place without a new build.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/restart", { app, env, branch }))
);

registerTool(
  "delete_lane",
  "Delete a lane and remove its container, workspace, and state from the agent.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/delete-lane", { app, env, branch }))
);

// ── App config ────────────────────────────────────────────────────────────────

registerTool(
  "get_app_config",
  "Get the current lane configuration for an app including access_policy, public_host, engine, host_port, service_port, and rollout settings.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("GET", `/api/apps/config${qs({ app, env, branch })}`))
);

registerTool(
  "set_app_config",
  "Update lane configuration for an app. Only supply the fields you want to change.",
  {
    app:            z.string(),
    env:            z.string(),
    branch:         z.string().default("main"),
    access_policy:  z.enum(["public", "relay-login", "signed-link", "ip-allowlist"]).optional()
                     .describe("Who can access the lane"),
    ip_allowlist:   z.string().optional().describe("Comma-separated CIDRs for ip-allowlist policy"),
    public_host:    z.string().optional().describe("Custom public hostname"),
    host_port:      z.number().int().optional().describe("External host port"),
    service_port:   z.number().int().optional().describe("Container service port"),
    engine:         z.enum(["docker", "station"]).optional(),
    webhook_secret: z.string().optional().describe("Per-app GitHub webhook secret"),
    expires_at:     z.number().optional().describe("Lane expiry as Unix ms timestamp"),
    traffic_mode:   z.enum(["edge", "session"]).optional().describe("Variants require edge traffic mode"),
    traffic_split_percent: z.number().int().min(1).max(100).optional().describe("New release share during an edge canary; session mode forces 100"),
    rollout_min_requests: z.number().int().min(1).optional(),
    rollout_error_percent: z.number().positive().max(100).optional().describe("Server-error threshold for monitored rollout assessment"),
    rollout_assess_seconds: z.number().int().min(30).optional(),
    cpu_limit:      z.string().optional().describe("Per-container Docker CPU limit, for example 0.5; empty clears the manual limit"),
    mem_limit:      z.string().optional().describe("Per-container Docker memory limit, for example 512m or 1g; empty uses the host-sized default"),
    resource_mode:  z.enum(["manual", "auto"]).optional().describe("Auto ignores manual caps; it does not dynamically autoscale"),
    public_hosts:   z.array(z.string()).optional().describe("Additional public hostnames"),
    notification_webhooks: z.string().optional().describe("Lane notification webhook configuration"),
  },
  async ({ app, env, branch, ...rest }) => {
    const payload = { app, env, branch, ...Object.fromEntries(
      Object.entries(rest).filter(([, v]) => v !== undefined)
    )};
    return text(await relayRequest("POST", "/api/apps/config", payload));
  }
);

// ── Secrets ───────────────────────────────────────────────────────────────────

registerTool(
  "list_secrets",
  "List secret key names (not values) for an app/env/branch.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("GET", `/api/apps/secrets${qs({ app, env, branch })}`))
);

registerTool(
  "add_secret",
  "Add or update a secret value for an app/env/branch. The value is encrypted at rest if RELAY_SECRET_KEY is configured on the agent.",
  {
    app:    z.string(),
    env:    z.string(),
    branch: z.string().default("main"),
    key:    z.string().describe("Secret key name"),
    value:  z.string().describe("Secret value"),
  },
  async ({ app, env, branch, key, value }) =>
    text(await relayRequest("POST", "/api/apps/secrets", { app, env, branch, key, value }))
);

registerTool(
  "remove_secret",
  "Delete a secret by key for an app/env/branch.",
  {
    app:    z.string(),
    env:    z.string(),
    branch: z.string().default("main"),
    key:    z.string().describe("Secret key name to delete"),
  },
  async ({ app, env, branch, key }) =>
    text(await relayRequest("DELETE", "/api/apps/secrets", { app, env, branch, key }))
);

// ── Promotions ────────────────────────────────────────────────────────────────

registerTool(
  "list_promotions",
  "List promotion requests. Shows source/target lane, image, approval state, and post-promote health status.",
  {
    app:    z.string().optional().describe("Filter by app name"),
    status: z.string().optional().describe("Filter by status: pending, approved, running, success, failed"),
  },
  async ({ app, status }) =>
    text(await relayRequest("GET", `/api/promotions${qs({ app, status })}`))
);

registerTool(
  "approve_promotion",
  "Approve a queued promotion request by ID. Requires owner role.",
  { id: z.string().describe("Promotion ID") },
  async ({ id }) =>
    text(await relayRequest("POST", "/api/promotions/approve", { id }))
);

// ── Users and audit ───────────────────────────────────────────────────────────

registerTool(
  "list_users",
  "List all user accounts with their roles. Requires owner role.",
  {},
  async () => text(await relayRequest("GET", "/api/users"))
);

registerTool(
  "get_audit_log",
  "Fetch recent audit log entries showing actor, action, target, and timestamp. Requires owner role.",
  {
    limit: z.number().int().min(1).max(500).default(50).describe("Max entries"),
  },
  async ({ limit }) =>
    text(await relayRequest("GET", `/api/audit${qs({ limit })}`))
);

// ── Server config ─────────────────────────────────────────────────────────────

registerTool(
  "get_server_config",
  "Get server-level config: base_domain, dashboard_host, ACME settings, and active theme. Requires owner role.",
  {},
  async () => text(await relayRequest("GET", "/api/server/config"))
);

registerTool(
  "set_server_config",
  "Update server-level config. Supply only the fields to change. Requires owner role.",
  {
    base_domain:    z.string().optional().describe("Wildcard base domain for managed lane hosts"),
    dashboard_host: z.string().optional().describe("Custom hostname for the admin dashboard"),
    acme_disabled:  z.boolean().optional().describe("Disable automatic TLS via ACME"),
    theme_name:     z.string().optional().describe("Built-in theme name: default, midnight, slate, forest, rose, ocean"),
    theme_css:      z.string().optional().describe("Custom CSS to apply as the dashboard theme"),
  },
  async (params) => {
    const payload = Object.fromEntries(
      Object.entries(params).filter(([, v]) => v !== undefined)
    );
    return text(await relayRequest("POST", "/api/server/config", payload));
  }
);

// ── Plugins ───────────────────────────────────────────────────────────────────

registerTool(
  "list_buildpack_plugins",
  "List installed buildpack plugins on the agent.",
  {},
  async () => text(await relayRequest("GET", "/api/plugins/buildpacks"))
);

registerTool(
  "remove_buildpack_plugin",
  "Remove an installed buildpack plugin by name. Requires RELAY_ENABLE_PLUGIN_MUTATIONS=true on the agent.",
  { name: z.string().describe("Plugin name to remove") },
  async ({ name }) =>
    text(await relayRequest("DELETE", `/api/plugins/buildpacks/${encodeURIComponent(name)}`))
);

// ── Version ───────────────────────────────────────────────────────────────────

registerTool(
  "get_version",
  "Get relayd build metadata and Station runtime version details.",
  {},
  async () => text(await relayRequest("GET", "/api/version"))
);

// ── Signed links ──────────────────────────────────────────────────────────────

registerTool(
  "create_signed_link",
  "Generate a time-bounded signed share URL for a lane that uses access_policy=signed-link.",
  {
    app:        z.string(),
    env:        z.string(),
    branch:     z.string().default("main"),
    expires_in: z.number().int().min(60).default(3600).describe("Link lifetime in seconds; rounded up to whole minutes and capped at seven days"),
  },
  async ({ app, env, branch, expires_in }) =>
    text(await relayRequest("POST", "/api/apps/signed-link", {
      app, env, branch,
      expires_in_minutes: Math.min(7 * 24 * 60, Math.ceil(expires_in / 60)),
    }))
);

// ── Companions ────────────────────────────────────────────────────────────────

registerTool(
  "list_companions",
  "List managed companion services (databases, caches, etc.) attached to an app lane.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("GET", `/api/apps/companions${qs({ app, env, branch })}`))
);

registerTool(
  "restart_companion",
  "Restart a named companion service in place.",
  {
    app:    z.string(),
    env:    z.string(),
    branch: z.string().default("main"),
    name:   z.string().describe("Companion service name"),
  },
  async ({ app, env, branch, name }) =>
    text(await relayRequest("POST", "/api/apps/companions/restart", { app, env, branch, name }))
);

// ── Variants and replicas ────────────────────────────────────────────────────

const reportPeriod = z.enum(["24h", "7d", "30d"]).default("7d");
const variantName = z.string().trim().toLowerCase().regex(/^[a-z0-9][a-z0-9-]{0,23}$/)
  .refine((name) => !["control", "live", "active", "new", "old", "blue", "green", "standby", "previous", "draining"].includes(name), "Reserved variant name");

registerTool(
  "list_variants",
  "Read a lane's control release, variants, traffic weights, request/error/latency metrics, and latest 50 feedback entries for 24h, 7d, or 30d. Includes support limitations.",
  { ...appParams, period: reportPeriod },
  async (params) => text(await relayRequest("GET", `/api/variants${qs(params)}`))
);

registerTool(
  "set_variant_weight",
  "Change a variant's share of new visitors (0-100 percent). Existing visitor assignments remain sticky. All running variant weights must total at most 100; control receives the remainder.",
  { ...appParams, name: variantName, weight: z.number().min(0).max(100) },
  async (params) => text(await relayRequest("POST", "/api/variants", { ...params, action: "weight" }))
);

registerTool(
  "promote_variant",
  "Queue a deployment of a variant's existing image as the lane's control release. Returns deploy_id; the variant is retired after that deployment succeeds. This is separate from cross-lane promotion approvals.",
  { ...appParams, name: variantName },
  async (params) => text(await relayRequest("POST", "/api/variants", { ...params, action: "promote" }))
);

registerTool(
  "remove_variant",
  "Remove a variant's route and containers. Visitors assigned to it fall back to control. Does not remove the lane's control release.",
  { ...appParams, name: variantName },
  async (params) => text(await relayRequest("POST", "/api/variants", { ...params, action: "remove" }))
);

registerTool(
  "reshuffle_variants",
  "Reset visitor assignments for a lane under its current variant weights. Every visitor is assigned again on a subsequent request.",
  appParams,
  async (params) => text(await relayRequest("POST", "/api/variants", { ...params, action: "reshuffle" }))
);

registerTool(
  "scale_app",
  "Set 1-16 replicas for an existing Docker lane. With apply=true, queues an image redeployment when a running lane needs resizing. With apply=false, saves the size for a later deployment. Returns deploy_id when queued.",
  { ...appParams, replicas: z.number().int().min(1).max(16), apply: z.boolean().default(true) },
  async (params) => text(await relayRequest("POST", "/api/apps/scale", params))
);

// These routes are registered only on relayd's network listener.
function requireHTTP(toolName) {
  if (config.socketPath) throw new Error(`${toolName} requires HTTP/HTTPS: this agent route is not exposed on the Unix socket. Configure RELAY_URL and unset RELAY_SOCKET.`);
}

registerTool(
  "get_analytics",
  "Read lane traffic, errors, latency, bandwidth, and runtime resource data, optionally filtered to one variant (or control). HTTP/HTTPS only; scoped viewer access is required.",
  { ...appParams, period: reportPeriod, variant: z.string().optional().describe("Variant name or control; omit for the whole lane") },
  async (params) => {
    requireHTTP("get_analytics");
    return text(await relayRequest("GET", `/api/analytics${qs(params)}`));
  }
);

registerTool(
  "abort_rollout",
  "Abort a canary rollout that is currently monitoring and restore its previous slot. Requires an active standby slot. HTTP/HTTPS only.",
  appParams,
  async (params) => {
    requireHTTP("abort_rollout");
    return text(await relayRequest("POST", "/api/deploys/rollout-abort", params));
  }
);

registerTool(
  "request_promotion",
  "Request promotion of a healthy lane's current image to a different lane. Requires deployer access to both lanes. Production requests by non-owners wait for owner approval when user accounts are enabled.",
  { app: z.string(), source_env: z.string(), source_branch: z.string().default("main"), target_env: z.string(), target_branch: z.string().optional().describe("Defaults to source_branch") },
  async (params) => text(await relayRequest("POST", "/api/promotions", { ...params, target_branch: params.target_branch ?? params.source_branch }))
);

// ── Runtime inspection ──────────────────────────────────────────────────────

registerTool(
  "get_runtime_targets",
  "List available app, replica, variant, proxy, and companion runtime log targets for a lane, plus the default target and offline state. Use the dashboard Runtime logs page to follow a target's live output.",
  appParams,
  async (params) => text(await relayRequest("GET", `/api/runtime/logs/targets${qs(params)}`))
);

registerTool(
  "get_admin_ops",
  "Read the owner-only operations snapshot: container CPU, memory, storage, lane health, daemon memory, and deployment comparisons.",
  {},
  async () => text(await relayRequest("GET", "/api/admin/ops"))
);

registerTool(
  "get_status",
  "Read the agent's runtime health inventory, including edge and active-slot state, stopped lanes, exit codes, and out-of-memory indicators.",
  {},
  async () => text(await relayRequest("GET", "/api/status"))
);

// ── Plugin catalog and installation ─────────────────────────────────────────

registerTool(
  "list_buildpack_catalog",
  "Read the built-in buildpack plugin catalog. Requires owner role.",
  {},
  async () => text(await relayRequest("GET", "/api/plugins/catalog"))
);

registerTool(
  "install_buildpack_from_catalog",
  "Install a named plugin from the agent's built-in catalog. Requires owner role and plugin mutations enabled on the agent.",
  { name: z.string().min(1) },
  async (params) => text(await relayRequest("POST", "/api/plugins/catalog", params))
);

registerTool(
  "install_buildpack_from_url",
  "Install buildpack plugin JSON from an HTTPS URL, optionally pinned to a SHA256 digest. Requires owner role and plugin mutations enabled. The agent validates and fetches the URL.",
  {
    url: z.string().url().refine((value) => new URL(value).protocol === "https:", "Use an HTTPS URL"),
    sha256: z.string().regex(/^[a-fA-F0-9]{64}$/).optional().describe("Expected SHA256 hex digest of the downloaded JSON"),
  },
  async (params) => text(await relayRequest("POST", "/api/plugins/buildpacks/install-url", params))
);

// ─── Start ────────────────────────────────────────────────────────────────────

const transport = new StdioServerTransport();
server.connect(transport).catch((err) => {
  process.stderr.write(`[relay-mcp] fatal: ${err.message}\n`);
  process.exit(1);
});
