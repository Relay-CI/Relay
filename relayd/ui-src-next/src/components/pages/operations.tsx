"use client";

import { useEffect, useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import {
  getAdminOps,
  type AdminOpsApp,
  type AdminOpsContainerUsage,
  type AdminOpsDeployDelta,
  type AdminOpsLane,
  type AdminOpsResponse,
} from "@/lib/api";

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size >= 10 || unit === 0 ? size.toFixed(unit === 0 ? 0 : 1) : size.toFixed(2)} ${units[unit]}`;
}

function formatRate(value: number): string {
  if (!Number.isFinite(value)) return "0%";
  return `${value.toFixed(value >= 10 ? 0 : 1)}%`;
}

function formatSignedDelta(value: number, suffix: string): string {
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  const abs = Math.abs(value);
  return `${sign}${abs.toFixed(abs >= 10 ? 0 : 1)}${suffix}`;
}

function formatDurationDelta(ms: number): string {
  const abs = Math.abs(ms);
  const seconds = abs / 1000;
  const body =
    seconds < 60
      ? `${seconds.toFixed(seconds >= 10 ? 0 : 1)}s`
      : `${(seconds / 60).toFixed(1)}m`;
  return `${ms > 0 ? "+" : ms < 0 ? "-" : ""}${body}`;
}

function formatDuration(ms: number): string {
  const abs = Math.max(0, ms);
  const seconds = abs / 1000;
  if (seconds < 60) return `${seconds.toFixed(seconds >= 10 ? 0 : 1)}s`;
  return `${(seconds / 60).toFixed(1)}m`;
}

function relativeTime(input?: string): string {
  if (!input) return "n/a";
  const ts = new Date(input).getTime();
  if (!Number.isFinite(ts)) return "n/a";
  const delta = Math.max(0, Date.now() - ts);
  const mins = Math.floor(delta / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function laneLabel(lane: AdminOpsLane): string {
  return lane.branch === "main" ? lane.env : `${lane.env}/${lane.branch}`;
}

function deltaTone(value: number, betterWhenLower = false): string {
  if (value === 0) return "text-white/45";
  const improved = betterWhenLower ? value < 0 : value > 0;
  return improved ? "text-emerald-300" : "text-amber-300";
}

function barWidth(percent: number, fallbackValue: number, cap = 100): string {
  if (percent > 0) return `${Math.max(4, Math.min(cap, percent))}%`;
  if (fallbackValue > 0) return `${Math.max(8, Math.min(cap, fallbackValue))}%`;
  return "4%";
}

export function OperationsPage() {
  const [data, setData] = useState<AdminOpsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    getAdminOps()
      .then((result) => {
        if (cancelled) return;
        setData(result);
        setLoading(false);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : "Failed to load operations data");
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const hottestApps = useMemo(() => {
    return (data?.apps ?? [])
      .slice()
      .sort((a, b) => b.usage.cpu_percent - a.usage.cpu_percent)
      .slice(0, 3);
  }, [data]);

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Administration</div>
        <h1 className="text-xl font-semibold text-white">Operations</h1>
        <p className="text-sm text-white/40 mt-1">
          Live lane resource usage plus deploy-to-deploy deltas for build time, traffic, and server error rate.
        </p>
      </div>

      {loading && <div className="text-sm text-white/30 py-8 text-center">Loading operations telemetry…</div>}
      {error && (
        <div className="rounded-xl border border-red-500/25 bg-red-500/10 px-4 py-3 text-sm text-red-300">
          {error}
        </div>
      )}

      {!loading && !error && data && (
        <>
          <div className="grid grid-cols-2 xl:grid-cols-5 gap-3">
            <SummaryCard label="Apps" value={String(data.summary.app_count)} meta={`${data.summary.lane_count} lanes`} />
            <SummaryCard label="Live CPU" value={formatRate(data.summary.cpu_percent)} meta="sum of running containers" />
            <SummaryCard label="Live Memory" value={formatBytes(data.summary.mem_usage_bytes)} meta={data.summary.mem_limit_bytes > 0 ? `${formatRate(data.summary.mem_percent)} of limits` : "no hard memory cap"} />
            <SummaryCard label="Runtime Storage" value={formatBytes(data.summary.storage_bytes)} meta="container rootfs + writable layers" />
            <SummaryCard label="Online Lanes" value={String(data.summary.online_lanes)} meta={`${data.summary.lane_count - data.summary.online_lanes} offline`} />
          </div>

          {!!hottestApps.length && (
            <div className="rounded-2xl border border-white/[0.08] bg-[linear-gradient(135deg,rgba(255,255,255,0.05),rgba(0,188,212,0.06),rgba(255,255,255,0.02))] p-5">
              <div className="flex items-center justify-between gap-3 mb-4">
                <div>
                  <div className="eyebrow mb-0.5">Heat map</div>
                  <div className="text-base font-semibold text-white">Hottest apps right now</div>
                </div>
                <div className="text-xs text-white/35">Snapshot updates each page load</div>
              </div>
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
                {hottestApps.map((app) => (
                  <div key={app.app} className="rounded-xl border border-white/[0.08] bg-black/25 p-4 space-y-3">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="text-sm font-semibold text-white">{app.app}</div>
                        <div className="text-xs text-white/40">{app.online_lanes}/{app.lane_count} lanes online</div>
                      </div>
                      <div className="text-right">
                        <div className="text-lg font-semibold text-white">{formatRate(app.usage.cpu_percent)}</div>
                        <div className="text-[10px] text-white/35">CPU now</div>
                      </div>
                    </div>
                    <UsageStrip label="Memory" value={formatBytes(app.usage.mem_usage_bytes)} percent={app.usage.mem_percent} />
                    <UsageStrip label="Storage" value={formatBytes(app.usage.storage_bytes)} percent={0} />
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="space-y-4">
            {data.apps.map((app) => (
              <AppSection key={app.app} app={app} />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function SummaryCard({ label, value, meta }: { label: string; value: string; meta: string }) {
  return (
    <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] px-4 py-4">
      <div className="eyebrow mb-1">{label}</div>
      <div className="text-2xl font-semibold text-white">{value}</div>
      <div className="text-xs text-white/38 mt-1">{meta}</div>
    </div>
  );
}

function UsageStrip({
  label,
  value,
  percent,
}: {
  label: string;
  value: string;
  percent: number;
}) {
  return (
    <div>
      <div className="flex items-center justify-between gap-3 text-xs text-white/50 mb-1">
        <span>{label}</span>
        <span>{value}</span>
      </div>
      <div className="h-2 rounded-full bg-white/[0.05] overflow-hidden">
        <div
          className="h-full rounded-full bg-[linear-gradient(90deg,rgba(0,188,212,0.45),rgba(255,255,255,0.75))]"
          style={{ width: barWidth(percent, percent) }}
        />
      </div>
    </div>
  );
}

function AppSection({ app }: { app: AdminOpsApp }) {
  return (
    <section className="rounded-2xl border border-white/[0.06] bg-white/[0.02] overflow-hidden">
      <div className="px-5 py-4 border-b border-white/[0.06] bg-[linear-gradient(90deg,rgba(255,255,255,0.04),rgba(0,188,212,0.05),transparent)]">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="text-base font-semibold text-white">{app.app}</div>
            <div className="text-xs text-white/40">
              {app.online_lanes}/{app.lane_count} lanes online
            </div>
          </div>
          <div className="flex flex-wrap gap-2 text-xs">
            <Badge label={`CPU ${formatRate(app.usage.cpu_percent)}`} />
            <Badge label={`MEM ${formatBytes(app.usage.mem_usage_bytes)}`} />
            <Badge label={`STORAGE ${formatBytes(app.usage.storage_bytes)}`} />
          </div>
        </div>
      </div>
      <div className="grid grid-cols-1 2xl:grid-cols-2 gap-4 p-4">
        {app.lanes.map((lane) => (
          <LaneCard key={`${lane.app}-${lane.env}-${lane.branch}`} lane={lane} />
        ))}
      </div>
    </section>
  );
}

function LaneCard({ lane }: { lane: AdminOpsLane }) {
  return (
    <div className="rounded-xl border border-white/[0.08] bg-black/20 p-4 space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <div className="text-sm font-semibold text-white">{laneLabel(lane)}</div>
            <span
              className={cn(
                "text-[10px] px-1.5 py-0.5 rounded border",
                lane.stopped || lane.usage.running_containers === 0
                  ? "border-white/[0.08] text-white/35 bg-white/[0.04]"
                  : "border-emerald-500/20 text-emerald-300 bg-emerald-500/10",
              )}
            >
              {lane.stopped || lane.usage.running_containers === 0 ? "offline" : "live"}
            </span>
          </div>
          <div className="text-xs text-white/40 mt-1">
            {lane.engine} {lane.public_host ? `· ${lane.public_host}` : lane.host_port ? `· port ${lane.host_port}` : ""}
          </div>
        </div>
        <div className="text-right">
          <div className="text-lg font-semibold text-white">{formatRate(lane.usage.cpu_percent)}</div>
          <div className="text-[10px] text-white/35">CPU now</div>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-2">
        <MiniMetric label="Memory" value={formatBytes(lane.usage.mem_usage_bytes)} sub={lane.usage.mem_limit_bytes > 0 ? formatRate(lane.usage.mem_percent) : "no cap"} />
        <MiniMetric label="Storage" value={formatBytes(lane.usage.storage_bytes)} sub={`${lane.usage.container_count} containers`} />
        <MiniMetric label="Network" value={formatBytes(lane.usage.net_rx_bytes + lane.usage.net_tx_bytes)} sub="rx + tx" />
      </div>

      {lane.latest && <DeployDeltaPanel delta={lane.latest} />}

      <div className="space-y-2">
        <div className="eyebrow">Container breakdown</div>
        {lane.usage.note && !lane.usage.measured && (
          <div className="text-xs text-white/35">{lane.usage.note}</div>
        )}
        <div className="space-y-2">
          {lane.usage.targets.map((target) => (
            <ContainerRow key={target.id} target={target} />
          ))}
        </div>
      </div>
    </div>
  );
}

function DeployDeltaPanel({ delta }: { delta: AdminOpsDeployDelta }) {
  const buildTone = deltaTone(delta.build_duration_delta_ms ?? 0, true);
  const errTone = deltaTone(delta.server_error_rate_delta ?? 0, true);
  const reqTone = deltaTone(delta.request_delta ?? 0, false);

  return (
    <div className="rounded-xl border border-white/[0.08] bg-white/[0.03] p-3.5 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="eyebrow mb-0.5">Latest deploy</div>
          <div className="text-sm font-semibold text-white">
            #{delta.current.build_number || "?"} · {relativeTime(delta.current.created_at)}
          </div>
        </div>
        <div className="text-right">
          <div className="text-xs text-white/35">Build duration</div>
          <div className="text-sm text-white">{formatDuration(delta.current.build_duration_ms ?? 0)}</div>
        </div>
      </div>

      {delta.previous ? (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
          <DeltaChip
            label="Build vs previous"
            value={formatDurationDelta(delta.build_duration_delta_ms ?? 0)}
            tone={buildTone}
            caption={delta.build_duration_delta_ms && delta.build_duration_delta_ms < 0 ? "faster" : "slower"}
          />
          <DeltaChip
            label="5xx rate delta"
            value={formatSignedDelta(delta.server_error_rate_delta ?? 0, " pts")}
            tone={errTone}
            caption={delta.analytics_available ? `window ${Math.round((delta.window.seconds || 0) / 60)}m` : "traffic unavailable"}
          />
          <DeltaChip
            label="Requests delta"
            value={`${delta.request_delta && delta.request_delta > 0 ? "+" : ""}${(delta.request_delta ?? 0).toLocaleString()}`}
            tone={reqTone}
            caption={delta.analytics_available ? "same post-deploy window" : "traffic unavailable"}
          />
        </div>
      ) : (
        <div className="text-xs text-white/35">{delta.analytics_note || "No prior successful deploy to compare yet."}</div>
      )}

      {delta.analytics_available && delta.current_traffic && delta.previous_traffic && (
        <div className="grid grid-cols-2 gap-2 text-xs">
          <TrafficBox title="Latest window" traffic={delta.current_traffic} />
          <TrafficBox title="Previous window" traffic={delta.previous_traffic} />
        </div>
      )}
      {!delta.analytics_available && delta.analytics_note && (
        <div className="text-xs text-white/35">{delta.analytics_note}</div>
      )}
    </div>
  );
}

function DeltaChip({
  label,
  value,
  tone,
  caption,
}: {
  label: string;
  value: string;
  tone: string;
  caption: string;
}) {
  return (
    <div className="rounded-lg border border-white/[0.06] bg-black/20 px-3 py-2.5">
      <div className="text-[10px] text-white/35 mb-1">{label}</div>
      <div className={cn("text-sm font-semibold", tone)}>{value}</div>
      <div className="text-[10px] text-white/30 mt-1">{caption}</div>
    </div>
  );
}

function TrafficBox({
  title,
  traffic,
}: {
  title: string;
  traffic: NonNullable<AdminOpsDeployDelta["current_traffic"]>;
}) {
  return (
    <div className="rounded-lg border border-white/[0.06] bg-black/20 px-3 py-2.5 space-y-1">
      <div className="text-[10px] text-white/35">{title}</div>
      <div className="text-xs text-white/70">Requests: {traffic.requests.toLocaleString()}</div>
      <div className="text-xs text-white/70">5xx rate: {formatRate(traffic.server_error_rate)}</div>
      <div className="text-xs text-white/70">Bandwidth: {formatBytes(traffic.bandwidth_bytes)}</div>
    </div>
  );
}

function ContainerRow({ target }: { target: AdminOpsContainerUsage }) {
  return (
    <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs font-medium text-white truncate">{target.label}</div>
          <div className="text-[10px] text-white/35 truncate">{target.container}</div>
        </div>
        <span
          className={cn(
            "text-[10px] px-1.5 py-0.5 rounded",
            target.running
              ? "bg-emerald-500/10 text-emerald-300"
              : "bg-white/[0.04] text-white/35",
          )}
        >
          {target.running ? "running" : "idle"}
        </span>
      </div>
      <div className="grid grid-cols-3 gap-3 mt-3">
        <TinyBar label="CPU" value={formatRate(target.cpu_percent)} percent={target.cpu_percent} accent="bg-emerald-400/75" />
        <TinyBar label="MEM" value={formatBytes(target.mem_usage_bytes)} percent={target.mem_percent} accent="bg-sky-400/75" />
        <TinyBar label="STORAGE" value={formatBytes(target.storage_bytes)} percent={0} accent="bg-amber-300/75" fallback={36} />
      </div>
    </div>
  );
}

function TinyBar({
  label,
  value,
  percent,
  accent,
  fallback = 18,
}: {
  label: string;
  value: string;
  percent: number;
  accent: string;
  fallback?: number;
}) {
  return (
    <div>
      <div className="flex items-center justify-between gap-2 text-[10px] text-white/40 mb-1">
        <span>{label}</span>
        <span>{value}</span>
      </div>
      <div className="h-1.5 rounded-full bg-white/[0.05] overflow-hidden">
        <div className={cn("h-full rounded-full", accent)} style={{ width: barWidth(percent, fallback) }} />
      </div>
    </div>
  );
}

function MiniMetric({
  label,
  value,
  sub,
}: {
  label: string;
  value: string;
  sub: string;
}) {
  return (
    <div className="rounded-lg border border-white/[0.06] bg-white/[0.03] px-3 py-2.5">
      <div className="text-[10px] text-white/35 mb-1">{label}</div>
      <div className="text-sm text-white">{value}</div>
      <div className="text-[10px] text-white/30 mt-1">{sub}</div>
    </div>
  );
}

function Badge({ label }: { label: string }) {
  return (
    <span className="rounded-full border border-white/[0.08] bg-white/[0.04] px-2.5 py-1 text-white/55">
      {label}
    </span>
  );
}
