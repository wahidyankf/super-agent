package envfile

import "testing"

// TestNewRequiresAnAbsolutePath checks that a credentials reader is never built for a relative env file path.
func TestNewRequiresAnAbsolutePath(t *testing.T) {
	if _, err := NewWithFiles("relative/root", noEnvironment, fakeFiles(nil, nil)); err == nil {
		t.Fatal("NewWithFiles accepted a relative path")
	}
}
