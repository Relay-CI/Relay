"use client";

import { useState, useEffect, useCallback } from "react";
import { cn } from "@/lib/utils";
import {
  buildSettingsConfig,
  envToText,
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
  getSecrets,
  setSecret,
  deleteSecret,
  getCompanions,
  saveCompanion,
  deleteCompanion,
  restartCompanion,
  deleteProject,
  type AppConfig,
  type Secret,
  type Companion,
  type CompanionConfig,
  type Service,
  type EnvInfo,
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
  onUpdated: () => void;
}

const ENGINE_OPTIONS = [
  { value: "docker", title: "Docker", summary: "Resolves a Dockerfile and builds/runs the image locally." },
  { value: "station", title: "Station", summary: "Runs the app natively using Relay Station via a saved workspace snapshot." },
];

const MODE_OPTIONS = [
  { value: "http", title: "HTTP", summary: "Relay proxies internet traffic to the app container port." },
  { value: "static", title: "Static", summary: "Serve a prebuilt static folder with no running container." },
  { value: "off", title: "Off", summary: "Deploy-only mode. No live traffic — use to run CI/CD pipelines without exposing a route." },
];

const POLICY_OPTIONS = [
  { value: "bluegreen", title: "Blue/Green", summary: "Swap traffic instantly between two stable containers." },
  { value: "rolling", title: "Rolling", summary: "Drain the current container gradually as the new one becomes healthy." },
];

export function SettingsPage({ selectedEnv, project, services, onUpdated }: SettingsPageProps) {
  const [config, setConfig] = useState<AppConfig>(buildSettingsConfig() as AppConfig);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "warn"; text: string } | null>(null);

  const [secrets, setSecrets] = useState<Secret[]>([]);
  const [draftSecret, setDraftSecret] = useState({ key: "", value: "" });

  const [companions, setCompanions] = useState<Companion[]>([]);
  const [companionBusy, setCompanionBusy] = useState(false);
  const [selectedCompanionName, setSelectedCompanionName] = useState("");
  const [companionDraft, setCompanionDraft] = useState<CompanionConfig>(() => defaultCompanionDraft("postgres") as CompanionConfig);
  const [companionEnvText, setCompanionEnvText] = useState("");
  const [companionVolumesText, setCompanionVolumesText] = useState("");

  const [deleteText, setDeleteText] = useState("");
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [deleteError, setDeleteError] = useState("");

  const load = useCallback(async () => {
    if (!selectedEnv) return;
    try {
      const [cfg, secs, comps] = await Promise.all([
        getAppConfig(selectedEnv),
        getSecrets(selectedEnv),
        getCompanions(selectedEnv),
      ]);
      const normalized = buildSettingsConfig(cfg) as AppConfig;
      setConfig(normalized);
      setSecrets(secs ?? []);
      setCompanions(comps ?? []);
    } catch (err) {
      setNotice({ tone: "warn", text: `Failed to load settings: ${err instanceof Error ? err.message : "unknown error"}` });
    }
  }, [selectedEnv?.app, selectedEnv?.env, selectedEnv?.branch]);

  useEffect(() => { load(); }, [load]);

  function upd(patch: Partial<AppConfig>) {
    setConfig((c) => ({ ...c, ...patch }));
  }

  function toApiPayload(cfg: AppConfig) {
    return {
      ...cfg,
      mode: uiModeToApi(cfg.mode),
      traffic_mode: uiTrafficModeToApi(cfg.traffic_mode),
      host_port: Number(cfg.host_port) || 0,
      service_port: Number(cfg.service_port) || 0,
    };
  }

  async function handleSave() {
    if (!selectedEnv) return;
    setBusy(true);
    try {
      const payload = toApiPayload(config);
      await saveAppConfig(selectedEnv, payload);
      setNotice({ tone: "warn", text: "Saved. The next deploy picks up this config automatically. Use Restart to apply it to the running container now." });
      await onUpdated();
    } catch (err) {
      setNotice({ tone: "warn", text: `Save failed: ${err instanceof Error ? err.message : "unknown error"}` });
    } finally { setBusy(false); }
  }

  async function handleSaveRestart() {
    if (!selectedEnv) return;
    setBusy(true);
    try {
      const payload = toApiPayload(config);
      await saveAppConfig(selectedEnv, payload);
      await restartApp(selectedEnv);
      setNotice({ tone: "ok", text: "Saved and restart sent. Relay is applying the updated route and traffic policy now." });
      await onUpdated();
    } catch (err) {
      setNotice({ tone: "warn", text: `Apply failed: ${err instanceof Error ? err.message : "unknown error"}` });
    } finally { setBusy(false); }
  }

  async function handleAddSecret() {
    if (!selectedEnv || !draftSecret.key || !draftSecret.value) return;
    await setSecret(selectedEnv, draftSecret.key, draftSecret.value);
    const next = await getSecrets(selectedEnv);
    setSecrets(next ?? []);
    setDraftSecret({ key: "", value: "" });
  }

  async function handleDeleteSecret(key: string) {
    if (!selectedEnv) return;
    await deleteSecret(selectedEnv, key);
    const next = await getSecrets(selectedEnv);
    setSecrets(next ?? []);
  }

  function hydrateCompanion(c: Companion) {
    const draft = { ...defaultCompanionDraft(c.config.type ?? "custom"), ...c.config };
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
    if (!selectedEnv) return;
    setCompanionBusy(true);
    try {
      const payload = {
        ...companionDraft,
        env: textToEnv(companionEnvText),
        volumes: companionVolumesText.split(/\r?\n/).map((l) => l.trim()).filter(Boolean),
      };
      await saveCompanion(selectedEnv, payload);
      const next = await getCompanions(selectedEnv);
      setCompanions(next ?? []);
      await onUpdated();
    } finally { setCompanionBusy(false); }
  }

  async function handleDeleteCompanion(name: string) {
    if (!selectedEnv) return;
    setCompanionBusy(true);
    try {
      await deleteCompanion(selectedEnv, name);
      const next = await getCompanions(selectedEnv);
      setCompanions(next ?? []);
      await onUpdated();
    } finally { setCompanionBusy(false); }
  }

  async function handleRestartCompanion(name: string) {
    if (!selectedEnv) return;
    setCompanionBusy(true);
    try {
      await restartCompanion(selectedEnv, name);
    } finally { setCompanionBusy(false); }
  }

  async function handleDeleteProject() {
    if (!project || deleteText !== project.name) return;
    setDeleteBusy(true); setDeleteError("");
    try {
      await deleteProject(project.name);
      setDeleteText("");
      await onUpdated();
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : "Failed to delete project");
    } finally { setDeleteBusy(false); }
  }

  if (!selectedEnv) {
    return <div className="flex items-center justify-center h-full text-white/30 text-sm">Select an environment to edit settings</div>;
  }

  const draftEngine = normalizeEngineValue(config.engine ?? "docker");

  return (
    <div className="space-y-6">
      <div>
        <div className="eyebrow mb-0.5">Configuration</div>
        <h1 className="text-xl font-semibold text-white">Settings — {selectedEnv.env}/{selectedEnv.branch}</h1>
      </div>

      {notice && (
        <div className={cn("rounded-lg px-4 py-3 text-sm border", notice.tone === "ok" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : "bg-amber-500/10 border-amber-500/30 text-amber-400")}>
          {notice.text}
        </div>
      )}

      {/* Runtime / Routing */}
      <SectionCard title="Runtime / Routing" eyebrow="Server controls">
        <div className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <SegmentCard label="Runtime Engine" description={(ENGINE_OPTIONS.find(o => o.value === draftEngine) ?? ENGINE_OPTIONS[0]).summary}>
              {ENGINE_OPTIONS.map((o) => (
                <SegButton key={o.value} active={draftEngine === o.value} onClick={() => upd({ engine: o.value })}>{o.title}</SegButton>
              ))}
            </SegmentCard>
            <SegmentCard label="Routing Mode" description={(MODE_OPTIONS.find(o => o.value === config.mode) ?? MODE_OPTIONS[0]).summary}>
              {MODE_OPTIONS.map((o) => (
                <SegButton key={o.value} active={config.mode === o.value} onClick={() => upd({ mode: o.value })}>{o.title}</SegButton>
              ))}
            </SegmentCard>
            <SegmentCard label="Traffic Policy" description={(POLICY_OPTIONS.find(o => o.value === config.traffic_mode) ?? POLICY_OPTIONS[0]).summary}>
              {POLICY_OPTIONS.map((o) => (
                <SegButton key={o.value} active={config.traffic_mode === o.value} onClick={() => upd({ traffic_mode: o.value })}>{o.title}</SegButton>
              ))}
            </SegmentCard>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <Field label="Public Host">
              <input className="text-input" value={config.public_host ?? ""} onChange={(e) => upd({ public_host: e.target.value })} placeholder="app.yourdomain.com" />
            </Field>
            <Field label="Host Port">
              <input type="number" className="text-input" value={config.host_port ?? 0} onChange={(e) => upd({ host_port: Number(e.target.value) })} />
            </Field>
            <Field label="Service Port">
              <input type="number" className="text-input" value={config.service_port ?? 0} onChange={(e) => upd({ service_port: Number(e.target.value) })} />
            </Field>
          </div>
          <div className="flex gap-2 flex-wrap">
            <button type="button" onClick={handleSaveRestart} disabled={busy} className="primary-btn">
              {busy ? "Working..." : "Apply & Restart"}
            </button>
            <button type="button" onClick={handleSave} disabled={busy} className="ghost-btn">
              Save for Later
            </button>
            <button type="button" onClick={() => { if (!selectedEnv) return; setBusy(true); stopApp(selectedEnv).finally(() => { setBusy(false); onUpdated(); }); }} disabled={busy} className="ghost-btn">
              Stop App
            </button>
            <button type="button" onClick={() => { if (!selectedEnv) return; setBusy(true); startApp(selectedEnv).finally(() => { setBusy(false); onUpdated(); }); }} disabled={busy} className="ghost-btn">
              Start App
            </button>
          </div>
        </div>
      </SectionCard>

      {/* GitHub / Webhooks */}
      <SectionCard title="GitHub / Webhooks" eyebrow="Per-project integration">
        <div className="space-y-3">
          <Field label="Repository URL">
            <input className="text-input" value={config.repo_url ?? ""} onChange={(e) => upd({ repo_url: e.target.value })} />
          </Field>
          <Field label="Webhook Secret">
            <input type="password" autoComplete="new-password" className="text-input" value={config.webhook_secret ?? ""} onChange={(e) => upd({ webhook_secret: e.target.value })} />
          </Field>
          <p className="text-xs text-white/35">
            App-specific webhook secret overrides the global <code className="font-mono text-white/50">RELAY_GITHUB_WEBHOOK_SECRET</code>.
          </p>
          <button type="button" onClick={handleSave} disabled={busy} className="primary-btn">
            {busy ? "Saving..." : "Save GitHub Settings"}
          </button>
        </div>
      </SectionCard>

      {/* Companion Services */}
      <SectionCard title="Companion services" eyebrow="Managed sidecar containers" wide>
        <div className="flex gap-2 mb-4 flex-wrap">
          {["postgres", "redis", "worker", "custom"].map((kind) => (
            <button key={kind} type="button" onClick={() => startNewCompanion(kind)} disabled={companionBusy} className="text-xs border border-white/[0.08] px-3 py-1.5 rounded text-white/60 hover:text-white hover:border-white/20 transition-colors">
              + {prettyCompanionType(kind)}
            </button>
          ))}
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-4">
          <div className="space-y-1">
            {!companions.length ? (
              <div className="text-sm text-white/25 text-center py-6 border border-dashed border-white/[0.08] rounded-lg">No companions yet</div>
            ) : (
              companions.map((c) => (
                <button key={c.config.name} type="button" onClick={() => hydrateCompanion(c)} className={cn("w-full text-left px-3 py-2.5 rounded-lg border transition-colors", selectedCompanionName === c.config.name ? "border-relay-accent/40 bg-relay-accent/5" : "border-white/[0.06] hover:border-white/[0.12] hover:bg-white/[0.02]")}>
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-white">{c.config.name}</span>
                    <span className={cn("text-[10px] px-1.5 py-0.5 rounded", c.running ? "text-emerald-400 bg-emerald-500/10" : "text-white/30 bg-white/[0.04]")}>{c.running ? "running" : "stopped"}</span>
                  </div>
                  <div className="text-xs text-white/30 mt-0.5">{prettyCompanionType(c.config.type ?? "custom")} {c.config.version ? `v${c.config.version}` : ""}</div>
                </button>
              ))
            )}
          </div>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <Field label="Name"><input className="text-input" value={companionDraft.name ?? ""} onChange={(e) => setCompanionDraft({ ...companionDraft, name: e.target.value })} /></Field>
              <Field label="Type">
                <select className="text-input" value={companionDraft.type ?? "custom"} onChange={(e) => setCompanionDraft({ ...companionDraft, type: e.target.value })}>
                  {["postgres", "redis", "worker", "custom", "mysql", "mongo"].map((t) => <option key={t} value={t}>{t}</option>)}
                </select>
              </Field>
              <Field label="Version"><input className="text-input" value={companionDraft.version ?? ""} onChange={(e) => setCompanionDraft({ ...companionDraft, version: e.target.value })} /></Field>
              <Field label="Image"><input className="text-input" value={companionDraft.image ?? ""} onChange={(e) => setCompanionDraft({ ...companionDraft, image: e.target.value })} /></Field>
              <Field label="Command" className="col-span-2"><input className="text-input" value={companionDraft.command ?? ""} onChange={(e) => setCompanionDraft({ ...companionDraft, command: e.target.value })} /></Field>
              <Field label="Desired State" className="col-span-2">
                <select className="text-input" value={companionDraft.stopped ? "stopped" : "running"} onChange={(e) => setCompanionDraft({ ...companionDraft, stopped: e.target.value === "stopped" })}>
                  <option value="running">Running with app</option>
                  <option value="stopped">Keep off</option>
                </select>
              </Field>
              <Field label="Container Port"><input type="number" className="text-input" value={companionDraft.port ?? 0} onChange={(e) => setCompanionDraft({ ...companionDraft, port: Number(e.target.value) })} /></Field>
              <Field label="Host Port"><input type="number" className="text-input" value={companionDraft.host_port ?? 0} onChange={(e) => setCompanionDraft({ ...companionDraft, host_port: Number(e.target.value) })} /></Field>
              <Field label="Environment Variables" className="col-span-2">
                <textarea className="text-input min-h-[80px] resize-y" value={companionEnvText} onChange={(e) => setCompanionEnvText(e.target.value)} placeholder={"KEY=value\nOTHER=value"} />
              </Field>
              <Field label="Volumes" className="col-span-2">
                <textarea className="text-input min-h-[60px] resize-y" value={companionVolumesText} onChange={(e) => setCompanionVolumesText(e.target.value)} placeholder={"/host/path:/container/path"} />
              </Field>
              <Field label="Health Check Command" className="col-span-2">
                <input className="text-input" value={companionDraft.health?.test ?? ""} onChange={(e) => setCompanionDraft({ ...companionDraft, health: { ...companionDraft.health, test: e.target.value } })} placeholder="redis-cli ping" />
              </Field>
            </div>
            <div className="flex gap-2 flex-wrap">
              <button type="button" onClick={handleSaveCompanion} disabled={companionBusy} className="primary-btn">
                {companionBusy ? "Saving..." : "Save Companion"}
              </button>
              {selectedCompanionName && (
                <>
                  <button type="button" onClick={() => handleRestartCompanion(selectedCompanionName)} disabled={companionBusy} className="ghost-btn">Restart</button>
                  <button type="button" onClick={() => handleDeleteCompanion(selectedCompanionName)} disabled={companionBusy} className="ghost-btn text-red-400/70 hover:text-red-400">Delete</button>
                </>
              )}
            </div>
          </div>
        </div>
      </SectionCard>

      {/* Secrets */}
      <SectionCard title={`Secrets — ${selectedEnv.env}/${selectedEnv.branch}`} eyebrow="Environment variables">
        <div className="grid grid-cols-2 gap-3 mb-3">
          <Field label="Key"><input className="text-input" value={draftSecret.key} onChange={(e) => setDraftSecret({ ...draftSecret, key: e.target.value })} /></Field>
          <Field label="Value"><input type="password" autoComplete="new-password" className="text-input" value={draftSecret.value} onChange={(e) => setDraftSecret({ ...draftSecret, value: e.target.value })} /></Field>
        </div>
        <button type="button" onClick={handleAddSecret} disabled={!draftSecret.key || !draftSecret.value} className="primary-btn mb-4">Add Secret</button>
        <div className="divide-y divide-white/[0.04]">
          {!secrets.length ? (
            <div className="text-sm text-white/25 py-3">No secrets configured for this environment.</div>
          ) : secrets.map((s) => (
            <div key={s.key} className="flex items-center justify-between py-2.5">
              <div>
                <div className="text-sm font-mono text-white">{s.key}</div>
                <div className="text-xs text-white/30">••••••••</div>
              </div>
              <button type="button" onClick={() => handleDeleteSecret(s.key)} className="text-xs text-white/40 hover:text-red-400 transition-colors px-2 py-1 rounded hover:bg-white/[0.04]">Delete</button>
            </div>
          ))}
        </div>
      </SectionCard>

      {/* Danger zone */}
      {project && (
        <SectionCard title="Danger zone" eyebrow="Irreversible actions" danger>
          <p className="text-sm text-white/50 mb-4">
            Permanently delete project <strong className="text-white">{project.name}</strong> including all deploys, workspaces, companion data, secrets, and runtime state. This cannot be undone.
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
              className={cn("text-sm px-4 py-2 rounded font-semibold border transition-colors", deleteText === project.name ? "bg-red-500 border-red-500 text-white hover:bg-red-600" : "bg-white/[0.04] border-white/[0.08] text-white/30 cursor-not-allowed")}
            >
              {deleteBusy ? "Deleting..." : "Delete project"}
            </button>
          </div>
          {deleteError && <div className="text-red-400 text-sm mt-2">{deleteError}</div>}
        </SectionCard>
      )}
    </div>
  );
}

function SectionCard({ title, eyebrow, children, wide, danger }: { title: string; eyebrow?: string; children: React.ReactNode; wide?: boolean; danger?: boolean }) {
  return (
    <div className={cn("bg-white/[0.02] border rounded-xl p-5 space-y-4", danger ? "border-red-500/20" : "border-white/[0.06]")}>
      <div>
        {eyebrow && <div className="eyebrow mb-0.5">{eyebrow}</div>}
        <h2 className="text-base font-semibold text-white">{title}</h2>
      </div>
      {children}
    </div>
  );
}

function Field({ label, children, className }: { label: string; children: React.ReactNode; className?: string }) {
  return (
    <label className={cn("block", className)}>
      <div className="text-xs text-white/40 mb-1.5">{label}</div>
      {children}
    </label>
  );
}

function SegmentCard({ label, description, children }: { label: string; description: string; children: React.ReactNode }) {
  return (
    <div className="bg-white/[0.02] border border-white/[0.06] rounded-lg p-3.5">
      <div className="eyebrow mb-2">{label}</div>
      <div className="seg-control mb-2">{children}</div>
      <p className="text-[11px] text-white/35 leading-relaxed">{description}</p>
    </div>
  );
}

function SegButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
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

// Style helpers - add to globals.css or use inline
// .text-input { @apply w-full bg-white/[0.04] border border-white/[0.08] rounded px-3 py-2 text-sm text-white placeholder:text-white/25 focus:outline-none focus:border-white/25; }
// .primary-btn { @apply text-sm bg-white text-black font-semibold px-4 py-2 rounded hover:bg-white/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed; }
// .ghost-btn { @apply text-sm border border-white/[0.10] text-white/60 hover:text-white px-4 py-2 rounded hover:bg-white/[0.06] transition-colors disabled:opacity-40 disabled:cursor-not-allowed; }
