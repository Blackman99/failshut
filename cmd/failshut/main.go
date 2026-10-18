// failshut audits AI coding-agent permission configs for fail-open gaps.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Blackman99/failshut/audit"
	"github.com/Blackman99/failshut/mcp"
	"github.com/Blackman99/failshut/report"
)

var version = "dev"

const usage = `failshut — leak-check for AI agent permission hooks

Tell you, in one command, whether your AI coding-agent permission hooks and
deny/ask rules actually fail shut — or silently fail open.

Usage:
  failshut                          # same as audit
  failshut audit [--json] [--project DIR] [--no-color] [--rule ID]
  failshut version
  failshut mcp                      # read-only MCP stdio (audit_report, list_findings)
  failshut help

Exit codes: 0 all fail shut / nothing configured · 1 any FAIL OPEN or error

This release is a static config auditor. It does not intercept live tool calls.
Optional --canary is not in v0.1.
`

var commands = []string{"audit", "version", "mcp", "help"}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		return cmdAudit(nil, out, errw)
	}
	switch args[0] {
	case "audit":
		return cmdAudit(args[1:], out, errw)
	case "mcp":
		mcp.Version = version
		if err := mcp.Run(os.Stdin, out); err != nil {
			fmt.Fprintln(errw, "failshut mcp:", err)
			return 1
		}
		return 0
	case "version", "-version", "--version":
		fmt.Fprintln(out, "failshut", version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return 0
	default:
		if strings.HasPrefix(args[0], "-") {
			// flags without subcommand → audit
			return cmdAudit(args, out, errw)
		}
		fmt.Fprintf(errw, "failshut: unknown command %q\n", args[0])
		if s := suggest(args[0], commands); s != "" {
			fmt.Fprintf(errw, "  did you mean: failshut %s\n", s)
		}
		fmt.Fprintln(errw, "  run 'failshut help' for usage")
		return 1
	}
}

func cmdAudit(args []string, out, errw io.Writer) int {
	asJSON, noColor, project := false, false, ""
	var ruleFilter []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			asJSON = true
		case a == "--no-color":
			noColor = true
		case a == "--canary":
			fmt.Fprintln(errw, "failshut: --canary is not in v0.1 (static audit only). See ROADMAP.md.")
			return 1
		case a == "--help" || a == "-h":
			fmt.Fprint(out, `failshut audit — scan harness permission configs

Flags:
  --json           machine-readable report
  --project DIR    also scan this project directory
  --rule ID        only run this rule (repeatable)
  --no-color       disable ANSI colors
  --canary         not in v0.1
`)
			return 0
		case a == "--project" && i+1 < len(args):
			i++
			project = args[i]
		case strings.HasPrefix(a, "--project="):
			project = strings.TrimPrefix(a, "--project=")
		case a == "--rule" && i+1 < len(args):
			i++
			ruleFilter = append(ruleFilter, args[i])
		case strings.HasPrefix(a, "--rule="):
			ruleFilter = append(ruleFilter, strings.TrimPrefix(a, "--rule="))
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(errw, "failshut audit: unknown flag %s\n", a)
			fmt.Fprintln(errw, "  run 'failshut audit --help' for usage")
			return 1
		default:
			fmt.Fprintf(errw, "failshut audit: unexpected argument %q\n", a)
			return 1
		}
	}

	res, err := audit.Run(audit.Options{Project: project, Rules: ruleFilter})
	if err != nil {
		fmt.Fprintln(errw, "failshut:", err)
		return 1
	}
	if asJSON {
		if err := report.WriteJSON(out, res); err != nil {
			fmt.Fprintln(errw, "failshut:", err)
			return 1
		}
	} else {
		useColor := !noColor && isTTY(out) && os.Getenv("NO_COLOR") == ""
		report.WriteHuman(out, res, report.ColorsFor(useColor))
	}
	if res.FailOpen > 0 {
		return 1
	}
	return 0
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

func suggest(got string, known []string) string {
	got = strings.ToLower(got)
	best, bestD := "", 3
	for _, k := range known {
		d := levenshtein(got, k)
		if d < bestD {
			bestD, best = d, k
		}
	}
	return best
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, ins, sub)
		}
		prev = cur
	}
	return prev[len(b)]
}
