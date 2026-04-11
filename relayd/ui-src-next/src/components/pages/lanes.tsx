"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import {
  computeConfiguredURL,
  computePreviewURL,
  deployKey,
  engineLabel,
  rolloutStrategy,
  timeAgo,
  trafficModeLabel,
  type NormalizedProject,
} from "@/lib/relay-utils";
import {
  startApp,
  stopApp,
  restartApp,
  deleteLane,
  saveAppConfig,
  type EnvInfo,
  type Deploy,
} from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
  latestDeploy?: Deploy;
  previewURL?: string;
}

interface LanesPageProps {
  project: NormalizedProject | null;
  contexts: SelectedEnvMeta[];
  isOwner: boolean;
  isDeployer: boolean;
  onUpdated: () => void;
  onNavigateSettings: (envKey: string) => void;
}

const ENV_PRESETS = ["prod", "staging", "dev", "preview", "test"];
const ENGINE_OPTIONS = ["docker", "station"];
const MODE_OPTIONS = [
  { value: "port", label: "HTTP" },
  { value: "static", label: "Static" },
];

function expiryCountdown(expiresAt: number | undefined): { label: string; urgent: boolean } | null {
  if (!expiresAt || expiresAt <= 0) return null;
  const msLeft = expiresAt * 1000 - Date.now();
  if (msLeft <= 0) return { label: "Expired", urgent: true };
  const totalMin = Math.floor(msLeft / 60000);
  if (totalMin < 60) return { label: `${totalMin}m left`, urgent: totalMin < 15 };
  const hours = Math.floor(totalMin / 60);
  const mins = totalMin % 60;
  if (hours < 24) return { label: `${hours}h ${mins}m left`, urgent: hours < 2 };
  const days = Math.floor(hours / 24);
  return { label: `${days}d left`, urgent: false };
}

function drainLabel(drainUntil: number | undefined): string | null {
  if (!drainUntil || drainUntil <= 0) return null;
  if (drainUntil * 1000 < Date.now()) return null;
  const d = new Date(drainUntil * 1000);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

interface CreateLaneModalProps {
  app: string;
  onClose: () => void;
  onCreated: () => void;
}

function CreateLaneModal({ app, onClose, onCreated }: CreateLaneModalProps) {
  const [env, setEnv] = useState("dev");
  const [customEnv, setCustomEnv] = useState("");
  const [branch, setBranch] = useState("main");
  const [engine, setEngine] = useState("docker");
  const [mode, setMode] = useState("port");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const resolvedEnv = env === "__custom__" ? customEnv.trim() : env;

  async function handleCreate() {
    if (!resolvedEnv || !branch) return;
    setBusy(true);
    setError(null);
    try {
      await saveAppConfig({ app, env: resolvedEnv, branch }, { engine, mode });
      onCreated();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create lane");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative z-10 bg-zinc-950 border border-white/[0.1] rounded-xl w-full max-w-md shadow-2xl">
        <div className="p-5 border-b border-white/[0.06]">
          <div className="eyebrow mb-0.5">Lane management</div>
          <h2 className="text-base font-semibold text-white">Create new lane</h2>
          <p className="text-xs text-white/40 mt-0.5">Adds a new environment to <span className="text-white/60">{app}</span>. A deploy must follow to activate it.</p>
        </div>
        <div className="p-5 space-y-4">
          {error && (
            <div className="text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded px-3 py-2">{error}</div>
          )}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-white/50 mb-1.5">Environment</label>
              <select
                className="w-full bg-zinc-900 border border-white/[0.1] rounded px-2.5 py-1.5 text-sm text-white outline-none focus:border-relay-accent/50 appearance-none cursor-pointer"
                value={env}
                onChange={(e) => setEnv(e.target.value)}
              >
                {ENV_PRESETS.map((e) => (
                  <option key={e} value={e} style={{ backgroundColor: "#18181b" }}>{e}</option>
                ))}
                <option value="__custom__" style={{ backgroundColor: "#18181b" }}>custom…</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-white/50 mb-1.5">Branch</label>
              <input
                className="w-full bg-zinc-900 border border-white/[0.1] rounded px-2.5 py-1.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                placeholder="main"
              />
            </div>
          </div>
          {env === "__custom__" && (
            <div>
              <label className="block text-xs text-white/50 mb-1.5">Custom env name</label>
              <input
                className="w-full bg-zinc-900 border border-white/[0.1] rounded px-2.5 py-1.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50"
                value={customEnv}
                onChange={(e) => setCustomEnv(e.target.value)}
                placeholder="e.g. qa"
              />
            </div>
          )}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-white/50 mb-1.5">Engine</label>
              <select
                className="w-full bg-zinc-900 border border-white/[0.1] rounded px-2.5 py-1.5 text-sm text-white outline-none focus:border-relay-accent/50 appearance-none cursor-pointer"
                value={engine}
                onChange={(e) => setEngine(e.target.value)}
              >
                {ENGINE_OPTIONS.map((e) => (
                  <option key={e} value={e} style={{ backgroundColor: "#18181b" }}>{engineLabel(e)}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs text-white/50 mb-1.5">Mode</label>
              <select
                className="w-full bg-zinc-900 border border-white/[0.1] rounded px-2.5 py-1.5 text-sm text-white outline-none focus:border-relay-accent/50 appearance-none cursor-pointer"
                value={mode}
                onChange={(e) => setMode(e.target.value)}
              >
                {MODE_OPTIONS.map((m) => (
                  <option key={m.value} value={m.value} style={{ backgroundColor: "#18181b" }}>{m.label}</option>
                ))}
              </select>
            </div>
          </div>
        </div>
        <div className="flex items-center justify-end gap-2 p-4 pt-0">
          <button
            type="button"
            onClick={onClose}
            className="text-sm text-white/50 hover:text-white px-3 py-1.5 rounded hover:bg-white/[0.06] transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleCreate}
            disabled={busy || !resolvedEnv || !branch}
            className="text-sm bg-relay-accent/90 hover:bg-relay-accent text-white font-semibold px-4 py-1.5 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {busy ? "Creating…" : "Create lane"}
          </button>
        </div>
      </div>
    </div>
  );
}

interface DeleteLaneDialogProps {
  target: { app: string; env: string; branch: string };
  onClose: () => void;
  onDeleted: () => void;
}

function DeleteLaneDialog({ target, onClose, onDeleted }: DeleteLaneDialogProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleDelete() {
    setBusy(true);
    setError(null);
    try {
      await deleteLane(target);
      onDeleted();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete lane");
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative z-10 bg-zinc-950 border border-red-500/20 rounded-xl w-full max-w-sm shadow-2xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5 text-red-400">Destructive action</div>
          <h2 className="text-base font-semibold text-white">Delete lane</h2>
          <p className="text-xs text-white/50 mt-1.5">
            This will permanently delete <span className="text-white">{target.env} / {target.branch}</span> and all its state. Running containers will be stopped.
          </p>
        </div>
        {error && <div className="text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded px-3 py-2">{error}</div>}
        <div className="flex items-center justify-end gap-2">
          <button type="button" onClick={onClose} className="text-sm text-white/50 hover:text-white px-3 py-1.5 rounded hover:bg-white/[0.06] transition-colors">Cancel</button>
          <button
            type="button"
            onClick={handleDelete}
            disabled={busy}
            className="text-sm bg-red-600 hover:bg-red-500 text-white font-semibold px-4 py-1.5 rounded transition-colors disabled:opacity-50"
          >
            {busy ? "Deleting…" : "Delete lane"}
          </button>
        </div>
      </div>
    </div>
  );
}

export function LanesPage({
  project,
  contexts,
  isOwner,
  isDeployer,
  onUpdated,
  onNavigateSettings,
}: LanesPageProps) {
  const [showCreate, setShowCreate] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ app: string; env: string; branch: string } | null>(null);
  const [busy, setBusy] = useState<Record<string, string>>({});
  const [notices, setNotices] = useState<Record<string, { tone: "ok" | "error"; text: string }>>({});

  if (!project) {
    return (
      <div className="flex items-center justify-center h-full text-white/30 text-sm">
        Select a project to manage lanes
      </div>
    );
  }

  function setRowBusy(key: string, action: string) {
    setBusy((b) => ({ ...b, [key]: action }));
  }
  function clearRowBusy(key: string) {
    setBusy((b) => { const n = { ...b }; delete n[key]; return n; });
  }
  function setRowNotice(key: string, tone: "ok" | "error", text: string) {
    setNotices((n) => ({ ...n, [key]: { tone, text } }));
    setTimeout(() => setNotices((n) => { const m = { ...n }; delete m[key]; return m; }), 4000);
  }

  async function handleAction(
    action: "start" | "stop" | "restart" | "extend",
    ctx: SelectedEnvMeta,
  ) {
    const key = deployKey(ctx.app, ctx.env, ctx.branch);
    setRowBusy(key, action);
    try {
      const target = { app: ctx.app, env: ctx.env, branch: ctx.branch };
      if (action === "start") await startApp(target);
      else if (action === "stop") await stopApp(target);
      else if (action === "restart") await restartApp(target);
      else if (action === "extend") {
        const newExpiry = Math.floor(Date.now() / 1000) + 24 * 3600;
        await saveAppConfig(target, { expires_at: newExpiry });
      }
      setRowNotice(key, "ok", action === "extend" ? "Expiry extended +24h" : `${action} sent`);
      onUpdated();
    } catch (e) {
      setRowNotice(key, "error", e instanceof Error ? e.message : "Failed");
    } finally {
      clearRowBusy(key);
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <div className="eyebrow mb-0.5">Environment management</div>
          <h1 className="text-xl font-semibold text-white">Lanes</h1>
          <p className="text-xs text-white/40 mt-0.5">{project.name} · {contexts.length} {contexts.length === 1 ? "lane" : "lanes"}</p>
        </div>
        {isDeployer && (
          <button
            type="button"
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 text-sm bg-relay-accent/90 hover:bg-relay-accent text-white font-semibold px-3 py-1.5 rounded transition-colors"
          >
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            New lane
          </button>
        )}
      </div>

      {contexts.length === 0 ? (
        <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-10 text-center">
          <div className="text-white/30 text-sm mb-3">No lanes configured for this project.</div>
          {isDeployer && (
            <button
              type="button"
              onClick={() => setShowCreate(true)}
              className="text-sm text-relay-accent hover:underline"
            >
              Create the first lane →
            </button>
          )}
        </div>
      ) : (
        <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl overflow-hidden">
          {/* Table header */}
          <div className="grid grid-cols-[1fr_80px_80px_100px_100px_auto] gap-3 px-4 py-2.5 border-b border-white/[0.06] text-[10px] uppercase tracking-widest text-white/25">
            <span>Lane</span>
            <span>Engine</span>
            <span>Traffic</span>
            <span>Status</span>
            <span>Expiry</span>
            <span className="text-right">Actions</span>
          </div>

          {/* Rows */}
          <div className="divide-y divide-white/[0.04]">
            {contexts.map((ctx) => {
              const key = deployKey(ctx.app, ctx.env, ctx.branch);
              const expiry = expiryCountdown(ctx.expires_at);
              const drain = drainLabel(ctx.drain_until);
              const rowBusy = busy[key];
              const notice = notices[key];
              const route = computeConfiguredURL(ctx) || computePreviewURL(ctx, ctx.latestDeploy ?? null);
              const isStopped = ctx.stopped;

              return (
                <div key={key} className="grid grid-cols-[1fr_80px_80px_100px_100px_auto] gap-3 items-center px-4 py-3">
                  {/* Lane info */}
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 mb-0.5">
                      <span
                        className={cn(
                          "w-1.5 h-1.5 rounded-full shrink-0",
                          isStopped ? "bg-white/20" :
                          ctx.latestDeploy?.status === "success" ? "bg-emerald-500" :
                          ctx.latestDeploy?.status !== undefined ? "bg-amber-400 animate-pulse" : "bg-white/20",
                        )}
                      />
                      <span className="text-sm font-semibold text-white">{ctx.env}</span>
                      {ctx.branch !== "main" && (
                        <span className="text-[10px] text-white/40 bg-white/[0.04] px-1.5 py-0.5 rounded font-mono">{ctx.branch}</span>
                      )}
                    </div>
                    {route && (
                      <a href={route} target="_blank" rel="noreferrer" className="text-[11px] text-white/30 hover:text-white/60 truncate block max-w-xs transition-colors">
                        {route}
                      </a>
                    )}
                    {drain && (
                      <div className="flex items-center gap-1 mt-1">
                        <span className="w-1 h-1 rounded-full bg-amber-400 animate-pulse" />
                        <span className="text-[10px] text-amber-400">Draining until {drain}</span>
                      </div>
                    )}
                    {notice && (
                      <div className={cn("text-[10px] mt-1", notice.tone === "ok" ? "text-emerald-400" : "text-red-400")}>{notice.text}</div>
                    )}
                  </div>

                  {/* Engine */}
                  <span className="text-xs text-white/50">{engineLabel(ctx.engine ?? "docker")}</span>

                  {/* Traffic */}
                  <div className="space-y-0.5">
                    <div className="text-xs text-white/50">{trafficModeLabel(ctx.traffic_mode ?? "")}</div>
                    <div className="text-[10px] text-white/30">{rolloutStrategy(ctx)}</div>
                  </div>

                  {/* Status */}
                  <div>
                    {isStopped ? (
                      <span className="text-[10px] bg-white/[0.04] border border-white/[0.08] text-white/40 px-2 py-0.5 rounded">stopped</span>
                    ) : ctx.latestDeploy ? (
                      <span className={cn(
                        "text-[10px] px-2 py-0.5 rounded border font-semibold uppercase tracking-wide",
                        ctx.latestDeploy.status === "success"
                          ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
                          : "bg-amber-500/10 border-amber-500/30 text-amber-400",
                      )}>
                        {ctx.latestDeploy.status}
                      </span>
                    ) : (
                      <span className="text-[10px] text-white/25">idle</span>
                    )}
                    {ctx.latestDeploy?.created_at && (
                      <div className="text-[10px] text-white/25 mt-0.5">{timeAgo(ctx.latestDeploy.created_at)} ago</div>
                    )}
                  </div>

                  {/* Expiry */}
                  <div>
                    {expiry ? (
                      <span className={cn(
                        "text-[10px] px-2 py-0.5 rounded border",
                        expiry.urgent
                          ? "bg-red-500/10 border-red-500/30 text-red-400"
                          : "bg-white/[0.04] border-white/[0.08] text-white/40",
                      )}>
                        {expiry.label}
                      </span>
                    ) : (
                      <span className="text-[10px] text-white/20">—</span>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-1 justify-end flex-wrap">
                    {isDeployer && (
                      <>
                        {isStopped ? (
                          <button
                            type="button"
                            disabled={!!rowBusy}
                            onClick={() => handleAction("start", ctx)}
                            className="text-[11px] text-emerald-400 hover:text-emerald-300 border border-emerald-500/20 px-2 py-1 rounded hover:bg-emerald-500/10 transition-colors disabled:opacity-40"
                          >
                            {rowBusy === "start" ? "…" : "Start"}
                          </button>
                        ) : (
                          <button
                            type="button"
                            disabled={!!rowBusy}
                            onClick={() => handleAction("stop", ctx)}
                            className="text-[11px] text-white/50 hover:text-white border border-white/[0.08] px-2 py-1 rounded hover:bg-white/[0.06] transition-colors disabled:opacity-40"
                          >
                            {rowBusy === "stop" ? "…" : "Stop"}
                          </button>
                        )}
                        <button
                          type="button"
                          disabled={!!rowBusy}
                          onClick={() => handleAction("restart", ctx)}
                          className="text-[11px] text-white/50 hover:text-white border border-white/[0.08] px-2 py-1 rounded hover:bg-white/[0.06] transition-colors disabled:opacity-40"
                        >
                          {rowBusy === "restart" ? "…" : "Restart"}
                        </button>
                        {ctx.expires_at && ctx.expires_at > 0 && (
                          <button
                            type="button"
                            disabled={!!rowBusy}
                            onClick={() => handleAction("extend", ctx)}
                            className="text-[11px] text-relay-teal hover:text-relay-teal/80 border border-relay-teal/20 px-2 py-1 rounded hover:bg-relay-teal/10 transition-colors disabled:opacity-40"
                          >
                            {rowBusy === "extend" ? "…" : "+24h"}
                          </button>
                        )}
                      </>
                    )}
                    <button
                      type="button"
                      onClick={() => onNavigateSettings(key)}
                      className="text-[11px] text-white/50 hover:text-white border border-white/[0.08] px-2 py-1 rounded hover:bg-white/[0.06] transition-colors"
                    >
                      Manage
                    </button>
                    {isOwner && (
                      <button
                        type="button"
                        onClick={() => setDeleteTarget({ app: ctx.app, env: ctx.env, branch: ctx.branch })}
                        className="text-[11px] text-red-400/70 hover:text-red-400 border border-red-500/10 px-2 py-1 rounded hover:bg-red-500/10 transition-colors"
                      >
                        Delete
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {showCreate && (
        <CreateLaneModal
          app={project.name}
          onClose={() => setShowCreate(false)}
          onCreated={() => { setShowCreate(false); onUpdated(); }}
        />
      )}

      {deleteTarget && (
        <DeleteLaneDialog
          target={deleteTarget}
          onClose={() => setDeleteTarget(null)}
          onDeleted={() => { setDeleteTarget(null); onUpdated(); }}
        />
      )}
    </div>
  );
}
