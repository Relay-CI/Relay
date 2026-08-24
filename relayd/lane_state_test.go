package main

import (
	"reflect"
	"testing"
)

func fullyConfiguredLaneState() *AppState {
	return &AppState{
		App:                  "demo",
		Env:                  EnvProd,
		Branch:               "main",
		RepoURL:              "https://example.com/repo.git",
		ProjectRoot:          "apps/web",
		BuildContext:         ".",
		Dockerfile:           "Dockerfile.prod",
		Engine:               EngineDocker,
		CurrentImage:         "relay/demo:old",
		PreviousImage:        "relay/demo:older",
		Mode:                 "port",
		HostPort:             9443,
		HostPortExplicit:     true,
		ServicePort:          8080,
		PublicHost:           "demo.example.com",
		PublicHosts:          []string{"demo.example.com", "www.example.com"},
		ActiveSlot:           "green",
		StandbySlot:          "blue",
		DrainUntil:           12345,
		TrafficMode:          "edge",
		AccessPolicy:         "private",
		IPAllowlist:          "203.0.113.0/24",
		ExpiresAt:            54321,
		RepoHash:             "old-hash",
		WebhookSecret:        "webhook-secret",
		NotificationWebhooks: "https://hooks.example.com/relay",
		TrafficSplitPercent:  25,
		RolloutMinRequests:   50,
		RolloutErrorPercent:  2.5,
		RolloutAssessSeconds: 600,
		RolloutStartedAt:     111,
		RolloutDeployID:      "deploy-old",
		RolloutStatus:        "monitoring",
		Stopped:              true,
		CPULimit:             "1.5",
		MemLimit:             "768m",
		ResourceMode:         "manual",
		Volumes:              []string{"relay-data:/data"},
		BuildpackKind:        "node",
		GitToken:             "git-secret",
	}
}

func TestBuildSuccessfulLaneStatePreservesStickyConfiguration(t *testing.T) {
	current := fullyConfiguredLaneState()
	got := buildSuccessfulLaneState(successfulLaneTransition{
		Request: DeployRequest{
			App:         current.App,
			Env:         current.Env,
			Branch:      current.Branch,
			RepoURL:     "https://example.com/new-repo.git",
			HostPort:    current.HostPort,
			ServicePort: current.ServicePort,
		},
		Current:            current,
		Engine:             EngineStation,
		BuildpackKind:      "go",
		CurrentImage:       "relay/demo:new",
		PreviousImage:      current.CurrentImage,
		RepoHash:           "new-hash",
		ActiveSlotFallback: "blue",
		DeployID:           "deploy-new",
		Stopped:            false,
	})

	want := *current
	want.RepoURL = "https://example.com/new-repo.git"
	want.Engine = EngineStation
	want.BuildpackKind = "go"
	want.CurrentImage = "relay/demo:new"
	want.PreviousImage = current.CurrentImage
	want.RepoHash = "new-hash"
	want.RolloutDeployID = "deploy-new"
	want.Stopped = false

	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("successful transition dropped or changed lane state\n got: %#v\nwant: %#v", *got, want)
	}
	got.PublicHosts[0] = "changed.example.com"
	got.Volumes[0] = "changed:/data"
	if current.PublicHosts[0] != "demo.example.com" || current.Volumes[0] != "relay-data:/data" {
		t.Fatal("transition result aliases sticky slices from the current state")
	}
}

func TestBuildRollbackLaneStatePreservesStickyConfiguration(t *testing.T) {
	current := fullyConfiguredLaneState()
	got := buildRollbackLaneState(rollbackLaneTransition{
		Request: DeployRequest{
			App:    current.App,
			Env:    current.Env,
			Branch: current.Branch,
		},
		Current:            current,
		Engine:             EngineDocker,
		CurrentImage:       current.PreviousImage,
		PreviousImage:      current.CurrentImage,
		ActiveSlotFallback: "blue",
	})

	want := *current
	want.CurrentImage = current.PreviousImage
	want.PreviousImage = current.CurrentImage
	want.Stopped = false

	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("rollback transition dropped or changed lane state\n got: %#v\nwant: %#v", *got, want)
	}
}
