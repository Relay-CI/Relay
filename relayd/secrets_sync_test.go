package main

import "testing"

func TestParseRawSecretEnvTextParsesDotenvStyleInput(t *testing.T) {
	parsed, err := parseRawSecretEnvText(`
# comment
export API_URL=https://api.example.com
JWT_SECRET="top-secret"
FEATURE_FLAG=true
`)
	if err != nil {
		t.Fatalf("parseRawSecretEnvText returned error: %v", err)
	}
	if got := parsed["API_URL"]; got != "https://api.example.com" {
		t.Fatalf("API_URL mismatch: got %q", got)
	}
	if got := parsed["JWT_SECRET"]; got != "top-secret" {
		t.Fatalf("JWT_SECRET mismatch: got %q", got)
	}
	if got := parsed["FEATURE_FLAG"]; got != "true" {
		t.Fatalf("FEATURE_FLAG mismatch: got %q", got)
	}
}

func TestParseRawSecretEnvTextRejectsInvalidLines(t *testing.T) {
	if _, err := parseRawSecretEnvText("NOT_A_PAIR"); err == nil {
		t.Fatalf("expected invalid raw env line to fail")
	}
	if _, err := parseRawSecretEnvText("1INVALID=value"); err == nil {
		t.Fatalf("expected invalid key to fail")
	}
}

func TestIsIgnoredWorkspaceEnvPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: ".env", want: true},
		{path: ".env.production", want: true},
		{path: "apps/web/.env.local", want: true},
		{path: ".env.example", want: false},
		{path: ".env.sample", want: false},
		{path: "README.md", want: false},
	}
	for _, tc := range cases {
		if got := isIgnoredWorkspaceEnvPath(tc.path); got != tc.want {
			t.Fatalf("isIgnoredWorkspaceEnvPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestMergeLaneSecretsIntoEnvIncludesAdminSecrets(t *testing.T) {
	s := newPreviewPortTestServer(t)
	encrypted := s.encryptSecret("admin-token")
	if _, err := s.db.Exec(
		`INSERT OR REPLACE INTO app_secrets (app, env, branch, key, value) VALUES (?, ?, ?, ?, ?)`,
		"demo", string(EnvProd), "main", "API_TOKEN", encrypted,
	); err != nil {
		t.Fatalf("insert secret: %v", err)
	}
	target := map[string]string{"API_TOKEN": "old", "OTHER": "keep"}
	s.mergeLaneSecretsIntoEnv("demo", EnvProd, "main", target, nil)
	if got := target["API_TOKEN"]; got != "admin-token" {
		t.Fatalf("expected secret override, got %q", got)
	}
	if got := target["OTHER"]; got != "keep" {
		t.Fatalf("expected existing unrelated env value to remain, got %q", got)
	}
}
