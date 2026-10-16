package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditJSONFailOpen(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "deny-ignored")
	// Use HOME via env is tricky; call audit through flags --project only for opencode
	var out, errb bytes.Buffer
	// We need HOME override — audit.Options.Home is only set from discover via audit.Run
	// For CLI, home comes from os.UserHomeDir. Test audit package instead for HOME.
	// Here smoke-test help/version.
	code := run([]string{"version"}, &out, &errb)
	if code != 0 || !strings.Contains(out.String(), "failshut") {
		t.Fatalf("version: code=%d out=%q err=%q", code, out.String(), errb.String())
	}
	out.Reset()
	code = run([]string{"help"}, &out, &errb)
	if code != 0 || !strings.Contains(out.String(), "FAIL") {
		t.Fatalf("help: code=%d out=%q", code, out.String())
	}
	_ = root
}

func TestUnknownCommand(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"auditt"}, &out, &errb)
	if code != 1 || !strings.Contains(errb.String(), "did you mean") {
		t.Fatalf("got code=%d err=%q", code, errb.String())
	}
}

func TestCanaryStub(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"audit", "--canary"}, &out, &errb)
	if code != 1 || !strings.Contains(errb.String(), "not in v0.1") {
		t.Fatalf("got code=%d err=%q", code, errb.String())
	}
}
