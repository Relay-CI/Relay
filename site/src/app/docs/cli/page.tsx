import Link from "next/link";
import {
  Command,
  FileCode2,
  FolderCog,
  KeyRound,
  LogIn,
  Network,
  RotateCcw,
  ScrollText,
  Unplug,
  Zap,
} from "lucide-react";
import { CodePanel, DocsPageLayout, InlineLink, DocsHero as PageHero, SectionHeading } from "@/components/site-shell";

const commandGroups = [
  {
    title: "Project setup",
    body: "Set project defaults with relay init, or run any command without a .relay.json and the interactive wizard starts automatically. The wizard prompts for connection details, app name, and environment, then writes the file. If an existing slot is already deployed under the same app/env/branch, the CLI warns and asks before overwriting.",
    icon: FolderCog,
    code: `relay init --socket ./data/relay.sock --app demo --env dev
relay deploy`,
  },
  {
    title: "Deploy and inspect",
    body: "Deploy with sync-first uploads, then inspect the latest slot with status, logs, list, and projects. Build logs stream by default; pass --no-stream to suppress.",
    icon: Zap,
    code: `relay deploy --host-port 3005
relay pull                    # sync server workspace back to local
relay status
relay logs <deploy-id>
relay logs <deploy-id> --no-stream
relay list --app demo --limit 10
relay projects`,
  },
  {
    title: "Control the slot",
    body: "Rollback or directly start, stop, and restart the running container for one app, env, and branch.",
    icon: RotateCcw,
    code: `relay rollback
relay stop
relay start
relay restart`,
  },
  {
    title: "Version and updates",
    body: "Check relay, relayd, and station versions from one command, then update agent binaries in place when a new release lands. Docker stays the default runtime; station is opt-in.",
    icon: Command,
    code: `relay version
relay agent status
relay agent update`,
  },
  {
    title: "Doctor and setup checks",
    body: "Check your connection and server setup with relay doctor. relay doctor checks the selected transport, local socket access, Docker reachability on same-machine installs, the agent health endpoint, version reporting, and the server-side doctor report that powers guided DNS and TLS setup.",
    icon: Network,
    code: `relay doctor
relay doctor --socket ./data/relay.sock
relay doctor --url https://relay.example.com --token $RELAY_TOKEN`,
  },
  {
    title: "Auth",
    body: "Log in through a browser flow to get a bearer token saved to ~/.relay-state.json, or log out to clear the saved session. Legacy token mode still works via RELAY_TOKEN or .relay.json.",
    icon: LogIn,
    code: `relay login     # opens browser -> saves token
relay logout    # clears saved session`,
  },
  {
    title: "Secrets and plugins",
    body: "Manage app secrets and buildpack plugins from the CLI. Search the catalog or install a plugin from a local file or verified HTTPS URL. Use the dashboard to create signed links and approve promotions.",
    icon: KeyRound,
    code: `relay secrets list
relay secrets add --key DATABASE_URL --value postgres://...
relay plugin list
relay plugin search astro
relay plugin install ./astro-static.json
relay plugin install-url https://example.com/plugins/astro-static.json --sha256 <hex>`,
  },
  {
    title: "Replicas and variants",
    body: "Scale the current Docker release or deploy another build as a variant in the same lane. Variant weights assign new visitors; existing assignments remain sticky. Use the Variants and Replicas guides for requirements and verification.",
    icon: Zap,
    code: `relay scale 3 --app my-app --env prod --branch main
relay deploy --app my-app --env prod --branch main --variant checkout --weight 10
relay variants list --app my-app --env prod --branch main --period 24h
relay variants feedback --app my-app --env prod --branch main
relay variants weight checkout 25 --app my-app --env prod --branch main
relay variants promote checkout --app my-app --env prod --branch main
relay variants rm checkout --app my-app --env prod --branch main
relay variants reshuffle --app my-app --env prod --branch main`,
  },
];

const transportModes = [
  {
    title: "Socket transport",
    body: "Use --socket or RELAY_SOCKET when the CLI runs on the same machine as relayd. No token is required because the socket is protected by filesystem permissions.",
    icon: Unplug,
    code: `{
  "socket": "./data/relay.sock",
  "app": "demo",
  "env": "dev",
  "branch": "main"
}`,
  },
  {
    title: "HTTP transport",
    body: "Use --url plus --token, or RELAY_URL and RELAY_TOKEN, when the CLI runs remotely. The client sends X-Relay-Token on each request.",
    icon: Network,
    code: `{
  "url": "http://127.0.0.1:8080",
  "token": "YOUR_TOKEN",
  "app": "demo",
  "env": "staging",
  "branch": "main"
}`,
  },
];

const deployStages = [
  {
    step: "01",
    title: "Start a sync session",
    body: "relay deploy calls /api/sync/start, sends the saved workspace version plus a local fingerprint hint, receives a session_id, and prepares a staging area on the agent.",
    icon: Command,
  },
  {
    step: "02",
    title: "Hash, diff, upload",
    body: "The CLI walks the workspace, applies .relayignore on top of built-in excludes, reuses cached hashes when size and mtime match, asks /api/sync/plan what changed, uploads only needed files, and posts deletes for stale paths.",
    icon: FileCode2,
  },
  {
    step: "03",
    title: "Finish and stream",
    body: "The final /api/sync/finish call triggers the build, lane policy resolution, and rollout. With --stream, the CLI follows /api/logs/stream/<id> until a terminal status arrives.",
    icon: ScrollText,
  },
];

export const metadata = { title: "CLI commands", alternates: { canonical: "/docs/cli" } };

export default function RelayPage() {
  return (
    <DocsPageLayout
      currentPath="/docs/cli"
      hero={
        <PageHero
          eyebrow="relay CLI reference"
          title="CLI commands"
          summary="Use relay to deploy apps, inspect logs, manage secrets, and check your server connection. Run project commands from your app folder; the CLI reads connection and lane defaults from .relay.json."
          aside={
            <CodePanel
              label="First local run"
              code={`relay init --socket ./data/relay.sock --app demo --env dev
relay deploy --host-port 3005
relay pull          # sync server workspace back
relay status
relay projects`}
              footer="If .relay.json is missing and stdin is a TTY, any command can start the setup wizard automatically. dev lanes default to Relay Login at the edge."
            />
          }
        />
      }
    >
      <p>For experiments, see <Link className="docs-text-link" href="/docs/variants">variants and feedback</Link>. For replica counts, CPU, memory, and readiness checks, see <Link className="docs-text-link" href="/docs/scaling">replicas and resource limits</Link>.</p>
      <section id="command-groups" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Command groups"
          title="Commands by task"
          body="Set up a project, deploy changes, inspect logs, or manage an app. Run commands from your app folder to use its saved connection settings."
        />
        <div className="grid gap-5 md:grid-cols-2">
          {commandGroups.map((group) => (
            <article key={group.title} className="site-article-card space-y-4">
              <div className="inline-flex h-12 w-12 items-center justify-center rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
                <group.icon className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-xl font-semibold tracking-[0.01em] text-[color:var(--ink)]">
                  {group.title}
                </h3>
                <p className="mt-2 text-sm leading-7 text-[color:var(--muted)]">{group.body}</p>
              </div>
              <CodePanel label={group.title} code={group.code} />
            </article>
          ))}
        </div>
      </section>

      <section id="transport-modes" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Transport modes"
          title="Use the socket locally. Use HTTP remotely."
          body="The command set is the same either way. Only the transport and auth model change."
        />
        <div className="grid gap-5 lg:grid-cols-2">
          {transportModes.map((mode) => (
            <article key={mode.title} className="site-article-card space-y-4">
              <div className="inline-flex h-12 w-12 items-center justify-center rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
                <mode.icon className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-xl font-semibold tracking-[0.01em] text-[color:var(--ink)]">
                  {mode.title}
                </h3>
                <p className="mt-2 text-sm leading-7 text-[color:var(--muted)]">{mode.body}</p>
              </div>
              <CodePanel label={mode.title} code={mode.code} />
            </article>
          ))}
        </div>
      </section>

      <section id="deploy-path" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="space-y-8">
          <SectionHeading
            label="Deploy path"
            title="What happens when you deploy"
            body="The client does not tar the whole repo by default. It computes local state, filters with .relayignore, reuses cached hashes when possible, asks the agent what changed, uploads only the diff, then triggers the build, lane policy resolution, and container swap."
          />
          <div className="grid gap-4 md:grid-cols-3">
            {deployStages.map((stage) => (
              <div
                key={stage.step}
                className="rounded-[1.5rem] border border-[color:var(--line)] bg-[rgba(8,24,44,0.7)] p-5"
              >
                <div className="flex items-center justify-between gap-3">
                  <span className="text-sm font-semibold tracking-[0.26em] text-[color:var(--accent)]">
                    {stage.step}
                  </span>
                  <stage.icon className="h-4 w-4 text-[color:var(--accent)]" />
                </div>
                <h3 className="mt-4 text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">
                  {stage.title}
                </h3>
                <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{stage.body}</p>
              </div>
            ))}
          </div>

          <div className="grid gap-5 lg:grid-cols-2">
            <CodePanel
              label="Deploy flags"
              code={`relay deploy \\
  --dir . \\
  --mode port \\
  --host-port 3005 \\
  --service-port 3000 \\
  --public-host demo-staging.relay.example.com \\
  --install-cmd "npm ci" \\
  --build-cmd "npm run build" \\
  --start-cmd "node server.js" \\
  --no-stream`}
              footer="Build logs stream by default. Pass --no-stream to suppress live output and just print the log ID. mode is validated by the agent as port or traefik."
            />
            <div className="rounded-[1.75rem] border border-[color:var(--line)] bg-[rgba(8,24,44,0.7)] p-6">
              <h3 className="text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">
                Before you deploy
              </h3>
              <div className="mt-4 space-y-3 text-sm leading-6 text-[color:var(--muted)]">
                <p>env can target production, staging, dev, or preview. The compiled CLI default is still preview when you do not set one.</p>
                <p>traffic_mode and access_policy exist on the server, but they are stored through app config and lane policy, not as relay deploy flags.</p>
                <p>preview_url comes back from the deploy record after rollout, even when the lane is staging or dev. The field name stayed the same for compatibility.</p>
                <p>Choose the runtime in the setup wizard. Docker is the default.</p>
                <p>.relayignore is project-local and only affects sync uploads. Built-in excludes still apply so node_modules, git state, and Relay metadata stay out of the deploy manifest.</p>
                <p>The CLI saves a local workspace fingerprint, so a stale saved base version does not force a pull when your local repo is clearly newer and has diverged.</p>
                <p>status, list, and projects are read-only views over the same deploy and app state tables the dashboard uses.</p>
              </div>
            </div>
          </div>
        </div>

        <CodePanel
          label="Streaming log flow"
          code={`GET /api/logs/stream/<deploy-id>

event: message
data: {"message":"selected buildpack: node-next"}

event: deploy-status
data: {"status":"running"}`}
          footer="Streaming is on by default. The client renders progress lines in place and exits with the final deploy status. Use --no-stream on either relay deploy or relay logs to get static output instead."
        />
      </section>

      <section id="config-resolution" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-[rgba(7,20,38,0.78)] p-7">
          <SectionHeading
            label="Config resolution"
            title="Flags win, then project config, then environment variables."
            body="resolveDeployArgs and resolveTransport both follow the same layered model, which keeps debugging straightforward."
          />
          <div className="mt-8 grid gap-4">
            {[
              ["1", "CLI flags", "Explicit flags like --socket, --url, --token, --app, and --branch override everything else for one invocation."],
              ["2", ".relay.json", "Project-local defaults written by relay init or the interactive setup wizard."],
              ["3", "Environment variables", "RELAY_SOCKET, RELAY_URL, RELAY_TOKEN, RELAY_APP, RELAY_ENV, and RELAY_BRANCH."],
              ["4", "Compiled defaults", "url defaults to http://127.0.0.1:8080, env to preview, and branch to main. preview remains the CLI fallback, even though the server supports four first-class lanes."],
              ["5", "Build hints", "relay.config.json is read for install_cmd, build_cmd, start_cmd, service_port, readiness_path, and optional monorepo fields such as project_root, build_context, and dockerfile when deploying. readiness_path is a Docker-only HTTP 2xx gate."],
              ["6", "Sync filters", ".relayignore augments Relay's built-in excludes for deploy manifests. Hash reuse is local-only and keyed from saved workspace state so unchanged files do not get re-hashed every deploy."],
            ].map(([step, title, body]) => (
              <div
                key={step}
                className="grid gap-3 rounded-[1.4rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5 sm:grid-cols-[56px_1fr]"
              >
                <div className="text-sm font-semibold tracking-[0.26em] text-[color:var(--accent)]">
                  {step}
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">
                    {title}
                  </h3>
                  <p className="mt-1 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <CodePanel
          label=".relay.json"
          code={`{
  "socket": "./data/relay.sock",
  "app": "demo",
  "env": "dev",
  "branch": "main",
  "dir": "."
}`}
          footer="When no config exists and stdin is interactive, relay.js launches the setup wizard instead of failing immediately."
        />
      </section>

      <section id="recent-additions" className="scroll-mt-28">
        <div className="rounded-[2.4rem] border border-[color:var(--line)] bg-[rgba(7,20,38,0.78)] px-7 py-10 sm:px-10 sm:py-12">
          <div className="grid gap-8 lg:grid-cols-[1fr_auto] lg:items-end">
            <div className="max-w-3xl space-y-4">
              <div className="site-eyebrow">Other CLI features</div>
              <h2 className="text-3xl font-semibold tracking-[-0.01em] sm:text-4xl text-[color:var(--ink)]">
                Other CLI features
              </h2>
              <div className="grid gap-3 sm:grid-cols-2">
                {[
                  ["Environments and branches", "The CLI can target production, staging, dev, or preview, while the server applies lane policy defaults for routing, retention, and promotion."],
                  ["Edge access defaults", "dev and staging default to relay-login on the server. The CLI only picks the lane; the edge policy is enforced centrally by relayd."],
                  ["Browser login flow", "relay login opens a browser tab, you authenticate once, and the bearer token is saved to ~/.relay-state.json. relay logout clears it."],
                  ["relay pull", "Downloads the current server workspace as a tar archive and writes it to the local directory. Useful when the server has diverged or after pulling on another machine."],
                  ["Workspace version check", "relay deploy still sends base_version with each sync, but it also sends a local fingerprint hint. If the server workspace is ahead and your local repo has diverged too, Relay can continue with the newer local content instead of reflexively forcing pull or force."],
                  ["relay doctor", "Check the connection, authentication, agent version, and server diagnostics. Local installs also check Docker access."],
                  [".relayignore and cached manifests", "Use .relayignore to exclude files from uploads. Relay reuses file hashes when size and modification time are unchanged."],
                  ["Interactive setup wizard", "Any command can bootstrap .relay.json when config is missing and stdin is a TTY."],
                  ["Socket-first local auth", "The CLI can skip tokens entirely on the same machine by targeting relay.sock."],
                  ["Docker default engine", "Docker is the default runtime. Choose station explicitly to use the experimental native runtime."],
                  ["Promotion approvals", "Promotion requests, owner approvals, signed-link generation, and lane expiry visibility live in the dashboard because those workflows depend on persistent lane state rather than one-shot deploy flags."],
                  ["Version visibility", "relay version reports CLI, installed binaries, latest release, and server version data from /api/version when reachable."],
                  ["status, list, and projects", "Read deploy history and project inventory without opening the dashboard. list accepts --limit to cap results."],
                  ["Secrets and plugin admin", "Manage app secrets and buildpack plugins directly from the CLI, including catalog search and verified HTTPS installs with optional SHA256 pinning."],
                  ["Streaming on by default", "relay deploy and relay logs stream build output without any flag. Pass --no-stream to suppress live output and print the log ID instead."],
                ].map(([title, body]) => (
                  <div key={title} className="rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-4">
                    <h4 className="font-semibold text-[color:var(--ink)]">{title}</h4>
                    <p className="mt-1 text-sm leading-5 text-[color:var(--muted)]">{body}</p>
                  </div>
                ))}
              </div>
            </div>
            <div className="flex flex-wrap gap-3">
              <InlineLink href="/docs/agent">Read the agent reference</InlineLink>
              <InlineLink href="/docs/plugins">Read the plugin guide</InlineLink>
            </div>
          </div>
        </div>
      </section>
    </DocsPageLayout>
  );
}
