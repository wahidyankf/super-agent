package domain

import (
	"errors"
	"strings"
	"testing"
)

// pluginPinFixture is a well-formed pin: 64 lower-case hex characters.
const pluginPinFixture = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// mustPluginRef returns the pinned plugin of the fixture path, failing the test on any error.
func mustPluginRef(t *testing.T, path, pin string) PluginRef {
	t.Helper()
	ref, err := NewPluginRef(path, pin)
	if err != nil {
		t.Fatalf("NewPluginRef(%q): %v", path, err)
	}
	return ref
}

// TestNewPluginRefAcceptsAnAbsolutePathAndAPin checks the valid shape: an absolute path and a 64-character
// lower-case hex pin.
func TestNewPluginRefAcceptsAnAbsolutePathAndAPin(t *testing.T) {
	ref := mustPluginRef(t, "/opt/plugins/wiki", pluginPinFixture)
	if ref.Path() != "/opt/plugins/wiki" {
		t.Fatalf("Path() = %q", ref.Path())
	}
	if ref.SHA256() != pluginPinFixture {
		t.Fatalf("SHA256() = %q", ref.SHA256())
	}
}

// TestNewPluginRefRefusesARelativePath checks that a relative plugin path is refused: the host must never resolve a
// plugin against the working directory.
func TestNewPluginRefRefusesARelativePath(t *testing.T) {
	for _, path := range []string{"", "plugins/wiki", "./wiki", "../wiki", "wiki"} {
		if _, err := NewPluginRef(path, pluginPinFixture); !errors.Is(err, ErrInvalidPluginPath) {
			t.Fatalf("NewPluginRef(%q) error = %v, want ErrInvalidPluginPath", path, err)
		}
	}
}

// TestNewPluginRefRefusesAMisshapedPin checks that a pin outside 64 lower-case hex characters is refused.
func TestNewPluginRefRefusesAMisshapedPin(t *testing.T) {
	bad := []string{
		"",
		"0123456789abcdef",
		strings.Repeat("0", 63),
		strings.Repeat("0", 65),
		strings.ToUpper(pluginPinFixture),
		strings.Repeat("g", 64),
		" " + pluginPinFixture,
		pluginPinFixture + "\n",
	}
	for _, pin := range bad {
		if _, err := NewPluginRef("/opt/plugins/wiki", pin); !errors.Is(err, ErrInvalidPluginPin) {
			t.Fatalf("NewPluginRef(pin %q) error = %v, want ErrInvalidPluginPin", pin, err)
		}
	}
}

// TestPluginPinErrorTextsNameNoValue checks that the pin errors state the rule in fixed words: no path or pin that a
// test passes reaches the text.
func TestPluginPinErrorTextsNameNoValue(t *testing.T) {
	if got := ErrInvalidPluginPath.Error(); strings.Contains(got, "/") || got == "" {
		t.Fatalf("ErrInvalidPluginPath = %q, want fixed words with no path", got)
	}
	if got := ErrInvalidPluginPin.Error(); strings.Contains(got, pluginPinFixture) || got == "" {
		t.Fatalf("ErrInvalidPluginPin = %q, want fixed words with no pin", got)
	}
}

// mustPluginSource returns the Jira source of the plugin fixtures, failing the test on any error.
func mustPluginSource(t *testing.T) Source {
	t.Helper()
	id, err := NewSourceID("tracker")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := NewEndpoint("https://tracker.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	credential, err := NewCredentialRef("EXAMPLE_JIRA_API_TOKEN")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	identity, err := NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	return NewSource(id, KindJira, mustTagSet(t), endpoint, credential).WithIdentity(identity)
}

// TestSourceHasNoPluginByDefault checks P2 at the value level: a source built without WithPlugin enables no plugin,
// so the host starts no process for it.
func TestSourceHasNoPluginByDefault(t *testing.T) {
	if src := mustPluginSource(t); src.HasPlugin() {
		t.Fatal("a source built without WithPlugin must have no plugin")
	}
}

// TestSourceWithPluginKeepsEveryOtherField checks that WithPlugin sets only the plugin: the receiver does not change,
// and the copy keeps the identity.
func TestSourceWithPluginKeepsEveryOtherField(t *testing.T) {
	base := mustPluginSource(t)
	ref := mustPluginRef(t, "/opt/plugins/tracker", pluginPinFixture)
	if base.HasPlugin() {
		t.Fatal("the receiver must not gain a plugin")
	}
	with := base.WithPlugin(ref)
	if !with.HasPlugin() {
		t.Fatal("the copy must have a plugin")
	}
	if with.Plugin() != ref {
		t.Fatalf("Plugin() = %+v, want %+v", with.Plugin(), ref)
	}
	if with.Identity() != base.Identity() || with.ID() != base.ID() || with.Kind() != base.Kind() {
		t.Fatal("WithPlugin changed a field other than the plugin")
	}
}
