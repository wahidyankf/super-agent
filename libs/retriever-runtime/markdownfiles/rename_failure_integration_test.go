//go:build integration

package markdownfiles

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteReportsAnExportThatCannotReplaceAFolder checks that a file name that is already a folder fails the rename, and
// that the temporary file the write made is removed, so the folder holds nothing new.
func TestWriteReportsAnExportThatCannotReplaceAFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "doc.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := realSink(dir).Write(context.Background(), "doc.md", []byte("body")); err == nil {
		t.Fatal("Write over a folder of the same name succeeded; want an error")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "doc.md" {
		t.Fatalf("folder holds %d entries, want only the folder that was already there", len(entries))
	}
}

// TestWriteReportsAFolderThatCannotTakeAFile checks that a folder the process cannot write to fails the temporary file
// step, before any export is placed. The mode is restored so the test folder can be removed.
func TestWriteReportsAFolderThatCannotTakeAFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if _, err := realSink(dir).Write(context.Background(), "doc.md", []byte("body")); err == nil {
		t.Fatal("Write into a read-only folder succeeded; want an error")
	}
}
