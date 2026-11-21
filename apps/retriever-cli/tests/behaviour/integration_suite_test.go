//go:build integration

package behaviour

import "testing"

// The integration level runs every feature file in full, against the real adapters, so each file's scenarios are
// covered at this level. No feature carries an integration exemption.
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

// TestIntegrationBehaviours runs every scenario that is not exempt at the integration level. They call the real adapters
// in process, against the loopback stub servers.
func TestIntegrationBehaviours(t *testing.T) {
	runSuite(t, levelIntegration, "@integration-exempt", func(outer *testing.T) backend {
		return newDiskBackend(outer, true, "")
	})
}
