"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import {
  deployKey,
  deployPhaseText,
  deployDurationLabel,
  deployStatusClass,
  engineLabel,
  formatCommitSHA,
  timeAgo,
  ACTIVE_STATUSES,
  type NormalizedProject,
} from "@/lib/relay-utils";
import type { Deploy, EnvInfo } from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
}

interface DeploymentsPageProps {
  deploys: Deploy[];
  envMap: Map<string, SelectedEnvMeta>;
  selectedEnv: SelectedEnvMeta | null;
  onOpenDeploy: (deploy: Deploy) => void;
  onCancelDeploy?: (deployId: string) => Promise<void>;
}

const ALL = "__all__";

export function DeploymentsPage({ deploys, envMap, selectedEnv, onOpenDeploy, onCancelDeploy }: DeploymentsPageProps) {
  const [envFilter, setEnvFilter] = useState<string>(ALL);
  const [statusFilter, setStatusFilter] = useState<string>(ALL);
  const [searchQuery, setSearchQuery] = useState("");

  const envKeys = Array.from(new Set(deploys.map((d) => deployKey(d.app, d.env, d.branch))));

  const filtered = deploys.filter((d) => {
    if (envFilter !== ALL && deployKey(d.app, d.env, d.branch) !== envFilter) return false;
    if (statusFilter !== ALL && d.status !== statusFilter) return false;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      if (!d.id.includes(q) && !(d.commit_sha ?? "").includes(q) && !(d.branch ?? "").includes(q)) return false;
    }
    return true;
  });

  return (
    <div className="space-y-4">
      <div>
        <div className="eyebrow mb-0.5">Deploy board</div>
        <h1 className="text-xl font-semibold text-white">Deployments</h1>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-2 items-center">
        <div className="flex gap-1.5 flex-wrap">
          <FilterPill active={envFilter === ALL} onClick={() => setEnvFilter(ALL)}>All lanes</FilterPill>
          {envKeys.map((k) => {
            const ctx = envMap.get(k);
            const label = ctx ? `${ctx.env}/${ctx.branch}` : k;
            return <FilterPill key={k} active={envFilter === k} onClick={() => setEnvFilter(k)}>{label}</FilterPill>;
          })}
        </div>
        <div className="h-4 w-px bg-white/[0.08]" />
        <div className="flex gap-1.5">
          <FilterPill active={statusFilter === ALL} onClick={() => setStatusFilter(ALL)}>All statuses</FilterPill>
          <FilterPill active={statusFilter === "success"} onClick={() => setStatusFilter("success")}>Success</FilterPill>
          <FilterPill active={statusFilter === "failed"} onClick={() => setStatusFilter("failed")}>Failed</FilterPill>
          <FilterPill active={statusFilter === "building"} onClick={() => setStatusFilter("building")}>Active</FilterPill>
        </div>
        <div className="ml-auto">
          <input
            type="text"
            placeholder="Search commit, branch..."
            className="text-sm bg-white/[0.04] border border-white/[0.08] rounded px-3 py-1.5 text-white/70 placeholder:text-white/25 focus:outline-none focus:border-white/20 w-52"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
      </div>

      {/* Table */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl overflow-hidden">
        <div className="grid grid-cols-[auto_1fr_auto_auto_auto_auto] text-[10px] uppercase tracking-wider text-white/30 font-semibold border-b border-white/[0.06] px-4 py-2.5 gap-3">
          <span>Status</span>
          <span>Build / commit</span>
          <span className="text-right">Lane</span>
          <span className="text-right">Duration</span>
          <span className="text-right">Age</span>
          <span />
        </div>
        {!filtered.length ? (
          <div className="empty-state">
            <div className="empty-state__icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>
            </div>
            <div className="empty-state__title">No deployments</div>
            <div className="empty-state__sub">No deploys match the current filters. Try adjusting your search or filter criteria.</div>
          </div>
        ) : (
          <div className="divide-y divide-white/[0.04]">
            {filtered.map((deploy) => {
              const isActive = ACTIVE_STATUSES.has(deploy.status as Parameters<typeof ACTIVE_STATUSES.has>[0]);
              const context = envMap.get(deployKey(deploy.app, deploy.env, deploy.branch));
              return (
                <div key={deploy.id} className={cn("grid grid-cols-[auto_1fr_auto_auto_auto_auto] gap-3 items-center px-4 py-3 hover:bg-white/[0.02] transition-colors", isActive && "border-l-2 border-l-amber-400")}>
                  <StatusDot status={deploy.status} />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-white">
                        {deploy.build_number ? `#${deploy.build_number}` : deploy.id.slice(0, 8)}
                      </span>
                      <span className={cn("text-[10px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded border", statusBadgeClass(deploy.status))}>
                        {deployPhaseText(deploy)}
                      </span>
                      {isActive && <span className="text-[10px] text-amber-400 animate-pulse">live</span>}
                    </div>
                    <div className="text-xs text-white/35 flex gap-2 flex-wrap mt-0.5">
                      {deploy.commit_sha && <span className="font-mono">{formatCommitSHA(deploy.commit_sha)}</span>}
                      {deploy.commit_message && <span className="truncate max-w-[260px]">{deploy.commit_message}</span>}
                      {context && <span>{engineLabel(context.engine ?? "docker")}</span>}
                    </div>
                  </div>
                  <div className="text-xs text-white/40 text-right shrink-0">{deploy.env}<span className="text-white/20">/</span>{deploy.branch}</div>
                  <div className="text-xs text-white/40 text-right font-mono shrink-0">{deployDurationLabel(deploy)}</div>
                  <div className="text-xs text-white/30 text-right shrink-0">{timeAgo(deploy.created_at)} ago</div>
                  <div className="flex items-center gap-1 shrink-0">
                    {isActive && onCancelDeploy && (
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); onCancelDeploy(deploy.id); }}
                        className="text-xs text-red-400/60 hover:text-red-400 transition-colors px-2 py-1 rounded hover:bg-red-500/[0.08] shrink-0"
                      >
                        Cancel
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => onOpenDeploy(deploy)}
                      className="text-xs text-white/40 hover:text-white transition-colors px-2 py-1 rounded hover:bg-white/[0.06] shrink-0"
                    >
                      Inspect
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
      <div className="text-xs text-white/25 text-right">{filtered.length} of {deploys.length} deployments</div>
    </div>
  );
}

function FilterPill({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn("text-[11px] font-medium px-2.5 py-1 rounded-full border transition-colors", active ? "bg-white/[0.08] border-white/20 text-white" : "border-white/[0.06] text-white/40 hover:text-white/70 hover:border-white/10")}
    >
      {children}
    </button>
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
