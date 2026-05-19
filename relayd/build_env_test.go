package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncodeBuildEnvArgValueSortsAndQuotes(t *testing.T) {
	encoded := encodeBuildEnvArgValue(map[string]string{
		"Z_VAR": "two words",
		"A_VAR": "it's-on",
	})
	if encoded == "" {
		t.Fatal("expected encoded build env")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode build env: %v", err)
	}
	got := string(decoded)
	want := "export A_VAR='it'\"'\"'s-on'\nexport Z_VAR='two words'\n"
	if got != want {
		t.Fatalf("unexpected build env script:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestInjectBuildEnvIntoDockerfileWrapsRunSteps(t *testing.T) {
	src := strings.Join([]string{
		"FROM node:22 AS builder",
		"RUN --mount=type=cache,target=/root/.npm npm ci",
		"RUN npm run build",
		"FROM nginx:alpine",
		"COPY --from=builder /app/dist /usr/share/nginx/html",
		"",
	}, "\n")
	got := injectBuildEnvIntoDockerfile(src, "YWJj", false)
	if !strings.Contains(got, "ARG "+relayBuildEnvArg+"\nRUN --mount=type=cache,target=/root/.npm /bin/sh -lc ") {
		t.Fatalf("expected build arg and wrapped cache run, got:\n%s", got)
	}
	if !strings.Contains(got, `exec /bin/sh -lc '"'"'npm run build'"'"'`) {
		t.Fatalf("expected wrapped shell run, got:\n%s", got)
	}
	if strings.Contains(got, "ARG "+relayBuildEnvArg+"=YWJj") {
		t.Fatalf("did not expect embedded default for docker build wrapper, got:\n%s", got)
	}
}

func TestInjectBuildEnvIntoDockerfileWrapsExecRunSteps(t *testing.T) {
	src := strings.Join([]string{
		"FROM node:22 AS builder",
		`RUN ["npm","run","build"]`,
		`RUN --mount=type=cache,target=/root/.npm ["npm","ci"]`,
		"",
	}, "\n")
	got := injectBuildEnvIntoDockerfile(src, "YWJj", false)
	if !strings.Contains(got, `RUN ["/bin/sh","-lc","if [ -n \"${RELAY_BUILD_ENV_B64:-}\" ]; then eval \"$(printf '%s' \"$RELAY_BUILD_ENV_B64\" | base64 -d)\"; fi; exec \"$0\" \"$@\"","npm","run","build"]`) {
		t.Fatalf("expected wrapped exec-form run, got:\n%s", got)
	}
	if !strings.Contains(got, `RUN --mount=type=cache,target=/root/.npm ["/bin/sh","-lc","if [ -n \"${RELAY_BUILD_ENV_B64:-}\" ]; then eval \"$(printf '%s' \"$RELAY_BUILD_ENV_B64\" | base64 -d)\"; fi; exec \"$0\" \"$@\"","npm","ci"]`) {
		t.Fatalf("expected wrapped exec-form run with flags, got:\n%s", got)
	}
}

func TestPrepareBuildDockerfileWithEnvCreatesWrappedCopy(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(orig, []byte("FROM alpine\nRUN echo hello\n"), 0644); err != nil {
		t.Fatalf("write original dockerfile: %v", err)
	}
	wrapped, cleanup, err := prepareBuildDockerfileWithEnv(dir, "", "YWJj", true)
	if err != nil {
		t.Fatalf("prepare wrapped dockerfile: %v", err)
	}
	defer cleanup()
	if wrapped == orig {
		t.Fatal("expected temp dockerfile path")
	}
	content, err := os.ReadFile(wrapped)
	if err != nil {
		t.Fatalf("read wrapped dockerfile: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "ARG "+relayBuildEnvArg+"=YWJj") {
		t.Fatalf("expected embedded default build arg, got:\n%s", text)
	}
	if !strings.Contains(text, `exec /bin/sh -lc '"'"'echo hello'"'"'`) {
		t.Fatalf("expected wrapped run command, got:\n%s", text)
	}
}
