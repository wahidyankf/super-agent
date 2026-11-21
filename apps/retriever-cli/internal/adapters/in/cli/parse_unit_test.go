package cli

import (
	"strings"
	"testing"
)

// TestParseNamesEveryMissingPositionalArgument checks that a command line which leaves out positional arguments is refused,
// and that the refusal names each missing argument in angle brackets. It parses a leaf directly, so no command runs.
func TestParseNamesEveryMissingPositionalArgument(t *testing.T) {
	l := &leaf{positional: []string{"id", "name"}}

	if _, err := l.parse([]string{"one"}); err == nil || !strings.Contains(err.Error(), "missing arguments: <name>") {
		t.Fatalf("parse with one argument = %v, want the missing <name>", err)
	}
	if _, err := l.parse(nil); err == nil || !strings.Contains(err.Error(), "missing arguments: <id> <name>") {
		t.Fatalf("parse with no arguments = %v, want the missing <id> <name>", err)
	}
}
