package catalog

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// pluginPin is a well-formed pin: 64 lower-case hex characters.
const pluginPin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// withPluginBlock returns oneSource with a plugin block appended to the source: the path and pin the test passes.
func withPluginBlock(path, pin string) string {
	return oneSource + "    plugin:\n      path: " + path + "\n      sha256: " + pin + "\n"
}

// TestPluginBlockEnablesAPluginForTheSource checks the opt-in: a source with a plugin block loads with the pinned
// path and pin, and a source without one loads with no plugin (P2).
func TestPluginBlockEnablesAPluginForTheSource(t *testing.T) {
	cat := mustLoad(t, withPluginBlock("/opt/plugins/wiki", pluginPin))
	all := cat.All()
	if len(all) != 1 {
		t.Fatalf("catalog holds %d sources, want 1", len(all))
	}
	src := all[0]
	if !src.HasPlugin() {
		t.Fatal("a source with a plugin block must have a plugin")
	}
	if got := src.Plugin().Path(); got != "/opt/plugins/wiki" {
		t.Fatalf("Plugin().Path() = %q", got)
	}
	if got := src.Plugin().SHA256(); got != pluginPin {
		t.Fatalf("Plugin().SHA256() = %q", got)
	}
}

// TestSourceWithoutAPluginBlockStartsNoPlugin checks the default: a source with no plugin block loads with no plugin,
// so the host serves it from its kind's gateway and starts no process (P2).
func TestSourceWithoutAPluginBlockStartsNoPlugin(t *testing.T) {
	cat := mustLoad(t, oneSource)
	for _, src := range cat.All() {
		if src.HasPlugin() {
			t.Fatalf("source %q without a plugin block must have no plugin", src.ID())
		}
	}
}

// TestPluginBlockRefusesARelativePath checks that a relative plugin path fails the load with the fixed rule, naming
// the source and the field and never the path.
func TestPluginBlockRefusesARelativePath(t *testing.T) {
	mustFailWith(t, withPluginBlock("plugins/wiki", pluginPin), "source 1 plugin: invalid plugin path: use an absolute path")
}

// TestPluginBlockRefusesAMisshapedPin checks that a pin outside 64 lower-case hex characters fails the load.
func TestPluginBlockRefusesAMisshapedPin(t *testing.T) {
	mustFailWith(t, withPluginBlock("/opt/plugins/wiki", "0123456789abcdef"),
		"source 1 plugin: invalid plugin pin: use 64 lower-case hex characters")
	mustFailWith(t, withPluginBlock("/opt/plugins/wiki", ""), "source 1 plugin: invalid plugin pin: use 64 lower-case hex characters")
}

// TestPluginBlockErrorsAreCatalogErrors checks that a plugin failure is a domain.CatalogError carrying the typed rule,
// so the CLI reports it with the catalog code.
func TestPluginBlockErrorsAreCatalogErrors(t *testing.T) {
	_, err := loadFrom(t, withPluginBlock("plugins/wiki", pluginPin))
	var ce domain.CatalogError
	if !errors.As(err, &ce) {
		t.Fatalf("Load() error = %T, want domain.CatalogError", err)
	}
	if !errors.Is(err, domain.ErrInvalidPluginPath) {
		t.Fatalf("Load() error = %v, want ErrInvalidPluginPath", err)
	}
}
