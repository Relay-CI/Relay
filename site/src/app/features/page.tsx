import type { Metadata } from "next";
import Link from "next/link";
import { DocsHero, DocsPageLayout } from "@/components/site-shell";
export const metadata: Metadata = { title: "Features", description: "Builds, deployments, databases, routing, and access controls in Relay.", alternates: { canonical: "/features" } };
const groups = [
  { id: "core-set", title: "Build and deploy", items: [
    ["Changed-file uploads", "Relay uploads files that have changed instead of sending the entire project on every deployment."],
    ["Framework detection", "Buildpacks detect your framework and choose build and start commands. You can override those commands or use a Dockerfile."],
    ["Databases and services", "Run PostgreSQL, Redis, MySQL, or MongoDB alongside your app. RelayDB adds PostgreSQL pooling, managed credentials, and readiness checks."],
  ], href: "/docs/cli", link: "CLI commands" },
  { id: "security", title: "Control access", items: [
    ["Lane access policies", "Require Relay Login, make an app public, share a signed link, or limit access to an IP allowlist. Dev and staging require Relay Login by default."],
    ["User accounts", "Assign roles and access to the people using your server. The agent checks permissions for each operation."],
    ["App secrets", "Manage app environment variables separately from source files. Configure RELAY_SECRET_KEY to encrypt stored secrets."],
  ], href: "/docs/agent#auth-surface", link: "Authentication and permissions" },
  { id: "build-tracking", title: "Inspect deployments", items: [
    ["Build history", "Each deployment has a build number, status, duration, and log. GitHub deployments also include commit details."],
    ["Runtime logs", "Read logs from a running app or companion service."],
    ["Audit log", "Review deployment actions, secret changes, and account changes with the actor and time recorded."],
  ], href: "/docs/cli#command-groups", link: "Deployment and log commands" },
  { id: "traffic-shape", title: "Manage environments and releases", items: [
    ["Separate lanes", "Deploy dev, staging, preview, and production independently. A lane is identified by project, environment, and branch."],
    ["Routing", "Expose an app through a host port or a managed hostname. Access rules belong to the lane."],
    ["Promotion and rollback", "Promote a staging image to production, require approval where needed, and restore a previous image if a release fails."],
    ["Temporary environments", "Set retention rules for dev and preview lanes so old environments can expire."],
  ], href: "/docs/previews", link: "Routing and lane configuration" },
  { id: "variants-scaling", title: "Variants, replicas, and measurements", items: [
    ["Variants", "Deploy another build alongside control, assign a share of new visitors, compare results, and promote or remove it."],
    ["Visitor feedback", "Collect votes, ratings, and comments through /__relay/feedback. Relay records the visitor’s assigned variant."],
    ["Replica scaling", "Run 1–16 Docker containers per release on the same server and balance traffic across them."],
    ["Readiness checks", "Require each replacement replica to return HTTP 2xx from a configured readiness path before switching traffic."],
    ["Resources and analytics", "Set per-container CPU and memory limits; inspect requests, latency, errors, and container usage."],
  ], href: "/docs/variants", link: "Variants and feedback" },
  { id: "runtime-posture", title: "Run on your infrastructure", items: [
    ["Docker runtime", "The agent builds images and manages containers, networks, and volumes on your server."],
    ["GitHub deployments", "Connect a repository to deploy branch previews and production releases from GitHub events."],
    ["MCP integration", "Connect an editor or assistant to inspect deployments, read logs, and operate existing lanes."],
  ], href: "/docs/architecture", link: "How Relay works" },
];
export default function FeaturesPage() {
  return <DocsPageLayout currentPath="/features" hero={<DocsHero title="What Relay does" summary="Build and deploy apps, run databases, and manage environments on your own server." />}>{groups.map((group) => <section id={group.id} key={group.id}><h2>{group.title}</h2><dl className="docs-concepts">{group.items.map(([title, body]) => <div key={title}><dt>{title}</dt><dd>{body}</dd></div>)}</dl><Link className="graphite-text-link" href={group.href}>{group.link}</Link>{group.id === "variants-scaling" && <p><Link className="graphite-text-link" href="/docs/scaling">Replicas and resource limits</Link> · <Link className="graphite-text-link" href="/docs/operations">Logs and analytics</Link></p>}</section>)}</DocsPageLayout>;
}
