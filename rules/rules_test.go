package rules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Blackman99/failshut/discover"
	"github.com/Blackman99/failshut/report"
	"github.com/Blackman99/failshut/rules"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func load(t *testing.T, home, project string) rules.Context {
	t.Helper()
	cfgs, err := discover.Discover(discover.Options{Home: home, Project: project})
	if err != nil {
		t.Fatal(err)
	}
	return rules.Context{Configs: cfgs, Project: project, Home: home}
}

func hasID(fs []report.Finding, id string) bool {
	for _, f := range fs {
		if f.ID == id && f.Severity == report.FailOpen {
			return true
		}
	}
	return false
}

func TestDenyIgnored(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "deny-ignored")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	fs, err := (rules.DenyIgnored{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(fs, "deny-ignored") {
		t.Fatalf("expected deny-ignored, got %#v", fs)
	}
}

func TestAskVsEffective(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "ask-vs-effective")
	ctx := load(t, filepath.Join(root, "home"), "")
	fs, err := (rules.AskVsEffective{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(fs, "ask-vs-effective") {
		t.Fatalf("expected ask-vs-effective, got %#v", fs)
	}
}

func TestHookMissing(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "hook-missing")
	ctx := load(t, filepath.Join(root, "home"), "")
	fs, err := (rules.HookMissing{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(fs, "hook-missing") {
		t.Fatalf("expected hook-missing, got %#v", fs)
	}
}

func TestPathTraversalConfig(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "path-traversal-config")
	proj := filepath.Join(root, "project")
	ctx := load(t, filepath.Join(root, "home"), proj)
	fs, err := (rules.PathTraversalConfig{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(fs, "path-traversal-config") {
		t.Fatalf("expected path-traversal-config, got %#v", fs)
	}
}

func TestDelegationEscalate(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "delegation-escalate")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	fs, err := (rules.DelegationEscalate{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(fs, "delegation-escalate") {
		t.Fatalf("expected delegation-escalate, got %#v", fs)
	}
}

func TestAskAutoAllow(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "ask-auto-allow")
	ctx := load(t, filepath.Join(root, "home"), "")
	fs, err := (rules.AskAutoAllow{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(fs, "ask-auto-allow") {
		t.Fatalf("expected ask-auto-allow, got %#v", fs)
	}
}

func TestCleanFailShut(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "clean")
	home := filepath.Join(root, "home")
	proj := filepath.Join(root, "project")
	ctx := load(t, home, proj)
	for _, r := range rules.All() {
		fs, err := r.Run(ctx)
		if err != nil {
			t.Fatalf("%s: %v", r.ID(), err)
		}
		for _, f := range fs {
			if f.Severity == report.FailOpen {
				t.Fatalf("clean fixture should not fail open on %s: %#v", r.ID(), f)
			}
		}
	}
}

func TestDiscoverSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	cfgs, err := discover.Discover(discover.Options{Home: dir, Project: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfgs) != 0 {
		t.Fatalf("expected empty, got %v", cfgs)
	}
	_ = os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
}

func run(t *testing.T, r rules.Rule, ctx rules.Context) []report.Finding {
	t.Helper()
	fs, err := r.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func countSev(fs []report.Finding, sev report.Severity) int {
	n := 0
	for _, f := range fs {
		if f.Severity == sev {
			n++
		}
	}
	return n
}

// claude-code#32990: project hook via "$CLAUDE_PROJECT_DIR"/..., script deleted.
func TestHookMissing32990ProjectDir(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "hook-missing-32990")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	fs := run(t, rules.HookMissing{}, ctx)
	if countSev(fs, report.FailOpen) != 1 {
		t.Fatalf("want 1 FAIL OPEN, got %#v", fs)
	}
	want := filepath.Join(root, "project", ".claude", "hooks", "block-src-writes.sh")
	if !strings.Contains(fs[0].Evidence, want) {
		t.Fatalf("evidence should name resolved script %s: %s", want, fs[0].Evidence)
	}
}

// claude-code#102649: "bash ~/.claude/hooks/x.sh" and "bash $HOME/..." with the scripts gone.
func TestHookMissing102649InterpreterPrefix(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "hook-missing-102649")
	ctx := load(t, filepath.Join(root, "home"), "")
	fs := run(t, rules.HookMissing{}, ctx)
	if countSev(fs, report.FailOpen) != 2 {
		t.Fatalf("want 2 FAIL OPEN, got %#v", fs)
	}
	for _, f := range fs {
		if strings.Contains(f.Evidence, "bash ~") && !strings.Contains(f.Evidence, filepath.Join(root, "home", ".claude", "hooks", "guard-bash.sh")) {
			t.Fatalf("interpreter prefix not stripped: %s", f.Evidence)
		}
	}
}

// Existing scripts behind interpreter prefixes, quotes, $HOME, ${HOME}, ~ and
// $CLAUDE_PROJECT_DIR must all be SEAL; bare commands, pipelines and prompt
// hooks are not file references.
func TestHookMissingSealedForms(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "hook-missing-sealed")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	if fs := run(t, rules.HookMissing{}, ctx); len(fs) != 0 {
		t.Fatalf("want no findings, got %#v", fs)
	}
}

// Without a project, a user-scope $CLAUDE_PROJECT_DIR hook is unknowable: no guess.
func TestHookMissingUnknownProjectDirSkipped(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "hook-missing-sealed")
	ctx := load(t, filepath.Join(root, "home"), "")
	if fs := run(t, rules.HookMissing{}, ctx); len(fs) != 0 {
		t.Fatalf("want no findings, got %#v", fs)
	}
}

func TestHookMissingExit2Interpreters(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "hook-missing-exit2")
	ctx := load(t, filepath.Join(root, "home"), "")
	fs := run(t, rules.HookMissing{}, ctx)
	if countSev(fs, report.FailOpen) != 1 || countSev(fs, report.FailShut) != 1 {
		t.Fatalf("want Stop=FAIL OPEN and PreToolUse=FAIL SHUT, got %#v", fs)
	}
	for _, f := range fs {
		if f.Severity == report.FailOpen && !strings.HasPrefix(f.Evidence, "Stop ") {
			t.Fatalf("FAIL OPEN should be the Stop hook: %s", f.Evidence)
		}
	}
}

func TestHookMissingNotExecutableDirect(t *testing.T) {
	home := t.TempDir()
	hooks := filepath.Join(home, ".claude", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "g.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"~/.claude/hooks/g.sh"},{"type":"command","command":"sh ~/.claude/hooks/g.sh"}]}]}}`
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := run(t, rules.HookMissing{}, load(t, home, ""))
	if len(fs) != 1 || !strings.Contains(fs[0].Evidence, "not executable") {
		t.Fatalf("want exactly the direct call flagged as not executable, got %#v", fs)
	}
}

// opencode#55728: tool-level deny and relative resources work → no finding.
func TestDenyIgnoredSealed(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "deny-ignored-sealed")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	if fs := run(t, rules.DenyIgnored{}, ctx); len(fs) != 0 {
		t.Fatalf("working denies must not be FAIL OPEN, got %#v", fs)
	}
}

func TestDenyIgnoredReportedShape(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "deny-ignored")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	fs := run(t, rules.DenyIgnored{}, ctx)
	if len(fs) != 1 || !strings.Contains(fs[0].Fix, `"Workshop/Checklist.md"`) {
		t.Fatalf("want one finding with the ./-free fix, got %#v", fs)
	}
}

// Absolute resource inside the project does not match the relative path (opencode#53365).
func TestDenyIgnoredAbsoluteInsideProject(t *testing.T) {
	proj := t.TempDir()
	cfg := `{"permissions":[{"action":"read","resource":"` + filepath.ToSlash(filepath.Join(proj, "secrets", "*")) + `","effect":"deny"},` +
		`{"action":"read","resource":"/etc/elsewhere/*","effect":"deny"}]}`
	if err := os.WriteFile(filepath.Join(proj, "opencode.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := run(t, rules.DenyIgnored{}, load(t, t.TempDir(), proj))
	if len(fs) != 1 || !strings.Contains(fs[0].Fix, `"secrets/*"`) {
		t.Fatalf("want only the in-project absolute rule, got %#v", fs)
	}
}

// opencode#55814: plan / edit-denying agents that can still launch "general".
func TestDelegationEscalate55814(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "delegation-escalate-55814")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	fs := run(t, rules.DelegationEscalate{}, ctx)
	if countSev(fs, report.FailOpen) != 2 {
		t.Fatalf("want plan + reviewer flagged, got %#v", fs)
	}
}

func TestDelegationEscalate55814Sealed(t *testing.T) {
	root := filepath.Join(fixtureRoot(t), "delegation-escalate-55814-sealed")
	ctx := load(t, filepath.Join(root, "home"), filepath.Join(root, "project"))
	if fs := run(t, rules.DelegationEscalate{}, ctx); len(fs) != 0 {
		t.Fatalf("maintainer workaround must be sealed, got %#v", fs)
	}
}

func onFailureFixture(t *testing.T, id string) []report.Finding {
	t.Helper()
	root := filepath.Join(fixtureRoot(t), id)
	return run(t, rules.HookMissing{}, load(t, t.TempDir(), filepath.Join(root, "project")))
}

// onFailure "block" (Claude Code v2.1.295+): a hook that cannot start blocks the call.
func TestHookMissingOnFailureBlock(t *testing.T) {
	fs := onFailureFixture(t, "hook-missing-onfailure-block")
	if len(fs) != 1 || fs[0].Severity != report.FailShut || !strings.Contains(fs[0].Evidence, "onFailure") {
		t.Fatalf("want one FAIL SHUT hook-missing note, got %#v", fs)
	}
	if strings.Contains(fs[0].Fix, "lets the tool call through") {
		t.Fatalf("fix text wrong for block case: %s", fs[0].Fix)
	}
}

func TestHookMissingOnFailureUnsetOrContinue(t *testing.T) {
	fs := onFailureFixture(t, "hook-missing-onfailure-unset")
	if len(fs) != 2 || countSev(fs, report.FailOpen) != 2 {
		t.Fatalf("want two FAIL OPEN, got %#v", fs)
	}
	for _, f := range fs {
		if !strings.Contains(f.Fix, `"onFailure": "block"`) || !strings.Contains(f.Fix, "v2.1.295") {
			t.Fatalf("fix should suggest onFailure block: %s", f.Fix)
		}
	}
}

func TestHookMissingOnFailureSealed(t *testing.T) {
	if fs := onFailureFixture(t, "hook-missing-onfailure-sealed"); len(fs) != 0 {
		t.Fatalf("want no findings, got %#v", fs)
	}
}

// onFailure has no effect on Stop-family or async hooks.
func TestHookMissingOnFailureNoEffect(t *testing.T) {
	fs := onFailureFixture(t, "hook-missing-onfailure-noeffect")
	if len(fs) != 2 || countSev(fs, report.FailOpen) != 2 {
		t.Fatalf("want Stop and async hooks FAIL OPEN, got %#v", fs)
	}
}

func TestHookMissingOnFailureNotExecutable(t *testing.T) {
	proj := t.TempDir()
	hooks := filepath.Join(proj, ".claude", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "g.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/g.sh","onFailure":"block"}]}]}}`
	if err := os.WriteFile(filepath.Join(proj, ".claude", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := run(t, rules.HookMissing{}, load(t, t.TempDir(), proj))
	if len(fs) != 1 || fs[0].Severity != report.FailShut || !strings.Contains(fs[0].Evidence, "not executable") {
		t.Fatalf("want FAIL SHUT not-executable, got %#v", fs)
	}
}
