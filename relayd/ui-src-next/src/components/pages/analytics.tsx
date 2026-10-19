"use client";

import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import { getAnalytics } from "@/lib/api";
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

interface Count {
  key: string;
  count: number;
}

interface AnalyticsData {
  total_requests: number;
  page_views: number;
  visitors: number;
  server_errors: number;
  server_error_rate: number;
  bandwidth_bytes: number;
  bot_requests: number;
  latency: { samples: number; p50_ms: number; p95_ms: number; p99_ms: number };
  by_country: { code: string; name: string; count: number }[];
  by_status: { status: number; count: number }[];
  by_hour: { ts: number; count: number; page_views: number; errors: number }[];
  by_host: { host: string; count: number }[];
  top_pages: Count[];
  top_referrers: Count[];
  error_paths: Count[];
  by_kind: Count[];
  by_variant: { variant: string; requests: number; server_errors: number; page_views: number; visitors: number }[];
}

function countryFlag(code: string) {
  if (!code || code.length !== 2 || code === "??") return "";
  const cp = [...code.toUpperCase()].map((c) => 0x1f1e6 + c.charCodeAt(0) - 65);
  return String.fromCodePoint(...cp);
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size >= 10 || unit === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unit]}`;
}

function compact(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 10_000) return `${Math.round(n / 1000)}k`;
  return n.toLocaleString();
}

const KIND_LABELS: Record<string, string> = {
  page: "Page views",
  asset: "Static assets",
  api: "API calls",
  bot: "Bots & crawlers",
  other: "Other",
  unknown: "Unclassified (pre-edge)",
};

export function AnalyticsPage({ selectedEnv }: AnalyticsPageProps) {
  const [period, setPeriod] = useState<Period>("7d");
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    setLoading(true);
    setError(null);
    const load = (initial: boolean) => {
      getAnalytics(selectedEnv?.app, selectedEnv?.env, selectedEnv?.branch, period)
        .then((result) => {
          if (cancelled) return;
          setData((result as AnalyticsData) ?? null);
          setError(null);
        })
        .catch((err) => {
          if (cancelled) return;
          if (initial) setData(null);
          setError(err instanceof Error ? err.message : "Failed to load analytics");
        })
        .finally(() => {
          if (cancelled) return;
          setLoading(false);
          // Light auto-refresh so a dashboard left open stays current.
          timer = setTimeout(() => load(false), 30_000);
        });
    };
    load(true);
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [period, selectedEnv?.app, selectedEnv?.env, selectedEnv?.branch]);

  const total = data?.total_requests ?? 0;
  const byHour = data?.by_hour ?? [];
  const byCountry = data?.by_country ?? [];
  const byStatus = data?.by_status ?? [];
  const maxHour = byHour.reduce((m, h) => Math.max(m, h.count), 1);
  const maxCountry = byCountry.reduce((m, c) => Math.max(m, c.count), 1);
  const statusCount = (lo: number, hi: number) =>
    byStatus.filter((s) => s.status >= lo && s.status < hi).reduce((sum, s) => sum + s.count, 0);
  const variants = (data?.by_variant ?? []).filter((v) => v.requests > 0);
  const showVariants = variants.length > 1 || (variants.length === 1 && variants[0].variant !== "control");

  function fmtBucket(ts: number) {
    if (!ts) return "";
    const d = new Date(ts * 1000);
    return period === "30d"
      ? d.toLocaleDateString([], { month: "short", day: "numeric" })
      : d.toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="eyebrow mb-0.5">Traffic</div>
          <h1 className="text-xl font-semibold text-white">Analytics</h1>
          <p className="text-sm text-white/40 mt-1">
            {selectedEnv ? `${selectedEnv.app} · ${selectedEnv.env}${selectedEnv.branch !== "main" ? `/${selectedEnv.branch}` : ""}` : "All apps"} — measured at the
            lane&apos;s edge proxy, cookieless (visitors are a daily-rotating hash).
          </p>
        </div>
        <div className="flex gap-1">
          {(["24h", "7d", "30d"] as Period[]).map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => setPeriod(p)}
              className={cn(
                "text-xs px-3 py-1.5 rounded border transition-colors",
                period === p ? "bg-white/[0.08] border-white/20 text-white" : "border-white/[0.06] text-white/40 hover:text-white/60",
              )}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {loading && <div className="text-sm text-white/30 py-8 text-center">Loading analytics…</div>}
      {error && <div className="bg-red-500/10 border border-red-500/30 text-red-400 text-sm rounded-lg px-4 py-3">{error}</div>}

      {!loading && data && total === 0 && (
        <div className="text-center py-12 text-white/30 text-sm">
          No traffic recorded for this period yet.
          <span className="text-xs text-white/20 mt-1 block">
            Requests are counted as they pass through the lane&apos;s edge proxy — open the app&apos;s URL and they appear within a few seconds.
          </span>
        </div>
      )}

      {!loading && data && total > 0 && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
            <MetricCard label="Visitors" value={compact(data.visitors)} meta="unique per day" />
            <MetricCard label="Page views" value={compact(data.page_views)} meta={`${compact(total)} total requests`} />
            <MetricCard
              label="5xx rate"
              value={`${data.server_error_rate.toFixed(data.server_error_rate >= 10 ? 0 : 2)}%`}
              meta={`${data.server_errors.toLocaleString()} server errors`}
              tone={data.server_error_rate >= 1 ? "bad" : "good"}
            />
            <MetricCard label="p50 latency" value={data.latency.samples ? `${data.latency.p50_ms} ms` : "—"} meta="pages + API" />
            <MetricCard
              label="p95 latency"
              value={data.latency.samples ? `${data.latency.p95_ms} ms` : "—"}
              meta={data.latency.samples ? `p99 ${data.latency.p99_ms} ms` : "edge data only"}
              tone={data.latency.p95_ms > 1500 ? "bad" : undefined}
            />
            <MetricCard label="Bandwidth" value={formatBytes(data.bandwidth_bytes)} meta={`${compact(data.bot_requests)} bot requests`} />
          </div>

          {byHour.length > 0 && (
            <Card title={`Traffic over time — ${period === "30d" ? "daily" : "hourly"}`} right={<Legend />}>
              <div className="flex items-end gap-0.5 h-32">
                {byHour.map((h) => (
                  <div
                    key={h.ts}
                    title={`${fmtBucket(h.ts)} — ${h.count.toLocaleString()} requests, ${h.page_views.toLocaleString()} page views, ${h.errors.toLocaleString()} 5xx`}
                    className="relative flex-1 h-full flex flex-col justify-end"
                  >
                    <div className="w-full bg-relay-teal/25 rounded-t-sm min-h-[2px]" style={{ height: `${Math.max(2, (h.count / maxHour) * 100)}%` }}>
                      <div className="w-full bg-relay-teal/70 rounded-t-sm" style={{ height: `${h.count ? (h.page_views / h.count) * 100 : 0}%` }} />
                    </div>
                    {h.errors > 0 && <div className="absolute bottom-0 w-full h-[3px] bg-red-500/80" />}
                  </div>
                ))}
              </div>
              <div className="flex justify-between mt-1">
                <span className="text-[10px] text-white/25">{fmtBucket(byHour[0]?.ts)}</span>
                <span className="text-[10px] text-white/25">{fmtBucket(byHour[byHour.length - 1]?.ts)}</span>
              </div>
            </Card>
          )}

          {showVariants && (
            <Card title="By variant">
              <div className="overflow-x-auto">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="text-left text-white/35">
                      <th className="font-normal pb-2">Variant</th>
                      <th className="font-normal pb-2 text-right">Visitors</th>
                      <th className="font-normal pb-2 text-right">Page views</th>
                      <th className="font-normal pb-2 text-right">Requests</th>
                      <th className="font-normal pb-2 text-right">5xx rate</th>
                    </tr>
                  </thead>
                  <tbody>
                    {variants.map((v) => {
                      const rate = v.requests ? (v.server_errors / v.requests) * 100 : 0;
                      return (
                        <tr key={v.variant} className="border-t border-white/[0.05]">
                          <td className="py-2 font-mono text-white">{v.variant}</td>
                          <td className="py-2 text-right text-white/70">{v.visitors.toLocaleString()}</td>
                          <td className="py-2 text-right text-white/70">{v.page_views.toLocaleString()}</td>
                          <td className="py-2 text-right text-white/70">{v.requests.toLocaleString()}</td>
                          <td className={cn("py-2 text-right", rate >= 1 ? "text-amber-300" : "text-white/70")}>{rate.toFixed(2)}%</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </Card>
          )}

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            <Card title="Top pages">
              <CountList items={data.top_pages} empty="No page views recorded yet." mono />
            </Card>
            <Card title="Top referrers">
              <CountList items={data.top_referrers} empty="No external referrers yet — direct traffic only." />
            </Card>
            <Card title="Top countries">
              <div className="space-y-2.5">
                {byCountry.slice(0, 12).map((c) => (
                  <BarRow key={c.code} label={<><span className="mr-1.5">{countryFlag(c.code)}</span>{c.name || c.code}</>} count={c.count} max={maxCountry} />
                ))}
              </div>
            </Card>
            <Card title="Responses">
              <div className="space-y-2.5 mb-5">
                {[
                  { label: "2xx Success", count: statusCount(200, 300), color: "bg-emerald-500/60" },
                  { label: "3xx Redirect", count: statusCount(300, 400), color: "bg-sky-500/60" },
                  { label: "4xx Client error", count: statusCount(400, 500), color: "bg-amber-500/60" },
                  { label: "5xx Server error", count: statusCount(500, 600), color: "bg-red-500/60" },
                ].map((row) => (
                  <BarRow key={row.label} label={row.label} count={row.count} max={total} color={row.color} />
                ))}
              </div>
              {data.by_kind.length > 0 && (
                <>
                  <div className="eyebrow mb-2">Request types</div>
                  <div className="space-y-2">
                    {data.by_kind.map((k) => (
                      <BarRow key={k.key} label={KIND_LABELS[k.key] ?? k.key} count={k.count} max={total} />
                    ))}
                  </div>
                </>
              )}
            </Card>
          </div>

          {data.error_paths.length > 0 && (
            <Card title="Paths returning 5xx">
              <CountList items={data.error_paths} mono tone="bad" />
            </Card>
          )}

          {!selectedEnv && data.by_host.length > 0 && (
            <Card title="Top hosts">
              <CountList items={data.by_host.map((h) => ({ key: h.host, count: h.count }))} mono />
            </Card>
          )}
        </>
      )}
    </div>
  );
}

function Legend() {
  return (
    <div className="flex items-center gap-3 text-[10px] text-white/40">
      <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-relay-teal/70" />page views</span>
      <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-relay-teal/25" />all requests</span>
      <span className="flex items-center gap-1"><span className="w-2 h-0.5 bg-red-500/80" />5xx</span>
    </div>
  );
}

function MetricCard({ label, value, meta, tone }: { label: string; value: string; meta?: string; tone?: "good" | "bad" }) {
  return (
    <div className="bg-white/[0.03] border border-white/[0.06] rounded-lg p-4">
      <div className="eyebrow mb-1">{label}</div>
      <div className={cn("text-2xl font-semibold", tone === "bad" ? "text-amber-300" : "text-white")}>{value}</div>
      {meta && <div className="text-xs text-white/40 mt-1">{meta}</div>}
    </div>
  );
}

function Card({ title, right, children }: { title: string; right?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
      <div className="flex items-center justify-between gap-3 mb-4">
        <div className="text-sm font-semibold text-white">{title}</div>
        {right}
      </div>
      {children}
    </div>
  );
}

function CountList({ items, empty, mono, tone }: { items: Count[]; empty?: string; mono?: boolean; tone?: "bad" }) {
  if (!items.length) return <div className="text-xs text-white/30">{empty ?? "Nothing yet."}</div>;
  const max = items.reduce((m, i) => Math.max(m, i.count), 1);
  return (
    <div className="space-y-2">
      {items.map((item) => (
        <BarRow
          key={item.key}
          label={<span className={mono ? "font-mono" : undefined} title={item.key}>{item.key || "/"}</span>}
          count={item.count}
          max={max}
          color={tone === "bad" ? "bg-red-500/60" : undefined}
          wide
        />
      ))}
    </div>
  );
}

function BarRow({
  label,
  count,
  max,
  color = "bg-relay-teal/50",
  wide,
}: {
  label: React.ReactNode;
  count: number;
  max: number;
  color?: string;
  wide?: boolean;
}) {
  const pct = max > 0 ? (count / max) * 100 : 0;
  return (
    <div className="flex items-center gap-3">
      <div className={cn("text-xs text-white/60 shrink-0 truncate", wide ? "w-44 sm:w-56" : "w-32")}>{label}</div>
      <div className="flex-1 h-1.5 bg-white/[0.06] rounded-full overflow-hidden">
        <div className={cn("h-full rounded-full", color)} style={{ width: `${pct}%` }} />
      </div>
      <div className="text-xs text-white/40 w-14 text-right shrink-0">{count.toLocaleString()}</div>
    </div>
  );
}
