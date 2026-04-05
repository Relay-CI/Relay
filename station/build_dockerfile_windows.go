//go:build windows

package main

// Dockerfile builder — Windows implementation.
//
// Delegates the build to the Linux station binary inside WSL2.
// The build output is written to a WSL2-native temp directory (/tmp) so that
// all file I/O during the build goes to ext4, not /mnt/c/ (9P). The Windows
// side only receives the manifest JSON; the full rootfs stays in WSL2 for a
// fast in-VM snapshot save (hardlinks within ext4 — no cross-FS copy needed).

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// wslBuildBase is the WSL2-internal temp directory used for build output.
const wslBuildBase = "/tmp/relay-native/builds"

// wslBuildDirFile is the name of the breadcrumb file written to the Windows
// outDir so that buildStationSnapshot can locate the WSL2-internal build dir.
const wslBuildDirFile = ".wsl-build-dir"

// BuildDockerfile converts Windows paths to WSL mount paths, invokes
// station build-dockerfile inside WSL2, then reads back the manifest.
//
// The actual build output is written to a WSL2-internal directory
// (wslBuildBase/<name>) rather than outDir so that Docker layer extraction
// and npm/go builds hit ext4 instead of the slow 9P filesystem.
// Only the manifest JSON is copied back to the Windows outDir; the full
// rootfs stays in WSL2 for a fast in-VM snapshot save.
// ctx is accepted for API consistency with the Linux implementation but is
// not used: the build is delegated to WSL2 via wsl.exe which lacks a
// cancellation channel.
func BuildDockerfile(_ context.Context, dockerfilePath, contextDir, outDir string, logf func(string, ...any), logw io.Writer) (*BuildManifest, error) {
	distro := wslDefaultDistro()
	if distro == "" {
		return nil, fmt.Errorf("no WSL2 distro found — install WSL2 first")
	}
	if err := wslEnsureStation(distro); err != nil {
		return nil, err
	}

	// Build to a WSL2-internal path so all I/O goes to native ext4.
	// filepath.Base(outDir) gives us the unique build ID already embedded in
	// the path by the caller (e.g. "relay__site__preview__main__<id>").
	wslBuildDir := wslBuildBase + "/" + filepath.Base(outDir)

	cmd := fmt.Sprintf(
		"rm -rf %s && mkdir -p %s && %s build-dockerfile %s %s %s",
		shq(wslBuildDir),
		shq(wslBuildBase),
		wslStationBin,
		shq(toWSLPath(dockerfilePath)),
		shq(toWSLPath(contextDir)),
		shq(wslBuildDir),
	)
	logf("[build] delegating to WSL2 distro %q", distro)

	if err := wslRunStreaming(distro, cmd, logw); err != nil {
		return nil, fmt.Errorf("build in WSL2: %w", err)
	}

	// Ensure outDir exists on the Windows side for manifest/breadcrumb files.
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("create outDir: %w", err)
	}

	// Copy the manifest JSON back to the Windows outDir so loadManifest works
	// without another WSL2 round-trip in the caller.
	if manifestJSON, readErr := wslRun(distro, "cat "+shq(wslBuildDir+"/station-manifest.json")); readErr == nil {
		manifestJSON = strings.TrimRight(manifestJSON, "\r\n")
		if manifestJSON != "" {
			_ = os.WriteFile(filepath.Join(outDir, "station-manifest.json"), []byte(manifestJSON), 0644)
		}
	}

	// Leave a breadcrumb so buildStationSnapshot knows the WSL2-internal build
	// dir, allowing a fast snapshot save via hardlinks within ext4.
	_ = os.WriteFile(filepath.Join(outDir, wslBuildDirFile), []byte(wslBuildDir+"\n"), 0644)

	// Read the manifest from the Windows outDir (just written above).
	m, loadErr := loadManifest(filepath.Join(outDir))
	if loadErr != nil {
		return &BuildManifest{Env: map[string]string{}}, nil
	}
	return m, nil
}

// isBuildRunInit is always false on Windows; the chroot re-exec only runs on Linux.
func isBuildRunInit() bool { return false }

// runBuildInit is a no-op on Windows.
func runBuildInit() {}


