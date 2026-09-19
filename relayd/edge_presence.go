package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const edgePresencePath = "/__relay/presence"
const edgePresenceScriptPath = "/__relay/presence.js"

// Reserve a short durable presence lease so requests within the lease do not
// write SQLite again. The reservation can delay expiry by at most this amount
// after a crash; it never expires an active visitor early.
const edgePresenceWriteLease = 10 * time.Second

func (s *Server) edgeSessionToken(app string, env DeployEnv, branch string) (string, error) {
	s.edgeTokenMu.Lock()
	defer s.edgeTokenMu.Unlock()
	// The key is kept outside the app-state API and survives a restart. A
	// container's internal proxy needs a lane-bound token to route requests.
	keyPath := filepath.Join(s.dataDir, "edge-session.key")
	key := s.edgeTokenKey
	var err error
	if len(key) == 0 {
		key, err = os.ReadFile(keyPath)
	}
	if os.IsNotExist(err) {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return "", err
		}
		if err = os.MkdirAll(s.dataDir, 0700); err != nil {
			return "", err
		}
		if err = os.WriteFile(keyPath, key, 0600); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	if len(key) != 32 {
		return "", fmt.Errorf("invalid edge session key")
	}
	s.edgeTokenKey = key
	mac := hmac.New(sha256.New, key)
	_, _ = io.WriteString(mac, app+"\x00"+string(env)+"\x00"+branch)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func edgeSessionProxyURL(port int, host string, app string, env DeployEnv, branch string) string {
	// Lane identity travels in internal proxy headers rather than proxy_pass
	// query arguments. Some reverse-proxy configurations rewrite/drop the
	// latter, which turns every asset request into "invalid lane" before it
	// reaches the app.
	return fmt.Sprintf("http://%s:%d/api/edge/session-proxy", host, port)
}

func edgePresenceTTL() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAY_SESSION_IDLE_SECONDS")))
	if err != nil || seconds < 15 {
		seconds = 180
	}
	return time.Duration(seconds) * time.Second
}

func edgeMaxDrain() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAY_SESSION_MAX_DRAIN_SECONDS")))
	if err != nil || seconds < 30 {
		seconds = 3600
	}
	return time.Duration(seconds) * time.Second
}

func (s *Server) ensureEdgePresenceSchema() error {
	s.edgePresenceMu.Lock()
	defer s.edgePresenceMu.Unlock()
	if s.edgePresenceReady {
		return nil
	}
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS edge_presence (
		id TEXT PRIMARY KEY, app TEXT NOT NULL, env TEXT NOT NULL, branch TEXT NOT NULL,
		slot TEXT NOT NULL, last_seen_at INTEGER NOT NULL)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS edge_presence_lane_slot ON edge_presence(app,env,branch,slot,last_seen_at)`)
	if err == nil {
		s.edgePresenceReady = true
	}
	return err
}

func newEdgePresenceID() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type edgePresenceSession struct {
	ID       string
	Slot     string
	LastSeen int64
}

func (s *Server) edgePresenceSession(app string, env DeployEnv, branch string, id string) (*edgePresenceSession, error) {
	if id == "" || len(id) > 64 {
		return nil, nil
	}
	var rec edgePresenceSession
	err := s.db.QueryRow(`SELECT id,slot,last_seen_at FROM edge_presence WHERE id=? AND app=? AND env=? AND branch=?`, id, app, string(env), branch).Scan(&rec.ID, &rec.Slot, &rec.LastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Server) touchEdgePresence(app string, env DeployEnv, branch, id, slot string, now int64) error {
	_, err := s.db.Exec(`INSERT INTO edge_presence(id,app,env,branch,slot,last_seen_at) VALUES(?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET last_seen_at=excluded.last_seen_at
		WHERE edge_presence.last_seen_at<=?`, id, app, string(env), branch, slot, now+edgePresenceWriteLease.Milliseconds(), now)
	return err
}

func (s *Server) activeOldSessions(st *AppState, now time.Time) (int, error) {
	if err := s.ensureEdgePresenceSchema(); err != nil {
		return 0, err
	}
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM edge_presence WHERE app=? AND env=? AND branch=? AND slot=? AND last_seen_at>=?`,
		st.App, string(st.Env), st.Branch, st.StandbySlot, now.Add(-edgePresenceTTL()).UnixMilli()).Scan(&count)
	return count, err
}

const edgePresenceScript = `(function(){
  if (window.__relayPresenceStarted) return;
  window.__relayPresenceStarted = true;
  var tab = (window.crypto && crypto.randomUUID) ? crypto.randomUUID() : String(Date.now()) + Math.random();
  var endpoint = '/__relay/presence';
  function beat() {
    fetch(endpoint, {method:'POST',credentials:'same-origin',cache:'no-store',
      headers:{'Content-Type':'text/plain'},body:tab,keepalive:true}).catch(function(){});
  }
  beat();
  setInterval(beat, 15000);
  window.addEventListener('pageshow', beat);
  window.addEventListener('online', beat);
  document.addEventListener('visibilitychange', function(){ if (document.visibilityState === 'visible') beat(); });
})();`

func injectEdgePresenceScript(body []byte) []byte {
	script := []byte(`<script src="` + edgePresenceScriptPath + `" defer></script>`)
	lower := bytes.ToLower(body)
	if start := bytes.Index(lower, []byte("<head")); start >= 0 {
		if end := bytes.IndexByte(lower[start:], '>'); end >= 0 {
			at := start + end + 1
			out := make([]byte, 0, len(body)+len(script))
			out = append(out, body[:at]...)
			out = append(out, script...)
			return append(out, body[at:]...)
		}
	}
	return append(script, body...)
}

func (s *Server) handleEdgeSessionProxy(w http.ResponseWriter, r *http.Request) {
	app := firstNonEmpty(r.Header.Get("X-Relay-Lane-App"), r.URL.Query().Get("app"))
	env := normalizeDeployEnv(firstNonEmpty(r.Header.Get("X-Relay-Lane-Env"), r.URL.Query().Get("env")))
	branch := firstNonEmpty(r.Header.Get("X-Relay-Lane-Branch"), r.URL.Query().Get("branch"))
	if !validDeployTarget(app, env, branch) {
		http.Error(w, "invalid lane", http.StatusBadRequest)
		return
	}
	token, err := s.edgeSessionToken(app, env, branch)
	if err != nil || !hmac.Equal([]byte(token), []byte(r.Header.Get("X-Relay-Edge-Token"))) {
		http.Error(w, "unauthorized edge", http.StatusUnauthorized)
		return
	}
	if err := s.ensureEdgePresenceSchema(); err != nil {
		http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
		return
	}
	selectionLock := s.edgeProxyLock(app, env, branch)
	selectionLock.Lock()
	locked := true
	defer func() {
		if locked {
			selectionLock.Unlock()
		}
	}()
	st, err := s.getAppState(app, env, branch)
	if err != nil || st == nil || st.Stopped || st.TrafficMode != "session" {
		http.Error(w, "session route unavailable", http.StatusServiceUnavailable)
		return
	}
	original := r.Header.Get("X-Relay-Original-Uri")
	uri, err := url.ParseRequestURI(original)
	if err != nil || !strings.HasPrefix(uri.Path, "/") {
		http.Error(w, "invalid request URI", http.StatusBadRequest)
		return
	}
	if uri.Path == edgePresenceScriptPath && r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, edgePresenceScript)
		return
	}
	now := time.Now()
	cookieName := edgeCookieName(app, env, branch)
	id := ""
	if cookie, cookieErr := r.Cookie(cookieName); cookieErr == nil {
		id = cookie.Value
	}
	rec, err := s.edgePresenceSession(app, env, branch, id)
	if err != nil {
		http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
		return
	}
	active := normalizeActiveSlot(st.ActiveSlot)
	standby := normalizeActiveSlot(st.StandbySlot)
	slot := active
	if rec != nil && now.UnixMilli()-rec.LastSeen <= edgePresenceTTL().Milliseconds() {
		if rec.Slot == active || (rec.Slot == standby && st.RolloutStatus != "retiring" && now.UnixMilli() < st.DrainUntil) {
			slot = rec.Slot
		}
	}
	if active == "" {
		http.Error(w, "no active slot", http.StatusServiceUnavailable)
		return
	}
	if uri.Path == edgePresencePath {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if rec == nil || rec.Slot != slot {
			w.WriteHeader(http.StatusGone)
			return
		}
		if rec.LastSeen <= now.UnixMilli() {
			if err := s.touchEdgePresence(app, env, branch, rec.ID, slot, now.UnixMilli()); err != nil {
				http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		s.setEdgePresenceCookie(w, r, cookieName, rec.ID)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if rec == nil || rec.Slot != slot || now.UnixMilli()-rec.LastSeen > edgePresenceTTL().Milliseconds() {
		id, err = newEdgePresenceID()
		if err != nil {
			http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	if rec == nil || rec.ID != id || rec.LastSeen <= now.UnixMilli() {
		if err := s.touchEdgePresence(app, env, branch, id, slot, now.UnixMilli()); err != nil {
			http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	s.setEdgePresenceCookie(w, r, cookieName, id)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Relay-Target", slot)
	w.Header().Set("X-Relay-Traffic-Mode", "session")
	runtime := s.runtimeForEngine(st.Engine)
	name := appSlotContainerName(app, env, branch, slot)
	upstream := ""
	if firstNonEmptyEngine(st.Engine) == EngineStation {
		upstream = stationSlotUpstream(runtime, name, st.ServicePort)
	} else if port := runtime.PublishedPort(name, st.ServicePort); port > 0 {
		upstream = "127.0.0.1:" + strconv.Itoa(port)
	}
	if upstream == "" {
		http.Error(w, "slot unavailable", http.StatusBadGateway)
		return
	}
	s.edgeRequestStart(app, env, branch, slot)
	defer s.edgeRequestDone(app, env, branch, slot)
	selectionLock.Unlock()
	locked = false
	// A long upload or streaming response is activity too. Keep its session
	// fresh even if the browser's background tab cannot run a timer.
	requestDone := make(chan struct{})
	defer close(requestDone)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-requestDone:
				return
			case <-ticker.C:
				_ = s.touchEdgePresence(app, env, branch, id, slot, time.Now().UnixMilli())
			}
		}
	}()
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = upstream
			req.URL.Path = uri.Path
			req.URL.RawPath = uri.RawPath
			req.URL.RawQuery = uri.RawQuery
			req.Host = r.Header.Get("X-Forwarded-Host")
			if req.Host == "" {
				req.Host = r.Host
			}
			req.Header.Del("X-Relay-Edge-Token")
			req.Header.Del("X-Relay-Original-Uri")
			req.Header.Set("Accept-Encoding", "identity")
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Set("Cache-Control", "private, no-store")
			if resp.Request.Method != http.MethodGet || resp.StatusCode != http.StatusOK ||
				!strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html") ||
				resp.Header.Get("Content-Encoding") != "" || resp.Header.Get("Content-Range") != "" {
				return nil
			}
			// Bound buffering. Large or streaming HTML is left unmodified; its
			// regular requests still refresh presence, and apps can include the
			// documented heartbeat script themselves.
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, (4<<20)+1))
			if readErr != nil {
				return readErr
			}
			if len(body) <= 4<<20 {
				body = injectEdgePresenceScript(body)
				resp.Header.Del("Content-Length")
				resp.ContentLength = int64(len(body))
				_ = resp.Body.Close()
				resp.Body = io.NopCloser(bytes.NewReader(body))
			} else {
				resp.Body = &edgePrefixedBody{Reader: io.MultiReader(bytes.NewReader(body), resp.Body), Closer: resp.Body}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "slot unavailable", http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(w, r)
}

type edgePrefixedBody struct {
	io.Reader
	io.Closer
}

func (s *Server) setEdgePresenceCookie(w http.ResponseWriter, r *http.Request, name, id string) {
	secure := strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	maxAge := edgeMaxDrain()
	if idleGrace := 2 * edgePresenceTTL(); idleGrace > maxAge {
		maxAge = idleGrace
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: id, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: int(maxAge.Seconds())})
}

type edgeRequestCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

var edgeRequests edgeRequestCounter

var edgeDrainWatches = struct {
	mu   sync.Mutex
	keys map[string]bool
}{keys: map[string]bool{}}

func (s *Server) startSessionDrain(app string, env DeployEnv, branch, active, old string) {
	key := edgeRequestKey(app, env, branch, old)
	edgeDrainWatches.mu.Lock()
	if edgeDrainWatches.keys[key] {
		edgeDrainWatches.mu.Unlock()
		return
	}
	edgeDrainWatches.keys[key] = true
	edgeDrainWatches.mu.Unlock()
	go func() {
		defer func() {
			edgeDrainWatches.mu.Lock()
			delete(edgeDrainWatches.keys, key)
			edgeDrainWatches.mu.Unlock()
		}()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			if s.sessionDrainTick(app, env, branch, active, old) {
				return
			}
			<-ticker.C
		}
	}()
}

// At the maximum drain time, old sessions are expired and future requests
// switch to the new slot. In-flight requests are never cut short; removal
// waits for their proxy handlers to return, even beyond the configured limit.
func (s *Server) sessionDrainTick(app string, env DeployEnv, branch, active, old string) bool {
	selectionLock := s.edgeProxyLock(app, env, branch)
	selectionLock.Lock()
	st, err := s.getAppState(app, env, branch)
	if err != nil {
		selectionLock.Unlock()
		return false
	}
	if st == nil || normalizeActiveSlot(st.ActiveSlot) != active || normalizeActiveSlot(st.StandbySlot) != old || st.TrafficMode != "session" {
		selectionLock.Unlock()
		return true
	}
	if st.RolloutStatus != "retiring" {
		count, err := s.activeOldSessions(st, time.Now())
		if err != nil {
			selectionLock.Unlock()
			return false
		} // Unknown presence must keep the old slot.
		if time.Now().UnixMilli() < st.DrainUntil && (count > 0 || s.activeEdgeRequests(app, env, branch, old) > 0) {
			selectionLock.Unlock()
			return false
		}
		st.RolloutStatus = "retiring"
		if err := s.saveAppState(st); err != nil {
			selectionLock.Unlock()
			return false
		}
		s.broadcastSnapshot()
		if count > 0 {
			s.auditLog("relay-rollout", "session.max_drain", app, fmt.Sprintf("env=%s branch=%s expired_sessions=%d; waiting for active requests", env, branch, count))
		}
	}
	selectionLock.Unlock()
	if s.activeEdgeRequests(app, env, branch, old) > 0 {
		return false
	}
	if firstNonEmptyEngine(st.Engine) == EngineStation {
		if err := s.ensurestationEdgeProxy(nil, app, env, branch, active, "", st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, st.PublicHost, false); err != nil {
			return false
		}
		s.runtimeForEngine(st.Engine).Remove(appSlotContainerName(app, env, branch, old))
		st.StandbySlot = ""
		st.DrainUntil = 0
		st.RolloutStatus = "drained"
		if err := s.saveAppState(st); err != nil {
			return false
		}
		s.broadcastSnapshot()
	} else {
		s.retireStandbySlot(app, env, branch, active, old, st.ServicePort, st.HostPort, st.Mode, st.TrafficMode, st.PublicHost)
		updated, err := s.getAppState(app, env, branch)
		if err != nil || updated == nil || normalizeActiveSlot(updated.StandbySlot) == old {
			return false
		}
	}
	_, _ = s.db.Exec(`DELETE FROM edge_presence WHERE app=? AND env=? AND branch=? AND slot=?`, app, string(env), branch, old)
	return true
}

func (s *Server) resumeSessionDrains() error {
	rows, err := s.db.Query(`SELECT app,env,branch,active_slot,standby_slot FROM app_state WHERE traffic_mode='session' AND COALESCE(standby_slot,'')!=''`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var app, env, branch, active, old string
		if err := rows.Scan(&app, &env, &branch, &active, &old); err != nil {
			return err
		}
		s.startSessionDrain(app, DeployEnv(env), branch, active, old)
	}
	return rows.Err()
}

func edgeRequestKey(app string, env DeployEnv, branch, slot string) string {
	return app + "\x00" + string(env) + "\x00" + branch + "\x00" + slot
}

func (s *Server) edgeRequestStart(app string, env DeployEnv, branch, slot string) {
	edgeRequests.mu.Lock()
	defer edgeRequests.mu.Unlock()
	if edgeRequests.counts == nil {
		edgeRequests.counts = map[string]int{}
	}
	edgeRequests.counts[edgeRequestKey(app, env, branch, slot)]++
}

func (s *Server) edgeRequestDone(app string, env DeployEnv, branch, slot string) {
	edgeRequests.mu.Lock()
	defer edgeRequests.mu.Unlock()
	key := edgeRequestKey(app, env, branch, slot)
	edgeRequests.counts[key]--
}

func (s *Server) activeEdgeRequests(app string, env DeployEnv, branch, slot string) int {
	edgeRequests.mu.Lock()
	defer edgeRequests.mu.Unlock()
	return edgeRequests.counts[edgeRequestKey(app, env, branch, slot)]
}
