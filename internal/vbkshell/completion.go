package vbkshell

import (
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var shellCommands = []string{
	"volumes", "use", "disks", "pwd", "ls", "ll", "cd", "cat", "get", "find", "stat", "tree", "grep", "help", "exit", "quit",
}

type completionState struct {
	completed    []string
	current      string
	currentStart int
}

func (s *Shell) completeInput(line string) []string {
	state := parseCompletionState(line)

	if len(state.completed) == 0 {
		return buildLineCompletions(line, state.currentStart, completeCommands(state.current, true))
	}

	cmd := state.completed[0]
	argIndex := len(state.completed) - 1

	switch cmd {
	case "use":
		if argIndex == 0 {
			return buildLineCompletions(line, state.currentStart, s.completeVolumeIndexes(state.current))
		}
	case "cd":
		if argIndex == 0 {
			return buildLineCompletions(line, state.currentStart, s.completePaths(state.current, true))
		}
	case "ls", "ll", "cat", "stat", "tree":
		if argIndex == 0 {
			return buildLineCompletions(line, state.currentStart, s.completePaths(state.current, false))
		}
	case "get":
		if argIndex == 0 {
			return buildLineCompletions(line, state.currentStart, s.completePaths(state.current, false))
		}
	case "find", "grep":
		if argIndex == 1 {
			return buildLineCompletions(line, state.currentStart, s.completePaths(state.current, true))
		}
	}

	if len(state.completed) == 1 {
		return buildLineCompletions(line, state.currentStart, completeCommands(state.current, true))
	}

	return nil
}

func parseCompletionState(line string) completionState {
	state := completionState{completed: []string{}, current: "", currentStart: len(line)}
	if line == "" {
		return state
	}

	var cur []rune
	curStart := -1
	escape := false
	inSingle := false
	inDouble := false

	flush := func() {
		if curStart != -1 {
			state.completed = append(state.completed, string(cur))
			cur = nil
			curStart = -1
		}
	}

	for i, r := range line {
		if escape {
			if curStart == -1 {
				curStart = i
			}
			cur = append(cur, r)
			escape = false
			continue
		}

		if r == '\\' && !inSingle {
			if curStart == -1 {
				curStart = i
			}
			escape = true
			continue
		}

		if inSingle {
			if r == '\'' {
				inSingle = false
				continue
			}
			cur = append(cur, r)
			continue
		}

		if inDouble {
			if r == '"' {
				inDouble = false
				continue
			}
			cur = append(cur, r)
			continue
		}

		if r == '\'' {
			if curStart == -1 {
				curStart = i
			}
			inSingle = true
			continue
		}
		if r == '"' {
			if curStart == -1 {
				curStart = i
			}
			inDouble = true
			continue
		}

		if unicode.IsSpace(r) {
			flush()
			continue
		}

		if curStart == -1 {
			curStart = i
		}
		cur = append(cur, r)
	}

	if len(line) > 0 {
		last := rune(line[len(line)-1])
		if unicode.IsSpace(last) {
			flush()
			state.currentStart = len(line)
			state.current = ""
			return state
		}
	}

	if curStart != -1 {
		state.current = string(cur)
		state.currentStart = curStart
		return state
	}

	return state
}

func buildLineCompletions(line string, currentStart int, tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
	prefix := line[:currentStart]
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, prefix+t)
	}
	return out
}

func completeCommands(prefix string, appendSpace bool) []string {
	lower := strings.ToLower(prefix)
	out := make([]string, 0, len(shellCommands))
	for _, cmd := range shellCommands {
		if strings.HasPrefix(strings.ToLower(cmd), lower) {
			if appendSpace {
				out = append(out, cmd+" ")
			} else {
				out = append(out, cmd)
			}
		}
	}
	sort.Strings(out)
	return out
}

func (s *Shell) completeVolumeIndexes(prefix string) []string {
	vols := s.VolumesInfo()
	out := make([]string, 0, len(vols))
	for _, v := range vols {
		idx := strconv.Itoa(v.Index)
		if strings.HasPrefix(idx, prefix) {
			out = append(out, idx+" ")
		}
	}
	return out
}

func (s *Shell) completePaths(prefix string, dirsOnly bool) []string {
	original := prefix
	decoded := unescapeToken(prefix)

	typedDir := ""
	leaf := ""
	isAbs := strings.HasPrefix(decoded, "/")

	if decoded == "" {
		typedDir = ""
		leaf = ""
	} else if strings.HasSuffix(decoded, "/") {
		typedDir = decoded
		leaf = ""
	} else {
		typedDir = path.Dir(decoded)
		if typedDir == "." {
			typedDir = ""
		}
		leaf = path.Base(decoded)
	}

	searchDir := s.cwd
	if typedDir != "" {
		searchDir = s.resolve(typedDir)
	}

	entries, err := s.LSEntries(searchDir)
	if err != nil {
		return nil
	}

	leafLower := strings.ToLower(leaf)
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if dirsOnly && !e.IsDir {
			continue
		}
		if leaf != "" && !strings.HasPrefix(strings.ToLower(e.Name), leafLower) {
			continue
		}

		var candidate string
		if typedDir == "" {
			if isAbs {
				candidate = "/" + e.Name
			} else {
				candidate = e.Name
			}
		} else {
			candidate = strings.TrimRight(typedDir, "/") + "/" + e.Name
		}
		if e.IsDir {
			candidate += "/"
		}
		out = append(out, escapeToken(candidate))
	}

	if decoded == "" && original == "" {
		sort.Strings(out)
		return out
	}

	sort.Strings(out)
	return out
}

func unescapeToken(s string) string {
	r := strings.NewReplacer(
		`\ `, " ",
		`\(`, "(",
		`\)`, ")",
		`\[`, "[",
		`\]`, "]",
		`\\`, `\`,
	)
	return r.Replace(s)
}

func escapeToken(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		" ", `\ `,
		"(", `\(`,
		")", `\)`,
		"[", `\[`,
		"]", `\]`,
	)
	return r.Replace(s)
}
