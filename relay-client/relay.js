#!/usr/bin/env node
// On Windows, switch the console to UTF-8 (code page 65001) before any output
// so that arrows (→), checkmarks (✓), and other Unicode symbols render correctly
// instead of showing garbled multi-byte sequences like â†' or âœ".
if (process.platform === "win32") {
  try {
    require("child_process").execSync("chcp 65001", { stdio: "pipe" });
  } catch (_) {}
}
/**
 * relay CLI  â€”  Node 18+
 *
 * Transport modes
 * â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
 * HTTP (default)   --url http://127.0.0.1:8080  --token TOKEN
 * Unix socket      --socket /path/to/relay.sock   (no token needed â€” socket ACL is auth)
 *                  env: RELAY_SOCKET=/path/to/relay.sock
 *
 * Commands
 * â”€â”€â”€â”€â”€â”€â”€â”€
 *   init            write .relay.json with defaults
 *   deploy          sync workspace and trigger a build + rollout
 *   status          show the latest deploy status for an app/env/branch
 *   logs <id>       stream or print logs for a deploy ID
 *   list            list recent deploys (optionally filtered by --app)
 *   projects        list projects and their environments
 *   rollback        roll the last deploy back to the previous image
 *   start/stop/restart   control a running app container
 *   secrets list    list secret keys for an app/env/branch
 *   secrets add     add or update a secret
 *   secrets rm      remove a secret
 *   plugin list     list installed buildpack plugins
 *   plugin install  install a buildpack plugin from a JSON file
 *   plugin remove   remove a buildpack plugin by name
 */

("use strict");

const fs = require("fs");
const fsp = require("fs/promises");
const http = require("http");
const https = require("https");
const readline = require("readline");
const path = require("path");
const os = require("os");
const crypto = require("crypto");
const { execSync, spawnSync } = require("child_process");
const {
  resolveDeployArgs,
  resolveServerArgs,
  resolveTransport,
} = require("./deploy");
const {
  loadRelayConfig,
  saveRelayConfig,
  getWorkspaceVersion,
  setWorkspaceVersion,
} = require("./config");

const CLI_VERSION = (() => {
  try {
    return String(require("./package.json").version || "dev");
  } catch {
    return "dev";
  }
})();

const RELEASES_API_LATEST =
  "https://api.github.com/repos/Relay-CI/Relay/releases/latest";

// â”€â”€â”€ ANSI colours (disabled when stdout is not a TTY) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

const tty = process.stdout.isTTY;
const c = {
  reset: tty ? "\x1b[0m" : "",
  bold: tty ? "\x1b[1m" : "",
  dim: tty ? "\x1b[2m" : "",
  green: tty ? "\x1b[32m" : "",
  yellow: tty ? "\x1b[33m" : "",
  red: tty ? "\x1b[31m" : "",
  cyan: tty ? "\x1b[36m" : "",
  white: tty ? "\x1b[97m" : "",
};

function ok(msg) {
  console.log(`${c.green}\u2713${c.reset} ${msg}`);
}
function warn(msg) {
  console.log(`${c.yellow}!${c.reset} ${msg}`);
}
function err(msg) {
  console.error(`${c.red}\u2717${c.reset} ${msg}`);
}
function info(msg) {
  console.log(`${c.dim}\u2192${c.reset} ${msg}`);
}

function die(msg) {
  err(msg);
  process.exit(1);
}

// â”€â”€â”€ Utilities â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

function nowMs() {
  return Date.now();
}

function formatDuration(ms) {
  if (!Number.isFinite(ms) || ms < 0) return "0ms";
  if (ms < 1000) return `${ms}ms`;
  const totalSeconds = Math.round(ms / 100) / 10;
  if (totalSeconds < 60) return `${totalSeconds}s`;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = Math.round((totalSeconds % 60) * 10) / 10;
  return `${minutes}m ${seconds}s`;
}

function isTerminalDeployStatus(status) {
  const s = String(status || "").toLowerCase();
  return s === "success" || s === "failed";
}

async function waitForTerminalDeployStatus(
  transport,
  deployId,
  timeoutMs = 180000,
  pollMs = 2000,
) {
  const started = nowMs();
  let last = null;
  while (nowMs() - started < timeoutMs) {
    last = await apiJSON(transport, "GET", `/api/deploys/${deployId}`);
    if (isTerminalDeployStatus(last?.status)) return last;
    await sleep(pollMs);
  }
  return last;
}

function statusColor(status) {
  if (!status) return c.dim;
  const s = String(status).toLowerCase();
  if (s === "success") return c.green;
  if (s === "failed" || s === "error") return c.red;
  if (s === "running" || s === "queued") return c.yellow;
  return c.dim;
}

function createSpinner(label) {
  const frames = ["|", "/", "-", "\\"];
  const startedAt = nowMs();
  let frame = 0;
  let timer = null;
  let lastLineLen = 0;

  const render = (text) => {
    const line = `${frames[frame]} ${text}`;
    const pad = Math.max(0, lastLineLen - line.length);
    process.stdout.write(`\r${line}${" ".repeat(pad)}`);
    lastLineLen = line.length;
    frame = (frame + 1) % frames.length;
  };

  return {
    start(extra = "") {
      render(extra ? `${label} ${extra}` : label);
      timer = setInterval(
        () => render(extra ? `${label} ${extra}` : label),
        120,
      );
    },
    update(extra = "") {
      render(extra ? `${label} ${extra}` : label);
    },
    stop(success, detail = "") {
      if (timer) clearInterval(timer);
      const elapsed = formatDuration(nowMs() - startedAt);
      const sym = success
        ? `${c.green}\u2713${c.reset}`
        : `${c.red}\u2717${c.reset}`;
      const line = `${sym} ${label}${detail ? ` ${c.dim}${detail}${c.reset}` : ""} ${c.dim}(${elapsed})${c.reset}`;
      const pad = Math.max(
        0,
        lastLineLen - line.replace(/\x1b\[[0-9;]*m/g, "").length,
      );
      process.stdout.write(`\r${line}${" ".repeat(pad)}\n`);
      lastLineLen = 0;
    },
  };
}

function parseArgs(argv) {
  const out = { _: [] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a.startsWith("--")) {
      const raw = a.slice(2);
      // Support both --key value and --key=value styles
      const eqIdx = raw.indexOf("=");
      if (eqIdx !== -1) {
        out[raw.slice(0, eqIdx)] = raw.slice(eqIdx + 1);
      } else {
        const v =
          argv[i + 1] && !argv[i + 1].startsWith("--") ? argv[++i] : "true";
        out[raw] = v;
      }
    } else {
      out._.push(a);
    }
  }
  return out;
}

function normalizeVersionTag(v) {
  return String(v || "")
    .trim()
    .replace(/^v/i, "");
}

function parseSemver(v) {
  const raw = normalizeVersionTag(v);
  const [core, pre = ""] = raw.split("-", 2);
  const [maj, min, pat] = core.split(".").map((x) => Number.parseInt(x, 10));
  if (![maj, min, pat].every(Number.isFinite)) return null;
  return { maj, min, pat, pre };
}

function compareVersions(a, b) {
  const va = parseSemver(a);
  const vb = parseSemver(b);
  if (!va || !vb) return 0;
  if (va.maj !== vb.maj) return va.maj - vb.maj;
  if (va.min !== vb.min) return va.min - vb.min;
  if (va.pat !== vb.pat) return va.pat - vb.pat;
  if (va.pre === vb.pre) return 0;
  if (!va.pre) return 1;
  if (!vb.pre) return -1;
  return va.pre.localeCompare(vb.pre);
}

function isOlderVersion(installed, latest) {
  return compareVersions(installed, latest) < 0;
}

function httpsGet(url) {
  return new Promise((resolve, reject) => {
    const follow = (u) => {
      https
        .get(
          u,
          {
            headers: {
              "User-Agent": "relay-cli",
              Accept: "application/vnd.github+json",
            },
          },
          (res) => {
            if (
              (res.statusCode === 301 || res.statusCode === 302) &&
              res.headers.location
            ) {
              follow(res.headers.location);
              return;
            }
            const chunks = [];
            res.on("data", (ch) => chunks.push(ch));
            res.on("end", () =>
              resolve({ status: res.statusCode, body: Buffer.concat(chunks) }),
            );
            res.on("error", reject);
          },
        )
        .on("error", reject);
    };
    follow(url);
  });
}

function httpsDownload(url, dest) {
  return new Promise((resolve, reject) => {
    const follow = (u) => {
      https
        .get(u, { headers: { "User-Agent": "relay-cli" } }, (res) => {
          if (
            (res.statusCode === 301 || res.statusCode === 302) &&
            res.headers.location
          ) {
            follow(res.headers.location);
            return;
          }
          if (res.statusCode !== 200) {
            reject(new Error(`HTTP ${res.statusCode}`));
            return;
          }
          const out = fs.createWriteStream(dest);
          res.pipe(out);
          out.on("finish", resolve);
          out.on("error", reject);
        })
        .on("error", reject);
    };
    follow(url);
  });
}

async function fetchLatestReleaseTag() {
  const { status, body } = await httpsGet(RELEASES_API_LATEST);
  if (status !== 200) throw new Error(`GitHub API returned HTTP ${status}`);
  let json;
  try {
    json = JSON.parse(body.toString("utf8"));
  } catch {
    throw new Error("Could not parse latest release from GitHub");
  }
  const tag = String(json.tag_name || "").trim();
  if (!tag) throw new Error("Latest release has no tag_name");
  return tag;
}

function readBinaryVersion(binPath) {
  if (!fs.existsSync(binPath)) return "";
  const out = spawnSync(binPath, ["--version"], { encoding: "utf8" });
  if (out.error || out.status !== 0) return "";
  return String(out.stdout || "").trim();
}

async function readJSONIfExists(p) {
  try {
    return JSON.parse(await fsp.readFile(p, "utf8"));
  } catch {
    return null;
  }
}

// ─── Interactive setup wizard ─────────────────────────────────────────────────

function prompt(question, defaultVal = "") {
  return new Promise((resolve) => {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    const hint = defaultVal ? ` ${c.dim}(${defaultVal})${c.reset}` : "";
    rl.question(`  ${question}${hint}: `, (answer) => {
      rl.close();
      resolve(answer.trim() || defaultVal);
    });
  });
}

function promptSecret(question) {
  return new Promise((resolve) => {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    // Mask input so the token is not echoed to the terminal.
    rl._writeToOutput = (str) => {
      if (str === "\n" || str === "\r\n" || str === "\r")
        process.stdout.write("\n");
    };
    rl.question(`  ${question}: `, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

function escapeHtml(value) {
  return String(value ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function renderCLILoginCallbackPage(user, role) {
  const safeUser = escapeHtml(user || "Unknown user");
  const safeRole = escapeHtml(role || "member");
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Relay CLI Login Complete</title>
    <style>
      :root {
        color-scheme: dark;
        --bg: #090b0e;
        --panel: rgba(17, 21, 27, 0.94);
        --line: rgba(207, 220, 234, 0.14);
        --text: #eef3f8;
        --muted: #95a3b3;
        --accent: #f28a41;
        --accent-soft: rgba(242, 138, 65, 0.16);
        --teal: #7ed8d3;
        --shadow: 0 30px 100px rgba(0, 0, 0, 0.45);
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        min-height: 100vh;
        display: grid;
        place-items: center;
        padding: 24px;
        font-family: "Segoe UI", Inter, system-ui, sans-serif;
        color: var(--text);
        background:
          radial-gradient(circle at 18% 14%, rgba(242, 138, 65, 0.16), transparent 26%),
          radial-gradient(circle at 82% 12%, rgba(126, 216, 211, 0.14), transparent 22%),
          linear-gradient(180deg, #090b0e 0%, #06080a 100%);
      }
      .card {
        width: min(560px, 100%);
        border-radius: 28px;
        padding: 28px;
        background:
          linear-gradient(155deg, rgba(242, 138, 65, 0.12), rgba(126, 216, 211, 0.08) 55%, rgba(255, 255, 255, 0.02)),
          var(--panel);
        border: 1px solid var(--line);
        box-shadow: var(--shadow);
        backdrop-filter: blur(18px);
      }
      .eyebrow {
        margin: 0 0 10px;
        text-transform: uppercase;
        letter-spacing: 0.24em;
        font-size: 0.72rem;
        color: var(--muted);
      }
      h1 {
        margin: 0;
        font-size: clamp(1.8rem, 4vw, 2.4rem);
        line-height: 1.05;
        letter-spacing: -0.04em;
      }
      p {
        margin: 14px 0 0;
        color: var(--muted);
        line-height: 1.65;
      }
      .status {
        display: inline-flex;
        align-items: center;
        gap: 10px;
        margin-top: 18px;
        padding: 10px 14px;
        border-radius: 999px;
        border: 1px solid rgba(126, 216, 211, 0.2);
        background: rgba(126, 216, 211, 0.08);
        color: var(--teal);
        font-weight: 600;
      }
      .dot {
        width: 10px;
        height: 10px;
        border-radius: 999px;
        background: currentColor;
        box-shadow: 0 0 14px currentColor;
      }
      .identity {
        margin-top: 22px;
        padding: 16px 18px;
        border-radius: 18px;
        border: 1px solid var(--line);
        background: rgba(255, 255, 255, 0.03);
      }
      .identity-label {
        text-transform: uppercase;
        letter-spacing: 0.16em;
        font-size: 0.72rem;
        color: var(--accent);
      }
      .identity-name {
        margin-top: 10px;
        font-size: 1.15rem;
        font-weight: 700;
      }
      .identity-role {
        margin-top: 6px;
        color: var(--muted);
        text-transform: capitalize;
      }
      .actions {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        margin-top: 24px;
      }
      .button {
        appearance: none;
        border: 0;
        border-radius: 14px;
        padding: 12px 16px;
        cursor: pointer;
        font: inherit;
      }
      .button--primary {
        background: linear-gradient(90deg, #c46a2d, #f28a41);
        color: #fff7f2;
        font-weight: 700;
      }
      .button--ghost {
        background: rgba(255, 255, 255, 0.04);
        color: var(--text);
        border: 1px solid var(--line);
      }
      .footer {
        margin-top: 18px;
        font-size: 0.88rem;
        color: var(--muted);
      }
      @media (max-width: 640px) {
        .card { padding: 22px; border-radius: 22px; }
        .actions { flex-direction: column; }
        .button { width: 100%; }
      }
    </style>
  </head>
  <body>
    <main class="card">
      <div class="eyebrow">Relay CLI</div>
      <h1>Login complete.</h1>
      <div class="status"><span class="dot"></span>Token handed off to your local Relay CLI session</div>
      <p>The browser authentication step is finished. Relay CLI should continue in your terminal now.</p>
      <section class="identity" aria-label="Signed in account">
        <div class="identity-label">Signed in as</div>
        <div class="identity-name">${safeUser}</div>
        <div class="identity-role">${safeRole}</div>
      </section>
      <div class="actions">
        <button class="button button--primary" type="button" onclick="window.close()">Close this tab</button>
        <button class="button button--ghost" type="button" onclick="location.href='about:blank'">Clear page</button>
      </div>
      <div class="footer">If the terminal does not continue, return to the CLI window and check for a timeout or connection error.</div>
    </main>
  </body>
</html>`;
}

async function runSetupWizard(args, cfgPath) {
  console.log(
    `\n${c.bold}Relay setup${c.reset}  ${c.dim}Answer a few questions to get connected.${c.reset}\n`,
  );
  console.log(
    `  ${c.dim}.relay.json stores CLI connection defaults. relay.json is for companion services only.${c.reset}\n`,
  );

  // ── 1. Connection type ──
  const existingCfg = loadRelayConfig(path.dirname(cfgPath)).data || {};

  const connRaw = await prompt(
    `Connection type  ${c.dim}[socket = local Unix socket,  http = remote / token]${c.reset}`,
    "http",
  );
  const useSocket = connRaw.toLowerCase().startsWith("s");

  let socket = null,
    url = null,
    token = null;

  if (useSocket) {
    socket = await prompt(
      "Socket path",
      args.socket ||
        process.env.RELAY_SOCKET ||
        path.join(".", "data", "relay.sock"),
    );
  } else {
    url = await prompt(
      "Server URL",
      args.url || process.env.RELAY_URL || existingCfg.url || "http://127.0.0.1:8080",
    );
    const existingToken = args.token || process.env.RELAY_TOKEN || existingCfg.token || "";
    // If an auth token already exists (e.g. from `relay login`), skip the
    // prompt — just reuse it. Only ask when there's genuinely nothing.
    if (existingToken) {
      token = existingToken;
      console.log(`  ${c.dim}Auth token: (using saved token)${c.reset}`);
    } else {
      token = await promptSecret("Auth token (hidden)");
    }
  }

  // ── 2. App details ──
  console.log("");
  const defaultApp = path
    .basename(process.cwd())
    .replace(/[^a-z0-9-]/gi, "-")
    .toLowerCase();
  const app = await prompt(
    "App name",
    args.app || process.env.RELAY_APP || existingCfg.app || defaultApp,
  );
  const env = await prompt(
    "Env     ",
    args.env || process.env.RELAY_ENV || existingCfg.env || "preview",
  );
  const branch = await prompt(
    "Branch  ",
    args.branch || process.env.RELAY_BRANCH || existingCfg.branch || "main",
  );

  // ── 3. Engine ──
  console.log("");
  const engineRaw = await prompt(
    `Engine   ${c.dim}[docker = recommended,  station = experimental runtime]${c.reset}`,
    args.engine || process.env.RELAY_ENGINE || existingCfg.engine || "docker",
  );
  const engine = engineRaw.toLowerCase().startsWith("d") ? "docker" : "station";

  // ── 4. Confirm save ──
  console.log("");
  const saveRaw = await prompt(`Save to ${path.basename(cfgPath)}?`, "yes");
  if (saveRaw.toLowerCase().startsWith("y")) {
    const cfg = {};
    if (socket) cfg.socket = socket;
    if (url) cfg.url = url;
    if (token) cfg.token = token;
    if (app) cfg.app = app;
    if (env) cfg.env = env;
    if (branch) cfg.branch = branch;
    cfg.engine = engine;
    // Merge with existing config so fields like `token` set by `relay login`
    // are never lost when the wizard is triggered for missing deploy fields.
    saveRelayConfig(cfg, path.dirname(cfgPath));
    ok(`Saved ${cfgPath}`);
  }
  console.log("");

  return { socket, url, token, app, env, branch, engine };
}

/**
 * Resolve transport (and optionally deploy args) from cli/env/config.
 * When resolution fails and stdin is a TTY, launch the interactive setup
 * wizard rather than crashing with a raw error message.
 */
async function resolveOrSetup(args, { needDeploy = false } = {}) {
  const cfgPath = path.join(process.cwd(), ".relay.json");
  try {
    const transport = resolveTransport(args);
    const resolved = needDeploy ? resolveDeployArgs(args) : null;
    return { transport, resolved };
  } catch {
    if (!process.stdin.isTTY) {
      // Non-interactive (CI/scripts): surface the error directly.
      try {
        resolveTransport(args);
      } catch (e) {
        die(e.message);
      }
      try {
        if (needDeploy) resolveDeployArgs(args);
      } catch (e) {
        die(e.message);
      }
    }
    const repoRelayPath = path.join(process.cwd(), "relay.json");
    if (fs.existsSync(repoRelayPath) && !fs.existsSync(cfgPath)) {
      console.log(
        `\n${c.dim}Found relay.json in this folder. That file defines companion services; CLI auth/defaults belong in .relay.json.${c.reset}`,
      );
    }
    const wiz = await runSetupWizard(args, cfgPath);
    // Patch args so the re-resolution picks up wizard answers as CLI overrides.
    if (wiz.socket) args.socket = wiz.socket;
    if (wiz.url) args.url = wiz.url;
    if (wiz.token) args.token = wiz.token;
    if (wiz.app) args.app = wiz.app;
    if (wiz.env) args.env = wiz.env;
    if (wiz.branch) args.branch = wiz.branch;
    if (wiz.engine) args.engine = wiz.engine;
    try {
      const transport = resolveTransport(args);
      const resolved = needDeploy ? resolveDeployArgs(args) : null;
      return { transport, resolved };
    } catch (e2) {
      die(e2.message);
    }
  }
}

// â”€â”€â”€ Transport layer â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
//
// A "transport" is one of:
//   { kind: "http",   baseUrl: "http://127.0.0.1:8080", token: "..." }
//   { kind: "socket", socketPath: "/run/relay.sock",    token: ""    }
//
// All request helpers below accept a transport and route accordingly.
// Unix-socket connections don't require a token because file-system permissions
// (mode 0600) already restrict access to the user running relayd.

function tokenHeader(token) {
  return token ? { "X-Relay-Token": token } : {};
}

/**
 * Make a raw request over a Unix socket or plain HTTP and return
 * { status, text }.  Does NOT throw on 4xx/5xx â€” callers decide.
 */
function rawRequest(
  transport,
  method,
  apiPath,
  extraHeaders = {},
  body = undefined,
) {
  if (transport.kind === "socket") {
    return new Promise((resolve, reject) => {
      const headers = {
        "Content-Type": "application/json",
        ...tokenHeader(transport.token),
        ...extraHeaders,
      };
      if (body !== undefined) {
        const encoded = typeof body === "string" ? body : JSON.stringify(body);
        headers["Content-Length"] = Buffer.byteLength(encoded);
      }
      const req = http.request(
        { socketPath: transport.socketPath, method, path: apiPath, headers },
        (res) => {
          const chunks = [];
          res.on("data", (d) => chunks.push(d));
          res.on("end", () =>
            resolve({
              status: res.statusCode,
              text: Buffer.concat(chunks).toString("utf8"),
            }),
          );
        },
      );
      req.on("error", reject);
      if (body !== undefined) {
        req.write(typeof body === "string" ? body : JSON.stringify(body));
      }
      req.end();
    });
  }

  // HTTP transport â€” use global fetch (Node 18+)
  const url = `${transport.baseUrl.replace(/\/$/, "")}${apiPath}`;
  const fetchOpts = {
    method,
    headers: {
      "Content-Type": "application/json",
      ...tokenHeader(transport.token),
      ...extraHeaders,
    },
  };
  if (body !== undefined)
    fetchOpts.body = typeof body === "string" ? body : JSON.stringify(body);
  return fetch(url, fetchOpts).then(async (res) => ({
    status: res.status,
    text: await res.text(),
  }));
}

async function apiJSON(transport, method, apiPath, body = undefined) {
  const { status, text } = await rawRequest(
    transport,
    method,
    apiPath,
    {},
    body,
  );
  let json;
  try {
    json = JSON.parse(text);
  } catch {
    json = { raw: text };
  }
  if (status >= 400) {
    const e = new Error(
      json && json.error ? json.error : `HTTP ${status}: ${text}`,
    );
    e.status = status;
    e.json = json;
    throw e;
  }
  return json;
}

/** PUT a file (raw binary) over the transport. */
function putFile(transport, apiPath, absPath) {
  if (transport.kind === "socket") {
    return new Promise((resolve, reject) => {
      const stat = fs.statSync(absPath);
      const headers = {
        ...tokenHeader(transport.token),
        "Content-Length": stat.size,
      };
      const req = http.request(
        {
          socketPath: transport.socketPath,
          method: "PUT",
          path: apiPath,
          headers,
        },
        (res) => {
          res.resume();
          res.on("end", () => {
            if (res.statusCode >= 400)
              reject(new Error(`HTTP ${res.statusCode}`));
            else resolve();
          });
        },
      );
      req.on("error", reject);
      fs.createReadStream(absPath).pipe(req);
    });
  }

  // HTTP â€” streaming PUT via fetch
  const url = `${transport.baseUrl.replace(/\/$/, "")}${apiPath}`;
  return fetch(url, {
    method: "PUT",
    headers: tokenHeader(transport.token),
    duplex: "half",
    body: fs.createReadStream(absPath),
  }).then(async (res) => {
    const txt = await res.text();
    if (!res.ok) {
      let j;
      try {
        j = JSON.parse(txt);
      } catch {}
      throw new Error(j && j.error ? j.error : `HTTP ${res.status}: ${txt}`);
    }
  });
}

/** Read an SSE stream via the transport. Returns final deploy status string. */
function streamLogsTransport(transport, deployId) {
  const apiPath = `/api/logs/stream/${deployId}`;

  function consumeSSE(incomingMsg) {
    return new Promise((resolve) => {
      let buf = "";
      let lastWasStatus = false;
      let lastLineLen = 0;
      let deployStatus = null;
      let resolved = false;
      let idleTimer = null;

      function finish() {
        if (resolved) return;
        resolved = true;
        if (idleTimer) clearTimeout(idleTimer);
        if (lastWasStatus) process.stdout.write("\n");
        resolve(deployStatus);
      }

      function resetIdleTimer() {
        if (idleTimer) clearTimeout(idleTimer);
        // Some deploys emit no lines after the build completes; don't block forever.
        idleTimer = setTimeout(() => {
          try {
            if (typeof incomingMsg.destroy === "function")
              incomingMsg.destroy();
          } catch {}
          finish();
        }, 45000);
      }

      resetIdleTimer();

      function processFrame(frame) {
        let eventName = "message";
        const dataLines = [];
        for (const line of frame.split("\n")) {
          if (line.startsWith("event: ")) {
            eventName = line.slice(7).trim();
            continue;
          }
          if (line.startsWith("data: ")) {
            dataLines.push(line.slice(6));
          }
        }
        if (!dataLines.length) return;
        let content = dataLines.join("\n");
        if (eventName === "deploy-status") {
          try {
            const j = JSON.parse(content);
            if (j?.status) deployStatus = j.status;
          } catch {
            deployStatus = content.trim();
          }
          return;
        }
        try {
          const j = JSON.parse(content);
          if (typeof j?.message === "string") content = j.message;
        } catch {}
        const text = content.replace(/\r/g, "");
        const isProgress =
          /(^progress|building|downloading|uploading|extracting|running|compiling|transpiling|installing|bundling)/i.test(
            text,
          ) ||
          text.includes("%") ||
          (text.length < 120 &&
            (text.endsWith("...") ||
              text.endsWith(".") ||
              text.endsWith("/") ||
              text.includes(" => ")));
        if (isProgress) {
          const pad = Math.max(0, lastLineLen - text.length);
          process.stdout.write(`\r${text}${" ".repeat(pad)}`);
          lastLineLen = text.length;
          lastWasStatus = true;
        } else {
          if (lastWasStatus) {
            process.stdout.write("\n");
            lastWasStatus = false;
            lastLineLen = 0;
          }
          console.log(text);
        }
      }

      incomingMsg.on("data", (chunk) => {
        if (resolved) return;
        resetIdleTimer();
        buf += chunk.toString("utf8");
        let idx;
        while ((idx = buf.indexOf("\n\n")) >= 0) {
          processFrame(buf.slice(0, idx));
          buf = buf.slice(idx + 2);
        }
      });
      incomingMsg.on("end", () => {
        finish();
      });
      incomingMsg.on("error", () => {
        finish();
      });
    });
  }

  if (transport.kind === "socket") {
    return new Promise((resolve, reject) => {
      const req = http.request(
        {
          socketPath: transport.socketPath,
          method: "GET",
          path: apiPath,
          headers: tokenHeader(transport.token),
        },
        (res) => {
          if (res.statusCode >= 400) {
            const chunks = [];
            res.on("data", (d) => chunks.push(d));
            res.on("end", () =>
              reject(
                new Error(
                  `log stream HTTP ${res.statusCode}: ${Buffer.concat(chunks)}`,
                ),
              ),
            );
            return;
          }
          consumeSSE(res).then(resolve, reject);
        },
      );
      req.on("error", reject);
      req.end();
    });
  }

  // HTTP transport
  const url = `${transport.baseUrl.replace(/\/$/, "")}${apiPath}`;
  return fetch(url, {
    method: "GET",
    headers: tokenHeader(transport.token),
  }).then((res) => {
    if (!res.ok)
      return res.text().then((t) => {
        throw new Error(`log stream HTTP ${res.status}: ${t}`);
      });
    // Convert web ReadableStream â†’ Node stream for uniform processing
    const { Readable } = require("stream");
    const nodeStream = Readable.fromWeb(res.body);
    return consumeSSE(nodeStream);
  });
}

// â”€â”€â”€ File scanning helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

function shouldIgnore(rel) {
  const top = rel.split("/")[0];
  const ignoreTop = new Set([
    "node_modules",
    ".git",
    ".next",
    "dist",
    ".turbo",
    "coverage",
    ".relay",
    "cache",
    "bin",
    "obj",
    "target",
  ]);
  if (ignoreTop.has(top)) return true;
  if (rel === ".relay.json" || rel === ".relayrc" || rel === ".relayrc.json")
    return true;
  return false;
}

async function walkFiles(rootDir) {
  const files = [];
  async function walk(dir) {
    const ents = await fsp.readdir(dir, { withFileTypes: true });
    for (const e of ents) {
      const abs = path.join(dir, e.name);
      const rel = path.relative(rootDir, abs).split(path.sep).join("/");
      if (rel === "" || shouldIgnore(rel)) {
        if (e.isDirectory()) continue;
        continue;
      }
      if (e.isDirectory()) await walk(abs);
      else if (e.isFile()) files.push({ abs, rel });
    }
  }
  await walk(rootDir);
  return files;
}

async function sha256File(absPath) {
  return new Promise((resolve, reject) => {
    const h = crypto.createHash("sha256");
    const s = fs.createReadStream(absPath);
    s.on("error", reject);
    s.on("data", (d) => h.update(d));
    s.on("end", () => resolve(h.digest("hex")));
  });
}

async function buildManifest(rootDir) {
  const list = await walkFiles(rootDir);
  const out = [];
  for (const f of list) {
    const st = await fsp.stat(f.abs);
    const sh = await sha256File(f.abs);
    out.push({
      Path: f.rel,
      Size: st.size,
      Mtime: st.mtimeMs
        ? Math.floor(st.mtimeMs)
        : Math.floor(st.mtime.getTime()),
      Sha256: sh,
      HashAlgo: "sha256",
      Hash: sh,
    });
  }
  return out;
}

// â”€â”€â”€ Usage / help â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

function printHelp() {
  console.log(`
${c.bold}relay${c.reset} â€” Relay deploy CLI

${c.bold}USAGE${c.reset}
  relay <command> [flags]

${c.bold}TRANSPORT FLAGS${c.reset} (global, apply to all commands)
  --socket <path>          Unix socket path  [env: RELAY_SOCKET]
  --url    <url>           HTTP base URL     [env: RELAY_URL,   default: http://127.0.0.1:8080]
  --token  <token>         Auth token        [env: RELAY_TOKEN] (not needed with --socket)

${c.bold}COMMANDS${c.reset}
  ${c.cyan}init${c.reset}                         Write .relay.json with current flag values
    --engine <docker|station>    Runtime engine (default: docker)
  ${c.cyan}deploy${c.reset}                       Sync workspace and build + roll out
    --app   --env  --branch  --dir
    --engine <docker|station>
    --mode  --host-port  --service-port  --public-host
    --install-cmd  --build-cmd  --start-cmd
    --stream                 Stream build logs until complete

  ${c.cyan}status${c.reset}                       Latest deploy for an app/env/branch
    --app  --env  --branch

  ${c.cyan}logs${c.reset} <deploy-id>             Print or stream logs for a deploy
    --no-stream              Print logs without streaming (fetch & exit)

  ${c.cyan}list${c.reset}                         List recent deploys
    --app                    Filter by app name
    --limit  <n>             Max rows (default 20)

  ${c.cyan}projects${c.reset}                     List projects and their environments

  ${c.cyan}rollback${c.reset}                     Roll back the last deploy
    --app  --env  --branch

  ${c.cyan}start${c.reset} / ${c.cyan}stop${c.reset} / ${c.cyan}restart${c.reset}       Control a running container
    --app  --env  --branch

  ${c.cyan}secrets list${c.reset}                 List secret keys
    --app  --env  --branch
  ${c.cyan}secrets add${c.reset}  --key K --value V   Add / update a secret
  ${c.cyan}secrets rm${c.reset}   --key K             Remove a secret

  ${c.cyan}plugin list${c.reset}                  List installed buildpack plugins
  ${c.cyan}plugin install${c.reset} <file.json>   Install a buildpack plugin
  ${c.cyan}plugin remove${c.reset}  <name>        Remove a buildpack plugin

  ${c.cyan}version${c.reset}                      Show relay, relayd, and station versions

  ${c.cyan}agent install${c.reset}                Download relayd + station binaries for this platform
    --version <v>            Pin a release version (default: latest)
    --dir     <path>         Install directory (default: ~/.relay/bin)
  ${c.cyan}agent update${c.reset}                 Update relayd + station to latest release
  ${c.cyan}agent status${c.reset}                 Show installed agent version and path
`);
}

// â”€â”€â”€ Main â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

async function main() {
  const args = parseArgs(process.argv.slice(2));
  let cmd = args._[0];

  // Allow positional env/branch shorthands for deploy-family commands:
  //   relay deploy dev           → --env dev
  //   relay deploy dev main      → --env dev --branch main
  //   relay status prod          → --env prod
  //   relay rollback prod main   → --env prod --branch main
  //   etc.
  const ENV_POSITIONAL_CMDS = new Set([
    "deploy", "status", "rollback", "start", "stop", "restart", "list", "pull",
  ]);
  if (ENV_POSITIONAL_CMDS.has(cmd)) {
    if (args._[1] && !args.env) args.env = args._[1];
    if (args._[2] && !args.branch) args.branch = args._[2];
  }

  if (!cmd && args.version === "true") {
    cmd = "version";
  }

  if (!cmd || args.help === "true" || args.h === "true") {
    printHelp();
    process.exit(0);
  }

  if (cmd === "version") {
    const installDir = path.resolve(
      args.dir ||
        process.env.RELAY_BIN_DIR ||
        path.join(os.homedir(), ".relay", "bin"),
    );
    const ext = process.platform === "win32" ? ".exe" : "";
    const vf = path.join(installDir, ".relay-version");

    let installed = "";
    try {
      installed = (await fsp.readFile(vf, "utf8")).trim();
    } catch {}

    let latest = "";
    try {
      latest = await fetchLatestReleaseTag();
    } catch (e) {
      warn(`Could not fetch latest release: ${e.message}`);
    }

    console.log(`\n${c.bold}Relay Versions${c.reset}\n`);
    console.log(`  relay CLI           ${c.green}${CLI_VERSION}${c.reset}`);
    console.log(
      `  relayd (installed)  ${installed ? `${c.green}${installed}${c.reset}` : `${c.yellow}not installed${c.reset}`}`,
    );
    if (latest) {
      console.log(`  relayd (latest)     ${c.cyan}${latest}${c.reset}`);
      if (installed && isOlderVersion(installed, latest)) {
        console.log(
          `  update status       ${c.yellow}outdated${c.reset}  ${c.dim}(run: relay agent update)${c.reset}`,
        );
      } else if (installed) {
        console.log(`  update status       ${c.green}up to date${c.reset}`);
      }
    }

    const relaydPath = path.join(installDir, `relayd${ext}`);
    const stationPath = path.join(installDir, `station${ext}`);
    const relaydBinaryVersion = readBinaryVersion(relaydPath);
    const stationBinaryVersion = readBinaryVersion(stationPath);
    if (relaydBinaryVersion)
      console.log(
        `  relayd binary       ${c.dim}${relaydBinaryVersion}${c.reset}`,
      );
    if (stationBinaryVersion)
      console.log(
        `  station binary      ${c.dim}${stationBinaryVersion}${c.reset}`,
      );

    try {
      const transport = resolveTransport(args);
      const serverVersion = await apiJSON(transport, "GET", "/api/version");
      if (serverVersion && serverVersion.version) {
        console.log(
          `  relayd (server)     ${c.green}${serverVersion.version}${c.reset}`,
        );
      }
      if (serverVersion && serverVersion.station_version) {
        console.log(
          `  station (server)    ${c.dim}${serverVersion.station_version}${c.reset}`,
        );
      }
    } catch {
      // Optional; relay version should still work without server connectivity.
    }

    console.log("");
    process.exit(0);
  }

  // â”€â”€ init â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "init") {
    const cfgPath = path.join(process.cwd(), ".relay.json");
    const hasFlags = args.url || args.socket || args.token || args.app;
    if (hasFlags) {
      // Non-interactive: save whatever flags were passed.
      const cfgOut = {};
      if (args.socket || process.env.RELAY_SOCKET)
        cfgOut.socket = args.socket || process.env.RELAY_SOCKET;
      if (args.url || process.env.RELAY_URL)
        cfgOut.url = args.url || process.env.RELAY_URL;
      if (args.token || process.env.RELAY_TOKEN)
        cfgOut.token = args.token || process.env.RELAY_TOKEN;
      if (args.app || process.env.RELAY_APP)
        cfgOut.app = args.app || process.env.RELAY_APP;
      if (args.env || process.env.RELAY_ENV)
        cfgOut.env = args.env || process.env.RELAY_ENV || "preview";
      if (args.branch || process.env.RELAY_BRANCH)
        cfgOut.branch = args.branch || process.env.RELAY_BRANCH || "main";
      if (args.engine || process.env.RELAY_ENGINE)
        cfgOut.engine = args.engine || process.env.RELAY_ENGINE;
      if (args.dir) cfgOut.dir = args.dir;
      saveRelayConfig(cfgOut, process.cwd());
      ok(`Wrote ${cfgPath}`);
    } else {
      // Interactive wizard.
      if (!process.stdin.isTTY)
        die(
          "No flags given and stdin is not a TTY. Pass --app, --url, --token etc.",
        );
      await runSetupWizard(args, cfgPath);
    }
    process.exit(0);
  }

  // â”€â”€ projects â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  // ── login ─────────────────────────────────────────────────────────────────────
  if (cmd === "login") {
    const existingCfg = loadRelayConfig(process.cwd()).data || {};
    const serverUrl = (
      args.url ||
      process.env.RELAY_URL ||
      existingCfg.url ||
      (await prompt("Server URL", "http://127.0.0.1:8080"))
    ).replace(/\/+$/, "");
    // Start the local callback server on a random port (port 0 lets the OS
    // pick a free one).  Using a single server avoids the TOCTOU race of the
    // old probe-close-rebind pattern.
    let authCodeResolve, authCodeReject;
    const authCodePromise = new Promise((res, rej) => {
      authCodeResolve = res;
      authCodeReject = rej;
    });
    const callbackSrv = http.createServer((req, cbRes) => {
      const u = new URL(req.url, `http://127.0.0.1`);
      const code = u.searchParams.get("code");
      const user = u.searchParams.get("user");
      const role = u.searchParams.get("role");
      cbRes.writeHead(200, { "Content-Type": "text/html" });
      cbRes.end(renderCLILoginCallbackPage(user, role));
      callbackSrv.close();
      if (code) authCodeResolve(code);
      else authCodeReject(new Error("no code in callback"));
    });
    callbackSrv.on("error", (srvErr) => {
      callbackSrv.close();
      authCodeReject(new Error(`callback server error: ${srvErr.message}`));
    });
    await new Promise((res, rej) =>
      callbackSrv.listen(0, "127.0.0.1", res).on("error", rej),
    );
    const callbackPort = callbackSrv.address().port;
    setTimeout(
      () => {
        callbackSrv.close();
        authCodeReject(new Error("login timed out"));
      },
      5 * 60 * 1000,
    );

    const loginUrl = `${serverUrl}/dashboard/?cli=1&port=${callbackPort}`;
    console.log(`\n  ${c.bold}Opening browser to log in…${c.reset}`);
    console.log(
      `  ${c.dim}If it doesn't open, visit:${c.reset}  ${c.cyan}${loginUrl}${c.reset}\n`,
    );
    try {
      const { spawn } = require("child_process");
      if (process.platform === "win32") {
        const child = spawn("cmd", ["/c", "start", "", loginUrl], {
          detached: true,
          stdio: "ignore",
          windowsHide: true,
        });
        child.unref();
      } else {
        const openCmd = process.platform === "darwin" ? "open" : "xdg-open";
        const child = spawn(openCmd, [loginUrl], {
          detached: true,
          stdio: "ignore",
        });
        child.unref();
      }
    } catch (_) {}
    const authCode = await authCodePromise.catch((e) => die(e.message));

    // Exchange one-time code for a bearer token.
    let tokenResp;
    try {
      tokenResp = await apiJSON(
        { kind: "http", baseUrl: serverUrl, token: "" },
        "POST",
        "/api/auth/cli/exchange",
        { code: authCode },
      );
    } catch (e) {
      die(`Token exchange failed: ${e.message}`);
    }
    if (!tokenResp || !tokenResp.token)
      die(`Login failed: ${tokenResp?.error || "no token returned"}`);

    const savedPath = saveRelayConfig({
      url: serverUrl,
      token: tokenResp.token,
    });
    ok(
      `Logged in as ${c.bold}${tokenResp.username}${c.reset} (${tokenResp.role})  →  saved to ${savedPath}`,
    );
    process.exit(0);
  }

  // ── logout ────────────────────────────────────────────────────────────────────
  if (cmd === "logout") {
    const { transport } = await resolveOrSetup(args);
    try {
      await apiJSON(transport, "DELETE", "/api/auth/session");
      ok("Logged out");
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  // ── pull ──────────────────────────────────────────────────────────────────────
  if (cmd === "pull") {
    const { transport, resolved } = await resolveOrSetup(args, {
      needDeploy: true,
    });
    const { app, env, branch, dir } = resolved;
    const rootDir = path.resolve(dir || ".");
    const baseUrl =
      transport.kind === "http" ? transport.baseUrl : "http://localhost";
    const pullSpinner = createSpinner("pulling workspace from server");
    pullSpinner.start();
    try {
      const pullPath = `/api/sync/pull/${encodeURIComponent(app)}/${encodeURIComponent(env)}/${encodeURIComponent(branch)}`;
      const wsVersion = await new Promise((resolve, reject) => {
        const parsed = new URL(pullPath, baseUrl);
        const mod = parsed.protocol === "https:" ? require("https") : http;
        const headers = transport.token
          ? { "X-Relay-Token": transport.token }
          : {};
        mod
          .get(
            {
              hostname: parsed.hostname,
              port: parsed.port,
              path: parsed.pathname,
              headers,
            },
            (res) => {
              const version = res.headers["x-workspace-version"] || "";
              const chunks = [];
              res.on("data", (chunk) => chunks.push(chunk));
              res.on("end", () => {
                const buf = Buffer.concat(chunks);
                let offset = 0,
                  fileCount = 0;
                const fsSync = require("fs");
                while (offset + 512 <= buf.length) {
                  const nameRaw = buf
                    .slice(offset, offset + 100)
                    .toString("utf8")
                    .replace(/\0/g, "");
                  if (!nameRaw) break;
                  const size =
                    parseInt(
                      buf
                        .slice(offset + 124, offset + 136)
                        .toString("utf8")
                        .trim(),
                      8,
                    ) || 0;
                  offset += 512;
                  if (size > 0) {
                    const absPath = path.join(rootDir, ...nameRaw.split("/"));
                    fsSync.mkdirSync(path.dirname(absPath), {
                      recursive: true,
                    });
                    fsSync.writeFileSync(
                      absPath,
                      buf.slice(offset, offset + size),
                    );
                    fileCount++;
                  }
                  offset += Math.ceil(size / 512) * 512;
                }
                pullSpinner.stop(
                  true,
                  `${fileCount} file${fileCount === 1 ? "" : "s"} pulled`,
                );
                resolve(version);
              });
            },
          )
          .on("error", (e) => {
            pullSpinner.stop(false);
            reject(e);
          });
      });
      if (wsVersion) setWorkspaceVersion(baseUrl, app, env, branch, wsVersion);
      ok(`Workspace up to date  (version ${wsVersion || "unknown"})`);
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  if (cmd === "projects") {
    const { transport } = await resolveOrSetup(args);
    try {
      const projects = await apiJSON(transport, "GET", "/api/projects");
      if (!projects.length) {
        info("No projects found");
        process.exit(0);
      }
      for (const p of projects) {
        console.log(`${c.bold}${p.name}${c.reset}`);
        for (const env of p.envs || []) {
          const status = env.latestDeploy?.status ?? "idle";
          const sc = statusColor(status);
          const location = env.stopped
            ? "offline"
            : env.public_host || env.host_port
              ? env.public_host || `port:${env.host_port}`
              : "no url";
          console.log(
            `  ${c.dim}${env.env}/${env.branch}${c.reset}  ${sc}${status}${c.reset}  ${c.dim}${location}${c.reset}`,
          );
        }
      }
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  // â”€â”€ list â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "list") {
    const { transport } = await resolveOrSetup(args);
    try {
      const deploys = await apiJSON(transport, "GET", "/api/deploys");
      const limit = Number(args.limit) || 20;
      let rows = Array.isArray(deploys) ? deploys : [];
      if (args.app) rows = rows.filter((d) => d.app === args.app);
      if (args.env) rows = rows.filter((d) => d.env === args.env);
      if (args.branch) rows = rows.filter((d) => d.branch === args.branch);
      rows = rows.slice(0, limit);
      if (!rows.length) {
        info("No deploys found");
        process.exit(0);
      }
      const idW = 10,
        appW = 18,
        envW = 8,
        brW = 16,
        stW = 10;
      const hdr = `${"ID".padEnd(idW)}  ${"APP".padEnd(appW)}  ${"ENV".padEnd(envW)}  ${"BRANCH".padEnd(brW)}  ${"STATUS".padEnd(stW)}  CREATED`;
      console.log(`${c.dim}${hdr}${c.reset}`);
      for (const d of rows) {
        const sc = statusColor(d.status);
        const created = d.created_at
          ? new Date(d.created_at).toLocaleString()
          : "";
        console.log(
          `${d.id.slice(0, 8).padEnd(idW)}  ${(d.app || "").padEnd(appW)}  ${(d.env || "").padEnd(envW)}  ${(d.branch || "").padEnd(brW)}  ${sc}${(d.status || "").padEnd(stW)}${c.reset}  ${c.dim}${created}${c.reset}`,
        );
      }
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  // â”€â”€ status â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "status") {
    const { transport, resolved } = await resolveOrSetup(args, {
      needDeploy: true,
    });
    try {
      const deploys = await apiJSON(transport, "GET", "/api/deploys");
      const match = deploys.find(
        (d) =>
          d.app === resolved.app &&
          d.env === resolved.env &&
          d.branch === resolved.branch,
      );
      if (!match) {
        warn(
          `No deploys found for ${resolved.app}/${resolved.env}/${resolved.branch}`,
        );
        process.exit(0);
      }
      const appState = await apiJSON(
        transport,
        "GET",
        `/api/apps/config?app=${encodeURIComponent(resolved.app)}&env=${encodeURIComponent(resolved.env)}&branch=${encodeURIComponent(resolved.branch)}`,
      );
      const appStopped = Boolean(appState?.Stopped ?? appState?.stopped);
      const sc = statusColor(match.status);
      console.log(
        `${c.bold}${match.app}${c.reset}  ${match.env}/${match.branch}`,
      );
      console.log(`  ID       ${c.dim}${match.id}${c.reset}`);
      console.log(`  Status   ${sc}${match.status}${c.reset}`);
      console.log(
        `  App      ${appStopped ? `${c.yellow}offline${c.reset}  ${c.dim}(image ready, start manually)${c.reset}` : `${c.green}online${c.reset}`}`,
      );
      if (match.created_at)
        console.log(
          `  Created  ${c.dim}${new Date(match.created_at).toLocaleString()}${c.reset}`,
        );
      if (match.started_at)
        console.log(
          `  Started  ${c.dim}${new Date(match.started_at).toLocaleString()}${c.reset}`,
        );
      if (match.ended_at)
        console.log(
          `  Ended    ${c.dim}${new Date(match.ended_at).toLocaleString()}${c.reset}`,
        );
      if (match.preview_url)
        console.log(`  URL      ${c.cyan}${match.preview_url}${c.reset}`);
      if (match.error)
        console.log(`  Error    ${c.red}${match.error}${c.reset}`);
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  // â”€â”€ logs â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "logs") {
    const deployId = args._[1];
    if (!deployId) die("Usage: relay logs <deploy-id> [--no-stream]");
    const { transport } = await resolveOrSetup(args);
    try {
      if (!(args["no-stream"] === "true" || args["no-stream"] === true)) {
        info(`Streaming logs for ${deployId}`);
        const status = await streamLogsTransport(transport, deployId);
        const sc = statusColor(status);
        console.log(`\n${sc}${status || "done"}${c.reset}`);
      } else {
        const text = await apiJSON(transport, "GET", `/api/logs/${deployId}`);
        process.stdout.write(
          typeof text === "string" ? text : JSON.stringify(text, null, 2),
        );
      }
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  // â”€â”€ rollback â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "rollback") {
    const { transport, resolved } = await resolveOrSetup(args, {
      needDeploy: true,
    });
    try {
      const res = await apiJSON(transport, "POST", "/api/deploys/rollback", {
        app: resolved.app,
        env: resolved.env,
        branch: resolved.branch,
      });
      ok(`Rollback queued  id=${res.id}`);
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  // â”€â”€ start / stop / restart â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "stop" || cmd === "start" || cmd === "restart") {
    const { transport, resolved } = await resolveOrSetup(args, {
      needDeploy: true,
    });
    try {
      await apiJSON(transport, "POST", `/api/apps/${cmd}`, {
        app: resolved.app,
        env: resolved.env,
        branch: resolved.branch,
      });
      ok(`${cmd} sent for ${resolved.app}/${resolved.env}/${resolved.branch}`);
    } catch (e) {
      die(e.message);
    }
    process.exit(0);
  }

  // â”€â”€ secrets â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "secrets") {
    const sub = args._[1];
    const { transport, resolved } = await resolveOrSetup(args, {
      needDeploy: true,
    });
    const qs = `?app=${encodeURIComponent(resolved.app)}&env=${encodeURIComponent(resolved.env)}&branch=${encodeURIComponent(resolved.branch)}`;

    if (!sub || sub === "list") {
      try {
        const secrets = await apiJSON(
          transport,
          "GET",
          `/api/apps/secrets${qs}`,
        );
        if (!secrets.length) {
          info("No secrets configured");
          process.exit(0);
        }
        for (const s of secrets) console.log(`  ${c.cyan}${s.key}${c.reset}`);
      } catch (e) {
        die(e.message);
      }
      process.exit(0);
    }

    if (sub === "add") {
      const key = args.key;
      const value = args.value;
      if (!key || !value)
        die(
          "Usage: relay secrets add --key KEY --value VALUE --app ... --env ... --branch ...",
        );
      try {
        await apiJSON(transport, "POST", "/api/apps/secrets", {
          app: resolved.app,
          env: resolved.env,
          branch: resolved.branch,
          key,
          value,
        });
        ok(`Secret ${key} saved`);
      } catch (e) {
        die(e.message);
      }
      process.exit(0);
    }

    if (sub === "rm" || sub === "remove") {
      const key = args.key || args._[2];
      if (!key)
        die(
          "Usage: relay secrets rm --key KEY --app ... --env ... --branch ...",
        );
      try {
        await apiJSON(
          transport,
          "DELETE",
          `/api/apps/secrets${qs}&key=${encodeURIComponent(key)}`,
        );
        ok(`Secret ${key} removed`);
      } catch (e) {
        die(e.message);
      }
      process.exit(0);
    }

    die(`Unknown secrets sub-command: ${sub}. Supported: list, add, rm`);
  }

  // â”€â”€ plugin â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd === "plugin") {
    const sub = args._[1];
    const { transport } = await resolveOrSetup(args);

    if (sub === "list") {
      try {
        const plugins = await apiJSON(
          transport,
          "GET",
          "/api/plugins/buildpacks",
        );
        if (!Array.isArray(plugins) || !plugins.length) {
          info("No buildpack plugins installed");
          process.exit(0);
        }
        for (const p of plugins) {
          const kind = p.plan?.kind
            ? ` ${c.dim}(${p.plan.kind})${c.reset}`
            : "";
          const desc = p.description
            ? `  ${c.dim}${p.description}${c.reset}`
            : "";
          console.log(`  ${c.cyan}${p.name}${c.reset}${kind}${desc}`);
        }
      } catch (e) {
        die(e.message);
      }
      process.exit(0);
    }

    if (sub === "install") {
      const ref = args.file || args._[2];
      if (!ref) die("Usage: relay plugin install <plugin.json>");
      let plugin;
      try {
        plugin = JSON.parse(await fsp.readFile(path.resolve(ref), "utf8"));
      } catch (e) {
        die(`Failed to read plugin file: ${e.message}`);
      }
      try {
        const res = await apiJSON(
          transport,
          "POST",
          "/api/plugins/buildpacks",
          plugin,
        );
        ok(`Installed buildpack plugin: ${res.name}`);
      } catch (e) {
        die(e.message);
      }
      process.exit(0);
    }

    if (sub === "remove" || sub === "uninstall") {
      const name = args._[2];
      if (!name) die("Usage: relay plugin remove <name>");
      try {
        await apiJSON(
          transport,
          "DELETE",
          `/api/plugins/buildpacks/${encodeURIComponent(name)}`,
        );
        ok(`Removed buildpack plugin: ${name}`);
      } catch (e) {
        die(e.message);
      }
      process.exit(0);
    }

    die(
      "Unknown plugin sub-command. Supported: list, install <file>, remove <name>",
    );
  }
  // ── agent ──────────────────────────────────────────────────────────────────
  if (cmd === "agent") {
    const sub = args._[1];

    const installDir = path.resolve(
      args.dir ||
        process.env.RELAY_BIN_DIR ||
        path.join(os.homedir(), ".relay", "bin"),
    );

    function platformAsset() {
      const p = process.platform;
      const a = process.arch;
      if (p === "linux" && a === "x64")
        return { tarball: "relay-linux-amd64.tar.gz", zip: false };
      if (p === "linux" && a === "arm64")
        return { tarball: "relay-linux-arm64.tar.gz", zip: false };
      if (p === "win32" && a === "x64")
        return { tarball: "relay-windows-amd64.zip", zip: true };
      die(
        `No prebuilt binary for ${p}/${a}. Build from source: https://github.com/Relay-CI/Relay`,
      );
    }

    async function installAgentRelease(version) {
      const asset = platformAsset();
      const downloadUrl = `https://github.com/Relay-CI/Relay/releases/download/${version}/${asset.tarball}`;
      const tmpFile = path.join(os.tmpdir(), asset.tarball);

      console.log(`\n${c.bold}Installing Relay agent ${version}${c.reset}\n`);
      info(`Platform : ${process.platform}/${process.arch}`);
      info(`Into     : ${installDir}`);
      console.log("");

      await fsp.mkdir(installDir, { recursive: true });

      const dlSpinner = createSpinner(`Downloading ${asset.tarball}`);
      dlSpinner.start();
      try {
        await httpsDownload(downloadUrl, tmpFile);
      } catch (e) {
        dlSpinner.stop(false, e.message);
        die(`Download failed - does release ${version} exist on GitHub?`);
      }
      dlSpinner.stop(true);

      const exSpinner = createSpinner("Extracting");
      exSpinner.start();
      try {
        if (asset.zip) {
          const dest = installDir.replace(/'/g, "''");
          const tmp = tmpFile.replace(/'/g, "''");
          execSync(
            `powershell -NoProfile -Command "Expand-Archive -Force -Path '${tmp}' -DestinationPath '${dest}'"`,
            { stdio: "pipe" },
          );
        } else {
          execSync(`tar xzf "${tmpFile}" -C "${installDir}"`, {
            stdio: "pipe",
          });
        }
        if (process.platform !== "win32") {
          for (const bin of ["relayd", "station"]) {
            const bp = path.join(installDir, bin);
            if (fs.existsSync(bp)) fs.chmodSync(bp, 0o755);
          }
        }
      } catch (e) {
        exSpinner.stop(false);
        die(`Extraction failed: ${e.message}`);
      }
      exSpinner.stop(true);

      const ext = process.platform === "win32" ? ".exe" : "";
      await fsp.writeFile(
        path.join(installDir, ".relay-version"),
        version,
        "utf8",
      );

      console.log("");
      ok(
        `relayd   ${c.dim}\u2192${c.reset} ${path.join(installDir, "relayd" + ext)}`,
      );
      ok(
        `station  ${c.dim}\u2192${c.reset} ${path.join(installDir, "station" + ext)}`,
      );

      console.log(`\n${c.bold}Add to PATH then start the agent:${c.reset}`);
      if (process.platform === "win32") {
        console.log(
          `  ${c.cyan}[Environment]::SetEnvironmentVariable("Path", $env:Path + ";${installDir}", "User")${c.reset}`,
        );
        console.log(`  ${c.cyan}relayd.exe${c.reset}`);
        console.log(
          `  ${c.dim}# if 8080 is busy: relayd.exe --port 9090${c.reset}`,
        );
      } else {
        const rc = process.env.SHELL?.includes("zsh")
          ? "~/.zshrc"
          : "~/.bashrc";
        console.log(
          `  ${c.cyan}echo 'export PATH="${installDir}:$PATH"' >> ${rc} && source ${rc}${c.reset}`,
        );
        console.log(`  ${c.cyan}relayd${c.reset}`);
        console.log(
          `  ${c.dim}# if 8080 is busy: relayd --port 9090${c.reset}`,
        );
      }
      console.log(
        `\n  Then in your project:  ${c.cyan}relay init${c.reset}  and  ${c.cyan}relay deploy --stream${c.reset}\n`,
      );
    }

    if (!sub || sub === "install") {
      let version = args.version || args.v;
      if (!version) {
        info("Fetching latest release...");
        try {
          version = await fetchLatestReleaseTag();
        } catch (e) {
          die(`Could not determine latest release: ${e.message}`);
        }
      }
      if (!version) die("Could not determine version. Pass --version v0.x.x");

      await installAgentRelease(version);
      process.exit(0);
    }

    if (sub === "update") {
      const vf = path.join(installDir, ".relay-version");
      let installed = "";
      try {
        installed = (await fsp.readFile(vf, "utf8")).trim();
      } catch {}

      let latest;
      try {
        latest = await fetchLatestReleaseTag();
      } catch (e) {
        die(`Could not check latest release: ${e.message}`);
      }

      if (installed && !isOlderVersion(installed, latest)) {
        ok(`Already up to date (${installed})`);
        process.exit(0);
      }

      if (installed) {
        info(`Updating ${installed} -> ${latest}`);
      } else {
        info(`Installing latest release ${latest}`);
      }
      await installAgentRelease(latest);
      process.exit(0);
    }

    if (sub === "status") {
      const vf = path.join(installDir, ".relay-version");
      const ext = process.platform === "win32" ? ".exe" : "";
      let installed = null;
      try {
        installed = (await fsp.readFile(vf, "utf8")).trim();
      } catch {}

      let latest = "";
      try {
        latest = await fetchLatestReleaseTag();
      } catch (e) {
        warn(`Could not fetch latest release: ${e.message}`);
      }

      const relaydPath = path.join(installDir, `relayd${ext}`);
      const stationPath = path.join(installDir, `station${ext}`);
      const relaydBinaryVersion = readBinaryVersion(relaydPath);
      const stationBinaryVersion = readBinaryVersion(stationPath);

      console.log(`\n${c.bold}Relay Agent${c.reset}\n`);
      console.log(`  Dir              ${c.dim}${installDir}${c.reset}`);
      console.log(
        `  Installed        ${installed ? `${c.green}${installed}${c.reset}` : `${c.yellow}not installed${c.reset}`}`,
      );
      if (latest) {
        console.log(`  Latest           ${c.cyan}${latest}${c.reset}`);
        if (installed && isOlderVersion(installed, latest)) {
          console.log(
            `  Update status    ${c.yellow}outdated${c.reset}  ${c.dim}(run: relay agent update)${c.reset}`,
          );
        } else if (installed) {
          console.log(`  Update status    ${c.green}up to date${c.reset}`);
        }
      }
      console.log(
        `  relayd binary    ${fs.existsSync(relaydPath) ? `${c.green}\u2713${c.reset}` : `${c.red}\u2717${c.reset}`}`,
      );
      if (relaydBinaryVersion) {
        console.log(
          `  relayd version   ${c.dim}${relaydBinaryVersion}${c.reset}`,
        );
      }
      console.log(
        `  station binary   ${fs.existsSync(stationPath) ? `${c.green}\u2713${c.reset}` : `${c.red}\u2717${c.reset}`}`,
      );
      if (stationBinaryVersion) {
        console.log(
          `  station version  ${c.dim}${stationBinaryVersion}${c.reset}`,
        );
      }
      console.log("");
      process.exit(0);
    }

    die(
      `Unknown agent sub-command: ${sub}. Supported: install, update, status`,
    );
  }

  // â”€â”€ deploy â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
  if (cmd !== "deploy") {
    err(`Unknown command: ${c.bold}${cmd}${c.reset}`);
    printHelp();
    process.exit(1);
  }

  const { transport, resolved } = await resolveOrSetup(args, {
    needDeploy: true,
  });

  const app = resolved.app;
  const env = resolved.env;
  const branch = resolved.branch;
  const dir = resolved.dir || ".";
  const hasModeOverride = Object.prototype.hasOwnProperty.call(args, "mode");
  const hasHostPortOverride = Object.prototype.hasOwnProperty.call(
    args,
    "host-port",
  );
  const hasServicePortOverride = Object.prototype.hasOwnProperty.call(
    args,
    "service-port",
  );
  const hasPublicHostOverride = Object.prototype.hasOwnProperty.call(
    args,
    "public-host",
  );
  const mode = hasModeOverride ? String(args["mode"]).trim() : "";
  const hostPort = hasHostPortOverride ? Number(args["host-port"]) || 0 : 0;
  const servicePort = hasServicePortOverride
    ? Number(args["service-port"]) || 0
    : 0;
  const publicHost = hasPublicHostOverride
    ? String(args["public-host"]).trim()
    : "";

  const rootDir = path.resolve(dir);
  const cfg = await readJSONIfExists(path.join(rootDir, "relay.config.json"));
  const localDefaults = await readJSONIfExists(
    path.join(rootDir, ".relay.json"),
  );

  const installCmd =
    args["install-cmd"] || localDefaults?.install_cmd || cfg?.install_cmd || "";
  const buildCmd =
    args["build-cmd"] || localDefaults?.build_cmd || cfg?.build_cmd || "";
  const startCmd =
    args["start-cmd"] || localDefaults?.start_cmd || cfg?.start_cmd || "";
  const engine = args.engine || localDefaults?.engine || cfg?.engine || "";

  const via =
    transport.kind === "socket"
      ? `socket:${transport.socketPath}`
      : transport.baseUrl;
  info(
    `deploying ${c.bold}${app}${c.reset}  env=${env} branch=${branch}  via ${c.dim}${via}${c.reset}`,
  );

  const totalStartedAt = nowMs();

  // Read the locally-saved workspace version so the server can detect
  // whether someone else deployed since our last sync.
  const baseUrl = transport.kind === "http" ? transport.baseUrl : "";
  const baseVersion = args.force
    ? ""
    : getWorkspaceVersion(baseUrl, app, env, branch);

  const syncStartSpinner = createSpinner(`sync start`);
  syncStartSpinner.start();
  let startResp;
  try {
    startResp = await apiJSON(transport, "POST", "/api/sync/start", {
      app,
      branch,
      env,
      base_version: baseVersion,
    });
  } catch (e) {
    syncStartSpinner.stop(false);
    // 409 = someone else deployed since our last pull
    if (
      e.status === 409 ||
      (e.message && e.message.includes("workspace behind"))
    ) {
      err(`Your workspace is behind the server.`);
      err(`Someone else deployed after your last sync.`);
      err(
        `Run:  ${c.bold}relay pull${c.reset}  to pull the latest files, then re-deploy.`,
      );
      err(`      (or use ${c.dim}--force${c.reset} to overwrite anyway)`);
      process.exit(1);
    }
    die(e.message);
  }
  syncStartSpinner.stop(true, `session=${startResp.session_id?.slice(0, 8)}`);

  const sessionId = startResp.session_id;
  if (!sessionId) die("sync start failed: no session_id returned");

  const manifestSpinner = createSpinner("hashing workspace");
  manifestSpinner.start();
  const manifest = await buildManifest(rootDir);
  manifestSpinner.stop(true, `files=${manifest.length}`);

  const planSpinner = createSpinner("diffing against agent workspace");
  planSpinner.start();
  const plan = await apiJSON(transport, "POST", `/api/sync/plan/${sessionId}`, {
    files: manifest,
  });
  planSpinner.stop(
    true,
    `need=${(plan.need || []).length} delete=${(plan.delete || []).length}`,
  );

  const need = plan.need || [];
  const del = plan.delete || [];

  const uploadStartedAt = nowMs();
  if (need.length) {
    info(`upload ${need.length} changed file${need.length === 1 ? "" : "s"}`);
    for (let i = 0; i < need.length; i++) {
      const rel = need[i];
      const abs = path.join(rootDir, rel.split("/").join(path.sep));
      process.stdout.write(
        `   [${i + 1}/${need.length}] ${c.dim}${rel}${c.reset}\n`,
      );
      const uploadPath = `/api/sync/upload/${sessionId}?path=${encodeURIComponent(rel)}`;
      await putFile(transport, uploadPath, abs);
    }
    ok(
      `uploaded ${need.length} file${need.length === 1 ? "" : "s"} in ${formatDuration(nowMs() - uploadStartedAt)}`,
    );
  }

  if (del.length) {
    info(`delete ${del.length} stale path${del.length === 1 ? "" : "s"}`);
    await apiJSON(transport, "POST", `/api/sync/delete/${sessionId}`, {
      paths: del,
    });
  }

  const finishSpinner = createSpinner("triggering build and rollout");
  finishSpinner.start();
  const deployPayload = {
    source: "sync",
    install_cmd: installCmd,
    build_cmd: buildCmd,
    start_cmd: startCmd,
  };
  if (engine) deployPayload.engine = engine;
  if (hasModeOverride) deployPayload.mode = mode;
  if (hasHostPortOverride) deployPayload.host_port = hostPort;
  if (hasServicePortOverride) deployPayload.service_port = servicePort;
  if (hasPublicHostOverride) deployPayload.public_host = publicHost;
  const deploy = await apiJSON(
    transport,
    "POST",
    `/api/sync/finish/${sessionId}`,
    deployPayload,
  );
  finishSpinner.stop(true, `id=${deploy.id}`);

  // Persist the new workspace version so the next deploy can detect staleness.
  if (deploy.workspace_version) {
    setWorkspaceVersion(baseUrl, app, env, branch, deploy.workspace_version);
  }

  const noStream = (args["no-stream"] === "true" || args["no-stream"] === true) && args["stream"] !== "true" && args["stream"] !== true;
  if (!noStream) {
    info("streaming logs");
    const streamStartedAt = nowMs();
    const streamedStatus = await streamLogsTransport(transport, deploy.id);
    let finalDeploy = await apiJSON(
      transport,
      "GET",
      `/api/deploys/${deploy.id}`,
    );
    if (!isTerminalDeployStatus(finalDeploy?.status)) {
      finalDeploy = await waitForTerminalDeployStatus(transport, deploy.id);
    }
    const appState = await apiJSON(
      transport,
      "GET",
      `/api/apps/config?app=${encodeURIComponent(app)}&env=${encodeURIComponent(env)}&branch=${encodeURIComponent(branch)}`,
    );
    const appStopped = Boolean(appState?.Stopped ?? appState?.stopped);
    const totalElapsed = formatDuration(nowMs() - totalStartedAt);
    const streamElapsed = formatDuration(nowMs() - streamStartedAt);
    const finalStatus = streamedStatus || finalDeploy?.status || "unknown";
    const duration =
      finalDeploy.started_at && finalDeploy.ended_at
        ? formatDuration(
            new Date(finalDeploy.ended_at) - new Date(finalDeploy.started_at),
          )
        : totalElapsed;
    const previewURL =
      finalDeploy.preview_url ||
      (hostPort ? `http://127.0.0.1:${hostPort}` : "");

    if (String(finalStatus).toLowerCase() === "success") {
      if (appStopped) {
        ok(
          `deploy built in ${c.bold}${duration}${c.reset} and kept offline  ${c.dim}(stream ${streamElapsed}, total ${totalElapsed})${c.reset}`,
        );
        if (finalDeploy.image_tag)
          console.log(`  image    ${c.dim}${finalDeploy.image_tag}${c.reset}`);
        console.log(
          `  state    ${c.yellow}offline${c.reset}  ${c.dim}ready for 'relay start' when you want it live${c.reset}`,
        );
      } else {
        ok(
          `deploy ready in ${c.bold}${duration}${c.reset}  ${c.dim}(stream ${streamElapsed}, total ${totalElapsed})${c.reset}`,
        );
        if (previewURL) console.log(`  ${c.cyan}${previewURL}${c.reset}`);
      }
    } else {
      const sc = statusColor(finalStatus);
      warn(
        `deploy finished with status ${sc}${finalStatus}${c.reset} after ${totalElapsed}`,
      );
      process.exitCode = 1;
    }
  } else {
    info(`logs: relay logs ${deploy.id}`);
  }
}

main().catch((e) => die(e.message));
