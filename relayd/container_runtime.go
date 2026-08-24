package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Container runtime module: the shared interface and Docker adapter.
// ContainerRuntime abstracts Docker-backed container operations used by relayd.
// Per-lane engine selection can dispatch around this for experimental backends.
type ContainerRuntime interface {
	// RunDetached starts a container in the background.
	RunDetached(spec ContainerSpec) error
	// Remove stops and deletes a container (ignores not-found errors).
	Remove(name string)
	// ContainerExists reports whether the named container or target still exists.
	ContainerExists(name string) bool
	// IsRunning reports whether the named container is currently running.
	IsRunning(name string) bool
	// ContainerIP returns the first IP address found in the container's networks.
	ContainerIP(name string) string
	// PublishedPort returns the host-side port mapped to the given container port.
	PublishedPort(name string, containerPort int) int
	// Exec runs a command inside a running container and returns combined output.
	Exec(container string, cmd []string) ([]byte, error)
	// NetworkConnect attaches a running container to an additional network.
	NetworkConnect(container, network string) error
	// EnsureNetwork creates a bridge network if it does not already exist.
	EnsureNetwork(name string) error
	// RemoveNetwork deletes a network (ignores errors).
	RemoveNetwork(name string)
	// RemoveVolume deletes a named volume (ignores errors).
	RemoveVolume(name string)
	// Pull pulls an image from a registry (ignores errors).
	Pull(image string) error
	// Build builds an image from contextDir tagged as tag.
	// dockerfilePath may be "" to use the default Dockerfile in contextDir.
	// kind is the buildpack kind (e.g. "next-standalone", "go") used to pick
	// an appropriate memory limit for the build container on small hosts.
	Build(ctx context.Context, tag, contextDir, dockerfilePath string, buildArgs map[string]string, logw io.Writer, kind string) error
	// RemoveImage removes an image by reference (ignores errors).
	RemoveImage(ref string)
	// ListImages returns all image refs matching the given repository name.
	ListImages(repo string) ([]string, error)
	// LogStream opens a streaming log reader for the named container.
	LogStream(ctx context.Context, name string, tail int, since string) (io.ReadCloser, error)
}

// ContainerSpec describes a container to launch via ContainerRuntime.RunDetached.
type ContainerSpec struct {
	Name          string
	Image         string
	Network       string
	RestartPolicy string   // "always", "unless-stopped", or "" (no --restart flag)
	Env           []string // "KEY=VALUE" pairs
	Volumes       []string // "source:target[:options]" bindings
	ExtraHosts    []string // "name:ip" aliases to inject via /etc/hosts
	PortBindings  []string // "hostSpec:containerPort" e.g. "127.0.0.1::3000"
	HealthArgs    []string // pre-computed --health-* flag pairs (from healthArgs())
	Command       []string // optional command override
	CPULimit      string   // docker --cpus value e.g. "0.5"
	MemLimit      string   // docker --memory value e.g. "512m"
}

// DockerRuntime implements ContainerRuntime by calling the local Docker CLI.
type DockerRuntime struct{}

func (r *DockerRuntime) RunDetached(spec ContainerSpec) error {
	args := []string{"run", "-d", "--name", spec.Name}
	if spec.RestartPolicy != "" {
		args = append(args, "--restart="+spec.RestartPolicy)
	}
	if spec.Network != "" {
		args = append(args, "--network", spec.Network)
	}
	for _, e := range spec.Env {
		args = append(args, "-e", e)
	}
	for _, v := range spec.Volumes {
		args = append(args, "-v", v)
	}
	for _, host := range spec.ExtraHosts {
		args = append(args, "--add-host", host)
	}
	for _, p := range spec.PortBindings {
		args = append(args, "-p", p)
	}
	if spec.CPULimit != "" {
		args = append(args, "--cpus="+spec.CPULimit)
	}
	if spec.MemLimit != "" {
		args = append(args, "--memory="+spec.MemLimit)
	}
	args = append(args, spec.HealthArgs...)
	args = append(args, spec.Image)
	args = append(args, spec.Command...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run %s: %v — %s", spec.Name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *DockerRuntime) Remove(name string) {
	_ = exec.Command("docker", "rm", "-f", name).Run()
}

func (r *DockerRuntime) ContainerExists(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	out, err := exec.Command("docker", "inspect", name).CombinedOutput()
	if err != nil {
		return false
	}
	return len(out) > 0
}

func (r *DockerRuntime) IsRunning(name string) bool {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", name).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func (r *DockerRuntime) ContainerIP(name string) string {
	out, err := exec.Command("docker", "inspect", "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", name).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r *DockerRuntime) PublishedPort(name string, containerPort int) int {
	format := fmt.Sprintf("{{with index .NetworkSettings.Ports %q}}{{(index . 0).HostPort}}{{end}}", fmt.Sprintf("%d/tcp", firstNonZero(containerPort, 3000)))
	out, err := exec.Command("docker", "inspect", "--format", format, name).CombinedOutput()
	if err != nil {
		return 0
	}
	port, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return port
}

func (r *DockerRuntime) Exec(container string, cmd []string) ([]byte, error) {
	args := append([]string{"exec", container}, cmd...)
	return exec.Command("docker", args...).CombinedOutput()
}

func (r *DockerRuntime) NetworkConnect(container, network string) error {
	out, err := exec.Command("docker", "network", "connect", network, container).CombinedOutput()
	if err != nil && strings.Contains(strings.ToLower(string(out)), "already exists in network") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("docker network connect %s %s: %w (%s)", network, container, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *DockerRuntime) EnsureNetwork(name string) error {
	out, _ := exec.Command("docker", "network", "inspect", name).CombinedOutput()
	if strings.Contains(string(out), `"Name"`) {
		return nil
	}
	out, err := exec.Command("docker", "network", "create", "--driver", "bridge", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker network create: %v — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *DockerRuntime) RemoveNetwork(name string) {
	_ = exec.Command("docker", "network", "rm", name).Run()
}

func (r *DockerRuntime) RemoveVolume(name string) {
	if strings.TrimSpace(name) == "" {
		return
	}
	_ = exec.Command("docker", "volume", "rm", "-f", name).Run()
}

func (r *DockerRuntime) Pull(image string) error {
	if strings.TrimSpace(image) == "" {
		return nil
	}
	return exec.Command("docker", "pull", image).Run()
}

func (r *DockerRuntime) Build(ctx context.Context, tag, contextDir, dockerfilePath string, buildArgs map[string]string, logw io.Writer, kind string) error {
	args := []string{"build", "-t", tag}
	if dockerfilePath != "" {
		args = append(args, "-f", dockerfilePath)
	}
	// Buildpack-aware memory limit: Node/Next builds fork heavily during npm
	// install and Turbopack allocates outside the V8 heap, so they get 85% of
	// host RAM on small hosts. Other buildpacks are sized proportionally. A flat
	// 60% was causing spawn ENOMEM on 2 GB hosts because the container hit its
	// cgroup limit while forking npm child processes.
	// --memory-swap=-1 lets the build spill into any swap the host has instead
	// of hard-failing at the RAM cap; swap is provisioned automatically by
	// maybeEnableAutoSwap before the build starts.
	// RELAY_BUILD_MEMORY_LIMIT overrides the automatic sizing.
	if mem := strings.TrimSpace(os.Getenv("RELAY_BUILD_MEMORY_LIMIT")); mem != "" {
		args = append(args, "--memory="+mem)
		args = append(args, "--memory-swap=-1")
	} else if limitMB := buildMemLimitMB(kind); limitMB > 0 {
		args = append(args, fmt.Sprintf("--memory=%dm", limitMB))
		args = append(args, "--memory-swap=-1")
	}
	if len(buildArgs) > 0 {
		keys := make([]string, 0, len(buildArgs))
		for key := range buildArgs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, buildArgs[key]))
		}
	}
	args = append(args, ".")
	return runCmdLoggedEnvCtx(ctx, contextDir, logw, []string{"DOCKER_BUILDKIT=1"}, "docker", args...)
}

func (r *DockerRuntime) RemoveImage(ref string) {
	if strings.TrimSpace(ref) == "" {
		return
	}
	_ = exec.Command("docker", "image", "rm", "-f", ref).Run()
}

func (r *DockerRuntime) ListImages(repo string) ([]string, error) {
	out, err := exec.Command("docker", "images", repo, "--format", "{{.Repository}}:{{.Tag}}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list images %s: %v (%s)", repo, err, strings.TrimSpace(string(out)))
	}
	var result []string
	for _, line := range strings.Split(string(out), "\n") {
		img := strings.TrimSpace(line)
		if img != "" && img != "<none>:<none>" {
			result = append(result, img)
		}
	}
	return result, nil
}

func (r *DockerRuntime) LogStream(ctx context.Context, name string, tail int, since string) (io.ReadCloser, error) {
	args := []string{"logs", "--timestamps", "--tail", strconv.Itoa(tail), "-f"}
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, name)
	cmd := exec.CommandContext(ctx, "docker", args...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return nil, err
	}
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
	}()
	return pr, nil
}

// ─────────────────────────────────────────────────────────────────────────────
