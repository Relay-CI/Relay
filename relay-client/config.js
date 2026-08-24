const fs = require("fs");
const path = require("path");
const os = require("os");

function findRepositoryRoot(startDir = process.cwd()) {
  const start = path.resolve(startDir);
  let cursor = start;
  while (true) {
    if (fs.existsSync(path.join(cursor, ".git"))) {
      return cursor;
    }
    const parent = path.dirname(cursor);
    if (parent === cursor) break;
    cursor = parent;
  }
  return null;
}

function findRelayConfig(startDir = process.cwd()) {
  const start = path.resolve(startDir);
  const repositoryRoot = findRepositoryRoot(start);

  // Inside a repository, inherit config only as far as that repository root.
  // Outside a repository, use only the requested directory. This prevents a
  // home-level .relay.json from silently selecting a server for every project.
  const boundary = repositoryRoot || start;
  let dir = start;

  while (true) {
    const p1 = path.join(dir, ".relay", "config.json");
    const p2 = path.join(dir, ".relay.json");

    if (fs.existsSync(p1)) return p1;
    if (fs.existsSync(p2)) return p2;

    if (dir === boundary) break;
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
  const dir = path.resolve(startDir);
  const cfgPath = path.join(dir, ".relay.json");
  let existing = {};
  if (fs.existsSync(cfgPath)) {
    try {
      existing = JSON.parse(fs.readFileSync(cfgPath, "utf8"));
    } catch {
      throw new Error(`Invalid JSON in ${cfgPath}`);
    }
  }
  const merged = { ...existing, ...fields };
  fs.writeFileSync(cfgPath, JSON.stringify(merged, null, 2) + "\n", "utf8");
  return cfgPath;
}

// ─── Workspace state (persisted per-machine, not committed to git) ────────────
// Stored in ~/.relay-state.json. Login sessions are keyed by server URL;
// workspace versions are keyed by server URL + app/env/branch.

function relayStatePath() {
  return process.env.RELAY_STATE_PATH || path.join(os.homedir(), ".relay-state.json");
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

function normalizeServerUrl(value) {
  const raw = String(value || "").trim().replace(/\/+$/, "");
  if (!raw) return "";
  try {
    return new URL(raw).toString().replace(/\/+$/, "");
  } catch {
    return raw;
  }
}

function getServerSession(baseUrl) {
  const key = normalizeServerUrl(baseUrl);
  if (!key) return null;
  const state = loadRelayState();
  return (state.server_sessions || {})[key] || null;
}

function setServerSession(baseUrl, session) {
  const key = normalizeServerUrl(baseUrl);
  if (!key) throw new Error("Server URL is required to save a login session");
  const state = loadRelayState();
  if (!state.server_sessions) state.server_sessions = {};
  state.server_sessions[key] = {
    ...session,
    updated_at: Date.now(),
  };
  saveRelayState(state);
}

function deleteServerSession(baseUrl) {
  const key = normalizeServerUrl(baseUrl);
  if (!key) return;
  const state = loadRelayState();
  if (!state.server_sessions || !state.server_sessions[key]) return;
  delete state.server_sessions[key];
  saveRelayState(state);
}

/**
 * Read the saved workspace version for a given server+app+env+branch combo.
 */
function getWorkspaceVersion(baseUrl, app, env, branch) {
  const state = loadRelayState();
  const key = `${baseUrl}|${app}|${env}|${branch}`;
  return (state.workspace_versions || {})[key] || "";
}

function getWorkspaceLocalFingerprint(baseUrl, app, env, branch) {
  const state = loadRelayState();
  const key = `${baseUrl}|${app}|${env}|${branch}`;
  return (state.workspace_local_fingerprints || {})[key] || "";
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

function setWorkspaceLocalFingerprint(baseUrl, app, env, branch, fingerprint) {
  const state = loadRelayState();
  if (!state.workspace_local_fingerprints) state.workspace_local_fingerprints = {};
  const key = `${baseUrl}|${app}|${env}|${branch}`;
  state.workspace_local_fingerprints[key] = fingerprint;
  saveRelayState(state);
}

module.exports = {
  findRepositoryRoot,
  findRelayConfig,
  loadRelayConfig,
  saveRelayConfig,
  loadRelayState,
  saveRelayState,
  normalizeServerUrl,
  getServerSession,
  setServerSession,
  deleteServerSession,
  getWorkspaceVersion,
  setWorkspaceVersion,
  getWorkspaceLocalFingerprint,
  setWorkspaceLocalFingerprint,
};
