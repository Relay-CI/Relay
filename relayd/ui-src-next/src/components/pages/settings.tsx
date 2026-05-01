"use client";

import { useState, useEffect, useCallback } from "react";
import { cn } from "@/lib/utils";
import {
  buildSettingsConfig,
  envToText,
  formatDateTime,
  textToEnv,
  normalizeEngineValue,
  prettyCompanionType,
  defaultCompanionDraft,
  uiModeToApi,
  uiTrafficModeToApi,
  type NormalizedProject,
} from "@/lib/relay-utils";
import {
  getAppConfig,
  saveAppConfig,
  restartApp,
  startApp,
  stopApp,
  deleteLane,
  getSecrets,
  setSecret,
  setSecretsRaw,
  deleteSecret,
  getCompanions,
  saveCompanion,
  deleteCompanion,
  restartCompanion,
  deleteProject,
  generateSignedLink,
  createDeploy,
  getPromotions,
  requestPromotion,
  approvePromotion,
  rollback,
  type AppConfig,
  type Secret,
  type Companion,
  type CompanionConfig,
  type Service,
  type EnvInfo,
  type PromotionRecord,
  type SignedLinkResponse,
} from "@/lib/api";

interface SelectedEnvMeta extends EnvInfo {
  app: string;
  env: string;
  branch: string;
}

interface SettingsPageProps {
  selectedEnv: SelectedEnvMeta | null;
  project: NormalizedProject | null;
  services: Service[];
  currentUser: { username: string; role: string } | null;
  onUpdated: () => void;
}

const ENGINE_OPTIONS = [
  {
    value: "docker",
    title: "Docker",
    summary:
      "Production-ready default. Builds and runs the image with the full Relay feature set.",
  },
  {
    value: "station",
    title: "Station",
    summary:
      "Experimental runtime. Keep this for local/WSL workflows until Station is stable.",
  },
];

const MODE_OPTIONS = [
  {
    value: "http",
    title: "HTTP",
    summary: "Relay proxies internet traffic to the app container port.",
  },
  {
    value: "static",
    title: "Static",
    summary: "Serve a prebuilt static folder with no running container.",
  },
  {
    value: "off",
    title: "Off",
    summary:
      "Deploy-only mode. No live traffic — use to run CI/CD pipelines without exposing a route.",
  },
];

const POLICY_OPTIONS = [
  {
    value: "bluegreen",
    title: "Blue/Green",
    summary: "Swap traffic instantly between two stable containers.",
  },
  {
    value: "rolling",
    title: "Rolling",
    summary:
      "Drain the current container gradually as the new one becomes healthy.",
  },
];

const ACCESS_OPTIONS = [
  {
    value: "public",
    title: "Public",
    summary: "Anyone who reaches the route can access the app.",
  },
  {
    value: "relay-login",
    title: "Relay Login",
    summary:
      "Requires an authenticated Relay user session before the app is served.",
  },
  {
    value: "signed-link",
    title: "Signed Link",
    summary:
      "Only requests with a valid Relay signed-link token are allowed through.",
  },
  {
    value: "ip-allowlist",
    title: "IP Allowlist",
    summary: "Only requests from listed IPs or CIDR ranges are allowed.",
  },
];

export function SettingsPage({
  selectedEnv,
  project,
  services,
  currentUser,
  onUpdated,
}: SettingsPageProps) {
  const [config, setConfig] = useState<AppConfig>(
    buildSettingsConfig() as AppConfig,
  );
  const [busy, setBusy] = useState(false);
  const [rolloutBusy, setRolloutBusy] = useState("");
  const [notice, setNotice] = useState<{
    tone: "ok" | "warn";
    text: string;
  } | null>(null);
  const [signedLinkMinutes, setSignedLinkMinutes] = useState(24 * 60);
  const [signedLink, setSignedLink] = useState<SignedLinkResponse | null>(null);
  const [promotionBusy, setPromotionBusy] = useState(false);
  const [promotions, setPromotions] = useState<PromotionRecord[]>([]);
  const [promotionTargetKey, setPromotionTargetKey] = useState("");
  const [manualDeployBusy, setManualDeployBusy] = useState(false);
  const [manualDeploySource, setManualDeploySource] = useState<"workspace" | "git">("workspace");
  const [manualDeployTargetKey, setManualDeployTargetKey] = useState("");

  const [secrets, setSecrets] = useState<Secret[]>([]);
  const [draftSecret, setDraftSecret] = useState({ key: "", value: "" });
  const [rawSecretsText, setRawSecretsText] = useState("");
  const [rawSecretsReplace, setRawSecretsReplace] = useState(false);
  const [rawSecretsBusy, setRawSecretsBusy] = useState(false);

  const [companions, setCompanions] = useState<Companion[]>([]);
  const [companionBusy, setCompanionBusy] = useState(false);
  const [selectedCompanionName, setSelectedCompanionName] = useState("");
  const [companionDraft, setCompanionDraft] = useState<CompanionConfig>(
    () => defaultCompanionDraft("postgres") as CompanionConfig,
  );
  const [companionEnvText, setCompanionEnvText] = useState("");
  const [companionVolumesText, setCompanionVolumesText] = useState("");

  const [deleteText, setDeleteText] = useState("");
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [deleteError, setDeleteError] = useState("");

  const load = useCallback(async () => {
    if (!selectedEnv) return;
    try {
      const [cfg, secs, comps, promotionItems] = await Promise.all([
        getAppConfig(selectedEnv),
        getSecrets(selectedEnv),
        getCompanions(selectedEnv),
        getPromotions(selectedEnv.app, selectedEnv.env, selectedEnv.branch),
      ]);
      const normalized = buildSettingsConfig(cfg) as AppConfig;
      setConfig(normalized);
      setSecrets(secs ?? []);
      setCompanions(comps ?? []);
      setPromotions(promotionItems ?? []);
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Failed to load settings: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    }
  }, [selectedEnv?.app, selectedEnv?.env, selectedEnv?.branch]);

  useEffect(() => {
    load();
  }, [load]);

  const promotionTargets = (project?.envs ?? [])
    .filter((envInfo) => {
      if (!selectedEnv) return false;
      return !(envInfo.env === selectedEnv.env && envInfo.branch === selectedEnv.branch);
    })
    .map((envInfo) => ({
      key: `${envInfo.env}::${envInfo.branch}`,
      env: envInfo.env,
      branch: envInfo.branch,
      label: `${envInfo.env}/${envInfo.branch}`,
    }));

  useEffect(() => {
    if (!selectedEnv) {
      setPromotionTargetKey("");
      setManualDeployTargetKey("");
      return;
    }
    if (promotionTargets.find((item) => item.key === promotionTargetKey)) {
      return;
    }
    const preferred =
      promotionTargets.find(
        (item) => item.env === "prod" && item.branch === selectedEnv.branch,
      ) ??
      promotionTargets.find((item) => item.env === "prod") ??
      promotionTargets.find(
        (item) => item.env === "staging" && item.branch === selectedEnv.branch,
      ) ??
      promotionTargets[0];
    setPromotionTargetKey(preferred?.key ?? "");
  }, [selectedEnv?.env, selectedEnv?.branch, promotionTargets.map((item) => item.key).join("|")]);

  useEffect(() => {
    if (!selectedEnv) {
      setManualDeployTargetKey("");
      return;
    }
    const availableTargets = [
      { key: `${selectedEnv.env}::${selectedEnv.branch}` },
      ...promotionTargets,
    ];
    if (availableTargets.find((item) => item.key === manualDeployTargetKey)) {
      return;
    }
    setManualDeployTargetKey(`${selectedEnv.env}::${selectedEnv.branch}`);
  }, [selectedEnv?.env, selectedEnv?.branch, promotionTargets.map((item) => item.key).join("|"), manualDeployTargetKey]);

  function upd(patch: Partial<AppConfig>) {
    setConfig((c) => ({ ...c, ...patch }));
  }

  function toApiPayload(cfg: AppConfig) {
    const publicHosts = normalizeHostEntries(cfg.public_hosts ?? []);
    return {
      ...cfg,
      mode: uiModeToApi(cfg.mode),
      traffic_mode: uiTrafficModeToApi(cfg.traffic_mode),
      host_port: Number(cfg.host_port) || 0,
      service_port: Number(cfg.service_port) || 0,
      public_host: publicHosts[0] ?? "",
      public_hosts: publicHosts,
    };
  }

  async function handleSave() {
    if (!selectedEnv || !canWrite) return;
    setBusy(true);
    try {
      const payload = toApiPayload(config);
      await saveAppConfig(selectedEnv, payload);
      setNotice({
        tone: "warn",
        text: "Saved. The next deploy picks up this config automatically. Use Restart to apply it to the running container now.",
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Save failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveRestart() {
    if (!selectedEnv || !canWrite) return;
    setBusy(true);
    try {
      const payload = toApiPayload(config);
      await saveAppConfig(selectedEnv, payload);
      await restartApp(selectedEnv);
      setNotice({
        tone: "ok",
        text: "Saved and restart sent. Relay is applying the updated route and traffic policy now.",
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Apply failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy(false);
    }
  }

  async function handleAddSecret() {
    if (!selectedEnv || !canWrite || !draftSecret.key || !draftSecret.value) return;
    try {
      await setSecret(selectedEnv, draftSecret.key, draftSecret.value);
      const next = await getSecrets(selectedEnv);
      setSecrets(next ?? []);
      setDraftSecret({ key: "", value: "" });
      setNotice({ tone: "ok", text: `Saved ${draftSecret.key}` });
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Failed to save secret: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    }
  }

  async function handleDeleteSecret(key: string) {
    if (!selectedEnv || !canWrite) return;
    try {
      await deleteSecret(selectedEnv, key);
      const next = await getSecrets(selectedEnv);
      setSecrets(next ?? []);
      setNotice({ tone: "ok", text: `Deleted ${key}` });
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Failed to delete secret: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    }
  }

  async function handleSaveRawSecrets() {
    if (!selectedEnv || !canWrite || !rawSecretsText.trim()) return;
    setRawSecretsBusy(true);
    try {
      const result = await setSecretsRaw(selectedEnv, rawSecretsText, rawSecretsReplace);
      const next = await getSecrets(selectedEnv);
      setSecrets(next ?? []);
      setRawSecretsText("");
      setNotice({
        tone: "ok",
        text: `Saved ${result.count ?? 0} env var${(result.count ?? 0) === 1 ? "" : "s"} from raw input.`,
      });
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Failed to save raw env: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setRawSecretsBusy(false);
    }
  }

  function hydrateCompanion(c: Companion) {
    const draft = {
      ...defaultCompanionDraft(c.config.type ?? "custom"),
      ...c.config,
    };
    setSelectedCompanionName(c.config.name ?? "");
    setCompanionDraft(draft);
    setCompanionEnvText(envToText(draft.env));
    setCompanionVolumesText((draft.volumes ?? []).join("\n"));
  }

  function startNewCompanion(kind: string) {
    const draft = defaultCompanionDraft(kind);
    setSelectedCompanionName("");
    setCompanionDraft(draft);
    setCompanionEnvText(envToText(draft.env));
    setCompanionVolumesText((draft.volumes ?? []).join("\n"));
  }

  async function handleSaveCompanion() {
    if (!selectedEnv || !canWrite) return;
    setCompanionBusy(true);
    try {
      const payload = {
        ...companionDraft,
        env: textToEnv(companionEnvText),
        volumes: companionVolumesText
          .split(/\r?\n/)
          .map((l) => l.trim())
          .filter(Boolean),
      };
      await saveCompanion(selectedEnv, payload);
      const next = await getCompanions(selectedEnv);
      setCompanions(next ?? []);
      await onUpdated();
    } finally {
      setCompanionBusy(false);
    }
  }

  async function handleDeleteCompanion(name: string) {
    if (!selectedEnv || !canWrite) return;
    setCompanionBusy(true);
    try {
      await deleteCompanion(selectedEnv, name);
      const next = await getCompanions(selectedEnv);
      setCompanions(next ?? []);
      await onUpdated();
    } finally {
      setCompanionBusy(false);
    }
  }

  async function handleRestartCompanion(name: string) {
    if (!selectedEnv || !canWrite) return;
    setCompanionBusy(true);
    try {
      await restartCompanion(selectedEnv, name);
    } finally {
      setCompanionBusy(false);
    }
  }

  async function handleDeleteProject() {
    if (!project || deleteText !== project.name) return;
    setDeleteBusy(true);
    setDeleteError("");
    try {
      await deleteProject(project.name);
      setDeleteText("");
      await onUpdated();
    } catch (err) {
      setDeleteError(
        err instanceof Error ? err.message : "Failed to delete project",
      );
    } finally {
      setDeleteBusy(false);
    }
  }

  async function handleGenerateSignedLink() {
    if (!selectedEnv || !canWrite) return;
    setBusy(true);
    try {
      const result = await generateSignedLink(selectedEnv, signedLinkMinutes);
      setSignedLink(result);
      setNotice({
        tone: "ok",
        text: "Signed link generated. Anyone with the URL can access this lane until it expires.",
      });
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Signed-link generation failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy(false);
    }
  }

  async function handleDeleteLane() {
    if (!selectedEnv || !canWrite) return;
    const laneLabel = `${selectedEnv.app}/${selectedEnv.env}/${selectedEnv.branch}`;
    const confirmed = window.confirm(
      `Delete lane ${laneLabel}?\n\nThis permanently removes runtime, workspace, deploy data, and lane state for this lane.`,
    );
    if (!confirmed) return;

    const typed = window.prompt(
      `Type DELETE to confirm lane deletion for ${laneLabel}.`,
      "",
    );
    if ((typed ?? "").trim() !== "DELETE") {
      setNotice({
        tone: "warn",
        text: "Lane deletion cancelled. Confirmation text did not match DELETE.",
      });
      return;
    }

    setBusy(true);
    try {
      await deleteLane(selectedEnv);
      setNotice({
        tone: "ok",
        text: `Lane ${laneLabel} was deleted permanently.`,
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Delete lane failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy(false);
    }
  }

  async function handleRequestPromotion() {
    if (!selectedEnv || !canWrite) return;
    const target = promotionTargets.find((item) => item.key === promotionTargetKey);
    setPromotionBusy(true);
    try {
      await requestPromotion({
        app: selectedEnv.app,
        source_env: selectedEnv.env,
        source_branch: selectedEnv.branch,
        target_env: target?.env,
        target_branch: target?.branch,
      });
      setPromotions(
        await getPromotions(
          selectedEnv.app,
          selectedEnv.env,
          selectedEnv.branch,
        ),
      );
      setNotice({
        tone: "ok",
        text:
          currentUser?.role === "owner"
            ? `Promotion started. Relay is moving ${selectedEnv.env}/${selectedEnv.branch} into ${target?.label ?? "the selected target lane"} now.`
            : `Promotion request created for ${target?.label ?? "the selected target lane"}. An owner can approve it from this lane.`,
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Promotion failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setPromotionBusy(false);
    }
  }

  async function handleApprovePromotion(id: string) {
    if (!selectedEnv) return;
    setPromotionBusy(true);
    try {
      await approvePromotion(id);
      setPromotions(
        await getPromotions(
          selectedEnv.app,
          selectedEnv.env,
          selectedEnv.branch,
        ),
      );
      setNotice({
        tone: "ok",
        text: "Promotion approved. Relay queued the production deployment and health watch.",
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Approval failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setPromotionBusy(false);
    }
  }

  async function handleRollbackLane() {
    if (!selectedEnv || !canWrite) return;
    const confirmed = window.confirm(
      `Queue a rollback for ${selectedEnv.app}/${selectedEnv.env}/${selectedEnv.branch}? Relay will re-activate the previous image for this lane.`,
    );
    if (!confirmed) return;

    setRolloutBusy("rollback");
    try {
      await rollback(selectedEnv);
      setNotice({
        tone: "ok",
        text: "Rollback queued. Relay is restoring the previous image for this lane now.",
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Rollback failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setRolloutBusy("");
    }
  }

  async function handlePromoteAllTraffic() {
    if (!selectedEnv || !canWrite) return;
    setRolloutBusy("promote-traffic");
    try {
      const next = { ...toApiPayload(config), traffic_split_percent: 100 };
      await saveAppConfig(selectedEnv, next);
      upd({ traffic_split_percent: 100 });
      setNotice({
        tone: "ok",
        text: "Traffic split set to 100%. Relay will treat this lane as a full cutover instead of a canary.",
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Traffic promotion failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setRolloutBusy("");
    }
  }

  async function handleManualDeploy() {
    if (!selectedEnv || !canWrite) return;
    const [targetEnv, targetBranch] = manualDeployTargetKey.split("::");
    const payload = {
      app: selectedEnv.app,
      env: targetEnv || selectedEnv.env,
      branch: targetBranch || selectedEnv.branch,
      repo_url: (config.repo_url ?? "").trim(),
      source: manualDeploySource,
      workspace_env: selectedEnv.env,
      workspace_branch: selectedEnv.branch,
    };
    if (manualDeploySource === "git" && !payload.repo_url) {
      setNotice({
        tone: "warn",
        text: "Manual git deploy needs a repository URL on this lane first.",
      });
      return;
    }
    setManualDeployBusy(true);
    try {
      await createDeploy(payload);
      setNotice({
        tone: "ok",
        text:
          manualDeploySource === "workspace"
            ? `Workspace deploy queued for ${payload.env}/${payload.branch}. Relay is rebuilding from the current ${selectedEnv.env}/${selectedEnv.branch} workspace now.`
            : `Git deploy queued for ${payload.env}/${payload.branch} from ${payload.repo_url || "the saved repository"}.`,
      });
      await onUpdated();
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Manual deploy failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setManualDeployBusy(false);
    }
  }

  if (!selectedEnv) {
    return (
      <div className="flex items-center justify-center h-full text-white/30 text-sm">
        Select an environment to edit settings
      </div>
    );
  }

  const draftEngine = normalizeEngineValue(config.engine ?? "docker");
  const isOwner = currentUser?.role === "owner";
  const canWrite = ["owner", "admin", "deployer"].includes(
    selectedEnv.access_role ?? currentUser?.role ?? "",
  );
  const laneExpiresAt = Number(
    config.expires_at ?? selectedEnv.expires_at ?? 0,
  );
  const pendingPromotion = promotions.find(
    (item) => item.status === "pending_approval",
  );
  const selectedPromotionTarget =
    promotionTargets.find((item) => item.key === promotionTargetKey) ?? null;
  const manualDeployTargets = [
    {
      key: `${selectedEnv.env}::${selectedEnv.branch}`,
      label: `${selectedEnv.env}/${selectedEnv.branch} (current lane)`,
    },
    ...promotionTargets.map((item) => ({ key: item.key, label: item.label })),
  ];

  return (
    <div className="space-y-6">
      <div>
        <div className="eyebrow mb-0.5">Configuration</div>
        <h1 className="text-xl font-semibold text-white">
          Settings — {selectedEnv.env}/{selectedEnv.branch}
        </h1>
      </div>

      {notice && (
        <div
          className={cn(
            "rounded-lg px-4 py-3 text-sm border",
            notice.tone === "ok"
              ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
              : "bg-amber-500/10 border-amber-500/30 text-amber-400",
          )}
        >
          {notice.text}
        </div>
      )}

      {!canWrite && (
        <div className="rounded-lg px-4 py-3 text-sm border bg-white/[0.03] border-white/[0.08] text-white/65">
          This lane is read-only for your account. An owner can grant deploy or admin access for {selectedEnv.app}/{selectedEnv.env}.
        </div>
      )}

      {/* Runtime / Routing */}
      <SectionCard title="Runtime / Routing" eyebrow="Server controls">
        <div className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <SegmentCard
              label="Runtime Engine"
              description={
                (
                  ENGINE_OPTIONS.find((o) => o.value === draftEngine) ??
                  ENGINE_OPTIONS[0]
                ).summary
              }
            >
              {ENGINE_OPTIONS.map((o) => (
                <SegButton
                  key={o.value}
                  active={draftEngine === o.value}
                  onClick={() => canWrite && upd({ engine: o.value })}
                >
                  {o.title}
                </SegButton>
              ))}
            </SegmentCard>
            <SegmentCard
              label="Routing Mode"
              description={
                (
                  MODE_OPTIONS.find((o) => o.value === config.mode) ??
                  MODE_OPTIONS[0]
                ).summary
              }
            >
              {MODE_OPTIONS.map((o) => (
                <SegButton
                  key={o.value}
                  active={config.mode === o.value}
                  onClick={() => canWrite && upd({ mode: o.value })}
                >
                  {o.title}
                </SegButton>
              ))}
            </SegmentCard>
            <SegmentCard
              label="Traffic Policy"
              description={
                (
                  POLICY_OPTIONS.find((o) => o.value === config.traffic_mode) ??
                  POLICY_OPTIONS[0]
                ).summary
              }
            >
              {POLICY_OPTIONS.map((o) => (
                <SegButton
                  key={o.value}
                  active={config.traffic_mode === o.value}
                  onClick={() => canWrite && upd({ traffic_mode: o.value })}
                >
                  {o.title}
                </SegButton>
              ))}
            </SegmentCard>
          </div>
          <SegmentCard
            label="Edge Access"
            description={
              (
                ACCESS_OPTIONS.find((o) => o.value === config.access_policy) ??
                ACCESS_OPTIONS[0]
              ).summary
            }
          >
            <div className="flex flex-wrap gap-2">
              {ACCESS_OPTIONS.map((o) => (
                <SegButton
                  key={o.value}
                  active={config.access_policy === o.value}
                  onClick={() => canWrite && upd({ access_policy: o.value })}
                >
                  {o.title}
                </SegButton>
              ))}
            </div>
          </SegmentCard>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <Field label="Public Hosts">
              <HostChipsInput
                value={normalizeHostEntries(config.public_hosts ?? (config.public_host ? [config.public_host] : []))}
                onChange={(hosts) => upd({ public_host: hosts[0] ?? "", public_hosts: hosts })}
                placeholder="app.yourdomain.com"
                disabled={!canWrite}
              />
            </Field>
            <Field label="Host Port">
              <input
                type="number"
                className="text-input"
                value={config.host_port || ""}
                onChange={(e) => upd({ host_port: Number(e.target.value) })}
                placeholder="auto"
                disabled={!canWrite}
              />
              <p className="text-[11px] text-white/30 mt-1">Leave 0 / blank to auto-assign. Relay resolves conflicts automatically.</p>
            </Field>
            <Field label="Service Port">
              <input
                type="number"
                className="text-input"
                value={config.service_port || ""}
                onChange={(e) => upd({ service_port: Number(e.target.value) })}
                placeholder="auto (3000)"
                disabled={!canWrite}
              />
              <p className="text-[11px] text-white/30 mt-1">Port the app listens on inside the container. 0 defaults to 3000. Set <code className="font-mono">PORT</code> env var via app.</p>
            </Field>
          </div>
          <Field label="IP Allowlist">
            <textarea
              className="text-input min-h-[88px] resize-y"
              value={config.ip_allowlist ?? ""}
              onChange={(e) => upd({ ip_allowlist: e.target.value })}
              placeholder={"203.0.113.10\n198.51.100.0/24"}
              disabled={!canWrite}
            />
          </Field>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <Field label="New Traffic Share (%)">
              <input
                type="number"
                min={1}
                max={100}
                className="text-input"
                value={config.traffic_split_percent ?? 100}
                onChange={(e) => upd({ traffic_split_percent: Number(e.target.value) || 100 })}
                disabled={!canWrite}
              />
            </Field>
            <Field label="Assess After (sec)">
              <input
                type="number"
                min={30}
                className="text-input"
                value={config.rollout_assess_seconds ?? 300}
                onChange={(e) => upd({ rollout_assess_seconds: Number(e.target.value) || 300 })}
                disabled={!canWrite}
              />
            </Field>
            <Field label="Rollback Above (%)">
              <input
                type="number"
                min={1}
                max={100}
                className="text-input"
                value={config.rollout_error_percent ?? 5}
                onChange={(e) => upd({ rollout_error_percent: Number(e.target.value) || 5 })}
                disabled={!canWrite}
              />
            </Field>
          </div>
          <Field label="Minimum Requests Before Decision">
            <input
              type="number"
              min={1}
              className="text-input"
              value={config.rollout_min_requests ?? 25}
              onChange={(e) => upd({ rollout_min_requests: Number(e.target.value) || 25 })}
              disabled={!canWrite}
            />
          </Field>
          <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 space-y-3">
            <div>
              <div className="eyebrow mb-0.5">Live rollout policy</div>
              <div className="text-sm font-medium text-white">
                {config.traffic_split_percent ?? 100}% new traffic, {config.rollout_min_requests ?? 25} request minimum, rollback above {config.rollout_error_percent ?? 5}% errors
              </div>
              <div className="text-xs text-white/35 mt-1">
                Relay monitors canary exposure for {config.rollout_assess_seconds ?? 300} seconds before deciding whether to continue or revert.
              </div>
            </div>
            <div className="flex gap-2 flex-wrap">
              <button
                type="button"
                onClick={handlePromoteAllTraffic}
                disabled={rolloutBusy === "promote-traffic" || !canWrite || (config.traffic_split_percent ?? 100) >= 100}
                className="ghost-btn"
              >
                {rolloutBusy === "promote-traffic" ? "Applying..." : "Promote All Traffic"}
              </button>
              <button
                type="button"
                onClick={handleRollbackLane}
                disabled={rolloutBusy === "rollback" || !canWrite}
                className="ghost-btn border-red-500/40 text-red-300 hover:bg-red-500/10"
              >
                {rolloutBusy === "rollback" ? "Queueing..." : "Rollback Lane"}
              </button>
            </div>
          </div>
          {(config.rollout_status || selectedEnv.rollout_status) && (
            <div className="text-xs text-white/35">
              Rollout status:{" "}
              <span className="text-white/70">
                {config.rollout_status ?? selectedEnv.rollout_status}
              </span>
              {(config.rollout_started_at ?? selectedEnv.rollout_started_at) ? (
                <>
                  {" "}since{" "}
                  <span className="text-white/70">
                    {formatDateTime(new Date(Number(config.rollout_started_at ?? selectedEnv.rollout_started_at)).toISOString())}
                  </span>
                </>
              ) : null}
            </div>
          )}
          {laneExpiresAt > 0 &&
            (selectedEnv.env === "dev" || selectedEnv.env === "preview") && (
              <div className="text-xs text-white/35">
                Auto-expiry is active for this lane. Relay will stop it on{" "}
                <span className="text-white/70">
                  {formatDateTime(new Date(laneExpiresAt).toISOString())}
                </span>{" "}
                unless you redeploy or restart it first.
              </div>
            )}
          {config.access_policy === "signed-link" && (
            <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 space-y-3">
              <div>
                <div className="eyebrow mb-0.5">Signed access</div>
                <div className="text-sm font-medium text-white">
                  Create a shareable signed link
                </div>
                <div className="text-xs text-white/35 mt-1">
                  Relay will append a short-lived token that the edge middleware
                  verifies before serving the app.
                </div>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-[180px_1fr] gap-3">
                <Field label="Expires in (minutes)">
                  <input
                    type="number"
                    className="text-input"
                    value={signedLinkMinutes}
                    min={1}
                    max={7 * 24 * 60}
                    onChange={(e) =>
                      setSignedLinkMinutes(Number(e.target.value) || 60)
                    }
                  />
                </Field>
                <div className="flex items-end">
                  <button
                    type="button"
                    onClick={handleGenerateSignedLink}
                    disabled={busy || !canWrite}
                    className="primary-btn"
                  >
                    {busy ? "Generating..." : "Generate Signed Link"}
                  </button>
                </div>
              </div>
              {signedLink && (
                <div className="rounded border border-white/[0.06] bg-black/20 p-3 space-y-2">
                  <div className="text-xs text-white/35">
                    Expires{" "}
                    {formatDateTime(
                      new Date(signedLink.expires_at * 1000).toISOString(),
                    )}
                  </div>
                  <div className="text-sm text-white break-all">
                    {signedLink.url}
                  </div>
                </div>
              )}
            </div>
          )}
          <div className="flex gap-2 flex-wrap">
            <button
              type="button"
              onClick={handleSaveRestart}
              disabled={busy || !canWrite}
              className="primary-btn"
            >
              {busy ? "Working..." : "Apply & Restart"}
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={busy || !canWrite}
              className="ghost-btn"
            >
              Save for Later
            </button>
            <button
              type="button"
              onClick={() => {
                if (!selectedEnv) return;
                setBusy(true);
                stopApp(selectedEnv).finally(() => {
                  setBusy(false);
                  onUpdated();
                });
              }}
              disabled={busy || !canWrite}
              className="ghost-btn"
            >
              Stop App
            </button>
            <button
              type="button"
              onClick={() => {
                if (!selectedEnv) return;
                setBusy(true);
                startApp(selectedEnv).finally(() => {
                  setBusy(false);
                  onUpdated();
                });
              }}
              disabled={busy || !canWrite}
              className="ghost-btn"
            >
              Start App
            </button>
            <button
              type="button"
              onClick={handleDeleteLane}
              disabled={busy || !canWrite}
              className="ghost-btn border-red-500/40 text-red-300 hover:bg-red-500/10"
            >
              Delete Lane
            </button>
          </div>
          <div className="text-xs text-white/35">
            Delete Lane permanently removes this lane&apos;s runtime, workspace, and lane data.
          </div>
        </div>
      </SectionCard>

      {(selectedEnv.env !== "prod" || promotions.length > 0) && (
        <SectionCard title="Rollout & Promotion" eyebrow="Promote, approve, revert">
          <div className="space-y-3">
            <p className="text-sm text-white/50">
              Promote the currently healthy{" "}
              <strong className="text-white">{selectedEnv.env}</strong> lane
              into another lane. Relay keeps an audit trail, waits for approval
              when required, and queues rollback if post-promote health fails.
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <MiniMetric
                label="Canary split"
                value={`${config.traffic_split_percent ?? selectedEnv.traffic_split_percent ?? 100}%`}
              />
              <MiniMetric
                label="Assess window"
                value={`${config.rollout_assess_seconds ?? selectedEnv.rollout_assess_seconds ?? 300}s`}
              />
              <MiniMetric
                label="Rollback gate"
                value={`${config.rollout_error_percent ?? selectedEnv.rollout_error_percent ?? 5}% / ${config.rollout_min_requests ?? selectedEnv.rollout_min_requests ?? 25} req`}
              />
            </div>
            <div className="flex gap-2 flex-wrap">
              {selectedEnv.env !== "prod" && (
                <Field label="Target lane" className="min-w-[220px]">
                  <select
                    className="text-input"
                    value={promotionTargetKey}
                    onChange={(e) => setPromotionTargetKey(e.target.value)}
                    disabled={!canWrite || promotionBusy}
                  >
                    {promotionTargets.map((target) => (
                      <option key={target.key} value={target.key}>
                        {target.label}
                      </option>
                    ))}
                  </select>
                </Field>
              )}
            </div>
            <div className="flex gap-2 flex-wrap">
              {selectedEnv.env !== "prod" && (
                <button
                  type="button"
                  onClick={handleRequestPromotion}
                  disabled={promotionBusy || !canWrite || !selectedPromotionTarget}
                  className="primary-btn"
                >
                  {promotionBusy
                    ? "Working..."
                    : isOwner
                      ? `Promote To ${selectedPromotionTarget?.env ?? "target"}`
                      : "Request Promotion"}
                </button>
              )}
              {isOwner && pendingPromotion && (
                <button
                  type="button"
                  onClick={() => handleApprovePromotion(pendingPromotion.id)}
                  disabled={promotionBusy}
                  className="ghost-btn"
                >
                  {promotionBusy ? "Approving..." : "Approve Pending Request"}
                </button>
              )}
              <button
                type="button"
                onClick={handleRollbackLane}
                disabled={rolloutBusy === "rollback" || !canWrite}
                className="ghost-btn border-red-500/40 text-red-300 hover:bg-red-500/10"
              >
                {rolloutBusy === "rollback" ? "Queueing..." : "Rollback This Lane"}
              </button>
            </div>
            {!promotions.length ? (
              <div className="text-sm text-white/25">
                No promotion activity recorded for this lane yet.
              </div>
            ) : (
              <div className="space-y-2">
                {promotions.map((promotion) => (
                  <div
                    key={promotion.id}
                    className="rounded border border-white/[0.06] bg-white/[0.02] px-3 py-2.5"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <div className="text-sm font-medium text-white">
                          {promotion.source_env}/{promotion.source_branch} →{" "}
                          {promotion.target_env}/{promotion.target_branch}
                        </div>
                        <div className="text-xs text-white/35 mt-0.5">
                          Requested by {promotion.requested_by || "unknown"} on{" "}
                          {formatDateTime(
                            new Date(promotion.requested_at).toISOString(),
                          )}
                        </div>
                      </div>
                      <div className="text-xs uppercase tracking-wide text-white/45">
                        {promotion.status.replaceAll("_", " ")}
                      </div>
                    </div>
                    {(promotion.health_status ||
                      promotion.health_detail ||
                      promotion.target_deploy_id ||
                      promotion.rollback_deploy_id) && (
                      <div className="text-xs text-white/35 mt-2 space-y-1">
                        {promotion.health_status && (
                          <div>
                            Health:{" "}
                            <span className="text-white/65">
                              {promotion.health_status}
                            </span>
                          </div>
                        )}
                        {promotion.health_detail && (
                          <div>{promotion.health_detail}</div>
                        )}
                        {promotion.target_deploy_id && (
                          <div>
                            Deploy:{" "}
                            <span className="font-mono text-white/65">
                              {promotion.target_deploy_id}
                            </span>
                          </div>
                        )}
                        {promotion.rollback_deploy_id && (
                          <div>
                            Rollback:{" "}
                            <span className="font-mono text-white/65">
                              {promotion.rollback_deploy_id}
                            </span>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </SectionCard>
      )}

      <SectionCard title="Build layout" eyebrow="Monorepo + Dockerfile">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <Field label="Project Root">
            <input
              className="text-input font-mono text-xs"
              value={config.project_root ?? ""}
              onChange={(e) => upd({ project_root: e.target.value })}
              placeholder="apps/api"
              disabled={!canWrite}
            />
          </Field>
          <Field label="Build Context">
            <input
              className="text-input font-mono text-xs"
              value={config.build_context ?? ""}
              onChange={(e) => upd({ build_context: e.target.value })}
              placeholder="."
              disabled={!canWrite}
            />
          </Field>
          <Field label="Dockerfile">
            <input
              className="text-input font-mono text-xs"
              value={config.dockerfile ?? ""}
              onChange={(e) => upd({ dockerfile: e.target.value })}
              placeholder="apps/api/Dockerfile"
              disabled={!canWrite}
            />
          </Field>
        </div>
        <p className="text-xs text-white/35">
          All three paths are optional and repo-relative. Leave them blank for automatic detection. Set only the pieces you need for monorepo apps.
        </p>
        <div>
          <button
            type="button"
            onClick={handleSave}
            disabled={busy || !canWrite}
            className="primary-btn"
          >
            {busy ? "Saving..." : "Save Build Layout"}
          </button>
        </div>
      </SectionCard>

      <SectionCard title="Manual Deploy" eyebrow="Rebuild or redeploy on demand">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <Field label="Deploy Source">
            <select
              className="text-input"
              value={manualDeploySource}
              onChange={(e) =>
                setManualDeploySource(e.target.value === "git" ? "git" : "workspace")
              }
              disabled={!canWrite || manualDeployBusy}
            >
              <option value="workspace">Current workspace</option>
              <option value="git">Saved repo_url + target branch</option>
            </select>
          </Field>
          <Field label="Target Lane">
            <select
              className="text-input"
              value={manualDeployTargetKey}
              onChange={(e) => setManualDeployTargetKey(e.target.value)}
              disabled={!canWrite || manualDeployBusy}
            >
              {manualDeployTargets.map((target) => (
                <option key={target.key} value={target.key}>
                  {target.label}
                </option>
              ))}
            </select>
          </Field>
        </div>
        <p className="text-xs text-white/35">
          Workspace deploys rebuild from the current lane&apos;s saved server workspace. Git deploys refresh the selected target lane from its saved repository and branch.
        </p>
        <div>
          <button
            type="button"
            onClick={handleManualDeploy}
            disabled={!canWrite || manualDeployBusy}
            className="primary-btn"
          >
            {manualDeployBusy ? "Queueing..." : "Queue Manual Deploy"}
          </button>
        </div>
      </SectionCard>

      {/* GitHub / Webhooks */}
      <SectionCard title="GitHub / Webhooks" eyebrow="Per-project integration">
        <div className="space-y-3">
          <Field label="Repository URL">
            <input
              className="text-input"
              value={config.repo_url ?? ""}
              onChange={(e) => upd({ repo_url: e.target.value })}
              disabled={!canWrite}
            />
          </Field>
          <Field label="Webhook Secret">
            <input
              type="password"
              autoComplete="new-password"
              className="text-input"
              value={config.webhook_secret ?? ""}
              onChange={(e) => upd({ webhook_secret: e.target.value })}
              disabled={!canWrite}
            />
          </Field>
          <Field label="Deploy Notification Webhooks">
            <textarea
              className="text-input min-h-[110px] resize-y font-mono text-xs"
              value={config.notification_webhooks ?? ""}
              onChange={(e) => upd({ notification_webhooks: e.target.value })}
              placeholder={"https://hooks.slack.com/services/...\nhttps://discord.com/api/webhooks/..."}
              disabled={!canWrite}
            />
          </Field>
          <p className="text-xs text-white/35">
            App-specific webhook secret overrides the global{" "}
            <code className="font-mono text-white/50">
              RELAY_GITHUB_WEBHOOK_SECRET
            </code>
            . Deploy notifications post JSON payloads to each URL listed above.
          </p>
          <button
            type="button"
            onClick={handleSave}
            disabled={busy || !canWrite}
            className="primary-btn"
          >
            {busy ? "Saving..." : "Save GitHub Settings"}
          </button>
        </div>
      </SectionCard>

      {/* Companion Services */}
      <SectionCard
        title="Companion services"
        eyebrow="Managed sidecar containers"
        wide
      >
        <div className="flex gap-2 mb-4 flex-wrap">
          {["postgres", "redis", "worker", "custom"].map((kind) => (
            <button
              key={kind}
              type="button"
              onClick={() => startNewCompanion(kind)}
              disabled={companionBusy}
              className="text-xs border border-white/[0.08] px-3 py-1.5 rounded text-white/60 hover:text-white hover:border-white/20 transition-colors"
            >
              + {prettyCompanionType(kind)}
            </button>
          ))}
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-4">
          <div className="space-y-1">
            {!companions.length ? (
              <div className="text-sm text-white/25 text-center py-6 border border-dashed border-white/[0.08] rounded-lg">
                No companions yet
              </div>
            ) : (
              companions.map((c) => (
                <button
                  key={c.config.name}
                  type="button"
                  onClick={() => hydrateCompanion(c)}
                  className={cn(
                    "w-full text-left px-3 py-2.5 rounded-lg border transition-colors",
                    selectedCompanionName === c.config.name
                      ? "border-relay-accent/40 bg-relay-accent/5"
                      : "border-white/[0.06] hover:border-white/[0.12] hover:bg-white/[0.02]",
                  )}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-white">
                      {c.config.name}
                    </span>
                    <span
                      className={cn(
                        "text-[10px] px-1.5 py-0.5 rounded",
                        c.running
                          ? "text-emerald-400 bg-emerald-500/10"
                          : "text-white/30 bg-white/[0.04]",
                      )}
                    >
                      {c.running ? "running" : "stopped"}
                    </span>
                  </div>
                  <div className="text-xs text-white/30 mt-0.5">
                    {prettyCompanionType(c.config.type ?? "custom")}{" "}
                    {c.config.version ? `v${c.config.version}` : ""}
                  </div>
                </button>
              ))
            )}
          </div>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <Field label="Name">
                <input
                  className="text-input"
                  value={companionDraft.name ?? ""}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      name: e.target.value,
                    })
                  }
                />
              </Field>
              <Field label="Type">
                <select
                  className="text-input"
                  value={companionDraft.type ?? "custom"}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      type: e.target.value,
                    })
                  }
                >
                  {[
                    "postgres",
                    "redis",
                    "worker",
                    "custom",
                    "mysql",
                    "mongo",
                  ].map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="Version">
                <input
                  className="text-input"
                  value={companionDraft.version ?? ""}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      version: e.target.value,
                    })
                  }
                />
              </Field>
              <Field label="Image">
                <input
                  className="text-input"
                  value={companionDraft.image ?? ""}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      image: e.target.value,
                    })
                  }
                />
              </Field>
              <Field label="Command" className="col-span-2">
                <input
                  className="text-input"
                  value={companionDraft.command ?? ""}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      command: e.target.value,
                    })
                  }
                />
              </Field>
              <Field label="Desired State" className="col-span-2">
                <select
                  className="text-input"
                  value={companionDraft.stopped ? "stopped" : "running"}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      stopped: e.target.value === "stopped",
                    })
                  }
                >
                  <option value="running">Running with app</option>
                  <option value="stopped">Keep off</option>
                </select>
              </Field>
              <Field label="Container Port">
                <input
                  type="number"
                  className="text-input"
                  value={companionDraft.port ?? 0}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      port: Number(e.target.value),
                    })
                  }
                />
              </Field>
              <Field label="Host Port">
                <input
                  type="number"
                  className="text-input"
                  value={companionDraft.host_port ?? 0}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      host_port: Number(e.target.value),
                    })
                  }
                />
              </Field>
              <Field label="Environment Variables" className="col-span-2">
                <textarea
                  className="text-input min-h-[80px] resize-y"
                  value={companionEnvText}
                  onChange={(e) => setCompanionEnvText(e.target.value)}
                  placeholder={"KEY=value\nOTHER=value"}
                />
              </Field>
              <Field label="Volumes" className="col-span-2">
                <textarea
                  className="text-input min-h-[60px] resize-y"
                  value={companionVolumesText}
                  onChange={(e) => setCompanionVolumesText(e.target.value)}
                  placeholder={"/host/path:/container/path"}
                />
              </Field>
              <Field label="Health Check Command" className="col-span-2">
                <input
                  className="text-input"
                  value={companionDraft.health?.test ?? ""}
                  onChange={(e) =>
                    setCompanionDraft({
                      ...companionDraft,
                      health: {
                        ...companionDraft.health,
                        test: e.target.value,
                      },
                    })
                  }
                  placeholder="redis-cli ping"
                />
              </Field>
            </div>
            <div className="flex gap-2 flex-wrap">
              <button
                type="button"
                onClick={handleSaveCompanion}
                disabled={companionBusy}
                className="primary-btn"
              >
                {companionBusy ? "Saving..." : "Save Companion"}
              </button>
              {selectedCompanionName && (
                <>
                  <button
                    type="button"
                    onClick={() =>
                      handleRestartCompanion(selectedCompanionName)
                    }
                    disabled={companionBusy}
                    className="ghost-btn"
                  >
                    Restart
                  </button>
                  <button
                    type="button"
                    onClick={() => handleDeleteCompanion(selectedCompanionName)}
                    disabled={companionBusy}
                    className="ghost-btn text-red-400/70 hover:text-red-400"
                  >
                    Delete
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      </SectionCard>

      {/* Secrets */}
      <SectionCard
        title={`Secrets — ${selectedEnv.env}/${selectedEnv.branch}`}
        eyebrow="Environment variables"
      >
        <div className="grid grid-cols-2 gap-3 mb-3">
          <Field label="Key">
            <input
              className="text-input"
              value={draftSecret.key}
              onChange={(e) =>
                setDraftSecret({ ...draftSecret, key: e.target.value })
              }
            />
          </Field>
          <Field label="Value">
            <input
              type="password"
              autoComplete="new-password"
              className="text-input"
              value={draftSecret.value}
              onChange={(e) =>
                setDraftSecret({ ...draftSecret, value: e.target.value })
              }
            />
          </Field>
        </div>
        <button
          type="button"
          onClick={handleAddSecret}
          disabled={!draftSecret.key || !draftSecret.value}
          className="primary-btn mb-4"
        >
          Add Secret
        </button>
        <div className="border-t border-white/[0.06] pt-4 mb-4 space-y-2">
          <div className="text-xs text-white/40">Paste Raw `.env`</div>
          <textarea
            className="text-input min-h-[120px] resize-y"
            value={rawSecretsText}
            onChange={(e) => setRawSecretsText(e.target.value)}
            placeholder={"API_URL=https://api.example.com\nJWT_SECRET=super-secret\nexport FEATURE_FLAG=true"}
          />
          <label className="inline-flex items-center gap-2 text-xs text-white/50">
            <input
              type="checkbox"
              checked={rawSecretsReplace}
              onChange={(e) => setRawSecretsReplace(e.target.checked)}
            />
            Replace existing secrets for this lane first
          </label>
          <div>
            <button
              type="button"
              onClick={handleSaveRawSecrets}
              disabled={!rawSecretsText.trim() || rawSecretsBusy}
              className="primary-btn"
            >
              {rawSecretsBusy ? "Saving..." : "Save Raw Env"}
            </button>
          </div>
        </div>
        <div className="divide-y divide-white/[0.04]">
          {!secrets.length ? (
            <div className="text-sm text-white/25 py-3">
              No secrets configured for this environment.
            </div>
          ) : (
            secrets.map((s) => (
              <div
                key={s.key}
                className="flex items-center justify-between py-2.5"
              >
                <div>
                  <div className="text-sm font-mono text-white">{s.key}</div>
                  <div className="text-xs text-white/30">••••••••</div>
                </div>
                <button
                  type="button"
                  onClick={() => handleDeleteSecret(s.key)}
                  className="text-xs text-white/40 hover:text-red-400 transition-colors px-2 py-1 rounded hover:bg-white/[0.04]"
                >
                  Delete
                </button>
              </div>
            ))
          )}
        </div>
      </SectionCard>

      {/* Danger zone */}
      {project && (
        <SectionCard title="Danger zone" eyebrow="Irreversible actions" danger>
          <p className="text-sm text-white/50 mb-4">
            Permanently delete project{" "}
            <strong className="text-white">{project.name}</strong> including all
            deploys, workspaces, companion data, secrets, and runtime state.
            This cannot be undone.
          </p>
          <div className="flex gap-3 items-center flex-wrap">
            <input
              className="text-input w-64"
              placeholder={`Type "${project.name}" to confirm`}
              value={deleteText}
              onChange={(e) => setDeleteText(e.target.value)}
            />
            <button
              type="button"
              onClick={handleDeleteProject}
              disabled={deleteBusy || deleteText !== project.name}
              className={cn(
                "text-sm px-4 py-2 rounded font-semibold border transition-colors",
                deleteText === project.name
                  ? "bg-red-500 border-red-500 text-white hover:bg-red-600"
                  : "bg-white/[0.04] border-white/[0.08] text-white/30 cursor-not-allowed",
              )}
            >
              {deleteBusy ? "Deleting..." : "Delete project"}
            </button>
          </div>
          {deleteError && (
            <div className="text-red-400 text-sm mt-2">{deleteError}</div>
          )}
        </SectionCard>
      )}
    </div>
  );
}

function SectionCard({
  title,
  eyebrow,
  children,
  wide,
  danger,
}: {
  title: string;
  eyebrow?: string;
  children: React.ReactNode;
  wide?: boolean;
  danger?: boolean;
}) {
  return (
    <div
      className={cn(
        "bg-white/[0.02] border rounded-xl p-5 space-y-4",
        danger ? "border-red-500/20" : "border-white/[0.06]",
      )}
    >
      <div>
        {eyebrow && <div className="eyebrow mb-0.5">{eyebrow}</div>}
        <h2 className="text-base font-semibold text-white">{title}</h2>
      </div>
      {children}
    </div>
  );
}

function Field({
  label,
  children,
  className,
}: {
  label: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <label className={cn("block", className)}>
      <div className="text-xs text-white/40 mb-1.5">{label}</div>
      {children}
    </label>
  );
}

function normalizeHostEntries(values: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of values) {
    const host = raw.trim().toLowerCase();
    if (!host || seen.has(host)) continue;
    seen.add(host);
    out.push(host);
  }
  return out;
}

function HostChipsInput({
  value,
  onChange,
  disabled,
  placeholder,
}: {
  value: string[];
  onChange: (value: string[]) => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  const [draft, setDraft] = useState("");

  function commit(nextRaw: string) {
    const additions = normalizeHostEntries(
      nextRaw
        .split(/[\s,;\n\r\t]+/)
        .map((item) => item.trim())
        .filter(Boolean),
    );
    if (!additions.length) {
      setDraft("");
      return;
    }
    onChange(normalizeHostEntries([...value, ...additions]));
    setDraft("");
  }

  function remove(host: string) {
    onChange(value.filter((item) => item !== host));
  }

  return (
    <div className="rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5">
      <div className="flex flex-wrap gap-2">
        {value.map((host, index) => (
          <span
            key={host}
            className="inline-flex items-center gap-2 rounded-full border border-white/[0.08] bg-white/[0.06] px-3 py-1 text-xs text-white"
          >
            <span className="font-mono">{host}</span>
            {index === 0 && (
              <span className="rounded-full bg-relay-accent/15 px-1.5 py-0.5 text-[10px] uppercase tracking-[0.18em] text-relay-accent">
                Primary
              </span>
            )}
            <button
              type="button"
              onClick={() => remove(host)}
              disabled={disabled}
              className="text-white/45 transition hover:text-white disabled:cursor-not-allowed disabled:text-white/20"
              aria-label={`Remove ${host}`}
            >
              x
            </button>
          </span>
        ))}
        <input
          className="min-w-[220px] flex-1 bg-transparent text-sm text-white outline-none placeholder:text-white/28"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (disabled) return;
            if (e.key === "Enter" || e.key === "," || e.key === "Tab") {
              const trimmed = draft.trim();
              if (trimmed) {
                e.preventDefault();
                commit(draft);
              }
              return;
            }
            if (e.key === "Backspace" && !draft && value.length > 0) {
              e.preventDefault();
              onChange(value.slice(0, -1));
            }
          }}
          onBlur={() => {
            if (!disabled && draft.trim()) commit(draft);
          }}
          onPaste={(e) => {
            if (disabled) return;
            const text = e.clipboardData.getData("text");
            if (!text) return;
            if (/[,\n\r;\t ]/.test(text)) {
              e.preventDefault();
              commit(text);
            }
          }}
          placeholder={placeholder}
          disabled={disabled}
        />
      </div>
      <p className="mt-2 text-[11px] text-white/32">
        Type a hostname and press Enter, Tab, or comma to add it. The first chip is the primary host used for preview links.
      </p>
    </div>
  );
}

function SegmentCard({
  label,
  description,
  children,
}: {
  label: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="bg-white/[0.02] border border-white/[0.06] rounded-lg p-3.5">
      <div className="eyebrow mb-2">{label}</div>
      <div className="seg-control mb-2">{children}</div>
      <p className="text-[11px] text-white/35 leading-relaxed">{description}</p>
    </div>
  );
}

function SegButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn("seg-btn flex-1", active && "seg-btn--active")}
    >
      {children}
    </button>
  );
}

function MiniMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2.5">
      <div className="text-[10px] text-white/35 mb-1">{label}</div>
      <div className="text-sm text-white">{value}</div>
    </div>
  );
}

// Style helpers - add to globals.css or use inline
// .text-input { @apply w-full bg-white/[0.04] border border-white/[0.08] rounded px-3 py-2 text-sm text-white placeholder:text-white/25 focus:outline-none focus:border-white/25; }
// .primary-btn { @apply text-sm bg-white text-black font-semibold px-4 py-2 rounded hover:bg-white/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed; }
// .ghost-btn { @apply text-sm border border-white/[0.10] text-white/60 hover:text-white px-4 py-2 rounded hover:bg-white/[0.06] transition-colors disabled:opacity-40 disabled:cursor-not-allowed; }
