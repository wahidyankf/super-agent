//go:build integration

package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestUntaggedTestsNeverChangeTheEnvironment checks that no untagged test calls a function that changes the process
// environment. The unit level runs every untagged test in process, so such a test must become an integration test, or
// it must receive its environment as a parameter. The check reads the direct calls of each file. It does not follow
// calls into helpers, and it does not check writes outside t.TempDir: proving that a path came from t.TempDir needs
// data-flow analysis, which a parser does not give.
func TestUntaggedTestsNeverChangeTheEnvironment(t *testing.T) {
	changing := map[string]bool{}
	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(moduleRoot, path)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		if hasBuildConstraint(file) {
			return nil
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !environmentMutators[sel.Sel.Name] {
				return true
			}
			changing[rel] = true
			if _, exempt := environmentExemptions[rel]; !exempt {
				t.Errorf("%s:%d calls %s, which changes the process environment in an untagged test; receive the environment "+
					"as a parameter, or tag the test as integration", rel, fset.Position(call.Pos()).Line, sel.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("read the test sources: %v", err)
	}
	for rel := range environmentExemptions {
		if !changing[rel] {
			t.Errorf("%s no longer changes the environment; remove it from environmentExemptions", rel)
		}
	}
}
