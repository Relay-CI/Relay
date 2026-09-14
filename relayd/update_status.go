package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// laneReleaseVersion exposes an opaque comparison token rather than an
// internal image tag, repository URL, or commit.
func laneReleaseVersion(app string, env DeployEnv, branch string, image string) string {
	if strings.TrimSpace(image) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(app + "\n" + string(env) + "\n" + branch + "\n" + image))
	return hex.EncodeToString(sum[:])[:16]
}

// handlePublicUpdate lets an app detect that Relay switched its lane to a new
// release. Public lanes are callable from browser code. Protected lanes keep
// their existing edge authorization policy.
func (s *Server) handlePublicUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	app := strings.TrimSpace(r.URL.Query().Get("app"))
	env := normalizeDeployEnv(r.URL.Query().Get("env"))
	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	if !validDeployTarget(app, env, branch) {
		httpError(w, http.StatusBadRequest, "app, env, branch required")
		return
	}
	state, err := s.getAppState(app, env, branch)
	if err != nil || state == nil {
		httpError(w, http.StatusNotFound, "lane not found")
		return
	}
	policy := firstNonEmpty(normalizeAccessPolicy(state.AccessPolicy), s.lanePolicy(env).DefaultAccessPolicy)
	status, message := s.authorizeEdgeAccess(r, app, env, branch, state.AccessPolicy, state.IPAllowlist)
	if status != http.StatusNoContent {
		httpError(w, status, message)
		return
	}
	if policy == AccessPolicyPublic {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	version := laneReleaseVersion(app, env, branch, state.CurrentImage)
	current := strings.TrimSpace(r.URL.Query().Get("current"))
	if len(current) > 128 {
		httpError(w, http.StatusBadRequest, "current version is too long")
		return
	}
	updateAvailable := current != "" && version != "" && current != version
	writeJSON(w, http.StatusOK, map[string]any{
		"version":             version,
		"ready":               version != "" && !state.Stopped,
		"update_available":    updateAvailable,
		"refresh_recommended": updateAvailable,
		"poll_after_seconds":  30,
	})
}
