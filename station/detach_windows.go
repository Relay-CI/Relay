//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

const windowsDetachedProcess = 0x00000008

// startDetachedProcess forks a background process that survives the parent.
// Uses CREATE_NEW_PROCESS_GROUP so Ctrl+C in the parent doesn't kill the child.
func startDetachedProcess(exe string, args []string) (int, error) {
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer devNull.Close()

	cmd := exec.Command(exe, args...)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windowsDetachedProcess,
		HideWindow:    true,
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	go cmd.Wait()
	return cmd.Process.Pid, nil
}
