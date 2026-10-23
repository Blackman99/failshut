Interpreter missing (the test runs with an empty PATH). The shell exits 127
("command not found"), a non-blocking error, so the call goes through:
- Write: script exists, python3 absent -> FAIL OPEN.
- Bash: script also missing -> still FAIL OPEN (python3 never runs, so it
  never exits 2; previously mis-scored FAIL_SHUT).
- Edit: onFailure "block" -> FAIL SHUT.
