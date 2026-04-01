

# relayd — Relay Agent

`relayd` is the local deployment agent for Relay-CI. It receives deploy requests over HTTP, builds app artifacts from simple buildpacks (Next.js, Vite, Expo Web, Sprint UI, Node, Go, .NET, Python, Java, Rust, C/C++, WASM Static, Static), runs lanes with Docker or the experimental Vessel backend, stores deploy history in SQLite, and exposes logs + a fast sync-based deploy flow (no GitHub required).

## Features

- **Multi-language support**: Auto-detects and builds Next.js (standalone & classic), Vite, Expo Web, Sprint UI, Go, .NET, Python, Java, Rust, C/C++, WASM static, and Static sites.
- **Secrets Management**: Inject environment variables per app/branch via the secure Web UI.
- **Concurrency Control**: Prevents parallel build conflicts for the same app context using an internal build queue.
- **Persistent Configuration**: Link GitHub repositories and configure ports/modes server-side.
- **Runtime engines**: Full Docker support plus an experimental Vessel lane engine in the admin panel.
- **GitHub Webhooks**: Native support for triggering redeployments on push events.
- **Zero External Dependencies**: Works purely with Docker and Git (optional).
- **Server-side buildpack plugins**: Install new framework support without recompiling the agent.

## What it does

- **Deploy pipeline**
  - Accepts deploy requests (`/api/deploys`)
  - Detects project type (buildpack auto-detect)
  - Generates a Dockerfile (unless you provide your own)
  - Builds a Docker image or Vessel snapshot, depending on the lane engine
  - Runs/replaces a Docker container or Vessel process for the app
  - Tracks deploys + app state in **SQLite** (`relay.db`)
  - Writes deploy logs to `data/logs/<deployId>.log`

- **Sync deploy (no Git)**
  - Start a sync session
  - Client uploads changed files (or a tar+zstd bundle)
  - Agent applies updates to a workspace repo folder
  - Agent triggers a deploy from that workspace

## Requirements

- Docker installed and running for Docker-backed lanes
- Go 1.22+ (to build from source)
- `vessel` beside `relayd`, or `vessel/` beside `relayd`, for Vessel-backed lanes
- Node/PNPM/Yarn etc remain optional on the host; builds happen inside Docker or Vessel
- (Optional) Node/PNPM/Yarn etc — not needed on host; builds happen in Docker

## Quick start

### 1) Build

From the repo root, the packaging script builds both `relayd` and a platform-matched `vessel` binary side by side:

```bash
./scripts/build-relayd-with-vessel.ps1
```

Manual build also works:

```bash
go build -o relayd .
```

and from [`vessel/`](C:/Users/aloys/Downloads/relay/vessel):

```bash
go build -o vessel .
```

On Windows, the packaged output also includes `vessel-linux`, which the Windows `vessel.exe` copies into WSL2 on first use so packaged `dist/` folders do not need the Go source tree beside them.

### 2) Run

```bash
# creates ./data + token on first run
./relayd
```

### Run 24/7 on Linux (systemd)

To keep `relayd` running after SSH disconnects and across reboots (PM2-style), install it as a systemd service:

```bash
# preview generated unit file
./relayd service unit --user relay --group relay --data-dir /var/lib/relayd

# install + enable service (requires sudo)
sudo ./relayd service install --user relay --group relay --data-dir /var/lib/relayd
```

Useful commands:

```bash
systemctl status relayd
journalctl -u relayd -f
sudo systemctl restart relayd
sudo systemctl stop relayd
```

Notes:

- The service uses `Restart=always` so crashes are auto-restarted.
- `enable --now` starts the service immediately and on every boot.
- If the VPS is powered off, no process can run; `relayd` comes back automatically when the VPS boots.

Important:

- `RELAY_DATA_DIR` defaults to `./data` relative to the current working directory.
- If you run `relayd` from inside a packaged `dist/<goos>-<goarch>/` folder, it creates `dist/<goos>-<goarch>/data/`.
- If you launch `dist/<goos>-<goarch>/relayd` while your shell is still somewhere else, it uses `./data` from that shell location instead.
- You do not manually copy a token into `dist/`. Relay creates `data/token.txt` on first run unless `RELAY_TOKEN` is already set.

On first run, the agent prints a token:

```
Relay token (save this): <TOKEN>
```

### 3) Health check

```bash
curl http://localhost:8080/health
# ok
```

### 4) Web Dashboard

Relay Agent now includes a secure, built-in Web UI for managing your deployments.

*   **URL:** `http://localhost:8080/dashboard/`
*   **Authentication:** Requires the same API token found in the active `data/token.txt` for that run.

The dashboard allows you to:
*   View all recent deployments and their status.
*   Stream live build/startup logs for any deployment.
*   Manage apps and environments.

## Configuration

Environment variables:

| Variable                    |                               Default | Description                                                        |
| --------------------------- | ------------------------------------: | ------------------------------------------------------------------ |
| `RELAY_ADDR`                |                               `:8080` | HTTP listen address                                                |
| `RELAY_DATA_DIR`            |                              `./data` | Storage directory for DB, logs, workspaces                         |
| `RELAY_TOKEN`               |                              *(auto)* | API auth token (if empty, generated and saved to `data/token.txt`) |
| `RELAY_MAX_UPLOAD_BYTES`    |                           `524288000` | Max bytes per sync session (default 500MB)                         |
| `RELAY_CORS_ORIGINS`        |                                  `""` | Comma-separated browser origin allowlist; empty means same-origin only |
| `RELAY_ENABLE_PLUGIN_MUTATIONS` |                           `false` | Allows plugin install/remove API mutations                         |
| `RELAY_VESSEL_BIN`          |                                  `""` | Explicit path to a `vessel` binary for the experimental engine     |
| `RELAY_VESSEL_SOURCE_DIR`   |                                  `""` | Explicit path to `vessel/`; Relay rebuilds `vessel` from it when newer |
| `RELAY_NODE_IMAGE`          |                             `node:22` | Build image for Node builds                                        |
| `RELAY_NODE_RUN_IMAGE`      |                        `node:22-slim` | Runtime image for Node apps                                        |
| `RELAY_GO_IMAGE`            |                         `golang:1.22` | Build image for Go builds                                          |
| `RELAY_GO_RUN_IMAGE`        |     `gcr.io/distroless/base-debian12` | Runtime for Go                                                     |
| `RELAY_DOTNET_SDK_IMAGE`    |    `mcr.microsoft.com/dotnet/sdk:8.0` | Build image for .NET                                               |
| `RELAY_DOTNET_ASPNET_IMAGE` | `mcr.microsoft.com/dotnet/aspnet:8.0` | Runtime image for .NET                                             |
| `RELAY_PY_IMAGE`            |                         `python:3.12` | Build image for Python                                             |
| `RELAY_PY_RUN_IMAGE`        |                    `python:3.12-slim` | Runtime image for Python                                           |
| `RELAY_JAVA_BUILD_IMAGE`    |        `maven:3.9-eclipse-temurin-21` | Build image for Java                                               |
| `RELAY_JAVA_RUN_IMAGE`      |              `eclipse-temurin:21-jre` | Runtime image for Java                                             |
| `RELAY_RUST_IMAGE`          |                           `rust:1.77` | Build image for Rust                                               |
| `RELAY_RUST_RUN_IMAGE`      |                    `debian:bookworm-slim` | Runtime for Rust                                                |
| `RELAY_CC_IMAGE`            |                     `debian:bookworm` | Build image for C / C++                                            |
| `RELAY_CC_RUN_IMAGE`        |                `debian:bookworm-slim` | Runtime for C / C++                                                |
| `RELAY_NGINX_IMAGE`         |                        `nginx:alpine` | Runtime for static sites / Vite static                             |

### Data layout

```
data/
  relay.db
  token.txt
  logs/
    <deployId>.log
  workspaces/
    <app>__<env>__<branch>/
      repo/         # synced or cloned repo content
      staging/      # sync upload staging (per session)
```

If you want a portable release folder, the expected layout is:

```
dist/<goos>-<goarch>/
  relayd(.exe)
  vessel(.exe)
  data/            # created on first run if you start relayd from this folder
    relay.db
    token.txt
    logs/
    workspaces/
```

## Authentication

All `/api/*` endpoints require a token via either:

* `Authorization: Bearer <token>`
* `X-Relay-Token: <token>`

Example:

```bash
curl -H "X-Relay-Token: $RELAY_TOKEN" http://localhost:8080/api/deploys
```

## Deploy API

### GitHub Webhooks (Auto-deploy on Push)

You can link a GitHub repository to Relay by adding a webhook in your GitHub repo settings.

1.  **URL:** `http://<your-agent-ip>:8080/api/webhooks/github`
2.  **Content type:** `application/json`
3.  **Events:** `Just the push event`

**How it works:**
The agent matches the `clone_url` and `branch` from the GitHub payload against apps that have been previously deployed or configured on the agent. When you push to a matched branch, the agent will:
1.  Pull the latest changes from GitHub (using `git clone` or `git pull`).
2.  Run the standard build/deploy pipeline.

**Linking a repository to an app:**
You can configure an app's repository URL and other defaults on the agent via the `/api/apps/config` endpoint. Once set, you no longer need to provide these details during manual deploys or webhooks.

Example using `curl`:
```bash
curl -X POST -H "X-Relay-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{"app":"demo", "env":"preview", "branch":"main", "repo_url":"https://github.com/user/repo.git"}' \
  http://localhost:8080/api/apps/config
```

### Create deploy (git-based or external orchestration)

`POST /api/deploys`

Body:

```json
{
  "app": "demo",
  "repo_url": "https://github.com/org/repo.git",
  "branch": "main",
  "commit_sha": "",
  "env": "preview",
  "mode": "port",
  "source": "git",
  "service_port": 3000,
  "host_port": 3001,
  "public_host": "",
  "install_cmd": "",
  "build_cmd": "",
  "start_cmd": ""
}
```

Useful `kind` values in this checkout:

- `next-standalone`
- `vite-static`
- `expo-web`
- `sprint-ui`
- `node`
- `go`
- `dotnet`
- `python`
- `java`
- `rust`
- `c`
- `cpp`
- `c-cpp`
- `wasm-static`
- `static`

Notes:

* Buildpack detects from the workspace repo folder.
* This agent currently focuses on Docker build/run. If you want git cloning, do it outside and sync the repo contents into the workspace (or implement cloning in your controller).

Response: `201` with the deploy record.

### List deploys

`GET /api/deploys`
Returns latest ~200 deploy records (from SQLite).

### Get deploy by ID

`GET /api/deploys/:id`

### Buildpack plugins

Relay supports JSON-based buildpack plugins stored on the server in `data/plugins/buildpacks/`.

Endpoints:

- `GET /api/plugins/buildpacks`
- `POST /api/plugins/buildpacks`
- `DELETE /api/plugins/buildpacks/:name`

Mutation note:

- `POST` and `DELETE` are disabled unless `RELAY_ENABLE_PLUGIN_MUTATIONS=true`

Plugin definitions let you add:

- custom detect rules
- Dockerfile templates
- install/build/start defaults
- cleanup paths

Example plugin file: [`plugins/astro-static.json`](C:/Users/aloys/Downloads/relay/plugins/astro-static.json)

### Logs (full)

`GET /api/logs/:id`
Returns plain text log file content.

### Logs (stream, SSE)

`GET /api/logs/stream/:id?from=<byteOffset>`

* Returns `text/event-stream`
* `from` is an optional byte offset into the log file.

## Sync API (no GitHub)

Sync is designed for “Vercel-like with no GitHub”: you upload a repo snapshot (or diffs) to the agent, then trigger a deploy.

### 1) Start session

`POST /api/sync/start`

```json
{ "app": "demo", "branch": "main", "env": "preview" }
```

Response:

```json
{ "session_id": "..." }
```

### 2) Plan which files are needed

`POST /api/sync/plan/:sessionId`

You send a manifest of files you have; server compares to its current workspace repo.

```json
{
  "files": [
    { "path": "package.json", "size": 123, "mtime": 1700000000000, "sha256": "..." },
    { "path": "app/page.tsx", "size": 456, "mtime": 1700000000000, "sha256": "..." }
  ]
}
```

Response:

```json
{ "need": ["app/page.tsx"], "delete": ["old/file.txt"] }
```

### 3a) Upload individual files

`PUT /api/sync/upload/:sessionId?path=app/page.tsx`
Body is raw file bytes.

### 3b) Upload bundle (tar + zstd)

`PUT /api/sync/bundle/:sessionId`
Body is a zstd-compressed tar stream with file entries.

### 4) Tell server which files to delete

`POST /api/sync/delete/:sessionId`

```json
{ "paths": ["old/file.txt"] }
```

### 5) Finish & deploy

`POST /api/sync/finish/:sessionId`

Optional body overrides:

```json
{
  "mode": "port",
  "host_port": 3001,
  "service_port": 3000,
  "public_host": "",
  "source": "sync",
  "install_cmd": "",
  "build_cmd": "",
  "start_cmd": ""
}
```

Response is a new Deploy record.

## App control

These endpoints operate on the currently saved app state (image tag, ports, mode).

* `POST /api/apps/stop`
* `POST /api/apps/start`
* `POST /api/apps/restart`

Body:

```json
{ "app": "demo", "branch": "main", "env": "preview" }
```

## Rollback

`POST /api/deploys/rollback`

Body:

```json
{ "app": "demo", "branch": "main", "env": "preview" }
```

Rolls back to the previously deployed image (if available in app_state).

## relay.config.json

Place `relay.config.json` in the repo root to influence buildpack selection and commands.

Example:

```json
{
  "kind": "next-standalone",
  "service_port": 3000,
  "install_cmd": "npm ci",
  "build_cmd": "npm run build",
  "start_cmd": "node server.js"
}
```

### Custom Dockerfile

If you want full control, specify a Dockerfile path inside the repo:

```json
{
  "dockerfile": "Dockerfile",
  "service_port": 8080
}
```

The agent will run:

```
docker build -f <that path> -t <tag> .
```

## Security notes (read this)

This is a **local-dev / self-hosted agent**. By default it:

* Runs Docker builds/containers or Vessel snapshots/processes on the host
* Accepts file uploads (sync)

Production recommendations:

* Set `RELAY_CORS_ORIGINS` to the exact dashboard origins you trust
* Leave `RELAY_ENABLE_PLUGIN_MUTATIONS=false` unless you are actively administering plugins
* Run behind TLS and a reverse proxy
* Treat the API token as full deploy access
* Treat Vessel as experimental: it currently forces port routing, edge cutover, and keeps companion services off

## License

Copyright 2026 babymonie

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
