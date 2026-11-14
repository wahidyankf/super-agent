package envfile

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// lookup returns the value of variable in the dotenv text r. The format is KEY=VALUE, one per line. Blank lines
// and lines that start with '#' are skipped, and an optional "export" prefix is ignored. A value in single or
// double quotes is returned as written between the quotes, with no escapes and no expansion. An unquoted value
// loses trailing spaces and any " #" comment. Other keys are checked for syntax and then ignored. When a key
// repeats, the last value wins.
//
// A malformed line returns an error that names its line number only. The line itself may hold a secret, so its
// text never appears in the error.
func lookup(r io.Reader, variable string) (string, error) {
	var value string
	scanner := bufio.NewScanner(r)
	for n := 1; scanner.Scan(); n++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = trimExport(line)
		key, raw, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return "", lineError(n, "expected NAME=VALUE")
		}
		parsed, err := parseValue(strings.TrimSpace(raw))
		if err != nil {
			return "", lineError(n, err.Error())
		}
		if key == variable {
			value = parsed
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading local env file: %w", err)
	}
	return value, nil
}

// trimExport drops a leading "export" keyword that is followed by a space or a tab.
func trimExport(line string) string {
	rest, ok := strings.CutPrefix(line, "export")
	if ok && rest != "" && (rest[0] == ' ' || rest[0] == '\t') {
		return strings.TrimLeft(rest, " \t")
	}
	return line
}

// parseValue reads one value that has already lost its outer spaces. A quoted value must close, and only a comment
// may follow the closing quote. An unquoted value is cut at a comment marker.
func parseValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if raw[0] != '"' && raw[0] != '\'' {
		if i := commentStart(raw); i >= 0 {
			raw = raw[:i]
		}
		return strings.TrimSpace(raw), nil
	}
	quote := raw[0]
	end := strings.IndexByte(raw[1:], quote)
	if end < 0 {
		return "", errors.New("unclosed quote")
	}
	rest := strings.TrimSpace(raw[end+2:])
	if rest != "" && rest[0] != '#' {
		return "", errors.New("text after a closing quote")
	}
	return raw[1 : end+1], nil
}

// commentStart returns the index of a " #" or tab-hash comment marker in raw, or -1 when there is none.
func commentStart(raw string) int {
	for i := 1; i < len(raw); i++ {
		if raw[i] == '#' && (raw[i-1] == ' ' || raw[i-1] == '\t') {
			return i
		}
	}
	return -1
}

// lineError reports a malformed line by its number. It never includes the line text.
func lineError(n int, reason string) error {
	return fmt.Errorf("local env file line %d: %s", n, reason)
}
