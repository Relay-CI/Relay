package main

import (
	"strings"
)

const (
	EngineDocker  = "docker"
	EngineStation = "station"
)

// ─── engine normalization ─────────────────────────────────────────────────────

func normalizeEngine(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return EngineStation
	case EngineDocker:
		return EngineDocker
	case EngineStation, "vessel":
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

// detectDefaultEngine returns the default engine when none is configured.
//
// Relay defaults to the Station runtime across platforms. Docker remains
// available when explicitly selected in app configuration.
func detectDefaultEngine() string {
	return EngineStation
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
