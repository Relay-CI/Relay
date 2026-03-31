//go:build !linux

package main

import "os"

// prepareImageRootfs materialises a working copy of snapshotName for
// containerID. On non-Linux platforms overlayfs is unavailable, so we always
// produce a hardlink copy via prepareImageWorkdir (cache.go).
func prepareImageRootfs(snapshotName, containerID string) (rootfsDir string, isOverlay bool, err error) {
	wd, err := prepareImageWorkdir(snapshotName, containerID)
	return wd, false, err
}

// releaseImageRootfs removes the hardlink working copy on container stop.
func releaseImageRootfs(rec *ContainerRecord) {
	if rec.WorkDir != "" {
		_ = os.RemoveAll(rec.WorkDir)
	}
}

// imageWorkPath returns the WorkDir path to store in ContainerRecord.
// On non-Linux, always a flat hardlink-copy directory.
func imageWorkPath(containerID string, _ bool) string {
	return workdirPath(containerID)
}
