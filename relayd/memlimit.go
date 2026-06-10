// memlimit.go — daemon memory self-governance for small hosts.
package main

import (
	"bufio"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// hostMemInfo holds totals parsed from /proc/meminfo (Linux). Zero values mean
// "unknown" — e.g. on macOS/Windows dev machines — and callers fall back to
// conservative defaults.
type hostMemInfo struct {
	totalMB int
	swapMB  int
}

var (
	hostMemOnce sync.Once
	hostMem     hostMemInfo
)

func readHostMemInfo() hostMemInfo {
	hostMemOnce.Do(func() {
		f, err := os.Open("/proc/meminfo")
		if err != nil {
			return
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			var dst *int
			switch {
			case strings.HasPrefix(line, "MemTotal:"):
				dst = &hostMem.totalMB
			case strings.HasPrefix(line, "SwapTotal:"):
				dst = &hostMem.swapMB
			default:
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.Atoi(fields[1]); err == nil {
					*dst = kb / 1024
				}
			}
		}
	})
	return hostMem
}

func hostTotalMemMB() int { return readHostMemInfo().totalMB }

func hostSwapTotalMB() int { return readHostMemInfo().swapMB }

// setupMemoryLimits keeps the daemon's own footprint predictable on small
// hosts. Go's default GC lets the heap grow to ~2x its live size before
// collecting, which on a 2 GB instance shows up as relayd idling at several
// hundred MB of RSS while builds get OOM-killed next door. A soft memory
// limit makes the GC return memory to the OS instead of hoarding it; it
// never kills the process.
//
// Explicit GOGC / GOMEMLIMIT env vars win — the runtime already applied
// them at startup, so this leaves them alone.
func setupMemoryLimits() {
	totalMB := hostTotalMemMB()

	if os.Getenv("GOMEMLIMIT") == "" {
		limitMB := 0
		if v := strings.TrimSpace(os.Getenv("RELAY_GOMEMLIMIT_MB")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limitMB = n
			}
		} else if totalMB > 0 {
			// A quarter of physical RAM, clamped: leaves the rest for Docker,
			// builds, and the apps relayd is hosting.
			limitMB = totalMB / 4
			if limitMB < 192 {
				limitMB = 192
			}
			if limitMB > 768 {
				limitMB = 768
			}
		}
		if limitMB > 0 {
			debug.SetMemoryLimit(int64(limitMB) << 20)
		}
	}

	if os.Getenv("GOGC") == "" && totalMB > 0 && totalMB <= 4096 {
		// Collect more eagerly on small hosts: trades a little CPU for a much
		// flatter RSS curve.
		debug.SetGCPercent(50)
	}
}

// defaultAppMemLimitMB returns the --memory cap (in MB) applied to app
// containers that have no explicit limit configured. Without it, one leaky
// app can take down the whole host — relayd, the other apps, and the build
// pipeline with it. Sized to leave room for a second app plus the daemon and
// Docker even on a 2 GB instance. The default container memory-swap allowance
// (2x the cap) means apps hit swap before they hit the OOM killer.
//
// RELAY_APP_MEM_LIMIT_MB=<n> overrides; RELAY_APP_MEM_LIMIT_MB=0 disables.
func defaultAppMemLimitMB() int {
	if v := strings.TrimSpace(os.Getenv("RELAY_APP_MEM_LIMIT_MB")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return 0
		}
		return n
	}
	total := hostTotalMemMB()
	if total <= 0 {
		return 0
	}
	limit := total * 45 / 100
	if limit < 256 {
		limit = 256
	}
	if limit > 4096 {
		limit = 4096
	}
	return limit
}

// daemonRSSBytes returns the daemon's resident set size, or 0 if unknown.
func daemonRSSBytes() int64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				return kb * 1024
			}
		}
	}
	return 0
}

// memSample is one point of the daemon memory history shown in the dashboard.
type memSample struct {
	AtMs      int64  `json:"at_ms"`
	RSSBytes  int64  `json:"rss_bytes"`
	HeapBytes uint64 `json:"heap_bytes"`
}

const memSampleCap = 120 // 1 hour of history at 30s cadence

var (
	memSamplesMu sync.Mutex
	memSamples   []memSample
)

func recordMemSample() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	sample := memSample{AtMs: time.Now().UnixMilli(), RSSBytes: daemonRSSBytes(), HeapBytes: ms.HeapAlloc}
	memSamplesMu.Lock()
	memSamples = append(memSamples, sample)
	if len(memSamples) > memSampleCap {
		memSamples = memSamples[len(memSamples)-memSampleCap:]
	}
	memSamplesMu.Unlock()
}

func snapshotMemSamples() []memSample {
	memSamplesMu.Lock()
	defer memSamplesMu.Unlock()
	out := make([]memSample, len(memSamples))
	copy(out, memSamples)
	return out
}

// adminOpsDaemon is the daemon self-report included in /api/admin/ops so
// operators can see relayd's own footprint instead of trusting it.
type adminOpsDaemon struct {
	RSSBytes        int64       `json:"rss_bytes"`
	HeapAllocBytes  uint64      `json:"heap_alloc_bytes"`
	HeapSysBytes    uint64      `json:"heap_sys_bytes"`
	GoMemLimitBytes int64       `json:"go_mem_limit_bytes,omitempty"`
	Goroutines      int         `json:"goroutines"`
	HostMemTotalMB  int         `json:"host_mem_total_mb,omitempty"`
	HostSwapMB      int         `json:"host_swap_mb"`
	Samples         []memSample `json:"samples"`
}

func collectDaemonStats() adminOpsDaemon {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	stats := adminOpsDaemon{
		RSSBytes:       daemonRSSBytes(),
		HeapAllocBytes: ms.HeapAlloc,
		HeapSysBytes:   ms.Sys,
		Goroutines:     runtime.NumGoroutine(),
		HostMemTotalMB: hostTotalMemMB(),
		HostSwapMB:     hostSwapTotalMB(),
		Samples:        snapshotMemSamples(),
	}
	// SetMemoryLimit(-1) reads the current limit without changing it.
	if limit := debug.SetMemoryLimit(-1); limit > 0 && limit < int64(1)<<62 {
		stats.GoMemLimitBytes = limit
	}
	return stats
}

// startMemorySampler records daemon memory every 30 seconds so operators can
// see the footprint (and catch regressions) instead of trusting it.
func startMemorySampler() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		recordMemSample()
		for range ticker.C {
			recordMemSample()
		}
	}()
}
