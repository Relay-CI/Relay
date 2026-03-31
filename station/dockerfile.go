package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ─── instruction types ────────────────────────────────────────────────────────

type DFInstructionKind string

const (
	DFFrom    DFInstructionKind = "FROM"
	DFRun     DFInstructionKind = "RUN"
	DFCopy    DFInstructionKind = "COPY"
	DFWorkdir DFInstructionKind = "WORKDIR"
	DFEnv     DFInstructionKind = "ENV"
	DFCmd     DFInstructionKind = "CMD"
	DFExpose  DFInstructionKind = "EXPOSE"
	DFArg     DFInstructionKind = "ARG"
	DFLabel   DFInstructionKind = "LABEL"
	DFUser    DFInstructionKind = "USER"
)

// DFInstruction is a single parsed Dockerfile instruction.
type DFInstruction struct {
	Kind DFInstructionKind

	// FROM
	Image     string // base image reference, e.g. "node:22"
	StageName string // AS <name>

	// RUN
	Shell     string       // shell form: sh -c <Shell>
	RunMounts []DFRunMount // BuildKit-style RUN mounts, e.g. --mount=type=cache,target=/root/.npm

	// COPY
	Srcs      []string // source paths
	Dest      string   // destination path
	FromStage string   // --from=<stage>

	// WORKDIR / USER
	Path string

	// ENV
	EnvKey string
	EnvVal string

	// CMD
	Cmd []string // exec form ["node","server.js"] or shell form wrapped in ["sh","-c","..."]

	// EXPOSE
	Port int

	// ARG
	ArgName    string
	ArgDefault string
}

// DFRunMount is a parsed RUN --mount option.
type DFRunMount struct {
	Type   string
	Target string
	Source string
	ID     string
}

// DFStage is one build stage (FROM ... to next FROM or EOF).
type DFStage struct {
	Name         string          // empty or AS name
	Image        string          // base image
	Instructions []DFInstruction // everything after the FROM
}

// Dockerfile is the parsed result.
type Dockerfile struct {
	Stages []DFStage
}

// StageByName returns the stage with the given name (case-insensitive), or nil.
func (d *Dockerfile) StageByName(name string) *DFStage {
	name = strings.ToLower(name)
	for i := range d.Stages {
		if strings.ToLower(d.Stages[i].Name) == name {
			return &d.Stages[i]
		}
	}
	return nil
}

// StageByIndex returns stage i or nil.
func (d *Dockerfile) StageByIndex(i int) *DFStage {
	if i < 0 || i >= len(d.Stages) {
		return nil
	}
	return &d.Stages[i]
}

// LastStage returns the final stage (the run image).
func (d *Dockerfile) LastStage() *DFStage {
	if len(d.Stages) == 0 {
		return nil
	}
	return &d.Stages[len(d.Stages)-1]
}

// ─── parser ───────────────────────────────────────────────────────────────────

// ParseDockerfile reads and parses a Dockerfile, extracting the instructions
// that relay's buildpacks use. Unknown instructions are silently skipped.
func ParseDockerfile(path string) (*Dockerfile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	var cont string
	for scanner.Scan() {
		raw := scanner.Text()
		// Handle line continuations (backslash at end of line).
		if strings.HasSuffix(raw, "\\") {
			cont += strings.TrimSuffix(raw, "\\") + " "
			continue
		}
		lines = append(lines, cont+raw)
		cont = ""
	}
	if cont != "" {
		lines = append(lines, cont)
	}

	df := &Dockerfile{}
	var current *DFStage

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split into keyword + rest.
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		keyword := strings.ToUpper(strings.TrimSpace(parts[0]))
		rest := strings.TrimSpace(parts[1])

		switch DFInstructionKind(keyword) {
		case DFFrom:
			stage := parseFrom(rest)
			df.Stages = append(df.Stages, stage)
			current = &df.Stages[len(df.Stages)-1]

		case DFRun:
			if current == nil {
				continue
			}
			current.Instructions = append(current.Instructions, parseRun(rest))

		case DFCopy:
			if current == nil {
				continue
			}
			ins := parseCopy(rest)
			current.Instructions = append(current.Instructions, ins)

		case DFWorkdir:
			if current == nil {
				continue
			}
			current.Instructions = append(current.Instructions, DFInstruction{
				Kind: DFWorkdir,
				Path: rest,
			})

		case DFEnv:
			if current == nil {
				continue
			}
			k, v := parseEnv(rest)
			current.Instructions = append(current.Instructions, DFInstruction{
				Kind:   DFEnv,
				EnvKey: k,
				EnvVal: v,
			})

		case DFCmd:
			if current == nil {
				continue
			}
			current.Instructions = append(current.Instructions, DFInstruction{
				Kind: DFCmd,
				Cmd:  parseCmd(rest),
			})

		case DFExpose:
			if current == nil {
				continue
			}
			var port int
			fmt.Sscanf(rest, "%d", &port)
			current.Instructions = append(current.Instructions, DFInstruction{
				Kind: DFExpose,
				Port: port,
			})

		case DFArg:
			if current == nil {
				continue
			}
			name, def := parseArg(rest)
			current.Instructions = append(current.Instructions, DFInstruction{
				Kind:       DFArg,
				ArgName:    name,
				ArgDefault: def,
			})

		// USER and LABEL are parsed but not executed (not needed for relay builds).
		case DFUser, DFLabel:
			if current != nil {
				current.Instructions = append(current.Instructions, DFInstruction{
					Kind: DFInstructionKind(keyword),
				})
			}
		}
	}

	if len(df.Stages) == 0 {
		return nil, fmt.Errorf("no FROM instruction found in Dockerfile")
	}
	return df, scanner.Err()
}

// ─── instruction parsers ──────────────────────────────────────────────────────

func parseFrom(rest string) DFStage {
	// FROM <image> [AS <name>]
	fields := strings.Fields(rest)
	stage := DFStage{Image: fields[0]}
	for i := 0; i < len(fields)-1; i++ {
		if strings.ToUpper(fields[i]) == "AS" {
			stage.Name = fields[i+1]
			break
		}
	}
	return stage
}

func parseCopy(rest string) DFInstruction {
	ins := DFInstruction{Kind: DFCopy}
	tokens := splitCopyTokens(rest)

	// Extract --from=<stage> flag.
	var paths []string
	for _, t := range tokens {
		if strings.HasPrefix(t, "--from=") {
			ins.FromStage = strings.TrimPrefix(t, "--from=")
		} else if strings.HasPrefix(t, "--chown=") || strings.HasPrefix(t, "--chmod=") {
			// ignore
		} else {
			paths = append(paths, t)
		}
	}
	if len(paths) >= 2 {
		ins.Srcs = paths[:len(paths)-1]
		ins.Dest = paths[len(paths)-1]
	}
	return ins
}

func parseRun(rest string) DFInstruction {
	ins := DFInstruction{Kind: DFRun}
	opts, shell := splitLeadingRunOpts(rest)
	ins.Shell = shell
	for _, opt := range opts {
		if !strings.HasPrefix(opt, "--mount=") {
			continue
		}
		ins.RunMounts = append(ins.RunMounts, parseRunMount(strings.TrimPrefix(opt, "--mount=")))
	}
	return ins
}

func splitLeadingRunOpts(s string) ([]string, string) {
	s = strings.TrimSpace(s)
	var opts []string
	for strings.HasPrefix(s, "--") {
		token, rest := nextShellToken(s)
		if token == "" {
			break
		}
		opts = append(opts, token)
		s = strings.TrimSpace(rest)
	}
	return opts, s
}

func nextShellToken(s string) (string, string) {
	var (
		inSingle bool
		inDouble bool
		escaped  bool
	)
	for i, r := range s {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case (r == ' ' || r == '\t') && !inSingle && !inDouble:
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

func parseRunMount(spec string) DFRunMount {
	spec = strings.Trim(strings.TrimSpace(spec), `"'`)
	mount := DFRunMount{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, hasVal := strings.Cut(part, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if !hasVal {
			val = ""
		}
		switch key {
		case "type":
			mount.Type = val
		case "target", "dst", "destination":
			mount.Target = val
		case "source", "src":
			mount.Source = val
		case "id":
			mount.ID = val
		}
	}
	return mount
}

// splitCopyTokens splits COPY arguments respecting quoted strings.
func splitCopyTokens(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQ := false
	for _, ch := range s {
		switch {
		case ch == '"' && !inQ:
			inQ = true
		case ch == '"' && inQ:
			inQ = false
		case ch == ' ' && !inQ:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func parseEnv(rest string) (key, val string) {
	// ENV KEY=VALUE  or  ENV KEY VALUE  (old form)
	if idx := strings.Index(rest, "="); idx > 0 {
		key = rest[:idx]
		val = strings.Trim(rest[idx+1:], "\"")
	} else {
		parts := strings.SplitN(rest, " ", 2)
		key = parts[0]
		if len(parts) > 1 {
			val = strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(key), strings.TrimSpace(val)
}

func parseCmd(rest string) []string {
	rest = strings.TrimSpace(rest)
	// Exec form: ["node","server.js"]
	if strings.HasPrefix(rest, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(rest), &arr); err == nil {
			return arr
		}
	}
	// Shell form: sh -c <rest>
	return []string{"sh", "-c", rest}
}

func parseArg(rest string) (name, def string) {
	if idx := strings.Index(rest, "="); idx > 0 {
		return rest[:idx], rest[idx+1:]
	}
	return rest, ""
}

// ─── BuildManifest ────────────────────────────────────────────────────────────

// BuildManifest is written alongside the built rootfs so station run knows what
// command to execute and what port to expose. Mirrors what docker inspect gives.
type BuildManifest struct {
	Cmd        []string          `json:"cmd"`
	Entrypoint []string          `json:"entrypoint,omitempty"`
	Env        map[string]string `json:"env"`
	Port       int               `json:"port"`
	WorkDir    string            `json:"workdir"`
}

func saveManifest(dir string, m *BuildManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dir+"/station-manifest.json", data, 0644)
}

func loadManifest(dir string) (*BuildManifest, error) {
	data, err := os.ReadFile(dir + "/station-manifest.json")
	if err != nil {
		return nil, err
	}
	var m BuildManifest
	return &m, json.Unmarshal(data, &m)
}

