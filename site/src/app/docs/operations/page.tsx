import type { Metadata } from "next";
import Link from "next/link";
import { CodePanel, DocsHero, DocsPageLayout } from "@/components/site-shell";

export const metadata: Metadata = {
  title: "Logs, analytics, and server operations",
  description: "Find build and runtime logs, read traffic analytics, and inspect server resource usage in Relay.",
  alternates: { canonical: "/docs/operations" },
};

export default function OperationsPage() {
  return (
    <DocsPageLayout currentPath="/docs/operations" hero={<DocsHero title="Logs, analytics, and server operations" summary="Find out why a deployment failed, inspect a running app, and check how traffic and containers are using your server." />}>
      <section id="build-logs">
        <h2>Read build and deployment logs</h2>
        <p>Select a project, open Deployments or Build logs, and inspect the deployment. Logs cover upload, build, readiness, and rollout work. You can also read them from your app folder:</p>
        <CodePanel label="CLI — replace the example deployment ID" code={`relay status
relay list --app my-app --env prod --branch main --limit 10
relay logs <deploy-id>
relay logs <deploy-id> --no-stream`} />
        <p>For MCP, use <code>list_deploys</code>, <code>get_deploy</code>, and <code>get_deploy_logs</code>. Build logs belong to a deployment; output from a container that is already running is in Runtime logs.</p>
      </section>
      <section id="runtime-logs">
        <h2>Read logs from a running container</h2>
        <p>Select the lane and open <strong>Runtime logs</strong>. Choose the app, replica, variant, proxy, or companion target when available. Filter by time window, level, or text. If the lane is stopped or the selected container is unavailable, start the lane or select an available target.</p>
        <p><code>get_runtime_targets</code> in MCP, or <code>GET /api/runtime/logs/targets?app=my-app&amp;env=prod&amp;branch=main</code>, returns targets, the default target, and offline state. Target IDs come from this response; do not guess container names.</p>
        <p><code>GET /api/runtime/logs/stream</code> takes the same lane fields plus <code>target</code>, optional <code>tail</code> (default 200, maximum 2,000), and optional <code>since</code>. It returns a live SSE stream rather than a finite JSON response. Follow it in the dashboard or an SSE-aware client; the MCP JSON tools do not buffer that live stream.</p>
      </section>
      <section id="analytics">
        <h2>Read traffic analytics</h2>
        <p>Open <strong>Analytics</strong> for the selected lane. Choose 24 hours, 7 days, or 30 days to inspect requests, visitors, page views, bandwidth, errors, and latency. Owner/admin accounts can access the all-apps report; other accounts must select an app they can view.</p>
        <CodePanel label="HTTP API query — use an authenticated request" code={`GET /api/analytics?app=my-app&env=prod&branch=main&period=24h

# Filter to a named variant or control
GET /api/analytics?app=my-app&env=prod&branch=main&period=7d&variant=checkout`} />
        <p>MCP exposes the same report as <code>get_analytics</code>. This route is registered on the HTTP/HTTPS listener only. If your MCP connection uses <code>RELAY_SOCKET</code>, unset it and configure <code>RELAY_URL</code> and credentials to use this tool.</p>
        <p>Traffic measurements come from the proxy. Direct container traffic is outside that view. Visitor counts use a daily rotating hash of client IP and user agent; they are not a persistent count of people across different days. Variant assignment uses a separate sticky cookie. Compare variant reports in the <Link className="docs-text-link" href="/docs/variants#metrics">Variants guide</Link>.</p>
      </section>
      <section id="server-operations">
        <h2>Check server usage</h2>
        <p>Owners can open <strong>Server admin → Operations</strong> to inspect Docker CPU, memory, storage, container counts, daemon memory, and differences between deployments. The page refreshes every ten seconds while visible and can be paused.</p>
        <p>Use <code>get_admin_ops</code> or <code>GET /api/admin/ops</code> for this snapshot. Use <code>get_status</code> or <code>GET /api/status</code> for the runtime health inventory, including edge/active-slot state and available exit or out-of-memory information.</p>
        <p>For connection, storage, Docker, DNS, TLS, and proxy setup problems, run <code>relay doctor</code> or MCP <code>get_doctor</code>. For capacity changes, see <Link className="docs-text-link" href="/docs/scaling">replicas and resource limits</Link>.</p>
      </section>
      <section id="rollouts">
        <h2>Stop a failing rollout</h2>
        <p>For an edge canary, lane settings include <code>traffic_split_percent</code> (1–100), <code>rollout_min_requests</code> (at least 1), <code>rollout_error_percent</code> (greater than 0 and at most 100), and <code>rollout_assess_seconds</code> (at least 30). Configure these before a deployment through Environment settings, the app-config API, or MCP <code>set_app_config</code>. Session traffic mode uses a full 100% handoff.</p>
        <p>For a canary rollout that is still monitoring and has a standby slot, the dashboard&apos;s abort action or MCP <code>abort_rollout</code> restores the previous slot. The API is <code>POST /api/deploys/rollout-abort</code> with the lane&apos;s <code>app</code>, <code>env</code>, and <code>branch</code>. It requires deployer access and HTTP/HTTPS.</p>
        <p>This action applies to an active monitored rollout. Use ordinary rollback for a completed release, or <code>cancel_deploy</code> to cancel an in-progress build. A long-running variant is a separate workflow; remove it or adjust its weight from Variants.</p>
      </section>
    </DocsPageLayout>
  );
}
