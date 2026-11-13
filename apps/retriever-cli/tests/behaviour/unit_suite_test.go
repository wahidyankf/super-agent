package behaviour

import "testing"

// TestUnitBehaviours runs the @unit scenarios. They call the CLI in process with no source and no network.
func TestUnitBehaviours(t *testing.T) {
	runSuite(t, levelUnit, "@unit", "")
}
