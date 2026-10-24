import type { Metadata } from "next";
import Link from "next/link";
import { CodePanel, DocsHero, DocsPageLayout } from "@/components/site-shell";

export const metadata: Metadata = {
  title: "Replicas and resource limits",
  description: "Scale a Docker lane to 1–16 replicas, configure readiness checks, and set CPU and memory limits.",
  alternates: { canonical: "/docs/scaling" },
};

export default function ScalingPage() {
  return (
    <DocsPageLayout currentPath="/docs/scaling" hero={<DocsHero title="Replicas and resource limits" summary="Run more copies of an app on your server, check that each one is ready, and control how much CPU and memory each container can use." />}>
      <section id="replicas">
        <h2>Run more copies of a lane</h2>
        <p>A Docker lane can run 1–16 replicas per release. The edge proxy balances requests across those containers on the same host. Replicas add capacity on that server; they do not provision additional machines.</p>
        <CodePanel label="Inside your app folder" code={`relay scale 3 --app my-app --env prod --branch main

# Or choose the count with a deployment
relay deploy --app my-app --env prod --branch main --replicas 3`} />
        <p>For a running lane that needs resizing, scale queues a deployment of its existing image. Relay starts the replacement replicas, waits for readiness, and switches traffic. Keep enough capacity for both releases during the change. For a stopped lane, the saved count applies when it next runs.</p>
        <p>In the dashboard, select the project and lane, then open <strong>Resources</strong>. Set the replica count there. The MCP tool <code>scale_app</code> accepts <code>replicas</code> and an optional <code>apply</code>; set <code>apply: false</code> to save the count without queuing a rollout.</p>
        <p>Keep application state outside individual containers when requests may reach different replicas. Replicas use the same lane services, environment, secrets, and configured volume mounts. More app replicas do not create extra database replicas.</p>
      </section>
      <section id="readiness">
        <h2>Check readiness before switching traffic</h2>
        <p>Set <code>replicas</code> and an optional <code>readiness_path</code> in <code>relay.config.json</code>. This example assumes your app has a health endpoint at <code>/health</code>.</p>
        <CodePanel label="relay.config.json — merge with your existing build settings" code={`{
  "service_port": 3000,
  "replicas": 3,
  "readiness_path": "/health"
}`} />
        <p>With a readiness path configured, every replacement replica must return an HTTP 2xx response before the slot takes traffic. Use an absolute path without a query string or fragment. A failed readiness check fails the deployment; inspect its build log before retrying.</p>
        <p>Application-level readiness paths require Docker. Without a custom path, Relay uses its normal runtime readiness checks. See <Link className="docs-text-link" href="/docs/previews#slot-routing">release routing</Link> for edge versus session handoff.</p>
      </section>
      <section id="limits">
        <h2>Set CPU and memory limits</h2>
        <p>Open <strong>Resources</strong> in the dashboard. In Manual mode, set a Docker CPU value such as <code>0.5</code> or <code>2</code>, and a memory value such as <code>512m</code> or <code>1g</code>. These limits apply per container when it next starts; saving the settings does not resize an already running container.</p>
        <CodePanel label="POST /api/apps/config — example manual limits" code={`{
  "app": "my-app",
  "env": "prod",
  "branch": "main",
  "resource_mode": "manual",
  "cpu_limit": "0.5",
  "mem_limit": "512m"
}`} />
        <p>Use <code>set_app_config</code> through MCP for the same fields. Three replicas at <code>512m</code> need up to three times that app-memory allowance, in addition to the agent, builds, proxy, databases, and any temporary replacement containers.</p>
        <p>Automatic mode currently ignores saved manual CPU and memory values. It does not adjust replica counts or dynamically tune limits. When no explicit memory cap is used, the agent can still apply its host-sized default cap; server settings such as <code>RELAY_APP_MEM_LIMIT_MB</code> control that default.</p>
      </section>
      <section id="verify">
        <h2>Check the result</h2>
        <ol className="mcp-step-list">
          <li>Follow the deployment ID returned by scale in the dashboard or with <code>relay logs &lt;id&gt;</code>.</li>
          <li>Check the configured replica count and running replica count in Resources or <code>GET /api/projects</code>.</li>
          <li>Use Runtime logs to inspect individual targets. Owners can also use Server admin → Operations to compare container usage.</li>
        </ol>
        <p>A failed resize may leave the saved desired count different from the currently running count. Read the deployment failure and confirm capacity before retrying. <Link className="docs-text-link" href="/docs/operations">Logs and analytics</Link> explain the available measurements.</p>
      </section>
      <section id="api">
        <h2>Scaling API</h2>
        <p><code>POST /api/apps/scale</code> takes <code>app</code>, <code>env</code>, <code>branch</code>, <code>replicas</code>, and optional <code>apply</code> (default true). It requires deployer access and a Docker lane. The response includes the saved <code>replicas</code> and, when a resize is queued, <code>deploy_id</code>.</p>
        <p>The scale route is available over HTTP and the local socket. A variant starts with the lane&apos;s configured replica count when it is deployed; changing that count does not itself rebuild existing variants. See <Link className="docs-text-link" href="/docs/variants">variants and visitor feedback</Link>.</p>
      </section>
    </DocsPageLayout>
  );
}
