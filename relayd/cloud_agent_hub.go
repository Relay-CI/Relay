package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

type cloudController struct {
	agentEnabled bool
	hubEnabled   bool

	token         string
	baseURL       string
	wsURLOverride string
	serverName    string
	workspaceID   string
	localAPIBase  string

	httpClient *http.Client
	upgrader   websocket.Upgrader

	// pgPool is set when RELAY_CLOUD_PG_URL is configured.
	// Hub uses it instead of SQLite for token auth and server tracking.
	pgPool *pgxpool.Pool

	hubMu sync.Mutex
	conns map[string]*websocket.Conn
}

type cloudAgentRegisterRequest struct {
	Token        string `json:"token"`
	ServerName   string `json:"server_name,omitempty"`
	AgentVersion string `json:"agent_version,omitempty"`
}

type cloudAgentRegisterResponse struct {
	WorkspaceID   string `json:"workspace_id"`
	ServerID      string `json:"server_id"`
	ServerName    string `json:"server_name"`
	WebSocketURL  string `json:"websocket_url"`
	HeartbeatURL  string `json:"heartbeat_url"`
	DisconnectURL string `json:"disconnect_url"`
}

type cloudAgentHeartbeatRequest struct {
	Token    string `json:"token"`
	ServerID string `json:"server_id,omitempty"`
}

type cloudAgentDisconnectRequest struct {
	Token    string `json:"token"`
	ServerID string `json:"server_id,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type cloudSocketMessage struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Command   string          `json:"command,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

func (s *Server) initCloud(cfg relaydRunConfig) error {
	agentEnabled := cfg.CloudAgent || getenvBool("RELAY_CLOUD_AGENT", false)
	hubEnabled := cfg.CloudHub || getenvBool("RELAY_CLOUD_HUB", false)
	if !agentEnabled && !hubEnabled {
		return nil
	}

	hostname, _ := os.Hostname()
	if strings.TrimSpace(hostname) == "" {
		hostname = "relay-agent"
	}
	serverName := strings.TrimSpace(os.Getenv("RELAY_CLOUD_SERVER_NAME"))
	if serverName == "" {
		serverName = hostname
	}
	baseURL := strings.TrimSpace(os.Getenv("RELAY_CLOUD_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	workspaceID := strings.TrimSpace(os.Getenv("RELAY_CLOUD_WORKSPACE_ID"))
	if workspaceID == "" {
		workspaceID = "default"
	}
	localAPIBase, err := localAgentBaseURL(s.httpAddr)
	if err != nil {
		return err
	}

	controller := &cloudController{
		agentEnabled:  agentEnabled,
		hubEnabled:    hubEnabled,
		token:         strings.TrimSpace(os.Getenv("RELAY_CLOUD_TOKEN")),
		baseURL:       strings.TrimRight(baseURL, "/"),
		wsURLOverride: strings.TrimSpace(os.Getenv("RELAY_CLOUD_WS_URL")),
		serverName:    serverName,
		workspaceID:   workspaceID,
		localAPIBase:  localAPIBase,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
		conns: map[string]*websocket.Conn{},
	}
	if controller.agentEnabled && controller.token == "" {
		return fmt.Errorf("RELAY_CLOUD_TOKEN is required when cloud-agent mode is enabled")
	}
	s.cloud = controller
	if controller.hubEnabled {
		pgURL := strings.TrimSpace(os.Getenv("RELAY_CLOUD_PG_URL"))
		if pgURL != "" {
			pool, err := pgxpool.New(context.Background(), pgURL)
			if err != nil {
				return fmt.Errorf("cloud hub: pgxpool.New: %w", err)
			}
			controller.pgPool = pool
		} else {
			// SQLite fallback: seed token so hub can validate it locally.
			if err := s.seedCloudToken(); err != nil {
				return err
			}
		}
	}
	return nil
}

// hashAgentToken returns the hex-encoded SHA-256 of the raw agent token.
// Must match the hash stored by cloud-site's createAgentToken.
func hashAgentToken(token string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(h[:])
}

func localAgentBaseURL(addr string) (string, error) {
	raw := strings.TrimSpace(addr)
	if raw == "" {
		raw = ":8080"
	}
	if strings.HasPrefix(raw, ":") {
		return "http://127.0.0.1" + raw, nil
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return "", fmt.Errorf("invalid RELAY_ADDR %q: %w", raw, err)
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port), nil
}

func (s *Server) registerCloudHubRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/agent/register", s.handleAgentRegister)
	mux.HandleFunc("/api/agent/ws", s.handleAgentWS)
	mux.HandleFunc("/api/agent/heartbeat", s.handleAgentHeartbeat)
	mux.HandleFunc("/api/agent/disconnect", s.handleAgentDisconnect)
}

func (s *Server) runCloudAgentLoop() {
	if s.cloud == nil || !s.cloud.agentEnabled {
		return
	}
	for {
		resp, err := s.cloudRegister()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cloud-agent register failed: %v\n", err)
			time.Sleep(3 * time.Second)
			continue
		}
		if err := s.cloudRunSocketSession(resp); err != nil {
			fmt.Fprintf(os.Stderr, "cloud-agent websocket disconnected: %v\n", err)
		}
		_ = s.cloudPostDisconnect(resp, "socket closed")
		time.Sleep(3 * time.Second)
	}
}

func (s *Server) cloudRegister() (*cloudAgentRegisterResponse, error) {
	reqBody := cloudAgentRegisterRequest{
		Token:        s.cloud.token,
		ServerName:   s.cloud.serverName,
		AgentVersion: relaydVersionLine(),
	}
	payload, _ := json.Marshal(reqBody)
	resBody, status, err := s.cloudJSONRequest(http.MethodPost, s.cloud.baseURL+"/api/agent/register", payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("register returned %d: %s", status, strings.TrimSpace(string(resBody)))
	}
	var resp cloudAgentRegisterResponse
	if err := json.Unmarshal(resBody, &resp); err != nil {
		return nil, fmt.Errorf("decode register response: %w", err)
	}
	if strings.TrimSpace(resp.WebSocketURL) == "" {
		return nil, fmt.Errorf("register response missing websocket_url")
	}
	return &resp, nil
}

func (s *Server) cloudRunSocketSession(reg *cloudAgentRegisterResponse) error {
	wsURL, err := appendTokenQuery(reg.WebSocketURL, s.cloud.token)
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+s.cloud.token)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	var writeMu sync.Mutex
	send := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v)
	}

	_ = send(map[string]any{
		"type":         "agent_connected",
		"server_id":    reg.ServerID,
		"workspace_id": reg.WorkspaceID,
		"server_name":  reg.ServerName,
		"timestamp":    time.Now().UnixMilli(),
	})

	readErr := make(chan error, 1)
	go func() {
		for {
			var msg cloudSocketMessage
			if err := conn.ReadJSON(&msg); err != nil {
				readErr <- err
				return
			}
			if strings.EqualFold(strings.TrimSpace(msg.Type), "command") {
				go s.handleCloudAgentCommand(send, reg, msg)
			}
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-readErr:
			return err
		case <-ticker.C:
			_ = send(map[string]any{
				"type":      "heartbeat",
				"server_id": reg.ServerID,
				"timestamp": time.Now().UnixMilli(),
			})
			_ = s.cloudPostHeartbeat(reg)
		}
	}
}

func (s *Server) handleCloudAgentCommand(send func(any) error, reg *cloudAgentRegisterResponse, msg cloudSocketMessage) {
	cmd := strings.ToLower(strings.TrimSpace(msg.Command))
	if msg.ID == "" {
		msg.ID = newID()
	}
	// This runs in its own goroutine driven by remote command input. An
	// unrecovered panic in any command handler would crash the entire relayd
	// process, so contain it here and report the failure back to the caller.
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("cloud-agent: recovered from panic handling command %q: %v\n", cmd, r)
			_ = send(map[string]any{
				"type":      "command_result",
				"id":        msg.ID,
				"command":   cmd,
				"status":    "error",
				"error":     fmt.Sprintf("internal panic: %v", r),
				"timestamp": time.Now().UnixMilli(),
			})
		}
	}()
	sendResult := func(status string, result any, err error) {
		out := map[string]any{
			"type":      "command_result",
			"id":        msg.ID,
			"command":   cmd,
			"status":    status,
			"timestamp": time.Now().UnixMilli(),
		}
		if err != nil {
			out["error"] = err.Error()
		}
		if result != nil {
			out["result"] = result
		}
		_ = send(out)
	}

	switch cmd {
	case "deploy":
		body, statusCode, err := s.invokeLocalAgentAPI(http.MethodPost, "/api/deploys", msg.Payload)
		if err != nil {
			sendResult("error", nil, err)
			return
		}
		if statusCode < 200 || statusCode >= 300 {
			sendResult("error", map[string]any{"status_code": statusCode, "body": string(body)}, fmt.Errorf("deploy failed with status %d", statusCode))
			return
		}
		sendResult("ok", map[string]any{"status_code": statusCode, "body": string(body)}, nil)
	case "restart":
		body, statusCode, err := s.invokeLocalAgentAPI(http.MethodPost, "/api/apps/restart", msg.Payload)
		if err != nil {
			sendResult("error", nil, err)
			return
		}
		if statusCode < 200 || statusCode >= 300 {
			sendResult("error", map[string]any{"status_code": statusCode, "body": string(body)}, fmt.Errorf("restart failed with status %d", statusCode))
			return
		}
		sendResult("ok", map[string]any{"status_code": statusCode, "body": string(body)}, nil)
	case "stop":
		body, statusCode, err := s.invokeLocalAgentAPI(http.MethodPost, "/api/apps/stop", msg.Payload)
		if err != nil {
			sendResult("error", nil, err)
			return
		}
		if statusCode < 200 || statusCode >= 300 {
			sendResult("error", map[string]any{"status_code": statusCode, "body": string(body)}, fmt.Errorf("stop failed with status %d", statusCode))
			return
		}
		sendResult("ok", map[string]any{"status_code": statusCode, "body": string(body)}, nil)
	case "stream-logs":
		err := s.streamDeployLogsToCloud(send, reg, msg.ID, msg.Payload)
		if err != nil {
			sendResult("error", nil, err)
			return
		}
		sendResult("ok", map[string]any{"stream": "completed"}, nil)
	default:
		sendResult("error", nil, fmt.Errorf("unsupported command %q", cmd))
	}
}

func (s *Server) streamDeployLogsToCloud(send func(any) error, reg *cloudAgentRegisterResponse, commandID string, payload json.RawMessage) error {
	var req struct {
		DeployID string `json:"deploy_id"`
		From     int64  `json:"from,omitempty"`
		MaxLines int    `json:"max_lines,omitempty"`
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &req); err != nil {
			return fmt.Errorf("invalid stream-logs payload: %w", err)
		}
	}
	req.DeployID = strings.TrimSpace(req.DeployID)
	if req.DeployID == "" {
		return fmt.Errorf("deploy_id is required for stream-logs")
	}
	if req.From < 0 {
		req.From = 0
	}
	if req.MaxLines <= 0 {
		req.MaxLines = 2000
	}
	if req.MaxLines > 10000 {
		req.MaxLines = 10000
	}

	deploy, err := s.getDeployFromDB(req.DeployID)
	if err != nil || deploy == nil {
		return fmt.Errorf("deploy %s not found", req.DeployID)
	}
	f, err := os.Open(deploy.LogPath)
	if err != nil {
		return fmt.Errorf("open deploy log: %w", err)
	}
	defer f.Close()
	if _, err := f.Seek(req.From, io.SeekStart); err != nil {
		return fmt.Errorf("seek deploy log: %w", err)
	}

	scanner := NewLineScanner(f)
	lines := 0
	offset := req.From
	for scanner.Scan() {
		line := scanner.Text()
		offset += int64(len(line) + 1)
		_ = send(map[string]any{
			"type":       "log",
			"command_id": commandID,
			"server_id":  reg.ServerID,
			"deploy_id":  req.DeployID,
			"offset":     offset,
			"line":       line,
			"timestamp":  time.Now().UnixMilli(),
		})
		lines++
		if lines >= req.MaxLines {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read deploy log: %w", err)
	}
	return nil
}

func (s *Server) invokeLocalAgentAPI(method string, path string, payload []byte) ([]byte, int, error) {
	endpoint := strings.TrimRight(s.cloud.localAPIBase, "/") + path
	var body io.Reader
	if len(payload) > 0 {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.cloud.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	resBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resBody, resp.StatusCode, nil
}

func (s *Server) cloudPostHeartbeat(reg *cloudAgentRegisterResponse) error {
	req := cloudAgentHeartbeatRequest{Token: s.cloud.token, ServerID: reg.ServerID}
	payload, _ := json.Marshal(req)
	endpoint := strings.TrimSpace(reg.HeartbeatURL)
	if endpoint == "" {
		endpoint = s.cloud.baseURL + "/api/agent/heartbeat"
	}
	_, status, err := s.cloudJSONRequest(http.MethodPost, endpoint, payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("heartbeat returned %d", status)
	}
	return nil
}

func (s *Server) cloudPostDisconnect(reg *cloudAgentRegisterResponse, reason string) error {
	req := cloudAgentDisconnectRequest{Token: s.cloud.token, ServerID: reg.ServerID, Reason: reason}
	payload, _ := json.Marshal(req)
	endpoint := strings.TrimSpace(reg.DisconnectURL)
	if endpoint == "" {
		endpoint = s.cloud.baseURL + "/api/agent/disconnect"
	}
	_, _, err := s.cloudJSONRequest(http.MethodPost, endpoint, payload)
	return err
}

func (s *Server) cloudJSONRequest(method string, endpoint string, payload []byte) ([]byte, int, error) {
	var body io.Reader
	if len(payload) > 0 {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, 0, err
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.cloud.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	resBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resBody, resp.StatusCode, nil
}

func appendTokenQuery(rawURL string, token string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	q := u.Query()
	if strings.TrimSpace(q.Get("token")) == "" {
		q.Set("token", token)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *Server) seedCloudToken() error {
	if s.cloud == nil || strings.TrimSpace(s.cloud.token) == "" {
		return nil
	}
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`INSERT INTO cloud_agent_tokens (token, workspace_id, server_name, created_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(token) DO UPDATE SET workspace_id=excluded.workspace_id, server_name=excluded.server_name`,
		s.cloud.token, s.cloud.workspaceID, s.cloud.serverName, now,
	)
	return err
}

func (s *Server) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if s.cloud == nil || !s.cloud.hubEnabled {
		httpError(w, 404, "cloud hub mode disabled")
		return
	}
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req cloudAgentRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	token := strings.TrimSpace(req.Token)
	workspaceID, defaultName, ok := s.resolveCloudAssignmentByToken(token)
	if !ok {
		httpError(w, 403, "invalid agent token")
		return
	}
	serverName := strings.TrimSpace(req.ServerName)
	if serverName == "" {
		serverName = defaultName
	}
	if serverName == "" {
		serverName = "relay-agent"
	}
	serverID, err := s.upsertConnectedServer(token, workspaceID, serverName, strings.TrimSpace(req.AgentVersion), "online", "")
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	wsURL := s.cloud.wsURLOverride
	if strings.TrimSpace(wsURL) == "" {
		wsURL = websocketURLForRequest(r) + "/api/agent/ws"
	} else if !strings.Contains(strings.TrimSpace(wsURL), "/api/agent/ws") {
		wsURL = strings.TrimRight(wsURL, "/") + "/api/agent/ws"
	}
	base := externalBaseURLForRequest(r)
	writeJSON(w, 200, cloudAgentRegisterResponse{
		WorkspaceID:   workspaceID,
		ServerID:      serverID,
		ServerName:    serverName,
		WebSocketURL:  wsURL,
		HeartbeatURL:  strings.TrimRight(base, "/") + "/api/agent/heartbeat",
		DisconnectURL: strings.TrimRight(base, "/") + "/api/agent/disconnect",
	})
}

func (s *Server) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	if s.cloud == nil || !s.cloud.hubEnabled {
		httpError(w, 404, "cloud hub mode disabled")
		return
	}
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	token, _ := s.requestToken(r)
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	workspaceID, defaultName, ok := s.resolveCloudAssignmentByToken(token)
	if !ok {
		httpError(w, 403, "invalid agent token")
		return
	}
	conn, err := s.cloud.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	sessionID := newID()
	serverID, err := s.upsertConnectedServer(token, workspaceID, defaultName, "", "online", sessionID)
	if err != nil {
		_ = conn.Close()
		return
	}
	if err := s.attachHubConn(token, conn); err != nil {
		_ = conn.Close()
		return
	}
	defer func() {
		s.detachHubConn(token, conn)
		_ = s.markConnectedServerOffline(token, "socket closed")
		_ = conn.Close()
	}()

	_ = conn.WriteJSON(map[string]any{
		"type":         "connected",
		"server_id":    serverID,
		"workspace_id": workspaceID,
		"session_id":   sessionID,
		"timestamp":    time.Now().UnixMilli(),
	})

	for {
		var msg cloudSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if strings.EqualFold(strings.TrimSpace(msg.Type), "heartbeat") {
			_ = s.markConnectedServerHeartbeat(token)
		}
	}
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if s.cloud == nil || !s.cloud.hubEnabled {
		httpError(w, 404, "cloud hub mode disabled")
		return
	}
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req cloudAgentHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		token, _ = s.requestToken(r)
	}
	if _, _, ok := s.resolveCloudAssignmentByToken(token); !ok {
		httpError(w, 403, "invalid agent token")
		return
	}
	if err := s.markConnectedServerHeartbeat(token); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"status": "ok", "timestamp": time.Now().UnixMilli()})
}

func (s *Server) handleAgentDisconnect(w http.ResponseWriter, r *http.Request) {
	if s.cloud == nil || !s.cloud.hubEnabled {
		httpError(w, 404, "cloud hub mode disabled")
		return
	}
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	var req cloudAgentDisconnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		token, _ = s.requestToken(r)
	}
	if _, _, ok := s.resolveCloudAssignmentByToken(token); !ok {
		httpError(w, 403, "invalid agent token")
		return
	}
	s.closeHubConn(token)
	if err := s.markConnectedServerOffline(token, req.Reason); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"status": "offline"})
}

func (s *Server) resolveCloudAssignmentByToken(token string) (string, string, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", false
	}
	// PG path: hash token, query agent_tokens.
	if s.cloud.pgPool != nil {
		h := hashAgentToken(token)
		var workspaceID, label string
		err := s.cloud.pgPool.QueryRow(
			context.Background(),
			`SELECT workspace_id::text, COALESCE(label,'') FROM agent_tokens
			 WHERE token_hash = $1 AND revoked_at IS NULL LIMIT 1`,
			h,
		).Scan(&workspaceID, &label)
		if err != nil {
			return "", "", false
		}
		// Touch last_used_at asynchronously — non-critical.
		go func() {
			_, _ = s.cloud.pgPool.Exec(
				context.Background(),
				`UPDATE agent_tokens SET last_used_at = NOW() WHERE token_hash = $1`,
				h,
			)
		}()
		if strings.TrimSpace(workspaceID) == "" {
			workspaceID = "default"
		}
		return workspaceID, strings.TrimSpace(label), true
	}
	// SQLite fallback.
	if strings.TrimSpace(s.cloud.token) != "" && subtle.ConstantTimeCompare([]byte(token), []byte(strings.TrimSpace(s.cloud.token))) == 1 {
		return s.cloud.workspaceID, s.cloud.serverName, true
	}
	var workspaceID string
	var serverName string
	err := s.db.QueryRow(`SELECT workspace_id, server_name FROM cloud_agent_tokens WHERE token=?`, token).Scan(&workspaceID, &serverName)
	if err != nil {
		return "", "", false
	}
	if strings.TrimSpace(workspaceID) == "" {
		workspaceID = "default"
	}
	return workspaceID, strings.TrimSpace(serverName), true
}

func (s *Server) upsertConnectedServer(token string, workspaceID string, serverName string, agentVersion string, status string, wsSessionID string) (string, error) {
	token = strings.TrimSpace(token)
	workspaceID = strings.TrimSpace(workspaceID)
	serverName = strings.TrimSpace(serverName)
	agentVersion = strings.TrimSpace(agentVersion)
	status = strings.TrimSpace(status)
	if workspaceID == "" {
		workspaceID = "default"
	}
	if serverName == "" {
		serverName = "relay-agent"
	}
	if status == "" {
		status = "online"
	}

	// PG path.
	if s.cloud.pgPool != nil {
		h := hashAgentToken(token)
		var id string
		err := s.cloud.pgPool.QueryRow(
			context.Background(),
			`INSERT INTO connected_servers
			   (workspace_id, name, status, relayd_version, last_heartbeat_at, token_hash)
			 VALUES ($1::uuid, $2, $3, NULLIF($4,''), NOW(), $5)
			 ON CONFLICT (token_hash) DO UPDATE SET
			   workspace_id      = EXCLUDED.workspace_id,
			   name             = EXCLUDED.name,
			   status           = EXCLUDED.status,
			   relayd_version   = COALESCE(NULLIF(EXCLUDED.relayd_version,''), connected_servers.relayd_version),
			   last_heartbeat_at = NOW(),
			   updated_at       = NOW()
			 RETURNING id::text`,
			workspaceID, serverName, status, agentVersion, h,
		).Scan(&id)
		if err != nil {
			return "", fmt.Errorf("upsertConnectedServer PG: %w", err)
		}
		return id, nil
	}

	// SQLite fallback.
	now := time.Now().UnixMilli()
	id := ""
	err := s.db.QueryRow(`SELECT id FROM connected_servers WHERE token=?`, token).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if id == "" {
		id = newID()
		_, err = s.db.Exec(
			`INSERT INTO connected_servers (id, token, workspace_id, server_name, status, ws_session_id, agent_version, last_heartbeat_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, token, workspaceID, serverName, status, wsSessionID, agentVersion, now, now, now,
		)
		if err != nil {
			return "", err
		}
		return id, nil
	}
	_, err = s.db.Exec(
		`UPDATE connected_servers
		 SET workspace_id=?, server_name=?, status=?, ws_session_id=COALESCE(NULLIF(?,''), ws_session_id),
		     agent_version=COALESCE(NULLIF(?,''), agent_version),
		     last_heartbeat_at=?, updated_at=?
		 WHERE token=?`,
		workspaceID, serverName, status, wsSessionID, agentVersion, now, now, token,
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Server) markConnectedServerHeartbeat(token string) error {
	if s.cloud.pgPool != nil {
		h := hashAgentToken(token)
		_, err := s.cloud.pgPool.Exec(
			context.Background(),
			`UPDATE connected_servers
			 SET status='online', last_heartbeat_at=NOW(), updated_at=NOW()
			 WHERE token_hash=$1`,
			h,
		)
		return err
	}
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`UPDATE connected_servers
		 SET status='online', last_heartbeat_at=?, updated_at=?
		 WHERE token=?`,
		now, now, strings.TrimSpace(token),
	)
	return err
}

func (s *Server) markConnectedServerOffline(token string, reason string) error {
	_ = strings.TrimSpace(reason)
	if s.cloud.pgPool != nil {
		h := hashAgentToken(token)
		_, err := s.cloud.pgPool.Exec(
			context.Background(),
			`UPDATE connected_servers
			 SET status='offline', updated_at=NOW()
			 WHERE token_hash=$1`,
			h,
		)
		return err
	}
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`UPDATE connected_servers
		 SET status='offline', ws_session_id='', last_disconnect_at=?, updated_at=?
		 WHERE token=?`,
		now, now, strings.TrimSpace(token),
	)
	return err
}

func (s *Server) attachHubConn(token string, conn *websocket.Conn) error {
	s.cloud.hubMu.Lock()
	defer s.cloud.hubMu.Unlock()
	if existing, ok := s.cloud.conns[token]; ok && existing != nil {
		_ = existing.Close()
	}
	s.cloud.conns[token] = conn
	return nil
}

func (s *Server) detachHubConn(token string, conn *websocket.Conn) {
	s.cloud.hubMu.Lock()
	defer s.cloud.hubMu.Unlock()
	current, ok := s.cloud.conns[token]
	if ok && current == conn {
		delete(s.cloud.conns, token)
	}
}

func (s *Server) closeHubConn(token string) {
	s.cloud.hubMu.Lock()
	defer s.cloud.hubMu.Unlock()
	if conn, ok := s.cloud.conns[token]; ok && conn != nil {
		_ = conn.Close()
		delete(s.cloud.conns, token)
	}
}

func websocketURLForRequest(r *http.Request) string {
	scheme := "ws"
	if isHTTPSRequest(r) {
		scheme = "wss"
	}
	host := strings.TrimSpace(r.Host)
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		host = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return scheme + "://" + host
}

func externalBaseURLForRequest(r *http.Request) string {
	scheme := "http"
	if isHTTPSRequest(r) {
		scheme = "https"
	}
	host := strings.TrimSpace(r.Host)
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		host = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return scheme + "://" + host
}

// NewLineScanner returns a scanner tuned for long deploy log lines.
func NewLineScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return scanner
}
