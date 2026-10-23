# Changelog

## Unreleased

## v0.1.3 — 2026-10-24

### Fixed

- `hook-missing`: a hook whose interpreter (e.g. `python3`) is not on PATH is now FAIL OPEN (shell exit 127, non-blocking), or FAIL SHUT with `"onFailure": "block"`. Previously it was silent when the script existed, and mis-scored FAIL SHUT when the script was also missing.

## v0.1.2 — 2026-10-23

### Fixed
- **`hook-missing` respects `"onFailure": "block"`** ([#7](https://github.com/Blackman99/failshut/issues/7)). Since Claude Code v2.1.295 a hook entry with `"onFailure": "block"` blocks the call when it can't start ([docs](https://code.claude.com/docs/en/hooks#block-the-action-when-a-hook-fails)). A missing or non-executable script on such a hook (PreToolUse, PermissionRequest, UserPromptSubmit) is now FAIL_SHUT with a note that every matching call is blocked, not a false FAIL OPEN. Stop-family and `async`/`asyncRewake` hooks, where `onFailure` has no effect, stay FAIL OPEN.
- `hook-missing` fix text no longer claims every missing hook lets calls through, and suggests adding `"onFailure": "block"` (Claude Code v2.1.295+) alongside restoring the script.

### Added
- Human report prints FAIL_SHUT findings (hooks broken shut) under a **Note** section; previously they appeared only in `--json`.

### Changed
- Logo recentred and rebalanced so the mark fills the canvas. `scripts/make_assets.py logo` now generates logo-512.png, icon-192.png, apple-touch-icon.png, favicon-32/16.png, favicon.ico, logo.svg and favicon.svg from one geometry. Favicon sizes use a simplified variant.

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
