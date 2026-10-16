package rules_test

import (
	"os"
	"path/filepath"
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
