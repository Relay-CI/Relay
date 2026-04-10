package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
)

type LanePolicy struct {
	Env                 DeployEnv `json:"env"`
	DisplayName         string    `json:"display_name,omitempty"`
	DefaultMode         string    `json:"default_mode,omitempty"`
	DefaultTrafficMode  string    `json:"default_traffic_mode,omitempty"`
	DefaultAccessPolicy string    `json:"default_access_policy,omitempty"`
	DefaultHostPort     int       `json:"default_host_port,omitempty"`
	AutoSubdomain       bool      `json:"auto_subdomain,omitempty"`
	RandomSubdomain     bool      `json:"random_subdomain,omitempty"`
	RetentionHours      int       `json:"retention_hours,omitempty"`
	PromoteTo           string    `json:"promote_to,omitempty"`
}

var builtInLanePolicies = map[DeployEnv]LanePolicy{
	EnvProd: {
		Env:                 EnvProd,
		DisplayName:         "production",
		DefaultMode:         "port",
		DefaultTrafficMode:  "edge",
		DefaultAccessPolicy: "public",
		DefaultHostPort:     3000,
		RetentionHours:      24 * 30,
	},
	EnvStaging: {
		Env:                 EnvStaging,
		DisplayName:         "staging",
		DefaultMode:         "port",
		DefaultTrafficMode:  "edge",
		DefaultAccessPolicy: "relay-login",
		DefaultHostPort:     3002,
		AutoSubdomain:       true,
		RetentionHours:      24 * 14,
		PromoteTo:           string(EnvProd),
	},
	EnvDev: {
		Env:                 EnvDev,
		DisplayName:         "dev",
		DefaultMode:         "port",
		DefaultTrafficMode:  "edge",
		DefaultAccessPolicy: "relay-login",
		DefaultHostPort:     3003,
		AutoSubdomain:       true,
		RandomSubdomain:     true,
		RetentionHours:      24 * 7,
		PromoteTo:           string(EnvStaging),
	},
	EnvPreview: {
		Env:                 EnvPreview,
		DisplayName:         "preview",
		DefaultMode:         "port",
		DefaultTrafficMode:  "edge",
		DefaultAccessPolicy: "public",
		DefaultHostPort:     3001,
		AutoSubdomain:       true,
		RandomSubdomain:     true,
		RetentionHours:      24 * 7,
	},
}

func normalizeDeployEnv(value string) DeployEnv {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "preview":
		return EnvPreview
	case "prod", "production":
		return EnvProd
	case "staging", "stage":
		return EnvStaging
	case "dev", "development":
		return EnvDev
	default:
		return ""
	}
}

func isKnownDeployEnv(env DeployEnv) bool {
	_, ok := builtInLanePolicies[normalizeDeployEnv(string(env))]
	return ok
}

func validDeployTarget(app string, env DeployEnv, branch string) bool {
	return strings.TrimSpace(app) != "" && strings.TrimSpace(branch) != "" && isKnownDeployEnv(env)
}

func firstNonEmptyDeployEnv(values ...DeployEnv) DeployEnv {
	for _, value := range values {
		if normalized := normalizeDeployEnv(string(value)); normalized != "" {
			return normalized
		}
	}
	return ""
}

func mergeLanePolicy(base LanePolicy, override LanePolicy) LanePolicy {
	if override.DisplayName != "" {
		base.DisplayName = override.DisplayName
	}
	if override.DefaultMode != "" {
		base.DefaultMode = override.DefaultMode
	}
	if override.DefaultTrafficMode != "" {
		base.DefaultTrafficMode = override.DefaultTrafficMode
	}
	if override.DefaultAccessPolicy != "" {
		base.DefaultAccessPolicy = override.DefaultAccessPolicy
	}
	if override.DefaultHostPort > 0 {
		base.DefaultHostPort = override.DefaultHostPort
	}
	if override.RetentionHours > 0 {
		base.RetentionHours = override.RetentionHours
	}
	if override.PromoteTo != "" {
		base.PromoteTo = string(normalizeDeployEnv(override.PromoteTo))
	}
	base.AutoSubdomain = override.AutoSubdomain
	base.RandomSubdomain = override.RandomSubdomain
	return base
}

func defaultLanePolicy(env DeployEnv) LanePolicy {
	policy, ok := builtInLanePolicies[normalizeDeployEnv(string(env))]
	if !ok {
		return LanePolicy{
			Env:                 normalizeDeployEnv(string(env)),
			DefaultMode:         "port",
			DefaultTrafficMode:  "edge",
			DefaultAccessPolicy: "public",
			DefaultHostPort:     3001,
			RetentionHours:      24 * 7,
		}
	}
	return policy
}

func seedLanePolicies(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS lane_policies (
		env TEXT PRIMARY KEY,
		display_name TEXT DEFAULT '',
		default_mode TEXT DEFAULT '',
		default_traffic_mode TEXT DEFAULT '',
		default_access_policy TEXT DEFAULT '',
		default_host_port INTEGER DEFAULT 0,
		auto_subdomain INTEGER DEFAULT 0,
		random_subdomain INTEGER DEFAULT 0,
		retention_hours INTEGER DEFAULT 0,
		promote_to TEXT DEFAULT ''
	)`); err != nil {
		return err
	}
	for _, policy := range builtInLanePolicies {
		if _, err := db.Exec(
			`INSERT OR IGNORE INTO lane_policies
			(env, display_name, default_mode, default_traffic_mode, default_access_policy, default_host_port, auto_subdomain, random_subdomain, retention_hours, promote_to)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			string(policy.Env), policy.DisplayName, policy.DefaultMode, policy.DefaultTrafficMode, policy.DefaultAccessPolicy,
			policy.DefaultHostPort, boolToInt(policy.AutoSubdomain), boolToInt(policy.RandomSubdomain), policy.RetentionHours, policy.PromoteTo,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) lanePolicy(env DeployEnv) LanePolicy {
	policy := defaultLanePolicy(env)
	if s == nil || s.db == nil {
		return policy
	}
	row := s.db.QueryRow(`SELECT
		COALESCE(display_name,''),
		COALESCE(default_mode,''),
		COALESCE(default_traffic_mode,''),
		COALESCE(default_access_policy,''),
		COALESCE(default_host_port,0),
		COALESCE(auto_subdomain,0),
		COALESCE(random_subdomain,0),
		COALESCE(retention_hours,0),
		COALESCE(promote_to,'')
		FROM lane_policies WHERE env=?`, string(policy.Env))
	var override LanePolicy
	override.Env = policy.Env
	var autoSubdomain int
	var randomSubdomain int
	if err := row.Scan(
		&override.DisplayName,
		&override.DefaultMode,
		&override.DefaultTrafficMode,
		&override.DefaultAccessPolicy,
		&override.DefaultHostPort,
		&autoSubdomain,
		&randomSubdomain,
		&override.RetentionHours,
		&override.PromoteTo,
	); err == nil {
		override.AutoSubdomain = autoSubdomain != 0
		override.RandomSubdomain = randomSubdomain != 0
		policy = mergeLanePolicy(policy, override)
	}
	policy.Env = firstNonEmptyDeployEnv(policy.Env, env)
	policy.DefaultAccessPolicy = firstNonEmpty(normalizeAccessPolicy(policy.DefaultAccessPolicy), "public")
	policy.DefaultTrafficMode = firstNonEmpty(normalizeTrafficMode(policy.DefaultTrafficMode), "edge")
	policy.DefaultMode = firstNonEmpty(strings.ToLower(strings.TrimSpace(policy.DefaultMode)), "port")
	policy.PromoteTo = string(normalizeDeployEnv(policy.PromoteTo))
	if policy.DefaultHostPort <= 0 {
		policy.DefaultHostPort = 3001
	}
	return policy
}

func randomSubdomainToken() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return newID()[:8]
	}
	return fmt.Sprintf("%02x%02x%02x%02x", buf[0], buf[1], buf[2], buf[3])
}

func laneNeedsManagedHost(policy LanePolicy, baseDomain string) bool {
	return strings.TrimSpace(baseDomain) != "" && policy.AutoSubdomain
}

func (s *Server) managedLaneHost(env DeployEnv, app string, branch string, existingHost string) string {
	existingHost = strings.TrimSpace(existingHost)
	if existingHost != "" {
		return existingHost
	}
	policy := s.lanePolicy(env)
	baseDomain := strings.TrimSpace(s.serverBaseDomain())
	if !laneNeedsManagedHost(policy, baseDomain) {
		return ""
	}
	switch policy.Env {
	case EnvProd:
		return ""
	case EnvStaging:
		return fmt.Sprintf("%s-staging.%s", safe(app), baseDomain)
	case EnvDev:
		if policy.RandomSubdomain {
			label := fmt.Sprintf("%s-dev-%s", safe(app), randomSubdomainToken())
			return fmt.Sprintf("%s.%s", label, baseDomain)
		}
		return fmt.Sprintf("%s-dev.%s", safe(app), baseDomain)
	case EnvPreview:
		if policy.RandomSubdomain {
			label := fmt.Sprintf("%s-preview-%s", safe(app), randomSubdomainToken())
			return fmt.Sprintf("%s.%s", label, baseDomain)
		}
		return fmt.Sprintf("%s-%s.%s", safe(app), safe(branch), baseDomain)
	default:
		return fmt.Sprintf("%s-%s.%s", safe(app), safe(branch), baseDomain)
	}
}

func (s *Server) constrainAppState(st *AppState) {
	if st == nil {
		return
	}
	st.Env = normalizeDeployEnv(string(st.Env))
	constrainAppStateForEngine(st)
	policy := s.lanePolicy(st.Env)
	st.TrafficMode = firstNonEmpty(normalizeTrafficMode(st.TrafficMode), policy.DefaultTrafficMode)
	st.AccessPolicy = firstNonEmpty(normalizeAccessPolicy(st.AccessPolicy), policy.DefaultAccessPolicy)
	st.IPAllowlist = normalizeIPAllowlist(st.IPAllowlist)
}

func (s *Server) applyLaneDefaultsToDeployRequest(req *DeployRequest) {
	if req == nil {
		return
	}
	req.Env = normalizeDeployEnv(string(req.Env))
	policy := s.lanePolicy(req.Env)
	if req.TrafficMode == "" {
		req.TrafficMode = policy.DefaultTrafficMode
	}
	if req.Mode == "" {
		if strings.TrimSpace(req.PublicHost) != "" {
			req.Mode = "traefik"
		} else {
			req.Mode = policy.DefaultMode
		}
	}
}
