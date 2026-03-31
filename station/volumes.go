package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func safeStateName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	replacer := strings.NewReplacer(":", "_", "/", "_", "\\", "_")
	return replacer.Replace(name)
}

func splitVolumeSpec(spec string) (source, remainder string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", ""
	}
	if len(spec) >= 2 && spec[1] == ':' {
		if idx := strings.Index(spec[2:], ":"); idx >= 0 {
			return spec[:idx+2], spec[idx+2:]
		}
		return spec, ""
	}
	if idx := strings.Index(spec, ":"); idx >= 0 {
		return spec[:idx], spec[idx:]
	}
	return spec, ""
}

func parseVolumeSpec(spec string) (source, target, options string, ok bool) {
	source, remainder := splitVolumeSpec(spec)
	if source == "" || remainder == "" || !strings.HasPrefix(remainder, ":") {
		return "", "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(remainder, ":"), ":", 2)
	target = strings.TrimSpace(parts[0])
	if target == "" {
		return "", "", "", false
	}
	if len(parts) == 2 {
		options = strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(source), target, options, true
}

func formatVolumeSpec(source, target, options string) string {
	spec := source + ":" + target
	if strings.TrimSpace(options) != "" {
		spec += ":" + strings.TrimSpace(options)
	}
	return spec
}

func isWindowsAbsPath(path string) bool {
	return len(path) >= 2 && path[1] == ':'
}

func isNamedVolumeSource(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	if filepath.IsAbs(source) || isWindowsAbsPath(source) {
		return false
	}
	if strings.HasPrefix(source, ".") {
		return false
	}
	return !strings.ContainsAny(source, `/\`)
}

func managedVolumeBaseDir() string {
	if override := strings.TrimSpace(os.Getenv("STATION_VOLUME_BASE")); override != "" {
		return override
	}
	return filepath.Join(stateBaseDir(), "volumes")
}

func managedVolumePath(name string) string {
	return filepath.Join(managedVolumeBaseDir(), safeStateName(name))
}

func resolveVolumeSpec(spec string) (string, error) {
	source, target, options, ok := parseVolumeSpec(spec)
	if !ok {
		return "", fmt.Errorf("invalid volume spec %q", spec)
	}
	resolvedSource := source
	if isNamedVolumeSource(source) {
		resolvedSource = managedVolumePath(source)
		if err := os.MkdirAll(resolvedSource, 0755); err != nil {
			return "", err
		}
	}
	return formatVolumeSpec(resolvedSource, target, options), nil
}

func resolveVolumeSpecs(specs []string) ([]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		resolved, err := resolveVolumeSpec(spec)
		if err != nil {
			return nil, err
		}
		out = append(out, resolved)
	}
	return out, nil
}

func wslVolumeSpec(spec string) string {
	resolved, err := resolveVolumeSpec(spec)
	if err != nil {
		return spec
	}
	source, target, options, ok := parseVolumeSpec(resolved)
	if !ok {
		return spec
	}
	if isWindowsAbsPath(source) {
		source = windowsPathToWSLPath(source)
	}
	return formatVolumeSpec(source, target, options)
}

func windowsPathToWSLPath(p string) string {
	if len(p) >= 2 && p[1] == ':' {
		drive := strings.ToLower(string(p[0]))
		rest := filepath.ToSlash(p[2:])
		return "/mnt/" + drive + rest
	}
	return filepath.ToSlash(p)
}
