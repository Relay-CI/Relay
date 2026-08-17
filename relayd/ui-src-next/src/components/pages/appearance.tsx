"use client";

import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import { getServerConfig, saveServerConfig } from "@/lib/api";
import { useTheme, BUILT_IN_THEMES } from "@/components/theme-provider";

type CurrentUser = { username: string; role: string } | null;

interface AppearancePageProps {
  currentUser: CurrentUser;
}

const FONT_OPTIONS = [
  { id: "system", label: "System UI", css: "" },
  { id: "inter", label: "Inter", css: "@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap'); :root { --relay-font: 'Inter', sans-serif; } body { font-family: var(--relay-font); }" },
  { id: "geist", label: "Geist", css: "@import url('https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600;700&display=swap'); :root { --relay-font: 'Geist', sans-serif; } body { font-family: var(--relay-font); }" },
  { id: "jetbrains", label: "JetBrains Mono", css: "@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&display=swap'); :root { --relay-font: 'JetBrains Mono', monospace; } body { font-family: var(--relay-font); }" },
  { id: "ibm", label: "IBM Plex Mono", css: "@import url('https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600&display=swap'); :root { --relay-font: 'IBM Plex Mono', monospace; } body { font-family: var(--relay-font); }" },
];

const RADIUS_OPTIONS = [
  { id: "sharp", label: "Sharp", css: ":root { --radius: 0px; --relay-radius: 0px; } .rounded, .rounded-lg, .rounded-xl, .rounded-md, .rounded-sm { border-radius: 0 !important; }" },
  { id: "default", label: "Default", css: "" },
  { id: "rounded", label: "Rounded", css: ":root { --radius: 12px; } .rounded { border-radius: 8px; } .rounded-lg { border-radius: 12px; } .rounded-xl { border-radius: 16px; } .rounded-md { border-radius: 8px; }" },
  { id: "pill", label: "Pill", css: ":root { --radius: 9999px; } .rounded, .rounded-md, .rounded-lg, .rounded-xl { border-radius: 24px; }" },
];

const DENSITY_OPTIONS = [
  { id: "compact", label: "Compact", css: ":root { --relay-spacing: 0.75; } .p-5 { padding: 1rem; } .p-3 { padding: 0.5rem; } .gap-3 { gap: 0.5rem; } .gap-5 { gap: 0.75rem; }" },
  { id: "default", label: "Default", css: "" },
  { id: "comfortable", label: "Comfortable", css: ":root { --relay-spacing: 1.25; } .p-5 { padding: 1.75rem; } .p-3 { padding: 1rem; } .gap-3 { gap: 1rem; } .gap-5 { gap: 1.5rem; }" },
];

const SIDEBAR_STYLE_OPTIONS = [
  { id: "default", label: "Default", css: "" },
  { id: "bordered", label: "Bordered", css: "aside { border-right: 1px solid rgba(255,255,255,0.12) !important; background: rgba(0,0,0,0.6) !important; backdrop-filter: blur(12px); }" },
  { id: "flat", label: "Flat", css: "aside { background: var(--relay-bg) !important; border-right: none !important; }" },
  { id: "accent", label: "Accent tint", css: "aside { background: color-mix(in srgb, var(--relay-accent) 6%, #000) !important; }" },
];

export function AppearancePage({ currentUser }: AppearancePageProps) {
  const isOwner = currentUser?.role === "owner";
  const theme = useTheme();

  const [savedThemeName, setSavedThemeName] = useState("default");
  const [savedCustomCSS, setSavedCustomCSS] = useState("");
  const [themeDraft, setThemeDraft] = useState({ name: "default", css: "" });
  const [fontId, setFontId] = useState("system");
  const [radiusId, setRadiusId] = useState("default");
  const [densityId, setDensityId] = useState("default");
  const [sidebarStyleId, setSidebarStyleId] = useState("default");
  const [showCustomCSS, setShowCustomCSS] = useState(false);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "err"; text: string } | null>(null);

  useEffect(() => {
    if (!isOwner) return;
    getServerConfig().then((data) => {
      const tn = data?.theme_name ?? "default";
      const tc = data?.theme_css ?? "";
      setSavedThemeName(tn);
      setSavedCustomCSS(tc);
      setThemeDraft({ name: tn, css: tc });
    }).catch((err) => {
      setNotice({
        tone: "err",
        text: `Failed to load saved appearance: ${err instanceof Error ? err.message : "unknown error"}. Showing defaults, not your saved theme.`,
      });
    });
  }, [isOwner]);

  function buildCompositeCSS(themeCSS: string, font: string, radius: string, density: string, sidebar: string, custom: string) {
    const parts = [
      FONT_OPTIONS.find((f) => f.id === font)?.css ?? "",
      RADIUS_OPTIONS.find((r) => r.id === radius)?.css ?? "",
      DENSITY_OPTIONS.find((d) => d.id === density)?.css ?? "",
      SIDEBAR_STYLE_OPTIONS.find((s) => s.id === sidebar)?.css ?? "",
      custom,
    ].filter(Boolean);
    return parts.join("\n");
  }

  function handleThemeChange(id: string) {
    setThemeDraft((d) => ({ ...d, name: id }));
    theme.previewTheme(id, themeDraft.css);
  }

  function handleCustomCSSChange(css: string) {
    setThemeDraft((d) => ({ ...d, css }));
  }

  async function save() {
    if (!isOwner) return;
    setBusy(true);
    setNotice(null);
    try {
      const fullCSS = buildCompositeCSS(
        themeDraft.css,
        fontId,
        radiusId,
        densityId,
        sidebarStyleId,
        themeDraft.css,
      );
      const saved = await saveServerConfig({
        theme_name: themeDraft.name,
        theme_css: fullCSS,
      });
      const tn = saved?.theme_name ?? "default";
      const tc = saved?.theme_css ?? "";
      setSavedThemeName(tn);
      setSavedCustomCSS(tc);
      setThemeDraft({ name: tn, css: tc });
      theme.previewTheme(tn, tc);
      setNotice({ tone: "ok", text: "Appearance saved and applied." });
    } catch (err) {
      setNotice({ tone: "err", text: err instanceof Error ? err.message : "Save failed" });
    } finally {
      setBusy(false);
    }
  }

  const isDirty =
    themeDraft.name !== savedThemeName ||
    themeDraft.css !== savedCustomCSS;

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Customization</div>
        <h1 className="text-xl font-semibold text-white">Appearance</h1>
        <p className="text-sm text-white/40 mt-1">
          Customize the look and feel of the Relay control room.
          {!isOwner && " Some options require owner access to persist across sessions."}
        </p>
      </div>

      {/* Color themes */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5">Color theme</div>
          <h2 className="text-base font-semibold text-white">Theme preset</h2>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {BUILT_IN_THEMES.map((t) => {
            const active = themeDraft.name === t.id;
            return (
              <button
                key={t.id}
                type="button"
                onClick={() => handleThemeChange(t.id)}
                className={cn(
                  "relative text-left rounded-lg border p-3 transition-all cursor-pointer",
                  active
                    ? "border-white/40 bg-white/[0.07] ring-1 ring-white/20"
                    : "border-white/[0.06] bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/20"
                )}
              >
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
                {active && <span className="absolute top-2 right-2 w-2 h-2 rounded-full bg-white/70" />}
              </button>
            );
          })}
        </div>
      </div>

      {/* Typography */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5">Typography</div>
          <h2 className="text-base font-semibold text-white">UI font</h2>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {FONT_OPTIONS.map((f) => (
            <button
              key={f.id}
              type="button"
              onClick={() => setFontId(f.id)}
              className={cn(
                "text-left rounded-lg border px-3 py-2.5 transition-all cursor-pointer",
                fontId === f.id
                  ? "border-white/40 bg-white/[0.07] ring-1 ring-white/20"
                  : "border-white/[0.06] bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/20"
              )}
            >
              <div className="text-xs font-medium text-white">{f.label}</div>
              <div className="text-[10px] text-white/35 mt-0.5 font-mono">Aa 0–9</div>
            </button>
          ))}
        </div>
      </div>

      {/* Border radius */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5">Geometry</div>
          <h2 className="text-base font-semibold text-white">Border radius</h2>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
          {RADIUS_OPTIONS.map((r) => (
            <button
              key={r.id}
              type="button"
              onClick={() => setRadiusId(r.id)}
              className={cn(
                "text-left border px-3 py-2.5 transition-all cursor-pointer",
                r.id === "sharp" ? "rounded-none" : r.id === "pill" ? "rounded-full" : r.id === "rounded" ? "rounded-2xl" : "rounded-lg",
                radiusId === r.id
                  ? "border-white/40 bg-white/[0.07] ring-1 ring-white/20"
                  : "border-white/[0.06] bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/20"
              )}
            >
              <div className="text-xs font-medium text-white">{r.label}</div>
            </button>
          ))}
        </div>
      </div>

      {/* Density */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5">Layout</div>
          <h2 className="text-base font-semibold text-white">Spacing density</h2>
        </div>
        <div className="seg-control flex">
          {DENSITY_OPTIONS.map((d) => (
            <button
              key={d.id}
              type="button"
              onClick={() => setDensityId(d.id)}
              className={cn("seg-btn flex-1 text-xs py-1.5", densityId === d.id && "seg-btn--active")}
            >
              {d.label}
            </button>
          ))}
        </div>
      </div>

      {/* Sidebar style */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <div className="eyebrow mb-0.5">Navigation</div>
          <h2 className="text-base font-semibold text-white">Sidebar style</h2>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
          {SIDEBAR_STYLE_OPTIONS.map((s) => (
            <button
              key={s.id}
              type="button"
              onClick={() => setSidebarStyleId(s.id)}
              className={cn(
                "text-left rounded-lg border px-3 py-2.5 transition-all cursor-pointer",
                sidebarStyleId === s.id
                  ? "border-white/40 bg-white/[0.07] ring-1 ring-white/20"
                  : "border-white/[0.06] bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/20"
              )}
            >
              <div className="text-xs font-medium text-white">{s.label}</div>
            </button>
          ))}
        </div>
      </div>

      {/* Custom CSS */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="eyebrow mb-0.5">Advanced</div>
            <h2 className="text-base font-semibold text-white">Custom CSS</h2>
            <p className="text-xs text-white/40 mt-1">Injected after all other styles. Overrides any variable or rule.</p>
          </div>
          <button
            type="button"
            onClick={() => setShowCustomCSS((v) => !v)}
            className="text-xs text-white/40 hover:text-white px-2 py-1 rounded hover:bg-white/[0.06] transition-colors"
          >
            {showCustomCSS ? "Hide" : "Edit"}
          </button>
        </div>
        {showCustomCSS && (
          <textarea
            className="text-input w-full font-mono text-xs"
            rows={10}
            placeholder={`:root {\n  --relay-accent: #7c3aed;\n  --relay-bg: #0f0a1e;\n}`}
            value={themeDraft.css}
            onChange={(e) => handleCustomCSSChange(e.target.value)}
          />
        )}
      </div>

      {!isOwner && (
        <div className="bg-amber-500/10 border border-amber-500/30 rounded-xl px-4 py-3 text-sm text-amber-400">
          You can preview themes, but saving requires owner access.
        </div>
      )}

      {notice && (
        <div className={cn(
          "rounded-lg px-4 py-3 text-sm border",
          notice.tone === "ok"
            ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
            : "bg-red-500/10 border-red-500/30 text-red-400"
        )}>
          {notice.text}
        </div>
      )}

      {isOwner && (
        <button
          type="button"
          onClick={save}
          disabled={busy || !isDirty}
          className={cn(
            "text-sm px-4 py-2 rounded font-semibold transition-colors",
            isDirty
              ? "bg-white text-black hover:bg-white/90"
              : "bg-white/[0.06] text-white/30 cursor-not-allowed"
          )}
        >
          {busy ? "Saving…" : "Save appearance"}
        </button>
      )}
    </div>
  );
}
