package main

import (
	"path/filepath"
	"strings"
)

func shouldLaunchDesktopUI(args []string, parentName string) bool {
	if len(args) != 1 {
		return false
	}

	parent := strings.ToLower(strings.TrimSpace(filepath.Base(parentName)))
	if parent == "" {
		// If parent inspection fails, prefer desktop mode for no-arg launches.
		return true
	}

	if isKnownShellParent(parent) {
		return false
	}

	return true
}

func isKnownShellParent(parent string) bool {
	switch parent {
	case "powershell.exe", "pwsh.exe", "cmd.exe", "bash.exe", "wsl.exe", "python.exe", "node.exe":
		return true
	case "conhost.exe", "openconsole.exe", "wt.exe", "windowsterminal.exe":
		return true
	default:
		return false
	}
}
