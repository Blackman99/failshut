# Roadmap

## v0.1.1 (shipped)
- Correctness release: hook command parsing, OpenCode deny false positive removed, opencode#55728 / #55814 shapes detected, honest README coverage table. See CHANGELOG.md.

## v0.1.0 (shipped)
- `failshut` / `failshut audit` / `version` / `mcp`
- Harnesses: Claude Code, Codex, OpenCode (best-effort Cursor when readable)
- Rules: `deny-ignored`, `ask-vs-effective`, `hook-missing`, `path-traversal-config`, `delegation-escalate`, `ask-auto-allow`
- Human report (FAIL OPEN first) + `--json`; exit 1 on any fail-open
- Read-only MCP stdio: `audit_report`, `list_findings`
- GoReleaser binaries; GitHub Pages copper-manifold landing

## v0.2
- Hook latency / C16 "hook doctor" report mode (time PreToolUse round-trips)
- Richer Cursor settings discovery when layout stabilizes

## v0.3
- Optional live `--canary` probes (deny a known tool, attempt, compare)
- SARIF output for CI gates

## v1.0
- Stable rule IDs + brew formula; documented fixture corpus mirroring tracked upstream issues
