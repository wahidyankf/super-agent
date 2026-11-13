package behaviour

import "testing"

// TestUnitBehaviours runs the @unit scenarios in memory. They call the CLI in process, with no real file, environment,
// working directory, or child process. Their only servers are the loopback stubs, and no source is reached.
func TestUnitBehaviours(t *testing.T) {
	runSuite(t, levelUnit, "@unit", newMemoryBackend)
}
