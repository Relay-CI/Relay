"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import {
  deployDurationLabel,
  deployPhaseText,
  deployStatusClass,
  formatCommitSHA,
  formatDateTime,
  logLineTone,
  buildLogStats,
  operationClass,
  operationLabel,
  repoProviderInfo,
  rolloutStrategy,
  shortImageTag,
  timeAgo,
  trafficModeLabel,
  computeConfiguredURL,
  computePreviewURL,
  engineLabel,
  oldTargetLabel,
  ACTIVE_STATUSES,
} from "@/lib/relay-utils";
import type { Deploy, EnvInfo, Service } from "@/lib/api";

interface DeployDetailDialogProps {
  deploy: Deploy;
  envInfo: EnvInfo | null;
  services: Service[];
  onClose: () => void;
  onCancel?: (deployId: string) => Promise<void>;
}

export function DeployDetailDialog({ deploy, envInfo, services, onClose, onCancel }: DeployDetailDialogProps) {
  const [lines, setLines] = useState<string[]>([]);
  const [status, setStatus] = useState("connecting");
  const logRef = useRef<HTMLDivElement>(null);

  const logStats = buildLogStats(lines);
  const preview = envInfo ? computePreviewURL(envInfo, deploy) : "";
  const configuredURL = envInfo ? computeConfiguredURL(envInfo) : "";
  const repoInfo = repoProviderInfo(envInfo?.repo_url ?? "");

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

        const reader = res.body!.getReader();
        const decoder = new TextDecoder("utf-8");
        let buffer = "";

        while (!cancelled) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let idx: number;
          while ((idx = buffer.indexOf("\n\n")) >= 0) {
            const frame = buffer.slice(0, idx);
            buffer = buffer.slice(idx + 2);
            let eventName = "message";
            const data: string[] = [];
            frame.split("\n").forEach((line) => {
              if (line.startsWith("event: ")) eventName = line.slice(7).trim();
              if (line.startsWith("data: ")) data.push(line.slice(6));
            });
            if (!data.length) continue;
            const payload = data.join("\n");
            if (eventName === "deploy-status") {
              try {
                const parsed = JSON.parse(payload) as { status: string };
                setStatus(parsed.status || "complete");
              } catch {
                setStatus(payload || "complete");
              }
              continue;
            }
            setLines((prev) => {
              const next = [...prev, payload];
              if (logRef.current) {
                requestAnimationFrame(() => {
                  if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
                });
              }
              return next;
            });
          }
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

  const isActive = ACTIVE_STATUSES.has(status as Parameters<typeof ACTIVE_STATUSES.has>[0]);
  const statusVariant = status === "success" || status === "complete" ? "ok" : status === "error" ? "danger" : "warn";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="bg-zinc-950 border border-white/[0.08] rounded-xl w-full max-w-5xl max-h-[90vh] flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-start justify-between p-5 border-b border-white/[0.06] shrink-0">
          <div>
            <div className="eyebrow mb-1">{operationLabel(deploy.source)} detail</div>
            <h2 className="text-base font-semibold text-white font-mono">{deploy.id}</h2>
            <div className="flex flex-wrap gap-1.5 mt-2">
              <span className={cn("text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded border", operationClass(deploy.source))}>
                {operationLabel(deploy.source)}
              </span>
              {deploy.image_tag && (
                <span className="text-[10px] font-mono bg-white/[0.04] border border-white/[0.08] px-2 py-0.5 rounded text-white/60">
                  to {shortImageTag(deploy.image_tag)}
                </span>
              )}
              {deploy.previous_image_tag && (
                <span className="text-[10px] font-mono bg-white/[0.04] border border-white/[0.08] px-2 py-0.5 rounded text-white/40">
                  from {shortImageTag(deploy.previous_image_tag)}
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <span className={cn("text-xs px-2.5 py-1 rounded-full border font-medium", {
              "bg-emerald-500/10 border-emerald-500/30 text-emerald-400": statusVariant === "ok",
              "bg-red-500/10 border-red-500/30 text-red-400": statusVariant === "danger",
              "bg-amber-500/10 border-amber-500/30 text-amber-400": statusVariant === "warn",
            })}>
              {isActive ? "streaming" : status}
            </span>
            {isActive && onCancel && (
              <button
                type="button"
                onClick={() => onCancel(deploy.id)}
                className="text-xs px-2.5 py-1 rounded-full border border-red-500/30 bg-red-500/10 text-red-400 hover:bg-red-500/20 transition-colors"
              >
                Cancel build
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              className="text-white/40 hover:text-white transition-colors w-7 h-7 flex items-center justify-center rounded hover:bg-white/[0.06]"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="flex flex-1 min-h-0 overflow-hidden">
          {/* Sidebar */}
          <div className="w-52 shrink-0 border-r border-white/[0.06] p-4 overflow-y-auto flex flex-col gap-3">
            {[
              { label: "Created", value: formatDateTime(deploy.created_at), meta: `${timeAgo(deploy.created_at)} ago` },
              { label: "Duration", value: deployDurationLabel(deploy), meta: "Build to completion" },
              { label: "Environment", value: deploy.env, meta: deploy.branch },
              { label: "Services", value: String(services.length), meta: "Companion containers" },
            ].map(({ label, value, meta }) => (
              <div key={label} className="bg-white/[0.03] rounded-lg p-3">
                <div className="eyebrow mb-1">{label}</div>
                <div className="text-sm font-semibold text-white">{value}</div>
                <div className="text-[11px] text-white/40 mt-0.5">{meta}</div>
              </div>
            ))}

            {/* Routing */}
            <div className="space-y-1.5">
              <div className="eyebrow">Build Summary</div>
              <KVRow label="Commit" value={formatCommitSHA(deploy.commit_sha ?? "")} mono />
              <KVRow label="Source" value={repoInfo.label} />
              <KVRow label="Image" value={shortImageTag(deploy.image_tag ?? "")} mono />
              <KVRow label="Previous" value={shortImageTag(deploy.previous_image_tag ?? "")} mono />
            </div>

            {envInfo && (
              <div className="space-y-1.5">
                <div className="eyebrow">Routing</div>
                <KVRow label="Preview" value={preview || "Private route"} />
                <KVRow label="Engine" value={engineLabel(envInfo.engine ?? "docker")} />
                <KVRow label="Mode" value={envInfo.mode ?? ""} />
                <KVRow label="Traffic" value={trafficModeLabel(envInfo.traffic_mode ?? "")} />
                <KVRow label="Rollout" value={rolloutStrategy(envInfo)} />
                <KVRow label="Standby" value={oldTargetLabel(envInfo)} />
              </div>
            )}
          </div>

          {/* Log terminal */}
          <div className="flex-1 min-w-0 flex flex-col">
            <div className="flex items-center justify-between px-4 py-3 border-b border-white/[0.06] shrink-0">
              <div>
                <div className="eyebrow">Build Stream</div>
                <h3 className="text-sm font-semibold text-white mt-0.5">Live output</h3>
              </div>
              <div className="flex gap-2 text-xs">
                <span className="bg-white/[0.04] border border-white/[0.08] px-2 py-0.5 rounded text-white/60">{logStats.total} lines</span>
                {logStats.warnings > 0 && <span className="bg-amber-500/10 border border-amber-500/20 px-2 py-0.5 rounded text-amber-400">{logStats.warnings} warns</span>}
                {logStats.errors > 0 && <span className="bg-red-500/10 border border-red-500/20 px-2 py-0.5 rounded text-red-400">{logStats.errors} errors</span>}
              </div>
            </div>
            <div
              ref={logRef}
              className="flex-1 min-h-0 overflow-y-auto p-4 font-mono text-xs leading-relaxed space-y-0.5 bg-[#040404]"
            >
              {!lines.length && (
                <div className="text-white/30">Connecting to log stream…</div>
              )}
              {lines.map((line, index) => (
                <div
                  key={`${index}-${line.slice(0, 8)}`}
                  className={cn("log-line", `log-line--${logLineTone(line)}`)}
                >
                  {line}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function KVRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-2">
      <span className="text-[11px] text-white/35 shrink-0">{label}</span>
      <span className={cn("text-[11px] text-white/70 text-right min-w-0 break-all", mono && "font-mono")}>
        {value || "—"}
      </span>
    </div>
  );
}
