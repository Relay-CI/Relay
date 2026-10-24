"use client";

import type { Deploy, EnvInfo, Service } from "@/lib/api";
import { computeConfiguredURL, computePreviewURL, deployDurationLabel, deployPhaseText, engineLabel, formatPercent, timeAgo, type NormalizedProject, type ProjectStats } from "@/lib/relay-utils";

interface SelectedEnvMeta extends EnvInfo { app: string; env: string; branch: string; latestDeploy?: Deploy; previewURL?: string; }
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

export function OverviewPage({ project, contexts, selectedEnv, services, deploys, projectStats, onOpenDeploy, onNavigateSettings }: OverviewPageProps) {
  if (!project) return <div className="empty-state"><h1>Select a project</h1><p>Choose a project from the menu above.</p></div>;
  const latest = selectedEnv?.latestDeploy ?? deploys[0] ?? null;
  const route = selectedEnv ? computeConfiguredURL(selectedEnv) || computePreviewURL(selectedEnv, latest) : "";
  const recent = deploys.slice(0, 8);
  return (
    <div>
      <header className="overview-heading"><div><h1>{project.name}</h1><p>Deployments and running lanes for this project.</p></div><div className="overview-actions">{route && <a href={route} target="_blank" rel="noreferrer">Open app</a>}<button type="button" onClick={onNavigateSettings}>Lane settings</button></div></header>
      <dl className="overview-stats"><div><dt>Lanes</dt><dd>{contexts.length}</dd></div><div><dt>Services in this lane</dt><dd>{services.length}</dd></div><div><dt>Deployments</dt><dd>{projectStats.total}</dd></div><div><dt>Successful deployments</dt><dd>{formatPercent(projectStats.successRate)}</dd></div></dl>
      <div className="overview-grid">
        <section className="overview-panel"><header><h2>Lanes</h2><span>{contexts.length} configured</span></header><div className="overview-table-wrap" role="region" aria-label="Project lanes" tabIndex={0}><table className="overview-table"><thead><tr><th scope="col">Environment</th><th scope="col">Branch</th><th scope="col">Status</th><th scope="col">Deployment</th></tr></thead><tbody>{contexts.map((lane) => <tr key={`${lane.env}/${lane.branch}`}><td>{lane.env}</td><td><code>{lane.branch}</code></td><td><span className="overview-status" data-status={lane.stopped ? "stopped" : lane.latestDeploy?.status}>{lane.stopped ? "Stopped" : lane.latestDeploy ? deployPhaseText(lane.latestDeploy) : "Not deployed"}</span></td><td>{lane.latestDeploy ? <button type="button" onClick={() => onOpenDeploy(lane.latestDeploy!)}>#{lane.latestDeploy.build_number ?? lane.latestDeploy.id.slice(0, 8)}</button> : "—"}</td></tr>)}</tbody></table></div></section>
        <section className="overview-panel"><header><h2>Selected lane</h2></header>{selectedEnv ? <dl className="overview-details"><div><dt>Environment and branch</dt><dd>{selectedEnv.env} / {selectedEnv.branch}</dd></div><div><dt>Runtime</dt><dd>{engineLabel(selectedEnv.engine ?? "docker")}</dd></div><div><dt>App URL</dt><dd>{route || "No route configured"}</dd></div><div><dt>Access</dt><dd>{selectedEnv.access_policy || "Default lane policy"}</dd></div></dl> : <p className="overview-empty">Choose a lane in the sidebar.</p>}</section>
      </div>
      <section className="overview-panel overview-section-gap"><header><h2>Recent deployments</h2><span>{deploys.length} total</span></header>{recent.length ? <div className="overview-table-wrap" role="region" aria-label="Recent deployments" tabIndex={0}><table className="overview-table"><thead><tr><th scope="col">Build</th><th scope="col">Lane</th><th scope="col">Status</th><th scope="col">Duration</th><th scope="col">Started</th><th scope="col">Details</th></tr></thead><tbody>{recent.map((deploy) => <tr key={deploy.id}><td>#{deploy.build_number ?? deploy.id.slice(0,8)}<small>{deploy.commit_message?.slice(0,60) || deploy.source || "Manual deployment"}</small></td><td>{deploy.env}<small>{deploy.branch}</small></td><td><span className="overview-status" data-status={deploy.status}>{deployPhaseText(deploy)}</span></td><td>{deployDurationLabel(deploy)}</td><td>{timeAgo(deploy.created_at)}</td><td><button type="button" onClick={() => onOpenDeploy(deploy)} aria-label={`View deployment ${deploy.build_number ?? deploy.id}`}>View</button></td></tr>)}</tbody></table></div> : <p className="overview-empty">No deployments yet. Run <code>relay deploy --stream</code> in your app folder.</p>}</section>
    </div>
  );
}
