//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// platformExec enters a running container's namespaces via WSL2 + nsenter.
// It searches /proc inside the WSL2 VM for a process whose environment
// contains CONTAINER_ID=<id>, then nsenter's into its mount/uts/ipc/pid/net
// namespaces. Falls back to a plain error for native Windows processes.
func platformExec(rec *ContainerRecord, shellArgs []string) error {
	if !pidAlive(rec.PID) {
		return fmt.Errorf("container %s is not running (pid %d)", rec.ID, rec.PID)
	}

	if !wslAvailable() {
		return fmt.Errorf("exec is only supported for WSL2 containers — install WSL2 for namespace entry")
	}

	distro := wslDefaultDistro()
	if distro == "" {
		return fmt.Errorf("no WSL2 distro registered")
	}

	shell := "/bin/sh"
	if len(shellArgs) > 0 {
		shell = shellArgs[0]
		shellArgs = shellArgs[1:]
	}

	cmdParts := make([]string, 0, len(shellArgs)+1)
	cmdParts = append(cmdParts, shq(shell))
	for _, arg := range shellArgs {
		cmdParts = append(cmdParts, shq(arg))
	}
	netFlag := ""
	if rec.NetMode == "bridge" {
		netFlag = " --net"
	}
	cwd := strings.TrimSpace(rec.ContainerCwd)
	if cwd == "" {
		cwd = "/"
	} else if !strings.HasPrefix(cwd, "/") {
		cwd = "/" + cwd
	}
	wdFlag := " --wd=" + shq(cwd)

	// Search every process's environment for CONTAINER_ID=<id>,
	// then nsenter into its namespaces.
	script := fmt.Sprintf(`
cid=%s
pid=""
for f in /proc/[0-9]*/environ; do
  tr '\0' '\n' < "$f" 2>/dev/null | grep -q "CONTAINER_ID=$cid" || continue
  pid="${f%%/environ}"; pid="${pid##*/}"; break
done
[ -z "$pid" ] && { echo "vessel: container $cid not found in WSL2" >&2; exit 1; }
exec nsenter --target "$pid" --mount --uts --ipc --pid%s --root="/proc/$pid/root"%s -- %s`,
		shq(rec.ID), netFlag, wdFlag, strings.Join(cmdParts, " "))

	cmd := exec.Command("wsl.exe", "-d", distro, "--", "bash", "-c", script)
	setCmdHideWindow(cmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
