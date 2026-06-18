// housekeeping.go — periodic pruning of old deploy artifacts so small hosts
// don't slowly fill their disks. Disk pressure evicts BuildKit's cache (which
// silently destroys incremental build speed) and eventually fails builds with
// "no space left on device".
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func housekeepingLogRetentionDays() int {
	if v := strings.TrimSpace(os.Getenv("RELAY_LOG_RETENTION_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 30
}

func housekeepingImageRetentionPerLane() int {
	if v := strings.TrimSpace(os.Getenv("RELAY_IMAGE_RETENTION_PER_LANE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 3
}

// housekeepingInterval returns how often to run the full housekeeping pass.
// On small hosts (≤2.2 GB RAM) images and logs accumulate faster relative to
// available disk, and BuildKit cache eviction silently destroys incremental
// build speed, so we prune more aggressively. Larger hosts can afford to wait.
func housekeepingInterval() time.Duration {
	if total := hostTotalMemMB(); total > 0 && total <= 2200 {
		return 3 * time.Hour
	}
	return 12 * time.Hour
}

// diskAvailableMB returns the available disk space in MB for the partition
// that contains path, or 0 if it cannot be determined.
func diskAvailableMB(path string) int {
	out, err := exec.Command("df", "-Pm", path).CombinedOutput()
	if err != nil || len(out) == 0 {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			if mb, err := strconv.Atoi(fields[3]); err == nil {
				return mb
			}
		}
	}
	return 0
}

func (s *Server) runHousekeepingWorker() {
	// Let startup (and any boot-time deploys) settle before the first pass.
	time.Sleep(2 * time.Minute)
	for {
		s.housekeepOnce()
		time.Sleep(housekeepingInterval())
	}
}

func (s *Server) housekeepOnce() {
	s.pruneDeployLogs()
	s.pruneDeployImages()
	s.pruneAbandonedSyncSessions()
	s.pruneOldInMemoryDeploys()
}

// pruneAbandonedSyncSessions removes sync sessions whose clients never called
// /finish — i.e. they were abandoned mid-upload by a network drop, a SIGKILL,
// or a bug. Without this, their staging directories (up to 500 MB each) sit on
// disk indefinitely and the in-memory map keeps growing.
//
// Sessions younger than sessionTTL are left alone so in-progress multi-chunk
// uploads are not disrupted.
func (s *Server) pruneAbandonedSyncSessions() {
	const sessionTTL = 4 * time.Hour
	cutoff := time.Now().Add(-sessionTTL)

	s.syncMu.Lock()
	var stale []*SyncSession
	for _, sess := range s.syncSessions {
		if sess.CreatedAt.Before(cutoff) {
			stale = append(stale, sess)
		}
	}
	for _, sess := range stale {
		delete(s.syncSessions, sess.ID)
	}
	s.syncMu.Unlock()

	for _, sess := range stale {
		_ = s.deleteSessionFromDB(sess.ID)
		if sess.StagingDir != "" {
			_ = os.RemoveAll(sess.StagingDir)
		}
	}
	if len(stale) > 0 {
		fmt.Printf("housekeeping: removed %d abandoned sync session(s) older than %s\n", len(stale), sessionTTL)
	}
}

// pruneOldInMemoryDeploys trims the in-memory deploys map so it never holds
// more than maxInMemoryDeploys entries. Older completed deploys are removed;
// they remain in the database and are fetched on demand by the log-stream and
// deploy-status handlers. This prevents a long-lived server from accumulating
// thousands of Deploy structs in RAM.
//
// Active (queued/running) deploys and the most recent N completed deploys per
// app+env+branch are kept.
func (s *Server) pruneOldInMemoryDeploys() {
	const maxInMemoryDeploys = 200

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.deploys) <= maxInMemoryDeploys {
		return
	}

	// Collect completed/failed deploys sorted oldest-first.
	type entry struct {
		id        string
		createdAt time.Time
		active    bool
	}
	entries := make([]entry, 0, len(s.deploys))
	for id, d := range s.deploys {
		if d == nil {
			continue
		}
		entries = append(entries, entry{
			id:        id,
			createdAt: d.CreatedAt,
			active:    isActiveDeployStatus(d.Status),
		})
	}
	// Sort oldest first so we evict from the front.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].createdAt.Before(entries[j].createdAt)
	})

	removed := 0
	for _, e := range entries {
		if len(s.deploys)-removed <= maxInMemoryDeploys {
			break
		}
		if e.active {
			continue // never evict running builds
		}
		delete(s.deploys, e.id)
		removed++
	}
	if removed > 0 {
		fmt.Printf("housekeeping: evicted %d old deploy record(s) from memory (still in DB)\n", removed)
	}
}

// pruneDeployLogs removes build/deploy log files older than the retention
// window. RELAY_LOG_RETENTION_DAYS overrides the 30-day default; 0 disables.
func (s *Server) pruneDeployLogs() {
	days := housekeepingLogRetentionDays()
	if days <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	entries, err := os.ReadDir(s.logsDir)
	if err != nil {
		return
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		if os.Remove(filepath.Join(s.logsDir, e.Name())) == nil {
			removed++
		}
	}
	if removed > 0 {
		fmt.Printf("housekeeping: removed %d deploy log(s) older than %d days\n", removed, days)
	}
}

type laneImageRecord struct {
	Lane  string // app__env__branch
	Image string // newest-first within a lane
}

// selectImagesToPrune returns the images to delete: everything beyond the
// newest `keep` per lane, except protected refs (current/previous images that
// rollback depends on). Records must be ordered newest-first.
func selectImagesToPrune(records []laneImageRecord, protected map[string]struct{}, keep int) []string {
	seenPerLane := map[string]int{}
	seenImage := map[string]struct{}{}
	var prune []string
	for _, rec := range records {
		if rec.Image == "" {
			continue
		}
		if _, dup := seenImage[rec.Image]; dup {
			continue
		}
		seenImage[rec.Image] = struct{}{}
		seenPerLane[rec.Lane]++
		if seenPerLane[rec.Lane] <= keep {
			continue
		}
		if _, ok := protected[rec.Image]; ok {
			continue
		}
		if !strings.HasPrefix(rec.Image, "relay/") {
			continue
		}
		prune = append(prune, rec.Image)
	}
	return prune
}

// pruneDeployImages deletes relay-built images beyond the newest N per lane.
// RELAY_IMAGE_RETENTION_PER_LANE overrides the default of 3; 0 disables.
// Images still referenced by app_state (current or previous, i.e. rollback
// targets) are never deleted, and `docker rmi` without -f refuses to remove
// anything a container still uses.
func (s *Server) pruneDeployImages() {
	keep := housekeepingImageRetentionPerLane()
	if keep <= 0 {
		return
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return
	}

	protected := map[string]struct{}{}
	if rows, err := s.db.Query(`SELECT COALESCE(current_image,''), COALESCE(previous_image,'') FROM app_state`); err == nil {
		for rows.Next() {
			var cur, prev string
			if rows.Scan(&cur, &prev) == nil {
				if cur != "" {
					protected[cur] = struct{}{}
				}
				if prev != "" {
					protected[prev] = struct{}{}
				}
			}
		}
		rows.Close()
	}

	var records []laneImageRecord
	if rows, err := s.db.Query(
		`SELECT app, env, branch, COALESCE(image_tag,'') FROM deploys
		 WHERE COALESCE(image_tag,'') != '' ORDER BY created_at DESC`,
	); err == nil {
		for rows.Next() {
			var app, env, branch, image string
			if rows.Scan(&app, &env, &branch, &image) == nil {
				records = append(records, laneImageRecord{
					Lane:  app + "__" + env + "__" + branch,
					Image: image,
				})
			}
		}
		rows.Close()
	}

	removed := 0
	for _, image := range selectImagesToPrune(records, protected, keep) {
		if exec.Command("docker", "rmi", image).Run() == nil {
			removed++
		}
	}
	// Dangling layers from superseded builds hold no tags and are always safe
	// to reclaim; this does not touch BuildKit cache mounts.
	_ = exec.Command("docker", "image", "prune", "-f").Run()
	if removed > 0 {
		fmt.Printf("housekeeping: removed %d old build image(s) (keeping %d per lane)\n", removed, keep)
	}
}
