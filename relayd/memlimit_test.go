package main

import (
	"fmt"
	"testing"
	"time"
)

func TestAdaptiveNodeBuildHeapMB(t *testing.T) {
	cases := []struct {
		totalMB int
		want    int
	}{
		{512, 384},   // tiny host: floor wins
		{1024, 384},  // 1 GB: floor wins
		{2048, 768},  // 2 GB GCP e2-small: leaves room for Docker + relayd + app
		{4096, 1792}, // 4 GB
		{8192, 2048}, // big host: ceiling wins
	}
	for _, tc := range cases {
		if got := adaptiveNodeBuildHeapMB(tc.totalMB); got != tc.want {
			t.Errorf("adaptiveNodeBuildHeapMB(%d) = %d, want %d", tc.totalMB, got, tc.want)
		}
	}
}

func TestNodeBuildHeapMBEnvOverrideWins(t *testing.T) {
	t.Setenv("RELAY_NODE_BUILD_HEAP_MB", "512")
	if got := nodeBuildHeapMB(); got != "512" {
		t.Fatalf("nodeBuildHeapMB() = %q, want 512", got)
	}
	t.Setenv("RELAY_NODE_BUILD_HEAP_MB", "not-a-number")
	if got := nodeBuildHeapMB(); got == "not-a-number" {
		t.Fatalf("invalid override must not be used verbatim, got %q", got)
	}
}

func TestWebhookAllowedPrunesStaleRepos(t *testing.T) {
	s := &Server{webhookHits: make(map[string][]time.Time)}
	stale := time.Now().Add(-10 * time.Minute)
	for i := 0; i < 200; i++ {
		s.webhookHits[fmt.Sprintf("https://example.com/repo-%d", i)] = []time.Time{stale}
	}
	if !s.webhookAllowed("https://example.com/fresh") {
		t.Fatal("fresh repo should be allowed")
	}
	if len(s.webhookHits) > 2 {
		t.Fatalf("stale repo entries should have been pruned, map still has %d entries", len(s.webhookHits))
	}
}
