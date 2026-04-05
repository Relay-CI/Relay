"use client";

import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import { getAnalytics } from "@/lib/api";
import { formatPercent } from "@/lib/relay-utils";
import type { EnvInfo } from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
}

interface AnalyticsPageProps {
  selectedEnv: SelectedEnvMeta | null;
}

type Period = "24h" | "7d" | "30d";

interface AnalyticsData {
  total_requests: number;
  by_country: { code: string; name: string; count: number }[];
  by_status: { status: number; count: number }[];
  by_hour: { ts: number; count: number }[];
  by_host: { host: string; count: number }[];
}

function countryFlag(code: string) {
  if (!code || code.length !== 2) return "";
  const cp = [...code.toUpperCase()].map((c) => 0x1f1e6 + c.charCodeAt(0) - 65);
  return String.fromCodePoint(...cp);
}

export function AnalyticsPage({ selectedEnv }: AnalyticsPageProps) {
  const [period, setPeriod] = useState<Period>("7d");
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    const params = new URLSearchParams({ period });
    if (selectedEnv?.app) params.set("app", selectedEnv.app);
    getAnalytics(selectedEnv?.app, period)
    return () => { cancelled = true; };
  }, [period, selectedEnv?.app]);

  const total = data?.total_requests ?? 0;
  const byCountry = data?.by_country ?? [];
  const byStatus = data?.by_status ?? [];
  const byHour = data?.by_hour ?? [];
  const byHost = data?.by_host ?? [];

  const maxCountry = byCountry.reduce((m, c) => Math.max(m, c.count), 1);
  const maxHour = byHour.reduce((m, h) => Math.max(m, h.count), 1);
  const success2xx = byStatus.filter((s) => s.status >= 200 && s.status < 300).reduce((sum, s) => sum + s.count, 0);
  const redirect3xx = byStatus.filter((s) => s.status >= 300 && s.status < 400).reduce((sum, s) => sum + s.count, 0);
  const error4xx = byStatus.filter((s) => s.status >= 400 && s.status < 500).reduce((sum, s) => sum + s.count, 0);
  const error5xx = byStatus.filter((s) => s.status >= 500).reduce((sum, s) => sum + s.count, 0);
  const successRate = total > 0 ? (success2xx / total) : 0;

  function fmtBucket(ts: number) {
    if (!ts) return "";
    const d = new Date(ts * 1000);
    return period === "30d"
      ? d.toLocaleDateString([], { month: "short", day: "numeric" })
      : d.toLocaleTimeString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <div className="eyebrow mb-0.5">Traffic</div>
          <h1 className="text-xl font-semibold text-white">Analytics</h1>
        </div>
        <div className="flex gap-1">
          {(["24h", "7d", "30d"] as Period[]).map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => setPeriod(p)}
              className={cn("text-xs px-3 py-1.5 rounded border transition-colors", period === p ? "bg-white/[0.08] border-white/20 text-white" : "border-white/[0.06] text-white/40 hover:text-white/60")}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {loading && <div className="text-sm text-white/30 py-8 text-center">Loading analytics…</div>}
      {error && <div className="bg-red-500/10 border border-red-500/30 text-red-400 text-sm rounded-lg px-4 py-3">{error}</div>}

      {!loading && !error && total === 0 && (
        <div className="text-center py-12 text-white/30 text-sm">
          No traffic data yet for this period.<br />
          <span className="text-xs text-white/20 mt-1 block">Traffic is recorded from the Caddy access log once the global proxy is active.</span>
        </div>
      )}

      {!loading && !error && total > 0 && (
        <>
          {/* Metric cards */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <MetricCard label="Requests" value={total.toLocaleString()} meta={`last ${period}`} />
            <MetricCard label="Countries" value={String(byCountry.length)} meta="unique" />
            <MetricCard label="Success rate" value={formatPercent(successRate)} meta="2xx responses" />
            <MetricCard label="Errors" value={(error4xx + error5xx).toLocaleString()} meta="4xx + 5xx" />
          </div>

          {/* Charts grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            <Card title="Top Countries">
              <div className="space-y-2.5">
                {byCountry.slice(0, 12).map((c) => (
                  <BarRow key={c.code} label={<><span className="mr-1.5">{countryFlag(c.code)}</span>{c.name}</>} count={c.count} max={maxCountry} />
                ))}
              </div>
            </Card>

            <Card title="Status Breakdown">
              <div className="space-y-2.5 mb-5">
                {[
                  { label: "2xx Success", count: success2xx, color: "bg-emerald-500/60" },
                  { label: "3xx Redirect", count: redirect3xx, color: "bg-sky-500/60" },
                  { label: "4xx Client Error", count: error4xx, color: "bg-amber-500/60" },
                  { label: "5xx Server Error", count: error5xx, color: "bg-red-500/60" },
                ].map((row) => (
                  <BarRow key={row.label} label={row.label} count={row.count} max={total} color={row.color} />
                ))}
              </div>

              {byHost.length > 0 && (
                <>
                  <div className="eyebrow mb-2">Top Hosts</div>
                  <div className="space-y-2">
                    {byHost.slice(0, 5).map((h) => (
                      <BarRow key={h.host} label={<span className="font-mono">{h.host}</span>} count={h.count} max={total} />
                    ))}
                  </div>
                </>
              )}
            </Card>
          </div>

          {/* Time series */}
          {byHour.length > 0 && (
            <Card title={`Requests over time — ${period === "30d" ? "daily" : "hourly"} buckets`}>
              <div className="flex items-end gap-0.5 h-24">
                {byHour.map((h) => (
                  <div
                    key={h.ts}
                    title={`${fmtBucket(h.ts)}: ${h.count.toLocaleString()} requests`}
                    className="flex-1 bg-relay-teal/40 hover:bg-relay-teal/60 transition-colors rounded-sm min-h-[2px]"
                    style={{ height: `${Math.max(4, (h.count / maxHour) * 100)}%` }}
                  />
                ))}
              </div>
              <div className="flex justify-between mt-1">
                <span className="text-[10px] text-white/25">{fmtBucket(byHour[0]?.ts)}</span>
                <span className="text-[10px] text-white/25">{fmtBucket(byHour[byHour.length - 1]?.ts)}</span>
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  );
}

function MetricCard({ label, value, meta }: { label: string; value: string; meta?: string }) {
  return (
    <div className="bg-white/[0.03] border border-white/[0.06] rounded-lg p-4">
      <div className="eyebrow mb-1">{label}</div>
      <div className="text-2xl font-semibold text-white">{value}</div>
      {meta && <div className="text-xs text-white/40 mt-1">{meta}</div>}
    </div>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
      <div className="text-sm font-semibold text-white mb-4">{title}</div>
      {children}
    </div>
  );
}

function BarRow({ label, count, max, color = "bg-relay-teal/50" }: { label: React.ReactNode; count: number; max: number; color?: string }) {
  const pct = max > 0 ? (count / max) * 100 : 0;
  return (
    <div className="flex items-center gap-3">
      <div className="text-xs text-white/60 w-32 shrink-0 truncate">{label}</div>
      <div className="flex-1 h-1.5 bg-white/[0.06] rounded-full overflow-hidden">
        <div className={cn("h-full rounded-full", color)} style={{ width: `${pct}%` }} />
      </div>
      <div className="text-xs text-white/40 w-12 text-right shrink-0">{count.toLocaleString()}</div>
    </div>
  );
}
