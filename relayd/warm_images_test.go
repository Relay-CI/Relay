package main

import "testing"

func TestWarmBuildpackBaseImagesListDefaults(t *testing.T) {
	images := warmBuildpackBaseImagesList()
	want := map[string]bool{
		"node:22":      false,
		"python:3.12":  false,
		"ruby:3.3":     false,
		"golang:1.22":  false,
		"nginx:alpine": false,
		"php:8.3-cli":  false,
	}
	for _, image := range images {
		if _, ok := want[image]; ok {
			want[image] = true
		}
	}
	for image, found := range want {
		if !found {
			t.Fatalf("expected default image list to include %q, got %v", image, images)
		}
	}
}

func TestWarmBuildpackBaseImagesListRespectsEnvOverride(t *testing.T) {
	t.Setenv("RELAY_NODE_IMAGE", "node:20")
	images := warmBuildpackBaseImagesList()
	for _, image := range images {
		if image == "node:22" {
			t.Fatalf("expected RELAY_NODE_IMAGE override to replace node:22, got list %v", images)
		}
	}
	found := false
	for _, image := range images {
		if image == "node:20" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected overridden node:20 in list, got %v", images)
	}
}
