// warm_images.go — best-effort background prefetch of common buildpack base
// images so the FIRST deploy of a new stack on this host doesn't pay for a
// multi-second `docker pull` stacked on top of a cold build cache. This is
// what makes "one command, first deploy is fast" actually true instead of
// only being true from the second deploy onward.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// warmBuildpackBaseImagesList is the set of base images relay's built-in
// buildpacks pull by default (see the Plan() functions in main.go). Kept as
// a flat list here rather than derived from the buildpacks themselves
// because a few images are chosen dynamically per-repo (e.g. Java's runtime
// varies by build tool) — this covers each stack's common, static default.
func warmBuildpackBaseImagesList() []string {
	return []string{
		getenv("RELAY_NODE_IMAGE", "node:22"),
		getenv("RELAY_NODE_RUN_IMAGE", "node:22-slim"),
		getenv("RELAY_NGINX_IMAGE", "nginx:alpine"),
		getenv("RELAY_GO_IMAGE", "golang:1.22"),
		getenv("RELAY_GO_RUN_IMAGE", "gcr.io/distroless/base-debian12"),
		getenv("RELAY_PY_IMAGE", "python:3.12"),
		getenv("RELAY_PY_RUN_IMAGE", "python:3.12-slim"),
		getenv("RELAY_RUBY_IMAGE", "ruby:3.3"),
		getenv("RELAY_RUBY_RUN_IMAGE", "ruby:3.3-slim"),
		getenv("RELAY_PHP_BUILD_IMAGE", "composer:2"),
		getenv("RELAY_PHP_IMAGE", "php:8.3-cli"),
	}
}

func dockerImagePresent(image string) bool {
	return exec.Command("docker", "image", "inspect", image).Run() == nil
}

// warmBuildpackBaseImages pulls any default base image not already present
// on this host, sequentially — this must never compete with an in-progress
// user build for bandwidth/CPU on a small host, so no parallel pulls and a
// startup delay so boot-time work (restoring the proxy, resuming rollout
// watches) always goes first. Runs once; RELAY_WARM_IMAGES=0 disables it
// entirely for firewalled or registry-restricted hosts.
func (s *Server) warmBuildpackBaseImages() {
	if strings.TrimSpace(os.Getenv("RELAY_WARM_IMAGES")) == "0" {
		return
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return
	}
	time.Sleep(30 * time.Second)

	pulled := 0
	for _, image := range warmBuildpackBaseImagesList() {
		image = strings.TrimSpace(image)
		if image == "" || dockerImagePresent(image) {
			continue
		}
		if exec.Command("docker", "pull", image).Run() == nil {
			pulled++
		}
	}
	if pulled > 0 {
		fmt.Printf("warm-images: pulled %d buildpack base image(s) ahead of first use\n", pulled)
	}
}
