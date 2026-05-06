"use client";

import { useEffect, useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import {
  getServerConfig,
  saveServerConfig,
  getVersion,
  type CustomHostRule,
  type DoctorReport,
  type ServerConfig,
  type VersionInfo,
} from "@/lib/api";

type CurrentUser = { username: string; role: string } | null;

interface ServerSettingsPageProps {
  currentUser: CurrentUser;
}

type ServerTab = "routing" | "rules" | "checks" | "overview";

type RuleDraft = CustomHostRule & { id: string };

type DraftState = {
  baseDomain: string;
  dashboardHost: string;
  acmeDisabled: boolean;
  customHostRules: RuleDraft[];
};

const SERVER_TABS: Array<{ id: ServerTab; label: string }> = [
  { id: "routing", label: "Routing" },
  { id: "rules", label: "Custom Rules" },
  { id: "checks", label: "Checks" },
  { id: "overview", label: "How It Works" },
];

const RULE_ACTIONS: Array<{
  value: RuleDraft["action"];
  label: string;
  detail: string;
}> = [
  {
    value: "redirect",
    label: "Redirect",
    detail: "Send requests to another URL with a redirect code.",
  },
  {
    value: "reverse_proxy",
    label: "Reverse Proxy",
    detail: "Proxy traffic to another HTTP service or upstream.",
  },
  {
    value: "static_response",
    label: "Static Response",
    detail: "Return a fixed status/body directly from the edge.",
  },
  {
    value: "relay_dashboard",
    label: "Relay Dashboard",
    detail: "Serve the Relay admin UI from an extra hostname.",
  },
];

const REDIRECT_CODES = [301, 302, 307, 308];

function createRuleDraft(partial: Partial<RuleDraft> = {}): RuleDraft {
  return {
    id:
      partial.id ??
      `rule-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    host: partial.host ?? "",
    action: partial.action ?? "redirect",
    redirect_url: partial.redirect_url ?? "https://",
    redirect_code: partial.redirect_code ?? 302,
    preserve_path: partial.preserve_path ?? true,
    upstream_url: partial.upstream_url ?? "http://127.0.0.1:9000",
    response_status: partial.response_status ?? 200,
    response_body: partial.response_body ?? "",
    response_content_type:
      partial.response_content_type ?? "text/plain; charset=utf-8",
  };
}

function normalizeRuleDraft(rule: RuleDraft): RuleDraft {
  return createRuleDraft({
    ...rule,
    host: rule.host.trim(),
    redirect_url: (rule.redirect_url ?? "").trim(),
    upstream_url: (rule.upstream_url ?? "").trim(),
    response_body: rule.response_body ?? "",
    response_content_type: (rule.response_content_type ?? "").trim(),
  });
}

function toDraftState(data?: ServerConfig | null): DraftState {
  const rules = Array.isArray(data?.custom_host_rules)
    ? data?.custom_host_rules.map((rule) => createRuleDraft(rule))
    : [];
  return {
    baseDomain: data?.base_domain ?? "",
    dashboardHost: data?.dashboard_host ?? "",
    acmeDisabled: data?.acme_disabled === "true",
    customHostRules: rules,
  };
}

function serializeDraft(draft: DraftState): string {
  return JSON.stringify({
    baseDomain: draft.baseDomain.trim(),
    dashboardHost: draft.dashboardHost.trim(),
    acmeDisabled: draft.acmeDisabled,
    customHostRules: draft.customHostRules.map((rule) => {
      const normalized = normalizeRuleDraft(rule);
      const { id, ...rest } = normalized;
      return rest;
    }),
  });
}

export function ServerSettingsPage({ currentUser }: ServerSettingsPageProps) {
  const isOwner = currentUser?.role === "owner";

  const [activeTab, setActiveTab] = useState<ServerTab>("routing");
  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null);
  const [doctor, setDoctor] = useState<DoctorReport | null>(null);
  const [savedDraft, setSavedDraft] = useState<DraftState>(() => toDraftState());
  const [draft, setDraft] = useState<DraftState>(() => toDraftState());
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{
    tone: "ok" | "danger";
    text: string;
  } | null>(null);

  useEffect(() => {
    getVersion().then((v) => setVersionInfo(v)).catch(() => {});
    if (!isOwner) return;
    getServerConfig()
      .then((data) => {
        const nextDraft = toDraftState(data);
        setDoctor((data?.doctor as DoctorReport | undefined) ?? null);
        setSavedDraft(nextDraft);
        setDraft(nextDraft);
      })
      .catch(() => {});
  }, [isOwner]);

  const dirty = useMemo(
    () => serializeDraft(draft) !== serializeDraft(savedDraft),
    [draft, savedDraft],
  );

  const checks = doctor?.checks ?? {};
  const exampleHost = draft.baseDomain
    ? `myapp-main.${draft.baseDomain}`
    : "myapp-main.example.com";

  if (!isOwner) {
    return (
      <div className="flex items-center justify-center h-full text-white/30 text-sm">
        Owner access required
      </div>
    );
  }

  async function save() {
    setBusy(true);
    setNotice(null);
    try {
      const payload: ServerConfig = {
        base_domain: draft.baseDomain.trim(),
        dashboard_host: draft.dashboardHost.trim(),
        acme_disabled: draft.acmeDisabled ? "true" : "",
        custom_host_rules: draft.customHostRules.map((rule) => {
          const normalized = normalizeRuleDraft(rule);
          const { id, ...rest } = normalized;
          return rest;
        }),
      };
      const saved = await saveServerConfig(payload);
      const nextDraft = toDraftState(saved);
      setDoctor((saved?.doctor as DoctorReport | undefined) ?? null);
      setSavedDraft(nextDraft);
      setDraft(nextDraft);
      setNotice({
        tone: "ok",
        text:
          activeTab === "rules"
            ? "Saved. Custom host rules were written to the global Caddy proxy."
            : "Saved. Global routing settings and managed domains were refreshed.",
      });
    } catch (err) {
      setNotice({
        tone: "danger",
        text: err instanceof Error ? err.message : "Save failed",
      });
    } finally {
      setBusy(false);
    }
  }

  function updateRule(id: string, patch: Partial<RuleDraft>) {
    setDraft((current) => ({
      ...current,
      customHostRules: current.customHostRules.map((rule) =>
        rule.id === id ? normalizeRuleDraft({ ...rule, ...patch }) : rule,
      ),
    }));
  }

  function addRule() {
    setDraft((current) => ({
      ...current,
      customHostRules: [...current.customHostRules, createRuleDraft()],
    }));
  }

  function removeRule(id: string) {
    setDraft((current) => ({
      ...current,
      customHostRules: current.customHostRules.filter((rule) => rule.id !== id),
    }));
  }

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Administration</div>
        <h1 className="text-xl font-semibold text-white">Server settings</h1>
      </div>

      {versionInfo && (
        <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
          <div className="eyebrow mb-1">Server info</div>
          <div className="text-base font-semibold text-white mb-4">
            relayd {versionInfo.version}
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <KVCard label="Version" value={versionInfo.version} mono />
            <KVCard
              label="Commit"
              value={versionInfo.commit?.slice(0, 12) ?? "—"}
              mono
            />
            <KVCard label="Build date" value={versionInfo.build_date ?? "—"} />
            <KVCard
              label="OS / Arch"
              value={`${versionInfo.os}/${versionInfo.arch}`}
              mono
            />
          </div>
        </div>
      )}

      <div className="flex gap-1 border-b border-white/[0.06] pb-0 -mb-0">
        {SERVER_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            className={cn(
              "px-3 py-2 text-sm font-medium border-b-2 transition-colors -mb-px",
              activeTab === tab.id
                ? "border-relay-accent text-white"
                : "border-transparent text-white/40 hover:text-white/70",
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {notice && (
        <div
          className={cn(
            "rounded-lg px-4 py-3 text-sm border",
            notice.tone === "ok"
              ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
              : "bg-red-500/10 border-red-500/30 text-red-400",
          )}
        >
          {notice.text}
        </div>
      )}

      {activeTab === "routing" && (
        <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
          <div>
            <div className="eyebrow mb-0.5">Global proxy / domain routing</div>
            <h2 className="text-base font-semibold text-white">
              Server-level routing
            </h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field label="Base Domain">
              <input
                className="text-input"
                value={draft.baseDomain}
                onChange={(e) =>
                  setDraft((current) => ({
                    ...current,
                    baseDomain: e.target.value,
                  }))
                }
                placeholder="yourdomain.com"
              />
            </Field>
            <Field label="Dashboard Host">
              <input
                className="text-input"
                value={draft.dashboardHost}
                onChange={(e) =>
                  setDraft((current) => ({
                    ...current,
                    dashboardHost: e.target.value,
                  }))
                }
                placeholder="admin.yourdomain.com"
              />
            </Field>
          </div>

          <label className="flex items-center gap-3 cursor-pointer select-none">
            <div className="relative">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={draft.acmeDisabled}
                onChange={(e) =>
                  setDraft((current) => ({
                    ...current,
                    acmeDisabled: e.target.checked,
                  }))
                }
              />
              <div className="w-9 h-5 rounded-full border border-white/20 bg-white/[0.06] peer-checked:bg-red-500/80 peer-checked:border-red-500/60 transition-colors" />
              <div className="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white/50 peer-checked:translate-x-4 transition-transform" />
            </div>
            <div>
              <div className="text-sm text-white font-medium">
                Disable ACME listener
              </div>
              <div className="text-xs text-white/40">
                Don&apos;t bind Relay&apos;s local ACME HTTP-01 listener on port
                80. Use this when another ACME process already owns port 80.
              </div>
            </div>
          </label>

          <p className="text-xs text-white/40 leading-relaxed">
            Apps deployed without an explicit{" "}
            <code className="font-mono text-white/50">public_host</code> get an
            auto-generated subdomain:{" "}
            <code className="font-mono text-white/50">{exampleHost}</code>.
            Relay starts a Caddy reverse proxy that handles TLS automatically.
            Set{" "}
            <code className="font-mono text-white/50">Dashboard Host</code> to
            route the Relay admin through Caddy. These values can also be set via{" "}
            <code className="font-mono text-white/50">
              RELAY_BASE_DOMAIN
            </code>{" "}
            /{" "}
            <code className="font-mono text-white/50">
              RELAY_DASHBOARD_HOST
            </code>
            .
          </p>

          <SaveBar
            busy={busy}
            dirty={dirty}
            onSave={save}
            label="Save global settings"
          />
        </div>
      )}

      {activeTab === "rules" && (
        <div className="space-y-4">
          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="eyebrow mb-0.5">Global proxy behavior</div>
                <h2 className="text-base font-semibold text-white">
                  Custom host rules
                </h2>
                <p className="text-xs text-white/40 mt-1 max-w-2xl">
                  Add host-level behaviors before app routing. Example: when{" "}
                  <code className="font-mono text-white/50">
                    relay.f4ust.com
                  </code>{" "}
                  is hit, redirect it somewhere else, proxy it to another
                  service, return a fixed response, or alias the Relay
                  dashboard.
                </p>
              </div>
              <button
                type="button"
                onClick={addRule}
                className="px-3 py-2 rounded-lg bg-white text-black text-sm font-semibold hover:bg-white/90 transition-colors"
              >
                Add rule
              </button>
            </div>

            {!draft.customHostRules.length && (
              <div className="rounded-lg border border-dashed border-white/[0.12] bg-white/[0.02] px-4 py-5 text-sm text-white/40">
                No custom host rules yet.
              </div>
            )}

            <div className="space-y-3">
              {draft.customHostRules.map((rule, index) => {
                const actionMeta =
                  RULE_ACTIONS.find((item) => item.value === rule.action) ??
                  RULE_ACTIONS[0];
                return (
                  <div
                    key={rule.id}
                    className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-4 space-y-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="eyebrow mb-0.5">
                          Rule {index + 1}
                        </div>
                        <div className="text-sm font-medium text-white">
                          {actionMeta.label}
                        </div>
                        <div className="text-xs text-white/35 mt-0.5">
                          {actionMeta.detail}
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => removeRule(rule.id)}
                        className="text-xs text-white/45 hover:text-red-300 transition-colors"
                      >
                        Remove
                      </button>
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                      <Field label="Host">
                        <input
                          className="text-input"
                          value={rule.host}
                          onChange={(e) =>
                            updateRule(rule.id, { host: e.target.value })
                          }
                          placeholder="relay.f4ust.com"
                        />
                      </Field>
                      <Field label="Action">
                        <select
                          className="text-input"
                          value={rule.action}
                          onChange={(e) =>
                            updateRule(rule.id, {
                              action: e.target.value as RuleDraft["action"],
                            })
                          }
                        >
                          {RULE_ACTIONS.map((item) => (
                            <option key={item.value} value={item.value}>
                              {item.label}
                            </option>
                          ))}
                        </select>
                      </Field>
                    </div>

                    {rule.action === "redirect" && (
                      <div className="grid grid-cols-1 lg:grid-cols-[1.4fr_0.7fr] gap-3">
                        <Field label="Redirect To">
                          <input
                            className="text-input"
                            value={rule.redirect_url ?? ""}
                            onChange={(e) =>
                              updateRule(rule.id, {
                                redirect_url: e.target.value,
                              })
                            }
                            placeholder="https://example.com"
                          />
                        </Field>
                        <Field label="Redirect Code">
                          <select
                            className="text-input"
                            value={String(rule.redirect_code ?? 302)}
                            onChange={(e) =>
                              updateRule(rule.id, {
                                redirect_code: Number(e.target.value),
                              })
                            }
                          >
                            {REDIRECT_CODES.map((code) => (
                              <option key={code} value={code}>
                                {code}
                              </option>
                            ))}
                          </select>
                        </Field>
                        <label className="flex items-center gap-2 text-sm text-white/70">
                          <input
                            type="checkbox"
                            checked={rule.preserve_path ?? true}
                            onChange={(e) =>
                              updateRule(rule.id, {
                                preserve_path: e.target.checked,
                              })
                            }
                          />
                          Preserve the incoming path and query string
                        </label>
                      </div>
                    )}

                    {rule.action === "reverse_proxy" && (
                      <Field label="Upstream URL">
                        <input
                          className="text-input"
                          value={rule.upstream_url ?? ""}
                          onChange={(e) =>
                            updateRule(rule.id, {
                              upstream_url: e.target.value,
                            })
                          }
                          placeholder="http://127.0.0.1:9000"
                        />
                      </Field>
                    )}

                    {rule.action === "static_response" && (
                      <div className="space-y-3">
                        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                          <Field label="Status Code">
                            <input
                              type="number"
                              min={100}
                              max={599}
                              className="text-input"
                              value={rule.response_status ?? 200}
                              onChange={(e) =>
                                updateRule(rule.id, {
                                  response_status: Number(e.target.value) || 200,
                                })
                              }
                            />
                          </Field>
                          <Field label="Content Type">
                            <input
                              className="text-input"
                              value={rule.response_content_type ?? ""}
                              onChange={(e) =>
                                updateRule(rule.id, {
                                  response_content_type: e.target.value,
                                })
                              }
                              placeholder="text/plain; charset=utf-8"
                            />
                          </Field>
                        </div>
                        <Field label="Response Body">
                          <textarea
                            className="text-input min-h-[120px] resize-y"
                            value={rule.response_body ?? ""}
                            onChange={(e) =>
                              updateRule(rule.id, {
                                response_body: e.target.value,
                              })
                            }
                            placeholder="Coming soon"
                          />
                        </Field>
                      </div>
                    )}

                    {rule.action === "relay_dashboard" && (
                      <div className="rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-3 text-xs text-white/40 leading-relaxed">
                        This host will serve the Relay admin UI through the
                        global proxy. Use it when you want extra aliases like{" "}
                        <code className="font-mono text-white/50">
                          relay.f4ust.com
                        </code>{" "}
                        or{" "}
                        <code className="font-mono text-white/50">
                          admin2.yourdomain.com
                        </code>
                        .
                      </div>
                    )}
                  </div>
                );
              })}
            </div>

            <SaveBar
              busy={busy}
              dirty={dirty}
              onSave={save}
              label="Save custom rules"
            />
          </div>

          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
            <div className="eyebrow mb-1">Examples</div>
            <div className="space-y-2.5">
              {[
                "relay.f4ust.com -> redirect -> https://f4ust.com",
                "status.f4ust.com -> static response -> 200 + plain text body",
                "grafana.f4ust.com -> reverse proxy -> http://127.0.0.1:3001",
                "admin-alt.f4ust.com -> Relay dashboard alias",
              ].map((item) => (
                <div
                  key={item}
                  className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-4 py-3 text-sm text-white/60"
                >
                  {item}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {activeTab === "checks" && (
        <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
          <div>
            <div className="eyebrow mb-0.5">Guided setup</div>
            <h2 className="text-base font-semibold text-white">
              DNS / TLS rollout
            </h2>
            <p className="text-xs text-white/40 mt-1">
              Save the host values above, then use these checks to verify DNS,
              HTTPS, Docker, and the helper listeners.
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-[1.15fr_1fr] gap-4">
            <div className="space-y-3">
              <GuideStep
                index="1"
                title="Choose Relay hostnames"
                detail={`Use Base Domain for managed app hosts like ${exampleHost}. Use Dashboard Host for Relay itself, for example admin.${draft.baseDomain || "example.com"}.`}
              />
              <GuideStep
                index="2"
                title="Create DNS records"
                detail="Point the dashboard host at this server. If you want managed app URLs, point a wildcard record (*.yourdomain.com) at the same machine too."
              />
              <GuideStep
                index="3"
                title="Allow ACME + TLS traffic"
                detail="Ports 80 and 443 must reach this machine. Keep the ACME listener enabled unless another proxy or certificate manager already owns port 80."
              />
              <GuideStep
                index="4"
                title="Wait for the green checks"
                detail="After saving, Relay refreshes the Caddy proxy and probes the setup again. DNS may lag for a few minutes before HTTPS starts passing."
              />
            </div>

            <div className="space-y-2.5">
              <CheckRow label="Data dir" check={checks.data_dir} />
              <CheckRow label="Secrets key" check={checks.secret_key} />
              <CheckRow label="Docker" check={checks.docker} />
              <CheckRow label="Socket" check={checks.socket} />
              <CheckRow label="ACME" check={checks.acme} />
              <CheckRow label="Dashboard host" check={checks.dashboard_host} />
              <CheckRow label="Managed subdomains" check={checks.base_domain} />
              <CheckRow label="Caddy proxy" check={checks.caddy_proxy} />
              <CheckRow label="Webhook URL" check={checks.webhook} />
            </div>
          </div>

          {doctor?.managed_example_url && (
            <div className="bg-white/[0.03] border border-white/[0.06] rounded-lg px-4 py-3">
              <div className="text-xs text-white/35 mb-1">
                Managed example URL
              </div>
              <div className="text-sm text-white font-mono break-all">
                {doctor.managed_example_url}
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === "overview" && (
        <div className="space-y-4">
          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
            <div className="eyebrow mb-1">How it works</div>
            <h2 className="text-base font-semibold text-white mb-4">
              Domain routing overview
            </h2>
            <div className="space-y-2.5">
              {[
                {
                  title: "Auto subdomains",
                  detail: `Set Base Domain here. New deploys auto-get {app}-{branch}.{domain}.`,
                },
                {
                  title: "Dashboard host",
                  detail:
                    "Set Dashboard Host to route the Relay admin through Caddy, e.g. admin.yourdomain.com.",
                },
                {
                  title: "Custom domain per app",
                  detail:
                    "Set Public Host in the app's Settings tab to override the auto-assigned subdomain.",
                },
                {
                  title: "Custom host rules",
                  detail:
                    "Use the Custom Rules tab for redirects, static edge responses, upstream proxies, and extra dashboard aliases.",
                },
                {
                  title: "ACME listener",
                  detail:
                    "Relay can run a lightweight HTTP listener on :80 for ACME HTTP-01 challenge files and optional HTTP->HTTPS redirects.",
                },
                {
                  title: "Caddy TLS",
                  detail:
                    "Relay runs a caddy:alpine container (relay-global-proxy) that terminates TLS and proxies to each app.",
                },
                {
                  title: "DNS requirement",
                  detail:
                    "Point your domain or wildcard (*.yourdomain.com) A record at this server's public IP.",
                },
              ].map((row) => (
                <div
                  key={row.title}
                  className="border border-white/[0.06] rounded-lg px-4 py-3"
                >
                  <div className="text-sm font-medium text-white">
                    {row.title}
                  </div>
                  <div className="text-xs text-white/40 mt-0.5">
                    {row.detail}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl px-5 py-4 flex items-center gap-3">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className="text-white/30 shrink-0"
            >
              <circle cx="12" cy="12" r="10" />
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
              <line x1="2" y1="12" x2="22" y2="12" />
            </svg>
            <div>
              <div className="text-sm font-medium text-white/70">
                Appearance settings have moved
              </div>
              <div className="text-xs text-white/35 mt-0.5">
                Use the globe icon in the topbar to access themes, fonts, and
                layout options.
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function SaveBar({
  busy,
  dirty,
  onSave,
  label,
}: {
  busy: boolean;
  dirty: boolean;
  onSave: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onSave}
      disabled={busy || !dirty}
      className={cn(
        "text-sm px-4 py-2 rounded font-semibold transition-colors",
        dirty
          ? "bg-white text-black hover:bg-white/90"
          : "bg-white/[0.06] text-white/30 cursor-not-allowed",
      )}
    >
      {busy ? "Saving..." : label}
    </button>
  );
}

function KVCard({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="bg-white/[0.03] rounded-lg px-3 py-2.5">
      <div className="text-[10px] text-white/35 mb-1">{label}</div>
      <div className={cn("text-sm text-white", mono && "font-mono")}>
        {value}
      </div>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <div className="text-xs text-white/40 mb-1.5">{label}</div>
      {children}
    </label>
  );
}

function GuideStep({
  index,
  title,
  detail,
}: {
  index: string;
  title: string;
  detail: string;
}) {
  return (
    <div className="flex gap-3">
      <div className="w-6 h-6 shrink-0 rounded-full bg-white/[0.06] border border-white/[0.1] text-xs text-white/60 flex items-center justify-center">
        {index}
      </div>
      <div>
        <div className="text-sm font-medium text-white">{title}</div>
        <div className="text-xs text-white/40 mt-0.5 leading-relaxed">
          {detail}
        </div>
      </div>
    </div>
  );
}

function CheckRow({
  label,
  check,
}: {
  label: string;
  check?: DoctorReport["checks"][string];
}) {
  const tone = check?.status ?? "info";
  const toneClass =
    tone === "ok"
      ? "bg-emerald-500/10 border-emerald-500/25 text-emerald-400"
      : tone === "warn"
        ? "bg-amber-500/10 border-amber-500/25 text-amber-400"
        : tone === "error"
          ? "bg-red-500/10 border-red-500/25 text-red-400"
          : "bg-white/[0.04] border-white/[0.08] text-white/50";

  return (
    <div className="border border-white/[0.06] rounded-lg px-3 py-2.5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-medium text-white">{label}</div>
          <div className="text-xs text-white/40 mt-0.5">
            {check?.summary ?? "Waiting for diagnostics"}
          </div>
        </div>
        <span
          className={cn(
            "text-[10px] uppercase tracking-wider font-semibold px-2 py-0.5 rounded border",
            toneClass,
          )}
        >
          {tone}
        </span>
      </div>
      {check?.detail && (
        <div className="text-[11px] text-white/35 mt-2 break-all">
          {check.detail}
        </div>
      )}
      {check?.hint && (
        <div className="text-[11px] text-white/28 mt-1">{check.hint}</div>
      )}
    </div>
  );
}
