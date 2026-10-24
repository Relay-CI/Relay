import Link from "next/link";
import { CodePanel, DocsPageLayout, DocsHero as PageHero, SectionHeading } from "@/components/site-shell";

export const metadata = { title: "Databases and services", alternates: { canonical: "/docs/services" } };

export default function ServicesPage() {
  return (
    <DocsPageLayout
      currentPath="/docs/services"
      hero={
        <PageHero
          eyebrow="Project services"
          title="Add databases and services"
          summary="Declare companion services in relay.json to run PostgreSQL, Redis, MySQL, or MongoDB on your app lane’s private network. Relay starts the services and supplies connection URLs to the app."
          aside={
            <CodePanel
              label="Minimal relay.json"
              code={`{\n  "project": "myapp",\n  "services": [\n    { "name": "db", "type": "postgres", "version": "16" },\n    { "name": "cache", "type": "redis", "version": "7" },\n    { "name": "web", "type": "app" }\n  ]\n}`}
            />
          }
        />
      }
    >
      <section id="managed-option" className="scroll-mt-28">
        <div className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--accent-soft)] px-6 py-5">
          <p className="text-sm leading-6 text-[color:var(--ink)]">
            <strong>Looking for a managed Postgres with connection pooling and encrypted credentials?</strong>{" "}
            Use <code className="font-mono">type: &quot;relaydb&quot;</code> instead of{" "}
            <code className="font-mono">type: &quot;postgres&quot;</code>.{" "}
            <Link href="/docs/relaydb" className="font-semibold underline underline-offset-2">
              See the RelayDB docs →
            </Link>
          </p>
        </div>
      </section>

      <section id="relay-json-shape" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="relay.json shape"
          title="The file describes the project and the named companion services."
          body="`relay.json` is read from the repo root. It is separate from `.relay.json` and `relay.config.json` because it describes the runtime topology of the deployed project, not CLI defaults or buildpack overrides."
        />
        <div className="grid gap-5 lg:grid-cols-3">
          {[
            {
              title: "project",
              body: "A human-readable project identifier used for grouping related services.",
            },
            {
              title: "services",
              body: "A list of named service containers like Postgres, Redis, MySQL, Mongo, and the main app service.",
            },
            {
              title: "name, type, version",
              body: "The service name becomes the hostname on the Docker network. Type and version select the image and default URL wiring.",
            },
          ].map((item) => (
            <article key={item.title} className="site-article-card">
              <h3 className="text-xl font-semibold tracking-[0.01em] text-[color:var(--ink)]">{item.title}</h3>
              <p className="mt-3 text-sm leading-7 text-[color:var(--muted)]">{item.body}</p>
            </article>
          ))}
        </div>
      </section>

      <section id="service-networking" className="scroll-mt-28 grid gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <div className="space-y-5">
          <SectionHeading
            label="Service networking"
            title="Relay creates one shared Docker network per app/env/branch."
            body="The network name follows the deploy slot, so the app container and its companion services can resolve each other by hostname. Relay also injects connection URLs into the app container so you do not have to hand-wire the obvious cases."
          />
          <div className="grid gap-4">
            {[
              ["db", "Reachable from the app as `db` inside the network."],
              ["cache", "Reachable from the app as `cache` inside the network."],
              ["DB_URL", "Injected when a database service exists."],
              ["CACHE_URL", "Injected when a cache service exists."],
            ].map(([name, body]) => (
              <div key={name} className="rounded-[1.5rem] border border-[color:var(--line)] bg-white/75 px-5 py-4">
                <div className="font-mono text-sm font-semibold text-[color:var(--ink)]">{name}</div>
                <p className="mt-1 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
              </div>
            ))}
          </div>
        </div>

        <CodePanel
          label="Typical injected URLs"
          code={`DB_URL=postgres://postgres:postgres@db:5432/app\nCACHE_URL=redis://cache:6379\nMONGO_URL=mongodb://mongo:27017/app`}
          footer="The exact URL shape depends on the service type Relay starts."
        />
      </section>

      <section id="operating-services" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-white/70 p-7">
          <SectionHeading
            label="Operating services"
            title="Operate and inspect services"
            body="Relay keeps these containers near the app because the goal is convenient companion services for previews and self-hosted environments. Persistent named volumes keep database state between container restarts."
          />
          <div className="mt-8 grid gap-4 md:grid-cols-2">
            {[
              ["Bridge network", "Created as relay-<app>-<env>-<branch>."],
              ["Named volumes", "Used for Postgres, MySQL, Redis, and Mongo persistence."],
              ["/api/projects", "Lists projects with live services and connection strings."],
              ["Dashboard services panel", "Shows running companion containers and copyable URLs."],
            ].map(([title, body]) => (
              <div key={title} className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5">
                <div className="font-semibold text-[color:var(--ink)]">{title}</div>
                <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
              </div>
            ))}
          </div>
        </div>

        <CodePanel
          label="Project inventory"
          code={`GET /api/projects\n\n[\n  {\n    "app": "demo",\n    "services": [\n      { "name": "db", "type": "postgres", "connection_url": "postgres://..." }\n    ]\n  }\n]`}
          footer="This endpoint is what the dashboard uses to populate project-level views."
        />
      </section>
    </DocsPageLayout>
  );
}
