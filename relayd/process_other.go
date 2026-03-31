//go:build !windows

package main

import (
	"syscall"
	"time"
)

func pidAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func killProcess(pid int) error {
	_ = syscall.Kill(pid, syscall.SIGTERM)
	for i := 0; i < 20; i++ {
		time.Sleep(200 * time.Millisecond)
		if !pidAlive(pid) {
			return nil
		}
	}
	return syscall.Kill(pid, syscall.SIGKILL)
}
