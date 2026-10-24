import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, ArrowUpRight, Check, GitBranch, LockKeyhole, Terminal } from "lucide-react";
import { SitePage, CodePanel } from "@/components/site-shell";
import { HeroLattice } from "@/components/hero-lattice";
import { LanePreview } from "@/components/lane-preview";

export const metadata: Metadata = {
  title: "Relay — deploy on your own server",
  description: "Deploy apps on your own server with framework-aware builds, persistent services, isolated lanes, and rollback controls.",
  alternates: { canonical: "/" },
};

export default function Home() {
  return (
    <SitePage>
      <div className="graphite-home">
        <section className="graphite-hero">
          <div className="graphite-hero-copy">
            <h1>Deploy on<br />your own server.</h1>
            <p>Relay builds and runs your apps, databases, and preview environments on a server you control.</p>
            <div className="graphite-actions"><Link className="site-button" href="/get-started">Deploy your first app <ArrowRight size={15} aria-hidden="true" /></Link><Link className="site-button-secondary" href="/docs">Read the docs</Link></div>
            <Link className="graphite-source-link" href="https://github.com/Relay-CI/Relay">Open source. Self-hosted. <ArrowUpRight size={14} aria-hidden="true" /></Link>
          </div>
          <HeroLattice />
        </section>

        <div className="framework-strip" aria-label="Supported frameworks"><span>Supported frameworks</span><div><strong>Next.js</strong><strong>Django</strong><strong>Rails</strong><strong>Go</strong><strong>FastAPI</strong><strong>.NET</strong><strong>Rust</strong></div><Link href="/features" aria-label="Explore framework support"><ArrowRight size={18} aria-hidden="true" /></Link></div>

        <section className="workspace-section" aria-labelledby="workspace-heading">
          <div className="graphite-section-heading"><h2 id="workspace-heading">See what’s running.</h2><p>Check build status, connected services, and app access in the dashboard. This example shows a dev lane.</p></div>
          <LanePreview />
          <div className="preview-caption"><span>A lane is a project + environment + branch.</span><Link href="/docs#concepts">How lanes work <ArrowRight size={14} aria-hidden="true" /></Link></div>
        </section>

        <section className="capability-section" aria-labelledby="capability-heading">
          <div className="graphite-section-heading"><h2 id="capability-heading">Apps and databases, together.</h2><p>Build apps, connect services, and manage releases from one dashboard.</p></div>
          <div className="feature-mosaic">
            <article className="feature-module feature-module-wide">
              <div className="feature-module-copy"><h3>Add a database<br />to your app.</h3><p>Add PostgreSQL, Redis, MySQL, or MongoDB in relay.json. Relay starts the service and passes its connection URL to your app.</p><Link className="graphite-text-link" href="/docs/services">Add your first service <ArrowRight size={15} aria-hidden="true" /></Link></div>
              <div className="service-map" aria-label="App and RelayDB connected on a private lane network"><div className="service-map-header"><span>my-app</span><span>dev / main</span></div><div className="service-map-nodes"><div><Terminal size={22} aria-hidden="true" /><strong>Application</strong><span>Your framework</span></div><div className="service-map-connection"><span /><small>Private network</small></div><div><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><ellipse cx="12" cy="5" rx="8" ry="3" /><path d="M4 5v14c0 4 16 4 16 0V5M4 12c0 4 16 4 16 0" /></svg><strong>RelayDB</strong><span>Postgres + pooling</span></div></div><div className="service-map-footer"><Check size={12} aria-hidden="true" /><code>DATABASE_URL</code> supplied to your app</div></div>
            </article>
            <article className="feature-module">
              <div className="lane-branches" aria-hidden="true"><div><GitBranch size={15} /><span>main</span><span>prod</span></div><div><GitBranch size={15} /><span>main</span><span>staging</span></div><div><GitBranch size={15} /><span>feature/ui</span><span>preview</span></div></div>
              <div className="feature-module-copy"><h3>Test changes separately.</h3><p>Separate lanes keep production, staging, and branch previews apart. Give each its own route and access policy.</p><Link className="graphite-text-link" href="/docs/previews">Understand lanes <ArrowRight size={15} aria-hidden="true" /></Link></div>
            </article>
            <article className="feature-module">
              <div className="access-graphic" aria-hidden="true"><LockKeyhole size={23} /><div><span>dev / main</span><strong>Relay Login required</strong></div><Check size={16} /></div>
              <div className="feature-module-copy"><h3>Choose who has access.</h3><p>Protect dev and staging with Relay Login. Choose public access, signed links, or an IP allowlist when a lane needs it.</p><Link className="graphite-text-link" href="/docs/previews#lane-lifecycle">Explore app access <ArrowRight size={15} aria-hidden="true" /></Link></div>
            </article>
            <article className="feature-module">
              <div className="feature-module-copy"><h3>Try another build.</h3><p>Run a variant beside the normal release. Give it a share of new visitors, compare traffic and feedback, then promote it or remove it.</p><Link className="graphite-text-link" href="/docs/variants">Use variants and feedback <ArrowRight size={15} aria-hidden="true" /></Link></div>
            </article>
            <article className="feature-module">
              <div className="feature-module-copy"><h3>Run more replicas.</h3><p>Balance requests across up to 16 Docker containers per release on the same server. Add readiness checks and set CPU and memory limits.</p><Link className="graphite-text-link" href="/docs/scaling">Scale an app <ArrowRight size={15} aria-hidden="true" /></Link></div>
            </article>
          </div>
        </section>

        <section className="command-section">
          <div><h2>Deploy from<br />your terminal.</h2><p>Run relay init in your project folder, check the connection, and deploy. Build logs stream to your terminal.</p><Link className="graphite-text-link" href="/docs/cli">View CLI commands <ArrowRight size={15} aria-hidden="true" /></Link></div>
          <CodePanel label="Inside your app folder" code={`relay init\nrelay doctor\nrelay deploy --stream\n\n# Inspect your deployment\nrelay status`} />
        </section>

        <section className="mcp-module"><div className="mcp-mark" aria-hidden="true"><span /><span /><span /></div><div><h2>Use Relay from your editor.</h2><p>Inspect deployments, read logs, and operate lanes from an MCP-aware editor or assistant.</p></div><Link className="site-button-secondary" href="/docs/mcp">Set up MCP <ArrowRight size={15} aria-hidden="true" /></Link></section>

        <section className="graphite-next"><h2>Deploy your first app.</h2><p>Install the agent, connect a project, and deploy.</p><Link className="site-button" href="/get-started">Get started <ArrowRight size={15} aria-hidden="true" /></Link></section>
      </div>
    </SitePage>
  );
}
