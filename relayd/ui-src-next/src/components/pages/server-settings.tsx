"use client";

import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import { getServerConfig, saveServerConfig, getVersion, type DoctorReport, type VersionInfo } from "@/lib/api";

type CurrentUser = { username: string; role: string } | null;

interface ServerSettingsPageProps {
  currentUser: CurrentUser;
}


export function ServerSettingsPage({ currentUser }: ServerSettingsPageProps) {
  const isOwner = currentUser?.role === "owner";

  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null);
  const [baseDomain, setBaseDomain] = useState("");
  const [dashboardHost, setDashboardHost] = useState("");
  const [acmeDisabled, setAcmeDisabled] = useState(false);
  const [doctor, setDoctor] = useState<DoctorReport | null>(null);
  const [draft, setDraft] = useState({ baseDomain: "", dashboardHost: "", acmeDisabled: false });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "danger"; text: string } | null>(null);

  useEffect(() => {
    getVersion().then((v) => setVersionInfo(v)).catch(() => {});
    if (isOwner) {
      getServerConfig().then((data) => {
        const bd = data?.base_domain ?? "";
        const dh = data?.dashboard_host ?? "";
        const ad = data?.acme_disabled === "true";
        setDoctor((data?.doctor as DoctorReport | undefined) ?? null);
        setBaseDomain(bd); setDashboardHost(dh); setAcmeDisabled(ad);
        setDraft({ baseDomain: bd, dashboardHost: dh, acmeDisabled: ad });
      }).catch(() => {});
    }
  }, [isOwner]);

  if (!isOwner) {
    return <div className="flex items-center justify-center h-full text-white/30 text-sm">Owner access required</div>;
  }

  const dirty = draft.baseDomain !== baseDomain || draft.dashboardHost !== dashboardHost || draft.acmeDisabled !== acmeDisabled;
  const exampleHost = draft.baseDomain ? `myapp-main.${draft.baseDomain}` : "myapp-main.example.com";
  const checks = doctor?.checks ?? {};

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
      setDoctor((saved?.doctor as DoctorReport | undefined) ?? null);
      setBaseDomain(bd); setDashboardHost(dh); setAcmeDisabled(ad);
      setDraft({ baseDomain: bd, dashboardHost: dh, acmeDisabled: ad });
      setNotice({ tone: "ok", text: "Saved. Caddy will route the dashboard host back to Relay, and new deploys without an explicit public host will auto-assign a subdomain." });
    } catch (err) {
      setNotice({ tone: "danger", text: err instanceof Error ? err.message : "Save failed" });
    } finally { setBusy(false); }
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

      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5">Guided setup</div>
          <h2 className="text-base font-semibold text-white">DNS / TLS rollout</h2>
          <p className="text-xs text-white/40 mt-1">
            Save the host values above, then use these checks to verify DNS, HTTPS, Docker, and the helper listeners.
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
            <div className="text-xs text-white/35 mb-1">Managed example URL</div>
            <div className="text-sm text-white font-mono break-all">{doctor.managed_example_url}</div>
          </div>
        )}
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

      {/* Appearance moved notice */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl px-5 py-4 flex items-center gap-3">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-white/30 shrink-0">
          <circle cx="12" cy="12" r="10"/>
          <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
          <line x1="2" y1="12" x2="22" y2="12"/>
        </svg>
        <div>
          <div className="text-sm font-medium text-white/70">Appearance settings have moved</div>
          <div className="text-xs text-white/35 mt-0.5">Use the globe icon in the topbar to access themes, fonts, and layout options.</div>
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

function GuideStep({ index, title, detail }: { index: string; title: string; detail: string }) {
  return (
    <div className="flex gap-3">
      <div className="w-6 h-6 shrink-0 rounded-full bg-white/[0.06] border border-white/[0.1] text-xs text-white/60 flex items-center justify-center">
        {index}
      </div>
      <div>
        <div className="text-sm font-medium text-white">{title}</div>
        <div className="text-xs text-white/40 mt-0.5 leading-relaxed">{detail}</div>
      </div>
    </div>
  );
}

function CheckRow({ label, check }: { label: string; check?: DoctorReport["checks"][string] }) {
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
          <div className="text-xs text-white/40 mt-0.5">{check?.summary ?? "Waiting for diagnostics"}</div>
        </div>
        <span className={cn("text-[10px] uppercase tracking-wider font-semibold px-2 py-0.5 rounded border", toneClass)}>
          {tone}
        </span>
      </div>
      {check?.detail && <div className="text-[11px] text-white/35 mt-2 break-all">{check.detail}</div>}
      {check?.hint && <div className="text-[11px] text-white/28 mt-1">{check.hint}</div>}
    </div>
  );
}
