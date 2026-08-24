package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type caddyLogEntry struct {
	Ts      float64 `json:"ts"`
	Request struct {
		RemoteIP string `json:"remote_ip"`
		Method   string `json:"method"`
		Host     string `json:"host"`
		URI      string `json:"uri"`
	} `json:"request"`
	Status int   `json:"status"`
	Size   int64 `json:"size"`
}

func (s *Server) analyticsStore() *sql.DB {
	if s.analyticsDB != nil {
		return s.analyticsDB
	}
	return s.db
}

// startLogTailer reads new lines appended to the Caddy access log, inserts
// analytics events into SQLite, and resolves country codes in the background.
func (s *Server) startLogTailer() {
	logPath := filepath.Join(s.caddyLogsDir, "access.log")
	adb := s.analyticsStore()

	// Persist the byte offset between restarts so we don't re-process old events.
	getOffset := func() int64 {
		var v string
		_ = s.db.QueryRow(`SELECT value FROM server_config WHERE key='analytics_log_offset'`).Scan(&v)
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	saveOffset := func(off int64) {
		_, _ = s.db.Exec(`INSERT OR REPLACE INTO server_config(key,value) VALUES('analytics_log_offset',?)`, strconv.FormatInt(off, 10))
	}

	offset := getOffset()

	for {
		f, err := os.Open(logPath)
		if err != nil {
			// File doesn't exist yet (Caddy not started). Wait and retry.
			time.Sleep(5 * time.Second)
			continue
		}

		fi, _ := f.Stat()
		// If the file shrank (roll / truncation), reset to beginning.
		if offset > fi.Size() {
			offset = 0
			saveOffset(0)
		}
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			f.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(f)
		var newEvents []caddyLogEntry
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			var entry caddyLogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			// Skip internal Caddy admin / health check lines.
			if entry.Ts == 0 || entry.Request.Host == "" {
				continue
			}
			newEvents = append(newEvents, entry)
		}

		newOffset, _ := f.Seek(0, io.SeekCurrent)
		f.Close()

		if len(newEvents) > 0 {
			s.insertAnalyticsEvents(adb, newEvents)
			saveOffset(newOffset)
			offset = newOffset
			// Kick off a non-blocking country resolver pass.
			go s.resolveAnalyticsCountries()
		} else {
			offset = newOffset
		}

		time.Sleep(2 * time.Second)
	}
}

func (s *Server) insertAnalyticsEvents(adb *sql.DB, entries []caddyLogEntry) {
	tx, err := adb.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare(`INSERT INTO analytics_events(ts,host,method,path,status,bytes,remote_ip) VALUES(?,?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()
	for _, e := range entries {
		ts := int64(e.Ts)
		uri := e.Request.URI
		if len(uri) > 256 {
			uri = uri[:256]
		}
		if _, err := stmt.Exec(ts, e.Request.Host, e.Request.Method, uri, e.Status, e.Size, e.Request.RemoteIP); err != nil {
			continue
		}
	}
	tx.Commit()
}

// ipAPIBatchResponse is one item from the ip-api.com /batch response.
type ipAPIBatchResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Query       string `json:"query"`
}

// resolveAnalyticsCountries batch-resolves remote IPs to country codes using
// ip-api.com (free, no key, 45 req/min).  Results are cached in ip_country_cache.
func (s *Server) resolveAnalyticsCountries() {
	adb := s.analyticsStore()
	// Collect distinct IPs that have no country assigned yet.
	rows, err := adb.Query(`
		SELECT DISTINCT remote_ip FROM analytics_events
		WHERE country_code='' AND remote_ip!=''
		LIMIT 500`)
	if err != nil {
		return
	}
	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err == nil && ip != "" {
			ips = append(ips, ip)
		}
	}
	rows.Close()
	if len(ips) == 0 {
		return
	}

	// Filter out already-cached IPs in a single IN query.
	uncached := ips[:0]
	if len(ips) > 0 {
		placeholders := make([]string, len(ips))
		args := make([]any, len(ips))
		for i, ip := range ips {
			placeholders[i] = "?"
			args[i] = ip
		}
		cachedRows, err2 := adb.Query(
			`SELECT ip FROM ip_country_cache WHERE ip IN (`+strings.Join(placeholders, ",")+`) AND country_code != ''`,
			args...)
		cached := map[string]bool{}
		if err2 == nil {
			for cachedRows.Next() {
				var ip string
				if cachedRows.Scan(&ip) == nil {
					cached[ip] = true
				}
			}
			cachedRows.Close()
		}
		for _, ip := range ips {
			if !cached[ip] {
				uncached = append(uncached, ip)
			}
		}
	}

	// Process in batches of up to 100 (ip-api.com limit).
	for i := 0; i < len(uncached); i += 100 {
		end := i + 100
		if end > len(uncached) {
			end = len(uncached)
		}
		batch := uncached[i:end]

		type req struct {
			Query  string `json:"query"`
			Fields string `json:"fields"`
		}
		reqs := make([]req, len(batch))
		for j, ip := range batch {
			reqs[j] = req{Query: ip, Fields: "status,country,countryCode,query"}
		}
		body, _ := json.Marshal(reqs)
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Post(
			"http://ip-api.com/batch", "application/json", bytes.NewReader(body))
		if err != nil {
			continue
		}
		var results []ipAPIBatchResponse
		_ = json.NewDecoder(resp.Body).Decode(&results)
		resp.Body.Close()

		now := time.Now().Unix()
		for _, r := range results {
			if r.Status != "success" || r.Query == "" {
				continue
			}
			_, _ = adb.Exec(
				`INSERT OR REPLACE INTO ip_country_cache(ip,country_code,country_name,updated_at) VALUES(?,?,?,?)`,
				r.Query, r.CountryCode, r.Country, now)
		}
		// Small back-off to stay under the 45 req/min rate limit.
		if i+100 < len(uncached) {
			time.Sleep(1500 * time.Millisecond)
		}
	}

	// Apply cached countries to analytics_events rows that are still empty.
	_, _ = adb.Exec(`
		UPDATE analytics_events
		SET country_code = (SELECT country_code FROM ip_country_cache WHERE ip=remote_ip),
		    country_name = (SELECT country_name FROM ip_country_cache WHERE ip=remote_ip)
		WHERE country_code='' AND remote_ip!=''
		  AND EXISTS (SELECT 1 FROM ip_country_cache WHERE ip=remote_ip)`)
}

type analyticsResponse struct {
	TotalRequests int64              `json:"total_requests"`
	PeriodLabel   string             `json:"period"`
	ByCountry     []analyticsCountry `json:"by_country"`
	ByStatus      []analyticsStatus  `json:"by_status"`
	ByHour        []analyticsHour    `json:"by_hour"`
	ByHost        []analyticsHost    `json:"by_host"`
}
type analyticsCountry struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
type analyticsStatus struct {
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}
type analyticsHour struct {
	Ts    int64 `json:"ts"`
	Count int64 `json:"count"`
}
type analyticsHost struct {
	Host  string `json:"host"`
	Count int64  `json:"count"`
}

type adminOpsComparisonWindow struct {
	Seconds int64 `json:"seconds"`
}

type adminOpsTrafficWindow struct {
	Requests      int64   `json:"requests"`
	ServerErrors  int64   `json:"server_errors"`
	ClientErrors  int64   `json:"client_errors"`
	Bandwidth     int64   `json:"bandwidth_bytes"`
	ServerErrRate float64 `json:"server_error_rate"`
}

type adminOpsDeploySummary struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Source           string `json:"source,omitempty"`
	CreatedAt        string `json:"created_at"`
	StartedAt        string `json:"started_at,omitempty"`
	EndedAt          string `json:"ended_at,omitempty"`
	BuildNumber      int    `json:"build_number,omitempty"`
	BuildDurationMS  int64  `json:"build_duration_ms,omitempty"`
	CommitSHA        string `json:"commit_sha,omitempty"`
	CommitMessage    string `json:"commit_message,omitempty"`
	DeployedBy       string `json:"deployed_by,omitempty"`
	ImageTag         string `json:"image_tag,omitempty"`
	PreviousImageTag string `json:"previous_image_tag,omitempty"`
}

type adminOpsDeployDelta struct {
	Current              adminOpsDeploySummary    `json:"current"`
	Previous             *adminOpsDeploySummary   `json:"previous,omitempty"`
	Window               adminOpsComparisonWindow `json:"window"`
	CurrentTraffic       *adminOpsTrafficWindow   `json:"current_traffic,omitempty"`
	PreviousTraffic      *adminOpsTrafficWindow   `json:"previous_traffic,omitempty"`
	BuildDurationDeltaMS int64                    `json:"build_duration_delta_ms,omitempty"`
	ServerErrRateDelta   float64                  `json:"server_error_rate_delta,omitempty"`
	RequestDelta         int64                    `json:"request_delta,omitempty"`
	BandwidthDelta       int64                    `json:"bandwidth_delta_bytes,omitempty"`
	AnalyticsAvailable   bool                     `json:"analytics_available"`
	AnalyticsNote        string                   `json:"analytics_note,omitempty"`
}

type adminOpsContainerUsage struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Kind          string  `json:"kind"`
	Container     string  `json:"container"`
	Running       bool    `json:"running"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsageBytes int64   `json:"mem_usage_bytes"`
	MemLimitBytes int64   `json:"mem_limit_bytes"`
	MemPercent    float64 `json:"mem_percent"`
	StorageBytes  int64   `json:"storage_bytes"`
	NetRxBytes    int64   `json:"net_rx_bytes"`
	NetTxBytes    int64   `json:"net_tx_bytes"`
	BlockRead     int64   `json:"block_read_bytes"`
	BlockWrite    int64   `json:"block_write_bytes"`
}

type adminOpsLaneUsage struct {
	CPUPercent        float64                  `json:"cpu_percent"`
	MemUsageBytes     int64                    `json:"mem_usage_bytes"`
	MemLimitBytes     int64                    `json:"mem_limit_bytes"`
	MemPercent        float64                  `json:"mem_percent"`
	StorageBytes      int64                    `json:"storage_bytes"`
	NetRxBytes        int64                    `json:"net_rx_bytes"`
	NetTxBytes        int64                    `json:"net_tx_bytes"`
	BlockReadBytes    int64                    `json:"block_read_bytes"`
	BlockWriteBytes   int64                    `json:"block_write_bytes"`
	RunningContainers int                      `json:"running_containers"`
	ContainerCount    int                      `json:"container_count"`
	Measured          bool                     `json:"measured"`
	Note              string                   `json:"note,omitempty"`
	Targets           []adminOpsContainerUsage `json:"targets"`
}

type adminOpsLane struct {
	App           string               `json:"app"`
	Env           string               `json:"env"`
	Branch        string               `json:"branch"`
	Engine        string               `json:"engine"`
	PublicHost    string               `json:"public_host,omitempty"`
	HostPort      int                  `json:"host_port,omitempty"`
	RepoURL       string               `json:"repo_url,omitempty"`
	Stopped       bool                 `json:"stopped"`
	CurrentImage  string               `json:"current_image,omitempty"`
	PreviousImage string               `json:"previous_image,omitempty"`
	Usage         adminOpsLaneUsage    `json:"usage"`
	Latest        *adminOpsDeployDelta `json:"latest,omitempty"`
}

type adminOpsApp struct {
	App         string            `json:"app"`
	LaneCount   int               `json:"lane_count"`
	OnlineLanes int               `json:"online_lanes"`
	Usage       adminOpsLaneUsage `json:"usage"`
	Lanes       []adminOpsLane    `json:"lanes"`
}

type adminOpsSummary struct {
	AppCount      int     `json:"app_count"`
	LaneCount     int     `json:"lane_count"`
	OnlineLanes   int     `json:"online_lanes"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsageBytes int64   `json:"mem_usage_bytes"`
	MemLimitBytes int64   `json:"mem_limit_bytes"`
	MemPercent    float64 `json:"mem_percent"`
	StorageBytes  int64   `json:"storage_bytes"`
}

type adminOpsResponse struct {
	GeneratedAt int64           `json:"generated_at"`
	Summary     adminOpsSummary `json:"summary"`
	Apps        []adminOpsApp   `json:"apps"`
	Daemon      adminOpsDaemon  `json:"daemon"`
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	adb := s.analyticsStore()
	period := r.URL.Query().Get("period")
	app := strings.TrimSpace(r.URL.Query().Get("app"))
	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	env := DeployEnv(strings.TrimSpace(r.URL.Query().Get("env")))
	var sess *UserSession
	if s.hasUsers() {
		sess = s.validateUserSession(r)
		if sess == nil {
			httpError(w, 401, "unauthorized")
			return
		}
	}

	var since int64
	var periodLabel string
	now := time.Now().Unix()
	switch period {
	case "24h":
		since = now - 86400
		periodLabel = "24h"
	case "30d":
		since = now - 30*86400
		periodLabel = "30d"
	default:
		since = now - 7*86400
		periodLabel = "7d"
	}

	hostFilter := ""
	hostArgs := []any{since}
	if app != "" {
		if sess != nil {
			allowed := false
			if env != "" {
				allowed = roleAtLeast(s.effectiveLaneRole(sess, app, env), "viewer")
			} else {
				rows, err := s.db.Query(`SELECT DISTINCT env FROM app_state WHERE app=?`, app)
				if err == nil {
					for rows.Next() {
						var laneEnv string
						if rows.Scan(&laneEnv) == nil && roleAtLeast(s.effectiveLaneRole(sess, app, DeployEnv(laneEnv)), "viewer") {
							allowed = true
							break
						}
					}
					rows.Close()
				}
			}
			if !allowed {
				httpError(w, 403, "insufficient app access")
				return
			}
		}
		hosts, err := s.analyticsHostsForSelection(app, env, branch)
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if len(hosts) == 0 {
			writeJSON(w, 200, analyticsResponse{PeriodLabel: periodLabel})
			return
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(hosts)), ",")
		hostFilter = ` AND host IN (` + placeholders + `)`
		for _, host := range hosts {
			hostArgs = append(hostArgs, host)
		}
	}

	// Total requests.
	var total int64
	_ = adb.QueryRow(`SELECT COUNT(*) FROM analytics_events WHERE ts>=?`+hostFilter, hostArgs...).Scan(&total)

	// By country.
	countryRows, _ := adb.Query(
		`SELECT COALESCE(NULLIF(country_code,''),'??') AS cc,
		        COALESCE(NULLIF(country_name,''),'Unknown') AS cn,
		        COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY cc ORDER BY cnt DESC LIMIT 30`,
		hostArgs...)
	var byCountry []analyticsCountry
	if countryRows != nil {
		for countryRows.Next() {
			var c analyticsCountry
			_ = countryRows.Scan(&c.Code, &c.Name, &c.Count)
			byCountry = append(byCountry, c)
		}
		countryRows.Close()
	}

	// By status class (group into 2xx, 3xx, 4xx, 5xx).
	statusRows, _ := adb.Query(
		`SELECT (status/100)*100 AS sc, COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY sc ORDER BY sc`,
		hostArgs...)
	var byStatus []analyticsStatus
	if statusRows != nil {
		for statusRows.Next() {
			var st analyticsStatus
			_ = statusRows.Scan(&st.Status, &st.Count)
			byStatus = append(byStatus, st)
		}
		statusRows.Close()
	}

	// By hour (bucket ts to nearest hour).
	bucketSize := int64(3600)
	if period == "30d" {
		bucketSize = 86400 // daily buckets for 30-day view
	}
	hourRows, _ := adb.Query(
		`SELECT (ts/?)*? AS bucket, COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY bucket ORDER BY bucket`,
		append([]any{bucketSize, bucketSize}, hostArgs...)...)
	var byHour []analyticsHour
	if hourRows != nil {
		for hourRows.Next() {
			var h analyticsHour
			_ = hourRows.Scan(&h.Ts, &h.Count)
			byHour = append(byHour, h)
		}
		hourRows.Close()
	}

	// By host.
	hostRows, _ := adb.Query(
		`SELECT host, COUNT(*) AS cnt
		 FROM analytics_events WHERE ts>=?`+hostFilter+`
		 GROUP BY host ORDER BY cnt DESC LIMIT 20`,
		hostArgs...)
	var byHost []analyticsHost
	if hostRows != nil {
		for hostRows.Next() {
			var h analyticsHost
			_ = hostRows.Scan(&h.Host, &h.Count)
			byHost = append(byHost, h)
		}
		hostRows.Close()
	}

	resp := analyticsResponse{
		TotalRequests: total,
		PeriodLabel:   periodLabel,
		ByCountry:     byCountry,
		ByStatus:      byStatus,
		ByHour:        byHour,
		ByHost:        byHost,
	}
	if resp.ByCountry == nil {
		resp.ByCountry = []analyticsCountry{}
	}
	if resp.ByStatus == nil {
		resp.ByStatus = []analyticsStatus{}
	}
	if resp.ByHour == nil {
		resp.ByHour = []analyticsHour{}
	}
	if resp.ByHost == nil {
		resp.ByHost = []analyticsHost{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type dockerStatsRow struct {
	CPUPercent float64
	MemUsage   int64
	MemLimit   int64
	MemPercent float64
	NetRx      int64
	NetTx      int64
	BlockRead  int64
	BlockWrite int64
}

type dockerStorageRow struct {
	Running    bool
	SizeRW     int64
	SizeRootFS int64
}

func parseDockerPercent(raw string) float64 {
	value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "%"))
	if value == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(value, 64)
	return f
}

func parseDockerBytes(raw string) int64 {
	value := strings.TrimSpace(raw)
	if value == "" || value == "--" || value == "0" || value == "0B" {
		return 0
	}
	value = strings.ReplaceAll(value, " ", "")
	splitAt := -1
	for i, r := range value {
		if !(r == '.' || r == '-' || (r >= '0' && r <= '9')) {
			splitAt = i
			break
		}
	}
	if splitAt <= 0 {
		n, _ := strconv.ParseFloat(value, 64)
		return int64(n)
	}
	numPart := value[:splitAt]
	unitPart := strings.ToUpper(strings.TrimSpace(value[splitAt:]))
	num, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0
	}
	units := map[string]float64{
		"B":  1,
		"KB": 1000, "MB": 1000 * 1000, "GB": 1000 * 1000 * 1000, "TB": 1000 * 1000 * 1000 * 1000,
		"KIB": 1024, "MIB": 1024 * 1024, "GIB": 1024 * 1024 * 1024, "TIB": 1024 * 1024 * 1024 * 1024,
	}
	multiplier, ok := units[unitPart]
	if !ok {
		multiplier = 1
	}
	return int64(num * multiplier)
}

func parseDockerIOPair(raw string) (int64, int64) {
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseDockerBytes(parts[0]), parseDockerBytes(parts[1])
}

func collectDockerStats(containers []string) map[string]dockerStatsRow {
	out := map[string]dockerStatsRow{}
	if len(containers) == 0 {
		return out
	}
	args := []string{
		"stats", "--no-stream",
		"--format", "{{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}",
	}
	args = append(args, containers...)
	cmd := exec.Command("docker", args...)
	data, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}
		memParts := strings.Split(fields[2], "/")
		memUsage := int64(0)
		memLimit := int64(0)
		if len(memParts) == 2 {
			memUsage = parseDockerBytes(memParts[0])
			memLimit = parseDockerBytes(memParts[1])
		}
		netRx, netTx := parseDockerIOPair(fields[4])
		blockRead, blockWrite := parseDockerIOPair(fields[5])
		out[strings.TrimSpace(fields[0])] = dockerStatsRow{
			CPUPercent: parseDockerPercent(fields[1]),
			MemUsage:   memUsage,
			MemLimit:   memLimit,
			MemPercent: parseDockerPercent(fields[3]),
			NetRx:      netRx,
			NetTx:      netTx,
			BlockRead:  blockRead,
			BlockWrite: blockWrite,
		}
	}
	return out
}

func collectDockerStorage(containers []string) map[string]dockerStorageRow {
	out := map[string]dockerStorageRow{}
	if len(containers) == 0 {
		return out
	}
	args := []string{
		"inspect", "--size",
		"--format", "{{.Name}}\t{{.State.Running}}\t{{.SizeRw}}\t{{.SizeRootFs}}",
	}
	args = append(args, containers...)
	cmd := exec.Command("docker", args...)
	data, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		name := strings.TrimPrefix(strings.TrimSpace(fields[0]), "/")
		sizeRW, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		sizeRootFS, _ := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64)
		out[name] = dockerStorageRow{
			Running:    strings.EqualFold(strings.TrimSpace(fields[1]), "true"),
			SizeRW:     sizeRW,
			SizeRootFS: sizeRootFS,
		}
	}
	return out
}

func deployBuildDurationMS(d *Deploy) int64 {
	if d == nil || d.StartedAt == nil || d.EndedAt == nil {
		return 0
	}
	return d.EndedAt.Sub(*d.StartedAt).Milliseconds()
}

func toAdminDeploySummary(d *Deploy) adminOpsDeploySummary {
	if d == nil {
		return adminOpsDeploySummary{}
	}
	return adminOpsDeploySummary{
		ID:        d.ID,
		Status:    string(d.Status),
		Source:    d.Source,
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
		StartedAt: func() string {
			if d.StartedAt != nil {
				return d.StartedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		EndedAt: func() string {
			if d.EndedAt != nil {
				return d.EndedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		BuildNumber:      d.BuildNumber,
		BuildDurationMS:  deployBuildDurationMS(d),
		CommitSHA:        d.CommitSHA,
		CommitMessage:    d.CommitMessage,
		DeployedBy:       d.DeployedBy,
		ImageTag:         d.ImageTag,
		PreviousImageTag: d.PrevImage,
	}
}

func analyticsWindowHost(host string, hostPort int) []string {
	candidates := []string{}
	appendIf := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, existing := range candidates {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		candidates = append(candidates, v)
	}
	appendIf(host)
	if hostPort > 0 {
		appendIf(fmt.Sprintf("127.0.0.1:%d", hostPort))
		appendIf(fmt.Sprintf("localhost:%d", hostPort))
	}
	return candidates
}

func analyticsWindowStats(adb *sql.DB, hosts []string, from int64, to int64) (*adminOpsTrafficWindow, bool) {
	if adb == nil || len(hosts) == 0 || to <= from {
		return nil, false
	}
	placeholders := make([]string, len(hosts))
	args := make([]any, 0, len(hosts)+2)
	args = append(args, from, to)
	for i, host := range hosts {
		placeholders[i] = "?"
		args = append(args, host)
	}
	row := adb.QueryRow(
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN status>=500 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status>=400 AND status<500 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(bytes),0)
		   FROM analytics_events
		  WHERE ts>=? AND ts<? AND host IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	var reqs, serverErrors, clientErrors, bytesSent int64
	if err := row.Scan(&reqs, &serverErrors, &clientErrors, &bytesSent); err != nil {
		return nil, false
	}
	window := &adminOpsTrafficWindow{
		Requests:      reqs,
		ServerErrors:  serverErrors,
		ClientErrors:  clientErrors,
		Bandwidth:     bytesSent,
		ServerErrRate: 0,
	}
	if reqs > 0 {
		window.ServerErrRate = float64(serverErrors) / float64(reqs) * 100
	}
	return window, true
}

func (s *Server) latestSuccessfulDeployPair(app string, env DeployEnv, branch string) (*Deploy, *Deploy, error) {
	rows, err := s.db.Query(
		`SELECT d.id, d.app, d.repo_url, d.branch, d.commit_sha, d.env, d.status, d.created_at, d.started_at, d.ended_at, d.error, d.log_path, d.image_tag, d.previous_image_tag, COALESCE(d.preview_url,''), COALESCE(r.source,''), COALESCE(d.build_number,0), COALESCE(d.deployed_by,''), COALESCE(d.commit_message,'')
		   FROM deploys d
		   LEFT JOIN deploy_requests r ON r.id=d.id
		  WHERE d.app=? AND d.env=? AND d.branch=? AND d.status=?
		  ORDER BY d.created_at DESC
		  LIMIT 2`,
		app, string(env), branch, string(StatusSuccess),
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []*Deploy{}
	for rows.Next() {
		d, scanErr := scanDeployRow(rows)
		if scanErr == nil {
			items = append(items, d)
		}
	}
	if len(items) == 0 {
		return nil, nil, nil
	}
	if len(items) == 1 {
		return items[0], nil, nil
	}
	return items[0], items[1], nil
}

func (s *Server) buildAdminOpsResponse() adminOpsResponse {
	resp := adminOpsResponse{
		GeneratedAt: time.Now().UnixMilli(),
		Apps:        []adminOpsApp{},
		Daemon:      collectDaemonStats(),
	}
	rows, err := s.db.Query(`SELECT app, env, branch FROM app_state ORDER BY app, env, branch`)
	if err != nil {
		return resp
	}
	defer rows.Close()

	type laneRef struct {
		App    string
		Env    DeployEnv
		Branch string
	}
	laneRefs := []laneRef{}
	for rows.Next() {
		var lane laneRef
		var env string
		if scanErr := rows.Scan(&lane.App, &env, &lane.Branch); scanErr == nil {
			lane.Env = DeployEnv(env)
			laneRefs = append(laneRefs, lane)
		}
	}

	type laneBundle struct {
		state    *AppState
		targets  []RuntimeLogTarget
		laneData RuntimeLogLaneState
		services []ProjectService
	}
	bundles := map[string]laneBundle{}
	containerNames := []string{}
	seenContainers := map[string]struct{}{}
	for _, ref := range laneRefs {
		st, err := s.getAppState(ref.App, ref.Env, ref.Branch)
		if err != nil || st == nil {
			continue
		}
		targets, laneData, _ := s.runtimeLogTargets(ref.App, ref.Env, ref.Branch)
		services, _ := s.getProjectServices(ref.App, string(ref.Env), ref.Branch)
		key := ref.App + "__" + string(ref.Env) + "__" + ref.Branch
		bundles[key] = laneBundle{state: st, targets: targets, laneData: laneData, services: services}
		for _, target := range targets {
			if target.Container == "" {
				continue
			}
			if _, ok := seenContainers[target.Container]; ok {
				continue
			}
			seenContainers[target.Container] = struct{}{}
			containerNames = append(containerNames, target.Container)
		}
	}

	statsByContainer := collectDockerStats(containerNames)
	storageByContainer := collectDockerStorage(containerNames)

	apps := map[string]*adminOpsApp{}
	adb := s.analyticsStore()
	nowSec := time.Now().Unix()
	for _, ref := range laneRefs {
		key := ref.App + "__" + string(ref.Env) + "__" + ref.Branch
		bundle, ok := bundles[key]
		if !ok || bundle.state == nil {
			continue
		}
		if _, ok := apps[ref.App]; !ok {
			apps[ref.App] = &adminOpsApp{
				App:   ref.App,
				Usage: adminOpsLaneUsage{Targets: []adminOpsContainerUsage{}},
				Lanes: []adminOpsLane{},
			}
		}
		appEntry := apps[ref.App]

		laneUsage := adminOpsLaneUsage{Targets: []adminOpsContainerUsage{}}
		if firstNonEmptyEngine(bundle.state.Engine) != EngineDocker {
			laneUsage.Note = "Live resource sampling currently supports Docker lanes."
		}
		for _, target := range bundle.targets {
			stat := statsByContainer[target.Container]
			size := storageByContainer[target.Container]
			targetUsage := adminOpsContainerUsage{
				ID:            target.ID,
				Label:         target.Label,
				Kind:          target.Kind,
				Container:     target.Container,
				Running:       target.Running,
				CPUPercent:    stat.CPUPercent,
				MemUsageBytes: stat.MemUsage,
				MemLimitBytes: stat.MemLimit,
				MemPercent:    stat.MemPercent,
				StorageBytes:  size.SizeRW + size.SizeRootFS,
				NetRxBytes:    stat.NetRx,
				NetTxBytes:    stat.NetTx,
				BlockRead:     stat.BlockRead,
				BlockWrite:    stat.BlockWrite,
			}
			laneUsage.Targets = append(laneUsage.Targets, targetUsage)
			laneUsage.ContainerCount++
			if target.Running {
				laneUsage.RunningContainers++
			}
			laneUsage.CPUPercent += targetUsage.CPUPercent
			laneUsage.MemUsageBytes += targetUsage.MemUsageBytes
			laneUsage.MemLimitBytes += targetUsage.MemLimitBytes
			laneUsage.StorageBytes += targetUsage.StorageBytes
			laneUsage.NetRxBytes += targetUsage.NetRxBytes
			laneUsage.NetTxBytes += targetUsage.NetTxBytes
			laneUsage.BlockReadBytes += targetUsage.BlockRead
			laneUsage.BlockWriteBytes += targetUsage.BlockWrite
			if target.Running && (targetUsage.CPUPercent > 0 || targetUsage.MemUsageBytes > 0 || targetUsage.StorageBytes > 0) {
				laneUsage.Measured = true
			}
		}
		if laneUsage.MemLimitBytes > 0 {
			laneUsage.MemPercent = float64(laneUsage.MemUsageBytes) / float64(laneUsage.MemLimitBytes) * 100
		}

		var latestDelta *adminOpsDeployDelta
		latest, previous, _ := s.latestSuccessfulDeployPair(ref.App, ref.Env, ref.Branch)
		if latest != nil {
			delta := &adminOpsDeployDelta{
				Current:            toAdminDeploySummary(latest),
				AnalyticsAvailable: false,
			}
			if previous != nil {
				prevSummary := toAdminDeploySummary(previous)
				delta.Previous = &prevSummary
				delta.BuildDurationDeltaMS = delta.Current.BuildDurationMS - prevSummary.BuildDurationMS
				latestAnchor := latest.CreatedAt.Unix()
				if latest.EndedAt != nil {
					latestAnchor = latest.EndedAt.Unix()
				}
				prevAnchor := previous.CreatedAt.Unix()
				if previous.EndedAt != nil {
					prevAnchor = previous.EndedAt.Unix()
				}
				windowSec := int64(0)
				if latestAnchor > prevAnchor {
					windowSec = latestAnchor - prevAnchor
				}
				if live := nowSec - latestAnchor; live > 0 && (windowSec == 0 || live < windowSec) {
					windowSec = live
				}
				if windowSec > 7*24*3600 {
					windowSec = 7 * 24 * 3600
				}
				if windowSec >= 300 {
					delta.Window.Seconds = windowSec
					hosts := analyticsWindowHost(bundle.state.PublicHost, bundle.state.HostPort)
					currentTraffic, currentOK := analyticsWindowStats(adb, hosts, latestAnchor, latestAnchor+windowSec)
					previousTraffic, prevOK := analyticsWindowStats(adb, hosts, prevAnchor, prevAnchor+windowSec)
					if currentOK && prevOK {
						delta.CurrentTraffic = currentTraffic
						delta.PreviousTraffic = previousTraffic
						delta.ServerErrRateDelta = currentTraffic.ServerErrRate - previousTraffic.ServerErrRate
						delta.RequestDelta = currentTraffic.Requests - previousTraffic.Requests
						delta.BandwidthDelta = currentTraffic.Bandwidth - previousTraffic.Bandwidth
						delta.AnalyticsAvailable = true
					} else {
						delta.AnalyticsNote = "Traffic comparison is only available for lanes with proxy analytics history."
					}
				} else {
					delta.AnalyticsNote = "Relay needs more post-deploy traffic time before it can compare this lane."
				}
			} else {
				delta.AnalyticsNote = "No previous successful deploy exists for this lane yet."
			}
			latestDelta = delta
		}

		lane := adminOpsLane{
			App:           ref.App,
			Env:           string(ref.Env),
			Branch:        ref.Branch,
			Engine:        firstNonEmptyEngine(bundle.state.Engine),
			PublicHost:    bundle.state.PublicHost,
			HostPort:      bundle.state.HostPort,
			RepoURL:       bundle.state.RepoURL,
			Stopped:       bundle.state.Stopped,
			CurrentImage:  bundle.state.CurrentImage,
			PreviousImage: bundle.state.PreviousImage,
			Usage:         laneUsage,
			Latest:        latestDelta,
		}
		appEntry.Lanes = append(appEntry.Lanes, lane)
		appEntry.LaneCount++
		if laneUsage.RunningContainers > 0 && !lane.Stopped {
			appEntry.OnlineLanes++
		}
		appEntry.Usage.CPUPercent += laneUsage.CPUPercent
		appEntry.Usage.MemUsageBytes += laneUsage.MemUsageBytes
		appEntry.Usage.MemLimitBytes += laneUsage.MemLimitBytes
		appEntry.Usage.StorageBytes += laneUsage.StorageBytes
		appEntry.Usage.RunningContainers += laneUsage.RunningContainers
		appEntry.Usage.ContainerCount += laneUsage.ContainerCount
		if laneUsage.Measured {
			appEntry.Usage.Measured = true
		}
	}

	appNames := make([]string, 0, len(apps))
	for name := range apps {
		appNames = append(appNames, name)
	}
	sort.Strings(appNames)
	for _, name := range appNames {
		app := apps[name]
		sort.Slice(app.Lanes, func(i, j int) bool {
			if app.Lanes[i].Env == app.Lanes[j].Env {
				return app.Lanes[i].Branch < app.Lanes[j].Branch
			}
			return app.Lanes[i].Env < app.Lanes[j].Env
		})
		if app.Usage.MemLimitBytes > 0 {
			app.Usage.MemPercent = float64(app.Usage.MemUsageBytes) / float64(app.Usage.MemLimitBytes) * 100
		}
		resp.Apps = append(resp.Apps, *app)
		resp.Summary.AppCount++
		resp.Summary.LaneCount += app.LaneCount
		resp.Summary.OnlineLanes += app.OnlineLanes
		resp.Summary.CPUPercent += app.Usage.CPUPercent
		resp.Summary.MemUsageBytes += app.Usage.MemUsageBytes
		resp.Summary.MemLimitBytes += app.Usage.MemLimitBytes
		resp.Summary.StorageBytes += app.Usage.StorageBytes
	}
	if resp.Summary.MemLimitBytes > 0 {
		resp.Summary.MemPercent = float64(resp.Summary.MemUsageBytes) / float64(resp.Summary.MemLimitBytes) * 100
	}
	return resp
}

func (s *Server) handleAdminOps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, 405, "method not allowed")
		return
	}
	writeJSON(w, 200, s.buildAdminOpsResponse())
}
