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

// moduleRoot is the module tree, relative to this package at apps/retriever-cli/tests/architecture.
const moduleRoot = "../.."

// environmentMutators are the calls that change the process environment: testing.T.Setenv, whose receiver is any value,
// and the os package's Setenv, Unsetenv, and Clearenv. The check matches the selector name, so it needs no type
// information.
var environmentMutators = map[string]bool{"Setenv": true, "Unsetenv": true, "Clearenv": true}

// environmentExemptions names the untagged test files that still change the process environment, each with its reason.
// The list may shrink but never grow, and a file that stops changing the environment must leave it.
var environmentExemptions = map[string]string{
	"tests/behaviour/world_test.go": "the in-process behaviour harness runs the unit and integration levels in one file " +
		"and sets the process environment for both; the integration-only part belongs in a tagged file (open item)",
}

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

// hasBuildConstraint reports whether the file carries a //go:build line before its package clause. Such a file runs only
// at the level its tag names.
func hasBuildConstraint(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, c := range group.List {
			if strings.HasPrefix(c.Text, "//go:build") {
				return true
			}
		}
	}
	return false
}
