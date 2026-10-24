# Replicas, resources, and operations

[All documentation](README.md) · [Variants and feedback](variants.md)

## Scale a Docker lane

Run in your app folder:

```sh
relay scale 3 --app my-app --env prod --branch main
relay deploy --app my-app --env prod --branch main --replicas 3
```

Relay supports 1–16 replicas per release on the same host. The edge proxy balances requests across the replica pool. It does not provision other machines or replicate databases.

Scaling a running lane queues a deployment of its existing image when the size needs to change. Replacement replicas must become ready before traffic switches. Allow capacity for both releases during rollout. A stopped lane keeps the desired count for its next start. Existing variants keep their current containers until redeployed.

Use **Resources** in the dashboard or MCP `scale_app` for the same operation. The API is `POST /api/apps/scale` with `app`, `env`, `branch`, `replicas`, and optional `apply` (default true). Set `apply: false` to save a count without queuing a rollout. Follow the returned `deploy_id` if present; saved and running replica counts can differ after a failed resize.

## Configure readiness

Merge into `relay.config.json`, using a real health endpoint in your app:

```json
{
  "service_port": 3000,
  "replicas": 3,
  "readiness_path": "/health"
}
```

Every replacement replica must return HTTP 2xx from the configured path before its slot takes traffic. The path must be absolute, without a query or fragment. Custom readiness paths require Docker.

## CPU and memory

In **Resources**, Manual mode accepts CPU values such as `0.5` and memory values such as `512m`. Limits apply per container at its next start. Saving does not change existing containers.

MCP `set_app_config`, or `POST /api/apps/config`, accepts `resource_mode`, `cpu_limit`, and `mem_limit` together with the lane identity. Automatic mode ignores saved manual limits but does not dynamically autoscale or tune them. The server's host-sized default memory cap may still apply when no explicit cap is used. See [small-host settings](small-host-memory.md).

## Find logs and measurements

| Task | Dashboard | MCP / API |
| --- | --- | --- |
| Read deployment output | Build logs / deployment details | `get_deploy_logs`; `GET /api/logs/<id>` |
| Discover app, replica, variant, proxy, or service logs | Runtime logs | `get_runtime_targets`; `GET /api/runtime/logs/targets` |
| Follow running output | Runtime logs | `GET /api/runtime/logs/stream` is live SSE, not finite JSON |
| Read requests, errors, latency, visitors, and bandwidth | Analytics | `get_analytics`; `GET /api/analytics` |
| Inspect container and daemon usage | Server admin → Operations | Owner-only `get_admin_ops`; `GET /api/admin/ops` |
| Inspect edge and active-slot health | Runtime/deployment views | `get_status`; `GET /api/status` |
| Diagnose Docker, storage, DNS, TLS, or proxy setup | Server settings | `get_doctor`; `relay doctor` |

Analytics accepts `period=24h`, `7d`, or `30d`, lane fields, and an optional `variant` name or `control`. It measures proxy traffic; direct container requests are excluded. Visitor IDs rotate daily and are separate from variant-assignment cookies.

`/api/analytics` and `/api/deploys/rollout-abort` are HTTP/HTTPS-only routes. Other routes above are also exposed on the local agent socket. MCP returns a clear connection error if you use a network-only tool with `RELAY_SOCKET`.

MCP `abort_rollout` restores the previous slot only for an active monitored canary rollout with a standby slot. Use ordinary rollback for a completed release, or `cancel_deploy` for an in-progress build.
