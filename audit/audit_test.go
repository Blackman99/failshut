package audit_test

import (
	"path/filepath"
	"testing"

	"github.com/Blackman99/failshut/audit"
)

func TestAuditFailOpenExitShape(t *testing.T) {
	root := filepath.Join("..", "testdata", "ask-vs-effective")
	res, err := audit.Run(audit.Options{Home: filepath.Join(root, "home")})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Configured || res.FailOpen < 1 {
		t.Fatalf("expected fail open, got %+v", res)
	}
}

func TestAuditNothingConfigured(t *testing.T) {
	dir := t.TempDir()
	res, err := audit.Run(audit.Options{Home: dir, Project: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Configured || res.FailOpen != 0 {
		t.Fatalf("expected empty sealed, got %+v", res)
	}
}
