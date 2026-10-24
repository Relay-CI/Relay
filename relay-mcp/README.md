# relay-mcp

Model Context Protocol server for [Relay](https://github.com/Relay-CI/Relay). Connect Cursor, Claude Desktop, VS Code, or another MCP-aware client to inspect and operate a Relay agent.

The client launches this package as a **local stdio process**. That process connects to `relayd` over HTTP/HTTPS or a local Unix socket. It does not upload your app source: use `relay deploy --stream` in your app folder, then inspect the deployment through MCP.

## Setup

You need Node.js 18 or later on the client computer, a running agent, and either a token for that agent or access to its local socket. `npx` downloads the package when needed; no global install is required.

Add to your MCP config (`~/.cursor/mcp.json`, Claude Desktop config, etc.):

```json
{
  "mcpServers": {
    "relay": {
      "command": "npx",
      "args": ["-y", "@relay-org/relay-mcp"],
      "env": {
        "RELAY_URL": "https://relay.example.com",
        "RELAY_TOKEN": "YOUR_RELAY_TOKEN"
      }
    }
  }
}
```

Replace the example URL and token. For a local HTTP trial, use `http://127.0.0.1:8080`. Merge the entry with existing servers and keep credentials out of source control.

**Windows:** if the client cannot launch `npx` directly, use `"command": "cmd"` with `"args": ["/c", "npx", "-y", "@relay-org/relay-mcp"]`. Node.js must be visible to the desktop app, not just a terminal.

**VS Code:** use `.vscode/mcp.json` with a top-level `servers` object instead of `mcpServers`, and add `"type": "stdio"` to the `relay` entry.

For a local agent that exposes a Unix socket, use its **absolute** path instead (no token needed):

```json
{
  "mcpServers": {
    "relay": {
      "command": "npx",
      "args": ["-y", "@relay-org/relay-mcp"],
      "env": {
        "RELAY_SOCKET": "/path/to/relay.sock"
      }
    }
  }
}
```

The socket path belongs to the agent's data directory, not your app folder. The MCP process needs filesystem permission to access it. Use HTTP if the agent is remote or the operating system does not expose the socket.

Restart or reconnect the MCP server after saving. Ask the assistant to run `get_version`, `get_doctor`, then `list_projects` to verify the connection without modifying the server.

## Environment variables

| Variable | Description |
|---|---|
| `RELAY_SOCKET` | Path to `relay.sock`. Takes priority over HTTP if set. |
| `RELAY_URL` | Agent base URL. Default: `http://127.0.0.1:8080` |
| `RELAY_TOKEN` | Bearer token for HTTP transport. |
| `RELAY_REQUEST_TIMEOUT_MS` | Total request deadline, including reading the response. Default `30000`; accepts integers from `1` to `600000`. |

`RELAY_URL` supports reverse-proxy path prefixes, for example `https://example.com/relay`. Configure the final URL: redirects are reported as errors and are not followed. Socket mode takes precedence over the URL. Responses larger than 16 MiB are rejected; request fewer records or use the dashboard for large logs.

## Tools

<!-- mcp-tools:start -->
There are **43 tools**. Lane branches default to `main` unless a tool states otherwise. The agent enforces account and lane permissions.

| Tool | Action | Description |
| --- | --- | --- |
| `get_doctor` | Inspect | Check agent setup and diagnostics, including Docker, persistent storage, DNS, TLS, and proxy health. Read-only; does not change server settings. |
| `list_projects` | Inspect | List all projects and their environments known to the Relay agent. |
| `list_deploys` | Inspect | List recent deploys. Filter by app, env, and/or branch. Returns build numbers, statuses, deployed_by, commit messages, and timestamps. |
| `get_deploy` | Inspect | Get a single deploy record by ID including status, logs reference, image tag, and commit info. |
| `get_deploy_logs` | Inspect | Fetch the stored build and deploy logs for a given deploy ID. |
| `cancel_deploy` | Change | Cancel an in-progress deploy by ID. |
| `rollback` | Change | Roll back the most recent deploy to the previous image for an app/env/branch. |
| `start_app` | Change | Start a stopped app container without triggering a new build. |
| `stop_app` | Change | Stop a running app container. |
| `restart_app` | Change | Restart a running app container in place without a new build. |
| `delete_lane` | Change | Delete a lane and remove its container, workspace, and state from the agent. |
| `get_app_config` | Inspect | Get the current lane configuration for an app including access_policy, public_host, engine, host_port, service_port, and rollout settings. |
| `set_app_config` | Change | Update lane configuration for an app. Only supply the fields you want to change. |
| `list_secrets` | Inspect | List secret key names (not values) for an app/env/branch. |
| `add_secret` | Change | Add or update a secret value for an app/env/branch. The value is encrypted at rest if RELAY_SECRET_KEY is configured on the agent. |
| `remove_secret` | Change | Delete a secret by key for an app/env/branch. |
| `list_promotions` | Inspect | List promotion requests. Shows source/target lane, image, approval state, and post-promote health status. |
| `approve_promotion` | Change | Approve a queued promotion request by ID. Requires owner role. |
| `list_users` | Inspect | List all user accounts with their roles. Requires owner role. |
| `get_audit_log` | Inspect | Fetch recent audit log entries showing actor, action, target, and timestamp. Requires owner role. |
| `get_server_config` | Inspect | Get server-level config: base_domain, dashboard_host, ACME settings, and active theme. Requires owner role. |
| `set_server_config` | Change | Update server-level config. Supply only the fields to change. Requires owner role. |
| `list_buildpack_plugins` | Inspect | List installed buildpack plugins on the agent. |
| `remove_buildpack_plugin` | Change | Remove an installed buildpack plugin by name. Requires RELAY_ENABLE_PLUGIN_MUTATIONS=true on the agent. |
| `get_version` | Inspect | Get relayd build metadata and Station runtime version details. |
| `create_signed_link` | Change | Generate a time-bounded signed share URL for a lane that uses access_policy=signed-link. |
| `list_companions` | Inspect | List managed companion services (databases, caches, etc.) attached to an app lane. |
| `restart_companion` | Change | Restart a named companion service in place. |
| `list_variants` | Inspect | Read a lane's control release, variants, traffic weights, request/error/latency metrics, and latest 50 feedback entries for 24h, 7d, or 30d. Includes support limitations. |
| `set_variant_weight` | Change | Change a variant's share of new visitors (0-100 percent). Existing visitor assignments remain sticky. All running variant weights must total at most 100; control receives the remainder. |
| `promote_variant` | Change | Queue a deployment of a variant's existing image as the lane's control release. Returns deploy_id; the variant is retired after that deployment succeeds. This is separate from cross-lane promotion approvals. |
| `remove_variant` | Change | Remove a variant's route and containers. Visitors assigned to it fall back to control. Does not remove the lane's control release. |
| `reshuffle_variants` | Change | Reset visitor assignments for a lane under its current variant weights. Every visitor is assigned again on a subsequent request. |
| `scale_app` | Change | Set 1-16 replicas for an existing Docker lane. With apply=true, queues an image redeployment when a running lane needs resizing. With apply=false, saves the size for a later deployment. Returns deploy_id when queued. |
| `get_analytics` | Inspect | Read lane traffic, errors, latency, bandwidth, and runtime resource data, optionally filtered to one variant (or control). HTTP/HTTPS only; scoped viewer access is required. |
| `abort_rollout` | Change | Abort a canary rollout that is currently monitoring and restore its previous slot. Requires an active standby slot. HTTP/HTTPS only. |
| `request_promotion` | Change | Request promotion of a healthy lane's current image to a different lane. Requires deployer access to both lanes. Production requests by non-owners wait for owner approval when user accounts are enabled. |
| `get_runtime_targets` | Inspect | List available app, replica, variant, proxy, and companion runtime log targets for a lane, plus the default target and offline state. Use the dashboard Runtime logs page to follow a target's live output. |
| `get_admin_ops` | Inspect | Read the owner-only operations snapshot: container CPU, memory, storage, lane health, daemon memory, and deployment comparisons. |
| `get_status` | Inspect | Read the agent's runtime health inventory, including edge and active-slot state, stopped lanes, exit codes, and out-of-memory indicators. |
| `list_buildpack_catalog` | Inspect | Read the built-in buildpack plugin catalog. Requires owner role. |
| `install_buildpack_from_catalog` | Change | Install a named plugin from the agent's built-in catalog. Requires owner role and plugin mutations enabled on the agent. |
| `install_buildpack_from_url` | Change | Install buildpack plugin JSON from an HTTPS URL, optionally pinned to a SHA256 digest. Requires owner role and plugin mutations enabled. The agent validates and fetches the URL. |
<!-- mcp-tools:end -->

`get_analytics` and `abort_rollout` require HTTP/HTTPS because the agent does not register those routes on its local socket. Runtime-log discovery works over either transport; follow live output in the dashboard rather than treating the SSE endpoint as finite JSON.

Variant creation from local source still uses `relay deploy --variant <name> --weight <percent>`. `scale_app` and `promote_variant` can queue deployments: inspect the returned deployment ID before assuming the change has finished. Variant promotion is within one lane and is separate from cross-lane promotion approvals.

`create_signed_link` takes a lifetime in seconds, rounds up to whole minutes, and caps it at seven days. Plugin installation requires owner access and plugin mutations enabled on the agent.

## Troubleshooting

- **The server appears idle in a terminal:** stdio waits for a client. Configure it in the client instead of expecting a standalone browser UI.
- **Cannot reach the agent:** check `RELAY_URL`, whether `relayd` is running, and any stale `RELAY_SOCKET` that may override HTTP.
- **HTTP 401:** check the token belongs to the selected server. **HTTP 403:** check the account role for the requested operation.
- **Socket failure:** check the absolute socket path and filesystem permissions.
- **Timeout:** inspect the agent and proxy first. Increase `RELAY_REQUEST_TIMEOUT_MS` only when the operation needs more time.
- **Missing tools:** reconnect and refresh the client tool list. Set `@relay-org/relay-mcp@latest` in the launch arguments to request the latest published package, or pin a specific version.

Agent failures return MCP tool results with `isError: true`. They do not terminate the MCP process, so you can correct the connection and retry.

## Local development

From `relay-mcp/`:

```sh
npm ci
npm test
npm run docs:sync
npm run docs:check
npm pack --dry-run
```

To use unpublished local changes, set the client's command to `node` and its arguments to the absolute path of this checkout's `relay-mcp/index.js`. Keep the same connection environment variables. Tests use a temporary mock agent and exercise the real stdio handshake, tool discovery, request routing, timeouts, and error recovery.

`docs:sync` discovers the local server's tools without calling an agent. It regenerates this README's tool table and `../site/src/lib/mcp-tools.json`. `docs:check` fails if either reference is stale. Commit the generated site file with the corresponding MCP changes in the Relay repository.

## Publishing (maintainers)

`@relay-org/relay-mcp` is published from GitHub Actions using npm trusted publishing (OIDC).

If publish fails with `npm ERR! code E404` on:

`PUT https://registry.npmjs.org/@relay-org%2frelay-mcp`

verify package-level trusted publisher settings on npmjs.com for `@relay-org/relay-mcp`:

- Organization/user: `Relay-CI`
- Repository: `Relay`
- Workflow filename: `auto-release.yml` (must match exactly)
- Environment: `npm-publish` (must match exactly if configured in workflow)

Important:

- Trusted publishing is configured per package, not per scope.
- npm currently allows one trusted publisher connection per package.
- If another workflow publishes this package, either:
  - switch the trusted publisher to that workflow, or
  - use a granular npm token in that alternate workflow.
