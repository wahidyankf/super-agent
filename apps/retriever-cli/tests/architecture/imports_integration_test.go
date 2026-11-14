//go:build integration

package architecture

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// repoRoot is the repository root, relative to this package at apps/retriever-cli/tests/architecture.
const repoRoot = "../../../.."

// projectFile is the part of an Nx project.json that the layering check reads.
type projectFile struct {
	Name                 string   `json:"name"`
	Tags                 []string `json:"tags"`
	ImplicitDependencies []string `json:"implicitDependencies"`
}

// readProject returns the Nx project of the module in dir, relative to the repository root.
func readProject(t *testing.T, dir string) projectFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, dir, "project.json"))
	if err != nil {
		t.Fatalf("read the Nx project of %s: %v", dir, err)
	}
	var project projectFile
	if err := json.Unmarshal(data, &project); err != nil {
		t.Fatalf("parse the Nx project of %s: %v", dir, err)
	}
	return project
}

// goFile is one Go source file of a module, with the import paths it names.
type goFile struct {
	path    string
	test    bool
	imports []string
}

// walkGoFiles returns every Go file under the directory dir, relative to the repository root. The imports of each file
// are read without its build constraints, so a file with a tag is checked as well as an untagged one.
func walkGoFiles(t *testing.T, dir string) []goFile {
	t.Helper()
	var files []goFile
	root := filepath.Join(repoRoot, dir)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "dist" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		f := goFile{path: path, test: strings.HasSuffix(path, "_test.go")}
		for _, spec := range file.Imports {
			p, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			f.imports = append(f.imports, p)
		}
		files = append(files, f)
		return nil
	})
	if err != nil {
		t.Fatalf("read the Go files of %s: %v", dir, err)
	}
	return files
}

// modulesOnDisk returns the directories under libs that hold a go.mod file, relative to the repository root.
func modulesOnDisk(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repoRoot, "libs"))
	if err != nil {
		t.Fatalf("read the libs directory: %v", err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(repoRoot, "libs", e.Name(), "go.mod")); err == nil {
			dirs = append(dirs, "libs/"+e.Name())
		}
	}
	return dirs
}

// TestWorkspaceDeclaresEveryModule checks that the libs on disk are the modules the layering table names, and that each
// table entry has a go.mod file.
func TestWorkspaceDeclaresEveryModule(t *testing.T) {
	for _, dir := range modulesOnDisk(t) {
		if _, ok := moduleKinds[dir]; !ok {
			t.Errorf("%s has a go.mod but no kind in the layering table; add it to moduleKinds", dir)
		}
	}
	for dir := range moduleKinds {
		if _, err := os.Stat(filepath.Join(repoRoot, dir, "go.mod")); err != nil {
			t.Errorf("module %s is in the layering table but has no go.mod: %v", dir, err)
		}
	}
}

// siblingDirectives returns the workspace modules that the go.mod of dir requires, and the directory each one is
// replaced with. go mod tidy and the build resolve a sibling through these lines, so they must name every sibling the
// Go files import, and nothing more.
func siblingDirectives(t *testing.T, dir string) (required map[string]bool, replaced map[string]string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, dir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod of %s: %v", dir, err)
	}
	required = map[string]bool{}
	replaced = map[string]string{}
	block := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == ")" {
			block = ""
			continue
		}
		kind := block
		if kind == "" {
			switch {
			case line == "require (":
				block = "require"
				continue
			case line == "replace (":
				block = "replace"
				continue
			case strings.HasPrefix(line, "require "):
				kind, line = "require", strings.TrimPrefix(line, "require ")
			case strings.HasPrefix(line, "replace "):
				kind, line = "replace", strings.TrimPrefix(line, "replace ")
			default:
				continue
			}
		}
		switch kind {
		case "require":
			if path, ok := strings.CutPrefix(strings.Fields(line)[0], workspacePrefix); ok {
				required[path] = true
			}
		case "replace":
			lhs, rhs, found := strings.Cut(line, "=>")
			if !found {
				continue
			}
			if path, ok := strings.CutPrefix(strings.Fields(lhs)[0], workspacePrefix); ok {
				replaced[path] = strings.Fields(rhs)[0]
			}
		}
	}
	return required, replaced
}

// TestGoModulesRequireAndReplaceTheSiblingsTheyImport checks that each module names, in its go.mod, every sibling module
// its Go files import, with a require line and a replace line to the sibling's directory, and no sibling it does not import.
func TestGoModulesRequireAndReplaceTheSiblingsTheyImport(t *testing.T) {
	for dir := range moduleKinds {
		required, replaced := siblingDirectives(t, dir)
		imported := map[string]bool{}
		for _, f := range walkGoFiles(t, dir) {
			for _, imp := range f.imports {
				if target := moduleOf(imp); target != "" && target != dir {
					imported[target] = true
				}
			}
		}
		for target := range imported {
			if !required[target] {
				t.Errorf("%s/go.mod does not require %s%s, which its Go files import", dir, workspacePrefix, target)
			}
			rel, err := filepath.Rel(dir, target)
			if err != nil {
				t.Fatalf("relative path from %s to %s: %v", dir, target, err)
			}
			if replaced[target] != filepath.ToSlash(rel) {
				t.Errorf("%s/go.mod replaces %s with %q; want %q", dir, target, replaced[target], filepath.ToSlash(rel))
			}
		}
		for target := range required {
			if !imported[target] {
				t.Errorf("%s/go.mod requires %s%s, which no Go file of the module imports", dir, workspacePrefix, target)
			}
		}
	}
}

// TestEveryModuleHasItsPathAndNxProject checks that each module's module path follows its directory, and that its Nx
// project carries the type tag of its kind.
func TestEveryModuleHasItsPathAndNxProject(t *testing.T) {
	for dir, kind := range moduleKinds {
		data, err := os.ReadFile(filepath.Join(repoRoot, dir, "go.mod"))
		if err != nil {
			t.Fatalf("read go.mod of %s: %v", dir, err)
		}
		first := strings.SplitN(string(data), "\n", 2)[0]
		if want := "module " + workspacePrefix + dir; strings.TrimSpace(first) != want {
			t.Errorf("%s/go.mod starts with %q; want %q", dir, strings.TrimSpace(first), want)
		}
		project := readProject(t, dir)
		if !contains(project.Tags, "type:"+kind) {
			t.Errorf("the Nx project %s has tags %v; want type:%s", project.Name, project.Tags, kind)
		}
	}
}

// TestImportsFollowTheLayering checks every Go file of every module, tests included, against the layering table.
func TestImportsFollowTheLayering(t *testing.T) {
	for dir := range moduleKinds {
		for _, f := range walkGoFiles(t, dir) {
			for _, imp := range f.imports {
				if reason := importViolation(dir, f.test, imp); reason != "" {
					rel, err := filepath.Rel(repoRoot, f.path)
					if err != nil {
						t.Fatalf("relative path of %s: %v", f.path, err)
					}
					t.Errorf("%s imports %s, which %s", filepath.ToSlash(rel), imp, reason)
				}
			}
		}
	}
}

// TestNxImplicitDependenciesNameEveryImportedModule checks that each module's Nx project lists, in implicitDependencies,
// every other module its Go files import. Nx cannot read Go imports, so this list is what orders the affected builds.
func TestNxImplicitDependenciesNameEveryImportedModule(t *testing.T) {
	projectNames := map[string]string{}
	for dir := range moduleKinds {
		projectNames[dir] = readProject(t, dir).Name
	}
	for dir := range moduleKinds {
		imported := map[string]bool{}
		for _, f := range walkGoFiles(t, dir) {
			for _, imp := range f.imports {
				if target := moduleOf(imp); target != "" && target != dir {
					imported[target] = true
				}
			}
		}
		declared := readProject(t, dir).ImplicitDependencies
		for target := range imported {
			if !contains(declared, projectNames[target]) {
				t.Errorf("the Nx project %s imports %s but its implicitDependencies %v do not name it", projectNames[dir], projectNames[target], declared)
			}
		}
	}
}

// TestPureDirectoriesReachNoEffects checks that the production code of each pure directory imports no standard-library
// package that reaches the outside world. The pure code reaches it through a port.
func TestPureDirectoriesReachNoEffects(t *testing.T) {
	for _, dir := range pureDirs {
		for _, f := range walkGoFiles(t, dir) {
			if f.test {
				continue
			}
			for _, imp := range f.imports {
				if effectViolation(imp) {
					rel, err := filepath.Rel(repoRoot, f.path)
					if err != nil {
						t.Fatalf("relative path of %s: %v", f.path, err)
					}
					t.Errorf("%s imports %s, which reaches outside the process; reach it through a port", filepath.ToSlash(rel), imp)
				}
			}
		}
	}
}

// contains reports whether the list holds the value.
func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
