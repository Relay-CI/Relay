package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoFingerprintIgnoresTransientDirs(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(repoDir, "src", "app.js"), "console.log('v1')\n")
	mustWriteTestFile(t, filepath.Join(repoDir, "package.json"), `{"name":"demo"}`)
	mustWriteTestFile(t, filepath.Join(repoDir, ".next", "cache", "index"), "ignored-1")
	mustWriteTestFile(t, filepath.Join(repoDir, "node_modules", "left-pad", "index.js"), "ignored-1")

	before := repoFingerprint(repoDir)

	mustWriteTestFile(t, filepath.Join(repoDir, ".next", "cache", "index"), "ignored-2")
	mustWriteTestFile(t, filepath.Join(repoDir, "node_modules", "left-pad", "index.js"), "ignored-2")

	afterIgnoredChange := repoFingerprint(repoDir)
	if before != afterIgnoredChange {
		t.Fatalf("ignored directories changed fingerprint: before=%s after=%s", before, afterIgnoredChange)
	}

	mustWriteTestFile(t, filepath.Join(repoDir, "src", "app.js"), "console.log('v2')\n")
	afterSourceChange := repoFingerprint(repoDir)
	if before == afterSourceChange {
		t.Fatalf("tracked source change did not change fingerprint: %s", before)
	}
}

func TestPreviousDeployImage(t *testing.T) {
	prev := &AppState{
		CurrentImage:  "current-image",
		PreviousImage: "older-image",
	}

	if got := previousDeployImage(prev, "current-image", true); got != "older-image" {
		t.Fatalf("reused deploy should keep previous image, got %q", got)
	}
	if got := previousDeployImage(prev, "new-image", false); got != "current-image" {
		t.Fatalf("new deploy should shift current image to previous, got %q", got)
	}
}

func mustWriteTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
