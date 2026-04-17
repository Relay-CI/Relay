# relay-mcp

Model Context Protocol server for [Relay](https://github.com/Relay-CI/Relay). Lets any MCP-aware AI tool (Cursor, Claude Desktop, VS Code Copilot, etc.) inspect and control a Relay agent directly.

## Setup

Add to your MCP config (`~/.cursor/mcp.json`, Claude Desktop config, etc.):

```json
{
  "mcpServers": {
    "relay": {
      "command": "npx",
      "args": ["-y", "@relay-org/relay-mcp"],
      "env": {
        "RELAY_URL": "http://your-server:8080",
        "RELAY_TOKEN": "your-token"
      }
    }
  }
}
```

For a local agent on the same machine, use a Unix socket instead (no token needed):

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

## Environment variables

| Variable | Description |
|---|---|
| `RELAY_SOCKET` | Path to `relay.sock`. Takes priority over HTTP if set. |
| `RELAY_URL` | Agent base URL. Default: `http://127.0.0.1:8080` |
| `RELAY_TOKEN` | Bearer token for HTTP transport. |

## Tools

| Tool | Description |
|---|---|
| `list_projects` | All projects and environments |
| `list_deploys` | Recent deploys, filterable by app/env/branch |
| `get_deploy` | Single deploy record by ID |
| `get_deploy_logs` | Build/deploy logs for a deploy ID |
| `cancel_deploy` | Cancel an in-progress deploy |
| `rollback` | Roll back to the previous image |
| `start_app` | Start a stopped container |
| `stop_app` | Stop a running container |
| `restart_app` | Restart a container without a new build |
| `delete_lane` | Remove a lane and its state |
| `get_app_config` | Lane config (access policy, hosts, engine) |
| `set_app_config` | Update lane config |
| `list_secrets` | Secret key names for an app lane |
| `add_secret` | Add or update a secret |
| `remove_secret` | Delete a secret |
| `list_promotions` | Promotion requests with approval state |
| `approve_promotion` | Approve a queued promotion |
| `list_users` | All user accounts (owner only) |
| `get_audit_log` | Recent audit entries (owner only) |
| `get_server_config` | Server-level config and theme |
| `set_server_config` | Update server config or theme |
| `list_buildpack_plugins` | Installed buildpack plugins |
| `remove_buildpack_plugin` | Remove a buildpack plugin |
| `get_version` | relayd and Station version info |
| `create_signed_link` | Time-bounded share URL for signed-link lanes |
| `list_companions` | Companion services for an app lane |
| `restart_companion` | Restart a companion service |

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
