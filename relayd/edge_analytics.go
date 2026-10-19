package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Edge analytics: every lane's edge proxy writes one JSON line per request to
// stdout (see edgeAccessLogFormat). Relay follows each running edge
// container's log stream and stores the requests, attributed to the lane,
// slot and variant that served them. This is what makes analytics and the
// Operations page show real traffic: before, only requests that passed
// through the global Caddy proxy were counted, so port-mode lanes and
// anything fronted directly by Cloudflare showed nothing.

type edgeLogLine struct {
	Ts      float64 `json:"ts"`
	Status  int     `json:"status"`
	Host    string  `json:"host"`
	Method  string  `json:"method"`
	Path    string  `json:"path"`
	Slot    string  `json:"slot"`
	App     string  `json:"app"`
	Env     string  `json:"env"`
	Branch  string  `json:"branch"`
	Variant string  `json:"variant"`
	IP      string  `json:"ip"`
	CC      string  `json:"cc"`
	RT      string  `json:"rt"`
	Bytes   string  `json:"bytes"`
	UA      string  `json:"ua"`
	Ref     string  `json:"ref"`
	Accept  string  `json:"accept"`
}

type analyticsEvent struct {
	TS         int64 // unix seconds
	Host       string
	Method     string
	Path       string
	Status     int
	Bytes      int64
	IP         string
	Country    string
	App        string
	Env        string
	Branch     string
	Slot       string
	Variant    string
	DurationMS int64
	Kind       string
	Visitor    string
	Referrer   string
	Source     string
}

var assetExtensions = map[string]bool{
	".js": true, ".mjs": true, ".css": true, ".map": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".svg": true, ".webp": true, ".avif": true, ".ico": true, ".woff": true, ".woff2": true,
	".ttf": true, ".otf": true, ".eot": true, ".mp4": true, ".webm": true, ".mp3": true, ".wasm": true,
	".txt": true, ".xml": true, ".json": true, ".pdf": true, ".zip": true,
}

func looksLikeBot(ua string) bool {
	ua = strings.ToLower(ua)
	if ua == "" {
		return true
	}
	for _, marker := range []string{"bot", "crawl", "spider", "slurp", "curl/", "wget/", "python-requests", "go-http-client", "headless", "lighthouse", "uptime", "monitor", "preview"} {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

// classifyRequest buckets a request the way a hosting dashboard does: "page"
// is an HTML document navigation (what page views count), "asset" a static
// file, "api" a backend call, "bot" a crawler page hit, "other" the rest.
func classifyRequest(method string, rawPath string, accept string, ua string) string {
	p := rawPath
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if strings.HasPrefix(p, "/__relay/") {
		return "other"
	}
	if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/graphql") || strings.HasPrefix(p, "/trpc/") {
		return "api"
	}
	if ext := strings.ToLower(path.Ext(p)); ext != "" && ext != ".html" && ext != ".htm" && ext != ".php" {
		if assetExtensions[ext] || strings.HasPrefix(p, "/_next/static/") || strings.HasPrefix(p, "/assets/") {
			return "asset"
		}
	}
	if strings.HasPrefix(p, "/_next/") || strings.HasPrefix(p, "/static/") {
		return "asset"
	}
	if (method == "GET" || method == "HEAD") && strings.Contains(strings.ToLower(accept), "text/html") {
		if looksLikeBot(ua) {
			return "bot"
		}
		return "page"
	}
	return "other"
}

func referrerHost(ref string, host string) string {
	if ref == "" || ref == "-" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return ""
	}
	h := strings.ToLower(u.Hostname())
	if strings.EqualFold(h, strings.Split(host, ":")[0]) {
		return "" // internal navigation isn't a referrer
	}
	return strings.TrimPrefix(h, "www.")
}

// parseEdgeLogLine turns one edge access log line into an analytics event.
func (s *Server) parseEdgeLogLine(line []byte) (analyticsEvent, int64, bool) {
	line = []byte(strings.TrimSpace(string(line)))
	if len(line) == 0 || line[0] != '{' {
		return analyticsEvent{}, 0, false
	}
	var l edgeLogLine
	if err := json.Unmarshal(line, &l); err != nil || l.Ts <= 0 || l.App == "" {
		return analyticsEvent{}, 0, false
	}
	tsMS := int64(l.Ts * 1000)
	at := time.UnixMilli(tsMS)
	rt, _ := strconv.ParseFloat(l.RT, 64)
	bytes, _ := strconv.ParseInt(l.Bytes, 10, 64)
	p := l.Path
	if len(p) > 256 {
		p = p[:256]
	}
	cc := strings.ToUpper(strings.TrimSpace(l.CC))
	if cc == "XX" || cc == "T1" || len(cc) != 2 {
		cc = ""
	}
	kind := classifyRequest(l.Method, l.Path, l.Accept, l.UA)
	ev := analyticsEvent{
		TS:         at.Unix(),
		Host:       l.Host,
		Method:     l.Method,
		Path:       p,
		Status:     l.Status,
		Bytes:      bytes,
		IP:         l.IP,
		Country:    cc,
		App:        l.App,
		Env:        l.Env,
		Branch:     l.Branch,
		Slot:       l.Slot,
		Variant:    l.Variant,
		DurationMS: int64(rt * 1000),
		Kind:       kind,
		Referrer:   referrerHost(l.Ref, l.Host),
		Source:     "edge",
	}
	if kind == "page" {
		ev.Visitor = s.visitorHash(l.IP, l.UA, at)
	}
	return ev, tsMS, true
}

func insertAnalyticsBatch(adb *sql.DB, events []analyticsEvent) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := adb.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO analytics_events
		(ts, host, method, path, status, bytes, remote_ip, country_code, app, env, branch, slot, variant, duration_ms, kind, visitor, referrer, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, e := range events {
		_, _ = stmt.Exec(e.TS, e.Host, e.Method, e.Path, e.Status, e.Bytes, e.IP, e.Country, e.App, e.Env, e.Branch, e.Slot, e.Variant, e.DurationMS, e.Kind, e.Visitor, e.Referrer, e.Source)
	}
	return tx.Commit()
}

func edgeLogCheckpointKey(container string) string {
	return "edge_log_ckpt:" + container
}

func (s *Server) edgeLogCheckpoint(container string) int64 {
	var v string
	_ = s.db.QueryRow(`SELECT value FROM server_config WHERE key=?`, edgeLogCheckpointKey(container)).Scan(&v)
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

func (s *Server) saveEdgeLogCheckpoint(container string, ms int64) {
	_, _ = s.db.Exec(`INSERT OR REPLACE INTO server_config(key, value) VALUES(?, ?)`, edgeLogCheckpointKey(container), strconv.FormatInt(ms, 10))
}

// startEdgeAnalytics keeps one log follower per running Docker edge proxy.
func (s *Server) startEdgeAnalytics() {
	if _, err := exec.LookPath("docker"); err != nil {
		return
	}
	var mu sync.Mutex
	followers := map[string]context.CancelFunc{}
	for {
		wanted := map[string]bool{}
		rows, err := s.db.Query(`SELECT app, env, branch FROM app_state WHERE COALESCE(stopped,0)=0 AND COALESCE(engine,'docker') IN ('', 'docker')`)
		if err == nil {
			for rows.Next() {
				var app, env, branch string
				if rows.Scan(&app, &env, &branch) == nil {
					wanted[appBaseContainerName(app, DeployEnv(env), branch)] = true
				}
			}
			rows.Close()
		}
		mu.Lock()
		for name, cancel := range followers {
			if !wanted[name] {
				cancel()
				delete(followers, name)
			}
		}
		for name := range wanted {
			if _, ok := followers[name]; ok || !s.runtime.IsRunning(name) {
				continue
			}
			ctx, cancel := context.WithCancel(context.Background())
			followers[name] = cancel
			go func(name string) {
				s.followEdgeLog(ctx, name)
				mu.Lock()
				if followers[name] != nil {
					delete(followers, name)
				}
				mu.Unlock()
				cancel()
			}(name)
		}
		mu.Unlock()
		time.Sleep(15 * time.Second)
	}
}

// followEdgeLog streams one edge container's access log until the container
// stops (or the lane goes away), inserting events in small batches and
// checkpointing the last stored timestamp so a restart neither drops nor
// double-counts traffic.
func (s *Server) followEdgeLog(ctx context.Context, container string) {
	ckpt := s.edgeLogCheckpoint(container)
	since := strconv.FormatInt(time.Now().Unix(), 10)
	if ckpt > 0 {
		since = strconv.FormatFloat(float64(ckpt)/1000, 'f', 3, 64)
	}
	cmd := exec.CommandContext(ctx, "docker", "logs", "-f", "--since", since, container)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	defer func() { _ = cmd.Wait() }()

	adb := s.analyticsStore()
	lines := make(chan []byte, 1024)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
		for sc.Scan() {
			b := append([]byte(nil), sc.Bytes()...)
			select {
			case lines <- b:
			case <-ctx.Done():
				return
			}
		}
	}()

	batch := make([]analyticsEvent, 0, 256)
	lastMS := ckpt
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if insertAnalyticsBatch(adb, batch) == nil {
			s.saveEdgeLogCheckpoint(container, lastMS)
		}
		batch = batch[:0]
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				flush()
				return
			}
			ev, tsMS, ok := s.parseEdgeLogLine(line)
			if !ok || tsMS <= ckpt {
				continue
			}
			batch = append(batch, ev)
			if tsMS > lastMS {
				lastMS = tsMS
			}
			if len(batch) >= 500 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}

var edgeHostCache = struct {
	sync.Mutex
	at    time.Time
	hosts map[string]bool
}{}

// edgeManagedHosts are public hosts served by a Docker lane's edge proxy.
// The Caddy tailer skips them so each request is counted once (by the edge,
// which also knows the lane, slot and variant).
func (s *Server) edgeManagedHosts() map[string]bool {
	edgeHostCache.Lock()
	defer edgeHostCache.Unlock()
	if edgeHostCache.hosts != nil && time.Since(edgeHostCache.at) < time.Minute {
		return edgeHostCache.hosts
	}
	hosts := map[string]bool{}
	rows, err := s.db.Query(`SELECT public_host, COALESCE(public_hosts,'') FROM app_state WHERE COALESCE(engine,'docker') IN ('', 'docker')`)
	if err == nil {
		for rows.Next() {
			var primary, extra string
			if rows.Scan(&primary, &extra) == nil {
				_, all := canonicalizePublicHosts(primary, parsePublicHosts(extra))
				for _, h := range all {
					hosts[strings.ToLower(strings.TrimSpace(h))] = true
				}
			}
		}
		rows.Close()
	}
	edgeHostCache.hosts, edgeHostCache.at = hosts, time.Now()
	return hosts
}
