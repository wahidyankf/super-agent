package architecture

import (
	"go/ast"
	"strings"
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
