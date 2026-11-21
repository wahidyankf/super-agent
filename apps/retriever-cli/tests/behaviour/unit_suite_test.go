package behaviour

import "testing"

// The unit level runs every feature file in full, in memory, so each file's scenarios are covered at this level.
//
// @scenario specs/behaviours/catalog/config-discovery.feature: *
// @scenario specs/behaviours/catalog/sources-catalog.feature: *
// @scenario specs/behaviours/cli/cli-contract.feature: *
// @scenario specs/behaviours/connectors/spec-conformance.feature: *
// @scenario specs/behaviours/connectors/spec-connectors.feature: *
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

// TestUnitBehaviours runs every scenario in memory. They call the CLI in process, with no real file, environment,
// working directory, or child process. Their only servers are the loopback stubs, and no source is reached.
func TestUnitBehaviours(t *testing.T) {
	runSuite(t, levelUnit, "", newMemoryBackend)
}
