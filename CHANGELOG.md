# Changelog

## v0.1.1 — 2026-10-18

Correctness release ([#3](https://github.com/Blackman99/failshut/pull/3)). Still a static config audit: nothing intercepts live tool calls.

### Fixed
- **`hook-missing` parses the hook command** instead of treating the whole string as a path. It handles quotes, `~`, `$HOME` / `${HOME}`, `"$CLAUDE_PROJECT_DIR"/…`, interpreter prefixes (`bash`, `sh`, `zsh`, `python3`, `node`, `bun`, `deno`, `tsx`, `ruby`, `perl`, `env`) and leading `VAR=x`. Commands are never executed. An existing script is sealed and a missing one is FAIL OPEN. v0.1.0 reported every `bash ~/.claude/hooks/x.sh` hook as FAIL OPEN even when the script existed, and never checked `"$CLAUDE_PROJECT_DIR"` paths. Modeled on [claude-code#32990](https://github.com/anthropics/claude-code/issues/32990) and [#102649](https://github.com/anthropics/claude-code/issues/102649).
  - A missing `python3` / `perl` script exits 2. On PreToolUse, PermissionRequest and UserPromptSubmit, Claude Code treats exit 2 as a block, so that case is reported as FAIL_SHUT (calls blocked), not open. Stop-family hooks stay FAIL OPEN ([claude-code#99171](https://github.com/anthropics/claude-code/issues/99171)).
  - Relative script paths resolve against the project, and the executable-bit check applies only to scripts run directly.
- **OpenCode `deny-ignored` false positive removed.** A plain `"read": "deny"` works and is no longer reported. The rule now flags `read`/`edit` deny/ask rules that never match: a `./`-prefixed resource (the exact [opencode#55728](https://github.com/anomalyco/opencode/issues/55728) config) or an absolute resource inside the project ([opencode#53365](https://github.com/anomalyco/opencode/pull/53365)).
- **Report summary** "N FAIL SHUT checks" now counts sealed rules (it always printed 0).
- `failshut mcp` reports the build version in `serverInfo` instead of a hard-coded `0.1.0`.

### Added
- **`delegation-escalate` for OpenCode v2 agents:** an agent (including `plan`) that denies edit but does not deny launching the write-capable `general` subagent ([opencode#55814](https://github.com/anomalyco/opencode/issues/55814)).
- **README Coverage table**, issue by issue. codex#54298, #54440, #54448 and claude-code#102582 are listed as not detected instead of being implied as covered.

## v0.1.0 — 2026-10-16

First release: `failshut audit`, six static rules, human + JSON report, read-only MCP, GoReleaser binaries.
