import Link from "next/link";
import { ArrowRight, Database, HardDrive, Server, TerminalSquare, Waypoints } from "lucide-react";
import { CodePanel, DocsPageLayout, DocsHero as PageHero, SectionHeading } from "@/components/site-shell";

export const metadata = { title: "Architecture", alternates: { canonical: "/docs/architecture" } };

export default function ArchitecturePage() {
  return (
    <DocsPageLayout
      currentPath="/docs/architecture"
      hero={
        <PageHero
          eyebrow="Architecture"
          title="How Relay works"
          summary="The relay CLI uploads changed files from your computer. The relayd agent builds and runs containers on your server. Each lane keeps its own deployment state, routing, and companion workloads."
          aside={
            <CodePanel
              label="Deploy path"
              code={`1. relay loads .relayignore + cache
2. relay computes manifest
3. agent answers diff plan
4. changed files upload
5. lane policy applies defaults
6. runtime.Build() images
7. container swaps + lane URL prints`}
            />
          }
        />
      }
    >
      <section id="split" className="scroll-mt-28 grid gap-5 lg:grid-cols-2">
        <article className="site-article-card">
          <div className="inline-flex h-12 w-12 items-center justify-center rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
            <TerminalSquare className="h-5 w-5" />
          </div>
          <h2 className="mt-5 text-3xl font-semibold tracking-[-0.01em] text-[color:var(--ink)]">
            relay CLI
          </h2>
          <div className="mt-5 space-y-4 text-sm leading-7 text-[color:var(--muted)]">
            <p>Computes the local file manifest, honors .relayignore, and reuses saved hashes when files have not changed.</p>
            <p>Uploads changed files or bundles to the current sync session.</p>
            <p>Targets a lane and sends the deploy request to the agent.</p>
            <p>Runs relay doctor for transport, version, and setup diagnostics before you hit the dashboard.</p>
            <p>Streams logs and prints deploy status to the terminal.</p>
          </div>
        </article>

        <article className="site-article-card lg:translate-y-8">
          <div className="inline-flex h-12 w-12 items-center justify-center rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
            <Server className="h-5 w-5" />
          </div>
          <h2 className="mt-5 text-3xl font-semibold tracking-[-0.01em] text-[color:var(--ink)]">
            relayd agent
          </h2>
          <div className="mt-5 space-y-4 text-sm leading-7 text-[color:var(--muted)]">
            <p>Stores app state and deploy history in SQLite.</p>
            <p>Exposes /api/doctor so the CLI and dashboard can guide DNS, TLS, Docker, socket, data-dir, and webhook setup with live checks.</p>
            <p>Materializes workspaces under the data directory.</p>
            <p>Selects buildpacks, writes Dockerfiles, and routes builds through a pluggable runtime backend.</p>
            <p>Applies lane defaults for production, staging, dev, and preview, then enforces edge access before traffic reaches the app.</p>
            <p>Performs container swaps and keeps one previous image for rollback. Docker is the default backend; Station is available as a per-app experimental alternative.</p>
          </div>
        </article>
      </section>

      <section id="storage" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Storage"
          title="Where Relay stores data"
          body="The agent stores configuration and deployment history in SQLite. Uploaded workspaces, logs, plugins, and database keys live in its data directory."
        />

        <div className="grid gap-5 lg:grid-cols-[1fr_24rem]">
          <div className="rounded-[2rem] border border-[color:var(--line)] bg-white/70 p-7">
            <div className="grid gap-4 md:grid-cols-2">
              {[
                {
                  title: "relay.db",
                  body: "Deploy history, app state, lane policy state, secrets, analytics rows, and project service records. Configuration and analytics queries use separate SQLite handles for the same file.",
                  icon: Database,
                },
                {
                  title: "logs/",
                  body: "One log file per deploy, readable by humans and streamable to the CLI.",
                  icon: HardDrive,
                },
                {
                  title: "workspaces/",
                  body: "Per-app repo snapshots, staging paths, and sync state between deploys.",
                  icon: Waypoints,
                },
                {
                  title: "plugins/",
                  body: "Server-installed buildpack plugin definitions kept under the data directory.",
                  icon: Server,
                },
              ].map((item) => (
                <article
                  key={item.title}
                  className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5"
                >
                  <div className="inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-[color:var(--line)] bg-white text-[color:var(--accent)]">
                    <item.icon className="h-4 w-4" />
                  </div>
                  <h3 className="mt-4 text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">
                    {item.title}
                  </h3>
                  <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{item.body}</p>
                </article>
              ))}
            </div>
          </div>

          <CodePanel
            label="Workspace shape"
            code={`data/
  relay.db
  token.txt
  logs/
    <deployId>.log
  workspaces/
    <app>__<env>__<branch>/
      repo/
      staging/
  plugins/
    buildpacks/`}
          />
        </div>
      </section>

      <section id="runtime-engine" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Runtime engine"
          title="How the agent runs containers"
          body="ContainerRuntime abstracts build, run, stop, network, log, and image operations. Docker is the default backend. The engine is stored per app. Select Station explicitly to use the experimental native runtime."
        />
        <div className="grid gap-5 lg:grid-cols-[1fr_24rem]">
          <div className="rounded-[2rem] border border-[color:var(--line)] bg-white/70 p-7">
            <div className="grid gap-4 md:grid-cols-2">
              {[
                {
                  title: "ContainerRuntime interface",
                  body: "Covers RunDetached, Remove, IsRunning, ContainerIP, PublishedPort, Exec, NetworkConnect, EnsureNetwork, RemoveNetwork, RemoveVolume, Build, RemoveImage, ListImages, and LogStream.",
                  icon: Waypoints,
                },
                {
                  title: "Docker (default)",
                  body: "DockerRuntime implements all interface methods via the Docker CLI. DOCKER_BUILDKIT=1 is set for builds. This is the backend relayd starts with.",
                  icon: Server,
                },
                {
                  title: "Station (experimental — not for production)",
                  body: 'StationRuntime is an experimental native container backend. Set engine to "station" in app config to opt in per app. It supports blue-green session handoff, but rejects partial canary splits. Docker and Station can coexist on the same server; Docker should be the default for production lanes.',
                  icon: HardDrive,
                },
                {
                  title: "Lane policy hooks",
                  body: "Runtime choice is only one part of the deploy contract. Relay also applies lane defaults for routing, traffic mode, retention, and edge access before the rollout starts.",
                  icon: TerminalSquare,
                },
              ].map((item) => (
                <article
                  key={item.title}
                  className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5"
                >
                  <div className="inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-[color:var(--line)] bg-white text-[color:var(--accent)]">
                    <item.icon className="h-4 w-4" />
                  </div>
                  <h3 className="mt-4 text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">
                    {item.title}
                  </h3>
                  <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{item.body}</p>
                </article>
              ))}
            </div>
          </div>
          <CodePanel
            label="Engine selection"
            code={`# Set per-app via /api/apps/config
{ "engine": "docker" }   // default
{ "engine": "station" }  // experimental

# lane policy is evaluated separately
# access_policy, traffic_mode, and routing still come from app state + lane defaults`}
          />
        </div>
      </section>

      <section
        id="boundary"
        className="scroll-mt-28 rounded-[2rem] border border-[color:var(--line)] bg-[color:var(--ink)] p-8 text-[color:var(--paper)] shadow-[0_28px_90px_rgba(24,22,19,0.18)]"
      >
        <div className="grid gap-8 lg:grid-cols-[1fr_auto] lg:items-end">
          <div className="space-y-4">
            <div className="site-eyebrow text-[color:rgba(244,240,232,0.65)] before:bg-[rgba(244,240,232,0.3)]">
              Boundary
            </div>
            <h2 className="text-3xl font-semibold tracking-[-0.01em] sm:text-4xl">
              Builds run on your server.
            </h2>
            <p className="max-w-3xl text-base leading-7 text-[color:rgba(244,240,232,0.72)]">
              The agent builds and runs your apps on the host where you installed it. Keep its data directory and service volumes in your backup plan.
            </p>
          </div>
          <Link
            href="/docs/cli"
            className="inline-flex items-center gap-2 text-sm font-semibold text-[color:var(--paper)] transition-opacity duration-200 hover:opacity-80"
          >
            See the CLI guide
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </section>
    </DocsPageLayout>
  );
}
