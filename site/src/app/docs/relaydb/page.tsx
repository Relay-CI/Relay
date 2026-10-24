import type { Metadata } from "next";
import { CodePanel, DocsPageLayout, DocsHero as PageHero, SectionHeading } from "@/components/site-shell";
import { Database, KeyRound, Shield, Zap } from "lucide-react";

export const metadata: Metadata = {
  title: "RelayDB",
  description:
    "RelayDB is a Relay-managed PostgreSQL workload with stable encrypted credentials, PgBouncer connection pooling, resource-aware tuning, and lane-scoped durable storage.",
  alternates: {
    canonical: "/docs/relaydb",
  },
};

const profiles = [
  {
    name: "starter",
    memory: "512 MB",
    cpus: "1",
    sharedBuffers: "128 MB",
    maxConns: "50",
    maxClients: "500",
    poolSize: "20",
    trigger: "Host < 8 GB RAM",
  },
  {
    name: "balanced",
    memory: "2 GB",
    cpus: "2",
    sharedBuffers: "512 MB",
    maxConns: "100",
    maxClients: "2000",
    poolSize: "50",
    trigger: "Host 8–32 GB RAM",
  },
  {
    name: "throughput",
    memory: "8 GB",
    cpus: "4",
    sharedBuffers: "2 GB",
    maxConns: "200",
    maxClients: "10 000",
    poolSize: "100",
    trigger: "Host ≥ 32 GB RAM",
  },
];

export default function RelayDBPage() {
  return (
    <DocsPageLayout
      currentPath="/docs/relaydb"
      hero={
        <PageHero
          eyebrow="RelayDB"
          title="Managed PostgreSQL with RelayDB"
          summary="Add a Relay-managed PostgreSQL data workload to a lane. RelayDB provides PgBouncer connection pooling, stable encrypted credentials, readiness checks, and durable storage."
          aside={
            <CodePanel
              label="relay.json — add RelayDB"
              code={`{
  "project": "myapp",
  "services": [
    { "name": "db", "type": "relaydb" },
    { "name": "web", "type": "app" }
  ]
}`}
            />
          }
        />
      }
    >
      <section id="relaydb-vs-postgres" className="scroll-mt-28 grid gap-5 lg:grid-cols-2">
        <article className="site-article-card">
          <div className="inline-flex h-12 w-12 items-center justify-center rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
            <Database className="h-5 w-5" />
          </div>
          <h2 className="mt-5 text-3xl font-semibold tracking-[-0.01em] text-[color:var(--ink)]">
            type: relaydb
          </h2>
          <div className="mt-5 space-y-4 text-sm leading-7 text-[color:var(--muted)]">
            <p>Starts two containers: a tuned PostgreSQL primary and a PgBouncer pooler in transaction mode. The app only ever connects to the pooler.</p>
            <p>Credentials are auto-generated per lane and encrypted at rest with AES-256-GCM. They persist across container restarts and deploys.</p>
            <p>Resource limits, shared_buffers, work_mem, and max_connections are set from a profile that Relay picks based on host memory, or that you pin explicitly.</p>
            <p>Health checks run on both the primary and the pooler before Relay reports the companion ready.</p>
          </div>
        </article>

        <article className="site-article-card lg:translate-y-8">
          <div className="inline-flex h-12 w-12 items-center justify-center rounded-[1.25rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
            <Zap className="h-5 w-5" />
          </div>
          <h2 className="mt-5 text-3xl font-semibold tracking-[-0.01em] text-[color:var(--ink)]">
            type: postgres
          </h2>
          <div className="mt-5 space-y-4 text-sm leading-7 text-[color:var(--muted)]">
            <p>Starts a single plain postgres container with no pooler and no tuning. Use this when you want full control over the image, credentials, and runtime config.</p>
            <p>Credentials are fixed: <code className="font-mono text-xs">relay:relay</code>. No encryption, no per-lane rotation.</p>
            <p>No resource limits applied by Relay. You can add them via the service config.</p>
            <p>Connection URL injected as <code className="font-mono text-xs">{"<NAME>"}_URL</code>. No pooler stands between the app and the database.</p>
          </div>
        </article>
      </section>

      <section id="profiles" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Profiles"
          title="Relay sizes the database container to the host."
          body={`Set \`profile\` to \`auto\`, \`starter\`, \`balanced\`, or \`throughput\`. \`auto\` (the default) reads total host memory and picks the right tier. Pin a profile when you want predictable resource limits regardless of host size.`}
        />
        <div className="grid gap-4 lg:grid-cols-3">
          {profiles.map((p) => (
            <div
              key={p.name}
              className="rounded-[1.5rem] border border-[color:var(--line)] bg-white/70 p-6"
            >
              <div className="font-mono text-sm font-semibold text-[color:var(--ink)]">{p.name}</div>
              <div className="mt-1 text-xs text-[color:var(--muted-strong)]">{p.trigger}</div>
              <div className="mt-5 space-y-2.5">
                {[
                  ["Memory limit", p.memory],
                  ["CPU limit", p.cpus],
                  ["shared_buffers", p.sharedBuffers],
                  ["max_connections", p.maxConns],
                  ["max_client_conn", p.maxClients],
                  ["default_pool_size", p.poolSize],
                ].map(([label, value]) => (
                  <div key={label} className="flex items-center justify-between text-sm">
                    <span className="text-[color:var(--muted)]">{label}</span>
                    <span className="font-mono font-semibold text-[color:var(--ink)]">{value}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <CodePanel
          label="Pin a profile"
          code={`{
  "services": [
    {
      "name": "db",
      "type": "relaydb",
      "profile": "balanced",
      "version": "17"
    }
  ]
}`}
          footer="Omit profile or set it to auto and Relay picks based on host memory at start time."
        />
      </section>

      <section id="connection-contract" className="scroll-mt-28 grid gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <div className="space-y-5">
          <SectionHeading
            label="Connection contract"
            title="Connect with DATABASE_URL"
            body="Relay injects the connection URL into the app container before the first process starts. The URL always points to the PgBouncer pooler, not the Postgres primary. The pooler runs in transaction mode, so statement-level prepared statements require the extended query protocol."
          />
          <div className="grid gap-4">
            {[
              [
                "DATABASE_URL",
                'Injected when the service name is "db". This is the conventional name for the primary database.',
              ],
              [
                "<NAME>_URL",
                "Injected for any other service name. A service named analytics injects ANALYTICS_URL.",
              ],
              [
                "application_name",
                "Set automatically to app_env_branch so pg_stat_activity shows which lane the connection came from.",
              ],
              [
                "sslmode=disable",
                "TLS is off inside the private Docker network. Enable TLS at the Caddy edge if external access is needed.",
              ],
            ].map(([name, body]) => (
              <div
                key={name}
                className="rounded-[1.5rem] border border-[color:var(--line)] bg-white/75 px-5 py-4"
              >
                <div className="font-mono text-sm font-semibold text-[color:var(--ink)]">{name}</div>
                <p className="mt-1 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
              </div>
            ))}
          </div>
        </div>

        <CodePanel
          label="Injected URL shape"
          code={`# service name: db
DATABASE_URL=postgresql://relay:<password>@<app>__db__pool:5432/relay?sslmode=disable&application_name=myapp_prod_main

# service name: analytics
ANALYTICS_URL=postgresql://relay:<password>@<app>__analytics__pool:5432/relay?sslmode=disable&application_name=myapp_prod_main`}
          footer="The pooler container name is the primary name with __pool appended. The password is unique per lane and encrypted in relay.db."
        />
      </section>

      <section id="security" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-white/70 p-7">
          <SectionHeading
            label="Security"
            title="Credentials, encryption, and backups"
            body="Relay generates a random 24-byte password for each lane on first start. It encrypts the password with AES-256-GCM using a 32-byte key stored in data/relaydb.key. The ciphertext is stored in the service_credentials table in relay.db. The plaintext password never appears in logs or environment variables outside the container."
          />
          <div className="mt-8 grid gap-4 md:grid-cols-2">
            {[
              {
                title: "Per-lane credentials",
                body: "Each lane gets its own username, password, and database name. Credentials do not rotate unless the lane is deleted.",
                icon: KeyRound,
              },
              {
                title: "AES-256-GCM encryption",
                body: "Stored password ciphertext is prefixed enc: and is not readable without the relaydb.key file in the data directory.",
                icon: Shield,
              },
              {
                title: "Managed env vars blocked",
                body: "POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, and POSTGRES_INITDB_ARGS cannot be overridden via the service env map.",
                icon: Database,
              },
              {
                title: "data/relaydb.key",
                body: "Created on first boot if absent. Back it up alongside relay.db. Losing it means existing credentials cannot be decrypted.",
                icon: KeyRound,
              },
            ].map((item) => (
              <div
                key={item.title}
                className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5"
              >
                <div className="inline-flex h-9 w-9 items-center justify-center rounded-[1rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] text-[color:var(--accent)]">
                  <item.icon className="h-4 w-4" />
                </div>
                <h3 className="mt-4 font-semibold text-[color:var(--ink)]">{item.title}</h3>
                <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{item.body}</p>
              </div>
            ))}
          </div>
        </div>

        <div className="site-panel">
          <div className="space-y-4">
            <div className="text-xs font-semibold uppercase tracking-[0.22em] text-[color:var(--muted-strong)]">
              Backup checklist
            </div>
            <div className="space-y-3 text-sm leading-6 text-[color:var(--muted)]">
              <p>Back up <code className="font-mono text-xs">data/relay.db</code> — holds the encrypted credentials.</p>
              <p>Back up <code className="font-mono text-xs">data/relaydb.key</code> — required to decrypt them.</p>
              <p>Back up the named Docker volume for each lane (<code className="font-mono text-xs">relay__app__env__branch__svc__db_data</code>).</p>
              <p>Both files and the volume are needed together for a full restore.</p>
            </div>
          </div>
        </div>
      </section>

      <section id="config-reference" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Config reference"
          title="All relay.json fields for RelayDB."
          body="Set these fields inside the service object where type is relaydb. Fields not listed here are ignored for RelayDB services."
        />

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[
            ["type", `"relaydb"`, "Required. Selects the managed Postgres+pooler path."],
            ["name", `"db"`, 'Name becomes the hostname on the Docker network. Use "db" to get DATABASE_URL.'],
            ["version", `"17"`, "Postgres major version. Defaults to 17."],
            ["profile", `"auto"`, "auto, starter, balanced, or throughput. auto picks from host memory."],
            ["host_port", "5433", "Expose the pooler on the host at this port. Useful for external tooling like psql or Postico."],
            ["pooler_image", `"edoburu/pgbouncer:..."`, "Override the PgBouncer image. Also overridable via RELAY_PGBOUNCER_IMAGE."],
          ].map(([field, example, body]) => (
            <div
              key={field}
              className="rounded-[1.5rem] border border-[color:var(--line)] bg-white/75 px-5 py-4"
            >
              <div className="font-mono text-sm font-semibold text-[color:var(--ink)]">{field}</div>
              <div className="mt-1 font-mono text-xs text-[color:var(--muted-strong)]">{example}</div>
              <p className="mt-3 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
            </div>
          ))}
        </div>

        <CodePanel
          label="Full example"
          code={`{
  "project": "myapp",
  "services": [
    {
      "name": "db",
      "type": "relaydb",
      "version": "17",
      "profile": "balanced",
      "host_port": 5433
    },
    { "name": "web", "type": "app" }
  ]
}`}
          footer="host_port lets external tools connect directly to the pooler. The app still connects via DATABASE_URL inside the Docker network."
        />
      </section>
    </DocsPageLayout>
  );
}
