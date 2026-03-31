//go:build !windows

package main

func maybeRunDesktopUI() bool {
	return false
}
