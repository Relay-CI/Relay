//go:build !linux && !windows

package main

// Dockerfile builder — macOS / other stub.
// Dockerfile builds are only supported on Linux (natively) and Windows (via WSL2).

import (
	"context"
	"fmt"
	"io"
)

func BuildDockerfile(_ context.Context, _, _, _ string, _ func(string, ...any), _ io.Writer) (*BuildManifest, error) {
	return nil, fmt.Errorf("build-dockerfile is only supported on Linux and Windows (WSL2)")
}

func isBuildRunInit() bool { return false }
func runBuildInit()        {}
