//go:build integration

package behaviour

import "testing"

// TestIntegrationBehaviours runs the @integration scenarios. They call the real adapters in process, against the
// loopback stub servers.
func TestIntegrationBehaviours(t *testing.T) {
	runSuite(t, levelIntegration, "@integration", "")
}
