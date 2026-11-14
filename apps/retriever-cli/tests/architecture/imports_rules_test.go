package architecture

import "testing"

// TestImportRulesByKind checks each edge kind of the layering table: which module may import which, in production code
// and in tests. A violation returns a reason, and an allowed edge returns "".
func TestImportRulesByKind(t *testing.T) {
	const (
		contract  = "libs/retriever-contract"
		runtime   = "libs/retriever-runtime"
		testkit   = "libs/retriever-testkit"
		outline   = "libs/retriever-connector-outline"
		app       = "apps/retriever-cli"
		domainPkg = workspacePrefix + "libs/retriever-contract/domain"
		outlinePk = workspacePrefix + "libs/retriever-connector-outline/outline"
		runtimePk = workspacePrefix + "libs/retriever-runtime/sources"
		kitPkg    = workspacePrefix + "libs/retriever-testkit/outlinestub"
		appPkg    = workspacePrefix + "apps/retriever-cli/internal/adapters/in/cli"
	)
	cases := []struct {
		name   string
		self   string
		test   bool
		imp    string
		reason bool
	}{
		{"a contract imports nothing of the workspace", contract, false, domainPkg, false},
		{"a contract imports a runtime", contract, false, runtimePk, true},
		{"a runtime imports the contract", runtime, false, domainPkg, false},
		{"a runtime imports a connector", runtime, false, outlinePk, true},
		{"a runtime test imports the test kit", runtime, true, kitPkg, false},
		{"a runtime imports the test kit", runtime, false, kitPkg, true},
		{"a connector imports the contract", outline, false, domainPkg, false},
		{"a connector imports a runtime", outline, false, runtimePk, true},
		{"a connector imports the test kit in production code", outline, false, kitPkg, true},
		{"a connector test imports the test kit", outline, true, kitPkg, false},
		{"a connector test imports a runtime", outline, true, runtimePk, true},
		{"the test kit imports the contract", testkit, false, domainPkg, false},
		{"the test kit imports a runtime", testkit, false, runtimePk, true},
		{"a contract test imports the test kit", contract, true, kitPkg, true},
		{"the app imports the contract", app, false, domainPkg, false},
		{"the app imports a connector", app, false, outlinePk, false},
		{"the app imports the test kit in production code", app, false, kitPkg, true},
		{"the app test imports the test kit", app, true, kitPkg, false},
		{"a library imports the app", outline, false, appPkg, true},
		{"a module imports itself", outline, false, outlinePk, false},
		{"an external module is never a workspace edge", contract, false, "example.test/other/pkg", false},
		{"a standard package is never a workspace edge", contract, false, "fmt", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := importViolation(c.self, c.test, c.imp)
			if (got != "") != c.reason {
				t.Fatalf("importViolation(%q, test=%v, %q) = %q; want a violation: %v", c.self, c.test, c.imp, got, c.reason)
			}
		})
	}
}

// TestModuleOfPicksTheLongestModule checks that a workspace import path maps to its module, and that an import of no
// module is reported as one.
func TestModuleOfPicksTheLongestModule(t *testing.T) {
	if got := moduleOf(workspacePrefix + "libs/retriever-contract/domain"); got != "libs/retriever-contract" {
		t.Fatalf("moduleOf(domain) = %q", got)
	}
	if got := moduleOf(workspacePrefix + "libs/retriever-connector-kiwi/kiwi"); got != "libs/retriever-connector-kiwi" {
		t.Fatalf("moduleOf(kiwi) = %q", got)
	}
	if got := moduleOf(workspacePrefix + "libs/retriever-missing/pkg"); got != "" {
		t.Fatalf("moduleOf(missing) = %q, want none", got)
	}
	if got := moduleOf("net/http"); got != "" {
		t.Fatalf("moduleOf(net/http) = %q, want none", got)
	}
	if got := importViolation("libs/retriever-contract", false, workspacePrefix+"libs/retriever-missing/pkg"); got == "" {
		t.Fatal("an import of no module must be a violation")
	}
}

// TestEffectfulStandardPackagesAreNamed checks the pure rule: the network, the disk, and processes are effects, and the
// logging package is not, because it only formats a redacted value.
func TestEffectfulStandardPackagesAreNamed(t *testing.T) {
	for _, p := range []string{"net/http", "os", "os/exec", "io/fs", "path/filepath", "syscall", "log"} {
		if !effectViolation(p) {
			t.Fatalf("%s must be an effect", p)
		}
	}
	for _, p := range []string{"fmt", "log/slog", "strings", "regexp"} {
		if effectViolation(p) {
			t.Fatalf("%s must not be an effect", p)
		}
	}
}
