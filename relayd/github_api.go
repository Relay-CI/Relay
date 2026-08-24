package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultGitHubAPIURL = "https://api.github.com"

type githubUser struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type githubRepository struct {
	ID             int64  `json:"id"`
	InstallationID int64  `json:"installation_id,omitempty"`
	FullName       string `json:"full_name"`
	CloneURL       string `json:"clone_url"`
	HTMLURL        string `json:"html_url"`
	DefaultBranch  string `json:"default_branch"`
	Private        bool   `json:"private"`
	Permissions    struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
	} `json:"permissions"`
}

type githubInstallation struct {
	ID      int64 `json:"id"`
	AppID   int64 `json:"app_id"`
	Account struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"account"`
	RepositorySelection string     `json:"repository_selection"`
	SuspendedAt         *time.Time `json:"suspended_at"`
}

type githubInstallationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type githubWebhook struct {
	ID int64 `json:"id"`
}

type githubCheckRun struct {
	ID int64 `json:"id"`
}

type githubAPIError struct {
	Status  int
	Message string
}

func (e *githubAPIError) Error() string {
	return fmt.Sprintf("GitHub returned HTTP %d: %s", e.Status, e.Message)
}

func (s *Server) githubAPIBaseURL() string {
	if value := strings.TrimSpace(s.githubAPIURL); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultGitHubAPIURL
}

func (s *Server) githubClient() *http.Client {
	if s.githubHTTPClient != nil {
		return s.githubHTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (s *Server) githubRequest(ctx context.Context, method string, path string, token string, payload any, result any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode GitHub request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.githubAPIBaseURL()+path, body)
	if err != nil {
		return fmt.Errorf("build GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Relay/"+strings.TrimPrefix(relaydVersion, "v"))
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.githubClient().Do(req)
	if err != nil {
		return fmt.Errorf("call GitHub: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read GitHub response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiMessage struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(responseBody, &apiMessage)
		if strings.TrimSpace(apiMessage.Message) == "" {
			apiMessage.Message = strings.TrimSpace(string(responseBody))
		}
		if apiMessage.Message == "" {
			apiMessage.Message = http.StatusText(resp.StatusCode)
		}
		return &githubAPIError{Status: resp.StatusCode, Message: apiMessage.Message}
	}
	if result != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, result); err != nil {
			return fmt.Errorf("decode GitHub response: %w", err)
		}
	}
	return nil
}

func githubRepoPath(fullName string) (string, error) {
	parts := strings.Split(strings.TrimSpace(fullName), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("repository must use owner/name format")
	}
	return "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]), nil
}

func (s *Server) fetchGitHubUser(ctx context.Context, token string) (githubUser, error) {
	var user githubUser
	err := s.githubRequest(ctx, http.MethodGet, "/user", token, nil, &user)
	return user, err
}

func (s *Server) convertGitHubAppManifest(ctx context.Context, code string) (githubAppManifestConversion, error) {
	var conversion githubAppManifestConversion
	path := "/app-manifests/" + url.PathEscape(strings.TrimSpace(code)) + "/conversions"
	err := s.githubRequest(ctx, http.MethodPost, path, "", nil, &conversion)
	return conversion, err
}

func (s *Server) fetchGitHubRepositories(ctx context.Context, token string) ([]githubRepository, error) {
	var repos []githubRepository
	err := s.githubRequest(ctx, http.MethodGet, "/user/repos?per_page=100&sort=updated&affiliation=owner,collaborator,organization_member", token, nil, &repos)
	return repos, err
}

func (s *Server) fetchGitHubInstallation(ctx context.Context, jwt string, installationID int64) (githubInstallation, error) {
	var installation githubInstallation
	err := s.githubRequest(ctx, http.MethodGet, fmt.Sprintf("/app/installations/%d", installationID), jwt, nil, &installation)
	return installation, err
}

func (s *Server) createGitHubInstallationToken(ctx context.Context, jwt string, installationID int64, repositoryID int64) (githubInstallationTokenResponse, error) {
	payload := map[string]any{
		"permissions": map[string]string{"contents": "read", "checks": "write"},
	}
	if repositoryID > 0 {
		payload["repository_ids"] = []int64{repositoryID}
	}
	var response githubInstallationTokenResponse
	err := s.githubRequest(ctx, http.MethodPost, fmt.Sprintf("/app/installations/%d/access_tokens", installationID), jwt, payload, &response)
	return response, err
}

func (s *Server) fetchGitHubInstallationRepositories(ctx context.Context, token string) ([]githubRepository, error) {
	var response struct {
		Repositories []githubRepository `json:"repositories"`
	}
	err := s.githubRequest(ctx, http.MethodGet, "/installation/repositories?per_page=100", token, nil, &response)
	return response.Repositories, err
}

func (s *Server) fetchGitHubRepository(ctx context.Context, token string, fullName string) (githubRepository, error) {
	path, err := githubRepoPath(fullName)
	if err != nil {
		return githubRepository{}, err
	}
	var repo githubRepository
	err = s.githubRequest(ctx, http.MethodGet, path, token, nil, &repo)
	return repo, err
}

func (s *Server) createGitHubWebhook(ctx context.Context, token string, fullName string, callbackURL string, secret string) (githubWebhook, error) {
	path, err := githubRepoPath(fullName)
	if err != nil {
		return githubWebhook{}, err
	}
	payload := map[string]any{
		"name":   "web",
		"active": true,
		"events": []string{"push", "pull_request"},
		"config": map[string]string{
			"url":          callbackURL,
			"content_type": "json",
			"insecure_ssl": "0",
			"secret":       secret,
		},
	}
	var hook githubWebhook
	err = s.githubRequest(ctx, http.MethodPost, path+"/hooks", token, payload, &hook)
	return hook, err
}

func (s *Server) updateGitHubWebhook(ctx context.Context, token string, fullName string, hookID int64, callbackURL string, secret string) error {
	path, err := githubRepoPath(fullName)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"active": true,
		"events": []string{"push", "pull_request"},
		"config": map[string]string{
			"url":          callbackURL,
			"content_type": "json",
			"insecure_ssl": "0",
			"secret":       secret,
		},
	}
	return s.githubRequest(ctx, http.MethodPatch, fmt.Sprintf("%s/hooks/%d", path, hookID), token, payload, nil)
}

func (s *Server) deleteGitHubWebhook(ctx context.Context, token string, fullName string, hookID int64) error {
	if hookID <= 0 {
		return nil
	}
	path, err := githubRepoPath(fullName)
	if err != nil {
		return err
	}
	return s.githubRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/hooks/%d", path, hookID), token, nil, nil)
}

func (s *Server) createGitHubCommitStatus(ctx context.Context, token string, fullName string, sha string, state string, targetURL string, description string, statusContext string) error {
	path, err := githubRepoPath(fullName)
	if err != nil {
		return err
	}
	payload := map[string]string{
		"state":       state,
		"description": description,
		"context":     statusContext,
	}
	if strings.TrimSpace(targetURL) != "" {
		payload["target_url"] = targetURL
	}
	return s.githubRequest(ctx, http.MethodPost, path+"/statuses/"+url.PathEscape(strings.TrimSpace(sha)), token, payload, nil)
}

func (s *Server) createGitHubCheckRun(ctx context.Context, token string, fullName string, payload map[string]any) (githubCheckRun, error) {
	path, err := githubRepoPath(fullName)
	if err != nil {
		return githubCheckRun{}, err
	}
	var check githubCheckRun
	err = s.githubRequest(ctx, http.MethodPost, path+"/check-runs", token, payload, &check)
	return check, err
}

func (s *Server) updateGitHubCheckRun(ctx context.Context, token string, fullName string, checkRunID int64, payload map[string]any) error {
	path, err := githubRepoPath(fullName)
	if err != nil {
		return err
	}
	return s.githubRequest(ctx, http.MethodPatch, fmt.Sprintf("%s/check-runs/%d", path, checkRunID), token, payload, nil)
}
