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
}

export function ProjectSelector({ projects, selected, onSelect }: ProjectSelectorProps) {
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
        className="flex items-center gap-2.5 h-10 px-3 rounded-md border border-white/[0.08] bg-white/[0.04] hover:bg-white/[0.08] transition-colors min-w-[160px] max-w-[240px]"
      >
        <RelayMark className="w-5 h-5 text-relay-accent shrink-0" />
        <div className="flex flex-col items-start min-w-0">
          <span className="eyebrow leading-none mb-0.5">Project</span>
          <span className="text-sm font-semibold text-white truncate leading-none">
            {current?.name ?? "Select project"}
          </span>
        </div>
        {current && (
          <span className="ml-auto text-xs text-white/40 shrink-0">{current.envs.length}L</span>
        )}
        <svg className="w-3 h-3 text-white/40 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>

      {open && (
        <div className="absolute top-full left-0 mt-1 w-72 z-50 rounded-lg border border-white/[0.08] bg-zinc-950 shadow-2xl overflow-hidden">
          <div className="p-2 border-b border-white/[0.06]">
            <input
              type="text"
              className="w-full bg-white/[0.04] border border-white/[0.08] rounded px-3 py-1.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50"
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
                    "w-full text-left px-3 py-2.5 hover:bg-white/[0.06] transition-colors",
                    project.name === selected && "bg-relay-accent/10 border-l-2 border-relay-accent",
                  )}
                  onClick={() => { onSelect(project.name); setOpen(false); setQuery(""); }}
                >
                  <div className="font-semibold text-sm text-white">{project.name}</div>
                  <div className="text-xs text-white/40 mt-0.5 flex gap-2">
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
              <div className="px-3 py-4 text-sm text-white/40 text-center">No projects matched</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
