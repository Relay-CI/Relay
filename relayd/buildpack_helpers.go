package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ---------------------- Minimal defaults for buildpacks ----------------------
// These keep your buildpacks compiling even if you haven’t added custom logic yet.
// You can refine them later.

func cfgStr(cfg *RelayConfig, field string) string {
	if cfg == nil {
		return ""
	}
	switch field {
	case "BuildImage":
		return strings.TrimSpace(cfg.BuildImage)
	case "RunImage":
		return strings.TrimSpace(cfg.RunImage)
	default:
		return ""
	}
}

func quoteForSh(s string) string {
	// wrap string in single quotes safely: ' becomes '"'"'
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func shQuote(s string) string { return quoteForSh(s) }

const relayBuildEnvArg = "RELAY_BUILD_ENV_B64"

func encodeBuildEnvArgValue(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var script strings.Builder
	for _, key := range keys {
		script.WriteString("export ")
		script.WriteString(key)
		script.WriteString("=")
		script.WriteString(shQuote(env[key]))
		script.WriteString("\n")
	}
	return base64.StdEncoding.EncodeToString([]byte(script.String()))
}

func prepareBuildDockerfileWithEnv(contextDir string, dockerfilePath string, buildEnvB64 string, embedDefault bool) (string, func(), error) {
	if strings.TrimSpace(buildEnvB64) == "" {
		return dockerfilePath, func() {}, nil
	}
	srcPath := strings.TrimSpace(dockerfilePath)
	if srcPath == "" {
		srcPath = filepath.Join(contextDir, "Dockerfile")
	}
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", nil, err
	}
	wrapped := injectBuildEnvIntoDockerfile(string(content), buildEnvB64, embedDefault)
	tmp, err := os.CreateTemp(contextDir, ".relay-build-*.Dockerfile")
	if err != nil {
		return "", nil, err
	}
	if _, err := tmp.WriteString(wrapped); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", nil, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", nil, err
	}
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}

func injectBuildEnvIntoDockerfile(content string, buildEnvB64 string, embedDefault bool) string {
	lines := splitDockerfileLogicalLines(content)
	var out strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(strings.ToUpper(trimmed), "FROM "):
			out.WriteString(line)
			if !strings.HasSuffix(line, "\n") {
				out.WriteString("\n")
			}
			out.WriteString(buildEnvArgInstruction(buildEnvB64, embedDefault))
		case strings.HasPrefix(strings.ToUpper(trimmed), "RUN "):
			out.WriteString(wrapDockerRunInstructionForBuildEnv(line))
		default:
			out.WriteString(line)
		}
	}
	return out.String()
}

func splitDockerfileLogicalLines(content string) []string {
	parts := strings.SplitAfter(content, "\n")
	if len(parts) == 0 {
		return nil
	}
	lines := make([]string, 0, len(parts))
	var current strings.Builder
	for _, part := range parts {
		current.WriteString(part)
		if dockerfileLineContinues(part) {
			continue
		}
		lines = append(lines, current.String())
		current.Reset()
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func dockerfileLineContinues(line string) bool {
	line = strings.TrimRight(line, "\r\n")
	line = strings.TrimRight(line, " \t")
	if line == "" {
		return false
	}
	backslashes := 0
	for i := len(line) - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func buildEnvArgInstruction(buildEnvB64 string, embedDefault bool) string {
	if embedDefault && strings.TrimSpace(buildEnvB64) != "" {
		return fmt.Sprintf("ARG %s=%s\n", relayBuildEnvArg, buildEnvB64)
	}
	return fmt.Sprintf("ARG %s\n", relayBuildEnvArg)
}

func wrapDockerRunInstructionForBuildEnv(line string) string {
	newline := ""
	switch {
	case strings.HasSuffix(line, "\r\n"):
		newline = "\r\n"
	case strings.HasSuffix(line, "\n"):
		newline = "\n"
	}
	trimmed := strings.TrimSpace(line)
	rest := strings.TrimSpace(trimmed[3:])
	flags, cmd, ok := splitDockerRunFlags(rest)
	if !ok {
		return line
	}
	wrappedCmd, ok := wrapDockerRunCommandForBuildEnv(cmd)
	if !ok {
		return line
	}
	return fmt.Sprintf("RUN %s%s%s", flags, wrappedCmd, newline)
}

func wrapDockerRunCommandForBuildEnv(cmd string) (string, bool) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", false
	}
	if strings.HasPrefix(cmd, "[") {
		return wrapDockerRunExecCommandForBuildEnv(cmd)
	}
	script := fmt.Sprintf(`if [ -n "${%s:-}" ]; then eval "$(printf '%%s' "$%s" | base64 -d)"; fi; exec /bin/sh -lc %s`,
		relayBuildEnvArg,
		relayBuildEnvArg,
		shQuote(cmd),
	)
	return fmt.Sprintf("/bin/sh -lc %s", shQuote(script)), true
}

func wrapDockerRunExecCommandForBuildEnv(cmd string) (string, bool) {
	var argv []string
	if err := json.Unmarshal([]byte(cmd), &argv); err != nil || len(argv) == 0 {
		return "", false
	}
	script := fmt.Sprintf(`if [ -n "${%s:-}" ]; then eval "$(printf '%%s' "$%s" | base64 -d)"; fi; exec "$0" "$@"`,
		relayBuildEnvArg,
		relayBuildEnvArg,
	)
	wrapped := append([]string{"/bin/sh", "-lc", script}, argv...)
	encoded, err := json.Marshal(wrapped)
	if err != nil {
		return "", false
	}
	return string(encoded), true
}

func splitDockerRunFlags(rest string) (string, string, bool) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", "", false
	}
	var flags []string
	for strings.HasPrefix(rest, "--") {
		idx := strings.IndexAny(rest, " \t\r\n")
		if idx <= 0 {
			return "", "", false
		}
		flags = append(flags, rest[:idx])
		rest = strings.TrimLeft(rest[idx:], " \t\r\n")
	}
	if rest == "" {
		return "", "", false
	}
	if len(flags) == 0 {
		return "", rest, true
	}
	return strings.Join(flags, " ") + " ", rest, true
}

func shellJSON(cmd string) string {
	// Dockerfile JSON CMD expects ["sh","-lc","..."] form for arbitrary strings
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		cmd = "sleep 3600"
	}
	b, _ := json.Marshal([]string{"sh", "-lc", cmd})
	return string(b)
}

func shellForm(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		cmd = "sleep 3600"
	}
	// shell form: sh -lc "..."
	return fmt.Sprintf(`["sh","-lc",%s]`, mustJSON(cmd))
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func packageManagerName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if i := strings.IndexByte(s, '@'); i >= 0 {
		s = s[:i]
	}
	return s
}

func bunProject(repoDir string) bool {
	if fileExists(filepath.Join(repoDir, "bun.lock")) || fileExists(filepath.Join(repoDir, "bun.lockb")) {
		return true
	}
	if pkg, _, ok := readPackageManifest(repoDir); ok && pkg != nil {
		return packageManagerName(pkg.PackageManager) == "bun"
	}
	return false
}

func bunHasLockfile(repoDir string) bool {
	return fileExists(filepath.Join(repoDir, "bun.lock")) || fileExists(filepath.Join(repoDir, "bun.lockb"))
}

func bunLockfileCopyInstruction(repoDir string) string {
	if !bunHasLockfile(repoDir) {
		return ""
	}
	return "COPY bun.lock* ./\n"
}

func bunInstallCmd(repoDir string) string {
	if bunHasLockfile(repoDir) {
		return "bun install --frozen-lockfile"
	}
	return "bun install"
}

func bunProdInstallCmd(repoDir string) string {
	if bunHasLockfile(repoDir) {
		return "bun install --production --frozen-lockfile"
	}
	return "bun install --production"
}

func bunPackageCacheDir() string {
	return "/root/.bun/install/cache"
}

func bunRunStepWithCache(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	return fmt.Sprintf("RUN --mount=type=cache,target=%s %s", bunPackageCacheDir(), cmd)
}

func bunDefaultBuildCmd(repoDir string) string {
	if nodePackageScript(repoDir, "build") != "" {
		return "bun run build"
	}
	return ""
}

func bunDefaultStartCmd(repoDir string) string {
	if nodePackageScript(repoDir, "start") != "" {
		return "bun run start"
	}
	for _, name := range []string{"server.ts", "server.js", "index.ts", "index.js"} {
		if fileExists(filepath.Join(repoDir, name)) {
			return "bun run " + name
		}
	}
	return "bun run start"
}

func rubyInstallCmd(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "Gemfile")) {
		return `sh -lc "bundle config set without 'development test' && bundle install"`
	}
	return ""
}

func rubyDefaultBuildCmd(repoDir string) string {
	if !rubyRailsApp(repoDir) {
		return ""
	}
	return `sh -lc "if [ -f bin/rails ]; then SECRET_KEY_BASE=dummy bundle exec bin/rails assets:precompile; else SECRET_KEY_BASE=dummy bundle exec rails assets:precompile; fi"`
}

func rubyDefaultStart(repoDir string) string {
	if rubyRailsApp(repoDir) {
		if fileExists(filepath.Join(repoDir, "bin", "rails")) {
			return `sh -lc "bundle exec bin/rails server -e production -b 0.0.0.0 -p ${PORT:-3000}"`
		}
		return `sh -lc "bundle exec rails server -e production -b 0.0.0.0 -p ${PORT:-3000}"`
	}
	if fileExists(filepath.Join(repoDir, "config.ru")) {
		return `sh -lc "bundle exec rackup --host 0.0.0.0 --port ${PORT:-3000}"`
	}
	return `sh -lc "bundle exec ruby app.rb"`
}

func rubyRailsApp(repoDir string) bool {
	if fileExists(filepath.Join(repoDir, "config", "application.rb")) || fileExists(filepath.Join(repoDir, "bin", "rails")) {
		return true
	}
	if b, err := os.ReadFile(filepath.Join(repoDir, "Gemfile")); err == nil {
		return strings.Contains(strings.ToLower(string(b)), "gem \"rails\"") || strings.Contains(strings.ToLower(string(b)), "gem 'rails'")
	}
	return false
}

func rubyRunStep(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	return "RUN " + cmd + "\n"
}

// Node helpers
var (
	nextConfigStandaloneLiteralRe   = regexp.MustCompile(`\boutput\s*:\s*(?:"standalone"|'standalone')`)
	nextConfigStandaloneAssignVarRe = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:"standalone"|'standalone')`)
	nextConfigExportLiteralRe       = regexp.MustCompile(`\boutput\s*:\s*(?:"export"|'export')`)
	nextConfigExportAssignVarRe     = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:"export"|'export')`)
)

func nodePackageManager(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if fileExists(filepath.Join(repoDir, "yarn.lock")) {
		return "yarn"
	}
	return "npm"
}

func nodeInstallCmd(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return "corepack enable && pnpm install --frozen-lockfile --prefer-offline"
	case "yarn":
		return "corepack enable && yarn install --frozen-lockfile --silent"
	default:
		// --no-optional skips packages listed under optionalDependencies (native
		// speed-up modules, platform-specific binaries, etc.) which are the
		// heaviest to install and the first to exhaust memory during post-install
		// scripts. Apps that genuinely need an optional dep can override via
		// relay.config.json install_cmd.
		return "npm ci --prefer-offline --no-audit --no-fund --no-optional"
	}
}

func nodeProdInstallCmd(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return "corepack enable && pnpm install --frozen-lockfile --prod --prefer-offline"
	case "yarn":
		return "corepack enable && yarn install --frozen-lockfile --production=true --silent"
	default:
		return "npm ci --omit=dev --prefer-offline --no-audit --no-fund --no-optional"
	}
}

func nodeProdDepsStage(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return fmt.Sprintf(`FROM deps AS prod-deps
RUN pnpm prune --prod || true
RUN mkdir -p /app/node_modules
`)
	case "yarn":
		return fmt.Sprintf(`FROM deps AS prod-deps
RUN rm -rf node_modules
%s
RUN mkdir -p /app/node_modules
`, nodeRunStepWithCaches(repoDir, "", nodeProdInstallCmd(repoDir)))
	default:
		return `FROM deps AS prod-deps
RUN npm prune --omit=dev --no-audit --no-fund || true
RUN mkdir -p /app/node_modules
`
	}
}

func nodePackageCacheDir(repoDir string) string {
	switch nodePackageManager(repoDir) {
	case "pnpm":
		return "/root/.local/share/pnpm/store"
	case "yarn":
		return "/usr/local/share/.cache/yarn"
	default:
		return "/root/.npm"
	}
}

// nodeRunStepWithCaches emits a Dockerfile RUN line with BuildKit cache mounts.
//
// The package-manager cache (npm/pnpm/yarn store) is intentionally shared across
// all apps — it is content-addressed so sharing only helps.
//
// extraTargets are app-specific caches (e.g. .next/cache). When laneID is non-empty
// each extra target gets an explicit id= so different deployment lanes never clobber
// each other's incremental build artifacts.
func nodeRunStepWithCaches(repoDir string, laneID string, cmd string, extraTargets ...string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}

	var mounts []string
	if target := strings.TrimSpace(nodePackageCacheDir(repoDir)); target != "" {
		// Shared across all builds — content-addressed, so cross-app hits are free.
		mounts = append(mounts, fmt.Sprintf("--mount=type=cache,target=%s", target))
	}
	for _, target := range extraTargets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		if laneID != "" {
			// Scope per deployment lane so concurrent Next.js (or similar) builds
			// don't overwrite each other's incremental cache.
			mounts = append(mounts, fmt.Sprintf("--mount=type=cache,id=%s,target=%s", laneID, target))
		} else {
			mounts = append(mounts, fmt.Sprintf("--mount=type=cache,target=%s", target))
		}
	}
	if len(mounts) == 0 {
		return fmt.Sprintf("RUN %s", cmd)
	}
	return fmt.Sprintf("RUN %s %s", strings.Join(mounts, " "), cmd)
}

func nodeDefaultBuildCmd(repoDir string) string {
	// Only emit a build step if package.json actually defines a "build" script.
	if nodePackageScript(repoDir, "build") == "" {
		return ""
	}
	// Pick package manager based on lockfile.
	if fileExists(filepath.Join(repoDir, "pnpm-lock.yaml")) {
		return "pnpm run build"
	}
	if fileExists(filepath.Join(repoDir, "yarn.lock")) {
		return "yarn build"
	}
	return "npm run build"
}

func nodeBuildHeapMB() string {
	value := strings.TrimSpace(os.Getenv("RELAY_NODE_BUILD_HEAP_MB"))
	if value != "" {
		if _, err := strconv.Atoi(value); err == nil {
			return value
		}
	}
	// No explicit setting: size the V8 heap to the host. V8's RSS overshoots
	// old-space by 30-50%, and Docker, relayd, and the apps being hosted all
	// share the same machine — so a fixed 1024 MB heap still OOM-killed builds
	// on 2 GB instances. Stay well under half of physical RAM.
	if total := hostTotalMemMB(); total > 0 {
		return strconv.Itoa(adaptiveNodeBuildHeapMB(total))
	}
	return "1024"
}

func adaptiveNodeBuildHeapMB(totalMB int) int {
	heap := totalMB/2 - 256
	if heap < 384 {
		heap = 384
	}
	if heap > 2048 {
		heap = 2048
	}
	return heap
}

func nodeBuildCmdWithMemoryGuard(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	if strings.TrimSpace(os.Getenv("RELAY_NODE_BUILD_MEMORY_GUARD")) == "0" {
		return cmd
	}
	if strings.Contains(cmd, "NODE_OPTIONS=") {
		return cmd
	}
	// The nice prefix deprioritizes the build relative to the apps already
	// serving traffic on the same host — container processes all share the
	// kernel scheduler, so niceness works across container boundaries. The
	// command substitution degrades to a no-op on images without `nice`.
	nice := ""
	if strings.TrimSpace(os.Getenv("RELAY_BUILD_NICE")) != "0" {
		nice = `$(command -v nice >/dev/null 2>&1 && echo nice -n 10) `
	}
	return fmt.Sprintf(`NODE_OPTIONS="${NODE_OPTIONS:-} --max-old-space-size=%s" NEXT_TELEMETRY_DISABLED=1 %s%s`, nodeBuildHeapMB(), nice, cmd)
}

func nodeDefaultStartCmd(repoDir string) string {
	if nodePackageScript(repoDir, "start") != "" {
		// Respect explicit lifecycle scripts when present.
		if fileExists(filepath.Join(repoDir, "pnpm-lock.yaml")) {
			return "pnpm start"
		}
		if fileExists(filepath.Join(repoDir, "yarn.lock")) {
			return "yarn start"
		}
		return "npm start"
	}
	for _, name := range []string{"server.js", "index.js"} {
		if fileExists(filepath.Join(repoDir, name)) {
			return "node " + name
		}
	}
	// Default to start script matching package manager
	if fileExists(filepath.Join(repoDir, "pnpm-lock.yaml")) {
		return "pnpm start"
	}
	if fileExists(filepath.Join(repoDir, "yarn.lock")) {
		return "yarn start"
	}
	return "npm start"
}

func nodePackageScript(repoDir string, name string) string {
	pkg, _, ok := readPackageManifest(repoDir)
	if !ok || pkg == nil {
		return ""
	}
	return strings.TrimSpace(pkg.Scripts[name])
}

func nextClassicStartCmd(repoDir string) string {
	if nodePackageScript(repoDir, "start") != "" {
		return nodeDefaultStartCmd(repoDir)
	}
	return `exec ./node_modules/.bin/next start --hostname 0.0.0.0 --port ${PORT:-3000}`
}

// writeRelayDockerignore writes a .dockerignore to repoDir if none exists,
// leaving a marker so cleanupRelayDockerignore can remove it afterwards.
// This prevents .git, leftover node_modules, etc. from inflating the build
// context and busting layer cache on every commit.
func writeRelayDockerignore(repoDir string) {
	if fileExists(filepath.Join(repoDir, ".dockerignore")) {
		return // respect the user's own .dockerignore
	}
	content := ".git\nnode_modules\n.next\ndist\nbuild\n*.log\n.env\n.env.*\n!.env.example\n!.env.sample\n!.env.template\n"
	_ = os.WriteFile(filepath.Join(repoDir, ".dockerignore"), []byte(content), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, ".relay_dockerignore_created"), []byte("1"), 0644)
}

// cleanupRelayDockerignore removes the .dockerignore only if we created it.
func cleanupRelayDockerignore(repoDir string) {
	marker := filepath.Join(repoDir, ".relay_dockerignore_created")
	if fileExists(marker) {
		_ = os.Remove(filepath.Join(repoDir, ".dockerignore"))
		_ = os.Remove(marker)
	}
}

func cleanupGeneratedBuildpackFiles(repoDir string) {
	cleanupRelayDockerignore(repoDir)
	_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
	marker := filepath.Join(repoDir, ".relay_default_conf_created")
	if fileExists(marker) {
		_ = os.Remove(filepath.Join(repoDir, "default.conf"))
		_ = os.Remove(marker)
	}
}

func isNextStandaloneEnabled(repoDir string) bool {
	configs := []string{"next.config.js", "next.config.mjs", "next.config.cjs", "next.config.ts", "next.config.mts"}
	for _, c := range configs {
		p := filepath.Join(repoDir, c)
		if b, err := os.ReadFile(p); err == nil {
			if nextConfigEnablesStandalone(string(b)) {
				return true
			}
		}
	}
	return false
}

func isNextExportEnabled(repoDir string) bool {
	configs := []string{"next.config.js", "next.config.mjs", "next.config.cjs", "next.config.ts", "next.config.mts"}
	for _, c := range configs {
		p := filepath.Join(repoDir, c)
		if b, err := os.ReadFile(p); err == nil {
			if nextConfigEnablesExport(string(b)) {
				return true
			}
		}
	}
	return false
}

func nextConfigEnablesStandalone(src string) bool {
	cleaned := stripJSComments(src)
	if nextConfigStandaloneLiteralRe.MatchString(cleaned) {
		return true
	}
	for _, match := range nextConfigStandaloneAssignVarRe.FindAllStringSubmatch(cleaned, -1) {
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if nextConfigUsesOutputVariable(cleaned, name) {
			return true
		}
		if name == "output" && nextConfigHasOutputShorthand(cleaned) {
			return true
		}
	}
	return false
}

func nextConfigEnablesExport(src string) bool {
	cleaned := stripJSComments(src)
	if nextConfigExportLiteralRe.MatchString(cleaned) {
		return true
	}
	for _, match := range nextConfigExportAssignVarRe.FindAllStringSubmatch(cleaned, -1) {
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if nextConfigUsesOutputVariable(cleaned, name) {
			return true
		}
		if name == "output" && nextConfigHasOutputShorthand(cleaned) {
			return true
		}
	}
	return false
}

func nextConfigUsesOutputVariable(src string, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	re := regexp.MustCompile(fmt.Sprintf(`\boutput\s*:\s*%s\b`, regexp.QuoteMeta(name)))
	return re.MatchString(src)
}

func nextConfigHasOutputShorthand(src string) bool {
	for i := 0; i < len(src); i++ {
		if !isJSIdentifierStart(src[i]) {
			continue
		}
		j := i + 1
		for j < len(src) && isJSIdentifierPart(src[j]) {
			j++
		}
		if src[i:j] != "output" {
			i = j - 1
			continue
		}
		prev := prevNonSpaceByte(src, i-1)
		next := nextNonSpaceByte(src, j)
		if (prev == '{' || prev == ',') && (next == '}' || next == ',') {
			return true
		}
		i = j - 1
	}
	return false
}

func prevNonSpaceByte(src string, idx int) byte {
	for idx >= 0 {
		if !isJSSpace(src[idx]) {
			return src[idx]
		}
		idx--
	}
	return 0
}

func nextNonSpaceByte(src string, idx int) byte {
	for idx < len(src) {
		if !isJSSpace(src[idx]) {
			return src[idx]
		}
		idx++
	}
	return 0
}

func isJSSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isJSIdentifierStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isJSIdentifierPart(ch byte) bool {
	return isJSIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func stripJSComments(src string) string {
	const (
		jsStateNormal = iota
		jsStateSingleQuote
		jsStateDoubleQuote
		jsStateTemplate
		jsStateLineComment
		jsStateBlockComment
	)
	var out strings.Builder
	out.Grow(len(src))
	state := jsStateNormal
	for i := 0; i < len(src); i++ {
		ch := src[i]
		switch state {
		case jsStateNormal:
			if ch == '/' && i+1 < len(src) {
				switch src[i+1] {
				case '/':
					state = jsStateLineComment
					i++
					continue
				case '*':
					state = jsStateBlockComment
					i++
					continue
				}
			}
			out.WriteByte(ch)
			switch ch {
			case '\'':
				state = jsStateSingleQuote
			case '"':
				state = jsStateDoubleQuote
			case '`':
				state = jsStateTemplate
			}
		case jsStateLineComment:
			if ch == '\n' || ch == '\r' {
				out.WriteByte(ch)
				state = jsStateNormal
			}
		case jsStateBlockComment:
			if ch == '*' && i+1 < len(src) && src[i+1] == '/' {
				i++
				state = jsStateNormal
				continue
			}
			if ch == '\n' || ch == '\r' {
				out.WriteByte(ch)
			}
		case jsStateSingleQuote:
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
				continue
			}
			if ch == '\'' {
				state = jsStateNormal
			}
		case jsStateDoubleQuote:
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
				continue
			}
			if ch == '"' {
				state = jsStateNormal
			}
		case jsStateTemplate:
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
				continue
			}
			if ch == '`' {
				state = jsStateNormal
			}
		}
	}
	return out.String()
}

// Python helpers
func pythonInstallCmd(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "requirements.txt")) {
		return `sh -lc "pip install --no-cache-dir -r requirements.txt"`
	}
	if fileExists(filepath.Join(repoDir, "pyproject.toml")) {
		return `sh -lc "pip install --no-cache-dir ."`
	}
	if fileExists(filepath.Join(repoDir, "Pipfile")) {
		return `sh -lc "pip install --no-cache-dir pipenv && pipenv install --system --deploy"`
	}
	return ""
}

func pythonEntryModule(repoDir string) string {
	for _, name := range []string{"main.py", "app.py"} {
		if fileExists(filepath.Join(repoDir, name)) {
			return strings.TrimSuffix(name, ".py")
		}
	}
	return "main"
}

func pythonDependencyFileText(repoDir string) string {
	var parts []string
	for _, name := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if b, err := os.ReadFile(filepath.Join(repoDir, name)); err == nil {
			parts = append(parts, strings.ToLower(string(b)))
		}
	}
	return strings.Join(parts, "\n")
}

func pythonHasDependency(repoDir string, names ...string) bool {
	s := pythonDependencyFileText(repoDir)
	if s == "" {
		return false
	}
	for _, name := range names {
		if strings.Contains(s, strings.ToLower(name)) {
			return true
		}
	}
	return false
}

func djangoProjectModule(repoDir string) string {
	if b, err := os.ReadFile(filepath.Join(repoDir, "manage.py")); err == nil {
		re := regexp.MustCompile(`DJANGO_SETTINGS_MODULE["']\s*,\s*["']([A-Za-z0-9_\.]+)\.settings["']`)
		if match := re.FindStringSubmatch(string(b)); len(match) == 2 {
			return strings.TrimSpace(match[1])
		}
	}
	for _, rel := range findFilesByExt(repoDir, ".py") {
		if filepath.Base(rel) != "settings.py" {
			continue
		}
		dir := filepath.Dir(rel)
		if dir == "." || dir == "" {
			continue
		}
		return strings.ReplaceAll(filepath.ToSlash(dir), "/", ".")
	}
	return ""
}

func pythonFramework(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "manage.py")) {
		return "django"
	}
	_, content := readFirstExisting(repoDir, "main.py", "app.py")
	if content != "" {
		s := strings.ToLower(content)
		switch {
		case strings.Contains(s, "fastapi"):
			return "fastapi"
		case strings.Contains(s, "flask"):
			return "flask"
		}
	}
	s := pythonDependencyFileText(repoDir)
	switch {
	case strings.Contains(s, "django"):
		return "django"
	case strings.Contains(s, "fastapi"), strings.Contains(s, "uvicorn"):
		return "fastapi"
	case strings.Contains(s, "flask"), strings.Contains(s, "gunicorn"):
		return "flask"
	}
	return "generic"
}

func pythonDefaultStart(repoDir string) string {
	module := pythonEntryModule(repoDir)
	switch pythonFramework(repoDir) {
	case "django":
		project := djangoProjectModule(repoDir)
		switch {
		case project != "" && fileExists(filepath.Join(repoDir, filepath.FromSlash(strings.ReplaceAll(project, ".", "/")), "asgi.py")) && pythonHasDependency(repoDir, "uvicorn"):
			return fmt.Sprintf(`sh -lc "python -m uvicorn %s.asgi:application --host 0.0.0.0 --port ${PORT:-8000}"`, project)
		case project != "" && fileExists(filepath.Join(repoDir, filepath.FromSlash(strings.ReplaceAll(project, ".", "/")), "wsgi.py")) && pythonHasDependency(repoDir, "gunicorn"):
			return fmt.Sprintf(`sh -lc "python -m gunicorn %s.wsgi:application --bind 0.0.0.0:${PORT:-8000}"`, project)
		default:
			return `sh -lc "python manage.py runserver 0.0.0.0:${PORT:-8000}"`
		}
	case "fastapi":
		return fmt.Sprintf(`sh -lc "python -m uvicorn %s:app --host 0.0.0.0 --port ${PORT:-8000}"`, module)
	case "flask":
		return fmt.Sprintf(`sh -lc "FLASK_APP=%s flask run --host 0.0.0.0 --port ${PORT:-8000}"`, module)
	}
	if fileExists(filepath.Join(repoDir, "main.py")) {
		return `sh -lc "python main.py"`
	}
	if fileExists(filepath.Join(repoDir, "app.py")) {
		return `sh -lc "python app.py"`
	}
	return `sh -lc "python -m http.server ${PORT:-8000} --bind 0.0.0.0"`
}

// Java helpers
func javaDefaultBuild(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "pom.xml")) {
		return `sh -lc "mvn -q -DskipTests package && cp target/*.jar app.jar"`
	}
	// gradle
	if fileExists(filepath.Join(repoDir, "build.gradle")) || fileExists(filepath.Join(repoDir, "build.gradle.kts")) {
		if fileExists(filepath.Join(repoDir, "gradlew")) {
			return `sh -lc "./gradlew -q build -x test && cp build/libs/*.jar app.jar"`
		}
		return `sh -lc "gradle -q build -x test && cp build/libs/*.jar app.jar"`
	}
	return `sh -lc "echo 'no build tool detected'; exit 1"`
}

// .NET helpers
func dotnetPickEntry(repoDir string) string {
	// Prefer .sln
	if m, _ := filepath.Glob(filepath.Join(repoDir, "*.sln")); len(m) > 0 {
		return filepath.Base(m[0])
	}
	if m, _ := filepath.Glob(filepath.Join(repoDir, "*.csproj")); len(m) > 0 {
		return filepath.Base(m[0])
	}
	return ""
}

func cCppDefaultBuildCmd(repoDir string) string {
	if fileExists(filepath.Join(repoDir, "CMakeLists.txt")) {
		return `sh -lc "cmake -S . -B build -DCMAKE_BUILD_TYPE=Release && cmake --build build --config Release -j && bin=$(find build -type f -perm /111 ! -name '*.so' ! -name '*.a' | head -n1); test -n \"$bin\"; mkdir -p /out; cp \"$bin\" /out/app"`
	}
	if fileExists(filepath.Join(repoDir, "Makefile")) {
		return `sh -lc "make && bin=$(find . -maxdepth 4 -type f -perm /111 ! -name '*.so' ! -name '*.a' ! -path './.git/*' | head -n1); test -n \"$bin\"; mkdir -p /out; cp \"$bin\" /out/app"`
	}
	if len(findFilesByExt(repoDir, ".cc", ".cpp", ".cxx")) > 0 {
		return `sh -lc "mkdir -p /out && g++ -O2 -std=c++17 -o /out/app $(find . -maxdepth 4 -type f \( -name '*.cc' -o -name '*.cpp' -o -name '*.cxx' \) | sort)"`
	}
	if len(findFilesByExt(repoDir, ".c")) > 0 {
		return `sh -lc "mkdir -p /out && gcc -O2 -o /out/app $(find . -maxdepth 4 -type f -name '*.c' | sort)"`
	}
	return `sh -lc "echo 'no C/C++ sources found'; exit 1"`
}

func writeStaticDockerArtifacts(repoDir string, dockerfile string, includeWasmMime bool) error {
	writeRelayDockerignore(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		return err
	}
	defPath := filepath.Join(repoDir, "default.conf")
	marker := filepath.Join(repoDir, ".relay_default_conf_created")
	if fileExists(defPath) {
		return nil
	}
	var conf strings.Builder
	conf.WriteString("server {\n")
	conf.WriteString("  listen 80;\n")
	conf.WriteString("  server_name _;\n")
	conf.WriteString("  root /usr/share/nginx/html;\n")
	conf.WriteString("  index index.html;\n")
	if includeWasmMime {
		conf.WriteString("  types {\n")
		conf.WriteString("    application/wasm wasm;\n")
		conf.WriteString("  }\n")
	}
	conf.WriteString("  location / {\n")
	conf.WriteString("    try_files $uri $uri/ /index.html;\n")
	conf.WriteString("  }\n")
	conf.WriteString("}\n")
	if err := os.WriteFile(defPath, []byte(conf.String()), 0644); err != nil {
		return err
	}
	return os.WriteFile(marker, []byte("1"), 0644)
}

func cleanupStaticDockerArtifacts(repoDir string, removeNodeArtifacts bool) error {
	cleanupRelayDockerignore(repoDir)
	if removeNodeArtifacts {
		_ = os.RemoveAll(filepath.Join(repoDir, "node_modules"))
		_ = os.RemoveAll(filepath.Join(repoDir, "dist"))
	}
	_ = os.Remove(filepath.Join(repoDir, "Dockerfile"))
	marker := filepath.Join(repoDir, ".relay_default_conf_created")
	if fileExists(marker) {
		_ = os.Remove(filepath.Join(repoDir, "default.conf"))
		_ = os.Remove(marker)
	}
	return nil
}
