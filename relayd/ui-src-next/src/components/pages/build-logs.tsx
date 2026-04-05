"use client";

import { cn } from "@/lib/utils";
import {
  deployKey,
  deployPhaseText,
  deployDurationLabel,
  deployStatusClass,
  formatCommitSHA,
  timeAgo,
  buildLogStats,
  type NormalizedProject,
  ACTIVE_STATUSES,
} from "@/lib/relay-utils";
import type { Deploy } from "@/lib/api";

interface BuildLogsPageProps {
  deploys: Deploy[];
  onOpenDeploy: (deploy: Deploy) => void;
}

export function BuildLogsPage({ deploys, onOpenDeploy }: BuildLogsPageProps) {
  return (
    <div className="space-y-4">
      <div>
        <div className="eyebrow mb-0.5">Build surface</div>
        <h1 className="text-xl font-semibold text-white">Build logs</h1>
        <p className="text-xs text-white/40 mt-1">
          Inspect raw build output for any deployment. Click a build entry to stream its log.
        </p>
      </div>

      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl overflow-hidden">
        <div className="grid grid-cols-[auto_1fr_auto_auto_auto] text-[10px] uppercase tracking-wider text-white/30 font-semibold border-b border-white/[0.06] px-4 py-2.5 gap-3">
          <span>State</span>
          <span>Build</span>
          <span className="text-right">Lane</span>
          <span className="text-right">Duration</span>
          <span />
        </div>
        {!deploys.length ? (
          <div className="text-sm text-white/30 text-center py-10">No builds found. Trigger a deploy to see build logs here.</div>
        ) : (
          <div className="divide-y divide-white/[0.04]">
            {deploys.map((deploy) => {
              const isActive = ACTIVE_STATUSES.has(deploy.status as Parameters<typeof ACTIVE_STATUSES.has>[0]);
              const stats = deploy.log ? buildLogStats(deploy.log.split("\n")) : null;
              return (
                <button
                  type="button"
                  key={deploy.id}
                  onClick={() => onOpenDeploy(deploy)}
                  className={cn("w-full text-left grid grid-cols-[auto_1fr_auto_auto_auto] gap-3 items-center px-4 py-3 hover:bg-white/[0.03] transition-colors group", isActive && "border-l-2 border-l-amber-400")}
                >
                  <StatusDot status={deploy.status} />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-white">
                        {deploy.build_number ? `#${deploy.build_number}` : deploy.id.slice(0, 8)}
                      </span>
                      <span className={cn("text-[10px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded border", statusBadgeClass(deploy.status))}>
                        {deployPhaseText(deploy)}
                      </span>
                    </div>
                    <div className="text-xs text-white/35 flex gap-2 mt-0.5">
                      {deploy.commit_sha && <span className="font-mono">{formatCommitSHA(deploy.commit_sha)}</span>}
                      {deploy.commit_message && <span className="truncate max-w-[320px]">{deploy.commit_message}</span>}
                      {stats && stats.errors > 0 && (
                        <span className="text-red-400">{stats.errors} error{stats.errors !== 1 ? "s" : ""}</span>
                      )}
                      {stats && stats.warnings > 0 && (
                        <span className="text-amber-400">{stats.warnings} warning{stats.warnings !== 1 ? "s" : ""}</span>
                      )}
                    </div>
                  </div>
                  <div className="text-xs text-white/40 text-right shrink-0">{deploy.env}<span className="text-white/20">/</span>{deploy.branch}</div>
                  <div className="text-xs text-white/40 text-right font-mono shrink-0">{deployDurationLabel(deploy)}</div>
                  <div className="text-xs text-relay-accent group-hover:text-white transition-colors shrink-0">View logs →</div>
                </button>
              );
            })}
          </div>
        )}
      </div>
      <div className="text-xs text-white/25 text-right">{deploys.length} builds</div>
    </div>
  );
}

function StatusDot({ status }: { status: string }) {
  const isActive = ACTIVE_STATUSES.has(status as Parameters<typeof ACTIVE_STATUSES.has>[0]);
  const cl = deployStatusClass(status);
  return (
    <span className={cn("w-2 h-2 rounded-full shrink-0", isActive ? "bg-amber-400 animate-pulse" : cl.includes("ok") ? "bg-emerald-500" : cl.includes("error") ? "bg-red-500" : "bg-white/20")} />
  );
}

function statusBadgeClass(status: string) {
  const cl = deployStatusClass(status);
  if (cl.includes("ok")) return "bg-emerald-500/10 border-emerald-500/30 text-emerald-400";
  if (cl.includes("error")) return "bg-red-500/10 border-red-500/30 text-red-400";
  return "bg-amber-500/10 border-amber-500/30 text-amber-400";
}
