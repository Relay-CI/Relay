package main

import "testing"

func TestEnqueueRollbackForLaneUsesSourceLaneImage(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.logsDir = t.TempDir()
	s.queue = make(chan DeployJob, 1)
	s.deploys = map[string]*Deploy{}

	target := &AppState{
		App:           "demo",
		Env:           EnvProd,
		Branch:        "main",
		Engine:        EngineDocker,
		CurrentImage:  "registry/demo:prod-new",
		PreviousImage: "registry/demo:prod-old",
		Mode:          "port",
		HostPort:      3000,
		ServicePort:   3000,
	}
	source := &AppState{
		App:          "demo",
		Env:          EnvStaging,
		Branch:       "main",
		Engine:       EngineDocker,
		CurrentImage: "registry/demo:staging-good",
		Mode:         "port",
		HostPort:     3002,
		ServicePort:  3000,
	}
	if err := s.saveAppState(target); err != nil {
		t.Fatalf("save target state: %v", err)
	}
	if err := s.saveAppState(source); err != nil {
		t.Fatalf("save source state: %v", err)
	}

	deployID, err := s.enqueueRollbackForLane(RollbackRequest{
		App:          "demo",
		Env:          EnvProd,
		Branch:       "main",
		SourceEnv:    EnvStaging,
		SourceBranch: "main",
	}, "alice", "rollback")
	if err != nil {
		t.Fatalf("enqueue rollback: %v", err)
	}
	if deployID == "" {
		t.Fatalf("expected deploy id")
	}

	select {
	case job := <-s.queue:
		if !job.Rollback {
			t.Fatalf("expected rollback job")
		}
		if job.RollbackImage != source.CurrentImage {
			t.Fatalf("expected rollback image %q, got %q", source.CurrentImage, job.RollbackImage)
		}
		if job.Req.Env != EnvProd {
			t.Fatalf("expected target env %q, got %q", EnvProd, job.Req.Env)
		}
	default:
		t.Fatalf("expected rollback job in queue")
	}
}

func TestEnqueueRollbackForLaneRejectsEngineMismatch(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.logsDir = t.TempDir()
	s.queue = make(chan DeployJob, 1)
	s.deploys = map[string]*Deploy{}

	if err := s.saveAppState(&AppState{
		App:           "demo",
		Env:           EnvProd,
		Branch:        "main",
		Engine:        EngineDocker,
		CurrentImage:  "registry/demo:prod-new",
		PreviousImage: "registry/demo:prod-old",
		Mode:          "port",
		HostPort:      3000,
		ServicePort:   3000,
	}); err != nil {
		t.Fatalf("save target state: %v", err)
	}
	if err := s.saveAppState(&AppState{
		App:          "demo",
		Env:          EnvStaging,
		Branch:       "main",
		Engine:       EngineStation,
		CurrentImage: "registry/demo:staging-good",
		Mode:         "port",
		HostPort:     3002,
		ServicePort:  3000,
	}); err != nil {
		t.Fatalf("save source state: %v", err)
	}

	_, err := s.enqueueRollbackForLane(RollbackRequest{
		App:          "demo",
		Env:          EnvProd,
		Branch:       "main",
		SourceEnv:    EnvStaging,
		SourceBranch: "main",
	}, "alice", "rollback")
	if err == nil {
		t.Fatalf("expected engine mismatch error")
	}
}

func TestEnsureBaselineLanesSeedsOnlyPreferredLaneByDefault(t *testing.T) {
	s := newPreviewPortTestServer(t)
	if err := s.ensureBaselineLanes("demo", "main", EnvStaging, "https://github.com/acme/demo.git", EngineDocker); err != nil {
		t.Fatalf("ensure baseline lanes: %v", err)
	}

	staging, err := s.getAppState("demo", EnvStaging, "main")
	if err != nil || staging == nil {
		t.Fatalf("expected staging lane state, err=%v", err)
	}

	if staging.RepoURL != "https://github.com/acme/demo.git" {
		t.Fatalf("expected staging repo url to match preferred lane, got %q", staging.RepoURL)
	}
	if prod, err := s.getAppState("demo", EnvProd, "main"); err == nil && prod != nil {
		t.Fatalf("expected prod lane to remain unseeded by default")
	}
	if dev, err := s.getAppState("demo", EnvDev, "main"); err == nil && dev != nil {
		t.Fatalf("expected dev lane to remain unseeded by default")
	}
}
