//go:build linux

package main

// OverlayFS-backed image rootfs for --image runs.
//
// When a container starts with --image <snapshot>, we mount an overlayfs with:
//   lowerdir  = the snapshot   (read-only, shared across all containers)
//   upperdir  = a fresh empty dir (writes go here — unique per container)
//   workdir   = overlayfs bookkeeping dir (same filesystem as upper)
//   merged    = the rootfs passed to doSpawn / chroot
//
// This gives true copy-on-write semantics: unchanged files cost zero extra
// disk space, writes are isolated to the per-container upper dir, and the
// snapshot is never mutated. Multiple containers can share the same snapshot
// simultaneously.
//
// Falls back to the hardlink copy in cache.go when overlayfs is unavailable
// (e.g. missing kernel module, or lower dir on an incompatible filesystem).

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// overlayBasePath returns the directory that holds merged/, upper/, and work/
// for a container. The whole tree is removed on container stop.
func overlayBasePath(containerID string) string {
	return filepath.Join(stateBaseDir(), "overlays", containerID)
}

// prepareImageRootfs creates an overlay mount from the named snapshot.
// Returns the rootfs directory to pass to doSpawn, whether overlay was used,
// and any error.  On failure it automatically falls back to hardlinkCopy.
func prepareImageRootfs(snapshotName, containerID string) (rootfsDir string, isOverlay bool, err error) {
	snap, _, err := resolveImageRootfs(snapshotName)
	if err != nil {
		return "", false, err
	}

	merged, overlayErr := tryOverlayMount(snap, containerID)
	if overlayErr == nil {
		return merged, true, nil
	}
	// Overlay unavailable (no kernel support, tmpfs lower on older kernel,
	// or insufficient privileges) — fall back to a hardlink copy.
	wd, err := prepareImageWorkdir(snapshotName, containerID)
	return wd, false, err
}

// tryOverlayMount attempts to set up an overlayfs at overlayBasePath(containerID).
// Returns the merged directory on success, or an error so the caller can fall back.
func tryOverlayMount(snap, containerID string) (string, error) {
	base := overlayBasePath(containerID)
	merged := filepath.Join(base, "merged")
	upper := filepath.Join(base, "upper")
	work := filepath.Join(base, "work")

	// Start clean.
	_ = os.RemoveAll(base)
	for _, d := range []string{merged, upper, work} {
		if err := os.MkdirAll(d, 0755); err != nil {
			_ = os.RemoveAll(base)
			return "", fmt.Errorf("overlay dir setup: %w", err)
		}
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", snap, upper, work)
	if err := syscall.Mount("overlay", merged, "overlay", 0, opts); err != nil {
		_ = os.RemoveAll(base)
		return "", fmt.Errorf("overlay mount: %w", err)
	}
	return merged, nil
}

// releaseImageRootfs unmounts the overlay (if active) and removes all
// ephemeral directories created for the --image run.
func releaseImageRootfs(rec *ContainerRecord) {
	if rec.OverlayActive && rec.WorkDir != "" {
		// Unmount the merged directory before deleting the tree.
		_ = syscall.Unmount(filepath.Join(rec.WorkDir, "merged"), 0)
	}
	if rec.WorkDir != "" {
		_ = os.RemoveAll(rec.WorkDir)
	}
}

// imageWorkPath returns the value to store in ContainerRecord.WorkDir.
// For overlay: the base dir (overlayBasePath) so releaseImageRootfs can clean
// up merged/ upper/ work/ in one RemoveAll after unmounting.
// For hardlink copies: the workdirPath.
func imageWorkPath(containerID string, isOverlay bool) string {
	if isOverlay {
		return overlayBasePath(containerID)
	}
	return workdirPath(containerID)
}
