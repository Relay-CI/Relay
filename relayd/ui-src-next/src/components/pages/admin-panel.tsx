"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import { ServerSettingsPage } from "@/components/pages/server-settings";
import { UsersPage } from "@/components/pages/users";
import { AuditPage } from "@/components/pages/audit";

type CurrentUser = { username: string; role: string } | null;

interface AdminPanelProps {
  currentUser: CurrentUser;
}

const ADMIN_TABS = [
  {
    id: "server",
    label: "Server config",
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
    id: "users",
    label: "Users",
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
];

export function AdminPanel({ currentUser }: AdminPanelProps) {
  const [subTab, setSubTab] = useState("server");

  if (currentUser?.role !== "owner") {
    return (
      <div className="flex items-center justify-center h-full text-white/30 text-sm">
        Owner access required.
      </div>
    );
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div>
        <div className="eyebrow mb-0.5">Administration</div>
        <h1 className="text-xl font-semibold text-white">Server admin</h1>
        <p className="text-sm text-white/40 mt-1">
          Server-wide configuration, user management, and audit trail.
        </p>
      </div>

      {/* Sub-navigation */}
      <div className="flex gap-1 border-b border-white/[0.06] pb-0 -mb-0">
        {ADMIN_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setSubTab(tab.id)}
            className={cn(
              "flex items-center gap-1.5 px-3 py-2 text-sm font-medium border-b-2 transition-colors -mb-px",
              subTab === tab.id
                ? "border-relay-accent text-white"
                : "border-transparent text-white/40 hover:text-white/70"
            )}
          >
            <span className="shrink-0">{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Sub-page content */}
      <div>
        {subTab === "server" && <ServerSettingsPage currentUser={currentUser} />}
        {subTab === "users" && <UsersPage currentUser={currentUser} />}
        {subTab === "audit" && <AuditPage />}
      </div>
    </div>
  );
}
