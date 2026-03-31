The Relay Master Plan
Everything discussed, organized into a single buildable roadmap.
add blue and new cahanges to ddocs site
The Three Product Tiers

Tier 1 — Relay OSS (today)
  You run relayd on your own server
  Local admin panel at :8080
  Token auth, single machine, Docker-based
  Always free, always open source

Tier 2 — Relay Cloud Connect (plan 2)
  You still run your own servers
  relayd agents connect to relay.com
  One unified dashboard for all your servers
  Your code and containers never leave your machines
  Free

Tier 3 — Relay Hosting (plan 3)
  Relay owns and operates the servers
  Users push code, Relay handles everything
  Global load balancers, all regions
  Canary + red/green built in
  Persistent containers, no cold starts
  Vercel competitor, free tier
Tier 1 — Fix What Exists (relayd)
Everything needed to go from current state to production-grade.

Code Quality (8 weeks)
Week 1-2 — Split the monolith


relayd/
├── cmd/relayd/main.go
├── internal/
│   ├── buildpack/        one file per buildpack + interface
│   ├── deploy/           pipeline, bluegreen, health checks, prune
│   ├── sync/             session lifecycle, zstd bundle
│   ├── api/              HTTP handlers split by domain
│   ├── store/            SQLite, migrations, queries
│   └── container/        all Docker operations
Week 3 — Replace os/exec Docker with SDK


go get github.com/docker/docker/client
// typed responses, proper streaming, context cancellation
Week 4 — Add context timeouts everywhere


ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
defer cancel()
// every build, every docker call, every db query gets ctx
Week 5-6 — Tests

Unit: buildpack detection (all 13), path validation, webhook signatures
Integration: full deploy pipeline against real Docker
Fuzz: isSafeRelPath against path traversal attempts
Target 70%+ coverage on internal packages
Week 7 — Database migrations


internal/store/migrations/
├── 001_initial_schema.up.sql
├── 002_add_accounts.up.sql
├── 003_add_canary.up.sql
No more blind ALTER TABLE on startup.

Week 8 — Security hardening

Rate limit webhook endpoint per repo (5 deploys/min)
Validate and cap all input lengths
Rotate token on first run, never write to git-tracked file
Add structured logging via log/slog
Canary System (already partially built)
Current behavior is correct and better than traditional canary:


New visitor  → new version  → pinned via session cookie
Returning    → old version  → stays until session expires
Result       → zero maintenance windows, zero UX disruption
What to add:

Per-cohort error rate comparison (new sessions vs old sessions)
Auto-revert: if new-visitor error rate > threshold, stop routing new visitors to new version
Dashboard view: X% of active sessions on new version, trending graph
Manual promote: "move all traffic to new version now"
Manual revert: "send all new visitors back to old version"
Red/Green System

Green = current production  (100% traffic)
Red   = new version         (built, warmed, zero traffic)

Deploy:
  1. Build new image       → Red environment
  2. Start Red containers  → health check passes
  3. Warm Red             → optional synthetic traffic
  4. Cutover              → atomic pointer swap, Green becomes Red
  5. Drain old Red        → 30s graceful, then remove
  6. Rollback             → swap pointer back, instant

vs Vercel:
  Vercel rollback = re-deploy (takes time)
  Relay rollback  = pointer swap (instant)
Tier 2 — Relay Cloud Connect
Self-hosted servers, unified relay.com dashboard.

Account System (2 weeks)

accounts     (id, email, username, password_hash, github_id, role)
sessions     (id, account_id, expires_at)
cli_states   (state, account_id, token, expires_at)  -- device flow
api_tokens   (id, account_id, name, token_hash, last_used)
CLI device flow:


relay login
# → opens browser to relay.com/auth/login?state=abc&cli=true
# → user authenticates
# → CLI polls GET /auth/cli/token?state=abc
# → token saved to ~/.relay/config.json
# → all commands now use this identity
Auth options in order of priority:

GitHub OAuth (primary — no passwords to manage)
Email + password (fallback)
SSO / SAML (Business tier)
WebSocket Agent Bridge (3 weeks)
The core of the unified panel. Agents phone home — no port forwarding needed.


relayd boots in cloud-connected mode
  → opens WebSocket to wss://agent.relay.com/connect
  → authenticates with account JWT
  → relay.com registers server, stores connection
  → stays connected, reconnects on drop

relay.com dashboard loads
  → fetches all registered servers for this account
  → for each server: queries latest state via WebSocket
  → streams live events through the same channel
Agent → relay.com (events):


{ "type": "deploy.started",   "app": "my-site", "id": "rel_01J..." }
{ "type": "deploy.log",       "id": "rel_01J...", "line": "Step 3/8" }
{ "type": "deploy.success",   "id": "rel_01J...", "url": "preview.x.com" }
{ "type": "server.heartbeat", "cpu": 42, "mem": 68, "disk": 34 }
{ "type": "app.state",        "apps": [...] }
relay.com → Agent (commands):


{ "type": "deploy.trigger", "app": "my-site", "branch": "main" }
{ "type": "app.restart",    "app": "my-site" }
{ "type": "app.stop",       "app": "my-site" }
{ "type": "secrets.set",    "app": "my-site", "key": "DB_URL", "value": "..." }
{ "type": "logs.stream",    "id": "rel_01J..." }
Critical requirements:

Agent works fully offline if relay.com unreachable
Events queued locally, replayed when connection restored
Commands acknowledged or rejected with reason
One WebSocket per relayd instance, not per app
Unified Dashboard (2 weeks)

relay.com/dashboard

┌─ Servers ──────────────────────────────────────────┐
│  server-A  NYC    ● online   4 apps   CPU 42%      │
│  server-B  London ● online   2 apps   CPU 18%      │
│  server-C  Home   ○ offline  3 apps   last seen 2h │
└────────────────────────────────────────────────────┘

┌─ All Apps ─────────────────────────────────────────┐
│  my-site       server-A   ● running   preview.x.com│
│  api-service   server-A   ● running                │
│  landing       server-B   ⟳ building               │
│  staging-api   server-C   ○ stopped                │
└────────────────────────────────────────────────────┘
Click any app → live logs stream from agent through relay.com
Trigger deploy → relay.com → WebSocket → agent → docker build → events back

Tier 3 — Relay Hosting
Relay owns the servers. Users just push code.

Infrastructure

relay.com
├── Global load balancers (per region)
│   ├── US-East   (AWS/Hetzner/own hardware)
│   ├── EU-West
│   ├── AP-South
│   └── ...
├── Build fleet      (dedicated build machines, not serving)
├── Serve fleet      (persistent containers, always warm)
├── Edge routing     (canary, red/green, session affinity)
└── Registry         (internal OCI image registry per account)
Why Not Serverless

Serverless (Vercel):           Relay Hosting:
  cold starts on low traffic     always warm, no cold starts
  function timeouts              long-running processes supported
  no WebSockets (or limited)     full WebSocket support
  no background jobs             cron jobs, queues work natively
  no stateful processes          stateful apps fine
  vendor-locked runtime          standard Docker containers
Container Runtime Strategy
Do not build a container runtime from scratch. Use:


Linux servers:
  containerd + runc (battle-tested, OCI-compliant)
  wrap with relay container/ package (Go Docker SDK)
  no Docker Desktop dependency in production

macOS dev machines (relay CLI local builds):
  Apple Virtualization.framework → lightweight Linux VM
  containerd inside the VM
  relay CLI talks to containerd via socket
  feels native, faster than Docker Desktop

Build pipeline:
  BuildKit embedded (open source, Apache 2.0)
  relay's buildpack frontend generates Dockerfiles
  OCI images stored in relay's registry
  delta sync on image layers (extend existing sync system)
Deploy Flow (Tier 3)

Developer:
  git push  or  relay deploy

relay.com receives:
  1. Authenticate request
  2. Route to nearest build region
  3. Sync changed files (delta, zstd)
  4. Detect framework (existing buildpack system)
  5. Build image (BuildKit)
  6. Push to internal registry
  7. Schedule on serve fleet (nearest region to traffic)
  8. Red/Green: start new containers, health check
  9. Canary: route new visitors to new version
  10. Full cutover after stability window
  11. Drain and remove old containers

Developer sees:
  relay deploy --stream
  ✓ synced 8 changed files (delta)
  ✓ detected: Next.js 14 (standalone)
  ✓ build complete (34s)
  ✓ containers healthy (3 regions)
  ✓ canary active (new visitors → v2)
  ✓ preview: https://my-site-git-main.relay.app
Business Model

Free tier:
  3 apps
  1 region
  512MB RAM per app
  relay.app subdomain
  Community support

Pro ($20/month):
  Unlimited apps
  3 regions
  2GB RAM per app
  Custom domains
  Log retention 30 days
  Priority builds

Team ($60/month per seat):
  Everything in Pro
  Team members + RBAC
  Audit logs
  Preview deploy comments on PRs
  Slack/Discord notifications

Business (custom):
  All regions
  Dedicated build machines
  SSO / SAML
  SLA
  Custom contracts
Full Chronological Build Order

PHASE 1 — Foundation (months 1-2)
  ├── Refactor relayd monolith into packages
  ├── Replace os/exec with Docker SDK
  ├── Add context timeouts throughout
  ├── Add database migrations
  ├── Write core tests
  └── Security hardening (rate limits, input validation)

PHASE 2 — Accounts (months 2-3)
  ├── Account + session tables in relayd
  ├── GitHub OAuth flow
  ├── CLI device flow (relay login)
  ├── JWT middleware replacing token auth
  ├── Admin vs user roles
  └── relay.com auth service (mirrors relayd auth, shared JWT secret)

PHASE 3 — Unified Panel (months 3-5)
  ├── relay.com agent gateway (WebSocket server)
  ├── relayd agent client (WebSocket, reconnect, queue)
  ├── Server registration API
  ├── Event streaming (deploys, logs, heartbeats)
  ├── Command routing (deploy, restart, stop)
  └── Unified dashboard MVP (read-only first, then commands)

PHASE 4 — Canary + Red/Green (months 4-5, parallel with phase 3)
  ├── Session affinity routing (cookie-based, already conceptually done)
  ├── Per-cohort error rate tracking
  ├── Auto-revert threshold
  ├── Manual promote/revert controls
  └── Dashboard canary view (% sessions on each version)

PHASE 5 — Relay Hosting MVP (months 5-8)
  ├── Internal OCI registry
  ├── Build fleet setup (dedicated machines)
  ├── Serve fleet setup (persistent containers, 2 regions)
  ├── Global load balancer config
  ├── relay.app subdomain routing
  ├── Free tier with limits enforced
  └── Billing integration (Stripe)

PHASE 6 — Scale + Polish (months 8-12)
  ├── Remaining regions
  ├── Custom domain SSL (Let's Encrypt, auto-renewal)
  ├── PR preview comments (GitHub App)
  ├── Team + RBAC system
  ├── Audit logs
  ├── Prometheus metrics export
  ├── Pro + Team billing tiers
  └── Public launch
What Makes Relay Different From Vercel At Each Tier

Tier 1 (OSS):
  Your server, your Docker, your data
  No vendor, no cost, no limits

Tier 2 (Cloud Connect):
  Your servers, relay.com just shows them
  Data never leaves your machines
  GDPR/compliance-friendly by default

Tier 3 (Hosting):
  Persistent containers (no cold starts)
  Session-affinity canary (better than Vercel's deploy model)
  Instant red/green rollback (Vercel re-deploys, you swap)
  OCI-standard containers (portable, not locked to runtime)
  Open-source agent (auditable, trustable)
The last point is underrated. Vercel is a black box. Relay's agent is open source — enterprises and security teams can read exactly what runs on their machines or exactly what code Relay executes. That trust gap is real and it wins deals.