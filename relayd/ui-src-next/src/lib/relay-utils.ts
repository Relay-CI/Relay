// ── Ported from ui-src/src/index.jsx ─────────────────────────────────────
// All pure utility functions from the original Vite/React dashboard.

import type { Deploy, EnvInfo, Project } from "./api";

export type { Project as NormalizedProject };

// ── Keys ──────────────────────────────────────────────────────────────────

export function deployKey(app: string, env: string, branch: string): string {
  return `${app}__${env}__${branch}`;
}

// ── Duration + time ───────────────────────────────────────────────────────

export function prettyDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "0ms";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(ms < 10000 ? 1 : 0)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.round((ms % 60000) / 1000);
  return `${minutes}m ${seconds}s`;
}

export function timeAgo(input: string | null | undefined): string {
  if (!input) return "now";
  const value = new Date(input);
  const diff = Math.max(0, Date.now() - value.getTime());
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h`;
  const day = Math.floor(hr / 24);
  if (day < 30) return `${day}d`;
  const month = Math.floor(day / 30);
  if (month < 12) return `${month}mo`;
  return `${Math.floor(month / 12)}y`;
}

export function formatDateTime(input: string | undefined | null): string {
  if (!input) return "Waiting";
  const value = new Date(input);
  if (!Number.isFinite(value.getTime())) return "Unknown";
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(value);
}

export function deployDurationMs(deploy: Deploy): number {
  if (!deploy?.started_at || !deploy?.ended_at) return 0;
  return Math.max(
    0,
    new Date(deploy.ended_at).getTime() - new Date(deploy.started_at).getTime(),
  );
}

export function deployDurationLabel(deploy: Deploy): string {
  const duration = deployDurationMs(deploy);
  if (duration) return prettyDuration(duration);
  return ACTIVE_STATUSES.has(deploy?.status) ? "running" : "n/a";
}

// ── URL helpers ───────────────────────────────────────────────────────────

export function computePreviewURL(
  envInfo: EnvInfo | undefined | null,
  deploy: Deploy | undefined | null,
): string {
  if (deploy?.preview_url) return deploy.preview_url;
  if (envInfo?.public_host) return `https://${envInfo.public_host}`;
  if (envInfo?.host_port) {
    return `${typeof window !== "undefined" ? window.location.protocol : "http:"}//${typeof window !== "undefined" ? window.location.hostname : "localhost"}:${envInfo.host_port}`;
  }
  return "";
}

export function computeConfiguredURL(
  envInfo: Partial<EnvInfo> | undefined | null,
): string {
  if (envInfo?.public_host) return `https://${envInfo.public_host}`;
  if ((envInfo?.mode || "port") === "port" && envInfo?.host_port) {
    return `${typeof window !== "undefined" ? window.location.protocol : "http:"}//${typeof window !== "undefined" ? window.location.hostname : "localhost"}:${envInfo.host_port}`;
  }
  return "";
}

export function sanitizeRepoURL(value: string): string {
  return value
    .replace(/^https?:\/\//i, "")
    .replace(/^git@/i, "")
    .replace(":", "/")
    .replace(/\.git$/i, "")
    .replace(/\/$/, "");
}

export function projectRepoURL(project: Project): string {
  return (project?.envs ?? []).map((env) => env.repo_url).find(Boolean) ?? "";
}

export function targetInspectURL(baseURL: string, target: string): string {
  if (!baseURL) return "";
  try {
    const url = new URL(
      baseURL,
      typeof window !== "undefined" ? window.location.href : "http://localhost",
    );
    url.searchParams.set("__relay_target", target);
    return url.toString();
  } catch {
    return "";
  }
}

// ── Repo info ─────────────────────────────────────────────────────────────

export interface RepoProviderInfo {
  connected: boolean;
  vendor: string;
  label: string;
  host: string;
  tone: string;
}

export function repoProviderInfo(repoURL: string): RepoProviderInfo {
  if (!repoURL) {
    return {
      connected: false,
      vendor: "Manual",
      label: "Manual deploy only",
      host: "No Git remote linked yet.",
      tone: "muted",
    };
  }
  const host = sanitizeRepoURL(repoURL);
  const lower = host.toLowerCase();
  if (lower.includes("github.com"))
    return {
      connected: true,
      vendor: "GitHub",
      label: "GitHub connected",
      host,
      tone: "teal",
    };
  if (lower.includes("gitlab.com"))
    return {
      connected: true,
      vendor: "GitLab",
      label: "GitLab connected",
      host,
      tone: "amber",
    };
  if (lower.includes("bitbucket"))
    return {
      connected: true,
      vendor: "Bitbucket",
      label: "Bitbucket connected",
      host,
      tone: "teal",
    };
  return {
    connected: true,
    vendor: "Git",
    label: "Git remote linked",
    host,
    tone: "amber",
  };
}

// ── Format helpers ────────────────────────────────────────────────────────

export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return "0%";
  return `${Math.round(value)}%`;
}

export function formatCommitSHA(value: string | undefined): string {
  return value ? value.slice(0, 7) : "manual";
}

export function shortImageTag(image: string | undefined): string {
  if (!image) return "n/a";
  const last = image.split(":").pop() ?? image;
  return last.length > 14 ? last.slice(0, 14) : last;
}

// ── Deploy status helpers ─────────────────────────────────────────────────

export const ACTIVE_STATUSES = new Set(["queued", "running", "building"]);

export function hasActiveDeploysIn(deploys: Deploy[]): boolean {
  return deploys.some((d) => ACTIVE_STATUSES.has(d.status));
}

export function deployStatusClass(status: string): string {
  if (status === "success") return "status-chip--ok";
  if (status === "failed" || status === "error") return "status-chip--danger";
  return "status-chip--warn";
}

export function deployPhaseText(deploy: Deploy | null | undefined): string {
  if (!deploy) return "idle";
  if (deploy.source === "promote" && ACTIVE_STATUSES.has(deploy.status))
    return "promotion in progress";
  if (deploy.source === "promote" && deploy.status === "success")
    return "promotion complete";
  if (
    deploy.source === "promote" &&
    (deploy.status === "failed" || deploy.status === "error")
  )
    return "promotion failed";
  if (deploy.source === "rollback" && ACTIVE_STATUSES.has(deploy.status))
    return "rollback in progress";
  if (deploy.source === "rollback" && deploy.status === "success")
    return "rollback complete";
  if (
    deploy.source === "rollback" &&
    (deploy.status === "failed" || deploy.status === "error")
  )
    return "rollback failed";
  if (ACTIVE_STATUSES.has(deploy.status)) return "deploy in progress";
  return deploy.status;
}

export function operationLabel(source: string | undefined): string {
  if (source === "promote") return "promotion";
  if (source === "rollback") return "rollback";
  if (source === "git") return "git deploy";
  if (source === "sync") return "sync deploy";
  return "deploy";
}

export function operationClass(source: string | undefined): string {
  if (source === "promote") return "operation-chip--git";
  if (source === "rollback") return "operation-chip--rollback";
  if (source === "git") return "operation-chip--git";
  if (source === "sync") return "operation-chip--sync";
  return "";
}

// ── Environment/lane helpers ──────────────────────────────────────────────

export function hasSlotRollout(envInfo: EnvInfo | null | undefined): boolean {
  return Boolean(envInfo?.active_slot);
}

export function rolloutStrategy(envInfo: EnvInfo | null | undefined): string {
  return hasSlotRollout(envInfo) ? "new / old handoff" : "single target";
}

export function liveTargetLabel(envInfo: EnvInfo | null | undefined): string {
  return hasSlotRollout(envInfo) ? "new live" : "single target";
}

export function oldTargetLabel(envInfo: EnvInfo | null | undefined): string {
  if (!hasSlotRollout(envInfo)) return "not staged";
  return (envInfo as EnvInfo)?.standby_slot ? "old draining" : "removed";
}

export function drainWindowLabel(envInfo: EnvInfo | null | undefined): string {
  if (!hasSlotRollout(envInfo)) return "n/a";
  if (!(envInfo as EnvInfo)?.standby_slot) return "complete";
  const remaining = Number((envInfo as EnvInfo)?.drain_until ?? 0) - Date.now();
  return remaining > 0 ? prettyDuration(remaining) : "expiring";
}

export function internalSlotLabel(slot: string | undefined): string {
  return slot ? `Internal slot id: ${slot}.` : "";
}

export function trafficModeLabel(value: string | undefined): string {
  return value === "session" ? "session sticky" : "edge cutover";
}

export function engineLabel(value: string | undefined): string {
  return value === "station" ? "Station" : "Docker";
}

export function normalizeEngineValue(value: string | undefined): string {
  return value === "station" ? "station" : "docker";
}

export function applyEngineConstraints<T extends { engine?: string }>(
  config: T,
): T {
  return { ...config, engine: normalizeEngineValue(config?.engine) };
}

// ── Mode translation: backend uses "port"/"traefik"/"", UI uses "http"/"static"/"off" ──

export function apiModeToUi(apiMode: string | undefined): string {
  if (apiMode === "traefik") return "static";
  if (apiMode === "" || apiMode === "off") return "off";
  return "http"; // "port" or default
}

export function uiModeToApi(uiMode: string | undefined): string {
  if (uiMode === "static") return "traefik";
  if (uiMode === "off") return "";
  return "port"; // "http" or default
}

// ── Traffic mode translation: backend uses "edge"/"session", UI uses "bluegreen"/"rolling" ──

export function apiTrafficModeToUi(apiMode: string | undefined): string {
  if (apiMode === "session") return "rolling";
  return "bluegreen"; // "edge" or default
}

export function uiTrafficModeToApi(uiMode: string | undefined): string {
  if (uiMode === "rolling") return "session";
  return "edge"; // "bluegreen" or default
}

export function buildSettingsConfig(
  selectedEnv?:
    | EnvInfo
    | {
        repo_url?: string;
        engine?: string;
        mode?: string;
        traffic_mode?: string;
        access_policy?: string;
        ip_allowlist?: string;
        expires_at?: number;
        host_port?: number;
        service_port?: number;
        public_host?: string;
        webhook_secret?: string;
      }
    | null,
): Record<string, unknown> {
  return applyEngineConstraints({
    repo_url: selectedEnv?.repo_url ?? "",
    engine: normalizeEngineValue(selectedEnv?.engine),
    mode: apiModeToUi(selectedEnv?.mode ?? "port"),
    traffic_mode: apiTrafficModeToUi(selectedEnv?.traffic_mode ?? "edge"),
    access_policy: selectedEnv?.access_policy ?? "public",
    ip_allowlist: selectedEnv?.ip_allowlist ?? "",
    expires_at: selectedEnv?.expires_at ?? 0,
    host_port: selectedEnv?.host_port ?? 0,
    service_port: selectedEnv?.service_port ?? 0,
    public_host: selectedEnv?.public_host ?? "",
    webhook_secret: selectedEnv?.webhook_secret ?? "",
  });
}

// ── Companion helpers ─────────────────────────────────────────────────────

export function prettyCompanionType(value: string | undefined): string {
  const kind = (value ?? "custom").toLowerCase();
  if (kind === "postgres") return "Postgres";
  if (kind === "redis") return "Redis";
  if (kind === "mysql") return "MySQL";
  if (kind === "mongo") return "Mongo";
  if (kind === "worker") return "Worker";
  return "Custom";
}

export function defaultCompanionDraft(kind = "postgres") {
  const base = {
    name: "",
    type: kind,
    version: "",
    image: "",
    command: "",
    stopped: false,
    port: 0,
    host_port: 0,
    env: {},
    volumes: [],
    health: {
      test: "",
      interval_seconds: 0,
      timeout_seconds: 0,
      retries: 0,
      start_period_seconds: 0,
    },
  };
  if (kind === "postgres")
    return { ...base, name: "db", version: "16", port: 5432 };
  if (kind === "redis")
    return { ...base, name: "cache", version: "7", port: 6379 };
  if (kind === "worker")
    return {
      ...base,
      name: "worker",
      image: "ghcr.io/your-org/worker:latest",
      command: "node worker.js",
    };
  return {
    ...base,
    name: "custom",
    type: "custom",
    image: "ghcr.io/your-org/service:latest",
  };
}

export function envToText(env: Record<string, string> | undefined): string {
  return Object.entries(env ?? {})
    .map(([key, value]) => `${key}=${value}`)
    .join("\n");
}

export function textToEnv(text: string): Record<string, string> {
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .reduce<Record<string, string>>((acc, line) => {
      const idx = line.indexOf("=");
      if (idx <= 0) return acc;
      acc[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
      return acc;
    }, {});
}

// ── Project normalization ─────────────────────────────────────────────────

export function normalizeProjects(projects: Project[]): Project[] {
  return (projects ?? []).map((project) => ({
    ...project,
    envs: (project?.envs ?? []).map((env) => ({
      ...env,
      engine: env?.engine ?? "docker",
    })),
    services: project?.services ?? [],
  }));
}

// ── Log helpers ───────────────────────────────────────────────────────────

export function logLineTone(line: string): string {
  const lower = (line ?? "").toLowerCase();
  if (!lower) return "muted";
  if (
    lower.includes("error") ||
    lower.includes("failed") ||
    lower.includes("panic") ||
    lower.includes("fatal")
  )
    return "danger";
  if (lower.includes("warn")) return "warn";
  if (
    lower.includes("ready") ||
    lower.includes("done") ||
    lower.includes("complete") ||
    lower.includes("success")
  )
    return "ok";
  return "neutral";
}

export function buildLogStats(lines: string[]) {
  return lines.reduce(
    (acc, line) => {
      acc.total += 1;
      const tone = logLineTone(line);
      if (tone === "danger") acc.errors += 1;
      if (tone === "warn") acc.warnings += 1;
      if (tone === "ok") acc.success += 1;
      return acc;
    },
    { total: 0, errors: 0, warnings: 0, success: 0 },
  );
}

export function runtimeLogLevel(line: string): string {
  const lower = (line ?? "").toLowerCase();
  if (lower.includes("fatal") || lower.includes("panic")) return "fatal";
  if (lower.includes("error") || lower.includes("failed")) return "error";
  if (lower.includes("warn")) return "warning";
  return "info";
}

export function runtimeLogLevelVariant(level: string): string {
  if (level === "fatal" || level === "error") return "danger";
  if (level === "warning") return "warning";
  return "muted";
}

export interface RuntimeLogEntry {
  raw: string;
  timestamp: string;
  time: string;
  message: string;
  level: string;
  host: string;
  request: string;
}

export function parseRuntimeLogEntry(
  raw: string,
  targetLabel: string,
): RuntimeLogEntry {
  const match = String(raw ?? "").match(/^(\d{4}-\d{2}-\d{2}T[^\s]+)\s(.*)$/);
  const timestamp = match?.[1] ?? "";
  const message = match?.[2] ?? String(raw ?? "");
  const level = runtimeLogLevel(message);
  return {
    raw: String(raw ?? ""),
    timestamp,
    time: timestamp
      ? formatDateTime(timestamp)
      : formatDateTime(new Date().toISOString()),
    message,
    level,
    host: targetLabel ?? "runtime",
    request: targetLabel ?? "runtime",
  };
}

export function sinceISO(windowFilter: string): string {
  const map: Record<string, number> = {
    "30m": 30 * 60 * 1000,
    "6h": 6 * 60 * 60 * 1000,
    "24h": 24 * 60 * 60 * 1000,
  };
  return new Date(Date.now() - (map[windowFilter] ?? map["30m"])).toISOString();
}

export function runtimeFilterMatches(
  entry: RuntimeLogEntry,
  levelFilter: string,
  query: string,
): boolean {
  const matchesLevel =
    levelFilter === "all" ||
    (levelFilter === "warning" && entry.level === "warning") ||
    (levelFilter === "error" && entry.level === "error") ||
    (levelFilter === "fatal" && entry.level === "fatal");
  if (!matchesLevel) return false;
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) return true;
  return [entry.message, entry.host, entry.request, entry.level]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(normalizedQuery));
}

// ── Runtime lane state ────────────────────────────────────────────────────

export interface RuntimeLaneState {
  appStopped: boolean;
  appRunning: boolean;
  hasRunningTarget: boolean;
  activeSlot: string;
  standbySlot: string;
  offlineReason: string;
}

export const EMPTY_RUNTIME_LANE_STATE: RuntimeLaneState = Object.freeze({
  appStopped: false,
  appRunning: false,
  hasRunningTarget: false,
  activeSlot: "",
  standbySlot: "",
  offlineReason: "",
});

export function normalizeRuntimeLaneState(
  lane:
    | {
        app_stopped?: boolean;
        app_running?: boolean;
        has_running_target?: boolean;
        active_slot?: string;
        standby_slot?: string;
        offline_reason?: string;
      }
    | null
    | undefined,
  selectedEnv: EnvInfo | null | undefined,
): RuntimeLaneState {
  return {
    appStopped: Boolean(lane?.app_stopped ?? selectedEnv?.stopped),
    appRunning: Boolean(lane?.app_running),
    hasRunningTarget: Boolean(lane?.has_running_target),
    activeSlot: lane?.active_slot ?? "",
    standbySlot: lane?.standby_slot ?? "",
    offlineReason:
      lane?.offline_reason ??
      (selectedEnv?.stopped
        ? "This app lane is currently off. Start or redeploy it to resume runtime logs."
        : ""),
  };
}

export function isRuntimeOfflineError(
  message: string,
  laneState: RuntimeLaneState,
): boolean {
  const normalized = String(message ?? "")
    .trim()
    .toLowerCase();
  if (!normalized) return Boolean(laneState?.appStopped);
  return (
    Boolean(laneState?.appStopped) ||
    normalized.includes("no such container") ||
    normalized.includes("currently off") ||
    normalized.includes("offline")
  );
}

export function humanizeRuntimeStreamError(
  message: string,
  laneState: RuntimeLaneState,
  targetMeta: { label?: string } | null | undefined,
): string {
  const raw = String(message ?? "").trim();
  if (!raw) return laneState?.offlineReason ?? "";
  if (
    /no such container/i.test(raw) ||
    (/exit status 1/i.test(raw) && laneState?.offlineReason)
  ) {
    return (
      laneState?.offlineReason ||
      `${targetMeta?.label ?? "Selected runtime target"} is offline right now.`
    );
  }
  return raw;
}

export function describeRuntimeLaneState(
  selectedEnv: EnvInfo | null | undefined,
  laneState: RuntimeLaneState,
  runningTargetCount: number,
): { title: string; body: string; badgeLabel: string; badgeVariant: string } {
  const liveLabel = `${runningTargetCount} live target${runningTargetCount === 1 ? "" : "s"}`;

  if (laneState.appStopped || selectedEnv?.stopped) {
    return {
      title: "App lane is off",
      body: `${laneState.offlineReason || "This app lane is currently off. Start or redeploy it to resume runtime logs."}${runningTargetCount ? " Running helper targets can still be inspected below." : ""}`,
      badgeLabel: runningTargetCount ? liveLabel : "No live targets",
      badgeVariant: "warning",
    };
  }
  if (!laneState.appRunning) {
    return {
      title: "Live app container unavailable",
      body:
        laneState.offlineReason ||
        "Relay cannot find a running app container for this lane yet.",
      badgeLabel: runningTargetCount ? liveLabel : "No live targets",
      badgeVariant: "warning",
    };
  }
  if (!laneState.hasRunningTarget) {
    return {
      title: "No runtime targets online",
      body:
        laneState.offlineReason ||
        "Runtime containers for this lane are not currently running.",
      badgeLabel: "No live targets",
      badgeVariant: "muted",
    };
  }
  return {
    title: "Runtime streaming ready",
    body: "Select a live target to follow container output in real time.",
    badgeLabel: liveLabel,
    badgeVariant: "success",
  };
}

// ── Stats helpers ─────────────────────────────────────────────────────────

export interface ProjectStats {
  total: number;
  successful: number;
  successes: number;
  failures: number;
  successRate: number;
  avgDuration: number;
  active: number;
  blueGreenContexts: number;
}

export function computeProjectStats(deploys: Deploy[]): ProjectStats {
  const total = deploys.length;
  const successful = deploys.filter((d) => d.status === "success").length;
  const failures = deploys.filter(
    (d) => d.status === "failed" || d.status === "error",
  ).length;
  const successRate = total ? (successful / total) * 100 : 0;
  const durations = deploys.map(deployDurationMs).filter(Boolean);
  const avgDuration = durations.length
    ? Math.round(durations.reduce((s, v) => s + v, 0) / durations.length)
    : 0;
  const active = deploys.filter((d) => ACTIVE_STATUSES.has(d.status)).length;
  return {
    total,
    successful,
    successes: successful,
    failures,
    successRate,
    avgDuration,
    active,
    blueGreenContexts: 0,
  };
}
