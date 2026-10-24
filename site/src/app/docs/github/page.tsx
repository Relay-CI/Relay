import { CodePanel, DocsPageLayout, DocsHero as PageHero, SectionHeading } from "@/components/site-shell";

export const metadata = { title: "GitHub deployments", alternates: { canonical: "/docs/github" } };

export default function GithubPage() {
  return (
    <DocsPageLayout
      currentPath="/docs/github"
      hero={
        <PageHero
          eyebrow="GitHub and webhooks"
          title="Deploy from GitHub"
          summary="Connect a GitHub App for preview and production deployments. Existing integrations can also use a manual signed webhook. Choose the workflow below that matches your setup."
          aside={
            <CodePanel
              label="Webhook endpoint"
              code={`POST https://relay.example.com/api/webhooks/github\nContent-Type: application/json\nX-Hub-Signature-256: sha256=...`}
              footer="Replace the example hostname with your agent's public HTTPS hostname."
            />
          }
        />
      }
    >
      <section id="github-app" className="space-y-6">
        <h2>Connect a repository with a GitHub App</h2>
        <p>Use this path for connected GitHub delivery. Before you begin, configure a public HTTPS dashboard hostname and secret encryption. Relay requires encryption before registering the App.</p>
        <ol className="docs-ordered-list">
          <li>As an owner, configure <code>RELAY_SECRET_KEY</code> in the server environment or use <strong>Server Settings → Security</strong> in the dashboard.</li>
          <li>Open <strong>Dashboard → GitHub</strong> and start the GitHub App setup. Choose the owning organization if needed; leave it empty for your personal account.</li>
          <li>Complete GitHub&apos;s App registration and installation flow. Select the repositories Relay should access.</li>
          <li>Link a Relay app to a repository and choose its production branch.</li>
        </ol>
        <p>The App requests <strong>Contents: read</strong> and <strong>Checks: write</strong>. Relay encrypts its private key and webhook secret in the SQLite Store. Installation tokens are scoped to the repository.</p>
      </section>

      <section id="connected-delivery" className="space-y-6">
        <h2>What happens after a push?</h2>
        <dl className="docs-concepts">
          <div><dt>A non-production branch changes</dt><dd>Relay queues an isolated preview deployment and publishes a <code>Relay Preview</code> Check Run with the app URL and log link.</dd></div>
          <div><dt>A pull request is updated</dt><dd>Relay reuses a matching branch deployment instead of building the same commit twice.</dd></div>
          <div><dt>A merge reaches the production branch</dt><dd>GitHub&apos;s push event queues production. A <code>Relay Production</code> Check Run stays in progress during the health window. If health checks fail, Relay restores the previous slot.</dd></div>
          <div><dt>You need to roll back</dt><dd>The dashboard&apos;s <strong>Rollback instantly</strong> control can abort an active rollout or restore the previous image after the rollout completes.</dd></div>
        </dl>
        <p>App payloads require a valid SHA-256 signature. Delivery IDs prevent repeated processing of the same delivery. Keep encryption keys with your server backups and check GitHub&apos;s delivery response when an event does not create a deployment.</p>
      </section>

      <section id="webhook-flow" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Webhook flow"
          title="Existing integrations: manual webhook"
          body="The manual webhook and personal-token connection remain available for compatibility. For a manual webhook, the agent reads the repository URL and branch from the delivery and matches saved app lane state. A matching per-app secret takes precedence over the global fallback."
        />
        <div className="grid gap-5 lg:grid-cols-4">
          {[
            ["01", "Store repo_url", "Set the repository URL in app config so the agent knows which app owns that repo."],
            ["02", "Add the webhook", "Point GitHub at `/api/webhooks/github` on the Relay agent."],
            ["03", "Verify the signature", "Relay checks the per-app secret when present, otherwise the global fallback secret."],
            ["04", "Start the deploy", "A matching push triggers the same build and rollout path as a manual deploy."],
          ].map(([step, title, body]) => (
            <article key={step} className="site-article-card">
              <div className="text-sm font-semibold tracking-[0.28em] text-[color:var(--accent)]">{step}</div>
              <h3 className="mt-3 text-xl font-semibold tracking-[0.01em] text-[color:var(--ink)]">{title}</h3>
              <p className="mt-2 text-sm leading-7 text-[color:var(--muted)]">{body}</p>
            </article>
          ))}
        </div>
      </section>

      <section id="connecting-a-repo" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-white/70 p-7">
          <SectionHeading
            label="Connecting a repo"
            title="Configure the manual repository mapping"
            body="For this legacy path, store repository metadata per app, environment, and branch. These settings tell the webhook handler which lane should receive a matching push. This does not configure the connected GitHub App workflow above."
          />
          <div className="mt-8 grid gap-4 md:grid-cols-2">
            {[
              ["repo_url", "The Git remote used to match incoming webhooks."],
              ["branch", "The branch this app slot should deploy from."],
              ["env", "Usually preview or prod, but stored explicitly."],
              ["webhook_secret", "Optional per-app secret used instead of the global fallback."],
            ].map(([title, body]) => (
              <div key={title} className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5">
                <div className="font-mono text-sm font-semibold text-[color:var(--ink)]">{title}</div>
                <p className="mt-2 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
              </div>
            ))}
          </div>
        </div>

        <CodePanel
          label="App config payload"
          code={`{\n  "app": "demo",\n  "env": "preview",\n  "branch": "main",\n  "repo_url": "https://github.com/org/repo.git",\n  "webhook_secret": "super-secret"\n}`}
          footer="You can set this via the dashboard or POST /api/apps/config."
        />
      </section>

      <section id="secrets-and-safety" className="scroll-mt-28 grid gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <div className="space-y-5">
          <SectionHeading
            label="Secrets and safety"
            title="Prefer per-app webhook secrets over one global secret."
            body="Per-app secrets make it easier to isolate repositories and rotate one integration at a time. The global RELAY_GITHUB_WEBHOOK_SECRET still works as a fallback, but it should not be your only control if multiple repos share the same agent."
          />
          <div className="rounded-[1.75rem] border border-[color:var(--line)] bg-white/75 p-6">
            <h3 className="text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">Checklist</h3>
            <div className="mt-4 space-y-3 text-sm leading-6 text-[color:var(--muted)]">
              <p>Expose the webhook endpoint only on the agent you intend GitHub to reach.</p>
              <p>Set an explicit webhook secret in the project settings or app config.</p>
              <p>Keep the general Relay API token separate from GitHub webhook auth.</p>
              <p>Use HTTPS and a stable hostname if the agent is reachable from the public internet.</p>
            </div>
          </div>
        </div>

        <CodePanel
          label="GitHub webhook settings"
          code={`URL: https://relay.example.com/api/webhooks/github\nContent type: application/json\nSecret: <per-app-secret>\nEvents: push`}
          footer="Replace the example URL and secret. Push events are used for this manual webhook path."
        />
      </section>
    </DocsPageLayout>
  );
}
