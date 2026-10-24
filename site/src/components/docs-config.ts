export type DocsSection = {
  id: string;
  label: string;
  keywords?: string[];
};

export type DocsPage = {
  href: string;
  label: string;
  summary: string;
  category: string;
  keywords?: string[];
  sections: DocsSection[];
};

export const docsPages: DocsPage[] = [
  {
    href: "/docs",
    label: "Documentation",
    summary: "Find a guide, learn the basics, and choose the right config file.",
    category: "Start here",
    keywords: ["overview", "guides", "docs", "manual", "index", "lanes", "edge access", "doctor", "dns", "tls"],
    sections: [
      { id: "starting-points", label: "Choose a guide", keywords: ["quickstart", "reference"] },
      { id: "concepts", label: "Core concepts", keywords: ["lane", "agent", "buildpack", "companion"] },
      { id: "config-model", label: "Configuration files", keywords: [".relay.json", "relay.config.json", "relay.json", ".relayignore"] },
      { id: "troubleshooting", label: "Troubleshooting", keywords: ["doctor", "login", "docker", "port"] },
    ],
  },
  {
    href: "/get-started",
    label: "Get started",
    summary: "Install Relay, connect to your server, and deploy an app.",
    category: "Start here",
    keywords: ["quickstart", "deploy", "lanes", "setup"],
    sections: [
      { id: "requirements", label: "Before you begin" },
      { id: "path", label: "Install and deploy" },
      { id: "verify", label: "Check your deployment", keywords: ["doctor", "status", "login"] },
      { id: "after-first-deploy", label: "Next steps" },
    ],
  },
  {
    href: "/features",
    label: "Features",
    summary: "Builds, databases, environments, access controls, and deployment history.",
    category: "Start here",
    keywords: ["features", "lanes", "access policy", "signed link", "promotion", "retention", "runtimes", "project services", "blue green", "canary"],
    sections: [
      { id: "core-set", label: "Build and deploy", keywords: ["lanes", "sync", "rollout", "services"] },
      { id: "security", label: "Security and access" },
      { id: "build-tracking", label: "Build tracking" },
      { id: "traffic-shape", label: "Routing and releases", keywords: ["blue green", "session", "edge", "access policy", "promotion", "rollback"] },
      { id: "variants-scaling", label: "Variants and scaling", keywords: ["replicas", "feedback", "analytics", "readiness"] },
      { id: "runtime-posture", label: "Server runtime", keywords: ["runtime", "docker", "station", "security"] },
    ],
  },
  {
    href: "/docs/architecture",
    label: "Architecture",
    summary: "How the CLI, agent, storage, and container runtime work together.",
    category: "Reference",
    keywords: ["architecture", "storage", "sqlite", "docker", "station", "lane policy"],
    sections: [
      { id: "split", label: "CLI and agent split", keywords: ["relay", "relayd"] },
      { id: "storage", label: "Storage model", keywords: ["logs", "workspaces", "plugins"] },
      { id: "runtime-engine", label: "Runtime engine", keywords: ["station", "docker", "containerruntime", "engine"] },
      { id: "boundary", label: "Boundary", keywords: ["scope", "build cloud"] },
    ],
  },
  {
    href: "/docs/cli",
    label: "CLI commands",
    summary: "Deploy, inspect logs, manage secrets, and check connectivity.",
    category: "Reference",
    keywords: ["cli", "commands", "doctor", "config", "lanes", "access policy", "signed link", "promotion", "plugin install", "plugin search", "install-url", "monorepo", ".relayignore"],
    sections: [
      { id: "command-groups", label: "Command groups", keywords: ["init", "deploy", "restart", "plugin", "doctor"] },
      { id: "transport-modes", label: "Transport modes", keywords: ["socket", "http", "token"] },
      { id: "deploy-path", label: "Deploy path", keywords: ["sync", "host_port", "service_port", "mode", "lane", ".relayignore", "cache", "workspace fingerprint"] },
      { id: "config-resolution", label: "Config resolution", keywords: [".relay.json", "env vars", "flags", ".relayignore", "project_root", "build_context", "dockerfile"] },
      { id: "recent-additions", label: "Other CLI features", keywords: ["status", "projects", "secrets", "wizard", "relay-login", "station", "promotion", "doctor", "plugin search"] },
    ],
  },
  {
    href: "/docs/agent",
    label: "Server configuration",
    summary: "Configure relayd, authentication, storage, and control APIs.",
    category: "Reference",
    keywords: ["agent", "cors", "token", "lanes", "edge auth", "signed link", "promotion", "retention", "plugins", "data", "doctor", "dns", "tls", "manual deploy", "operations"],
    sections: [
      { id: "environment-model", label: "Server settings", keywords: ["relay_addr", "relay_token", "cors"] },
      { id: "auth-surface", label: "Authentication", keywords: ["authorization", "x-relay-token", "cookie", "socket", "relay-login", "signed link"] },
      { id: "operational-shape", label: "Data and storage", keywords: ["data dir", "relay.db", "plugins"] },
      { id: "control-surface", label: "HTTP API", keywords: ["apps/config", "apps/signed-link", "edge/authz", "promotions", "companions", "events", "webhooks", "doctor", "admin/ops", "deploys"] },
      { id: "runtime-engine", label: "Runtime engine", keywords: ["station", "docker", "containerruntime", "engine"] },
    ],
  },
  {
    href: "/docs/plugins",
    label: "Buildpack plugins",
    summary: "How server-side framework plugins detect apps and generate Docker plans.",
    category: "Integrations",
    keywords: ["plugin", "buildpack", "astro", "dockerfile template", "install", "search", "install-url", "sha256", "https"],
    sections: [
      { id: "plugin-model", label: "Plugin model", keywords: ["json", "server-side", "detect"] },
      { id: "installing-plugins", label: "Installing plugins", keywords: ["relay plugin install", "list", "remove"] },
      { id: "authoring-plugins", label: "Authoring plugins", keywords: ["detect_rules", "plan_spec", "template"] },
      { id: "built-in-buildpacks", label: "Built-in buildpacks" },
      { id: "what-comes-next", label: "Plugin roadmap" },
    ],
  },
  {
    href: "/docs/github",
    label: "GitHub and webhooks",
    summary: "Connect a GitHub App for delivery or configure a legacy webhook.",
    category: "Integrations",
    keywords: ["github", "webhook", "repo_url", "secret", "push deploy"],
    sections: [
      { id: "github-app", label: "Connect a GitHub App", keywords: ["installation", "repository", "checks"] },
      { id: "connected-delivery", label: "Preview and production delivery", keywords: ["pull request", "rollback", "health"] },
      { id: "webhook-flow", label: "Manual webhook", keywords: ["push", "deploy", "api/webhooks/github"] },
      { id: "connecting-a-repo", label: "Manual repository mapping", keywords: ["repo_url", "app config", "settings"] },
      { id: "secrets-and-safety", label: "Secrets and safety", keywords: ["signature", "webhook secret", "global secret"] },
    ],
  },
  {
    href: "/docs/services",
    label: "Databases and services",
    summary: "Use relay.json to boot Postgres, Redis, MySQL, or Mongo next to your app.",
    category: "Integrations",
    keywords: ["relay.json", "postgres", "redis", "mysql", "mongo", "DB_URL"],
    sections: [
      { id: "managed-option", label: "Choose RelayDB or a service" },
      { id: "relay-json-shape", label: "relay.json shape", keywords: ["services", "project", "containers"] },
      { id: "service-networking", label: "Service networking", keywords: ["docker network", "hostname", "urls"] },
      { id: "operating-services", label: "Operating services", keywords: ["volumes", "dashboard", "api/projects"] },
    ],
  },
  {
    href: "/docs/relaydb",
    label: "RelayDB",
    summary: "Managed PostgreSQL with PgBouncer pooling, encrypted per-lane credentials, and resource-aware tuning.",
    category: "Integrations",
    keywords: [
      "relaydb", "postgres", "postgresql", "pgbouncer", "connection pool", "DATABASE_URL",
      "profile", "starter", "balanced", "throughput", "credentials", "encryption", "aes", "companion",
    ],
    sections: [
      { id: "relaydb-vs-postgres", label: "RelayDB vs postgres", keywords: ["pooler", "tuning", "credentials"] },
      { id: "profiles", label: "Profiles", keywords: ["starter", "balanced", "throughput", "auto", "memory", "connections"] },
      { id: "connection-contract", label: "Connection contract", keywords: ["DATABASE_URL", "env var", "pooled url", "application_name"] },
      { id: "security", label: "Security", keywords: ["aes-256-gcm", "encryption", "relaydb.key", "credentials", "backup"] },
      { id: "config-reference", label: "Config reference", keywords: ["relay.json", "version", "profile", "host_port", "pooler_image"] },
    ],
  },
  {
    href: "/docs/previews",
    label: "Routing and app access",
    summary: "Port mode, managed lane hostnames, edge access policy, signed sharing, lifecycle, and how deploy links are determined.",
    category: "Operations",
    keywords: ["lanes", "host-port", "RELAY_BASE_DOMAIN", "public_host", "signed link", "retention", "promotion", "blue green", "session", "access policy"],
    sections: [
      { id: "port-mode", label: "Port mode", keywords: ["host-port", "localhost", "preview URL"] },
      { id: "host-mode", label: "Managed hosts", keywords: ["RELAY_BASE_DOMAIN", "dev", "staging", "preview"] },
      { id: "lane-lifecycle", label: "Lane lifecycle", keywords: ["expiry", "signed link", "promotion", "approval"] },
      { id: "slot-routing", label: "Slot routing", keywords: ["blue", "green", "traffic_mode", "session", "access_policy"] },
      { id: "host-ownership", label: "App and dashboard hostnames" },
      { id: "verification", label: "Verify DNS and routing", keywords: ["curl", "tls", "dns"] },
      { id: "what-gets-recorded", label: "What gets recorded", keywords: ["deploy record", "dashboard", "route", "visit link"] },
    ],
  },
  {
    href: "/docs/variants", label: "Variants and feedback",
    summary: "Deploy alternate builds, assign visitors, compare results, and promote or remove a variant.",
    category: "Operations", keywords: ["variant", "variants", "experiment", "A/B", "weight", "sticky", "feedback", "control", "reshuffle"],
    sections: [
      { id: "requirements", label: "Requirements" },
      { id: "deploy", label: "Deploy a variant", keywords: ["--variant", "--weight"] },
      { id: "traffic", label: "Traffic and assignments", keywords: ["cookie", "weight", "reshuffle", "__relay_variant"] },
      { id: "metrics", label: "Compare metrics" },
      { id: "feedback", label: "Visitor feedback", keywords: ["/__relay/feedback", "rating", "vote"] },
      { id: "finish", label: "Promote or remove" },
      { id: "api", label: "Variants API" },
    ],
  },
  {
    href: "/docs/scaling", label: "Replicas and resource limits",
    summary: "Run 1–16 Docker replicas, configure readiness checks, and set CPU and memory limits.",
    category: "Operations", keywords: ["replica", "replicas", "scale", "cpu", "memory", "resource_mode", "readiness_path", "health"],
    sections: [
      { id: "replicas", label: "Replica count", keywords: ["--replicas", "scale_app"] },
      { id: "readiness", label: "Readiness checks" },
      { id: "limits", label: "CPU and memory limits", keywords: ["manual", "auto", "cpu_limit", "mem_limit"] },
      { id: "verify", label: "Check a resize" },
      { id: "api", label: "Scaling API" },
    ],
  },
  {
    href: "/docs/operations", label: "Logs and analytics",
    summary: "Read build and runtime logs, compare traffic, inspect server usage, and abort a monitored rollout.",
    category: "Operations", keywords: ["logs", "runtime", "analytics", "operations", "telemetry", "latency", "bandwidth", "rollout", "abort"],
    sections: [
      { id: "build-logs", label: "Build logs" },
      { id: "runtime-logs", label: "Runtime targets and logs" },
      { id: "analytics", label: "Traffic analytics", keywords: ["get_analytics", "visitors", "p95"] },
      { id: "server-operations", label: "Server operations", keywords: ["get_admin_ops", "get_status", "doctor"] },
      { id: "rollouts", label: "Abort a rollout" },
    ],
  },
  {
    href: "/docs/mcp",
    label: "MCP server",
    summary: "Configure your AI client, verify the connection, and inspect or operate your agent.",
    category: "Integrations",
    keywords: ["mcp", "cursor", "claude", "copilot", "ai", "model context protocol", "relay-mcp", "npx"],
    sections: [
      { id: "install", label: "Before you connect", keywords: ["npx", "npm", "node", "package"] },
      { id: "client-setup", label: "Configure your client", keywords: ["cursor", "claude desktop", "vs code", "windows", "mcp.json", "config"] },
      { id: "verify", label: "Verify the connection", keywords: ["doctor", "get_doctor", "get_version"] },
      { id: "transport", label: "Connection settings", keywords: ["socket", "http", "token", "RELAY_SOCKET", "RELAY_URL", "timeout"] },
      { id: "tool-categories", label: "What you can do", keywords: ["deploys", "secrets", "promotions", "admin", "control"] },
      { id: "variants-and-scaling", label: "Variants and scaling", keywords: ["scale_app", "list_variants", "get_analytics"] },
      { id: "tool-reference", label: "Tool reference", keywords: ["tools", "list", "reference", "api"] },
      { id: "troubleshooting", label: "MCP troubleshooting", keywords: ["npm", "install", "publish"] },
    ],
  },
];

export function getDocsPage(pathname: string) {
  return docsPages.find((page) => page.href === pathname) ?? docsPages[0];
}
