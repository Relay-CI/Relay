"use client";

import { useState, useEffect } from "react";
import { getAuditLog } from "@/lib/api";
import type { AuditEntry } from "@/lib/api";

export function AuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);

  function load() {
    setLoading(true);
    getAuditLog(100)
      .then((data) => setEntries(data ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  function fmtTime(ts: string) {
    if (!ts) return "";
    return new Date(ts).toLocaleString();
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <div className="eyebrow mb-0.5">Security</div>
          <h1 className="text-xl font-semibold text-white">Activity log</h1>
        </div>
        <button
          type="button"
          onClick={load}
          disabled={loading}
          className="text-xs border border-white/[0.08] text-white/50 hover:text-white hover:border-white/20 px-3 py-1.5 rounded transition-colors disabled:opacity-40"
        >
          Refresh
        </button>
      </div>

      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl overflow-hidden">
        <div className="grid grid-cols-[1fr_1fr_auto_1fr_auto] text-[10px] uppercase tracking-wider text-white/30 font-semibold border-b border-white/[0.06] px-5 py-2.5 gap-4">
          <span>Action</span>
          <span>Target</span>
          <span>Actor</span>
          <span>Detail</span>
          <span className="text-right">Time</span>
        </div>

        {loading && (
          <div className="text-sm text-white/30 text-center py-10">Loading…</div>
        )}

        {!loading && !entries.length && (
          <div className="empty-state">
            <div className="empty-state__icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            </div>
            <div className="empty-state__title">No activity yet</div>
            <div className="empty-state__sub">Audit events will appear here as users perform actions.</div>
          </div>
        )}

        {!loading && entries.length > 0 && (
          <div className="divide-y divide-white/[0.04]">
            {entries.map((entry) => (
              <div key={entry.id} className="grid grid-cols-[1fr_1fr_auto_1fr_auto] gap-4 items-start px-5 py-3">
                <div className="text-sm font-mono text-white truncate">{entry.action}</div>
                <div className="text-sm text-white/60 truncate">{entry.target}</div>
                <div className="text-xs text-white/40 shrink-0">{entry.actor ?? "—"}</div>
                <div className="text-xs text-white/40 truncate">{entry.detail ?? ""}</div>
                <div className="text-xs text-white/30 shrink-0 text-right">{fmtTime(entry.ts ?? entry.created_at)}</div>
              </div>
            ))}
          </div>
        )}
      </div>
      {!loading && entries.length > 0 && (
        <div className="text-xs text-white/25 text-right">{entries.length} entries</div>
      )}
    </div>
  );
}
