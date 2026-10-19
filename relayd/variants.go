package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Production variants (experiments). A lane's normal blue/green release is the
// "control"; a variant is an extra build of the same lane that serves a
// weighted share of visitors at the same time. Assignment happens in the
// lane's edge proxy: a visitor's first request is bucketed by a hash of
// client IP + user agent (so every parallel asset request of that first page
// load lands on the same build), and the result is pinned in a cookie so the
// visitor keeps seeing one version. Variants are independent of lanes and of
// blue/green: deploying the control does not touch them, and promoting a
// variant just redeploys its image as the new control.

const maxLaneVariants = 8

var variantNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,23}$`)

var reservedVariantNames = map[string]bool{
	"control": true, "live": true, "active": true, "new": true, "old": true,
	"blue": true, "green": true, "standby": true, "previous": true, "draining": true,
}

type laneVariant struct {
	App         string  `json:"app"`
	Env         string  `json:"env"`
	Branch      string  `json:"branch"`
	Name        string  `json:"name"`
	Image       string  `json:"image"`
	Weight      float64 `json:"weight"`
	Status      string  `json:"status"`
	DeployID    string  `json:"deploy_id,omitempty"`
	CommitSHA   string  `json:"commit_sha,omitempty"`
	Description string  `json:"description,omitempty"`
	Generation  int     `json:"generation"`
	Replicas    int     `json:"replicas"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

func normalizeVariantName(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if !variantNamePattern.MatchString(name) {
		return "", fmt.Errorf("variant name must be 1-24 chars of a-z, 0-9 or '-'")
	}
	if reservedVariantNames[name] {
		return "", fmt.Errorf("%q is reserved; pick another variant name", name)
	}
	return name, nil
}

func normalizeVariantWeight(w float64) (float64, error) {
	if w < 0 || w > 100 {
		return 0, fmt.Errorf("variant weight must be between 0 and 100 (percent of visitors)")
	}
	// nginx split_clients accepts at most two decimals.
	return float64(int64(w*100+0.5)) / 100, nil
}

func (s *Server) listLaneVariants(app string, env DeployEnv, branch string) ([]laneVariant, error) {
	if s == nil || s.db == nil {
		return []laneVariant{}, nil
	}
	rows, err := s.db.Query(
		`SELECT app, env, branch, name, image, weight, status, deploy_id, commit_sha, description, generation, replicas, created_at, updated_at
		   FROM lane_variants WHERE app=? AND env=? AND branch=? ORDER BY created_at, name`,
		app, string(env), branch,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []laneVariant{}
	for rows.Next() {
		var v laneVariant
		if err := rows.Scan(&v.App, &v.Env, &v.Branch, &v.Name, &v.Image, &v.Weight, &v.Status, &v.DeployID, &v.CommitSHA, &v.Description, &v.Generation, &v.Replicas, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Server) getLaneVariant(app string, env DeployEnv, branch string, name string) (*laneVariant, error) {
	variants, err := s.listLaneVariants(app, env, branch)
	if err != nil {
		return nil, err
	}
	for i := range variants {
		if variants[i].Name == name {
			return &variants[i], nil
		}
	}
	return nil, nil
}

func (s *Server) saveLaneVariant(v laneVariant) error {
	now := time.Now().UnixMilli()
	if v.CreatedAt == 0 {
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	_, err := s.db.Exec(
		`INSERT INTO lane_variants (app, env, branch, name, image, weight, status, deploy_id, commit_sha, description, generation, replicas, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(app, env, branch, name) DO UPDATE SET
		   image=excluded.image, weight=excluded.weight, status=excluded.status,
		   deploy_id=excluded.deploy_id, commit_sha=excluded.commit_sha,
		   description=excluded.description, generation=excluded.generation,
		   replicas=excluded.replicas, updated_at=excluded.updated_at`,
		v.App, v.Env, v.Branch, v.Name, v.Image, v.Weight, v.Status, v.DeployID, v.CommitSHA, v.Description, v.Generation, max(v.Replicas, 1), v.CreatedAt, v.UpdatedAt,
	)
	return err
}

func (s *Server) deleteLaneVariant(app string, env DeployEnv, branch string, name string) error {
	_, err := s.db.Exec(`DELETE FROM lane_variants WHERE app=? AND env=? AND branch=? AND name=?`, app, string(env), branch, name)
	return err
}

// variantImages lists every image referenced by any variant, so image pruning
// never deletes the build a live variant is running.
func (s *Server) variantImages(app string, env DeployEnv, branch string) []string {
	if s == nil || s.db == nil {
		return nil
	}
	query := `SELECT image FROM lane_variants WHERE image != ''`
	args := []any{}
	if app != "" {
		query += ` AND app=? AND env=? AND branch=?`
		args = append(args, app, string(env), branch)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var image string
		if rows.Scan(&image) == nil {
			out = append(out, image)
		}
	}
	return out
}

func variantContainerBase(app string, env DeployEnv, branch string, name string, generation int) string {
	return fmt.Sprintf("%s__v-%s-g%d", appBaseContainerName(app, env, branch), name, max(generation, 1))
}

func variantReplicaName(app string, env DeployEnv, branch string, name string, generation int, i int) string {
	base := variantContainerBase(app, env, branch, name, generation)
	if i <= 1 {
		return base
	}
	return base + "__r" + strconv.Itoa(i)
}

func variantPoolAlias(app string, env DeployEnv, branch string, name string, generation int) string {
	return variantContainerBase(app, env, branch, name, generation) + "-pool"
}

func (s *Server) removeVariantGeneration(app string, env DeployEnv, branch string, name string, generation int, replicas int) {
	for i := 1; i <= max(replicas, 1); i++ {
		s.runtime.Remove(variantReplicaName(app, env, branch, name, generation, i))
	}
}

func formatSplitPercent(w float64) string {
	return strings.TrimSuffix(strings.TrimSuffix(strconv.FormatFloat(w, 'f', 2, 64), "0"), ".0") + "%"
}

// edgeVariantRouting is the nginx config a lane needs to route variants.
type edgeVariantRouting struct {
	Active       bool
	HTTPBlock    string // maps inside http {}
	LocationHead string // directives inside `location /`
	Upstream     string // variable to proxy_pass to
}

// buildEdgeVariantRouting renders variant routing for the edge proxy. With no
// variants (or in session traffic mode, where relayd itself picks the
// upstream) it still defines $relay_variant so the access log format and the
// X-Relay-Variant request header always resolve.
func buildEdgeVariantRouting(variants []laneVariant, seed int, servicePort int, trafficMode string, aliasFor func(laneVariant) string) edgeVariantRouting {
	live := []laneVariant{}
	for _, v := range variants {
		if v.Status == "running" {
			live = append(live, v)
		}
	}
	if len(live) == 0 || trafficMode == "session" {
		return edgeVariantRouting{
			HTTPBlock: "  map $host $relay_variant {\n    default \"\";\n  }\n",
			Upstream:  "$relay_upstream",
		}
	}
	sort.Slice(live, func(i, j int) bool { return live[i].Name < live[j].Name })
	seed = max(seed, 1)
	cookie := "relay_v" + strconv.Itoa(seed)
	port := firstNonZero(servicePort, 3000)

	var b strings.Builder
	// Bucket on client IP + user agent rather than $request_id so every
	// parallel request from one browser's first page load (HTML, JS chunks,
	// CSS) picks the same build — mixing builds breaks hashed asset URLs.
	fmt.Fprintf(&b, "  split_clients \"${relay_client_ip}${http_user_agent}#%d\" $relay_variant_assigned {\n", seed)
	for _, v := range live {
		if v.Weight > 0 {
			fmt.Fprintf(&b, "    %s %s;\n", formatSplitPercent(v.Weight), v.Name)
		}
	}
	b.WriteString("    * control;\n  }\n")
	valid := func(header string, variable string) {
		fmt.Fprintf(&b, "  map %s %s {\n    default \"\";\n    control control;\n", header, variable)
		for _, v := range live {
			fmt.Fprintf(&b, "    %s %s;\n", v.Name, v.Name)
		}
		b.WriteString("  }\n")
	}
	valid("$cookie_"+cookie, "$relay_variant_cookie")
	valid("$arg___relay_variant", "$relay_variant_query")
	b.WriteString("  map $relay_variant_cookie $relay_variant_sticky {\n    \"\" $relay_variant_assigned;\n    default $relay_variant_cookie;\n  }\n")
	b.WriteString("  map $relay_variant_query $relay_variant {\n    \"\" $relay_variant_sticky;\n    default $relay_variant_query;\n  }\n")
	fmt.Fprintf(&b, "  map \"$cookie_%s|$relay_variant\" $relay_variant_set_cookie {\n", cookie)
	fmt.Fprintf(&b, "    default \"%s=$relay_variant; Path=/; Max-Age=2592000; SameSite=Lax\";\n", cookie)
	b.WriteString("    \"control|control\" \"\";\n")
	for _, v := range live {
		fmt.Fprintf(&b, "    \"%s|%s\" \"\";\n", v.Name, v.Name)
	}
	b.WriteString("  }\n")
	b.WriteString("  map $relay_variant $relay_final_upstream {\n    default $relay_upstream;\n")
	for _, v := range live {
		fmt.Fprintf(&b, "    %s %s:%d;\n", v.Name, aliasFor(v), port)
	}
	b.WriteString("  }\n")

	return edgeVariantRouting{
		Active:    true,
		HTTPBlock: b.String(),
		LocationHead: "      add_header X-Relay-Variant $relay_variant always;\n" +
			"      add_header Set-Cookie $relay_variant_set_cookie always;\n",
		Upstream: "$relay_final_upstream",
	}
}

func (s *Server) edgeVariantRoutingFor(app string, env DeployEnv, branch string, servicePort int, trafficMode string) edgeVariantRouting {
	variants, _ := s.listLaneVariants(app, env, branch)
	seed := s.getLaneTopology(app, env, branch).VariantSeed
	return buildEdgeVariantRouting(variants, seed, servicePort, trafficMode, func(v laneVariant) string {
		return variantPoolAlias(app, env, branch, v.Name, v.Generation)
	})
}

// reloadLaneEdge re-renders and reloads a running lane's edge proxy so variant
// or replica routing changes take effect without touching app containers.
func (s *Server) reloadLaneEdge(app string, env DeployEnv, branch string) error {
	st, err := s.getAppState(app, env, branch)
	if err != nil || st == nil {
		return fmt.Errorf("lane not found")
	}
	lock := s.edgeProxyLock(app, env, branch)
	lock.Lock()
	defer lock.Unlock()
	return s.refreshLiveLaneTrafficPolicyLocked(st, st)
}

func variantsSupported(st *AppState) error {
	if st == nil {
		return fmt.Errorf("deploy this lane once before adding variants")
	}
	if firstNonEmptyEngine(st.Engine) != EngineDocker {
		return fmt.Errorf("variants require the docker engine")
	}
	if normalizeTrafficMode(st.TrafficMode) == "session" {
		return fmt.Errorf("variants route in the edge proxy; switch this lane's traffic mode from session to edge")
	}
	if st.Stopped || normalizeActiveSlot(st.ActiveSlot) == "" {
		return fmt.Errorf("the lane's control release must be running before a variant can share its traffic")
	}
	return nil
}

// deployVariant starts a new generation of a variant next to the running
// control, waits for every replica to answer, switches the edge to it, and
// only then removes the previous generation — the same zero-downtime ordering
// blue/green uses.
func (s *Server) deployVariant(log func(string, ...any), req DeployRequest, image string, networkName string, extraEnv map[string]string) (*laneVariant, error) {
	name, err := normalizeVariantName(req.Variant)
	if err != nil {
		return nil, err
	}
	st, _ := s.getAppState(req.App, req.Env, req.Branch)
	if err := variantsSupported(st); err != nil {
		return nil, err
	}
	existing, err := s.getLaneVariant(req.App, req.Env, req.Branch, name)
	if err != nil {
		return nil, err
	}
	all, _ := s.listLaneVariants(req.App, req.Env, req.Branch)
	if existing == nil && len(all) >= maxLaneVariants {
		return nil, fmt.Errorf("a lane can run at most %d variants", maxLaneVariants)
	}
	weight := 0.0
	prevGen, prevReplicas := 0, 0
	v := laneVariant{App: req.App, Env: string(req.Env), Branch: req.Branch, Name: name}
	if existing != nil {
		v = *existing
		weight = existing.Weight
		prevGen, prevReplicas = existing.Generation, existing.Replicas
	}
	if req.VariantWeight != nil {
		if weight, err = normalizeVariantWeight(*req.VariantWeight); err != nil {
			return nil, err
		}
	}
	if err := checkVariantWeightBudget(all, name, weight); err != nil {
		return nil, err
	}
	if networkName == "" {
		networkName = appNetworkName(req.App, req.Env, req.Branch)
	}
	servicePort := firstNonZero(st.ServicePort, firstNonZero(req.ServicePort, 3000))
	replicas := s.laneReplicas(req.App, req.Env, req.Branch)
	generation := prevGen + 1
	alias := variantPoolAlias(req.App, req.Env, req.Branch, name, generation)

	base := s.buildAppContainerSpec(s.runtime, req.App, req.Env, req.Branch, variantReplicaName(req.App, req.Env, req.Branch, name, generation, 1), image, servicePort, networkName, extraEnv)
	base.Env = append(base.Env, "RELAY_VARIANT="+name)
	for i := 1; i <= replicas; i++ {
		spec := replicaContainerSpec(base, variantReplicaName(req.App, req.Env, req.Branch, name, generation, i), alias)
		if log != nil {
			log("variant %s: starting %s (replica %d/%d)", name, spec.Name, i, replicas)
		}
		if err := s.runtime.RunDetached(spec); err != nil {
			s.removeVariantGeneration(req.App, req.Env, req.Branch, name, generation, replicas)
			return nil, err
		}
	}
	for i := 1; i <= replicas; i++ {
		cname := variantReplicaName(req.App, req.Env, req.Branch, name, generation, i)
		if err := s.waitForRuntimeContainerReadyOnPath(s.runtime, log, cname, servicePort, s.rolloutReadyTimeoutDuration(), req.ReadinessPath); err != nil {
			s.removeVariantGeneration(req.App, req.Env, req.Branch, name, generation, replicas)
			return nil, fmt.Errorf("variant %s did not become ready: %w", name, err)
		}
	}

	v.Image, v.Weight, v.Status = image, weight, "running"
	v.Generation, v.Replicas = generation, replicas
	v.CommitSHA = req.CommitSHA
	if req.VariantDescription != "" {
		v.Description = req.VariantDescription
	}
	if err := s.saveLaneVariant(v); err != nil {
		s.removeVariantGeneration(req.App, req.Env, req.Branch, name, generation, replicas)
		return nil, err
	}
	if err := s.reloadLaneEdge(req.App, req.Env, req.Branch); err != nil {
		// Roll the row back to the generation that is still routed.
		if existing != nil {
			_ = s.saveLaneVariant(*existing)
		} else {
			_ = s.deleteLaneVariant(req.App, req.Env, req.Branch, name)
		}
		s.removeVariantGeneration(req.App, req.Env, req.Branch, name, generation, replicas)
		return nil, fmt.Errorf("edge proxy reload failed: %w", err)
	}
	if prevGen > 0 {
		// nginx -s reload lets in-flight requests finish on old workers; give
		// them a moment before the containers under them disappear.
		time.Sleep(2 * time.Second)
		s.removeVariantGeneration(req.App, req.Env, req.Branch, name, prevGen, prevReplicas)
	}
	if log != nil {
		log("variant %s live: %s of new visitors (sticky per visitor); preview with ?__relay_variant=%s", name, formatSplitPercent(weight), name)
	}
	return &v, nil
}

func checkVariantWeightBudget(all []laneVariant, name string, weight float64) error {
	total := weight
	for _, other := range all {
		if other.Name != name && other.Status == "running" {
			total += other.Weight
		}
	}
	if total > 100.0001 {
		return fmt.Errorf("variant weights add up to %.2f%%; they must leave room for the control (max 100%%)", total)
	}
	return nil
}

func (s *Server) setVariantWeight(app string, env DeployEnv, branch string, name string, weight float64) (*laneVariant, error) {
	v, err := s.getLaneVariant(app, env, branch, name)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, fmt.Errorf("variant %q not found", name)
	}
	if weight, err = normalizeVariantWeight(weight); err != nil {
		return nil, err
	}
	all, _ := s.listLaneVariants(app, env, branch)
	if err := checkVariantWeightBudget(all, name, weight); err != nil {
		return nil, err
	}
	previous := *v
	v.Weight = weight
	if err := s.saveLaneVariant(*v); err != nil {
		return nil, err
	}
	if err := s.reloadLaneEdge(app, env, branch); err != nil {
		_ = s.saveLaneVariant(previous)
		return nil, err
	}
	return v, nil
}

// removeVariant stops routing to a variant first and only then deletes its
// containers; visitors pinned to it fall back to the control on their next
// request (their cookie no longer matches a valid variant).
func (s *Server) removeVariant(app string, env DeployEnv, branch string, name string) error {
	v, err := s.getLaneVariant(app, env, branch, name)
	if err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("variant %q not found", name)
	}
	if err := s.deleteLaneVariant(app, env, branch, name); err != nil {
		return err
	}
	if err := s.reloadLaneEdge(app, env, branch); err != nil {
		_ = s.saveLaneVariant(*v)
		return err
	}
	time.Sleep(2 * time.Second)
	s.removeVariantGeneration(app, env, branch, name, v.Generation, v.Replicas)
	return nil
}

// reshuffleVariants changes the cookie name and bucketing seed, so every
// visitor is reassigned under the current weights.
func (s *Server) reshuffleVariants(app string, env DeployEnv, branch string) error {
	topo := s.getLaneTopology(app, env, branch)
	topo.VariantSeed++
	if err := s.saveLaneTopology(app, env, branch, topo); err != nil {
		return err
	}
	return s.reloadLaneEdge(app, env, branch)
}

// ─── Feedback ────────────────────────────────────────────────────────────────

var feedbackRate = struct {
	sync.Mutex
	hits map[string][]time.Time
}{hits: map[string][]time.Time{}}

func feedbackAllowed(key string) bool {
	feedbackRate.Lock()
	defer feedbackRate.Unlock()
	if len(feedbackRate.hits) > 4096 {
		feedbackRate.hits = map[string][]time.Time{}
	}
	return slidingWindowCount(feedbackRate.hits, key, time.Minute) < 10 && func() bool {
		feedbackRate.hits[key] = append(feedbackRate.hits[key], time.Now())
		return true
	}()
}

// handleEdgeFeedback stores visitor feedback posted to /__relay/feedback on
// any app. The lane's edge proxy forwards it here with the lane identity, the
// visitor's assigned variant and real client IP, authenticated by the lane's
// edge token — so a visitor can't attribute feedback to another variant.
func (s *Server) handleEdgeFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, 405, "method not allowed")
		return
	}
	app := r.Header.Get("X-Relay-Lane-App")
	env := normalizeDeployEnv(r.Header.Get("X-Relay-Lane-Env"))
	branch := r.Header.Get("X-Relay-Lane-Branch")
	if !validDeployTarget(app, env, branch) {
		httpError(w, 400, "invalid lane")
		return
	}
	token, err := s.edgeSessionToken(app, env, branch)
	if err != nil || !hmac.Equal([]byte(token), []byte(r.Header.Get("X-Relay-Edge-Token"))) {
		httpError(w, 401, "unauthorized edge")
		return
	}
	ip := strings.TrimSpace(r.Header.Get("X-Relay-Client-IP"))
	if !feedbackAllowed(app + "|" + string(env) + "|" + branch + "|" + ip) {
		httpError(w, 429, "too much feedback; try again in a minute")
		return
	}
	var body struct {
		Vote    string `json:"vote"`   // "up" | "down"
		Rating  int    `json:"rating"` // 1-5 stars
		Comment string `json:"comment"`
		Page    string `json:"page"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&body); err != nil {
		httpError(w, 400, "invalid json")
		return
	}
	vote := 0
	switch strings.ToLower(strings.TrimSpace(body.Vote)) {
	case "up", "like", "+1":
		vote = 1
	case "down", "dislike", "-1":
		vote = -1
	case "":
	default:
		httpError(w, 400, `vote must be "up" or "down"`)
		return
	}
	if body.Rating < 0 || body.Rating > 5 {
		httpError(w, 400, "rating must be 1-5")
		return
	}
	comment := truncateRunes(strings.TrimSpace(body.Comment), 2000)
	if vote == 0 && body.Rating == 0 && comment == "" {
		httpError(w, 400, "send a vote, a rating or a comment")
		return
	}
	variant := strings.TrimSpace(r.Header.Get("X-Relay-Variant"))
	if variant == "" {
		variant = "control"
	}
	_, err = s.analyticsStore().Exec(
		`INSERT INTO variant_feedback (ts, app, env, branch, variant, vote, rating, comment, page, visitor) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UnixMilli(), app, string(env), branch, variant, vote, body.Rating, comment,
		truncateRunes(body.Page, 256), s.visitorHash(ip, r.Header.Get("User-Agent"), time.Now()),
	)
	if err != nil {
		httpError(w, 500, "failed to store feedback")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "variant": variant})
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// visitorHash is a cookieless, daily-rotating visitor id (the same approach
// privacy-friendly analytics use): it counts unique visitors per day without
// storing a cross-day identifier or the raw IP.
func (s *Server) visitorHash(ip string, ua string, at time.Time) string {
	if strings.TrimSpace(ip) == "" {
		return ""
	}
	key := s.secretKey
	if len(key) == 0 {
		key = []byte("relay-analytics")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = io.WriteString(mac, at.UTC().Format("2006-01-02")+"\x00"+ip+"\x00"+ua)
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

// ─── API ─────────────────────────────────────────────────────────────────────

type variantMetrics struct {
	Requests     int64   `json:"requests"`
	ServerErrors int64   `json:"server_errors"`
	ErrorRate    float64 `json:"error_rate"`
	PageViews    int64   `json:"page_views"`
	Visitors     int64   `json:"visitors"`
	P50MS        int64   `json:"p50_ms"`
	P95MS        int64   `json:"p95_ms"`
	Feedback     int64   `json:"feedback"`
	ThumbsUp     int64   `json:"thumbs_up"`
	ThumbsDown   int64   `json:"thumbs_down"`
	AvgRating    float64 `json:"avg_rating"`
}

type variantFeedbackItem struct {
	TS      int64  `json:"ts"`
	Variant string `json:"variant"`
	Vote    int    `json:"vote"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
	Page    string `json:"page"`
}

func (s *Server) variantMetricsFor(app string, env DeployEnv, branch string, variant string, since int64) variantMetrics {
	adb := s.analyticsStore()
	var m variantMetrics
	match := variant
	if variant == "control" {
		match = ""
	}
	where := `app=? AND env=? AND branch=? AND ts>=? AND (variant=? OR (?='control' AND variant='control'))`
	args := []any{app, string(env), branch, since / 1000, match, variant}
	_ = adb.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status>=500 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN kind='page' AND status<400 THEN 1 ELSE 0 END),0),
		        COUNT(DISTINCT CASE WHEN kind='page' THEN NULLIF(visitor,'') END)
		   FROM analytics_events WHERE `+where, args...,
	).Scan(&m.Requests, &m.ServerErrors, &m.PageViews, &m.Visitors)
	if m.Requests > 0 {
		m.ErrorRate = float64(m.ServerErrors) / float64(m.Requests) * 100
		m.P50MS = durationPercentile(adb, where, args, m.Requests, 0.50)
		m.P95MS = durationPercentile(adb, where, args, m.Requests, 0.95)
	}
	var avg sql.NullFloat64
	_ = adb.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN vote=1 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN vote=-1 THEN 1 ELSE 0 END),0),
		        AVG(CASE WHEN rating>=1 THEN rating END)
		   FROM variant_feedback WHERE app=? AND env=? AND branch=? AND variant=? AND ts>=?`,
		app, string(env), branch, variant, since,
	).Scan(&m.Feedback, &m.ThumbsUp, &m.ThumbsDown, &avg)
	if avg.Valid {
		m.AvgRating = avg.Float64
	}
	return m
}

// durationPercentile returns the p-th percentile of duration_ms over the
// rows matched by where (SQLite has no percentile function).
func durationPercentile(adb *sql.DB, where string, args []any, count int64, p float64) int64 {
	if count <= 0 {
		return 0
	}
	offset := int64(float64(count-1) * p)
	var v int64
	_ = adb.QueryRow(
		`SELECT duration_ms FROM analytics_events WHERE `+where+` ORDER BY duration_ms LIMIT 1 OFFSET ?`,
		append(append([]any{}, args...), offset)...,
	).Scan(&v)
	return v
}

func (s *Server) handleVariants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	app := strings.TrimSpace(q.Get("app"))
	env := normalizeDeployEnv(q.Get("env"))
	branch := firstNonEmpty(strings.TrimSpace(q.Get("branch")), "main")
	if r.Method != http.MethodGet {
		var body struct {
			App    string   `json:"app"`
			Env    string   `json:"env"`
			Branch string   `json:"branch"`
			Action string   `json:"action"`
			Name   string   `json:"name"`
			Weight *float64 `json:"weight"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		app, env, branch = strings.TrimSpace(body.App), normalizeDeployEnv(body.Env), firstNonEmpty(strings.TrimSpace(body.Branch), "main")
		if !validDeployTarget(app, env, branch) {
			httpError(w, 400, "app, env and branch required")
			return
		}
		if _, ok := s.requireLaneAccess(w, r, app, env, "deployer"); !ok {
			return
		}
		actor := requestActorLabel(s, r)
		name := strings.ToLower(strings.TrimSpace(body.Name))
		switch body.Action {
		case "weight":
			if body.Weight == nil {
				httpError(w, 400, "weight required")
				return
			}
			v, err := s.setVariantWeight(app, env, branch, name, *body.Weight)
			if err != nil {
				httpError(w, 400, err.Error())
				return
			}
			s.auditLog(actor, "variant.weight", app, fmt.Sprintf("env=%s branch=%s variant=%s weight=%.2f", env, branch, name, v.Weight))
		case "remove":
			if err := s.removeVariant(app, env, branch, name); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			s.auditLog(actor, "variant.remove", app, fmt.Sprintf("env=%s branch=%s variant=%s", env, branch, name))
		case "promote":
			deployID, err := s.promoteVariant(app, env, branch, name, actor)
			if err != nil {
				httpError(w, 400, err.Error())
				return
			}
			s.broadcastSnapshot()
			writeJSON(w, 200, map[string]any{"ok": true, "deploy_id": deployID})
			return
		case "reshuffle":
			if err := s.reshuffleVariants(app, env, branch); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			s.auditLog(actor, "variant.reshuffle", app, fmt.Sprintf("env=%s branch=%s", env, branch))
		default:
			httpError(w, 400, "action must be weight, remove, promote or reshuffle")
			return
		}
		s.broadcastSnapshot()
	}
	if !validDeployTarget(app, env, branch) {
		httpError(w, 400, "app, env and branch required")
		return
	}
	if r.Method == http.MethodGet {
		if _, ok := s.requireLaneAccess(w, r, app, env, "viewer"); !ok {
			return
		}
	}
	writeJSON(w, 200, s.variantsReport(app, env, branch, q.Get("period")))
}

type variantReportItem struct {
	laneVariant
	Metrics variantMetrics `json:"metrics"`
}

func (s *Server) variantsReport(app string, env DeployEnv, branch string, period string) map[string]any {
	since := time.Now().Add(-7 * 24 * time.Hour)
	switch period {
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	}
	variants, _ := s.listLaneVariants(app, env, branch)
	st, _ := s.getAppState(app, env, branch)
	controlWeight := 100.0
	items := []variantReportItem{}
	for _, v := range variants {
		if v.Status == "running" {
			controlWeight -= v.Weight
		}
		items = append(items, variantReportItem{laneVariant: v, Metrics: s.variantMetricsFor(app, env, branch, v.Name, since.UnixMilli())})
	}
	control := map[string]any{
		"name":    "control",
		"weight":  max(controlWeight, 0),
		"metrics": s.variantMetricsFor(app, env, branch, "control", since.UnixMilli()),
	}
	if st != nil {
		control["image"] = st.CurrentImage
	}
	feedback := []variantFeedbackItem{}
	rows, err := s.analyticsStore().Query(
		`SELECT ts, variant, vote, rating, comment, page FROM variant_feedback
		  WHERE app=? AND env=? AND branch=? AND ts>=? ORDER BY ts DESC LIMIT 50`,
		app, string(env), branch, since.UnixMilli(),
	)
	if err == nil {
		for rows.Next() {
			var f variantFeedbackItem
			if rows.Scan(&f.TS, &f.Variant, &f.Vote, &f.Rating, &f.Comment, &f.Page) == nil {
				feedback = append(feedback, f)
			}
		}
		rows.Close()
	}
	supportErr := ""
	if err := variantsSupported(st); err != nil {
		supportErr = err.Error()
	}
	return map[string]any{
		"app": app, "env": env, "branch": branch,
		"control":     control,
		"variants":    items,
		"feedback":    feedback,
		"supported":   supportErr == "",
		"unsupported": supportErr,
		"seed":        s.getLaneTopology(app, env, branch).VariantSeed,
	}
}

// promoteVariant makes a variant the lane's new control by redeploying its
// image through the normal blue/green path, then retires the variant once
// that deploy is queued (its visitors fall back to the control, which now
// runs the same build).
func (s *Server) promoteVariant(app string, env DeployEnv, branch string, name string, actor string) (string, error) {
	v, err := s.getLaneVariant(app, env, branch, name)
	if err != nil {
		return "", err
	}
	if v == nil || v.Image == "" {
		return "", fmt.Errorf("variant %q not found", name)
	}
	deployID, err := s.enqueueImageRedeploy(app, env, branch, v.Image, actor, "variant-promote", false)
	if err != nil {
		return "", err
	}
	s.auditLog(actor, "variant.promote", app, fmt.Sprintf("env=%s branch=%s variant=%s deploy=%s", env, branch, name, deployID))
	go s.retireVariantAfterDeploy(app, env, branch, name, deployID)
	return deployID, nil
}

func (s *Server) retireVariantAfterDeploy(app string, env DeployEnv, branch string, name string, deployID string) {
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		d, err := s.getDeployFromDB(deployID)
		if err != nil || d == nil {
			continue
		}
		switch d.Status {
		case StatusSuccess:
			_ = s.removeVariant(app, env, branch, name)
			s.broadcastSnapshot()
			return
		case StatusFailed:
			return
		}
	}
}

// enqueueImageRedeploy queues a blue/green deploy of an existing image onto a
// lane (used to promote variants and to roll out a replica count change).
func (s *Server) enqueueImageRedeploy(app string, env DeployEnv, branch string, image string, actor string, source string, preserveHistory bool) (string, error) {
	st, err := s.getAppState(app, env, branch)
	if err != nil || st == nil {
		return "", fmt.Errorf("lane not found")
	}
	if strings.TrimSpace(image) == "" {
		return "", fmt.Errorf("lane has no deployed image")
	}
	deployID := newID()
	deploy := &Deploy{
		ID:        deployID,
		App:       app,
		Branch:    branch,
		Env:       env,
		Status:    StatusQueued,
		CreatedAt: time.Now(),
		LogPath:   filepath.Join(s.logsDir, deployID+".log"),
	}
	s.mu.Lock()
	s.deploys[deployID] = deploy
	s.mu.Unlock()
	req := DeployRequest{
		App:              app,
		Branch:           branch,
		Env:              env,
		Mode:             st.Mode,
		TrafficMode:      st.TrafficMode,
		HostPort:         st.HostPort,
		HostPortExplicit: st.HostPortExplicit,
		ServicePort:      st.ServicePort,
		PublicHost:       st.PublicHost,
		Source:           source,
		DeployedBy:       actor,
	}
	if err := s.saveDeployToDB(deploy, req); err != nil {
		return "", err
	}
	s.queue <- DeployJob{ID: deployID, Req: req, Rollback: true, RollbackImage: image, PreserveHistory: preserveHistory}
	return deployID, nil
}

// stopLaneVariants removes a lane's variant containers. When the lane is only
// stopped the variants are kept as "stopped" (no longer routed) so their
// history and feedback stay visible; deleting the lane drops them entirely.
func (s *Server) stopLaneVariants(app string, env DeployEnv, branch string, deleteRows bool) {
	variants, err := s.listLaneVariants(app, env, branch)
	if err != nil {
		return
	}
	for _, v := range variants {
		s.removeVariantGeneration(app, env, branch, v.Name, v.Generation, v.Replicas)
		if deleteRows {
			_ = s.deleteLaneVariant(app, env, branch, v.Name)
		} else if v.Status != "stopped" {
			v.Status = "stopped"
			_ = s.saveLaneVariant(v)
		}
	}
	if deleteRows {
		_, _ = s.db.Exec(`DELETE FROM lane_topology WHERE app=? AND env=? AND branch=?`, app, string(env), branch)
	}
}

// validateDeployExtras rejects bad variant/replica options before a build is
// queued, so a typo fails in a second instead of after a full build.
func (s *Server) validateDeployExtras(app string, env DeployEnv, branch string, variant string, weight *float64, replicas int) error {
	if replicas < 0 || replicas > maxLaneReplicas {
		return fmt.Errorf("replicas must be between 1 and %d", maxLaneReplicas)
	}
	if strings.TrimSpace(variant) == "" {
		if weight != nil {
			return fmt.Errorf("--weight needs --variant")
		}
		return nil
	}
	name, err := normalizeVariantName(variant)
	if err != nil {
		return err
	}
	st, _ := s.getAppState(app, env, branch)
	if err := variantsSupported(st); err != nil {
		return err
	}
	if weight != nil {
		w, err := normalizeVariantWeight(*weight)
		if err != nil {
			return err
		}
		all, _ := s.listLaneVariants(app, env, branch)
		if err := checkVariantWeightBudget(all, name, w); err != nil {
			return err
		}
	}
	return nil
}
