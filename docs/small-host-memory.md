# Running Relay on Small Hosts (2 GB RAM)

Relay sizes itself to the machine automatically. On a 2 GB instance (e.g. GCP
e2-small) the defaults below apply with no configuration.

## What relayd does automatically

- **Daemon memory limit** — relayd sets a Go soft memory limit of 1/4 of
  physical RAM (clamped to 192–768 MB) and collects garbage more eagerly on
  hosts with ≤ 4 GB RAM. The daemon's RSS stays flat instead of drifting into
  hundreds of MB.
- **App container memory caps** — apps without an explicit memory limit get a
  default `--memory` cap of 45% of host RAM (clamped to 256–4096 MB), so one
  leaky app can spill to swap and restart instead of taking the whole host
  down with it.
- **Node build heap sizing** — `npm run build` / `next build` / `vite build`
  run with `NODE_OPTIONS=--max-old-space-size` derived from host RAM
  (half of RAM minus 256 MB, clamped to 384–2048 MB). On 2 GB hosts that is
  768 MB, which leaves room for Docker, relayd, and the apps already running.
- **Build deprioritization** — Node build steps run under `nice -n 10`, so a
  deploy never starves the apps already serving traffic of CPU.
- **Build caching** — npm/pnpm/yarn stores and `.next/cache` persist across
  builds via BuildKit cache mounts, and unchanged inputs reuse the previous
  image entirely, so repeat deploys skip the expensive work.
- **Housekeeping** — every 12 hours relayd deletes deploy logs older than
  30 days and relay-built images beyond the newest 3 per lane (current and
  rollback images are never touched), so disks don't fill up and evict the
  build cache.
- **Low-memory warning** — when a build starts on a host with ≤ 2 GB RAM and
  no swap, the deploy log prints a warning with the one-liner to add swap.

## Observability

The Operations page in the dashboard shows the daemon's own footprint (RSS,
Go heap, soft limit, host RAM/swap) with an hour of history, sourced from
`/api/admin/ops` (`daemon` section). Set `RELAY_PPROF=1` to expose
owner-authenticated Go profiling endpoints under `/debug/pprof/`.

## Overrides

| Variable | Effect |
| --- | --- |
| `RELAY_NODE_BUILD_HEAP_MB` | Explicit V8 heap cap for Node builds (MB). |
| `RELAY_NODE_BUILD_MEMORY_GUARD=0` | Disable the Node build heap cap entirely. |
| `RELAY_BUILD_NICE=0` | Run Node build steps at normal CPU priority. |
| `RELAY_APP_MEM_LIMIT_MB` | Default app container memory cap (MB); `0` disables. |
| `RELAY_GOMEMLIMIT_MB` | Explicit soft memory limit for the relayd daemon (MB). |
| `GOMEMLIMIT` / `GOGC` | Standard Go runtime knobs; when set, relayd does not override them. |
| `RELAY_AUTO_SWAP=1` | Create and enable a swapfile at startup (root, Linux, no existing swap). |
| `RELAY_AUTO_SWAP_MB` / `RELAY_AUTO_SWAP_PATH` | Swapfile size (default 2048) and location (default `/swapfile`). |
| `RELAY_LOG_RETENTION_DAYS` | Deploy log retention (default 30); `0` disables pruning. |
| `RELAY_IMAGE_RETENTION_PER_LANE` | Built images kept per lane (default 3); `0` disables pruning. |
| `RELAY_PPROF=1` | Enable owner-gated `/debug/pprof/` endpoints. |

## Recommended: add swap once per host

A small swapfile makes OOM kills effectively impossible during builds and
costs nothing when idle:

```sh
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```
