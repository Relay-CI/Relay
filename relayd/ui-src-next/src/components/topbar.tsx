"use client";

import { cn } from "@/lib/utils";
import { RelayMark } from "@/components/relay-mark";
import { ProjectSelector } from "@/components/project-selector";
import type { NormalizedProject } from "@/lib/relay-utils";
import type { SessionInfo } from "@/lib/api";

type CurrentUser = { username: string; role: string } | null;

interface TopbarProps {
  projects: NormalizedProject[];
  selectedProjectName: string;
  onSelectProject: (name: string) => void;
  currentUser: CurrentUser;
  onLogout: () => void;
  isLive: boolean;
  onRefresh: () => void;
  refreshing: boolean;
}

export function Topbar({
  projects,
  selectedProjectName,
  onSelectProject,
  currentUser,
  onLogout,
  isLive,
  onRefresh,
  refreshing,
}: TopbarProps) {
  const initials = currentUser?.username
    ? currentUser.username.slice(0, 2).toUpperCase()
    : "RL";

  return (
    <header className="flex items-center h-12 px-4 gap-4 border-b border-white/[0.06] bg-zinc-950 shrink-0 z-40">
      {/* Brand */}
      <div className="flex items-center gap-2 shrink-0">
        <RelayMark className="w-6 h-6 text-white" />
        <div className="hidden sm:block">
          <div className="text-sm font-semibold text-white leading-none">Relayd</div>
          <div className="text-[10px] text-white/30 leading-none mt-0.5">Control Room</div>
        </div>
      </div>

      {/* Divider */}
      <div className="w-px h-5 bg-white/[0.08] shrink-0" />

      {/* Project selector */}
      <ProjectSelector
        projects={projects}
        selected={selectedProjectName}
        onSelect={onSelectProject}
      />

      {/* Spacer */}
      <div className="flex-1" />

      {/* Live indicator */}
      <div className={cn(
        "hidden sm:flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border",
        isLive
          ? "border-amber-500/30 bg-amber-500/10 text-amber-400"
          : "border-emerald-500/20 bg-emerald-500/[0.08] text-emerald-500",
      )}>
        <span className={cn(
          "w-1.5 h-1.5 rounded-full",
          isLive ? "bg-amber-400 animate-pulse" : "bg-emerald-500",
        )} />
        {isLive ? "Deploying" : "Live"}
      </div>

      {/* Refresh */}
      <button
        type="button"
        onClick={onRefresh}
        disabled={refreshing}
        className="text-xs text-white/50 hover:text-white transition-colors px-2 py-1 rounded hover:bg-white/[0.06]"
      >
        {refreshing ? "Refreshing…" : "Refresh"}
      </button>

      {/* User */}
      {currentUser && (
        <div className="flex items-center gap-2 pl-2 border-l border-white/[0.08]">
          <div className="w-6 h-6 rounded-full bg-relay-accent/20 border border-relay-accent/30 flex items-center justify-center text-[10px] font-bold text-relay-accent">
            {initials}
          </div>
          <div className="hidden sm:block">
            <div className="text-xs font-medium text-white leading-none">{currentUser.username}</div>
            <div className="text-[10px] text-white/30 leading-none mt-0.5">{currentUser.role}</div>
          </div>
          <button
            type="button"
            onClick={onLogout}
            className="text-xs text-white/40 hover:text-white transition-colors px-2 py-1 rounded hover:bg-white/[0.06]"
          >
            Sign out
          </button>
        </div>
      )}
    </header>
  );
}
