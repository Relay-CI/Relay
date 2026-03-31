//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// createNoWindow prevents child console processes from opening a visible
// console window or Windows Terminal tab on the Windows desktop.
const createNoWindow uint32 = 0x08000000

// setCmdHideWindow launches cmd without a console window and in its own
// process group so background helpers do not steal focus.
func setCmdHideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow,
	}
}
