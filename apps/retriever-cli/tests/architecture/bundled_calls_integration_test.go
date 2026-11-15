//go:build integration

package architecture

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestParseBundledHasOneCaller proves the L9 control that Parse serves the user folder and ParseBundled serves the
// embed: exactly one production file of the workspace calls ParseBundled, and it is the loader of the embedded specs.
// A test may call it, and the definition itself does not count.
func TestParseBundledHasOneCaller(t *testing.T) {
	files := map[string][]byte{}
	for _, root := range hermeticRoots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = data
			return nil
		})
		if err != nil {
			t.Fatalf("read the Go sources under %s: %v", root, err)
		}
	}
	got := bundledCallers(files)
	want := []string{"apps/retriever-cli/internal/connectorspec/bundled.go"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("ParseBundled callers = %q, want %q", got, want)
	}
}
