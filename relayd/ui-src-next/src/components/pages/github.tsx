"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import { formatCommitSHA, timeAgo, type NormalizedProject } from "@/lib/relay-utils";
import {
  abortRollout,
  connectGitHub,
  connectGitHubProject,
  disconnectGitHub,
  disconnectGitHubProject,
  getGitHubApp,
  getGitHubConnection,
  getGitHubProjects,
  getGitHubRepositories,
  rollback,
  startGitHubAppInstall,
  startGitHubAppManifest,
  type GitHubAppStatus,
  type GitHubConnection,
  type GitHubProject,
  type GitHubRepository,
} from "@/lib/api";

type CurrentUser = { username: string; role: string } | null;

interface GitHubPageProps {
  project: NormalizedProject | null;
  currentUser: CurrentUser;
  onUpdated: () => Promise<void>;
}

type RailState = "waiting" | "active" | "done" | "failed";

function RailStep({ index, title, detail, state }: { index: number; title: string; detail: string; state: RailState }) {
  return (
    <div className="relative min-w-0">
      <div className={cn(
        "relative z-10 w-8 h-8 rounded-full border flex items-center justify-center font-mono text-[11px] font-bold transition-colors",
        state === "done" && "border-emerald-400/60 bg-emerald-400/15 text-emerald-300",
        state === "active" && "border-relay-accent bg-relay-accent/20 text-white shadow-[0_0_22px_rgba(59,130,246,0.2)]",
        state === "failed" && "border-red-400/60 bg-red-500/15 text-red-300",
        state === "waiting" && "border-white/10 bg-zinc-900 text-white/30",
      )}>
        {state === "done" ? (
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="20 6 9 17 4 12" /></svg>
        ) : state === "active" ? (
          <span className="w-2 h-2 rounded-full bg-relay-accent animate-pulse" />
        ) : state === "failed" ? "!" : index}
      </div>
      <div className="mt-3 pr-3">
        <div className="text-xs font-semibold text-white/85">{title}</div>
        <div className="text-[11px] text-white/35 mt-1 leading-relaxed min-h-8">{detail}</div>
      </div>
    </div>
  );
}

function WorkflowRail({ workflow }: { workflow: GitHubProject }) {
  const preview = workflow.previews[0];
  const production = workflow.production;
  const previewFailed = preview?.status === "failed";
  const previewDone = Boolean(preview?.preview_url && preview?.status === "success");
  const productionFailed = production.deploy_status === "failed" || production.health_status === "unhealthy";
  const productionDone = production.deploy_status === "success";
  const mergeObserved = workflow.last_event?.outcome === "merged_waiting_for_production_push" || Boolean(production.deploy_id);

  const steps: Array<{ title: string; detail: string; state: RailState }> = [
    { title: "Connected", detail: workflow.repo_full_name, state: "done" },
    {
      title: "Branch push",
      detail: preview ? `${preview.branch}${preview.head_sha ? ` · ${formatCommitSHA(preview.head_sha)}` : ""}` : "Waiting for a non-production branch",
      state: preview ? "done" : "waiting",
    },
    {
      title: "HTTPS preview",
      detail: previewFailed ? "Preview deployment failed" : previewDone ? "Route published to GitHub" : preview ? "Building and checking" : "Generated after the first push",
      state: previewFailed ? "failed" : previewDone ? "done" : preview ? "active" : "waiting",
    },
    {
      title: "Merge",
      detail: mergeObserved ? `Production branch ${workflow.production_branch}` : "Merge the reviewed pull request",
      state: mergeObserved ? "done" : "waiting",
    },
    {
      title: "Production",
      detail: production.deploy_id ? `${production.deploy_status ?? "queued"} · ${production.branch}` : "Deploys when the production branch moves",
      state: productionFailed ? "failed" : productionDone ? "done" : production.deploy_id ? "active" : "waiting",
    },
    {
      title: "Health watch",
      detail: production.health_detail || "Readiness and rollout signals appear here",
      state: production.health_status === "healthy" ? "done" : production.health_status === "unhealthy" ? "failed" : production.health_status === "deploying" ? "active" : "waiting",
    },
  ];

  return (
    <div className="relative">
      <div className="hidden lg:block absolute left-4 right-[9%] top-4 h-px bg-gradient-to-r from-emerald-400/50 via-relay-accent/35 to-white/10" />
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 lg:gap-2">
        {steps.map((step, index) => <RailStep key={step.title} index={index + 1} {...step} />)}
      </div>
    </div>
  );
}

export function GitHubPage({ project, currentUser, onUpdated }: GitHubPageProps) {
  const isOwner = currentUser?.role === "owner";
  const [connection, setConnection] = useState<GitHubConnection | null>(null);
  const [githubApp, setGitHubApp] = useState<GitHubAppStatus | null>(null);
  const [workflow, setWorkflow] = useState<GitHubProject | null>(null);
  const [repositories, setRepositories] = useState<GitHubRepository[]>([]);
  const [token, setToken] = useState("");
  const [organization, setOrganization] = useState("");
  const [repoName, setRepoName] = useState("");
  const [productionBranch, setProductionBranch] = useState("main");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState<{ tone: "ok" | "warn"; text: string } | null>(null);

  const load = useCallback(async () => {
    try {
      const [nextConnection, nextApp, projects] = await Promise.all([
        getGitHubConnection(),
        getGitHubApp(),
        project ? getGitHubProjects(project.name) : Promise.resolve([]),
      ]);
      setConnection(nextConnection);
      setGitHubApp(nextApp);
      setWorkflow(projects[0] ?? null);
    } catch (error) {
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Failed to load GitHub workflow." });
    }
  }, [project]);

  useEffect(() => {
    load();
    const timer = window.setInterval(load, 5000);
    return () => window.clearInterval(timer);
  }, [load]);

  useEffect(() => {
    if ((!connection?.connected && !githubApp?.registered) || !isOwner || workflow) return;
    getGitHubRepositories()
      .then((items) => {
        setRepositories(items);
        if (!repoName && items[0]) {
          setRepoName(items[0].full_name);
          setProductionBranch(items[0].default_branch || "main");
        }
      })
      .catch((error) => setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Could not list repositories." }));
  }, [connection?.connected, githubApp?.registered, isOwner, workflow, repoName]);

  const selectedRepository = useMemo(
    () => repositories.find((repo) => repo.full_name === repoName),
    [repositories, repoName],
  );
  const latestPreview = workflow?.previews[0];
  const githubReady = Boolean(connection?.connected || (githubApp?.registered && repositories.length > 0));

  async function handleRegisterApp() {
    setBusy("register-app");
    setNotice(null);
    try {
      const registration = await startGitHubAppManifest(organization);
      const form = document.createElement("form");
      form.method = "POST";
      form.action = `${registration.action}?state=${encodeURIComponent(registration.state)}`;
      const manifest = document.createElement("input");
      manifest.type = "hidden";
      manifest.name = "manifest";
      manifest.value = JSON.stringify(registration.manifest);
      form.appendChild(manifest);
      document.body.appendChild(form);
      form.submit();
    } catch (error) {
      setBusy("");
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Could not start GitHub App registration." });
    }
  }

  async function handleInstallApp() {
    setBusy("install-app");
    setNotice(null);
    try {
      const installation = await startGitHubAppInstall();
      window.location.assign(installation.install_url);
    } catch (error) {
      setBusy("");
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Could not start GitHub App installation." });
    }
  }

  async function handleConnect() {
    if (!token.trim()) return;
    setBusy("connect");
    setNotice(null);
    try {
      const next = await connectGitHub(token.trim());
      setConnection(next);
      setToken("");
      setNotice({ tone: "ok", text: `Connected as ${next.login}. Choose a repository for this project.` });
    } catch (error) {
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "GitHub connection failed." });
    } finally {
      setBusy("");
    }
  }

  async function handleLinkProject() {
    if (!project || !repoName) return;
    setBusy("link");
    setNotice(null);
    try {
      const next = await connectGitHubProject({
        app: project.name,
        repo_full_name: repoName,
        production_branch: productionBranch || selectedRepository?.default_branch || "main",
        preview_enabled: true,
        production_enabled: true,
      });
      setWorkflow(next);
      await onUpdated();
      setNotice({ tone: "ok", text: "Delivery workflow installed. Push a branch to start the first HTTPS preview." });
    } catch (error) {
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Could not connect this repository." });
    } finally {
      setBusy("");
    }
  }

  async function handleRollback() {
    if (!workflow?.production.rollback_available) return;
    if (!window.confirm(`Restore the previous production image for ${workflow.app}/${workflow.production.branch}?`)) return;
    setBusy("rollback");
    try {
      if (workflow.production.rollout_status === "monitoring") {
        await abortRollout({ app: workflow.app, env: "prod", branch: workflow.production.branch });
        setNotice({ tone: "ok", text: "Rollout stopped. Relay restored the previously active production slot." });
      } else {
        await rollback({ app: workflow.app, env: "prod", branch: workflow.production.branch });
        setNotice({ tone: "ok", text: "Rollback queued. Relay is restoring the previous production image now." });
      }
      await load();
      await onUpdated();
    } catch (error) {
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Rollback failed." });
    } finally {
      setBusy("");
    }
  }

  async function handleUnlink() {
    if (!project || !window.confirm(`Disconnect GitHub from ${project.name}? Existing Relay lanes stay intact.`)) return;
    setBusy("unlink");
    try {
      await disconnectGitHubProject(project.name);
      setWorkflow(null);
      setNotice({ tone: "ok", text: "Repository connection removed. Existing deployments were preserved." });
    } catch (error) {
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Could not disconnect repository." });
    } finally {
      setBusy("");
    }
  }

  async function handleDisconnectAccount() {
    if (!window.confirm("Disconnect the GitHub account from this Relay server?")) return;
    setBusy("disconnect");
    try {
      const next = await disconnectGitHub();
      setConnection(next);
      setRepositories([]);
      setNotice({ tone: "ok", text: "GitHub account disconnected." });
    } catch (error) {
      setNotice({ tone: "warn", text: error instanceof Error ? error.message : "Disconnect failed." });
    } finally {
      setBusy("");
    }
  }

  if (!project) {
    return <div className="empty-state"><div className="empty-state__title">Select a project</div><div className="empty-state__sub">GitHub delivery workflows are linked per Relay project.</div></div>;
  }

  return (
    <div className="space-y-5">
      <div className="relative overflow-hidden rounded-xl border border-white/[0.08] bg-zinc-950 p-6">
        <div className="absolute inset-0 opacity-40 bg-[radial-gradient(circle_at_75%_0%,rgba(59,130,246,0.17),transparent_38%)]" />
        <div className="relative flex flex-col md:flex-row md:items-start md:justify-between gap-4">
          <div>
            <div className="eyebrow mb-1">GitHub delivery rail</div>
            <h1 className="text-2xl font-semibold text-white">Push code. Follow it to production.</h1>
            <p className="text-sm text-white/45 mt-2 max-w-2xl leading-relaxed">
              Relay turns branch events into isolated previews, publishes the HTTPS route back to the commit, then watches the production rollout after merge.
            </p>
          </div>
          <div className={cn("shrink-0 rounded-full border px-3 py-1.5 text-xs font-semibold", githubReady || workflow ? "border-emerald-400/30 bg-emerald-400/10 text-emerald-300" : "border-white/10 bg-white/[0.03] text-white/40")}>
            {githubApp?.registered ? `GitHub App · ${githubApp.app_name || githubApp.app_slug}` : connection?.connected ? `Token · ${connection.login}` : "Not connected"}
          </div>
        </div>
      </div>

      {notice && <div className={cn("rounded-lg border px-4 py-3 text-sm", notice.tone === "ok" ? "border-emerald-500/25 bg-emerald-500/10 text-emerald-300" : "border-amber-500/25 bg-amber-500/10 text-amber-200")}>{notice.text}</div>}

      {connection && !connection.secret_key_configured && (
        <div className="rounded-xl border border-amber-500/30 bg-amber-500/[0.07] p-5">
          <div className="text-sm font-semibold text-amber-200">Encryption key required</div>
          <p className="text-xs text-amber-100/55 mt-1">Set <code className="font-mono">RELAY_SECRET_KEY</code> in Server Settings -&gt; Security before registering the GitHub App.</p>
        </div>
      )}

      {!githubReady ? (
        <div className="grid grid-cols-1 xl:grid-cols-[1fr_360px] gap-5">
          <div className="rounded-xl border border-white/[0.08] bg-white/[0.02] p-5">
            <div className="eyebrow mb-1">Step 01</div>
            <h2 className="text-base font-semibold text-white">{githubApp?.registered ? "Install Relay on GitHub" : "Create your Relay GitHub App"}</h2>
            <p className="text-xs text-white/40 mt-1 max-w-xl">{githubApp?.registered ? "Choose the organizations and repositories this Relay server may deploy. GitHub returns short-lived installation credentials automatically." : "Relay creates a private GitHub App owned by you, with only repository contents read and Checks write access."}</p>
            {!githubApp?.registered && <label className="block mt-5 max-w-sm"><span className="eyebrow block mb-1.5">Organization owner <span className="normal-case tracking-normal text-white/20">optional</span></span><input className="text-input font-mono" value={organization} onChange={(event) => setOrganization(event.target.value)} placeholder="acme-org" disabled={!isOwner || Boolean(busy)} /><span className="block text-[11px] text-white/25 mt-1.5">Leave empty for repositories owned by your personal GitHub account.</span></label>}
            <button type="button" onClick={githubApp?.registered ? handleInstallApp : handleRegisterApp} disabled={!isOwner || Boolean(busy) || !connection?.secret_key_configured} className="primary-btn px-5 mt-5">{busy === "register-app" ? "Opening GitHub…" : busy === "install-app" ? "Opening installation…" : githubApp?.registered ? "Select repositories on GitHub ↗" : "Create GitHub App ↗"}</button>
            {!isOwner && <p className="text-xs text-white/30 mt-3">Only a Relay owner can connect repository credentials.</p>}
            {!githubApp?.registered && <details className="mt-5 border-t border-white/[0.06] pt-4"><summary className="cursor-pointer text-xs text-white/35 hover:text-white/60">Legacy personal-token connection</summary><div className="mt-3 flex flex-col sm:flex-row gap-2"><input type="password" autoComplete="new-password" value={token} onChange={(event) => setToken(event.target.value)} disabled={!isOwner || busy === "connect" || !connection?.secret_key_configured} placeholder="github_pat_…" className="text-input flex-1 font-mono" /><button type="button" onClick={handleConnect} disabled={!isOwner || !token.trim() || busy === "connect" || !connection?.secret_key_configured} className="ghost-btn px-5">{busy === "connect" ? "Connecting…" : "Use token"}</button></div></details>}
          </div>
          <div className="rounded-xl border border-white/[0.08] bg-black/30 p-5 font-mono text-xs">
            <div className="text-white/30 uppercase tracking-widest mb-3">App permissions</div>
            <div className="space-y-2 text-white/60">
              <div className="flex justify-between"><span>Contents</span><span className="text-relay-accent">read</span></div>
              <div className="flex justify-between"><span>Checks</span><span className="text-relay-accent">write</span></div>
              <div className="flex justify-between"><span>Access token</span><span className="text-emerald-300">1 hour</span></div>
            </div>
          </div>
        </div>
      ) : !workflow ? (
        <div className="rounded-xl border border-white/[0.08] bg-white/[0.02] p-5">
          <div className="eyebrow mb-1">Step 02</div>
          <h2 className="text-base font-semibold text-white">Link {project.name} to a repository</h2>
          <div className="grid grid-cols-1 md:grid-cols-[1fr_220px_auto] gap-3 mt-5 items-end">
            <label className="block"><span className="eyebrow block mb-1.5">Repository</span><select className="text-input" value={repoName} onChange={(event) => { const value = event.target.value; setRepoName(value); const repo = repositories.find((item) => item.full_name === value); if (repo) setProductionBranch(repo.default_branch || "main"); }} disabled={!isOwner || busy === "link"}>{repositories.map((repo) => <option key={repo.full_name} value={repo.full_name}>{repo.full_name}{repo.private ? " · private" : ""}</option>)}</select></label>
            <label className="block"><span className="eyebrow block mb-1.5">Production branch</span><input className="text-input font-mono" value={productionBranch} onChange={(event) => setProductionBranch(event.target.value)} disabled={!isOwner || busy === "link"} /></label>
            <button type="button" className="primary-btn h-10 px-5" onClick={handleLinkProject} disabled={!isOwner || !repoName || busy === "link"}>{busy === "link" ? "Installing…" : "Install workflow"}</button>
          </div>
          <p className="text-xs text-white/30 mt-3">Webhook: <span className="font-mono text-white/50">{githubApp?.webhook_url || connection?.webhook_url}</span></p>
        </div>
      ) : (
        <>
          <div className="rounded-xl border border-white/[0.08] bg-white/[0.02] p-5">
            <div className="flex items-start justify-between gap-4 mb-6">
              <div>
                <a href={workflow.html_url} target="_blank" rel="noreferrer" className="text-base font-semibold text-white hover:text-relay-accent transition-colors">{workflow.repo_full_name} ↗</a>
                <div className="text-xs text-white/35 mt-1">Production branch <span className="font-mono text-white/60">{workflow.production_branch}</span> · {workflow.auth_mode === "app" ? `installation #${workflow.installation_id}` : `webhook #${workflow.webhook_id}`}</div>
              </div>
              {isOwner && <button type="button" onClick={handleUnlink} disabled={busy === "unlink"} className="ghost-btn text-xs">{busy === "unlink" ? "Removing…" : "Disconnect repo"}</button>}
            </div>
            <WorkflowRail workflow={workflow} />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            <div className="rounded-xl border border-white/[0.08] bg-white/[0.02] p-5">
              <div className="eyebrow mb-1">Latest branch delivery</div>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="text-base font-semibold text-white">{latestPreview?.branch ?? "Waiting for a push"}</h2>
                  {latestPreview && <div className="text-xs text-white/35 mt-1">{latestPreview.pr_number ? `PR #${latestPreview.pr_number} · ` : ""}{latestPreview.head_sha ? formatCommitSHA(latestPreview.head_sha) : ""} · {timeAgo(new Date(latestPreview.updated_at).toISOString())} ago</div>}
                </div>
                <span className={cn("rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wider", latestPreview?.status === "success" ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-300" : latestPreview?.status === "failed" ? "border-red-500/30 bg-red-500/10 text-red-300" : "border-amber-500/30 bg-amber-500/10 text-amber-200")}>{latestPreview?.status ?? "idle"}</span>
              </div>
              {latestPreview?.preview_url ? <a href={latestPreview.preview_url} target="_blank" rel="noreferrer" className="mt-5 flex items-center justify-between rounded-lg border border-relay-accent/30 bg-relay-accent/10 px-4 py-3 text-sm text-relay-accent hover:bg-relay-accent/15 transition-colors"><span className="truncate">{latestPreview.preview_url}</span><span className="ml-3">Open ↗</span></a> : <div className="mt-5 rounded-lg border border-dashed border-white/10 px-4 py-3 text-xs text-white/30">The HTTPS preview appears here after readiness succeeds.</div>}
            </div>

            <div className="rounded-xl border border-white/[0.08] bg-white/[0.02] p-5">
              <div className="eyebrow mb-1">Production guard</div>
              <div className="flex items-start justify-between gap-3">
                <div><h2 className="text-base font-semibold text-white">{workflow.production.health_status}</h2><p className="text-xs text-white/40 mt-1 leading-relaxed">{workflow.production.health_detail}</p></div>
                <span className={cn("w-2.5 h-2.5 rounded-full mt-1.5 shrink-0", workflow.production.health_status === "healthy" ? "bg-emerald-400" : workflow.production.health_status === "unhealthy" ? "bg-red-400" : workflow.production.health_status === "deploying" ? "bg-amber-400 animate-pulse" : "bg-white/20")} />
              </div>
              <div className="flex flex-wrap gap-2 mt-5">
                {workflow.production.url && <a href={workflow.production.url} target="_blank" rel="noreferrer" className="ghost-btn">Open production ↗</a>}
                <button type="button" onClick={handleRollback} disabled={!workflow.production.rollback_available || busy === "rollback"} className="ghost-btn border-red-500/35 text-red-300 hover:bg-red-500/10 disabled:opacity-35">{busy === "rollback" ? "Restoring…" : "Rollback instantly"}</button>
              </div>
              <div className="text-[11px] text-white/25 mt-3">{workflow.production.rollback_available ? "Previous image is warm and ready to restore." : "Rollback becomes available after the second successful production image."}</div>
            </div>
          </div>
        </>
      )}

      {connection?.connected && !githubApp?.registered && isOwner && !workflow && <button type="button" onClick={handleDisconnectAccount} disabled={busy === "disconnect"} className="text-xs text-white/30 hover:text-red-300 transition-colors">Disconnect legacy GitHub token</button>}
    </div>
  );
}
