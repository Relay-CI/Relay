package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	promotionStatusPendingApproval = "pending_approval"
	promotionStatusApproved        = "approved"
	promotionStatusRunning         = "running"
	promotionStatusSuccess         = "success"
	promotionStatusFailed          = "failed"
	promotionStatusRolledBack      = "rolled_back"
)

type PromotionRecord struct {
	ID               string `json:"id"`
	App              string `json:"app"`
	SourceEnv        string `json:"source_env"`
	SourceBranch     string `json:"source_branch"`
	SourceDeployID   string `json:"source_deploy_id,omitempty"`
	SourceImage      string `json:"source_image,omitempty"`
	TargetEnv        string `json:"target_env"`
	TargetBranch     string `json:"target_branch"`
	Status           string `json:"status"`
	ApprovalRequired bool   `json:"approval_required"`
	RequestedBy      string `json:"requested_by,omitempty"`
	RequestedAt      int64  `json:"requested_at"`
	ApprovedBy       string `json:"approved_by,omitempty"`
	ApprovedAt       int64  `json:"approved_at,omitempty"`
	TargetDeployID   string `json:"target_deploy_id,omitempty"`
	RollbackDeployID string `json:"rollback_deploy_id,omitempty"`
	HealthStatus     string `json:"health_status,omitempty"`
	HealthDetail     string `json:"health_detail,omitempty"`
}

type promotionRequestBody struct {
	App          string    `json:"app"`
	SourceEnv    DeployEnv `json:"source_env"`
	SourceBranch string    `json:"source_branch"`
	TargetEnv    DeployEnv `json:"target_env"`
	TargetBranch string    `json:"target_branch"`
}

func (s *Server) handlePromotions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app := strings.TrimSpace(r.URL.Query().Get("app"))
		sourceEnv := normalizeDeployEnv(r.URL.Query().Get("source_env"))
		targetEnv := normalizeDeployEnv(r.URL.Query().Get("target_env"))
		branch := strings.TrimSpace(r.URL.Query().Get("branch"))
		items, err := s.listPromotions(app, sourceEnv, targetEnv, branch)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, items)
	case http.MethodPost:
		var body promotionRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		body.SourceEnv = normalizeDeployEnv(string(body.SourceEnv))
		body.TargetEnv = normalizeDeployEnv(string(body.TargetEnv))
		body.App = strings.TrimSpace(body.App)
		body.SourceBranch = strings.TrimSpace(body.SourceBranch)
		body.TargetBranch = strings.TrimSpace(body.TargetBranch)
		if !validDeployTarget(body.App, body.SourceEnv, body.SourceBranch) {
			httpError(w, 400, "app, source_env, and source_branch are required")
			return
		}
		if body.TargetEnv == "" {
			body.TargetEnv = normalizeDeployEnv(s.lanePolicy(body.SourceEnv).PromoteTo)
		}
		if body.TargetBranch == "" {
			body.TargetBranch = body.SourceBranch
		}
		if !isKnownDeployEnv(body.TargetEnv) || strings.TrimSpace(body.TargetBranch) == "" {
			httpError(w, 400, "target_env and target_branch are required")
			return
		}
		if body.SourceEnv == body.TargetEnv && body.SourceBranch == body.TargetBranch {
			httpError(w, 400, "source and target lanes must differ")
			return
		}

		sourceState, err := s.getAppState(body.App, body.SourceEnv, body.SourceBranch)
		if err != nil || sourceState == nil || strings.TrimSpace(sourceState.CurrentImage) == "" {
			httpError(w, 400, "source lane has no deployed image to promote")
			return
		}
		healthy, healthStatus, healthDetail := s.evaluateLaneHealth(body.App, body.SourceEnv, body.SourceBranch, sourceState)
		if !healthy {
			httpError(w, 409, "source lane is not healthy enough to promote: "+healthDetail)
			return
		}

		latest, _ := s.latestSuccessfulDeploy(body.App, body.SourceEnv, body.SourceBranch)
		actor := requestActorLabel(s, r)
		role := requestActorRole(s, r)
		approvalRequired := s.hasUsers() && body.TargetEnv == EnvProd && role != "owner"
		now := time.Now().UnixMilli()
		rec := PromotionRecord{
			ID:               newID(),
			App:              body.App,
			SourceEnv:        string(body.SourceEnv),
			SourceBranch:     body.SourceBranch,
			SourceImage:      sourceState.CurrentImage,
			TargetEnv:        string(body.TargetEnv),
			TargetBranch:     body.TargetBranch,
			Status:           promotionStatusPendingApproval,
			ApprovalRequired: approvalRequired,
			RequestedBy:      actor,
			RequestedAt:      now,
			HealthStatus:     healthStatus,
			HealthDetail:     healthDetail,
		}
		if latest != nil {
			rec.SourceDeployID = latest.ID
		}
		if !approvalRequired {
			rec.Status = promotionStatusApproved
			rec.ApprovedBy = actor
			rec.ApprovedAt = now
		}
		if err := s.insertPromotion(rec); err != nil {
			httpError(w, 500, "failed to store promotion request: "+err.Error())
			return
		}
		s.auditLog(actor, "promotion.request", rec.App, fmt.Sprintf("from=%s/%s to=%s/%s", rec.SourceEnv, rec.SourceBranch, rec.TargetEnv, rec.TargetBranch))

		if !approvalRequired {
			s.auditLog(actor, "promotion.approve", rec.App, fmt.Sprintf("promotion=%s", rec.ID))
			if err := s.launchPromotion(rec.ID, actor); err != nil {
				_ = s.updatePromotionFields(rec.ID, promotionStatusFailed, "", 0, "", "", "launch_failed", err.Error())
				httpError(w, 500, "failed to start promotion: "+err.Error())
				return
			}
			updated, _ := s.getPromotion(rec.ID)
			writeJSON(w, 201, updated)
			return
		}

		writeJSON(w, 201, rec)
	default:
		httpError(w, 405, "method not allowed")
	}
}

func (s *Server) handlePromotionApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	body.ID = strings.TrimSpace(body.ID)
	if body.ID == "" {
		httpError(w, 400, "id required")
		return
	}
	rec, err := s.getPromotion(body.ID)
	if err != nil || rec == nil {
		httpError(w, 404, "promotion not found")
		return
	}
	if rec.Status != promotionStatusPendingApproval {
		httpError(w, 400, "promotion is not awaiting approval")
		return
	}
	actor := requestActorLabel(s, r)
	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`UPDATE promotions SET status=?, approved_by=?, approved_at=? WHERE id=?`, promotionStatusApproved, actor, now, rec.ID); err != nil {
		httpError(w, 500, "failed to approve promotion: "+err.Error())
		return
	}
	s.auditLog(actor, "promotion.approve", rec.App, fmt.Sprintf("promotion=%s", rec.ID))
	if err := s.launchPromotion(rec.ID, actor); err != nil {
		_ = s.updatePromotionFields(rec.ID, promotionStatusFailed, "", 0, "", "", "launch_failed", err.Error())
		httpError(w, 500, "failed to start promotion: "+err.Error())
		return
	}
	updated, _ := s.getPromotion(rec.ID)
	writeJSON(w, 200, updated)
}

func (s *Server) insertPromotion(rec PromotionRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO promotions
		(id, app, source_env, source_branch, source_deploy_id, source_image, target_env, target_branch, status, approval_required, requested_by, requested_at, approved_by, approved_at, target_deploy_id, rollback_deploy_id, health_status, health_detail)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID, rec.App, rec.SourceEnv, rec.SourceBranch, rec.SourceDeployID, rec.SourceImage, rec.TargetEnv, rec.TargetBranch, rec.Status, boolToInt(rec.ApprovalRequired), rec.RequestedBy, rec.RequestedAt, rec.ApprovedBy, rec.ApprovedAt, rec.TargetDeployID, rec.RollbackDeployID, rec.HealthStatus, rec.HealthDetail,
	)
	return err
}

func (s *Server) getPromotion(id string) (*PromotionRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, app, source_env, source_branch, COALESCE(source_deploy_id,''), COALESCE(source_image,''), target_env, target_branch, status, COALESCE(approval_required,0), COALESCE(requested_by,''), COALESCE(requested_at,0), COALESCE(approved_by,''), COALESCE(approved_at,0), COALESCE(target_deploy_id,''), COALESCE(rollback_deploy_id,''), COALESCE(health_status,''), COALESCE(health_detail,'')
		 FROM promotions WHERE id=?`,
		id,
	)
	var rec PromotionRecord
	var approvalRequired int
	if err := row.Scan(&rec.ID, &rec.App, &rec.SourceEnv, &rec.SourceBranch, &rec.SourceDeployID, &rec.SourceImage, &rec.TargetEnv, &rec.TargetBranch, &rec.Status, &approvalRequired, &rec.RequestedBy, &rec.RequestedAt, &rec.ApprovedBy, &rec.ApprovedAt, &rec.TargetDeployID, &rec.RollbackDeployID, &rec.HealthStatus, &rec.HealthDetail); err != nil {
		return nil, err
	}
	rec.ApprovalRequired = approvalRequired != 0
	return &rec, nil
}

func (s *Server) listPromotions(app string, sourceEnv DeployEnv, targetEnv DeployEnv, branch string) ([]PromotionRecord, error) {
	filters := []string{"1=1"}
	args := []any{}
	if app = strings.TrimSpace(app); app != "" {
		filters = append(filters, "app=?")
		args = append(args, app)
	}
	if sourceEnv != "" {
		filters = append(filters, "source_env=?")
		args = append(args, string(sourceEnv))
	}
	if targetEnv != "" {
		filters = append(filters, "target_env=?")
		args = append(args, string(targetEnv))
	}
	if branch = strings.TrimSpace(branch); branch != "" {
		filters = append(filters, "(source_branch=? OR target_branch=?)")
		args = append(args, branch, branch)
	}
	args = append(args, 50)
	rows, err := s.db.Query(
		`SELECT id, app, source_env, source_branch, COALESCE(source_deploy_id,''), COALESCE(source_image,''), target_env, target_branch, status, COALESCE(approval_required,0), COALESCE(requested_by,''), COALESCE(requested_at,0), COALESCE(approved_by,''), COALESCE(approved_at,0), COALESCE(target_deploy_id,''), COALESCE(rollback_deploy_id,''), COALESCE(health_status,''), COALESCE(health_detail,'')
		 FROM promotions WHERE `+strings.Join(filters, " AND ")+` ORDER BY requested_at DESC LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PromotionRecord, 0, 16)
	for rows.Next() {
		var rec PromotionRecord
		var approvalRequired int
		if err := rows.Scan(&rec.ID, &rec.App, &rec.SourceEnv, &rec.SourceBranch, &rec.SourceDeployID, &rec.SourceImage, &rec.TargetEnv, &rec.TargetBranch, &rec.Status, &approvalRequired, &rec.RequestedBy, &rec.RequestedAt, &rec.ApprovedBy, &rec.ApprovedAt, &rec.TargetDeployID, &rec.RollbackDeployID, &rec.HealthStatus, &rec.HealthDetail); err != nil {
			continue
		}
		rec.ApprovalRequired = approvalRequired != 0
		out = append(out, rec)
	}
	return out, nil
}

func (s *Server) updatePromotionFields(id string, status string, targetDeployID string, approvedAt int64, rollbackDeployID string, approvedBy string, healthStatus string, healthDetail string) error {
	_, err := s.db.Exec(
		`UPDATE promotions
		 SET status=?,
		     target_deploy_id=CASE WHEN ?='' THEN target_deploy_id ELSE ? END,
		     approved_at=CASE WHEN ?=0 THEN approved_at ELSE ? END,
		     rollback_deploy_id=CASE WHEN ?='' THEN rollback_deploy_id ELSE ? END,
		     approved_by=CASE WHEN ?='' THEN approved_by ELSE ? END,
		     health_status=?,
		     health_detail=?
		 WHERE id=?`,
		status,
		targetDeployID, targetDeployID,
		approvedAt, approvedAt,
		rollbackDeployID, rollbackDeployID,
		approvedBy, approvedBy,
		healthStatus, healthDetail,
		id,
	)
	return err
}

func (s *Server) latestSuccessfulDeploy(app string, env DeployEnv, branch string) (*Deploy, error) {
	row := s.db.QueryRow(
		`SELECT id, app, repo_url, branch, commit_sha, env, status, created_at, started_at, ended_at, error, log_path, image_tag, previous_image_tag, COALESCE(preview_url,''), COALESCE(build_number,0), COALESCE(deployed_by,''), COALESCE(commit_message,'')
		 FROM deploys
		 WHERE app=? AND env=? AND branch=? AND status=?
		 ORDER BY created_at DESC LIMIT 1`,
		app, string(env), branch, string(StatusSuccess),
	)
	var d Deploy
	var envStr string
	var status string
	var created int64
	var started sql.NullInt64
	var ended sql.NullInt64
	var errText sql.NullString
	if err := row.Scan(&d.ID, &d.App, &d.RepoURL, &d.Branch, &d.CommitSHA, &envStr, &status, &created, &started, &ended, &errText, &d.LogPath, &d.ImageTag, &d.PrevImage, &d.PreviewURL, &d.BuildNumber, &d.DeployedBy, &d.CommitMessage); err != nil {
		return nil, err
	}
	d.Env = DeployEnv(envStr)
	d.Status = DeployStatus(status)
	d.CreatedAt = millisToTime(created)
	if started.Valid {
		startedAt := millisToTime(started.Int64)
		d.StartedAt = &startedAt
	}
	if ended.Valid {
		endedAt := millisToTime(ended.Int64)
		d.EndedAt = &endedAt
	}
	if errText.Valid {
		d.Error = errText.String
	}
	return &d, nil
}

func (s *Server) launchPromotion(id string, actor string) error {
	rec, err := s.getPromotion(id)
	if err != nil {
		return err
	}
	sourceEnv := DeployEnv(rec.SourceEnv)
	targetEnv := DeployEnv(rec.TargetEnv)
	sourceState, err := s.getAppState(rec.App, sourceEnv, rec.SourceBranch)
	if err != nil || sourceState == nil {
		return fmt.Errorf("source lane state not found")
	}
	targetState, _ := s.getAppState(rec.App, targetEnv, rec.TargetBranch)
	sourceEngine := firstNonEmptyEngine(sourceState.Engine)
	targetEngine := sourceEngine
	if targetState != nil {
		targetEngine = firstNonEmptyEngine(targetState.Engine)
		if strings.TrimSpace(targetState.Engine) != "" && targetEngine != sourceEngine {
			return fmt.Errorf("promotion requires the same runtime engine on source and target lanes")
		}
	}
	latest, _ := s.latestSuccessfulDeploy(rec.App, sourceEnv, rec.SourceBranch)
	req := DeployRequest{
		App: rec.App,
		RepoURL: firstNonEmpty(sourceState.RepoURL, func() string {
			if targetState != nil {
				return targetState.RepoURL
			}
			return ""
		}()),
		Branch: rec.TargetBranch,
		CommitSHA: func() string {
			if latest != nil {
				return latest.CommitSHA
			}
			return ""
		}(),
		Env: targetEnv,
		ServicePort: func() int {
			if targetState != nil && targetState.ServicePort > 0 {
				return targetState.ServicePort
			}
			return sourceState.ServicePort
		}(),
		HostPort: func() int {
			if targetState != nil && targetState.HostPort > 0 {
				return targetState.HostPort
			}
			return defaultHostPort(targetEnv)
		}(),
		HostPortExplicit: targetState != nil && targetState.HostPortExplicit,
		PublicHost: func() string {
			if targetState != nil {
				return targetState.PublicHost
			}
			return ""
		}(),
		Mode: firstNonEmpty(func() string {
			if targetState != nil {
				return targetState.Mode
			}
			return ""
		}(), sourceState.Mode),
		TrafficMode: firstNonEmpty(func() string {
			if targetState != nil {
				return targetState.TrafficMode
			}
			return ""
		}(), sourceState.TrafficMode),
		Source: "promote",
		Engine: targetEngine,
		CommitMessage: func() string {
			if latest != nil {
				return latest.CommitMessage
			}
			return ""
		}(),
		DeployedBy: actor,
	}
	deployID := newID()
	logPath := filepath.Join(s.logsDir, deployID+".log")
	deploy := &Deploy{
		ID:        deployID,
		App:       req.App,
		RepoURL:   req.RepoURL,
		Branch:    req.Branch,
		CommitSHA: req.CommitSHA,
		Env:       req.Env,
		Status:    StatusQueued,
		CreatedAt: time.Now(),
		LogPath:   logPath,
	}
	s.mu.Lock()
	s.deploys[deployID] = deploy
	s.mu.Unlock()
	if err := s.saveDeployToDB(deploy, req); err != nil {
		return err
	}
	if err := s.updatePromotionFields(rec.ID, promotionStatusRunning, deployID, 0, "", "", "", ""); err != nil {
		return err
	}
	s.auditLog(actor, "promotion.start", rec.App, fmt.Sprintf("promotion=%s target=%s/%s deploy=%s", rec.ID, rec.TargetEnv, rec.TargetBranch, deployID))
	s.queue <- DeployJob{ID: deployID, Req: req, PromoteImage: rec.SourceImage}
	s.broadcastSnapshot()
	go s.watchPromotion(rec.ID, actor)
	return nil
}

func (s *Server) watchPromotion(id string, actor string) {
	for attempt := 0; attempt < 180; attempt++ {
		rec, err := s.getPromotion(id)
		if err != nil || rec == nil {
			return
		}
		if strings.TrimSpace(rec.TargetDeployID) == "" {
			time.Sleep(2 * time.Second)
			continue
		}
		deploy, err := s.getDeployFromDB(rec.TargetDeployID)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if isActiveDeployStatus(deploy.Status) {
			time.Sleep(2 * time.Second)
			continue
		}
		if deploy.Status != StatusSuccess {
			detail := firstNonEmpty(strings.TrimSpace(deploy.Error), "promotion deploy failed")
			_ = s.updatePromotionFields(rec.ID, promotionStatusFailed, "", 0, "", "", "failed", detail)
			s.auditLog(actor, "promotion.failed", rec.App, fmt.Sprintf("promotion=%s detail=%s", rec.ID, detail))
			return
		}

		targetState, _ := s.getAppState(rec.App, DeployEnv(rec.TargetEnv), rec.TargetBranch)
		healthy, healthStatus, healthDetail := s.evaluateLaneHealth(rec.App, DeployEnv(rec.TargetEnv), rec.TargetBranch, targetState)
		if healthy {
			_ = s.updatePromotionFields(rec.ID, promotionStatusSuccess, "", 0, "", "", healthStatus, healthDetail)
			s.auditLog(actor, "promotion.success", rec.App, fmt.Sprintf("promotion=%s", rec.ID))
			return
		}

		rollbackID, rollbackErr := s.enqueueRollbackForLane(RollbackRequest{
			App:    rec.App,
			Env:    DeployEnv(rec.TargetEnv),
			Branch: rec.TargetBranch,
		}, "system", "rollback")
		if rollbackErr != nil {
			_ = s.updatePromotionFields(rec.ID, promotionStatusFailed, "", 0, "", "", healthStatus, healthDetail)
			s.auditLog("system", "promotion.health_fail", rec.App, fmt.Sprintf("promotion=%s detail=%s", rec.ID, healthDetail))
			return
		}

		_ = s.updatePromotionFields(rec.ID, promotionStatusRolledBack, "", 0, rollbackID, "", healthStatus, healthDetail)
		s.auditLog("system", "promotion.rollback_queued", rec.App, fmt.Sprintf("promotion=%s rollback=%s", rec.ID, rollbackID))
		return
	}
	_ = s.updatePromotionFields(id, promotionStatusFailed, "", 0, "", "", "timeout", "promotion watcher timed out")
}

func (s *Server) evaluateLaneHealth(app string, env DeployEnv, branch string, st *AppState) (bool, string, string) {
	if st == nil {
		return false, "unhealthy", "lane state not found"
	}
	if st.Stopped {
		return false, "unhealthy", "lane is stopped"
	}
	latest, err := s.latestSuccessfulDeploy(app, env, branch)
	if err != nil || latest == nil {
		return false, "unhealthy", "lane does not have a successful deploy"
	}
	if !s.appLaneRunning(app, env, branch) {
		return false, "unhealthy", "no running app target is available"
	}
	accessPolicy := firstNonEmpty(normalizeAccessPolicy(st.AccessPolicy), s.lanePolicy(env).DefaultAccessPolicy)
	if accessPolicy != AccessPolicyPublic {
		return true, "healthy", "runtime target is live; HTTP probe skipped because the route is protected"
	}
	if ok, detail := probePromotionRoute(st); !ok {
		return false, "unhealthy", detail
	}
	return true, "healthy", "runtime target is live and the public route responded"
}

func probePromotionRoute(st *AppState) (bool, string) {
	candidates := []string{}
	if host := strings.TrimSpace(st.PublicHost); host != "" {
		candidates = append(candidates, "https://"+host, "http://"+host)
	} else if st.HostPort > 0 {
		candidates = append(candidates, fmt.Sprintf("http://127.0.0.1:%d", st.HostPort))
	}
	if len(candidates) == 0 {
		return true, "no public route configured; runtime-only health accepted"
	}
	client := &http.Client{Timeout: 5 * time.Second}
	lastErr := ""
	for _, candidate := range candidates {
		req, err := http.NewRequest(http.MethodGet, candidate, nil)
		if err != nil {
			lastErr = err.Error()
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err.Error()
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true, "route probe succeeded"
		}
		lastErr = fmt.Sprintf("route probe returned HTTP %d", resp.StatusCode)
	}
	if lastErr == "" {
		lastErr = "route probe failed"
	}
	return false, lastErr
}

func (s *Server) enqueueRollbackForLane(req RollbackRequest, actor string, source string) (string, error) {
	if !validDeployTarget(req.App, req.Env, req.Branch) {
		return "", fmt.Errorf("app, branch, env required")
	}
	state, err := s.getAppState(req.App, req.Env, req.Branch)
	if err != nil || state == nil || strings.TrimSpace(state.PreviousImage) == "" {
		return "", fmt.Errorf("no previous image to rollback")
	}
	deployID := newID()
	logPath := filepath.Join(s.logsDir, deployID+".log")
	deploy := &Deploy{
		ID:        deployID,
		App:       req.App,
		Branch:    req.Branch,
		Env:       req.Env,
		Status:    StatusQueued,
		CreatedAt: time.Now(),
		LogPath:   logPath,
	}
	s.mu.Lock()
	s.deploys[deployID] = deploy
	s.mu.Unlock()
	rollbackReq := DeployRequest{
		App:              req.App,
		Branch:           req.Branch,
		Env:              req.Env,
		Mode:             state.Mode,
		HostPort:         state.HostPort,
		HostPortExplicit: state.HostPortExplicit,
		ServicePort:      state.ServicePort,
		PublicHost:       state.PublicHost,
		Source:           source,
		DeployedBy:       actor,
	}
	if err := s.saveDeployToDB(deploy, rollbackReq); err != nil {
		return "", err
	}
	s.queue <- DeployJob{ID: deployID, Req: rollbackReq, Rollback: true, RollbackImage: state.PreviousImage}
	s.auditLog(actor, "deploy.rollback", req.App, fmt.Sprintf("env=%s branch=%s deploy=%s", req.Env, req.Branch, deployID))
	s.broadcastSnapshot()
	return deployID, nil
}
