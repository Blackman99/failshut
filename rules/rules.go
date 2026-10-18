// Package rules implements static fail-open detectors.
package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Blackman99/failshut/discover"
	"github.com/Blackman99/failshut/report"
	"github.com/pelletier/go-toml/v2"
)

// Context carries discovery + optional project cwd for path rules.
type Context struct {
	Configs []discover.Config
	Project string // cwd / --project for path-traversal checks
	Home    string
}

// All returns every rule implementation.
func All() []Rule {
	return []Rule{
		DenyIgnored{},
		AskVsEffective{},
		HookMissing{},
		PathTraversalConfig{},
		DelegationEscalate{},
		AskAutoAllow{},
	}
}

// Rule is one static check.
type Rule interface {
	ID() string
	Run(ctx Context) ([]report.Finding, error)
}

// --- helpers ---

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func quoteEvidence(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	if max > 0 && len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func parseJSON(path string) (map[string]any, []byte, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, b, err
	}
	return m, b, nil
}

func parseTOML(path string) (map[string]any, []byte, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, nil, err
	}
	var m map[string]any
	if err := toml.Unmarshal(b, &m); err != nil {
		return nil, b, err
	}
	return m, b, nil
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func boolishFalse(v any) bool {
	switch t := v.(type) {
	case bool:
		return !t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "false" || s == "0" || s == "no" || s == "off"
	case float64:
		return t == 0
	case int64:
		return t == 0
	default:
		return false
	}
}

// DenyIgnored flags OpenCode deny/ask path rules that the runtime never matches,
// so the read or edit they were meant to stop goes through.
//
// OpenCode compares a rule's resource literally against the project-relative
// path the read/edit tool checks. A resource spelled "./Workshop/x.md" never
// matches "Workshop/x.md" (anomalyco/opencode#55728, reproduced by the
// maintainers' bot), and an absolute resource pointing inside the project does
// not match either (open fix anomalyco/opencode#53365). A plain tool-level
// deny such as {"read": "deny"} or a relative resource without "./" does work
// and is not reported.
type DenyIgnored struct{}

func (DenyIgnored) ID() string { return "deny-ignored" }

// opencodeRule is one v2 {action, resource, effect} permission rule.
type opencodeRule struct {
	Action, Resource, Effect string
}

func opencodeRules(v any) []opencodeRule {
	var out []opencodeRule
	for _, item := range asSlice(v) {
		m := asMap(item)
		if m == nil {
			continue
		}
		a, _ := m["action"].(string)
		r, _ := m["resource"].(string)
		e, _ := m["effect"].(string)
		if a == "" || e == "" {
			continue
		}
		if r == "" {
			r = "*"
		}
		out = append(out, opencodeRule{Action: strings.ToLower(a), Resource: r, Effect: strings.ToLower(e)})
	}
	return out
}

// opencodeProjectDir is the project an OpenCode config applies to, "" if unknown.
func opencodeProjectDir(project string) string {
	if project == "" {
		return ""
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return ""
	}
	return abs
}

func (DenyIgnored) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	for _, c := range ctx.Configs {
		if c.Harness != "opencode" {
			continue
		}
		m, _, err := parseJSON(c.Path)
		if err != nil || m == nil {
			continue
		}
		proj := opencodeProjectDir(ctx.Project)
		check := func(where string, rules []opencodeRule) {
			for i, r := range rules {
				if r.Action != "read" && r.Action != "edit" {
					continue
				}
				if r.Effect != "deny" && r.Effect != "ask" {
					continue
				}
				ev := fmt.Sprintf(`%s[%d] {action:%q, resource:%q, effect:%q}`, where, i, r.Action, r.Resource, r.Effect)
				switch {
				case strings.HasPrefix(r.Resource, "./"):
					out = append(out, report.Finding{
						ID: "deny-ignored", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
						Evidence: quoteEvidence(ev+" — a leading ./ never matches the project-relative path the tool checks", 240),
						Fix:      fmt.Sprintf("Drop the leading ./: \"resource\": %q (anomalyco/opencode#55728).", strings.TrimPrefix(r.Resource, "./")),
					})
				case proj != "" && filepath.IsAbs(r.Resource) && pathInside(r.Resource, proj):
					rel, _ := filepath.Rel(proj, r.Resource)
					out = append(out, report.Finding{
						ID: "deny-ignored", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
						Evidence: quoteEvidence(ev+" — absolute path inside the project; the tool checks the relative path", 240),
						Fix:      fmt.Sprintf("Use the project-relative form \"resource\": %q until anomalyco/opencode#53365 lands.", filepath.ToSlash(rel)),
					})
				}
			}
		}
		check("permissions", opencodeRules(m["permissions"]))
		for _, key := range []string{"agents", "agent"} {
			for name, av := range asMap(m[key]) {
				if am := asMap(av); am != nil {
					check(fmt.Sprintf("%s.%s.permissions", key, name), opencodeRules(am["permissions"]))
				}
			}
		}
	}
	return out, nil
}

// pathInside reports whether p is dir or below it (lexically, glob suffix ignored).
func pathInside(p, dir string) bool {
	if i := strings.IndexAny(p, "*?["); i >= 0 {
		p = p[:i]
	}
	rel, err := filepath.Rel(dir, filepath.Clean(p))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// AskVsEffective flags UI/config ask + effective sandbox_approval=false (Codex#54298/#54440).
type AskVsEffective struct{}

func (AskVsEffective) ID() string { return "ask-vs-effective" }

func (AskVsEffective) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	for _, c := range ctx.Configs {
		if c.Harness != "codex" {
			continue
		}
		var m map[string]any
		var err error
		if strings.HasSuffix(c.Path, ".toml") {
			m, _, err = parseTOML(c.Path)
		} else if strings.HasSuffix(c.Path, ".json") {
			m, _, err = parseJSON(c.Path)
		} else {
			continue
		}
		if err != nil || m == nil {
			continue
		}
		checkAskMismatch(c, m, "", &out)
		// named profiles
		if profiles := asMap(m["profiles"]); profiles != nil {
			for name, pv := range profiles {
				if pm := asMap(pv); pm != nil {
					checkAskMismatch(c, pm, "profiles."+name+".", &out)
				}
			}
		}
		if profiles := asMap(m["profile"]); profiles != nil {
			for name, pv := range profiles {
				if pm := asMap(pv); pm != nil {
					checkAskMismatch(c, pm, "profile."+name+".", &out)
				}
			}
		}
	}
	return out, nil
}

func checkAskMismatch(c discover.Config, m map[string]any, prefix string, out *[]report.Finding) {
	approval := strings.ToLower(asString(m["approval_policy"]))
	if approval == "" {
		approval = strings.ToLower(asString(m["approvalPolicy"]))
	}
	asks := approval == "ask" || approval == "untrusted" || approval == "on-request" || approval == "on_request"

	// sandbox_approval = false while UI asks
	sa, hasSA := m["sandbox_approval"]
	if !hasSA {
		sa, hasSA = m["sandboxApproval"]
	}
	if asks && hasSA && boolishFalse(sa) {
		*out = append(*out, report.Finding{
			ID: "ask-vs-effective", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
			Evidence: quoteEvidence(fmt.Sprintf("%sapproval_policy=%q but %ssandbox_approval=%v", prefix, approval, prefix, sa), 180),
			Fix:      "Set sandbox_approval=true (or remove the override) so Ask in the UI cannot be skipped by the sandbox path (codex#54298).",
		})
	}

	// require_escalated with silent sandbox / auto
	re, hasRE := m["require_escalated"]
	if !hasRE {
		re, hasRE = m["requireEscalated"]
	}
	sandbox := strings.ToLower(asString(m["sandbox_mode"]))
	if sandbox == "" {
		sandbox = strings.ToLower(asString(m["sandboxMode"]))
	}
	if hasRE && !boolishFalse(re) { // require_escalated true
		if sandbox == "danger-full-access" || sandbox == "danger_full_access" || sandbox == "disabled" || sandbox == "none" {
			*out = append(*out, report.Finding{
				ID: "ask-vs-effective", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
				Evidence: quoteEvidence(fmt.Sprintf("%srequire_escalated=%v with sandbox_mode=%q", prefix, re, sandbox), 180),
				Fix:      "require_escalated is ineffective when sandbox_mode disables the sandbox; tighten sandbox_mode (codex#54440).",
			})
		}
	}

	// approval ask but auto-approve style flags
	for _, key := range []string{"auto_approve", "autoApprove", "skip_approvals", "skipApprovals", "never_ask", "neverAsk"} {
		if v, ok := m[key]; ok && !boolishFalse(v) && asks {
			*out = append(*out, report.Finding{
				ID: "ask-vs-effective", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
				Evidence: quoteEvidence(fmt.Sprintf("%sapproval_policy=%q with %s%s=%v", prefix, approval, prefix, key, v), 180),
				Fix:      fmt.Sprintf("Disable %s or change approval_policy so Ask cannot be auto-answered.", key),
			})
		}
	}
}

// HookMissing flags hook commands whose script is missing or cannot be run.
//
// Claude Code treats a hook that exits with any code other than 2 as a
// non-blocking error and lets the tool call proceed, so a hook whose script is
// gone fails open (claude-code#32990, #102649). The command string is parsed
// like a shell would (quotes, ~, $HOME, $CLAUDE_PROJECT_DIR, interpreter
// prefixes such as "bash", "python3", "node"), never executed.
type HookMissing struct{}

func (HookMissing) ID() string { return "hook-missing" }

// Events where Claude Code treats exit 2 as a block. Stop-family events are
// excluded on purpose: their "missing script" heuristic downgrades exit 2 with
// "No such file" / "can't open" stderr to non-blocking (claude-code#99171).
var exit2BlockingEvents = map[string]bool{"PreToolUse": true, "PermissionRequest": true, "UserPromptSubmit": true}

// Interpreters that exit 2 when the script file cannot be opened.
func exitsTwoOnMissingScript(interp string) bool {
	return strings.HasPrefix(interp, "python") || interp == "perl"
}

type hookScope struct {
	cfg        discover.Config
	env        map[string]string
	relBase    string // directory relative script paths resolve against ("" = unknown)
	configBase string
}

func (HookMissing) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	home := ctx.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	for _, c := range ctx.Configs {
		if c.Harness != "claude-code" && c.Harness != "opencode" && c.Harness != "cursor" {
			continue
		}
		m, _, err := parseJSON(c.Path)
		if err != nil || m == nil {
			continue
		}
		sc := hookScope{cfg: c, env: map[string]string{}, configBase: filepath.Dir(c.Path)}
		if home != "" {
			sc.env["HOME"] = home
		}
		projectDir := ""
		if c.Harness == "claude-code" && (c.Kind == "project" || c.Kind == "project.local") {
			// <project>/.claude/settings.json → hooks run with the project as cwd.
			projectDir = filepath.Dir(filepath.Dir(c.Path))
		} else if ctx.Project != "" {
			projectDir = ctx.Project
		}
		if projectDir != "" {
			if abs, err := filepath.Abs(projectDir); err == nil {
				projectDir = abs
			}
			sc.env["CLAUDE_PROJECT_DIR"] = projectDir
		}
		if c.Harness == "claude-code" {
			sc.relBase = projectDir
		} else {
			sc.relBase = sc.configBase
		}
		collectHooks(sc, "", m["hooks"], &out)
		if plugins := asMap(m["plugins"]); plugins != nil {
			for name, pv := range plugins {
				if s, ok := pv.(string); ok {
					checkHookCommand(sc, "", fmt.Sprintf("plugins.%s", name), s, &out)
				}
				if pm := asMap(pv); pm != nil {
					for _, k := range []string{"path", "command", "script", "entry"} {
						if s, ok := pm[k].(string); ok && s != "" {
							checkHookCommand(sc, "", fmt.Sprintf("plugins.%s.%s", name, k), s, &out)
						}
					}
				}
			}
		}
	}
	return out, nil
}

func collectHooks(sc hookScope, event string, v any, out *[]report.Finding) {
	switch t := v.(type) {
	case map[string]any:
		for ev, sub := range t {
			collectHooks(sc, ev, sub, out)
		}
	case []any:
		for _, item := range t {
			if m := asMap(item); m != nil {
				if typ, _ := m["type"].(string); typ != "" && typ != "command" {
					continue // prompt / agent / http hooks have no script file
				}
				for _, k := range []string{"command", "path", "script"} {
					if s, ok := m[k].(string); ok && s != "" {
						checkHookCommand(sc, event, k, s, out)
					}
				}
				if nested := m["hooks"]; nested != nil {
					collectHooks(sc, event, nested, out)
				}
			} else if s, ok := item.(string); ok {
				checkHookCommand(sc, event, "hooks", s, out)
			}
		}
	case string:
		checkHookCommand(sc, event, "hooks", t, out)
	}
}

func checkHookCommand(sc hookScope, event, field, cmd string, out *[]report.Finding) {
	inv, ok := parseHookCommand(cmd, sc.env)
	if !ok {
		return // bare command on PATH, inline code, or not statically resolvable
	}
	resolved := inv.Script
	if !filepath.IsAbs(resolved) {
		if sc.relBase == "" {
			return // relative to an unknown session cwd: do not guess
		}
		resolved = filepath.Join(sc.relBase, resolved)
	}
	label := field
	if event != "" {
		label = event + " " + field
	}
	st, err := os.Stat(resolved)
	if err != nil || st.IsDir() {
		f := report.Finding{
			ID: "hook-missing", Harness: sc.cfg.Harness, File: sc.cfg.Path, Severity: report.FailOpen,
			Evidence: quoteEvidence(fmt.Sprintf("%s %q runs %s — missing", label, cmd, resolved), 240),
			Fix:      "Restore the hook script or fix the path. A hook that cannot start exits non-2, and Claude Code lets the tool call through.",
		}
		if sc.cfg.Harness == "claude-code" && exitsTwoOnMissingScript(inv.Interpreter) && exit2BlockingEvents[event] {
			// python3/perl exit 2 on a missing script, which Claude Code treats as a block:
			// the gate is broken shut (every matching call blocked), not open.
			f.Severity = report.FailShut
			f.Evidence = quoteEvidence(fmt.Sprintf("%s %q runs %s — missing; %s exits 2, so matching calls are blocked, not allowed", label, cmd, resolved, inv.Interpreter), 240)
			f.Fix = "Restore the hook script; until then every matching tool call is blocked."
		}
		*out = append(*out, f)
		return
	}
	if inv.Interpreter == "" && runtime.GOOS != "windows" && st.Mode()&0o111 == 0 {
		*out = append(*out, report.Finding{
			ID: "hook-missing", Harness: sc.cfg.Harness, File: sc.cfg.Path, Severity: report.FailOpen,
			Evidence: quoteEvidence(fmt.Sprintf("%s %q runs %s directly but it is not executable (mode %s)", label, cmd, resolved, st.Mode()), 240),
			Fix:      "chmod +x the hook script, or invoke it through its interpreter (bash/python3/node).",
		})
	}
}

// PathTraversalConfig flags project/session paths with ".." that can load sibling opencode.json (opencode#55909).
type PathTraversalConfig struct{}

func (PathTraversalConfig) ID() string { return "path-traversal-config" }

func hasDotDotSegment(p string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(p), "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

func (PathTraversalConfig) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	// Only flag --project when it still contains a ".." segment (caller passed a walking path).
	if ctx.Project != "" && hasDotDotSegment(ctx.Project) {
		// Prefer absolute form in evidence when available.
		out = append(out, report.Finding{
			ID: "path-traversal-config", Harness: "opencode", File: ctx.Project, Severity: report.FailOpen,
			Evidence: quoteEvidence(fmt.Sprintf("project path contains '..' segment: %s", ctx.Project), 180),
			Fix:      "Use a canonical project path without '..'; OpenCode may resolve a sibling opencode.json (opencode#55909).",
		})
	}
	for _, c := range ctx.Configs {
		if c.Harness != "opencode" {
			continue
		}
		if hasDotDotSegment(c.Path) {
			out = append(out, report.Finding{
				ID: "path-traversal-config", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
				Evidence: quoteEvidence(fmt.Sprintf("config path contains '..' segment: %s", c.Path), 180),
				Fix:      "Relocate or canonicalize the path so '..' cannot walk into a sibling config (opencode#55909).",
			})
		}
		m, raw, err := parseJSON(c.Path)
		if err != nil || m == nil {
			continue
		}
		_ = raw
		for _, key := range []string{"root", "project", "cwd", "workspace", "sessionDir", "session_dir", "directory"} {
			if v, ok := m[key]; ok {
				s := asString(v)
				if hasDotDotSegment(s) {
					out = append(out, report.Finding{
						ID: "path-traversal-config", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
						Evidence: quoteEvidence(fmt.Sprintf("%s=%q contains '..'", key, s), 180),
						Fix:      "Replace '..' segments with an absolute canonical path; sibling opencode.json may load instead (opencode#55909).",
					})
				}
			}
		}
	}
	return out, nil
}

// DelegationEscalate flags plan/read-only or named restrictive profiles that can spawn
// general/write-capable subagents without re-applying the same profile (opencode#55814, codex#54448).
type DelegationEscalate struct{}

func (DelegationEscalate) ID() string { return "delegation-escalate" }

func (DelegationEscalate) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	for _, c := range ctx.Configs {
		switch c.Harness {
		case "opencode", "claude-code":
			m, _, err := parseJSON(c.Path)
			if err != nil || m == nil {
				continue
			}
			if c.Harness == "opencode" {
				out = append(out, opencodeSubagentEscalation(c, m)...)
			}
			mode := strings.ToLower(asString(m["permissionMode"]))
			if mode == "" {
				mode = strings.ToLower(asString(m["mode"]))
			}
			planLike := mode == "plan" || mode == "read-only" || mode == "readonly" || mode == "ask"
			if agents := asMap(m["agents"]); agents != nil && planLike {
				for name, av := range agents {
					am := asMap(av)
					agentMode := ""
					if am != nil {
						agentMode = strings.ToLower(asString(am["mode"]))
						if agentMode == "" {
							agentMode = strings.ToLower(asString(am["permissionMode"]))
						}
					}
					if agentMode == "" || agentMode == "default" || agentMode == "general" || agentMode == "write" || agentMode == "acceptedits" {
						out = append(out, report.Finding{
							ID: "delegation-escalate", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
							Evidence: quoteEvidence(fmt.Sprintf("permissionMode=%q but agents.%s mode=%q", mode, name, agentMode), 180),
							Fix:      "Pin subagent permissionMode to the same restrictive mode; plan→general escalates via subagent (opencode#55814).",
						})
					}
				}
			}
			// allow_agents / subagents without profile inheritance
			if planLike {
				if v, ok := m["allow_subagents"]; ok && !boolishFalse(v) {
					out = append(out, report.Finding{
						ID: "delegation-escalate", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
						Evidence: quoteEvidence(fmt.Sprintf("permissionMode=%q with allow_subagents=%v", mode, v), 180),
						Fix:      "Disable subagents in plan/read-only mode or force inherited deny/ask on every spawned agent.",
					})
				}
			}
		case "codex":
			m, _, err := parseTOML(c.Path)
			if err != nil {
				m, _, err = parseJSON(c.Path)
			}
			if err != nil || m == nil {
				continue
			}
			// named restrictive profile + delegation without profile
			profiles := asMap(m["profiles"])
			if profiles == nil {
				profiles = asMap(m["profile"])
			}
			defaultProfile := asString(m["profile"])
			if defaultProfile == "" {
				defaultProfile = asString(m["default_profile"])
			}
			if profiles != nil && defaultProfile != "" && defaultProfile != "<nil>" {
				if pm := asMap(profiles[defaultProfile]); pm != nil {
					ap := strings.ToLower(asString(pm["approval_policy"]))
					restrictive := ap == "ask" || ap == "untrusted" || ap == "never"
					deleg := m["delegate"]
					if deleg == nil {
						deleg = m["delegation"]
					}
					if restrictive && deleg != nil {
						dm := asMap(deleg)
						inherit := true
						if dm != nil {
							if _, ok := dm["profile"]; !ok {
								inherit = false
							}
							if p := asString(dm["profile"]); p == "" || p == "<nil>" {
								inherit = false
							}
						} else if !boolishFalse(deleg) {
							inherit = false
						}
						if !inherit {
							out = append(out, report.Finding{
								ID: "delegation-escalate", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
								Evidence: quoteEvidence(fmt.Sprintf("default profile %q is restrictive but delegation does not pin profile", defaultProfile), 180),
								Fix:      "Set delegation.profile to the same named profile so delegates cannot drop Ask (codex#54448).",
							})
						}
					}
				}
			}
		}
	}
	return out, nil
}

// opencodeSubagentEscalation flags OpenCode v2 agents that deny edit but do not
// deny launching the write-capable "general" subagent. A child session runs
// with its own policy, so the parent's edit deny does not carry over
// (anomalyco/opencode#55814; the shipped plan agent denies edit but not subagent).
func opencodeSubagentEscalation(c discover.Config, m map[string]any) []report.Finding {
	var out []report.Finding
	for _, key := range []string{"agents", "agent"} {
		for name, av := range asMap(m[key]) {
			am := asMap(av)
			if am == nil {
				continue
			}
			rules := opencodeRules(am["permissions"])
			if _, isArray := am["permissions"].([]any); !isArray && name != "plan" {
				continue
			}
			editDenied := name == "plan" // shipped plan policy denies edit
			if e, ok := lastMatch(rules, "edit", "*"); ok {
				editDenied = e == "deny"
			}
			if !editDenied {
				continue
			}
			if e, ok := lastMatch(rules, "subagent", "general"); ok && e == "deny" {
				continue
			}
			out = append(out, report.Finding{
				ID: "delegation-escalate", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
				Evidence: quoteEvidence(fmt.Sprintf("%s.%s denies edit but does not deny subagent \"general\", which runs with its own (write-capable) policy", key, name), 240),
				Fix:      fmt.Sprintf(`Add to %s.%s.permissions: {"action":"subagent","resource":"*","effect":"deny"} then {"action":"subagent","resource":"explore","effect":"allow"} (anomalyco/opencode#55814).`, key, name),
			})
		}
	}
	return out
}

// lastMatch returns the effect of the last rule for action whose resource glob
// matches res (OpenCode evaluates rules last-match-wins).
func lastMatch(rules []opencodeRule, action, res string) (string, bool) {
	effect, found := "", false
	for _, r := range rules {
		if r.Action != action {
			continue
		}
		if ok, _ := path.Match(r.Resource, res); ok || r.Resource == "*" {
			effect, found = r.Effect, true
		}
	}
	return effect, found
}

// AskAutoAllow flags hooks that return ask paired with desktop/auto-allow (claude-code#102582).
type AskAutoAllow struct{}

func (AskAutoAllow) ID() string { return "ask-auto-allow" }

func (AskAutoAllow) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	for _, c := range ctx.Configs {
		if c.Harness != "claude-code" {
			continue
		}
		m, _, err := parseJSON(c.Path)
		if err != nil || m == nil {
			continue
		}
		hasAskHook := hooksMentionAsk(m["hooks"])
		auto := false
		var autoEv string
		for _, key := range []string{"autoAllow", "auto_allow", "desktopAutoAllow", "desktop_auto_allow", "autoApproveHooks", "auto_approve_hooks"} {
			if v, ok := m[key]; ok && !boolishFalse(v) {
				auto = true
				autoEv = fmt.Sprintf("%s=%v", key, v)
				break
			}
		}
		// nested desktop / app settings
		for _, nest := range []string{"desktop", "app", "ui"} {
			if nm := asMap(m[nest]); nm != nil {
				for _, key := range []string{"autoAllow", "auto_allow", "autoApprove", "auto_approve", "skipHookPrompts"} {
					if v, ok := nm[key]; ok && !boolishFalse(v) {
						auto = true
						autoEv = fmt.Sprintf("%s.%s=%v", nest, key, v)
					}
				}
			}
		}
		if hasAskHook && auto {
			out = append(out, report.Finding{
				ID: "ask-auto-allow", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
				Evidence: quoteEvidence(fmt.Sprintf("hooks request ask but %s answers without the user", autoEv), 180),
				Fix:      "Disable desktop/auto-allow so hook 'ask' decisions surface to you (claude-code#102582).",
			})
		}
		// permissionMode ask + allow list that includes dangerous tools with auto
		pm := strings.ToLower(asString(m["permissionMode"]))
		if (pm == "ask" || pm == "default") && auto {
			out = append(out, report.Finding{
				ID: "ask-auto-allow", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
				Evidence: quoteEvidence(fmt.Sprintf("permissionMode=%q with %s", pm, autoEv), 180),
				Fix:      "Turn off auto-allow / desktop auto-approve when permissionMode expects human confirmation.",
			})
		}
	}
	return out, nil
}

func hooksMentionAsk(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		for _, ev := range t {
			if hooksMentionAsk(ev) {
				return true
			}
		}
	case []any:
		for _, item := range t {
			if hooksMentionAsk(item) {
				return true
			}
		}
	case string:
		return strings.Contains(strings.ToLower(t), "ask")
	}
	if m := asMap(v); m != nil {
		if d := strings.ToLower(asString(m["decision"])); d == "ask" {
			return true
		}
		if d := strings.ToLower(asString(m["permissionDecision"])); d == "ask" {
			return true
		}
		for _, k := range []string{"command", "prompt"} {
			if strings.Contains(strings.ToLower(asString(m[k])), "ask") {
				return true
			}
		}
		if hooksMentionAsk(m["hooks"]) {
			return true
		}
	}
	return false
}
