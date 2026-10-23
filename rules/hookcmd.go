package rules

import (
	"path/filepath"
	"strings"
)

// hookInvocation is what a hook "command" string actually runs, as far as a
// static reader can tell without executing anything.
type hookInvocation struct {
	Script      string // script path after quote removal and $VAR / ~ expansion
	Interpreter string // basename of the interpreter ("bash", "python3"), "" when the script is run directly
	Exe         string // the interpreter word as written ("python3", "/usr/bin/python3"), "" when the script is run directly
}

// splitShellWords splits a hook command the way a POSIX shell would for one
// simple command: single/double quotes, backslash escapes, unquoted leading ~,
// and $NAME / ${NAME} expansion from env. ok is false when the command uses
// anything a static reader should not guess about (pipes, lists, redirects,
// command substitution, unknown variables).
func splitShellWords(s string, env map[string]string) (words []string, ok bool) {
	var cur strings.Builder
	inWord := false
	atWordStart := true
	flush := func() {
		if inWord {
			words = append(words, cur.String())
		}
		cur.Reset()
		inWord = false
		atWordStart = true
	}
	expand := func(i int) (string, int, bool) { // s[i] == '$'
		j := i + 1
		if j < len(s) && s[j] == '{' {
			end := strings.IndexByte(s[j:], '}')
			if end < 0 {
				return "", 0, false
			}
			name := s[j+1 : j+end]
			v, found := env[name]
			return v, j + end + 1, found && validVarName(name)
		}
		k := j
		for k < len(s) && (s[k] == '_' || isAlnum(s[k])) {
			k++
		}
		if k == j {
			return "", 0, false // "$(", "$?", lone "$" ...
		}
		v, found := env[s[j:k]]
		return v, k, found
	}
	for i := 0; i < len(s); {
		ch := s[i]
		switch {
		case ch == ' ' || ch == '\t':
			flush()
			i++
		case ch == '\n' || ch == '|' || ch == '&' || ch == ';' || ch == '<' || ch == '>' || ch == '`' || ch == '(' || ch == ')':
			return nil, false
		case ch == '\\':
			if i+1 >= len(s) {
				return nil, false
			}
			cur.WriteByte(s[i+1])
			inWord, atWordStart = true, false
			i += 2
		case ch == '\'':
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return nil, false
			}
			cur.WriteString(s[i+1 : i+1+end])
			inWord, atWordStart = true, false
			i += end + 2
		case ch == '"':
			i++
			closed := false
			for i < len(s) {
				c := s[i]
				if c == '"' {
					closed = true
					i++
					break
				}
				switch c {
				case '\\':
					if i+1 < len(s) && strings.IndexByte("$`\"\\", s[i+1]) >= 0 {
						cur.WriteByte(s[i+1])
						i += 2
					} else {
						cur.WriteByte(c)
						i++
					}
				case '$':
					v, next, good := expand(i)
					if !good {
						return nil, false
					}
					cur.WriteString(v)
					i = next
				case '`':
					return nil, false
				default:
					cur.WriteByte(c)
					i++
				}
			}
			if !closed {
				return nil, false
			}
			inWord, atWordStart = true, false
		case ch == '$':
			v, next, good := expand(i)
			if !good {
				return nil, false
			}
			cur.WriteString(v)
			inWord, atWordStart = true, false
			i = next
		case ch == '~' && atWordStart:
			// ~ or ~/... expands to HOME; ~user is not modeled.
			if i+1 < len(s) && s[i+1] != '/' && s[i+1] != ' ' {
				return nil, false
			}
			home, found := env["HOME"]
			if !found {
				return nil, false
			}
			cur.WriteString(home)
			inWord, atWordStart = true, false
			i++
		default:
			cur.WriteByte(ch)
			inWord, atWordStart = true, false
			i++
		}
	}
	flush()
	return words, true
}

func isAlnum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func validVarName(n string) bool {
	if n == "" {
		return false
	}
	for i := 0; i < len(n); i++ {
		if n[i] != '_' && !isAlnum(n[i]) {
			return false
		}
	}
	return true
}

func isAssignment(w string) bool {
	eq := strings.IndexByte(w, '=')
	return eq > 0 && validVarName(w[:eq])
}

// isInlineFlag reports interpreter flags that mean "the code is inline / a
// module, not a script file" (bash -c, python3 -m, node -e, perl -e ...).
func isInlineFlag(interp, a string) bool {
	switch {
	case strings.HasPrefix(interp, "python"):
		return a == "-c" || a == "-m"
	case interp == "node" || interp == "nodejs" || interp == "bun" || interp == "deno" || interp == "tsx" || interp == "ts-node":
		return a == "-e" || a == "-p" || a == "--eval" || a == "--print"
	case interp == "ruby" || interp == "perl":
		return a == "-e" || a == "-E"
	default: // shells
		return a == "-c"
	}
}

// interpreter flags that consume the next word as their value.
var valueFlags = map[string]bool{"-r": true, "--require": true, "--import": true, "--loader": true, "-I": true, "-W": true, "-X": true, "--config": true}

func interpreterName(exe string) string {
	b := strings.ToLower(filepath.Base(exe))
	b = strings.TrimSuffix(b, ".exe")
	switch {
	case b == "bash" || b == "sh" || b == "zsh" || b == "dash" || b == "ksh":
		return b
	case b == "python" || strings.HasPrefix(b, "python3") || strings.HasPrefix(b, "python2"):
		return b
	case b == "node" || b == "nodejs" || b == "bun" || b == "deno" || b == "tsx" || b == "ts-node":
		return b
	case b == "ruby" || b == "perl":
		return b
	}
	return ""
}

// parseHookCommand returns the script a hook command runs. ok is false for
// bare commands on PATH (jq, npx, claude ...), inline code, or anything the
// splitter refuses; those are not file references failshut can check.
func parseHookCommand(cmd string, env map[string]string) (hookInvocation, bool) {
	words, ok := splitShellWords(strings.TrimSpace(cmd), env)
	if !ok {
		return hookInvocation{}, false
	}
	for len(words) > 0 && isAssignment(words[0]) {
		words = words[1:]
	}
	if len(words) > 0 && filepath.Base(words[0]) == "env" {
		words = words[1:]
		for len(words) > 0 && (isAssignment(words[0]) || strings.HasPrefix(words[0], "-")) {
			words = words[1:]
		}
	}
	if len(words) == 0 {
		return hookInvocation{}, false
	}
	exe := words[0]
	if interp := interpreterName(exe); interp != "" {
		args := words[1:]
		if (interp == "deno" || interp == "bun") && len(args) > 0 && args[0] == "run" {
			args = args[1:]
		}
		for i := 0; i < len(args); i++ {
			a := args[i]
			if isInlineFlag(interp, a) {
				return hookInvocation{}, false
			}
			if valueFlags[a] {
				i++
				continue
			}
			if strings.HasPrefix(a, "-") {
				continue
			}
			return hookInvocation{Script: a, Interpreter: interp, Exe: exe}, true
		}
		return hookInvocation{}, false // interpreter reading stdin
	}
	if strings.ContainsRune(exe, '/') {
		return hookInvocation{Script: exe}, true
	}
	return hookInvocation{}, false
}
