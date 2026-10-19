"use client";

import { useCallback, useEffect, useState } from "react";
import { cn } from "@/lib/utils";
import { getVariants, variantAction } from "@/lib/api";
import type { EnvInfo, VariantMetrics, VariantsReport } from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
  previewURL?: string;
}

interface VariantsPageProps {
  selectedEnv: SelectedEnvMeta | null;
  isDeployer: boolean;
}

type Period = "24h" | "7d" | "30d";

const PALETTE = ["bg-relay-teal/70", "bg-violet-400/70", "bg-amber-300/70", "bg-sky-400/70", "bg-rose-400/70", "bg-lime-300/70", "bg-fuchsia-400/70", "bg-orange-300/70", "bg-cyan-300/70"];

function laneURL(env: SelectedEnvMeta | null): string {
  if (!env) return "";
  if (env.public_host) return `https://${env.public_host}`;
  return env.previewURL ?? "";
}

export function VariantsPage({ selectedEnv, isDeployer }: VariantsPageProps) {
  const [period, setPeriod] = useState<Period>("7d");
  const [report, setReport] = useState<VariantsReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  const target = selectedEnv ? { app: selectedEnv.app, env: selectedEnv.env, branch: selectedEnv.branch } : null;
  const targetKey = target ? `${target.app}/${target.env}/${target.branch}` : "";

  const load = useCallback(async () => {
    if (!target) return;
    try {
      const next = await getVariants(target, period);
      setReport(next);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load variants");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [targetKey, period]);

  useEffect(() => {
    setLoading(true);
    setDrafts({});
    void load();
    const timer = setInterval(() => void load(), 15_000);
    return () => clearInterval(timer);
  }, [load]);

  async function act(action: "weight" | "remove" | "promote" | "reshuffle", name?: string, weight?: number) {
    if (!target) return;
    setBusy(`${action}:${name ?? ""}`);
    setNotice("");
    setError("");
    try {
      const res = await variantAction(target, action, name, weight);
      if (action === "promote") setNotice(`Promoting ${name} — a blue/green deploy of its build is rolling out (deploy ${res.deploy_id?.slice(0, 8)}). The variant is retired once it's live.`);
      if (action === "remove") setNotice(`${name} removed. Its visitors fall back to the control on their next request.`);
      if (action === "reshuffle") setNotice("Every visitor will be reassigned under the current weights.");
      if (action === "weight") setNotice(`${name} now receives ${weight}% of new visitors. Visitors already assigned keep their variant.`);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Action failed");
    } finally {
      setBusy("");
    }
  }

  if (!selectedEnv) {
    return <div className="flex items-center justify-center h-full text-white/30 text-sm">Select a lane to manage its variants.</div>;
  }

  const variants = report?.variants ?? [];
  const running = variants.filter((v) => v.status === "running");
  const url = laneURL(selectedEnv);
  const segments = report
    ? [{ name: "control", weight: report.control.weight }, ...running.map((v) => ({ name: v.name, weight: v.weight }))]
    : [];

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="max-w-2xl">
          <div className="eyebrow mb-0.5">Experiments</div>
          <h1 className="text-xl font-semibold text-white">Production variants</h1>
          <p className="text-sm text-white/40 mt-1 leading-relaxed">
            Ship alternative builds next to the live release and let real visitors decide. Each visitor is assigned once and sticks to the
            same version (no flicker between builds), weights control the share of new visitors, and the control is whatever this lane&apos;s
            normal deploys produce. Promote the winner when you&apos;re done.
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

      {loading && <div className="text-sm text-white/30 py-8 text-center">Loading variants…</div>}
      {error && <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300">{error}</div>}
      {notice && <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-300">{notice}</div>}

      {!loading && report && !report.supported && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">{report.unsupported}</div>
      )}

      {!loading && report && (
        <>
          <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-5 space-y-3">
            <div className="flex items-center justify-between gap-3">
              <div className="text-sm font-semibold text-white">Traffic split for new visitors</div>
              {isDeployer && running.length > 0 && (
                <button
                  type="button"
                  disabled={!!busy}
                  onClick={() => {
                    if (confirm("Reassign every visitor (including ones already pinned to a variant) under the current weights?")) void act("reshuffle");
                  }}
                  className="text-xs rounded border border-white/[0.1] px-2.5 py-1 text-white/60 hover:text-white hover:border-white/25 disabled:opacity-40"
                >
                  Reshuffle visitors
                </button>
              )}
            </div>
            <div className="flex h-3 rounded-full overflow-hidden bg-white/[0.05]">
              {segments.map((s, i) => (
                <div key={s.name} className={PALETTE[i % PALETTE.length]} style={{ width: `${s.weight}%` }} title={`${s.name}: ${s.weight}%`} />
              ))}
            </div>
            <div className="flex flex-wrap gap-3 text-xs text-white/55">
              {segments.map((s, i) => (
                <span key={s.name} className="flex items-center gap-1.5">
                  <span className={cn("w-2 h-2 rounded-sm", PALETTE[i % PALETTE.length])} />
                  <span className="font-mono">{s.name}</span>
                  <span className="text-white/35">{s.weight.toFixed(s.weight % 1 ? 1 : 0)}%</span>
                </span>
              ))}
            </div>
          </div>

          <div className="space-y-3">
            <VariantRow
              name="control"
              color={PALETTE[0]}
              subtitle={report.control.image ? `live release · ${shortImage(report.control.image)}` : "live release"}
              weightLabel={`${report.control.weight.toFixed(report.control.weight % 1 ? 1 : 0)}% (remainder)`}
              metrics={report.control.metrics}
              previewURL={url ? `${url}/?__relay_variant=control` : ""}
            />
            {variants.map((v, i) => {
              const draft = drafts[v.name] ?? String(v.weight);
              const dirty = Number(draft) !== v.weight;
              const live = v.status === "running";
              return (
                <VariantRow
                  key={v.name}
                  name={v.name}
                  color={live ? PALETTE[(running.findIndex((r) => r.name === v.name) + 1) % PALETTE.length] : "bg-white/20"}
                  subtitle={`${live ? "running" : v.status} · ${shortImage(v.image)}${v.replicas > 1 ? ` · ×${v.replicas}` : ""}${v.description ? ` · ${v.description}` : ""}`}
                  weightLabel={`${v.weight}%${v.weight === 0 ? " (preview only)" : ""}`}
                  metrics={v.metrics}
                  previewURL={url && live ? `${url}/?__relay_variant=${encodeURIComponent(v.name)}` : ""}
                  actions={
                    isDeployer && live ? (
                      <div className="flex flex-wrap items-center gap-2">
                        <div className="flex items-center gap-1">
                          <input
                            className="text-input w-20 py-1 text-xs"
                            inputMode="decimal"
                            value={draft}
                            onChange={(e) => setDrafts((d) => ({ ...d, [v.name]: e.target.value }))}
                            aria-label={`Weight for ${v.name}`}
                          />
                          <span className="text-xs text-white/35">%</span>
                          <button
                            type="button"
                            disabled={!dirty || !!busy || !Number.isFinite(Number(draft))}
                            onClick={() => void act("weight", v.name, Number(draft))}
                            className="text-xs rounded border border-white/[0.1] px-2 py-1 text-white/60 hover:text-white disabled:opacity-30"
                          >
                            Set
                          </button>
                        </div>
                        <button
                          type="button"
                          disabled={!!busy}
                          onClick={() => {
                            if (confirm(`Make ${v.name} the live release for everyone? Its build is rolled out blue/green, then the variant is retired.`)) void act("promote", v.name);
                          }}
                          className="text-xs rounded bg-white text-black px-2.5 py-1 font-semibold hover:bg-white/90 disabled:opacity-40"
                        >
                          Promote
                        </button>
                        <button
                          type="button"
                          disabled={!!busy}
                          onClick={() => {
                            if (confirm(`Remove ${v.name}? Its visitors fall back to the control.`)) void act("remove", v.name);
                          }}
                          className="text-xs rounded border border-red-500/30 px-2.5 py-1 text-red-300 hover:bg-red-500/10 disabled:opacity-40"
                        >
                          Remove
                        </button>
                      </div>
                    ) : null
                  }
                />
              );
            })}
            {variants.length === 0 && (
              <div className="rounded-xl border border-dashed border-white/[0.1] px-5 py-8 text-center text-sm text-white/40">
                No variants on this lane yet.
              </div>
            )}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-5 space-y-3">
              <div className="text-sm font-semibold text-white">Ship a variant</div>
              <p className="text-xs text-white/40 leading-relaxed">
                Deploy from the same project with a variant name. It starts beside the live release, passes readiness, and then receives its
                share of new visitors. Re-deploying the same name updates it in place with zero downtime. A weight of 0 makes it preview-only.
              </p>
              <Snippet text={`relay deploy ${selectedEnv.env} --variant checkout-v2 --weight 10`} />
              <Snippet text="relay variants" />
            </div>
            <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-5 space-y-3">
              <div className="text-sm font-semibold text-white">Collect feedback</div>
              <p className="text-xs text-white/40 leading-relaxed">
                Any page can post to <code className="font-mono">/__relay/feedback</code>. Relay files it under the variant that visitor is actually
                seeing, so you never have to pass the variant yourself (and visitors can&apos;t fake it).
              </p>
              <Snippet
                text={`fetch("/__relay/feedback", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ vote: "up", rating: 5, comment: "Love the new layout", page: location.pathname }),
});`}
              />
              <p className="text-[11px] text-white/30">Your app also receives an <code className="font-mono">X-Relay-Variant</code> request header and a <code className="font-mono">RELAY_VARIANT</code> env var.</p>
            </div>
          </div>

          <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-5">
            <div className="text-sm font-semibold text-white mb-3">Recent feedback</div>
            {(report.feedback ?? []).length === 0 ? (
              <div className="text-xs text-white/30">No feedback in this period yet.</div>
            ) : (
              <div className="space-y-2">
                {report.feedback.map((f) => (
                  <div key={`${f.ts}-${f.variant}-${f.comment.slice(0, 12)}`} className="flex gap-3 text-xs border-t border-white/[0.05] pt-2 first:border-0 first:pt-0">
                    <span className="font-mono text-white/70 w-24 shrink-0 truncate">{f.variant}</span>
                    <span className="w-10 shrink-0">{f.vote > 0 ? "👍" : f.vote < 0 ? "👎" : ""}</span>
                    <span className="w-16 shrink-0 text-amber-300">{f.rating ? "★".repeat(f.rating) : ""}</span>
                    <span className="flex-1 text-white/70 break-words">{f.comment || <span className="text-white/25">no comment</span>}</span>
                    <span className="text-white/30 shrink-0 hidden sm:block">{f.page}</span>
                    <span className="text-white/30 shrink-0">{new Date(f.ts).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}

function shortImage(image?: string): string {
  if (!image) return "no image";
  const tag = image.split(":").pop() ?? image;
  return tag.length > 24 ? `${tag.slice(0, 24)}…` : tag;
}

function VariantRow({
  name,
  color,
  subtitle,
  weightLabel,
  metrics,
  previewURL,
  actions,
}: {
  name: string;
  color: string;
  subtitle: string;
  weightLabel: string;
  metrics: VariantMetrics;
  previewURL: string;
  actions?: React.ReactNode;
}) {
  const feedback = metrics.feedback
    ? `👍 ${metrics.thumbs_up} · 👎 ${metrics.thumbs_down}${metrics.avg_rating ? ` · ★ ${metrics.avg_rating.toFixed(1)}` : ""}`
    : "—";
  return (
    <div className="rounded-xl border border-white/[0.08] bg-black/20 p-4 space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn("w-2.5 h-2.5 rounded-sm shrink-0", color)} />
            <span className="text-sm font-semibold text-white font-mono">{name}</span>
            <span className="text-xs text-white/45">{weightLabel}</span>
          </div>
          <div className="text-[11px] text-white/35 mt-1 truncate">{subtitle}</div>
        </div>
        <div className="flex items-center gap-2">
          {previewURL && (
            <a href={previewURL} target="_blank" rel="noreferrer" className="text-xs rounded border border-white/[0.1] px-2.5 py-1 text-white/60 hover:text-white hover:border-white/25">
              Preview ↗
            </a>
          )}
          {actions}
        </div>
      </div>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2">
        <Metric label="Visitors" value={metrics.visitors.toLocaleString()} />
        <Metric label="Page views" value={metrics.page_views.toLocaleString()} />
        <Metric label="Requests" value={metrics.requests.toLocaleString()} />
        <Metric label="5xx rate" value={metrics.requests ? `${metrics.error_rate.toFixed(2)}%` : "—"} bad={metrics.error_rate >= 1} />
        <Metric label="p95" value={metrics.p95_ms ? `${metrics.p95_ms} ms` : "—"} />
        <Metric label="Feedback" value={feedback} />
      </div>
    </div>
  );
}

function Metric({ label, value, bad }: { label: string; value: string; bad?: boolean }) {
  return (
    <div className="rounded-lg border border-white/[0.06] bg-white/[0.03] px-3 py-2">
      <div className="text-[10px] text-white/35 mb-0.5">{label}</div>
      <div className={cn("text-sm", bad ? "text-amber-300" : "text-white")}>{value}</div>
    </div>
  );
}

function Snippet({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="relative">
      <pre className="text-[11px] font-mono text-white/75 bg-black/30 border border-white/[0.06] rounded-lg p-3 pr-16 overflow-x-auto whitespace-pre">{text}</pre>
      <button
        type="button"
        onClick={() => {
          void navigator.clipboard?.writeText(text).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
          });
        }}
        className="absolute top-2 right-2 text-[10px] rounded border border-white/[0.1] px-2 py-0.5 text-white/50 hover:text-white"
      >
        {copied ? "Copied" : "Copy"}
      </button>
    </div>
  );
}
