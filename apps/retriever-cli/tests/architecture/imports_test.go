// Package architecture checks the dependency direction of the module with the Go parser, not with a build. The core
// packages may import only other core packages and effect-free libraries; the shell (the adapters and the composition
// root) may import the core, and the core never imports the shell.
package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// module is the import path of this module.
const module = "github.com/wahidyankf/super-agent/apps/retriever-cli"

// coreRoot is the core source tree, relative to this package at apps/retriever-cli/tests/architecture.
const coreRoot = "../../internal/core"

// shellPrefixes are the import prefixes the core must never reach: the adapters, the composition root, and the tests.
var shellPrefixes = []string{
	module + "/internal/adapters",
	module + "/cmd",
	module + "/tests",
}

// effectfulStd are the standard-library packages that reach the world outside the process: the network, the file system,
// and processes. The core reaches them only through a port. The log/slog package is not listed: the core imports it only
// to return a redacted value from LogValue, which is pure, and the logger that writes it belongs to the shell.
var effectfulStd = map[string]bool{
	"net":           true,
	"net/http":      true,
	"os":            true,
	"os/exec":       true,
	"path/filepath": true,
	"io/fs":         true,
	"log":           true,
	"syscall":       true,
}

// coreImports parses the imports of every non-test Go file under the core tree. It returns each import path with the file
// that holds it.
func coreImports(t *testing.T) map[string][]string {
	t.Helper()
	found := map[string][]string{}
	err := filepath.WalkDir(coreRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			p, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			found[p] = append(found[p], path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("read the core imports: %v", err)
	}
	return found
}

// TestCoreImportsNoShell checks that no core file imports an adapter, the composition root, or a test package.
func TestCoreImportsNoShell(t *testing.T) {
	if _, err := os.Stat(coreRoot); err != nil {
		t.Fatalf("the core tree is missing: %v", err)
	}
	for imp, files := range coreImports(t) {
		for _, prefix := range shellPrefixes {
			if strings.HasPrefix(imp, prefix) {
				for _, f := range files {
					t.Errorf("%s imports %s, which is in the shell", f, imp)
				}
			}
		}
	}
}

// TestCoreImportsNoEffects checks that no core file imports a standard-library package that reaches the outside world.
func TestCoreImportsNoEffects(t *testing.T) {
	for imp, files := range coreImports(t) {
		if effectfulStd[imp] {
			for _, f := range files {
				t.Errorf("%s imports %s, which reaches outside the process; reach it through a port", f, imp)
			}
		}
	}
}
