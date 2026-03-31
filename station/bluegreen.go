package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const proxyMagic = "__station_proxy__"

type SlotRecord struct {
	App             string `json:"app"`
	ProxyPort       int    `json:"proxy_port"`
	PID             int    `json:"pid,omitempty"`
	ActiveUpstream  string `json:"active_upstream,omitempty"`
	StandbyUpstream string `json:"standby_upstream,omitempty"`
	ActiveSlot      string `json:"active_slot,omitempty"`
	StandbySlot     string `json:"standby_slot,omitempty"`
	TrafficMode     string `json:"traffic_mode,omitempty"`
	CookieName      string `json:"cookie_name,omitempty"`
	PublicHost      string `json:"public_host,omitempty"`

	Upstream string `json:"upstream,omitempty"`
	Blue     string `json:"blue,omitempty"`
	Green    string `json:"green,omitempty"`
	Active   string `json:"active,omitempty"`
}

type proxyArgs struct {
	App             string
	ProxyPort       int
	ActiveUpstream  string
	StandbyUpstream string
	ActiveSlot      string
	StandbySlot     string
	TrafficMode     string
	CookieName      string
	PublicHost      string
	ClearStandby    bool
	ClearPublicHost bool
}

type proxyTargetMeta struct {
	Slot        string
	Upstream    string
	TrafficMode string
	CookieName  string
}

type proxyTargetKey struct{}

func firstProxyValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeProxySlot(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "blue":
		return "blue"
	case "green":
		return "green"
	default:
		return ""
	}
}

func normalizeProxyTrafficMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "session":
		return "session"
	case "", "edge":
		return "edge"
	default:
		return ""
	}
}

func proxyDir(app string) string {
	return filepath.Join(stateBaseDir(), "proxies", app)
}

func proxyConfigPath(app string) string {
	return filepath.Join(proxyDir(app), "proxy.json")
}

func proxyLogPath(app string) string {
	return filepath.Join(proxyDir(app), "proxy.log")
}

func normalizeSlotRecord(rec *SlotRecord) *SlotRecord {
	if rec == nil {
		return nil
	}
	if rec.ActiveUpstream == "" {
		rec.ActiveUpstream = strings.TrimSpace(rec.Upstream)
	}
	rec.ActiveSlot = firstProxyValue(normalizeProxySlot(rec.ActiveSlot), normalizeProxySlot(rec.Active), "blue")
	rec.StandbySlot = normalizeProxySlot(rec.StandbySlot)
	rec.ActiveUpstream = strings.TrimSpace(rec.ActiveUpstream)
	rec.StandbyUpstream = strings.TrimSpace(rec.StandbyUpstream)
	rec.TrafficMode = normalizeProxyTrafficMode(rec.TrafficMode)
	if rec.CookieName == "" {
		rec.CookieName = "station_slot"
	}
	rec.PublicHost = strings.TrimSpace(rec.PublicHost)
	if rec.StandbySlot == rec.ActiveSlot {
		rec.StandbySlot = ""
		rec.StandbyUpstream = ""
	}
	if rec.StandbyUpstream == "" {
		rec.StandbySlot = ""
	}
	return rec
}

func saveSlotRecord(rec *SlotRecord) error {
	rec = normalizeSlotRecord(rec)
	if err := os.MkdirAll(proxyDir(rec.App), 0755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(rec, "", "  ")
	return os.WriteFile(proxyConfigPath(rec.App), data, 0644)
}

func loadSlotRecord(app string) *SlotRecord {
	data, err := os.ReadFile(proxyConfigPath(app))
	if err != nil {
		return nil
	}
	var rec SlotRecord
	if json.Unmarshal(data, &rec) != nil {
		return nil
	}
	return normalizeSlotRecord(&rec)
}

func allSlotRecords() []*SlotRecord {
	matches, _ := filepath.Glob(filepath.Join(stateBaseDir(), "proxies", "*", "proxy.json"))
	var out []*SlotRecord
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		var rec SlotRecord
		if json.Unmarshal(data, &rec) == nil {
			out = append(out, normalizeSlotRecord(&rec))
		}
	}
	return out
}

func parseProxyArgs(args []string) proxyArgs {
	var cfg proxyArgs
	for i := 0; i < len(args); i++ {
		arg := args[i]
		next := func() string {
			if i+1 >= len(args) {
				die("missing value for %s", arg)
			}
			i++
			return args[i]
		}
		switch {
		case arg == "--app":
			cfg.App = next()
		case strings.HasPrefix(arg, "--app="):
			cfg.App = strings.TrimPrefix(arg, "--app=")
		case arg == "--port":
			cfg.ProxyPort, _ = strconv.Atoi(next())
		case strings.HasPrefix(arg, "--port="):
			cfg.ProxyPort, _ = strconv.Atoi(strings.TrimPrefix(arg, "--port="))
		case arg == "--upstream":
			cfg.ActiveUpstream = next()
		case strings.HasPrefix(arg, "--upstream="):
			cfg.ActiveUpstream = strings.TrimPrefix(arg, "--upstream=")
		case arg == "--active-upstream":
			cfg.ActiveUpstream = next()
		case strings.HasPrefix(arg, "--active-upstream="):
			cfg.ActiveUpstream = strings.TrimPrefix(arg, "--active-upstream=")
		case arg == "--standby-upstream":
			cfg.StandbyUpstream = next()
		case strings.HasPrefix(arg, "--standby-upstream="):
			cfg.StandbyUpstream = strings.TrimPrefix(arg, "--standby-upstream=")
		case arg == "--active-slot":
			cfg.ActiveSlot = next()
		case strings.HasPrefix(arg, "--active-slot="):
			cfg.ActiveSlot = strings.TrimPrefix(arg, "--active-slot=")
		case arg == "--standby-slot":
			cfg.StandbySlot = next()
		case strings.HasPrefix(arg, "--standby-slot="):
			cfg.StandbySlot = strings.TrimPrefix(arg, "--standby-slot=")
		case arg == "--traffic-mode":
			cfg.TrafficMode = next()
		case strings.HasPrefix(arg, "--traffic-mode="):
			cfg.TrafficMode = strings.TrimPrefix(arg, "--traffic-mode=")
		case arg == "--cookie-name":
			cfg.CookieName = next()
		case strings.HasPrefix(arg, "--cookie-name="):
			cfg.CookieName = strings.TrimPrefix(arg, "--cookie-name=")
		case arg == "--public-host":
			cfg.PublicHost = next()
		case strings.HasPrefix(arg, "--public-host="):
			cfg.PublicHost = strings.TrimPrefix(arg, "--public-host=")
		case arg == "--clear-standby":
			cfg.ClearStandby = true
		case arg == "--clear-public-host":
			cfg.ClearPublicHost = true
		}
	}
	return cfg
}

func isProxyRun() bool {
	return len(os.Args) > 1 && os.Args[1] == proxyMagic
}

func proxyHealthHandler(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func stripProxyHostPort(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	if parsed, err := url.Parse("http://" + host); err == nil && parsed.Hostname() != "" {
		return strings.ToLower(parsed.Hostname())
	}
	return host
}

func proxyTargetForRequest(rec *SlotRecord, r *http.Request) proxyTargetMeta {
	slot := rec.ActiveSlot
	override := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("__relay_target")))
	switch override {
	case "new", "live", "active", rec.ActiveSlot:
		slot = rec.ActiveSlot
	case "old", "previous", "standby", "draining", rec.StandbySlot:
		if rec.StandbyUpstream != "" {
			slot = rec.StandbySlot
		}
	default:
		if rec.TrafficMode == "session" && rec.CookieName != "" {
			if cookie, err := r.Cookie(rec.CookieName); err == nil {
				if normalizeProxySlot(cookie.Value) == rec.StandbySlot && rec.StandbyUpstream != "" {
					slot = rec.StandbySlot
				} else if normalizeProxySlot(cookie.Value) == rec.ActiveSlot {
					slot = rec.ActiveSlot
				}
			}
		}
	}

	upstream := rec.ActiveUpstream
	if slot == rec.StandbySlot && rec.StandbyUpstream != "" {
		upstream = rec.StandbyUpstream
	}
	return proxyTargetMeta{
		Slot:        slot,
		Upstream:    upstream,
		TrafficMode: rec.TrafficMode,
		CookieName:  rec.CookieName,
	}
}

func runProxyDaemon() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "proxy daemon: missing app name")
		os.Exit(1)
	}
	app := os.Args[2]

	if err := os.MkdirAll(proxyDir(app), 0755); err == nil {
		if lf, err := os.OpenFile(proxyLogPath(app), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			log.SetOutput(lf)
			defer lf.Close()
		}
	}

	rec := loadSlotRecord(app)
	if rec == nil {
		log.Printf("[proxy:%s] missing config", app)
		os.Exit(1)
	}

	var current atomic.Value
	current.Store(rec)

	go func() {
		lastJSON, _ := json.Marshal(rec)
		for {
			time.Sleep(500 * time.Millisecond)
			fresh := loadSlotRecord(app)
			if fresh == nil {
				os.Exit(0)
			}
			nextJSON, _ := json.Marshal(fresh)
			if string(nextJSON) == string(lastJSON) {
				continue
			}
			lastJSON = nextJSON
			current.Store(fresh)
			log.Printf("[proxy:%s] active=%s standby=%s traffic=%s host=%s", app, fresh.ActiveUpstream, fresh.StandbyUpstream, fresh.TrafficMode, fresh.PublicHost)
		}
	}()

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			cfg := current.Load().(*SlotRecord)
			meta := proxyTargetForRequest(cfg, req)
			ctx := context.WithValue(req.Context(), proxyTargetKey{}, meta)
			*req = *req.WithContext(ctx)
			req.URL.Scheme = "http"
			req.URL.Host = meta.Upstream
			if req.Header.Get("X-Forwarded-Host") == "" {
				req.Header.Set("X-Forwarded-Host", req.Host)
			}
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			req.Header.Set("X-Relay-Target", meta.Slot)
			req.Header.Set("X-Relay-Traffic-Mode", meta.TrafficMode)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			meta, _ := r.Context().Value(proxyTargetKey{}).(proxyTargetMeta)
			log.Printf("[proxy:%s] target=%s upstream=%s error=%v", app, meta.Slot, meta.Upstream, err)
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		},
		ModifyResponse: func(resp *http.Response) error {
			meta, _ := resp.Request.Context().Value(proxyTargetKey{}).(proxyTargetMeta)
			resp.Header.Del("X-Powered-By")
			resp.Header.Set("X-Relay-Target", meta.Slot)
			resp.Header.Set("X-Relay-Traffic-Mode", meta.TrafficMode)
			if meta.TrafficMode == "session" && meta.CookieName != "" && meta.Slot != "" {
				resp.Header.Add("Set-Cookie", fmt.Sprintf("%s=%s; Path=/; Max-Age=86400; SameSite=Lax", meta.CookieName, meta.Slot))
			}
			return nil
		},
		FlushInterval: -1,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/__relay/health", func(w http.ResponseWriter, r *http.Request) {
		proxyHealthHandler(w)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cfg := current.Load().(*SlotRecord)
		if cfg.PublicHost != "" && stripProxyHostPort(r.Host) != stripProxyHostPort(cfg.PublicHost) {
			http.NotFound(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", rec.ProxyPort)
	log.Printf("[proxy:%s] listening on %s active=%s standby=%s traffic=%s host=%s", app, addr, rec.ActiveUpstream, rec.StandbyUpstream, rec.TrafficMode, rec.PublicHost)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[proxy:%s] %v", app, err)
	}
}

func cmdProxyStart(cfg proxyArgs) {
	if cfg.App == "" {
		die("--app is required")
	}
	if cfg.ProxyPort == 0 {
		port, err := findFreePort()
		if err != nil {
			die("find free port: %v", err)
		}
		cfg.ProxyPort = port
	}
	cfg.ActiveUpstream = strings.TrimSpace(cfg.ActiveUpstream)
	if cfg.ActiveUpstream == "" {
		die("--active-upstream <host:port> is required")
	}
	if _, err := url.Parse("http://" + cfg.ActiveUpstream); err != nil {
		die("invalid active upstream %q: %v", cfg.ActiveUpstream, err)
	}
	if cfg.StandbyUpstream != "" {
		if _, err := url.Parse("http://" + cfg.StandbyUpstream); err != nil {
			die("invalid standby upstream %q: %v", cfg.StandbyUpstream, err)
		}
	}
	rec := normalizeSlotRecord(&SlotRecord{
		App:             cfg.App,
		ProxyPort:       cfg.ProxyPort,
		ActiveUpstream:  cfg.ActiveUpstream,
		StandbyUpstream: strings.TrimSpace(cfg.StandbyUpstream),
		ActiveSlot:      firstProxyValue(normalizeProxySlot(cfg.ActiveSlot), "blue"),
		StandbySlot:     normalizeProxySlot(cfg.StandbySlot),
		TrafficMode:     firstProxyValue(normalizeProxyTrafficMode(cfg.TrafficMode), "edge"),
		CookieName:      firstProxyValue(strings.TrimSpace(cfg.CookieName), "station_slot"),
		PublicHost:      strings.TrimSpace(cfg.PublicHost),
	})
	if err := saveSlotRecord(rec); err != nil {
		die("save proxy config: %v", err)
	}

	self, err := os.Executable()
	if err != nil {
		die("resolve executable: %v", err)
	}
	daemon, err := startDetachedProcess(self, []string{proxyMagic, cfg.App})
	if err != nil {
		die("start proxy daemon: %v", err)
	}
	rec.PID = daemon
	if err := saveSlotRecord(rec); err != nil {
		die("save proxy pid: %v", err)
	}
	fmt.Printf("proxy started  app=%s  port=%d  active=%s  standby=%s  pid=%d\n", cfg.App, cfg.ProxyPort, rec.ActiveUpstream, rec.StandbyUpstream, daemon)
}

func cmdProxySwap(cfg proxyArgs) {
	rec := loadSlotRecord(cfg.App)
	if rec == nil {
		die("no proxy running for app %q", cfg.App)
	}
	if strings.TrimSpace(cfg.ActiveUpstream) != "" {
		rec.ActiveUpstream = strings.TrimSpace(cfg.ActiveUpstream)
	}
	if cfg.ClearStandby {
		rec.StandbyUpstream = ""
		rec.StandbySlot = ""
	} else if strings.TrimSpace(cfg.StandbyUpstream) != "" {
		rec.StandbyUpstream = strings.TrimSpace(cfg.StandbyUpstream)
		rec.StandbySlot = firstProxyValue(normalizeProxySlot(cfg.StandbySlot), rec.StandbySlot, nextProxySlot(rec.ActiveSlot))
	}
	if slot := normalizeProxySlot(cfg.ActiveSlot); slot != "" {
		rec.ActiveSlot = slot
	}
	if slot := normalizeProxySlot(cfg.StandbySlot); slot != "" && !cfg.ClearStandby {
		rec.StandbySlot = slot
	}
	if mode := normalizeProxyTrafficMode(cfg.TrafficMode); mode != "" {
		rec.TrafficMode = mode
	}
	if cookie := strings.TrimSpace(cfg.CookieName); cookie != "" {
		rec.CookieName = cookie
	}
	if cfg.ClearPublicHost {
		rec.PublicHost = ""
	} else if cfg.PublicHost != "" {
		rec.PublicHost = strings.TrimSpace(cfg.PublicHost)
	}
	if err := saveSlotRecord(rec); err != nil {
		die("save proxy config: %v", err)
	}
	fmt.Printf("proxy %s updated  active=%s  standby=%s  traffic=%s  host=%s\n", cfg.App, rec.ActiveUpstream, rec.StandbyUpstream, rec.TrafficMode, rec.PublicHost)
}

func nextProxySlot(slot string) string {
	if normalizeProxySlot(slot) == "blue" {
		return "green"
	}
	return "blue"
}

func cmdProxyStop(app string) {
	if err := stopProxy(app); err != nil {
		die("%v", err)
	}
	fmt.Printf("proxy %s stopped\n", app)
}

func stopProxy(app string) error {
	rec := loadSlotRecord(app)
	if rec == nil {
		return fmt.Errorf("no proxy config for app %q", app)
	}
	_ = os.RemoveAll(proxyDir(app))
	if rec.PID > 0 && pidAlive(rec.PID) {
		_ = killProcess(rec.PID)
	}
	return nil
}

func cmdProxyList() {
	recs := allSlotRecords()
	if len(recs) == 0 {
		fmt.Println("no proxies")
		return
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].App < recs[j].App })
	fmt.Printf("%-24s %-6s %-22s %-22s %-8s %-7s\n", "APP", "PORT", "ACTIVE", "STANDBY", "TRAFFIC", "PID")
	for _, rec := range recs {
		status := "stopped"
		if rec.PID > 0 && pidAlive(rec.PID) {
			status = fmt.Sprintf("%d", rec.PID)
		}
		fmt.Printf("%-24s %-6d %-22s %-22s %-8s %-7s\n", rec.App, rec.ProxyPort, rec.ActiveUpstream, rec.StandbyUpstream, rec.TrafficMode, status)
	}
}

