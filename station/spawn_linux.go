//go:build linux

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// initMagic is passed as os.Args[1] when the binary re-execs itself inside
// the new namespaces. The child detects it and jumps to containerInit().
const initMagic = "__station_init__"

const alpineURL = "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz"

func platformName() string            { return "linux (namespaces + chroot)" }
func platformSetupRootfsHint() string { return "Download Alpine Linux mini rootfs" }

// platformSkipImagePrep always returns false on Linux — prepareImageRootfs
// is needed to create the overlayfs mount from the local snapshot store.
func platformSkipImagePrep(_ string) bool { return false }

// resolveDir is a pass-through on Linux — dir is always the chroot rootfs.
func resolveDir(dir string, cmdArgs []string) (string, []string) {
	return dir, cmdArgs
}

// isContainerInit returns true when this process is the re-exec'd child that
// should run inside the container namespaces.
func isContainerInit() bool {
	return len(os.Args) > 1 && os.Args[1] == initMagic
}

func runContainerInit() { containerInit() }

// pidAlive uses kill(pid, 0) — no signal sent, just checks existence.
func pidAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// killProcess sends SIGTERM then waits up to 4s before SIGKILL.
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

// doSpawn starts the container. On Linux it re-execs the binary inside fresh
// namespaces (PID + mount + UTS + IPC + optionally user or net). Returns the
// host PID on success.
//
// foreground=true: blocks until the process exits (interactive use).
// foreground=false: starts detached, returns PID immediately.
func doSpawn(rec *ContainerRecord, foreground bool, logWriter io.Writer) (int, error) {
	self, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("resolve executable: %w", err)
	}

	// Re-exec: station __station_init__ <dir> <cmd> [args...]
	reArgs := append([]string{initMagic, rec.Dir}, rec.Command...)
	child := exec.Command(self, reArgs...)
	child.Env = buildEnv(rec.Env)

	var netSyncWrite *os.File // write end of the bridge-networking sync pipe

	if rec.NetMode == "bridge" {
		// Bridge networking requires root; we skip CLONE_NEWUSER so the
		// process runs with real root capabilities in the host user ns.
		// containerInit will read one byte from fd 3 before configuring eth0.
		pr, pw, err := os.Pipe()
		if err != nil {
			return 0, fmt.Errorf("create network sync pipe: %w", err)
		}
		netSyncWrite = pw
		child.ExtraFiles = []*os.File{pr}
		child.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWIPC |
				syscall.CLONE_NEWNET,
		}
	} else {
		// Default rootless mode: map the caller's UID to root inside the
		// container. No real root needed on the host.
		child.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWIPC |
				syscall.CLONE_NEWUSER, // rootless: caller UID → root inside
			UidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getuid(), Size: 1},
			},
			GidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getgid(), Size: 1},
			},
		}
	}

	if foreground {
		// For foreground + bridge we must use Start/Wait so we can set up
		// networking between Start and the process running its first command.
		if rec.NetMode == "bridge" {
			child.Stdin = os.Stdin
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			if err := child.Start(); err != nil {
				netSyncWrite.Close()
				child.ExtraFiles[0].Close()
				return 0, err
			}
			child.ExtraFiles[0].Close()
			if err := setupContainerNetwork(rec, child.Process.Pid); err != nil {
				netSyncWrite.Close()
				_ = child.Process.Kill()
				return 0, fmt.Errorf("network setup: %w", err)
			}
			_, _ = netSyncWrite.Write([]byte{1})
			netSyncWrite.Close()
			return 0, child.Wait()
		}
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		return 0, child.Run()
	}

	child.Stdout = logWriter
	child.Stderr = logWriter
	if err := child.Start(); err != nil {
		if netSyncWrite != nil {
			netSyncWrite.Close()
		}
		if len(child.ExtraFiles) > 0 {
			child.ExtraFiles[0].Close()
		}
		return 0, err
	}
	// Close the child's (read) end of the sync pipe in the parent process.
	if len(child.ExtraFiles) > 0 {
		child.ExtraFiles[0].Close()
	}

	if rec.NetMode == "bridge" {
		// Set up veth pair and move the peer into the container's netns.
		// Then write one byte to unblock containerInit.
		if err := setupContainerNetwork(rec, child.Process.Pid); err != nil {
			_, _ = netSyncWrite.Write([]byte{0}) // unblock child so it exits cleanly
			netSyncWrite.Close()
			_ = child.Process.Kill()
			return 0, fmt.Errorf("network setup: %w", err)
		}
		_, _ = netSyncWrite.Write([]byte{1})
		netSyncWrite.Close()
	}

	go child.Wait() // reap the zombie asynchronously
	return child.Process.Pid, nil
}

// containerInit runs inside the new namespaces after the re-exec. It:
//  1. Sets hostname
//  2. Mounts /proc and /dev
//  3. Configures bridge networking (if CONTAINER_NET_MODE=bridge)
//  4. Writes /etc/hosts aliases and bind-mounts volumes
//  5. Chroots into the rootfs
//  6. Exec's the requested command
//
// This function must never return on success.
func containerInit() {
	// Args: station __station_init__ <rootfs> <cmd> [args...]
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "container init: malformed args")
		os.Exit(1)
	}
	rootfs := os.Args[2]
	cmdArgs := os.Args[3:]
	id := getenv("CONTAINER_ID", "container")

	// 1. Hostname — UTS namespace is fresh so this won't affect the host.
	_ = syscall.Sethostname([]byte(id))

	// 2. Mount /proc — the new PID namespace needs its own procfs.
	procDir := filepath.Join(rootfs, "proc")
	_ = os.MkdirAll(procDir, 0555)
	if err := syscall.Mount("proc", procDir, "proc", 0, ""); err != nil {
		fmt.Fprintf(os.Stderr, "warn: mount proc: %v\n", err)
	}

	// 3. Mount /dev — try devtmpfs, fall back to a bind-mount of the host /dev.
	devDir := filepath.Join(rootfs, "dev")
	_ = os.MkdirAll(devDir, 0755)
	if err := syscall.Mount("devtmpfs", devDir, "devtmpfs", syscall.MS_NOSUID, "mode=0755"); err != nil {
		_ = syscall.Mount("/dev", devDir, "", syscall.MS_BIND|syscall.MS_REC, "")
	}

	// 4. Bridge networking: wait for the parent to move the veth peer into
	//    this network namespace, then configure eth0 before chrootting.
	if os.Getenv("CONTAINER_NET_MODE") == "bridge" {
		containerConfigureNetwork(rootfs)
	}

	if err := writeContainerHosts(rootfs); err != nil {
		fmt.Fprintf(os.Stderr, "warn: write hosts: %v\n", err)
	}

	// 5. Volume mounts: bind host directories into the rootfs before chroot.
	//    Format: CONTAINER_VOLUMES=hostpath:containerpath,hostpath:containerpath
	if vols := os.Getenv("CONTAINER_VOLUMES"); vols != "" {
		for _, vol := range strings.Split(vols, ",") {
			source, targetPath, _, ok := parseVolumeSpec(vol)
			if !ok || source == "" || targetPath == "" {
				continue
			}
			parts := []string{source, targetPath}
			target := filepath.Join(rootfs, filepath.Clean("/"+targetPath))
			_ = os.MkdirAll(target, 0755)
			if err := syscall.Mount(source, target, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
				fmt.Fprintf(os.Stderr, "warn: volume %s→%s: %v\n", parts[0], parts[1], err)
			}
		}
	}

	// 6. Chroot into the rootfs directory.
	if err := syscall.Chroot(rootfs); err != nil {
		fmt.Fprintf(os.Stderr, "chroot %q: %v\n", rootfs, err)
		os.Exit(1)
	}
	if err := syscall.Chdir("/"); err != nil {
		fmt.Fprintf(os.Stderr, "chdir /: %v\n", err)
		os.Exit(1)
	}
	if cwd := getenv("CONTAINER_WORKDIR", ""); cwd != "" {
		if !strings.HasPrefix(cwd, "/") {
			cwd = "/" + cwd
		}
		_ = os.MkdirAll(cwd, 0755)
		if err := syscall.Chdir(cwd); err != nil {
			fmt.Fprintf(os.Stderr, "chdir %q: %v\n", cwd, err)
			os.Exit(1)
		}
	}

	// 7. Resolve the binary path inside the chroot.
	binary := resolveBinary(cmdArgs[0])

	// 8. Build a minimal, clean environment.
	env := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/root",
		"TERM=" + getenv("TERM", "xterm"),
		"CONTAINER_ID=" + id,
		"PORT=" + getenv("PORT", ""),
	}
	if forward := getenv("CONTAINER_FORWARD_ENV", ""); forward != "" {
		for _, key := range strings.Split(forward, ",") {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if val, ok := os.LookupEnv(key); ok {
				env = append(env, key+"="+val)
			}
		}
	}

	// 9. Replace this process image — never returns on success.
	if err := syscall.Exec(binary, cmdArgs, env); err != nil {
		fmt.Fprintf(os.Stderr, "exec %q: %v\n", binary, err)
		os.Exit(1)
	}
}

// containerConfigureNetwork waits for the parent to finish bridge setup.
// The parent process now moves and configures the peer via nsenter because
// WSL kernels may ignore the requested peer name for veth pairs.
func containerConfigureNetwork(rootfs string) {
	// Block until the parent signals that the veth peer has been moved in.
	syncFd := os.NewFile(uintptr(3), "net-sync")
	buf := make([]byte, 1)
	_, _ = syncFd.Read(buf)
	syncFd.Close()

	// Write /etc/resolv.conf into the rootfs for DNS inside the container.
	_ = os.MkdirAll(filepath.Join(rootfs, "etc"), 0755)
	_ = os.WriteFile(filepath.Join(rootfs, "etc", "resolv.conf"),
		[]byte("nameserver 8.8.8.8\nnameserver 8.8.4.4\n"), 0644)
}

// findHostBin returns the first path that exists for a given binary name,
// searching standard host locations (pre-chroot, so host FS is active).
func findHostBin(name string) string {
	for _, dir := range []string{"/sbin", "/usr/sbin", "/bin", "/usr/bin"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return name
}

func resolveBinary(name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}
	if found, err := exec.LookPath(name); err == nil {
		return found
	}
	for _, dir := range []string{"/bin", "/usr/bin", "/sbin", "/usr/sbin"} {
		p := dir + "/" + name
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return name
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func writeContainerHosts(rootfs string) error {
	lines := []string{
		"127.0.0.1 localhost",
		"::1 localhost ip6-localhost ip6-loopback",
	}
	id := strings.TrimSpace(getenv("CONTAINER_ID", "container"))
	appName := strings.TrimSpace(getenv("APP_NAME", ""))
	if ip := strings.TrimSpace(getenv("CONTAINER_NET_IP", "")); ip != "" {
		names := []string{id}
		if appName != "" && appName != id {
			names = append([]string{appName}, names...)
		}
		lines = append(lines, fmt.Sprintf("%s %s", ip, strings.Join(names, " ")))
	}
	for _, raw := range strings.Split(getenv("CONTAINER_EXTRA_HOSTS", ""), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		ip := strings.TrimSpace(parts[1])
		if name == "" || ip == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s", ip, name))
	}
	etcDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(etcDir, "hosts"), []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// platformWslWarm is a no-op on Linux; WSL2 keepalive is Windows-only.
func platformWslWarm() {
	fmt.Fprintln(os.Stderr, "wsl-warm is a Windows-only command (WSL2 VM warm-up)")
}

// platformWslEnsure is a no-op on Linux; station already runs natively here.
func platformWslEnsure() {
	fmt.Fprintln(os.Stderr, "wsl-ensure is a Windows-only command")
}

// platformSetupRootfs downloads the Alpine Linux mini rootfs (~3 MB gzipped).
// Alpine is a full BusyBox + apk environment and makes a good default rootfs.
func platformSetupRootfs(dest string) error {
	fmt.Printf("downloading Alpine Linux mini rootfs → %s/\n", dest)
	fmt.Printf("url: %s\n\n", alpineURL)

	resp, err := http.Get(alpineURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	destAbs, _ := filepath.Abs(dest)
	count := extractTar(tar.NewReader(gr), destAbs)

	for _, d := range []string{"proc", "sys", "dev", "tmp", "run"} {
		_ = os.MkdirAll(filepath.Join(dest, d), 0755)
	}

	fmt.Printf("extracted %d files\n\n", count)
	fmt.Println("next steps:")
	fmt.Printf("  ./station run-fg %s /bin/sh\n", dest)
	fmt.Printf("  ./station run    %s /bin/sleep 30\n", dest)
	return nil
}

func extractTar(tr *tar.Reader, destAbs string) int {
	count := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		target := filepath.Join(destAbs, filepath.Clean("/"+hdr.Name))
		if !strings.HasPrefix(target, destAbs+string(filepath.Separator)) && target != destAbs {
			continue // path escape guard
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, hdr.FileInfo().Mode())
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if err == nil {
				_, _ = io.Copy(f, tr)
				f.Close()
				count++
			}
		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Remove(target)
			_ = os.Symlink(hdr.Linkname, target)
		case tar.TypeLink:
			old := filepath.Join(destAbs, filepath.Clean("/"+hdr.Linkname))
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Link(old, target)
		}
	}
	return count
}

