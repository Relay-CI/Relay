package main

import (
	"reflect"
	"testing"
)

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
	if got := housekeepingBuildCacheKeepGB(); got != 5 {
		t.Fatalf("default keep-GB = %d, want 5", got)
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
	if got := housekeepingBuildCacheKeepGB(); got != 5 {
		t.Fatalf("invalid keep-GB = %d, want 5 (default)", got)
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
