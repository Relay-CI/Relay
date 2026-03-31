package main

import (
	"runtime"
	"strings"
)

const (
	EngineDocker = "docker"
	EngineStation = "station"
)

// ─── engine normalization ─────────────────────────────────────────────────────

func normalizeEngine(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", EngineDocker:
		return EngineDocker
	case EngineStation:
		return EngineStation
	default:
		return ""
	}
}

// firstNonEmptyEngine returns the first recognizable engine name in values,
// falling back to detectDefaultEngine() when none is set.
func firstNonEmptyEngine(values ...string) string {
	for _, value := range values {
		if normalized := normalizeEngine(value); normalized != "" {
			return normalized
		}
	}
	return detectDefaultEngine()
}

// detectDefaultEngine returns the best available engine for the current host.
//
// Priority:
//   1. Linux (non-WSL): always vessel — it runs natively with no overhead.
//   2. Windows with a healthy vessel daemon: vessel — daemon is already warm.
//   3. Windows with WSL2 present but daemon not yet started: vessel — agent
//      will spin up the daemon on the first build.
//   4. Everything else (no WSL2, degraded state): docker.
func detectDefaultEngine() string {
	if runtime.GOOS != "windows" {
		return EngineStation
	}
	// On Windows: prefer vessel when WSL2 is available.
	// stationAgentHealthy() is a non-blocking probe if the daemon is already up;
	// wslAvailableForAgent() is a cheap fallback that just checks the distro list.
	if stationAgentHealthy() || wslAvailableForAgent() {
		return EngineStation
	}
	return EngineDocker
}

// ─── capability queries ───────────────────────────────────────────────────────

func engineSupportsCompanions(engine string) bool {
	return firstNonEmptyEngine(engine) == EngineDocker
}

func engineSupportsPublicHost(_ string) bool {
	return true
}

func engineSupportsTrafficMode(_, trafficMode string) bool {
	return normalizeTrafficMode(trafficMode) != "" || strings.TrimSpace(trafficMode) == ""
}

func (s *Server) runtimeForEngine(engine string) ContainerRuntime {
	if firstNonEmptyEngine(engine) == EngineStation && s.stationRuntime != nil {
		return s.stationRuntime
	}
	return s.runtime
}

func (s *Server) stopDockerAppLane(app string, env DeployEnv, branch string) {
	s.runtime.Remove(appBaseContainerName(app, env, branch))
	s.runtime.Remove(appSlotContainerName(app, env, branch, "blue"))
	s.runtime.Remove(appSlotContainerName(app, env, branch, "green"))
}

func (s *Server) stopLaneServices(app string, env DeployEnv, branch string) {
	if running, err := s.getProjectServices(app, string(env), branch); err == nil {
		for _, svc := range running {
			s.stopProjectServiceRuntime(app, string(env), branch, svc.Name)
		}
	}
}

func constrainDeployRequestForEngine(engine string, req *DeployRequest) {
	engine = firstNonEmptyEngine(engine)
	if req.TrafficMode == "" {
		req.TrafficMode = "edge"
	}
	if req.Mode == "" {
		if req.PublicHost != "" {
			req.Mode = "traefik"
		} else {
			req.Mode = "port"
		}
	}
}

func constrainAppStateForEngine(st *AppState) {
	if st == nil {
		return
	}
	st.Engine = firstNonEmptyEngine(st.Engine)
	if st.Mode == "" {
		if strings.TrimSpace(st.PublicHost) != "" {
			st.Mode = "traefik"
		} else {
			st.Mode = "port"
		}
	}
	st.TrafficMode = firstNonEmpty(normalizeTrafficMode(st.TrafficMode), "edge")
}

func (s *Server) pruneRuntimeArtifacts(engine string, app string, env DeployEnv, branch string, keep ...string) error {
	if firstNonEmptyEngine(engine) == EngineStation {
		return pruneStationSnapshots(app, env, branch, keep...)
	}
	return s.pruneAppImages(app, env, branch, keep...)
}

