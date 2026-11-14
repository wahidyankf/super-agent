package architecture

import (
	"strings"
)

// workspacePrefix is the import path prefix of every module in this repository.
const workspacePrefix = "github.com/wahidyankf/super-agent/"

// The kinds of module. Each one matches the type tag of its Nx project, which the integration test checks.
const (
	kindApp       = "app"
	kindContract  = "contract"
	kindRuntime   = "runtime"
	kindConnector = "connector"
	kindTestkit   = "testkit"
	kindSpec      = "spec"
)

// moduleKinds maps the directory of each Go module, relative to the repository root, to its kind. Every module of the
// workspace appears here, and the integration test checks the list against the files on disk.
var moduleKinds = map[string]string{
	"apps/retriever-cli":               kindApp,
	"libs/retriever-contract":          kindContract,
	"libs/retriever-runtime":           kindRuntime,
	"libs/retriever-testkit":           kindTestkit,
	"libs/retriever-connector-outline": kindConnector,
	"libs/retriever-connector-kiwi":    kindConnector,
	"libs/retriever-connector-jira":    kindConnector,
	"libs/retriever-spec":              kindSpec,
}

// pureDirs are the directories, relative to the repository root, whose production code reaches the outside world only
// through a port. The contract is pure, and so is the use-case package of the runtime, and so is the spec module, which
// reads a document it is given and touches no file or network.
var pureDirs = []string{
	"libs/retriever-contract",
	"libs/retriever-runtime/sources",
	"libs/retriever-spec",
}

// effectfulStd are the standard-library packages that reach the world outside the process: the network, the file system,
// and processes. A pure directory reaches them only through a port. The log/slog package is not listed: the core imports
// it only to return a redacted value from LogValue, which is pure, and the logger that writes it belongs to the shell.
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

// moduleOf returns the directory of the module that holds importPath, or "" when no module of the workspace holds it. The
// longest matching module directory wins, so a nested module would not be shadowed by its parent.
func moduleOf(importPath string) string {
	rel, ok := strings.CutPrefix(importPath, workspacePrefix)
	if !ok {
		return ""
	}
	best := ""
	for dir := range moduleKinds {
		if (rel == dir || strings.HasPrefix(rel, dir+"/")) && len(dir) > len(best) {
			best = dir
		}
	}
	return best
}

// importViolation returns why a file of the module in directory self may import importPath, or "" when it may. test is
// true for a _test.go file. A library imports the contract and nothing else of the workspace, a test may also import
// the test kit, and only the app imports the app-level packages.
func importViolation(self string, test bool, importPath string) string {
	if !strings.HasPrefix(importPath, workspacePrefix) {
		return ""
	}
	target := moduleOf(importPath)
	if target == "" {
		return "is in no module of the workspace"
	}
	if target == self {
		return ""
	}
	selfKind, targetKind := moduleKinds[self], moduleKinds[target]
	if targetKind == kindApp {
		return "is the app, and no library or other module may import the app"
	}
	if targetKind == kindTestkit {
		if test && selfKind != kindContract && selfKind != kindTestkit {
			return ""
		}
		if selfKind == kindApp && test {
			return ""
		}
		return "is the test kit, and only a test of a connector, a runtime, or the app may import it"
	}
	switch selfKind {
	case kindApp:
		return ""
	case kindContract:
		return "is the contract, and the contract may import no other module of the workspace"
	case kindRuntime, kindConnector, kindTestkit, kindSpec:
		if targetKind == kindContract {
			return ""
		}
		return "may import only the contract from the workspace"
	}
	return "belongs to a module of unknown kind " + selfKind
}

// effectViolation reports whether a file under a pure directory imports a package that reaches the outside world.
func effectViolation(importPath string) bool {
	return effectfulStd[importPath]
}
