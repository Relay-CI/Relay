"use client";

import Link from "next/link";
import { cn } from "@/lib/utils";
import {
  computeConfiguredURL,
  computePreviewURL,
  deployKey,
  engineLabel,
  formatPercent,
  repoProviderInfo,
  projectRepoURL,
  rolloutStrategy,
  trafficModeLabel,
  type NormalizedProject,
  type ProjectStats,
} from "@/lib/relay-utils";
import type { EnvInfo } from "@/lib/api";

interface SelectedEnvWithMeta extends EnvInfo {
  // EnvInfo already has app, env, branch, latestDeploy, previewURL
}


interface SidebarProps {
  activeTab: string;
  onTabChange: (tab: string) => void;
  isOwner: boolean;
  selectedProject: NormalizedProject | null;
  selectedEnv: SelectedEnvWithMeta | null;
  envOptions: SelectedEnvWithMeta[];
  selectedEnvKey: string;
  onEnvChange: (key: string) => void;
  projectStats: ProjectStats;
}

const NAV_ITEMS = [
  { id: "overview", label: "Overview", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
  )},
  { id: "deployments", label: "Deployments", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>
  )},
  { id: "logs", label: "Build Logs", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
  )},
  { id: "runtime-logs", label: "Runtime Logs", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
  )},
  { id: "settings", label: "Environment", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="3"/><path d="M19.07 4.93a10 10 0 1 0 0 14.14M19.07 4.93L12 12"/></svg>
  )},
  { id: "analytics", label: "Analytics", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
  )},
];

const OWNER_ITEMS = [
  { id: "server", label: "Server Settings", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
  )},
  { id: "users", label: "Users", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
  )},
  { id: "audit", label: "Audit Log", icon: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
  )},
];

export function Sidebar({
  activeTab,
  onTabChange,
  isOwner,
  selectedProject,
  selectedEnv,
  envOptions,
  selectedEnvKey,
  onEnvChange,
  projectStats,
}: SidebarProps) {
  const repoInfo = selectedProject ? repoProviderInfo(projectRepoURL(selectedProject)) : null;
  const selectedRoute = selectedEnv
    ? computeConfiguredURL(selectedEnv) || computePreviewURL(selectedEnv, selectedEnv.latestDeploy ?? null) || ""
    : "";
  const allItems = [...NAV_ITEMS, ...(isOwner ? OWNER_ITEMS : [])];

  return (
    <aside className="flex flex-col w-52 shrink-0 border-r border-white/[0.06] bg-zinc-950 overflow-y-auto">
      {/* Project card */}
      <div className="p-3 border-b border-white/[0.06]">
        {selectedProject ? (
          <>
            <div className="flex items-start justify-between mb-1.5">
              <div className="min-w-0">
                <div className="eyebrow mb-0.5">Project</div>
                <div className="text-sm font-semibold text-white truncate">{selectedProject.name}</div>
              </div>
              <span className="text-[10px] text-white/40 border border-white/[0.1] rounded px-1.5 py-0.5 shrink-0 ml-2">
                {selectedProject.envs.length}L
              </span>
            </div>
            {repoInfo && (
              <div className="text-[11px] text-white/40 mb-2">{repoInfo.label}</div>
            )}
            <div className="grid grid-cols-3 gap-1">
              <div className="text-center bg-white/[0.03] rounded p-1.5">
                <div className="text-xs font-semibold text-white">{formatPercent(projectStats.successRate)}</div>
                <div className="text-[10px] text-white/40">success</div>
              </div>
              <div className="text-center bg-white/[0.03] rounded p-1.5">
                <div className="text-xs font-semibold text-white">{projectStats.failures}</div>
                <div className="text-[10px] text-white/40">failures</div>
              </div>
              <div className="text-center bg-white/[0.03] rounded p-1.5">
                <div className="text-xs font-semibold text-white">{projectStats.total}</div>
                <div className="text-[10px] text-white/40">deploys</div>
              </div>
            </div>
          </>
        ) : (
          <div className="text-xs text-white/30 py-1">Select a project above</div>
        )}
      </div>

      {/* Navigation */}
      <div className="px-1.5 py-2 flex-1">
        <div className="text-[10px] uppercase tracking-widest text-white/25 px-2 mb-1">Navigate</div>
        <nav className="flex flex-col gap-0.5">
          {allItems.map(({ id, label, icon }) => (
            <button
              key={id}
              type="button"
              onClick={() => onTabChange(id)}
              className={cn(
                "flex items-center gap-2 px-2 py-1.5 rounded text-sm text-white/60 hover:text-white hover:bg-white/[0.06] transition-colors w-full text-left",
                activeTab === id && "text-white bg-relay-accent/15 border-l-2 border-relay-accent pl-[6px]",
              )}
            >
              <span className="shrink-0 text-inherit">{icon}</span>
              <span>{label}</span>
            </button>
          ))}
        </nav>
      </div>

      {/* Lane selector + runtime card */}
      {envOptions.length > 0 && (
        <div className="p-3 border-t border-white/[0.06]">
          <div className="eyebrow mb-1.5">Lane</div>
          <select
            className="w-full bg-white/[0.04] border border-white/[0.08] rounded px-2 py-1.5 text-xs text-white/80 outline-none focus:border-relay-accent/50 appearance-none cursor-pointer"
            value={selectedEnvKey}
            onChange={(e) => onEnvChange(e.target.value)}
          >
            {envOptions.map((env) => (
              <option key={deployKey(env.app, env.env, env.branch)} value={deployKey(env.app, env.env, env.branch)}>
                {env.env} / {env.branch}
              </option>
            ))}
          </select>

          {selectedEnv && (
            <div className="mt-2">
              <div className="text-[11px] text-white/30 truncate mb-1">
                {selectedRoute || "No public route"}
              </div>
              <div className="flex flex-wrap gap-1">
                <span className="text-[10px] bg-white/[0.06] px-1.5 py-0.5 rounded text-white/50">
                  {trafficModeLabel(selectedEnv.traffic_mode ?? "")}
                </span>
                <span className="text-[10px] bg-white/[0.06] px-1.5 py-0.5 rounded text-white/50">
                  {rolloutStrategy(selectedEnv)}
                </span>
              </div>
            </div>
          )}
        </div>
      )}
    </aside>
  );
}
