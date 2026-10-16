// Package discover locates AI coding-agent permission configs on disk.
package discover

import (
	"os"
	"path/filepath"
	"strings"
)

// Config is one discovered harness config file.
type Config struct {
	Harness string // claude-code | codex | opencode | cursor
	Path    string
	Kind    string // settings | settings.local | project | config.toml | opencode.json | ...
}

// Options control discovery.
type Options struct {
	Home    string // defaults to os.UserHomeDir
	Project string // optional project directory
}

// Discover finds known config paths (best-effort; missing files are skipped).
func Discover(opt Options) ([]Config, error) {
	home := opt.Home
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}
	var out []Config

	// Claude Code
	for _, p := range []struct{ path, kind string }{
		{filepath.Join(home, ".claude", "settings.json"), "settings"},
		{filepath.Join(home, ".claude", "settings.local.json"), "settings.local"},
	} {
		if fileExists(p.path) {
			out = append(out, Config{Harness: "claude-code", Path: p.path, Kind: p.kind})
		}
	}
	if opt.Project != "" {
		proj := filepath.Join(opt.Project, ".claude", "settings.json")
		if fileExists(proj) {
			out = append(out, Config{Harness: "claude-code", Path: proj, Kind: "project"})
		}
		projLocal := filepath.Join(opt.Project, ".claude", "settings.local.json")
		if fileExists(projLocal) {
			out = append(out, Config{Harness: "claude-code", Path: projLocal, Kind: "project.local"})
		}
	}

	// Codex
	codexDir := filepath.Join(home, ".codex")
	for _, name := range []string{"config.toml", "config.yaml", "config.yml", "config.json"} {
		p := filepath.Join(codexDir, name)
		if fileExists(p) {
			out = append(out, Config{Harness: "codex", Path: p, Kind: name})
		}
	}
	// sibling profiles / auth configs sometimes sit next to config.toml
	if entries, err := os.ReadDir(codexDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			if strings.HasPrefix(n, "config.") && !strings.HasSuffix(n, ".toml") &&
				!strings.HasSuffix(n, ".yaml") && !strings.HasSuffix(n, ".yml") &&
				!strings.HasSuffix(n, ".json") {
				continue
			}
			if strings.HasSuffix(n, ".toml") && n != "config.toml" {
				p := filepath.Join(codexDir, n)
				out = append(out, Config{Harness: "codex", Path: p, Kind: n})
			}
		}
	}

	// OpenCode
	candidates := []string{}
	if opt.Project != "" {
		candidates = append(candidates,
			filepath.Join(opt.Project, "opencode.json"),
			filepath.Join(opt.Project, ".opencode", "config.json"),
			filepath.Join(opt.Project, ".opencode", "opencode.json"),
			filepath.Join(opt.Project, ".opencode.json"),
		)
	}
	candidates = append(candidates,
		filepath.Join(home, ".config", "opencode", "opencode.json"),
		filepath.Join(home, ".config", "opencode", "config.json"),
		filepath.Join(home, ".opencode", "opencode.json"),
		filepath.Join(home, ".opencode", "config.json"),
	)
	seen := map[string]bool{}
	for _, p := range candidates {
		if seen[p] || !fileExists(p) {
			continue
		}
		seen[p] = true
		kind := "opencode.json"
		if strings.Contains(p, ".opencode") {
			kind = "project-opencode"
		}
		out = append(out, Config{Harness: "opencode", Path: p, Kind: kind})
	}

	// Cursor (best-effort readable)
	cursorPaths := []string{
		filepath.Join(home, ".cursor", "cli-config.json"),
		filepath.Join(home, ".cursor", "permissions.json"),
	}
	if opt.Project != "" {
		cursorPaths = append(cursorPaths, filepath.Join(opt.Project, ".cursor", "cli.json"))
	}
	for _, p := range cursorPaths {
		if fileExists(p) {
			out = append(out, Config{Harness: "cursor", Path: p, Kind: filepath.Base(p)})
		}
	}

	return out, nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
