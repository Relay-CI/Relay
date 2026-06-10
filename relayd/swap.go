// swap.go — opt-in swapfile provisioning for small hosts.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// maybeEnableAutoSwap creates and enables a swapfile at startup when
// RELAY_AUTO_SWAP=1, the daemon runs as root on Linux, and the host has no
// swap configured. Swap turns "the kernel killed the build" into "the build
// ran a little slower" — on 2 GB hosts that is the difference between OOM
// kills being rare and being impossible. relayd re-runs this on every boot
// (via systemd), so no fstab entry is needed.
func maybeEnableAutoSwap() {
	if !getenvBool("RELAY_AUTO_SWAP", false) {
		return
	}
	if runtime.GOOS != "linux" {
		return
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "auto-swap: RELAY_AUTO_SWAP=1 ignored — relayd is not running as root")
		return
	}
	if hostSwapTotalMB() > 0 {
		return
	}
	sizeMB := 2048
	if v := strings.TrimSpace(os.Getenv("RELAY_AUTO_SWAP_MB")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			sizeMB = n
		}
	}
	path := getenv("RELAY_AUTO_SWAP_PATH", "/swapfile")
	if err := enableSwapfile(path, sizeMB); err != nil {
		fmt.Fprintf(os.Stderr, "auto-swap: %v\n", err)
		return
	}
	fmt.Printf("auto-swap: enabled %d MB swapfile at %s\n", sizeMB, path)
}

func enableSwapfile(path string, sizeMB int) error {
	if _, err := os.Stat(path); err == nil {
		// A file already exists at the swap path. Only try to activate it —
		// never mkswap a file we did not create, since that would destroy
		// whatever it currently holds.
		if out, err := exec.Command("swapon", path).CombinedOutput(); err != nil {
			return fmt.Errorf("%s exists but could not be activated (%v — %s); remove it or point RELAY_AUTO_SWAP_PATH elsewhere", path, err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	// fallocate is instant; dd is the fallback for filesystems without it.
	if out, err := exec.Command("fallocate", "-l", fmt.Sprintf("%dM", sizeMB), path).CombinedOutput(); err != nil {
		if out2, err2 := exec.Command("dd", "if=/dev/zero", "of="+path, "bs=1M", fmt.Sprintf("count=%d", sizeMB)).CombinedOutput(); err2 != nil {
			_ = os.Remove(path)
			return fmt.Errorf("create %s: fallocate: %v (%s); dd: %v (%s)", path, err, strings.TrimSpace(string(out)), err2, strings.TrimSpace(string(out2)))
		}
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	if out, err := exec.Command("mkswap", path).CombinedOutput(); err != nil {
		return fmt.Errorf("mkswap %s: %v — %s", path, err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("swapon", path).CombinedOutput(); err != nil {
		return fmt.Errorf("swapon %s: %v — %s", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}
