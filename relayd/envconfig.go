package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// appReadOnlyRootFS returns whether app containers run with a read-only root
// filesystem. DB value (admin panel) takes priority over
// RELAY_APP_READ_ONLY_ROOTFS. Default: false.
func (s *Server) appReadOnlyRootFS() bool {
	if v := s.serverConfigGet("app_read_only_rootfs"); v != "" {
		return v == "true"
	}
	return getenvBool("RELAY_APP_READ_ONLY_ROOTFS", false)
}

// appRunAs returns the user[:group] to run app containers as.
// DB value takes priority over RELAY_APP_RUN_AS.
func (s *Server) appRunAs() string {
	if v := s.serverConfigGet("app_run_as"); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("RELAY_APP_RUN_AS"))
}

// rolloutReadyTimeoutSecs returns the effective rollout readiness timeout in
// seconds. DB value takes priority over RELAY_ROLLOUT_READY_TIMEOUT_SECONDS.
// Default: 60.
func (s *Server) rolloutReadyTimeoutSecs() int {
	if v := s.serverConfigGet("rollout_ready_timeout_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAY_ROLLOUT_READY_TIMEOUT_SECONDS"))); err == nil && n > 0 {
		return n
	}
	return 60
}

func (s *Server) rolloutReadyTimeoutDuration() time.Duration {
	return time.Duration(s.rolloutReadyTimeoutSecs()) * time.Second
}

// rolloutDrainSecs returns the effective rollout drain period in seconds.
// DB value takes priority over RELAY_ROLLOUT_DRAIN_SECONDS. Default: 30.
func (s *Server) rolloutDrainSecs() int {
	if v := s.serverConfigGet("rollout_drain_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAY_ROLLOUT_DRAIN_SECONDS"))); err == nil && n >= 0 {
		return n
	}
	return 30
}

func (s *Server) rolloutDrainDuration() time.Duration {
	return time.Duration(s.rolloutDrainSecs()) * time.Second
}

// maxUploadBytes returns the maximum allowed sync upload in bytes.
// DB value takes priority over RELAY_MAX_UPLOAD_BYTES. Default: 500 MB.
func (s *Server) maxUploadBytes() int64 {
	if v := s.serverConfigGet("max_upload_bytes"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	if v := os.Getenv("RELAY_MAX_UPLOAD_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 524288000
}

// corsOriginsRaw returns the raw CORS origins string.
// DB value takes priority over RELAY_CORS_ORIGINS.
func (s *Server) corsOriginsRaw() string {
	if v := s.serverConfigGet("cors_origins"); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("RELAY_CORS_ORIGINS"))
}

// setCORSOrigins atomically replaces the live CORS allowlist.
func (s *Server) setCORSOrigins(origins map[string]struct{}, allowAll bool) {
	s.corsMu.Lock()
	defer s.corsMu.Unlock()
	s.corsOrigins = origins
	s.allowAllCORS = allowAll
}

// acmeEmailSetting returns the ACME registration email.
// DB value takes priority over RELAY_ACME_EMAIL.
func (s *Server) acmeEmailSetting() string {
	if v := s.serverConfigGet("acme_email"); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("RELAY_ACME_EMAIL"))
}

// cloudflareAPITokenSetting returns the Cloudflare API token used for DNS-01
// ACME challenges. DB value takes priority over CLOUDFLARE_API_TOKEN.
func (s *Server) cloudflareAPITokenSetting() string {
	if v := s.serverConfigGet("cloudflare_api_token"); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN"))
}

// maxConcurrentBuildsRaw returns the raw value for the concurrent build limit.
// DB value takes priority over RELAY_MAX_CONCURRENT_BUILDS. Empty = auto.
// Changes take effect on restart.
func (s *Server) maxConcurrentBuildsRaw() string {
	if v := s.serverConfigGet("max_concurrent_builds"); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("RELAY_MAX_CONCURRENT_BUILDS"))
}
