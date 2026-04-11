"use client";

import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import { getServerConfig, saveServerConfig, getVersion, type VersionInfo } from "@/lib/api";
import { useTheme, BUILT_IN_THEMES } from "@/components/theme-provider";

type CurrentUser = { username: string; role: string } | null;

interface ServerSettingsPageProps {
  currentUser: CurrentUser;
}


export function ServerSettingsPage({ currentUser }: ServerSettingsPageProps) {
  const isOwner = currentUser?.role === "owner";
  const theme = useTheme();

  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null);
  const [baseDomain, setBaseDomain] = useState("");
  const [dashboardHost, setDashboardHost] = useState("");
  const [acmeDisabled, setAcmeDisabled] = useState(false);
  const [draft, setDraft] = useState({ baseDomain: "", dashboardHost: "", acmeDisabled: false });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "danger"; text: string } | null>(null);

  // Appearance state
  const [savedThemeName, setSavedThemeName] = useState("default");
  const [savedCustomCSS, setSavedCustomCSS] = useState("");
  const [themeDraft, setThemeDraft] = useState({ name: "default", css: "" });
  const [appearanceBusy, setAppearanceBusy] = useState(false);
  const [appearanceNotice, setAppearanceNotice] = useState<{ tone: "ok" | "danger"; text: string } | null>(null);

  useEffect(() => {
    getVersion().then((v) => setVersionInfo(v)).catch(() => {});
    if (isOwner) {
      getServerConfig().then((data) => {
        const bd = data?.base_domain ?? "";
        const dh = data?.dashboard_host ?? "";
        const ad = data?.acme_disabled === "true";
        setBaseDomain(bd); setDashboardHost(dh); setAcmeDisabled(ad);
        setDraft({ baseDomain: bd, dashboardHost: dh, acmeDisabled: ad });
        // Load saved appearance
        const tn = data?.theme_name ?? "default";
        const tc = data?.theme_css ?? "";
        setSavedThemeName(tn);
        setSavedCustomCSS(tc);
        setThemeDraft({ name: tn, css: tc });
      }).catch(() => {});
    }
  }, [isOwner]);

  if (!isOwner) {
    return <div className="flex items-center justify-center h-full text-white/30 text-sm">Owner access required</div>;
  }

  const dirty = draft.baseDomain !== baseDomain || draft.dashboardHost !== dashboardHost || draft.acmeDisabled !== acmeDisabled;
  const exampleHost = draft.baseDomain ? `myapp-main.${draft.baseDomain}` : "myapp-main.example.com";

  async function save() {
    setBusy(true); setNotice(null);
    try {
      const saved = await saveServerConfig({
        base_domain: draft.baseDomain,
        dashboard_host: draft.dashboardHost,
        acme_disabled: draft.acmeDisabled ? "true" : "",
      });
      const bd = saved?.base_domain ?? "";
      const dh = saved?.dashboard_host ?? "";
      const ad = saved?.acme_disabled === "true";
      setBaseDomain(bd); setDashboardHost(dh); setAcmeDisabled(ad);
      setDraft({ baseDomain: bd, dashboardHost: dh, acmeDisabled: ad });
      setNotice({ tone: "ok", text: "Saved. Caddy will route the dashboard host back to Relay, and new deploys without an explicit public host will auto-assign a subdomain." });
    } catch (err) {
      setNotice({ tone: "danger", text: err instanceof Error ? err.message : "Save failed" });
    } finally { setBusy(false); }
  }

  async function saveAppearance() {
    setAppearanceBusy(true); setAppearanceNotice(null);
    try {
      const saved = await saveServerConfig({
        theme_name: themeDraft.name,
        theme_css: themeDraft.css,
      });
      const tn = saved?.theme_name ?? "default";
      const tc = saved?.theme_css ?? "";
      setSavedThemeName(tn);
      setSavedCustomCSS(tc);
      setThemeDraft({ name: tn, css: tc });
      // Apply the saved theme immediately via the provider
      theme.previewTheme(tn, tc);
      setAppearanceNotice({ tone: "ok", text: "Theme saved and applied." });
    } catch (err) {
      setAppearanceNotice({ tone: "danger", text: err instanceof Error ? err.message : "Save failed" });
    } finally { setAppearanceBusy(false); }
  }

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Administration</div>
        <h1 className="text-xl font-semibold text-white">Server settings</h1>
      </div>

      {/* Version info */}
      {versionInfo && (
        <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
          <div className="eyebrow mb-1">Server info</div>
          <div className="text-base font-semibold text-white mb-4">relayd {versionInfo.version}</div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <KVCard label="Version" value={versionInfo.version} mono />
            <KVCard label="Commit" value={versionInfo.commit?.slice(0, 12) ?? "—"} mono />
            <KVCard label="Build date" value={versionInfo.build_date ?? "—"} />
            <KVCard label="OS / Arch" value={`${versionInfo.os}/${versionInfo.arch}`} mono />
          </div>
        </div>
      )}

      {/* Global proxy settings */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5">Global proxy / domain routing</div>
          <h2 className="text-base font-semibold text-white">Server-level settings</h2>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Field label="Base Domain">
            <input
              className="text-input"
              value={draft.baseDomain}
              onChange={(e) => setDraft((d) => ({ ...d, baseDomain: e.target.value }))}
              placeholder="yourdomain.com"
            />
          </Field>
          <Field label="Dashboard Host">
            <input
              className="text-input"
              value={draft.dashboardHost}
              onChange={(e) => setDraft((d) => ({ ...d, dashboardHost: e.target.value }))}
              placeholder="admin.yourdomain.com"
            />
          </Field>
        </div>

        {/* ACME listener toggle */}
        <label className="flex items-center gap-3 cursor-pointer select-none">
          <div className="relative">
            <input
              type="checkbox"
              className="sr-only peer"
              checked={draft.acmeDisabled}
              onChange={(e) => setDraft((d) => ({ ...d, acmeDisabled: e.target.checked }))}
            />
            <div className="w-9 h-5 rounded-full border border-white/20 bg-white/[0.06] peer-checked:bg-red-500/80 peer-checked:border-red-500/60 transition-colors" />
            <div className="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white/50 peer-checked:translate-x-4 transition-transform" />
          </div>
          <div>
            <div className="text-sm text-white font-medium">Disable ACME listener</div>
            <div className="text-xs text-white/40">Don&apos;t bind Relay&apos;s local ACME HTTP-01 listener on port 80. Use this when another ACME process already owns port 80.</div>
          </div>
        </label>

        <p className="text-xs text-white/40 leading-relaxed">
          Apps deployed without an explicit <code className="font-mono text-white/50">public_host</code> get an auto-generated subdomain:{" "}
          <code className="font-mono text-white/50">{exampleHost}</code>. Relay starts a Caddy reverse proxy that handles TLS automatically.
          Set <code className="font-mono text-white/50">Dashboard Host</code> to route the Relay admin through Caddy. These values can also be set
          via <code className="font-mono text-white/50">RELAY_BASE_DOMAIN</code> / <code className="font-mono text-white/50">RELAY_DASHBOARD_HOST</code>.
        </p>

        {notice && (
          <div className={cn("rounded-lg px-4 py-3 text-sm border", notice.tone === "ok" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : "bg-red-500/10 border-red-500/30 text-red-400")}>
            {notice.text}
          </div>
        )}

        <button type="button" onClick={save} disabled={busy || !dirty} className={cn("text-sm px-4 py-2 rounded font-semibold transition-colors", dirty ? "bg-white text-black hover:bg-white/90" : "bg-white/[0.06] text-white/30 cursor-not-allowed")}>
          {busy ? "Saving…" : "Save global settings"}
        </button>
      </div>

      {/* How it works */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
        <div className="eyebrow mb-1">How it works</div>
        <h2 className="text-base font-semibold text-white mb-4">Domain routing overview</h2>
        <div className="space-y-2.5">
          {[
            { title: "Auto subdomains", detail: `Set Base Domain here. New deploys auto-get {app}-{branch}.{domain}.` },
            { title: "Dashboard host", detail: "Set Dashboard Host to route the Relay admin through Caddy, e.g. admin.yourdomain.com." },
            { title: "Custom domain per app", detail: "Set Public Host in the app's Settings tab to override the auto-assigned subdomain." },
            { title: "ACME listener", detail: "Relay can run a lightweight HTTP listener on :80 for ACME HTTP-01 challenge files and optional HTTP->HTTPS redirects." },
            { title: "Caddy TLS", detail: "Relay runs a caddy:alpine container (relay-global-proxy) that terminates TLS and proxies to each app." },
            { title: "DNS requirement", detail: "Point your domain or wildcard (*.yourdomain.com) A record at this server's public IP." },
          ].map((row) => (
            <div key={row.title} className="border border-white/[0.06] rounded-lg px-4 py-3">
              <div className="text-sm font-medium text-white">{row.title}</div>
              <div className="text-xs text-white/40 mt-0.5">{row.detail}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Appearance / Themes */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-5">
        <div>
          <div className="eyebrow mb-0.5">Customization</div>
          <h2 className="text-base font-semibold text-white">Appearance</h2>
          <p className="text-xs text-white/40 mt-1">Choose a built-in theme or write your own CSS. Custom CSS is applied after the preset and can override any variable or rule.</p>
        </div>

        {/* Preset theme grid */}
        <div>
          <div className="text-xs text-white/40 mb-2">Theme preset</div>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
            {BUILT_IN_THEMES.map((t) => {
              const active = themeDraft.name === t.id;
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => {
                    setThemeDraft((d) => ({ ...d, name: t.id }));
                    theme.previewTheme(t.id, themeDraft.css);
                  }}
                  className={cn(
                    "relative text-left rounded-lg border p-3 transition-all cursor-pointer",
                    active
                      ? "border-white/40 bg-white/[0.07] ring-1 ring-white/20"
                      : "border-white/[0.06] bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/20"
                  )}
                >
                  {/* Swatch row */}
                  <div className="flex gap-1 mb-2">
                    {t.swatches.map((color, i) => (
                      <span
                        key={i}
                        className="w-4 h-4 rounded-full border border-white/10 flex-shrink-0"
                        style={{ background: color }}
                      />
                    ))}
                  </div>
                  <div className="text-xs font-medium text-white leading-tight">{t.name}</div>
                  <div className="text-[10px] text-white/35 mt-0.5 leading-tight">{t.description}</div>
                  {active && (
                    <span className="absolute top-2 right-2 w-2 h-2 rounded-full bg-white/70" />
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* Custom CSS textarea */}
        <div>
          <div className="text-xs text-white/40 mb-1.5">Custom CSS</div>
          <textarea
            className="text-input font-mono text-xs resize-y"
            rows={8}
            placeholder={`:root {\n  --relay-accent: #7c3aed;\n  --relay-accent-bright: #8b5cf6;\n  /* any CSS variable or rule */\n}`}
            value={themeDraft.css}
            onChange={(e) => {
              const css = e.target.value;
              setThemeDraft((d) => ({ ...d, css }));
              theme.previewTheme(themeDraft.name, css);
            }}
            spellCheck={false}
          />
          <p className="text-[10px] text-white/30 mt-1">
            Raw CSS injected after the preset. Override <code className="font-mono">--relay-accent</code>, <code className="font-mono">--relay-bg</code>, shadcn tokens, or any rule.
            Changes preview live. Hit Save to persist.
          </p>
        </div>

        {appearanceNotice && (
          <div className={cn("rounded-lg px-4 py-3 text-sm border", appearanceNotice.tone === "ok" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : "bg-red-500/10 border-red-500/30 text-red-400")}>
            {appearanceNotice.text}
          </div>
        )}

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={saveAppearance}
            disabled={appearanceBusy || (themeDraft.name === savedThemeName && themeDraft.css === savedCustomCSS)}
            className={cn(
              "text-sm px-4 py-2 rounded font-semibold transition-colors",
              themeDraft.name !== savedThemeName || themeDraft.css !== savedCustomCSS
                ? "bg-white text-black hover:bg-white/90"
                : "bg-white/[0.06] text-white/30 cursor-not-allowed"
            )}
          >
            {appearanceBusy ? "Saving…" : "Save appearance"}
          </button>
          {(themeDraft.name !== savedThemeName || themeDraft.css !== savedCustomCSS) && (
            <button
              type="button"
              onClick={() => {
                setThemeDraft({ name: savedThemeName, css: savedCustomCSS });
                theme.previewTheme(savedThemeName, savedCustomCSS);
              }}
              className="text-xs text-white/40 hover:text-white/70 transition-colors"
            >
              Revert
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

function KVCard({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="bg-white/[0.03] rounded-lg px-3 py-2.5">
      <div className="text-[10px] text-white/35 mb-1">{label}</div>
      <div className={cn("text-sm text-white", mono && "font-mono")}>{value}</div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="text-xs text-white/40 mb-1.5">{label}</div>
      {children}
    </label>
  );
}
