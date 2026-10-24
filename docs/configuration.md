# Choose the right configuration file

[All documentation](README.md) · [First deployment](getting-started.md)

Relay separates CLI defaults, build instructions, service declarations, and upload exclusions. Start with `relay init`; you do not need to create all four files for a first deployment.

| File in the app folder | What it controls | When you need it |
| --- | --- | --- |
| `.relay.json` | Server URL or socket, credentials, app, environment, branch, and optional directory | Created by `relay init`; used for CLI defaults |
| `relay.config.json` | Framework hints, install/build/start commands, service port, and monorepo paths | When automatic detection needs an override |
| `relay.json` | Databases and companion services | When the app needs services alongside it |
| `.relayignore` | Additional paths excluded from file synchronization | When generated or irrelevant files should not be uploaded |

## Connect the CLI: `.relay.json`

Run `relay init` in your app's root folder to create this file. The following example omits credentials; use `relay login` for your server or follow the wizard's authentication prompts.

```json
{
  "url": "https://relay.example.com",
  "app": "my-app",
  "env": "dev",
  "branch": "main"
}
```

Replace the example URL and lane values. Keep tokens private and do not commit a file containing credentials. See the [CLI reference](../relay-client/README.md) for authentication and configuration precedence.

## Change the build: `relay.config.json`

Use this file when the detected framework needs different commands or paths. For example, an app in a monorepo can specify its root and Dockerfile:

```json
{
  "project_root": "apps/api",
  "build_context": "apps/api",
  "dockerfile": "apps/api/Dockerfile",
  "service_port": 3000
}
```

These are illustrative paths and a container port; match them to your project. `service_port` is the port inside the app container. A `host_port` is the port exposed on the server in port mode.

For command overrides, supported fields include `install_cmd`, `build_cmd`, and `start_cmd`. Keep secret values out of build instructions.

Docker apps can also set `replicas` (1–16), `readiness_path` (an HTTP endpoint that returns 2xx), and `volumes` for persistent app mounts. For example, merge `"replicas": 3` and `"readiness_path": "/health"` into the file when your app provides that health endpoint. See [replicas and readiness](scaling-and-operations.md) for rollout behavior and capacity requirements.

## Add a service: `relay.json`

This example declares a RelayDB workload for the project:

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

RelayDB runs PostgreSQL with PgBouncer pooling and injects `DATABASE_URL` into the app. Its data and credentials stay associated with the lane across deployments. Preserve `relay.db` and `relaydb.key` in backups; the key is needed to recover stored RelayDB credentials.

See [RelayDB in the main reference](../README.md#relaydb) for profiles and operational limits, and the [agent reference](../relayd/README.md) for companion service behavior.

## Exclude uploads: `.relayignore`

Use this file for project-local exclusions on top of Relay's built-in ignores:

```gitignore
coverage/
tmp/
```

Exclude only files the remote build does not need. If a required source file is absent on the server, check upload filters before changing the build command.

## Store app secrets

Use `relay secrets add` for app environment variables that should not live in committed config. See the [CLI reference](../relay-client/README.md) for the command options. Server environment variables such as `RELAY_DATA_DIR` configure `relayd` itself; they are documented in the [agent reference](../relayd/README.md).
