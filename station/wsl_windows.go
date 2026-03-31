//go:build windows

// WSL2 integration for vessel.
//
// Rather than being a process manager that shells to wsl.exe, this uses the
// native wslapi.dll Windows API to launch processes directly inside a WSL2
// distro. WSL2 uses a real Hyper-V Linux VM, so processes launched this way
// get a genuine Linux kernel with full namespace support — the same isolation
// our Linux spawn_linux.go provides.
//
// Architecture:
//   Windows station          WSL2 (Hyper-V Linux VM)
//   ─────────────────        ──────────────────────────────────────────
//   port allocation    →     station run-fg <wslDir> <cmd>
//   state (PID/logs)         ↓ CLONE_NEWPID + CLONE_NEWNS + chroot
//   WslLaunch API     ←─────  Linux process (node, python, go, etc.)
//   Windows PID (trackable)
//
// The Windows side is the source of truth for state (IDs, ports, logs).
// The Linux side handles namespace setup and runs the actual process.
// Because WSL2 processes have real Windows PIDs, pidAlive/killProcess work.

package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ─── wslapi.dll bindings ─────────────────────────────────────────────────────
//
// wslapi.dll ships with Windows 10 build 1903+ when WSL is installed.
// We lazy-load it so station still works on systems without WSL.

// createNoWindow prevents child console processes from opening a visible
// console window or Windows Terminal tab on the Windows desktop.
const createNoWindow uint32 = 0x08000000

func setCmdHideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow,
	}
}

var (
	wslapi = syscall.NewLazyDLL("wslapi.dll")

	// WslIsDistributionRegistered(PCWSTR) BOOL
	procWslIsDistributionRegistered = wslapi.NewProc("WslIsDistributionRegistered")

	// WslLaunch(PCWSTR distro, PCWSTR cmd, BOOL useCwd,
	//           HANDLE in, HANDLE out, HANDLE err, HANDLE* proc) HRESULT
	procWslLaunch = wslapi.NewProc("WslLaunch")

	// WslLaunchInteractive(PCWSTR distro, PCWSTR cmd, BOOL useCwd, DWORD* exit) HRESULT
	procWslLaunchInteractive = wslapi.NewProc("WslLaunchInteractive")
)

// ─── detection ───────────────────────────────────────────────────────────────

// wslAvailable returns true if wslapi.dll can be loaded and wsl.exe is on PATH.
func wslAvailable() bool {
	if err := wslapi.Load(); err != nil {
		return false
	}
	_, err := exec.LookPath("wsl.exe")
	return err == nil
}

func isRunnableWSLDistro(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return false
	}
	switch name {
	case "docker-desktop", "docker-desktop-data":
		return false
	default:
		return true
	}
}

// wslDefaultDistro returns the name of the default WSL2 distro, or "".
// wsl.exe --list --quiet emits UTF-16 LE on some Windows builds, so we strip
// null bytes rather than trying to decode it as UTF-16.
func wslDefaultDistro() string {
	cmd := exec.Command("wsl.exe", "--list", "--quiet")
	setCmdHideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	var fallback string
	for _, line := range strings.Split(stripNulls(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if fallback == "" {
			fallback = line
		}
		if isRunnableWSLDistro(line) {
			return line
		}
	}
	if isRunnableWSLDistro(fallback) {
		return fallback
	}
	// No runnable user distro found — auto-provision a minimal Alpine one.
	if err := wslProvisionStationDistro(); err != nil {
		fmt.Fprintf(os.Stderr, "[station] warn: could not provision %s: %v\n", stationLinuxDistro, err)
		return ""
	}
	return stationLinuxDistro
}

// wslIsRegistered checks via wslapi.dll whether a named distro is installed.
func wslIsRegistered(distro string) bool {
	p, err := syscall.UTF16PtrFromString(distro)
	if err != nil {
		return false
	}
	r, _, _ := procWslIsDistributionRegistered.Call(uintptr(unsafe.Pointer(p)))
	return r != 0
}

// ─── path conversion ─────────────────────────────────────────────────────────

// toWSLPath converts a Windows absolute path to its WSL2 /mnt/X mount path.
//
//	C:\Users\alice\myapp  →  /mnt/c/Users/alice/myapp
func toWSLPath(p string) string {
	if len(p) >= 2 && p[1] == ':' {
		drive := strings.ToLower(string(p[0]))
		rest := filepath.ToSlash(p[2:])
		return "/mnt/" + drive + rest
	}
	return filepath.ToSlash(p)
}

// ─── station auto-install inside WSL2 ────────────────────────────────────────

const wslStationBin = "/usr/local/bin/station"

// stationLinuxDistro is the WSL2 distro name station provisions for itself
// when no user distro (e.g. Ubuntu) is installed.
const stationLinuxDistro = "station-linux"

// wslStationReady returns true if the Linux station binary is already present.
func wslStationReady(distro string) bool {
	out, _ := wslRun(distro, "test -x "+wslStationBin+" && echo yes || echo no")
	return strings.TrimSpace(out) == "yes"
}

// wslSidecarHashFile is where the SHA-256 of the installed sidecar is stored
// inside WSL's native filesystem.  Using a hash avoids reading the Windows
// binary via 9P (/mnt/c/) on every check, which can hang when WSL is cold.
const wslSidecarHashFile = wslStationBin + ".sidecar-sha256"

// sidecarSHA256 computes the hex SHA-256 of the sidecar binary on Windows.
func sidecarSHA256(binaryPath string) (string, error) {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

// wslInstalledMatchesSidecar returns true when the WSL-installed station binary
// was copied from binaryPath (checked via a stored SHA-256 in WSL's own FS).
// No 9P / /mnt/c/ access — avoids hangs during WSL cold-boot mount phase.
func wslInstalledMatchesSidecar(distro, binaryPath string) bool {
	if strings.TrimSpace(binaryPath) == "" {
		return false
	}
	want, err := sidecarSHA256(binaryPath)
	if err != nil {
		return false
	}
	got, err := wslRun(distro, "cat "+shq(wslSidecarHashFile)+" 2>/dev/null")
	if err != nil {
		return false
	}
	return strings.TrimSpace(got) == want
}

func wslLinuxGoArch() string {
	if runtime.GOARCH == "arm64" || strings.EqualFold(strings.TrimSpace(os.Getenv("PROCESSOR_ARCHITECTURE")), "ARM64") {
		return "arm64"
	}
	return "amd64"
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func validStationSourceDir(dir string) bool {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false
	}
	return pathExists(filepath.Join(dir, "go.mod")) && pathExists(filepath.Join(dir, "main.go"))
}

func wslStationSidecarCandidates(selfDir string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 6)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if _, ok := seen[abs]; ok || !pathExists(abs) {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	add(os.Getenv("STATION_WSL_BIN"))
	add(os.Getenv("RELAY_STATION_WSL_BIN"))
	add(filepath.Join(selfDir, "station-linux"))
	add(filepath.Join(selfDir, "station-linux-"+wslLinuxGoArch()))
	return out
}

func wslStationSourceDirCandidates(selfDir string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 8)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if _, ok := seen[abs]; ok || !validStationSourceDir(abs) {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	add(os.Getenv("STATION_SOURCE_DIR"))
	add(os.Getenv("RELAY_STATION_SOURCE_DIR"))
	add(selfDir)
	add(filepath.Join(selfDir, "station"))
	add(filepath.Join(selfDir, "..", "station"))
	add(filepath.Join(selfDir, "..", "..", "station"))
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "station"))
		add(filepath.Join(wd, "..", "station"))
	}
	return out
}

func installWSLStationFromSidecar(distro, binaryPath string) error {
	fmt.Printf("[station] installing packaged Linux binary into WSL2 distro %q...\n", distro)

	hash, err := sidecarSHA256(binaryPath)
	if err != nil {
		return fmt.Errorf("hash sidecar: %w", err)
	}

	// Copy via 9P then immediately write the hash to WSL-native storage.
	// Future checks use the hash file only — no 9P reads after this point.
	cmd := fmt.Sprintf(
		"mkdir -p /usr/local/bin && cp %s %s && chmod 0755 %s && printf '%%s' %s > %s",
		shq(toWSLPath(binaryPath)),
		shq(wslStationBin),
		shq(wslStationBin),
		shq(hash),
		shq(wslSidecarHashFile),
	)
	out, err := wslRunCombined(distro, cmd)
	if err != nil {
		return fmt.Errorf("install packaged Linux station in WSL2:\n%s\n%w", out, err)
	}
	fmt.Printf("[station] installed packaged binary from %s to %s inside WSL2 distro %q\n", binaryPath, wslStationBin, distro)
	return nil
}

func installWSLStationFromSource(distro, sourceDir string) error {
	wslSrc := toWSLPath(sourceDir)

	fmt.Printf("[station] setting up Linux binary in WSL2 distro %q...\n", distro)

	// Check for Go.
	goPath, _ := wslRun(distro, "which go 2>/dev/null || echo missing")
	if strings.TrimSpace(goPath) == "missing" || goPath == "" {
		fmt.Println("[station] Go not found in WSL2 — attempting install (requires sudo)...")
		installCmd := "if command -v apk >/dev/null 2>&1; then apk add --no-cache go" +
			"; elif sudo apt-get update -qq 2>&1 && sudo apt-get install -y -qq golang-go 2>&1; then true" +
			"; elif sudo yum install -y golang 2>&1; then true" +
			"; else echo INSTALL_FAILED && exit 1; fi"
		out, err := wslRunCombined(distro, installCmd)
		if err != nil || strings.Contains(out, "INSTALL_FAILED") {
			return fmt.Errorf(
				"Go not found in WSL2 distro %q and auto-install failed.\n"+
					"Fix: open WSL2 and run:  sudo apt install golang-go  (or: apk add go)\n"+
					"Then retry.", distro)
		}
	}

	// Build station from the source files (accessible via /mnt/...).
	buildCmd := fmt.Sprintf("cd '%s' && GOOS=linux GOARCH=%s go build -o '%s' . 2>&1",
		wslSrc, wslLinuxGoArch(), wslStationBin)
	fmt.Printf("[station] building: %s\n", buildCmd)
	out, err := wslRunCombined(distro, buildCmd)
	if err != nil {
		return fmt.Errorf("build station in WSL2:\n%s\n%w", out, err)
	}
	fmt.Printf("[station] installed at %s inside WSL2 distro %q\n", wslStationBin, distro)
	return nil
}

// wslInstallVessel installs the Linux station binary into WSL2.
// It prefers a packaged sidecar binary beside the Windows executable, then
// falls back to building from a valid local source directory.
func wslInstallVessel(distro string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	selfDir := filepath.Dir(self)

	for _, candidate := range wslStationSidecarCandidates(selfDir) {
		if err := installWSLStationFromSidecar(distro, candidate); err == nil {
			return nil
		}
	}
	for _, candidate := range wslStationSourceDirCandidates(selfDir) {
		if err := installWSLStationFromSource(distro, candidate); err == nil {
			return nil
		}
	}
	return fmt.Errorf(
		"could not install the Linux station binary into WSL2.\n" +
			"Expected either a packaged sidecar like station-linux beside station.exe or a valid station source dir.\n" +
			"Fix: rebuild with scripts/build.ps1, or set RELAY_STATION_SOURCE_DIR to the repo's station folder",
	)
}

func wslEnsureStation(distro string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	selfDir := filepath.Dir(self)
	sidecars := wslStationSidecarCandidates(selfDir)
	if len(sidecars) > 0 {
		for _, candidate := range sidecars {
			if wslInstalledMatchesSidecar(distro, candidate) {
				return nil
			}
			if err := installWSLStationFromSidecar(distro, candidate); err == nil {
				return nil
			}
		}
	}
	if wslStationReady(distro) {
		return nil
	}
	return wslInstallVessel(distro)
}

// ─── WslLaunch wrappers ───────────────────────────────────────────────────────

// wslLaunchDetached launches cmd inside distro with stdin=NUL, stdout/stderr
// redirected to hLog. Returns a Windows HANDLE for the launched process.
// The caller is responsible for calling CloseHandle on the returned handle.
func wslLaunchDetached(distro, cmd string, hLog syscall.Handle) (syscall.Handle, error) {
	distroW, err := syscall.UTF16PtrFromString(distro)
	if err != nil {
		return 0, err
	}
	cmdW, err := syscall.UTF16PtrFromString(cmd)
	if err != nil {
		return 0, err
	}

	hNull, err := openNUL()
	if err != nil {
		return 0, fmt.Errorf("open NUL: %w", err)
	}
	defer syscall.CloseHandle(hNull)

	var procHandle syscall.Handle
	hr, _, _ := procWslLaunch.Call(
		uintptr(unsafe.Pointer(distroW)),
		uintptr(unsafe.Pointer(cmdW)),
		0,              // useCurrentWorkingDirectory = false
		uintptr(hNull), // stdin  = NUL
		uintptr(hLog),  // stdout = log
		uintptr(hLog),  // stderr = log (merged)
		uintptr(unsafe.Pointer(&procHandle)),
	)
	if int32(hr) < 0 {
		return 0, fmt.Errorf("WslLaunch: HRESULT=0x%08X — is %q a WSL2 distro?", uint32(hr), distro)
	}
	return procHandle, nil
}

// wslLaunchFg launches cmd interactively, blocking until it exits.
func wslLaunchFg(distro, cmd string) error {
	distroW, _ := syscall.UTF16PtrFromString(distro)
	cmdW, _ := syscall.UTF16PtrFromString(cmd)

	var exitCode uint32
	hr, _, _ := procWslLaunchInteractive.Call(
		uintptr(unsafe.Pointer(distroW)),
		uintptr(unsafe.Pointer(cmdW)),
		1, // useCurrentWorkingDirectory
		uintptr(unsafe.Pointer(&exitCode)),
	)
	if int32(hr) < 0 {
		return fmt.Errorf("WslLaunchInteractive: HRESULT=0x%08X", uint32(hr))
	}
	if exitCode != 0 {
		return fmt.Errorf("container exited with code %d", exitCode)
	}
	return nil
}

// ─── Windows kernel32 helpers ─────────────────────────────────────────────────

var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	procGetProcID = kernel32.NewProc("GetProcessId")
)

// winPIDFromHandle returns the Windows PID for a process HANDLE.
func winPIDFromHandle(h syscall.Handle) (int, error) {
	pid, _, err := procGetProcID.Call(uintptr(h))
	if pid == 0 {
		return 0, fmt.Errorf("GetProcessId: %w", err)
	}
	return int(pid), nil
}

// openNUL opens the Windows NUL device (equivalent to /dev/null) for reading.
func openNUL() (syscall.Handle, error) {
	p, _ := syscall.UTF16PtrFromString("NUL")
	return syscall.CreateFile(p,
		syscall.GENERIC_READ, syscall.FILE_SHARE_READ,
		nil, syscall.OPEN_EXISTING, 0, 0)
}

// ─── WSL spawn ────────────────────────────────────────────────────────────────

// wslLinuxSnapshotBase is the snapshot store path inside WSL2's native filesystem.
const wslLinuxSnapshotBase = "/tmp/relay-native/snapshots"

// wslSnapshotExists returns true if the named snapshot has been synced into
// WSL2's own snapshot store (/tmp/relay-native/snapshots/<name>).
func wslSnapshotExists(distro, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	out, err := wslRun(distro, "test -d "+shq(wslLinuxSnapshotBase+"/"+name)+" && echo yes || echo no")
	return err == nil && strings.TrimSpace(out) == "yes"
}

// doSpawnWSL runs a container inside WSL2 using the Linux station binary.
// The Linux binary uses CLONE_NEWPID + CLONE_NEWNS + chroot for real isolation.
//
// Port forwarding: WSL2 automatically forwards ports bound inside the VM to
// localhost on Windows, so the app is reachable at 127.0.0.1:<port> from
// Windows browsers/tools without any extra configuration.
func doSpawnWSL(distro string, rec *ContainerRecord, foreground bool, logWriter io.Writer) (int, error) {
	// First run: compile and install the Linux station binary inside WSL2.
	if err := wslEnsureStation(distro); err != nil {
		return 0, err
	}

	// Build the Linux station command.
	// We call run-fg so the WSL2 process IS the container (direct stdio or
	// redirected to the Windows log file). No daemon needed.
	wslDir := toWSLPath(rec.Dir)

	// If the snapshot was synced to WSL2's own store, let the Linux vessel
	// materialise an overlayfs on native ext4 instead of chrooting into
	// /mnt/c/ (WSL2 9P). Node.js startup from ext4 is 10-30× faster.
	useLinuxImage := rec.Image != "" && wslSnapshotExists(distro, rec.Image)
	if useLinuxImage {
		wslDir = "" // Linux station uses --image; no Windows path needed
	}

	cmd := buildWSLCmd(rec, wslDir, useLinuxImage)

	// Bridge mode requires root inside WSL2 for ip/iptables commands.
	// Pre-allocate the container IP via the Linux station so the Windows-side
	// ContainerRecord reflects the actual 10.88.x.x address. The Linux vessel
	// will return the same IP when cmdRun calls prepareNetworkEnv (idempotent).
	if rec.NetMode == "bridge" {
		if ipOut, err := wslRun(distro, fmt.Sprintf("%s _net-alloc %s", wslStationBin, shq(containerNetworkKey(rec.App, rec.ID)))); err == nil {
			if ip := strings.TrimSpace(ipOut); ip != "" {
				rec.IP = ip
			}
		}
		if foreground {
			c := exec.Command("wsl.exe", "-d", distro, "-u", "root", "--", "sh", "-c", cmd)
			setCmdHideWindow(c)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return 0, c.Run()
		}
		return wslSubprocessRoot(distro, cmd, logWriter)
	}

	if foreground {
		// Interactive: WslLaunchInteractive attaches the current console to
		// the WSL2 process — Ctrl+C, terminal size, colour codes all work.
		return 0, wslLaunchFg(distro, cmd)
	}

	// Detached: redirect WSL2 stdout/stderr to the Windows log file so
	// station logs <id> works across the WSL/Windows boundary.
	// WslLaunch has been unreliable for detached long-running Node/Next
	// workloads with redirected stdio on Windows, leading to immediate exit
	// loops. The hidden wsl.exe subprocess path is slower but stable.
	return wslSubprocess(distro, cmd, logWriter)
}

// wslSubprocessRoot launches the Linux command as root (uid=0) inside WSL2.
// Required for bridge networking (ip link, iptables) and other root-only ops.
func wslSubprocessRoot(distro, linuxCmd string, logWriter io.Writer) (int, error) {
	cmd := exec.Command("wsl.exe", "-d", distro, "-u", "root", "--", "sh", "-c", linuxCmd)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	go cmd.Wait()
	return cmd.Process.Pid, nil
}

// buildWSLCmd assembles the station Linux command string to run inside WSL2.
// When useLinuxImage is true the snapshot has been synced to WSL2's own store
// and the Linux station will materialise an overlayfs there; no Windows path or
// explicit command is needed (the image manifest supplies both).
func buildWSLCmd(rec *ContainerRecord, wslDir string, useLinuxImage bool) string {
	var b strings.Builder
	b.WriteString(wslStationBin)
	// WSL process IS the container; Windows redirects output via logWriter.
	b.WriteString(" run-fg")
	if rec.App != "" {
		b.WriteString(" --app ")
		b.WriteString(shq(rec.App))
	}
	// Always pin the port so the Linux station doesn't allocate a different one.
	if rec.Port > 0 {
		b.WriteString(" --port ")
		b.WriteString(strconv.Itoa(rec.Port))
	}
	if rec.RestartPolicy != "" {
		b.WriteString(" --restart ")
		b.WriteString(rec.RestartPolicy)
	}
	if rec.ContainerCwd != "" {
		b.WriteString(" --workdir ")
		b.WriteString(shq(rec.ContainerCwd))
	}
	for _, v := range rec.Volumes {
		b.WriteString(" --volume ")
		b.WriteString(shq(wslVolumeSpec(v)))
	}
	for _, host := range rec.ExtraHosts {
		b.WriteString(" --add-host ")
		b.WriteString(shq(host))
	}
	if len(rec.UserEnv) > 0 {
		keys := make([]string, 0, len(rec.UserEnv))
		for k := range rec.UserEnv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(" --env ")
			b.WriteString(shq(k + "=" + rec.UserEnv[k]))
		}
	}
	if rec.NetMode != "" {
		b.WriteString(" --net ")
		b.WriteString(rec.NetMode)
	}
	if useLinuxImage {
		// Snapshot is in WSL2's own store; pass --image so the Linux vessel
		// creates an overlayfs on native ext4 (fast) and gets the default cmd
		// from the manifest — no /mnt/c/ path or explicit command needed.
		b.WriteString(" --image ")
		b.WriteString(shq(rec.Image))
	} else {
		// When the Windows side has already materialised the snapshot into rec.WorkDir
		// (which becomes wslDir), do NOT tell the Linux station to re-load from its
		// own snapshot store — on Linux /tmp and Windows %TEMP% are unrelated paths.
		// The materialized workdir content is already accessible at wslDir.
		if rec.Image != "" && rec.WorkDir == "" {
			b.WriteString(" --image ")
			b.WriteString(shq(rec.Image))
		}
		b.WriteString(" ")
		b.WriteString(shq(wslDir))
		for _, arg := range rec.Command {
			b.WriteString(" ")
			b.WriteString(shq(arg))
		}
	}
	return b.String()
}

// wslSubprocess is the fallback when WslLaunch is unavailable: start wsl.exe
// as a child process of vessel. Less direct but still runs Linux.
func wslSubprocess(distro, linuxCmd string, logWriter io.Writer) (int, error) {
	cmd := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c", linuxCmd)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	go cmd.Wait()
	return cmd.Process.Pid, nil
}

// ─── WSL rootfs setup ─────────────────────────────────────────────────────────

// wslWindowsArch returns the Alpine architecture string matching the running
// Windows host: "x86_64" for AMD64, "aarch64" for ARM64.
func wslWindowsArch() string {
	if strings.EqualFold(os.Getenv("PROCESSOR_ARCHITECTURE"), "ARM64") {
		return "aarch64"
	}
	return "x86_64"
}

// wslAlpineRootfsURL resolves the URL for the latest Alpine minirootfs tarball
// for the current host architecture. It fetches latest-releases.yaml from the
// Alpine CDN; on failure it falls back to a pinned stable version.
func wslAlpineRootfsURL() (string, error) {
	arch := wslWindowsArch()
	base := "https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/" + arch + "/"
	resp, err := http.Get(base + "latest-releases.yaml") // #nosec G107
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "file: alpine-minirootfs-") && strings.HasSuffix(line, arch+".tar.gz") {
				return base + strings.TrimPrefix(line, "file: "), nil
			}
		}
	}
	// Static fallback pinned to a known good Alpine 3.21 release.
	return "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/" + arch +
		"/alpine-minirootfs-3.21.3-" + arch + ".tar.gz", nil
}

// wslDownloadFile downloads url to dest on the Windows filesystem.
func wslDownloadFile(url, dest string) error {
	resp, err := http.Get(url) // #nosec G107 — caller-controlled CDN URL
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// wslProvisionStationDistro registers a minimal Alpine WSL2 distro named
// "station-linux". No pre-existing Linux installation is needed — the Alpine
// minirootfs (~5 MB) is downloaded via Windows HTTP and imported with
// wsl.exe --import. Subsequent calls are no-ops if already registered.
func wslProvisionStationDistro() error {
	if wslIsRegistered(stationLinuxDistro) {
		return nil
	}

	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	stateDir := filepath.Join(appData, "station", "distros", stationLinuxDistro)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("create station distro dir: %w", err)
	}

	url, err := wslAlpineRootfsURL()
	if err != nil {
		return fmt.Errorf("resolve Alpine rootfs URL: %w", err)
	}
	tarPath := filepath.Join(stateDir, "rootfs.tar.gz")
	fmt.Printf("[station] downloading Alpine minirootfs from %s ...\n", url)
	if err := wslDownloadFile(url, tarPath); err != nil {
		return fmt.Errorf("download Alpine rootfs: %w", err)
	}

	fmt.Printf("[station] importing %s into %s ...\n", stationLinuxDistro, stateDir)
	cmd := exec.Command("wsl.exe", "--import", stationLinuxDistro, stateDir, tarPath, "--version", "2")
	setCmdHideWindow(cmd)
	out, err := cmd.CombinedOutput()
	_ = os.Remove(tarPath) // cleanup regardless of outcome
	if err != nil {
		return fmt.Errorf("wsl --import %s: %w\n%s", stationLinuxDistro, err, strings.TrimSpace(string(out)))
	}

	fmt.Printf("[station] %s ready\n", stationLinuxDistro)
	return nil
}

// wslSetupRootfs delegates the Alpine download to the Linux station inside WSL2.
// The rootfs is created inside the WSL2 filesystem (fast, native Linux paths).
func wslSetupRootfs(distro, dest string) error {
	if err := wslEnsureStation(distro); err != nil {
		return err
	}
	wslDest := dest
	// If dest looks like a Windows path, keep it in the WSL filesystem instead.
	if !strings.HasPrefix(dest, "/") {
		wslDest = "/home/" + distro + "/rootfs"
	}
	fmt.Printf("[station] downloading Alpine rootfs inside WSL2 → %s\n", wslDest)
	cmd := fmt.Sprintf("%s setup-rootfs '%s'", wslStationBin, wslDest)
	out, err := wslRunCombined(distro, cmd)
	fmt.Print(out)
	if err != nil {
		return fmt.Errorf("setup-rootfs in WSL2: %w", err)
	}
	fmt.Printf("\nRootfs ready at WSL2 path: %s\n", wslDest)
	fmt.Printf("Run: station run-fg %s /bin/sh\n", wslDest)
	return nil
}

// ─── WSL keepalive ───────────────────────────────────────────────────────────
//
// The WSL2 Hyper-V VM shuts down when every process inside every running
// distro exits.  On a cold system (fresh boot, or idle for hours) the first
// wsl.exe call pays a 1-3 s VM boot penalty — exactly the lag the user sees.
//
// Docker Desktop avoids this by keeping a persistent process in its distro
// 24/7 via its Windows tray app.  We do the same: after any successful spawn
// we fire a detached "sleep 86400" inside the distro.  That orphaned process
// keeps the VM hot for 24 hours; each subsequent station run refreshes it.
//
// For the very first run after a cold boot, add "station wsl-warm" to Windows
// Task Scheduler → "At log on" so the distro is already running by the time
// you use station (same trick Docker Desktop uses with its startup service).

// wslKeepaliveScript is idempotent: starts a 24 h sleep only if none is alive.
const wslKeepaliveScript = `pid=$(cat /tmp/.station-keepalive.pid 2>/dev/null); ` +
	`[ -n "$pid" ] && kill -0 "$pid" 2>/dev/null && exit 0; ` +
	`sleep 86400 & echo $! > /tmp/.station-keepalive.pid`

// wslEnsureKeepalive fires a detached keepalive in distro (async, safe to
// call in a goroutine).  The sleep outlives the station process itself, keeping
// the WSL VM hot for subsequent calls.
func wslEnsureKeepalive(distro string) {
	c := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c", wslKeepaliveScript)
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow}
	_ = c.Start()
	go c.Wait()
}

// stationTaskName is the Windows Task Scheduler task name for the wsl-warm startup job.
const stationTaskName = "station-wsl-warm"

// isWindowsAdmin returns true when the current process is running elevated
// (member of the built-in Administrators group, S-1-5-32-544).
// Uses raw advapi32.dll calls to avoid importing golang.org/x/sys/windows.
func isWindowsAdmin() bool {
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	allocSid := advapi32.NewProc("AllocateAndInitializeSid")
	freeSid := advapi32.NewProc("FreeSid")
	checkMembership := advapi32.NewProc("CheckTokenMembership")

	// SidIdentifierAuthority for SECURITY_NT_AUTHORITY = {0,0,0,0,0,5}
	auth := [6]byte{0, 0, 0, 0, 0, 5}
	var sid uintptr
	r, _, _ := allocSid.Call(
		uintptr(unsafe.Pointer(&auth)),
		2,     // SubAuthorityCount
		0x20,  // SECURITY_BUILTIN_DOMAIN_RID
		0x220, // DOMAIN_ALIAS_RID_ADMINS
		0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&sid)),
	)
	if r == 0 || sid == 0 {
		return false
	}
	defer freeSid.Call(sid)

	var isMember uint32
	r, _, _ = checkMembership.Call(0, sid, uintptr(unsafe.Pointer(&isMember)))
	return r != 0 && isMember != 0
}

// wslStartupTaskExists returns true if the Task Scheduler entry already exists.
func wslStartupTaskExists() bool {
	return exec.Command("schtasks", "/query", "/tn", stationTaskName).Run() == nil
}

// wslRegisterStartupTask creates a Task Scheduler entry that runs
// "station wsl-warm" at every log on for the current user.
func wslRegisterStartupTask(exePath string) error {
	out, err := exec.Command("schtasks", "/create",
		"/tn", stationTaskName,
		"/tr", `"`+exePath+`" wsl-warm`,
		"/sc", "onlogon",
		"/ru", os.Getenv("USERNAME"),
		"/f",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// wslWarm synchronously warms distro, starts the keepalive, and registers a
// Windows startup task the first time so it runs automatically at every log on.
func wslWarm(distro string) {
	fmt.Printf("[station] warming WSL2 distro %q...\n", distro)
	if _, err := wslRun(distro, wslKeepaliveScript); err != nil {
		fmt.Fprintf(os.Stderr, "[station] warn: keepalive: %v\n", err)
		return
	}
	fmt.Printf("[station] WSL2 distro %q is warm — keepalive running for 24 h.\n", distro)

	if wslStartupTaskExists() {
		fmt.Printf("[station] startup task %q is already registered.\n", stationTaskName)
		return
	}

	if !isWindowsAdmin() {
		fmt.Println("[station] To register the startup task, re-run as Administrator:")
		fmt.Println("         Right-click Command Prompt → Run as administrator, then: station wsl-warm")
		return
	}

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[station] warn: resolve executable: %v\n", err)
		return
	}
	if err := wslRegisterStartupTask(self); err != nil {
		fmt.Fprintf(os.Stderr, "[station] warn: could not register startup task: %v\n", err)
		return
	}
	fmt.Printf("[station] registered startup task %q — WSL2 warms automatically at log on.\n", stationTaskName)
}

// ─── utilities ────────────────────────────────────────────────────────────────

// wslRunTimeout is the deadline for short WSL probe commands (liveness checks,
// hash reads, snapshot tests).  Long operations (builds, installs) use
// wslRunCombined which has no timeout.
const wslRunTimeout = 30 * time.Second

// wslRun runs a shell command inside distro and returns stdout.
// Times out after wslRunTimeout so a stuck 9P mount never hangs the caller.
func wslRun(distro, cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), wslRunTimeout)
	defer cancel()
	c := exec.CommandContext(ctx, "wsl.exe", "-d", distro, "--", "sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow}
	out, err := c.Output()
	return stripNulls(out), err
}

// wslRunCombined runs a shell command and returns combined stdout+stderr.
// No timeout — used for long-running operations (builds, package installs).
func wslRunCombined(distro, cmd string) (string, error) {
	c := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow}
	out, err := c.CombinedOutput()
	return stripNulls(out), err
}

// nullStripWriter wraps an io.Writer, stripping null bytes and carriage
// returns that wsl.exe emits when it outputs UTF-16 LE on some Windows builds.
type nullStripWriter struct{ w io.Writer }

func (n nullStripWriter) Write(p []byte) (int, error) {
	filtered := make([]byte, 0, len(p))
	for _, b := range p {
		if b != 0x00 && b != 0x0D {
			filtered = append(filtered, b)
		}
	}
	if len(filtered) == 0 {
		return len(p), nil
	}
	_, err := n.w.Write(filtered)
	return len(p), err
}

// wslRunStreaming runs a shell command inside distro, streaming combined
// stdout+stderr directly to w in real-time as each line is produced.
// No timeout — used for long-running operations (builds, package installs).
func wslRunStreaming(distro, cmd string, w io.Writer) error {
	c := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow}
	nsw := nullStripWriter{w: w}
	c.Stdout = nsw
	c.Stderr = nsw
	return c.Run()
}

// stripNulls removes null bytes and carriage returns emitted by wsl.exe output
// (wsl.exe sometimes produces UTF-16 LE; stripping nulls recovers ASCII).
func stripNulls(b []byte) string {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c != 0x00 && c != 0x0D {
			out = append(out, c)
		}
	}
	return string(out)
}

// shq wraps a string in single quotes for safe shell interpolation.
func shq(s string) string {
	if !strings.ContainsAny(s, " \t\"'\\(){}$`") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
