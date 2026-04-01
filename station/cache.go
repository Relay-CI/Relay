package main

// Image snapshot system — hardlink-based copy-on-write.
//
// Without overlayfs, we use hard links: both the snapshot and the working
// copy point to the same inodes. A write inside the container breaks the
// link for that file (new inode) while non-written files cost zero extra
// disk space. The original snapshot stays clean.
//
// Commands:
//   station snapshot save  <name> <rootfs-dir>  — commit rootfs as an image
//   station snapshot load  <name> <dest-dir>    — materialize a working copy
//   station snapshot list                       — list saved images
//   station snapshot delete <name>              — remove an image
//
// Integration with run:
//   station run --image <name> <cmd> [args...]  — run from a fresh snapshot
//   copy. The working copy is removed when the container is stopped.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── paths ────────────────────────────────────────────────────────────────────

func snapshotStore() string {
	return filepath.Join(stateBaseDir(), "snapshots")
}

func snapshotPath(name string) string {
	return filepath.Join(snapshotStore(), name)
}

func workdirPath(containerID string) string {
	return filepath.Join(stateBaseDir(), "workdirs", containerID)
}

// ─── hardlink copy ────────────────────────────────────────────────────────────

// hardlinkCopy recursively copies src to dst, using hard links for regular
// files so unchanged data costs zero extra disk space. New directories are
// created with the same permissions as the source. Symlinks are recreated.
func hardlinkCopy(src, dst string) error {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	return filepath.Walk(srcAbs, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// Skip unreadable entries rather than aborting the entire copy.
			return nil
		}
		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		// Symlink — recreate with the same target.
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return nil // skip dangling symlinks
			}
			_ = os.Remove(target) // remove stale entry
			return os.Symlink(link, target)
		}

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		// Skip special files (devices, sockets, named pipes, etc.) — they
		// cannot be meaningfully copied and opening them may return errors
		// (e.g. /dev/autofs returns "invalid argument" on read).
		if !info.Mode().IsRegular() {
			return nil
		}

		// Regular file: hard-link to reuse the inode.
		_ = os.Remove(target) // unlink stale copy first
		if err := os.Link(path, target); err != nil {
			// Hard links across filesystems fail; fall back to data copy.
			return copyFile(path, target, info.Mode())
		}
		return nil
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ─── commands ─────────────────────────────────────────────────────────────────

func cmdSnapshotSave(name, srcDir string) {
	if strings.ContainsAny(name, "/\\:") {
		die("snapshot name must not contain path separators")
	}
	absDir := mustAbs(srcDir)
	if _, err := os.Stat(absDir); err != nil {
		die("dir %q: %v", absDir, err)
	}
	dest := snapshotPath(name)
	if _, err := os.Stat(dest); err == nil {
		die("snapshot %q already exists — delete it first: station snapshot delete %s", name, name)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		die("create snapshot store: %v", err)
	}
	fmt.Printf("saving snapshot %q from %s ...\n", name, absDir)
	start := time.Now()
	if err := hardlinkCopy(absDir, dest); err != nil {
		die("snapshot: %v", err)
	}
	fmt.Printf("saved in %s  (%s)\n", time.Since(start).Round(time.Millisecond), dest)
}

func cmdSnapshotLoad(name, destDir string) {
	src := snapshotPath(name)
	if _, err := os.Stat(src); err != nil {
		die("snapshot %q not found", name)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		die("mkdir %q: %v", destDir, err)
	}
	fmt.Printf("loading snapshot %q → %s ...\n", name, destDir)
	start := time.Now()
	if err := hardlinkCopy(src, destDir); err != nil {
		die("load: %v", err)
	}
	fmt.Printf("done in %s\n", time.Since(start).Round(time.Millisecond))
}

func cmdSnapshotList() {
	entries, err := os.ReadDir(snapshotStore())
	if err != nil {
		fmt.Println("no snapshots")
		return
	}
	if len(entries) == 0 {
		fmt.Println("no snapshots")
		return
	}
	fmt.Printf("%-20s  %s\n", "NAME", "PATH")
	for _, e := range entries {
		if e.IsDir() {
			fmt.Printf("%-20s  %s\n", e.Name(), snapshotPath(e.Name()))
		}
	}
}

func cmdSnapshotDelete(name string) {
	dest := snapshotPath(name)
	if _, err := os.Stat(dest); err != nil {
		die("snapshot %q not found", name)
	}
	if err := os.RemoveAll(dest); err != nil {
		die("delete: %v", err)
	}
	fmt.Printf("snapshot %q deleted\n", name)
}

// prepareImageWorkdir creates a fresh hardlink copy of a named snapshot for
// use as a container rootfs. The container record stores the workdir path so
// cmdStop can clean it up.
func prepareImageWorkdir(image, containerID string) (string, error) {
	src, _, err := resolveImageRootfs(image)
	if err != nil {
		return "", err
	}
	dest := workdirPath(containerID)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return "", fmt.Errorf("create workdir: %w", err)
	}
	if err := hardlinkCopy(src, dest); err != nil {
		_ = os.RemoveAll(dest)
		return "", fmt.Errorf("hardlink copy: %w", err)
	}
	return dest, nil
}
