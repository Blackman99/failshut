// Package report defines findings and renders human / JSON output.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Severity for a finding. FailOpen is the only actionable severity in v0.1.
type Severity string

const (
	FailOpen Severity = "FAIL_OPEN"
	FailShut Severity = "FAIL_SHUT"
)

// Finding is one static audit hit.
type Finding struct {
	ID       string   `json:"id"`
	Harness  string   `json:"harness"`
	File     string   `json:"file"`
	Evidence string   `json:"evidence"`
	Fix      string   `json:"fix"`
	Severity Severity `json:"severity"`
}

// Result is the full audit report.
type Result struct {
	Findings   []Finding `json:"findings"`
	FailOpen   int       `json:"fail_open"`
	FailShut   int       `json:"fail_shut"`
	Configs    int       `json:"configs_scanned"`
	Configured bool      `json:"configured"`
	Message    string    `json:"message,omitempty"`
}

// WriteJSON emits the result as JSON.
func WriteJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// Color codes (ANSI). Empty when disabled.
type Colors struct {
	Red, Green, Bold, Dim, Reset string
}

func ColorsFor(enabled bool) Colors {
	if !enabled {
		return Colors{}
	}
	return Colors{
		Red: "\033[31m", Green: "\033[32m", Bold: "\033[1m",
		Dim: "\033[2m", Reset: "\033[0m",
	}
}

// WriteHuman prints FAIL OPEN findings first, then a FAIL SHUT summary.
func WriteHuman(w io.Writer, r Result, c Colors) {
	if !r.Configured {
		fmt.Fprintln(w, "failshut: no Claude Code / Codex / OpenCode permission configs found on this machine.")
		fmt.Fprintln(w, "  Nothing configured → treated as FAIL SHUT (exit 0).")
		fmt.Fprintln(w, "  Point at a project with: failshut audit --project <dir>")
		return
	}

	opens := make([]Finding, 0)
	notes := make([]Finding, 0) // FAIL_SHUT findings: broken shut (calls blocked), worth knowing
	for _, f := range r.Findings {
		switch f.Severity {
		case FailOpen:
			opens = append(opens, f)
		case FailShut:
			notes = append(notes, f)
		}
	}
	defer writeNotes(w, notes, c)
	shuts := r.FailShut // rules that came back sealed

	if len(opens) == 0 {
		fmt.Fprintf(w, "%sFAIL SHUT%s — %d config(s) scanned, no fail-open findings.\n", c.Green+c.Bold, c.Reset, r.Configs)
		if shuts > 0 {
			fmt.Fprintf(w, "%s  %d check(s) confirmed sealed.%s\n", c.Dim, shuts, c.Reset)
		}
		return
	}

	fmt.Fprintf(w, "%sFAIL OPEN%s — %d finding(s) where policy may not hold:\n\n", c.Red+c.Bold, c.Reset, len(opens))
	for i, f := range opens {
		fmt.Fprintf(w, "%s[%d] %s%s  %s(%s)%s\n", c.Bold, i+1, f.ID, c.Reset, c.Dim, f.Harness, c.Reset)
		fmt.Fprintf(w, "    file:     %s\n", f.File)
		ev := strings.TrimSpace(f.Evidence)
		if len(ev) > 200 {
			ev = ev[:197] + "..."
		}
		fmt.Fprintf(w, "    evidence: %s\n", ev)
		fmt.Fprintf(w, "    fix:      %s\n\n", f.Fix)
	}
	fmt.Fprintf(w, "%sSummary:%s %s%d FAIL OPEN%s · %s%d FAIL SHUT checks%s · %d config(s)\n",
		c.Bold, c.Reset, c.Red, len(opens), c.Reset, c.Green, shuts, c.Reset, r.Configs)
}

// writeNotes prints FAIL_SHUT findings: the gate is broken but fails closed.
func writeNotes(w io.Writer, notes []Finding, c Colors) {
	if len(notes) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%sNote%s — %d hook(s) broken shut (matching calls are blocked, policy never runs):\n", c.Bold, c.Reset, len(notes))
	for _, f := range notes {
		ev := strings.TrimSpace(f.Evidence)
		if len(ev) > 200 {
			ev = ev[:197] + "..."
		}
		fmt.Fprintf(w, "  %s%s%s  %s(%s)%s %s\n", c.Dim, f.ID, c.Reset, c.Dim, f.Harness, c.Reset, f.File)
		fmt.Fprintf(w, "    evidence: %s\n", ev)
		fmt.Fprintf(w, "    fix:      %s\n", f.Fix)
	}
}
