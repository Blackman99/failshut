# Goal

**Tell you, in one command, whether your AI coding-agent permission hooks and deny/ask rules actually fail shut — or silently fail open.**

In scope: discover Claude Code / Codex / OpenCode (and Cursor when readable) permission configs on this machine; static rules that flag deny-ignored shapes, ask-vs-effective mismatches, missing hooks, `..` path config loads, delegation escalation gaps, and ask-auto-allow pairings; human + JSON report; exit 1 on any FAIL OPEN; optional read-only MCP (`audit_report`, `list_findings`).

Out of scope (v0.1): live runtime interception / canary probes (optional `--canary` stub only); building another harness or orchestrator; hook latency flame tables (→ v0.2); lethal-trifecta egress as a separate product.
