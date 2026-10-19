package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Replicas: a lane can run N identical containers per slot. Every replica of
// a slot joins the lane network under one shared pool alias, and the edge
// proxy targets that alias, so Docker's embedded DNS hands nginx every
// replica's address and nginx round-robins across them (retrying the next
// replica when one refuses or 502s). Replica 1 keeps the historical slot
// container name so logs, health checks and rollout code keep working.

const maxLaneReplicas = 16

// laneTopology is the per-lane scaling/experiment config stored alongside
// app_state (kept in its own table so app_state's wide scan stays untouched).
type laneTopology struct {
	Replicas    int            `json:"replicas"`
	SlotPools   map[string]int `json:"slot_pools"`
	VariantSeed int            `json:"variant_seed"`
}

func normalizeReplicaCount(n int) int {
	if n < 1 {
		return 1
	}
	if n > maxLaneReplicas {
		return maxLaneReplicas
	}
	return n
}

func (s *Server) getLaneTopology(app string, env DeployEnv, branch string) laneTopology {
	topo := laneTopology{Replicas: 1, SlotPools: map[string]int{}, VariantSeed: 1}
	if s == nil || s.db == nil {
		return topo
	}
	var pools string
	err := s.db.QueryRow(
		`SELECT replicas, slot_pools, variant_seed FROM lane_topology WHERE app=? AND env=? AND branch=?`,
		app, string(env), branch,
	).Scan(&topo.Replicas, &pools, &topo.VariantSeed)
	if err != nil {
		return topo
	}
	topo.Replicas = normalizeReplicaCount(topo.Replicas)
	if topo.VariantSeed < 1 {
		topo.VariantSeed = 1
	}
	_ = json.Unmarshal([]byte(pools), &topo.SlotPools)
	if topo.SlotPools == nil {
		topo.SlotPools = map[string]int{}
	}
	return topo
}

func (s *Server) saveLaneTopology(app string, env DeployEnv, branch string, topo laneTopology) error {
	if s == nil || s.db == nil {
		return nil
	}
	if topo.SlotPools == nil {
		topo.SlotPools = map[string]int{}
	}
	pools, _ := json.Marshal(topo.SlotPools)
	_, err := s.db.Exec(
		`INSERT INTO lane_topology (app, env, branch, replicas, slot_pools, variant_seed, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(app, env, branch) DO UPDATE SET
		   replicas=excluded.replicas, slot_pools=excluded.slot_pools,
		   variant_seed=excluded.variant_seed, updated_at=excluded.updated_at`,
		app, string(env), branch, normalizeReplicaCount(topo.Replicas), string(pools), max(topo.VariantSeed, 1), time.Now().UnixMilli(),
	)
	return err
}

func (s *Server) laneReplicas(app string, env DeployEnv, branch string) int {
	return s.getLaneTopology(app, env, branch).Replicas
}

func (s *Server) setLaneReplicas(app string, env DeployEnv, branch string, replicas int) error {
	topo := s.getLaneTopology(app, env, branch)
	topo.Replicas = normalizeReplicaCount(replicas)
	return s.saveLaneTopology(app, env, branch, topo)
}

// slotPoolSize is how many replicas were launched for slot under the pool
// alias. 0 means the slot predates replicas (or was never launched) and must
// be addressed by its container name instead of the alias.
func (s *Server) slotPoolSize(app string, env DeployEnv, branch string, slot string) int {
	return s.getLaneTopology(app, env, branch).SlotPools[slot]
}

func (s *Server) setSlotPoolSize(app string, env DeployEnv, branch string, slot string, n int) {
	topo := s.getLaneTopology(app, env, branch)
	if n <= 0 {
		delete(topo.SlotPools, slot)
	} else {
		topo.SlotPools[slot] = n
	}
	_ = s.saveLaneTopology(app, env, branch, topo)
}

func appSlotPoolAlias(app string, env DeployEnv, branch string, slot string) string {
	return appSlotContainerName(app, env, branch, slot) + "-pool"
}

// appSlotReplicaName returns replica i (1-based) of slot. Replica 1 is the
// plain slot container so everything that predates replicas still finds it.
func appSlotReplicaName(app string, env DeployEnv, branch string, slot string, i int) string {
	base := appSlotContainerName(app, env, branch, slot)
	if i <= 1 {
		return base
	}
	return base + "__r" + strconv.Itoa(i)
}

// slotUpstreamHost is the hostname the edge proxy should send slot traffic
// to: the pool alias for pooled slots, the container name for legacy ones.
func (s *Server) slotUpstreamHost(app string, env DeployEnv, branch string, slot string) string {
	if s.slotPoolSize(app, env, branch, slot) > 0 {
		return appSlotPoolAlias(app, env, branch, slot)
	}
	return appSlotContainerName(app, env, branch, slot)
}

// slotReplicaNames lists every replica container that may exist for slot.
func (s *Server) slotReplicaNames(app string, env DeployEnv, branch string, slot string) []string {
	n := max(s.slotPoolSize(app, env, branch, slot), 1)
	names := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		names = append(names, appSlotReplicaName(app, env, branch, slot, i))
	}
	return names
}

// removeSlotContainers removes every replica of slot and forgets its pool.
func (s *Server) removeSlotContainers(runtime ContainerRuntime, app string, env DeployEnv, branch string, slot string) {
	if runtime == nil {
		runtime = s.runtime
	}
	for _, name := range s.slotReplicaNames(app, env, branch, slot) {
		runtime.Remove(name)
	}
	if s.slotPoolSize(app, env, branch, slot) > 0 {
		s.setSlotPoolSize(app, env, branch, slot, 0)
	}
}

// waitForSlotReplicasReady waits for every replica of slot. A slot only takes
// traffic once all of its replicas answer, so a partial pool never goes live.
func (s *Server) waitForSlotReplicasReady(runtime ContainerRuntime, log func(string, ...any), app string, env DeployEnv, branch string, slot string, port int, timeout time.Duration, readinessPath string) error {
	var errs []error
	for _, name := range s.slotReplicaNames(app, env, branch, slot) {
		if err := s.waitForRuntimeContainerReadyOnPath(runtime, log, name, port, timeout, readinessPath); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

// activeReplicaCount is the number of app containers Relay expects to keep
// running across all live lanes (each lane's replica count, summed). The
// default memory cap is divided by this, so scaling a lane up shrinks the
// per-container share instead of overcommitting the host.
func (s *Server) activeReplicaCount() int {
	if s == nil || s.db == nil {
		return 1
	}
	var count sql.NullInt64
	err := s.db.QueryRow(
		`SELECT SUM(COALESCE(t.replicas, 1))
		   FROM app_state a
		   LEFT JOIN lane_topology t ON t.app=a.app AND t.env=a.env AND t.branch=a.branch
		  WHERE COALESCE(a.stopped,0)=0`,
	).Scan(&count)
	if err != nil || !count.Valid || count.Int64 < 1 {
		return 1
	}
	return int(count.Int64)
}

// replicaContainerSpec derives replica i's spec from replica 1's.
func replicaContainerSpec(base ContainerSpec, name string, poolAlias string) ContainerSpec {
	spec := base
	spec.Name = name
	spec.NetworkAliases = append(append([]string{}, base.NetworkAliases...), poolAlias)
	return spec
}

func configReplicas(cfg *RelayConfig) int {
	if cfg == nil {
		return 0
	}
	return cfg.Replicas
}

// handleAppScale sets a lane's replica count. A running Docker lane rolls the
// change out immediately by redeploying its current image blue/green-style
// (new pool starts, passes readiness, then takes traffic), keeping the
// lane's rollback target unchanged.
func (s *Server) handleAppScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var body struct {
		App      string `json:"app"`
		Env      string `json:"env"`
		Branch   string `json:"branch"`
		Replicas int    `json:"replicas"`
		Apply    *bool  `json:"apply"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	env := normalizeDeployEnv(body.Env)
	branch := firstNonEmpty(strings.TrimSpace(body.Branch), "main")
	if !validDeployTarget(body.App, env, branch) {
		httpError(w, 400, "app, env and branch required")
		return
	}
	if body.Replicas < 1 || body.Replicas > maxLaneReplicas {
		httpError(w, 400, fmt.Sprintf("replicas must be between 1 and %d", maxLaneReplicas))
		return
	}
	if _, ok := s.requireLaneAccess(w, r, body.App, env, "deployer"); !ok {
		return
	}
	st, err := s.getAppState(body.App, env, branch)
	if err != nil || st == nil {
		httpError(w, 404, "lane not found")
		return
	}
	if firstNonEmptyEngine(st.Engine) != EngineDocker {
		httpError(w, 400, "replicas require the docker engine")
		return
	}
	if err := s.setLaneReplicas(body.App, env, branch, body.Replicas); err != nil {
		httpError(w, 500, "failed to save replica count")
		return
	}
	actor := requestActorLabel(s, r)
	s.auditLog(actor, "app.scale", body.App, fmt.Sprintf("env=%s branch=%s replicas=%d", env, branch, body.Replicas))
	resp := map[string]any{"ok": true, "replicas": normalizeReplicaCount(body.Replicas)}
	apply := body.Apply == nil || *body.Apply
	active := normalizeActiveSlot(st.ActiveSlot)
	running := s.slotPoolSize(body.App, env, branch, active)
	if apply && !st.Stopped && active != "" && strings.TrimSpace(st.CurrentImage) != "" && running != normalizeReplicaCount(body.Replicas) {
		deployID, err := s.enqueueImageRedeploy(body.App, env, branch, st.CurrentImage, actor, "scale", true)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		resp["deploy_id"] = deployID
	}
	s.broadcastSnapshot()
	writeJSON(w, 200, resp)
}
