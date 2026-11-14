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
		spec      = "libs/retriever-spec"
		app       = "apps/retriever-cli"
		domainPkg = workspacePrefix + "libs/retriever-contract/domain"
		outlinePk = workspacePrefix + "libs/retriever-connector-outline/outline"
		runtimePk = workspacePrefix + "libs/retriever-runtime/sources"
		kitPkg    = workspacePrefix + "libs/retriever-testkit/outlinestub"
		appPkg    = workspacePrefix + "apps/retriever-cli/internal/adapters/in/cli"
		specPk    = workspacePrefix + "libs/retriever-spec"
		engine    = "libs/retriever-engine"
		enginePk  = workspacePrefix + "libs/retriever-engine/engine"
		replayPk  = workspacePrefix + "libs/retriever-engine/engine/replay"
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
		{"a spec imports the contract", spec, false, domainPkg, false},
		{"a spec imports a runtime", spec, false, runtimePk, true},
		{"a spec imports a connector", spec, false, outlinePk, true},
		{"a spec imports the test kit in production code", spec, false, kitPkg, true},
		{"a spec test imports the test kit", spec, true, kitPkg, false},
		{"a spec test imports a connector", spec, true, outlinePk, true},
		{"a spec imports the app", spec, false, appPkg, true},
		{"a connector imports the spec", outline, false, specPk, true},
		{"a runtime imports the spec", runtime, false, specPk, true},
		{"the contract imports the spec", contract, false, specPk, true},
		{"the app imports the spec", app, false, specPk, false},
		{"an engine imports the contract", engine, false, domainPkg, false},
		{"an engine imports the spec module", engine, false, specPk, false},
		{"an engine imports its replay stub", engine, false, replayPk, false},
		{"an engine imports a connector", engine, false, outlinePk, true},
		{"an engine imports a runtime", engine, false, runtimePk, true},
		{"an engine imports the test kit in production code", engine, false, kitPkg, true},
		{"an engine test imports the test kit", engine, true, kitPkg, false},
		{"an engine test imports the spec module", engine, true, specPk, false},
		{"an engine imports the app", engine, false, appPkg, true},
		{"a spec imports the engine", spec, false, enginePk, true},
		{"the contract imports the engine", contract, false, enginePk, true},
		{"a connector imports the engine", outline, false, enginePk, true},
		{"a runtime imports the engine", runtime, false, enginePk, true},
		{"the app imports the engine", app, false, enginePk, false},
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

// TestSpecModuleIsPure checks that the spec module is one of the pure directories, so that its production code is held to
// the purity rule of the module layout: it reaches the outside world through no standard package in the effect list.
func TestSpecModuleIsPure(t *testing.T) {
	for _, dir := range pureDirs {
		if dir == "libs/retriever-spec" {
			return
		}
	}
	t.Fatal("libs/retriever-spec must be listed in pureDirs: the spec module is pure")
}

// TestEngineForbidsLocalEffects checks the engine rule of the module layout: the engine imports net/http and the TLS
// packages, but no package that reads or writes a local file, a folder, the working directory, or a process.
func TestEngineForbidsLocalEffects(t *testing.T) {
	for _, p := range []string{"os", "os/exec", "path/filepath", "io/fs", "syscall"} {
		if !engineEffectViolation(p) {
			t.Fatalf("%s must be an engine effect violation", p)
		}
	}
	for _, p := range []string{"net", "net/http", "crypto/tls", "crypto/x509", "io", "time", "encoding/json", "net/netip"} {
		if engineEffectViolation(p) {
			t.Fatalf("%s must be allowed in the engine", p)
		}
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
