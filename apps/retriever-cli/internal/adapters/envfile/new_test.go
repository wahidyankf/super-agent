package envfile

import "testing"

// TestNewRefusesARelativeRepositoryRoot proves New refuses a relative root before any file is touched, so this test
// needs no temporary folder and runs at the unit layer.
func TestNewRefusesARelativeRepositoryRoot(t *testing.T) {
	if _, err := New("relative/root"); err == nil {
		t.Fatal("New accepted a relative root, want an error")
	}
}
