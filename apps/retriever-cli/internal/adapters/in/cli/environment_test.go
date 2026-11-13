//go:build integration

package cli

import "testing"

// TestSecretEnvReadsProcessEnvironment proves the lookup returns the process value of each variable retriever reads, and
// reads no other key.
func TestSecretEnvReadsProcessEnvironment(t *testing.T) {
	t.Setenv("EXAMPLE_OUTLINE_API_KEY", "outline-key")
	t.Setenv("EXAMPLE_KIWI_API_KEY", "kiwi-key")
	t.Setenv("EXAMPLE_JIRA_API_TOKEN", "jira-key")
	t.Setenv("EXAMPLE_JIRA_EMAIL", "jira-identity@example.test")
	t.Setenv("RETRIEVER_TEST_OTHER_KEY", "other")

	cases := map[string]string{
		"EXAMPLE_OUTLINE_API_KEY":  "outline-key",
		"EXAMPLE_KIWI_API_KEY":     "kiwi-key",
		"EXAMPLE_JIRA_API_TOKEN":   "jira-key",
		"EXAMPLE_JIRA_EMAIL":       "jira-identity@example.test",
		"RETRIEVER_TEST_OTHER_KEY": "",
	}
	for key, want := range cases {
		if got := SecretEnv(key); got != want {
			t.Errorf("SecretEnv(%q) = %q, want %q", key, got, want)
		}
	}
}

// TestSecretEnvEmptyWhenUnset proves an unset variable reads as the empty string.
func TestSecretEnvEmptyWhenUnset(t *testing.T) {
	keys := []string{"EXAMPLE_OUTLINE_API_KEY", "EXAMPLE_KIWI_API_KEY", "EXAMPLE_JIRA_API_TOKEN", "EXAMPLE_JIRA_EMAIL"}
	for _, key := range keys {
		t.Setenv(key, "")
	}
	for _, key := range keys {
		if got := SecretEnv(key); got != "" {
			t.Errorf("SecretEnv(%q) = %q, want empty", key, got)
		}
	}
}
