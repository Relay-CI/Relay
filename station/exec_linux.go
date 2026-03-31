//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// platformExec enters a running container's namespaces using nsenter(1) and
// replaces the current process image so stdio is fully attached (pty-aware).
// Supports Linux native containers only; WSL2 path is in exec_windows.go.
func platformExec(rec *ContainerRecord, shellArgs []string) error {
	if !pidAlive(rec.PID) {
		return fmt.Errorf("container %s is not running (pid %d)", rec.ID, rec.PID)
	}

	nsenter, err := exec.LookPath("nsenter")
	if err != nil {
		return fmt.Errorf("nsenter not found — install util-linux: %w", err)
	}

	shell := "/bin/sh"
	if len(shellArgs) > 0 {
		shell = shellArgs[0]
		shellArgs = shellArgs[1:]
	}

	args := []string{
		nsenter,
		fmt.Sprintf("--target=%d", rec.PID),
		"--mount", "--uts", "--ipc", "--pid",
	}
	// Only enter network namespace if the container has one.
	if rec.NetMode == "bridge" {
		args = append(args, "--net")
	}
	args = append(args, fmt.Sprintf("--root=/proc/%d/root", rec.PID))
	cwd := strings.TrimSpace(rec.ContainerCwd)
	if cwd == "" {
		cwd = "/"
	} else if cwd[0] != '/' {
		cwd = "/" + cwd
	}
	args = append(args, "--wd="+cwd)
	args = append(args, "--", shell)
	args = append(args, shellArgs...)

	// syscall.Exec replaces this process — no return on success.
	return syscall.Exec(nsenter, args, os.Environ())
}
