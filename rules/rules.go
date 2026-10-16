// Package rules implements static fail-open detectors.
package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// DenyIgnored flags OpenCode-style deny rules on read tools that harness quirks ignore.
// Mirrors anomalyco/opencode#55728 shape: permission.deny includes read/grep/glob-like tools
// that the runtime still allows.
type DenyIgnored struct{}

func (DenyIgnored) ID() string { return "deny-ignored" }

var ignoredDenyTools = regexp.MustCompile(`(?i)^(read|grep|glob|list|ls|search|file_read|view)$`)

func (DenyIgnored) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	for _, c := range ctx.Configs {
		if c.Harness != "opencode" && c.Harness != "claude-code" {
			continue
		}
		m, raw, err := parseJSON(c.Path)
		if err != nil || m == nil {
			continue
		}
		// OpenCode: permission.deny / permissions.deny arrays or map
		for _, key := range []string{"permission", "permissions"} {
			perm := asMap(m[key])
			if perm == nil {
				continue
			}
			for _, dk := range []string{"deny", "blocked", "block"} {
				findDenyHits(c, perm[dk], raw, &out)
			}
			// also: permission: { "read": "deny" } style
			for tool, val := range perm {
				vs := strings.ToLower(asString(val))
				if (vs == "deny" || vs == "blocked" || vs == "false") && ignoredDenyTools.MatchString(tool) {
					out = append(out, report.Finding{
						ID: "deny-ignored", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
						Evidence: quoteEvidence(fmt.Sprintf(`%s.%s = %q`, key, tool, asString(val)), 180),
						Fix:      "Move deny off read-only tools that OpenCode may ignore; deny write/edit/bash instead, or upgrade past the deny-ignored bug (opencode#55728).",
					})
				}
			}
		}
		// Claude-style top-level deny arrays
		findDenyHits(c, m["deny"], raw, &out)
		if pm := asMap(m["permissions"]); pm != nil {
			findDenyHits(c, pm["deny"], raw, &out)
		}
	}
	return out, nil
}

func findDenyHits(c discover.Config, v any, raw []byte, out *[]report.Finding) {
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			s := asString(item)
			tool := firstToken(s)
			if ignoredDenyTools.MatchString(tool) || ignoredDenyTools.MatchString(s) {
				*out = append(*out, report.Finding{
					ID: "deny-ignored", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
					Evidence: quoteEvidence(fmt.Sprintf("deny entry %q", s), 180),
					Fix:      "Deny rules on read/grep/glob-like tools are ignored by some harness versions; deny mutating tools or pin a fixed release (opencode#55728).",
				})
			}
		}
	case map[string]any:
		for tool, val := range t {
			vs := strings.ToLower(asString(val))
			if ignoredDenyTools.MatchString(tool) && (vs == "deny" || vs == "true" || vs == "blocked" || vs == "ask") {
				// ask on read is ok-ish; only flag deny/blocked
				if vs == "ask" {
					continue
				}
				*out = append(*out, report.Finding{
					ID: "deny-ignored", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
					Evidence: quoteEvidence(fmt.Sprintf("deny[%q]=%v", tool, val), 180),
					Fix:      "Replace ignored read-tool denies with write/bash denies, or verify runtime still enforces after upgrade.",
				})
			}
		}
	}
	_ = raw
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	for _, sep := range []string{" ", ":", "/", "("} {
		if i := strings.Index(s, sep); i > 0 {
			return s[:i]
		}
	}
	return s
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

// HookMissing flags referenced hook/plugin paths that are missing or not executable.
type HookMissing struct{}

func (HookMissing) ID() string { return "hook-missing" }

func (HookMissing) Run(ctx Context) ([]report.Finding, error) {
	var out []report.Finding
	for _, c := range ctx.Configs {
		if c.Harness != "claude-code" && c.Harness != "opencode" && c.Harness != "cursor" {
			continue
		}
		m, _, err := parseJSON(c.Path)
		if err != nil || m == nil {
			continue
		}
		baseDir := filepath.Dir(c.Path)
		collectHooks(c, m["hooks"], baseDir, &out)
		if plugins := asMap(m["plugins"]); plugins != nil {
			for name, pv := range plugins {
				checkHookPath(c, fmt.Sprintf("plugins.%s", name), asString(pv), baseDir, &out)
				if pm := asMap(pv); pm != nil {
					for _, k := range []string{"path", "command", "script", "entry"} {
						if p := asString(pm[k]); p != "" && p != "<nil>" {
							checkHookPath(c, fmt.Sprintf("plugins.%s.%s", name, k), p, baseDir, &out)
						}
					}
				}
			}
		}
		// Claude Code format: hooks.PreToolUse = [{hooks:[{command:...}]}] or hooks.PreToolUse = [{command:...}]
	}
	return out, nil
}

func collectHooks(c discover.Config, v any, baseDir string, out *[]report.Finding) {
	switch t := v.(type) {
	case map[string]any:
		for event, ev := range t {
			collectHooks(c, ev, baseDir, out)
			_ = event
		}
	case []any:
		for _, item := range t {
			if m := asMap(item); m != nil {
				for _, k := range []string{"command", "path", "script", "matcher"} {
					// matcher is not a path
					if k == "matcher" {
						continue
					}
					if p := asString(m[k]); p != "" && p != "<nil>" {
						checkHookPath(c, k, p, baseDir, out)
					}
				}
				if nested := m["hooks"]; nested != nil {
					collectHooks(c, nested, baseDir, out)
				}
			} else if s, ok := item.(string); ok {
				checkHookPath(c, "hooks", s, baseDir, out)
			}
		}
	case string:
		checkHookPath(c, "hooks", t, baseDir, out)
	}
}

func checkHookPath(c discover.Config, field, p, baseDir string, out *[]report.Finding) {
	p = strings.TrimSpace(p)
	if p == "" || p == "<nil>" {
		return
	}
	// skip inline shell one-liners / JSON payloads
	if strings.ContainsAny(p, "\n|$") || strings.HasPrefix(p, "{") {
		return
	}
	// only look like paths
	if !strings.Contains(p, "/") && !strings.HasSuffix(p, ".sh") && !strings.HasSuffix(p, ".js") &&
		!strings.HasSuffix(p, ".mjs") && !strings.HasSuffix(p, ".py") && !strings.HasSuffix(p, ".ts") {
		// bare command names are ok
		if !strings.HasPrefix(p, ".") {
			return
		}
	}
	resolved := p
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		resolved = filepath.Join(home, p[2:])
	} else if !filepath.IsAbs(p) {
		resolved = filepath.Join(baseDir, p)
	}
	st, err := os.Stat(resolved)
	if err != nil {
		*out = append(*out, report.Finding{
			ID: "hook-missing", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
			Evidence: quoteEvidence(fmt.Sprintf("%s references %q (resolved %q) — missing", field, p, resolved), 180),
			Fix:      "Create the hook script or fix the path; a missing hook means the deny/ask gate never runs.",
		})
		return
	}
	if st.IsDir() {
		return
	}
	// executable bit (unix)
	if st.Mode()&0o111 == 0 && (strings.HasSuffix(resolved, ".sh") || strings.HasPrefix(p, "./") || strings.HasPrefix(p, "/")) {
		*out = append(*out, report.Finding{
			ID: "hook-missing", Harness: c.Harness, File: c.Path, Severity: report.FailOpen,
			Evidence: quoteEvidence(fmt.Sprintf("%s path %q exists but is not executable (mode %s)", field, resolved, st.Mode()), 180),
			Fix:      "chmod +x the hook script so the harness can invoke it.",
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
