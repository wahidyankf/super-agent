package cli

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// TestPluginFailuresKeepTheirTextAndCode checks that each plugin failure a command can return prints its fixed text
// and keeps its stable code in every command that reports it: the text names the rule and never a value, because a
// frame may echo a secret.
func TestPluginFailuresKeepTheirTextAndCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		text string
		code string
	}{
		{
			name: "pin mismatch",
			err:  plugin.PinMismatchError{},
			text: "the plugin bytes differ from the approved pin",
			code: "retriever.connector.plugin_pin_mismatch",
		},
		{
			name: "protocol",
			err:  plugin.ProtocolError{Rule: "the plugin sent a notification"},
			text: "the plugin sent a notification",
			code: "retriever.connector.plugin_protocol",
		},
		{
			name: "start",
			err:  plugin.StartError{Rule: "could not start the plugin process"},
			text: "could not start the plugin process",
			code: "retriever.connector.plugin_start_failed",
		},
		{
			name: "timeout",
			err:  plugin.TimeoutError{Limit: 1000000000},
			text: "request to Outline timed out",
			code: "retriever.outline.timed_out",
		},
	}
	for _, c := range cases {
		if got := publicMessage(c.err); got != c.text {
			t.Errorf("%s: publicMessage = %q, want %q", c.name, got, c.text)
		}
		if got := errorCode(c.err, "outline"); got != c.code {
			t.Errorf("%s: errorCode = %q, want %q", c.name, got, c.code)
		}
	}
}

// TestPluginTimeoutExits124 checks that a silent plugin exits 124, like any other bounded wait, while every other
// plugin failure exits 2: a failed run is not a result, and a failed start is never 126 or 127.
func TestPluginTimeoutExits124(t *testing.T) {
	if got := exitStatusOf(plugin.TimeoutError{}); got != exitTimedOut {
		t.Fatalf("exitStatusOf(timeout) = %d, want %d", got, exitTimedOut)
	}
	for _, err := range []error{plugin.PinMismatchError{}, plugin.ProtocolError{}, plugin.StartError{}} {
		if got := exitStatusOf(err); got != exitFailed {
			t.Fatalf("exitStatusOf(%T) = %d, want %d", err, got, exitFailed)
		}
	}
}

// TestPluginTimeoutIsRetryable checks that only the timeout of the plugin failures may succeed when tried again
// later: a pin mismatch, a refused reply, and a failed start will fail the same way again.
func TestPluginTimeoutIsRetryable(t *testing.T) {
	if !retryable(plugin.TimeoutError{}) {
		t.Fatal("a plugin timeout must be retryable")
	}
	for _, err := range []error{plugin.PinMismatchError{}, plugin.ProtocolError{}, plugin.StartError{}} {
		if retryable(err) {
			t.Fatalf("%T must not be retryable", err)
		}
	}
}

// TestPluginFailuresThroughAKindKeepTheirCode checks that a plugin failure wrapped with a source kind keeps its
// stable connector code instead of taking the kind's area.
func TestPluginFailuresThroughAKindKeepTheirCode(t *testing.T) {
	err := withKind(domain.KindKiwi, plugin.PinMismatchError{})
	if got := errorCode(err, "kiwi"); got != "retriever.connector.plugin_pin_mismatch" {
		t.Fatalf("errorCode = %q, want the connector code", got)
	}
	if !errors.Is(err, plugin.PinMismatchError{}) {
		t.Fatalf("a kind-wrapped pin failure must still match: %v", err)
	}
}
