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

func (s *Server) runHousekeepingWorker() {
	// Let startup (and any boot-time deploys) settle before the first pass.
	time.Sleep(2 * time.Minute)
	for {
		s.housekeepOnce()
		time.Sleep(12 * time.Hour)
	}
}

func (s *Server) housekeepOnce() {
	s.pruneDeployLogs()
	s.pruneDeployImages()
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
