package unit_test

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// @scenario specs/libs/retriever-contract/behaviours/plugin-pins.feature: An absolute path with a lower-case hex pin of 64 characters is accepted
func TestPluginPinsAcceptAbsolutePathWithLowerCaseHexPin(t *testing.T) {
	ref, err := domain.NewPluginRef("/opt/retriever/plugins/example", validPluginPin)
	noError(t, err)
	if got := ref.Path(); got != "/opt/retriever/plugins/example" {
		t.Errorf("path = %q, want %q", got, "/opt/retriever/plugins/example")
	}
	if got := ref.SHA256(); got != validPluginPin {
		t.Errorf("pin = %q, want the same pin", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/plugin-pins.feature: A plugin path that is not absolute is refused
func TestPluginPinsRefusePathThatIsNotAbsolute(t *testing.T) {
	for _, path := range []string{"plugins/example", "./plugins/example", ""} {
		t.Run(path, func(t *testing.T) {
			ref, err := domain.NewPluginRef(path, validPluginPin)
			wantText(t, err, "invalid plugin path: use an absolute path")
			if ref != (domain.PluginRef{}) {
				t.Errorf("returned a plugin reference, want none")
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/plugin-pins.feature: A plugin pin that is not 64 lower-case hex characters is refused
func TestPluginPinsRefusePinThatIsNotSixtyFourLowerCaseHexCharacters(t *testing.T) {
	for _, pin := range []string{
		"0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdefg",
		"",
	} {
		t.Run(pin, func(t *testing.T) {
			ref, err := domain.NewPluginRef("/opt/retriever/plugins/example", pin)
			wantText(t, err, "invalid plugin pin: use 64 lower-case hex characters")
			if ref != (domain.PluginRef{}) {
				t.Errorf("returned a plugin reference, want none")
			}
		})
	}
}
