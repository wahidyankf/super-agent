package cli

import (
	"strings"
	"testing"
)

// TestParseGlobalsReadsTheConnectorsFolderInAnyPosition checks that --connectors-dir is a path option like --catalog.
func TestParseGlobalsReadsTheConnectorsFolderInAnyPosition(t *testing.T) {
	g, err := ParseGlobals([]string{"--connectors-dir", "/work/specs", "sources", "list"})
	if err != nil || g.ConnectorsDir != "/work/specs" {
		t.Fatalf("ParseGlobals = %+v, %v; want the folder before the command", g, err)
	}
	g, err = ParseGlobals([]string{"sources", "list", "--connectors-dir=/work/specs"})
	if err != nil || g.ConnectorsDir != "/work/specs" {
		t.Fatalf("ParseGlobals = %+v, %v; want the folder given with an equals sign", g, err)
	}
}

// TestConnectorsFolderWithoutAValueIsAUsageError checks that a missing folder path is refused like a missing catalog path.
func TestConnectorsFolderWithoutAValueIsAUsageError(t *testing.T) {
	_, err := ParseGlobals([]string{"sources", "list", "--connectors-dir"})
	if err == nil || err.Error() != "missing value for --connectors-dir" {
		t.Fatalf("ParseGlobals error = %v, want a missing value", err)
	}
}

// TestHelpListsTheConnectorsFolderOption checks that the global help names the new option with its value.
func TestHelpListsTheConnectorsFolderOption(t *testing.T) {
	for _, row := range globalOptionRows() {
		if strings.HasPrefix(row[0], "--connectors-dir <path>") {
			return
		}
	}
	t.Fatal("the global options do not list --connectors-dir <path>")
}
