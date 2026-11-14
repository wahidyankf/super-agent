//go:build integration

package architecture

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hermeticRoots are the trees the guard walks, relative to the repository root: the app and every library. The guard
// covers the whole workspace, because each library's untagged tests run in the same unit level.
var hermeticRoots = []string{"apps/retriever-cli", "libs"}

// TestUntaggedTestsStayHermetic checks every untagged test file of the workspace with the rule in hermetic_test.go. An
// untagged test runs in the unit level's one process, so it may not use the real disk, change the environment, start a
// child process, or use a pipe. A test that needs any of those carries the integration or e2e tag, in its own file, and
// receives the file system, the environment, or the process through a seam. The walk reads the files as they are on
// disk; the rule itself is pure and is tested against in-memory source in hermetic_test.go.
func TestUntaggedTestsStayHermetic(t *testing.T) {
	folders := map[string]map[string][]byte{}
	for _, root := range hermeticRoots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			dir := filepath.Dir(path)
			if folders[dir] == nil {
				folders[dir] = map[string][]byte{}
			}
			folders[dir][filepath.Base(path)] = data
			return nil
		})
		if err != nil {
			t.Fatalf("read the test sources under %s: %v", root, err)
		}
	}
	behaviour := filepath.Join(repoRoot, "apps", "retriever-cli", "tests", "behaviour")
	if _, ok := folders[behaviour]; !ok {
		t.Fatalf("the walk found no test file in %s; the repository root %q is wrong", behaviour, repoRoot)
	}
	outline := filepath.Join(repoRoot, "libs", "retriever-connector-outline", "outline")
	if _, ok := folders[outline]; !ok {
		t.Fatalf("the walk found no test file in %s; the library roots are wrong", outline)
	}
	for dir, files := range folders {
		findings, err := analyzeDir(files)
		if err != nil {
			t.Fatalf("analyze %s: %v", dir, err)
		}
		for _, f := range findings {
			rel, err := filepath.Rel(repoRoot, filepath.Join(dir, f.File))
			if err != nil {
				t.Fatalf("relative path of %s: %v", f.File, err)
			}
			t.Errorf("%s:%d uses %s, which an untagged test may not use; move the test to a file with the integration tag, "+
				"or pass the file system, the environment, or the process in as a seam", filepath.ToSlash(rel), f.Line, f.Use)
		}
	}
}
