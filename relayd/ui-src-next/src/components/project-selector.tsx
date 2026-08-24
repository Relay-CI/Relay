"use client";

import { useMemo, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { timeAgo, repoProviderInfo, projectRepoURL } from "@/lib/relay-utils";
import { RelayMark } from "@/components/relay-mark";
import type { NormalizedProject } from "@/lib/relay-utils";

interface ProjectSelectorProps {
  projects: NormalizedProject[];
  selected: string;
  onSelect: (name: string) => void;
  onCreateNew?: () => void;
}

export function ProjectSelector({ projects, selected, onSelect, onCreateNew }: ProjectSelectorProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const ref = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return projects;
    return projects.filter((p) => p.name.toLowerCase().includes(q));
  }, [projects, query]);

  const current = projects.find((p) => p.name === selected);

  return (
    <div
      className="relative"
      ref={ref}
      onBlur={(e) => {
        if (!ref.current?.contains(e.relatedTarget as Node)) setOpen(false);
      }}
    >
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2.5 h-10 px-3 rounded-xl border border-slate-200 bg-white/80 hover:bg-white transition-colors min-w-[160px] max-w-[240px] shadow-sm"
      >
        <RelayMark className="w-5 h-5 text-relay-accent shrink-0" />
        <div className="flex flex-col items-start min-w-0">
          <span className="eyebrow leading-none mb-0.5">Project</span>
          <span className="text-sm font-semibold text-slate-950 truncate leading-none">
            {current?.name ?? "Select project"}
          </span>
        </div>
        {current && (
          <span className="ml-auto text-xs text-slate-400 shrink-0">{current.envs.length}L</span>
        )}
        <svg className="w-3 h-3 text-slate-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>

      {open && (
        <div className="absolute top-full left-0 mt-2 w-72 z-50 rounded-2xl border border-slate-200 bg-white shadow-2xl overflow-hidden">
          <div className="p-2 border-b border-slate-100">
            <input
              type="text"
              className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-1.5 text-sm text-slate-950 placeholder:text-slate-400 outline-none focus:border-relay-accent/50"
              placeholder="Search projects…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              autoFocus
            />
          </div>
          <div className="max-h-72 overflow-y-auto">
            {filtered.map((project) => {
              const repoInfo = repoProviderInfo(projectRepoURL(project));
              return (
                <button
                  key={project.name}
                  type="button"
                  className={cn(
                    "w-full text-left px-3 py-2.5 hover:bg-slate-50 transition-colors",
                    project.name === selected && "bg-cyan-50 border-l-2 border-relay-accent",
                  )}
                  onClick={() => { onSelect(project.name); setOpen(false); setQuery(""); }}
                >
                  <div className="font-semibold text-sm text-slate-950">{project.name}</div>
                  <div className="text-xs text-slate-500 mt-0.5 flex gap-2">
                    <span>{project.envs.length} lanes</span>
                    <span>·</span>
                    <span>{repoInfo.label}</span>
                    {project.latestDeployAt ? (
                      <>
                        <span>·</span>
                        <span>{timeAgo(new Date(project.latestDeployAt).toISOString())} ago</span>
                      </>
                    ) : null}
                  </div>
                </button>
              );
            })}
            {!filtered.length && (
              <div className="px-3 py-4 text-sm text-slate-500 text-center">No projects matched</div>
            )}
          </div>
          {onCreateNew && (
            <button
              type="button"
              onClick={() => { onCreateNew(); setOpen(false); setQuery(""); }}
              className="w-full flex items-center gap-2 px-3 py-2.5 text-xs text-relay-accent hover:bg-cyan-50 border-t border-slate-100 transition-colors"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
              </svg>
              New project…
            </button>
          )}
        </div>
      )}
    </div>
  );
}
