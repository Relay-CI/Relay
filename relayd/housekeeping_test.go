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
	if got := defaultAppMemLimitMB(); got != 512 {
		t.Fatalf("explicit override: got %d, want 512", got)
	}
	t.Setenv("RELAY_APP_MEM_LIMIT_MB", "0")
	if got := defaultAppMemLimitMB(); got != 0 {
		t.Fatalf("RELAY_APP_MEM_LIMIT_MB=0 must disable the cap, got %d", got)
	}
}
