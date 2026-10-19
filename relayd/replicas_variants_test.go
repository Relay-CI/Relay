package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

type recordingRuntime struct {
	mockRuntime
	specs []ContainerSpec
}

func (r *recordingRuntime) RunDetached(spec ContainerSpec) error {
	r.specs = append(r.specs, spec)
	r.running[spec.Name] = true
	return nil
}

func (r *recordingRuntime) Remove(name string) {
	r.mockRuntime.Remove(name)
	delete(r.running, name)
}

func newRecordingServer(t *testing.T) (*Server, *recordingRuntime) {
	t.Helper()
	s := newPreviewPortTestServer(t)
	rt := &recordingRuntime{mockRuntime: mockRuntime{running: map[string]bool{}, exists: map[string]bool{}, published: map[string]int{}}}
	s.runtime = rt
	return s, rt
}

func TestSlotLaunchStartsEveryReplicaOnASharedPoolAlias(t *testing.T) {
	s, rt := newRecordingServer(t)
	if err := s.setLaneReplicas("demo", EnvProd, "main", 3); err != nil {
		t.Fatal(err)
	}
	if err := s.runSlotContainer(nil, "demo", EnvProd, "main", "blue", "relay/demo:1", 3000, "net", nil); err != nil {
		t.Fatal(err)
	}
	if len(rt.specs) != 3 {
		t.Fatalf("expected 3 replica containers, got %d", len(rt.specs))
	}
	alias := appSlotPoolAlias("demo", EnvProd, "main", "blue")
	for i, spec := range rt.specs {
		if want := appSlotReplicaName("demo", EnvProd, "main", "blue", i+1); spec.Name != want {
			t.Fatalf("replica %d name = %q, want %q", i+1, spec.Name, want)
		}
		if !slices.Contains(spec.NetworkAliases, alias) {
			t.Fatalf("replica %s missing pool alias %s: %v", spec.Name, alias, spec.NetworkAliases)
		}
	}
	if rt.specs[0].Name != appSlotContainerName("demo", EnvProd, "main", "blue") {
		t.Fatalf("replica 1 must keep the historical slot container name")
	}
	if got := s.slotUpstreamHost("demo", EnvProd, "main", "blue"); got != alias {
		t.Fatalf("edge upstream = %q, want pool alias %q", got, alias)
	}

	s.removeSlotContainers(s.runtime, "demo", EnvProd, "main", "blue")
	for _, spec := range rt.specs {
		if rt.running[spec.Name] {
			t.Fatalf("replica %s still running after slot removal", spec.Name)
		}
	}
	if got := s.slotUpstreamHost("demo", EnvProd, "main", "blue"); got != appSlotContainerName("demo", EnvProd, "main", "blue") {
		t.Fatalf("removed slot should fall back to its container name, got %q", got)
	}
}

func TestLegacySlotWithoutPoolKeepsContainerNameUpstream(t *testing.T) {
	s, _ := newRecordingServer(t)
	if got := s.slotUpstreamHost("demo", EnvProd, "main", "green"); got != appSlotContainerName("demo", EnvProd, "main", "green") {
		t.Fatalf("a slot launched before replicas existed must be addressed by name, got %q", got)
	}
}

func TestEdgeConfigTargetsReplicaPoolAndRetriesAcrossReplicas(t *testing.T) {
	s, _ := newRecordingServer(t)
	_ = s.setLaneReplicas("demo", EnvProd, "main", 2)
	if err := s.runSlotContainer(nil, "demo", EnvProd, "main", "blue", "relay/demo:1", 3000, "net", nil); err != nil {
		t.Fatal(err)
	}
	path, err := s.writeEdgeProxyConfig("demo", EnvProd, "main", "blue", "", 3000, "edge", 100)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	for _, want := range []string{
		appSlotPoolAlias("demo", EnvProd, "main", "blue") + ":3000",
		"proxy_next_upstream error timeout http_502 http_503 http_504;",
		"proxy_set_header X-Relay-Variant $relay_variant;",
		"map $http_cf_connecting_ip $relay_client_ip",
		`"app":"demo","env":"prod","branch":"main"`,
		"location = /__relay/feedback",
	} {
		if !strings.Contains(text, want) && !(want == "location = /__relay/feedback" && listenAddrPort(s.httpAddr) == 0) {
			t.Fatalf("edge config missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "split_clients \"${relay_client_ip}") {
		t.Fatalf("no variants configured, but variant bucketing was emitted:\n%s", text)
	}
}

func TestVariantRoutingIsStickyWeightedAndPreviewable(t *testing.T) {
	variants := []laneVariant{
		{Name: "b", Weight: 10, Status: "running", Generation: 2},
		{Name: "c", Weight: 0, Status: "running", Generation: 1},
		{Name: "old", Weight: 5, Status: "stopped", Generation: 1},
	}
	routing := buildEdgeVariantRouting(variants, 3, 3000, "edge", func(v laneVariant) string { return "pool-" + v.Name })
	if !routing.Active || routing.Upstream != "$relay_final_upstream" {
		t.Fatalf("expected active routing via $relay_final_upstream, got %+v", routing)
	}
	text := routing.HTTPBlock
	for _, want := range []string{
		`split_clients "${relay_client_ip}${http_user_agent}#3" $relay_variant_assigned`,
		"10% b;",
		"* control;",
		"map $cookie_relay_v3 $relay_variant_cookie",
		"map $arg___relay_variant $relay_variant_query",
		"b pool-b:3000;",
		"c pool-c:3000;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("variant routing missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "0% c;") {
		t.Fatalf("a 0%% variant must not receive bucketed traffic (preview-only):\n%s", text)
	}
	if strings.Contains(text, "pool-old") {
		t.Fatalf("stopped variants must not be routed:\n%s", text)
	}
	if !strings.Contains(routing.LocationHead, "Set-Cookie $relay_variant_set_cookie") {
		t.Fatalf("variant assignment must be pinned with a cookie: %s", routing.LocationHead)
	}
}

func TestVariantRoutingDisabledInSessionModeAndWithoutVariants(t *testing.T) {
	for _, tc := range []struct {
		name     string
		variants []laneVariant
		mode     string
	}{
		{"none", nil, "edge"},
		{"session", []laneVariant{{Name: "b", Weight: 50, Status: "running"}}, "session"},
	} {
		routing := buildEdgeVariantRouting(tc.variants, 1, 3000, tc.mode, func(v laneVariant) string { return v.Name })
		if routing.Active || routing.Upstream != "$relay_upstream" || !strings.Contains(routing.HTTPBlock, "$relay_variant") {
			t.Fatalf("%s: expected inactive routing that still defines $relay_variant, got %+v", tc.name, routing)
		}
	}
}

func TestVariantNamesAndWeightsAreValidated(t *testing.T) {
	for _, bad := range []string{"", "control", "blue", "Has Space", strings.Repeat("a", 30), "-lead"} {
		if _, err := normalizeVariantName(bad); err == nil {
			t.Fatalf("variant name %q should be rejected", bad)
		}
	}
	if name, err := normalizeVariantName("  New-Checkout "); err != nil || name != "new-checkout" {
		t.Fatalf("normalize = %q, %v", name, err)
	}
	all := []laneVariant{{Name: "b", Weight: 60, Status: "running"}, {Name: "c", Weight: 30, Status: "running"}}
	if err := checkVariantWeightBudget(all, "d", 20); err == nil {
		t.Fatal("weights over 100% must be rejected")
	}
	if err := checkVariantWeightBudget(all, "c", 40); err != nil {
		t.Fatalf("re-weighting c to 40 leaves the total at 100%%: %v", err)
	}
	if w, err := normalizeVariantWeight(12.345); err != nil || w != 12.35 {
		t.Fatalf("weight rounding = %v, %v", w, err)
	}
}

func TestVariantRowsRoundTripAndProtectImagesFromPruning(t *testing.T) {
	s, _ := newRecordingServer(t)
	v := laneVariant{App: "demo", Env: "prod", Branch: "main", Name: "b", Image: "relay/demo:prod-main-abc", Weight: 25, Status: "running", Generation: 1, Replicas: 1}
	if err := s.saveLaneVariant(v); err != nil {
		t.Fatal(err)
	}
	got, err := s.getLaneVariant("demo", EnvProd, "main", "b")
	if err != nil || got == nil || got.Weight != 25 || got.Image != v.Image {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
	if images := s.variantImages("demo", EnvProd, "main"); !slices.Contains(images, v.Image) {
		t.Fatalf("variant image must be protected from pruning, got %v", images)
	}
	s.stopLaneVariants("demo", EnvProd, "main", false)
	if got, _ := s.getLaneVariant("demo", EnvProd, "main", "b"); got == nil || got.Status != "stopped" {
		t.Fatalf("stopping a lane keeps variants as stopped, got %+v", got)
	}
	s.stopLaneVariants("demo", EnvProd, "main", true)
	if got, _ := s.getLaneVariant("demo", EnvProd, "main", "b"); got != nil {
		t.Fatalf("deleting a lane removes its variants, got %+v", got)
	}
}

func TestActiveReplicaCountSumsReplicasAcrossLiveLanes(t *testing.T) {
	s, _ := newRecordingServer(t)
	for _, app := range []string{"a", "b"} {
		if _, err := s.db.Exec(`INSERT INTO app_state (app, env, branch, stopped) VALUES (?, 'prod', 'main', 0)`, app); err != nil {
			t.Fatal(err)
		}
	}
	_ = s.setLaneReplicas("a", EnvProd, "main", 4)
	if got := s.activeReplicaCount(); got != 5 {
		t.Fatalf("activeReplicaCount = %d, want 4 + 1", got)
	}
}

func TestParseEdgeLogLineAttributesLaneVariantAndClassifiesPages(t *testing.T) {
	s, _ := newRecordingServer(t)
	line := `{"ts":1760000000.250,"status":200,"host":"shop.example.com","method":"GET","path":"/pricing?ref=x","slot":"blue","request_id":"r1","app":"shop","env":"prod","branch":"main","variant":"b","ip":"203.0.113.9","cc":"SG","rt":"0.042","bytes":"5120","ua":"Mozilla/5.0","ref":"https://news.example.org/post","accept":"text/html,application/xhtml+xml"}`
	ev, ms, ok := s.parseEdgeLogLine([]byte(line))
	if !ok {
		t.Fatal("expected line to parse")
	}
	if ms != 1760000000250 || ev.TS != 1760000000 {
		t.Fatalf("timestamps = %d/%d", ms, ev.TS)
	}
	if ev.App != "shop" || ev.Variant != "b" || ev.Slot != "blue" || ev.Country != "SG" || ev.DurationMS != 42 || ev.Bytes != 5120 {
		t.Fatalf("unexpected event %+v", ev)
	}
	if ev.Kind != "page" || ev.Visitor == "" || ev.Referrer != "news.example.org" || ev.Source != "edge" {
		t.Fatalf("page classification/visitor/referrer wrong: %+v", ev)
	}
	if _, _, ok := s.parseEdgeLogLine([]byte(`2026/10/19 [error] upstream timed out`)); ok {
		t.Fatal("nginx error lines must be ignored")
	}
}

func TestClassifyRequest(t *testing.T) {
	cases := []struct{ method, path, accept, ua, want string }{
		{"GET", "/", "text/html", "Mozilla/5.0", "page"},
		{"GET", "/_next/static/chunks/app.js", "*/*", "Mozilla/5.0", "asset"},
		{"GET", "/logo.svg", "image/*", "Mozilla/5.0", "asset"},
		{"POST", "/api/cart", "application/json", "Mozilla/5.0", "api"},
		{"GET", "/", "text/html", "Googlebot/2.1", "bot"},
		{"GET", "/__relay/presence.js", "*/*", "Mozilla/5.0", "other"},
	}
	for _, tc := range cases {
		if got := classifyRequest(tc.method, tc.path, tc.accept, tc.ua); got != tc.want {
			t.Fatalf("classify(%s %s) = %s, want %s", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestVisitorHashRotatesDaily(t *testing.T) {
	s, _ := newRecordingServer(t)
	day1 := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	a := s.visitorHash("203.0.113.9", "UA", day1)
	if a == "" || a != s.visitorHash("203.0.113.9", "UA", day1.Add(3*time.Hour)) {
		t.Fatal("same visitor on the same day must hash the same")
	}
	if a == s.visitorHash("203.0.113.9", "UA", day1.AddDate(0, 0, 1)) {
		t.Fatal("visitor ids must not persist across days")
	}
}

func TestDockerRunArgsCarriesPoolAliasAndLogRotation(t *testing.T) {
	args := dockerRunArgs(ContainerSpec{Name: "c", Image: "img", Network: "net", NetworkAliases: []string{"pool"}, LogMaxSize: "10m", LogMaxFile: 3})
	joined := strings.Join(args, " ")
	for _, want := range []string{"--network net --network-alias pool", "--log-opt max-size=10m", "--log-opt max-file=3"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("docker args missing %q: %s", want, joined)
		}
	}
}

func TestCaddyRoutesRetryDialFailures(t *testing.T) {
	block := caddyReverseProxy("relay__demo__prod__main:3000")
	if !strings.Contains(block, "lb_try_duration 5s") || !strings.Contains(block, "reverse_proxy relay__demo__prod__main:3000 {") {
		t.Fatalf("caddy route must retry while an upstream restarts:\n%s", block)
	}
}

type failingNginxRuntime struct {
	recordingRuntime
}

func (r *failingNginxRuntime) Exec(name string, args []string) ([]byte, error) {
	return []byte("nginx: [emerg] simulated"), errSimulatedNginx
}

var errSimulatedNginx = &simulatedErr{"nginx -t failed"}

type simulatedErr struct{ msg string }

func (e *simulatedErr) Error() string { return e.msg }

func seedLiveLane(t *testing.T, s *Server, running map[string]bool) {
	t.Helper()
	if err := s.saveAppState(&AppState{App: "demo", Env: EnvProd, Branch: "main", Engine: EngineDocker, ActiveSlot: "blue", Mode: "port", HostPort: 3100, ServicePort: 3000, TrafficMode: "edge", CurrentImage: "relay/demo:prod-main-1"}); err != nil {
		t.Fatal(err)
	}
	running[appBaseContainerName("demo", EnvProd, "main")] = true
	running[appSlotContainerName("demo", EnvProd, "main", "blue")] = true
}

func TestVariantWeightChangeRollsBackWhenEdgeReloadFails(t *testing.T) {
	s := newPreviewPortTestServer(t)
	rt := &failingNginxRuntime{recordingRuntime{mockRuntime: mockRuntime{running: map[string]bool{}, exists: map[string]bool{}, published: map[string]int{}}}}
	rt.published[appBaseContainerName("demo", EnvProd, "main")] = 3100
	s.runtime = rt
	seedLiveLane(t, s, rt.running)
	if err := s.saveLaneVariant(laneVariant{App: "demo", Env: "prod", Branch: "main", Name: "b", Weight: 20, Status: "running", Generation: 1, Replicas: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.setVariantWeight("demo", EnvProd, "main", "b", 30); err == nil {
		t.Fatal("expected the weight change to fail when the edge proxy can't reload")
	}
	v, _ := s.getLaneVariant("demo", EnvProd, "main", "b")
	if v == nil || v.Weight != 20 {
		t.Fatalf("stored weight must stay at the still-routed 20%%, got %+v", v)
	}
}

func TestScaleHandlerValidatesAndRollsOutWithoutLosingRollbackTarget(t *testing.T) {
	s, rt := newRecordingServer(t)
	s.queue = make(chan DeployJob, 2)
	s.deploys = map[string]*Deploy{}
	s.logsDir = t.TempDir()
	seedLiveLane(t, s, rt.running)

	for _, n := range []int{0, 17} {
		rec := httptest.NewRecorder()
		s.handleAppScale(rec, httptest.NewRequest(http.MethodPost, "/api/apps/scale", strings.NewReader(fmt.Sprintf(`{"app":"demo","env":"prod","branch":"main","replicas":%d}`, n))))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("replicas=%d: status %d, want 400", n, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	s.handleAppScale(rec, httptest.NewRequest(http.MethodPost, "/api/apps/scale", strings.NewReader(`{"app":"demo","env":"prod","branch":"main","replicas":3}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("scale: %d %s", rec.Code, rec.Body.String())
	}
	if got := s.laneReplicas("demo", EnvProd, "main"); got != 3 {
		t.Fatalf("replicas saved = %d, want 3", got)
	}
	select {
	case job := <-s.queue:
		if job.RollbackImage != "relay/demo:prod-main-1" || !job.PreserveHistory {
			t.Fatalf("scale must redeploy the current image without moving the rollback target: %+v", job)
		}
	default:
		t.Fatal("scaling a running lane should queue a zero-downtime redeploy")
	}
}

func TestEdgeFeedbackRequiresTheLaneEdgeToken(t *testing.T) {
	s, _ := newRecordingServer(t)
	s.dataDir = t.TempDir()
	post := func(token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/edge/feedback", strings.NewReader(`{"vote":"up","comment":"nice","page":"/x"}`))
		r.Header.Set("X-Relay-Lane-App", "demo")
		r.Header.Set("X-Relay-Lane-Env", "prod")
		r.Header.Set("X-Relay-Lane-Branch", "main")
		r.Header.Set("X-Relay-Variant", "b")
		r.Header.Set("X-Relay-Client-IP", "203.0.113.4")
		if token != "" {
			r.Header.Set("X-Relay-Edge-Token", token)
		}
		rec := httptest.NewRecorder()
		s.handleEdgeFeedback(rec, r)
		return rec
	}
	if rec := post("forged"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("forged token: %d, want 401", rec.Code)
	}
	token, err := s.edgeSessionToken("demo", EnvProd, "main")
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(token); rec.Code != http.StatusOK {
		t.Fatalf("valid token: %d %s", rec.Code, rec.Body.String())
	}
	var variant string
	var vote int
	if err := s.db.QueryRow(`SELECT variant, vote FROM variant_feedback WHERE app='demo'`).Scan(&variant, &vote); err != nil || variant != "b" || vote != 1 {
		t.Fatalf("stored feedback variant=%q vote=%d err=%v", variant, vote, err)
	}
}

func TestDeployExtrasValidationFailsFastBeforeABuild(t *testing.T) {
	s, rt := newRecordingServer(t)
	seedLiveLane(t, s, rt.running)
	w := 120.0
	for _, tc := range []struct {
		name     string
		variant  string
		weight   *float64
		replicas int
	}{
		{"reserved name", "control", nil, 0},
		{"weight over 100", "b", &w, 0},
		{"weight without variant", "", &w, 0},
		{"too many replicas", "", nil, 99},
	} {
		if err := s.validateDeployExtras("demo", EnvProd, "main", tc.variant, tc.weight, tc.replicas); err == nil {
			t.Fatalf("%s: expected rejection", tc.name)
		}
	}
	ok := 15.0
	if err := s.validateDeployExtras("demo", EnvProd, "main", "b", &ok, 2); err != nil {
		t.Fatalf("valid variant deploy rejected: %v", err)
	}
	if err := s.validateDeployExtras("ghost", EnvProd, "main", "b", nil, 0); err == nil {
		t.Fatal("variants on a lane that was never deployed must be rejected")
	}
}
