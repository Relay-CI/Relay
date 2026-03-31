package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
)

// portState maps app name → allocated port. Persisted to disk so ports survive
// restarts and remain stable across redeploys of the same app.
type portState map[string]int

func portsFilePath() string {
	return filepath.Join(stateBaseDir(), "ports.json")
}

func loadPortState() portState {
	data, err := os.ReadFile(portsFilePath())
	if err != nil {
		return make(portState)
	}
	var m portState
	if json.Unmarshal(data, &m) != nil {
		return make(portState)
	}
	return m
}

func savePortState(m portState) {
	_ = os.MkdirAll(stateBaseDir(), 0755)
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(portsFilePath(), data, 0644)
}

// allocPort returns a stable port for appName. If the app already has an
// allocated port and it is still free on the host, the same port is reused
// (important for relay's redeploy workflow — the reverse-proxy keeps working).
// Otherwise a fresh OS-assigned free port is allocated.
func allocPort(appName string) (int, error) {
	m := loadPortState()
	if port, ok := m[appName]; ok && portFree(port) {
		return port, nil // reuse existing allocation
	}
	port, err := findFreePort()
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	m[appName] = port
	savePortState(m)
	return port, nil
}

// releasePort removes the port allocation for appName.
func releasePort(appName string) {
	m := loadPortState()
	delete(m, appName)
	savePortState(m)
}

// lookupPort returns the port currently allocated to appName, or 0 if none.
func lookupPort(appName string) int {
	return loadPortState()[appName]
}

// findFreePort asks the OS to bind on port 0, which causes it to pick any
// available ephemeral port. We immediately close the listener and return that
// port number. There is a small TOCTOU window, but it is fine in practice for
// a single-host dev/deploy tool.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

// portFree returns true if nothing is currently listening on the given port.
func portFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

// cmdPortList prints all current app → port allocations.
func cmdPortList() {
	m := loadPortState()
	if len(m) == 0 {
		fmt.Println("no port allocations")
		return
	}
	apps := make([]string, 0, len(m))
	for a := range m {
		apps = append(apps, a)
	}
	sort.Strings(apps)
	fmt.Printf("%-20s  %s\n", "APP", "PORT")
	for _, a := range apps {
		status := "free"
		if !portFree(m[a]) {
			status = "in use"
		}
		fmt.Printf("%-20s  %d  (%s)\n", a, m[a], status)
	}
}

// cmdPortFree releases a port allocation by app name.
func cmdPortFree(appName string) {
	if lookupPort(appName) == 0 {
		die("no port allocated for app %q", appName)
	}
	releasePort(appName)
	fmt.Printf("released port for %s\n", appName)
}
