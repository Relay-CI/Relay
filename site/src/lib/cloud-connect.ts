import { cache } from "react";
import { isDatabaseConfigured, query } from "@/lib/postgres";

export type OverviewSource = "database" | "database-empty" | "demo";

export type WorkspaceSummary = {
  id: string;
  name: string;
  slug: string;
  plan: string;
  region: string;
  websocketEndpoint: string;
  teamMembers: number;
  roleCount: number;
  logRetentionDays: number;
  ssoEnabled: boolean;
};

export type FleetSummary = {
  serversTotal: number;
  serversOnline: number;
  appsTotal: number;
  deploysToday: number;
  activeAlerts: number;
  queuedActions: number;
};

export type ServerCard = {
  id: string;
  name: string;
  region: string;
  environment: string;
  connected: boolean;
  agentVersion: string;
  appCount: number;
  cpuPercent: number;
  memoryPercent: number;
  websocketLatencyMs: number;
  activeAlerts: number;
  minutesSinceSeen: number;
};

export type AppCard = {
  id: string;
  name: string;
  runtime: string;
  environment: string;
  status: string;
  serverName: string;
  liveInstances: number;
  desiredInstances: number;
  releaseChannel: string;
  latestVersion: string;
  deployStatus: string;
  releaseAgeMinutes: number;
};

export type DeploymentRow = {
  id: string;
  appName: string;
  serverName: string;
  version: string;
  status: string;
  triggerKind: string;
  triggeredBy: string;
  durationSeconds: number | null;
  finishedMinutesAgo: number;
};

export type RemoteActionRow = {
  id: string;
  actionKind: string;
  targetName: string;
  status: string;
  requestedBy: string;
  requestedMinutesAgo: number;
  note: string | null;
};

export type EntitlementRow = {
  featureKey: string;
  status: string;
  quota: number | null;
  notes: string | null;
};

export type CloudConnectOverview = {
  source: OverviewSource;
  generatedAt: string;
  workspace: WorkspaceSummary;
  summary: FleetSummary;
  servers: ServerCard[];
  apps: AppCard[];
  deployments: DeploymentRow[];
  actions: RemoteActionRow[];
  entitlements: EntitlementRow[];
};

export type LeadStatus =
  | "new"
  | "reviewing"
  | "qualified"
  | "pilot"
  | "won"
  | "rejected";

export type LeadPriority = "low" | "medium" | "high" | "urgent";

export type CloudLead = {
  id: string;
  email: string;
  teamName: string | null;
  fleetSize: number | null;
  needs: string | null;
  status: LeadStatus;
  priority: LeadPriority;
  ownerName: string | null;
  reviewNotes: string | null;
  customerToken: string;
  createdAt: string;
  updatedAt: string;
  lastContactedAt: string | null;
};

export type CloudLeadAdminSummary = {
  total: number;
  new: number;
  reviewing: number;
  qualified: number;
  highPriority: number;
};

export type CloudLeadAdminDashboard = {
  source: OverviewSource;
  generatedAt: string;
  summary: CloudLeadAdminSummary;
  leads: CloudLead[];
};

export type CloudCustomerPanel = {
  source: OverviewSource;
  lead: CloudLead;
  recommendedPlan: string;
  nextStepLabel: string;
  nextStepBody: string;
  workspaceName: string;
};

type WorkspaceRow = WorkspaceSummary & FleetSummary;
type LeadRow = Omit<CloudLead, "fleetSize"> & { fleetSize: number | null };
type LeadSummaryRow = CloudLeadAdminSummary;

const demoOverview: CloudConnectOverview = {
  source: "demo",
  generatedAt: "2026-05-01T12:00:00.000Z",
  workspace: {
    id: "workspace-demo",
    name: "Relay Cloud Connect",
    slug: "relay-cloud-connect",
    plan: "Scale",
    region: "us-east-1",
    websocketEndpoint: "wss://relay.com/connect",
    teamMembers: 7,
    roleCount: 4,
    logRetentionDays: 30,
    ssoEnabled: true,
  },
  summary: {
    serversTotal: 5,
    serversOnline: 4,
    appsTotal: 12,
    deploysToday: 18,
    activeAlerts: 2,
    queuedActions: 3,
  },
  servers: [
    {
      id: "srv-1",
      name: "nyc-edge-1",
      region: "New York",
      environment: "production",
      connected: true,
      agentVersion: "relayd 0.8.4",
      appCount: 4,
      cpuPercent: 38,
      memoryPercent: 64,
      websocketLatencyMs: 42,
      activeAlerts: 0,
      minutesSinceSeen: 1,
    },
    {
      id: "srv-2",
      name: "fra-build-2",
      region: "Frankfurt",
      environment: "staging",
      connected: true,
      agentVersion: "relayd 0.8.4",
      appCount: 3,
      cpuPercent: 57,
      memoryPercent: 71,
      websocketLatencyMs: 88,
      activeAlerts: 1,
      minutesSinceSeen: 2,
    },
    {
      id: "srv-3",
      name: "lon-preview-1",
      region: "London",
      environment: "preview",
      connected: true,
      agentVersion: "relayd 0.8.3",
      appCount: 2,
      cpuPercent: 21,
      memoryPercent: 44,
      websocketLatencyMs: 61,
      activeAlerts: 0,
      minutesSinceSeen: 1,
    },
    {
      id: "srv-4",
      name: "home-lab",
      region: "Dubai",
      environment: "dev",
      connected: false,
      agentVersion: "relayd 0.8.1",
      appCount: 3,
      cpuPercent: 0,
      memoryPercent: 0,
      websocketLatencyMs: 0,
      activeAlerts: 1,
      minutesSinceSeen: 37,
    },
  ],
  apps: [
    {
      id: "app-1",
      name: "dashboard-web",
      runtime: "Next.js",
      environment: "production",
      status: "healthy",
      serverName: "nyc-edge-1",
      liveInstances: 3,
      desiredInstances: 3,
      releaseChannel: "stable",
      latestVersion: "web-2026.05.01-4",
      deployStatus: "succeeded",
      releaseAgeMinutes: 19,
    },
    {
      id: "app-2",
      name: "billing-api",
      runtime: "Go",
      environment: "production",
      status: "healthy",
      serverName: "nyc-edge-1",
      liveInstances: 2,
      desiredInstances: 2,
      releaseChannel: "stable",
      latestVersion: "api-2026.05.01-2",
      deployStatus: "succeeded",
      releaseAgeMinutes: 44,
    },
    {
      id: "app-3",
      name: "ops-runner",
      runtime: "Node",
      environment: "staging",
      status: "degraded",
      serverName: "fra-build-2",
      liveInstances: 1,
      desiredInstances: 2,
      releaseChannel: "candidate",
      latestVersion: "ops-2026.05.01-8",
      deployStatus: "running",
      releaseAgeMinutes: 6,
    },
    {
      id: "app-4",
      name: "preview-gateway",
      runtime: "Rust",
      environment: "preview",
      status: "healthy",
      serverName: "lon-preview-1",
      liveInstances: 1,
      desiredInstances: 1,
      releaseChannel: "preview",
      latestVersion: "preview-9481e",
      deployStatus: "succeeded",
      releaseAgeMinutes: 3,
    },
  ],
  deployments: [
    {
      id: "dep-1",
      appName: "ops-runner",
      serverName: "fra-build-2",
      version: "ops-2026.05.01-8",
      status: "running",
      triggerKind: "manual",
      triggeredBy: "ali@relay.dev",
      durationSeconds: null,
      finishedMinutesAgo: 6,
    },
    {
      id: "dep-2",
      appName: "dashboard-web",
      serverName: "nyc-edge-1",
      version: "web-2026.05.01-4",
      status: "succeeded",
      triggerKind: "git",
      triggeredBy: "ci@github",
      durationSeconds: 143,
      finishedMinutesAgo: 19,
    },
    {
      id: "dep-3",
      appName: "billing-api",
      serverName: "nyc-edge-1",
      version: "api-2026.05.01-2",
      status: "succeeded",
      triggerKind: "manual",
      triggeredBy: "sara@relay.dev",
      durationSeconds: 98,
      finishedMinutesAgo: 44,
    },
  ],
  actions: [
    {
      id: "act-1",
      actionKind: "deploy",
      targetName: "ops-runner",
      status: "running",
      requestedBy: "ali@relay.dev",
      requestedMinutesAgo: 5,
      note: "Promote candidate build to staging",
    },
    {
      id: "act-2",
      actionKind: "restart",
      targetName: "billing-api",
      status: "queued",
      requestedBy: "nina@relay.dev",
      requestedMinutesAgo: 9,
      note: "Drain unhealthy worker after alert",
    },
    {
      id: "act-3",
      actionKind: "logs",
      targetName: "home-lab",
      status: "queued",
      requestedBy: "marc@relay.dev",
      requestedMinutesAgo: 14,
      note: "Fetch buffered logs after reconnect",
    },
  ],
  entitlements: [
    {
      featureKey: "teams",
      status: "enabled",
      quota: 10,
      notes: "Multi-user workspace access",
    },
    {
      featureKey: "rbac",
      status: "enabled",
      quota: 4,
      notes: "Admin, operator, deployer, observer",
    },
    {
      featureKey: "log_retention",
      status: "metered",
      quota: 30,
      notes: "Retention window in days",
    },
    {
      featureKey: "advanced_features",
      status: "enabled",
      quota: null,
      notes: "Remote actions, alerts, monitoring exports",
    },
    {
      featureKey: "sso",
      status: "enabled",
      quota: null,
      notes: "SAML and SCIM for company domains",
    },
  ],
};

const demoLeads: CloudLead[] = [
  {
    id: "lead-demo-1",
    email: "ops@northstar.dev",
    teamName: "Northstar Platform",
    fleetSize: 12,
    needs: "RBAC for operators, 30-day log retention, and Okta SSO before the summer cutover.",
    status: "qualified",
    priority: "high",
    ownerName: "Ali",
    reviewNotes: "Strong platform fit. Wants pilot on staging first.",
    customerToken: "northstar-demo",
    createdAt: "2026-05-01T11:18:00.000Z",
    updatedAt: "2026-05-01T12:42:00.000Z",
    lastContactedAt: "2026-05-01T12:05:00.000Z",
  },
  {
    id: "lead-demo-2",
    email: "infra@aperture.ai",
    teamName: "Aperture Infra",
    fleetSize: 28,
    needs: "Remote deploy approvals, customer-owned servers, and longer log retention for incident review.",
    status: "reviewing",
    priority: "urgent",
    ownerName: "Sara",
    reviewNotes: "Multi-region fleet and strong enterprise expansion path.",
    customerToken: "aperture-demo",
    createdAt: "2026-05-01T09:12:00.000Z",
    updatedAt: "2026-05-01T10:34:00.000Z",
    lastContactedAt: "2026-05-01T10:02:00.000Z",
  },
  {
    id: "lead-demo-3",
    email: "hello@loworbit.io",
    teamName: "Low Orbit",
    fleetSize: 4,
    needs: "A cleaner way to deploy to customer boxes without asking for port forwarding.",
    status: "new",
    priority: "medium",
    ownerName: null,
    reviewNotes: null,
    customerToken: "loworbit-demo",
    createdAt: "2026-05-01T08:54:00.000Z",
    updatedAt: "2026-05-01T08:54:00.000Z",
    lastContactedAt: null,
  },
];

function clampPercent(value: number | null | undefined) {
  const safe = Number(value ?? 0);
  return Math.max(0, Math.min(100, Math.round(safe)));
}

function normalizeOverview(
  source: OverviewSource,
  workspace: WorkspaceRow,
  servers: ServerCard[],
  apps: AppCard[],
  deployments: DeploymentRow[],
  actions: RemoteActionRow[],
  entitlements: EntitlementRow[],
): CloudConnectOverview {
  return {
    source,
    generatedAt: new Date().toISOString(),
    workspace: {
      id: workspace.id,
      name: workspace.name,
      slug: workspace.slug,
      plan: workspace.plan,
      region: workspace.region,
      websocketEndpoint: workspace.websocketEndpoint,
      teamMembers: workspace.teamMembers,
      roleCount: workspace.roleCount,
      logRetentionDays: workspace.logRetentionDays,
      ssoEnabled: workspace.ssoEnabled,
    },
    summary: {
      serversTotal: workspace.serversTotal,
      serversOnline: workspace.serversOnline,
      appsTotal: workspace.appsTotal,
      deploysToday: workspace.deploysToday,
      activeAlerts: workspace.activeAlerts,
      queuedActions: workspace.queuedActions,
    },
    servers: servers.map((server) => ({
      ...server,
      cpuPercent: clampPercent(server.cpuPercent),
      memoryPercent: clampPercent(server.memoryPercent),
    })),
    apps,
    deployments,
    actions,
    entitlements:
      entitlements.length > 0
        ? entitlements
        : demoOverview.entitlements.map((item) => ({ ...item })),
  };
}

function normalizeLead(row: LeadRow): CloudLead {
  return {
    ...row,
    fleetSize: row.fleetSize === null ? null : Number(row.fleetSize),
  };
}

function summarizeLeads(leads: CloudLead[]): CloudLeadAdminSummary {
  return {
    total: leads.length,
    new: leads.filter((lead) => lead.status === "new").length,
    reviewing: leads.filter((lead) => lead.status === "reviewing").length,
    qualified: leads.filter(
      (lead) => lead.status === "qualified" || lead.status === "pilot",
    ).length,
    highPriority: leads.filter(
      (lead) => lead.priority === "high" || lead.priority === "urgent",
    ).length,
  };
}

function estimatedPlan(fleetSize: number | null) {
  if (!fleetSize || fleetSize <= 5) {
    return "Starter";
  }

  if (fleetSize <= 20) {
    return "Growth";
  }

  return "Scale";
}

function buildNextStep(status: LeadStatus) {
  switch (status) {
    case "reviewing":
      return {
        label: "Architecture review in progress",
        body: "We are mapping your fleet size, required roles, and retention policy into a pilot workspace.",
      };
    case "qualified":
      return {
        label: "Pilot scope approved",
        body: "Next step is provisioning a workspace, inviting operators, and connecting the first relayd agents.",
      };
    case "pilot":
      return {
        label: "Pilot is active",
        body: "Your team is in the rollout window. We focus on first servers, deploy approval rules, and retention settings.",
      };
    case "won":
      return {
        label: "Workspace live",
        body: "Cloud Connect is already serving as the control plane for your fleet.",
      };
    case "rejected":
      return {
        label: "Request closed",
        body: "This request is not active right now, but the captured requirements remain available for follow-up.",
      };
    default:
      return {
        label: "Request received",
        body: "The team now has your fleet size and buying signal. The next step is triage and pilot qualification.",
      };
  }
}

function toCustomerPanel(source: OverviewSource, lead: CloudLead): CloudCustomerPanel {
  const nextStep = buildNextStep(lead.status);
  return {
    source,
    lead,
    recommendedPlan: estimatedPlan(lead.fleetSize),
    nextStepLabel: nextStep.label,
    nextStepBody: nextStep.body,
    workspaceName: lead.teamName
      ? `${lead.teamName} Workspace`
      : "Relay Cloud Connect Workspace",
  };
}

export const getCloudConnectOverview = cache(async (): Promise<CloudConnectOverview> => {
  if (!isDatabaseConfigured()) {
    return demoOverview;
  }

  try {
    const workspaceResult = await query<WorkspaceRow>(
      `
        SELECT
          w.id::text AS id,
          w.name,
          w.slug,
          w.plan,
          w.region,
          w.websocket_endpoint AS "websocketEndpoint",
          w.log_retention_days AS "logRetentionDays",
          COALESCE(
            (SELECT COUNT(*)::int FROM workspace_memberships wm WHERE wm.workspace_id = w.id),
            0
          ) AS "teamMembers",
          COALESCE(
            (SELECT COUNT(DISTINCT wm.role)::int FROM workspace_memberships wm WHERE wm.workspace_id = w.id),
            0
          ) AS "roleCount",
          EXISTS(
            SELECT 1
            FROM sso_connections sc
            WHERE sc.workspace_id = w.id AND sc.status = 'active'
          ) AS "ssoEnabled",
          COALESCE(
            (SELECT COUNT(*)::int FROM servers s WHERE s.workspace_id = w.id),
            0
          ) AS "serversTotal",
          COALESCE(
            (SELECT COUNT(*)::int FROM servers s WHERE s.workspace_id = w.id AND s.connected = true),
            0
          ) AS "serversOnline",
          COALESCE(
            (SELECT COUNT(*)::int FROM apps a WHERE a.workspace_id = w.id),
            0
          ) AS "appsTotal",
          COALESCE(
            (
              SELECT COUNT(*)::int
              FROM deployments d
              WHERE d.workspace_id = w.id
                AND d.started_at >= NOW() - INTERVAL '24 hours'
            ),
            0
          ) AS "deploysToday",
          COALESCE(
            (SELECT SUM(s.active_alerts)::int FROM servers s WHERE s.workspace_id = w.id),
            0
          ) AS "activeAlerts",
          COALESCE(
            (
              SELECT COUNT(*)::int
              FROM remote_actions ra
              WHERE ra.workspace_id = w.id
                AND ra.status IN ('queued', 'running')
            ),
            0
          ) AS "queuedActions"
        FROM workspaces w
        ORDER BY w.created_at ASC
        LIMIT 1
      `,
    );

    const workspace = workspaceResult.rows[0];
    if (!workspace) {
      return { ...demoOverview, source: "database-empty" };
    }

    const workspaceId = workspace.id;
    const [
      serversResult,
      appsResult,
      deploymentsResult,
      actionsResult,
      entitlementsResult,
    ] = await Promise.all([
      query<ServerCard>(
        `
          SELECT
            s.id::text AS id,
            s.name,
            s.region,
            s.environment,
            s.connected,
            s.agent_version AS "agentVersion",
            COALESCE(apps.app_count, 0) AS "appCount",
            COALESCE(metrics.cpu_percent, s.cpu_percent, 0) AS "cpuPercent",
            COALESCE(metrics.memory_percent, s.memory_percent, 0) AS "memoryPercent",
            COALESCE(metrics.websocket_latency_ms, 0) AS "websocketLatencyMs",
            COALESCE(s.active_alerts, 0) AS "activeAlerts",
            GREATEST(
              FLOOR(EXTRACT(EPOCH FROM (NOW() - COALESCE(s.last_seen_at, metrics.captured_at, s.created_at))) / 60),
              0
            )::int AS "minutesSinceSeen"
          FROM servers s
          LEFT JOIN LATERAL (
            SELECT COUNT(*)::int AS app_count
            FROM apps a
            WHERE a.server_id = s.id
          ) apps ON true
          LEFT JOIN LATERAL (
            SELECT
              ms.captured_at,
              ms.cpu_percent,
              ms.memory_percent,
              ms.websocket_latency_ms
            FROM metric_snapshots ms
            WHERE ms.server_id = s.id
            ORDER BY ms.captured_at DESC
            LIMIT 1
          ) metrics ON true
          WHERE s.workspace_id = $1::uuid
          ORDER BY s.connected DESC, s.active_alerts DESC, s.name ASC
          LIMIT 6
        `,
        [workspaceId],
      ),
      query<AppCard>(
        `
          SELECT
            a.id::text AS id,
            a.name,
            a.runtime,
            a.environment,
            a.status,
            s.name AS "serverName",
            a.live_instances AS "liveInstances",
            a.desired_instances AS "desiredInstances",
            a.release_channel AS "releaseChannel",
            COALESCE(dep.version, 'pending') AS "latestVersion",
            COALESCE(dep.status, 'pending') AS "deployStatus",
            GREATEST(
              FLOOR(EXTRACT(EPOCH FROM (NOW() - COALESCE(dep.finished_at, dep.started_at, a.created_at))) / 60),
              0
            )::int AS "releaseAgeMinutes"
          FROM apps a
          LEFT JOIN servers s ON s.id = a.server_id
          LEFT JOIN LATERAL (
            SELECT d.version, d.status, d.finished_at, d.started_at
            FROM deployments d
            WHERE d.app_id = a.id
            ORDER BY d.started_at DESC
            LIMIT 1
          ) dep ON true
          WHERE a.workspace_id = $1::uuid
          ORDER BY
            CASE a.status
              WHEN 'degraded' THEN 0
              WHEN 'deploying' THEN 1
              ELSE 2
            END,
            a.name ASC
          LIMIT 8
        `,
        [workspaceId],
      ),
      query<DeploymentRow>(
        `
          SELECT
            d.id::text AS id,
            a.name AS "appName",
            s.name AS "serverName",
            d.version,
            d.status,
            d.trigger_kind AS "triggerKind",
            d.triggered_by AS "triggeredBy",
            d.duration_seconds AS "durationSeconds",
            GREATEST(
              FLOOR(
                EXTRACT(
                  EPOCH FROM (
                    NOW() - COALESCE(d.finished_at, d.started_at, d.created_at)
                  )
                ) / 60
              ),
              0
            )::int AS "finishedMinutesAgo"
          FROM deployments d
          LEFT JOIN apps a ON a.id = d.app_id
          LEFT JOIN servers s ON s.id = d.server_id
          WHERE d.workspace_id = $1::uuid
          ORDER BY d.created_at DESC
          LIMIT 8
        `,
        [workspaceId],
      ),
      query<RemoteActionRow>(
        `
          SELECT
            ra.id::text AS id,
            ra.action_kind AS "actionKind",
            COALESCE(a.name, s.name, 'workspace') AS "targetName",
            ra.status,
            ra.requested_by AS "requestedBy",
            GREATEST(
              FLOOR(EXTRACT(EPOCH FROM (NOW() - ra.requested_at)) / 60),
              0
            )::int AS "requestedMinutesAgo",
            ra.note
          FROM remote_actions ra
          LEFT JOIN apps a ON a.id = ra.app_id
          LEFT JOIN servers s ON s.id = ra.server_id
          WHERE ra.workspace_id = $1::uuid
          ORDER BY ra.requested_at DESC
          LIMIT 6
        `,
        [workspaceId],
      ),
      query<EntitlementRow>(
        `
          SELECT
            feature_key AS "featureKey",
            status,
            quota,
            notes
          FROM workspace_entitlements
          WHERE workspace_id = $1::uuid
          ORDER BY feature_key ASC
        `,
        [workspaceId],
      ),
    ]);

    return normalizeOverview(
      "database",
      workspace,
      serversResult.rows,
      appsResult.rows,
      deploymentsResult.rows,
      actionsResult.rows,
      entitlementsResult.rows,
    );
  } catch (error) {
    console.error("Failed to load Cloud Connect overview", error);
    return demoOverview;
  }
});

export const getCloudLeadAdminDashboard = cache(
  async (): Promise<CloudLeadAdminDashboard> => {
    if (!isDatabaseConfigured()) {
      return {
        source: "demo",
        generatedAt: new Date().toISOString(),
        summary: summarizeLeads(demoLeads),
        leads: demoLeads,
      };
    }

    try {
      const [summaryResult, leadsResult] = await Promise.all([
        query<LeadSummaryRow>(
          `
            SELECT
              COUNT(*)::int AS total,
              COUNT(*) FILTER (WHERE status = 'new')::int AS new,
              COUNT(*) FILTER (WHERE status = 'reviewing')::int AS reviewing,
              COUNT(*) FILTER (WHERE status IN ('qualified', 'pilot'))::int AS qualified,
              COUNT(*) FILTER (WHERE priority IN ('high', 'urgent'))::int AS "highPriority"
            FROM interest_leads
          `,
        ),
        query<LeadRow>(
          `
            SELECT
              id::text AS id,
              email,
              team_name AS "teamName",
              fleet_size AS "fleetSize",
              needs,
              status,
              priority,
              owner_name AS "ownerName",
              review_notes AS "reviewNotes",
              customer_token::text AS "customerToken",
              created_at::text AS "createdAt",
              updated_at::text AS "updatedAt",
              last_contacted_at::text AS "lastContactedAt"
            FROM interest_leads
            ORDER BY
              CASE priority
                WHEN 'urgent' THEN 0
                WHEN 'high' THEN 1
                WHEN 'medium' THEN 2
                ELSE 3
              END,
              created_at DESC
          `,
        ),
      ]);

      const rows = leadsResult.rows.map(normalizeLead);
      return {
        source: rows.length > 0 ? "database" : "database-empty",
        generatedAt: new Date().toISOString(),
        summary: summaryResult.rows[0] ?? summarizeLeads([]),
        leads: rows,
      };
    } catch (error) {
      console.error("Failed to load Cloud Connect leads", error);
      return {
        source: "demo",
        generatedAt: new Date().toISOString(),
        summary: summarizeLeads(demoLeads),
        leads: demoLeads,
      };
    }
  },
);

export async function getCloudCustomerPanel(
  token: string,
): Promise<CloudCustomerPanel | null> {
  if (!isDatabaseConfigured()) {
    const lead =
      demoLeads.find((item) => item.customerToken === token) ?? demoLeads[0];
    return toCustomerPanel("demo", lead);
  }

  try {
    const result = await query<LeadRow>(
      `
        SELECT
          id::text AS id,
          email,
          team_name AS "teamName",
          fleet_size AS "fleetSize",
          needs,
          status,
          priority,
          owner_name AS "ownerName",
          review_notes AS "reviewNotes",
          customer_token::text AS "customerToken",
          created_at::text AS "createdAt",
          updated_at::text AS "updatedAt",
          last_contacted_at::text AS "lastContactedAt"
        FROM interest_leads
        WHERE customer_token = $1::uuid
        LIMIT 1
      `,
      [token],
    );

    const row = result.rows[0];
    if (!row) {
      return null;
    }

    return toCustomerPanel("database", normalizeLead(row));
  } catch (error) {
    console.error("Failed to load customer panel", error);
    return null;
  }
}

export async function createCloudLead(input: {
  email: string;
  teamName: string | null;
  fleetSize: number | null;
  needs: string | null;
}) {
  if (!isDatabaseConfigured()) {
    return {
      stored: false as const,
      reason: "database-not-configured" as const,
      token: demoLeads[0].customerToken,
    };
  }

  const result = await query<{ customerToken: string }>(
    `
      INSERT INTO interest_leads (
        email,
        team_name,
        fleet_size,
        needs,
        status,
        priority
      )
      VALUES (
        $1,
        NULLIF($2, ''),
        $3,
        NULLIF($4, ''),
        'new',
        CASE
          WHEN $3 IS NOT NULL AND $3 >= 20 THEN 'high'
          WHEN $3 IS NOT NULL AND $3 >= 10 THEN 'medium'
          ELSE 'low'
        END
      )
      ON CONFLICT (email)
      DO UPDATE SET
        team_name = EXCLUDED.team_name,
        fleet_size = EXCLUDED.fleet_size,
        needs = EXCLUDED.needs,
        updated_at = NOW()
      RETURNING customer_token::text AS "customerToken"
    `,
    [input.email, input.teamName, input.fleetSize, input.needs],
  );

  return {
    stored: true as const,
    token: result.rows[0]?.customerToken ?? demoLeads[0].customerToken,
  };
}

export async function updateCloudLeadReview(input: {
  leadId: string;
  status: LeadStatus;
  priority: LeadPriority;
  ownerName: string | null;
  reviewNotes: string | null;
}) {
  if (!isDatabaseConfigured()) {
    return { updated: false as const, reason: "database-not-configured" as const };
  }

  await query(
    `
      UPDATE interest_leads
      SET
        status = $2,
        priority = $3,
        owner_name = NULLIF($4, ''),
        review_notes = NULLIF($5, ''),
        last_contacted_at = NOW(),
        updated_at = NOW()
      WHERE id = $1::uuid
    `,
    [input.leadId, input.status, input.priority, input.ownerName, input.reviewNotes],
  );

  return { updated: true as const };
}
