//go:build integration

package markdownfiles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteStoresTheBytesUnderTheFolderAndReturnsThePath checks the default sink: the file is written inside the folder
// the sink was given, and the location it reports is that file.
func TestWriteStoresTheBytesUnderTheFolderAndReturnsThePath(t *testing.T) {
	dir := t.TempDir()
	sink := realSink(dir)
	got, err := sink.Write(context.Background(), "getting-started.md", []byte("# Hi\n"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	want := filepath.Join(dir, "getting-started.md")
	if got != want {
		t.Fatalf("location = %q; want %q", got, want)
	}
	data, err := os.ReadFile(want)
	if err != nil || string(data) != "# Hi\n" {
		t.Fatalf("file = %q, %v; want the Markdown bytes", data, err)
	}
}

// TestWriteCreatesTheFolderWhenItIsMissing checks that the --out folder is created on the first write.
func TestWriteCreatesTheFolderWhenItIsMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "exports", "outline")
	if _, err := realSink(dir).Write(context.Background(), "doc.md", []byte("x")); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "doc.md")); err != nil {
		t.Fatalf("the file was not created: %v", err)
	}
}

// TestWriteReplacesAnExistingFileAndLeavesNoTemporaryFile checks that a second export replaces the file, and that the
// folder holds only the exported file afterwards.
func TestWriteReplacesAnExistingFileAndLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	sink := realSink(dir)
	if _, err := sink.Write(context.Background(), "doc.md", []byte("first")); err != nil {
		t.Fatalf("first Write error: %v", err)
	}
	if _, err := sink.Write(context.Background(), "doc.md", []byte("second")); err != nil {
		t.Fatalf("second Write error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "doc.md"))
	if err != nil || string(data) != "second" {
		t.Fatalf("file = %q, %v; want the second export", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("folder holds %d entries, %v; want only doc.md", len(entries), err)
	}
}

// TestWriteRefusesAnyNameThatIsNotAPlainFileName checks that a name cannot leave the folder: a separator, a parent
// reference, an empty name, a NUL byte, or a name without the .md extension is refused before anything is written.
func TestWriteRefusesAnyNameThatIsNotAPlainFileName(t *testing.T) {
	names := []string{"", ".", "..", "../escape.md", "sub/doc.md", `sub\doc.md`, "doc\x00.md", "doc.txt", ".md"}
	for _, name := range names {
		t.Run(strings.ReplaceAll(name, "\x00", "NUL"), func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "out")
			if _, err := realSink(dir).Write(context.Background(), name, []byte("x")); err == nil {
				t.Fatalf("Write(%q) succeeded; want a refusal", name)
			}
			// The name is checked before anything is created, so a refused write leaves the folder untouched.
			if entries, _ := os.ReadDir(root); len(entries) != 0 {
				t.Fatalf("a refused name %q left %d entries behind", name, len(entries))
			}
		})
	}
}

// TestWriteReportsAFolderThatIsAFile checks that a folder which is really a file yields an error, not a partial write.
func TestWriteReportsAFolderThatIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-folder")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := realSink(file).Write(context.Background(), "doc.md", []byte("y")); err == nil {
		t.Fatal("Write into a file succeeded; want an error")
	}
}
