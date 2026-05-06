# The Relay Master Plan

Everything discussed, organized into a single buildable roadmap.

## What This Plan Was Missing

The original roadmap had strong product ideas, but it skipped the easiest path to revenue:

- Paid control-plane features before full hosted compute
- A managed self-hosting offer users can buy immediately
- A provider-specific packaging story, especially Hetzner
- Clear monetization beyond "be a Vercel competitor"
- A phase dedicated to billing, metering, support, and services

The biggest missing layer is an intermediate product between:

- "run Relay yourself for free"
- "Relay owns a global hosting platform"

That middle product should be a managed server offer built on Hetzner first.

## The Product Ladder

### Tier 1 - Relay OSS

- You run `relayd` on your own server
- Local admin panel at `:8080`
- Single machine, Docker/container runtime based
- Always free, always open source

### Tier 2 - Relay Cloud Connect

- You still run your own servers
- `relayd` agents connect to `relay.com`
- One unified dashboard for all your servers
- Your code and containers never leave your machines
- Good default for hobbyists and small teams

This should not stay fully free forever. It is the first clean SaaS surface to monetize.

### Tier 2.5 - Relay Managed Nodes (Hetzner First)

- User buys a ready-to-run Relay server
- Relay provisions a Hetzner VM with Relay preinstalled
- DNS, TLS, deploy agent, updates, and backups are handled for them
- Still single-tenant infrastructure per customer
- Much easier to sell than full multi-region hosting

This is the fastest route to profit because:

- infrastructure is simple
- margins can be positive quickly
- customers get a clear outcome
- it avoids the cost and risk of building Tier 3 too early

### Tier 3 - Relay Hosting

- Relay owns and operates the serving platform
- Users push code and Relay handles everything
- Global load balancers, multiple regions
- Canary and red/green built in
- Persistent containers, no cold starts
- Direct Vercel competitor

## Tier 1 - Fix What Exists (`relayd`)

Everything needed to go from current state to production-grade.

### Code Quality (8 weeks)

#### Week 1-2 - Split the monolith

```text
relayd/
|- cmd/relayd/main.go
|- internal/
|  |- buildpack/        one file per buildpack + interface
|  |- deploy/           pipeline, bluegreen, health checks, prune
|  |- sync/             session lifecycle, zstd bundle
|  |- api/              HTTP handlers split by domain
|  |- store/            SQLite, migrations, queries
|  `- container/        all Docker operations
```

#### Week 3 - Replace `os/exec` Docker usage with SDK

```go
go get github.com/docker/docker/client
// typed responses, proper streaming, context cancellation
```

#### Week 4 - Add context timeouts everywhere

```go
ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
defer cancel()
// every build, every docker call, every db query gets ctx
```

#### Week 5-6 - Tests

- Unit: buildpack detection, path validation, webhook signatures
- Integration: full deploy pipeline against real Docker
- Fuzz: `isSafeRelPath` against path traversal attempts
- Target `70%+` coverage on internal packages

#### Week 7 - Database migrations

```text
internal/store/migrations/
|- 001_initial_schema.up.sql
|- 002_add_accounts.up.sql
`- 003_add_canary.up.sql
```

No more blind `ALTER TABLE` on startup.

#### Week 8 - Security hardening

- Rate limit webhook endpoint per repo
- Validate and cap all input lengths
- Rotate token on first run, never write to git-tracked files
- Add structured logging via `log/slog`

### Canary system

Current behavior is directionally correct and better than traditional canary:

```text
New visitor  -> new version  -> pinned via session cookie
Returning    -> old version  -> stays until session expires
Result       -> zero maintenance windows, zero UX disruption
```

What to add:

- Per-cohort error-rate comparison: new sessions vs old sessions
- Auto-revert if new-visitor error rate crosses threshold
- Dashboard view: percent of active sessions on new version
- Manual promote: move all traffic now
- Manual revert: send all new visitors back to old version

### Red/Green system

```text
Green = current production (100% traffic)
Red   = new version        (built, warmed, zero traffic)
```

Deploy flow:

1. Build new image to Red
2. Start Red containers and pass health checks
3. Warm Red with optional synthetic traffic
4. Cut over with atomic pointer swap
5. Drain old containers gracefully
6. Roll back by swapping pointer back

Compared with Vercel:

- Vercel rollback = re-deploy
- Relay rollback = pointer swap

## Tier 2 - Relay Cloud Connect

Self-hosted servers, unified `relay.com` dashboard.

### Account system (2 weeks)

```text
accounts   (id, email, username, password_hash, github_id, role)
sessions   (id, account_id, expires_at)
cli_states (state, account_id, token, expires_at)
api_tokens (id, account_id, name, token_hash, last_used)
```

CLI device flow:

```text
relay login
-> opens browser to relay.com/auth/login?state=abc&cli=true
-> user authenticates
-> CLI polls GET /auth/cli/token?state=abc
-> token saved locally
-> all commands now use this identity
```

Auth options in priority order:

- GitHub OAuth first
- Email + password fallback
- SSO / SAML for business tier

### WebSocket agent bridge (3 weeks)

The core of the unified panel. Agents phone home so users do not need port forwarding.

```text
relayd boots in cloud-connected mode
-> opens WebSocket to wss://agent.relay.com/connect
-> authenticates with account JWT
-> relay.com registers server
-> stays connected and reconnects on drop
```

Dashboard behavior:

```text
relay.com dashboard loads
-> fetches all registered servers for this account
-> queries state through the WebSocket
-> streams live events through the same channel
```

Agent to Relay events:

```json
{ "type": "deploy.started",   "app": "my-site", "id": "rel_01J..." }
{ "type": "deploy.log",       "id": "rel_01J...", "line": "Step 3/8" }
{ "type": "deploy.success",   "id": "rel_01J...", "url": "preview.x.com" }
{ "type": "server.heartbeat", "cpu": 42, "mem": 68, "disk": 34 }
{ "type": "app.state",        "apps": [] }
```

Relay to agent commands:

```json
{ "type": "deploy.trigger", "app": "my-site", "branch": "main" }
{ "type": "app.restart",    "app": "my-site" }
{ "type": "app.stop",       "app": "my-site" }
{ "type": "secrets.set",    "app": "my-site", "key": "DB_URL", "value": "..." }
{ "type": "logs.stream",    "id": "rel_01J..." }
```

Critical requirements:

- Agent works fully offline if `relay.com` is unreachable
- Events queue locally and replay on reconnect
- Commands are acknowledged or rejected with reason
- One WebSocket per `relayd` instance, not per app

### Unified dashboard (2 weeks)

First version should show:

- All servers
- All apps across servers
- Online/offline health
- Live logs
- Trigger deploy, restart, stop, rollback

## Tier 2.5 - Relay Managed Nodes (Hetzner First)

This is the missing commercial bridge between self-hosting and full Relay hosting.

### Product definition

Relay sells a ready-to-run server with:

- Hetzner VM provisioned automatically
- `relayd` preinstalled
- container runtime preinstalled
- reverse proxy and TLS configured
- optional automatic backups
- optional monitoring and alerting
- updates managed by Relay

The customer experience should be:

```text
relay create-server --provider hetzner
-> authenticate with Relay
-> choose region, VM size, and project name
-> Relay provisions the machine
-> Relay installs relayd, runtime, proxy, TLS
-> server appears in Cloud Connect
-> user deploys first app
```

### Delivery options

#### Option A - Bring your own Hetzner account

- user provides Hetzner API token
- Relay provisions inside their account
- lowest risk to launch
- best first version

#### Option B - Relay-managed Hetzner account

- Relay provisions the machine under Relay billing
- customer pays Relay a single monthly invoice
- better margin and simpler UX
- adds support and billing complexity

#### Option C - Golden image / one-click bootstrap

- prebuilt image, snapshot, or cloud-init template
- can be used for self-serve installs
- also works as a fallback if deeper API integration slips

### What must be built for this tier

- Provider abstraction for VM provisioning
- Hetzner implementation first
- Cloud-init installer for `relayd`, proxy, TLS, firewall
- `relay doctor` bootstrap checks
- DNS and custom-domain workflow
- Node backups and restore workflow
- Node update channel: stable, candidate
- Billing for setup fee and monthly management fee

### Why Hetzner first

- Good price/performance for early infrastructure users
- Strong fit for European indie developers and agencies
- Simple enough to validate the managed-node business
- Lets Relay learn operations before running a full hosting fleet

### What to charge

Example model:

- One-time setup fee
- Monthly management fee per node
- Infrastructure passed through at cost plus margin, or bundled into a flat plan
- Paid add-ons for backups, monitoring, extra storage, premium support

This tier can be profitable well before Tier 3 exists.

## Tier 3 - Relay Hosting

Relay owns the serving platform. Users just push code.

### Infrastructure

```text
relay.com
|- global load balancers per region
|- build fleet       (dedicated build machines)
|- serve fleet       (persistent containers, always warm)
|- edge routing      (canary, red/green, session affinity)
`- registry          (internal OCI registry per account)
```

Suggested regional rollout:

- US East
- EU West
- AP South later

### Why not serverless

```text
Serverless (Vercel)            Relay Hosting
----------------------------   ------------------------------
cold starts                    always warm
function timeouts              long-running processes
limited WebSockets             full WebSocket support
weak background jobs           cron jobs and queues
stateless bias                 stateful apps can work
locked runtime                 standard containers
```

### Container runtime strategy

Do not build a runtime from scratch.

Linux servers:

- `containerd` + `runc`
- Relay wrapper around runtime operations
- no Docker Desktop dependency in production

macOS dev machines:

- Apple Virtualization framework
- lightweight Linux VM
- `containerd` inside the VM
- Relay CLI talks to the runtime socket

### Build pipeline

- BuildKit embedded
- Relay buildpack frontend generates Dockerfiles
- OCI images stored in Relay registry
- delta sync on image layers using the existing sync approach

### Deploy flow

Developer:

```text
git push
or
relay deploy --stream
```

Relay receives:

1. Authenticate request
2. Route to nearest build region
3. Sync changed files with delta compression
4. Detect framework
5. Build image
6. Push to internal registry
7. Schedule on serve fleet
8. Start Red/Green rollout
9. Start canary
10. Full cutover after stability window
11. Drain old containers

Developer sees:

```text
synced changed files
detected framework
build complete
containers healthy
canary active
preview URL ready
```

## How Relay Makes Money

Relay should not rely on only one revenue stream.

### 1. Cloud Connect subscriptions

Charge for the control plane, not just compute.

Paid features:

- multiple users
- RBAC
- audit logs
- longer log retention
- PR previews and GitHub comments
- SSO / SAML
- premium support

### 2. Managed nodes on Hetzner

Charge for:

- setup
- monthly node management
- backups
- monitoring
- support
- upgrade handling

This is likely the first meaningful revenue stream.

### 3. Full Relay Hosting

Charge for:

- RAM and CPU allocation
- build minutes
- bandwidth
- storage
- regions
- custom domains

### 4. Professional services

Charge for:

- migrations from Vercel, Render, Railway, or raw VPS setups
- production hardening
- private onboarding
- custom deployment workflows
- enterprise rollout help

### 5. Support and SLA plans

Charge for:

- response-time guarantees
- shared Slack/Discord support
- on-call escalation
- managed incident support

### 6. Marketplace revenue

Later, Relay can take a percentage from:

- paid buildpack plugins
- templates
- deployment recipes
- integrations

### 7. Agency / reseller plans

Agencies should be able to manage many client nodes and bill clients on top.

Charge for:

- white-label dashboard options
- client isolation
- reseller margin
- seat bundles

## Suggested Pricing Shape

These are planning numbers, not final public pricing.

### Cloud Connect

- Free: solo use, limited servers, short retention
- Pro: paid workspace or paid owner tier
- Team: collaboration, RBAC, audits, notifications
- Business: SSO, SLA, contracts

### Managed Nodes

- Starter node: small single-tenant VM with Relay preinstalled
- Growth node: larger managed VM with backups and monitoring
- Agency node bundle: multiple managed nodes, shared billing

### Full Hosting

- Free tier with small limits
- Pro tier with more RAM, regions, custom domains
- Team tier with collaboration and governance
- Business tier with dedicated capacity and support

## Fastest Path To Profit

Build order should follow revenue, not technical elegance alone.

1. Finish a stable OSS core
2. Ship Cloud Connect login and unified dashboard
3. Add paid Cloud Connect team features
4. Launch managed Hetzner nodes
5. Sell migrations and support
6. Only then build full Relay Hosting

This sequence matters because:

- Cloud Connect proves product demand
- Managed nodes prove users will pay Relay
- Services bring cash before platform scale
- Tier 3 becomes a funded expansion, not a blind bet

## Revised Build Order

### Phase 1 - Foundation (months 1-2)

- Refactor `relayd` monolith into packages
- Replace `os/exec` with SDK/runtime abstraction
- Add context timeouts throughout
- Add database migrations
- Write core tests
- Security hardening

### Phase 2 - Accounts (months 2-3)

- Account + session tables in `relayd`
- GitHub OAuth flow
- CLI device flow with `relay login`
- JWT middleware replacing token auth
- Admin vs user roles
- `relay.com` auth service

### Phase 3 - Unified Panel (months 3-5)

- `relay.com` agent gateway
- `relayd` agent client with reconnect and queue
- Server registration API
- Event streaming
- Command routing
- Unified dashboard MVP

### Phase 4 - Canary + Red/Green (months 4-5, parallel)

- Session-affinity routing
- Per-cohort error tracking
- Auto-revert threshold
- Manual promote/revert controls
- Dashboard canary view

### Phase 5 - Revenue Before Full Hosting (months 5-6)

- Stripe billing foundation
- Subscription plans for Cloud Connect
- Usage/metering model for logs, seats, retention
- Hetzner provisioning flow
- Cloud-init installer and golden image
- Managed-node setup and update system
- Backups, monitoring, restore path

### Phase 6 - Managed Nodes Launch (months 6-7)

- Launch Relay Managed Nodes on Hetzner
- Add support workflows and incident runbooks
- Add migration service offer
- Add agency/reseller account support

### Phase 7 - Relay Hosting MVP (months 7-10)

- Internal OCI registry
- Build fleet setup
- Serve fleet setup in 2 regions
- Global load balancer config
- `relay.app` subdomain routing
- Free-tier limits
- Hosting billing

### Phase 8 - Scale + Polish (months 10-12)

- More regions
- Custom-domain SSL automation
- PR preview comments
- Team + RBAC polish
- Audit logs
- Metrics export
- Pro + Team billing refinement
- Public launch

## What Makes Relay Different

### Tier 1 - OSS

- Your server, your runtime, your data
- No vendor, no mandatory spend, no artificial limits

### Tier 2 - Cloud Connect

- Your servers, Relay only coordinates them
- Data stays on your machines
- Good default for compliance-sensitive teams

### Tier 2.5 - Managed Nodes

- Dedicated server per customer
- Much simpler and more trustworthy than opaque multi-tenant PaaS
- Faster onboarding than raw self-hosting
- Better economics than building global hosting too early

### Tier 3 - Hosting

- Persistent containers with no cold starts
- Session-affinity canary
- Instant red/green rollback
- OCI-standard containers
- Open-source agent

That last point matters. Vercel is a black box. Relay can win trust by being inspectable at the agent layer even when the control plane and hosted product become commercial.
