"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import { saveAppConfig } from "@/lib/api";
import type { EnvInfo } from "@/lib/api";
import type { NormalizedProject } from "@/lib/relay-utils";

interface ResourcesPageProps {
  selectedEnv: EnvInfo | null;
  project: NormalizedProject | null;
  onUpdated: () => void;
}

const CPU_PRESETS = [
  { label: "0.25 CPU", value: "0.25" },
  { label: "0.5 CPU", value: "0.5" },
  { label: "1 CPU", value: "1" },
  { label: "2 CPU", value: "2" },
  { label: "4 CPU", value: "4" },
];

const MEM_PRESETS = [
  { label: "128 MB", value: "128m" },
  { label: "256 MB", value: "256m" },
  { label: "512 MB", value: "512m" },
  { label: "1 GB", value: "1g" },
  { label: "2 GB", value: "2g" },
  { label: "4 GB", value: "4g" },
];

function UsageBar({ value, max, color }: { value: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0;
  return (
    <div className="h-1.5 w-full bg-white/[0.06] rounded-full overflow-hidden">
      <div
        className={cn("h-full rounded-full transition-all", color)}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}

function ResourceCard({
  env,
  onUpdated,
}: {
  env: EnvInfo;
  onUpdated: () => void;
}) {
  const [mode, setMode] = useState<"manual" | "auto">(
    (env.resource_mode as "manual" | "auto") ?? "manual"
  );
  const [cpuLimit, setCpuLimit] = useState(env.cpu_limit ?? "");
  const [memLimit, setMemLimit] = useState(env.mem_limit ?? "");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "err"; text: string } | null>(null);

  const isDirty =
    mode !== (env.resource_mode ?? "manual") ||
    cpuLimit !== (env.cpu_limit ?? "") ||
    memLimit !== (env.mem_limit ?? "");

  async function save() {
    setBusy(true);
    setNotice(null);
    try {
      await saveAppConfig(
        { app: env.app, env: env.env, branch: env.branch },
        { cpu_limit: cpuLimit, mem_limit: memLimit, resource_mode: mode },
      );
      setNotice({ tone: "ok", text: "Saved. Limits apply on next container start." });
      onUpdated();
    } catch (err) {
      setNotice({ tone: "err", text: err instanceof Error ? err.message : "Save failed" });
    } finally {
      setBusy(false);
    }
  }

  const laneLabel = env.env === env.branch || env.branch === "main"
    ? env.env
    : `${env.env} / ${env.branch}`;

  return (
    <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="eyebrow mb-0.5">{env.app}</div>
          <div className="text-base font-semibold text-white">{laneLabel}</div>
        </div>
        <div className="flex items-center gap-1.5">
          <span className={cn(
            "w-2 h-2 rounded-full",
            env.stopped ? "bg-white/20" : "bg-relay-success"
          )} />
          <span className="text-xs text-white/40">{env.stopped ? "stopped" : "running"}</span>
        </div>
      </div>

      {/* Mode toggle */}
      <div>
        <div className="text-xs text-white/40 mb-2">Resource mode</div>
        <div className="seg-control flex">
          <button
            type="button"
            onClick={() => setMode("manual")}
            className={cn("seg-btn flex-1 text-xs py-1.5", mode === "manual" && "seg-btn--active")}
          >
            Manual
          </button>
          <button
            type="button"
            onClick={() => setMode("auto")}
            className={cn("seg-btn flex-1 text-xs py-1.5", mode === "auto" && "seg-btn--active")}
          >
            Automatic
          </button>
        </div>
        {mode === "auto" && (
          <p className="text-xs text-white/35 mt-2 leading-relaxed">
            Automatic mode currently runs the lane without fixed CPU or memory caps. Relay saves the manual values here, but it does not tune them dynamically yet.
          </p>
        )}
      </div>

      {mode === "manual" && (
        <>
          {/* CPU limit */}
          <div>
            <div className="text-xs text-white/40 mb-2">CPU limit</div>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {CPU_PRESETS.map((p) => (
                <button
                  key={p.value}
                  type="button"
                  onClick={() => setCpuLimit(p.value)}
                  className={cn(
                    "text-xs px-2.5 py-1 rounded border transition-colors",
                    cpuLimit === p.value
                      ? "border-relay-accent/60 bg-relay-accent/15 text-white"
                      : "border-white/[0.08] text-white/50 hover:border-white/20 hover:text-white"
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
            <input
              className="text-input w-full"
              placeholder="Custom, e.g. 1.5"
              value={cpuLimit}
              onChange={(e) => setCpuLimit(e.target.value)}
            />
            <p className="text-[11px] text-white/30 mt-1">Docker <code className="font-mono">--cpus</code> value. Leave blank for no limit.</p>
          </div>

          {/* Memory limit */}
          <div>
            <div className="text-xs text-white/40 mb-2">Memory limit</div>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {MEM_PRESETS.map((p) => (
                <button
                  key={p.value}
                  type="button"
                  onClick={() => setMemLimit(p.value)}
                  className={cn(
                    "text-xs px-2.5 py-1 rounded border transition-colors",
                    memLimit === p.value
                      ? "border-relay-accent/60 bg-relay-accent/15 text-white"
                      : "border-white/[0.08] text-white/50 hover:border-white/20 hover:text-white"
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
            <input
              className="text-input w-full"
              placeholder="Custom, e.g. 768m or 1.5g"
              value={memLimit}
              onChange={(e) => setMemLimit(e.target.value)}
            />
            <p className="text-[11px] text-white/30 mt-1">Docker <code className="font-mono">--memory</code> value. Leave blank for no limit.</p>
          </div>
        </>
      )}

      {/* Summary badges */}
      <div className="flex gap-2 flex-wrap">
        <span className="text-[11px] bg-white/[0.04] border border-white/[0.06] px-2 py-1 rounded font-mono text-white/50">
          cpu: {mode === "auto" ? "auto" : (cpuLimit || "unlimited")}
        </span>
        <span className="text-[11px] bg-white/[0.04] border border-white/[0.06] px-2 py-1 rounded font-mono text-white/50">
          mem: {mode === "auto" ? "auto" : (memLimit || "unlimited")}
        </span>
      </div>

      {notice && (
        <div className={cn(
          "rounded-lg px-4 py-3 text-sm border",
          notice.tone === "ok"
            ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
            : "bg-red-500/10 border-red-500/30 text-red-400"
        )}>
          {notice.text}
        </div>
      )}

      <button
        type="button"
        onClick={save}
        disabled={busy || !isDirty}
        className={cn(
          "text-sm px-4 py-2 rounded font-semibold transition-colors",
          isDirty
            ? "bg-white text-black hover:bg-white/90"
            : "bg-white/[0.06] text-white/30 cursor-not-allowed"
        )}
      >
        {busy ? "Saving…" : "Save resource config"}
      </button>
    </div>
  );
}

export function ResourcesPage({ selectedEnv, project, onUpdated }: ResourcesPageProps) {
  if (!project) {
    return (
      <div className="flex items-center justify-center h-full text-white/30 text-sm">
        Select a project to manage resources.
      </div>
    );
  }

  const envs = project.envs;

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Infrastructure</div>
        <h1 className="text-xl font-semibold text-white">Resource limits</h1>
        <p className="text-sm text-white/40 mt-1">
          Configure CPU and memory limits per lane. Use manual caps for predictable limits, or automatic mode to run the lane without fixed caps.
          Limits take effect on the next container start or deploy.
        </p>
      </div>

      {/* How it works */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
        <div className="eyebrow mb-1">How it works</div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-3">
          {[
            {
              icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/></svg>,
              title: "Manual limits",
              detail: "Set fixed CPU and memory caps. Docker enforces them at runtime. Good for predictable workloads."
            },
            {
              icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>,
              title: "Automatic",
              detail: "Automatic mode currently means no fixed CPU or memory caps. It is useful when you want Docker to manage headroom instead of enforcing a hard ceiling."
            },
            {
              icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>,
              title: "Apply on restart",
              detail: "Changes are applied when the container is next started. Restart or re-deploy the lane to make them active immediately."
            },
          ].map((row) => (
            <div key={row.title} className="flex gap-3">
              <span className="text-relay-accent shrink-0 mt-0.5">{row.icon}</span>
              <div>
                <div className="text-sm font-medium text-white">{row.title}</div>
                <div className="text-xs text-white/40 mt-0.5 leading-relaxed">{row.detail}</div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Per-lane cards — highlight selected lane first */}
      {selectedEnv && (
        <ResourceCard
          key={`${selectedEnv.app}--${selectedEnv.env}--${selectedEnv.branch}`}
          env={selectedEnv}
          onUpdated={onUpdated}
        />
      )}

      {envs
        .filter(
          (e) =>
            !selectedEnv ||
            e.env !== selectedEnv.env ||
            e.branch !== selectedEnv.branch
        )
        .map((e) => {
          const fullEnv = { ...e, app: project.name } as EnvInfo;
          return (
            <ResourceCard
              key={`${fullEnv.app}--${fullEnv.env}--${fullEnv.branch}`}
              env={fullEnv}
              onUpdated={onUpdated}
            />
          );
        })}

      {envs.length === 0 && (
        <div className="text-sm text-white/30 text-center py-8">
          No lanes deployed yet.
        </div>
      )}
    </div>
  );
}
