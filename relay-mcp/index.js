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
const http = require("http");
const https = require("https");

// ─── Transport config ─────────────────────────────────────────────────────────

const RELAY_SOCKET = process.env.RELAY_SOCKET || null;
const RELAY_URL    = (process.env.RELAY_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const RELAY_TOKEN  = process.env.RELAY_TOKEN || "";

if (!RELAY_SOCKET && !RELAY_TOKEN) {
  process.stderr.write(
    "[relay-mcp] Warning: neither RELAY_SOCKET nor RELAY_TOKEN is set. " +
    "Requests will likely be rejected. Set RELAY_SOCKET for local socket auth " +
    "or RELAY_URL + RELAY_TOKEN for HTTP auth.\n"
  );
}

// ─── HTTP helper ──────────────────────────────────────────────────────────────

/**
 * Make a request to the Relay agent. Returns parsed JSON.
 * @param {"GET"|"POST"|"DELETE"|"PATCH"} method
 * @param {string} apiPath  e.g. "/api/projects"
 * @param {object|undefined} body  JSON body for POST/DELETE/PATCH
 */
function relayRequest(method, apiPath, body) {
  return new Promise((resolve, reject) => {
    const bodyStr = body !== undefined ? JSON.stringify(body) : null;

    const headers = {
      "Content-Type": "application/json",
      Accept: "application/json",
    };
    if (RELAY_TOKEN) headers["X-Relay-Token"] = RELAY_TOKEN;
    if (bodyStr) headers["Content-Length"] = String(Buffer.byteLength(bodyStr));

    function handleResponse(res) {
      const chunks = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => {
        const raw = Buffer.concat(chunks).toString("utf8");
        if (res.statusCode >= 400) {
          let msg = raw.trim();
          try { msg = JSON.parse(raw).error || msg; } catch {}
          return reject(new Error(`HTTP ${res.statusCode}: ${msg}`));
        }
        if (!raw.trim()) return resolve({});
        try {
          resolve(JSON.parse(raw));
        } catch {
          resolve({ text: raw });
        }
      });
      res.on("error", reject);
    }

    let req;
    if (RELAY_SOCKET) {
      req = http.request(
        { socketPath: RELAY_SOCKET, method, path: apiPath, headers },
        handleResponse
      );
    } else {
      const url = new URL(RELAY_URL);
      const isHttps = url.protocol === "https:";
      const lib = isHttps ? https : http;
      req = lib.request(
        {
          hostname: url.hostname,
          port: url.port || (isHttps ? 443 : 80),
          path: apiPath,
          method,
          headers,
        },
        handleResponse
      );
    }

    req.on("error", reject);
    if (bodyStr) req.write(bodyStr);
    req.end();
  });
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
  version: "0.1.0",
});

// ── Projects ───────────────────────────────────────────────────────────────────

server.tool(
  "list_projects",
  "List all projects and their environments known to the Relay agent.",
  {},
  async () => text(await relayRequest("GET", "/api/projects"))
);

// ── Deploys ───────────────────────────────────────────────────────────────────

server.tool(
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

server.tool(
  "get_deploy",
  "Get a single deploy record by ID including status, logs reference, image tag, and commit info.",
  { id: z.string().describe("Deploy ID") },
  async ({ id }) =>
    text(await relayRequest("GET", `/api/deploys/${encodeURIComponent(id)}`))
);

server.tool(
  "get_deploy_logs",
  "Fetch the stored build and deploy logs for a given deploy ID.",
  { id: z.string().describe("Deploy ID") },
  async ({ id }) => {
    const data = await relayRequest("GET", `/api/logs/${encodeURIComponent(id)}`);
    const out = typeof data.text === "string" ? data.text : fmt(data);
    return { content: [{ type: "text", text: out }] };
  }
);

server.tool(
  "cancel_deploy",
  "Cancel an in-progress deploy by ID.",
  { id: z.string().describe("Deploy ID to cancel") },
  async ({ id }) =>
    text(await relayRequest("POST", `/api/deploys/cancel/${encodeURIComponent(id)}`))
);

server.tool(
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

server.tool(
  "start_app",
  "Start a stopped app container without triggering a new build.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/start", { app, env, branch }))
);

server.tool(
  "stop_app",
  "Stop a running app container.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/stop", { app, env, branch }))
);

server.tool(
  "restart_app",
  "Restart a running app container in place without a new build.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/restart", { app, env, branch }))
);

server.tool(
  "delete_lane",
  "Delete a lane and remove its container, workspace, and state from the agent.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("POST", "/api/apps/delete-lane", { app, env, branch }))
);

// ── App config ────────────────────────────────────────────────────────────────

server.tool(
  "get_app_config",
  "Get the current lane configuration for an app including access_policy, public_host, engine, host_port, service_port, and rollout settings.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("GET", `/api/apps/config${qs({ app, env, branch })}`))
);

server.tool(
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
  },
  async ({ app, env, branch, ...rest }) => {
    const payload = { app, env, branch, ...Object.fromEntries(
      Object.entries(rest).filter(([, v]) => v !== undefined)
    )};
    return text(await relayRequest("POST", "/api/apps/config", payload));
  }
);

// ── Secrets ───────────────────────────────────────────────────────────────────

server.tool(
  "list_secrets",
  "List secret key names (not values) for an app/env/branch.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("GET", `/api/apps/secrets${qs({ app, env, branch })}`))
);

server.tool(
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

server.tool(
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

server.tool(
  "list_promotions",
  "List promotion requests. Shows source/target lane, image, approval state, and post-promote health status.",
  {
    app:    z.string().optional().describe("Filter by app name"),
    status: z.string().optional().describe("Filter by status: pending, approved, running, success, failed"),
  },
  async ({ app, status }) =>
    text(await relayRequest("GET", `/api/promotions${qs({ app, status })}`))
);

server.tool(
  "approve_promotion",
  "Approve a queued promotion request by ID. Requires owner role.",
  { id: z.string().describe("Promotion ID") },
  async ({ id }) =>
    text(await relayRequest("POST", "/api/promotions/approve", { id }))
);

// ── Users and audit ───────────────────────────────────────────────────────────

server.tool(
  "list_users",
  "List all user accounts with their roles. Requires owner role.",
  {},
  async () => text(await relayRequest("GET", "/api/users"))
);

server.tool(
  "get_audit_log",
  "Fetch recent audit log entries showing actor, action, target, and timestamp. Requires owner role.",
  {
    limit: z.number().int().min(1).max(500).default(50).describe("Max entries"),
  },
  async ({ limit }) =>
    text(await relayRequest("GET", `/api/audit${qs({ limit })}`))
);

// ── Server config ─────────────────────────────────────────────────────────────

server.tool(
  "get_server_config",
  "Get server-level config: base_domain, dashboard_host, ACME settings, and active theme. Requires owner role.",
  {},
  async () => text(await relayRequest("GET", "/api/server/config"))
);

server.tool(
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

server.tool(
  "list_buildpack_plugins",
  "List installed buildpack plugins on the agent.",
  {},
  async () => text(await relayRequest("GET", "/api/plugins/buildpacks"))
);

server.tool(
  "remove_buildpack_plugin",
  "Remove an installed buildpack plugin by name. Requires RELAY_ENABLE_PLUGIN_MUTATIONS=true on the agent.",
  { name: z.string().describe("Plugin name to remove") },
  async ({ name }) =>
    text(await relayRequest("DELETE", `/api/plugins/buildpacks/${encodeURIComponent(name)}`))
);

// ── Version ───────────────────────────────────────────────────────────────────

server.tool(
  "get_version",
  "Get relayd build metadata and Station runtime version details.",
  {},
  async () => text(await relayRequest("GET", "/api/version"))
);

// ── Signed links ──────────────────────────────────────────────────────────────

server.tool(
  "create_signed_link",
  "Generate a time-bounded signed share URL for a lane that uses access_policy=signed-link.",
  {
    app:        z.string(),
    env:        z.string(),
    branch:     z.string().default("main"),
    expires_in: z.number().int().min(60).default(3600).describe("Link lifetime in seconds"),
  },
  async ({ app, env, branch, expires_in }) =>
    text(await relayRequest("POST", "/api/apps/signed-link", { app, env, branch, expires_in }))
);

// ── Companions ────────────────────────────────────────────────────────────────

server.tool(
  "list_companions",
  "List managed companion services (databases, caches, etc.) attached to an app lane.",
  appParams,
  async ({ app, env, branch }) =>
    text(await relayRequest("GET", `/api/apps/companions${qs({ app, env, branch })}`))
);

server.tool(
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

// ─── Start ────────────────────────────────────────────────────────────────────

const transport = new StdioServerTransport();
server.connect(transport).catch((err) => {
  process.stderr.write(`[relay-mcp] fatal: ${err.message}\n`);
  process.exit(1);
});
