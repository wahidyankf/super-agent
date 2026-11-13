package cli

import (
	"strings"
	"testing"
)

// TestMCPHelpDocumentsTheSignalStatuses checks that the mcp help page names the statuses a signal ends the session with,
// 130 for an interrupt and 143 for SIGTERM. The command returns both, so the page must list both (CLI convention 001).
func TestMCPHelpDocumentsTheSignalStatuses(t *testing.T) {
	// A protocol leaf writes its help to standard error, so the page is read from both streams.
	code, out, errOut := runRetriever(t, []string{"mcp", "--help"}, outlineSetup{})
	if code != 0 {
		t.Fatalf("mcp --help exit = %d, want 0", code)
	}
	page := out + errOut
	for _, line := range []string{"\n  130  ", "\n  143  "} {
		if !strings.Contains(page, line) {
			t.Fatalf("mcp --help does not list the signal status %q:\n%s", line, page)
		}
	}
}
