import type { Metadata } from "next";
import Link from "next/link";
import { CodePanel, DocsHero, DocsPageLayout } from "@/components/site-shell";
import allTools from "@/lib/mcp-tools.json";
import { McpSetup } from "@/components/mcp-setup";

export const metadata: Metadata = {
  title: "Connect an AI tool with MCP",
  description: "Configure Relay MCP for Cursor, Claude Desktop, or VS Code. Verify the connection and inspect the available tools.",
  alternates: { canonical: "/docs/mcp" },
};


export default function McpPage() {
  return (
    <DocsPageLayout currentPath="/docs/mcp" hero={<DocsHero title="Connect an AI tool with MCP" summary="Let your editor or assistant inspect and operate a Relay agent. Set up the connection first, verify it with a read-only tool, then use the reference below." />}>
      <section id="install">
        <h2>Before you connect</h2>
        <ul className="docs-requirements">
          <li>A running Relay agent. If you have not deployed an app yet, follow the <Link className="docs-text-link" href="/get-started">first deployment guide</Link>.</li>
          <li>Node.js 18 or later on the computer running your AI client.</li>
          <li>An agent URL and token for HTTP, or access to its local Unix socket.</li>
        </ul>
        <p>Your AI client starts <code>relay-mcp</code> as a local process and talks to it over stdio. That process calls the agent over HTTP or a Unix socket. It does not run inside the deployed app.</p>
        <p className="mt-4">The configuration below uses <code>npx</code>, so no global package installation is needed. Node.js must be installed before you launch the client.</p>
      </section>
      <section id="client-setup">
        <h2>1. Configure your client</h2>
        <p className="mt-4">Choose your client, operating system, and connection. Copy the resulting JSON into the indicated configuration file.</p>
        <McpSetup />
        <p className="mt-4">For additional VS Code options, see its <a className="docs-text-link" href="https://code.visualstudio.com/docs/agents/reference/mcp-configuration">MCP configuration reference</a>.</p>
      </section>
      <section id="verify">
        <h2>2. Verify the connection</h2>
        <ol className="mcp-step-list">
          <li>Restart or reconnect the Relay MCP server in your client.</li>
          <li>Check that Relay appears as connected and its tools are listed.</li>
          <li>Ask your assistant: <strong>“Use Relay to get the agent version and run the server diagnostics.”</strong></li>
        </ol>
        <p>This uses <code>get_version</code> and <code>get_doctor</code>. Neither changes the server. Then try <code>list_projects</code> to inspect the available apps and lanes.</p>
        <div className="docs-callout mt-6"><h3>Deploying local source files</h3><p>The MCP tools inspect and operate the agent; they do not upload your local app folder. Use <code>relay deploy --stream</code> in your app folder for that workflow. Your assistant can then inspect the deployment through MCP.</p></div>
      </section>
      <section id="transport">
        <h2>Connection settings</h2>
        <div className="docs-table-wrap" tabIndex={0} role="region" aria-label="MCP environment variables"><table className="docs-table"><thead><tr><th scope="col">Variable</th><th scope="col">Purpose</th></tr></thead><tbody>
          <tr><th scope="row"><code>RELAY_URL</code></th><td>Agent HTTP or HTTPS base URL. Defaults to <code>http://127.0.0.1:8080</code>. A reverse-proxy path prefix is supported.</td></tr>
          <tr><th scope="row"><code>RELAY_TOKEN</code></th><td>Token for that agent. HTTP requests send it in <code>X-Relay-Token</code>.</td></tr>
          <tr><th scope="row"><code>RELAY_SOCKET</code></th><td>Absolute local Unix socket path. Takes precedence over HTTP and uses filesystem access instead of a token.</td></tr>
          <tr><th scope="row"><code>RELAY_REQUEST_TIMEOUT_MS</code></th><td>Total request deadline in milliseconds. Defaults to 30000; accepts integers from 1 to 600000.</td></tr>
        </tbody></table></div>
        <p className="mt-4">Set only the connection you intend to use. A stale <code>RELAY_SOCKET</code> value overrides a valid HTTP URL. Redirects are reported as errors; configure the final agent URL.</p>
      </section>
      <section id="tool-categories">
        <h2>What you can do</h2>
        <dl className="docs-concepts">
          <div><dt>Inspect deployments</dt><dd>Find projects, read deployment records and stored logs, inspect runtime targets, compare traffic analytics, and check agent diagnostics.</dd></div>
          <div><dt>Operate a lane</dt><dd>Start, stop, or restart apps; scale replicas; manage variants; update resource limits and secrets; request and approve promotions.</dd></div>
          <div><dt>Manage the server</dt><dd>Inspect users, audit records, plugin catalogs, companions, server settings, and the owner operations report. The agent enforces your account permissions.</dd></div>
        </dl>
        <p className="mt-4">Tools identify whether they inspect or change the server so the client can present the appropriate action. For every lane operation, specify the app, environment, and branch. A missing branch defaults to <code>main</code>.</p>
      </section>
      <section id="variants-and-scaling">
        <h2>Variants, scaling, and analytics</h2>
        <p>Use <code>list_variants</code> to compare control, variants, traffic weights, metrics, and feedback. Change an existing experiment with <code>set_variant_weight</code>, <code>promote_variant</code>, <code>remove_variant</code>, or <code>reshuffle_variants</code>. Create or rebuild its image with <code>relay deploy --variant</code> in your app folder.</p>
        <CodePanel label="Example arguments for scale_app" code={'{ "app": "my-app", "env": "prod", "branch": "main", "replicas": 3, "apply": true }'} />
        <p>Scaling can queue a deployment and returns its ID. Inspect that deployment before assuming all replicas are running. See <Link className="docs-text-link" href="/docs/scaling">replicas and resource limits</Link> and <Link className="docs-text-link" href="/docs/variants">variants and feedback</Link>.</p>
        <p><code>get_analytics</code> and <code>abort_rollout</code> require HTTP/HTTPS; those agent routes are not exposed on the Unix socket. <code>get_runtime_targets</code> discovers runtime-log targets; follow their live output in the dashboard.</p>
      </section>
      <section id="tool-reference">
        <h2>Tool reference</h2>
        <p className="mt-4">{allTools.length} tools. “Inspect” tools read data; “Change” tools can modify running workloads, settings, or access.</p>
        <div className="docs-table-wrap mcp-tools" role="region" aria-label="MCP tool reference" tabIndex={0}><table className="docs-table"><thead><tr><th scope="col">Tool</th><th scope="col">Action</th><th scope="col">Purpose</th></tr></thead><tbody>
          {allTools.map(({name, description, readOnly}) => <tr key={name}><th scope="row"><code>{name}</code></th><td className="mcp-tool-kind">{readOnly ? "Inspect" : "Change"}</td><td>{description}</td></tr>)}
        </tbody></table></div>
      </section>
      <section id="troubleshooting">
        <h2>Connection problems</h2>
        <dl className="docs-concepts">
          <div><dt>The server does not start</dt><dd>Check the client’s MCP logs. Confirm Node.js is available to the desktop app, then choose the Windows launcher in the setup above if needed. Running a stdio server in a terminal can look idle because it waits for a client; this alone is not an error.</dd></div>
          <div><dt>HTTP 401 or 403</dt><dd>Check that the token belongs to the configured agent. A 403 means the account lacks permission for that action; owner-only tools require an owner account.</dd></div>
          <div><dt>Socket connection fails</dt><dd>Use the agent’s absolute socket path and check filesystem permissions. Use HTTP when the agent is on another machine or no socket is available.</dd></div>
          <div><dt>A request times out</dt><dd>Run <code>relay doctor</code> from a terminal connected to the same agent. Check the URL, proxy, and agent health before increasing <code>RELAY_REQUEST_TIMEOUT_MS</code>. Failed requests are returned as MCP tool errors without stopping the MCP server.</dd></div>
          <div><dt>A tool is missing</dt><dd>Reconnect the server and refresh the tool list. To request a newer published package, use <code>@relay-org/relay-mcp@latest</code> in the launch arguments. This reference follows the current source; for tools that have not been published yet, run the local checkout as described below.</dd></div>
        </dl>
        <details className="docs-overview"><summary>Developing against a local checkout</summary><div><p>Install dependencies in <code>relay-mcp/</code>, then configure the client to run Node.js with the absolute path to that checkout’s <code>index.js</code>. This uses local changes before they are published.</p><CodePanel label="Example local development launch" code={'"command": "node",\n"args": ["/absolute/path/to/relay/relay-mcp/index.js"]'} /></div></details>
      </section>
    </DocsPageLayout>
  );
}
