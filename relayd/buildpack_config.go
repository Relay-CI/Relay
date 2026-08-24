package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ---------------------- Config ----------------------

type RelayConfig struct {
	Kind         string   `json:"kind"`         // optional hint; else auto-detect
	BuildImage   string   `json:"build_image"`  // docker image to run install/build
	RunImage     string   `json:"run_image"`    // runtime image; if empty, defaults per pack
	ServicePort  int      `json:"service_port"` // container port
	InstallCmd   string   `json:"install_cmd"`
	BuildCmd     string   `json:"build_cmd"`
	StartCmd     string   `json:"start_cmd"`
	ProjectRoot  string   `json:"project_root"`      // repo-relative app root for monorepos
	BuildContext string   `json:"build_context"`     // repo-relative docker build context
	Dockerfile   string   `json:"dockerfile"`        // repo-relative dockerfile path
	Volumes      []string `json:"volumes,omitempty"` // persistent volume mounts e.g. ["/data"]
}

func cleanRepoRelativePath(value string) (string, error) {
	trimmed := filepath.ToSlash(strings.TrimSpace(value))
	trimmed = strings.TrimPrefix(trimmed, "./")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "", nil
	}
	if strings.Contains(trimmed, "\x00") {
		return "", fmt.Errorf("path contains invalid null byte")
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." {
		return "", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path must stay inside the repository")
	}
	return cleaned, nil
}

func resolveRepoRelativePath(repoDir string, value string) (string, string, error) {
	rel, err := cleanRepoRelativePath(value)
	if err != nil {
		return "", "", err
	}
	if rel == "" {
		return repoDir, "", nil
	}
	abs := filepath.Join(repoDir, filepath.FromSlash(rel))
	return abs, rel, nil
}

func candidateDockerfiles(root string) []string {
	candidates := []string{
		"Dockerfile",
		"dockerfile",
		"Containerfile",
		"containerfile",
	}
	out := make([]string, 0, len(candidates))
	for _, name := range candidates {
		p := filepath.Join(root, name)
		info, err := os.Stat(p)
		if err == nil && info != nil && !info.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

func findDockerfileRecursive(root string, maxDepth int) string {
	type item struct {
		dir   string
		depth int
	}
	if root == "" || maxDepth < 0 {
		return ""
	}
	queue := []item{{dir: root, depth: 0}}
	skipDirs := map[string]bool{
		".git":         true,
		".next":        true,
		".turbo":       true,
		"bin":          true,
		"build":        true,
		"coverage":     true,
		"dist":         true,
		"node_modules": true,
		"obj":          true,
		"out":          true,
		"target":       true,
		"vendor":       true,
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if matches := candidateDockerfiles(current.dir); len(matches) > 0 {
			return matches[0]
		}
		if current.depth >= maxDepth {
			continue
		}
		entries, err := os.ReadDir(current.dir)
		if err != nil {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if skipDirs[strings.ToLower(strings.TrimSpace(entry.Name()))] {
				continue
			}
			queue = append(queue, item{dir: filepath.Join(current.dir, entry.Name()), depth: current.depth + 1})
		}
	}
	return ""
}

func detectUserDockerfile(repoDir string, projectRoot string, buildContext string) string {
	searchRoots := []string{projectRoot, buildContext, repoDir}
	seen := map[string]bool{}
	for _, root := range searchRoots {
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		if matches := candidateDockerfiles(root); len(matches) > 0 {
			if rel, err := filepath.Rel(repoDir, matches[0]); err == nil {
				return filepath.ToSlash(rel)
			}
		}
	}
	for _, root := range searchRoots {
		if root == "" {
			continue
		}
		if match := findDockerfileRecursive(root, 4); match != "" {
			if rel, err := filepath.Rel(repoDir, match); err == nil {
				return filepath.ToSlash(rel)
			}
		}
	}
	return ""
}

func readRelayConfig(repoDir string) (*RelayConfig, error) {
	p := filepath.Join(repoDir, "relay.config.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c RelayConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
