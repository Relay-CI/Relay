import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { DocsHero, DocsPageLayout } from "@/components/site-shell";

export const metadata: Metadata = {
  title: "Documentation",
  description: "Learn how Relay works, deploy your first app, and find guides for configuration, databases, routing, and server operations.",
  alternates: { canonical: "/docs" },
};

const guides = [
  ["Deploy your first app", "Install Relay and connect your project to a server.", "/get-started"],
  ["Use the CLI", "Deploy, check status, read logs, and manage secrets.", "/docs/cli"],
  ["Configure your server", "Set up relayd, authentication, persistent storage, and APIs.", "/docs/agent"],
  ["Add a database or service", "Run PostgreSQL, Redis, MySQL, or MongoDB alongside an app.", "/docs/services"],
  ["Use RelayDB", "Add managed PostgreSQL with connection pooling and stable credentials.", "/docs/relaydb"],
  ["Set up domains and app access", "Understand ports, lane URLs, login requirements, and sharing.", "/docs/previews"],
  ["Test a variant", "Assign a share of visitors, compare results, and collect feedback.", "/docs/variants"],
  ["Scale an app", "Run replicas, add readiness checks, and set resource limits.", "/docs/scaling"],
  ["Read logs and analytics", "Inspect running containers, traffic, errors, and server usage.", "/docs/operations"],
  ["Deploy from GitHub", "Connect a repository and trigger deployments from webhooks.", "/docs/github"],
  ["Add framework support", "Install or write a buildpack plugin.", "/docs/plugins"],
  ["Connect an AI tool", "Configure the MCP server for your editor or assistant.", "/docs/mcp"],
  ["Understand the architecture", "Follow what happens between your CLI, agent, and containers.", "/docs/architecture"],
];

export default function DocsPage() {
  return (
    <DocsPageLayout currentPath="/docs" hero={<DocsHero title="Relay documentation" summary="Deploy apps on your own server. Start with your first deployment, then use the guides below to configure and operate it." actions={<Link className="site-command site-command--accent" href="/get-started">Deploy your first app <ArrowRight size={16} aria-hidden="true" /></Link>} />}>
      <section id="starting-points">
        <h2>What do you want to do?</h2>
        <div className="docs-guide-list">
          {guides.map(([title, description, href]) => <Link key={href} href={href}><strong>{title}</strong><span>{description}</span><ArrowRight size={16} aria-hidden="true" /></Link>)}
        </div>
      </section>
      <section id="concepts">
        <h2>Understand the basics</h2>
        <p className="mt-4">Relay has two main parts: a CLI on your computer and an agent on the machine that runs your apps.</p>
        <dl className="docs-concepts">
          <div><dt>CLI (<code>relay</code>)</dt><dd>The command you run in your app folder. It uploads changed files and asks the agent to build and deploy them.</dd></div>
          <div><dt>Agent (<code>relayd</code>)</dt><dd>The server process that builds images, manages containers, stores deployment history, and serves the dashboard. Docker is the default container runtime.</dd></div>
          <div><dt>Lane</dt><dd>An isolated deployment target, identified by project, environment, and branch. For example, <code>my-app / staging / main</code> is a different lane from <code>my-app / prod / main</code>. Supported environments are <code>preview</code>, <code>dev</code>, <code>staging</code>, and <code>prod</code>.</dd></div>
          <div><dt>Variant</dt><dd>An alternate build that shares a lane with its normal release (control). Each variant can receive a chosen share of new visitors.</dd></div>
          <div><dt>Replica</dt><dd>Another container running the same release in a lane. The edge proxy balances requests across replicas on the same server.</dd></div>
          <div><dt>Buildpack</dt><dd>A planning adapter that detects your framework and chooses how to build and run the app. Use build configuration when you need to override those choices.</dd></div>
          <div><dt>Companion</dt><dd>A stateful or background workload on the same private network as an application lane. RelayDB is a Relay-managed PostgreSQL data workload with pooling and durable storage scoped to one lane.</dd></div>
        </dl>
      </section>
      <section id="config-model">
        <h2>Which configuration file do I need?</h2>
        <p className="mt-4">Start with <code>relay init</code>, which writes your CLI defaults. Add the other files only when your app needs them.</p>
        <div className="docs-table-wrap" role="region" aria-label="Configuration files" tabIndex={0}>
          <table className="docs-table"><thead><tr><th scope="col">File in your app folder</th><th scope="col">Use it to…</th><th scope="col">Learn more</th></tr></thead>
            <tbody>
              <tr><th scope="row"><code>.relay.json</code></th><td>Choose the server connection, app, environment, and branch. Created by <code>relay init</code>.</td><td><Link className="docs-text-link" href="/docs/cli#config-resolution">CLI defaults</Link></td></tr>
              <tr><th scope="row"><code>relay.config.json</code></th><td>Override build commands, service port, or monorepo paths.</td><td><Link className="docs-text-link" href="/docs/cli#config-resolution">Build configuration</Link></td></tr>
              <tr><th scope="row"><code>relay.json</code></th><td>Declare databases and companion services.</td><td><Link className="docs-text-link" href="/docs/services#relay-json-shape">Service configuration</Link></td></tr>
              <tr><th scope="row"><code>.relayignore</code></th><td>Exclude generated or irrelevant files from uploads, in addition to built-in exclusions.</td><td><Link className="docs-text-link" href="/docs/cli#deploy-path">File synchronization</Link></td></tr>
            </tbody>
          </table>
        </div>
        <p className="mt-4">Keep secret values out of committed files. Use <code>relay secrets add</code> for app secrets, and keep connection tokens private.</p>
      </section>
      <section id="troubleshooting">
        <h2>Something not working?</h2>
        <dl className="docs-concepts">
          <div><dt>The CLI cannot connect</dt><dd>Run <code>relay doctor</code> in the project folder. Check the server URL, credentials, and whether <code>relayd</code> is running. See <Link className="docs-text-link" href="/docs/cli#transport-modes">connection options</Link>.</dd></div>
          <div><dt>The build or deployment fails</dt><dd>Use <code>relay status</code> and <code>relay logs &lt;deploy-id&gt;</code> to inspect the failure. Check Docker on the server and your app&apos;s install, build, and start commands.</dd></div>
          <div><dt>A deployed app asks for login</dt><dd>Dev and staging lanes default to Relay Login. Check the lane&apos;s access policy in the dashboard. See <Link className="docs-text-link" href="/docs/previews#lane-lifecycle">lane access and lifecycle</Link>.</dd></div>
          <div><dt>A URL or hostname does not work</dt><dd>Check port mode versus managed host routing, then run <code>relay doctor</code> for DNS and TLS diagnostics. See <Link className="docs-text-link" href="/docs/previews">routing and app access</Link>.</dd></div>
        </dl>
      </section>
    </DocsPageLayout>
  );
}
