//go:build integration

package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

// newRoot returns a temporary repository root. A non-empty envContent becomes the root's local env file,
// which only the test program creates and reads.
func newRoot(t *testing.T, envContent string) string {
	t.Helper()
	root := t.TempDir()
	if envContent != "" {
		mustWrite(t, filepath.Join(root, envFileName), envContent)
	}
	return root
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWrite(t *testing.T, file, content string) {
	t.Helper()
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", file, err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %s -> %s: %v", link, target, err)
	}
}

// unsetEnv removes a process variable for the rest of the test. t.Setenv restores the original value afterwards.
func unsetEnv(t *testing.T, name string) {
	t.Helper()
	t.Setenv(name, "")
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("unset %s: %v", name, err)
	}
}

// newCredentials builds Credentials over root and fails the test when the root is refused.
func newCredentials(t *testing.T, root string) *Credentials {
	t.Helper()
	c, err := newReal(filepath.Join(root, envFileName))
	if err != nil {
		t.Fatalf("New(%q): %v", root, err)
	}
	return c
}
