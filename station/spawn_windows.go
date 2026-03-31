//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

func platformName() string {
	if wslAvailable() {
		if d := wslDefaultDistro(); d != "" {
			return fmt.Sprintf("windows → WSL2 (%s) — Linux namespaces + chroot", d)
		}
	}
	return "windows (WSL2 not available — station run is disabled; install WSL2 for container support)"
}

func platformSetupRootfsHint() string {
	return "Download Alpine mini rootfs inside WSL2 (requires WSL2)"
}

// isContainerInit is always false on Windows — the re-exec pattern only runs
// on Linux inside WSL2, not in the Windows host process.
func isContainerInit() bool { return false }
func runContainerInit()     { /* no-op: re-exec only happens on Linux inside WSL2 */ }

// resolveDir handles the common Windows mistake of passing an exe path where
// a directory is expected. If dir is a file, use its parent as the working
// directory and prepend the file as the command.
func resolveDir(dir string, cmdArgs []string) (string, []string) {
	info, err := os.Stat(dir)
	if err == nil && !info.IsDir() {
		return filepath.Dir(dir), append([]string{dir}, cmdArgs...)
	}
	return dir, cmdArgs
}

// ─── process liveness ────────────────────────────────────────────────────────

// pidAlive checks whether a process (Windows PID) is still running.
// WSL2 processes have real Windows PIDs, so this works for both native and
// WSL2-launched containers.
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
	return code == 259 // STILL_ACTIVE
}

// killProcess terminates a process. Sends WM_CLOSE first for graceful exit,
// then falls back to TerminateProcess. Works for both native and WSL2 PIDs.
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

// postCloseMessages sends WM_CLOSE to all windows owned by the target PID,
// giving GUI/console apps a chance to exit cleanly.
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

// ─── spawn ────────────────────────────────────────────────────────────────────

// doSpawn always routes through WSL2 for real Linux isolation (PID + mount +
// UTS + IPC namespaces inside a Hyper-V Linux VM). It auto-provisions a
// minimal Alpine distro named "station-linux" on first use when no user distro
// (e.g. Ubuntu) is present. If WSL2 is not installed at all, it returns an
// error — station does not silently fall back to a non-isolated Windows process.
func doSpawn(rec *ContainerRecord, foreground bool, logWriter io.Writer) (int, error) {
	if !wslAvailable() {
		return 0, fmt.Errorf(
			"station requires WSL2 for container isolation on Windows\n" +
				"Install WSL2: https://learn.microsoft.com/windows/wsl/install\n" +
				"After installing, station will auto-provision a minimal Alpine Linux distro.")
	}
	distro := wslDefaultDistro() // auto-provisions vessel-linux if needed
	if distro == "" || !wslIsRegistered(distro) {
		return 0, fmt.Errorf("WSL2 is available but no Linux distro could be found or provisioned")
	}
	pid, err := doSpawnWSL(distro, rec, foreground, logWriter)
	if err == nil {
		// Keep the WSL VM hot so the next station call doesn't pay cold-boot cost.
		go wslEnsureKeepalive(distro)
	}
	return pid, err
}

// platformWslEnsure ensures the Linux station binary is installed in the default
// WSL2 distro.  Exposed via "station wsl-ensure" so relayd's station agent can
// trigger installation without duplicating wslEnsureStation logic.
func platformWslEnsure() {
	if !wslAvailable() {
		fmt.Println("WSL2 is not available.")
		return
	}
	distro := wslDefaultDistro()
	if distro == "" || !wslIsRegistered(distro) {
		fmt.Println("No WSL2 distro found.")
		return
	}
	if err := wslEnsureStation(distro); err != nil {
		fmt.Fprintf(os.Stderr, "station wsl-ensure: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("station ready in WSL2 distro %q\n", distro)
}

// platformWslWarm warms the default WSL2 distro synchronously and starts a
// keepalive. Exposed via "station wsl-warm" for use in Windows startup tasks.
func platformWslWarm() {
	if !wslAvailable() {
		fmt.Println("WSL2 is not available.")
		return
	}
	distro := wslDefaultDistro()
	if distro == "" || !wslIsRegistered(distro) {
		fmt.Println("No WSL2 distro found.")
		return
	}
	wslWarm(distro)
}

// doSpawnNative runs the process directly on Windows with no isolation.
// Used as a fallback when WSL2 is unavailable.
func doSpawnNative(rec *ContainerRecord, foreground bool, logWriter io.Writer) (int, error) {
	cmd := exec.Command(rec.Command[0], rec.Command[1:]...)
	cmd.Dir = rec.Dir
	cmd.Env = buildEnv(rec.Env)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	if foreground {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return 0, cmd.Run()
	}
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	go cmd.Wait()
	return cmd.Process.Pid, nil
}

// ─── setup-rootfs ────────────────────────────────────────────────────────────

// platformSkipImagePrep returns true when the named image exists in WSL2's
// own snapshot store.  In that case doSpawnWSL will pass --image to the Linux
// vessel, which materialises an overlayfs on native ext4 — no Windows-side
// hardlink working copy is needed.
func platformSkipImagePrep(image string) bool {
	if image == "" || !wslAvailable() {
		return false
	}
	distro := wslDefaultDistro()
	if distro == "" {
		return false
	}
	return wslSnapshotExists(distro, image)
}

// platformSetupRootfs downloads the Alpine rootfs inside WSL2 where it is
// actually useful (Linux filesystem, run by the Linux station binary).
func platformSetupRootfs(dest string) error {
	if !wslAvailable() {
		fmt.Println("WSL2 is not installed.")
		fmt.Println("setup-rootfs downloads an Alpine Linux rootfs for use with Linux containers.")
		fmt.Println("It requires WSL2. Install it from: https://learn.microsoft.com/windows/wsl/install")
		return nil
	}
	distro := wslDefaultDistro()
	if distro == "" {
		fmt.Println("WSL2 is installed but no Linux distro is registered.")
		fmt.Println("Install a distro first: wsl --install -d Ubuntu")
		return nil
	}
	return wslSetupRootfs(distro, dest)
}
