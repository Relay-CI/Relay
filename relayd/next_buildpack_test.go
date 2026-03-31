package main

import (
	"path/filepath"
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
