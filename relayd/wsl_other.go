//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// setCmdHideWindow is a no-op on non-Windows platforms (wsl.exe is Windows-only).
func setCmdHideWindow(cmd *exec.Cmd) {}

// setLongRunCmdAttrs puts the process into its own process group so that a
// timeout can kill the entire tree (station + build containers + containerd).
func setLongRunCmdAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killLongRunCmd kills the entire process group started by cmd.
func killLongRunCmd(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// Negative PID targets the process group.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_ = cmd.Process.Kill() // belt-and-suspenders: also kill the leader
}
