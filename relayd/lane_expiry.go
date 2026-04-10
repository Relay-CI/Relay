package main

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *Server) refreshLaneExpiry(st *AppState, now time.Time) {
	if st == nil {
		return
	}
	switch st.Env {
	case EnvDev, EnvPreview:
		policy := s.lanePolicy(st.Env)
		if policy.RetentionHours > 0 {
			st.ExpiresAt = now.Add(time.Duration(policy.RetentionHours) * time.Hour).UnixMilli()
			return
		}
	}
	st.ExpiresAt = 0
}

func (s *Server) runLaneExpiryWorker() {
	if s == nil || s.db == nil {
		return
	}
	s.expireLanesOnce()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.expireLanesOnce()
	}
}

func (s *Server) expireLanesOnce() {
	rows, err := s.db.Query(
		`SELECT app, env, branch
		 FROM app_state
		 WHERE env IN (?, ?) AND COALESCE(expires_at,0) > 0 AND COALESCE(expires_at,0) <= ? AND COALESCE(stopped,0)=0`,
		string(EnvDev), string(EnvPreview), time.Now().UnixMilli(),
	)
	if err != nil {
		return
	}
	defer rows.Close()

	expired := make([]RollbackRequest, 0, 4)
	for rows.Next() {
		var app string
		var env string
		var branch string
		if scanErr := rows.Scan(&app, &env, &branch); scanErr != nil {
			continue
		}
		expired = append(expired, RollbackRequest{App: app, Env: DeployEnv(env), Branch: branch})
	}

	changed := false
	for _, target := range expired {
		st, err := s.getAppState(target.App, target.Env, target.Branch)
		if err != nil || st == nil {
			continue
		}
		s.stopDockerAppLane(target.App, target.Env, target.Branch)
		_ = s.stopStationLane(target.App, target.Env, target.Branch)
		s.stopLaneServices(target.App, target.Env, target.Branch)
		st.Stopped = true
		st.ExpiresAt = 0
		s.constrainAppState(st)
		if saveErr := s.saveAppState(st); saveErr != nil {
			continue
		}
		changed = true
		s.auditLog("system", "lane.expire", target.App, fmt.Sprintf("env=%s branch=%s", target.Env, target.Branch))
	}
	if changed {
		s.broadcastSnapshot()
	}
}

func scanNullableInt64(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}
