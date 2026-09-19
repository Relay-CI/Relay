package main

import "testing"

func TestApplyLaneRolloutPersistsStateAfterRouteChange(t *testing.T) {
	s := newPreviewPortTestServer(t)
	planned := &AppState{App: "demo", Env: EnvPreview, Branch: "main", Engine: EngineDocker, ActiveSlot: "green", StandbySlot: "blue", TrafficMode: "edge", ServicePort: 3000}
	routed := false
	if err := s.applyLaneRollout(planned, func() error { routed = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !routed {
		t.Fatal("route transition was not called")
	}
	got, err := s.getAppState("demo", EnvPreview, "main")
	if err != nil || got == nil || got.ActiveSlot != "green" || got.StandbySlot != "blue" {
		t.Fatalf("planned Lane State not saved: %#v, %v", got, err)
	}
	var intents int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM lane_rollout_intents`).Scan(&intents); err != nil || intents != 0 {
		t.Fatalf("rollout intent not cleared: %d, %v", intents, err)
	}
}
