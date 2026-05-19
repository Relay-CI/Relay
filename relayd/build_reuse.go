package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

func (s *Server) canReuseRuntimeArtifact(prev *AppState, engine, repoHash string) bool {
	if prev == nil || strings.TrimSpace(prev.CurrentImage) == "" {
		return false
	}
	if strings.TrimSpace(repoHash) == "" || prev.RepoHash != repoHash {
		return false
	}
	if firstNonEmptyEngine(prev.Engine) != firstNonEmptyEngine(engine) {
		return false
	}
	return runtimeArtifactExists(engine, prev.CurrentImage)
}

func runtimeArtifactExists(engine, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if firstNonEmptyEngine(engine) == EngineStation {
		// On Windows, the primary snapshot store is WSL2-native; check via agent.
		if runtime.GOOS == "windows" {
			if agent, err := getStationAgent(); err == nil && agent != nil {
				return agent.SnapshotExists(ref)
			}
		}
		return fileExists(stationSnapshotManifestPath(ref))
	}
	cmd := exec.Command("docker", "image", "inspect", ref)
	return cmd.Run() == nil
}

func previousDeployImage(prev *AppState, artifactRef string, reused bool) string {
	if prev == nil {
		return ""
	}
	if reused && strings.TrimSpace(prev.CurrentImage) == strings.TrimSpace(artifactRef) {
		return prev.PreviousImage
	}
	return prev.CurrentImage
}

func shouldIgnoreRepoPath(rel string) bool {
	top := strings.Split(rel, "/")[0]
	switch top {
	case ".git", "node_modules", ".next", "dist", ".turbo", "coverage", ".relay", "cache", "bin", "obj", "target":
		return true
	default:
		return false
	}
}

func buildInputFingerprint(repoDir string, buildEnv map[string]string) string {
	base := repoFingerprint(repoDir)
	if len(buildEnv) == 0 {
		return base
	}
	h := sha256.New()
	_, _ = h.Write([]byte(base))
	_, _ = h.Write([]byte{0})
	keys := make([]string, 0, len(buildEnv))
	for key := range buildEnv {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _ = h.Write([]byte(key))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(buildEnv[key]))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
