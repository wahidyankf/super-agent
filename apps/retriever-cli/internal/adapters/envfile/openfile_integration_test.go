//go:build integration

package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOpenFileRefusesAMissingPathWithoutAValue checks the real opener on a path that does not exist: it returns an error
// and a nil reader, so the caller's nil check is never fooled by a typed nil.
func TestOpenFileRefusesAMissingPathWithoutAValue(t *testing.T) {
	f, err := openFile("/nonexistent-retriever-test-folder/.env.local")
	if err == nil || f != nil {
		t.Fatalf("openFile = %v, %v; want an error and no reader", f, err)
	}
}

// TestOpenFileOpensARealFileForReading checks the real opener on one file in a test folder: the reader describes that file.
func TestOpenFileOpensARealFileForReading(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.local")
	if err := os.WriteFile(path, []byte("OUTLINE_API_KEY=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := openFile(path)
	if err != nil {
		t.Fatalf("openFile: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	opened, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat of the opened file: %v", err)
	}
	want, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(opened, want) {
		t.Fatalf("the opened file is not the file at the path")
	}
}
