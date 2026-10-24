import {
  ClipboardList,
  Database,
  GitBranch,
  Globe2,
  Hash,
  Layers3,
  Lock,
  Palette,
  PlugZap,
  RotateCcw,
  Server,
  Shield,
  Users,
  Workflow,
} from "lucide-react";
import type { Metadata } from "next";
import { HoloCard, Reveal, SpotlightCard } from "@/components/home-effects";
import { CodePanel, DocsPageLayout, DocsHero as PageHero, SectionHeading } from "@/components/site-shell";

export const metadata: Metadata = {
  title: "relayd Agent",
  description:
    "Server docs for relayd covering lanes, edge access, user accounts, secrets encryption, build tracking, APIs, runtime routing, and the settings that drive app and dashboard hosts.",
  alternates: {
    canonical: "/docs/agent",
  },
};

const environmentCards = [
  {
    title: "Core process settings",
    body: "RELAY_ADDR controls the TCP listener, RELAY_DATA_DIR controls the persistent state root, and RELAY_SOCKET controls the local Unix socket path.",
    icon: Server,
  },
  {
    title: "Diagnostics endpoints",
    body: "GET /api/version reports relayd build metadata and Station runtime version details. GET /api/doctor reports data-dir, Docker, socket, ACME, dashboard host, managed subdomain DNS, Caddy proxy, and webhook health so the CLI and dashboard can guide setup from one source of truth.",
    icon: Database,
  },
  {
    title: "Auth and browser policy",
    body: "User accounts take precedence when any exist. RELAY_TOKEN seeds legacy single-token mode as a fallback. RELAY_CORS_ORIGINS gates browser origins. The dashboard stores the session in an HttpOnly relay_session cookie, and relay-login edge access reuses that identity model.",
    icon: Shield,
  },
  {
    title: "Secrets encryption",
    body: "Set RELAY_SECRET_KEY to any string and all new secrets are stored as AES-256-GCM ciphertext. Unencrypted secrets written before the key was set are still readable during migration.",
    icon: Lock,
  },
  {
    title: "Mutation and webhook policy",
    body: "RELAY_ENABLE_PLUGIN_MUTATIONS stays false by default, while RELAY_GITHUB_WEBHOOK_SECRET acts as the global webhook fallback when no per-app secret matches. Remote plugin installs stay HTTPS-only and can be SHA256-pinned by the caller.",
    icon: PlugZap,
  },
  {
    title: "Lane hosts and rollout behavior",
    body: "RELAY_BASE_DOMAIN lets Relay derive managed lane hosts, RELAY_DASHBOARD_HOST gives the admin its own hostname behind the global proxy, and rollout timing is controlled by RELAY_ROLLOUT_READY_TIMEOUT_SECONDS and RELAY_ROLLOUT_DRAIN_SECONDS.",
    icon: Globe2,
  },
  {
    title: "Edge signing and app access",
    body: "RELAY_EDGE_SIGNING_SECRET signs share links for signed-link lanes. If unset, relayd derives the edge signing secret from RELAY_SECRET_KEY or the API token. dev and staging default to relay-login unless app state overrides the access policy.",
    icon: Workflow,
  },
  {
    title: "Lane retention worker",
    body: "dev and preview lanes can carry expires_at state derived from lane policy retention. relayd refreshes that timestamp on deploy and on manual starts or restarts, then stops expired lanes from a background worker.",
    icon: Layers3,
  },
  {
    title: "Themes and appearance",
    body: "The Appearance section in Server Settings lets operators pick one of six built-in presets or supply a custom CSS block that replaces all dashboard styles. The chosen theme persists in server config and loads via /api/public/theme before the login screen so the branding is consistent across authenticated and unauthenticated pages.",
    icon: Palette,
  },
  {
    title: "Upload limits",
    body: "RELAY_MAX_UPLOAD_BYTES caps the size of any single file accepted by the sync protocol. Unset by default, so the limit is whatever the OS allows.",
    icon: Layers3,
  },
];

const apiCards = [
  {
    title: "/api/variants and /api/apps/scale",
    body: "GET /api/variants reads variant weights, metrics, and feedback. POST changes a weight, promotes or removes a variant, or resets visitor assignments. POST /api/apps/scale sets 1–16 replicas for a Docker lane and can queue a resize deployment. Both routes work over HTTP and the local socket.",
    icon: Layers3,
  },
  {
    title: "/api/analytics and runtime logs",
    body: "GET /api/analytics returns traffic and resource reports for 24h, 7d, or 30d; it is HTTP-only. Runtime log discovery uses GET /api/runtime/logs/targets; /api/runtime/logs/stream follows a selected target as SSE. Runtime-log routes work over both transports.",
    icon: Database,
  },
  {
    title: "/api/auth/*",
    body: "GET /api/auth/session returns current auth state including username and role. POST /api/auth/login authenticates with username and password and sets a session cookie. POST /api/auth/setup creates the first owner account. DELETE /api/auth/session logs out. POST /api/auth/cli/exchange trades a short-lived code for a bearer token.",
    icon: Shield,
  },
  {
    title: "/api/users",
    body: "GET lists all user accounts. POST creates a new user with username, password, and role. DELETE removes a user and revokes sessions. PATCH changes a user's role.",
    icon: Users,
  },
  {
    title: "/api/audit",
    body: "GET returns recent audit log entries including actor, action, target, and detail. Requires authentication.",
    icon: ClipboardList,
  },
  {
    title: "/api/deploys",
    body: "GET /api/deploys lists deploy records including build_number, deployed_by, commit_message, and source. GET /api/deploys/<id> returns a single record. POST /api/deploys can queue a manual deploy from a saved workspace or git branch, and POST /api/deploys/rollback queues a rollback to the previous image.",
    icon: Hash,
  },
  {
    title: "/api/apps/start | stop | restart",
    body: "POST each endpoint with app, env, and branch to directly control the running container without triggering a new build.",
    icon: Server,
  },
  {
    title: "/api/apps/config",
    body: "Stores repo_url, mode, traffic_mode, host_port, service_port, public_host, access_policy, ip_allowlist, expires_at, engine, and webhook_secret for one app lane. engine accepts docker or station.",
    icon: Layers3,
  },
  {
    title: "/api/apps/signed-link",
    body: "Generates a temporary signed share URL for lanes using access_policy=signed-link. relayd signs relay_exp and relay_sig from the lane identity plus the edge signing secret.",
    icon: Workflow,
  },
  {
    title: "/api/edge/authz",
    body: "Edge proxies call this endpoint before serving app traffic. relayd resolves the lane, loads app state, and enforces public, relay-login, signed-link, or ip-allowlist access centrally.",
    icon: Shield,
  },
  {
    title: "/api/promotions",
    body: "Lists promotion requests and creates new ones. Relay stores the source image, target lane, approval requirement, target deploy id, rollback id, and health result as the promotion proceeds.",
    icon: GitBranch,
  },
  {
    title: "/api/promotions/approve",
    body: "Owners approve a queued promotion request here. Once approved, relayd queues a production deploy from the staged image and watches post-promote health before marking success.",
    icon: ClipboardList,
  },
  {
    title: "/api/server/config",
    body: "Stores server-level routing settings and appearance config. Gets and sets base_domain, dashboard_host, acme_disabled, theme_name, and theme_css. Responses also include the current doctor report so the dashboard can guide DNS and TLS setup beside the writable config.",
    icon: Globe2,
  },
  {
    title: "/api/doctor",
    body: "Returns the server-side operator report used by relay doctor and the dashboard setup guide. Checks include data-dir writability, secret-key presence, Docker, socket, ACME listener, dashboard host, managed subdomain DNS, Caddy proxy health, and webhook reachability.",
    icon: Workflow,
  },
  {
    title: "/api/public/theme",
    body: "GET returns the active theme_name and theme_css without requiring authentication. The dashboard fetches this on startup so the selected appearance applies before the login screen renders, then re-applies after session validation.",
    icon: Palette,
  },
  {
    title: "/api/apps/companions",
    body: "Lists, creates, and deletes managed companion services. /api/apps/companions/restart restarts a named companion in place.",
    icon: Workflow,
  },
  {
    title: "/api/sync/pull/<app>/<env>/<branch>",
    body: "GET streams the current server workspace as a tar archive. Used by relay pull to sync the server-side repo back to the local machine. Responds with X-Workspace-Version containing the current repo hash.",
    icon: RotateCcw,
  },
  {
    title: "/api/plugins/buildpacks",
    body: "GET lists installed buildpack plugins. POST installs a new one when mutation is enabled. POST /api/plugins/buildpacks/install-url installs from a verified HTTPS URL, and DELETE removes a plugin.",
    icon: PlugZap,
  },
  {
    title: "/api/admin/ops",
    body: "Returns the owner-only operations snapshot used by the Operations tab: live Docker CPU, memory, storage, container counts, and latest-deploy versus previous-deploy comparisons across apps and lanes.",
    icon: Database,
  },
  {
    title: "/api/events and /api/projects",
    body: "Provide the dashboard with live project and deploy state via SSE plus a current inventory endpoint.",
    icon: Database,
  },
  {
    title: "/api/webhooks/github",
    body: "Matches repo_url against app_state rows, verifies GitHub signatures, extracts the head commit message, assigns a build number, and queues the same deploy path the CLI uses.",
    icon: GitBranch,
  },
];

export default function RelaydPage() {
  return (
    <DocsPageLayout
      currentPath="/docs/agent"
      hero={
        <PageHero
          eyebrow="relayd agent"
          title="Configure the relayd server"
          summary="Run relayd on the machine that hosts your apps. This reference covers server settings, authentication, persistent storage, diagnostics, and the HTTP APIs used by the CLI and dashboard."
          aside={
            <CodePanel
              label="Common env"
              code={`RELAY_ADDR=:8080
RELAY_DATA_DIR=./data
RELAY_SOCKET=./data/relay.sock
RELAY_SECRET_KEY=strong-passphrase
RELAY_CORS_ORIGINS=https://dashboard.example.com
RELAY_ENABLE_PLUGIN_MUTATIONS=false
RELAY_BASE_DOMAIN=relay.example.com
RELAY_DASHBOARD_HOST=admin.relay.example.com
RELAY_EDGE_SIGNING_SECRET=replace-me
RELAY_ACME_EMAIL=ops@example.com
RELAY_ACME_ADDR=:80`}
              footer="If RELAY_TOKEN is empty and no users exist, the dashboard prompts to create the first owner account. relayd --version, /api/version, and /api/doctor feed the Server tab and relay doctor."
            />
          }
        />
      }
    >
      <section id="environment-model" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Server settings"
          title="Server settings"
          body="The agent stays fairly small on purpose. Environment variables control listen addresses, auth behavior, secrets encryption, lane host generation, dashboard host routing, upload quotas, rollout timing, diagnostics, and language image defaults."
        />
        <div className="grid gap-5 md:grid-cols-2">
          {environmentCards.map((item, index) => (
            <Reveal key={item.title} delay={index * 70}>
              <SpotlightCard className="h-full rounded-[1.6rem]">
                <HoloCard className="h-full rounded-[1.6rem]">
                  <article className="site-article-card h-full">
                    <div className="inline-flex h-12 w-12 items-center justify-center rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
                      <item.icon className="h-5 w-5" />
                    </div>
                    <h3 className="mt-5 text-xl font-semibold tracking-[0.01em] text-[color:var(--ink)]">
                      {item.title}
                    </h3>
                    <p className="mt-3 text-sm leading-7 text-[color:var(--muted)]">{item.body}</p>
                  </article>
                </HoloCard>
              </SpotlightCard>
            </Reveal>
          ))}
        </div>
      </section>

      <section id="auth-surface" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-[rgba(7,20,38,0.78)] p-7">
          <SectionHeading
            label="Authentication"
            title="User accounts, roles, Relay Login, and legacy token mode can all coexist."
            body="When at least one user account exists, the server validates session tokens from the users table. When no users exist, it falls back to legacy RELAY_TOKEN or token.txt mode. Socket requests are always trusted via filesystem ACL, and relay-login edge access reuses the same authenticated session."
          />
          <div className="mt-8 grid gap-4 md:grid-cols-2">
            {[
              ["User accounts", "POST /api/auth/setup creates the first owner. POST /api/auth/login issues a session. Roles: owner, deployer, viewer."],
              ["CLI login flow", "relay login opens a browser tab. After login the server generates a short-lived code, the CLI exchanges it at /api/auth/cli/exchange, and saves the bearer token to ~/.relay-state.json."],
              ["Bearer token", "Authorization: Bearer <token> is accepted for user sessions and legacy tokens alike."],
              ["Header token", "X-Relay-Token is what relay.js sends for HTTP transport. Checked before the session cookie."],
              ["Dashboard cookie", "POST /api/auth/login and /api/auth/setup set an HttpOnly relay_session cookie for the browser UI."],
              ["Lane access", "relay-login lanes accept an authenticated Relay session before proxying the app. signed-link and ip-allowlist are enforced through the same edge auth boundary."],
              ["Signed shares", "signed-link lanes can now mint time-bounded share URLs through the dashboard or /api/apps/signed-link without handing operators raw edge secret material."],
              ["Socket auth", "Requests over relay.sock are treated as local auth and rely on socket file permissions instead of a token."],
              ["Same-origin checks", "State-changing browser requests require same-origin unless the origin is explicitly allowed by RELAY_CORS_ORIGINS."],
            ].map(([title, body], index) => (
              <Reveal key={title} delay={index * 40}>
                <SpotlightCard className="rounded-[1.5rem]">
                  <div className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5">
                    <div className="font-semibold text-[color:var(--ink)]">{title}</div>
                    <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
                  </div>
                </SpotlightCard>
              </Reveal>
            ))}
          </div>
        </div>

        <CodePanel
          label="Auth endpoints"
          code={`GET    /api/auth/session
POST   /api/auth/login
POST   /api/auth/setup
DELETE /api/auth/session
GET    /api/auth/me
POST   /api/auth/cli/exchange

GET    /api/users
POST   /api/users
DELETE /api/users/:id
PATCH  /api/users/:id

GET    /api/version
GET    /api/edge/authz

Authorization: Bearer <token>
X-Relay-Token: <token>`}
          footer="GET /api/auth/session returns the current authenticated user session, or legacy-mode metadata when the server still runs without accounts."
        />
      </section>

      <section id="operational-shape" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-[rgba(7,20,38,0.78)] p-7">
        <SectionHeading
          label="Data and storage"
          title="Persistent data and backups"
          body="Deploy history, user accounts, secrets, runtime state, and analytics rows all live in relay.db. If you need to answer what happened, you can usually do it with files in data/ and a SQLite shell."
        />
          <div className="mt-8 grid gap-4 md:grid-cols-2">
            {[
              ["relay.db", "Deploy records, app_state, sync sessions, encrypted app secrets, user accounts, user sessions, audit log, project_services, lane policy state, promotion records, and analytics rows. Control-plane and analytics paths now use separate SQLite handles against the same file."],
              ["token.txt", "Persisted API token when RELAY_TOKEN is not supplied explicitly. Only used when no user accounts exist."],
              ["logs/", "Per-deploy log files that back relay logs and the dashboard."],
              ["workspaces/", "Per app/env/branch repo and staging directories used by the sync protocol. relay pull streams a tar of the repo/ subdirectory."],
              ["plugins/buildpacks/", "Installed JSON buildpack plugins loaded ahead of built-in buildpacks."],
              ["relay.sock", "Optional local API socket that mirrors the API without serving the UI."],
            ].map(([title, body], index) => (
              <Reveal key={title} delay={index * 40}>
                <SpotlightCard className="rounded-[1.5rem]">
                  <div className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5">
                    <div className="font-mono text-sm font-semibold text-[color:var(--ink)]">{title}</div>
                    <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
                  </div>
                </SpotlightCard>
              </Reveal>
            ))}
          </div>
        </div>

        <CodePanel
          label="Data layout"
          code={`data/
  relay.db
  token.txt
  logs/
  workspaces/
    <app>__<env>__<branch>/
      repo/
      staging/
  plugins/
    buildpacks/
  relay.sock`}
          footer="relay.db is where deploy history, app state, users, audit data, and service records live."
        />
      </section>

      <section id="control-surface" className="scroll-mt-28 grid gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <div className="space-y-5">
          <SectionHeading
            label="HTTP API"
            title="HTTP API reference"
            body="The agent API covers deployments, variants, scaling, lane settings, services, users, and measurements. routes.go defines which endpoints are available on HTTP and the local socket."
          />
          <div className="grid gap-4">
            {apiCards.map((item, index) => (
              <Reveal key={item.title} delay={index * 60}>
                <SpotlightCard className="rounded-[1.5rem]">
                  <div className="rounded-[1.5rem] border border-[color:var(--line)] bg-[rgba(8,24,44,0.7)] px-5 py-4">
                    <div className="inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
                      <item.icon className="h-4 w-4" />
                    </div>
                    <h3 className="mt-4 text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">
                      {item.title}
                    </h3>
                    <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{item.body}</p>
                  </div>
                </SpotlightCard>
              </Reveal>
            ))}
          </div>
        </div>

        <div className="space-y-5">
          <CodePanel
            label="App config payload"
            code={`{
  "app": "demo",
  "env": "dev",
  "branch": "main",
  "repo_url": "https://github.com/org/repo.git",
  "project_root": "apps/api",
  "build_context": "apps/api",
  "dockerfile": "apps/api/Dockerfile",
  "mode": "traefik",
  "traffic_mode": "edge",
  "host_port": 3005,
  "service_port": 3000,
  "public_host": "demo-dev-a1b2c3d4.relay.example.com",
  "access_policy": "relay-login",
  "expires_at": 1776332200000,
  "ip_allowlist": "",
  "engine": "docker",
  "webhook_secret": "super-secret"
}`}
            footer="mode is validated as port or traefik. traffic_mode is validated as edge or session. access_policy is validated as public, relay-login, signed-link, or ip-allowlist. project_root, build_context, and dockerfile must stay inside the repo."
          />
          <CodePanel
            label="Deploy record"
            code={`{
  "id": "f3a2...",
  "build_number": 42,
  "deployed_by": "alice",
  "commit_message": "fix: auth middleware",
  "source": "promote",
  "app": "demo",
  "env": "staging",
  "branch": "main",
  "status": "success",
  "commit_sha": "a1b2c3d...",
  "created_at": "...",
  "started_at": "...",
  "ended_at": "...",
  "image_tag": "...",
  "preview_url": "https://..."
}`}
            footer="The field name preview_url is still used on deploy records even though the server now handles more than preview lanes."
          />
          <CodePanel
            label="Promotion record"
            code={`{
  "id": "promo_...",
  "app": "demo",
  "source_env": "staging",
  "source_branch": "main",
  "source_image": "relay/demo:staging-main-...",
  "target_env": "prod",
  "target_branch": "main",
  "status": "running",
  "approval_required": true,
  "requested_by": "alice",
  "approved_by": "owner-user",
  "target_deploy_id": "deploy_...",
  "health_status": "healthy"
}`}
            footer="Promotion records tie approval, target deploy id, health, and rollback together so the dashboard can show the full production handoff."
          />
          <CodePanel
            label="Build and runtime image overrides"
            code={`RELAY_NODE_IMAGE=node:22
RELAY_NODE_RUN_IMAGE=node:22-slim
RELAY_GO_IMAGE=golang:1.22
RELAY_GO_RUN_IMAGE=debian:bookworm-slim
RELAY_PY_IMAGE=python:3.12
RELAY_PY_RUN_IMAGE=python:3.12-slim
RELAY_JAVA_BUILD_IMAGE=eclipse-temurin:21-jdk
RELAY_JAVA_RUN_IMAGE=eclipse-temurin:21-jre
RELAY_DOTNET_SDK_IMAGE=mcr.microsoft.com/dotnet/sdk:8.0
RELAY_DOTNET_ASPNET_IMAGE=mcr.microsoft.com/dotnet/aspnet:8.0
RELAY_RUST_IMAGE=rust:1.78
RELAY_RUST_RUN_IMAGE=debian:bookworm-slim
RELAY_CC_IMAGE=gcc:13
RELAY_CC_RUN_IMAGE=debian:bookworm-slim
RELAY_NGINX_IMAGE=nginx:alpine`}
            footer="Only set the overrides you need. Defaults are pinned in relayd."
          />
        </div>
      </section>

      <section id="runtime-engine" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-[rgba(7,20,38,0.78)] p-7">
          <SectionHeading
            label="Runtime engine"
            title="Container runtimes"
            body="ContainerRuntime covers build, run, remove, inspect, exec, network, volume, image, and log operations. Docker is the default. Station is available as a per-app experimental alternative. Both backends can run on the same server at the same time."
          />
          <div className="mt-8 grid gap-4 md:grid-cols-2">
            {[
              ["ContainerRuntime interface", "Covers RunDetached, Remove, IsRunning, ContainerIP, PublishedPort, Exec, NetworkConnect, EnsureNetwork, RemoveNetwork, RemoveVolume, Build, RemoveImage, ListImages, and LogStream."],
              ["DockerRuntime (default)", "Runs builds and containers through the Docker CLI with BuildKit enabled. Docker is the default engine."],
              ["StationRuntime (station)", "Experimental native runtime. Set engine to station in the app config to use it for that lane."],
              ["Station constraints", "Station is experimental. It supports blue-green session handoff but rejects partial canary splits; Docker remains the production runtime with the full rollout feature set."],
            ].map(([title, body], index) => (
              <Reveal key={title} delay={index * 40}>
                <SpotlightCard className="rounded-[1.5rem]">
                  <div className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5">
                    <div className="font-semibold text-[color:var(--ink)]">{title}</div>
                    <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
                  </div>
                </SpotlightCard>
              </Reveal>
            ))}
          </div>
        </div>

        <CodePanel
          label="Engine config"
          code={`POST /api/apps/config
{
  "engine": "docker"
}

POST /api/apps/config
{
  "engine": "station"
}

// Station is experimental
// partial canary traffic is rejected
// use Docker for production rollout parity`}
        />
      </section>
    </DocsPageLayout>
  );
}
