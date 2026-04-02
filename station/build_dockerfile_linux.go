//go:build linux

package main

// Dockerfile builder — Linux implementation.
//
// Executes each stage of a multi-stage Dockerfile using vessel's own
// namespace isolation:
//   - Each stage gets a fresh rootfs directory.
//   - FROM pulls the base image via oci.go.
//   - WORKDIR/ENV/COPY execute on the host (no namespace needed).
//   - RUN executes inside a chroot + mount namespace so install commands
//     (npm ci, pip install, go build) run isolated from the host.
//   - The final stage's rootfs becomes the container image.
//
// No Docker daemon required. Build speed is significantly faster because
// there are no image layers to hash, push, or pull after the first build.

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// BuildDockerfile executes the Dockerfile at dockerfilePath with build context
// contextDir. Writes the final rootfs to outDir and returns a BuildManifest
// (CMD, ENV, EXPOSE). outDir is created if it does not exist.
func BuildDockerfile(dockerfilePath, contextDir, outDir string, logf func(string, ...any), logw io.Writer) (*BuildManifest, error) {
	df, err := ParseDockerfile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("parse Dockerfile: %w", err)
	}
	if len(df.Stages) == 0 {
		return nil, fmt.Errorf("Dockerfile has no stages")
	}

	stageDirs := make(map[string]string)
	var finalManifest *BuildManifest

	ctx := stageCtx{
		outDir:      outDir,
		stageDirs:   stageDirs,
		stageHashes: make(map[string]string),
		contextDir:  contextDir,
		logf:        logf,
		logw:        logw,
	}

	for i, stage := range df.Stages {
		isFinal := i == len(df.Stages)-1
		m, stageDir, owned, err := buildStage(i, stage, isFinal, ctx)
		if err != nil {
			return nil, fmt.Errorf("stage %d (%s): %w", i, stage.Image, err)
		}
		stageDirs[fmt.Sprintf("%d", i)] = stageDir
		if stage.Name != "" {
			stageDirs[strings.ToLower(stage.Name)] = stageDir
		}
		finalManifest = m
		if !isFinal && owned {
			defer os.RemoveAll(stageDir) //nolint:gocritic
		}
	}

	if finalManifest == nil {
		finalManifest = &BuildManifest{Env: map[string]string{}}
	}
	if err := saveManifest(outDir, finalManifest); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}
	return finalManifest, nil
}

// stageCtx carries the shared context passed to every stage execution.
type stageCtx struct {
	outDir           string
	stageDirs        map[string]string
	stageHashes      map[string]string // stage name → cache key computed after build
	contextDir       string
	logf             func(string, ...any)
	logw             io.Writer
}

type stageWorkspace struct {
	rootfs   string // merged overlayfs dir, or plain dir when overlay unavailable
	upperDir string // overlayfs upper layer; empty when not overlay-backed
	baseDir  string // base rootfs used as the overlay lower dir; empty when not overlay-backed
	cleanup  func()
}

func (w *stageWorkspace) Close() {
	if w == nil || w.cleanup == nil {
		return
	}
	w.cleanup()
	w.cleanup = nil
}

// buildStage executes one Dockerfile stage and returns its manifest, rootfs
// dir, and whether the caller owns that directory and should remove it later.
func buildStage(idx int, stage DFStage, isFinal bool, ctx stageCtx) (*BuildManifest, string, bool, error) {
	as := ""
	if stage.Name != "" {
		as = " AS " + stage.Name
	}
	ctx.logf("[build] stage %d: FROM %s%s", idx, stage.Image, as)

	stageDir := ctx.outDir
	if !isFinal {
		stageDir = filepath.Join(os.TempDir(), fmt.Sprintf("station-stage-%s-%d", randID(), idx))
	}
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return nil, "", false, fmt.Errorf("mkdir: %w", err)
	}

	// ── stage cache: check for a prior build with identical inputs ─────────────
	var cacheKey string
	if !isFinal {
		if key, ok := stageInputHash(stage, ctx.contextDir, ctx.stageDirs, ctx.stageHashes); ok {
			cacheKey = key
			cacheDir := stageCacheDir(key)
			if fi, err := os.Stat(cacheDir); err == nil && fi.IsDir() {
				ctx.logf("[build] stage %d: cache hit", idx)
				if m, err := loadManifest(cacheDir); err == nil {
					if m == nil {
						m = &BuildManifest{Env: map[string]string{}}
					}
					_ = os.RemoveAll(stageDir)
					recordStageHash(stage, cacheKey, ctx.stageHashes)
					return m, cacheDir, false, nil
				}
				ctx.logf("[build] stage %d: cache restore failed, rebuilding", idx)
				_ = os.RemoveAll(stageDir)
				_ = os.MkdirAll(stageDir, 0755)
			}
		}
	}

	baseManifest, workspace, err := prepareStageRootfs(idx, stage, stageDir, ctx.stageDirs, ctx.logf)
	if err != nil {
		return nil, stageDir, true, err
	}
	defer workspace.Close()
	rootfs := workspace.rootfs

	m := manifestFromBase(baseManifest)
	workdir := firstNonEmptyStr(m.WorkDir, "/")

	for _, ins := range stage.Instructions {
		var execErr error
		workdir, execErr = execInstruction(ins, rootfs, workdir, m, ctx.stageDirs, ctx.contextDir, ctx.logw)
		if execErr != nil {
			return nil, stageDir, true, execErr
		}
	}
	if err := saveManifest(rootfs, m); err != nil {
		return nil, stageDir, true, fmt.Errorf("save stage manifest: %w", err)
	}

	// ── stage cache: persist this build for future runs ───────────────────────
	if !isFinal && cacheKey != "" {
		cacheDir := stageCacheDir(cacheKey)
		if mkErr := os.MkdirAll(filepath.Dir(cacheDir), 0755); mkErr == nil {
			if saveErr := saveStageCache(workspace, cacheDir, m, ctx.logf, idx); saveErr == nil {
				recordStageHash(stage, cacheKey, ctx.stageHashes)
				return m, cacheDir, false, nil
			}
		}
		recordStageHash(stage, cacheKey, ctx.stageHashes)
	}

	if err := materializeStageRootfs(rootfs, stageDir); err != nil {
		return nil, stageDir, true, fmt.Errorf("materialize stage rootfs: %w", err)
	}

	return m, stageDir, true, nil
}

// prepareStageRootfs resolves the FROM base and exposes it as a writable stage
// workspace. Overlayfs avoids copying the whole base rootfs before RUN.
func prepareStageRootfs(idx int, stage DFStage, stageDir string, stageDirs map[string]string, logf func(string, ...any)) (*BuildManifest, *stageWorkspace, error) {
	baseRootfs, baseManifest, err := resolveStageBase(idx, stage, stageDirs, logf)
	if err != nil {
		return nil, nil, err
	}
	if overlayRootfs, upperDir, cleanup, overlayErr := createBuildOverlayRootfs(baseRootfs); overlayErr == nil {
		return baseManifest, &stageWorkspace{rootfs: overlayRootfs, upperDir: upperDir, baseDir: baseRootfs, cleanup: cleanup}, nil
	}
	if err := copyDirContents(baseRootfs, stageDir); err != nil {
		return nil, nil, fmt.Errorf("seed rootfs from %s: %w", stage.Image, err)
	}
	return baseManifest, &stageWorkspace{rootfs: stageDir}, nil
}

// resolveStageBase returns the rootfs directory and manifest for a stage's
// FROM instruction without copying the files yet.
func resolveStageBase(idx int, stage DFStage, stageDirs map[string]string, logf func(string, ...any)) (string, *BuildManifest, error) {
	if prev, ok := stageDirs[strings.ToLower(stage.Image)]; ok {
		logf("[build] stage %d: base = stage %s", idx, stage.Image)
		m, _ := loadManifest(prev)
		return prev, m, nil
	}
	rootfs, m, err := PullImage(stage.Image, logf)
	if err != nil {
		return "", nil, fmt.Errorf("pull %s: %w", stage.Image, err)
	}
	return rootfs, m, nil
}

func createBuildOverlayRootfs(lowerDir string) (merged, upper string, cleanup func(), err error) {
	base := filepath.Join(stateBaseDir(), "build-overlays", randID())
	merged = filepath.Join(base, "merged")
	upper = filepath.Join(base, "upper")
	work := filepath.Join(base, "work")
	for _, dir := range []string{merged, upper, work} {
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			_ = os.RemoveAll(base)
			return "", "", nil, fmt.Errorf("overlay dir setup: %w", mkErr)
		}
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upper, work)
	if mountErr := syscall.Mount("overlay", merged, "overlay", 0, opts); mountErr != nil {
		_ = os.RemoveAll(base)
		return "", "", nil, fmt.Errorf("overlay mount: %w", mountErr)
	}
	cleanup = func() {
		_ = syscall.Unmount(merged, 0)
		_ = os.RemoveAll(base)
	}
	return merged, upper, cleanup, nil
}

// saveStageCache writes the stage's content to cacheDir and returns nil only
// when the cache is fully committed (manifest written last as an atomic marker).
//
// When the workspace is overlay-backed we reconstruct the merged view from its
// two constituent parts — base rootfs and upper layer — instead of reading
// through the overlay mount point.  Both parts live under stateBaseDir() (same
// tmpfs), so os.Link succeeds and the copy is effectively instant.  Reading
// through the overlay mount always crosses a filesystem boundary (overlay →
// tmpfs), forcing os.Link to fall back to a slow byte-by-byte io.Copy for
// every file in the tree (hundreds of MB for a typical Node image).
func saveStageCache(workspace *stageWorkspace, cacheDir string, m *BuildManifest, logf func(string, ...any), idx int) error {
	var copyErr error
	if workspace.baseDir != "" && workspace.upperDir != "" {
		// Fast path: hardlink base + upper separately (same filesystem).
		_ = os.RemoveAll(cacheDir)
		if copyErr = hardlinkCopy(workspace.baseDir, cacheDir); copyErr == nil {
			copyErr = hardlinkCopy(workspace.upperDir, cacheDir)
		}
	} else {
		// Fallback: plain dir workspace — materialize from rootfs directly.
		copyErr = materializeStageRootfs(workspace.rootfs, cacheDir)
	}
	if copyErr != nil {
		logf("[build] stage %d: warn: stage cache save failed: %v", idx, copyErr)
		_ = os.RemoveAll(cacheDir)
		return copyErr
	}
	// Write manifest last as a commit marker — if interrupted before this
	// point the cache dir has no manifest and loadManifest will reject it.
	if mfErr := saveManifest(cacheDir, m); mfErr != nil {
		logf("[build] stage %d: warn: stage cache manifest write failed: %v", idx, mfErr)
		_ = os.RemoveAll(cacheDir)
		return mfErr
	}
	return nil
}

func materializeStageRootfs(src, dest string) error {
	if src == dest {
		return nil
	}
	_ = os.RemoveAll(dest)
	return hardlinkCopy(src, dest)
}

// manifestFromBase clones a base manifest into a new mutable one.
func manifestFromBase(base *BuildManifest) *BuildManifest {
	m := &BuildManifest{Env: map[string]string{}}
	if base == nil {
		return m
	}
	m.Cmd = base.Cmd
	m.Entrypoint = base.Entrypoint
	m.Port = base.Port
	m.WorkDir = base.WorkDir
	for k, v := range base.Env {
		m.Env[k] = v
	}
	return m
}

// execInstruction runs one Dockerfile instruction, mutating m and returning
// the (possibly updated) workdir.
func execInstruction(ins DFInstruction, stageDir, workdir string, m *BuildManifest, stageDirs map[string]string, contextDir string, logw io.Writer) (string, error) {
	switch ins.Kind {
	case DFWorkdir:
		workdir = resolveContainerPath(workdir, ins.Path)
		m.WorkDir = workdir
		_ = os.MkdirAll(rootfsContainerPath(stageDir, workdir), 0755)
	case DFEnv:
		m.Env[ins.EnvKey] = ins.EnvVal
	case DFExpose:
		if ins.Port > 0 {
			m.Port = ins.Port
		}
	case DFCmd:
		m.Cmd = ins.Cmd
	case DFCopy:
		src, err := resolveCopySrc(ins, stageDir, stageDirs, contextDir)
		if err != nil {
			return workdir, fmt.Errorf("COPY --from: %w", err)
		}
		destPath := resolveContainerPath(workdir, ins.Dest)
		if err := copyPaths(src, ins.Srcs, rootfsContainerPath(stageDir, destPath), ins.Dest); err != nil {
			return workdir, fmt.Errorf("COPY %v → %s: %w", ins.Srcs, ins.Dest, err)
		}
	case DFRun:
		if err := execRun(ins, stageDir, workdir, m, logw); err != nil {
			return workdir, err
		}
	}
	return workdir, nil
}

// execRun handles a DFRun instruction with dep-prewarm cache support.
// It checks the dep cache before running the install command and saves to the
// cache afterwards, keeping execInstruction's complexity below the linter limit.
func execRun(ins DFInstruction, stageDir, workdir string, m *BuildManifest, logw io.Writer) error {
	key, artifactDir, cached := depCacheKey(ins.Shell, stageDir, workdir)
	if cached && depCacheRestore(key, artifactDir, stageDir, workdir) {
		fmt.Fprintf(logw, "[dep-cache] restored %s (skipping install)\n", artifactDir)
		return nil
	}
	if err := runInChroot(stageDir, workdir, m.Env, ins.Shell, ins.RunMounts, logw); err != nil {
		return fmt.Errorf("RUN %s: %w", truncate(ins.Shell, 60), err)
	}
	if cached {
		depCacheSave(key, artifactDir, stageDir, workdir)
	}
	return nil
}

// runInChroot executes a shell command inside the stage rootfs using a mount
// namespace so /proc and /dev are isolated and the install commands work
// correctly (npm, pip, apt-get, etc. all need /proc).
func runInChroot(rootfs, workdir string, envMap map[string]string, shell string, mounts []DFRunMount, logw io.Writer) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	// Re-exec ourselves with a magic subcommand that sets up chroot + exec.
	cmd := exec.Command(self, "__station_run__", rootfs, workdir, shell)
	cmd.Stdout = logw
	cmd.Stderr = logw

	// Build env: PATH + image env + any host proxies.
	env := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/root",
	}
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}
	// Forward proxy settings from the host so package managers can reach the internet.
	for _, key := range []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY", "no_proxy", "NO_PROXY"} {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}
	if len(mounts) > 0 {
		data, err := json.Marshal(mounts)
		if err != nil {
			return fmt.Errorf("encode RUN mounts: %w", err)
		}
		env = append(env, buildRunMountsEnv+"="+string(data))
	}
	cmd.Env = env

	// Mount namespace only — we don't need PID or user namespace for builds,
	// and skipping CLONE_NEWUSER means the process runs as the real caller's
	// UID so it can write to the rootfs (which we own).
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}
	return cmd.Run()
}

// isBuildRunInit detects the __station_run__ sub-command used by runInChroot.
func isBuildRunInit() bool {
	return len(os.Args) > 1 && os.Args[1] == "__station_run__"
}

// runBuildInit is called when isBuildRunInit() is true.
// It runs inside a fresh mount namespace with the same UID as the parent.
// Mounts /proc and /dev, chroots, then exec's sh -c <shell>.
func runBuildInit() {
	// Args: station __station_run__ <rootfs> <workdir> <shell>
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "build-run-init: bad args")
		os.Exit(1)
	}
	rootfs := os.Args[2]
	workdir := os.Args[3]
	shell := os.Args[4]

	mounts, err := loadBuildRunMounts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode RUN mounts: %v\n", err)
		os.Exit(1)
	}
	if err := mountBuildRunTargets(rootfs, workdir, mounts); err != nil {
		fmt.Fprintf(os.Stderr, "prepare RUN mounts: %v\n", err)
		os.Exit(1)
	}

	// /proc — required by npm, pip, and many other tools.
	procDir := filepath.Join(rootfs, "proc")
	_ = os.MkdirAll(procDir, 0555)
	_ = syscall.Mount("proc", procDir, "proc", 0, "")

	// /dev — bind-mount host /dev so /dev/null, /dev/urandom etc. work.
	devDir := filepath.Join(rootfs, "dev")
	_ = os.MkdirAll(devDir, 0755)
	_ = syscall.Mount("/dev", devDir, "", syscall.MS_BIND|syscall.MS_REC, "")

	// /etc/resolv.conf — bind-mount so DNS works inside the chroot.
	resolvDest := filepath.Join(rootfs, "etc", "resolv.conf")
	_ = os.MkdirAll(filepath.Dir(resolvDest), 0755)
	if _, err := os.Stat(resolvDest); err != nil {
		_, _ = os.Create(resolvDest)
	}
	_ = syscall.Mount("/etc/resolv.conf", resolvDest, "", syscall.MS_BIND, "")

	if err := syscall.Chroot(rootfs); err != nil {
		fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
		os.Exit(1)
	}
	_ = syscall.Chdir("/")

	if workdir != "" && workdir != "/" {
		_ = os.MkdirAll(workdir, 0755)
		_ = syscall.Chdir(workdir)
	}

	if err := syscall.Exec("/bin/sh", []string{"sh", "-c", shell}, os.Environ()); err != nil {
		// /bin/sh might not exist — try /bin/bash or busybox sh.
		for _, sh := range []string{"/bin/bash", "/usr/bin/sh", "/usr/bin/bash"} {
			if _, err2 := os.Stat(sh); err2 == nil {
				_ = syscall.Exec(sh, []string{sh, "-c", shell}, os.Environ())
			}
		}
		fmt.Fprintf(os.Stderr, "exec sh: %v\n", err)
		os.Exit(1)
	}
}

const buildRunMountsEnv = "STATION_BUILD_RUN_MOUNTS"

func loadBuildRunMounts() ([]DFRunMount, error) {
	raw := strings.TrimSpace(os.Getenv(buildRunMountsEnv))
	if raw == "" {
		return nil, nil
	}
	var mounts []DFRunMount
	if err := json.Unmarshal([]byte(raw), &mounts); err != nil {
		return nil, err
	}
	return mounts, nil
}

func mountBuildRunTargets(rootfs, workdir string, mounts []DFRunMount) error {
	for _, mount := range mounts {
		switch strings.ToLower(strings.TrimSpace(mount.Type)) {
		case "cache":
			if err := mountBuildCache(rootfs, workdir, mount); err != nil {
				return err
			}
		case "":
			return fmt.Errorf("RUN --mount missing type for target %q", mount.Target)
		default:
			return fmt.Errorf("RUN --mount type=%s is not supported yet", mount.Type)
		}
	}
	return nil
}

func mountBuildCache(rootfs, workdir string, mount DFRunMount) error {
	target := buildMountTargetPath(rootfs, workdir, mount.Target)
	if target == "" {
		return fmt.Errorf("cache mount requires a target")
	}
	cacheDir := filepath.Join(buildCacheRootDir(), sanitizeBuildCacheKey(firstNonEmptyStr(mount.ID, mount.Source, mount.Target)))
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if info, err := os.Stat(target); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("cache target %q is not a directory", mount.Target)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("create cache target %q: %w", mount.Target, err)
		}
	} else {
		return err
	}
	if err := syscall.Mount(cacheDir, target, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind cache %s -> %s: %w", cacheDir, mount.Target, err)
	}
	return nil
}

func buildCacheRootDir() string {
	return filepath.Join(stateBaseDir(), "build-cache")
}

func sanitizeBuildCacheKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "cache"
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		".", "_",
		" ", "_",
	)
	key = strings.Trim(replacer.Replace(key), "_")
	if key == "" {
		return "cache"
	}
	if len(key) > 80 {
		key = key[:80]
	}
	return key
}

func resolveContainerPath(workdir, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return firstNonEmptyStr(workdir, "/")
	}
	resolved := filepath.Clean(filepath.FromSlash(path))
	if !filepath.IsAbs(resolved) {
		base := firstNonEmptyStr(strings.TrimSpace(workdir), "/")
		resolved = filepath.Clean(filepath.Join(filepath.FromSlash(base), resolved))
	}
	if !strings.HasPrefix(resolved, string(os.PathSeparator)) {
		resolved = string(os.PathSeparator) + strings.TrimPrefix(resolved, string(os.PathSeparator))
	}
	return resolved
}

func rootfsContainerPath(rootfs, containerPath string) string {
	rel := strings.TrimPrefix(filepath.Clean(filepath.FromSlash(containerPath)), string(os.PathSeparator))
	if rel == "." || rel == "" {
		return rootfs
	}
	return filepath.Join(rootfs, rel)
}

func buildMountTargetPath(rootfs, workdir, target string) string {
	target = resolveContainerPath(workdir, target)
	if target == "" {
		return ""
	}
	return rootfsContainerPath(rootfs, target)
}

// ─── COPY helpers ─────────────────────────────────────────────────────────────

// resolveCopySrc returns the source root directory for a COPY instruction.
// For COPY --from=<stage> it is the stage rootfs, otherwise the build context.
func resolveCopySrc(ins DFInstruction, _ string, stageDirs map[string]string, contextDir string) (string, error) {
	if ins.FromStage == "" {
		return contextDir, nil
	}
	key := strings.ToLower(ins.FromStage)
	if dir, ok := stageDirs[key]; ok {
		return dir, nil
	}
	return "", fmt.Errorf("unknown stage %q in COPY --from", ins.FromStage)
}

// copyPaths copies each src (relative to srcRoot) to dest inside the rootfs.
func copyPaths(srcRoot string, srcs []string, dest, destSpec string) error {
	for _, src := range srcs {
		// Support glob patterns like *.jar, *.lock.
		matches, err := filepath.Glob(filepath.Join(srcRoot, src))
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			// No match — try as literal path.
			if hasGlobMeta(src) {
				continue
			}
			matches = []string{filepath.Join(srcRoot, src)}
		}
		for _, m := range matches {
			info, err := os.Lstat(m)
			if err != nil {
				return err
			}
			target := dest
			if copyDestIsDir(info, dest, destSpec, len(srcs) > 1) {
				if err := os.MkdirAll(dest, 0755); err != nil {
					return err
				}
				if !info.IsDir() {
					target = filepath.Join(dest, filepath.Base(m))
				}
			} else if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return err
			}
			if err := copyFSEntry(m, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyDestIsDir(srcInfo os.FileInfo, dest, destSpec string, multiSrc bool) bool {
	if multiSrc || srcInfo.IsDir() {
		return true
	}
	spec := strings.TrimSpace(strings.ReplaceAll(destSpec, "\\", "/"))
	if spec == "." || spec == "./" || strings.HasSuffix(spec, "/") {
		return true
	}
	if info, err := os.Stat(dest); err == nil {
		return info.IsDir()
	}
	return false
}

func hasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// ─── stage build cache ────────────────────────────────────────────────────────

// stageCacheDir returns the on-disk directory for a cached stage rootfs.
func stageCacheDir(key string) string {
	return filepath.Join(stateBaseDir(), "stage-cache", key[:2], key)
}

// recordStageHash stores the cache key for a built stage so that downstream
// stages can include it in their own cache key computation.
func recordStageHash(stage DFStage, key string, stageHashes map[string]string) {
	stageHashes[strings.ToLower(stage.Image)] = key
	if stage.Name != "" {
		stageHashes[strings.ToLower(stage.Name)] = key
	}
}

// hashFilesForCache hashes the content of files matched by patterns under dir
// into h. patterns may contain shell glob characters.
func hashFilesForCache(h io.Writer, dir string, patterns []string) error {
	for _, p := range patterns {
		abs := filepath.Join(dir, filepath.FromSlash(p))
		if hasGlobMeta(p) {
			matches, err := filepath.Glob(abs)
			if err != nil {
				return err
			}
			for _, match := range matches {
				if err := hashFSPath(h, dir, match); err != nil {
					return err
				}
			}
		} else {
			if err := hashFSPath(h, dir, abs); err != nil {
				return err
			}
		}
	}
	return nil
}

// hashFSPath hashes a single file or directory tree into h.
func hashFSPath(h io.Writer, baseDir, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.Walk(path, func(p string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil || fi.IsDir() {
				return walkErr
			}
			rel, _ := filepath.Rel(baseDir, p)
			data, readErr := os.ReadFile(p)
			if readErr != nil {
				return readErr
			}
			fmt.Fprintf(h, "f:%s:", rel)
			_, _ = h.Write(data)
			fmt.Fprint(h, "\n")
			return nil
		})
	}
	rel, _ := filepath.Rel(baseDir, path)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(h, "f:%s:", rel)
	_, _ = h.Write(data)
	fmt.Fprint(h, "\n")
	return nil
}

// stageInputHash computes a deterministic sha256 cache key for a Dockerfile
// stage. COPY sources are content-hashed; RUN/ENV/WORKDIR are string-hashed.
// Returns ("", false) if any input cannot be fully determined.
func stageInputHash(stage DFStage, contextDir string, stageDirs, stageHashes map[string]string) (string, bool) {
	h := sha256.New()

	// Base layer
	baseKey := strings.ToLower(strings.TrimSpace(stage.Image))
	if _, ok := stageDirs[baseKey]; ok {
		// FROM <previous stage> — include that stage's own cache key
		prevHash, ok := stageHashes[baseKey]
		if !ok {
			return "", false // previous stage wasn't cached — skip cache
		}
		fmt.Fprintf(h, "from-stage:%s\n", prevHash)
	} else {
		// FROM <external image> — use the image reference as-is
		fmt.Fprintf(h, "from-image:%s\n", stage.Image)
	}

	for _, ins := range stage.Instructions {
		switch ins.Kind {
		case DFCopy:
			srcDir := contextDir
			if ins.FromStage != "" {
				key := strings.ToLower(strings.TrimSpace(ins.FromStage))
				prev, ok := stageDirs[key]
				if !ok {
					return "", false
				}
				srcDir = prev
			}
			if err := hashFilesForCache(h, srcDir, ins.Srcs); err != nil {
				return "", false
			}
			fmt.Fprintf(h, "->%s\n", ins.Dest)
		case DFRun:
			fmt.Fprintf(h, "run:%s\n", ins.Shell)
			for _, m := range ins.RunMounts {
				fmt.Fprintf(h, "mount:%s:%s:%s\n", m.Type, m.Target, m.ID)
			}
		case DFEnv:
			fmt.Fprintf(h, "env:%s=%s\n", ins.EnvKey, ins.EnvVal)
		case DFWorkdir:
			fmt.Fprintf(h, "workdir:%s\n", ins.Path)
		case DFCmd:
			for _, c := range ins.Cmd {
				fmt.Fprintf(h, "cmd:%s\n", c)
			}
		case DFExpose:
			fmt.Fprintf(h, "expose:%d\n", ins.Port)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), true
}

// copyFSEntry copies a single file or directory tree.
func copyFSEntry(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDirContents(src, dest)
	}
	return copyFile(src, dest, info.Mode()) // copyFile is defined in cache.go
}

// copyDirContents recursively copies src/ into dest/.
func copyDirContents(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		// Preserve symlinks.
		if info.Mode()&os.ModeSymlink != 0 {
			link, _ := os.Readlink(path)
			_ = os.Remove(target)
			return os.Symlink(link, target)
		}
		return copyFile(path, target, info.Mode())
	})
}

// ─── utils ────────────────────────────────────────────────────────────────────

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}


