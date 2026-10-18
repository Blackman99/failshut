Modeled on anthropics/claude-code#102649: hook commands of the form
`bash ~/.claude/hooks/<name>.sh` after ~/.claude/hooks vanished. bash exits
127 on a missing script, which Claude Code treats as non-blocking.
