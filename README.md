# Relay

Self-hosted deployment platform — sync changed files, auto-detect buildpacks, roll out containers. No GitHub Actions, no cloud platform required.

| Component     | What it does                                                                 |
| ------------- | ---------------------------------------------------------------------------- |
| **`relay`**   | Node.js CLI — deploy from any machine                                        |
| **`relayd`**  | Go agent — runs on your server, builds images, manages containers            |
| **`station`** | Experimental Go runtime — keep this secondary until the stable release lands |

---

## Install

### CLI

```bash
npm install -g @relay-org/relay
```

### Agent binaries (on your server)

```bash
relay agent install
```

Auto-detects your platform (Linux amd64/arm64, Windows) and downloads `relayd` plus the optional `station` runtime from the latest GitHub Release into `~/.relay/bin/`. No Go required.

Add `~/.relay/bin` to your PATH — the command prints the exact line for your shell.

---

## Quick Start

**1. Start the agent on your server**

```bash
relayd
```

For Linux production (keeps running after SSH closes and restarts on reboot):

```bash
sudo relayd service install --user relay --group relay --data-dir /var/lib/relayd
```

On first run with no user accounts configured, open `http://<server>:8080` — the dashboard walks you through creating the first owner account. If you prefer a static token instead, set `RELAY_TOKEN` and the browser will prompt for it.

**2. Init your project** (run inside your app folder)

```bash
relay init
```

Walks through server URL, token or login, app name, env, and branch. Writes `.relay.json`.

**3. Deploy**

```bash
relay deploy --stream
```

Relay detects your buildpack, syncs only changed files, builds a container, and does a zero-downtime rollout. Each deploy gets a sequential build number (`#1`, `#2`, …) visible in the dashboard.

---

## MCP Server

Use Relay from MCP-aware AI tools (Cursor, Claude Desktop, VS Code Copilot):

```bash
npx -y @relay-org/relay-mcp
```

Config examples and tool list:

- [relay-mcp README](relay-mcp/README.md)

---

## Commands

```
relay init                         Interactive setup → writes .relay.json
relay deploy [--stream]            Sync + build + rollout
relay pull                         Download server workspace to local directory
relay status                       Latest deploy status
relay logs <id>                    Stream build logs
relay list                         Recent deploys
relay projects                     All projects and environments
relay rollback                     Roll back to previous image
relay start / stop / restart       Control a running container
relay secrets list/add/rm          Manage app secrets
relay login                        Browser-based login → saves bearer token
relay logout                       Clear saved session token
relay plugin list/search           Inspect the server-side plugin catalog
relay plugin install/remove        Install or remove local plugin JSON
relay plugin install-url           Install a remote plugin over HTTPS with optional SHA256 pin
relay version                      Show relay/relayd/station versions
relay doctor                       Check agent connectivity, Docker, DNS, TLS, and socket state
relay agent install [--version v]  Download relayd + station binaries
relay agent update                 Update relayd + station to latest release
relay agent status                 Show installed/latest versions and outdated status
```

---

## Buildpacks

Relay auto-detects your framework:

`node-next` · `node-vite` · `expo-web` · `sprint-ui` · `bun` · `node-generic` · `go` · `dotnet` · `python` (Django/Flask/FastAPI) · `ruby` (Rails/Rack) · `java` (Maven/Gradle, Spring Boot-friendly) · `rust` · `c-cpp` · `wasm-static` · `static`

Add more without rebuilding `relayd` — see [Server-Side Buildpack Plugins](#server-side-buildpack-plugins).

---

## Config

| File                | Purpose                                                     |
| ------------------- | ----------------------------------------------------------- |
| `.relay.json`       | CLI connection defaults — url, token, app, env, branch      |
| `relay.config.json` | Build command overrides — install_cmd, build_cmd, start_cmd |

Common flags (all commands): `--url` `--token` `--app` `--env` `--branch` `--dir` `--host-port` `--mode port|traefik` `--public-host`

Supported lanes: `preview`, `dev`, `staging`, `prod`

---

## RelayDB

RelayDB is Relay's production-oriented PostgreSQL companion. It provisions a
PostgreSQL 17 primary plus PgBouncer transaction pooling on the lane's private
network, generates a unique encrypted credential, waits for both containers to
be ready, and injects `DATABASE_URL` into the app.

Create `relay.json` in the app repository:

```json
{
  "project": "my-app",
  "services": [
    {
      "name": "db",
      "type": "relaydb",
      "profile": "auto"
    }
  ]
}
```

Profiles are `auto`, `starter` (512 MB), `balanced` (2 GB), and `throughput`
(8 GB). `auto` selects conservatively from host memory. The database volume and
credential are stable across deploys and container replacement. Back up both
`relay.db` and `relaydb.key`; losing `relaydb.key` makes stored RelayDB
credentials unrecoverable.

RelayDB removes connection setup overhead and gives applications a strong
single-node foundation. Discord- or YouTube-scale deployments still require
application-specific indexes, caching, partitioning, replicas, object storage,
and eventually multiple nodes; no single database setting replaces that work.

---

`relay.config.json` also supports `project_root`, `build_context`, and `dockerfile` for monorepos and custom Docker builds.

## Monorepos And Dockerfiles

Relay now supports monorepo app roots and custom build contexts directly in `relay.config.json`:

```json
{
  "project_root": "apps/api",
  "build_context": "apps/api",
  "dockerfile": "apps/api/Dockerfile",
  "service_port": 3000
}
```

- `project_root` selects the app root inside the repo for detection and generated artifacts.
- `build_context` selects the Docker build context when it differs from the app root.
- `dockerfile` points to a repo-relative Dockerfile or Containerfile.

If you do not set `dockerfile`, Relay auto-detects root and nested `Dockerfile`, `dockerfile`, `Containerfile`, or `containerfile` files under the selected roots before falling back to buildpack detection.

These same fields are editable in the dashboard under **Settings -> Build layout**.

---

## Runtime Engines

| Engine      | Best for                                                            |
| ----------- | ------------------------------------------------------------------- |
| **Docker**  | Default and recommended — full feature set on any host              |
| **Station** | Experimental local / WSL2 path — not the default production runtime |

> **⚠️ Use Docker.** Docker is Relay's default engine and the supported path for production, staging, dev, and preview lanes. Station is still experimental; keep it for local or WSL2 testing until a stable release is announced.

Switch per app in the dashboard under **Settings → Runtime / Routing**.

---

## Auth

Relay supports two auth modes:

### User accounts (recommended)

On first run with an empty `users` table, the dashboard prompts for setup. Create the first **owner** account, then add additional users from **Server → User Management**.

| Role       | Can deploy | Can manage secrets | Can manage users |
| ---------- | ---------- | ------------------ | ---------------- |
| `owner`    | ✓          | ✓                  | ✓                |
| `deployer` | ✓          | ✓                  | —                |
| `viewer`   | —          | —                  | —                |

Use `relay login --url https://your-relay-server.example` for browser-based CLI auth. Sessions are saved in `~/.relay-state.json` keyed by server URL, so logging into another Relay server does not replace the first session. Each project stores its own server URL in its repository-local `.relay.json`.

### Legacy token mode

If `RELAY_TOKEN` is set (or `data/token.txt` exists) and no users have been created, the server operates in legacy single-token mode. All existing setups continue to work unchanged.

Every request to `relayd` must include your token:

```
X-Relay-Token: <token>
Authorization: Bearer <token>
```

The web dashboard uses an HttpOnly session cookie after login.

---

## Build Tracking

Every deploy is assigned a sequential **build number** per app (`#1`, `#2`, …). The dashboard shows:

- **Build number** — `#42` in the deployment list header
- **Deployed by** — the username who triggered each deploy (for sync/CLI deploys)
- **Commit message** — first line of the git commit message (for webhook-triggered deploys)

---

## GitHub Delivery Workflow

Relay can own the complete branch-to-production path without a separate GitHub Actions workflow:

`Connect GitHub -> push branch -> HTTPS preview -> merge -> production deploy -> health watch -> instant rollback`

Before connecting, configure a public HTTPS dashboard hostname in **Server Settings** (or with `RELAY_DASHBOARD_HOST`) and enable secret encryption. Owners can set `RELAY_SECRET_KEY` from **Server Settings -> Security**, or continue to provide it as an environment variable. Relay will not persist a GitHub credential unless secret encryption is enabled.

In **Dashboard -> GitHub**, choose **Create GitHub App**. Enter an organization name if the repositories are organization-owned, or leave it empty for your personal account. Relay uses GitHub's manifest flow to create a private App owned by that account with only these repository permissions:

- **Contents: read** to clone the selected repository
- **Checks: write** to publish build, preview, health, and production results

After registration, install the App and select exactly which repositories Relay may access. Relay generates repository-scoped installation tokens on demand, keeps them only in memory, and renews them before GitHub's one-hour expiry. Non-production branch pushes create isolated preview lanes and publish the HTTPS route and log link through a GitHub Check Run. A push to the configured production branch—normally GitHub's push after a pull request merge—starts the production rollout. Relay reports build and rollout failures back to GitHub, monitors production health, automatically restores the previous slot when a canary fails, and exposes the same immediate rollback control in the GitHub dashboard page.

The App private key and webhook secret are encrypted in SQLite. Webhook deliveries are signature-verified and idempotent. Removing or suspending an installation immediately blocks new GitHub deploys without deleting existing Relay lanes. Fine-grained personal tokens and manually configured push webhooks remain supported for existing installations.

---

The dashboard also exposes a manual deploy flow with a target lane picker. You can redeploy from the saved server workspace for another lane, or trigger from the saved `repo_url` and branch without opening a shell.

---

## Admin Operations

Owner accounts now get an **Operations** tab in the admin UI. It aggregates:

- live CPU and memory usage
- runtime storage usage
- per-app and per-lane container breakdowns
- latest deploy versus previous deploy deltas for build duration, request volume, bandwidth, and server error rate

Current limitation: live usage metrics are available for Docker-backed lanes. Station lanes show state, but not fake resource telemetry.

---

## Secrets Encryption at Rest

Set `RELAY_SECRET_KEY` to enable AES-256-GCM encryption for all secrets stored in `relay.db`. You can set it from **Dashboard -> Server Settings -> Security**, or provide it as an environment variable:

```bash
RELAY_SECRET_KEY="your-strong-passphrase" relayd
```

The key is hashed to 32 bytes (SHA-256) before use. Secrets written before this key was set remain readable as plain text. New writes are stored as `enc:<base64>`. The deploy path decrypts transparently.

If the value is saved from the dashboard, Relay stores it in `server_config` and uses it on later restarts when the environment variable is absent. `RELAY_SECRET_KEY` from the process environment takes precedence. Relay blocks unsafe dashboard key changes when encrypted rows already exist because old encrypted values need the same key to decrypt.

---

## Audit Log

Every significant action is recorded in `relay.db` and exposed at `GET /api/audit`:

| Action           | Trigger                           |
| ---------------- | --------------------------------- |
| `deploy.trigger` | CLI deploy or GitHub webhook push |
| `secret.set`     | Secret created or updated         |
| `user.create`    | New user account created          |
| `user.delete`    | User account removed              |
| `user.role`      | User role changed                 |

The **Server → Activity Log** panel in the dashboard shows the last 100 entries with actor, target, and timestamp.

---

## Server-Side Buildpack Plugins

Extend framework support without rebuilding `relayd`:

```bash
# Enable mutations on the server first:
RELAY_ENABLE_PLUGIN_MUTATIONS=true relayd

# Install a plugin from any client:
relay plugin install plugins/astro-static.json
relay plugin search astro
relay plugin install-url https://example.com/plugins/astro-static.json --sha256 <hex>

relay plugin list
relay plugin remove astro-static
```

Remote plugin install is intentionally narrower now:

- HTTPS only
- optional SHA256 pin for downloaded plugin JSON
- owner-only mutation endpoints
- plugin mutations can stay disabled outside admin windows

Owners can also browse, install, and remove plugins from the dashboard admin UI, including install-from-file, install-from-URL, paste-JSON, and catalog search.

Sample: [`plugins/astro-static.json`](plugins/astro-static.json)

---

## Production Checklist

- Put `relayd` behind TLS + a reverse proxy (nginx, Caddy, Traefik)
- Create an owner account through the dashboard on first boot (or set `RELAY_TOKEN` for legacy mode)
- Set `RELAY_SECRET_KEY` from the environment or **Server Settings -> Security** to encrypt secrets at rest
- Set `RELAY_CORS_ORIGINS` to your domain allowlist
- Set `RELAY_ENABLE_PLUGIN_MUTATIONS=false` unless actively managing plugins
- Persist `RELAY_DATA_DIR` on a durable volume and back it up
- `relayd` creates `relay.db`, `relaydb.key`, `logs/`, and `token.txt` inside `RELAY_DATA_DIR` (defaults to `./data`)
- Review the Audit Log in the dashboard regularly — especially after onboarding new team members

---

## Repo Layout

```
relay-client/   Node.js CLI (relay)
relayd/         Go agent (relayd) — HTTP API, dashboard, builds, orchestration
station/        Experimental Go runtime (station) — snapshot engine for local / WSL2 workflows
plugins/        Sample buildpack plugins
smoke-apps/     Framework smoke test fixtures
docs/           Roadmap and release notes
```

---

## Docs

- [Contributing](CONTRIBUTING.md)
- [Release & versioning](docs/release-versioning.md)
- [Extensibility roadmap](docs/extensibility-roadmap.md)
- [Agent docs](relayd/README.md)
- [CLI docs](relay-client/README.md)
- [MCP server docs](relay-mcp/README.md)

---

## License

MIT — see [LICENSE](LICENSE).
