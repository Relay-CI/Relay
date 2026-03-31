

# relay — Relay CLI

[![npm](https://img.shields.io/npm/v/@relay-org/relay)](https://www.npmjs.com/package/@relay-org/relay)
[![license](https://img.shields.io/npm/l/@relay-org/relay)](LICENSE)

Zero-dependency Node 18+ CLI for the [Relay](https://github.com/Relay-CI/Relay) self-hosted deployment platform.

- Syncs only changed files (sha256 diff)
- Auto-detects buildpacks (Node, Go, Python, Rust, .NET, Java, C/C++, WASM, static)
- Streams build + deploy logs in real time
- Downloads `relayd` + `station` agent binaries automatically

## Requirements

- Node **18+**
- A running `relayd` agent + token
- Works on Windows / macOS / Linux

## Install

```bash
npm install -g @relay-org/relay
```

## Quick Start

```bash
# 1. Download and install the agent binaries (one-time)
relay agent install

# 2. Start the agent on your server
relayd

# 3. Init your project (writes .relay.json)
relay init

# 4. Deploy
relay deploy --stream
```

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

## Usage

### `deploy`

```bash
relay deploy --stream
```

With explicit flags (overrides `.relay.json`):

```bash
relay deploy \
  --url http://127.0.0.1:8080 \
  --token YOURTOKEN \
  --app myapp \
  --env preview \
  --branch main \
  --dir . \
  --mode port \
  --host-port 3001 \
  --service-port 3000 \
  --public-host "" \
  --stream
```

#### Optional overrides

These override defaults from config files / agent buildpacks:

* `--install-cmd "npm ci"`
* `--build-cmd "npm run build"`
* `--start-cmd "npm start"`
* `--service-port 3000`

### `init`

```bash
relay init
# or with flags:
relay init --url http://127.0.0.1:8080 --token YOURTOKEN --app myapp --env preview
```

### `start` / `stop` / `restart`

```bash
relay restart --app myapp --env preview --branch main
```

### `plugin list / install / remove`

```bash
relay plugin list
relay plugin install ./plugins/astro-static.json
relay plugin remove astro-static
```

> Plugin install/remove requires `RELAY_ENABLE_PLUGIN_MUTATIONS=true` on the agent.

### `agent install`

Downloads the correct `relayd` + `station` binaries for your platform from GitHub Releases:

```bash
relay agent install           # latest
relay agent install --version v0.1.7
```

Installs to `~/.relay/bin/` and prints the PATH line to add.

## Flags

| Flag             | Required | Example                 | Notes                                                   |
| ---------------- | -------: | ----------------------- | ------------------------------------------------------- |
| `--url`          |  usually | `http://127.0.0.1:8080` | Relay Agent base URL                                    |
| `--token`        |      yes | `abcd...`               | X-Relay-Token / Bearer auth                             |
| `--app`          |      yes | `moneypasar`            | Workspace key                                           |
| `--env`          |      yes | `preview` or `prod`     | Determines default host port behavior on agent          |
| `--branch`       |      yes | `main`                  | Included in workspace key                               |
| `--dir`          |      yes | `.`                     | Local folder to deploy                                  |
| `--mode`         |       no | `port`                  | Agent supports `port` (and can later support `traefik`) |
| `--host-port`    |       no | `3001`                  | Host port mapping (mode=port)                           |
| `--service-port` |       no | `3000`                  | Container port (if your app doesn’t use defaults)       |
| `--public-host`  |       no | `demo.local`            | Stored as metadata (useful with a reverse proxy)        |
| `--stream`       |       no | `true`                  | Stream logs via SSE                                     |

## Config resolution order

The CLI merges settings in this order (highest priority first):

1. CLI flags (`--token`, `--app`, etc.)
2. Local `.relay.json` (created by `init`)
3. `relay.config.json` (build/install/start overrides only)
4. Environment variables (`RELAY_URL`, `RELAY_TOKEN`, `RELAY_APP`, `RELAY_ENV`, `RELAY_BRANCH`)
5. Fallback defaults (`url=http://127.0.0.1:8080`, `env=preview`, `branch=main`, `dir=.`)

`relay.json` is not part of CLI connection resolution. It is reserved for project companion services such as Postgres and Redis.

## What gets uploaded

The CLI walks files under `--dir` and ignores common heavy/build output folders (must match the agent’s ignore list):

* `node_modules`, `.git`, `.next`, `dist`, `.turbo`, `coverage`, `.relay`, `cache`, `bin`, `obj`, `target`

It sends a manifest containing `{path, size, mtime, sha256}` and uploads only the files the agent requests.

## Security notes

* Treat the Relay token like a password. Don’t commit it.
* By default, the agent may be configured with permissive CORS and broad local access—keep the agent bound to `127.0.0.1` unless you’ve hardened it behind a proxy.

## Troubleshooting

### “Missing --token / --app”

Run `init` or provide the flags:

```bash
node relay-deploy.js init --url http://127.0.0.1:8080 --token YOURTOKEN --app myapp --env preview --branch main --dir .
```

### Upload is slow on big repos

Right now the manifest computes sha256 for every file. Later on will improve.
## License
Copyright 2026 babymonie

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
