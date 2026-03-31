package main

import "testing"

func TestMergeContainerExtraEnvPreservesProtectedKeys(t *testing.T) {
	env := map[string]string{
		"PORT":         "50075",
		"APP_NAME":     "demo-app",
		"CONTAINER_ID": "abc123",
	}
	extra := map[string]string{
		"PORT":                    "3000",
		"APP_NAME":                "wrong-app",
		"CONTAINER_ID":            "wrong-id",
		"NODE_ENV":                "production",
		"NEXT_TELEMETRY_DISABLED": "1",
	}

	keys := mergeContainerExtraEnv(env, extra)

	if got := env["PORT"]; got != "50075" {
		t.Fatalf("expected runtime PORT to win, got %q", got)
	}
	if got := env["APP_NAME"]; got != "demo-app" {
		t.Fatalf("expected APP_NAME to be preserved, got %q", got)
	}
	if got := env["CONTAINER_ID"]; got != "abc123" {
		t.Fatalf("expected CONTAINER_ID to be preserved, got %q", got)
	}
	if got := env["NODE_ENV"]; got != "production" {
		t.Fatalf("expected NODE_ENV to be merged, got %q", got)
	}
	if got := env["NEXT_TELEMETRY_DISABLED"]; got != "1" {
		t.Fatalf("expected NEXT_TELEMETRY_DISABLED to be merged, got %q", got)
	}
	if len(keys) != 2 || keys[0] != "NEXT_TELEMETRY_DISABLED" || keys[1] != "NODE_ENV" {
		t.Fatalf("unexpected forwarded keys: %#v", keys)
	}
}
