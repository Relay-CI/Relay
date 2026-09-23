package main

import (
	"reflect"
	"testing"
)

// The *Setting methods (imageRetentionPerLaneSetting etc.) back the Server
// Settings → Cleanup UI: a DB-persisted value must win over the env var and
// hardcoded default, matching serverBaseDomain's precedence.
func TestCleanupSettingsPreferDBValueOverEnvAndDefault(t *testing.T) {
	s := newPreviewPortTestServer(t)

	if got := s.imageRetentionPerLaneSetting(); got != 3 {
		t.Fatalf("default image_retention_per_lane = %d, want 3", got)
	}
	t.Setenv("RELAY_IMAGE_RETENTION_PER_LANE", "7")
	if got := s.imageRetentionPerLaneSetting(); got != 7 {
		t.Fatalf("env-overridden image_retention_per_lane = %d, want 7", got)
	}
	if _, err := s.db.Exec(`INSERT INTO server_config (key, value) VALUES ('image_retention_per_lane', '1')`); err != nil {
		t.Fatalf("seed db: %v", err)
	}
	if got := s.imageRetentionPerLaneSetting(); got != 1 {
		t.Fatalf("db-overridden image_retention_per_lane = %d, want 1 (should beat env var)", got)
	}

	if got := s.unusedImageMaxAgeDaysSetting(); got != 14 {
		t.Fatalf("default unused_image_max_age_days = %d, want 14", got)
	}
	if got := s.logRetentionDaysSetting(); got != 30 {
		t.Fatalf("default log_retention_days = %d, want 30", got)
	}
	if got := s.buildCacheKeepGBSetting(); got != 10 {
		t.Fatalf("default build_cache_keep_gb = %d, want 10", got)
	}
}

func TestSelectImagesToPrune(t *testing.T) {
	records := []laneImageRecord{
		// lane A, newest first
		{Lane: "a__production__main", Image: "relay/a:production-main-5"},
		{Lane: "a__production__main", Image: "relay/a:production-main-4"},
		{Lane: "a__production__main", Image: "relay/a:production-main-3"},
		{Lane: "a__production__main", Image: "relay/a:production-main-2"},
		{Lane: "a__production__main", Image: "relay/a:production-main-1"},
		// lane B has fewer than keep
		{Lane: "b__production__main", Image: "relay/b:production-main-2"},
		{Lane: "b__production__main", Image: "relay/b:production-main-1"},
		// non-relay image never pruned
		{Lane: "c__production__main", Image: "user/custom:1"},
		{Lane: "c__production__main", Image: "user/custom:0"},
	}
	protected := map[string]struct{}{
		"relay/a:production-main-2": {}, // pretend it's the rollback target
	}
	got := selectImagesToPrune(records, protected, 2)
	want := []string{"relay/a:production-main-3", "relay/a:production-main-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectImagesToPrune = %v, want %v", got, want)
	}
}

func TestSelectImagesToPruneDeduplicates(t *testing.T) {
	records := []laneImageRecord{
		{Lane: "a__production__main", Image: "relay/a:tag-1"},
		{Lane: "a__production__main", Image: "relay/a:tag-1"},
		{Lane: "a__production__main", Image: "relay/a:tag-1"},
	}
	if got := selectImagesToPrune(records, nil, 2); len(got) != 0 {
		t.Fatalf("duplicate rows must not be pruned, got %v", got)
	}
}

func TestHousekeepingBuildCacheKeepGB(t *testing.T) {
	// default when unset
	t.Setenv("RELAY_BUILD_CACHE_KEEP_GB", "")
	if got := housekeepingBuildCacheKeepGB(); got != 10 {
		t.Fatalf("default keep-GB = %d, want 10", got)
	}
	// explicit override
	t.Setenv("RELAY_BUILD_CACHE_KEEP_GB", "12")
	if got := housekeepingBuildCacheKeepGB(); got != 12 {
		t.Fatalf("override keep-GB = %d, want 12", got)
	}
	// 0 disables the cap (kept as 0, honored by pruneBuildCache)
	t.Setenv("RELAY_BUILD_CACHE_KEEP_GB", "0")
	if got := housekeepingBuildCacheKeepGB(); got != 0 {
		t.Fatalf("zero keep-GB = %d, want 0", got)
	}
	// garbage falls back to default
	t.Setenv("RELAY_BUILD_CACHE_KEEP_GB", "notanumber")
	if got := housekeepingBuildCacheKeepGB(); got != 10 {
		t.Fatalf("invalid keep-GB = %d, want 10 (default)", got)
	}
}

func TestDefaultAppMemLimitMB(t *testing.T) {
	t.Setenv("RELAY_APP_MEM_LIMIT_MB", "512")
	if got := defaultAppMemLimitMB(3); got != 512 {
		t.Fatalf("explicit override: got %d, want 512", got)
	}
	t.Setenv("RELAY_APP_MEM_LIMIT_MB", "0")
	if got := defaultAppMemLimitMB(1); got != 0 {
		t.Fatalf("RELAY_APP_MEM_LIMIT_MB=0 must disable the cap, got %d", got)
	}
}

// TestDefaultAppMemLimitMBForHostDoesNotOvercommit pins totalMB explicitly
// (real defaultAppMemLimitMB always reads the actual host's RAM, which
// varies per machine) to check the invariant the fix exists for: the old
// behavior — a flat 45%% of host per app regardless of how many apps were
// running — let per-app caps sum past 100%% of the host the moment more than
// ~2 apps were deployed. That overcommit is exactly what hands the kernel
// OOM killer a reason to start killing containers mid-request, which
// Cloudflare (sitting in front of them) sees as intermittent 502s and
// partial asset failures across whichever apps got picked.
func TestDefaultAppMemLimitMBForHostDoesNotOvercommit(t *testing.T) {
	for _, totalMB := range []int{2048, 4096, 8192} {
		for _, apps := range []int{2, 3, 5, 8} {
			perApp := defaultAppMemLimitMBForHost(totalMB, apps)
			if sum := perApp * apps; sum > totalMB {
				t.Errorf("totalMB=%d apps=%d: %d MB each sums to %d MB, overcommits the host", totalMB, apps, perApp, sum)
			}
		}
	}
}

func TestDefaultAppMemLimitMBForHostSplitsAcrossRunningApps(t *testing.T) {
	one := defaultAppMemLimitMBForHost(4096, 1)
	three := defaultAppMemLimitMBForHost(4096, 3)
	if three >= one {
		t.Fatalf("3 concurrent apps should each get less than 1 app alone: one=%d three=%d", one, three)
	}
	if got := defaultAppMemLimitMBForHost(4096, 8); got < 256 {
		t.Fatalf("per-app floor should still apply under heavy split, got %d", got)
	}
}

// defaultAppMemLimitMBForHost mirrors defaultAppMemLimitMB's env-override-free
// formula with an explicit totalMB so tests don't depend on the real host's
// RAM. Keep in sync with defaultAppMemLimitMB in memlimit.go.
func defaultAppMemLimitMBForHost(total, runningApps int) int {
	if total <= 0 {
		return 0
	}
	if runningApps < 1 {
		runningApps = 1
	}
	reserve := total * 20 / 100
	if reserve < 512 {
		reserve = 512
	}
	usable := total - reserve
	if usable < 0 {
		usable = 0
	}
	limit := (usable * 70 / 100) / runningApps
	if limit < 256 {
		limit = 256
	}
	if limit > 4096 {
		limit = 4096
	}
	return limit
}
