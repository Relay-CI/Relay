

# relay-deploy — Relay Node Client (CLI)

A zero-dependency Node 18+ CLI that deploys a local folder to a Relay Agent using the **sync API** (no GitHub required). It:
- starts a sync session
- computes a file manifest (sha256)
- uploads only the changed files
- triggers a build + container swap on the agent
- optionally streams deploy logs over SSE

## Requirements

- Node **18+** (uses built-in `fetch`)
- A running Relay Agent (`relayd`) + token
- Works on Windows/macOS/Linux

## Install

### Option A: run directly
Keep `relay-deploy.js` in your repo and run with Node:
```bash
node relay-deploy.js deploy --url http://127.0.0.1:8080 --token YOURTOKEN --app myapp --env preview --branch main --dir . --stream
````

### Option B: make it a proper CLI (recommended)

Add to `package.json`:

```json
{
  "bin": {
    "relay-deploy": "./relay-deploy.js"
  }
}
```

Then install globally or use `npx` from a local package:

```bash
npm i -g .
# now:
relay-deploy deploy --url http://127.0.0.1:8080 --token YOURTOKEN --app myapp --env preview --branch main --dir . --stream
```

## Quickstart

### 1) Init local config (optional)

Writes `.relay.json` in the current directory:

```bash
node relay-deploy.js init --url http://127.0.0.1:8080 --token YOURTOKEN --app myapp --env preview --branch main --dir .
```

### 2) Deploy

```bash
node relay-deploy.js deploy --dir . --stream
```

If you used `init`, you can omit most flags.

## Commands

### `deploy`

Syncs your local directory to the agent workspace and triggers a deploy.

Example:

```bash
node relay-deploy.js deploy \
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

Writes `.relay.json` into your current directory:

```bash
node relay-deploy.js init --url http://127.0.0.1:8080 --token YOURTOKEN --app myapp --env preview --branch main --dir .
```

### `start` / `stop` / `restart`

Controls the currently deployed container for an app/env/branch:

```bash
node relay-deploy.js restart --app myapp --env preview --branch main
```

### `plugin list`

Lists server-installed buildpack plugins:

```bash
relay plugin list --url http://127.0.0.1:8080 --token YOURTOKEN
```

### `plugin install`

Uploads a JSON buildpack plugin definition to the server:

```bash
relay plugin install ./plugins/astro-static.json --url http://127.0.0.1:8080 --token YOURTOKEN
```

Server note:

- plugin install/remove requires `RELAY_ENABLE_PLUGIN_MUTATIONS=true` on the agent

### `plugin remove`

Removes a server-installed buildpack plugin:

```bash
relay plugin remove astro-static --url http://127.0.0.1:8080 --token YOURTOKEN
```

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
