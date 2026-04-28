package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// tailFileToWriter copies new bytes from src to dst until done is closed, then
// drains any remaining bytes.  Used to provide live terminal output while
// directing cmd.Stdout/Stderr to an *os.File (which avoids the goroutine leak
// inside cmd.Wait() that occurs when a non-File writer is used and a detached
// grandchild process keeps the pipe write-end alive).
func tailFileToWriter(src *os.File, dst io.Writer, done <-chan struct{}) {
	buf := make([]byte, 8192)
	for {
		n, _ := src.Read(buf)
		if n > 0 {
			_, _ = dst.Write(buf[:n])
			continue
		}
		select {
		case <-done:
			_, _ = io.Copy(dst, src)
			return
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// BuildRecord is persisted after each build so past results are inspectable.
type BuildRecord struct {
	ID       string    `json:"id"`
	App      string    `json:"app"`
	Dir      string    `json:"dir"`
	Command  []string  `json:"command"`
	ExitCode int       `json:"exit_code"`
	Started  time.Time `json:"started"`
	Elapsed  string    `json:"elapsed"`
}

func buildDir(id string) string { return filepath.Join(stateBaseDir(), "builds", id) }
func buildLogFile(id string) string { return filepath.Join(buildDir(id), "output.log") }
func buildMetaFile(id string) string { return filepath.Join(buildDir(id), "build.json") }

// cmdBuild runs one build command inside dir, streaming output to the terminal
// while also saving it to a log file.
//
// Usage: station build [--app <name>] <dir> <cmd> [args...]
//
// Chain multiple steps to do a full build:
//   station build ./myapp npm ci
//   station build ./myapp npm run build
//   station build --app myapi ./myapp go build -o server .
func cmdBuild(appName, dir string, cmdArgs []string) {
	absDir := mustAbs(dir)
	if _, err := os.Stat(absDir); err != nil {
		die("dir %q: %v", absDir, err)
	}

	id := randID()
	if err := os.MkdirAll(buildDir(id), 0755); err != nil {
		die("create build dir: %v", err)
	}

	lf, err := os.Create(buildLogFile(id))
	if err != nil {
		die("create build log: %v", err)
	}

	app := appName
	if app == "" {
		app = filepath.Base(absDir)
	}

	start := time.Now()
	fmt.Printf("[build:%s] %s\n", id, strings.Join(cmdArgs, " "))
	fmt.Printf("[build:%s] dir: %s\n", id, absDir)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = absDir
	// Write directly to the log *os.File so exec uses no internal copy goroutine.
	// A separate tail goroutine streams to the terminal.  This avoids a hang when
	// the build command spawns detached grandchildren (npm daemons, esbuild, etc.)
	// that inherit the pipe write-end and keep cmd.Wait() blocked indefinitely.
	cmd.Stdout = lf
	cmd.Stderr = lf

	tailDone := make(chan struct{})
	tr, trErr := os.Open(lf.Name())
	if trErr == nil {
		go tailFileToWriter(tr, os.Stdout, tailDone)
	}

	runErr := cmd.Run()
	close(tailDone)
	if tr != nil {
		_ = tr.Close()
	}
	_ = lf.Close()

	elapsed := time.Since(start).Round(time.Millisecond)
	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			die("build: %v", runErr)
		}
	}

	rec := BuildRecord{
		ID: id, App: app, Dir: absDir, Command: cmdArgs,
		ExitCode: exitCode, Started: start, Elapsed: elapsed.String(),
	}
	if data, err := json.MarshalIndent(rec, "", "  "); err == nil {
		_ = os.WriteFile(buildMetaFile(id), data, 0644)
	}

	if exitCode == 0 {
		fmt.Printf("[build:%s] done in %s\n", id, elapsed)
	} else {
		fmt.Printf("[build:%s] FAILED (exit %d) in %s — station build-logs %s\n", id, exitCode, elapsed, id)
		os.Exit(exitCode)
	}
}

// cmdBuildLogs prints the saved output for a past build.
func cmdBuildLogs(id string) {
	data, err := os.ReadFile(buildLogFile(id))
	if err != nil {
		die("no build log for %s: %v", id, err)
	}
	_, _ = os.Stdout.Write(data)
}

