package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultGitHubConnectionID = "default"

type githubConnectionRecord struct {
	ID        string
	Login     string
	AvatarURL string
	HTMLURL   string
	Token     string
	CreatedAt int64
	UpdatedAt int64
}

type githubConnectionView struct {
	Connected           bool   `json:"connected"`
	Login               string `json:"login,omitempty"`
	AvatarURL           string `json:"avatar_url,omitempty"`
	HTMLURL             string `json:"html_url,omitempty"`
	WebhookURL          string `json:"webhook_url"`
	SecretKeyConfigured bool   `json:"secret_key_configured"`
	UpdatedAt           int64  `json:"updated_at,omitempty"`
}

type githubProjectRecord struct {
	App               string `json:"app"`
	ConnectionID      string `json:"-"`
	RepoFullName      string `json:"repo_full_name"`
	CloneURL          string `json:"clone_url"`
	HTMLURL           string `json:"html_url"`
	ProductionBranch  string `json:"production_branch"`
	PreviewEnabled    bool   `json:"preview_enabled"`
	ProductionEnabled bool   `json:"production_enabled"`
	WebhookID         int64  `json:"webhook_id,omitempty"`
	WebhookSecret     string `json:"-"`
	StatusContext     string `json:"status_context"`
	AuthMode          string `json:"auth_mode"`
	InstallationID    int64  `json:"installation_id,omitempty"`
	RepositoryID      int64  `json:"repository_id,omitempty"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

type githubPreviewRecord struct {
	RepoFullName       string `json:"repo_full_name"`
	PRNumber           int    `json:"pr_number,omitempty"`
	App                string `json:"app"`
	Branch             string `json:"branch"`
	StatusRepoFullName string `json:"status_repo_full_name,omitempty"`
	HeadSHA            string `json:"head_sha,omitempty"`
	DeployID           string `json:"deploy_id,omitempty"`
	PreviewURL         string `json:"preview_url,omitempty"`
	Status             string `json:"status"`
	UpdatedAt          int64  `json:"updated_at"`
}

type githubProductionView struct {
	Branch            string `json:"branch"`
	DeployID          string `json:"deploy_id,omitempty"`
	DeployStatus      string `json:"deploy_status,omitempty"`
	HealthStatus      string `json:"health_status"`
	HealthDetail      string `json:"health_detail,omitempty"`
	URL               string `json:"url,omitempty"`
	RollbackAvailable bool   `json:"rollback_available"`
	RolloutStatus     string `json:"rollout_status,omitempty"`
}

type githubProjectView struct {
	githubProjectRecord
	WebhookURL string                 `json:"webhook_url"`
	Previews   []githubPreviewRecord  `json:"previews"`
	Production githubProductionView   `json:"production"`
	LastEvent  *githubDeliverySummary `json:"last_event,omitempty"`
}

type githubDeliverySummary struct {
	Event      string `json:"event"`
	Action     string `json:"action,omitempty"`
	Outcome    string `json:"outcome"`
	ReceivedAt int64  `json:"received_at"`
}

type githubWebhookPayload struct {
	Action       string             `json:"action"`
	Ref          string             `json:"ref"`
	After        string             `json:"after"`
	Installation githubInstallation `json:"installation"`
	Repository   struct {
		ID            int64  `json:"id"`
		FullName      string `json:"full_name"`
		CloneURL      string `json:"clone_url"`
		HTMLURL       string `json:"html_url"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
	HeadCommit struct {
		Message string `json:"message"`
	} `json:"head_commit"`
	PullRequest struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Merged bool   `json:"merged"`
		Head   struct {
			Ref  string `json:"ref"`
			SHA  string `json:"sha"`
			Repo struct {
				FullName string `json:"full_name"`
				CloneURL string `json:"clone_url"`
			} `json:"repo"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
	Number              int                `json:"number"`
	RepositoriesAdded   []githubRepository `json:"repositories_added"`
	RepositoriesRemoved []githubRepository `json:"repositories_removed"`
}

func (s *Server) githubWebhookURL() string {
	return webhookProbeURL(s.httpAddr, s.serverDashboardHost())
}

func validateGitHubWebhookURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || strings.TrimSpace(u.Hostname()) == "" {
		return fmt.Errorf("GitHub connection requires a public HTTPS dashboard host")
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("GitHub connection requires a public HTTPS dashboard host")
	}
	return nil
}

func (s *Server) getGitHubConnection() (*githubConnectionRecord, error) {
	var rec githubConnectionRecord
	var tokenEnc string
	err := s.db.QueryRow(
		`SELECT id, login, avatar_url, html_url, token_enc, created_at, updated_at
		 FROM github_connections WHERE id=?`,
		defaultGitHubConnectionID,
	).Scan(&rec.ID, &rec.Login, &rec.AvatarURL, &rec.HTMLURL, &tokenEnc, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}
	rec.Token = s.decryptSecret(tokenEnc)
	return &rec, nil
}

func (s *Server) githubConnectionView() githubConnectionView {
	view := githubConnectionView{
		WebhookURL:          s.githubWebhookURL(),
		SecretKeyConfigured: s.relaySecretKeyConfigured(),
	}
	rec, err := s.getGitHubConnection()
	if err != nil {
		return view
	}
	view.Connected = true
	view.Login = rec.Login
	view.AvatarURL = rec.AvatarURL
	view.HTMLURL = rec.HTMLURL
	view.UpdatedAt = rec.UpdatedAt
	return view
}

func (s *Server) handleGitHubConnection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.githubConnectionView())
	case http.MethodPost:
		if !s.relaySecretKeyConfigured() {
			httpError(w, http.StatusConflict, "set RELAY_SECRET_KEY before connecting GitHub so the access token is encrypted at rest")
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, http.StatusBadRequest, "invalid json")
			return
		}
		body.Token = strings.TrimSpace(body.Token)
		if body.Token == "" {
			httpError(w, http.StatusBadRequest, "GitHub token required")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		user, err := s.fetchGitHubUser(ctx, body.Token)
		if err != nil {
			httpError(w, http.StatusBadGateway, "GitHub connection failed: "+err.Error())
			return
		}
		if strings.TrimSpace(user.Login) == "" {
			httpError(w, http.StatusBadGateway, "GitHub returned an account without a login")
			return
		}
		now := time.Now().UnixMilli()
		_, err = s.db.Exec(
			`INSERT INTO github_connections (id, login, avatar_url, html_url, token_enc, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET login=excluded.login, avatar_url=excluded.avatar_url,
			 html_url=excluded.html_url, token_enc=excluded.token_enc, updated_at=excluded.updated_at`,
			defaultGitHubConnectionID, user.Login, user.AvatarURL, user.HTMLURL, s.encryptSecret(body.Token), now, now,
		)
		if err != nil {
			httpError(w, http.StatusInternalServerError, "store GitHub connection: "+err.Error())
			return
		}
		s.auditLog(requestActorLabel(s, r), "github.connect", user.Login, "fine-grained token connected")
		writeJSON(w, http.StatusOK, s.githubConnectionView())
	case http.MethodDelete:
		var projectCount int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM github_projects WHERE auth_mode!='app'`).Scan(&projectCount)
		if projectCount > 0 {
			httpError(w, http.StatusConflict, "disconnect linked GitHub projects first")
			return
		}
		if _, err := s.db.Exec(`DELETE FROM github_connections WHERE id=?`, defaultGitHubConnectionID); err != nil {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.auditLog(requestActorLabel(s, r), "github.disconnect", "github", "connection removed")
		writeJSON(w, http.StatusOK, s.githubConnectionView())
	default:
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGitHubRepositories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, appErr := s.getGitHubAppConfig(); appErr == nil {
		repos, err := s.listGitHubInstallationRepositories()
		if err != nil {
			httpError(w, http.StatusInternalServerError, "list GitHub App repositories: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, repos)
		return
	}
	connection, err := s.getGitHubConnection()
	if err != nil {
		httpError(w, http.StatusConflict, "connect GitHub first")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	repos, err := s.fetchGitHubRepositories(ctx, connection.Token)
	if err != nil {
		httpError(w, http.StatusBadGateway, "list GitHub repositories: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repos)
}

func (s *Server) handleGitHubProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app := strings.TrimSpace(r.URL.Query().Get("app"))
		if app != "" {
			project, projectErr := s.getGitHubProjectByApp(app)
			if projectErr == nil && project != nil {
				if _, ok := s.requireLaneAccess(w, r, app, EnvProd, "viewer"); !ok {
					return
				}
			}
		}
		projects, err := s.listGitHubProjectViews(app)
		if err != nil {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if app == "" && s.hasUsers() {
			sess := s.validateUserSession(r)
			filtered := projects[:0]
			for _, project := range projects {
				if sess != nil && roleAtLeast(s.effectiveLaneRole(sess, project.App, EnvProd), "viewer") {
					filtered = append(filtered, project)
				}
			}
			projects = filtered
		}
		writeJSON(w, http.StatusOK, projects)
	case http.MethodPost:
		var body struct {
			App               string `json:"app"`
			RepoFullName      string `json:"repo_full_name"`
			ProductionBranch  string `json:"production_branch"`
			PreviewEnabled    *bool  `json:"preview_enabled"`
			ProductionEnabled *bool  `json:"production_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, http.StatusBadRequest, "invalid json")
			return
		}
		body.App = strings.TrimSpace(body.App)
		body.RepoFullName = strings.TrimSpace(body.RepoFullName)
		if body.App == "" || body.RepoFullName == "" {
			httpError(w, http.StatusBadRequest, "app and repo_full_name are required")
			return
		}
		if err := s.connectGitHubProject(r.Context(), body.App, body.RepoFullName, body.ProductionBranch, body.PreviewEnabled, body.ProductionEnabled); err != nil {
			httpError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.auditLog(requestActorLabel(s, r), "github.project.connect", body.App, "repo="+body.RepoFullName)
		projects, listErr := s.listGitHubProjectViews(body.App)
		if listErr != nil || len(projects) == 0 {
			httpError(w, http.StatusInternalServerError, "GitHub project was connected but could not be reloaded")
			return
		}
		writeJSON(w, http.StatusCreated, projects[0])
	case http.MethodDelete:
		app := strings.TrimSpace(r.URL.Query().Get("app"))
		if app == "" {
			httpError(w, http.StatusBadRequest, "app required")
			return
		}
		if err := s.disconnectGitHubProject(r.Context(), app); err != nil {
			httpError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.auditLog(requestActorLabel(s, r), "github.project.disconnect", app, "webhook removed")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func randomGitHubWebhookSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func (s *Server) connectGitHubProject(ctx context.Context, app string, repoFullName string, productionBranch string, previewEnabled *bool, productionEnabled *bool) error {
	if appRepository, appErr := s.getGitHubInstallationRepository(repoFullName); appErr == nil && appRepository != nil {
		return s.connectGitHubAppProject(ctx, app, appRepository, productionBranch, previewEnabled, productionEnabled)
	}
	connection, err := s.getGitHubConnection()
	if err != nil {
		return fmt.Errorf("connect GitHub first")
	}
	callbackURL := s.githubWebhookURL()
	if err := validateGitHubWebhookURL(callbackURL); err != nil {
		return err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	repo, err := s.fetchGitHubRepository(requestCtx, connection.Token, repoFullName)
	if err != nil {
		return fmt.Errorf("read GitHub repository: %w", err)
	}
	productionBranch = strings.TrimSpace(productionBranch)
	if productionBranch == "" {
		productionBranch = strings.TrimSpace(repo.DefaultBranch)
	}
	if !validDeployTarget(app, EnvProd, productionBranch) {
		return fmt.Errorf("app and production branch must be valid Relay deployment names")
	}
	previewOn := true
	productionOn := true
	if previewEnabled != nil {
		previewOn = *previewEnabled
	}
	if productionEnabled != nil {
		productionOn = *productionEnabled
	}

	existing, _ := s.getGitHubProjectByApp(app)
	secret, err := randomGitHubWebhookSecret()
	if err != nil {
		return fmt.Errorf("generate webhook secret: %w", err)
	}
	hookID := int64(0)
	createdHook := false
	if existing != nil && strings.EqualFold(existing.RepoFullName, repo.FullName) && existing.WebhookID > 0 {
		hookID = existing.WebhookID
		if err := s.updateGitHubWebhook(requestCtx, connection.Token, repo.FullName, hookID, callbackURL, secret); err != nil {
			return fmt.Errorf("update GitHub webhook: %w", err)
		}
	} else {
		hook, err := s.createGitHubWebhook(requestCtx, connection.Token, repo.FullName, callbackURL, secret)
		if err != nil {
			return fmt.Errorf("create GitHub webhook (token needs repository Webhooks: write): %w", err)
		}
		hookID = hook.ID
		createdHook = true
	}

	now := time.Now().UnixMilli()
	_, err = s.db.Exec(
		`INSERT INTO github_projects
		 (app, connection_id, repo_full_name, clone_url, html_url, production_branch, preview_enabled, production_enabled, webhook_id, webhook_secret_enc, status_context, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'relay/preview', ?, ?)
		 ON CONFLICT(app) DO UPDATE SET connection_id=excluded.connection_id, repo_full_name=excluded.repo_full_name,
		 clone_url=excluded.clone_url, html_url=excluded.html_url, production_branch=excluded.production_branch,
		 preview_enabled=excluded.preview_enabled, production_enabled=excluded.production_enabled,
		 webhook_id=excluded.webhook_id, webhook_secret_enc=excluded.webhook_secret_enc, updated_at=excluded.updated_at`,
		app, defaultGitHubConnectionID, repo.FullName, repo.CloneURL, repo.HTMLURL, productionBranch,
		boolToInt(previewOn), boolToInt(productionOn), hookID, s.encryptSecret(secret), now, now,
	)
	if err != nil {
		if createdHook {
			_ = s.deleteGitHubWebhook(requestCtx, connection.Token, repo.FullName, hookID)
		}
		return fmt.Errorf("store GitHub project: %w", err)
	}
	if existing != nil && !strings.EqualFold(existing.RepoFullName, repo.FullName) && existing.WebhookID > 0 {
		_ = s.deleteGitHubWebhook(requestCtx, connection.Token, existing.RepoFullName, existing.WebhookID)
	}
	if err := s.ensureBaselineLanes(app, productionBranch, EnvProd, repo.CloneURL, ""); err != nil {
		return fmt.Errorf("create production lane: %w", err)
	}
	return nil
}

func (s *Server) connectGitHubAppProject(ctx context.Context, app string, installed *githubRepository, productionBranch string, previewEnabled *bool, productionEnabled *bool) error {
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	token, err := s.githubInstallationToken(requestCtx, installed.InstallationID, installed.ID)
	if err != nil {
		return fmt.Errorf("authenticate GitHub App installation: %w", err)
	}
	repo, err := s.fetchGitHubRepository(requestCtx, token, installed.FullName)
	if err != nil {
		return fmt.Errorf("read GitHub App repository: %w", err)
	}
	productionBranch = strings.TrimSpace(productionBranch)
	if productionBranch == "" {
		productionBranch = strings.TrimSpace(repo.DefaultBranch)
	}
	if !validDeployTarget(app, EnvProd, productionBranch) {
		return fmt.Errorf("app and production branch must be valid Relay deployment names")
	}
	previewOn, productionOn := true, true
	if previewEnabled != nil {
		previewOn = *previewEnabled
	}
	if productionEnabled != nil {
		productionOn = *productionEnabled
	}
	existing, _ := s.getGitHubProjectByApp(app)
	now := time.Now().UnixMilli()
	_, err = s.db.Exec(
		`INSERT INTO github_projects
		 (app, connection_id, repo_full_name, clone_url, html_url, production_branch,
		 preview_enabled, production_enabled, webhook_id, webhook_secret_enc, status_context,
		 auth_mode, installation_id, repository_id, created_at, updated_at)
		 VALUES (?, 'github-app', ?, ?, ?, ?, ?, ?, 0, ?, 'relay/preview', 'app', ?, ?, ?, ?)
		 ON CONFLICT(app) DO UPDATE SET connection_id=excluded.connection_id,
		 repo_full_name=excluded.repo_full_name, clone_url=excluded.clone_url, html_url=excluded.html_url,
		 production_branch=excluded.production_branch, preview_enabled=excluded.preview_enabled,
		 production_enabled=excluded.production_enabled, webhook_id=0,
		 webhook_secret_enc=excluded.webhook_secret_enc, auth_mode='app',
		 installation_id=excluded.installation_id, repository_id=excluded.repository_id,
		 updated_at=excluded.updated_at`,
		app, repo.FullName, repo.CloneURL, repo.HTMLURL, productionBranch, boolToInt(previewOn),
		boolToInt(productionOn), s.encryptSecret(""), installed.InstallationID, installed.ID, now, now,
	)
	if err != nil {
		return fmt.Errorf("store GitHub App project: %w", err)
	}
	if existing != nil && existing.AuthMode != "app" && existing.WebhookID > 0 {
		if connection, connectionErr := s.getGitHubConnection(); connectionErr == nil {
			_ = s.deleteGitHubWebhook(requestCtx, connection.Token, existing.RepoFullName, existing.WebhookID)
		}
	}
	if err := s.ensureBaselineLanes(app, productionBranch, EnvProd, repo.CloneURL, ""); err != nil {
		return fmt.Errorf("create production lane: %w", err)
	}
	return nil
}

func (s *Server) disconnectGitHubProject(ctx context.Context, app string) error {
	project, err := s.getGitHubProjectByApp(app)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	connection, connectionErr := s.getGitHubConnection()
	if project.AuthMode != "app" && connectionErr == nil && project.WebhookID > 0 {
		requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := s.deleteGitHubWebhook(requestCtx, connection.Token, project.RepoFullName, project.WebhookID); err != nil {
			return fmt.Errorf("remove GitHub webhook: %w", err)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM github_previews WHERE app=?`, app); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`DELETE FROM github_projects WHERE app=?`, app); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Server) scanGitHubProject(scanner interface{ Scan(...any) error }) (*githubProjectRecord, error) {
	var rec githubProjectRecord
	var previewEnabled, productionEnabled int
	var secretEnc string
	err := scanner.Scan(
		&rec.App, &rec.ConnectionID, &rec.RepoFullName, &rec.CloneURL, &rec.HTMLURL,
		&rec.ProductionBranch, &previewEnabled, &productionEnabled, &rec.WebhookID,
		&secretEnc, &rec.StatusContext, &rec.AuthMode, &rec.InstallationID, &rec.RepositoryID,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.PreviewEnabled = previewEnabled != 0
	rec.ProductionEnabled = productionEnabled != 0
	rec.WebhookSecret = s.decryptSecret(secretEnc)
	return &rec, nil
}

const githubProjectSelect = `SELECT app, connection_id, repo_full_name, clone_url, html_url,
	production_branch, preview_enabled, production_enabled, webhook_id, webhook_secret_enc,
	status_context, auth_mode, installation_id, repository_id, created_at, updated_at FROM github_projects`

func (s *Server) getGitHubProjectByApp(app string) (*githubProjectRecord, error) {
	return s.scanGitHubProject(s.db.QueryRow(githubProjectSelect+` WHERE app=?`, strings.TrimSpace(app)))
}

func (s *Server) getGitHubProjectByRepo(repoFullName string) (*githubProjectRecord, error) {
	return s.scanGitHubProject(s.db.QueryRow(githubProjectSelect+` WHERE LOWER(repo_full_name)=LOWER(?)`, strings.TrimSpace(repoFullName)))
}

func (s *Server) githubProjectToken(app string) string {
	project, err := s.getGitHubProjectByApp(app)
	if err != nil || project == nil {
		return ""
	}
	if project.AuthMode == "app" {
		token, _ := s.githubInstallationToken(context.Background(), project.InstallationID, project.RepositoryID)
		return token
	}
	connection, err := s.getGitHubConnection()
	if err != nil || connection == nil || connection.ID != project.ConnectionID {
		return ""
	}
	return connection.Token
}

func (s *Server) githubProjectAccessToken(ctx context.Context, project *githubProjectRecord) (string, error) {
	if project == nil {
		return "", fmt.Errorf("GitHub project is unavailable")
	}
	if project.AuthMode == "app" {
		return s.githubInstallationToken(ctx, project.InstallationID, project.RepositoryID)
	}
	connection, err := s.getGitHubConnection()
	if err != nil || connection == nil || connection.ID != project.ConnectionID {
		return "", fmt.Errorf("GitHub connection is unavailable")
	}
	return connection.Token, nil
}

func (s *Server) listGitHubProjectViews(app string) ([]githubProjectView, error) {
	query := githubProjectSelect
	args := []any{}
	if app != "" {
		query += ` WHERE app=?`
		args = append(args, app)
	}
	query += ` ORDER BY app`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	views := []githubProjectView{}
	for rows.Next() {
		project, err := s.scanGitHubProject(rows)
		if err != nil {
			return nil, err
		}
		view := githubProjectView{githubProjectRecord: *project, WebhookURL: s.githubWebhookURL()}
		view.WebhookSecret = ""
		view.Previews, _ = s.listGitHubPreviews(project.App, 12)
		view.Production = s.githubProductionView(project)
		view.LastEvent = s.latestGitHubDelivery(project.RepoFullName)
		views = append(views, view)
	}
	return views, rows.Err()
}

func (s *Server) listGitHubPreviews(app string, limit int) ([]githubPreviewRecord, error) {
	rows, err := s.db.Query(
		`SELECT repo_full_name, pr_number, app, branch, status_repo_full_name, head_sha,
		 deploy_id, preview_url, status, updated_at FROM github_previews
		 WHERE app=? ORDER BY updated_at DESC LIMIT ?`, app, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []githubPreviewRecord{}
	for rows.Next() {
		var item githubPreviewRecord
		if err := rows.Scan(&item.RepoFullName, &item.PRNumber, &item.App, &item.Branch, &item.StatusRepoFullName, &item.HeadSHA, &item.DeployID, &item.PreviewURL, &item.Status, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) githubProductionView(project *githubProjectRecord) githubProductionView {
	view := githubProductionView{Branch: project.ProductionBranch, HealthStatus: "waiting"}
	state, _ := s.getAppState(project.App, EnvProd, project.ProductionBranch)
	if state != nil {
		view.URL = previewURLFromConfig(state.Mode, state.PublicHost, state.HostPort)
		view.RollbackAvailable = strings.TrimSpace(state.PreviousImage) != "" || (state.RolloutStatus == "monitoring" && strings.TrimSpace(state.StandbySlot) != "")
		view.RolloutStatus = state.RolloutStatus
	}
	deploy, err := s.latestDeployForLane(project.App, EnvProd, project.ProductionBranch)
	if err != nil || deploy == nil {
		view.HealthDetail = "Production deploy has not run yet."
		return view
	}
	view.DeployID = deploy.ID
	view.DeployStatus = string(deploy.Status)
	if deploy.PreviewURL != "" {
		view.URL = deploy.PreviewURL
	}
	switch deploy.Status {
	case StatusQueued, StatusRunning:
		view.HealthStatus = "deploying"
		view.HealthDetail = "Relay is building and checking the production candidate."
	case StatusFailed:
		view.HealthStatus = "unhealthy"
		view.HealthDetail = firstNonEmpty(deploy.Error, "Production deployment failed.")
	case StatusSuccess:
		if state == nil || !s.appLaneRunning(project.App, EnvProd, project.ProductionBranch) {
			view.HealthStatus = "unhealthy"
			view.HealthDetail = "Production deployment succeeded but no running app target is available."
			break
		}
		view.HealthStatus = "healthy"
		switch state.RolloutStatus {
		case "monitoring":
			view.HealthStatus = "deploying"
			view.HealthDetail = "The new production slot is live and Relay is monitoring rollout traffic."
		case "rolled_back":
			view.HealthDetail = "Relay restored the previous production slot after the rollout crossed its health threshold."
		case "graduated":
			view.HealthDetail = "The production rollout graduated after its health window."
		default:
			view.HealthDetail = "The production runtime target is live."
		}
	}
	return view
}

func (s *Server) latestDeployForLane(app string, env DeployEnv, branch string) (*Deploy, error) {
	row := s.db.QueryRow(
		`SELECT d.id, d.app, d.repo_url, d.branch, d.commit_sha, d.env, d.status, d.created_at,
		 d.started_at, d.ended_at, d.error, d.log_path, d.image_tag, d.previous_image_tag,
		 COALESCE(d.preview_url,''), COALESCE(r.source,''), COALESCE(d.build_number,0),
		 COALESCE(d.deployed_by,''), COALESCE(d.commit_message,'')
		 FROM deploys d LEFT JOIN deploy_requests r ON r.id=d.id
		 WHERE d.app=? AND d.env=? AND d.branch=? ORDER BY d.created_at DESC LIMIT 1`,
		app, string(env), branch,
	)
	return scanDeployRow(row)
}

func (s *Server) latestGitHubDelivery(repoFullName string) *githubDeliverySummary {
	var item githubDeliverySummary
	err := s.db.QueryRow(
		`SELECT event, action, outcome, received_at FROM github_deliveries
		 WHERE LOWER(repo_full_name)=LOWER(?) ORDER BY received_at DESC LIMIT 1`, repoFullName,
	).Scan(&item.Event, &item.Action, &item.Outcome, &item.ReceivedAt)
	if err != nil {
		return nil
	}
	return &item
}

func (s *Server) handleConnectedGitHubWebhook(w http.ResponseWriter, r *http.Request, event string, body []byte) bool {
	var payload githubWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil || strings.TrimSpace(payload.Repository.FullName) == "" {
		return false
	}
	project, err := s.getGitHubProjectByRepo(payload.Repository.FullName)
	if err != nil || project == nil {
		return false
	}
	webhookSecret := project.WebhookSecret
	if project.AuthMode == "app" {
		config, configErr := s.getGitHubAppConfig()
		if configErr != nil || payload.Installation.ID != project.InstallationID || payload.Repository.ID != project.RepositoryID || !s.githubInstallationRepositoryActive(project.InstallationID, project.RepositoryID) {
			httpError(w, http.StatusUnauthorized, "GitHub App installation does not match this Relay project")
			return true
		}
		webhookSecret = config.WebhookSecret
	}
	if !verifyGithubSig256([]byte(webhookSecret), body, r.Header.Get("X-Hub-Signature-256")) {
		httpError(w, http.StatusUnauthorized, "invalid signature")
		return true
	}
	deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
	if deliveryID == "" {
		deliveryID = fileHashByAlgoBytes(body, event)
	}
	claimed, err := s.claimGitHubDelivery(deliveryID, event, payload.Action, project.RepoFullName)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "record GitHub delivery: "+err.Error())
		return true
	}
	if !claimed {
		writeJSON(w, http.StatusOK, map[string]any{"status": "duplicate", "delivery_id": deliveryID})
		return true
	}

	outcome, deployID, processErr := s.processConnectedGitHubEvent(project, event, payload)
	if processErr != nil {
		_, _ = s.db.Exec(`DELETE FROM github_deliveries WHERE delivery_id=?`, deliveryID)
		httpError(w, http.StatusInternalServerError, processErr.Error())
		return true
	}
	_, _ = s.db.Exec(`UPDATE github_deliveries SET outcome=? WHERE delivery_id=?`, outcome, deliveryID)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":      outcome,
		"delivery_id": deliveryID,
		"deploy_id":   deployID,
	})
	return true
}

func fileHashByAlgoBytes(body []byte, salt string) string {
	// A stable fallback id for non-GitHub test senders; real GitHub deliveries
	// always carry X-GitHub-Delivery.
	digest := sha256.Sum256(append([]byte(salt+"\x00"), body...))
	return fmt.Sprintf("fallback-%x", digest[:])
}

func (s *Server) claimGitHubDelivery(deliveryID string, event string, action string, repoFullName string) (bool, error) {
	result, err := s.db.Exec(
		`INSERT OR IGNORE INTO github_deliveries
		 (delivery_id, event, action, repo_full_name, outcome, received_at)
		 VALUES (?, ?, ?, ?, 'received', ?)`,
		deliveryID, event, action, repoFullName, time.Now().UnixMilli(),
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (s *Server) processConnectedGitHubEvent(project *githubProjectRecord, event string, payload githubWebhookPayload) (string, string, error) {
	token, err := s.githubProjectAccessToken(context.Background(), project)
	if err != nil {
		return "failed", "", fmt.Errorf("GitHub connection is unavailable")
	}
	switch event {
	case "ping":
		return "connected", "", nil
	case "push":
		if !strings.HasPrefix(payload.Ref, "refs/heads/") || strings.Trim(payload.After, "0") == "" {
			return "ignored", "", nil
		}
		branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
		env := EnvPreview
		outcome := "preview_queued"
		if branch == project.ProductionBranch {
			if !project.ProductionEnabled {
				return "production_disabled", "", nil
			}
			env = EnvProd
			outcome = "production_queued"
		} else if !project.PreviewEnabled {
			return "preview_disabled", "", nil
		}
		deploy, err := s.queueGitHubDeploy(project, payload.Repository.CloneURL, branch, payload.After, payload.HeadCommit.Message, env, token)
		if err != nil {
			return "failed", "", err
		}
		if env == EnvPreview {
			if err := s.upsertGitHubPreview(githubPreviewRecord{
				RepoFullName:       project.RepoFullName,
				App:                project.App,
				Branch:             branch,
				StatusRepoFullName: project.RepoFullName,
				HeadSHA:            payload.After,
				DeployID:           deploy.ID,
				Status:             string(deploy.Status),
				UpdatedAt:          time.Now().UnixMilli(),
			}); err != nil {
				return "failed", "", fmt.Errorf("store GitHub preview: %w", err)
			}
		}
		go s.emitGitHubDeployStatusByID(deploy.ID)
		return outcome, deploy.ID, nil
	case "pull_request":
		prNumber := payload.PullRequest.Number
		if prNumber == 0 {
			prNumber = payload.Number
		}
		branch := strings.TrimSpace(payload.PullRequest.Head.Ref)
		if payload.Action == "closed" {
			status := "closed"
			if payload.PullRequest.Merged {
				status = "merged"
			}
			_, _ = s.db.Exec(
				`UPDATE github_previews SET status=?, updated_at=? WHERE LOWER(repo_full_name)=LOWER(?) AND pr_number=?`,
				status, time.Now().UnixMilli(), project.RepoFullName, prNumber,
			)
			if payload.PullRequest.Merged && payload.PullRequest.Base.Ref == project.ProductionBranch && project.ProductionEnabled {
				return "merged_waiting_for_production_push", "", nil
			}
			return status, "", nil
		}
		if payload.Action != "opened" && payload.Action != "reopened" && payload.Action != "synchronize" {
			return "ignored", "", nil
		}
		if !project.PreviewEnabled {
			return "preview_disabled", "", nil
		}
		statusRepo := firstNonEmpty(strings.TrimSpace(payload.PullRequest.Head.Repo.FullName), project.RepoFullName)
		cloneURL := firstNonEmpty(strings.TrimSpace(payload.PullRequest.Head.Repo.CloneURL), project.CloneURL)
		laneBranch := branch
		if !strings.EqualFold(statusRepo, project.RepoFullName) {
			laneBranch = fmt.Sprintf("pr-%d-%s", prNumber, branch)
		}
		deploy, err := s.queueGitHubDeploy(project, cloneURL, laneBranch, payload.PullRequest.Head.SHA, payload.PullRequest.Title, EnvPreview, token)
		if err != nil {
			return "failed", "", err
		}
		if err := s.upsertGitHubPreview(githubPreviewRecord{
			RepoFullName:       project.RepoFullName,
			PRNumber:           prNumber,
			App:                project.App,
			Branch:             laneBranch,
			StatusRepoFullName: statusRepo,
			HeadSHA:            payload.PullRequest.Head.SHA,
			DeployID:           deploy.ID,
			Status:             string(deploy.Status),
			UpdatedAt:          time.Now().UnixMilli(),
		}); err != nil {
			return "failed", "", fmt.Errorf("store GitHub pull request preview: %w", err)
		}
		go s.emitGitHubDeployStatusByID(deploy.ID)
		return "preview_queued", deploy.ID, nil
	default:
		return "ignored", "", nil
	}
}

func (s *Server) queueGitHubDeploy(project *githubProjectRecord, cloneURL string, branch string, commitSHA string, commitMessage string, env DeployEnv, token string) (*Deploy, error) {
	if !s.webhookAllowed(project.RepoFullName) {
		return nil, fmt.Errorf("GitHub webhook rate limit exceeded")
	}
	if existing, _ := s.findGitHubDeploy(project.App, env, branch, commitSHA); existing != nil {
		return existing, nil
	}
	cloneURL = firstNonEmpty(strings.TrimSpace(cloneURL), project.CloneURL)
	if err := s.ensureBaselineLanes(project.App, branch, env, cloneURL, ""); err != nil {
		return nil, fmt.Errorf("prepare %s lane: %w", env, err)
	}
	state, _ := s.getAppState(project.App, env, branch)
	req := DeployRequest{
		App:           project.App,
		RepoURL:       cloneURL,
		Branch:        branch,
		CommitSHA:     commitSHA,
		Env:           env,
		Source:        "github",
		CommitMessage: strings.SplitN(strings.TrimSpace(commitMessage), "\n", 2)[0],
		DeployedBy:    "github",
		GitToken:      token,
	}
	if state != nil {
		req.Engine = state.Engine
		req.Mode = state.Mode
		req.TrafficMode = state.TrafficMode
		req.HostPort = state.HostPort
		req.HostPortExplicit = state.HostPortExplicit
		req.ServicePort = state.ServicePort
		req.PublicHost = state.PublicHost
		req.PublicHosts = append([]string(nil), state.PublicHosts...)
	}
	deployID := newID()
	deploy := &Deploy{
		ID:            deployID,
		App:           project.App,
		RepoURL:       cloneURL,
		Branch:        branch,
		CommitSHA:     commitSHA,
		Env:           env,
		Status:        StatusQueued,
		CreatedAt:     time.Now(),
		LogPath:       filepath.Join(s.logsDir, deployID+".log"),
		BuildNumber:   s.nextBuildNumber(project.App),
		CommitMessage: req.CommitMessage,
		DeployedBy:    "github",
	}
	s.mu.Lock()
	s.deploys[deployID] = deploy
	s.mu.Unlock()
	if err := s.saveDeployToDB(deploy, req); err != nil {
		return nil, fmt.Errorf("store GitHub deployment: %w", err)
	}
	s.auditLog("github", "deploy.trigger", project.App, fmt.Sprintf("env=%s branch=%s commit=%s", env, branch, shortCommitSHA(commitSHA)))
	if s.queue == nil {
		return nil, fmt.Errorf("deployment queue is unavailable")
	}
	s.queue <- DeployJob{ID: deployID, Req: req}
	s.broadcastSnapshot()
	return deploy, nil
}

func shortCommitSHA(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 7 {
		return value[:7]
	}
	return value
}

func (s *Server) findGitHubDeploy(app string, env DeployEnv, branch string, commitSHA string) (*Deploy, error) {
	row := s.db.QueryRow(
		`SELECT d.id, d.app, d.repo_url, d.branch, d.commit_sha, d.env, d.status, d.created_at,
		 d.started_at, d.ended_at, d.error, d.log_path, d.image_tag, d.previous_image_tag,
		 COALESCE(d.preview_url,''), COALESCE(r.source,''), COALESCE(d.build_number,0),
		 COALESCE(d.deployed_by,''), COALESCE(d.commit_message,'')
		 FROM deploys d LEFT JOIN deploy_requests r ON r.id=d.id
		 WHERE d.app=? AND d.env=? AND d.branch=? AND d.commit_sha=?
		 AND d.status IN (?, ?, ?) ORDER BY d.created_at DESC LIMIT 1`,
		app, string(env), branch, commitSHA, string(StatusQueued), string(StatusRunning), string(StatusSuccess),
	)
	return scanDeployRow(row)
}

func (s *Server) upsertGitHubPreview(item githubPreviewRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO github_previews
		 (repo_full_name, pr_number, app, branch, status_repo_full_name, head_sha, deploy_id, preview_url, status, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(repo_full_name, branch) DO UPDATE SET pr_number=excluded.pr_number,
		 app=excluded.app, status_repo_full_name=excluded.status_repo_full_name,
		 head_sha=excluded.head_sha, deploy_id=excluded.deploy_id, preview_url=excluded.preview_url,
		 status=excluded.status, updated_at=excluded.updated_at`,
		item.RepoFullName, item.PRNumber, item.App, item.Branch, item.StatusRepoFullName,
		item.HeadSHA, item.DeployID, item.PreviewURL, item.Status, item.UpdatedAt,
	)
	return err
}

func (s *Server) emitGitHubDeployStatusByID(deployID string) {
	deploy, err := s.getDeployFromDB(deployID)
	if err != nil || deploy == nil || strings.TrimSpace(deploy.CommitSHA) == "" {
		return
	}
	project, err := s.getGitHubProjectByApp(deploy.App)
	if err != nil || project == nil {
		return
	}
	if project.AuthMode == "app" {
		if err := s.emitGitHubCheckRun(deploy, project, ""); err != nil {
			fmt.Fprintf(os.Stderr, "github check run %s: %v\n", deploy.ID, err)
		}
		return
	}
	connection, err := s.getGitHubConnection()
	if err != nil || connection == nil {
		return
	}
	statusRepo := project.RepoFullName
	statusContext := project.StatusContext
	if deploy.Env == EnvProd {
		statusContext = "relay/production"
	}
	var preview githubPreviewRecord
	previewErr := s.db.QueryRow(
		`SELECT repo_full_name, pr_number, app, branch, status_repo_full_name, head_sha,
		 deploy_id, preview_url, status, updated_at FROM github_previews WHERE deploy_id=?`, deploy.ID,
	).Scan(&preview.RepoFullName, &preview.PRNumber, &preview.App, &preview.Branch, &preview.StatusRepoFullName, &preview.HeadSHA, &preview.DeployID, &preview.PreviewURL, &preview.Status, &preview.UpdatedAt)
	if previewErr == nil && strings.TrimSpace(preview.StatusRepoFullName) != "" {
		statusRepo = preview.StatusRepoFullName
	}

	state := "pending"
	description := "Relay is preparing this deployment"
	targetURL := deploy.PreviewURL
	switch deploy.Status {
	case StatusRunning:
		description = "Relay is building and checking the deployment"
	case StatusFailed:
		state = "failure"
		description = "Relay deployment failed"
	case StatusSuccess:
		lane, _ := s.getAppState(deploy.App, deploy.Env, deploy.Branch)
		if targetURL == "" {
			if lane != nil {
				targetURL = previewURLFromConfig(lane.Mode, lane.PublicHost, lane.HostPort)
			}
		}
		if deploy.Env == EnvProd && lane != nil && lane.RolloutStatus == "monitoring" {
			description = "Relay is monitoring production health"
			break
		}
		state = "success"
		description = "Relay deployment is healthy"
	}
	if previewErr == nil {
		_, _ = s.db.Exec(
			`UPDATE github_previews SET status=?, preview_url=?, updated_at=? WHERE deploy_id=?`,
			string(deploy.Status), targetURL, time.Now().UnixMilli(), deploy.ID,
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.createGitHubCommitStatus(ctx, connection.Token, statusRepo, deploy.CommitSHA, state, targetURL, description, statusContext); err != nil {
		fmt.Fprintf(os.Stderr, "github commit status %s: %v\n", deploy.ID, err)
	}
}

func (s *Server) githubDeployDetailsURL(deployID string) string {
	baseURL, err := s.githubPublicBaseURL()
	if err != nil {
		return ""
	}
	return baseURL + "/?deploy=" + url.QueryEscape(strings.TrimSpace(deployID))
}

func (s *Server) emitGitHubCheckRun(deploy *Deploy, project *githubProjectRecord, forcedFailure string) error {
	if deploy == nil || project == nil || project.InstallationID <= 0 || project.RepositoryID <= 0 || strings.TrimSpace(deploy.CommitSHA) == "" {
		return nil
	}
	s.githubCheckMu.Lock()
	defer s.githubCheckMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	token, err := s.githubInstallationToken(ctx, project.InstallationID, project.RepositoryID)
	if err != nil {
		return err
	}
	name := "Relay Preview"
	if deploy.Env == EnvProd {
		name = "Relay Production"
	}
	status := "queued"
	conclusion := ""
	title := fmt.Sprintf("Relay build #%d queued", deploy.BuildNumber)
	summary := fmt.Sprintf("Branch `%s` is waiting for a Relay build.", deploy.Branch)
	targetURL := deploy.PreviewURL
	if targetURL == "" {
		if lane, _ := s.getAppState(deploy.App, deploy.Env, deploy.Branch); lane != nil {
			targetURL = previewURLFromConfig(lane.Mode, lane.PublicHost, lane.HostPort)
		}
	}
	if forcedFailure != "" {
		status = "completed"
		conclusion = "failure"
		title = "Relay restored the previous production deployment"
		summary = forcedFailure
	} else {
		switch deploy.Status {
		case StatusRunning:
			status = "in_progress"
			title = fmt.Sprintf("Relay build #%d is running", deploy.BuildNumber)
			summary = "Relay is building the image and checking the runtime candidate."
		case StatusFailed:
			status = "completed"
			conclusion = "failure"
			title = fmt.Sprintf("Relay build #%d failed", deploy.BuildNumber)
			summary = firstNonEmpty(strings.TrimSpace(deploy.Error), "The Relay deployment failed. Open the deployment for complete logs.")
		case StatusSuccess:
			monitoring := false
			if deploy.Env == EnvProd {
				if lane, _ := s.getAppState(deploy.App, deploy.Env, deploy.Branch); lane != nil && lane.RolloutStatus == "monitoring" {
					monitoring = true
				}
			}
			if monitoring {
				status = "in_progress"
				title = "Relay is monitoring production health"
				summary = "The production candidate is receiving traffic while Relay evaluates its health window."
			} else {
				status = "completed"
				conclusion = "success"
				title = fmt.Sprintf("Relay build #%d is healthy", deploy.BuildNumber)
				summary = "Relay completed the deployment successfully."
			}
		}
	}
	if targetURL != "" {
		summary += "\n\n[Open deployment](" + targetURL + ")"
	}
	if deploy.Env == EnvPreview {
		_, _ = s.db.Exec(
			`UPDATE github_previews SET status=?, preview_url=?, updated_at=? WHERE deploy_id=?`,
			string(deploy.Status), targetURL, time.Now().UnixMilli(), deploy.ID,
		)
	}
	detailsURL := s.githubDeployDetailsURL(deploy.ID)
	if detailsURL != "" {
		summary += "\n\n[View build and runtime logs](" + detailsURL + ")"
	}
	payload := map[string]any{
		"status":      status,
		"details_url": detailsURL,
		"output": map[string]any{
			"title":   title,
			"summary": summary,
		},
	}
	if conclusion != "" {
		payload["conclusion"] = conclusion
		payload["completed_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	var checkRunID int64
	err = s.db.QueryRow(`SELECT check_run_id FROM github_check_runs WHERE deploy_id=?`, deploy.ID).Scan(&checkRunID)
	if err == sql.ErrNoRows {
		payload["name"] = name
		payload["head_sha"] = deploy.CommitSHA
		check, createErr := s.createGitHubCheckRun(ctx, token, project.RepoFullName, payload)
		if createErr != nil {
			return createErr
		}
		now := time.Now().UnixMilli()
		_, err = s.db.Exec(
			`INSERT INTO github_check_runs
			 (deploy_id, installation_id, repository_id, check_run_id, head_sha, check_name, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			deploy.ID, project.InstallationID, project.RepositoryID, check.ID, deploy.CommitSHA, name, now, now,
		)
		return err
	}
	if err != nil {
		return err
	}
	if err := s.updateGitHubCheckRun(ctx, token, project.RepoFullName, checkRunID, payload); err != nil {
		return err
	}
	_, _ = s.db.Exec(`UPDATE github_check_runs SET updated_at=? WHERE deploy_id=?`, time.Now().UnixMilli(), deploy.ID)
	return nil
}

func (s *Server) emitGitHubRolloutFailure(app string, env DeployEnv, branch string, detail string) {
	deploy, err := s.latestDeployForLane(app, env, branch)
	if err != nil || deploy == nil || strings.TrimSpace(deploy.CommitSHA) == "" {
		return
	}
	project, err := s.getGitHubProjectByApp(app)
	if err != nil || project == nil {
		return
	}
	description := "Relay rolled back the production rollout"
	if strings.TrimSpace(detail) != "" {
		description = "Relay rollback: " + strings.TrimSpace(detail)
	}
	if len(description) > 140 {
		description = description[:140]
	}
	if project.AuthMode == "app" {
		if err := s.emitGitHubCheckRun(deploy, project, description); err != nil {
			fmt.Fprintf(os.Stderr, "github rollout check %s: %v\n", deploy.ID, err)
		}
		return
	}
	connection, err := s.getGitHubConnection()
	if err != nil || connection == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.createGitHubCommitStatus(ctx, connection.Token, project.RepoFullName, deploy.CommitSHA, "failure", deploy.PreviewURL, description, "relay/production"); err != nil {
		fmt.Fprintf(os.Stderr, "github rollout status %s: %v\n", deploy.ID, err)
	}
}
