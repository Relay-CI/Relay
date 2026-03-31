//go:build !windows

package main

import "os/exec"

// setCmdHideWindow is a no-op on non-Windows platforms (wsl.exe is Windows-only).
func setCmdHideWindow(cmd *exec.Cmd) {}
