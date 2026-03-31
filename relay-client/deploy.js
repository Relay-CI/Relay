const { loadRelayConfig } = require("./config");

function pick(...vals) {
  for (const v of vals) {
    if (v === undefined || v === null) continue;
    if (typeof v === "string" && v.trim() === "") continue;
    return v;
  }
  return undefined;
}

function resolveDeployArgs(cli = {}) {
  const cfg = loadRelayConfig();
  const socket = pick(cli.socket, process.env.RELAY_SOCKET, cfg.data.socket);

  const resolved = {
    url:    pick(cli.url,    process.env.RELAY_URL,    cfg.data.url,    "http://127.0.0.1:8080"),
    token:  pick(cli.token,  process.env.RELAY_TOKEN,  cfg.data.token),
    socket: socket || null,
    app:    pick(cli.app,    process.env.RELAY_APP,    cfg.data.app),
    env:    pick(cli.env,    process.env.RELAY_ENV,    cfg.data.env,    "preview"),
    branch: pick(cli.branch, process.env.RELAY_BRANCH, cfg.data.branch, "main"),
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
  const cfg = loadRelayConfig();
  const socket = pick(cli.socket, process.env.RELAY_SOCKET, cfg.data.socket);
  const resolved = {
    url:    pick(cli.url,   process.env.RELAY_URL,   cfg.data.url,   "http://127.0.0.1:8080"),
    token:  pick(cli.token, process.env.RELAY_TOKEN, cfg.data.token),
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
  const cfg    = loadRelayConfig();
  const socket = pick(cli.socket, process.env.RELAY_SOCKET, cfg.data.socket);
  if (socket) {
    return { kind: "socket", socketPath: socket, token: pick(cli.token, process.env.RELAY_TOKEN, cfg.data.token) || "" };
  }
  const baseUrl = pick(cli.url, process.env.RELAY_URL, cfg.data.url, "http://127.0.0.1:8080");
  const token   = pick(cli.token, process.env.RELAY_TOKEN, cfg.data.token);
  if (!token) {
    const used = cfg.path ? `Loaded config: ${cfg.path}` : "No config found";
    throw new Error(`${used}\nMissing --token (or RELAY_TOKEN or config token), or --socket for local auth`);
  }
  return { kind: "http", baseUrl, token };
}

module.exports = { resolveDeployArgs, resolveServerArgs, resolveTransport };
