const fs = require("fs");
const path = require("path");
const os = require("os");

function findRelayConfig(startDir = process.cwd()) {
  let dir = startDir;

  while (true) {
    const p1 = path.join(dir, ".relay", "config.json");
    const p2 = path.join(dir, ".relay.json");
    const p3 = path.join(dir, "relay.config.json");

    if (fs.existsSync(p1)) return p1;
    if (fs.existsSync(p2)) return p2;
    if (fs.existsSync(p3)) return p3;

    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return null;
}

function loadRelayConfig(startDir) {
  const p = findRelayConfig(startDir);
  if (!p) return { path: null, data: {} };

  const raw = fs.readFileSync(p, "utf8");
  let data = {};
  try {
    data = JSON.parse(raw);
  } catch (e) {
    throw new Error(`Invalid JSON in ${p}`);
  }
  return { path: p, data };
}

/**
 * Write (or merge) fields into the project .relay.json.
 * Creates the file if it doesn't exist.
 */
function saveRelayConfig(fields, startDir = process.cwd()) {
  const existing = loadRelayConfig(startDir);
  const cfgPath = existing.path || path.join(startDir, ".relay.json");
  const merged = { ...existing.data, ...fields };
  fs.writeFileSync(cfgPath, JSON.stringify(merged, null, 2) + "\n", "utf8");
  return cfgPath;
}

// ─── Workspace state (persisted per-machine, not committed to git) ────────────
// Stored in ~/.relay-state.json (global, keyed by server URL + app/env/branch).

function relayStatePath() {
  return path.join(os.homedir(), ".relay-state.json");
}

function loadRelayState() {
  try {
    return JSON.parse(fs.readFileSync(relayStatePath(), "utf8"));
  } catch {
    return {};
  }
}

function saveRelayState(state) {
  fs.writeFileSync(relayStatePath(), JSON.stringify(state, null, 2) + "\n", "utf8");
}

/**
 * Read the saved workspace version for a given server+app+env+branch combo.
 */
function getWorkspaceVersion(baseUrl, app, env, branch) {
  const state = loadRelayState();
  const key = `${baseUrl}|${app}|${env}|${branch}`;
  return (state.workspace_versions || {})[key] || "";
}

/**
 * Persist the workspace version returned by the server after a successful deploy.
 */
function setWorkspaceVersion(baseUrl, app, env, branch, version) {
  const state = loadRelayState();
  if (!state.workspace_versions) state.workspace_versions = {};
  const key = `${baseUrl}|${app}|${env}|${branch}`;
  state.workspace_versions[key] = version;
  saveRelayState(state);
}

module.exports = {
  findRelayConfig,
  loadRelayConfig,
  saveRelayConfig,
  loadRelayState,
  saveRelayState,
  getWorkspaceVersion,
  setWorkspaceVersion,
};
