python3 exits 2 when its script is missing. On PreToolUse, exit 2 blocks the
call (broken shut, reported as FAIL_SHUT). On Stop, Claude Code downgrades
exit 2 + "can't open" stderr to non-blocking (claude-code#99171): FAIL OPEN.
