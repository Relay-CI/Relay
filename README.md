# Relay

Self-hosted deployment platform — sync changed files, auto-detect buildpacks, roll out containers. No GitHub Actions, no cloud platform required.

| Component | What it does |
|---|---|
| **`relay`** | Node.js CLI — deploy from any machine |
| **`relayd`** | Go agent — runs on your server, builds images, manages containers |
| **`station`** | Go runtime — snapshot-based fast deployment for local/WSL2 targets |

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

Auto-detects your platform (Linux amd64/arm64, Windows) and downloads `relayd` + `station` from the latest GitHub Release into `~/.relay/bin/`. No Go required.

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
relay plugin list/install/remove   Manage server-side buildpack plugins
relay version                      Show relay/relayd/station versions
relay agent install [--version v]  Download relayd + station binaries
relay agent update                 Update relayd + station to latest release
relay agent status                 Show installed/latest versions and outdated status
```

---

## Buildpacks

Relay auto-detects your framework:

`node-next` · `node-vite` · `expo-web` · `node-generic` · `go` · `dotnet` · `python` (Flask/FastAPI) · `java` · `rust` · `c-cpp` · `wasm-static` · `static` · `sprint-ui`

Add more without rebuilding `relayd` — see [Server-Side Buildpack Plugins](#server-side-buildpack-plugins).

---

## Config

| File | Purpose |
|---|---|
| `.relay.json` | CLI connection defaults — url, token, app, env, branch |
| `relay.config.json` | Build command overrides — install_cmd, build_cmd, start_cmd |

Common flags (all commands): `--url` `--token` `--app` `--env` `--branch` `--dir` `--host-port` `--mode port|traefik` `--public-host`

---

## Runtime Engines

| Engine | Best for |
|---|---|
| **Docker** | Production — full feature set on any host |
| **Station** | Fast local / WSL2 — snapshot-based, instant rollout |

> **⚠️ Use Docker.** Station is under active development and is currently unstable — deploys can be slow or fail unexpectedly. When `relay init` asks which engine to use, pick **docker**. Do not switch to Station in the dashboard until a stable release is announced.

Switch per app in the dashboard under **Settings → Runtime / Routing**.

---

## Auth

Relay supports two auth modes:

### User accounts (recommended)

On first run with an empty `users` table, the dashboard prompts for setup. Create the first **owner** account, then add additional users from **Server → User Management**.

| Role | Can deploy | Can manage secrets | Can manage users |
|---|---|---|---|
| `owner` | ✓ | ✓ | ✓ |
| `deployer` | ✓ | ✓ | — |
| `viewer` | — | — | — |

Use `relay login` for browser-based CLI auth — opens a browser tab, you log in once, and the token is saved to `~/.relay-state.json`.

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

## Secrets Encryption at Rest

Set `RELAY_SECRET_KEY` to enable AES-256-GCM encryption for all secrets stored in `relay.db`:

```bash
RELAY_SECRET_KEY="your-strong-passphrase" relayd
```

The key is hashed to 32 bytes (SHA-256) before use. Secrets written before this key was set remain readable as plain text. New writes are stored as `enc:<base64>`. The deploy path decrypts transparently.

---

## Audit Log

Every significant action is recorded in `relay.db` and exposed at `GET /api/audit`:

| Action | Trigger |
|---|---|
| `deploy.trigger` | CLI deploy or GitHub webhook push |
| `secret.set` | Secret created or updated |
| `user.create` | New user account created |
| `user.delete` | User account removed |
| `user.role` | User role changed |

The **Server → Activity Log** panel in the dashboard shows the last 100 entries with actor, target, and timestamp.

---

## Server-Side Buildpack Plugins

Extend framework support without rebuilding `relayd`:

```bash
# Enable mutations on the server first:
RELAY_ENABLE_PLUGIN_MUTATIONS=true relayd

# Install a plugin from any client:
relay plugin install plugins/astro-static.json

relay plugin list
relay plugin remove astro-static
```

Sample: [`plugins/astro-static.json`](plugins/astro-static.json)

---

## Production Checklist

- Put `relayd` behind TLS + a reverse proxy (nginx, Caddy, Traefik)
- Create an owner account through the dashboard on first boot (or set `RELAY_TOKEN` for legacy mode)
- Set `RELAY_SECRET_KEY` to encrypt secrets at rest
- Set `RELAY_CORS_ORIGINS` to your domain allowlist
- Set `RELAY_ENABLE_PLUGIN_MUTATIONS=false` unless actively managing plugins
- Persist `RELAY_DATA_DIR` on a durable volume and back it up
- `relayd` creates `relay.db`, `logs/`, and `token.txt` inside `RELAY_DATA_DIR` (defaults to `./data`)
- Review the Audit Log in the dashboard regularly — especially after onboarding new team members

---

## Repo Layout

```
relay-client/   Node.js CLI (relay)
relayd/         Go agent (relayd) — HTTP API, dashboard, builds, orchestration
station/        Go runtime (station) — snapshot engine, WSL2 sidecar, desktop UI
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

---

## License

MIT — see [LICENSE](LICENSE).
