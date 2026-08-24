package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeGitHubServer struct {
	server         *httptest.Server
	mu             sync.Mutex
	hooks          int
	status         []map[string]string
	manifestPEM    string
	appConversions int
	tokenRequests  int
	checks         []map[string]any
	shortTokens    bool
}

func newFakeGitHubServer(t *testing.T) *fakeGitHubServer {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate fake GitHub App key: %v", err)
	}
	fake := &fakeGitHubServer{manifestPEM: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}))}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/app-manifests/manifest-code/conversions" {
			fake.mu.Lock()
			fake.appConversions++
			fake.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             991,
				"client_id":      "Iv1.relay-test",
				"slug":           "relay-test",
				"name":           "Relay Test",
				"pem":            fake.manifestPEM,
				"webhook_secret": "app-webhook-secret",
				"owner":          map[string]string{"login": "acme"},
			})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/app/installations/7001" && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer eyJ") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":7001,"app_id":991,"account":{"id":88,"login":"acme","type":"Organization"},"repository_selection":"selected","suspended_at":null}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/app/installations/7001/access_tokens" && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer eyJ") {
			fake.mu.Lock()
			fake.tokenRequests++
			shortTokens := fake.shortTokens
			fake.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			expiresAt := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
			if shortTokens {
				expiresAt = time.Now().Add(time.Minute)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "github-installation-token", "expires_at": expiresAt})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/installation/repositories" && r.Header.Get("Authorization") == "Bearer github-installation-token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total_count":1,"repositories":[{"id":501,"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main","private":true}]}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/repos/acme/widget" && r.Header.Get("Authorization") == "Bearer github-installation-token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":501,"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main","private":true}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/repos/acme/widget/check-runs" && r.Header.Get("Authorization") == "Bearer github-installation-token" {
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			payload["method"] = "POST"
			fake.mu.Lock()
			fake.checks = append(fake.checks, payload)
			fake.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":8101}`))
			return
		}
		if r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/widget/check-runs/8101" && r.Header.Get("Authorization") == "Bearer github-installation-token" {
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			payload["method"] = "PATCH"
			fake.mu.Lock()
			fake.checks = append(fake.checks, payload)
			fake.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":8101}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer github-test-token" {
			http.Error(w, `{"message":"bad token"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user":
			_, _ = w.Write([]byte(`{"login":"octo-owner","avatar_url":"https://avatars.example/octo","html_url":"https://github.com/octo-owner"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/user/repos":
			_, _ = w.Write([]byte(`[{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main","private":true,"permissions":{"admin":true,"push":true}}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/widget":
			_, _ = w.Write([]byte(`{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main","private":true,"permissions":{"admin":true,"push":true}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/widget/hooks":
			var payload struct {
				Events []string          `json:"events"`
				Config map[string]string `json:"config"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if payload.Config["url"] != "https://relay.example.com/api/webhooks/github" || payload.Config["secret"] == "" {
				http.Error(w, `{"message":"invalid hook"}`, http.StatusUnprocessableEntity)
				return
			}
			fake.mu.Lock()
			fake.hooks++
			fake.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":42}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/widget/hooks/42":
			_, _ = w.Write([]byte(`{"id":42}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/acme/widget/hooks/42":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/repos/acme/widget/statuses/"):
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)
			fake.mu.Lock()
			fake.status = append(fake.status, payload)
			fake.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		}
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func newGitHubWorkflowTestServer(t *testing.T, fake *fakeGitHubServer) *Server {
	t.Helper()
	dataDir := t.TempDir()
	store, err := openSQLiteStore(filepath.Join(dataDir, "github-workflow.db"))
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	db := store.Primary
	t.Cleanup(func() { _ = store.Close() })
	if _, err := db.Exec(`INSERT INTO server_config(key,value) VALUES('dashboard_host','relay.example.com')`); err != nil {
		t.Fatalf("configure dashboard host: %v", err)
	}
	s := &Server{
		deploys:          map[string]*Deploy{},
		queue:            make(chan DeployJob, 16),
		eventsChans:      map[chan []byte]struct{}{},
		dataDir:          dataDir,
		workspacesDir:    filepath.Join(dataDir, "workspaces"),
		logsDir:          filepath.Join(dataDir, "logs"),
		db:               db,
		secretKey:        bytes.Repeat([]byte{7}, 32),
		webhookHits:      map[string][]time.Time{},
		githubAPIURL:     fake.server.URL,
		githubHTTPClient: fake.server.Client(),
		runtime:          &mockRuntime{running: map[string]bool{}, exists: map[string]bool{}, published: map[string]int{}},
	}
	t.Cleanup(s.waitGitHubDeployStatusAsync)
	return s
}

func invokeJSONHandler(t *testing.T, handler http.HandlerFunc, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

func connectGitHubTestProject(t *testing.T, s *Server) *githubProjectRecord {
	t.Helper()
	response := invokeJSONHandler(t, s.handleGitHubConnection, http.MethodPost, "/api/github/connection", `{"token":"github-test-token"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("connect GitHub: status=%d body=%s", response.Code, response.Body.String())
	}
	response = invokeJSONHandler(t, s.handleGitHubProjects, http.MethodPost, "/api/github/projects", `{"app":"widget","repo_full_name":"acme/widget","production_branch":"main"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("connect GitHub project: status=%d body=%s", response.Code, response.Body.String())
	}
	project, err := s.getGitHubProjectByApp("widget")
	if err != nil {
		t.Fatalf("load GitHub project: %v", err)
	}
	return project
}

func signedGitHubWebhookRequest(t *testing.T, project *githubProjectRecord, event string, deliveryID string, payload string) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(project.WebhookSecret))
	_, _ = mac.Write([]byte(payload))
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", strings.NewReader(payload))
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-GitHub-Delivery", deliveryID)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	return req
}

func TestGitHubConnectionAndProjectCreateEncryptedWebhook(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	project := connectGitHubTestProject(t, s)

	var storedToken string
	if err := s.db.QueryRow(`SELECT token_enc FROM github_connections WHERE id='default'`).Scan(&storedToken); err != nil {
		t.Fatalf("read stored GitHub token: %v", err)
	}
	if !strings.HasPrefix(storedToken, "enc:") || strings.Contains(storedToken, "github-test-token") {
		t.Fatalf("GitHub token was not encrypted: %q", storedToken)
	}
	if project.WebhookID != 42 || project.ProductionBranch != "main" || !project.PreviewEnabled || !project.ProductionEnabled {
		t.Fatalf("unexpected GitHub project: %+v", project)
	}
	if project.WebhookSecret == "" {
		t.Fatal("generated webhook secret was not persisted")
	}
	state, err := s.getAppState("widget", EnvProd, "main")
	if err != nil || state == nil || state.RepoURL != "https://github.com/acme/widget.git" {
		t.Fatalf("production lane was not seeded from GitHub: state=%+v err=%v", state, err)
	}
	fake.mu.Lock()
	hookCount := fake.hooks
	fake.mu.Unlock()
	if hookCount != 1 {
		t.Fatalf("created %d GitHub hooks, want 1", hookCount)
	}
}

func TestGitHubAppManifestRegistrationIsEncryptedAndSingleUse(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)

	start := invokeJSONHandler(t, s.handleGitHubAppManifest, http.MethodPost, "/api/github/app/manifest", `{}`)
	if start.Code != http.StatusOK {
		t.Fatalf("start manifest: status=%d body=%s", start.Code, start.Body.String())
	}
	var manifest struct {
		Action   string         `json:"action"`
		State    string         `json:"state"`
		Manifest map[string]any `json:"manifest"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("decode manifest response: %v", err)
	}
	if manifest.Action != "https://github.com/settings/apps/new" || manifest.State == "" {
		t.Fatalf("unexpected manifest start: %+v", manifest)
	}
	permissions, _ := manifest.Manifest["default_permissions"].(map[string]any)
	if permissions["contents"] != "read" || permissions["checks"] != "write" {
		t.Fatalf("manifest permissions=%v", permissions)
	}

	callbackRequest := httptest.NewRequest(http.MethodGet, "/api/github/app/manifest/callback?code=manifest-code&state="+manifest.State, nil)
	callback := httptest.NewRecorder()
	s.handleGitHubAppManifestCallback(callback, callbackRequest)
	if callback.Code != http.StatusSeeOther {
		t.Fatalf("manifest callback: status=%d body=%s", callback.Code, callback.Body.String())
	}

	status := invokeJSONHandler(t, s.handleGitHubApp, http.MethodGet, "/api/github/app", "")
	if status.Code != http.StatusOK {
		t.Fatalf("read app status: status=%d body=%s", status.Code, status.Body.String())
	}
	var appStatus struct {
		Registered bool   `json:"registered"`
		Slug       string `json:"app_slug"`
		InstallURL string `json:"install_url"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &appStatus); err != nil {
		t.Fatalf("decode app status: %v", err)
	}
	if !appStatus.Registered || appStatus.Slug != "relay-test" || !strings.Contains(appStatus.InstallURL, "/apps/relay-test/installations/new") {
		t.Fatalf("unexpected app status: %+v", appStatus)
	}

	replay := httptest.NewRecorder()
	s.handleGitHubAppManifestCallback(replay, callbackRequest.Clone(callbackRequest.Context()))
	if replay.Code != http.StatusBadRequest {
		t.Fatalf("replayed manifest state: status=%d body=%s", replay.Code, replay.Body.String())
	}
	var privateKeyEnc, webhookSecretEnc string
	if err := s.db.QueryRow(`SELECT private_key_enc, webhook_secret_enc FROM github_app_config WHERE id='default'`).Scan(&privateKeyEnc, &webhookSecretEnc); err != nil {
		t.Fatalf("read stored app credentials: %v", err)
	}
	if !strings.HasPrefix(privateKeyEnc, "enc:") || strings.Contains(privateKeyEnc, "PRIVATE KEY") || !strings.HasPrefix(webhookSecretEnc, "enc:") {
		t.Fatal("GitHub App credentials were not encrypted")
	}
}

func TestGitHubAppManifestCanBeOwnedByAnOrganization(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	start := invokeJSONHandler(t, s.handleGitHubAppManifest, http.MethodPost, "/api/github/app/manifest", `{"organization":"acme"}`)
	if start.Code != http.StatusOK {
		t.Fatalf("start organization manifest: status=%d body=%s", start.Code, start.Body.String())
	}
	var response struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode organization manifest: %v", err)
	}
	if response.Action != "https://github.com/organizations/acme/settings/apps/new" {
		t.Fatalf("organization manifest action=%q", response.Action)
	}
}

func registerGitHubTestApp(t *testing.T, s *Server) {
	t.Helper()
	start := invokeJSONHandler(t, s.handleGitHubAppManifest, http.MethodPost, "/api/github/app/manifest", `{}`)
	if start.Code != http.StatusOK {
		t.Fatalf("start manifest: status=%d body=%s", start.Code, start.Body.String())
	}
	var response struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode manifest state: %v", err)
	}
	callback := httptest.NewRecorder()
	s.handleGitHubAppManifestCallback(callback, httptest.NewRequest(http.MethodGet, "/api/github/app/manifest/callback?code=manifest-code&state="+response.State, nil))
	if callback.Code != http.StatusSeeOther {
		t.Fatalf("complete manifest: status=%d body=%s", callback.Code, callback.Body.String())
	}
}

func TestGitHubAppInstallationMakesSelectedRepositoriesAvailable(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	registerGitHubTestApp(t, s)

	install := invokeJSONHandler(t, s.handleGitHubAppInstall, http.MethodPost, "/api/github/app/install", `{}`)
	if install.Code != http.StatusOK {
		t.Fatalf("start installation: status=%d body=%s", install.Code, install.Body.String())
	}
	var installResponse struct {
		URL   string `json:"install_url"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(install.Body.Bytes(), &installResponse); err != nil {
		t.Fatalf("decode installation response: %v", err)
	}
	if !strings.Contains(installResponse.URL, "/apps/relay-test/installations/new?state=") || installResponse.State == "" {
		t.Fatalf("unexpected installation URL: %+v", installResponse)
	}

	setup := httptest.NewRecorder()
	setupRequest := httptest.NewRequest(http.MethodGet, "/api/github/app/setup?installation_id=7001&state="+installResponse.State, nil)
	s.handleGitHubAppSetup(setup, setupRequest)
	if setup.Code != http.StatusSeeOther {
		t.Fatalf("complete installation: status=%d body=%s", setup.Code, setup.Body.String())
	}

	repositories := invokeJSONHandler(t, s.handleGitHubRepositories, http.MethodGet, "/api/github/repos", "")
	if repositories.Code != http.StatusOK {
		t.Fatalf("list installation repositories: status=%d body=%s", repositories.Code, repositories.Body.String())
	}
	var items []githubRepository
	if err := json.Unmarshal(repositories.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode installation repositories: %v", err)
	}
	if len(items) != 1 || items[0].ID != 501 || items[0].InstallationID != 7001 || items[0].FullName != "acme/widget" {
		t.Fatalf("unexpected installation repositories: %+v", items)
	}
	fake.mu.Lock()
	tokenRequests := fake.tokenRequests
	fake.mu.Unlock()
	if tokenRequests != 1 {
		t.Fatalf("installation requested %d access tokens, want 1", tokenRequests)
	}
}

func installGitHubTestApp(t *testing.T, s *Server) {
	t.Helper()
	registerGitHubTestApp(t, s)
	install := invokeJSONHandler(t, s.handleGitHubAppInstall, http.MethodPost, "/api/github/app/install", `{}`)
	if install.Code != http.StatusOK {
		t.Fatalf("start installation: status=%d body=%s", install.Code, install.Body.String())
	}
	var installResponse struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(install.Body.Bytes(), &installResponse); err != nil {
		t.Fatalf("decode installation state: %v", err)
	}
	setup := httptest.NewRecorder()
	s.handleGitHubAppSetup(setup, httptest.NewRequest(http.MethodGet, "/api/github/app/setup?installation_id=7001&state="+installResponse.State, nil))
	if setup.Code != http.StatusSeeOther {
		t.Fatalf("complete installation: status=%d body=%s", setup.Code, setup.Body.String())
	}
}

func TestGitHubAppProjectLinkUsesInstallationWithoutRepositoryWebhook(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	installGitHubTestApp(t, s)

	response := invokeJSONHandler(t, s.handleGitHubProjects, http.MethodPost, "/api/github/projects", `{"app":"widget","repo_full_name":"acme/widget","production_branch":"main"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("link App repository: status=%d body=%s", response.Code, response.Body.String())
	}
	var project githubProjectView
	if err := json.Unmarshal(response.Body.Bytes(), &project); err != nil {
		t.Fatalf("decode linked App project: %v", err)
	}
	if project.AuthMode != "app" || project.InstallationID != 7001 || project.RepositoryID != 501 {
		t.Fatalf("unexpected App project: %+v", project.githubProjectRecord)
	}
	fake.mu.Lock()
	hookCount := fake.hooks
	fake.mu.Unlock()
	if hookCount != 0 {
		t.Fatalf("App project created %d repository webhooks", hookCount)
	}
}

func TestGitHubAppRenewsInstallationTokenBeforeExpiry(t *testing.T) {
	fake := newFakeGitHubServer(t)
	fake.shortTokens = true
	s := newGitHubWorkflowTestServer(t, fake)
	installGitHubTestApp(t, s)

	for range 2 {
		response := invokeJSONHandler(t, s.handleGitHubProjects, http.MethodPost, "/api/github/projects", `{"app":"widget","repo_full_name":"acme/widget","production_branch":"main"}`)
		if response.Code != http.StatusCreated {
			t.Fatalf("link App repository: status=%d body=%s", response.Code, response.Body.String())
		}
	}
	fake.mu.Lock()
	tokenRequests := fake.tokenRequests
	fake.mu.Unlock()
	if tokenRequests != 3 {
		t.Fatalf("installation token requests=%d, want setup plus two pre-expiry renewals", tokenRequests)
	}
}

func TestGitHubAppWebhookQueuesPreviewWithInstallationToken(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	installGitHubTestApp(t, s)
	response := invokeJSONHandler(t, s.handleGitHubProjects, http.MethodPost, "/api/github/projects", `{"app":"widget","repo_full_name":"acme/widget","production_branch":"main"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("link App repository: status=%d body=%s", response.Code, response.Body.String())
	}

	payload := `{"ref":"refs/heads/feature/app-auth","after":"ffffffffffffffffffffffffffffffffffffffff","installation":{"id":7001},"repository":{"id":501,"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main"},"head_commit":{"message":"Use installation auth"}}`
	mac := hmac.New(sha256.New, []byte("app-webhook-secret"))
	_, _ = mac.Write([]byte(payload))
	request := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", strings.NewReader(payload))
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Delivery", "app-delivery-preview")
	request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	recorder := httptest.NewRecorder()
	s.handleGithubWebhook(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("App push status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	job := <-s.queue
	if job.Req.Env != EnvPreview || job.Req.Branch != "feature/app-auth" || job.Req.GitToken != "github-installation-token" {
		t.Fatalf("unexpected App preview job: %+v", job.Req)
	}
}

func signedGitHubAppWebhookRequest(event string, deliveryID string, payload string) *http.Request {
	mac := hmac.New(sha256.New, []byte("app-webhook-secret"))
	_, _ = mac.Write([]byte(payload))
	request := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", strings.NewReader(payload))
	request.Header.Set("X-GitHub-Event", event)
	request.Header.Set("X-GitHub-Delivery", deliveryID)
	request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	return request
}

func TestGitHubAppInstallationDeletionRevokesRepositoryAccess(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	installGitHubTestApp(t, s)
	linked := invokeJSONHandler(t, s.handleGitHubProjects, http.MethodPost, "/api/github/projects", `{"app":"widget","repo_full_name":"acme/widget","production_branch":"main"}`)
	if linked.Code != http.StatusCreated {
		t.Fatalf("link App repository: status=%d body=%s", linked.Code, linked.Body.String())
	}

	deletedPayload := `{"action":"deleted","installation":{"id":7001,"app_id":991,"account":{"id":88,"login":"acme","type":"Organization"},"repository_selection":"selected"}}`
	deleted := httptest.NewRecorder()
	s.handleGithubWebhook(deleted, signedGitHubAppWebhookRequest("installation", "app-installation-deleted", deletedPayload))
	if deleted.Code != http.StatusAccepted {
		t.Fatalf("installation deletion: status=%d body=%s", deleted.Code, deleted.Body.String())
	}

	repositories := invokeJSONHandler(t, s.handleGitHubRepositories, http.MethodGet, "/api/github/repos", "")
	var items []githubRepository
	if repositories.Code != http.StatusOK || json.Unmarshal(repositories.Body.Bytes(), &items) != nil || len(items) != 0 {
		t.Fatalf("repositories remained after uninstall: status=%d body=%s", repositories.Code, repositories.Body.String())
	}

	pushPayload := `{"ref":"refs/heads/feature/after-delete","after":"1212121212121212121212121212121212121212","installation":{"id":7001},"repository":{"id":501,"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git"},"head_commit":{"message":"Must not deploy"}}`
	push := httptest.NewRecorder()
	s.handleGithubWebhook(push, signedGitHubAppWebhookRequest("push", "app-push-after-delete", pushPayload))
	if push.Code != http.StatusUnauthorized || len(s.queue) != 0 {
		t.Fatalf("deleted installation accepted push: status=%d queued=%d body=%s", push.Code, len(s.queue), push.Body.String())
	}
}

func TestGitHubAppRepositoryRemovalBlocksFutureDeployments(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	installGitHubTestApp(t, s)
	linked := invokeJSONHandler(t, s.handleGitHubProjects, http.MethodPost, "/api/github/projects", `{"app":"widget","repo_full_name":"acme/widget","production_branch":"main"}`)
	if linked.Code != http.StatusCreated {
		t.Fatalf("link App repository: status=%d body=%s", linked.Code, linked.Body.String())
	}

	removedPayload := `{"action":"removed","installation":{"id":7001,"app_id":991},"repositories_removed":[{"id":501,"full_name":"acme/widget"}]}`
	removed := httptest.NewRecorder()
	s.handleGithubWebhook(removed, signedGitHubAppWebhookRequest("installation_repositories", "app-repository-removed", removedPayload))
	if removed.Code != http.StatusAccepted {
		t.Fatalf("repository removal: status=%d body=%s", removed.Code, removed.Body.String())
	}

	pushPayload := `{"ref":"refs/heads/feature/removed","after":"5656565656565656565656565656565656565656","installation":{"id":7001},"repository":{"id":501,"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git"},"head_commit":{"message":"Must not deploy"}}`
	push := httptest.NewRecorder()
	s.handleGithubWebhook(push, signedGitHubAppWebhookRequest("push", "app-push-after-repository-removal", pushPayload))
	if push.Code != http.StatusUnauthorized || len(s.queue) != 0 {
		t.Fatalf("removed repository accepted push: status=%d queued=%d body=%s", push.Code, len(s.queue), push.Body.String())
	}
}

func TestGitHubAppCheckRunPublishesPreviewAndLogsLink(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	installGitHubTestApp(t, s)
	linked := invokeJSONHandler(t, s.handleGitHubProjects, http.MethodPost, "/api/github/projects", `{"app":"widget","repo_full_name":"acme/widget","production_branch":"main"}`)
	if linked.Code != http.StatusCreated {
		t.Fatalf("link App repository: status=%d body=%s", linked.Code, linked.Body.String())
	}

	payload := `{"ref":"refs/heads/feature/checks","after":"3434343434343434343434343434343434343434","installation":{"id":7001},"repository":{"id":501,"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget"},"head_commit":{"message":"Publish a Check Run"}}`
	push := httptest.NewRecorder()
	s.handleGithubWebhook(push, signedGitHubAppWebhookRequest("push", "app-check-run", payload))
	if push.Code != http.StatusAccepted {
		t.Fatalf("queue App preview: status=%d body=%s", push.Code, push.Body.String())
	}
	job := <-s.queue
	lane, err := s.getAppState("widget", EnvPreview, "feature/checks")
	if err != nil || lane == nil {
		t.Fatalf("load preview lane: %v", err)
	}
	lane.Mode = "traefik"
	lane.PublicHost = "widget-feature-checks.preview.example.com"
	if err := s.saveAppState(lane); err != nil {
		t.Fatalf("save preview route: %v", err)
	}
	ended := time.Now()
	if _, err := s.db.Exec(`UPDATE deploys SET status='success', ended_at=?, preview_url=? WHERE id=?`, ended.UnixMilli(), "https://widget-feature-checks.preview.example.com", job.ID); err != nil {
		t.Fatalf("complete preview deploy: %v", err)
	}
	s.emitGitHubDeployStatusByID(job.ID)

	fake.mu.Lock()
	checks := append([]map[string]any(nil), fake.checks...)
	statuses := append([]map[string]string(nil), fake.status...)
	fake.mu.Unlock()
	found := false
	for _, check := range checks {
		output, _ := check["output"].(map[string]any)
		summary, _ := output["summary"].(string)
		details, _ := check["details_url"].(string)
		if check["status"] == "completed" && check["conclusion"] == "success" && strings.Contains(summary, "https://widget-feature-checks.preview.example.com") && strings.Contains(details, "relay.example.com") {
			found = true
		}
	}
	if !found {
		t.Fatalf("successful preview Check Run missing route or logs link: %+v", checks)
	}
	if len(statuses) != 0 {
		t.Fatalf("App project also published legacy commit statuses: %+v", statuses)
	}
	projectResponse := invokeJSONHandler(t, s.handleGitHubProjects, http.MethodGet, "/api/github/projects?app=widget", "")
	var projects []githubProjectView
	if projectResponse.Code != http.StatusOK || json.Unmarshal(projectResponse.Body.Bytes(), &projects) != nil || len(projects) != 1 || len(projects[0].Previews) != 1 {
		t.Fatalf("read App workflow after Check Run: status=%d body=%s", projectResponse.Code, projectResponse.Body.String())
	}
	if projects[0].Previews[0].Status != "success" || projects[0].Previews[0].PreviewURL != "https://widget-feature-checks.preview.example.com" {
		t.Fatalf("Relay preview state did not follow Check Run: %+v", projects[0].Previews[0])
	}
}

func TestGitHubConnectionRequiresSecretKey(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	s.secretKey = nil

	response := invokeJSONHandler(t, s.handleGitHubConnection, http.MethodPost, "/api/github/connection", `{"token":"github-test-token"}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("connect without secret key: status=%d body=%s", response.Code, response.Body.String())
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM github_connections`).Scan(&count); err != nil {
		t.Fatalf("count GitHub connections: %v", err)
	}
	if count != 0 {
		t.Fatalf("stored %d GitHub connections without encryption", count)
	}
}

func TestConnectedGitHubWebhookRejectsInvalidSignature(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	_ = connectGitHubTestProject(t, s)
	payload := `{"ref":"refs/heads/feature/unsafe","after":"dddddddddddddddddddddddddddddddddddddddd","repository":{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git"},"head_commit":{"message":"Do not deploy"}}`
	request := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", strings.NewReader(payload))
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Delivery", "delivery-invalid-signature")
	request.Header.Set("X-Hub-Signature-256", "sha256=invalid")

	response := httptest.NewRecorder()
	s.handleGithubWebhook(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature: status=%d body=%s", response.Code, response.Body.String())
	}
	if len(s.queue) != 0 {
		t.Fatalf("invalid signature queued %d deploys", len(s.queue))
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM github_deliveries WHERE delivery_id='delivery-invalid-signature'`).Scan(&count); err != nil {
		t.Fatalf("count invalid delivery: %v", err)
	}
	if count != 0 {
		t.Fatalf("invalid signature was recorded as a trusted delivery")
	}
}

func TestConnectedGitHubPushQueuesPreviewAndProductionOnce(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	project := connectGitHubTestProject(t, s)

	previewPayload := `{"ref":"refs/heads/feature/invoices","after":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","repository":{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main"},"head_commit":{"message":"Add invoices"}}`
	request := signedGitHubWebhookRequest(t, project, "push", "delivery-preview", previewPayload)
	recorder := httptest.NewRecorder()
	s.handleGithubWebhook(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("preview push status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	previewJob := <-s.queue
	if previewJob.Req.Env != EnvPreview || previewJob.Req.Branch != "feature/invoices" || previewJob.Req.GitToken != "github-test-token" {
		t.Fatalf("unexpected preview job: %+v", previewJob.Req)
	}

	duplicate := httptest.NewRecorder()
	s.handleGithubWebhook(duplicate, signedGitHubWebhookRequest(t, project, "push", "delivery-preview", previewPayload))
	if duplicate.Code != http.StatusOK || len(s.queue) != 0 {
		t.Fatalf("duplicate delivery queued work: status=%d queued=%d", duplicate.Code, len(s.queue))
	}

	productionPayload := `{"ref":"refs/heads/main","after":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","repository":{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main"},"head_commit":{"message":"Merge pull request #12"}}`
	production := httptest.NewRecorder()
	s.handleGithubWebhook(production, signedGitHubWebhookRequest(t, project, "push", "delivery-production", productionPayload))
	if production.Code != http.StatusAccepted {
		t.Fatalf("production push status=%d body=%s", production.Code, production.Body.String())
	}
	productionJob := <-s.queue
	if productionJob.Req.Env != EnvProd || productionJob.Req.Branch != "main" {
		t.Fatalf("unexpected production job: %+v", productionJob.Req)
	}
}

func TestGitHubPullRequestReusesBranchDeployAndPublishesPreviewURL(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	project := connectGitHubTestProject(t, s)
	sha := "cccccccccccccccccccccccccccccccccccccccc"
	pushPayload := `{"ref":"refs/heads/feature/search","after":"` + sha + `","repository":{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git","html_url":"https://github.com/acme/widget","default_branch":"main"},"head_commit":{"message":"Add search"}}`
	push := httptest.NewRecorder()
	s.handleGithubWebhook(push, signedGitHubWebhookRequest(t, project, "push", "delivery-branch", pushPayload))
	job := <-s.queue

	prPayload := `{"action":"opened","number":17,"repository":{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git"},"pull_request":{"number":17,"title":"Add search","head":{"ref":"feature/search","sha":"` + sha + `","repo":{"full_name":"acme/widget","clone_url":"https://github.com/acme/widget.git"}},"base":{"ref":"main"}}}`
	pr := httptest.NewRecorder()
	s.handleGithubWebhook(pr, signedGitHubWebhookRequest(t, project, "pull_request", "delivery-pr", prPayload))
	if pr.Code != http.StatusAccepted || len(s.queue) != 0 {
		t.Fatalf("PR webhook duplicated branch deploy: status=%d queued=%d body=%s", pr.Code, len(s.queue), pr.Body.String())
	}

	lane, err := s.getAppState("widget", EnvPreview, "feature/search")
	if err != nil || lane == nil {
		t.Fatalf("load preview lane: %v", err)
	}
	lane.PublicHost = "widget-feature-search.preview.example.com"
	lane.Mode = "traefik"
	lane.CurrentImage = "relay/widget:preview"
	if err := s.saveAppState(lane); err != nil {
		t.Fatalf("save preview lane: %v", err)
	}
	ended := time.Now()
	if _, err := s.db.Exec(`UPDATE deploys SET status='success', ended_at=?, image_tag=?, preview_url=? WHERE id=?`, ended.UnixMilli(), "relay/widget:preview", "https://widget-feature-search.preview.example.com", job.ID); err != nil {
		t.Fatalf("complete preview deploy: %v", err)
	}
	s.emitGitHubDeployStatusByID(job.ID)

	fake.mu.Lock()
	statuses := append([]map[string]string(nil), fake.status...)
	fake.mu.Unlock()
	found := false
	for _, status := range statuses {
		if status["state"] == "success" && status["target_url"] == "https://widget-feature-search.preview.example.com" && status["context"] == "relay/preview" {
			found = true
		}
	}
	if !found {
		t.Fatalf("GitHub success status did not include HTTPS preview URL: %+v", statuses)
	}
	var prNumber int
	var previewURL string
	if err := s.db.QueryRow(`SELECT pr_number, preview_url FROM github_previews WHERE deploy_id=?`, job.ID).Scan(&prNumber, &previewURL); err != nil {
		t.Fatalf("read GitHub preview record: %v", err)
	}
	if prNumber != 17 || previewURL != "https://widget-feature-search.preview.example.com" {
		t.Fatalf("unexpected preview record: pr=%d url=%q", prNumber, previewURL)
	}
}

func TestGitHubProductionStatusStaysPendingUntilHealthWatchGraduates(t *testing.T) {
	fake := newFakeGitHubServer(t)
	s := newGitHubWorkflowTestServer(t, fake)
	_ = connectGitHubTestProject(t, s)

	deploy := &Deploy{
		ID:         "production-health-watch",
		App:        "widget",
		RepoURL:    "https://github.com/acme/widget.git",
		Branch:     "main",
		CommitSHA:  "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Env:        EnvProd,
		Status:     StatusSuccess,
		CreatedAt:  time.Now(),
		PreviewURL: "https://widget.example.com",
	}
	if err := s.saveDeployToDB(deploy, DeployRequest{App: deploy.App, RepoURL: deploy.RepoURL, Branch: deploy.Branch, CommitSHA: deploy.CommitSHA, Env: deploy.Env, Source: "github"}); err != nil {
		t.Fatalf("save production deploy: %v", err)
	}
	lane, err := s.getAppState("widget", EnvProd, "main")
	if err != nil || lane == nil {
		t.Fatalf("load production lane: %v", err)
	}
	lane.RolloutStatus = "monitoring"
	lane.RolloutDeployID = deploy.ID
	lane.ActiveSlot = "green"
	lane.StandbySlot = "blue"
	lane.TrafficSplitPercent = 10
	if err := s.saveAppState(lane); err != nil {
		t.Fatalf("save monitoring lane: %v", err)
	}

	s.emitGitHubDeployStatusByID(deploy.ID)
	fake.mu.Lock()
	monitoring := fake.status[len(fake.status)-1]
	fake.mu.Unlock()
	if monitoring["state"] != "pending" || monitoring["context"] != "relay/production" {
		t.Fatalf("monitoring rollout status=%+v, want pending production status", monitoring)
	}

	lane.RolloutStatus = "graduated"
	if err := s.saveAppState(lane); err != nil {
		t.Fatalf("save graduated lane: %v", err)
	}
	s.emitGitHubDeployStatusByID(deploy.ID)
	fake.mu.Lock()
	graduated := fake.status[len(fake.status)-1]
	fake.mu.Unlock()
	if graduated["state"] != "success" || graduated["target_url"] != "https://widget.example.com" {
		t.Fatalf("graduated rollout status=%+v, want successful production URL", graduated)
	}
}
