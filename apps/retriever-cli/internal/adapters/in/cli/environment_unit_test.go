package cli

import "testing"

// envOf returns a lookup over the given values, so a test sets variables without changing the process environment.
func envOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// TestSecretLookupReadsOutlineKey proves the lookup returns the value of the Outline API key, one of the variables a
// source names in its catalog entry.
func TestSecretLookupReadsOutlineKey(t *testing.T) {
	lookup := SecretLookup(envOf(map[string]string{"EXAMPLE_OUTLINE_API_KEY": "outline-key"}))

	if got := lookup("EXAMPLE_OUTLINE_API_KEY"); got != "outline-key" {
		t.Errorf("SecretLookup(EXAMPLE_OUTLINE_API_KEY) = %q, want %q", got, "outline-key")
	}
}

// TestSecretLookupOutlineKeyEmptyWhenUnset proves an unset Outline API key reads as the empty string.
func TestSecretLookupOutlineKeyEmptyWhenUnset(t *testing.T) {
	lookup := SecretLookup(envOf(nil))

	if got := lookup("EXAMPLE_OUTLINE_API_KEY"); got != "" {
		t.Errorf("SecretLookup(EXAMPLE_OUTLINE_API_KEY) = %q, want empty", got)
	}
}

// TestSecretLookupAsksOnlyForAcceptedKeys proves the lookup never asks the environment for a key the tool does not accept,
// so a variable outside the list stays unread even when it is set.
func TestSecretLookupAsksOnlyForAcceptedKeys(t *testing.T) {
	var asked []string
	lookup := SecretLookup(func(key string) string {
		asked = append(asked, key)
		return "set"
	})

	if got := lookup("RETRIEVER_TEST_OTHER_KEY"); got != "" {
		t.Errorf("an unaccepted key must read as empty, got %q", got)
	}
	if len(asked) != 0 {
		t.Errorf("the lookup asked the environment for %q, want no read", asked)
	}
}
