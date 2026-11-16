//go:build integration

package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// profileWriteMembers are the os members that write a file or a folder. A production reference to one outside the
// allowlist below means the tool writes somewhere its contract does not name.
var profileWriteMembers = map[string]bool{
	"Create": true, "CreateTemp": true, "Mkdir": true, "MkdirAll": true, "OpenFile": true,
	"Remove": true, "RemoveAll": true, "Rename": true, "WriteFile": true,
}

// profileWriteAllowlist names the production files that may reference a write member, with why. The export sink is
// the tool's only writer, and it writes into the --out folder the caller names, never into the user configuration
// folder that holds the profiles.
var profileWriteAllowlist = map[string]map[string]string{
	"apps/retriever-cli/cmd/retriever/disk.go": {
		"CreateTemp": "the export sink writes its temporary file into the --out folder",
		"MkdirAll":   "the export sink creates the --out folder",
		"Remove":     "the export sink removes its temporary file",
		"Rename":     "the export sink moves the finished file into the --out folder",
	},
}

// TestProductionWritesStayInTheExportSink checks F6: no production package writes to the user configuration folder,
// because the export sink is the only writer and its folder is the caller's --out folder. The walk reads the files as
// they are on disk; the rule is a name check over the parsed calls.
func TestProductionWritesStayInTheExportSink(t *testing.T) {
	roots := []string{"apps/retriever-cli", "libs"}
	for _, root := range roots {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// The test kit is test scaffolding, not production code: its stub commands record test traffic into
			// files the tests name, never into the user configuration folder.
			if d.IsDir() && path == filepath.Join(repoRoot, "libs", "retriever-testkit") {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			for _, finding := range profileWriteRefs(t, path) {
				t.Errorf("%s", finding)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("read the sources under %s: %v", root, err)
		}
	}
}

// profileWriteRefs returns one finding per write-member call the production file at path makes outside the allowlist.
// A finding names the file relative to the repository root, the line, and the reason the call is refused.
func profileWriteRefs(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	names := map[string]string{}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if importPath != "os" && importPath != "io/ioutil" {
			continue
		}
		name := "os"
		if strings.HasSuffix(importPath, "ioutil") {
			name = "ioutil"
		}
		if spec.Name != nil {
			name = spec.Name.Name
		}
		names[name] = importPath
	}
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("relative path of %s: %v", path, err)
	}
	rel = filepath.ToSlash(rel)
	var findings []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if _, ok := names[id.Name]; !ok || !profileWriteMembers[sel.Sel.Name] {
			return true
		}
		if reason, ok := profileWriteAllowlist[rel][sel.Sel.Name]; ok && reason != "" {
			return true
		}
		line := fset.Position(call.Pos()).Line
		findings = append(findings, rel+":"+strconv.Itoa(line)+" calls os."+sel.Sel.Name+
			", which writes outside the export sink")
		return true
	})
	return findings
}
