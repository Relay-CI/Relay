package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ---------------------- Buildpacks ----------------------

type BuildPlan struct {
	Kind        string
	ServicePort int

	BuildImage string
	RunImage   string

	InstallCmd string
	BuildCmd   string
	StartCmd   string

	// Create / overwrite Dockerfile unless Config.Dockerfile is used
	WriteDockerfile func(repoDir string) error

	Verify  func(repoDir string) error
	Cleanup func(repoDir string) error
}

// Buildpack interface for automatic detection and plan generation
type Buildpack interface {
	Name() string
	Detect(repoDir string, cfg *RelayConfig) bool
	Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error)
}

type PluginDetectRules struct {
	Kind              string   `json:"kind,omitempty"`
	Kinds             []string `json:"kinds,omitempty"`
	FilesAny          []string `json:"files_any,omitempty"`
	FilesAll          []string `json:"files_all,omitempty"`
	DirsAny           []string `json:"dirs_any,omitempty"`
	DirsAll           []string `json:"dirs_all,omitempty"`
	PackageDepsAny    []string `json:"package_deps_any,omitempty"`
	PackageDepsAll    []string `json:"package_deps_all,omitempty"`
	FileExtensionsAny []string `json:"file_extensions_any,omitempty"`
	FileExtensionsAll []string `json:"file_extensions_all,omitempty"`
}

type PluginPlanSpec struct {
	Kind               string   `json:"kind"`
	ServicePort        int      `json:"service_port,omitempty"`
	BuildImage         string   `json:"build_image,omitempty"`
	RunImage           string   `json:"run_image,omitempty"`
	InstallCmd         string   `json:"install_cmd,omitempty"`
	BuildCmd           string   `json:"build_cmd,omitempty"`
	StartCmd           string   `json:"start_cmd,omitempty"`
	DockerfileTemplate string   `json:"dockerfile_template"`
	WriteDefaultConf   bool     `json:"write_default_conf,omitempty"`
	WasmMime           bool     `json:"wasm_mime,omitempty"`
	CleanupPaths       []string `json:"cleanup_paths,omitempty"`
}

type BuildpackPlugin struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	DetectRules PluginDetectRules `json:"detect"`
	PlanSpec    PluginPlanSpec    `json:"plan"`
}

type PluginBuildpack struct {
	plugin *BuildpackPlugin
}

// monorepoSubdirHint scans one level into common monorepo layouts (apps/*,
// packages/*, services/*) for a directory that matches a buildpack when the
// repo root itself didn't match anything. Buildpack detection only looks at
// the given root — this doesn't auto-deploy the subdir, it just turns a bare
// "no buildpack matched" into an actionable pointer for the common
// apps/web + apps/api monorepo shape instead of a dead end.
func monorepoSubdirHint(repoDir string, packs []Buildpack, cfg *RelayConfig) string {
	for _, parent := range []string{"apps", "packages", "services"} {
		entries, err := os.ReadDir(filepath.Join(repoDir, parent))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			sub := filepath.ToSlash(filepath.Join(parent, e.Name()))
			for _, bp := range packs {
				if bp.Detect(filepath.Join(repoDir, parent, e.Name()), cfg) {
					return fmt.Sprintf(" Found a %s project at %s/ — set \"project_root\": %q in relay.config.json to deploy it.", bp.Name(), sub, sub)
				}
			}
		}
	}
	return ""
}

// defaultBuildpacks returns the ordered list of buildpacks we support.
func defaultBuildpacks() []Buildpack {
	return []Buildpack{
		&NodeNextStandaloneBuildpack{},
		&NodeNextBuildpack{},
		// SvelteKit and Remix must be checked before NodeVite: both also ship
		// a vite.config.ts, and NodeVite's static-SPA assumption is wrong for
		// either (see SvelteKitBuildpack/RemixBuildpack doc comments).
		&SvelteKitBuildpack{},
		&RemixBuildpack{},
		&NodeViteBuildpack{},
		&ExpoWebBuildpack{},
		&NuxtBuildpack{},
		&PHPBuildpack{},
		&SprintUIBuildpack{},
		&BunBuildpack{},
		&NodeGenericBuildpack{},
		&GoBuildpack{},
		&DotnetBuildpack{},
		&PythonBuildpack{},
		&RubyBuildpack{},
		&JavaBuildpack{},
		&RustBuildpack{},
		&CCppBuildpack{},
		&WasmStaticBuildpack{},
		&StaticBuildpack{},
	}
}

func (b *PluginBuildpack) Name() string { return b.plugin.Name }

func (b *PluginBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	rules := b.plugin.DetectRules
	if cfg != nil && cfg.Kind != "" {
		want := strings.ToLower(strings.TrimSpace(cfg.Kind))
		if want != "" {
			if strings.EqualFold(rules.Kind, want) {
				return true
			}
			for _, k := range rules.Kinds {
				if strings.EqualFold(k, want) {
					return true
				}
			}
		}
	}
	if !pluginDetectRuleMatch(repoDir, rules) {
		return false
	}
	return true
}

func (b *PluginBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	spec := b.plugin.PlanSpec
	if strings.TrimSpace(spec.DockerfileTemplate) == "" {
		return BuildPlan{}, fmt.Errorf("plugin %s has empty dockerfile_template", b.plugin.Name)
	}
	servicePort := spec.ServicePort
	if servicePort == 0 {
		servicePort = 3000
	}
	if req.ServicePort != 0 {
		servicePort = req.ServicePort
	} else if cfg != nil && cfg.ServicePort != 0 {
		servicePort = cfg.ServicePort
	}
	plan := BuildPlan{
		Kind:        firstNonEmpty(spec.Kind, b.plugin.Name),
		ServicePort: servicePort,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), spec.BuildImage),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), spec.RunImage),
		InstallCmd:  firstNonEmpty(req.InstallCmd, spec.InstallCmd),
		BuildCmd:    firstNonEmpty(req.BuildCmd, spec.BuildCmd),
		StartCmd:    firstNonEmpty(req.StartCmd, spec.StartCmd),
	}
	plan.WriteDockerfile = func(repoDir string) error {
		rendered, err := renderPluginDockerfile(b.plugin, plan)
		if err != nil {
			return err
		}
		if spec.WriteDefaultConf {
			return writeStaticDockerArtifacts(repoDir, rendered, spec.WasmMime)
		}
		return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(rendered), 0644)
	}
	plan.Cleanup = func(repoDir string) error {
		for _, rel := range spec.CleanupPaths {
			rel = filepath.Clean(rel)
			if rel == "." || strings.HasPrefix(rel, "..") {
				continue
			}
			_ = os.RemoveAll(filepath.Join(repoDir, rel))
		}
		_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
		if spec.WriteDefaultConf {
			marker := filepath.Join(repoDir, ".relay_default_conf_created")
			if fileExists(marker) {
				_ = os.Remove(filepath.Join(repoDir, "default.conf"))
				_ = os.Remove(marker)
			}
		}
		return nil
	}
	return plan, nil
}

func pluginDetectRuleMatch(repoDir string, rules PluginDetectRules) bool {
	if len(rules.FilesAny) == 0 && len(rules.FilesAll) == 0 &&
		len(rules.DirsAny) == 0 && len(rules.DirsAll) == 0 &&
		len(rules.PackageDepsAny) == 0 && len(rules.PackageDepsAll) == 0 &&
		len(rules.FileExtensionsAny) == 0 && len(rules.FileExtensionsAll) == 0 {
		return false
	}
	if len(rules.FilesAny) > 0 && !anyPathExists(repoDir, false, rules.FilesAny) {
		return false
	}
	if len(rules.FilesAll) > 0 && !allPathExists(repoDir, false, rules.FilesAll) {
		return false
	}
	if len(rules.DirsAny) > 0 && !anyPathExists(repoDir, true, rules.DirsAny) {
		return false
	}
	if len(rules.DirsAll) > 0 && !allPathExists(repoDir, true, rules.DirsAll) {
		return false
	}
	if len(rules.PackageDepsAny) > 0 && !anyPackageDep(repoDir, rules.PackageDepsAny) {
		return false
	}
	if len(rules.PackageDepsAll) > 0 && !allPackageDeps(repoDir, rules.PackageDepsAll) {
		return false
	}
	if len(rules.FileExtensionsAny) > 0 && !anyFileExt(repoDir, rules.FileExtensionsAny) {
		return false
	}
	if len(rules.FileExtensionsAll) > 0 && !allFileExt(repoDir, rules.FileExtensionsAll) {
		return false
	}
	return true
}

func renderPluginDockerfile(plugin *BuildpackPlugin, plan BuildPlan) (string, error) {
	tpl, err := template.New(plugin.Name).Funcs(template.FuncMap{
		"shellJSON":  shellJSON,
		"shellForm":  shellForm,
		"quoteForSh": quoteForSh,
		"shQuote":    shQuote,
	}).Parse(plugin.PlanSpec.DockerfileTemplate)
	if err != nil {
		return "", fmt.Errorf("plugin %s template parse error: %w", plugin.Name, err)
	}
	data := map[string]any{
		"Name":        plugin.Name,
		"Description": plugin.Description,
		"Kind":        plan.Kind,
		"ServicePort": plan.ServicePort,
		"BuildImage":  plan.BuildImage,
		"RunImage":    plan.RunImage,
		"InstallCmd":  plan.InstallCmd,
		"BuildCmd":    plan.BuildCmd,
		"StartCmd":    plan.StartCmd,
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("plugin %s template exec error: %w", plugin.Name, err)
	}
	return buf.String(), nil
}

type NodeNextBuildpack struct{}

type NodeNextStandaloneBuildpack struct{}

func (b *NodeNextStandaloneBuildpack) Name() string { return "next-standalone" }
func (b *NodeNextStandaloneBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "next-standalone") {
		return true
	}
	return isNextStandaloneEnabled(repoDir)
}
func (b *NodeNextStandaloneBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	forced := &RelayConfig{Kind: "next-standalone"}
	if cfg != nil {
		copyCfg := *cfg
		copyCfg.Kind = "next-standalone"
		forced = &copyCfg
	}
	return (&NodeNextBuildpack{}).Plan(req, repoDir, forced)
}

func (b *NodeNextBuildpack) Name() string { return "node-next" }
func (b *NodeNextBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "next-standalone") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "next.config.js")) ||
		fileExists(filepath.Join(repoDir, "next.config.mjs")) ||
		fileExists(filepath.Join(repoDir, "next.config.cjs")) ||
		fileExists(filepath.Join(repoDir, "next.config.ts")) ||
		fileExists(filepath.Join(repoDir, "next.config.mts"))
}
func (b *NodeNextBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim")

	standalone := isNextStandaloneEnabled(repoDir)
	if cfg != nil && strings.EqualFold(cfg.Kind, "next-standalone") {
		standalone = true
	}
	exported := isNextExportEnabled(repoDir)

	port := 3000
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := nodeBuildCmdWithMemoryGuard(firstNonEmpty(req.BuildCmd, nodeDefaultBuildCmd(repoDir)))
	// Unique ID per deployment lane so each app's .next/cache stays isolated.
	laneID := fmt.Sprintf("%s-%s-%s", safe(req.App), safe(string(req.Env)), safe(req.Branch))

	if standalone {
		return BuildPlan{
			Kind:        "next-standalone",
			ServicePort: port,
			BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
			RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
			InstallCmd:  install,
			BuildCmd:    build,
			StartCmd:    `node server.js`,
			// Verification happens inside the built image; host checks are unreliable.
			Verify: nil,
			WriteDockerfile: func(repoDir string) error {
				// Multi-stage: install with a cached package store, build with a persistent .next cache,
				// then copy the standalone runtime output into the final image.
				df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
RUN mkdir -p /app/public/uploads && chown -R node:node /app || true
USER node
EXPOSE 3000
CMD ["node","server.js"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, laneID, build, "/app/.next/cache"), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg))

				writeRelayDockerignore(repoDir)
				return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
			},
			Cleanup: func(repoDir string) error {
				cleanupRelayDockerignore(repoDir)
				_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
				return nil
			},
		}, nil
	}

	if exported {
		runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
		return BuildPlan{
			Kind:        "next-export",
			ServicePort: 80,
			BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
			RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
			InstallCmd:  install,
			BuildCmd:    build,
			StartCmd:    "",
			Verify:      nil,
			WriteDockerfile: func(repoDir string) error {
				df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s

FROM %s
COPY --from=builder /app/out /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, laneID, build, "/app/.next/cache"), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg))

				writeRelayDockerignore(repoDir)
				if err := os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644); err != nil {
					return err
				}
				defPath := filepath.Join(repoDir, "default.conf")
				marker := filepath.Join(repoDir, ".relay_default_conf_created")
				if !fileExists(defPath) {
					defaultConf := `server {
	listen 80;
	server_name _;
	root /usr/share/nginx/html;
	index index.html;
	location / {
		try_files $uri $uri/ /index.html;
	}
}
`
					if err := os.WriteFile(defPath, []byte(defaultConf), 0644); err != nil {
						return err
					}
					_ = os.WriteFile(marker, []byte("1"), 0644)
				}
				return nil
			},
			Cleanup: func(repoDir string) error {
				cleanupRelayDockerignore(repoDir)
				_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
				marker := filepath.Join(repoDir, ".relay_default_conf_created")
				if fileExists(marker) {
					_ = os.Remove(filepath.Join(repoDir, "default.conf"))
					_ = os.Remove(marker)
				}
				return nil
			},
		}, nil
	}

	// Classic Next.js (no standalone)
	start := firstNonEmpty(req.StartCmd, nextClassicStartCmd(repoDir))
	return BuildPlan{
		Kind:        "next-classic",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			// Keep dev deps in the build stage, derive production deps from that cached install,
			// and reuse a persistent .next cache between builds.
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s
RUN rm -rf node_modules

%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
EXPOSE 3000
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, laneID, build, "/app/.next/cache"), nodeProdDepsStage(repoDir), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), shellJSON(start))

			writeRelayDockerignore(repoDir)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			cleanupRelayDockerignore(repoDir)
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type NodeViteBuildpack struct{}

func (b *NodeViteBuildpack) Name() string { return "node-vite" }
func (b *NodeViteBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "vite-static") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "vite.config.ts")) ||
		fileExists(filepath.Join(repoDir, "vite.config.js")) ||
		fileExists(filepath.Join(repoDir, "vite.config.mjs"))
}
func (b *NodeViteBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := nodeBuildCmdWithMemoryGuard(firstNonEmpty(req.BuildCmd, nodeDefaultBuildCmd(repoDir)))

	return BuildPlan{
		Kind:        "vite-static",
		ServicePort: 80,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    "",
		// Build runs inside Docker; host filesystem checks are unreliable.
		Verify: nil,
		WriteDockerfile: func(repoDir string) error {
			// Multi-stage: build with node using a cached package store, then serve with nginx.
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s

FROM %s
COPY --from=builder /app/dist /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, "", build), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg))

			writeRelayDockerignore(repoDir)
			if err := os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644); err != nil {
				return err
			}
			// Only write default.conf if the repo doesn't provide one already
			defPath := filepath.Join(repoDir, "default.conf")
			marker := filepath.Join(repoDir, ".relay_default_conf_created")
			if !fileExists(defPath) {
				defaultConf := `server {
			listen 80;
			server_name _;
			root /usr/share/nginx/html;
			index index.html;
			location / {
				try_files $uri $uri/ /index.html;
			}
			}
`
				if err := os.WriteFile(defPath, []byte(defaultConf), 0644); err != nil {
					return err
				}
				_ = os.WriteFile(marker, []byte("1"), 0644)
				return nil
			}
			return nil
		},
		Cleanup: func(repoDir string) error {
			cleanupRelayDockerignore(repoDir)
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.RemoveAll(filepath.Join(repoDir, "dist"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			marker := filepath.Join(repoDir, ".relay_default_conf_created")
			if fileExists(marker) {
				_ = os.Remove(filepath.Join(repoDir, "default.conf"))
				_ = os.Remove(marker)
			}
			return nil
		},
	}, nil
}

// nodeServerBuildpackPlan is the shared Plan() body for Node meta-frameworks
// that build to a self-hosted server entrypoint (SvelteKit adapter-node,
// Remix/React Router, Nuxt's Nitro server) rather than a static SPA. It
// mirrors NodeGenericBuildpack's multi-stage layout (cached install, build,
// slim prod-deps runtime image) so these frameworks get the same build-cache
// speed benefits instead of a bespoke, uncached Dockerfile per framework.
func nodeServerBuildpackPlan(kind string, req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := nodeBuildCmdWithMemoryGuard(firstNonEmpty(req.BuildCmd, nodeDefaultBuildCmd(repoDir)))
	start := firstNonEmpty(req.StartCmd, nodeDefaultStartCmd(repoDir))
	port := firstNonZero(req.ServicePort, 3000)

	return BuildPlan{
		Kind:        kind,
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s
RUN rm -rf node_modules

%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=%d
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, "", build), nodeProdDepsStage(repoDir), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, shellJSON(start))

			writeRelayDockerignore(repoDir)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			cleanupRelayDockerignore(repoDir)
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type SvelteKitBuildpack struct{}

func (b *SvelteKitBuildpack) Name() string { return "sveltekit" }

// Detect must win over NodeViteBuildpack: SvelteKit apps also ship a
// vite.config.ts (the SvelteKit Vite plugin needs it), so if this ran after
// NodeViteBuildpack in priority order every SvelteKit app would be
// misdetected as a static Vite SPA and nginx-served from a "dist/" directory
// that SvelteKit never produces — a real, silent deploy failure this was
// hitting before this buildpack existed.
func (b *SvelteKitBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "sveltekit") {
		return true
	}
	if !hasPackageDependency(repoDir, "@sveltejs/kit") {
		return false
	}
	return fileExists(filepath.Join(repoDir, "svelte.config.js")) ||
		fileExists(filepath.Join(repoDir, "svelte.config.ts")) ||
		fileExists(filepath.Join(repoDir, "svelte.config.mjs"))
}
func (b *SvelteKitBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	// Targets adapter-node (or adapter-auto falling back to node in a
	// generic Docker/Linux environment, which is the common self-hosted
	// case): build/index.js, started via the "start" script SvelteKit's
	// adapter-node scaffolds — nodeDefaultStartCmd already resolves that.
	// A project explicitly using @sveltejs/adapter-static should override
	// with "kind": "vite-static" in relay.config.json to get the nginx
	// static-file path instead.
	return nodeServerBuildpackPlan("sveltekit", req, repoDir, cfg)
}

type RemixBuildpack struct{}

func (b *RemixBuildpack) Name() string { return "remix" }

// Same rationale as SvelteKitBuildpack: Remix's Vite plugin also requires a
// vite.config.ts, so this must be detected (and listed) before
// NodeViteBuildpack or Remix apps get misdetected as a static SPA.
func (b *RemixBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "remix") || strings.EqualFold(cfg.Kind, "react-router")) {
		return true
	}
	return anyPackageDep(repoDir, []string{"@remix-run/dev", "@remix-run/serve", "@remix-run/react", "@react-router/dev", "@react-router/serve"}) ||
		fileExists(filepath.Join(repoDir, "remix.config.js"))
}
func (b *RemixBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	return nodeServerBuildpackPlan("remix", req, repoDir, cfg)
}

type NuxtBuildpack struct{}

func (b *NuxtBuildpack) Name() string { return "nuxt" }
func (b *NuxtBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "nuxt") {
		return true
	}
	if !hasPackageDependency(repoDir, "nuxt") {
		return false
	}
	return fileExists(filepath.Join(repoDir, "nuxt.config.js")) ||
		fileExists(filepath.Join(repoDir, "nuxt.config.ts")) ||
		fileExists(filepath.Join(repoDir, "nuxt.config.mjs"))
}
func (b *NuxtBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	// Nuxt's own "start" script runs `node .output/server/index.mjs` (the
	// Nitro server), which nodeDefaultStartCmd already resolves via the
	// package.json "start" script — no framework-specific override needed.
	return nodeServerBuildpackPlan("nuxt", req, repoDir, cfg)
}

type PHPBuildpack struct{}

func (b *PHPBuildpack) Name() string { return "php" }
func (b *PHPBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "php") || strings.EqualFold(cfg.Kind, "laravel")) {
		return true
	}
	return fileExists(filepath.Join(repoDir, "composer.json"))
}

// phpIsLaravel reports whether composer.json declares laravel/framework —
// Laravel's convention is to serve from public/ (index.php lives there,
// everything else is deliberately outside the webroot); a bare PHP app
// without that convention is served from the repo root instead.
func phpIsLaravel(repoDir string) bool {
	b, err := os.ReadFile(filepath.Join(repoDir, "composer.json"))
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), `"laravel/framework"`)
}

func (b *PHPBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_PHP_BUILD_IMAGE", "composer:2")
	runImg := getenv("RELAY_PHP_IMAGE", "php:8.3-cli")
	port := firstNonZero(req.ServicePort, 8000)

	docroot := "."
	if phpIsLaravel(repoDir) {
		docroot = "public"
	}

	install := firstNonEmpty(req.InstallCmd, `sh -lc "composer install --no-dev --optimize-autoloader --no-interaction"`)
	// Laravel's artisan cache commands (config:cache, route:cache) need
	// runtime env (DATABASE_URL, APP_KEY, etc.) that Relay only injects when
	// the container starts, not at `docker build` time — running them here
	// would hit the exact "env var required during build" failure class this
	// project has already hit once with a Next.js app. Skipped deliberately.
	start := firstNonEmpty(req.StartCmd, fmt.Sprintf(`sh -lc "php -S 0.0.0.0:${PORT:-%d} -t %s"`, port, docroot))

	return BuildPlan{
		Kind:        "php",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS deps
WORKDIR /app
COPY composer.json composer.lock* ./
RUN %s

FROM %s
WORKDIR /app
ENV PORT=%d
COPY --from=deps /app/vendor ./vendor
COPY . .
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), install, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, shellForm(start))
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.RemoveAll(filepath.Join(repoDir, "vendor"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type ExpoWebBuildpack struct{}

func (b *ExpoWebBuildpack) Name() string { return "expo-web" }
func (b *ExpoWebBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "expo-web") {
		return true
	}
	if !fileExists(filepath.Join(repoDir, "package.json")) {
		return false
	}
	if !hasPackageDependency(repoDir, "expo") {
		return false
	}
	return fileExists(filepath.Join(repoDir, "app.json")) ||
		fileExists(filepath.Join(repoDir, "app.config.js")) ||
		fileExists(filepath.Join(repoDir, "app.config.ts"))
}
func (b *ExpoWebBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, `sh -lc "npx expo export --platform web --output-dir dist || npx expo export -p web --output-dir dist || npx expo export --platform web"`)

	return BuildPlan{
		Kind:        "expo-web",
		ServicePort: 80,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s

FROM %s
COPY --from=builder /app/dist /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, "", build), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg))

			return writeStaticDockerArtifacts(repoDir, df, false)
		},
		Cleanup: func(repoDir string) error {
			return cleanupStaticDockerArtifacts(repoDir, true)
		},
	}, nil
}

type BunBuildpack struct{}

type NodeGenericBuildpack struct{}

type SprintUIBuildpack struct{}

func (b *SprintUIBuildpack) Name() string { return "sprint-ui" }
func (b *SprintUIBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "sprint-ui") {
		return true
	}
	if !fileExists(filepath.Join(repoDir, "package.json")) {
		return false
	}
	if !fileExists(filepath.Join(repoDir, "config.sui")) {
		return false
	}
	return fileExists(filepath.Join(repoDir, ".sprint", "build.js")) ||
		fileExists(filepath.Join(repoDir, ".sprint", "server.js"))
}
func (b *SprintUIBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, `npm run build`)
	start := firstNonEmpty(req.StartCmd, `if [ -f build/server.mjs ]; then exec node build/server.mjs; else exec serve -s build -l ${PORT:-3000}; fi`)
	port := firstNonZero(req.ServicePort, 3000)

	return BuildPlan{
		Kind:        "sprint-ui",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
WORKDIR /app
COPY . .
%s
RUN rm -rf node_modules

%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=%d
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
RUN npm install -g serve@14
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, "", build), nodeProdDepsStage(repoDir), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, shellJSON(start))

			writeRelayDockerignore(repoDir)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			cleanupRelayDockerignore(repoDir)
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.RemoveAll(filepath.Join(repoDir, "build"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

func (b *BunBuildpack) Name() string { return "bun" }
func (b *BunBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "bun") || strings.EqualFold(cfg.Kind, "bun-generic")) {
		return true
	}
	if !fileExists(filepath.Join(repoDir, "package.json")) {
		return false
	}
	return bunProject(repoDir)
}
func (b *BunBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_BUN_IMAGE", "oven/bun:1")
	runImg := getenv("RELAY_BUN_RUN_IMAGE", "oven/bun:1-slim")
	install := firstNonEmpty(req.InstallCmd, bunInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, bunDefaultBuildCmd(repoDir))
	start := firstNonEmpty(req.StartCmd, bunDefaultStartCmd(repoDir))
	port := firstNonZero(req.ServicePort, 3000)

	return BuildPlan{
		Kind:        "bun",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			lockCopy := bunLockfileCopyInstruction(repoDir)
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
COPY package.json ./
%s%s

FROM deps AS builder
COPY . .
%s
RUN rm -rf node_modules

FROM %s AS prod-deps
WORKDIR /app
COPY package.json ./
%s%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=%d
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), lockCopy, bunRunStepWithCache(install), bunRunStepWithCache(build), firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), lockCopy, bunRunStepWithCache(bunProdInstallCmd(repoDir)), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, shellJSON(start))

			writeRelayDockerignore(repoDir)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			cleanupRelayDockerignore(repoDir)
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

func (b *NodeGenericBuildpack) Name() string { return "node-generic" }
func (b *NodeGenericBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "node") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "package.json"))
}
func (b *NodeGenericBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_NODE_IMAGE", "node:22")
	runImg := getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim")
	install := firstNonEmpty(req.InstallCmd, nodeInstallCmd(repoDir))
	build := nodeBuildCmdWithMemoryGuard(firstNonEmpty(req.BuildCmd, nodeDefaultBuildCmd(repoDir)))
	start := firstNonEmpty(req.StartCmd, nodeDefaultStartCmd(repoDir))
	port := firstNonZero(req.ServicePort, 3000)

	return BuildPlan{
		Kind:        "node",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			// Keep dev deps in the builder and derive a production-only dependency set for runtime.
			df := fmt.Sprintf(`# syntax=docker/dockerfile:1.7
FROM %s AS deps
WORKDIR /app
ENV CI=true
COPY package.json ./
COPY package-lock.json* pnpm-lock.yaml* yarn.lock* ./
%s

FROM deps AS builder
COPY . .
%s
RUN rm -rf node_modules

%s

FROM %s
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=%d
COPY --from=builder /app /app
COPY --from=prod-deps /app/node_modules ./node_modules
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), nodeRunStepWithCaches(repoDir, "", install), nodeRunStepWithCaches(repoDir, "", build), nodeProdDepsStage(repoDir), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, shellJSON(start))

			writeRelayDockerignore(repoDir)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			cleanupRelayDockerignore(repoDir)
			_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type GoBuildpack struct{}

func (b *GoBuildpack) Name() string { return "go" }
func (b *GoBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "go") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "go.mod"))
}
func (b *GoBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_GO_IMAGE", "golang:1.22")
	runImg := getenv("RELAY_GO_RUN_IMAGE", "gcr.io/distroless/base-debian12")

	port := firstNonZero(req.ServicePort, 8080)
	install := "" // go mod download included in build
	build := firstNonEmpty(req.BuildCmd, fmt.Sprintf(`sh -lc "go mod download && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags=\"-s -w\" -o /out/app ./"`))
	start := firstNonEmpty(req.StartCmd, `/app/app`)

	return BuildPlan{
		Kind:        "go",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
		WORKDIR /src
		COPY go.mod go.sum ./
		RUN go mod download
		COPY . .
		RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/app ./

		FROM %s
		WORKDIR /app
		COPY --from=builder /out/app /app/app
		ENV PORT=%d
		EXPOSE %d
		USER 65532:65532
		ENTRYPOINT ["/app/app"]
		`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		// Verify is not performed on host; docker build will fail if build step fails.
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type DotnetBuildpack struct{}

func (b *DotnetBuildpack) Name() string { return "dotnet" }
func (b *DotnetBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "dotnet") {
		return true
	}
	// Detect common .NET solution/project files
	m, _ := filepath.Glob(filepath.Join(repoDir, "*.sln"))
	if len(m) > 0 {
		return true
	}
	m, _ = filepath.Glob(filepath.Join(repoDir, "*.csproj"))
	return len(m) > 0
}
func (b *DotnetBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_DOTNET_SDK_IMAGE", "mcr.microsoft.com/dotnet/sdk:8.0")
	runImg := getenv("RELAY_DOTNET_ASPNET_IMAGE", "mcr.microsoft.com/dotnet/aspnet:8.0")

	port := firstNonZero(req.ServicePort, 8080)
	install := "" // restore in build
	project := dotnetPickEntry(repoDir)
	if project == "" {
		return BuildPlan{}, fmt.Errorf("dotnet: no .sln or .csproj found")
	}

	build := firstNonEmpty(req.BuildCmd, fmt.Sprintf(`sh -lc "dotnet restore %s && dotnet publish %s -c Release -o /out"`, shQuote(project), shQuote(project)))
	start := firstNonEmpty(req.StartCmd, `dotnet app.dll`)

	return BuildPlan{
		Kind:        "dotnet",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY . .
RUN dotnet restore %s
RUN dotnet publish %s -c Release -o /out

FROM %s
WORKDIR /app
COPY --from=builder /out ./
ENV ASPNETCORE_URLS=http://0.0.0.0:%d
EXPOSE %d
CMD ["sh","-lc","dll=$(ls *.dll | head -n1); echo Running $dll; dotnet $dll"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), shQuote(project), shQuote(project), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		// No host-side Verify; docker build ensures publish succeeded.
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type PythonBuildpack struct{}

func (b *PythonBuildpack) Name() string { return "python" }
func (b *PythonBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "python") || strings.EqualFold(cfg.Kind, "django") ||
		strings.EqualFold(cfg.Kind, "fastapi") || strings.EqualFold(cfg.Kind, "flask")) {
		return true
	}
	return fileExists(filepath.Join(repoDir, "requirements.txt")) ||
		fileExists(filepath.Join(repoDir, "pyproject.toml")) ||
		fileExists(filepath.Join(repoDir, "Pipfile")) ||
		fileExists(filepath.Join(repoDir, "manage.py"))
}
func (b *PythonBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_PY_IMAGE", "python:3.12")
	runImg := getenv("RELAY_PY_RUN_IMAGE", "python:3.12-slim")

	port := firstNonZero(req.ServicePort, 8000)

	install := firstNonEmpty(req.InstallCmd, pythonInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, "") // optional
	start := firstNonEmpty(req.StartCmd, pythonDefaultStart(repoDir))

	// Kind reflects the actually-detected framework (django/fastapi/flask),
	// not just "python" — visible in deploy records/UI so a wrong framework
	// guess (which changes the start command) is easy to spot and override
	// via relay.config.json's "kind" instead of silently shipping a broken
	// CMD.
	kind := pythonFramework(repoDir)
	if kind == "generic" {
		kind = "python"
	}

	return BuildPlan{
		Kind:        kind,
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			install := firstNonEmpty(req.InstallCmd, pythonInstallCmd(repoDir))
			if strings.TrimSpace(install) == "" {
				install = `sh -lc "pip install --no-cache-dir -U pip"`
			}

			df := fmt.Sprintf(`FROM %s
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV PORT=%d
COPY . .
RUN %s
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, install, port, shellForm(start))
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type RubyBuildpack struct{}

func (b *RubyBuildpack) Name() string { return "ruby" }
func (b *RubyBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "ruby") || strings.EqualFold(cfg.Kind, "rails") || strings.EqualFold(cfg.Kind, "sinatra")) {
		return true
	}
	return fileExists(filepath.Join(repoDir, "Gemfile"))
}
func (b *RubyBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_RUBY_IMAGE", "ruby:3.3")
	runImg := getenv("RELAY_RUBY_RUN_IMAGE", "ruby:3.3-slim")

	port := firstNonZero(req.ServicePort, 3000)
	install := firstNonEmpty(req.InstallCmd, rubyInstallCmd(repoDir))
	build := firstNonEmpty(req.BuildCmd, rubyDefaultBuildCmd(repoDir))
	start := firstNonEmpty(req.StartCmd, rubyDefaultStart(repoDir))

	// Kind reflects the detected framework so it's visible in the deploy
	// record instead of a flat "ruby" for both a Rails app and a bare Rack
	// script.
	kind := "ruby"
	if rubyRailsApp(repoDir) {
		kind = "rails"
	}

	return BuildPlan{
		Kind:        kind,
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s
WORKDIR /app
ENV PORT=%d
ENV RACK_ENV=production
ENV RAILS_ENV=production
ENV BUNDLE_WITHOUT=development:test
RUN apt-get update && apt-get install -y --no-install-recommends build-essential git pkg-config libsqlite3-dev && rm -rf /var/lib/apt/lists/*
COPY Gemfile Gemfile.lock* ./
RUN %s
COPY . .
%s
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, install, rubyRunStep(build), port, shellForm(start))
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type JavaBuildpack struct{}

func (b *JavaBuildpack) Name() string { return "java" }
func (b *JavaBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "java") || strings.EqualFold(cfg.Kind, "jvm") || strings.EqualFold(cfg.Kind, "spring-boot")) {
		return true
	}
	return fileExists(filepath.Join(repoDir, "pom.xml")) || fileExists(filepath.Join(repoDir, "build.gradle")) || fileExists(filepath.Join(repoDir, "build.gradle.kts"))
}
func (b *JavaBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_JAVA_BUILD_IMAGE", "maven:3.9-eclipse-temurin-21")
	runImg := getenv("RELAY_JAVA_RUN_IMAGE", "eclipse-temurin:21-jre")

	port := firstNonZero(req.ServicePort, 8080)
	install := "" // maven/gradle handles
	build := firstNonEmpty(req.BuildCmd, javaDefaultBuild(repoDir))
	start := firstNonEmpty(req.StartCmd, `java -jar app.jar`)

	return BuildPlan{
		Kind:        "java",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			isMaven := fileExists(filepath.Join(repoDir, "pom.xml"))
			isGradle := fileExists(filepath.Join(repoDir, "build.gradle")) || fileExists(filepath.Join(repoDir, "build.gradle.kts"))
			if isMaven {
				df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY pom.xml mvnw* .
COPY .mvn .mvn
COPY src ./src
RUN mvn -q -DskipTests package

FROM %s
WORKDIR /app
COPY --from=builder /src/target/*.jar /app/app.jar
ENV PORT=%d
ENV SERVER_PORT=%d
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, port, shellForm(start))
				return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
			} else if isGradle {
				gradleCmd := "gradle build -x test"
				if fileExists(filepath.Join(repoDir, "gradlew")) {
					gradleCmd = "./gradlew build -x test"
				}
				df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY . .
RUN %s

FROM %s
WORKDIR /app
COPY --from=builder /src/build/libs/*.jar /app/app.jar
ENV PORT=%d
ENV SERVER_PORT=%d
EXPOSE %d
CMD %s
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), gradleCmd, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port, port, shellForm(start))
				return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
			} else {
				return fmt.Errorf("java: no pom.xml or build.gradle found")
			}
		},
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type RustBuildpack struct{}

func (b *RustBuildpack) Name() string { return "rust" }
func (b *RustBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "rust") {
		return true
	}
	return fileExists(filepath.Join(repoDir, "Cargo.toml"))
}
func (b *RustBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_RUST_IMAGE", "rust:1.77")
	runImg := getenv("RELAY_RUST_RUN_IMAGE", "debian:bookworm-slim")

	port := firstNonZero(req.ServicePort, 8080)
	// Let the Dockerfile perform the cargo build and extraction; avoid noisy default copy here.
	build := firstNonEmpty(req.BuildCmd, "")
	start := firstNonEmpty(req.StartCmd, `/app/app`)

	return BuildPlan{
		Kind:        "rust",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
COPY Cargo.toml Cargo.lock ./
RUN cargo fetch
COPY . .
RUN cargo build --release && set -eu; \
  bin=$(find target/release -maxdepth 1 -type f -executable \
    ! -name "*.d" ! -name "*.rlib" ! -name "*.so" ! -name "*.a" | head -n1); \
  test -n "$bin"; mkdir -p /out; cp "$bin" /out/app

FROM %s
WORKDIR /app
COPY --from=builder /out/app /app/app
ENV PORT=%d
EXPOSE %d
USER 65532:65532
ENTRYPOINT ["/app/app"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)

			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Verify: nil,
		Cleanup: func(repoDir string) error {
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type CCppBuildpack struct{}

func (b *CCppBuildpack) Name() string { return "c-cpp" }
func (b *CCppBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && (strings.EqualFold(cfg.Kind, "c") || strings.EqualFold(cfg.Kind, "cpp") || strings.EqualFold(cfg.Kind, "c-cpp")) {
		return true
	}
	if fileExists(filepath.Join(repoDir, "CMakeLists.txt")) || fileExists(filepath.Join(repoDir, "Makefile")) {
		return true
	}
	return len(findFilesByExt(repoDir, ".c", ".cc", ".cpp", ".cxx")) > 0
}
func (b *CCppBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	buildImg := getenv("RELAY_CC_IMAGE", "debian:bookworm")
	runImg := getenv("RELAY_CC_RUN_IMAGE", "debian:bookworm-slim")
	port := firstNonZero(req.ServicePort, 8080)
	install := firstNonEmpty(req.InstallCmd, `sh -lc "apt-get update && apt-get install -y --no-install-recommends build-essential cmake pkg-config && rm -rf /var/lib/apt/lists/*"`)
	build := firstNonEmpty(req.BuildCmd, cCppDefaultBuildCmd(repoDir))
	start := firstNonEmpty(req.StartCmd, `/app/app`)

	return BuildPlan{
		Kind:        "c-cpp",
		ServicePort: port,
		BuildImage:  firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg),
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		InstallCmd:  install,
		BuildCmd:    build,
		StartCmd:    start,
		WriteDockerfile: func(repoDir string) error {
			df := fmt.Sprintf(`FROM %s AS builder
WORKDIR /src
RUN %s
COPY . .
RUN %s

FROM %s
WORKDIR /app
COPY --from=builder /out/app /app/app
ENV PORT=%d
EXPOSE %d
CMD ["/app/app"]
`, firstNonEmpty(cfgStr(cfg, "BuildImage"), buildImg), install, build, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), port, port)
			return os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(df), 0644)
		},
		Cleanup: func(repoDir string) error {
			_ = os.RemoveAll(filepath.Join(repoDir, "build"))
			_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
			return nil
		},
	}, nil
}

type WasmStaticBuildpack struct{}

func (b *WasmStaticBuildpack) Name() string { return "wasm-static" }
func (b *WasmStaticBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "wasm-static") {
		return true
	}
	if !hasWasmAssets(repoDir) {
		return false
	}
	return fileExists(filepath.Join(repoDir, "index.html")) || fileExists(filepath.Join(repoDir, "public", "index.html"))
}
func (b *WasmStaticBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	return BuildPlan{
		Kind:        "wasm-static",
		ServicePort: 80,
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		WriteDockerfile: func(repoDir string) error {
			root := "."
			if fileExists(filepath.Join(repoDir, "public", "index.html")) {
				root = "public"
			}
			df := fmt.Sprintf(`FROM %s
COPY %s /usr/share/nginx/html
COPY default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), root)
			return writeStaticDockerArtifacts(repoDir, df, true)
		},
		Cleanup: func(repoDir string) error {
			return cleanupStaticDockerArtifacts(repoDir, false)
		},
	}, nil
}

type StaticBuildpack struct{}

func (b *StaticBuildpack) Name() string { return "static" }
func (b *StaticBuildpack) Detect(repoDir string, cfg *RelayConfig) bool {
	if cfg != nil && strings.EqualFold(cfg.Kind, "static") {
		return true
	}
	// If there's an index.html at root and no package.json/go.mod/etc, treat as static
	if fileExists(filepath.Join(repoDir, "index.html")) && !fileExists(filepath.Join(repoDir, "package.json")) && !fileExists(filepath.Join(repoDir, "go.mod")) {
		return true
	}
	// common "public" static folder
	if fileExists(filepath.Join(repoDir, "public", "index.html")) && !fileExists(filepath.Join(repoDir, "package.json")) && !fileExists(filepath.Join(repoDir, "go.mod")) {
		return true
	}
	return false
}
func (b *StaticBuildpack) Plan(req DeployRequest, repoDir string, cfg *RelayConfig) (BuildPlan, error) {
	runImg := getenv("RELAY_NGINX_IMAGE", "nginx:alpine")
	port := 80
	return BuildPlan{
		Kind:        "static",
		ServicePort: port,
		BuildImage:  "",
		RunImage:    firstNonEmpty(cfgStr(cfg, "RunImage"), runImg),
		WriteDockerfile: func(repoDir string) error {
			// serve ./public if present else serve repo root
			root := "."
			if fileExists(filepath.Join(repoDir, "public", "index.html")) {
				root = "public"
			}
			df := fmt.Sprintf(`FROM %s
COPY %s /usr/share/nginx/html
EXPOSE 80
`, firstNonEmpty(cfgStr(cfg, "RunImage"), runImg), root)
			return writeStaticDockerArtifacts(repoDir, df, false)
		},
		Cleanup: func(repoDir string) error {
			return cleanupStaticDockerArtifacts(repoDir, false)
		},
	}, nil
}
