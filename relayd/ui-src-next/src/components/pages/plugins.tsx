"use client";

import { useEffect, useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import {
  getBuildpackPlugins,
  getPluginCatalog,
  getServerConfig,
  installCatalogPlugin,
  installBuildpackPlugin,
  installBuildpackPluginFromURL,
  removeBuildpackPlugin,
  type BuildpackPlugin,
  type PluginCatalogEntry,
} from "@/lib/api";

export function PluginsPage() {
  const [loading, setLoading] = useState(true);
  const [installed, setInstalled] = useState<BuildpackPlugin[]>([]);
  const [catalog, setCatalog] = useState<PluginCatalogEntry[]>([]);
  const [pluginMutationsEnabled, setPluginMutationsEnabled] = useState(false);
  const [search, setSearch] = useState("");
  const [urlInput, setURLInput] = useState("");
  const [urlChecksum, setURLChecksum] = useState("");
  const [jsonInput, setJSONInput] = useState("");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState<{
    tone: "ok" | "warn";
    text: string;
  } | null>(null);

  async function load() {
    setLoading(true);
    try {
      const [installedItems, catalogItems, serverConfig] = await Promise.all([
        getBuildpackPlugins(),
        getPluginCatalog(),
        getServerConfig(),
      ]);
      setInstalled(installedItems ?? []);
      setCatalog(catalogItems ?? []);
      setPluginMutationsEnabled(!!serverConfig.plugin_mutations_enabled);
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Failed to load plugins: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const filteredCatalog = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return catalog;
    return catalog.filter((item) => {
      const haystack = [
        item.name,
        item.description ?? "",
        ...(item.tags ?? []),
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(q);
    });
  }, [catalog, search]);

  function setBusyState(value: string) {
    setBusy(value);
    setNotice(null);
  }

  async function refreshWithNotice(text: string) {
    await load();
    setNotice({ tone: "ok", text });
  }

  async function handleInstallCatalog(name: string) {
    setBusyState(`catalog:${name}`);
    try {
      await installCatalogPlugin(name);
      await refreshWithNotice(`Installed catalog plugin: ${name}`);
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Catalog install failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy("");
    }
  }

  async function handleInstallURL() {
    if (!urlInput.trim()) return;
    setBusyState("url");
    try {
      const installedPlugin = await installBuildpackPluginFromURL(
        urlInput.trim(),
        urlChecksum.trim() || undefined,
      );
      setURLInput("");
      setURLChecksum("");
      await refreshWithNotice(`Installed remote plugin: ${installedPlugin.name}`);
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Remote install failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy("");
    }
  }

  async function installJSONPlugin(raw: string, sourceLabel: string) {
    let plugin: BuildpackPlugin;
    try {
      plugin = JSON.parse(raw) as BuildpackPlugin;
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Invalid plugin JSON from ${sourceLabel}: ${err instanceof Error ? err.message : "parse error"}`,
      });
      return;
    }

    setBusyState(`json:${sourceLabel}`);
    try {
      const installedPlugin = await installBuildpackPlugin(plugin);
      if (sourceLabel === "editor") setJSONInput("");
      await refreshWithNotice(`Installed plugin from ${sourceLabel}: ${installedPlugin.name}`);
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Plugin install failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy("");
    }
  }

  async function handleFilePicked(file: File | null) {
    if (!file) return;
    const text = await file.text();
    await installJSONPlugin(text, file.name);
  }

  async function handleRemove(name: string) {
    const confirmed = window.confirm(`Remove buildpack plugin ${name}?`);
    if (!confirmed) return;
    setBusyState(`remove:${name}`);
    try {
      await removeBuildpackPlugin(name);
      await refreshWithNotice(`Removed plugin: ${name}`);
    } catch (err) {
      setNotice({
        tone: "warn",
        text: `Remove failed: ${err instanceof Error ? err.message : "unknown error"}`,
      });
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Administration</div>
        <h1 className="text-xl font-semibold text-white">Plugins</h1>
        <p className="text-sm text-white/40 mt-1">
          Browse buildpack plugins, install them from a catalog or URL, and manage what is active on this server.
        </p>
      </div>

      <div className={cn(
        "rounded-xl border px-4 py-3 text-sm",
        pluginMutationsEnabled
          ? "border-emerald-500/25 bg-emerald-500/10 text-emerald-300"
          : "border-amber-500/25 bg-amber-500/10 text-amber-300",
      )}>
        {pluginMutationsEnabled
          ? "Plugin installs and removals are enabled on this server."
          : "Plugin installs and removals are disabled. Start relayd with RELAY_ENABLE_PLUGIN_MUTATIONS=true to install or remove from the admin UI."}
      </div>

      {notice && (
        <div className={cn(
          "rounded-lg px-4 py-3 text-sm border",
          notice.tone === "ok"
            ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
            : "bg-amber-500/10 border-amber-500/30 text-amber-400",
        )}>
          {notice.text}
        </div>
      )}

      <div className="grid grid-cols-1 xl:grid-cols-[1.05fr_0.95fr] gap-5">
        <SectionCard title="Installed plugins" eyebrow="Active on this server">
          {loading ? (
            <div className="text-sm text-white/30">Loading installed plugins...</div>
          ) : !installed.length ? (
            <div className="text-sm text-white/25">No buildpack plugins installed yet.</div>
          ) : (
            <div className="space-y-2.5">
              {installed.map((plugin) => (
                <div
                  key={plugin.name}
                  className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-4 py-3"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="text-sm font-semibold text-white">{plugin.name}</div>
                      <div className="text-xs text-white/35 mt-0.5">
                        {plugin.description || "No description"}
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRemove(plugin.name)}
                      disabled={!pluginMutationsEnabled || busy === `remove:${plugin.name}`}
                      className="text-xs border border-red-500/20 text-red-300 px-2.5 py-1 rounded hover:bg-red-500/10 disabled:opacity-40"
                    >
                      {busy === `remove:${plugin.name}` ? "Removing..." : "Remove"}
                    </button>
                  </div>
                  <div className="grid grid-cols-2 gap-2 mt-3 text-xs text-white/40">
                    <MetaRow label="Priority" value={String(plugin.priority ?? 0)} />
                    <MetaRow label="Kind" value={plugin.plan.kind || plugin.name} />
                    <MetaRow label="Build image" value={plugin.plan.build_image || "auto"} />
                    <MetaRow label="Run image" value={plugin.plan.run_image || "auto"} />
                  </div>
                  <div className="text-xs text-white/35 mt-3">
                    Detects with {pluginDetectionSummary(plugin)}
                  </div>
                </div>
              ))}
            </div>
          )}
        </SectionCard>

        <div className="space-y-5">
          <SectionCard title="Plugin catalog" eyebrow="Browse and install">
            <div className="space-y-3">
              <input
                className="text-input"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search Astro, docs, static-site..."
              />
              {!filteredCatalog.length ? (
                <div className="text-sm text-white/25">No catalog items match this search.</div>
              ) : (
                <div className="space-y-2.5">
                  {filteredCatalog.map((item) => (
                    <div
                      key={item.name}
                      className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-4 py-3"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="text-sm font-semibold text-white">{item.name}</div>
                          <div className="text-xs text-white/35 mt-0.5">
                            {item.description || "No description"}
                          </div>
                          {!!item.tags?.length && (
                            <div className="flex flex-wrap gap-1 mt-2">
                              {item.tags.map((tag) => (
                                <span
                                  key={tag}
                                  className="text-[10px] px-2 py-0.5 rounded border border-white/[0.08] bg-white/[0.04] text-white/45"
                                >
                                  {tag}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                        <button
                          type="button"
                          onClick={() => handleInstallCatalog(item.name)}
                          disabled={!pluginMutationsEnabled || item.installed || busy === `catalog:${item.name}`}
                          className="text-xs bg-white text-black font-semibold px-3 py-1.5 rounded hover:bg-white/90 disabled:opacity-40"
                        >
                          {item.installed
                            ? "Installed"
                            : busy === `catalog:${item.name}`
                              ? "Installing..."
                              : "Install"}
                        </button>
                      </div>
                      <div className="flex items-center gap-2 mt-3 text-xs text-white/35 flex-wrap">
                        {item.homepage && (
                          <a href={item.homepage} target="_blank" rel="noreferrer" className="hover:text-white">
                            Homepage
                          </a>
                        )}
                        {item.source_url && (
                          <a href={item.source_url} target="_blank" rel="noreferrer" className="hover:text-white">
                            JSON source
                          </a>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </SectionCard>

          <SectionCard title="Install sources" eyebrow="URL, file, or JSON">
            <div className="space-y-4">
              <div>
                <div className="text-xs text-white/40 mb-1.5">Install from URL</div>
                <div className="space-y-2">
                  <input
                    className="text-input"
                    value={urlInput}
                    onChange={(e) => setURLInput(e.target.value)}
                    placeholder="https://example.com/my-plugin.json"
                  />
                  <input
                    className="text-input font-mono text-xs"
                    value={urlChecksum}
                    onChange={(e) => setURLChecksum(e.target.value)}
                    placeholder="Optional SHA256 checksum"
                  />
                  <div className="flex items-center justify-between gap-3">
                    <div className="text-[11px] text-white/35">
                      Remote installs require HTTPS. Add a SHA256 to pin the exact plugin JSON you expect.
                    </div>
                    <button
                      type="button"
                      onClick={handleInstallURL}
                      disabled={!pluginMutationsEnabled || !urlInput.trim() || busy === "url"}
                      className="primary-btn shrink-0"
                    >
                      {busy === "url" ? "Installing..." : "Install URL"}
                    </button>
                  </div>
                </div>
              </div>

              <div>
                <div className="text-xs text-white/40 mb-1.5">Install from file</div>
                <label className="inline-flex items-center gap-2 text-sm border border-white/[0.10] text-white/70 hover:text-white px-3 py-2 rounded cursor-pointer hover:bg-white/[0.06] transition-colors">
                  <input
                    type="file"
                    accept=".json,application/json"
                    className="hidden"
                    onChange={(e) => {
                      const file = e.target.files?.[0] ?? null;
                      void handleFilePicked(file);
                      e.currentTarget.value = "";
                    }}
                    disabled={!pluginMutationsEnabled}
                  />
                  Choose plugin JSON
                </label>
              </div>

              <div>
                <div className="text-xs text-white/40 mb-1.5">Paste plugin JSON</div>
                <textarea
                  className="text-input min-h-[180px] resize-y font-mono text-xs"
                  value={jsonInput}
                  onChange={(e) => setJSONInput(e.target.value)}
                  placeholder='{"name":"my-plugin","detect":{"files_any":["foo.txt"]},"plan":{"dockerfile_template":"FROM nginx:alpine","service_port":80}}'
                />
                <div className="mt-2">
                  <button
                    type="button"
                    onClick={() => void installJSONPlugin(jsonInput, "editor")}
                    disabled={!pluginMutationsEnabled || !jsonInput.trim() || busy === "json:editor"}
                    className="primary-btn"
                  >
                    {busy === "json:editor" ? "Installing..." : "Install JSON"}
                  </button>
                </div>
              </div>
            </div>
          </SectionCard>
        </div>
      </div>
    </div>
  );
}

function pluginDetectionSummary(plugin: BuildpackPlugin) {
  const bits: string[] = [];
  if (plugin.detect.files_any?.length) bits.push(`files: ${plugin.detect.files_any.join(", ")}`);
  if (plugin.detect.package_deps_any?.length) bits.push(`deps: ${plugin.detect.package_deps_any.join(", ")}`);
  if (plugin.detect.dirs_any?.length) bits.push(`dirs: ${plugin.detect.dirs_any.join(", ")}`);
  if (plugin.detect.file_extensions_any?.length) bits.push(`ext: ${plugin.detect.file_extensions_any.join(", ")}`);
  return bits.length ? bits.join(" • ") : "custom detect rules";
}

function SectionCard({
  title,
  eyebrow,
  children,
}: {
  title: string;
  eyebrow: string;
  children: React.ReactNode;
}) {
  return (
    <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
      <div>
        <div className="eyebrow mb-0.5">{eyebrow}</div>
        <h2 className="text-base font-semibold text-white">{title}</h2>
      </div>
      {children}
    </div>
  );
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white/[0.03] rounded px-2.5 py-2">
      <div className="text-[10px] text-white/30 mb-0.5">{label}</div>
      <div className="text-white/70 break-all">{value}</div>
    </div>
  );
}
