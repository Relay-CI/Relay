package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------- Buildpack Plugins API ----------------------

func (s *Server) pluginBuildpacksDir() string {
	return filepath.Join(s.pluginsDir, "buildpacks")
}

func (s *Server) reloadBuildpacks() error {
	plugins, err := s.loadPluginBuildpacks()
	if err != nil {
		return err
	}
	packs := make([]Buildpack, 0, len(plugins)+len(defaultBuildpacks()))
	for _, plugin := range plugins {
		packs = append(packs, &PluginBuildpack{plugin: plugin})
	}
	packs = append(packs, defaultBuildpacks()...)
	s.buildpacks = packs
	return nil
}

func (s *Server) loadPluginBuildpacks() ([]*BuildpackPlugin, error) {
	dir := s.pluginBuildpacksDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var plugins []*BuildpackPlugin
	for _, ent := range ents {
		if ent.IsDir() || !strings.HasSuffix(strings.ToLower(ent.Name()), ".json") {
			continue
		}
		p := filepath.Join(dir, ent.Name())
		plugin, err := readBuildpackPluginFile(p)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	sort.SliceStable(plugins, func(i, j int) bool {
		if plugins[i].Priority == plugins[j].Priority {
			return plugins[i].Name < plugins[j].Name
		}
		return plugins[i].Priority > plugins[j].Priority
	})
	return plugins, nil
}

func readBuildpackPluginFile(p string) (*BuildpackPlugin, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var plugin BuildpackPlugin
	if err := json.Unmarshal(b, &plugin); err != nil {
		return nil, err
	}
	if err := validateBuildpackPlugin(&plugin); err != nil {
		return nil, err
	}
	return &plugin, nil
}

func validateBuildpackPlugin(plugin *BuildpackPlugin) error {
	name := safe(plugin.Name)
	if name == "" || name == "x" {
		return fmt.Errorf("plugin name required")
	}
	plugin.Name = name
	plugin.PlanSpec.Kind = firstNonEmpty(strings.TrimSpace(plugin.PlanSpec.Kind), plugin.Name)
	if plugin.PlanSpec.ServicePort < 0 {
		return fmt.Errorf("service_port must be >= 0")
	}
	if strings.TrimSpace(plugin.PlanSpec.DockerfileTemplate) == "" {
		return fmt.Errorf("dockerfile_template required")
	}
	rules := plugin.DetectRules
	if strings.TrimSpace(rules.Kind) == "" &&
		len(rules.Kinds) == 0 &&
		len(rules.FilesAny) == 0 &&
		len(rules.FilesAll) == 0 &&
		len(rules.DirsAny) == 0 &&
		len(rules.DirsAll) == 0 &&
		len(rules.PackageDepsAny) == 0 &&
		len(rules.PackageDepsAll) == 0 &&
		len(rules.FileExtensionsAny) == 0 &&
		len(rules.FileExtensionsAll) == 0 {
		return fmt.Errorf("plugin %s must define at least one detect rule", plugin.Name)
	}
	return nil
}

type pluginCatalogSource struct {
	Name        string
	Description string
	Tags        []string
	Homepage    string
	SourceURL   string
	Plugin      BuildpackPlugin
}

type PluginCatalogEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	SourceURL   string   `json:"source_url,omitempty"`
	Installed   bool     `json:"installed"`
}

var builtInPluginCatalog = []pluginCatalogSource{
	{
		Name:        "astro-static",
		Description: "Build and serve Astro static sites with nginx.",
		Tags:        []string{"astro", "static-site", "frontend"},
		Homepage:    "https://astro.build",
		SourceURL:   "https://raw.githubusercontent.com/babymonie/relay/main/plugins/astro-static.json",
		Plugin: BuildpackPlugin{
			Name:        "astro-static",
			Description: "Astro static site buildpack plugin",
			Priority:    900,
			DetectRules: PluginDetectRules{
				FilesAny:       []string{"astro.config.mjs", "astro.config.js", "astro.config.ts"},
				PackageDepsAny: []string{"astro"},
			},
			PlanSpec: PluginPlanSpec{
				Kind:               "astro-static",
				ServicePort:        80,
				BuildImage:         "node:22",
				RunImage:           "nginx:alpine",
				InstallCmd:         "npm install",
				BuildCmd:           "npm run build",
				DockerfileTemplate: "FROM {{ .BuildImage }} AS builder\nWORKDIR /app\nCOPY package.json ./\nCOPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./\nRUN {{ .InstallCmd }}\nCOPY . .\nRUN {{ .BuildCmd }}\n\nFROM {{ .RunImage }}\nCOPY --from=builder /app/dist /usr/share/nginx/html\nCOPY default.conf /etc/nginx/conf.d/default.conf\nEXPOSE {{ .ServicePort }}\n",
				WriteDefaultConf:   true,
				CleanupPaths:       []string{"node_modules", "dist"},
			},
		},
	},
	{
		Name:        "docusaurus-static",
		Description: "Build Docusaurus docs sites and serve the generated static output.",
		Tags:        []string{"docusaurus", "docs", "static-site"},
		Homepage:    "https://docusaurus.io",
		Plugin: BuildpackPlugin{
			Name:        "docusaurus-static",
			Description: "Docusaurus static site buildpack plugin",
			Priority:    880,
			DetectRules: PluginDetectRules{
				FilesAny:       []string{"docusaurus.config.js", "docusaurus.config.ts", "docusaurus.config.mjs"},
				PackageDepsAny: []string{"@docusaurus/core"},
			},
			PlanSpec: PluginPlanSpec{
				Kind:               "docusaurus-static",
				ServicePort:        80,
				BuildImage:         "node:22",
				RunImage:           "nginx:alpine",
				InstallCmd:         "npm install",
				BuildCmd:           "npm run build",
				DockerfileTemplate: "FROM {{ .BuildImage }} AS builder\nWORKDIR /app\nCOPY package.json ./\nCOPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./\nRUN {{ .InstallCmd }}\nCOPY . .\nRUN {{ .BuildCmd }}\n\nFROM {{ .RunImage }}\nCOPY --from=builder /app/build /usr/share/nginx/html\nCOPY default.conf /etc/nginx/conf.d/default.conf\nEXPOSE {{ .ServicePort }}\n",
				WriteDefaultConf:   true,
				CleanupPaths:       []string{"node_modules", "build"},
			},
		},
	},
	{
		Name:        "eleventy-static",
		Description: "Build Eleventy projects into a static nginx site.",
		Tags:        []string{"11ty", "eleventy", "static-site"},
		Homepage:    "https://www.11ty.dev",
		Plugin: BuildpackPlugin{
			Name:        "eleventy-static",
			Description: "Eleventy static site buildpack plugin",
			Priority:    860,
			DetectRules: PluginDetectRules{
				FilesAny:       []string{".eleventy.js", ".eleventy.cjs", ".eleventy.mjs"},
				PackageDepsAny: []string{"@11ty/eleventy"},
			},
			PlanSpec: PluginPlanSpec{
				Kind:               "eleventy-static",
				ServicePort:        80,
				BuildImage:         "node:22",
				RunImage:           "nginx:alpine",
				InstallCmd:         "npm install",
				BuildCmd:           "npx @11ty/eleventy",
				DockerfileTemplate: "FROM {{ .BuildImage }} AS builder\nWORKDIR /app\nCOPY package.json ./\nCOPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./\nRUN {{ .InstallCmd }}\nCOPY . .\nRUN {{ .BuildCmd }}\n\nFROM {{ .RunImage }}\nCOPY --from=builder /app/_site /usr/share/nginx/html\nCOPY default.conf /etc/nginx/conf.d/default.conf\nEXPOSE {{ .ServicePort }}\n",
				WriteDefaultConf:   true,
				CleanupPaths:       []string{"node_modules", "_site"},
			},
		},
	},
}

func cloneBuildpackPlugin(plugin BuildpackPlugin) BuildpackPlugin {
	return plugin
}

func (s *Server) pluginCatalogEntries() ([]PluginCatalogEntry, error) {
	plugins, err := s.loadPluginBuildpacks()
	if err != nil {
		return nil, err
	}
	installed := map[string]bool{}
	for _, plugin := range plugins {
		installed[plugin.Name] = true
	}
	out := make([]PluginCatalogEntry, 0, len(builtInPluginCatalog))
	for _, item := range builtInPluginCatalog {
		out = append(out, PluginCatalogEntry{
			Name:        item.Name,
			Description: item.Description,
			Tags:        append([]string(nil), item.Tags...),
			Homepage:    item.Homepage,
			SourceURL:   item.SourceURL,
			Installed:   installed[item.Name],
		})
	}
	return out, nil
}

func buildpackPluginFromCatalog(name string) (*BuildpackPlugin, error) {
	target := safe(strings.TrimSpace(name))
	if target == "" || target == "x" {
		return nil, fmt.Errorf("plugin name required")
	}
	for _, item := range builtInPluginCatalog {
		if item.Name == target {
			plugin := cloneBuildpackPlugin(item.Plugin)
			if err := validateBuildpackPlugin(&plugin); err != nil {
				return nil, err
			}
			return &plugin, nil
		}
	}
	return nil, fmt.Errorf("catalog plugin not found")
}

func fetchBuildpackPluginFromURL(rawURL string, expectedSHA256 string) (*BuildpackPlugin, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("plugin URL must use https")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch plugin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch plugin: remote returned %s", resp.Status)
	}
	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read plugin JSON: %w", err)
	}
	if checksum := strings.ToLower(strings.TrimSpace(expectedSHA256)); checksum != "" {
		sum := sha256.Sum256(rawBody)
		if hex.EncodeToString(sum[:]) != checksum {
			return nil, fmt.Errorf("plugin checksum mismatch")
		}
	}
	var plugin BuildpackPlugin
	if err := json.Unmarshal(rawBody, &plugin); err != nil {
		return nil, fmt.Errorf("decode plugin JSON: %w", err)
	}
	if err := validateBuildpackPlugin(&plugin); err != nil {
		return nil, err
	}
	return &plugin, nil
}

func (s *Server) installBuildpackPlugin(plugin *BuildpackPlugin) error {
	if err := validateBuildpackPlugin(plugin); err != nil {
		return err
	}
	path := filepath.Join(s.pluginBuildpacksDir(), safe(plugin.Name)+".json")
	body, _ := json.MarshalIndent(plugin, "", "  ")
	if err := os.WriteFile(path, body, 0644); err != nil {
		return err
	}
	return s.reloadBuildpacks()
}

func (s *Server) handleBuildpackPluginCatalog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		entries, err := s.pluginCatalogEntries()
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, entries)
	case http.MethodPost:
		if !s.pluginMutationsEnabled() {
			httpError(w, 403, "plugin mutations are disabled on this server")
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		plugin, err := buildpackPluginFromCatalog(body.Name)
		if err != nil {
			httpError(w, 400, err.Error())
			return
		}
		if err := s.installBuildpackPlugin(plugin); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 201, plugin)
	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) handleBuildpackPluginInstallURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	if !s.pluginMutationsEnabled() {
		httpError(w, 403, "plugin mutations are disabled on this server")
		return
	}
	var body struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	plugin, err := fetchBuildpackPluginFromURL(body.URL, body.SHA256)
	if err != nil {
		httpError(w, 400, err.Error())
		return
	}
	if err := s.installBuildpackPlugin(plugin); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, plugin)
}

func (s *Server) handleBuildpackPlugins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		plugins, err := s.loadPluginBuildpacks()
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, plugins)
	case http.MethodPost:
		if !s.pluginMutationsEnabled() {
			httpError(w, 403, "plugin mutations are disabled on this server")
			return
		}
		var plugin BuildpackPlugin
		if err := json.NewDecoder(r.Body).Decode(&plugin); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		if err := s.installBuildpackPlugin(&plugin); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 201, plugin)
	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) handleBuildpackPluginByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/plugins/buildpacks/")
	name = safe(path.Base(strings.TrimSpace(name)))
	if name == "" || name == "x" {
		httpError(w, 400, "plugin name required")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if !s.pluginMutationsEnabled() {
			httpError(w, 403, "plugin mutations are disabled on this server")
			return
		}
		if err := os.Remove(filepath.Join(s.pluginBuildpacksDir(), name+".json")); err != nil {
			if os.IsNotExist(err) {
				httpError(w, 404, "plugin not found")
				return
			}
			httpError(w, 500, err.Error())
			return
		}
		if err := s.reloadBuildpacks(); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"deleted": name})
	default:
		httpError(w, 405, "method not allowed")
	}
}
