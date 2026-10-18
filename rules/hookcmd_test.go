package rules

import "testing"

func TestParseHookCommand(t *testing.T) {
	env := map[string]string{"HOME": "/h", "CLAUDE_PROJECT_DIR": "/p"}
	cases := []struct {
		cmd, script, interp string
		ok                  bool
	}{
		{`~/.claude/hooks/a.sh`, "/h/.claude/hooks/a.sh", "", true},
		{`bash ~/.claude/hooks/a.sh`, "/h/.claude/hooks/a.sh", "bash", true},
		{`/usr/bin/env bash -e ~/x.sh --flag`, "/h/x.sh", "bash", true},
		{`python3 "$HOME/hooks/check env.py" --strict`, "/h/hooks/check env.py", "python3", true},
		{`node ${HOME}/a.js`, "/h/a.js", "node", true},
		{`"$CLAUDE_PROJECT_DIR"/.claude/hooks/b.sh`, "/p/.claude/hooks/b.sh", "", true},
		{`FOO=1 bash '/opt/my hooks/c.sh'`, "/opt/my hooks/c.sh", "bash", true},
		{`node --require ts-node/register hooks/d.ts`, "hooks/d.ts", "node", true},
		{`deno run -A hooks/e.ts`, "hooks/e.ts", "deno", true},
		{`"~/.claude/hooks/a.sh"`, "~/.claude/hooks/a.sh", "", true}, // quoted ~ is literal in bash
		{`bash -c 'exit 0'`, "", "", false},
		{`python3 -m hooks.guard`, "", "", false},
		{`jq -r .tool_name`, "", "", false},
		{`echo ask | tee /tmp/x`, "", "", false},
		{`$UNKNOWN/x.sh`, "", "", false},
		{`bash $(pwd)/x.sh`, "", "", false},
		{`bash`, "", "", false},
	}
	for _, c := range cases {
		inv, ok := parseHookCommand(c.cmd, env)
		if ok != c.ok || inv.Script != c.script || inv.Interpreter != c.interp {
			t.Errorf("%q: got (%q, %q, %v), want (%q, %q, %v)", c.cmd, inv.Script, inv.Interpreter, ok, c.script, c.interp, c.ok)
		}
	}
}
