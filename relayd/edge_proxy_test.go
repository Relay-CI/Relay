package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteEdgeProxyConfigUsesStdStreamsForLogs(t *testing.T) {
	s := &Server{dataDir: t.TempDir()}
	configPath, err := s.writeEdgeProxyConfig("demo", EnvPreview, "main", "blue", "", 3000, "edge", 100)
	if err != nil {
		t.Fatalf("write edge proxy config: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read edge proxy config: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "error_log /dev/stderr;") {
		t.Fatalf("expected config to write error logs to stderr, got:\n%s", text)
	}
	if !strings.Contains(text, "access_log /dev/stdout relay_json;") {
		t.Fatalf("expected config to write access logs to stdout, got:\n%s", text)
	}
	if strings.Contains(text, "/var/log/nginx/access.log") {
		t.Fatalf("did not expect legacy /var/log/nginx/access.log path in config")
	}
}

// This hop (Caddy -> this nginx -> app container) is always plain HTTP, so
// unconditionally setting X-Forwarded-Proto to $scheme would always send
// "http" downstream, even when the public client used https (or an
// upstream CDN like Cloudflare already set the correct header). That's the
// exact cause of an http<->https redirect loop for any app that redirects
// based on this header — pin the fix: an incoming X-Forwarded-Proto must
// win over the local (always-http) scheme.
func TestWriteEdgeProxyConfigPreservesIncomingForwardedProto(t *testing.T) {
	s := &Server{dataDir: t.TempDir()}
	configPath, err := s.writeEdgeProxyConfig("demo", EnvPreview, "main", "blue", "", 3000, "edge", 100)
	if err != nil {
		t.Fatalf("write edge proxy config: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read edge proxy config: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "X-Forwarded-Proto $scheme") {
		t.Fatalf("expected X-Forwarded-Proto to not be unconditionally overwritten with the local $scheme, got:\n%s", text)
	}
	if !strings.Contains(text, "map $http_x_forwarded_proto $relay_xfp") {
		t.Fatalf("expected a map preserving an incoming X-Forwarded-Proto, got:\n%s", text)
	}
	if !strings.Contains(text, "proxy_set_header X-Forwarded-Proto $relay_xfp;") {
		t.Fatalf("expected the app-facing proxy to forward the preserved value, got:\n%s", text)
	}
}

func TestValidateEdgeProxyLogPathsRejectsUnboundFileLogs(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "nginx.conf")
	cfg := `
http {
  access_log /var/log/nginx/access.log relay_json;
  error_log /dev/stderr;
}
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	err := validateEdgeProxyLogPaths(cfgPath, []string{"C:/relay/nginx.conf:/etc/nginx/nginx.conf:ro"})
	if err == nil {
		t.Fatalf("expected validation to fail for unbound file log path")
	}
	if !strings.Contains(err.Error(), "/var/log/nginx/access.log") {
		t.Fatalf("expected missing log path in validation error, got: %v", err)
	}
}

func TestValidateEdgeProxyLogPathsAllowsStdStreams(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "nginx.conf")
	cfg := `
http {
  access_log /dev/stdout relay_json;
  error_log /dev/stderr;
}
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := validateEdgeProxyLogPaths(cfgPath, []string{"C:/relay/nginx.conf:/etc/nginx/nginx.conf:ro"}); err != nil {
		t.Fatalf("expected std stream logs to validate, got: %v", err)
	}
}

func TestEdgeProxyVolumeTargetParsesWindowsAndUnixSpecs(t *testing.T) {
	cases := []struct {
		spec string
		want string
	}{
		{spec: "C:/relay/nginx.conf:/etc/nginx/nginx.conf:ro", want: "/etc/nginx/nginx.conf"},
		{spec: "/var/lib/relay/nginx.conf:/etc/nginx/nginx.conf:ro", want: "/etc/nginx/nginx.conf"},
		{spec: "relay_data:/var/lib/mysql", want: "/var/lib/mysql"},
	}
	for _, tc := range cases {
		if got := edgeProxyVolumeTarget(tc.spec); got != tc.want {
			t.Fatalf("edgeProxyVolumeTarget(%q) = %q, want %q", tc.spec, got, tc.want)
		}
	}
}
