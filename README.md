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
npm install -g @relay-ci/relay
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

On first run it creates `data/token.txt` — your auth token. Open `http://<server>:8080` for the dashboard.

**2. Init your project** (run inside your app folder)

```bash
relay init
```

Walks through server URL, token, app name, env, and branch. Writes `.relay.json`.

**3. Deploy**

```bash
relay deploy --stream
```

Relay detects your buildpack, syncs only changed files, builds a container, and does a zero-downtime rollout.

---

## Commands

```
relay init                         Interactive setup → writes .relay.json
relay deploy [--stream]            Sync + build + rollout
relay status                       Latest deploy status
relay logs <id>                    Stream build logs
relay list                         Recent deploys
relay projects                     All projects and environments
relay rollback                     Roll back to previous image
relay start / stop / restart       Control a running container
relay secrets list/add/rm          Manage app secrets
relay plugin list/install/remove   Manage server-side buildpack plugins
relay agent install [--version v]  Download relayd + station binaries
relay agent status                 Show installed agent version and path
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

Switch per app in the dashboard under **Settings → Runtime / Routing**.

---

## Auth

Every request to `relayd` must include your token:

```
X-Relay-Token: <token>
Authorization: Bearer <token>
```

The web dashboard uses an HttpOnly session cookie after login.

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
- Set `RELAY_TOKEN` explicitly instead of relying on auto-generated `token.txt`
- Set `RELAY_CORS_ORIGINS` to your domain allowlist
- Set `RELAY_ENABLE_PLUGIN_MUTATIONS=false` unless actively managing plugins
- Persist `RELAY_DATA_DIR` on a durable volume and back it up
- `relayd` creates `relay.db`, `logs/`, and `token.txt` inside `RELAY_DATA_DIR` (defaults to `./data`)

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
