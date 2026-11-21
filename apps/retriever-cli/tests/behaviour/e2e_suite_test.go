//go:build e2e

package behaviour

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The end-to-end level runs the feature files against the built binary. spec-connectors.feature carries two end-to-end
// exemptions, so its non-exempt scenarios are named below one by one; a `*` there would also cover the exempt ones (B06).
// Every other feature runs in full at this level.
//
// @scenario specs/behaviours/catalog/config-discovery.feature: *
// @scenario specs/behaviours/catalog/sources-catalog.feature: *
// @scenario specs/behaviours/cli/cli-contract.feature: *
// @scenario specs/behaviours/connectors/spec-conformance.feature: *
// @scenario specs/behaviours/connectors/spec-connectors.feature: A spec operation with the CLI off and the MCP on has no command and keeps its tool
// @scenario specs/behaviours/connectors/spec-connectors.feature: A spec operation with the CLI on and the MCP off has a command and no tool
// @scenario specs/behaviours/connectors/spec-connectors.feature: A catalog pin that does not match the spec file fails with the pin code and sends no request
// @scenario specs/behaviours/connectors/spec-connectors.feature: A spec that declares a header it may not set stops start-up with its file and field path and no value
// @scenario specs/behaviours/connectors/spec-connectors.feature: A spec kind that names a built-in kind or a command word stops start-up
// @scenario specs/behaviours/connectors/spec-connectors.feature: A second spec with the kind of the first stops start-up and names the second file
// @scenario specs/behaviours/connectors/spec-connectors.feature: A loaded spec leaves the sources of the built-in kinds listed as they were
// @scenario specs/behaviours/connectors/spec-connectors.feature: A kind filter with an unknown kind names only the built-in kinds when a spec is loaded
// @scenario specs/behaviours/connectors/spec-connectors.feature: An explicit connectors folder that does not exist exits 2 with its code and names no path
// @scenario specs/behaviours/connectors/spec-connectors.feature: A default connectors folder that does not exist is not an error
// @scenario specs/behaviours/connectors/spec-connectors.feature: A spec operation that needs a Markdown converter fails closed at run time and sends no request
// @scenario specs/behaviours/connectors/spec-connectors.feature: Only yml and yaml files in the connectors folder are read as connector specs
// @scenario specs/behaviours/credentials/credentials.feature: *
// @scenario specs/behaviours/export/export-markdown-files.feature: *
// @scenario specs/behaviours/graphql/graphql-serve.feature: *
// @scenario specs/behaviours/jira/jira-operations.feature: *
// @scenario specs/behaviours/kiwi/kiwi-operations.feature: *
// @scenario specs/behaviours/mcp/mcp-server.feature: *
// @scenario specs/behaviours/outline/outline-critical-actions.feature: *
// @scenario specs/behaviours/outline/outline-reads.feature: *
// @scenario specs/behaviours/outline/outline-redaction.feature: *
// @scenario specs/behaviours/outline/outline-writes.feature: *
// @scenario specs/behaviours/transport/https-and-limits.feature: *

// e2eBinary is the built retriever executable that the end-to-end scenarios run. TestMain builds it once per test run.
var e2eBinary string

// TestMain builds the retriever binary from the module root, runs the tests, and removes the build folder.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "retriever-e2e-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create the build folder: %v\n", err)
		os.Exit(1)
	}
	bin := filepath.Join(dir, "retriever")
	build := exec.Command("go", "build", "-trimpath", "-o", bin, "./cmd/retriever")
	build.Dir = "../.."
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build the retriever binary: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	e2eBinary = bin
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// TestE2EBehaviours runs every scenario that is not exempt at the end-to-end level, against the built binary.
func TestE2EBehaviours(t *testing.T) {
	runSuite(t, levelE2E, "@e2e-exempt", func(outer *testing.T) backend {
		return newDiskBackend(outer, false, e2eBinary)
	})
}
