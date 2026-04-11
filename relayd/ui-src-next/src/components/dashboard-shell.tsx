"use client";

import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/hooks/use-auth";
import { useDashboardData } from "@/hooks/use-dashboard-data";
import { LoginPage, SetupPage } from "@/components/auth-screens";
import { Topbar } from "@/components/topbar";
import { Sidebar } from "@/components/sidebar";
import { OverviewPage } from "@/components/pages/overview";
import { DeploymentsPage } from "@/components/pages/deployments";
import { BuildLogsPage } from "@/components/pages/build-logs";
import { RuntimeLogsPage } from "@/components/pages/runtime-logs";
import { SettingsPage } from "@/components/pages/settings";
import { AnalyticsPage } from "@/components/pages/analytics";
import { ServerSettingsPage } from "@/components/pages/server-settings";
import { UsersPage } from "@/components/pages/users";
import { AuditPage } from "@/components/pages/audit";
import { LanesPage } from "@/components/pages/lanes";
import { DeployDetailDialog } from "@/components/deploy-detail-dialog";
import {
  computeProjectStats,
  deployKey,
  computePreviewURL,
  normalizeProjects,
  type NormalizedProject,
} from "@/lib/relay-utils";
import type { Deploy, EnvInfo } from "@/lib/api";
import { cancelDeploy, cliStart, saveAppConfig } from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
  previewURL?: string;
}

export default function DashboardShell() {
  const auth = useAuth();
  const [dashboard, refreshDashboard] = useDashboardData(auth.authed);
  const [refreshing, setRefreshing] = useState(false);
  const [activeTab, setActiveTab] = useState("overview");
  const [authView, setAuthView] = useState<"login" | "setup">("login");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [createProjectOpen, setCreateProjectOpen] = useState(false);
  const [selectedProjectName, setSelectedProjectName] = useState("");
  const [selectedEnvKey, setSelectedEnvKey] = useState("");
  const [selectedDeploy, setSelectedDeploy] = useState<Deploy | null>(null);

  /* ── Derived project/env data ─────────────────────────────────────────── */

  const deploysByContext = useMemo(() => {
    const map = new Map<string, Deploy[]>();
    for (const deploy of dashboard.deploys) {
      const key = deployKey(deploy.app, deploy.env, deploy.branch);
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(deploy);
    }
    return map;
  }, [dashboard.deploys]);

  const projectOptions = useMemo<NormalizedProject[]>(() => {
    return normalizeProjects(dashboard.projects);
  }, [dashboard.projects, deploysByContext]);

  const envMap = useMemo(() => {
    const map = new Map<string, SelectedEnvMeta>();
    for (const project of projectOptions) {
      for (const envInfo of project.envs) {
        const key = deployKey(project.name, envInfo.env, envInfo.branch);
        const latestDeploy = deploysByContext.get(key)?.[0] ?? undefined;
        map.set(key, {
          ...envInfo,
          app: project.name,
          env: envInfo.env,
          branch: envInfo.branch,
          latestDeploy,
          previewURL: computePreviewURL(envInfo, latestDeploy ?? null),
        });
      }
    }
    return map;
  }, [projectOptions, deploysByContext]);

  // Auto-select first project
  useEffect(() => {
    if (!projectOptions.length) {
      setSelectedProjectName("");
      return;
    }
    if (!projectOptions.find((p) => p.name === selectedProjectName)) {
      setSelectedProjectName(projectOptions[0].name);
    }
  }, [projectOptions, selectedProjectName]);

  const selectedProject = useMemo(
    () => projectOptions.find((p) => p.name === selectedProjectName) ?? null,
    [projectOptions, selectedProjectName],
  );

  const envOptions = useMemo<SelectedEnvMeta[]>(() => {
    if (!selectedProject) return [];
    return selectedProject.envs
      .slice()
      .sort((a, b) => {
        if (a.env === b.env) return a.branch.localeCompare(b.branch);
        if (a.env === "prod") return -1;
        if (b.env === "prod") return 1;
        return a.env.localeCompare(b.env);
      })
      .map((envInfo) => {
        const key = deployKey(
          selectedProject.name,
          envInfo.env,
          envInfo.branch,
        );
        const latestDeploy = deploysByContext.get(key)?.[0] ?? undefined;
        return {
          ...envInfo,
          app: selectedProject.name,
          env: envInfo.env,
          branch: envInfo.branch,
          latestDeploy,
          previewURL: computePreviewURL(envInfo, latestDeploy ?? null),
        };
      });
  }, [selectedProject, deploysByContext]);

  // Auto-select first env when project changes
  useEffect(() => {
    if (!envOptions.length) {
      setSelectedEnvKey("");
      return;
    }
    if (
      !envOptions.find(
        (e) => deployKey(e.app, e.env, e.branch) === selectedEnvKey,
      )
    ) {
      setSelectedEnvKey(
        deployKey(envOptions[0].app, envOptions[0].env, envOptions[0].branch),
      );
    }
  }, [envOptions, selectedEnvKey]);

  const selectedEnv = useMemo(
    () => envMap.get(selectedEnvKey) ?? null,
    [envMap, selectedEnvKey],
  );

  const currentProjectDeploys = useMemo(() => {
    if (!selectedProject) return dashboard.deploys;
    return dashboard.deploys.filter((d) => d.app === selectedProject.name);
  }, [dashboard.deploys, selectedProject]);

  const services = useMemo(() => {
    if (!selectedProject) return [];
    return (selectedProject.services ?? []).filter((s) => {
      if (!selectedEnv) return true;
      return (
        (s as { env?: string }).env === selectedEnv?.env &&
        (s as { branch?: string }).branch === selectedEnv?.branch
      );
    });
  }, [selectedProject, selectedEnv]);

  const projectStats = useMemo(
    () => computeProjectStats(currentProjectDeploys),
    [currentProjectDeploys],
  );

  const selectedEnvDeploys = useMemo(() => {
    if (!selectedEnvKey) return [];
    return deploysByContext.get(selectedEnvKey) ?? [];
  }, [deploysByContext, selectedEnvKey]);

  const selectedEnvStats = useMemo(
    () => computeProjectStats(selectedEnvDeploys),
    [selectedEnvDeploys],
  );

  const isOwner = auth.user?.role === "owner";
  const isDeployer = auth.user?.role === "owner" || auth.user?.role === "admin" || auth.user?.role === "deployer";

  async function handleRefresh() {
    setRefreshing(true);
    await refreshDashboard();
    setRefreshing(false);
  }

  useEffect(() => {
    if (auth.authed) return;
    setAuthView(auth.setupAvailable && !auth.legacyMode ? "setup" : "login");
  }, [auth.authed, auth.setupAvailable, auth.legacyMode]);

  // If the user is already authenticated and the URL has CLI params, mint a
  // code via the existing session and redirect back to the local CLI server.
  useEffect(() => {
    if (!auth.authed) return;
    const params = new URLSearchParams(window.location.search);
    if (params.get("cli") !== "1") return;
    const parsedPort = Number.parseInt(params.get("port") ?? "", 10);
    if (!Number.isInteger(parsedPort) || parsedPort <= 0 || parsedPort > 65535) return;
    cliStart(parsedPort)
      .then((resp) => {
        if (resp.cli_redirect) window.location.assign(resp.cli_redirect);
      })
      .catch(() => {/* ignore – fall through to normal dashboard */});
  }, [auth.authed]);

  /* ── Auth gating ──────────────────────────────────────────────────────── */

  if (auth.loading) {
    return (
      <div className="min-h-screen bg-black flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-relay-accent border-t-transparent rounded-full animate-spin mx-auto mb-3" />
          <div className="eyebrow">Relayd Control Room</div>
          <div className="text-sm text-white/50 mt-1">Checking session…</div>
        </div>
      </div>
    );
  }

  if (auth.setupAvailable && !auth.authed && authView === "setup") {
    return (
      <SetupPage
        onShowLogin={auth.legacyMode ? () => setAuthView("login") : undefined}
        onSuccess={auth.refresh}
      />
    );
  }

  if (!auth.authed) {
    return (
      <LoginPage
        legacyMode={auth.legacyMode}
        showSetupOption={auth.setupAvailable}
        onShowSetup={
          auth.setupAvailable ? () => setAuthView("setup") : undefined
        }
        onSuccess={auth.refresh}
      />
    );
  }

  /* ── Main shell ───────────────────────────────────────────────────────── */

  function renderContent() {
    switch (activeTab) {
      case "overview":
        return (
          <OverviewPage
            project={selectedProject}
            contexts={envOptions}
            selectedEnv={selectedEnv}
            services={services}
            deploys={currentProjectDeploys}
            envMap={envMap}
            projectStats={projectStats}
            selectedEnvStats={selectedEnvStats}
            onOpenDeploy={setSelectedDeploy}
            onNavigateSettings={() => setActiveTab("settings")}
          />
        );
      case "deployments":
        return (
          <DeploymentsPage
            deploys={currentProjectDeploys}
            envMap={envMap}
            selectedEnv={selectedEnv}
            onOpenDeploy={setSelectedDeploy}
            onCancelDeploy={cancelDeploy}
          />
        );
      case "logs":
        return (
          <BuildLogsPage
            deploys={currentProjectDeploys}
            onOpenDeploy={setSelectedDeploy}
          />
        );
      case "runtime-logs":
        return <RuntimeLogsPage selectedEnv={selectedEnv} />;
      case "settings":
        return (
          <SettingsPage
            selectedEnv={selectedEnv}
            project={selectedProject}
            services={services}
            currentUser={auth.user}
            onUpdated={refreshDashboard}
          />
        );
      case "lanes":
        return (
          <LanesPage
            project={selectedProject}
            contexts={envOptions}
            isOwner={isOwner}
            isDeployer={isDeployer}
            onUpdated={refreshDashboard}
            onNavigateSettings={(envKey) => {
              setSelectedEnvKey(envKey);
              setActiveTab("settings");
            }}
          />
        );
      case "analytics":
        return <AnalyticsPage selectedEnv={selectedEnv} />;
      case "server":
        return isOwner ? <ServerSettingsPage currentUser={auth.user} /> : null;
      case "users":
        return isOwner ? <UsersPage currentUser={auth.user} /> : null;
      case "audit":
        return isOwner ? <AuditPage /> : null;
      default:
        return null;
    }
  }

  const selectedDeployEnvInfo = selectedDeploy
    ? (envMap.get(
        deployKey(
          selectedDeploy.app,
          selectedDeploy.env,
          selectedDeploy.branch,
        ),
      ) ?? null)
    : null;

  return (
    <div className="flex flex-col h-screen bg-black overflow-hidden">
      <Topbar
        projects={projectOptions}
        selectedProjectName={selectedProjectName}
        onSelectProject={(name) => {
          setSelectedProjectName(name);
          setSelectedEnvKey("");
        }}
        currentUser={auth.user}
        onLogout={auth.signOut}
        isLive={dashboard.isLive}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        onToggleSidebar={() => setSidebarOpen((v) => !v)}
        onCreateProject={() => setCreateProjectOpen(true)}
      />

      <div className="flex flex-1 min-h-0 overflow-hidden">
        <Sidebar
          activeTab={activeTab}
          onTabChange={setActiveTab}
          isOwner={isOwner}
          isDeployer={isDeployer}
          selectedProject={selectedProject}
          selectedEnv={selectedEnv}
          envOptions={envOptions}
          selectedEnvKey={selectedEnvKey}
          onEnvChange={setSelectedEnvKey}
          projectStats={projectStats}
          isOpen={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
        />

        <main className="flex-1 min-w-0 overflow-y-auto p-5">
          {!selectedProject && !dashboard.loading ? (
            <div className="flex flex-col items-center justify-center h-full text-center gap-3">
              <div className="eyebrow">No Projects Yet</div>
              <h2 className="text-xl font-semibold text-white/70">
                Deploy an app to populate the admin.
              </h2>
              <p className="text-sm text-white/40 max-w-sm">
                Projects appear as soon as Relay has app state. Trigger a deploy
                from the CLI to get started, or create a project shell here.
              </p>
              <button
                type="button"
                onClick={() => setCreateProjectOpen(true)}
                className="mt-2 flex items-center gap-2 px-4 py-2 rounded-md bg-relay-accent text-white text-sm font-semibold hover:bg-relay-accent/80 transition-colors"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                Create project
              </button>
            </div>
          ) : (
            renderContent()
          )}
        </main>
      </div>

      {selectedDeploy && (
        <DeployDetailDialog
          deploy={selectedDeploy}
          envInfo={selectedDeployEnvInfo}
          services={services}
          onClose={() => setSelectedDeploy(null)}
          onCancel={cancelDeploy}
        />
      )}

      {createProjectOpen && (
        <CreateProjectModal
          onClose={() => setCreateProjectOpen(false)}
          onCreated={(name) => {
            setCreateProjectOpen(false);
            refreshDashboard().then(() => setSelectedProjectName(name));
          }}
        />
      )}
    </div>
  );
}

/* ── Create Project Modal ─────────────────────────────────────────────────── */

interface CreateProjectModalProps {
  onClose: () => void;
  onCreated: (name: string) => void;
}

function CreateProjectModal({ onClose, onCreated }: CreateProjectModalProps) {
  const [name, setName] = useState("");
  const [env, setEnv] = useState("prod");
  const [branch, setBranch] = useState("main");
  const [engine, setEngine] = useState("docker");
  const [mode, setMode] = useState("port");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) { setError("Project name is required."); return; }
    setSaving(true);
    setError("");
    try {
      await saveAppConfig(
        { app: name.trim(), env, branch },
        { engine, traffic_mode: mode },
      );
      onCreated(name.trim());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-zinc-950 border border-white/[0.1] rounded-xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between p-5 border-b border-white/[0.06]">
          <h2 className="text-base font-semibold text-white">New project</h2>
          <button type="button" onClick={onClose} className="text-white/40 hover:text-white transition-colors">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 flex flex-col gap-4">
          <div>
            <label className="eyebrow block mb-1.5">Project name</label>
            <input
              type="text"
              className="w-full bg-white/[0.04] border border-white/[0.08] rounded px-3 py-2 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50"
              placeholder="my-app"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="eyebrow block mb-1.5">Default env</label>
              <select
                value={env}
                onChange={(e) => setEnv(e.target.value)}
                className="w-full bg-zinc-900 border border-white/[0.08] rounded px-3 py-2 text-sm text-white outline-none focus:border-relay-accent/50"
              >
                <option value="prod">prod</option>
                <option value="staging">staging</option>
                <option value="dev">dev</option>
                <option value="preview">preview</option>
              </select>
            </div>
            <div>
              <label className="eyebrow block mb-1.5">Branch</label>
              <input
                type="text"
                className="w-full bg-white/[0.04] border border-white/[0.08] rounded px-3 py-2 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50"
                placeholder="main"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="eyebrow block mb-1.5">Engine</label>
              <select
                value={engine}
                onChange={(e) => setEngine(e.target.value)}
                className="w-full bg-zinc-900 border border-white/[0.08] rounded px-3 py-2 text-sm text-white outline-none focus:border-relay-accent/50"
              >
                <option value="docker">docker</option>
                <option value="station">station</option>
              </select>
            </div>
            <div>
              <label className="eyebrow block mb-1.5">Mode</label>
              <select
                value={mode}
                onChange={(e) => setMode(e.target.value)}
                className="w-full bg-zinc-900 border border-white/[0.08] rounded px-3 py-2 text-sm text-white outline-none focus:border-relay-accent/50"
              >
                <option value="port">HTTP</option>
                <option value="static">Static</option>
              </select>
            </div>
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 rounded-md border border-white/[0.1] text-sm text-white/60 hover:text-white hover:bg-white/[0.06] transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="flex-1 px-4 py-2 rounded-md bg-relay-accent text-white text-sm font-semibold hover:bg-relay-accent/80 disabled:opacity-50 transition-colors"
            >
              {saving ? "Creating…" : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
