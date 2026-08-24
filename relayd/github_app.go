package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultGitHubAppConfigID = "default"

var githubAccountLoginPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

type githubAppConfig struct {
	ID            string
	AppID         int64
	ClientID      string
	AppSlug       string
	AppName       string
	OwnerLogin    string
	PrivateKey    string
	WebhookSecret string
	CreatedAt     int64
	UpdatedAt     int64
}

type githubAppManifestConversion struct {
	ID            int64  `json:"id"`
	ClientID      string `json:"client_id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	PEM           string `json:"pem"`
	WebhookSecret string `json:"webhook_secret"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type githubAppView struct {
	Registered   bool   `json:"registered"`
	AppID        int64  `json:"app_id,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	AppSlug      string `json:"app_slug,omitempty"`
	AppName      string `json:"app_name,omitempty"`
	OwnerLogin   string `json:"owner_login,omitempty"`
	InstallURL   string `json:"install_url,omitempty"`
	WebhookURL   string `json:"webhook_url"`
	RegisteredAt int64  `json:"registered_at,omitempty"`
}

type githubCachedInstallationToken struct {
	Token     string
	ExpiresAt time.Time
}

func (s *Server) githubPublicBaseURL() (string, error) {
	webhookURL := s.githubWebhookURL()
	if err := validateGitHubWebhookURL(webhookURL); err != nil {
		return "", err
	}
	u, err := url.Parse(webhookURL)
	if err != nil {
		return "", err
	}
	return u.Scheme + "://" + u.Host, nil
}

func newGitHubAppState() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	state := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(state))
	return state, hex.EncodeToString(hash[:]), nil
}

func githubAppStateHash(state string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(state)))
	return hex.EncodeToString(hash[:])
}

func (s *Server) issueGitHubAppState(purpose string, actor string) (string, error) {
	state, stateHash, err := newGitHubAppState()
	if err != nil {
		return "", err
	}
	now := time.Now().UnixMilli()
	_, err = s.db.Exec(
		`INSERT INTO github_app_states (state_hash, purpose, actor, expires_at, used_at, created_at)
		 VALUES (?, ?, ?, ?, 0, ?)`,
		stateHash, purpose, actor, time.Now().Add(15*time.Minute).UnixMilli(), now,
	)
	return state, err
}

func (s *Server) consumeGitHubAppState(state string, purpose string) bool {
	now := time.Now().UnixMilli()
	result, err := s.db.Exec(
		`UPDATE github_app_states SET used_at=?
		 WHERE state_hash=? AND purpose=? AND used_at=0 AND expires_at>?`,
		now, githubAppStateHash(state), purpose, now,
	)
	if err != nil {
		return false
	}
	rows, _ := result.RowsAffected()
	return rows == 1
}

func parseGitHubAppPrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(value)))
	if block == nil {
		return nil, fmt.Errorf("GitHub returned an invalid App private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("GitHub returned an invalid App private key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("GitHub returned a non-RSA App private key")
	}
	return key, nil

}

func validateGitHubAppPrivateKey(value string) error {
	_, err := parseGitHubAppPrivateKey(value)
	return err
}

func base64URLJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil

}

func githubAppJWT(config *githubAppConfig, now time.Time) (string, error) {
	key, err := parseGitHubAppPrivateKey(config.PrivateKey)
	if err != nil {
		return "", err
	}
	header, err := base64URLJSON(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	issuer := strings.TrimSpace(config.ClientID)
	if issuer == "" {
		issuer = strconv.FormatInt(config.AppID, 10)
	}
	payload, err := base64URLJSON(map[string]any{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": issuer,
	})
	if err != nil {
		return "", err
	}
	unsigned := header + "." + payload
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *Server) githubInstallationToken(ctx context.Context, installationID int64, repositoryID int64) (string, error) {
	cacheKey := fmt.Sprintf("%d:%d", installationID, repositoryID)
	s.githubTokenMu.Lock()
	defer s.githubTokenMu.Unlock()
	if cached, ok := s.githubTokens[cacheKey]; ok && cached.Token != "" && time.Until(cached.ExpiresAt) > 5*time.Minute {
		return cached.Token, nil
	}
	config, err := s.getGitHubAppConfig()
	if err != nil {
		return "", fmt.Errorf("load GitHub App: %w", err)
	}
	jwt, err := githubAppJWT(config, time.Now())
	if err != nil {
		return "", fmt.Errorf("sign GitHub App JWT: %w", err)
	}
	response, err := s.createGitHubInstallationToken(ctx, jwt, installationID, repositoryID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(response.Token) == "" {
		return "", fmt.Errorf("GitHub returned an empty installation token")
	}
	if s.githubTokens == nil {
		s.githubTokens = map[string]githubCachedInstallationToken{}
	}
	s.githubTokens[cacheKey] = githubCachedInstallationToken{Token: response.Token, ExpiresAt: response.ExpiresAt}
	return response.Token, nil
}

func (s *Server) invalidateGitHubInstallationTokens(installationID int64) {
	prefix := strconv.FormatInt(installationID, 10) + ":"
	s.githubTokenMu.Lock()
	defer s.githubTokenMu.Unlock()
	for key := range s.githubTokens {
		if strings.HasPrefix(key, prefix) {
			delete(s.githubTokens, key)
		}
	}
}

func (s *Server) getGitHubAppConfig() (*githubAppConfig, error) {
	var config githubAppConfig
	var privateKeyEnc, webhookSecretEnc string
	err := s.db.QueryRow(
		`SELECT id, app_id, client_id, app_slug, app_name, owner_login,
		 private_key_enc, webhook_secret_enc, created_at, updated_at
		 FROM github_app_config WHERE id=?`, defaultGitHubAppConfigID,
	).Scan(&config.ID, &config.AppID, &config.ClientID, &config.AppSlug, &config.AppName,
		&config.OwnerLogin, &privateKeyEnc, &webhookSecretEnc, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}
	config.PrivateKey = s.decryptSecret(privateKeyEnc)
	config.WebhookSecret = s.decryptSecret(webhookSecretEnc)
	return &config, nil
}

func (s *Server) githubAppView() githubAppView {
	view := githubAppView{WebhookURL: s.githubWebhookURL()}
	config, err := s.getGitHubAppConfig()
	if err != nil {
		return view
	}
	view.Registered = true
	view.AppID = config.AppID
	view.ClientID = config.ClientID
	view.AppSlug = config.AppSlug
	view.AppName = config.AppName
	view.OwnerLogin = config.OwnerLogin
	view.InstallURL = "https://github.com/apps/" + url.PathEscape(config.AppSlug) + "/installations/new"
	view.RegisteredAt = config.CreatedAt
	return view
}

func (s *Server) handleGitHubAppManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.relaySecretKeyConfigured() {
		httpError(w, http.StatusConflict, "set RELAY_SECRET_KEY before registering a GitHub App")
		return
	}
	var request struct {
		Organization string `json:"organization"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json")
		return
	}
	request.Organization = strings.TrimSpace(request.Organization)
	if request.Organization != "" && !githubAccountLoginPattern.MatchString(request.Organization) {
		httpError(w, http.StatusBadRequest, "organization must be a valid GitHub account name")
		return
	}
	baseURL, err := s.githubPublicBaseURL()
	if err != nil {
		httpError(w, http.StatusConflict, err.Error())
		return
	}
	state, err := s.issueGitHubAppState("manifest", requestActorLabel(s, r))
	if err != nil {
		httpError(w, http.StatusInternalServerError, "create GitHub App registration state: "+err.Error())
		return
	}
	host := strings.TrimSpace(s.serverDashboardHost())
	nameSuffix := strings.NewReplacer(".", "-", ":", "-").Replace(host)
	if len(nameSuffix) > 35 {
		nameSuffix = nameSuffix[:35]
	}
	manifest := map[string]any{
		"name":         "Relay " + nameSuffix,
		"url":          baseURL,
		"redirect_url": baseURL + "/api/github/app/manifest/callback",
		"setup_url":    baseURL + "/api/github/app/setup",
		"hook_attributes": map[string]any{
			"url":    baseURL + "/api/webhooks/github",
			"active": true,
		},
		"public": false,
		"default_permissions": map[string]string{
			"contents": "read",
			"checks":   "write",
		},
		"default_events": []string{"push", "pull_request", "check_run"},
	}
	action := "https://github.com/settings/apps/new"
	if request.Organization != "" {
		action = "https://github.com/organizations/" + url.PathEscape(request.Organization) + "/settings/apps/new"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"action":   action,
		"state":    state,
		"manifest": manifest,
	})
}

func (s *Server) handleGitHubAppManifestCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || state == "" || !s.consumeGitHubAppState(state, "manifest") {
		httpError(w, http.StatusBadRequest, "invalid, expired, or already used GitHub App registration state")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	conversion, err := s.convertGitHubAppManifest(ctx, code)
	if err != nil {
		httpError(w, http.StatusBadGateway, "complete GitHub App registration: "+err.Error())
		return
	}
	if conversion.ID <= 0 || strings.TrimSpace(conversion.Slug) == "" || strings.TrimSpace(conversion.WebhookSecret) == "" {
		httpError(w, http.StatusBadGateway, "GitHub returned incomplete App credentials")
		return
	}
	if err := validateGitHubAppPrivateKey(conversion.PEM); err != nil {
		httpError(w, http.StatusBadGateway, err.Error())
		return
	}
	now := time.Now().UnixMilli()
	_, err = s.db.Exec(
		`INSERT INTO github_app_config
		 (id, app_id, client_id, app_slug, app_name, owner_login, private_key_enc, webhook_secret_enc, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET app_id=excluded.app_id, client_id=excluded.client_id,
		 app_slug=excluded.app_slug, app_name=excluded.app_name, owner_login=excluded.owner_login,
		 private_key_enc=excluded.private_key_enc, webhook_secret_enc=excluded.webhook_secret_enc,
		 updated_at=excluded.updated_at`,
		defaultGitHubAppConfigID, conversion.ID, conversion.ClientID, conversion.Slug, conversion.Name,
		conversion.Owner.Login, s.encryptSecret(conversion.PEM), s.encryptSecret(conversion.WebhookSecret), now, now,
	)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "store GitHub App credentials: "+err.Error())
		return
	}
	s.auditLog(requestActorLabel(s, r), "github.app.register", conversion.Slug, "GitHub App manifest completed")
	http.Redirect(w, r, "/?github=registered", http.StatusSeeOther)
}

func (s *Server) handleGitHubApp(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.githubAppView())
	case http.MethodDelete:
		var installationCount int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM github_installations WHERE status!='deleted'`).Scan(&installationCount)
		if installationCount > 0 {
			httpError(w, http.StatusConflict, "uninstall active GitHub App installations first")
			return
		}
		if _, err := s.db.Exec(`DELETE FROM github_app_config WHERE id=?`, defaultGitHubAppConfigID); err != nil && err != sql.ErrNoRows {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s.githubAppView())
	default:
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGitHubAppInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	config, err := s.getGitHubAppConfig()
	if err != nil {
		httpError(w, http.StatusConflict, "register the GitHub App first")
		return
	}
	state, err := s.issueGitHubAppState("install", requestActorLabel(s, r))
	if err != nil {
		httpError(w, http.StatusInternalServerError, "create GitHub installation state: "+err.Error())
		return
	}
	installURL := "https://github.com/apps/" + url.PathEscape(config.AppSlug) + "/installations/new?state=" + url.QueryEscape(state)
	writeJSON(w, http.StatusOK, map[string]string{"install_url": installURL, "state": state})
}

func (s *Server) saveGitHubInstallation(installation githubInstallation, repositories []githubRepository) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	status := "active"
	suspendedAt := int64(0)
	if installation.SuspendedAt != nil {
		status = "suspended"
		suspendedAt = installation.SuspendedAt.UnixMilli()
	}
	_, err = tx.Exec(
		`INSERT INTO github_installations
		 (installation_id, account_id, account_login, account_type, repository_selection, status, suspended_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(installation_id) DO UPDATE SET account_id=excluded.account_id,
		 account_login=excluded.account_login, account_type=excluded.account_type,
		 repository_selection=excluded.repository_selection, status=excluded.status,
		 suspended_at=excluded.suspended_at, updated_at=excluded.updated_at`,
		installation.ID, installation.Account.ID, installation.Account.Login, installation.Account.Type,
		installation.RepositorySelection, status, suspendedAt, now, now,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`UPDATE github_installation_repositories SET active=0, updated_at=? WHERE installation_id=?`, now, installation.ID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, repository := range repositories {
		_, err = tx.Exec(
			`INSERT INTO github_installation_repositories
			 (installation_id, repository_id, full_name, clone_url, html_url, default_branch, private, active, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)
			 ON CONFLICT(installation_id, repository_id) DO UPDATE SET full_name=excluded.full_name,
			 clone_url=excluded.clone_url, html_url=excluded.html_url, default_branch=excluded.default_branch,
			 private=excluded.private, active=1, updated_at=excluded.updated_at`,
			installation.ID, repository.ID, repository.FullName, repository.CloneURL, repository.HTMLURL,
			repository.DefaultBranch, boolToInt(repository.Private), now,
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Server) syncGitHubInstallation(ctx context.Context, installation githubInstallation) error {
	token, err := s.githubInstallationToken(ctx, installation.ID, 0)
	if err != nil {
		return fmt.Errorf("create installation token: %w", err)
	}
	repositories, err := s.fetchGitHubInstallationRepositories(ctx, token)
	if err != nil {
		return fmt.Errorf("list installation repositories: %w", err)
	}
	return s.saveGitHubInstallation(installation, repositories)
}

func (s *Server) handleGitHubAppSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	installationID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("installation_id")), 10, 64)
	if err != nil || installationID <= 0 || !s.consumeGitHubAppState(state, "install") {
		httpError(w, http.StatusBadRequest, "invalid, expired, or already used GitHub App installation state")
		return
	}
	config, err := s.getGitHubAppConfig()
	if err != nil {
		httpError(w, http.StatusConflict, "register the GitHub App first")
		return
	}
	jwt, err := githubAppJWT(config, time.Now())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "authenticate GitHub App: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	installation, err := s.fetchGitHubInstallation(ctx, jwt, installationID)
	if err != nil || installation.ID != installationID || installation.AppID != config.AppID {
		httpError(w, http.StatusBadGateway, "GitHub App installation could not be verified")
		return
	}
	if err := s.syncGitHubInstallation(ctx, installation); err != nil {
		httpError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.auditLog(requestActorLabel(s, r), "github.app.install", installation.Account.Login, fmt.Sprintf("installation=%d", installation.ID))
	http.Redirect(w, r, "/?github=installed", http.StatusSeeOther)
}

func (s *Server) listGitHubInstallationRepositories() ([]githubRepository, error) {
	rows, err := s.db.Query(
		`SELECT r.repository_id, r.installation_id, r.full_name, r.clone_url, r.html_url,
		 r.default_branch, r.private
		 FROM github_installation_repositories r
		 JOIN github_installations i ON i.installation_id=r.installation_id
		 WHERE r.active=1 AND i.status='active' ORDER BY r.full_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []githubRepository{}
	for rows.Next() {
		var item githubRepository
		var private int
		if err := rows.Scan(&item.ID, &item.InstallationID, &item.FullName, &item.CloneURL, &item.HTMLURL, &item.DefaultBranch, &private); err != nil {
			return nil, err
		}
		item.Private = private != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) getGitHubInstallationRepository(fullName string) (*githubRepository, error) {
	var item githubRepository
	var private int
	err := s.db.QueryRow(
		`SELECT r.repository_id, r.installation_id, r.full_name, r.clone_url, r.html_url,
		 r.default_branch, r.private
		 FROM github_installation_repositories r
		 JOIN github_installations i ON i.installation_id=r.installation_id
		 WHERE LOWER(r.full_name)=LOWER(?) AND r.active=1 AND i.status='active'`,
		strings.TrimSpace(fullName),
	).Scan(&item.ID, &item.InstallationID, &item.FullName, &item.CloneURL, &item.HTMLURL, &item.DefaultBranch, &private)
	if err != nil {
		return nil, err
	}
	item.Private = private != 0
	return &item, nil
}

func (s *Server) githubInstallationActive(installationID int64) bool {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM github_installations WHERE installation_id=? AND status='active'`, installationID).Scan(&count)
	return err == nil && count == 1
}

func (s *Server) githubInstallationRepositoryActive(installationID int64, repositoryID int64) bool {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM github_installation_repositories r
		 JOIN github_installations i ON i.installation_id=r.installation_id
		 WHERE r.installation_id=? AND r.repository_id=? AND r.active=1 AND i.status='active'`,
		installationID, repositoryID,
	).Scan(&count)
	return err == nil && count == 1
}

func (s *Server) markGitHubInstallationUnavailable(installationID int64, status string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	suspendedAt := int64(0)
	if status == "suspended" {
		suspendedAt = now
	}
	if _, err = tx.Exec(`UPDATE github_installations SET status=?, suspended_at=?, updated_at=? WHERE installation_id=?`, status, suspendedAt, now, installationID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`UPDATE github_installation_repositories SET active=0, updated_at=? WHERE installation_id=?`, now, installationID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.invalidateGitHubInstallationTokens(installationID)
	return nil
}

func (s *Server) updateGitHubInstallationRepositories(installationID int64, added []githubRepository, removed []githubRepository) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, repository := range removed {
		if _, err = tx.Exec(`UPDATE github_installation_repositories SET active=0, updated_at=? WHERE installation_id=? AND repository_id=?`, now, installationID, repository.ID); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	for _, repository := range added {
		if _, err = tx.Exec(
			`INSERT INTO github_installation_repositories
			 (installation_id, repository_id, full_name, clone_url, html_url, default_branch, private, active, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)
			 ON CONFLICT(installation_id, repository_id) DO UPDATE SET full_name=excluded.full_name,
			 clone_url=excluded.clone_url, html_url=excluded.html_url, default_branch=excluded.default_branch,
			 private=excluded.private, active=1, updated_at=excluded.updated_at`,
			installationID, repository.ID, repository.FullName, repository.CloneURL, repository.HTMLURL,
			repository.DefaultBranch, boolToInt(repository.Private), now,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.invalidateGitHubInstallationTokens(installationID)
	return nil
}

func (s *Server) handleGitHubAppLifecycleWebhook(w http.ResponseWriter, r *http.Request, event string, body []byte) bool {
	if event != "installation" && event != "installation_repositories" {
		return false
	}
	config, err := s.getGitHubAppConfig()
	if err != nil || config == nil {
		return false
	}
	if !verifyGithubSig256([]byte(config.WebhookSecret), body, r.Header.Get("X-Hub-Signature-256")) {
		httpError(w, http.StatusUnauthorized, "invalid GitHub App signature")
		return true
	}
	var payload githubWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil || payload.Installation.ID <= 0 {
		httpError(w, http.StatusBadRequest, "invalid GitHub App lifecycle payload")
		return true
	}
	deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
	if deliveryID == "" {
		deliveryID = fileHashByAlgoBytes(body, event)
	}
	claimed, err := s.claimGitHubDelivery(deliveryID, event, payload.Action, fmt.Sprintf("installation:%d", payload.Installation.ID))
	if err != nil {
		httpError(w, http.StatusInternalServerError, "record GitHub App delivery: "+err.Error())
		return true
	}
	if !claimed {
		writeJSON(w, http.StatusOK, map[string]any{"status": "duplicate", "delivery_id": deliveryID})
		return true
	}
	outcome := payload.Action
	switch event {
	case "installation":
		switch payload.Action {
		case "deleted":
			err = s.markGitHubInstallationUnavailable(payload.Installation.ID, "deleted")
		case "suspend":
			err = s.markGitHubInstallationUnavailable(payload.Installation.ID, "suspended")
		case "created", "unsuspend", "new_permissions_accepted":
			s.invalidateGitHubInstallationTokens(payload.Installation.ID)
			ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
			err = s.syncGitHubInstallation(ctx, payload.Installation)
			cancel()
		default:
			outcome = "ignored"
		}
	case "installation_repositories":
		err = s.updateGitHubInstallationRepositories(payload.Installation.ID, payload.RepositoriesAdded, payload.RepositoriesRemoved)
	}
	if err != nil {
		_, _ = s.db.Exec(`DELETE FROM github_deliveries WHERE delivery_id=?`, deliveryID)
		httpError(w, http.StatusInternalServerError, "process GitHub App lifecycle: "+err.Error())
		return true
	}
	_, _ = s.db.Exec(`UPDATE github_deliveries SET outcome=? WHERE delivery_id=?`, outcome, deliveryID)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": outcome, "delivery_id": deliveryID})
	return true
}
