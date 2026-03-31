package main

import (
	"os/exec"
	"runtime"
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
