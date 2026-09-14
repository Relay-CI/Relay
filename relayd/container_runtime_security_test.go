package main

import (
	"slices"
	"testing"
)

func TestDockerRunArgsIncludesContainerSandbox(t *testing.T) {
	args := dockerRunArgs(ContainerSpec{
		Name:              "candidate",
		Image:             "example/app:latest",
		NoNewPrivileges:   true,
		DropCapabilities:  []string{"ALL"},
		ReadOnlyRootFS:    true,
		PIDsLimit:         256,
		User:              "10001:10001",
		Tmpfs:             []string{"/tmp:rw,noexec,nosuid,size=64m"},
	})
	for _, expected := range []string{
		"--security-opt=no-new-privileges:true",
		"--cap-drop=ALL",
		"--read-only",
		"--pids-limit=256",
		"--user=10001:10001",
		"--tmpfs",
		"/tmp:rw,noexec,nosuid,size=64m",
	} {
		if !slices.Contains(args, expected) {
			t.Fatalf("missing %q in docker args: %v", expected, args)
		}
	}
	if args[len(args)-1] != "example/app:latest" {
		t.Fatalf("image must remain the final argument when no command is configured: %v", args)
	}
}
