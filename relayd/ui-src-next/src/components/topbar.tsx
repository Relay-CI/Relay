"use client";

import { useMemo, useState } from "react";
import { Menu, Search } from "lucide-react";
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
  onToggleSidebar?: () => void;
  onCreateProject?: () => void;
  isOwner?: boolean;
  onAdminClick?: () => void;
  onAppearanceClick?: () => void;
  activeTab?: string;
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
  onToggleSidebar,
  onCreateProject,
  isOwner,
  onAdminClick,
  onAppearanceClick,
  activeTab,
}: TopbarProps) {
  const initials = currentUser?.username
    ? currentUser.username.slice(0, 2).toUpperCase()
    : "RL";
  const [query, setQuery] = useState("");
  const searchMatches = useMemo(() => {
    const value = query.trim().toLowerCase();
    if (!value) return [];
    return projects
      .filter((project) => project.name.toLowerCase().includes(value))
      .slice(0, 5);
  }, [projects, query]);

  return (
    <header className="relay-topbar flex items-center h-16 px-4 md:px-5 gap-3 shrink-0 z-40">
      {/* Brand */}
      <div className="flex items-center gap-2 shrink-0">
        {onToggleSidebar && (
          <button
            type="button"
            onClick={onToggleSidebar}
            className="md:hidden p-2 text-slate-500 hover:text-slate-950 transition-colors rounded-lg hover:bg-white -ml-1"
            aria-label="Toggle navigation"
          >
            <Menu size={17} strokeWidth={2} />
          </button>
        )}
        <div className="relay-brand-tile">
          <RelayMark className="w-5 h-5 text-white" />
        </div>
        <div className="hidden sm:block">
          <div className="text-sm font-semibold text-slate-950 leading-none">Relay</div>
          <div className="text-[10px] text-slate-400 leading-none mt-0.5">Admin</div>
        </div>
      </div>

      {/* Divider */}
      <div className="hidden sm:block w-px h-6 bg-slate-200 shrink-0" />

      {/* Project selector */}
      <ProjectSelector
        projects={projects}
        selected={selectedProjectName}
        onSelect={onSelectProject}
        onCreateNew={onCreateProject}
      />

      <div className="relative hidden lg:flex items-center gap-2 flex-1 max-w-md ml-auto rounded-xl border border-slate-200 bg-white/80 px-3 py-2 shadow-sm">
        <Search size={14} className="text-slate-400 shrink-0" />
        <input
          className="min-w-0 flex-1 bg-transparent text-xs text-slate-700 placeholder:text-slate-400 outline-none"
          placeholder="Search projects, deploys, logs..."
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && searchMatches[0]) {
              onSelectProject(searchMatches[0].name);
              setQuery("");
            }
            if (event.key === "Escape") {
              setQuery("");
            }
          }}
        />
        <span className="rounded-md border border-slate-200 px-1.5 py-0.5 text-[10px] text-slate-400">Enter</span>
        {searchMatches.length > 0 && (
          <div className="absolute left-0 right-0 top-full mt-2 rounded-2xl border border-slate-200 bg-white p-1 shadow-2xl">
            {searchMatches.map((project) => (
              <button
                key={project.name}
                type="button"
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => {
                  onSelectProject(project.name);
                  setQuery("");
                }}
                className="w-full rounded-xl px-3 py-2 text-left text-xs font-medium text-slate-700 hover:bg-slate-50"
              >
                {project.name}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Live indicator */}
      <div className={cn(
        "hidden sm:flex items-center gap-1.5 text-xs px-2.5 py-1.5 rounded-xl border shadow-sm",
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
        className="relay-topbar-action"
      >
        {refreshing ? "Refreshing…" : "Refresh"}
      </button>

      {/* Appearance */}
      {onAppearanceClick && (
        <button
          type="button"
          onClick={onAppearanceClick}
          title="Appearance"
          className={cn(
            "relay-icon-btn",
            activeTab === "appearance"
              ? "relay-icon-btn--active"
              : ""
          )}
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            <line x1="2" y1="12" x2="22" y2="12"/>
          </svg>
        </button>
      )}

      {/* Admin */}
      {isOwner && onAdminClick && (
        <button
          type="button"
          onClick={onAdminClick}
          title="Server admin"
          className={cn(
            "relay-icon-btn",
            activeTab === "admin"
              ? "relay-icon-btn--active"
              : ""
          )}
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          </svg>
        </button>
      )}

      {/* User */}
      {currentUser && (
        <div className="flex items-center gap-2 pl-2 border-l border-slate-200">
          <div className="w-8 h-8 rounded-xl bg-slate-950 flex items-center justify-center text-[10px] font-bold text-white shadow-sm">
            {initials}
          </div>
          <div className="hidden sm:block">
            <div className="text-xs font-medium text-slate-950 leading-none">{currentUser.username}</div>
            <div className="text-[10px] text-slate-400 leading-none mt-0.5">{currentUser.role}</div>
          </div>
          <button
            type="button"
            onClick={onLogout}
            className="relay-topbar-action"
          >
            Sign out
          </button>
        </div>
      )}
    </header>
  );
}
