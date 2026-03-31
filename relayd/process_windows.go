//go:build windows

package main

import (
	"syscall"
	"time"
	"unsafe"
)

func pidAlive(pid int) bool {
	const processQueryLimitedInformation = 0x1000
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259
}

func killProcess(pid int) error {
	postCloseMessages(uint32(pid))
	for i := 0; i < 15; i++ {
		time.Sleep(200 * time.Millisecond)
		if !pidAlive(pid) {
			return nil
		}
	}
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	return syscall.TerminateProcess(h, 1)
}

func postCloseMessages(targetPID uint32) {
	user32 := syscall.NewLazyDLL("user32.dll")
	enumWindows := user32.NewProc("EnumWindows")
	getWindowThreadProcessId := user32.NewProc("GetWindowThreadProcessId")
	postMessage := user32.NewProc("PostMessageW")
	const wmClose = 0x0010
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == targetPID {
			postMessage.Call(hwnd, wmClose, 0, 0)
		}
		return 1
	})
	enumWindows.Call(cb, 0)
}
