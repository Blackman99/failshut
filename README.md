<p align="center">
  <img src="site/logo.svg" width="96" height="96" alt="failshut logo"/>
</p>

# failshut

**Tell you, in one command, whether your AI coding-agent permission hooks and deny/ask rules actually fail shut — or silently fail open.**

You configured deny/ask/hooks for Claude Code, Codex, or OpenCode. After an upgrade, does policy still hold — or did a harness quirk turn it into a silent allow? `failshut audit` pressurizes the config manifold and reports every joint that bubbles.

Pain this maps to: [opencode#55728](https://github.com/anomalyco/opencode/issues/55728) · [opencode#55909](https://github.com/anomalyco/opencode/issues/55909) · [opencode#55814](https://github.com/anomalyco/opencode/issues/55814) · [codex#54298](https://github.com/openai/codex/issues/54298) · [codex#54440](https://github.com/openai/codex/issues/54440) · [codex#54448](https://github.com/openai/codex/issues/54448) · [claude-code#102582](https://github.com/anthropics/claude-code/issues/102582)

## Install

```bash
# Linux amd64 example — see Releases for darwin/windows + arm64
curl -sL https://github.com/Blackman99/failshut/releases/download/v0.1.0/failshut_0.1.0_linux_amd64.tar.gz | tar xz
./failshut audit
```

Or: `go install github.com/Blackman99/failshut/cmd/failshut@latest`

## Usage

```bash
failshut audit
failshut audit --project ~/code/my-app
failshut audit --json
failshut mcp          # read-only: audit_report, list_findings
```

Sample FAIL OPEN output:

```text
FAIL OPEN — 1 finding(s) where policy may not hold:

[1] ask-vs-effective  (codex)
    file:     ~/.codex/config.toml
    evidence: approval_policy="ask" but sandbox_approval=false
    fix:      Set sandbox_approval=true so Ask cannot be skipped

Summary: 1 FAIL OPEN · 5 FAIL SHUT checks · 1 config(s)
```

![demo](site/demo.gif)

Exit `1` on any FAIL OPEN · `0` if sealed or nothing configured.

**v0.1 is static.** It does not intercept live tool calls (`--canary` → not in v0.1).

## Rules

| ID | Catches |
|---|---|
| `deny-ignored` | Deny on read/grep-like tools harnesses may ignore |
| `ask-vs-effective` | Ask UI vs `sandbox_approval=false` / silent escalate |
| `hook-missing` | Hook path missing or not executable |
| `path-traversal-config` | `..` path can load a sibling config |
| `delegation-escalate` | Plan/restrictive profile → general subagent |
| `ask-auto-allow` | Hook asks, desktop auto-allows |

## Docs & site

https://blackman99.github.io/failshut/ · [docs](https://blackman99.github.io/failshut/docs/)

## License

MIT
