//go:build !linux

package main

import "os"

// prepareNetworkEnv is a no-op on non-Linux platforms.
func prepareNetworkEnv(rec *ContainerRecord) error { return nil }

// teardownContainerNetwork is a no-op on non-Linux platforms.
func teardownContainerNetwork(rec *ContainerRecord) {}

// netAllocCmd is a no-op stub on non-Linux; the real implementation lives
// in network_linux.go and is invoked via WSL2 by the Windows host.
func netAllocCmd(key string) { os.Exit(1) }
