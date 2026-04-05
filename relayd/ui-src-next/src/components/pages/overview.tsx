"use client";

import { cn } from "@/lib/utils";
import {
  computeConfiguredURL,
  computePreviewURL,
  deployKey,
  deployPhaseText,
  deployDurationLabel,
  deployStatusClass,
  engineLabel,
  formatCommitSHA,
  formatPercent,
  liveTargetLabel,
  oldTargetLabel,
  repoProviderInfo,
  projectRepoURL,
  rolloutStrategy,
  timeAgo,
  trafficModeLabel,
  computeProjectStats,
  type NormalizedProject,
  type ProjectStats,
  ACTIVE_STATUSES,
} from "@/lib/relay-utils";
import type { Deploy, EnvInfo, Service } from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
  latestDeploy?: Deploy;
  previewURL?: string;
}

interface OverviewPageProps {
  project: NormalizedProject | null;
  contexts: SelectedEnvMeta[];
  selectedEnv: SelectedEnvMeta | null;
  services: Service[];
  deploys: Deploy[];
  envMap: Map<string, SelectedEnvMeta>;
  projectStats: ProjectStats;
  selectedEnvStats: ProjectStats;
  onOpenDeploy: (deploy: Deploy) => void;
  onNavigateSettings: () => void;
}

function MetricCard({ label, value, meta, tone }: { label: string; value: string; meta?: string; tone?: string }) {
  return (
    <div className={cn("bg-white/[0.03] border border-white/[0.06] rounded-lg p-4", tone && `border-l-2 border-l-relay-${tone}`)}>
      <div className="eyebrow mb-1">{label}</div>
      <div className="text-2xl font-semibold text-white">{value}</div>
      {meta && <div className="text-xs text-white/40 mt-1">{meta}</div>}
    </div>
  );
}

export function OverviewPage({
  project,
  contexts,
  selectedEnv,
  services,
  deploys,
  envMap,
  projectStats,
  selectedEnvStats,
  onOpenDeploy,
  onNavigateSettings,
}: OverviewPageProps) {
  if (!project) {
    return (
      <div className="flex items-center justify-center h-full text-white/30 text-sm">
        Select a project to view the overview
      </div>
    );
  }

  const latestDeploy = selectedEnv?.latestDeploy ?? deploys[0] ?? null;
  const repoInfo = repoProviderInfo(projectRepoURL(project));
  const selectedRoute = selectedEnv
    ? computeConfiguredURL(selectedEnv) || computePreviewURL(selectedEnv, latestDeploy)
    : "";
  const recent = deploys.slice(0, 6);

  return (
    <div className="space-y-5">
      {/* Hero */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-6">
        <div className="flex items-start justify-between gap-4 mb-5">
          <div>
            <div className="eyebrow mb-1">Project control surface</div>
            <h1 className="text-2xl font-semibold text-white">{project.name}</h1>
            <div className="flex flex-wrap gap-2 mt-3">
              <span className="text-[10px] font-semibold uppercase tracking-wider bg-white/[0.04] border border-white/[0.08] px-2 py-0.5 rounded text-white/60">
                {repoInfo.label}
              </span>
              <span className={cn("text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded border", latestDeploy?.status === "success" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : latestDeploy ? "bg-amber-500/10 border-amber-500/30 text-amber-400" : "bg-white/[0.04] border-white/[0.08] text-white/40")}>
                {latestDeploy ? deployPhaseText(latestDeploy) : "No deploy yet"}
              </span>
              {selectedEnv && (
                <span className="text-[10px] font-semibold uppercase tracking-wider bg-relay-teal/10 border border-relay-teal/30 px-2 py-0.5 rounded text-relay-teal">
                  {selectedEnv.env} / {selectedEnv.branch}
                </span>
              )}
            </div>
          </div>
          <div className="flex gap-2 shrink-0">
            {selectedRoute ? (
              <a
                href={selectedRoute}
                target="_blank"
                rel="noreferrer"
                className="text-sm bg-white text-black font-semibold px-3 py-1.5 rounded hover:bg-white/90 transition-colors"
              >
                Open route
              </a>
            ) : (
              <button type="button" disabled className="text-sm bg-white/[0.08] text-white/40 px-3 py-1.5 rounded cursor-not-allowed">
                Route pending
              </button>
            )}
            <button
              type="button"
              onClick={onNavigateSettings}
              className="text-sm border border-white/[0.12] text-white/70 hover:text-white px-3 py-1.5 rounded hover:bg-white/[0.06] transition-colors"
            >
              Manage env
            </button>
          </div>
        </div>

        {/* Command grid */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <MetricCard label="Environments" value={String(contexts.length)} meta="active lanes" tone="teal" />
          <MetricCard label="Services" value={String(project.services?.length ?? 0)} meta="companions" />
          <MetricCard label="Success rate" value={formatPercent(projectStats.successRate)} meta={`${projectStats.total} deploys`} tone="amber" />
          <MetricCard
            label="Latest build"
            value={latestDeploy ? (latestDeploy.build_number ? `#${latestDeploy.build_number}` : latestDeploy.id.slice(0, 8)) : "idle"}
            meta={latestDeploy ? `${timeAgo(latestDeploy.created_at)} ago` : "waiting for first deploy"}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-[1fr_280px] gap-5">
        <div className="space-y-5">
          {/* Lane map */}
          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
            <div className="mb-4">
              <div className="eyebrow mb-0.5">Lane map</div>
              <h2 className="text-base font-semibold text-white">Environment routing</h2>
              <p className="text-xs text-white/40 mt-0.5">Each lane keeps its route, traffic policy, and rollout state visible.</p>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
              {contexts.map((context) => {
                const isActive = selectedEnv && deployKey(context.app, context.env, context.branch) === deployKey(selectedEnv.app, selectedEnv.env, selectedEnv.branch);
                return (
                  <div
                    key={deployKey(context.app, context.env, context.branch)}
                    className={cn("border rounded-lg p-3.5 transition-colors", isActive ? "border-relay-accent/40 bg-relay-accent/5" : "border-white/[0.06] bg-white/[0.02]")}
                  >
                    <div className="flex items-start justify-between gap-2 mb-2">
                      <div>
                        <div className="text-sm font-semibold text-white">{context.env} / {context.branch}</div>
                        <div className="text-xs text-white/35 mt-0.5 truncate">{context.previewURL || "No preview route"}</div>
                      </div>
                      <span className={cn("text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded border shrink-0", context.latestDeploy?.status === "success" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : context.latestDeploy ? "bg-amber-500/10 border-amber-500/30 text-amber-400" : "bg-white/[0.04] border-white/[0.08] text-white/40")}>
                        {context.latestDeploy?.status ?? "idle"}
                      </span>
                    </div>
                    <div className="flex gap-1 flex-wrap mb-2">
                      <span className="text-[10px] bg-white/[0.04] px-1.5 py-0.5 rounded text-white/40">{trafficModeLabel(context.traffic_mode ?? "")}</span>
                      <span className="text-[10px] bg-white/[0.04] px-1.5 py-0.5 rounded text-white/40">{rolloutStrategy(context)}</span>
                      <span className="text-[10px] bg-white/[0.04] px-1.5 py-0.5 rounded text-white/40">{engineLabel(context.engine ?? "docker")}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="flex gap-1">
                        <span className="text-[10px] bg-white/[0.06] px-1.5 py-0.5 rounded text-white/50">{liveTargetLabel(context)}</span>
                        {context.standby_slot && <span className="text-[10px] bg-white/[0.06] px-1.5 py-0.5 rounded text-white/50">{oldTargetLabel(context)}</span>}
                      </div>
                      {context.latestDeploy && (
                        <button type="button" onClick={() => onOpenDeploy(context.latestDeploy!)} className="text-[11px] text-white/40 hover:text-white transition-colors">
                          Logs →
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Recent deploys */}
          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
            <div className="flex items-center justify-between mb-4">
              <div>
                <div className="eyebrow mb-0.5">Recent deployments</div>
                <h2 className="text-base font-semibold text-white">Release board</h2>
              </div>
              <span className="text-xs text-white/40 border border-white/[0.08] px-2 py-0.5 rounded">{deploys.length} total</span>
            </div>
            {!recent.length ? (
              <div className="text-sm text-white/30 text-center py-6">No deployments yet. Trigger a deploy to see activity here.</div>
            ) : (
              <div className="divide-y divide-white/[0.04]">
                {recent.map((deploy) => {
                  const context = envMap.get(deployKey(deploy.app, deploy.env, deploy.branch));
                  return (
                    <div key={deploy.id} className="flex items-center gap-3 py-2.5">
                      <span className={cn("w-2 h-2 rounded-full shrink-0", ACTIVE_STATUSES.has(deploy.status as Parameters<typeof ACTIVE_STATUSES.has>[0]) ? "bg-amber-400 animate-pulse" : deployStatusClass(deploy.status).includes("ok") ? "bg-emerald-500" : deployStatusClass(deploy.status).includes("error") ? "bg-red-500" : "bg-white/20")} />
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-white">
                          {deploy.build_number ? `#${deploy.build_number}` : deploy.id.slice(0, 8)}
                        </div>
                        <div className="text-xs text-white/40 flex gap-2 flex-wrap">
                          <span>{deploy.env} / {deploy.branch}</span>
                          {deploy.commit_sha && <span className="font-mono">{formatCommitSHA(deploy.commit_sha)}</span>}
                          <span>{deployDurationLabel(deploy)}</span>
                          {context && <span>{engineLabel(context.engine ?? "docker")}</span>}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className={cn("text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded border", deployStatusClass(deploy.status) === "status-ok" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : deployStatusClass(deploy.status) === "status-error" ? "bg-red-500/10 border-red-500/30 text-red-400" : "bg-amber-500/10 border-amber-500/30 text-amber-400")}>
                          {deployPhaseText(deploy)}
                        </span>
                        <span className="text-xs text-white/30">{timeAgo(deploy.created_at)} ago</span>
                        <button type="button" onClick={() => onOpenDeploy(deploy)} className="text-xs text-white/40 hover:text-white transition-colors px-2 py-0.5 rounded hover:bg-white/[0.06]">
                          Inspect
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Services */}
          {services.length > 0 && (
            <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
              <div className="mb-3">
                <div className="eyebrow mb-0.5">Stack surface</div>
                <h2 className="text-base font-semibold text-white">Companion services</h2>
              </div>
              <div className="divide-y divide-white/[0.04]">
                {services.map((service) => (
                  <div key={`${(service as { env?: string }).env}-${(service as { branch?: string }).branch}-${service.name}`} className="py-2.5 flex items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium text-white">{service.name}</div>
                      <div className="text-xs text-white/40">{service.type} {service.version ? `v${service.version}` : ""}</div>
                    </div>
                    <div className="flex gap-1.5">
                      <span className={cn("text-[10px] px-2 py-0.5 rounded border", service.running ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : "bg-white/[0.04] border-white/[0.08] text-white/30")}>
                        {service.running ? "running" : "stopped"}
                      </span>
                      {service.port ? <span className="text-[10px] bg-white/[0.04] border border-white/[0.06] px-2 py-0.5 rounded text-white/40">port {service.port}</span> : null}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Right panel */}
        <div className="space-y-4">
          {selectedEnv && (
            <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-4">
              <div className="eyebrow mb-1">Live lane</div>
              <div className="text-base font-semibold text-white mb-3">{selectedEnv.env} / {selectedEnv.branch}</div>
              <div className="space-y-2">
                <KVRow label="Route" value={computeConfiguredURL(selectedEnv) || computePreviewURL(selectedEnv, selectedEnv.latestDeploy ?? null) || "No public route"} />
                <KVRow label="Runtime" value={engineLabel(selectedEnv.engine ?? "docker")} />
                <KVRow label="Traffic" value={trafficModeLabel(selectedEnv.traffic_mode ?? "")} />
                <KVRow label="Rollout" value={rolloutStrategy(selectedEnv)} />
              </div>
            </div>
          )}

          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-4">
            <div className="eyebrow mb-1">Release health</div>
            <div className="text-base font-semibold text-white mb-3">Project signal</div>
            <div className="grid grid-cols-2 gap-2.5">
              <HealthCell label="Project success" value={formatPercent(projectStats.successRate)} />
              <HealthCell label="Lane success" value={formatPercent(selectedEnvStats.successRate)} />
              <HealthCell label="Failures" value={String(projectStats.failures)} />
              <HealthCell label="Total" value={String(projectStats.total)} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function KVRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-2">
      <span className="text-xs text-white/35 shrink-0">{label}</span>
      <span className="text-xs text-white/70 text-right">{value}</span>
    </div>
  );
}

function HealthCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white/[0.03] rounded-lg p-2.5 text-center">
      <div className="text-sm font-semibold text-white">{value}</div>
      <div className="text-[10px] text-white/35 mt-0.5">{label}</div>
    </div>
  );
}
