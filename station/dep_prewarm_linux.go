//go:build linux

package main

// Dependency prewarming cache — avoids cold npm ci / go mod download / pip install
// runs when the lockfile and base image have not changed.
//
// How it works:
//   1. execInstruction detects a recognized install command (npm ci, yarn
//      install --frozen-lockfile, go mod download, pip install -r requirements.txt).
//   2. It computes a cache key from the install command + relevant lockfile
//      content + any COPY-injected lockfile bytes already in the stage rootfs.
//   3. On cache hit: the cached dependency directory (node_modules, vendor,
//      __pycache__ + site-packages) is hard-linked into the stage rootfs — this
//      costs milliseconds instead of minutes.
//   4. On cache miss: the install runs normally; afterwards the installed
//      directory is saved to the dep cache for future builds.
//
// Cache keys are SHA-256 hashes; the store lives at
//   /tmp/relay-native/dep-cache/<first-2-hex>/<full-hash>/

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// depInstallRule maps a recognized install command prefix to the lockfile and
// installed-artifact directory it produces.
type depInstallRule struct {
	// cmdPrefix is the beginning of the RUN shell command (case-sensitive).
	cmdPrefix string
	// lockfile is the path relative to the stage rootfs workdir to hash.
	lockfile string
	// artifactDir is the directory name produced by the install (relative to workdir).
	artifactDir string
}

var depInstallRules = []depInstallRule{
	{"npm ci", "package-lock.json", "node_modules"},
	{"npm install --frozen-lockfile", "package-lock.json", "node_modules"},
	{"npm install --ci", "package-lock.json", "node_modules"},
	{"yarn install --frozen-lockfile", "yarn.lock", "node_modules"},
	{"yarn install --immutable", "yarn.lock", "node_modules"},
	{"yarn --frozen-lockfile", "yarn.lock", "node_modules"},
	{"pnpm install --frozen-lockfile", "pnpm-lock.yaml", "node_modules"},
	{"pip install -r requirements.txt", "requirements.txt", ".pip-installed"},
	{"pip install --no-cache-dir -r requirements.txt", "requirements.txt", ".pip-installed"},
	{"go mod download", "go.sum", ""},  // caches the Go module cache, not a workdir subdir
	{"go mod vendor", "go.sum", "vendor"},
}

// depCacheDir returns the root of the dep cache store.
func depCacheDir() string {
	return filepath.Join(stateBaseDir(), "dep-cache")
}

// depCachePath returns the on-disk directory for a given cache key.
func depCachePath(key string) string {
	if len(key) < 2 {
		return filepath.Join(depCacheDir(), key)
	}
	return filepath.Join(depCacheDir(), key[:2], key)
}

// depCacheKey computes a deterministic SHA-256 key for a dep install command.
// It hashes:
//   - the matched command prefix (identifies the tool)
//   - the content of the lockfile found in rootfsWorkdir
//
// Returns ("", "", false) when the shell command is not a recognized install
// command or when the lockfile cannot be read.
func depCacheKey(shell, rootfs, workdir string) (key, artifactDir string, ok bool) {
	shell = strings.TrimSpace(shell)
	for _, rule := range depInstallRules {
		if !strings.HasPrefix(shell, rule.cmdPrefix) {
			continue
		}
		if rule.lockfile == "" {
			return "", "", false // go mod download uses GOPATH, not a workdir subdir
		}
		lockPath := filepath.Join(rootfsContainerPath(rootfs, workdir), rule.lockfile)
		data, err := os.ReadFile(lockPath)
		if err != nil {
			// Lockfile not yet in rootfs — COPY hasn't run yet; skip cache.
			return "", "", false
		}
		h := sha256.New()
		fmt.Fprintf(h, "dep-prewarm:%s\n", rule.cmdPrefix)
		_, _ = h.Write(data)
		return fmt.Sprintf("%x", h.Sum(nil)), rule.artifactDir, true
	}
	return "", "", false
}

// depCacheRestore hard-links the cached artifact directory into rootfs at
// workdir/artifactDir.  Returns true on success.
func depCacheRestore(key, artifactDir, rootfs, workdir string) bool {
	if artifactDir == "" {
		return false
	}
	cacheSrc := filepath.Join(depCachePath(key), "artifact")
	if fi, err := os.Stat(cacheSrc); err != nil || !fi.IsDir() {
		return false
	}
	dest := filepath.Join(rootfsContainerPath(rootfs, workdir), artifactDir)
	_ = os.RemoveAll(dest)
	if err := hardlinkCopy(cacheSrc, dest); err != nil {
		_ = os.RemoveAll(dest)
		return false
	}
	return true
}

// depCacheSave stores artifactDir from rootfs/workdir into the dep cache under
// key.  Uses hard links — fast and storage-efficient.  Best-effort: errors are
// silently discarded so a cache-save failure never breaks a build.
func depCacheSave(key, artifactDir, rootfs, workdir string) {
	if artifactDir == "" || key == "" {
		return
	}
	src := filepath.Join(rootfsContainerPath(rootfs, workdir), artifactDir)
	if _, err := os.Stat(src); err != nil {
		return
	}
	dest := filepath.Join(depCachePath(key), "artifact")
	_ = os.RemoveAll(dest)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return
	}
	_ = hardlinkCopy(src, dest)
}
