//go:build !linux

package main

import (
	"fmt"
	"os"
)

// cmdDaemon is a no-op on non-Linux platforms.  The daemon runs inside WSL2
// (Linux), so this branch is never reached in normal usage.
func cmdDaemon(_ string) {
	fmt.Fprintln(os.Stderr, "station daemon: only supported on Linux (run inside WSL2)")
	os.Exit(1)
}

