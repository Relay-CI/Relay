package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNodeNextBuildpackClassicStartCmdUsesPackageManagerStart(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `export default { reactCompiler: true }`)
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"scripts":{"start":"NODE_OPTIONS=--inspect next start"},
		"dependencies":{"next":"16.1.6"}
	}`)

	plan, err := (&NodeNextBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Kind != "next-classic" {
		t.Fatalf("expected classic next plan, got %q", plan.Kind)
	}
	if plan.StartCmd != "npm start" {
		t.Fatalf("expected classic next buildpack to honor package start script via npm start, got %q", plan.StartCmd)
	}
}

func TestNodeNextBuildpackClassicStartCmdFallsBackToNextCLIWhenStartScriptMissing(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `export default { reactCompiler: true }`)
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{"name":"demo","dependencies":{"next":"16.1.6"}}`)

	plan, err := (&NodeNextBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	want := `exec ./node_modules/.bin/next start --hostname 0.0.0.0 --port ${PORT:-3000}`
	if plan.StartCmd != want {
		t.Fatalf("unexpected classic next fallback start command: got %q want %q", plan.StartCmd, want)
	}
}

func TestNodeNextBuildpackExportUsesStaticRuntime(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `export default { output: "export" }`)
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{
		"name":"demo",
		"scripts":{"build":"next build","start":"next start"},
		"dependencies":{"next":"16.1.6"}
	}`)

	plan, err := (&NodeNextBuildpack{}).Plan(DeployRequest{}, repoDir, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Kind != "next-export" {
		t.Fatalf("expected next-export plan, got %q", plan.Kind)
	}
	if plan.ServicePort != 80 {
		t.Fatalf("expected static export to serve port 80, got %d", plan.ServicePort)
	}
	if plan.StartCmd != "" {
		t.Fatalf("expected static export to avoid node start command, got %q", plan.StartCmd)
	}
	if err := plan.WriteDockerfile(repoDir); err != nil {
		t.Fatalf("write dockerfile: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(repoDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "COPY --from=builder /app/out /usr/share/nginx/html") {
		t.Fatalf("expected dockerfile to copy exported out directory, got:\n%s", got)
	}
	if strings.Contains(got, "next start") || strings.Contains(got, "prod-deps") {
		t.Fatalf("expected dockerfile to avoid Next server runtime work, got:\n%s", got)
	}
}

func TestIsNextStandaloneEnabledIgnoresComments(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `// output: "standalone"
export default { reactCompiler: true }`)

	if isNextStandaloneEnabled(repoDir) {
		t.Fatalf("comment-only standalone hint should not enable next-standalone")
	}
}

func TestIsNextStandaloneEnabledDetectsOutputShorthand(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `const output = "standalone"
export default { output }`)

	if !isNextStandaloneEnabled(repoDir) {
		t.Fatalf("output shorthand should enable next-standalone")
	}
}

func TestIsNextStandaloneEnabledDetectsOutputVariableAssignment(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `const mode = "standalone"
export default { output: mode }`)

	if !isNextStandaloneEnabled(repoDir) {
		t.Fatalf("output variable assignment should enable next-standalone")
	}
}

func TestIsNextExportEnabledDetectsOutputExport(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "next.config.ts"), `const mode = "export"
export default { output: mode }`)

	if !isNextExportEnabled(repoDir) {
		t.Fatalf("output export variable assignment should enable next-export")
	}
}
