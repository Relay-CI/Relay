package main

import (
	"fmt"
	"os"
	"strings"
)

func emptyBuildManifest() *BuildManifest {
	return &BuildManifest{Env: map[string]string{}}
}

func resolveImageRootfs(image string) (rootfs string, manifest *BuildManifest, err error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return "", nil, fmt.Errorf("image name required")
	}

	snapshot := snapshotPath(image)
	if _, statErr := os.Stat(snapshot); statErr == nil {
		manifest, err = loadManifest(snapshot)
		if err != nil {
			manifest = emptyBuildManifest()
		}
		return snapshot, manifest, nil
	}

	rootfs, manifest, err = PullImage(image, func(string, ...any) {})
	if err != nil {
		return "", nil, err
	}
	if manifest == nil {
		manifest = emptyBuildManifest()
	}
	return rootfs, manifest, nil
}

func loadImageManifest(image string) (*BuildManifest, error) {
	_, manifest, err := resolveImageRootfs(image)
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		manifest = emptyBuildManifest()
	}
	return manifest, nil
}
