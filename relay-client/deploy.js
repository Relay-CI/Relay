const path = require("path");
const {
  loadRelayConfig,
  getServerSession,
  normalizeServerUrl,
} = require("./config");

function pick(...vals) {
  for (const v of vals) {
    if (v === undefined || v === null) continue;
    if (typeof v === "string" && v.trim() === "") continue;
    return v;
  }
  return undefined;
}

function normalizeLaneEnv(value) {
  const raw = String(value ?? "").trim().toLowerCase();
  if (!raw) return "";
  if (raw === "prod" || raw === "production") return "prod";
  if (raw === "staging" || raw === "stage") return "staging";
  if (raw === "dev" || raw === "development" || raw === "ddev") return "dev";
  if (raw === "preview") return "preview";
  return raw;
}

function loadCommandConfig(cli = {}) {
  const startDir = cli.dir && cli.dir !== true
    ? path.resolve(String(cli.dir))
    : process.cwd();
  return loadRelayConfig(startDir);
}

function resolveConnection(cli = {}) {
  const cfg = loadCommandConfig(cli);
  const socket = pick(cli.socket, cfg.data.socket, process.env.RELAY_SOCKET);
  const url = pick(
    cli.url,
    cfg.data.url,
    process.env.RELAY_URL,
    "http://127.0.0.1:8080",
  );
  const normalizedUrl = normalizeServerUrl(url);
  const configuredUrl = normalizeServerUrl(cfg.data.url);
  const configuredToken = !configuredUrl || configuredUrl === normalizedUrl
    ? cfg.data.token
    : undefined;
  const savedSession = getServerSession(normalizedUrl);
  const token = pick(
    cli.token,
    configuredToken,
    savedSession && savedSession.token,
    process.env.RELAY_TOKEN,
  );
  return { cfg, socket: socket || null, url: normalizedUrl, token };
}

function resolveDeployArgs(cli = {}) {
  const { cfg, socket, url, token } = resolveConnection(cli);

  const resolved = {
    url,
    token,
    socket: socket || null,
    app:    pick(cli.app,    cfg.data.app,    process.env.RELAY_APP),
    env:    normalizeLaneEnv(pick(cli.env,    cfg.data.env,    process.env.RELAY_ENV,    "preview")),
    branch: pick(cli.branch, cfg.data.branch, process.env.RELAY_BRANCH, "main"),
    dir:    pick(cli.dir,    cfg.data.dir, "."),
  };

  const missing = [];
  if (!resolved.app) missing.push("--app (or RELAY_APP or config app)");
  // Token is not required when a socket is available — filesystem ACL is the gate.
  if (!resolved.socket && !resolved.token) missing.push("--token (or RELAY_TOKEN or config token), or --socket");
  if (!resolved.env) missing.push("--env (or RELAY_ENV or config env)");
  if (!resolved.branch) missing.push("--branch (or RELAY_BRANCH or config branch)");

  if (missing.length) {
    const used = cfg.path ? `Loaded config: ${cfg.path}` : "No config found";
    throw new Error(`${used}\nMissing required: ${missing.join(", ")}`);
  }

  return resolved;
}

function resolveServerArgs(cli = {}) {
  const { cfg, socket, url, token } = resolveConnection(cli);
  const resolved = {
    url,
    token,
    socket: socket || null,
  };
  if (!resolved.socket && !resolved.token) {
    const used = cfg.path ? `Loaded config: ${cfg.path}` : "No config found";
    throw new Error(`${used}\nMissing --token (or RELAY_TOKEN or config token), or --socket`);
  }
  return resolved;
}

/**
 * Resolve and return a transport object from CLI args / env vars / config.
 *
 *   { kind: "socket", socketPath: "...", token: "" }
 *   { kind: "http",   baseUrl: "...",    token: "..." }
 */
function resolveTransport(cli = {}) {
  const { cfg, socket, url, token } = resolveConnection(cli);
  if (socket) {
    return { kind: "socket", socketPath: socket, token: token || "" };
  }
  if (!token) {
    const used = cfg.path ? `Loaded config: ${cfg.path}` : "No config found";
    throw new Error(`${used}\nMissing --token (or RELAY_TOKEN or config token), or --socket for local auth`);
  }
  return { kind: "http", baseUrl: url, token };
}

module.exports = { resolveDeployArgs, resolveServerArgs, resolveTransport, normalizeLaneEnv };
