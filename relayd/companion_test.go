package main

import (
	"errors"
	"testing"
)

type failingCompanionRuntime struct {
	mockRuntime
	ensureErr error
	runErr    error
}

func (r *failingCompanionRuntime) EnsureNetwork(string) error {
	return r.ensureErr
}

func (r *failingCompanionRuntime) RunDetached(ContainerSpec) error {
	return r.runErr
}

func TestOrchestrateCompanionsStartsAndPersistsService(t *testing.T) {
	s := newPreviewPortTestServer(t)
	result, err := s.orchestrateCompanions(
		nil,
		"demo",
		EnvDev,
		"main",
		[]ServiceConfig{{Name: "cache", Type: "redis"}},
		false,
		true,
		companionFailureFatal,
	)
	if err != nil {
		t.Fatalf("orchestrate companions: %v", err)
	}
	if result.NetworkName != "relay-demo-dev-main" {
		t.Fatalf("network = %q", result.NetworkName)
	}
	if result.Environment["CACHE_URL"] == "" {
		t.Fatalf("missing companion environment: %#v", result.Environment)
	}
	stored, err := s.getProjectService("demo", string(EnvDev), "main", "cache")
	if err != nil {
		t.Fatalf("get persisted companion: %v", err)
	}
	if stored.Type != "redis" || stored.EnvKey != "CACHE_URL" {
		t.Fatalf("unexpected persisted companion: %#v", stored)
	}
}

func TestOrchestrateCompanionsKeepsCallerFailurePolicy(t *testing.T) {
	service := []ServiceConfig{{Name: "cache", Type: "redis"}}

	t.Run("manual operations fail", func(t *testing.T) {
		s := newPreviewPortTestServer(t)
		runtime := &failingCompanionRuntime{
			mockRuntime: mockRuntime{running: map[string]bool{}, exists: map[string]bool{}, published: map[string]int{}},
			ensureErr:   errors.New("network unavailable"),
		}
		s.runtime = runtime
		if _, err := s.orchestrateCompanions(nil, "demo", EnvDev, "main", service, false, true, companionFailureFatal); err == nil {
			t.Fatal("expected fatal orchestration error")
		}
	})

	t.Run("deploy logs and continues", func(t *testing.T) {
		s := newPreviewPortTestServer(t)
		runtime := &failingCompanionRuntime{
			mockRuntime: mockRuntime{running: map[string]bool{}, exists: map[string]bool{}, published: map[string]int{}},
			ensureErr:   errors.New("network unavailable"),
		}
		s.runtime = runtime
		if _, err := s.orchestrateCompanions(nil, "demo", EnvDev, "main", service, false, true, companionFailureWarning); err != nil {
			t.Fatalf("warning policy returned error: %v", err)
		}
	})
}
