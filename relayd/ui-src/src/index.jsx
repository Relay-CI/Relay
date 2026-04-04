import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";
import { Button } from "./components/ui/button";
import { Badge } from "./components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./components/ui/card";
import { Input } from "./components/ui/input";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "./components/ui/dialog";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "./components/ui/accordion";
import { ScrollArea } from "./components/ui/scroll-area";
import { Tabs, TabsList, TabsTrigger } from "./components/ui/tabs";

const API = "";

function cx(...parts) {
  return parts.filter(Boolean).join(" ");
}

function RelayMark({ className, title = "Relay" }) {
  return (
    <svg className={className} viewBox="0 0 256 256" role="img" aria-label={title}>
      <path fill="#E8E6E3" d="M128 138L92 110L82 122L118 150V190H138V150L174 122L164 110Z" />
      <circle cx="82" cy="122" r="18" fill="#E8E6E3" />
      <circle cx="128" cy="200" r="18" fill="#E8E6E3" />
      <circle cx="174" cy="122" r="18" fill="#6FC2A6" />
    </svg>
  );
}

const NAV_ICONS = {
  overview: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" />
      <rect x="3" y="14" width="7" height="7" /><rect x="14" y="14" width="7" height="7" />
    </svg>
  ),
  deployments: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <line x1="6" y1="3" x2="6" y2="15" /><circle cx="18" cy="6" r="3" /><circle cx="6" cy="18" r="3" />
      <path d="M18 9a9 9 0 0 1-9 9" />
    </svg>
  ),
  logs: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 6h16" />
      <path d="M4 12h10" />
      <path d="M4 18h13" />
      <circle cx="18" cy="12" r="2.5" />
      <circle cx="20" cy="18" r="1.5" />
    </svg>
  ),
  settings: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 7H4" /><path d="M4 12h16" /><path d="M4 17h16" />
      <circle cx="8" cy="7" r="2" /><circle cx="16" cy="12" r="2" /><circle cx="8" cy="17" r="2" />
    </svg>
  ),
  server: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
      <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
      <line x1="6" y1="6" x2="6.01" y2="6"/>
      <line x1="6" y1="18" x2="6.01" y2="18"/>
    </svg>
  ),
  analytics: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/>
      <line x1="6" y1="20" x2="6" y2="14"/><line x1="3" y1="20" x2="21" y2="20"/>
    </svg>
  ),
};

async function api(path, options = {}) {
  const res = await fetch(`${API}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      "X-Requested-With": "relay-dashboard",
      ...(options.headers || {}),
    },
    ...options,
  });

  const contentType = res.headers.get("content-type") || "";
  const data = contentType.includes("application/json") ? await res.json() : await res.text();

  if (!res.ok) {
    const error = typeof data === "object" && data?.error ? data.error : `HTTP ${res.status}`;
    const err = new Error(error);
    err.status = res.status;
    throw err;
  }

  return data;
}

function deployKey(app, env, branch) {
  return `${app}__${env}__${branch}`;
}

function prettyDuration(ms) {
  if (!Number.isFinite(ms) || ms < 0) return "0ms";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(ms < 10000 ? 1 : 0)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.round((ms % 60000) / 1000);
  return `${minutes}m ${seconds}s`;
}

function timeAgo(input) {
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

function computePreviewURL(envInfo, deploy) {
  if (deploy?.preview_url) return deploy.preview_url;
  if (envInfo?.public_host) return `https://${envInfo.public_host}`;
  if (envInfo?.host_port) return `${window.location.protocol}//${window.location.hostname}:${envInfo.host_port}`;
  return "";
}

function computeConfiguredURL(envInfo) {
  if (envInfo?.public_host) return `https://${envInfo.public_host}`;
  if ((envInfo?.mode || "port") === "port" && envInfo?.host_port) {
    return `${window.location.protocol}//${window.location.hostname}:${envInfo.host_port}`;
  }
  return "";
}

function deployDurationMs(deploy) {
  if (!deploy?.started_at || !deploy?.ended_at) return 0;
  return Math.max(0, new Date(deploy.ended_at) - new Date(deploy.started_at));
}

function formatPercent(value) {
  if (!Number.isFinite(value)) return "0%";
  return `${Math.round(value)}%`;
}

function hasSlotRollout(envInfo) {
  return Boolean(envInfo?.active_slot);
}

function rolloutStrategy(envInfo) {
  return hasSlotRollout(envInfo) ? "new / old handoff" : "single target";
}

function liveTargetLabel(envInfo) {
  return hasSlotRollout(envInfo) ? "new live" : "single target";
}

function oldTargetLabel(envInfo) {
  if (!hasSlotRollout(envInfo)) return "not staged";
  return envInfo?.standby_slot ? "old draining" : "removed";
}

function drainWindowLabel(envInfo) {
  if (!hasSlotRollout(envInfo)) return "n/a";
  if (!envInfo?.standby_slot) return "complete";
  const remaining = Number(envInfo?.drain_until || 0) - Date.now();
  return remaining > 0 ? prettyDuration(remaining) : "expiring";
}

function internalSlotLabel(slot) {
  return slot ? `Internal slot id: ${slot}.` : "";
}

function targetInspectURL(baseURL, target) {
  if (!baseURL) return "";
  try {
    const url = new URL(baseURL, window.location.href);
    url.searchParams.set("__relay_target", target);
    return url.toString();
  } catch {
    return "";
  }
}

function trafficModeLabel(value) {
  return value === "session" ? "session sticky" : "edge cutover";
}

function engineLabel(value) {
  return value === "station" ? "Station" : "Docker";
}

function normalizeEngineValue(value) {
  return value === "station" ? "station" : "docker";
}

function applyEngineConstraints(config) {
  return {
    ...config,
    engine: normalizeEngineValue(config?.engine),
  };
}

function buildSettingsConfig(selectedEnv) {
  return applyEngineConstraints({
    repo_url: selectedEnv?.repo_url || "",
    engine: normalizeEngineValue(selectedEnv?.engine),
    mode: selectedEnv?.mode || "port",
    traffic_mode: selectedEnv?.traffic_mode || "edge",
    host_port: selectedEnv?.host_port || 0,
    service_port: selectedEnv?.service_port || 0,
    public_host: selectedEnv?.public_host || "",
    webhook_secret: selectedEnv?.webhook_secret || "",
  });
}

function prettyCompanionType(value) {
  const kind = (value || "custom").toLowerCase();
  if (kind === "postgres") return "Postgres";
  if (kind === "redis") return "Redis";
  if (kind === "mysql") return "MySQL";
  if (kind === "mongo") return "Mongo";
  if (kind === "worker") return "Worker";
  return "Custom";
}

function defaultCompanionDraft(kind = "postgres") {
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
  if (kind === "postgres") {
    return { ...base, name: "db", version: "16", port: 5432 };
  }
  if (kind === "redis") {
    return { ...base, name: "cache", version: "7", port: 6379 };
  }
  if (kind === "worker") {
    return { ...base, name: "worker", image: "ghcr.io/your-org/worker:latest", command: "node worker.js" };
  }
  return { ...base, name: "custom", type: "custom", image: "ghcr.io/your-org/service:latest" };
}

function envToText(env) {
  return Object.entries(env || {}).map(([key, value]) => `${key}=${value}`).join("\n");
}

function textToEnv(text) {
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .reduce((acc, line) => {
      const idx = line.indexOf("=");
      if (idx <= 0) return acc;
      acc[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
      return acc;
    }, {});
}

function normalizeProjects(projects) {
  return (projects || []).map((project) => ({
    ...project,
    envs: (project?.envs || []).map((env) => ({
      ...env,
      engine: env?.engine || "docker",
    })),
    services: project?.services || [],
  }));
}

const ACTIVE_STATUSES = new Set(["queued", "running", "building"]);

function hasActiveDeploysIn(deploys) {
  return deploys.some((d) => ACTIVE_STATUSES.has(d.status));
}

function deployStatusClass(status) {
  if (status === "success") return "status-chip--ok";
  if (status === "failed" || status === "error") return "status-chip--danger";
  return "status-chip--warn"; // queued, running, building, etc.
}

function operationLabel(source) {
  if (source === "rollback") return "rollback";
  if (source === "git") return "git deploy";
  if (source === "sync") return "sync deploy";
  return "deploy";
}

function operationClass(source) {
  if (source === "rollback") return "operation-chip--rollback";
  if (source === "git") return "operation-chip--git";
  if (source === "sync") return "operation-chip--sync";
  return "";
}

function deployPhaseText(deploy) {
  if (!deploy) return "idle";
  if (deploy.source === "rollback" && ACTIVE_STATUSES.has(deploy.status)) return "rollback in progress";
  if (deploy.source === "rollback" && deploy.status === "success") return "rollback complete";
  if (deploy.source === "rollback" && (deploy.status === "failed" || deploy.status === "error")) return "rollback failed";
  if (ACTIVE_STATUSES.has(deploy.status)) return "deploy in progress";
  return deploy.status;
}

function shortImageTag(image) {
  if (!image) return "n/a";
  const last = image.split(":").pop() || image;
  return last.length > 14 ? last.slice(0, 14) : last;
}

function deployDurationLabel(deploy) {
  const duration = deployDurationMs(deploy);
  if (duration) return prettyDuration(duration);
  return ACTIVE_STATUSES.has(deploy?.status) ? "running" : "n/a";
}

function formatDateTime(input) {
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

function formatCommitSHA(value) {
  return value ? value.slice(0, 7) : "manual";
}

function sanitizeRepoURL(value) {
  if (!value) return "";
  return value
    .replace(/^https?:\/\//i, "")
    .replace(/^git@/i, "")
    .replace(":", "/")
    .replace(/\.git$/i, "")
    .replace(/\/$/, "");
}

function projectRepoURL(project) {
  return (project?.envs || []).map((envInfo) => envInfo.repo_url).find(Boolean) || "";
}

function repoProviderInfo(repoURL) {
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

  if (lower.includes("github.com")) return { connected: true, vendor: "GitHub", label: "GitHub connected", host, tone: "teal" };
  if (lower.includes("gitlab.com")) return { connected: true, vendor: "GitLab", label: "GitLab connected", host, tone: "amber" };
  if (lower.includes("bitbucket")) return { connected: true, vendor: "Bitbucket", label: "Bitbucket connected", host, tone: "teal" };
  return { connected: true, vendor: "Git", label: "Git remote linked", host, tone: "amber" };
}

function logLineTone(line) {
  const lower = (line || "").toLowerCase();
  if (!lower) return "muted";
  if (lower.includes("error") || lower.includes("failed") || lower.includes("panic") || lower.includes("fatal")) return "danger";
  if (lower.includes("warn")) return "warn";
  if (lower.includes("ready") || lower.includes("done") || lower.includes("complete") || lower.includes("success")) return "ok";
  return "neutral";
}

function buildLogStats(lines) {
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

function runtimeLogLevel(line) {
  const lower = (line || "").toLowerCase();
  if (lower.includes("fatal") || lower.includes("panic")) return "fatal";
  if (lower.includes("error") || lower.includes("failed")) return "error";
  if (lower.includes("warn")) return "warning";
  return "info";
}

function runtimeLogLevelVariant(level) {
  if (level === "fatal" || level === "error") return "danger";
  if (level === "warning") return "warning";
  return "muted";
}

function parseRuntimeLogEntry(raw, targetLabel) {
  const match = String(raw || "").match(/^(\d{4}-\d{2}-\d{2}T[^\s]+)\s(.*)$/);
  const timestamp = match?.[1] || "";
  const message = match?.[2] || String(raw || "");
  const level = runtimeLogLevel(message);
  return {
    raw: String(raw || ""),
    timestamp,
    time: timestamp ? formatDateTime(timestamp) : formatDateTime(new Date().toISOString()),
    message,
    level,
    host: targetLabel || "runtime",
    request: targetLabel || "runtime",
  };
}

function sinceISO(windowFilter) {
  const map = {
    "30m": 30 * 60 * 1000,
    "6h": 6 * 60 * 60 * 1000,
    "24h": 24 * 60 * 60 * 1000,
  };
  return new Date(Date.now() - (map[windowFilter] || map["30m"])).toISOString();
}

function runtimeFilterMatches(entry, levelFilter, query) {
  const matchesLevel =
    levelFilter === "all"
      || (levelFilter === "warning" && entry.level === "warning")
      || (levelFilter === "error" && entry.level === "error")
      || (levelFilter === "fatal" && entry.level === "fatal");

  if (!matchesLevel) return false;
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) return true;
  return [entry.message, entry.host, entry.request, entry.level]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(normalizedQuery));
}

const EMPTY_RUNTIME_LANE_STATE = Object.freeze({
  appStopped: false,
  appRunning: false,
  hasRunningTarget: false,
  activeSlot: "",
  standbySlot: "",
  offlineReason: "",
});

function normalizeRuntimeLaneState(lane, selectedEnv) {
  return {
    appStopped: Boolean(lane?.app_stopped ?? selectedEnv?.stopped),
    appRunning: Boolean(lane?.app_running),
    hasRunningTarget: Boolean(lane?.has_running_target),
    activeSlot: lane?.active_slot || "",
    standbySlot: lane?.standby_slot || "",
    offlineReason: lane?.offline_reason || (selectedEnv?.stopped ? "This app lane is currently off. Start or redeploy it to resume runtime logs." : ""),
  };
}

function isRuntimeOfflineError(message, laneState) {
  const normalized = String(message || "").trim().toLowerCase();
  if (!normalized) return Boolean(laneState?.appStopped);
  return Boolean(laneState?.appStopped)
    || normalized.includes("no such container")
    || normalized.includes("currently off")
    || normalized.includes("offline");
}

function humanizeRuntimeStreamError(message, laneState, targetMeta) {
  const raw = String(message || "").trim();
  if (!raw) return laneState?.offlineReason || "";
  if (/no such container/i.test(raw) || (/exit status 1/i.test(raw) && laneState?.offlineReason)) {
    return laneState?.offlineReason || `${targetMeta?.label || "Selected runtime target"} is offline right now.`;
  }
  return raw;
}

function describeRuntimeLaneState(selectedEnv, laneState, runningTargetCount) {
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
      body: laneState.offlineReason || "Relay cannot find a running app container for this lane yet.",
      badgeLabel: runningTargetCount ? liveLabel : "No live targets",
      badgeVariant: "warning",
    };
  }

  if (!laneState.hasRunningTarget) {
    return {
      title: "No runtime targets online",
      body: laneState.offlineReason || "Runtime containers for this lane are not currently running.",
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

function useDashboardData(enabled) {
  const [state, setState] = useState({
    loading: true,
    refreshing: false,
    projects: [],
    deploys: [],
    error: "",
  });

  function applyData(projects, deploys) {
    const sorted = (deploys || []).slice().sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
    setState({ loading: false, refreshing: false, projects: normalizeProjects(projects), deploys: sorted, error: "" });
  }

  useEffect(() => {
    if (!enabled) return undefined;
    const es = new EventSource(`${API}/api/events`, { withCredentials: true });

    es.addEventListener("snapshot", (e) => {
      try {
        const { projects, deploys } = JSON.parse(e.data);
        applyData(projects, deploys);
      } catch {}
    });
    es.addEventListener("update", (e) => {
      try {
        const { projects, deploys } = JSON.parse(e.data);
        applyData(projects, deploys);
      } catch {}
    });
    es.onerror = () => {
      setState((prev) => ({
        ...prev,
        loading: false,
        error: prev.loading ? "Cannot connect to agent" : "Connection lost — reconnecting…",
      }));
    };

    return () => es.close();
  }, [enabled]);

  const isLive = state.deploys.length > 0 && hasActiveDeploysIn(state.deploys);

  const manualRefresh = useCallback(async () => {
    setState((prev) => ({ ...prev, refreshing: true }));
    try {
      const [projects, deploys] = await Promise.all([api("/api/projects"), api("/api/deploys")]);
      applyData(projects, deploys);
    } catch (err) {
      setState((prev) => ({ ...prev, refreshing: false, error: err.message }));
    }
  }, []);

  return [{ ...state, isLive }, manualRefresh];
}

function useRuntimeLogs(selectedEnv, selectedTarget, windowFilter) {
  const [state, setState] = useState({
    loading: false,
    targets: [],
    defaultTarget: "",
    lines: [],
    status: "idle",
    targetMeta: null,
    lane: EMPTY_RUNTIME_LANE_STATE,
    error: "",
  });

  useEffect(() => {
    if (!selectedEnv?.app || !selectedEnv?.env || !selectedEnv?.branch) {
      setState({
        loading: false,
        targets: [],
        defaultTarget: "",
        lines: [],
        status: "idle",
        targetMeta: null,
        lane: EMPTY_RUNTIME_LANE_STATE,
        error: "",
      });
      return undefined;
    }

    let cancelled = false;
    setState((prev) => ({ ...prev, loading: true, error: "" }));

    api(`/api/runtime/logs/targets?app=${encodeURIComponent(selectedEnv.app)}&env=${encodeURIComponent(selectedEnv.env)}&branch=${encodeURIComponent(selectedEnv.branch)}`)
      .then((data) => {
        if (cancelled) return;
        const lane = normalizeRuntimeLaneState(data?.lane, selectedEnv);
        const defaultTarget = data?.default_target || "";
        setState((prev) => ({
          ...prev,
          loading: false,
          targets: data?.targets || [],
          defaultTarget,
          lines: [],
          status: defaultTarget ? "idle" : lane.hasRunningTarget ? "idle" : "offline",
          targetMeta: null,
          lane,
          error: "",
        }));
      })
      .catch((err) => {
        if (cancelled) return;
        setState((prev) => ({
          ...prev,
          loading: false,
          targets: [],
          defaultTarget: "",
          lines: [],
          status: "error",
          targetMeta: null,
          lane: normalizeRuntimeLaneState(null, selectedEnv),
          error: err.message,
        }));
      });

    return () => {
      cancelled = true;
    };
  }, [selectedEnv]);

  useEffect(() => {
    if (!selectedEnv?.app || !selectedEnv?.env || !selectedEnv?.branch) return undefined;
    const effectiveTarget = selectedTarget || state.defaultTarget;
    const effectiveTargetMeta = state.targets.find((target) => target.id === effectiveTarget) || null;
    const laneState = state.lane;

    if (!effectiveTarget) {
      setState((prev) => ({
        ...prev,
        lines: [],
        status: prev.lane.hasRunningTarget ? "idle" : "offline",
        targetMeta: null,
        error: "",
      }));
      return undefined;
    }

    if (effectiveTargetMeta && !effectiveTargetMeta.running) {
      setState((prev) => ({
        ...prev,
        lines: [],
        status: "offline",
        targetMeta: effectiveTargetMeta,
        error: "",
      }));
      return undefined;
    }

    const controller = new AbortController();
    let cancelled = false;
    setState((prev) => ({ ...prev, lines: [], status: "connecting", targetMeta: effectiveTargetMeta, error: "" }));

    async function run() {
      try {
        const params = new URLSearchParams({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
          target: effectiveTarget,
          tail: "300",
          since: sinceISO(windowFilter),
        });
        const res = await fetch(`/api/runtime/logs/stream?${params.toString()}`, {
          credentials: "include",
          signal: controller.signal,
        });
        if (!res.ok) {
          const contentType = res.headers.get("content-type") || "";
          const payload = contentType.includes("application/json") ? await res.json() : await res.text();
          const message =
            (typeof payload === "object" && payload?.error)
              || (typeof payload === "string" && payload)
              || `HTTP ${res.status}`;
          throw new Error(humanizeRuntimeStreamError(message, laneState, effectiveTargetMeta));
        }
        setState((prev) => ({ ...prev, status: "live" }));

        const reader = res.body.getReader();
        const decoder = new TextDecoder("utf-8");
        let buffer = "";

        while (!cancelled) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let idx;
          while ((idx = buffer.indexOf("\n\n")) >= 0) {
            const frame = buffer.slice(0, idx);
            buffer = buffer.slice(idx + 2);
            let eventName = "message";
            const data = [];
            frame.split("\n").forEach((line) => {
              if (line.startsWith("event: ")) eventName = line.slice(7).trim();
              if (line.startsWith("data: ")) data.push(line.slice(6));
            });
            if (!data.length) continue;
            const payload = data.join("\n");
            if (eventName === "runtime-target") {
              try {
                const parsed = JSON.parse(payload);
                setState((prev) => ({ ...prev, targetMeta: parsed }));
              } catch {}
              continue;
            }
            if (eventName === "runtime-status") {
              try {
                const parsed = JSON.parse(payload);
                setState((prev) => ({
                  ...prev,
                  ...(() => {
                    const nextError = humanizeRuntimeStreamError(parsed.error || prev.error, prev.lane, prev.targetMeta || effectiveTargetMeta);
                    return {
                      status: parsed.status === "error" && isRuntimeOfflineError(nextError, prev.lane) ? "offline" : (parsed.status || "complete"),
                      error: parsed.status === "error" ? nextError : (parsed.error ? nextError : prev.error),
                    };
                  })(),
                }));
              } catch {
                setState((prev) => ({ ...prev, status: payload || "complete" }));
              }
              continue;
            }
            setState((prev) => ({
              ...prev,
              lines: [...prev.lines, parseRuntimeLogEntry(payload, prev.targetMeta?.label || effectiveTarget)].slice(-500),
            }));
          }
        }

        setState((prev) => ({
          ...prev,
          status: prev.status === "live" ? "complete" : prev.status,
        }));
      } catch (err) {
        if (!cancelled && !controller.signal.aborted) {
          const message = humanizeRuntimeStreamError(err.message, laneState, effectiveTargetMeta);
          setState((prev) => ({
            ...prev,
            status: isRuntimeOfflineError(message, laneState) ? "offline" : "error",
            error: message,
          }));
        }
      }
    }

    run();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [selectedEnv, selectedTarget, state.defaultTarget, state.lane, state.targets, windowFilter]);

  return state;
}

function SplashScreen({ label }) {
  return (
    <div className="screen-center">
      <div className="panel splash">
        <div className="spinner" />
        <div>{label}</div>
      </div>
    </div>
  );
}

function LoginScreen({ onLogin, error, legacyMode }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState("");
  const [pending, setPending] = useState(false);

  return (
    <div className="screen-center">
      <form
        className="panel login-card"
        onSubmit={async (event) => {
          event.preventDefault();
          setPending(true);
          try {
            await onLogin(legacyMode ? token : { username, password });
          } finally {
            setPending(false);
          }
        }}
      >
        <RelayMark className="brand__glyph" title="Relay mark" />
        <div className="eyebrow">Secure Agent Access</div>
        <h1 className="login-card__title">Relay Control Room</h1>
        {legacyMode ? (
          <>
            <p className="login-card__body">
              Enter your relay token to access the dashboard.
            </p>
            <input
              className="text-input"
              type="password"
              autoComplete="current-password"
              placeholder="Paste relay token"
              value={token}
              onChange={(e) => setToken(e.target.value)}
            />
          </>
        ) : (
          <>
            <p className="login-card__body">
              Sign in to manage your deployments.
            </p>
            <input
              className="text-input"
              type="text"
              autoComplete="username"
              placeholder="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
            <input
              className="text-input"
              type="password"
              autoComplete="current-password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </>
        )}
        {error && <div className="error-banner">{error}</div>}
        <button
          type="submit"
          className="primary-button"
          disabled={(legacyMode ? !token : (!username || !password)) || pending}
        >
          {pending ? "Signing in…" : "Sign In"}
        </button>
        {legacyMode && (
          <div className="helper-row">
            Token source <span className="helper-pill">data/token.txt</span>
          </div>
        )}
      </form>
    </div>
  );
}

function SetupScreen({ onSetup, error }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pending, setPending] = useState(false);
  const mismatch = confirm && password !== confirm;

  return (
    <div className="screen-center">
      <form
        className="panel login-card"
        onSubmit={async (event) => {
          event.preventDefault();
          if (mismatch) return;
          setPending(true);
          try {
            await onSetup({ username, password });
          } finally {
            setPending(false);
          }
        }}
      >
        <RelayMark className="brand__glyph" title="Relay mark" />
        <div className="eyebrow">First-time Setup</div>
        <h1 className="login-card__title">Create Owner Account</h1>
        <p className="login-card__body">
          No accounts exist yet. Create the first owner account to secure your dashboard.
        </p>
        <input
          className="text-input"
          type="text"
          autoComplete="username"
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
        <input
          className="text-input"
          type="password"
          autoComplete="new-password"
          placeholder="Password (min 8 chars)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <input
          className="text-input"
          type="password"
          autoComplete="new-password"
          placeholder="Confirm password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
        />
        {mismatch && <div className="error-banner">Passwords do not match</div>}
        {error && <div className="error-banner">{error}</div>}
        <button
          type="submit"
          className="primary-button"
          disabled={!username || password.length < 8 || mismatch || pending}
        >
          {pending ? "Creating account…" : "Create Account"}
        </button>
      </form>
    </div>
  );
}

function ProjectSelector({ projects, selected, onSelect }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const ref = useRef(null);

  useEffect(() => {
    function onDocumentClick(event) {
      if (ref.current && !ref.current.contains(event.target)) setOpen(false);
    }
    document.addEventListener("mousedown", onDocumentClick);
    return () => document.removeEventListener("mousedown", onDocumentClick);
  }, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return projects;
    return projects.filter((project) => project.name.toLowerCase().includes(q));
  }, [projects, query]);

  const current = projects.find((project) => project.name === selected);

  return (
    <div className="project-selector" ref={ref}>
      <button type="button" className="selector-button" onClick={() => setOpen((value) => !value)}>
        <span>
          <span className="eyebrow">Project</span>
          <strong>{current?.name || "Select project"}</strong>
        </span>
        <span className="selector-button__count">{projects.length}</span>
      </button>
      {open && (
        <div className="selector-menu panel">
          <input
            className="text-input text-input--compact"
            type="text"
            placeholder="Search projects"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <div className="selector-list">
            {filtered.map((project) => (
              <button
                key={project.name}
                type="button"
                className={cx("selector-item", project.name === selected && "selector-item--active")}
                onClick={() => {
                  onSelect(project.name);
                  setOpen(false);
                }}
              >
                <div>
                  <strong>{project.name}</strong>
                  <div className="selector-item__meta">
                    {project.envs.length} envs · {project.services.length} services
                  </div>
                </div>
                <div className="selector-item__meta">
                  {project.latestDeployAt ? `${timeAgo(project.latestDeployAt)} ago` : "No deploys"}
                </div>
              </button>
            ))}
            {!filtered.length && <div className="empty-inline">No projects matched the filter.</div>}
          </div>
        </div>
      )}
    </div>
  );
}

function SourceBadge({ info, compact = false }) {
  return (
    <span className={cx("source-chip", `source-chip--${info.tone}`, compact && "source-chip--compact")}>
      {info.label}
    </span>
  );
}

function ProjectCommandSelector({ projects, selected, onSelect }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const ref = useRef(null);

  useEffect(() => {
    function onDocumentClick(event) {
      if (ref.current && !ref.current.contains(event.target)) setOpen(false);
    }
    document.addEventListener("mousedown", onDocumentClick);
    return () => document.removeEventListener("mousedown", onDocumentClick);
  }, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return projects;
    return projects.filter((project) => project.name.toLowerCase().includes(q));
  }, [projects, query]);

  const current = projects.find((project) => project.name === selected);
  const currentRepoInfo = repoProviderInfo(projectRepoURL(current));

  return (
    <div className="project-command-selector" ref={ref}>
      <button type="button" className="project-command-selector__button" onClick={() => setOpen((value) => !value)}>
        <div className="project-command-selector__copy">
          <span className="eyebrow">Project Surface</span>
          <strong>{current?.name || "Select project"}</strong>
          <span className="project-command-selector__meta">
            {current
              ? `${currentRepoInfo.label} · ${current.envs.length} environments · ${current.services.length} services`
              : "Pick a project, source connection, and deployment surface."}
          </span>
        </div>
        <div className="project-command-selector__actions">
          {current && <SourceBadge info={currentRepoInfo} compact />}
          <span className="selector-button__count">{projects.length}</span>
        </div>
      </button>

      {open && (
        <div className="selector-menu panel selector-menu--command">
          <input
            className="text-input text-input--compact"
            type="text"
            placeholder="Search projects"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <div className="selector-list">
            {filtered.map((project) => {
              const repoInfo = repoProviderInfo(projectRepoURL(project));
              return (
                <button
                  key={project.name}
                  type="button"
                  className={cx("selector-item", project.name === selected && "selector-item--active")}
                  onClick={() => {
                    onSelect(project.name);
                    setOpen(false);
                  }}
                >
                  <div className="selector-item__content">
                    <div>
                      <strong>{project.name}</strong>
                      <div className="selector-item__meta">
                        {project.envs.length} envs · {project.services.length} services
                      </div>
                    </div>
                    <div className="selector-item__footer">
                      <SourceBadge info={repoInfo} compact />
                      <span className="selector-item__meta selector-item__host">{repoInfo.host}</span>
                    </div>
                  </div>
                  <div className="selector-item__meta">
                    {project.latestDeployAt ? `${timeAgo(project.latestDeployAt)} ago` : "No deploys"}
                  </div>
                </button>
              );
            })}
            {!filtered.length && <div className="empty-inline">No projects matched the filter.</div>}
          </div>
        </div>
      )}
    </div>
  );
}

function MetricCard({ label, value, meta, accent, tone = "neutral" }) {
  return (
    <div className={cx("metric-card", accent && "metric-card--accent", tone !== "neutral" && `metric-card--${tone}`)}>
      <div className="metric-card__top">
        <div className="eyebrow">{label}</div>
        <span className="metric-card__spark" />
      </div>
      <div className="metric-card__value">{value}</div>
      <div className="metric-card__meta">{meta}</div>
    </div>
  );
}

function ContextBar({ contexts, selected, onSelect }) {
  return (
    <section className="context-lanes">
      <div className="section-card__header section-card__header--tight">
        <div>
          <div className="eyebrow">Delivery Lanes</div>
          <h3>Environment routing</h3>
        </div>
        <div className="muted">Choose the lane you want to inspect or operate.</div>
      </div>
      <div className="context-grid">
        {contexts.map((context) => {
          const key = deployKey(context.app, context.env, context.branch);
          return (
            <button
              key={key}
              type="button"
              className={cx("context-card panel", selected === key && "context-card--active")}
              onClick={() => onSelect(key)}
            >
              <div className="context-card__top">
                <span className={cx("status-chip", deployStatusClass(context.latestDeploy?.status))}>
                  <span className="status-dot" />
                  {context.env}
                </span>
                <span className="context-card__branch">{context.branch}</span>
              </div>
              <div className="context-card__headline">
                <strong>{context.env}</strong>
                <span>{liveTargetLabel(context)}</span>
              </div>
              <div className="context-card__url">
                {context.previewURL ? <span>{context.previewURL}</span> : <span className="muted">No public URL yet</span>}
              </div>
              <div className="inline-pills inline-pills--compact">
                <span className="metric-pill">{trafficModeLabel(context.traffic_mode)}</span>
                <span className="metric-pill">{engineLabel(context.engine || "docker")}</span>
                <span className="metric-pill">{rolloutStrategy(context)}</span>
              </div>
              <div className="context-card__footer">
                <span>{context.latestDeploy ? deployPhaseText(context.latestDeploy) : "idle"}</span>
                <span>{context.latestDeploy ? `${timeAgo(context.latestDeploy.created_at)} ago` : "waiting"}</span>
              </div>
            </button>
          );
        })}
      </div>
    </section>
  );
}

function Detail({ label, value, link }) {
  return (
    <div className="detail-card">
      <div className="eyebrow">{label}</div>
      {link ? (
        <a className="link-teal detail-card__value" href={link} target="_blank" rel="noreferrer">
          {value}
        </a>
      ) : (
        <div className="detail-card__value">{value}</div>
      )}
    </div>
  );
}

function ServiceCard({ service }) {
  return (
    <div className="row-card row-card--service">
      <div>
        <div className="row-card__title">
          {service.name} <span className="row-card__badge">{service.type}</span>
        </div>
        <div className="row-card__meta mono">{service.container}</div>
        <div className="row-card__meta mono">{service.network}</div>
      </div>
      <div className="row-card__actions row-card__actions--column">
        <div className="mono connection-pill">{service.env_key}</div>
        <button type="button" className="ghost-button" onClick={() => navigator.clipboard.writeText(service.env_val)}>
          Copy URL
        </button>
      </div>
    </div>
  );
}

function OverviewTab({ project, contexts, selectedEnv, services, onOpenDeploy, projectStats, selectedEnvStats }) {
  const selectedPreviewURL = selectedEnv ? computePreviewURL(selectedEnv, selectedEnv.latestDeploy) : "";
  const newTargetURL = targetInspectURL(selectedPreviewURL, "new");
  const oldTargetURL = targetInspectURL(selectedPreviewURL, "old");

  return (
    <section className="grid-two">
      <div className="panel section-card">
        <div className="section-card__header">
          <div>
            <div className="eyebrow">Environment Surface</div>
            <h3>Live contexts</h3>
          </div>
        </div>
        <div className="stack-list">
          {contexts.map((context) => (
            <div key={deployKey(context.app, context.env, context.branch)} className="row-card">
              <div>
                <div className="row-card__title">
                  {context.env} / {context.branch}
                </div>
                <div className="row-card__meta">
                  {context.previewURL ? (
                    <a className="link-teal" href={context.previewURL} target="_blank" rel="noreferrer">
                      {context.previewURL}
                    </a>
                  ) : (
                    "No preview URL"
                  )}
                </div>
              <div className="inline-pills">
                  <span className="metric-pill">{rolloutStrategy(context)}</span>
                  <span className="metric-pill">{trafficModeLabel(context.traffic_mode)}</span>
                  <span className="metric-pill">{liveTargetLabel(context)}</span>
                  {context.standby_slot && <span className="metric-pill">{oldTargetLabel(context)}</span>}
                </div>
              </div>
              <div className="row-card__actions">
                <span className={cx("status-chip", deployStatusClass(context.latestDeploy?.status))}>
                  <span className="status-dot" />
                  {context.latestDeploy?.status || "idle"}
                </span>
                {!context.stopped && !context.running && context.latestDeploy?.status === "success" && (
                  <span className="status-chip">container offline</span>
                )}
                {context.latestDeploy && (
                  <button type="button" className="ghost-button" onClick={() => onOpenDeploy(context.latestDeploy)}>
                    Logs
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      {selectedEnv && (
        <div className="panel section-card">
          <div className="section-card__header">
            <div>
              <div className="eyebrow">Rollout Flow</div>
              <h3>{rolloutStrategy(selectedEnv)}</h3>
            </div>
            <div className="inline-pills">
              <span className="status-chip status-chip--ok">{liveTargetLabel(selectedEnv)}</span>
              <span className="status-chip status-chip--warn">{oldTargetLabel(selectedEnv)}</span>
            </div>
          </div>
          <div className="rollout-grid">
            <div className="rollout-card rollout-card--accent">
              <div className="eyebrow">New Target</div>
              <div className="rollout-card__value">{hasSlotRollout(selectedEnv) ? "live" : "single target"}</div>
              <div className="rollout-card__meta">
                {hasSlotRollout(selectedEnv)
                  ? `All new requests are routed here. ${internalSlotLabel(selectedEnv.active_slot)}`
                  : "This environment is still serving a single target without slot handoff."}
              </div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">Old Target</div>
              <div className="rollout-card__value">{oldTargetLabel(selectedEnv)}</div>
              <div className="rollout-card__meta">
                {selectedEnv.standby_slot
                  ? `Pinned browsers can still land here until drain completes. ${internalSlotLabel(selectedEnv.standby_slot)}`
                  : hasSlotRollout(selectedEnv)
                    ? "The previous target is already gone. Relay will warm the alternate target on the next deploy."
                    : "No previous target is being kept alive in this environment."}
              </div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">Traffic Policy</div>
              <div className="rollout-card__value">{trafficModeLabel(selectedEnv.traffic_mode)}</div>
              <div className="rollout-card__meta">
                {selectedEnv.traffic_mode === "session"
                  ? "Relay pins this browser to a target by cookie. Closing the tab or hard-refreshing does not switch it."
                  : "There is no sticky cookie. Once readiness passes, every request moves to the live target."}
              </div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">New Request Flow</div>
              <div className="rollout-card__value">new target only</div>
              <div className="rollout-card__meta">
                {selectedEnv.standby_slot
                  ? selectedEnv.traffic_mode === "session"
                    ? "Only browsers already pinned to the old target stay there during drain. Every new session goes to the new target."
                    : "The old target may still be alive briefly, but new traffic no longer goes there."
                  : hasSlotRollout(selectedEnv)
                    ? "The rollout has already converged on the new target."
                    : "Single-target environments do not split traffic between generations."}
              </div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">Environment Success</div>
              <div className="rollout-card__value">{formatPercent(selectedEnvStats.successRate)}</div>
              <div className="rollout-card__meta">{selectedEnvStats.total} tracked deploys on this lane.</div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">Average Deploy</div>
              <div className="rollout-card__value">{selectedEnvStats.avgDuration ? prettyDuration(selectedEnvStats.avgDuration) : "n/a"}</div>
              <div className="rollout-card__meta">Measured from build start to ready state.</div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">Readiness Gate</div>
              <div className="rollout-card__value">60s</div>
              <div className="rollout-card__meta">Current default candidate warmup timeout.</div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">Old Target Window</div>
              <div className="rollout-card__value">{drainWindowLabel(selectedEnv)}</div>
              <div className="rollout-card__meta">
                {selectedEnv.standby_slot
                  ? "Relay removes the old target when this drain window expires."
                  : hasSlotRollout(selectedEnv)
                    ? "No old target is currently kept alive."
                    : "Single-target environments do not run a separate drain window."}
              </div>
            </div>
            <div className="rollout-card">
              <div className="eyebrow">Traffic Visibility</div>
              <div className="rollout-card__value">not tracked</div>
              <div className="rollout-card__meta">Relay does not yet count how many users or requests remain on the old target versus the new one.</div>
            </div>
          </div>
          {selectedPreviewURL && hasSlotRollout(selectedEnv) && (
            <div className="rollout-note rollout-note--warn">
              <strong>Inspecting new versus old</strong>
              <span>
                {selectedEnv.traffic_mode === "session"
                  ? "Session mode keeps a sticky cookie, so use these links when you need to pin a browser to the live target or the draining target on purpose."
                  : "Use these links when you need to inspect the live target or the draining target directly from a browser."}
              </span>
              <div className="button-row button-row--compact">
                <a className="ghost-button ghost-button--compact" href={newTargetURL} target="_blank" rel="noreferrer">
                  Pin browser to new
                </a>
                {selectedEnv.standby_slot && (
                  <a className="ghost-button ghost-button--compact" href={oldTargetURL} target="_blank" rel="noreferrer">
                    Pin browser to old
                  </a>
                )}
              </div>
            </div>
          )}
          <div className="rollout-note">
            <strong>Weighted canary</strong>
            <span>Weighted traffic splitting is not enabled in this build. Relay does full cutover after readiness, then optionally keeps the old target alive only for sticky-session drain.</span>
          </div>
        </div>
      )}

      <div className="panel section-card">
        <div className="section-card__header">
          <div>
            <div className="eyebrow">Project Telemetry</div>
            <h3>Deploy pulse</h3>
          </div>
        </div>
        <div className="rollout-grid">
          <div className="rollout-card">
            <div className="eyebrow">Project Success</div>
            <div className="rollout-card__value">{formatPercent(projectStats.successRate)}</div>
            <div className="rollout-card__meta">{projectStats.total} deploys tracked across this project.</div>
          </div>
          <div className="rollout-card">
            <div className="eyebrow">Average Deploy</div>
            <div className="rollout-card__value">{projectStats.avgDuration ? prettyDuration(projectStats.avgDuration) : "n/a"}</div>
            <div className="rollout-card__meta">Average rollout time across recorded deploys.</div>
          </div>
          <div className="rollout-card">
            <div className="eyebrow">New/Old Lanes</div>
            <div className="rollout-card__value">{projectStats.blueGreenContexts}</div>
            <div className="rollout-card__meta">Contexts already using the new/old slot handoff flow.</div>
          </div>
          <div className="rollout-card">
            <div className="eyebrow">Failure Count</div>
            <div className="rollout-card__value">{projectStats.failures}</div>
            <div className="rollout-card__meta">Failed or errored deploys in the tracked history.</div>
          </div>
        </div>
      </div>

      <div className="panel section-card">
        <div className="section-card__header">
          <div>
            <div className="eyebrow">relay.json Stack</div>
            <h3>Companion services</h3>
          </div>
        </div>
        {services.length ? (
          <div className="stack-list">
            {services.map((service) => (
              <ServiceCard key={`${service.env}-${service.branch}-${service.name}`} service={service} />
            ))}
          </div>
        ) : (
          <div className="code-callout">
            <p>No companion containers are attached to this environment yet.</p>
            <pre>{`{
  "project": "${project.name}",
  "services": [
    { "name": "db", "type": "postgres", "version": "16" },
    { "name": "cache", "type": "redis", "version": "7" },
    { "name": "web", "type": "app" }
  ]
}`}</pre>
          </div>
        )}
      </div>

      {selectedEnv && (
        <div className="panel section-card section-card--wide">
          <div className="section-card__header">
            <div>
              <div className="eyebrow">Selected Environment</div>
              <h3>{selectedEnv.env} / {selectedEnv.branch}</h3>
            </div>
            {selectedEnv.latestDeploy && (
              <div className="inline-pills">
                <span className={cx("operation-chip", operationClass(selectedEnv.latestDeploy.source))}>{operationLabel(selectedEnv.latestDeploy.source)}</span>
                <span className={cx("status-chip", deployStatusClass(selectedEnv.latestDeploy.status))}>
                  <span className="status-dot" />
                  {deployPhaseText(selectedEnv.latestDeploy)}
                </span>
              </div>
            )}
          </div>
          {selectedEnv.latestDeploy?.source === "rollback" && (
            <div className={cx("rollout-note", ACTIVE_STATUSES.has(selectedEnv.latestDeploy.status) ? "rollout-note--warn" : selectedEnv.latestDeploy.status === "success" ? "rollout-note--ok" : "rollout-note--danger")}>
              <strong>{ACTIVE_STATUSES.has(selectedEnv.latestDeploy.status) ? "Rollback running" : selectedEnv.latestDeploy.status === "success" ? "Rollback completed" : "Rollback needs attention"}</strong>
              <span>
                {ACTIVE_STATUSES.has(selectedEnv.latestDeploy.status)
                  ? `Relay is switching traffic back from ${shortImageTag(selectedEnv.latestDeploy.prev_image_tag || selectedEnv.latestDeploy.prevImage)} to ${shortImageTag(selectedEnv.latestDeploy.image_tag || selectedEnv.latestDeploy.imageTag)}.`
                  : `Latest rollback image ${shortImageTag(selectedEnv.latestDeploy.image_tag || selectedEnv.latestDeploy.imageTag)} with previous live image ${shortImageTag(selectedEnv.latestDeploy.prev_image_tag || selectedEnv.latestDeploy.prevImage)}.`}
              </span>
            </div>
          )}
          <div className="detail-grid">
            <Detail label="Engine" value={engineLabel(selectedEnv.engine || "docker")} />
            <Detail label="Mode" value={selectedEnv.mode || "port"} />
            <Detail label="Rollout Flow" value={rolloutStrategy(selectedEnv)} />
            <Detail label="Traffic Policy" value={trafficModeLabel(selectedEnv.traffic_mode)} />
            <Detail label="New Target" value={hasSlotRollout(selectedEnv) ? `live (${selectedEnv.active_slot})` : "single target"} />
            <Detail label="Old Target" value={selectedEnv.standby_slot ? `draining (${selectedEnv.standby_slot})` : hasSlotRollout(selectedEnv) ? "removed" : "not staged"} />
            <Detail label="Host Port" value={selectedEnv.host_port || "n/a"} />
            <Detail label="Service Port" value={selectedEnv.service_port || "n/a"} />
            <Detail label="Live Route" value={selectedEnv.previewURL || "No public URL"} link={selectedEnv.previewURL} />
            <Detail label="Saved Route" value={computeConfiguredURL(selectedEnv) || "No saved route"} link={computeConfiguredURL(selectedEnv)} />
            <Detail label="Repo URL" value={selectedEnv.repo_url || "Not linked"} />
            <Detail
              label="Latest Deploy Duration"
              value={
                selectedEnv.latestDeploy?.started_at && selectedEnv.latestDeploy?.ended_at
                  ? prettyDuration(new Date(selectedEnv.latestDeploy.ended_at) - new Date(selectedEnv.latestDeploy.started_at))
                  : "Pending"
              }
            />
          </div>
        </div>
      )}
    </section>
  );
}

function DeploymentsTab({ project, deploys, envMap, selectedEnv, onOpenDeploy }) {
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [laneFilter, setLaneFilter] = useState("all");

  const repoInfo = useMemo(() => repoProviderInfo(projectRepoURL(project)), [project]);

  const statusCounts = useMemo(
    () => ({
      all: deploys.length,
      active: deploys.filter((deploy) => ACTIVE_STATUSES.has(deploy.status)).length,
      success: deploys.filter((deploy) => deploy.status === "success").length,
      issues: deploys.filter((deploy) => deploy.status === "failed" || deploy.status === "error").length,
    }),
    [deploys],
  );

  const laneButtons = useMemo(() => {
    const buttons = [{ id: "all", label: "All lanes" }];
    if (selectedEnv) buttons.push({ id: "current", label: `${selectedEnv.env} / ${selectedEnv.branch}` });
    Array.from(new Set(deploys.map((deploy) => deploy.env))).slice(0, 3).forEach((env) => {
      buttons.push({ id: env, label: env });
    });
    return buttons.filter((item, index, items) => items.findIndex((candidate) => candidate.id === item.id) === index);
  }, [deploys, selectedEnv]);

  const filteredDeploys = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    return deploys.filter((deploy) => {
      const matchesQuery = !normalizedQuery || [
        deploy.id,
        deploy.app,
        deploy.env,
        deploy.branch,
        deploy.commit_sha,
        deploy.image_tag,
        deploy.previous_image_tag,
      ].filter(Boolean).some((value) => String(value).toLowerCase().includes(normalizedQuery));

      const matchesStatus =
        statusFilter === "all"
          || (statusFilter === "active" && ACTIVE_STATUSES.has(deploy.status))
          || (statusFilter === "success" && deploy.status === "success")
          || (statusFilter === "issues" && (deploy.status === "failed" || deploy.status === "error"));

      const matchesLane =
        laneFilter === "all"
          || (laneFilter === "current" && selectedEnv && deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch)
          || deploy.env === laneFilter;

      return matchesQuery && matchesStatus && matchesLane;
    });
  }, [deploys, laneFilter, query, selectedEnv, statusFilter]);

  const latestDeploy = filteredDeploys[0] || deploys[0] || null;
  const durations = filteredDeploys.map(deployDurationMs).filter(Boolean);
  const avgDuration = durations.length ? Math.round(durations.reduce((sum, value) => sum + value, 0) / durations.length) : 0;
  const successRate = filteredDeploys.length
    ? (filteredDeploys.filter((deploy) => deploy.status === "success").length / Math.max(1, filteredDeploys.length)) * 100
    : 0;

  return (
    <section className="deployments-stage">
      <div className="panel section-card deployments-stage__hero">
        <div>
          <div className="eyebrow">Deployment Board</div>
          <h3>Release activity</h3>
          <p className="muted">
            Scan recent builds, compare image flow, and jump into a rollout without leaving the project command deck.
          </p>
        </div>
        <div className="inline-pills">
          <SourceBadge info={repoInfo} />
          <span className="metric-pill">{deploys.length} deploys tracked</span>
          {selectedEnv && <span className="metric-pill">{selectedEnv.env} / {selectedEnv.branch}</span>}
        </div>
      </div>

      <div className="deployments-toolbar panel">
        <input
          className="text-input deployments-toolbar__search"
          type="search"
          placeholder="Search deploy id, branch, commit, or image"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <div className="segment-list">
          {[
            ["all", "All"],
            ["active", "Active"],
            ["success", "Ready"],
            ["issues", "Issues"],
          ].map(([id, label]) => (
            <button
              key={id}
              type="button"
              className={cx("segment-chip", statusFilter === id && "segment-chip--active")}
              onClick={() => setStatusFilter(id)}
            >
              {label}
              <span>{statusCounts[id]}</span>
            </button>
          ))}
        </div>
        <div className="segment-list">
          {laneButtons.map((lane) => (
            <button
              key={lane.id}
              type="button"
              className={cx("segment-chip", laneFilter === lane.id && "segment-chip--active")}
              onClick={() => setLaneFilter(lane.id)}
            >
              {lane.label}
            </button>
          ))}
        </div>
      </div>

      <div className="deployments-summary-grid">
        <div className="panel ops-stat-card">
          <div className="eyebrow">Latest Build</div>
          <div className="ops-stat-card__value">
            {latestDeploy ? (latestDeploy.build_number ? `#${latestDeploy.build_number}` : latestDeploy.id.slice(0, 8)) : "No deploys"}
          </div>
          <div className="ops-stat-card__meta">
            {latestDeploy
              ? `${deployPhaseText(latestDeploy)} · ${deployDurationLabel(latestDeploy)} · ${formatDateTime(latestDeploy.created_at)}`
              : "Trigger a deploy to populate the board."}
          </div>
        </div>
        <div className="panel ops-stat-card">
          <div className="eyebrow">Average Duration</div>
          <div className="ops-stat-card__value">{avgDuration ? prettyDuration(avgDuration) : "n/a"}</div>
          <div className="ops-stat-card__meta">Based on the currently filtered deployment set.</div>
        </div>
        <div className="panel ops-stat-card">
          <div className="eyebrow">Success Rate</div>
          <div className="ops-stat-card__value">{formatPercent(successRate)}</div>
          <div className="ops-stat-card__meta">{statusCounts.issues} deploys currently need attention.</div>
        </div>
        <div className="panel ops-stat-card">
          <div className="eyebrow">Source</div>
          <div className="ops-stat-card__value">{repoInfo.vendor}</div>
          <div className="ops-stat-card__meta">{repoInfo.host}</div>
        </div>
      </div>

      <div className="deployment-list">
        {!filteredDeploys.length && (
          <section className="panel section-card empty-state empty-state--compact">
            <div className="eyebrow">No Matches</div>
            <h2>Nothing landed in this filter set.</h2>
            <p>Adjust the search or status chips to widen the deploy history.</p>
          </section>
        )}

        {filteredDeploys.map((deploy) => {
          const context = envMap.get(deployKey(deploy.app, deploy.env, deploy.branch));
          const preview = computePreviewURL(context, deploy);
          const configured = context ? computeConfiguredURL(context) : "";
          const deployRepo = repoProviderInfo(context?.repo_url || projectRepoURL(project));
          const isCurrentLane = selectedEnv && deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch;

          return (
            <article key={deploy.id} className={cx("panel deployment-entry", isCurrentLane && "deployment-entry--active")}>
              <div className="deployment-entry__top">
                <div>
                  <div className="deployment-entry__headline">
                    <button type="button" className="deployment-entry__id" onClick={() => onOpenDeploy(deploy)}>
                      {deploy.build_number ? `#${deploy.build_number}` : deploy.id.slice(0, 8)}
                    </button>
                    <span className={cx("status-chip", deployStatusClass(deploy.status))}>
                      <span className="status-dot" />
                      {deployPhaseText(deploy)}
                    </span>
                    <span className={cx("operation-chip", operationClass(deploy.source))}>{operationLabel(deploy.source)}</span>
                    <SourceBadge info={deployRepo} compact />
                  </div>
                  <div className="deployment-entry__meta">
                    <span>{deploy.env} / {deploy.branch}</span>
                    <span>{formatDateTime(deploy.created_at)}</span>
                    <span className="mono">{formatCommitSHA(deploy.commit_sha)}</span>
                    {deploy.deployed_by && <span>by {deploy.deployed_by}</span>}
                  </div>
                  {deploy.commit_message && (
                    <div className="deployment-entry__commit-msg">{deploy.commit_message}</div>
                  )}
                </div>
                <div className="deployment-entry__actions">
                  {(preview || configured) && (
                    <a className="ghost-button ghost-button--compact" href={preview || configured} target="_blank" rel="noreferrer">
                      Visit
                    </a>
                  )}
                  <button type="button" className="ghost-button ghost-button--compact" onClick={() => onOpenDeploy(deploy)}>
                    Inspect
                  </button>
                </div>
              </div>

              <div className="deployment-entry__grid">
                <div className="deployment-kv">
                  <span>Duration</span>
                  <strong>{deployDurationLabel(deploy)}</strong>
                </div>
                <div className="deployment-kv">
                  <span>Image</span>
                  <strong className="mono">{shortImageTag(deploy.image_tag || deploy.imageTag)}</strong>
                </div>
                <div className="deployment-kv">
                  <span>Previous</span>
                  <strong className="mono">{shortImageTag(deploy.previous_image_tag || deploy.prevImage)}</strong>
                </div>
                <div className="deployment-kv">
                  <span>Route</span>
                  <strong>{context?.public_host || (context?.host_port ? `:${context.host_port}` : "Private")}</strong>
                </div>
              </div>

              <div className="deployment-entry__footer">
                <span className="deployment-entry__route">{preview || configured || "Private route"}</span>
                {isCurrentLane && <span className="metric-pill">Current lane</span>}
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}

function LogsTab({ project, selectedEnv, deploys, envMap, onOpenDeploy }) {
  const [windowFilter, setWindowFilter] = useState("30m");
  const [levelFilter, setLevelFilter] = useState("all");
  const [laneFilter, setLaneFilter] = useState("current");
  const [query, setQuery] = useState("");
  const [selectedTarget, setSelectedTarget] = useState("");

  const scopedDeploys = useMemo(() => {
    return deploys.filter((deploy) => {
      if (laneFilter === "current") {
        return selectedEnv ? deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch : true;
      }
      if (laneFilter === "all") return true;
      return deploy.env === laneFilter;
    });
  }, [deploys, laneFilter, selectedEnv]);

  const runtimeEnv = useMemo(() => {
    if (laneFilter === "current" || laneFilter === "all") return selectedEnv;
    return Array.from(envMap.values()).find((item) => item.env === laneFilter) || selectedEnv;
  }, [envMap, laneFilter, selectedEnv]);

  const runtime = useRuntimeLogs(runtimeEnv, selectedTarget, windowFilter);

  useEffect(() => {
    if (!runtime.targets.length) {
      setSelectedTarget("");
      return;
    }
    const currentTarget = runtime.targets.find((target) => target.id === selectedTarget);
    const nextTarget = currentTarget?.running
      ? selectedTarget
      : runtime.defaultTarget || runtime.targets.find((target) => target.running)?.id || "";
    if (nextTarget !== selectedTarget) {
      setSelectedTarget(nextTarget);
    }
  }, [runtime.defaultTarget, runtime.targets, selectedTarget]);

  const latestBuild = scopedDeploys[0] || deploys[0] || null;
  const latestContext = latestBuild ? envMap.get(deployKey(latestBuild.app, latestBuild.env, latestBuild.branch)) : selectedEnv;
  const latestPreview = latestBuild ? computePreviewURL(latestContext, latestBuild) : "";
  const latestRepo = repoProviderInfo(latestContext?.repo_url || projectRepoURL(project));
  const filteredRuntimeLines = useMemo(
    () => runtime.lines.filter((entry) => runtimeFilterMatches(entry, levelFilter, query)),
    [levelFilter, query, runtime.lines],
  );
  const runtimeCounts = useMemo(
    () =>
      runtime.lines.reduce(
        (acc, entry) => {
          acc.all += 1;
          if (entry.level === "warning") acc.warning += 1;
          if (entry.level === "error") acc.error += 1;
          if (entry.level === "fatal") acc.fatal += 1;
          return acc;
        },
        { all: 0, warning: 0, error: 0, fatal: 0 },
      ),
    [runtime.lines],
  );
  const runningRuntimeTargets = useMemo(() => runtime.targets.filter((target) => target.running), [runtime.targets]);
  const runtimeLaneSummary = useMemo(
    () => describeRuntimeLaneState(runtimeEnv, runtime.lane, runningRuntimeTargets.length),
    [runtime.lane, runtimeEnv, runningRuntimeTargets.length],
  );
  const showRuntimeLaneNotice = Boolean(
    runtimeEnv
      && runtime.targets.length
      && (runtime.status === "offline" || runtime.lane.appStopped || !runtime.lane.appRunning || (!runtime.loading && runningRuntimeTargets.length === 0)),
  );
  const activeRuntimeTarget = runtime.targets.find((target) => target.id === (selectedTarget || runtime.defaultTarget)) || runtime.targetMeta;

  return (
    <section className="logs-stage">
      <div className="logs-stage__layout">
        <Card className="logs-filter-rail">
          <CardHeader>
            <div className="eyebrow">Runtime Filters</div>
            <CardTitle>Log watch</CardTitle>
            <CardDescription>
              Follow the live app slot, edge proxy, or companion services for the selected lane.
            </CardDescription>
          </CardHeader>

          <CardContent className="logs-filter-rail__content">
            {showRuntimeLaneNotice && (
              <div className="logs-runtime-state logs-runtime-state--offline">
                <div className="logs-runtime-state__top">
                  <div className="logs-runtime-state__copy">
                    <strong>{runtimeLaneSummary.title}</strong>
                    <p>{runtimeLaneSummary.body}</p>
                  </div>
                  <Badge variant={runtimeLaneSummary.badgeVariant}>{runtimeLaneSummary.badgeLabel}</Badge>
                </div>
                {latestBuild && (
                  <Button type="button" variant="secondary" className="logs-runtime-state__action" onClick={() => onOpenDeploy(latestBuild)}>
                    Open latest build
                  </Button>
                )}
              </div>
            )}

            <div className="filter-cluster">
            <div className="eyebrow">Time Range</div>
            <div className="filter-button-list">
              {[
                ["30m", "Last 30 minutes"],
                ["6h", "Last 6 hours"],
                ["24h", "Last 24 hours"],
              ].map(([id, label]) => (
                <Button
                  key={id}
                  variant={windowFilter === id ? "secondary" : "ghost"}
                  className={cx("filter-button", windowFilter === id && "filter-button--active")}
                  onClick={() => setWindowFilter(id)}
                >
                  {label}
                </Button>
              ))}
            </div>
            </div>

            <div className="filter-cluster">
            <div className="eyebrow">Console Level</div>
            <div className="filter-button-list">
              {[
                ["all", "All", runtimeCounts.all],
                ["warning", "Warning", runtimeCounts.warning],
                ["error", "Error", runtimeCounts.error],
                ["fatal", "Fatal", runtimeCounts.fatal],
              ].map(([id, label, count]) => (
                <Button
                  key={id}
                  variant={levelFilter === id ? "secondary" : "ghost"}
                  className={cx("filter-button filter-button--count", levelFilter === id && "filter-button--active")}
                  onClick={() => setLevelFilter(id)}
                >
                  <span>{label}</span>
                  <span>{count}</span>
                </Button>
              ))}
            </div>
            </div>

            <div className="filter-cluster">
            <div className="eyebrow">Lane Scope</div>
            <Tabs value={laneFilter} onValueChange={setLaneFilter} className="logs-filter-tabs">
              <TabsList className="logs-filter-tabs__list">
                <TabsTrigger value="current">{selectedEnv ? `${selectedEnv.env} / ${selectedEnv.branch}` : "Current lane"}</TabsTrigger>
                <TabsTrigger value="prod">Production</TabsTrigger>
                <TabsTrigger value="preview">Preview</TabsTrigger>
                <TabsTrigger value="all">All envs</TabsTrigger>
              </TabsList>
            </Tabs>
            </div>

            <div className="filter-cluster">
              <div className="eyebrow">Runtime Targets</div>
              <div className="filter-button-list">
                {!runtime.targets.length && <div className="empty-inline">No runtime targets found for this lane.</div>}
                {runtime.targets.map((target) => (
                  <Button
                    key={target.id}
                    variant={selectedTarget === target.id ? "secondary" : "ghost"}
                    className={cx("filter-button filter-button--target", selectedTarget === target.id && "filter-button--active", !target.running && "filter-button--offline")}
                    disabled={!target.running}
                    onClick={() => setSelectedTarget(target.id)}
                  >
                    <span>{target.label}</span>
                    <Badge variant={target.running ? "success" : "muted"}>{target.running ? "live" : "offline"}</Badge>
                  </Button>
                ))}
              </div>
            </div>

            <div className="logs-filter-note">
              <strong>Streaming source</strong>
              <span>
                Runtime logs tail the selected engine directly for the live app slot, edge proxy, or selected companion service.
              </span>
            </div>
          </CardContent>
        </Card>

        <div className="logs-stage__main">
            <Card className="logs-console">
              <CardHeader className="logs-console__toolbar">
                <Input value={query} onChange={(event) => setQuery(event.target.value)} type="search" placeholder="Search runtime logs..." />
                <div className="inline-pills">
                <Badge variant={runtime.status === "error" ? "danger" : runtime.status === "live" ? "success" : runtime.status === "offline" ? "warning" : "muted"}>
                  {runtime.status === "live" ? "Live stream" : runtime.status === "offline" ? "Runtime offline" : runtime.status}
                </Badge>
                <Badge variant="outline">{windowFilter}</Badge>
                <Badge variant="outline">{levelFilter}</Badge>
                {activeRuntimeTarget?.label && <Badge variant="teal">{activeRuntimeTarget.label}</Badge>}
              </div>
            </CardHeader>
            <CardContent className="logs-console__content">
            <div className="logs-console__header">
              <span>Time</span>
              <span>Level</span>
              <span>Host</span>
              <span>Lane</span>
              <span>Messages</span>
            </div>
            <ScrollArea className="logs-console__scroll">
              <div className="logs-console__stream">
                {runtime.error && runtime.status !== "offline" && <div className="error-banner">{runtime.error}</div>}
                {!filteredRuntimeLines.length ? (
                  <div className="logs-console__empty">
                    <div className="logs-console__glyph" />
                    <h3>{runtime.loading ? "Loading runtime targets" : showRuntimeLaneNotice ? runtimeLaneSummary.title : "No runtime logs in this window"}</h3>
                    <p>
                      {runtime.loading
                        ? "Relay is resolving live containers for the selected lane."
                        : showRuntimeLaneNotice
                          ? runtimeLaneSummary.body
                          : "The selected filter set is quiet right now. Use the build handoff on the right if you need the latest deployment session."}
                    </p>
                  </div>
                ) : (
                  filteredRuntimeLines.map((entry, index) => (
                    <div key={`${index}-${entry.raw.slice(0, 16)}`} className="runtime-log-row">
                      <span className="runtime-log-row__time mono">{entry.time}</span>
                      <Badge variant={runtimeLogLevelVariant(entry.level)}>{entry.level}</Badge>
                      <span className="runtime-log-row__host">{entry.host}</span>
                      <span className="runtime-log-row__request">{runtimeEnv ? `${runtimeEnv.env} / ${runtimeEnv.branch}` : "runtime"}</span>
                      <div className="runtime-log-row__message mono">{entry.message}</div>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>
            </CardContent>
          </Card>

          <div className="logs-sidebar">
            <Card className="logs-sidebar-card">
              <CardHeader>
                <div className="eyebrow">Build Log Handoff</div>
                <CardTitle>{latestBuild ? latestBuild.id.slice(0, 8) : "No build selected"}</CardTitle>
                <CardDescription>
                Keep runtime monitoring on the left, then jump directly into the most recent build log when you need image flow, duration, or rollout detail.
                </CardDescription>
              </CardHeader>
              <CardContent>
              {latestBuild && (
                <>
                  <div className="logs-sidebar-card__grid">
                    <div className="deployment-kv">
                      <span>Status</span>
                      <strong>{deployPhaseText(latestBuild)}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Duration</span>
                      <strong>{deployDurationLabel(latestBuild)}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Image</span>
                      <strong className="mono">{shortImageTag(latestBuild.image_tag || latestBuild.imageTag)}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Route</span>
                      <strong>{latestContext?.public_host || (latestContext?.host_port ? `:${latestContext.host_port}` : "Private")}</strong>
                    </div>
                  </div>
                  <div className="inline-pills">
                    <SourceBadge info={latestRepo} compact />
                    {latestPreview && <Badge variant="outline">{latestPreview}</Badge>}
                  </div>
                  <Button type="button" variant="outline" className="logs-sidebar-card__action" onClick={() => onOpenDeploy(latestBuild)}>
                    Open build detail
                  </Button>
                </>
              )}
              </CardContent>
            </Card>

            <Card className="logs-sidebar-card">
              <CardHeader>
                <div className="eyebrow">Recent Build Sessions</div>
                <CardTitle>Inspect latest rollouts</CardTitle>
              </CardHeader>
              <CardContent>
              <div className="logs-build-list">
                {(scopedDeploys.length ? scopedDeploys : deploys).slice(0, 6).map((deploy) => (
                  <button key={deploy.id} type="button" className="logs-build-item" onClick={() => onOpenDeploy(deploy)}>
                    <div>
                      <div className="logs-build-item__headline">
                        <strong>{deploy.id.slice(0, 8)}</strong>
                        <span className={cx("status-chip", deployStatusClass(deploy.status))}>
                          <span className="status-dot" />
                          {deploy.status}
                        </span>
                      </div>
                      <div className="logs-build-item__meta">
                        {deploy.env} / {deploy.branch} · {deployDurationLabel(deploy)}
                      </div>
                    </div>
                    <span className="logs-build-item__time">{timeAgo(deploy.created_at)} ago</span>
                  </button>
                ))}
              </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </section>
  );
}

function SettingsTab({ project, selectedEnv, services, onUpdated }) {
  const [config, setConfig] = useState(() => buildSettingsConfig(selectedEnv));
  const [secrets, setSecrets] = useState([]);
  const [draftSecret, setDraftSecret] = useState({ key: "", value: "" });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState(null);
  const [pendingApplyHint, setPendingApplyHint] = useState(false);
  const [companions, setCompanions] = useState([]);
  const [companionBusy, setCompanionBusy] = useState(false);
  const [companionDraft, setCompanionDraft] = useState(() => defaultCompanionDraft("postgres"));
  const [companionEnvText, setCompanionEnvText] = useState("");
  const [companionVolumesText, setCompanionVolumesText] = useState("");
  const [selectedCompanion, setSelectedCompanion] = useState("");
  const [showAdvancedPorts, setShowAdvancedPorts] = useState(() => {
    const servicePort = Number(selectedEnv.service_port) || 0;
    return servicePort > 0 && servicePort !== 3000;
  });
  const [deleteProjectText, setDeleteProjectText] = useState("");
  const [deleteProjectBusy, setDeleteProjectBusy] = useState(false);
  const [deleteProjectError, setDeleteProjectError] = useState("");
  const webhookURL = `${window.location.origin}/api/webhooks/github`;
  const updateConfig = useCallback((patch) => {
    setConfig((current) => applyEngineConstraints({ ...current, ...patch }));
  }, []);
  const draftConfig = applyEngineConstraints({
    ...selectedEnv,
    ...config,
    host_port: Number(config.host_port) || 0,
    service_port: Number(config.service_port) || 0,
  });
  const savedRoute = computeConfiguredURL(selectedEnv);
  const draftRoute = computeConfiguredURL(draftConfig);
  const liveRoute = selectedEnv.latestDeploy?.preview_url || savedRoute;
  const restartRequired = Boolean(savedRoute && liveRoute && savedRoute !== liveRoute);
  const applyRequired = pendingApplyHint || restartRequired;
  const savedEngine = normalizeEngineValue(selectedEnv.engine);
  const draftEngine = normalizeEngineValue(config.engine);
  const draftIsStation = draftEngine === "station";
  const savedHostPort = Number(selectedEnv.host_port) || 0;
  const draftHostPort = Number(config.host_port) || 0;
  const savedServicePort = Number(selectedEnv.service_port) || 0;
  const draftServicePort = Number(config.service_port) || 0;
  const liveMode = selectedEnv.mode || "port";
  const savedMode = selectedEnv.mode || "port";
  const draftMode = config.mode || "port";
  const liveTrafficMode = selectedEnv.traffic_mode || "edge";
  const savedTrafficMode = selectedEnv.traffic_mode || "edge";
  const draftTrafficMode = config.traffic_mode || "edge";
  const configDirty =
    savedEngine !== draftEngine ||
    (selectedEnv.mode || "port") !== (config.mode || "port") ||
    (selectedEnv.traffic_mode || "edge") !== (config.traffic_mode || "edge") ||
    (selectedEnv.public_host || "") !== (config.public_host || "") ||
    savedHostPort !== draftHostPort ||
    savedServicePort !== draftServicePort;
  const applyState = configDirty ? "draft" : applyRequired ? "restart" : "live";
  const applySummary =
    applyState === "draft"
      ? {
          title: "Draft changes are not saved yet",
          detail: "Save and restart to apply the new route or traffic policy to the running app.",
        }
      : applyState === "restart"
        ? {
            title: "Saved config is waiting for restart",
            detail: "The next deploy already uses the saved settings. Restart only if you want the current container to switch now.",
          }
        : {
          title: "Live app matches saved config",
          detail: "Changes you save later will not affect the running app until restart or the next deploy.",
        };
  const primaryActionLabel = configDirty ? "Apply Now" : restartRequired ? "Restart to Apply" : "Restart App";
  const engineOptions = [
    {
      value: "docker",
      title: "Docker",
      summary: "Full Relay feature set: companion services, host routing, and session-sticky rollouts.",
      hint: "Best when you need the most mature runtime behavior.",
    },
    {
      value: "station",
      title: "Station",
      summary: "Native snapshot runner. Faster start path than Docker — no image pull on each deploy.",
      hint: "Supports session cutover, companion services, and host-based routing alongside Docker.",
    },
  ];
  const policyOptions = [
    {
      value: "edge",
      title: "Edge cutover",
      summary: "New requests switch to the new target as soon as deploy health checks pass.",
      hint: "Best for stateless apps and APIs with backward-compatible changes.",
    },
    {
      value: "session",
      title: "Session sticky",
      summary: "Browsers stay pinned to one target by cookie until the old target drains.",
      hint: "Safer when old pages must keep talking to old API behavior during rollout.",
    },
  ];
  const modeOptions = [
    {
      value: "port",
      title: "Port routing",
      summary: "Expose the app on a direct host port like 2000.",
    },
    {
      value: "traefik",
      title: "Host routing",
      summary: "Route by hostname through Traefik instead of a public port.",
    },
  ];
  const pendingChanges = [];
  if (savedEngine !== draftEngine) pendingChanges.push(`engine: ${engineLabel(savedEngine)} -> ${engineLabel(draftEngine)}`);
  if (savedMode !== draftMode) pendingChanges.push(`mode: ${savedMode} -> ${draftMode}`);
  if (savedTrafficMode !== draftTrafficMode) pendingChanges.push(`traffic: ${trafficModeLabel(savedTrafficMode)} -> ${trafficModeLabel(draftTrafficMode)}`);
  if ((selectedEnv.public_host || "") !== (config.public_host || "")) pendingChanges.push(`host: ${selectedEnv.public_host || "none"} -> ${config.public_host || "none"}`);
  if (savedHostPort !== draftHostPort) pendingChanges.push(`host port: ${savedHostPort || "none"} -> ${draftHostPort || "none"}`);
  if (savedServicePort !== draftServicePort) pendingChanges.push(`service port: ${savedServicePort || "none"} -> ${draftServicePort || "none"}`);
  const runtimeMatrixRows = [
    { label: "Route", live: liveRoute || "No live route yet", saved: savedRoute || "No saved route", draft: draftRoute || "No draft route" },
    { label: "Engine", live: applyRequired ? "Restart to confirm" : engineLabel(savedEngine), saved: engineLabel(savedEngine), draft: engineLabel(draftEngine) },
    { label: "Mode", live: liveMode, saved: savedMode, draft: draftMode },
    { label: "Traffic", live: trafficModeLabel(liveTrafficMode), saved: trafficModeLabel(savedTrafficMode), draft: trafficModeLabel(draftTrafficMode) },
  ];
  const selectedEngineOption = engineOptions.find((option) => option.value === draftEngine) || engineOptions[0];
  const selectedModeOption = modeOptions.find((option) => option.value === config.mode) || modeOptions[0];
  const selectedPolicyOption = policyOptions.find((option) => option.value === config.traffic_mode) || policyOptions[0];
  const deleteProjectLocked = deleteProjectText.trim() !== project.name;

  useEffect(() => {
    setConfig(buildSettingsConfig(selectedEnv));
    const servicePort = Number(selectedEnv.service_port) || 0;
    setShowAdvancedPorts(servicePort > 0 && servicePort !== 3000);
    api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`)
      .then((data) => setSecrets(data || []))
      .catch(() => setSecrets([]));
    loadCompanions().catch(() => setCompanions([]));
  }, [selectedEnv]);

  useEffect(() => {
    setNotice(null);
    setPendingApplyHint(false);
  }, [selectedEnv.app, selectedEnv.env, selectedEnv.branch]);

  useEffect(() => {
    setDeleteProjectText("");
    setDeleteProjectError("");
  }, [project.name]);

  async function saveConfig() {
    setBusy(true);
    try {
      const saved = await api("/api/apps/config", {
        method: "POST",
        body: JSON.stringify({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
          ...config,
          host_port: Number(config.host_port) || 0,
          service_port: Number(config.service_port) || 0,
        }),
      });
      setConfig(buildSettingsConfig(saved));
      setNotice({
        tone: "warn",
        text: "Saved on the agent. The next deploy uses this config automatically. Restart only if you want the current container to switch now.",
      });
      setPendingApplyHint(true);
      await onUpdated();
    } finally {
      setBusy(false);
    }
  }

  async function saveAndRestart() {
    setBusy(true);
    try {
      const saved = await api("/api/apps/config", {
        method: "POST",
        body: JSON.stringify({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
          ...config,
          host_port: Number(config.host_port) || 0,
          service_port: Number(config.service_port) || 0,
        }),
      });
      setConfig(buildSettingsConfig(saved));
      await api("/api/apps/restart", {
        method: "POST",
        body: JSON.stringify({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
        }),
      });
      setNotice({
        tone: "ok",
        text: "Saved and restart sent. Relay is applying the new route and traffic policy to the running app now.",
      });
      setPendingApplyHint(false);
      await onUpdated();
    } finally {
      setBusy(false);
    }
  }

  async function control(path) {
    setBusy(true);
    try {
      await api(path, {
        method: "POST",
        body: JSON.stringify({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
        }),
      });
      setNotice({
        tone: path.endsWith("/stop") ? "warn" : "ok",
        text:
          path.endsWith("/restart")
            ? savedEngine === "station"
              ? "Restart sent. Relay is recycling the saved Station snapshot and companion set now."
              : "Restart sent. The live route should move to the saved port after the refresh finishes."
            : path.endsWith("/start")
              ? savedEngine === "station"
                ? "Start sent. Relay is bringing the latest saved Station snapshot and enabled companions online now."
                : "Start sent. Relay is bringing the latest saved image and enabled companions online now."
              : "Stop sent. The app lane is now kept offline, and future deploys will only build/update until you start it again.",
      });
      if (!path.endsWith("/stop")) setPendingApplyHint(false);
      await onUpdated();
    } finally {
      setBusy(false);
    }
  }

  async function addSecret() {
    if (!draftSecret.key || !draftSecret.value) return;
    await api("/api/apps/secrets", {
      method: "POST",
      body: JSON.stringify({
        app: selectedEnv.app,
        env: selectedEnv.env,
        branch: selectedEnv.branch,
        key: draftSecret.key,
        value: draftSecret.value,
      }),
    });
    const next = await api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`);
    setSecrets(next || []);
    setDraftSecret({ key: "", value: "" });
  }

  async function removeSecret(key) {
    await api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}&key=${encodeURIComponent(key)}`, {
      method: "DELETE",
    });
    const next = await api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`);
    setSecrets(next || []);
  }

  async function loadCompanions() {
    const data = await api(`/api/apps/companions?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`);
    setCompanions(data || []);
    if (data?.length) {
      const current = data.find((item) => item.config?.name === selectedCompanion) || data[0];
      if (current?.config) {
        setSelectedCompanion(current.config.name);
        setCompanionDraft(current.config);
        setCompanionEnvText(envToText(current.config.env));
        setCompanionVolumesText((current.config.volumes || []).join("\n"));
      }
    } else {
      const next = defaultCompanionDraft("postgres");
      setSelectedCompanion("");
      setCompanionDraft(next);
      setCompanionEnvText("");
      setCompanionVolumesText("");
    }
  }

  function hydrateCompanionDraft(config) {
    const next = {
      ...defaultCompanionDraft(config.type || "custom"),
      ...config,
      health: {
        ...defaultCompanionDraft(config.type || "custom").health,
        ...(config.health || {}),
      },
    };
    setSelectedCompanion(next.name || "");
    setCompanionDraft(next);
    setCompanionEnvText(envToText(next.env));
    setCompanionVolumesText((next.volumes || []).join("\n"));
  }

  function startCompanion(kind) {
    const next = defaultCompanionDraft(kind);
    setSelectedCompanion("");
    setCompanionDraft(next);
    setCompanionEnvText(envToText(next.env));
    setCompanionVolumesText((next.volumes || []).join("\n"));
  }

  async function saveCompanion() {
    setCompanionBusy(true);
    try {
      const payload = {
        ...companionDraft,
        env: textToEnv(companionEnvText),
        volumes: companionVolumesText.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
      };
      await api("/api/apps/companions", {
        method: "POST",
        body: JSON.stringify({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
          config: payload,
        }),
      });
      await loadCompanions();
      await onUpdated();
    } finally {
      setCompanionBusy(false);
    }
  }

  async function deleteCompanion(name) {
    setCompanionBusy(true);
    try {
      await api(`/api/apps/companions?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}&name=${encodeURIComponent(name)}`, {
        method: "DELETE",
      });
      await loadCompanions();
      await onUpdated();
    } finally {
      setCompanionBusy(false);
    }
  }

  async function restartCompanion(name) {
    setCompanionBusy(true);
    try {
      await api("/api/apps/companions/restart", {
        method: "POST",
        body: JSON.stringify({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
          name,
        }),
      });
      await loadCompanions();
      await onUpdated();
    } finally {
      setCompanionBusy(false);
    }
  }

  async function deleteProject() {
    if (deleteProjectLocked) return;
    if (!window.confirm(`Delete project "${project.name}" and all deploys, workspaces, companion data, secrets, and runtime state? This cannot be undone.`)) {
      return;
    }
    setDeleteProjectBusy(true);
    setDeleteProjectError("");
    try {
      await api("/api/projects/delete", {
        method: "POST",
        body: JSON.stringify({
          app: project.name,
        }),
      });
      setDeleteProjectText("");
      await onUpdated();
    } catch (err) {
      setDeleteProjectError(err.message);
    } finally {
      setDeleteProjectBusy(false);
    }
  }

  return (
    <section className="grid-two settings-page">
      <div className="panel section-card section-card--wide settings-runtime-card">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--teal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="2" y="3" width="20" height="14" rx="2"/>
                <path d="M8 21h8M12 17v4"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">Runtime / Routing</div>
              <h3>Server controls</h3>
            </div>
          </div>
        </div>
        <div className="settings-runtime-layout">
          <div className="settings-runtime-sidebar">
            <div className="runtime-stage runtime-stage--compact">
              <div className="runtime-stage__header">
                <div>
                  <div className="eyebrow">Apply State</div>
                  <strong>{applySummary.title}</strong>
                </div>
                <span className={cx("runtime-stage__badge", applyState === "live" ? "runtime-stage__badge--ok" : "runtime-stage__badge--warn")}>
                  {applyState === "draft" ? "draft only" : applyState === "restart" ? "restart needed" : "live now"}
                </span>
              </div>
              <p className="runtime-stage__copy">{applySummary.detail}</p>
            </div>
            <div className={cx("runtime-diff-card", pendingChanges.length && "runtime-diff-card--pending")}>
              <div className="runtime-diff-card__header">
                <div className="eyebrow">What Changes Next</div>
                <strong>{pendingChanges.length ? `${pendingChanges.length} pending change${pendingChanges.length === 1 ? "" : "s"}` : "No draft changes"}</strong>
              </div>
              {pendingChanges.length ? (
                <div className="runtime-diff-list">
                  {pendingChanges.map((item) => (
                    <span key={item} className="runtime-diff-pill">{item}</span>
                  ))}
                </div>
              ) : (
                <p className="runtime-diff-empty">The form already matches the saved config. Restart only if the running app still has not picked up the saved settings.</p>
              )}
            </div>
            {(notice || applyRequired) && (
              <div className={cx("settings-notice", notice?.tone === "ok" && "settings-notice--ok", (notice?.tone === "warn" || applyRequired) && "settings-notice--warn")}>
                <strong>{applyRequired ? "Restart required" : notice?.tone === "ok" ? "Action sent" : "Config saved"}</strong>
                <span>{applyRequired ? "Saved settings are ahead of the running app. Restart to apply the latest route or traffic policy now." : notice?.text}</span>
              </div>
            )}
          </div>
          <div className="settings-runtime-main">
            <div className="runtime-matrix-card">
              <div className="runtime-matrix">
                <div className="runtime-matrix__head runtime-matrix__head--stub">Config</div>
                <div className="runtime-matrix__head">Running</div>
                <div className="runtime-matrix__head">Saved</div>
                <div className="runtime-matrix__head">Draft</div>
                {runtimeMatrixRows.map((row) => (
                  <React.Fragment key={row.label}>
                    <div className="runtime-matrix__label">{row.label}</div>
                    <div className="runtime-matrix__cell">{row.live}</div>
                    <div className="runtime-matrix__cell">{row.saved}</div>
                    <div className={cx("runtime-matrix__cell", row.saved !== row.draft && "runtime-matrix__cell--pending")}>{row.draft}</div>
                  </React.Fragment>
                ))}
              </div>
            </div>
            <div className="runtime-editor-grid">
              <div className="runtime-control-card">
                <div className="runtime-control-card__header">
                  <div>
                    <div className="eyebrow">Runtime Engine</div>
                    <strong>{selectedEngineOption.title}</strong>
                  </div>
                  <span className="runtime-control-card__meta">{draftIsStation ? "station lane" : "docker lane"}</span>
                </div>
                <div className="runtime-segmented">
                  {engineOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      className={cx("runtime-segmented__button", draftEngine === option.value && "runtime-segmented__button--active")}
                      onClick={() => updateConfig({ engine: option.value })}
                    >
                      {option.title}
                    </button>
                  ))}
                </div>
                <p className="runtime-control-card__copy">{selectedEngineOption.summary} {selectedEngineOption.hint}</p>
              </div>
              <div className="runtime-control-card">
                <div className="runtime-control-card__header">
                  <div>
                    <div className="eyebrow">Routing Mode</div>
                    <strong>{selectedModeOption.title}</strong>
                  </div>
                  <span className="runtime-control-card__meta">How users reach this app</span>
                </div>
                <div className="runtime-segmented">
                  {modeOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      className={cx("runtime-segmented__button", config.mode === option.value && "runtime-segmented__button--active")}
                      onClick={() => updateConfig({ mode: option.value })}
                    >
                      {option.title}
                    </button>
                  ))}
                </div>
                <p className="runtime-control-card__copy">{selectedModeOption.summary}</p>
              </div>
              <div className="runtime-control-card">
                <div className="runtime-control-card__header">
                  <div>
                    <div className="eyebrow">Traffic Policy</div>
                    <strong>{selectedPolicyOption.title}</strong>
                  </div>
                  <span className="runtime-control-card__meta">What happens during rollout</span>
                </div>
                <div className="runtime-segmented">
                  {policyOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      className={cx("runtime-segmented__button", config.traffic_mode === option.value && "runtime-segmented__button--active")}
                      onClick={() => updateConfig({ traffic_mode: option.value })}
                    >
                      {option.title}
                    </button>
                  ))}
                </div>
                <p className="runtime-control-card__copy">{selectedPolicyOption.summary} {selectedPolicyOption.hint}</p>
              </div>
            </div>

            <div className="field-grid runtime-form-grid">
              <label className="field">
                <span>Public Host</span>
                <input className="text-input" value={config.public_host} onChange={(e) => updateConfig({ public_host: e.target.value })} placeholder="" />
              </label>
              <label className="field">
                <span>Host Port</span>
                <input type="number" inputMode="numeric" min="0" className="text-input" value={config.host_port} onChange={(e) => updateConfig({ host_port: e.target.value })} />
              </label>
              {showAdvancedPorts && (
                <label className="field">
                  <span>Service Port</span>
                  <input type="number" inputMode="numeric" min="0" className="text-input" value={config.service_port} onChange={(e) => updateConfig({ service_port: e.target.value })} />
                </label>
              )}
            </div>
            <div className="settings-advanced settings-advanced--inline">
              <button type="button" className="ghost-button ghost-button--compact" onClick={() => setShowAdvancedPorts((value) => !value)}>
                {showAdvancedPorts ? "Hide Service Port" : "Show Service Port"}
              </button>
              <span className="settings-advanced__summary">
                {draftIsStation
                  ? <><code>Host Port</code> is where the Station edge proxy listens. <code>Public Host</code> and session policy work the same as Docker.</>
                  : <><code>Host Port</code> is public. <code>Service Port</code> stays on the app default unless you know it changed.</>}
              </span>
            </div>
            <div className="button-row button-row--split">
              <button type="button" className="primary-button" onClick={configDirty ? saveAndRestart : () => control("/api/apps/restart")} disabled={busy}>
                {busy ? "Working..." : primaryActionLabel}
              </button>
              <button type="button" className="ghost-button" onClick={saveConfig} disabled={busy || !configDirty}>
                Save for Later
              </button>
            </div>
            <div className="button-row button-row--compact">
              <button type="button" className="ghost-button" onClick={() => control("/api/apps/stop")} disabled={busy}>Stop App</button>
              <button type="button" className="ghost-button" onClick={() => control("/api/apps/start")} disabled={busy}>Start App</button>
            </div>
            <div className="settings-footnote">
              {draftIsStation
                ? <><strong>Station note:</strong> save the engine switch, then restart or deploy to move this lane onto the latest Station snapshot. Routing, session cutover, and companions follow the same saved lane settings.</>
                : <><strong>Safe default:</strong> change the route or traffic policy, then use <code>Apply Now</code>. If you want the next deploy to pick it up without touching the running app, use <code>Save for Later</code>.</>}
            </div>
          </div>
        </div>
      </div>

      <div className="panel section-card settings-meta-card">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--green">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">GitHub / Webhooks</div>
              <h3>Per-project integration</h3>
            </div>
          </div>
        </div>
        <label className="field">
          <span>Repository URL</span>
          <input className="text-input" value={config.repo_url} onChange={(e) => updateConfig({ repo_url: e.target.value })} />
        </label>
        <label className="field">
          <span>Webhook URL</span>
          <input className="text-input" value={webhookURL} readOnly />
        </label>
        <label className="field">
          <span>Webhook Secret</span>
          <input
            className="text-input"
            type="password"
            autoComplete="new-password"
            value={config.webhook_secret}
            onChange={(e) => updateConfig({ webhook_secret: e.target.value })}
          />
        </label>
        <p className="muted">
          When set, the app-specific webhook secret overrides the global <code>RELAY_GITHUB_WEBHOOK_SECRET</code>.
        </p>
        <div className="button-row">
          <button type="button" className="primary-button" onClick={saveConfig} disabled={busy}>
            {busy ? "Saving..." : "Save GitHub Settings"}
          </button>
        </div>
      </div>

      <div className="panel section-card section-card--wide">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--purple">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="12 2 2 7 12 12 22 7 12 2"/>
                <polyline points="2 17 12 22 22 17"/>
                <polyline points="2 12 12 17 22 12"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">Companion Services</div>
              <h3>Companion editor</h3>
            </div>
          </div>
          <div className="inline-pills">
            <span className="metric-pill">{engineLabel(draftEngine)}</span>
            <span className="metric-pill">{companions.length} companions</span>
            <span className="metric-pill">agent config overrides repo on deploy</span>
          </div>
        </div>

        <div className="companion-preset-row">
          {[
            { kind: "postgres", icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg> },
            { kind: "redis",    icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg> },
            { kind: "worker",   icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M9 2v2M15 2v2M9 20v2M15 20v2M20 9h2M20 15h2M2 9h2M2 15h2"/></svg> },
            { kind: "custom",   icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg> },
          ].map(({ kind, icon }) => (
            <button key={kind} type="button" className="companion-preset-btn" onClick={() => startCompanion(kind)} disabled={companionBusy}>
              {icon}
              Add {prettyCompanionType(kind)}
            </button>
          ))}
        </div>
        <div className="companion-layout">
          <div className="companion-list">
            {!companions.length && (
              <div className="companion-empty">
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.3 }}>
                  <polygon points="12 2 2 7 12 12 22 7 12 2"/>
                  <polyline points="2 17 12 22 22 17"/>
                  <polyline points="2 12 12 17 22 12"/>
                </svg>
                <span>No companions yet</span>
                <span>Use a preset above to add your first service.</span>
              </div>
            )}
            {companions.map((item) => (
              <button
                key={item.config.name}
                type="button"
                className={cx("companion-list__item", selectedCompanion === item.config.name && "companion-list__item--active")}
                onClick={() => hydrateCompanionDraft(item.config)}
              >
                <div className="companion-list__title">
                  <strong>{item.config.name}</strong>
                  <span>{prettyCompanionType(item.config.type)}</span>
                </div>
                <div className="companion-list__meta">
                  <span>{item.config.image || `${item.config.type}${item.config.version ? `:${item.config.version}` : ""}`}</span>
                  <span>{item.source || "agent"}</span>
                </div>
                <div className="inline-pills inline-pills--compact">
                  {item.config.stopped ? <span className="status-chip">kept off</span> : item.running ? <span className="status-chip status-chip--ok">running</span> : <span className="status-chip">not running</span>}
                  {item.config.port ? <span className="metric-pill">port {item.config.port}</span> : null}
                  {item.config.host_port ? <span className="metric-pill">host {item.config.host_port}</span> : null}
                </div>
              </button>
            ))}
          </div>
          <div className="companion-editor">
            <div className="companion-form-section">
              <div className="companion-form-section__label">Identity</div>
              <div className="companion-editor__grid">
                <label className="field">
                  <span>Name</span>
                  <input className="text-input" value={companionDraft.name} onChange={(e) => setCompanionDraft({ ...companionDraft, name: e.target.value })} />
                </label>
                <label className="field">
                  <span>Type</span>
                  <select className="text-input" value={companionDraft.type} onChange={(e) => setCompanionDraft({ ...companionDraft, type: e.target.value })}>
                    <option value="postgres">postgres</option>
                    <option value="redis">redis</option>
                    <option value="worker">worker</option>
                    <option value="custom">custom</option>
                    <option value="mysql">mysql</option>
                    <option value="mongo">mongo</option>
                  </select>
                </label>
                <label className="field">
                  <span>Version</span>
                  <input className="text-input" value={companionDraft.version || ""} onChange={(e) => setCompanionDraft({ ...companionDraft, version: e.target.value })} />
                </label>
                <label className="field field--two-col">
                  <span>Image</span>
                  <input className="text-input" value={companionDraft.image || ""} onChange={(e) => setCompanionDraft({ ...companionDraft, image: e.target.value })} />
                </label>
                <label className="field field--full">
                  <span>Command</span>
                  <input className="text-input" value={companionDraft.command || ""} onChange={(e) => setCompanionDraft({ ...companionDraft, command: e.target.value })} />
                </label>
              </div>
            </div>
            <div className="companion-form-section">
              <div className="companion-form-section__label">Runtime Policy</div>
              <div className="companion-editor__grid">
                <label className="field field--full">
                  <span>Desired State</span>
                  <select className="text-input" value={companionDraft.stopped ? "stopped" : "running"} onChange={(e) => setCompanionDraft({ ...companionDraft, stopped: e.target.value === "stopped" })}>
                    <option value="running">Running with app</option>
                    <option value="stopped">Keep off</option>
                  </select>
                </label>
              </div>
            </div>
            <div className="companion-form-section">
              <div className="companion-form-section__label">Ports</div>
              <div className="companion-editor__grid">
                <label className="field">
                  <span>Container Port</span>
                  <input type="number" inputMode="numeric" min="0" className="text-input" value={companionDraft.port || 0} onChange={(e) => setCompanionDraft({ ...companionDraft, port: Number(e.target.value) || 0 })} />
                </label>
                <label className="field">
                  <span>Host Port</span>
                  <input type="number" inputMode="numeric" min="0" className="text-input" value={companionDraft.host_port || 0} onChange={(e) => setCompanionDraft({ ...companionDraft, host_port: Number(e.target.value) || 0 })} />
                </label>
              </div>
            </div>
            <div className="companion-form-section">
              <div className="companion-form-section__label">Config</div>
              <div className="companion-editor__grid">
                <label className="field field--full">
                  <span>Environment Variables</span>
                  <textarea className="text-input textarea-input" value={companionEnvText} onChange={(e) => setCompanionEnvText(e.target.value)} placeholder={"KEY=value\nOTHER=value"} />
                </label>
                <label className="field field--full">
                  <span>Volumes</span>
                  <textarea className="text-input textarea-input" value={companionVolumesText} onChange={(e) => setCompanionVolumesText(e.target.value)} placeholder={"/host/path:/container/path\nvolume_name:/data"} />
                </label>
              </div>
            </div>
            <div className="companion-form-section">
              <div className="companion-form-section__label">Health Check</div>
              <div className="companion-editor__grid">
                <label className="field field--full">
                  <span>Command</span>
                  <input className="text-input" value={companionDraft.health?.test || ""} onChange={(e) => setCompanionDraft({ ...companionDraft, health: { ...(companionDraft.health || {}), test: e.target.value } })} placeholder="redis-cli ping || exit 1" />
                </label>
                <label className="field">
                  <span>Interval (sec)</span>
                  <input type="number" inputMode="numeric" min="0" className="text-input" value={companionDraft.health?.interval_seconds || 0} onChange={(e) => setCompanionDraft({ ...companionDraft, health: { ...(companionDraft.health || {}), interval_seconds: Number(e.target.value) || 0 } })} />
                </label>
                <label className="field">
                  <span>Timeout (sec)</span>
                  <input type="number" inputMode="numeric" min="0" className="text-input" value={companionDraft.health?.timeout_seconds || 0} onChange={(e) => setCompanionDraft({ ...companionDraft, health: { ...(companionDraft.health || {}), timeout_seconds: Number(e.target.value) || 0 } })} />
                </label>
              </div>
            </div>
            <div className="button-row button-row--split">
              <button type="button" className="primary-button" onClick={saveCompanion} disabled={companionBusy}>
                {companionBusy ? "Saving..." : companionDraft.stopped ? "Save and Keep Off" : "Save Companion"}
              </button>
              {selectedCompanion ? (
                <button type="button" className="ghost-button" onClick={() => restartCompanion(selectedCompanion)} disabled={companionBusy || companionDraft.stopped}>
                  Restart This Companion
                </button>
              ) : null}
              {selectedCompanion ? (
                <button type="button" className="ghost-button" onClick={() => deleteCompanion(selectedCompanion)} disabled={companionBusy}>
                  Delete Companion
                </button>
              ) : null}
            </div>
            <div className="settings-footnote">
              Companion state persists across deploys. If a companion is marked <code>Keep off</code>, deploys and app restarts leave it off until you turn it back on. Stopping the app also stops its running companions.
            </div>
          </div>
        </div>
      </div>

      <div className="panel section-card">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--teal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">Environment Variables</div>
              <h3>Secrets for {selectedEnv.env} / {selectedEnv.branch}</h3>
            </div>
          </div>
        </div>
        <div className="field-grid">
          <label className="field">
            <span>Key</span>
            <input className="text-input" value={draftSecret.key} onChange={(e) => setDraftSecret({ ...draftSecret, key: e.target.value })} />
          </label>
          <label className="field">
            <span>Value</span>
            <input className="text-input" type="password" autoComplete="new-password" value={draftSecret.value} onChange={(e) => setDraftSecret({ ...draftSecret, value: e.target.value })} />
          </label>
        </div>
        <div className="button-row">
          <button type="button" className="primary-button" onClick={addSecret}>Add Secret</button>
        </div>
        <div className="stack-list">
          {!secrets.length && <div className="empty-inline">No environment variables configured.</div>}
          {secrets.map((secret) => (
            <div key={secret.key} className="row-card">
              <div>
                <div className="row-card__title mono">{secret.key}</div>
                <div className="row-card__meta">••••••••</div>
              </div>
              <button type="button" className="ghost-button" onClick={() => removeSecret(secret.key)}>
                Delete
              </button>
            </div>
          ))}
        </div>
      </div>

      <div className="panel section-card danger-zone">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--danger">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M3 6h18"/>
                <path d="M8 6V4h8v2"/>
                <path d="M19 6l-1 14H6L5 6"/>
                <path d="M10 11v6M14 11v6"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">Danger Zone</div>
              <h3>Delete this project everywhere</h3>
            </div>
          </div>
        </div>
        <p className="danger-zone__copy">
          This deletes every environment for <code>{project.name}</code>, plus deploy history, saved config,
          secrets, companion state, sync sessions, workspaces, logs, edge proxy configs, and stored service volumes.
        </p>
        <label className="field">
          <span>Type <code>{project.name}</code> to confirm</span>
          <input
            className="text-input"
            value={deleteProjectText}
            onChange={(e) => setDeleteProjectText(e.target.value)}
            placeholder={project.name}
          />
        </label>
        {deleteProjectError && <div className="error-banner">{deleteProjectError}</div>}
        <div className="button-row">
          <button
            type="button"
            className="ghost-button ghost-button--danger"
            onClick={deleteProject}
            disabled={deleteProjectBusy || deleteProjectLocked}
          >
            {deleteProjectBusy ? "Deleting project..." : "Delete Project and All Data"}
          </button>
        </div>
      </div>

      <div className="panel section-card">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--green">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
                <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
                <line x1="12" y1="22.08" x2="12" y2="12"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">Companion Containers</div>
              <h3>Services on the shared network</h3>
            </div>
          </div>
        </div>
        {services.length ? (
          <div className="stack-list">
            {services.map((service) => (
              <ServiceCard key={`${service.env}-${service.branch}-${service.name}`} service={service} />
            ))}
          </div>
        ) : (
          <div className="code-callout">
            <p>This environment has no companion services yet. Add a <code>relay.json</code> at the repo root.</p>
            <pre>{`{
  "project": "${project.name}",
  "services": [
    { "name": "db", "type": "postgres", "version": "16" },
    { "name": "cache", "type": "redis", "version": "7" },
    { "name": "web", "type": "app" }
  ]
}`}</pre>
          </div>
        )}
      </div>
    </section>
  );
}

function LogViewer({ deploy, envInfo, services, onClose }) {
  const [lines, setLines] = useState([]);
  const [status, setStatus] = useState("connecting");
  const logRef = useRef(null);

  const logStats = useMemo(() => buildLogStats(lines), [lines]);
  const preview = computePreviewURL(envInfo, deploy);
  const configuredURL = computeConfiguredURL(envInfo);
  const repoInfo = repoProviderInfo(envInfo?.repo_url);

  useEffect(() => {
    const controller = new AbortController();
    let cancelled = false;
    setLines([]);
    setStatus("connecting");

    async function run() {
      try {
        const res = await fetch(`/api/logs/stream/${deploy.id}`, {
          credentials: "include",
          signal: controller.signal,
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        setStatus("live");
        const reader = res.body.getReader();
        const decoder = new TextDecoder("utf-8");
        let buffer = "";

        while (!cancelled) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let idx;
          while ((idx = buffer.indexOf("\n\n")) >= 0) {
            const frame = buffer.slice(0, idx);
            buffer = buffer.slice(idx + 2);
            let eventName = "message";
            const data = [];
            frame.split("\n").forEach((line) => {
              if (line.startsWith("event: ")) eventName = line.slice(7).trim();
              if (line.startsWith("data: ")) data.push(line.slice(6));
            });
            if (!data.length) continue;
            const payload = data.join("\n");
            if (eventName === "deploy-status") {
              try {
                const parsed = JSON.parse(payload);
                setStatus(parsed.status || "complete");
              } catch {
                setStatus(payload || "complete");
              }
              continue;
            }
            setLines((prev) => [...prev, payload]);
          }
          if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
        }

        setStatus((prev) => (prev === "live" ? "complete" : prev));
      } catch (err) {
        if (!cancelled && !controller.signal.aborted) setStatus("error");
      }
    }

    run();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [deploy.id]);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="modal modal--deploy" aria-label={`Deployment ${deploy.id}`}>
        <DialogHeader className="modal__header">
          <div>
            <div className="eyebrow">{operationLabel(deploy.source)} detail</div>
            <DialogTitle>{deploy.id}</DialogTitle>
            <DialogDescription className="inline-pills">
              <Badge variant="outline" className={cx("operation-chip", operationClass(deploy.source))}>{operationLabel(deploy.source)}</Badge>
              {(deploy.image_tag || deploy.imageTag) && <Badge variant="outline" className="mono">to {shortImageTag(deploy.image_tag || deploy.imageTag)}</Badge>}
              {(deploy.previous_image_tag || deploy.prevImage) && <Badge variant="outline" className="mono">from {shortImageTag(deploy.previous_image_tag || deploy.prevImage)}</Badge>}
            </DialogDescription>
          </div>
          <div className="modal__actions">
            <Badge variant={status === "success" || status === "complete" ? "success" : status === "error" ? "danger" : "warning"}>
              {status === "live" ? "streaming" : status}
            </Badge>
            <Button type="button" variant="outline" onClick={onClose}>Close</Button>
          </div>
        </DialogHeader>

        <div className="deployment-detail-layout">
          <div className="deployment-detail-sidebar">
            <div className="deployment-detail-metrics">
              <Card className="deployment-metric-card">
                <CardContent>
                  <div className="eyebrow">Created</div>
                  <div className="deployment-metric-card__value">{formatDateTime(deploy.created_at)}</div>
                  <div className="deployment-metric-card__meta">{timeAgo(deploy.created_at)} ago</div>
                </CardContent>
              </Card>
              <Card className="deployment-metric-card">
                <CardContent>
                  <div className="eyebrow">Duration</div>
                  <div className="deployment-metric-card__value">{deployDurationLabel(deploy)}</div>
                  <div className="deployment-metric-card__meta">From build start to completion.</div>
                </CardContent>
              </Card>
              <Card className="deployment-metric-card">
                <CardContent>
                  <div className="eyebrow">Environment</div>
                  <div className="deployment-metric-card__value">{deploy.env}</div>
                  <div className="deployment-metric-card__meta">{deploy.branch}</div>
                </CardContent>
              </Card>
              <Card className="deployment-metric-card">
                <CardContent>
                  <div className="eyebrow">Services</div>
                  <div className="deployment-metric-card__value">{services.length}</div>
                  <div className="deployment-metric-card__meta">Companion containers in this lane.</div>
                </CardContent>
              </Card>
            </div>

            <Accordion type="single" collapsible defaultValue="summary" className="detail-accordion-list">
              <AccordionItem value="summary">
                <AccordionTrigger>Build Summary</AccordionTrigger>
                <AccordionContent>
                  <div className="deployment-kv-grid">
                    <div className="deployment-kv">
                      <span>Commit</span>
                      <strong className="mono">{formatCommitSHA(deploy.commit_sha)}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Source</span>
                      <strong>{repoInfo.label}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Image</span>
                      <strong className="mono">{shortImageTag(deploy.image_tag || deploy.imageTag)}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Previous</span>
                      <strong className="mono">{shortImageTag(deploy.previous_image_tag || deploy.prevImage)}</strong>
                    </div>
                  </div>
                </AccordionContent>
              </AccordionItem>

              <AccordionItem value="routing">
                <AccordionTrigger>Routing Surface</AccordionTrigger>
                <AccordionContent>
                  <div className="deployment-kv-grid">
                    <div className="deployment-kv">
                      <span>Preview</span>
                      <strong>{preview || "Private route"}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Saved URL</span>
                      <strong>{configuredURL || "No configured route"}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Engine</span>
                      <strong>{engineLabel(envInfo?.engine || "docker")}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Mode</span>
                      <strong>{envInfo?.mode || "port"}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Traffic</span>
                      <strong>{trafficModeLabel(envInfo?.traffic_mode)}</strong>
                    </div>
                  </div>
                </AccordionContent>
              </AccordionItem>

              <AccordionItem value="runtime">
                <AccordionTrigger>Runtime Notes</AccordionTrigger>
                <AccordionContent>
                  <div className="deployment-kv-grid">
                    <div className="deployment-kv">
                      <span>Host port</span>
                      <strong>{envInfo?.host_port || "n/a"}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Service port</span>
                      <strong>{envInfo?.service_port || "n/a"}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Rollout</span>
                      <strong>{rolloutStrategy(envInfo)}</strong>
                    </div>
                    <div className="deployment-kv">
                      <span>Standby</span>
                      <strong>{oldTargetLabel(envInfo)}</strong>
                    </div>
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>

          <div className="panel deployment-terminal">
            <div className="deployment-terminal__header">
              <div>
                <div className="eyebrow">Build Stream</div>
                <h3>Live output</h3>
              </div>
              <div className="inline-pills">
                <Badge variant="outline">{logStats.total} lines</Badge>
                <Badge variant="warning">{logStats.warnings} warnings</Badge>
                <Badge variant="danger">{logStats.errors} errors</Badge>
              </div>
            </div>

            <div className="deployment-terminal__summary">
              <div className="deployment-terminal__status">
                <Badge variant={status === "success" || status === "complete" ? "success" : status === "error" ? "danger" : "warning"}>
                  {status === "live" ? "streaming build output" : status}
                </Badge>
                <SourceBadge info={repoInfo} compact />
              </div>
              <div className="deployment-kv-grid">
                <div className="deployment-kv">
                  <span>Preview</span>
                  <strong>{preview || "Private route"}</strong>
                </div>
                <div className="deployment-kv">
                  <span>Image target</span>
                  <strong className="mono">{shortImageTag(deploy.image_tag || deploy.imageTag)}</strong>
                </div>
                <div className="deployment-kv">
                  <span>Warnings</span>
                  <strong>{logStats.warnings}</strong>
                </div>
                <div className="deployment-kv">
                  <span>Errors</span>
                  <strong>{logStats.errors}</strong>
                </div>
              </div>
            </div>

            <div className="log-output mono" ref={logRef}>
              {!lines.length && <div className="muted">Connecting to log stream...</div>}
              {lines.map((line, index) => (
                <div key={`${index}-${line.slice(0, 8)}`} className={cx("log-line", `log-line--${logLineTone(line)}`)}>
                  {line}
                </div>
              ))}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function EmptyState() {
  return (
    <section className="panel section-card empty-state">
      <div className="eyebrow">No Projects Yet</div>
      <h2>Deploy an app to populate the admin.</h2>
      <p>The project dropdown is now driven by `/api/projects`, so projects appear as soon as Relay has app state.</p>
    </section>
  );
}

// ── Country code → flag emoji helper ─────────────────────────────────────────
function countryFlag(code) {
  if (!code || code.length !== 2) return "";
  const cp = [...code.toUpperCase()].map(c => 0x1f1e6 + c.charCodeAt(0) - 65);
  return String.fromCodePoint(...cp);
}

// ── Analytics tab ─────────────────────────────────────────────────────────────
function AnalyticsTab({ selectedEnv }) {
  const [period, setPeriod] = useState("7d");
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    const params = new URLSearchParams({ period });
    if (selectedEnv?.app) params.set("app", selectedEnv.app);
    api(`/api/analytics?${params}`)
      .then(d => { if (!cancelled) setData(d); })
      .catch(e => { if (!cancelled) setError(e.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [period, selectedEnv?.app]);

  const totalRequests = data?.total_requests ?? 0;
  const byCountry = data?.by_country ?? [];
  const byStatus = data?.by_status ?? [];
  const byHour = data?.by_hour ?? [];
  const byHost = data?.by_host ?? [];

  const maxCountry = byCountry.reduce((m, c) => Math.max(m, c.count), 1);
  const maxHour = byHour.reduce((m, h) => Math.max(m, h.count), 1);

  const success2xx = byStatus.filter(s => s.status >= 200 && s.status < 300).reduce((sum, s) => sum + s.count, 0);
  const redirect3xx = byStatus.filter(s => s.status >= 300 && s.status < 400).reduce((sum, s) => sum + s.count, 0);
  const error4xx = byStatus.filter(s => s.status >= 400 && s.status < 500).reduce((sum, s) => sum + s.count, 0);
  const error5xx = byStatus.filter(s => s.status >= 500).reduce((sum, s) => sum + s.count, 0);
  const successRate = totalRequests > 0 ? Math.round((success2xx / totalRequests) * 100) : 0;

  // Time labels: first and last bucket
  const firstTs = byHour[0]?.ts ?? 0;
  const lastTs = byHour[byHour.length - 1]?.ts ?? 0;
  function fmtBucket(ts) {
    if (!ts) return "";
    const d = new Date(ts * 1000);
    return period === "30d"
      ? d.toLocaleDateString([], { month: "short", day: "numeric" })
      : d.toLocaleTimeString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
  }

  return (
    <section className="analytics-pane">
      <div className="analytics-header">
        <h2>Traffic Analytics</h2>
        <div className="analytics-period-pills">
          {[["24h", "24h"], ["7d", "7 days"], ["30d", "30 days"]].map(([val, label]) => (
            <button
              key={val}
              type="button"
              className={cx("analytics-period-pill", period === val && "analytics-period-pill--active")}
              onClick={() => setPeriod(val)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {loading && <div className="muted" style={{ padding: "32px 0" }}>Loading analytics…</div>}
      {error && <div className="error-banner">{error}</div>}

      {!loading && !error && totalRequests === 0 && (
        <div className="analytics-empty">
          No traffic data yet for this period.<br />
          <span className="eyebrow" style={{ marginTop: 8, display: "block" }}>
            Traffic is recorded from the Caddy access log once the global proxy is running.
          </span>
        </div>
      )}

      {!loading && !error && totalRequests > 0 && (
        <>
          <div className="metric-row">
            <MetricCard label="Requests" value={totalRequests.toLocaleString()} meta={`last ${period}`} tone="teal" />
            <MetricCard label="Countries" value={String(byCountry.length)} meta="unique" />
            <MetricCard label="Success Rate" value={`${successRate}%`} meta="2xx responses" tone="amber" />
            <MetricCard label="Errors" value={(error4xx + error5xx).toLocaleString()} meta="4xx + 5xx" accent={error4xx + error5xx > 0} />
          </div>

          <div className="grid-two">
            {/* Top countries */}
            <div className="panel section-card">
              <div className="section-card__header">
                <div className="section-card__title">Top Countries</div>
              </div>
              {byCountry.slice(0, 12).map(c => (
                <div className="analytics-bar-row" key={c.code}>
                  <div className="analytics-bar-label">
                    <span className="analytics-flag">{countryFlag(c.code)}</span>
                    {c.name}
                  </div>
                  <div className="analytics-bar-track">
                    <div className="analytics-bar-fill" style={{ width: `${(c.count / maxCountry) * 100}%` }} />
                  </div>
                  <div className="analytics-bar-count">{c.count.toLocaleString()}</div>
                </div>
              ))}
            </div>

            {/* Status codes + top hosts */}
            <div className="panel section-card">
              <div className="section-card__header">
                <div className="section-card__title">Status Breakdown</div>
              </div>
              {[
                { label: "2xx Success", count: success2xx, cls: "" },
                { label: "3xx Redirect", count: redirect3xx, cls: "" },
                { label: "4xx Client Error", count: error4xx, cls: "analytics-bar-fill--amber" },
                { label: "5xx Server Error", count: error5xx, cls: "analytics-bar-fill--danger" },
              ].map(row => (
                <div className="analytics-bar-row" key={row.label}>
                  <div className="analytics-bar-label">{row.label}</div>
                  <div className="analytics-bar-track">
                    <div className={cx("analytics-bar-fill", row.cls)} style={{ width: `${totalRequests > 0 ? (row.count / totalRequests) * 100 : 0}%` }} />
                  </div>
                  <div className="analytics-bar-count">{row.count.toLocaleString()}</div>
                </div>
              ))}

              {byHost.length > 0 && (
                <>
                  <div className="section-card__header" style={{ marginTop: 20 }}>
                    <div className="section-card__title">Top Hosts</div>
                  </div>
                  {byHost.slice(0, 5).map(h => (
                    <div className="analytics-bar-row" key={h.host}>
                      <div className="analytics-bar-label" style={{ fontFamily: "var(--mono)", fontSize: 11 }}>{h.host}</div>
                      <div className="analytics-bar-track">
                        <div className="analytics-bar-fill" style={{ width: `${(h.count / totalRequests) * 100}%` }} />
                      </div>
                      <div className="analytics-bar-count">{h.count.toLocaleString()}</div>
                    </div>
                  ))}
                </>
              )}
            </div>
          </div>

          {/* Requests over time */}
          {byHour.length > 0 && (
            <div className="panel section-card">
              <div className="section-card__header">
                <div className="section-card__title">Requests Over Time</div>
                <div className="eyebrow">{period === "30d" ? "daily" : "hourly"} buckets</div>
              </div>
              <div className="analytics-timeseries">
                {byHour.map(h => (
                  <div
                    key={h.ts}
                    className="analytics-ts-bar"
                    style={{ height: `${Math.max(4, (h.count / maxHour) * 100)}%` }}
                    title={`${fmtBucket(h.ts)}: ${h.count.toLocaleString()} requests`}
                  />
                ))}
              </div>
              <div className="analytics-ts-labels">
                <span className="analytics-ts-label">{fmtBucket(firstTs)}</span>
                <span className="analytics-ts-label">{fmtBucket(lastTs)}</span>
              </div>
            </div>
          )}
        </>
      )}
    </section>
  );
}

function ServerVersionCard() {
  const [info, setInfo] = useState(null);

  useEffect(() => {
    api("/api/version").then(setInfo).catch(() => {});
  }, []);

  if (!info) return null;

  return (
    <div className="panel section-card">
      <div className="section-card__header">
        <div className="section-card__header-group">
          <div className="section-icon section-icon--teal">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
            </svg>
          </div>
          <div>
            <div className="eyebrow">Server Info</div>
            <h3>relayd {info.version}</h3>
          </div>
        </div>
      </div>
      <div className="stack-list">
        <div className="row-card"><div><div className="row-card__title">Version</div><div className="row-card__meta mono">{info.version}</div></div></div>
        <div className="row-card"><div><div className="row-card__title">Commit</div><div className="row-card__meta mono">{info.commit}</div></div></div>
        <div className="row-card"><div><div className="row-card__title">Build Date</div><div className="row-card__meta">{info.build_date}</div></div></div>
        <div className="row-card"><div><div className="row-card__title">OS / Arch</div><div className="row-card__meta mono">{info.os}/{info.arch}</div></div></div>
      </div>
    </div>
  );
}

function UsersPanel({ currentUser }) {
  const [users, setUsers] = useState(null);
  const [form, setForm] = useState({ username: "", password: "", role: "deployer" });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState(null);

  const isOwner = currentUser?.role === "owner";

  const load = () => {
    if (!isOwner) return;
    api("/api/users").then(setUsers).catch(() => {});
  };

  useEffect(load, [isOwner]);

  if (!isOwner) return null;

  async function createUser(e) {
    e.preventDefault();
    setBusy(true);
    setNotice(null);
    try {
      await api("/api/users", { method: "POST", body: JSON.stringify(form) });
      setForm({ username: "", password: "", role: "deployer" });
      setNotice({ tone: "ok", text: "User created." });
      load();
    } catch (err) {
      setNotice({ tone: "danger", text: err.message || "Failed." });
    } finally {
      setBusy(false);
    }
  }

  async function changeRole(id, role) {
    await api(`/api/users/${id}`, { method: "PATCH", body: JSON.stringify({ role }) });
    load();
  }

  async function deleteUser(id) {
    if (!confirm("Delete this user?")) return;
    await api(`/api/users/${id}`, { method: "DELETE" });
    load();
  }

  return (
    <div className="panel section-card section-card--wide">
      <div className="section-card__header">
        <div className="section-card__header-group">
          <div className="section-icon section-icon--teal">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
          </div>
          <div>
            <div className="eyebrow">Team</div>
            <h3>User Management</h3>
          </div>
        </div>
      </div>

      {users && users.length > 0 && (
        <div className="stack-list" style={{ marginBottom: "1rem" }}>
          {users.map((u) => (
            <div key={u.id} className="row-card" style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
              <div>
                <div className="row-card__title">{u.username}</div>
                <div className="row-card__meta">{u.role}</div>
              </div>
              <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                <select
                  className="text-input"
                  style={{ padding: "0.25rem 0.5rem", fontSize: "0.75rem" }}
                  value={u.role}
                  onChange={(e) => changeRole(u.id, e.target.value)}
                >
                  <option value="owner">owner</option>
                  <option value="deployer">deployer</option>
                  <option value="viewer">viewer</option>
                </select>
                <button type="button" className="ghost-button ghost-button--compact" onClick={() => deleteUser(u.id)}>Remove</button>
              </div>
            </div>
          ))}
        </div>
      )}

      <form onSubmit={createUser} style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
        <div className="eyebrow" style={{ marginBottom: 0 }}>Add User</div>
        <label className="field">
          <span>Username</span>
          <input className="text-input" value={form.username} onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))} required />
        </label>
        <label className="field">
          <span>Password</span>
          <input className="text-input" type="password" value={form.password} onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))} required minLength={8} />
        </label>
        <label className="field">
          <span>Role</span>
          <select className="text-input" value={form.role} onChange={(e) => setForm((f) => ({ ...f, role: e.target.value }))}>
            <option value="owner">owner</option>
            <option value="deployer">deployer</option>
            <option value="viewer">viewer</option>
          </select>
        </label>
        {notice && (
          <div className={cx("settings-notice", notice.tone === "ok" && "settings-notice--ok", notice.tone === "danger" && "settings-notice--danger")}>
            {notice.text}
          </div>
        )}
        <div className="button-row">
          <button type="submit" className="primary-button" disabled={busy}>{busy ? "Creating…" : "Create User"}</button>
        </div>
      </form>
    </div>
  );
}

function AuditLogPanel() {
  const [entries, setEntries] = useState(null);
  const [busy, setBusy] = useState(false);

  function load() {
    setBusy(true);
    api("/api/audit?limit=100").then(setEntries).catch(() => {}).finally(() => setBusy(false));
  }

  useEffect(load, []);

  function fmtAuditTime(ts) {
    if (!ts) return "";
    return new Date(ts).toLocaleString();
  }

  return (
    <div className="panel section-card section-card--wide">
      <div className="section-card__header">
        <div className="section-card__header-group">
          <div className="section-icon section-icon--amber">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/>
            </svg>
          </div>
          <div>
            <div className="eyebrow">Security</div>
            <h3>Activity Log</h3>
          </div>
        </div>
        <button type="button" className="ghost-button ghost-button--compact" onClick={load} disabled={busy}>Refresh</button>
      </div>

      {!entries && <div className="muted">Loading…</div>}
      {entries && entries.length === 0 && <div className="muted">No activity recorded yet.</div>}
      {entries && entries.length > 0 && (
        <div className="stack-list">
          {entries.map((e) => (
            <div key={e.id} className="row-card">
              <div style={{ flex: 1 }}>
                <div className="row-card__title">
                  <span className="mono" style={{ marginRight: "0.5rem" }}>{e.action}</span>
                  <span>{e.target}</span>
                </div>
                <div className="row-card__meta">
                  {e.actor && <span style={{ marginRight: "0.75rem" }}>by {e.actor}</span>}
                  {e.detail && <span style={{ marginRight: "0.75rem" }}>{e.detail}</span>}
                  <span>{fmtAuditTime(e.ts)}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ServerSettingsTab({ currentUser }) {
  const [baseDomain, setBaseDomain] = useState("");
  const [dashboardHost, setDashboardHost] = useState("");
  const [draft, setDraft] = useState({ baseDomain: "", dashboardHost: "" });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState(null);

  useEffect(() => {
    api("/api/server/config")
      .then((data) => {
        setBaseDomain(data?.base_domain || "");
        setDashboardHost(data?.dashboard_host || "");
        setDraft({
          baseDomain: data?.base_domain || "",
          dashboardHost: data?.dashboard_host || "",
        });
      })
      .catch(() => {});
  }, []);

  const dirty = draft.baseDomain !== baseDomain || draft.dashboardHost !== dashboardHost;

  async function save() {
    setBusy(true);
    setNotice(null);
    try {
      const saved = await api("/api/server/config", {
        method: "POST",
        body: JSON.stringify({ base_domain: draft.baseDomain, dashboard_host: draft.dashboardHost }),
      });
      setBaseDomain(saved?.base_domain || "");
      setDashboardHost(saved?.dashboard_host || "");
      setDraft({
        baseDomain: saved?.base_domain || "",
        dashboardHost: saved?.dashboard_host || "",
      });
      setNotice({ tone: "ok", text: "Saved. Caddy will route the dashboard host back to Relay, and new deploys without an explicit public host will auto-assign a subdomain." });
    } catch (err) {
      setNotice({ tone: "danger", text: err.message || "Save failed." });
    } finally {
      setBusy(false);
    }
  }

  const exampleHost = draft.baseDomain ? `myapp-main.${draft.baseDomain}` : "myapp-main.example.com";

  return (
    <section className="grid-two settings-page">
      <ServerVersionCard />
      <div className="panel section-card section-card--wide">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--teal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                <line x1="6" y1="6" x2="6.01" y2="6"/>
                <line x1="6" y1="18" x2="6.01" y2="18"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">Global Proxy / Domain Routing</div>
              <h3>Server-level settings</h3>
            </div>
          </div>
        </div>

        <label className="field">
          <span>Base Domain</span>
          <input
            className="text-input"
            value={draft.baseDomain}
            onChange={(e) => setDraft((current) => ({ ...current, baseDomain: e.target.value }))}
            placeholder="yourdomain.com"
          />
        </label>
        <label className="field">
          <span>Dashboard Host</span>
          <input
            className="text-input"
            value={draft.dashboardHost}
            onChange={(e) => setDraft((current) => ({ ...current, dashboardHost: e.target.value }))}
            placeholder="admin.yourdomain.com"
          />
        </label>
        <p className="muted">
          When set, apps deployed without an explicit <code>public_host</code> get an auto-generated subdomain:{" "}
          <code>{exampleHost}</code>. Relay also starts a Caddy reverse proxy container that handles TLS automatically.
        </p>
        <p className="muted">
          Set <code>Dashboard Host</code> to something like <code>admin.f4ust.com</code> so Relay owns that hostname, while apps use other hosts. You can also set these via <code>RELAY_BASE_DOMAIN</code> and <code>RELAY_DASHBOARD_HOST</code>. Saved values here take precedence.
        </p>

        {notice && (
          <div className={cx("settings-notice", notice.tone === "ok" && "settings-notice--ok", notice.tone === "danger" && "settings-notice--danger")}>
            <span>{notice.text}</span>
          </div>
        )}

        <div className="button-row">
          <button type="button" className="primary-button" onClick={save} disabled={busy || !dirty}>
            {busy ? "Saving..." : "Save Global Settings"}
          </button>
        </div>
        <div className="settings-footnote">
          The global proxy container (<code>relay-global-proxy</code>) is automatically restarted whenever app domain routing changes. Caddy handles TLS certificate provisioning for any domain you point at this server.
        </div>
      </div>

      <div className="panel section-card">
        <div className="section-card__header">
          <div className="section-card__header-group">
            <div className="section-icon section-icon--amber">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
            </div>
            <div>
              <div className="eyebrow">How it Works</div>
              <h3>Domain routing overview</h3>
            </div>
          </div>
        </div>
        <div className="stack-list">
          <div className="row-card">
            <div>
              <div className="row-card__title">Auto subdomains</div>
              <div className="row-card__meta">Set Base Domain here. New deploys auto-get <code>{"{app}-{branch}.{domain}"}</code>.</div>
            </div>
          </div>
          <div className="row-card">
            <div>
              <div className="row-card__title">Dashboard host</div>
              <div className="row-card__meta">Set <code>Dashboard Host</code> to route the Relay admin itself through Caddy, for example <code>admin.f4ust.com</code>.</div>
            </div>
          </div>
          <div className="row-card">
            <div>
              <div className="row-card__title">Custom domain per app</div>
              <div className="row-card__meta">Set <code>Public Host</code> in the app's Settings tab to override the auto-assigned subdomain with any domain.</div>
            </div>
          </div>
          <div className="row-card">
            <div>
              <div className="row-card__title">Caddy TLS</div>
              <div className="row-card__meta">Relay runs a <code>caddy:alpine</code> container (<code>relay-global-proxy</code>) that terminates TLS and proxies to each app.</div>
            </div>
          </div>
          <div className="row-card">
            <div>
              <div className="row-card__title">DNS requirement</div>
              <div className="row-card__meta">Point your domain or wildcard (<code>*.yourdomain.com</code>) A record at this server's public IP.</div>
            </div>
          </div>
        </div>
      </div>

      <UsersPanel currentUser={currentUser} />
      <AuditLogPanel />
    </section>
  );
}

function App() {
  const [authState, setAuthState] = useState("checking"); // checking | setup | login | ready
  const [loginError, setLoginError] = useState("");
  const [legacyMode, setLegacyMode] = useState(false);
  const [currentUser, setCurrentUser] = useState(null); // { username, role }
  const [activeTab, setActiveTab] = useState("overview");
  const [selectedProjectName, setSelectedProjectName] = useState("");
  const [selectedEnvKey, setSelectedEnvKey] = useState("");
  const [selectedDeploy, setSelectedDeploy] = useState(null);
  const [dashboard, refreshDashboard] = useDashboardData(authState === "ready");

  useEffect(() => {
    let cancelled = false;
    api("/api/auth/session")
      .then((session) => {
        if (cancelled) return;
        if (session?.setup_required) { setAuthState("setup"); return; }
        if (session?.authenticated) {
          setLegacyMode(!!session.legacy_mode);
          if (session.username) setCurrentUser({ username: session.username, role: session.role });
          setAuthState("ready");
          return;
        }
        setLegacyMode(!!session?.legacy_mode);
        setAuthState("login");
      })
      .catch(() => {
        if (!cancelled) setAuthState("login");
      });
    return () => { cancelled = true; };
  }, []);

  const deploysByContext = useMemo(() => {
    const map = new Map();
    for (const deploy of dashboard.deploys) {
      const key = deployKey(deploy.app, deploy.env, deploy.branch);
      if (!map.has(key)) map.set(key, []);
      map.get(key).push(deploy);
    }
    return map;
  }, [dashboard.deploys]);

  const projectOptions = useMemo(() => {
    return dashboard.projects
      .slice()
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((project) => ({
        ...project,
        latestDeployAt: Math.max(
          0,
          ...project.envs.map((envInfo) => {
            const deploy = (deploysByContext.get(deployKey(project.name, envInfo.env, envInfo.branch)) || [])[0];
            return deploy ? new Date(deploy.created_at).getTime() : 0;
          }),
        ),
      }))
      .sort((a, b) => b.latestDeployAt - a.latestDeployAt);
  }, [dashboard.projects, deploysByContext]);

  useEffect(() => {
    if (!projectOptions.length) {
      setSelectedProjectName("");
      return;
    }
    if (!projectOptions.find((item) => item.name === selectedProjectName)) {
      setSelectedProjectName(projectOptions[0].name);
    }
  }, [projectOptions, selectedProjectName]);

  const selectedProject = useMemo(
    () => projectOptions.find((project) => project.name === selectedProjectName) || null,
    [projectOptions, selectedProjectName],
  );

  const envOptions = useMemo(() => {
    if (!selectedProject) return [];
    return selectedProject.envs
      .slice()
      .sort((a, b) => {
        if (a.env === b.env) return a.branch.localeCompare(b.branch);
        if (a.env === "prod") return -1;
        if (b.env === "prod") return 1;
        return a.env.localeCompare(b.env);
      })
      .map((envInfo) => {
        const latestDeploy = (deploysByContext.get(deployKey(selectedProject.name, envInfo.env, envInfo.branch)) || [])[0] || null;
        return {
          ...envInfo,
          latestDeploy,
          previewURL: computePreviewURL(envInfo, latestDeploy),
        };
      });
  }, [selectedProject, deploysByContext]);

  useEffect(() => {
    if (!envOptions.length) {
      setSelectedEnvKey("");
      return;
    }
    if (!envOptions.find((item) => deployKey(item.app, item.env, item.branch) === selectedEnvKey)) {
      setSelectedEnvKey(deployKey(envOptions[0].app, envOptions[0].env, envOptions[0].branch));
    }
  }, [envOptions, selectedEnvKey]);

  const selectedEnv = envOptions.find((item) => deployKey(item.app, item.env, item.branch) === selectedEnvKey) || null;
  const projectEnvMap = useMemo(
    () => new Map(envOptions.map((item) => [deployKey(item.app, item.env, item.branch), item])),
    [envOptions],
  );
  const projectServices = useMemo(() => {
    if (!selectedProject || !selectedEnv) return [];
    return (selectedProject.services || []).filter(
      (service) => service.env === selectedEnv.env && service.branch === selectedEnv.branch,
    );
  }, [selectedProject, selectedEnv]);

  const projectDeploys = useMemo(() => {
    if (!selectedProject) return [];
    return dashboard.deploys.filter((deploy) => deploy.app === selectedProject.name);
  }, [dashboard.deploys, selectedProject]);

  const selectedEnvDeploys = useMemo(() => {
    if (!selectedEnv) return [];
    return dashboard.deploys.filter(
      (deploy) => deploy.app === selectedEnv.app && deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch,
    );
  }, [dashboard.deploys, selectedEnv]);

  const projectStats = useMemo(() => {
    const completed = projectDeploys.filter((deploy) => deploy.status === "success" || deploy.status === "failed" || deploy.status === "error");
    const successes = completed.filter((deploy) => deploy.status === "success").length;
    const durations = projectDeploys.map(deployDurationMs).filter(Boolean);
    const avgDuration = durations.length ? Math.round(durations.reduce((sum, value) => sum + value, 0) / durations.length) : 0;
    return {
      total: projectDeploys.length,
      failures: completed.filter((deploy) => deploy.status !== "success").length,
      successRate: completed.length ? (successes / completed.length) * 100 : 0,
      avgDuration,
      blueGreenContexts: (selectedProject?.envs || []).filter((envInfo) => Boolean(envInfo.active_slot)).length,
    };
  }, [projectDeploys, selectedProject]);

  const selectedEnvStats = useMemo(() => {
    const completed = selectedEnvDeploys.filter((deploy) => deploy.status === "success" || deploy.status === "failed" || deploy.status === "error");
    const successes = completed.filter((deploy) => deploy.status === "success").length;
    const durations = selectedEnvDeploys.map(deployDurationMs).filter(Boolean);
    return {
      total: selectedEnvDeploys.length,
      successRate: completed.length ? (successes / completed.length) * 100 : 0,
      avgDuration: durations.length ? Math.round(durations.reduce((sum, value) => sum + value, 0) / durations.length) : 0,
    };
  }, [selectedEnvDeploys]);

  const projectRepoInfo = useMemo(() => repoProviderInfo(projectRepoURL(selectedProject)), [selectedProject]);
  const selectedEnvRepoInfo = useMemo(
    () => repoProviderInfo(selectedEnv?.repo_url || projectRepoURL(selectedProject)),
    [selectedEnv, selectedProject],
  );
  const latestProjectDeploy = projectDeploys[0] || null;
  const selectedDeployEnvInfo = selectedDeploy ? projectEnvMap.get(deployKey(selectedDeploy.app, selectedDeploy.env, selectedDeploy.branch)) || null : null;
  const selectedDeployServices = useMemo(() => {
    if (!selectedProject || !selectedDeploy) return [];
    return (selectedProject.services || []).filter(
      (service) => service.env === selectedDeploy.env && service.branch === selectedDeploy.branch,
    );
  }, [selectedDeploy, selectedProject]);

  useEffect(() => {
    if (selectedDeploy && selectedProject && selectedDeploy.app !== selectedProject.name) {
      setSelectedDeploy(null);
    }
  }, [selectedDeploy, selectedProject]);

  const selectedPreviewURL = selectedEnv ? computePreviewURL(selectedEnv, selectedEnv.latestDeploy) : "";
  const selectedConfiguredURL = selectedEnv ? computeConfiguredURL(selectedEnv) : "";
  const currentOperationValue = selectedEnv?.latestDeploy ? operationLabel(selectedEnv.latestDeploy.source).toUpperCase() : "IDLE";
  const currentOperationMeta = selectedEnv?.latestDeploy ? deployPhaseText(selectedEnv.latestDeploy) : "waiting for first deploy";
  const currentOperationTone =
    selectedEnv?.latestDeploy?.status === "failed" || selectedEnv?.latestDeploy?.status === "error"
      ? "danger"
      : dashboard.isLive
        ? "warn"
        : "teal";

  async function handleLogin(creds) {
    setLoginError("");
    try {
      if (legacyMode) {
        // Legacy: POST token to old session endpoint
        await api("/api/auth/session", { method: "POST", body: JSON.stringify({ token: creds }) });
      } else {
        const resp = await api("/api/auth/login", { method: "POST", body: JSON.stringify(creds) });
        if (resp?.setup_required) { setAuthState("setup"); return; }
        if (resp?.username) setCurrentUser({ username: resp.username, role: resp.role });
      }
      setAuthState("ready");
    } catch (err) {
      setLoginError(err.message);
    }
  }

  async function handleSetup(creds) {
    setLoginError("");
    try {
      const resp = await api("/api/auth/setup", { method: "POST", body: JSON.stringify(creds) });
      if (resp?.username) setCurrentUser({ username: resp.username, role: resp.role });
      setAuthState("ready");
    } catch (err) {
      setLoginError(err.message);
    }
  }

  async function handleLogout() {
    await api("/api/auth/session", { method: "DELETE" });
    setCurrentUser(null);
    setAuthState("login");
    setSelectedDeploy(null);
  }

  if (authState === "checking") {
    return <SplashScreen label="Checking dashboard session" />;
  }

  if (authState === "setup") {
    return <SetupScreen onSetup={handleSetup} error={loginError} />;
  }

  if (authState !== "ready") {
    return <LoginScreen onLogin={handleLogin} error={loginError} legacyMode={legacyMode} />;
  }

  if (dashboard.loading) {
    return <SplashScreen label="Loading projects, services, and deploy history" />;
  }

  return (
    <div className="app-shell">
      <aside className="sidebar panel">
        <div className="sidebar__masthead">
          <div className="brand">
            <RelayMark className="brand__glyph" title="Relay mark" />
            <div>
              <div className="brand__title">Relay</div>
              <div className="eyebrow">Control Room</div>
            </div>
          </div>
          <div className="status-chip status-chip--ok">
            <span className="status-dot" />
            Agent live
          </div>
        </div>

        {selectedProject && (
          <div className="sidebar-project-card">
            <div className="eyebrow">Project Surface</div>
            <div className="sidebar-project-card__title">{selectedProject.name}</div>
            <div className="sidebar-project-card__meta">
              {selectedProject.envs.length} environments · {selectedProject.services.length} services
            </div>
            <div className="inline-pills">
              <SourceBadge info={projectRepoInfo} compact />
              <span className="metric-pill">{formatPercent(projectStats.successRate)} success</span>
              <span className="metric-pill">{projectStats.blueGreenContexts} new/old</span>
            </div>
          </div>
        )}

        <div className="nav-section-label">Navigation</div>
        <nav className="nav">
          {[
            ["overview", "Overview"],
            ["deployments", "Deployments"],
            ["logs", "Logs"],
            ["analytics", "Analytics"],
            ["settings", "Settings"],
            ["server", "Server"],
          ].map(([id, label]) => (
            <button
              key={id}
              type="button"
              className={cx("nav__item", activeTab === id && "nav__item--active")}
              onClick={() => setActiveTab(id)}
            >
              {NAV_ICONS[id]}
              {label}
            </button>
          ))}
        </nav>

        <div className="sidebar__footer">
          {currentUser && (
            <div className="sidebar__footer-card">
              <div className="eyebrow">{currentUser.role}</div>
              <div className="sidebar__footer-copy">{currentUser.username}</div>
            </div>
          )}
          {!currentUser && (
            <div className="sidebar__footer-card">
              <div className="eyebrow">Operator Hint</div>
              <div className="sidebar__footer-copy">Overview for rollout state, Deployments for history, Logs for live monitoring, Settings for runtime changes.</div>
            </div>
          )}
          <button type="button" className="ghost-button" onClick={handleLogout}>
            Log out
          </button>
        </div>
      </aside>

      <main className="main">
        <header className="dashboard-hero panel">
          <div className="dashboard-hero__mast">
            <div className="dashboard-hero__identity">
              <ProjectCommandSelector
                projects={projectOptions}
                selected={selectedProjectName}
                onSelect={(name) => {
                  setSelectedProjectName(name);
                  setActiveTab("overview");
                }}
              />
              <div className="breadcrumb">
                <span>{selectedProject ? selectedProject.name : "No project"}</span>
                <span className="breadcrumb__sep">/</span>
                <span className="breadcrumb__active">{activeTab}</span>
              </div>
              <h1 className="dashboard-hero__title">{selectedProject ? selectedProject.name : "Relay Control Room"}</h1>
              <p className="dashboard-hero__copy">
                {selectedEnv
                  ? `Operating ${selectedEnv.env} / ${selectedEnv.branch} with ${trafficModeLabel(selectedEnv.traffic_mode)} and ${rolloutStrategy(selectedEnv)}. New requests already land on the live target${selectedEnv.standby_slot ? ", while pinned browsers can still drain on the old target." : "."}`
                  : "Select a project lane to inspect deploy flow, routing, and runtime state."}
              </p>
            </div>
            <div className="topbar__meta">
              {dashboard.isLive ? (
                <div className="status-chip status-chip--warn live-badge">
                  <span className="status-dot status-dot--pulse" />
                  Deploying live
                </div>
              ) : (
                <button type="button" className="ghost-button" onClick={refreshDashboard}>
                  {dashboard.refreshing ? "Refreshing..." : "Refresh state"}
                </button>
              )}
              <div className="metric-pill">{dashboard.deploys.length} deploys tracked</div>
            </div>
          </div>
          <div className="dashboard-command-grid">
            <div className="command-deck-card">
              <div className="eyebrow">Source Control</div>
              <div className="command-deck-card__title">{selectedEnvRepoInfo.label}</div>
              <div className="command-deck-card__meta">{selectedEnvRepoInfo.host}</div>
              <div className="inline-pills">
                <SourceBadge info={selectedEnvRepoInfo} compact />
                <span className="metric-pill">{selectedProject ? `${selectedProject.envs.length} environments` : "No project selected"}</span>
              </div>
            </div>
            <div className="command-deck-card">
              <div className="eyebrow">Deployment Settings</div>
              <div className="command-deck-card__title">{selectedEnv ? trafficModeLabel(selectedEnv.traffic_mode) : "Waiting for lane"}</div>
              <div className="command-deck-card__meta">
                {selectedEnv
                  ? `${selectedEnv.mode || "port"} mode · ${selectedConfiguredURL || `host port ${selectedEnv.host_port || "n/a"}`}`
                  : "Select an environment to inspect routing, host, and traffic settings."}
              </div>
              <div className="inline-pills">
                <span className="metric-pill">{selectedEnv ? rolloutStrategy(selectedEnv) : "idle"}</span>
                <span className="metric-pill">{selectedEnv ? liveTargetLabel(selectedEnv) : "no target"}</span>
              </div>
            </div>
            <div className="command-deck-card">
              <div className="eyebrow">Latest Build</div>
              <div className="command-deck-card__title">{latestProjectDeploy ? latestProjectDeploy.id.slice(0, 8) : "No deploy yet"}</div>
              <div className="command-deck-card__meta">
                {latestProjectDeploy
                  ? `${deployPhaseText(latestProjectDeploy)} · ${deployDurationLabel(latestProjectDeploy)} · ${formatDateTime(latestProjectDeploy.created_at)}`
                  : "Run a deploy to unlock image flow, duration, and build summary."}
              </div>
              <div className="inline-pills">
                {latestProjectDeploy && (
                  <span className={cx("status-chip", deployStatusClass(latestProjectDeploy.status))}>
                    <span className="status-dot" />
                    {latestProjectDeploy.status}
                  </span>
                )}
                {selectedPreviewURL && <span className="metric-pill">{selectedPreviewURL}</span>}
              </div>
              {latestProjectDeploy && (
                <button type="button" className="ghost-button ghost-button--compact command-deck-card__action" onClick={() => setSelectedDeploy(latestProjectDeploy)}>
                  Inspect build
                </button>
              )}
            </div>
          </div>
          <div className="dashboard-hero__grid">
            <div className="dashboard-hero__route">
              <div className="eyebrow">Selected Route</div>
              {selectedPreviewURL ? (
                <a className="link-teal dashboard-hero__route-link" href={selectedPreviewURL} target="_blank" rel="noreferrer">
                  {selectedPreviewURL}
                </a>
              ) : (
                <div className="dashboard-hero__route-link muted">No public route yet</div>
              )}
            </div>
            <div className="dashboard-hero__stack">
              <span className="metric-pill">{selectedEnv ? `${selectedEnv.env} / ${selectedEnv.branch}` : "no lane selected"}</span>
              <span className="metric-pill">{selectedEnv ? trafficModeLabel(selectedEnv.traffic_mode) : "waiting"}</span>
              <span className="metric-pill">{selectedEnv ? liveTargetLabel(selectedEnv) : "idle"}</span>
              {selectedEnv?.standby_slot && <span className="metric-pill">{oldTargetLabel(selectedEnv)}</span>}
            </div>
          </div>
        </header>

        {dashboard.error && <div className="error-banner">{dashboard.error}</div>}

        {!selectedProject ? (
          activeTab === "server" ? (
            <ServerSettingsTab currentUser={currentUser} />
          ) : activeTab === "analytics" ? (
            <AnalyticsTab selectedEnv={null} />
          ) : (
            <EmptyState />
          )
        ) : (
          <>
            <div className="metric-row">
              <MetricCard label="Environments" value={String(selectedProject.envs.length)} meta="active project lanes" tone="teal" />
              <MetricCard label="Services" value={String(selectedProject.services.length)} meta="companion containers" accent />
              <MetricCard label="Success Rate" value={formatPercent(projectStats.successRate)} meta={`${projectStats.total} deploys tracked`} tone="amber" />
              <MetricCard
                label="Current Operation"
                value={currentOperationValue}
                meta={currentOperationMeta}
                accent={selectedEnv?.latestDeploy?.status === "failed" || selectedEnv?.latestDeploy?.status === "error"}
                tone={currentOperationTone}
              />
            </div>

            <ContextBar contexts={envOptions} selected={selectedEnvKey} onSelect={setSelectedEnvKey} />

            {activeTab === "overview" && (
              <OverviewTab
                project={selectedProject}
                contexts={envOptions}
                selectedEnv={selectedEnv}
                services={projectServices}
                projectStats={projectStats}
                selectedEnvStats={selectedEnvStats}
                onOpenDeploy={(deploy) => {
                  setSelectedDeploy(deploy);
                }}
              />
            )}

            {activeTab === "deployments" && (
              <DeploymentsTab
                project={selectedProject}
                deploys={projectDeploys}
                envMap={projectEnvMap}
                selectedEnv={selectedEnv}
                onOpenDeploy={setSelectedDeploy}
              />
            )}

            {activeTab === "logs" && (
              <LogsTab
                project={selectedProject}
                selectedEnv={selectedEnv}
                deploys={projectDeploys}
                envMap={projectEnvMap}
                onOpenDeploy={setSelectedDeploy}
              />
            )}

            {activeTab === "settings" && selectedEnv && (
              <SettingsTab
                project={selectedProject}
                selectedEnv={selectedEnv}
                services={projectServices}
                onUpdated={refreshDashboard}
              />
            )}

            {activeTab === "server" && (
              <ServerSettingsTab currentUser={currentUser} />
            )}

            {activeTab === "analytics" && (
              <AnalyticsTab selectedEnv={selectedEnv} />
            )}
          </>
        )}

        {selectedDeploy && (
          <LogViewer
            deploy={selectedDeploy}
            envInfo={selectedDeployEnvInfo}
            services={selectedDeployServices}
            onClose={() => setSelectedDeploy(null)}
          />
        )}
      </main>
    </div>
  );
}

createRoot(document.getElementById("root")).render(<App />);
