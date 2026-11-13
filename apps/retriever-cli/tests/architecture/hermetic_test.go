package architecture

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The untagged unit level runs every untagged test in one process, in memory, with no real file, no environment change,
// and no child process. The rule is enforced here by reading the source of each untagged test file. The check is pure:
// it parses the source it is given and reads no disk, so it runs at the unit level too. The repository walk that feeds
// it is in hermetic_integration_test.go.

// forbiddenPackages maps an import path to the packages whose every member an untagged test may not reference.
var forbiddenPackages = map[string]bool{
	"os/exec": true,
}

// forbiddenMembers maps an import path to the members of that package an untagged test may not reference. A reference
// is any use of the name, a call or a value, because a value can be called later through a field or a variable.
var forbiddenMembers = map[string]map[string]bool{
	"os": set(
		// The environment, and the user folders that the environment names.
		"Setenv", "Unsetenv", "Clearenv", "Getenv", "LookupEnv", "Environ", "UserHomeDir", "UserConfigDir", "UserCacheDir",
		// The working directory and the temporary folders.
		"Chdir", "Getwd", "TempDir", "MkdirTemp", "CreateTemp",
		// Files and folders on the real disk.
		"Open", "OpenFile", "Create", "Mkdir", "MkdirAll", "ReadFile", "WriteFile", "ReadDir", "Remove", "RemoveAll",
		"Rename", "Stat", "Lstat", "Chmod", "Chown", "Lchown", "Chtimes", "Symlink", "Link", "Readlink", "Truncate", "DirFS",
		// Processes and pipes.
		"Pipe", "StartProcess", "Executable",
	),
	"io/ioutil":     set("ReadFile", "WriteFile", "ReadDir", "TempDir", "TempFile"),
	"path/filepath": set("Walk", "WalkDir", "Glob", "Abs", "EvalSymlinks"),
}

// forbiddenMethods are method names that an untagged test may not call on any receiver. testing.T.TempDir and
// testing.T.Setenv are the two that reach the disk or the environment. The check matches the name, so it needs no type
// information. A value of another type that shares one of these names is renamed, not excepted.
var forbiddenMethods = map[string]bool{
	"TempDir": true,
	"Setenv":  true,
}

// set returns the names as a set.
func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// finding is one forbidden use in an untagged test file.
type finding struct {
	File string // the base name of the file
	Line int
	Use  string // the import path and member, such as "os.Setenv", or "method TempDir"
}

// compiledUntagged reports whether a file compiles in the untagged build, where no build tag is set. Its
// //go:build line, when it has one, must hold with every tag false. A file tagged integration or e2e compiles only
// at its own level, so the untagged rule does not reach it.
func compiledUntagged(file *ast.File) (bool, error) {
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, c := range group.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				return false, err
			}
			return expr.Eval(func(string) bool { return false }), nil
		}
	}
	return true, nil
}

// importNames maps the name each import is used by in a file to its import path. A blank or dot import binds no name.
func importNames(file *ast.File) map[string]string {
	names := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "_" || name == "." {
			continue
		}
		names[name] = path
	}
	return names
}

// analyzeDir checks the untagged test files of one package directory. sources maps the base name of each file to its
// source. Only names that end in _test.go are checked, and a file that compiles only at a tagged level is skipped.
//
// Every declaration and every initialiser of an untagged file is inspected, so a forbidden use is found where it is
// written, in whichever helper or method holds it. That reaches the indirect uses too: a test that calls a helper which
// clears the environment is clean in its own body, and the helper's body is still checked.
func analyzeDir(sources map[string][]byte) ([]finding, error) {
	names := make([]string, 0, len(sources))
	for name := range sources {
		if strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var findings []finding
	for _, name := range names {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, name, sources[name], parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		untagged, err := compiledUntagged(file)
		if err != nil {
			return nil, fmt.Errorf("read the build constraint of %s: %w", name, err)
		}
		if !untagged {
			continue
		}
		imports := importNames(file)
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if use, forbidden := forbiddenUse(sel, imports); forbidden {
				findings = append(findings, finding{File: name, Line: fset.Position(sel.Pos()).Line, Use: use})
			}
			return true
		})
	}
	return findings, nil
}

// forbiddenUse reports whether one selector is a forbidden use, and names it. A selector whose base is an imported
// package is checked against that package's members. Any other selector is checked by its method name.
func forbiddenUse(sel *ast.SelectorExpr, imports map[string]string) (string, bool) {
	member := sel.Sel.Name
	if base, ok := sel.X.(*ast.Ident); ok {
		if path, isImport := imports[base.Name]; isImport {
			if forbiddenPackages[path] || forbiddenMembers[path][member] {
				return path + "." + member, true
			}
			return "", false
		}
	}
	if forbiddenMethods[member] {
		return "method " + member, true
	}
	return "", false
}

// untaggedFile returns a package behaviour test file with the given declarations and the harness imports. The parser
// does not check that each import is used.
func untaggedFile(decls string) string {
	return "package behaviour\n\n" + harnessImports + "\n" + decls + "\n"
}

// harnessImports is the import block of the fixture files: the packages whose uses the rule names, and testing.
const harnessImports = `import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)`

// TestGuardFlagsEveryForbiddenForm proves, on in-memory source, that the rule flags each form it names. Each case is
// one untagged test file that holds one forbidden use.
func TestGuardFlagsEveryForbiddenForm(t *testing.T) {
	cases := []struct {
		name string
		use  string
		src  string
	}{
		{"testing TempDir", "method TempDir", untaggedFile(`func TestA(t *testing.T) { _ = t.TempDir() }`)},
		{"testing Setenv", "method Setenv", untaggedFile(`func TestA(t *testing.T) { t.Setenv("K", "v") }`)},
		{"os Setenv", "os.Setenv", untaggedFile(`func TestA(t *testing.T) { _ = os.Setenv("K", "v") }`)},
		{"os Unsetenv", "os.Unsetenv", untaggedFile(`func TestA(t *testing.T) { _ = os.Unsetenv("K") }`)},
		{"os Clearenv", "os.Clearenv", untaggedFile(`func TestA(t *testing.T) { os.Clearenv() }`)},
		{"os Getenv", "os.Getenv", untaggedFile(`func TestA(t *testing.T) { _ = os.Getenv("K") }`)},
		{"os Chdir", "os.Chdir", untaggedFile(`func TestA(t *testing.T) { _ = os.Chdir("/") }`)},
		{"os Getwd", "os.Getwd", untaggedFile(`func TestA(t *testing.T) { _, _ = os.Getwd() }`)},
		{"os WriteFile", "os.WriteFile", untaggedFile(`func TestA(t *testing.T) { _ = os.WriteFile("f", nil, 0o600) }`)},
		{"os MkdirAll", "os.MkdirAll", untaggedFile(`func TestA(t *testing.T) { _ = os.MkdirAll("d", 0o755) }`)},
		{"os Pipe", "os.Pipe", untaggedFile(`func TestA(t *testing.T) { _, _, _ = os.Pipe() }`)},
		{"os Open as a value", "os.Open", untaggedFile(`var open = os.Open`)},
		{"os ReadFile in a struct literal", "os.ReadFile",
			untaggedFile(`var files = struct{ read func(string) ([]byte, error) }{read: os.ReadFile}`)},
		{"aliased os import", "os.Setenv",
			"package behaviour\n\nimport fsys \"os\"\n\nfunc TestA(t *testing.T) { _ = fsys.Setenv(\"K\", \"v\") }\n"},
		{"exec Command", "os/exec.Command", untaggedFile(`func TestA(t *testing.T) { _ = exec.Command("true").Run() }`)},
		{"filepath Walk", "path/filepath.Walk", untaggedFile(`func TestA(t *testing.T) { _ = filepath.Walk("d", nil) }`)},
		{"filepath WalkDir as a value", "path/filepath.WalkDir", untaggedFile(`var walk = filepath.WalkDir`)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings, err := analyzeDir(map[string][]byte{"case_test.go": []byte(c.src)})
			if err != nil {
				t.Fatalf("analyze: %v", err)
			}
			if len(findings) != 1 || findings[0].Use != c.use {
				t.Fatalf("findings = %+v; want one finding of %q", findings, c.use)
			}
		})
	}
}

// TestGuardFlagsAUseReachedThroughAHelper proves that a forbidden use in a helper is flagged where the helper lives,
// so a test that calls it through a method looks clean only in its own body. The helper and the test are in different
// files.
func TestGuardFlagsAUseReachedThroughAHelper(t *testing.T) {
	sources := map[string][]byte{
		"world_test.go": []byte(untaggedFile(`func (w *world) adoptProcessEnv() { os.Clearenv() }`)),
		"wiring_test.go": []byte(untaggedFile(`func TestUnit(t *testing.T) {
	w := &world{}
	w.adoptProcessEnv()
}`)),
	}
	findings, err := analyzeDir(sources)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(findings) != 1 || findings[0].File != "world_test.go" || findings[0].Use != "os.Clearenv" {
		t.Fatalf("findings = %+v; want the Clearenv in the helper's file only", findings)
	}
}

// TestGuardAllowsPureAndEmbeddedUses proves that the rule leaves alone the uses that touch no disk, environment, or
// process.
func TestGuardAllowsPureAndEmbeddedUses(t *testing.T) {
	src := untaggedFile(`func TestA(t *testing.T) {
	_ = filepath.Join("a", "b")
	_ = filepath.Rel("a", "a/b")
	_ = filepath.Base("a/b")
	_ = os.SameFile(nil, nil)
	_ = os.ErrNotExist
	var _ os.FileMode
	_ = fs.ErrNotExist
	_ = syscall.EPIPE
}`)
	findings, err := analyzeDir(map[string][]byte{"allowed_test.go": []byte(src)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %+v; want none for the pure uses", findings)
	}
}

// TestGuardSkipsFilesThatCompileOnlyAtATaggedLevel proves that a file with a level tag is not checked in the untagged
// build, and that a constraint that holds without any tag is checked.
func TestGuardSkipsFilesThatCompileOnlyAtATaggedLevel(t *testing.T) {
	body := `func TestA(t *testing.T) { _ = os.Setenv("K", "v") }`
	cases := []struct {
		name    string
		header  string
		checked bool
	}{
		{"no constraint", "", true},
		{"integration", "//go:build integration\n\n", false},
		{"e2e", "//go:build e2e\n\n", false},
		{"integration or e2e", "//go:build integration || e2e\n\n", false},
		{"negated integration", "//go:build !integration\n\n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings, err := analyzeDir(map[string][]byte{"tagged_test.go": []byte(c.header + untaggedFile(body))})
			if err != nil {
				t.Fatalf("analyze: %v", err)
			}
			if got := len(findings) == 1; got != c.checked {
				t.Fatalf("checked = %v, want %v (findings %+v)", got, c.checked, findings)
			}
		})
	}
}

// TestGuardReadsOnlyTestFiles proves that a production file in the same folder is outside the rule, so the rule never
// reports a line of production code.
func TestGuardReadsOnlyTestFiles(t *testing.T) {
	src := untaggedFile(`func Load() { _ = os.WriteFile("f", nil, 0o600) }`)
	findings, err := analyzeDir(map[string][]byte{"catalog.go": []byte(src)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %+v; want none, the file is not a test", findings)
	}
}

// TestGuardReportsFaultsInsteadOfPassing proves that a file the parser cannot read, or a build constraint it cannot
// evaluate, is an error and never a silent pass.
func TestGuardReportsFaultsInsteadOfPassing(t *testing.T) {
	if _, err := analyzeDir(map[string][]byte{"broken_test.go": []byte("package behaviour\nfunc (")}); err == nil {
		t.Fatal("a file that does not parse was accepted")
	}
	bad := "//go:build (\n\npackage behaviour\n"
	if _, err := analyzeDir(map[string][]byte{"bad_test.go": []byte(bad)}); err == nil {
		t.Fatal("a file with a malformed build constraint was accepted")
	}
}
