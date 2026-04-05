"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import { useRuntimeLogs } from "@/hooks/use-runtime-logs";
import {
  runtimeLogLevel,
  runtimeLogLevelVariant,
  runtimeFilterMatches,
  describeRuntimeLaneState,
} from "@/lib/relay-utils";
import type { EnvInfo } from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
}

interface RuntimeLogsPageProps {
  selectedEnv: SelectedEnvMeta | null;
}

const WINDOWS = [
  { label: "1h", value: "1h" },
  { label: "6h", value: "6h" },
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
] as const;

const LEVELS = ["", "error", "warn", "info", "debug"] as const;

export function RuntimeLogsPage({ selectedEnv }: RuntimeLogsPageProps) {
  const [window, setWindow] = useState("1h");
  const [levelFilter, setLevelFilter] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedTarget, setSelectedTarget] = useState<string>("");

  const envArg = selectedEnv ?? null;
  const { targets, lines, status: state, error } = useRuntimeLogs(envArg, selectedTarget, window);

  const visible = lines.filter((entry) =>
    runtimeFilterMatches(entry, levelFilter, searchQuery)
  );

  return (
    <div className="flex flex-col gap-4 h-full">
      <div>
        <div className="eyebrow mb-0.5">Runtime surface</div>
        <h1 className="text-xl font-semibold text-white">Runtime logs</h1>
        <p className="text-xs text-white/40 mt-1">Live stream from the running container. Reconnects automatically.</p>
      </div>

      {/* Controls */}
      <div className="flex flex-wrap gap-2 items-center">
        {/* Target selector */}
        {targets.length > 0 && (
          <select
            value={selectedTarget ?? ""}
            onChange={(e) => setSelectedTarget(e.target.value)}
            className="text-sm bg-white/[0.04] border border-white/[0.08] rounded px-2.5 py-1.5 text-white/70 focus:outline-none focus:border-white/20"
          >
            <option value="">Auto target</option>
            {targets.map((t) => (
              <option key={t.id} value={t.id}>{t.label ?? t.id}</option>
            ))}
          </select>
        )}

        {/* Time window */}
        <div className="flex gap-1">
          {WINDOWS.map((w) => (
            <button
              key={w.value}
              type="button"
              onClick={() => setWindow(w.value)}
              className={cn("text-[11px] px-2.5 py-1 rounded border transition-colors", window === w.value ? "bg-white/[0.08] border-white/20 text-white" : "border-white/[0.06] text-white/40 hover:text-white/60")}
            >
              {w.label}
            </button>
          ))}
        </div>

        {/* Level filter */}
        <div className="flex gap-1">
          {LEVELS.map((lv) => (
            <button
              key={lv || "all"}
              type="button"
              onClick={() => setLevelFilter(lv)}
              className={cn("text-[11px] px-2.5 py-1 rounded border transition-colors", levelFilter === lv ? "bg-white/[0.08] border-white/20 text-white" : "border-white/[0.06] text-white/40 hover:text-white/60")}
            >
              {lv || "all"}
            </button>
          ))}
        </div>

        <div className="ml-auto flex items-center gap-2">
          <input
            type="text"
            placeholder="Filter logs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="text-sm bg-white/[0.04] border border-white/[0.08] rounded px-3 py-1.5 text-white/70 placeholder:text-white/25 focus:outline-none focus:border-white/20 w-44"
          />
          <StatusBadge state={state} />
        </div>
      </div>

      {/* Log terminal */}
      <div className="flex-1 bg-black/60 border border-white/[0.06] rounded-xl flex flex-col overflow-hidden min-h-[400px]">
        <div className="flex items-center justify-between px-4 py-2 border-b border-white/[0.06] bg-white/[0.02]">
          <div className="flex items-center gap-2">
            <span className="text-[10px] text-white/30 font-mono">stdout/stderr</span>
            {selectedEnv && (
              <span className="text-[10px] bg-white/[0.04] px-2 py-0.5 rounded text-white/30">
                {selectedEnv.env}/{selectedEnv.branch}
              </span>
            )}
          </div>
          <span className="text-[10px] text-white/25">{visible.length} lines</span>
        </div>
        <div className="flex-1 overflow-y-auto p-4 font-mono text-xs space-y-0.5">
          {state === "idle" || state === "connecting" ? (
            <div className="flex items-center gap-2 text-white/30">
              <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse" />
              {state === "connecting" ? "Connecting to log stream..." : "Waiting for log stream..."}
            </div>
          ) : error ? (
            <div className="text-red-400">{error}</div>
          ) : !visible.length ? (
            <div className="text-white/20">No log entries yet. The output will appear here as messages arrive.</div>
          ) : (
            visible.map((entry, i) => (
              <div key={i} className={cn("log-line flex gap-3 leading-relaxed", logLevelClass(entry.level ?? ""))}>
                <span className="text-white/20 shrink-0 select-none">{entry.timestamp ? entry.timestamp.slice(11, 23) : ""}</span>
                {entry.level && <span className={cn("shrink-0 w-10 text-right uppercase text-[10px] font-semibold", levelColor(entry.level))}>{entry.level}</span>}
                <span className="flex-1 break-all whitespace-pre-wrap">{entry.message}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function StatusBadge({ state }: { state: string }) {
  const configMap: Record<string, { dot: string; label: string }> = {
    idle: { dot: "bg-white/20", label: "idle" },
    connecting: { dot: "bg-amber-400 animate-pulse", label: "connecting" },
    live: { dot: "bg-emerald-500 animate-pulse", label: "live" },
    complete: { dot: "bg-white/30", label: "complete" },
    offline: { dot: "bg-white/20", label: "offline" },
    error: { dot: "bg-red-500", label: "error" },
  };
  const cfg = configMap[state] ?? configMap.idle;
  return (
    <div className="flex items-center gap-1.5 text-[11px] text-white/40">
      <span className={cn("w-1.5 h-1.5 rounded-full", cfg.dot)} />
      {cfg.label}
    </div>
  );
}

function logLevelClass(level: string) {
  if (level === "error") return "log-line--error";
  if (level === "warn") return "log-line--warn";
  if (level === "info") return "log-line--info";
  return "";
}

function levelColor(level: string) {
  if (level === "error" || level === "fatal") return "text-red-400";
  if (level === "warn") return "text-amber-400";
  if (level === "info") return "text-sky-400";
  if (level === "debug" || level === "trace") return "text-white/35";
  return "text-white/50";
}
