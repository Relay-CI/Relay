package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// applyLaneRollout records the desired Lane State before changing the public
// route. If the process stops after the route change but before the state
// write, the durable intent lets the reconciler finish the same transition.
func (s *Server) applyLaneRollout(planned *AppState, switchRoute func() error) error {
	encoded, err := json.Marshal(planned)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO lane_rollout_intents(app,env,branch,planned_state,created_at)
		VALUES(?,?,?,?,?) ON CONFLICT(app,env,branch) DO UPDATE SET
		planned_state=excluded.planned_state,created_at=excluded.created_at`,
		planned.App, string(planned.Env), planned.Branch, string(encoded), time.Now().UnixMilli())
	if err != nil {
		return fmt.Errorf("record rollout intent: %w", err)
	}
	if err := switchRoute(); err != nil {
		_, _ = s.db.Exec(`DELETE FROM lane_rollout_intents WHERE app=? AND env=? AND branch=?`, planned.App, string(planned.Env), planned.Branch)
		return err
	}
	if err := s.saveAppState(planned); err != nil {
		// Keep the intent and candidate: removing a now-live slot would turn a
		// transient SQLite failure into an outage. The reconciler retries.
		return fmt.Errorf("route changed; Lane State pending reconciliation: %w", err)
	}
	_, err = s.db.Exec(`DELETE FROM lane_rollout_intents WHERE app=? AND env=? AND branch=?`, planned.App, string(planned.Env), planned.Branch)
	if err != nil {
		return fmt.Errorf("Lane State saved; clear rollout intent: %w", err)
	}
	return nil
}

func (s *Server) runRolloutIntentReconciler() {
	for {
		s.reconcileLaneRolloutIntents()
		time.Sleep(10 * time.Second)
	}
}

func (s *Server) reconcileLaneRolloutIntents() {
	rows, err := s.db.Query(`SELECT planned_state FROM lane_rollout_intents ORDER BY created_at`)
	if err != nil {
		return
	}
	var planned []AppState
	for rows.Next() {
		var raw string
		if rows.Scan(&raw) == nil {
			var st AppState
			if json.Unmarshal([]byte(raw), &st) == nil && validDeployTarget(st.App, st.Env, st.Branch) {
				planned = append(planned, st)
			}
		}
	}
	_ = rows.Close()
	for i := range planned {
		st := &planned[i]
		lock := s.edgeProxyLock(st.App, st.Env, st.Branch)
		lock.Lock()
		s.reconcileLaneRolloutIntent(st)
		lock.Unlock()
	}
}

func (s *Server) reconcileLaneRolloutIntent(planned *AppState) {
	runtime := s.runtimeForEngine(planned.Engine)
	if !runtime.IsRunning(appSlotContainerName(planned.App, planned.Env, planned.Branch, planned.ActiveSlot)) {
		return
	}
	current, err := s.getAppState(planned.App, planned.Env, planned.Branch)
	if err != nil {
		return
	}
	if current != nil && current.ActiveSlot == planned.ActiveSlot && current.StandbySlot == planned.StandbySlot {
		_, _ = s.db.Exec(`DELETE FROM lane_rollout_intents WHERE app=? AND env=? AND branch=?`, planned.App, string(planned.Env), planned.Branch)
		return
	}
	if current != nil {
		planned.GitToken = current.GitToken
	}
	if firstNonEmptyEngine(planned.Engine) == EngineStation {
		err = s.ensurestationEdgeProxy(nil, planned.App, planned.Env, planned.Branch, planned.ActiveSlot, planned.StandbySlot, planned.ServicePort, planned.HostPort, planned.Mode, planned.TrafficMode, planned.PublicHost, false)
	} else {
		err = s.ensureEdgeProxyLocked(nil, planned.App, planned.Env, planned.Branch, appNetworkName(planned.App, planned.Env, planned.Branch), planned.ActiveSlot, planned.StandbySlot, planned.ServicePort, planned.HostPort, planned.Mode, planned.TrafficMode, planned.PublicHost, planned.TrafficSplitPercent, false)
	}
	if err != nil {
		return
	}
	if err = s.saveAppState(planned); err != nil {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM lane_rollout_intents WHERE app=? AND env=? AND branch=?`, planned.App, string(planned.Env), planned.Branch)
	s.broadcastSnapshot()
	if planned.StandbySlot != "" && planned.TrafficMode == "session" {
		s.startSessionDrain(planned.App, planned.Env, planned.Branch, planned.ActiveSlot, planned.StandbySlot)
	}
}
