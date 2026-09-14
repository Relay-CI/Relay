"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import { ServerSettingsPage } from "@/components/pages/server-settings";
import { UsersPage } from "@/components/pages/users";
import { AuditPage } from "@/components/pages/audit";
import { PluginsPage } from "@/components/pages/plugins";
import { OperationsPage } from "@/components/pages/operations";

type CurrentUser = { username: string; role: string } | null;

interface AdminPanelProps {
  currentUser: CurrentUser;
}

const ADMIN_TABS = [
  {
    id: "server",
    label: "Server config",
    description: "Domains, TLS and retention",
    icon: (
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <rect x="2" y="2" width="20" height="8" rx="2"/>
        <rect x="2" y="14" width="20" height="8" rx="2"/>
        <line x1="6" y1="6" x2="6.01" y2="6"/>
        <line x1="6" y1="18" x2="6.01" y2="18"/>
      </svg>
    ),
  },
  {
    id: "operations",
    label: "Operations",
    description: "Health and resource pressure",
    icon: (
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M4 19h16"/>
        <path d="M6 15l3-4 3 2 5-7 1 1"/>
        <circle cx="6" cy="15" r="1"/>
        <circle cx="12" cy="13" r="1"/>
        <circle cx="17" cy="6" r="1"/>
      </svg>
    ),
  },
  {
    id: "users",
    label: "Users",
    description: "Roles and lane access",
    icon: (
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
        <circle cx="9" cy="7" r="4"/>
        <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
        <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
      </svg>
    ),
  },
  {
    id: "audit",
    label: "Audit log",
    description: "Review administrative activity",
    icon: (
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
        <polyline points="14 2 14 8 20 8"/>
        <line x1="16" y1="13" x2="8" y2="13"/>
        <line x1="16" y1="17" x2="8" y2="17"/>
        <polyline points="10 9 9 9 8 9"/>
      </svg>
    ),
  },
  {
    id: "plugins",
    label: "Plugins",
    description: "Manage buildpack extensions",
    icon: (
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M12 2l3 3-2 2 2 2-3 3-2-2-2 2-3-3 2-2-2-2 3-3 2 2 2-2z"/>
        <path d="M14 14l8 8"/>
        <path d="M17 21l4-4"/>
      </svg>
    ),
  },
];

export function AdminPanel({ currentUser }: AdminPanelProps) {
  const [subTab, setSubTab] = useState("server");

  if (currentUser?.role !== "owner") {
    return (
      <div className="min-h-[24rem] grid place-items-center">
        <div className="max-w-md rounded-2xl border border-white/[0.08] bg-white/[0.025] p-7 text-center">
          <div className="eyebrow mb-2">Restricted area</div>
          <h1 className="text-xl font-semibold text-white">Owner access required</h1>
          <p className="mt-2 text-sm leading-6 text-white/40">Server configuration and account controls are available only to Relay owners.</p>
        </div>
      </div>
    );
  }

  const active = ADMIN_TABS.find((tab) => tab.id === subTab) ?? ADMIN_TABS[0];

  return (
    <section className="admin-workspace space-y-5">
      <header className="flex flex-col gap-4 border-b border-white/[0.07] pb-5 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div className="eyebrow mb-1">Administration</div>
          <h1 className="text-2xl font-semibold tracking-[-0.035em] text-white">Server control</h1>
          <p className="mt-1 max-w-2xl text-sm leading-6 text-white/40">Configure Relay, inspect host health, and control who can deploy.</p>
        </div>
        <div className="inline-flex w-fit items-center gap-2 rounded-xl border border-emerald-500/20 bg-emerald-500/[0.07] px-3 py-2 text-xs font-medium text-emerald-500">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
          Owner session
        </div>
      </header>

      <div className="grid items-start gap-5 lg:grid-cols-[15rem_minmax(0,1fr)]">
        <aside className="rounded-2xl border border-white/[0.08] bg-white/[0.025] p-2 lg:sticky lg:top-0" aria-label="Admin sections">
          <div className="px-3 pb-2 pt-2 text-[10px] font-semibold uppercase tracking-[0.14em] text-white/25">Control groups</div>
          <nav className="grid grid-cols-1 gap-1 sm:grid-cols-2 lg:grid-cols-1">
            {ADMIN_TABS.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => setSubTab(tab.id)}
                aria-current={subTab === tab.id ? "page" : undefined}
                className={cn(
                  "group flex min-w-0 items-start gap-3 rounded-xl px-3 py-3 text-left transition-[background-color,color,transform] active:translate-y-px",
                  subTab === tab.id
                    ? "bg-white/[0.09] text-white shadow-sm"
                    : "text-white/45 hover:bg-white/[0.045] hover:text-white/80",
                )}
              >
                <span className={cn("mt-0.5 grid h-7 w-7 shrink-0 place-items-center rounded-lg border", subTab === tab.id ? "border-relay-accent/30 bg-relay-accent/10 text-relay-accent-bright" : "border-white/[0.07] bg-white/[0.025]")}>{tab.icon}</span>
                <span className="min-w-0">
                  <span className="block text-sm font-semibold leading-5">{tab.label}</span>
                  <span className="block truncate text-[11px] leading-4 text-white/30">{tab.description}</span>
                </span>
              </button>
            ))}
          </nav>
        </aside>

        <div className="min-w-0 rounded-2xl border border-white/[0.08] bg-white/[0.018] p-4 sm:p-6">
          <div className="mb-5 flex items-center gap-2 border-b border-white/[0.06] pb-3 lg:hidden">
            <span className="text-relay-accent-bright">{active.icon}</span>
            <span className="text-sm font-semibold text-white">{active.label}</span>
          </div>
          {subTab === "server" && <ServerSettingsPage currentUser={currentUser} />}
          {subTab === "operations" && <OperationsPage />}
          {subTab === "users" && <UsersPage currentUser={currentUser} />}
          {subTab === "audit" && <AuditPage />}
          {subTab === "plugins" && <PluginsPage />}
        </div>
      </div>
    </section>
  );
}
