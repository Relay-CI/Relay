package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultBuildpacksPreferBunOverNodeGeneric(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"packageManager":"bun@1.2.0",
		"scripts":{"start":"bun run server.ts"}
	}`)
	mustWriteTestFile(t, filepath.Join(repoDir, "bun.lock"), "")

	var selected Buildpack
	for _, bp := range defaultBuildpacks() {
		if bp.Detect(repoDir, nil) {
			selected = bp
			break
		}
	}
	if selected == nil {
		t.Fatalf("expected a buildpack match")
	}
	if selected.Name() != "bun" {
		t.Fatalf("expected bun buildpack first, got %q", selected.Name())
	}
}

func TestBunBuildpackUsesBunCommands(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"packageManager":"bun@1.2.0",
		"scripts":{"build":"bun build src/index.ts --outdir dist","start":"bun run server.ts"}
	}`)
	mustWriteTestFile(t, filepath.Join(repoDir, "bun.lock"), "")

	plan, err := (&BunBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.InstallCmd != "bun install --frozen-lockfile" {
		t.Fatalf("unexpected bun install command: %q", plan.InstallCmd)
	}
	if plan.BuildCmd != "bun run build" {
		t.Fatalf("unexpected bun build command: %q", plan.BuildCmd)
	}
	if plan.StartCmd != "bun run start" {
		t.Fatalf("unexpected bun start command: %q", plan.StartCmd)
	}
}

func TestNodeGenericBuildpackFallsBackToServerJSWhenStartScriptMissing(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{"name":"demo"}`)
	mustWriteTestFile(t, filepath.Join(repoDir, "server.js"), `console.log("hello")`)

	plan, err := (&NodeGenericBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.StartCmd != "node server.js" {
		t.Fatalf("unexpected node generic fallback start command: %q", plan.StartCmd)
	}
}

func TestNodeGenericBuildpackPrefersPackageManagerStartScriptWhenPresent(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"scripts":{"start":"node server.js"}
	}`)
	mustWriteTestFile(t, filepath.Join(repoDir, "server.js"), `console.log("hello")`)

	plan, err := (&NodeGenericBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.StartCmd != "npm start" {
		t.Fatalf("unexpected node generic scripted start command: %q", plan.StartCmd)
	}
}

func TestPythonBuildpackDjangoUsesUvicornWhenAvailable(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "manage.py"), `import os
os.environ.setdefault("DJANGO_SETTINGS_MODULE", "demo.settings")
`)
	mustWriteTestFile(t, filepath.Join(repoDir, "demo", "asgi.py"), `application = None`)
	mustWriteTestFile(t, filepath.Join(repoDir, "requirements.txt"), "Django==5.2\nuvicorn==0.35.0\n")

	plan, err := (&PythonBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	want := `sh -lc "python -m uvicorn demo.asgi:application --host 0.0.0.0 --port ${PORT:-8000}"`
	if plan.StartCmd != want {
		t.Fatalf("unexpected django uvicorn start command: got %q want %q", plan.StartCmd, want)
	}
}

func TestPythonBuildpackDjangoFallsBackToRunserver(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "manage.py"), `import os
os.environ.setdefault("DJANGO_SETTINGS_MODULE", "demo.settings")
`)
	mustWriteTestFile(t, filepath.Join(repoDir, "demo", "wsgi.py"), `application = None`)
	mustWriteTestFile(t, filepath.Join(repoDir, "requirements.txt"), "Django==5.2\n")

	plan, err := (&PythonBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	want := `sh -lc "python manage.py runserver 0.0.0.0:${PORT:-8000}"`
	if plan.StartCmd != want {
		t.Fatalf("unexpected django fallback start command: got %q want %q", plan.StartCmd, want)
	}
}

func TestJavaBuildpackDockerfileExportsServerPort(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "pom.xml"), `<project></project>`)
	mustWriteTestFile(t, filepath.Join(repoDir, "src", "main", "java", "com", "example", "App.java"), `class App {}`)

	plan, err := (&JavaBuildpack{}).Plan(DeployRequest{ServicePort: 9090}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := plan.WriteDockerfile(repoDir); err != nil {
		t.Fatalf("write dockerfile: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(repoDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "ENV SERVER_PORT=9090") {
		t.Fatalf("expected SERVER_PORT env in dockerfile, got:\n%s", content)
	}
}

// SvelteKit and Remix apps both ship a vite.config.ts (their Vite plugin
// requires it), so without a higher-priority, framework-aware buildpack they
// used to be misdetected as a static Vite SPA by NodeViteBuildpack and
// nginx-served from a "dist/" directory neither framework ever produces —
// a real, silent deploy failure. These tests pin the fix.
func TestDefaultBuildpacksPreferSvelteKitOverNodeVite(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"dependencies":{"@sveltejs/kit":"^2.0.0"},
		"scripts":{"build":"vite build","start":"node build/index.js"}
	}`)
	mustWriteTestFile(t, filepath.Join(repoDir, "svelte.config.js"), `export default {};`)
	mustWriteTestFile(t, filepath.Join(repoDir, "vite.config.ts"), `export default {};`)

	var selected Buildpack
	for _, bp := range defaultBuildpacks() {
		if bp.Detect(repoDir, nil) {
			selected = bp
			break
		}
	}
	if selected == nil {
		t.Fatalf("expected a buildpack match")
	}
	if selected.Name() != "sveltekit" {
		t.Fatalf("expected sveltekit buildpack to win over node-vite, got %q", selected.Name())
	}
}

func TestDefaultBuildpacksPreferRemixOverNodeVite(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"devDependencies":{"@remix-run/dev":"^2.0.0"},
		"dependencies":{"@remix-run/serve":"^2.0.0"},
		"scripts":{"build":"remix vite:build","start":"remix-serve build/server/index.js"}
	}`)
	mustWriteTestFile(t, filepath.Join(repoDir, "vite.config.ts"), `export default {};`)

	var selected Buildpack
	for _, bp := range defaultBuildpacks() {
		if bp.Detect(repoDir, nil) {
			selected = bp
			break
		}
	}
	if selected == nil {
		t.Fatalf("expected a buildpack match")
	}
	if selected.Name() != "remix" {
		t.Fatalf("expected remix buildpack to win over node-vite, got %q", selected.Name())
	}
}

func TestNuxtBuildpackDetected(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"dependencies":{"nuxt":"^3.0.0"},
		"scripts":{"build":"nuxt build","start":"node .output/server/index.mjs"}
	}`)
	mustWriteTestFile(t, filepath.Join(repoDir, "nuxt.config.ts"), `export default defineNuxtConfig({});`)

	var selected Buildpack
	for _, bp := range defaultBuildpacks() {
		if bp.Detect(repoDir, nil) {
			selected = bp
			break
		}
	}
	if selected == nil {
		t.Fatalf("expected a buildpack match")
	}
	if selected.Name() != "nuxt" {
		t.Fatalf("expected nuxt buildpack, got %q", selected.Name())
	}
}

func TestPHPBuildpackLaravelUsesPublicDocroot(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "composer.json"), `{
		"require": {"laravel/framework": "^11.0"}
	}`)

	plan, err := (&PHPBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !strings.Contains(plan.StartCmd, "-t public") {
		t.Fatalf("expected laravel start command to serve from public/, got %q", plan.StartCmd)
	}
}

func TestPHPBuildpackGenericUsesRepoRootDocroot(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "composer.json"), `{
		"require": {"slim/slim": "^4.0"}
	}`)

	plan, err := (&PHPBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !strings.Contains(plan.StartCmd, "-t .") {
		t.Fatalf("expected generic php start command to serve from repo root, got %q", plan.StartCmd)
	}
}

func TestMonorepoSubdirHintFindsNestedApp(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "apps", "web", "package.json"), `{"name":"web"}`)

	hint := monorepoSubdirHint(repoDir, defaultBuildpacks(), nil)
	if !strings.Contains(hint, "apps/web") {
		t.Fatalf("expected hint to mention apps/web, got %q", hint)
	}
	if !strings.Contains(hint, `"project_root"`) {
		t.Fatalf("expected hint to suggest project_root override, got %q", hint)
	}
}

func TestMonorepoSubdirHintEmptyWhenNothingMatches(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "README.md"), "nothing deployable here")

	hint := monorepoSubdirHint(repoDir, defaultBuildpacks(), nil)
	if hint != "" {
		t.Fatalf("expected empty hint, got %q", hint)
	}
}

func TestRubyBuildpackDetectsRailsStartCommand(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "Gemfile"), `source "https://rubygems.org"
gem "rails", "~> 8.0"
`)
	mustWriteTestFile(t, filepath.Join(repoDir, "config", "application.rb"), "module Demo; end\n")
	mustWriteTestFile(t, filepath.Join(repoDir, "bin", "rails"), "#!/usr/bin/env ruby\n")

	plan, err := (&RubyBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	want := `sh -lc "bundle exec bin/rails server -e production -b 0.0.0.0 -p ${PORT:-3000}"`
	if plan.StartCmd != want {
		t.Fatalf("unexpected rails start command: got %q want %q", plan.StartCmd, want)
	}
}
