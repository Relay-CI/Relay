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
import { DeployDetailDialog } from "@/components/deploy-detail-dialog";
import {
  computeProjectStats,
  deployKey,
  computePreviewURL,
  normalizeProjects,
  type NormalizedProject,
} from "@/lib/relay-utils";
import type { Deploy, EnvInfo } from "@/lib/api";

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
    if (!projectOptions.length) { setSelectedProjectName(""); return; }
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
        const key = deployKey(selectedProject.name, envInfo.env, envInfo.branch);
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
    if (!envOptions.length) { setSelectedEnvKey(""); return; }
    if (!envOptions.find((e) => deployKey(e.app, e.env, e.branch) === selectedEnvKey)) {
      setSelectedEnvKey(deployKey(envOptions[0].app, envOptions[0].env, envOptions[0].branch));
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
      return (s as { env?: string }).env === selectedEnv?.env &&
        (s as { branch?: string }).branch === selectedEnv?.branch;
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

  async function handleRefresh() {
    setRefreshing(true);
    await refreshDashboard();
    setRefreshing(false);
  }

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

  if (auth.setupAvailable && !auth.authed) {
    return <SetupPage />;
  }

  if (!auth.authed) {
    return <LoginPage />;
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
            onUpdated={refreshDashboard}
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
    ? envMap.get(deployKey(selectedDeploy.app, selectedDeploy.env, selectedDeploy.branch)) ?? null
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
      />

      <div className="flex flex-1 min-h-0 overflow-hidden">
        <Sidebar
          activeTab={activeTab}
          onTabChange={setActiveTab}
          isOwner={isOwner}
          selectedProject={selectedProject}
          selectedEnv={selectedEnv}
          envOptions={envOptions}
          selectedEnvKey={selectedEnvKey}
          onEnvChange={setSelectedEnvKey}
          projectStats={projectStats}
        />

        <main className="flex-1 min-w-0 overflow-y-auto p-5">
          {!selectedProject && !dashboard.loading ? (
            <div className="flex flex-col items-center justify-center h-full text-center gap-3">
              <div className="eyebrow">No Projects Yet</div>
              <h2 className="text-xl font-semibold text-white/70">Deploy an app to populate the admin.</h2>
              <p className="text-sm text-white/40 max-w-sm">
                Projects appear as soon as Relay has app state. Trigger a deploy from the CLI to get started.
              </p>
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
        />
      )}
    </div>
  );
}
